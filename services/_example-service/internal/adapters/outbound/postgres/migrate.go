package postgres

import (
	"context"
	"database/sql"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrate — запускает миграции из папки "migrations" (file://).
// Важно: запускаем ОДИН раз при старте сервера (dev), а в проде это обычно отдельный job.
func Migrate(ctx context.Context) error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}
	// golang-migrate использует database/sql драйвер (pgx stdlib)
	db, err := sql.Open("pgx", url)
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	// источник миграций — файловая папка (корень репо/migrations)
	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
