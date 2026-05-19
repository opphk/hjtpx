package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// BehaviorPattern 行为模式
type BehaviorPattern string

const (
	BehaviorNormal          BehaviorPattern = "normal"
	BehaviorFastClick       BehaviorPattern = "fast_click"
	BehaviorSlowClick       BehaviorPattern = "slow_click"
	BehaviorRandom          BehaviorPattern = "random"
	BehaviorAutomated       BehaviorPattern = "automated"
	BehaviorHuman           BehaviorPattern = "human"
)

// RealTimeBehavior 实时行为数据
type RealTimeBehavior struct {
	UserID           string
	ActionType       string
	Timestamp        time.Time
	Duration         time.Duration
	Accuracy         float64
	Deviation        float64
	Velocity         float64
	Acceleration     float64
	PathComplexity   float64
	ErrorRate        float64
	SessionCount     int
	ConsecutiveFails int
	ConsecutiveSuccess int
}

// DifficultyAdjustment 难度调整
type DifficultyAdjustment struct {
	UserID           string
	PreviousDifficulty DifficultyLevel
	NewDifficulty     DifficultyLevel
	AdjustmentAmount  float64
	Reason           string
	Confidence       float64
	Timestamp        time.Time
	BehaviorPattern  BehaviorPattern
}

// EnhancedDynamicDifficultyService 增强版动态难度调整服务
type EnhancedDynamicDifficultyService struct {
	userProfiles     map[string]*DynamicUserProfile
	behaviorAnalyzer *BehaviorAnalyzer
	adjustmentEngine *AdjustmentEngine
	config           *DynamicDifficultyConfig
	mu               sync.RWMutex
}

// DynamicUserProfile 动态用户档案
type DynamicUserProfile struct {
	UserID                  string
	CurrentDifficulty       DifficultyLevel
	BaseDifficulty          DifficultyLevel
	BehaviorHistory         []*RealTimeBehavior
	AdjustmentHistory       []*DifficultyAdjustment
	RiskScore               float64
	BehaviorPattern         BehaviorPattern
	ConsistencyScore        float64
	LearningProgress        float64
	LastAdjustmentTime      time.Time
	SessionStart            time.Time
	ActionCount             int
	SuccessCount            int
	FailureCount            int
	ConsecutiveFails        int
	ConsecutiveSuccess      int
}

// BehaviorAnalyzer 行为分析器
type BehaviorAnalyzer struct {
	patternRecognizers []DifficultyPatternRecognizer
	anomalyDetector   *AnomalyDetector
}

// DifficultyPatternRecognizer 难度调整模式识别器接口
type DifficultyPatternRecognizer interface {
	Recognize(behavior *RealTimeBehavior) (BehaviorPattern, float64)
}

// AdjustmentEngine 调整引擎
type AdjustmentEngine struct {
	rules           []AdjustmentRule
	learningRate    float64
	smoothingFactor float64
}

// AdjustmentRule 调整规则
type AdjustmentRule struct {
	Name           string
	Condition      func(profile *DynamicUserProfile) bool
	Adjustment     func(profile *DynamicUserProfile) float64
	Reason         string
	Priority       int
}

// DynamicDifficultyConfig 动态难度配置
type DynamicDifficultyConfig struct {
	MinAdjustmentInterval time.Duration
	MaxAdjustmentPerSession int
	ConfidenceThreshold    float64
	AnomalySensitivity     float64
	LearningRate           float64
	SmoothingFactor        float64
	BehaviorWindowSize     int
	MaxBehaviorHistory     int
}

// NewEnhancedDynamicDifficultyService 创建增强版动态难度服务
func NewEnhancedDynamicDifficultyService() *EnhancedDynamicDifficultyService {
	service := &EnhancedDynamicDifficultyService{
		userProfiles: make(map[string]*DynamicUserProfile),
		config: &DynamicDifficultyConfig{
			MinAdjustmentInterval: 30 * time.Second,
			MaxAdjustmentPerSession: 3,
			ConfidenceThreshold:    0.7,
			AnomalySensitivity:     2.0,
			LearningRate:           0.15,
			SmoothingFactor:        0.3,
			BehaviorWindowSize:     10,
			MaxBehaviorHistory:     100,
		},
	}
	service.behaviorAnalyzer = service.createBehaviorAnalyzer()
	service.adjustmentEngine = service.createAdjustmentEngine()
	return service
}

