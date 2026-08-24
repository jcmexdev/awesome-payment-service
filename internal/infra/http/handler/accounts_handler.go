package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jcmexdev/payment-service/internal/domain"
	"github.com/jcmexdev/payment-service/internal/domain/errors"
	"github.com/jcmexdev/payment-service/internal/domain/ports"
	"github.com/jcmexdev/payment-service/internal/infra/http/dtos"
	"github.com/jcmexdev/payment-service/internal/infra/http/response"
)

type AccountController interface {
	CreateAccount(http.ResponseWriter, *http.Request)
}
type AccountHandler struct {
	createAccountUseCase ports.CreateAccountUseCase
	serviceName          string
}

func NewAccountHandler(createAccountUseCase ports.CreateAccountUseCase, serviceName string) *AccountHandler {
	return &AccountHandler{createAccountUseCase: createAccountUseCase, serviceName: serviceName}
}

func (a AccountHandler) CreateAccount(writer http.ResponseWriter, request *http.Request) {
	var req domain.CreateAccountRequest
	err := parseCreateAccountRequest(request, &req)
	if err != nil {
		response.HandleError(writer, request, err, "CreateAccount Failed")
		return
	}

	err = validateCreateAccountRequest(&req)
	if err != nil {
		response.HandleError(writer, request, err, "CreateAccount Failed")
		return
	}

	ctx := request.Context()
	account, err := a.createAccountUseCase.Execute(ctx, &req)
	if err != nil {
		response.HandleError(writer, request, err, "CreateAccount failed")
		return
	}

	response.SendSuccess(writer, request, http.StatusCreated, errors.CodeAccountCreated, dtos.NewCreateAccountResponse(account))
}

func parseCreateAccountRequest(request *http.Request, req *domain.CreateAccountRequest) error {
	err := json.NewDecoder(request.Body).Decode(req)
	if err != nil {
		return errors.NewAppError(errors.TypeValidationError,
			errors.CodeMalformedJSON,
			errors.GetMessage(errors.CodeMalformedJSON),
			err,
		)
	}
	return nil
}

func validateCreateAccountRequest(req *domain.CreateAccountRequest) error {
	appError := errors.NewAppError(
		errors.TypeValidationError,
		errors.CodeInvalidAttributes,
		errors.GetMessage(errors.CodeInvalidAttributes),
		nil)
	if req.UserID == "" {
		appError.WithDetail("user_id", errors.GetMessage(errors.CodeMissingUserId))
	}

	if req.Currency == "" || len(req.Currency) != 3 {
		appError.WithDetail("currency", errors.GetMessage(errors.CodeInvalidCurrency))
	}

	if len(appError.Details) > 0 {
		return appError
	}
	return nil
}
