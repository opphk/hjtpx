package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type AdaptiveEcosystemService struct {
	userProfiles     map[string]*model.UserProfile
	attackHistory    map[string][]model.AttackHistory
	ecosystemMetrics *model.EcosystemMetrics
	captchaConfigs   map[model.CaptchaType]*model.CaptchaConfig
	ecosystemChanges []model.EcosystemChange
	difficultyAdjustments []model.DifficultyAdjustment
	mu               sync.RWMutex
	modelVersion     string
	ecosystemStatus  model.EcosystemStatus
	evolutionStage   int
}

func NewAdaptiveEcosystemService() *AdaptiveEcosystemService {
	service := &AdaptiveEcosystemService{
		userProfiles:     make(map[string]*model.UserProfile),
		attackHistory:    make(map[string][]model.AttackHistory),
		captchaConfigs:   make(map[model.CaptchaType]*model.CaptchaConfig),
		ecosystemChanges: make([]model.EcosystemChange, 0),
		mu:               sync.RWMutex{},
		modelVersion:     "v2.1-ecosystem",
		ecosystemStatus:  model.EcosystemStatusInitializing,
		evolutionStage:    1,
	}

	service.initializeCaptchaConfigs()
	service.initializeMetrics()

	return service
}

func (s *AdaptiveEcosystemService) initializeCaptchaConfigs() {
	s.captchaConfigs[model.CaptchaTypeSlider] = &model.CaptchaConfig{
		ConfigID:        "config-slider",
		CaptchaType:     model.CaptchaTypeSlider,
		DifficultyLevel: model.DifficultyMedium,
		TimeLimit:       60,
		MaxAttempts:    3,
		SuccessThreshold: 0.8,
		Features:        map[string]interface{}{"tolerance": 5},
		Enabled:         true,
	}

	s.captchaConfigs[model.CaptchaTypeEmoji] = &model.CaptchaConfig{
		ConfigID:        "config-emoji",
		CaptchaType:     model.CaptchaTypeEmoji,
		DifficultyLevel: model.DifficultyMedium,
		TimeLimit:       45,
		MaxAttempts:    3,
		SuccessThreshold: 0.85,
		Features:        map[string]interface{}{"emojiCount": 8},
		Enabled:         true,
	}

	s.captchaConfigs[model.CaptchaType3D] = &model.CaptchaConfig{
		ConfigID:        "config-3d",
		CaptchaType:     model.CaptchaType3D,
		DifficultyLevel: model.DifficultyHard,
		TimeLimit:       90,
		MaxAttempts:    2,
		SuccessThreshold: 0.75,
		Features:        map[string]interface{}{"rotationSteps": 360},
		Enabled:         true,
	}

	s.captchaConfigs[model.CaptchaTypeMultisensory] = &model.CaptchaConfig{
		ConfigID:        "config-multisensory",
		CaptchaType:     model.CaptchaTypeMultisensory,
		DifficultyLevel: model.DifficultyHard,
		TimeLimit:       120,
		MaxAttempts:    2,
		SuccessThreshold: 0.7,
		Features:        map[string]interface{}{"sensoryModes": 3},
		Enabled:         true,
	}

	s.captchaConfigs[model.CaptchaTypeSpatial] = &model.CaptchaConfig{
		ConfigID:        "config-spatial",
		CaptchaType:     model.CaptchaTypeSpatial,
		DifficultyLevel: model.DifficultyExpert,
		TimeLimit:       180,
		MaxAttempts:    1,
		SuccessThreshold: 0.65,
		Features:        map[string]interface{}{"complexity": "high"},
		Enabled:         true,
	}
}

func (s *AdaptiveEcosystemService) initializeMetrics() {
	s.ecosystemMetrics = &model.EcosystemMetrics{
		MetricsID:         fmt.Sprintf("metrics_%d", time.Now().UnixNano()),
		Timestamp:         time.Now().Unix(),
		TotalCaptchas:     0,
		SuccessRate:       0.85,
		AvgResponseTime:   15.0,
		AttackCount:       0,
		AttackSuccessRate: 0.05,
		ActiveUsers:       0,
		ModelVersion:      s.modelVersion,
		HealthScore:       1.0,
		EvolutionStage:     1,
		OptimizationScore: 0.8,
	}
}

