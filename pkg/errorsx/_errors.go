package errorsx

import (
	"errors"
	"fmt"
)

// Канонические корневые (сентинел) ошибки — единые для всех сервисов.
// Их мы будем маппить в коды gRPC/HTTP (см. grpcx/ и слой BFF).
var (
	ErrInvalidArgument    = errors.New("invalid argument")    // 400 / INVALID_ARGUMENT
	ErrNotFound           = errors.New("not found")           // 404 / NOT_FOUND
	ErrAlreadyExists      = errors.New("already exists")      // 409 / ALREADY_EXISTS
	ErrConflict           = errors.New("conflict")            // 409 / CONFLICT (общий)
	ErrAborted            = errors.New("aborted")             // 409 / ABORTED (оптимистическая блокировка)
	ErrFailedPrecondition = errors.New("failed precondition") // 412/422 / FAILED_PRECONDITION (инварианты)
	ErrUnauthenticated    = errors.New("unauthenticated")     // 401 / UNAUTHENTICATED
	ErrPermissionDenied   = errors.New("permission denied")   // 403 / PERMISSION_DENIED
	ErrResourceExhausted  = errors.New("resource exhausted")  // 429 / RESOURCE_EXHAUSTED (лимиты/квоты)
	ErrUnavailable        = errors.New("unavailable")         // 503 / UNAVAILABLE
	ErrInternal           = errors.New("internal")            // 500 / INTERNAL
)

// ---------- Проверки (Is*) ----------

func IsInvalidArgument(err error) bool    { return errors.Is(err, ErrInvalidArgument) }
func IsNotFound(err error) bool           { return errors.Is(err, ErrNotFound) }
func IsAlreadyExists(err error) bool      { return errors.Is(err, ErrAlreadyExists) }
func IsConflict(err error) bool           { return errors.Is(err, ErrConflict) }
func IsAborted(err error) bool            { return errors.Is(err, ErrAborted) }
func IsFailedPrecondition(err error) bool { return errors.Is(err, ErrFailedPrecondition) }
func IsUnauthenticated(err error) bool    { return errors.Is(err, ErrUnauthenticated) }
func IsPermissionDenied(err error) bool   { return errors.Is(err, ErrPermissionDenied) }
func IsResourceExhausted(err error) bool  { return errors.Is(err, ErrResourceExhausted) }
func IsUnavailable(err error) bool        { return errors.Is(err, ErrUnavailable) }
func IsInternal(err error) bool           { return errors.Is(err, ErrInternal) }

// ---------- Обёртки (Wrap*) ----------
// Возвращают error с сохранением «корня» через %w, чтобы Is(...) работал.
// Используй их в use case'ах: return errorsx.NotFoundf("item %d", id)

func InvalidArgument(msg string) error { return fmt.Errorf("%w: %s", ErrInvalidArgument, msg) }
func InvalidArgumentf(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidArgument}, a...)...)
}

func NotFound(msg string) error { return fmt.Errorf("%w: %s", ErrNotFound, msg) }
func NotFoundf(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrNotFound}, a...)...)
}

func AlreadyExists(msg string) error { return fmt.Errorf("%w: %s", ErrAlreadyExists, msg) }
func AlreadyExistsf(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrAlreadyExists}, a...)...)
}

func Conflict(msg string) error { return fmt.Errorf("%w: %s", ErrConflict, msg) }
func Conflictf(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrConflict}, a...)...)
}

func Aborted(msg string) error { return fmt.Errorf("%w: %s", ErrAborted, msg) } // оптимистическая блокировка и т.п.
func Abortedf(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrAborted}, a...)...)
}

func FailedPrecondition(msg string) error { return fmt.Errorf("%w: %s", ErrFailedPrecondition, msg) }
func FailedPreconditionf(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrFailedPrecondition}, a...)...)
}

func Unauthenticated(msg string) error { return fmt.Errorf("%w: %s", ErrUnauthenticated, msg) }
func Unauthenticatedf(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrUnauthenticated}, a...)...)
}

func PermissionDenied(msg string) error { return fmt.Errorf("%w: %s", ErrPermissionDenied, msg) }
func PermissionDeniedf(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrPermissionDenied}, a...)...)
}

func ResourceExhausted(msg string) error { return fmt.Errorf("%w: %s", ErrResourceExhausted, msg) }
func ResourceExhaustedf(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrResourceExhausted}, a...)...)
}

func Unavailable(msg string) error { return fmt.Errorf("%w: %s", ErrUnavailable, msg) }
func Unavailablef(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrUnavailable}, a...)...)
}

func Internal(msg string) error { return fmt.Errorf("%w: %s", ErrInternal, msg) }
func Internalf(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInternal}, a...)...)
}

// ---------- Примечания по использованию ----------
//
// 1) В use case возвращай «класс» через Wrap*:
//    - отсутствие записи:        return errorsx.NotFoundf("item %d", id)
//    - уникальный конфликт:      return errorsx.AlreadyExists("slug taken")
//    - оптимистическая блок.:    return errorsx.Aborted("version mismatch")
//    - инвариант/бизнес-правило: return errorsx.FailedPrecondition("would become negative")
//    - плохой ввод (одно поле):
//         return errorsx.InvalidArgument("price_cents must be >= 0")
//       (для множественной валидации используем ValidationError — добавим в отдельном файле validation.go)
//
// 2) В gRPC-сервере маппим коды:
//    if errorsx.IsInvalidArgument(err)   -> codes.InvalidArgument
//    if errorsx.IsNotFound(err)          -> codes.NotFound
//    if errorsx.IsAlreadyExists(err)     -> codes.AlreadyExists
//    if errorsx.IsAborted(err)           -> codes.Aborted
//    if errorsx.IsFailedPrecondition(err)-> codes.FailedPrecondition
//    ...
//
// 3) Для множественной валидации (несколько полей) — будет pkg/errsx/validation.go:
//    ValidationError с Unwrap()->ErrInvalidArgument и списком Violations.
//    В gRPC упакуем через pkg/grpcx.InvalidArgumentError(...).
