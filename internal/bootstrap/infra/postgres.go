package infra

import (
	"context"
	"database/sql"
	"time"

	"github.com/terrynullson/mntrng/internal/config"
)

type DBPoolDefaults struct {
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetimeMin int
	ConnMaxIdleTimeMin int
}

func OpenPostgres(databaseURL string) (*sql.DB, error) {
	return sql.Open("postgres", databaseURL)
}

func ConfigureDBPoolFromEnv(db *sql.DB, defaults DBPoolDefaults) {
	maxOpen := config.IntAtLeast(config.GetInt("DB_MAX_OPEN_CONNS", defaults.MaxOpenConns), 1)
	maxIdle := config.IntAtLeast(config.GetInt("DB_MAX_IDLE_CONNS", defaults.MaxIdleConns), 1)
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	connMaxLifetimeMin := config.IntAtLeast(config.GetInt("DB_CONN_MAX_LIFETIME_MIN", defaults.ConnMaxLifetimeMin), 1)
	connMaxIdleTimeMin := config.IntAtLeast(config.GetInt("DB_CONN_MAX_IDLE_TIME_MIN", defaults.ConnMaxIdleTimeMin), 1)

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(connMaxLifetimeMin) * time.Minute)
	db.SetConnMaxIdleTime(time.Duration(connMaxIdleTimeMin) * time.Minute)
}

func PingDB(ctx context.Context, db *sql.DB, timeout time.Duration) error {
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return db.PingContext(pingCtx)
}
