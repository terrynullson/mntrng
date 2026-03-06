package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

type APIEmbedWhitelistRepo struct {
	db *sql.DB
}

func NewAPIEmbedWhitelistRepo(db *sql.DB) *APIEmbedWhitelistRepo {
	return &APIEmbedWhitelistRepo{db: db}
}

func (r *APIEmbedWhitelistRepo) ListEmbedWhitelist(ctx context.Context, companyID int64) ([]domain.EmbedWhitelistItem, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, company_id, domain, enabled, created_at, created_by_user_id
         FROM embed_whitelist
         WHERE company_id = $1
         ORDER BY id ASC`,
		companyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.EmbedWhitelistItem, 0)
	for rows.Next() {
		var item domain.EmbedWhitelistItem
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.Domain, &item.Enabled, &item.CreatedAt, &item.CreatedByUserID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *APIEmbedWhitelistRepo) CreateEmbedWhitelist(ctx context.Context, companyID int64, domainName string, createdByUserID int64) (domain.EmbedWhitelistItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.EmbedWhitelistItem{}, err
	}
	defer tx.Rollback()
	var item domain.EmbedWhitelistItem
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO embed_whitelist (company_id, domain, enabled, created_by_user_id)
         VALUES ($1, $2, TRUE, $3)
         RETURNING id, company_id, domain, enabled, created_at, created_by_user_id`,
		companyID,
		strings.ToLower(strings.TrimSpace(domainName)),
		createdByUserID,
	).Scan(&item.ID, &item.CompanyID, &item.Domain, &item.Enabled, &item.CreatedAt, &item.CreatedByUserID)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.EmbedWhitelistItem{}, domain.ErrEmbedWhitelistAlreadyExists
		}
		return domain.EmbedWhitelistItem{}, err
	}
	if err := InsertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		strconv.FormatInt(createdByUserID, 10),
		domain.AuditEntityTypeEmbedWhitelist,
		item.ID,
		domain.AuditActionEmbedWhitelistAdd,
		map[string]interface{}{"domain": item.Domain, "enabled": item.Enabled},
	); err != nil {
		return domain.EmbedWhitelistItem{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.EmbedWhitelistItem{}, err
	}
	return item, nil
}

func (r *APIEmbedWhitelistRepo) PatchEmbedWhitelist(ctx context.Context, companyID int64, id int64, enabled bool, actorUserID int64) (domain.EmbedWhitelistItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.EmbedWhitelistItem{}, err
	}
	defer tx.Rollback()
	var item domain.EmbedWhitelistItem
	err = tx.QueryRowContext(
		ctx,
		`UPDATE embed_whitelist
         SET enabled = $3
         WHERE company_id = $1 AND id = $2
         RETURNING id, company_id, domain, enabled, created_at, created_by_user_id`,
		companyID,
		id,
		enabled,
	).Scan(&item.ID, &item.CompanyID, &item.Domain, &item.Enabled, &item.CreatedAt, &item.CreatedByUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.EmbedWhitelistItem{}, domain.ErrEmbedWhitelistNotFound
		}
		return domain.EmbedWhitelistItem{}, err
	}
	if err := InsertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		strconv.FormatInt(actorUserID, 10),
		domain.AuditEntityTypeEmbedWhitelist,
		item.ID,
		domain.AuditActionEmbedWhitelistToggle,
		map[string]interface{}{"domain": item.Domain, "enabled": item.Enabled},
	); err != nil {
		return domain.EmbedWhitelistItem{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.EmbedWhitelistItem{}, err
	}
	return item, nil
}

func (r *APIEmbedWhitelistRepo) DeleteEmbedWhitelist(ctx context.Context, companyID int64, id int64, actorUserID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var deleted domain.EmbedWhitelistItem
	err = tx.QueryRowContext(
		ctx,
		`DELETE FROM embed_whitelist
         WHERE company_id = $1 AND id = $2
         RETURNING id, company_id, domain, enabled, created_at, created_by_user_id`,
		companyID,
		id,
	).Scan(&deleted.ID, &deleted.CompanyID, &deleted.Domain, &deleted.Enabled, &deleted.CreatedAt, &deleted.CreatedByUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrEmbedWhitelistNotFound
		}
		return err
	}
	if err := InsertAuditLogTx(
		ctx,
		tx,
		companyID,
		domain.AuditActorTypeAPI,
		strconv.FormatInt(actorUserID, 10),
		domain.AuditEntityTypeEmbedWhitelist,
		deleted.ID,
		domain.AuditActionEmbedWhitelistRemove,
		map[string]interface{}{"domain": deleted.Domain},
	); err != nil {
		return err
	}
	return tx.Commit()
}
