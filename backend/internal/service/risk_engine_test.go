package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDeepRiskEngineStateExtraction(t *testing.T) {
	engine := NewDeepRiskEngine()
	ctx := context.Background()

	state := engine.ExtractState(ctx, "test_fingerprint", "192.168.1.1", 85.0, 90.0, 95.0, map[string]interface{}{
		"fail_count":     0,
		"request_count":  10,
	})

	assert.NotNil(t, state)
	assert.GreaterOrEqual(t, state.DeviceScore, 0.0)
	assert.LessOrEqual(t, state.DeviceScore, 100.0)
	assert.GreaterOrEqual(t, state.IPScore, 0.0)
	assert.LessOrEqual(t, state.IPScore, 100.0)
	assert.GreaterOrEqual(t, state.BehaviorScore, 0.0)
	assert.LessOrEqual(t, state.BehaviorScore, 100.0)
}

func TestDeepRiskEngineStateToVector(t *testing.T) {
	engine := NewDeepRiskEngine()

	state := &RiskState{
		DeviceScore:     80.0,
		IPScore:         85.0,
		BehaviorScore:   90.0,
		GeoScore:        95.0,
		HistoricalScore: 88.0,
		TimeScore:       100.0,
		SessionScore:    92.0,
	}

	vector := engine.StateToVector(state)

	assert.Equal(t, 7, len(vector))
	assert.InDelta(t, 0.8, vector[0], 0.001)
	assert.InDelta(t, 0.85, vector[1], 0.001)
	assert.InDelta(t, 0.9, vector[2], 0.001)
}

func TestDeepRiskEngineSelectAction(t *testing.T) {
	engine := NewDeepRiskEngine()

	testCases := []struct {
		name           string
		state          *RiskState
		expectedActions []RiskAction
	}{
		{
			name: "低风险状态",
			state: &RiskState{
				DeviceScore:     90.0,
				IPScore:         85.0,
				BehaviorScore:   88.0,
				GeoScore:        92.0,
				HistoricalScore: 90.0,
				TimeScore:       100.0,
				SessionScore:    95.0,
			},
			expectedActions: []RiskAction{ActionAllow},
		},
		{
			name: "中风险状态",
			state: &RiskState{
				DeviceScore:     60.0,
				IPScore:         65.0,
				BehaviorScore:   70.0,
				GeoScore:        68.0,
				HistoricalScore: 72.0,
				TimeScore:       80.0,
				SessionScore:    75.0,
			},
			expectedActions: []RiskAction{ActionCaptcha, ActionReview},
		},
		{
			name: "高风险状态",
			state: &RiskState{
				DeviceScore:     30.0,
				IPScore:         35.0,
				BehaviorScore:   40.0,
				GeoScore:        32.0,
				HistoricalScore: 38.0,
				TimeScore:       50.0,
				SessionScore:    45.0,
			},
			expectedActions: []RiskAction{ActionBlock, ActionChallenge},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			action := engine.SelectAction(tc.state)
			assert.Contains(t, tc.expectedActions, action)
		})
	}
}

func TestDeepRiskEngineCalculateReward(t *testing.T) {
	engine := NewDeepRiskEngine()

	testCases := []struct {
		name          string
		state         *RiskState
		action        RiskAction
		outcome       string
		isHuman       bool
		minReward     float64
		maxReward     float64
	}{
		{
			name:    "允许正常用户",
			state:   &RiskState{DeviceScore: 90, IPScore: 85, BehaviorScore: 88},
			action:  ActionAllow,
			outcome: "success",
			isHuman: true,
			minReward: 5.0,
			maxReward: 15.0,
		},
		{
			name:    "阻止攻击",
			state:   &RiskState{DeviceScore: 20, IPScore: 25, BehaviorScore: 30},
			action:  ActionBlock,
			outcome: "blocked_attack",
			isHuman: false,
			minReward: 10.0,
			maxReward: 20.0,
		},
		{
			name:    "误拦正常用户",
			state:   &RiskState{DeviceScore: 85, IPScore: 90, BehaviorScore: 88},
			action:  ActionBlock,
			outcome: "blocked_human",
			isHuman: true,
			minReward: -30.0,
			maxReward: -20.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reward := engine.CalculateReward(tc.state, tc.action, tc.outcome, tc.isHuman)
			assert.GreaterOrEqual(t, reward, tc.minReward)
			assert.LessOrEqual(t, reward, tc.maxReward)
		})
	}
}

func TestDeepRiskEngineStoreExperience(t *testing.T) {
	engine := NewDeepRiskEngine()

	state := &RiskState{
		DeviceScore:     80.0,
		IPScore:         85.0,
		BehaviorScore:   90.0,
		GeoScore:        95.0,
		HistoricalScore: 88.0,
		TimeScore:       100.0,
		SessionScore:    92.0,
	}

	nextState := &RiskState{
		DeviceScore:     75.0,
		IPScore:         80.0,
		BehaviorScore:   85.0,
		GeoScore:        90.0,
		HistoricalScore: 83.0,
		TimeScore:       95.0,
		SessionScore:    87.0,
	}

	engine.StoreExperience(state, ActionCaptcha, 5.0, nextState, false)

	assert.Equal(t, 1, len(engine.replayBuffer))
	assert.Equal(t, "captcha", engine.replayBuffer[0].Action)
	assert.Equal(t, 5.0, engine.replayBuffer[0].Reward)
}

