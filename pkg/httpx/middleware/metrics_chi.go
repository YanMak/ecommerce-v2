package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics/prom"
	"github.com/go-chi/chi/v5"
)

// WithMetricsChi — считает HTTP-запросы и время обработки для chi-роутера.
func WithMetricsChi(c *prom.Collectors) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
			rc := NewCapture(w) // делаем обёртку

			start := time.Now()

			next.ServeHTTP(rc, r)

			// шаблон маршрута (надежнее брать ПОСЛЕ next — к этому моменту он уже финализирован)
			route := r.URL.Path
			if rc := chi.RouteContext(r.Context()); rc != nil {
				if pat := rc.RoutePattern(); pat != "" {
					route = pat
				}
			}

			if route == "/metrics" {
				return
			}

			status := rc.Status
			if status == 0 {
				status = http.StatusOK
			} // если WriteHeader/Write не вызывали

			method := r.Method
			latency := time.Since(start).Seconds()

			c.HTTPRequestsTotal.WithLabelValues(route, method, strconv.Itoa(status)).Inc()
			c.HTTPRequestDuration.WithLabelValues(route, method).Observe(latency)
		})
	}
}
