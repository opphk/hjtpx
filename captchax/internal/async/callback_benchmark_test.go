package async

import (
	"context"
	"sync"
	"testing"
	"time"
)

func BenchmarkCallbackSubmit(b *testing.B) {
	manager := NewCallbackManager(10)
	defer manager.Stop()

	ctx := context.Background()
	result := VerificationResult{
		CaptchaID:  "test",
		AppID:     "app",
		Success:   true,
		VerifiedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.SubmitCallback(ctx, CallbackTypeHTTP, "http://example.com/callback", result)
	}
}

func BenchmarkCallbackManagerParallel(b *testing.B) {
	manager := NewCallbackManager(20)
	defer manager.Stop()

	ctx := context.Background()
	result := VerificationResult{
		CaptchaID:  "test",
		AppID:     "app",
		Success:   true,
		VerifiedAt: time.Now(),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			manager.SubmitCallback(ctx, CallbackTypeHTTP, "http://example.com/callback", result)
		}
	})
}

func TestCallbackManagerSubmit(t *testing.T) {
	manager := NewCallbackManager(5)
	defer manager.Stop()

	ctx := context.Background()
	result := VerificationResult{
		CaptchaID:  "test-captcha",
		AppID:     "test-app",
		Success:   true,
		VerifiedAt: time.Now(),
	}

	taskID, err := manager.SubmitCallback(ctx, CallbackTypeHTTP, "http://example.com/callback", result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if taskID == "" {
		t.Error("expected non-empty task ID")
	}
}

func TestCallbackManagerGetStatus(t *testing.T) {
	manager := NewCallbackManager(5)
	defer manager.Stop()

	ctx := context.Background()
	result := VerificationResult{
		CaptchaID:  "test",
		Success:   true,
		VerifiedAt: time.Now(),
	}

	taskID, _ := manager.SubmitCallback(ctx, CallbackTypeHTTP, "http://example.com/callback", result)

	task, exists := manager.GetTaskStatus(taskID)
	if !exists {
		t.Error("expected task to exist")
	}
	if task == nil {
		t.Error("expected task to not be nil")
	}
}

func TestCallbackManagerCancel(t *testing.T) {
	manager := NewCallbackManager(5)
	defer manager.Stop()

	ctx := context.Background()
	result := VerificationResult{
		CaptchaID:  "test",
		Success:   true,
		VerifiedAt: time.Now(),
	}

	taskID, _ := manager.SubmitCallback(ctx, CallbackTypeHTTP, "http://example.com/callback", result)

	canceled := manager.CancelTask(taskID)
	if !canceled {
		t.Error("expected task to be canceled")
	}
}

func TestCallbackManagerStats(t *testing.T) {
	manager := NewCallbackManager(5)
	defer manager.Stop()

	ctx := context.Background()
	result := VerificationResult{
		CaptchaID:  "test",
		Success:   true,
		VerifiedAt: time.Now(),
	}

	for i := 0; i < 10; i++ {
		manager.SubmitCallback(ctx, CallbackTypeHTTP, "http://example.com/callback", result)
	}

	stats := manager.GetStats()
	if stats.TotalSubmitted != 10 {
		t.Errorf("expected 10 submitted, got %d", stats.TotalSubmitted)
	}
}

func TestCallbackManagerCleanup(t *testing.T) {
	manager := NewCallbackManager(5)
	defer manager.Stop()

	ctx := context.Background()
	result := VerificationResult{
		CaptchaID:  "test",
		Success:   true,
		VerifiedAt: time.Now(),
	}

	for i := 0; i < 5; i++ {
		manager.SubmitCallback(ctx, CallbackTypeHTTP, "http://example.com/callback", result)
	}

	count := manager.Cleanup(1 * time.Hour)
	if count != 0 {
		t.Errorf("expected 0 cleaned up (too recent), got %d", count)
	}
}

func TestCallbackOptions(t *testing.T) {
	manager := NewCallbackManager(5)
	defer manager.Stop()

	ctx := context.Background()
	result := VerificationResult{
		CaptchaID:  "test",
		Success:   true,
		VerifiedAt: time.Now(),
	}

	taskID, err := manager.SubmitCallback(
		ctx,
		CallbackTypeHTTP,
		"http://example.com/callback",
		result,
		WithMethod("POST"),
		WithTimeout(5*time.Second),
		WithRetryCount(5),
		WithHeaders(map[string]string{"X-Custom": "value"}),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if taskID == "" {
		t.Error("expected non-empty task ID")
	}
}

func TestCallbackManagerConcurrency(t *testing.T) {
	manager := NewCallbackManager(10)
	defer manager.Stop()

	ctx := context.Background()
	result := VerificationResult{
		CaptchaID:  "test",
		Success:   true,
		VerifiedAt: time.Now(),
	}

	var wg sync.WaitGroup
	concurrency := 100

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				manager.SubmitCallback(ctx, CallbackTypeHTTP, "http://example.com/callback", result)
			}
		}()
	}

	wg.Wait()

	stats := manager.GetStats()
	if stats.TotalSubmitted != int64(concurrency*10) {
		t.Errorf("expected %d submitted, got %d", concurrency*10, stats.TotalSubmitted)
	}
}

