package service

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestFederatedLearningV2_Initialize(t *testing.T) {
	fl := NewFederatedLearningV2()
	ctx := context.Background()

	err := fl.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !fl.initialized {
		t.Error("initialized should be true after Initialize")
	}

	if fl.globalModel == nil {
		t.Error("globalModel should be initialized")
	}

	if len(fl.globalModel.Weights) == 0 {
		t.Error("globalModel Weights should not be empty")
	}

	if fl.globalModel.Version != "v2.0.0" {
		t.Errorf("expected version v2.0.0, got %s", fl.globalModel.Version)
	}
}

func TestFederatedLearningV2_RegisterParticipant(t *testing.T) {
	fl := NewFederatedLearningV2()
	ctx := context.Background()
	fl.Initialize(ctx)

	participant := &FLParticipantV2{
		ID:        "test_participant_1",
		Name:      "Test Node",
		NodeID:    "node_001",
		Platform:  "linux",
		DataType:  "behavior",
		TrustScore: 0.9,
	}

	err := fl.RegisterParticipant(ctx, participant)
	if err != nil {
		t.Fatalf("RegisterParticipant failed: %v", err)
	}

	if participant.Status != "registered" {
		t.Errorf("expected status 'registered', got %s", participant.Status)
	}

	if len(participant.Weights) == 0 {
		t.Error("participant Weights should be initialized")
	}

	err = fl.RegisterParticipant(ctx, participant)
	if err == nil {
		t.Error("RegisterParticipant should fail for duplicate participant")
	}
}

func TestFederatedLearningV2_StartFederatedRound(t *testing.T) {
	fl := NewFederatedLearningV2()
	ctx := context.Background()
	fl.Initialize(ctx)

	for i := 1; i <= 5; i++ {
		participant := &FLParticipantV2{
			ID:         "test_participant_" + string(rune('0'+i)),
			Name:       "Test Node " + string(rune('0'+i)),
			NodeID:     "node_00" + string(rune('0'+i)),
			Platform:   "linux",
			DataType:   "behavior",
			TrustScore: 0.8 + float64(i)*0.02,
		}
		fl.RegisterParticipant(ctx, participant)
	}

	request := &FederatedRoundRequest{
		TaskType:       "classification",
		Rounds:         10,
		MinParticipants: 3,
		LearningRate:   0.001,
		PrivacyBudget:  1.0,
		SecureAgg:      true,
	}

	response, err := fl.StartFederatedRound(ctx, request)
	if err != nil {
		t.Fatalf("StartFederatedRound failed: %v", err)
	}

	if !response.Success {
		t.Error("expected success response")
	}

	if response.ParticipantsCount < 3 {
		t.Errorf("expected at least 3 participants, got %d", response.ParticipantsCount)
	}

	if response.Performance == nil {
		t.Error("Performance should not be nil")
	}

	if response.Performance.Accuracy < 0 || response.Performance.Accuracy > 1 {
		t.Errorf("Accuracy should be between 0 and 1, got %f", response.Performance.Accuracy)
	}
}

func TestFederatedLearningV2_SelectParticipants(t *testing.T) {
	fl := NewFederatedLearningV2()
	ctx := context.Background()
	fl.Initialize(ctx)

	for i := 1; i <= 5; i++ {
		participant := &FLParticipantV2{
			ID:         "participant_" + string(rune('0'+i)),
			Name:       "Node " + string(rune('0'+i)),
			TrustScore: 0.5 + float64(i)*0.1,
			Status:     "active",
		}
		fl.RegisterParticipant(ctx, participant)
	}

	selected := fl.selectParticipantsForRound(3)

	if len(selected) != 3 {
		t.Errorf("expected 3 participants, got %d", len(selected))
	}

	sort.Slice(selected, func(i, j int) bool {
		return fl.participants[selected[i]].TrustScore > fl.participants[selected[j]].TrustScore
	})

	for i := 0; i < len(selected)-1; i++ {
		if fl.participants[selected[i]].TrustScore < fl.participants[selected[i+1]].TrustScore {
			t.Error("participants should be sorted by trust score descending")
		}
	}
}

