package crypto

import (
	"crypto/elliptic"
	"encoding/json"
	"math/big"
	"testing"
	"time"
)

func TestZKProofGeneration(t *testing.T) {
	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create ZK proof generator: %v", err)
	}

	witness := &Witness{
		SecretValues: []string{"secret_value_1", "secret_value_2"},
		WitnessType:  string(StatementKnowledge),
	}

	publicInput := &PublicInput{
		Values: map[string]interface{}{
			"context": "test_context",
			"app_id": "test_app",
		},
		CurveType: CurveP256,
	}

	proof, err := generator.CreateProof(witness, publicInput, StatementKnowledge)
	if err != nil {
		t.Fatalf("failed to create proof: %v", err)
	}

	if proof == nil {
		t.Fatal("proof should not be nil")
	}

	if len(proof.ProofData) == 0 {
		t.Error("proof data should not be empty")
	}

	if proof.Statement != StatementKnowledge {
		t.Errorf("expected statement type %s, got %s", StatementKnowledge, proof.Statement)
	}

	if proof.CurveType != CurveP256 {
		t.Errorf("expected curve type %s, got %s", CurveP256, proof.CurveType)
	}

	if proof.CreatedAt == 0 {
		t.Error("created at should be set")
	}

	if proof.ExpiresAt <= proof.CreatedAt {
		t.Error("expires at should be after created at")
	}
}

func TestZKProofVerification(t *testing.T) {
	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	verifier, err := NewZKProofVerifier(CurveP256)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	witness := &Witness{
		SecretValues: []string{"test_secret"},
		WitnessType:  string(StatementRangeProof),
	}

	publicInput := &PublicInput{
		Values: map[string]interface{}{
			"context": "verification_test",
		},
		CurveType: CurveP256,
	}

	proof, err := generator.CreateProof(witness, publicInput, StatementRangeProof)
	if err != nil {
		t.Fatalf("failed to create proof: %v", err)
	}

	valid, err := verifier.VerifyProof(proof)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if !valid {
		t.Error("proof should be valid")
	}
}

func TestZKProofExpiration(t *testing.T) {
	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	witness := &Witness{
		SecretValues: []string{"expiring_secret"},
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

	if proof.IsExpired() {
		t.Error("freshly created proof should not be expired")
	}

	remaining := proof.RemainingValidity()
	if remaining <= 0 {
		t.Error("remaining validity should be positive for fresh proof")
	}

	proof.ExpiresAt = time.Now().Add(-1 * time.Hour).Unix()
	if !proof.IsExpired() {
		t.Error("expired proof should report as expired")
	}
}

func TestCommitmentGeneration(t *testing.T) {
	generator, err := NewZKProofGenerator(CurveP384)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	witness := &Witness{
		SecretValues: []string{"commitment_secret"},
		WitnessType:  string(StatementEquality),
	}

	publicInput := &PublicInput{
		Values: map[string]interface{}{
			"context": "commitment_test",
			"nonce":  "random_nonce",
		},
		CurveType: CurveP384,
	}

	commitment, err := generator.ComputeCommitment(witness, publicInput)
	if err != nil {
		t.Fatalf("failed to compute commitment: %v", err)
	}

	if commitment == nil {
		t.Fatal("commitment should not be nil")
	}

	if commitment.Commitment == "" {
		t.Error("commitment value should not be empty")
	}

	if commitment.Challenge == "" {
		t.Error("challenge should not be empty")
	}

	if commitment.Response == "" {
		t.Error("response should not be empty")
	}

	if commitment.CreatedAt == 0 {
		t.Error("created at should be set")
	}
}

func TestProofSerialization(t *testing.T) {
	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	witness := &Witness{
		SecretValues: []string{"serialization_test"},
		WitnessType:  string(StatementKnowledge),
	}

	publicInput := &PublicInput{
		Values:    map[string]interface{}{"context": "test"},
		CurveType: CurveP256,
	}

	originalProof, err := generator.CreateProof(witness, publicInput, StatementKnowledge)
	if err != nil {
		t.Fatalf("failed to create proof: %v", err)
	}

	jsonStr, err := originalProof.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize proof: %v", err)
	}

	if jsonStr == "" {
		t.Error("serialized proof should not be empty")
	}

	parsedProof, err := ParseProofFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("failed to parse proof: %v", err)
	}

	if parsedProof.Statement != originalProof.Statement {
		t.Errorf("statement mismatch: expected %s, got %s", originalProof.Statement, parsedProof.Statement)
	}

	if parsedProof.CurveType != originalProof.CurveType {
		t.Errorf("curve type mismatch: expected %s, got %s", originalProof.CurveType, parsedProof.CurveType)
	}
}

