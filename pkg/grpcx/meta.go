package grpcx

import (
	"context"

	tctx "github.com/YanMak/ecommerce/v2/pkg/telemetry/ctx"
	"google.golang.org/grpc/metadata"
)

const (
	MDRequestID      = "x-request-id"
	MDIdempotencyKey = "idempotency-key"
)

// IncomingToContext переносит входящие gRPC метаданные в наш context.
// Используй в начале серверных хендлеров или серверного интерсептора.
func IncomingToContext(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	if vals := md.Get(MDRequestID); len(vals) > 0 {
		ctx = tctx.WithRequestID(ctx, vals[0])
	}
	if vals := md.Get(MDIdempotencyKey); len(vals) > 0 {
		ctx = tctx.WithIdempotencyKey(ctx, vals[0])
	}
	return ctx
}

// ContextToOutgoing добавляет значения из нашего ctx в исходящий gRPC metadata.
// Используй в клиенте (HTTP-шлюз) перед вызовом RPC.
func ContextToOutgoing(ctx context.Context) context.Context {
	md := metadata.MD{}
	if id := tctx.RequestID(ctx); id != "" {
		md.Append(MDRequestID, id)
	}
	if k := tctx.IdempotencyKey(ctx); k != "" {
		md.Append(MDIdempotencyKey, k)
	}
	if len(md) == 0 {
		return ctx
	}
	return metadata.NewOutgoingContext(ctx, md)
}
