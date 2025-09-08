package ctx

import "context"

type key string

const (
	reqIDKey key = "request-id"
	idemKey  key = "idempotency-key"
)

func WithRequestID(parent context.Context, id string) context.Context {
	return context.WithValue(parent, reqIDKey, id)
}
func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(reqIDKey).(string); ok {
		return v
	}
	return ""
}

func WithIdempotencyKey(parent context.Context, key string) context.Context {
	return context.WithValue(parent, idemKey, key)
}
func IdempotencyKey(ctx context.Context) string {
	if v, ok := ctx.Value(idemKey).(string); ok {
		return v
	}
	return ""
}
