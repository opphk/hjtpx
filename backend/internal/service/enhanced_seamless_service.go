package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type EnhancedSeamlessService struct {
	fingerprintEngine *EnhancedFingerprintEngine
	trustScorer       *MultiDimensionalTrustScorer
	continuousLearner *OnlineContinuousLearner
	disturbSuppressor *IntelligentDisturbSuppressor
	behaviorPredictor *BehaviorPredictor
	config            *EnhancedSeamlessConfig
	mu                sync.RWMutex
}

type EnhancedSeamlessConfig struct {
	MinConfidenceForSkip     float64
	MaxFalsePositiveRate    float64
	LearningWindowDays       int
	TrustScoreDecayRate      float64
	AnomalyThreshold         float64
	EnableOnlineLearning     bool
	EnableAdaptiveWeights    bool
	QuietHoursEnabled        bool
	QuietHoursStart          int
	QuietHoursEnd            int
	MaxChallengesPerDay      int
	MaxChallengesPerWeek     int
}

type EnhancedFingerprintComponents struct {
	UserAgent          string            `json:"user_agent"`
	ScreenInfo         string            `json:"screen_info"`
	ColorDepth         int               `json:"color_depth"`
	Timezone           string            `json:"timezone"`
	Language           string            `json:"language"`
	Platform           string            `json:"platform"`
	CanvasHash         string            `json:"canvas_hash"`
	WebGLVendor        string            `json:"webgl_vendor"`
	WebGLRenderer      string            `json:"webgl_renderer"`
	AudioFingerprint   string            `json:"audio_fingerprint"`
	FontList           []string          `json:"font_list"`
	PluginList         []string          `json:"plugin_list"`
	DoNotTrack         string            `json:"do_not_track"`
	TouchSupport       map[string]interface{} `json:"touch_support"`
	DeviceMemory       string            `json:"device_memory"`
	HardwareConcurrency int              `json:"hardware_concurrency"`
	ConnectionType     string            `json:"connection_type"`
	WebRTCSupport       bool              `json:"webrtc_support"`
	IndexedDBSupport   bool              `json:"indexed_db_support"`
	LocalStorageSupport bool             `json:"local_storage_support"`
	SessionStorageSupport bool           `json:"session_storage_support"`
	CookiesEnabled      bool             `json:"cookies_enabled"`
	AdBlockerDetected  bool              `json:"ad_blocker_detected"`
	AutomationDetected  bool             `json:"automation_detected"`
	BatteryStatus      *BatteryInfo       `json:"battery_status,omitempty"`
	MediaDevices       []string          `json:"media_devices"`
	GPUInfo            *GPUInfo           `json:"gpu_info,omitempty"`
}

type BatteryInfo struct {
	Level           float64 `json:"level"`
	Charging        bool    `json:"charging"`
	ChargingTime    float64 `json:"charging_time"`
	DischargingTime float64 `json:"discharging_time"`
}

type GPUInfo struct {
	VendorID    string `json:"vendor_id"`
	DeviceID   string `json:"device_id"`
	DriverVersion string `json:"driver_version"`
	Architecture string `json:"architecture"`
}

type EnhancedFingerprintEngine struct {
	historicalHashes map[string]*FingerprintStability
	componentWeights map[string]float64
	stabilityScores map[string]float64
	mu              sync.RWMutex
}

type FingerprintStability struct {
	Fingerprint    string
	AppearanceCount int
	FirstSeen      time.Time
	LastSeen       time.Time
	ConsistencyScore float64
	AssociatedUsers map[string]int
	VariationHistory []string
}

type MultiDimensionalTrustScorer struct {
	dimensionWeights map[string]float64
	dimensionScores map[string]map[string]*DimensionScore
	adaptiveWeights *AdaptiveWeightEngine
	mu              sync.RWMutex
}

type DimensionScore struct {
	Dimension     string
	RawScore      float64
	WeightedScore float64
	Confidence    float64
	Factors       []string
	LastUpdated   time.Time
}

type AdaptiveWeightEngine struct {
	performanceMetrics map[string]*DimensionPerformance
	updateCount       int
	decayFactor       float64
	mu                sync.RWMutex
}

type DimensionPerformance struct {
	Dimension         string
	TruePositiveRate  float64
	FalsePositiveRate float64
	Accuracy          float64
	RecentAccuracy    []float64
	Weight            float64
}

type OnlineContinuousLearner struct {
	userModels map[string]*UserBehaviorModel
	globalModel *GlobalBehaviorModel
	config     *LearnerConfig
	eventLog   []LearningEvent
	mu         sync.RWMutex
}

type UserBehaviorModel struct {
	UserID          string
	LoginTimePattern *TimePattern
	LocationPattern  *LocationPattern
	DevicePattern    *DevicePattern
	BehaviorPattern  *DetailedBehaviorPattern
	TrustEvolution  []float64
	LastUpdate      time.Time
	ModelVersion    int
	Confidence      float64
}

type TimePattern struct {
	PreferredHours   map[int]int
	PreferredDays    map[int]int
	TypicalInterval  time.Duration
	VarianceHours    float64
	RecentHours      []int
}

type LocationPattern struct {
	KnownLocations  []string
	LocationSequence []string
	TravelSpeed     float64
	AnomalyLocations []string
	LastLocations   []string
}

type DevicePattern struct {
	KnownDevices    map[string]*SeamlessDeviceInfo
	DeviceAffinity  map[string]float64
	TypicalDeviceCount int
	NewDeviceAlert  bool
}

type SeamlessDeviceInfo struct {
	Fingerprint    string
	FirstSeen       time.Time
	LastSeen        time.Time
	UseCount        int
	SuccessRate     float64
	AvgResponseTime time.Duration
}

type DetailedBehaviorPattern struct {
	AvgResponseTime    time.Duration
	ResponseTimeVariance float64
	MouseMovementAvg   float64
	KeyboardTypingSpeed float64
	ClickFrequency      float64
	ScrollPattern      string
	TabSwitches        int
	ErrorRate          float64
	SuccessRate        float64
	RecentMetrics       *RecentBehaviorMetrics
}

