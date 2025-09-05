package postgres

import (
	"context"
	"time"

	"github.com/YanMak/ecommerce/v2/services/crm/internal/dbgen"
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

func (r *Repo) InTx(ctx context.Context, fn func(q *dbgen.Queries) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := dbgen.New(tx)
	if err := fn(qtx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// // попытка выполнить fn в tx с ретраями на 40001/40P01
// func inTxRetry(ctx context.Context, pool *pgxpool.Pool, opts pgx.TxOptions, attempts int, fn func(q *dbgen.Queries) error) error {
// 	if attempts < 1 {
// 		attempts = 1
// 	}
// 	backoff := 50 * time.Millisecond

// 	for try := 1; try <= attempts; try++ {
// 		tx, err := pool.BeginTx(ctx, opts)
// 		if err != nil {
// 			return err
// 		}
// 		qtx := dbgen.New(tx)

// 		err = fn(qtx) // ваши sqlc-вызовы внутри tx
// 		if err == nil {
// 			err = tx.Commit(ctx)
// 		}
// 		if err == nil {
// 			return nil
// 		}

// 		_ = tx.Rollback(ctx) // безопасный откат

// 		var pgerr *pgconn.PgError
// 		if errors.As(err, &pgerr) && (pgerr.Code == "40001" || pgerr.Code == "40P01") && try < attempts {
// 			time.Sleep(backoff + time.Duration(rand.Intn(50))*time.Millisecond) // backoff+jitter
// 			backoff *= 2
// 			continue
// 		}
// 		return err
// 	}
// 	return fmt.Errorf("tx failed after %d attempts", attempts)
// }
