package trace

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type DynamicThresholdConfig struct {
	MinThreshold   float64
	MaxThreshold   float64
	AdaptationRate float64
	UpdateInterval time.Duration
}

type RiskLevel string

const (
	RiskLevelMinimal RiskLevel = "minimal"
	RiskLevelLow    RiskLevel = "low"
	RiskLevelMedium RiskLevel = "medium"
	RiskLevelHigh   RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type UnifiedRiskScorer struct {
	traceExtractor        *TraceExtractor
	lstmExtractor         *LSTMFeatureExtractor
	transformerPredictor *TransformerPredictor
	patternLibrary       *BehaviorPatternLibrary
	thresholdConfig      *DynamicThresholdConfig
	dynamicThresholds    map[string]float64
	mu                   sync.RWMutex
	lastUpdateTime       time.Time
	historyScores        []float64
	maxHistorySize       int
}

type ComprehensiveRiskResult struct {
	TotalRiskScore        float64            `json:"total_risk_score"`
	BotProbability       float64            `json:"bot_probability"`
	HumanProbability     float64            `json:"human_probability"`
	Confidence           float64            `json:"confidence"`
	RiskLevel            RiskLevel          `json:"risk_level"`
	IsBot                bool               `json:"is_bot"`
	ComponentScores      map[string]float64 `json:"component_scores"`
	DetectedPatterns     []string           `json:"detected_patterns"`
	Recommendations      []string           `json:"recommendations"`
	AnalysisTimestamp    time.Time          `json:"analysis_timestamp"`
	ProcessingTimeMs     int64              `json:"processing_time_ms"`
	FeatureCounts        map[string]int     `json:"feature_counts"`
	IntentRecognition    *IntentRecognitionResult `json:"intent_recognition,omitempty"`
	AnomalyPatterns      []AnomalyPattern   `json:"anomaly_patterns,omitempty"`
}

func NewUnifiedRiskScorer() *UnifiedRiskScorer {
	scorer := &UnifiedRiskScorer{
		traceExtractor:        NewTraceExtractor(),
		lstmExtractor:         NewLSTMFeatureExtractor(),
		transformerPredictor:  NewTransformerPredictor(),
		patternLibrary:       NewBehaviorPatternLibrary(),
		thresholdConfig:      &DynamicThresholdConfig{
			MinThreshold:   0.2,
			MaxThreshold:   0.8,
			AdaptationRate: 0.1,
			UpdateInterval: time.Minute * 5,
		},
		dynamicThresholds: make(map[string]float64),
		historyScores:     make([]float64, 0),
		maxHistorySize:    1000,
	}

	scorer.initializeThresholds()

	return scorer
}

func (s *UnifiedRiskScorer) initializeThresholds() {
	s.dynamicThresholds["speed_anomaly"] = 0.7
	s.dynamicThresholds["acceleration_anomaly"] = 0.6
	s.dynamicThresholds["trajectory_anomaly"] = 0.65
	s.dynamicThresholds["pattern_anomaly"] = 0.75
	s.dynamicThresholds["lstm_anomaly"] = 0.6
	s.dynamicThresholds["transformer_anomaly"] = 0.65
}

func (s *UnifiedRiskScorer) AnalyzeComprehensiveRisk(ctx context.Context, traceData *model.TraceData) (*ComprehensiveRiskResult, error) {
	if traceData == nil || len(traceData.Points) < 2 {
		return nil, errors.New("轨迹数据不足")
	}

	startTime := time.Now()

	result := &ComprehensiveRiskResult{
		ComponentScores:   make(map[string]float64),
		DetectedPatterns: []string{},
		Recommendations:  []string{},
		AnalysisTimestamp: startTime,
	}

	basicFeatures, err := s.traceExtractor.ExtractFeatures(traceData)
	if err == nil && basicFeatures != nil {
		result.ComponentScores["basic_risk_score"] = s.calculateBasicRiskScore(basicFeatures)
		result.FeatureCounts = map[string]int{
			"total_distance": int(basicFeatures.TotalDistance),
			"move_count":     basicFeatures.MoveCount,
			"pause_count":    basicFeatures.PauseCount,
			"total_time":     int(basicFeatures.TotalTime),
		}
	}

	advancedFeatures, err := s.traceExtractor.ExtractAdvancedFeatures(traceData)
	if err == nil && advancedFeatures != nil {
		result.ComponentScores["advanced_risk_score"] = s.calculateAdvancedRiskScore(advancedFeatures)
	}

	lstmFeatures, err := s.lstmExtractor.ExtractRiskFeatures(traceData)
	if err == nil && lstmFeatures != nil {
		result.ComponentScores["lstm_risk_score"] = s.calculateLSTMRiskScore(lstmFeatures)
	}

	transformerResult, err := s.transformerPredictor.PredictTrajectory(traceData)
	if err == nil && transformerResult != nil {
		result.ComponentScores["transformer_risk_score"] = transformerResult.RiskScore
		result.ComponentScores["transformer_confidence"] = transformerResult.Confidence
		result.BotProbability = transformerResult.BotProbability
		result.HumanProbability = transformerResult.HumanProbability
		result.Confidence = transformerResult.Confidence
	}

	patternResult := s.patternLibrary.AnalyzeComprehensiveRisk(traceData, basicFeatures, advancedFeatures)
	if patternResult != nil {
		result.ComponentScores["pattern_risk_score"] = patternResult.CombinedRiskScore
		result.ComponentScores["bot_pattern_score"] = patternResult.BotPatternScore
		result.ComponentScores["speed_anomaly_score"] = patternResult.SpeedAnomalyScore
		result.ComponentScores["movement_anomaly_score"] = patternResult.MovementAnomalyScore
		result.DetectedPatterns = append(result.DetectedPatterns, patternResult.DetectedPatterns...)
	}

	result.TotalRiskScore = s.calculateTotalRiskScore(result.ComponentScores)
	result.RiskLevel = s.determineRiskLevel(result.TotalRiskScore)
	result.IsBot = s.determineIsBot(result)

	result.Recommendations = s.generateRecommendations(result)

	result.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	s.updateHistory(result.TotalRiskScore)
	s.adaptThresholds()

	return result, nil
}

func (s *UnifiedRiskScorer) calculateBasicRiskScore(features *model.TraceFeatures) float64 {
	score := 0.0

	if features.SpeedVariance < 10 && features.TotalTime > 1000 {
		score += 0.2
	}

	if features.AvgSpeed > 500 {
		score += 0.15
	}

	if features.AvgSpeed < 10 && features.TotalTime > 5000 {
		score += 0.1
	}

	if features.PauseCount == 0 && features.TotalTime > 2000 {
		score += 0.2
	}

	if features.PathRatio < 1.1 {
		score += 0.25
	}

	if features.MaxAcceleration > 3000 {
		score += 0.15
	}

	if features.Smoothness < 0.05 {
		score += 0.2
	}

	if features.TotalDistance < 10 && features.TotalTime > 1000 {
		score += 0.15
	}

	return math.Min(score, 1.0)
}

func (s *UnifiedRiskScorer) calculateAdvancedRiskScore(features *AdvancedFeatures) float64 {
	score := 0.0

	if features.SpeedVariance < 5 {
		score += 0.15
	}

	if features.SpeedEntropy < 1.5 {
		score += 0.1
	}

	if features.AccelerationVariance < 0.001 {
		score += 0.15
	}

	if features.CurvatureVariance < 0.001 {
		score += 0.1
	}

	if features.DirectionEntropy < 1.0 {
		score += 0.1
	}

	if features.Sinuosity < 1.05 {
		score += 0.2
	}

	if features.JerkMean < 0.01 {
		score += 0.1
	}

	return math.Min(score, 1.0)
}

func (s *UnifiedRiskScorer) calculateLSTMRiskScore(features map[string]float64) float64 {
	score := 0.0

	if val, ok := features["velocity_variance"]; ok && val < 0.1 {
		score += 0.15
	}

	if val, ok := features["acceleration_mean"]; ok && val < 0.01 {
		score += 0.1
	}

	if val, ok := features["direction_change_rate"]; ok && val < 0.1 {
		score += 0.15
	}

	if val, ok := features["curvature_mean"]; ok && val < 0.05 {
		score += 0.1
	}

	if val, ok := features["pause_ratio"]; ok && val < 0.05 {
		score += 0.2
	}

	if val, ok := features["speed_entropy"]; ok && val < 1.5 {
		score += 0.15
	}

	if val, ok := features["sinuosity"]; ok && val < 1.05 {
		score += 0.2
	}

	return math.Min(score, 1.0)
}

func (s *UnifiedRiskScorer) calculateTotalRiskScore(componentScores map[string]float64) float64 {
	weights := map[string]float64{
		"basic_risk_score":     0.15,
		"advanced_risk_score":  0.15,
		"lstm_risk_score":      0.25,
		"transformer_risk_score": 0.30,
		"pattern_risk_score":   0.15,
	}

	var totalScore, totalWeight float64

	for name, weight := range weights {
		if score, ok := componentScores[name]; ok {
			totalScore += score * weight
			totalWeight += weight
		}
	}

	if totalWeight > 0 {
		return math.Min(totalScore/totalWeight, 1.0)
	}

	return 0.5
}

func (s *UnifiedRiskScorer) determineRiskLevel(score float64) RiskLevel {
	s.mu.RLock()
	highThreshold := s.dynamicThresholds["high_risk"]
	mediumThreshold := s.dynamicThresholds["medium_risk"]
	lowThreshold := s.dynamicThresholds["low_risk"]
	s.mu.RUnlock()

	if highThreshold == 0 {
		highThreshold = 0.7
	}
	if mediumThreshold == 0 {
		mediumThreshold = 0.5
	}
	if lowThreshold == 0 {
		lowThreshold = 0.3
	}

	switch {
	case score >= highThreshold:
		return RiskLevelCritical
	case score >= mediumThreshold:
		return RiskLevelHigh
	case score >= lowThreshold:
		return RiskLevelMedium
	case score >= 0.15:
		return RiskLevelLow
	default:
		return RiskLevelMinimal
	}
}

func (s *UnifiedRiskScorer) determineIsBot(result *ComprehensiveRiskResult) bool {
	botScore := 0.0

	if result.TotalRiskScore >= 0.6 {
		botScore += 0.4
	}

	if result.BotProbability >= 0.7 {
		botScore += 0.3
	}

	if len(result.DetectedPatterns) >= 3 {
		botScore += 0.2
	}

	for _, pattern := range result.DetectedPatterns {
		if containsBotIndicator(pattern) {
			botScore += 0.1
		}
	}

	return botScore >= 0.5
}

func containsBotIndicator(pattern string) bool {
	indicators := []string{
		"机器特征",
		"恒定",
		"过于",
		"异常",
		"重复",
		"完美",
		"机械",
	}

	for _, indicator := range indicators {
		if contains(pattern, indicator) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (s *UnifiedRiskScorer) generateRecommendations(result *ComprehensiveRiskResult) []string {
	recommendations := []string{}

	switch result.RiskLevel {
	case RiskLevelCritical:
		recommendations = append(recommendations, "强烈建议阻止此请求并记录日志")
		recommendations = append(recommendations, "考虑加入黑名单")
	case RiskLevelHigh:
		recommendations = append(recommendations, "建议增加验证难度")
		recommendations = append(recommendations, "可能需要人工审核")
	case RiskLevelMedium:
		recommendations = append(recommendations, "建议启用额外验证")
		recommendations = append(recommendations, "监控后续行为")
	case RiskLevelLow:
		recommendations = append(recommendations, "基本通过，可正常处理")
	case RiskLevelMinimal:
		recommendations = append(recommendations, "完全通过")
	}

	for _, pattern := range result.DetectedPatterns {
		if contains(pattern, "速度") {
			recommendations = append(recommendations, "检测到异常速度模式，建议深入分析")
		}
		if contains(pattern, "轨迹") {
			recommendations = append(recommendations, "检测到异常轨迹模式")
		}
	}

	return recommendations
}

func (s *UnifiedRiskScorer) updateHistory(score float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.historyScores = append(s.historyScores, score)

	if len(s.historyScores) > s.maxHistorySize {
		s.historyScores = s.historyScores[1:]
	}

	s.lastUpdateTime = time.Now()
}

func (s *UnifiedRiskScorer) adaptThresholds() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.historyScores) < 10 {
		return
	}

	avgRecent := s.calculateRecentAverage(100)

	variance := s.calculateVariance(s.historyScores)

	if variance < 0.01 {
		adjustment := s.thresholdConfig.AdaptationRate * (avgRecent - 0.5)
		s.dynamicThresholds["high_risk"] = math.Max(0.6, math.Min(0.8, 0.7+adjustment))
		s.dynamicThresholds["medium_risk"] = math.Max(0.4, math.Min(0.6, 0.5+adjustment))
		s.dynamicThresholds["low_risk"] = math.Max(0.2, math.Min(0.4, 0.3+adjustment))
	}
}

func (s *UnifiedRiskScorer) calculateRecentAverage(window int) float64 {
	if len(s.historyScores) == 0 {
		return 0.5
	}

	start := len(s.historyScores) - window
	if start < 0 {
		start = 0
	}

	sum := 0.0
	for i := start; i < len(s.historyScores); i++ {
		sum += s.historyScores[i]
	}

	return sum / float64(len(s.historyScores)-start)
}

func (s *UnifiedRiskScorer) calculateVariance(scores []float64) float64 {
	if len(scores) < 2 {
		return 0
	}

	mean := 0.0
	for _, score := range scores {
		mean += score
	}
	mean /= float64(len(scores))

	variance := 0.0
	for _, score := range scores {
		diff := score - mean
		variance += diff * diff
	}

	return variance / float64(len(scores))
}

func (s *UnifiedRiskScorer) GetThresholds() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	thresholds := make(map[string]float64)
	for k, v := range s.dynamicThresholds {
		thresholds[k] = v
	}

	return thresholds
}

