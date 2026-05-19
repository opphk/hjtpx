package service

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

type DifficultyLevelV2 string

const (
	DifficultyV2Easy   DifficultyLevelV2 = "easy"
	DifficultyV2Medium DifficultyLevelV2 = "medium"
	DifficultyV2Hard   DifficultyLevelV2 = "hard"
	DifficultyV2Expert DifficultyLevelV2 = "expert"
)

type RiskScoreComponent struct {
	DeviceRisk        float64
	BehavioralRisk     float64
	HistoricalRisk     float64
	ContextualRisk     float64
	NetworkRisk        float64
	GeolocationRisk    float64
	TimePatternRisk    float64
}

type DifficultyRiskScore struct {
	TotalScore        float64
	Components        *RiskScoreComponent
	Confidence        float64
	DataSufficiency   float64
	AnomalyIndicators []string
	LastCalculated    time.Time
}

type DifficultyBehaviorPattern struct {
	ClickIntervalStats  *IntervalStats
	MouseSpeedStats     *SpeedStats
	TrajectoryComplexity float64
	PreferredTimes       map[int]int
	SuccessRateByHour    map[int]float64
	ResponseTimeTrend    []float64
}

type DifficultyVerificationResult struct {
	Timestamp      time.Time
	Difficulty     DifficultyLevelV2
	Success        bool
	FailureReason  string
	ResponseTime   time.Duration
}


type UserRiskProfileV2 struct {
	UserID           string
	CompositeRisk    *DifficultyRiskScore
	SuccessHistory   []*DifficultyVerificationResult
	FailureHistory   []*DifficultyVerificationResult
	SessionMetrics   *SessionMetrics
	BehaviorPattern  *DifficultyBehaviorPattern
	DeviceTrust      *DeviceTrust
	LastUpdated      time.Time
	RetryState       *RetryState
	TimeoutState     *TimeoutState
}

type SessionMetrics struct {
	TotalAttempts      int
	SuccessfulAttempts  int
	FailedAttempts     int
	AverageTime        float64
	MedianTime         float64
	FastAttempts       int
	SlowAttempts       int
	TimeoutAttempts    int
	CurrentStreak      int
	BestStreak         int
	SessionStartTime   time.Time
	LastAttemptTime    time.Time
}

type IntervalStats struct {
	Mean     float64
	Median   float64
	StdDev   float64
	Min      float64
	Max      float64
	Skewness float64
}

type SpeedStats struct {
	Average    float64
	MaxSpeed   float64
	MinSpeed   float64
	Variance   float64
	Jitter     float64
}

type TypingRhythm struct {
	AverageInterval float64
	StdDeviation    float64
	ErrorRate       float64
}

type DeviceTrust struct {
	DeviceID         string
	Fingerprint      string
	TrustScore       float64
	IsKnownDevice    bool
	DeviceSwitchCount int
	LastDeviceUsed   time.Time
}

type RetryState struct {
	CurrentRetry    int
	MaxRetries      int
	BackoffStrategy *BackoffStrategy
	LastRetryTime   time.Time
	TotalRetries    int
	RetryHistory    []time.Time
}

type BackoffStrategy struct {
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	Multiplier     float64
	CurrentDelay   time.Duration
	JitterFactor   float64
	CurrentRetry   int
}

type TimeoutState struct {
	IsActive        bool
	StartTime       time.Time
	TimeoutDuration time.Duration
	GracePeriod     time.Duration
	RemainingTime   time.Duration
	Extensions      int
	MaxExtensions   int
}

type DynamicDifficultyConfig struct {
	BaseThreshold    float64
	AdjustmentSpeed  float64
	SmoothingFactor  float64
	MinDifficulty    DifficultyLevelV2
	MaxDifficulty    DifficultyLevelV2
	AggressionMode   bool
	ConservativeMode bool
}

type DifficultyAdjustment struct {
	FromLevel        DifficultyLevelV2
	ToLevel          DifficultyLevelV2
	AdjustmentReason string
	AdjustmentFactor float64
	SmoothingApplied bool
	TransitionState  *TransitionInfo
}

type TransitionInfo struct {
	IsTransitioning bool
	StepsRemaining  int
	TotalSteps      int
	Progress        float64
	StartedAt       time.Time
	EstimatedEnd    time.Time
}

type AdaptiveDifficultyServiceV2 struct {
	profiles           map[string]*UserRiskProfileV2
	config             *DynamicDifficultyConfig
	mu                 sync.RWMutex
	timeoutManager     *TimeoutManager
	retryManager       *RetryManager
	riskCalculator     *MultiDimensionalRiskCalculator
	difficultyEngine   *DifficultyAdjustmentEngine
}

type TimeoutManager struct {
	activeTimeouts map[string]*TimeoutState
	defaultTimeout time.Duration
	gracePeriod    time.Duration
	mu             sync.RWMutex
}

type RetryManager struct {
	retryStates      map[string]*RetryState
	defaultMaxRetry int
	backoffConfig    *BackoffStrategy
	mu               sync.RWMutex
}

type MultiDimensionalRiskCalculator struct {
	componentWeights map[string]float64
	anomalyThresholds *AnomalyThresholds
	mu               sync.RWMutex
}

type AnomalyThresholds struct {
	DeviceSwitchWeight     float64
	SpeedAnomalyWeight     float64
	TimeAnomalyWeight      float64
	GeographicWeight       float64
	NetworkAnomalyWeight   float64
}

