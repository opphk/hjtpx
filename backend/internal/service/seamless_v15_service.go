package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

type SeamlessV15Service struct {
	deviceFingerprintLearner *DeviceFingerprintLearner
	behaviorModeler          *BehaviorModeler
	trustScoreEngine         *TrustScoreEngine
	switchController         *SwitchController
	reportGenerator          *ReportGenerator
	mu                       sync.RWMutex
}

type DeviceFingerprintLearner struct {
	fingerprintModels map[string]*FingerprintModel
	learningConfig     *FingerprintLearningConfig
	mu                 sync.RWMutex
}

type FingerprintLearningConfig struct {
	InitialTrustScore     float64
	LearningRate          float64
	DecayRate             float64
	MinConfidenceSamples  int
	MaxHistorySize        int
	SimilarityThreshold   float64
	UpdateInterval        time.Duration
}

type FingerprintModel struct {
	Fingerprint       string
	ComponentHashes   map[string]string
	UsageHistory       []*FingerprintUsage
	StabilityScore     float64
	ConfidenceLevel    float64
	FirstSeenAt        time.Time
	LastUpdatedAt      time.Time
	SuccessfulVerifies int
	FailedVerifies     int
	AnomalyCount       int
	AverageRiskScore   float64
	FeatureVector      []float64
	ClusterID          int
	Version            int
}

type FingerprintUsage struct {
	Timestamp    time.Time
	IPAddress    string
	UserAgent    string
	RiskScore    float64
	Success      bool
	BehaviorHash string
}

type BehaviorModeler struct {
	userModels    map[string]*V15UserBehaviorModel
	modelConfig   *BehaviorModelConfig
	mu            sync.RWMutex
}

type BehaviorModelConfig struct {
	WindowSize          time.Duration
	FeatureUpdateRate   float64
	AnomalyThreshold    float64
	MinSamplesForModel  int
	HabitDecayFactor    float64
	SeasonalityEnabled  bool
}

type V15UserBehaviorModel struct {
	UserID            string
	SessionPatterns    []*SessionPattern
	TypingProfile      *TypingProfile
	MouseProfile       *MouseProfile
	TimePreferences    *TimePreferences
	DevicePreferences  map[string]*DevicePreference
	LocationHistory    []*LocationRecord
	BehavioralEntropy  float64
	HabitStrength      float64
	ModelConfidence    float64
	LastUpdatedAt     time.Time
	TotalSessions      int
	SuccessfulSessions int
}

type SessionPattern struct {
	SessionID       string
	StartTime       time.Time
	Duration        time.Duration
	MouseMoves      int
	KeyboardEvents  int
	Clicks          int
	ScrollEvents    int
	AverageSpeed    float64
	PatternHash     string
	Outcome         string
}

type TypingProfile struct {
	AverageKeystrokeDelay float64
	KeystrokeDeviation    float64
	TypingRhythm          []float64
	ErrorRate             float64
	CommonMistakes        map[string]int
	AverageWordLength     float64
	SpeedWPM              float64
}

type MouseProfile struct {
	AverageSpeed        float64
	SpeedVariance       float64
	AverageAcceleration float64
	TrajectoryComplexity float64
	ClickPatterns       map[string]int
	ScrollPreferences   *ScrollPreferences
	MovementEntropy      float64
}

type ScrollPreferences struct {
	AverageScrollSpeed float64
	ScrollDirection    map[string]int
	ScrollAmount       float64
}

type LocationRecord struct {
	IPAddress   string
	Location    string
	Coordinates string
	Timestamp   time.Time
	Trusted     bool
}

type DevicePreference struct {
	Fingerprint       string
	DeviceName        string
	TrustLevel        float64
	UseCount          int
	LastUsedAt        time.Time
	KnownLocations    []string
	AverageSessionLen time.Duration
}

type TimePreferences struct {
	PreferredHours    map[int]int
	PreferredDays     map[int]int
	PreferredMonths   map[int]int
	AverageSessionLen time.Duration
	SessionVariance   float64
}

type TrustScoreEngine struct {
	scoreCache     map[string]*CachedTrustScore
	scoringWeights *TrustScoreWeights
	adaptiveParams *AdaptiveScoringParams
	mu             sync.RWMutex
}

type CachedTrustScore struct {
	UserID          string
	Fingerprint     string
	BaseScore       float64
	AdjustedScore   float64
	Factors         map[string]float64
	CalculationTime time.Time
	ExpiresAt       time.Time
}

type TrustScoreWeights struct {
	DeviceWeight      float64
	BehaviorWeight    float64
	TimeWeight        float64
	LocationWeight    float64
	HistoryWeight     float64
	AnomalyWeight     float64
}

type AdaptiveScoringParams struct {
	BaseTrustLevel       float64
	MinTrustThreshold    float64
	MaxTrustThreshold    float64
	SuspiciousThreshold  float64
	RiskAdjustmentRate   float64
	RecoveryRate         float64
}

type SwitchController struct {
	strategyConfig   *SwitchStrategyConfig
	currentStrategy  string
	switchHistory    []*SwitchRecord
	mu               sync.RWMutex
}

type SwitchStrategyConfig struct {
	SeamlessThreshold  float64
	StrongThreshold    float64
	HighRiskThreshold  float64
	EnableProgressive  bool
	ProgressiveSteps   int
	CooldownPeriod     time.Duration
	ForceStrongOnNew   bool
	ForceStrongOnRisk  bool
	EnableABTesting    bool
	TestingRatio       float64
}

type SwitchRecord struct {
	Timestamp       time.Time
	UserID          string
	PreviousState   string
	NewState        string
	Reason          string
	RiskScore       float64
	TrustScore      float64
	BehaviorScore   float64
}

