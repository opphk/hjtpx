package trace

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestLSTMFeatureExtractor(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "move"},
			{Timestamp: 1300, X: 30, Y: 30, Event: "move"},
			{Timestamp: 1400, X: 40, Y: 40, Event: "end"},
		},
		TotalTime: 400,
		StartX:    0,
		StartY:    0,
		EndX:      40,
		EndY:      40,
	}

	t.Run("Extract Features", func(t *testing.T) {
		features, err := extractor.ExtractFeatures(traceData)
		if err != nil {
			t.Fatalf("ExtractFeatures failed: %v", err)
		}

		if features == nil {
			t.Fatal("Features should not be nil")
		}

		if len(features) != LSTMFeatureDim {
			t.Errorf("Expected feature dimension %d, got %d", LSTMFeatureDim, len(features))
		}

		t.Logf("Feature vector length: %d", len(features))
	})

	t.Run("Prepare Sequence", func(t *testing.T) {
		seq, err := extractor.PrepareSequence(traceData)
		if err != nil {
			t.Fatalf("PrepareSequence failed: %v", err)
		}

		if seq == nil {
			t.Fatal("Sequence should not be nil")
		}

		if len(seq.NormalizedSeq) != len(traceData.Points) {
			t.Errorf("Normalized sequence length mismatch")
		}

		t.Logf("Sequence prepared successfully, points: %d", len(seq.Points))
	})

	t.Run("Extract Risk Features", func(t *testing.T) {
		riskFeatures, err := extractor.ExtractRiskFeatures(traceData)
		if err != nil {
			t.Fatalf("ExtractRiskFeatures failed: %v", err)
		}

		if riskFeatures == nil {
			t.Fatal("Risk features should not be nil")
		}

		if len(riskFeatures) == 0 {
			t.Error("Risk features should not be empty")
		}

		t.Logf("Risk features extracted: %v", riskFeatures)
	})
}

func TestLSTMInsufficientData(t *testing.T) {
	extractor := NewLSTMFeatureExtractor()

	t.Run("Empty Data", func(t *testing.T) {
		traceData := &model.TraceData{
			Points: []model.TracePoint{},
		}

		_, err := extractor.ExtractFeatures(traceData)
		if err == nil {
			t.Error("Expected error for empty data")
		}
	})

	t.Run("Single Point", func(t *testing.T) {
		traceData := &model.TraceData{
			Points: []model.TracePoint{
				{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			},
		}

		_, err := extractor.ExtractFeatures(traceData)
		if err == nil {
			t.Error("Expected error for single point")
		}
	})
}

func TestTransformerPredictor(t *testing.T) {
	predictor := NewTransformerPredictor()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 25, Y: 30, Event: "move"},
			{Timestamp: 1300, X: 50, Y: 60, Event: "move"},
			{Timestamp: 1400, X: 80, Y: 100, Event: "end"},
		},
		TotalTime: 400,
	}

	t.Run("Predict Trajectory", func(t *testing.T) {
		result, err := predictor.PredictTrajectory(traceData)
		if err != nil {
			t.Fatalf("PredictTrajectory failed: %v", err)
		}

		if result == nil {
			t.Fatal("Prediction result should not be nil")
		}

		if result.RiskScore < 0 || result.RiskScore > 1 {
			t.Errorf("Risk score should be between 0 and 1, got %f", result.RiskScore)
		}

		if result.BotProbability < 0 || result.BotProbability > 1 {
			t.Errorf("Bot probability should be between 0 and 1, got %f", result.BotProbability)
		}

		if result.HumanProbability < 0 || result.HumanProbability > 1 {
			t.Errorf("Human probability should be between 0 and 1, got %f", result.HumanProbability)
		}

		t.Logf("Risk Score: %f, Bot Prob: %f, Human Prob: %f",
			result.RiskScore, result.BotProbability, result.HumanProbability)
	})

	t.Run("Predict with Features", func(t *testing.T) {
		features := make([]float64, LSTMFeatureDim)
		for i := range features {
			features[i] = float64(i) / float64(len(features))
		}

		result, err := predictor.PredictWithFeatures(features)
		if err != nil {
			t.Fatalf("PredictWithFeatures failed: %v", err)
		}

		if result == nil {
			t.Fatal("Prediction result should not be nil")
		}

		t.Logf("Feature-based prediction: %f", result.RiskScore)
	})
}

func TestTransformerAttention(t *testing.T) {
	predictor := NewTransformerPredictor()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 15, Y: 15, Event: "move"},
			{Timestamp: 1200, X: 30, Y: 30, Event: "move"},
			{Timestamp: 1300, X: 45, Y: 45, Event: "move"},
			{Timestamp: 1400, X: 60, Y: 60, Event: "end"},
		},
	}

	extractor := NewLSTMFeatureExtractor()
	seq, err := extractor.PrepareSequence(traceData)
	if err != nil {
		t.Fatalf("PrepareSequence failed: %v", err)
	}

	result, err := predictor.Predict(seq)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if result == nil {
		t.Fatal("Prediction result should not be nil")
	}

	if result.RiskScore < 0 || result.RiskScore > 1 {
		t.Errorf("Risk score should be between 0 and 1, got %f", result.RiskScore)
	}

	t.Logf("Attention-based prediction successful, risk: %f", result.RiskScore)
}

