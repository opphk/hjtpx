package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDataCollector(t *testing.T) {
	config := DataCollectionConfig{
		MinSamples:     100,
		MaxSamples:    10000,
		BalanceRatio:  1.0,
		TrainSplit:   0.7,
		ValidationSplit: 0.15,
		TestSplit:    0.15,
	}

	collector := NewDataCollector(config)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.samples)
	assert.NotNil(t, collector.labels)
}

func TestDataCollectorAddSample(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})

	sample := &TrainingSample{
		Features: []float64{1.0, 2.0, 3.0},
		Label:   0,
		Weight:  1.0,
		SourceID: "test_sample",
	}

	err := collector.AddSample(sample)
	assert.NoError(t, err)

	retrieved, err := collector.GetSample("test_sample_0")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
}

func TestDataCollectorAddSampleInvalid(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})

	err := collector.AddSample(nil)
	assert.Error(t, err)

	err = collector.AddSample(&TrainingSample{
		Features: []float64{},
	})
	assert.Error(t, err)
}

func TestDataCollectorAddSamples(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})

	samples := []*TrainingSample{
		{Features: []float64{1.0, 2.0}, SourceID: "s1"},
		{Features: []float64{3.0, 4.0}, SourceID: "s2"},
		{Features: []float64{5.0, 6.0}, SourceID: "s3"},
	}

	err := collector.AddSamples(samples)
	assert.NoError(t, err)

	allSamples := collector.GetAllSamples()
	assert.Len(t, allSamples, 3)
}

func TestDataCollectorLabelSample(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})

	sample := &TrainingSample{
		Features: []float64{1.0, 2.0},
		Label:   -1,
		SourceID: "sample1",
	}

	err := collector.AddSample(sample)
	assert.NoError(t, err)

	sampleID := "sample1_0"
	err = collector.LabelSample(sampleID, LabelHuman)
	assert.NoError(t, err)

	retrieved, err := collector.GetSample(sampleID)
	assert.NoError(t, err)
	assert.Equal(t, 0, retrieved.Label)
	assert.False(t, retrieved.IsBot)
	assert.True(t, retrieved.Verified)

	err = collector.LabelSample(sampleID, LabelBot)
	assert.NoError(t, err)

	retrieved, err = collector.GetSample(sampleID)
	assert.NoError(t, err)
	assert.Equal(t, 1, retrieved.Label)
	assert.True(t, retrieved.IsBot)
}

func TestDataCollectorGetLabeledSamples(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})

	samples := []*TrainingSample{
		{Features: []float64{1.0}, Label: 0, SourceID: "h1"},
		{Features: []float64{2.0}, Label: 1, SourceID: "b1"},
		{Features: []float64{3.0}, Label: -1, SourceID: "u1"},
	}

	collector.AddSamples(samples)

	labeled := collector.GetLabeledSamples()
	assert.Len(t, labeled, 2)
}

func TestDataCollectorGetVerifiedSamples(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})

	samples := []*TrainingSample{
		{Features: []float64{1.0}, Label: 0, Verified: true, SourceID: "v1"},
		{Features: []float64{2.0}, Label: 1, Verified: false, SourceID: "nv1"},
	}

	collector.AddSamples(samples)

	verified := collector.GetVerifiedSamples()
	assert.Len(t, verified, 1)
}

func TestDataCollectorRemoveSample(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})

	sample := &TrainingSample{
		Features: []float64{1.0, 2.0},
		SourceID: "remove_test",
	}

	collector.AddSample(sample)
	sampleID := "remove_test_0"

	err := collector.RemoveSample(sampleID)
	assert.NoError(t, err)

	_, err = collector.GetSample(sampleID)
	assert.Error(t, err)
}

func TestDataCollectorClear(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})

	samples := []*TrainingSample{
		{Features: []float64{1.0}, SourceID: "c1"},
		{Features: []float64{2.0}, SourceID: "c2"},
	}

	collector.AddSamples(samples)
	assert.Len(t, collector.GetAllSamples(), 2)

	collector.Clear()
	assert.Len(t, collector.GetAllSamples(), 0)
}

func TestDataCollectorGetStatistics(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})

	samples := []*TrainingSample{
		{Features: []float64{1.0, 2.0}, Label: 0, Verified: true, SourceID: "s1"},
		{Features: []float64{2.0, 3.0}, Label: 0, Verified: true, SourceID: "s2"},
		{Features: []float64{3.0, 4.0}, Label: 1, Verified: true, SourceID: "s3"},
		{Features: []float64{4.0, 5.0}, Label: 1, Verified: false, SourceID: "s4"},
		{Features: []float64{5.0, 6.0}, Label: -1, Verified: false, SourceID: "s5"},
	}

	collector.AddSamples(samples)

	stats := collector.GetStatistics()
	assert.Equal(t, 5, stats.TotalSamples)
	assert.Equal(t, 2, stats.HumanSamples)
	assert.Equal(t, 2, stats.BotSamples)
	assert.Equal(t, 1, stats.UnknownSamples)
	assert.Equal(t, 3, stats.VerifiedSamples)
	assert.Equal(t, 2, stats.UnverifiedSamples)
}

