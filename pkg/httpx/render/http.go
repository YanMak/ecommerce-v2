package render

import (
	"encoding/json"
	"net/http"

	tctx "github.com/YanMak/ecommerce/v2/pkg/telemetry/ctx"
)

// Validation — единый рендер 400 с полями, совместимыми по форме с gRPC-деталями.
func Validation(w http.ResponseWriter, r *http.Request, code, message string, v []Violation) {
	body := ErrorBody{
		Code:           code,
		Message:        message,
		RequestID:      tctx.RequestID(r.Context()),
		IdempotencyKey: tctx.IdempotencyKey(r.Context()),
		Violations:     v,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(body)
}
