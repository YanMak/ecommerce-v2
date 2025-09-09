package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/YanMak/ecommerce/v2/pkg/paging"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit/pgtest"
	"github.com/YanMak/ecommerce/v2/pkg/ptr"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/adapters/outbound/postgres"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/app/contracts"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/dbgen"
)

func Test_CertificatesRepo_Search(t *testing.T) {
	t.Setenv("CRM_DB_URL", "postgres://postgres:postgres@localhost:5432/crm?sslmode=disable")
	pool := pgtest.PoolFromEnv(t, "CRM_DB_URL")

	pgtest.WithRollback(t, pool, func(ctx context.Context, tx pgtest.Tx) {
		q := dbgen.New(tx)

		now := time.Now().UTC().Truncate(time.Second)

		// вставляем запись напрямую через dbgen (тестируем только чтение репозитория)
		p := dbgen.UpsertCertificateParams{
			ID:              2001,
			XmlID:           pgkit.OptText(nil),
			Title:           "Repo Search Certificate",
			CreatedBy:       1,
			UpdatedBy:       1,
			MovedBy:         1,
			CreatedTime:     pgkit.OptTimestamptz(ptr.To(now)),
			UpdatedTime:     pgkit.OptTimestamptz(ptr.To(now)),
			MovedTime:       pgkit.OptTimestamptz(ptr.To(now)),
			CategoryID:      3,
			Opened:          false,
			PreviousStageID: pgkit.OptText(nil),
			Begindate:       pgkit.OptTimestamptz(nil),
			Closedate:       pgkit.OptTimestamptz(nil),
			CompanyID:       pgkit.OptInt8(nil),
			ContactID:       pgkit.OptInt8(nil),

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

			UfInn:         pgkit.OptText(ptr.To("9876543210")),
			UfCompanyName: pgkit.OptText(ptr.To("Globex LLC")),
			UfNumber:      pgkit.OptText(ptr.To("R-REPO-001")),

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

		repo := postgres.NewCertificatesRepo(tx)

		rows, total, hasNext, err := repo.Search(ctx,
			contracts.SearchFilter{
				Q:          ptr.To("REPO"),
				Inn:        ptr.To("9876543210"),
				CategoryID: ptr.To(int64(3)),
				Opened:     ptr.To(false),
				// даты можно опустить или задать окно now±1m:
				CreatedFrom: ptr.To(now.Add(-time.Minute)),
				CreatedTo:   ptr.To(now.Add(+time.Minute)),
			},
			paging.OffsetParams{Page: 1, PerPage: 10},
		)
		if err != nil {
			t.Fatalf("repo.Search: %v", err)
		}
		if total != 1 || hasNext {
			t.Fatalf("want total=1, hasNext=false; got total=%d, hasNext=%v", total, hasNext)
		}
		if len(rows) != 1 {
			t.Fatalf("want 1 row, got %d", len(rows))
		}
		if rows[0].ID != p.ID {
			t.Fatalf("id mismatch: want %d, got %d", p.ID, rows[0].ID)
		}
		if rows[0].UfNumber == nil || *rows[0].UfNumber != "R-REPO-001" {
			t.Fatalf("uf_number mismatch")
		}
		if rows[0].UfInn == nil || *rows[0].UfInn != "9876543210" {
			t.Fatalf("uf_inn mismatch")
		}
	})
}
