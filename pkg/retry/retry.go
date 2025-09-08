package retry

import (
	"context"
	"math/rand"
	"time"
)

type Predicate func(error) bool

// Only — синтаксический сахар: можно передавать просто errkit.IsTransient.
func Only(p Predicate) Predicate {
	return p
}

type AttemptHook func(ctx context.Context, attempt int, err error)

type config struct {
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
	onAttempt   AttemptHook
}

type Option func(*config)

func WithMaxAttempts(n int) Option         { return func(c *config) { c.maxAttempts = n } }
func WithBaseDelay(d time.Duration) Option { return func(c *config) { c.baseDelay = d } }
func WithMaxDelay(d time.Duration) Option  { return func(c *config) { c.maxDelay = d } }
func WithOnAttempt(h AttemptHook) Option   { return func(c *config) { c.onAttempt = h } }

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// Do вызывает fn и повторяет её при ошибке, если predicate(err)==true.
// Экспоненциальный backoff + полный джиттер. Уважает ctx.
func Do(ctx context.Context, fn func(context.Context) error, predicate Predicate, opts ...Option) error {
	cfg := config{
		maxAttempts: 4,
		baseDelay:   100 * time.Millisecond,
		maxDelay:    2 * time.Second,
	}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.maxAttempts < 1 {
		cfg.maxAttempts = 1
	}

	var attempt int
	for {
		attempt++
		err := fn(ctx)
		if err == nil {
			return nil
		}
		// если контекст уже умер — возвращаем его ошибку
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// не retryable — выходим сразу
		if predicate == nil || !predicate(err) {
			return err
		}
		// достигли лимита попыток — отдаём последнюю ошибку
		if attempt >= cfg.maxAttempts {
			return err
		}
		// backoff = min(base * 2^(attempt-1), maxDelay)
		backoff := cfg.baseDelay << (attempt - 1)
		if backoff > cfg.maxDelay {
			backoff = cfg.maxDelay
		}
		// полный джиттер: [0 .. backoff]
		jitterNs := rng.Int63n(backoff.Nanoseconds() + 1)
		delay := time.Duration(jitterNs)

		// спим, уважая ctx
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