type DifficultyAdjustmentEngine struct {
	adjustmentHistory map[string][]*DifficultyAdjustment
	smoothingWindow   int
	trendAnalyzer     *TrendAnalyzer
	mu               sync.RWMutex
}

type TrendAnalyzer struct {
	windowSize   int
	sensitivity  float64
	trendCache   map[string]*TrendData
}

type TrendData struct {
	Direction      string
	Velocity       float64
	Acceleration   float64
	Seasonality   map[int]float64
	Confidence     float64
}

func NewAdaptiveDifficultyServiceV2() *AdaptiveDifficultyServiceV2 {
	service := &AdaptiveDifficultyServiceV2{
		profiles: make(map[string]*UserRiskProfileV2),
		config: &DynamicDifficultyConfig{
			BaseThreshold:    50.0,
			AdjustmentSpeed: 0.3,
			SmoothingFactor: 0.4,
			MinDifficulty:   DifficultyV2Easy,
			MaxDifficulty:   DifficultyV2Expert,
			AggressionMode:  false,
			ConservativeMode: false,
		},
		timeoutManager:   NewTimeoutManager(),
		retryManager:     NewRetryManager(),
		riskCalculator:   NewMultiDimensionalRiskCalculator(),
		difficultyEngine: NewDifficultyAdjustmentEngine(),
	}
	return service
}

func NewTimeoutManager() *TimeoutManager {
	return &TimeoutManager{
		activeTimeouts:  make(map[string]*TimeoutState),
		defaultTimeout: 60 * time.Second,
		gracePeriod:    10 * time.Second,
	}
}

func NewRetryManager() *RetryManager {
	return &RetryManager{
		retryStates:     make(map[string]*RetryState),
		defaultMaxRetry: 3,
		backoffConfig: &BackoffStrategy{
			InitialDelay: 5 * time.Second,
			MaxDelay:    60 * time.Second,
			Multiplier:  2.0,
			JitterFactor: 0.2,
		},
	}
}

func NewMultiDimensionalRiskCalculator() *MultiDimensionalRiskCalculator {
	return &MultiDimensionalRiskCalculator{
		componentWeights: map[string]float64{
			"device":        0.15,
			"behavioral":    0.25,
			"historical":    0.20,
			"contextual":    0.15,
			"network":       0.10,
			"geolocation":   0.08,
			"time_pattern":  0.07,
		},
		anomalyThresholds: &AnomalyThresholds{
			DeviceSwitchWeight:   15.0,
			SpeedAnomalyWeight:  20.0,
			TimeAnomalyWeight:   10.0,
			GeographicWeight:     25.0,
			NetworkAnomalyWeight: 18.0,
		},
	}
}

func NewDifficultyAdjustmentEngine() *DifficultyAdjustmentEngine {
	return &DifficultyAdjustmentEngine{
		adjustmentHistory: make(map[string][]*DifficultyAdjustment),
		smoothingWindow:   5,
		trendAnalyzer: &TrendAnalyzer{
			windowSize:  10,
			sensitivity: 0.15,
			trendCache:  make(map[string]*TrendData),
		},
	}
}

func (s *AdaptiveDifficultyServiceV2) CalculateMultiDimensionalRiskScore(userID string, context *RiskContextV2) *DifficultyRiskScore {
	profile := s.GetOrCreateProfileV2(userID)

	components := &RiskScoreComponent{}

	components.DeviceRisk = s.calculateDeviceRisk(profile, context)
	components.BehavioralRisk = s.calculateBehavioralRisk(profile, context)
	components.HistoricalRisk = s.calculateHistoricalRisk(profile)
	components.ContextualRisk = s.calculateContextualRisk(profile, context)
	components.NetworkRisk = s.calculateNetworkRisk(context)
	components.GeolocationRisk = s.calculateGeolocationRisk(profile, context)
	components.TimePatternRisk = s.calculateTimePatternRisk(profile, context)

	totalScore := components.DeviceRisk*0.15 +
		components.BehavioralRisk*0.25 +
		components.HistoricalRisk*0.20 +
		components.ContextualRisk*0.15 +
		components.NetworkRisk*0.10 +
		components.GeolocationRisk*0.08 +
		components.TimePatternRisk*0.07

	anomalyIndicators := s.detectAnomalyIndicators(components)

	confidence := s.calculateRiskConfidence(profile, context)

	return &DifficultyRiskScore{
		TotalScore:        math.Min(100, math.Max(0, totalScore)),
		Components:        components,
		Confidence:        confidence,
		DataSufficiency:   s.calculateDataSufficiency(profile),
		AnomalyIndicators: anomalyIndicators,
		LastCalculated:    time.Now(),
	}
}

func (s *AdaptiveDifficultyServiceV2) calculateDeviceRisk(profile *UserRiskProfileV2, context *RiskContextV2) float64 {
	risk := 0.0

	if profile != nil && profile.DeviceTrust != nil {
		if profile.DeviceTrust.IsKnownDevice {
			risk -= 20.0
		}

		risk += float64(profile.DeviceTrust.DeviceSwitchCount) * 15.0

		if profile.DeviceTrust.TrustScore < 50 {
			risk += (50 - profile.DeviceTrust.TrustScore) * 0.5
		}
	}

	if context != nil {
		if context.NewDevice {
			risk += 25.0
		}
		if context.FingerprintMismatch {
			risk += 30.0
		}
	}

	return math.Min(100, math.Max(0, risk))
}

