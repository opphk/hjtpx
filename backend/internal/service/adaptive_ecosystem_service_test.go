package service

import (
	"context"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/backend/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestAdaptiveEcosystemService_GenerateCaptcha(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	req := &model.AdaptiveEcosystemRequest{
		UserID:       "test-user-123",
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
		Fingerprint:  "fp123abc",
		Context:      map[string]interface{}{},
	}

	resp, err := service.GenerateCaptcha(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.SessionID)
	assert.NotNil(t, resp.CaptchaConfig)
	assert.NotNil(t, resp.RiskAssessment)
	assert.Equal(t, int64(300), resp.ExpiresIn)
}

func TestAdaptiveEcosystemService_GenerateCaptcha_WithContext(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	req := &model.AdaptiveEcosystemRequest{
		UserID:       "test-user-456",
		IPAddress:    "192.168.1.2",
		Fingerprint:  "fp456xyz",
		Context: map[string]interface{}{
			"headless":   true,
			"automation": false,
		},
	}

	resp, err := service.GenerateCaptcha(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.RiskAssessment)
}

func TestAdaptiveEcosystemService_GenerateCaptcha_WithUserProfile(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	_, _ = service.GenerateCaptcha(&model.AdaptiveEcosystemRequest{
		UserID: "existing-user",
	})

	resp, err := service.GenerateCaptcha(&model.AdaptiveEcosystemRequest{
		UserID: "existing-user",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestAdaptiveEcosystemService_VerifyCaptcha_Success(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	genResp, _ := service.GenerateCaptcha(&model.AdaptiveEcosystemRequest{
		UserID: "verify-user-success",
	})

	req := &model.AdaptiveVerifyRequest{
		SessionID:    genResp.SessionID,
		UserID:       "verify-user-success",
		Answer:       "correct_answer",
		ResponseTime: 5000,
		BehaviorData: map[string]interface{}{},
	}

	resp, err := service.VerifyCaptcha(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.GreaterOrEqual(t, resp.Score, 0.0)
}

func TestAdaptiveEcosystemService_VerifyCaptcha_Failure(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	genResp, _ := service.GenerateCaptcha(&model.AdaptiveEcosystemRequest{
		UserID: "verify-user-fail",
	})

	req := &model.AdaptiveVerifyRequest{
		SessionID:    genResp.SessionID,
		UserID:       "verify-user-fail",
		Answer:       "wrong_answer",
		ResponseTime: 60000,
		BehaviorData: map[string]interface{}{},
	}

	resp, err := service.VerifyCaptcha(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestAdaptiveEcosystemService_VerifyCaptcha_NewUser(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	genResp, _ := service.GenerateCaptcha(&model.AdaptiveEcosystemRequest{
		UserID: "new-user-123",
	})

	req := &model.AdaptiveVerifyRequest{
		SessionID:    genResp.SessionID,
		UserID:       "new-user-123",
		Answer:       "answer",
		ResponseTime: 10000,
	}

	resp, err := service.VerifyCaptcha(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestAdaptiveEcosystemService_GetUserProfile(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	_, _ = service.GenerateCaptcha(&model.AdaptiveEcosystemRequest{
		UserID: "profile-user",
	})

	profile, exists := service.GetUserProfile("profile-user")
	assert.True(t, exists)
	assert.NotNil(t, profile)
	assert.Equal(t, "profile-user", profile.UserID)
}

func TestAdaptiveEcosystemService_GetUserProfile_NotFound(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	profile, exists := service.GetUserProfile("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, profile)
}

func TestAdaptiveEcosystemService_GetEcosystemMetrics(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	metrics := service.GetEcosystemMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, "v2.1-ecosystem", metrics.ModelVersion)
}

func TestAdaptiveEcosystemService_AssessRisk_LowRisk(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	req := &model.AdaptiveEcosystemRequest{
		UserID:      "low-risk-user",
		Fingerprint: "fp123",
		Context:     map[string]interface{}{},
	}

	risk := service.assessRisk(req)
	assert.NotNil(t, risk)
	assert.GreaterOrEqual(t, risk.RiskScore, 0.0)
	assert.Contains(t, []string{"low", "medium", "high", "critical"}, risk.RiskLevel)
}

func TestAdaptiveEcosystemService_AssessRisk_HeadlessBrowser(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	req := &model.AdaptiveEcosystemRequest{
		UserID:      "headless-user",
		Fingerprint: "fp456",
		Context: map[string]interface{}{
			"headless": true,
		},
	}

	risk := service.assessRisk(req)
	assert.NotNil(t, risk)
	assert.Greater(t, risk.RiskScore, 0.3)
}

func TestAdaptiveEcosystemService_AssessRisk_Automation(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	req := &model.AdaptiveEcosystemRequest{
		UserID:      "automation-user",
		Fingerprint: "fp789",
		Context: map[string]interface{}{
			"automation": true,
		},
	}

	risk := service.assessRisk(req)
	assert.NotNil(t, risk)
	assert.Greater(t, risk.RiskScore, 0.5)
}

func TestAdaptiveEcosystemService_RecordAttack(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	attack := model.AttackHistory{
		AttackID:      "attack-001",
		AttackType:    "brute_force",
		Timestamp:     time.Now().Unix(),
		TargetCaptcha: model.CaptchaTypeSlider,
		Method:        "script",
		Success:       false,
		DefenseAction: "blocked",
		IPAddress:     "192.168.1.100",
		Fingerprint:   "fp-attack",
	}

	service.RecordAttack(attack)

	metrics := service.GetEcosystemMetrics()
	assert.Equal(t, 1, metrics.AttackCount)
}

func TestAdaptiveEcosystemService_RecordAttack_Success(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	attack := model.AttackHistory{
		AttackID:      "attack-002",
		AttackType:    "bypass",
		Timestamp:     time.Now().Unix(),
		TargetCaptcha: model.CaptchaTypeEmoji,
		Success:       true,
		IPAddress:     "192.168.1.101",
	}

	service.RecordAttack(attack)

	metrics := service.GetEcosystemMetrics()
	assert.Equal(t, 1, metrics.AttackCount)
	assert.Greater(t, metrics.AttackSuccessRate, 0.0)
}

func TestAdaptiveEcosystemService_SelectCaptchaType(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	userProfile := &model.UserProfile{
		UserID:         "select-user",
		SuccessRate:    0.8,
		PreferredTypes: []model.CaptchaType{model.CaptchaTypeSlider},
	}

	riskAssessment := &model.RiskAssessment{
		RiskScore: 0.3,
		RiskLevel: "low",
	}

	captchaType := service.selectCaptchaType(userProfile, riskAssessment)
	assert.Contains(t, []model.CaptchaType{
		model.CaptchaTypeSlider,
		model.CaptchaTypeEmoji,
		model.CaptchaType3D,
	}, captchaType)
}

func TestAdaptiveEcosystemService_SelectCaptchaType_HighRisk(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	userProfile := &model.UserProfile{
		UserID:      "high-risk-select",
		SuccessRate: 0.8,
	}

	riskAssessment := &model.RiskAssessment{
		RiskScore: 0.8,
		RiskLevel: "critical",
	}

	captchaType := service.selectCaptchaType(userProfile, riskAssessment)
	assert.Equal(t, model.CaptchaTypeMultisensory, captchaType)
}

func TestAdaptiveEcosystemService_CalculateAdjustedDifficulty(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	tests := []struct {
		name     string
		profile  *model.UserProfile
		risk     *model.RiskAssessment
		expected model.DifficultyLevel
	}{
		{
			name: "high success rate",
			profile: &model.UserProfile{
				SuccessRate:        0.95,
				PreferredDifficulty: model.DifficultyMedium,
			},
			risk:     &model.RiskAssessment{RiskScore: 0.3},
			expected: model.DifficultyHard,
		},
		{
			name: "low success rate",
			profile: &model.UserProfile{
				SuccessRate:        0.5,
				PreferredDifficulty: model.DifficultyMedium,
			},
			risk:     &model.RiskAssessment{RiskScore: 0.3},
			expected: model.DifficultyEasy,
		},
		{
			name: "high risk",
			profile: &model.UserProfile{
				SuccessRate:        0.85,
				PreferredDifficulty: model.DifficultyEasy,
			},
			risk:     &model.RiskAssessment{RiskScore: 0.8},
			expected: model.DifficultyMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			difficulty := service.calculateAdjustedDifficulty(tt.profile, tt.risk)
			assert.Equal(t, tt.expected, difficulty)
		})
	}
}

func TestAdaptiveEcosystemService_GenerateCaptchaData(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	userProfile := &model.UserProfile{}

	tests := []struct {
		captchaType model.CaptchaType
		difficulty  model.DifficultyLevel
	}{
		{model.CaptchaTypeSlider, model.DifficultyEasy},
		{model.CaptchaTypeSlider, model.DifficultyHard},
		{model.CaptchaTypeEmoji, model.DifficultyMedium},
		{model.CaptchaType3D, model.DifficultyHard},
		{model.CaptchaTypeMultisensory, model.DifficultyExpert},
	}

	for _, tt := range tests {
		t.Run(string(tt.captchaType)+"_"+string(tt.difficulty), func(t *testing.T) {
			data := service.generateCaptchaData(tt.captchaType, tt.difficulty, userProfile)
			assert.NotNil(t, data)
			assert.Equal(t, tt.captchaType, data.(map[string]interface{})["type"])
		})
	}
}

func TestAdaptiveEcosystemService_OptimizeSelf(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	req := &model.SelfOptimizationRequest{
		OptimizationGoal: "increase_security",
		MetricsSnapshot:  service.GetEcosystemMetrics(),
		Constraints:      map[string]interface{}{},
	}

	resp, err := service.OptimizeSelf(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.OptimizationID)
}

func TestAdaptiveEcosystemService_OptimizeSelf_Usability(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	req := &model.SelfOptimizationRequest{
		OptimizationGoal: "improve_usability",
		MetricsSnapshot:  service.GetEcosystemMetrics(),
	}

	resp, err := service.OptimizeSelf(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestAdaptiveEcosystemService_GetEcosystemStatus(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	status := service.GetEcosystemStatus()
	assert.Contains(t, []model.EcosystemStatus{
		model.EcosystemStatusInitializing,
		model.EcosystemStatusActive,
		model.EcosystemStatusEvolving,
		model.EcosystemStatusOptimizing,
		model.EcosystemStatusDegraded,
	}, status)
}

func TestAdaptiveEcosystemService_GetEvolutionHistory(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	history := service.GetEvolutionHistory()
	assert.NotNil(t, history)
}

func TestAdaptiveEcosystemService_UpdateCaptchaConfig(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	newConfig := &model.CaptchaConfig{
		ConfigID:         "config-slider-updated",
		CaptchaType:     model.CaptchaTypeSlider,
		DifficultyLevel: model.DifficultyHard,
		TimeLimit:       30,
		MaxAttempts:     2,
		SuccessThreshold: 0.9,
		Enabled:         true,
	}

	err := service.UpdateCaptchaConfig(model.CaptchaTypeSlider, newConfig)
	assert.NoError(t, err)

	config := service.getCaptchaConfig(model.CaptchaTypeSlider)
	assert.Equal(t, model.DifficultyHard, config.DifficultyLevel)
	assert.Equal(t, 30, config.TimeLimit)
}

func TestAdaptiveEcosystemService_IncreaseDifficulty(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	tests := []struct {
		input    model.DifficultyLevel
		expected model.DifficultyLevel
	}{
		{model.DifficultyEasy, model.DifficultyMedium},
		{model.DifficultyMedium, model.DifficultyHard},
		{model.DifficultyHard, model.DifficultyExpert},
		{model.DifficultyExpert, model.DifficultyExpert},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := service.increaseDifficulty(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAdaptiveEcosystemService_DecreaseDifficulty(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	tests := []struct {
		input    model.DifficultyLevel
		expected model.DifficultyLevel
	}{
		{model.DifficultyExpert, model.DifficultyHard},
		{model.DifficultyHard, model.DifficultyMedium},
		{model.DifficultyMedium, model.DifficultyEasy},
		{model.DifficultyEasy, model.DifficultyEasy},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := service.decreaseDifficulty(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAdaptiveEcosystemService_GenerateAdaptiveHints(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	userProfile := &model.UserProfile{
		AdaptationLevel: 0.2,
	}

	hints := service.generateAdaptiveHints(model.CaptchaTypeSlider, model.DifficultyEasy, userProfile)
	assert.NotNil(t, hints)
}

func TestAdaptiveEcosystemService_IdentifyThreatFactors(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	req := &model.AdaptiveEcosystemRequest{
		Fingerprint: "fp-threat",
		Context: map[string]interface{}{
			"headless":   true,
			"automation": true,
		},
	}

	factors := service.identifyThreatFactors(req, 0.6)
	assert.NotNil(t, factors)
	assert.Greater(t, len(factors), 0)
}

func TestAdaptiveEcosystemService_GenerateRiskRecommendations(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	tests := []struct {
		riskLevel string
	}{
		{"critical"},
		{"high"},
		{"medium"},
		{"low"},
	}

	for _, tt := range tests {
		t.Run(tt.riskLevel, func(t *testing.T) {
			recs := service.generateRiskRecommendations(tt.riskLevel)
			assert.NotNil(t, recs)
			assert.Greater(t, len(recs), 0)
		})
	}
}

func TestAdaptiveEcosystemService_ValidateAnswer(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	tests := []struct {
		name    string
		req     *model.AdaptiveVerifyRequest
	}{
		{
			name: "valid answer",
			req: &model.AdaptiveVerifyRequest{
				UserID: "validate-user",
				Answer: "correct",
			},
		},
		{
			name: "nil answer",
			req: &model.AdaptiveVerifyRequest{
				UserID: "validate-user-nil",
				Answer: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.validateAnswer(tt.req)
			if tt.req.Answer == nil {
				assert.False(t, result)
			}
		})
	}
}

func TestAdaptiveEcosystemService_CalculateSuccessScore(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	userProfile := &model.UserProfile{
		SuccessRate: 0.9,
	}

	score := service.calculateSuccessScore(3000, userProfile)
	assert.Greater(t, score, 0.0)

	score = service.calculateSuccessScore(40000, userProfile)
	assert.Less(t, score, 1.0)
}

func TestAdaptiveEcosystemService_CalculateNextDifficulty(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	tests := []struct {
		name          string
		userProfile   *model.UserProfile
		isCorrect     bool
		responseTime  int64
		expected      model.DifficultyLevel
	}{
		{
			name: "correct fast",
			userProfile: &model.UserProfile{
				SuccessRate:        0.9,
				PreferredDifficulty: model.DifficultyMedium,
			},
			isCorrect:    true,
			responseTime: 5000,
			expected:     model.DifficultyHard,
		},
		{
			name: "incorrect slow",
			userProfile: &model.UserProfile{
				SuccessRate:        0.8,
				PreferredDifficulty: model.DifficultyMedium,
			},
			isCorrect:    false,
			responseTime: 25000,
			expected:     model.DifficultyEasy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateNextDifficulty(tt.userProfile, tt.isCorrect, tt.responseTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAdaptiveEcosystemService_GetAttackCount(t *testing.T) {
	service := NewAdaptiveEcosystemService()

	count := service.getAttackCount("192.168.1.1", "")
	assert.Equal(t, 0, count)

	service.RecordAttack(model.AttackHistory{
		AttackID:   "atk1",
		Timestamp:  time.Now().Add(-1 * time.Hour).Unix(),
		IPAddress:  "192.168.1.1",
		Success:    false,
	})

	count = service.getAttackCount("192.168.1.1", "")
	assert.Equal(t, 1, count)
}

func TestBoolToFloat(t *testing.T) {
	assert.Equal(t, 1.0, boolToFloat(true))
	assert.Equal(t, 0.0, boolToFloat(false))
}

func TestSerializeUserProfile(t *testing.T) {
	profile := &model.UserProfile{
		UserID:      "serialize-user",
		SuccessRate: 0.85,
	}

	data, err := SerializeUserProfile(profile)
	assert.NoError(t, err)
	assert.Greater(t, len(data), 0)

	deserialized, err := DeserializeUserProfile(data)
	assert.NoError(t, err)
	assert.Equal(t, profile.UserID, deserialized.UserID)
	assert.Equal(t, profile.SuccessRate, deserialized.SuccessRate)
}

func TestDeserializeUserProfile_Invalid(t *testing.T) {
	_, err := DeserializeUserProfile([]byte("invalid json"))
	assert.Error(t, err)
}

func TestSortBehaviorPatterns(t *testing.T) {
	patterns := []model.BehaviorPattern{
		{PatternID: "p1", LastSeen: 1000},
		{PatternID: "p2", LastSeen: 3000},
		{PatternID: "p3", LastSeen: 2000},
	}

	sorted := SortBehaviorPatterns(patterns)
	assert.Len(t, sorted, 3)
	assert.Equal(t, "p2", sorted[0].PatternID)
	assert.Equal(t, "p3", sorted[1].PatternID)
	assert.Equal(t, "p1", sorted[2].PatternID)
}

func TestUserProfile_Fields(t *testing.T) {
	profile := &model.UserProfile{
		UserID:             "profile-123",
		SuccessRate:        0.9,
		AvgResponseTime:    12.5,
		AttemptsPerCaptcha: 1.1,
		PreferredTypes:     []model.CaptchaType{model.CaptchaTypeSlider},
		PreferredDifficulty: model.DifficultyHard,
		LastCaptchaTime:    time.Now().Unix(),
		TotalCaptchas:      100,
		SuccessCaptchas:    90,
		AdaptationLevel:    0.8,
		LearningRate:       0.1,
		LastUpdated:        time.Now().Unix(),
	}

	assert.Equal(t, "profile-123", profile.UserID)
	assert.Equal(t, 0.9, profile.SuccessRate)
	assert.Equal(t, 100, profile.TotalCaptchas)
	assert.Len(t, profile.PreferredTypes, 1)
}

func TestRiskProfile_Fields(t *testing.T) {
	risk := &model.RiskProfile{
		BaseScore:        0.5,
		EnvironmentScore: 0.6,
		BehaviorScore:    0.7,
		HistoryScore:     0.4,
		CompositeScore:   0.55,
		RiskLevel:        "medium",
		ThreatIndicators: []string{"indicator1", "indicator2"},
		MitigationActions: []string{"action1"},
		LastAssessed:     time.Now().Unix(),
	}

	assert.Equal(t, 0.5, risk.BaseScore)
	assert.Equal(t, "medium", risk.RiskLevel)
	assert.Len(t, risk.ThreatIndicators, 2)
}

func TestCaptchaConfig_Fields(t *testing.T) {
	config := &model.CaptchaConfig{
		ConfigID:         "config-001",
		CaptchaType:      model.CaptchaTypeSlider,
		DifficultyLevel:  model.DifficultyMedium,
		TimeLimit:        60,
		MaxAttempts:      3,
		SuccessThreshold: 0.8,
		Enabled:          true,
	}

	assert.Equal(t, "config-001", config.ConfigID)
	assert.Equal(t, model.CaptchaTypeSlider, config.CaptchaType)
	assert.Equal(t, 60, config.TimeLimit)
	assert.True(t, config.Enabled)
}

func TestEcosystemMetrics_Fields(t *testing.T) {
	metrics := &model.EcosystemMetrics{
		MetricsID:         "metrics-001",
		Timestamp:         time.Now().Unix(),
		TotalCaptchas:     1000,
		SuccessRate:       0.85,
		AvgResponseTime:   15.5,
		AttackCount:       5,
		AttackSuccessRate: 0.02,
		ActiveUsers:       200,
		ModelVersion:      "v2.0",
		HealthScore:       0.95,
		EvolutionStage:    3,
		OptimizationScore: 0.85,
	}

	assert.Equal(t, "metrics-001", metrics.MetricsID)
	assert.Equal(t, 1000, metrics.TotalCaptchas)
	assert.Equal(t, 0.85, metrics.SuccessRate)
}

func TestRiskAssessment_Fields(t *testing.T) {
	assessment := &model.RiskAssessment{
		AssessmentID:   "risk-001",
		RiskLevel:      "low",
		RiskScore:      0.25,
		ThreatFactors:  []model.ThreatFactor{},
		Recommendations: []string{"rec1", "rec2"},
		Confidence:     0.9,
		ModelUsed:      "v2.0",
	}

	assert.Equal(t, "risk-001", assessment.AssessmentID)
	assert.Equal(t, "low", assessment.RiskLevel)
	assert.Equal(t, 0.25, assessment.RiskScore)
	assert.Equal(t, 0.9, assessment.Confidence)
}

func TestLearningUpdate_Fields(t *testing.T) {
	update := &model.LearningUpdate{
		UpdateID:        "lu-001",
		PatternUpdated: "success_rate",
		Changes:        map[string]float64{"success_rate": 0.02},
		ConfidenceDelta: 0.05,
		Effectiveness:  0.85,
	}

	assert.Equal(t, "lu-001", update.UpdateID)
	assert.Equal(t, "success_rate", update.PatternUpdated)
	assert.Equal(t, 0.05, update.ConfidenceDelta)
}