func (s *AdaptiveEcosystemService) GenerateCaptcha(req *model.AdaptiveEcosystemRequest) (*model.AdaptiveEcosystemResponse, error) {
	sessionID := fmt.Sprintf("eco_%d_%s", time.Now().UnixNano(), req.UserID)
	expiresAt := time.Now().Add(5 * time.Minute)

	userProfile := s.getOrCreateUserProfile(req.UserID)
	riskAssessment := s.assessRisk(req)
	captchaType := s.selectCaptchaType(userProfile, riskAssessment)
	captchaConfig := s.getCaptchaConfig(captchaType)
	adjustedDifficulty := s.calculateAdjustedDifficulty(userProfile, riskAssessment)

	captchaData := s.generateCaptchaData(captchaType, adjustedDifficulty, userProfile)
	adaptiveHints := s.generateAdaptiveHints(captchaType, adjustedDifficulty, userProfile)

	s.mu.Lock()
	s.ecosystemMetrics.TotalCaptchas++
	s.ecosystemMetrics.ActiveUsers = len(s.userProfiles)
	s.mu.Unlock()

	return &model.AdaptiveEcosystemResponse{
		SessionID:       sessionID,
		CaptchaConfig:   captchaConfig,
		CaptchaData:     captchaData,
		AdaptiveHints:   adaptiveHints,
		RiskAssessment:  riskAssessment,
		EcosystemStatus: s.ecosystemStatus,
		ExpiresIn:       int64(5 * time.Minute / time.Second),
		ExpiresAt:       expiresAt.Unix(),
		ModelVersion:    s.modelVersion,
	}, nil
}

func (s *AdaptiveEcosystemService) VerifyCaptcha(req *model.AdaptiveVerifyRequest) (*model.AdaptiveVerifyResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userProfile, exists := s.userProfiles[req.UserID]
	if !exists {
		userProfile = s.createDefaultProfile(req.UserID)
		s.userProfiles[req.UserID] = userProfile
	}

	isCorrect := s.validateAnswer(req)
	feedback := &model.VerificationFeedback{
		IsCorrect:     isCorrect,
		TimeTaken:     req.ResponseTime,
		DifficultyHit: s.getCurrentDifficulty(userProfile),
		HintUsed:      false,
		UserStruggled: req.ResponseTime > 30000,
	}

	var score float64
	var message string
	if isCorrect {
		score = s.calculateSuccessScore(req.ResponseTime, userProfile)
		message = "验证成功"
		userProfile.SuccessCaptchas++
		userProfile.TotalCaptchas++
		userProfile.SuccessRate = float64(userProfile.SuccessCaptchas) / float64(userProfile.TotalCaptchas)
	} else {
		score = 0.0
		message = "验证失败"
		userProfile.TotalCaptchas++
	}

	s.updateBehaviorPattern(userProfile, req)
	nextDifficulty := s.calculateNextDifficulty(userProfile, isCorrect, req.ResponseTime)
	learningUpdate := s.performLearningUpdate(userProfile, req, isCorrect)

	if isCorrect {
		s.recordSuccessfulVerification(userProfile, req)
	} else {
		s.handleFailedVerification(userProfile, req)
	}

	s.updateEcosystemMetrics(isCorrect, req.ResponseTime)

	return &model.AdaptiveVerifyResponse{
		Success:        isCorrect,
		Score:          score,
		Message:        message,
		NextDifficulty: nextDifficulty,
		LearningUpdate: learningUpdate,
		Feedback:       feedback,
	}, nil
}

func (s *AdaptiveEcosystemService) getOrCreateUserProfile(userID string) *model.UserProfile {
	s.mu.RLock()
	profile, exists := s.userProfiles[userID]
	s.mu.RUnlock()

	if !exists {
		profile = s.createDefaultProfile(userID)
		s.mu.Lock()
		s.userProfiles[userID] = profile
		s.mu.Unlock()
	}

	return profile
}

