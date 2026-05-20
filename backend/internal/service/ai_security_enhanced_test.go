package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAISecurityEnhancedService(t *testing.T) {
	service := NewAISecurityEnhancedService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.defenseEngine)
	assert.NotNil(t, service.poisoningDetector)
	assert.NotNil(t, service.backdoorDetector)
	assert.NotNil(t, service.watermarkEngine)
	assert.NotNil(t, service.models)
	assert.NotNil(t, service.detectionHistory)
}

func TestRegisterModel(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		Architecture: "resnet",
		InputShape:   []int{3, 224, 224},
		OutputShape:  []int{1000},
		Version:      "1.0.0",
	}

	err := service.RegisterModel(model)
	require.NoError(t, err)

	retrieved, err := service.GetModel("model-1")
	require.NoError(t, err)
	assert.Equal(t, "model-1", retrieved.ModelID)
}

func TestGetModelNotFound(t *testing.T) {
	service := NewAISecurityEnhancedService()

	_, err := service.GetModel("nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidModel, err)
}

func TestDetectAdversarialSamples(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	inputs := [][]float64{
		{1.0, 2.0, 3.0},
		{2.0, 3.0, 4.0},
		{10.0, 11.0, 12.0},
	}
	labels := []int{0, 1, 1}

	req := &AdversarialDetectionRequest{
		Input:        inputs,
		Labels:       labels,
		ModelWeights: model.Weights,
	}

	resp, err := service.DetectAdversarialSamples(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestDefendAgainstAdversarial(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	input := []float64{1.0, 2.0, 3.0, 4.0}

	req := &AdversarialDefenseRequest{
		ModelID:     "model-1",
		Input:       input,
		DefenseType: "gaussian_noise",
		Threshold:   0.5,
	}

	resp, err := service.DefendAgainstAdversarial(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Result)
}

func TestDefendAgainstAdversarialInvalidModel(t *testing.T) {
	service := NewAISecurityEnhancedService()

	req := &AdversarialDefenseRequest{
		ModelID:     "nonexistent",
		Input:       []float64{1.0, 2.0},
		DefenseType: "gaussian_noise",
		Threshold:   0.5,
	}

	resp, err := service.DefendAgainstAdversarial(context.Background(), req)
	assert.False(t, resp.Success)
	assert.Error(t, err)
}

func TestDetectModelPoisoning(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	trainingData := [][]float64{
		{1.0, 2.0, 3.0},
		{2.0, 3.0, 4.0},
		{5.0, 6.0, 7.0},
	}
	labels := []int{0, 1, 1}

	req := &PoisoningDetectionRequest{
		TrainingData: trainingData,
		Labels:       labels,
		ModelID:      "model-1",
		Threshold:    0.8,
	}

	resp, err := service.DetectModelPoisoning(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestDetectBackdoor(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	testInputs := [][]float64{
		{1.0, 2.0, 3.0},
		{2.0, 3.0, 4.0},
	}

	trigger := &BackdoorTrigger{
		TriggerID:   "trigger-1",
		Pattern:     []float64{1.0, 1.0, 1.0},
		Location:    []int{0, 0},
		TriggerMask: []bool{true, true, true},
		TargetClass: 1,
	}

	req := &BackdoorDetectionRequest{
		Model:      model,
		TestInputs: testInputs,
		Trigger:    trigger,
	}

	resp, err := service.DetectBackdoor(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestEmbedWatermark(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	watermark := []byte("test-watermark-data")

	req := &WatermarkRequest{
		ModelID:       "model-1",
		Watermark:     watermark,
		EmbeddingType: "low_frequency",
		Strength:      0.5,
	}

	resp, err := service.EmbedWatermark(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Watermark)
	assert.NotEmpty(t, resp.Watermark.WatermarkID)
}

func TestVerifyWatermark(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	watermark := []byte("test-watermark-data")

	embedReq := &WatermarkRequest{
		ModelID:       "model-1",
		Watermark:     watermark,
		EmbeddingType: "low_frequency",
		Strength:      0.5,
	}

	_, err := service.EmbedWatermark(context.Background(), embedReq)
	require.NoError(t, err)

	verifyReq := &WatermarkVerificationRequest{
		Model:         model,
		Watermark:     watermark,
		EmbeddingType: "low_frequency",
	}

	resp, err := service.VerifyWatermark(context.Background(), verifyReq)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestVerifyWatermarkNotFound(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-unwatermarked",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	verifyReq := &WatermarkVerificationRequest{
		Model:         model,
		Watermark:     []byte("test"),
		EmbeddingType: "low_frequency",
	}

	resp, err := service.VerifyWatermark(context.Background(), verifyReq)
	require.NoError(t, err)
	assert.False(t, resp.IsValid)
}

func TestPerformAdversarialTraining(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	adversarialSamples := [][]float64{
		{1.0, 2.0, 3.0},
		{2.0, 3.0, 4.0},
	}
	labels := []int{0, 1}

	req := &AdversarialTrainingRequest{
		ModelID:         "model-1",
		AdversarialSamples: adversarialSamples,
		Labels:         labels,
		TrainingEpochs: 5,
		LearningRate:   0.001,
	}

	resp, err := service.PerformAdversarialTraining(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.TrainedModel)
	assert.Greater(t, resp.Accuracy, 0.0)
}

func TestHardenModel(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	req := &ModelHardeningRequest{
		ModelID:            "model-1",
		HardeningStrategies: []string{"gaussian_noise", "gradient_masking"},
	}

	resp, err := service.HardenModel(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.HardenedModel)
	assert.NotEmpty(t, resp.StrategiesApplied)
}

func TestMitigateBackdoor(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}, {0.3, 0.4}, {3.0, 4.0}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	req := &BackdoorMitigationRequest{
		ModelID:     "model-1",
		CleanDataset: [][]float64{{1.0, 2.0}},
		CleanLabels:  []int{0},
	}

	resp, err := service.MitigateBackdoor(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.CleanedModel)
}

func TestGenerateFGSMAttack(t *testing.T) {
	service := NewAISecurityEnhancedService()

	input := []float64{1.0, 2.0, 3.0, 4.0}
	epsilon := 0.1

	advSample, err := service.GenerateFGSMAttack(context.Background(), input, epsilon, 0)
	require.NoError(t, err)
	assert.NotNil(t, advSample)
	assert.Equal(t, AdversarialFGSM, advSample.AttackType)
	assert.NotEmpty(t, advSample.AdversarialInput)
	assert.NotEmpty(t, advSample.Perturbation)
}

func TestGeneratePGDAttack(t *testing.T) {
	service := NewAISecurityEnhancedService()

	input := []float64{1.0, 2.0, 3.0, 4.0}
	epsilon := 0.1
	alpha := 0.01
	iterations := 5

	advSample, err := service.GeneratePGDAttack(context.Background(), input, epsilon, alpha, iterations)
	require.NoError(t, err)
	assert.NotNil(t, advSample)
	assert.Equal(t, AdversarialPGD, advSample.AttackType)
	assert.NotEmpty(t, advSample.AdversarialInput)
}

func TestGenerateBackdoorTrigger(t *testing.T) {
	service := NewAISecurityEnhancedService()

	trigger, err := service.GenerateBackdoorTrigger(context.Background(), 10, 1)
	require.NoError(t, err)
	assert.NotNil(t, trigger)
	assert.NotEmpty(t, trigger.Pattern)
	assert.Len(t, trigger.TriggerMask, 10)
	assert.Equal(t, 1, trigger.TargetClass)
}

func TestInjectBackdoor(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	trigger := &BackdoorTrigger{
		TriggerID:   "trigger-1",
		Pattern:     []float64{1.0, 1.0},
		Location:    []int{0, 0},
		TriggerMask: []bool{true, true},
		TargetClass: 1,
	}

	backdooredModel, err := service.InjectBackdoor(context.Background(), "model-1", trigger)
	require.NoError(t, err)
	assert.NotNil(t, backdooredModel)
	assert.Contains(t, backdooredModel.ModelID, "backdoored")
}

func TestPerformSecurityAudit(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	auditTypes := []string{"adversarial", "poisoning", "backdoor"}

	req := &SecurityAuditRequest{
		ModelID:    "model-1",
		AuditTypes: auditTypes,
	}

	resp, err := service.PerformSecurityAudit(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.OverallRisk)
	assert.NotEmpty(t, resp.Recommendations)
}

func TestCompareModels(t *testing.T) {
	service := NewAISecurityEnhancedService()

	modelA := &AIModel{
		ModelID:      "model-a",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	modelB := &AIModel{
		ModelID:      "model-b",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.3, 0.4}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}

	service.RegisterModel(modelA)
	service.RegisterModel(modelB)

	req := &ModelComparisonRequest{
		ModelAID:        "model-a",
		ModelBID:        "model-b",
		ComparisonType: "security",
	}

	resp, err := service.CompareModels(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Recommendation)
}

func TestExportSecurityReport(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	req := &ExportSecurityReportRequest{
		ModelID: "model-1",
		Format:  "json",
	}

	resp, err := service.ExportSecurityReport(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Report)
}

func TestGetModels(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model1 := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	model2 := &AIModel{
		ModelID:      "model-2",
		ModelType:    "detector",
		Weights:      [][]float64{{0.2}},
		Architecture: "yolo",
		Version:      "1.0.0",
	}

	service.RegisterModel(model1)
	service.RegisterModel(model2)

	models := service.GetModels()
	assert.Len(t, models, 2)
}

func TestGetWatermarks(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	watermark := []byte("watermark")
	req := &WatermarkRequest{
		ModelID:       "model-1",
		Watermark:     watermark,
		EmbeddingType: "low_frequency",
		Strength:      0.5,
	}

	_, err := service.EmbedWatermark(context.Background(), req)
	require.NoError(t, err)

	watermarks := service.GetWatermarks()
	assert.Len(t, watermarks, 1)
}

func TestGetModelCount(t *testing.T) {
	service := NewAISecurityEnhancedService()

	for i := 0; i < 3; i++ {
		model := &AIModel{
			ModelID:      "model-" + string(rune('a'+i)),
			ModelType:    "classifier",
			Weights:      [][]float64{{0.1}},
			Architecture: "resnet",
			Version:      "1.0.0",
		}
		service.RegisterModel(model)
	}

	count := service.GetModelCount()
	assert.Equal(t, 3, count)
}

func TestGetWatermarkCount(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	watermark := []byte("watermark")
	req := &WatermarkRequest{
		ModelID:       "model-1",
		Watermark:     watermark,
		EmbeddingType: "low_frequency",
		Strength:      0.5,
	}

	_, err := service.EmbedWatermark(context.Background(), req)
	require.NoError(t, err)

	count := service.GetWatermarkCount()
	assert.Equal(t, 1, count)
}

func TestRegisterDefenseStrategy(t *testing.T) {
	service := NewAISecurityEnhancedService()

	strategy := &DefenseStrategy{
		Name:      "test_strategy",
		Type:      "input_transform",
		Threshold: 0.5,
		IsActive:  true,
		Parameters: map[string]interface{}{
			"noise_level": 0.01,
		},
	}

	err := service.RegisterDefenseStrategy(strategy)
	require.NoError(t, err)

	strategies := service.GetDefenseStrategies()
	assert.NotEmpty(t, strategies)
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	assert.True(t, contains(slice, "a"))
	assert.False(t, contains(slice, "d"))
}

func TestDefenseStrategies(t *testing.T) {
	service := NewAISecurityEnhancedService()

	model := &AIModel{
		ModelID:      "model-1",
		ModelType:    "classifier",
		Weights:      [][]float64{{0.1, 0.2}},
		Architecture: "resnet",
		Version:      "1.0.0",
	}
	service.RegisterModel(model)

	defenseTypes := []string{"gaussian_noise", "input_transform", "gradient_masking"}

	for _, defenseType := range defenseTypes {
		req := &AdversarialDefenseRequest{
			ModelID:     "model-1",
			Input:       []float64{1.0, 2.0},
			DefenseType: defenseType,
			Threshold:   0.5,
		}

		resp, err := service.DefendAgainstAdversarial(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	}
}

func TestAdversarialDefenseEngine(t *testing.T) {
	engine := NewAdversarialDefenseEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.defenseStrategies)
}

func TestPoisoningDetectionEngine(t *testing.T) {
	engine := NewPoisoningDetectionEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.detectors)
}

func TestBackdoorDetectionEngine(t *testing.T) {
	engine := NewBackdoorDetectionEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.triggers)
}

func TestWatermarkEngine(t *testing.T) {
	engine := NewWatermarkEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.watermarks)
}

func TestComputeAdversarialScore(t *testing.T) {
	service := NewAISecurityEnhancedService()

	input := []float64{1.0, 2.0, 3.0}
	weights := [][]float64{{0.1, 0.2, 0.3}}

	score := service.computeAdversarialScore(input, weights)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
}

func TestComputePoisoningScore(t *testing.T) {
	service := NewAISecurityEnhancedService()

	data := []float64{1.0, 2.0, 3.0}
	label := 0
	modelID := "model-1"

	score := service.computePoisoningScore(data, label, modelID)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
}

func TestDetectDataAnomaly(t *testing.T) {
	service := NewAISecurityEnhancedService()

	normalData := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	score := service.detectDataAnomaly(normalData)
	assert.GreaterOrEqual(t, score, 0.0)

	anomalousData := []float64{100.0, 2.0, 3.0, 4.0, 5.0}
	score = service.detectDataAnomaly(anomalousData)
	assert.GreaterOrEqual(t, score, 0.0)
}

func TestGenerateSampleID(t *testing.T) {
	id1 := generateSampleID()
	id2 := generateSampleID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
}

func TestApplyGaussianNoise(t *testing.T) {
	service := NewAISecurityEnhancedService()

	input := []float64{1.0, 2.0, 3.0, 4.0}
	stdDev := 0.1

	noisy := service.addGaussianNoise(input, stdDev)

	assert.Len(t, noisy, len(input))
	for i := range noisy {
		assert.InDelta(t, input[i], noisy[i], 0.5)
	}
}

func TestApplyInputTransformation(t *testing.T) {
	service := NewAISecurityEnhancedService()

	input := []float64{-1.0, 0.0, 1.0, 2.0}

	transformed := service.applyInputTransformation(input)

	assert.Len(t, transformed, len(input))
	for i, v := range transformed {
		expected := math.Tanh(input[i])
		assert.InDelta(t, expected, v, 0.001)
	}
}

func TestComputeGradientMagnitude(t *testing.T) {
	service := NewAISecurityEnhancedService()

	input := []float64{1.0, 2.0, 3.0}
	weights := [][]float64{{0.1, 0.2, 0.3}}

	magnitude := service.computeGradientMagnitude(input, weights)
	assert.GreaterOrEqual(t, magnitude, 0.0)
}

func TestDetectSignificantPerturbation(t *testing.T) {
	service := NewAISecurityEnhancedService()

	smallPerturbation := []float64{0.01, 0.01, 0.01}
	assert.False(t, service.detectSignificantPerturbation(smallPerturbation, 0.1))

	largePerturbation := []float64{0.5, 0.5, 0.5}
	assert.True(t, service.detectSignificantPerturbation(largePerturbation, 0.1))
}
