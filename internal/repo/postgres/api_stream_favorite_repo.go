package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

// APIStreamFavoriteRepo manages stream_favorites (API, tenant-scoped).
type APIStreamFavoriteRepo struct {
	db *sql.DB
}

// NewAPIStreamFavoriteRepo returns a new APIStreamFavoriteRepo.
func NewAPIStreamFavoriteRepo(db *sql.DB) *APIStreamFavoriteRepo {
	return &APIStreamFavoriteRepo{db: db}
}

// EnsureStreamInCompany checks stream exists and belongs to company.
func (r *APIStreamFavoriteRepo) EnsureStreamInCompany(ctx context.Context, companyID int64, streamID int64) error {
	var id int64
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id FROM streams WHERE company_id = $1 AND id = $2`,
		companyID,
		streamID,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrStreamNotFound
		}
		return err
	}
	return nil
}

// AddFavorite inserts or updates row: ensure row exists (is_pinned unchanged unless already false).
func (r *APIStreamFavoriteRepo) AddFavorite(ctx context.Context, userID int64, companyID int64, streamID int64) error {
	if err := r.EnsureStreamInCompany(ctx, companyID, streamID); err != nil {
		return err
	}
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO stream_favorites (user_id, stream_id, is_pinned, sort_order)
         VALUES ($1, $2, FALSE, 0)
         ON CONFLICT (user_id, stream_id) DO NOTHING`,
		userID,
		streamID,
	)
	return err
}

// RemoveFavorite deletes the row.
func (r *APIStreamFavoriteRepo) RemoveFavorite(ctx context.Context, userID int64, companyID int64, streamID int64) error {
	result, err := r.db.ExecContext(
		ctx,
		`DELETE FROM stream_favorites
         WHERE user_id = $1 AND stream_id IN (SELECT id FROM streams WHERE company_id = $2 AND id = $3)`,
		userID,
		companyID,
		streamID,
	)
	if err != nil {
		return err
	}
	_, _ = result.RowsAffected()
	return nil
}

// AddPin sets is_pinned=true and optional sort_order.
func (r *APIStreamFavoriteRepo) AddPin(ctx context.Context, userID int64, companyID int64, streamID int64, sortOrder *int) error {
	if err := r.EnsureStreamInCompany(ctx, companyID, streamID); err != nil {
		return err
	}
	order := 0
	if sortOrder != nil {
		order = *sortOrder
	}
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO stream_favorites (user_id, stream_id, is_pinned, sort_order)
         VALUES ($1, $2, TRUE, $3)
         ON CONFLICT (user_id, stream_id) DO UPDATE SET is_pinned = TRUE, sort_order = $3`,
		userID,
		streamID,
		order,
	)
	return err
}

// RemovePin sets is_pinned=false.
func (r *APIStreamFavoriteRepo) RemovePin(ctx context.Context, userID int64, companyID int64, streamID int64) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE stream_favorites SET is_pinned = FALSE
         WHERE user_id = $1 AND stream_id IN (SELECT id FROM streams WHERE company_id = $2 AND id = $3)`,
		userID,
		companyID,
		streamID,
	)
	if err != nil {
		return err
	}
	_, _ = result.RowsAffected()
	return nil
}

// ListFavorites returns streams that user favorited in company, pinned first then by sort_order.
func (r *APIStreamFavoriteRepo) ListFavorites(ctx context.Context, userID int64, companyID int64) ([]domain.StreamWithFavorite, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT s.id, s.company_id, s.project_id, s.name, s.source_type, s.source_url, s.url, s.is_active, s.created_at, s.updated_at,
                COALESCE(f.is_pinned, FALSE), COALESCE(f.sort_order, 0)
         FROM stream_favorites f
         JOIN streams s ON s.id = f.stream_id AND s.company_id = $1
         WHERE f.user_id = $2
         ORDER BY f.is_pinned DESC, f.sort_order ASC, s.id ASC`,
		companyID,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.StreamWithFavorite
	for rows.Next() {
		var row domain.StreamWithFavorite
		if err := rows.Scan(
			&row.Stream.ID,
			&row.Stream.CompanyID,
			&row.Stream.ProjectID,
			&row.Stream.Name,
			&row.Stream.SourceType,
			&row.Stream.SourceURL,
			&row.Stream.URL,
			&row.Stream.IsActive,
			&row.Stream.CreatedAt,
			&row.Stream.UpdatedAt,
			&row.IsPinned,
			&row.SortOrder,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
