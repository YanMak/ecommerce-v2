package postgres

import (
	"context"
	"time"

	pgxerr "github.com/YanMak/ecommerce/v2/pkg/errkit/pgx"
	"github.com/YanMak/ecommerce/v2/pkg/paging"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit"
	"github.com/YanMak/ecommerce/v2/services/telemetry/internal/app/contracts"
	"github.com/YanMak/ecommerce/v2/services/telemetry/internal/app/repoports"
	"github.com/YanMak/ecommerce/v2/services/telemetry/internal/dbgen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// repo реализует repoports.CertificatesRepo поверх sqlc/dbgen.
type certificatesRepo struct {
	q *dbgen.Queries
	// Правила пагинации для CRM (можно переопределить конструктором)
	defaultPerPage int32
	maxPerPage     int32
}

func NewCertificatesRepo(db dbgen.DBTX) repoports.CertificatesRepo {
	return &certificatesRepo{
		q:              dbgen.New(db),
		defaultPerPage: 20,
		maxPerPage:     200,
	}
}

func (r *certificatesRepo) Search(ctx context.Context, f contracts.SearchFilter, p paging.OffsetParams) ([]contracts.CertificateRow, int64, bool, error) {
	norm := paging.NormalizeOffset(p, paging.OffsetOpts{DefaultPerPage: r.defaultPerPage, MaxPerPage: r.maxPerPage})

	params := dbgen.SearchCertificatesParams{
		Q:           pgkit.OptText(f.Q),
		Inn:         pgkit.OptText(f.Inn),
		CreatedFrom: pgkit.OptTimestamptz(f.CreatedFrom),
		CreatedTo:   pgkit.OptTimestamptz(f.CreatedTo),
		UpdatedFrom: pgkit.OptTimestamptz(f.UpdatedFrom),
		UpdatedTo:   pgkit.OptTimestamptz(f.UpdatedTo),
		CategoryID:  pgkit.OptInt8(f.CategoryID),
		Opened:      pgkit.OptBool(f.Opened),
		Limit:       norm.PerPage,
		Offset:      norm.Offset,
	}

	rows, err := r.q.SearchCertificates(ctx, params)
	if err != nil {
		return nil, 0, false, pgxerr.Map(err)
	}

	total, err := r.q.CountCertificates(ctx, dbgen.CountCertificatesParams{
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
		return nil, 0, false, pgxerr.Map(err)
	}

	out := make([]contracts.CertificateRow, 0, len(rows))
	for _, rrow := range rows {
		var ufNum *string
		if rrow.UfNumber.Valid {
			s := rrow.UfNumber.String
			ufNum = &s
		}
		var ufInn *string
		if rrow.UfInn.Valid {
			s := rrow.UfInn.String
			ufInn = &s
		}
		var created *time.Time
		if rrow.CreatedTime.Valid {
			t := rrow.CreatedTime.Time
			created = &t
		}
		var updated *time.Time
		if rrow.UpdatedTime.Valid {
			t := rrow.UpdatedTime.Time
			updated = &t
		}

		out = append(out, contracts.CertificateRow{
			ID:          rrow.ID,
			Title:       rrow.Title,
			UfNumber:    ufNum,
			UfInn:       ufInn,
			CategoryID:  rrow.CategoryID,
			Opened:      rrow.Opened,
			CreatedTime: created,
			UpdatedTime: updated,
		})
	}

	hasNext := int64(norm.Offset)+int64(len(out)) < total
	return out, total, hasNext, nil
}

func (r *certificatesRepo) UpsertDocument(ctx context.Context, d contracts.UpsertDocument) error {
	return r.q.UpsertDocument(ctx, dbgen.UpsertDocumentParams{
		ID:            d.ID,
		CertificateID: d.CertificateID,
		Url:           d.URL,
		UrlMachine:    d.URLMachine,
	})
}

func (r *certificatesRepo) UpsertCertificateMin(ctx context.Context, in contracts.UpsertCertificateMin) error {
	now := time.Now()

	var ufUUID pgtype.UUID
	if in.UfUUID != "" {
		if u, err := uuid.Parse(in.UfUUID); err == nil {
			ufUUID = pgtype.UUID{Bytes: u, Valid: true}
		}
	}

	// ВАЖНО: не-null поля обязательно задаём.
	// Остальные — в NULL (pgtype.* с Valid=false) или пустые массивы.
	params := dbgen.UpsertCertificateParams{
		ID:           in.ID,
		Title:        in.Title,
		CategoryID:   in.CategoryID,
		Opened:       in.Opened,
		EntityTypeID: in.EntityTypeID,

		// времена (NOT NULL):
		CreatedTime: pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedTime: pgtype.Timestamptz{Time: now, Valid: true},
		MovedTime:   pgtype.Timestamptz{Time: now, Valid: true},

		// act/by: по схеме NOT NULL DEFAULT 0 — явно кладём 0
		CreatedBy: 0,
		UpdatedBy: 0,
		MovedBy:   0,

		// обязательные NOT NULL массивы — пустые, не nil
		Observers:  []int32{},
		ContactIds: []int32{},

		// опционалки — как NULL (zero value pgtype.* => Valid=false)
		// XmlID, PreviousStageID, Begindate, Closedate, CompanyID, ContactID, Opportunity,
		// TaxValue, CurrencyID, OpportunityAccount, TaxValueAccount, AccountCurrencyID,
		// MycompanyID, SourceID, SourceDescription, WebformID, UfInn, UfCompanyName, UfNumber,
		// UfStartDate, UfContractDate, UfEndDate, UfStatus, UfIdsDocuments, AssignedByID,
		// LastActivityBy, LastActivityTime, UtmSource, UtmMedium, UtmCampaign, UtmContent, UtmTerm
		// — оставляем по умолчанию (NULL).

		// флаги (NOT NULL) — задаём явно:
		IsManualOpportunity: false,

		// uuid:
		UfUuid: ufUUID,
	}

	return r.q.UpsertCertificate(ctx, params)
}
