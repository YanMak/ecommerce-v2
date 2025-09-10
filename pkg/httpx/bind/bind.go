package bind

import (
	"context"
	"net/http"

	"github.com/YanMak/ecommerce/v2/pkg/httpx/render"
)

type Binder[T any] func(r *http.Request) (T, []render.Violation)

type ctxKey[T any] struct{}

// WithDTO: вызывает binder; при ошибках — 400 и стоп; иначе кладёт DTO в контекст.
func WithDTO[T any](fn Binder[T]) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dto, viol := fn(r)
			if len(viol) > 0 {
				render.Validation(w, r, "INVALID", "invalid request", viol)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKey[T]{}, dto)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// DTO: достать ранее привязанный DTO из контекста.
func DTO[T any](r *http.Request) (T, bool) {
	v, ok := r.Context().Value(ctxKey[T]{}).(T)
	return v, ok
}
