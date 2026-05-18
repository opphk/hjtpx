package service

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// 类型别名以保持与旧代码的兼容性
type PersonalizedProfile = EnhancedPersonalizedProfile
type HistoricalDataAnalyzer = EnhancedHistoricalDataAnalyzer
type UserHistoricalData = EnhancedUserHistoricalData
type DifficultyAnalytics = EnhancedDifficultyAnalytics

type EnhancedAdaptiveDifficultyService struct {
	*AdaptiveDifficultyService
	historicalAnalyzer    *EnhancedHistoricalDataAnalyzer
	realtimeAdjuster      *RealtimeDifficultyAdjuster
	personalizationEngine *PersonalizationEngine
	mu                    sync.RWMutex
}

// EnhancedHistoricalDataAnalyzer 增强版历史数据分析器
type EnhancedHistoricalDataAnalyzer struct {
	userHistory        map[string]*EnhancedUserHistoricalData
	analyticsDB        map[string]*EnhancedDifficultyAnalytics
	patternRecognizer  *PatternRecognizer
	timeSeriesAnalyzer *TimeSeriesAnalyzer
	cohortAnalyzer     *CohortAnalyzer
	mu                 sync.RWMutex
}

// EnhancedUserHistoricalData 增强版用户历史数据
type EnhancedUserHistoricalData struct {
	UserID               string
	VerificationHistory  [][]VerificationAttempt
	SuccessPatterns      []SuccessPattern
	FailurePatterns      []FailurePattern
	PreferredTimes       map[int][]time.Time
	AvgDifficulty        float64
	TotalAttempts         int
	SuccessCount          int
	SuccessRate           float64
	AvgTimeByDifficulty   map[DifficultyLevel]float64
	LastUpdated          time.Time
	// 增强字段
	DeviceFingerprint   string
	IPAddress           string
	GeographicLocation  string
	SessionCount        int
	AvgSessionLength    float64
	SessionHistory      []SessionData
	MethodPreference    map[string]int
	DifficultyProgress  []DifficultyProgress
	BehavioralBiometrics *BehavioralBiometrics
	ResponseTimeTrend   []float64
	ErrorDistribution   map[string]int
	TimeZonePattern      map[string]int
	DeviceSwitchCount    int
	LastDeviceID         string
	NetworkQuality       float64
}

// SessionData 会话数据
type SessionData struct {
	SessionID     string
	StartTime     time.Time
	EndTime       time.Time
	Attempts      int
	SuccessCount  int
	AvgDifficulty float64
	DeviceInfo    string
	IPAddress     string
}

// DifficultyProgress 难度进度追踪
type DifficultyProgress struct {
	PreviousDifficulty DifficultyLevel
	NewDifficulty     DifficultyLevel
	Timestamp         time.Time
	Reason             string
	SuccessAfterChange bool
}

// BehavioralBiometrics 行为生物识别数据
type BehavioralBiometrics struct {
	TypingSpeed       float64
	MouseSpeed        float64
	ClickPressure     float64
	SwipePattern      float64
	HoldTime          float64
	InterKeyInterval  float64
	TrajectoryEntropy float64
}

// PatternRecognizer 模式识别器
type PatternRecognizer struct {
	patterns      map[string]*RecognizedPattern
	learningRate  float64
	mu            sync.RWMutex
}

// RecognizedPattern 识别的模式
type RecognizedPattern struct {
	PatternType   string
	Confidence    float64
	Frequency     int
	FirstSeen     time.Time
	LastSeen      time.Time
	AssociatedRisk float64
}

// TimeSeriesAnalyzer 时间序列分析器
type TimeSeriesAnalyzer struct {
	seriesData     map[string][]TimeSeriesPoint
	windowSize     int
	smoothingFactor float64
	mu             sync.RWMutex
}

// TimeSeriesPoint 时间序列数据点
type TimeSeriesPoint struct {
	Timestamp   time.Time
	Value       float64
	IsAnomaly   bool
	Seasonality float64
	Trend       float64
}

// CohortAnalyzer 队列分析器
type CohortAnalyzer struct {
	cohorts      map[string]*CohortData
	cohortWindow time.Duration
	mu           sync.RWMutex
}

// CohortData 队列数据
type CohortData struct {
	CohortID        string
	CohortType      string
	Users           int
	TotalAttempts   int
	SuccessRate     float64
	AvgDifficulty   float64
	RetentionRate   float64
	ChurnRate       float64
	AvgLifetime     float64
}

// EnhancedDifficultyAnalytics 增强版难度分析数据
type EnhancedDifficultyAnalytics struct {
	TotalAttempts          int
	SuccessRate            float64
	AvgTime                float64
	MedianTime             float64
	MinTime                float64
	MaxTime                float64
	TimeVariance           float64
	DifficultyDistribution map[DifficultyLevel]int
	SuccessByHour          map[int]int
	Trend                  string
	// 新增分析维度
	TimeOfDayPattern       map[int]float64
	WeekdayPattern         map[int]float64
	DevicePattern          map[string]float64
	NetworkPattern         map[string]float64
	SuccessRateVariance    float64
	ConsistencyScore       float64
	ProgressVelocity       float64
	AnomalyScore           float64
	RiskTrend              float64
	SessionEngagement      float64
	DifficultyAdaptationRate float64
	TimeToSuccess          float64
	AttemptsPerSession     float64
	MethodEffectiveness    map[string]float64
	UserSegment            string
	EngagementLevel        string
	PredictedChurnRisk     float64
}

// RealtimeDifficultyAdjuster 实时难度调整器
type RealtimeDifficultyAdjuster struct {
	currentAdjustments map[string]*Adjustment
	config             *RealtimeConfig
	smoothingEngine    *SmoothingEngine
	anomalyDetector    *AnomalyDetector
	mu                 sync.RWMutex
}

// SmoothingEngine 平滑引擎
type SmoothingEngine struct {
	smoothingWindow   int
	smoothingFactor   float64
	historyBuffer     map[string][]float64
	mu                sync.RWMutex
}

// Adjustment 调整记录
type Adjustment struct {
	UserID             string
	BaseDifficulty     DifficultyLevel
	Adjustment         float64
	Reason             string
	ValidUntil         time.Time
	ConsecutiveOK      int
	ConsecutiveFail    int
	SmoothedAdjustment float64
	RawAdjustments     []float64
	TransitionState    TransitionState
}

// TransitionState 过渡状态
type TransitionState struct {
	FromDifficulty  DifficultyLevel
	ToDifficulty    DifficultyLevel
	TransitionRate  float64
	StepsRemaining  int
	IsTransitioning bool
	StartTime       time.Time
}

// AnomalyDetector 异常检测器
type AnomalyDetector struct {
	baselineMetrics map[string]*BaselineMetrics
	sensitivity     float64
	mu              sync.RWMutex
}

// BaselineMetrics 基线指标
type BaselineMetrics struct {
	AvgSuccessRate float64
	AvgTime        float64
	AvgDifficulty  float64
	StdDeviation   float64
	UpperBound     float64
	LowerBound     float64
	LastUpdated    time.Time
}

// RealtimeConfig 实时配置
type RealtimeConfig struct {
	AdjustmentWindow           time.Duration
	MaxAdjustment              float64
	ConsecutiveOKThreshold     int
	ConsecutiveFailThreshold   int
	CooldownPeriod             time.Duration
	SmoothingWindow            int
	SmoothingFactor            float64
	AnomalyThreshold           float64
	TransitionSteps            int
	TransitionDelay            time.Duration
}

// PersonalizationEngine 个性化引擎
type PersonalizationEngine struct {
	userProfiles      map[string]*EnhancedPersonalizedProfile
	globalStats        *GlobalStats
	userSegmentation   *UserSegmentationEngine
	experienceManager  *ExperienceBalanceManager
	mu                 sync.RWMutex
}

// EnhancedPersonalizedProfile 增强版个性化档案
type EnhancedPersonalizedProfile struct {
	UserID               string
	PreferredMethod      string
	PreferredDifficulty  DifficultyLevel
	OptimalHours         []int
	OptimalDays          []int
	AvgSuccessRate       float64
	AvgTime              float64
	ComfortZone          DifficultyLevel
	ChallengeLevel       DifficultyLevel
	SuccessHistory       []SuccessRecord
	LastUpdated          time.Time
	// 增强字段
	UserSegment          string
	ExperienceLevel      int
	SkillScore          float64
	PreferredDevice      string
	PreferredNetwork     string
	EngagementScore      float64
	ChurnRisk           float64
	LearningProgress     float64
	AdaptiveRate        float64
	StressLevel         float64
	PreferredTimeSlots   map[string]int
	ContextualPreferences map[string]interface{}
	SafetyBalance       float64
	ExperienceBalance   float64
}