type ReportGenerator struct {
	reportConfig *ReportConfig
	mu           sync.RWMutex
}

type ReportConfig struct {
	ReportInterval  time.Duration
	MetricsToTrack []string
	RetentionDays   int
	EnableRealTime  bool
	AggWindowSize   time.Duration
}

type SeamlessReport struct {
	ReportID         string
	GeneratedAt      time.Time
	PeriodStart      time.Time
	PeriodEnd        time.Time
	Summary          *ReportSummary
	DeviceAnalysis   *DeviceAnalysisReport
	BehaviorAnalysis *BehaviorAnalysisReport
	TrustAnalysis    *TrustAnalysisReport
	SwitchAnalysis   *SwitchAnalysisReport
	Recommendations  []string
}

type ReportSummary struct {
	TotalVerifications      int
	SeamlessVerifications    int
	StrongVerifications      int
	BlockedVerifications     int
	SeamlessRate            float64
	BlockRate               float64
	AverageTrustScore       float64
	AverageRiskScore        float64
	UserSatisfactionScore   float64
	FalsePositiveRate       float64
	FalseNegativeRate       float64
}

type DeviceAnalysisReport struct {
	TotalDevices          int
	TrustedDevices        int
	NewDevices            int
	SuspiciousDevices     int
	DevicesByStability    map[string]int
	TopFingerprintComponents []string
	FingerprintAccuracy   float64
}

type BehaviorAnalysisReport struct {
	TotalUsers             int
	ActiveModels           int
	ModelAccuracy          float64
	AnomalyDetectionRate   float64
	BehavioralEntropyAvg   float64
	HabitStrengthAvg       float64
	CommonPatterns         []string
}

type TrustAnalysisReport struct {
	TrustDistribution     map[string]int
	AverageBaseTrust      float64
	AverageAdjustedTrust   float64
	TrustScoreVariance    float64
	UsersAboveThreshold   int
	UsersBelowThreshold   int
}

type SwitchAnalysisReport struct {
	TotalSwitches         int
	SeamlessToStrong      int
	StrongToSeamless      int
	SwitchReasons         map[string]int
	AverageSwitchLatency  float64
	SwitchSuccessRate     float64
}

func NewSeamlessV15Service() *SeamlessV15Service {
	return &SeamlessV15Service{
		deviceFingerprintLearner: newDeviceFingerprintLearner(),
		behaviorModeler:          newBehaviorModeler(),
		trustScoreEngine:         newTrustScoreEngine(),
		switchController:        newSwitchController(),
		reportGenerator:          newReportGenerator(),
	}
}

func newDeviceFingerprintLearner() *DeviceFingerprintLearner {
	return &DeviceFingerprintLearner{
		fingerprintModels: make(map[string]*FingerprintModel),
		learningConfig: &FingerprintLearningConfig{
			InitialTrustScore:    0.5,
			LearningRate:         0.1,
			DecayRate:            0.05,
			MinConfidenceSamples: 5,
			MaxHistorySize:       1000,
			SimilarityThreshold: 0.85,
			UpdateInterval:       24 * time.Hour,
		},
	}
}

func newBehaviorModeler() *BehaviorModeler {
	return &BehaviorModeler{
		userModels: make(map[string]*V15UserBehaviorModel),
		modelConfig: &BehaviorModelConfig{
			WindowSize:         30 * 24 * time.Hour,
			FeatureUpdateRate:  0.15,
			AnomalyThreshold:   2.5,
			MinSamplesForModel: 10,
			HabitDecayFactor:   0.02,
			SeasonalityEnabled: true,
		},
	}
}

func newTrustScoreEngine() *TrustScoreEngine {
	return &TrustScoreEngine{
		scoreCache: make(map[string]*CachedTrustScore),
		scoringWeights: &TrustScoreWeights{
			DeviceWeight:      0.30,
			BehaviorWeight:    0.25,
			TimeWeight:        0.15,
			LocationWeight:    0.15,
			HistoryWeight:     0.10,
			AnomalyWeight:     0.05,
		},
		adaptiveParams: &AdaptiveScoringParams{
			BaseTrustLevel:      0.5,
			MinTrustThreshold:   0.2,
			MaxTrustThreshold:   0.95,
			SuspiciousThreshold: 0.3,
			RiskAdjustmentRate:  0.1,
			RecoveryRate:        0.05,
		},
	}
}

func newSwitchController() *SwitchController {
	return &SwitchController{
		strategyConfig: &SwitchStrategyConfig{
			SeamlessThreshold: 0.7,
			StrongThreshold:   0.3,
			HighRiskThreshold: 80.0,
			EnableProgressive: true,
			ProgressiveSteps:  3,
			CooldownPeriod:     5 * time.Minute,
			ForceStrongOnNew:  true,
			ForceStrongOnRisk: true,
			EnableABTesting:   false,
			TestingRatio:      0.1,
		},
		currentStrategy: "adaptive",
		switchHistory:    make([]*SwitchRecord, 0),
	}
}

func newReportGenerator() *ReportGenerator {
	return &ReportGenerator{
		reportConfig: &ReportConfig{
			ReportInterval: 24 * time.Hour,
			MetricsToTrack: []string{
				"total_verifications",
				"seamless_rate",
				"trust_score_avg",
				"risk_score_avg",
				"switch_count",
			},
			RetentionDays:  90,
			EnableRealTime: true,
			AggWindowSize:  1 * time.Hour,
		},
	}
}

