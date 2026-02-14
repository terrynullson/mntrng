package checks

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

func FetchPlaylist(ctx context.Context, streamURL string, playlistTimeout time.Duration) (string, error) {
	requestCtx, cancel := context.WithTimeout(ctx, playlistTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, streamURL, nil)
	if err != nil {
		return "", err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", errors.New("playlist request returned non-2xx")
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	playlistBody := string(body)
	if !strings.Contains(playlistBody, "#EXTM3U") {
		return "", errors.New("playlist does not contain EXTM3U marker")
	}

	return playlistBody, nil
}
