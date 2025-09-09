package search

import (
	"fmt"
	"lesson2/adapters/inbound/httpapi/respond"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PathTenant struct {
	TenantID string `path:"tenant_id" json:"tenant_id"`
}

func BindPathTenant(r *http.Request) (PathTenant, []respond.FieldError, error) {
	id := chi.URLParam(r, "tenant_id")
	if _, err := uuid.Parse(id); err != nil {
		return PathTenant{}, []respond.FieldError{{Loc: "path.tenant_id", Msg: "must be uuid"}}, fmt.Errorf("invalid")
	}
	return PathTenant{TenantID: id}, nil, nil
}