func (s *AdaptiveEcosystemService) createDefaultProfile(userID string) *model.UserProfile {
	return &model.UserProfile{
		UserID:             userID,
		SuccessRate:        0.85,
		AvgResponseTime:    15000,
		AttemptsPerCaptcha: 1.2,
		PreferredTypes:     []model.CaptchaType{model.CaptchaTypeSlider},
		PreferredDifficulty: model.DifficultyMedium,
		LastCaptchaTime:    time.Now().Unix(),
		TotalCaptchas:      0,
		SuccessCaptchas:    0,
		BehaviorPatterns:   make([]model.BehaviorPattern, 0),
		RiskProfile:        s.createDefaultRiskProfile(),
		AdaptationLevel:    0.5,
		LearningRate:       0.1,
		LastUpdated:        time.Now().Unix(),
	}
}

func (s *AdaptiveEcosystemService) createDefaultRiskProfile() *model.RiskProfile {
	return &model.RiskProfile{
		BaseScore:        0.5,
		EnvironmentScore: 0.5,
		BehaviorScore:    0.5,
		HistoryScore:     0.5,
		CompositeScore:   0.5,
		RiskLevel:        "low",
		ThreatIndicators: make([]string, 0),
		MitigationActions: make([]string, 0),
		LastAssessed:     time.Now().Unix(),
	}
}

func (s *AdaptiveEcosystemService) assessRisk(req *model.AdaptiveEcosystemRequest) *model.RiskAssessment {
	assessmentID := fmt.Sprintf("risk_%d_%s", time.Now().UnixNano(), req.UserID)

	baseRisk := 0.3
	if len(req.Fingerprint) == 0 {
		baseRisk += 0.1
	}
	if req.Context != nil {
		if _, ok := req.Context["headless"]; ok {
			baseRisk += 0.2
		}
		if _, ok := req.Context["automation"]; ok {
			baseRisk += 0.3
		}
	}

	userProfile, exists := s.userProfiles[req.UserID]
	if exists && userProfile.RiskProfile != nil {
		baseRisk = baseRisk*0.5 + userProfile.RiskProfile.CompositeScore*0.5
	}

	attackCount := s.getAttackCount(req.IPAddress, req.Fingerprint)
	baseRisk += float64(attackCount) * 0.05

	var riskLevel string
	if baseRisk < 0.3 {
		riskLevel = "low"
	} else if baseRisk < 0.6 {
		riskLevel = "medium"
	} else if baseRisk < 0.8 {
		riskLevel = "high"
	} else {
		riskLevel = "critical"
	}

	threatFactors := s.identifyThreatFactors(req, baseRisk)
	recommendations := s.generateRiskRecommendations(riskLevel)

	return &model.RiskAssessment{
		AssessmentID:   assessmentID,
		RiskLevel:      riskLevel,
		RiskScore:      baseRisk,
		ThreatFactors:  threatFactors,
		Recommendations: recommendations,
		Confidence:     0.8,
		ModelUsed:      s.modelVersion,
	}
}

func (s *AdaptiveEcosystemService) identifyThreatFactors(req *model.AdaptiveEcosystemRequest, riskScore float64) []model.ThreatFactor {
	var factors []model.ThreatFactor

	if req.Context != nil {
		if _, ok := req.Context["headless"]; ok {
			factors = append(factors, model.ThreatFactor{
				FactorID:     "tf_headless",
				FactorName:   "Headless Browser Detected",
				Weight:       0.3,
				Score:        0.8,
				Contribution: 0.24,
				Description:  "检测到无头浏览器特征",
			})
		}

		if _, ok := req.Context["automation"]; ok {
			factors = append(factors, model.ThreatFactor{
				FactorID:     "tf_automation",
				FactorName:   "Automation Framework Detected",
				Weight:       0.4,
				Score:        0.9,
				Contribution: 0.36,
				Description:  "检测到自动化框架",
			})
		}
	}

	if len(req.Fingerprint) == 0 {
		factors = append(factors, model.ThreatFactor{
			FactorID:     "tf_no_fp",
			FactorName:   "Missing Fingerprint",
			Weight:       0.2,
			Score:        0.6,
			Contribution: 0.12,
			Description:  "缺少设备指纹",
		})
	}

	attackCount := s.getAttackCount(req.IPAddress, req.Fingerprint)
	if attackCount > 0 {
		factors = append(factors, model.ThreatFactor{
			FactorID:     "tf_attack_history",
			FactorName:   "Historical Attack Attempts",
			Weight:       0.3,
			Score:        math.Min(1.0, float64(attackCount)*0.2),
			Contribution: 0.06,
			Description:  fmt.Sprintf("历史攻击尝试: %d次", attackCount),
		})
	}

	return factors
}

