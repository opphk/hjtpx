package service

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"gorm.io/gorm"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type LabelType int

const (
	LabelHuman LabelType = iota
	LabelBot
	LabelUnknown
)

type TrainingSample struct {
	ID         string    `json:"id"`
	Features   []float64 `json:"features"`
	Label      int       `json:"label"`
	IsBot      bool      `json:"is_bot"`
	Weight     float64   `json:"weight"`
	SourceID   string    `json:"source_id"`
	CollectedAt time.Time `json:"collected_at"`
	Verified   bool      `json:"verified"`
}

type DatasetSplit struct {
	Train      []*TrainingSample `json:"train"`
	Validation []*TrainingSample `json:"validation"`
	Test       []*TrainingSample `json:"test"`
}

type DataCollectionConfig struct {
	MinSamples      int       `json:"min_samples"`
	MaxSamples     int       `json:"max_samples"`
	BalanceRatio   float64   `json:"balance_ratio"`
	Augmentation   bool      `json:"augmentation"`
	Normalize      bool      `json:"normalize"`
	TrainSplit     float64   `json:"train_split"`
	ValidationSplit float64  `json:"validation_split"`
	TestSplit      float64   `json:"test_split"`
}

type DataCollector struct {
	samples     map[string]*TrainingSample
	labels     map[string]LabelType
	config     DataCollectionConfig
	mutex      sync.RWMutex
	db         *gorm.DB
}

type TrainingDataManager struct {
	collector      *DataCollector
	modelService   *ModelService
	config        TrainingConfig
	isTraining    bool
	trainingMutex sync.Mutex
}

type AugmentationConfig struct {
	NoiseLevel       float64 `json:"noise_level"`
	ScaleRange       []float64 `json:"scale_range"`
	RotationRange    []float64 `json:"rotation_range"`
	ShiftRange       []float64 `json:"shift_range"`
}

type DataStatistics struct {
	TotalSamples    int            `json:"total_samples"`
	HumanSamples    int            `json:"human_samples"`
	BotSamples      int            `json:"bot_samples"`
	UnknownSamples  int            `json:"unknown_samples"`
	VerifiedSamples int            `json:"verified_samples"`
	UnverifiedSamples int          `json:"unverified_samples"`
	FeatureMeans    []float64      `json:"feature_means"`
	FeatureStds     []float64      `json:"feature_stds"`
	ClassDistribution map[int]int `json:"class_distribution"`
}

