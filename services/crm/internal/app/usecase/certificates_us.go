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

	crmmetrics "github.com/YanMak/ecommerce/v2/services/crm/internal/metrics"
)

type CertificatesUC struct {
	pool *pgxpool.Pool
	m    *crmmetrics.CRM
}

func NewCertificatesUC(pool *pgxpool.Pool, crmM *crmmetrics.CRM) *CertificatesUC {
	return &CertificatesUC{pool: pool, m: crmM}
}

func (uc *CertificatesUC) Search(
	ctx context.Context,
	f contracts.SearchFilter,
	p paging.OffsetParams,
) (rows []contracts.CertificateRow, total int64, hasNext bool, err error) {

	log := tlog.FromContext(ctx) // достали *zap.Logger из контекста
	start := time.Now()

	if dl, ok := ctx.Deadline(); ok {
		log.Info("crm.rpc.deadline", zap.Duration("left", time.Until(dl)))
	}

	// функция, которую будем ретраить
	run := func(ctx context.Context) error {
		return tx.InTx(ctx, uc.pool, func(ctx context.Context, dbtx pgx.Tx) error {
			r := repo.NewCertificatesRepo(dbtx)

			// imitationf of operation timeout
			//ctxOp, _ := context.WithTimeout(ctx, 1*time.Nanosecond)

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
		//retry.WithMaxDelay(2*time.Second),
		retry.WithMaxDelay(10*time.Second),
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
	durSeconds := dur.Seconds()

	if err != nil {
		// итоговый фейл
		log.Error("search_failed",
			zap.String("op", "crm.search"),
			zap.Duration("duration", dur),
			zap.Error(err),
		)

		uc.m.SearchTotal.WithLabelValues("error").Inc()
		uc.m.SearchDuration.WithLabelValues("error").Observe(durSeconds)

		return nil, 0, false, err
	}

	// успех
	uc.m.SearchTotal.WithLabelValues("ok").Inc()
	uc.m.SearchDuration.WithLabelValues("ok").Observe(durSeconds)

	log.Info("search_ok",
		zap.String("op", "crm.search"),
		zap.Duration("duration", dur),
		zap.Int("rows", len(rows)),
		zap.Int64("total", total),
		zap.Bool("has_next", hasNext),
	)

	return rows, total, hasNext, nil
}
