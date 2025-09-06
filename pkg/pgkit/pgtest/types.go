package pgtest

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX — мини-интерфейс для Exec (совместим и с pgx.Tx, и с *pgxpool.Pool).
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Tx — алиас интерфейса транзакции (достаточно для тестов).
type Tx interface {
	DBTX
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
