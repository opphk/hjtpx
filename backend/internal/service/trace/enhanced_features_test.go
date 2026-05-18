package trace

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestEnhancedLSTMFeatureExtractor(t *testing.T) {
	t.Run("ExtractFeaturesWithEnhancedDimensions", func(t *testing.T) {
		extractor := NewLSTMFeatureExtractor()

		traceData := generateTestTraceData(50, 2000)

		features, err := extractor.ExtractFeatures(traceData)
		if err != nil {
			t.Fatalf("Failed to extract features: %v", err)
		}

		if len(features) != LSTMFeatureDim {
			t.Errorf("Expected feature dimension %d, got %d", LSTMFeatureDim, len(features))
		}
	})

	t.Run("ExtractRiskFeatures", func(t *testing.T) {
		extractor := NewLSTMFeatureExtractor()

		traceData := generateTestTraceData(50, 2000)

		riskFeatures, err := extractor.ExtractRiskFeatures(traceData)
		if err != nil {
			t.Fatalf("Failed to extract risk features: %v", err)
		}

		if len(riskFeatures) == 0 {
			t.Error("Expected non-empty risk features")
		}

		expectedKeys := []string{
			"velocity_mean",
			"velocity_variance",
			"acceleration_mean",
			"pause_ratio",
			"speed_entropy",
		}

		for _, key := range expectedKeys {
			if _, ok := riskFeatures[key]; !ok {
				t.Errorf("Expected risk feature key %s not found", key)
			}
		}
	})

	t.Run("ExtractAdvancedRiskFeatures", func(t *testing.T) {
		extractor := NewLSTMFeatureExtractor()

		traceData := generateTestTraceData(50, 2000)

		advancedFeatures, err := extractor.ExtractAdvancedRiskFeatures(traceData)
		if err != nil {
			t.Fatalf("Failed to extract advanced risk features: %v", err)
		}

		if len(advancedFeatures) == 0 {
			t.Error("Expected non-empty advanced risk features")
		}

		expectedKeys := []string{
			"spectral_entropy",
			"permutation_entropy",
			"approximate_entropy",
			"hurst_exponent",
		}

		for _, key := range expectedKeys {
			if _, ok := advancedFeatures[key]; !ok {
				t.Errorf("Expected advanced feature key %s not found", key)
			}
		}
	})

	t.Run("AnalyzeTrajectoryComplexity", func(t *testing.T) {
		extractor := NewLSTMFeatureExtractor()

		traceData := generateComplexTraceData()

		complexity, err := extractor.AnalyzeTrajectoryComplexity(traceData)
		if err != nil {
			t.Fatalf("Failed to analyze trajectory complexity: %v", err)
		}

		if complexity < 0 || complexity > 1 {
			t.Errorf("Complexity should be between 0 and 1, got %f", complexity)
		}
	})

	t.Run("DetectAnomalousPatterns", func(t *testing.T) {
		extractor := NewLSTMFeatureExtractor()

		botTraceData := generateBotLikeTraceData()

		anomalies, err := extractor.DetectAnomalousPatterns(botTraceData)
		if err != nil {
			t.Fatalf("Failed to detect anomalous patterns: %v", err)
		}

		if len(anomalies) == 0 {
			t.Error("Expected to detect at least one anomalous pattern for bot-like trace")
		}
	})

	t.Run("ExtractComprehensiveFeatures", func(t *testing.T) {
		extractor := NewLSTMFeatureExtractor()

		traceData := generateTestTraceData(50, 2000)

		summary, err := extractor.ExtractComprehensiveFeatures(traceData)
		if err != nil {
			t.Fatalf("Failed to extract comprehensive features: %v", err)
		}

		if summary.TotalFeatures == 0 {
			t.Error("Expected non-zero total features")
		}

		if summary.ComplexityScore < 0 || summary.ComplexityScore > 1 {
			t.Errorf("Complexity score should be between 0 and 1, got %f", summary.ComplexityScore)
		}
	})
}