type RecentBehaviorMetrics struct {
	Last7DaysSuccessRate float64
	Last30DaysSuccessRate float64
	TotalAttempts        int
	FailedAttempts        int
	AvgSessionDuration    time.Duration
	ChallengeCount       int
	SkipCount            int
}

type GlobalBehaviorModel struct {
	TotalUsers        int
	AvgSuccessRate    float64
	AnomalyPatterns   []string
	TrustedDeviceRate float64
	LastUpdated       time.Time
}

type LearningEvent struct {
	Timestamp    time.Time
	UserID       string
	EventType    string
	Success      bool
	Features     map[string]interface{}
	NewModel     *UserBehaviorModel
}

type LearnerConfig struct {
	LearningRate       float64
	DecayRate          float64
	MinSamples         int
	MaxModelAge        time.Duration
	WindowSize         int
	ConfidenceThreshold float64
}

type IntelligentDisturbSuppressor struct {
	userPreferences map[string]*UserDisturbanceProfile
	globalStats     *GlobalDisturbanceStats
	riskDecisions   *RiskDecisionCache
	config          *DisturbSuppressConfig
	mu              sync.RWMutex
}

type UserDisturbanceProfile struct {
	UserID             string
	MinDisturbLevel    int
	PreferredHours     []int
	AvoidDays          []int
	AlwaysVerifyNewDevice bool
	TrustDurationDays  int
	MaxDailyChallenges int
	MaxWeeklyChallenges int
	CustomRules        []DisturbanceRule
	EffectiveTrustLevel float64
	LastUpdated        time.Time
}

type DisturbanceRule struct {
	Condition   string
	SkipChallenge bool
	RiskThreshold float64
	TimeRange    *TimeRange
	DeviceType   string
}

type TimeRange struct {
	Start int
	End   int
	Days  []int
}

type GlobalDisturbanceStats struct {
	TotalChallenges    int
	SkippedChallenges  int
	FalseNegatives     int
	UserSatisfaction   float64
	AvgChallengeRate   float64
	RecentRates        []float64
	LastUpdated        time.Time
}

type RiskDecisionCache struct {
	cache map[string]*CachedDecision
	mu    sync.RWMutex
}

type CachedDecision struct {
	Hash        string
	Decision    string
	RiskScore   float64
	Timestamp   time.Time
	TTL         time.Duration
}

type DisturbSuppressConfig struct {
	EnableAdaptiveSuppression bool
	MaxSkipRate               float64
	MinSatisfactionScore      float64
	QuietHoursEnabled         bool
	ProgressiveChallenge      bool
	AutoTuneThresholds         bool
}

type BehaviorPredictor struct {
	models        map[string]*PredictionModel
	globalStats   *PredictionGlobalStats
	featureWeights map[string]float64
	mu            sync.RWMutex
}

type PredictionModel struct {
	UserID          string
	Features        []string
	Weights         map[string]float64
	Bias            float64
	PredictionAccuracy float64
	LastUpdated     time.Time
	TrainingSamples int
}

type PredictionGlobalStats struct {
	TotalPredictions    int
	CorrectPredictions  int
	AvgConfidence       float64
	FeatureImportance   map[string]float64
}

func NewEnhancedSeamlessService() *EnhancedSeamlessService {
	return &EnhancedSeamlessService{
		fingerprintEngine: newEnhancedFingerprintEngine(),
		trustScorer:       newMultiDimensionalTrustScorer(),
		continuousLearner: newOnlineContinuousLearner(),
		disturbSuppressor: newIntelligentDisturbSuppressor(),
		behaviorPredictor: newBehaviorPredictor(),
		config: &EnhancedSeamlessConfig{
			MinConfidenceForSkip:     0.7,
			MaxFalsePositiveRate:     0.05,
			LearningWindowDays:       30,
			TrustScoreDecayRate:      0.01,
			AnomalyThreshold:         30.0,
			EnableOnlineLearning:     true,
			EnableAdaptiveWeights:    true,
			QuietHoursStart:          22,
			QuietHoursEnd:            8,
			MaxChallengesPerDay:      5,
			MaxChallengesPerWeek:     20,
		},
	}
}

func newEnhancedFingerprintEngine() *EnhancedFingerprintEngine {
	return &EnhancedFingerprintEngine{
		historicalHashes: make(map[string]*FingerprintStability),
		componentWeights: map[string]float64{
			"canvas":         0.20,
			"webgl":          0.18,
			"audio":          0.12,
			"fonts":          0.10,
			"screen":         0.08,
			"timezone":       0.08,
			"language":       0.06,
			"platform":       0.06,
			"plugins":        0.05,
			"touch_support":  0.03,
			"hardware":       0.04,
		},
		stabilityScores: make(map[string]float64),
	}
}

func newMultiDimensionalTrustScorer() *MultiDimensionalTrustScorer {
	return &MultiDimensionalTrustScorer{
		dimensionWeights: map[string]float64{
			"device_history":    0.25,
			"behavior_pattern": 0.25,
			"location":          0.20,
			"time_pattern":      0.15,
			"network":           0.10,
			"application":       0.05,
		},
		dimensionScores: make(map[string]map[string]*DimensionScore),
		adaptiveWeights: &AdaptiveWeightEngine{
			performanceMetrics: make(map[string]*DimensionPerformance),
			updateCount:        0,
			decayFactor:         0.95,
		},
	}
}

func newOnlineContinuousLearner() *OnlineContinuousLearner {
	return &OnlineContinuousLearner{
		userModels:  make(map[string]*UserBehaviorModel),
		globalModel: &GlobalBehaviorModel{},
		config: &LearnerConfig{
			LearningRate:         0.1,
			DecayRate:            0.05,
			MinSamples:           10,
			MaxModelAge:          30 * 24 * time.Hour,
			WindowSize:           100,
			ConfidenceThreshold: 0.7,
		},
		eventLog: make([]LearningEvent, 0, 1000),
	}
}

func newIntelligentDisturbSuppressor() *IntelligentDisturbSuppressor {
	return &IntelligentDisturbSuppressor{
		userPreferences: make(map[string]*UserDisturbanceProfile),
		globalStats: &GlobalDisturbanceStats{
			RecentRates: make([]float64, 0, 100),
		},
		riskDecisions: &RiskDecisionCache{
			cache: make(map[string]*CachedDecision),
		},
		config: &DisturbSuppressConfig{
			EnableAdaptiveSuppression: true,
			MaxSkipRate:                0.95,
			MinSatisfactionScore:        0.8,
			QuietHoursEnabled:          true,
			ProgressiveChallenge:       true,
			AutoTuneThresholds:         true,
		},
	}
}