func (s *AdaptiveEcosystemService) generateRiskRecommendations(riskLevel string) []string {
	switch riskLevel {
	case "critical":
		return []string{"启用最高防护级别", "要求多因素认证", "限制访问频率", "启用人工审核"}
	case "high":
		return []string{"提高验证码难度", "启用行为分析", "记录详细日志"}
	case "medium":
		return []string{"保持当前防护", "监控异常行为"}
	case "low":
		return []string{"使用标准验证码", "继续监控"}
	default:
		return []string{"保持观察"}
	}
}

func (s *AdaptiveEcosystemService) selectCaptchaType(userProfile *model.UserProfile, riskAssessment *model.RiskAssessment) model.CaptchaType {
	if riskAssessment.RiskScore > 0.7 {
		return model.CaptchaTypeMultisensory
	}

	if len(userProfile.PreferredTypes) > 0 && rand.Float64() < 0.7 {
		return userProfile.PreferredTypes[rand.Intn(len(userProfile.PreferredTypes))]
	}

	availableTypes := []model.CaptchaType{
		model.CaptchaTypeSlider,
		model.CaptchaTypeEmoji,
		model.CaptchaType3D,
	}

	return availableTypes[rand.Intn(len(availableTypes))]
}

func (s *AdaptiveEcosystemService) getCaptchaConfig(captchaType model.CaptchaType) *model.CaptchaConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if config, exists := s.captchaConfigs[captchaType]; exists {
		return config
	}

	return &model.CaptchaConfig{
		ConfigID:         fmt.Sprintf("config_%s", captchaType),
		CaptchaType:      captchaType,
		DifficultyLevel:  model.DifficultyMedium,
		TimeLimit:        60,
		MaxAttempts:      3,
		SuccessThreshold: 0.8,
		Enabled:          true,
	}
}

func (s *AdaptiveEcosystemService) calculateAdjustedDifficulty(userProfile *model.UserProfile, riskAssessment *model.RiskAssessment) model.DifficultyLevel {
	baseDifficulty := userProfile.PreferredDifficulty

	if riskAssessment.RiskScore > 0.7 {
		if baseDifficulty == model.DifficultyEasy {
			baseDifficulty = model.DifficultyMedium
		} else if baseDifficulty == model.DifficultyMedium {
			baseDifficulty = model.DifficultyHard
		}
	}

	if userProfile.SuccessRate > 0.95 {
		baseDifficulty = s.increaseDifficulty(baseDifficulty)
	} else if userProfile.SuccessRate < 0.6 {
		baseDifficulty = s.decreaseDifficulty(baseDifficulty)
	}

	return baseDifficulty
}

func (s *AdaptiveEcosystemService) increaseDifficulty(level model.DifficultyLevel) model.DifficultyLevel {
	switch level {
	case model.DifficultyEasy:
		return model.DifficultyMedium
	case model.DifficultyMedium:
		return model.DifficultyHard
	case model.DifficultyHard:
		return model.DifficultyExpert
	default:
		return model.DifficultyExpert
	}
}

func (s *AdaptiveEcosystemService) decreaseDifficulty(level model.DifficultyLevel) model.DifficultyLevel {
	switch level {
	case model.DifficultyExpert:
		return model.DifficultyHard
	case model.DifficultyHard:
		return model.DifficultyMedium
	case model.DifficultyMedium:
		return model.DifficultyEasy
	default:
		return model.DifficultyEasy
	}
}

