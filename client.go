// Package cryptomepay provides a Go client for the Cryptome Pay API.
//
// Cryptome Pay is a non-custodial cryptocurrency payment gateway.
// Payments go directly to your wallet - we never hold your funds.
//
// Example usage:
//
//	client := cryptomepay.NewClient("sk_live_xxx", "your_secret")
//
//	payment, err := client.CreatePayment(&cryptomepay.CreatePaymentParams{
//	    OrderID:   "ORDER_001",
//	    Amount:    100.00,
//	    NotifyURL: "https://example.com/webhook",
//	    ChainType: cryptomepay.ChainBSC,
//	})
package cryptomepay

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Version is the SDK version
const Version = "1.0.0"

// Default base URLs
const (
	ProductionURL = "https://api.cryptomepay.com/api/v1"
	SandboxURL    = "https://sandbox.cryptomepay.com/api/v1"
	StagingURL    = "https://staging.cryptomepay.com/api/v1"
)

// Chain types
const (
	ChainTRC20    = "TRC20"
	ChainBSC      = "BSC"
	ChainPolygon  = "POLYGON"
	ChainETH      = "ETH"
	ChainArbitrum = "ARBITRUM"
)

// Payment status codes
const (
	StatusPending = 1
	StatusPaid    = 2
	StatusExpired = 3
)

