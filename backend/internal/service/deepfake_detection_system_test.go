package service

import (
	"context"
	"testing"
)

func TestDeepfakeDetectionSystem(t *testing.T) {
	system := NewDeepfakeDetectionSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	if !system.initialized {
		t.Error("System should be initialized")
	}
}

func TestFaceSwapDetector(t *testing.T) {
	detector := NewFaceSwapDetector()
	ctx := context.Background()

	if err := detector.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize detector: %v", err)
	}

	result, err := detector.DetectFaceSwap(ctx, []byte("test image data"), nil)
	if err != nil {
		t.Fatalf("Failed to detect face swap: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Confidence < 0 || result.Confidence > 100 {
		t.Errorf("Confidence should be between 0 and 100, got %f", result.Confidence)
	}
}

func TestVoiceSynthesisDetector(t *testing.T) {
	detector := NewVoiceSynthesisDetector()
	ctx := context.Background()

	if err := detector.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize detector: %v", err)
	}

	result, err := detector.DetectVoiceSynthesis(ctx, []byte("test audio data"), nil)
	if err != nil {
		t.Fatalf("Failed to detect voice synthesis: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Confidence < 0 || result.Confidence > 100 {
		t.Errorf("Confidence should be between 0 and 100, got %f", result.Confidence)
	}
}

func TestImageTamperingDetector(t *testing.T) {
	detector := NewImageTamperingDetector()
	ctx := context.Background()

	if err := detector.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize detector: %v", err)
	}

	result, err := detector.DetectTampering(ctx, []byte("test image data"), nil)
	if err != nil {
		t.Fatalf("Failed to detect tampering: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Confidence < 0 || result.Confidence > 100 {
		t.Errorf("Confidence should be between 0 and 100, got %f", result.Confidence)
	}
}

func TestDeepfakeAlertSystem(t *testing.T) {
	alertSystem := NewDeepfakeAlertSystem()
	ctx := context.Background()

	alert, err := alertSystem.CreateAlert(ctx, "test_type", "high", "test_source", "test message", nil)
	if err != nil {
		t.Fatalf("Failed to create alert: %v", err)
	}

	if alert == nil {
		t.Fatal("Alert should not be nil")
	}

	if alert.ID == "" {
		t.Error("Alert ID should be set")
	}

	alerts, err := alertSystem.GetAlerts(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to get alerts: %v", err)
	}

	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}
}

func TestComprehensiveDeepfakeDetection(t *testing.T) {
	system := NewDeepfakeDetectionSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	result, err := system.ComprehensiveDetection(ctx, "image", []byte("test data"), nil)
	if err != nil {
		t.Fatalf("Failed to perform comprehensive detection: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.RiskLevel == "" {
		t.Error("Risk level should be set")
	}

	if len(result.Recommendations) == 0 {
		t.Error("Should have at least one recommendation")
	}
}
