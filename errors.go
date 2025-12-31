package cryptomepay

import "fmt"

// Error codes
const (
	ErrCodeInvalidAPIKey        = 1001
	ErrCodeSignatureVerifyFailed = 1002
	ErrCodeAPIKeyExpired        = 1003
	ErrCodeIPNotWhitelisted     = 1004
	ErrCodeMerchantSuspended    = 1005

	ErrCodeInvalidOrderID       = 10001
	ErrCodeOrderExists          = 10002
	ErrCodeNoAvailableWallet    = 10003
	ErrCodeInvalidAmount        = 10004
	ErrCodeAmountChannelUnavail = 10005
	ErrCodeExchangeRateError    = 10006
	ErrCodeOrderAlreadyPaid     = 10007
	ErrCodeOrderNotFound        = 10008
	ErrCodeOrderExpired         = 10009

	ErrCodeInvalidChainType     = 20001
	ErrCodeChainUnavailable     = 20002
	ErrCodeChainMonitoringDelay = 20003

	ErrCodeRateLimitExceeded = 50001
	ErrCodeBurstLimitExceeded = 50002
)

// APIError represents an API error response
type APIError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("cryptomepay: %s (code=%d, request_id=%s)", e.Message, e.StatusCode, e.RequestID)
}

// IsRetryable returns true if the error can be retried
func (e *APIError) IsRetryable() bool {
	// Rate limit and server errors are retryable
	return e.StatusCode == 429 || e.StatusCode >= 500
}

// IsAuthError returns true if the error is an authentication error
func (e *APIError) IsAuthError() bool {
	return e.StatusCode >= 1001 && e.StatusCode <= 1005
}

// IsValidationError returns true if the error is a validation error
func (e *APIError) IsValidationError() bool {
	return e.StatusCode >= 10001 && e.StatusCode <= 10009
}

// IsChainError returns true if the error is a chain-related error
func (e *APIError) IsChainError() bool {
	return e.StatusCode >= 20001 && e.StatusCode <= 20003
}

// NewAPIError creates a new API error from a response
func NewAPIError(statusCode int, message, requestID string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		RequestID:  requestID,
	}
}
