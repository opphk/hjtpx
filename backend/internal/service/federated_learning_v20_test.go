package service

import (
	"context"
	"testing"
	"time"
)

func TestFederatedLearningV20(t *testing.T) {
	system := NewFederatedLearningV20()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	if !system.initialized {
		t.Error("System should be initialized")
	}
}

func TestRegisterParticipantV20(t *testing.T) {
	system := NewFederatedLearningV20()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	participant := &FLParticipantV20{
		ID:       "participant_1",
		Name:     "Test Participant",
		Platform: "web",
		DataType: "behavior",
	}

	if err := system.RegisterParticipant(ctx, participant); err != nil {
		t.Fatalf("Failed to register participant: %v", err)
	}

	if len(system.participants) != 1 {
		t.Errorf("Expected 1 participant, got %d", len(system.participants))
	}
}

func TestFederatedRoundV20(t *testing.T) {
	system := NewFederatedLearningV20()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	for i := 1; i <= 3; i++ {
		participant := &FLParticipantV20{
			ID:       "participant_" + string(rune('0'+i)),
			Name:     "Test Participant",
			Platform: "web",
			DataType: "behavior",
		}
		if err := system.RegisterParticipant(ctx, participant); err != nil {
			t.Fatalf("Failed to register participant: %v", err)
		}
	}

	result, err := system.PerformFederatedRound(ctx, "fedavg")
	if err != nil {
		t.Fatalf("Failed to perform federated round: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if !result.Success {
		t.Error("Round should be successful")
	}

	if result.RoundNumber != 1 {
		t.Errorf("Expected round 1, got %d", result.RoundNumber)
	}
}

func TestModelPruningV20(t *testing.T) {
	system := NewFederatedLearningV20()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	system.globalModel.Weights = make([]float64, 256)
	for i := range system.globalModel.Weights {
		system.globalModel.Weights[i] = float64(i % 10)
	}

	pruningRate := 0.5
	if err := system.PruneModel(ctx, pruningRate); err != nil {
		t.Fatalf("Failed to prune model: %v", err)
	}

	if system.globalModel.PruningRate != pruningRate {
		t.Errorf("Expected pruning rate %f, got %f", pruningRate, system.globalModel.PruningRate)
	}
}

func TestModelQuantizationV20(t *testing.T) {
	system := NewFederatedLearningV20()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	system.globalModel.Weights = make([]float64, 256)
	for i := range system.globalModel.Weights {
		system.globalModel.Weights[i] = float64(i % 10)
	}

	bits := 8
	if err := system.QuantizeModel(ctx, bits); err != nil {
		t.Fatalf("Failed to quantize model: %v", err)
	}

	if system.globalModel.QuantizationLevel != bits {
		t.Errorf("Expected quantization level %d, got %d", bits, system.globalModel.QuantizationLevel)
	}
}

func TestEnhancedPrivacyEngine(t *testing.T) {
	engine := NewEnhancedPrivacyEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	data := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	epsilon := 1.0
	delta := 1e-5

	noisyData, err := engine.ApplyDifferentialPrivacy(ctx, data, epsilon, delta)
	if err != nil {
		t.Fatalf("Failed to apply differential privacy: %v", err)
	}

	if len(noisyData) != len(data) {
		t.Errorf("Expected %d values, got %d", len(data), len(noisyData))
	}
}

func TestPrivacyBudget(t *testing.T) {
	engine := NewEnhancedPrivacyEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	total, spent := engine.GetPrivacyBudget()
	if total <= 0 {
		t.Error("Total budget should be positive")
	}

	if spent < 0 {
		t.Error("Spent budget should not be negative")
	}

	t.Logf("Privacy budget: total=%.2f, spent=%.2f", total, spent)
}

func TestFederatedAggregationEngine(t *testing.T) {
	engine := NewFederatedAggregationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	updates := []ModelUpdate{
		{
			ParticipantID: "p1",
			Weights:      []float64{0.1, 0.2, 0.3},
			SampleCount:  100,
		},
		{
			ParticipantID: "p2",
			Weights:      []float64{0.2, 0.3, 0.4},
			SampleCount:  100,
		},
	}

	globalWeights := []float64{0.0, 0.0, 0.0}

	result, err := engine.Aggregate(ctx, updates, globalWeights, "fedavg")
	if err != nil {
		t.Fatalf("Failed to aggregate: %v", err)
	}

	if len(result) != len(globalWeights) {
		t.Errorf("Expected %d weights, got %d", len(globalWeights), len(result))
	}
}

func TestFLMonitoringPanel(t *testing.T) {
	monitor := NewFLMonitoringPanel()
	ctx := context.Background()

	metrics := &RoundMetricsV20{
		RoundNumber:       1,
		Timestamp:        time.Now(),
		ParticipantsCount: 3,
		Accuracy:         0.85,
		Loss:             0.15,
		PrivacySpend:     0.5,
		AvgLatency:       10 * time.Millisecond,
		Converged:        false,
	}

	monitor.RecordRound(ctx, 1, metrics)

	flMetrics := monitor.GetMetrics(ctx)
	if flMetrics.TotalRounds != 1 {
		t.Errorf("Expected 1 round, got %d", flMetrics.TotalRounds)
	}

	rounds := monitor.GetRoundsHistory(ctx, 10)
	if len(rounds) != 1 {
		t.Errorf("Expected 1 round in history, got %d", len(rounds))
	}
}

func TestSecureCommunicationLayer(t *testing.T) {
	layer := NewSecureCommunicationLayer()
	ctx := context.Background()

	channel, err := layer.EstablishChannel(ctx, "node_1")
	if err != nil {
		t.Fatalf("Failed to establish channel: %v", err)
	}

	if channel == nil {
		t.Fatal("Channel should not be nil")
	}

	if !channel.Established {
		t.Error("Channel should be established")
	}

	if !layer.IsAuthenticated("node_1") {
		t.Error("Node should be authenticated")
	}

	if err := layer.RevokeAccess(ctx, "node_1"); err != nil {
		t.Fatalf("Failed to revoke access: %v", err)
	}

	if layer.IsAuthenticated("node_1") {
		t.Error("Node should not be authenticated after revocation")
	}
}