func TestChallengeGeneration(t *testing.T) {
	verifier, err := NewZKProofVerifier(CurveP256)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	publicInputs := map[string]string{
		"app_id": "test_app",
		"user_id": "user_123",
	}

	challenge, err := verifier.GenerateChallenge(publicInputs)
	if err != nil {
		t.Fatalf("failed to generate challenge: %v", err)
	}

	if challenge == nil {
		t.Fatal("challenge should not be nil")
	}

	if challenge.Challenge == "" {
		t.Error("challenge value should not be empty")
	}

	if challenge.RandomNonce == "" {
		t.Error("random nonce should not be empty")
	}

	if challenge.Timestamp == 0 {
		t.Error("timestamp should be set")
	}
}

func TestStatementTypes(t *testing.T) {
	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	verifier, err := NewZKProofVerifier(CurveP256)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	statementTypes := []StatementType{
		StatementRangeProof,
		StatementSetMembership,
		StatementKnowledge,
		StatementEquality,
	}

	for _, stmtType := range statementTypes {
		t.Run(string(stmtType), func(t *testing.T) {
			witness := &Witness{
				SecretValues: []string{"test_secret"},
				WitnessType:  string(stmtType),
			}

			publicInput := &PublicInput{
				Values:    map[string]interface{}{"context": "test"},
				CurveType: CurveP256,
			}

			proof, err := generator.CreateProof(witness, publicInput, stmtType)
			if err != nil {
				t.Fatalf("failed to create proof for %s: %v", stmtType, err)
			}

			valid, err := verifier.VerifyProof(proof)
			if err != nil {
				t.Fatalf("verification failed for %s: %v", stmtType, err)
			}

			if !valid {
				t.Errorf("proof should be valid for statement type %s", stmtType)
			}
		})
	}
}

func TestCurveTypes(t *testing.T) {
	curveTypes := []CurveType{CurveP256, CurveP384, CurveP521}

	for _, curveType := range curveTypes {
		t.Run(string(curveType), func(t *testing.T) {
			generator, err := NewZKProofGenerator(curveType)
			if err != nil {
				t.Fatalf("failed to create generator for %s: %v", curveType, err)
			}

			verifier, err := NewZKProofVerifier(curveType)
			if err != nil {
				t.Fatalf("failed to create verifier for %s: %v", curveType, err)
			}

			witness := &Witness{
				SecretValues: []string{"curve_test"},
				WitnessType:  string(StatementKnowledge),
			}

			publicInput := &PublicInput{
				Values:    map[string]interface{}{"context": "test"},
				CurveType: curveType,
			}

			proof, err := generator.CreateProof(witness, publicInput, StatementKnowledge)
			if err != nil {
				t.Fatalf("failed to create proof for %s: %v", curveType, err)
			}

			valid, err := verifier.VerifyProof(proof)
			if err != nil {
				t.Fatalf("verification failed for %s: %v", curveType, err)
			}

			if !valid {
				t.Errorf("proof should be valid for curve %s", curveType)
			}
		})
	}
}

func TestProofMetadata(t *testing.T) {
	metadata := CreateProofMetadata("prover_123", "verifier_456", StatementKnowledge, 30*time.Minute)

	if metadata == nil {
		t.Fatal("metadata should not be nil")
	}

	if metadata.ProverID != "prover_123" {
		t.Errorf("expected prover ID prover_123, got %s", metadata.ProverID)
	}

	if metadata.VerifierID != "verifier_456" {
		t.Errorf("expected verifier ID verifier_456, got %s", metadata.VerifierID)
	}

	if metadata.StatementType != StatementKnowledge {
		t.Errorf("expected statement type %s, got %s", StatementKnowledge, metadata.StatementType)
	}

	if metadata.Result {
		t.Error("new metadata should not have result set")
	}

	if !metadata.VerifiedAt.IsZero() {
		t.Error("new metadata should not have verified at set")
	}

	metadata.MarkVerified(true)

	if !metadata.Result {
		t.Error("result should be true after marking verified")
	}

	if metadata.VerifiedAt.IsZero() {
		t.Error("verified at should be set after marking verified")
	}
}

func TestInvalidProof(t *testing.T) {
	verifier, err := NewZKProofVerifier(CurveP256)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	tests := []struct {
		name  string
		proof *ZKProof
	}{
		{"nil proof", nil},
		{"empty proof", &ZKProof{}},
		{"empty proof data", &ZKProof{Statement: StatementKnowledge, CurveType: CurveP256}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := verifier.VerifyProof(tt.proof)
			if err == nil && valid {
				t.Errorf("proof %s should fail verification", tt.name)
			}
		})
	}
}

