package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YanMak/ecommerce/v2/pkg/errkit"
)

func Test_Retry_SucceedsOnSecondAttempt(t *testing.T) {
	var calls int
	var hookCalls int

	fn := func(context.Context) error {
		calls++
		if calls == 1 {
			return wrap(errkit.ErrTransient) // имитируем временную ошибку
		}
		return nil
	}
	err := Do(
		context.Background(),
		fn,
		Only(errkit.IsTransient),
		WithMaxAttempts(3),
		WithBaseDelay(10*time.Millisecond),
		WithMaxDelay(20*time.Millisecond),
		WithOnAttempt(func(ctx context.Context, attempt int, err error) {
			hookCalls++
		}),
	)
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("want 2 calls, got %d", calls)
	}
	if hookCalls == 0 {
		t.Fatalf("want 1 hookCalls, got %d", hookCalls)
	}
}

func Test_Retry_NoRetryOnNonRetryable(t *testing.T) {
	var calls int
	var hookCalls int

	fn := func(context.Context) error {
		calls++
		return errors.New("boom")
	}
	err := Do(
		context.Background(),
		fn,
		Only(errkit.IsTransient),
		WithOnAttempt(func(ctx context.Context, attempt int, err error) {
			hookCalls++
		}),
	)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if calls != 1 {
		t.Fatalf("want 1 call, got %d", calls)
	}
	if hookCalls != 0 {
		t.Fatalf("want 0 hookCalls, got %d", hookCalls)
	}
}

func Test_Retry_RespectsContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	fn := func(context.Context) error {
		return wrap(errkit.ErrTransient)
	}
	err := Do(ctx, fn, Only(errkit.IsTransient),
		WithMaxAttempts(10), WithBaseDelay(50*time.Millisecond), WithMaxDelay(100*time.Millisecond))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline, got %v", err)
	}
}

func wrap(sentinel error) error { return errors.Join(sentinel, errors.New("underlying")) }