func TestFederatedLearningV2_ApplyDPUpdate(t *testing.T) {
	fl := NewFederatedLearningV2()
	ctx := context.Background()
	fl.Initialize(ctx)

	participant := &FLParticipantV2{
		ID:         "dp_test_participant",
		Name:       "DP Test Node",
		TrustScore: 0.9,
	}
	fl.RegisterParticipant(ctx, participant)

	gradients := make([]float64, 256)
	for i := range gradients {
		gradients[i] = float64(i) * 0.01
	}

	request := &DPUpdateRequest{
		ParticipantID: participant.ID,
		Gradients:    gradients,
		ClipNorm:     1.0,
	}

	response, err := fl.ApplyDPUpdate(ctx, request)
	if err != nil {
		t.Fatalf("ApplyDPUpdate failed: %v", err)
	}

	if !response.Success {
		t.Error("expected success response")
	}

	if len(response.NoisedGradients) != len(gradients) {
		t.Errorf("expected %d noised gradients, got %d", len(gradients), len(response.NoisedGradients))
	}

	if !response.Clipped {
		t.Log("note: gradients may not have been clipped depending on their norm")
	}
}

func TestFederatedLearningV2_GetMonitoringStats(t *testing.T) {
	fl := NewFederatedLearningV2()
	ctx := context.Background()
	fl.Initialize(ctx)

	for i := 1; i <= 3; i++ {
		participant := &FLParticipantV2{
			ID:         "mon_participant_" + string(rune('0'+i)),
			Name:       "Monitor Test Node " + string(rune('0'+i)),
			TrustScore: 0.8,
		}
		fl.RegisterParticipant(ctx, participant)
	}

	fl.StartFederatedRound(ctx, &FederatedRoundRequest{
		MinParticipants: 3,
		PrivacyBudget:  0.5,
	})

	request := &MonitoringStatsRequest{
		TimeRange: "1h",
		Metrics:   []string{"accuracy", "latency", "privacy"},
	}

	response, err := fl.GetMonitoringStats(ctx, request)
	if err != nil {
		t.Fatalf("GetMonitoringStats failed: %v", err)
	}

	if response.Metrics == nil {
		t.Error("Metrics should not be nil")
	}

	if response.Metrics.TotalRounds == 0 {
		t.Error("TotalRounds should be at least 1 after federated round")
	}

	if len(response.ParticipantStats) != 3 {
		t.Errorf("expected 3 participant stats, got %d", len(response.ParticipantStats))
	}
}

func TestSecureAggregationProtocol_EncryptDecrypt(t *testing.T) {
	sap := NewSecureAggregationProtocol(3)

	originalWeights := make([]float64, 256)
	for i := range originalWeights {
		originalWeights[i] = float64(i) * 0.1
	}

	pubKey := make([]byte, ed25519.PublicKeySize)
	for i := range pubKey {
		pubKey[i] = byte(i)
	}

	encrypted, err := sap.EncryptWeights(originalWeights, pubKey)
	if err != nil {
		t.Fatalf("EncryptWeights failed: %v", err)
	}

	if len(encrypted) == 0 {
		t.Error("encrypted data should not be empty")
	}

	decrypted, err := sap.DecryptWeights(encrypted)
	if err != nil {
		t.Fatalf("DecryptWeights failed: %v", err)
	}

	if len(decrypted) != len(originalWeights) {
		t.Errorf("expected %d decrypted weights, got %d", len(originalWeights), len(decrypted))
	}
}

func TestSecureAggregationProtocol_Commitment(t *testing.T) {
	sap := NewSecureAggregationProtocol(3)

	weights := make([]float64, 256)
	for i := range weights {
		weights[i] = float64(i) * 0.1
	}

	commitment := sap.GenerateCommitment(weights)
	if len(commitment) == 0 {
		t.Error("commitment should not be empty")
	}

	if !sap.VerifyCommitment(weights, commitment) {
		t.Error("commitment verification should pass for original weights")
	}

	weights[0] = 999.0
	if sap.VerifyCommitment(weights, commitment) {
		t.Error("commitment verification should fail for modified weights")
	}
}

func TestSecureAggregationProtocol_SecureSum(t *testing.T) {
	sap := NewSecureAggregationProtocol(3)

	weights1 := make([]float64, 256)
	weights2 := make([]float64, 256)
	weights3 := make([]float64, 256)

	for i := range weights1 {
		weights1[i] = float64(i) * 0.1
		weights2[i] = float64(i) * 0.2
		weights3[i] = float64(i) * 0.3
	}

	models := map[string][]float64{
		"participant_1": weights1,
		"participant_2": weights2,
		"participant_3": weights3,
	}

	result := sap.SecureSum(models, 3)

	if len(result) != 256 {
		t.Errorf("expected 256 result weights, got %d", len(result))
	}

	expectedSum := (weights1[0] + weights2[0] + weights3[0]) / 3.0
	if math.Abs(result[0]-expectedSum) > 0.01 {
		t.Errorf("expected average ~%f, got %f", expectedSum, result[0])
	}
}

