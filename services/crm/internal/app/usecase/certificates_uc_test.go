package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/YanMak/ecommerce/v2/pkg/paging"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit/pgtest"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit/tx"
	"github.com/YanMak/ecommerce/v2/pkg/ptr"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/app/repoports"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/app/usecase"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/dbgen"
)

func Test_CertificatesUC_Search(t *testing.T) {
	t.Setenv("CRM_DB_URL", "postgres://postgres:postgres@localhost:5432/crm?sslmode=disable")
	pool := pgtest.PoolFromEnv(t, "CRM_DB_URL")
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const id int64 = 31001

	// --- setup: вставляем одну запись и COMMIT ---
	if err := tx.InTx(ctx, pool, func(ctx context.Context, dbtx pgx.Tx) error {
		q := dbgen.New(dbtx)
		return q.UpsertCertificate(ctx, dbgen.UpsertCertificateParams{
			ID:              id,
			XmlID:           pgkit.OptText(nil),
			Title:           "UC Search Cert",
			CreatedBy:       1,
			UpdatedBy:       1,
			MovedBy:         1,
			CreatedTime:     pgkit.OptTimestamptz(ptr.To(now)),
			UpdatedTime:     pgkit.OptTimestamptz(ptr.To(now)),
			MovedTime:       pgkit.OptTimestamptz(ptr.To(now)),
			CategoryID:      5,
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

			UfInn:         pgkit.OptText(ptr.To("5555555555")),
			UfCompanyName: pgkit.OptText(ptr.To("Wayne Corp")),
			UfNumber:      pgkit.OptText(ptr.To("UC-TEST-001")),

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
		})
	}); err != nil {
		t.Fatalf("setup insert: %v", err)
	}

	uc := usecase.NewCertificatesUC(pool)

	rows, total, hasNext, err := uc.Search(ctx,
		repoports.SearchFilter{
			Q:           ptr.To("UC-TEST"),
			Inn:         ptr.To("5555555555"),
			CategoryID:  ptr.To(int64(5)),
			Opened:      ptr.To(false),
			CreatedFrom: ptr.To(now.Add(-time.Minute)),
			CreatedTo:   ptr.To(now.Add(+time.Minute)),
		},
		paging.OffsetParams{Page: 1, PerPage: 10},
	)
	if err != nil {
		t.Fatalf("uc.Search: %v", err)
	}
	if total != 1 || hasNext {
		t.Fatalf("want total=1, hasNext=false; got total=%d, hasNext=%v", total, hasNext)
	}
	if len(rows) != 1 || rows[0].ID != id {
		t.Fatalf("want 1 row with id=%d, got len=%d, id(if any)", id, len(rows))
	}

	// --- cleanup: удаляем запись (COMMIT) ---
	if err := tx.InTx(ctx, pool, func(ctx context.Context, dbtx pgx.Tx) error {
		_, err := dbtx.Exec(ctx, `DELETE FROM certificates WHERE id=$1`, id)
		return err
	}); err != nil {
		t.Fatalf("cleanup delete: %v", err)
	}
}
