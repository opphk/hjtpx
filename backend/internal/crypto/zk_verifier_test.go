package crypto

import (
	"testing"
	"time"
)

func TestZKVerifierCreation(t *testing.T) {
	verifier, err := NewZKVerifier(CurveP256, nil)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	if verifier == nil {
		t.Fatal("verifier should not be nil")
	}
}

func TestZKVerifierWithConfig(t *testing.T) {
	config := &VerificationConfig{
		Mode:              ModeStrict,
		Timeout:           30 * time.Second,
		EnableReplayCheck: true,
		MaxProofAge:       30 * time.Minute,
		StrictValidation:  true,
		EnableProofCaching: true,
		CacheTTL:          5 * time.Minute,
	}

	verifier, err := NewZKVerifier(CurveP256, config)
	if err != nil {
		t.Fatalf("failed to create verifier with config: %v", err)
	}

	if verifier == nil {
		t.Fatal("verifier should not be nil")
	}
}

func TestProofVerification(t *testing.T) {
	verifier, err := NewZKVerifier(CurveP256, nil)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	witness := &Witness{
		SecretValues: []string{"verification_test"},
		WitnessType:  string(StatementKnowledge),
	}

	publicInput := &PublicInput{
		Values:    map[string]interface{}{"context": "test"},
		CurveType: CurveP256,
	}

	proof, err := generator.CreateProof(witness, publicInput, StatementKnowledge)
	if err != nil {
		t.Fatalf("failed to create proof: %v", err)
	}

	request := &VerificationRequest{
		Proof:        proof,
		PublicInputs: publicInput.Values,
		SessionID:    "test_session",
	}

	result, err := verifier.Verify(request)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if result == nil {
		t.Fatal("verification result should not be nil")
	}

	if result.ProofID == "" {
		t.Error("proof ID should be set")
	}

	if result.Duration < 0 {
		t.Error("duration should be non-negative")
	}
}

func TestZKBatchVerification(t *testing.T) {
	verifier, err := NewZKVerifier(CurveP256, nil)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	requests := make([]VerificationRequest, 0, 3)
	for i := 0; i < 3; i++ {
		witness := &Witness{
			SecretValues: []string{"batch_test"},
			WitnessType:  string(StatementKnowledge),
		}

		publicInput := &PublicInput{
			Values:    map[string]interface{}{"index": i},
			CurveType: CurveP256,
		}

		proof, err := generator.CreateProof(witness, publicInput, StatementKnowledge)
		if err != nil {
			t.Fatalf("failed to create proof %d: %v", i, err)
		}

		requests = append(requests, VerificationRequest{
			Proof:        proof,
			PublicInputs: publicInput.Values,
			SessionID:    "batch_session",
		})
	}

	batchRequest := &BatchVerificationRequest{
		Proofs: requests,
		Mode:   ModeStandard,
	}

	batchResult, err := verifier.BatchVerify(batchRequest)
	if err != nil {
		t.Fatalf("batch verification failed: %v", err)
	}

	if batchResult == nil {
		t.Fatal("batch result should not be nil")
	}

	if len(batchResult.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(batchResult.Results))
	}

	if batchResult.TotalDuration <= 0 {
		t.Error("total duration should be positive")
	}
}

func TestReplayAttackDetection(t *testing.T) {
	config := &VerificationConfig{
		Mode:              ModeStandard,
		EnableReplayCheck: true,
		EnableProofCaching: false,
	}

	verifier, err := NewZKVerifier(CurveP256, config)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	witness := &Witness{
		SecretValues: []string{"replay_test"},
		WitnessType:  string(StatementKnowledge),
	}

	publicInput := &PublicInput{
		Values:    map[string]interface{}{"context": "test"},
		CurveType: CurveP256,
	}

	proof, err := generator.CreateProof(witness, publicInput, StatementKnowledge)
	if err != nil {
		t.Fatalf("failed to create proof: %v", err)
	}
	t.Logf("Proof data length: %d", len(proof.ProofData))

	request := &VerificationRequest{
		Proof:        proof,
		PublicInputs: publicInput.Values,
	}

	result1, err := verifier.Verify(request)
	if err != nil {
		t.Fatalf("first verification failed: %v", err)
	}
	t.Logf("First verification result: valid=%v, proof=%p", result1.Valid, proof)

	result2, err := verifier.Verify(request)
	t.Logf("Second verification result: valid=%v, err=%v", result2.Valid, err)
	if err == nil {
		t.Error("replay attack should be detected")
	} else if result2.Valid {
		t.Error("replay attack should result in invalid proof")
	}
}

func TestProofExpiration(t *testing.T) {
	config := &VerificationConfig{
		Mode:              ModeStandard,
		EnableReplayCheck: false,
		MaxProofAge:       1 * time.Second,
	}

	verifier, err := NewZKVerifier(CurveP256, config)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	witness := &Witness{
		SecretValues: []string{"expiry_test"},
		WitnessType:  string(StatementKnowledge),
	}

	publicInput := &PublicInput{
		Values:    map[string]interface{}{"context": "test"},
		CurveType: CurveP256,
	}

	proof, err := generator.CreateProof(witness, publicInput, StatementKnowledge)
	if err != nil {
		t.Fatalf("failed to create proof: %v", err)
	}

	request := &VerificationRequest{
		Proof:        proof,
		PublicInputs: publicInput.Values,
	}

	_, err = verifier.Verify(request)
	if err != nil {
		t.Fatalf("fresh proof should verify: %v", err)
	}

	proof.ExpiresAt = time.Now().Add(-2 * time.Second).Unix()

	_, err = verifier.Verify(request)
	if err == nil {
		t.Error("expired proof should fail verification")
	}
}

