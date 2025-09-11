package logger

import "context"

type Logger interface {
	Info(ctx context.Context, msg string, kv ...any)
	Error(ctx context.Context, err error, msg string, kv ...any)
}

// Noop — заглушка по умолчанию (ничего не пишет).
type Noop struct{}

func (Noop) Info(context.Context, string, ...any)         {}
func (Noop) Error(context.Context, error, string, ...any) {}

var L Logger = Noop{} // можно переопределить в main() на zap/logrus/otelzap
