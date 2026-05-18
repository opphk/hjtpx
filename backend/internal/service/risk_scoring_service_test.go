package service

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestRiskScoringService_DefaultConfig(t *testing.T) {
	service := NewRiskScoringService()
	assert.NotNil(t, service)

	config := service.GetConfig()
	assert.NotNil(t, config)
	assert.True(t, config.IsEnabled)
	assert.True(t, config.AutoAdjust)
	assert.True(t, config.Weights.Validate())
}

func TestRiskScoringWeights_Validate(t *testing.T) {
	tests := []struct {
		name     string
		weights  model.RiskScoringWeights
		expected bool
	}{
		{
			name: "valid weights",
			weights: model.RiskScoringWeights{
				TraceWeight:    0.25,
				EnvWeight:      0.20,
				BehaviorWeight: 0.25,
				DeviceWeight:   0.15,
				HistoryWeight:  0.15,
			},
			expected: true,
		},
		{
			name: "weights sum to 1",
			weights: model.RiskScoringWeights{
				TraceWeight:    0.30,
				EnvWeight:      0.20,
				BehaviorWeight: 0.20,
				DeviceWeight:   0.15,
				HistoryWeight:  0.15,
			},
			expected: true,
		},
		{
			name: "invalid weights sum less than 1",
			weights: model.RiskScoringWeights{
				TraceWeight:    0.10,
				EnvWeight:      0.10,
				BehaviorWeight: 0.10,
				DeviceWeight:   0.10,
				HistoryWeight:  0.10,
			},
			expected: false,
		},
		{
			name: "invalid weights sum more than 1",
			weights: model.RiskScoringWeights{
				TraceWeight:    0.30,
				EnvWeight:      0.30,
				BehaviorWeight: 0.30,
				DeviceWeight:   0.30,
				HistoryWeight:  0.30,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.weights.Validate()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRiskScoringWeights_Normalize(t *testing.T) {
	weights := model.RiskScoringWeights{
		TraceWeight:    30,
		EnvWeight:      20,
		BehaviorWeight: 25,
		DeviceWeight:   15,
		HistoryWeight:  10,
	}

	weights.Normalize()

	total := weights.TraceWeight + weights.EnvWeight + weights.BehaviorWeight +
		weights.DeviceWeight + weights.HistoryWeight

	assert.InDelta(t, 1.0, total, 0.001)
}

func TestRiskScoringService_UpdateWeights(t *testing.T) {
	service := NewRiskScoringService()

	weights := &model.RiskScoringWeights{
		TraceWeight:    0.30,
		EnvWeight:      0.20,
		BehaviorWeight: 0.20,
		DeviceWeight:   0.15,
		HistoryWeight:  0.15,
	}

	err := service.UpdateWeights(weights)
	assert.NoError(t, err)

	retrieved := service.GetWeights()
	assert.Equal(t, 0.30, retrieved.TraceWeight)
	assert.Equal(t, 0.20, retrieved.EnvWeight)
}

func TestRiskScoringService_UpdateWeights_Invalid(t *testing.T) {
	service := NewRiskScoringService()

	weights := &model.RiskScoringWeights{
		TraceWeight:    -0.10,
		EnvWeight:      0.20,
		BehaviorWeight: 0.20,
		DeviceWeight:   0.15,
		HistoryWeight:  0.15,
	}

	err := service.UpdateWeights(weights)
	assert.Error(t, err)
}

func TestRiskScoringService_UpdateThresholds(t *testing.T) {
	service := NewRiskScoringService()

	thresholds := &model.RiskThresholds{
		LowMax:      25,
		MediumMax:   45,
		HighMax:     65,
		CriticalMax: 100,
		VerifyMin:   35,
		BlockMin:    75,
	}

	err := service.UpdateThresholds(thresholds)
	assert.NoError(t, err)

	retrieved := service.GetThresholds()
	assert.Equal(t, 25.0, retrieved.LowMax)
	assert.Equal(t, 45.0, retrieved.MediumMax)
}

func TestRiskScoringService_UpdateThresholds_InvalidOrder(t *testing.T) {
	service := NewRiskScoringService()

	thresholds := &model.RiskThresholds{
		LowMax:      50,
		MediumMax:   30,
		HighMax:     65,
		CriticalMax: 100,
		VerifyMin:   35,
		BlockMin:    75,
	}

	err := service.UpdateThresholds(thresholds)
	assert.Error(t, err)
}

func TestRiskScoringService_CalculateScore_EmptyContext(t *testing.T) {
	service := NewRiskScoringService()

	ctx := &model.RiskContext{}
	score := service.CalculateScore(ctx)

	assert.NotNil(t, score)
	assert.Greater(t, score.TotalScore, 0.0)
	assert.LessOrEqual(t, score.TotalScore, 100.0)
}

func TestRiskScoringService_CalculateScore_WithTraceData(t *testing.T) {
	service := NewRiskScoringService()

	traceData := []model.TracePoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 100, Y: 100, Timestamp: 100},
		{X: 200, Y: 200, Timestamp: 200},
		{X: 300, Y: 300, Timestamp: 300},
	}

	ctx := &model.RiskContext{
		TraceData: traceData,
	}

	score := service.CalculateScore(ctx)

	assert.NotNil(t, score)
	assert.Greater(t, score.TraceScore, 0.0)
}

func TestRiskScoringService_CalculateScore_WithHighRiskIndicators(t *testing.T) {
	service := NewRiskScoringService()

	ctx := &model.RiskContext{
		IsProxy:      true,
		IsVPN:       true,
		FailureCount: 5,
		MouseSpeed:   3000,
		TimeFromStart: 300,
	}

	score := service.CalculateScore(ctx)

	assert.NotNil(t, score)
	assert.Greater(t, score.TotalScore, 50.0)
}

func TestRiskScoringService_CalculateScore_WithLowRiskIndicators(t *testing.T) {
	service := NewRiskScoringService()

	ctx := &model.RiskContext{
		VerificationCount: 10,
		FailureCount:     0,
		Timezone:         "Asia/Shanghai",
		Language:         "zh-CN",
		HasTouchDevice:   true,
	}

	score := service.CalculateScore(ctx)

	assert.NotNil(t, score)
	assert.Less(t, score.TotalScore, 50.0)
}

func TestRiskScoringService_DetermineRiskLevel(t *testing.T) {
	service := NewRiskScoringService()

	tests := []struct {
		name       string
		score      float64
		expected   model.RiskLevel
	}{
		{"very low score", 10.0, model.RiskLevelLow},
		{"low boundary", 30.0, model.RiskLevelLow},
		{"medium boundary", 31.0, model.RiskLevelMedium},
		{"medium", 45.0, model.RiskLevelMedium},
		{"high boundary", 50.0, model.RiskLevelHigh},
		{"high", 65.0, model.RiskLevelHigh},
		{"critical", 85.0, model.RiskLevelCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := service.determineRiskLevel(tt.score)
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestRiskScoringService_GetAction(t *testing.T) {
	service := NewRiskScoringService()

	tests := []struct {
		name     string
		score    float64
		expected string
	}{
		{"low risk - allow", 20.0, "allow"},
		{"medium risk - verify", 45.0, "verify"},
		{"high risk - block", 85.0, "block"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := &model.MultiDimensionalScore{
				TotalScore: tt.score,
			}
			action := service.GetAction(score)
			assert.Equal(t, tt.expected, action)
		})
	}
}

func TestRiskScoringService_ExportImportConfig(t *testing.T) {
	service := NewRiskScoringService()

	customWeights := &model.RiskScoringWeights{
		TraceWeight:    0.30,
		EnvWeight:      0.15,
		BehaviorWeight: 0.30,
		DeviceWeight:   0.10,
		HistoryWeight:  0.15,
	}
	service.UpdateWeights(customWeights)

	configJSON, err := service.ExportConfig()
	assert.NoError(t, err)
	assert.NotEmpty(t, configJSON)

	service2 := NewRiskScoringService()
	err = service2.ImportConfig(configJSON)
	assert.NoError(t, err)

	importedWeights := service2.GetWeights()
	assert.InDelta(t, customWeights.TraceWeight, importedWeights.TraceWeight, 0.001)
}

func TestRiskScoringService_ResetToDefault(t *testing.T) {
	service := NewRiskScoringService()

	weights := &model.RiskScoringWeights{
		TraceWeight:    0.50,
		EnvWeight:      0.10,
		BehaviorWeight: 0.20,
		DeviceWeight:   0.10,
		HistoryWeight:  0.10,
	}
	service.UpdateWeights(weights)

	service.ResetToDefault()

	defaultWeights := model.DefaultRiskScoringConfig().Weights
	retrieved := service.GetWeights()

	assert.InDelta(t, defaultWeights.TraceWeight, retrieved.TraceWeight, 0.001)
}

func TestRiskScoringService_GetScoreBreakdown(t *testing.T) {
	service := NewRiskScoringService()

	ctx := &model.RiskContext{
		SessionID:  "test-session",
		IPAddress:  "192.168.1.1",
		FailureCount: 2,
	}

	breakdown := service.GetScoreBreakdown(ctx)

	assert.NotNil(t, breakdown)
	assert.Contains(t, breakdown, "trace")
	assert.Contains(t, breakdown, "environment")
	assert.Contains(t, breakdown, "behavior")
	assert.Contains(t, breakdown, "device")
	assert.Contains(t, breakdown, "history")
	assert.Contains(t, breakdown, "total")

	totalMap := breakdown["total"].(map[string]interface{})
	assert.Contains(t, totalMap, "score")
	assert.Contains(t, totalMap, "level")
	assert.Contains(t, totalMap, "action")
}

func TestRiskScoringService_CalculateConfidence(t *testing.T) {
	service := NewRiskScoringService()

	confidence := service.calculateConfidence(50, 50, 50, 50, 50)
	assert.Greater(t, confidence, 0.5)
	assert.LessOrEqual(t, confidence, 0.95)

	confidence = service.calculateConfidence(0, 0, 0, 0, 0)
	assert.Less(t, confidence, 0.9)
}

func TestMultiDimensionalScore_JSON(t *testing.T) {
	score := &model.MultiDimensionalScore{
		TraceScore:    75.5,
		EnvScore:      30.0,
		BehaviorScore: 45.0,
		DeviceScore:   20.0,
		HistoryScore:  35.0,
		TotalScore:    45.5,
		RiskLevel:     model.RiskLevelMedium,
		Confidence:    0.85,
		Timestamp:     time.Now().Unix(),
	}

	data, err := json.Marshal(score)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded model.MultiDimensionalScore
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, score.TraceScore, decoded.TraceScore)
	assert.Equal(t, score.RiskLevel, decoded.RiskLevel)
}

func TestRiskScoringConfig_DefaultValues(t *testing.T) {
	config := model.DefaultRiskScoringConfig()

	assert.NotNil(t, config)
	assert.True(t, config.IsEnabled)
	assert.True(t, config.AutoAdjust)
	assert.Equal(t, 0.25, config.Weights.TraceWeight)
	assert.Equal(t, 0.20, config.Weights.EnvWeight)
	assert.Equal(t, 0.25, config.Weights.BehaviorWeight)
	assert.Equal(t, 0.15, config.Weights.DeviceWeight)
	assert.Equal(t, 0.15, config.Weights.HistoryWeight)
	assert.Equal(t, 30.0, config.Thresholds.LowMax)
	assert.Equal(t, 50.0, config.Thresholds.MediumMax)
	assert.Equal(t, 70.0, config.Thresholds.HighMax)
	assert.Equal(t, 100.0, config.Thresholds.CriticalMax)
}

func TestRiskScoringHistory_TableName(t *testing.T) {
	history := model.RiskScoringHistory{}
	assert.Equal(t, "risk_scoring_history", history.TableName())
}

func TestScoreBand_DefaultBands(t *testing.T) {
	bands := model.DefaultScoreBands

	assert.Len(t, bands, 4)

	assert.Equal(t, 0.0, bands[0].MinScore)
	assert.Equal(t, 30.0, bands[0].MaxScore)
	assert.Equal(t, "low", bands[0].Label)

	assert.Equal(t, 30.0, bands[1].MinScore)
	assert.Equal(t, 50.0, bands[1].MaxScore)
	assert.Equal(t, "medium", bands[1].Label)

	assert.Equal(t, 50.0, bands[2].MinScore)
	assert.Equal(t, 70.0, bands[2].MaxScore)
	assert.Equal(t, "high", bands[2].Label)

	assert.Equal(t, 70.0, bands[3].MinScore)
	assert.Equal(t, 100.0, bands[3].MaxScore)
	assert.Equal(t, "critical", bands[3].Label)
}

func TestRiskScoringService_TraceScoreCalculation(t *testing.T) {
	service := NewRiskScoringService()

	tests := []struct {
		name      string
		traceData []model.TracePoint
		minScore  float64
		maxScore  float64
	}{
		{
			name:      "insufficient data",
			traceData: []model.TracePoint{},
			minScore:  45.0,
			maxScore:  55.0,
		},
		{
			name: "straight line - high risk",
			traceData: []model.TracePoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 100, Y: 0, Timestamp: 100},
				{X: 200, Y: 0, Timestamp: 200},
				{X: 300, Y: 0, Timestamp: 300},
			},
			minScore: 20.0,
			maxScore: 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &model.RiskContext{
				TraceData: tt.traceData,
			}
			score := service.CalculateScore(ctx)
			assert.GreaterOrEqual(t, score.TraceScore, tt.minScore)
			assert.LessOrEqual(t, score.TraceScore, tt.maxScore)
		})
	}
}

