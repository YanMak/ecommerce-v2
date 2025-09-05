// Layer: Outbound Adapter (Postgres + sqlc)
package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"

	repoports "example.com/sqlchello/internal/core/ports/repo"
	"example.com/sqlchello/internal/dbgen"
)

// Компилятор проверит, что Repo реализует ItemRepository.
var _ repoports.ItemRepository = (*Repo)(nil)

type Repo struct {
	pool *pgxpool.Pool
}

func (r *Repo) q(db dbgen.DBTX) *dbgen.Queries { return dbgen.New(db) }
func (r *Repo) Q(db dbgen.DBTX) *dbgen.Queries { return dbgen.New(db) }

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }
