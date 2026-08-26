package response

type Meta struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id"`
	TraceID   string `json:"trace_id"`
	Version   string `json:"version,omitempty"`
}

type SuccessResponse[T any] struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Meta    *Meta  `json:"meta"`
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
	Meta    *Meta         `json:"meta"`
	Details []ErrorDetail `json:"details,omitempty"`
}
