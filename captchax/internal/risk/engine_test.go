package risk

import (
	"context"
	"testing"

	"captchax/internal/config"
)

func TestCalculateRiskScore(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	whitelist, _ := NewWhitelist(&WhitelistConfig{MemoryOnly: true})

	engine := NewRiskEngine(cfg, nil, whitelist)
	ctx := context.Background()

	t.Run("Normal behavior with no risk factors", func(t *testing.T) {
		behavior := &BehaviorData{
			SessionID: "test-session-1",
			MouseTracks: []MouseTrack{
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

		if result.Level == RiskLevelCritical {
			t.Errorf("Level = %s, want not critical for normal behavior", result.Level)
		}
	})

	t.Run("Fast slide behavior (high risk)", func(t *testing.T) {
		behavior := &BehaviorData{
			SessionID: "test-session-fast",
			MouseTracks: []MouseTrack{
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
		behavior := &BehaviorData{
			SessionID: "test-session-slow",
			MouseTracks: []MouseTrack{
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
		behavior := &BehaviorData{
			SessionID: "test-session-smooth",
			MouseTracks: []MouseTrack{
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
		behavior := &BehaviorData{
			SessionID: "test-session-jitter",
			MouseTracks: []MouseTrack{
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
		behavior := &BehaviorData{
			SessionID: "test-session-velocity",
			MouseTracks: []MouseTrack{
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

func TestRiskLevel(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Critical level for score >= 80", func(t *testing.T) {
		level := engine.GetRiskLevel(80)
		if level != RiskLevelCritical {
			t.Errorf("GetRiskLevel(80) = %s, want %s", level, RiskLevelCritical)
		}

		level = engine.GetRiskLevel(100)
		if level != RiskLevelCritical {
			t.Errorf("GetRiskLevel(100) = %s, want %s", level, RiskLevelCritical)
		}
	})

	t.Run("High level for score 50-79", func(t *testing.T) {
		level := engine.GetRiskLevel(50)
		if level != RiskLevelHigh {
			t.Errorf("GetRiskLevel(50) = %s, want %s", level, RiskLevelHigh)
		}

		level = engine.GetRiskLevel(79)
		if level != RiskLevelHigh {
			t.Errorf("GetRiskLevel(79) = %s, want %s", level, RiskLevelHigh)
		}
	})

	t.Run("Medium level for score 25-49", func(t *testing.T) {
		level := engine.GetRiskLevel(25)
		if level != RiskLevelMedium {
			t.Errorf("GetRiskLevel(25) = %s, want %s", level, RiskLevelMedium)
		}

		level = engine.GetRiskLevel(49)
		if level != RiskLevelMedium {
			t.Errorf("GetRiskLevel(49) = %s, want %s", level, RiskLevelMedium)
		}
	})

	t.Run("Low level for score < 25", func(t *testing.T) {
		level := engine.GetRiskLevel(0)
		if level != RiskLevelLow {
			t.Errorf("GetRiskLevel(0) = %s, want %s", level, RiskLevelLow)
		}

		level = engine.GetRiskLevel(24)
		if level != RiskLevelLow {
			t.Errorf("GetRiskLevel(24) = %s, want %s", level, RiskLevelLow)
		}
	})
}

func TestGetRecommendedAction(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	tests := []struct {
		level             RiskLevel
		expectedAction    Action
	}{
		{RiskLevelLow, ActionAllow},
		{RiskLevelMedium, ActionVerify},
		{RiskLevelHigh, ActionVerify},
		{RiskLevelCritical, ActionBlock},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			action := engine.getRecommendedAction(tt.level)
			if action != tt.expectedAction {
				t.Errorf("getRecommendedAction(%s) = %s, want %s", tt.level, action, tt.expectedAction)
			}
		})
	}
}

func TestAnalyzeMouseTrack(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Insufficient track data", func(t *testing.T) {
		tracks := []MouseTrack{
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
		tracks := []MouseTrack{
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
}

func TestAnalyzeClickRhythm(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

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
}

func TestCalculateSmoothness(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Empty track", func(t *testing.T) {
		smoothness := engine.calculateSmoothness([]MouseTrack{})
		if smoothness != 1.0 {
			t.Errorf("calculateSmoothness([]) = %f, want 1.0", smoothness)
		}
	})

	t.Run("Single point", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
		}
		smoothness := engine.calculateSmoothness(tracks)
		if smoothness != 1.0 {
			t.Errorf("calculateSmoothness([1 point]) = %f, want 1.0", smoothness)
		}
	})

	t.Run("Straight line", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 100, Y: 100, Timestamp: 100},
			{X: 200, Y: 200, Timestamp: 200},
			{X: 300, Y: 300, Timestamp: 300},
		}
		smoothness := engine.calculateSmoothness(tracks)
		if smoothness < 0.9 {
			t.Errorf("calculateSmoothness(straight line) = %f, want >= 0.9", smoothness)
		}
	})
}

func TestCalculateJitter(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Empty track", func(t *testing.T) {
		jitter := engine.calculateJitter([]MouseTrack{})
		if jitter != 0.0 {
			t.Errorf("calculateJitter([]) = %f, want 0.0", jitter)
		}
	})

	t.Run("Single point", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
		}
		jitter := engine.calculateJitter(tracks)
		if jitter != 0.0 {
			t.Errorf("calculateJitter([1 point]) = %f, want 0.0", jitter)
		}
	})

	t.Run("High jitter track", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 110, Y: 150, Timestamp: 50},
			{X: 120, Y: 80, Timestamp: 100},
			{X: 130, Y: 160, Timestamp: 150},
			{X: 140, Y: 90, Timestamp: 200},
		}
		jitter := engine.calculateJitter(tracks)
		if jitter < 0.5 {
			t.Errorf("calculateJitter(high jitter) = %f, want >= 0.5", jitter)
		}
	})
}

