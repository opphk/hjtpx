package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRiskContext_Build(t *testing.T) {
	service := NewRiskRuleService()

	inputData := map[string]interface{}{
		"session_id":       "test-session-123",
		"ip_address":       "192.168.1.1",
		"user_agent":       "Mozilla/5.0",
		"fingerprint":      "abc123",
		"device_type":      "desktop",
		"request_count":    float64(50),
		"failure_count":    float64(3),
		"success_count":    float64(47),
		"hour":             float64(14),
		"timestamp":        float64(1234567890),
		"is_blacklisted":   false,
		"is_whitelisted":   false,
		"is_vpn":           true,
		"is_proxy":         false,
		"is_tor":           false,
		"is_hosting":       false,
		"ip_reputation":     "medium",
		"country":          "CN",
		"asn":              float64(12345),
		"speed_metrics": map[string]interface{}{
			"avg_speed":          float64(500),
			"max_speed":          float64(800),
			"min_speed":          float64(100),
			"speed_variance":      float64(0.1),
			"speed_consistency":   float64(0.6),
		},
		"trajectory_data": map[string]interface{}{
			"path_efficiency":     float64(0.85),
			"curvature_avg":       float64(0.1),
			"curvature_variance": float64(0.05),
			"direction_changes":   float64(5),
			"micro_corrections":   float64(3),
			"backtrack_count":     float64(1),
			"sinuosity":           float64(1.2),
		},
		"behavior_data": map[string]interface{}{
			"pause_count":          float64(2),
			"total_pause_duration": float64(200),
			"hesitation_time":       float64(100),
			"click_regularity":      float64(0.5),
			"position_entropy":      float64(3.0),
			"human_likeness_score":  float64(0.7),
			"anomaly_score":         float64(0.3),
		},
		"custom_data": map[string]interface{}{
			"fingerprint_occurrences": float64(2),
			"custom_field1":          "value1",
			"custom_field2":          float64(100),
		},
	}

	ctx := service.buildRiskContext(inputData)

	assert.NotNil(t, ctx)
	assert.Equal(t, "test-session-123", ctx.SessionID)
	assert.Equal(t, "192.168.1.1", ctx.IPAddress)
	assert.Equal(t, "Mozilla/5.0", ctx.UserAgent)
	assert.Equal(t, "abc123", ctx.Fingerprint)
	assert.Equal(t, "desktop", ctx.DeviceType)
	assert.Equal(t, 50, ctx.RequestCount)
	assert.Equal(t, 3, ctx.FailureCount)
	assert.Equal(t, 47, ctx.SuccessCount)
	assert.Equal(t, 14, ctx.Hour)
	assert.Equal(t, int64(1234567890), ctx.Timestamp)
	assert.False(t, ctx.IsBlacklisted)
	assert.False(t, ctx.IsWhitelisted)
	assert.True(t, ctx.IsVPN)
	assert.False(t, ctx.IsProxy)
	assert.False(t, ctx.IsTor)
	assert.False(t, ctx.IsHosting)
	assert.Equal(t, "medium", ctx.IPReputation)
	assert.Equal(t, "CN", ctx.Country)
	assert.Equal(t, 12345, ctx.ASN)

	assert.NotNil(t, ctx.SpeedMetrics)
	assert.Equal(t, 500.0, ctx.SpeedMetrics.AvgSpeed)
	assert.Equal(t, 800.0, ctx.SpeedMetrics.MaxSpeed)
	assert.Equal(t, 100.0, ctx.SpeedMetrics.MinSpeed)
	assert.Equal(t, 0.1, ctx.SpeedMetrics.SpeedVariance)
	assert.Equal(t, 0.6, ctx.SpeedMetrics.SpeedConsistency)

	assert.NotNil(t, ctx.TrajectoryData)
	assert.Equal(t, 0.85, ctx.TrajectoryData.PathEfficiency)
	assert.Equal(t, 0.1, ctx.TrajectoryData.CurvatureAvg)
	assert.Equal(t, 0.05, ctx.TrajectoryData.CurvatureVariance)
	assert.Equal(t, 5, ctx.TrajectoryData.DirectionChanges)
	assert.Equal(t, 3, ctx.TrajectoryData.MicroCorrections)
	assert.Equal(t, 1, ctx.TrajectoryData.BacktrackCount)
	assert.Equal(t, 1.2, ctx.TrajectoryData.Sinuosity)

	assert.NotNil(t, ctx.BehaviorData)
	assert.Equal(t, 2, ctx.BehaviorData.PauseCount)
	assert.Equal(t, 200.0, ctx.BehaviorData.TotalPauseDuration)
	assert.Equal(t, 100.0, ctx.BehaviorData.HesitationTime)
	assert.Equal(t, 0.5, ctx.BehaviorData.ClickRegularity)
	assert.Equal(t, 3.0, ctx.BehaviorData.PositionEntropy)
	assert.Equal(t, 0.7, ctx.BehaviorData.HumanLikenessScore)
	assert.Equal(t, 0.3, ctx.BehaviorData.AnomalyScore)

	assert.NotNil(t, ctx.CustomData)
	assert.Equal(t, float64(2), ctx.CustomData["fingerprint_occurrences"])
	assert.Equal(t, "value1", ctx.CustomData["custom_field1"])
	assert.Equal(t, float64(100), ctx.CustomData["custom_field2"])
}

