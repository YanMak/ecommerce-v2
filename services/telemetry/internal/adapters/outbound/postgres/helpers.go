package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// безопасно достаём time.Time
func ToTime(ts pgtype.Timestamptz) time.Time {
	if ts.Valid {
		return ts.Time
	}
	return time.Time{} // нулевое время, если внезапно NULL
}

// переводчик для опционального текста
func OptTime(tm *time.Time) pgtype.Timestamptz {
	if tm == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *tm, Valid: true}
}

// переводчик для опционального текста
func OptText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// переводчик для опционального int64
func OptInt8(n *int64) pgtype.Int8 {
	if n == nil {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: *n, Valid: true}
}