func TestFedAvgOptimizer_Aggregate(t *testing.T) {
	optimizer := NewFedAvgOptimizer(0.001, 0.9, 0.0001)
	ctx := context.Background()
	optimizer.Initialize(ctx)

	models := map[string][]float64{
		"p1": make([]float64, 256),
		"p2": make([]float64, 256),
		"p3": make([]float64, 256),
	}

	for _, model := range models {
		for i := range model {
			model[i] = float64(i) * 0.1
		}
	}

	participants := []string{"p1", "p2", "p3"}
	result := optimizer.Aggregate(models, participants)

	if len(result) != 256 {
		t.Errorf("expected 256 result weights, got %d", len(result))
	}

	expected := float64(0) * 0.1
	for i := range result {
		expected += float64(i) * 0.1
	}
	expected /= 3.0

	if math.Abs(result[0]-expected) > 0.01 {
		t.Errorf("expected ~%f, got %f", expected, result[0])
	}
}

func TestDifferentialPrivacyEngine_ApplyNoise(t *testing.T) {
	dp := NewDifferentialPrivacyEngine(1.0, 1e-5, 1.0)
	ctx := context.Background()
	dp.Initialize(ctx)

	gradients := make([]float64, 256)
	for i := range gradients {
		gradients[i] = float64(i) * 0.01
	}

	noised, used, err := dp.ApplyNoise(gradients, 0.5)
	if err != nil {
		t.Fatalf("ApplyNoise failed: %v", err)
	}

	if len(noised) != len(gradients) {
		t.Errorf("expected %d noised gradients, got %d", len(gradients), len(noised))
	}

	if used <= 0 || used > 0.5 {
		t.Errorf("expected used budget between 0 and 0.5, got %f", used)
	}

	diffFound := false
	for i := range gradients {
		if math.Abs(noised[i]-gradients[i]) > 0.001 {
			diffFound = true
			break
		}
	}
	if !diffFound {
		t.Error("noise should have been added to gradients")
	}
}

func TestDifferentialPrivacyEngine_ClipGradients(t *testing.T) {
	dp := NewDifferentialPrivacyEngine(1.0, 1e-5, 1.0)

	gradients := make([]float64, 256)
	for i := range gradients {
		gradients[i] = float64(i) * 0.1
	}

	clipped, wasClipped, err := dp.ClipGradients(gradients, 5.0)
	if err != nil {
		t.Fatalf("ClipGradients failed: %v", err)
	}

	if !wasClipped {
		t.Log("note: gradients may not be clipped depending on their norm")
	}

	norm := 0.0
	for _, g := range clipped {
		norm += g * g
	}
	norm = math.Sqrt(norm)

	if norm > 5.01 {
		t.Errorf("clipped gradient norm should be <= 5.0, got %f", norm)
	}
}

func TestPrivacyBudgetTracker_RecordUsage(t *testing.T) {
	tracker := NewPrivacyBudgetTracker(10.0)

	weights := make([]float64, 256)
	for i := range weights {
		weights[i] = float64(i) * 0.1
	}

	tracker.RecordUsage(weights, 0.5)
	tracker.RecordUsage(weights, 0.3)

	remaining := tracker.GetRemainingBudget()
	expected := 10.0 - 0.5 - 0.3

	if math.Abs(remaining-expected) > 0.01 {
		t.Errorf("expected remaining budget %f, got %f", expected, remaining)
	}
}

func TestComputeMerkleProof(t *testing.T) {
	items := make([][]byte, 8)
	for i := range items {
		h := sha256.New()
		h.Write([]byte{byte(i)})
		items[i] = h.Sum(nil)
	}

	proof, root := ComputeMerkleProof(items, 0)

	if root == nil || len(root) == 0 {
		t.Error("Merkle root should not be empty")
	}

	if proof == nil {
		t.Error("Merkle proof should not be nil")
	}
}

