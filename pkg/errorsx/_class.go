package errorsx

import (
	"errors"
	"fmt"
)

// Kind — класс (таксономия) ошибки, независимый от транспорта.
type Kind string

const (
	KindInvalid            Kind = "INVALID"             // плохой ввод/команда (обычно 400 / INVALID_ARGUMENT)
	KindNotFound           Kind = "NOT_FOUND"           // 404 / NOT_FOUND
	KindAlreadyExists      Kind = "ALREADY_EXISTS"      // 409 / ALREADY_EXISTS
	KindConflict           Kind = "CONFLICT"            // 409 / CONFLICT (общий случай)
	KindAborted            Kind = "ABORTED"             // 409 / ABORTED (напр., оптимистическая блокировка)
	KindFailedPrecondition Kind = "FAILED_PRECONDITION" // 412/422 / FAILED_PRECONDITION (нарушение инварианта)
	KindUnauthenticated    Kind = "UNAUTHENTICATED"     // 401 / UNAUTHENTICATED
	KindPermission         Kind = "PERMISSION"          // 403 / PERMISSION_DENIED
	KindRateLimited        Kind = "RATE_LIMITED"        // 429 / RESOURCE_EXHAUSTED
	KindUnavailable        Kind = "UNAVAILABLE"         // 503 / UNAVAILABLE (временная недоступность)
	KindInternal           Kind = "INTERNAL"            // 500 / INTERNAL
)

// E — «богатая» ошибка с классификацией.
// Важно: Unwrap() возвращает цепочку с корневым сентинелом (ErrInvalidArgument и т.п.),
// чтобы errors.Is(err, ErrInvalidArgument) продолжал работать.
type E struct {
	Kind       Kind
	Code       string      // публичный машинный код, напр. "ITEM_NOT_FOUND", "OPTIMISTIC_CONFLICT"
	Retryable  bool        // можно ли автоматически ретраить
	Violations []Violation // для KindInvalid (много-полевая валидация), можно оставить nil
	Err        error       // внутренняя «причина»; включает корневой сентинел через %w
}

// Error — короткое описание; детали берите из полей, Violations и Unwrap().
func (e *E) Error() string {
	if e == nil {
		return "error(nil)"
	}
	if e.Code != "" {
		return fmt.Sprintf("%s [%s]", e.Kind, e.Code)
	}
	return string(e.Kind)
}

// Unwrap — возвращает причину, в которую уже «вшит» корневой сентинел (через %w).
func (e *E) Unwrap() error { return e.Err }

// ---------- Публичные конструкторы (удобные шорткаты) ----------

func Invalid(code string, v []Violation) error { return newE(KindInvalid, code, nil, v, nil) }
func InvalidWithCause(code string, v []Violation, cause error) error {
	return newE(KindInvalid, code, nil, v, cause)
}

func NotFound(code string) error { return newE(KindNotFound, code, nil, nil, nil) }
func NotFoundWithCause(code string, cause error) error {
	return newE(KindNotFound, code, nil, nil, cause)
}

func AlreadyExists(code string) error { return newE(KindAlreadyExists, code, nil, nil, nil) }
func AlreadyExistsWithCause(code string, cause error) error {
	return newE(KindAlreadyExists, code, nil, nil, cause)
}

func Conflict(code string) error { return newE(KindConflict, code, nil, nil, nil) }
func ConflictWithCause(code string, cause error) error {
	return newE(KindConflict, code, nil, nil, cause)
}

func Aborted(code string) error { return newE(KindAborted, code, nil, nil, nil) }
func AbortedWithCause(code string, cause error) error {
	return newE(KindAborted, code, nil, nil, cause)
}

func FailedPrecondition(code string) error { return newE(KindFailedPrecondition, code, nil, nil, nil) }
func FailedPreconditionWithCause(code string, cause error) error {
	return newE(KindFailedPrecondition, code, nil, nil, cause)
}

func Unauthenticated(code string) error { return newE(KindUnauthenticated, code, nil, nil, nil) }
func UnauthenticatedWithCause(code string, cause error) error {
	return newE(KindUnauthenticated, code, nil, nil, cause)
}

func PermissionDenied(code string) error { return newE(KindPermission, code, nil, nil, nil) }
func PermissionDeniedWithCause(code string, cause error) error {
	return newE(KindPermission, code, nil, nil, cause)
}

