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

type CaptureRequest struct {
	AuthorizationID string `json:"authorization_id"`
	Amount          int    `json:"amount"`
}

type CaptureResponse struct {
	CaptureID string `json:"capture_id"`
	Status    string `json:"status"`
}

type VoidRequest struct {
	AuthorizationID string `json:"authorization_id"`
}

type VoidResponse struct {
	VoidID          string `json:"void_id"`
	AuthorizationID string `json:"authorization_id"`
	Status          string `json:"status"`
	VoidedAt        string `json:"voided_at"`
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
	case "authorization_already_used":
		return domain.WrapAppError(domain.ErrInvalidStateTransition, "payment cannot be changed in its current state", raw) // this is a terminal state, so we don't want to retry
	default:
		// unknown 4xx or non-JSON body: likely our bug, so log it and don't blame the card
		// includes authorization_not_found: the gateway only sends IDs the bank gave it,
		return domain.WrapAppError(domain.ErrInternal, "unexpected bank response", raw)
	}
}

// post helper
func (b *BankClient) post(ctx context.Context, path, idempotencyKey string, in, out any) error {
	body, err := json.Marshal(in) // marshal the request body to JSON
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err) // return an error if marshalling fails
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+path, bytes.NewReader(body)) // create a new HTTP POST request with the given context, path, and request body
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err) // return an error if request creation fails
	}
	req.Header.Set("Content-Type", "application/json") // set the Content-Type header to application/json
	req.Header.Set("Idempotency-Key", idempotencyKey)  // set the Idempotency-Key header

	resp, err := b.httpClient.Do(req) // send the HTTP request using the httpClient
	if err != nil {
		domain.WrapAppError(domain.ErrBankUnavailable, "bank is unavailable", err) // wrap and return an error if the request fails
	}

	defer resp.Body.Close() // ensure the response body is closed after reading

	respBody, err := io.ReadAll(resp.Body) // read the response body
	if err != nil {
		return domain.WrapAppError(domain.ErrBankUnavailable, "failed to read bank response", err) // wrap and return an error if reading the response fails
	}
	if resp.StatusCode >= 400 { // check if the response status code indicates an error
		return mapBankError(resp.StatusCode, respBody) // map and return the bank error
	}
	if err := json.Unmarshal(respBody, out); err != nil { // unmarshal the response body into the output parameter
		return domain.WrapAppError(domain.ErrBankUnavailable, "invalid bank response", err) // wrap and return an error if unmarshalling fails
	}
	return nil // return nil if everything succeeds
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

func (b *BankClient) Capture(ctx context.Context, req *CaptureRequest, idempotencyKey string) (*CaptureResponse, error) {
	var out CaptureResponse
	if err := b.post(ctx, "/api/v1/captures", idempotencyKey, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (b *BankClient) Void(ctx context.Context, req *VoidRequest, key string) (*VoidResponse, error) {
	var out VoidResponse
	if err := b.post(ctx, "/api/v1/voids", key, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
