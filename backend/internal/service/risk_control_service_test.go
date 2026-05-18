package service

import (
	"context"
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestRiskControlService_NewRiskControlService(t *testing.T) {
	service := NewRiskControlService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.deviceRiskCache)
	assert.NotNil(t, service.ipRiskCache)
	assert.NotNil(t, service.behaviorProfiles)
	assert.NotNil(t, service.rules)
	assert.Len(t, service.rules, 15)
}

func TestRiskControlService_GetRiskControlService(t *testing.T) {
	s1 := GetRiskControlService()
	s2 := GetRiskControlService()
	assert.Same(t, s1, s2)
}

func TestRiskControlService_DefaultRules(t *testing.T) {
	service := NewRiskControlService()
	rules := service.GetRules()

	assert.NotEmpty(t, rules)

	var foundDeviceRule, foundIPRule, foundBehaviorRule, foundEnvRule bool
	for _, rule := range rules {
		switch rule.Dimension {
		case "device":
			foundDeviceRule = true
		case "ip":
			foundIPRule = true
		case "behavior":
			foundBehaviorRule = true
		case "env":
			foundEnvRule = true
		}
	}

	assert.True(t, foundDeviceRule, "Should have device dimension rules")
	assert.True(t, foundIPRule, "Should have IP dimension rules")
	assert.True(t, foundBehaviorRule, "Should have behavior dimension rules")
	assert.True(t, foundEnvRule, "Should have environment dimension rules")
}

func TestRiskControlService_EvaluateRisk_LowRisk(t *testing.T) {
	service := NewRiskControlService()

	ctx := &model.RiskContext{
		SessionID:     "test-session-1",
		IPAddress:    "192.168.1.100",
		Fingerprint:  "fp-normal-user-001",
		MouseSpeed:   150,
		HasTouchDevice: true,
		Language:     "zh-CN",
		Timezone:     "Asia/Shanghai",
		PositionDiff: 3,
		TimeFromStart: 5000,
		TraceData: []model.TracePoint{
			{Timestamp: 0, X: 100, Y: 100},
			{Timestamp: 100, X: 150, Y: 150},
			{Timestamp: 200, X: 200, Y: 200},
			{Timestamp: 300, X: 250, Y: 250},
			{Timestamp: 400, X: 300, Y: 300},
		},
		BrowserPlugins: []string{"plugin1", "plugin2"},
	}

	result, err := service.EvaluateRisk(context.Background(), ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.LessOrEqual(t, result.RiskScore, float64(50))
	assert.GreaterOrEqual(t, result.ProcessingTime, int64(0))
}

func TestRiskControlService_EvaluateRisk_HighRiskProxy(t *testing.T) {
	service := NewRiskControlService()

	ctx := &model.RiskContext{
		SessionID:   "test-session-2",
		IPAddress:   "10.0.0.1",
		Fingerprint: "fp-proxy-user-001",
		IsProxy:     true,
		IsVPN:       true,
		MouseSpeed:  2500,
		PositionDiff: 50,
		TimeFromStart: 300,
		TraceData: []model.TracePoint{
			{Timestamp: 0},
		},
	}

	result, err := service.EvaluateRisk(context.Background(), ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.RiskScore, float64(40))
	assert.True(t, result.RecommendVerify || result.RiskScore >= 40)
}

func TestRiskControlService_EvaluateRisk_TorUser(t *testing.T) {
	service := NewRiskControlService()

	ctx := &model.RiskContext{
		SessionID:   "test-session-tor",
		IPAddress:   "185.220.x.x",
		Fingerprint: "fp-tor-user-001",
		IsTor:       true,
		MouseSpeed:  100,
		PositionDiff: 0,
	}

	result, err := service.EvaluateRisk(context.Background(), ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, []model.RiskLevel{model.RiskLevelMedium, model.RiskLevelHigh, model.RiskLevelCritical}, result.RiskLevel)
}

func TestRiskControlService_EvaluateRisk_NewDevice(t *testing.T) {
	service := NewRiskControlService()

	ctx := &model.RiskContext{
		SessionID:   "test-session-new-dev",
		IPAddress:   "172.16.0.50",
		Fingerprint: "fp-brand-new-device",
		MouseSpeed:  200,
	}

	result, err := service.EvaluateRisk(context.Background(), ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.DeviceScore, float64(30))
}

func TestRiskControlService_EvaluateRisk_FastMouse(t *testing.T) {
	service := NewRiskControlService()

	ctx := &model.RiskContext{
		SessionID:   "test-session-fast-mouse",
		IPAddress:   "192.168.2.1",
		Fingerprint: "fp-fast-mouse-001",
		MouseSpeed:  3000,
		TimeFromStart: 200,
		PositionDiff: 30,
	}

	result, err := service.EvaluateRisk(context.Background(), ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.BehaviorScore, float64(30))
}

func TestRiskControlService_EvaluateRisk_BotLikeBehavior(t *testing.T) {
	service := NewRiskControlService()

	ctx := &model.RiskContext{
		SessionID:    "test-session-bot",
		IPAddress:    "10.10.10.10",
		Fingerprint:  "fp-bot-like-001",
		MouseSpeed:   2500,
		TimeFromStart: 100,
		PositionDiff: 100,
		TraceData:    []model.TracePoint{},
	}

	result, err := service.EvaluateRisk(context.Background(), ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.BehaviorScore, float64(50))
}

func TestRiskControlService_UpdateRules(t *testing.T) {
	service := NewRiskControlService()

	newRules := []RiskRuleConfig{
		{
			ID:           "custom_rule_1",
			Name:         "自定义规则1",
			Dimension:    "device",
			ScoreImpact:  20,
			Action:       ActionChallenge,
			Enabled:      true,
			Priority:     1,
			Description:  "测试自定义规则",
		},
	}

	err := service.UpdateRules(newRules)
	assert.NoError(t, err)

	rules := service.GetRules()
	assert.Len(t, rules, 1)
	assert.Equal(t, "custom_rule_1", rules[0].ID)
}

func TestRiskControlService_UpdateRules_InvalidScoreImpact(t *testing.T) {
	service := NewRiskControlService()

	invalidRules := []RiskRuleConfig{
		{
			ID:          "invalid_rule",
			Name:        "无效规则",
			Dimension:   "device",
			ScoreImpact: 100,
			Action:      ActionBlock,
			Enabled:     true,
		},
	}

	err := service.UpdateRules(invalidRules)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "score impact")
}

