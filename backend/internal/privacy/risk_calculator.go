package privacy

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

type RiskCalculator struct {
	config         RiskConfig
	factors        []RiskFactor
	mitigations    map[string]float64
	assetValues    map[string]float64
	threatLevels   map[string]float64
	mu             sync.RWMutex
}

type RiskConfig struct {
	BaseScore       float64
	ImpactWeight    float64
	LikelihoodWeight float64
	VulnerabilityWeight float64
}

type RiskFactor struct {
	Name         string
	Category     RiskCategory
	Weight       float64
	Value        float64
	ContributesTo []RiskCategory
}

type RiskCategory int

const (
	ConfidentialityRisk RiskCategory = iota
	IntegrityRisk
	AvailabilityRisk
	PrivacyRisk
	ComplianceRisk
	ReputationRisk
	FinancialRisk
	OperationalRisk
)

type RiskScore struct {
	OverallScore    float64
	CategoryScores  map[RiskCategory]float64
	FactorScores     map[string]float64
	ConfidenceLevel  float64
	AssessmentDate   time.Time
}

type RiskTreatment struct {
	RiskID          string
	TreatmentOption TreatmentOption
	Cost            float64
	Effectiveness   float64
	Priority        PriorityLevel
	ImplementationDate time.Time
}

type TreatmentOption int

const (
	TreatOptionAvoid TreatmentOption = iota
	TreatOptionMitigate
	TreatOptionTransfer
	TreatOptionAccept
)

func NewRiskCalculator(config RiskConfig) *RiskCalculator {
	return &RiskCalculator{
		config:      config,
		factors:     make([]RiskFactor, 0),
		mitigations: make(map[string]float64),
		assetValues: make(map[string]float64),
		threatLevels: make(map[string]float64),
	}
}

func (rc *RiskCalculator) AddFactor(factor RiskFactor) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.factors = append(rc.factors, factor)
}

func (rc *RiskCalculator) SetMitigation(riskID string, effectiveness float64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.mitigations[riskID] = effectiveness
}

func (rc *RiskCalculator) SetAssetValue(assetID string, value float64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.assetValues[assetID] = value
}

func (rc *RiskCalculator) SetThreatLevel(threatID string, level float64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.threatLevels[threatID] = level
}

func (rc *RiskCalculator) CalculateRisk() RiskScore {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	score := RiskScore{
		CategoryScores: make(map[RiskCategory]float64),
		FactorScores:    make(map[string]float64),
		AssessmentDate: time.Now(),
	}

	for _, factor := range rc.factors {
		factorScore := factor.Weight * factor.Value
		score.FactorScores[factor.Name] = factorScore
	}

	for category := ConfidentialityRisk; category <= OperationalRisk; category++ {
		categoryScore := rc.calculateCategoryScore(category)
		score.CategoryScores[category] = categoryScore
	}

	totalScore := 0.0
	weightSum := 0.0
	for category, catScore := range score.CategoryScores {
		weight := rc.getCategoryWeight(category)
		totalScore += catScore * weight
		weightSum += weight
	}

	if weightSum > 0 {
		score.OverallScore = totalScore / weightSum
	}

	score.ConfidenceLevel = rc.calculateConfidence()

	return score
}

func (rc *RiskCalculator) calculateCategoryScore(category RiskCategory) float64 {
	categoryScore := 0.0
	count := 0

	for _, factor := range rc.factors {
		for _, contrib := range factor.ContributesTo {
			if contrib == category {
				categoryScore += factor.Weight * factor.Value
				count++
				break
			}
		}
	}

	if count > 0 {
		return categoryScore / float64(count)
	}
	return 0.0
}

func (rc *RiskCalculator) getCategoryWeight(category RiskCategory) float64 {
	switch category {
	case PrivacyRisk:
		return rc.config.PrivacyWeight()
	case ConfidentialityRisk:
		return rc.config.ConfidentialityWeight()
	case IntegrityRisk:
		return rc.config.IntegrityWeight()
	case AvailabilityRisk:
		return rc.config.AvailabilityWeight()
	default:
		return 1.0
	}
}

