package dbgen_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/YanMak/ecommerce/v2/pkg/pgkit"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit/pgtest"
	"github.com/YanMak/ecommerce/v2/pkg/ptr"
	"github.com/YanMak/ecommerce/v2/services/telemetry/internal/dbgen"
)

func Test_UpsertAndGetCertificate_Minimal(t *testing.T) {
	t.Setenv("CRM_DB_URL", "postgres://postgres:postgres@localhost:5432/crm?sslmode=disable")
	pool := pgtest.PoolFromEnv(t, "CRM_DB_URL")

	pgtest.WithRollback(t, pool, func(ctx context.Context, tx pgtest.Tx) {
		q := dbgen.New(tx)
		now := time.Now().UTC()

		p := dbgen.UpsertCertificateParams{
			ID:              1001,
			XmlID:           pgkit.OptText(nil),
			Title:           "Test Certificate",
			CreatedBy:       1,
			UpdatedBy:       1,
			MovedBy:         1,
			CreatedTime:     pgkit.OptTimestamptz(ptr.To(now)),
			UpdatedTime:     pgkit.OptTimestamptz(ptr.To(now)),
			MovedTime:       pgkit.OptTimestamptz(ptr.To(now)),
			CategoryID:      1,
			Opened:          true,
			PreviousStageID: pgkit.OptText(nil),
			Begindate:       pgkit.OptTimestamptz(nil),
			Closedate:       pgkit.OptTimestamptz(nil),
			CompanyID:       pgkit.OptInt8(nil),
			ContactID:       pgkit.OptInt8(nil),

			// numeric/uuid опциональные — оставим NULL
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

			// эти заполним, чтобы потом удобно искать/проверять
			UfInn:         pgkit.OptText(ptr.To("0000000000")),
			UfCompanyName: pgkit.OptText(ptr.To("ACME LLC")),
			UfNumber:      pgkit.OptText(ptr.To("T-001")),

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

			// массивы NOT NULL — передаём пустые слайсы
			Observers:    []int32{},
			ContactIds:   []int32{},
			EntityTypeID: 131,
		}

		if err := q.UpsertCertificate(ctx, p); err != nil {
			t.Fatalf("UpsertCertificate: %v", err)
		}

		got, err := q.GetCertificate(ctx, p.ID)
		if err != nil {
			t.Fatalf("GetCertificate: %v", err)
		}

		if got.Title != p.Title {
			t.Fatalf("title mismatch: want %q, got %q", p.Title, got.Title)
		}
		if got.Opened != p.Opened {
			t.Fatalf("opened mismatch: want %v, got %v", p.Opened, got.Opened)
		}
		if got.EntityTypeID != p.EntityTypeID {
			t.Fatalf("entity_type_id mismatch: want %d, got %d", p.EntityTypeID, got.EntityTypeID)
		}
	})
}
