package postgres

import (
	"context"
	"database/sql"

	"github.com/terrynullson/mntrng/internal/domain"
)

func updateUserRoleTx(ctx context.Context, tx *sql.Tx, userID int64, role string, companyID *int64) (domain.AuthUser, error) {
	var item domain.AuthUser
	err := tx.QueryRowContext(
		ctx,
		`UPDATE users
         SET role = $1,
             company_id = $2,
             updated_at = NOW()
         WHERE id = $3
         RETURNING id, company_id, email, login, role, status, created_at, updated_at`,
		role,
		companyID,
		userID,
	).Scan(
		&item.ID,
		&item.CompanyID,
		&item.Email,
		&item.Login,
		&item.Role,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func scanAuthUser(scanner interface {
	Scan(dest ...interface{}) error
}) (domain.AuthUser, error) {
	var item domain.AuthUser
	err := scanner.Scan(
		&item.ID,
		&item.CompanyID,
		&item.Email,
		&item.Login,
		&item.Role,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return domain.AuthUser{}, err
	}
	return item, nil
}
