package worker

import (
	"context"
	"log"
	"time"
)

type App struct {
	pollInterval     time.Duration
	cleanupInterval  time.Duration
	retryMax         int
	retryBackoff     time.Duration
	isRetryable      func(error) bool
	processCycle     func(context.Context) error
	retentionCleanup func(context.Context) error
}

func NewApp(
	pollInterval time.Duration,
	cleanupInterval time.Duration,
	retryMax int,
	retryBackoff time.Duration,
	isRetryable func(error) bool,
	processCycle func(context.Context) error,
	retentionCleanup func(context.Context) error,
) *App {
	return &App{
		pollInterval:     pollInterval,
		cleanupInterval:  cleanupInterval,
		retryMax:         retryMax,
		retryBackoff:     retryBackoff,
		isRetryable:      isRetryable,
		processCycle:     processCycle,
		retentionCleanup: retentionCleanup,
	}
}

func (a *App) Run(ctx context.Context) {
	if err := a.processCycleWithRetry(ctx); err != nil {
		log.Printf("worker cycle failed: %v", err)
	}
	if err := a.runRetentionCleanupWithRetry(ctx); err != nil {
		log.Printf("worker retention cleanup failed: %v", err)
	}

	cycleTicker := time.NewTicker(a.pollInterval)
	defer cycleTicker.Stop()
	cleanupTicker := time.NewTicker(a.cleanupInterval)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("worker skeleton stopped")
			return
		case currentTime := <-cycleTicker.C:
			log.Printf("worker skeleton heartbeat: %s", currentTime.UTC().Format(time.RFC3339))
			if err := a.processCycleWithRetry(ctx); err != nil {
				log.Printf("worker cycle failed: %v", err)
			}
		case currentTime := <-cleanupTicker.C:
			log.Printf("worker retention cleanup heartbeat: %s", currentTime.UTC().Format(time.RFC3339))
			if err := a.runRetentionCleanupWithRetry(ctx); err != nil {
				log.Printf("worker retention cleanup failed: %v", err)
			}
		}
	}
}

func (a *App) processCycleWithRetry(ctx context.Context) error {
	startedAt := time.Now()
	for attempt := 0; ; attempt++ {
		err := a.processCycle(ctx)
		if err == nil {
			observeWorkerCycle("ok", startedAt)
			return nil
		}
		if !a.isRetryable(err) || attempt >= a.retryMax {
			observeWorkerCycle("error", startedAt)
			return err
		}

		backoff := a.retryBackoff * time.Duration(1<<attempt)
		log.Printf("worker retry attempt=%d backoff=%s err=%v", attempt+1, backoff, err)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return err
		}
	}
}

func (a *App) runRetentionCleanupWithRetry(ctx context.Context) error {
	for attempt := 0; ; attempt++ {
		err := a.retentionCleanup(ctx)
		if err == nil {
			observeRetentionCleanup("ok")
			return nil
		}
		if !a.isRetryable(err) || attempt >= a.retryMax {
			observeRetentionCleanup("error")
			return err
		}

		backoff := a.retryBackoff * time.Duration(1<<attempt)
		log.Printf("worker retention cleanup retry attempt=%d backoff=%s err=%v", attempt+1, backoff, err)
		if err := sleepWithContext(ctx, backoff); err != nil {
			return err
		}
	}
}
