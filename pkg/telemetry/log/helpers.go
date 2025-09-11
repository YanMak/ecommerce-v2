package log

import "context"

// Info — удобный вызов с контекста.
func Info(ctx context.Context, msg string, kv ...any) {
	l := FromContext(ctx)
	l.Sugar().Infow(msg, kv...)
}

// Error — добавляет err в поля и пишет ошибку.
func Error(ctx context.Context, err error, msg string, kv ...any) {
	l := FromContext(ctx)
	kv = append(kv, "error", err)
	l.Sugar().Errorw(msg, kv...)
}
