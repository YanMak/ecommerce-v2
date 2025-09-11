package grpcx

import (
	"context"

	tctx "github.com/YanMak/ecommerce/v2/pkg/telemetry/ctx"
	tlog "github.com/YanMak/ecommerce/v2/pkg/telemetry/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func UnaryServerZapLogger(base *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		l := base.With(
			zap.String("rpc", info.FullMethod),
			zap.String("request_id", tctx.RequestID(ctx)),
			zap.String("idempotency_key", tctx.IdempotencyKey(ctx)),
		)
		ctx = tlog.IntoContext(ctx, l) // ← кладём request-logger
		return handler(ctx, req)
	}
}
