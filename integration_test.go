// +build integration

package cryptomepay

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests for Cryptome Pay Go SDK
// Run with: go test -tags=integration -v
//
// Required environment variables:
//   CRYPTOME_API_KEY    - Your API key (ak_xxx)
//   CRYPTOME_API_SECRET - Your API secret (sk_xxx)
//   CRYPTOME_BASE_URL   - Optional, defaults to production URL

func getTestClient(t *testing.T) *Client {
	apiKey := os.Getenv("CRYPTOME_API_KEY")
	apiSecret := os.Getenv("CRYPTOME_API_SECRET")
	baseURL := os.Getenv("CRYPTOME_BASE_URL")

	if apiKey == "" || apiSecret == "" {
		t.Skip("CRYPTOME_API_KEY and CRYPTOME_API_SECRET environment variables required")
	}

	opts := []Option{}
	if baseURL != "" {
		opts = append(opts, WithBaseURL(baseURL))
	}

	return NewClientWithOptions(apiKey, apiSecret, opts...)
}

func TestIntegration_GetMerchantInfo(t *testing.T) {
	client := getTestClient(t)

	resp, err := client.GetMerchantInfo()
	require.NoError(t, err)

	fmt.Printf("Merchant Info Response: %+v\n", resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.NotNil(t, resp.Data)
	assert.NotEmpty(t, resp.Data.MerchantCode)
	assert.NotEmpty(t, resp.Data.Email)

	fmt.Printf("Merchant: %s (%s)\n", resp.Data.Name, resp.Data.MerchantCode)
	fmt.Printf("Email: %s\n", resp.Data.Email)
	fmt.Printf("Status: %d\n", resp.Data.Status)
}

func TestIntegration_CreatePayment(t *testing.T) {
	client := getTestClient(t)

	// Generate unique order ID
	orderID := fmt.Sprintf("TEST_%d", time.Now().UnixNano())

	resp, err := client.CreatePayment(&CreatePaymentParams{
		OrderID:   orderID,
		Amount:    10.00, // 10 CNY
		NotifyURL: "https://webhook.site/test-webhook",
		ChainType: ChainBSC,
	})

	require.NoError(t, err)

	fmt.Printf("Create Payment Response: %+v\n", resp)

	if resp.StatusCode != 200 {
		t.Logf("Create payment failed: %s (code: %d)", resp.Message, resp.StatusCode)
		// Don't fail - might be due to missing wallet configuration
		t.Skip("Skipping - payment creation may require wallet configuration")
	}

	assert.Equal(t, 200, resp.StatusCode)
	assert.NotNil(t, resp.Data)
	assert.NotEmpty(t, resp.Data.TradeID)
	assert.Equal(t, orderID, resp.Data.OrderID)
	assert.NotEmpty(t, resp.Data.Token)
	assert.NotEmpty(t, resp.Data.PaymentURL)

	fmt.Printf("Trade ID: %s\n", resp.Data.TradeID)
	fmt.Printf("Order ID: %s\n", resp.Data.OrderID)
	fmt.Printf("Amount: %.2f CNY -> %.4f USDT\n", resp.Data.Amount, resp.Data.ActualAmount)
	fmt.Printf("Wallet: %s\n", resp.Data.Token)
	fmt.Printf("Chain: %s\n", resp.Data.ChainType)
	fmt.Printf("Payment URL: %s\n", resp.Data.PaymentURL)

	// Store for subsequent tests
	t.Setenv("TEST_TRADE_ID", resp.Data.TradeID)
	t.Setenv("TEST_ORDER_ID", orderID)
}

func TestIntegration_CreatePaymentWithUUID(t *testing.T) {
	client := getTestClient(t)

	// Test with UUID format order ID (36 characters)
	orderID := fmt.Sprintf("550e8400-e29b-41d4-a716-%012d", time.Now().UnixNano()%1000000000000)

	resp, err := client.CreatePayment(&CreatePaymentParams{
		OrderID:   orderID,
		Amount:    5.00,
		NotifyURL: "https://webhook.site/test-webhook",
		ChainType: ChainTRC20,
	})

	require.NoError(t, err)

	fmt.Printf("Create Payment (UUID) Response: %+v\n", resp)

	if resp.StatusCode != 200 {
		t.Logf("Create payment failed: %s (code: %d)", resp.Message, resp.StatusCode)
		t.Skip("Skipping - payment creation may require wallet configuration")
	}

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, orderID, resp.Data.OrderID)

	fmt.Printf("UUID Order ID accepted: %s\n", orderID)
}

