package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type PaymentRequest struct {
	AccountID string  `json:"user_id" validate:"required"`
	Amount    float64 `json:"amount" validate:"required,gt=0"`
	Currency  string  `json:"currency" validate:"required,len=3"`
}

type PaymentResponse struct {
	TransactionID string    `json:"transaction_id"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"crated_at"`
}

type Meta struct {
	TraceID   string `json:"trace_id,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}
type ErrorResponse struct {
	Status  string            `json:"status"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Meta    *Meta             `json:"meta,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

type SuccessResponse[T any] struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Meta    *Meta  `json:"meta,omitempty"`
	Data    T      `json:"data"`
}

type PaymentResponseDTO struct {
	ID        string `json:"payment_id"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

const (
	CodeMalformedJson   = "MALFORMED_JSON"
	CodeInternalError   = "INTERNAL_SERVER_ERROR"
	CodePaymentAccepted = "PAYMENT_ACCEPTED"
)

var errorMessages = map[string]string{
	CodeMalformedJson:   "El cuerpo de la petición no se pudo procesar porque el JSON está mal formado.",
	CodeInternalError:   "Ha ocurrido un error inesperado.",
	CodePaymentAccepted: "Payment request successfully validated and placed in queue",
}

var successMessages = map[string]string{}

func GetMessage(code string) string {
	if msg, exists := errorMessages[code]; exists {
		return msg
	}
	return errorMessages[CodeInternalError]
}

func main() {
	r := NewRouter()
	http.ListenAndServe(":8080", r)
}

func NewRouter() *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/ping", pingHandler)
	router.Post("/v1/payments", createPaymentHandler)
	return router
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(struct {
		Message string
	}{Message: "Pong"})
}

func createPaymentHandler(w http.ResponseWriter, r *http.Request) {
	var req PaymentRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ResponseWithError(w, http.StatusBadRequest, CodeMalformedJson, nil)
		return
	}

	dto := PaymentResponseDTO{}

	RespondWithSuccess(w, http.StatusAccepted, CodePaymentAccepted, dto)
}

func ResponseWithError(w http.ResponseWriter, statusCode int, errorCode string, details map[string]string) {
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

func RespondWithSuccess[T any](w http.ResponseWriter, statusCode int, successCode string, data T) {
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