type LabeledBehaviorData struct {
	ID             uint      `json:"id"`
	SessionID      string    `json:"session_id"`
	UserID         uint      `json:"user_id"`
	Features       string    `json:"features"`
	Label          int       `json:"label"`
	IsBot          bool      `json:"is_bot"`
	Confidence     float64   `json:"confidence"`
	Source         string    `json:"source"`
	Verified       bool      `json:"verified"`
	VerifiedBy     string    `json:"verified_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func NewDataCollector(config DataCollectionConfig) *DataCollector {
	if config.MinSamples == 0 {
		config.MinSamples = 100
	}
	if config.MaxSamples == 0 {
		config.MaxSamples = 100000
	}
	if config.BalanceRatio == 0 {
		config.BalanceRatio = 1.0
	}
	if config.TrainSplit == 0 {
		config.TrainSplit = 0.7
	}
	if config.ValidationSplit == 0 {
		config.ValidationSplit = 0.15
	}
	if config.TestSplit == 0 {
		config.TestSplit = 0.15
	}
	
	return &DataCollector{
		samples: make(map[string]*TrainingSample),
		labels: make(map[string]LabelType),
		config: config,
	}
}

func NewTrainingDataManager(collector *DataCollector, modelService *ModelService) *TrainingDataManager {
	return &TrainingDataManager{
		collector:    collector,
		modelService: modelService,
		config: TrainingConfig{
			LearningRate:   0.01,
			MaxIterations:  1000,
			TrainTestSplit: 0.8,
			BatchSize:      32,
			Regularization: 0.01,
		},
	}
}

func (c *DataCollector) AddSample(sample *TrainingSample) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if sample == nil || len(sample.Features) == 0 {
		return fmt.Errorf("invalid sample")
	}
	
	sampleID := fmt.Sprintf("%s_%d", sample.SourceID, len(c.samples))
	sample.ID = sampleID
	sample.CollectedAt = time.Now()
	
	if sample.Weight == 0 {
		sample.Weight = 1.0
	}
	
	c.samples[sampleID] = sample
	
	return nil
}

func (c *DataCollector) AddSamples(samples []*TrainingSample) error {
	for _, sample := range samples {
		if err := c.AddSample(sample); err != nil {
			return err
		}
	}
	return nil
}

func (c *DataCollector) LabelSample(sampleID string, label LabelType) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	sample, exists := c.samples[sampleID]
	if !exists {
		return fmt.Errorf("sample not found: %s", sampleID)
	}
	
	c.labels[sampleID] = label
	
	switch label {
	case LabelHuman:
		sample.Label = 0
		sample.IsBot = false
	case LabelBot:
		sample.Label = 1
		sample.IsBot = true
	case LabelUnknown:
		sample.Label = -1
	}
	
	sample.Verified = true
	
	return nil
}

func (c *DataCollector) GetSample(sampleID string) (*TrainingSample, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	sample, exists := c.samples[sampleID]
	if !exists {
		return nil, fmt.Errorf("sample not found: %s", sampleID)
	}
	
	return sample, nil
}

func (c *DataCollector) GetAllSamples() []*TrainingSample {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	samples := make([]*TrainingSample, 0, len(c.samples))
	for _, sample := range c.samples {
		samples = append(samples, sample)
	}
	
	return samples
}

func (c *DataCollector) GetLabeledSamples() []*TrainingSample {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	samples := make([]*TrainingSample, 0)
	for _, sample := range c.samples {
		if sample.Label >= 0 {
			samples = append(samples, sample)
		}
	}
	
	return samples
}

func (c *DataCollector) GetVerifiedSamples() []*TrainingSample {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	samples := make([]*TrainingSample, 0)
	for _, sample := range c.samples {
		if sample.Verified {
			samples = append(samples, sample)
		}
	}
	
	return samples
}

func (c *DataCollector) RemoveSample(sampleID string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if _, exists := c.samples[sampleID]; !exists {
		return fmt.Errorf("sample not found: %s", sampleID)
	}
	
	delete(c.samples, sampleID)
	delete(c.labels, sampleID)
	
	return nil
}

func (c *DataCollector) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.samples = make(map[string]*TrainingSample)
	c.labels = make(map[string]LabelType)
}

func (c *DataCollector) GetStatistics() *DataStatistics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	stats := &DataStatistics{
		ClassDistribution: make(map[int]int),
	}
	
	for _, sample := range c.samples {
		stats.TotalSamples++
		
		if sample.Verified {
			stats.VerifiedSamples++
		} else {
			stats.UnverifiedSamples++
		}
		
		switch sample.Label {
		case 0:
			stats.HumanSamples++
			stats.ClassDistribution[0]++
		case 1:
			stats.BotSamples++
			stats.ClassDistribution[1]++
		default:
			stats.UnknownSamples++
		}
	}
	
	if len(c.samples) > 0 {
		numFeatures := 0
		for _, sample := range c.samples {
			if numFeatures == 0 {
				numFeatures = len(sample.Features)
				break
			}
		}
		
		if numFeatures > 0 {
			stats.FeatureMeans = make([]float64, numFeatures)
			stats.FeatureStds = make([]float64, numFeatures)
			
			for j := 0; j < numFeatures; j++ {
				sum := 0.0
				count := 0
				for _, sample := range c.samples {
					if j < len(sample.Features) {
						sum += sample.Features[j]
						count++
					}
				}
				if count > 0 {
					stats.FeatureMeans[j] = sum / float64(count)
				}
			}
			
			for j := 0; j < numFeatures; j++ {
				sumSq := 0.0
				count := 0
				for _, sample := range c.samples {
					if j < len(sample.Features) {
						diff := sample.Features[j] - stats.FeatureMeans[j]
						sumSq += diff * diff
						count++
					}
				}
				if count > 0 {
					stats.FeatureStds[j] = math.Sqrt(sumSq / float64(count))
					if stats.FeatureStds[j] == 0 {
						stats.FeatureStds[j] = 1.0
					}
				}
			}
		}
	}
	
	return stats
}

func (c *DataCollector) SplitDataset() (*DatasetSplit, error) {
	samples := c.GetLabeledSamples()
	
	if len(samples) < 10 {
		return nil, fmt.Errorf("insufficient samples for splitting")
	}
	
	humanSamples := make([]*TrainingSample, 0)
	botSamples := make([]*TrainingSample, 0)
	
	for _, sample := range samples {
		if sample.Label == 0 {
			humanSamples = append(humanSamples, sample)
		} else if sample.Label == 1 {
			botSamples = append(botSamples, sample)
		}
	}
	
	if c.config.Augmentation && c.config.BalanceRatio != 1.0 {
		humanSamples, botSamples = c.balanceDataset(humanSamples, botSamples)
	}
	
	rand.Shuffle(len(humanSamples), func(i, j int) {
		humanSamples[i], humanSamples[j] = humanSamples[j], humanSamples[i]
	})
	rand.Shuffle(len(botSamples), func(i, j int) {
		botSamples[i], botSamples[i] = botSamples[i], botSamples[j]
	})
	
	humanSplit := int(float64(len(humanSamples)) * c.config.TrainSplit)
	botSplit := int(float64(len(botSamples)) * c.config.TrainSplit)
	
	trainHuman := humanSamples[:humanSplit]
	valHuman := humanSamples[humanSplit:]
	
	valHumanSplit := int(float64(len(valHuman)) * (c.config.ValidationSplit / (c.config.ValidationSplit + c.config.TestSplit)))
	
	validationHuman := valHuman[:valHumanSplit]
	testHuman := valHuman[valHumanSplit:]
	
	trainBot := botSamples[:botSplit]
	valBot := botSamples[botSplit:]
	
	valBotSplit := int(float64(len(valBot)) * (c.config.ValidationSplit / (c.config.ValidationSplit + c.config.TestSplit)))
	
	validationBot := valBot[:valBotSplit]
	testBot := valBot[valBotSplit:]
	
	split := &DatasetSplit{
		Train:      append(trainHuman, trainBot...),
		Validation: append(validationHuman, validationBot...),
		Test:       append(testHuman, testBot...),
	}
	
	rand.Shuffle(len(split.Train), func(i, j int) {
		split.Train[i], split.Train[j] = split.Train[j], split.Train[i]
	})
	rand.Shuffle(len(split.Validation), func(i, j int) {
		split.Validation[i], split.Validation[j] = split.Validation[j], split.Validation[i]
	})
	rand.Shuffle(len(split.Test), func(i, j int) {
		split.Test[i], split.Test[j] = split.Test[j], split.Test[i]
	})
	
	return split, nil
}

func (c *DataCollector) balanceDataset(humanSamples, botSamples []*TrainingSample) ([]*TrainingSample, []*TrainingSample) {
	targetRatio := c.config.BalanceRatio
	
	if len(humanSamples) > len(botSamples) {
		multiplier := int(float64(len(humanSamples)) / float64(len(botSamples)))
		if multiplier > 1 && targetRatio >= 1.0 {
			augmentedBot := c.augmentSamples(botSamples, (multiplier-1)*len(botSamples))
			botSamples = append(botSamples, augmentedBot...)
		}
	} else if len(botSamples) > len(humanSamples) {
		multiplier := int(float64(len(botSamples)) / float64(len(humanSamples)))
		if multiplier > 1 && targetRatio <= 1.0 {
			augmentedHuman := c.augmentSamples(humanSamples, (multiplier-1)*len(humanSamples))
			humanSamples = append(humanSamples, augmentedHuman...)
		}
	}
	
	return humanSamples, botSamples
}

func (c *DataCollector) augmentSamples(samples []*TrainingSample, targetCount int) []*TrainingSample {
	if len(samples) == 0 {
		return nil
	}
	
	augmented := make([]*TrainingSample, 0)
	
	config := AugmentationConfig{
		NoiseLevel:     0.05,
		ScaleRange:     []float64{0.9, 1.1},
		ShiftRange:     []float64{-0.1, 0.1},
	}
	
	for len(augmented) < targetCount {
		for _, sample := range samples {
			if len(augmented) >= targetCount {
				break
			}
			
			augmentedSample := c.augmentSample(sample, config)
			augmented = append(augmented, augmentedSample)
		}
	}
	
	return augmented
}

func (c *DataCollector) augmentSample(sample *TrainingSample, config AugmentationConfig) *TrainingSample {
	augmentedFeatures := make([]float64, len(sample.Features))
	
	for i, value := range sample.Features {
		noise := (rand.Float64()*2 - 1) * config.NoiseLevel * value
		augmentedFeatures[i] = value + noise
	}
	
	return &TrainingSample{
		Features:    augmentedFeatures,
		Label:      sample.Label,
		Weight:     sample.Weight,
		SourceID:   sample.SourceID + "_aug",
		CollectedAt: time.Now(),
		Verified:   false,
	}
}

func (c *DataCollector) NormalizeFeatures(features [][]float64) [][]float64 {
	if len(features) == 0 || len(features[0]) == 0 {
		return features
	}
	
	numFeatures := len(features[0])
	means := make([]float64, numFeatures)
	stds := make([]float64, numFeatures)
	
	for j := 0; j < numFeatures; j++ {
		sum := 0.0
		for i := 0; i < len(features); i++ {
			if j < len(features[i]) {
				sum += features[i][j]
			}
		}
		means[j] = sum / float64(len(features))
	}
	
	for j := 0; j < numFeatures; j++ {
		sumSq := 0.0
		for i := 0; i < len(features); i++ {
			if j < len(features[i]) {
				diff := features[i][j] - means[j]
				sumSq += diff * diff
			}
		}
		stds[j] = math.Sqrt(sumSq / float64(len(features)))
		if stds[j] == 0 {
			stds[j] = 1.0
		}
	}
	
	normalized := make([][]float64, len(features))
	for i := 0; i < len(features); i++ {
		normalized[i] = make([]float64, numFeatures)
		for j := 0; j < numFeatures; j++ {
			if j < len(features[i]) {
				normalized[i][j] = (features[i][j] - means[j]) / stds[j]
			}
		}
	}
	
	return normalized
}

func (m *TrainingDataManager) CollectFromVerification(verification *models.Verification, behaviorData []models.BehaviorData, isBot bool) error {
	if verification == nil {
		return fmt.Errorf("verification is nil")
	}
	
	behaviorService := NewBehaviorAnalysisService()
	result, err := behaviorService.AnalyzeBehavior(behaviorData)
	if err != nil {
		return err
	}
	
	features := m.extractFeaturesFromResult(result)
	
	sample := &TrainingSample{
		Features:     features,
		Label:        0,
		Weight:       1.0,
		SourceID:     fmt.Sprintf("verif_%d", verification.ID),
		CollectedAt:  time.Now(),
		Verified:     true,
	}
	
	if isBot {
		sample.Label = 1
	}
	
	return m.collector.AddSample(sample)
}

func (m *TrainingDataManager) extractFeaturesFromResult(result *AnalysisResult) []float64 {
	features := make([]float64, 0, 20)
	
	features = append(features, result.Trajectory.AverageSpeed)
	features = append(features, result.Trajectory.MaxSpeed)
	features = append(features, result.Trajectory.MinSpeed)
	features = append(features, result.Trajectory.SpeedVariance)
	features = append(features, result.Trajectory.TotalDistance)
	features = append(features, result.Trajectory.PathEfficiency)
	features = append(features, float64(result.Trajectory.DirectionChanges))
	features = append(features, result.Trajectory.Smoothness)
	
	features = append(features, float64(result.ClickPattern.ClickCount))
	features = append(features, result.ClickPattern.AverageInterval)
	features = append(features, result.ClickPattern.ClickSpeed)
	features = append(features, result.ClickPattern.Regularity)
	
	features = append(features, float64(len(result.Trajectory.Points)))
	features = append(features, result.Features.DataPointDensity)
	
	if result.MLPrediction != nil {
		features = append(features, result.MLPrediction.BotScore)
		features = append(features, result.MLPrediction.Confidence)
	} else {
		features = append(features, 0.5)
		features = append(features, 0.5)
	}
	
	features = append(features, result.RiskScore/100)
	
	return features
}

func (m *TrainingDataManager) PrepareTrainingData() (*TrainingData, error) {
	samples := m.collector.GetVerifiedSamples()
	
	if len(samples) < int(m.config.TrainTestSplit)*10 {
		return nil, fmt.Errorf("insufficient training samples: %d", len(samples))
	}
	
	features := make([][]float64, len(samples))
	labels := make([]int, len(samples))
	
	for i, sample := range samples {
		features[i] = sample.Features
		labels[i] = sample.Label
	}
	
	if m.collector.config.Normalize {
		features = m.collector.NormalizeFeatures(features)
	}
	
	return &TrainingData{
		Features: features,
		Labels:   labels,
	}, nil
}

func (m *TrainingDataManager) Train() error {
	m.trainingMutex.Lock()
	defer m.trainingMutex.Unlock()
	
	if m.isTraining {
		return fmt.Errorf("training already in progress")
	}
	
	m.isTraining = true
	defer func() { m.isTraining = false }()
	
	trainingData, err := m.PrepareTrainingData()
	if err != nil {
		return err
	}
	
	if err := m.modelService.Train(trainingData); err != nil {
		return err
	}
	
	return nil
}

func (m *TrainingDataManager) TrainWithCrossValidation(folds int) error {
	m.trainingMutex.Lock()
	defer m.trainingMutex.Unlock()
	
	if m.isTraining {
		return fmt.Errorf("training already in progress")
	}
	
	m.isTraining = true
	defer func() { m.isTraining = false }()
	
	trainingData, err := m.PrepareTrainingData()
	if err != nil {
		return err
	}
	
	cvResult, err := m.modelService.CrossValidate(trainingData, folds)
	if err != nil {
		return err
	}
	
	if cvResult.MeanAccuracy < 0.7 {
		return fmt.Errorf("cross-validation accuracy too low: %.2f", cvResult.MeanAccuracy)
	}
	
	if err := m.modelService.Train(trainingData); err != nil {
		return err
	}
	
	return nil
}

func (m *TrainingDataManager) SaveTrainingData(path string) error {
	samples := m.collector.GetAllSamples()
	
	_, err := json.MarshalIndent(samples, "", "  ")
	return err
}

func (m *TrainingDataManager) LoadTrainingData(path string) error {
	return nil
}

func (m *TrainingDataManager) ExportToCSV(path string) error {
	samples := m.collector.GetAllSamples()
	
	if len(samples) == 0 {
		return fmt.Errorf("no samples to export")
	}
	
	return nil
}

func (m *TrainingDataManager) GetStatistics() *DataStatistics {
	return m.collector.GetStatistics()
}

func (m *TrainingDataManager) GenerateSyntheticData(count int, isBot bool) error {
	rand.Seed(time.Now().UnixNano())
	
	for i := 0; i < count; i++ {
		features := m.generateSyntheticFeatures(isBot)
		
		sample := &TrainingSample{
			Features:     features,
			Label:       0,
			Weight:      1.0,
			SourceID:    fmt.Sprintf("synthetic_%d", i),
			CollectedAt: time.Now(),
			Verified:    false,
		}
		
		if isBot {
			sample.Label = 1
		}
		
		if err := m.collector.AddSample(sample); err != nil {
			return err
		}
	}
	
	return nil
}

func (m *TrainingDataManager) generateSyntheticFeatures(isBot bool) []float64 {
	features := make([]float64, 18)
	
	if isBot {
		features[0] = rand.Float64()*2 + 3
		features[1] = rand.Float64()*5 + 5
		features[2] = rand.Float64() * 0.1
		features[3] = rand.Float64() * 10
		features[4] = rand.Float64()*500 + 100
		features[5] = rand.Float64()*0.1 + 0.9
		features[6] = float64(rand.Intn(5))
		features[7] = rand.Float64()*0.2 + 0.8
		features[8] = float64(rand.Intn(3) + 3)
		features[9] = rand.Float64()*10 + 50
		features[10] = rand.Float64()*2 + 10
		features[11] = rand.Float64()*0.1 + 0.9
		features[12] = float64(rand.Intn(20) + 5)
		features[13] = rand.Float64() * 10
		features[14] = rand.Float64()*0.3 + 0.7
		features[15] = rand.Float64()*0.3 + 0.7
		features[16] = rand.Float64()*0.3 + 0.7
		features[17] = rand.Float64()*0.3 + 0.7
	} else {
		features[0] = rand.Float64()*2 + 0.5
		features[1] = rand.Float64()*5 + 1
		features[2] = rand.Float64() * 0.5
		features[3] = rand.Float64() * 50
		features[4] = rand.Float64()*200 + 50
		features[5] = rand.Float64()*0.3 + 0.4
		features[6] = float64(rand.Intn(30) + 10)
		features[7] = rand.Float64()*0.4 + 0.3
		features[8] = float64(rand.Intn(5) + 1)
		features[9] = rand.Float64()*200 + 100
		features[10] = rand.Float64()*2 + 0.5
		features[11] = rand.Float64()*0.4 + 0.2
		features[12] = float64(rand.Intn(100) + 50)
		features[13] = rand.Float64()*20 + 5
		features[14] = rand.Float64()*0.2 + 0.1
		features[15] = rand.Float64()*0.3 + 0.1
		features[16] = rand.Float64()*0.3 + 0.1
		features[17] = rand.Float64()*0.3 + 0.1
	}
	
	return features
}

func (m *TrainingDataManager) ValidateDataQuality() (bool, []string) {
	issues := []string{}
	
	stats := m.collector.GetStatistics()
	
	if stats.TotalSamples < m.collector.config.MinSamples {
		issues = append(issues, fmt.Sprintf("insufficient samples: %d < %d", stats.TotalSamples, m.collector.config.MinSamples))
	}
	
	if stats.HumanSamples == 0 {
		issues = append(issues, "no human samples")
	}
	
	if stats.BotSamples == 0 {
		issues = append(issues, "no bot samples")
	}
	
	imbalance := float64(stats.HumanSamples) / float64(stats.BotSamples)
	if imbalance < 0.1 || imbalance > 10 {
		issues = append(issues, fmt.Sprintf("class imbalance: %.2f", imbalance))
	}
	
	for _, sample := range m.collector.GetAllSamples() {
		for i, val := range sample.Features {
			if math.IsNaN(val) || math.IsInf(val, 0) {
				issues = append(issues, fmt.Sprintf("invalid feature value at sample %s, feature %d", sample.SourceID, i))
			}
		}
	}
	
	return len(issues) == 0, issues
}

func (m *TrainingDataManager) CleanData() error {
	samples := m.collector.GetAllSamples()
	
	for _, sample := range samples {
		isValid := true
		
		for _, val := range sample.Features {
			if math.IsNaN(val) || math.IsInf(val, 0) {
				isValid = false
				break
			}
		}
		
		if !isValid {
			if err := m.collector.RemoveSample(sample.SourceID); err != nil {
				return err
			}
		}
	}
	
	return nil
}

func (m *TrainingDataManager) GetTrainingProgress() map[string]interface{} {
	stats := m.GetStatistics()
	
	return map[string]interface{}{
		"total_samples":    stats.TotalSamples,
		"human_samples":    stats.HumanSamples,
		"bot_samples":      stats.BotSamples,
		"verified_samples": stats.VerifiedSamples,
		"is_training":      m.isTraining,
	}
}
