package metrics

import (
	"context"
	"time"
)

type Metrics interface {
	RetryAttempt(ctx context.Context, op string, attempt int, err error)
	ObserveLatency(ctx context.Context, op string, d time.Duration, ok bool)
}

type Noop struct{}

func (Noop) RetryAttempt(context.Context, string, int, error)            {}
func (Noop) ObserveLatency(context.Context, string, time.Duration, bool) {}

var M Metrics = Noop{} // переопредели в main() на Prom/Otel
