package text

import (
	"captchax/config"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// MockRedisClient 模拟 Redis 客户端
type MockRedisClient struct {
	data map[string]string
	ttls map[string]time.Duration
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]string),
		ttls: make(map[string]time.Duration),
	}
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	var strVal string
	switch v := value.(type) {
	case string:
		strVal = v
	case []byte:
		strVal = string(v)
	default:
		data, _ := json.Marshal(v)
		strVal = string(data)
	}
	m.data[key] = strVal
	m.ttls[key] = expiration
	return nil
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return "", ErrCaptchaNotFound
}

func (m *MockRedisClient) GetBytes(ctx context.Context, key string) ([]byte, error) {
	if val, ok := m.data[key]; ok {
		return []byte(val), nil
	}
	return nil, ErrCaptchaNotFound
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		delete(m.data, key)
		delete(m.ttls, key)
	}
	return nil
}

func (m *MockRedisClient) Exists(ctx context.Context, key string) (int64, error) {
	if _, ok := m.data[key]; ok {
		return 1, nil
	}
	return 0, nil
}

func (m *MockRedisClient) ExistsCtx(ctx context.Context, keys ...string) (int64, error) {
	count := int64(0)
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			count++
		}
	}
	return count, nil
}

func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	m.ttls[key] = expiration
	return nil
}

func (m *MockRedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	if ttl, ok := m.ttls[key]; ok {
		return ttl, nil
	}
	return 0, nil
}

func (m *MockRedisClient) Incr(ctx context.Context, key string) (int64, error) {
	var val int64
	if s, ok := m.data[key]; ok {
		_ = json.Unmarshal([]byte(s), &val)
	}
	val++
	data, _ := json.Marshal(val)
	m.data[key] = string(data)
	return val, nil
}

func (m *MockRedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	var result []string
	for key := range m.data {
		result = append(result, key)
	}
	return result, nil
}

func (m *MockRedisClient) Ping(ctx context.Context) error {
	return nil
}

func (m *MockRedisClient) Close() error {
	return nil
}

func (m *MockRedisClient) Client() interface{} {
	return nil
}

// MockCacheManager 模拟缓存管理器
type MockCacheManager struct {
	redis *MockRedisClient
	cfg   *config.CaptchaConfig
}

func NewMockCacheManager(cfg *config.CaptchaConfig) *MockCacheManager {
	return &MockCacheManager{
		redis: NewMockRedisClient(),
		cfg:   cfg,
	}
}

func (cm *MockCacheManager) keyForID(id string) string {
	return fmt.Sprintf("captcha:text:%s", id)
}

func (cm *MockCacheManager) Set(ctx context.Context, id string, data *CacheData) error {
	key := cm.keyForID(id)
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}
	expiration := time.Duration(cm.cfg.ExpireMinutes) * time.Minute
	if err := cm.redis.Set(ctx, key, dataBytes, expiration); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}
	return nil
}

func (cm *MockCacheManager) Get(ctx context.Context, id string) (*CacheData, error) {
	key := cm.keyForID(id)
	dataStr, err := cm.redis.Get(ctx, key)
	if err != nil {
		if err.Error() == ErrCaptchaNotFound.Error() {
			return nil, ErrCaptchaNotFound
		}
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}
	var data CacheData
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache data: %w", err)
	}
	if cm.isExpired(&data) {
		_ = cm.Delete(ctx, id)
		return nil, ErrCaptchaExpired
	}
	return &data, nil
}

func (cm *MockCacheManager) Delete(ctx context.Context, id string) error {
	key := cm.keyForID(id)
	if err := cm.redis.Del(ctx, key); err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	return nil
}

func (cm *MockCacheManager) MarkVerified(ctx context.Context, id string) error {
	data, err := cm.Get(ctx, id)
	if err != nil {
		return err
	}
	if data.Verified {
		return ErrCaptchaVerified
	}
	data.Verified = true
	return cm.Set(ctx, id, data)
}

func (cm *MockCacheManager) isExpired(data *CacheData) bool {
	if data.CreatedAt == 0 {
		return false
	}
	expiration := time.Duration(cm.cfg.ExpireMinutes) * time.Minute
	expirationTime := time.Unix(data.CreatedAt, 0).Add(expiration)
	return time.Now().After(expirationTime)
}

