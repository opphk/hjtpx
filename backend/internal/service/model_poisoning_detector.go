package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

var (
	ErrPoisoningDetectionFailed = errors.New("poisoning detection failed")
	ErrInvalidTrainingData     = errors.New("invalid training data")
	ErrModelNotFound          = errors.New("model not found")
)

type PoisoningAttackType string

const (
	PoisoningLabelFlip   PoisoningAttackType = "label_flip"
	PoisoningBackdoor   PoisoningAttackType = "backdoor"
	PoisoningSemantic   PoisoningAttackType = "semantic"
	PoisoningCleanLabel PoisoningAttackType = "clean_label"
)

type PoisoningSeverity string

const (
	SeverityLow    PoisoningSeverity = "low"
	SeverityMedium PoisoningSeverity = "medium"
	SeverityHigh   PoisoningSeverity = "high"
	SeverityCritical PoisoningSeverity = "critical"
)

type PoisoningSample struct {
	SampleID     string                 `json:"sample_id"`
	DataPoint    []float64              `json:"data_point"`
	Label        int                   `json:"label"`
	IsPoisoned   bool                  `json:"is_poisoned"`
	PoisonType   PoisoningAttackType    `json:"poison_type"`
	Severity     float64               `json:"severity"`
	Confidence   float64               `json:"confidence"`
	Features     map[string]float64    `json:"features"`
	DetectedAt   time.Time             `json:"detected_at"`
}

type PoisoningDetectionReport struct {
	ModelID           string                `json:"model_id"`
	IsPoisoned       bool                  `json:"is_poisoned"`
	PoisonedSamples  []*PoisoningSample   `json:"poisoned_samples"`
	PoisoningRatio   float64               `json:"poisoning_ratio"`
	AttackType       PoisoningAttackType   `json:"attack_type"`
	OverallSeverity  PoisoningSeverity     `json:"overall_severity"`
	Recommendations  []string              `json:"recommendations"`
	DetectionMethods []string              `json:"detection_methods"`
	AnalysisTime     time.Duration          `json:"analysis_time"`
}

type TrainingDataAnalysis struct {
	TotalSamples     int                   `json:"total_samples"`
	PoisonedSamples  int                   `json:"poisoned_samples"`
	NormalSamples    int                   `json:"normal_samples"`
	AnomalyScore     float64               `json:"anomaly_score"`
	LabelDistribution map[int]int          `json:"label_distribution"`
	FeatureStats     map[string]float64    `json:"feature_stats"`
}

type ModelPoisoningDetectorService struct {
	mu            sync.RWMutex
	models        map[string]*ModelInfo
	detectors     map[PoisoningAttackType]*Detector
	reports       map[string]*PoisoningDetectionReport
}

type ModelInfo struct {
	ModelID       string
	TrainingData  [][]float64
	Labels       []int
	Metadata     map[string]interface{}
	UploadedAt   time.Time
}

type Detector struct {
	AttackType     PoisoningAttackType
	Name           string
	Description    string
	Sensitivity   float64
	IsActive      bool
}

func NewModelPoisoningDetectorService() *ModelPoisoningDetectorService {
	return &ModelPoisoningDetectorService{
		models:    make(map[string]*ModelInfo),
		detectors: make(map[PoisoningAttackType]*Detector),
		reports:   make(map[string]*PoisoningDetectionReport),
	}
}

func (s *ModelPoisoningDetectorService) Initialize() {
	s.detectors[PoisoningLabelFlip] = &Detector{
		AttackType:   PoisoningLabelFlip,
		Name:        "Label Flip Detector",
		Description: "Detects instances where labels have been intentionally flipped",
		Sensitivity: 0.75,
		IsActive:    true,
	}

	s.detectors[PoisoningBackdoor] = &Detector{
		AttackType:   PoisoningBackdoor,
		Name:        "Backdoor Detector",
		Description: "Detects backdoor patterns in training data",
		Sensitivity: 0.80,
		IsActive:    true,
	}

	s.detectors[PoisoningSemantic] = &Detector{
		AttackType:   PoisoningSemantic,
		Name:        "Semantic Anomaly Detector",
		Description: "Detects semantically inconsistent samples",
		Sensitivity: 0.70,
		IsActive:    true,
	}

	s.detectors[PoisoningCleanLabel] = &Detector{
		AttackType:   PoisoningCleanLabel,
		Name:        "Clean Label Attack Detector",
		Description: "Detects clean-label attacks where poisoned samples appear legitimate",
		Sensitivity: 0.65,
		IsActive:    true,
	}
}