func TestRiskScoringService_EnvScoreCalculation(t *testing.T) {
	service := NewRiskScoringService()

	tests := []struct {
		name        string
		ctx         *model.RiskContext
		minExpected float64
		maxExpected float64
	}{
		{
			name: "clean environment",
			ctx: &model.RiskContext{
				Timezone: "Asia/Shanghai",
				Language: "zh-CN",
			},
			minExpected: 0.0,
			maxExpected: 15.0,
		},
		{
			name: "risky environment",
			ctx: &model.RiskContext{
				IsProxy:      true,
				IsVPN:       true,
				IsTor:       true,
				IPReputation: "bad",
			},
			minExpected: 60.0,
			maxExpected: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.CalculateScore(tt.ctx)
			assert.GreaterOrEqual(t, score.EnvScore, tt.minExpected)
			assert.LessOrEqual(t, score.EnvScore, tt.maxExpected)
		})
	}
}

func TestRiskScoringService_BehaviorScoreCalculation(t *testing.T) {
	service := NewRiskScoringService()

	tests := []struct {
		name        string
		ctx         *model.RiskContext
		minExpected float64
		maxExpected float64
	}{
		{
			name: "good behavior",
			ctx: &model.RiskContext{
				VerificationCount: 10,
				FailureCount:     0,
				TimeFromStart:    5000,
				MouseSpeed:       200,
			},
			minExpected: 0.0,
			maxExpected: 15.0,
		},
		{
			name: "suspicious behavior",
			ctx: &model.RiskContext{
				FailureCount:  5,
				TimeFromStart: 300,
				MouseSpeed:    3000,
			},
			minExpected: 50.0,
			maxExpected: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.CalculateScore(tt.ctx)
			assert.GreaterOrEqual(t, score.BehaviorScore, tt.minExpected)
			assert.LessOrEqual(t, score.BehaviorScore, tt.maxExpected)
		})
	}
}

