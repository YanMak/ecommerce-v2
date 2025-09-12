package middleware

import (
	"net"
	"net/http"
	"time"

	tctx "github.com/YanMak/ecommerce/v2/pkg/telemetry/ctx"
	tlog "github.com/YanMak/ecommerce/v2/pkg/telemetry/log"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func WithZapLogger(base *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rc := NewCapture(w)

			start := time.Now()

			route := "" // шаблон маршрута chi (чтобы не плодить кардинальность)
			if rcx := chi.RouteContext(r.Context()); rcx != nil {
				route = rcx.RoutePattern()
			}
			if route == "" {
				route = r.URL.Path // запасной вариант
			}

			reqID := tctx.RequestID(r.Context())
			idem := tctx.IdempotencyKey(r.Context())
			method := r.Method
			ua := r.UserAgent()
			remote := clientIP(r)

			reqLog := base.With(
				zap.String("request_id", reqID),
				zap.String("idempotency_key", idem),
				zap.String("route", route),
				zap.String("method", method),
				zap.String("remote_ip", remote),
				zap.String("user_agent", ua),
			)
			// положим request-логгер в контекст для нижних слоёв
			ctx := tlog.IntoContext(r.Context(), reqLog)

			next.ServeHTTP(rc, r.WithContext(ctx))

			status := rc.Status
			if status == 0 {
				status = http.StatusOK
			}

			if route == "/metrics" {
				return
			}

			durMs := time.Since(start).Seconds() * 1000
			reqLog.Info("http_request",
				zap.Int("status", status),
				zap.Float64("duration_ms", durMs),
				zap.Int("bytes", rc.Bytes),
			)

			// опционально: промполя для кореляции с метриками
			_ = reqLog
		})
	}
}

func clientIP(r *http.Request) string {
	// очень простой вариант; при наличии прокси можно смотреть X-Forwarded-For
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
