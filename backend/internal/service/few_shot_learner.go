package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

type FewShotLearnerService struct {
	mu                sync.RWMutex
	initialized       bool
	modelPrior        *FSModelPrior
	supportSets       map[string]*FSSupportSet
	querySets         map[string]*FSQuerySet
	episodeCount      int
	strategies        map[string]*FSStrategy
	metrics           *FSLearnerMetrics
}

type FSModelPrior struct {
	BaseParameters   []float64
	Variance         float64
	KnowledgeBase    []*FSPriorKnowledge
	HyperParameters  *FSHyperParameters
}

type FSPriorKnowledge struct {
	ConceptID     string
	ConceptName   string
	Parameters    map[string]float64
	Confidence    float64
	SourceDomain  string
	LastUpdated   time.Time
}

type FSHyperParameters struct {
	LearningRate    float64
	Regularization float64
	BatchNorm       bool
	DropoutRate     float64
	WeightDecay     float64
	GradientClip    float64
}

type FSSupportSet struct {
	TaskID          string
	SupportSize     int
	QuerySize       int
	Classes         []string
	Samples         []*FSSample
	TaskComplexity  float64
	QualityScore    float64
	CreatedAt       time.Time
}

type FSQuerySet struct {
	TaskID      string
	QuerySize   int
	Predictions []string
	Confidence  []float64
	EvaluatedAt time.Time
}

type FSSample struct {
	SampleID    string
	Features    []float64
	Label       string
	IsSupport   bool
	Augmented   bool
	NoiseLevel  float64
	Quality     float64
}

type FSStrategy struct {
	StrategyID   string
	Name         string
	Parameters   map[string]float64
	SuccessRate  float64
	UsageCount   int
	Performance  float64
	IsActive     bool
}

type FSLearnerMetrics struct {
	TotalEpisodes    int
	AverageAccuracy  float64
	AverageConfidence float64
	TotalAdaptations int
	BestStrategy     string
	SuccessCount     int
}

type FSFewShotResult struct {
	TaskID          string                  `json:"task_id"`
	Predictions     []string                `json:"predictions"`
	Confidence      float64                 `json:"confidence"`
	LearningCurve   []float64               `json:"learning_curve"`
	AdaptationSteps int                     `json:"adaptation_steps"`
	StrategyUsed    string                  `json:"strategy_used"`
	Accuracy        float64                 `json:"accuracy"`
}

type FSFewShotTask struct {
	TaskID        string
	SupportSamples []*FSSample
	QuerySamples  []*FSSample
	Classes       []string
	NWay          int
	KShot         int
	Complexity    float64
	TaskDomain    string
}

func NewFewShotLearnerService() *FewShotLearnerService {
	return &FewShotLearnerService{
		modelPrior: &FSModelPrior{
			BaseParameters: make([]float64, 64),
			Variance:       0.1,
			KnowledgeBase:  make([]*FSPriorKnowledge, 0),
			HyperParameters: &FSHyperParameters{
				LearningRate:    0.001,
				Regularization:  0.01,
				BatchNorm:       true,
				DropoutRate:     0.5,
				WeightDecay:     0.0001,
				GradientClip:    1.0,
			},
		},
		supportSets:  make(map[string]*FSSupportSet),
		querySets:    make(map[string]*FSQuerySet),
		episodeCount: 0,
		strategies:  make(map[string]*FSStrategy),
		metrics: &FSLearnerMetrics{
			TotalEpisodes:    0,
			AverageAccuracy:  0.0,
			AverageConfidence: 0.0,
			TotalAdaptations: 0,
			SuccessCount:     0,
		},
	}
}

func (fsl *FewShotLearnerService) Initialize(ctx context.Context) error {
	fsl.mu.Lock()
	defer fsl.mu.Unlock()

	if fsl.initialized {
		return nil
	}

	for i := range fsl.modelPrior.BaseParameters {
		fsl.modelPrior.BaseParameters[i] = 0.01
	}

	fsl.initializeStrategies()

	fsl.initialized = true
	return nil
}

