package otelx

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// MetricsShutdown — тип для graceful-останова провайдера метрик.
type MetricsShutdown func(ctx context.Context) error

// InitMetrics настраивает экспорт OTel-метрик по OTLP/gRPC в Collector.
// Вызывает otel.SetMeterProvider(...) и возвращает shutdown-функцию.
func InitMetrics(ctx context.Context, serviceName string) (MetricsShutdown, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317" // твой otel-collector из compose
	}

	// Экспортёр метрик OTLP/gRPC (без блокирующего dial — коннектится в фоне).
	exp, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(), // dev; в проде заменишь на TLS
	)
	if err != nil {
		return nil, err
	}

	// Общие атрибуты ресурса (видны в лейблах и фильтрах).
	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironment(os.Getenv("DEPLOY_ENV")), // dev/stage/prod
			// при желании добавь: semconv.ServiceVersion(os.Getenv("SERVICE_VERSION")),
		),
	)

	// Периодический экспорт (по умолчанию раз в 1м; сократим до 5с для dev).
	reader := metric.NewPeriodicReader(exp, metric.WithInterval(5*time.Second))

	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(res),
	)

	otel.SetMeterProvider(mp)
	return mp.Shutdown, nil
}
