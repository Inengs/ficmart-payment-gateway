// business logic, orchestrates repo + bank client
package service

import (
	"time"

	"github.com/Inengs/ficmart-payment-gateway/internal/client"
	"github.com/Inengs/ficmart-payment-gateway/internal/domain"
	"github.com/Inengs/ficmart-payment-gateway/internal/repository"
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

	err := s.repo.CreatePayment(payment)

	if err != nil {
		return nil, err
	}

	return payment, nil
}
