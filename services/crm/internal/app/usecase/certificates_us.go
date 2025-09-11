package usecase

import (
	"context"
	"time"

	"github.com/YanMak/ecommerce/v2/pkg/errkit"
	"github.com/YanMak/ecommerce/v2/pkg/paging"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit/tx"
	"github.com/YanMak/ecommerce/v2/pkg/retry"
	repo "github.com/YanMak/ecommerce/v2/services/crm/internal/adapters/outbound/postgres"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/app/contracts"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	tlog "github.com/YanMak/ecommerce/v2/pkg/telemetry/log"
	"github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics"
	"go.uber.org/zap"
)

type CertificatesUC struct {
	pool *pgxpool.Pool
}

func NewCertificatesUC(pool *pgxpool.Pool) *CertificatesUC {
	return &CertificatesUC{pool: pool}
}

func (uc *CertificatesUC) Search(
	ctx context.Context,
	f contracts.SearchFilter,
	p paging.OffsetParams,
) (rows []contracts.CertificateRow, total int64, hasNext bool, err error) {

	log := tlog.FromContext(ctx) // достали *zap.Logger из контекста
	start := time.Now()

	// функция, которую будем ретраить
	run := func(ctx context.Context) error {
		return tx.InTx(ctx, uc.pool, func(ctx context.Context, dbtx pgx.Tx) error {
			r := repo.NewCertificatesRepo(dbtx)

			var e error
			rows, total, hasNext, e = r.Search(ctx, f, p)
			return e
		})
	}

	// ретраи: 3 попытки, логируем каждую неудачу
	err = retry.Do(
		ctx,
		run,
		retry.Only(errkit.IsTransient),
		retry.WithMaxAttempts(3),
		retry.WithBaseDelay(100*time.Millisecond),
		retry.WithMaxDelay(2*time.Second),
		retry.WithOnAttempt(func(ctx context.Context, attempt int, err error) {
			// сработает для attempt=1,2,... на каждую ошибку перед следующей попыткой
			log.Warn("retry_attempt",
				zap.String("op", "crm.search"),
				zap.Int("attempt", attempt),
				zap.Error(err),
			)
			// если у тебя есть доменные метрики ретраев — зови их здесь
			// metrics.M.RetryAttempt(ctx, "crm.search", attempt, err)
			metrics.M.RetryAttempt(ctx, "crm.search", attempt, err)
		}),
	)

	dur := time.Since(start)

	if err != nil {
		// итоговый фейл
		log.Error("search_failed",
			zap.String("op", "crm.search"),
			zap.Duration("duration", dur),
			zap.Error(err),
		)
		return nil, 0, false, err
	}

	// успех
	log.Info("search_ok",
		zap.String("op", "crm.search"),
		zap.Duration("duration", dur),
		zap.Int("rows", len(rows)),
		zap.Int64("total", total),
		zap.Bool("has_next", hasNext),
	)

	return rows, total, hasNext, nil
}
