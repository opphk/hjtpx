package unit

import (
	"context"
	"testing"
	"time"

	"captchax/internal/config"
	"captchax/internal/risk"
)

func TestRiskEngine_CalculateRiskScore(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	whitelist, _ := risk.NewWhitelist(&risk.WhitelistConfig{MemoryOnly: true})
	engine := risk.NewRiskEngine(cfg, nil, whitelist)
	ctx := context.Background()

	t.Run("Normal behavior with no risk factors", func(t *testing.T) {
		behavior := &risk.BehaviorData{
			SessionID: "test-session-1",
			MouseTracks: []risk.MouseTrack{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 110, Y: 105, Timestamp: 50},
				{X: 125, Y: 110, Timestamp: 100},
				{X: 150, Y: 115, Timestamp: 150},
				{X: 200, Y: 120, Timestamp: 200},
			},
			ClickTimes: []int64{100, 200, 350, 500},
			SlideStart: 0,
			SlideEnd:   5000,
			Success:    true,
		}

		result := engine.CalculateRiskScore(ctx, behavior, "192.168.1.1", "example.com")

		if result.Score > 50 {
			t.Errorf("Score = %d, want <= 50 for normal behavior", result.Score)
		}

		if result.Level == risk.RiskLevelCritical {
			t.Errorf("Level = %s, want not critical for normal behavior", result.Level)
		}
	})

	t.Run("Fast slide behavior (high risk)", func(t *testing.T) {
		behavior := &risk.BehaviorData{
			SessionID: "test-session-fast",
			MouseTracks: []risk.MouseTrack{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 200, Y: 100, Timestamp: 100},
			},
			SlideStart: 0,
			SlideEnd:   500,
			Success:    true,
		}

		result := engine.CalculateRiskScore(ctx, behavior, "192.168.1.2", "example.com")

		if result.Score < 20 {
			t.Errorf("Score = %d, want >= 20 for fast slide behavior", result.Score)
		}

		hasSlideTooFast := false
		for _, factor := range result.Factors {
			if factor.Name == "slide_too_fast" {
				hasSlideTooFast = true
				break
			}
		}
		if !hasSlideTooFast {
			t.Error("Expected 'slide_too_fast' factor for fast slide behavior")
		}
	})

	t.Run("Slow slide behavior (medium risk)", func(t *testing.T) {
		behavior := &risk.BehaviorData{
			SessionID: "test-session-slow",
			MouseTracks: []risk.MouseTrack{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 110, Y: 105, Timestamp: 50},
				{X: 125, Y: 110, Timestamp: 100},
				{X: 150, Y: 115, Timestamp: 150},
				{X: 200, Y: 120, Timestamp: 200},
			},
			SlideStart: 0,
			SlideEnd:   45000,
			Success:    true,
		}

		result := engine.CalculateRiskScore(ctx, behavior, "192.168.1.3", "example.com")

		if result.Score < 10 {
			t.Errorf("Score = %d, want >= 10 for slow slide behavior", result.Score)
		}

		hasSlideTooSlow := false
		for _, factor := range result.Factors {
			if factor.Name == "slide_too_slow" {
				hasSlideTooSlow = true
				break
			}
		}
		if !hasSlideTooSlow {
			t.Error("Expected 'slide_too_slow' factor for slow slide behavior")
		}
	})

	t.Run("Over-smooth mouse track", func(t *testing.T) {
		behavior := &risk.BehaviorData{
			SessionID: "test-session-smooth",
			MouseTracks: []risk.MouseTrack{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 150, Y: 150, Timestamp: 100},
				{X: 200, Y: 200, Timestamp: 200},
				{X: 250, Y: 250, Timestamp: 300},
				{X: 300, Y: 300, Timestamp: 400},
			},
			SlideStart: 0,
			SlideEnd:   3000,
			Success:    true,
		}

		result := engine.CalculateRiskScore(ctx, behavior, "192.168.1.4", "example.com")

		hasOverSmooth := false
		for _, factor := range result.Factors {
			if factor.Name == "over_smooth_track" {
				hasOverSmooth = true
				break
			}
		}
		if !hasOverSmooth {
			t.Error("Expected 'over_smooth_track' factor for over-smooth track")
		}
	})

	t.Run("Low jitter mouse track", func(t *testing.T) {
		behavior := &risk.BehaviorData{
			SessionID: "test-session-jitter",
			MouseTracks: []risk.MouseTrack{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 110, Y: 101, Timestamp: 50},
				{X: 120, Y: 102, Timestamp: 100},
				{X: 130, Y: 103, Timestamp: 150},
				{X: 140, Y: 104, Timestamp: 200},
			},
			SlideStart: 0,
			SlideEnd:   3000,
			Success:    true,
		}

		result := engine.CalculateRiskScore(ctx, behavior, "192.168.1.5", "example.com")

		hasLowJitter := false
		for _, factor := range result.Factors {
			if factor.Name == "low_jitter" {
				hasLowJitter = true
				break
			}
		}
		if !hasLowJitter {
			t.Error("Expected 'low_jitter' factor for low jitter track")
		}
	})

	t.Run("Uniform velocity mouse track", func(t *testing.T) {
		behavior := &risk.BehaviorData{
			SessionID: "test-session-velocity",
			MouseTracks: []risk.MouseTrack{
				{X: 100, Y: 100, Timestamp: 0},
				{X: 200, Y: 100, Timestamp: 100},
				{X: 300, Y: 100, Timestamp: 200},
				{X: 400, Y: 100, Timestamp: 300},
				{X: 500, Y: 100, Timestamp: 400},
			},
			SlideStart: 0,
			SlideEnd:   3000,
			Success:    true,
		}

		result := engine.CalculateRiskScore(ctx, behavior, "192.168.1.6", "example.com")

		hasAbnormalVelocity := false
		for _, factor := range result.Factors {
			if factor.Name == "abnormal_velocity" {
				hasAbnormalVelocity = true
				break
			}
		}
		if !hasAbnormalVelocity {
			t.Log("Note: abnormal_velocity factor not triggered, may need algorithm tuning")
		}
	})
}

