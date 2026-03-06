package api

import (
	"context"
	"errors"

	"github.com/terrynullson/hls_mntrng/internal/domain"
)

// StreamFavoriteStore for favorites/pins (API).
type StreamFavoriteStore interface {
	EnsureStreamInCompany(ctx context.Context, companyID int64, streamID int64) error
	AddFavorite(ctx context.Context, userID int64, companyID int64, streamID int64) error
	RemoveFavorite(ctx context.Context, userID int64, companyID int64, streamID int64) error
	AddPin(ctx context.Context, userID int64, companyID int64, streamID int64, sortOrder *int) error
	RemovePin(ctx context.Context, userID int64, companyID int64, streamID int64) error
	ListFavorites(ctx context.Context, userID int64, companyID int64) ([]domain.StreamWithFavorite, error)
}

// StreamFavoriteService handles favorite/pin actions.
type StreamFavoriteService struct {
	store StreamFavoriteStore
}

// NewStreamFavoriteService returns a new StreamFavoriteService.
func NewStreamFavoriteService(store StreamFavoriteStore) *StreamFavoriteService {
	return &StreamFavoriteService{store: store}
}

// AddFavorite adds stream to user favorites in company scope.
func (s *StreamFavoriteService) AddFavorite(ctx context.Context, userID int64, companyID int64, streamID int64) error {
	err := s.store.AddFavorite(ctx, userID, companyID, streamID)
	if err != nil {
		if errors.Is(err, domain.ErrStreamNotFound) {
			return NewNotFoundError("stream not found", map[string]interface{}{"stream_id": streamID})
		}
		return NewInternalError()
	}
	return nil
}

// RemoveFavorite removes stream from user favorites.
func (s *StreamFavoriteService) RemoveFavorite(ctx context.Context, userID int64, companyID int64, streamID int64) error {
	_ = s.store.RemoveFavorite(ctx, userID, companyID, streamID)
	return nil
}

// AddPin pins stream for user (optional sort_order).
func (s *StreamFavoriteService) AddPin(ctx context.Context, userID int64, companyID int64, streamID int64, sortOrder *int) error {
	err := s.store.AddPin(ctx, userID, companyID, streamID, sortOrder)
	if err != nil {
		if errors.Is(err, domain.ErrStreamNotFound) {
			return NewNotFoundError("stream not found", map[string]interface{}{"stream_id": streamID})
		}
		return NewInternalError()
	}
	return nil
}

// RemovePin unpins stream.
func (s *StreamFavoriteService) RemovePin(ctx context.Context, userID int64, companyID int64, streamID int64) error {
	_ = s.store.RemovePin(ctx, userID, companyID, streamID)
	return nil
}

// ListFavorites returns user's favorite streams in company (pinned first).
func (s *StreamFavoriteService) ListFavorites(ctx context.Context, userID int64, companyID int64) ([]domain.StreamWithFavorite, error) {
	items, err := s.store.ListFavorites(ctx, userID, companyID)
	if err != nil {
		return nil, NewInternalError()
	}
	return items, nil
}
