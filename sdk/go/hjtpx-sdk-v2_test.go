package sdk

import (
	"context"
	"testing"
	"time"
)

func TestHjtpxSDK_New(t *testing.T) {
	config := &Config{
		APIKey:        "test-api-key",
		APISecret:    "test-api-secret",
		BaseURL:      "https://api.test.com/v2",
		Timeout:      30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:   time.Second,
	}

	sdk := New(config)

	if sdk.APIKey != config.APIKey {
		t.Errorf("Expected API key %s, got %s", config.APIKey, sdk.APIKey)
	}

	if sdk.BaseURL != config.BaseURL {
		t.Errorf("Expected base URL %s, got %s", config.BaseURL, sdk.BaseURL)
	}

	if sdk.RetryAttempts != config.RetryAttempts {
		t.Errorf("Expected retry attempts %d, got %d", config.RetryAttempts, sdk.RetryAttempts)
	}
}

func TestHjtpxSDK_UsePlugin(t *testing.T) {
	sdk := New(&Config{
		APIKey:     "test-key",
		APISecret: "test-secret",
	})

	plugin := &RetryPlugin{maxRetries: 3, delay: time.Second}
	sdk.UsePlugin(plugin)

	if len(sdk.Plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(sdk.Plugins))
	}
}

func TestHjtpxSDK_UseMiddleware(t *testing.T) {
	sdk := New(&Config{
		APIKey:     "test-key",
		APISecret: "test-secret",
	})

	middleware := func(req *http.Request) error {
		return nil
	}
	sdk.UseMiddleware(middleware)

	if len(sdk.Middleware) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(sdk.Middleware))
	}
}

func TestHjtpxSDK_CreateCaptcha(t *testing.T) {
	sdk := New(&Config{
		APIKey:     "test-key",
		APISecret: "test-secret",
	})

	req := &CaptchaRequest{
		AppID:       "app-123",
		CaptchaType: "image",
		Action:      "create",
		UserID:      "user-456",
	}

	resp, err := sdk.CreateCaptcha(context.Background(), req)
	if err != nil {
		t.Errorf("CreateCaptcha() error = %v", err)
	}

	if resp == nil {
		t.Error("CreateCaptcha() returned nil response")
	}
}

func TestHjtpxSDK_Verify(t *testing.T) {
	sdk := New(&Config{
		APIKey:     "test-key",
		APISecret: "test-secret",
	})

	req := &VerificationRequest{
		CaptchaID: "captcha-123",
		Token:     "test-token",
		UserID:    "user-456",
	}

	resp, err := sdk.Verify(context.Background(), req)
	if err != nil {
		t.Errorf("Verify() error = %v", err)
	}

	if resp == nil {
		t.Error("Verify() returned nil response")
	}
}

func TestHjtpxSDK_GetAnalytics(t *testing.T) {
	sdk := New(&Config{
		APIKey:     "test-key",
		APISecret: "test-secret",
	})

	req := &AnalyticsRequest{
		AppID:     "app-123",
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now(),
		Metrics:   []string{"requests", "verifications"},
	}

	resp, err := sdk.GetAnalytics(context.Background(), req)
	if err != nil {
		t.Errorf("GetAnalytics() error = %v", err)
	}

	if resp == nil {
		t.Error("GetAnalytics() returned nil response")
	}
}

func TestHjtpxSDK_GetAppConfig(t *testing.T) {
	sdk := New(&Config{
		APIKey:     "test-key",
		APISecret: "test-secret",
	})

	config, err := sdk.GetAppConfig(context.Background(), "app-123")
	if err != nil {
		t.Errorf("GetAppConfig() error = %v", err)
	}

	if config == nil {
		t.Error("GetAppConfig() returned nil config")
	}
}

