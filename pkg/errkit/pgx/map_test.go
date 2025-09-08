package pgx

import (
	"context"
	"errors"
	"testing"

	"github.com/YanMak/ecommerce/v2/pkg/errkit"
	"github.com/jackc/pgx/v5/pgconn"
)

func Test_Map_PgCodes(t *testing.T) {
	cases := []struct {
		code string
		want error
	}{
		{"40001", errkit.ErrTransient},
		{"40P01", errkit.ErrTransient},
		{"55P03", errkit.ErrTransient},
		{"08006", errkit.ErrTransient},
		{"23505", errkit.ErrConflict},
		{"23502", errkit.ErrInvalid},
	}
	for _, c := range cases {
		err := Map(&pgconn.PgError{Code: c.code})
		if !errors.Is(err, c.want) {
			t.Fatalf("code %s: want %v, got %v", c.code, c.want, err)
		}
	}
}

func Test_Map_Context(t *testing.T) {
	if !errors.Is(Map(context.DeadlineExceeded), errkit.ErrDeadline) {
		t.Fatal("deadline not mapped")
	}
	if !errors.Is(Map(context.Canceled), errkit.ErrCanceled) {
		t.Fatal("canceled not mapped")
	}
}
