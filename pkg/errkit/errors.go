package errkit

import "errors"

// Сентинелы (категории ошибок), видимые всем слоям.
var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrInvalid      = errors.New("invalid")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")

	ErrTransient = errors.New("transient") // временная/повторяемая
	ErrDeadline  = errors.New("deadline exceeded")
	ErrCanceled  = errors.New("canceled")
	ErrInternal  = errors.New("internal")
)

// Утилиты
func IsTransient(err error) bool { return errors.Is(err, ErrTransient) }
