// HTTP calls to the mock bank
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Inengs/ficmart-payment-gateway/internal/domain"
)

type BankClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewBankClient(baseURL string) *BankClient {
	return &BankClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// this is the request from our gateway to the mock_bank api
type AuthorizeRequest struct {
	Amount      int    `json:"amount"`
	CardNumber  string `json:"card_number"`
	CVV         string `json:"cvv"`
	ExpiryMonth int    `json:"expiry_month"`
	ExpiryYear  int    `json:"expiry_year"`
}

type AuthorizeResponse struct {
	AuthorizationID string `json:"authorization_id"`
	Status          string `json:"status"`
	Amount          int    `json:"amount"`
	Currency        string `json:"currency"`
	CreatedAt       string `json:"created_at"`
	ExpiresAt       string `json:"expires_at"`
}

type bankErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func mapBankError(status int, body []byte) error {
	var be bankErrorBody
	if err := json.Unmarshal(body, &be); err != nil {
		be = bankErrorBody{} // body wasn't JSON, fall back to status only
	}
	raw := fmt.Errorf("bank returned %d: error=%q message=%q", status, be.Error, be.Message)

	if status >= 500 {
		return domain.WrapAppError(domain.ErrBankUnavailable, "bank is unavailable", raw)
	}

	switch be.Error {
	case "insufficient_funds":
		return domain.WrapAppError(domain.ErrInsufficientFunds, "insufficient funds", raw)
	case "card_expired":
		return domain.WrapAppError(domain.ErrCardExpired, "card has expired", raw)
	case "invalid_cvv", "invalid_card":
		return domain.WrapAppError(domain.ErrCardDeclined, "card details are invalid", raw)
	case "invalid_amount":
		return domain.WrapAppError(domain.ErrValidation, "invalid amount", raw)
	default:
		// unknown 4xx or non-JSON body: likely our bug, so log it and don't blame the card
		return domain.WrapAppError(domain.ErrInternal, "unexpected bank response", raw)
	}
}

func (b *BankClient) Authorize(ctx context.Context, req *AuthorizeRequest, idempotencyKey string) (*AuthorizeResponse, error) {
	// 1. Marshal request body to JSON
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	// 2. Create the request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.baseURL+"/api/v1/authorizations", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// 3. Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Idempotency-Key", idempotencyKey)

	// 4. Send it
	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, domain.WrapAppError(domain.ErrBankUnavailable, "bank is unavailable", err)
	}

	// Leaking this holds the connection open and exhausts the pool.
	defer resp.Body.Close()

	// read the body once; it's a stream and can't be read twice
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, domain.WrapAppError(domain.ErrBankUnavailable, "failed to read bank response", err)
	}

	// STATUS CODE FIRST. Decoding a 500 into AuthorizeResponse succeeds
	// with every field empty — the call looks like it worked and the
	// authorization ID is silently "".
	if resp.StatusCode >= 400 {
		return nil, mapBankError(resp.StatusCode, respBody)
	}

	// convert *http.Response to *AuthorizeResponse
	var authorizeResp AuthorizeResponse
	if err := json.Unmarshal(respBody, &authorizeResp); err != nil {
		return nil, domain.WrapAppError(domain.ErrBankUnavailable, "invalid bank response", err)
	}
	return &authorizeResp, nil
}