func newBehaviorPredictor() *BehaviorPredictor {
	return &BehaviorPredictor{
		models:        make(map[string]*PredictionModel),
		globalStats:   &PredictionGlobalStats{},
		featureWeights: map[string]float64{
			"response_time":     0.3,
			"mouse_movement":    0.25,
			"keyboard_pattern":  0.2,
			"click_location":    0.15,
			"session_duration":  0.1,
		},
	}
}

func (s *EnhancedSeamlessService) OptimizeVerification(
	userID string,
	deviceFingerprint string,
	behaviorData []models.BehaviorData,
	environmentData map[string]interface{},
	previousRiskScore float64,
) (*EnhancedSeamlessResult, error) {
	
	result := &EnhancedSeamlessResult{
		OriginalRiskScore: previousRiskScore,
		FinalRiskScore:    previousRiskScore,
		ShouldChallenge:   true,
		TrustLevel:        0.5,
		OptimizationApplied: []string{},
		Confidence:        0.5,
	}

	fpComponents := s.fingerprintEngine.parseFingerprintComponents(deviceFingerprint)
	fpStability := s.fingerprintEngine.getFingerprintStability(deviceFingerprint)
	
	fingerprintScore := s.fingerprintEngine.calculateFingerprintScore(fpComponents, fpStability)
	result.OptimizationApplied = append(result.OptimizationApplied, fmt.Sprintf("fingerprint_score:%.2f", fingerprintScore))
	
	trustDimensions := s.trustScorer.calculateAllDimensions(userID, deviceFingerprint, behaviorData, environmentData)
	result.DimensionScores = trustDimensions
	
	trustLevel := s.trustScorer.calculateWeightedTrust(trustDimensions)
	result.TrustLevel = trustLevel
	result.OptimizationApplied = append(result.OptimizationApplied, fmt.Sprintf("trust_level:%.2f", trustLevel))
	
	if s.config.EnableOnlineLearning {
		userModel := s.continuousLearner.getOrUpdateUserModel(userID, behaviorData, trustLevel)
		predictedBehavior := s.continuousLearner.predictUserBehavior(userModel)
		result.PredictedBehavior = predictedBehavior
		
		behaviorConsistency := s.calculateBehaviorConsistency(behaviorData, predictedBehavior)
		if behaviorConsistency > 0.8 {
			result.FinalRiskScore = math.Max(0, result.FinalRiskScore-10*behaviorConsistency)
			result.OptimizationApplied = append(result.OptimizationApplied, "behavior_consistency_bonus")
		} else if behaviorConsistency < 0.5 {
			result.FinalRiskScore = math.Min(100, result.FinalRiskScore+15*(1-behaviorConsistency))
			result.OptimizationApplied = append(result.OptimizationApplied, "behavior_inconsistency_penalty")
		}
	}
	
	if fpStability != nil && fpStability.ConsistencyScore > 0.9 {
		stableBonus := fpStability.ConsistencyScore * 15
		result.FinalRiskScore = math.Max(0, result.FinalRiskScore-stableBonus)
		result.OptimizationApplied = append(result.OptimizationApplied, "fingerprint_stability_bonus")
	}
	
	suppressionDecision := s.disturbSuppressor.shouldSuppressChallenge(userID, deviceFingerprint, result.FinalRiskScore, behaviorData)
	if suppressionDecision.ShouldSuppress {
		result.ShouldChallenge = false
		result.SkipReason = suppressionDecision.Reason
		result.OptimizationApplied = append(result.OptimizationApplied, "challenge_suppressed")
	}
	
	if s.isInQuietHours() {
		result.ShouldChallenge = false
		result.SkipReason = "安静时段跳过验证"
		result.OptimizationApplied = append(result.OptimizationApplied, "quiet_hours_skip")
	}
	
	if trustLevel > s.config.MinConfidenceForSkip && result.FinalRiskScore < s.config.AnomalyThreshold {
		result.ShouldChallenge = false
		result.SkipReason = "高信任度用户"
		result.OptimizationApplied = append(result.OptimizationApplied, "high_trust_skip")
	}
	
	result.Confidence = s.calculateResultConfidence(fpStability, trustDimensions)
	
	return result, nil
}

func (s *EnhancedFingerprintEngine) parseFingerprintComponents(fingerprint string) *EnhancedFingerprintComponents {
	components := &EnhancedFingerprintComponents{}
	
	if len(fingerprint) >= 64 {
		components.CanvasHash = fingerprint[0:64]
	}
	
	return components
}

func (s *EnhancedFingerprintEngine) getFingerprintStability(fingerprint string) *FingerprintStability {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.historicalHashes[fingerprint]
}

func (s *EnhancedFingerprintEngine) calculateFingerprintScore(components *EnhancedFingerprintComponents, stability *FingerprintStability) float64 {
	score := 0.0
	
	if components == nil {
		return 0.0
	}
	
	if components.CanvasHash != "" && components.CanvasHash != "error" {
		score += s.componentWeights["canvas"] * 100
	}
	
	if components.WebGLRenderer != "" {
		score += s.componentWeights["webgl"] * 100
	}
	
	if components.AudioFingerprint != "" && components.AudioFingerprint != "error" {
		score += s.componentWeights["audio"] * 100
	}
	
	if len(components.FontList) > 5 {
		score += s.componentWeights["fonts"] * 100 * math.Min(1.0, float64(len(components.FontList))/10.0)
	}
	
	if components.ScreenInfo != "" && components.ScreenInfo != "0x0" {
		score += s.componentWeights["screen"] * 100
	}
	
	if components.Timezone != "" {
		score += s.componentWeights["timezone"] * 100
	}
	
	if components.Language != "" {
		score += s.componentWeights["language"] * 100
	}
	
	if components.Platform != "" {
		score += s.componentWeights["platform"] * 100
	}
	
	if len(components.PluginList) > 0 {
		score += s.componentWeights["plugins"] * 100 * math.Min(1.0, float64(len(components.PluginList))/5.0)
	}
	
	if stability != nil {
		score = score * 0.7 + score * 0.3 * stability.ConsistencyScore
	}
	
	return math.Min(100, score)
}

