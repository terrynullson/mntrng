package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter limits the number of requests per key (e.g. IP) per minute.
// Allow returns true if the request is allowed, false if rate limit exceeded.
type Limiter interface {
	Allow(ctx context.Context, key string) (allowed bool, err error)
}

type rateWindow struct {
	count int
	start time.Time
}

// InMemLimiter is a fixed-window in-memory rate limiter (single instance).
type InMemLimiter struct {
	mu         sync.Mutex
	perMin     int
	windows    map[string]*rateWindow
	windowSize time.Duration
}

// NewInMemLimiter creates an in-memory limiter allowing perMin requests per key per minute.
func NewInMemLimiter(perMin int) *InMemLimiter {
	if perMin < 1 {
		perMin = 10
	}
	return &InMemLimiter{
		perMin:     perMin,
		windows:    make(map[string]*rateWindow),
		windowSize: time.Minute,
	}
}

// Allow returns true if the key is under the limit.
func (l *InMemLimiter) Allow(ctx context.Context, key string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().UTC()
	w := l.windows[key]
	if w == nil || now.Sub(w.start) >= l.windowSize {
		l.windows[key] = &rateWindow{count: 1, start: now}
		return true, nil
	}
	if w.count >= l.perMin {
		return false, nil
	}
	w.count++
	return true, nil
}

// RedisLimiter uses Redis for fixed-window rate limiting (shared across API instances).
type RedisLimiter struct {
	rdb       *redis.Client
	perMin    int
	keyPrefix string
}

// NewRedisLimiter creates a Redis-backed limiter. perMin is max requests per key per minute.
func NewRedisLimiter(rdb *redis.Client, perMin int) *RedisLimiter {
	if perMin < 1 {
		perMin = 10
	}
	return &RedisLimiter{rdb: rdb, perMin: perMin, keyPrefix: "ratelimit:auth:"}
}

// incrWithExpireScript: INCR key, set EXPIRE only when key is new (count==1).
const incrWithExpireScript = `local c = redis.call('INCR', KEYS[1]); if c == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end; return c`

// Allow returns true if the key is under the limit. On Redis errors, allows the request (fail open).
func (l *RedisLimiter) Allow(ctx context.Context, key string) (bool, error) {
	rkey := l.keyPrefix + key
	val, err := l.rdb.Eval(ctx, incrWithExpireScript, []string{rkey}, 60).Int()
	if err != nil {
		return true, fmt.Errorf("redis rate limit: %w", err)
	}
	return val <= l.perMin, nil
}
