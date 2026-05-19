package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type AdaptiveVerificationEngine struct {
	mu                    sync.RWMutex
	riskAssessor          *RealtimeRiskAssessor
	strategyEngine        *DynamicStrategyEngine
	personalizationEngine *PersonalizationEngine
	learningEngine        *SelfLearningEngine
	initialized           bool
}

type RealtimeRiskAssessor struct {
	mu          sync.RWMutex
	thresholds  RiskThresholds
	featureWeights map[string]float64
	historySize int
	history     []RiskSnapshot
}

type RiskThresholds struct {
	Critical float64
	High     float64
	Medium   float64
	Low      float64
}

type RiskSnapshot struct {
	Timestamp time.Time
	Score     float64
	Level     string
	Factors   []string
}

type DynamicStrategyEngine struct {
	mu           sync.RWMutex
	strategies   map[string]*VerificationStrategy
	currentLevel int
	adaptationRate float64
}

type VerificationStrategy struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Difficulty        int                    `json:"difficulty"`
	Timeout           time.Duration          `json:"timeout"`
	MaxAttempts       int                    `json:"max_attempts"`
	RequiredFactors   []string               `json:"required_factors"`
	ScoringWeights    map[string]float64    `json:"scoring_weights"`
	SuccessCriteria   SuccessCriteria        `json:"success_criteria"`
	FallbackEnabled   bool                   `json:"fallback_enabled"`
}

type SuccessCriteria struct {
	MinScore      float64 `json:"min_score"`
	MaxTime       int64   `json:"max_time"`
	MaxErrors     int     `json:"max_errors"`
	RequiredPattern bool  `json:"required_pattern"`
}

type PersonalizationEngine struct {
	mu          sync.RWMutex
	userProfiles map[string]*UserProfile
	modelCache   map[string]*PersonalizationModel
}

type UserProfile struct {
	UserID            string                 `json:"user_id"`
	InteractionStyle   string                 `json:"interaction_style"`
	PreferredCaptcha   string                 `json:"preferred_captcha"`
	DifficultyHistory  []int                  `json:"difficulty_history"`
	SuccessRate        float64                `json:"success_rate"`
	AvgCompletionTime  time.Duration          `json:"avg_completion_time"`
	FailurePatterns    []string               `json:"failure_patterns"`
	LastInteraction    time.Time              `json:"last_interaction"`
	TrustScore         float64                `json:"trust_score"`
	BehavioralFeatures map[string]float64    `json:"behavioral_features"`
}

type PersonalizationModel struct {
	UserID        string             `json:"user_id"`
	Features      []float64          `json:"features"`
	Preferences   map[string]float64 `json:"preferences"`
	AdaptationRate float64           `json:"adaptation_rate"`
	LastUpdate    time.Time          `json:"last_update"`
}

type SelfLearningEngine struct {
	mu              sync.RWMutex
	modelVersion    string
	trainingData    []TrainingSample
	modelParameters map[string]float64
	feedbackBuffer  []FeedbackEntry
	updateInterval  time.Duration
	lastUpdate      time.Time
	learningRate    float64
}

type TrainingSample struct {
	Features   []float64 `json:"features"`
	Label     bool      `json:"label"`
	Timestamp time.Time `json:"timestamp"`
	Context   map[string]interface{} `json:"context"`
}

