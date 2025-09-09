package respond

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

type FieldError struct {
	Loc string `json:"loc"` // "body.field", "query.limit", "path.tenant_id", "header.Authorization", "cookie.session_id"
	Msg string `json:"msg"`
}

func JSON(w http.ResponseWriter, r *http.Request, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	if rid := middleware.GetReqID(r.Context()); rid != "" {
		w.Header().Set("X-Request-ID", rid)
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func ValidationError(w http.ResponseWriter, r *http.Request, fields []FieldError) {
	w.Header().Set("Content-Type", "application/json")
	if rid := middleware.GetReqID(r.Context()); rid != "" {
		w.Header().Set("X-Request-ID", rid)
	}
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":   "invalid",
		"error":  "validation failed",
		"errors": fields,
	})
}

// AppError — универсальная запись бизнес-ошибки (Problem-style без errors[]).
func AppError(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/json")
	if rid := middleware.GetReqID(r.Context()); rid != "" {
		w.Header().Set("X-Request-ID", rid)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":  code,
		"error": detail,
	})
}
