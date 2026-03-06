package infra

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/terrynullson/mntrng/internal/ratelimit"
)

func NewAuthRateLimiter(redisAddr string, authPerMin int, redisPingTimeout time.Duration) ratelimit.Limiter {
	if redisAddr == "" {
		return ratelimit.NewInMemLimiter(authPerMin)
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	pingCtx, cancel := context.WithTimeout(context.Background(), redisPingTimeout)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		log.Printf("redis ping failed, using in-memory rate limiter: %v", err)
		return ratelimit.NewInMemLimiter(authPerMin)
	}
	return ratelimit.NewRedisLimiter(rdb, authPerMin)
}