func (fsl *FewShotLearnerService) initializeStrategies() {
	strategyConfigs := []struct {
		id   string
		name string
		lr   float64
	}{
		{"gradient", "Gradient-Based", 0.01},
		{"reptile", "Reptile", 0.001},
		{"maml", "MAML", 0.0001},
		{"protonet", "ProtoNet", 0.01},
		{"matching_net", "Matching Networks", 0.001},
		{"relation_net", "Relation Networks", 0.0001},
	}

	for _, cfg := range strategyConfigs {
		fsl.strategies[cfg.id] = &FSStrategy{
			StrategyID:  cfg.id,
			Name:        cfg.name,
			Parameters:  map[string]float64{"learning_rate": cfg.lr},
			SuccessRate: 0.8,
			UsageCount:  0,
			Performance: 0.75,
			IsActive:    true,
		}
	}
}

func (fsl *FewShotLearnerService) LearnFewShot(ctx context.Context, task *FSFewShotTask) (*FSFewShotResult, error) {
	fsl.mu.Lock()
	defer fsl.mu.Unlock()

	if !fsl.initialized {
		return nil, fmt.Errorf("few-shot learner service not initialized")
	}

	result := &FSFewShotResult{
		TaskID:      task.TaskID,
		Predictions: make([]string, 0),
		LearningCurve: make([]float64, 0),
	}

	supportSize := len(task.SupportSamples)
	querySize := len(task.QuerySamples)

	if supportSize == 0 {
		return nil, fmt.Errorf("no support samples provided")
	}

	result.AdaptationSteps = supportSize

	learningCurve := fsl.simulateLearningCurve(supportSize, task.Complexity)
	result.LearningCurve = learningCurve

	bestStrategy := fsl.selectBestStrategy(task)
	result.StrategyUsed = bestStrategy

	prototypes := fsl.computePrototypes(task)

	for i := 0; i < querySize; i++ {
		prediction := fsl.predictWithPrototypes(task.QuerySamples[i], prototypes, task.Classes)
		result.Predictions = append(result.Predictions, prediction)
	}

	result.Confidence = fsl.calculateConfidence(learningCurve)
	result.Accuracy = fsl.calculateAccuracy(result.Predictions, task)

	supportSet := &FSSupportSet{
		TaskID:          task.TaskID,
		SupportSize:     supportSize,
		QuerySize:       querySize,
		Classes:         task.Classes,
		Samples:         task.SupportSamples,
		TaskComplexity:  task.Complexity,
		QualityScore:    result.Accuracy,
		CreatedAt:       time.Now(),
	}

	fsl.supportSets[task.TaskID] = supportSet

	querySet := &FSQuerySet{
		TaskID:      task.TaskID,
		QuerySize:   querySize,
		Predictions: result.Predictions,
		Confidence:  make([]float64, len(result.Predictions)),
		EvaluatedAt: time.Now(),
	}
	for i := range querySet.Confidence {
		querySet.Confidence[i] = result.Confidence
	}
	fsl.querySets[task.TaskID] = querySet

	fsl.episodeCount++
	fsl.updateMetrics(result, bestStrategy)

	return result, nil
}

func (fsl *FewShotLearnerService) simulateLearningCurve(supportSize int, complexity float64) []float64 {
	curve := make([]float64, supportSize)

	baseConvergence := 0.5 + (1.0-complexity)*0.3

	for i := 0; i < supportSize; i++ {
		progress := float64(i+1) / float64(supportSize)
		noise := math.Mod(float64(i), 0.1) * 0.05

		curve[i] = math.Min(1.0, math.Max(0.0, baseConvergence+progress*0.4-noise))
	}

	return curve
}

func (fsl *FewShotLearnerService) selectBestStrategy(task *FSFewShotTask) string {
	bestStrategy := "maml"
	bestScore := 0.0

	for id, strategy := range fsl.strategies {
		if !strategy.IsActive {
			continue
		}

		baseScore := strategy.SuccessRate
		complexityFactor := 1.0 - task.Complexity*0.2
		domainFactor := fsl.getDomainFactor(task.TaskDomain, id)

		expectedScore := baseScore * complexityFactor * domainFactor

		if expectedScore > bestScore {
			bestScore = expectedScore
			bestStrategy = id
		}
	}

	fsl.strategies[bestStrategy].UsageCount++

	return bestStrategy
}

