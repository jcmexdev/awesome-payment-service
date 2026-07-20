package response

import (
	"encoding/json"
	"net/http"
)

func SendError(w http.ResponseWriter, statusCode int, errorCode string, details []ErrorDetail) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := ErrorResponse{
		Status:  "error",
		Code:    errorCode,
		Message: GetMessage(errorCode),
		Details: details,
	}
	json.NewEncoder(w).Encode(resp)
}

func SendSuccess[T any](w http.ResponseWriter, statusCode int, successCode string, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := SuccessResponse[T]{
		Status:  "success",
		Code:    successCode,
		Message: GetMessage(successCode),
		Data:    data,
	}

	json.NewEncoder(w).Encode(resp)
}
