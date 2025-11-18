package dto

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/YanMak/ecommerce/v2/pkg/httpx/render"
	"github.com/go-playground/validator/v10"
)

type UpsertDocumentDTO struct {
	ID            int64  `json:"id"`
	CertificateID int64  `json:"certificate_id"`
	URL           string `json:"url"`
	URLMachine    string `json:"url_machine"`
}

func BindCRMUpsertDocQuery(r *http.Request) (UpsertDocumentDTO, []render.Violation) {
	var b UpsertDocumentDTO

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil {
		if name := unknownJSONField(err); name != "" {
			return b, []render.Violation{{Field: "body." + name, Reason: "unknown field"}}
		}
		var te *json.UnmarshalTypeError
		if errors.As(err, &te) && te.Field != "" {
			return b, []render.Violation{{Field: "body." + te.Field, Reason: "invalid type"}}
		}
		return b, []render.Violation{{Field: "body", Reason: "bad json"}}
	}

	// // валидация по тегам
	// v := validate.New("json")
	// if err := v.Struct(b); err != nil {
	// 	fe := err.(validator.ValidationErrors)
	// 	fields := make([]respond.FieldError, 0, len(fe))
	// 	for _, e := range fe {
	// 		fields = append(fields, respond.FieldError{
	// 			Loc: "body." + e.Field(),
	// 			Msg: humanMsg(e),
	// 		})
	// 	}
	// 	return b, fields
	// }
	return b, nil
}

func humanMsg(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "is required"
	case "min":
		return "must be >= " + e.Param()
	case "max":
		return "must be <= " + e.Param()
	case "uuid4":
		return "must be uuid"
	default:
		return "invalid"
	}
}

func unknownJSONField(err error) string {
	const p = "json: unknown field "
	if s := err.Error(); strings.HasPrefix(s, p) {
		return strings.Trim(s[len(p):], `"`)
	}
	return ""
}
