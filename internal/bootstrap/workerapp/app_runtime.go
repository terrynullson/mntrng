package workerapp

import (
	"context"
	"database/sql"

	"github.com/terrynullson/mntrng/internal/bootstrap/infra"
	workerservice "github.com/terrynullson/mntrng/internal/service/worker"
)

type RuntimeApp struct {
	db           *sql.DB
	app          *workerservice.App
	metricsPort  int
	metricsToken string
}

func NewRuntimeApp(cfg RuntimeConfig) (*RuntimeApp, error) {
	db, err := infra.OpenPostgres(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	infra.ConfigureDBPoolFromEnv(db, infra.DBPoolDefaults{
		MaxOpenConns:       20,
		MaxIdleConns:       10,
		ConnMaxLifetimeMin: 30,
		ConnMaxIdleTimeMin: 10,
	})
	if err := infra.PingDB(context.Background(), db, cfg.DBPingTimeout); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &RuntimeApp{
		db:           db,
		app:          NewApp(db, cfg),
		metricsPort:  cfg.MetricsPort,
		metricsToken: cfg.MetricsToken,
	}, nil
}

func (a *RuntimeApp) Run(ctx context.Context) {
	StartMetricsServer(ctx, a.metricsPort, a.metricsToken)
	a.app.Run(ctx)
}

func (a *RuntimeApp) Close() error {
	return a.db.Close()
}
