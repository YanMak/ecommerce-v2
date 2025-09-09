package search

import (
	"fmt"
	"lesson2/adapters/inbound/httpapi/respond"
	"net/http"

	"github.com/google/uuid"
)

type CookieSession struct {
	SessionID string `cookie:"session_id" json:"session_id"`
}

func BindCookieSession(r *http.Request) (CookieSession, []respond.FieldError, error) {
	c, err := r.Cookie("session_id")
	if err != nil {
		return CookieSession{}, []respond.FieldError{{Loc: "cookie.session_id", Msg: "is required"}}, fmt.Errorf("invalid")
	}
	if _, err := uuid.Parse(c.Value); err != nil {
		return CookieSession{}, []respond.FieldError{{Loc: "cookie.session_id", Msg: "must be uuid"}}, fmt.Errorf("invalid")
	}
	return CookieSession{SessionID: c.Value}, nil, nil
}
