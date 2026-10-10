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
	// Guard before slicing: len(CardNumber)-4 panics on anything shorter than four characters.
	if len(req.CardNumber) < 4 {
		return nil, domain.NewAppError(domain.ErrValidation, "card number too short")
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
	// against. Calling first would leave money reserved on a card
	// with no record of it anywhere.
	if err := s.repo.CreatePayment(payment); err != nil {
		return nil, err
	}

	// The bank key is derived from the client's Idempotency-Key, so a retry of
	// the same request (even as a new HTTP request) reaches the bank with the
	// same key and cannot double-charge.
	bankKey := "authorize:" + idempotencyKey

	bankReq := &client.AuthorizeRequest{
		Amount:      req.Amount,
		CardNumber:  req.CardNumber,
		CVV:         req.CVV,
		ExpiryMonth: req.ExpiryMonth,
		ExpiryYear:  req.ExpiryYear,
	}

	bankResp, err := withRetry(ctx, func() (*client.AuthorizeResponse, error) {
		return s.bank.Authorize(ctx, bankReq, bankKey)
	})
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) && isPermanent(appErr.Code) {
			if terr := state.Transition(payment.Status, domain.StatusFailed); terr == nil {
				if merr := s.repo.MarkFailed(payment.ID, string(appErr.Code), time.Now()); merr != nil {
					log.Printf("failed to mark payment %s FAILED: %v", payment.ID, merr)
				}
			}
		}
		// transient (BANK_UNAVAILABLE) and INTERNAL_ERROR stay PENDING for the reconciler
		return nil, err
	}

	if err := state.Transition(payment.Status, domain.StatusAuthorized); err != nil {
		return nil, err
	}

	authorizedAt := time.Now()
	if err := s.repo.MarkAuthorized(payment.ID, bankResp.AuthorizationID, authorizedAt); err != nil {
		return nil, err
	}

	return s.repo.GetPaymentByID(payment.ID)
}

func (s *PaymentService) Capture(ctx context.Context, idempotencyKey, paymentID string) (*domain.Payment, error) {
	p, err := s.repo.GetPaymentByID(paymentID)
	if err != nil {
		return nil, err
	}
	if err := state.Transition(p.Status, domain.StatusCaptured); err != nil {
		return nil, err
	}
	if p.BankAuthID == nil {
		return nil, domain.NewAppError(domain.ErrInternal, "payment has no bank authorization")
	}

	bankKey := "capture:" + idempotencyKey
	resp, err := withRetry(ctx, func() (*client.CaptureResponse, error) {
		return s.bank.Capture(ctx, &client.CaptureRequest{AuthorizationID: *p.BankAuthID, Amount: p.Amount}, bankKey)
	})
	if err != nil {
		return nil, err // payment stays AUTHORIZED; safe to retry with the same key
	}

	capturedAt := time.Now()
	if err := s.repo.MarkCaptured(p.ID, resp.CaptureID, capturedAt); err != nil {
		return nil, err
	}
	return s.repo.GetPaymentByID(p.ID)
}

func (s *PaymentService) Void(ctx context.Context, idempotencyKey, paymentID string) (*domain.Payment, error) {
	p, err := s.repo.GetPaymentByID(paymentID) // get the payment record from the repository using the provided payment ID
	if err != nil {
		return nil, err // return the error if the payment record cannot be retrieved
	}
	if err := state.Transition(p.Status, domain.StatusVoided); err != nil {
		return nil, err // return the error if the payment record cannot be transitioned to the VOIDED state
	}
	if p.BankAuthID == nil {
		return nil, domain.NewAppError(domain.ErrInternal, "payment has no bank authorization") // return an internal error if the payment record does not have a bank authorization ID, which is required to void the payment
	}

	bankKey := "void:" + idempotencyKey // create a unique bank key for the void operation using the provided idempotency key
	resp, err := withRetry(ctx, func() (*client.VoidResponse, error) {
		return s.bank.Void(ctx, &client.VoidRequest{AuthorizationID: *p.BankAuthID}, bankKey) // call the bank client to void the payment using the bank authorization ID and the unique bank key, with retry logic for transient failures
	})
	if err != nil {
		return nil, err // stays AUTHORIZED; safe to retry with the same key
	}

	if err := s.repo.MarkVoided(p.ID, resp.VoidID, time.Now()); err != nil {
		return nil, err // return the error if the payment record cannot be marked as VOIDED in the repository
	}

	updated, err := s.repo.GetPaymentByID(p.ID) // re-read the payment record from the repository to get the updated state after the void operation
	if err != nil {
		log.Printf("re-read after void failed for %s: %v", p.ID, err) // log the error if the payment record cannot be re-read from the repository, but do not return an error to the caller
		return p, nil                                                 // the void did succeed
	}
	return updated, nil // return the updated payment record to the caller, which should now be in the VOIDED state
}