func (s *SeamlessV15Service) LearnDeviceFingerprint(fingerprint string, components map[string]string, usage *FingerprintUsage) {
	s.deviceFingerprintLearner.mu.Lock()
	defer s.deviceFingerprintLearner.mu.Unlock()

	model, exists := s.deviceFingerprintLearner.fingerprintModels[fingerprint]
	if !exists {
		model = &FingerprintModel{
			Fingerprint:     fingerprint,
			ComponentHashes: make(map[string]string),
			UsageHistory:    make([]*FingerprintUsage, 0),
			StabilityScore:  0.5,
			ConfidenceLevel: 0.0,
			FirstSeenAt:     time.Now(),
			LastUpdatedAt:   time.Now(),
			FeatureVector:   make([]float64, 0),
		}
		s.deviceFingerprintLearner.fingerprintModels[fingerprint] = model
	}

	for key, value := range components {
		if model.ComponentHashes[key] == "" {
			model.ComponentHashes[key] = value
		}
	}

	if usage != nil {
		model.UsageHistory = append(model.UsageHistory, usage)
		if len(model.UsageHistory) > s.deviceFingerprintLearner.learningConfig.MaxHistorySize {
			model.UsageHistory = model.UsageHistory[len(model.UsageHistory)-s.deviceFingerprintLearner.learningConfig.MaxHistorySize:]
		}

		if usage.Success {
			model.SuccessfulVerifies++
		} else {
			model.FailedVerifies++
		}

		model.AverageRiskScore = model.AverageRiskScore*0.9 + usage.RiskScore*0.1
	}

	model.StabilityScore = s.calculateStabilityScore(model)
	model.ConfidenceLevel = s.calculateConfidenceLevel(model)
	model.LastUpdatedAt = time.Now()
	model.FeatureVector = s.extractFeatureVector(model)
}

func (s *SeamlessV15Service) calculateStabilityScore(model *FingerprintModel) float64 {
	if len(model.UsageHistory) < 2 {
		return 0.5
	}

	ipSet := make(map[string]bool)
	userAgentSet := make(map[string]bool)
	for _, usage := range model.UsageHistory {
		ipSet[usage.IPAddress] = true
		userAgentSet[usage.UserAgent] = true
	}

	uniqueIPs := len(ipSet)
	uniqueAgents := len(userAgentSet)
	total := len(model.UsageHistory)

	ipConsistency := 1.0 - float64(uniqueIPs)/float64(total)
	agentConsistency := 1.0 - float64(uniqueAgents)/float64(total)

	stability := (ipConsistency + agentConsistency) / 2.0

	failRate := float64(model.FailedVerifies) / float64(model.SuccessfulVerifies+model.FailedVerifies)
	stability -= failRate * 0.3

	return math.Max(0, math.Min(1.0, stability))
}

func (s *SeamlessV15Service) calculateConfidenceLevel(model *FingerprintModel) float64 {
	minSamples := s.deviceFingerprintLearner.learningConfig.MinConfidenceSamples
	samples := len(model.UsageHistory)

	if samples < minSamples {
		return float64(samples) / float64(minSamples) * 0.5
	}

	componentCount := len(model.ComponentHashes)
	componentScore := math.Min(1.0, float64(componentCount)/10.0)

	usageScore := math.Min(1.0, float64(samples-minSamples)/float64(minSamples))

	return (componentScore + usageScore + model.StabilityScore) / 3.0
}

func (s *SeamlessV15Service) extractFeatureVector(model *FingerprintModel) []float64 {
	features := make([]float64, 0, 10)

	features = append(features, model.StabilityScore)
	features = append(features, model.ConfidenceLevel)
	features = append(features, float64(len(model.UsageHistory)))
	features = append(features, float64(len(model.ComponentHashes)))
	features = append(features, model.AverageRiskScore)

	if len(model.UsageHistory) > 0 {
		successRate := float64(model.SuccessfulVerifies) / float64(model.SuccessfulVerifies+model.FailedVerifies)
		features = append(features, successRate)
	} else {
		features = append(features, 0.5)
	}

	features = append(features, float64(model.AnomalyCount))
	features = append(features, float64(model.SuccessfulVerifies))
	features = append(features, float64(model.FailedVerifies))

	var timeVariance float64
	if len(model.UsageHistory) > 1 {
		times := make([]float64, len(model.UsageHistory))
		for i, u := range model.UsageHistory {
			times[i] = float64(u.Timestamp.Unix())
		}
		mean := 0.0
		for _, t := range times {
			mean += t
		}
		mean /= float64(len(times))
		for _, t := range times {
			timeVariance += (t - mean) * (t - mean)
		}
		timeVariance /= float64(len(times))
	}
	features = append(features, timeVariance)

	return features
}

func (s *SeamlessV15Service) ModelUserBehavior(userID string, sessionData *SessionPattern) {
	s.behaviorModeler.mu.Lock()
	defer s.behaviorModeler.mu.Unlock()

	model, exists := s.behaviorModeler.userModels[userID]
	if !exists {
		model = &V15UserBehaviorModel{
			UserID:            userID,
			DevicePreferences: make(map[string]*DevicePreference),
			LastUpdatedAt:     time.Now(),
			TypingProfile: &TypingProfile{
				CommonMistakes: make(map[string]int),
			},
			MouseProfile: &MouseProfile{
				ClickPatterns:    make(map[string]int),
				ScrollPreferences: &ScrollPreferences{
					ScrollDirection: make(map[string]int),
				},
			},
			TimePreferences: &TimePreferences{
				PreferredHours:  make(map[int]int),
				PreferredDays:   make(map[int]int),
				PreferredMonths: make(map[int]int),
			},
		}
		s.behaviorModeler.userModels[userID] = model
	}

	if sessionData != nil {
		model.SessionPatterns = append(model.SessionPatterns, sessionData)
		model.TotalSessions++

		if sessionData.Outcome == "success" {
			model.SuccessfulSessions++
		}

		hour := sessionData.StartTime.Hour()
		model.TimePreferences.PreferredHours[hour]++

		day := int(sessionData.StartTime.Weekday())
		model.TimePreferences.PreferredDays[day]++

		month := int(sessionData.StartTime.Month())
		model.TimePreferences.PreferredMonths[month]++

		model.TypingProfile = s.updateTypingProfile(model.TypingProfile, sessionData)
		model.MouseProfile = s.updateMouseProfile(model.MouseProfile, sessionData)
	}

	model.BehavioralEntropy = s.calculateEntropy(model)
	model.HabitStrength = s.calculateHabitStrength(model)
	model.ModelConfidence = s.calculateModelConfidence(model)
	model.LastUpdatedAt = time.Now()
}

