package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAIDefenseHub(t *testing.T) {
	hub := NewAIDefenseHub()
	assert.NotNil(t, hub)
	assert.NotNil(t, hub.adversarialDefense)
	assert.NotNil(t, hub.poisoningDetector)
	assert.NotNil(t, hub.watermarking)
}

func TestApplyGaussianNoiseDefense(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	
	req := &DefenseRequest{
		Input:         input,
		DefenseMethod: DefenseGaussianNoise,
		Parameters: map[string]interface{}{
			"mean":   0.0,
			"stddev": 0.01,
		},
	}

	result, err := hub.ApplyDefense(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, DefenseGaussianNoise, result.AppliedDefense)
	assert.NotEqual(t, input, result.DefendedInput)
}

func TestApplyInputTransformDefense(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{0.5, -0.5, 1.5, -1.5}

	req := &DefenseRequest{
		Input:         input,
		DefenseMethod: DefenseInputTransform,
		Parameters: map[string]interface{}{
			"transform_type": "tanh",
		},
	}

	result, err := hub.ApplyDefense(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, DefenseInputTransform, result.AppliedDefense)

	for _, v := range result.DefendedInput {
		assert.LessOrEqual(t, v, 1.0)
		assert.GreaterOrEqual(t, v, -1.0)
	}
}

func TestApplyFeatureSqueezeDefense(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.234, 2.567, 3.8910, 4.1234}

	req := &DefenseRequest{
		Input:         input,
		DefenseMethod: DefenseFeatureSqueeze,
		Parameters: map[string]interface{}{
			"bit_depth": 5,
		},
	}

	result, err := hub.ApplyDefense(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, DefenseFeatureSqueeze, result.AppliedDefense)
}