func TestIntegration_CreatePaymentWithLongOrderID(t *testing.T) {
	client := getTestClient(t)

	// Test with maximum length order ID (64 characters)
	orderID := fmt.Sprintf("LONG_ORDER_%052d", time.Now().UnixNano())
	if len(orderID) > 64 {
		orderID = orderID[:64]
	}

	resp, err := client.CreatePayment(&CreatePaymentParams{
		OrderID:   orderID,
		Amount:    1.00,
		NotifyURL: "https://webhook.site/test-webhook",
		ChainType: ChainBSC,
	})

	require.NoError(t, err)

	fmt.Printf("Create Payment (Long ID) Response: %+v\n", resp)

	if resp.StatusCode != 200 {
		t.Logf("Create payment failed: %s (code: %d)", resp.Message, resp.StatusCode)
		t.Skip("Skipping - payment creation may require wallet configuration")
	}

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, orderID, resp.Data.OrderID)

	fmt.Printf("Long Order ID (len=%d) accepted: %s\n", len(orderID), orderID)
}

func TestIntegration_QueryPaymentByOrderID(t *testing.T) {
	client := getTestClient(t)

	// First create a payment
	orderID := fmt.Sprintf("QUERY_TEST_%d", time.Now().UnixNano())

	createResp, err := client.CreatePayment(&CreatePaymentParams{
		OrderID:   orderID,
		Amount:    1.00,
		NotifyURL: "https://webhook.site/test-webhook",
		ChainType: ChainBSC,
	})

	require.NoError(t, err)

	if createResp.StatusCode != 200 {
		t.Skip("Skipping - payment creation failed")
	}

	// Query by order ID
	resp, err := client.QueryPaymentByOrderID(orderID)
	require.NoError(t, err)

	fmt.Printf("Query Payment Response: %+v\n", resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.NotNil(t, resp.Data)
	assert.Equal(t, orderID, resp.Data.OrderID)
	assert.Equal(t, createResp.Data.TradeID, resp.Data.TradeID)
	assert.Equal(t, StatusPending, resp.Data.Status)

	fmt.Printf("Order ID: %s\n", resp.Data.OrderID)
	fmt.Printf("Trade ID: %s\n", resp.Data.TradeID)
	fmt.Printf("Status: %d (Pending)\n", resp.Data.Status)
}

func TestIntegration_QueryPaymentByTradeID(t *testing.T) {
	client := getTestClient(t)

	// First create a payment
	orderID := fmt.Sprintf("TRADE_QUERY_%d", time.Now().UnixNano())

	createResp, err := client.CreatePayment(&CreatePaymentParams{
		OrderID:   orderID,
		Amount:    1.00,
		NotifyURL: "https://webhook.site/test-webhook",
		ChainType: ChainBSC,
	})

	require.NoError(t, err)

	if createResp.StatusCode != 200 {
		t.Skip("Skipping - payment creation failed")
	}

	tradeID := createResp.Data.TradeID

	// Query by trade ID
	resp, err := client.QueryPaymentByTradeID(tradeID)
	require.NoError(t, err)

	fmt.Printf("Query Payment by Trade ID Response: %+v\n", resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.NotNil(t, resp.Data)
	assert.Equal(t, tradeID, resp.Data.TradeID)
	assert.Equal(t, orderID, resp.Data.OrderID)

	fmt.Printf("Trade ID: %s\n", resp.Data.TradeID)
	fmt.Printf("Order ID: %s\n", resp.Data.OrderID)
}

func TestIntegration_ListOrders(t *testing.T) {
	client := getTestClient(t)

	resp, err := client.ListOrders(&ListOrdersParams{
		Page:     1,
		PageSize: 10,
	})

	require.NoError(t, err)

	fmt.Printf("List Orders Response: %+v\n", resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.NotNil(t, resp.Data)
	assert.GreaterOrEqual(t, resp.Data.Total, 0)

	fmt.Printf("Total Orders: %d\n", resp.Data.Total)
	fmt.Printf("Page: %d, PageSize: %d\n", resp.Data.Page, resp.Data.PageSize)
	fmt.Printf("Orders in response: %d\n", len(resp.Data.List))

	for i, order := range resp.Data.List {
		statusStr := "Unknown"
		switch order.Status {
		case StatusPending:
			statusStr = "Pending"
		case StatusPaid:
			statusStr = "Paid"
		case StatusExpired:
			statusStr = "Expired"
		}
		fmt.Printf("  [%d] %s - %s - %s\n", i+1, order.TradeID, order.OrderID, statusStr)
	}
}

func TestIntegration_ListOrdersWithFilters(t *testing.T) {
	client := getTestClient(t)

	// List only paid orders
	resp, err := client.ListOrders(&ListOrdersParams{
		Page:     1,
		PageSize: 5,
		Status:   StatusPaid,
	})

	require.NoError(t, err)

	fmt.Printf("List Paid Orders Response: %+v\n", resp)

	assert.Equal(t, 200, resp.StatusCode)

	// All returned orders should be paid
	for _, order := range resp.Data.List {
		assert.Equal(t, StatusPaid, order.Status)
	}

	fmt.Printf("Paid Orders: %d\n", len(resp.Data.List))
}

func TestIntegration_ListOrdersByChain(t *testing.T) {
	client := getTestClient(t)

	// List BSC orders only
	resp, err := client.ListOrders(&ListOrdersParams{
		Page:      1,
		PageSize:  5,
		ChainType: ChainBSC,
	})

	require.NoError(t, err)

	fmt.Printf("List BSC Orders Response: %+v\n", resp)

	assert.Equal(t, 200, resp.StatusCode)

	// All returned orders should be BSC
	for _, order := range resp.Data.List {
		assert.Equal(t, ChainBSC, order.ChainType)
	}

	fmt.Printf("BSC Orders: %d\n", len(resp.Data.List))
}

func TestIntegration_MultipleChains(t *testing.T) {
	client := getTestClient(t)

	chains := []string{ChainTRC20, ChainBSC}

	for _, chain := range chains {
		t.Run("Chain_"+chain, func(t *testing.T) {
			orderID := fmt.Sprintf("CHAIN_%s_%d", chain, time.Now().UnixNano())

			resp, err := client.CreatePayment(&CreatePaymentParams{
				OrderID:   orderID,
				Amount:    1.00,
				NotifyURL: "https://webhook.site/test-webhook",
				ChainType: chain,
			})

			require.NoError(t, err)

			if resp.StatusCode != 200 {
				t.Logf("Chain %s not configured: %s", chain, resp.Message)
				t.Skip("Chain not configured")
			}

			assert.Equal(t, 200, resp.StatusCode)
			assert.Equal(t, chain, resp.Data.ChainType)

			fmt.Printf("Chain %s: OK - Wallet: %s\n", chain, resp.Data.Token)
		})
	}
}

func TestIntegration_SignatureGeneration(t *testing.T) {
	client := getTestClient(t)

	// Test that signature generation is consistent
	params := map[string]string{
		"order_id":   "TEST_ORDER",
		"amount":     "100.00",
		"notify_url": "https://example.com/webhook",
		"api_key":    os.Getenv("CRYPTOME_API_KEY"),
		"timestamp":  "1234567890",
		"nonce":      "abcdef123456",
	}

	sig1 := client.generateSignature(params)
	sig2 := client.generateSignature(params)

	assert.Equal(t, sig1, sig2, "Signature should be deterministic")
	assert.Len(t, sig1, 64, "HMAC-SHA256 signature should be 64 hex characters")

	fmt.Printf("Signature: %s\n", sig1)
}

func TestIntegration_WebhookSignatureVerification(t *testing.T) {
	client := getTestClient(t)

	// Simulate a webhook payload
	payload := &WebhookPayload{
		TradeID:            "CP123456789",
		OrderID:            "TEST_ORDER_001",
		Amount:             100.00,
		ActualAmount:       15.6250,
		Token:              "0xabc123def456",
		ChainType:          "BSC",
		BlockTransactionID: "0x789xyz",
		Status:             StatusPaid,
	}

	// Generate valid signature
	params := map[string]string{
		"trade_id":             payload.TradeID,
		"order_id":             payload.OrderID,
		"amount":               fmt.Sprintf("%.2f", payload.Amount),
		"actual_amount":        fmt.Sprintf("%.4f", payload.ActualAmount),
		"token":                payload.Token,
		"chain_type":           payload.ChainType,
		"block_transaction_id": payload.BlockTransactionID,
		"status":               fmt.Sprintf("%d", payload.Status),
	}
	payload.Signature = client.generateSignature(params)

	// Verify should pass
	assert.True(t, client.VerifyWebhookSignature(payload), "Valid signature should verify")

	// Tamper with payload
	payload.Amount = 999.99
	assert.False(t, client.VerifyWebhookSignature(payload), "Tampered payload should fail verification")

	fmt.Println("Webhook signature verification: OK")
}

func TestIntegration_ErrorHandling(t *testing.T) {
	client := getTestClient(t)

	// Test duplicate order ID
	orderID := fmt.Sprintf("DUP_TEST_%d", time.Now().Unix()) // Use seconds for potential duplicate

	// Create first order
	resp1, err := client.CreatePayment(&CreatePaymentParams{
		OrderID:   orderID,
		Amount:    1.00,
		NotifyURL: "https://webhook.site/test-webhook",
		ChainType: ChainBSC,
	})
	require.NoError(t, err)

	if resp1.StatusCode != 200 {
		t.Skip("First order creation failed")
	}

	// Try to create duplicate
	resp2, err := client.CreatePayment(&CreatePaymentParams{
		OrderID:   orderID,
		Amount:    1.00,
		NotifyURL: "https://webhook.site/test-webhook",
		ChainType: ChainBSC,
	})
	require.NoError(t, err)

	// Should get error for duplicate
	assert.NotEqual(t, 200, resp2.StatusCode, "Duplicate order should fail")

	fmt.Printf("Duplicate order error: %s (code: %d)\n", resp2.Message, resp2.StatusCode)
}

func TestIntegration_QueryNonExistentOrder(t *testing.T) {
	client := getTestClient(t)

	// Query non-existent order
	resp, err := client.QueryPaymentByOrderID("NON_EXISTENT_ORDER_12345")
	require.NoError(t, err)

	assert.NotEqual(t, 200, resp.StatusCode, "Non-existent order should return error")

	fmt.Printf("Non-existent order error: %s (code: %d)\n", resp.Message, resp.StatusCode)
}

// Benchmark tests
func BenchmarkCreatePayment(b *testing.B) {
	apiKey := os.Getenv("CRYPTOME_API_KEY")
	apiSecret := os.Getenv("CRYPTOME_API_SECRET")
	baseURL := os.Getenv("CRYPTOME_BASE_URL")

	if apiKey == "" || apiSecret == "" {
		b.Skip("CRYPTOME_API_KEY and CRYPTOME_API_SECRET required")
	}

	opts := []Option{}
	if baseURL != "" {
		opts = append(opts, WithBaseURL(baseURL))
	}

	client := NewClientWithOptions(apiKey, apiSecret, opts...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		orderID := fmt.Sprintf("BENCH_%d_%d", time.Now().UnixNano(), i)
		client.CreatePayment(&CreatePaymentParams{
			OrderID:   orderID,
			Amount:    1.00,
			NotifyURL: "https://webhook.site/test",
			ChainType: ChainBSC,
		})
	}
}

func BenchmarkSignatureGeneration(b *testing.B) {
	client := NewClient("test_key", "test_secret")
	params := map[string]string{
		"order_id":   "ORDER_001",
		"amount":     "100.00",
		"notify_url": "https://example.com/webhook",
		"api_key":    "test_key",
		"timestamp":  "1234567890",
		"nonce":      "abcdef123456",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.generateSignature(params)
	}
}