func (s *SeamlessV15Service) updateTypingProfile(profile *TypingProfile, session *SessionPattern) *TypingProfile {
	if profile == nil {
		profile = &TypingProfile{CommonMistakes: make(map[string]int)}
	}

	if session.KeyboardEvents > 0 {
		profile.AverageKeystrokeDelay = profile.AverageKeystrokeDelay*0.9 + float64(session.Duration)/float64(session.KeyboardEvents)*0.1
	}

	return profile
}

func (s *SeamlessV15Service) updateMouseProfile(profile *MouseProfile, session *SessionPattern) *MouseProfile {
	if profile == nil {
		profile = &MouseProfile{
			ClickPatterns:    make(map[string]int),
			ScrollPreferences: &ScrollPreferences{ScrollDirection: make(map[string]int)},
		}
	}

	if session.MouseMoves > 0 {
		profile.AverageSpeed = profile.AverageSpeed*0.9 + session.AverageSpeed*0.1
	}

	if session.Clicks > 0 {
		profile.ClickPatterns["click"] += session.Clicks
	}

	if session.ScrollEvents > 0 {
		profile.ScrollPreferences.ScrollDirection["scroll"] += session.ScrollEvents
	}

	return profile
}

func (s *SeamlessV15Service) calculateEntropy(model *V15UserBehaviorModel) float64 {
	if len(model.SessionPatterns) < 2 {
		return 0.5
	}

	hourCounts := make(map[int]int)
	for _, session := range model.SessionPatterns {
		hourCounts[session.StartTime.Hour()]++
	}

	total := len(model.SessionPatterns)
	if total == 0 {
		return 0.5
	}

	entropy := 0.0
	for _, count := range hourCounts {
		p := float64(count) / float64(total)
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	maxEntropy := math.Log2(24.0)
	if maxEntropy == 0 {
		return 0.5
	}
	return entropy / maxEntropy
}

func (s *SeamlessV15Service) calculateHabitStrength(model *V15UserBehaviorModel) float64 {
	if len(model.SessionPatterns) < 2 {
		return float64(len(model.SessionPatterns)) / 5.0 * 0.5
	}

	patternCount := 0
	recentCount := 10
	if len(model.SessionPatterns) < recentCount {
		recentCount = len(model.SessionPatterns)
	}
	recentSessions := model.SessionPatterns[len(model.SessionPatterns)-recentCount:]

	for i := 1; i < len(recentSessions); i++ {
		if math.Abs(float64(recentSessions[i].StartTime.Hour()-recentSessions[i-1].StartTime.Hour())) <= 2 {
			patternCount++
		}
	}

	consistency := float64(patternCount) / float64(len(recentSessions)-1)

	sessionRate := math.Min(1.0, float64(len(model.SessionPatterns))/100.0)

	return (consistency + sessionRate) / 2.0
}

func (s *SeamlessV15Service) calculateModelConfidence(model *V15UserBehaviorModel) float64 {
	sampleScore := math.Min(1.0, float64(len(model.SessionPatterns))/float64(s.behaviorModeler.modelConfig.MinSamplesForModel))
	entropyScore := 1.0 - model.BehavioralEntropy
	strengthScore := model.HabitStrength

	return (sampleScore*0.4 + entropyScore*0.3 + strengthScore*0.3)
}

func (s *SeamlessV15Service) CalculateTrustScore(userID, fingerprint string, behaviorScore, timeScore, locationScore float64) float64 {
	s.trustScoreEngine.mu.Lock()
	defer s.trustScoreEngine.mu.Unlock()

	cacheKey := fmt.Sprintf("%s:%s", userID, fingerprint)
	if cached, exists := s.trustScoreEngine.scoreCache[cacheKey]; exists {
		if time.Now().Before(cached.ExpiresAt) {
			return cached.AdjustedScore
		}
	}

	weights := s.trustScoreEngine.scoringWeights
	params := s.trustScoreEngine.adaptiveParams

	deviceModel := s.deviceFingerprintLearner.getFingerprintModel(fingerprint)
	behaviorModel := s.behaviorModeler.getUserBehaviorModel(userID)

	deviceFactor := 0.5
	if deviceModel != nil {
		deviceFactor = deviceModel.StabilityScore * deviceModel.ConfidenceLevel
	}

	behaviorFactor := behaviorScore
	if behaviorModel != nil {
		behaviorFactor = 1.0 - behaviorModel.BehavioralEntropy*behaviorModel.HabitStrength
	}

	baseScore := params.BaseTrustLevel
	baseScore += weights.DeviceWeight * deviceFactor
	baseScore += weights.BehaviorWeight * behaviorFactor
	baseScore += weights.TimeWeight * timeScore
	baseScore += weights.LocationWeight * locationScore

	if deviceModel != nil {
		historyFactor := float64(deviceModel.SuccessfulVerifies) / float64(deviceModel.SuccessfulVerifies+deviceModel.FailedVerifies+1)
		baseScore += weights.HistoryWeight * historyFactor
	}

	if deviceModel != nil && deviceModel.AnomalyCount > 0 {
		anomalyPenalty := float64(deviceModel.AnomalyCount) * 0.05
		baseScore -= weights.AnomalyWeight * anomalyPenalty
	}

	adjustedScore := math.Max(params.MinTrustThreshold, math.Min(params.MaxTrustThreshold, baseScore))

	cached := &CachedTrustScore{
		UserID:          userID,
		Fingerprint:     fingerprint,
		BaseScore:       baseScore,
		AdjustedScore:   adjustedScore,
		CalculationTime: time.Now(),
		ExpiresAt:       time.Now().Add(5 * time.Minute),
		Factors: map[string]float64{
			"device":   deviceFactor,
			"behavior": behaviorFactor,
			"time":     timeScore,
			"location": locationScore,
		},
	}
	s.trustScoreEngine.scoreCache[cacheKey] = cached

	return adjustedScore
}

func (s *DeviceFingerprintLearner) getFingerprintModel(fingerprint string) *FingerprintModel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fingerprintModels[fingerprint]
}

