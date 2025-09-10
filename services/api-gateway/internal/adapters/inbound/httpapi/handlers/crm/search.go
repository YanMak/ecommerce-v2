package crmhandlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/YanMak/ecommerce/v2/pkg/httpx/bind"
	"github.com/YanMak/ecommerce/v2/pkg/httpx/render"
	"github.com/YanMak/ecommerce/v2/services/api-gateway/internal/adapters/inbound/httpapi/dto"
	"github.com/YanMak/ecommerce/v2/services/api-gateway/internal/app/usecase"
)

func Healtz(uc *usecase.CertificatesUC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(time.Now())
	}
}

func Search(uc *usecase.CertificatesUC) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d, _ := bind.DTO[dto.CRMSearchDTO](r) // валидный DTO уже в контексте
		resp, err := uc.Search(r.Context(), d.ToParams())
		if err != nil {
			render.GRPCError(w, r, err)
			return
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}
