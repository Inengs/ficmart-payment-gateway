// valid transition rules
package state

import (
	"fmt"

	"github.com/Inengs/ficmart-payment-gateway/internal/domain"
)

var validTransitions = map[domain.PaymentStatus][]domain.PaymentStatus{
	domain.StatusPending:    {domain.StatusAuthorized, domain.StatusFailed},
	domain.StatusAuthorized: {domain.StatusCaptured, domain.StatusVoided},
	domain.StatusCaptured:   {domain.StatusRefunded},
}

func Transition(current, next domain.PaymentStatus) error {
	allowed, ok := validTransitions[current] // allowed = value stored in the map, ok (bool) = tells whether it was found

	if !ok {
		return domain.NewAppError(domain.ErrInvalidStateTransition, fmt.Sprintf("cannot move from %s to %s", current, next))
	}

	for _, s := range allowed {
		if s == next { // if the expected state is found, it is a valid transition and return nil error
			return nil
		}
	}

	return domain.NewAppError(domain.ErrBankUnavailable, fmt.Sprintf("cannot move from %s to %s", current, next))
}
