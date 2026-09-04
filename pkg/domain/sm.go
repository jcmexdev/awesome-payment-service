package domain

import (
	"errors"
)

var ErrInvalidTransitionEvent = errors.New("invalid transition event")

var allowedTransitions = map[PaymentStatus]map[TransitionEvent]PaymentStatus{
	PaymentStatusCreated: {
		EventStartProcessing: PaymentStatusProcessing,
		EventFail:            PaymentStatusFailed,
	},
	PaymentStatusProcessing: {
		EventSettled: PaymentStatusSettled,
		EventFail:    PaymentStatusFailed,
	},
}