func (fsl *FewShotLearnerService) getDomainFactor(domain, strategyID string) float64 {
	domainStrategyMap := map[string][]string{
		"image":    {"protonet", "matching_net", "relation_net"},
		"text":     {"gradient", "reptile", "maml"},
		"audio":    {"maml", "reptile"},
		"tabular":  {"gradient", "maml"},
		"video":    {"protonet", "gradient"},
	}

	preferredStrategies, exists := domainStrategyMap[domain]
	if !exists {
		return 0.8
	}

	for _, s := range preferredStrategies {
		if s == strategyID {
			return 1.2
		}
	}

	return 0.9
}

func (fsl *FewShotLearnerService) computePrototypes(task *FSFewShotTask) map[string][]float64 {
	prototypes := make(map[string][]float64)

	classFeatures := make(map[string][][]float64)

	for _, sample := range task.SupportSamples {
		if sample.IsSupport {
			classFeatures[sample.Label] = append(classFeatures[sample.Label], sample.Features)
		}
	}

	for class, features := range classFeatures {
		if len(features) == 0 {
			continue
		}

		prototype := make([]float64, len(features[0]))
		for i := range prototype {
			sum := 0.0
			for _, f := range features {
				if i < len(f) {
					sum += f[i]
				}
			}
			prototype[i] = sum / float64(len(features))
		}

		prototypes[class] = prototype
	}

	return prototypes
}

func (fsl *FewShotLearnerService) predictWithPrototypes(sample *FSSample, prototypes map[string][]float64, classes []string) string {
	if len(prototypes) == 0 {
		if len(classes) > 0 {
			return classes[0]
		}
		return "unknown"
	}

	bestClass := classes[0]
	if bestClass == "" {
		for class := range prototypes {
			bestClass = class
			break
		}
	}

	minDistance := math.MaxFloat64

	for class, prototype := range prototypes {
		distance := fsl.euclideanDistance(sample.Features, prototype)
		if distance < minDistance {
			minDistance = distance
			bestClass = class
		}
	}

	return bestClass
}

func (fsl *FewShotLearnerService) euclideanDistance(a, b []float64) float64 {
	if len(a) != len(b) {
		minLen := len(a)
		if len(b) < minLen {
			minLen = len(b)
		}
	}

	sum := 0.0
	for i := 0; i < len(a) && i < len(b); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return math.Sqrt(sum)
}

func (fsl *FewShotLearnerService) calculateConfidence(learningCurve []float64) float64 {
	if len(learningCurve) == 0 {
		return 0.0
	}

	total := 0.0
	for _, score := range learningCurve {
		total += score
	}

	convergence := learningCurve[len(learningCurve)-1]
	avgScore := total / float64(len(learningCurve))

	return (convergence + avgScore) / 2.0
}

func (fsl *FewShotLearnerService) calculateAccuracy(predictions []string, task *FSFewShotTask) float64 {
	if len(predictions) == 0 || len(task.QuerySamples) == 0 {
		return 0.0
	}

	correct := 0
	for i, pred := range predictions {
		if i < len(task.QuerySamples) {
			actual := task.QuerySamples[i].Label
			if pred == actual {
				correct++
			}
		}
	}

	return float64(correct) / float64(len(predictions))
}

