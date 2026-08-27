package payment

import (
	"errors"
)

var ErrInvalidTransitionEvent = errors.New("invalid transition event")

var allowedTransitions = map[PaymentStatus]map[TransitionEvent]PaymentStatus{
	StatusCreated: {
		EventStartProcessing: StatusProcessing,
		EventFail:            StatusFailed,
	},
	StatusProcessing: {
		EventSettle: StatusSettled,
		EventFail:   StatusFailed,
	},
}