func (rc *RiskConfig) PrivacyWeight() float64 {
	return rc.ImpactWeight
}

func (rc *RiskConfig) ConfidentialityWeight() float64 {
	return rc.ImpactWeight * 0.8
}

func (rc *RiskConfig) IntegrityWeight() float64 {
	return rc.ImpactWeight * 0.7
}

func (rc *RiskConfig) AvailabilityWeight() float64 {
	return rc.ImpactWeight * 0.6
}

func (rc *RiskCalculator) calculateConfidence() float64 {
	if len(rc.factors) == 0 {
		return 0.0
	}

	dataCoverage := math.Min(1.0, float64(len(rc.factors))/10.0)

	threatDataQuality := 0.0
	if len(rc.threatLevels) > 0 {
		threatDataQuality = 0.8
	}

	historicalDataQuality := 0.7

	confidence := (dataCoverage + threatDataQuality + historicalDataQuality) / 3.0
	return confidence * 100
}

func (rc *RiskCalculator) CalculateResidualRisk(originalScore float64) float64 {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	totalMitigation := 0.0
	for _, effectiveness := range rc.mitigations {
		totalMitigation += effectiveness
	}

	if len(rc.mitigations) > 0 {
		avgMitigation := totalMitigation / float64(len(rc.mitigations))
		return originalScore * (1.0 - avgMitigation)
	}

	return originalScore
}

func (rc *RiskCalculator) GenerateTreatmentPlan(riskScore RiskScore) []RiskTreatment {
	treatments := make([]RiskTreatment, 0)

	for category, score := range riskScore.CategoryScores {
		if score > 50 {
			treatment := RiskTreatment{
				RiskID:        category.String(),
				TreatmentOption: TreatOptionMitigate,
				Cost:          rc.estimateTreatmentCost(category),
				Effectiveness: rc.estimateTreatmentEffectiveness(category),
				Priority:      rc.determinePriority(score),
			}
			treatments = append(treatments, treatment)
		}
	}

	return treatments
}

func (rc *RiskCalculator) estimateTreatmentCost(category RiskCategory) float64 {
	switch category {
	case PrivacyRisk:
		return 10000
	case ComplianceRisk:
		return 8000
	case FinancialRisk:
		return 15000
	default:
		return 5000
	}
}

func (rc *RiskCalculator) estimateTreatmentEffectiveness(category RiskCategory) float64 {
	switch category {
	case PrivacyRisk:
		return 0.8
	case ComplianceRisk:
		return 0.9
	case FinancialRisk:
		return 0.7
	default:
		return 0.6
	}
}

func (rc *RiskCalculator) determinePriority(score float64) PriorityLevel {
	if score >= 80 {
		return PriorityCritical
	} else if score >= 60 {
		return PriorityHigh
	} else if score >= 40 {
		return PriorityMedium
	}
	return PriorityLow
}

func (rc *RiskCalculator) GetTopRisks(n int) []RiskFactor {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	scoredFactors := make([]RiskFactor, len(rc.factors))
	copy(scoredFactors, rc.factors)

	for i := 0; i < len(scoredFactors)-1; i++ {
		for j := i + 1; j < len(scoredFactors); j++ {
			if scoredFactors[i].Weight*scoredFactors[i].Value < scoredFactors[j].Weight*scoredFactors[j].Value {
				scoredFactors[i], scoredFactors[j] = scoredFactors[j], scoredFactors[i]
			}
		}
	}

	if n > len(scoredFactors) {
		n = len(scoredFactors)
	}
	return scoredFactors[:n]
}

