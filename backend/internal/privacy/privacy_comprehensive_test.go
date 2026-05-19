package privacy

import (
	"testing"
	"time"
)

func TestFederatedLearning_Registration(t *testing.T) {
	config := FederatedLearningConfig{
		NumClients:        5,
		NumRounds:         10,
		LocalEpochs:       3,
		BatchSize:         32,
		LearningRate:      0.01,
		ModelType:         "linear",
		AggregationMethod: "fedavg",
		UseDP:            false,
	}

	fl := NewFederatedLearning(config)

	client := fl.RegisterClient("client1", 100)
	if client == nil {
		t.Error("Client registration should return a client")
	}

	if fl.GetNumClients() != 1 {
		t.Errorf("Expected 1 client, got %d", fl.GetNumClients())
	}
}

func TestFederatedLearning_Unregistration(t *testing.T) {
	config := FederatedLearningConfig{
		NumClients:        5,
		NumRounds:         10,
		LocalEpochs:       3,
		AggregationMethod: "fedavg",
	}

	fl := NewFederatedLearning(config)

	fl.RegisterClient("client1", 100)
	fl.RegisterClient("client2", 200)

	if fl.GetNumClients() != 2 {
		t.Error("Should have 2 clients registered")
	}

	success := fl.UnregisterClient("client1")
	if !success {
		t.Error("Unregistration should succeed")
	}

	if fl.GetNumClients() != 1 {
		t.Errorf("Expected 1 client after unregistration, got %d", fl.GetNumClients())
	}
}

func TestFederatedClient_Training(t *testing.T) {
	config := FederatedClientConfig{
		ClientID:     "test-client",
		DataSize:     50,
		LocalEpochs:  2,
		BatchSize:    10,
		LearningRate: 0.01,
		UseDP:        false,
		ModelType:    "linear",
	}

	client := NewFederatedClient(config)

	model := &ModelParameters{
		Weights: map[string][]float64{
			"layer1": {0.1, 0.2, 0.3, 0.4, 0.5},
		},
		Biases: map[string][]float64{
			"layer1": {0.0},
		},
		Round: 0,
	}

	update := client.Train(model, 0)
	if update == nil {
		t.Error("Training should return an update")
	}

	if update.ClientID != "test-client" {
		t.Errorf("Expected client ID 'test-client', got '%s'", update.ClientID)
	}

	if update.DataSize != 50 {
		t.Errorf("Expected data size 50, got %d", update.DataSize)
	}
}

func TestFederatedClient_DataManagement(t *testing.T) {
	config := FederatedClientConfig{
		ClientID:     "test-client",
		DataSize:     10,
		LocalEpochs:  1,
		BatchSize:    5,
		LearningRate: 0.01,
		UseDP:        false,
	}

	client := NewFederatedClient(config)

	initialSize := client.GetLocalDataSize()

	newData := []DataPoint{
		{Features: []float64{1.0, 2.0}, Label: 3.0},
		{Features: []float64{4.0, 5.0}, Label: 9.0},
	}

	client.AddLocalData(newData)

	newSize := client.GetLocalDataSize()
	if newSize <= initialSize {
		t.Error("Data should have been added")
	}
}

func TestFederatedServer_Aggregation(t *testing.T) {
	config := FederatedServerConfig{
		AggregationMethod: "fedavg",
		UseDP:            false,
	}

	server := NewFederatedServer(config)

	updates := []*ClientUpdate{
		{
			ClientID: "client1",
			Weights: map[string][]float64{
				"layer1": {1.0, 2.0, 3.0},
			},
			Biases: map[string][]float64{
				"layer1": {0.1},
			},
			DataSize: 100,
			Round:    1,
		},
		{
			ClientID: "client2",
			Weights: map[string][]float64{
				"layer1": {4.0, 5.0, 6.0},
			},
			Biases: map[string][]float64{
				"layer1": {0.2},
			},
			DataSize: 100,
			Round:    1,
		},
	}

	server.AggregateUpdates(updates)

	model := server.GetGlobalModel()
	if model == nil {
		t.Error("Should have a global model after aggregation")
	}

	expectedWeights := []float64{2.5, 3.5, 4.5}
	actualWeights := model.Weights["layer1"]

	for i := range expectedWeights {
		if actualWeights[i] != expectedWeights[i] {
			t.Errorf("Expected weight %f at index %d, got %f",
				expectedWeights[i], i, actualWeights[i])
		}
	}
}

