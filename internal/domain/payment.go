// Payment struct, status constants, error types
package domain

import "time"

type PaymentStatus string

const (
	StatusPending    PaymentStatus = "PENDING"    // Payment record created, authorize call not yet sent
	StatusAuthorized PaymentStatus = "AUTHORIZED" // Bank has reserved funds on the card
	StatusCaptured   PaymentStatus = "CAPTURED"   // Funds have been transferred to merchant
	StatusVoided     PaymentStatus = "VOIDED"     // Authorization cancelled before capture
	StatusRefunded   PaymentStatus = "REFUNDED"   // Captured funds returned to customer
)

type ErrorCode string

const (
	ErrCardDeclined           ErrorCode = "CARD_DECLINED"            // Bank declined — permanent failure, do not retry
	ErrCardExpired            ErrorCode = "CARD_EXPIRED"             // Card expiry date is in the past
	ErrInsufficientFunds      ErrorCode = "INSUFFICIENT_FUNDS"       // Balance too low — permanent failure
	ErrInvalidStateTransition ErrorCode = "INVALID_STATE_TRANSITION" // e.g. trying to capture a voided payment
	ErrPaymentNotFound        ErrorCode = "PAYMENT_NOT_FOUND"        // Payment ID does not exist in your system
	ErrIdempotencyConflict    ErrorCode = "IDEMPOTENCY_CONFLICT"     // Same key used with different request body
	ErrBankUnavailable        ErrorCode = "BANK_UNAVAILABLE"         // Bank returned 5xx after all retries exhausted
)

type Payment struct {
	ID            string        `db:"id"`
	OrderID       string        `db:"order_id"`
	CustomerID    string        `db:"customer_id"`
	Amount        int           `db:"amount"`
	Currency      string        `db:"currency"`
	Status        PaymentStatus `db:"status"`
	CardLastFour  string        `db:"card_last_four"`
	BankAuthID    *string       `db:"bank_auth_id" json:"bank_auth_id"`
	BankCaptureID *string       `db:"bank_capture_id" json:"bank_capture_id"`
	BankVoidID    *string       `db:"bank_void_id" json:"bank_void_id"`
	BankRefundID  *string       `db:"bank_refund_id" json:"bank_refund_id"`
	AuthorizedAt  *time.Time    `db:"authorized_at" json:"authorized_at"`
	CapturedAt    *time.Time    `db:"captured_at" json:"captured_at"`
	VoidedAt      *time.Time    `db:"voided_at" json:"voided_at"`
	RefundedAt    *time.Time    `db:"refunded_at" json:"refunded_at"`
	CreatedAt     time.Time     `db:"created_at"`
	UpdatedAt     time.Time     `db:"updated_at"`
}