// Client is the Cryptome Pay API client
type Client struct {
	apiKey     string
	apiSecret  string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Cryptome Pay client with default settings
func NewClient(apiKey, apiSecret string) *Client {
	return &Client{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   ProductionURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Option is a function that configures the client
type Option func(*Client)

// WithBaseURL sets a custom base URL
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// NewClientWithOptions creates a new client with custom options
func NewClientWithOptions(apiKey, apiSecret string, opts ...Option) *Client {
	c := NewClient(apiKey, apiSecret)
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// UseSandbox switches the client to use the sandbox environment
func (c *Client) UseSandbox() *Client {
	c.baseURL = SandboxURL
	return c
}

// UseProduction switches the client to use the production environment
func (c *Client) UseProduction() *Client {
	c.baseURL = ProductionURL
	return c
}

// CreatePaymentParams holds parameters for creating a payment
type CreatePaymentParams struct {
	OrderID     string  `json:"order_id"`
	Amount      float64 `json:"amount"`
	NotifyURL   string  `json:"notify_url"`
	RedirectURL string  `json:"redirect_url,omitempty"`
	ChainType   string  `json:"chain_type,omitempty"`
}

// PaymentData holds payment response data
type PaymentData struct {
	TradeID        string  `json:"trade_id"`
	OrderID        string  `json:"order_id"`
	Amount         float64 `json:"amount"`
	ActualAmount   float64 `json:"actual_amount"`
	Token          string  `json:"token"`
	ChainType      string  `json:"chain_type"`
	ChainName      string  `json:"chain_name"`
	ExpirationTime int64   `json:"expiration_time"`
	PaymentURL     string  `json:"payment_url"`
}

// PaymentResponse is the API response for payment operations
type PaymentResponse struct {
	StatusCode int          `json:"status_code"`
	Message    string       `json:"message"`
	Data       *PaymentData `json:"data"`
	RequestID  string       `json:"request_id"`
}

// OrderData holds order query data
type OrderData struct {
	TradeID            string  `json:"trade_id"`
	OrderID            string  `json:"order_id"`
	Amount             float64 `json:"amount"`
	ActualAmount       float64 `json:"actual_amount"`
	Token              string  `json:"token"`
	ChainType          string  `json:"chain_type"`
	Status             int     `json:"status"`
	BlockTransactionID string  `json:"block_transaction_id"`
	CreatedAt          string  `json:"created_at"`
	PaidAt             string  `json:"paid_at"`
}

// OrderResponse is the API response for order queries
type OrderResponse struct {
	StatusCode int        `json:"status_code"`
	Message    string     `json:"message"`
	Data       *OrderData `json:"data"`
	RequestID  string     `json:"request_id"`
}

// OrderListData holds paginated order list data
type OrderListData struct {
	List     []OrderData `json:"list"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// OrderListResponse is the API response for order list
type OrderListResponse struct {
	StatusCode int            `json:"status_code"`
	Message    string         `json:"message"`
	Data       *OrderListData `json:"data"`
	RequestID  string         `json:"request_id"`
}

// ListOrdersParams holds parameters for listing orders
type ListOrdersParams struct {
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
	Status    int    `json:"status,omitempty"`
	ChainType string `json:"chain_type,omitempty"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
}

// WebhookPayload represents a webhook callback payload
type WebhookPayload struct {
	TradeID            string  `json:"trade_id"`
	OrderID            string  `json:"order_id"`
	Amount             float64 `json:"amount"`
	ActualAmount       float64 `json:"actual_amount"`
	Token              string  `json:"token"`
	ChainType          string  `json:"chain_type"`
	BlockTransactionID string  `json:"block_transaction_id"`
	Status             int     `json:"status"`
	Signature          string  `json:"signature"`
}

// MerchantData holds merchant profile data
type MerchantData struct {
	MerchantID   int    `json:"merchant_id"`
	MerchantCode string `json:"merchant_code"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Status       string `json:"status"`
	KYCStatus    string `json:"kyc_status"`
	KYCLevel     int    `json:"kyc_level"`
	CreatedAt    string `json:"created_at"`
}

// MerchantResponse is the API response for merchant operations
type MerchantResponse struct {
	StatusCode int           `json:"status_code"`
	Message    string        `json:"message"`
	Data       *MerchantData `json:"data"`
	RequestID  string        `json:"request_id"`
}

// CreatePayment creates a new payment order
func (c *Client) CreatePayment(params *CreatePaymentParams) (*PaymentResponse, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := generateNonce()

	// Build params map for signing
	paramsMap := map[string]string{
		"api_key":    c.apiKey,
		"timestamp":  timestamp,
		"nonce":      nonce,
		"order_id":   params.OrderID,
		"amount":     formatAmount(params.Amount),
		"notify_url": params.NotifyURL,
	}

	if params.RedirectURL != "" {
		paramsMap["redirect_url"] = params.RedirectURL
	}
	if params.ChainType != "" {
		paramsMap["chain_type"] = params.ChainType
	}

	// Generate HMAC-SHA256 signature
	signature := c.generateSignature(paramsMap)

	// Build request body
	body := map[string]interface{}{
		"api_key":    c.apiKey,
		"timestamp":  timestamp,
		"nonce":      nonce,
		"order_id":   params.OrderID,
		"amount":     params.Amount,
		"notify_url": params.NotifyURL,
		"signature":  signature,
	}

	if params.RedirectURL != "" {
		body["redirect_url"] = params.RedirectURL
	}
	if params.ChainType != "" {
		body["chain_type"] = params.ChainType
	}

	var resp PaymentResponse
	err := c.request("POST", "/order/create-transaction", body, &resp)
	return &resp, err
}

// QueryPaymentByTradeID queries a payment by trade_id
func (c *Client) QueryPaymentByTradeID(tradeID string) (*OrderResponse, error) {
	var resp OrderResponse
	err := c.request("GET", "/order/query?trade_id="+url.QueryEscape(tradeID), nil, &resp)
	return &resp, err
}

// QueryPaymentByOrderID queries a payment by order_id
func (c *Client) QueryPaymentByOrderID(orderID string) (*OrderResponse, error) {
	var resp OrderResponse
	err := c.request("GET", "/order/query?order_id="+url.QueryEscape(orderID), nil, &resp)
	return &resp, err
}

// ListOrders lists orders with optional filters
func (c *Client) ListOrders(params *ListOrdersParams) (*OrderListResponse, error) {
	query := url.Values{}

	if params.Page > 0 {
		query.Set("page", fmt.Sprintf("%d", params.Page))
	}
	if params.PageSize > 0 {
		query.Set("page_size", fmt.Sprintf("%d", params.PageSize))
	}
	if params.Status > 0 {
		query.Set("status", fmt.Sprintf("%d", params.Status))
	}
	if params.ChainType != "" {
		query.Set("chain_type", params.ChainType)
	}
	if params.StartDate != "" {
		query.Set("start_date", params.StartDate)
	}
	if params.EndDate != "" {
		query.Set("end_date", params.EndDate)
	}

	endpoint := "/merchant/orders"
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var resp OrderListResponse
	err := c.request("GET", endpoint, nil, &resp)
	return &resp, err
}

// GetMerchantInfo gets the merchant profile
func (c *Client) GetMerchantInfo() (*MerchantResponse, error) {
	var resp MerchantResponse
	err := c.request("GET", "/merchant/info", nil, &resp)
	return &resp, err
}

// VerifyWebhookSignature verifies a webhook payload signature
func (c *Client) VerifyWebhookSignature(payload *WebhookPayload) bool {
	params := map[string]string{
		"trade_id":             payload.TradeID,
		"order_id":             payload.OrderID,
		"amount":               formatAmount(payload.Amount),
		"actual_amount":        formatActualAmount(payload.ActualAmount),
		"token":                payload.Token,
		"chain_type":           payload.ChainType,
		"block_transaction_id": payload.BlockTransactionID,
		"status":               fmt.Sprintf("%d", payload.Status),
	}

	expected := c.generateSignature(params)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(payload.Signature)) == 1
}

// VerifyWebhookSignatureFromMap verifies a webhook signature from a map
func (c *Client) VerifyWebhookSignatureFromMap(payload map[string]interface{}) bool {
	signature, ok := payload["signature"].(string)
	if !ok {
		return false
	}

	params := make(map[string]string)
	for k, v := range payload {
		if k != "signature" {
			params[k] = fmt.Sprintf("%v", v)
		}
	}

	expected := c.generateSignature(params)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 1
}

// generateSignature generates HMAC-SHA256 signature
func (c *Client) generateSignature(params map[string]string) string {
	// Get sorted keys (excluding empty values and signature)
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k != "signature" && v != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// Build query string
	var builder strings.Builder
	for i, k := range keys {
		if i > 0 {
			builder.WriteString("&")
		}
		builder.WriteString(k)
		builder.WriteString("=")
		builder.WriteString(params[k])
	}

	// Calculate HMAC-SHA256
	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(builder.String()))
	return hex.EncodeToString(h.Sum(nil))
}

// generateNonce generates a random nonce string
func generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// request makes an HTTP request
func (c *Client) request(method, endpoint string, body interface{}, result interface{}) error {
	var reqBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "cryptomepay-go/"+Version)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

func formatAmount(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}

func formatActualAmount(amount float64) string {
	return fmt.Sprintf("%.4f", amount)
}
