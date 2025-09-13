package otelx

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// InitTracer настраивает экспорт трейсов в OTLP/gRPC (Jaeger all-in-one) и
// регистрирует глобальные провайдер и пропагатор.
// Возвращает shutdown-функцию: вызови её на graceful-stop, чтобы слить буфер.
func InitTracer(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317" // Jaeger all-in-one OTLP gRPC
	}

	// Экспортёр: отправляет спаны по OTLP/gRPC (без TLS — dev).
	exp, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	// Ресурс: атрибуты процесса (видно в UI и фильтрах).
	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			attribute.String("deployment.environment", os.Getenv("DEPLOY_ENV")),
		),
	)

	// Провайдер трассировки: 100% семплинг в dev, батч-отправка.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(
			sdktrace.ParentBased(sdktrace.TraceIDRatioBased(1.0)),
		),
		sdktrace.WithBatcher(
			exp,
			sdktrace.WithBatchTimeout(3*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
	)

	// Глобальные настройки для всего процесса (их увидят интерсепторы gRPC).
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, // W3C traceparent: несёт trace_id
			propagation.Baggage{},      // опциональные ключ-значения «в нагрузку»
		),
	)

	return tp.Shutdown, nil
}
