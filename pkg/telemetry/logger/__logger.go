package logger

import (
	"context"

	tlog "github.com/YanMak/ecommerce/v2/pkg/telemetry/log"
	"go.uber.org/zap"
)

// Современный прод-конструктор (реэкспорт из pkg/telemetry/log)
func NewProduction() (*zap.Logger, error) { return tlog.NewProduction() }

// Достаём/кладём request-scoped логгер через общий код в pkg/telemetry/log
func IntoContext(ctx context.Context, l *zap.Logger) context.Context { return tlog.IntoContext(ctx, l) }
func FromContext(ctx context.Context) *zap.Logger                    { return tlog.FromContext(ctx) }