func TestFederatedServer_DPAggregation(t *testing.T) {
	config := FederatedServerConfig{
		AggregationMethod: "fedavg",
		UseDP:            true,
		DPConfig: DPConfig{
			Epsilon:     1.0,
			Delta:       1e-5,
			ClipNorm:    1.0,
			MaxGradNorm: 1.0,
		},
	}

	server := NewFederatedServer(config)

	updates := []*ClientUpdate{
		{
			ClientID: "client1",
			Weights: map[string][]float64{
				"layer1": {1.0, 2.0, 3.0},
			},
			Biases: map[string][]float64{
				"layer1": {0.1},
			},
			DataSize: 100,
			Round:    1,
		},
	}

	server.AggregateUpdates(updates)

	remaining := server.GetPrivacyBudgetRemaining()
	if remaining >= 1.0 {
		t.Error("Privacy budget should have been spent")
	}
}

func TestFedAvgStrategy(t *testing.T) {
	strategy := &FedAvgStrategy{}

	updates := []*ClientUpdate{
		{
			ClientID: "client1",
			Weights: map[string][]float64{
				"layer1": {2.0, 4.0},
			},
			Biases: map[string][]float64{
				"layer1": {1.0},
			},
			DataSize: 2,
			Round:    1,
		},
		{
			ClientID: "client2",
			Weights: map[string][]float64{
				"layer1": {4.0, 6.0},
			},
			Biases: map[string][]float64{
				"layer1": {2.0},
			},
			DataSize: 2,
			Round:    1,
		},
	}

	weights := map[string]float64{
		"client1": 2,
		"client2": 2,
	}

	result, err := strategy.Aggregate(updates, weights)
	if err != nil {
		t.Errorf("Aggregation failed: %v", err)
	}

	expectedWeight := 3.0
	if result.Weights["layer1"][0] != expectedWeight {
		t.Errorf("Expected first weight 3.0, got %f", result.Weights["layer1"][0])
	}
}

func TestPrivacyImpactAssessment(t *testing.T) {
	pia := NewPrivacyImpactAssessment("Test Project", "Test Description")

	if pia.ProjectName != "Test Project" {
		t.Errorf("Expected project name 'Test Project', got '%s'", pia.ProjectName)
	}

	if pia.Status != StatusDraft {
		t.Error("New PIA should be in draft status")
	}

	dataType := DataType{
		Name:        "Email",
		Category:    ContactInfo,
		Sensitivity: HighSensitivity,
		IsPersonal:  true,
	}

	pia.AddDataType(dataType)

	if len(pia.DataTypes) != 1 {
		t.Error("Should have 1 data type")
	}
}

func TestPIAFinding(t *testing.T) {
	pia := NewPrivacyImpactAssessment("Test Project", "Test Description")

	finding := PIAFinding{
		ID:          "FIND-001",
		Category:    CategoryDataCollection,
		Description: "Test finding",
		Severity:    SeverityMajor,
		Likelihood:  LikelihoodLikely,
	}

	pia.AddFinding(finding)

	if len(pia.Findings) != 1 {
		t.Error("Should have 1 finding")
	}

	score := pia.GetRiskScore()
	if score <= 0 {
		t.Error("Risk score should be positive for a finding")
	}
}

func TestPIARecommendations(t *testing.T) {
	pia := NewPrivacyImpactAssessment("Test Project", "Test Description")

	recommendation := Recommendation{
		ID:          "REC-001",
		Description: "Test recommendation",
		Priority:    PriorityHigh,
		Status:      RecStatusPending,
	}

	pia.AddRecommendation(recommendation)

	pending := pia.GetPendingRecommendations()
	if len(pending) != 1 {
		t.Error("Should have 1 pending recommendation")
	}
}

func TestPIAApproval(t *testing.T) {
	pia := NewPrivacyImpactAssessment("Test Project", "Test Description")

	pia.Approve("Approver Name")

	if pia.Status != StatusApproved {
		t.Error("PIA should be approved")
	}

	if pia.Approver != "Approver Name" {
		t.Error("Approver name should be set")
	}

	if pia.ExpiryDate.IsZero() {
		t.Error("Expiry date should be set")
	}
}

func TestRiskCalculator(t *testing.T) {
	config := RiskConfig{
		BaseScore:        50.0,
		ImpactWeight:     0.5,
		LikelihoodWeight: 0.3,
	}

	calculator := NewRiskCalculator(config)

	factor := RiskFactor{
		Name:          "Data Sensitivity",
		Category:      PrivacyRisk,
		Weight:        0.8,
		Value:         0.7,
		ContributesTo: []RiskCategory{PrivacyRisk},
	}

	calculator.AddFactor(factor)

	score := calculator.CalculateRisk()

	if score.OverallScore <= 0 {
		t.Error("Overall score should be positive")
	}
}

