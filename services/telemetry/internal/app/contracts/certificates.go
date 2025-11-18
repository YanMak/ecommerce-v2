package contracts

import "time"

// Входные фильтры (чистые Go-типы)
type SearchFilter struct {
	Q           *string
	Inn         *string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	CategoryID  *int64
	Opened      *bool
}

// Read-модель строки (без pgtype)
type CertificateRow struct {
	ID          int64
	Title       string
	UfNumber    *string
	UfInn       *string
	CategoryID  int64
	Opened      bool
	CreatedTime *time.Time
	UpdatedTime *time.Time
}