func (s *PaymentService) Refund(ctx context.Context, idempotencyKey, paymentID string) (*domain.Payment, error) {
	p, err := s.repo.GetPaymentByID(paymentID) // get the payment record from the repository using the provided payment ID
	if err != nil {
		return nil, err // return the error if the payment record cannot be retrieved
	}
	if err := state.Transition(p.Status, domain.StatusRefunded); err != nil {
		return nil, err // return the error if the payment record cannot be transitioned to the REFUNDED state
	}
	if p.BankCaptureID == nil {
		return nil, domain.NewAppError(domain.ErrInternal, "payment has no bank capture") // return an internal error if the payment record does not have a bank capture ID, which is required to refund the payment
	}

	bankKey := "refund:" + idempotencyKey // create a unique bank key for the refund operation using the provided idempotency key
	resp, err := withRetry(ctx, func() (*client.RefundResponse, error) {
		return s.bank.Refund(ctx, &client.RefundRequest{CaptureID: *p.BankCaptureID, Amount: p.Amount}, bankKey) // call the bank client to refund the payment using the bank capture ID and the unique bank key, with retry logic for transient failures
	})
	if err != nil {
		return nil, err // stays CAPTURED; safe to retry with the same key
	}

	if err := s.repo.MarkRefunded(p.ID, resp.RefundID, time.Now()); err != nil {
		return nil, err // return the error if the payment record cannot be marked as REFUNDED in the repository
	}

	updated, err := s.repo.GetPaymentByID(p.ID)
	if err != nil {
		log.Printf("re-read after refund failed for %s: %v", p.ID, err) // log the error if the payment record cannot be re-read from the repository, but do not return an error to the caller
		return p, nil                                                   // the refund did succeed
	}
	return updated, nil // return the updated payment record to the caller, which should now be in the REFUNDED state
}

func (s *PaymentService) GetPayment(id string) (*domain.Payment, error) {
	return s.repo.GetPaymentByID(id)
}

func (s *PaymentService) GetPaymentByOrder(orderID string) (*domain.Payment, error) {
	return s.repo.GetByOrderID(orderID)
}

func (s *PaymentService) ListCustomerPayments(customerID string, limit, offset int) ([]domain.Payment, error) {
	return s.repo.ListByCustomerID(customerID, limit, offset)
}

// Backoff and retry for transient failures. The bank is a separate system
// and can be down or slow. We don't want to fail the payment if the bank
// is just having a bad moment, so we retry a few times with exponential
// backoff. The bank client already returns an error for 5xx responses,
// so we don't have to check the status code here. This is the backoff with jitter pattern recommended by AWS: https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
func withRetry[T any](ctx context.Context, call func() (T, error)) (T, error) {
	const maxAttempts = 3             // maximum number of attempts to call the bank service before giving up
	backoff := 200 * time.Millisecond // initial backoff duration before retrying, which will be doubled after each attempt
	var zero T                        // zero value of the generic type T, used to return in case of an error
	var lastErr error                 //

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := call() // call the provided function to make the bank service request
		if err == nil {
			return resp, nil // return the response if the call was successful
		}
		lastErr = err // store the last error encountered during the call for logging and returning if all attempts fail

		var appErr *domain.AppError // declare a variable to hold the application error returned by the bank service
		if !errors.As(err, &appErr) || appErr.Code != domain.ErrBankUnavailable {
			return zero, err // return the error if it is not a bank unavailable error, as we only want to retry on transient failures
		}
		if ctx.Err() != nil {
			return zero, ctx.Err() // return the context error if the context has been canceled or timed out, as we don't want to continue retrying in that case
		}
		if attempt == maxAttempts {
			break // break the loop if we have reached the maximum number of attempts, as we don't want to retry anymore
		}
		wait := backoff + time.Duration(rand.Int63n(int64(backoff))) // calculate the wait time before the next retry attempt, which is a random duration between the backoff and twice the backoff to add jitter and avoid thundering herd problems
		select {
		case <-time.After(wait): // wait for the calculated duration before retrying the call to the bank service
		case <-ctx.Done():
			return zero, ctx.Err() // return the context error if the context has been canceled or timed out while waiting, as we don't want to continue retrying in that case
		}
		backoff *= 2 // double the backoff duration for the next retry attempt, to implement exponential backoff and reduce the load on the bank service during transient failures
	}
	return zero, lastErr // return the last error encountered during the call if all attempts have failed, as we want to propagate the error to the caller for handling
}

func isPermanent(code domain.ErrorCode) bool {
	switch code {
	case domain.ErrCardDeclined, domain.ErrCardExpired,
		domain.ErrInsufficientFunds, domain.ErrValidation: // these are permanent errors that should not be retried, as they indicate a problem with the payment request or the card itself
		return true
	}
	return false
}