func (rc *RiskCalculator) MonteCarloSimulation(iterations int) SimulationResult {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	scores := make([]float64, iterations)

	for i := 0; i < iterations; i++ {
		simulatedScore := 0.0
		for _, factor := range rc.factors {
			variation := (rand.Float64() - 0.5) * 0.2 * factor.Value
			simulatedScore += factor.Weight * (factor.Value + variation)
		}
		scores[i] = simulatedScore
	}

	result := SimulationResult{
		Mean:   calculateMean(scores),
		Median: calculateMedian(scores),
		StdDev: calculateStdDev(scores),
		Min:    calculateMin(scores),
		Max:    calculateMax(scores),
	}

	result.Percentile5 = calculatePercentile(scores, 5)
	result.Percentile95 = calculatePercentile(scores, 95)

	return result
}

type SimulationResult struct {
	Mean       float64
	Median     float64
	StdDev     float64
	Min        float64
	Max        float64
	Percentile5  float64
	Percentile95 float64
}

func calculateMean(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateMedian(values []float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	sorted := make([]float64, n)
	copy(sorted, values)

	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func calculateStdDev(values []float64) float64 {
	mean := calculateMean(values)
	sumSq := 0.0
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(values)))
}

func calculateMin(values []float64) float64 {
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func calculateMax(values []float64) float64 {
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func calculatePercentile(values []float64, percentile int) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	sorted := make([]float64, n)
	copy(sorted, values)

	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := (float64(percentile) / 100.0) * float64(n-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func (c RiskCategory) String() string {
	switch c {
	case ConfidentialityRisk:
		return "Confidentiality"
	case IntegrityRisk:
		return "Integrity"
	case AvailabilityRisk:
		return "Availability"
	case PrivacyRisk:
		return "Privacy"
	case ComplianceRisk:
		return "Compliance"
	case ReputationRisk:
		return "Reputation"
	case FinancialRisk:
		return "Financial"
	case OperationalRisk:
		return "Operational"
	default:
		return "Unknown"
	}
}

type RiskDashboard struct {
	CurrentScore    RiskScore
	HistoricalScores []RiskScore
	Trending        TrendDirection
	Recommendations []string
	mu              sync.RWMutex
}

type TrendDirection int

const (
	TrendStable TrendDirection = iota
	TrendIncreasing
	TrendDecreasing
)

func NewRiskDashboard() *RiskDashboard {
	return &RiskDashboard{
		HistoricalScores: make([]RiskScore, 0),
		Trending:         TrendStable,
		Recommendations:  make([]string, 0),
	}
}

func (rd *RiskDashboard) UpdateScore(score RiskScore) {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	rd.CurrentScore = score
	rd.HistoricalScores = append(rd.HistoricalScores, score)
	rd.updateTrend()
	rd.generateRecommendations()
}

func (rd *RiskDashboard) updateTrend() {
	if len(rd.HistoricalScores) < 2 {
		rd.Trending = TrendStable
		return
	}

	recent := rd.HistoricalScores[len(rd.HistoricalScores)-1]
	previous := rd.HistoricalScores[len(rd.HistoricalScores)-2]

	if recent.OverallScore > previous.OverallScore*1.05 {
		rd.Trending = TrendIncreasing
	} else if recent.OverallScore < previous.OverallScore*0.95 {
		rd.Trending = TrendDecreasing
	} else {
		rd.Trending = TrendStable
	}
}

func (rd *RiskDashboard) generateRecommendations() {
	rd.Recommendations = make([]string, 0)

	if rd.CurrentScore.OverallScore > 70 {
		rd.Recommendations = append(rd.Recommendations, "立即采取风险缓解措施")
	}

	if rd.Trending == TrendIncreasing {
		rd.Recommendations = append(rd.Recommendations, "风险呈上升趋势,需要紧急审查")
	}

	for category, score := range rd.CurrentScore.CategoryScores {
		if score > 60 {
			rd.Recommendations = append(rd.Recommendations, "加强"+category.String()+"保护措施")
		}
	}
}
