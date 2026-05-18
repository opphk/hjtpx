package service

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

type CaptchaType string

const (
	CaptchaTypeSlider     CaptchaType = "slider"
	CaptchaTypeClick      CaptchaType = "click"
	CaptchaTypeGesture     CaptchaType = "gesture"
	CaptchaTypeLianLianKan CaptchaType = "lianliankan"
	CaptchaTypeVoice      CaptchaType = "voice"
	CaptchaType3D         CaptchaType = "3d"
	CaptchaTypeSeamless   CaptchaType = "seamless"
)

type AIRecommendationService struct {
	behaviorAnalysis *BehaviorAnalysisService
	envDetector     *EnvDetectorService
	userHistory     map[string]*AIRecommendationUserHistory
	mu              sync.RWMutex
	cacheExpiration time.Duration
}

type AIRecommendationUserHistory struct {
	UserID           string
	Fingerprint      string
	IPAddress        string
	SuccessCount     int
	FailureCount     int
	TotalAttempts    int
	LastCaptchaType  CaptchaType
	SuccessRate      float64
	AvgDuration      int64
	PreferredTypes   map[CaptchaType]int
	LastVerifiedAt   time.Time
	CreatedAt        time.Time
	BehaviorProfiles []AIRecommendationBehaviorProfile
}

type AIRecommendationBehaviorProfile struct {
	CaptchaType   CaptchaType
	SuccessRate   float64
	AvgDuration   int64
	AttemptsCount int
}

type CaptchaRecommendationRequest struct {
	UserID          string                 `json:"user_id"`
	Fingerprint     string                 `json:"fingerprint"`
	IPAddress       string                 `json:"ip_address"`
	SessionID       string                 `json:"session_id"`
	ApplicationID   uint                   `json:"application_id"`
	EnvInfo        *EnvInfo               `json:"env_info"`
	RiskScore      float64                `json:"risk_score"`
	TimeOfDay      int                    `json:"time_of_day"`
	AccessFrequency float64               `json:"access_frequency"`
	DeviceTrust    float64                `json:"device_trust"`
	BehaviorData   []BehaviorDataPoint    `json:"behavior_data"`
}

type CaptchaRecommendation struct {
	RecommendedType   CaptchaType           `json:"recommended_type"`
	Confidence        float64               `json:"confidence"`
	Alternatives      []AlternativeCaptcha   `json:"alternatives"`
	Difficulty        CaptchaDifficulty     `json:"difficulty"`
	Factors          []RecommendationFactor `json:"factors"`
	EstimatedDuration int64                 `json:"estimated_duration_ms"`
	Reason           string                 `json:"reason"`
}

type AlternativeCaptcha struct {
	Type   CaptchaType `json:"type"`
	Score  float64     `json:"score"`
	Reason string      `json:"reason"`
}

type CaptchaDifficulty struct {
	Level         string `json:"level"`
	Score         float64 `json:"score"`
	SliderOffset  int    `json:"slider_offset,omitempty"`
	JigsawPieces  int    `json:"jigsaw_pieces,omitempty"`
	ClickCount    int    `json:"click_count,omitempty"`
	TimeLimit     int    `json:"time_limit_seconds"`
}

type RecommendationFactor struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
	Impact float64 `json:"impact"`
	Score  float64 `json:"score"`
}

type DifficultyRequest struct {
	UserID       string       `json:"user_id"`
	Fingerprint  string       `json:"fingerprint"`
	CaptchaType  CaptchaType `json:"captcha_type"`
	RiskScore    float64      `json:"risk_score"`
	TimeOfDay    int          `json:"time_of_day"`
	SuccessRate  float64      `json:"success_rate"`
	AvgDuration  int64        `json:"avg_duration_ms"`
	FailureCount int          `json:"failure_count"`
}

