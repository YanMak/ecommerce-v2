package contracts

// Минимальный набор полей для апсерта сертификата.
type UpsertCertificateMin struct {
	ID           int64 // внешний ID из Bitrix
	Title        string
	CategoryID   int64
	Opened       bool
	EntityTypeID int64
	UfUUID       string // строка UUID, например "6bdc0d56-20de-11f0-8268-000c295259c6"
}