func (s *AdaptiveEcosystemService) generateCaptchaData(captchaType model.CaptchaType, difficulty model.DifficultyLevel, userProfile *model.UserProfile) interface{} {
	baseData := map[string]interface{}{
		"type":       captchaType,
		"difficulty": difficulty,
		"timestamp":  time.Now().Unix(),
	}

	switch captchaType {
	case model.CaptchaTypeSlider:
		tolerance := 5
		if difficulty == model.DifficultyEasy {
			tolerance = 8
		} else if difficulty == model.DifficultyHard {
			tolerance = 3
		} else if difficulty == model.DifficultyExpert {
			tolerance = 2
		}
		baseData["tolerance"] = tolerance

	case model.CaptchaTypeEmoji:
		emojiCount := 8
		if difficulty == model.DifficultyEasy {
			emojiCount = 6
		} else if difficulty == model.DifficultyHard {
			emojiCount = 10
		} else if difficulty == model.DifficultyExpert {
			emojiCount = 12
		}
		baseData["emojiCount"] = emojiCount

	case model.CaptchaType3D:
		rotationSteps := 360
		if difficulty == model.DifficultyEasy {
			rotationSteps = 180
		} else if difficulty == model.DifficultyHard {
			rotationSteps = 720
		}
		baseData["rotationSteps"] = rotationSteps

	case model.CaptchaTypeMultisensory:
		sensoryModes := 2
		if difficulty == model.DifficultyEasy {
			sensoryModes = 1
		} else if difficulty == model.DifficultyHard || difficulty == model.DifficultyExpert {
			sensoryModes = 3
		}
		baseData["sensoryModes"] = sensoryModes
	}

	return baseData
}

func (s *AdaptiveEcosystemService) generateAdaptiveHints(captchaType model.CaptchaType, difficulty model.DifficultyLevel, userProfile *model.UserProfile) []string {
	var hints []string

	if userProfile.AdaptationLevel < 0.3 {
		switch captchaType {
		case model.CaptchaTypeSlider:
			hints = append(hints, "将滑块拖动到缺口位置")
		case model.CaptchaTypeEmoji:
			hints = append(hints, "点击与示例相同的表情")
		case model.CaptchaType3D:
			hints = append(hints, "旋转图形使它与示例匹配")
		}
	}

	if difficulty == model.DifficultyHard || difficulty == model.DifficultyExpert {
		hints = append(hints, "此验证较为困难，请仔细观察")
	}

	return hints
}

func (s *AdaptiveEcosystemService) validateAnswer(req *model.AdaptiveVerifyRequest) bool {
	if req.Answer == nil {
		return false
	}

	correctRate := 0.85
	if userProfile, exists := s.userProfiles[req.UserID]; exists {
		correctRate = userProfile.SuccessRate
	}

	if userProfile, exists := s.userProfiles[req.UserID]; exists && userProfile.RiskProfile != nil {
		if userProfile.RiskProfile.CompositeScore > 0.7 {
			correctRate *= 0.8
		}
	}

	successProbability := correctRate * 0.9
	return rand.Float64() < successProbability
}

func (s *AdaptiveEcosystemService) calculateSuccessScore(responseTime int64, userProfile *model.UserProfile) float64 {
	baseScore := 1.0

	if responseTime < 5000 {
		baseScore += 0.1
	} else if responseTime > 30000 {
		baseScore -= 0.2
	}

	if userProfile.SuccessRate > 0.9 {
		baseScore *= 1.1
	}

	return math.Min(1.0, baseScore)
}

func (s *AdaptiveEcosystemService) getCurrentDifficulty(userProfile *model.UserProfile) model.DifficultyLevel {
	return userProfile.PreferredDifficulty
}

func (s *AdaptiveEcosystemService) calculateNextDifficulty(userProfile *model.UserProfile, isCorrect bool, responseTime int64) model.DifficultyLevel {
	currentDifficulty := userProfile.PreferredDifficulty

	if isCorrect {
		if responseTime < 10000 && userProfile.SuccessRate > 0.85 {
			currentDifficulty = s.increaseDifficulty(currentDifficulty)
		}
	} else {
		if responseTime > 20000 || userProfile.SuccessRate < 0.7 {
			currentDifficulty = s.decreaseDifficulty(currentDifficulty)
		}
	}

	userProfile.PreferredDifficulty = currentDifficulty
	return currentDifficulty
}

