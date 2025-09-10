package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics/prom"
	"github.com/go-chi/chi/v5"
)

// обёртка, чтобы перехватить код ответа
type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

// WithMetricsChi — считает HTTP-запросы и время обработки для chi-роутера.
func WithMetricsChi(c *prom.Collectors) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
			start := time.Now()

			next.ServeHTTP(sw, r)

			// шаблон маршрута (надежнее брать ПОСЛЕ next — к этому моменту он уже финализирован)
			route := r.URL.Path
			if rc := chi.RouteContext(r.Context()); rc != nil {
				if pat := rc.RoutePattern(); pat != "" {
					route = pat
				}
			}

			method := r.Method
			sec := time.Since(start).Seconds()

			c.HTTPRequestsTotal.WithLabelValues(route, method, strconv.Itoa(sw.code)).Inc()
			c.HTTPRequestDuration.WithLabelValues(route, method).Observe(sec)
		})
	}
}
