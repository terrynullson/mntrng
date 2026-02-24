package api

import (
	"context"
	"errors"
	"strings"

	"github.com/example/hls-monitoring-platform/internal/domain"
)

type EmbedWhitelistStore interface {
	ListEmbedWhitelist(ctx context.Context, companyID int64) ([]domain.EmbedWhitelistItem, error)
	CreateEmbedWhitelist(ctx context.Context, companyID int64, domainName string, createdByUserID int64) (domain.EmbedWhitelistItem, error)
	PatchEmbedWhitelist(ctx context.Context, companyID int64, id int64, enabled bool, actorUserID int64) (domain.EmbedWhitelistItem, error)
	DeleteEmbedWhitelist(ctx context.Context, companyID int64, id int64, actorUserID int64) error
}

type EmbedWhitelistService struct {
	store EmbedWhitelistStore
}

func NewEmbedWhitelistService(store EmbedWhitelistStore) *EmbedWhitelistService {
	return &EmbedWhitelistService{store: store}
}

func (s *EmbedWhitelistService) List(ctx context.Context, companyID int64) ([]domain.EmbedWhitelistItem, error) {
	items, err := s.store.ListEmbedWhitelist(ctx, companyID)
	if err != nil {
		return nil, NewInternalError()
	}
	return items, nil
}

func (s *EmbedWhitelistService) Create(ctx context.Context, companyID int64, rawDomain string, userID int64) (domain.EmbedWhitelistItem, error) {
	domainName, err := normalizeWhitelistDomain(rawDomain)
	if err != nil {
		return domain.EmbedWhitelistItem{}, err
	}
	item, createErr := s.store.CreateEmbedWhitelist(ctx, companyID, domainName, userID)
	if createErr == nil {
		return item, nil
	}
	if errors.Is(createErr, domain.ErrEmbedWhitelistAlreadyExists) {
		return domain.EmbedWhitelistItem{}, NewConflictError("domain already exists in embed whitelist", map[string]interface{}{"domain": domainName})
	}
	return domain.EmbedWhitelistItem{}, NewInternalError()
}

func (s *EmbedWhitelistService) Patch(ctx context.Context, companyID int64, id int64, enabled *bool, userID int64) (domain.EmbedWhitelistItem, error) {
	if enabled == nil {
		return domain.EmbedWhitelistItem{}, NewValidationError("enabled is required", map[string]interface{}{"field": "enabled"})
	}
	item, patchErr := s.store.PatchEmbedWhitelist(ctx, companyID, id, *enabled, userID)
	if patchErr == nil {
		return item, nil
	}
	if errors.Is(patchErr, domain.ErrEmbedWhitelistNotFound) {
		return domain.EmbedWhitelistItem{}, NewNotFoundError("embed whitelist item not found", map[string]interface{}{"id": id})
	}
	return domain.EmbedWhitelistItem{}, NewInternalError()
}

func (s *EmbedWhitelistService) Delete(ctx context.Context, companyID int64, id int64, userID int64) error {
	err := s.store.DeleteEmbedWhitelist(ctx, companyID, id, userID)
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrEmbedWhitelistNotFound) {
		return NewNotFoundError("embed whitelist item not found", map[string]interface{}{"id": id})
	}
	return NewInternalError()
}

func normalizeWhitelistDomain(rawDomain string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(rawDomain))
	if trimmed == "" {
		return "", NewValidationError("domain is required", map[string]interface{}{"field": "domain"})
	}
	if strings.Contains(trimmed, "://") || strings.Contains(trimmed, "/") || strings.Contains(trimmed, ":") {
		return "", NewValidationError("domain must not contain scheme/path/port", map[string]interface{}{"field": "domain"})
	}
	if strings.HasPrefix(trimmed, ".") || strings.HasSuffix(trimmed, ".") {
		return "", NewValidationError("domain format is invalid", map[string]interface{}{"field": "domain"})
	}
	if !strings.Contains(trimmed, ".") {
		return "", NewValidationError("domain format is invalid", map[string]interface{}{"field": "domain"})
	}
	return trimmed, nil
}
