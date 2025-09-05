package postgres

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/YanMak/ecommerce/v2/services/crm/internal/dbgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type TxExtra struct {
	Retries     int           // сколько попыток всего (включая первую), обычно 2–3
	BaseBackoff time.Duration // стартовая пауза перед 2-й попыткой (напр. 50ms)
	MaxBackoff  time.Duration // потолок бэкоффа (напр. 1s)
	StmtTimeout time.Duration // SET LOCAL statement_timeout
	LockTimeout time.Duration // SET LOCAL lock_timeout
	IdleInTx    time.Duration // SET LOCAL idle_in_transaction_session_timeout
}

func (r *Repo) InTxRetry(
	ctx context.Context,
	//pool *pgxpool.Pool,
	pgxOpts pgx.TxOptions, // IsoLevel/ReadOnly/Deferrable — стандартный pgx
	extra TxExtra, // доп.опции
	rng *rand.Rand,
	fn func(ctx context.Context, q *dbgen.Queries) error, // ваши sqlc шаги
) error {
	if extra.Retries < 1 {
		extra.Retries = 1
	}
	if extra.BaseBackoff <= 0 {
		extra.BaseBackoff = 50 * time.Millisecond
	}
	if extra.MaxBackoff <= 0 {
		extra.MaxBackoff = time.Second
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	backoff := extra.BaseBackoff

	for try := 1; try <= extra.Retries; try++ {
		tx, err := r.pool.BeginTx(ctx, pgxOpts)
		if err != nil {
			return err
		}
		asMS := func(d time.Duration) string { return fmt.Sprintf("%dms", d/time.Millisecond) }
		if extra.StmtTimeout > 0 {
			if _, err := tx.Exec(ctx, "SELECT set_config('statement_timeout', $1, true)", asMS(extra.StmtTimeout)); err != nil {
				return err
			}
		}
		if extra.LockTimeout > 0 {
			if _, err := tx.Exec(ctx, "SELECT set_config('lock_timeout', $1, true)", asMS(extra.LockTimeout)); err != nil {
				return err
			}
		}
		if extra.IdleInTx > 0 {
			if _, err := tx.Exec(ctx, "SELECT set_config('idle_in_transaction_session_timeout', $1, true)", asMS(extra.IdleInTx)); err != nil {
				return err
			}
		}

		qtx := dbgen.New(tx)
		err = fn(ctx, qtx)
		if err == nil {
			err = tx.Commit(ctx) // commit тоже может дать 40001/40P01
		}
		if err == nil {
			return nil
		}
		_ = tx.Rollback(ctx) // безопасный откат

		// решаем — ретраить ли
		if !isRetriablePgErr(err) || try == extra.Retries {
			return err
		}

		// backoff + jitter, уважая дедлайн контекста
		//sleep := backoff + time.Duration(rand.Intn(50))*time.Millisecond
		sleep := backoff + time.Duration(rng.Intn(50))*time.Millisecond

		if sleep > extra.MaxBackoff {
			sleep = extra.MaxBackoff
		}
		if dl, ok := ctx.Deadline(); ok {
			remain := time.Until(dl)
			if remain <= 0 {
				return ctx.Err()
			}
			if sleep > remain {
				sleep = remain
			}
		}
		select {
		case <-time.After(sleep):
		case <-ctx.Done():
			return ctx.Err()
		}
		if backoff < extra.MaxBackoff {
			backoff *= 2
			if backoff > extra.MaxBackoff {
				backoff = extra.MaxBackoff
			}
		}
	}
	return fmt.Errorf("unreachable")
}

func isRetriablePgErr(err error) bool {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return false
	}
	switch pg.Code {
	case "40001", // serialization_failure
		"40P01", // deadlock_detected
		"55P03": // lock_not_available (lock timeout)
		return true
	}
	return false
}
