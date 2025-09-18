package httpmetricschi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Middleware инкапсулирует инструменты метрик для HTTP-сервера на chi.
type Middleware struct {
	reqCount    metric.Int64Counter
	reqDuration metric.Float64Histogram
}

// New создаёт инстанс мидлвари. scope — имя instrumentation scope,
// обычно импортный путь пакета, например: "your/module/pkg/otelx/httpmetricschi".
func New(scope string) (*Middleware, error) {
	m := otel.GetMeterProvider().Meter(scope)

	count, err := m.Int64Counter("http.server.request.count")
	if err != nil {
		return nil, err
	}
	dur, err := m.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	return &Middleware{
		reqCount:    count,
		reqDuration: dur,
	}, nil
}

// Handler — chi-совместимая мидлварь: r.Use(mw.Handler).
// Лейблы: http.method, http.route (шаблон), http.status_code.
func (mw *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(ww, r)

		route := ""
		if rc := chi.RouteContext(r.Context()); rc != nil {
			route = rc.RoutePattern() // даёт именно шаблон, низкая кардинальность
		}
		if route == "" {
			route = "unknown"
		}

		attrs := []attribute.KeyValue{
			attribute.String("http.method", r.Method),
			attribute.String("http.route", route),
			attribute.String("http.status_code", strconv.Itoa(ww.status)),
		}

		ctx := r.Context()
		mw.reqCount.Add(ctx, 1, metric.WithAttributes(attrs...))
		mw.reqDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(attrs...))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