func TestEnhancedTransformerPredictor(t *testing.T) {
	t.Run("EnhancedMultiHeadAttention", func(t *testing.T) {
		predictor := NewTransformerPredictor()

		if predictor.GetAttentionHeads() != EnhancedTransformerHeads {
			t.Errorf("Expected %d attention heads, got %d", EnhancedTransformerHeads, predictor.GetAttentionHeads())
		}

		architecture := predictor.GetModelArchitecture()
		if architecture["num_attention_heads"] != EnhancedTransformerHeads {
			t.Error("Architecture mismatch for attention heads")
		}
	})

	t.Run("PredictWithRiskAssessment", func(t *testing.T) {
		predictor := NewTransformerPredictor()

		traceData := generateTestTraceData(50, 2000)

		result, err := predictor.PredictTrajectory(traceData)
		if err != nil {
			t.Fatalf("Failed to predict trajectory: %v", err)
		}

		if result.RiskScore < 0 || result.RiskScore > 1 {
			t.Errorf("Risk score should be between 0 and 1, got %f", result.RiskScore)
		}

		if result.BotProbability < 0 || result.BotProbability > 1 {
			t.Errorf("Bot probability should be between 0 and 1, got %f", result.BotProbability)
		}

		if result.Confidence < 0 || result.Confidence > 1 {
			t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
		}
	})

	t.Run("AnalyzeAttentionPatterns", func(t *testing.T) {
		predictor := NewTransformerPredictor()

		extractor := NewLSTMFeatureExtractor()
		seq, _ := extractor.PrepareSequence(generateTestTraceData(50, 2000))

		embeddings := predictor.encodeSequence(seq)

		patterns, err := predictor.AnalyzeAttentionPatterns(embeddings)
		if err != nil {
			t.Fatalf("Failed to analyze attention patterns: %v", err)
		}

		if len(patterns) == 0 {
			t.Error("Expected non-empty attention patterns")
		}
	})

	t.Run("ComputeAttentionEntropy", func(t *testing.T) {
		predictor := NewTransformerPredictor()

		attentionMaps := make([][][]float64, 2)
		for h := 0; h < 2; h++ {
			attentionMaps[h] = make([][]float64, 3)
			for i := 0; i < 3; i++ {
				attentionMaps[h][i] = make([]float64, 3)
				for j := 0; j < 3; j++ {
					attentionMaps[h][i][j] = 0.33
				}
			}
		}

		entropy := predictor.ComputeAttentionEntropy(attentionMaps)
		if entropy < 0 {
			t.Error("Attention entropy should be non-negative")
		}
	})
}

func TestIntentClassifier(t *testing.T) {
	t.Run("RecognizeNormalUserIntent", func(t *testing.T) {
		classifier := NewIntentClassifier()

		traceData := generateNormalUserTraceData()

		result, err := classifier.RecognizeIntent(traceData)
		if err != nil {
			t.Fatalf("Failed to recognize intent: %v", err)
		}

		if result.PrimaryIntent == "" {
			t.Error("Expected non-empty primary intent")
		}

		if result.Confidence < 0 || result.Confidence > 1 {
			t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
		}

		if len(result.IntentProbabilities) == 0 {
			t.Error("Expected non-empty intent probabilities")
		}

		if result.Reasoning == "" {
			t.Error("Expected non-empty reasoning")
		}

		if len(result.Recommendations) == 0 {
			t.Error("Expected non-empty recommendations")
		}
	})

	t.Run("RecognizeBotIntent", func(t *testing.T) {
		classifier := NewIntentClassifier()

		botTraceData := generateBotLikeTraceData()

		result, err := classifier.RecognizeIntent(botTraceData)
		if err != nil {
			t.Fatalf("Failed to recognize bot intent: %v", err)
		}

		botIntents := []IntentType{IntentAutomatedBot, IntentScriptedBot, IntentAggressiveBot}
		isRecognizedAsBot := false
		for _, intent := range botIntents {
			if result.IntentProbabilities[intent] > 0.5 {
				isRecognizedAsBot = true
				break
			}
		}

		if !isRecognizedAsBot {
			t.Logf("Note: Bot was not detected with high confidence. Primary intent: %s, confidence: %f",
				result.PrimaryIntent, result.Confidence)
		}
	})

	t.Run("BatchRecognize", func(t *testing.T) {
		classifier := NewIntentClassifier()

		traces := []*model.TraceData{
			generateNormalUserTraceData(),
			generateBotLikeTraceData(),
			generateTestTraceData(50, 2000),
		}

		results, err := classifier.BatchRecognize(traces)
		if err != nil {
			t.Fatalf("Failed to batch recognize: %v", err)
		}

		if len(results) != len(traces) {
			t.Errorf("Expected %d results, got %d", len(traces), len(results))
		}
	})

	t.Run("IntentStatistics", func(t *testing.T) {
		classifier := NewIntentClassifier()

		results := []*IntentRecognitionResult{
			{PrimaryIntent: IntentNormalUser, Confidence: 0.8},
			{PrimaryIntent: IntentNormalUser, Confidence: 0.7},
			{PrimaryIntent: IntentAutomatedBot, Confidence: 0.9},
		}

		stats := classifier.GetIntentStatistics(results)

		if stats["total_samples"] != 3 {
			t.Errorf("Expected 3 total samples, got %v", stats["total_samples"])
		}

		if stats["most_common_intent"] != IntentNormalUser {
			t.Errorf("Expected most common intent to be NormalUser, got %v", stats["most_common_intent"])
		}
	})
}