func (cm *MockCacheManager) Exists(ctx context.Context, id string) (bool, error) {
	key := cm.keyForID(id)
	count, err := cm.redis.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return count > 0, nil
}

// 测试生成验证码
func TestGenerateCaptcha(t *testing.T) {
	cfg := &config.CaptchaConfig{
		Width:         200,
		Height:        80,
		ExpireMinutes: 5,
	}

	t.Run("Generate captcha creates valid result", func(t *testing.T) {
		text := &Text{
			cfg:   cfg,
			redis: nil, // 不使用 Redis
		}

		result, err := text.GenerateCaptcha(context.Background())
		if err != nil {
			t.Fatalf("GenerateCaptcha() error = %v", err)
		}

		if result.ID == "" {
			t.Error("GenerateCaptcha() returned empty ID")
		}

		if result.ImageB64 == "" {
			t.Error("GenerateCaptcha() returned empty ImageB64")
		}
	})
}

// 测试缓存操作
func TestCacheSetGet(t *testing.T) {
	cfg := &config.CaptchaConfig{
		ExpireMinutes: 5,
	}

	cacheManager := NewMockCacheManager(cfg)
	ctx := context.Background()

	t.Run("Set and Get cache data", func(t *testing.T) {
		testData := &CacheData{
			ID:        "test-captcha-123",
			Code:      "TEST123",
			CreatedAt: time.Now().Unix(),
			Verified:  false,
		}

		err := cacheManager.Set(ctx, testData.ID, testData)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		retrieved, err := cacheManager.Get(ctx, testData.ID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if retrieved.ID != testData.ID {
			t.Errorf("ID = %s, want %s", retrieved.ID, testData.ID)
		}

		if retrieved.Code != testData.Code {
			t.Errorf("Code = %s, want %s", retrieved.Code, testData.Code)
		}

		if retrieved.Verified != testData.Verified {
			t.Errorf("Verified = %v, want %v", retrieved.Verified, testData.Verified)
		}
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		_, err := cacheManager.Get(ctx, "non-existent-key")
		if err == nil {
			t.Error("Get() expected error for non-existent key")
		}
	})

	t.Run("Delete cache data", func(t *testing.T) {
		testData := &CacheData{
			ID:        "test-captcha-delete",
			Code:      "DELETE",
			CreatedAt: time.Now().Unix(),
		}

		cacheManager.Set(ctx, testData.ID, testData)

		err := cacheManager.Delete(ctx, testData.ID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		_, err = cacheManager.Get(ctx, testData.ID)
		if err == nil {
			t.Error("Get() expected error after Delete()")
		}
	})

	t.Run("Check existence", func(t *testing.T) {
		testData := &CacheData{
			ID:        "test-captcha-exists",
			Code:      "EXISTS",
			CreatedAt: time.Now().Unix(),
		}

		cacheManager.Set(ctx, testData.ID, testData)

		exists, err := cacheManager.Exists(ctx, testData.ID)
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if !exists {
			t.Error("Exists() = false, want true")
		}

		exists, _ = cacheManager.Exists(ctx, "non-existent")
		if exists {
			t.Error("Exists() = true for non-existent key, want false")
		}
	})
}

// 测试验证功能
func TestVerify(t *testing.T) {
	cfg := &config.CaptchaConfig{
		ExpireMinutes: 5,
	}

	cacheManager := NewMockCacheManager(cfg)
	verifyService := &VerifyService{
		cache: cacheManager,
		cfg:   cfg,
	}

	ctx := context.Background()

	t.Run("Verify with correct code (case insensitive)", func(t *testing.T) {
		cacheData := &CacheData{
			ID:        "verify-correct-123",
			Code:      "TestCode",
			CreatedAt: time.Now().Unix(),
			Verified:  false,
		}

		err := cacheManager.Set(ctx, cacheData.ID, cacheData)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		req := &VerifyRequest{
			CaptchaID: cacheData.ID,
			Code:      "testcode",
		}

		result, err := verifyService.Verify(ctx, req)
		if err != nil {
			t.Fatalf("Verify() error = %v", err)
		}

		if !result.Success {
			t.Errorf("Verify() Success = false, want true. Message: %s", result.Message)
		}

		// 验证成功后应该删除缓存
		_, err = cacheManager.Get(ctx, cacheData.ID)
		if err == nil {
			t.Error("Cache should be deleted after successful verification")
		}
	})

	t.Run("Verify with wrong code", func(t *testing.T) {
		cacheData := &CacheData{
			ID:        "verify-wrong-123",
			Code:      "CorrectCode",
			CreatedAt: time.Now().Unix(),
			Verified:  false,
		}

		cacheManager.Set(ctx, cacheData.ID, cacheData)

		req := &VerifyRequest{
			CaptchaID: cacheData.ID,
			Code:      "WrongCode",
		}

		result, err := verifyService.Verify(ctx, req)
		if err == nil {
			t.Error("Verify() expected error for wrong code")
		}

		if result.Success {
			t.Error("Verify() with wrong code should return Success=false")
		}
	})

	t.Run("Verify with non-existent captcha", func(t *testing.T) {
		req := &VerifyRequest{
			CaptchaID: "non-existent-captcha",
			Code:      "anycode",
		}

		result, err := verifyService.Verify(ctx, req)
		if err == nil {
			t.Error("Verify() expected error for non-existent captcha")
		}

		if result.Success {
			t.Error("Verify() with non-existent captcha should return Success=false")
		}
	})

	t.Run("Verify with empty captcha ID", func(t *testing.T) {
		req := &VerifyRequest{
			CaptchaID: "",
			Code:      "anycode",
		}

		result, err := verifyService.Verify(ctx, req)
		if err == nil {
			t.Error("Verify() expected error for empty captcha ID")
		}

		if result.Success {
			t.Error("Verify() with empty ID should return Success=false")
		}
	})

	t.Run("Verify with already verified captcha", func(t *testing.T) {
		cacheData := &CacheData{
			ID:        "verify-already-123",
			Code:      "AlreadyVerified",
			CreatedAt: time.Now().Unix(),
			Verified:  true,
		}

		cacheManager.Set(ctx, cacheData.ID, cacheData)

		req := &VerifyRequest{
			CaptchaID: cacheData.ID,
			Code:      "AlreadyVerified",
		}

		result, err := verifyService.Verify(ctx, req)
		if err == nil {
			t.Error("Verify() expected error for already verified captcha")
		}

		if result.Success {
			t.Error("Verify() with already verified captcha should return Success=false")
		}
	})

	t.Run("Verify with leading/trailing whitespace", func(t *testing.T) {
		cacheData := &CacheData{
			ID:        "verify-whitespace-123",
			Code:      "CleanCode",
			CreatedAt: time.Now().Unix(),
			Verified:  false,
		}

		cacheManager.Set(ctx, cacheData.ID, cacheData)

		req := &VerifyRequest{
			CaptchaID: cacheData.ID,
			Code:      "  CleanCode  ",
		}

		result, err := verifyService.Verify(ctx, req)
		if err != nil {
			t.Fatalf("Verify() error = %v", err)
		}

		if !result.Success {
			t.Errorf("Verify() with whitespace should succeed. Message: %s", result.Message)
		}
	})
}

// 测试 MarkVerified
func TestMarkVerified(t *testing.T) {
	cfg := &config.CaptchaConfig{
		ExpireMinutes: 5,
	}

	cacheManager := NewMockCacheManager(cfg)
	ctx := context.Background()

	t.Run("Mark verified successfully", func(t *testing.T) {
		cacheData := &CacheData{
			ID:        "mark-verified-123",
			Code:      "MarkMe",
			CreatedAt: time.Now().Unix(),
			Verified:  false,
		}

		cacheManager.Set(ctx, cacheData.ID, cacheData)

		err := cacheManager.MarkVerified(ctx, cacheData.ID)
		if err != nil {
			t.Fatalf("MarkVerified() error = %v", err)
		}

		retrieved, _ := cacheManager.Get(ctx, cacheData.ID)
		if !retrieved.Verified {
			t.Error("Verified should be true after MarkVerified()")
		}
	})

	t.Run("Mark verified twice should fail", func(t *testing.T) {
		cacheData := &CacheData{
			ID:        "mark-verified-twice-123",
			Code:      "MarkTwice",
			CreatedAt: time.Now().Unix(),
			Verified:  false,
		}

		cacheManager.Set(ctx, cacheData.ID, cacheData)

		cacheManager.MarkVerified(ctx, cacheData.ID)

		err := cacheManager.MarkVerified(ctx, cacheData.ID)
		if err == nil {
			t.Error("MarkVerified() should fail when already verified")
		}
	})
}
