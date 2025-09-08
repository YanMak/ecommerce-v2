package usecase

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YanMak/ecommerce/v2/pkg/errkit"
	"github.com/YanMak/ecommerce/v2/pkg/paging"
	"github.com/YanMak/ecommerce/v2/pkg/pgkit/tx"
	"github.com/YanMak/ecommerce/v2/pkg/retry"
	"github.com/YanMak/ecommerce/v2/pkg/telemetry/logger"
	"github.com/YanMak/ecommerce/v2/pkg/telemetry/metrics"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/adapters/outbound/postgres"
	"github.com/YanMak/ecommerce/v2/services/crm/internal/app/repoports"
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
	f repoports.SearchFilter,
	p paging.OffsetParams,
) (rows []repoports.CertificateRow, total int64, hasNext bool, err error) {

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
			logger.L.Error(ctx, err, "retry attempt", "op", "crm.search", "attempt", attempt)
			metrics.M.RetryAttempt(ctx, "crm.search", attempt, err)
		}),
	)

	return
}