func (s *AdaptiveDifficultyServiceV2) calculateBehavioralRisk(profile *UserRiskProfileV2, context *RiskContextV2) float64 {
	risk := 0.0

	if profile != nil && profile.BehaviorPattern != nil {
		if profile.BehaviorPattern.ClickIntervalStats != nil {
			if profile.BehaviorPattern.ClickIntervalStats.StdDev < 0.1 {
				risk += 30.0
			}
		}

		if profile.BehaviorPattern.MouseSpeedStats != nil {
			if profile.BehaviorPattern.MouseSpeedStats.Jitter < 0.05 {
				risk += 25.0
			}
			if profile.BehaviorPattern.MouseSpeedStats.MaxSpeed > 2000 {
				risk += 15.0
			}
		}

		if profile.BehaviorPattern.TrajectoryComplexity < 0.3 {
			risk += 20.0
		}
	}

	if context != nil {
		if context.MechanicalBehavior {
			risk += 35.0
		}
		if context.UnnaturalSpeed {
			risk += 25.0
		}
		if context.NoHumanPause {
			risk += 15.0
		}
	}

	return math.Min(100, math.Max(0, risk))
}

func (s *AdaptiveDifficultyServiceV2) calculateHistoricalRisk(profile *UserRiskProfileV2) float64 {
	if profile == nil || len(profile.SuccessHistory) == 0 {
		return 50.0
	}

	risk := 0.0

	totalAttempts := len(profile.SuccessHistory) + len(profile.FailureHistory)
	if totalAttempts < 3 {
		return 40.0
	}

	successRate := float64(len(profile.SuccessHistory)) / float64(totalAttempts)

	if successRate < 0.5 {
		risk += (0.5 - successRate) * 60
	} else if successRate > 0.9 {
		risk -= 20.0
	}

	if profile.SessionMetrics != nil {
		if profile.SessionMetrics.TimeoutAttempts > 2 {
			risk += float64(profile.SessionMetrics.TimeoutAttempts) * 10
		}

		if profile.SessionMetrics.CurrentStreak > 5 {
			risk -= 15.0
		}

		if profile.SessionMetrics.FastAttempts > totalAttempts/2 {
			risk += 20.0
		}
	}

	if len(profile.FailureHistory) > 0 {
		recentFailures := 0
		for i := len(profile.FailureHistory) - 1; i >= 0 && i >= len(profile.FailureHistory)-3; i-- {
			if time.Since(profile.FailureHistory[i].Timestamp) < 5*time.Minute {
				recentFailures++
			}
		}
		risk += float64(recentFailures) * 15
	}

	return math.Min(100, math.Max(0, risk))
}

func (s *AdaptiveDifficultyServiceV2) calculateContextualRisk(profile *UserRiskProfileV2, context *RiskContextV2) float64 {
	risk := 0.0

	if context != nil {
		if context.HighTrafficPeriod {
			risk += 10.0
		}

		if context.UnusualTime {
			risk += 15.0
		}

		if context.MultipleAccountsFromIP > 3 {
			risk += 20.0
		}

		if context.SuspiciousReferer {
			risk += 10.0
		}
	}

	if profile != nil && profile.TimeoutState != nil {
		if profile.TimeoutState.IsActive {
			risk += 25.0
		}
		if profile.TimeoutState.Extensions > 0 {
			risk += float64(profile.TimeoutState.Extensions) * 5
		}
	}

	return math.Min(100, math.Max(0, risk))
}

func (s *AdaptiveDifficultyServiceV2) calculateNetworkRisk(context *RiskContextV2) float64 {
	risk := 0.0

	if context == nil {
		return 20.0
	}

	if context.IsVPN {
		risk += 25.0
	}
	if context.IsProxy {
		risk += 20.0
	}
	if context.IsTor {
		risk += 35.0
	}
	if context.IsHosting {
		risk += 30.0
	}
	if context.IsDatacenter {
		risk += 25.0
	}

	if context.NetworkQuality < 0.3 {
		risk += 15.0
	}

	return math.Min(100, math.Max(0, risk))
}

func (s *AdaptiveDifficultyServiceV2) calculateGeolocationRisk(profile *UserRiskProfileV2, context *RiskContextV2) float64 {
	risk := 0.0

	if context == nil || profile == nil {
		return 15.0
	}

	if len(profile.SuccessHistory) > 0 {
		lastLocation := profile.SuccessHistory[len(profile.SuccessHistory)-1].Timestamp
		_ = lastLocation

		if context.IPReputation == "bad" {
			risk += 30.0
		} else if context.IPReputation == "suspicious" {
			risk += 15.0
		}

		if context.CountryChange {
			risk += 40.0
		}

		if context.ASNumberChange {
			risk += 25.0
		}
	}

	return math.Min(100, math.Max(0, risk))
}

func (s *AdaptiveDifficultyServiceV2) calculateTimePatternRisk(profile *UserRiskProfileV2, context *RiskContextV2) float64 {
	risk := 0.0

	if profile == nil || profile.BehaviorPattern == nil || len(profile.BehaviorPattern.PreferredTimes) == 0 {
		return 10.0
	}

	currentHour := time.Now().Hour()

	if count, exists := profile.BehaviorPattern.PreferredTimes[currentHour]; exists {
		if count < 2 {
			risk += 20.0
		}
	} else {
		risk += 15.0
	}

	if context != nil && context.UnusualTime {
		risk += 10.0
	}

	return math.Min(100, math.Max(0, risk))
}

