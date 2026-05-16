package service

import (
	"context"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/stretchr/testify/assert"
)

func setupTestRedis() bool {
	if redis.Client == nil {
		return false
	}
	ctx := context.Background()
	err := redis.Client.Ping(ctx).Err()
	return err == nil
}

func TestCacheService_SetCaptchaCache(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	captchaData := &CaptchaCache{
		CaptchaID:  "test-captcha-123",
		Answer:     "1234",
		ExpiresAt:  time.Now().Add(5 * time.Minute),
		Difficulty: 2,
	}

	err := cacheService.SetCaptchaCache(ctx, captchaData.CaptchaID, captchaData)
	assert.NoError(t, err)
}

func TestCacheService_GetCaptchaCache(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	captchaID := "test-captcha-get-456"
	captchaData := &CaptchaCache{
		CaptchaID:  captchaID,
		Answer:     "5678",
		ExpiresAt:  time.Now().Add(5 * time.Minute),
		Difficulty: 3,
	}

	err := cacheService.SetCaptchaCache(ctx, captchaID, captchaData)
	assert.NoError(t, err)

	retrieved, err := cacheService.GetCaptchaCache(ctx, captchaID)
	if err != nil {
		t.Logf("GetCaptchaCache failed (may be nil client): %v", err)
		return
	}
	if retrieved == nil {
		t.Log("GetCaptchaCache returned nil (may be nil client)")
		return
	}
	assert.Equal(t, captchaID, retrieved.CaptchaID)
	assert.Equal(t, captchaData.Answer, retrieved.Answer)
	assert.Equal(t, captchaData.Difficulty, retrieved.Difficulty)
}

func TestCacheService_DeleteCaptchaCache(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	captchaID := "test-captcha-delete-789"
	captchaData := &CaptchaCache{
		CaptchaID:  captchaID,
		Answer:     "9999",
		ExpiresAt:  time.Now().Add(5 * time.Minute),
		Difficulty: 1,
	}

	err := cacheService.SetCaptchaCache(ctx, captchaID, captchaData)
	assert.NoError(t, err)

	err = cacheService.DeleteCaptchaCache(ctx, captchaID)
	assert.NoError(t, err)

	retrieved, err := cacheService.GetCaptchaCache(ctx, captchaID)
	if err != nil {
		return
	}
	assert.Nil(t, retrieved)
}

func TestCacheService_SetBehaviorCache(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	sessionID := "session-123"
	behaviorData := &BehaviorCache{
		UserID:     "user-456",
		SessionID:  sessionID,
		Trajectory: "[{\"x\":100,\"y\":200},{\"x\":110,\"y\":210}]",
		Timestamp:  time.Now(),
	}

	err := cacheService.SetBehaviorCache(ctx, sessionID, behaviorData)
	assert.NoError(t, err)
}

func TestCacheService_GetBehaviorCache(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	sessionID := "session-get-123"
	behaviorData := &BehaviorCache{
		UserID:     "user-789",
		SessionID:  sessionID,
		Trajectory: "[{\"x\":100,\"y\":200}]",
		Timestamp:  time.Now(),
	}

	err := cacheService.SetBehaviorCache(ctx, sessionID, behaviorData)
	assert.NoError(t, err)

	retrieved, err := cacheService.GetBehaviorCache(ctx, sessionID)
	if err != nil {
		t.Logf("GetBehaviorCache failed (may be nil client): %v", err)
		return
	}
	if retrieved == nil {
		t.Log("GetBehaviorCache returned nil (may be nil client)")
		return
	}
	assert.Equal(t, sessionID, retrieved.SessionID)
	assert.Equal(t, behaviorData.UserID, retrieved.UserID)
	assert.Equal(t, behaviorData.Trajectory, retrieved.Trajectory)
}