// 创建行为分析器
func (s *EnhancedDynamicDifficultyService) createBehaviorAnalyzer() *BehaviorAnalyzer {
	return &BehaviorAnalyzer{
		patternRecognizers: []DifficultyPatternRecognizer{
			&SpeedPatternRecognizer{},
			&AccuracyPatternRecognizer{},
			&PathPatternRecognizer{},
			&ConsistencyPatternRecognizer{},
		},
		anomalyDetector: NewAnomalyDetector(),
	}
}

// 创建调整引擎
func (s *EnhancedDynamicDifficultyService) createAdjustmentEngine() *AdjustmentEngine {
	return &AdjustmentEngine{
		rules: []AdjustmentRule{
			{
				Name:     "consecutive_fails",
				Priority: 10,
				Condition: func(p *DynamicUserProfile) bool {
					return p.ConsecutiveFails >= 2
				},
				Adjustment: func(p *DynamicUserProfile) float64 {
					return -0.5 * float64(p.ConsecutiveFails)
				},
				Reason: "连续失败，降低难度",
			},
			{
				Name:     "consecutive_success",
				Priority: 9,
				Condition: func(p *DynamicUserProfile) bool {
					return p.ConsecutiveSuccess >= 3
				},
				Adjustment: func(p *DynamicUserProfile) float64 {
					return 0.3 * float64(p.ConsecutiveSuccess)
				},
				Reason: "连续成功，提升难度",
			},
			{
				Name:     "automated_pattern",
				Priority: 8,
				Condition: func(p *DynamicUserProfile) bool {
					return p.BehaviorPattern == BehaviorAutomated
				},
				Adjustment: func(p *DynamicUserProfile) float64 {
					return 1.5
				},
				Reason: "检测到自动化行为，大幅提升难度",
			},
			{
				Name:     "slow_response",
				Priority: 7,
				Condition: func(p *DynamicUserProfile) bool {
					return p.ConsistencyScore < 0.3
				},
				Adjustment: func(p *DynamicUserProfile) float64 {
					return -0.3
				},
				Reason: "响应不一致，降低难度",
			},
			{
				Name:     "high_risk",
				Priority: 6,
				Condition: func(p *DynamicUserProfile) bool {
					return p.RiskScore > 70
				},
				Adjustment: func(p *DynamicUserProfile) float64 {
					return (p.RiskScore - 70) / 30
				},
				Reason: "高风险评分，提升难度",
			},
			{
				Name:     "learning_progress",
				Priority: 5,
				Condition: func(p *DynamicUserProfile) bool {
					return p.LearningProgress > 0.7
				},
				Adjustment: func(p *DynamicUserProfile) float64 {
					return (p.LearningProgress - 0.7) * 2
				},
				Reason: "学习进度良好，适当提升难度",
			},
		},
		learningRate:    0.15,
		smoothingFactor: 0.3,
	}
}

// GetDifficulty 获取当前难度
func (s *EnhancedDynamicDifficultyService) GetDifficulty(userID string) DifficultyLevel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile := s.getOrCreateProfile(userID)
	return profile.CurrentDifficulty
}

// UpdateBehavior 更新行为数据
func (s *EnhancedDynamicDifficultyService) UpdateBehavior(ctx context.Context, behavior *RealTimeBehavior) (*DifficultyAdjustment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.getOrCreateProfile(behavior.UserID)
	
	profile.BehaviorHistory = append(profile.BehaviorHistory, behavior)
	if len(profile.BehaviorHistory) > s.config.MaxBehaviorHistory {
		profile.BehaviorHistory = profile.BehaviorHistory[len(profile.BehaviorHistory)-s.config.MaxBehaviorHistory:]
	}

	profile.ActionCount++
	if behavior.Accuracy > 0.8 {
		profile.SuccessCount++
		profile.ConsecutiveSuccess++
		profile.ConsecutiveFails = 0
	} else {
		profile.FailureCount++
		profile.ConsecutiveFails++
		profile.ConsecutiveSuccess = 0
	}

	pattern, confidence := s.analyzeBehavior(profile)
	profile.BehaviorPattern = pattern
	profile.ConsistencyScore = confidence

	s.updateLearningProgress(profile)

	return s.evaluateAdjustment(profile)
}

