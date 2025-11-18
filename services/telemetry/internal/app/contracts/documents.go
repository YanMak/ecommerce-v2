package contracts

type UpsertDocument struct {
	ID            int64
	CertificateID int64
	URL           string
	URLMachine    string
}