type DifficultyResponse struct {
	RecommendedLevel string              `json:"recommended_level"`
	Difficulty       CaptchaDifficulty  `json:"difficulty"`
	Confidence       float64            `json:"confidence"`
	Factors          []AIRecommendationFactorDetail `json:"factors"`
	AdjustmentReason string              `json:"adjustment_reason"`
}

type AIRecommendationFactorDetail struct {
	Name     string  `json:"name"`
	Impact   float64 `json:"impact"`
	NewValue float64 `json:"new_value"`
	OldValue float64 `json:"old_value"`
}

type CaptchaTypeRecommendationStats struct {
	Type          CaptchaType `json:"type"`
	SuccessRate   float64     `json:"success_rate"`
	AvgDuration   int64       `json:"avg_duration_ms"`
	TotalAttempts int         `json:"total_attempts"`
	FailureRate   float64     `json:"failure_rate"`
	ComfortScore  float64     `json:"comfort_score"`
}

var defaultCaptchaStats = map[CaptchaType]*CaptchaTypeRecommendationStats{
	CaptchaTypeSlider: {
		Type:          CaptchaTypeSlider,
		SuccessRate:   0.85,
		AvgDuration:   5000,
		TotalAttempts: 1000,
		FailureRate:   0.15,
		ComfortScore:  0.8,
	},
	CaptchaTypeClick: {
		Type:          CaptchaTypeClick,
		SuccessRate:   0.80,
		AvgDuration:   6000,
		TotalAttempts: 800,
		FailureRate:   0.20,
		ComfortScore:  0.75,
	},
	CaptchaTypeGesture: {
		Type:          CaptchaTypeGesture,
		SuccessRate:   0.75,
		AvgDuration:   8000,
		TotalAttempts: 600,
		FailureRate:   0.25,
		ComfortScore:  0.7,
	},
	CaptchaTypeLianLianKan: {
		Type:          CaptchaTypeLianLianKan,
		SuccessRate:   0.90,
		AvgDuration:   10000,
		TotalAttempts: 500,
		FailureRate:   0.10,
		ComfortScore:  0.85,
	},
	CaptchaTypeVoice: {
		Type:          CaptchaTypeVoice,
		SuccessRate:   0.70,
		AvgDuration:   7000,
		TotalAttempts: 400,
		FailureRate:   0.30,
		ComfortScore:  0.65,
	},
	CaptchaType3D: {
		Type:          CaptchaType3D,
		SuccessRate:   0.88,
		AvgDuration:   5500,
		TotalAttempts: 450,
		FailureRate:   0.12,
		ComfortScore:  0.82,
	},
}

func NewAIRecommendationService() *AIRecommendationService {
	return &AIRecommendationService{
		behaviorAnalysis: NewBehaviorAnalysisService(),
		envDetector:     NewEnvDetectorService(),
		userHistory:     make(map[string]*AIRecommendationUserHistory),
		cacheExpiration: 30 * time.Minute,
	}
}

func (s *AIRecommendationService) GetRecommendation(ctx context.Context, req *CaptchaRecommendationRequest) (*CaptchaRecommendation, error) {
	userKey := s.getUserKey(req.UserID, req.Fingerprint)
	userHistory := s.getUserHistory(userKey)

	var envRiskScore float64
	if req.EnvInfo != nil {
		envRisk := s.envDetector.envDetector.EvaluateRisk(req.EnvInfo)
		envRiskScore = envRisk.Score
	}

	var behaviorRiskScore float64
	if len(req.BehaviorData) > 0 {
		features := ExtractFeatures(s.convertToTrajectory(req.BehaviorData))
		behaviorRiskScore = features.RiskScore
	}

	combinedRisk := s.calculateCombinedRisk(req.RiskScore, envRiskScore, behaviorRiskScore)

	recommendedType := s.selectOptimalCaptchaType(userHistory, combinedRisk, req)

	difficulty := s.calculateDifficulty(recommendedType, combinedRisk, userHistory, req)

	alternatives := s.generateAlternatives(recommendedType, userHistory, combinedRisk, req)

	factors := s.analyzeRecommendationFactors(recommendedType, combinedRisk, userHistory, req)

	estimatedDuration := s.estimateDuration(recommendedType, difficulty)

	reason := s.generateRecommendationReason(recommendedType, combinedRisk, userHistory)

	return &CaptchaRecommendation{
		RecommendedType:   recommendedType,
		Confidence:        s.calculateConfidence(recommendedType, userHistory, combinedRisk),
		Alternatives:      alternatives,
		Difficulty:        difficulty,
		Factors:           factors,
		EstimatedDuration: estimatedDuration,
		Reason:            reason,
	}, nil
}