func TestRiskCalculator_TopRisks(t *testing.T) {
	config := RiskConfig{
		BaseScore: 50.0,
	}

	calculator := NewRiskCalculator(config)

	factors := []RiskFactor{
		{Name: "Factor1", Category: PrivacyRisk, Weight: 0.5, Value: 0.5, ContributesTo: []RiskCategory{PrivacyRisk}},
		{Name: "Factor2", Category: PrivacyRisk, Weight: 0.9, Value: 0.9, ContributesTo: []RiskCategory{PrivacyRisk}},
		{Name: "Factor3", Category: PrivacyRisk, Weight: 0.3, Value: 0.3, ContributesTo: []RiskCategory{PrivacyRisk}},
	}

	for _, f := range factors {
		calculator.AddFactor(f)
	}

	topRisks := calculator.GetTopRisks(2)
	if len(topRisks) != 2 {
		t.Errorf("Expected 2 top risks, got %d", len(topRisks))
	}
}

func TestRiskCalculator_MonteCarlo(t *testing.T) {
	config := RiskConfig{
		BaseScore: 50.0,
	}

	calculator := NewRiskCalculator(config)

	factor := RiskFactor{
		Name:          "Test Factor",
		Category:      PrivacyRisk,
		Weight:        0.7,
		Value:         0.6,
		ContributesTo: []RiskCategory{PrivacyRisk},
	}

	calculator.AddFactor(factor)

	result := calculator.MonteCarloSimulation(100)

	if result.Mean <= 0 {
		t.Error("Mean should be positive")
	}

	if result.Percentile95 <= result.Percentile5 {
		t.Error("95th percentile should be greater than 5th percentile")
	}
}

func TestMitigationPlanner(t *testing.T) {
	riskConfig := RiskConfig{
		BaseScore: 50.0,
	}
	riskTracker := NewRiskCalculator(riskConfig)

	planner := NewMitigationPlanner(riskTracker)

	mitigation := planner.CreateMitigation(
		"Test Mitigation",
		"Test Description",
		"risk-001",
		0.8,
		5000.0,
		30*24*time.Hour,
	)

	if mitigation == nil {
		t.Error("Should create a mitigation")
	}

	if mitigation.Effectiveness != 0.8 {
		t.Errorf("Expected effectiveness 0.8, got %f", mitigation.Effectiveness)
	}
}

func TestMitigationPlanner_Milestones(t *testing.T) {
	riskTracker := NewRiskCalculator(RiskConfig{BaseScore: 50.0})
	planner := NewMitigationPlanner(riskTracker)

	mitigation := planner.CreateMitigation("Test", "Description", "risk-001", 0.8, 5000, 24*time.Hour)

	milestone := planner.AddMilestone(
		mitigation.ID,
		"Phase 1",
		"Complete first phase",
		time.Now().Add(7*24*time.Hour),
	)

	if milestone == nil {
		t.Error("Should create a milestone")
	}
}

func TestMitigationPlanner_StartMitigation(t *testing.T) {
	riskTracker := NewRiskCalculator(RiskConfig{BaseScore: 50.0})
	planner := NewMitigationPlanner(riskTracker)

	mitigation := planner.CreateMitigation("Test", "Description", "risk-001", 0.8, 5000, 24*time.Hour)

	err := planner.StartMitigation(mitigation.ID, "John Doe")
	if err != nil {
		t.Errorf("StartMitigation failed: %v", err)
	}

	m := planner.GetMitigation(mitigation.ID)
	if m.Status != MitigationStatusInProgress {
		t.Error("Mitigation should be in progress")
	}
}

func TestMitigationPlanner_Complete(t *testing.T) {
	riskTracker := NewRiskCalculator(RiskConfig{BaseScore: 50.0})
	planner := NewMitigationPlanner(riskTracker)

	mitigation := planner.CreateMitigation("Test", "Description", "risk-001", 0.8, 5000, 24*time.Hour)
	planner.StartMitigation(mitigation.ID, "John Doe")

	err := planner.CompleteMitigation(mitigation.ID)
	if err != nil {
		t.Errorf("CompleteMitigation failed: %v", err)
	}

	m := planner.GetMitigation(mitigation.ID)
	if m.Status != MitigationStatusCompleted {
		t.Error("Mitigation should be completed")
	}
}