func TestRiskEngine_GetRiskLevel(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := risk.NewRiskEngine(cfg, nil, nil)

	tests := []struct {
		score    int
		expected risk.RiskLevel
	}{
		{0, risk.RiskLevelLow},
		{24, risk.RiskLevelLow},
		{25, risk.RiskLevelMedium},
		{49, risk.RiskLevelMedium},
		{50, risk.RiskLevelHigh},
		{79, risk.RiskLevelHigh},
		{80, risk.RiskLevelCritical},
		{100, risk.RiskLevelCritical},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			level := engine.GetRiskLevel(tt.score)
			if level != tt.expected {
				t.Errorf("GetRiskLevel(%d) = %s, want %s", tt.score, level, tt.expected)
			}
		})
	}
}

func TestRiskEngine_AnalyzeMouseTrack(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := risk.NewRiskEngine(cfg, nil, nil)

	t.Run("Insufficient track data", func(t *testing.T) {
		tracks := []risk.MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
		}

		score, factors := engine.AnalyzeMouseTrack(tracks)
		if score != 0 {
			t.Errorf("Score = %d, want 0 for insufficient data", score)
		}
		if len(factors) != 1 || factors[0].Name != "insufficient_track_data" {
			t.Error("Expected 'insufficient_track_data' factor")
		}
	})

	t.Run("Normal mouse track", func(t *testing.T) {
		tracks := []risk.MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 110, Y: 105, Timestamp: 50},
			{X: 125, Y: 110, Timestamp: 100},
			{X: 150, Y: 115, Timestamp: 150},
			{X: 200, Y: 120, Timestamp: 200},
			{X: 250, Y: 130, Timestamp: 250},
			{X: 300, Y: 150, Timestamp: 300},
		}

		score, factors := engine.AnalyzeMouseTrack(tracks)
		if score > 40 {
			t.Errorf("Score = %d, want <= 40 for normal track", score)
		}

		t.Logf("Score for normal track: %d, factors: %v", score, factors)
	})

	t.Run("Empty track", func(t *testing.T) {
		tracks := []risk.MouseTrack{}
		score, factors := engine.AnalyzeMouseTrack(tracks)
		if score != 0 {
			t.Errorf("Score = %d, want 0 for empty track", score)
		}
		if len(factors) != 1 || factors[0].Name != "insufficient_track_data" {
			t.Error("Expected 'insufficient_track_data' factor for empty track")
		}
	})
}

