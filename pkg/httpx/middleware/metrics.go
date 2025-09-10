package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics/prom"
)

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

// WithMetrics — middleware для подсчета HTTP запросов и латентности.
func WithMetrics(c *prom.Collectors) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := r.URL.Path // если используешь chi — можно брать chi.RouteContext(r.Context()).RoutePattern()
			method := r.Method

			sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(sw, r)
			sec := time.Since(start).Seconds()

			c.HTTPRequestsTotal.WithLabelValues(route, method, strconv.Itoa(sw.code)).Inc()
			c.HTTPRequestDuration.WithLabelValues(route, method).Observe(sec)
		})
	}
}
