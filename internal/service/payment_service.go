// business logic, orchestrates repo + bank client
package service

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/Inengs/ficmart-payment-gateway/internal/client"
	"github.com/Inengs/ficmart-payment-gateway/internal/domain"
	"github.com/Inengs/ficmart-payment-gateway/internal/repository"
	"github.com/Inengs/ficmart-payment-gateway/internal/state"
	"github.com/google/uuid"
)

type PaymentService struct {
	repo *repository.PaymentRepository
	bank *client.BankClient
}

// this creates and returns a PaymentRepository instance with the database connection injected into it.
func NewPaymentService(repo *repository.PaymentRepository, bank *client.BankClient) *PaymentService {
	return &PaymentService{repo: repo, bank: bank}
}

// this is the request coming from ficmart
type AuthorizeRequest struct {
	OrderID     string `json:"order_id"`
	CustomerID  string `json:"customer_id"`
	Amount      int    `json:"amount"`
	Currency    string `json:"currency"`
	CardNumber  string `json:"card_number"`
	CVV         string `json:"cvv"`
	ExpiryMonth int    `json:"expiry_month"`
	ExpiryYear  int    `json:"expiry_year"`
}

func (s *PaymentService) Authorize(ctx context.Context, idempotencyKey string, req *AuthorizeRequest) (*domain.Payment, error) {
	// Guard before slicing. len(CardNumber)-4 panics on anything shorter
	// than four characters, taking down the handler with it.
	if len(req.CardNumber) < 4 {
		return nil, errors.New("card number too short")
	}

	now := time.Now()
	payment := &domain.Payment{
		ID:           uuid.New().String(),
		OrderID:      req.OrderID,
		CustomerID:   req.CustomerID,
		Amount:       req.Amount,
		Currency:     req.Currency,
		CardLastFour: req.CardNumber[len(req.CardNumber)-4:],
		Status:       domain.StatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// PERSIST BEFORE CALLING THE BANK. If we crash after the bank
	// authorizes but before we write, a PENDING row exists to reconcile
	// against (§3.4). Calling first would leave money reserved on a card
	// with no record of it anywhere.
	if err := s.repo.CreatePayment(payment); err != nil {
		return nil, err
	}

	bankKey := "authorize:" + idempotencyKey
	// The payment ID is the idempotency key sent to the bank. It's already
	// unique per attempt, so a retry of this same authorization reaches
	// the bank with the same key and cannot double-charge.
	bankResp, err := s.authorizeWithRetry(ctx, &client.AuthorizeRequest{
		Amount:      req.Amount,
		CardNumber:  req.CardNumber,
		CVV:         req.CVV,
		ExpiryMonth: req.ExpiryMonth,
		ExpiryYear:  req.ExpiryYear,
	}, bankKey)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) && isPermanent(appErr.Code) {
			if transition_error := state.Transition(payment.Status, domain.StatusFailed); transition_error == nil {
				if merr := s.repo.MarkFailed(payment.ID, string(appErr.Code), time.Now()); merr != nil {
					log.Printf("failed to mark payment %s FAILED: %v", payment.ID, merr)
				}
			}
		}
		// transient (BANK_UNAVAILABLE) and INTERNAL_ERROR stay PENDING for the reconciler
		return nil, err
	}

	// Checked even though the current state is known, because the rule
	// belongs in one place. A later capture or void path calls the same
	// function rather than reimplementing the logic.
	if err := state.Transition(payment.Status, domain.StatusAuthorized); err != nil {
		return nil, err
	}

	authorizedAt := time.Now()
	if err := s.repo.MarkAuthorized(payment.ID, bankResp.AuthorizationID, authorizedAt); err != nil {
		return nil, err
	}

	payment.Status = domain.StatusAuthorized
	payment.BankAuthID = &bankResp.AuthorizationID
	payment.AuthorizedAt = &authorizedAt

	return payment, nil
}

// Backoff and retry for transient failures. The bank is a separate system
// and can be down or slow. We don't want to fail the payment if the bank
// is just having a bad moment, so we retry a few times with exponential
// backoff. The bank client already returns an error for 5xx responses,
// so we don't have to check the status code here. This is the backoff with jitter pattern recommended by AWS: https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
func (s *PaymentService) authorizeWithRetry(ctx context.Context, req *client.AuthorizeRequest, key string) (*client.AuthorizeResponse, error) {
	const maxAttempts = 3
	backoff := 200 * time.Millisecond
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := s.bank.Authorize(ctx, req, key) // key should be unique per request
		if err == nil {
			return resp, nil
		}
		lastErr = err

		// only transient failures should be retried, not permanent ones like card declined or expired
		var appErr *domain.AppError
		if !errors.As(err, &appErr) || appErr.Code != domain.ErrBankUnavailable {
			return nil, err
		}

		// caller is gone, so stop retrying
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if attempt == maxAttempts {
			break
		}

		// backoff with jitter; wakes early if the context is cancelled
		wait := backoff + time.Duration(rand.Int63n(int64(backoff)))
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		backoff *= 2
	}
	return nil, lastErr
}

func isPermanent(code domain.ErrorCode) bool {
	switch code {
	case domain.ErrCardDeclined, domain.ErrCardExpired,
		domain.ErrInsufficientFunds, domain.ErrValidation:
		return true
	}
	return false
}
