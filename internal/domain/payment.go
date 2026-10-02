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
	StatusFailed     PaymentStatus = "FAILED"     // Permanent bank rejection, terminal and irreversible
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
	ErrValidation             ErrorCode = "VALIDATION_ERROR"         // this is for if the request body is invalid, e.g. missing required fields, invalid values, etc. the request itself is bad
	ErrInternal               ErrorCode = "INTERNAL_ERROR"           // this is for unexpected errors that are not the fault of the client, e.g. a bug in our code, a database outage, etc. the request itself is valid, but we failed to process it
)

type Payment struct {
	ID            string        `db:"id" json:"payment_id"`
	OrderID       string        `db:"order_id" json:"order_id"`
	CustomerID    string        `db:"customer_id" json:"customer_id"`
	Amount        int           `db:"amount" json:"amount"`
	Currency      string        `db:"currency" json:"currency"`
	Status        PaymentStatus `db:"status" json:"status"`
	CardLastFour  string        `db:"card_last_four" json:"card_last_four"`
	BankAuthID    *string       `db:"bank_auth_id" json:"bank_auth_id"`
	BankCaptureID *string       `db:"bank_capture_id" json:"bank_capture_id"`
	BankVoidID    *string       `db:"bank_void_id" json:"bank_void_id"`
	BankRefundID  *string       `db:"bank_refund_id" json:"bank_refund_id"`
	AuthorizedAt  *time.Time    `db:"authorized_at" json:"authorized_at"`
	CapturedAt    *time.Time    `db:"captured_at" json:"captured_at"`
	VoidedAt      *time.Time    `db:"voided_at" json:"voided_at"`
	RefundedAt    *time.Time    `db:"refunded_at" json:"refunded_at"`
	CreatedAt     time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time     `db:"updated_at" json:"updated_at"`
	FailureReason *string    `db:"failure_reason" json:"failure_reason"`
	FailedAt      *time.Time `db:"failed_at" json:"failed_at"`
}

// this is a custom error type that includes an error code and an optional underlying cause, the error code is used to categorize the error and can be used for programmatic handling of errors, while the underlying cause can be used for debugging and logging purposes
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error // optional underlying cause
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

func NewAppError(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

func WrapAppError(code ErrorCode, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}
