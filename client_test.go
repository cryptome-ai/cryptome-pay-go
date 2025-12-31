package cryptomepay

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	client := NewClient("sk_test_key", "test_secret")

	assert.NotNil(t, client)
	assert.Equal(t, "sk_test_key", client.apiKey)
	assert.Equal(t, "test_secret", client.apiSecret)
	assert.Equal(t, ProductionURL, client.baseURL)
}

func TestClientWithOptions(t *testing.T) {
	client := NewClientWithOptions(
		"sk_test_key",
		"test_secret",
		WithBaseURL(SandboxURL),
	)

	assert.Equal(t, SandboxURL, client.baseURL)
}

func TestUseSandbox(t *testing.T) {
	client := NewClient("sk_test_key", "test_secret")
	client.UseSandbox()

	assert.Equal(t, SandboxURL, client.baseURL)
}

func TestGenerateSignature(t *testing.T) {
	client := NewClient("sk_test_key", "test_secret")

	params := map[string]string{
		"order_id":   "ORDER_001",
		"amount":     "100.00",
		"notify_url": "https://example.com/webhook",
		"chain_type": "BSC",
	}

	signature := client.generateSignature(params)

	// Signature should be 32 character hex string
	assert.Len(t, signature, 32)

	// Same params should produce same signature
	signature2 := client.generateSignature(params)
	assert.Equal(t, signature, signature2)
}

func TestGenerateSignatureOrder(t *testing.T) {
	client := NewClient("sk_test_key", "test_secret")

	// Different order, same result
	params1 := map[string]string{
		"order_id":   "ORDER_001",
		"amount":     "100.00",
		"notify_url": "https://example.com/webhook",
	}

	params2 := map[string]string{
		"notify_url": "https://example.com/webhook",
		"order_id":   "ORDER_001",
		"amount":     "100.00",
	}

	assert.Equal(t, client.generateSignature(params1), client.generateSignature(params2))
}

func TestCreatePayment(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/order/create-transaction", r.URL.Path)
		assert.Equal(t, "Bearer sk_test_key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Decode request body
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		assert.Equal(t, "ORDER_001", body["order_id"])
		assert.Equal(t, float64(100), body["amount"])
		assert.NotEmpty(t, body["signature"])

		// Return mock response
		resp := PaymentResponse{
			StatusCode: 200,
			Message:    "success",
			Data: &PaymentData{
				TradeID:      "CP123456789",
				OrderID:      "ORDER_001",
				Amount:       100,
				ActualAmount: 15.6250,
				Token:        "0xabc123",
				ChainType:    "BSC",
				PaymentURL:   "https://pay.example.com/CP123456789",
			},
			RequestID: "req_123",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client with mock server
	client := NewClientWithOptions(
		"sk_test_key",
		"test_secret",
		WithBaseURL(server.URL),
	)

	// Test CreatePayment
	payment, err := client.CreatePayment(&CreatePaymentParams{
		OrderID:   "ORDER_001",
		Amount:    100.00,
		NotifyURL: "https://example.com/webhook",
		ChainType: ChainBSC,
	})

	assert.NoError(t, err)
	assert.Equal(t, 200, payment.StatusCode)
	assert.Equal(t, "CP123456789", payment.Data.TradeID)
	assert.Equal(t, 15.6250, payment.Data.ActualAmount)
}

func TestQueryPayment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/order/query")

		resp := OrderResponse{
			StatusCode: 200,
			Message:    "success",
			Data: &OrderData{
				TradeID:            "CP123456789",
				OrderID:            "ORDER_001",
				Status:             StatusPaid,
				BlockTransactionID: "0xdef456",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithOptions("sk_test_key", "test_secret", WithBaseURL(server.URL))

	result, err := client.QueryPaymentByTradeID("CP123456789")

	assert.NoError(t, err)
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, StatusPaid, result.Data.Status)
}

func TestVerifyWebhookSignature(t *testing.T) {
	client := NewClient("sk_test_key", "test_secret")

	// First, generate a valid signature
	params := map[string]string{
		"trade_id":             "CP123",
		"order_id":             "ORDER_001",
		"amount":               "100.00",
		"actual_amount":        "15.6250",
		"token":                "0xabc",
		"chain_type":           "BSC",
		"block_transaction_id": "0x123",
		"status":               "2",
	}
	validSignature := client.generateSignature(params)

	// Test with valid signature
	payload := &WebhookPayload{
		TradeID:            "CP123",
		OrderID:            "ORDER_001",
		Amount:             100.00,
		ActualAmount:       15.6250,
		Token:              "0xabc",
		ChainType:          "BSC",
		BlockTransactionID: "0x123",
		Status:             2,
		Signature:          validSignature,
	}

	assert.True(t, client.VerifyWebhookSignature(payload))

	// Test with invalid signature
	payload.Signature = "invalid_signature_here"
	assert.False(t, client.VerifyWebhookSignature(payload))
}

func TestListOrders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/merchant/orders")
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		assert.Equal(t, "20", r.URL.Query().Get("page_size"))

		resp := OrderListResponse{
			StatusCode: 200,
			Message:    "success",
			Data: &OrderListData{
				List: []OrderData{
					{TradeID: "CP1", OrderID: "O1", Status: StatusPaid},
					{TradeID: "CP2", OrderID: "O2", Status: StatusPending},
				},
				Total:    100,
				Page:     1,
				PageSize: 20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithOptions("sk_test_key", "test_secret", WithBaseURL(server.URL))

	result, err := client.ListOrders(&ListOrdersParams{
		Page:     1,
		PageSize: 20,
	})

	assert.NoError(t, err)
	assert.Equal(t, 200, result.StatusCode)
	assert.Len(t, result.Data.List, 2)
	assert.Equal(t, 100, result.Data.Total)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "TRC20", ChainTRC20)
	assert.Equal(t, "BSC", ChainBSC)
	assert.Equal(t, "POLYGON", ChainPolygon)
	assert.Equal(t, "ETH", ChainETH)
	assert.Equal(t, "ARBITRUM", ChainArbitrum)

	assert.Equal(t, 1, StatusPending)
	assert.Equal(t, 2, StatusPaid)
	assert.Equal(t, 3, StatusExpired)
}
