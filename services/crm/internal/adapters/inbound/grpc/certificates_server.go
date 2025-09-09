package grpcin

import (
	"context"
	"errors"
	"time"

	crmpb "github.com/YanMak/ecommerce/v2/api/gen/go/crm/certificates/v1"
	"github.com/YanMak/ecommerce/v2/pkg/errkit"
	"github.com/YanMak/ecommerce/v2/pkg/errkitx"
	"github.com/YanMak/ecommerce/v2/pkg/grpcx"
	"github.com/YanMak/ecommerce/v2/pkg/paging"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/app/contracts"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/app/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

type CertificatesServer struct {
	crmpb.UnimplementedCertificatesServer
	uc *usecase.CertificatesUC
}

func NewCertificatesServer(uc *usecase.CertificatesUC) *CertificatesServer {
	return &CertificatesServer{uc: uc}
}

func (s *CertificatesServer) SearchCertificates(ctx context.Context, req *crmpb.SearchCertificatesRequest) (*crmpb.SearchCertificatesResponse, error) {

	// переносим метаданные в наш context
	//ctx = grpcx.IncomingToContext(ctx)

	// 0) валидация входа
	if err := validateSearchRequest(req); err != nil {
		return nil, toStatus(ctx, err)
	}

	// 1) собрать фильтры
	f := contracts.SearchFilter{
		Q:           grpcx.SPtr(req.Q),
		Inn:         grpcx.SPtr(req.Inn),
		CreatedFrom: grpcx.TSPtr(req.CreatedFrom),
		CreatedTo:   grpcx.TSPtr(req.CreatedTo),
		UpdatedFrom: grpcx.TSPtr(req.UpdatedFrom),
		UpdatedTo:   grpcx.TSPtr(req.UpdatedTo),
		CategoryID:  grpcx.I64Ptr(req.CategoryId),
		Opened:      grpcx.BPtr(req.Opened),
	}
	p := paging.OffsetParams{Page: int32(req.Page), PerPage: int32(req.PerPage)}

	// 2) usecase
	rows, total, hasNext, err := s.uc.Search(ctx, f, p)
	if err != nil {
		return nil, toStatus(ctx, err)
	}

	// 3) ответ
	out := make([]*crmpb.CertificateRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, &crmpb.CertificateRow{
			Id:          r.ID,
			Title:       r.Title,
			UfNumber:    grpcx.S(r.UfNumber),
			UfInn:       grpcx.S(r.UfInn),
			CategoryId:  r.CategoryID,
			Opened:      r.Opened,
			CreatedTime: grpcx.TS(r.CreatedTime),
			UpdatedTime: grpcx.TS(r.UpdatedTime),
		})
	}

	return &crmpb.SearchCertificatesResponse{
		Rows:    out,
		Total:   total,
		HasNext: hasNext,
	}, nil
}

func validateSearchRequest(req *crmpb.SearchCertificatesRequest) error {
	var v []errkitx.Violation

	// Пагинация
	if req.Page <= 0 {
		v = append(v, errkitx.Violation{Field: "page", Reason: "must be >= 1"})
	}
	if req.PerPage <= 0 {
		v = append(v, errkitx.Violation{Field: "per_page", Reason: "must be > 0"})
	}
	// (опционально) верхняя граница per_page
	if req.PerPage > 200 {
		v = append(v, errkitx.Violation{Field: "per_page", Reason: "must be <= 200"})
	}

	// Диапазоны дат (если оба заданы)
	if req.CreatedFrom != nil && req.CreatedTo != nil && req.CreatedFrom.AsTime().After(req.CreatedTo.AsTime()) {
		v = append(v, errkitx.Violation{Field: "created_from/created_to", Reason: "created_from must be <= created_to"})
	}
	if req.UpdatedFrom != nil && req.UpdatedTo != nil && req.UpdatedFrom.AsTime().After(req.UpdatedTo.AsTime()) {
		v = append(v, errkitx.Violation{Field: "updated_from/updated_to", Reason: "updated_from must be <= updated_to"})
	}

	if len(v) > 0 {
		return errkitx.Invalid("invalid search parameters", v...)
	}
	return nil
}

// ----- helpers -----

// unwrap google.protobuf.*Value -> *T
func strPtr(v *wrapperspb.StringValue) *string {
	if v == nil {
		return nil
	}
	s := v.Value
	return &s
}
func i64Ptr(v *wrapperspb.Int64Value) *int64 {
	if v == nil {
		return nil
	}
	i := v.Value
	return &i
}
func boolPtr(v *wrapperspb.BoolValue) *bool {
	if v == nil {
		return nil
	}
	b := v.Value
	return &b
}

// unwrap Timestamp -> *time.Time
func tsPtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

// wrap *time.Time -> *Timestamp (может быть nil)
func tsWrap(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

// wrap *string -> *StringValue (может быть nil)
func strWrap(s *string) *wrapperspb.StringValue {
	if s == nil {
		return nil
	}
	return &wrapperspb.StringValue{Value: *s}
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	// если уже статус — не трогаем
	if s, ok := status.FromError(err); ok && s.Code() != codes.OK {
		return err
	}

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
