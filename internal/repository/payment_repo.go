package repository

import (
	"github.com/Inengs/ficmart-payment-gateway/internal/domain"
	"github.com/jmoiron/sqlx"
)

type PaymentRepository struct { // repository struct
	db *sqlx.DB
}

func NewPaymentRepository(db *sqlx.DB) *PaymentRepository { // constructor
	return &PaymentRepository{db: db}
}


func (r *PaymentRepository) CreatePayment(p *domain.Payment) error {
	query := `
		INSERT INTO payments (id, order_id, customer_id, amount, currency, status, card_last_four, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)	
	`

	_, err := r.db.Exec(query, p.ID, p.OrderID, p.CustomerID, p.Amount, p.Currency, domain.StatusPending, p.CardLastFour, p.CreatedAt, p.UpdatedAt)

	return err
}	