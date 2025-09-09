package grpcx

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// ---------- Wrap: *T -> protobuf ----------

func S(s *string) *wrapperspb.StringValue {
	if s == nil {
		return nil
	}
	return wrapperspb.String(*s)
}

func I64(i *int64) *wrapperspb.Int64Value {
	if i == nil {
		return nil
	}
	return wrapperspb.Int64(*i)
}

func B(b *bool) *wrapperspb.BoolValue {
	if b == nil {
		return nil
	}
	return wrapperspb.Bool(*b)
}

func TS(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

// ---------- Unwrap: protobuf -> *T ----------

func SPtr(v *wrapperspb.StringValue) *string {
	if v == nil {
		return nil
	}
	s := v.Value
	return &s
}

func I64Ptr(v *wrapperspb.Int64Value) *int64 {
	if v == nil {
		return nil
	}
	x := v.Value
	return &x
}

func BPtr(v *wrapperspb.BoolValue) *bool {
	if v == nil {
		return nil
	}
	x := v.Value
	return &x
}

func TSPtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}