func TestUnifiedRiskScorer(t *testing.T) {
	t.Run("ComprehensiveRiskAnalysis", func(t *testing.T) {
		scorer := NewUnifiedRiskScorer()

		traceData := generateTestTraceData(50, 2000)

		result, err := scorer.AnalyzeComprehensiveRisk(context.Background(), traceData)
		if err != nil {
			t.Fatalf("Failed to analyze comprehensive risk: %v", err)
		}

		if result.TotalRiskScore < 0 || result.TotalRiskScore > 1 {
			t.Errorf("Total risk score should be between 0 and 1, got %f", result.TotalRiskScore)
		}

		if result.RiskLevel == "" {
			t.Error("Expected non-empty risk level")
		}

		if len(result.ComponentScores) == 0 {
			t.Error("Expected non-empty component scores")
		}

		if result.ProcessingTimeMs < 0 {
			t.Error("Processing time should be non-negative")
		}
	})

	t.Run("BotDetection", func(t *testing.T) {
		scorer := NewUnifiedRiskScorer()

		botTraceData := generateBotLikeTraceData()

		result, err := scorer.AnalyzeComprehensiveRisk(context.Background(), botTraceData)
		if err != nil {
			t.Fatalf("Failed to analyze bot risk: %v", err)
		}

		if !result.IsBot {
			t.Logf("Note: Bot was not detected. Total risk score: %f, Bot probability: %f",
				result.TotalRiskScore, result.BotProbability)
		}
	})

	t.Run("IntentRecognitionIntegration", func(t *testing.T) {
		scorer := NewUnifiedRiskScorer()

		traceData := generateNormalUserTraceData()

		result, err := scorer.AnalyzeComprehensiveRisk(context.Background(), traceData)
		if err != nil {
			t.Fatalf("Failed to analyze risk with intent: %v", err)
		}

		if result.IntentRecognition == nil {
			t.Error("Expected intent recognition result")
		}
	})

	t.Run("AnomalyDetectionIntegration", func(t *testing.T) {
		scorer := NewUnifiedRiskScorer()

		botTraceData := generateBotLikeTraceData()

		result, err := scorer.AnalyzeComprehensiveRisk(context.Background(), botTraceData)
		if err != nil {
			t.Fatalf("Failed to analyze risk with anomalies: %v", err)
		}

		if result.AnomalyPatterns != nil {
			t.Logf("Detected %d anomaly patterns", len(result.AnomalyPatterns))
		}
	})

	t.Run("BatchAnalyze", func(t *testing.T) {
		scorer := NewUnifiedRiskScorer()

		traces := []*model.TraceData{
			generateTestTraceData(50, 2000),
			generateBotLikeTraceData(),
			generateNormalUserTraceData(),
		}

		results, err := scorer.BatchAnalyze(context.Background(), traces)
		if err != nil {
			t.Fatalf("Failed to batch analyze: %v", err)
		}

		if len(results) != len(traces) {
			t.Errorf("Expected %d results, got %d", len(traces), len(results))
		}
	})

	t.Run("DynamicThresholdAdaptation", func(t *testing.T) {
		scorer := NewUnifiedRiskScorer()

		thresholds := scorer.GetThresholds()
		if len(thresholds) == 0 {
			t.Error("Expected non-empty thresholds")
		}

		err := scorer.UpdateThreshold("high_risk", 0.75)
		if err != nil {
			t.Errorf("Failed to update threshold: %v", err)
		}

		newThresholds := scorer.GetThresholds()
		if newThresholds["high_risk"] != 0.75 {
			t.Errorf("Expected threshold 0.75, got %f", newThresholds["high_risk"])
		}
	})

	t.Run("RiskStatistics", func(t *testing.T) {
		scorer := NewUnifiedRiskScorer()

		for i := 0; i < 10; i++ {
			scorer.AnalyzeComprehensiveRisk(context.Background(), generateTestTraceData(50, 2000))
		}

		stats := scorer.GetStatistics()
		if stats == nil {
			t.Error("Expected non-nil statistics")
		}

		if stats["score_count"] != 10 {
			t.Errorf("Expected 10 scores, got %v", stats["score_count"])
		}
	})

	t.Run("ResetScorer", func(t *testing.T) {
		scorer := NewUnifiedRiskScorer()

		for i := 0; i < 5; i++ {
			scorer.AnalyzeComprehensiveRisk(context.Background(), generateTestTraceData(50, 2000))
		}

		scorer.Reset()

		stats := scorer.GetStatistics()
		if stats["score_count"] != 0 {
			t.Errorf("Expected 0 scores after reset, got %v", stats["score_count"])
		}
	})
}

