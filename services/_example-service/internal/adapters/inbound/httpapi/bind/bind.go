package bind

import (
	"context"
	"lesson2/adapters/inbound/httpapi/respond"
	"net/http"
)

type ctxKey[T any] struct{}

type BindFn[T any] func(r *http.Request) (T, []respond.FieldError, error)

func WithDTO[T any](fn BindFn[T]) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dto, fields, err := fn(r)
			if err != nil {
				respond.ValidationError(w, r, fields)
				return
			}
			ctx := context.WithValue(r.Context(), ctxKey[T]{}, dto)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func DTO[T any](r *http.Request) (T, bool) {
	v := r.Context().Value(ctxKey[T]{})
	if v == nil {
		var zero T
		return zero, false
	}
	d, _ := v.(T)
	return d, true
}