func (s *AIRecommendationService) GetDifficultyRecommendation(ctx context.Context, req *DifficultyRequest) (*DifficultyResponse, error) {
	var userHistory *AIRecommendationUserHistory
	if req.UserID != "" || req.Fingerprint != "" {
		userKey := s.getUserKey(req.UserID, req.Fingerprint)
		userHistory = s.getUserHistory(userKey)
	}

	baseDifficulty := s.getBaseDifficulty(req.CaptchaType)

	adjustedDifficulty := s.adjustDifficultyForContext(baseDifficulty, req, userHistory)

	factors := s.analyzeDifficultyFactors(adjustedDifficulty, req, userHistory)

	adjustmentReason := s.generateAdjustmentReason(adjustedDifficulty, req, userHistory)

	return &DifficultyResponse{
		RecommendedLevel: adjustedDifficulty.Level,
		Difficulty:       adjustedDifficulty,
		Confidence:       s.calculateDifficultyConfidence(req, userHistory),
		Factors:          factors,
		AdjustmentReason: adjustmentReason,
	}, nil
}

func (s *AIRecommendationService) UpdateUserHistory(userID, fingerprint, ipAddress string, captchaType CaptchaType, success bool, duration int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userKey := s.getUserKey(userID, fingerprint)
	history, exists := s.userHistory[userKey]

	if !exists {
		history = &AIRecommendationUserHistory{
			UserID:         userID,
			Fingerprint:    fingerprint,
			IPAddress:      ipAddress,
			PreferredTypes: make(map[CaptchaType]int),
			CreatedAt:      time.Now(),
		}
		s.userHistory[userKey] = history
	}

	history.TotalAttempts++
	history.LastVerifiedAt = time.Now()

	if success {
		history.SuccessCount++
		history.PreferredTypes[captchaType]++
		if history.LastCaptchaType == captchaType {
			history.BehaviorProfiles = append(history.BehaviorProfiles, AIRecommendationBehaviorProfile{
				CaptchaType:   captchaType,
				SuccessRate:   1.0,
				AvgDuration:   duration,
				AttemptsCount: 1,
			})
		}
	} else {
		history.FailureCount++
	}

	history.SuccessRate = float64(history.SuccessCount) / float64(history.TotalAttempts)
	history.AvgDuration = (history.AvgDuration*int64(history.TotalAttempts-1) + duration) / int64(history.TotalAttempts)
	history.LastCaptchaType = captchaType

	s.cleanupOldHistory()
}

func (s *AIRecommendationService) getUserKey(userID, fingerprint string) string {
	if userID != "" {
		return "user:" + userID
	}
	if fingerprint != "" {
		return "fp:" + fingerprint
	}
	return "unknown"
}

func (s *AIRecommendationService) getUserHistory(userKey string) *AIRecommendationUserHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if history, exists := s.userHistory[userKey]; exists {
		return history
	}
	return nil
}

func (s *AIRecommendationService) calculateCombinedRisk(clientRisk, envRisk, behaviorRisk float64) float64 {
	weights := map[string]float64{
		"client":   0.30,
		"env":      0.35,
		"behavior": 0.35,
	}

	combined := clientRisk*weights["client"] + envRisk*weights["env"] + behaviorRisk*weights["behavior"]

	if envRisk < 50 && behaviorRisk < 30 {
		combined = combined * 0.8
	}

	if clientRisk > 80 || envRisk > 80 || behaviorRisk > 80 {
		combined = math.Min(combined*1.2, 100)
	}

	return math.Round(math.Max(math.Min(combined, 100), 0)*100) / 100
}