func (s *EnhancedFingerprintEngine) updateFingerprintStability(fingerprint, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	
	if stability, exists := s.historicalHashes[fingerprint]; exists {
		stability.AppearanceCount++
		stability.LastSeen = now
		stability.ConsistencyScore = math.Min(1.0, float64(stability.AppearanceCount)/10.0)
		
		if stability.AssociatedUsers == nil {
			stability.AssociatedUsers = make(map[string]int)
		}
		stability.AssociatedUsers[userID]++
	} else {
		s.historicalHashes[fingerprint] = &FingerprintStability{
			Fingerprint:      fingerprint,
			AppearanceCount:  1,
			FirstSeen:        now,
			LastSeen:         now,
			ConsistencyScore: 0.1,
			AssociatedUsers: map[string]int{userID: 1},
		}
	}
	
	s.recalculateStabilityScores()
}

func (s *EnhancedFingerprintEngine) recalculateStabilityScores() {
	for fp, stability := range s.historicalHashes {
		age := time.Since(stability.FirstSeen)
		usageRate := float64(stability.AppearanceCount) / (age.Hours() / 24 + 1)
		
		uniqueUsers := len(stability.AssociatedUsers)
		userDiversityFactor := 1.0
		if uniqueUsers > 1 {
			userDiversityFactor = 1.0 / math.Log(float64(uniqueUsers)+1)
		}
		
		consistencyScore := math.Min(1.0, stability.ConsistencyScore*0.6+usageRate*0.3+userDiversityFactor*0.1)
		s.stabilityScores[fp] = consistencyScore
	}
}

func (s *MultiDimensionalTrustScorer) calculateAllDimensions(userID, deviceFingerprint string, behaviorData []models.BehaviorData, environmentData map[string]interface{}) map[string]*DimensionScore {
	dimensions := make(map[string]*DimensionScore)
	
	dimensions["device_history"] = s.calculateDeviceHistoryScore(userID, deviceFingerprint)
	dimensions["behavior_pattern"] = s.calculateBehaviorPatternScore(userID, behaviorData)
	dimensions["location"] = s.calculateLocationScore(userID, environmentData)
	dimensions["time_pattern"] = s.calculateTimePatternScore(userID)
	dimensions["network"] = s.calculateNetworkScore(environmentData)
	dimensions["application"] = s.calculateApplicationScore(userID, environmentData)
	
	if s.adaptiveWeights != nil {
		s.updateAdaptiveWeights(dimensions)
	}
	
	return dimensions
}

func (s *MultiDimensionalTrustScorer) calculateDeviceHistoryScore(userID, deviceFingerprint string) *DimensionScore {
	score := &DimensionScore{
		Dimension:   "device_history",
		RawScore:    50.0,
		Confidence: 0.5,
		Factors:    []string{},
		LastUpdated: time.Now(),
	}
	
	score.RawScore = 60.0
	score.Factors = append(score.Factors, "设备历史评估")
	score.Confidence = 0.7
	
	return score
}

func (s *MultiDimensionalTrustScorer) calculateBehaviorPatternScore(userID string, behaviorData []models.BehaviorData) *DimensionScore {
	score := &DimensionScore{
		Dimension:   "behavior_pattern",
		RawScore:    50.0,
		Confidence: 0.5,
		Factors:    []string{},
		LastUpdated: time.Now(),
	}
	
	if len(behaviorData) > 5 {
		score.RawScore = 70.0
		score.Factors = append(score.Factors, "丰富行为数据")
		score.Confidence = 0.8
	} else if len(behaviorData) > 0 {
		score.RawScore = 60.0
		score.Factors = append(score.Factors, "有限行为数据")
		score.Confidence = 0.6
	}
	
	return score
}

func (s *MultiDimensionalTrustScorer) calculateLocationScore(userID string, environmentData map[string]interface{}) *DimensionScore {
	score := &DimensionScore{
		Dimension:   "location",
		RawScore:    50.0,
		Confidence: 0.5,
		Factors:    []string{},
		LastUpdated: time.Now(),
	}
	
	if ip, ok := environmentData["ip_address"].(string); ok && ip != "" {
		score.RawScore = 65.0
		score.Factors = append(score.Factors, "IP地理位置可用")
		score.Confidence = 0.7
	}
	
	return score
}

func (s *MultiDimensionalTrustScorer) calculateTimePatternScore(userID string) *DimensionScore {
	score := &DimensionScore{
		Dimension:   "time_pattern",
		RawScore:    50.0,
		Confidence: 0.5,
		Factors:    []string{},
		LastUpdated: time.Now(),
	}
	
	currentHour := time.Now().Hour()
	if currentHour >= 9 && currentHour <= 22 {
		score.RawScore = 70.0
		score.Factors = append(score.Factors, "正常时间段")
		score.Confidence = 0.6
	} else {
		score.RawScore = 60.0
		score.Factors = append(score.Factors, "非标准时间段")
		score.Confidence = 0.5
	}
	
	return score
}

func (s *MultiDimensionalTrustScorer) calculateNetworkScore(environmentData map[string]interface{}) *DimensionScore {
	score := &DimensionScore{
		Dimension:   "network",
		RawScore:    50.0,
		Confidence: 0.5,
		Factors:    []string{},
		LastUpdated: time.Now(),
	}
	
	if proxy, ok := environmentData["proxy_detected"].(bool); ok && proxy {
		score.RawScore = 20.0
		score.Factors = append(score.Factors, "检测到代理")
		score.Confidence = 0.9
	} else {
		score.RawScore = 75.0
		score.Factors = append(score.Factors, "未检测到代理")
		score.Confidence = 0.7
	}
	
	return score
}

func (s *MultiDimensionalTrustScorer) calculateApplicationScore(userID string, environmentData map[string]interface{}) *DimensionScore {
	score := &DimensionScore{
		Dimension:   "application",
		RawScore:    50.0,
		Confidence: 0.5,
		Factors:    []string{},
		LastUpdated: time.Now(),
	}
	
	score.RawScore = 60.0
	score.Confidence = 0.6
	
	return score
}

