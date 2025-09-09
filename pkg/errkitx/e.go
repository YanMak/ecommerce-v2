package errkitx

import (
	"errors"
	"fmt"

	"github.com/YanMak/ecommerce/v2/pkg/errkit"
)

type Kind int

const (
	KindInvalid Kind = iota
	KindNotFound
	KindConflict
	KindUnauthorized
	KindForbidden
	KindTransient
	KindDeadline
	KindCanceled
	KindInternal
)

type Violation struct {
	Field  string // "page", "per_page", "created_from"
	Reason string // "must be >= 1", "invalid format", ...
}

type E struct {
	Kind       Kind
	Code       string      // "INVALID_PAGING", "CERT_NOT_FOUND", ...
	Msg        string      // человекочитаемое короткое
	Violations []Violation // только для KindInvalid (может быть nil)
	Retryable  bool
	Err        error // внутренняя причина; ДОЛЖНА включать корневой errkit-сентинел через %w
}

func (e *E) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return fmt.Sprint(e.Err)
}
func (e *E) Unwrap() error { return e.Err }

func Invalid(msg string, v ...Violation) error {
	return &E{Kind: KindInvalid, Code: "INVALID", Msg: msg, Violations: v, Err: errkit.ErrInvalid}
}
func Conflict(code, msg string, cause error) error {
	if cause == nil {
		cause = errkit.ErrConflict
	}
	return &E{Kind: KindConflict, Code: code, Msg: msg, Err: cause}
}

// … при необходимости добавишь конструкторы под другие виды

// Удобная проверка по виду:
func IsKind(kind Kind, err error) bool {
	var e *E
	if errors.As(err, &e) && e.Kind == kind {
		return true
	}
	// плюс уважим базовые сентинелы:
	switch kind {
	case KindInvalid:
		return errors.Is(err, errkit.ErrInvalid)
	case KindNotFound:
		return errors.Is(err, errkit.ErrNotFound)
	case KindConflict:
		return errors.Is(err, errkit.ErrConflict)
	case KindTransient:
		return errkit.IsTransient(err)
	case KindDeadline:
		return errors.Is(err, errkit.ErrDeadline)
	case KindCanceled:
		return errors.Is(err, errkit.ErrCanceled)
	case KindUnauthorized:
		return errors.Is(err, errkit.ErrUnauthorized)
	case KindForbidden:
		return errors.Is(err, errkit.ErrForbidden)
	case KindInternal:
		return errors.Is(err, errkit.ErrInternal)
	default:
		return false
	}
}