// UserSegmentationEngine 用户细分引擎
type UserSegmentationEngine struct {
	segments map[string]*UserSegment
	mu       sync.RWMutex
}

// UserSegment 用户细分
type UserSegment struct {
	SegmentID        string
	SegmentName     string
	UserCount       int
	AvgSuccessRate  float64
	AvgDifficulty   float64
	Characteristics map[string]interface{}
	RiskProfile     *DifficultyRiskProfile
}

// DifficultyRiskProfile 难度风险档案（与预测服务中的 RiskProfile 区分）
type DifficultyRiskProfile struct {
	BaseRisk       float64
	BehavioralRisk float64
	ContextualRisk float64
	DeviceRisk     float64
	NetworkRisk    float64
	TotalRisk      float64
}

// ExperienceBalanceManager 体验平衡管理器
type ExperienceBalanceManager struct {
	experienceMetrics map[string]*ExperienceMetrics
	balanceThresholds *BalanceThresholds
	mu                sync.RWMutex
}

// ExperienceMetrics 体验指标
type ExperienceMetrics struct {
	UserID              string
	SuccessRateTrend    float64
	TimeTrend           float64
	SatisfactionScore   float64
	FrustrationLevel    float64
	CompletionRate      float64
	DropoutRisk         float64
	OverallExperience   float64
}

// BalanceThresholds 平衡阈值
type BalanceThresholds struct {
	MinSuccessRate      float64
	MaxFrustrationLevel float64
	MinSatisfactionScore float64
	MaxTimeAllowed      float64
	MinCompletionRate   float64
}

// GlobalStats 全局统计
type GlobalStats struct {
	TotalUsers             int
	AvgSuccessRate         float64
	MostPopularMethod      string
	DifficultyDistribution map[DifficultyLevel]float64
	TimePatterns           map[int]float64
	SegmentDistribution    map[string]float64
	AnomalyRate            float64
}

// SuccessRecord 成功记录
type SuccessRecord struct {
	Timestamp        time.Time
	Difficulty       DifficultyLevel
	Method           string
	TimeSpent        float64
	UserSatisfaction float64
}

// EnhancedVerificationContext 增强版验证上下文
type EnhancedVerificationContext struct {
	SessionID           string
	DeviceFingerprint   string
	DeviceID            string
	IPAddress           string
	GeographicLocation  string
	NetworkQuality      float64
	TimeZone            string
	UserAgent           string
	Platform            string
	Browser             string
	IsMobile            bool
	IsTor               bool
	IsVPN               bool
	IsProxy             bool
}

// EnhancedUserAnalyticsReport 增强版用户分析报告
type EnhancedUserAnalyticsReport struct {
	UserID             string
	TotalAttempts      int
	SuccessCount       int
	SuccessRate        float64
	AvgTime            float64
	MedianTime         float64
	Trend              string
	PreferredMethod     string
	OptimalTimes       []int
	AvgDifficulty      float64
	UserSegment        string
	EngagementLevel    string
	PredictedChurnRisk float64
	SessionEngagement  float64
	ConsistencyScore   float64
	ProgressVelocity   float64
	SkillScore         float64
	ExperienceLevel    int
	SatisfactionScore  float64
	FrustrationLevel   float64
	DevicePreference   string
	NetworkPreference  string
}

// GetGlobalStats 获取全局统计
func (pe *PersonalizationEngine) GetGlobalStats() *GlobalStats {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.globalStats
}

// UpdateGlobalStats 更新全局统计
func (pe *PersonalizationEngine) UpdateGlobalStats() {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	totalUsers := len(pe.userProfiles)
	if totalUsers == 0 {
		return
	}
	
	var totalSuccessRate float64
	difficultyCounts := make(map[DifficultyLevel]int)
	hourlySuccess := make(map[int]int)
	hourlyTotal := make(map[int]int)
	
	for _, profile := range pe.userProfiles {
		totalSuccessRate += profile.AvgSuccessRate
		
		if profile.PreferredDifficulty != "" {
			difficultyCounts[profile.PreferredDifficulty]++
		}
		
		for _, record := range profile.SuccessHistory {
			hour := record.Timestamp.Hour()
			hourlyTotal[hour]++
			if record.Difficulty != "" {
				hourlySuccess[hour]++
			}
		}
	}
	
	pe.globalStats.TotalUsers = totalUsers
	pe.globalStats.AvgSuccessRate = totalSuccessRate / float64(totalUsers)
	
	for difficulty, count := range difficultyCounts {
		pe.globalStats.DifficultyDistribution[difficulty] = float64(count) / float64(totalUsers)
	}
	
	for hour := 0; hour < 24; hour++ {
		if hourlyTotal[hour] > 0 {
			pe.globalStats.TimePatterns[hour] = float64(hourlySuccess[hour]) / float64(hourlyTotal[hour])
		}
	}
}

// GetUserSegment 获取用户细分
func (use *UserSegmentationEngine) GetUserSegment(userID string, profile *EnhancedPersonalizedProfile) *UserSegment {
	use.mu.RLock()
	defer use.mu.RUnlock()
	
	if segment, exists := use.segments[profile.UserSegment]; exists {
		return segment
	}
	
	return &UserSegment{
		SegmentID:       profile.UserSegment,
		SegmentName:     profile.UserSegment,
		UserCount:       1,
		AvgSuccessRate: profile.AvgSuccessRate,
		AvgDifficulty:  float64(use.difficultyToScore(profile.PreferredDifficulty)),
		Characteristics: make(map[string]interface{}),
	}
}

// GetSegmentDistribution 获取细分分布
func (use *UserSegmentationEngine) GetSegmentDistribution() map[string]float64 {
	use.mu.RLock()
	defer use.mu.RUnlock()
	
	distribution := make(map[string]float64)
	total := 0
	
	for _, segment := range use.segments {
		total += segment.UserCount
		distribution[segment.SegmentName] = float64(segment.UserCount)
	}
	
	if total > 0 {
		for name := range distribution {
			distribution[name] = distribution[name] / float64(total)
		}
	}
	
	return distribution
}

// difficultyToScore 难度转分数
func (use *UserSegmentationEngine) difficultyToScore(d DifficultyLevel) float64 {
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

// GetRecommendedDifficulty 获取推荐难度
func (pe *PersonalizationEngine) GetRecommendedDifficulty(
	userID string,
	securityLevel float64,
	experienceLevel float64,
) DifficultyLevel {
	
	profile := pe.getEnhancedProfile(userID)
	
	baseScore := pe.difficultyToScore(profile.PreferredDifficulty)
	
	securityWeight := 0.4
	experienceWeight := 0.3
	personalizationWeight := 0.3
	
	recommendedScore := baseScore*personalizationWeight + 
					   securityLevel*securityWeight + 
					   experienceLevel*experienceWeight
	
	return pe.scoreToDifficulty(recommendedScore)
}

type VerificationAttempt struct {
	Timestamp        time.Time
	Difficulty       DifficultyLevel
	Success          bool
	ResponseTime     float64
	MethodUsed       string
	RiskScore        float64
}

type SuccessPattern struct {
	Difficulty      DifficultyLevel
	AvgTime         float64
	Count           int
	PreferredMethod string
	TimeOfDay       int
}

type FailurePattern struct {
	Difficulty      DifficultyLevel
	AvgTimeBeforeFail float64
	Count           int
	CommonReasons    []string
}

// NewEnhancedAdaptiveDifficultyService 创建增强版自适应难度服务
func NewEnhancedAdaptiveDifficultyService() *EnhancedAdaptiveDifficultyService {
	service := &EnhancedAdaptiveDifficultyService{
		AdaptiveDifficultyService: NewAdaptiveDifficultyService(),
		historicalAnalyzer: &EnhancedHistoricalDataAnalyzer{
			userHistory:        make(map[string]*EnhancedUserHistoricalData),
			analyticsDB:        make(map[string]*EnhancedDifficultyAnalytics),
			patternRecognizer:  NewPatternRecognizer(),
			timeSeriesAnalyzer: NewTimeSeriesAnalyzer(),
			cohortAnalyzer:     NewCohortAnalyzer(),
		},
		realtimeAdjuster: &RealtimeDifficultyAdjuster{
			currentAdjustments: make(map[string]*Adjustment),
			config: &RealtimeConfig{
				AdjustmentWindow:      10 * time.Minute,
				MaxAdjustment:        2.0,
				ConsecutiveOKThreshold: 3,
				ConsecutiveFailThreshold: 2,
				CooldownPeriod:       5 * time.Minute,
				SmoothingWindow:      5,
				SmoothingFactor:      0.3,
				AnomalyThreshold:     2.0,
				TransitionSteps:     3,
				TransitionDelay:      30 * time.Second,
			},
			smoothingEngine: NewSmoothingEngine(),
			anomalyDetector: NewAnomalyDetector(),
		},
		personalizationEngine: &PersonalizationEngine{
			userProfiles:     make(map[string]*EnhancedPersonalizedProfile),
			globalStats: &GlobalStats{
				DifficultyDistribution: make(map[DifficultyLevel]float64),
				TimePatterns: make(map[int]float64),
				SegmentDistribution: make(map[string]float64),
			},
			userSegmentation: NewUserSegmentationEngine(),
			experienceManager: NewExperienceBalanceManager(),
		},
	}
	
	service.personalizationEngine.globalStats.DifficultyDistribution[DifficultyEasy] = 0.2
	service.personalizationEngine.globalStats.DifficultyDistribution[DifficultyMedium] = 0.4
	service.personalizationEngine.globalStats.DifficultyDistribution[DifficultyHard] = 0.3
	service.personalizationEngine.globalStats.DifficultyDistribution[DifficultyExpert] = 0.1
	
	return service
}

// NewPatternRecognizer 创建模式识别器
func NewPatternRecognizer() *PatternRecognizer {
	return &PatternRecognizer{
		patterns:     make(map[string]*RecognizedPattern),
		learningRate: 0.1,
	}
}

// NewTimeSeriesAnalyzer 创建时间序列分析器
func NewTimeSeriesAnalyzer() *TimeSeriesAnalyzer {
	return &TimeSeriesAnalyzer{
		seriesData:      make(map[string][]TimeSeriesPoint),
		windowSize:      7,
		smoothingFactor: 0.3,
	}
}

// NewCohortAnalyzer 创建队列分析器
func NewCohortAnalyzer() *CohortAnalyzer {
	return &CohortAnalyzer{
		cohorts:      make(map[string]*CohortData),
		cohortWindow: 7 * 24 * time.Hour,
	}
}

// NewSmoothingEngine 创建平滑引擎
func NewSmoothingEngine() *SmoothingEngine {
	return &SmoothingEngine{
		smoothingWindow: 5,
		smoothingFactor: 0.3,
		historyBuffer:   make(map[string][]float64),
	}
}

// NewAnomalyDetector 创建异常检测器
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		baselineMetrics: make(map[string]*BaselineMetrics),
		sensitivity:     2.0,
	}
}