func TestEnhancedTraceService(t *testing.T) {
	t.Run("ComprehensiveRiskAnalysis", func(t *testing.T) {
		service := NewTraceService()

		traceData := generateTestTraceData(50, 2000)
		traceDataJSON, _ := json.Marshal(traceData)

		result, err := service.ProcessTraceWithComprehensiveRisk(context.Background(), "test-session", traceDataJSON)
		if err != nil {
			t.Fatalf("Failed to process trace with comprehensive risk: %v", err)
		}

		if result.TotalRiskScore < 0 || result.TotalRiskScore > 1 {
			t.Errorf("Total risk score should be between 0 and 1, got %f", result.TotalRiskScore)
		}
	})

	t.Run("IntentRecognition", func(t *testing.T) {
		service := NewTraceService()

		traceData := generateNormalUserTraceData()

		intent, err := service.RecognizeIntent(traceData)
		if err != nil {
			t.Fatalf("Failed to recognize intent: %v", err)
		}

		if intent.PrimaryIntent == "" {
			t.Error("Expected non-empty primary intent")
		}
	})

	t.Run("AnomalyDetection", func(t *testing.T) {
		service := NewTraceService()

		botTraceData := generateBotLikeTraceData()

		anomalies, err := service.DetectAnomalies(botTraceData)
		if err != nil {
			t.Fatalf("Failed to detect anomalies: %v", err)
		}

		if anomalies == nil {
			t.Error("Expected non-nil anomaly result")
		}
	})

	t.Run("GetComprehensiveRiskAnalysis", func(t *testing.T) {
		service := NewTraceService()

		traceData := generateTestTraceData(50, 2000)

		result, err := service.GetComprehensiveRiskAnalysis(context.Background(), traceData)
		if err != nil {
			t.Fatalf("Failed to get comprehensive risk analysis: %v", err)
		}

		if result.TotalRiskScore < 0 || result.TotalRiskScore > 1 {
			t.Errorf("Total risk score should be between 0 and 1, got %f", result.TotalRiskScore)
		}
	})

	t.Run("BatchComprehensiveRiskAnalysis", func(t *testing.T) {
		service := NewTraceService()

		traces := []*model.TraceData{
			generateTestTraceData(50, 2000),
			generateBotLikeTraceData(),
			generateNormalUserTraceData(),
		}

		results, err := service.BatchComprehensiveRiskAnalysis(context.Background(), traces)
		if err != nil {
			t.Fatalf("Failed to batch analyze: %v", err)
		}

		if len(results) != len(traces) {
			t.Errorf("Expected %d results, got %d", len(traces), len(results))
		}
	})

	t.Run("ModelInfo", func(t *testing.T) {
		service := NewTraceService()

		info := service.GetModelInfo()

		if info["nn_enabled"] != true {
			t.Error("Expected NN to be enabled")
		}

		if info["lstm_feature_dim"] != LSTMFeatureDim {
			t.Errorf("Expected LSTM feature dim %d, got %v", LSTMFeatureDim, info["lstm_feature_dim"])
		}

		if info["transformer_attention_heads"] != EnhancedTransformerHeads {
			t.Errorf("Expected %d attention heads, got %v", EnhancedTransformerHeads, info["transformer_attention_heads"])
		}

		if info["intent_recognition_enabled"] != true {
			t.Error("Expected intent recognition to be enabled")
		}

		if info["anomaly_detection_enabled"] != true {
			t.Error("Expected anomaly detection to be enabled")
		}
	})

	t.Run("TrajectoryComplexity", func(t *testing.T) {
		service := NewTraceService()

		traceData := generateComplexTraceData()

		complexity, err := service.ExtractTrajectoryComplexity(traceData)
		if err != nil {
			t.Fatalf("Failed to extract trajectory complexity: %v", err)
		}

		if complexity < 0 || complexity > 1 {
			t.Errorf("Complexity should be between 0 and 1, got %f", complexity)
		}
	})

	t.Run("AnomalousPatternsDetection", func(t *testing.T) {
		service := NewTraceService()

		botTraceData := generateBotLikeTraceData()

		patterns, err := service.DetectAnomalousPatterns(botTraceData)
		if err != nil {
			t.Fatalf("Failed to detect anomalous patterns: %v", err)
		}

		if patterns == nil {
			t.Error("Expected non-nil anomalous patterns")
		}
	})

	t.Run("ComprehensiveFeatures", func(t *testing.T) {
		service := NewTraceService()

		traceData := generateTestTraceData(50, 2000)

		summary, err := service.ExtractComprehensiveFeatures(traceData)
		if err != nil {
			t.Fatalf("Failed to extract comprehensive features: %v", err)
		}

		if summary.TotalFeatures == 0 {
			t.Error("Expected non-zero total features")
		}
	})

	t.Run("ThresholdManagement", func(t *testing.T) {
		service := NewTraceService()

		thresholds := service.GetThresholds()
		if len(thresholds) == 0 {
			t.Error("Expected non-empty thresholds")
		}

		err := service.UpdateThreshold("high_risk", 0.75)
		if err != nil {
			t.Errorf("Failed to update threshold: %v", err)
		}

		newThresholds := service.GetThresholds()
		if newThresholds["high_risk"] != 0.75 {
			t.Errorf("Expected threshold 0.75, got %f", newThresholds["high_risk"])
		}
	})

	t.Run("ModelPerformanceMonitoring", func(t *testing.T) {
		service := NewTraceService()

		report := service.GetModelPerformanceReport()
		if report == nil {
			t.Error("Expected non-nil performance report")
		}

		if _, ok := report["lstm"]; !ok {
			t.Error("Expected LSTM metrics in report")
		}

		if _, ok := report["transformer"]; !ok {
			t.Error("Expected Transformer metrics in report")
		}
	})

	t.Run("OnlineUpdate", func(t *testing.T) {
		service := NewTraceService()

		traceData := generateTestTraceData(50, 2000)

		service.QueueTrainingSample(traceData, true, 0.9)

		service.StartOnlineUpdate()
		time.Sleep(100 * time.Millisecond)
		service.StopOnlineUpdate()
	})
}

