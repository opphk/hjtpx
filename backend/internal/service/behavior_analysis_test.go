package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestNewBehaviorAnalysisService(t *testing.T) {
	service := NewBehaviorAnalysisService()
	if service == nil {
		t.Error("NewBehaviorAnalysisService should return a non-nil service")
	}
}

func TestAnalyzeBehavior(t *testing.T) {
	service := NewBehaviorAnalysisService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 100, X: 150, Y: 120, Event: "move"},
			{Timestamp: 200, X: 200, Y: 150, Event: "move"},
			{Timestamp: 300, X: 250, Y: 180, Event: "move"},
		},
		TotalTime: 300,
	}

	result, err := service.AnalyzeBehavior(context.Background(), traceData)
	if err != nil {
		t.Fatalf("AnalyzeBehavior failed: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	if result.Score < 0 || result.Score > 100 {
		t.Errorf("Expected score in [0, 100], got %v", result.Score)
	}
}

func TestAnalyzeBehavior_EmptyTrace(t *testing.T) {
	service := NewBehaviorAnalysisService()

	traceData := &model.TraceData{
		Points:    []model.TracePoint{},
		TotalTime: 0,
	}

	result, err := service.AnalyzeBehavior(context.Background(), traceData)
	if err != nil {
		t.Fatalf("AnalyzeBehavior should not fail on empty trace: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result for empty trace")
	}
}

func TestAnalyzeBehavior_MinimalTrace(t *testing.T) {
	service := NewBehaviorAnalysisService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 100, X: 110, Y: 110, Event: "move"},
		},
		TotalTime: 100,
	}

	result, err := service.AnalyzeBehavior(context.Background(), traceData)
	if err != nil {
		t.Fatalf("AnalyzeBehavior failed on minimal trace: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result for minimal trace")
	}
}

func TestBehaviorFeatures(t *testing.T) {
	service := NewBehaviorAnalysisService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 50, X: 150, Y: 120, Event: "move"},
			{Timestamp: 100, X: 200, Y: 150, Event: "click"},
		},
		TotalTime: 100,
	}

	features, err := service.ExtractFeatures(context.Background(), traceData)
	if err != nil {
		t.Fatalf("ExtractFeatures failed: %v", err)
	}

	if features == nil {
		t.Error("Expected non-nil features")
	}

	if len(features) == 0 {
		t.Error("Expected non-empty features")
	}
}

func TestMouseVelocityAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 0, Y: 0, Event: "move"},
			{Timestamp: 100, X: 100, Y: 100, Event: "move"},
			{Timestamp: 200, X: 200, Y: 200, Event: "move"},
		},
		TotalTime: 200,
	}

	result, err := service.AnalyzeBehavior(context.Background(), traceData)
	if err != nil {
		t.Fatalf("AnalyzeBehavior failed: %v", err)
	}

	if result.FeatureVector != nil {
		if len(result.FeatureVector) > 0 {
			for i, f := range result.FeatureVector {
				if math.IsNaN(f) || math.IsInf(f, 0) {
					t.Errorf("Feature %d has invalid value: %v", i, f)
				}
			}
		}
	}
}

func TestClickPatternAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 50, X: 150, Y: 150, Event: "move"},
			{Timestamp: 100, X: 200, Y: 200, Event: "click"},
			{Timestamp: 150, X: 250, Y: 250, Event: "move"},
			{Timestamp: 200, X: 300, Y: 300, Event: "click"},
		},
		TotalTime: 200,
	}

	result, err := service.AnalyzeBehavior(context.Background(), traceData)
	if err != nil {
		t.Fatalf("AnalyzeBehavior failed: %v", err)
	}

	if result.ClickCount < 0 {
		t.Errorf("Expected non-negative click count, got %v", result.ClickCount)
	}
}

func TestTrajectoryPatternAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := make([]model.TracePoint, 20)
	for i := 0; i < 20; i++ {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 50),
			X:         float64(i * 10),
			Y:         float64(i * 10),
			Event:     "move",
		}
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 950,
	}

	result, err := service.AnalyzeBehavior(context.Background(), traceData)
	if err != nil {
		t.Fatalf("AnalyzeBehavior failed: %v", err)
	}

	if result.TrajectoryPattern == "" {
		t.Log("Trajectory pattern is empty - this may be expected for short traces")
	}
}

