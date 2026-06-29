package domain

import "time"

type PaymentStatus string

const (
    StatusPending    PaymentStatus = "PENDING" // Payment record created, authorize call not yet sent
    StatusAuthorized PaymentStatus = "AUTHORIZED" // Bank has reserved funds on the card
    StatusCaptured   PaymentStatus = "CAPTURED" // Funds have been transferred to merchant
    StatusVoided     PaymentStatus = "VOIDED" // Authorization cancelled before capture
    StatusRefunded   PaymentStatus = "REFUNDED" // Captured funds returned to customer
)

type ErrorCode string

const (
	ErrCardDeclined ErrorCode="CARD_DECLINED" // Bank declined — permanent failure, do not retry
	ErrCardExpired ErrorCode="CARD_EXPIRED" // Card expiry date is in the past
	ErrInsufficientFunds ErrorCode="INSUFFICIENT_FUNDS" // Balance too low — permanent failure
	ErrInvalidStateTransition ErrorCode="INVALID_STATE_TRANSITION" // e.g. trying to capture a voided payment
	ErrPaymentNotFound ErrorCode="PAYMENT_NOT_FOUND" // Payment ID does not exist in your system
	ErrIdempotencyConflict ErrorCode="IDEMPOTENCY_CONFLICT" // Same key used with different request body
	ErrBankUnavailable ErrorCode="BANK_UNAVAILABLE" // Bank returned 5xx after all retries exhausted
)

type Payment struct {
	ID            string
	OrderID       string
	CustomerID    string
	Amount        int
	Currency      string
	Status        PaymentStatus
	CardLastFour  string
	BankAuthID    string
	BankCaptureID string
	BankVoidID    string
	BankRefundID  string
	AuthorizedAt  time.Time
	CapturedAt time.Time
	VoidedAt time.Time
	RedundedAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}