// NewUserSegmentationEngine 创建用户细分引擎
func NewUserSegmentationEngine() *UserSegmentationEngine {
	return &UserSegmentationEngine{
		segments: make(map[string]*UserSegment),
	}
}

// NewExperienceBalanceManager 创建体验平衡管理器
func NewExperienceBalanceManager() *ExperienceBalanceManager {
	return &ExperienceBalanceManager{
		experienceMetrics: make(map[string]*ExperienceMetrics),
		balanceThresholds: &BalanceThresholds{
			MinSuccessRate:       0.7,
			MaxFrustrationLevel: 0.6,
			MinSatisfactionScore: 3.0,
			MaxTimeAllowed:      30.0,
			MinCompletionRate:    0.8,
		},
	}
}

func (s *EnhancedAdaptiveDifficultyService) GetEnhancedDifficulty(
	userID string,
	context *DifficultyContext,
) (DifficultyLevel, *DifficultyRecommendation) {
	
	baseDifficulty := s.AdaptiveDifficultyService.GetDifficulty(userID)
	
	historicalData := s.historicalAnalyzer.getUserHistoryEnhanced(userID)
	
	personalizedDifficulty := s.personalizationEngine.getEnhancedPersonalizedDifficulty(
		userID,
		baseDifficulty,
		historicalData,
	)
	
	realtimeAdjustment := s.realtimeAdjuster.getAdjustment(userID)
	
	smoothedAdjustment := s.realtimeAdjuster.smoothingEngine.applySmoothing(
		userID,
		realtimeAdjustment,
	)
	
	finalDifficulty := s.calculateFinalDifficultyWithTransition(
		personalizedDifficulty,
		smoothedAdjustment,
		context,
		realtimeAdjustment,
	)
	
	recommendation := &DifficultyRecommendation{
		RecommendedDifficulty: finalDifficulty,
		BaseDifficulty:        baseDifficulty,
		PersonalizationBonus:  s.calculatePersonalizationBonus(userID, historicalData),
		HistoricalAdjustment:  s.calculateHistoricalAdjustment(historicalData),
		RealtimeAdjustment:   realtimeAdjustment,
		Confidence:           s.calculateRecommendationConfidence(userID),
		AlternativeMethods:   s.suggestAlternativeMethods(userID, finalDifficulty),
		Reasoning:            s.generateReasoning(finalDifficulty, baseDifficulty, historicalData),
	}
	
	return finalDifficulty, recommendation
}

// GetEnhancedUserAnalytics 获取增强版用户分析报告
func (s *EnhancedAdaptiveDifficultyService) GetEnhancedUserAnalytics(userID string) *EnhancedUserAnalyticsReport {
	history := s.historicalAnalyzer.getUserHistoryEnhanced(userID)
	analytics := s.historicalAnalyzer.calculateEnhancedAnalytics(userID)
	profile := s.personalizationEngine.getEnhancedProfile(userID)
	experienceMetrics := s.personalizationEngine.experienceManager.getMetrics(userID)
	
	report := &EnhancedUserAnalyticsReport{
		UserID:          userID,
		TotalAttempts:   history.TotalAttempts,
		SuccessCount:    history.SuccessCount,
		SuccessRate:     analytics.SuccessRate,
		AvgTime:         analytics.AvgTime,
		MedianTime:      analytics.MedianTime,
		Trend:           analytics.Trend,
		PreferredMethod: profile.PreferredMethod,
		OptimalTimes:    profile.OptimalHours,
		AvgDifficulty:   history.AvgDifficulty,
		// 增强字段
		UserSegment:          analytics.UserSegment,
		EngagementLevel:      analytics.EngagementLevel,
		PredictedChurnRisk:   analytics.PredictedChurnRisk,
		SessionEngagement:     analytics.SessionEngagement,
		ConsistencyScore:     analytics.ConsistencyScore,
		ProgressVelocity:     analytics.ProgressVelocity,
		SkillScore:           profile.SkillScore,
		ExperienceLevel:      profile.ExperienceLevel,
		SatisfactionScore:    experienceMetrics.SatisfactionScore,
		FrustrationLevel:     experienceMetrics.FrustrationLevel,
		DevicePreference:     profile.PreferredDevice,
		NetworkPreference:    profile.PreferredNetwork,
	}
	
	return report
}

// calculateFinalDifficultyWithTransition 计算带平滑过渡的最终难度
func (s *EnhancedAdaptiveDifficultyService) calculateFinalDifficultyWithTransition(
	personalized DifficultyLevel,
	smoothedAdjustment float64,
	context *DifficultyContext,
	adjustment *Adjustment,
) DifficultyLevel {
	
	difficultyScore := s.difficultyToScore(personalized)
	
	if adjustment != nil && time.Now().Before(adjustment.ValidUntil) {
		if adjustment.TransitionState.IsTransitioning {
			currentScore := s.difficultyToScore(adjustment.TransitionState.FromDifficulty)
			targetScore := s.difficultyToScore(adjustment.TransitionState.ToDifficulty)
			transitionProgress := 1.0 - float64(adjustment.TransitionState.StepsRemaining)/float64(s.realtimeAdjuster.config.TransitionSteps)
			difficultyScore = currentScore + (targetScore-currentScore)*transitionProgress
		} else {
			difficultyScore += smoothedAdjustment
		}
	}
	
	if context != nil {
		if context.HighRiskContext {
			difficultyScore += 1.0
		}
		if context.TimeSensitive {
			difficultyScore -= 0.5
		}
		if context.UserRequestedDifficulty != "" {
			requestedDifficulty := DifficultyLevel(context.UserRequestedDifficulty)
			requestedScore := s.difficultyToScore(requestedDifficulty)
			difficultyScore = difficultyScore*0.7 + requestedScore*0.3
		}
	}
	
	difficultyScore = math.Max(0, math.Min(4, difficultyScore))
	
	return s.scoreToDifficulty(difficultyScore)
}

// InitiateSmoothTransition 初始化平滑过渡
func (rda *RealtimeDifficultyAdjuster) InitiateSmoothTransition(
	userID string,
	from, to DifficultyLevel,
) {
	rda.mu.Lock()
	defer rda.mu.Unlock()
	
	adj, exists := rda.currentAdjustments[userID]
	if !exists {
		return
	}
	
	adj.TransitionState = TransitionState{
		FromDifficulty:  from,
		ToDifficulty:    to,
		TransitionRate:  1.0 / float64(rda.config.TransitionSteps),
		StepsRemaining:  rda.config.TransitionSteps,
		IsTransitioning: true,
		StartTime:       time.Now(),
	}
}

