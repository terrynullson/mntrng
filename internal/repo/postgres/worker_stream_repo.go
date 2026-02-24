package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

func (r *WorkerRepo) LoadStreamURL(ctx context.Context, companyID int64, streamID int64) (string, error) {
	var streamURL string
	var sourceType string
	err := r.db.QueryRowContext(
		ctx,
		`SELECT source_type, source_url
         FROM streams
         WHERE company_id = $1
           AND id = $2`,
		companyID,
		streamID,
	).Scan(&sourceType, &streamURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("stream not found in tenant context")
		}
		return "", err
	}
	if strings.TrimSpace(sourceType) != "HLS" {
		return "", errors.New("stream source type is not HLS")
	}
	streamURL = strings.TrimSpace(streamURL)
	if streamURL == "" {
		return "", errors.New("stream url is empty")
	}
	return streamURL, nil
}
