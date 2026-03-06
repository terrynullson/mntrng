package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

type APIAuthRepo struct {
	db *sql.DB
}

func NewAPIAuthRepo(db *sql.DB) *APIAuthRepo {
	return &APIAuthRepo{db: db}
}

func (r *APIAuthRepo) GetUserByLoginOrEmail(ctx context.Context, identity string) (domain.UserRecord, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, email, login, password_hash, role, status, created_at, updated_at
         FROM users
         WHERE LOWER(login) = LOWER($1) OR LOWER(email) = LOWER($1)
         LIMIT 1`,
		identity,
	)
	item, err := scanUserRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.UserRecord{}, domain.ErrUserNotFound
		}
		return domain.UserRecord{}, err
	}
	return item, nil
}

func (r *APIAuthRepo) GetUserByID(ctx context.Context, userID int64) (domain.UserRecord, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, email, login, password_hash, role, status, created_at, updated_at
         FROM users
         WHERE id = $1`,
		userID,
	)
	item, err := scanUserRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.UserRecord{}, domain.ErrUserNotFound
		}
		return domain.UserRecord{}, err
	}
	return item, nil
}

func (r *APIAuthRepo) CreateSession(
	ctx context.Context,
	user domain.AuthUser,
	accessTokenHash string,
	refreshTokenHash string,
	accessExpiresAt time.Time,
	refreshExpiresAt time.Time,
) (domain.AuthSession, error) {
	var item domain.AuthSession
	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO auth_sessions (
             user_id,
             company_id,
             access_token_hash,
             refresh_token_hash,
             access_expires_at,
             refresh_expires_at
         ) VALUES ($1, $2, $3, $4, $5, $6)
         RETURNING id, user_id, company_id, access_token_hash, refresh_token_hash, access_expires_at, refresh_expires_at, revoked_at, created_at, updated_at`,
		user.ID,
		user.CompanyID,
		accessTokenHash,
		refreshTokenHash,
		accessExpiresAt,
		refreshExpiresAt,
	).Scan(
		&item.ID,
		&item.UserID,
		&item.CompanyID,
		&item.AccessTokenHash,
		&item.RefreshTokenHash,
		&item.AccessExpiresAt,
		&item.RefreshExpiresAt,
		&item.RevokedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return domain.AuthSession{}, err
	}
	return item, nil
}

func (r *APIAuthRepo) GetSessionByAccessTokenHash(ctx context.Context, accessTokenHash string) (domain.AuthSessionUser, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT
            s.id,
            s.user_id,
            s.company_id,
            s.access_token_hash,
            s.refresh_token_hash,
            s.access_expires_at,
            s.refresh_expires_at,
            s.revoked_at,
            s.created_at,
            s.updated_at,
            u.id,
            u.company_id,
            u.email,
            u.login,
            u.role,
            u.status,
            u.created_at,
            u.updated_at
         FROM auth_sessions s
         JOIN users u ON u.id = s.user_id
         WHERE s.access_token_hash = $1
         LIMIT 1`,
		accessTokenHash,
	)
	item, err := scanAuthSessionUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AuthSessionUser{}, domain.ErrSessionNotFound
		}
		return domain.AuthSessionUser{}, err
	}
	return item, nil
}

func (r *APIAuthRepo) GetSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (domain.AuthSessionUser, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT
            s.id,
            s.user_id,
            s.company_id,
            s.access_token_hash,
            s.refresh_token_hash,
            s.access_expires_at,
            s.refresh_expires_at,
            s.revoked_at,
            s.created_at,
            s.updated_at,
            u.id,
            u.company_id,
            u.email,
            u.login,
            u.role,
            u.status,
            u.created_at,
            u.updated_at
         FROM auth_sessions s
         JOIN users u ON u.id = s.user_id
         WHERE s.refresh_token_hash = $1
         LIMIT 1`,
		refreshTokenHash,
	)
	item, err := scanAuthSessionUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AuthSessionUser{}, domain.ErrSessionNotFound
		}
		return domain.AuthSessionUser{}, err
	}
	return item, nil
}

func (r *APIAuthRepo) RotateSessionTokens(
	ctx context.Context,
	sessionID int64,
	accessTokenHash string,
	refreshTokenHash string,
	accessExpiresAt time.Time,
	refreshExpiresAt time.Time,
) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE auth_sessions
         SET
            access_token_hash = $1,
            refresh_token_hash = $2,
            access_expires_at = $3,
            refresh_expires_at = $4,
            revoked_at = NULL,
            updated_at = NOW()
         WHERE id = $5 AND revoked_at IS NULL`,
		accessTokenHash,
		refreshTokenHash,
		accessExpiresAt,
		refreshExpiresAt,
		sessionID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

func (r *APIAuthRepo) RevokeSessionByID(ctx context.Context, sessionID int64) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE auth_sessions
         SET revoked_at = NOW(), updated_at = NOW()
         WHERE id = $1 AND revoked_at IS NULL`,
		sessionID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

func (r *APIAuthRepo) RevokeSessionByRefreshToken(ctx context.Context, userID int64, refreshTokenHash string) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE auth_sessions
         SET revoked_at = NOW(), updated_at = NOW()
         WHERE user_id = $1
           AND refresh_token_hash = $2
           AND revoked_at IS NULL`,
		userID,
		refreshTokenHash,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

func (r *APIAuthRepo) HasPendingRegistration(ctx context.Context, identity string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1
         FROM registration_requests
         WHERE status = 'pending'
           AND (LOWER(login) = LOWER($1) OR LOWER(email) = LOWER($1))
         LIMIT 1`,
		identity,
	).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func (r *APIAuthRepo) UpsertTelegramLink(ctx context.Context, user domain.AuthUser, telegramUserID int64, telegramUsername *string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO user_telegram_links (user_id, telegram_user_id, telegram_username)
         VALUES ($1, $2, $3)
         ON CONFLICT (user_id)
         DO UPDATE SET
            telegram_user_id = EXCLUDED.telegram_user_id,
            telegram_username = EXCLUDED.telegram_username,
            linked_at = NOW()`,
		user.ID,
		telegramUserID,
		telegramUsername,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrTelegramLinkConflict
		}
		return err
	}

	if user.CompanyID != nil {
		auditPayload := map[string]interface{}{
			"telegram_user_id": telegramUserID,
		}
		if telegramUsername != nil {
			auditPayload["telegram_username"] = *telegramUsername
		}
		if err := InsertAuditLogTx(
			ctx,
			tx,
			*user.CompanyID,
			domain.AuditActorTypeAPI,
			strconv.FormatInt(user.ID, 10),
			"user_telegram_link",
			user.ID,
			"link",
			auditPayload,
		); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (r *APIAuthRepo) GetUserByTelegramUserID(ctx context.Context, telegramUserID int64) (domain.UserRecord, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT u.id, u.company_id, u.email, u.login, u.password_hash, u.role, u.status, u.created_at, u.updated_at
         FROM user_telegram_links l
         JOIN users u ON u.id = l.user_id
         WHERE l.telegram_user_id = $1
         LIMIT 1`,
		telegramUserID,
	)
	item, err := scanUserRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.UserRecord{}, domain.ErrTelegramLinkNotFound
		}
		return domain.UserRecord{}, err
	}
	return item, nil
}
