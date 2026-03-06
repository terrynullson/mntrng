package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/terrynullson/mntrng/internal/domain"
)

type APIRegistrationRepo struct {
	db *sql.DB
}

func NewAPIRegistrationRepo(db *sql.DB) *APIRegistrationRepo {
	return &APIRegistrationRepo{db: db}
}

func (r *APIRegistrationRepo) CreateRegistrationRequest(
	ctx context.Context,
	companyID int64,
	email string,
	login string,
	passwordHash string,
	requestedRole string,
) (domain.RegistrationRequest, error) {
	var item domain.RegistrationRequest
	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO registration_requests (company_id, email, login, password_hash, requested_role)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, company_id, email, login, requested_role, status, created_at, updated_at, processed_at, processed_by_user_id, decision_reason`,
		companyID,
		email,
		login,
		passwordHash,
		requestedRole,
	).Scan(
		&item.ID,
		&item.CompanyID,
		&item.Email,
		&item.Login,
		&item.RequestedRole,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.ProcessedAt,
		&item.ProcessedByUserID,
		&item.DecisionReason,
	)
	if err != nil {
		if isForeignKeyViolation(err) {
			return domain.RegistrationRequest{}, domain.ErrCompanyNotFound
		}
		if isUniqueViolation(err) {
			return domain.RegistrationRequest{}, domain.ErrRegistrationConflict
		}
		return domain.RegistrationRequest{}, err
	}
	return item, nil
}

func (r *APIRegistrationRepo) ListPendingRegistrationRequests(ctx context.Context) ([]domain.RegistrationRequest, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, company_id, email, login, requested_role, status, created_at, updated_at, processed_at, processed_by_user_id, decision_reason
         FROM registration_requests
         WHERE status = 'pending'
         ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.RegistrationRequest, 0)
	for rows.Next() {
		item, err := scanRegistrationRequest(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *APIRegistrationRepo) ApproveRegistrationRequest(
	ctx context.Context,
	requestID int64,
	companyID int64,
	role string,
	actorUserID int64,
) (domain.AuthUser, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.AuthUser{}, err
	}
	defer tx.Rollback()

	requestItem, err := loadRegistrationRequestForUpdate(ctx, tx, requestID)
	if err != nil {
		return domain.AuthUser{}, err
	}
	if requestItem.Record.Status != domain.RegistrationStatusPending {
		return domain.AuthUser{}, domain.ErrRegistrationNotPending
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE registration_requests
         SET
            status = 'approved',
            company_id = $1,
            requested_role = $2,
            processed_at = NOW(),
            processed_by_user_id = $3,
            updated_at = NOW(),
            decision_reason = NULL
         WHERE id = $4`,
		companyID,
		role,
		actorUserID,
		requestID,
	); err != nil {
		if isForeignKeyViolation(err) {
			return domain.AuthUser{}, domain.ErrCompanyNotFound
		}
		return domain.AuthUser{}, err
	}

	var user domain.AuthUser
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO users (company_id, email, login, password_hash, role, status)
         VALUES ($1, $2, $3, $4, $5, 'active')
         RETURNING id, company_id, email, login, role, status, created_at, updated_at`,
		companyID,
		requestItem.Record.Email,
		requestItem.Record.Login,
		requestItemPasswordHash(requestItem),
		role,
	).Scan(
		&user.ID,
		&user.CompanyID,
		&user.Email,
		&user.Login,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.AuthUser{}, domain.ErrUserAlreadyExists
		}
		if isForeignKeyViolation(err) {
			return domain.AuthUser{}, domain.ErrCompanyNotFound
		}
		return domain.AuthUser{}, err
	}

	if err := InsertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		strconv.FormatInt(actorUserID, 10),
		"registration_request",
		requestID,
		"approve",
		map[string]interface{}{
			"user_id":           user.ID,
			"company_id":        companyID,
			"requested_role":    requestItem.Record.RequestedRole,
			"approved_role":     role,
			"source_company_id": requestItem.Record.CompanyID,
		},
	); err != nil {
		return domain.AuthUser{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.AuthUser{}, err
	}
	return user, nil
}

func (r *APIRegistrationRepo) RejectRegistrationRequest(ctx context.Context, requestID int64, actorUserID int64, reason *string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	requestItem, err := loadRegistrationRequestForUpdate(ctx, tx, requestID)
	if err != nil {
		return err
	}
	if requestItem.Record.Status != domain.RegistrationStatusPending {
		return domain.ErrRegistrationNotPending
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE registration_requests
         SET
            status = 'rejected',
            processed_at = NOW(),
            processed_by_user_id = $1,
            decision_reason = $2,
            updated_at = NOW()
         WHERE id = $3`,
		actorUserID,
		reason,
		requestID,
	)
	if err != nil {
		return err
	}

	auditPayload := map[string]interface{}{}
	if reason != nil {
		auditPayload["reason"] = *reason
	}
	if err := InsertAuditLogTx(
		ctx,
		tx,
		requestItem.Record.CompanyID,
		domain.AuditActorTypeAPI,
		strconv.FormatInt(actorUserID, 10),
		"registration_request",
		requestID,
		"reject",
		auditPayload,
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *APIRegistrationRepo) ChangeUserRole(
	ctx context.Context,
	userID int64,
	role string,
	companyID *int64,
	actorUserID int64,
) (domain.AuthUser, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.AuthUser{}, err
	}
	defer tx.Rollback()

	currentUser, err := scanAuthUser(tx.QueryRowContext(
		ctx,
		`SELECT id, company_id, email, login, role, status, created_at, updated_at
         FROM users
         WHERE id = $1
         FOR UPDATE`,
		userID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AuthUser{}, domain.ErrUserNotFound
		}
		return domain.AuthUser{}, err
	}

	updatedUser, err := updateUserRoleTx(ctx, tx, userID, role, companyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AuthUser{}, domain.ErrUserNotFound
		}
		if isForeignKeyViolation(err) {
			return domain.AuthUser{}, domain.ErrCompanyNotFound
		}
		return domain.AuthUser{}, err
	}

	auditCompanyID := updatedUser.CompanyID
	if auditCompanyID == nil {
		auditCompanyID = currentUser.CompanyID
	}
	if auditCompanyID == nil {
		return domain.AuthUser{}, domain.ErrCompanyNotFound
	}

	payload := map[string]interface{}{
		"from_role": currentUser.Role,
		"to_role":   updatedUser.Role,
	}
	if currentUser.CompanyID != nil {
		payload["from_company_id"] = *currentUser.CompanyID
	}
	if updatedUser.CompanyID != nil {
		payload["to_company_id"] = *updatedUser.CompanyID
	}

	if err := InsertAuditLogTx(
		ctx,
		tx,
		*auditCompanyID,
		domain.AuditActorTypeAPI,
		strconv.FormatInt(actorUserID, 10),
		"user",
		userID,
		"role_change",
		payload,
	); err != nil {
		return domain.AuthUser{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.AuthUser{}, err
	}
	return updatedUser, nil
}