// 获取或创建用户档案
func (s *EnhancedDynamicDifficultyService) getOrCreateProfile(userID string) *DynamicUserProfile {
	profile, exists := s.userProfiles[userID]
	if !exists {
		profile = &DynamicUserProfile{
			UserID:            userID,
			CurrentDifficulty: DifficultyMedium,
			BaseDifficulty:    DifficultyMedium,
			BehaviorHistory:   make([]*RealTimeBehavior, 0),
			AdjustmentHistory: make([]*DifficultyAdjustment, 0),
			RiskScore:         50,
			BehaviorPattern:   BehaviorNormal,
			ConsistencyScore:  0.5,
			LearningProgress:  0,
			SessionStart:      time.Now(),
		}
		s.userProfiles[userID] = profile
	}
	return profile
}

// 分析行为
func (s *EnhancedDynamicDifficultyService) analyzeBehavior(profile *DynamicUserProfile) (BehaviorPattern, float64) {
	if len(profile.BehaviorHistory) < 3 {
		return BehaviorNormal, 0.5
	}

	startIdx := len(profile.BehaviorHistory) - s.config.BehaviorWindowSize
if startIdx < 0 {
	startIdx = 0
}
window := profile.BehaviorHistory[startIdx:]
	
	var finalPattern BehaviorPattern
	maxConfidence := 0.0

	for _, recognizer := range s.behaviorAnalyzer.patternRecognizers {
		for _, behavior := range window {
			pattern, confidence := recognizer.Recognize(behavior)
			if confidence > maxConfidence {
				maxConfidence = confidence
				finalPattern = pattern
			}
		}
	}

	if maxConfidence < s.config.ConfidenceThreshold {
		finalPattern = BehaviorNormal
	}

	return finalPattern, maxConfidence
}

// 更新学习进度
func (s *EnhancedDynamicDifficultyService) updateLearningProgress(profile *DynamicUserProfile) {
	if profile.ActionCount < 5 {
		return
	}

	successRate := float64(profile.SuccessCount) / float64(profile.ActionCount)
	
	recentStartIdx := len(profile.BehaviorHistory) - 5
if recentStartIdx < 0 {
	recentStartIdx = 0
}
recentWindow := profile.BehaviorHistory[recentStartIdx:]
	recentSuccess := 0
	for _, b := range recentWindow {
		if b.Accuracy > 0.8 {
			recentSuccess++
		}
	}
	recentRate := float64(recentSuccess) / float64(len(recentWindow))

	improvement := recentRate - successRate
	profile.LearningProgress = math.Min(1.0, profile.LearningProgress+improvement*0.1)
}

// 评估难度调整
func (s *EnhancedDynamicDifficultyService) evaluateAdjustment(profile *DynamicUserProfile) (*DifficultyAdjustment, error) {
	if time.Since(profile.LastAdjustmentTime) < s.config.MinAdjustmentInterval {
		return nil, nil
	}

	adjustmentCount := 0
	for _, adj := range profile.AdjustmentHistory {
		if adj.Timestamp.After(profile.SessionStart) {
			adjustmentCount++
		}
	}
	if adjustmentCount >= s.config.MaxAdjustmentPerSession {
		return nil, nil
	}

	sort.Slice(s.adjustmentEngine.rules, func(i, j int) bool {
		return s.adjustmentEngine.rules[i].Priority > s.adjustmentEngine.rules[j].Priority
	})

	var totalAdjustment float64
	var reasons []string

	for _, rule := range s.adjustmentEngine.rules {
		if rule.Condition(profile) {
			adjustment := rule.Adjustment(profile)
			totalAdjustment += adjustment
			reasons = append(reasons, rule.Reason)
		}
	}

	if math.Abs(totalAdjustment) < 0.1 {
		return nil, nil
	}

	baseScore := s.difficultyToScore(profile.CurrentDifficulty)
	newScore := baseScore + totalAdjustment*s.adjustmentEngine.learningRate
	newScore = math.Max(0, math.Min(3, newScore))

	newDifficulty := s.scoreToDifficulty(newScore)
	
	if newDifficulty == profile.CurrentDifficulty {
		return nil, nil
	}

	adjustment := &DifficultyAdjustment{
		UserID:           profile.UserID,
		PreviousDifficulty: profile.CurrentDifficulty,
		NewDifficulty:     newDifficulty,
		AdjustmentAmount:  totalAdjustment,
		Reason:           fmt.Sprintf("%s (置信度: %.2f)", reasons[0], profile.ConsistencyScore),
		Confidence:       profile.ConsistencyScore,
		Timestamp:        time.Now(),
		BehaviorPattern:  profile.BehaviorPattern,
	}

	profile.CurrentDifficulty = newDifficulty
	profile.LastAdjustmentTime = time.Now()
	profile.AdjustmentHistory = append(profile.AdjustmentHistory, adjustment)

	if len(profile.AdjustmentHistory) > 50 {
		profile.AdjustmentHistory = profile.AdjustmentHistory[len(profile.AdjustmentHistory)-50:]
	}

	return adjustment, nil
}

