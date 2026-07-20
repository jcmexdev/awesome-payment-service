package response

type Meta struct {
	RequestID string `json:"request_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
}

type SuccessResponse[T any] struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Meta    *Meta  `json:"meta,omitempty"`
	Data    T      `json:"data"`
}

type ErrorDetail struct {
	Field  string `json:"field,omitempty"`
	Reason string `json:"reason"`
}

type ErrorResponse struct {
	Status  string        `json:"status"`
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Meta    *Meta         `json:"meta,omitempty"`
	Details []ErrorDetail `json:"details,omitempty"`
}
