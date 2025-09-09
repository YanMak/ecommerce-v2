package search

import (
	"encoding/json"
	"errors"
	"fmt"
	"lesson2/adapters/inbound/httpapi/respond"
	"lesson2/adapters/inbound/httpapi/validate"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Price struct {
	Min *float64 `json:"min" validate:"omitempty"`
	Max *float64 `json:"max" validate:"omitempty"`
}

type SearchBody struct {
	Q           string   `json:"q" validate:"omitempty,min=1"`
	CategoryIDs []string `json:"category_ids" validate:"dive,uuid4"` // массив UUID
	Price       Price    `json:"price"`
	Tags        []string `json:"tags" validate:"dive,min=1"`
}

func BindSearchBody(r *http.Request) (SearchBody, []respond.FieldError, error) {
	var b SearchBody

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil {
		if name := unknownJSONField(err); name != "" {
			return b, []respond.FieldError{{Loc: "body." + name, Msg: "unknown field"}}, fmt.Errorf("bad json")
		}
		var te *json.UnmarshalTypeError
		if errors.As(err, &te) && te.Field != "" {
			return b, []respond.FieldError{{Loc: "body." + te.Field, Msg: "invalid type"}}, fmt.Errorf("bad json")
		}
		return b, []respond.FieldError{{Loc: "body", Msg: "bad json"}}, fmt.Errorf("bad json")
	}

	// доп. проверка min<=max
	if b.Price.Min != nil && b.Price.Max != nil && *b.Price.Min > *b.Price.Max {
		return b, []respond.FieldError{{Loc: "body.price", Msg: "min must be <= max"}}, fmt.Errorf("invalid price range")
	}
	// нормализованная проверка UUID массива (нагляднее, чем одна только тэг-валидация)
	for i, id := range b.CategoryIDs {
		if _, err := uuid.Parse(id); err != nil {
			return b, []respond.FieldError{{Loc: fmt.Sprintf("body.category_ids[%d]", i), Msg: "must be uuid"}}, fmt.Errorf("invalid")
		}
	}

	// валидация по тегам
	v := validate.New("json")
	if err := v.Struct(b); err != nil {
		fe := err.(validator.ValidationErrors)
		fields := make([]respond.FieldError, 0, len(fe))
		for _, e := range fe {
			fields = append(fields, respond.FieldError{
				Loc: "body." + e.Field(),
				Msg: humanMsg(e),
			})
		}
		return b, fields, fmt.Errorf("validation failed")
	}
	return b, nil, nil
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