func TestRiskContext_Build_MinimalData(t *testing.T) {
	service := NewRiskRuleService()

	inputData := map[string]interface{}{
		"session_id": "minimal-session",
	}

	ctx := service.buildRiskContext(inputData)

	assert.NotNil(t, ctx)
	assert.Equal(t, "minimal-session", ctx.SessionID)
	assert.Equal(t, "", ctx.IPAddress)
	assert.Equal(t, 0, ctx.RequestCount)
	assert.Nil(t, ctx.SpeedMetrics)
	assert.Nil(t, ctx.TrajectoryData)
	assert.Nil(t, ctx.BehaviorData)
}

func TestRiskContext_Build_EmptyData(t *testing.T) {
	service := NewRiskRuleService()

	ctx := service.buildRiskContext(map[string]interface{}{})

	assert.NotNil(t, ctx)
	assert.Equal(t, "", ctx.SessionID)
}

func TestParseValue(t *testing.T) {
	service := NewRiskRuleService()

	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{"string with double quotes", "\"hello\"", "hello"},
		{"string with single quotes", "'world'", "world"},
		{"integer", "42", float64(42)},
		{"float", "3.14", 3.14},
		{"boolean true", "true", true},
		{"boolean false", "false", false},
		{"plain string", "plaintext", "plaintext"},
		{"negative number", "-10", float64(-10)},
		{"number with spaces", "  100  ", float64(100)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.parseValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompareNumeric(t *testing.T) {
	service := NewRiskRuleService()

	assert.True(t, service.compareNumeric(10.0, 5.0, 1))
	assert.True(t, service.compareNumeric(10.0, 10.0, 0))
	assert.True(t, service.compareNumeric(5.0, 10.0, -1))
	assert.False(t, service.compareNumeric(5.0, 10.0, 1))
	assert.False(t, service.compareNumeric(10.0, 10.0, 1))
}

func TestCompareEqual(t *testing.T) {
	service := NewRiskRuleService()

	assert.True(t, service.compareEqual(10.0, 10.0))
	assert.True(t, service.compareEqual("hello", "hello"))
	assert.True(t, service.compareEqual(nil, nil))
	assert.False(t, service.compareEqual(10.0, 20.0))
	assert.False(t, service.compareEqual("hello", "world"))
	assert.False(t, service.compareEqual(10.0, "10"))
}

func TestStringContains(t *testing.T) {
	service := NewRiskRuleService()

	assert.True(t, service.stringContains("hello world", "world"))
	assert.True(t, service.stringContains("test", "test"))
	assert.False(t, service.stringContains("hello", "world"))
	assert.False(t, service.stringContains("", "test"))
	assert.True(t, service.stringContains("test", ""))
}

func TestValueIn(t *testing.T) {
	service := NewRiskRuleService()

	assert.True(t, service.valueIn("test", "[test, demo, sample]"))
	assert.True(t, service.valueIn("demo", "[test, demo, sample]"))
	assert.False(t, service.valueIn("unknown", "[test, demo, sample]"))
	assert.False(t, service.valueIn("test", ""))
}

func TestMatchesRegex(t *testing.T) {
	service := NewRiskRuleService()

	assert.True(t, service.matchesRegex("test123", "test\\d+"))
	assert.True(t, service.matchesRegex("hello", ".*"))
	assert.False(t, service.matchesRegex("test", "\\d+"))
	assert.True(t, service.matchesRegex("192.168.1.1", "\\d+\\.\\d+\\.\\d+\\.\\d+"))
}

func TestToFloat64(t *testing.T) {
	service := NewRiskRuleService()

	assert.Equal(t, 10.0, service.toFloat64(10))
	assert.Equal(t, 10.0, service.toFloat64(int64(10)))
	assert.Equal(t, 10.0, service.toFloat64(int32(10)))
	assert.Equal(t, 10.5, service.toFloat64(float32(10.5)))
	assert.Equal(t, 10.5, service.toFloat64(10.5))
	assert.Equal(t, 10.0, service.toFloat64("10"))
	assert.Equal(t, 0.0, service.toFloat64("invalid"))
	assert.Equal(t, 0.0, service.toFloat64(nil))
}

func TestEvaluateConditionExpr_BasicComparison(t *testing.T) {
	service := NewRiskRuleService()

	ctx := &RiskContext{
		RequestCount:  100,
		FailureCount:  5,
		Hour:          14,
	}

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{"greater than", "request_count > 50", true},
		{"greater than false", "request_count > 150", false},
		{"less than", "request_count < 150", true},
		{"less than false", "request_count < 50", false},
		{"greater than or equal", "request_count >= 100", true},
		{"less than or equal", "request_count <= 100", true},
		{"equal", "failure_count == 5", true},
		{"not equal", "failure_count != 0", true},
		{"equal false", "hour == 10", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.evaluateConditionExpr(tt.condition, ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateConditionExpr_StringComparison(t *testing.T) {
	service := NewRiskRuleService()

	ctx := &RiskContext{
		IPReputation: "low",
		Country:      "CN",
	}

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{"string equal", "ip_reputation == \"low\"", true},
		{"string equal false", "ip_reputation == \"high\"", false},
		{"country equal", "country == 'CN'", true},
		{"country not equal", "country != 'US'", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.evaluateConditionExpr(tt.condition, ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateConditionExpr_BooleanComparison(t *testing.T) {
	service := NewRiskRuleService()

	ctx := &RiskContext{
		IsBlacklisted: true,
		IsWhitelisted: false,
		IsVPN:         true,
		IsProxy:       false,
		IsTor:         false,
		IsHosting:     false,
	}

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{"is blacklisted true", "is_blacklisted == true", true},
		{"is blacklisted", "is_blacklisted", true},
		{"is whitelisted true", "is_whitelisted == true", false},
		{"is vpn true", "is_vpn == true", true},
		{"is proxy true", "is_proxy == true", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.evaluateConditionExpr(tt.condition, ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateConditionExpr_SpeedMetrics(t *testing.T) {
	service := NewRiskRuleService()

	ctx := &RiskContext{
		SpeedMetrics: &SpeedMetrics{
			AvgSpeed:         2500,
			MaxSpeed:         3500,
			SpeedVariance:    0.001,
			SpeedConsistency: 0.99,
		},
	}

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{"avg speed greater than", "avg_speed > 2000", true},
		{"avg speed less than", "avg_speed < 3000", true},
		{"max speed greater than", "max_speed > 3000", true},
		{"speed variance less than", "speed_variance < 0.01", true},
		{"speed consistency greater than", "speed_consistency > 0.95", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.evaluateConditionExpr(tt.condition, ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateConditionExpr_TrajectoryMetrics(t *testing.T) {
	service := NewRiskRuleService()

	ctx := &RiskContext{
		TrajectoryData: &TrajectoryMetrics{
			PathEfficiency:    0.99,
			CurvatureAvg:      0.001,
			DirectionChanges:  0,
			MicroCorrections:  0,
			BacktrackCount:    0,
			Sinuosity:         1.01,
		},
	}

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{"path efficiency high", "path_efficiency > 0.95", true},
		{"path efficiency too high", "path_efficiency > 0.98", true},
		{"curvature low", "curvature_avg < 0.02", true},
		{"direction changes less", "direction_changes < 3", true},
		{"micro corrections zero", "micro_corrections == 0", true},
		{"sinuosity low", "sinuosity < 1.05", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.evaluateConditionExpr(tt.condition, ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateConditionExpr_BehaviorMetrics(t *testing.T) {
	service := NewRiskRuleService()

	ctx := &RiskContext{
		BehaviorData: &BehaviorMetrics{
			PauseCount:         0,
			TotalPauseDuration:  0,
			HesitationTime:      30,
			ClickRegularity:     0.99,
			HumanLikenessScore: 0.1,
			AnomalyScore:        0.85,
		},
	}

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{"pause count zero", "pause_count == 0", true},
		{"hesitation short", "hesitation_time < 50", true},
		{"click regularity high", "click_regularity > 0.95", true},
		{"human likeness low", "human_likeness_score < 0.2", true},
		{"anomaly score high", "anomaly_score > 0.8", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.evaluateConditionExpr(tt.condition, ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateConditionExpr_CustomData(t *testing.T) {
	service := NewRiskRuleService()

	ctx := &RiskContext{
		CustomData: map[string]interface{}{
			"fingerprint_occurrences": float64(6),
			"custom_field":            "value",
		},
	}

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{"fingerprint occurrences", "fingerprint_occurrences > 5", true},
		{"custom field equal", "custom_field == \"value\"", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.evaluateConditionExpr(tt.condition, ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseConditions_AND(t *testing.T) {
	service := NewRiskRuleService()

	conditions := "request_count > 50 AND failure_count > 0"
	result := service.parseConditions(conditions)

	assert.Len(t, result, 2)
	assert.Contains(t, conditions, result[0])
	assert.Contains(t, conditions, result[1])
}

func TestParseConditions_OR(t *testing.T) {
	service := NewRiskRuleService()

	conditions := "is_vpn == true OR is_proxy == true"
	result := service.parseConditions(conditions)

	assert.Len(t, result, 1)
	assert.Contains(t, result[0], "OR")
}

func TestParseConditions_Mixed(t *testing.T) {
	service := NewRiskRuleService()

	conditions := "request_count > 50 AND is_vpn == true OR failure_count > 3"
	result := service.parseConditions(conditions)

	assert.NotEmpty(t, result)
}

func TestEvaluateSingleCondition_WithParentheses(t *testing.T) {
	service := NewRiskRuleService()

	ctx := &RiskContext{
		Hour:         3,
		RequestCount: 100,
	}

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{"parentheses OR - first true", "(is_vpn == true OR request_count > 50)", true},
		{"parentheses OR - all false", "(is_proxy == true OR is_tor == true)", false},
		{"complex expression", "(hour < 6 OR hour > 22) AND request_count > 50", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.evaluateSingleCondition(tt.condition, ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateSimpleCondition_Whitelisted(t *testing.T) {
	service := NewRiskRuleService()

	ctx := &RiskContext{
		IsWhitelisted: true,
		RequestCount:  1000,
	}

	result := service.evaluateSimpleCondition("request_count > 50", ctx)
	assert.False(t, result)
}

func TestEvaluateSimpleCondition_AllMustBeTrue(t *testing.T) {
	service := NewRiskRuleService()

	tests := []struct {
		name      string
		condition string
		ctx       *RiskContext
		expected  bool
	}{
		{
			"AND - both true",
			"request_count > 50 AND failure_count > 0",
			&RiskContext{RequestCount: 100, FailureCount: 5},
			true,
		},
		{
			"AND - first false",
			"request_count > 50 AND failure_count > 0",
			&RiskContext{RequestCount: 10, FailureCount: 5},
			false,
		},
		{
			"AND - second false",
			"request_count > 50 AND failure_count > 0",
			&RiskContext{RequestCount: 100, FailureCount: 0},
			false,
		},
		{
			"AND - both false",
			"request_count > 50 AND failure_count > 0",
			&RiskContext{RequestCount: 10, FailureCount: 0},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.evaluateSimpleCondition(tt.condition, tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateExpression_Integration(t *testing.T) {
	service := NewRiskRuleService()

	ctx := &RiskContext{
		SpeedMetrics: &SpeedMetrics{
			AvgSpeed:         2500,
			SpeedConsistency: 0.99,
		},
		TrajectoryData: &TrajectoryMetrics{
			PathEfficiency: 0.99,
		},
	}

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{
			"异常速度检测",
			"avg_speed > 2000 OR path_efficiency > 0.95",
			true,
		},
		{
			"过于完美的轨迹",
			"path_efficiency > 0.98 AND speed_consistency > 0.98",
			true,
		},
		{
			"组合风险",
			"path_efficiency > 0.95 AND avg_speed > 1500",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.evaluateExpression(tt.condition, ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateConditionWithContext(t *testing.T) {
	service := NewRiskRuleService()

	tests := []struct {
		name      string
		condition string
		inputData map[string]interface{}
		expected  bool
	}{
		{
			"empty condition",
			"",
			map[string]interface{}{},
			false,
		},
		{
			"IP频率限制 - true",
			"requests_per_minute > 100",
			map[string]interface{}{
				"requests_per_minute": float64(150),
			},
			true,
		},
		{
			"IP频率限制 - false",
			"requests_per_minute > 100",
			map[string]interface{}{
				"requests_per_minute": float64(50),
			},
			false,
		},
		{
			"黑名单IP - true",
			"is_blacklisted == true",
			map[string]interface{}{
				"is_blacklisted": true,
			},
			true,
		},
		{
			"黑名单IP - false",
			"is_blacklisted == true",
			map[string]interface{}{
				"is_blacklisted": false,
			},
			false,
		},
		{
			"VPN检测",
			"is_vpn == true",
			map[string]interface{}{
				"is_vpn": true,
			},
			true,
		},
		{
			"异常时间检测 - true",
			"(hour < 6 OR hour > 22) AND request_count > 50",
			map[string]interface{}{
				"hour":         float64(3),
				"request_count": float64(100),
			},
			true,
		},
		{
			"异常时间检测 - false",
			"(hour < 6 OR hour > 22) AND request_count > 50",
			map[string]interface{}{
				"hour":          float64(14),
				"request_count": float64(100),
			},
			false,
		},
		{
			"复杂组合",
			"avg_speed > 2000 OR path_efficiency > 0.95",
			map[string]interface{}{
				"speed_metrics": map[string]interface{}{
					"avg_speed": float64(2500),
				},
				"trajectory_data": map[string]interface{}{
					"path_efficiency": float64(0.99),
				},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.evaluateConditionWithContext(tt.condition, tt.inputData)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnhancedTemplates(t *testing.T) {
	service := NewRiskRuleService()

	assert.NotNil(t, service)

	_ = &RiskContext{
		RequestCount: 60,
	}
	result1 := service.evaluateConditionWithContext("request_count > 50", map[string]interface{}{
		"request_count": float64(60),
	})
	assert.True(t, result1)

	_ = &RiskContext{
		IsBlacklisted: true,
	}
	result2 := service.evaluateConditionWithContext("is_blacklisted == true", map[string]interface{}{
		"is_blacklisted": true,
	})
	assert.True(t, result2)

	result3 := service.evaluateConditionWithContext("country in [\"XX\", \"YY\"]", map[string]interface{}{
		"country": "XX",
	})
	assert.True(t, result3)

	result4 := service.evaluateConditionWithContext("country in [\"XX\", \"YY\"]", map[string]interface{}{
		"country": "CN",
	})
	assert.False(t, result4)
}

func TestRiskRuleEnginePerformance(t *testing.T) {
	service := NewRiskRuleService()

	inputData := map[string]interface{}{
		"session_id":    "perf-test",
		"ip_address":    "192.168.1.1",
		"request_count": float64(100),
		"failure_count": float64(5),
		"is_vpn":        true,
		"speed_metrics": map[string]interface{}{
			"avg_speed":          float64(2500),
			"max_speed":          float64(3500),
			"speed_variance":      float64(0.001),
			"speed_consistency":   float64(0.99),
		},
		"trajectory_data": map[string]interface{}{
			"path_efficiency":     float64(0.99),
			"curvature_avg":       float64(0.001),
			"direction_changes":   float64(0),
			"micro_corrections":   float64(0),
		},
		"behavior_data": map[string]interface{}{
			"pause_count":          float64(0),
			"hesitation_time":       float64(30),
			"click_regularity":      float64(0.99),
			"human_likeness_score": float64(0.1),
			"anomaly_score":         float64(0.85),
		},
	}

	condition := "request_count > 50 AND (avg_speed > 2000 OR path_efficiency > 0.95) AND (is_vpn == true OR is_proxy == true)"

	for i := 0; i < 100; i++ {
		result := service.evaluateConditionWithContext(condition, inputData)
		assert.NotNil(t, result)
	}
}

func TestConditionSerialization(t *testing.T) {
	condition := "request_count > 50 AND failure_count > 0"

	data, err := json.Marshal(condition)
	assert.NoError(t, err)

	var decoded string
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, condition, decoded)
}

func TestRiskContextSerialization(t *testing.T) {
	ctx := &RiskContext{
		SessionID:   "test-session",
		IPAddress:    "192.168.1.1",
		RequestCount: 100,
		IsVPN:        true,
		SpeedMetrics: &SpeedMetrics{
			AvgSpeed: 500,
		},
		TrajectoryData: &TrajectoryMetrics{
			PathEfficiency: 0.85,
		},
		BehaviorData: &BehaviorMetrics{
			PauseCount: 2,
		},
		CustomData: map[string]interface{}{
			"custom_key": "custom_value",
		},
	}

	data, err := json.Marshal(ctx)
	assert.NoError(t, err)

	var decoded RiskContext
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, ctx.SessionID, decoded.SessionID)
	assert.Equal(t, ctx.IPAddress, decoded.IPAddress)
	assert.Equal(t, ctx.RequestCount, decoded.RequestCount)
	assert.Equal(t, ctx.IsVPN, decoded.IsVPN)
	assert.Equal(t, ctx.SpeedMetrics.AvgSpeed, decoded.SpeedMetrics.AvgSpeed)
	assert.Equal(t, ctx.TrajectoryData.PathEfficiency, decoded.TrajectoryData.PathEfficiency)
	assert.Equal(t, ctx.BehaviorData.PauseCount, decoded.BehaviorData.PauseCount)
	assert.Equal(t, "custom_value", decoded.CustomData["custom_key"])
}
