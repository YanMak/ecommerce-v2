package errorsx

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Violation — нейтральное (транспорт-независимое) описание нарушения.
// Его удобно потом маппить и в gRPC errdetails.BadRequest, и в RFC 7807 (HTTP).
type Violation struct {
	Field   string            // dot-path: "name", "price_cents", "items.0.qty"
	Code    string            // машиночитаемый код: "REQUIRED", "OUT_OF_RANGE", "DUPLICATE"...
	Message string            // человекочитаемое пояснение
	Params  map[string]string // опциональные параметры: {"min":"3","max":"64"}
}

// ValidationError — несколько нарушений за один запрос.
// ВАЖНО: Unwrap() возвращает ErrInvalidArgument, чтобы errors.Is(err, ErrInvalidArgument) == true.
type ValidationError struct {
	violations []Violation
}

// Error — компактная сводка для логов (полные детали — через Violations()).
func (e *ValidationError) Error() string {
	if e == nil {
		return "validation failed (0)"
	}
	n := len(e.violations)
	const sampleN = 3
	sample := make([]string, 0, min(n, sampleN))
	for i := 0; i < min(n, sampleN); i++ {
		v := e.violations[i]
		sample = append(sample, fmt.Sprintf("%s:%s", v.Field, v.Code))
	}
	ellipsis := ""
	if n > sampleN {
		ellipsis = ", …"
	}
	return fmt.Sprintf("validation failed (%d): %s%s", n, strings.Join(sample, ", "), ellipsis)
}

// Unwrap — главный крючок для классовой проверки.
func (e *ValidationError) Unwrap() error { return ErrInvalidArgument }

// Violations — возвращает КОПИЮ списка нарушений (иммутабельно наружу).
func (e *ValidationError) Violations() []Violation {
	if e == nil {
		return nil
	}
	out := make([]Violation, len(e.violations))
	copy(out, e.violations)
	return out
}

func (e *ValidationError) Len() int {
	if e == nil {
		return 0
	}
	return len(e.violations)
}
func (e *ValidationError) IsEmpty() bool { return e.Len() == 0 }

// Add — добавить нарушение; возвращает self для чейнинга.
// Пример: ve.Add("name","REQUIRED","name is required")
func (e *ValidationError) Add(field, code, message string, params map[string]string) *ValidationError {
	if e == nil {
		e = &ValidationError{}
	}
	var p map[string]string
	if len(params) > 0 {
		p = make(map[string]string, len(params))
		for k, v := range params {
			p[k] = v
		}
	}
	e.violations = append(e.violations, Violation{
		Field: field, Code: code, Message: message, Params: p,
	})
	return e
}

// Merge — добавить все нарушения из другой ошибки (nil-safe).
func (e *ValidationError) Merge(other *ValidationError) *ValidationError {
	if other == nil || len(other.violations) == 0 {
		return e
	}
	if e == nil {
		e = &ValidationError{}
	}
	e.violations = append(e.violations, other.violations...)
	return e
}

// WithPrefix — префикс для полей (вложенные структуры): "address.city" и т.п.
func (e *ValidationError) WithPrefix(prefix string) *ValidationError {
	if e == nil || prefix == "" {
		return e
	}
	prefix = strings.TrimSuffix(prefix, ".")
	for i := range e.violations {
		if e.violations[i].Field == "" {
			continue
		}
		e.violations[i].Field = prefix + "." + e.violations[i].Field
	}
	return e
}

// Sort — стабилизировать порядок (тесты/логи).
func (e *ValidationError) Sort() *ValidationError {
	if e == nil {
		return e
	}
	sort.SliceStable(e.violations, func(i, j int) bool {
		a, b := e.violations[i], e.violations[j]
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Message < b.Message
	})
	return e
}

// AsValidation — вытащить *ValidationError из error-цепочки.
func AsValidation(err error) (*ValidationError, bool) {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve, true
	}
	return nil, false
}

// Хелперы-конструкторы
func NewValidation() *ValidationError { return &ValidationError{} }
func SingleViolation(v Violation) *ValidationError {
	return (&ValidationError{}).Add(v.Field, v.Code, v.Message, v.Params)
}

// ——— утилитарный минимум ———
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
