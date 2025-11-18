package telemetryhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/YanMak/ecommerce/v2/pkg/httpx/bind"
	"github.com/YanMak/ecommerce/v2/pkg/httpx/render"
	"github.com/YanMak/ecommerce/v2/services/api-gateway-simple/internal/adapters/inbound/httpapi/dto"
	"github.com/YanMak/ecommerce/v2/services/api-gateway-simple/internal/app/usecase"
	"go.uber.org/zap"
)

func Telemetry(uc *usecase.CertificatesUC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// w.Header().Set("Content-Type", "application/json")
		// _ = json.NewEncoder(w).Encode(time.Now())

		d, _ := bind.DTO[dto.CRMSearchDTO](r) // валидный DTO уже в контексте

		// Е2Е дедлайн на запрос (HTTP → gRPC → CRM → БД)
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second) // можно вынести в ENV
		defer cancel()

		if dl, ok := ctx.Deadline(); ok {
			fmt.Println("gw.grpc.deadline", zap.Duration("left", time.Until(dl)))
		}

		resp, err := uc.Search(ctx, d.ToParams())
		if err != nil {
			render.GRPCError(w, r, err)
			return
		}
		_ = json.NewEncoder(w).Encode(resp)

	}
}

func Search(uc *usecase.CertificatesUC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d, _ := bind.DTO[dto.CRMSearchDTO](r) // валидный DTO уже в контексте

		// Е2Е дедлайн на запрос (HTTP → gRPC → CRM → БД)
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second) // можно вынести в ENV
		defer cancel()

		if dl, ok := ctx.Deadline(); ok {
			fmt.Println("gw.grpc.deadline", zap.Duration("left", time.Until(dl)))
		}

		resp, err := uc.Search(ctx, d.ToParams())
		if err != nil {
			render.GRPCError(w, r, err)
			return
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}
