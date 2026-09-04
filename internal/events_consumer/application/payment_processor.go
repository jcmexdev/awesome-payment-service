package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jcmexdev/payment-service/internal/events_consumer/ports"
	"github.com/jcmexdev/payment-service/pkg/domain"
)

type PaymentProcessor struct {
	repository ports.PaymentRepository
	// gateway ports.PaymentGatewayClient // Descomentar para llamadas a la pasarela real
}

func NewPaymentProcessor(repo ports.PaymentRepository) *PaymentProcessor {
	return &PaymentProcessor{
		repository: repo,
	}
}

// PaymentEventDTO para deserializar el evento recibido de la cola
type PaymentEventDTO struct {
	ID string `json:"id"`
}

func (p *PaymentProcessor) ProcessTransaction(ctx context.Context, payload []byte) error {
	var event PaymentEventDTO
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("invalid payment event payload: %w", err)
	}

	if event.ID == "" {
		return errors.New("event payload missing payment ID")
	}

	// 2. Obtener el estado actual del pago desde la BD
	pay, err := p.repository.FindByID(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("payment not found ID %s: %w", event.ID, err)
	}

	err = pay.TransitionTo(domain.EventStartProcessing)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidTransitionEvent) {
			fmt.Printf("[PaymentProcessor] Ignorando evento duplicado o no valido para ID %s: %v\n", pay.ID, err)
			return nil
		}
		return err
	}

	// 4. Persistir el cambio de estado inicial a PROCESSING
	if err := p.repository.UpdateStatus(ctx, pay.ID, pay.Status); err != nil {
		return fmt.Errorf("failed to set status PROCESSING for payment %s: %w", pay.ID, err)
	}

	// 5. Ejecutar la Lógica Pesada / Pasarela de Pago Externa[cite: 3]
	// err = p.gateway.Charge(ctx, pay)
	processingErr := p.executeThirdPartyCharge(ctx, pay)

	// 6. Determinar el estado final según el resultado de la integración
	if processingErr != nil {
		// Si falla el cobro, la FSM pasa el pago a FAILED
		_ = pay.TransitionTo(domain.EventFail)
		_ = p.repository.UpdateStatus(ctx, pay.ID, pay.Status)

		// Retornar error solo si es un fallo transitorio que amerite reintento en SQS,
		// o nil si es un fallo definitivo de negocio (tarjeta rechazada)
		return nil
	}

	// 7. Transición Exitosa (COMPLETED / SETTLED)[cite: 3]
	if err := pay.TransitionTo(domain.EventSettled); err != nil {
		return err
	}

	return p.repository.UpdateStatus(ctx, pay.ID, pay.Status)
}

func (p *PaymentProcessor) executeThirdPartyCharge(ctx context.Context, pay *domain.Payment) error {
	// Simulación de llamada externa
	//return errors.New("not implemented")
	return nil
}