func TestRiskScoringService_WeightedTotalScore(t *testing.T) {
	service := NewRiskScoringService()

	ctx := &model.RiskContext{}

	score := service.CalculateScore(ctx)

	weights := service.GetWeights()
	expectedTotal := score.TraceScore*weights.TraceWeight +
		score.EnvScore*weights.EnvWeight +
		score.BehaviorScore*weights.BehaviorWeight +
		score.DeviceScore*weights.DeviceWeight +
		score.HistoryScore*weights.HistoryWeight

	assert.InDelta(t, expectedTotal, score.TotalScore, 0.001)
}

func TestRiskScoringService_EvaluateWithVerification(t *testing.T) {
	service := NewRiskScoringService()

	ctx := &model.RiskContext{
		SessionID: "test-session",
		IPAddress: "192.168.1.1",
	}

	result, err := service.EvaluateWithVerification(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.RiskScore, 0.0)
	assert.LessOrEqual(t, result.RiskScore, 100.0)
	assert.Contains(t, []model.RiskLevel{model.RiskLevelLow, model.RiskLevelMedium, model.RiskLevelHigh, model.RiskLevelCritical}, result.RiskLevel)
}

func TestRiskScoringService_NormalUserFalsePositiveRate(t *testing.T) {
	service := NewRiskScoringService()

	falsePositives := 0
	totalTests := 1000

	for i := 0; i < totalTests; i++ {
		ctx := &model.RiskContext{
			SessionID:         "normal-user",
			VerificationCount: 5,
			FailureCount:     0,
			Timezone:         "Asia/Shanghai",
			Language:         "zh-CN",
			HasTouchDevice:   true,
			TimeFromStart:    5000,
			MouseSpeed:       300,
		}

		score := service.CalculateScore(ctx)

		if score.TotalScore >= service.GetThresholds().VerifyMin {
			falsePositives++
		}
	}

	falsePositiveRate := float64(falsePositives) / float64(totalTests)
	assert.Less(t, falsePositiveRate, 0.005, "False positive rate should be less than 0.5%")
}

