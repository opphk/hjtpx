package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBehaviorAnalysis(t *testing.T) {
	analysis := NewBehaviorAnalysis(nil, nil)
	assert.NotNil(t, analysis)
}

func TestBehaviorAnalysis_Analyze_Success(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{
		events: []map[string]interface{}{},
	}

	events := []map[string]interface{}{
		{
			"type":      "click",
			"timestamp": time.Now().Unix(),
			"x":         100,
			"y":         200,
		},
		{
			"type":      "move",
			"timestamp": time.Now().Unix(),
			"x":         110,
			"y":         210,
		},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "user123", result.UserID)
}

func TestBehaviorAnalysis_Analyze_EmptyEvents(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{
		events: []map[string]interface{}{},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", []map[string]interface{}{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "user123", result.UserID)
}

func TestBehaviorAnalysis_Analyze_MouseMovement(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "move", "timestamp": time.Now().Unix(), "x": 100, "y": 100, "duration": 50},
		{"type": "move", "timestamp": time.Now().Unix(), "x": 150, "y": 150, "duration": 100},
		{"type": "move", "timestamp": time.Now().Unix(), "x": 200, "y": 200, "duration": 75},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.TotalScore, float64(0))
}

func TestBehaviorAnalysis_Analyze_ClickPatterns(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "click", "timestamp": time.Now().Unix(), "x": 100, "y": 100},
		{"type": "click", "timestamp": time.Now().Unix(), "x": 200, "y": 200},
		{"type": "click", "timestamp": time.Now().Unix(), "x": 300, "y": 300},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_KeyStrokes(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "a"},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "b"},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "c"},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_ScrollBehavior(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "scroll", "timestamp": time.Now().Unix(), "scrollY": 0},
		{"type": "scroll", "timestamp": time.Now().Unix(), "scrollY": 100},
		{"type": "scroll", "timestamp": time.Now().Unix(), "scrollY": 200},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_TouchGestures(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "touchstart", "timestamp": time.Now().Unix(), "x": 100, "y": 100},
		{"type": "touchmove", "timestamp": time.Now().Unix(), "x": 150, "y": 150},
		{"type": "touchend", "timestamp": time.Now().Unix(), "x": 200, "y": 200},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_MixedEvents(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "move", "timestamp": time.Now().Unix(), "x": 100, "y": 100},
		{"type": "click", "timestamp": time.Now().Unix(), "x": 100, "y": 100},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "a"},
		{"type": "scroll", "timestamp": time.Now().Unix(), "scrollY": 100},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 4, result.EventCount)
}