func TestRiskControlService_UpdateRules_EmptyID(t *testing.T) {
	service := NewRiskControlService()

	invalidRules := []RiskRuleConfig{
		{
			ID:          "",
			Name:        "空ID规则",
			Dimension:   "device",
			ScoreImpact: 20,
			Action:      ActionBlock,
			Enabled:     true,
		},
	}

	err := service.UpdateRules(invalidRules)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ID")
}

func TestRiskControlService_UpdateWeights(t *testing.T) {
	service := NewRiskControlService()

	weights := RiskWeights{
		DeviceWeight:   0.30,
		IPWeight:       0.25,
		BehaviorWeight: 0.30,
		EnvWeight:      0.15,
	}

	err := service.UpdateWeights(weights)
	assert.NoError(t, err)
}

func TestRiskControlService_UpdateWeights_InvalidSum(t *testing.T) {
	service := NewRiskControlService()

	weights := RiskWeights{
		DeviceWeight:   0.5,
		IPWeight:       0.5,
		BehaviorWeight: 0.5,
		EnvWeight:      0.5,
	}

	err := service.UpdateWeights(weights)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sum to 1")
}

func TestRiskControlService_UpdateWeights_InvalidDeviceWeight(t *testing.T) {
	service := NewRiskControlService()

	weights := RiskWeights{
		DeviceWeight:   1.5,
		IPWeight:       0.2,
		BehaviorWeight: 0.2,
		EnvWeight:      0.2,
	}

	err := service.UpdateWeights(weights)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "device weight")
}

func TestRiskControlService_GetStatistics(t *testing.T) {
	service := NewRiskControlService()

	stats, err := service.GetStatistics()

	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.NotNil(t, stats.RiskLevelStats)
}

func TestRiskControlService_GetDeviceProfile(t *testing.T) {
	service := NewRiskControlService()

	deviceID := "fp-profile-test-device"

	ctx := &model.RiskContext{
		SessionID:   "profile-test-session",
		IPAddress:   "192.168.200.1",
		Fingerprint: deviceID,
	}

	service.EvaluateRisk(context.Background(), ctx)

	profile, err := service.GetDeviceProfile(deviceID)

	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, deviceID, profile.DeviceID)
}

func TestRiskControlService_GetDeviceProfile_NotFound(t *testing.T) {
	service := NewRiskControlService()

	profile, err := service.GetDeviceProfile("non-existent-device")

	assert.Error(t, err)
	assert.Nil(t, profile)
}

func TestRiskControlService_GetIPProfile(t *testing.T) {
	service := NewRiskControlService()

	ipAddress := "192.168.150.100"

	ctx := &model.RiskContext{
		SessionID:   "ip-profile-test",
		IPAddress:   ipAddress,
		Fingerprint: "fp-ip-profile-001",
	}

	service.EvaluateRisk(context.Background(), ctx)

	profile, err := service.GetIPProfile(ipAddress)

	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, ipAddress, profile.IPAddress)
}

func TestRiskControlService_GetIPProfile_NotFound(t *testing.T) {
	service := NewRiskControlService()

	profile, err := service.GetIPProfile("192.168.99.99")

	assert.Error(t, err)
	assert.Nil(t, profile)
}

func TestRiskControlService_ResetDeviceRisk(t *testing.T) {
	service := NewRiskControlService()

	deviceID := "fp-reset-device"

	ctx := &model.RiskContext{
		SessionID:   "reset-device-test",
		IPAddress:   "192.168.250.1",
		Fingerprint: deviceID,
	}

	service.EvaluateRisk(context.Background(), ctx)

	profile, err := service.GetDeviceProfile(deviceID)
	assert.NoError(t, err)
	assert.NotNil(t, profile)

	err = service.ResetDeviceRisk(deviceID)
	assert.NoError(t, err)

	_, err = service.GetDeviceProfile(deviceID)
	assert.Error(t, err)
}