func (s *AdaptiveEcosystemService) updateBehaviorPattern(userProfile *model.UserProfile, req *model.AdaptiveVerifyRequest) {
	if req.BehaviorData == nil {
		return
	}

	pattern := model.BehaviorPattern{
		PatternID:   fmt.Sprintf("bp_%d", time.Now().UnixNano()),
		PatternType: "response_time",
		Features:    make(map[string]float64),
		Confidence:  0.5,
		Frequency:   1.0,
		FirstSeen:   time.Now().Unix(),
		LastSeen:    time.Now().Unix(),
		SuccessCount: 0,
		FailCount:   0,
	}

	if req.ResponseTime > 0 {
		pattern.Features["avg_response_time"] = float64(req.ResponseTime)
	}

	userProfile.BehaviorPatterns = append(userProfile.BehaviorPatterns, pattern)

	if len(userProfile.BehaviorPatterns) > 50 {
		userProfile.BehaviorPatterns = userProfile.BehaviorPatterns[len(userProfile.BehaviorPatterns)-50:]
	}
}

func (s *AdaptiveEcosystemService) performLearningUpdate(userProfile *model.UserProfile, req *model.AdaptiveVerifyRequest, isCorrect bool) *model.LearningUpdate {
	update := &model.LearningUpdate{
		UpdateID:        fmt.Sprintf("lu_%d", time.Now().UnixNano()),
		PatternUpdated:  "success_rate",
		Changes:         make(map[string]float64),
		ConfidenceDelta: 0.05,
		Effectiveness:  0.8,
	}

	if isCorrect {
		update.Changes["success_rate"] = 0.02
		update.Changes["adaptation_level"] = 0.01
		userProfile.AdaptationLevel = math.Min(1.0, userProfile.AdaptationLevel+0.01)
	} else {
		update.Changes["success_rate"] = -0.02
		update.Effectiveness = 0.6
	}

	userProfile.LastUpdated = time.Now().Unix()

	return update
}

func (s *AdaptiveEcosystemService) recordSuccessfulVerification(userProfile *model.UserProfile, req *model.AdaptiveVerifyRequest) {
	if userProfile.RiskProfile != nil {
		userProfile.RiskProfile.CompositeScore *= 0.95
		if userProfile.RiskProfile.CompositeScore < 0.2 {
			userProfile.RiskProfile.CompositeScore = 0.2
		}
	}
}

func (s *AdaptiveEcosystemService) handleFailedVerification(userProfile *model.UserProfile, req *model.AdaptiveVerifyRequest) {
	if userProfile.RiskProfile != nil {
		userProfile.RiskProfile.CompositeScore *= 1.05
		if userProfile.RiskProfile.CompositeScore > 1.0 {
			userProfile.RiskProfile.CompositeScore = 1.0
		}
	}
}

func (s *AdaptiveEcosystemService) getAttackCount(ipAddress, fingerprint string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := ipAddress
	if fingerprint != "" {
		key = fingerprint
	}

	attacks := s.attackHistory[key]
	count := 0
	for _, attack := range attacks {
		if time.Since(time.Unix(attack.Timestamp, 0)) < 24*time.Hour {
			count++
		}
	}
	return count
}

func (s *AdaptiveEcosystemService) updateEcosystemMetrics(isCorrect bool, responseTime int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	total := float64(s.ecosystemMetrics.TotalCaptchas)
	if total > 0 {
		currentRate := s.ecosystemMetrics.SuccessRate
		s.ecosystemMetrics.SuccessRate = (currentRate*total + boolToFloat(isCorrect)) / (total + 1)
	}

	currentAvg := s.ecosystemMetrics.AvgResponseTime
	if total > 0 {
		s.ecosystemMetrics.AvgResponseTime = (currentAvg*total + float64(responseTime)) / (total + 1)
	} else {
		s.ecosystemMetrics.AvgResponseTime = float64(responseTime)
	}

	if s.ecosystemMetrics.TotalCaptchas%100 == 0 {
		s.evolveEcosystem()
	}
}

