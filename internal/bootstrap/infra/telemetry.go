package infra

import (
	"context"

	"github.com/terrynullson/mntrng/internal/telemetry"
)

func InitTelemetry(ctx context.Context) error {
	return telemetry.InitTracer(ctx)
}

func ShutdownTelemetry(ctx context.Context) error {
	return telemetry.Shutdown(ctx)
}
