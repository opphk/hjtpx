package slider

import (
	"captchax/config"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

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
	return "", errors.New("redis: nil")
}

func (m *MockRedisClient) GetBytes(ctx context.Context, key string) ([]byte, error) {
	if val, ok := m.data[key]; ok {
		return []byte(val), nil
	}
	return nil, errors.New("redis: nil")
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
		fmt.Sscanf(s, "%d", &val)
	}
	val++
	m.data[key] = fmt.Sprintf("%d", val)
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

func (cm *MockCacheManager) Set(ctx context.Context, id string, data *CacheData) error {
	key := fmt.Sprintf("captcha:slider:%s", id)

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
	key := fmt.Sprintf("captcha:slider:%s", id)

	dataStr, err := cm.redis.Get(ctx, key)
	if err != nil {
		if err.Error() == "redis: nil" {
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
	key := fmt.Sprintf("captcha:slider:%s", id)
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
	key := fmt.Sprintf("captcha:slider:%s", id)
	count, err := cm.redis.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return count > 0, nil
}

func (cm *MockCacheManager) keyForID(id string) string {
	return fmt.Sprintf("captcha:slider:%s", id)
}

type MockSliderGenerator struct {
	cfg *config.CaptchaConfig
}

func NewMockSliderGenerator(cfg *config.CaptchaConfig) *MockSliderGenerator {
	return &MockSliderGenerator{cfg: cfg}
}

func TestGenerateCaptcha(t *testing.T) {
	cfg := &config.CaptchaConfig{
		Width:         300,
		Height:        200,
		SliderSize:    50,
		Tolerance:     5,
		ExpireMinutes: 5,
	}

	t.Run("Generate captcha creates valid result", func(t *testing.T) {
		slider := &Slider{
			cfg:   cfg,
			redis: nil,
		}

		result, err := slider.GenerateCaptcha(context.Background())
		if err != nil {
			t.Fatalf("GenerateCaptcha() error = %v", err)
		}

		if result.ID == "" {
			t.Error("GenerateCaptcha() returned empty ID")
		}

		if result.BackgroundB64 == "" {
			t.Error("GenerateCaptcha() returned empty BackgroundB64")
		}

		if result.SliderB64 == "" {
			t.Error("GenerateCaptcha() returned empty SliderB64")
		}

		if result.TargetX <= 0 {
			t.Errorf("TargetX = %d, want > 0", result.TargetX)
		}

		if result.TargetY <= 0 {
			t.Errorf("TargetY = %d, want > 0", result.TargetY)
		}
	})
}

func TestCacheSetGet(t *testing.T) {
	cfg := &config.CaptchaConfig{
		ExpireMinutes: 5,
	}

	cacheManager := NewMockCacheManager(cfg)

	ctx := context.Background()

	t.Run("Set and Get cache data", func(t *testing.T) {
		testData := &CacheData{
			ID:        "test-captcha-123",
			TargetX:   150,
			TargetY:   100,
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

		if retrieved.TargetX != testData.TargetX {
			t.Errorf("TargetX = %d, want %d", retrieved.TargetX, testData.TargetX)
		}

		if retrieved.TargetY != testData.TargetY {
			t.Errorf("TargetY = %d, want %d", retrieved.TargetY, testData.TargetY)
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
			TargetX:   100,
			TargetY:   100,
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
			TargetX:   100,
			TargetY:   100,
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

func TestVerifyCorrect(t *testing.T) {
	cfg := &config.CaptchaConfig{
		Tolerance:     5,
		ExpireMinutes: 5,
	}

	cacheManager := NewMockCacheManager(cfg)

	verifyService := &VerifyService{
		cache: cacheManager,
		cfg:   cfg,
	}

	ctx := context.Background()

	t.Run("Verify with correct position", func(t *testing.T) {
		targetX := 150
		targetY := 100

		cacheData := &CacheData{
			ID:        "verify-correct-123",
			TargetX:   targetX,
			TargetY:   targetY,
			CreatedAt: time.Now().Unix(),
			Verified:  false,
		}

		err := cacheManager.Set(ctx, cacheData.ID, cacheData)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		req := &VerifyRequest{
			CaptchaID: cacheData.ID,
			TargetX:   targetX,
			TargetY:   targetY,
		}

		result, err := verifyService.Verify(ctx, req)
		if err != nil {
			t.Fatalf("Verify() error = %v", err)
		}

		if !result.Success {
			t.Errorf("Verify() Success = false, want true. Message: %s", result.Message)
		}

		if result.Message != "verification successful" {
			t.Errorf("Message = %s, want 'verification successful'", result.Message)
		}
	})

	t.Run("Verify with position within tolerance", func(t *testing.T) {
		targetX := 150
		targetY := 100

		cacheData := &CacheData{
			ID:        "verify-tolerance-123",
			TargetX:   targetX,
			TargetY:   targetY,
			CreatedAt: time.Now().Unix(),
			Verified:  false,
		}

		cacheManager.Set(ctx, cacheData.ID, cacheData)

		req := &VerifyRequest{
			CaptchaID: cacheData.ID,
			TargetX:   targetX + 3,
			TargetY:   targetY + 3,
		}

		result, err := verifyService.Verify(ctx, req)
		if err != nil {
			t.Fatalf("Verify() error = %v", err)
		}

		if !result.Success {
			t.Errorf("Verify() with position within tolerance should succeed. Message: %s", result.Message)
		}
	})
}

func TestVerifyWrong(t *testing.T) {
	cfg := &config.CaptchaConfig{
		Tolerance:     5,
		ExpireMinutes: 5,
	}

	cacheManager := NewMockCacheManager(cfg)

	verifyService := &VerifyService{
		cache: cacheManager,
		cfg:   cfg,
	}

	ctx := context.Background()

	t.Run("Verify with wrong position", func(t *testing.T) {
		cacheData := &CacheData{
			ID:        "verify-wrong-123",
			TargetX:   150,
			TargetY:   100,
			CreatedAt: time.Now().Unix(),
			Verified:  false,
		}

		cacheManager.Set(ctx, cacheData.ID, cacheData)

		req := &VerifyRequest{
			CaptchaID: cacheData.ID,
			TargetX:   200,
			TargetY:   200,
		}

		result, err := verifyService.Verify(ctx, req)
		if err == nil {
			t.Error("Verify() expected error for wrong position")
		}

		if result.Success {
			t.Error("Verify() with wrong position should return Success=false")
		}
	})

	t.Run("Verify with non-existent captcha", func(t *testing.T) {
		req := &VerifyRequest{
			CaptchaID: "non-existent-captcha",
			TargetX:   100,
			TargetY:   100,
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
			TargetX:   100,
			TargetY:   100,
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
			TargetX:   150,
			TargetY:   100,
			CreatedAt: time.Now().Unix(),
			Verified:  true,
		}

		cacheManager.Set(ctx, cacheData.ID, cacheData)

		req := &VerifyRequest{
			CaptchaID: cacheData.ID,
			TargetX:   150,
			TargetY:   100,
		}

		result, err := verifyService.Verify(ctx, req)
		if err == nil {
			t.Error("Verify() expected error for already verified captcha")
		}

		if result.Success {
			t.Error("Verify() with already verified captcha should return Success=false")
		}
	})
}

func TestValidatePosition(t *testing.T) {
	cfg := &config.CaptchaConfig{}
	verifyService := &VerifyService{cfg: cfg}

	t.Run("Valid position", func(t *testing.T) {
		err := verifyService.ValidatePosition(100, 100)
		if err != nil {
			t.Errorf("ValidatePosition(100, 100) error = %v, want nil", err)
		}
	})

	t.Run("Invalid negative X", func(t *testing.T) {
		err := verifyService.ValidatePosition(-1, 100)
		if err == nil {
			t.Error("ValidatePosition(-1, 100) expected error")
		}
	})

	t.Run("Invalid negative Y", func(t *testing.T) {
		err := verifyService.ValidatePosition(100, -1)
		if err == nil {
			t.Error("ValidatePosition(100, -1) expected error")
		}
	})
}

func TestAbsDiff(t *testing.T) {
	tests := []struct {
		name     string
		x, y     int
		expected int
	}{
		{"positive numbers", 10, 5, 5},
		{"negative difference", 5, 10, 5},
		{"same numbers", 10, 10, 0},
		{"with zero", 10, 0, 10},
		{"negative input", -10, -5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := absDiff(tt.x, tt.y)
			if result != tt.expected {
				t.Errorf("absDiff(%d, %d) = %d, want %d", tt.x, tt.y, result, tt.expected)
			}
		})
	}
}

func TestIsWithinTolerance(t *testing.T) {
	verifyService := &VerifyService{cfg: &config.CaptchaConfig{Tolerance: 5}}

	tests := []struct {
		name     string
		actual   int
		expected int
		tol      int
		want     bool
	}{
		{"exact match", 100, 100, 5, true},
		{"within tolerance", 103, 100, 5, true},
		{"at tolerance boundary", 105, 100, 5, true},
		{"outside tolerance", 106, 100, 5, false},
		{"far outside tolerance", 200, 100, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifyService.isWithinTolerance(tt.actual, tt.expected, tt.tol)
			if result != tt.want {
				t.Errorf("isWithinTolerance(%d, %d, %d) = %v, want %v",
					tt.actual, tt.expected, tt.tol, result, tt.want)
			}
		})
	}
}

func TestKeyForID(t *testing.T) {
	cacheManager := NewMockCacheManager(&config.CaptchaConfig{})

	tests := []struct {
		id       string
		expected string
	}{
		{"test-123", "captcha:slider:test-123"},
		{"abc-def", "captcha:slider:abc-def"},
		{"", "captcha:slider:"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			result := cacheManager.keyForID(tt.id)
			if result != tt.expected {
				t.Errorf("keyForID(%s) = %s, want %s", tt.id, result, tt.expected)
			}
		})
	}
}

func TestMarkVerified(t *testing.T) {
	cfg := &config.CaptchaConfig{
		Tolerance:     5,
		ExpireMinutes: 5,
	}

	cacheManager := NewMockCacheManager(cfg)
	ctx := context.Background()

	t.Run("Mark verified successfully", func(t *testing.T) {
		cacheData := &CacheData{
			ID:        "mark-verified-123",
			TargetX:   150,
			TargetY:   100,
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
			TargetX:   150,
			TargetY:   100,
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
