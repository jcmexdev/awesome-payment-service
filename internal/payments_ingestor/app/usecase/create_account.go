package usecase

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/ports"
)

type CreateAccountUseCase struct {
	repo ports.AccountRepository
}

func NewCreateAccountUseCase(repo ports.AccountRepository) *CreateAccountUseCase {
	return &CreateAccountUseCase{repo: repo}
}

func (c CreateAccountUseCase) Execute(ctx context.Context, input *domain.CreateAccountRequest) (*domain.Account, error) {
	/*tr := otel.Tracer("payment-service")
	ctx, span := tr.Start(ctx, "payment.create_account")
	defer span.End(*/

	account := input.NewDomainAccount()

	err := c.repo.CreateAccount(ctx, account)
	if err != nil {
		return nil, err
	}

	/*err := u.ledgerRepo.CreateAccount(ctx, account)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetSt tus(codes.Ok, "account created successfully")*/
	return account, nil
}