func (s *AIRecommendationService) selectOptimalCaptchaType(history *AIRecommendationUserHistory, riskScore float64, req *CaptchaRecommendationRequest) CaptchaType {
	typeScores := make(map[CaptchaType]float64)

	for captchaType, stats := range defaultCaptchaStats {
		score := s.calculateCaptchaTypeScore(captchaType, stats, history, riskScore, req)
		typeScores[captchaType] = score
	}

	var bestType CaptchaType
	var bestScore float64

	for t, score := range typeScores {
		if score > bestScore {
			bestScore = score
			bestType = t
		}
	}

	if bestType == "" {
		bestType = CaptchaTypeSlider
	}

	return bestType
}

func (s *AIRecommendationService) calculateCaptchaTypeScore(captchaType CaptchaType, stats *CaptchaTypeRecommendationStats, history *AIRecommendationUserHistory, riskScore float64, req *CaptchaRecommendationRequest) float64 {
	score := 0.0

	comfortWeight := 0.25
	successWeight := 0.30
	speedWeight := 0.15
	historyWeight := 0.20
	riskWeight := 0.10

	baseSuccessRate := stats.SuccessRate
	if baseSuccessRate == 0 {
		baseSuccessRate = 0.7
	}
	score += baseSuccessRate * successWeight * 100

	score += stats.ComfortScore * comfortWeight * 100

	speedFactor := 1.0 - float64(stats.AvgDuration)/20000.0
	if speedFactor < 0 {
		speedFactor = 0
	}
	score += speedFactor * speedWeight * 100

	if history != nil {
		if histTypeCount, exists := history.PreferredTypes[captchaType]; exists {
			historyFactor := float64(histTypeCount) / float64(history.TotalAttempts)
			score += historyFactor * historyWeight * 100
		}

		if history.SuccessRate > 0.8 && riskScore < 40 {
			preferredTypes := s.getTopPreferredTypes(history)
			for _, pt := range preferredTypes {
				if pt == captchaType {
					score += 10
					break
				}
			}
		}
	}

	if riskScore > 60 {
		if captchaType == CaptchaType3D || captchaType == CaptchaTypeLianLianKan {
			score += riskWeight * 100
		}
	} else if riskScore < 30 {
		if captchaType == CaptchaTypeSeamless || captchaType == CaptchaTypeSlider {
			score += riskWeight * 100
		}
	}

	if req != nil && req.DeviceTrust > 0.7 {
		if captchaType == CaptchaTypeClick || captchaType == CaptchaTypeSlider {
			score += 5
		}
	}

	return score
}

func (s *AIRecommendationService) getTopPreferredTypes(history *AIRecommendationUserHistory) []CaptchaType {
	if history == nil || len(history.PreferredTypes) == 0 {
		return []CaptchaType{}
	}

	type count struct {
		captchaType CaptchaType
		count      int
	}

	var counts []count
	for t, c := range history.PreferredTypes {
		counts = append(counts, count{t, c})
	}

	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	result := make([]CaptchaType, len(counts))
	for i, c := range counts {
		result[i] = c.captchaType
	}

	return result
}