func (fsl *FewShotLearnerService) updateMetrics(result *FSFewShotResult, strategyID string) {
	fsl.metrics.TotalEpisodes++

	totalAccuracy := fsl.metrics.AverageAccuracy*float64(fsl.metrics.TotalEpisodes-1) + result.Accuracy
	fsl.metrics.AverageAccuracy = totalAccuracy / float64(fsl.metrics.TotalEpisodes)

	totalConfidence := fsl.metrics.AverageConfidence*float64(fsl.metrics.TotalEpisodes-1) + result.Confidence
	fsl.metrics.AverageConfidence = totalConfidence / float64(fsl.metrics.TotalEpisodes)

	fsl.metrics.TotalAdaptations += result.AdaptationSteps

	if result.Accuracy >= 0.7 {
		fsl.metrics.SuccessCount++
	}

	successRate := float64(fsl.metrics.SuccessCount) / float64(fsl.metrics.TotalEpisodes)
	if successRate > fsl.strategies[strategyID].SuccessRate {
		fsl.strategies[strategyID].SuccessRate = successRate
	}

	fsl.metrics.BestStrategy = strategyID
}

func (fsl *FewShotLearnerService) GetMetrics() *FSLearnerMetrics {
	fsl.mu.RLock()
	defer fsl.mu.RUnlock()

	metrics := *fsl.metrics
	return &metrics
}

func (fsl *FewShotLearnerService) GetStrategies() []*FSStrategy {
	fsl.mu.RLock()
	defer fsl.mu.RUnlock()

	strategies := make([]*FSStrategy, 0)
	for _, strategy := range fsl.strategies {
		s := *strategy
		strategies = append(strategies, &s)
	}

	return strategies
}

func (fsl *FewShotLearnerService) GetSupportSet(taskID string) (*FSSupportSet, error) {
	fsl.mu.RLock()
	defer fsl.mu.RUnlock()

	supportSet, exists := fsl.supportSets[taskID]
	if !exists {
		return nil, fmt.Errorf("support set not found for task: %s", taskID)
	}

	return supportSet, nil
}

func (fsl *FewShotLearnerService) GetQuerySet(taskID string) (*FSQuerySet, error) {
	fsl.mu.RLock()
	defer fsl.mu.RUnlock()

	querySet, exists := fsl.querySets[taskID]
	if !exists {
		return nil, fmt.Errorf("query set not found for task: %s", taskID)
	}

	return querySet, nil
}

func (fsl *FewShotLearnerService) UpdateStrategy(ctx context.Context, strategyID string, params map[string]interface{}) error {
	fsl.mu.Lock()
	defer fsl.mu.Unlock()

	strategy, exists := fsl.strategies[strategyID]
	if !exists {
		return fmt.Errorf("strategy not found: %s", strategyID)
	}

	for key, value := range params {
		switch key {
		case "learning_rate":
			if lr, ok := value.(float64); ok {
				strategy.Parameters["learning_rate"] = lr
			}
		case "is_active":
			if active, ok := value.(bool); ok {
				strategy.IsActive = active
			}
		}
	}

	return nil
}

func (fsl *FewShotLearnerService) AugmentSupportSet(ctx context.Context, taskID string, augmentationType string) error {
	fsl.mu.Lock()
	defer fsl.mu.Unlock()

	supportSet, exists := fsl.supportSets[taskID]
	if !exists {
		return fmt.Errorf("support set not found for task: %s", taskID)
	}

	augmentedSamples := make([]*FSSample, 0)

	for _, sample := range supportSet.Samples {
		augmented := fsl.augmentSample(sample, augmentationType)
		augmentedSamples = append(augmentedSamples, augmented)
	}

	supportSet.Samples = append(supportSet.Samples, augmentedSamples...)
	supportSet.SupportSize = len(supportSet.Samples)

	return nil
}

func (fsl *FewShotLearnerService) augmentSample(sample *FSSample, augmentationType string) *FSSample {
	augmented := &FSSample{
		SampleID:  sample.SampleID + "_aug",
		Features:  make([]float64, len(sample.Features)),
		Label:     sample.Label,
		IsSupport: true,
		Augmented: true,
	}

	switch augmentationType {
	case "noise":
		noiseLevel := 0.1
		for i := range augmented.Features {
			if i < len(sample.Features) {
				noise := (math.randomFloat64() - 0.5) * noiseLevel
				augmented.Features[i] = sample.Features[i] + noise
			}
		}
	case "scaling":
		scale := 0.9 + math.randomFloat64()*0.2
		for i := range augmented.Features {
			if i < len(sample.Features) {
				augmented.Features[i] = sample.Features[i] * scale
			}
		}
	case "rotation":
		for i := range augmented.Features {
			if i < len(sample.Features) {
				augmented.Features[i] = sample.Features[len(sample.Features)-1-i]
			}
		}
	default:
		copy(augmented.Features, sample.Features)
	}

	augmented.NoiseLevel = 0.1
	augmented.Quality = sample.Quality * 0.95

	return augmented
}

