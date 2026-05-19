package service

import (
	"encoding/json"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type InvisibleCaptchaService struct {
	fingerprintEngine    *InvisibleFingerprintEngine
	confidenceAssessor  *BehavioralConfidenceAssessor
	behaviorAnalyzer     *HistoricalBehaviorAnalyzer
	trustCalculator      *CompositeTrustCalculator
	config              *InvisibleCaptchaConfig
	mu                  sync.RWMutex
}

type InvisibleCaptchaConfig struct {
	EnableFingerprintOptimization   bool
	EnableConfidenceAssessment     bool
	EnableBehaviorAnalysis         bool
	EnableTrustCalculation          bool
	MinConfidenceThreshold         float64
	MaxRiskScore                   float64
	LearningWindowHours            int
	TrustDecayRate                 float64
	HistoryRetentionDays           int
	EnableAdaptiveScoring          bool
}

type InvisibleFingerprintEngine struct {
	fingerprintCache    map[string]*DeviceFingerprintRecord
	componentWeights    map[string]float64
	stabilityTracker    *FingerprintStabilityTracker
	uniquenessCalculator *UniquenessCalculator
	mu                  sync.RWMutex
}

type DeviceFingerprintRecord struct {
	Fingerprint        string                     `json:"fingerprint"`
	Components        *FingerprintComponents      `json:"components"`
	FirstSeen         time.Time                  `json:"first_seen"`
	LastSeen          time.Time                  `json:"last_seen"`
	AppearanceCount   int                        `json:"appearance_count"`
	ConsistencyScore  float64                    `json:"consistency_score"`
	UniquenessScore   float64                    `json:"uniqueness_score"`
	TrustScore        float64                    `json:"trust_score"`
	AssociatedUsers   map[string]int             `json:"associated_users"`
	ComponentHistory  map[string][]string         `json:"component_history"`
	VersionHistory    []string                   `json:"version_history"`
}

type FingerprintComponents struct {
	CanvasHash        string   `json:"canvas_hash"`
	WebGLHash         string   `json:"webgl_hash"`
	AudioHash         string   `json:"audio_hash"`
	FontHash          string   `json:"font_hash"`
	ScreenHash        string   `json:"screen_hash"`
	TimezoneHash      string   `json:"timezone_hash"`
	LanguageHash      string   `json:"language_hash"`
	PlatformHash      string   `json:"platform_hash"`
	HardwareHash      string   `json:"hardware_hash"`
	PluginHash        string   `json:"plugin_hash"`
	TouchSupport      bool     `json:"touch_support"`
	ColorDepth        int      `json:"color_depth"`
	PixelRatio        float64  `json:"pixel_ratio"`
	HardwareConcurrency int    `json:"hardware_concurrency"`
	DeviceMemory      float64  `json:"device_memory"`
	DoNotTrack        string   `json:"do_not_track"`
	WebGLVendor       string   `json:"webgl_vendor"`
	WebGLRenderer     string   `json:"webgl_renderer"`
	MediaDevices      []string `json:"media_devices"`
}

type FingerprintStabilityTracker struct {
	historicalRecords map[string][]*StabilitySnapshot
	snapshotInterval  time.Duration
	maxSnapshots     int
	mu               sync.RWMutex
}

type StabilitySnapshot struct {
	Timestamp        time.Time `json:"timestamp"`
	ConsistencyScore float64  `json:"consistency_score"`
	ComponentMatch   float64  `json:"component_match"`
	AnomalyDetected  bool     `json:"anomaly_detected"`
}

type UniquenessCalculator struct {
	knownSignatures map[string]int
	signatureCount  int
	collisionMap    map[string][]string
	mu              sync.RWMutex
}

type BehavioralConfidenceAssessor struct {
	confidenceModels map[string]*ConfidenceModel
	scoringRules     []ConfidenceRule
	temporalWeights  map[string]float64
	behaviorPatterns *PatternDatabase
	mu               sync.RWMutex
}

type ConfidenceModel struct {
	UserID             string                   `json:"user_id"`
	DeviceFingerprint  string                   `json:"device_fingerprint"`
	ConfidenceHistory  []ConfidenceRecord       `json:"confidence_history"`
	CurrentConfidence  float64                  `json:"current_confidence"`
	BaseScore         float64                  `json:"base_score"`
	ModifierFactors   map[string]float64       `json:"modifier_factors"`
	LastUpdated       time.Time                `json:"last_updated"`
	SampleCount       int                      `json:"sample_count"`
}

type ConfidenceRecord struct {
	Timestamp       time.Time              `json:"timestamp"`
	ConfidenceScore  float64               `json:"confidence_score"`
	Factors         []ConfidenceFactor     `json:"factors"`
	VerificationResult bool                 `json:"verification_result"`
}

type ConfidenceFactor struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
	Score  float64 `json:"score"`
}

type ConfidenceRule struct {
	Name            string
	Condition       func(*ConfidenceContext) bool
	ScoreModifier   float64
	Weight          float64
	Priority        int
}

type ConfidenceContext struct {
	UserID           string
	DeviceFingerprint string
	BehaviorData     []models.BehaviorData
	TimeOfDay        int
	DayOfWeek        int
	IsKnownDevice    bool
	IsKnownLocation  bool
	HistoricalSuccessRate float64
	RecentFailureCount    int
	RequestFrequency      float64
}

type PatternDatabase struct {
	loginPatterns     map[string]*LoginPattern
	behaviorPatterns map[string]*InvisibleCaptchaBehaviorPattern
	locationPatterns map[string]*InvisibleCaptchaLocationPattern
	mu               sync.RWMutex
}

type LoginPattern struct {
	UserID           string          `json:"user_id"`
	PreferredHours   map[int]int     `json:"preferred_hours"`
	PreferredDays    map[int]int     `json:"preferred_days"`
	TypicalInterval  time.Duration   `json:"typical_interval"`
	AvgSessionDuration time.Duration `json:"avg_session_duration"`
	RecentHours      []int           `json:"recent_hours"`
	RecentDays       []int           `json:"recent_days"`
}

type InvisibleCaptchaBehaviorPattern struct {
	UserID           string          `json:"user_id"`
	AvgResponseTime  time.Duration   `json:"avg_response_time"`
	ResponseVariance float64         `json:"response_variance"`
	ClickFrequency   float64         `json:"click_frequency"`
	ErrorRate        float64         `json:"error_rate"`
	SuccessRate      float64         `json:"success_rate"`
	MouseSpeedAvg    float64         `json:"mouse_speed_avg"`
	KeyboardSpeedAvg float64         `json:"keyboard_speed_avg"`
}

type InvisibleCaptchaLocationPattern struct {
	UserID          string     `json:"user_id"`
	KnownLocations  []string   `json:"known_locations"`
	LocationHistory []string   `json:"location_history"`
	TravelSpeed     float64    `json:"travel_speed"`
	AnomalyLocations []string  `json:"anomaly_locations"`
	LastLocation    string     `json:"last_location"`
}

type HistoricalBehaviorAnalyzer struct {
	userHistories   map[string]*UserBehaviorHistory
	sessionAnalyzer *SessionAnalyzer
	anomalyDetector *BehaviorAnomalyDetector
	trendAnalyzer   *InvisibleCaptchaTrendAnalyzer
	mu              sync.RWMutex
}

type UserBehaviorHistory struct {
	UserID              string                 `json:"user_id"`
	VerificationHistory []*VerificationRecord  `json:"verification_history"`
	DeviceHistory       []*DeviceRecord        `json:"device_history"`
	LocationHistory     []*LocationRecord      `json:"location_history"`
	SessionMetrics      *InvisibleCaptchaSessionMetrics `json:"session_metrics"`
	RiskEvolution       []float64              `json:"risk_evolution"`
	LastUpdate          time.Time              `json:"last_update"`
}

type VerificationRecord struct {
	Timestamp           time.Time              `json:"timestamp"`
	Fingerprint         string                 `json:"fingerprint"`
	RiskScore           float64               `json:"risk_score"`
	Confidence          float64               `json:"confidence"`
	TrustScore          float64               `json:"trust_score"`
	VerificationSuccess bool                  `json:"verification_success"`
	ChallengeIssued     bool                  `json:"challenge_issued"`
	ResponseTimeMs      int64                 `json:"response_time_ms"`
	EnvironmentData     map[string]interface{} `json:"environment_data"`
}

type DeviceRecord struct {
	Fingerprint    string    `json:"fingerprint"`
	FirstSeen     time.Time  `json:"first_seen"`
	LastSeen      time.Time  `json:"last_seen"`
	UseCount      int        `json:"use_count"`
	SuccessRate   float64    `json:"success_rate"`
	AvgRiskScore  float64    `json:"avg_risk_score"`
	IsTrusted     bool       `json:"is_trusted"`
}

