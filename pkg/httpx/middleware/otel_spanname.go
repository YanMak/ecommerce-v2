package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
)

// SpanNameFromChiRoute переименовывает текущий HTTP-спан в "METHOD <route-pattern>"
// Например: "GET /crm/certificates/search". Если паттерн недоступен — берём фактический URL.Path.
func SpanNameFromChiRoute() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span := trace.SpanFromContext(r.Context())
			if span != nil {
				route := ""
				if rc := chi.RouteContext(r.Context()); rc != nil {
					route = rc.RoutePattern()
				}
				if route == "" {
					route = r.URL.Path
				}
				span.SetName(r.Method + " " + route)
			}
			next.ServeHTTP(w, r)
		})
	}
}