func (s *BehaviorModeler) getUserBehaviorModel(userID string) *V15UserBehaviorModel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userModels[userID]
}

func (s *SeamlessV15Service) DetermineVerificationType(userID, fingerprint string, baseRiskScore float64) *VerificationDecision {
	s.mu.Lock()
	defer s.mu.Unlock()

	trustScore := s.CalculateTrustScore(userID, fingerprint, 0.5, 0.5, 0.5)
	riskScore := baseRiskScore

	deviceModel := s.deviceFingerprintLearner.getFingerprintModel(fingerprint)
	behaviorModel := s.behaviorModeler.getUserBehaviorModel(userID)

	if deviceModel != nil && deviceModel.ConfidenceLevel > 0.8 {
		riskScore -= deviceModel.StabilityScore * 20
	}

	if behaviorModel != nil && behaviorModel.HabitStrength > 0.7 {
		riskScore -= behaviorModel.HabitStrength * 15
	}

	riskScore = math.Max(0, math.Min(100, riskScore))

	config := s.switchController.strategyConfig
	decision := &VerificationDecision{
		TrustScore:       trustScore,
		RiskScore:        riskScore,
		RecommendedType:  "seamless",
		Confidence:       0.5,
		Reasons:          make([]string, 0),
		ProgressiveLevel: 0,
	}

	if config.ForceStrongOnNew && deviceModel == nil {
		decision.RecommendedType = "strong"
		decision.Reasons = append(decision.Reasons, "新设备检测")
	}

	if config.ForceStrongOnRisk && riskScore >= config.HighRiskThreshold {
		decision.RecommendedType = "strong"
		decision.Reasons = append(decision.Reasons, "高风险评分")
	}

	if riskScore >= config.HighRiskThreshold {
		decision.RecommendedType = "block"
		decision.Reasons = append(decision.Reasons, "阻止请求")
	}

	if decision.RecommendedType == "seamless" {
		if trustScore >= config.SeamlessThreshold && riskScore < 30 {
			decision.Reasons = append(decision.Reasons, "信任评分高且风险低")
		} else if config.EnableProgressive {
			decision.ProgressiveLevel = s.calculateProgressiveLevel(trustScore, riskScore)
			if decision.ProgressiveLevel > 0 {
				decision.RecommendedType = "progressive"
				decision.Reasons = append(decision.Reasons, fmt.Sprintf("渐进式验证级别 %d", decision.ProgressiveLevel))
			}
		}
	}

	decision.Confidence = s.calculateDecisionConfidence(deviceModel, behaviorModel)

	s.recordSwitchDecision(userID, decision)

	return decision
}

func (s *SeamlessV15Service) calculateProgressiveLevel(trustScore, riskScore float64) int {
	config := s.switchController.strategyConfig

	score := (trustScore + (100-riskScore)/100) / 2

	if score >= 0.8 {
		return 0
	} else if score >= 0.6 {
		return 1
	} else if score >= 0.4 {
		return 2
	}

	return config.ProgressiveSteps
}

func (s *SeamlessV15Service) calculateDecisionConfidence(deviceModel *FingerprintModel, behaviorModel *V15UserBehaviorModel) float64 {
	deviceConfidence := 0.0
	if deviceModel != nil {
		deviceConfidence = deviceModel.ConfidenceLevel
	}

	behaviorConfidence := 0.0
	if behaviorModel != nil {
		behaviorConfidence = behaviorModel.ModelConfidence
	}

	totalConfidence := (deviceConfidence + behaviorConfidence) / 2

	if deviceModel == nil && behaviorModel == nil {
		return 0.3
	}

	return totalConfidence
}

func (s *SeamlessV15Service) recordSwitchDecision(userID string, decision *VerificationDecision) {
	record := &SwitchRecord{
		Timestamp:     time.Now(),
		UserID:        userID,
		NewState:      decision.RecommendedType,
		RiskScore:     decision.RiskScore,
		TrustScore:    decision.TrustScore,
	}

	s.switchController.switchHistory = append(s.switchController.switchHistory, record)

	maxHistory := 10000
	if len(s.switchController.switchHistory) > maxHistory {
		s.switchController.switchHistory = s.switchController.switchHistory[len(s.switchController.switchHistory)-maxHistory:]
	}
}

