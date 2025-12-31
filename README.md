# Cryptome Pay Go SDK

Official Go SDK for [Cryptome Pay](https://cryptomepay.com) - Non-custodial cryptocurrency payment gateway.

## Installation

```bash
go get github.com/cryptome-ai/cryptome-pay-go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "time"

    cryptomepay "github.com/cryptome-ai/cryptome-pay-go"
)

func main() {
    // Create client
    client := cryptomepay.NewClient(
        "sk_live_your_api_key",
        "your_api_secret",
    )

    // Create payment
    payment, err := client.CreatePayment(&cryptomepay.CreatePaymentParams{
        OrderID:   fmt.Sprintf("ORDER_%d", time.Now().Unix()),
        Amount:    100.00,
        NotifyURL: "https://your-site.com/webhook",
        ChainType: cryptomepay.ChainBSC,
    })

    if err != nil {
        log.Fatal(err)
    }

    if payment.StatusCode == 200 {
        fmt.Println("Payment URL:", payment.Data.PaymentURL)
        fmt.Printf("Pay %.4f USDT to %s\n", payment.Data.ActualAmount, payment.Data.Token)
    } else {
        fmt.Println("Error:", payment.Message)
    }
}
```

## Configuration

### Using Options

```go
// With custom base URL (for sandbox)
client := cryptomepay.NewClientWithOptions(
    "sk_sandbox_xxx",
    "secret",
    cryptomepay.WithBaseURL(cryptomepay.SandboxURL),
    cryptomepay.WithTimeout(60 * time.Second),
)

// Or switch environments
client := cryptomepay.NewClient("key", "secret")
client.UseSandbox() // Switch to sandbox
client.UseProduction() // Switch back to production
```

### Environments

| Environment | Base URL | Description |
|-------------|----------|-------------|
| Production | `https://api.cryptomepay.com/api/v1` | Live transactions |
| Sandbox | `https://sandbox.cryptomepay.com/api/v1` | Testing with mock data |
| Staging | `https://staging.cryptomepay.com/api/v1` | Testing with testnet |

## API Reference

### Create Payment

```go
payment, err := client.CreatePayment(&cryptomepay.CreatePaymentParams{
    OrderID:     "ORDER_001",
    Amount:      100.00,           // CNY amount
    NotifyURL:   "https://...",    // Webhook URL
    RedirectURL: "https://...",    // Optional: redirect after payment
    ChainType:   cryptomepay.ChainBSC, // Optional: TRC20, BSC, POLYGON, ETH, ARBITRUM
})
```

### Query Payment

```go
// By trade_id
result, err := client.QueryPaymentByTradeID("CP202312271648380592")

// By order_id
result, err := client.QueryPaymentByOrderID("ORDER_001")

if result.Data.Status == cryptomepay.StatusPaid {
    fmt.Println("Payment confirmed!")
}
```

### List Orders

```go
orders, err := client.ListOrders(&cryptomepay.ListOrdersParams{
    Page:      1,
    PageSize:  20,
    Status:    cryptomepay.StatusPaid, // Optional filter
    ChainType: cryptomepay.ChainBSC,   // Optional filter
    StartDate: "2025-12-01",           // Optional filter
    EndDate:   "2025-12-31",           // Optional filter
})

for _, order := range orders.Data.List {
    fmt.Printf("%s: %.4f USDT\n", order.OrderID, order.ActualAmount)
}
```

### Get Merchant Info

```go
merchant, err := client.GetMerchantInfo()
fmt.Println("Merchant:", merchant.Data.Name)
```

## Webhook Handling

### Verify Signature

```go
func webhookHandler(w http.ResponseWriter, r *http.Request) {
    var payload cryptomepay.WebhookPayload
    json.NewDecoder(r.Body).Decode(&payload)

    // Verify signature
    if !client.VerifyWebhookSignature(&payload) {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }

    // Process payment
    if payload.Status == cryptomepay.StatusPaid {
        // Order paid!
        processOrder(payload.OrderID, payload.BlockTransactionID)
    }

    w.Write([]byte("ok"))
}
```

### From Map (for raw JSON)

```go
var payload map[string]interface{}
json.NewDecoder(r.Body).Decode(&payload)

if !client.VerifyWebhookSignatureFromMap(payload) {
    // Invalid signature
}
```

## Supported Chains

| Constant | Chain | Network |
|----------|-------|---------|
| `ChainTRC20` | TRON | TRC20 USDT |
| `ChainBSC` | BNB Smart Chain | BEP20 USDT |
| `ChainPolygon` | Polygon | USDT |
| `ChainETH` | Ethereum | ERC20 USDT |
| `ChainArbitrum` | Arbitrum One | USDT |

## Payment Status

| Constant | Value | Description |
|----------|-------|-------------|
| `StatusPending` | 1 | Awaiting payment |
| `StatusPaid` | 2 | Payment confirmed |
| `StatusExpired` | 3 | Payment expired |

## Error Handling

```go
payment, err := client.CreatePayment(params)
if err != nil {
    // Network or parsing error
    log.Fatal(err)
}

if payment.StatusCode != 200 {
    // API error
    switch payment.StatusCode {
    case cryptomepay.ErrCodeOrderExists:
        // Order already exists
    case cryptomepay.ErrCodeInvalidAmount:
        // Invalid amount
    default:
        fmt.Println("Error:", payment.Message)
    }
}
```

## Framework Examples

### Gin

```go
import "github.com/gin-gonic/gin"

func main() {
    r := gin.Default()

    r.POST("/api/payments", func(c *gin.Context) {
        payment, _ := client.CreatePayment(&cryptomepay.CreatePaymentParams{
            OrderID:   "ORDER_" + uuid.New().String(),
            Amount:    100.00,
            NotifyURL: "https://example.com/webhook",
        })

        c.JSON(200, gin.H{"payment_url": payment.Data.PaymentURL})
    })

    r.POST("/webhook", func(c *gin.Context) {
        var payload cryptomepay.WebhookPayload
        c.BindJSON(&payload)

        if !client.VerifyWebhookSignature(&payload) {
            c.String(401, "Invalid signature")
            return
        }

        // Process...
        c.String(200, "ok")
    })

    r.Run(":8080")
}
```

### Fiber

```go
import "github.com/gofiber/fiber/v2"

func main() {
    app := fiber.New()

    app.Post("/webhook", func(c *fiber.Ctx) error {
        var payload cryptomepay.WebhookPayload
        c.BodyParser(&payload)

        if !client.VerifyWebhookSignature(&payload) {
            return c.Status(401).SendString("Invalid signature")
        }

        return c.SendString("ok")
    })

    app.Listen(":3000")
}
```

## Testing

Run tests:

```bash
go test -v ./...
```

## Documentation

- [API Reference](https://docs.cryptomepay.com/api)
- [Webhooks Guide](https://docs.cryptomepay.com/api/WEBHOOKS.md)
- [Error Codes](https://docs.cryptomepay.com/api/ERROR_CODES.md)

## License

MIT License - see [LICENSE](LICENSE) file.

## Support

- Email: support@cryptomepay.com
- GitHub: https://github.com/cryptome-ai/cryptome-pay-go/issues