func TestDeepRiskEngineSampleExperiences(t *testing.T) {
	engine := NewDeepRiskEngine()

	for i := 0; i < 50; i++ {
		state := &RiskState{
			DeviceScore: float64(70 + i%30),
			IPScore:     float64(75 + i%25),
			BehaviorScore: float64(80 + i%20),
			GeoScore:    float64(85 + i%15),
			HistoricalScore: float64(78 + i%22),
			TimeScore:   90.0,
			SessionScore: float64(82 + i%18),
		}
		nextState := &RiskState{
			DeviceScore: float64(65 + i%30),
			IPScore:     float64(70 + i%25),
			BehaviorScore: float64(75 + i%20),
			GeoScore:    float64(80 + i%15),
			HistoricalScore: float64(73 + i%22),
			TimeScore:   85.0,
			SessionScore: float64(77 + i%18),
		}
		engine.StoreExperience(state, RiskAction("allow"), float64(i), nextState, false)
	}

	samples := engine.SampleExperiences(10)
	assert.Equal(t, 10, len(samples))
}

func TestRiskProfileServiceDeviceProfile(t *testing.T) {
	profileService := NewRiskProfileService()
	ctx := context.Background()

	deviceData := map[string]interface{}{
		"user_agent":            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
		"screen_resolution":     "1920x1080",
		"color_depth":           24,
		"timezone":             "Asia/Shanghai",
		"language":             "zh-CN",
		"platform":             "Win32",
		"hardware_concurrency": 8,
		"device_memory":        8.0,
		"touch_points":          0,
		"is_mobile":            false,
	}

	profile, err := profileService.CreateOrUpdateDeviceProfile(ctx, "test_fp_001", deviceData)

	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, "test_fp_001", profile.Fingerprint)
	assert.Greater(t, profile.RequestCount, int64(0))
	assert.LessOrEqual(t, profile.RiskScore, 100.0)
}

func TestRiskProfileServiceBehaviorProfile(t *testing.T) {
	profileService := NewRiskProfileService()

	behaviorData := map[string]interface{}{
		"fingerprint":       "test_fp_002",
		"ip_address":        "192.168.1.100",
		"mouse_speed":      450.0,
		"click_frequency":   3.5,
		"path_efficiency":  0.75,
		"straightness":     0.65,
		"total_clicks":     5,
		"total_moves":      50,
		"trajectory_points": 30,
	}

	profile, err := profileService.CreateOrUpdateBehaviorProfile("session_002", behaviorData)

	assert.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, "session_002", profile.SessionID)
	assert.True(t, profile.IsHuman)
	assert.Less(t, profile.RiskScore, 100.0)
}

func TestHotUpdateServiceGetCurrentVersion(t *testing.T) {
	hotUpdateService := NewHotUpdateService()

	version := hotUpdateService.GetCurrentVersion()

	assert.NotNil(t, version)
	assert.NotEmpty(t, version.Version)
}

func TestHotUpdateServiceGetAllRules(t *testing.T) {
	hotUpdateService := NewHotUpdateService()

	rules := hotUpdateService.GetAllRules()

	assert.NotNil(t, rules)
	assert.GreaterOrEqual(t, len(rules), 0)
}

func TestHotUpdateServiceEvaluateRules(t *testing.T) {
	hotUpdateService := NewHotUpdateService()
	ctx := context.Background()

	riskContext := map[string]interface{}{
		"ip_request_count": 150.0,
		"mouse_speed":      2500.0,
		"path_efficiency": 0.98,
		"is_vpn":          true,
	}

	action, riskScore, triggeredRules := hotUpdateService.EvaluateRules(ctx, riskContext)

	assert.NotEmpty(t, action)
	assert.GreaterOrEqual(t, riskScore, 0.0)
	assert.NotNil(t, triggeredRules)
}

func TestMonitoringServiceRecordMetric(t *testing.T) {
	monitoringService := NewMonitoringService()
	ctx := context.Background()

	err := monitoringService.RecordMetric(ctx, "test", "test_metric", 85.5, "dimension", "value", "percent", map[string]interface{}{
		"source": "unit_test",
	})

	assert.NoError(t, err)
}

func TestMonitoringServiceGetRiskMetrics(t *testing.T) {
	monitoringService := NewMonitoringService()
	ctx := context.Background()

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	metrics, err := monitoringService.GetRiskMetrics(ctx, startTime, endTime)

	assert.NoError(t, err)
	assert.NotNil(t, metrics)
}

func TestMonitoringServiceGetActiveAlerts(t *testing.T) {
	monitoringService := NewMonitoringService()
	ctx := context.Background()

	alerts, err := monitoringService.GetActiveAlerts(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, alerts)
}