func scanRegistrationRequest(scanner interface {
	Scan(dest ...interface{}) error
}) (domain.RegistrationRequest, error) {
	var item domain.RegistrationRequest
	err := scanner.Scan(
		&item.ID,
		&item.CompanyID,
		&item.Email,
		&item.Login,
		&item.RequestedRole,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.ProcessedAt,
		&item.ProcessedByUserID,
		&item.DecisionReason,
	)
	if err != nil {
		return domain.RegistrationRequest{}, err
	}
	return item, nil
}

type registrationRequestForUpdate struct {
	Record       domain.RegistrationRequest
	PasswordHash string
}

func loadRegistrationRequestForUpdate(ctx context.Context, tx *sql.Tx, requestID int64) (registrationRequestForUpdate, error) {
	var item registrationRequestForUpdate
	err := tx.QueryRowContext(
		ctx,
		`SELECT id, company_id, email, login, password_hash, requested_role, status, created_at, updated_at, processed_at, processed_by_user_id, decision_reason
         FROM registration_requests
         WHERE id = $1
         FOR UPDATE`,
		requestID,
	).Scan(
		&item.Record.ID,
		&item.Record.CompanyID,
		&item.Record.Email,
		&item.Record.Login,
		&item.PasswordHash,
		&item.Record.RequestedRole,
		&item.Record.Status,
		&item.Record.CreatedAt,
		&item.Record.UpdatedAt,
		&item.Record.ProcessedAt,
		&item.Record.ProcessedByUserID,
		&item.Record.DecisionReason,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return registrationRequestForUpdate{}, domain.ErrRegistrationNotFound
		}
		return registrationRequestForUpdate{}, err
	}
	return item, nil
}

func requestItemPasswordHash(item registrationRequestForUpdate) string {
	return item.PasswordHash
}
