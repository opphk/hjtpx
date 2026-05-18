package captcha

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:8080")
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.baseURL != "http://localhost:8080" {
		t.Errorf("Expected baseURL to be 'http://localhost:8080', got '%s'", client.baseURL)
	}
	if client.httpClient == nil {
		t.Error("Expected non-nil httpClient")
	}
	if client.retryConfig == nil {
		t.Error("Expected non-nil retryConfig")
	}
	if client.rateLimiter == nil {
		t.Error("Expected non-nil rateLimiter")
	}
}

func TestClientOptions(t *testing.T) {
	customClient := &http.Client{Timeout: 60 * time.Second}
	apiKey := "test-api-key"
	retryConfig := &RetryConfig{MaxRetries: 5, BaseDelay: 200 * time.Millisecond}

	client := NewClient(
		"http://localhost:8080",
		WithHTTPClient(customClient),
		WithAPIKey(apiKey),
		WithRetryConfig(retryConfig),
	)

	if client.httpClient != customClient {
		t.Error("Expected custom HTTP client to be set")
	}
	if client.apiKey != apiKey {
		t.Errorf("Expected API key to be '%s', got '%s'", apiKey, client.apiKey)
	}
	if client.retryConfig.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries to be 5, got %d", client.retryConfig.MaxRetries)
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, time.Second)

	for i := 0; i < 10; i++ {
		if !rl.Allow() {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
	}

	if rl.Allow() {
		t.Error("Expected 11th request to be denied")
	}
}

func TestRateLimiterRefill(t *testing.T) {
	rl := NewRateLimiter(5, 100*time.Millisecond)

	rl.Allow()
	rl.Allow()

	time.Sleep(150 * time.Millisecond)

	rl.mu.Lock()
	rl.refill()
	rl.mu.Unlock()

	if rl.tokens < 5 {
		t.Errorf("Expected tokens to be refilled to at least 5, got %d", rl.tokens)
	}
}

func TestSliderCaptchaHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/captcha/slider" {
			t.Errorf("Expected path '/api/v1/captcha/slider', got '%s'", r.URL.Path)
		}
		if r.URL.Query().Get("width") != "320" {
			t.Errorf("Expected width '320', got '%s'", r.URL.Query().Get("width"))
		}

		w.Write([]byte(`{"code":0,"message":"success","data":{"session_id":"test-session","image_url":"http://example.com/image.png","puzzle_url":"http://example.com/puzzle.png","secret_y":50,"image_width":320,"image_height":160}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	captcha, err := client.GetSliderCaptcha(320, 160, 8)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if captcha.SessionID != "test-session" {
		t.Errorf("Expected session ID 'test-session', got '%s'", captcha.SessionID)
	}
	if captcha.SecretY != 50 {
		t.Errorf("Expected secret Y 50, got %d", captcha.SecretY)
	}
}

func TestVerifyCaptchaHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/captcha/verify" {
			t.Errorf("Expected path '/api/v1/captcha/verify', got '%s'", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected method 'POST', got '%s'", r.Method)
		}

		w.Write([]byte(`{"code":0,"message":"success","data":{"success":true,"message":"Verification passed","remaining_attempts":3}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.VerifyCaptcha(&VerifyCaptchaRequest{
		SessionID: "test-session",
		X:         100,
		Y:         50,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}
	if result.Message != "Verification passed" {
		t.Errorf("Expected message 'Verification passed', got '%s'", result.Message)
	}
	if result.Remaining != 3 {
		t.Errorf("Expected remaining attempts 3, got %d", result.Remaining)
	}
}

func TestVerifyCaptchaWithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":0,"message":"success","data":{"success":true,"message":"OK"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.VerifyCaptchaWithContext(ctx, &VerifyCaptchaRequest{
		SessionID: "test-session",
		X:         100,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}
}

func TestBatchVerify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":0,"message":"success","data":{"success":true,"message":"OK","remaining_attempts":3}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	requests := []VerifyCaptchaRequest{
		{SessionID: "session-1", X: 100},
		{SessionID: "session-2", X: 150},
		{SessionID: "session-3", X: 200},
	}

	result, err := client.BatchVerify(requests)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(result.Results))
	}
	if result.Success+result.Failed != 3 {
		t.Errorf("Expected total 3, got success=%d + failed=%d", result.Success, result.Failed)
	}
}

func TestBatchVerifyEmpty(t *testing.T) {
	client := NewClient("http://localhost:8080")
	result, err := client.BatchVerify([]VerifyCaptchaRequest{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(result.Results))
	}
}

func TestAsyncVerify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/captcha/async/verify" {
			t.Errorf("Expected path '/api/v1/captcha/async/verify', got '%s'", r.URL.Path)
		}

		w.Write([]byte(`{"code":0,"message":"success","data":{"task_id":"task-123","status":"pending","created_at":1700000000}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.AsyncVerify(&AsyncVerifyRequest{
		SessionID:  "session-1",
		X:          100,
		CallbackURL: "https://example.com/callback",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.TaskID != "task-123" {
		t.Errorf("Expected task ID 'task-123', got '%s'", result.TaskID)
	}
	if result.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", result.Status)
	}
}

func TestGetAsyncResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/captcha/async/result/task-123"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		w.Write([]byte(`{"code":0,"message":"success","data":{"task_id":"task-123","status":"completed","result":{"success":true,"message":"OK"},"completed_at":1700000001}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.GetAsyncResult("task-123")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", result.Status)
	}
	if result.Result == nil {
		t.Fatal("Expected non-nil result")
	}
	if !result.Result.Success {
		t.Error("Expected result success to be true")
	}
}

func TestWaitAsyncResult(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.Write([]byte(`{"code":0,"message":"success","data":{"task_id":"task-123","status":"processing"}}`))
		} else {
			w.Write([]byte(`{"code":0,"message":"success","data":{"task_id":"task-123","status":"completed","result":{"success":true,"message":"OK"}}}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	result, err := client.WaitAsyncResultWithContext(ctx, "task-123", 5*time.Second)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", result.Status)
	}
}

func TestUserAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" {
			t.Errorf("Expected path '/api/v1/auth/login', got '%s'", r.URL.Path)
		}

		w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"token123","refresh_token":"refresh456","expires_in":3600}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	auth := client.Auth()

	result, err := auth.Login(&LoginRequest{
		Username: "testuser",
		Password: "testpass",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.AccessToken != "token123" {
		t.Errorf("Expected access token 'token123', got '%s'", result.AccessToken)
	}
}

func TestEnvironment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/detect/script" {
			t.Errorf("Expected path '/api/v1/detect/script', got '%s'", r.URL.Path)
		}

		w.Write([]byte(`// detection script content`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	env := client.Env()

	script, err := env.GetDetectionScript("")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := `// detection script content`
	if script != expected {
		t.Errorf("Expected script '%s', got '%s'", expected, script)
	}
}

func TestWithRateLimiter(t *testing.T) {
	client := NewClient(
		"http://localhost:8080",
		WithRateLimiter(50, time.Second),
	)

	if client.rateLimiter.maxTokens != 50 {
		t.Errorf("Expected max tokens 50, got %d", client.rateLimiter.maxTokens)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
