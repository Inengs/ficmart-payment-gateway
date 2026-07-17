// HTTP handlers, parses requests, calls service
package handler

import (
	"net/http"
	"time"

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
	// Parse JSON
	var req service.AuthorizeRequest
	err := c.ShouldBindJSON(&req)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // gin.H is a quick way to build JSON, It is short hand for map[string]interface{}
		return
	}

	// input validation
	if req.Amount < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be greater than 0"})
		return
	}

	if req.CardNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "card number is required"})
		return
	}

	if req.CVV == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CVV is required"})
		return
	}

	if req.ExpiryMonth < 1 || req.ExpiryMonth > 12 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expiry month"})
		return
	}

	if req.ExpiryYear < time.Now().Year() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Card has expired"})
		return
	}

	if req.OrderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orderid is empty"})
		return
	}

	if req.CustomerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customerid is empty"})
		return
	}

	// call authorize service
	payment, err := h.service.Authorize(&req)

	// return JSON error on failure
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "incomplete response"})
		return
	}

	// return JSON response with payment details on success
	c.JSON(http.StatusOK, gin.H{
		"payment_id":     payment.ID,
		"status":         payment.Status,
		"amount":         payment.Amount,
		"currency":       payment.Currency,
		"authorized_at":  payment.AuthorizedAt,
	})
}
