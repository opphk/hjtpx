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
	ErrCaptchaExpired     = errors.New("captcha expired")
	ErrInvalidCaptchaType = errors.New("invalid captcha type")
	ErrEmptyChallengeID   = errors.New("challenge ID is empty")
	ErrEmptyAnswer        = errors.New("answer is empty")
	ErrInvalidTrajectory  = errors.New("invalid trajectory data")
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
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

func GetSDKErrorMessage(err error) string {
	var sdkErr *SDKError
	if errors.As(err, &sdkErr) {
		return sdkErr.Message
	}
	return ""
}

func HandleError(err error) {
	if err == nil {
		return
	}

	if IsSDKError(err) {
		code := GetSDKErrorCode(err)
		msg := GetSDKErrorMessage(err)
		switch code {
		case http.StatusUnauthorized:
			fmt.Println("Authentication failed - check your credentials")
		case http.StatusTooManyRequests:
			fmt.Println("Rate limit exceeded - please wait before retrying")
		case http.StatusInternalServerError:
			fmt.Println("Server error - please try again later")
		case http.StatusBadRequest:
			fmt.Println("Invalid request parameters")
		case 400:
			if strings.Contains(strings.ToLower(msg), "required") {
				fmt.Printf("Missing required field: %s\n", msg)
			} else {
				fmt.Printf("Invalid parameter: %s\n", msg)
			}
		default:
			fmt.Printf("API Error (code %d): %v\n", code, err)
		}
	} else if errors.Is(err, ErrTimeout) {
		fmt.Println("Request timed out - please check your network")
	} else if errors.Is(err, ErrNetworkError) {
		fmt.Println("Network error - please check your connection")
	} else if errors.Is(err, ErrCaptchaExpired) {
		fmt.Println("Captcha has expired - please request a new one")
	} else if errors.Is(err, ErrMaxRetriesExceeded) {
		fmt.Println("Max retries exceeded - please try again later")
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

func (e *SDKError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

func ValidateChallengeID(challengeID string) error {
	if challengeID == "" {
		return NewSDKError(400, "challenge ID is required")
	}
	return nil
}

func ValidateAnswer(answer string) error {
	if answer == "" {
		return NewSDKError(400, "answer is required")
	}
	return nil
}

func ValidateTrajectory(trajectory []TrajectoryPoint) error {
	if len(trajectory) == 0 {
		return NewSDKError(400, "trajectory must contain at least one point")
	}
	for i, point := range trajectory {
		if point.T < 0 {
			return fmt.Errorf("invalid timestamp at index %d: %w", i, ErrInvalidTrajectory)
		}
	}
	return nil
}