func RateLimited(code string) error { return newE(KindRateLimited, code, nil, nil, nil) }
func RateLimitedWithCause(code string, cause error) error {
	return newE(KindRateLimited, code, nil, nil, cause)
}

func Unavailable(code string) error { return newE(KindUnavailable, code, nil, nil, nil) }
func UnavailableWithCause(code string, cause error) error {
	return newE(KindUnavailable, code, nil, nil, cause)
}

func Internal(code string) error { return newE(KindInternal, code, nil, nil, nil) }
func InternalWithCause(code string, cause error) error {
	return newE(KindInternal, code, nil, nil, cause)
}

// Wrap — универсальный конструктор, если хочется явно задать retryable/причину/violations.
func Wrap(k Kind, code string, retryable bool, v []Violation, cause error) error {
	return newE(k, code, &retryable, v, cause)
}

// ---------- Инспекция (геттеры/проверки) ----------

func AsE(err error) (*E, bool) {
	var e *E
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

func KindOf(err error) Kind {
	if e, ok := AsE(err); ok {
		return e.Kind
	}
	// Фоллбэк по корневым сентинелам для «старых» ошибок.
	switch {
	case errors.Is(err, ErrInvalidArgument):
		return KindInvalid
	case errors.Is(err, ErrNotFound):
		return KindNotFound
	case errors.Is(err, ErrAlreadyExists):
		return KindAlreadyExists
	case errors.Is(err, ErrConflict):
		return KindConflict
	case errors.Is(err, ErrAborted):
		return KindAborted
	case errors.Is(err, ErrFailedPrecondition):
		return KindFailedPrecondition
	case errors.Is(err, ErrUnauthenticated):
		return KindUnauthenticated
	case errors.Is(err, ErrPermissionDenied):
		return KindPermission
	case errors.Is(err, ErrResourceExhausted):
		return KindRateLimited
	case errors.Is(err, ErrUnavailable):
		return KindUnavailable
	case errors.Is(err, ErrInternal):
		return KindInternal
	default:
		return "" // неизвестный Kind
	}
}

func IsKind(err error, k Kind) bool { return KindOf(err) == k }

func CodeOf(err error) string {
	if e, ok := AsE(err); ok {
		return e.Code
	}
	return ""
}

func RetryableOf(err error) bool {
	if e, ok := AsE(err); ok {
		return e.Retryable
	}
	// Дефолтная политика для «старых» ошибок по Kind.
	switch KindOf(err) {
	case KindUnavailable, KindRateLimited:
		return true
	default:
		return false
	}
}

// ---------- внутренняя кухня ----------

func newE(k Kind, code string, retryable *bool, v []Violation, cause error) error {
	if code == "" {
		code = string(k)
	}
	// Скопируем violations, чтобы снаружи никому не пришла в голову их мутировать.
	var vv []Violation
	if len(v) > 0 {
		vv = make([]Violation, len(v))
		copy(vv, v)
	}

	// Сформируем цепочку причины: корневой сентинел + (опционально) прикладная причина.
	root := sentinelForKind(k)
	if cause != nil {
		root = fmt.Errorf("%w: %v", root, cause)
	}

	e := &E{
		Kind:       k,
		Code:       code,
		Retryable:  defaultRetryable(k),
		Violations: vv,
		Err:        root,
	}
	if retryable != nil {
		e.Retryable = *retryable
	}
	return e
}

func sentinelForKind(k Kind) error {
	switch k {
	case KindInvalid:
		return ErrInvalidArgument
	case KindNotFound:
		return ErrNotFound
	case KindAlreadyExists:
		return ErrAlreadyExists
	case KindConflict:
		return ErrConflict
	case KindAborted:
		return ErrAborted
	case KindFailedPrecondition:
		return ErrFailedPrecondition
	case KindUnauthenticated:
		return ErrUnauthenticated
	case KindPermission:
		return ErrPermissionDenied
	case KindRateLimited:
		return ErrResourceExhausted
	case KindUnavailable:
		return ErrUnavailable
	case KindInternal:
		return ErrInternal
	default:
		// Незнакомый класс — считаем как INTERNAL, чтобы не уронить поведение.
		return ErrInternal
	}
}

func defaultRetryable(k Kind) bool {
	switch k {
	case KindUnavailable, KindRateLimited:
		return true // временные, имеет смысл ретраить
	case KindInternal:
		return false // по умолчанию не ретраим; можно переопределить Wrap(..., retryable=true)
	default:
		return false
	}
}
