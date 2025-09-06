package pgtest

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrWouldBlock возвращается, если строка уже заблокирована (SQLSTATE 55P03).
var ErrWouldBlock = errors.New("row is locked (would block)")

// StartRowLock берет SELECT ... FOR UPDATE и держит блокировку до вызова release().
// ВАЖНО: table должен быть литералом из теста (не пользовательский ввод).
func StartRowLock(t *testing.T, db DBTX, table string, id any) (release func(context.Context)) {
	t.Helper()

	// начинаем "ручную" транзакцию, чтобы держать лок
	ctx := context.Background()
	// db может быть пулом — создадим тут свою tx
	var tx pgx.Tx
	switch d := db.(type) {
	case interface {
		Begin(context.Context) (pgx.Tx, error)
	}:
		var err error
		tx, err = d.Begin(ctx)
		if err != nil {
			t.Fatalf("begin lock tx: %v", err)
		}
	default:
		t.Fatalf("StartRowLock: db must support Begin(ctx) (got %T)", db)
	}

	sql := fmt.Sprintf("SELECT 1 FROM %s WHERE id = $1 FOR UPDATE", table)
	if _, err := tx.Exec(ctx, sql, id); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("lock row in %s: %v", table, err)
	}
	return func(ctx context.Context) { _ = tx.Rollback(ctx) }
}

// TryLockRowNowait пытается взять SELECT ... FOR UPDATE NOWAIT.
// Если блокировка занята — вернет ErrWouldBlock.
func TryLockRowNowait(ctx context.Context, db DBTX, table string, id any) error {
	sql := fmt.Sprintf("SELECT 1 FROM %s WHERE id = $1 FOR UPDATE NOWAIT", table)
	if _, err := db.Exec(ctx, sql, id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "55P03" { // lock_not_available
			return ErrWouldBlock
		}
		return err
	}
	return nil
}