// ProcessTransitionStep 处理过渡步骤
func (rda *RealtimeDifficultyAdjuster) ProcessTransitionStep(userID string) {
	rda.mu.Lock()
	defer rda.mu.Unlock()
	
	adj, exists := rda.currentAdjustments[userID]
	if !exists || !adj.TransitionState.IsTransitioning {
		return
	}
	
	if time.Since(adj.TransitionState.StartTime) >= rda.config.TransitionDelay {
		adj.TransitionState.StepsRemaining--
		adj.TransitionState.StartTime = time.Now()
		
		if adj.TransitionState.StepsRemaining <= 0 {
			adj.TransitionState.IsTransitioning = false
			adj.BaseDifficulty = adj.TransitionState.ToDifficulty
		}
	}
}

// applySmoothing 应用平滑算法
func (se *SmoothingEngine) applySmoothing(userID string, adjustment *Adjustment) float64 {
	if adjustment == nil {
		return 0
	}
	
	se.mu.Lock()
	defer se.mu.Unlock()
	
	history := se.historyBuffer[userID]
	history = append(history, adjustment.Adjustment)
	
	if len(history) > se.smoothingWindow {
		history = history[len(history)-se.smoothingWindow:]
	}
	
	se.historyBuffer[userID] = history
	
	if len(history) == 0 {
		return 0
	}
	
	var smoothedValue float64
	for i, val := range history {
		weight := math.Pow(se.smoothingFactor, float64(len(history)-i-1))
		smoothedValue += val * weight
	}
	
	totalWeight := 0.0
	for i := 0; i < len(history); i++ {
		totalWeight += math.Pow(se.smoothingFactor, float64(len(history)-i-1))
	}
	
	return smoothedValue / totalWeight
}

// DetectAnomaly 检测异常
func (ad *AnomalyDetector) DetectAnomaly(userID string, metrics *BaselineMetrics) bool {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	
	baseline, exists := ad.baselineMetrics[userID]
	if !exists {
		ad.baselineMetrics[userID] = metrics
		return false
	}
	
	zScore := math.Abs(metrics.AvgTime - baseline.AvgTime) / baseline.StdDeviation
	
	if zScore > ad.sensitivity {
		return true
	}
	
	return false
}

// UpdateBaseline 更新基线
func (ad *AnomalyDetector) UpdateBaseline(userID string, newMetrics *BaselineMetrics) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	
	baseline, exists := ad.baselineMetrics[userID]
	if !exists {
		ad.baselineMetrics[userID] = newMetrics
		return
	}
	
	baseline.AvgSuccessRate = baseline.AvgSuccessRate*0.9 + newMetrics.AvgSuccessRate*0.1
	baseline.AvgTime = baseline.AvgTime*0.9 + newMetrics.AvgTime*0.1
	baseline.AvgDifficulty = baseline.AvgDifficulty*0.9 + newMetrics.AvgDifficulty*0.1
	baseline.StdDeviation = baseline.StdDeviation*0.9 + newMetrics.StdDeviation*0.1
	baseline.LastUpdated = time.Now()
}

func (s *EnhancedAdaptiveDifficultyService) UpdateDifficultyWithResult(
	userID string,
	difficulty DifficultyLevel,
	success bool,
	responseTime time.Duration,
	method string,
) {
	s.AdaptiveDifficultyService.UpdateProfile(userID, success, responseTime)
	
	attempt := VerificationAttempt{
		Timestamp:    time.Now(),
		Difficulty:   difficulty,
		Success:      success,
		ResponseTime: responseTime.Seconds(),
		MethodUsed:   method,
	}
	
	s.historicalAnalyzer.recordEnhancedAttempt(userID, attempt)
	
	s.realtimeAdjuster.updateAdjustment(userID, difficulty, success, responseTime)
	
	s.personalizationEngine.updateEnhancedProfile(userID, difficulty, method, responseTime)
	
	s.analyzeAndAdapt(userID)
	
	s.historicalAnalyzer.timeSeriesAnalyzer.addDataPoint(userID, responseTime.Seconds())
	
	s.historicalAnalyzer.patternRecognizer.recognizePattern(userID, difficulty, success, responseTime)
}

// UpdateEnhancedDifficultyWithContext 使用上下文更新难度
func (s *EnhancedAdaptiveDifficultyService) UpdateEnhancedDifficultyWithContext(
	userID string,
	difficulty DifficultyLevel,
	success bool,
	responseTime time.Duration,
	method string,
	context *EnhancedVerificationContext,
) {
	s.UpdateDifficultyWithResult(userID, difficulty, success, responseTime, method)
	
	if context != nil {
		s.historicalAnalyzer.updateContextData(userID, context)
	}
	
	s.personalizationEngine.experienceManager.updateExperienceMetrics(
		userID,
		success,
		responseTime,
	)
	
	s.realtimeAdjuster.processTransitionStepIfNeeded(userID)
}

// getUserHistoryEnhanced 获取增强版用户历史
func (hda *EnhancedHistoricalDataAnalyzer) getUserHistoryEnhanced(userID string) *EnhancedUserHistoricalData {
	hda.mu.RLock()
	defer hda.mu.RUnlock()
	return hda.userHistory[userID]
}

// recordEnhancedAttempt 记录增强版验证尝试
func (hda *EnhancedHistoricalDataAnalyzer) recordEnhancedAttempt(userID string, attempt VerificationAttempt) {
	hda.mu.Lock()
	defer hda.mu.Unlock()
	
	history, exists := hda.userHistory[userID]
	if !exists {
		history = &EnhancedUserHistoricalData{
			UserID:               userID,
			VerificationHistory:  make([][]VerificationAttempt, 0),
			PreferredTimes:        make(map[int][]time.Time),
			AvgTimeByDifficulty:  make(map[DifficultyLevel]float64),
			MethodPreference:     make(map[string]int),
			DifficultyProgress:   make([]DifficultyProgress, 0),
			ErrorDistribution:    make(map[string]int),
			TimeZonePattern:      make(map[string]int),
			ResponseTimeTrend:   make([]float64, 0),
		}
		hda.userHistory[userID] = history
	}
	
	history.VerificationHistory = append(history.VerificationHistory, []VerificationAttempt{attempt})
	history.TotalAttempts++
	if attempt.Success {
		history.SuccessCount++
	}
	history.LastUpdated = time.Now()
	
	history.MethodPreference[attempt.MethodUsed]++
	
	hour := attempt.Timestamp.Hour()
	history.PreferredTimes[hour] = append(history.PreferredTimes[hour], attempt.Timestamp)
	
	if len(history.ResponseTimeTrend) < 100 {
		history.ResponseTimeTrend = append(history.ResponseTimeTrend, attempt.ResponseTime)
	} else {
		history.ResponseTimeTrend = append(history.ResponseTimeTrend[1:], attempt.ResponseTime)
	}
	
	pattern := SuccessPattern{
		Difficulty: attempt.Difficulty,
		AvgTime:    attempt.ResponseTime,
		Count:      1,
		TimeOfDay:  hour,
	}
	history.SuccessPatterns = append(history.SuccessPatterns, pattern)
	
	hda.limitEnhancedHistory(history)
	
	if len(history.SuccessPatterns) > 50 {
		history.SuccessPatterns = history.SuccessPatterns[len(history.SuccessPatterns)-50:]
	}
	if len(history.FailurePatterns) > 50 {
		history.FailurePatterns = history.FailurePatterns[len(history.FailurePatterns)-50:]
	}
}

// updateContextData 更新上下文数据
func (hda *EnhancedHistoricalDataAnalyzer) updateContextData(userID string, context *EnhancedVerificationContext) {
	hda.mu.Lock()
	defer hda.mu.Unlock()
	
	history, exists := hda.userHistory[userID]
	if !exists {
		return
	}
	
	if context.DeviceFingerprint != "" {
		if history.DeviceFingerprint != "" && history.DeviceFingerprint != context.DeviceFingerprint {
			history.DeviceSwitchCount++
		}
		history.DeviceFingerprint = context.DeviceFingerprint
		history.LastDeviceID = context.DeviceID
	}
	
	if context.IPAddress != "" {
		history.IPAddress = context.IPAddress
	}
	
	if context.GeographicLocation != "" {
		history.GeographicLocation = context.GeographicLocation
	}
	
	if context.NetworkQuality > 0 {
		history.NetworkQuality = history.NetworkQuality*0.9 + context.NetworkQuality*0.1
	}
	
	if context.SessionID != "" {
		history.SessionCount++
	}
	
	if context.TimeZone != "" {
		history.TimeZonePattern[context.TimeZone]++
	}
}