func (s *AdaptiveDifficultyServiceV2) detectAnomalyIndicators(components *RiskScoreComponent) []string {
	indicators := []string{}

	if components.DeviceRisk > 70 {
		indicators = append(indicators, "high_device_risk")
	}
	if components.BehavioralRisk > 75 {
		indicators = append(indicators, "mechanical_behavior_detected")
	}
	if components.HistoricalRisk > 60 {
		indicators = append(indicators, "poor_historical_performance")
	}
	if components.ContextualRisk > 65 {
		indicators = append(indicators, "suspicious_context")
	}
	if components.NetworkRisk > 80 {
		indicators = append(indicators, "high_risk_network")
	}
	if components.GeolocationRisk > 70 {
		indicators = append(indicators, "location_anomaly")
	}
	if components.TimePatternRisk > 60 {
		indicators = append(indicators, "unusual_time_pattern")
	}

	return indicators
}

func (s *AdaptiveDifficultyServiceV2) calculateRiskConfidence(profile *UserRiskProfileV2, context *RiskContextV2) float64 {
	confidence := 0.5

	totalAttempts := 0
	if profile != nil {
		totalAttempts = len(profile.SuccessHistory) + len(profile.FailureHistory)
	}

	if totalAttempts > 20 {
		confidence += 0.3
	} else if totalAttempts > 10 {
		confidence += 0.2
	} else if totalAttempts > 5 {
		confidence += 0.1
	}

	if profile != nil && profile.DeviceTrust != nil && profile.DeviceTrust.IsKnownDevice {
		confidence += 0.1
	}

	if context != nil {
		if context.HasCompleteFingerprint {
			confidence += 0.1
		}
	}

	return math.Min(0.95, confidence)
}

func (s *AdaptiveDifficultyServiceV2) calculateDataSufficiency(profile *UserRiskProfileV2) float64 {
	if profile == nil {
		return 0.2
	}

	sufficiency := 0.3

	totalAttempts := len(profile.SuccessHistory) + len(profile.FailureHistory)
	if totalAttempts > 20 {
		sufficiency += 0.3
	} else if totalAttempts > 10 {
		sufficiency += 0.2
	} else if totalAttempts > 5 {
		sufficiency += 0.1
	}

	if profile.DeviceTrust != nil && profile.DeviceTrust.IsKnownDevice {
		sufficiency += 0.15
	}

	if profile.BehaviorPattern != nil && len(profile.BehaviorPattern.ResponseTimeTrend) > 10 {
		sufficiency += 0.15
	}

	if profile.SessionMetrics != nil && profile.SessionMetrics.TotalAttempts > 10 {
		sufficiency += 0.1
	}

	return math.Min(1.0, sufficiency)
}

func (s *AdaptiveDifficultyServiceV2) GetOrCreateProfileV2(userID string) *UserRiskProfileV2 {
	s.mu.Lock()
	defer s.mu.Unlock()

	if profile, exists := s.profiles[userID]; exists {
		return profile
	}

	profile := &UserRiskProfileV2{
		UserID:          userID,
		CompositeRisk:   &DifficultyRiskScore{TotalScore: 50.0},
		SuccessHistory:  make([]*DifficultyVerificationResult, 0),
		FailureHistory:  make([]*DifficultyVerificationResult, 0),
		SessionMetrics: &SessionMetrics{
			SessionStartTime: time.Now(),
		},
		BehaviorPattern: &DifficultyBehaviorPattern{
			PreferredTimes:    make(map[int]int),
			SuccessRateByHour: make(map[int]float64),
			ResponseTimeTrend: make([]float64, 0),
		},
		DeviceTrust: &DeviceTrust{
			TrustScore: 50.0,
		},
		LastUpdated: time.Now(),
		RetryState: &RetryState{
			MaxRetries: s.retryManager.defaultMaxRetry,
			BackoffStrategy: &BackoffStrategy{
				InitialDelay: 5 * time.Second,
				MaxDelay:     60 * time.Second,
				Multiplier:   2.0,
				JitterFactor:  0.2,
			},
			RetryHistory: make([]time.Time, 0),
		},
		TimeoutState: &TimeoutState{
			TimeoutDuration: s.timeoutManager.defaultTimeout,
			GracePeriod:     s.timeoutManager.gracePeriod,
			MaxExtensions:    2,
		},
	}

	s.profiles[userID] = profile
	return profile
}

func (s *AdaptiveDifficultyServiceV2) AdjustDifficultyDynamically(userID string, riskScore *DifficultyRiskScore) (DifficultyLevelV2, *DifficultyAdjustment) {
	profile := s.GetOrCreateProfileV2(userID)
	currentDifficulty := s.getCurrentDifficultyFromProfile(profile)

	baseDifficulty := s.riskScoreToDifficulty(riskScore.TotalScore)

	var adjustment *DifficultyAdjustment

	if profile.SessionMetrics != nil {
		baseDifficulty = s.applySessionBasedAdjustment(profile.SessionMetrics, baseDifficulty)
	}

	if riskScore.Components != nil && len(riskScore.AnomalyIndicators) > 2 {
		baseDifficulty = s.applyAnomalyAdjustment(riskScore, baseDifficulty)
	}

	baseDifficulty = s.applySmoothing(profile, currentDifficulty, baseDifficulty)

	if s.config.AggressionMode {
		baseDifficulty = s.applyAggressionMode(baseDifficulty)
	} else if s.config.ConservativeMode {
		baseDifficulty = s.applyConservativeMode(baseDifficulty)
	}

	if baseDifficulty != currentDifficulty {
		adjustment = &DifficultyAdjustment{
			FromLevel:        currentDifficulty,
			ToLevel:          baseDifficulty,
			AdjustmentReason: s.generateAdjustmentReason(riskScore),
			AdjustmentFactor: s.calculateAdjustmentFactor(currentDifficulty, baseDifficulty),
			SmoothingApplied: true,
		}
		s.recordAdjustment(userID, adjustment)
	}

	return baseDifficulty, adjustment
}

