// business logic, orchestrates repo + bank client
package service

import (
	"errors"
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

func (s *PaymentService) Authorize(req *AuthorizeRequest) (*domain.Payment, error) {
	// Guard before slicing. len(CardNumber)-4 panics on anything shorter
	// than four characters, taking down the handler with it.
	if len(req.CardNumber) < 4 {
		return nil, errors.New("card number too short")
	}

	payment := &domain.Payment{
		ID:           uuid.New().String(),
		OrderID:      req.OrderID,
		CustomerID:   req.CustomerID,
		Amount:       req.Amount,
		Currency:     req.Currency,
		CardLastFour: req.CardNumber[len(req.CardNumber)-4:],
		Status:       domain.StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// PERSIST BEFORE CALLING THE BANK. If we crash after the bank
	// authorizes but before we write, a PENDING row exists to reconcile
	// against (§3.4). Calling first would leave money reserved on a card
	// with no record of it anywhere.
	if err := s.repo.CreatePayment(payment); err != nil {
		return nil, err
	}

	// The payment ID is the idempotency key sent to the bank. It's already
	// unique per attempt, so a retry of this same authorization reaches
	// the bank with the same key and cannot double-charge.
	bankResp, err := s.bank.Authorize(&client.AuthorizeRequest{
		Amount:      req.Amount,
		CardNumber:  req.CardNumber,
		CVV:         req.CVV,
		ExpiryMonth: req.ExpiryMonth,
		ExpiryYear:  req.ExpiryYear,
	}, payment.ID)
	if err != nil {
		// Left PENDING deliberately. We don't know whether the bank
		// authorized — reconciliation decides, not a guess here.
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
	payment.BankAuthID = bankResp.AuthorizationID
	payment.AuthorizedAt = authorizedAt

	return payment, nil
}