func TestMonitoringServiceCreateAlert(t *testing.T) {
	monitoringService := NewMonitoringService()
	ctx := context.Background()

	alert, err := monitoringService.CreateAlert(
		ctx,
		"test",
		"测试告警",
		"medium",
		"这是一个测试告警",
		"test_metric",
		50.0,
		"<",
	)

	assert.NoError(t, err)
	assert.NotNil(t, alert)
	assert.Equal(t, "test", alert.AlertType)
	assert.Equal(t, "medium", alert.Severity)
}

func TestRiskStateNormalization(t *testing.T) {
	testCases := []struct {
		name     string
		input    *RiskState
		minValue float64
		maxValue float64
	}{
		{
			name: "正常范围",
			input: &RiskState{
				DeviceScore:     80.0,
				IPScore:         85.0,
				BehaviorScore:   90.0,
				GeoScore:        95.0,
				HistoricalScore: 88.0,
				TimeScore:       100.0,
				SessionScore:    92.0,
			},
			minValue: 80.0,
			maxValue: 100.0,
		},
		{
			name: "低于范围",
			input: &RiskState{
				DeviceScore:     -10.0,
				IPScore:         -5.0,
				BehaviorScore:   0.0,
				GeoScore:        -15.0,
				HistoricalScore: -8.0,
				TimeScore:       -20.0,
				SessionScore:    -12.0,
			},
			minValue: 0.0,
			maxValue: 0.0,
		},
		{
			name: "超过范围",
			input: &RiskState{
				DeviceScore:     150.0,
				IPScore:         120.0,
				BehaviorScore:   110.0,
				GeoScore:        130.0,
				HistoricalScore: 115.0,
				TimeScore:       140.0,
				SessionScore:    125.0,
			},
			minValue: 100.0,
			maxValue: 100.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			normalized := normalizeRiskState(tc.input)

			assert.GreaterOrEqual(t, normalized.DeviceScore, tc.minValue)
			assert.LessOrEqual(t, normalized.DeviceScore, tc.maxValue)
			assert.GreaterOrEqual(t, normalized.IPScore, tc.minValue)
			assert.LessOrEqual(t, normalized.IPScore, tc.maxValue)
		})
	}
}

func normalizeRiskState(state *RiskState) *RiskState {
	state.DeviceScore = clamp(state.DeviceScore, 0, 100)
	state.IPScore = clamp(state.IPScore, 0, 100)
	state.BehaviorScore = clamp(state.BehaviorScore, 0, 100)
	state.GeoScore = clamp(state.GeoScore, 0, 100)
	state.HistoricalScore = clamp(state.HistoricalScore, 0, 100)
	state.TimeScore = clamp(state.TimeScore, 0, 100)
	state.SessionScore = clamp(state.SessionScore, 0, 100)
	return state
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func TestRiskActionDetermination(t *testing.T) {
	testCases := []struct {
		name           string
		riskScore      float64
		expectedAction string
	}{
		{"极低风险", 85.0, "allow"},
		{"低风险", 75.0, "allow"},
		{"中低风险", 65.0, "captcha"},
		{"中等风险", 55.0, "captcha"},
		{"中高风险", 45.0, "review"},
		{"高风险", 35.0, "block"},
		{"极高风险", 15.0, "challenge"},
		{"临界风险", 20.0, "block"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			action := determineAction(tc.riskScore)
			assert.Equal(t, tc.expectedAction, action)
		})
	}
}

func determineAction(riskScore float64) string {
	switch {
	case riskScore >= 80:
		return "allow"
	case riskScore >= 60:
		return "captcha"
	case riskScore >= 40:
		return "review"
	case riskScore >= 20:
		return "block"
	default:
		return "challenge"
	}
}

func TestIPScoreCalculation(t *testing.T) {
	engine := NewDeepRiskEngine()

	testCases := []struct {
		name      string
		ipAddress string
	}{
		{"内网IP", "192.168.1.1"},
		{"本地IP", "127.0.0.1"},
		{"公网IP", "8.8.8.8"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := engine.calculateIPScore(tc.ipAddress)
			assert.GreaterOrEqual(t, score, 0.0)
			assert.LessOrEqual(t, score, 100.0)
		})
	}
}

func TestDeviceScoreCalculation(t *testing.T) {
	engine := NewDeepRiskEngine()

	testCases := []struct {
		name        string
		fingerprint string
		maxPenalty  float64
	}{
		{"正常设备", "normal_browser_Chrome", 0},
		{"无头浏览器", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome", 40},
		{"Selenium", "Mozilla/5.0 Chrome/91.0 selenium automation", 45},
		{"PhantomJS", "Mozilla/5.0 PhantomJS", 50},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := engine.calculateDeviceScore(tc.fingerprint)
			expectedMin := 100.0 - tc.maxPenalty
			assert.GreaterOrEqual(t, score, expectedMin)
			assert.LessOrEqual(t, score, 100.0)
		})
	}
}
