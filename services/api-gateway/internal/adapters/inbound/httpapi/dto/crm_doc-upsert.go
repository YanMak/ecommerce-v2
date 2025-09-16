package dto

type UpsertDocumentDTO struct {
	ID            int64  `json:"id"`
	CertificateID int64  `json:"certificate_id"`
	URL           string `json:"url"`
	URLMachine    string `json:"url_machine"`
}