type VerificationDecision struct {
	TrustScore       float64
	RiskScore        float64
	RecommendedType  string
	Confidence       float64
	Reasons          []string
	ProgressiveLevel int
}

func (s *SeamlessV15Service) GenerateReport(periodStart, periodEnd time.Time) *SeamlessReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	report := &SeamlessReport{
		ReportID:    fmt.Sprintf("seamless_report_%d", time.Now().Unix()),
		GeneratedAt: time.Now(),
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Summary:     s.generateSummary(periodStart, periodEnd),
	}

	report.DeviceAnalysis = s.generateDeviceAnalysis()
	report.BehaviorAnalysis = s.generateBehaviorAnalysis()
	report.TrustAnalysis = s.generateTrustAnalysis()
	report.SwitchAnalysis = s.generateSwitchAnalysis(periodStart, periodEnd)
	report.Recommendations = s.generateRecommendations(report)

	return report
}

func (s *SeamlessV15Service) generateSummary(periodStart, periodEnd time.Time) *ReportSummary {
	summary := &ReportSummary{}

	switchRecords := s.getSwitchRecordsInPeriod(periodStart, periodEnd)

	summary.TotalVerifications = len(switchRecords)
	for _, record := range switchRecords {
		switch record.NewState {
		case "seamless":
			summary.SeamlessVerifications++
		case "strong", "progressive":
			summary.StrongVerifications++
		case "block":
			summary.BlockedVerifications++
		}
	}

	if summary.TotalVerifications > 0 {
		summary.SeamlessRate = float64(summary.SeamlessVerifications) / float64(summary.TotalVerifications) * 100
		summary.BlockRate = float64(summary.BlockedVerifications) / float64(summary.TotalVerifications) * 100
	}

	return summary
}

func (s *SeamlessV15Service) getSwitchRecordsInPeriod(start, end time.Time) []*SwitchRecord {
	records := make([]*SwitchRecord, 0)
	for _, record := range s.switchController.switchHistory {
		if record.Timestamp.After(start) && record.Timestamp.Before(end) {
			records = append(records, record)
		}
	}
	return records
}

func (s *SeamlessV15Service) generateDeviceAnalysis() *DeviceAnalysisReport {
	analysis := &DeviceAnalysisReport{
		DevicesByStability: make(map[string]int),
	}

	s.deviceFingerprintLearner.mu.RLock()
	defer s.deviceFingerprintLearner.mu.RUnlock()

	analysis.TotalDevices = len(s.deviceFingerprintLearner.fingerprintModels)

	for _, model := range s.deviceFingerprintLearner.fingerprintModels {
		if model.StabilityScore > 0.8 {
			analysis.TrustedDevices++
			analysis.DevicesByStability["high"]++
		} else if model.StabilityScore > 0.5 {
			analysis.DevicesByStability["medium"]++
		} else {
			analysis.SuspiciousDevices++
			analysis.DevicesByStability["low"]++
		}

		if len(model.UsageHistory) <= 2 {
			analysis.NewDevices++
		}
	}

	if analysis.TotalDevices > 0 {
		analysis.FingerprintAccuracy = float64(analysis.TrustedDevices) / float64(analysis.TotalDevices) * 100
	}

	return analysis
}

func (s *SeamlessV15Service) generateBehaviorAnalysis() *BehaviorAnalysisReport {
	analysis := &BehaviorAnalysisReport{
		CommonPatterns: make([]string, 0),
	}

	s.behaviorModeler.mu.RLock()
	defer s.behaviorModeler.mu.RUnlock()

	analysis.TotalUsers = len(s.behaviorModeler.userModels)

	var totalEntropy, totalStrength float64
	for _, model := range s.behaviorModeler.userModels {
		if model.ModelConfidence > 0.5 {
			analysis.ActiveModels++
		}
		totalEntropy += model.BehavioralEntropy
		totalStrength += model.HabitStrength
	}

	if analysis.TotalUsers > 0 {
		analysis.BehavioralEntropyAvg = totalEntropy / float64(analysis.TotalUsers)
		analysis.HabitStrengthAvg = totalStrength / float64(analysis.TotalUsers)
		analysis.ModelAccuracy = float64(analysis.ActiveModels) / float64(analysis.TotalUsers) * 100
	}

	return analysis
}

func (s *SeamlessV15Service) generateTrustAnalysis() *TrustAnalysisReport {
	analysis := &TrustAnalysisReport{
		TrustDistribution: make(map[string]int),
	}

	s.trustScoreEngine.mu.RLock()
	defer s.trustScoreEngine.mu.RUnlock()

	var totalBase, totalAdjusted, varianceSum float64
	trustScores := make([]float64, 0)

	for _, cached := range s.trustScoreEngine.scoreCache {
		totalBase += cached.BaseScore
		totalAdjusted += cached.AdjustedScore
		trustScores = append(trustScores, cached.AdjustedScore)

		switch {
		case cached.AdjustedScore >= 0.8:
			analysis.TrustDistribution["very_high"]++
			analysis.UsersAboveThreshold++
		case cached.AdjustedScore >= 0.6:
			analysis.TrustDistribution["high"]++
			analysis.UsersAboveThreshold++
		case cached.AdjustedScore >= 0.4:
			analysis.TrustDistribution["medium"]++
		case cached.AdjustedScore >= 0.2:
			analysis.TrustDistribution["low"]++
			analysis.UsersBelowThreshold++
		default:
			analysis.TrustDistribution["very_low"]++
			analysis.UsersBelowThreshold++
		}
	}

	count := len(trustScores)
	if count > 0 {
		analysis.AverageBaseTrust = totalBase / float64(count)
		analysis.AverageAdjustedTrust = totalAdjusted / float64(count)

		mean := analysis.AverageAdjustedTrust
		for _, score := range trustScores {
			varianceSum += (score - mean) * (score - mean)
		}
		analysis.TrustScoreVariance = varianceSum / float64(count)
	}

	return analysis
}