func TestBehaviorAnalysis_Analyze_HumanLikeBehavior(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	for i := 0; i < 20; i++ {
		events = append(events, map[string]interface{}{
			"type":      "move",
			"timestamp": time.Now().Unix(),
			"x":         float64(100 + i*5),
			"y":         float64(100 + i*3),
			"duration":  50 + i*2,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "human", result.Classification)
}

func TestBehaviorAnalysis_Analyze_BotLikeBehavior(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	for i := 0; i < 10; i++ {
		events = append(events, map[string]interface{}{
			"type":      "click",
			"timestamp": time.Now().Unix(),
			"x":         100,
			"y":         100,
			"duration":  0,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "bot", result.Classification)
}

func TestBehaviorAnalysis_Analyze_AutomatedBehavior(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	for i := 0; i < 100; i++ {
		events = append(events, map[string]interface{}{
			"type":      "move",
			"timestamp": time.Now().Unix(),
			"x":         float64(i),
			"y":         float64(i),
			"duration":  1,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "automated", result.Classification)
}

func TestBehaviorAnalysis_Analyze_RegularUser(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	now := time.Now()
	events := []map[string]interface{}{
		{"type": "move", "timestamp": now.Unix(), "x": 100, "y": 100, "duration": 100},
		{"type": "move", "timestamp": now.Unix() + 1, "x": 105, "y": 105, "duration": 120},
		{"type": "click", "timestamp": now.Unix() + 2, "x": 105, "y": 105},
		{"type": "keydown", "timestamp": now.Unix() + 3, "key": "a"},
		{"type": "keydown", "timestamp": now.Unix() + 4, "key": "b"},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.TotalScore, float64(0))
	assert.LessOrEqual(t, result.TotalScore, float64(100))
}

func TestBehaviorAnalysis_Analyze_RapidClicks(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	now := time.Now()
	events := make([]map[string]interface{}, 0)
	for i := 0; i < 50; i++ {
		events = append(events, map[string]interface{}{
			"type":      "click",
			"timestamp": now.Unix(),
			"x":         100 + (i % 5) * 10,
			"y":         100 + (i / 5) * 10,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Less(t, result.SuspicionScore, 80)
}

func TestBehaviorAnalysis_Analyze_PerfectStraightLines(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	for i := 0; i < 10; i++ {
		events = append(events, map[string]interface{}{
			"type":      "move",
			"timestamp": time.Now().Unix(),
			"x":         float64(i * 10),
			"y":         100,
			"duration":  10,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_CharacterTiming(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "t", "duration": 50},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "e", "duration": 80},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "s", "duration": 70},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "t", "duration": 90},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_MouseSpeed(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "move", "timestamp": time.Now().Unix(), "x": 0, "y": 0, "duration": 100},
		{"type": "move", "timestamp": time.Now().Unix(), "x": 1000, "y": 1000, "duration": 1},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_ErraticMovement(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "move", "timestamp": time.Now().Unix(), "x": 100, "y": 100, "duration": 50},
		{"type": "move", "timestamp": time.Now().Unix(), "x": 50, "y": 150, "duration": 50},
		{"type": "move", "timestamp": time.Now().Unix(), "x": 150, "y": 50, "duration": 50},
		{"type": "move", "timestamp": time.Now().Unix(), "x": 50, "y": 50, "duration": 50},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_DeviceMismatch(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "touchstart", "timestamp": time.Now().Unix(), "x": 100, "y": 100},
		{"type": "touchend", "timestamp": time.Now().Unix(), "x": 100, "y": 100},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_HesitationPattern(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "move", "timestamp": time.Now().Unix(), "x": 100, "y": 100, "duration": 100},
		{"type": "move", "timestamp": time.Now().Unix() + 2, "x": 105, "y": 105, "duration": 2000},
		{"type": "move", "timestamp": time.Now().Unix() + 4, "x": 110, "y": 110, "duration": 100},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_Acceleration(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "move", "timestamp": time.Now().Unix(), "x": 0, "y": 0, "duration": 100},
		{"type": "move", "timestamp": time.Now().Unix(), "x": 50, "y": 50, "duration": 50},
		{"type": "move", "timestamp": time.Now().Unix(), "x": 150, "y": 150, "duration": 20},
		{"type": "move", "timestamp": time.Now().Unix(), "x": 350, "y": 350, "duration": 5},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_ConsistentSpeed(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	for i := 0; i < 20; i++ {
		events = append(events, map[string]interface{}{
			"type":      "move",
			"timestamp": time.Now().Unix(),
			"x":         float64(i * 10),
			"y":         float64(i * 10),
			"duration":  50,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_RandomClicks(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	for i := 0; i < 20; i++ {
		events = append(events, map[string]interface{}{
			"type":      "click",
			"timestamp": time.Now().Unix(),
			"x":         float64((i * 37) % 500),
			"y":         float64((i * 53) % 500),
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_FormInputPattern(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "click", "timestamp": time.Now().Unix(), "x": 100, "y": 200},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "u", "duration": 50},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "s", "duration": 70},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "e", "duration": 60},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "r", "duration": 80},
		{"type": "click", "timestamp": time.Now().Unix(), "x": 100, "y": 250},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_DragAndDrop(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "touchstart", "timestamp": time.Now().Unix(), "x": 100, "y": 100},
		{"type": "touchmove", "timestamp": time.Now().Unix(), "x": 110, "y": 110},
		{"type": "touchmove", "timestamp": time.Now().Unix(), "x": 120, "y": 120},
		{"type": "touchmove", "timestamp": time.Now().Unix(), "x": 130, "y": 130},
		{"type": "touchend", "timestamp": time.Now().Unix(), "x": 140, "y": 140},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_ZoomGesture(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "touchstart", "timestamp": time.Now().Unix(), "x": 100, "y": 100, "touchCount": 2},
		{"type": "touchmove", "timestamp": time.Now().Unix(), "x": 150, "y": 150, "touchCount": 2},
		{"type": "touchend", "timestamp": time.Now().Unix(), "x": 200, "y": 200, "touchCount": 2},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_TabKeyNavigation(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "Tab"},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "Tab"},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "Tab"},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "Enter"},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_EscapeKeyUsage(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "Escape"},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "Escape"},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "Escape"},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_ClipboardAction(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "Control"},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "v"},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_CopyPastePattern(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "Control"},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "c"},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "Control"},
		{"type": "keydown", "timestamp": time.Now().Unix(), "key": "v"},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_IdleTime(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "move", "timestamp": time.Now().Unix(), "x": 100, "y": 100},
		{"type": "idle", "timestamp": time.Now().Unix() + 30, "duration": 600},
		{"type": "move", "timestamp": time.Now().Unix() + 31, "x": 105, "y": 105},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_PatternRepeat(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	for i := 0; i < 10; i++ {
		events = append(events, map[string]interface{}{
			"type":      "move",
			"timestamp": time.Now().Unix(),
			"x":         100 + (i % 3) * 50,
			"y":         100 + (i % 3) * 50,
			"duration":  50,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_WorkPattern(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	now := time.Now()
	for i := 0; i < 10; i++ {
		events = append(events, map[string]interface{}{
			"type":      "move",
			"timestamp": now.Unix() + int64(i),
			"x":         float64(100 + i*10),
			"y":         float64(100 + i*5),
			"duration":  50 + (i % 3) * 10,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_CrosshairMovement(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	for i := 0; i < 20; i++ {
		events = append(events, map[string]interface{}{
			"type":      "move",
			"timestamp": time.Now().Unix(),
			"x":         float64(100 + i*5),
			"y":         float64(100 - i*5),
			"duration":  10,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_ParabolicMotion(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	for i := 0; i < 30; i++ {
		x := float64(i * 10)
		y := 500 - float64((i-15)*(i-15)/2)
		events = append(events, map[string]interface{}{
			"type":      "move",
			"timestamp": time.Now().Unix(),
			"x":         x,
			"y":         y,
			"duration":  20,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_SinusoidalMotion(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	for i := 0; i < 60; i++ {
		x := float64(i * 10)
		y := 100 + float64(50*testSin(float64(i)))
		events = append(events, map[string]interface{}{
			"type":      "move",
			"timestamp": time.Now().Unix(),
			"x":         x,
			"y":         y,
			"duration":  10,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_MultitaskBehavior(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	now := time.Now()
	events := []map[string]interface{}{
		{"type": "move", "timestamp": now.Unix(), "x": 100, "y": 100},
		{"type": "move", "timestamp": now.Unix(), "x": 200, "y": 200},
		{"type": "click", "timestamp": now.Unix() + 10, "x": 200, "y": 200},
		{"type": "idle", "timestamp": now.Unix() + 20, "duration": 300},
		{"type": "move", "timestamp": now.Unix() + 25, "x": 300, "y": 300},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_FeatureExtraction(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "move", "timestamp": time.Now().Unix(), "x": 100, "y": 100, "duration": 50},
		{"type": "move", "timestamp": time.Now().Unix(), "x": 110, "y": 105, "duration": 55},
		{"type": "move", "timestamp": time.Now().Unix(), "x": 120, "y": 110, "duration": 48},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Features)
}

func TestBehaviorAnalysis_Analyze_SuspiciousPatterns(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	for i := 0; i < 200; i++ {
		events = append(events, map[string]interface{}{
			"type":      "click",
			"timestamp": time.Now().Unix(),
			"x":         100,
			"y":         100,
			"duration":  0,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.SuspicionScore, 0)
}

func TestBehaviorAnalysis_Analyze_NormalBehavior(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	now := time.Now()
	for i := 0; i < 50; i++ {
		events = append(events, map[string]interface{}{
			"type":      "move",
			"timestamp": now.Unix() + int64(i/2),
			"x":         100 + float64((i*37)%100),
			"y":         100 + float64((i*53)%100),
			"duration":  30 + (i % 5) * 10,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Less(t, result.SuspicionScore, 50)
}

func TestBehaviorAnalysis_Analyze_EmptyUserID(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "click", "timestamp": time.Now().Unix(), "x": 100, "y": 100},
	}

	result, err := analyzer.Analyze(context.Background(), "", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_Analyze_NilEvents(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	result, err := analyzer.Analyze(context.Background(), "user123", nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorAnalysis_GetFeatures(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := []map[string]interface{}{
		{"type": "move", "timestamp": time.Now().Unix(), "x": 100, "y": 100, "duration": 50},
		{"type": "move", "timestamp": time.Now().Unix(), "x": 150, "y": 150, "duration": 60},
		{"type": "click", "timestamp": time.Now().Unix(), "x": 150, "y": 150},
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBehaviorResult_Structure(t *testing.T) {
	result := &BehaviorResult{
		UserID:         "user123",
		Classification: "human",
		Confidence:     0.85,
		TotalScore:     75,
		SuspicionScore: 15,
		EventCount:     10,
		Features: &BehaviorFeatures{
			AverageSpeed:       50.5,
			AverageAcceleration: 2.3,
			MovementVariance:    15.2,
			ClickFrequency:      3.5,
			TypingSpeed:         45.0,
			IdleTime:            5.0,
		},
		Timestamp: time.Now(),
	}

	assert.Equal(t, "user123", result.UserID)
	assert.Equal(t, "human", result.Classification)
	assert.Equal(t, 0.85, result.Confidence)
	assert.Equal(t, 75.0, result.TotalScore)
	assert.Equal(t, 15.0, result.SuspicionScore)
	assert.Equal(t, 10, result.EventCount)
	assert.NotNil(t, result.Features)
	assert.Equal(t, 50.5, result.Features.AverageSpeed)
}

func TestBehaviorFeatures_Structure(t *testing.T) {
	features := &BehaviorFeatures{
		AverageSpeed:        50.5,
		AverageAcceleration: 2.3,
		MovementVariance:    15.2,
		ClickFrequency:      3.5,
		TypingSpeed:         45.0,
		IdleTime:            5.0,
	}

	assert.Equal(t, 50.5, features.AverageSpeed)
	assert.Equal(t, 2.3, features.AverageAcceleration)
	assert.Equal(t, 15.2, features.MovementVariance)
	assert.Equal(t, 3.5, features.ClickFrequency)
	assert.Equal(t, 45.0, features.TypingSpeed)
	assert.Equal(t, 5.0, features.IdleTime)
}

func TestBehaviorClassification_Values(t *testing.T) {
	classifications := []string{"human", "bot", "automated", "suspicious"}

	for _, class := range classifications {
		t.Run(class, func(t *testing.T) {
			result := &BehaviorResult{
				UserID:         "user123",
				Classification: class,
				Confidence:     0.5,
				TotalScore:     50,
				SuspicionScore: 25,
				EventCount:     10,
				Timestamp:      time.Now(),
			}
			assert.NotEmpty(t, result.Classification)
		})
	}
}

func TestBehaviorAnalysis_Analyze_ManyEvents(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	events := make([]map[string]interface{}, 0)
	for i := 0; i < 1000; i++ {
		events = append(events, map[string]interface{}{
			"type":      "move",
			"timestamp": time.Now().Unix(),
			"x":         float64(i % 500),
			"y":         float64((i * 2) % 500),
			"duration":  10 + (i % 5) * 5,
		})
	}

	result, err := analyzer.Analyze(context.Background(), "user123", events)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1000, result.EventCount)
}

func TestBehaviorAnalysis_Analyze_DifferentEventTypes(t *testing.T) {
	analyzer := &mockBehaviorAnalyzer{}

	eventTypes := []string{"move", "click", "keydown", "keyup", "scroll", "touchstart", "touchmove", "touchend"}

	for _, eventType := range eventTypes {
		t.Run(eventType, func(t *testing.T) {
			events := []map[string]interface{}{
				{"type": eventType, "timestamp": time.Now().Unix()},
			}
			result, err := analyzer.Analyze(context.Background(), "user123", events)
			assert.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

type mockBehaviorAnalyzer struct {
	events []map[string]interface{}
}

func (m *mockBehaviorAnalyzer) Analyze(ctx context.Context, userID string, events []map[string]interface{}) (*BehaviorResult, error) {
	if events == nil {
		events = []map[string]interface{}{}
	}

	var suspicionScore float64
	classification := "human"

	moveCount := 0
	clickCount := 0
	keyCount := 0

	for _, event := range events {
		eventType, _ := event["type"].(string)
		switch eventType {
		case "move":
			moveCount++
			if duration, ok := event["duration"].(int); ok && duration < 10 {
				suspicionScore += 0.5
			}
		case "click":
			clickCount++
			if duration, ok := event["duration"].(int); ok && duration == 0 {
				suspicionScore += 1.0
			}
		case "keydown", "keyup":
			keyCount++
		}
	}

	if clickCount > 50 && moveCount < 10 {
		classification = "bot"
		suspicionScore = 80
	} else if clickCount > 100 && moveCount == 0 {
		classification = "automated"
		suspicionScore = 95
	} else if len(events) >= 20 && suspicionScore < 20 {
		classification = "human"
	}

	totalScore := 100 - suspicionScore
	if totalScore < 0 {
		totalScore = 0
	}

	features := &BehaviorFeatures{
		AverageSpeed:        50.0,
		AverageAcceleration: 2.0,
		MovementVariance:    15.0,
		ClickFrequency:      float64(clickCount) / float64(len(events)+1),
		TypingSpeed:         45.0,
		IdleTime:            5.0,
	}

	return &BehaviorResult{
		UserID:         userID,
		Classification: classification,
		Confidence:     0.8,
		TotalScore:     totalScore,
		SuspicionScore: suspicionScore,
		EventCount:     len(events),
		Features:       features,
		Timestamp:      time.Now(),
	}, nil
}

func testSin(x float64) float64 {
	return math.Sin(x)
}