func (s *AdaptiveEcosystemService) evolveEcosystem() {
	s.ecosystemStatus = model.EcosystemStatusEvolving
	s.evolutionStage++

	previousMetrics := *s.ecosystemMetrics

	if s.ecosystemMetrics.SuccessRate > 0.9 {
		s.ecosystemMetrics.OptimizationScore += 0.01
	}

	if s.ecosystemMetrics.AttackSuccessRate < 0.05 {
		s.ecosystemMetrics.HealthScore = 1.0
	} else {
		s.ecosystemMetrics.HealthScore -= 0.1
		if s.ecosystemMetrics.HealthScore < 0.5 {
			s.ecosystemMetrics.HealthScore = 0.5
			s.ecosystemStatus = model.EcosystemStatusDegraded
		}
	}

	change := model.EcosystemChange{
		ChangeType: "evolution",
		Target:     "ecosystem",
		OldValue:   previousMetrics,
		NewValue:   s.ecosystemMetrics,
		Reason:     fmt.Sprintf("Stage %d evolution", s.evolutionStage),
	}
	s.ecosystemChanges = append(s.ecosystemChanges, change)

	s.ecosystemStatus = model.EcosystemStatusActive
}

func (s *AdaptiveEcosystemService) GetEcosystemMetrics() *model.EcosystemMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.ecosystemMetrics
}

func (s *AdaptiveEcosystemService) GetUserProfile(userID string) (*model.UserProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile, exists := s.userProfiles[userID]
	return profile, exists
}

func (s *AdaptiveEcosystemService) RecordAttack(attack model.AttackHistory) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := attack.IPAddress
	if attack.Fingerprint != "" {
		key = attack.Fingerprint
	}

	s.attackHistory[key] = append(s.attackHistory[key], attack)
	s.ecosystemMetrics.AttackCount++

	if attack.Success {
		s.ecosystemMetrics.AttackSuccessRate = (s.ecosystemMetrics.AttackSuccessRate*float64(s.ecosystemMetrics.AttackCount-1) + 1) / float64(s.ecosystemMetrics.AttackCount)
	}
}

func (s *AdaptiveEcosystemService) OptimizeSelf(req *model.SelfOptimizationRequest) (*model.SelfOptimizationResponse, error) {
	optID := fmt.Sprintf("opt_%d", time.Now().UnixNano())

	var changes []model.EcosystemChange

	if req.OptimizationGoal == "increase_security" {
		if s.ecosystemMetrics.AttackSuccessRate > 0.05 {
			changes = append(changes, model.EcosystemChange{
				ChangeType: "difficulty_increase",
				Target:     "all_captchas",
				OldValue:   "current",
				NewValue:   "increased",
				Reason:     "Attack success rate above threshold",
			})
		}
	}

	if req.OptimizationGoal == "improve_usability" {
		if s.ecosystemMetrics.AvgResponseTime > 20000 {
			changes = append(changes, model.EcosystemChange{
				ChangeType: "difficulty_decrease",
				Target:     "standard_captchas",
				OldValue:   "current",
				NewValue:   "decreased",
				Reason:     "Response time above threshold",
			})
		}
	}

	predictedImpact := 0.1
	if len(changes) > 0 {
		predictedImpact = 0.15
	}

	return &model.SelfOptimizationResponse{
		OptimizationID:    optID,
		RecommendedChanges: changes,
		PredictedImpact:   predictedImpact,
		Confidence:        0.75,
		ImplementationPlan: []string{"review_changes", "apply_in_staging", "deploy_to_production"},
	}, nil
}

func (s *AdaptiveEcosystemService) GetEcosystemStatus() model.EcosystemStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ecosystemStatus
}

func (s *AdaptiveEcosystemService) GetEvolutionHistory() []model.EcosystemChange {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ecosystemChanges
}

func (s *AdaptiveEcosystemService) UpdateCaptchaConfig(captchaType model.CaptchaType, config *model.CaptchaConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.captchaConfigs[captchaType] = config
	return nil
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func SerializeUserProfile(profile *model.UserProfile) ([]byte, error) {
	return json.Marshal(profile)
}

func DeserializeUserProfile(data []byte) (*model.UserProfile, error) {
	var profile model.UserProfile
	err := json.Unmarshal(data, &profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func SortBehaviorPatterns(patterns []model.BehaviorPattern) []model.BehaviorPattern {
	sorted := make([]model.BehaviorPattern, len(patterns))
	copy(sorted, patterns)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LastSeen > sorted[j].LastSeen
	})
	return sorted
}
