// HTTP calls to the mock bank
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type BankClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewBankClient(baseURL string) *BankClient

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

func (b *BankClient) Authorize(req *AuthorizeRequest, idempotencyKey string) (*AuthorizeResponse, error) {
	// 1. Marshal request body to JSON
	body, err := json.Marshal(req)

	// 2. Create the request
	httpReq, err := http.NewRequest("POST", b.baseURL+"/api/v1/authorizations", bytes.NewBuffer(body))

	// 3. Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Idempotency-Key", idempotencyKey)

	// 4. Send it
	resp, err := b.httpClient.Do(httpReq)

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// convert *http.Response to *AuthorizeResponse
	var authorizeResp AuthorizeResponse
	err = json.NewDecoder(resp.Body).Decode(&authorizeResp)

	if err != nil {
		return nil, fmt.Errorf("failed to decode bank response: %w", err)
	}

	return &authorizeResp, nil
}
