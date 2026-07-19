package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type PaymentRequest struct {
	UserId   string  `json:"user_id" validate:"required"`
	Amount   float64 `json:"amount" validate:"required,gt=0"`
	Currency string  `json:"currency" validate:"required,len=3"`
}

type PaymentResponse struct {
	TransactionId string    `json:"transaction_id"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"crated_at"`
}

type APIError struct {
	Status  int               `json:"status"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

const (
	CodeMalformedJson = "MALFORMED_JSON"
	CodeInternalError = "INTERNAL_SERVER_ERROR"
)

var messages = map[string]string{
	CodeMalformedJson: "El cuerpo de la petición no se pudo procesar porque el JSON está mal formado.",
	CodeInternalError: "Ha ocurrido un error inesperado.",
}

func GetMessage(code string) string {
	if msg, exists := messages[code]; exists {
		return msg
	}
	return messages[CodeInternalError]
}

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(struct {
			Message string
		}{Message: "Pong"})
	})
	r.Post("/v1/payments", func(w http.ResponseWriter, r *http.Request) {
		var req PaymentRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			ResponseWithError(w, http.StatusBadRequest, CodeMalformedJson, nil)
		}

		json.NewEncoder(w).Encode(struct {
			Message string
		}{Message: "pago realizado con éxito"})
	})
	http.ListenAndServe(":8080", r)
}

func ResponseWithError(w http.ResponseWriter, status int, errorCode string, details map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := APIError{
		Status:  status,
		Code:    errorCode,
		Message: GetMessage(errorCode),
		Details: details,
	}
	json.NewEncoder(w).Encode(resp)
}
