package grpcx

import (
	"sort"
	"strings"

	"github.com/YanMak/ecommerce/v2/pkg/errorsx"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InvalidArgumentError строит gRPC-ошибку с кодом INVALID_ARGUMENT
// и прикручивает google.rpc.BadRequest с FieldViolation'ами.
// summary — короткий заголовок ("invalid request"); пустой заменяется на дефолт.
func InvalidArgumentError(summary string, v []errorsx.Violation) error {
	if summary == "" {
		summary = "invalid request"
	}
	st := status.New(codes.InvalidArgument, summary)
	br := badRequestFrom(v)
	stWith, err := st.WithDetails(br)
	if err != nil {
		// На случай несовместимых деталей — вернём ошибку без деталей.
		return st.Err()
	}
	return stWith.Err()
}

// IsInvalidArgument — быстро проверить код gRPC-ошибки.
func IsInvalidArgument(err error) bool {
	st, ok := status.FromError(err)
	return ok && st.Code() == codes.InvalidArgument
}

// ExtractBadRequest — достать google.rpc.BadRequest из gRPC-ошибки (если есть).
func ExtractBadRequest(err error) (*errdetails.BadRequest, bool) {
	st, ok := status.FromError(err)
	if !ok {
		return nil, false
	}
	for _, d := range st.Details() {
		if br, ok := d.(*errdetails.BadRequest); ok {
			return br, true
		}
	}
	return nil, false
}

// ---- внутреннее ----

func badRequestFrom(vs []errorsx.Violation) *errdetails.BadRequest {
	br := &errdetails.BadRequest{FieldViolations: make([]*errdetails.BadRequest_FieldViolation, 0)}
	if len(vs) == 0 {
		return br
	}
	// Стабильный порядок для тестов/логов.
	sort.SliceStable(vs, func(i, j int) bool {
		a, b := vs[i], vs[j]
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Message < b.Message
	})
	for _, v := range vs {
		desc := joinDescription(v.Code, v.Message, v.Params)
		field := v.Field
		if field == "" {
			field = "_" // защита от пустых имён
		}
		br.FieldViolations = append(br.FieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       field,
			Description: desc,
		})
	}
	return br
}

func joinDescription(code, msg string, params map[string]string) string {
	var parts []string
	if code != "" {
		parts = append(parts, code)
	}
	if msg != "" {
		parts = append(parts, msg)
	}
	base := strings.Join(parts, ": ")
	if len(params) == 0 {
		return base
	}
	// стабилизируем порядок параметров
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	kv := make([]string, 0, len(keys))
	for _, k := range keys {
		kv = append(kv, k+"="+params[k])
	}
	if base == "" {
		return "(" + strings.Join(kv, ", ") + ")"
	}
	return base + " (" + strings.Join(kv, ", ") + ")"
}