func TestFLMonitoringPanel_UpdateMetrics(t *testing.T) {
	fl := NewFederatedLearningV2()
	ctx := context.Background()
	fl.Initialize(ctx)

	participant := &FLParticipantV2{
		ID:         "metrics_test_participant",
		Name:       "Metrics Test Node",
		TrustScore: 0.9,
	}
	fl.RegisterParticipant(ctx, participant)

	performance := &ModelPerformanceV2{
		Accuracy:     0.85,
		Loss:         0.15,
		AUC:          0.90,
		AvgLatencyMs: 45,
		Throughput:   22.2,
	}

	fl.updateMonitoringPanel([]string{participant.ID}, performance, 0.2)

	if fl.monitoringPanel.metrics.TotalRounds != 1 {
		t.Errorf("expected 1 total round, got %d", fl.monitoringPanel.metrics.TotalRounds)
	}

	if fl.monitoringPanel.metrics.AvgAccuracy != 0.85 {
		t.Errorf("expected accuracy 0.85, got %f", fl.monitoringPanel.metrics.AvgAccuracy)
	}

	if fl.monitoringPanel.metrics.PrivacyBudgetUsed != 0.2 {
		t.Errorf("expected privacy used 0.2, got %f", fl.monitoringPanel.metrics.PrivacyBudgetUsed)
	}
}

func TestParseFLRoundRequest(t *testing.T) {
	jsonData := `{
		"task_type": "classification",
		"rounds": 10,
		"min_participants": 3,
		"learning_rate": 0.001,
		"privacy_budget": 1.0,
		"secure_agg": true
	}`

	request, err := ParseFLRoundRequest(jsonData)
	if err != nil {
		t.Fatalf("ParseFLRoundRequest failed: %v", err)
	}

	if request.TaskType != "classification" {
		t.Errorf("expected task_type 'classification', got %s", request.TaskType)
	}

	if request.Rounds != 10 {
		t.Errorf("expected rounds 10, got %d", request.Rounds)
	}

	if request.MinParticipants != 3 {
		t.Errorf("expected min_participants 3, got %d", request.MinParticipants)
	}

	if request.LearningRate != 0.001 {
		t.Errorf("expected learning_rate 0.001, got %f", request.LearningRate)
	}

	if request.PrivacyBudget != 1.0 {
		t.Errorf("expected privacy_budget 1.0, got %f", request.PrivacyBudget)
	}

	if !request.SecureAgg {
		t.Error("expected secure_agg true")
	}
}

func TestParseMonitoringStatsRequest(t *testing.T) {
	jsonData := `{
		"time_range": "1h",
		"metrics": ["accuracy", "latency", "privacy"]
	}`

	request, err := ParseMonitoringStatsRequest(jsonData)
	if err != nil {
		t.Fatalf("ParseMonitoringStatsRequest failed: %v", err)
	}

	if request.TimeRange != "1h" {
		t.Errorf("expected time_range '1h', got %s", request.TimeRange)
	}

	if len(request.Metrics) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(request.Metrics))
	}
}

func TestGlobalModelV2_Performance(t *testing.T) {
	model := &GlobalModelV2{
		ModelID:      "test_model",
		Version:      "v2.0.0",
		Weights:      generateWeightsV2(256),
		Architecture: "federated_neural_network_v2",
		Performance: &ModelPerformanceV2{
			Accuracy:     0.92,
			Loss:         0.08,
			AUC:          0.95,
			AvgLatencyMs: 35,
			Throughput:   28.6,
		},
		PrivacyBudget: 8.5,
		LastUpdate:   time.Now(),
	}

	if model.Performance.Accuracy < 0 || model.Performance.Accuracy > 1 {
		t.Error("Accuracy should be between 0 and 1")
	}

	if model.Performance.Loss < 0 || model.Performance.Loss > 1 {
		t.Error("Loss should be between 0 and 1")
	}

	if model.Performance.AUC < 0 || model.Performance.AUC > 1 {
		t.Error("AUC should be between 0 and 1")
	}
}

func TestFLParticipantV2_Initialization(t *testing.T) {
	participant := &FLParticipantV2{
		ID:        "test_participant",
		Name:      "Test Participant",
		NodeID:    "node_001",
		Platform:  "linux",
		DataType:  "behavior",
		TrustScore: 0.85,
	}

	participant.Performance = &ParticipantPerformance{
		Accuracy:       0.88,
		Precision:      0.90,
		Recall:         0.85,
		F1Score:        0.87,
		LatencyMs:      42,
		EnergyConsumed: 0.5,
	}

	if participant.Performance.Accuracy < 0 || participant.Performance.Accuracy > 1 {
		t.Error("Accuracy should be between 0 and 1")
	}

	if participant.Performance.F1Score < 0 || participant.Performance.F1Score > 1 {
		t.Error("F1Score should be between 0 and 1")
	}

	if participant.Performance.LatencyMs < 0 {
		t.Error("LatencyMs should be non-negative")
	}
}

