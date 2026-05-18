package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetermineRiskLevel(t *testing.T) {
	tests := []struct {
		name     string
		score    float64
		expected RiskLevel
	}{
		{"critical low score", 0, RiskLevelCritical},
		{"critical high boundary", 39.9, RiskLevelCritical},
		{"high boundary", 40, RiskLevelHigh},
		{"high upper boundary", 59.9, RiskLevelHigh},
		{"medium boundary", 60, RiskLevelMedium},
		{"medium upper boundary", 79.9, RiskLevelMedium},
		{"low boundary", 80, RiskLevelLow},
		{"low max score", 100, RiskLevelLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := DetermineRiskLevel(tt.score)
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestCalculateHumanProbability(t *testing.T) {
	tests := []struct {
		name    string
		score   float64
		minProb float64
		maxProb float64
	}{
		{"zero score", 0, 1.0, 1.0},
		{"low score", 10, 10.8, 10.9},
		{"medium score", 50, 50.0, 50.1},
		{"high score", 80, 79.4, 79.5},
		{"max score", 100, 99.0, 99.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prob := CalculateHumanProbability(tt.score)
			assert.GreaterOrEqual(t, prob, tt.minProb)
			assert.LessOrEqual(t, prob, tt.maxProb)
		})
	}
}

func TestNewRiskContext(t *testing.T) {
	ctx := NewRiskContext()

	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.TraceData)
	assert.NotNil(t, ctx.BrowserPlugins)
	assert.NotNil(t, ctx.EnvInfo)
	assert.NotNil(t, ctx.DeviceInfo)
	assert.Empty(t, ctx.TraceData)
	assert.Empty(t, ctx.BrowserPlugins)
}

func TestRiskContext_HasHighRiskIndicators(t *testing.T) {
	tests := []struct {
		name          string
		isProxy       bool
		isVPN         bool
		isTor         bool
		failureCount  int
		mouseSpeed    float64
		timeFromStart int64
		expected      bool
	}{
		{"all safe", false, false, false, 0, 100, 5000, false},
		{"is proxy", true, false, false, 0, 100, 5000, true},
		{"is VPN", false, true, false, 0, 100, 5000, true},
		{"is Tor", false, false, true, 0, 100, 5000, true},
		{"high failure count", false, false, false, 3, 100, 5000, true},
		{"high mouse speed", false, false, false, 0, 2001, 5000, true},
		{"too fast", false, false, false, 0, 100, 400, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RiskContext{
				IsProxy:       tt.isProxy,
				IsVPN:         tt.isVPN,
				IsTor:         tt.isTor,
				FailureCount:  tt.failureCount,
				MouseSpeed:    tt.mouseSpeed,
				TimeFromStart: tt.timeFromStart,
			}
			assert.Equal(t, tt.expected, ctx.HasHighRiskIndicators())
		})
	}
}