type FeedbackEntry struct {
	SampleID   string                 `json:"sample_id"`
	IsCorrect  bool                   `json:"is_correct"`
	Confidence float64                `json:"confidence"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type AdaptiveRiskResult struct {
	RiskScore       float64                  `json:"risk_score"`
	RiskLevel       string                   `json:"risk_level"`
	Confidence      float64                  `json:"confidence"`
	ContributingFactors []string            `json:"contributing_factors"`
	Recommendations []string                `json:"recommendations"`
	ProcessingTime  time.Duration            `json:"processing_time"`
}

type AdaptiveStrategyResult struct {
	Strategy      *VerificationStrategy `json:"strategy"`
	RecommendedDifficulty int           `json:"recommended_difficulty"`
	AdjustedTimeout       time.Duration `json:"adjusted_timeout"`
	SpecialInstructions    []string      `json:"special_instructions"`
	Reasoning              string        `json:"reasoning"`
}

type PersonalizationResult struct {
	PreferredCaptcha   string                 `json:"preferred_captcha"`
	OptimalDifficulty int                    `json:"optimal_difficulty"`
	CustomInstructions []string               `json:"custom_instructions"`
	TrustAdjustment    float64                `json:"trust_adjustment"`
	Confidence         float64                `json:"confidence"`
}

type LearningUpdateResult struct {
	UpdatedSamples int     `json:"updated_samples"`
	ModelVersion   string  `json:"model_version"`
	AccuracyImprovement float64 `json:"accuracy_improvement"`
	NewPatternsDiscovered int   `json:"new_patterns_discovered"`
}

func NewAdaptiveVerificationEngine() *AdaptiveVerificationEngine {
	return &AdaptiveVerificationEngine{
		riskAssessor:          NewRealtimeRiskAssessor(),
		strategyEngine:        NewDynamicStrategyEngine(),
		personalizationEngine: NewPersonalizationEngine(),
		learningEngine:        NewSelfLearningEngine(),
	}
}

func (e *AdaptiveVerificationEngine) Initialize(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if e.initialized {
		return nil
	}
	
	if err := e.riskAssessor.Initialize(ctx); err != nil {
		return err
	}
	
	if err := e.strategyEngine.Initialize(ctx); err != nil {
		return err
	}
	
	if err := e.personalizationEngine.Initialize(ctx); err != nil {
		return err
	}
	
	if err := e.learningEngine.Initialize(ctx); err != nil {
		return err
	}
	
	e.initialized = true
	return nil
}

func NewRealtimeRiskAssessor() *RealtimeRiskAssessor {
	return &RealtimeRiskAssessor{
		thresholds: RiskThresholds{
			Critical: 0.85,
			High:     0.70,
			Medium:   0.50,
			Low:      0.30,
		},
		featureWeights: map[string]float64{
			"mouse_velocity":      0.15,
			"click_timing":        0.12,
			"scroll_pattern":      0.10,
			"keyboard_rhythm":     0.08,
			"touch_pressure":      0.10,
			"device_fingerprint":  0.15,
			"session_behavior":    0.12,
			"network_pattern":     0.08,
			"time_distribution":   0.05,
			"error_pattern":       0.05,
		},
		historySize: 100,
		history:     make([]RiskSnapshot, 0),
	}
}

func (a *RealtimeRiskAssessor) Initialize(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return nil
}

func (a *RealtimeRiskAssessor) AssessRisk(ctx context.Context, context *model.RiskContext, traceData *model.TraceData) (*AdaptiveRiskResult, error) {
	start := time.Now()
	
	features := a.extractRiskFeatures(context, traceData)
	
	rawScore := a.computeWeightedScore(features)
	
	temporalScore := a.applyTemporalAdjustment(rawScore)
	
	contextualScore := a.applyContextualAdjustment(temporalScore, context)
	
	finalScore := math.Min(math.Max(contextualScore, 0), 100)
	
	level := a.determineRiskLevel(finalScore)
	factors := a.identifyContributingFactors(features)
	recommendations := a.generateRecommendations(finalScore, level, factors)
	
	snapshot := RiskSnapshot{
		Timestamp: time.Now(),
		Score:     finalScore,
		Level:     level,
		Factors:   factors,
	}
	a.addSnapshot(snapshot)
	
	return &AdaptiveRiskResult{
		RiskScore:          finalScore,
		RiskLevel:          level,
		Confidence:         a.calculateConfidence(features),
		ContributingFactors: factors,
		Recommendations:    recommendations,
		ProcessingTime:     time.Since(start),
	}, nil
}

func (a *RealtimeRiskAssessor) extractRiskFeatures(ctx *model.RiskContext, traceData *model.TraceData) map[string]float64 {
	features := make(map[string]float64)
	
	if traceData != nil && len(traceData.Points) > 0 {
		features["mouse_velocity"] = a.calculateMouseVelocity(traceData)
		features["click_timing"] = a.calculateClickTiming(traceData)
		features["scroll_pattern"] = a.calculateScrollPattern(traceData)
		features["touch_pressure"] = a.calculateTouchPressure(traceData)
	} else {
		features["mouse_velocity"] = 0.5
		features["click_timing"] = 0.5
		features["scroll_pattern"] = 0.5
		features["touch_pressure"] = 0.5
	}
	
	if ctx != nil {
		features["device_fingerprint"] = a.evaluateDeviceFingerprint(ctx)
		features["session_behavior"] = a.evaluateSessionBehavior(ctx)
		features["network_pattern"] = a.evaluateNetworkPattern(ctx)
		features["time_distribution"] = a.evaluateTimeDistribution(ctx)
		features["error_pattern"] = a.evaluateErrorPattern(ctx)
	} else {
		features["device_fingerprint"] = 0.5
		features["session_behavior"] = 0.5
		features["network_pattern"] = 0.5
		features["time_distribution"] = 0.5
		features["error_pattern"] = 0.5
	}
	
	features["keyboard_rhythm"] = 0.5
	
	return features
}

func (a *RealtimeRiskAssessor) calculateMouseVelocity(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 0.5
	}
	
	var totalVelocity float64
	var count int
	
	for i := 1; i < len(traceData.Points); i++ {
		dx := traceData.Points[i].X - traceData.Points[i-1].X
		dy := traceData.Points[i].Y - traceData.Points[i-1].Y
		dist := math.Sqrt(dx*dx + dy*dy)
		dt := float64(traceData.Points[i].Timestamp - traceData.Points[i-1].Timestamp)
		
		if dt > 0 {
			velocity := dist / dt
			totalVelocity += velocity
			count++
		}
	}
	
	if count == 0 {
		return 0.5
	}
	
	avgVelocity := totalVelocity / float64(count)
	
	if avgVelocity > 5 {
		return 0.9
	} else if avgVelocity > 2 {
		return 0.6
	} else if avgVelocity > 0.5 {
		return 0.3
	}
	return 0.1
}

func (a *RealtimeRiskAssessor) calculateClickTiming(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 0.5
	}
	
	var intervals []float64
	for i := 1; i < len(traceData.Points); i++ {
		if traceData.Points[i].Event == "click" {
			interval := float64(traceData.Points[i].Timestamp - traceData.Points[i-1].Timestamp)
			intervals = append(intervals, interval)
		}
	}
	
	if len(intervals) < 2 {
		return 0.5
	}
	
	variance := a.calculateVariance(intervals)
	
	if variance < 10 {
		return 0.9
	} else if variance < 100 {
		return 0.5
	}
	return 0.2
}

func (a *RealtimeRiskAssessor) calculateScrollPattern(traceData *model.TraceData) float64 {
	if len(traceData.ScrollData) == 0 {
		return 0.5
	}
	
	avgVelocity := 0.0
	for _, scroll := range traceData.ScrollData {
		avgVelocity += scroll.Velocity
	}
	avgVelocity /= float64(len(traceData.ScrollData))
	
	regularity := 0.5
	if len(traceData.ScrollData) > 1 {
		regularity = a.calculateVarianceOfVelocities(traceData.ScrollData)
	}
	
	return (avgVelocity/100.0 + regularity) / 2
}

func (a *RealtimeRiskAssessor) calculateTouchPressure(traceData *model.TraceData) float64 {
	if len(traceData.Points) == 0 {
		return 0.5
	}
	
	var totalPressure float64
	var count int
	
	for _, point := range traceData.Points {
		if point.Pressure > 0 {
			totalPressure += point.Pressure
			count++
		}
	}
	
	if count == 0 {
		return 0.5
	}
	
	avgPressure := totalPressure / float64(count)
	
	if avgPressure > 0.8 || avgPressure < 0.2 {
		return 0.3
	}
	return 0.6
}

func (a *RealtimeRiskAssessor) evaluateDeviceFingerprint(ctx *model.RiskContext) float64 {
	if ctx == nil {
		return 0.5
	}
	
	score := 0.5
	
	if ctx.IsProxy || ctx.IsVPN || ctx.IsTor {
		score += 0.3
	}
	
	if ctx.Fingerprint != "" && len(ctx.Fingerprint) > 32 {
		score += 0.1
	}
	
	if len(ctx.BrowserPlugins) > 0 {
		score += 0.1
	}
	
	return math.Min(score, 1.0)
}

func (a *RealtimeRiskAssessor) evaluateSessionBehavior(ctx *model.RiskContext) float64 {
	if ctx == nil {
		return 0.5
	}
	
	score := 0.5
	
	if ctx.FailureCount > 3 {
		score += 0.3
	} else if ctx.FailureCount > 1 {
		score += 0.1
	}
	
	if ctx.VerificationCount > 10 {
		score -= 0.2
	}
	
	return math.Min(math.Max(score, 0), 1.0)
}

func (a *RealtimeRiskAssessor) evaluateNetworkPattern(ctx *model.RiskContext) float64 {
	if ctx == nil || ctx.IPReputation == "" {
		return 0.5
	}
	
	switch ctx.IPReputation {
	case "clean":
		return 0.2
	case "suspicious":
		return 0.6
	case "malicious":
		return 0.9
	default:
		return 0.5
	}
}

func (a *RealtimeRiskAssessor) evaluateTimeDistribution(ctx *model.RiskContext) float64 {
	if ctx == nil {
		return 0.5
	}
	
	hour := time.Now().Hour()
	
	if hour >= 2 && hour <= 5 {
		return 0.7
	}
	
	return 0.3
}

func (a *RealtimeRiskAssessor) evaluateErrorPattern(ctx *model.RiskContext) float64 {
	if ctx == nil || ctx.FailureCount == 0 {
		return 0.3
	}
	
	if ctx.FailureCount > 5 {
		return 0.9
	} else if ctx.FailureCount > 3 {
		return 0.7
	} else if ctx.FailureCount > 1 {
		return 0.5
	}
	return 0.4
}

func (a *RealtimeRiskAssessor) computeWeightedScore(features map[string]float64) float64 {
	var totalScore float64
	var totalWeight float64
	
	for feature, value := range features {
		weight := a.featureWeights[feature]
		if weight == 0 {
			weight = 0.1
		}
		totalScore += value * weight
		totalWeight += weight
	}
	
	if totalWeight == 0 {
		return 50.0
	}
	
	return (totalScore / totalWeight) * 100
}

func (a *RealtimeRiskAssessor) applyTemporalAdjustment(score float64) float64 {
	if len(a.history) == 0 {
		return score
	}
	
	var recentAvg float64
	recentCount := math.Min(10, float64(len(a.history)))
	
	for i := len(a.history) - int(recentCount); i < len(a.history); i++ {
		if i >= 0 {
			recentAvg += a.history[i].Score
		}
	}
	recentAvg /= recentCount
	
	trendWeight := 0.2
	return score*(1-trendWeight) + recentAvg*trendWeight
}

func (a *RealtimeRiskAssessor) applyContextualAdjustment(score float64, ctx *model.RiskContext) float64 {
	if ctx == nil {
		return score
	}
	
	if ctx.IsProxy || ctx.IsVPN || ctx.IsTor {
		score = math.Min(score*1.2, 100)
	}
	
	if ctx.HasHighRiskIndicators() {
		score = math.Min(score*1.1, 100)
	}
	
	return score
}

func (a *RealtimeRiskAssessor) determineRiskLevel(score float64) string {
	switch {
	case score >= a.thresholds.Critical*100:
		return "critical"
	case score >= a.thresholds.High*100:
		return "high"
	case score >= a.thresholds.Medium*100:
		return "medium"
	case score >= a.thresholds.Low*100:
		return "low"
	default:
		return "minimal"
	}
}

func (a *RealtimeRiskAssessor) identifyContributingFactors(features map[string]float64) []string {
	var factors []string
	
	threshold := 0.7
	for feature, value := range features {
		if value >= threshold {
			switch feature {
			case "mouse_velocity":
				factors = append(factors, "异常鼠标移动速度")
			case "click_timing":
				factors = append(factors, "规律的点击时序")
			case "scroll_pattern":
				factors = append(factors, "异常的滚动行为")
			case "device_fingerprint":
				factors = append(factors, "可疑的设备指纹")
			case "session_behavior":
				factors = append(factors, "异常的会话行为")
			case "network_pattern":
				factors = append(factors, "可疑的网络模式")
			case "time_distribution":
				factors = append(factors, "非正常访问时间")
			case "error_pattern":
				factors = append(factors, "高失败率")
			}
		}
	}
	
	return factors
}

func (a *RealtimeRiskAssessor) generateRecommendations(score float64, level string, factors []string) []string {
	var recommendations []string
	
	switch level {
	case "critical", "high":
		recommendations = append(recommendations, "建议启用增强验证")
		recommendations = append(recommendations, "考虑添加多因素认证")
	case "medium":
		recommendations = append(recommendations, "建议增加验证难度")
	}
	
	if len(factors) > 3 {
		recommendations = append(recommendations, "检测到多种异常行为，建议人工审核")
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "当前行为正常")
	}
	
	return recommendations
}

func (a *RealtimeRiskAssessor) calculateConfidence(features map[string]float64) float64 {
	dataPoints := len(features)
	
	completeness := float64(dataPoints) / 10.0
	
	variance := a.calculateVarianceOfMapValues(features)
	consistency := 1.0 - math.Min(variance, 1.0)
	
	confidence := (completeness*0.6 + consistency*0.4) * 100
	
	return math.Min(math.Max(confidence, 0), 100)
}

func (a *RealtimeRiskAssessor) calculateVariance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values) - 1)
	
	return variance
}

func (a *RealtimeRiskAssessor) calculateVarianceOfVelocities(scrolls []model.ScrollInfo) float64 {
	if len(scrolls) < 2 {
		return 0
	}
	
	velocities := make([]float64, len(scrolls))
	for i, s := range scrolls {
		velocities[i] = s.Velocity
	}
	
	return a.calculateVariance(velocities)
}

func (a *RealtimeRiskAssessor) calculateVarianceOfMapValues(m map[string]float64) float64 {
	if len(m) < 2 {
		return 0
	}
	
	values := make([]float64, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	
	return a.calculateVariance(values)
}

func (a *RealtimeRiskAssessor) addSnapshot(snapshot RiskSnapshot) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.history = append(a.history, snapshot)
	
	if len(a.history) > a.historySize {
		a.history = a.history[len(a.history)-a.historySize:]
	}
}

func NewDynamicStrategyEngine() *DynamicStrategyEngine {
	return &DynamicStrategyEngine{
		strategies: map[string]*VerificationStrategy{
			"minimal": {
				ID:              "minimal",
				Name:            "Minimal Verification",
				Difficulty:      1,
				Timeout:         30 * time.Second,
				MaxAttempts:     3,
				RequiredFactors: []string{"basic"},
				ScoringWeights:  map[string]float64{"basic": 1.0},
				SuccessCriteria: SuccessCriteria{MinScore: 50, MaxTime: 30000, MaxErrors: 2},
				FallbackEnabled: false,
			},
			"standard": {
				ID:              "standard",
				Name:            "Standard Verification",
				Difficulty:      2,
				Timeout:         45 * time.Second,
				MaxAttempts:     3,
				RequiredFactors: []string{"basic", "behavior"},
				ScoringWeights:  map[string]float64{"basic": 0.6, "behavior": 0.4},
				SuccessCriteria: SuccessCriteria{MinScore: 65, MaxTime: 45000, MaxErrors: 1},
				FallbackEnabled: true,
			},
			"enhanced": {
				ID:              "enhanced",
				Name:            "Enhanced Verification",
				Difficulty:      3,
				Timeout:         60 * time.Second,
				MaxAttempts:     2,
				RequiredFactors: []string{"basic", "behavior", "device"},
				ScoringWeights:  map[string]float64{"basic": 0.4, "behavior": 0.35, "device": 0.25},
				SuccessCriteria: SuccessCriteria{MinScore: 75, MaxTime: 60000, MaxErrors: 1},
				FallbackEnabled: true,
			},
			"critical": {
				ID:              "critical",
				Name:            "Critical Verification",
				Difficulty:      4,
				Timeout:         90 * time.Second,
				MaxAttempts:     2,
				RequiredFactors: []string{"basic", "behavior", "device", "contextual"},
				ScoringWeights:  map[string]float64{"basic": 0.3, "behavior": 0.3, "device": 0.2, "contextual": 0.2},
				SuccessCriteria: SuccessCriteria{MinScore: 85, MaxTime: 90000, MaxErrors: 0, RequiredPattern: true},
				FallbackEnabled: false,
			},
		},
		currentLevel:    2,
		adaptationRate: 0.1,
	}
}

func (e *DynamicStrategyEngine) Initialize(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return nil
}

func (e *DynamicStrategyEngine) DetermineStrategy(ctx context.Context, riskResult *AdaptiveRiskResult, userProfile *UserProfile) (*AdaptiveStrategyResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	strategyKey := e.selectStrategyBasedOnRisk(riskResult)
	
	if userProfile != nil && userProfile.TrustScore > 0.8 {
		strategyKey = e.adjustStrategyForTrust(strategyKey, userProfile.TrustScore)
	}
	
	strategy := e.strategies[strategyKey]
	if strategy == nil {
		strategy = e.strategies["standard"]
	}
	
	adjustedStrategy := e.adjustStrategyForContext(strategy, riskResult)
	
	result := &AdaptiveStrategyResult{
		Strategy:              adjustedStrategy,
		RecommendedDifficulty: adjustedStrategy.Difficulty,
		AdjustedTimeout:       adjustedStrategy.Timeout,
		SpecialInstructions:   e.generateSpecialInstructions(adjustedStrategy, riskResult),
		Reasoning:             e.generateReasoning(adjustedStrategy, riskResult, userProfile),
	}
	
	return result, nil
}

func (e *DynamicStrategyEngine) selectStrategyBasedOnRisk(riskResult *AdaptiveRiskResult) string {
	switch riskResult.RiskLevel {
	case "critical":
		return "critical"
	case "high":
		return "enhanced"
	case "medium":
		return "standard"
	default:
		return "minimal"
	}
}

func (e *DynamicStrategyEngine) adjustStrategyForTrust(baseStrategy string, trustScore float64) string {
	if trustScore > 0.9 {
		return "minimal"
	} else if trustScore > 0.8 {
		return "standard"
	}
	return baseStrategy
}

func (e *DynamicStrategyEngine) adjustStrategyForContext(strategy *VerificationStrategy, riskResult *AdaptiveRiskResult) *VerificationStrategy {
	adjusted := *strategy
	
	if riskResult.RiskScore > 80 {
		adjusted.Timeout = time.Duration(float64(adjusted.Timeout) * 0.8)
		adjusted.MaxAttempts = math.Max(1, adjusted.MaxAttempts-1)
	} else if riskResult.RiskScore < 30 {
		adjusted.Timeout = time.Duration(float64(adjusted.Timeout) * 1.2)
	}
	
	return &adjusted
}

func (e *DynamicStrategyEngine) generateSpecialInstructions(strategy *VerificationStrategy, riskResult *AdaptiveRiskResult) []string {
	var instructions []string
	
	if strategy.Difficulty >= 3 {
		instructions = append(instructions, "请注意验证时间限制")
		instructions = append(instructions, "仔细观察图像细节")
	}
	
	if len(riskResult.ContributingFactors) > 2 {
		instructions = append(instructions, "系统检测到异常行为，请按正常方式完成验证")
	}
	
	return instructions
}

func (e *DynamicStrategyEngine) generateReasoning(strategy *VerificationStrategy, riskResult *AdaptiveRiskResult, userProfile *UserProfile) string {
	reasoning := fmt.Sprintf("基于风险评分 %.2f (级别: %s) 和置信度 %.2f%%",
		riskResult.RiskScore, riskResult.RiskLevel, riskResult.Confidence)
	
	if userProfile != nil {
		reasoning += fmt.Sprintf("，用户信任评分 %.2f", userProfile.TrustScore)
	}
	
	reasoning += fmt.Sprintf("，选择策略: %s (难度等级: %d)",
		strategy.Name, strategy.Difficulty)
	
	return reasoning
}

func NewPersonalizationEngine() *PersonalizationEngine {
	return &PersonalizationEngine{
		userProfiles: make(map[string]*UserProfile),
		modelCache:   make(map[string]*PersonalizationModel),
	}
}

func (e *PersonalizationEngine) Initialize(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return nil
}

func (e *PersonalizationEngine) GetPersonalization(ctx context.Context, userID string, riskResult *AdaptiveRiskResult) (*PersonalizationResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	profile := e.userProfiles[userID]
	
	if profile == nil {
		profile = e.createDefaultProfile(userID)
	}
	
	model := e.modelCache[userID]
	if model == nil {
		model = e.createDefaultModel(userID)
	}
	
	result := &PersonalizationResult{
		PreferredCaptcha:   profile.PreferredCaptcha,
		OptimalDifficulty:  e.calculateOptimalDifficulty(profile, riskResult),
		CustomInstructions: e.generateCustomInstructions(profile),
		TrustAdjustment:    e.calculateTrustAdjustment(profile),
		Confidence:         model.AdaptationRate * 100,
	}
	
	return result, nil
}

func (e *PersonalizationEngine) UpdateProfile(ctx context.Context, userID string, interaction *InteractionData) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	profile, exists := e.userProfiles[userID]
	if !exists {
		profile = e.createDefaultProfile(userID)
		e.userProfiles[userID] = profile
	}
	
	profile.LastInteraction = time.Now()
	
	if interaction.Success {
		profile.SuccessRate = profile.SuccessRate*0.9 + 0.1
	} else {
		profile.SuccessRate = profile.SuccessRate*0.9
		if interaction.ErrorType != "" {
			profile.FailurePatterns = append(profile.FailurePatterns, interaction.ErrorType)
			if len(profile.FailurePatterns) > 10 {
				profile.FailurePatterns = profile.FailurePatterns[len(profile.FailurePatterns)-10:]
			}
		}
	}
	
	profile.DifficultyHistory = append(profile.DifficultyHistory, interaction.Difficulty)
	if len(profile.DifficultyHistory) > 20 {
		profile.DifficultyHistory = profile.DifficultyHistory[len(profile.DifficultyHistory)-20:]
	}
	
	if interaction.CompletionTime > 0 {
		totalTime := profile.AvgCompletionTime * time.Duration(len(profile.DifficultyHistory)-1)
		profile.AvgCompletionTime = (totalTime + interaction.CompletionTime) / time.Duration(len(profile.DifficultyHistory))
	}
	
	e.updateModelCache(userID, profile)
	
	return nil
}

type InteractionData struct {
	UserID         string
	Difficulty     int
	Success        bool
	CompletionTime time.Duration
	ErrorType      string
	CaptchaType    string
}

func (e *PersonalizationEngine) createDefaultProfile(userID string) *UserProfile {
	return &UserProfile{
		UserID:           userID,
		InteractionStyle: "normal",
		PreferredCaptcha: "slider",
		DifficultyHistory: make([]int, 0),
		SuccessRate:      0.7,
		AvgCompletionTime: 5 * time.Second,
		FailurePatterns:  make([]string, 0),
		LastInteraction:  time.Now(),
		TrustScore:       0.5,
		BehavioralFeatures: make(map[string]float64),
	}
}

func (e *PersonalizationEngine) createDefaultModel(userID string) *PersonalizationModel {
	return &PersonalizationModel{
		UserID:         userID,
		Features:       make([]float64, 32),
		Preferences:    map[string]float64{"slider": 0.4, "click": 0.3, "rotate": 0.2, "puzzle": 0.1},
		AdaptationRate: 0.3,
		LastUpdate:     time.Now(),
	}
}

func (e *PersonalizationEngine) calculateOptimalDifficulty(profile *UserProfile, riskResult *AdaptiveRiskResult) int {
	baseDifficulty := 2
	
	avgDifficulty := 0.0
	if len(profile.DifficultyHistory) > 0 {
		for _, d := range profile.DifficultyHistory {
			avgDifficulty += float64(d)
		}
		avgDifficulty /= float64(len(profile.DifficultyHistory))
	}
	
	difficultyAdjust := int(avgDifficulty * 0.5)
	
	riskAdjust := 0
	if riskResult.RiskScore > 70 {
		riskAdjust = 1
	} else if riskResult.RiskScore < 30 {
		riskAdjust = -1
	}
	
	optimal := baseDifficulty + difficultyAdjust + riskAdjust
	
	if optimal < 1 {
		optimal = 1
	}
	if optimal > 5 {
		optimal = 5
	}
	
	return optimal
}

func (e *PersonalizationEngine) generateCustomInstructions(profile *UserProfile) []string {
	var instructions []string
	
	if profile.SuccessRate < 0.5 {
		instructions = append(instructions, "请仔细阅读验证提示")
		instructions = append(instructions, "按照提示要求完成操作")
	}
	
	if len(profile.FailurePatterns) > 0 {
		lastFailure := profile.FailurePatterns[len(profile.FailurePatterns)-1]
		switch lastFailure {
		case "timeout":
			instructions = append(instructions, "注意控制操作时间")
		case "wrong_answer":
			instructions = append(instructions, "请仔细确认选择是否正确")
		}
	}
	
	return instructions
}

func (e *PersonalizationEngine) calculateTrustAdjustment(profile *UserProfile) float64 {
	if len(profile.DifficultyHistory) < 5 {
		return 0
	}
	
	improvement := profile.SuccessRate - 0.7
	
	return improvement * 0.2
}

func (e *PersonalizationEngine) updateModelCache(userID string, profile *UserProfile) {
	model := &PersonalizationModel{
		UserID:          userID,
		AdaptationRate:  math.Min(0.9, profile.TrustScore),
		LastUpdate:      time.Now(),
	}
	
	e.modelCache[userID] = model
}

func NewSelfLearningEngine() *SelfLearningEngine {
	return &SelfLearningEngine{
		modelVersion:    "v1.0",
		trainingData:    make([]TrainingSample, 0),
		modelParameters: make(map[string]float64),
		feedbackBuffer:  make([]FeedbackEntry, 0),
		updateInterval:  1 * time.Hour,
		learningRate:    0.01,
	}
}

func (e *SelfLearningEngine) Initialize(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.modelParameters = map[string]float64{
		"velocity_weight":     0.15,
		"timing_weight":       0.12,
		"pattern_weight":      0.10,
		"context_weight":      0.13,
		"threshold_adjust":    0.0,
	}
	
	return nil
}

func (e *SelfLearningEngine) RecordSample(ctx context.Context, features []float64, isHuman bool, context map[string]interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	sample := TrainingSample{
		Features:   features,
		Label:      isHuman,
		Timestamp:  time.Now(),
		Context:    context,
	}
	
	e.trainingData = append(e.trainingData, sample)
	
	if len(e.trainingData) > 10000 {
		e.trainingData = e.trainingData[len(e.trainingData)-10000:]
	}
	
	return nil
}

func (e *SelfLearningEngine) RecordFeedback(ctx context.Context, sampleID string, isCorrect bool, confidence float64, metadata map[string]interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	entry := FeedbackEntry{
		SampleID:   sampleID,
		IsCorrect:  isCorrect,
		Confidence: confidence,
		Timestamp:  time.Now(),
		Metadata:   metadata,
	}
	
	e.feedbackBuffer = append(e.feedbackBuffer, entry)
	
	if len(e.feedbackBuffer) > 1000 {
		e.applyLearning()
	}
	
	return nil
}

func (e *SelfLearningEngine) UpdateModel(ctx context.Context) (*LearningUpdateResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	previousAccuracy := e.calculateCurrentAccuracy()
	
	e.applyLearning()
	
	previousModelVersion := e.modelVersion
	e.modelVersion = fmt.Sprintf("v1.%d", len(e.trainingData)/100)
	
	newAccuracy := e.calculateCurrentAccuracy()
	
	result := &LearningUpdateResult{
		UpdatedSamples:       len(e.trainingData),
		ModelVersion:         e.modelVersion,
		AccuracyImprovement:  newAccuracy - previousAccuracy,
		NewPatternsDiscovered: e.detectNewPatterns(),
	}
	
	e.lastUpdate = time.Now()
	
	return result, nil
}

func (e *SelfLearningEngine) applyLearning() {
	if len(e.feedbackBuffer) == 0 {
		return
	}
	
	positiveCount := 0
	negativeCount := 0
	
	for _, feedback := range e.feedbackBuffer {
		if feedback.IsCorrect {
			positiveCount++
		} else {
			negativeCount++
		}
	}
	
	if positiveCount > negativeCount {
		for key := range e.modelParameters {
			e.modelParameters[key] += e.learningRate * 0.1
		}
	} else if negativeCount > positiveCount {
		for key := range e.modelParameters {
			e.modelParameters[key] -= e.learningRate * 0.1
		}
	}
	
	e.feedbackBuffer = make([]FeedbackEntry, 0)
}

func (e *SelfLearningEngine) calculateCurrentAccuracy() float64 {
	if len(e.trainingData) == 0 {
		return 0.7
	}
	
	correct := 0
	for _, sample := range e.trainingData[len(e.trainingData)-100:] {
		prediction := e.predictSample(sample.Features)
		if (prediction > 0.5) == sample.Label {
			correct++
		}
	}
	
	return float64(correct) / 100.0
}

func (e *SelfLearningEngine) predictSample(features []float64) float64 {
	if len(features) == 0 || len(e.modelParameters) == 0 {
		return 0.5
	}
	
	score := 0.0
	for i, f := range features {
		if i < 4 {
			key := fmt.Sprintf("feature_%d_weight", i)
			weight := e.modelParameters[key]
			if weight == 0 {
				weight = 0.25
			}
			score += f * weight
		}
	}
	
	return 1.0 / (1.0 + math.Exp(-score))
}

func (e *SelfLearningEngine) detectNewPatterns() int {
	if len(e.trainingData) < 100 {
		return 0
	}
	
	recentSamples := e.trainingData[len(e.trainingData)-100:]
	
	var humanCount, botCount int
	for _, s := range recentSamples {
		if s.Label {
			humanCount++
		} else {
			botCount++
		}
	}
	
	patterns := 0
	
	if humanCount > 80 {
		patterns++
	}
	
	if botCount > 20 {
		patterns++
	}
	
	return patterns
}

func (e *AdaptiveVerificationEngine) PerformAdaptiveAssessment(ctx context.Context, riskCtx *model.RiskContext, traceData *model.TraceData, userID string) (map[string]interface{}, error) {
	if !e.initialized {
		return nil, fmt.Errorf("engine not initialized")
	}
	
	riskResult, err := e.riskAssessor.AssessRisk(ctx, riskCtx, traceData)
	if err != nil {
		return nil, err
	}
	
	var userProfile *UserProfile
	if userID != "" {
		e.personalizationEngine.mu.RLock()
		userProfile = e.personalizationEngine.userProfiles[userID]
		e.personalizationEngine.mu.RUnlock()
	}
	
	strategyResult, err := e.strategyEngine.DetermineStrategy(ctx, riskResult, userProfile)
	if err != nil {
		return nil, err
	}
	
	personalizationResult, err := e.personalizationEngine.GetPersonalization(ctx, userID, riskResult)
	if err != nil {
		return nil, err
	}
	
	result := map[string]interface{}{
		"risk_assessment":  riskResult,
		"strategy":         strategyResult,
		"personalization":  personalizationResult,
		"recommended_action": e.determineAction(riskResult, strategyResult),
		"timestamp":        time.Now(),
	}
	
	return result, nil
}

func (e *AdaptiveVerificationEngine) determineAction(riskResult *AdaptiveRiskResult, strategyResult *AdaptiveStrategyResult) string {
	if riskResult.RiskLevel == "critical" {
		return "block"
	}
	
	if riskResult.RiskLevel == "high" {
		return "challenge"
	}
	
	if strategyResult.Strategy.Difficulty >= 3 {
		return "verify"
	}
	
	return "allow"
}

func (e *AdaptiveVerificationEngine) RecordInteraction(ctx context.Context, interaction *InteractionData) error {
	if !e.initialized {
		return fmt.Errorf("engine not initialized")
	}
	
	if err := e.personalizationEngine.UpdateProfile(ctx, interaction.UserID, interaction); err != nil {
		return err
	}
	
	features := e.extractFeaturesFromInteraction(interaction)
	isHuman := interaction.Success
	
	if err := e.learningEngine.RecordSample(ctx, features, isHuman, nil); err != nil {
		return err
	}
	
	return nil
}

func (e *AdaptiveVerificationEngine) extractFeaturesFromInteraction(interaction *InteractionData) []float64 {
	return []float64{
		float64(interaction.Difficulty),
		interaction.CompletionTime.Seconds(),
		0.5,
		0.5,
	}
}

func (e *AdaptiveVerificationEngine) TriggerModelUpdate(ctx context.Context) (*LearningUpdateResult, error) {
	if !e.initialized {
		return nil, fmt.Errorf("engine not initialized")
	}
	
	return e.learningEngine.UpdateModel(ctx)
}

func (e *AdaptiveVerificationEngine) GetRiskHistory(ctx context.Context) ([]RiskSnapshot, error) {
	e.riskAssessor.mu.RLock()
	defer e.riskAssessor.mu.RUnlock()
	
	snapshots := make([]RiskSnapshot, len(e.riskAssessor.history))
	copy(snapshots, e.riskAssessor.history)
	
	return snapshots, nil
}

type AdaptiveAssessmentRequest struct {
	UserID     string            `json:"user_id"`
	RiskContext *model.RiskContext `json:"risk_context"`
	TraceData  *model.TraceData   `json:"trace_data"`
}

type AdaptiveAssessmentResponse struct {
	Success           bool                   `json:"success"`
	RiskAssessment    *AdaptiveRiskResult    `json:"risk_assessment"`
	Strategy          *AdaptiveStrategyResult `json:"strategy"`
	Personalization   *PersonalizationResult `json:"personalization"`
	RecommendedAction string                 `json:"recommended_action"`
}

type RecordInteractionRequest struct {
	UserID         string `json:"user_id" binding:"required"`
	Difficulty     int    `json:"difficulty" binding:"required"`
	Success        bool   `json:"success" binding:"required"`
	CompletionTime int64  `json:"completion_time"`
	ErrorType      string `json:"error_type"`
	CaptchaType    string `json:"captcha_type"`
}

type RecordInteractionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ModelUpdateResponse struct {
	Success bool                  `json:"success"`
	Result  *LearningUpdateResult `json:"result"`
}

func ParseAdaptiveRequest(data string) (*AdaptiveAssessmentRequest, error) {
	var req AdaptiveAssessmentRequest
	err := json.Unmarshal([]byte(data), &req)
	return &req, err
}
