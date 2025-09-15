package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// InFlight ограничивает одновременные запросы. При переполнении — 503 + Retry-After: 1.
func InFlight(limit int) func(http.Handler) http.Handler {
	if limit <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	sem := make(chan struct{}, limit)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				next.ServeHTTP(w, r)
			default:
				if span := trace.SpanFromContext(r.Context()); span != nil {
					span.SetAttributes(attribute.Bool("throttled", true))
				}
				w.Header().Set("Retry-After", "1")
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			}
		})
	}
}