func (s *AIRecommendationService) calculateDifficulty(captchaType CaptchaType, riskScore float64, history *AIRecommendationUserHistory, req *CaptchaRecommendationRequest) CaptchaDifficulty {
	difficulty := CaptchaDifficulty{
		TimeLimit: 30,
	}

	if history != nil && history.TotalAttempts > 5 {
		successRateFactor := 0.0
		if history.SuccessRate > 0.9 {
			successRateFactor = -0.2
		} else if history.SuccessRate > 0.7 {
			successRateFactor = 0.0
		} else if history.SuccessRate > 0.5 {
			successRateFactor = 0.15
		} else {
			successRateFactor = 0.3
		}

		avgDurationFactor := 0.0
		if history.AvgDuration > 10000 {
			avgDurationFactor = 0.1
		} else if history.AvgDuration < 3000 {
			avgDurationFactor = -0.1
		}

		riskFactor := (riskScore - 50) / 100

		adjustment := successRateFactor + avgDurationFactor + riskFactor

		baseScore := 50.0
		newScore := baseScore + adjustment*30
		newScore = math.Max(math.Min(newScore, 100), 10)

		difficulty.Score = math.Round(newScore*100) / 100

		if newScore < 30 {
			difficulty.Level = "easy"
			difficulty.TimeLimit = 45
		} else if newScore < 60 {
			difficulty.Level = "medium"
			difficulty.TimeLimit = 30
		} else if newScore < 80 {
			difficulty.Level = "hard"
			difficulty.TimeLimit = 20
		} else {
			difficulty.Level = "extreme"
			difficulty.TimeLimit = 15
		}
	} else {
		difficulty.Score = 50
		if riskScore < 30 {
			difficulty.Level = "easy"
			difficulty.TimeLimit = 45
		} else if riskScore < 60 {
			difficulty.Level = "medium"
			difficulty.TimeLimit = 30
		} else {
			difficulty.Level = "hard"
			difficulty.TimeLimit = 20
		}
	}

	switch captchaType {
	case CaptchaTypeSlider:
		baseOffset := 50
		if difficulty.Level == "easy" {
			difficulty.SliderOffset = int(float64(baseOffset) * 0.7)
		} else if difficulty.Level == "medium" {
			difficulty.SliderOffset = baseOffset
		} else if difficulty.Level == "hard" {
			difficulty.SliderOffset = int(float64(baseOffset) * 1.3)
		} else {
			difficulty.SliderOffset = int(float64(baseOffset) * 1.5)
		}

	case CaptchaTypeClick:
		baseCount := 4
		if difficulty.Level == "easy" {
			difficulty.ClickCount = baseCount - 1
		} else if difficulty.Level == "medium" {
			difficulty.ClickCount = baseCount
		} else if difficulty.Level == "hard" {
			difficulty.ClickCount = baseCount + 2
		} else {
			difficulty.ClickCount = baseCount + 4
		}

	case CaptchaType3D:
		basePieces := 6
		if difficulty.Level == "easy" {
			difficulty.JigsawPieces = basePieces - 2
		} else if difficulty.Level == "medium" {
			difficulty.JigsawPieces = basePieces
		} else if difficulty.Level == "hard" {
			difficulty.JigsawPieces = basePieces + 2
		} else {
			difficulty.JigsawPieces = basePieces + 4
		}
	}

	return difficulty
}

func (s *AIRecommendationService) getBaseDifficulty(captchaType CaptchaType) CaptchaDifficulty {
	base := CaptchaDifficulty{
		Score:     50,
		Level:     "medium",
		TimeLimit: 30,
	}

	switch captchaType {
	case CaptchaTypeSlider:
		base.SliderOffset = 50
	case CaptchaTypeClick:
		base.ClickCount = 4
	case CaptchaType3D:
		base.JigsawPieces = 6
	}

	return base
}

func (s *AIRecommendationService) generateAlternatives(primary CaptchaType, history *AIRecommendationUserHistory, riskScore float64, req *CaptchaRecommendationRequest) []AlternativeCaptcha {
	var alternatives []AlternativeCaptcha

	types := []CaptchaType{CaptchaTypeSlider, CaptchaTypeClick, CaptchaType3D, CaptchaTypeLianLianKan}
	for _, t := range types {
		if t == primary {
			continue
		}

		stats := defaultCaptchaStats[t]
		if stats == nil {
			continue
		}

		score := s.calculateCaptchaTypeScore(t, stats, history, riskScore, req)

		reason := s.generateAlternativeReason(t, score, history, riskScore)

		alternatives = append(alternatives, AlternativeCaptcha{
			Type:   t,
			Score:  math.Round(score*100) / 100,
			Reason: reason,
		})
	}

	sort.Slice(alternatives, func(i, j int) bool {
		return alternatives[i].Score > alternatives[j].Score
	})

	if len(alternatives) > 3 {
		alternatives = alternatives[:3]
	}

	return alternatives
}

