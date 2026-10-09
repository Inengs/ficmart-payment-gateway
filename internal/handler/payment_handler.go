// HTTP handlers, parses requests, calls service
package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Inengs/ficmart-payment-gateway/internal/domain"
	"github.com/Inengs/ficmart-payment-gateway/internal/service"
	"github.com/gin-gonic/gin"
)

// struct -> constructor -> methods
type PaymentHandler struct {
	service *service.PaymentService
}

func NewPaymentHandler(service *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func (h *PaymentHandler) Authorize(c *gin.Context) {
	key := c.GetString("idempotency_key")

	// Parse JSON
	var req service.AuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, domain.WrapAppError(domain.ErrValidation, "invalid request body", err))
		return
	}
	if err := validateAuthorize(&req); err != nil {
		respondError(c, err)
		return
	}

	// call authorize service
	payment, err := h.service.Authorize(c.Request.Context(), key, &req) // call the service layer to authorize the payment
	if err != nil {
		respondError(c, err) // respond with appropriate error based on the type of error returned by the service layer
		return
	}

	// return JSON response with payment details on success
	c.JSON(http.StatusOK, gin.H{
		"payment_id":    payment.ID,
		"status":        payment.Status,
		"amount":        payment.Amount,
		"currency":      payment.Currency,
		"authorized_at": payment.AuthorizedAt,
	})
}

func (h *PaymentHandler) Capture(c *gin.Context) {
	key := c.GetString("idempotency_key")
	payment, err := h.service.Capture(c.Request.Context(), key, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, payment)
}

func (h *PaymentHandler) Void(c *gin.Context) {
	key := c.GetString("idempotency_key")
	payment, err := h.service.Void(c.Request.Context(), key, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, payment)
}

func (h *PaymentHandler) Refund(c *gin.Context) {
	key := c.GetString("idempotency_key")
	payment, err := h.service.Refund(c.Request.Context(), key, c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, payment)
}

func validateAuthorize(req *service.AuthorizeRequest) error {
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.CustomerID = strings.TrimSpace(req.CustomerID)
	req.CardNumber = strings.TrimSpace(req.CardNumber)
	req.CVV = strings.TrimSpace(req.CVV)

	// required fields
	if req.OrderID == "" {
		return domain.NewAppError(domain.ErrValidation, "order_id is required")
	}
	if req.CustomerID == "" {
		return domain.NewAppError(domain.ErrValidation, "customer_id is required")
	}
	if req.CardNumber == "" || req.CVV == "" {
		return domain.NewAppError(domain.ErrValidation, "card_number and cvv are required")
	}

	// amount (cents)
	if req.Amount <= 0 {
		return domain.NewAppError(domain.ErrValidation, "amount must be greater than 0")
	}

	// currency: default to USD, reject anything else
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.Currency != "USD" {
		return domain.NewAppError(domain.ErrValidation, "only USD is supported")
	}

	// card format (the bank does the Luhn check)
	if !isDigits(req.CardNumber) || len(req.CardNumber) < 13 || len(req.CardNumber) > 19 {
		return domain.NewAppError(domain.ErrValidation, "card_number must be 13 to 19 digits")
	}
	if !isDigits(req.CVV) || len(req.CVV) < 3 || len(req.CVV) > 4 {
		return domain.NewAppError(domain.ErrValidation, "cvv must be 3 or 4 digits")
	}

	// expiry: month in range, and not before the current month
	if req.ExpiryMonth < 1 || req.ExpiryMonth > 12 {
		return domain.NewAppError(domain.ErrValidation, "expiry_month must be between 1 and 12")
	}
	now := time.Now().UTC()
	if req.ExpiryYear < now.Year() ||
		(req.ExpiryYear == now.Year() && req.ExpiryMonth < int(now.Month())) {
		return domain.NewAppError(domain.ErrCardExpired, "card has expired")
	}

	return nil
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func statusFor(code domain.ErrorCode) int {
	switch code {
	case domain.ErrValidation:
		return http.StatusBadRequest // 400
	case domain.ErrCardDeclined, domain.ErrCardExpired, domain.ErrInsufficientFunds:
		return http.StatusPaymentRequired // 402
	case domain.ErrPaymentNotFound:
		return http.StatusNotFound // 404
	case domain.ErrInvalidStateTransition, domain.ErrIdempotencyConflict:
		return http.StatusConflict // 409
	case domain.ErrBankUnavailable:
		return http.StatusServiceUnavailable // 503
	default:
		return http.StatusInternalServerError // 500
	}
}

func respondError(c *gin.Context, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		log.Printf("code=%s cause=%v", appErr.Code, appErr.Err)
		if appErr.Code == domain.ErrBankUnavailable {
			c.Header("Retry-After", "5")
		}
		c.JSON(statusFor(appErr.Code), gin.H{"code": appErr.Code, "message": appErr.Message})
		return
	}
	log.Printf("unexpected error: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"code": domain.ErrInternal, "message": "internal error"})
}
