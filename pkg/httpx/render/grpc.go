package render

import (
	"encoding/json"
	"net/http"

	tctx "github.com/YanMak/ecommerce/v2/pkg/telemetry/ctx"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Violation struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

type ErrorBody struct {
	Code           string      `json:"code,omitempty"`
	Message        string      `json:"message"`
	RequestID      string      `json:"request_id,omitempty"`
	IdempotencyKey string      `json:"idempotency_key,omitempty"`
	Violations     []Violation `json:"violations,omitempty"`
}

func GRPCError(w http.ResponseWriter, r *http.Request, err error) {
	st, ok := status.FromError(err)
	if !ok {
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	body := ErrorBody{
		Message:        st.Message(),
		RequestID:      tctx.RequestID(r.Context()),
		IdempotencyKey: tctx.IdempotencyKey(r.Context()),
	}

	// детали: ErrorInfo(Code) и BadRequest(violations)
	for _, d := range st.Details() {
		switch x := d.(type) {
		case *errdetails.ErrorInfo:
			if x.Reason != "" {
				body.Code = x.Reason
			}
			// если в статусе есть request_id/idempotency_key — приоритет отдаём им
			if v := x.Metadata["request_id"]; v != "" {
				body.RequestID = v
			}
			if v := x.Metadata["idempotency_key"]; v != "" {
				body.IdempotencyKey = v
			}
		case *errdetails.BadRequest:
			for _, fv := range x.FieldViolations {
				body.Violations = append(body.Violations, Violation{
					Field: fv.Field, Reason: fv.Description,
				})
			}
		}
	}

	code := httpStatusFromCode(st.Code())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func httpStatusFromCode(c codes.Code) int {
	switch c {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists, codes.FailedPrecondition:
		return http.StatusConflict
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unavailable, codes.DeadlineExceeded:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