func (s *AIRecommendationService) generateAlternativeReason(captchaType CaptchaType, score float64, history *AIRecommendationUserHistory, riskScore float64) string {
	if history != nil {
		if count, exists := history.PreferredTypes[captchaType]; exists && count > 3 {
			return "您之前成功使用过此类型验证"
		}
	}

	switch captchaType {
	case CaptchaTypeSlider:
		if riskScore < 50 {
			return "低风险环境下用户友好度高"
		}
		return "操作简单，适合大多数用户"
	case CaptchaTypeClick:
		if riskScore > 50 {
			return "可有效区分机器人和真人"
		}
		return "视觉效果直观"
	case CaptchaType3D:
		if riskScore > 60 {
			return "高安全性，适合中高风险场景"
		}
		return "用户体验流畅"
	case CaptchaTypeLianLianKan:
		if riskScore < 30 {
			return "娱乐性强，用户接受度高"
		}
		return "识别准确率高"
	}

	return "综合评估推荐"
}

func (s *AIRecommendationService) analyzeRecommendationFactors(captchaType CaptchaType, riskScore float64, history *AIRecommendationUserHistory, req *CaptchaRecommendationRequest) []RecommendationFactor {
	var factors []RecommendationFactor

	factors = append(factors, RecommendationFactor{
		Name:   "risk_score",
		Weight: 0.35,
		Impact: 0.35,
		Score:  riskScore,
	})

	if history != nil {
		factors = append(factors, RecommendationFactor{
			Name:   "user_success_rate",
			Weight: 0.25,
			Impact: 0.25,
			Score:  history.SuccessRate * 100,
		})

		if count, exists := history.PreferredTypes[captchaType]; exists {
			prefScore := float64(count) / float64(history.TotalAttempts) * 100
			factors = append(factors, RecommendationFactor{
				Name:   "user_preference",
				Weight: 0.20,
				Impact: 0.20,
				Score:  prefScore,
			})
		}
	}

	if req != nil {
		if req.AccessFrequency > 10 {
			factors = append(factors, RecommendationFactor{
				Name:   "access_frequency",
				Weight: 0.10,
				Impact: 0.10,
				Score:  80,
			})
		}

		if req.DeviceTrust > 0.7 {
			factors = append(factors, RecommendationFactor{
				Name:   "device_trust",
				Weight: 0.10,
				Impact: 0.10,
				Score:  req.DeviceTrust * 100,
			})
		}
	}

	stats := defaultCaptchaStats[captchaType]
	if stats != nil {
		factors = append(factors, RecommendationFactor{
			Name:   "captcha_comfort",
			Weight: 0.15,
			Impact: 0.15,
			Score:  stats.ComfortScore * 100,
		})
	}

	return factors
}

func (s *AIRecommendationService) estimateDuration(captchaType CaptchaType, difficulty CaptchaDifficulty) int64 {
	baseDuration := int64(5000)

	switch captchaType {
	case CaptchaTypeSlider:
		baseDuration = 5000
	case CaptchaTypeClick:
		baseDuration = 6000
	case CaptchaTypeGesture:
		baseDuration = 8000
	case CaptchaTypeLianLianKan:
		baseDuration = 10000
	case CaptchaTypeVoice:
		baseDuration = 7000
	case CaptchaType3D:
		baseDuration = 5500
	case CaptchaTypeSeamless:
		baseDuration = 0
	}

	switch difficulty.Level {
	case "easy":
		baseDuration = int64(float64(baseDuration) * 0.8)
	case "hard":
		baseDuration = int64(float64(baseDuration) * 1.3)
	case "extreme":
		baseDuration = int64(float64(baseDuration) * 1.5)
	}

	return baseDuration
}