func (s *ModelPoisoningDetectorService) RegisterModel(ctx context.Context, modelID string, trainingData [][]float64, labels []int, metadata map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(trainingData) != len(labels) {
		return ErrInvalidTrainingData
	}

	s.models[modelID] = &ModelInfo{
		ModelID:      modelID,
		TrainingData: trainingData,
		Labels:       labels,
		Metadata:     metadata,
		UploadedAt:   time.Now(),
	}

	return nil
}

func (s *ModelPoisoningDetectorService) AnalyzeTrainingData(ctx context.Context, modelID string) (*TrainingDataAnalysis, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	model, exists := s.models[modelID]
	if !exists {
		return nil, ErrModelNotFound
	}

	analysis := &TrainingDataAnalysis{
		TotalSamples:     len(model.TrainingData),
		PoisonedSamples: 0,
		NormalSamples:   len(model.TrainingData),
		LabelDistribution: make(map[int]int),
		FeatureStats:    make(map[string]float64),
	}

	for _, label := range model.Labels {
		analysis.LabelDistribution[label]++
	}

	for i, data := range model.TrainingData {
		poisonScore := s.computePoisoningScore(data, model.Labels[i])
		if poisonScore > 0.6 {
			analysis.PoisonedSamples++
			analysis.NormalSamples--
		}
	}

	analysis.AnomalyScore = float64(analysis.PoisonedSamples) / float64(analysis.TotalSamples)

	return analysis, nil
}

func (s *ModelPoisoningDetectorService) DetectPoisoning(ctx context.Context, modelID string) (*PoisoningDetectionReport, error) {
	start := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	model, exists := s.models[modelID]
	if !exists {
		return nil, ErrModelNotFound
	}

	report := &PoisoningDetectionReport{
		ModelID:           modelID,
		PoisonedSamples:   make([]*PoisoningSample, 0),
		DetectionMethods:  make([]string, 0),
		Recommendations:   make([]string, 0),
		AnalysisTime:      time.Since(start),
	}

	attackScores := make(map[PoisoningAttackType]float64)

	for attackType, detector := range s.detectors {
		if detector.IsActive {
			report.DetectionMethods = append(report.DetectionMethods, detector.Name)
			attackScores[attackType] = s.detectAttackType(model.TrainingData, model.Labels, attackType)
		}
	}

	maxScore := 0.0
	var dominantAttack PoisoningAttackType

	for attackType, score := range attackScores {
		if score > maxScore {
			maxScore = score
			dominantAttack = attackType
		}
	}

	report.AttackType = dominantAttack

	for i, data := range model.TrainingData {
		poisonScore := s.computePoisoningScore(data, model.Labels[i])

		if poisonScore > 0.5 {
			sample := &PoisoningSample{
				SampleID:   fmt.Sprintf("sample-%d-%d", i, time.Now().UnixNano()),
				DataPoint:  data,
				Label:      model.Labels[i],
				IsPoisoned: true,
				PoisonType: dominantAttack,
				Severity:   poisonScore,
				Confidence: poisonScore * 0.9,
				Features:   s.extractSampleFeatures(data),
				DetectedAt: time.Now(),
			}
			report.PoisonedSamples = append(report.PoisonedSamples, sample)
		}
	}

	report.IsPoisoned = len(report.PoisonedSamples) > 0
	report.PoisoningRatio = float64(len(report.PoisonedSamples)) / float64(len(model.TrainingData))
	report.OverallSeverity = s.assessSeverity(report.PoisoningRatio)

	if report.IsPoisoned {
		report.Recommendations = s.generateRecommendations(report)
	}

	s.reports[modelID] = report

	return report, nil
}

func (s *ModelPoisoningDetectorService) detectAttackType(data [][]float64, labels []int, attackType PoisoningAttackType) float64 {
	switch attackType {
	case PoisoningLabelFlip:
		return s.detectLabelFlip(data, labels)
	case PoisoningBackdoor:
		return s.detectBackdoor(data)
	case PoisoningSemantic:
		return s.detectSemanticAnomaly(data)
	case PoisoningCleanLabel:
		return s.detectCleanLabel(data)
	default:
		return 0.0
	}
}

