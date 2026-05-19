package service

import (
	"context"
	"testing"
	"time"
)

func TestFederatedLearningSystem(t *testing.T) {
	system := NewFederatedLearningSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	if !system.initialized {
		t.Error("System should be initialized")
	}
}

func TestRegisterParticipant(t *testing.T) {
	system := NewFederatedLearningSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	participant := &FLParticipant{
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

func TestFederatedTraining(t *testing.T) {
	system := NewFederatedLearningSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	for i := 1; i <= 3; i++ {
		participant := &FLParticipant{
			ID:       "participant_" + string(rune('0'+i)),
			Name:     "Test Participant",
			Platform: "web",
			DataType: "behavior",
		}
		if err := system.RegisterParticipant(ctx, participant); err != nil {
			t.Fatalf("Failed to register participant: %v", err)
		}
	}

	request := &FederatedTrainingRequest{
		TaskType:        "classification",
		Rounds:          10,
		MinParticipants: 2,
		LearningRate:   0.01,
		PrivacyBudget:  1.0,
	}

	result, err := system.StartFederatedTraining(ctx, request)
	if err != nil {
		t.Fatalf("Failed to start training: %v", err)
	}

	if result == nil {
		t.Fatal("Training result should not be nil")
	}

	if result.Result == nil {
		t.Fatal("Round result should not be nil")
	}

	if len(result.Result.ParticipatingNodes) == 0 {
		t.Error("Should have participating nodes")
	}
}

func TestFederatedFeatureExtractor(t *testing.T) {
	extractor := NewFederatedFeatureExtractor()
	ctx := context.Background()

	if err := extractor.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize extractor: %v", err)
	}

	request := &FeatureExtractionRequest{
		ParticipantID: "test_participant",
		DataType:     "behavior",
		Features:     []string{"mouse_velocity", "click_timing"},
		PrivacyLevel: "medium",
	}

	result, err := extractor.ExtractFeatures(ctx, request)
	if err != nil {
		t.Fatalf("Failed to extract features: %v", err)
	}

	if result == nil {
		t.Fatal("Extraction result should not be nil")
	}

	if len(result.ExtractedFeatures) == 0 {
		t.Error("Should have extracted features")
	}

	if result.PrivacyBudgetUsed <= 0 {
		t.Error("Privacy budget should be used")
	}
}

func TestPrivacyProtectionEngine(t *testing.T) {
	engine := NewPrivacyProtectionEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	data := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	epsilon := 1.0
	delta := 1e-5

	noisyData, err := engine.ApplyDifferentialPrivacy(data, epsilon, delta)
	if err != nil {
		t.Fatalf("Failed to apply differential privacy: %v", err)
	}

	if len(noisyData) != len(data) {
		t.Errorf("Expected %d values, got %d", len(data), len(noisyData))
	}
}

func TestSecureAggregation(t *testing.T) {
	engine := NewPrivacyProtectionEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	updates := map[string][]float64{
		"participant_1": {0.1, 0.2, 0.3, 0.4, 0.5},
		"participant_2": {0.2, 0.3, 0.4, 0.5, 0.6},
		"participant_3": {0.3, 0.4, 0.5, 0.6, 0.7},
	}

	result, err := engine.SecureAggregate(updates)
	if err != nil {
		t.Fatalf("Failed to aggregate: %v", err)
	}

	if result == nil {
		t.Fatal("Aggregation result should not be nil")
	}

	if len(result) == 0 {
		t.Error("Result should not be empty")
	}
}

func TestFLCoordinationService(t *testing.T) {
	service := NewFLCoordinationService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	if service.currentRound != 0 {
		t.Errorf("Expected round 0, got %d", service.currentRound)
	}

	if service.minParticipants != 3 {
		t.Errorf("Expected min participants 3, got %d", service.minParticipants)
	}
}

func TestCrossPlatformAnalysis(t *testing.T) {
	system := NewFederatedLearningSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	for i := 1; i <= 3; i++ {
		participant := &FLParticipant{
			ID:           "platform_" + string(rune('0'+i)),
			Name:         "Test Platform",
			Platform:     "platform_" + string(rune('0'+i)),
			DataType:     "behavior",
			TrustScore:   0.8,
			LastSync:     time.Now(),
			Contributions: 10,
		}
		participant.LocalData = &LocalDataset{
			SampleCount: 1000,
			QualityScore: 0.9,
		}
		if err := system.RegisterParticipant(ctx, participant); err != nil {
			t.Fatalf("Failed to register participant: %v", err)
		}
	}

	analysis, err := system.PerformCrossPlatformAnalysis(ctx)
	if err != nil {
		t.Fatalf("Failed to perform analysis: %v", err)
	}

	if analysis == nil {
		t.Fatal("Analysis result should not be nil")
	}

	if len(analysis.platforms) != 3 {
		t.Errorf("Expected 3 platforms, got %d", len(analysis.platforms))
	}
}

func TestGetGlobalModel(t *testing.T) {
	system := NewFederatedLearningSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	model, err := system.GetGlobalModel(ctx)
	if err != nil {
		t.Fatalf("Failed to get global model: %v", err)
	}

	if model == nil {
		t.Fatal("Global model should not be nil")
	}

	if model.ModelID == "" {
		t.Error("Model ID should be set")
	}
}

func TestGetParticipantStats(t *testing.T) {
	system := NewFederatedLearningSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	for i := 1; i <= 2; i++ {
		participant := &FLParticipant{
			ID:           "participant_" + string(rune('0'+i)),
			Name:         "Test Participant",
			Platform:     "web",
			DataType:     "behavior",
			TrustScore:   0.8,
			Contributions: 10,
		}
		if err := system.RegisterParticipant(ctx, participant); err != nil {
			t.Fatalf("Failed to register participant: %v", err)
		}
	}

	stats, err := system.GetParticipantStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	if stats["total_participants"].(int) != 2 {
		t.Errorf("Expected 2 participants, got %d", stats["total_participants"].(int))
	}
}