func TestHjtpxSDK_UpdateAppConfig(t *testing.T) {
	sdk := New(&Config{
		APIKey:     "test-key",
		APISecret: "test-secret",
	})

	config := &AppConfig{
		AppID:         "app-123",
		Name:          "Test App",
		SecurityLevel: "high",
	}

	updated, err := sdk.UpdateAppConfig(context.Background(), "app-123", config)
	if err != nil {
		t.Errorf("UpdateAppConfig() error = %v", err)
	}

	if updated == nil {
		t.Error("UpdateAppConfig() returned nil config")
	}
}

func TestHjtpxSDK_RegisterWebhook(t *testing.T) {
	sdk := New(&Config{
		APIKey:     "test-key",
		APISecret: "test-secret",
	})

	err := sdk.RegisterWebhook(context.Background(), "app-123", "verification.completed", "https://example.com/webhook")
	if err != nil {
		t.Errorf("RegisterWebhook() error = %v", err)
	}
}

func TestHjtpxSDK_ListWebhooks(t *testing.T) {
	sdk := New(&Config{
		APIKey:     "test-key",
		APISecret: "test-secret",
	})

	webhooks, err := sdk.ListWebhooks(context.Background(), "app-123")
	if err != nil {
		t.Errorf("ListWebhooks() error = %v", err)
	}

	if webhooks == nil {
		t.Error("ListWebhooks() returned nil")
	}
}

func TestHjtpxSDK_Builder(t *testing.T) {
	sdk := New(&Config{
		APIKey:     "test-key",
		APISecret: "test-secret",
	})

	resp, err := sdk.NewBuilder().
		AppID("app-123").
		Type("image").
		UserID("user-456").
		SessionID("session-789").
		IPAddress("192.168.1.1").
		UserAgent("Mozilla/5.0").
		Parameter("difficulty", "medium").
		Metadata("key", "value").
		Build(context.Background())

	if err != nil {
		t.Errorf("Builder.Build() error = %v", err)
	}

	if resp == nil {
		t.Error("Builder.Build() returned nil response")
	}
}

func TestRetryPlugin(t *testing.T) {
	plugin := NewRetryPlugin(3, time.Second)

	if plugin.Name() != "retry" {
		t.Errorf("Expected name 'retry', got '%s'", plugin.Name())
	}

	if plugin.Version() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", plugin.Version())
	}

	_, err := plugin.Execute(context.Background(), &CaptchaRequest{})
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
}

func TestCachePlugin(t *testing.T) {
	plugin := NewCachePlugin()

	if plugin.Name() != "cache" {
		t.Errorf("Expected name 'cache', got '%s'", plugin.Name())
	}

	cached := plugin.Get("test-key")
	if cached != nil {
		t.Error("Cache should be empty initially")
	}

	response := &CaptchaResponse{
		CaptchaID: "test-123",
		Status:   "success",
		Type:     "image",
		Data:     make(map[string]interface{}),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	plugin.Store("test-key", response)

	cached = plugin.Get("test-key")
	if cached == nil {
		t.Error("Cache should contain the response")
	}
}

func TestRateLimitPlugin(t *testing.T) {
	plugin := NewRateLimitPlugin(5, time.Minute)

	if plugin.Name() != "rate_limiter" {
		t.Errorf("Expected name 'rate_limiter', got '%s'", plugin.Name())
	}

	for i := 0; i < 5; i++ {
		_, err := plugin.Execute(context.Background(), &CaptchaRequest{})
		if err != nil {
			t.Errorf("Execute() should not return error for requests within limit, got %v", err)
		}
	}

	_, err := plugin.Execute(context.Background(), &CaptchaRequest{})
	if err == nil {
		t.Error("Execute() should return error when rate limit exceeded")
	}
}

func TestMetricsPlugin(t *testing.T) {
	plugin := NewMetricsPlugin()

	if plugin.Name() != "metrics" {
		t.Errorf("Expected name 'metrics', got '%s'", plugin.Name())
	}

	plugin.RecordSuccess(100)
	plugin.RecordSuccess(200)
	plugin.RecordFailure()

	metrics := plugin.GetMetrics()
	if metrics["total_requests"].(int64) != 3 {
		t.Errorf("Expected 3 total requests, got %v", metrics["total_requests"])
	}

	if metrics["success_count"].(int64) != 2 {
		t.Errorf("Expected 2 success count, got %v", metrics["success_count"])
	}

	if metrics["failure_count"].(int64) != 1 {
		t.Errorf("Expected 1 failure count, got %v", metrics["failure_count"])
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Minute)

	if cb.GetState() != "closed" {
		t.Errorf("Expected state 'closed', got '%s'", cb.GetState())
	}

	success := 0
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			success++
			return nil
		})
		if err != nil {
			t.Errorf("Execute() should succeed, got %v", err)
		}
	}

	if success != 3 {
		t.Errorf("Expected 3 successful executions, got %d", success)
	}
}

