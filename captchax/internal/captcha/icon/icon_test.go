package icon

import (
	"captchax/config"
	"context"
	"testing"
	"time"
)

// MockCacheManager is a mock implementation of CacheManagerInterface
type MockCacheManager struct {
	data map[string]*CacheData
}

func NewMockCacheManager() *MockCacheManager {
	return &MockCacheManager{
		data: make(map[string]*CacheData),
	}
}

func (m *MockCacheManager) Get(ctx context.Context, id string) (*CacheData, error) {
	if d, ok := m.data[id]; ok {
		return d, nil
	}
	return nil, ErrCaptchaNotFound
}

func (m *MockCacheManager) Set(ctx context.Context, id string, data *CacheData) error {
	m.data[id] = data
	return nil
}

func (m *MockCacheManager) Delete(ctx context.Context, id string) error {
	delete(m.data, id)
	return nil
}

func (m *MockCacheManager) Exists(ctx context.Context, id string) (bool, error) {
	_, ok := m.data[id]
	return ok, nil
}

func TestLoadIconLibrary(t *testing.T) {
	icons := loadIconLibrary()
	if len(icons) == 0 {
		t.Error("Expected icon library to have icons")
	}
}

func TestGenerateCaptcha(t *testing.T) {
	cfg := &config.CaptchaConfig{
		ExpireMinutes: 5,
	}
	ic := New(cfg, nil) // nil Redis for test
	ctx := context.Background()
	
	result, err := ic.GenerateCaptcha(ctx)
	if err != nil {
		t.Fatalf("GenerateCaptcha failed: %v", err)
	}
	
	if result.ID == "" {
		t.Error("Expected captcha ID to be set")
	}
	
	if len(result.TargetIcons) < MinTargetIcons || len(result.TargetIcons) > MaxTargetIcons {
		t.Errorf("Target icons count %d out of range [%d, %d]", 
			len(result.TargetIcons), MinTargetIcons, MaxTargetIcons)
	}
	
	totalExpected := DefaultGridCols * DefaultGridRows
	if len(result.AllIcons) != totalExpected {
		t.Errorf("Expected %d icons, got %d", totalExpected, len(result.AllIcons))
	}
}

func TestVerifyIcons(t *testing.T) {
	cfg := &config.CaptchaConfig{
		ExpireMinutes: 5,
	}
	mockCache := NewMockCacheManager()
	vs := NewVerifyServiceWithInterface(cfg, mockCache)
	ctx := context.Background()
	
	// Setup test data
	testID := "test-captcha-id"
	targetIDs := []string{"icon-1", "icon-2", "icon-3"}
	mockCache.data[testID] = &CacheData{
		ID:            testID,
		TargetIconIDs: targetIDs,
		CreatedAt:     time.Now().Unix(),
		Verified:      false,
	}
	
	// Test valid verification
	req := &VerifyRequest{
		CaptchaID: testID,
		IconIDs:   targetIDs,
	}
	result, err := vs.Verify(ctx, req)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !result.Success {
		t.Error("Expected verification to succeed")
	}
	
	// Test captcha should be deleted after verification
	_, err = mockCache.Get(ctx, testID)
	if err == nil {
		t.Error("Expected captcha to be deleted after verification")
	}
	
	// Test with wrong icons
	mockCache.data[testID] = &CacheData{
		ID:            testID,
		TargetIconIDs: targetIDs,
		CreatedAt:     time.Now().Unix(),
		Verified:      false,
	}
	
	req = &VerifyRequest{
		CaptchaID: testID,
		IconIDs:   []string{"icon-1", "icon-2", "wrong-icon-id"},
	}
	result, _ = vs.Verify(ctx, req)
	if result.Success {
		t.Error("Expected verification to fail with wrong icons")
	}
}

func TestHelperFunctions(t *testing.T) {
	icons := loadIconLibrary()
	
	// Test selectRandomIcons
	selected := selectRandomIcons(icons, 3)
	if len(selected) != 3 {
		t.Error("Expected 3 selected icons")
	}
	
	// Test getIconIDs
	ids := getIconIDs(selected)
	if len(ids) != 3 {
		t.Error("Expected 3 icon IDs")
	}
	
	// Test selectRandomIconsExcluding
	excluded := selectRandomIconsExcluding(icons, 2, ids)
	for _, icon := range excluded {
		for _, excludedID := range ids {
			if icon.ID == excludedID {
				t.Error("Excluded icon should not be selected")
			}
		}
	}
}
