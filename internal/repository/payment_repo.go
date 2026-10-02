// SQL queries (CREATE, UPDATE, GET)
package repository

import (
	"github.com/Inengs/ficmart-payment-gateway/internal/domain"
	"github.com/jmoiron/sqlx"

	"errors"
	"time"
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

func (r *PaymentRepository) MarkAuthorized(paymentID, bankAuthID string, authorizedAt time.Time) error {
	query := `
		UPDATE payments
		SET status = $1, bank_auth_id = $2, authorized_at = $3, updated_at = $4
		WHERE id = $5 AND status = $6
	`

	res, err := r.db.Exec(query, domain.StatusAuthorized, bankAuthID,
		authorizedAt, time.Now(), paymentID, domain.StatusPending)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("payment not in PENDING state")
	}

	return nil
}


func (r *PaymentRepository) MarkFailed(paymentID, reason string, failedAt time.Time) error {
	res, err := r.db.Exec(`
		UPDATE payments
		SET status = $1, failure_reason = $2, failed_at = $3, updated_at = $4
		WHERE id = $5 AND status = $6`,
		domain.StatusFailed, reason, failedAt, time.Now(), paymentID, domain.StatusPending)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("payment not in PENDING state")
	}
	return nil
}