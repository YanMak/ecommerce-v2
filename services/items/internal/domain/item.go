// Layer: Domain (Entity)
package domain

import "time"

// Наша модель предметной области (без sqlc/pgx/HTTP).
type Item struct {
	ID          int64
	Slug        string
	Name        string
	Description string
	PriceCents  int64
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