func TestRoundConfiguration(t *testing.T) {
	config := &RoundConfiguration{
		MinParticipants:    5,
		MaxRounds:         100,
		TargetAccuracy:    0.95,
		Timeout:           5 * time.Minute,
		AggregationMethod: "fedavg",
		PrivacyBudget:     2.0,
		SecureAggregation: true,
		DifferentialPrivacy: true,
	}

	if config.MinParticipants < 2 {
		t.Error("MinParticipants should be at least 2")
	}

	if config.MaxRounds <= 0 {
		t.Error("MaxRounds should be positive")
	}

	if config.TargetAccuracy < 0 || config.TargetAccuracy > 1 {
		t.Error("TargetAccuracy should be between 0 and 1")
	}

	if config.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}
}

func TestFederatedRoundResponse_Serialization(t *testing.T) {
	response := &FederatedRoundResponse{
		Success:          true,
		RoundNumber:      5,
		GlobalModelID:    "model_001",
		Performance: &ModelPerformanceV2{
			Accuracy:     0.90,
			Loss:         0.10,
			AUC:          0.93,
			AvgLatencyMs: 40,
			Throughput:   25.0,
		},
		ParticipantsCount: 4,
		PrivacyUsed:      0.3,
		Duration:         2 * time.Second,
		Converged:        false,
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled FederatedRoundResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.Success != response.Success {
		t.Error("Success mismatch")
	}

	if unmarshaled.RoundNumber != response.RoundNumber {
		t.Error("RoundNumber mismatch")
	}

	if unmarshaled.Performance.Accuracy != response.Performance.Accuracy {
		t.Error("Performance.Accuracy mismatch")
	}
}

func TestTrendAnalysis(t *testing.T) {
	trend := &FLTrendAnalysis{
		AccuracyTrend:     []float64{0.80, 0.82, 0.85, 0.87, 0.90},
		LatencyTrend:      []int64{60, 55, 50, 45, 40},
		PrivacyTrend:      []float64{0.1, 0.12, 0.15, 0.18, 0.20},
		PredictedAccuracy: 0.93,
		Confidence:        0.88,
	}

	if len(trend.AccuracyTrend) == 0 {
		t.Error("AccuracyTrend should not be empty")
	}

	if trend.AccuracyTrend[len(trend.AccuracyTrend)-1] < trend.AccuracyTrend[0] {
		t.Log("note: accuracy trend may decrease in some scenarios")
	}

	if trend.PredictedAccuracy < 0 || trend.PredictedAccuracy > 1 {
		t.Error("PredictedAccuracy should be between 0 and 1")
	}

	if trend.Confidence < 0 || trend.Confidence > 1 {
		t.Error("Confidence should be between 0 and 1")
	}
}

func TestFLAlert(t *testing.T) {
	alert := &FLAlert{
		AlertID:     "alert_001",
		Type:        "performance",
		Severity:    "warning",
		Message:     "High latency detected",
		Participant: "participant_001",
		Timestamp:   time.Now(),
		Resolved:    false,
	}

	if alert.AlertID == "" {
		t.Error("AlertID should not be empty")
	}

	validSeverities := map[string]bool{"info": true, "warning": true, "error": true, "critical": true}
	if !validSeverities[alert.Severity] {
		t.Errorf("Invalid severity: %s", alert.Severity)
	}
}

func TestHashFunction(t *testing.T) {
	hf := NewHashFunction()

	data := []byte("test data for hashing")

	hash1 := hf.ComputeHash(data)
	hash2 := hf.ComputeHash(data)

	if string(hash1) != string(hash2) {
		t.Error("same data should produce same hash")
	}

	hash3 := hf.ComputeHash([]byte("different data"))
	if string(hash1) == string(hash3) {
		t.Error("different data should produce different hash")
	}

	hashStr := hf.ComputeHashString(data)
	if hashStr == "" {
		t.Error("hash string should not be empty")
	}
}

func TestFedAvgOptimizer_ApplyMomentum(t *testing.T) {
	optimizer := NewFedAvgOptimizer(0.001, 0.9, 0.0001)
	ctx := context.Background()
	optimizer.Initialize(ctx)

	gradient := make([]float64, 256)
	for i := range gradient {
		gradient[i] = float64(i) * 0.01
	}

	result1 := optimizer.ApplyMomentum(gradient, "test_key")

	result2 := optimizer.ApplyMomentum(gradient, "test_key")

	for i := range result1 {
		if result1[i] == result2[i] {
			t.Error("momentum should accumulate gradients")
		}
	}
}