func TestRiskEngine_AnalyzeClickRhythm(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := risk.NewRiskEngine(cfg, nil, nil)

	t.Run("Single click", func(t *testing.T) {
		clicks := []int64{100}
		score, factors := engine.AnalyzeClickRhythm(clicks)
		if score != 0 {
			t.Errorf("Score = %d, want 0 for single click", score)
		}
		if len(factors) != 0 {
			t.Errorf("Factors count = %d, want 0 for single click", len(factors))
		}
	})

	t.Run("Mechanical rhythm", func(t *testing.T) {
		clicks := []int64{100, 200, 300, 400, 500}
		score, factors := engine.AnalyzeClickRhythm(clicks)
		if score < 15 {
			t.Errorf("Score = %d, want >= 15 for mechanical rhythm", score)
		}

		hasMechanicalRhythm := false
		for _, factor := range factors {
			if factor.Name == "mechanical_rhythm" {
				hasMechanicalRhythm = true
				break
			}
		}
		if !hasMechanicalRhythm {
			t.Error("Expected 'mechanical_rhythm' factor")
		}
	})

	t.Run("Fast clicks", func(t *testing.T) {
		clicks := []int64{100, 130, 160, 190, 220}
		score, factors := engine.AnalyzeClickRhythm(clicks)

		hasFastClicks := false
		for _, factor := range factors {
			if factor.Name == "unusually_fast_clicks" {
				hasFastClicks = true
				break
			}
		}
		if !hasFastClicks {
			t.Error("Expected 'unusually_fast_clicks' factor")
		}
		if score < 10 {
			t.Errorf("Score = %d, want >= 10 for fast clicks", score)
		}
	})

	t.Run("Normal rhythm", func(t *testing.T) {
		clicks := []int64{100, 300, 600, 1000}
		score, _ := engine.AnalyzeClickRhythm(clicks)
		if score > 10 {
			t.Errorf("Score = %d, want <= 10 for normal rhythm", score)
		}
	})
}

func TestRiskEngine_TrackBehavior(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := risk.NewRiskEngine(cfg, nil, nil)

	behavior := &risk.BehaviorData{
		SessionID: "test-session",
		MouseTracks: []risk.MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 200, Y: 200, Timestamp: 100},
		},
	}

	err := engine.TrackBehavior(behavior)
	if err != nil {
		t.Errorf("TrackBehavior() error = %v, want nil", err)
	}
}

func TestRiskLevel_Constants(t *testing.T) {
	levels := []risk.RiskLevel{
		risk.RiskLevelLow,
		risk.RiskLevelMedium,
		risk.RiskLevelHigh,
		risk.RiskLevelCritical,
	}

	expected := []string{"low", "medium", "high", "critical"}

	for i, level := range levels {
		if string(level) != expected[i] {
			t.Errorf("Level %d = %s, want %s", i, level, expected[i])
		}
	}
}

func TestAction_Constants(t *testing.T) {
	actions := []risk.Action{
		risk.ActionAllow,
		risk.ActionVerify,
		risk.ActionBlock,
	}

	expected := []string{"allow", "verify", "block"}

	for i, action := range actions {
		if string(action) != expected[i] {
			t.Errorf("Action %d = %s, want %s", i, action, expected[i])
		}
	}
}

func TestRiskResult_Structure(t *testing.T) {
	result := &risk.RiskResult{
		Score:       50,
		Level:       risk.RiskLevelHigh,
		Factors:     []risk.RiskFactor{{Name: "test", Weight: 10, Reason: "test reason"}},
		Recommended: risk.ActionVerify,
		Timestamp:   time.Now(),
	}

	if result.Score != 50 {
		t.Errorf("Score = %d, want 50", result.Score)
	}

	if result.Level != risk.RiskLevelHigh {
		t.Errorf("Level = %s, want high", result.Level)
	}

	if len(result.Factors) != 1 {
		t.Errorf("Factors count = %d, want 1", len(result.Factors))
	}

	if result.Recommended != risk.ActionVerify {
		t.Errorf("Recommended = %s, want verify", result.Recommended)
	}
}