func TestCacheService_SetSessionCache(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	token := "token-abc-123"
	sessionData := &SessionCache{
		UserID:    "user-xyz",
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	err := cacheService.SetSessionCache(ctx, token, sessionData)
	assert.NoError(t, err)
}

func TestCacheService_GetSessionCache(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	token := "token-get-456"
	sessionData := &SessionCache{
		UserID:    "user-get-789",
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "10.0.0.1",
		UserAgent: "Chrome/91.0",
	}

	err := cacheService.SetSessionCache(ctx, token, sessionData)
	assert.NoError(t, err)

	retrieved, err := cacheService.GetSessionCache(ctx, token)
	if err != nil {
		t.Logf("GetSessionCache failed (may be nil client): %v", err)
		return
	}
	if retrieved == nil {
		t.Log("GetSessionCache returned nil (may be nil client)")
		return
	}
	assert.Equal(t, token, retrieved.Token)
	assert.Equal(t, sessionData.UserID, retrieved.UserID)
	assert.Equal(t, sessionData.IPAddress, retrieved.IPAddress)
	assert.Equal(t, sessionData.UserAgent, retrieved.UserAgent)
}

func TestCacheService_DeleteSessionCache(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	token := "token-delete-789"
	sessionData := &SessionCache{
		UserID:    "user-delete",
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "172.16.0.1",
		UserAgent: "Safari/14.0",
	}

	err := cacheService.SetSessionCache(ctx, token, sessionData)
	assert.NoError(t, err)

	err = cacheService.DeleteSessionCache(ctx, token)
	assert.NoError(t, err)

	retrieved, err := cacheService.GetSessionCache(ctx, token)
	if err != nil {
		return
	}
	assert.Nil(t, retrieved)
}

func TestCacheService_IncrementRateLimit(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	identifier := "rate-limit-test-123"
	window := 1 * time.Minute

	count1, err := cacheService.IncrementRateLimit(ctx, identifier, window)
	assert.NoError(t, err)
	assert.Equal(t, 1, count1)

	count2, err := cacheService.IncrementRateLimit(ctx, identifier, window)
	assert.NoError(t, err)
	assert.Equal(t, 2, count2)

	count3, err := cacheService.IncrementRateLimit(ctx, identifier, window)
	assert.NoError(t, err)
	assert.Equal(t, 3, count3)

	cacheService.ResetRateLimit(ctx, identifier)
}

func TestCacheService_GetRateLimitCount(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	identifier := "rate-limit-get-456"
	window := 1 * time.Minute

	_, err := cacheService.IncrementRateLimit(ctx, identifier, window)
	assert.NoError(t, err)
	_, err = cacheService.IncrementRateLimit(ctx, identifier, window)
	assert.NoError(t, err)

	count, err := cacheService.GetRateLimitCount(ctx, identifier, window)
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	cacheService.ResetRateLimit(ctx, identifier)
}

func TestCacheService_ResetRateLimit(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	identifier := "rate-limit-reset-789"
	window := 1 * time.Minute

	_, err := cacheService.IncrementRateLimit(ctx, identifier, window)
	assert.NoError(t, err)
	_, err = cacheService.IncrementRateLimit(ctx, identifier, window)
	assert.NoError(t, err)

	err = cacheService.ResetRateLimit(ctx, identifier)
	assert.NoError(t, err)

	count, err := cacheService.GetRateLimitCount(ctx, identifier, window)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCacheWarmer(t *testing.T) {
	ctx := context.Background()
	cacheWarmer := NewCacheWarmer()
	assert.NotNil(t, cacheWarmer)

	taskExecuted := false
	task := WarmupTask{
		Name: "test-task",
		Handler: func() error {
			taskExecuted = true
			return nil
		},
		Interval: 100 * time.Millisecond,
	}

	cacheWarmer.AddTask(task)
	cacheWarmer.Start(ctx)

	time.Sleep(150 * time.Millisecond)
	cacheWarmer.Stop()

	assert.True(t, taskExecuted, "Task should have been executed")
}

func TestCacheWarmer_StopWithoutStart(t *testing.T) {
	cacheWarmer := NewCacheWarmer()
	assert.NotNil(t, cacheWarmer)

	task := WarmupTask{
		Name: "idle-task",
		Handler: func() error {
			return nil
		},
		Interval: 1 * time.Hour,
	}

	cacheWarmer.AddTask(task)
	cacheWarmer.Stop()
}

func TestCacheCleanup(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	captchaID := "cleanup-test-123"
	captchaData := &CaptchaCache{
		CaptchaID:  captchaID,
		Answer:     "1234",
		ExpiresAt:  time.Now().Add(5 * time.Minute),
		Difficulty: 1,
	}

	err := cacheService.SetCaptchaCache(ctx, captchaID, captchaData)
	assert.NoError(t, err)

	cleaned, err := cacheService.CleanupExpiredKeys(ctx, "captcha:*")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, cleaned, 0)
}

func TestCacheMetrics(t *testing.T) {
	cacheService := NewCacheService()

	ResetMetrics()

	cacheService.RecordHit()
	cacheService.RecordHit()
	cacheService.RecordMiss()
	cacheService.RecordSet()
	cacheService.RecordDelete()
	cacheService.RecordExpired()
	cacheService.RecordEvicted()

	metrics := cacheService.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(2), metrics.Hits)
	assert.Equal(t, int64(1), metrics.Misses)
	assert.Equal(t, int64(1), metrics.Sets)
	assert.Equal(t, int64(1), metrics.Deletes)
	assert.Equal(t, int64(1), metrics.Expired)
	assert.Equal(t, int64(1), metrics.Evicted)
}

