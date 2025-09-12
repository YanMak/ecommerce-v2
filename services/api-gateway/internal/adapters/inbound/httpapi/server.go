package httpapi

import (
	"net/http"

	"github.com/YanMak/ecommerce/v2/pkg/httpx/bind"
	crmhandlers "github.com/YanMak/ecommerce/v2/services/api-gateway/internal/adapters/inbound/httpapi/handlers/crm"
	"github.com/YanMak/ecommerce/v2/services/api-gateway/internal/app/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	gwdto "github.com/YanMak/ecommerce/v2/services/api-gateway/internal/adapters/inbound/httpapi/dto"

	prommetrics "github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics/prom"
)

type Server struct{ router chi.Router }

// @title       Architecture-1 API
// @version     1.0
// @description Training project: super-endpoint covering full HTTP inputs with validation.
// @BasePath    /v1
// @schemes     http
func NewServer(r *chi.Mux, logger *zap.Logger, reg *prometheus.Registry, cols *prommetrics.Collectors, uc *usecase.CertificatesUC) *Server {
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

	// chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
	// 	//zap.L().Info("route", zap.String("method", method), zap.String("pattern", route))
	// 	logger.Info("route", zap.String("method", method), zap.String("pattern", route))
	// 	return nil
	// })

	// тут позже смонтируем /v1 и контроллеры
	return &Server{router: r}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
