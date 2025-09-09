package search

import (
	"lesson2/adapters/inbound/httpapi/apperr"
	"lesson2/adapters/inbound/httpapi/bind"
	"lesson2/adapters/inbound/httpapi/respond"
	appsearch "lesson2/app/search"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// SearchEchoResponse — что вернём на 200 для демонстрации всех входов.
type SearchEchoResponse struct {
	Path    PathTenant    `json:"path"`
	Headers Headers       `json:"headers"`
	Cookie  CookieSession `json:"cookie"`
	Query   SearchQuery   `json:"query"`
	Body    SearchBody    `json:"body"`
}

type Controller struct {
	UC *appsearch.UseCase
}

func New(uc *appsearch.UseCase) *Controller { return &Controller{UC: uc} }

func (c *Controller) Routes(r chi.Router) {
	r.With(
		bind.WithDTO(BindPathTenant),
		bind.WithDTO(BindHeaders),
		bind.WithDTO(BindCookieSession),
		bind.WithDTO(BindSearchQuery),
		bind.WithDTO(BindSearchBody),
	).Post("/tenants/{tenant_id}/catalog:search", c.search)
}

// @Summary      Search catalog (training super-endpoint)
// @Description  Один эндпоинт, показывающий парсинг и валидацию path/query/header/cookie/body
// @Tags         search
// @Param        tenant_id        path   string   true  "Tenant ID (uuid)"                    format(uuid)
// // @Param        session_id       cookie string   true  "Session ID (uuid)"                   format(uuid)
// @Param        Cookie  header  string  true  "Cookie header: session_id=<uuid>"  format(uuid)
// @Param        limit            query  int      false "Limit [1..100]"                      minimum(1) maximum(100) default(20)
// @Param        offset           query  int      false "Offset >= 0"                         minimum(0)  default(0)
// @Param        sort             query  []string false "CSV sort (e.g. -created_at,name)"    collectionFormat(csv)
// @Param        created_from     query  string   false "From (RFC3339)"                      format(date-time)
// @Param        created_to       query  string   false "To (RFC3339)"                        format(date-time)
// @Param        include_archived query  bool     false "Include archived"
// @Param        Authorization    header string   true  "Bearer token"
// @Param        X-Client-Version header string   false "Client version (semver)"
// @Param        X-Locale         header string   false "Locale"                              Enums(en,ru)
// @Accept       json
// @Produce      json
// @Param        body             body   search.SearchBody true "Filters body"
// @Success      200  {object}   search.SearchEchoResponse
// @Failure      400  {object}   respond.Problem
// @Router       /v1/tenants/{tenant_id}/catalog:search [post]
func (c *Controller) search(w http.ResponseWriter, r *http.Request) {
	p, _ := bind.DTO[PathTenant](r)
	h, _ := bind.DTO[Headers](r)
	ck, _ := bind.DTO[CookieSession](r)
	q, _ := bind.DTO[SearchQuery](r)
	b, _ := bind.DTO[SearchBody](r)

	// Map DTO -> App Query
	var aq appsearch.Query
	aq.TenantID = p.TenantID
	aq.SessionID = ck.SessionID
	aq.AuthBearer = h.AuthBearer
	aq.Locale = h.Locale
	aq.Limit, aq.Offset = q.Limit, q.Offset
	aq.Sort = q.Sort
	aq.CreatedFrom, aq.CreatedTo = q.CreatedFrom, q.CreatedTo
	aq.IncludeArchived = q.IncludeArchived
	aq.Body.Q = b.Q
	aq.Body.CategoryIDs = b.CategoryIDs
	aq.Body.PriceMin, aq.Body.PriceMax = b.Price.Min, b.Price.Max
	aq.Body.Tags = b.Tags

	// Call use-case (бизнес-валидирует; может вернуть app.Error)
	if err := c.UC.Handle(r.Context(), aq); err != nil {
		apperr.Write(w, r, err)
		return
	}

	// Успех — пока оставляем учебный эхо-ответ (контракт не меняем)
	respond.JSON(w, r, http.StatusOK, map[string]any{
		"path": p, "headers": h, "cookie": ck, "query": q, "body": b,
	})
}
