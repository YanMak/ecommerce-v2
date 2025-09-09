package apperr

import (
	"lesson2/adapters/inbound/httpapi/respond"
	"lesson2/app"
	"net/http"
)

func Write(w http.ResponseWriter, r *http.Request, err error) {
	code, ok := app.CodeOf(err)
	if !ok {
		respond.AppError(w, r, http.StatusInternalServerError, string(app.CodeInternal), "internal error")
		return
	}
	respond.AppError(w, r, statusBy(code), string(code), err.Error())
}

func statusBy(c app.Code) int {
	switch c {
	case app.CodeInvalid:
		return http.StatusBadRequest
	case app.CodeNotFound:
		return http.StatusNotFound
	case app.CodeConflict:
		return http.StatusConflict
	case app.CodeUnauthorized:
		return http.StatusUnauthorized
	case app.CodeForbidden:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
