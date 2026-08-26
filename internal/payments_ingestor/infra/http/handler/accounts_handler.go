package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/errors"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/ports"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/http/dtos"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/http/response"
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

	response.SendSuccess(writer, request, http.StatusCreated, constants.CodeAccountCreated, dtos.NewCreateAccountResponse(account))
}

func parseCreateAccountRequest(request *http.Request, req *domain.CreateAccountRequest) error {
	err := json.NewDecoder(request.Body).Decode(req)
	if err != nil {
		return errors.NewAppError(constants.TypeValidationError,
			constants.CodeMalformedJSON,
			constants.GetMessage(constants.CodeMalformedJSON),
			err,
		)
	}
	return nil
}

func validateCreateAccountRequest(req *domain.CreateAccountRequest) error {
	appError := errors.NewAppError(
		constants.TypeValidationError,
		constants.CodeInvalidAttributes,
		constants.GetMessage(constants.CodeInvalidAttributes),
		nil)
	if req.UserID == "" {
		appError.WithDetail("user_id", constants.GetMessage(constants.CodeMissingUserId))
	}

	if req.Currency == "" || len(req.Currency) != 3 {
		appError.WithDetail("currency", constants.GetMessage(constants.CodeInvalidCurrency))
	}

	if len(appError.Details) > 0 {
		return appError
	}
	return nil
}