func TestRiskControlService_ResetIPRisk(t *testing.T) {
	service := NewRiskControlService()

	ipAddress := "192.168.251.100"

	ctx := &model.RiskContext{
		SessionID:   "reset-ip-test",
		IPAddress:   ipAddress,
		Fingerprint: "fp-reset-ip-001",
	}

	service.EvaluateRisk(context.Background(), ctx)

	profile, err := service.GetIPProfile(ipAddress)
	assert.NoError(t, err)
	assert.NotNil(t, profile)

	err = service.ResetIPRisk(ipAddress)
	assert.NoError(t, err)

	_, err = service.GetIPProfile(ipAddress)
	assert.Error(t, err)
}

func TestRiskControlService_CleanupOldProfiles(t *testing.T) {
	service := NewRiskControlService()

	ctx := &model.RiskContext{
		SessionID:   "cleanup-test",
		IPAddress:   "192.168.252.1",
		Fingerprint: "fp-cleanup-device",
	}

	service.EvaluateRisk(context.Background(), ctx)

	assert.NotNil(t, service.deviceRiskCache["fp-cleanup-device"])
	assert.NotNil(t, service.ipRiskCache["192.168.252.1"])

	service.CleanupOldProfiles(0)

	assert.Nil(t, service.deviceRiskCache["fp-cleanup-device"])
	assert.Nil(t, service.ipRiskCache["192.168.252.1"])
}

func TestRiskControlService_RiskScoresInRange(t *testing.T) {
	service := NewRiskControlService()

	testCases := []struct {
		name     string
		ctx      *model.RiskContext
		minScore float64
		maxScore float64
	}{
		{
			name: "最低风险",
			ctx: &model.RiskContext{
				SessionID:     "range-test-1",
				IPAddress:     "192.168.1.1",
				Fingerprint:   "fp-range-001",
				MouseSpeed:    100,
				HasTouchDevice: true,
				Language:     "zh-CN",
				Timezone:     "Asia/Shanghai",
				PositionDiff: 2,
				TimeFromStart: 8000,
				TraceData: []model.TracePoint{
					{Timestamp: 0, X: 100, Y: 100},
					{Timestamp: 500, X: 200, Y: 200},
					{Timestamp: 1000, X: 300, Y: 300},
					{Timestamp: 1500, X: 400, Y: 400},
					{Timestamp: 2000, X: 500, Y: 500},
				},
			},
			minScore: 0,
			maxScore: 40,
		},
		{
			name: "最高风险",
			ctx: &model.RiskContext{
				SessionID:    "range-test-2",
				IPAddress:    "10.0.0.1",
				Fingerprint:  "fp-range-002",
				IsProxy:      true,
				IsVPN:        true,
				IsTor:        true,
				MouseSpeed:   3000,
				PositionDiff: 100,
				TimeFromStart: 100,
				TraceData:    []model.TracePoint{},
			},
			minScore: 50,
			maxScore: 100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.EvaluateRisk(context.Background(), tc.ctx)
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, result.RiskScore, tc.minScore)
			assert.LessOrEqual(t, result.RiskScore, tc.maxScore)
		})
	}
}

func TestRiskControlService_ActionDetermination(t *testing.T) {
	service := NewRiskControlService()

	testCases := []struct {
		name       string
		riskScore  float64
		expected   string
	}{
		{"低风险-放行", 20, ActionAllow},
		{"中风险-标记", 35, ActionFlag},
		{"中高风险-挑战", 55, ActionChallenge},
		{"高风险-阻止", 80, ActionBlock},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			action := service.determineAction(tc.riskScore, []string{})
			assert.Equal(t, tc.expected, action)
		})
	}
}

func TestRiskControlService_PositionRisk(t *testing.T) {
	service := NewRiskControlService()

	testCases := []struct {
		name         string
		positionDiff int
		minExpected  float64
		maxExpected  float64
	}{
		{"精准位置", 3, 0, 10},
		{"轻微偏差", 8, 15, 30},
		{"中度偏差", 15, 30, 50},
		{"严重偏差", 30, 50, 70},
		{"完全错误", 100, 75, 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &model.RiskContext{
				SessionID:    "position-test",
				IPAddress:   "192.168.10.1",
				Fingerprint: "fp-position-test",
				PositionDiff: tc.positionDiff,
			}

			result, err := service.EvaluateRisk(context.Background(), ctx)
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, result.PositionScore, tc.minExpected)
			assert.LessOrEqual(t, result.PositionScore, tc.maxExpected)
		})
	}
}

func TestRiskControlConstants(t *testing.T) {
	assert.Equal(t, "allow", ActionAllow)
	assert.Equal(t, "challenge", ActionChallenge)
	assert.Equal(t, "block", ActionBlock)
	assert.Equal(t, "flag", ActionFlag)
}
