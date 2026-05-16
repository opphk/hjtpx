package service

import (
	"testing"

	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestNewBehaviorAnalysisService(t *testing.T) {
	service := NewBehaviorAnalysisService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.weights)
}

func TestAnalyzeBehaviorNormal(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{
			SessionID: "test-session",
			DataType:  "mousemove",
			Data:      `{"x": 100, "y": 200, "timestamp": 1000, "event": "mousemove"}`,
		},
		{
			SessionID: "test-session",
			DataType:  "mousemove",
			Data:      `{"x": 110, "y": 210, "timestamp": 1100, "event": "mousemove"}`,
		},
		{
			SessionID: "test-session",
			DataType:  "mousemove",
			Data:      `{"x": 120, "y": 220, "timestamp": 1200, "event": "mousemove"}`,
		},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.RiskScore, 0.0)
	assert.LessOrEqual(t, result.RiskScore, 100.0)
}

func TestAnalyzeBehaviorWithClicks(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{
		{
			SessionID: "test-session",
			DataType:  "mousemove",
			Data:      `{"x": 100, "y": 200, "timestamp": 1000, "event": "mousemove"}`,
		},
		{
			SessionID: "test-session",
			DataType:  "mousemove",
			Data:      `{"x": 110, "y": 210, "timestamp": 1100, "event": "click"}`,
		},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAnalyzeBehaviorEmptyData(t *testing.T) {
	service := NewBehaviorAnalysisService()

	behaviorData := []models.BehaviorData{}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAnalyzeBehaviorWithExtendedData(t *testing.T) {
	service := NewBehaviorAnalysisService()

	extendedJSON := `{
		"mouse_trajectory": [
			{"x": 0, "y": 0, "timestamp": 0},
			{"x": 50, "y": 50, "timestamp": 100},
			{"x": 100, "y": 100, "timestamp": 200}
		],
		"click_data": [
			{"x": 100, "y": 100, "button": 0, "timestamp": 300}
		]
	}`

	behaviorData := []models.BehaviorData{
		{
			SessionID: "test-session",
			DataType:  "extended",
			Data:      extendedJSON,
		},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAnalyzeBehaviorWithClickInfo(t *testing.T) {
	service := NewBehaviorAnalysisService()

	extendedJSON := `{
		"mouse_trajectory": [
			{"x": 0, "y": 0, "timestamp": 0},
			{"x": 100, "y": 100, "timestamp": 100}
		],
		"click_data": [
			{"x": 100, "y": 100, "button": 0, "timestamp": 200, "hold_duration": 50},
			{"x": 200, "y": 200, "button": 0, "timestamp": 400, "hold_duration": 60}
		]
	}`

	behaviorData := []models.BehaviorData{
		{
			SessionID: "test-session",
			DataType:  "extended",
			Data:      extendedJSON,
		},
	}

	result, err := service.AnalyzeBehavior(behaviorData)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRiskWeights(t *testing.T) {
	service := NewBehaviorAnalysisService()

	weights := service.weights

	assert.Greater(t, weights.SpeedWeight, 0.0)
	assert.Greater(t, weights.AccelerationWeight, 0.0)
	assert.Greater(t, weights.TrajectoryWeight, 0.0)
	assert.Greater(t, weights.ClickWeight, 0.0)
	assert.Greater(t, weights.KeyboardWeight, 0.0)
	assert.Greater(t, weights.EnvironmentWeight, 0.0)

	totalWeight := weights.SpeedWeight + weights.AccelerationWeight + weights.TrajectoryWeight +
		weights.ClickWeight + weights.KeyboardWeight + weights.EnvironmentWeight
	assert.Equal(t, 1.0, totalWeight)
}

func TestMouseTrajectoryStructure(t *testing.T) {
	trajectory := MouseTrajectory{
		Points: []BehaviorDataPoint{
			{X: 0, Y: 0, Timestamp: 0, Event: "mousemove"},
			{X: 100, Y: 100, Timestamp: 100, Event: "mousemove"},
		},
		TotalDistance:    141.42,
		AverageSpeed:    1.41,
		MaxSpeed:        2.0,
		DirectionChanges: 1,
	}

	assert.Len(t, trajectory.Points, 2)
	assert.Greater(t, trajectory.TotalDistance, 0.0)
}

func TestClickPatternStructure(t *testing.T) {
	pattern := ClickPattern{
		Clicks: []BehaviorDataPoint{
			{X: 100, Y: 200, Timestamp: 1000, Event: "click"},
			{X: 200, Y: 300, Timestamp: 1200, Event: "click"},
		},
		ClickCount:     2,
		ClickSpeed:     5.0,
		Regularity:     0.9,
	}

	assert.Len(t, pattern.Clicks, 2)
	assert.Equal(t, 2, pattern.ClickCount)
}

func TestAnalysisResultStructure(t *testing.T) {
	result := &AnalysisResult{
		Trajectory: MouseTrajectory{
			Points: []BehaviorDataPoint{},
		},
		ClickPattern: ClickPattern{
			Clicks: []BehaviorDataPoint{},
		},
		RiskScore:      45.0,
		RiskIndicators: []string{"slow_movement", "few_clicks"},
		IsBotLikely:    false,
		Confidence:      0.85,
	}

	assert.Equal(t, 45.0, result.RiskScore)
	assert.Len(t, result.RiskIndicators, 2)
	assert.False(t, result.IsBotLikely)
	assert.Equal(t, 0.85, result.Confidence)
}

func TestFeatureVectorStructure(t *testing.T) {
	features := FeatureVector{
		MouseSpeedAvg:   2.5,
		MouseSpeedMax:   5.0,
		ClickCount:      10,
		TrajectoryLength: 500.0,
		PathEfficiency:   0.8,
		DirectionChanges: 5,
	}

	assert.Equal(t, 2.5, features.MouseSpeedAvg)
	assert.Equal(t, 10, features.ClickCount)
}

func TestMLPredictionStructure(t *testing.T) {
	prediction := MLPrediction{
		IsBot:        false,
		Confidence:   0.92,
		BotScore:     0.15,
		HumanScore:   0.85,
		ModelVersion: "v1.0.0",
		FeaturesUsed: []string{"speed", "trajectory", "clicks"},
	}

	assert.False(t, prediction.IsBot)
	assert.Equal(t, 0.92, prediction.Confidence)
	assert.Equal(t, 0.15, prediction.BotScore)
}

func TestExtendedBehaviorDataStructure(t *testing.T) {
	extData := ExtendedBehaviorData{
		MouseTrajectory: []MousePoint{
			{X: 0, Y: 0, Timestamp: 0},
			{X: 100, Y: 100, Timestamp: 100},
		},
		ClickData: []ClickInfo{
			{X: 100, Y: 100, Button: 0, Timestamp: 200},
		},
		EnvironmentData: EnvironmentInfo{
			ScreenWidth:  1920,
			ScreenHeight: 1080,
			Platform:    "Win32",
		},
	}

	assert.Len(t, extData.MouseTrajectory, 2)
	assert.Len(t, extData.ClickData, 1)
	assert.Equal(t, 1920, extData.EnvironmentData.ScreenWidth)
}

func TestClickInfoStructure(t *testing.T) {
	click := ClickInfo{
		X:            100.5,
		Y:            200.5,
		Button:       0,
		Timestamp:    1000,
		HoldDuration: 50,
	}

	assert.Equal(t, 100.5, click.X)
	assert.Equal(t, 200.5, click.Y)
	assert.Equal(t, int64(50), click.HoldDuration)
}

func TestKeyStrokeInfoStructure(t *testing.T) {
	keystroke := KeyStrokeInfo{
		Key:         "a",
		KeyCode:     65,
		Timestamp:   1000,
		HoldTime:    30,
		IsModifier:  false,
	}

	assert.Equal(t, "a", keystroke.Key)
	assert.Equal(t, 65, keystroke.KeyCode)
	assert.False(t, keystroke.IsModifier)
}

func TestScrollInfoStructure(t *testing.T) {
	scroll := ScrollInfo{
		ScrollX:   0,
		ScrollY:   500,
		DeltaX:    0,
		DeltaY:    100,
		Timestamp: 1000,
		Velocity:  50.0,
	}

	assert.Equal(t, int64(500), scroll.ScrollY)
	assert.Equal(t, 50.0, scroll.Velocity)
}

func TestEnvironmentInfoStructure(t *testing.T) {
	env := EnvironmentInfo{
		ScreenWidth:  1920,
		ScreenHeight: 1080,
		ColorDepth:   24,
		Timezone:     "Asia/Shanghai",
		Language:     "zh-CN",
		Platform:     "Win32",
		IsHeadless:  false,
		HasTouchSupport: true,
	}

	assert.Equal(t, 1920, env.ScreenWidth)
	assert.Equal(t, 1080, env.ScreenHeight)
	assert.False(t, env.IsHeadless)
	assert.True(t, env.HasTouchSupport)
}

func TestKeyStrokeAnalysisStructure(t *testing.T) {
	analysis := KeyStrokeAnalysis{
		TotalKeystrokes:  50,
		AverageInterval:  100.0,
		IntervalVariance: 20.0,
		AverageHoldTime:  50.0,
		IsTypingPattern: true,
		TypingRhythm:    0.85,
	}

	assert.Equal(t, 50, analysis.TotalKeystrokes)
	assert.True(t, analysis.IsTypingPattern)
}

func TestScrollAnalysisStructure(t *testing.T) {
	analysis := ScrollAnalysis{
		TotalScrolls:    10,
		AverageVelocity: 50.0,
		MaxVelocity:     100.0,
		ScrollPattern:  0.75,
		DirectionCount: 3,
	}

	assert.Equal(t, 10, analysis.TotalScrolls)
	assert.Equal(t, 3, analysis.DirectionCount)
}

func TestAnalysisCacheStructure(t *testing.T) {
	cache := &AnalysisCache{
		entries: make(map[string]*CachedResult),
		maxSize: 1000,
	}

	assert.NotNil(t, cache.entries)
	assert.Equal(t, 1000, cache.maxSize)
}

func TestStreamProcessorStructure(t *testing.T) {
	processor := &StreamProcessor{
		buffer:        make([]BehaviorDataPoint, 0),
		maxBufferSize: 1000,
	}

	assert.NotNil(t, processor.buffer)
	assert.Equal(t, 1000, processor.maxBufferSize)
	assert.Equal(t, 0, len(processor.buffer))
}