func (s *AdaptiveDifficultyServiceV2) getCurrentDifficultyFromProfile(profile *UserRiskProfileV2) DifficultyLevelV2 {
	if profile == nil || len(profile.SuccessHistory) == 0 {
		return DifficultyV2Medium
	}

	lastResult := profile.SuccessHistory[len(profile.SuccessHistory)-1]
	if len(profile.FailureHistory) > 0 {
		lastFailure := profile.FailureHistory[len(profile.FailureHistory)-1]
		if lastFailure.Timestamp.After(lastResult.Timestamp) {
			lastResult = &DifficultyVerificationResult{
				Difficulty: DifficultyLevelV2(lastFailure.Difficulty),
			}
		}
	}

	if lastResult == nil {
		return DifficultyV2Medium
	}

	return DifficultyLevelV2(lastResult.Difficulty)
}

func (s *AdaptiveDifficultyServiceV2) riskScoreToDifficulty(riskScore float64) DifficultyLevelV2 {
	switch {
	case riskScore < 25:
		return DifficultyV2Easy
	case riskScore < 50:
		return DifficultyV2Medium
	case riskScore < 75:
		return DifficultyV2Hard
	default:
		return DifficultyV2Expert
	}
}

func (s *AdaptiveDifficultyServiceV2) applySessionBasedAdjustment(metrics *SessionMetrics, base DifficultyLevelV2) DifficultyLevelV2 {
	if metrics == nil {
		return base
	}

	if metrics.CurrentStreak >= 5 && metrics.CurrentStreak <= 10 {
		base = s.decreaseDifficulty(base)
	} else if metrics.CurrentStreak > 10 {
		base = s.decreaseDifficulty(s.decreaseDifficulty(base))
	}

	if metrics.FailedAttempts > metrics.SuccessfulAttempts && metrics.TotalAttempts > 3 {
		base = s.decreaseDifficulty(base)
	}

	if metrics.TimeoutAttempts >= 2 {
		base = s.decreaseDifficulty(base)
	}

	if metrics.FastAttempts > metrics.TotalAttempts/2 && metrics.TotalAttempts >= 5 {
		base = s.increaseDifficulty(base)
	}

	return base
}

func (s *AdaptiveDifficultyServiceV2) applyAnomalyAdjustment(riskScore *DifficultyRiskScore, base DifficultyLevelV2) DifficultyLevelV2 {
	if riskScore.Components.BehavioralRisk > 70 {
		base = s.increaseDifficulty(base)
	}

	if riskScore.Components.NetworkRisk > 60 {
		base = s.increaseDifficulty(base)
	}

	if len(riskScore.AnomalyIndicators) >= 4 {
		base = s.increaseDifficulty(base)
	}

	return base
}

func (s *AdaptiveDifficultyServiceV2) applySmoothing(profile *UserRiskProfileV2, current, target DifficultyLevelV2) DifficultyLevelV2 {
	if profile == nil {
		return target
	}

	history := s.difficultyEngine.adjustmentHistory[profile.UserID]
	if len(history) < 2 {
		return target
	}

	recentAdjustments := history[len(history)-s.difficultyEngine.smoothingWindow:]
	upCount := 0
	downCount := 0

	for _, adj := range recentAdjustments {
		if adj.ToLevel != adj.FromLevel {
			if s.difficultyToScore(adj.ToLevel) > s.difficultyToScore(adj.FromLevel) {
				upCount++
			} else {
				downCount++
			}
		}
	}

	if upCount >= 3 {
		target = s.decreaseDifficulty(target)
	} else if downCount >= 3 {
		target = s.increaseDifficulty(target)
	}

	return target
}

func (s *AdaptiveDifficultyServiceV2) applyAggressionMode(base DifficultyLevelV2) DifficultyLevelV2 {
	return s.increaseDifficulty(base)
}

func (s *AdaptiveDifficultyServiceV2) applyConservativeMode(base DifficultyLevelV2) DifficultyLevelV2 {
	return s.decreaseDifficulty(base)
}

func (s *AdaptiveDifficultyServiceV2) decreaseDifficulty(level DifficultyLevelV2) DifficultyLevelV2 {
	switch level {
	case DifficultyV2Expert:
		return DifficultyV2Hard
	case DifficultyV2Hard:
		return DifficultyV2Medium
	case DifficultyV2Medium:
		return DifficultyV2Easy
	default:
		return DifficultyV2Easy
	}
}

func (s *AdaptiveDifficultyServiceV2) increaseDifficulty(level DifficultyLevelV2) DifficultyLevelV2 {
	switch level {
	case DifficultyV2Easy:
		return DifficultyV2Medium
	case DifficultyV2Medium:
		return DifficultyV2Hard
	case DifficultyV2Hard:
		return DifficultyV2Expert
	default:
		return DifficultyV2Expert
	}
}