func (s *MultiDimensionalTrustScorer) calculateWeightedTrust(dimensions map[string]*DimensionScore) float64 {
	totalWeight := 0.0
	weightedSum := 0.0
	
	for dimName, dimScore := range dimensions {
		weight := s.dimensionWeights[dimName]
		if weight == 0 {
			weight = 0.1
		}
		
		weightedSum += dimScore.RawScore * weight * dimScore.Confidence
		totalWeight += weight * dimScore.Confidence
	}
	
	if totalWeight == 0 {
		return 0.5
	}
	
	return weightedSum / totalWeight / 100.0
}

func (s *MultiDimensionalTrustScorer) updateAdaptiveWeights(dimensions map[string]*DimensionScore) {
	s.adaptiveWeights.mu.Lock()
	defer s.adaptiveWeights.mu.Unlock()
	
	s.adaptiveWeights.updateCount++
	
	for dimName, dimScore := range dimensions {
		perf, exists := s.adaptiveWeights.performanceMetrics[dimName]
		if !exists {
			perf = &DimensionPerformance{
				Dimension:      dimName,
				Weight:         s.dimensionWeights[dimName],
				RecentAccuracy: make([]float64, 0, 10),
			}
			s.adaptiveWeights.performanceMetrics[dimName] = perf
		}
		
		estimatedAccuracy := dimScore.Confidence * 0.8
		perf.RecentAccuracy = append(perf.RecentAccuracy, estimatedAccuracy)
		
		if len(perf.RecentAccuracy) > 10 {
			perf.RecentAccuracy = perf.RecentAccuracy[len(perf.RecentAccuracy)-10:]
		}
		
		sum := 0.0
		for _, acc := range perf.RecentAccuracy {
			sum += acc
		}
		perf.Accuracy = sum / float64(len(perf.RecentAccuracy))
	}
	
	s.rebalanceWeights()
}

func (s *MultiDimensionalTrustScorer) rebalanceWeights() {
	totalAccuracy := 0.0
	dimensionAccuracies := make(map[string]float64)
	
	for dimName, perf := range s.adaptiveWeights.performanceMetrics {
		dimensionAccuracies[dimName] = perf.Accuracy
		totalAccuracy += perf.Accuracy
	}
	
	if totalAccuracy == 0 {
		return
	}
	
	newWeights := make(map[string]float64)
	for dimName, acc := range dimensionAccuracies {
		baseWeight := s.dimensionWeights[dimName]
		adaptiveWeight := acc / totalAccuracy
		newWeights[dimName] = baseWeight*0.7 + adaptiveWeight*0.3
	}
	
	sum := 0.0
	for _, w := range newWeights {
		sum += w
	}
	for dimName := range newWeights {
		newWeights[dimName] /= sum
	}
	
	for dimName, w := range newWeights {
		s.dimensionWeights[dimName] = s.adaptiveWeights.decayFactor*s.dimensionWeights[dimName] + 
			(1-s.adaptiveWeights.decayFactor)*w
	}
}

func (s *OnlineContinuousLearner) getOrUpdateUserModel(userID string, behaviorData []models.BehaviorData, trustLevel float64) *UserBehaviorModel {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	model, exists := s.userModels[userID]
	if !exists {
		model = &UserBehaviorModel{
			UserID:          userID,
			LoginTimePattern: &TimePattern{},
			LocationPattern:  &LocationPattern{},
			DevicePattern:    &DevicePattern{},
			BehaviorPattern:  &DetailedBehaviorPattern{},
			TrustEvolution:   make([]float64, 0),
			ModelVersion:     1,
			Confidence:       0.1,
		}
		s.userModels[userID] = model
	}
	
	s.updateTimePattern(model, time.Now())
	s.updateBehaviorPattern(model, behaviorData)
	
	model.TrustEvolution = append(model.TrustEvolution, trustLevel)
	if len(model.TrustEvolution) > 30 {
		model.TrustEvolution = model.TrustEvolution[len(model.TrustEvolution)-30:]
	}
	
	model.Confidence = math.Min(1.0, model.Confidence+0.05)
	model.LastUpdate = time.Now()
	
	return model
}

func (s *OnlineContinuousLearner) updateTimePattern(model *UserBehaviorModel, t time.Time) {
	if model.LoginTimePattern.PreferredHours == nil {
		model.LoginTimePattern.PreferredHours = make(map[int]int)
	}
	if model.LoginTimePattern.PreferredDays == nil {
		model.LoginTimePattern.PreferredDays = make(map[int]int)
	}
	
	hour := t.Hour()
	day := int(t.Weekday())
	
	model.LoginTimePattern.PreferredHours[hour]++
	model.LoginTimePattern.PreferredDays[day]++
	
	model.LoginTimePattern.RecentHours = append(model.LoginTimePattern.RecentHours, hour)
	if len(model.LoginTimePattern.RecentHours) > 20 {
		model.LoginTimePattern.RecentHours = model.LoginTimePattern.RecentHours[len(model.LoginTimePattern.RecentHours)-20:]
	}
	
	var sum float64
	for _, h := range model.LoginTimePattern.RecentHours {
		sum += float64(h)
	}
	avg := sum / float64(len(model.LoginTimePattern.RecentHours))
	
	var varianceSum float64
	for _, h := range model.LoginTimePattern.RecentHours {
		varianceSum += math.Pow(float64(h)-avg, 2)
	}
	model.LoginTimePattern.VarianceHours = math.Sqrt(varianceSum / float64(len(model.LoginTimePattern.RecentHours)))
}

func (s *OnlineContinuousLearner) updateBehaviorPattern(model *UserBehaviorModel, behaviorData []models.BehaviorData) {
	if model.BehaviorPattern.RecentMetrics == nil {
		model.BehaviorPattern.RecentMetrics = &RecentBehaviorMetrics{}
	}
	
	if model.BehaviorPattern.RecentMetrics.TotalAttempts == 0 {
		model.BehaviorPattern.RecentMetrics.TotalAttempts = 1
	} else {
		model.BehaviorPattern.RecentMetrics.TotalAttempts++
	}
	
	if len(behaviorData) > 0 {
		model.BehaviorPattern.RecentMetrics.TotalAttempts += len(behaviorData)
		
		var totalTime float64
		for _, data := range behaviorData {
			totalTime += float64(len(data.Data))
		}
		avgTime := totalTime / float64(len(behaviorData))
		
		if model.BehaviorPattern.AvgResponseTime == 0 {
			model.BehaviorPattern.AvgResponseTime = time.Duration(avgTime) * time.Millisecond
		} else {
			model.BehaviorPattern.AvgResponseTime = time.Duration(float64(model.BehaviorPattern.AvgResponseTime) * 0.9 + avgTime * 0.1)
		}
	}
}

