package state

import (
	"errors"

	"github.com/Inengs/ficmart-payment-gateway/internal/domain"
)

var validTransitions = map[domain.PaymentStatus][]domain.PaymentStatus{
    domain.StatusPending:    {domain.StatusAuthorized},
    domain.StatusAuthorized: {domain.StatusCaptured, domain.StatusVoided},
    domain.StatusCaptured:   {domain.StatusRefunded},
}

func Transition(current, next domain.PaymentStatus) error {
	allowed, ok := validTransitions[current] // allowed = value stored in the map, ok (bool) = tells whether it was found

	if !ok {
		return errors.New("unknown state")
	}

	for _, s := range allowed {
		if s == next { // if the expected state is found, it is a valid transition and return nil error
			return nil
		}
	}

	return errors.New("invalid transaction")
}