func TestRiskScoringService_MultipleDimensionConsistency(t *testing.T) {
	service := NewRiskScoringService()

	ctx := &model.RiskContext{
		SessionID:     "test-user",
		FailureCount: 1,
		Timezone:     "UTC",
		Language:     "en",
	}

	score := service.CalculateScore(ctx)

	assert.Greater(t, score.TotalScore, 0.0)
	assert.LessOrEqual(t, score.TotalScore, 100.0)
	assert.Greater(t, score.Confidence, 0.0)
	assert.LessOrEqual(t, score.Confidence, 0.95)
}

func TestRiskScoringService_ConcurrentAccess(t *testing.T) {
	service := NewRiskScoringService()

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = service.CalculateScore(&model.RiskContext{
					SessionID: "concurrent-test",
				})
				_ = service.GetWeights()
				_ = service.GetThresholds()
				_ = service.GetConfig()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRiskScoringService_UpdateConfig_InvalidWeights(t *testing.T) {
	service := NewRiskScoringService()

	config := &model.RiskScoringConfig{
		Weights: model.RiskScoringWeights{
			TraceWeight:    -1.0,
			EnvWeight:      0.20,
			BehaviorWeight: 0.25,
			DeviceWeight:   0.15,
			HistoryWeight:  0.15,
		},
	}

	err := service.UpdateConfig(config)
	assert.Error(t, err)
}

func TestRiskScoringService_BoundaryConditions(t *testing.T) {
	service := NewRiskScoringService()

	ctx := &model.RiskContext{
		MouseSpeed: math.MaxFloat64,
	}
	score := service.CalculateScore(ctx)
	assert.LessOrEqual(t, score.TotalScore, 100.0)

	ctx2 := &model.RiskContext{
		MouseSpeed: -math.MaxFloat64,
	}
	score2 := service.CalculateScore(ctx2)
	assert.GreaterOrEqual(t, score2.TotalScore, 0.0)

	ctx3 := &model.RiskContext{
		FailureCount: math.MaxInt,
	}
	score3 := service.CalculateScore(ctx3)
	assert.LessOrEqual(t, score3.BehaviorScore, 100.0)
}