func TestApplyJPEGCompressionDefense(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.1, 2.2, 3.3, 4.4, 5.5}

	req := &DefenseRequest{
		Input:         input,
		DefenseMethod: DefenseJPEGCompression,
		Parameters: map[string]interface{}{
			"quality": 75,
		},
	}

	result, err := hub.ApplyDefense(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestApplyRandomizationDefense(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	req := &DefenseRequest{
		Input:         input,
		DefenseMethod: DefenseRandomization,
		Parameters: map[string]interface{}{
			"resize_range": []float64{0.9, 1.1},
			"pad_size":     4,
		},
	}

	result, err := hub.ApplyDefense(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestDetectAdversarial(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	benignInput := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	result, err := hub.DetectAdversarial(ctx, benignInput)
	require.NoError(t, err)
	assert.NotNil(t, result)

	adversarialInput := []float64{10.0, 20.0, 30.0, 40.0, 50.0}

	result2, err := hub.DetectAdversarial(ctx, adversarialInput)
	require.NoError(t, err)
	assert.NotNil(t, result2)
}

func TestGenerateFGSMAdversarial(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	result, err := hub.GenerateAdversarial(ctx, input, 0.1, "fgsm", 1)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "fgsm", result.AttackType)
	assert.Equal(t, 0.1, result.Epsilon)
	assert.Len(t, result.AdversarialInput, len(input))
}

func TestGeneratePGDAdversarial(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	result, err := hub.GenerateAdversarial(ctx, input, 0.3, "pgd", 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "pgd", result.AttackType)
	assert.Equal(t, 10, result.Iterations)
}

func TestGenerateCarliniAdversarial(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	result, err := hub.GenerateAdversarial(ctx, input, 0.1, "carlini", 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "carlini", result.AttackType)
}

func TestGenerateDeepFoolAdversarial(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	result, err := hub.GenerateAdversarial(ctx, input, 0.1, "deepfool", 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "deepfool", result.AttackType)
}

func TestRegisterModelForPoisoning(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	modelID := "model-test-1"
	trainingData := [][]float64{
		{1.0, 2.0, 3.0},
		{2.0, 3.0, 4.0},
		{3.0, 4.0, 5.0},
	}
	labels := []int{0, 1, 1}

	err := hub.RegisterModelForPoisoning(ctx, modelID, trainingData, labels, nil)
	require.NoError(t, err)
}

func TestDetectPoisoning(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	modelID := "model-poisoning-test"
	trainingData := [][]float64{
		{1.0, 2.0, 3.0},
		{2.0, 3.0, 4.0},
		{10.0, 20.0, 30.0},
		{15.0, 25.0, 35.0},
	}
	labels := []int{0, 1, 0, 1}

	err := hub.RegisterModelForPoisoning(ctx, modelID, trainingData, labels, nil)
	require.NoError(t, err)

	report, err := hub.DetectModelPoisoning(ctx, modelID)
	require.NoError(t, err)
	assert.NotNil(t, report)
}

func TestAnalyzeTrainingData(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	modelID := "model-analysis-test"
	trainingData := [][]float64{
		{1.0, 2.0, 3.0},
		{2.0, 3.0, 4.0},
		{3.0, 4.0, 5.0},
	}
	labels := []int{0, 1, 1}

	err := hub.RegisterModelForPoisoning(ctx, modelID, trainingData, labels, nil)
	require.NoError(t, err)

	analysis, err := hub.AnalyzeTrainingData(ctx, modelID)
	require.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.Equal(t, 3, analysis.TotalSamples)
}

func TestEmbedWatermark(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	modelID := "model-watermark-test"
	message := []byte("watermark-message-123")

	watermark, err := hub.EmbedModelWatermark(ctx, modelID, message, "robust")
	require.NoError(t, err)
	assert.NotNil(t, watermark)
	assert.Equal(t, WatermarkTypeRobust, watermark.Type)
}

func TestVerifyWatermark(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	modelID := "model-verify-watermark"
	message := []byte("verify-watermark-message")

	watermark, err := hub.EmbedModelWatermark(ctx, modelID, message, "robust")
	require.NoError(t, err)

	result, err := hub.VerifyModelWatermark(ctx, modelID, message)
	require.NoError(t, err)
	assert.True(t, result.IsValid)
	assert.Equal(t, watermark.Hash, result.ExtractedHash)
}

func TestEmbedMultipleWatermarks(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	modelID := "model-multi-watermark"

	watermark1, err := hub.EmbedModelWatermark(ctx, modelID, []byte("watermark1"), "robust")
	require.NoError(t, err)

	watermark2, err := hub.EmbedModelWatermark(ctx, modelID, []byte("watermark2"), "fragile")
	require.NoError(t, err)

	assert.NotEqual(t, watermark1.ID, watermark2.ID)
	assert.NotEqual(t, watermark1.Type, watermark2.Type)
}

func TestRemoveWatermark(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	modelID := "model-remove-watermark"
	message := []byte("watermark-to-remove")

	_, err := hub.EmbedModelWatermark(ctx, modelID, message, "robust")
	require.NoError(t, err)

	result, err := hub.RemoveModelWatermark(ctx, modelID)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestGetAvailableDefenses(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	defenses := hub.GetAvailableDefenses(ctx)
	assert.NotEmpty(t, defenses)
	assert.GreaterOrEqual(t, len(defenses), 5)
}

func TestEnableDisableDefense(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	err := hub.DisableDefense(ctx, DefenseGaussianNoise)
	require.NoError(t, err)

	defenses := hub.GetAvailableDefenses(ctx)
	for _, d := range defenses {
		if d.Method == DefenseGaussianNoise {
			assert.False(t, d.IsActive)
		}
	}

	err = hub.EnableDefense(ctx, DefenseGaussianNoise)
	require.NoError(t, err)
}

func TestGenerateAdversarialWithDifferentEpsilons(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	epsilons := []float64{0.01, 0.1, 0.5, 1.0}

	for _, eps := range epsilons {
		result, err := hub.GenerateAdversarial(ctx, input, eps, "fgsm", 0)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, eps, result.Epsilon)
	}
}

func TestDefenseWithDifferentMethods(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	methods := []DefenseMethod{
		DefenseGaussianNoise,
		DefenseInputTransform,
		DefenseFeatureSqueeze,
		DefenseMagNet,
		DefenseJPEGCompression,
		DefenseRandomization,
	}

	for _, method := range methods {
		req := &DefenseRequest{
			Input:         input,
			DefenseMethod: method,
		}

		result, err := hub.ApplyDefense(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, method, result.AppliedDefense)
	}
}

func TestPoisoningDetectionWithDifferentAttackTypes(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	attackTypes := []string{"label_flip", "backdoor", "semantic", "clean_label"}

	for _, attackType := range attackTypes {
		modelID := "model-attack-" + attackType
		trainingData := [][]float64{
			{1.0, 2.0, 3.0},
			{2.0, 3.0, 4.0},
			{10.0, 20.0, 30.0},
		}
		labels := []int{0, 1, 0}

		err := hub.RegisterModelForPoisoning(ctx, modelID, trainingData, labels, nil)
		require.NoError(t, err)

		report, err := hub.DetectModelPoisoning(ctx, modelID)
		require.NoError(t, err)
		assert.NotNil(t, report)
	}

	_ = attackType
}

func TestWatermarkWithDifferentTypes(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	types := []string{"robust", "fragile", "semi_fragile"}

	for _, wmType := range types {
		modelID := "model-wm-" + wmType
		message := []byte("watermark-" + wmType)

		watermark, err := hub.EmbedModelWatermark(ctx, modelID, message, wmType)
		require.NoError(t, err)
		assert.NotNil(t, watermark)
	}

	_ = types
}

func TestDefenseResultMetadata(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	req := &DefenseRequest{
		Input:          input,
		DefenseMethod:  DefenseGaussianNoise,
		ReturnDetails: true,
	}

	result, err := hub.ApplyDefense(ctx, req)
	require.NoError(t, err)
	assert.Greater(t, result.ProcessingTime, time.Duration(0))
}

func TestMultipleModelsPoisoningDetection(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		modelID := "model-multi-" + string(rune('0'+i))
		trainingData := [][]float64{
			{1.0, 2.0},
			{2.0, 3.0},
			{3.0, 4.0},
		}
		labels := []int{0, 1, 1}

		err := hub.RegisterModelForPoisoning(ctx, modelID, trainingData, labels, nil)
		require.NoError(t, err)

		report, err := hub.DetectModelPoisoning(ctx, modelID)
		require.NoError(t, err)
		assert.NotNil(t, report)
	}
}

func TestGetServiceStatistics(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	stats := hub.GetServiceStatistics(ctx)
	assert.NotNil(t, stats)
}

func TestDefenseWithInvalidMethod(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	req := &DefenseRequest{
		Input:         input,
		DefenseMethod: "invalid_defense_method",
	}

	_, err := hub.ApplyDefense(ctx, req)
	assert.Error(t, err)
}

func TestGenerateAdversarialWithInvalidType(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	_, err := hub.GenerateAdversarial(ctx, input, 0.1, "invalid_attack", 0)
	assert.Error(t, err)
}

func TestVerifyWatermarkWithWrongMessage(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	modelID := "model-wrong-verify"
	originalMessage := []byte("original-message")
	wrongMessage := []byte("wrong-message")

	_, err := hub.EmbedModelWatermark(ctx, modelID, originalMessage, "robust")
	require.NoError(t, err)

	result, err := hub.VerifyModelWatermark(ctx, modelID, wrongMessage)
	require.NoError(t, err)
	assert.False(t, result.IsValid)
}

func TestPoisoningDetectionWithCleanData(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	modelID := "model-clean"
	trainingData := [][]float64{
		{1.0, 2.0, 3.0},
		{2.0, 3.0, 4.0},
		{3.0, 4.0, 5.0},
		{4.0, 5.0, 6.0},
		{5.0, 6.0, 7.0},
	}
	labels := []int{0, 1, 1, 0, 1}

	err := hub.RegisterModelForPoisoning(ctx, modelID, trainingData, labels, nil)
	require.NoError(t, err)

	report, err := hub.DetectModelPoisoning(ctx, modelID)
	require.NoError(t, err)
	assert.NotNil(t, report)
}

func TestWatermarkExtraction(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	modelID := "model-extract"
	message := []byte("extract-this-watermark")

	_, err := hub.EmbedModelWatermark(ctx, modelID, message, "robust")
	require.NoError(t, err)

	extracted, err := hub.ExtractWatermark(ctx, modelID)
	require.NoError(t, err)
	assert.NotNil(t, extracted)
	assert.Equal(t, WatermarkTypeRobust, extracted.Type)
}

func TestDefenseStrengthVariations(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	strengths := []float64{0.01, 0.1, 0.5, 1.0}

	for _, strength := range strengths {
		req := &DefenseRequest{
			Input:         input,
			DefenseMethod: DefenseGaussianNoise,
			Parameters: map[string]interface{}{
				"stddev": strength,
			},
		}

		result, err := hub.ApplyDefense(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, result)
	}

	_ = strengths
}

func TestAdversarialDetectionConsistency(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	result1, err := hub.DetectAdversarial(ctx, input)
	require.NoError(t, err)

	result2, err := hub.DetectAdversarial(ctx, input)
	require.NoError(t, err)

	assert.Equal(t, result1.AdversarialScore, result2.AdversarialScore)
}

func TestMultipleVerificationAttempts(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	modelID := "model-multi-verify"
	message := []byte("multi-verify-message")

	_, err := hub.EmbedModelWatermark(ctx, modelID, message, "robust")
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		result, err := hub.VerifyModelWatermark(ctx, modelID, message)
		require.NoError(t, err)
		assert.True(t, result.IsValid)
	}
}

func TestDefensePerformance(t *testing.T) {
	hub := NewAIDefenseHub()
	ctx := context.Background()

	input := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	req := &DefenseRequest{
		Input:         input,
		DefenseMethod: DefenseGaussianNoise,
	}

	start := time.Now()
	result, err := hub.ApplyDefense(ctx, req)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Less(t, duration, 5*time.Second)
}