type LocationRecord struct {
	Location     string    `json:"location"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	VisitCount   int       `json:"visit_count"`
	IsAnomaly    bool      `json:"is_anomaly"`
}

type InvisibleCaptchaSessionMetrics struct {
	TotalSessions      int                      `json:"total_sessions"`
	AvgSessionDuration time.Duration            `json:"avg_session_duration"`
	SessionIntervals   []time.Duration          `json:"session_intervals"`
	ActiveHours        map[int]int             `json:"active_hours"`
}

type SessionAnalyzer struct {
	sessionCache map[string]*InvisibleCaptchaSessionData
	maxCacheSize int
	mu           sync.RWMutex
}

type InvisibleCaptchaSessionData struct {
	SessionID    string          `json:"session_id"`
	UserID       string          `json:"user_id"`
	StartTime    time.Time       `json:"start_time"`
	EndTime      *time.Time      `json:"end_time,omitempty"`
	BehaviorData []models.BehaviorData `json:"behavior_data"`
	RequestCount int             `json:"request_count"`
	SuccessCount int             `json:"success_count"`
}

type BehaviorAnomalyDetector struct {
	anomalyRules   []AnomalyRule
	baselineModels map[string]*InvisibleCaptchaAnomalyBaseline
	mu             sync.RWMutex
}

type AnomalyRule struct {
	Name        string
	Type        string
	Threshold   float64
	Severity    float64
	Weight      float64
	Enabled     bool
}

type InvisibleCaptchaAnomalyBaseline struct {
	UserID      string
	MetricName  string
	Mean        float64
	StdDev      float64
	SampleCount int
}

type InvisibleCaptchaTrendAnalyzer struct {
	trendData      map[string]*InvisibleCaptchaTrendData
	analysisWindow time.Duration
	mu            sync.RWMutex
}

type InvisibleCaptchaTrendData struct {
	MetricName   string      `json:"metric_name"`
	DataPoints   []InvisibleCaptchaDataPoint `json:"data_points"`
	Trend        string      `json:"trend"`
	ChangeRate   float64     `json:"change_rate"`
	Seasonality  bool        `json:"seasonality"`
}

type InvisibleCaptchaDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type CompositeTrustCalculator struct {
	trustComponents map[string]*TrustComponent
	weightManager   *TrustWeightManager
	riskEngine     *RiskScoreEngine
	mu             sync.RWMutex
}

type TrustComponent struct {
	Name           string  `json:"name"`
	BaseWeight     float64 `json:"base_weight"`
	CurrentWeight  float64 `json:"current_weight"`
	Score          float64 `json:"score"`
	Confidence     float64 `json:"confidence"`
	LastUpdated    time.Time `json:"last_updated"`
}

type TrustWeightManager struct {
	adaptiveWeights map[string]float64
	performanceLog  map[string][]float64
	updateCount     int
	mu              sync.RWMutex
}

type RiskScoreEngine struct {
	riskFactors   map[string]RiskFactor
	globalRules   []GlobalRiskRule
	historicalRisk map[string][]float64
	mu            sync.RWMutex
}

type RiskFactor struct {
	Name     string  `json:"name"`
	Weight   float64 `json:"weight"`
	Score    float64 `json:"score"`
	Evidence []string `json:"evidence"`
}

type GlobalRiskRule struct {
	Name       string
	Conditions []RiskCondition
	ScoreMod   float64
	Enabled    bool
}

type RiskCondition struct {
	Field    string
	Operator string
	Value    interface{}
}

func NewInvisibleCaptchaService() *InvisibleCaptchaService {
	return &InvisibleCaptchaService{
		fingerprintEngine:   newInvisibleFingerprintEngine(),
		confidenceAssessor: newBehavioralConfidenceAssessor(),
		behaviorAnalyzer:   newHistoricalBehaviorAnalyzer(),
		trustCalculator:   newCompositeTrustCalculator(),
		config: &InvisibleCaptchaConfig{
			EnableFingerprintOptimization:  true,
			EnableConfidenceAssessment:    true,
			EnableBehaviorAnalysis:        true,
			EnableTrustCalculation:        true,
			MinConfidenceThreshold:        0.6,
			MaxRiskScore:                  100.0,
			LearningWindowHours:           720,
			TrustDecayRate:                0.01,
			HistoryRetentionDays:           90,
			EnableAdaptiveScoring:          true,
		},
	}
}

func newInvisibleFingerprintEngine() *InvisibleFingerprintEngine {
	return &InvisibleFingerprintEngine{
		fingerprintCache:    make(map[string]*DeviceFingerprintRecord),
		componentWeights: map[string]float64{
			"canvas":          0.18,
			"webgl":           0.16,
			"audio":           0.10,
			"fonts":           0.12,
			"screen":          0.08,
			"timezone":        0.06,
			"language":        0.05,
			"platform":        0.05,
			"hardware":        0.08,
			"plugins":         0.04,
			"touch_support":   0.04,
			"color_depth":      0.02,
			"pixel_ratio":      0.02,
		},
		stabilityTracker: &FingerprintStabilityTracker{
			historicalRecords: make(map[string][]*StabilitySnapshot),
			snapshotInterval:  24 * time.Hour,
			maxSnapshots:      30,
		},
		uniquenessCalculator: &UniquenessCalculator{
			knownSignatures: make(map[string]int),
			collisionMap:    make(map[string][]string),
		},
	}
}

func newBehavioralConfidenceAssessor() *BehavioralConfidenceAssessor {
	return &BehavioralConfidenceAssessor{
		confidenceModels: make(map[string]*ConfidenceModel),
		scoringRules:     initConfidenceRules(),
		temporalWeights: map[string]float64{
			"morning":       1.0,
			"afternoon":     1.1,
			"evening":       1.0,
			"night":         0.9,
			"weekday":       1.0,
			"weekend":       1.1,
		},
		behaviorPatterns: &PatternDatabase{
			loginPatterns:     make(map[string]*LoginPattern),
			behaviorPatterns: make(map[string]*InvisibleCaptchaBehaviorPattern),
			locationPatterns: make(map[string]*InvisibleCaptchaLocationPattern),
		},
	}
}

func initConfidenceRules() []ConfidenceRule {
	return []ConfidenceRule{
		{
			Name:  "known_device_high_confidence",
			Condition: func(ctx *ConfidenceContext) bool {
				return ctx.IsKnownDevice && ctx.HistoricalSuccessRate > 0.9
			},
			ScoreModifier: 15.0,
			Weight:        1.2,
			Priority:      1,
		},
		{
			Name:  "known_location_trusted",
			Condition: func(ctx *ConfidenceContext) bool {
				return ctx.IsKnownLocation && ctx.RecentFailureCount == 0
			},
			ScoreModifier: 10.0,
			Weight:        1.1,
			Priority:      2,
		},
		{
			Name:  "high_success_rate",
			Condition: func(ctx *ConfidenceContext) bool {
				return ctx.HistoricalSuccessRate > 0.95
			},
			ScoreModifier: 20.0,
			Weight:        1.3,
			Priority:      1,
		},
		{
			Name:  "recent_failures",
			Condition: func(ctx *ConfidenceContext) bool {
				return ctx.RecentFailureCount > 3
			},
			ScoreModifier: -25.0,
			Weight:        1.5,
			Priority:      1,
		},
		{
			Name:  "unusual_time",
			Condition: func(ctx *ConfidenceContext) bool {
				return ctx.TimeOfDay < 5 || ctx.TimeOfDay > 23
			},
			ScoreModifier: -10.0,
			Weight:        0.8,
			Priority:      3,
		},
		{
			Name:  "high_frequency_requests",
			Condition: func(ctx *ConfidenceContext) bool {
				return ctx.RequestFrequency > 10.0
			},
			ScoreModifier: -15.0,
			Weight:        1.0,
			Priority:      2,
		},
	}
}

func newHistoricalBehaviorAnalyzer() *HistoricalBehaviorAnalyzer {
	return &HistoricalBehaviorAnalyzer{
		userHistories:   make(map[string]*UserBehaviorHistory),
		sessionAnalyzer: &SessionAnalyzer{
			sessionCache: make(map[string]*InvisibleCaptchaSessionData),
			maxCacheSize: 1000,
		},
		anomalyDetector: &BehaviorAnomalyDetector{
			anomalyRules:   initAnomalyRules(),
			baselineModels: make(map[string]*InvisibleCaptchaAnomalyBaseline),
		},
		trendAnalyzer: &InvisibleCaptchaTrendAnalyzer{
			trendData:      make(map[string]*InvisibleCaptchaTrendData),
			analysisWindow: 7 * 24 * time.Hour,
		},
	}
}

func initAnomalyRules() []AnomalyRule {
	return []AnomalyRule{
		{Name: "rapid_fire", Type: "frequency", Threshold: 10.0, Severity: 0.8, Weight: 1.5, Enabled: true},
		{Name: "unusual_time", Type: "temporal", Threshold: 3.0, Severity: 0.6, Weight: 1.2, Enabled: true},
		{Name: "location_change", Type: "geographic", Threshold: 500.0, Severity: 0.9, Weight: 1.8, Enabled: true},
		{Name: "device_mismatch", Type: "device", Threshold: 0.5, Severity: 0.7, Weight: 1.4, Enabled: true},
		{Name: "pattern_deviation", Type: "behavioral", Threshold: 2.0, Severity: 0.75, Weight: 1.5, Enabled: true},
		{Name: "response_time_anomaly", Type: "timing", Threshold: 0.1, Severity: 0.5, Weight: 1.0, Enabled: true},
	}
}

func newCompositeTrustCalculator() *CompositeTrustCalculator {
	return &CompositeTrustCalculator{
		trustComponents: map[string]*TrustComponent{
			"device_trust": {
				Name:          "device_trust",
				BaseWeight:    0.25,
				CurrentWeight: 0.25,
			},
			"behavior_trust": {
				Name:          "behavior_trust",
				BaseWeight:    0.30,
				CurrentWeight: 0.30,
			},
			"location_trust": {
				Name:          "location_trust",
				BaseWeight:    0.15,
				CurrentWeight: 0.15,
			},
			"history_trust": {
				Name:          "history_trust",
				BaseWeight:    0.20,
				CurrentWeight: 0.20,
			},
			"reputation_trust": {
				Name:          "reputation_trust",
				BaseWeight:    0.10,
				CurrentWeight: 0.10,
			},
		},
		weightManager: &TrustWeightManager{
			adaptiveWeights: make(map[string]float64),
			performanceLog:  make(map[string][]float64),
		},
		riskEngine: &RiskScoreEngine{
			riskFactors:   make(map[string]RiskFactor),
			globalRules:   initGlobalRiskRules(),
			historicalRisk: make(map[string][]float64),
		},
	}
}

func initGlobalRiskRules() []GlobalRiskRule {
	return []GlobalRiskRule{
		{
			Name: "proxy_detection",
			Conditions: []RiskCondition{
				{Field: "is_proxy", Operator: "==", Value: true},
			},
			ScoreMod: 25.0,
			Enabled:  true,
		},
		{
			Name: "vpn_usage",
			Conditions: []RiskCondition{
				{Field: "is_vpn", Operator: "==", Value: true},
			},
			ScoreMod: 20.0,
			Enabled:  true,
		},
		{
			Name: "tor_exit",
			Conditions: []RiskCondition{
				{Field: "is_tor", Operator: "==", Value: true},
			},
			ScoreMod: 35.0,
			Enabled:  true,
		},
		{
			Name: "hosting_provider",
			Conditions: []RiskCondition{
				{Field: "is_hosting", Operator: "==", Value: true},
			},
			ScoreMod: 30.0,
			Enabled:  true,
		},
		{
			Name: "new_device",
			Conditions: []RiskCondition{
				{Field: "device_age_days", Operator: "<", Value: 7},
			},
			ScoreMod: 15.0,
			Enabled:  true,
		},
	}
}

func (s *InvisibleCaptchaService) ProcessInvisibleVerification(req *InvisibleVerificationRequest) (*InvisibleVerificationResult, error) {
	result := &InvisibleVerificationResult{
		SessionID: req.SessionID,
		Timestamp: time.Now(),
	}

	if s.config.EnableFingerprintOptimization {
		fpResult := s.fingerprintEngine.OptimizeFingerprint(req.DeviceFingerprint, req.UserID, req.FingerprintComponents)
		result.FingerprintScore = fpResult.StabilityScore
		result.FingerprintUniqueness = fpResult.UniquenessScore
		result.FingerprintComponents = fpResult.Components
		result.FingerprintConfidence = fpResult.ConsistencyScore
	}

	if s.config.EnableConfidenceAssessment {
		confContext := &ConfidenceContext{
			UserID:              req.UserID,
			DeviceFingerprint:   req.DeviceFingerprint,
			BehaviorData:        req.BehaviorData,
			TimeOfDay:           time.Now().Hour(),
			DayOfWeek:           int(time.Now().Weekday()),
			IsKnownDevice:       s.isKnownDevice(req.UserID, req.DeviceFingerprint),
			IsKnownLocation:     s.isKnownLocation(req.UserID, req.IPAddress),
			HistoricalSuccessRate: s.getHistoricalSuccessRate(req.UserID),
			RecentFailureCount:    s.getRecentFailureCount(req.UserID),
			RequestFrequency:      s.calculateRequestFrequency(req.UserID),
		}
		confResult := s.confidenceAssessor.AssessConfidence(confContext)
		result.ConfidenceScore = confResult.TotalScore
		result.ConfidenceFactors = confResult.Factors
		result.ConfidenceLevel = confResult.Level
	}

	if s.config.EnableBehaviorAnalysis {
		behaviorResult := s.behaviorAnalyzer.AnalyzeBehavior(req.UserID, req.BehaviorData, req.EnvironmentData)
		result.BehaviorAnomalyScore = behaviorResult.AnomalyScore
		result.BehaviorPatternMatch = behaviorResult.PatternMatch
		result.BehaviorTrends = behaviorResult.Trends
		result.RecommendedAction = s.determineAction(behaviorResult, result.ConfidenceScore)
	}

	if s.config.EnableTrustCalculation {
		trustResult := s.trustCalculator.CalculateTrustScore(s.buildTrustContext(req, result))
		result.TrustScore = trustResult.TotalScore
		result.TrustBreakdown = trustResult.ComponentScores
		result.RiskScore = trustResult.RiskScore
		result.RiskFactors = trustResult.RiskFactors
	}

	result.ShouldChallenge = s.shouldIssueChallenge(result)
	result.SkipReason = s.determineSkipReason(result)

	return result, nil
}

func (s *InvisibleFingerprintEngine) OptimizeFingerprint(fingerprint, userID string, components *FingerprintComponents) *FingerprintOptimizationResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := &FingerprintOptimizationResult{
		Fingerprint: fingerprint,
		Components:  components,
	}

	record, exists := s.fingerprintCache[fingerprint]
	if !exists {
		record = &DeviceFingerprintRecord{
			Fingerprint:       fingerprint,
			Components:        components,
			FirstSeen:         time.Now(),
			LastSeen:          time.Now(),
			AppearanceCount:   1,
			ConsistencyScore:  0.5,
			UniquenessScore:   s.uniquenessCalculator.CalculateUniqueness(fingerprint),
			AssociatedUsers:   make(map[string]int),
			ComponentHistory:  make(map[string][]string),
			VersionHistory:    []string{fingerprint},
		}
		s.fingerprintCache[fingerprint] = record
		s.uniquenessCalculator.RegisterSignature(fingerprint)
	} else {
		record.LastSeen = time.Now()
		record.AppearanceCount++
	}

	record.AssociatedUsers[userID]++

	if components != nil {
		result.StabilityScore = s.calculateStabilityScore(record, components)
		result.UniquenessScore = s.uniquenessCalculator.CalculateUniqueness(fingerprint)
		result.ConsistencyScore = s.calculateConsistencyScore(record)
		result.QualityScore = s.calculateQualityScore(components)
	}

	s.updateStabilitySnapshot(fingerprint, result)

	return result
}

func (s *InvisibleFingerprintEngine) calculateStabilityScore(record *DeviceFingerprintRecord, components *FingerprintComponents) float64 {
	if record.AppearanceCount < 2 {
		return 0.3
	}

	usageScore := math.Min(1.0, float64(record.AppearanceCount)/10.0) * 0.4

	userDiversity := 1.0
	if len(record.AssociatedUsers) > 1 {
		userDiversity = 1.0 / math.Log(float64(len(record.AssociatedUsers))+1)
	}
	userScore := userDiversity * 0.3

	ageHours := time.Since(record.FirstSeen).Hours()
	ageScore := math.Min(1.0, ageHours/168.0) * 0.3

	componentScore := 0.0
	if components != nil {
		componentScore = s.calculateComponentStability(components) * 0.3
	}

	return usageScore + userScore + ageScore + componentScore
}

func (s *InvisibleFingerprintEngine) calculateComponentStability(components *FingerprintComponents) float64 {
	score := 0.0
	count := 0

	if components.CanvasHash != "" && components.CanvasHash != "error" {
		score += s.componentWeights["canvas"]
		count++
	}
	if components.WebGLHash != "" {
		score += s.componentWeights["webgl"]
		count++
	}
	if components.AudioHash != "" && components.AudioHash != "error" {
		score += s.componentWeights["audio"]
		count++
	}
	if len(components.FontHash) > 0 {
		score += s.componentWeights["fonts"]
		count++
	}
	if components.ScreenHash != "" {
		score += s.componentWeights["screen"]
		count++
	}
	if components.TimezoneHash != "" {
		score += s.componentWeights["timezone"]
		count++
	}

	if count == 0 {
		return 0.0
	}

	return score / 0.8
}

func (s *InvisibleFingerprintEngine) calculateConsistencyScore(record *DeviceFingerprintRecord) float64 {
	if record.AppearanceCount < 2 {
		return 0.5
	}

	snapshots := s.stabilityTracker.historicalRecords[record.Fingerprint]
	if len(snapshots) < 2 {
		return 0.6
	}

	var totalConsistency float64
	for _, snap := range snapshots {
		totalConsistency += snap.ConsistencyScore
	}

	return totalConsistency / float64(len(snapshots))
}

func (s *InvisibleFingerprintEngine) calculateQualityScore(components *FingerprintComponents) float64 {
	if components == nil {
		return 0.0
	}

	score := 0.0
	totalWeight := 0.0

	componentsWithWeight := map[string]struct {
		present bool
		weight  float64
	}{
		"canvas":          {components.CanvasHash != "", 0.20},
		"webgl":           {components.WebGLHash != "", 0.18},
		"audio":           {components.AudioHash != "", 0.12},
		"fonts":           {len(components.FontHash) > 0, 0.10},
		"screen":          {components.ScreenHash != "", 0.08},
		"hardware":        {components.HardwareConcurrency > 0, 0.10},
		"plugins":         {components.WebGLRenderer != "", 0.08},
		"touch_support":   {true, 0.04},
		"color_depth":     {components.ColorDepth > 0, 0.05},
		"pixel_ratio":     {components.PixelRatio > 0, 0.05},
	}

	for _, data := range componentsWithWeight {
		totalWeight += data.weight
		if data.present {
			score += data.weight
		}
	}

	if totalWeight == 0 {
		return 0.0
	}

	return score / totalWeight * 100
}

func (s *InvisibleFingerprintEngine) updateStabilitySnapshot(fingerprint string, result *FingerprintOptimizationResult) {
	snapshot := &StabilitySnapshot{
		Timestamp:        time.Now(),
		ConsistencyScore: result.ConsistencyScore,
		ComponentMatch:   result.QualityScore / 100.0,
		AnomalyDetected:  false,
	}

	s.stabilityTracker.historicalRecords[fingerprint] = append(
		s.stabilityTracker.historicalRecords[fingerprint], snapshot,
	)

	if len(s.stabilityTracker.historicalRecords[fingerprint]) > s.stabilityTracker.maxSnapshots {
		s.stabilityTracker.historicalRecords[fingerprint] = s.stabilityTracker.historicalRecords[fingerprint][1:]
	}
}

func (s *UniquenessCalculator) CalculateUniqueness(fingerprint string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := s.knownSignatures[fingerprint]

	baseScore := 100.0
	if count == 0 {
		baseScore = 95.0
	} else if count == 1 {
		baseScore = 85.0
	} else if count < 5 {
		baseScore = 70.0
	} else {
		baseScore = 50.0
	}

	return baseScore
}

func (s *UniquenessCalculator) RegisterSignature(fingerprint string) {
	s.knownSignatures[fingerprint]++
	s.signatureCount++

	for existing := range s.knownSignatures {
		if existing != fingerprint && s.calculateSimilarity(fingerprint, existing) > 0.9 {
			s.collisionMap[fingerprint] = append(s.collisionMap[fingerprint], existing)
		}
	}
}

func (s *UniquenessCalculator) calculateSimilarity(fp1, fp2 string) float64 {
	if len(fp1) != len(fp2) {
		return 0
	}

	matches := 0
	for i := range fp1 {
		if fp1[i] == fp2[i] {
			matches++
		}
	}

	return float64(matches) / float64(len(fp1))
}

func (s *BehavioralConfidenceAssessor) AssessConfidence(ctx *ConfidenceContext) *ConfidenceAssessmentResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := &ConfidenceAssessmentResult{
		BaseScore: 50.0,
		Factors:   make([]ConfidenceFactor, 0),
		Level:     "medium",
	}

	result.BaseScore += s.evaluateDeviceConfidence(ctx)
	result.BaseScore += s.evaluateHistoricalConfidence(ctx)
	result.BaseScore += s.evaluateTemporalConfidence(ctx)
	result.BaseScore += s.evaluateBehaviorConfidence(ctx)

	for _, rule := range s.scoringRules {
		if rule.Condition(ctx) {
			modifier := rule.ScoreModifier * rule.Weight
			result.BaseScore += modifier
			result.Factors = append(result.Factors, ConfidenceFactor{
				Name:   rule.Name,
				Weight: rule.Weight,
				Score:  modifier,
			})
		}
	}

	result.TotalScore = math.Max(0, math.Min(100, result.BaseScore))

	if result.TotalScore >= 80 {
		result.Level = "high"
	} else if result.TotalScore >= 60 {
		result.Level = "medium"
	} else if result.TotalScore >= 40 {
		result.Level = "low"
	} else {
		result.Level = "critical"
	}

	s.updateConfidenceModel(ctx, result)

	return result
}

func (s *BehavioralConfidenceAssessor) evaluateDeviceConfidence(ctx *ConfidenceContext) float64 {
	score := 0.0

	if ctx.IsKnownDevice {
		score += 25.0
	} else {
		score -= 10.0
	}

	if ctx.HistoricalSuccessRate > 0.9 {
		score += 15.0
	} else if ctx.HistoricalSuccessRate > 0.7 {
		score += 5.0
	}

	return score
}

func (s *BehavioralConfidenceAssessor) evaluateHistoricalConfidence(ctx *ConfidenceContext) float64 {
	score := 0.0

	score += ctx.HistoricalSuccessRate * 20.0

	if ctx.RecentFailureCount == 0 {
		score += 10.0
	} else if ctx.RecentFailureCount <= 2 {
		score += 5.0
	} else if ctx.RecentFailureCount <= 5 {
		score -= 10.0
	} else {
		score -= 20.0
	}

	return score
}

func (s *BehavioralConfidenceAssessor) evaluateTemporalConfidence(ctx *ConfidenceContext) float64 {
	score := 0.0

	timeWeight := 1.0
	if ctx.TimeOfDay >= 6 && ctx.TimeOfDay <= 22 {
		timeWeight = s.temporalWeights["daytime"]
	} else {
		timeWeight = s.temporalWeights["night"]
	}
	score += (timeWeight - 1.0) * 10.0

	if ctx.DayOfWeek >= 1 && ctx.DayOfWeek <= 5 {
		score += (s.temporalWeights["weekday"] - 1.0) * 10.0
	} else {
		score += (s.temporalWeights["weekend"] - 1.0) * 10.0
	}

	return score
}

func (s *BehavioralConfidenceAssessor) evaluateBehaviorConfidence(ctx *ConfidenceContext) float64 {
	score := 0.0

	if ctx.RequestFrequency > 0 && ctx.RequestFrequency <= 5 {
		score += 10.0
	} else if ctx.RequestFrequency > 5 && ctx.RequestFrequency <= 10 {
		score += 5.0
	} else if ctx.RequestFrequency > 10 {
		score -= 15.0
	}

	if len(ctx.BehaviorData) > 10 {
		score += 10.0
	} else if len(ctx.BehaviorData) > 5 {
		score += 5.0
	}

	return score
}

func (s *BehavioralConfidenceAssessor) updateConfidenceModel(ctx *ConfidenceContext, result *ConfidenceAssessmentResult) {
	modelKey := ctx.UserID + ":" + ctx.DeviceFingerprint

	model, exists := s.confidenceModels[modelKey]
	if !exists {
		model = &ConfidenceModel{
			UserID:            ctx.UserID,
			DeviceFingerprint: ctx.DeviceFingerprint,
			ConfidenceHistory: make([]ConfidenceRecord, 0),
			CurrentConfidence: result.TotalScore,
			BaseScore:         result.BaseScore,
			ModifierFactors:   make(map[string]float64),
			LastUpdated:       time.Now(),
			SampleCount:       0,
		}
		s.confidenceModels[modelKey] = model
	}

	for _, factor := range result.Factors {
		model.ModifierFactors[factor.Name] = factor.Score
	}

	model.CurrentConfidence = result.TotalScore
	model.SampleCount++
	model.LastUpdated = time.Now()

	record := ConfidenceRecord{
		Timestamp:        time.Now(),
		ConfidenceScore:  result.TotalScore,
		Factors:          result.Factors,
		VerificationResult: true,
	}
	model.ConfidenceHistory = append(model.ConfidenceHistory, record)

	if len(model.ConfidenceHistory) > 100 {
		model.ConfidenceHistory = model.ConfidenceHistory[1:]
	}
}

func (s *HistoricalBehaviorAnalyzer) GetOrCreateHistory(userID string) *UserBehaviorHistory {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, exists := s.userHistories[userID]
	if !exists {
		history = &UserBehaviorHistory{
			UserID:              userID,
			VerificationHistory: make([]*VerificationRecord, 0),
			DeviceHistory:       make([]*DeviceRecord, 0),
			LocationHistory:     make([]*LocationRecord, 0),
			SessionMetrics: &InvisibleCaptchaSessionMetrics{
				ActiveHours: make(map[int]int),
			},
			RiskEvolution: make([]float64, 0),
			LastUpdate:    time.Now(),
		}
		s.userHistories[userID] = history
	}

	return history
}

func (s *HistoricalBehaviorAnalyzer) AnalyzeBehavior(userID string, behaviorData []models.BehaviorData, envData map[string]interface{}) *BehaviorAnalysisResult {
	result := &BehaviorAnalysisResult{
		UserID: userID,
	}

	s.mu.Lock()
	history, exists := s.userHistories[userID]
	if !exists {
		history = &UserBehaviorHistory{
			UserID:              userID,
			VerificationHistory: make([]*VerificationRecord, 0),
			DeviceHistory:       make([]*DeviceRecord, 0),
			LocationHistory:     make([]*LocationRecord, 0),
			SessionMetrics: &InvisibleCaptchaSessionMetrics{
				ActiveHours: make(map[int]int),
			},
			RiskEvolution: make([]float64, 0),
			LastUpdate:    time.Now(),
		}
		s.userHistories[userID] = history
	}
	s.mu.Unlock()

	result.AnomalyScore = s.anomalyDetector.DetectAnomalies(history, behaviorData, envData)
	result.PatternMatch = s.calculatePatternMatch(history, behaviorData)
	result.Trends = s.trendAnalyzer.AnalyzeTrends(history)

	if len(history.VerificationHistory) > 0 {
		result.HistoricalScore = s.calculateHistoricalScore(history)
		result.RecentSuccessRate = s.calculateRecentSuccessRate(history)
	}

	s.updateHistory(history, behaviorData, envData)

	return result
}

func (s *BehaviorAnomalyDetector) DetectAnomalies(history *UserBehaviorHistory, behaviorData []models.BehaviorData, envData map[string]interface{}) float64 {
	anomalyScore := 0.0

	for _, rule := range s.anomalyRules {
		if !rule.Enabled {
			continue
		}

		score := s.evaluateRule(rule, history, behaviorData, envData)
		if score > rule.Threshold {
			anomalyScore += rule.Severity * rule.Weight
		}
	}

	return math.Min(100, anomalyScore)
}

func (s *BehaviorAnomalyDetector) evaluateRule(rule AnomalyRule, history *UserBehaviorHistory, behaviorData []models.BehaviorData, envData map[string]interface{}) float64 {
	switch rule.Type {
	case "frequency":
		if history != nil && len(history.VerificationHistory) > 0 {
			lastHour := time.Now().Add(-1 * time.Hour)
			count := 0
			for _, v := range history.VerificationHistory {
				if v.Timestamp.After(lastHour) {
					count++
				}
			}
			return float64(count)
		}
	case "temporal":
		currentHour := time.Now().Hour()
		if history != nil && history.SessionMetrics != nil {
			typicalHour := false
			if count, ok := history.SessionMetrics.ActiveHours[currentHour]; ok && count > 5 {
				typicalHour = true
			}
			if !typicalHour {
				return 1.0
			}
		}
	case "geographic":
		if ip, ok := envData["ip_country"].(string); ok {
			for _, loc := range history.LocationHistory {
				if loc.Location == ip && time.Since(loc.LastSeen) < 30*time.Minute {
					return 0.0
				}
			}
		}
	case "behavioral":
		if len(behaviorData) > 0 {
			avgResponse := s.calculateAvgResponseTime(behaviorData)
			if history != nil && history.SessionMetrics != nil {
				typicalAvg := float64(history.SessionMetrics.AvgSessionDuration.Milliseconds())
				if typicalAvg > 0 {
					deviation := math.Abs(float64(avgResponse) - typicalAvg) / typicalAvg
					return deviation
				}
			}
		}
	}

	return 0.0
}

func (s *BehaviorAnomalyDetector) calculateAvgResponseTime(behaviorData []models.BehaviorData) int64 {
	if len(behaviorData) == 0 {
		return 0
	}

	var total int64
	for _, data := range behaviorData {
		total += int64(len(data.Data))
	}

	return total / int64(len(behaviorData))
}

func (s *HistoricalBehaviorAnalyzer) calculatePatternMatch(history *UserBehaviorHistory, behaviorData []models.BehaviorData) float64 {
	if history == nil || len(history.VerificationHistory) == 0 {
		return 0.5
	}

	patternScore := 0.0
	weight := 0.0

	if len(history.VerificationHistory) >= 5 {
		successCount := 0
		for _, v := range history.VerificationHistory[len(history.VerificationHistory)-5:] {
			if v.VerificationSuccess {
				successCount++
			}
		}
		patternScore += float64(successCount) / 5.0 * 40.0
		weight += 40.0
	}

	if len(behaviorData) > 0 {
		avgResponse := s.anomalyDetector.calculateAvgResponseTime(behaviorData)
		patternScore += 30.0
		weight += 30.0
		_ = avgResponse
	}

	currentHour := time.Now().Hour()
	if history.SessionMetrics != nil {
		if count, ok := history.SessionMetrics.ActiveHours[currentHour]; ok && count > 3 {
			patternScore += 30.0
			weight += 30.0
		}
	}

	if weight == 0 {
		return 0.5
	}

	return patternScore / weight
}

func (s *InvisibleCaptchaTrendAnalyzer) AnalyzeTrends(history *UserBehaviorHistory) map[string]interface{} {
	trends := make(map[string]interface{})

	if history == nil || len(history.RiskEvolution) < 3 {
		trends["status"] = "insufficient_data"
		return trends
	}

	trends["status"] = "analyzed"

	if len(history.RiskEvolution) >= 3 {
		recent := history.RiskEvolution[len(history.RiskEvolution)-3:]
		older := history.RiskEvolution[:len(history.RiskEvolution)-3]

		if len(older) > 0 {
			avgRecent := s.average(recent)
			avgOlder := s.average(older)
			change := (avgRecent - avgOlder) / math.Max(1, avgOlder)

			if change > 0.2 {
				trends["risk_trend"] = "increasing"
			} else if change < -0.2 {
				trends["risk_trend"] = "decreasing"
			} else {
				trends["risk_trend"] = "stable"
			}
			trends["risk_change_rate"] = change
		}
	}

	return trends
}

func (s *InvisibleCaptchaTrendAnalyzer) average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

func (s *HistoricalBehaviorAnalyzer) calculateHistoricalScore(history *UserBehaviorHistory) float64 {
	if len(history.VerificationHistory) == 0 {
		return 50.0
	}

	successCount := 0
	totalCount := len(history.VerificationHistory)

	for _, v := range history.VerificationHistory {
		if v.VerificationSuccess {
			successCount++
		}
	}

	baseScore := float64(successCount) / float64(totalCount) * 70.0

	challengeCount := 0
	challengeSuccess := 0
	for _, v := range history.VerificationHistory {
		if v.ChallengeIssued {
			challengeCount++
			if v.VerificationSuccess {
				challengeSuccess++
			}
		}
	}

	if challengeCount > 0 {
		challengeScore := float64(challengeSuccess) / float64(challengeCount) * 30.0
		baseScore += challengeScore
	} else {
		baseScore += 30.0
	}

	return baseScore
}

func (s *HistoricalBehaviorAnalyzer) calculateRecentSuccessRate(history *UserBehaviorHistory) float64 {
	if len(history.VerificationHistory) == 0 {
		return 0.5
	}

	window := 10
	if len(history.VerificationHistory) < window {
		window = len(history.VerificationHistory)
	}

	recent := history.VerificationHistory[len(history.VerificationHistory)-window:]
	successCount := 0

	for _, v := range recent {
		if v.VerificationSuccess {
			successCount++
		}
	}

	return float64(successCount) / float64(window)
}

func (s *HistoricalBehaviorAnalyzer) updateHistory(history *UserBehaviorHistory, behaviorData []models.BehaviorData, envData map[string]interface{}) {
	record := &VerificationRecord{
		Timestamp:       time.Now(),
		RiskScore:       0.0,
		Confidence:      0.0,
		TrustScore:      0.0,
		VerificationSuccess: true,
		ChallengeIssued: false,
		EnvironmentData: envData,
	}

	if len(behaviorData) > 0 {
		record.ResponseTimeMs = int64(len(behaviorData[len(behaviorData)-1].Data))
	}

	history.VerificationHistory = append(history.VerificationHistory, record)
	history.LastUpdate = time.Now()

	if len(history.VerificationHistory) > 500 {
		history.VerificationHistory = history.VerificationHistory[1:]
	}

	history.RiskEvolution = append(history.RiskEvolution, record.RiskScore)
	if len(history.RiskEvolution) > 100 {
		history.RiskEvolution = history.RiskEvolution[1:]
	}
}

func (s *CompositeTrustCalculator) CalculateTrustScore(ctx *TrustCalculationContext) *TrustCalculationResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := &TrustCalculationResult{
		ComponentScores: make(map[string]float64),
		RiskFactors:     make([]string, 0),
	}

	result.ComponentScores["device_trust"] = s.calculateDeviceTrust(ctx)
	result.ComponentScores["behavior_trust"] = s.calculateBehaviorTrust(ctx)
	result.ComponentScores["location_trust"] = s.calculateLocationTrust(ctx)
	result.ComponentScores["history_trust"] = s.calculateHistoryTrust(ctx)
	result.ComponentScores["reputation_trust"] = s.calculateReputationTrust(ctx)

	totalWeight := 0.0
	weightedSum := 0.0
	for name, component := range s.trustComponents {
		weight := component.CurrentWeight
		if s.weightManager != nil {
			if adaptiveWeight, ok := s.weightManager.adaptiveWeights[name]; ok {
				weight = adaptiveWeight
			}
		}
		weightedSum += result.ComponentScores[name] * weight
		totalWeight += weight
	}

	if totalWeight > 0 {
		result.TotalScore = weightedSum / totalWeight
	} else {
		result.TotalScore = 50.0
	}

	var riskFactors []string
	result.RiskScore = s.riskEngine.CalculateRiskScore(ctx, &riskFactors)
	result.RiskFactors = riskFactors

	return result
}

func (s *CompositeTrustCalculator) calculateDeviceTrust(ctx *TrustCalculationContext) float64 {
	score := 50.0

	if ctx.IsKnownDevice {
		score += 25.0
	}

	if ctx.DeviceAgeDays > 30 {
		score += 15.0
	} else if ctx.DeviceAgeDays > 7 {
		score += 10.0
	} else if ctx.DeviceAgeDays > 0 {
		score += 5.0
	}

	if ctx.DeviceUseCount > 10 {
		score += 10.0
	} else if ctx.DeviceUseCount > 5 {
		score += 5.0
	}

	if ctx.DeviceSuccessRate > 0.95 {
		score += 15.0
	} else if ctx.DeviceSuccessRate > 0.8 {
		score += 10.0
	} else if ctx.DeviceSuccessRate < 0.5 {
		score -= 20.0
	}

	return math.Max(0, math.Min(100, score))
}

func (s *CompositeTrustCalculator) calculateBehaviorTrust(ctx *TrustCalculationContext) float64 {
	score := 50.0

	if ctx.BehaviorConsistency > 0.9 {
		score += 30.0
	} else if ctx.BehaviorConsistency > 0.7 {
		score += 20.0
	} else if ctx.BehaviorConsistency < 0.5 {
		score -= 25.0
	}

	if ctx.ResponseTimeScore > 0.9 {
		score += 10.0
	} else if ctx.ResponseTimeScore < 0.5 {
		score -= 15.0
	}

	if ctx.PatternMatchScore > 0.8 {
		score += 15.0
	} else if ctx.PatternMatchScore > 0.6 {
		score += 5.0
	}

	return math.Max(0, math.Min(100, score))
}

func (s *CompositeTrustCalculator) calculateLocationTrust(ctx *TrustCalculationContext) float64 {
	score := 50.0

	if ctx.IsKnownLocation {
		score += 30.0
	}

	if ctx.LocationChangeRate < 0.1 {
		score += 15.0
	} else if ctx.LocationChangeRate < 0.3 {
		score += 5.0
	} else {
		score -= 20.0
	}

	if !ctx.IsSuspiciousLocation {
		score += 10.0
	}

	return math.Max(0, math.Min(100, score))
}

func (s *CompositeTrustCalculator) calculateHistoryTrust(ctx *TrustCalculationContext) float64 {
	score := 50.0

	if ctx.HistoricalSuccessRate > 0.95 {
		score += 35.0
	} else if ctx.HistoricalSuccessRate > 0.85 {
		score += 25.0
	} else if ctx.HistoricalSuccessRate > 0.7 {
		score += 10.0
	} else if ctx.HistoricalSuccessRate < 0.5 {
		score -= 30.0
	}

	if ctx.AccountAgeDays > 365 {
		score += 15.0
	} else if ctx.AccountAgeDays > 90 {
		score += 10.0
	} else if ctx.AccountAgeDays < 30 {
		score -= 10.0
	}

	if ctx.TotalVerifications > 100 {
		score += 10.0
	} else if ctx.TotalVerifications < 10 {
		score -= 5.0
	}

	return math.Max(0, math.Min(100, score))
}

func (s *CompositeTrustCalculator) calculateReputationTrust(ctx *TrustCalculationContext) float64 {
	score := 50.0

	if !ctx.IsProxy && !ctx.IsVPN && !ctx.IsTor && !ctx.IsHosting {
		score += 30.0
	} else {
		if ctx.IsProxy {
			score -= 15.0
		}
		if ctx.IsVPN {
			score -= 10.0
		}
		if ctx.IsTor {
			score -= 25.0
		}
		if ctx.IsHosting {
			score -= 20.0
		}
	}

	if ctx.IPReputationScore > 80 {
		score += 20.0
	} else if ctx.IPReputationScore > 50 {
		score += 10.0
	} else if ctx.IPReputationScore < 30 {
		score -= 25.0
	}

	return math.Max(0, math.Min(100, score))
}

func (s *RiskScoreEngine) CalculateRiskScore(ctx *TrustCalculationContext, riskFactors *[]string) float64 {
	riskScore := 0.0

	for _, rule := range s.globalRules {
		if !rule.Enabled {
			continue
		}

		if s.evaluateConditions(ctx, rule.Conditions) {
			riskScore += rule.ScoreMod
			if riskFactors != nil {
				*riskFactors = append(*riskFactors, rule.Name)
			}
		}
	}

	if ctx.RecentFailureCount > 3 {
		riskScore += float64(ctx.RecentFailureCount-3) * 5.0
		if riskFactors != nil {
			*riskFactors = append(*riskFactors, "recent_failures")
		}
	}

	if ctx.RequestFrequency > 20 {
		riskScore += 20.0
		if riskFactors != nil {
			*riskFactors = append(*riskFactors, "high_request_frequency")
		}
	}

	if ctx.ConfidenceScore < 40 {
		riskScore += (40 - ctx.ConfidenceScore) * 0.5
		if riskFactors != nil {
			*riskFactors = append(*riskFactors, "low_confidence")
		}
	}

	return math.Max(0, math.Min(100, riskScore))
}

func (s *RiskScoreEngine) evaluateConditions(ctx *TrustCalculationContext, conditions []RiskCondition) bool {
	for _, cond := range conditions {
		switch cond.Field {
		case "is_proxy":
			if ctx.IsProxy != (cond.Value.(bool)) {
				return false
			}
		case "is_vpn":
			if ctx.IsVPN != (cond.Value.(bool)) {
				return false
			}
		case "is_tor":
			if ctx.IsTor != (cond.Value.(bool)) {
				return false
			}
		case "is_hosting":
			if ctx.IsHosting != (cond.Value.(bool)) {
				return false
			}
		case "device_age_days":
			threshold := cond.Value.(int)
			if cond.Operator == "<" && ctx.DeviceAgeDays >= threshold {
				return false
			}
			if cond.Operator == ">" && ctx.DeviceAgeDays <= threshold {
				return false
			}
		}
	}
	return true
}

func (s *InvisibleCaptchaService) buildTrustContext(req *InvisibleVerificationRequest, result *InvisibleVerificationResult) *TrustCalculationContext {
	return &TrustCalculationContext{
		UserID:                 req.UserID,
		DeviceFingerprint:      req.DeviceFingerprint,
		IsKnownDevice:         s.isKnownDevice(req.UserID, req.DeviceFingerprint),
		DeviceAgeDays:          s.getDeviceAgeDays(req.UserID, req.DeviceFingerprint),
		DeviceUseCount:         s.getDeviceUseCount(req.UserID, req.DeviceFingerprint),
		DeviceSuccessRate:     s.getDeviceSuccessRate(req.UserID, req.DeviceFingerprint),
		IsKnownLocation:        s.isKnownLocation(req.UserID, req.IPAddress),
		LocationChangeRate:     s.calculateLocationChangeRate(req.UserID),
		IsSuspiciousLocation:   s.isSuspiciousLocation(req.IPAddress),
		BehaviorConsistency:    result.BehaviorPatternMatch,
		ResponseTimeScore:      s.calculateResponseTimeScore(req.BehaviorData),
		PatternMatchScore:      result.BehaviorPatternMatch,
		HistoricalSuccessRate: s.getHistoricalSuccessRate(req.UserID),
		AccountAgeDays:         s.getAccountAgeDays(req.UserID),
		TotalVerifications:    s.getTotalVerificationCount(req.UserID),
		RecentFailureCount:    s.getRecentFailureCount(req.UserID),
		RequestFrequency:       s.calculateRequestFrequency(req.UserID),
		IsProxy:                s.detectProxy(req.EnvironmentData),
		IsVPN:                  s.detectVPN(req.EnvironmentData),
		IsTor:                  s.detectTor(req.EnvironmentData),
		IsHosting:              s.detectHosting(req.EnvironmentData),
		IPReputationScore:      s.getIPReputationScore(req.IPAddress),
		ConfidenceScore:        result.ConfidenceScore,
	}
}

func (s *InvisibleCaptchaService) shouldIssueChallenge(result *InvisibleVerificationResult) bool {
	if result.ConfidenceScore < 40 {
		return true
	}

	if result.BehaviorAnomalyScore > 60 {
		return true
	}

	if result.RiskScore > 70 {
		return true
	}

	if result.TrustScore < 50 {
		return true
	}

	return false
}

func (s *InvisibleCaptchaService) determineSkipReason(result *InvisibleVerificationResult) string {
	if result.TrustScore >= 80 && result.ConfidenceScore >= 80 && result.BehaviorAnomalyScore < 20 {
		return "high_trust_user"
	}

	if result.ConfidenceScore >= 90 && result.BehaviorPatternMatch >= 0.9 {
		return "consistent_behavior_pattern"
	}

	if result.FingerprintConfidence >= 0.9 && result.FingerprintUniqueness >= 80 {
		return "trusted_device"
	}

	return ""
}

func (s *InvisibleCaptchaService) determineAction(behaviorResult *BehaviorAnalysisResult, confidence float64) string {
	if behaviorResult.AnomalyScore > 80 {
		return "block"
	}

	if behaviorResult.AnomalyScore > 50 || confidence < 40 {
		return "challenge"
	}

	if behaviorResult.PatternMatch > 0.8 && confidence >= 70 {
		return "allow"
	}

	return "review"
}

func (s *InvisibleCaptchaService) isKnownDevice(userID, fingerprint string) bool {
	history := s.behaviorAnalyzer.GetOrCreateHistory(userID)
	for _, device := range history.DeviceHistory {
		if device.Fingerprint == fingerprint {
			return true
		}
	}
	return false
}

func (s *InvisibleCaptchaService) isKnownLocation(userID, ipAddress string) bool {
	history := s.behaviorAnalyzer.GetOrCreateHistory(userID)
	for _, loc := range history.LocationHistory {
		if loc.Location == ipAddress && !loc.IsAnomaly {
			return true
		}
	}
	return false
}

func (s *InvisibleCaptchaService) getHistoricalSuccessRate(userID string) float64 {
	history := s.behaviorAnalyzer.GetOrCreateHistory(userID)
	if len(history.VerificationHistory) == 0 {
		return 0.5
	}

	successCount := 0
	for _, v := range history.VerificationHistory {
		if v.VerificationSuccess {
			successCount++
		}
	}

	return float64(successCount) / float64(len(history.VerificationHistory))
}

func (s *InvisibleCaptchaService) getRecentFailureCount(userID string) int {
	history := s.behaviorAnalyzer.GetOrCreateHistory(userID)
	if len(history.VerificationHistory) == 0 {
		return 0
	}

	count := 0
	startIdx := 0
	if len(history.VerificationHistory) > 10 {
		startIdx = len(history.VerificationHistory) - 10
	}
	for _, v := range history.VerificationHistory[startIdx:] {
		if !v.VerificationSuccess {
			count++
		}
	}

	return count
}

func (s *InvisibleCaptchaService) calculateRequestFrequency(userID string) float64 {
	history := s.behaviorAnalyzer.GetOrCreateHistory(userID)
	if len(history.VerificationHistory) == 0 {
		return 1.0
	}

	lastHour := time.Now().Add(-1 * time.Hour)
	count := 0
	for _, v := range history.VerificationHistory {
		if v.Timestamp.After(lastHour) {
			count++
		}
	}

	return float64(count)
}

func (s *InvisibleCaptchaService) getDeviceAgeDays(userID, fingerprint string) int {
	history := s.behaviorAnalyzer.GetOrCreateHistory(userID)
	for _, device := range history.DeviceHistory {
		if device.Fingerprint == fingerprint {
			return int(time.Since(device.FirstSeen).Hours() / 24)
		}
	}
	return 0
}

func (s *InvisibleCaptchaService) getDeviceUseCount(userID, fingerprint string) int {
	history := s.behaviorAnalyzer.GetOrCreateHistory(userID)
	for _, device := range history.DeviceHistory {
		if device.Fingerprint == fingerprint {
			return device.UseCount
		}
	}
	return 0
}

func (s *InvisibleCaptchaService) getDeviceSuccessRate(userID, fingerprint string) float64 {
	history := s.behaviorAnalyzer.GetOrCreateHistory(userID)
	for _, device := range history.DeviceHistory {
		if device.Fingerprint == fingerprint {
			return device.SuccessRate
		}
	}
	return 0.5
}

func (s *InvisibleCaptchaService) calculateLocationChangeRate(userID string) float64 {
	history := s.behaviorAnalyzer.GetOrCreateHistory(userID)
	if len(history.LocationHistory) < 2 {
		return 0.0
	}

	changes := 0
	for i := 1; i < len(history.LocationHistory) && i < 10; i++ {
		if history.LocationHistory[i].Location != history.LocationHistory[i-1].Location {
			changes++
		}
	}

	return float64(changes) / float64(math.Min(10, float64(len(history.LocationHistory))))
}

func (s *InvisibleCaptchaService) isSuspiciousLocation(ipAddress string) bool {
	suspicious := []string{"hosting", "datacenter", "cloud", "vpn"}
	lowerIP := strings.ToLower(ipAddress)
	for _, s := range suspicious {
		if strings.Contains(lowerIP, s) {
			return true
		}
	}
	return false
}

func (s *InvisibleCaptchaService) getAccountAgeDays(userID string) int {
	return 30
}

func (s *InvisibleCaptchaService) getTotalVerificationCount(userID string) int {
	history := s.behaviorAnalyzer.GetOrCreateHistory(userID)
	return len(history.VerificationHistory)
}

func (s *InvisibleCaptchaService) calculateResponseTimeScore(behaviorData []models.BehaviorData) float64 {
	if len(behaviorData) == 0 {
		return 0.5
	}

	return 0.8
}

func (s *InvisibleCaptchaService) detectProxy(envData map[string]interface{}) bool {
	if val, ok := envData["proxy_detected"].(bool); ok {
		return val
	}
	return false
}

func (s *InvisibleCaptchaService) detectVPN(envData map[string]interface{}) bool {
	if val, ok := envData["vpn_detected"].(bool); ok {
		return val
	}
	return false
}

func (s *InvisibleCaptchaService) detectTor(envData map[string]interface{}) bool {
	if val, ok := envData["tor_detected"].(bool); ok {
		return val
	}
	if val, ok := envData["is_tor_exit"].(bool); ok {
		return val
	}
	return false
}

func (s *InvisibleCaptchaService) detectHosting(envData map[string]interface{}) bool {
	if val, ok := envData["is_hosting"].(bool); ok {
		return val
	}
	return false
}

func (s *InvisibleCaptchaService) getIPReputationScore(ipAddress string) float64 {
	return 75.0
}

type InvisibleVerificationRequest struct {
	SessionID           string                 `json:"session_id"`
	UserID              string                 `json:"user_id"`
	DeviceFingerprint   string                 `json:"device_fingerprint"`
	FingerprintComponents *FingerprintComponents `json:"fingerprint_components,omitempty"`
	BehaviorData        []models.BehaviorData  `json:"behavior_data,omitempty"`
	EnvironmentData     map[string]interface{} `json:"environment_data,omitempty"`
	IPAddress           string                 `json:"ip_address,omitempty"`
	UserAgent           string                 `json:"user_agent,omitempty"`
	Timestamp           time.Time              `json:"timestamp,omitempty"`
}

type InvisibleVerificationResult struct {
	SessionID            string                    `json:"session_id"`
	Timestamp            time.Time                 `json:"timestamp"`

	FingerprintScore     float64                   `json:"fingerprint_score"`
	FingerprintUniqueness float64                  `json:"fingerprint_uniqueness"`
	FingerprintComponents *FingerprintComponents     `json:"fingerprint_components,omitempty"`
	FingerprintConfidence float64                 `json:"fingerprint_confidence"`

	ConfidenceScore      float64                   `json:"confidence_score"`
	ConfidenceFactors    []ConfidenceFactor        `json:"confidence_factors,omitempty"`
	ConfidenceLevel      string                    `json:"confidence_level"`

	BehaviorAnomalyScore float64                  `json:"behavior_anomaly_score"`
	BehaviorPatternMatch float64                  `json:"behavior_pattern_match"`
	BehaviorTrends       map[string]interface{}    `json:"behavior_trends,omitempty"`

	TrustScore           float64                   `json:"trust_score"`
	TrustBreakdown       map[string]float64        `json:"trust_breakdown,omitempty"`
	RiskScore            float64                   `json:"risk_score"`
	RiskFactors          []string                  `json:"risk_factors,omitempty"`

	ShouldChallenge      bool                      `json:"should_challenge"`
	SkipReason           string                    `json:"skip_reason,omitempty"`
	RecommendedAction    string                    `json:"recommended_action,omitempty"`

	HistoricalScore      float64                  `json:"historical_score,omitempty"`
	RecentSuccessRate    float64                  `json:"recent_success_rate,omitempty"`
}

type FingerprintOptimizationResult struct {
	Fingerprint      string                    `json:"fingerprint"`
	Components       *FingerprintComponents     `json:"components,omitempty"`
	StabilityScore   float64                   `json:"stability_score"`
	UniquenessScore  float64                   `json:"uniqueness_score"`
	ConsistencyScore float64                   `json:"consistency_score"`
	QualityScore     float64                   `json:"quality_score"`
}

type ConfidenceAssessmentResult struct {
	BaseScore    float64            `json:"base_score"`
	TotalScore   float64            `json:"total_score"`
	Level        string             `json:"level"`
	Factors      []ConfidenceFactor `json:"factors,omitempty"`
}

type BehaviorAnalysisResult struct {
	UserID            string                   `json:"user_id"`
	AnomalyScore      float64                  `json:"anomaly_score"`
	PatternMatch      float64                  `json:"pattern_match"`
	Trends            map[string]interface{}    `json:"trends,omitempty"`
	HistoricalScore   float64                  `json:"historical_score,omitempty"`
	RecentSuccessRate float64                  `json:"recent_success_rate,omitempty"`
}

type TrustCalculationContext struct {
	UserID               string
	DeviceFingerprint    string
	IsKnownDevice        bool
	DeviceAgeDays        int
	DeviceUseCount       int
	DeviceSuccessRate    float64
	IsKnownLocation      bool
	LocationChangeRate   float64
	IsSuspiciousLocation bool
	BehaviorConsistency  float64
	ResponseTimeScore    float64
	PatternMatchScore    float64
	HistoricalSuccessRate float64
	AccountAgeDays       int
	TotalVerifications   int
	RecentFailureCount   int
	RequestFrequency     float64
	IsProxy              bool
	IsVPN                bool
	IsTor                bool
	IsHosting            bool
	IPReputationScore    float64
	ConfidenceScore      float64
}

type TrustCalculationResult struct {
	TotalScore      float64            `json:"total_score"`
	ComponentScores map[string]float64 `json:"component_scores"`
	RiskScore       float64            `json:"risk_score"`
	RiskFactors     []string           `json:"risk_factors,omitempty"`
}

func (s *InvisibleCaptchaService) RecordVerificationResult(req *InvisibleVerificationRequest, result *InvisibleVerificationResult, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	history := s.behaviorAnalyzer.GetOrCreateHistory(req.UserID)

	record := &VerificationRecord{
		Timestamp:           time.Now(),
		Fingerprint:         req.DeviceFingerprint,
		RiskScore:           result.RiskScore,
		Confidence:          result.ConfidenceScore,
		TrustScore:          result.TrustScore,
		VerificationSuccess: success,
		ChallengeIssued:     result.ShouldChallenge,
		ResponseTimeMs:      0,
		EnvironmentData:     req.EnvironmentData,
	}

	history.VerificationHistory = append(history.VerificationHistory, record)

	deviceFound := false
	for _, device := range history.DeviceHistory {
		if device.Fingerprint == req.DeviceFingerprint {
			device.LastSeen = time.Now()
			device.UseCount++
			deviceFound = true
			if success {
				device.SuccessRate = device.SuccessRate*0.9 + 0.1
			} else {
				device.SuccessRate = device.SuccessRate * 0.9
			}
			break
		}
	}

	if !deviceFound {
		history.DeviceHistory = append(history.DeviceHistory, &DeviceRecord{
			Fingerprint:   req.DeviceFingerprint,
			FirstSeen:     time.Now(),
			LastSeen:      time.Now(),
			UseCount:      1,
			SuccessRate:   0.5,
			AvgRiskScore:  result.RiskScore,
			IsTrusted:     false,
		})
	}

	if req.IPAddress != "" {
		locFound := false
		for _, loc := range history.LocationHistory {
			if loc.Location == req.IPAddress {
				loc.LastSeen = time.Now()
				loc.VisitCount++
				locFound = true
				break
			}
		}

		if !locFound {
			history.LocationHistory = append(history.LocationHistory, &LocationRecord{
				Location:   req.IPAddress,
				FirstSeen:  time.Now(),
				LastSeen:   time.Now(),
				VisitCount: 1,
				IsAnomaly:  false,
			})
		}
	}

	history.LastUpdate = time.Now()
}

func (s *InvisibleCaptchaService) GetUserTrustProfile(userID string) *UserTrustProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := s.behaviorAnalyzer.GetOrCreateHistory(userID)

	profile := &UserTrustProfile{
		UserID:              userID,
		TotalVerifications: len(history.VerificationHistory),
		DeviceCount:        len(history.DeviceHistory),
		LocationCount:       len(history.LocationHistory),
		LastActivity:       history.LastUpdate,
	}

	if len(history.VerificationHistory) > 0 {
		profile.SuccessRate = s.calculateHistoricalScore(history)
		profile.LastVerification = history.VerificationHistory[len(history.VerificationHistory)-1].Timestamp
	}

	trustedDevices := make([]string, 0)
	for _, device := range history.DeviceHistory {
		if device.SuccessRate > 0.8 && device.UseCount > 5 {
			trustedDevices = append(trustedDevices, device.Fingerprint)
		}
	}
	profile.TrustedDevices = trustedDevices

	profile.RiskLevel = "low"
	if profile.SuccessRate < 0.7 {
		profile.RiskLevel = "high"
	} else if profile.SuccessRate < 0.85 {
		profile.RiskLevel = "medium"
	}

	return profile
}

type UserTrustProfile struct {
	UserID              string    `json:"user_id"`
	TotalVerifications  int       `json:"total_verifications"`
	DeviceCount         int       `json:"device_count"`
	LocationCount       int       `json:"location_count"`
	SuccessRate         float64   `json:"success_rate"`
	RiskLevel           string    `json:"risk_level"`
	LastActivity        time.Time `json:"last_activity"`
	LastVerification    time.Time `json:"last_verification,omitempty"`
	TrustedDevices      []string  `json:"trusted_devices,omitempty"`
}

func (s *InvisibleCaptchaService) calculateHistoricalScore(history *UserBehaviorHistory) float64 {
	if len(history.VerificationHistory) == 0 {
		return 0.5
	}

	successCount := 0
	for _, v := range history.VerificationHistory {
		if v.VerificationSuccess {
			successCount++
		}
	}

	return float64(successCount) / float64(len(history.VerificationHistory))
}

func (s *InvisibleCaptchaService) ExportInvisibleCaptchaConfig() string {
	config, _ := json.MarshalIndent(s.config, "", "  ")
	return string(config)
}

func (s *InvisibleCaptchaService) UpdateConfig(config *InvisibleCaptchaConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

func NewInvisibleCaptchaServiceForTest() *InvisibleCaptchaService {
	return NewInvisibleCaptchaService()
}
