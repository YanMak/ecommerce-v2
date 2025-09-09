package middleware

import (
	"net/http"

	tctx "github.com/YanMak/ecommerce/v2/pkg/telemetry/ctx"
)

const HeaderIdempotencyKey = "Idempotency-Key"

// WithIdempotencyKey: кладёт ключ идемпотентности из заголовка в контекст (если есть).
func WithIdempotencyKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if k := r.Header.Get(HeaderIdempotencyKey); k != "" {
			ctx := tctx.WithIdempotencyKey(r.Context(), k)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