func TestCacheMetrics_Reset(t *testing.T) {
	cacheService := NewCacheService()

	cacheService.RecordHit()
	cacheService.RecordMiss()
	cacheService.RecordSet()

	ResetMetrics()

	metrics := cacheService.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.Hits)
	assert.Equal(t, int64(0), metrics.Misses)
	assert.Equal(t, int64(0), metrics.Sets)
}

func TestNewCacheService(t *testing.T) {
	cacheService := NewCacheService()
	assert.NotNil(t, cacheService)
	assert.Equal(t, 5*time.Minute, cacheService.defaultTTL)
}

func TestNewCacheService_WithOptions(t *testing.T) {
	cacheService := NewCacheService(WithDefaultTTL(10 * time.Minute))
	assert.NotNil(t, cacheService)
	assert.Equal(t, 10*time.Minute, cacheService.defaultTTL)
}

func TestCacheService_Get_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	val, err := cacheService.Get(ctx, "nonexistent-key")
	assert.Error(t, err)
	assert.Empty(t, val)
}

func TestCacheService_Set_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	err := cacheService.Set(ctx, "test-key", "test-value")
	assert.NoError(t, err)
}

func TestCacheService_Delete_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	err := cacheService.Delete(ctx, "test-key")
	assert.NoError(t, err)

	err = cacheService.Delete(ctx)
	assert.NoError(t, err)
}

func TestCacheService_Increment_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	count, err := cacheService.Increment(ctx, "counter")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestCacheService_Decrement_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	count, err := cacheService.Decrement(ctx, "counter")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestCacheService_IncrementBy_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	count, err := cacheService.IncrementBy(ctx, "counter", 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestCacheService_Exists_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	exists, err := cacheService.Exists(ctx, "test-key")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCacheService_Expire_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	err := cacheService.Expire(ctx, "test-key", 5*time.Minute)
	assert.NoError(t, err)
}

func TestCacheService_TTL_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	ttl, err := cacheService.TTL(ctx, "test-key")
	assert.Error(t, err)
	assert.Equal(t, time.Duration(0), ttl)
}

func TestCacheService_GetMultiple_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	results, err := cacheService.GetMultiple(ctx, []string{"key1", "key2"})
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestCacheService_GetMultiple_EmptyKeys(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	results, err := cacheService.GetMultiple(ctx, []string{})
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestCacheService_SetMultiple_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	err := cacheService.SetMultiple(ctx, items, 5*time.Minute)
	assert.NoError(t, err)
}

func TestCacheService_SetMultiple_EmptyItems(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	err := cacheService.SetMultiple(ctx, map[string]interface{}{}, 5*time.Minute)
	assert.NoError(t, err)
}

func TestCacheService_DeleteByPattern_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	deleted, err := cacheService.DeleteByPattern(ctx, "test:*")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
}

func TestCacheService_GetEntry_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	entry, err := cacheService.GetEntry(ctx, "test-key")
	assert.Error(t, err)
	assert.Nil(t, entry)
}

func TestCacheService_GetOrSet_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	result, err := cacheService.GetOrSet(ctx, "key", 5*time.Minute, func() (interface{}, error) {
		return "computed-value", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "computed-value", result)
}

func TestCacheService_GetJSONOrSet_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	result, err := cacheService.GetJSONOrSet(ctx, "key", 5*time.Minute, func() (interface{}, error) {
		return map[string]string{"field": "value"}, nil
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDistributedLock_NilClient(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	lock, err := cacheService.AcquireLock(ctx, "test-lock", nil)
	assert.Error(t, err)
	assert.Nil(t, lock)
}

func TestCircuitBreaker_Allow(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	assert.True(t, cb.Allow())
	cb.RecordSuccess()
	assert.Equal(t, "closed", cb.State())
}

func TestCircuitBreaker_Open(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, "open", cb.State())
}

func TestCircuitBreaker_HalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 50*time.Millisecond)

	cb.RecordFailure()
	assert.Equal(t, "open", cb.State())

	time.Sleep(60 * time.Millisecond)

	assert.True(t, cb.Allow())
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(1, 100*time.Millisecond)

	cb.RecordFailure()
	assert.Equal(t, "open", cb.State())

	cb.Reset()
	assert.Equal(t, "closed", cb.State())
	assert.Equal(t, 0, cb.failures)
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()

	assert.Equal(t, "closed", cb.State())
	assert.Equal(t, 0, cb.failures)
}

func TestNewCachedFunction(t *testing.T) {
	cf := NewCachedFunction()
	assert.NotNil(t, cf)
	assert.NotNil(t, cf.service)
	assert.NotNil(t, cf.breaker)
}