func TestTraceServiceNNIntegration(t *testing.T) {
	service := NewTraceService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "move"},
			{Timestamp: 1300, X: 30, Y: 30, Event: "move"},
			{Timestamp: 1400, X: 40, Y: 40, Event: "end"},
		},
		TotalTime: 400,
	}

	t.Run("Extract NN Features", func(t *testing.T) {
		ctx := context.Background()
		features, err := service.ExtractNNFeatures(ctx, traceData)
		if err != nil {
			t.Fatalf("ExtractNNFeatures failed: %v", err)
		}

		if features == nil {
			t.Fatal("Features should not be nil")
		}

		if len(features) == 0 {
			t.Error("Features should not be empty")
		}

		t.Logf("Extracted %d NN features", len(features))
	})

	t.Run("Predict Risk Score", func(t *testing.T) {
		ctx := context.Background()
		score, err := service.PredictRiskScore(ctx, traceData)
		if err != nil {
			t.Fatalf("PredictRiskScore failed: %v", err)
		}

		if score < 0 || score > 1 {
			t.Errorf("Risk score should be between 0 and 1, got %f", score)
		}

		t.Logf("Risk score predicted: %f", score)
	})

	t.Run("Process Trace With NN", func(t *testing.T) {
		ctx := context.Background()
		traceDataJSON, _ := json.Marshal(traceData)

		features, score, nnResult, err := service.ProcessTraceWithNN(ctx, "test-session", traceDataJSON)
		if err != nil {
			t.Fatalf("ProcessTraceWithNN failed: %v", err)
		}

		if features == nil {
			t.Fatal("Features should not be nil")
		}

		if score == nil {
			t.Fatal("Score should not be nil")
		}

		if nnResult == nil {
			t.Fatal("NN result should not be nil")
		}

		t.Logf("ProcessTraceWithNN successful, NN risk: %f", nnResult.RiskScore)
	})

	t.Run("Get Model Info", func(t *testing.T) {
		info := service.GetModelInfo()
		if info == nil {
			t.Fatal("Model info should not be nil")
		}

		if !info["nn_enabled"].(bool) {
			t.Error("NN should be enabled")
		}

		t.Logf("Model info: %+v", info)
	})

	t.Run("Enable/Disable NN", func(t *testing.T) {
		service.EnableNNAnalysis(false)
		if service.enableNN {
			t.Error("NN should be disabled")
		}

		service.EnableNNAnalysis(true)
		if !service.enableNN {
			t.Error("NN should be enabled")
		}
	})
}

func TestTraceServiceNNDisabled(t *testing.T) {
	service := NewTraceService()
	service.EnableNNAnalysis(false)

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 20, Y: 20, Event: "end"},
		},
	}

	ctx := context.Background()
	traceDataJSON, _ := json.Marshal(traceData)

	features, score, nnResult, err := service.ProcessTraceWithNN(ctx, "test-session", traceDataJSON)
	if err != nil {
		t.Fatalf("ProcessTraceWithNN failed: %v", err)
	}

	if features == nil {
		t.Fatal("Features should not be nil even when NN is disabled")
	}

	if score == nil {
		t.Fatal("Score should not be nil even when NN is disabled")
	}

	if nnResult != nil && nnResult.RiskScore != 0 {
		t.Logf("NN result present but NN disabled, this is acceptable")
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("Very Short Trajectory", func(t *testing.T) {
		service := NewTraceService()
		traceData := &model.TraceData{
			Points: []model.TracePoint{
				{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
				{Timestamp: 1100, X: 1, Y: 1, Event: "end"},
			},
		}

		ctx := context.Background()
		score, err := service.PredictRiskScore(ctx, traceData)
		if err != nil {
			t.Logf("Short trajectory error (may be expected): %v", err)
		} else {
			t.Logf("Short trajectory score: %f", score)
		}
	})

	t.Run("Straight Line Trajectory", func(t *testing.T) {
		service := NewTraceService()
		traceData := &model.TraceData{
			Points: []model.TracePoint{
				{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
				{Timestamp: 1100, X: 100, Y: 0, Event: "move"},
				{Timestamp: 1200, X: 200, Y: 0, Event: "move"},
				{Timestamp: 1300, X: 300, Y: 0, Event: "end"},
			},
		}

		ctx := context.Background()
		score, err := service.PredictRiskScore(ctx, traceData)
		if err != nil {
			t.Fatalf("Straight line trajectory failed: %v", err)
		}

		t.Logf("Straight line trajectory score: %f", score)
	})

	t.Run("Erratic Trajectory", func(t *testing.T) {
		service := NewTraceService()
		traceData := &model.TraceData{
			Points: []model.TracePoint{
				{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
				{Timestamp: 1050, X: 50, Y: 10, Event: "move"},
				{Timestamp: 1100, X: 10, Y: 40, Event: "move"},
				{Timestamp: 1150, X: 60, Y: 30, Event: "move"},
				{Timestamp: 1200, X: 20, Y: 70, Event: "move"},
				{Timestamp: 1250, X: 80, Y: 50, Event: "move"},
				{Timestamp: 1300, X: 40, Y: 90, Event: "end"},
			},
		}

		ctx := context.Background()
		score, err := service.PredictRiskScore(ctx, traceData)
		if err != nil {
			t.Fatalf("Erratic trajectory failed: %v", err)
		}

		t.Logf("Erratic trajectory score: %f", score)
	})
}
