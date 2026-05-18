package captcha

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestBatchClientCreation(t *testing.T) {
	client := NewClient("http://localhost:8080")
	batchClient := NewBatchClient(client, 5, 3)

	if batchClient.GetWorkerPoolSize() != 5 {
		t.Errorf("Expected worker pool size 5, got %d", batchClient.GetWorkerPoolSize())
	}

	if batchClient.GetMaxRetries() != 3 {
		t.Errorf("Expected max retries 3, got %d", batchClient.GetMaxRetries())
	}
}

func TestBatchClientDefaultValues(t *testing.T) {
	client := NewClient("http://localhost:8080")
	batchClient := NewBatchClient(client, 0, -1)

	if batchClient.GetWorkerPoolSize() != 5 {
		t.Errorf("Expected default worker pool size 5, got %d", batchClient.GetWorkerPoolSize())
	}

	if batchClient.GetMaxRetries() != 0 {
		t.Errorf("Expected default max retries 0, got %d", batchClient.GetMaxRetries())
	}
}

func TestBatchClientSetOptions(t *testing.T) {
	client := NewClient("http://localhost:8080")
	batchClient := NewBatchClient(client, 5, 3)

	batchClient.SetWorkerPoolSize(10)
	if batchClient.GetWorkerPoolSize() != 10 {
		t.Errorf("Expected worker pool size 10, got %d", batchClient.GetWorkerPoolSize())
	}

	batchClient.SetMaxRetries(5)
	if batchClient.GetMaxRetries() != 5 {
		t.Errorf("Expected max retries 5, got %d", batchClient.GetMaxRetries())
	}
}

func TestBatchVerifyEmptyRequests(t *testing.T) {
	client := NewClient("http://localhost:8080")
	batchClient := NewBatchClient(client, 5, 0)

	ctx := context.Background()
	requests := []BatchRequest{}
	result := batchClient.BatchVerify(ctx, requests)

	if result.Total != 0 {
		t.Errorf("Expected total 0, got %d", result.Total)
	}

	if result.Successful != 0 {
		t.Errorf("Expected successful 0, got %d", result.Successful)
	}

	if result.Failed != 0 {
		t.Errorf("Expected failed 0, got %d", result.Failed)
	}
}

