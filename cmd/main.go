package main

import (
	"net/http"

	router "github.com/jcmexdev/payment-service/internal/infra/http"
)

func main() {
	r := router.NewRouter()
	http.ListenAndServe(":8080", r)
}