func (s *OnlineContinuousLearner) predictUserBehavior(model *UserBehaviorModel) *PredictedBehavior {
	prediction := &PredictedBehavior{
		ExpectedLoginHour:  make(map[int]float64),
		ExpectedLocation:   "unknown",
		ExpectedDeviceType: "typical",
		Confidence:         model.Confidence,
	}
	
	if model.LoginTimePattern != nil && len(model.LoginTimePattern.PreferredHours) > 0 {
		maxCount := 0
		preferredHour := 10
		
		for hour, count := range model.LoginTimePattern.PreferredHours {
			if count > maxCount {
				maxCount = count
				preferredHour = hour
			}
			prediction.ExpectedLoginHour[hour] = float64(count)
		}
		
		totalCount := 0
		for _, count := range model.LoginTimePattern.PreferredHours {
			totalCount += count
		}
		for hour := range prediction.ExpectedLoginHour {
			prediction.ExpectedLoginHour[hour] /= float64(totalCount)
		}
		
		prediction.ExpectedLoginHour[preferredHour] = 1.0
	}
	
	return prediction
}

type PredictedBehavior struct {
	ExpectedLoginHour   map[int]float64 `json:"expected_login_hour"`
	ExpectedLocation    string          `json:"expected_location"`
	ExpectedDeviceType  string          `json:"expected_device_type"`
	Confidence          float64         `json:"confidence"`
}

func (s *EnhancedSeamlessService) calculateBehaviorConsistency(behaviorData []models.BehaviorData, prediction *PredictedBehavior) float64 {
	if prediction == nil || len(prediction.ExpectedLoginHour) == 0 {
		return 0.5
	}
	
	consistencyScore := 0.5
	
	if len(behaviorData) > 5 {
		consistencyScore += 0.2
	}
	
	currentHour := time.Now().Hour()
	if prob, exists := prediction.ExpectedLoginHour[currentHour]; exists && prob > 0.3 {
		consistencyScore += 0.3 * prob
	}
	
	return math.Min(1.0, consistencyScore)
}

func (s *IntelligentDisturbSuppressor) shouldSuppressChallenge(userID, deviceFingerprint string, riskScore float64, behaviorData []models.BehaviorData) *SuppressionDecision {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	decision := &SuppressionDecision{
		ShouldSuppress: false,
		Reason:         "",
		Confidence:     0.5,
	}
	
	profile := s.getUserProfile(userID)
	
	if profile != nil && profile.MinDisturbLevel >= 3 && riskScore < 25 {
		decision.ShouldSuppress = true
		decision.Reason = "用户偏好低打扰"
		decision.Confidence = 0.9
		return decision
	}
	
	if s.globalStats != nil {
		skipRate := float64(s.globalStats.SkippedChallenges) / float64(math.Max(1, float64(s.globalStats.TotalChallenges)))
		if skipRate > s.config.MaxSkipRate {
			decision.ShouldSuppress = false
			decision.Reason = "全局跳过率过高"
			decision.Confidence = 0.7
			return decision
		}
	}
	
	if len(behaviorData) > 10 {
		decision.ShouldSuppress = true
		decision.Reason = "行为数据充足，可信度高"
		decision.Confidence = 0.8
		return decision
	}
	
	return decision
}

func (s *IntelligentDisturbSuppressor) getUserProfile(userID string) *UserDisturbanceProfile {
	profile, exists := s.userPreferences[userID]
	if !exists {
		profile = &UserDisturbanceProfile{
			UserID:              userID,
			MinDisturbLevel:     1,
			MaxDailyChallenges:  5,
			MaxWeeklyChallenges:  20,
			EffectiveTrustLevel: 0.5,
			LastUpdated:         time.Now(),
		}
		s.userPreferences[userID] = profile
	}
	return profile
}

func (s *EnhancedSeamlessService) isInQuietHours() bool {
	if !s.config.QuietHoursEnabled {
		return false
	}
	
	hour := time.Now().Hour()
	start := s.config.QuietHoursStart
	end := s.config.QuietHoursEnd
	
	if start > end {
		return hour >= start || hour < end
	}
	return hour >= start && hour < end
}

func (s *EnhancedSeamlessService) calculateResultConfidence(stability *FingerprintStability, dimensions map[string]*DimensionScore) float64 {
	confidence := 0.5
	
	if stability != nil {
		confidence += 0.2 * stability.ConsistencyScore
	}
	
	var dimConfidenceSum float64
	var dimCount float64
	for _, dim := range dimensions {
		dimConfidenceSum += dim.Confidence
		dimCount++
	}
	if dimCount > 0 {
		confidence += 0.3 * (dimConfidenceSum / dimCount)
	}
	
	return math.Min(1.0, confidence)
}

func (s *EnhancedSeamlessService) UpdateLearningFromResult(userID, deviceFingerprint string, verificationSuccess bool, responseTime time.Duration) {
	s.continuousLearner.mu.Lock()
	defer s.continuousLearner.mu.Unlock()
	
	event := LearningEvent{
		Timestamp: time.Now(),
		UserID:    userID,
		EventType: "verification_result",
		Success:   verificationSuccess,
		Features:  map[string]interface{}{},
	}
	
	s.continuousLearner.eventLog = append(s.continuousLearner.eventLog, event)
	
	if len(s.continuousLearner.eventLog) > 1000 {
		s.continuousLearner.eventLog = s.continuousLearner.eventLog[len(s.continuousLearner.eventLog)-1000:]
	}
	
	s.fingerprintEngine.updateFingerprintStability(deviceFingerprint, userID)
	
	s.updateTrustScorerPerformance(userID, deviceFingerprint, verificationSuccess)
}