func (s *AdaptiveDifficultyServiceV2) generateAdjustmentReason(riskScore *DifficultyRiskScore) string {
	if riskScore == nil {
		return "基于默认配置调整"
	}

	if len(riskScore.AnomalyIndicators) > 0 {
		return fmt.Sprintf("检测到异常: %s", riskScore.AnomalyIndicators[0])
	}

	if riskScore.Components.BehavioralRisk > 60 {
		return "行为模式异常"
	}

	if riskScore.Components.HistoricalRisk > 50 {
		return "历史表现不佳"
	}

	return "基于综合风险评分调整"
}

func (s *AdaptiveDifficultyServiceV2) calculateAdjustmentFactor(from, to DifficultyLevelV2) float64 {
	return math.Abs(s.difficultyToScore(to) - s.difficultyToScore(from)) * 0.5
}

func (s *AdaptiveDifficultyServiceV2) difficultyToScore(level DifficultyLevelV2) float64 {
	switch level {
	case DifficultyV2Easy:
		return 0
	case DifficultyV2Medium:
		return 1
	case DifficultyV2Hard:
		return 2
	case DifficultyV2Expert:
		return 3
	default:
		return 1
	}
}

func (s *AdaptiveDifficultyServiceV2) recordAdjustment(userID string, adjustment *DifficultyAdjustment) {
	s.difficultyEngine.mu.Lock()
	defer s.difficultyEngine.mu.Unlock()

	history := s.difficultyEngine.adjustmentHistory[userID]
	history = append(history, adjustment)

	if len(history) > 50 {
		history = history[len(history)-50:]
	}

	s.difficultyEngine.adjustmentHistory[userID] = history
}

func (s *AdaptiveDifficultyServiceV2) HandleTimeout(userID string) *TimeoutState {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.GetOrCreateProfileV2(userID)
	timeout := profile.TimeoutState

	if timeout == nil {
		timeout = &TimeoutState{
			TimeoutDuration: s.timeoutManager.defaultTimeout,
			GracePeriod:      s.timeoutManager.gracePeriod,
			MaxExtensions:    2,
		}
	}

	if !timeout.IsActive {
		timeout.IsActive = true
		timeout.StartTime = time.Now()
		timeout.RemainingTime = timeout.TimeoutDuration
		timeout.Extensions = 0
		return timeout
	}

	elapsed := time.Since(timeout.StartTime)
	timeout.RemainingTime = timeout.TimeoutDuration - elapsed

	if timeout.RemainingTime <= 0 {
		if timeout.Extensions < timeout.MaxExtensions {
			timeout.Extensions++
			timeout.StartTime = time.Now()
			timeout.RemainingTime = timeout.TimeoutDuration + timeout.GracePeriod
		} else {
			timeout.IsActive = false
			s.recordTimeoutAttempt(userID)
		}
	}

	return timeout
}

func (s *AdaptiveDifficultyServiceV2) recordTimeoutAttempt(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.profiles[userID]
	if profile == nil {
		return
	}

	if profile.SessionMetrics != nil {
		profile.SessionMetrics.TimeoutAttempts++
	}

	failure := &DifficultyVerificationResult{
		Timestamp:     time.Now(),
		Success:       false,
		FailureReason: "timeout",
	}
	profile.FailureHistory = append(profile.FailureHistory, failure)

	if len(profile.FailureHistory) > 100 {
		profile.FailureHistory = profile.FailureHistory[len(profile.FailureHistory)-100:]
	}
}

func (s *AdaptiveDifficultyServiceV2) CheckTimeoutStatus(userID string) *TimeoutState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile := s.profiles[userID]
	if profile == nil || profile.TimeoutState == nil {
		return nil
	}

	timeout := profile.TimeoutState
	if timeout.IsActive {
		elapsed := time.Since(timeout.StartTime)
		timeout.RemainingTime = timeout.TimeoutDuration - elapsed
	}

	return timeout
}

func (s *AdaptiveDifficultyServiceV2) CancelTimeout(userID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.profiles[userID]
	if profile == nil || profile.TimeoutState == nil {
		return false
	}

	profile.TimeoutState.IsActive = false
	profile.TimeoutState.RemainingTime = 0
	return true
}

func (s *AdaptiveDifficultyServiceV2) ShouldAllowRetry(userID string) (bool, *RetryState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.GetOrCreateProfileV2(userID)
	retryState := profile.RetryState

	if retryState == nil {
		retryState = &RetryState{
			MaxRetries: s.retryManager.defaultMaxRetry,
			RetryHistory: make([]time.Time, 0),
		}
		profile.RetryState = retryState
	}

	if retryState.CurrentRetry >= retryState.MaxRetries {
		timeSinceLastRetry := time.Since(retryState.LastRetryTime)
		if timeSinceLastRetry < retryState.BackoffStrategy.MaxDelay {
			return false, retryState
		}
		retryState.CurrentRetry = 0
	}

	elapsed := time.Since(retryState.LastRetryTime)
	if elapsed < retryState.BackoffStrategy.CurrentDelay {
		return false, retryState
	}

	return true, retryState
}

