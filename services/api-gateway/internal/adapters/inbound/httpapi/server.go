package httpapi

import (
	"net/http"

	"github.com/YanMak/ecommerce/v2/pkg/httpx/bind"
	crmhandlers "github.com/YanMak/ecommerce/v2/services/api-gateway/internal/adapters/inbound/httpapi/handlers/crm"
	"github.com/YanMak/ecommerce/v2/services/api-gateway/internal/app/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	httpmdw "github.com/YanMak/ecommerce/v2/pkg/httpx/middleware"

	gwdto "github.com/YanMak/ecommerce/v2/services/api-gateway/internal/adapters/inbound/httpapi/dto"

	prommetrics "github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics/prom"
)

type Server struct{ router chi.Router }

// @title       Architecture-1 API
// @version     1.0
// @description Training project: super-endpoint covering full HTTP inputs with validation.
// @BasePath    /v1
// @schemes     http
func NewServer(logger *zap.Logger, reg *prometheus.Registry, cols *prommetrics.Collectors, uc *usecase.CertificatesUC) *Server {
	r := chi.NewRouter()

	r.Use(
		//middleware.RequestID,
		middleware.Recoverer,
		//middleware.Logger,
		middleware.Compress(5),
	)

	// кросс-срезовые
	r.Use(httpmdw.WithRequestID)
	r.Use(httpmdw.WithIdempotencyKey)

	r.Use(httpmdw.WithZapLogger(logger))
	r.Use(httpmdw.WithMetricsChi(cols))

	//r.Method("GET", "/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	r.Method("GET", "/metrics_", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// маршрут: биндеры → тонкий хендлер
	r.With(
		bind.WithDTO(gwdto.BindCRMSearchQuery), // query → DTO + Validate()
		// bind.WithDTO(BindHeaders), bind.WithDTO(BindCookies), bind.WithDTO(BindPath) — добавим по мере надобности
	).Get("/crm/certificates/search", crmhandlers.Search(uc))

	// маршрут: биндеры → тонкий хендлер
	r.With(
		bind.WithDTO(gwdto.BindCRMSearchQuery), // query → DTO + Validate()
		// bind.WithDTO(BindHeaders), bind.WithDTO(BindCookies), bind.WithDTO(BindPath) — добавим по мере надобности
	).Get("/crm/certificates/search_test", crmhandlers.Search(uc))

	// тут позже смонтируем /v1 и контроллеры
	return &Server{router: r}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