func (s *EnhancedSeamlessService) updateTrustScorerPerformance(userID, deviceFingerprint string, success bool) {
	s.trustScorer.mu.Lock()
	defer s.trustScorer.mu.Unlock()
	
	for _, perf := range s.trustScorer.adaptiveWeights.performanceMetrics {
		if success {
			perf.TruePositiveRate = perf.TruePositiveRate*0.9 + 0.1
		} else {
			perf.FalsePositiveRate = perf.FalsePositiveRate*0.9 + 0.1
		}
		perf.Accuracy = (perf.TruePositiveRate + (1 - perf.FalsePositiveRate)) / 2
	}
}

type EnhancedSeamlessResult struct {
	OriginalRiskScore    float64                `json:"original_risk_score"`
	FinalRiskScore       float64                `json:"final_risk_score"`
	ShouldChallenge      bool                   `json:"should_challenge"`
	SkipReason           string                 `json:"skip_reason,omitempty"`
	TrustLevel           float64                `json:"trust_level"`
	DimensionScores      map[string]*DimensionScore `json:"dimension_scores"`
	OptimizationApplied  []string               `json:"optimization_applied"`
	Confidence           float64                `json:"confidence"`
	PredictedBehavior    *PredictedBehavior     `json:"predicted_behavior,omitempty"`
}

type SuppressionDecision struct {
	ShouldSuppress bool    `json:"should_suppress"`
	Reason         string  `json:"reason,omitempty"`
	Confidence     float64 `json:"confidence"`
}

func (s *EnhancedSeamlessService) GenerateEnhancedFingerprint(components *EnhancedFingerprintComponents) string {
	fpData, _ := json.Marshal(components)
	hash := sha256.Sum256(fpData)
	return hex.EncodeToString(hash[:])
}

func (s *EnhancedSeamlessService) ValidateFingerprintStability(fingerprint string, minAppearances int, maxAge time.Duration) (bool, float64) {
	stability := s.fingerprintEngine.getFingerprintStability(fingerprint)
	if stability == nil {
		return false, 0.0
	}
	
	if stability.AppearanceCount < minAppearances {
		return false, 0.0
	}
	
	if time.Since(stability.FirstSeen) > maxAge {
		return false, stability.ConsistencyScore
	}
	
	return true, stability.ConsistencyScore
}

func (s *EnhancedSeamlessService) GetTrustScoreBreakdown(userID, deviceFingerprint string) map[string]interface{} {
	breakdown := make(map[string]interface{})
	
	breakdown["device_trust"] = s.trustScorer.dimensionWeights["device_history"]
	breakdown["behavior_trust"] = s.trustScorer.dimensionWeights["behavior_pattern"]
	breakdown["location_trust"] = s.trustScorer.dimensionWeights["location"]
	breakdown["time_trust"] = s.trustScorer.dimensionWeights["time_pattern"]
	breakdown["network_trust"] = s.trustScorer.dimensionWeights["network"]
	breakdown["application_trust"] = s.trustScorer.dimensionWeights["application"]
	
	breakdown["fingerprint_stability"] = s.fingerprintEngine.stabilityScores[deviceFingerprint]
	
	if model, exists := s.continuousLearner.userModels[userID]; exists {
		breakdown["model_confidence"] = model.Confidence
		if model.BehaviorPattern.RecentMetrics != nil {
			breakdown["total_attempts"] = model.BehaviorPattern.RecentMetrics.TotalAttempts
		}
		breakdown["success_rate"] = model.BehaviorPattern.SuccessRate
	}
	
	return breakdown
}

func (s *EnhancedSeamlessService) SetUserPreference(userID string, pref *UserDisturbanceProfile) {
	s.disturbSuppressor.mu.Lock()
	defer s.disturbSuppressor.mu.Unlock()
	
	pref.LastUpdated = time.Now()
	s.disturbSuppressor.userPreferences[userID] = pref
}

func (s *EnhancedSeamlessService) GetUserPreference(userID string) *UserDisturbanceProfile {
	return s.disturbSuppressor.getUserProfile(userID)
}

func (s *EnhancedSeamlessService) GetGlobalDisturbanceStats() *GlobalDisturbanceStats {
	return s.disturbSuppressor.globalStats
}

func (s *EnhancedSeamlessService) RecordChallengeResult(userID string, wasSkipped, wasSuccessful bool) {
	s.disturbSuppressor.mu.Lock()
	defer s.disturbSuppressor.mu.Unlock()
	
	s.disturbSuppressor.globalStats.TotalChallenges++
	if wasSkipped {
		s.disturbSuppressor.globalStats.SkippedChallenges++
	}
	
	skipRate := float64(s.disturbSuppressor.globalStats.SkippedChallenges) / float64(s.disturbSuppressor.globalStats.TotalChallenges)
	s.disturbSuppressor.globalStats.RecentRates = append(s.disturbSuppressor.globalStats.RecentRates, skipRate)
	
	if len(s.disturbSuppressor.globalStats.RecentRates) > 100 {
		s.disturbSuppressor.globalStats.RecentRates = s.disturbSuppressor.globalStats.RecentRates[len(s.disturbSuppressor.globalStats.RecentRates)-100:]
	}
	
	s.disturbSuppressor.globalStats.AvgChallengeRate = skipRate
	s.disturbSuppressor.globalStats.LastUpdated = time.Now()
	
	if !wasSkipped && wasSuccessful {
		s.disturbSuppressor.globalStats.UserSatisfaction = math.Min(1.0, s.disturbSuppressor.globalStats.UserSatisfaction+0.01)
	} else if !wasSkipped && !wasSuccessful {
		s.disturbSuppressor.globalStats.UserSatisfaction = math.Max(0, s.disturbSuppressor.globalStats.UserSatisfaction-0.02)
	}
}

