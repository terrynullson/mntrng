package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/terrynullson/mntrng/internal/domain"
)

type APIStreamRepo struct {
	db *sql.DB
}

func NewAPIStreamRepo(db *sql.DB) *APIStreamRepo {
	return &APIStreamRepo{db: db}
}

func (r *APIStreamRepo) CreateStream(
	ctx context.Context,
	companyID int64,
	projectID int64,
	name string,
	sourceType string,
	sourceURL string,
	isActive bool,
) (domain.Stream, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Stream{}, err
	}
	defer tx.Rollback()

	var item domain.Stream
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO streams (company_id, project_id, name, source_type, source_url, url, is_active)
         SELECT $1, $2, $3, $4, $5, $6, $7
         WHERE EXISTS (
             SELECT 1 FROM projects p
             WHERE p.company_id = $1 AND p.id = $2
         )
         RETURNING id, company_id, project_id, name, source_type, source_url, url, is_active, created_at, updated_at`,
		companyID,
		projectID,
		name,
		sourceType,
		sourceURL,
		sourceURL,
		isActive,
	).Scan(&item.ID, &item.CompanyID, &item.ProjectID, &item.Name, &item.SourceType, &item.SourceURL, &item.URL, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || isForeignKeyViolation(err) {
			return domain.Stream{}, domain.ErrStreamProjectMiss
		}
		if isUniqueViolation(err) {
			return domain.Stream{}, domain.ErrStreamAlreadyExists
		}
		return domain.Stream{}, err
	}

	if err := InsertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeStream,
		item.ID,
		domain.AuditActionStreamCreate,
		map[string]interface{}{
			"project_id":  item.ProjectID,
			"name":        item.Name,
			"source_type": item.SourceType,
			"source_url":  item.SourceURL,
			"url":         item.URL,
			"is_active":   item.IsActive,
		},
	); err != nil {
		return domain.Stream{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Stream{}, err
	}

	return item, nil
}

func (r *APIStreamRepo) ListStreams(ctx context.Context, companyID int64, filter domain.StreamListFilter) ([]domain.Stream, error) {
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

	// Cap list size for production scalability; no pagination params yet to avoid contract change.
	const maxStreamsList = 500
	query := fmt.Sprintf(
		`SELECT id, company_id, project_id, name, source_type, source_url, url, is_active, created_at, updated_at
         FROM streams
         WHERE %s
         ORDER BY id ASC
         LIMIT %d`,
		strings.Join(conditions, " AND "),
		maxStreamsList,
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Stream, 0)
	for rows.Next() {
		var item domain.Stream
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.ProjectID, &item.Name, &item.SourceType, &item.SourceURL, &item.URL, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *APIStreamRepo) ListLatestStatuses(ctx context.Context, companyID int64) ([]domain.StreamLatestStatus, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT s.id,
                cr.status,
                cr.created_at
         FROM streams s
         LEFT JOIN LATERAL (
             SELECT c.status, c.created_at
             FROM check_results c
             WHERE c.company_id = s.company_id
               AND c.stream_id = s.id
             ORDER BY c.created_at DESC, c.id DESC
             LIMIT 1
         ) cr ON TRUE
         WHERE s.company_id = $1
         ORDER BY s.id ASC`,
		companyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.StreamLatestStatus, 0)
	for rows.Next() {
		var item domain.StreamLatestStatus
		var status sql.NullString
		var createdAt sql.NullTime
		if err := rows.Scan(&item.StreamID, &status, &createdAt); err != nil {
			return nil, err
		}
		if status.Valid {
			formatted := formatCheckResultStatus(status.String)
			item.Status = &formatted
		}
		if createdAt.Valid {
			createdAtUTC := createdAt.Time.UTC()
			item.LastCheckAt = &createdAtUTC
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *APIStreamRepo) GetStream(ctx context.Context, companyID int64, streamID int64) (domain.Stream, error) {
	var item domain.Stream
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, company_id, project_id, name, source_type, source_url, url, is_active, created_at, updated_at
         FROM streams
         WHERE company_id = $1 AND id = $2`,
		companyID,
		streamID,
	).Scan(&item.ID, &item.CompanyID, &item.ProjectID, &item.Name, &item.SourceType, &item.SourceURL, &item.URL, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Stream{}, domain.ErrStreamNotFound
		}
		return domain.Stream{}, err
	}
	return item, nil
}

func (r *APIStreamRepo) PatchStream(
	ctx context.Context,
	companyID int64,
	streamID int64,
	patch domain.StreamPatchInput,
) (domain.Stream, error) {
	query, args, changePayload := buildStreamPatchQuery(patch, companyID, streamID)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Stream{}, err
	}
	defer tx.Rollback()

	item, err := runStreamPatchQueryTx(ctx, tx, query, args)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Stream{}, domain.ErrStreamNotFound
		}
		if isUniqueViolation(err) {
			return domain.Stream{}, domain.ErrStreamAlreadyExists
		}
		return domain.Stream{}, err
	}

	if err := InsertAuditLogTx(
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

func (r *APIStreamRepo) DeleteStream(ctx context.Context, companyID int64, streamID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var deleted domain.Stream
	err = tx.QueryRowContext(
		ctx,
		`DELETE FROM streams
         WHERE company_id = $1 AND id = $2
         RETURNING id, company_id, project_id, name, source_type, source_url, url, is_active, created_at, updated_at`,
		companyID,
		streamID,
	).Scan(
		&deleted.ID,
		&deleted.CompanyID,
		&deleted.ProjectID,
		&deleted.Name,
		&deleted.SourceType,
		&deleted.SourceURL,
		&deleted.URL,
		&deleted.IsActive,
		&deleted.CreatedAt,
		&deleted.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrStreamNotFound
		}
		return err
	}

	if err := InsertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		domain.AuditActorIDSystem,
		domain.AuditEntityTypeStream,
		deleted.ID,
		domain.AuditActionStreamDelete,
		map[string]interface{}{
			"project_id":  deleted.ProjectID,
			"name":        deleted.Name,
			"source_type": deleted.SourceType,
			"source_url":  deleted.SourceURL,
			"url":         deleted.URL,
			"is_active":   deleted.IsActive,
		},
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func buildStreamPatchQuery(patch domain.StreamPatchInput, companyID int64, streamID int64) (string, []interface{}, map[string]interface{}) {
	setClauses := make([]string, 0, 6)
	args := make([]interface{}, 0, 5)
	changePayload := make(map[string]interface{})
	nextPlaceholder := 1

	if patch.SourceType != nil {
		setClauses = append(setClauses, fmt.Sprintf("source_type = $%d", nextPlaceholder))
		args = append(args, *patch.SourceType)
		changePayload["source_type"] = *patch.SourceType
		nextPlaceholder++
	}
	if patch.SourceURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("source_url = $%d", nextPlaceholder))
		args = append(args, *patch.SourceURL)
		changePayload["source_url"] = *patch.SourceURL
		nextPlaceholder++
	}
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
         RETURNING id, company_id, project_id, name, source_type, source_url, url, is_active, created_at, updated_at`,
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
		&item.SourceType,
		&item.SourceURL,
		&item.URL,
		&item.IsActive,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func (r *APIStreamRepo) IsEmbedDomainAllowed(ctx context.Context, companyID int64, host string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1
         FROM embed_whitelist
         WHERE company_id = $1
           AND enabled = TRUE
           AND ($2 = domain OR $2 LIKE ('%.' || domain))
         LIMIT 1`,
		companyID,
		strings.ToLower(strings.TrimSpace(host)),
	).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}
