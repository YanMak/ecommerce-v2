package dto

import (
	"net/http"
	"net/url"
	"time"

	"github.com/YanMak/ecommerce/v2/pkg/httpx/query"
	"github.com/YanMak/ecommerce/v2/pkg/httpx/render"
	"github.com/YanMak/ecommerce/v2/services/api-gateway/internal/app/usecase"
)

func BindCRMSearchQuery(r *http.Request) (CRMSearchDTO, []render.Violation) {
	d := FromQuery(r.URL.Query())
	return d, d.Validate()
}

const maxPerPage = 200

type CRMSearchDTO struct {
	Q, Inn                 *string
	CreatedFrom, CreatedTo *time.Time
	UpdatedFrom, UpdatedTo *time.Time
	CategoryID             *int64
	Opened                 *bool
	Page, PerPage          int
}

// FromQuery — парсим RFC 3339 даты и числа с дефолтами.
func FromQuery(q url.Values) CRMSearchDTO {
	return CRMSearchDTO{
		Q:           query.StringPtr(q, "q"),
		Inn:         query.StringPtr(q, "inn"),
		CreatedFrom: query.TimePtrRFC3339(q, "created_from"),
		CreatedTo:   query.TimePtrRFC3339(q, "created_to"),
		UpdatedFrom: query.TimePtrRFC3339(q, "updated_from"),
		UpdatedTo:   query.TimePtrRFC3339(q, "updated_to"),
		CategoryID:  query.Int64Ptr(q, "category_id"),
		Opened:      query.BoolPtr(q, "opened"),
		Page:        query.IntDefault(q, "page", 1),
		PerPage:     query.IntDefault(q, "per_page", 10),
	}
}

// Validate — возвращает список нарушений; если пусто — всё ок.
func (d CRMSearchDTO) Validate() []render.Violation {
	var v []render.Violation
	if d.Page <= 0 {
		v = append(v, render.Violation{Field: "page", Reason: "must be >= 1"})
	}
	if d.PerPage <= 0 || d.PerPage > maxPerPage {
		v = append(v, render.Violation{Field: "per_page", Reason: "must be in [1..200]"})
	}
	if d.CreatedFrom != nil && d.CreatedTo != nil && d.CreatedFrom.After(*d.CreatedTo) {
		v = append(v, render.Violation{Field: "created_from/created_to", Reason: "created_from must be <= created_to"})
	}
	if d.UpdatedFrom != nil && d.UpdatedTo != nil && d.UpdatedFrom.After(*d.UpdatedTo) {
		v = append(v, render.Violation{Field: "updated_from/updated_to", Reason: "updated_from must be <= updated_to"})
	}
	return v
}

// ToParams — адаптируем к usecase шлюза.
func (d CRMSearchDTO) ToParams() usecase.CertsSearchParams {
	return usecase.CertsSearchParams{
		Q: d.Q, Inn: d.Inn,
		CreatedFrom: d.CreatedFrom, CreatedTo: d.CreatedTo,
		UpdatedFrom: d.UpdatedFrom, UpdatedTo: d.UpdatedTo,
		CategoryID: d.CategoryID, Opened: d.Opened,
		Page: d.Page, PerPage: d.PerPage,
	}
}
