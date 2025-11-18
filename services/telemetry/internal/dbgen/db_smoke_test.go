package dbgen_test

import (
	"context"
	"testing"

	"github.com/YanMak/ecommerce/v2/pkg/pgkit"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit/pgtest"
	"github.com/YanMak/ecommerce/v2/pkg/ptr"
	"github.com/YanMak/ecommerce/v2/services/telemetry/internal/dbgen"
)

// 1) Базовый "ничего не найдено" с заведомо левыми фильтрами.
func Test_SearchCertificates_NoMatch(t *testing.T) {
	t.Setenv("CRM_DB_URL", "postgres://postgres:postgres@localhost:5432/crm?sslmode=disable")
	pool := pgtest.PoolFromEnv(t, "CRM_DB_URL")

	pgtest.WithRollback(t, pool, func(ctx context.Context, tx pgtest.Tx) {
		q := dbgen.New(tx)

		params := dbgen.SearchCertificatesParams{
			Q:           pgkit.OptText(ptr.To("___NO_SUCH_NUMBER___")),
			Inn:         pgkit.OptText(ptr.To("___NO_SUCH_INN___")),
			CreatedFrom: pgkit.OptTimestamptz(nil),
			CreatedTo:   pgkit.OptTimestamptz(nil),
			UpdatedFrom: pgkit.OptTimestamptz(nil),
			UpdatedTo:   pgkit.OptTimestamptz(nil),
			CategoryID:  pgkit.OptInt8(nil),
			Opened:      pgkit.OptBool(nil),
			Limit:       5,
			Offset:      0,
		}

		rows, err := q.SearchCertificates(ctx, params)
		if err != nil {
			t.Fatalf("SearchCertificates: %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("expected 0 rows, got %d", len(rows))
		}

		cnt, err := q.CountCertificates(ctx, dbgen.CountCertificatesParams{
			Q:           params.Q,
			Inn:         params.Inn,
			CreatedFrom: params.CreatedFrom,
			CreatedTo:   params.CreatedTo,
			UpdatedFrom: params.UpdatedFrom,
			UpdatedTo:   params.UpdatedTo,
			CategoryID:  params.CategoryID,
			Opened:      params.Opened,
		})
		if err != nil {
			t.Fatalf("CountCertificates: %v", err)
		}
		if cnt != 0 {
			t.Fatalf("expected count=0, got %d", cnt)
		}
	})
}

// 2) Пустые фильтры (все NULL) + проверка limit/offset = 0 результатов и корректная компиляция типов.
func Test_SearchCertificates_EmptyFilters_CompileAndRun(t *testing.T) {
	t.Setenv("CRM_DB_URL", "postgres://postgres:postgres@localhost:5432/crm?sslmode=disable")
	pool := pgtest.PoolFromEnv(t, "CRM_DB_URL")

	pgtest.WithRollback(t, pool, func(ctx context.Context, tx pgtest.Tx) {
		q := dbgen.New(tx)

		params := dbgen.SearchCertificatesParams{
			Q:           pgkit.OptText(nil),
			Inn:         pgkit.OptText(nil),
			CreatedFrom: pgkit.OptTimestamptz(nil),
			CreatedTo:   pgkit.OptTimestamptz(nil),
			UpdatedFrom: pgkit.OptTimestamptz(nil),
			UpdatedTo:   pgkit.OptTimestamptz(nil),
			CategoryID:  pgkit.OptInt8(nil),
			Opened:      pgkit.OptBool(nil),
			Limit:       1,
			Offset:      10, // при пустой БД просто вернёт 0 строк
		}

		if _, err := q.SearchCertificates(ctx, params); err != nil {
			t.Fatalf("SearchCertificates(empty): %v", err)
		}
	})
}