func TestRiskContext_GetTrustScore(t *testing.T) {
	tests := []struct {
		name              string
		verificationCount int
		failureCount      int
		isProxy           bool
		isVPN             bool
		isTor             bool
		hasTouchDevice    bool
		timezone          string
		language          string
		expectedScore     float64
	}{
		{"base only", 0, 0, false, false, false, false, "", "", 120.0},
		{"with verification", 6, 0, false, false, false, false, "", "", 145.0},
		{"with success", 1, 0, false, false, false, false, "", "", 135.0},
		{"safe network", 0, 0, false, false, false, false, "", "", 120.0},
		{"with touch", 0, 0, false, false, false, true, "", "", 125.0},
		{"with meta", 0, 0, false, false, false, false, "Asia/Shanghai", "zh-CN", 125.0},
		{"full trust", 10, 0, false, false, false, true, "America/New_York", "en-US", 155.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RiskContext{
				VerificationCount: tt.verificationCount,
				FailureCount:      tt.failureCount,
				IsProxy:           tt.isProxy,
				IsVPN:             tt.isVPN,
				IsTor:             tt.isTor,
				HasTouchDevice:    tt.hasTouchDevice,
				Timezone:          tt.timezone,
				Language:          tt.language,
			}
			score := ctx.GetTrustScore()
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}

func TestRiskResult_AddRiskFactor(t *testing.T) {
	result := &RiskResult{
		RiskFactors: []string{},
	}

	result.AddRiskFactor("factor1")
	assert.Len(t, result.RiskFactors, 1)
	assert.Contains(t, result.RiskFactors, "factor1")

	result.AddRiskFactor("factor2")
	assert.Len(t, result.RiskFactors, 2)

	result.AddRiskFactor("factor1")
	assert.Len(t, result.RiskFactors, 2)
}

func TestRiskResult_SortRiskFactors(t *testing.T) {
	result := &RiskResult{
		RiskFactors: []string{"zebra", "apple", "mango"},
	}

	result.SortRiskFactors()
	assert.Equal(t, []string{"apple", "mango", "zebra"}, result.RiskFactors)
}

func TestRiskResult_ToJSON(t *testing.T) {
	result := &RiskResult{
		RiskLevel:        RiskLevelLow,
		RiskScore:        85.5,
		PositionScore:    90.0,
		TraceScore:       80.0,
		EnvScore:         85.0,
		RiskFactors:      []string{"factor1"},
		Action:           "pass",
		RecommendVerify:  false,
		HumanProbability: 95.0,
	}

	jsonStr, err := result.ToJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonStr)
	assert.Contains(t, jsonStr, "low")
	assert.Contains(t, jsonStr, "85.5")
}

func TestParseRiskResult(t *testing.T) {
	jsonStr := `{
		"risk_level": "medium",
		"risk_score": 65.5,
		"position_score": 70.0,
		"trace_score": 60.0,
		"env_score": 65.0,
		"risk_factors": ["factor1", "factor2"],
		"action": "verify",
		"recommend_verify": true,
		"human_probability": 75.0
	}`

	result, err := ParseRiskResult(jsonStr)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, RiskLevelMedium, result.RiskLevel)
	assert.Equal(t, 65.5, result.RiskScore)
	assert.Len(t, result.RiskFactors, 2)
	assert.True(t, result.RecommendVerify)
}

func TestParseRiskResult_InvalidJSON(t *testing.T) {
	_, err := ParseRiskResult("invalid json")
	assert.Error(t, err)
}

func TestRiskLog_SetRiskFactors(t *testing.T) {
	log := &RiskLog{}

	err := log.SetRiskFactors([]string{"factor1", "factor2", "factor3"})
	assert.NoError(t, err)
	assert.Equal(t, `["factor1","factor2","factor3"]`, log.RiskFactors)
}

func TestRiskLog_GetRiskFactors(t *testing.T) {
	log := &RiskLog{
		RiskFactors: `["factor1","factor2","factor3"]`,
	}

	factors, err := log.GetRiskFactors()
	assert.NoError(t, err)
	assert.Len(t, factors, 3)
	assert.Contains(t, factors, "factor1")
	assert.Contains(t, factors, "factor2")
	assert.Contains(t, factors, "factor3")
}

func TestRiskLog_GetRiskFactors_Empty(t *testing.T) {
	log := &RiskLog{
		RiskFactors: "",
	}

	factors, err := log.GetRiskFactors()
	assert.NoError(t, err)
	assert.Empty(t, factors)
}

func TestRiskLog_GetRiskFactors_InvalidJSON(t *testing.T) {
	log := &RiskLog{
		RiskFactors: "invalid",
	}

	_, err := log.GetRiskFactors()
	assert.Error(t, err)
}

func TestRiskLevel_Constants(t *testing.T) {
	assert.Equal(t, RiskLevel("low"), RiskLevelLow)
	assert.Equal(t, RiskLevel("medium"), RiskLevelMedium)
	assert.Equal(t, RiskLevel("high"), RiskLevelHigh)
	assert.Equal(t, RiskLevel("critical"), RiskLevelCritical)
}