func TestBatchCallbackManager(t *testing.T) {
	manager := NewBatchCallbackManager(5, 10, 100*time.Millisecond)
	defer manager.Stop()

	result := VerificationResult{
		CaptchaID:  "test",
		Success:   true,
		VerifiedAt: time.Now(),
	}

	for i := 0; i < 5; i++ {
		manager.Submit(CallbackTypeHTTP, "http://example.com/callback", result)
	}

	time.Sleep(200 * time.Millisecond)

	stats := manager.GetStats()
	if stats.TotalSubmitted < 5 {
		t.Errorf("expected at least 5 submitted, got %d", stats.TotalSubmitted)
	}
}

func TestBatchCallbackManagerBatchFlush(t *testing.T) {
	manager := NewBatchCallbackManager(5, 5, 1*time.Second)
	defer manager.Stop()

	result := VerificationResult{
		CaptchaID:  "test",
		Success:   true,
		VerifiedAt: time.Now(),
	}

	for i := 0; i < 5; i++ {
		manager.Submit(CallbackTypeHTTP, "http://example.com/callback", result)
	}

	time.Sleep(100 * time.Millisecond)

	stats := manager.GetStats()
	if stats.TotalSubmitted < 5 {
		t.Errorf("expected at least 5 submitted after batch flush, got %d", stats.TotalSubmitted)
	}
}

func TestHTTPCallbackHandler(t *testing.T) {
	handler := NewHTTPCallbackHandler(5*time.Second, 3)

	if handler.Type() != CallbackTypeHTTP {
		t.Errorf("expected type HTTP, got %s", handler.Type())
	}
}

func BenchmarkAsyncPool(b *testing.B) {
	pool := newAsyncPool(10, 1000)
	pool.Start()
	defer pool.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			task := &CallbackTask{
				ID:        "test",
				CreatedAt: time.Now(),
			}
			pool.Submit(task)
		}
	})
}

func BenchmarkCallbackStats(b *testing.B) {
	manager := NewCallbackManager(10)
	defer manager.Stop()

	ctx := context.Background()
	result := VerificationResult{
		CaptchaID:  "test",
		Success:   true,
		VerifiedAt: time.Now(),
	}

	for i := 0; i < 100; i++ {
		manager.SubmitCallback(ctx, CallbackTypeHTTP, "http://example.com/callback", result)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetStats()
	}
}

func TestCallbackManagerStop(t *testing.T) {
	manager := NewCallbackManager(5)

	manager.Stop()

	stats := manager.GetStats()
	if stats.TotalSubmitted != 0 {
		t.Errorf("expected 0 submitted after stop, got %d", stats.TotalSubmitted)
	}
}

func TestCallbackTaskFields(t *testing.T) {
	task := &CallbackTask{
		ID:        "test-id",
		Status:    CallbackStatusPending,
		Attempts:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Error:    nil,
	}

	if task.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", task.ID)
	}
	if task.Status != CallbackStatusPending {
		t.Errorf("expected status Pending, got %s", task.Status)
	}
}

func TestVerificationResultFields(t *testing.T) {
	result := VerificationResult{
		CaptchaID:  "captcha-123",
		AppID:     "app-456",
		Success:   true,
		Score:     95.5,
		Message:   "verified",
		VerifiedAt: time.Now(),
		Metadata:  map[string]interface{}{"key": "value"},
	}

	if result.CaptchaID != "captcha-123" {
		t.Errorf("unexpected CaptchaID: %s", result.CaptchaID)
	}
	if result.Score != 95.5 {
		t.Errorf("unexpected Score: %f", result.Score)
	}
}

func TestCallbackStatus(t *testing.T) {
	tests := []struct {
		status CallbackStatus
		want  string
	}{
		{CallbackStatusPending, "pending"},
		{CallbackStatusSuccess, "success"},
		{CallbackStatusFailed, "failed"},
		{CallbackStatusRetrying, "retrying"},
		{CallbackStatusCanceled, "canceled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("expected %s, got %s", tt.want, string(tt.status))
		}
	}
}

func TestCallbackType(t *testing.T) {
	tests := []struct {
		ctype CallbackType
		want  string
	}{
		{CallbackTypeHTTP, "http"},
		{CallbackTypeWebhook, "webhook"},
	}

	for _, tt := range tests {
		if string(tt.ctype) != tt.want {
			t.Errorf("expected %s, got %s", tt.want, string(tt.ctype))
		}
	}
}