// calculateEnhancedAnalytics 计算增强分析
func (hda *EnhancedHistoricalDataAnalyzer) calculateEnhancedAnalytics(userID string) *EnhancedDifficultyAnalytics {
	hda.mu.RLock()
	defer hda.mu.RUnlock()
	
	history := hda.userHistory[userID]
	if history == nil {
		return &EnhancedDifficultyAnalytics{}
	}
	
	analytics := &EnhancedDifficultyAnalytics{
		DifficultyDistribution: make(map[DifficultyLevel]int),
		SuccessByHour:        make(map[int]int),
		TimeOfDayPattern:     make(map[int]float64),
		WeekdayPattern:       make(map[int]float64),
		DevicePattern:        make(map[string]float64),
		NetworkPattern:       make(map[string]float64),
		MethodEffectiveness:  make(map[string]float64),
	}
	
	if len(history.VerificationHistory) == 0 {
		return analytics
	}
	
	analytics.TotalAttempts = history.TotalAttempts
	analytics.SuccessRate = float64(history.SuccessCount) / float64(history.TotalAttempts) * 100
	
	allTimes := make([]float64, 0)
	successTimes := make([]float64, 0)
	hourCounts := make(map[int]int)
	weekdayCounts := make(map[int]int)
	
	for _, dayHistory := range history.VerificationHistory {
		for _, attempt := range dayHistory {
			allTimes = append(allTimes, attempt.ResponseTime)
			analytics.DifficultyDistribution[attempt.Difficulty]++
			
			hour := attempt.Timestamp.Hour()
			hourCounts[hour]++
			
			weekday := int(attempt.Timestamp.Weekday())
			weekdayCounts[weekday]++
			
			if attempt.Success {
				analytics.SuccessByHour[hour]++
				successTimes = append(successTimes, attempt.ResponseTime)
			}
			
			if attempt.MethodUsed != "" {
				if _, exists := analytics.MethodEffectiveness[attempt.MethodUsed]; !exists {
					analytics.MethodEffectiveness[attempt.MethodUsed] = 0
				}
				if attempt.Success {
					analytics.MethodEffectiveness[attempt.MethodUsed]++
				}
			}
		}
	}
	
	for hour, count := range hourCounts {
		analytics.TimeOfDayPattern[hour] = float64(analytics.SuccessByHour[hour]) / float64(count) * 100
	}
	
	for weekday, count := range weekdayCounts {
		analytics.WeekdayPattern[weekday] = float64(count) / float64(analytics.TotalAttempts) * 100
	}
	
	if len(allTimes) > 0 {
		analytics.AvgTime = meanFloat(allTimes)
		sort.Float64s(allTimes)
		analytics.MedianTime = allTimes[len(allTimes)/2]
		analytics.MinTime = allTimes[0]
		analytics.MaxTime = allTimes[len(allTimes)-1]
		analytics.TimeVariance = varianceFloat(allTimes)
		analytics.SuccessRateVariance = varianceFloat(successTimes)
	}
	
	if len(history.ResponseTimeTrend) >= 7 {
		analytics.ConsistencyScore = 1.0 / (1.0 + varianceFloat(history.ResponseTimeTrend))
		
		recentTimes := history.ResponseTimeTrend[len(history.ResponseTimeTrend)-7:]
		olderTimes := history.ResponseTimeTrend[:len(history.ResponseTimeTrend)-7]
		if len(olderTimes) > 0 {
			recentAvg := meanFloat(recentTimes)
			olderAvg := meanFloat(olderTimes)
			if olderAvg > 0 {
				analytics.ProgressVelocity = (olderAvg - recentAvg) / olderAvg
			}
		}
	}
	
	analytics.Trend = hda.calculateTrend(history.VerificationHistory)
	
	if history.TotalAttempts > 0 {
		analytics.AttemptsPerSession = float64(history.TotalAttempts) / float64(math.Max(1, float64(history.SessionCount)))
	}
	
	if len(history.SuccessPatterns) > 0 {
		analytics.TimeToSuccess = history.SuccessPatterns[len(history.SuccessPatterns)-1].AvgTime
	}
	
	analytics.UserSegment = hda.determineUserSegment(history)
	analytics.EngagementLevel = hda.determineEngagementLevel(history)
	
	if analytics.SuccessRate < 50 && history.TotalAttempts > 10 {
		analytics.PredictedChurnRisk = 0.8
	} else if analytics.SuccessRate < 70 && history.TotalAttempts > 5 {
		analytics.PredictedChurnRisk = 0.5
	} else {
		analytics.PredictedChurnRisk = 0.2
	}
	
	return analytics
}

// limitEnhancedHistory 限制增强历史大小
func (hda *EnhancedHistoricalDataAnalyzer) limitEnhancedHistory(history *EnhancedUserHistoricalData) {
	maxDays := 30
	if len(history.VerificationHistory) > maxDays {
		history.VerificationHistory = history.VerificationHistory[len(history.VerificationHistory)-maxDays:]
	}
}

// determineUserSegment 确定用户细分
func (hda *EnhancedHistoricalDataAnalyzer) determineUserSegment(history *EnhancedUserHistoricalData) string {
	if history.TotalAttempts < 3 {
		return "new"
	}
	
	if history.SuccessRate > 90 && history.TotalAttempts > 20 {
		return "expert"
	} else if history.SuccessRate > 70 {
		return "regular"
	} else if history.SuccessRate > 50 {
		return "struggling"
	} else {
		return "at_risk"
	}
}

// determineEngagementLevel 确定参与度级别
func (hda *EnhancedHistoricalDataAnalyzer) determineEngagementLevel(history *EnhancedUserHistoricalData) string {
	if history.TotalAttempts < 3 {
		return "low"
	}
	
	if history.SessionCount > 10 && history.TotalAttempts > 30 {
		return "high"
	} else if history.SessionCount > 5 && history.TotalAttempts > 15 {
		return "medium"
	} else {
		return "low"
	}
}

// calculateTrend 计算趋势
func (hda *EnhancedHistoricalDataAnalyzer) calculateTrend(history [][]VerificationAttempt) string {
	if len(history) < 7 {
		return "stable"
	}
	
	recentSuccess := 0
	recentTotal := 0
	oldSuccess := 0
	oldTotal := 0
	
	for i, day := range history {
		for _, attempt := range day {
			if i >= len(history)-3 {
				recentTotal++
				if attempt.Success {
					recentSuccess++
				}
			}
			if i < len(history)-7 && i >= len(history)-7 {
				oldTotal++
				if attempt.Success {
					oldSuccess++
				}
			}
		}
	}
	
	if recentTotal == 0 || oldTotal == 0 {
		return "stable"
	}
	
	recentRate := float64(recentSuccess) / float64(recentTotal)
	oldRate := float64(oldSuccess) / float64(oldTotal)
	
	diff := recentRate - oldRate
	if diff > 0.1 {
		return "improving"
	} else if diff < -0.1 {
		return "declining"
	}
	return "stable"
}

// addDataPoint 添加时间序列数据点
func (tsa *TimeSeriesAnalyzer) addDataPoint(userID string, value float64) {
	tsa.mu.Lock()
	defer tsa.mu.Unlock()
	
	point := TimeSeriesPoint{
		Timestamp: time.Now(),
		Value:     value,
	}
	
	series := tsa.seriesData[userID]
	series = append(series, point)
	
	if len(series) > tsa.windowSize*10 {
		series = series[len(series)-tsa.windowSize*10:]
	}
	
	tsa.seriesData[userID] = series
}

// recognizePattern 识别模式
func (pr *PatternRecognizer) recognizePattern(userID string, difficulty DifficultyLevel, success bool, responseTime time.Duration) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	
	patternKey := fmt.Sprintf("%s_%s_%s", userID, difficulty, success)
	
	if pattern, exists := pr.patterns[patternKey]; exists {
		pattern.Frequency++
		pattern.LastSeen = time.Now()
		pattern.Confidence = math.Min(1.0, float64(pattern.Frequency)/10.0)
	} else {
		pr.patterns[patternKey] = &RecognizedPattern{
			PatternType:   patternKey,
			Confidence:    0.1,
			Frequency:     1,
			FirstSeen:     time.Now(),
			LastSeen:      time.Now(),
			AssociatedRisk: 0.5,
		}
	}
	
	if !success && responseTime < 2*time.Second {
		riskPatternKey := fmt.Sprintf("fast_fail_%s", difficulty)
		if pattern, exists := pr.patterns[riskPatternKey]; exists {
			pattern.Frequency++
			pattern.AssociatedRisk = math.Min(1.0, pattern.AssociatedRisk+0.1)
		}
	}
}

