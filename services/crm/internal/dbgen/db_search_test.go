package dbgen_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/YanMak/ecommerce/v2/pkg/pgkit"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit/pgtest"
	"github.com/YanMak/ecommerce/v2/pkg/ptr"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/dbgen"
)

func Test_SearchCertificates_InsertThenFind(t *testing.T) {
	t.Setenv("CRM_DB_URL", "postgres://postgres:postgres@localhost:5432/crm?sslmode=disable")
	pool := pgtest.PoolFromEnv(t, "CRM_DB_URL")

	pgtest.WithRollback(t, pool, func(ctx context.Context, tx pgtest.Tx) {
		q := dbgen.New(tx)

		now := time.Now().UTC().Truncate(time.Second)

		// 1) вставляем одну запись (минимально необходимые поля)
		p := dbgen.UpsertCertificateParams{
			ID:              1002,
			XmlID:           pgkit.OptText(nil),
			Title:           "Searchable Certificate",
			CreatedBy:       1,
			UpdatedBy:       1,
			MovedBy:         1,
			CreatedTime:     pgkit.OptTimestamptz(ptr.To(now)),
			UpdatedTime:     pgkit.OptTimestamptz(ptr.To(now)),
			MovedTime:       pgkit.OptTimestamptz(ptr.To(now)),
			CategoryID:      2,
			Opened:          false,
			PreviousStageID: pgkit.OptText(nil),
			Begindate:       pgkit.OptTimestamptz(nil),
			Closedate:       pgkit.OptTimestamptz(nil),
			CompanyID:       pgkit.OptInt8(nil),
			ContactID:       pgkit.OptInt8(nil),

			// numeric / uuid опциональные
			Opportunity:        pgtype.Numeric{},
			TaxValue:           pgtype.Numeric{},
			OpportunityAccount: pgtype.Numeric{},
			TaxValueAccount:    pgtype.Numeric{},
			CurrencyID:         pgkit.OptText(nil),
			AccountCurrencyID:  pgkit.OptText(nil),
			MycompanyID:        pgkit.OptInt8(nil),
			SourceID:           pgkit.OptText(nil),
			SourceDescription:  pgkit.OptText(nil),
			WebformID:          pgkit.OptInt8(nil),
			UfUuid:             pgtype.UUID{},

			// поля, по которым будем искать
			UfInn:         pgkit.OptText(ptr.To("1234567890")),
			UfCompanyName: pgkit.OptText(ptr.To("ACME LLC")),
			UfNumber:      pgkit.OptText(ptr.To("T-SEARCH-001")),

			UfStartDate:      pgkit.OptTimestamptz(nil),
			UfContractDate:   pgkit.OptTimestamptz(nil),
			UfEndDate:        pgkit.OptTimestamptz(nil),
			UfStatus:         pgkit.OptInt8(nil),
			UfIdsDocuments:   pgkit.OptText(nil),
			AssignedByID:     pgkit.OptInt8(nil),
			LastActivityBy:   pgkit.OptInt8(nil),
			LastActivityTime: pgkit.OptTimestamptz(nil),
			UtmSource:        pgkit.OptText(nil),
			UtmMedium:        pgkit.OptText(nil),
			UtmCampaign:      pgkit.OptText(nil),
			UtmContent:       pgkit.OptText(nil),
			UtmTerm:          pgkit.OptText(nil),

			Observers:    []int32{},
			ContactIds:   []int32{},
			EntityTypeID: 131,
		}
		if err := q.UpsertCertificate(ctx, p); err != nil {
			t.Fatalf("UpsertCertificate: %v", err)
		}

		// 2) поиск по фильтрам — ожидаем 1 строку с нашим ID
		params := dbgen.SearchCertificatesParams{
			Q:           pgkit.OptText(ptr.To("SEARCH")),
			Inn:         pgkit.OptText(ptr.To("1234567890")),
			CreatedFrom: pgkit.OptTimestamptz(ptr.To(now.Add(-1 * time.Minute))),
			CreatedTo:   pgkit.OptTimestamptz(ptr.To(now.Add(+1 * time.Minute))),
			UpdatedFrom: pgkit.OptTimestamptz(nil),
			UpdatedTo:   pgkit.OptTimestamptz(nil),
			CategoryID:  pgkit.OptInt8(ptr.To(int64(2))),
			Opened:      pgkit.OptBool(ptr.To(false)),
			Limit:       10,
			Offset:      0,
		}
		rows, err := q.SearchCertificates(ctx, params)
		if err != nil {
			t.Fatalf("SearchCertificates: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		if rows[0].ID != p.ID {
			t.Fatalf("expected id=%d, got %d", p.ID, rows[0].ID)
		}

		// 3) count с теми же фильтрами — ожидаем 1
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
		if cnt != 1 {
			t.Fatalf("expected count=1, got %d", cnt)
		}

		// 4) негатив по датам: created_to = now (исключительно) — должно быть 0
		params.CreatedTo = pgkit.OptTimestamptz(ptr.To(now))
		rows, err = q.SearchCertificates(ctx, params)
		if err != nil {
			t.Fatalf("SearchCertificates (narrow range): %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("expected 0 rows with CreatedTo=now, got %d", len(rows))
		}
	})
}
