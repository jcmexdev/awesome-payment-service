package usecase

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type CreateAccountUseCase struct {
	repo ports.AccountRepository
}

func NewCreateAccountUseCase(repo ports.AccountRepository) *CreateAccountUseCase {
	return &CreateAccountUseCase{repo: repo}
}

func (c CreateAccountUseCase) Execute(ctx context.Context, input *domain.CreateAccountRequest) (*domain.Account, error) {
	tr := otel.Tracer("payments_ingestor")
	ctx, span := tr.Start(ctx, "payment.create_account")
	defer span.End()

	account := input.NewDomainAccount()

	err := c.repo.CreateAccount(ctx, account)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "account created successfully")
	return account, nil
}
