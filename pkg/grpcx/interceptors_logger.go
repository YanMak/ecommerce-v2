package grpcx

import (
	"context"
	"fmt"

	tctx "github.com/YanMak/ecommerce/v2/pkg/telemetry/ctx"
	tlog "github.com/YanMak/ecommerce/v2/pkg/telemetry/log"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func UnaryServerZapLogger(base *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// l := base.With(
		// 	zap.String("rpc", info.FullMethod),
		// 	zap.String("request_id", tctx.RequestID(ctx)),
		// 	zap.String("idempotency_key", tctx.IdempotencyKey(ctx)),
		// )
		// ctx = tlog.IntoContext(ctx, l) // ← кладём request-logger
		// return handler(ctx, req)

		// базовые поля
		fields := []zap.Field{
			zap.String("rpc", info.FullMethod),
			zap.String("request_id", tctx.RequestID(ctx)),
			zap.String("idempotency_key", tctx.IdempotencyKey(ctx)),
		}
		// добавим trace/span если в контексте есть активный спан OTel
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			fields = append(fields,
				zap.String("trace_id", sc.TraceID().String()),
				zap.String("span_id", sc.SpanID().String()),
			)
			fmt.Printf("trace_id=%s , span_id=%s \n", sc.TraceID().String(), sc.SpanID().String())
		}

		l := base.With(fields...)
		ctx = tlog.IntoContext(ctx, l) // ← кладём request-logger

		return handler(ctx, req)
	}
}
