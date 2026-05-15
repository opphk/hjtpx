package rotate

import (
	"captchax/config"
	"context"
	"testing"
	"time"
)

// MockCacheManager 用于测试的模拟缓存管理器
type MockCacheManager struct {
	data map[string]*CacheData
}

func NewMockCacheManager() *MockCacheManager {
	return &MockCacheManager{
		data: make(map[string]*CacheData),
	}
}

func (m *MockCacheManager) Set(ctx context.Context, id string, data *CacheData) error {
	m.data[id] = data
	return nil
}

func (m *MockCacheManager) Get(ctx context.Context, id string) (*CacheData, error) {
	data, ok := m.data[id]
	if !ok {
		return nil, ErrCaptchaNotFound
	}
	return data, nil
}

func (m *MockCacheManager) Delete(ctx context.Context, id string) error {
	delete(m.data, id)
	return nil
}

func (m *MockCacheManager) Exists(ctx context.Context, id string) (bool, error) {
	_, ok := m.data[id]
	return ok, nil
}

// TestNormalizeAngle 测试角度标准化
func TestNormalizeAngle(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"零角度", 0, 0},
		{"标准角度", 45, 45},
		{"360度", 360, 0},
		{"400度", 400, 40},
		{"负角度", -45, 315},
		{"负360度", -360, 0},
		{"超过一圈", 720, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeAngle(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeAngle(%d) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestAngleDiff 测试角度差计算
func TestAngleDiff(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"相同角度", 0, 0, 0},
		{"45度差", 0, 45, 45},
		{"跨越360", 350, 10, 20},
		{"跨越0度反向", 10, 350, 20},
		{"最大差", 0, 180, 180},
		{"超过最大差", 0, 190, 170},
		{"负数差", -45, 45, 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := angleDiff(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("angleDiff(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestVerifySuccess 测试验证成功（±5度内）
func TestVerifySuccess(t *testing.T) {
	ctx := context.Background()
	mockCache := NewMockCacheManager()
	cfg := &config.CaptchaConfig{
		ExpireMinutes: 5,
	}
	verifyService := NewVerifyServiceWithInterface(cfg, mockCache)

	testID := "test-123"
	targetAngle := 90

	// 设置测试数据
	mockCache.Set(ctx, testID, &CacheData{
		ID:        testID,
		Angle:     targetAngle,
		CreatedAt: time.Now().Unix(),
		Verified:  false,
	})

	// 测试1：完全匹配
	req := &VerifyRequest{
		CaptchaID: testID,
		Angle:     targetAngle,
	}
	result, err := verifyService.Verify(ctx, req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got false")
	}
	if result.Message != "verification successful" {
		t.Errorf("unexpected message: %s", result.Message)
	}

	// 验证成功后数据应该被删除
	_, err = mockCache.Get(ctx, testID)
	if err != ErrCaptchaNotFound {
		t.Errorf("expected captcha to be deleted")
	}

	// 重新设置数据，测试+3度
	mockCache.Set(ctx, testID, &CacheData{
		ID:        testID,
		Angle:     targetAngle,
		CreatedAt: time.Now().Unix(),
		Verified:  false,
	})
	req = &VerifyRequest{
		CaptchaID: testID,
		Angle:     targetAngle + 3,
	}
	result, err = verifyService.Verify(ctx, req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success for +3 degrees, got false")
	}

	// 重新设置数据，测试-5度
	mockCache.Set(ctx, testID, &CacheData{
		ID:        testID,
		Angle:     targetAngle,
		CreatedAt: time.Now().Unix(),
		Verified:  false,
	})
	req = &VerifyRequest{
		CaptchaID: testID,
		Angle:     targetAngle - 5,
	}
	result, err = verifyService.Verify(ctx, req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success for -5 degrees, got false")
	}
}

// TestVerifyFailure 测试验证失败
func TestVerifyFailure(t *testing.T) {
	ctx := context.Background()
	mockCache := NewMockCacheManager()
	cfg := &config.CaptchaConfig{
		ExpireMinutes: 5,
	}
	verifyService := NewVerifyServiceWithInterface(cfg, mockCache)

	testID := "test-456"
	targetAngle := 90

	// 设置测试数据
	mockCache.Set(ctx, testID, &CacheData{
		ID:        testID,
		Angle:     targetAngle,
		CreatedAt: time.Now().Unix(),
		Verified:  false,
	})

	// 测试6度误差
	req := &VerifyRequest{
		CaptchaID: testID,
		Angle:     targetAngle + 6,
	}
	result, err := verifyService.Verify(ctx, req)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if result.Success {
		t.Errorf("expected failure, got success")
	}

	// 测试无效ID
	req = &VerifyRequest{
		CaptchaID: "invalid-id",
		Angle:     targetAngle,
	}
	result, err = verifyService.Verify(ctx, req)
	if err == nil {
		t.Errorf("expected error for invalid id, got nil")
	}
	if result.Success {
		t.Errorf("expected failure for invalid id, got success")
	}

	// 测试空ID
	req = &VerifyRequest{
		CaptchaID: "",
		Angle:     targetAngle,
	}
	result, err = verifyService.Verify(ctx, req)
	if err == nil {
		t.Errorf("expected error for empty id, got nil")
	}
	if result.Success {
		t.Errorf("expected failure for empty id, got success")
	}
}

// TestVerifyAlreadyVerified 测试重复验证
func TestVerifyAlreadyVerified(t *testing.T) {
	ctx := context.Background()
	mockCache := NewMockCacheManager()
	cfg := &config.CaptchaConfig{
		ExpireMinutes: 5,
	}
	verifyService := NewVerifyServiceWithInterface(cfg, mockCache)

	testID := "test-789"
	targetAngle := 90

	// 设置已验证的数据
	mockCache.Set(ctx, testID, &CacheData{
		ID:        testID,
		Angle:     targetAngle,
		CreatedAt: time.Now().Unix(),
		Verified:  true,
	})

	req := &VerifyRequest{
		CaptchaID: testID,
		Angle:     targetAngle,
	}
	result, err := verifyService.Verify(ctx, req)
	if err == nil {
		t.Errorf("expected error for already verified, got nil")
	}
	if result.Success {
		t.Errorf("expected failure for already verified, got success")
	}
	if result.Message != "captcha already verified" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

// TestRotateCaptcha 测试旋转验证码生成
func TestRotateCaptcha(t *testing.T) {
	ctx := context.Background()
	cfg := &config.CaptchaConfig{
		ExpireMinutes: 5,
		Width:         300,
		Height:        300,
	}

	// 使用 nil redis 测试
	rotateCaptcha := New(cfg, nil)

	result, err := rotateCaptcha.GenerateCaptcha(ctx)
	if err != nil {
		t.Fatalf("GenerateCaptcha failed: %v", err)
	}

	if result.ID == "" {
		t.Error("expected non-empty ID")
	}

	if result.ImageB64 == "" {
		t.Error("expected non-empty ImageB64")
	}

	if result.OriginalB64 == "" {
		t.Error("expected non-empty OriginalB64")
	}
}
