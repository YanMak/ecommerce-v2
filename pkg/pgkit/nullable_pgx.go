// pkg/pgkit/nullable_pgx.go
package pgkit

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func OptText(p *string) pgtype.Text {
	if p == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *p, Valid: true}
}

func OptInt8(p *int64) pgtype.Int8 {
	if p == nil {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: *p, Valid: true}
}

func OptBool(p *bool) pgtype.Bool {
	if p == nil {
		return pgtype.Bool{Valid: false}
	}
	return pgtype.Bool{Bool: *p, Valid: true}
}

func OptTimestamptz(p *time.Time) pgtype.Timestamptz {
	if p == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *p, Valid: true}
}

// Обратные преобразования (удобно в мапперах):
func TextPtr(n pgtype.Text) *string {
	if !n.Valid {
		return nil
	}
	s := n.String
	return &s
}
