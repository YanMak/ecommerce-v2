package validate

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

func New(tag string) *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := fld.Tag.Get(tag)
		if name == "" {
			name = strings.Split(fld.Tag.Get("json"), ",")[0]
		} else {
			name = strings.Split(name, ",")[0]
		}
		if name == "" || name == "-" {
			name = fld.Name
		}
		return name
	})
	return v
}
