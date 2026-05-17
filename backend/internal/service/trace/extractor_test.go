package trace

import (
	"encoding/json"
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestExtractFeatures(t *testing.T) {
	extractor := NewTraceExtractor()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "move"},
			{Timestamp: 1300, X: 30, Y: 30, Event: "move"},
			{Timestamp: 1400, X: 40, Y: 40, Event: "end"},
		},
		TotalTime: 400,
		StartX:    0,
		StartY:    0,
		EndX:      40,
		EndY:      40,
	}

	features, err := extractor.ExtractFeatures(traceData)
	if err != nil {
		t.Fatalf("ExtractFeatures failed: %v", err)
	}

	if features == nil {
		t.Fatal("Features should not be nil")
	}

	if features.TotalTime != 400 {
		t.Errorf("Expected TotalTime 400, got %d", features.TotalTime)
	}

	if features.MoveCount != 3 {
		t.Errorf("Expected MoveCount 3, got %d", features.MoveCount)
	}

	t.Logf("Features: %+v", features)
}

func TestCalculateScore(t *testing.T) {
	matcher := NewTraceMatcher()

	features := &model.TraceFeatures{
		TotalTime:       3000,
		MoveCount:       50,
		AvgSpeed:        150,
		MaxSpeed:        300,
		MinSpeed:        50,
		SpeedVariance:   100,
		MaxAcceleration: 500,
		Smoothness:      0.5,
		PauseCount:      2,
		TotalDistance:   450,
		DirectDistance:  424,
		PathRatio:       1.06,
		RiskFactors:     []string{},
	}

	score := matcher.CalculateScore(features)

	if score == nil {
		t.Fatal("Score should not be nil")
	}

	if score.TotalScore <= 0 || score.TotalScore > 100 {
		t.Errorf("TotalScore should be between 0 and 100, got %f", score.TotalScore)
	}

	if score.SpeedScore <= 0 || score.SpeedScore > 100 {
		t.Errorf("SpeedScore should be between 0 and 100, got %f", score.SpeedScore)
	}

	t.Logf("Score: %+v", score)
}

func TestIsBot(t *testing.T) {
	matcher := NewTraceMatcher()

	tests := []struct {
		name     string
		score    *model.TraceScore
		expected bool
	}{
		{
			name: "Low score with multiple risk factors - bot",
			score: &model.TraceScore{
				TotalScore:  25,
				SpeedScore:  20,
				SmoothScore: 20,
				RiskFactors: []string{"速度方差过小", "平均速度过快", "无正常停顿行为"},
			},
			expected: true,
		},
		{
			name: "High score with no risk factors - human",
			score: &model.TraceScore{
				TotalScore:  85,
				SpeedScore:  75,
				SmoothScore: 80,
				RiskFactors: []string{},
			},
			expected: false,
		},
		{
			name: "Low scores on speed and smoothness - bot",
			score: &model.TraceScore{
				TotalScore:  50,
				SpeedScore:  25,
				SmoothScore: 25,
				RiskFactors: []string{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.IsBot(tt.score)
			if result != tt.expected {
				t.Errorf("IsBot() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestExtractFeaturesInsufficientPoints(t *testing.T) {
	extractor := NewTraceExtractor()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
		},
		TotalTime: 0,
	}

	_, err := extractor.ExtractFeatures(traceData)
	if err == nil {
		t.Error("Expected error for insufficient points, got nil")
	}
}

func TestTraceServiceProcessTrace(t *testing.T) {
	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 50, Y: 50, Event: "move"},
			{Timestamp: 1200, X: 100, Y: 100, Event: "move"},
			{Timestamp: 1300, X: 150, Y: 150, Event: "end"},
		},
		TotalTime: 300,
	}

	traceDataJSON, _ := json.Marshal(traceData)

	matcher := NewTraceMatcher()

	var td model.TraceData
	json.Unmarshal(traceDataJSON, &td)

	features, score, err := matcher.ExtractAndScore(&td)
	if err != nil {
		t.Fatalf("ExtractAndScore failed: %v", err)
	}

	if features == nil {
		t.Fatal("Features should not be nil")
	}

	if score == nil {
		t.Fatal("Score should not be nil")
	}

	if features.TotalDistance <= 0 {
		t.Error("TotalDistance should be positive")
	}

	if features.DirectDistance <= 0 {
		t.Error("DirectDistance should be positive")
	}

	t.Logf("Features: %+v", features)
	t.Logf("Score: %+v", score)
}

func TestRiskFactorsDetection(t *testing.T) {
	extractor := NewTraceExtractor()

	tests := []struct {
		name            string
		features        *model.TraceFeatures
		expectRiskCount int
	}{
		{
			name: "Too uniform speed",
			features: &model.TraceFeatures{
				TotalTime:     2000,
				AvgSpeed:      100,
				SpeedVariance: 5,
			},
			expectRiskCount: 1,
		},
		{
			name: "Too fast average speed",
			features: &model.TraceFeatures{
				TotalTime: 1000,
				AvgSpeed:  1500,
			},
			expectRiskCount: 1,
		},
		{
			name: "Too straight path",
			features: &model.TraceFeatures{
				TotalTime:     1000,
				AvgSpeed:      100,
				PathRatio:     1.05,
				SpeedVariance: 100,
			},
			expectRiskCount: 1,
		},
		{
			name: "No pause in long trace",
			features: &model.TraceFeatures{
				TotalTime:     3000,
				PauseCount:    0,
				AvgSpeed:      100,
				SpeedVariance: 100,
			},
			expectRiskCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			riskFactors := extractor.detectRiskFactors(tt.features)
			if len(riskFactors) < tt.expectRiskCount {
				t.Errorf("Expected at least %d risk factors, got %d: %v",
					tt.expectRiskCount, len(riskFactors), riskFactors)
			}
		})
	}
}
