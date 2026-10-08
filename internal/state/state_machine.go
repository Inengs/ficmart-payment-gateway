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
    for _, s := range validTransitions[current] { // iterate over valid next states for the current state
        if s == next { // check if the next state is valid
            return nil // valid transition found
        }
    }

    return domain.NewAppError(
        domain.ErrInvalidStateTransition, // error type
        fmt.Sprintf("cannot move from %s to %s", current, next), // error message
    )
}
