package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewModelService(t *testing.T) {
	service := NewModelService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.models)
	assert.NotNil(t, service.cache)
}

func TestModelServicePredict(t *testing.T) {
	service := NewModelService()

	tests := []struct {
		name     string
		features *FeatureVector
	}{
		{
			name: "nil features",
			features: nil,
		},
		{
			name: "empty feature vector",
			features: &FeatureVector{
				FeatureVector: []float64{},
			},
		},
		{
			name: "normal features",
			features: &FeatureVector{
				MouseSpeedAvg:    1.5,
				MouseSpeedMax:    3.0,
				MouseSpeedMin:    0.5,
				MouseSpeedVar:    10.0,
				TrajectoryLength: 200.0,
				PathEfficiency:  0.6,
				DirectionChanges: 15,
				Smoothness:      0.5,
				ClickCount:       3,
				ClickRegularity:  0.3,
				ClickSpeed:       1.5,
				DataPointDensity: 10.0,
				FeatureVector:    []float64{1.5, 3.0, 0.5, 10.0, 0.2, 0.6, 1.5, 0.5, 0.3, 0.3, 0.15, 10.0},
			},
		},
		{
			name: "bot-like features",
			features: &FeatureVector{
				MouseSpeedAvg:    10.0,
				MouseSpeedMax:    20.0,
				MouseSpeedMin:    5.0,
				MouseSpeedVar:    100.0,
				TrajectoryLength: 500.0,
				PathEfficiency:  0.98,
				DirectionChanges: 2,
				Smoothness:      0.9,
				ClickCount:       5,
				ClickRegularity:  0.95,
				ClickSpeed:       15.0,
				DataPointDensity: 2.0,
				FeatureVector:    []float64{10.0, 20.0, 5.0, 100.0, 0.5, 0.98, 0.2, 0.9, 0.5, 0.95, 1.5, 2.0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prediction := service.Predict(tt.features)
			assert.NotNil(t, prediction)
			assert.GreaterOrEqual(t, prediction.BotScore, 0.0)
			assert.LessOrEqual(t, prediction.BotScore, 1.0)
			assert.GreaterOrEqual(t, prediction.HumanScore, 0.0)
			assert.LessOrEqual(t, prediction.HumanScore, 1.0)
			assert.GreaterOrEqual(t, prediction.Confidence, 0.0)
			assert.LessOrEqual(t, prediction.Confidence, 1.0)
		})
	}
}

func TestModelServiceTrain(t *testing.T) {
	service := NewModelService()

	features := [][]float64{
		{1.0, 2.0, 3.0, 4.0, 5.0},
		{1.5, 2.5, 3.5, 4.5, 5.5},
		{0.8, 1.8, 2.8, 3.8, 4.8},
		{10.0, 20.0, 30.0, 40.0, 50.0},
		{11.0, 21.0, 31.0, 41.0, 51.0},
		{9.5, 19.5, 29.5, 39.5, 49.5},
	}

	labels := []int{0, 0, 0, 1, 1, 1}

	data := &TrainingData{
		Features: features,
		Labels:   labels,
	}

	err := service.Train(data)
	assert.NoError(t, err)

	modelNames := service.ListModels()
	assert.Greater(t, len(modelNames), 0)
}

func TestModelServiceTrainWithInvalidData(t *testing.T) {
	service := NewModelService()

	tests := []struct {
		name string
		data *TrainingData
	}{
		{
			name: "nil data",
			data: nil,
		},
		{
			name: "empty features",
			data: &TrainingData{
				Features: [][]float64{},
				Labels:   []int{},
			},
		},
		{
			name: "mismatched lengths",
			data: &TrainingData{
				Features: [][]float64{{1.0, 2.0}, {3.0, 4.0}},
				Labels:   []int{0, 1, 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Train(tt.data)
			assert.Error(t, err)
		})
	}
}

func TestModelServiceCrossValidate(t *testing.T) {
	service := NewModelService()

	features := [][]float64{
		{1.0, 2.0, 3.0},
		{1.2, 2.2, 3.2},
		{0.8, 1.8, 2.8},
		{10.0, 20.0, 30.0},
		{11.0, 21.0, 31.0},
		{9.0, 19.0, 29.0},
		{1.5, 2.5, 3.5},
		{0.9, 1.9, 2.9},
		{10.5, 20.5, 30.5},
		{9.5, 19.5, 29.5},
	}

	labels := []int{0, 0, 0, 0, 0, 1, 1, 1, 1, 1}

	data := &TrainingData{
		Features: features,
		Labels:   labels,
	}

	result, err := service.CrossValidate(data, 3)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.Folds)
	assert.Len(t, result.AccuracyScores, 3)
}

func TestModelServiceSaveAndLoad(t *testing.T) {
	service := NewModelService()

	modelNames := service.ListModels()
	assert.Greater(t, len(modelNames), 0)

	err := service.DeleteModel("nonexistent_model")
	assert.Error(t, err)
}

func TestModelServiceDeleteModel(t *testing.T) {
	service := NewModelService()

	err := service.DeleteModel("default")
	assert.Error(t, err)
}

func TestModelServiceSetCurrentModel(t *testing.T) {
	service := NewModelService()

	err := service.SetCurrentModel("default")
	assert.NoError(t, err)

	err = service.SetCurrentModel("nonexistent")
	assert.Error(t, err)
}

func TestModelServicePredictBatch(t *testing.T) {
	service := NewModelService()

	featuresList := []*FeatureVector{
		{
			MouseSpeedAvg: 1.0,
			FeatureVector: []float64{1.0, 2.0, 3.0},
		},
		{
			MouseSpeedAvg: 5.0,
			FeatureVector: []float64{5.0, 6.0, 7.0},
		},
		nil,
	}

	results := service.PredictBatch(featuresList)
	assert.Len(t, results, 3)
	assert.NotNil(t, results[0])
	assert.NotNil(t, results[1])
	assert.NotNil(t, results[2])
}

func TestModelServiceClearCache(t *testing.T) {
	service := NewModelService()

	features := &FeatureVector{
		FeatureVector: []float64{1.0, 2.0, 3.0},
	}

	service.Predict(features)
	service.Predict(features)

	service.ClearCache()
}

func TestModelServiceNormalizeFeatures(t *testing.T) {
	service := NewModelService()

	features := [][]float64{
		{1.0, 2.0, 3.0},
		{2.0, 4.0, 6.0},
		{3.0, 6.0, 9.0},
	}

	normalized := service.normalizeFeatures(features)
	assert.Len(t, normalized, len(features))
	for i := range normalized {
		assert.Len(t, normalized[i], len(features[i]))
	}
}

func TestDecisionTree(t *testing.T) {
	service := NewModelService()

	features := [][]float64{
		{1.0, 2.0},
		{1.5, 2.5},
		{3.0, 4.0},
		{3.5, 4.5},
	}

	labels := []int{0, 0, 1, 1}

	tree := service.trainDecisionTree(features, labels, 5)
	assert.NotNil(t, tree)
}

func TestCalculateEntropy(t *testing.T) {
	service := NewModelService()

	entropy := service.calculateEntropy([]int{0, 0, 0, 0})
	assert.Equal(t, 0.0, entropy)

	entropy = service.calculateEntropy([]int{0, 1, 0, 1})
	assert.Greater(t, entropy, 0.0)

	entropy = service.calculateEntropy([]int{})
	assert.Equal(t, 0.0, entropy)
}

func TestCalculateInformationGain(t *testing.T) {
	service := NewModelService()

	features := [][]float64{
		{1.0},
		{2.0},
		{3.0},
		{4.0},
	}
	labels := []int{0, 0, 1, 1}

	gain := service.calculateInformationGain(features, labels, 0, 2.5)
	assert.GreaterOrEqual(t, gain, 0.0)
}

func TestGetUniqueClasses(t *testing.T) {
	service := NewModelService()

	classes := service.getUniqueClasses([]int{0, 1, 0, 2, 1})
	assert.Contains(t, classes, 0)
	assert.Contains(t, classes, 1)
	assert.Contains(t, classes, 2)
}

func TestEvaluateModel(t *testing.T) {
	service := NewModelService()

	model := &TrainedModel{
		Weights: []float64{0.1, 0.1, 0.1},
		Bias:    0.0,
	}

	features := [][]float64{
		{1.0, 2.0, 3.0},
		{0.5, 1.0, 1.5},
		{2.0, 3.0, 4.0},
	}
	labels := []int{0, 0, 1}

	metrics := service.evaluateModel(model, features, labels)
	assert.Contains(t, metrics, "accuracy")
	assert.Contains(t, metrics, "precision")
	assert.Contains(t, metrics, "recall")
	assert.Contains(t, metrics, "f1")
}

func TestExtractFeatureVector(t *testing.T) {
	service := NewModelService()

	features := &FeatureVector{
		MouseSpeedAvg:    1.0,
		MouseSpeedMax:    2.0,
		MouseSpeedMin:    0.5,
		MouseSpeedVar:    10.0,
		TrajectoryLength: 200.0,
		PathEfficiency:  0.6,
		DirectionChanges: 15,
		Smoothness:      0.5,
		ClickCount:       3,
		ClickRegularity:  0.3,
		ClickSpeed:       1.5,
		DataPointDensity: 10.0,
	}

	vector := service.extractFeatureVector(features)
	assert.NotNil(t, vector)
	assert.Len(t, vector, 12)
}

func TestMLPredictionModel(t *testing.T) {
	prediction := &MLPrediction{
		IsBot:        true,
		Confidence:   0.85,
		BotScore:     0.9,
		HumanScore:   0.1,
		ModelVersion: "1.0.0",
		FeaturesUsed: []string{"feature1", "feature2"},
	}

	jsonData, err := json.Marshal(prediction)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	var decoded MLPrediction
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, prediction.IsBot, decoded.IsBot)
	assert.Equal(t, prediction.Confidence, decoded.Confidence)
	assert.Equal(t, prediction.BotScore, decoded.BotScore)
}