func (s *SeamlessV15Service) generateSwitchAnalysis(periodStart, periodEnd time.Time) *SwitchAnalysisReport {
	analysis := &SwitchAnalysisReport{
		SwitchReasons: make(map[string]int),
	}

	records := s.getSwitchRecordsInPeriod(periodStart, periodEnd)
	analysis.TotalSwitches = len(records) - 1

	var prevState string
	for _, record := range records {
		if prevState != "" && prevState != record.NewState {
			if prevState == "seamless" {
				analysis.SeamlessToStrong++
			} else {
				analysis.StrongToSeamless++
			}
		}

		if record.Reason != "" {
			analysis.SwitchReasons[record.Reason]++
		}

		prevState = record.NewState
	}

	if analysis.TotalSwitches > 0 {
		analysis.SwitchSuccessRate = float64(analysis.SeamlessToStrong+analysis.StrongToSeamless) / float64(analysis.TotalSwitches) * 100
	}

	return analysis
}

func (s *SeamlessV15Service) generateRecommendations(report *SeamlessReport) []string {
	recommendations := make([]string, 0)

	if report.Summary.SeamlessRate < 50 {
		recommendations = append(recommendations, "建议降低无缝验证阈值以提升用户体验")
	}

	if report.DeviceAnalysis.TotalDevices > 0 && float64(report.DeviceAnalysis.NewDevices) > float64(report.DeviceAnalysis.TotalDevices)*0.3 {
		recommendations = append(recommendations, "新设备比例较高，考虑优化首次验证流程")
	}

	if report.Summary.TotalVerifications > 0 && float64(report.SwitchAnalysis.TotalSwitches) > float64(report.Summary.TotalVerifications)*0.2 {
		recommendations = append(recommendations, "验证类型切换频繁，建议调整阈值参数")
	}

	if report.BehaviorAnalysis.ModelAccuracy < 70 {
		recommendations = append(recommendations, "行为模型准确率偏低，需要更多训练数据")
	}

	if report.TrustAnalysis.UsersBelowThreshold > report.TrustAnalysis.UsersAboveThreshold {
		recommendations = append(recommendations, "低信任用户较多，建议加强用户引导")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "系统运行良好，继续监控关键指标")
	}

	return recommendations
}

func (s *SeamlessV15Service) UpdateBehaviorData(userID string, behaviorData *BehaviorUpdateData) error {
	if behaviorData == nil {
		return fmt.Errorf("行为数据为空")
	}

	sessionPattern := &SessionPattern{
		SessionID:       behaviorData.SessionID,
		StartTime:       behaviorData.Timestamp,
		Duration:        time.Duration(behaviorData.Duration) * time.Millisecond,
		MouseMoves:      behaviorData.MouseMoves,
		KeyboardEvents:  behaviorData.KeyboardEvents,
		Clicks:          behaviorData.Clicks,
		ScrollEvents:    behaviorData.ScrollEvents,
		AverageSpeed:    behaviorData.AverageSpeed,
		Outcome:         "success",
	}

	s.ModelUserBehavior(userID, sessionPattern)

	if behaviorData.Fingerprint != "" {
		usage := &FingerprintUsage{
			Timestamp:   behaviorData.Timestamp,
			IPAddress:   behaviorData.IPAddress,
			UserAgent:   behaviorData.UserAgent,
			RiskScore:   behaviorData.RiskScore,
			Success:     behaviorData.Success,
			BehaviorHash: behaviorData.BehaviorHash,
		}
		s.LearnDeviceFingerprint(behaviorData.Fingerprint, behaviorData.FingerprintComponents, usage)
	}

	return nil
}

func (s *SeamlessV15Service) GetTrustScore(userID, fingerprint string) *TrustScoreResult {
	trustScore := s.CalculateTrustScore(userID, fingerprint, 0.5, 0.5, 0.5)

	deviceModel := s.deviceFingerprintLearner.getFingerprintModel(fingerprint)
	behaviorModel := s.behaviorModeler.getUserBehaviorModel(userID)

	result := &TrustScoreResult{
		UserID:        userID,
		Fingerprint:   fingerprint,
		TrustScore:    trustScore,
		RiskScore:     1.0 - trustScore,
		DeviceStable:  false,
		BehaviorKnown: false,
	}

	if deviceModel != nil {
		result.DeviceStable = deviceModel.StabilityScore > 0.7
		result.DeviceConfidence = deviceModel.ConfidenceLevel
		result.DeviceUsageCount = len(deviceModel.UsageHistory)
	}

	if behaviorModel != nil {
		result.BehaviorKnown = behaviorModel.HabitStrength > 0.5
		result.BehaviorConfidence = behaviorModel.ModelConfidence
		result.SessionCount = behaviorModel.TotalSessions
	}

	return result
}

func (s *SeamlessV15Service) PerformSeamlessVerification(userID, fingerprint string, riskScore float64) *VerificationResult {
	decision := s.DetermineVerificationType(userID, fingerprint, riskScore)

	result := &VerificationResult{
		Success:            decision.RecommendedType != "block",
		VerificationType:   decision.RecommendedType,
		TrustScore:         decision.TrustScore,
		RiskScore:          decision.RiskScore,
		Confidence:         decision.Confidence,
		Reasons:            decision.Reasons,
		ProgressiveLevel:   decision.ProgressiveLevel,
		Token:              "",
	}

	if result.Success {
		result.Token = fmt.Sprintf("st_%d_%s", time.Now().UnixNano(), userID)
	}

	return result
}