func (s *AIRecommendationService) generateRecommendationReason(captchaType CaptchaType, riskScore float64, history *AIRecommendationUserHistory) string {
	if history != nil && history.TotalAttempts > 5 {
		if successRate := history.SuccessRate; successRate > 0.85 {
			if lastType := history.LastCaptchaType; lastType == captchaType {
				return "基于您的历史验证成功率和偏好自动推荐"
			}
		}
	}

	if riskScore < 30 {
		return "环境风险低，推荐用户体验友好的验证码类型"
	} else if riskScore < 60 {
		return "综合风险评估，推荐平衡安全性和用户体验的类型"
	} else {
		return "检测到较高风险，推荐高安全性的验证码类型"
	}
}

func (s *AIRecommendationService) calculateConfidence(captchaType CaptchaType, history *AIRecommendationUserHistory, riskScore float64) float64 {
	confidence := 0.5

	if history != nil && history.TotalAttempts > 10 {
		confidence += 0.2

		if _, exists := history.PreferredTypes[captchaType]; exists {
			confidence += 0.15
		}
	}

	if riskScore < 50 || riskScore > 70 {
		confidence += 0.1
	}

	if confidence > 0.95 {
		confidence = 0.95
	}

	return math.Round(confidence*100) / 100
}

func (s *AIRecommendationService) adjustDifficultyForContext(base CaptchaDifficulty, req *DifficultyRequest, history *AIRecommendationUserHistory) CaptchaDifficulty {
	difficulty := base

	if req.SuccessRate > 0 {
		if req.SuccessRate > 0.9 {
			difficulty.Score = math.Max(difficulty.Score-15, 20)
			difficulty.Level = "easy"
			difficulty.TimeLimit = 45
		} else if req.SuccessRate > 0.7 {
			difficulty.Score = math.Max(difficulty.Score-5, 30)
			difficulty.Level = "medium"
			difficulty.TimeLimit = 35
		} else if req.SuccessRate < 0.5 {
			difficulty.Score = math.Min(difficulty.Score+10, 90)
			difficulty.Level = "hard"
			difficulty.TimeLimit = 25
		}
	}

	if req.FailureCount > 3 {
		difficulty.Score = math.Min(difficulty.Score+5, 95)
		if difficulty.Level == "easy" {
			difficulty.Level = "medium"
		}
	}

	if req.RiskScore > 70 {
		difficulty.Score = math.Min(difficulty.Score+15, 100)
		difficulty.Level = "hard"
		difficulty.TimeLimit = 20
	}

	if req.TimeOfDay >= 22 || req.TimeOfDay < 6 {
		difficulty.Score = math.Max(difficulty.Score-5, 20)
	}

	switch req.CaptchaType {
	case CaptchaTypeSlider:
		if difficulty.Level == "easy" {
			difficulty.SliderOffset = int(float64(difficulty.SliderOffset) * 0.8)
		} else if difficulty.Level == "hard" {
			difficulty.SliderOffset = int(float64(difficulty.SliderOffset) * 1.3)
		}
	case CaptchaTypeClick:
		if difficulty.Level == "easy" {
			difficulty.ClickCount = int(math.Max(float64(difficulty.ClickCount)-1, 2))
		} else if difficulty.Level == "hard" {
			difficulty.ClickCount = int(math.Min(float64(difficulty.ClickCount)+2, 10))
		}
	case CaptchaType3D:
		if difficulty.Level == "easy" {
			difficulty.JigsawPieces = int(math.Max(float64(difficulty.JigsawPieces)-2, 3))
		} else if difficulty.Level == "hard" {
			difficulty.JigsawPieces = int(math.Min(float64(difficulty.JigsawPieces)+2, 12))
		}
	}

	return difficulty
}