func TestBatchVerifyContextCancellation(t *testing.T) {
	client := NewClient("http://localhost:8080")
	batchClient := NewBatchClient(client, 5, 0)

	ctx, cancel := context.WithCancel(context.Background())

	requests := []BatchRequest{
		{SessionID: "test-session-1", Type: "slider", Data: map[string]interface{}{"x": 100}},
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	result := batchClient.BatchVerify(ctx, requests)

	if result.Total != 1 {
		t.Errorf("Expected total 1, got %d", result.Total)
	}
}

func TestSDKErrorCreation(t *testing.T) {
	err := NewSDKError(400, "Bad request")

	sdkErr, ok := err.(*SDKError)
	if !ok {
		t.Errorf("Expected *SDKError, got %T", err)
		return
	}

	if sdkErr.Code != 400 {
		t.Errorf("Expected code 400, got %d", sdkErr.Code)
	}

	if sdkErr.Message != "Bad request" {
		t.Errorf("Expected message 'Bad request', got '%s'", sdkErr.Message)
	}

	expected := "SDK Error 400: Bad request"
	if sdkErr.Error() != expected {
		t.Errorf("Expected error string '%s', got '%s'", expected, sdkErr.Error())
	}
}

func TestSDKErrorWithCause(t *testing.T) {
	cause := NewSDKError(500, "Internal error")
	err := NewSDKErrorWithCause(400, "Bad request", cause)

	sdkErr, ok := err.(*SDKError)
	if !ok {
		t.Errorf("Expected *SDKError, got %T", err)
		return
	}

	if sdkErr.Code != 400 {
		t.Errorf("Expected code 400, got %d", sdkErr.Code)
	}

	if sdkErr.Err != cause {
		t.Errorf("Expected cause to be set correctly")
	}

	unwrapped := sdkErr.Unwrap()
	if unwrapped != cause {
		t.Errorf("Expected unwrapped error to be the cause")
	}
}

func TestSDKErrorFromResponse(t *testing.T) {
	err := NewSDKErrorFromResponse(401, "Unauthorized")

	if err.Code != 401 {
		t.Errorf("Expected code 401, got %d", err.Code)
	}

	if err.Message != "Unauthorized" {
		t.Errorf("Expected message 'Unauthorized', got '%s'", err.Message)
	}
}

func TestRetryStrategy(t *testing.T) {
	tests := []struct {
		attempt    int
		baseDelay  time.Duration
		expected   time.Duration
	}{
		{1, 100 * time.Millisecond, 100 * time.Millisecond},
		{2, 100 * time.Millisecond, 200 * time.Millisecond},
		{3, 100 * time.Millisecond, 400 * time.Millisecond},
		{4, 100 * time.Millisecond, 800 * time.Millisecond},
	}

	for _, tt := range tests {
		result := RetryStrategy(tt.attempt, tt.baseDelay)
		if result != tt.expected {
			t.Errorf("RetryStrategy(%d, %v) = %v, want %v",
				tt.attempt, tt.baseDelay, result, tt.expected)
		}
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		statusCode   int
		responseBody string
		expectCode   int
		expectMsg    string
	}{
		{500, "internal server error", 500, "Server error"},
		{502, "bad gateway", 502, "Server error"},
		{429, "rate limited", 429, "Rate limited"},
		{401, "unauthorized", 401, "Unauthorized"},
		{400, "invalid parameters", 400, "Invalid parameters"},
		{400, "some error", 400, "Bad request"},
		{404, "not found", 404, "Request failed"},
	}

	for _, tt := range tests {
		err := ClassifyError(tt.statusCode, tt.responseBody)
		sdkErr, ok := err.(*SDKError)
		if !ok {
			t.Errorf("Expected *SDKError, got %T", err)
			continue
		}

		if sdkErr.Code != tt.expectCode {
			t.Errorf("ClassifyError(%d, '%s'): expected code %d, got %d",
				tt.statusCode, tt.responseBody, tt.expectCode, sdkErr.Code)
		}
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		err           error
		expectRetryable bool
	}{
		{WrapError(429, "Rate limited", nil), true},
		{WrapError(500, "Internal error", nil), true},
		{WrapError(502, "Bad gateway", nil), true},
		{WrapError(503, "Service unavailable", nil), true},
		{WrapError(504, "Gateway timeout", nil), true},
		{WrapError(400, "Bad request", nil), false},
		{WrapError(401, "Unauthorized", nil), false},
		{WrapError(404, "Not found", nil), false},
		{ErrNetworkError, true},
		{ErrTimeout, true},
		{nil, false},
	}

	for _, tt := range tests {
		result := IsRetryableError(tt.err)
		if result != tt.expectRetryable {
			t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, result, tt.expectRetryable)
		}
	}
}

func TestWrapError(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := WrapError(400, "Test error", cause)

	sdkErr, ok := err.(*SDKError)
	if !ok {
		t.Errorf("Expected *SDKError, got %T", err)
		return
	}

	if sdkErr.Code != 400 {
		t.Errorf("Expected code 400, got %d", sdkErr.Code)
	}

	if sdkErr.Message != "Test error" {
		t.Errorf("Expected message 'Test error', got '%s'", sdkErr.Message)
	}

	if sdkErr.Err != cause {
		t.Errorf("Expected cause to be set correctly")
	}
}

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected int
	}{
		{"StatusOK", 0},
		{"StatusInvalidParams", 400},
		{"StatusUnauthorized", 401},
		{"StatusForbidden", 403},
		{"StatusNotFound", 404},
		{"StatusMethodNotAllowed", 405},
		{"StatusTimeout", 408},
		{"StatusConflict", 409},
		{"StatusRateLimited", 429},
		{"StatusInternalError", 500},
		{"StatusBadGateway", 502},
		{"StatusUnavailable", 503},
		{"StatusTimeoutError", 504},
	}

	for _, tt := range tests {
		var got int
		switch tt.constant {
		case "StatusOK":
			got = StatusOK
		case "StatusInvalidParams":
			got = StatusInvalidParams
		case "StatusUnauthorized":
			got = StatusUnauthorized
		case "StatusForbidden":
			got = StatusForbidden
		case "StatusNotFound":
			got = StatusNotFound
		case "StatusMethodNotAllowed":
			got = StatusMethodNotAllowed
		case "StatusTimeout":
			got = StatusTimeout
		case "StatusConflict":
			got = StatusConflict
		case "StatusRateLimited":
			got = StatusRateLimited
		case "StatusInternalError":
			got = StatusInternalError
		case "StatusBadGateway":
			got = StatusBadGateway
		case "StatusUnavailable":
			got = StatusUnavailable
		case "StatusTimeoutError":
			got = StatusTimeoutError
		}

		if got != tt.expected {
			t.Errorf("Expected %s = %d, got %d", tt.constant, tt.expected, got)
		}
	}
}