func TestVerificationStats(t *testing.T) {
	verifier, err := NewZKVerifier(CurveP256, nil)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	stats := verifier.GetStats()
	if stats == nil {
		t.Fatal("stats should not be nil")
	}

	if stats.TotalVerifications != 0 {
		t.Error("initial total verifications should be 0")
	}

	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	witness := &Witness{
		SecretValues: []string{"stats_test"},
		WitnessType:  string(StatementKnowledge),
	}

	publicInput := &PublicInput{
		Values:    map[string]interface{}{"context": "test"},
		CurveType: CurveP256,
	}

	proof, _ := generator.CreateProof(witness, publicInput, StatementKnowledge)

	request := &VerificationRequest{
		Proof:        proof,
		PublicInputs: publicInput.Values,
	}

	verifier.Verify(request)

	stats = verifier.GetStats()
	if stats.TotalVerifications != 1 {
		t.Errorf("expected 1 verification, got %d", stats.TotalVerifications)
	}
}

func TestVerificationModes(t *testing.T) {
	modes := []VerificationMode{ModeStrict, ModeStandard, ModeFast}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			config := &VerificationConfig{
				Mode:              mode,
				EnableReplayCheck: false,
				StrictValidation:  mode == ModeStrict,
			}

			verifier, err := NewZKVerifier(CurveP256, config)
			if err != nil {
				t.Fatalf("failed to create verifier for mode %s: %v", mode, err)
			}

			if verifier == nil {
				t.Error("verifier should not be nil")
			}
		})
	}
}

func TestInvalidProofVerification(t *testing.T) {
	verifier, err := NewZKVerifier(CurveP256, nil)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	tests := []struct {
		name    string
		request *VerificationRequest
	}{
		{
			name:    "nil request",
			request: nil,
		},
		{
			name: "nil proof",
			request: &VerificationRequest{
				Proof: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := verifier.Verify(tt.request)
			if err == nil && result != nil && result.Valid {
				t.Errorf("verification should fail for %s", tt.name)
			}
		})
	}
}

func TestZKChallengeGeneration(t *testing.T) {
	verifier, err := NewZKVerifier(CurveP256, nil)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	publicInputs := map[string]string{
		"app_id": "test_app",
		"user_id": "user_123",
	}

	challenge, err := verifier.GenerateChallenge(publicInputs)
	if err != nil {
		t.Fatalf("challenge generation failed: %v", err)
	}

	if challenge == nil {
		t.Fatal("challenge should not be nil")
	}

	if challenge.Challenge == "" {
		t.Error("challenge value should not be empty")
	}

	if challenge.VerifierID == "" {
		t.Error("verifier ID should not be empty")
	}

	if challenge.ExpiresAt <= time.Now().Unix() {
		t.Error("challenge should have future expiry time")
	}
}

func TestProofCache(t *testing.T) {
	config := &VerificationConfig{
		Mode:                ModeStandard,
		EnableProofCaching:  true,
		CacheTTL:            5 * time.Minute,
	}

	verifier, err := NewZKVerifier(CurveP256, config)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	witness := &Witness{
		SecretValues: []string{"cache_test"},
		WitnessType:  string(StatementKnowledge),
	}

	publicInput := &PublicInput{
		Values:    map[string]interface{}{"context": "test"},
		CurveType: CurveP256,
	}

	proof, err := generator.CreateProof(witness, publicInput, StatementKnowledge)
	if err != nil {
		t.Fatalf("failed to create proof: %v", err)
	}

	request := &VerificationRequest{
		Proof:        proof,
		PublicInputs: publicInput.Values,
	}

	_, err = verifier.Verify(request)
	if err != nil {
		t.Fatalf("first verification failed: %v", err)
	}

	verifier.CleanupExpiredCache()
}

func TestVerifierExportImportKey(t *testing.T) {
	verifier, err := NewZKVerifier(CurveP256, nil)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	data, err := verifier.ExportVerificationKey()
	if err != nil {
		t.Fatalf("failed to export key: %v", err)
	}

	if len(data) == 0 {
		t.Error("exported data should not be empty")
	}

	newVerifier, err := NewZKVerifier(CurveP256, nil)
	if err != nil {
		t.Fatalf("failed to create new verifier: %v", err)
	}

	err = newVerifier.ImportVerificationKey(data)
	if err != nil {
		t.Fatalf("failed to import key: %v", err)
	}

	t.Log("Verification key export/import successful")
}

func TestVerifierService(t *testing.T) {
	service := NewVerifierService(CurveP256, nil)
	if service == nil {
		t.Fatal("verifier service should not be nil")
	}

	stats := service.GetStats()
	if stats == nil {
		t.Fatal("stats should not be nil")
	}

	policy := &VerificationPolicy{
		PolicyID:       "test_policy",
		AllowedTypes:   []StatementType{StatementKnowledge, StatementRangeProof},
		MaxProofAge:    30 * time.Minute,
		RequireFreshness: true,
	}

	service.AddPolicy(policy)

	retrieved := service.GetPolicy("test_policy")
	if retrieved == nil {
		t.Error("policy should be retrievable")
	}
}
