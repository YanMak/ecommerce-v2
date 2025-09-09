// # лог/метрики/трейсинг, unary/server/client
package grpcx

import (
	"context"

	tctx "github.com/YanMak/ecommerce/v2/pkg/telemetry/ctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryClientMetaInterceptor
// Клиентский интерсептор: добавляет x-request-id и idempotency-key
// из нашего context в исходящий gRPC metadata (и НЕ затирает уже существующий).
func UnaryClientMetaInterceptor(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	// соберём/домерджим outgoing metadata
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()

	if id := tctx.RequestID(ctx); id != "" {
		md.Set(MDRequestID, id)
	}
	if k := tctx.IdempotencyKey(ctx); k != "" {
		md.Set(MDIdempotencyKey, k)
	}
	if len(md) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

// UnaryServerMetaInterceptor
// Серверный интерсептор: переносит входящий gRPC metadata в наш context,
// чтобы дальше usecase/репо видели request-id/idempotency-key через pkg/telemetry/ctx.
func UnaryServerMetaInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	ctx = IncomingToContext(ctx)
	return handler(ctx, req)
}

// (опционально) stream-версия для серверной стороны, если позже появятся streaming RPC:

type serverStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStreamWithContext) Context() context.Context { return s.ctx }

// StreamServerMetaInterceptor переносит метаданные в контекст для streaming RPC.
func StreamServerMetaInterceptor(
	srv any,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	ctx := IncomingToContext(ss.Context())
	wrapped := &serverStreamWithContext{ServerStream: ss, ctx: ctx}
	return handler(srv, wrapped)
}
