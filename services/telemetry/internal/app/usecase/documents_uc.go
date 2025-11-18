package usecase

import (
	"context"
	"time"

	"github.com/YanMak/ecommerce/v2/pkg/errkit"
	tx "github.com/YanMak/ecommerce/v2/pkg/pgkit/tx"
	"github.com/YanMak/ecommerce/v2/pkg/retry"
	repo "github.com/YanMak/ecommerce/v2/services/telemetry/internal/adapters/outbound/postgres"
	"github.com/YanMak/ecommerce/v2/services/telemetry/internal/app/contracts"
	"github.com/jackc/pgx/v5"

	tlog "github.com/YanMak/ecommerce/v2/pkg/telemetry/log"
	"github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics"
	"go.uber.org/zap"
)

func (uc *CertificatesUC) UpsertDocument(ctx context.Context, d contracts.UpsertDocument) (err error) {
	log := tlog.FromContext(ctx)

	// функция, которую будем ретраить
	run := func(ctx context.Context) error {
		return tx.InTx(ctx, uc.pool, func(ctx context.Context, dbtx pgx.Tx) error {
			r := repo.NewCertificatesRepo(dbtx)

			// imitationf of operation timeout
			//ctxOp, _ := context.WithTimeout(ctx, 1*time.Nanosecond)

			e := r.UpsertDocument(ctx, d)
			return e
		})
	}

	// Ретраим только транзиентные PG-ошибки (deadlock/serialization и т.п.)
	err = retry.Do(
		ctx,
		run,
		retry.Only(errkit.IsTransient),
		retry.WithMaxAttempts(3),
		retry.WithBaseDelay(100*time.Millisecond),
		retry.WithMaxDelay(2*time.Second),
		retry.WithOnAttempt(func(ctx context.Context, attempt int, err error) {
			log.Warn("upsert_document_retry",
				zap.String("op", "crm.upsert"),
				zap.Int("attempt", attempt),
				zap.Error(err),
			)
			metrics.M.RetryAttempt(ctx, "crm.upsert", attempt, err)
		}),
	)

	return err
}
