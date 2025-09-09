package postgres

import (
	"context"
	"time"

	pgxerr "github.com/YanMak/ecommerce/v2/pkg/errkit/pgx"
	"github.com/YanMak/ecommerce/v2/pkg/paging"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/app/contracts"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/app/repoports"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/dbgen"
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