func (s *EnhancedSeamlessService) OptimizeDisturbanceThresholds() map[string]float64 {
	s.disturbSuppressor.mu.Lock()
	defer s.disturbSuppressor.mu.Unlock()
	
	thresholds := make(map[string]float64)
	
	if len(s.disturbSuppressor.globalStats.RecentRates) < 10 {
		thresholds["low_risk"] = 20.0
		thresholds["medium_risk"] = 50.0
		thresholds["high_risk"] = 80.0
		return thresholds
	}
	
	recentAvg := 0.0
	for _, rate := range s.disturbSuppressor.globalStats.RecentRates[len(s.disturbSuppressor.globalStats.RecentRates)-10:] {
		recentAvg += rate
	}
	recentAvg /= 10.0
	
	targetSkipRate := s.disturbSuppressor.config.MaxSkipRate * 0.95
	
	if recentAvg > targetSkipRate {
		adjustment := (recentAvg - targetSkipRate) * 10
		thresholds["low_risk"] = math.Max(10, 20-adjustment)
		thresholds["medium_risk"] = math.Max(40, 50-adjustment)
		thresholds["high_risk"] = math.Max(70, 80-adjustment)
	} else if recentAvg < targetSkipRate*0.8 {
		adjustment := (targetSkipRate*0.8 - recentAvg) * 10
		thresholds["low_risk"] = math.Min(30, 20+adjustment)
		thresholds["medium_risk"] = math.Min(60, 50+adjustment)
		thresholds["high_risk"] = math.Min(90, 80+adjustment)
	} else {
		thresholds["low_risk"] = 20.0
		thresholds["medium_risk"] = 50.0
		thresholds["high_risk"] = 80.0
	}
	
	return thresholds
}

func (s *EnhancedSeamlessService) UpdateFingerprintFromComponents(fingerprint string, components *EnhancedFingerprintComponents) {
	s.fingerprintEngine.mu.Lock()
	defer s.fingerprintEngine.mu.Unlock()
	
	if len(fingerprint) >= 64 {
		s.fingerprintEngine.historicalHashes[fingerprint] = &FingerprintStability{
			Fingerprint:       fingerprint,
			AppearanceCount:   1,
			FirstSeen:         time.Now(),
			LastSeen:          time.Now(),
			ConsistencyScore:  0.1,
			AssociatedUsers:   make(map[string]int),
		}
	}
}

func (s *EnhancedSeamlessService) CleanupOldData(maxAge time.Duration) int {
	s.fingerprintEngine.mu.Lock()
	defer s.fingerprintEngine.mu.Unlock()
	
	s.continuousLearner.mu.Lock()
	defer s.continuousLearner.mu.Unlock()
	
	removed := 0
	now := time.Now()
	
	for fp, stability := range s.fingerprintEngine.historicalHashes {
		if now.Sub(stability.LastSeen) > maxAge {
			delete(s.fingerprintEngine.historicalHashes, fp)
			delete(s.fingerprintEngine.stabilityScores, fp)
			removed++
		}
	}
	
	for userID, model := range s.continuousLearner.userModels {
		if now.Sub(model.LastUpdate) > maxAge {
			delete(s.continuousLearner.userModels, userID)
			removed++
		}
	}
	
	s.continuousLearner.eventLog = s.continuousLearner.filterOldEvents(s.continuousLearner.eventLog, maxAge)
	
	return removed
}

func (s *OnlineContinuousLearner) filterOldEvents(events []LearningEvent, maxAge time.Duration) []LearningEvent {
	cutoff := time.Now().Add(-maxAge)
	filtered := make([]LearningEvent, 0, len(events))

	for _, event := range events {
		if event.Timestamp.After(cutoff) {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

func NewEnhancedFingerprintEngineForTest() *EnhancedFingerprintEngine {
	return newEnhancedFingerprintEngine()
}

func (s *EnhancedFingerprintEngine) CalculateFingerprintScore(components *EnhancedFingerprintComponents, stability *FingerprintStability) float64 {
	return s.calculateFingerprintScore(components, stability)
}

func (s *EnhancedFingerprintEngine) GetFingerprintStability(fingerprint string) *FingerprintStability {
	return s.getFingerprintStability(fingerprint)
}

func (s *EnhancedFingerprintEngine) UpdateFingerprintStability(fingerprint, userID string) {
	s.updateFingerprintStability(fingerprint, userID)
}

func NewMultiDimensionalTrustScorerForTest() *MultiDimensionalTrustScorer {
	return newMultiDimensionalTrustScorer()
}

func (s *MultiDimensionalTrustScorer) CalculateAllDimensions(userID, deviceFingerprint string, behaviorData []models.BehaviorData, environmentData map[string]interface{}) map[string]*DimensionScore {
	return s.calculateAllDimensions(userID, deviceFingerprint, behaviorData, environmentData)
}

func (s *MultiDimensionalTrustScorer) CalculateWeightedTrust(dimensions map[string]*DimensionScore) float64 {
	return s.calculateWeightedTrust(dimensions)
}

func NewOnlineContinuousLearnerForTest() *OnlineContinuousLearner {
	return newOnlineContinuousLearner()
}

func (s *OnlineContinuousLearner) GetOrUpdateUserModel(userID string, behaviorData []models.BehaviorData, trustLevel float64) *UserBehaviorModel {
	return s.getOrUpdateUserModel(userID, behaviorData, trustLevel)
}

func (s *OnlineContinuousLearner) PredictUserBehavior(model *UserBehaviorModel) *PredictedBehavior {
	return s.predictUserBehavior(model)
}

func NewIntelligentDisturbSuppressorForTest() *IntelligentDisturbSuppressor {
	return newIntelligentDisturbSuppressor()
}

func (s *IntelligentDisturbSuppressor) ShouldSuppressChallenge(userID, deviceFingerprint string, riskScore float64, behaviorData []models.BehaviorData) *SuppressionDecision {
	return s.shouldSuppressChallenge(userID, deviceFingerprint, riskScore, behaviorData)
}

func NewEnhancedSeamlessServiceForTest() *EnhancedSeamlessService {
	return NewEnhancedSeamlessService()
}

func (s *EnhancedSeamlessService) UpdateFingerprintStabilityForTest(fingerprint, userID string) {
	s.fingerprintEngine.updateFingerprintStability(fingerprint, userID)
}

func NewSeamlessIntegrationServiceForTest() *SeamlessIntegrationService {
	return NewSeamlessIntegrationService()
}

func (s *SeamlessIntegrationService) CalculateBehaviorConsistencyForTest(behaviorData []models.BehaviorData, prediction *PredictedBehavior) float64 {
	return s.enhancedService.calculateBehaviorConsistency(behaviorData, prediction)
}
