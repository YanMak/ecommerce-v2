package log

import (
	"context"

	"go.uber.org/zap"
)

type ctxKey struct{}

func IntoContext(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}
func FromContext(ctx context.Context) *zap.Logger {
	if v, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok && v != nil {
		return v
	}
	return zap.NewNop()
}
