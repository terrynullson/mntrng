package telemetry

import (
	"context"
	"os"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var (
	tp   *sdktrace.TracerProvider
	once sync.Once
)

// InitTracer initializes the global tracer provider from environment.
// If OTEL_EXPORTER_OTLP_ENDPOINT is set, uses OTLP HTTP exporter; otherwise no-op (no tracer).
func InitTracer(ctx context.Context) error {
	var err error
	once.Do(func() {
		endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if endpoint == "" {
			return
		}
		exporter, e := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithInsecure(),
		)
		if e != nil {
			err = e
			return
		}
		res, e := resource.Merge(
			resource.Default(),
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName("hls-monitoring-api"),
			),
		)
		if e != nil {
			err = e
			return
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
	})
	return err
}

// Shutdown flushes and shuts down the tracer provider if it was initialized.
func Shutdown(ctx context.Context) error {
	if tp != nil {
		return tp.Shutdown(ctx)
	}
	return nil
}

// Enabled returns true if tracing was initialized (OTLP endpoint was set).
func Enabled() bool {
	return tp != nil
}
