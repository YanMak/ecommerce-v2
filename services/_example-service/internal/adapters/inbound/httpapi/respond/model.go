package respond

// Problem — модель ошибки для Swagger (Problem-style).
// Используется в @Failure 400 {object} respond.Problem
type Problem struct {
	Code   string       `json:"code"  example:"invalid"`
	Error  string       `json:"error" example:"validation failed"`
	Errors []FieldError `json:"errors,omitempty"`
}
