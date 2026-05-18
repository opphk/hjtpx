package trace

import (
	"encoding/json"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type BehaviorSequenceAnalyzer struct {
	sequencePatterns map[string]*SequencePattern
	transitionMatrix map[string]map[string]float64
	featureExtractor *BehaviorFeatureExtractor
	patternMatcher   *SequencePatternMatcher
}

type SequencePattern struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Pattern     []string   `json:"pattern"`
	Weight      float64    `json:"weight"`
	Description string     `json:"description"`
	RiskLevel   string     `json:"risk_level"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type SequenceMatchResult struct {
	PatternID     string    `json:"pattern_id"`
	PatternName   string    `json:"pattern_name"`
	MatchScore    float64   `json:"match_score"`
	Confidence    float64   `json:"confidence"`
	RiskLevel     string    `json:"risk_level"`
	MatchedEvents []string  `json:"matched_events"`
	StartIndex    int       `json:"start_index"`
	EndIndex      int       `json:"end_index"`
}

type BehaviorFeatureExtractor struct {
}

func NewBehaviorSequenceAnalyzer() *BehaviorSequenceAnalyzer {
	analyzer := &BehaviorSequenceAnalyzer{
		sequencePatterns: make(map[string]*SequencePattern),
		transitionMatrix: make(map[string]map[string]float64),
		featureExtractor: NewBehaviorFeatureExtractor(),
		patternMatcher:   NewSequencePatternMatcher(),
	}
	analyzer.initDefaultPatterns()
	return analyzer
}

func NewBehaviorFeatureExtractor() *BehaviorFeatureExtractor {
	return &BehaviorFeatureExtractor{}
}

func (a *BehaviorSequenceAnalyzer) initDefaultPatterns() {
	patterns := []*SequencePattern{
		{
			ID:          "rapid_login_attempts",
			Name:        "快速登录尝试",
			Pattern:     []string{"page_load", "focus", "input", "submit", "error", "input", "submit"},
			Weight:      0.85,
			Description: "短时间内多次登录尝试失败后再次尝试",
			RiskLevel:   "high",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "automated_form_filling",
			Name:        "自动化表单填写",
			Pattern:     []string{"page_load", "input", "input", "input", "submit"},
			Weight:      0.75,
			Description: "极短时间内完成多个表单字段填写",
			RiskLevel:   "medium",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "bot_like_navigation",
			Name:        "机器人式导航",
			Pattern:     []string{"page_load", "click", "page_load", "click", "page_load"},
			Weight:      0.8,
			Description: "规律性的页面跳转模式",
			RiskLevel:   "high",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "suspicious_scroll",
			Name:        "可疑滚动行为",
			Pattern:     []string{"page_load", "scroll", "scroll", "scroll", "scroll"},
			Weight:      0.6,
			Description: "快速连续滚动",
			RiskLevel:   "low",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "session_hijacking_attempt",
			Name:        "会话劫持尝试",
			Pattern:     []string{"page_load", "cookie_read", "local_storage_read", "api_call", "unauthorized_access"},
			Weight:      0.95,
			Description: "尝试读取敏感存储并进行未授权访问",
			RiskLevel:   "critical",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, p := range patterns {
		a.sequencePatterns[p.ID] = p
	}
}

func (a *BehaviorSequenceAnalyzer) AddPattern(pattern *SequencePattern) error {
	if pattern.ID == "" {
		return errors.New("pattern ID cannot be empty")
	}
	pattern.CreatedAt = time.Now()
	pattern.UpdatedAt = time.Now()
	a.sequencePatterns[pattern.ID] = pattern
	return nil
}

func (a *BehaviorSequenceAnalyzer) RemovePattern(patternID string) error {
	if _, exists := a.sequencePatterns[patternID]; !exists {
		return errors.New("pattern not found")
	}
	delete(a.sequencePatterns, patternID)
	return nil
}

func (a *BehaviorSequenceAnalyzer) GetPattern(patternID string) (*SequencePattern, bool) {
	pattern, exists := a.sequencePatterns[patternID]
	return pattern, exists
}

func (a *BehaviorSequenceAnalyzer) ExtractSequenceFeatures(traceData *model.TraceData) (*SequenceFeatures, error) {
	if traceData == nil || len(traceData.Points) < 2 {
		return nil, errors.New("invalid trace data")
	}

	features := &SequenceFeatures{
		TotalPoints:      len(traceData.Points),
		SequenceLength:   len(traceData.Points),
		TimeDurationMs:   calculateDuration(traceData),
		EventDensity:     calculateEventDensity(traceData),
		VelocityFeatures: extractVelocityFeatures(traceData),
		DirectionChanges: countDirectionChanges(traceData),
	}

	features.PatternComplexity = calculatePatternComplexity(traceData)
	features.Entropy = calculateSequenceEntropy(traceData)
	features.Periodicity = detectPeriodicity(traceData)
	features.Burstiness = calculateBurstiness(traceData)

	return features, nil
}

type SequenceFeatures struct {
	TotalPoints        int                 `json:"total_points"`
	SequenceLength     int                 `json:"sequence_length"`
	TimeDurationMs     int64               `json:"time_duration_ms"`
	EventDensity       float64             `json:"event_density"`
	VelocityFeatures   *VelocityFeatures   `json:"velocity_features"`
	DirectionChanges   int                 `json:"direction_changes"`
	PatternComplexity  float64             `json:"pattern_complexity"`
	Entropy            float64             `json:"entropy"`
	Periodicity        float64             `json:"periodicity"`
	Burstiness         float64             `json:"burstiness"`
}

type VelocityFeatures struct {
	AverageVelocity     float64 `json:"average_velocity"`
	MaxVelocity         float64 `json:"max_velocity"`
	MinVelocity         float64 `json:"min_velocity"`
	VelocityVariance    float64 `json:"velocity_variance"`
	AccelerationChanges int     `json:"acceleration_changes"`
}

func calculateDuration(traceData *model.TraceData) int64 {
	if len(traceData.Points) < 2 {
		return 0
	}
	return traceData.Points[len(traceData.Points)-1].Timestamp - traceData.Points[0].Timestamp
}

func calculateEventDensity(traceData *model.TraceData) float64 {
	duration := calculateDuration(traceData)
	if duration == 0 {
		return 0
	}
	return float64(len(traceData.Points)) / float64(duration) * 1000
}

func extractVelocityFeatures(traceData *model.TraceData) *VelocityFeatures {
	if len(traceData.Points) < 2 {
		return &VelocityFeatures{}
	}

	velocities := []float64{}
	accelerations := []float64{}

	for i := 1; i < len(traceData.Points); i++ {
		dx := float64(traceData.Points[i].X - traceData.Points[i-1].X)
		dy := float64(traceData.Points[i].Y - traceData.Points[i-1].Y)
		dt := float64(traceData.Points[i].Timestamp - traceData.Points[i-1].Timestamp)

		if dt > 0 {
			velocity := math.Sqrt(dx*dx+dy*dy) / dt
			velocities = append(velocities, velocity)
		}
	}

	for i := 1; i < len(velocities); i++ {
		accelerations = append(accelerations, velocities[i]-velocities[i-1])
	}

	avgVelocity := 0.0
	maxVelocity := 0.0
	minVelocity := math.MaxFloat64
	variance := 0.0

	if len(velocities) > 0 {
		sum := 0.0
		for _, v := range velocities {
			sum += v
			if v > maxVelocity {
				maxVelocity = v
			}
			if v < minVelocity {
				minVelocity = v
			}
		}
		avgVelocity = sum / float64(len(velocities))

		for _, v := range velocities {
			variance += math.Pow(v-avgVelocity, 2)
		}
		variance /= float64(len(velocities))
	}

	accelerationChanges := 0
	for i := 1; i < len(accelerations); i++ {
		if math.Abs(accelerations[i]-accelerations[i-1]) > 0.1 {
			accelerationChanges++
		}
	}

	return &VelocityFeatures{
		AverageVelocity:     avgVelocity,
		MaxVelocity:         maxVelocity,
		MinVelocity:         minVelocity,
		VelocityVariance:    variance,
		AccelerationChanges: accelerationChanges,
	}
}

func countDirectionChanges(traceData *model.TraceData) int {
	if len(traceData.Points) < 3 {
		return 0
	}

	changes := 0
	for i := 2; i < len(traceData.Points); i++ {
		dx1 := traceData.Points[i-1].X - traceData.Points[i-2].X
		dy1 := traceData.Points[i-1].Y - traceData.Points[i-2].Y
		dx2 := traceData.Points[i].X - traceData.Points[i-1].X
		dy2 := traceData.Points[i].Y - traceData.Points[i-1].Y

		dot := dx1*dx2 + dy1*dy2
		if dot < 0 {
			changes++
		}
	}

	return changes
}

func calculatePatternComplexity(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 3 {
		return 0
	}

	curvatures := []float64{}
	for i := 1; i < len(traceData.Points)-1; i++ {
		v1x := float64(traceData.Points[i].X - traceData.Points[i-1].X)
		v1y := float64(traceData.Points[i].Y - traceData.Points[i-1].Y)
		v2x := float64(traceData.Points[i+1].X - traceData.Points[i].X)
		v2y := float64(traceData.Points[i+1].Y - traceData.Points[i].Y)

		dot := v1x*v2x + v1y*v2y
		mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
		mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

		if mag1 > 0 && mag2 > 0 {
			cosAngle := dot / (mag1 * mag2)
			if cosAngle > 1 {
				cosAngle = 1
			}
			if cosAngle < -1 {
				cosAngle = -1
			}
			curvatures = append(curvatures, math.Abs(math.Acos(cosAngle)))
		}
	}

	if len(curvatures) == 0 {
		return 0
	}

	sum := 0.0
	for _, c := range curvatures {
		sum += c
	}

	return sum / float64(len(curvatures))
}

func calculateSequenceEntropy(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 2 {
		return 0
	}

	directions := map[string]int{}
	for i := 1; i < len(traceData.Points); i++ {
		dx := traceData.Points[i].X - traceData.Points[i-1].X
		dy := traceData.Points[i].Y - traceData.Points[i-1].Y

		var dir string
		if dx > 5 {
			dir += "R"
		} else if dx < -5 {
			dir += "L"
		}
		if dy > 5 {
			dir += "D"
		} else if dy < -5 {
			dir += "U"
		}

		if dir != "" {
			directions[dir]++
		}
	}

	if len(directions) == 0 {
		return 0
	}

	total := 0
	for _, count := range directions {
		total += count
	}

	entropy := 0.0
	for _, count := range directions {
		p := float64(count) / float64(total)
		entropy -= p * math.Log2(p)
	}

	maxEntropy := math.Log2(float64(len(directions)))
	if maxEntropy > 0 {
		entropy /= maxEntropy
	}

	return entropy
}

func detectPeriodicity(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 10 {
		return 0
	}

	timeIntervals := []float64{}
	for i := 1; i < len(traceData.Points); i++ {
		timeIntervals = append(timeIntervals, float64(traceData.Points[i].Timestamp-traceData.Points[i-1].Timestamp))
	}

	if len(timeIntervals) < 5 {
		return 0
	}

	mean := 0.0
	for _, t := range timeIntervals {
		mean += t
	}
	mean /= float64(len(timeIntervals))

	stdDev := 0.0
	for _, t := range timeIntervals {
		stdDev += math.Pow(t-mean, 2)
	}
	stdDev = math.Sqrt(stdDev / float64(len(timeIntervals)))

	if mean == 0 {
		return 0
	}

	cv := stdDev / mean
	if cv < 0.15 {
		return 1.0 - cv*2
	}

	return 0
}

func calculateBurstiness(traceData *model.TraceData) float64 {
	if len(traceData.Points) < 5 {
		return 0
	}

	timeIntervals := []float64{}
	for i := 1; i < len(traceData.Points); i++ {
		timeIntervals = append(timeIntervals, float64(traceData.Points[i].Timestamp-traceData.Points[i-1].Timestamp))
	}

	sort.Float64s(timeIntervals)

	shortIntervals := timeIntervals[:len(timeIntervals)/4]
	longIntervals := timeIntervals[3*len(timeIntervals)/4:]

	avgShort := 0.0
	for _, t := range shortIntervals {
		avgShort += t
	}
	if len(shortIntervals) > 0 {
		avgShort /= float64(len(shortIntervals))
	}

	avgLong := 0.0
	for _, t := range longIntervals {
		avgLong += t
	}
	if len(longIntervals) > 0 {
		avgLong /= float64(len(longIntervals))
	}

	if avgLong == 0 {
		return 0
	}

	return 1.0 - avgShort/avgLong
}

func (a *BehaviorSequenceAnalyzer) AnalyzeSequence(traceData *model.TraceData) (*SequenceAnalysisResult, error) {
	features, err := a.ExtractSequenceFeatures(traceData)
	if err != nil {
		return nil, err
	}

	matches := a.patternMatcher.MatchPatterns(traceData, a.sequencePatterns)

	result := &SequenceAnalysisResult{
		Features:     features,
		PatternMatches: matches,
		RiskScore:    calculateSequenceRisk(features, matches),
		IsSuspicious: isSequenceSuspicious(features, matches),
	}

	return result, nil
}

type SequenceAnalysisResult struct {
	Features        *SequenceFeatures   `json:"features"`
	PatternMatches  []*SequenceMatchResult `json:"pattern_matches"`
	RiskScore       float64             `json:"risk_score"`
	IsSuspicious    bool                `json:"is_suspicious"`
}

func calculateSequenceRisk(features *SequenceFeatures, matches []*SequenceMatchResult) float64 {
	riskScore := 0.0

	if features.Burstiness > 0.8 {
		riskScore += 15
	}

	if features.Periodicity > 0.8 {
		riskScore += 20
	}

	if features.Entropy < 0.3 {
		riskScore += 15
	}

	if features.VelocityFeatures.VelocityVariance < 0.01 {
		riskScore += 20
	}

	for _, match := range matches {
		riskScore += match.MatchScore * match.Confidence * 10
	}

	return math.Min(riskScore, 100)
}

func isSequenceSuspicious(features *SequenceFeatures, matches []*SequenceMatchResult) bool {
	if features.Periodicity > 0.9 {
		return true
	}

	if features.VelocityFeatures.VelocityVariance < 0.005 {
		return true
	}

	for _, match := range matches {
		if match.RiskLevel == "critical" && match.Confidence > 0.7 {
			return true
		}
		if match.RiskLevel == "high" && match.Confidence > 0.85 {
			return true
		}
	}

	return false
}

type SequencePatternMatcher struct {
}

func NewSequencePatternMatcher() *SequencePatternMatcher {
	return &SequencePatternMatcher{}
}

func (m *SequencePatternMatcher) MatchPatterns(traceData *model.TraceData, patterns map[string]*SequencePattern) []*SequenceMatchResult {
	if traceData == nil || len(traceData.Points) < 2 {
		return []*SequenceMatchResult{}
	}

	results := []*SequenceMatchResult{}

	for _, pattern := range patterns {
		result := m.matchPattern(traceData, pattern)
		if result.MatchScore > 0.6 {
			results = append(results, result)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].MatchScore*results[i].Confidence > results[j].MatchScore*results[j].Confidence
	})

	return results
}

func (m *SequencePatternMatcher) matchPattern(traceData *model.TraceData, pattern *SequencePattern) *SequenceMatchResult {
	result := &SequenceMatchResult{
		PatternID:   pattern.ID,
		PatternName: pattern.Name,
		RiskLevel:   pattern.RiskLevel,
	}

	pointEvents := extractPointEvents(traceData)
	patternEvents := pattern.Pattern

	if len(pointEvents) < len(patternEvents) {
		return result
	}

	bestScore := 0.0
	bestConfidence := 0.0
	bestStart := 0
	bestEnd := 0

	for i := 0; i <= len(pointEvents)-len(patternEvents); i++ {
		score, confidence := m.calculateMatchScore(pointEvents[i:i+len(patternEvents)], patternEvents)
		if score > bestScore || (score == bestScore && confidence > bestConfidence) {
			bestScore = score
			bestConfidence = confidence
			bestStart = i
			bestEnd = i + len(patternEvents)
		}
	}

	result.MatchScore = bestScore
	result.Confidence = bestConfidence
	result.StartIndex = bestStart
	result.EndIndex = bestEnd
	result.MatchedEvents = pointEvents[bestStart:bestEnd]

	return result
}

func extractPointEvents(traceData *model.TraceData) []string {
	events := []string{}
	for _, point := range traceData.Points {
		events = append(events, classifyPointEvent(point))
	}
	return events
}

func classifyPointEvent(point model.TracePoint) string {
	if point.X == 0 && point.Y == 0 {
		return "page_load"
	}
	return "move"
}

func (m *SequencePatternMatcher) calculateMatchScore(events, pattern []string) (float64, float64) {
	if len(events) != len(pattern) {
		return 0, 0
	}

	matches := 0
	for i := range events {
		if events[i] == pattern[i] {
			matches++
		}
	}

	score := float64(matches) / float64(len(pattern))
	confidence := score

	return score, confidence
}

func (a *BehaviorSequenceAnalyzer) UpdateTransitionMatrix(traceData *model.TraceData) {
	if traceData == nil || len(traceData.Points) < 2 {
		return
	}

	events := extractPointEvents(traceData)

	for i := 1; i < len(events); i++ {
		from := events[i-1]
		to := events[i]

		if _, exists := a.transitionMatrix[from]; !exists {
			a.transitionMatrix[from] = make(map[string]float64)
		}
		a.transitionMatrix[from][to]++
	}
}

func (a *BehaviorSequenceAnalyzer) GetTransitionMatrix() map[string]map[string]float64 {
	return a.transitionMatrix
}

func (a *BehaviorSequenceAnalyzer) DetectAnomalousTransitions(traceData *model.TraceData) []string {
	anomalies := []string{}

	if len(a.transitionMatrix) == 0 {
		return anomalies
	}

	events := extractPointEvents(traceData)

	for i := 1; i < len(events); i++ {
		from := events[i-1]
		to := events[i]

		if transitions, exists := a.transitionMatrix[from]; exists {
			total := 0.0
			for _, count := range transitions {
				total += count
			}

			if count, exists := transitions[to]; exists {
				probability := count / total
				if probability < 0.01 {
					anomalies = append(anomalies, from+"->"+to)
				}
			}
		}
	}

	return anomalies
}

func (a *BehaviorSequenceAnalyzer) ExportPatterns() ([]byte, error) {
	return json.MarshalIndent(a.sequencePatterns, "", "  ")
}

func (a *BehaviorSequenceAnalyzer) ImportPatterns(data []byte) error {
	var patterns map[string]*SequencePattern
	if err := json.Unmarshal(data, &patterns); err != nil {
		return err
	}

	for id, pattern := range patterns {
		pattern.ID = id
		pattern.UpdatedAt = time.Now()
		a.sequencePatterns[id] = pattern
	}

	return nil
}

func (a *BehaviorSequenceAnalyzer) GetAllPatterns() []*SequencePattern {
	patterns := make([]*SequencePattern, 0, len(a.sequencePatterns))
	for _, p := range a.sequencePatterns {
		patterns = append(patterns, p)
	}
	return patterns
}

func (a *BehaviorSequenceAnalyzer) TrainOnSequence(traceData *model.TraceData, isBot bool) {
	features, _ := a.ExtractSequenceFeatures(traceData)

	for _, pattern := range a.sequencePatterns {
		if isBot {
			pattern.Weight = math.Min(1.0, pattern.Weight+0.01)
		} else {
			pattern.Weight = math.Max(0.0, pattern.Weight-0.005)
		}
		pattern.UpdatedAt = time.Now()
	}

	_ = features
}