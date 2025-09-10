package render

type Violation struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

type ErrorBody struct {
	Code           string      `json:"code,omitempty"`
	Message        string      `json:"message"`
	RequestID      string      `json:"request_id,omitempty"`
	IdempotencyKey string      `json:"idempotency_key,omitempty"`
	Violations     []Violation `json:"violations,omitempty"`
}
