// services/crm/internal/adapters/inbound/grpc/errors.go
package grpcin

import (
	"context"
	"errors"

	"github.com/YanMak/ecommerce/v2/pkg/errkit"
	"github.com/YanMak/ecommerce/v2/pkg/errkitx"
	tctx "github.com/YanMak/ecommerce/v2/pkg/telemetry/ctx"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toStatus(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	// уже статус — не перепаковываем
	if s, ok := status.FromError(err); ok && s.Code() != codes.OK {
		return err
	}

	var ee *errkitx.E
	if errors.As(err, &ee) {
		code := codeFromKind(ee.Kind)
		st := status.New(code, ee.Msg)

		// ErrorInfo с code + request-id/idempotency-key
		info := &errdetails.ErrorInfo{
			Reason: ee.Code,
			Metadata: map[string]string{
				"request_id":      tctx.RequestID(ctx),
				"idempotency_key": tctx.IdempotencyKey(ctx),
				"retryable":       boolToStr(ee.Retryable),
			},
		}
		st, _ = st.WithDetails(info)

		// BadRequest для валидации
		if ee.Kind == errkitx.KindInvalid && len(ee.Violations) > 0 {
			br := &errdetails.BadRequest{}
			for _, v := range ee.Violations {
				br.FieldViolations = append(br.FieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       v.Field,
					Description: v.Reason,
				})
			}
			st, _ = st.WithDetails(br)
		}
		return st.Err()
	}

	// Fallback по базовым сентинелам:
	switch {
	case errors.Is(err, errkit.ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid")
	case errors.Is(err, errkit.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, errkit.ErrConflict):
		return status.Error(codes.AlreadyExists, "conflict")
	case errors.Is(err, errkit.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, "unauthenticated")
	case errors.Is(err, errkit.ErrForbidden):
		return status.Error(codes.PermissionDenied, "forbidden")
	case errkit.IsTransient(err):
		return status.Error(codes.Unavailable, "transient")
	case errors.Is(err, errkit.ErrDeadline):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	case errors.Is(err, errkit.ErrCanceled):
		return status.Error(codes.Canceled, "canceled")
	default:
		return status.Error(codes.Internal, "internal")
	}
}

func codeFromKind(k errkitx.Kind) codes.Code {
	switch k {
	case errkitx.KindInvalid:
		return codes.InvalidArgument
	case errkitx.KindNotFound:
		return codes.NotFound
	case errkitx.KindConflict:
		return codes.AlreadyExists
	case errkitx.KindUnauthorized:
		return codes.Unauthenticated
	case errkitx.KindForbidden:
		return codes.PermissionDenied
	case errkitx.KindTransient:
		return codes.Unavailable
	case errkitx.KindDeadline:
		return codes.DeadlineExceeded
	case errkitx.KindCanceled:
		return codes.Canceled
	default:
		return codes.Internal
	}
}
func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