func TestModelMetadata(t *testing.T) {
	metadata := ModelMetadata{
		Name:      "test_model",
		Version:   "1.0.0",
		Type:      ModelTypeSVM,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Accuracy:  0.85,
		Precision: 0.82,
		Recall:    0.88,
		F1Score:   0.85,
		Features:  []string{"f1", "f2", "f3"},
	}

	assert.Equal(t, "test_model", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Equal(t, ModelTypeSVM, metadata.Type)
	assert.Greater(t, metadata.Accuracy, 0.0)
}

func TestTrainingConfig(t *testing.T) {
	config := TrainingConfig{
		LearningRate:    0.01,
		MaxIterations:   1000,
		TrainTestSplit:  0.8,
		BatchSize:       32,
		Regularization:  0.01,
	}

	assert.Equal(t, 0.01, config.LearningRate)
	assert.Equal(t, 1000, config.MaxIterations)
	assert.Equal(t, 0.8, config.TrainTestSplit)
}

func TestTrainedModel(t *testing.T) {
	model := &TrainedModel{
		Metadata: ModelMetadata{
			Name:    "test",
			Version: "1.0.0",
		},
		Config: TrainingConfig{
			LearningRate:  0.01,
			MaxIterations: 100,
		},
		Weights: []float64{0.1, 0.2, 0.3},
		Bias:    0.5,
		SVMParams: SVMParams{
			Kernel: "rbf",
			Gamma:  0.1,
			C:      1.0,
		},
	}

	assert.NotNil(t, model)
	assert.Len(t, model.Weights, 3)
	assert.Equal(t, 0.5, model.Bias)
}

func TestPredictWithTree(t *testing.T) {
	service := NewModelService()

	tree := &DecisionTree{
		IsLeaf:     true,
		Prediction: 1,
		ClassCounts: map[int]int{1: 10},
	}

	features := []float64{1.0, 2.0, 3.0}
	prediction := service.predictWithTree(tree, features)
	assert.Equal(t, 1, prediction)

	tree = &DecisionTree{
		IsLeaf: false,
		SplitFeature: 0,
		SplitValue: 2.0,
		Left: &DecisionTree{
			IsLeaf:     true,
			Prediction: 0,
		},
		Right: &DecisionTree{
			IsLeaf:     true,
			Prediction: 1,
		},
	}

	prediction = service.predictWithTree(tree, []float64{1.0})
	assert.Equal(t, 0, prediction)

	prediction = service.predictWithTree(tree, []float64{3.0})
	assert.Equal(t, 1, prediction)
}

func TestSplitData(t *testing.T) {
	service := NewModelService()

	features := [][]float64{
		{1.0},
		{2.0},
		{3.0},
		{4.0},
	}
	labels := []int{0, 0, 1, 1}

	leftFeatures, leftLabels, rightFeatures, rightLabels := service.splitData(features, labels, 0, 2.5)

	assert.Len(t, leftFeatures, 2)
	assert.Len(t, leftLabels, 2)
	assert.Len(t, rightFeatures, 2)
	assert.Len(t, rightLabels, 2)

	for _, f := range leftFeatures {
		assert.LessOrEqual(t, f[0], 2.5)
	}
	for _, f := range rightFeatures {
		assert.Greater(t, f[0], 2.5)
	}
}

func TestGenerateCacheKey(t *testing.T) {
	service := NewModelService()

	key1 := service.generateCacheKey([]float64{1.0, 2.0, 3.0})
	key2 := service.generateCacheKey([]float64{1.0, 2.0, 3.0})
	key3 := service.generateCacheKey([]float64{1.0, 2.0, 4.0})

	assert.Equal(t, key1, key2)
	assert.NotEqual(t, key1, key3)
}