func (fsl *FewShotLearnerService) CrossValidate(ctx context.Context, taskID string, kFold int) (*FSCrossValidationResult, error) {
	fsl.mu.RLock()
	defer fsl.mu.RUnlock()

	supportSet, exists := fsl.supportSets[taskID]
	if !exists {
		return nil, fmt.Errorf("support set not found for task: %s", taskID)
	}

	result := &FSCrossValidationResult{
		TaskID:        taskID,
		KFold:         kFold,
		FoldAccuracies: make([]float64, kFold),
	}

	foldSize := len(supportSet.Samples) / kFold

	for fold := 0; fold < kFold; fold++ {
		startIdx := fold * foldSize
		endIdx := startIdx + foldSize
		if fold == kFold-1 {
			endIdx = len(supportSet.Samples)
		}

		validationSamples := supportSet.Samples[startIdx:endIdx]
		trainingSamples := make([]*FSSample, 0)
		trainingSamples = append(trainingSamples, supportSet.Samples[:startIdx]...)
		trainingSamples = append(trainingSamples, supportSet.Samples[endIdx:]...)

		foldAccuracy := fsl.evaluateFold(trainingSamples, validationSamples)
		result.FoldAccuracies[fold] = foldAccuracy
	}

	result.MeanAccuracy = fsl.calculateMeanAccuracy(result.FoldAccuracies)
	result.StdDeviation = fsl.calculateStdDeviation(result.FoldAccuracies)

	return result, nil
}

func (fsl *FewShotLearnerService) evaluateFold(trainingSamples, validationSamples []*FSSample) float64 {
	if len(validationSamples) == 0 {
		return 0.0
	}

	correct := 0
	for _, sample := range validationSamples {
		predicted := fsl.predictSample(sample, trainingSamples)
		if predicted == sample.Label {
			correct++
		}
	}

	return float64(correct) / float64(len(validationSamples))
}

func (fsl *FewShotLearnerService) predictSample(sample *FSSample, trainingSamples []*FSSample) string {
	classFeatures := make(map[string][][]float64)

	for _, s := range trainingSamples {
		classFeatures[s.Label] = append(classFeatures[s.Label], s.Features)
	}

	prototypes := make(map[string][]float64)
	for class, features := range classFeatures {
		prototype := make([]float64, len(features[0]))
		for i := range prototype {
			sum := 0.0
			for _, f := range features {
				if i < len(f) {
					sum += f[i]
				}
			}
			prototype[i] = sum / float64(len(features))
		}
		prototypes[class] = prototype
	}

	return fsl.predictWithPrototypes(sample, prototypes, []string{})
}

func (fsl *FewShotLearnerService) calculateMeanAccuracy(accuracies []float64) float64 {
	if len(accuracies) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, acc := range accuracies {
		sum += acc
	}

	return sum / float64(len(accuracies))
}

func (fsl *FewShotLearnerService) calculateStdDeviation(accuracies []float64) float64 {
	if len(accuracies) == 0 {
		return 0.0
	}

	mean := fsl.calculateMeanAccuracy(accuracies)

	sumSquaredDiff := 0.0
	for _, acc := range accuracies {
		diff := acc - mean
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(len(accuracies))
	return math.Sqrt(variance)
}

type FSCrossValidationResult struct {
	TaskID        string
	KFold         int
	FoldAccuracies []float64
	MeanAccuracy  float64
	StdDeviation  float64
}

type fsmath struct{}

func (m fsmath) randomFloat64() float64 {
	return float64(time.Now().UnixNano()%10000) / 10000.0
}
