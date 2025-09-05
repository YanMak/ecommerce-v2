package errorsx

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Violation — нейтральная запись нарушения (transport-agnostic).
type Violation struct {
	Field   string            // "name", "price_cents", "items.0.qty"
	Code    string            // "REQUIRED", "OUT_OF_RANGE", ...
	Message string            // человекочитаемое пояснение
	Params  map[string]string // опционально: {"min":"3","max":"64"}
}

// ValidationError — несколько нарушений; Unwrap() => ErrInvalidArgument.
type ValidationError struct {
	violations []Violation
}

func NewValidation() *ValidationError { return &ValidationError{} }

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
	e.violations = append(e.violations, Violation{Field: field, Code: code, Message: message, Params: p})
	return e
}

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

func (e *ValidationError) WithPrefix(prefix string) *ValidationError {
	if e == nil || prefix == "" {
		return e
	}
	prefix = strings.TrimSuffix(prefix, ".")
	for i := range e.violations {
		if e.violations[i].Field != "" {
			e.violations[i].Field = prefix + "." + e.violations[i].Field
		}
	}
	return e
}

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

func (e *ValidationError) Error() string {
	n := e.Len()
	if n == 0 {
		return "validation failed (0)"
	}
	const k = 3
	sample := make([]string, 0, min(n, k))
	for i := 0; i < min(n, k); i++ {
		v := e.violations[i]
		sample = append(sample, fmt.Sprintf("%s:%s", v.Field, v.Code))
	}
	if n > k {
		return fmt.Sprintf("validation failed (%d): %s, …", n, strings.Join(sample, ", "))
	}
	return fmt.Sprintf("validation failed (%d): %s", n, strings.Join(sample, ", "))
}

func (e *ValidationError) Unwrap() error { return ErrInvalidArgument }

func (e *ValidationError) Violations() []Violation {
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

func AsValidation(err error) (*ValidationError, bool) {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve, true
	}
	return nil, false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