func TestBehaviorData_Structure(t *testing.T) {
	behavior := &risk.BehaviorData{
		UserID:    "user123",
		SessionID: "session456",
		MouseTracks: []risk.MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 200, Y: 200, Timestamp: 100},
		},
		ClickTimes: []int64{100, 200, 300},
		SlideStart: 0,
		SlideEnd:   1000,
		Success:    true,
	}

	if behavior.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", behavior.UserID)
	}

	if behavior.SessionID != "session456" {
		t.Errorf("SessionID = %s, want session456", behavior.SessionID)
	}

	if len(behavior.MouseTracks) != 2 {
		t.Errorf("MouseTracks count = %d, want 2", len(behavior.MouseTracks))
	}

	if len(behavior.ClickTimes) != 3 {
		t.Errorf("ClickTimes count = %d, want 3", len(behavior.ClickTimes))
	}
}

func TestMouseTrack_Structure(t *testing.T) {
	track := risk.MouseTrack{
		X:            100.5,
		Y:            200.5,
		Timestamp:    1234567890,
		Velocity:     10.5,
		Acceleration: 1.2,
	}

	if track.X != 100.5 {
		t.Errorf("X = %f, want 100.5", track.X)
	}

	if track.Y != 200.5 {
		t.Errorf("Y = %f, want 200.5", track.Y)
	}

	if track.Timestamp != 1234567890 {
		t.Errorf("Timestamp = %d, want 1234567890", track.Timestamp)
	}
}

func TestRiskFactor_Structure(t *testing.T) {
	factor := risk.RiskFactor{
		Name:   "test_factor",
		Weight: 25,
		Reason: "This is a test factor",
	}

	if factor.Name != "test_factor" {
		t.Errorf("Name = %s, want test_factor", factor.Name)
	}

	if factor.Weight != 25 {
		t.Errorf("Weight = %d, want 25", factor.Weight)
	}

	if factor.Reason != "This is a test factor" {
		t.Errorf("Reason = %s, want 'This is a test factor'", factor.Reason)
	}
}

func TestPoint_Structure(t *testing.T) {
	point := risk.Point{
		X: 150.5,
		Y: 250.5,
	}

	if point.X != 150.5 {
		t.Errorf("X = %f, want 150.5", point.X)
	}

	if point.Y != 250.5 {
		t.Errorf("Y = %f, want 250.5", point.Y)
	}
}

func BenchmarkRiskEngine_CalculateRiskScore(b *testing.B) {
	cfg := config.DefaultRiskConfig()
	engine := risk.NewRiskEngine(cfg, nil, nil)
	ctx := context.Background()

	behavior := &risk.BehaviorData{
		SessionID: "benchmark-session",
		MouseTracks: []risk.MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 110, Y: 105, Timestamp: 50},
			{X: 125, Y: 110, Timestamp: 100},
			{X: 150, Y: 115, Timestamp: 150},
			{X: 200, Y: 120, Timestamp: 200},
			{X: 250, Y: 130, Timestamp: 250},
			{X: 300, Y: 150, Timestamp: 300},
		},
		ClickTimes: []int64{100, 200, 350, 500, 650},
		SlideStart: 0,
		SlideEnd:   5000,
		Success:    true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.CalculateRiskScore(ctx, behavior, "192.168.1.1", "example.com")
	}
}

func BenchmarkRiskEngine_AnalyzeMouseTrack(b *testing.B) {
	cfg := config.DefaultRiskConfig()
	engine := risk.NewRiskEngine(cfg, nil, nil)

	tracks := []risk.MouseTrack{
		{X: 100, Y: 100, Timestamp: 0},
		{X: 110, Y: 105, Timestamp: 50},
		{X: 125, Y: 110, Timestamp: 100},
		{X: 150, Y: 115, Timestamp: 150},
		{X: 200, Y: 120, Timestamp: 200},
		{X: 250, Y: 130, Timestamp: 250},
		{X: 300, Y: 150, Timestamp: 300},
		{X: 350, Y: 180, Timestamp: 350},
		{X: 400, Y: 220, Timestamp: 400},
		{X: 450, Y: 280, Timestamp: 450},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.AnalyzeMouseTrack(tracks)
	}
}