func TestDataCollectorSplitDataset(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{
		TrainSplit:     0.7,
		ValidationSplit: 0.15,
		TestSplit:      0.15,
	})

	for i := 0; i < 10; i++ {
		collector.AddSample(&TrainingSample{
			Features: []float64{float64(i), float64(i + 1)},
			Label:   0,
			SourceID: "human",
		})
	}

	for i := 0; i < 10; i++ {
		collector.AddSample(&TrainingSample{
			Features: []float64{float64(i + 100), float64(i + 101)},
			Label:   1,
			SourceID: "bot",
		})
	}

	split, err := collector.SplitDataset()
	assert.NoError(t, err)
	assert.NotNil(t, split)

	assert.Greater(t, len(split.Train), 0)
	assert.Greater(t, len(split.Validation), 0)
	assert.Greater(t, len(split.Test), 0)

	total := len(split.Train) + len(split.Validation) + len(split.Test)
	assert.Equal(t, 20, total)
}

func TestDataCollectorAugmentSample(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{
		Augmentation: true,
	})

	sample := &TrainingSample{
		Features: []float64{1.0, 2.0, 3.0},
		Label:   0,
		Weight:  1.0,
		SourceID: "original",
	}

	config := AugmentationConfig{
		NoiseLevel:  0.1,
		ScaleRange:  []float64{0.9, 1.1},
		ShiftRange:  []float64{-0.1, 0.1},
	}

	augmented := collector.augmentSample(sample, config)
	assert.NotNil(t, augmented)
	assert.Len(t, augmented.Features, len(sample.Features))
	assert.Equal(t, sample.Label, augmented.Label)
}

func TestDataCollectorNormalizeFeatures(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{
		Normalize: true,
	})

	features := [][]float64{
		{1.0, 2.0, 3.0},
		{2.0, 4.0, 6.0},
		{3.0, 6.0, 9.0},
	}

	normalized := collector.NormalizeFeatures(features)
	assert.Len(t, normalized, len(features))

	for i := range normalized {
		assert.Len(t, normalized[i], len(features[i]))
	}
}

func TestNewTrainingDataManager(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})
	modelService := NewModelService()

	manager := NewTrainingDataManager(collector, modelService)
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.collector)
	assert.NotNil(t, manager.modelService)
}

func TestTrainingDataManagerGenerateSyntheticData(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})
	modelService := NewModelService()
	manager := NewTrainingDataManager(collector, modelService)

	err := manager.GenerateSyntheticData(5, false)
	assert.NoError(t, err)

	err = manager.GenerateSyntheticData(5, true)
	assert.NoError(t, err)

	stats := manager.GetStatistics()
	assert.Equal(t, 10, stats.TotalSamples)
	assert.Equal(t, 5, stats.HumanSamples)
	assert.Equal(t, 5, stats.BotSamples)
}

func TestTrainingDataManagerValidateDataQuality(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{
		MinSamples: 10,
	})
	modelService := NewModelService()
	manager := NewTrainingDataManager(collector, modelService)

	manager.GenerateSyntheticData(10, false)
	manager.GenerateSyntheticData(10, true)

	valid, issues := manager.ValidateDataQuality()
	assert.True(t, valid)
	assert.Empty(t, issues)
}

func TestTrainingDataManagerCleanData(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})
	modelService := NewModelService()
	manager := NewTrainingDataManager(collector, modelService)

	collector.AddSample(&TrainingSample{
		Features: []float64{1.0, 2.0},
		SourceID: "valid",
	})

	collector.AddSample(&TrainingSample{
		Features: []float64{1.0, 2.0},
		SourceID: "invalid",
	})

	err := manager.CleanData()
	assert.NoError(t, err)
}

func TestTrainingDataManagerGetTrainingProgress(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})
	modelService := NewModelService()
	manager := NewTrainingDataManager(collector, modelService)

	manager.GenerateSyntheticData(5, false)

	progress := manager.GetTrainingProgress()
	assert.Contains(t, progress, "total_samples")
	assert.Contains(t, progress, "human_samples")
	assert.Contains(t, progress, "bot_samples")
	assert.Contains(t, progress, "is_training")

	assert.Equal(t, 5, progress["total_samples"])
	assert.Equal(t, 5, progress["human_samples"])
	assert.False(t, progress["is_training"].(bool))
}

