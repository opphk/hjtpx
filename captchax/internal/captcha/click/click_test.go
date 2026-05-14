package click

import (
	"context"
	"testing"
	"time"
)

func TestRemoveDuplicateChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty string", "", 0},
		{"all unique", "abc", 3},
		{"all duplicates", "aaa", 1},
		{"mixed", "abca", 3},
		{"chinese chars", "中中文", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeDuplicateChars(tt.input)
			if len(result) != tt.expected {
				t.Errorf("removeDuplicateChars(%q) = %d chars, want %d", tt.input, len(result), tt.expected)
			}
		})
	}
}

func TestRandomInt(t *testing.T) {
	tests := []struct {
		name string
		min  int
		max  int
	}{
		{"small range", 1, 10},
		{"large range", 0, 100},
		{"negative range", -50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 100; i++ {
				result := randomInt(tt.min, tt.max)
				if result < tt.min || result > tt.max {
					t.Errorf("randomInt(%d, %d) = %d, want between %d and %d", tt.min, tt.max, result, tt.min, tt.max)
				}
			}
		})
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-100, 100},
	}

	for _, tt := range tests {
		result := absInt(tt.input)
		if result != tt.expected {
			t.Errorf("absInt(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestGenerateID(t *testing.T) {
	id1, err := generateID()
	if err != nil {
		t.Fatalf("generateID() error = %v", err)
	}

	if len(id1) == 0 {
		t.Error("generateID() returned empty string")
	}

	if id1[:6] != "click_" {
		t.Errorf("generateID() = %s, want prefix 'click_'", id1)
	}

	id2, _ := generateID()
	if id1 == id2 {
		t.Error("generateID() returned duplicate IDs")
	}
}

func TestCaptchaGenerator(t *testing.T) {
	gen, err := NewCaptchaGenerator()
	if err != nil {
		t.Skipf("Skipping test: font not available (%v)", err)
	}

	t.Run("GenerateCaptcha default", func(t *testing.T) {
		captcha, err := gen.GenerateCaptcha(4)
		if err != nil {
			t.Fatalf("GenerateCaptcha(4) error = %v", err)
		}

		if captcha.ID == "" {
			t.Error("Captcha ID is empty")
		}

		if len(captcha.TargetChars) != 4 {
			t.Errorf("TargetChars length = %d, want 4", len(captcha.TargetChars))
		}

		if len(captcha.CharPositions) != 4 {
			t.Errorf("CharPositions length = %d, want 4", len(captcha.CharPositions))
		}

		if captcha.Image == "" {
			t.Error("Captcha Image is empty")
		}
	})

	t.Run("GenerateCaptcha min chars", func(t *testing.T) {
		captcha, err := gen.GenerateCaptcha(2)
		if err != nil {
			t.Fatalf("GenerateCaptcha(2) error = %v", err)
		}

		if len(captcha.TargetChars) != 3 {
			t.Errorf("TargetChars length = %d, want 3 (minimum enforced)", len(captcha.TargetChars))
		}
	})

	t.Run("GenerateCaptcha max chars", func(t *testing.T) {
		captcha, err := gen.GenerateCaptcha(10)
		if err != nil {
			t.Fatalf("GenerateCaptcha(10) error = %v", err)
		}

		if len(captcha.TargetChars) != 5 {
			t.Errorf("TargetChars length = %d, want 5 (maximum enforced)", len(captcha.TargetChars))
		}
	})

	t.Run("SelectRandomChars uniqueness", func(t *testing.T) {
		chars, err := gen.selectRandomChars(10)
		if err != nil {
			t.Fatalf("selectRandomChars(10) error = %v", err)
		}

		seen := make(map[string]bool)
		for _, c := range chars {
			if seen[c] {
				t.Errorf("selectRandomChars returned duplicate char: %s", c)
			}
			seen[c] = true
		}
	})
}

func TestMockCacheManager(t *testing.T) {
	cache := NewMockCacheManager()
	ctx := context.Background()

	t.Run("Store and Get", func(t *testing.T) {
		data := &CaptchaData{
			ID:          "test_id_1",
			TargetChars: []string{"中", "心", "测", "试"},
			CharPositions: []CharPosition{
				{Char: "中", X: 50, Y: 80, Width: 40, Height: 50},
				{Char: "心", X: 125, Y: 80, Width: 40, Height: 50},
				{Char: "测", X: 200, Y: 80, Width: 40, Height: 50},
				{Char: "试", X: 275, Y: 80, Width: 40, Height: 50},
			},
			CreatedAt: time.Now(),
		}

		err := cache.Store(ctx, data)
		if err != nil {
			t.Fatalf("Store() error = %v", err)
		}

		retrieved, err := cache.Get(ctx, "test_id_1")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if retrieved.ID != data.ID {
			t.Errorf("ID = %s, want %s", retrieved.ID, data.ID)
		}

		if len(retrieved.TargetChars) != len(data.TargetChars) {
			t.Errorf("TargetChars length = %d, want %d", len(retrieved.TargetChars), len(data.TargetChars))
		}
	})

	t.Run("Get non-existent", func(t *testing.T) {
		_, err := cache.Get(ctx, "non_existent_id")
		if err == nil {
			t.Error("Get() expected error for non-existent key")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		data := &CaptchaData{
			ID:        "test_id_delete",
			CreatedAt: time.Now(),
		}

		cache.Store(ctx, data)
		cache.Delete(ctx, "test_id_delete")

		_, err := cache.Get(ctx, "test_id_delete")
		if err == nil {
			t.Error("Get() expected error after Delete()")
		}
	})

	t.Run("Exists", func(t *testing.T) {
		data := &CaptchaData{
			ID:        "test_id_exists",
			CreatedAt: time.Now(),
		}

		cache.Store(ctx, data)

		exists, err := cache.Exists(ctx, "test_id_exists")
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if !exists {
			t.Error("Exists() = false, want true")
		}

		exists, _ = cache.Exists(ctx, "non_existent")
		if exists {
			t.Error("Exists() = true for non-existent key, want false")
		}
	})

	t.Run("Close", func(t *testing.T) {
		if err := cache.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
}

func TestClickVerifier(t *testing.T) {
	cache := NewMockCacheManager()
	verifier := NewClickVerifier(cache)
	ctx := context.Background()

	captchaData := &CaptchaData{
		ID:          "verify_test_id",
		TargetChars: []string{"中", "心", "测", "试"},
		CharPositions: []CharPosition{
			{Char: "中", X: 50, Y: 80, Width: 40, Height: 50},
			{Char: "心", X: 125, Y: 80, Width: 40, Height: 50},
			{Char: "测", X: 200, Y: 80, Width: 40, Height: 50},
			{Char: "试", X: 275, Y: 80, Width: 40, Height: 50},
		},
		CreatedAt: time.Now(),
	}

	t.Run("Empty captcha ID", func(t *testing.T) {
		req := &VerifyRequest{
			CaptchaID: "",
			Clicks:    []ClickPosition{{X: 70, Y: 105}},
		}

		resp, _ := verifier.Verify(ctx, req)
		if resp.Success {
			t.Error("Verify() with empty ID should return Success=false")
		}
		if resp.Message != "captcha ID is required" {
			t.Errorf("Message = %s, want 'captcha ID is required'", resp.Message)
		}
	})

	t.Run("Empty clicks", func(t *testing.T) {
		req := &VerifyRequest{
			CaptchaID: "some_id",
			Clicks:    []ClickPosition{},
		}

		resp, _ := verifier.Verify(ctx, req)
		if resp.Success {
			t.Error("Verify() with no clicks should return Success=false")
		}
	})

	t.Run("Invalid click count", func(t *testing.T) {
		cache.Store(ctx, captchaData)

		req := &VerifyRequest{
			CaptchaID: "verify_test_id",
			Clicks:    []ClickPosition{{X: 70, Y: 105}},
		}

		resp, _ := verifier.Verify(ctx, req)
		if resp.Success {
			t.Error("Verify() with wrong click count should return Success=false")
		}
	})

	t.Run("Correct verification", func(t *testing.T) {
		cache.Store(ctx, captchaData)

		clicks := []ClickPosition{
			{X: 70, Y: 105},
			{X: 145, Y: 105},
			{X: 220, Y: 105},
			{X: 295, Y: 105},
		}

		req := &VerifyRequest{
			CaptchaID: "verify_test_id",
			Clicks:    clicks,
		}

		resp, _ := verifier.Verify(ctx, req)
		if !resp.Success {
			t.Errorf("Verify() Success = false, want true. Message: %s", resp.Message)
		}
	})
}

func TestVerifyWithTolerance(t *testing.T) {
	tests := []struct {
		name      string
		x1, y1    int
		x2, y2    int
		tolerance int
		expected  bool
	}{
		{"exact match", 100, 100, 100, 100, 0, true},
		{"within tolerance", 100, 100, 105, 105, 10, true},
		{"outside tolerance", 100, 100, 200, 200, 10, false},
		{"zero distance", 50, 50, 50, 50, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyWithTolerance(tt.x1, tt.y1, tt.x2, tt.y2, tt.tolerance)
			if result != tt.expected {
				t.Errorf("VerifyWithTolerance(%d, %d, %d, %d, %d) = %v, want %v",
					tt.x1, tt.y1, tt.x2, tt.y2, tt.tolerance, result, tt.expected)
			}
		})
	}
}

func TestCalculateDistance(t *testing.T) {
	tests := []struct {
		name     string
		x1, y1   int
		x2, y2   int
		expected float64
	}{
		{"zero distance", 0, 0, 0, 0, 0},
		{"same x", 0, 0, 0, 10, 10},
		{"same y", 0, 0, 10, 0, 10},
		{"diagonal", 0, 0, 3, 4, 5},
		{"negative coords", -5, -5, 0, 0, 7.07},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateDistance(tt.x1, tt.y1, tt.x2, tt.y2)
			diff := result - tt.expected
			if diff < -0.1 || diff > 0.1 {
				t.Errorf("CalculateDistance(%d, %d, %d, %d) = %f, want ~%f",
					tt.x1, tt.y1, tt.x2, tt.y2, result, tt.expected)
			}
		})
	}
}

func TestCharPosition(t *testing.T) {
	pos := CharPosition{
		Char:   "中",
		X:      50,
		Y:      80,
		Width:  40,
		Height: 50,
	}

	centerX := pos.X + pos.Width/2
	centerY := pos.Y + pos.Height/2

	if centerX != 70 {
		t.Errorf("CenterX = %d, want 70", centerX)
	}

	if centerY != 105 {
		t.Errorf("CenterY = %d, want 105", centerY)
	}
}