func (s *UnifiedRiskScorer) UpdateThreshold(name string, value float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if value < s.thresholdConfig.MinThreshold || value > s.thresholdConfig.MaxThreshold {
		return errors.New("阈值超出允许范围")
	}

	s.dynamicThresholds[name] = value
	return nil
}

func (s *UnifiedRiskScorer) GetStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	if len(s.historyScores) > 0 {
		sum := 0.0
		for _, score := range s.historyScores {
			sum += score
		}
		stats["average_score"] = sum / float64(len(s.historyScores))
		stats["score_count"] = len(s.historyScores)
	}

	stats["thresholds"] = s.GetThresholds()
	stats["last_update"] = s.lastUpdateTime

	return stats
}

func (s *UnifiedRiskScorer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.historyScores = make([]float64, 0)
	s.initializeThresholds()
	s.lastUpdateTime = time.Time{}
}

func (s *UnifiedRiskScorer) BatchAnalyze(ctx context.Context, traces []*model.TraceData) ([]*ComprehensiveRiskResult, error) {
	results := make([]*ComprehensiveRiskResult, 0, len(traces))

	for _, trace := range traces {
		result, err := s.AnalyzeComprehensiveRisk(ctx, trace)
		if err != nil {
			continue
		}
		results = append(results, result)
	}

	return results, nil
}
