package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	baseURL := strings.TrimSuffix(strings.TrimSpace(os.Getenv("SCHEDULER_API_BASE_URL")), "/")
	token := strings.TrimSpace(os.Getenv("SCHEDULER_ACCESS_TOKEN"))
	intervalMin := getIntEnv("SCHEDULER_INTERVAL_MIN", 30)
	enabled := getBoolEnv("SCHEDULER_ENABLED", true)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if !enabled {
		log.Println("scheduler disabled by SCHEDULER_ENABLED; holding process")
		<-ctx.Done()
		return
	}
	if baseURL == "" {
		log.Fatal("SCHEDULER_API_BASE_URL is required")
	}
	if token == "" {
		log.Fatal("SCHEDULER_ACCESS_TOKEN is required")
	}
	if intervalMin < 1 {
		intervalMin = 30
	}

	interval := time.Duration(intervalMin) * time.Minute
	client := &http.Client{Timeout: 15 * time.Second}

	log.Printf("scheduler started: api=%s interval=%s", baseURL, interval)

	runOnce(ctx, client, baseURL, token)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("scheduler stopped")
			return
		case <-ticker.C:
			runOnce(ctx, client, baseURL, token)
		}
	}
}

func runOnce(ctx context.Context, client *http.Client, baseURL, token string) {
	companies, err := fetchCompanies(ctx, client, baseURL, token)
	if err != nil {
		log.Printf("scheduler fetch companies: %v", err)
		return
	}
	var enqueued, skipped, failed int
	for _, c := range companies {
		streams, err := fetchStreams(ctx, client, baseURL, token, c.ID)
		if err != nil {
			log.Printf("scheduler fetch streams company_id=%d: %v", c.ID, err)
			failed++
			continue
		}
		for _, s := range streams {
			if !s.IsActive {
				continue
			}
			status, err := enqueueCheckJob(ctx, client, baseURL, token, c.ID, s.ID)
			if err != nil {
				log.Printf("scheduler enqueue company_id=%d stream_id=%d: %v", c.ID, s.ID, err)
				failed++
				continue
			}
			switch status {
			case http.StatusAccepted:
				enqueued++
			case http.StatusConflict:
				skipped++
			default:
				log.Printf("scheduler enqueue company_id=%d stream_id=%d: status=%d", c.ID, s.ID, status)
				failed++
			}
		}
	}
	log.Printf("scheduler cycle: enqueued=%d skipped=%d failed=%d companies=%d", enqueued, skipped, failed, len(companies))
}

type companyItem struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type companyListResponse struct {
	Items []companyItem `json:"items"`
}

type streamItem struct {
	ID        int64 `json:"id"`
	CompanyID int64 `json:"company_id"`
	IsActive  bool  `json:"is_active"`
}

type streamListResponse struct {
	Items []streamItem `json:"items"`
}

func fetchCompanies(ctx context.Context, client *http.Client, baseURL, token string) ([]companyItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/companies", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &httpErr{status: resp.StatusCode}
	}
	var out companyListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func fetchStreams(ctx context.Context, client *http.Client, baseURL, token string, companyID int64) ([]streamItem, error) {
	url := baseURL + "/api/v1/companies/" + strconv.FormatInt(companyID, 10) + "/streams?is_active=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &httpErr{status: resp.StatusCode}
	}
	var out streamListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func enqueueCheckJob(ctx context.Context, client *http.Client, baseURL, token string, companyID, streamID int64) (int, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	body := map[string]string{"planned_at": now}
	raw, _ := json.Marshal(body)
	url := baseURL + "/api/v1/companies/" + strconv.FormatInt(companyID, 10) + "/streams/" + strconv.FormatInt(streamID, 10) + "/check-jobs"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

type httpErr struct{ status int }

func (e *httpErr) Error() string { return "HTTP " + strconv.Itoa(e.status) }

func getIntEnv(key string, defaultVal int) int {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}

func getBoolEnv(key string, defaultVal bool) bool {
	s := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if s == "" {
		return defaultVal
	}
	return s == "1" || s == "true" || s == "yes"
}