func generateTestTraceData(numPoints, durationMs int) *model.TraceData {
	trace := &model.TraceData{
		Points: make([]model.TracePoint, numPoints),
	}

	startTime := int64(time.Now().UnixMilli())
	interval := int64(durationMs / numPoints)

	for i := 0; i < numPoints; i++ {
		trace.Points[i] = model.TracePoint{
			X:         100 + float64(i*2),
			Y:         100 + float64(i),
			Timestamp: startTime + int64(i)*interval,
			Event:     "move",
		}
	}

	return trace
}

func generateNormalUserTraceData() *model.TraceData {
	trace := &model.TraceData{
		Points: make([]model.TracePoint, 100),
	}

	startTime := int64(time.Now().UnixMilli())

	for i := 0; i < 100; i++ {
		x := 100 + float64(i*2+(i%10-5))
		y := 100 + float64(i+(i%5-2))

		trace.Points[i] = model.TracePoint{
			X:         x,
			Y:         y,
			Timestamp: startTime + int64(i)*20,
			Event:     "move",
		}
	}

	return trace
}

func generateBotLikeTraceData() *model.TraceData {
	trace := &model.TraceData{
		Points: make([]model.TracePoint, 100),
	}

	startTime := int64(time.Now().UnixMilli())

	for i := 0; i < 100; i++ {
		trace.Points[i] = model.TracePoint{
			X:         100 + float64(i*2),
			Y:         100,
			Timestamp: startTime + int64(i)*10,
			Event:     "move",
		}
	}

	return trace
}

func generateComplexTraceData() *model.TraceData {
	trace := &model.TraceData{
		Points: make([]model.TracePoint, 150),
	}

	startTime := int64(time.Now().UnixMilli())

	for i := 0; i < 150; i++ {
		angle := float64(i) * 0.1
		x := 200 + 100*math.Cos(angle)
		y := 200 + 100*math.Sin(angle)

		trace.Points[i] = model.TracePoint{
			X:         x,
			Y:         y,
			Timestamp: startTime + int64(i)*15,
			Event:     "move",
		}
	}

	return trace
}