func (s *AdaptiveDifficultyServiceV2) RecordRetryAttempt(userID string) *RetryState {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.GetOrCreateProfileV2(userID)
	retryState := profile.RetryState

	if retryState == nil {
		retryState = &RetryState{
			MaxRetries: s.retryManager.defaultMaxRetry,
			RetryHistory: make([]time.Time, 0),
		}
		profile.RetryState = retryState
	}

	retryState.CurrentRetry++
	retryState.TotalRetries++
	retryState.LastRetryTime = time.Now()
	retryState.RetryHistory = append(retryState.RetryHistory, time.Now())

	if len(retryState.RetryHistory) > 50 {
		retryState.RetryHistory = retryState.RetryHistory[len(retryState.RetryHistory)-50:]
	}

	if retryState.BackoffStrategy != nil {
		retryState.BackoffStrategy.CurrentRetry = retryState.CurrentRetry
		retryState.BackoffStrategy.CurrentDelay = calculateNextDelayForRetry(retryState.BackoffStrategy)
	}

	return retryState
}

func calculateNextDelayForRetry(bs *BackoffStrategy) time.Duration {
	if bs.InitialDelay == 0 {
		bs.InitialDelay = 5 * time.Second
	}
	if bs.MaxDelay == 0 {
		bs.MaxDelay = 60 * time.Second
	}
	if bs.Multiplier == 0 {
		bs.Multiplier = 2.0
	}

	delay := float64(bs.InitialDelay) * math.Pow(bs.Multiplier, float64(bs.CurrentRetry))
	delay = math.Min(delay, float64(bs.MaxDelay))

	if bs.JitterFactor > 0 {
		jitter := delay * bs.JitterFactor * (2*math.Mod(float64(time.Now().UnixNano()), 1.0) - 1)
		delay += jitter
	}

	return time.Duration(delay)
}

func (rs *RetryState) GetNextRetryDelay() time.Duration {
	if rs.BackoffStrategy == nil {
		return 5 * time.Second
	}
	return rs.BackoffStrategy.CurrentDelay
}

func (rs *RetryState) GetRetryProgress() (current, total int, percentage float64) {
	current = rs.CurrentRetry
	total = rs.MaxRetries
	if total > 0 {
		percentage = float64(current) / float64(total) * 100
	}
	return
}

func (s *AdaptiveDifficultyServiceV2) ResetRetryState(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.profiles[userID]
	if profile != nil && profile.RetryState != nil {
		profile.RetryState.CurrentRetry = 0
		profile.RetryState.LastRetryTime = time.Time{}
		profile.RetryState.RetryHistory = make([]time.Time, 0)
		if profile.RetryState.BackoffStrategy != nil {
			profile.RetryState.BackoffStrategy.CurrentDelay = profile.RetryState.BackoffStrategy.InitialDelay
		}
	}
}

func (s *AdaptiveDifficultyServiceV2) RecordVerificationResult(userID string, result *DifficultyVerificationResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.GetOrCreateProfileV2(userID)
	result.Timestamp = time.Now()

	if result.Success {
		profile.SuccessHistory = append(profile.SuccessHistory, result)
		profile.SessionMetrics.SuccessfulAttempts++
		profile.SessionMetrics.CurrentStreak++

		if profile.SessionMetrics.CurrentStreak > profile.SessionMetrics.BestStreak {
			profile.SessionMetrics.BestStreak = profile.SessionMetrics.CurrentStreak
		}
	} else {
		profile.FailureHistory = append(profile.FailureHistory, result)
		profile.SessionMetrics.FailedAttempts++
		profile.SessionMetrics.CurrentStreak = 0
	}

	profile.SessionMetrics.TotalAttempts++
	profile.SessionMetrics.LastAttemptTime = time.Now()

	s.updateSessionMetrics(profile)
	s.updateBehaviorPattern(profile, result)
	s.updateRiskProfile(profile)

	if len(profile.SuccessHistory) > 200 {
		profile.SuccessHistory = profile.SuccessHistory[len(profile.SuccessHistory)-200:]
	}
	if len(profile.FailureHistory) > 100 {
		profile.FailureHistory = profile.FailureHistory[len(profile.FailureHistory)-100:]
	}

	s.CancelTimeout(userID)

	if result.Success {
		s.ResetRetryState(userID)
	}
}

func (s *AdaptiveDifficultyServiceV2) updateSessionMetrics(profile *UserRiskProfileV2) {
	if profile.SessionMetrics == nil {
		return
	}

	if len(profile.SuccessHistory) == 0 {
		return
	}

	times := make([]float64, 0, len(profile.SuccessHistory))
	for _, result := range profile.SuccessHistory {
		times = append(times, result.ResponseTime.Seconds())
	}

	if len(times) > 0 {
		profile.SessionMetrics.AverageTime = meanFloat(times)

		sortedTimes := make([]float64, len(times))
		copy(sortedTimes, times)
		sort.Float64s(sortedTimes)
		profile.SessionMetrics.MedianTime = sortedTimes[len(sortedTimes)/2]
	}

	threshold := 2.0
	for _, result := range profile.SuccessHistory {
		if result.ResponseTime.Seconds() < threshold {
			profile.SessionMetrics.FastAttempts++
		} else if result.ResponseTime.Seconds() > 15.0 {
			profile.SessionMetrics.SlowAttempts++
		}
	}
}