func TestRangeProof(t *testing.T) {
	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	verifier, err := NewZKProofVerifier(CurveP256)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	rangeProof := &RangeProof{
		A:          "commitment_a",
		B:          "commitment_b",
		C:          "commitment_c",
		Z:          "randomness",
		LowerBound: 0,
		UpperBound: 100,
	}

	witness := &Witness{
		SecretValues: []string{"range_secret"},
		WitnessType:  string(StatementRangeProof),
	}

	publicInput := &PublicInput{
		Values: map[string]interface{}{
			"context":    "range_test",
			"lower":     rangeProof.LowerBound,
			"upper":     rangeProof.UpperBound,
		},
		CurveType: CurveP256,
	}

	proof, err := generator.CreateProof(witness, publicInput, StatementRangeProof)
	if err != nil {
		t.Fatalf("failed to create range proof: %v", err)
	}

	valid, err := verifier.VerifyProof(proof)
	if err != nil {
		t.Fatalf("range proof verification failed: %v", err)
	}

	if !valid {
		t.Error("range proof should be valid")
	}

	t.Logf("Range proof created successfully with bounds [%d, %d]", rangeProof.LowerBound, rangeProof.UpperBound)
}

func TestMembershipProof(t *testing.T) {
	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	verifier, err := NewZKProofVerifier(CurveP256)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	membershipProof := &MembershipProof{
		Commitment:    "member_commitment",
		ProofElements: []string{"elem1", "elem2", "elem3"},
		SetHash:       "set_hash_value",
		ProofType:     "merkle",
	}

	witness := &Witness{
		SecretValues: []string{"membership_secret"},
		WitnessType:  string(StatementSetMembership),
	}

	publicInput := &PublicInput{
		Values: map[string]interface{}{
			"context": "membership_test",
			"set_id":  "allowed_users",
		},
		CurveType: CurveP256,
	}

	proof, err := generator.CreateProof(witness, publicInput, StatementSetMembership)
	if err != nil {
		t.Fatalf("failed to create membership proof: %v", err)
	}

	valid, err := verifier.VerifyProof(proof)
	if err != nil {
		t.Fatalf("membership proof verification failed: %v", err)
	}

	if !valid {
		t.Error("membership proof should be valid")
	}

	t.Logf("Membership proof created with %d elements", len(membershipProof.ProofElements))
}

func TestEqualityProof(t *testing.T) {
	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	verifier, err := NewZKProofVerifier(CurveP256)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	witness := &Witness{
		SecretValues: []string{"equality_secret"},
		WitnessType:  string(StatementEquality),
	}

	publicInput := &PublicInput{
		Values: map[string]interface{}{
			"context": "equality_test",
			"target":  "same_value",
		},
		CurveType: CurveP256,
	}

	proof, err := generator.CreateProof(witness, publicInput, StatementEquality)
	if err != nil {
		t.Fatalf("failed to create equality proof: %v", err)
	}

	valid, err := verifier.VerifyProof(proof)
	if err != nil {
		t.Fatalf("equality proof verification failed: %v", err)
	}

	if !valid {
		t.Error("equality proof should be valid")
	}
}

func TestZKSystemParams(t *testing.T) {
	params := &ZKSystemParams{
		Curve:        elliptic.P256(),
		CurveType:    CurveP256,
		G:            &Point{X: big.NewInt(1), Y: big.NewInt(2)},
		H:            &Point{X: big.NewInt(3), Y: big.NewInt(4)},
		Generator:    &Point{X: big.NewInt(5), Y: big.NewInt(6)},
		SetupTime:    time.Now().Unix(),
		TrustedSetup: false,
		ProofSize:    64,
	}

	if params.CurveType != CurveP256 {
		t.Errorf("expected curve type %s, got %s", CurveP256, params.CurveType)
	}

	if params.ProofSize != 64 {
		t.Errorf("expected proof size 64, got %d", params.ProofSize)
	}

	if params.TrustedSetup {
		t.Error("trusted setup should be false for simulation")
	}
}

func TestProofJSON(t *testing.T) {
	proof := &ZKProof{
		ProofData: []byte("test_proof_data"),
		PublicInputs: map[string]interface{}{
			"app_id": "test_app",
		},
		Statement:   StatementKnowledge,
		CurveType:  CurveP256,
		CreatedAt:  time.Now().Unix(),
		ExpiresAt:  time.Now().Add(30 * time.Minute).Unix(),
		Nonce:      "test_nonce",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	jsonBytes, err := json.Marshal(proof)
	if err != nil {
		t.Fatalf("failed to marshal proof: %v", err)
	}

	var parsedProof ZKProof
	err = json.Unmarshal(jsonBytes, &parsedProof)
	if err != nil {
		t.Fatalf("failed to unmarshal proof: %v", err)
	}

	if parsedProof.Statement != proof.Statement {
		t.Errorf("statement mismatch")
	}

	if parsedProof.CurveType != proof.CurveType {
		t.Errorf("curve type mismatch")
	}
}

func TestNonceGeneration(t *testing.T) {
	generator, err := NewZKProofGenerator(CurveP256)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	nonce1, err := generator.GenerateNonce()
	if err != nil {
		t.Fatalf("failed to generate nonce: %v", err)
	}

	nonce2, err := generator.GenerateNonce()
	if err != nil {
		t.Fatalf("failed to generate second nonce: %v", err)
	}

	if nonce1 == nonce2 {
		t.Error("nonces should be unique")
	}

	if len(nonce1) == 0 {
		t.Error("nonce should not be empty")
	}
}
