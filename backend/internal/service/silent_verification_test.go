package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSilentVerificationService(t *testing.T) {
	service := NewSilentVerificationService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.behaviorService)
}

func TestSilentVerificationProcessVerification(t *testing.T) {
	service := NewSilentVerificationService()

	req := &SilentVerifyRequest{
		DeviceFingerprint: "test-fingerprint-123",
		SessionID:       "test-session-456",
		BehaviorData: []BehaviorDataPoint{
			{X: 100, Y: 200, Timestamp: 1000, Event: "mousemove"},
			{X: 110, Y: 210, Timestamp: 1100, Event: "mousemove"},
			{X: 120, Y: 220, Timestamp: 1200, Event: "click"},
		},
		Timestamp: 1000,
		UserID:    1,
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
	}

	resp, err := service.ProcessVerification(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)
	assert.Contains(t, []string{"pass", "challenge", "block"}, resp.RiskLevel)
	assert.Contains(t, []string{"pass", "challenge", "block"}, resp.RiskLevel)
}

func TestSilentVerificationProcessVerificationEmptyData(t *testing.T) {
	service := NewSilentVerificationService()

	req := &SilentVerifyRequest{
		DeviceFingerprint: "test-fingerprint",
		SessionID:       "test-session",
		BehaviorData:     []BehaviorDataPoint{},
		Timestamp:       1000,
		UserID:          1,
		IPAddress:      "192.168.1.1",
		UserAgent:      "Mozilla/5.0",
	}

	resp, err := service.ProcessVerification(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestSilentVerificationEvaluateDeviceTrust(t *testing.T) {
	service := NewSilentVerificationService()

	req := &SilentVerifyRequest{
		DeviceFingerprint: "test-fingerprint",
		UserID:           1,
		IPAddress:       "192.168.1.1",
	}

	score := service.evaluateDeviceTrust(req)
	assert.NotNil(t, score)
	assert.GreaterOrEqual(t, score.FingerprintMatch, 0.0)
	assert.LessOrEqual(t, score.FingerprintMatch, 100.0)
	assert.GreaterOrEqual(t, score.TotalScore, 0.0)
	assert.LessOrEqual(t, score.TotalScore, 100.0)
}

func TestSilentVerificationCalculateFingerprintMatch(t *testing.T) {
	service := NewSilentVerificationService()

	tests := []struct {
		name        string
		fingerprint string
		minScore   float64
		maxScore   float64
	}{
		{
			name:        "empty fingerprint",
			fingerprint: "",
			minScore:   0,
			maxScore:   0,
		},
		{
			name:        "short fingerprint",
			fingerprint: "abc",
			minScore:   0,
			maxScore:   100,
		},
		{
			name:        "normal fingerprint",
			fingerprint: "this-is-a-normal-length-fingerprint",
			minScore:   50,
			maxScore:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.calculateFingerprintMatch(tt.fingerprint)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestSilentVerificationEvaluateBehaviorTrust(t *testing.T) {
	service := NewSilentVerificationService()

	tests := []struct {
		name        string
		behaviorData []BehaviorDataPoint
	}{
		{
			name:        "empty data",
			behaviorData: []BehaviorDataPoint{},
		},
		{
			name: "normal mouse movement",
			behaviorData: []BehaviorDataPoint{
				{X: 100, Y: 200, Timestamp: 1000, Event: "mousemove"},
				{X: 110, Y: 210, Timestamp: 1050, Event: "mousemove"},
				{X: 120, Y: 220, Timestamp: 1100, Event: "mousemove"},
			},
		},
		{
			name: "with clicks",
			behaviorData: []BehaviorDataPoint{
				{X: 100, Y: 200, Timestamp: 1000, Event: "mousemove"},
				{X: 100, Y: 200, Timestamp: 1100, Event: "click"},
				{X: 200, Y: 300, Timestamp: 1200, Event: "mousemove"},
				{X: 200, Y: 300, Timestamp: 1300, Event: "click"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.evaluateBehaviorTrust(tt.behaviorData)
			assert.NotNil(t, score)
			assert.GreaterOrEqual(t, score.TotalScore, 0.0)
			assert.LessOrEqual(t, score.TotalScore, 100.0)
		})
	}
}

func TestSilentVerificationAnalyzeMouseTrajectoryTrust(t *testing.T) {
	service := NewSilentVerificationService()

	tests := []struct {
		name        string
		data       []BehaviorDataPoint
		minScore   float64
		maxScore   float64
	}{
		{
			name:        "insufficient points",
			data:       []BehaviorDataPoint{{X: 100, Y: 200, Timestamp: 1000, Event: "move"}},
			minScore:   0,
			maxScore:   100,
		},
		{
			name: "normal movement",
			data: []BehaviorDataPoint{
				{X: 100, Y: 200, Timestamp: 1000, Event: "mousemove"},
				{X: 110, Y: 210, Timestamp: 1050, Event: "mousemove"},
				{X: 120, Y: 220, Timestamp: 1100, Event: "mousemove"},
				{X: 130, Y: 230, Timestamp: 1150, Event: "mousemove"},
				{X: 140, Y: 240, Timestamp: 1200, Event: "mousemove"},
				{X: 150, Y: 250, Timestamp: 1250, Event: "mousemove"},
				{X: 160, Y: 260, Timestamp: 1300, Event: "mousemove"},
				{X: 170, Y: 270, Timestamp: 1350, Event: "mousemove"},
				{X: 180, Y: 280, Timestamp: 1400, Event: "mousemove"},
				{X: 190, Y: 290, Timestamp: 1450, Event: "mousemove"},
			},
			minScore:   0,
			maxScore:   100,
		},
		{
			name: "too fast movement",
			data: []BehaviorDataPoint{
				{X: 0, Y: 0, Timestamp: 0, Event: "mousemove"},
				{X: 10000, Y: 10000, Timestamp: 1, Event: "mousemove"},
				{X: 20000, Y: 20000, Timestamp: 2, Event: "mousemove"},
				{X: 30000, Y: 30000, Timestamp: 3, Event: "mousemove"},
				{X: 40000, Y: 40000, Timestamp: 4, Event: "mousemove"},
				{X: 50000, Y: 50000, Timestamp: 5, Event: "mousemove"},
				{X: 60000, Y: 60000, Timestamp: 6, Event: "mousemove"},
				{X: 70000, Y: 70000, Timestamp: 7, Event: "mousemove"},
				{X: 80000, Y: 80000, Timestamp: 8, Event: "mousemove"},
				{X: 90000, Y: 90000, Timestamp: 9, Event: "mousemove"},
			},
			minScore:   0,
			maxScore:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.analyzeMouseTrajectoryTrust(tt.data)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestSilentVerificationAnalyzeClickPatternTrust(t *testing.T) {
	service := NewSilentVerificationService()

	tests := []struct {
		name      string
		data     []BehaviorDataPoint
		minScore float64
		maxScore float64
	}{
		{
			name:      "no clicks",
			data:     []BehaviorDataPoint{},
			minScore: 40,
			maxScore: 60,
		},
		{
			name: "single click",
			data: []BehaviorDataPoint{
				{X: 100, Y: 200, Timestamp: 1000, Event: "click"},
			},
			minScore: 0,
			maxScore: 100,
		},
		{
			name: "regular clicks",
			data: []BehaviorDataPoint{
				{X: 100, Y: 200, Timestamp: 1000, Event: "click"},
				{X: 200, Y: 300, Timestamp: 1100, Event: "click"},
				{X: 300, Y: 400, Timestamp: 1200, Event: "click"},
				{X: 400, Y: 500, Timestamp: 1300, Event: "click"},
			},
			minScore: 0,
			maxScore: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.analyzeClickPatternTrust(tt.data)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestSilentVerificationAnalyzeKeyboardPatternTrust(t *testing.T) {
	service := NewSilentVerificationService()

	tests := []struct {
		name      string
		data     []BehaviorDataPoint
		minScore float64
		maxScore float64
	}{
		{
			name:      "no key events",
			data:     []BehaviorDataPoint{},
			minScore: 40,
			maxScore: 60,
		},
		{
			name: "few key events",
			data: []BehaviorDataPoint{
				{X: 0, Y: 0, Timestamp: 1000, Event: "keydown"},
				{X: 0, Y: 0, Timestamp: 1100, Event: "keyup"},
			},
			minScore: 0,
			maxScore: 100,
		},
		{
			name: "normal typing",
			data: []BehaviorDataPoint{
				{X: 0, Y: 0, Timestamp: 1000, Event: "keydown"},
				{X: 0, Y: 0, Timestamp: 1150, Event: "keydown"},
				{X: 0, Y: 0, Timestamp: 1300, Event: "keydown"},
				{X: 0, Y: 0, Timestamp: 1450, Event: "keydown"},
				{X: 0, Y: 0, Timestamp: 1600, Event: "keydown"},
				{X: 0, Y: 0, Timestamp: 1750, Event: "keydown"},
			},
			minScore: 0,
			maxScore: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.analyzeKeyboardPatternTrust(tt.data)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestSilentVerificationAnalyzeScrollBehaviorTrust(t *testing.T) {
	service := NewSilentVerificationService()

	tests := []struct {
		name      string
		data     []BehaviorDataPoint
		minScore float64
		maxScore float64
	}{
		{
			name:      "no scrolls",
			data:     []BehaviorDataPoint{},
			minScore: 40,
			maxScore: 60,
		},
		{
			name: "few scrolls",
			data: []BehaviorDataPoint{
				{X: 0, Y: 0, Timestamp: 1000, Event: "scroll"},
				{X: 0, Y: 0, Timestamp: 1100, Event: "scroll"},
			},
			minScore: 0,
			maxScore: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.analyzeScrollBehaviorTrust(tt.data)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestSilentVerificationAnalyzeTouchBehaviorTrust(t *testing.T) {
	service := NewSilentVerificationService()

	tests := []struct {
		name      string
		data     []BehaviorDataPoint
		minScore float64
		maxScore float64
	}{
		{
			name:      "no touch",
			data:     []BehaviorDataPoint{},
			minScore: 40,
			maxScore: 60,
		},
		{
			name: "with touch",
			data: []BehaviorDataPoint{
				{X: 100, Y: 200, Timestamp: 1000, Event: "touchstart"},
				{X: 110, Y: 210, Timestamp: 1100, Event: "touchmove"},
				{X: 120, Y: 220, Timestamp: 1200, Event: "touchend"},
			},
			minScore: 0,
			maxScore: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.analyzeTouchBehaviorTrust(tt.data)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestSilentVerificationCalculateCompositeScore(t *testing.T) {
	service := NewSilentVerificationService()

	deviceScore := &DeviceTrustScore{
		TotalScore: 70,
	}
	behaviorScore := &BehaviorTrustScore{
		TotalScore: 60,
	}
	historyScore := &HistoryTrustScore{
		TotalScore: 80,
	}

	composite := service.calculateCompositeScore(deviceScore, behaviorScore, historyScore)
	assert.GreaterOrEqual(t, composite, 0.0)
	assert.LessOrEqual(t, composite, 100.0)
}

func TestSilentVerificationDetermineStrategy(t *testing.T) {
	service := NewSilentVerificationService()

	tests := []struct {
		name           string
		riskScore     float64
		deviceScore   *DeviceTrustScore
		behaviorScore *BehaviorTrustScore
		historyScore  *HistoryTrustScore
	}{
		{
			name:       "very low risk",
			riskScore:  10,
			deviceScore: &DeviceTrustScore{TotalScore: 90},
			behaviorScore: &BehaviorTrustScore{TotalScore: 85},
			historyScore: &HistoryTrustScore{TotalScore: 90},
		},
		{
			name:       "high risk",
			riskScore:  80,
			deviceScore: &DeviceTrustScore{TotalScore: 30},
			behaviorScore: &BehaviorTrustScore{TotalScore: 40},
			historyScore: &HistoryTrustScore{TotalScore: 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := service.determineStrategy(tt.riskScore, tt.deviceScore, tt.behaviorScore, tt.historyScore)
			assert.NotNil(t, strategy)
			assert.Contains(t, []string{"pass", "challenge", "block"}, strategy.Level)
		})
	}
}

func TestSilentVerificationMatchConditions(t *testing.T) {
	service := NewSilentVerificationService()

	deviceScore := &DeviceTrustScore{TotalScore: 70}
	behaviorScore := &BehaviorTrustScore{TotalScore: 60}
	historyScore := &HistoryTrustScore{TotalScore: 80}

	tests := []struct {
		name       string
		conditions []RuleCondition
		expected   bool
	}{
		{
			name: "match >=",
			conditions: []RuleCondition{
				{Field: "device_score", Operator: ">=", Value: float64(60)},
			},
			expected: true,
		},
		{
			name: "no match >=",
			conditions: []RuleCondition{
				{Field: "device_score", Operator: ">=", Value: float64(80)},
			},
			expected: false,
		},
		{
			name: "match <",
			conditions: []RuleCondition{
				{Field: "behavior_score", Operator: "<", Value: float64(70)},
			},
			expected: true,
		},
		{
			name: "no match <",
			conditions: []RuleCondition{
				{Field: "behavior_score", Operator: "<", Value: float64(50)},
			},
			expected: false,
		},
		{
			name: "multiple conditions all match",
			conditions: []RuleCondition{
				{Field: "device_score", Operator: ">=", Value: float64(60)},
				{Field: "behavior_score", Operator: ">=", Value: float64(50)},
			},
			expected: true,
		},
		{
			name: "multiple conditions one fails",
			conditions: []RuleCondition{
				{Field: "device_score", Operator: ">=", Value: float64(80)},
				{Field: "behavior_score", Operator: ">=", Value: float64(50)},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.matchConditions(tt.conditions, deviceScore, behaviorScore, historyScore)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSilentVerificationGenerateToken(t *testing.T) {
	service := NewSilentVerificationService()

	req := &SilentVerifyRequest{
		DeviceFingerprint: "test-fingerprint",
		SessionID:       "test-session",
		UserID:          1,
	}

	token1 := service.generateToken(req)
	token2 := service.generateToken(req)

	assert.NotEmpty(t, token1)
	assert.NotEqual(t, token1, token2)
}

func TestSilentVerificationGetConfig(t *testing.T) {
	service := NewSilentVerificationService()

	config := service.GetConfig()
	assert.NotNil(t, config)
	assert.True(t, config.Enabled)
	assert.Greater(t, config.RiskThreshold, 0.0)
	assert.Greater(t, config.MinBehaviorDataPoints, 0)
}

func TestSilentVerificationUpdateConfig(t *testing.T) {
	service := NewSilentVerificationService()

	newConfig := &SilentVerificationConfig{
		Enabled:              true,
		RiskThreshold:        40.0,
		MinBehaviorDataPoints: 30,
		MaxVerifyDuration:   400,
		EnableDeviceCheck:   true,
		EnableBehaviorCheck: true,
		EnableHistoryCheck:  true,
		CacheTTL:            800,
	}

	service.UpdateConfig(newConfig)

	updatedConfig := service.GetConfig()
	assert.Equal(t, newConfig.RiskThreshold, updatedConfig.RiskThreshold)
	assert.Equal(t, newConfig.MinBehaviorDataPoints, updatedConfig.MinBehaviorDataPoints)
}

func TestSilentVerificationGetStrategyRules(t *testing.T) {
	service := NewSilentVerificationService()

	rules := service.GetStrategyRules()
	assert.NotNil(t, rules)
	assert.Greater(t, len(rules), 0)
}

func TestSilentVerificationDegradeToNormalVerification(t *testing.T) {
	service := NewSilentVerificationService()

	strategy := service.DegradeToNormalVerification()

	assert.NotNil(t, strategy)
	assert.Equal(t, "challenge", strategy.Level)
	assert.True(t, strategy.NeedCaptcha)
	assert.Equal(t, "slider", strategy.CaptchaType)
}

func TestSilentVerificationGenerateRiskReasons(t *testing.T) {
	service := NewSilentVerificationService()

	deviceScore := &DeviceTrustScore{TotalScore: 30, FingerprintMatch: 40}
	behaviorScore := &BehaviorTrustScore{TotalScore: 20, MouseTrajectoryScore: 30}
	historyScore := &HistoryTrustScore{TotalScore: 30}

	reasons := service.generateRiskReasons(deviceScore, behaviorScore, historyScore)
	assert.NotNil(t, reasons)
	assert.Greater(t, len(reasons), 0)
}

func TestSilentVerificationGenerateSuggestions(t *testing.T) {
	service := NewSilentVerificationService()

	tests := []struct {
		name   string
		action StrategyAction
	}{
		{
			name: "pass action",
			action: StrategyAction{
				Level:       "pass",
				NeedCaptcha: false,
				CaptchaType: "none",
			},
		},
		{
			name: "challenge with slider",
			action: StrategyAction{
				Level:       "challenge",
				NeedCaptcha: true,
				CaptchaType: "slider",
			},
		},
		{
			name: "challenge with click",
			action: StrategyAction{
				Level:       "challenge",
				NeedCaptcha: true,
				CaptchaType: "click",
			},
		},
		{
			name: "block action",
			action: StrategyAction{
				Level:       "block",
				NeedCaptcha: true,
				CaptchaType: "click",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := service.generateSuggestions(tt.action)
			assert.NotNil(t, suggestions)
			assert.Greater(t, len(suggestions), 0)
		})
	}
}

func TestSilentVerificationGetMessageForStrategy(t *testing.T) {
	service := NewSilentVerificationService()

	tests := []struct {
		level    string
		expected string
	}{
		{"pass", "验证通过"},
		{"challenge", "请完成验证"},
		{"block", "验证被拦截"},
		{"unknown", "验证处理中"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			strategy := &VerificationStrategy{Level: tt.level}
			message := service.getMessageForStrategy(strategy)
			assert.Equal(t, tt.expected, message)
		})
	}
}

func TestIPParsing(t *testing.T) {
	service := NewSilentVerificationService()

	tests := []struct {
		ip       string
		expected []int
	}{
		{"192.168.1.1", []int{192, 168, 1, 1}},
		{"10.0.0.1", []int{10, 0, 0, 1}},
		{"255.255.255.255", []int{255, 255, 255, 255}},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			parts := service.parseIP(tt.ip)
			assert.Equal(t, tt.expected, parts)
		})
	}
}

func TestIPSameRegion(t *testing.T) {
	service := NewSilentVerificationService()

	tests := []struct {
		name     string
		ip1      string
		ip2      string
		expected bool
	}{
		{
			name:     "same IP",
			ip1:      "192.168.1.1",
			ip2:      "192.168.1.1",
			expected: true,
		},
		{
			name:     "same region",
			ip1:      "192.168.1.1",
			ip2:      "192.168.2.1",
			expected: true,
		},
		{
			name:     "different region",
			ip1:      "192.168.1.1",
			ip2:      "10.0.0.1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ipsInSameRegion(tt.ip1, tt.ip2)
			assert.Equal(t, tt.expected, result)
		})
	}
}