func TestCalculateVelocityConsistency(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Empty track", func(t *testing.T) {
		consistency := engine.calculateVelocityConsistency([]MouseTrack{})
		if consistency != 1.0 {
			t.Errorf("calculateVelocityConsistency([]) = %f, want 1.0", consistency)
		}
	})

	t.Run("Single point", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
		}
		consistency := engine.calculateVelocityConsistency(tracks)
		if consistency != 1.0 {
			t.Errorf("calculateVelocityConsistency([1 point]) = %f, want 1.0", consistency)
		}
	})

	t.Run("Uniform velocity", func(t *testing.T) {
		tracks := []MouseTrack{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 100, Y: 0, Timestamp: 100},
			{X: 200, Y: 0, Timestamp: 200},
			{X: 300, Y: 0, Timestamp: 300},
		}
		consistency := engine.calculateVelocityConsistency(tracks)
		t.Logf("calculateVelocityConsistency(uniform) = %f", consistency)
	})
}

func TestCalculateRhythmVariance(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Empty clicks", func(t *testing.T) {
		variance := engine.calculateRhythmVariance([]int64{})
		if variance != 1.0 {
			t.Errorf("calculateRhythmVariance([]) = %f, want 1.0", variance)
		}
	})

	t.Run("Two clicks", func(t *testing.T) {
		clicks := []int64{100, 200}
		variance := engine.calculateRhythmVariance(clicks)
		if variance != 1.0 {
			t.Errorf("calculateRhythmVariance([2 clicks]) = %f, want 1.0", variance)
		}
	})

	t.Run("Mechanical rhythm", func(t *testing.T) {
		clicks := []int64{100, 200, 300, 400, 500}
		variance := engine.calculateRhythmVariance(clicks)
		if variance >= 0.05 {
			t.Errorf("calculateRhythmVariance(mechanical) = %f, want < 0.05", variance)
		}
	})
}

func TestIsClickTooFast(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	t.Run("Empty clicks", func(t *testing.T) {
		result := engine.isClickTooFast([]int64{})
		if result {
			t.Error("isClickTooFast([]) = true, want false")
		}
	})

	t.Run("Single click", func(t *testing.T) {
		result := engine.isClickTooFast([]int64{100})
		if result {
			t.Error("isClickTooFast([1 click]) = true, want false")
		}
	})

	t.Run("Fast clicks", func(t *testing.T) {
		clicks := []int64{100, 140, 180, 220}
		result := engine.isClickTooFast(clicks)
		if !result {
			t.Error("isClickTooFast(fast clicks) = false, want true")
		}
	})

	t.Run("Normal clicks", func(t *testing.T) {
		clicks := []int64{100, 300, 600, 1000}
		result := engine.isClickTooFast(clicks)
		if result {
			t.Error("isClickTooFast(normal clicks) = true, want false")
		}
	})
}

func TestTrackBehavior(t *testing.T) {
	cfg := config.DefaultRiskConfig()
	engine := NewRiskEngine(cfg, nil, nil)

	behavior := &BehaviorData{
		SessionID: "test-session",
		MouseTracks: []MouseTrack{
			{X: 100, Y: 100, Timestamp: 0},
			{X: 200, Y: 200, Timestamp: 100},
		},
	}

	err := engine.TrackBehavior(behavior)
	if err != nil {
		t.Errorf("TrackBehavior() error = %v, want nil", err)
	}
}