// 获取用户档案
func (s *EnhancedDynamicDifficultyService) GetUserProfile(userID string) (*DynamicUserProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile, exists := s.userProfiles[userID]
	if !exists {
		return nil, fmt.Errorf("profile not found for user: %s", userID)
	}
	return profile, nil
}

// 设置风险评分
func (s *EnhancedDynamicDifficultyService) SetRiskScore(userID string, riskScore float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.getOrCreateProfile(userID)
	profile.RiskScore = math.Max(0, math.Min(100, riskScore))
}

// 获取调整历史
func (s *EnhancedDynamicDifficultyService) GetAdjustmentHistory(userID string) []*DifficultyAdjustment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile, exists := s.userProfiles[userID]
	if !exists {
		return nil
	}
	return profile.AdjustmentHistory
}

// 重置用户会话
func (s *EnhancedDynamicDifficultyService) ResetSession(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.getOrCreateProfile(userID)
	profile.SessionStart = time.Now()
	profile.ConsecutiveFails = 0
	profile.ConsecutiveSuccess = 0
}

// 清理过期会话
func (s *EnhancedDynamicDifficultyService) CleanupExpiredSessions(ctx context.Context, maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	now := time.Now()
	for userID, profile := range s.userProfiles {
		if now.Sub(profile.SessionStart) > maxAge {
			delete(s.userProfiles, userID)
			count++
		}
	}
	return count
}

// 难度转分数
func (s *EnhancedDynamicDifficultyService) difficultyToScore(d DifficultyLevel) float64 {
	switch d {
	case DifficultyEasy:
		return 0
	case DifficultyMedium:
		return 1
	case DifficultyHard:
		return 2
	case DifficultyExpert:
		return 3
	default:
		return 1
	}
}

// 分数转难度
func (s *EnhancedDynamicDifficultyService) scoreToDifficulty(score float64) DifficultyLevel {
	if score < 0.5 {
		return DifficultyEasy
	} else if score < 1.5 {
		return DifficultyMedium
	} else if score < 2.5 {
		return DifficultyHard
	} else {
		return DifficultyExpert
	}
}

// SpeedPatternRecognizer 速度模式识别器
type SpeedPatternRecognizer struct{}

func (r *SpeedPatternRecognizer) Recognize(behavior *RealTimeBehavior) (BehaviorPattern, float64) {
	if behavior.Duration < time.Second {
		return BehaviorFastClick, 0.8
	}
	if behavior.Duration > 30*time.Second {
		return BehaviorSlowClick, 0.7
	}
	return BehaviorNormal, 0.3
}

// AccuracyPatternRecognizer 准确率模式识别器
type AccuracyPatternRecognizer struct{}

func (r *AccuracyPatternRecognizer) Recognize(behavior *RealTimeBehavior) (BehaviorPattern, float64) {
	if behavior.ErrorRate > 0.5 {
		return BehaviorRandom, 0.75
	}
	if behavior.Accuracy > 0.95 && behavior.Duration < 2*time.Second {
		return BehaviorAutomated, 0.85
	}
	if behavior.Accuracy > 0.8 && behavior.Duration > 3*time.Second {
		return BehaviorHuman, 0.7
	}
	return BehaviorNormal, 0.2
}