// processTransitionStepIfNeeded 处理过渡步骤
func (rda *RealtimeDifficultyAdjuster) processTransitionStepIfNeeded(userID string) {
	rda.mu.Lock()
	defer rda.mu.Unlock()
	
	adj, exists := rda.currentAdjustments[userID]
	if !exists || !adj.TransitionState.IsTransitioning {
		return
	}
	
	if time.Since(adj.TransitionState.StartTime) >= rda.config.TransitionDelay {
		adj.TransitionState.StepsRemaining--
		adj.TransitionState.StartTime = time.Now()
		
		if adj.TransitionState.StepsRemaining <= 0 {
			adj.TransitionState.IsTransitioning = false
			adj.BaseDifficulty = adj.TransitionState.ToDifficulty
		}
	}
}

func (s *EnhancedAdaptiveDifficultyService) analyzeAndAdapt(userID string) {
	history := s.historicalAnalyzer.getUserHistory(userID)
	if history == nil || len(history.VerificationHistory) < 5 {
		return
	}
	
	analytics := s.historicalAnalyzer.calculateAnalytics(userID)
	
	if analytics.SuccessRate < 70 && history.AvgDifficulty < 60 {
		s.AdaptiveDifficultyService.UpdateConfig(&DifficultyConfig{
			EasyThreshold:   15.0,
			MediumThreshold: 35.0,
			HardThreshold:   55.0,
			ExpertThreshold: 75.0,
			FailureWeight:   10.0,
			SuccessWeight:   -3.0,
			TimePenalty:     1.5,
		})
	} else if analytics.SuccessRate > 95 && history.AvgDifficulty > 60 {
		s.AdaptiveDifficultyService.UpdateConfig(&DifficultyConfig{
			EasyThreshold:   25.0,
			MediumThreshold: 45.0,
			HardThreshold:   65.0,
			ExpertThreshold: 85.0,
			FailureWeight:   20.0,
			SuccessWeight:   -8.0,
			TimePenalty:     3.0,
		})
	}
}

func (s *EnhancedAdaptiveDifficultyService) GetUserAnalytics(userID string) *UserAnalyticsReport {
	history := s.historicalAnalyzer.getUserHistory(userID)
	analytics := s.historicalAnalyzer.calculateAnalytics(userID)
	profile := s.personalizationEngine.getProfile(userID)
	
	report := &UserAnalyticsReport{
		UserID:          userID,
		TotalAttempts:   history.TotalAttempts,
		SuccessCount:    history.SuccessCount,
		SuccessRate:     analytics.SuccessRate,
		AvgTime:         analytics.AvgTime,
		MedianTime:      analytics.MedianTime,
		Trend:           analytics.Trend,
		PreferredMethod: profile.PreferredMethod,
		OptimalTimes:    profile.OptimalHours,
		AvgDifficulty:   history.AvgDifficulty,
	}
	
	return report
}

// GetRecommendedDifficulty 获取推荐难度
func (s *EnhancedAdaptiveDifficultyService) GetRecommendedDifficulty(
	userID string,
	securityLevel float64,
	experienceLevel float64,
) DifficultyLevel {
	return s.personalizationEngine.GetRecommendedDifficulty(userID, securityLevel, experienceLevel)
}

// GetPersonalizationGlobalStats 获取个性化全局统计
func (s *EnhancedAdaptiveDifficultyService) GetPersonalizationGlobalStats() *GlobalStats {
	return s.personalizationEngine.GetGlobalStats()
}

// InitiateTransition 发起难度转换
func (s *EnhancedAdaptiveDifficultyService) InitiateTransition(userID string, from, to DifficultyLevel) {
	s.realtimeAdjuster.InitiateSmoothTransition(userID, from, to)
}

// GetTransitionSteps 获取转换步骤数
func (s *EnhancedAdaptiveDifficultyService) GetTransitionSteps() int {
	return s.realtimeAdjuster.config.TransitionSteps
}

func (s *EnhancedAdaptiveDifficultyService) calculateFinalDifficulty(
	personalized DifficultyLevel,
	adjustment *Adjustment,
	context *DifficultyContext,
) DifficultyLevel {
	
	difficultyScore := s.difficultyToScore(personalized)
	
	if adjustment != nil && time.Now().Before(adjustment.ValidUntil) {
		difficultyScore += adjustment.Adjustment
	}
	
	if context != nil {
		if context.HighRiskContext {
			difficultyScore += 1.0
		}
		if context.TimeSensitive {
			difficultyScore -= 0.5
		}
		if context.UserRequestedDifficulty != "" {
			requestedDifficulty := DifficultyLevel(context.UserRequestedDifficulty)
			requestedScore := s.difficultyToScore(requestedDifficulty)
			difficultyScore = difficultyScore*0.7 + requestedScore*0.3
		}
	}
	
	difficultyScore = math.Max(0, math.Min(4, difficultyScore))
	
	return s.scoreToDifficulty(difficultyScore)
}

func (s *EnhancedAdaptiveDifficultyService) calculatePersonalizationBonus(userID string, history *UserHistoricalData) float64 {
	if history == nil || len(history.SuccessPatterns) == 0 {
		return 0
	}
	
	bonus := 0.0
	if len(history.SuccessPatterns) >= 3 {
		avgSuccessTime := 0.0
		for _, sp := range history.SuccessPatterns {
			avgSuccessTime += sp.AvgTime
		}
		avgSuccessTime /= float64(len(history.SuccessPatterns))
		
		if avgSuccessTime < 3.0 {
			bonus += 0.5
		} else if avgSuccessTime < 5.0 {
			bonus += 0.3
		}
	}
	
	if history.SuccessCount > history.TotalAttempts*90/100 {
		bonus += 0.3
	}
	
	return bonus
}

func (s *EnhancedAdaptiveDifficultyService) calculateHistoricalAdjustment(history *UserHistoricalData) float64 {
	if history == nil {
		return 0
	}
	
	if len(history.FailurePatterns) == 0 {
		return -0.5
	}
	
	maxFailureDifficulty := float64(0)
	for _, fp := range history.FailurePatterns {
		difficultyScore := s.difficultyToScore(fp.Difficulty)
		if difficultyScore > maxFailureDifficulty {
			maxFailureDifficulty = difficultyScore
		}
	}
	
	return -maxFailureDifficulty * 0.1
}

func (s *EnhancedAdaptiveDifficultyService) calculateRecommendationConfidence(userID string) float64 {
	history := s.historicalAnalyzer.getUserHistory(userID)
	if history == nil {
		return 0.3
	}
	
	confidence := math.Min(1.0, float64(history.TotalAttempts)/20.0)
	confidence *= history.SuccessRate / 100.0
	
	return confidence
}

func (s *EnhancedAdaptiveDifficultyService) suggestAlternativeMethods(userID string, difficulty DifficultyLevel) []string {
	methods := []string{}
	
	switch difficulty {
	case DifficultyEasy:
		methods = append(methods, "slider_simple", "click_simple", "seamless")
	case DifficultyMedium:
		methods = append(methods, "slider", "click", "3d_rotate")
	case DifficultyHard:
		methods = append(methods, "3d_click", "lianliankan", "voice")
	case DifficultyExpert:
		methods = append(methods, "3d_complete", "multi_step", "biometrics")
	}
	
	profile := s.personalizationEngine.getProfile(userID)
	if profile != nil && profile.PreferredMethod != "" {
		for i, m := range methods {
			if m == profile.PreferredMethod {
				methods = append([]string{m}, append(methods[:i], methods[i+1:]...)...)
				break
			}
		}
	}
	
	return methods
}

func (s *EnhancedAdaptiveDifficultyService) generateReasoning(
	final, base DifficultyLevel,
	history *UserHistoricalData,
) string {
	
	reasoning := fmt.Sprintf("基于用户历史表现，推荐难度从 %s 调整为 %s。", base, final)
	
	if history != nil && history.TotalAttempts > 0 {
		reasoning += fmt.Sprintf(" 总验证次数: %d, 成功率: %.1f%%", 
			history.TotalAttempts, history.SuccessRate)
	}
	
	if final == DifficultyEasy {
		reasoning += " 考虑到用户表现良好，降低难度以提升用户体验。"
	} else if final == DifficultyHard || final == DifficultyExpert {
		reasoning += " 检测到异常行为，提升难度以增强安全性。"
	}
	
	return reasoning
}

