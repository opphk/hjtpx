package captcha

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var (
	ErrNetworkError        = errors.New("network error")
	ErrTimeout            = errors.New("request timeout")
	ErrInvalidResponse    = errors.New("invalid response")
	ErrServerError        = errors.New("server error")
	ErrInvalidParams      = errors.New("invalid parameters")
	ErrVerificationFailed = errors.New("verification failed")
	ErrRateLimited        = errors.New("rate limited")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInternalError      = errors.New("internal error")
)

type SDKError struct {
	Code    int
	Message string
	Err     error
}

func (e *SDKError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("SDK Error %d: %s - %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("SDK Error %d: %s", e.Code, e.Message)
}

func (e *SDKError) Unwrap() error {
	return e.Err
}

func IsSDKError(err error) bool {
	var sdkErr *SDKError
	return errors.As(err, &sdkErr)
}

func GetSDKErrorCode(err error) int {
	var sdkErr *SDKError
	if errors.As(err, &sdkErr) {
		return sdkErr.Code
	}
	return 0
}

func HandleError(err error) {
	if err == nil {
		return
	}

	if IsSDKError(err) {
		code := GetSDKErrorCode(err)
		switch code {
		case http.StatusUnauthorized:
			fmt.Println("Authentication failed - check your credentials")
		case http.StatusTooManyRequests:
			fmt.Println("Rate limit exceeded - please wait before retrying")
		case http.StatusInternalServerError:
			fmt.Println("Server error - please try again later")
		case http.StatusBadRequest:
			fmt.Println("Invalid request parameters")
		default:
			fmt.Printf("API Error (code %d): %v\n", code, err)
		}
	} else if errors.Is(err, ErrTimeout) {
		fmt.Println("Request timed out - please check your network")
	} else if errors.Is(err, ErrNetworkError) {
		fmt.Println("Network error - please check your connection")
	} else {
		fmt.Printf("Unexpected error: %v\n", err)
	}
}

func WrapError(code int, message string, err error) error {
	return &SDKError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func ClassifyError(statusCode int, responseBody string) error {
	switch {
	case statusCode >= 500:
		return WrapError(statusCode, "Server error", ErrServerError)
	case statusCode == 429:
		return WrapError(statusCode, "Rate limited", ErrRateLimited)
	case statusCode == 401:
		return WrapError(statusCode, "Unauthorized", ErrUnauthorized)
	case statusCode == 400:
		if strings.Contains(strings.ToLower(responseBody), "invalid") {
			return WrapError(statusCode, "Invalid parameters", ErrInvalidParams)
		}
		return WrapError(statusCode, "Bad request", ErrInvalidParams)
	default:
		return WrapError(statusCode, "Request failed", ErrInternalError)
	}
}

func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	if IsSDKError(err) {
		code := GetSDKErrorCode(err)
		return code == http.StatusTooManyRequests ||
			code == http.StatusInternalServerError ||
			code == http.StatusBadGateway ||
			code == http.StatusServiceUnavailable ||
			code == http.StatusGatewayTimeout
	}

	return errors.Is(err, ErrNetworkError) || errors.Is(err, ErrTimeout)
}

func RetryStrategy(attempt int, baseDelay time.Duration) time.Duration {
	return baseDelay * time.Duration(1<<uint(attempt-1))
}

const (
	StatusOK               = 0
	StatusInvalidParams    = 400
	StatusUnauthorized     = 401
	StatusForbidden        = 403
	StatusNotFound         = 404
	StatusMethodNotAllowed = 405
	StatusTimeout          = 408
	StatusConflict         = 409
	StatusRateLimited      = 429
	StatusInternalError    = 500
	StatusBadGateway       = 502
	StatusUnavailable      = 503
	StatusTimeoutError     = 504
)

func NewSDKError(code int, message string) error {
	return &SDKError{
		Code:    code,
		Message: message,
	}
}

func NewSDKErrorWithCause(code int, message string, cause error) error {
	return &SDKError{
		Code:    code,
		Message: message,
		Err:     cause,
	}
}