func (s *AdaptiveDifficultyServiceV2) updateBehaviorPattern(profile *UserRiskProfileV2, result *DifficultyVerificationResult) {
	if profile.BehaviorPattern == nil {
		profile.BehaviorPattern = &DifficultyBehaviorPattern{
			PreferredTimes:    make(map[int]int),
			SuccessRateByHour: make(map[int]float64),
			ResponseTimeTrend: make([]float64, 0),
		}
	}

	hour := result.Timestamp.Hour()
	profile.BehaviorPattern.PreferredTimes[hour]++

	profile.BehaviorPattern.ResponseTimeTrend = append(
		profile.BehaviorPattern.ResponseTimeTrend,
		result.ResponseTime.Seconds(),
	)

	if len(profile.BehaviorPattern.ResponseTimeTrend) > 100 {
		profile.BehaviorPattern.ResponseTimeTrend = profile.BehaviorPattern.ResponseTimeTrend[len(profile.BehaviorPattern.ResponseTimeTrend)-100:]
	}

	totalForHour := 0
	successForHour := 0
	for _, r := range profile.SuccessHistory {
		if r.Timestamp.Hour() == hour {
			totalForHour++
			if r.Success {
				successForHour++
			}
		}
	}
	if totalForHour > 0 {
		profile.BehaviorPattern.SuccessRateByHour[hour] = float64(successForHour) / float64(totalForHour) * 100
	}
}

func (s *AdaptiveDifficultyServiceV2) updateRiskProfile(profile *UserRiskProfileV2) {
	if profile.CompositeRisk == nil {
		profile.CompositeRisk = &DifficultyRiskScore{}
	}

	if len(profile.SuccessHistory) > 0 && len(profile.FailureHistory) > 0 {
		total := len(profile.SuccessHistory) + len(profile.FailureHistory)
		successRate := float64(len(profile.SuccessHistory)) / float64(total)

		baseRisk := 50.0
		profile.CompositeRisk.TotalScore = baseRisk - (successRate-0.5)*40
	}

	profile.CompositeRisk.LastCalculated = time.Now()
}

func (s *AdaptiveDifficultyServiceV2) GetDifficultyRecommendation(userID string, context *RiskContextV2) (DifficultyLevelV2, float64) {
	riskScore := s.CalculateMultiDimensionalRiskScore(userID, context)

	difficulty, _ := s.AdjustDifficultyDynamically(userID, riskScore)

	return difficulty, riskScore.TotalScore
}

func (s *AdaptiveDifficultyServiceV2) GetUserAnalyticsV2(userID string) *UserAnalyticsV2 {
	profile := s.GetOrCreateProfileV2(userID)

	totalAttempts := len(profile.SuccessHistory) + len(profile.FailureHistory)
	successCount := len(profile.SuccessHistory)

	var successRate float64
	if totalAttempts > 0 {
		successRate = float64(successCount) / float64(totalAttempts) * 100
	}

	var avgTime float64
	if len(profile.SuccessHistory) > 0 {
		times := make([]float64, 0)
		for _, r := range profile.SuccessHistory {
			times = append(times, r.ResponseTime.Seconds())
		}
		avgTime = meanFloat(times)
	}

	trend := s.analyzeTrend(profile)

	return &UserAnalyticsV2{
		UserID:           userID,
		TotalAttempts:    totalAttempts,
		SuccessCount:     successCount,
		FailureCount:     len(profile.FailureHistory),
		SuccessRate:      successRate,
		AverageTime:      avgTime,
		CurrentStreak:    profile.SessionMetrics.CurrentStreak,
		BestStreak:       profile.SessionMetrics.BestStreak,
		RiskScore:        profile.CompositeRisk.TotalScore,
		CurrentDifficulty: s.getCurrentDifficultyFromProfile(profile),
		Trend:            trend,
		RetryState:       profile.RetryState,
		TimeoutAttempts:  profile.SessionMetrics.TimeoutAttempts,
	}
}

func (s *AdaptiveDifficultyServiceV2) analyzeTrend(profile *UserRiskProfileV2) string {
	if len(profile.SuccessHistory) < 5 {
		return "insufficient_data"
	}

	recentSuccessRate := 0.0
	recentCount := 0
	oldSuccessRate := 0.0
	oldCount := 0

	cutoff := len(profile.SuccessHistory) - 5
	for i, r := range profile.SuccessHistory {
		if i >= cutoff {
			recentCount++
			if r.Success {
				recentSuccessRate++
			}
		} else {
			oldCount++
			if r.Success {
				oldSuccessRate++
			}
		}
	}

	if recentCount == 0 || oldCount == 0 {
		return "stable"
	}

	recentRate := recentSuccessRate / float64(recentCount)
	oldRate := oldSuccessRate / float64(oldCount)

	diff := recentRate - oldRate

	if diff > 0.15 {
		return "improving"
	} else if diff < -0.15 {
		return "declining"
	}
	return "stable"
}

type UserAnalyticsV2 struct {
	UserID            string
	TotalAttempts     int
	SuccessCount      int
	FailureCount      int
	SuccessRate       float64
	AverageTime       float64
	CurrentStreak     int
	BestStreak        int
	RiskScore         float64
	CurrentDifficulty DifficultyLevelV2
	Trend             string
	RetryState        *RetryState
	TimeoutAttempts   int
}

type RiskContextV2 struct {
	SessionID           string
	IPAddress           string
	DeviceID            string
	Fingerprint         string
	NewDevice           bool
	FingerprintMismatch bool
	IsVPN               bool
	IsProxy             bool
	IsTor               bool
	IsHosting           bool
	IsDatacenter        bool
	NetworkQuality      float64
	HighTrafficPeriod   bool
	UnusualTime         bool
	MultipleAccountsFromIP int
	SuspiciousReferer   bool
	HasCompleteFingerprint bool
	CountryChange       bool
	ASNumberChange      bool
	IPReputation        string
	MechanicalBehavior  bool
	UnnaturalSpeed      bool
	NoHumanPause        bool
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