func (s *EnhancedAdaptiveDifficultyService) difficultyToScore(d DifficultyLevel) float64 {
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

func (s *EnhancedAdaptiveDifficultyService) scoreToDifficulty(score float64) DifficultyLevel {
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

func (hda *HistoricalDataAnalyzer) getUserHistory(userID string) *UserHistoricalData {
	hda.mu.RLock()
	defer hda.mu.RUnlock()
	return hda.userHistory[userID]
}

func (hda *HistoricalDataAnalyzer) recordAttempt(userID string, attempt VerificationAttempt) {
	hda.mu.Lock()
	defer hda.mu.Unlock()
	
	history, exists := hda.userHistory[userID]
	if !exists {
		history = &UserHistoricalData{
			UserID:               userID,
			VerificationHistory:  make([][]VerificationAttempt, 0),
			PreferredTimes:        make(map[int][]time.Time),
			AvgTimeByDifficulty:  make(map[DifficultyLevel]float64),
		}
		hda.userHistory[userID] = history
	}
	
	history.VerificationHistory = append(history.VerificationHistory, []VerificationAttempt{attempt})
	history.TotalAttempts++
	if attempt.Success {
		history.SuccessCount++
	}
	history.LastUpdated = time.Now()
	
	hour := attempt.Timestamp.Hour()
	history.PreferredTimes[hour] = append(history.PreferredTimes[hour], attempt.Timestamp)
	
	pattern := SuccessPattern{
		Difficulty: attempt.Difficulty,
		AvgTime:    attempt.ResponseTime,
		Count:      1,
		TimeOfDay:  hour,
	}
	history.SuccessPatterns = append(history.SuccessPatterns, pattern)
	
	limitHistory(history)
}

func (hda *HistoricalDataAnalyzer) calculateAnalytics(userID string) *DifficultyAnalytics {
	hda.mu.RLock()
	defer hda.mu.RUnlock()
	
	history := hda.userHistory[userID]
	if history == nil {
		return &DifficultyAnalytics{}
	}
	
	analytics := &DifficultyAnalytics{
		DifficultyDistribution: make(map[DifficultyLevel]int),
		SuccessByHour:        make(map[int]int),
	}
	
	if len(history.VerificationHistory) == 0 {
		return analytics
	}
	
	analytics.TotalAttempts = history.TotalAttempts
	analytics.SuccessRate = float64(history.SuccessCount) / float64(history.TotalAttempts) * 100
	
	allTimes := make([]float64, 0)
	for _, dayHistory := range history.VerificationHistory {
		for _, attempt := range dayHistory {
			allTimes = append(allTimes, attempt.ResponseTime)
			analytics.DifficultyDistribution[attempt.Difficulty]++
			
			if attempt.Success {
				analytics.SuccessByHour[attempt.Timestamp.Hour()]++
			}
		}
	}
	
	if len(allTimes) > 0 {
		analytics.AvgTime = meanFloat(allTimes)
		sort.Float64s(allTimes)
		analytics.MedianTime = allTimes[len(allTimes)/2]
		analytics.MinTime = allTimes[0]
		analytics.MaxTime = allTimes[len(allTimes)-1]
		analytics.TimeVariance = varianceFloat(allTimes)
	}
	
	analytics.Trend = calculateTrend(history.VerificationHistory)
	
	return analytics
}

func (rda *RealtimeDifficultyAdjuster) getAdjustment(userID string) *Adjustment {
	rda.mu.RLock()
	defer rda.mu.RUnlock()
	return rda.currentAdjustments[userID]
}

func (rda *RealtimeDifficultyAdjuster) updateAdjustment(
	userID string,
	difficulty DifficultyLevel,
	success bool,
	responseTime time.Duration,
) {
	rda.mu.Lock()
	defer rda.mu.Unlock()
	
	adj, exists := rda.currentAdjustments[userID]
	if !exists {
		adj = &Adjustment{
			UserID:         userID,
			BaseDifficulty: difficulty,
			Adjustment:    0,
			ValidUntil:    time.Now().Add(rda.config.AdjustmentWindow),
		}
		rda.currentAdjustments[userID] = adj
	}
	
	if success {
		adj.ConsecutiveOK++
		adj.ConsecutiveFail = 0
		
		if adj.ConsecutiveOK >= rda.config.ConsecutiveOKThreshold {
			adj.Adjustment = math.Max(-rda.config.MaxAdjustment, adj.Adjustment-0.5)
			adj.Reason = "连续成功，降低难度"
			adj.ValidUntil = time.Now().Add(rda.config.AdjustmentWindow)
		}
	} else {
		adj.ConsecutiveFail++
		adj.ConsecutiveOK = 0
		
		if adj.ConsecutiveFail >= rda.config.ConsecutiveFailThreshold {
			adj.Adjustment = math.Min(rda.config.MaxAdjustment, adj.Adjustment+1.0)
			adj.Reason = "连续失败，提升难度"
			adj.ValidUntil = time.Now().Add(rda.config.AdjustmentWindow)
		}
	}
	
	if responseTime.Seconds() > 15 && !success {
		adj.Adjustment = math.Max(-rda.config.MaxAdjustment, adj.Adjustment-0.3)
		adj.Reason = "超时未完成，降低难度"
	}
}

func (pe *PersonalizationEngine) getPersonalizedDifficulty(
	userID string,
	base DifficultyLevel,
	history *UserHistoricalData,
) DifficultyLevel {
	
	profile := pe.getProfile(userID)
	if profile != nil && profile.ComfortZone != "" {
		comfortScore := pe.difficultyToScore(profile.ComfortZone)
		baseScore := pe.difficultyToScore(base)
		
		combinedScore := baseScore*0.6 + comfortScore*0.4
		return pe.scoreToDifficulty(combinedScore)
	}
	
	return base
}

// getEnhancedPersonalizedDifficulty 获取增强版个性化难度
func (pe *PersonalizationEngine) getEnhancedPersonalizedDifficulty(
	userID string,
	base DifficultyLevel,
	history *EnhancedUserHistoricalData,
) DifficultyLevel {
	
	profile := pe.getEnhancedProfile(userID)
	if profile == nil {
		return base
	}
	
	baseScore := pe.difficultyToScore(base)
	comfortScore := pe.difficultyToScore(profile.ComfortZone)
	challengeScore := pe.difficultyToScore(profile.ChallengeLevel)
	
	experienceWeight := math.Min(1.0, float64(profile.ExperienceLevel)/10.0)
	engagementFactor := profile.EngagementScore / 100.0
	
	combinedScore := baseScore*(1-experienceWeight*0.3) + 
					comfortScore*(experienceWeight*0.3) + 
					challengeScore*(engagementFactor*0.2)
	
	return pe.scoreToDifficulty(combinedScore)
}

// getEnhancedProfile 获取增强版档案
func (pe *PersonalizationEngine) getEnhancedProfile(userID string) *EnhancedPersonalizedProfile {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	
	if profile, exists := pe.userProfiles[userID]; exists {
		return profile
	}
	
	return &EnhancedPersonalizedProfile{
		UserID:             userID,
		AvgSuccessRate:      80.0,
		AvgTime:           5.0,
		ExperienceLevel:   1,
		EngagementScore:   50.0,
		SkillScore:        0.5,
		ChurnRisk:         0.3,
		LearningProgress:  0.0,
		StressLevel:       0.0,
		SafetyBalance:     0.5,
		ExperienceBalance: 0.5,
	}
}

// updateEnhancedProfile 更新增强版档案
func (pe *PersonalizationEngine) updateEnhancedProfile(
	userID string,
	difficulty DifficultyLevel,
	method string,
	responseTime time.Duration,
) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	profile, exists := pe.userProfiles[userID]
	if !exists {
		profile = &EnhancedPersonalizedProfile{
			UserID:              userID,
			AvgSuccessRate:      80.0,
			AvgTime:            5.0,
			SuccessHistory:      make([]SuccessRecord, 0),
			ExperienceLevel:    1,
			EngagementScore:    50.0,
			SkillScore:         0.5,
			PreferredTimeSlots: make(map[string]int),
		}
		pe.userProfiles[userID] = profile
	}
	
	record := SuccessRecord{
		Timestamp:        time.Now(),
		Difficulty:       difficulty,
		Method:           method,
		TimeSpent:        responseTime.Seconds(),
		UserSatisfaction: 5.0,
	}
	profile.SuccessHistory = append(profile.SuccessHistory, record)
	
	if len(profile.SuccessHistory) > 100 {
		profile.SuccessHistory = profile.SuccessHistory[len(profile.SuccessHistory)-100:]
	}
	
	profile.AvgTime = profile.AvgTime*0.9 + responseTime.Seconds()*0.1
	profile.LastUpdated = time.Now()
	
	if method != "" {
		profile.PreferredMethod = method
	}
	
	profile.updateExperienceMetrics()
	profile.updateSkillScore()
	profile.updateEngagementScore()
}

// updateExperienceMetrics 更新体验指标
func (ep *EnhancedPersonalizedProfile) updateExperienceMetrics() {
	if len(ep.SuccessHistory) < 2 {
		return
	}
	
	for i := len(ep.SuccessHistory) - 5; i < len(ep.SuccessHistory); i++ {
		if i >= 0 {
			ep.AvgSuccessRate = ep.AvgSuccessRate*0.95 + 1.0*0.05
		}
	}
	
	if ep.ExperienceLevel < 10 {
		if ep.AvgSuccessRate > 90 {
			ep.ExperienceLevel++
		}
	}
	
	ep.StressLevel = ep.StressLevel * 0.9
	if ep.AvgSuccessRate < 70 {
		ep.StressLevel += 0.1
	}
}

// updateSkillScore 更新技能分数
func (ep *EnhancedPersonalizedProfile) updateSkillScore() {
	if len(ep.SuccessHistory) < 3 {
		return
	}
	
	avgRecentTime := 0.0
	count := 0
	for i := len(ep.SuccessHistory) - 5; i < len(ep.SuccessHistory); i++ {
		if i >= 0 {
			avgRecentTime += ep.SuccessHistory[i].TimeSpent
			count++
		}
	}
	
	if count > 0 {
		avgRecentTime /= float64(count)
	}
	
	if avgRecentTime < 3.0 {
		ep.SkillScore = math.Min(1.0, ep.SkillScore+0.05)
	} else if avgRecentTime > 10.0 {
		ep.SkillScore = math.Max(0.0, ep.SkillScore-0.02)
	}
}

// updateEngagementScore 更新参与度分数
func (ep *EnhancedPersonalizedProfile) updateEngagementScore() {
	if len(ep.SuccessHistory) < 2 {
		return
	}
	
	ep.EngagementScore = ep.EngagementScore*0.95 + 0.5*0.05
	
	ep.ChurnRisk = ep.ChurnRisk * 0.95
	if ep.StressLevel > 0.7 || ep.AvgSuccessRate < 50 {
		ep.ChurnRisk += 0.05
	}
	
	ep.LearningProgress = float64(ep.ExperienceLevel) / 10.0
	
	ep.SafetyBalance = 0.5
	ep.ExperienceBalance = 0.5
	
	if ep.ChurnRisk > 0.6 {
		ep.SafetyBalance = 0.3
		ep.ExperienceBalance = 0.7
	}
}

// getMetrics 获取体验指标
func (ebm *ExperienceBalanceManager) getMetrics(userID string) *ExperienceMetrics {
	ebm.mu.RLock()
	defer ebm.mu.RUnlock()
	
	if metrics, exists := ebm.experienceMetrics[userID]; exists {
		return metrics
	}
	
	return &ExperienceMetrics{
		UserID:            userID,
		SatisfactionScore: 5.0,
		FrustrationLevel:  0.0,
		OverallExperience: 5.0,
	}
}

// updateExperienceMetrics 更新体验指标
func (ebm *ExperienceBalanceManager) updateExperienceMetrics(
	userID string,
	success bool,
	responseTime time.Duration,
) {
	ebm.mu.Lock()
	defer ebm.mu.Unlock()
	
	metrics, exists := ebm.experienceMetrics[userID]
	if !exists {
		metrics = &ExperienceMetrics{
			UserID: userID,
		}
		ebm.experienceMetrics[userID] = metrics
	}
	
	if success {
		metrics.SuccessRateTrend = metrics.SuccessRateTrend*0.95 + 0.05
		metrics.CompletionRate = metrics.CompletionRate*0.95 + 0.05
		metrics.SatisfactionScore = metrics.SatisfactionScore*0.9 + 5.0*0.1
		metrics.FrustrationLevel = metrics.FrustrationLevel * 0.8
	} else {
		metrics.SuccessRateTrend = metrics.SuccessRateTrend * 0.95
		metrics.SatisfactionScore = metrics.SatisfactionScore * 0.9
		metrics.FrustrationLevel = math.Min(1.0, metrics.FrustrationLevel+0.2)
	}
	
	metrics.TimeTrend = metrics.TimeTrend*0.9 + responseTime.Seconds()*0.1
	metrics.DropoutRisk = metrics.calculateDropoutRisk(ebm.balanceThresholds)
	
	if responseTime.Seconds() > ebm.balanceThresholds.MaxTimeAllowed {
		metrics.DropoutRisk = math.Min(1.0, metrics.DropoutRisk+0.3)
	}
	
	metrics.OverallExperience = metrics.calculateOverallExperience()
}

// calculateDropoutRisk 计算流失风险
func (em *ExperienceMetrics) calculateDropoutRisk(thresholds *BalanceThresholds) float64 {
	risk := 0.0
	
	if em.SatisfactionScore < thresholds.MinSatisfactionScore {
		risk += 0.3
	}
	
	if em.FrustrationLevel > thresholds.MaxFrustrationLevel {
		risk += 0.4
	}
	
	if em.CompletionRate < thresholds.MinCompletionRate {
		risk += 0.3
	}
	
	return math.Min(1.0, risk)
}

// calculateOverallExperience 计算整体体验
func (em *ExperienceMetrics) calculateOverallExperience() float64 {
	experience := (em.SatisfactionScore/5.0)*0.4 + 
				(1.0-em.FrustrationLevel)*0.3 + 
				em.CompletionRate*0.3
	
	return experience * 10.0
}

func (pe *PersonalizationEngine) getProfile(userID string) *PersonalizedProfile {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.userProfiles[userID]
}

func (pe *PersonalizationEngine) updateProfile(
	userID string,
	difficulty DifficultyLevel,
	method string,
	responseTime time.Duration,
) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	profile, exists := pe.userProfiles[userID]
	if !exists {
		profile = &EnhancedPersonalizedProfile{
			UserID:           userID,
			AvgSuccessRate:   80.0,
			AvgTime:          5.0,
			SuccessHistory:   make([]SuccessRecord, 0),
		}
		pe.userProfiles[userID] = profile
	}
	
	record := SuccessRecord{
		Timestamp:      time.Now(),
		Difficulty:     difficulty,
		Method:         method,
		TimeSpent:      responseTime.Seconds(),
		UserSatisfaction: 5.0,
	}
	profile.SuccessHistory = append(profile.SuccessHistory, record)
	
	if len(profile.SuccessHistory) > 100 {
		profile.SuccessHistory = profile.SuccessHistory[len(profile.SuccessHistory)-100:]
	}
	
	profile.AvgTime = profile.AvgTime*0.9 + responseTime.Seconds()*0.1
	profile.LastUpdated = time.Now()
	
	if method != "" {
		profile.PreferredMethod = method
	}
}

