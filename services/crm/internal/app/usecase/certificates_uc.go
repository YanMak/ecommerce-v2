package usecase

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/YanMak/ecommerce/v2/pkg/errkit"
	"github.com/YanMak/ecommerce/v2/pkg/paging"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit/tx"
	"github.com/YanMak/ecommerce/v2/pkg/telemetry/logger"
	"github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/adapters/outbound/postgres"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/app/contracts"

	"github.com/YanMak/ecommerce/v2/pkg/retry"
	tlog "github.com/YanMak/ecommerce/v2/pkg/telemetry/log"
)

type CertificatesUC struct {
	pool *pgxpool.Pool
}

func NewCertificatesUC(pool *pgxpool.Pool) *CertificatesUC {
	return &CertificatesUC{pool: pool}
}

// Search — read-only путь: каждая попытка выполняется в своей транзакции.
// Ретраим только Transient-ошибки (deadlock/serialization/сеть и т.п.).
func (uc *CertificatesUC) Search(
	ctx context.Context,
	f contracts.SearchFilter,
	p paging.OffsetParams,
) (rows []contracts.CertificateRow, total int64, hasNext bool, err error) {

	log := tlog.FromContext(ctx)

	// общий дедлайн на операцию (по желанию можно вынести в конфиг)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err = retry.Do(ctx, func(ctx context.Context) error {
		return tx.InTx(ctx, uc.pool, func(ctx context.Context, dbtx pgx.Tx) error {
			repo := postgres.NewCertificatesRepo(dbtx) // DBTX=tx
			var e error
			rows, total, hasNext, e = repo.Search(ctx, f, p)
			return e
		})
	}, retry.Only(errkit.IsTransient),
		retry.WithMaxAttempts(4),
		retry.WithBaseDelay(100*time.Millisecond),
		retry.WithMaxDelay(2*time.Second),
		retry.WithOnAttempt(func(ctx context.Context, attempt int, err error) {
			// Старая заглушка
			logger.L.Error(ctx, err, "retry attempt", "op", "crm.search", "attempt", attempt)
			metrics.M.RetryAttempt(ctx, "crm.search", attempt, err)

			//Новое добавление изучаем телеметрию
			log.Warn("retry_attempt",
				zap.Int("attempt", a.N),
				zap.Int("max_attempts", 3),
				zap.Float64("sleep_ms", a.NextDelay.Seconds()*1000),
				zap.Float64("elapsed_ms", a.Elapsed.Seconds()*1000),
				zap.Error(a.Err),
			)

		}),
	)

	return
}