func (s *AIRecommendationService) analyzeDifficultyFactors(difficulty CaptchaDifficulty, req *DifficultyRequest, history *AIRecommendationUserHistory) []AIRecommendationFactorDetail {
	var factors []AIRecommendationFactorDetail

	if req.SuccessRate > 0 {
		factors = append(factors, AIRecommendationFactorDetail{
			Name:     "user_success_rate",
			Impact:   0.3,
			NewValue: difficulty.Score,
			OldValue: 50,
		})
	}

	if req.RiskScore > 0 {
		riskImpact := (req.RiskScore - 50) / 100 * 30
		factors = append(factors, AIRecommendationFactorDetail{
			Name:     "risk_score",
			Impact:   riskImpact,
			NewValue: difficulty.Score,
			OldValue: difficulty.Score - riskImpact,
		})
	}

	if req.FailureCount > 0 {
		failureImpact := float64(req.FailureCount) * 2
		factors = append(factors, AIRecommendationFactorDetail{
			Name:     "failure_count",
			Impact:   failureImpact,
			NewValue: difficulty.Score,
			OldValue: difficulty.Score - failureImpact,
		})
	}

	return factors
}

func (s *AIRecommendationService) generateAdjustmentReason(difficulty CaptchaDifficulty, req *DifficultyRequest, history *AIRecommendationUserHistory) string {
	parts := []string{}

	if req.SuccessRate > 0.85 {
		parts = append(parts, "您近期验证成功率高，适度降低难度以提升体验")
	} else if req.SuccessRate < 0.5 {
		parts = append(parts, "检测到您近期验证成功率较低，已调整难度")
	}

	if req.RiskScore > 70 {
		parts = append(parts, "环境风险较高，采用更具挑战性的设置")
	} else if req.RiskScore < 30 {
		parts = append(parts, "环境安全，采用更友好的难度设置")
	}

	if req.FailureCount > 3 {
		parts = append(parts, "连续失败次数较多，已优化参数")
	}

	if len(parts) == 0 {
		parts = append(parts, "基于综合评估推荐最佳难度")
	}

	return strings.Join(parts, "；") + "。"
}

func (s *AIRecommendationService) calculateDifficultyConfidence(req *DifficultyRequest, history *AIRecommendationUserHistory) float64 {
	confidence := 0.5

	if history != nil && history.TotalAttempts > 10 {
		confidence += 0.25
	}

	if req.SuccessRate > 0 {
		confidence += 0.15
	}

	if req.RiskScore > 0 {
		confidence += 0.1
	}

	return math.Min(confidence, 0.95)
}

func (s *AIRecommendationService) convertToTrajectory(behaviorData []BehaviorDataPoint) []TrajectoryPoint {
	trajectory := make([]TrajectoryPoint, len(behaviorData))
	for i, bd := range behaviorData {
		trajectory[i] = TrajectoryPoint{
			X:         bd.X,
			Y:         bd.Y,
			Timestamp: bd.Timestamp,
		}
	}
	return trajectory
}

func (s *AIRecommendationService) cleanupOldHistory() {
	now := time.Now()
	for key, history := range s.userHistory {
		if now.Sub(history.LastVerifiedAt) > 24*time.Hour*30 {
			delete(s.userHistory, key)
		}
	}
}

func (s *AIRecommendationService) GetCaptchaTypeStats() []*CaptchaTypeRecommendationStats {
	var stats []*CaptchaTypeRecommendationStats
	for _, stat := range defaultCaptchaStats {
		stats = append(stats, stat)
	}
	return stats
}

func (s *AIRecommendationService) GetUserStats(userID, fingerprint string) *AIRecommendationUserHistory {
	userKey := s.getUserKey(userID, fingerprint)
	return s.getUserHistory(userKey)
}
