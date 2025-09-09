package search

import (
	"fmt"
	"lesson2/adapters/inbound/httpapi/respond"
	"net/http"
	"strings"
)

type Headers struct {
	AuthBearer string `header:"Authorization" json:"Authorization"` // Bearer token
	ClientVer  string `header:"X-Client-Version" json:"X-Client-Version"`
	Locale     string `header:"X-Locale" json:"X-Locale"` // en|ru
}

func BindHeaders(r *http.Request) (Headers, []respond.FieldError, error) {
	h := Headers{
		AuthBearer: strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "),
		ClientVer:  r.Header.Get("X-Client-Version"),
		Locale:     r.Header.Get("X-Locale"),
	}
	if h.AuthBearer == "" {
		return h, []respond.FieldError{{Loc: "header.Authorization", Msg: "missing Bearer token"}}, fmt.Errorf("invalid")
	}
	if h.Locale != "" && h.Locale != "en" && h.Locale != "ru" {
		return h, []respond.FieldError{{Loc: "header.X-Locale", Msg: "must be en|ru"}}, fmt.Errorf("invalid")
	}
	return h, nil, nil
}
