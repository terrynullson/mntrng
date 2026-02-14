package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/example/hls-monitoring-platform/internal/domain"
	serviceapi "github.com/example/hls-monitoring-platform/internal/service/api"
)

type APIProjectRepo struct {
	db *sql.DB
}

func NewAPIProjectRepo(db *sql.DB) *APIProjectRepo {
	return &APIProjectRepo{db: db}
}

func (r *APIProjectRepo) CreateProject(ctx context.Context, companyID int64, name string) (domain.Project, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Project{}, err
	}
	defer tx.Rollback()

	var item domain.Project
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO projects (company_id, name) VALUES ($1, $2) RETURNING id, company_id, name, created_at, updated_at`,
		companyID,
		name,
	).Scan(&item.ID, &item.CompanyID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Project{}, serviceapi.ErrProjectAlreadyExists
		}
		if isForeignKeyViolation(err) {
			return domain.Project{}, serviceapi.ErrProjectCompanyMissing
		}
		return domain.Project{}, err
	}

	if err := insertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeProject,
		item.ID,
		domain.AuditActionProjectCreate,
		map[string]interface{}{"name": item.Name},
	); err != nil {
		return domain.Project{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Project{}, err
	}

	return item, nil
}

func (r *APIProjectRepo) ListProjects(ctx context.Context, companyID int64) ([]domain.Project, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, company_id, name, created_at, updated_at FROM projects WHERE company_id = $1 ORDER BY id ASC`,
		companyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Project, 0)
	for rows.Next() {
		var item domain.Project
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *APIProjectRepo) GetProject(ctx context.Context, companyID int64, projectID int64) (domain.Project, error) {
	var item domain.Project
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, name, created_at, updated_at FROM projects WHERE company_id = $1 AND id = $2`,
		companyID,
		projectID,
	).Scan(&item.ID, &item.CompanyID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Project{}, serviceapi.ErrProjectNotFound
		}
		return domain.Project{}, err
	}
	return item, nil
}

func (r *APIProjectRepo) UpdateProject(ctx context.Context, companyID int64, projectID int64, name string) (domain.Project, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Project{}, err
	}
	defer tx.Rollback()

	var item domain.Project
	err = tx.QueryRowContext(
		ctx,
		`UPDATE projects SET name = $1, updated_at = NOW() WHERE company_id = $2 AND id = $3 RETURNING id, company_id, name, created_at, updated_at`,
		name,
		companyID,
		projectID,
	).Scan(&item.ID, &item.CompanyID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Project{}, serviceapi.ErrProjectNotFound
		}
		if isUniqueViolation(err) {
			return domain.Project{}, serviceapi.ErrProjectAlreadyExists
		}
		return domain.Project{}, err
	}

	if err := insertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeProject,
		item.ID,
		domain.AuditActionProjectUpdate,
		map[string]interface{}{"changes": map[string]interface{}{"name": name}},
	); err != nil {
		return domain.Project{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Project{}, err
	}
	return item, nil
}

func (r *APIProjectRepo) DeleteProject(ctx context.Context, companyID int64, projectID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var deleted domain.Project
	err = tx.QueryRowContext(
		ctx,
		`DELETE FROM projects
         WHERE company_id = $1 AND id = $2
         RETURNING id, company_id, name, created_at, updated_at`,
		companyID,
		projectID,
	).Scan(
		&deleted.ID,
		&deleted.CompanyID,
		&deleted.Name,
		&deleted.CreatedAt,
		&deleted.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return serviceapi.ErrProjectNotFound
		}
		return err
	}

	if err := insertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeProject,
		deleted.ID,
		domain.AuditActionProjectDelete,
		map[string]interface{}{"name": deleted.Name},
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
