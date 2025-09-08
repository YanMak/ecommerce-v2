package pgx

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/YanMak/ecommerce/v2/pkg/errkit"
)

// Map переводит ошибки драйвера/контекста в категории errkit.
func Map(err error) error {
	if err == nil {
		return nil
	}
	// Контекст
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("%w: %v", errkit.ErrDeadline, err)
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("%w: %v", errkit.ErrCanceled, err)
	}

	// Ошибки Postgres
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		code := string(pgErr.Code)

		// Retryable / transient
		switch code {
		case "40001", // serialization_failure
			"40P01",                   // deadlock_detected
			"55P03",                   // lock_not_available
			"08000", "08003", "08006", // connection_exception
			"57P01", // admin_shutdown
			"53300": // too_many_connections
			return fmt.Errorf("%w: %v", errkit.ErrTransient, err)
		}

		// Конфликты/валидация
		switch code {
		case "23505": // unique_violation
			return fmt.Errorf("%w: %v", errkit.ErrConflict, err)
		case "23502", // not_null_violation
			"23503", // foreign_key_violation
			"22001": // string_data_right_truncation
			return fmt.Errorf("%w: %v", errkit.ErrInvalid, err)
		}

		// Технические/миграционные — отнесём к internal
		return fmt.Errorf("%w: %v", errkit.ErrInternal, err)
	}

	// Всё прочее
	return fmt.Errorf("%w: %v", errkit.ErrInternal, err)
}
