package api

import (
	"context"
	"database/sql"
	"errors"

	"github.com/example/hls-monitoring-platform/internal/domain"
	serviceapi "github.com/example/hls-monitoring-platform/internal/service/api"
)

type companyStore struct {
	db *sql.DB
}

func newCompanyStore(db *sql.DB) *companyStore {
	return &companyStore{db: db}
}

func (s *companyStore) CreateCompany(ctx context.Context, name string) (domain.Company, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Company{}, err
	}
	defer tx.Rollback()

	var item domain.Company
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO companies (name) VALUES ($1) RETURNING id, name, created_at`,
		name,
	).Scan(&item.ID, &item.Name, &item.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Company{}, serviceapi.ErrCompanyAlreadyExists
		}
		return domain.Company{}, err
	}

	if err := insertAuditLogTx(
		ctx,
		tx,
		item.ID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeCompany,
		item.ID,
		domain.AuditActionCompanyCreate,
		map[string]interface{}{"name": item.Name},
	); err != nil {
		return domain.Company{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Company{}, err
	}

	return item, nil
}

func (s *companyStore) ListCompanies(ctx context.Context) ([]domain.Company, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, created_at FROM companies ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Company, 0)
	for rows.Next() {
		var item domain.Company
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *companyStore) GetCompany(ctx context.Context, companyID int64) (domain.Company, error) {
	var item domain.Company
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, created_at FROM companies WHERE id = $1`,
		companyID,
	).Scan(&item.ID, &item.Name, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Company{}, serviceapi.ErrCompanyNotFound
		}
		return domain.Company{}, err
	}
	return item, nil
}

func (s *companyStore) UpdateCompany(ctx context.Context, companyID int64, name string) (domain.Company, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Company{}, err
	}
	defer tx.Rollback()

	var item domain.Company
	err = tx.QueryRowContext(
		ctx,
		`UPDATE companies SET name = $1 WHERE id = $2 RETURNING id, name, created_at`,
		name,
		companyID,
	).Scan(&item.ID, &item.Name, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Company{}, serviceapi.ErrCompanyNotFound
		}
		if isUniqueViolation(err) {
			return domain.Company{}, serviceapi.ErrCompanyAlreadyExists
		}
		return domain.Company{}, err
	}

	if err := insertAuditLogTx(
		ctx,
		tx,
		item.ID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeCompany,
		item.ID,
		domain.AuditActionCompanyUpdate,
		map[string]interface{}{"changes": map[string]interface{}{"name": name}},
	); err != nil {
		return domain.Company{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Company{}, err
	}
	return item, nil
}

func (s *companyStore) DeleteCompany(ctx context.Context, companyID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existing domain.Company
	err = tx.QueryRowContext(
		ctx,
		`SELECT id, name, created_at
         FROM companies
         WHERE id = $1
         FOR UPDATE`,
		companyID,
	).Scan(&existing.ID, &existing.Name, &existing.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return serviceapi.ErrCompanyNotFound
		}
		return err
	}

	if err := insertAuditLogTx(
		ctx,
		tx,
		existing.ID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeCompany,
		existing.ID,
		domain.AuditActionCompanyDelete,
		map[string]interface{}{"name": existing.Name},
	); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM companies WHERE id = $1`, companyID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return serviceapi.ErrCompanyNotFound
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