func TestMitigationPlanner_Progress(t *testing.T) {
	riskTracker := NewRiskCalculator(RiskConfig{BaseScore: 50.0})
	planner := NewMitigationPlanner(riskTracker)

	planner.CreateMitigation("Mit1", "Desc1", "risk-001", 0.8, 1000, 24*time.Hour)
	planner.CreateMitigation("Mit2", "Desc2", "risk-002", 0.7, 2000, 24*time.Hour)

	progress := planner.GetProgress()

	if progress.Total != 2 {
		t.Errorf("Expected 2 total mitigations, got %d", progress.Total)
	}
}

func TestPIAReporter(t *testing.T) {
	reporter := NewPIAReporter()

	pia1 := NewPrivacyImpactAssessment("Project1", "Description1")
	pia2 := NewPrivacyImpactAssessment("Project2", "Description2")

	reporter.AddAssessment(pia1)
	reporter.AddAssessment(pia2)

	assessments := reporter.GetAllAssessments()
	if len(assessments) != 2 {
		t.Errorf("Expected 2 assessments, got %d", len(assessments))
	}

	summary := reporter.GenerateSummary()
	if summary.TotalAssessments != 2 {
		t.Errorf("Expected 2 total assessments in summary, got %d", summary.TotalAssessments)
	}
}

func TestHomomorphicEncryption(t *testing.T) {
	he, err := NewHomomorphicEncryption(PaillierScheme, 1024)
	if err != nil {
		t.Fatalf("Failed to create HE: %v", err)
	}

	m1 := int64(100)
	m2 := int64(200)

	ct1, err := he.Encrypt([]byte(string(rune(m1))))
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	ct2, err := he.Encrypt([]byte(string(rune(m2))))
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	ctSum := he.HomomorphicAdd(ct1, ct2)

	result, err := he.Decrypt(ctSum)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	t.Logf("Homomorphic addition result: %s", string(result))
}

func TestPartialHomomorphicEncryption(t *testing.T) {
	phe, err := NewPartialHE(1024)
	if err != nil {
		t.Fatalf("Failed to create Partial HE: %v", err)
	}

	m1 := int64(100)
	m2 := int64(200)

	ct1, err := phe.Encrypt(m1)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	ct2, err := phe.Encrypt(m2)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	ctSum := phe.Add(ct1, ct2)

	result, err := phe.Decrypt(ctSum)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	expected := m1 + m2
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestPartialHE_ScalarMultiplication(t *testing.T) {
	phe, err := NewPartialHE(1024)
	if err != nil {
		t.Fatalf("Failed to create Partial HE: %v", err)
	}

	m := int64(50)
	scalar := int64(3)

	ct, err := phe.Encrypt(m)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	ctMult := phe.Multiply(ct, scalar)

	result, err := phe.Decrypt(ctMult)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	expected := m * scalar
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestPartialHE_VectorSum(t *testing.T) {
	phe, err := NewPartialHE(1024)
	if err != nil {
		t.Fatalf("Failed to create Partial HE: %v", err)
	}

	values := []int64{10, 20, 30, 40, 50}

	ciphertexts, err := phe.EncryptVector(values)
	if err != nil {
		t.Fatalf("Vector encryption failed: %v", err)
	}

	sum := phe.SumVector(ciphertexts)

	result, err := phe.Decrypt(sum)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	expected := int64(150)
	if result != expected {
		t.Errorf("Expected sum %d, got %d", expected, result)
	}
}

func TestSEALLikeScheme(t *testing.T) {
	scheme := NewSEALLikeScheme(4096, 1024)

	values := []int64{1, 2, 3, 4, 5}

	encoded := scheme.Encode(values)
	if len(encoded) == 0 {
		t.Error("Encoding should produce output")
	}

	decoded := scheme.Decode(encoded)
	if len(decoded) != len(values) {
		t.Errorf("Expected decoded length %d, got %d", len(values), len(decoded))
	}
}

func TestSEALLike_AddPlain(t *testing.T) {
	scheme := NewSEALLikeScheme(4096, 1024)

	encrypted := []int64{100, 200, 300}
	plaintext := []int64{10, 20, 30}

	result := scheme.AddPlain(encrypted, plaintext)

	for i := range result {
		expected := (encrypted[i] + plaintext[i]) % scheme.plainModulus
		if result[i] != expected {
			t.Errorf("At index %d: expected %d, got %d", i, expected, result[i])
		}
	}
}

func TestHETest(t *testing.T) {
	test := NewHETest()

	errors := test.RunAllTests()
	if len(errors) > 0 {
		for _, err := range errors {
			t.Errorf("HETest error: %v", err)
		}
	}
}
