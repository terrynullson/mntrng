package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

func (r *APIRegistrationRepo) ListUsers(ctx context.Context, filter domain.AdminUserListFilter) ([]domain.AuthUser, error) {
	args := make([]interface{}, 0, 4)
	// Explicit super_admin policy allows cross-company reads for this admin endpoint.
	query := `SELECT id, company_id, email, login, role, status, created_at, updated_at
              FROM users
              WHERE 1 = 1`

	if filter.CompanyID != nil {
		args = append(args, *filter.CompanyID)
		query += " AND company_id = $" + strconv.Itoa(len(args))
	}
	if filter.Role != nil {
		args = append(args, *filter.Role)
		query += " AND role = $" + strconv.Itoa(len(args))
	}
	if filter.Status != nil {
		args = append(args, *filter.Status)
		query += " AND status = $" + strconv.Itoa(len(args))
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	args = append(args, limit)
	query += " ORDER BY created_at DESC, id DESC LIMIT $" + strconv.Itoa(len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.AuthUser, 0)
	for rows.Next() {
		item, err := scanAuthUser(rows)
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

func (r *APIRegistrationRepo) ChangeUserStatus(
	ctx context.Context,
	userID int64,
	status string,
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
	if currentUser.CompanyID == nil {
		return domain.AuthUser{}, domain.ErrUserScopeNotSupported
	}

	updatedUser, err := scanAuthUser(tx.QueryRowContext(
		ctx,
		`UPDATE users
         SET status = $1,
             updated_at = NOW()
         WHERE id = $2
         RETURNING id, company_id, email, login, role, status, created_at, updated_at`,
		status,
		userID,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AuthUser{}, domain.ErrUserNotFound
		}
		return domain.AuthUser{}, err
	}

	payload := map[string]interface{}{
		"user_id":       userID,
		"old_status":    currentUser.Status,
		"new_status":    updatedUser.Status,
		"actor_user_id": actorUserID,
	}
	if err := InsertAuditLogTx(
		ctx,
		tx,
		*currentUser.CompanyID,
		domain.AuditActorTypeAPI,
		strconv.FormatInt(actorUserID, 10),
		"user",
		userID,
		"status_change",
		payload,
	); err != nil {
		return domain.AuthUser{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.AuthUser{}, err
	}

	return updatedUser, nil
}
