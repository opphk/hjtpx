package service

import (
	"testing"
)

func TestClickAnalyzer_EnhancedTimingAnalysis(t *testing.T) {
	analyzer := NewClickAnalyzer()
	
	tests := []struct {
		name        string
		clicks      []ClickData
		wantPattern string
	}{
		{
			name: "正常人类点击模式",
			clicks: []ClickData{
				{X: 50, Y: 50, Timestamp: 1000, Index: 0},
				{X: 150, Y: 100, Timestamp: 1500, Index: 1},
				{X: 250, Y: 150, Timestamp: 2200, Index: 2},
			},
			wantPattern: "varied",
		},
		{
			name: "快速机器人点击模式",
			clicks: []ClickData{
				{X: 50, Y: 50, Timestamp: 1000, Index: 0},
				{X: 150, Y: 100, Timestamp: 1005, Index: 1},
				{X: 250, Y: 150, Timestamp: 1010, Index: 2},
			},
			wantPattern: "linear",
		},
		{
			name: "犹豫的人类点击模式",
			clicks: []ClickData{
				{X: 50, Y: 50, Timestamp: 1000, Index: 0},
				{X: 150, Y: 100, Timestamp: 2000, Index: 1},
				{X: 250, Y: 150, Timestamp: 3500, Index: 2},
			},
			wantPattern: "varied",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verification := &ClickVerification{
				Clicks: tt.clicks,
			}
			
			result := analyzer.AnalyzeClickVerification(verification)
			
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			
			if result.ClickPattern == nil {
				t.Fatal("Expected non-nil ClickPattern")
			}
			
			if result.TimingAnalysis == nil {
				t.Fatal("Expected non-nil TimingAnalysis")
			}
			
			if result.TimingAnalysis.TimingPattern != tt.wantPattern {
				t.Errorf("TimingPattern = %v, want %v", 
					result.TimingAnalysis.TimingPattern, tt.wantPattern)
			}
		})
	}
}

func TestClickAnalyzer_EnhancedMultiTargetMatching(t *testing.T) {
	analyzer := NewClickAnalyzer()
	
	tests := []struct {
		name          string
		clicks        []ClickData
		targets       []TargetImage
		wantHitRate   float64
		wantAccuracy  float64
	}{
		{
			name: "正常点击多目标",
			clicks: []ClickData{
				{X: 100, Y: 100, Timestamp: 1000, Index: 0},
				{X: 200, Y: 200, Timestamp: 1500, Index: 1},
				{X: 300, Y: 300, Timestamp: 2000, Index: 2},
			},
			targets: []TargetImage{
				{X: 100, Y: 100, Width: 50, Height: 50},
				{X: 200, Y: 200, Width: 50, Height: 50},
				{X: 300, Y: 300, Width: 50, Height: 50},
			},
			wantHitRate:  1.0,
			wantAccuracy: 1.0,
		},
		{
			name: "误点击多目标",
			clicks: []ClickData{
				{X: 100, Y: 100, Timestamp: 1000, Index: 0},
				{X: 250, Y: 250, Timestamp: 1500, Index: 1},
				{X: 300, Y: 300, Timestamp: 2000, Index: 2},
			},
			targets: []TargetImage{
				{X: 100, Y: 100, Width: 50, Height: 50},
				{X: 200, Y: 200, Width: 50, Height: 50},
				{X: 300, Y: 300, Width: 50, Height: 50},
			},
			wantHitRate:  0.67,
			wantAccuracy: 0.67,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verification := &ClickVerification{
				Clicks:       tt.clicks,
				TargetImages: tt.targets,
			}
			
			result := analyzer.AnalyzeClickVerification(verification)
			
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			
			if result.MultiTargetAnalysis == nil {
				t.Fatal("Expected non-nil MultiTargetAnalysis")
			}
			
			if result.MultiTargetAnalysis.TargetHitRate != tt.wantHitRate {
				t.Errorf("TargetHitRate = %v, want %v", 
					result.MultiTargetAnalysis.TargetHitRate, tt.wantHitRate)
			}
		})
	}
}

func TestClickAnalyzer_EnhancedFaultTolerance(t *testing.T) {
	analyzer := NewClickAnalyzer()
	
	verification := &ClickVerification{
		Clicks: []ClickData{
			{X: 100, Y: 100, Timestamp: 1000, Index: 0},
			{X: 235, Y: 235, Timestamp: 1500, Index: 1},
			{X: 300, Y: 300, Timestamp: 2000, Index: 2},
		},
		TargetImages: []TargetImage{
			{X: 100, Y: 100, Width: 50, Height: 50},
			{X: 200, Y: 200, Width: 50, Height: 50},
			{X: 300, Y: 300, Width: 50, Height: 50},
		},
	}
	
	result := analyzer.AnalyzeClickVerification(verification)
	
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	
	if result.FaultTolerance == nil {
		t.Fatal("Expected non-nil FaultTolerance")
	}
	
	if !result.FaultTolerance.Enabled {
		t.Error("Expected FaultTolerance to be enabled")
	}
}