type BehaviorUpdateData struct {
	SessionID            string
	Timestamp            time.Time
	Duration             int64
	MouseMoves           int
	KeyboardEvents       int
	Clicks               int
	ScrollEvents         int
	AverageSpeed         float64
	RiskScore            float64
	Success              bool
	BehaviorHash         string
	Fingerprint          string
	FingerprintComponents map[string]string
	IPAddress            string
	UserAgent            string
}

type TrustScoreResult struct {
	UserID             string
	Fingerprint        string
	TrustScore         float64
	RiskScore          float64
	DeviceStable       bool
	DeviceConfidence   float64
	DeviceUsageCount   int
	BehaviorKnown      bool
	BehaviorConfidence float64
	SessionCount       int
}

type VerificationResult struct {
	Success            bool
	VerificationType   string
	TrustScore         float64
	RiskScore          float64
	Confidence         float64
	Reasons            []string
	ProgressiveLevel   int
	Token              string
}

func (s *SeamlessV15Service) GetGlobalStats() map[string]interface{} {
	stats := make(map[string]interface{})

	s.deviceFingerprintLearner.mu.RLock()
	deviceCount := len(s.deviceFingerprintLearner.fingerprintModels)
	var totalStability, totalConfidence float64
	for _, model := range s.deviceFingerprintLearner.fingerprintModels {
		totalStability += model.StabilityScore
		totalConfidence += model.ConfidenceLevel
	}
	s.deviceFingerprintLearner.mu.RUnlock()

	s.behaviorModeler.mu.RLock()
	userCount := len(s.behaviorModeler.userModels)
	s.behaviorModeler.mu.RUnlock()

	s.trustScoreEngine.mu.RLock()
	trustCount := len(s.trustScoreEngine.scoreCache)
	s.trustScoreEngine.mu.RUnlock()

	s.switchController.mu.RLock()
	switchCount := len(s.switchController.switchHistory)
	s.switchController.mu.RUnlock()

	stats["total_devices"] = deviceCount
	stats["total_users"] = userCount
	stats["total_cached_trust_scores"] = trustCount
	stats["total_switch_records"] = switchCount

	if deviceCount > 0 {
		stats["average_device_stability"] = totalStability / float64(deviceCount)
		stats["average_device_confidence"] = totalConfidence / float64(deviceCount)
	}

	return stats
}

func (s *SeamlessV15Service) ExportModelData() ([]byte, error) {
	data := struct {
		DeviceFingerprints map[string]*FingerprintModel `json:"device_fingerprints"`
		UserBehaviors      map[string]*V15UserBehaviorModel `json:"user_behaviors"`
		SwitchHistory      []*SwitchRecord `json:"switch_history"`
		ExportedAt         time.Time `json:"exported_at"`
	}{
		DeviceFingerprints: s.deviceFingerprintLearner.fingerprintModels,
		UserBehaviors:      s.behaviorModeler.userModels,
		SwitchHistory:      s.switchController.switchHistory,
		ExportedAt:         time.Now(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func (s *SeamlessV15Service) ImportModelData(jsonData []byte) error {
	data := struct {
		DeviceFingerprints map[string]*FingerprintModel `json:"device_fingerprints"`
		UserBehaviors      map[string]*V15UserBehaviorModel `json:"user_behaviors"`
		SwitchHistory      []*SwitchRecord `json:"switch_history"`
	}{}

	if err := json.Unmarshal(jsonData, &data); err != nil {
		return err
	}

	s.deviceFingerprintLearner.mu.Lock()
	for k, v := range data.DeviceFingerprints {
		s.deviceFingerprintLearner.fingerprintModels[k] = v
	}
	s.deviceFingerprintLearner.mu.Unlock()

	s.behaviorModeler.mu.Lock()
	for k, v := range data.UserBehaviors {
		s.behaviorModeler.userModels[k] = v
	}
	s.behaviorModeler.mu.Unlock()

	s.switchController.mu.Lock()
	s.switchController.switchHistory = append(s.switchController.switchHistory, data.SwitchHistory...)
	s.switchController.mu.Unlock()

	return nil
}

func (s *SeamlessV15Service) CleanupOldData(retentionDays int) int {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	cleaned := 0

	s.deviceFingerprintLearner.mu.Lock()
	for fingerprint, model := range s.deviceFingerprintLearner.fingerprintModels {
		if model.LastUpdatedAt.Before(cutoff) {
			delete(s.deviceFingerprintLearner.fingerprintModels, fingerprint)
			cleaned++
		}
	}
	s.deviceFingerprintLearner.mu.Unlock()

	s.behaviorModeler.mu.Lock()
	for userID, model := range s.behaviorModeler.userModels {
		if model.LastUpdatedAt.Before(cutoff) {
			delete(s.behaviorModeler.userModels, userID)
			cleaned++
		}
	}
	s.behaviorModeler.mu.Unlock()

	s.switchController.mu.Lock()
	filtered := make([]*SwitchRecord, 0)
	for _, record := range s.switchController.switchHistory {
		if record.Timestamp.After(cutoff) {
			filtered = append(filtered, record)
		} else {
			cleaned++
		}
	}
	s.switchController.switchHistory = filtered
	s.switchController.mu.Unlock()

	s.trustScoreEngine.mu.Lock()
	for key, cached := range s.trustScoreEngine.scoreCache {
		if cached.ExpiresAt.Before(time.Now()) {
			delete(s.trustScoreEngine.scoreCache, key)
			cleaned++
		}
	}
	s.trustScoreEngine.mu.Unlock()

	return cleaned
}