func TestKeyboardPatternAnalysis(t *testing.T) {
	service := NewBehaviorAnalysisService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 0, Y: 0, Event: "keydown"},
			{Timestamp: 100, X: 0, Y: 0, Event: "keyup"},
			{Timestamp: 200, X: 0, Y: 0, Event: "keydown"},
			{Timestamp: 300, X: 0, Y: 0, Event: "keyup"},
		},
		TotalTime: 300,
	}

	result, err := service.AnalyzeBehavior(context.Background(), traceData)
	if err != nil {
		t.Fatalf("AnalyzeBehavior failed: %v", err)
	}

	if result.KeyboardMetrics != nil {
		if result.KeyboardMetrics.TotalKeystrokes < 0 {
			t.Errorf("Expected non-negative keystroke count")
		}
	}
}

func TestBehaviorRiskLevel(t *testing.T) {
	service := NewBehaviorAnalysisService()

	testCases := []struct {
		name  string
		trace *model.TraceData
	}{
		{
			name: "Normal human behavior",
			trace: &model.TraceData{
				Points: []model.TracePoint{
					{Timestamp: 0, X: 100, Y: 100, Event: "move"},
					{Timestamp: 50, X: 110, Y: 105, Event: "move"},
					{Timestamp: 100, X: 120, Y: 110, Event: "move"},
					{Timestamp: 150, X: 200, Y: 200, Event: "click"},
				},
				TotalTime: 150,
			},
		},
		{
			name: "Suspicious rapid movement",
			trace: &model.TraceData{
				Points: []model.TracePoint{
					{Timestamp: 0, X: 0, Y: 0, Event: "move"},
					{Timestamp: 1, X: 1000, Y: 1000, Event: "move"},
					{Timestamp: 2, X: 2000, Y: 2000, Event: "move"},
				},
				TotalTime: 2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.AnalyzeBehavior(context.Background(), tc.trace)
			if err != nil {
				t.Fatalf("AnalyzeBehavior failed: %v", err)
			}

			if result.RiskLevel != "" {
				validLevels := map[string]bool{
					"low":      true,
					"medium":   true,
					"high":     true,
					"critical": true,
				}
				if !validLevels[result.RiskLevel] {
					t.Errorf("Invalid risk level: %s", result.RiskLevel)
				}
			}
		})
	}
}

func TestBehaviorAnalysisMetrics(t *testing.T) {
	service := NewBehaviorAnalysisService()

	metrics := service.GetMetrics()
	if metrics == nil {
		t.Error("Expected non-nil metrics")
	}
}

func TestBehaviorAnalysisTimeout(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := make([]model.TracePoint, 100)
	for i := 0; i < 100; i++ {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 10),
			X:         float64(i * 5),
			Y:         float64(i * 5),
			Event:     "move",
		}
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 990,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, err := service.AnalyzeBehavior(ctx, traceData)
	if err != nil && ctx.Err() == context.DeadlineExceeded {
		t.Log("Context timeout handled correctly")
	}
}

func TestBehaviorFeaturesNormalization(t *testing.T) {
	service := NewBehaviorAnalysisService()

	points := make([]model.TracePoint, 50)
	for i := 0; i < 50; i++ {
		points[i] = model.TracePoint{
			Timestamp: int64(i * 100),
			X:         float64(i * 100),
			Y:         float64(i * 100),
			Event:     "move",
		}
	}

	traceData := &model.TraceData{
		Points:    points,
		TotalTime: 4900,
	}

	features, err := service.ExtractFeatures(context.Background(), traceData)
	if err != nil {
		t.Fatalf("ExtractFeatures failed: %v", err)
	}

	for i, f := range features {
		if math.Abs(f) > 1000 {
			t.Logf("Feature %d has large absolute value: %f (may indicate need for normalization)", i, f)
		}
	}
}

func TestBehaviorAnalysisConcurrency(t *testing.T) {
	service := NewBehaviorAnalysisService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 100, X: 150, Y: 120, Event: "move"},
		},
		TotalTime: 100,
	}

	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func() {
			_, _ = service.AnalyzeBehavior(context.Background(), traceData)
			done <- true
		}()
	}

	timeout := time.After(5 * time.Second)
	for i := 0; i < 5; i++ {
		select {
		case <-done:
		case <-timeout:
			t.Error("Timeout waiting for concurrent analysis")
			return
		}
	}
}

func TestBehaviorAnalysisWithDeviceInfo(t *testing.T) {
	service := NewBehaviorAnalysisService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100, Event: "move"},
			{Timestamp: 100, X: 150, Y: 120, Event: "move"},
		},
		TotalTime: 100,
		DeviceInfo: &model.DeviceInfo{
			UserAgent:   "Mozilla/5.0",
			ScreenWidth: 1920,
			ScreenHeight: 1080,
			Platform:    "Win32",
		},
	}

	result, err := service.AnalyzeBehavior(context.Background(), traceData)
	if err != nil {
		t.Fatalf("AnalyzeBehavior failed with device info: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result with device info")
	}
}