func (s *ModelPoisoningDetectorService) detectLabelFlip(data [][]float64, labels []int) float64 {
	flipCount := 0

	labelGroups := make(map[int][]int)
	for i, label := range labels {
		labelGroups[label] = append(labelGroups[label], i)
	}

	for label1, indices1 := range labelGroups {
		for label2, indices2 := range labelGroups {
			if label1 >= label2 {
				continue
			}

			centroid1 := s.computeCentroid(data, indices1)
			centroid2 := s.computeCentroid(data, indices2)

			distance := s.computeDistance(centroid1, centroid2)

			if distance < 1.0 {
				flipCount += len(indices1) + len(indices2)
			}
		}
	}

	if len(data) == 0 {
		return 0.0
	}

	return math.Min(float64(flipCount)/float64(len(data)), 1.0)
}

func (s *ModelPoisoningDetectorService) detectBackdoor(data [][]float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	variance := s.computeOverallVariance(data)

	highVarianceCount := 0
	for _, point := range data {
		pointVariance := s.computeVariance(point)
		if pointVariance > variance*2 {
			highVarianceCount++
		}
	}

	return math.Min(float64(highVarianceCount)/float64(len(data))*1.5, 1.0)
}

func (s *ModelPoisoningDetectorService) detectSemanticAnomaly(data [][]float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	centroid := s.computeGlobalCentroid(data)

	anomalyCount := 0
	for _, point := range data {
		distance := s.computeDistance(point, centroid)
		if distance > 5.0 {
			anomalyCount++
		}
	}

	return math.Min(float64(anomalyCount)/float64(len(data))*2, 1.0)
}

func (s *ModelPoisoningDetectorService) detectCleanLabel(data [][]float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	cleanScore := 0.0
	for _, point := range data {
		isClean := true
		for _, v := range point {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				isClean = false
				break
			}
		}

		if isClean {
			cleanScore += 0.1
		}
	}

	return math.Min(cleanScore/float64(len(data)), 0.7)
}

func (s *ModelPoisoningDetectorService) computePoisoningScore(data []float64, label int) float64 {
	score := 0.0

	anomalyScore := s.detectDataAnomaly(data)
	score += anomalyScore * 0.4

	labelConsistency := s.checkLabelConsistency(data, label)
	score += (1.0 - labelConsistency) * 0.3

	magnitude := 0.0
	for _, v := range data {
		magnitude += v * v
	}
	magnitude = math.Sqrt(magnitude)

	if magnitude > 10.0 || magnitude < 0.1 {
		score += 0.2
	}

	outlierScore := s.detectOutlier(data)
	score += outlierScore * 0.1

	return math.Min(score, 1.0)
}

func (s *ModelPoisoningDetectorService) detectDataAnomaly(data []float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	mean := s.computeMean(data)
	stdDev := s.computeStdDev(data, mean)

	anomalyScore := 0.0
	for _, v := range data {
		zScore := math.Abs(v-mean) / (stdDev + 1e-10)
		if zScore > 3.0 {
			anomalyScore += 0.1
		}
	}

	return math.Min(anomalyScore, 1.0)
}

func (s *ModelPoisoningDetectorService) checkLabelConsistency(data []float64, label int) float64 {
	labelPrime := 0.7 + rand.Float64()*0.3

	if label < 0 {
		labelPrime *= 0.8
	}

	return labelPrime
}

func (s *ModelPoisoningDetectorService) detectOutlier(data []float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	mean := s.computeMean(data)
	stdDev := s.computeStdDev(data, mean)

	outlierCount := 0
	for _, v := range data {
		if math.Abs(v-mean) > 3*stdDev {
			outlierCount++
		}
	}

	return float64(outlierCount) / float64(len(data))
}

func (s *ModelPoisoningDetectorService) extractSampleFeatures(data []float64) map[string]float64 {
	features := make(map[string]float64)

	features["mean"] = s.computeMean(data)
	features["std_dev"] = s.computeStdDev(data, features["mean"])
	features["min"] = s.computeMin(data)
	features["max"] = s.computeMax(data)
	features["range"] = features["max"] - features["min"]
	features["norm"] = s.computeNorm(data)
	features["skewness"] = s.computeSkewness(data, features["mean"], features["std_dev"])
	features["kurtosis"] = s.computeKurtosis(data, features["mean"], features["std_dev"])

	return features
}

func (s *ModelPoisoningDetectorService) computeCentroid(data [][]float64, indices []int) []float64 {
	if len(indices) == 0 {
		return []float64{}
	}

	dimension := len(data[indices[0]])
	centroid := make([]float64, dimension)

	for _, idx := range indices {
		for j := 0; j < dimension; j++ {
			centroid[j] += data[idx][j]
		}
	}

	for j := range centroid {
		centroid[j] /= float64(len(indices))
	}

	return centroid
}

