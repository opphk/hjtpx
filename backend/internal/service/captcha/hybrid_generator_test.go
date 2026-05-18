package captcha

import (
	"context"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
)

func TestHybridGeneratorService_Create(t *testing.T) {
	gen := NewHybridGeneratorService(nil, nil)
	
	req := &CreateHybridRequest{
		Width:        320,
		Height:       160,
		SliderWidth:  40,
		SliderHeight: 40,
		ClickCount:   3,
		ClientIP:     "127.0.0.1",
		UserAgent:    "test",
		Fingerprint:  "test-fingerprint",
	}
	
	ctx := context.Background()
	result, err := gen.Create(ctx, req)
	
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	if result.SessionID == "" {
		t.Error("Expected non-empty session ID")
	}
	
	if result.Phase != HybridPhaseSlider {
		t.Errorf("Expected phase %s, got %s", HybridPhaseSlider, result.Phase)
	}
	
	if result.BackgroundURL == "" {
		t.Error("Expected non-empty background URL")
	}
	
	if result.SliderURL == "" {
		t.Error("Expected non-empty slider URL")
	}
	
	if result.SliderGapX <= 0 {
		t.Error("Expected positive gap X")
	}
	
	if result.ExpiresIn <= 0 {
		t.Error("Expected positive expires in")
	}
	
	t.Logf("Hybrid captcha created: session=%s, phase=%s, gapX=%d", 
		result.SessionID, result.Phase, result.SliderGapX)
}

func TestHybridGeneratorService_GenerateClickTargets(t *testing.T) {
	gen := NewHybridGeneratorService(nil, nil)
	
	tests := []struct {
		width      int
		height     int
		count      int
		expectMin  int
	}{
		{200, 100, 3, 2},
		{320, 160, 5, 4},
		{400, 200, 9, 8},
	}
	
	for _, tt := range tests {
		targets := gen.generateClickTargets(tt.width, tt.height, tt.count)
		
		if len(targets) < tt.expectMin {
			t.Errorf("width=%d, height=%d, count=%d: expected at least %d targets, got %d",
				tt.width, tt.height, tt.count, tt.expectMin, len(targets))
		}
		
		for i, target := range targets {
			if target.X < 0 || target.X >= tt.width {
				t.Errorf("target[%d].X out of bounds: %d", i, target.X)
			}
			if target.Y < 0 || target.Y >= tt.height {
				t.Errorf("target[%d].Y out of bounds: %d", i, target.Y)
			}
			if target.Width <= 0 || target.Height <= 0 {
				t.Errorf("target[%d] has invalid size: %dx%d", i, target.Width, target.Height)
			}
		}
	}
}

func TestHybridVerifierService_VerifySlider(t *testing.T) {
	gen := NewHybridGeneratorService(nil, nil)
	ver := NewHybridVerifierService(nil, nil)
	
	ctx := context.Background()
	
	createResult, err := gen.Create(ctx, &CreateHybridRequest{
		Width:       320,
		Height:      160,
		ClickCount:  3,
		ClientIP:    "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Failed to create captcha: %v", err)
	}
	
	sliderReq := &VerifyHybridSliderRequest{
		SessionID:  createResult.SessionID,
		PositionX:  createResult.SliderGapX,
		PositionY:  createResult.SliderGapY,
		Trajectory: []TrajectoryData{
			{X: 0, Y: 0, Timestamp: time.Now().UnixMilli()},
			{X: 50, Y: 5, Timestamp: time.Now().Add(100*time.Millisecond).UnixMilli()},
			{X: createResult.SliderGapX, Y: 0, Timestamp: time.Now().Add(500*time.Millisecond).UnixMilli()},
		},
		RiskScore: 0.5,
	}
	
	result, err := ver.VerifySlider(ctx, sliderReq)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	t.Logf("Slider verification result: success=%v, phase=%s, message=%s", 
		result.Success, result.Phase, result.Message)
}

func TestHybridVerifierService_VerifySliderInvalidPosition(t *testing.T) {
	gen := NewHybridGeneratorService(nil, nil)
	ver := NewHybridVerifierService(nil, nil)
	
	ctx := context.Background()
	
	createResult, err := gen.Create(ctx, &CreateHybridRequest{
		Width:      320,
		Height:     160,
		ClickCount: 3,
		ClientIP:   "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Failed to create captcha: %v", err)
	}
	
	sliderReq := &VerifyHybridSliderRequest{
		SessionID: createResult.SessionID,
		PositionX: 0,
		PositionY: 0,
		Trajectory: []TrajectoryData{
			{X: 0, Y: 0, Timestamp: time.Now().UnixMilli()},
		},
		RiskScore: 0.5,
	}
	
	result, err := ver.VerifySlider(ctx, sliderReq)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if result.Success {
		t.Error("Expected verification to fail for invalid position")
	}
	
	if result.Phase != HybridPhaseSlider {
		t.Errorf("Expected phase to remain %s, got %s", HybridPhaseSlider, result.Phase)
	}
}

func TestHybridVerifierService_AnalyzeSliderTrajectory(t *testing.T) {
	ver := NewHybridVerifierService(nil, nil)
	
	tests := []struct {
		name      string
		trajectory []TrajectoryData
		minScore  float64
	}{
		{
			name: "normal human trajectory",
			trajectory: []TrajectoryData{
				{X: 0, Y: 50, Timestamp: 0},
				{X: 20, Y: 52, Timestamp: 100},
				{X: 45, Y: 48, Timestamp: 200},
				{X: 70, Y: 51, Timestamp: 300},
				{X: 100, Y: 50, Timestamp: 400},
			},
			minScore: 0.5,
		},
		{
			name: "too short trajectory",
			trajectory: []TrajectoryData{
				{X: 0, Y: 0, Timestamp: 0},
			},
			minScore: 0.4,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := ver.analyzeSliderTrajectory(tt.trajectory, 100)
			if score < tt.minScore {
				t.Errorf("Expected score >= %f, got %f", tt.minScore, score)
			}
		})
	}
}

func TestHybridVerifierService_GetSessionStatus(t *testing.T) {
	ver := NewHybridVerifierService(nil, nil)
	
	valid, message := ver.CheckSessionValid(context.Background(), "non-existent-session")
	if valid {
		t.Error("Expected invalid for non-existent session")
	}
	if message == "" {
		t.Error("Expected non-empty error message")
	}
}