// PathPatternRecognizer 路径模式识别器
type PathPatternRecognizer struct{}

func (r *PathPatternRecognizer) Recognize(behavior *RealTimeBehavior) (BehaviorPattern, float64) {
	if behavior.PathComplexity < 0.1 {
		return BehaviorAutomated, 0.7
	}
	if behavior.PathComplexity > 0.8 {
		return BehaviorHuman, 0.6
	}
	return BehaviorNormal, 0.3
}

// ConsistencyPatternRecognizer 一致性模式识别器
type ConsistencyPatternRecognizer struct{}

func (r *ConsistencyPatternRecognizer) Recognize(behavior *RealTimeBehavior) (BehaviorPattern, float64) {
	if behavior.Deviation < 0.05 && behavior.Velocity > 10 {
		return BehaviorAutomated, 0.8
	}
	if behavior.Deviation > 0.3 {
		return BehaviorRandom, 0.6
	}
	return BehaviorNormal, 0.4
}

// GetAnalysisReport 获取分析报告
func (s *EnhancedDynamicDifficultyService) GetAnalysisReport(userID string) *DynamicDifficultyReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile, exists := s.userProfiles[userID]
	if !exists {
		return nil
	}

	actionCount := profile.ActionCount
	if actionCount == 0 {
		actionCount = 1
	}
	
	report := &DynamicDifficultyReport{
		UserID:              userID,
		CurrentDifficulty:   profile.CurrentDifficulty,
		BaseDifficulty:      profile.BaseDifficulty,
		RiskScore:           profile.RiskScore,
		BehaviorPattern:     profile.BehaviorPattern,
		ConsistencyScore:    profile.ConsistencyScore,
		LearningProgress:    profile.LearningProgress,
		ActionCount:         profile.ActionCount,
		SuccessRate:         float64(profile.SuccessCount) / float64(actionCount),
		ConsecutiveSuccess:  profile.ConsecutiveSuccess,
		ConsecutiveFails:    profile.ConsecutiveFails,
		AdjustmentCount:     len(profile.AdjustmentHistory),
		SessionDuration:     time.Since(profile.SessionStart),
	}

	return report
}

// DynamicDifficultyReport 动态难度报告
type DynamicDifficultyReport struct {
	UserID              string
	CurrentDifficulty   DifficultyLevel
	BaseDifficulty      DifficultyLevel
	RiskScore           float64
	BehaviorPattern     BehaviorPattern
	ConsistencyScore    float64
	LearningProgress    float64
	ActionCount         int
	SuccessRate         float64
	ConsecutiveSuccess  int
	ConsecutiveFails    int
	AdjustmentCount     int
	SessionDuration     time.Duration
}

// BatchUpdateBehavior 批量更新行为数据
func (s *EnhancedDynamicDifficultyService) BatchUpdateBehavior(ctx context.Context, behaviors []*RealTimeBehavior) ([]*DifficultyAdjustment, error) {
	var adjustments []*DifficultyAdjustment
	for _, behavior := range behaviors {
		adj, err := s.UpdateBehavior(ctx, behavior)
		if err != nil {
			return adjustments, err
		}
		if adj != nil {
			adjustments = append(adjustments, adj)
		}
	}
	return adjustments, nil
}

// GetGlobalStats 获取全局统计
func (s *EnhancedDynamicDifficultyService) GetGlobalStats() *GlobalDifficultyStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &GlobalDifficultyStats{
		TotalUsers:           len(s.userProfiles),
		DifficultyDistribution: make(map[DifficultyLevel]int),
		BehaviorPatterns:     make(map[BehaviorPattern]int),
	}

	for _, profile := range s.userProfiles {
		stats.DifficultyDistribution[profile.CurrentDifficulty]++
		stats.BehaviorPatterns[profile.BehaviorPattern]++
		stats.TotalActions += profile.ActionCount
		stats.TotalSuccesses += profile.SuccessCount
	}

	return stats
}

// GlobalDifficultyStats 全局难度统计
type GlobalDifficultyStats struct {
	TotalUsers             int
	TotalActions           int
	TotalSuccesses         int
	DifficultyDistribution map[DifficultyLevel]int
	BehaviorPatterns       map[BehaviorPattern]int
}

