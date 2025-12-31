package cryptomepay_test

import (
	"fmt"
	"time"

	cryptomepay "github.com/cryptome-ai/cryptome-pay-go"
)

func ExampleNewClient() {
	client := cryptomepay.NewClient(
		"sk_live_your_api_key",
		"your_api_secret",
	)

	fmt.Println(client != nil)
	// Output: true
}

func ExampleClient_CreatePayment() {
	client := cryptomepay.NewClient("sk_live_xxx", "secret")

	payment, err := client.CreatePayment(&cryptomepay.CreatePaymentParams{
		OrderID:   fmt.Sprintf("ORDER_%d", time.Now().Unix()),
		Amount:    100.00,
		NotifyURL: "https://example.com/webhook",
		ChainType: cryptomepay.ChainBSC,
	})

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if payment.StatusCode == 200 {
		fmt.Println("Trade ID:", payment.Data.TradeID)
		fmt.Println("Payment URL:", payment.Data.PaymentURL)
	}
}

func ExampleClient_QueryPaymentByTradeID() {
	client := cryptomepay.NewClient("sk_live_xxx", "secret")

	result, err := client.QueryPaymentByTradeID("CP202312271648380592")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	switch result.Data.Status {
	case cryptomepay.StatusPending:
		fmt.Println("Payment pending...")
	case cryptomepay.StatusPaid:
		fmt.Println("Payment confirmed!")
		fmt.Println("TX:", result.Data.BlockTransactionID)
	case cryptomepay.StatusExpired:
		fmt.Println("Payment expired")
	}
}

func ExampleClient_ListOrders() {
	client := cryptomepay.NewClient("sk_live_xxx", "secret")

	orders, err := client.ListOrders(&cryptomepay.ListOrdersParams{
		Page:     1,
		PageSize: 20,
		Status:   cryptomepay.StatusPaid,
	})

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, order := range orders.Data.List {
		fmt.Printf("%s: %.4f USDT\n", order.OrderID, order.ActualAmount)
	}

	fmt.Printf("Total: %d orders\n", orders.Data.Total)
}

func ExampleClient_VerifyWebhookSignature() {
	client := cryptomepay.NewClient("sk_live_xxx", "secret")

	payload := &cryptomepay.WebhookPayload{
		TradeID:            "CP123",
		OrderID:            "ORDER_001",
		Amount:             100.00,
		ActualAmount:       15.6250,
		Token:              "0xabc",
		ChainType:          "BSC",
		BlockTransactionID: "0x123",
		Status:             2,
		Signature:          "received_signature",
	}

	if client.VerifyWebhookSignature(payload) {
		fmt.Println("Signature valid")
		// Process payment...
	} else {
		fmt.Println("Invalid signature!")
	}
}

func ExampleClient_UseSandbox() {
	client := cryptomepay.NewClient("sk_sandbox_xxx", "secret")
	client.UseSandbox()

	// Now using sandbox environment
	payment, _ := client.CreatePayment(&cryptomepay.CreatePaymentParams{
		OrderID:   "TEST_001",
		Amount:    100.01, // Auto-success amount in sandbox
		NotifyURL: "https://webhook.site/test",
	})

	if payment != nil && payment.StatusCode == 200 {
		fmt.Println("Sandbox payment created")
	}
}