func (pe *PersonalizationEngine) difficultyToScore(d DifficultyLevel) float64 {
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

func (pe *PersonalizationEngine) scoreToDifficulty(score float64) DifficultyLevel {
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

type DifficultyContext struct {
	HighRiskContext       bool
	TimeSensitive         bool
	UserRequestedDifficulty string
}

type DifficultyRecommendation struct {
	RecommendedDifficulty DifficultyLevel
	BaseDifficulty        DifficultyLevel
	PersonalizationBonus  float64
	HistoricalAdjustment  float64
	RealtimeAdjustment    *Adjustment
	Confidence            float64
	AlternativeMethods    []string
	Reasoning             string
}

type UserAnalyticsReport struct {
	UserID          string
	TotalAttempts   int
	SuccessCount    int
	SuccessRate     float64
	AvgTime         float64
	MedianTime      float64
	Trend           string
	PreferredMethod string
	OptimalTimes    []int
	AvgDifficulty   float64
}

func limitHistory(history *UserHistoricalData) {
	maxDays := 30
	if len(history.VerificationHistory) > maxDays {
		history.VerificationHistory = history.VerificationHistory[len(history.VerificationHistory)-maxDays:]
	}
}

func calculateTrend(history [][]VerificationAttempt) string {
	if len(history) < 7 {
		return "stable"
	}
	
	recentSuccess := 0
	recentTotal := 0
	oldSuccess := 0
	oldTotal := 0
	
	for i, day := range history {
		for _, attempt := range day {
			if i >= len(history)-3 {
				recentTotal++
				if attempt.Success {
					recentSuccess++
				}
			}
			if i < len(history)-7 && i >= len(history)-7 {
				oldTotal++
				if attempt.Success {
					oldSuccess++
				}
			}
		}
	}
	
	if recentTotal == 0 || oldTotal == 0 {
		return "stable"
	}
	
	recentRate := float64(recentSuccess) / float64(recentTotal)
	oldRate := float64(oldSuccess) / float64(oldTotal)
	
	diff := recentRate - oldRate
	if diff > 0.1 {
		return "improving"
	} else if diff < -0.1 {
		return "declining"
	}
	return "stable"
}

func meanFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func varianceFloat(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := meanFloat(values)
	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-mean, 2)
	}
	return sum / float64(len(values))
}