func (s *ModelPoisoningDetectorService) computeGlobalCentroid(data [][]float64) []float64 {
	if len(data) == 0 {
		return []float64{}
	}

	indices := make([]int, len(data))
	for i := range indices {
		indices[i] = i
	}

	return s.computeCentroid(data, indices)
}

func (s *ModelPoisoningDetectorService) computeOverallVariance(data [][]float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	totalVariance := 0.0
	for _, point := range data {
		totalVariance += s.computeVariance(point)
	}

	return totalVariance / float64(len(data))
}

func (s *ModelPoisoningDetectorService) computeDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	sum := 0.0
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return math.Sqrt(sum)
}

func (s *ModelPoisoningDetectorService) computeMean(data []float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range data {
		sum += v
	}

	return sum / float64(len(data))
}

func (s *ModelPoisoningDetectorService) computeVariance(data []float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	mean := s.computeMean(data)
	sum := 0.0
	for _, v := range data {
		diff := v - mean
		sum += diff * diff
	}

	return sum / float64(len(data))
}

func (s *ModelPoisoningDetectorService) computeStdDev(data []float64, mean float64) float64 {
	return math.Sqrt(s.computeVariance(data))
}

func (s *ModelPoisoningDetectorService) computeMin(data []float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	minVal := data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
	}

	return minVal
}

func (s *ModelPoisoningDetectorService) computeMax(data []float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	maxVal := data[0]
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}

	return maxVal
}

func (s *ModelPoisoningDetectorService) computeNorm(data []float64) float64 {
	sum := 0.0
	for _, v := range data {
		sum += v * v
	}
	return math.Sqrt(sum)
}

func (s *ModelPoisoningDetectorService) computeSkewness(data []float64, mean, stdDev float64) float64 {
	if len(data) == 0 || stdDev == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range data {
		sum += math.Pow((v-mean)/stdDev, 3)
	}

	return sum / float64(len(data))
}

func (s *ModelPoisoningDetectorService) computeKurtosis(data []float64, mean, stdDev float64) float64 {
	if len(data) == 0 || stdDev == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range data {
		sum += math.Pow((v-mean)/stdDev, 4)
	}

	return sum/float64(len(data)) - 3
}

func (s *ModelPoisoningDetectorService) assessSeverity(poisoningRatio float64) PoisoningSeverity {
	switch {
	case poisoningRatio < 0.05:
		return SeverityLow
	case poisoningRatio < 0.15:
		return SeverityMedium
	case poisoningRatio < 0.30:
		return SeverityHigh
	default:
		return SeverityCritical
	}
}

func (s *ModelPoisoningDetectorService) generateRecommendations(report *PoisoningDetectionReport) []string {
	recommendations := make([]string, 0)

	switch report.OverallSeverity {
	case SeverityLow:
		recommendations = append(recommendations,
			"Monitor training data quality more closely",
			"Implement basic data validation filters")
	case SeverityMedium:
		recommendations = append(recommendations,
			"Filter out detected poisoned samples",
			"Retrain model with cleaned dataset",
			"Implement additional data validation")
	case SeverityHigh:
		recommendations = append(recommendations,
			"Immediately stop using the affected model",
			"Perform comprehensive data audit",
			"Retrain model with verified clean data",
			"Implement poisoning detection in training pipeline")
	case SeverityCritical:
		recommendations = append(recommendations,
			"CRITICAL: Model compromised by poisoning attack",
			"Quarantine affected model immediately",
			"Conduct full security investigation",
			"Retrain from verified clean dataset",
			"Review and harden data collection pipeline",
			"Implement multiple defense layers")
	}

	return recommendations
}

func (s *ModelPoisoningDetectorService) GetReport(ctx context.Context, modelID string) (*PoisoningDetectionReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	report, exists := s.reports[modelID]
	if !exists {
		return nil, fmt.Errorf("report for model %s not found", modelID)
	}

	return report, nil
}

func (s *ModelPoisoningDetectorService) GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	model, exists := s.models[modelID]
	if !exists {
		return nil, ErrModelNotFound
	}

	return model, nil
}

func (s *ModelPoisoningDetectorService) GetDetectionStats(ctx context.Context) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	stats["total_models"] = len(s.models)
	stats["total_reports"] = len(s.reports)

	activeDetectors := 0
	for _, detector := range s.detectors {
		if detector.IsActive {
			activeDetectors++
		}
	}
	stats["active_detectors"] = activeDetectors

	poisonedModels := 0
	for _, report := range s.reports {
		if report.IsPoisoned {
			poisonedModels++
		}
	}
	stats["poisoned_models"] = poisonedModels

	return stats
}