func TestTrainingDataManagerExtractFeaturesFromResult(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{})
	modelService := NewModelService()
	manager := NewTrainingDataManager(collector, modelService)

	result := &AnalysisResult{
		Trajectory: MouseTrajectory{
			AverageSpeed:     1.5,
			MaxSpeed:        3.0,
			MinSpeed:        0.5,
			SpeedVariance:   10.0,
			TotalDistance:   200.0,
			PathEfficiency:  0.6,
			DirectionChanges: 15,
			Smoothness:      0.5,
			Points:          []BehaviorDataPoint{{X: 0, Y: 0}},
		},
		ClickPattern: ClickPattern{
			ClickCount:      3,
			AverageInterval: 100.0,
			ClickSpeed:     1.5,
			Regularity:     0.3,
		},
		Features: FeatureVector{
			DataPointDensity: 10.0,
		},
		MLPrediction: &MLPrediction{
			BotScore:   0.3,
			Confidence: 0.7,
		},
		RiskScore: 30.0,
	}

	features := manager.extractFeaturesFromResult(result)
	assert.NotNil(t, features)
	assert.Greater(t, len(features), 0)
}

func TestDataCollectorBalanceDataset(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{
		BalanceRatio: 1.0,
	})

	humanSamples := []*TrainingSample{
		{Features: []float64{1.0}, Label: 0},
		{Features: []float64{2.0}, Label: 0},
		{Features: []float64{3.0}, Label: 0},
	}

	botSamples := []*TrainingSample{
		{Features: []float64{10.0}, Label: 1},
	}

	balancedHuman, balancedBot := collector.balanceDataset(humanSamples, botSamples)
	assert.NotNil(t, balancedHuman)
	assert.NotNil(t, balancedBot)
}

func TestTrainingDataManagerPrepareTrainingData(t *testing.T) {
	collector := NewDataCollector(DataCollectionConfig{
		MinSamples: 5,
	})
	modelService := NewModelService()
	manager := NewTrainingDataManager(collector, modelService)

	collector.AddSample(&TrainingSample{
		Features: []float64{1.0, 2.0},
		Label:   0,
		Verified: true,
		SourceID: "t1",
	})
	collector.AddSample(&TrainingSample{
		Features: []float64{3.0, 4.0},
		Label:   1,
		Verified: true,
		SourceID: "t2",
	})

	data, err := manager.PrepareTrainingData()
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Len(t, data.Features, 2)
	assert.Len(t, data.Labels, 2)
}

func TestDataStatistics(t *testing.T) {
	stats := &DataStatistics{
		TotalSamples:     100,
		HumanSamples:    60,
		BotSamples:      30,
		UnknownSamples:  10,
		VerifiedSamples: 80,
		ClassDistribution: map[int]int{
			0: 60,
			1: 30,
		},
	}

	assert.Equal(t, 100, stats.TotalSamples)
	assert.Equal(t, 60, stats.HumanSamples)
	assert.Equal(t, 30, stats.BotSamples)
}

func TestTrainingSample(t *testing.T) {
	sample := &TrainingSample{
		Features:   []float64{1.0, 2.0, 3.0},
		Label:      0,
		Weight:     1.5,
		SourceID:   "test_source",
		Verified:   true,
	}

	assert.Len(t, sample.Features, 3)
	assert.Equal(t, 0, sample.Label)
	assert.Equal(t, 1.5, sample.Weight)
	assert.True(t, sample.Verified)
}

func TestDatasetSplit(t *testing.T) {
	split := &DatasetSplit{
		Train: []*TrainingSample{
			{Features: []float64{1.0}, Label: 0},
		},
		Validation: []*TrainingSample{
			{Features: []float64{2.0}, Label: 0},
		},
		Test: []*TrainingSample{
			{Features: []float64{3.0}, Label: 1},
		},
	}

	assert.Len(t, split.Train, 1)
	assert.Len(t, split.Validation, 1)
	assert.Len(t, split.Test, 1)
}

func TestDataCollectionConfig(t *testing.T) {
	config := DataCollectionConfig{
		MinSamples:      100,
		MaxSamples:     100000,
		BalanceRatio:   1.0,
		Augmentation:   true,
		Normalize:      true,
		TrainSplit:     0.7,
		ValidationSplit: 0.15,
		TestSplit:      0.15,
	}

	assert.Equal(t, 100, config.MinSamples)
	assert.Equal(t, 100000, config.MaxSamples)
	assert.Equal(t, 1.0, config.BalanceRatio)
	assert.True(t, config.Augmentation)
	assert.True(t, config.Normalize)
}

func TestAugmentationConfig(t *testing.T) {
	config := AugmentationConfig{
		NoiseLevel:    0.05,
		ScaleRange:   []float64{0.9, 1.1},
		RotationRange: []float64{-15, 15},
		ShiftRange:    []float64{-0.1, 0.1},
	}

	assert.Equal(t, 0.05, config.NoiseLevel)
	assert.Len(t, config.ScaleRange, 2)
	assert.Len(t, config.RotationRange, 2)
	assert.Len(t, config.ShiftRange, 2)
}

func TestLabelType(t *testing.T) {
	assert.Equal(t, LabelType(0), LabelHuman)
	assert.Equal(t, LabelType(1), LabelBot)
	assert.Equal(t, LabelType(2), LabelUnknown)
}