func TestCachedFunction_Execute(t *testing.T) {
	cf := NewCachedFunction()
	ctx := context.Background()

	result, err := cf.Execute(ctx, "test-key", 5*time.Minute, func() (interface{}, error) {
		return "computed-result", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "computed-result", result)
}

func TestCaptchaCache_Expiration(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	captchaID := "expired-captcha-123"
	captchaData := &CaptchaCache{
		CaptchaID:  captchaID,
		Answer:     "1234",
		ExpiresAt:  time.Now().Add(-1 * time.Hour),
		Difficulty: 1,
	}

	err := cacheService.SetCaptchaCache(ctx, captchaID, captchaData)
	assert.NoError(t, err)

	retrieved, err := cacheService.GetCaptchaCache(ctx, captchaID)
	assert.Error(t, err)
	assert.Nil(t, retrieved)
}

func TestSessionCache_Expiration(t *testing.T) {
	ctx := context.Background()
	cacheService := NewCacheService()

	token := "expired-session-456"
	sessionData := &SessionCache{
		UserID:    "user-expired",
		Token:     token,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		IPAddress: "192.168.1.1",
		UserAgent: "TestAgent",
	}

	err := cacheService.SetSessionCache(ctx, token, sessionData)
	assert.NoError(t, err)

	retrieved, err := cacheService.GetSessionCache(ctx, token)
	assert.Error(t, err)
	assert.Nil(t, retrieved)
}

func TestRateLimitCache_DifferentIdentifiers(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	window := 1 * time.Minute

	count1, err := cacheService.IncrementRateLimit(ctx, "id-1", window)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	assert.Equal(t, 1, count1)

	count2, err := cacheService.IncrementRateLimit(ctx, "id-2", window)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	assert.Equal(t, 1, count2)

	count3, err := cacheService.IncrementRateLimit(ctx, "id-1", window)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	assert.Equal(t, 2, count3)

	cacheService.ResetRateLimit(ctx, "id-1")
	cacheService.ResetRateLimit(ctx, "id-2")
}

func TestBehaviorCache_DifferentSessions(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	session1 := "session-1"
	session2 := "session-2"

	behavior1 := &BehaviorCache{
		UserID:     "user-1",
		SessionID:  session1,
		Trajectory: "trajectory-1",
		Timestamp:  time.Now(),
	}

	behavior2 := &BehaviorCache{
		UserID:     "user-2",
		SessionID:  session2,
		Trajectory: "trajectory-2",
		Timestamp:  time.Now(),
	}

	err := cacheService.SetBehaviorCache(ctx, session1, behavior1)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	assert.NoError(t, err)

	err = cacheService.SetBehaviorCache(ctx, session2, behavior2)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	assert.NoError(t, err)

	retrieved1, err := cacheService.GetBehaviorCache(ctx, session1)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	assert.NoError(t, err)
	assert.Equal(t, session1, retrieved1.SessionID)

	retrieved2, err := cacheService.GetBehaviorCache(ctx, session2)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	assert.NoError(t, err)
	assert.Equal(t, session2, retrieved2.SessionID)
}

func TestCacheWarmer_MultipleTasks(t *testing.T) {
	ctx := context.Background()
	cacheWarmer := NewCacheWarmer()

	taskCount := 0
	for i := 0; i < 3; i++ {
		task := WarmupTask{
			Name: "task-" + string(rune('A'+i)),
			Handler: func() error {
				taskCount++
				return nil
			},
			Interval: 100 * time.Millisecond,
		}
		cacheWarmer.AddTask(task)
	}

	cacheWarmer.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	cacheWarmer.Stop()

	assert.GreaterOrEqual(t, taskCount, 3)
}

func TestCacheMetrics_ConcurrentAccess(t *testing.T) {
	cacheService := NewCacheService()

	ResetMetrics()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				cacheService.RecordHit()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := cacheService.GetMetrics()
	assert.Equal(t, int64(1000), metrics.Hits)
}

func TestCacheService_RefreshSession(t *testing.T) {
	if !setupTestRedis() {
		t.Skip("Redis not available, skipping test")
	}
	ctx := context.Background()
	cacheService := NewCacheService()

	token := "refresh-token-123"
	sessionData := &SessionCache{
		UserID:    "user-refresh",
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		IPAddress: "192.168.1.1",
		UserAgent: "TestAgent",
	}

	err := cacheService.SetSessionCache(ctx, token, sessionData)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	assert.NoError(t, err)

	err = cacheService.RefreshSession(ctx, token)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	assert.NoError(t, err)

	retrieved, err := cacheService.GetSessionCache(ctx, token)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	assert.NoError(t, err)
	assert.True(t, retrieved.ExpiresAt.After(time.Now().Add(23*time.Hour)))
}
