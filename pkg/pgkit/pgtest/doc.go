// Package pgtest содержит общие утилиты для Postgres-тестов (pgx/v5):
// - PoolFromEnv(t, "SERVICE_DB_URL")
// - BeginTx / WithRollback
// - StartRowLock / TryLockRowNowait (для сценариев конкурентного доступа)
//
// Используйте PoolFromEnv в каждом тесте вместо глобального пула.
package pgtest
