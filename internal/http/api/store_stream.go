package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/domain"
	serviceapi "github.com/example/hls-monitoring-platform/internal/service/api"
)

type streamStore struct {
	db *sql.DB
}

func newStreamStore(db *sql.DB) *streamStore {
	return &streamStore{db: db}
}

func (s *streamStore) CreateStream(
	ctx context.Context,
	companyID int64,
	projectID int64,
	name string,
	url string,
	isActive bool,
) (domain.Stream, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Stream{}, err
	}
	defer tx.Rollback()

	var item domain.Stream
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO streams (company_id, project_id, name, url, is_active)
         SELECT $1, $2, $3, $4, $5
         WHERE EXISTS (
             SELECT 1 FROM projects p
             WHERE p.company_id = $1 AND p.id = $2
         )
         RETURNING id, company_id, project_id, name, url, is_active, created_at, updated_at`,
		companyID,
		projectID,
		name,
		url,
		isActive,
	).Scan(&item.ID, &item.CompanyID, &item.ProjectID, &item.Name, &item.URL, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || isForeignKeyViolation(err) {
			return domain.Stream{}, serviceapi.ErrStreamProjectMiss
		}
		if isUniqueViolation(err) {
			return domain.Stream{}, serviceapi.ErrStreamAlreadyExists
		}
		return domain.Stream{}, err
	}

	if err := insertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeStream,
		item.ID,
		domain.AuditActionStreamCreate,
		map[string]interface{}{
			"project_id": item.ProjectID,
			"name":       item.Name,
			"url":        item.URL,
			"is_active":  item.IsActive,
		},
	); err != nil {
		return domain.Stream{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Stream{}, err
	}

	return item, nil
}

func (s *streamStore) ListStreams(ctx context.Context, companyID int64, filter serviceapi.StreamListFilter) ([]domain.Stream, error) {
	args := []interface{}{companyID}
	conditions := []string{"company_id = $1"}
	nextPlaceholder := 2

	if filter.ProjectID != nil {
		conditions = append(conditions, fmt.Sprintf("project_id = $%d", nextPlaceholder))
		args = append(args, *filter.ProjectID)
		nextPlaceholder++
	}

	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", nextPlaceholder))
		args = append(args, *filter.IsActive)
		nextPlaceholder++
	}

	query := fmt.Sprintf(
		`SELECT id, company_id, project_id, name, url, is_active, created_at, updated_at
         FROM streams
         WHERE %s
         ORDER BY id ASC`,
		strings.Join(conditions, " AND "),
	)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Stream, 0)
	for rows.Next() {
		var item domain.Stream
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.ProjectID, &item.Name, &item.URL, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *streamStore) GetStream(ctx context.Context, companyID int64, streamID int64) (domain.Stream, error) {
	var item domain.Stream
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, project_id, name, url, is_active, created_at, updated_at
         FROM streams
         WHERE company_id = $1 AND id = $2`,
		companyID,
		streamID,
	).Scan(&item.ID, &item.CompanyID, &item.ProjectID, &item.Name, &item.URL, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Stream{}, serviceapi.ErrStreamNotFound
		}
		return domain.Stream{}, err
	}
	return item, nil
}

func (s *streamStore) PatchStream(
	ctx context.Context,
	companyID int64,
	streamID int64,
	patch serviceapi.StreamPatchInput,
) (domain.Stream, error) {
	query, args, changePayload := buildStreamPatchQuery(patch, companyID, streamID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Stream{}, err
	}
	defer tx.Rollback()

	item, err := runStreamPatchQueryTx(ctx, tx, query, args)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Stream{}, serviceapi.ErrStreamNotFound
		}
		if isUniqueViolation(err) {
			return domain.Stream{}, serviceapi.ErrStreamAlreadyExists
		}
		return domain.Stream{}, err
	}

	if err := insertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeStream,
		item.ID,
		domain.AuditActionStreamUpdate,
		map[string]interface{}{
			"project_id": item.ProjectID,
			"changes":    changePayload,
		},
	); err != nil {
		return domain.Stream{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Stream{}, err
	}
	return item, nil
}

func buildStreamPatchQuery(patch serviceapi.StreamPatchInput, companyID int64, streamID int64) (string, []interface{}, map[string]interface{}) {
	setClauses := make([]string, 0, 3)
	args := make([]interface{}, 0, 5)
	changePayload := make(map[string]interface{})
	nextPlaceholder := 1

	if patch.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", nextPlaceholder))
		args = append(args, *patch.Name)
		changePayload["name"] = *patch.Name
		nextPlaceholder++
	}
	if patch.URL != nil {
		setClauses = append(setClauses, fmt.Sprintf("url = $%d", nextPlaceholder))
		args = append(args, *patch.URL)
		changePayload["url"] = *patch.URL
		nextPlaceholder++
	}
	if patch.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", nextPlaceholder))
		args = append(args, *patch.IsActive)
		changePayload["is_active"] = *patch.IsActive
		nextPlaceholder++
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	companyPlaceholder := nextPlaceholder
	streamPlaceholder := nextPlaceholder + 1
	args = append(args, companyID, streamID)

	query := fmt.Sprintf(
		`UPDATE streams
         SET %s
         WHERE company_id = $%d
           AND id = $%d
           AND EXISTS (
               SELECT 1 FROM projects p
               WHERE p.id = streams.project_id
                 AND p.company_id = streams.company_id
           )
         RETURNING id, company_id, project_id, name, url, is_active, created_at, updated_at`,
		strings.Join(setClauses, ", "),
		companyPlaceholder,
		streamPlaceholder,
	)
	return query, args, changePayload
}

func runStreamPatchQueryTx(ctx context.Context, tx *sql.Tx, query string, args []interface{}) (domain.Stream, error) {
	var item domain.Stream
	err := tx.QueryRowContext(ctx, query, args...).Scan(
		&item.ID,
		&item.CompanyID,
		&item.ProjectID,
		&item.Name,
		&item.URL,
		&item.IsActive,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func (s *streamStore) DeleteStream(ctx context.Context, companyID int64, streamID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var deleted domain.Stream
	err = tx.QueryRowContext(
		ctx,
		`DELETE FROM streams
         WHERE company_id = $1 AND id = $2
         RETURNING id, company_id, project_id, name, url, is_active, created_at, updated_at`,
		companyID,
		streamID,
	).Scan(
		&deleted.ID,
		&deleted.CompanyID,
		&deleted.ProjectID,
		&deleted.Name,
		&deleted.URL,
		&deleted.IsActive,
		&deleted.CreatedAt,
		&deleted.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return serviceapi.ErrStreamNotFound
		}
		return err
	}

	if err := insertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeStream,
		deleted.ID,
		domain.AuditActionStreamDelete,
		map[string]interface{}{
			"project_id": deleted.ProjectID,
			"name":       deleted.Name,
			"url":        deleted.URL,
			"is_active":  deleted.IsActive,
		},
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