func TestCircuitBreaker_Open(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Second)

	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return nil
		})
	}

	cb.Execute(func() error {
		return nil
	})

	if cb.GetState() != "closed" {
		t.Errorf("Expected state 'closed', got '%s'", cb.GetState())
	}
}

func TestSDKError(t *testing.T) {
	err := &SDKError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
		Details: "Additional details",
	}

	expected := "[TEST_ERROR] Test error message: Additional details"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestCaptchaRequest(t *testing.T) {
	req := &CaptchaRequest{
		AppID:       "app-123",
		CaptchaType: "image",
		Action:      "create",
		UserID:      "user-456",
		SessionID:   "session-789",
		IPAddress:   "192.168.1.1",
		UserAgent:   "Mozilla/5.0",
		Parameters: map[string]interface{}{
			"difficulty": "medium",
		},
		Metadata: map[string]string{
			"source": "web",
		},
	}

	if req.AppID != "app-123" {
		t.Errorf("Expected AppID 'app-123', got '%s'", req.AppID)
	}

	if req.CaptchaType != "image" {
		t.Errorf("Expected CaptchaType 'image', got '%s'", req.CaptchaType)
	}
}

func TestCaptchaResponse(t *testing.T) {
	resp := &CaptchaResponse{
		CaptchaID: "captcha-123",
		Status:   "success",
		Type:     "image",
		Data: map[string]interface{}{
			"image_url": "https://example.com/captcha.png",
		},
		CreatedAt: time.Now(),
	}

	if resp.CaptchaID != "captcha-123" {
		t.Errorf("Expected CaptchaID 'captcha-123', got '%s'", resp.CaptchaID)
	}

	if resp.Status != "success" {
		t.Errorf("Expected Status 'success', got '%s'", resp.Status)
	}
}

func TestVerificationRequest(t *testing.T) {
	req := &VerificationRequest{
		CaptchaID:  "captcha-123",
		Token:     "user-token",
		Solution:  "1234",
		UserID:    "user-456",
		SessionID: "session-789",
	}

	if req.CaptchaID != "captcha-123" {
		t.Errorf("Expected CaptchaID 'captcha-123', got '%s'", req.CaptchaID)
	}

	if req.Token != "user-token" {
		t.Errorf("Expected Token 'user-token', got '%s'", req.Token)
	}
}

func TestVerificationResponse(t *testing.T) {
	resp := &VerificationResponse{
		Valid:         true,
		Score:         0.95,
		RiskLevel:    "low",
		Reasons:       []string{},
		SessionID:     "session-789",
		RemainingTries: 3,
	}

	if !resp.Valid {
		t.Error("Expected Valid to be true")
	}

	if resp.Score != 0.95 {
		t.Errorf("Expected Score 0.95, got %f", resp.Score)
	}
}

func TestWebhookInfo(t *testing.T) {
	webhook := &WebhookInfo{
		ID:        "webhook-123",
		AppID:     "app-123",
		EventType: "verification.completed",
		URL:       "https://example.com/webhook",
		Status:    "active",
		CreatedAt: time.Now(),
	}

	if webhook.ID != "webhook-123" {
		t.Errorf("Expected ID 'webhook-123', got '%s'", webhook.ID)
	}

	if webhook.Status != "active" {
		t.Errorf("Expected Status 'active', got '%s'", webhook.Status)
	}
}
