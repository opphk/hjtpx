package service

import (
	"context"
	"testing"

	"github.com/hjtpx/hjtpx/internal/model"
)

func TestNNServiceBasic(t *testing.T) {
	service := NewTrajectoryNNService()

	traceData := &model.TraceData{
		Points: []model.TracePoint{
			{Timestamp: 1000, X: 0, Y: 0, Event: "start"},
			{Timestamp: 1100, X: 10, Y: 10, Event: "move"},
			{Timestamp: 1200, X: 25, Y: 25, Event: "move"},
			{Timestamp: 1300, X: 40, Y: 40, Event: "end"},
		},
		TotalTime: 300,
	}

	t.Run("Initialize", func(t *testing.T) {
		ctx := context.Background()
		err := service.Initialize(ctx)
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		if !service.IsLoaded() {
			t.Error("Service should be loaded after Initialize")
		}
	})

	t.Run("Predict Risk", func(t *testing.T) {
		ctx := context.Background()
		result, err := service.PredictRiskFromData(ctx, traceData)
		if err != nil {
			t.Fatalf("PredictRiskFromData failed: %v", err)
		}

		if result == nil {
			t.Fatal("Result should not be nil")
		}

		if result.CombinedScore < 0 || result.CombinedScore > 1 {
			t.Errorf("Combined score should be between 0 and 1, got %f", result.CombinedScore)
		}

		t.Logf("Combined score: %f, Risk level: %s", result.CombinedScore, result.RiskLevel)
	})

	t.Run("Get Config", func(t *testing.T) {
		config := service.GetConfig()
		if config == nil {
			t.Fatal("Config should not be nil")
		}

		if !config.EnableLSTM {
			t.Error("LSTM should be enabled by default")
		}
	})
}
