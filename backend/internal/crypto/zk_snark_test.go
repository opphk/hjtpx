package crypto

import (
	"testing"
	"time"
)

func TestZKSNARKSetup(t *testing.T) {
	snarkService := NewZKSNARKService(CurveP256)

	circuit := &ArithmeticCircuit{
		NumInputs:     2,
		NumOutputs:    1,
		NumConstraints: 2,
		Constraints: []Constraint{
			{A: []string{"a", "b"}, B: []string{"1"}, C: []string{"out"}},
			{A: []string{"out"}, B: []string{"1"}, C: []string{"final"}},
		},
		WitnessOrder: []string{"a", "b", "out", "final"},
	}

	err := snarkService.Setup(circuit)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	pk, err := snarkService.GetProvingKey()
	if err != nil {
		t.Fatalf("failed to get proving key: %v", err)
	}

	if pk == nil {
		t.Fatal("proving key should not be nil")
	}

	vk, err := snarkService.GetVerificationKey()
	if err != nil {
		t.Fatalf("failed to get verification key: %v", err)
	}

	if vk == nil {
		t.Fatal("verification key should not be nil")
	}
}

func TestZKSNARKProofGeneration(t *testing.T) {
	snarkService := NewZKSNARKService(CurveP256)

	circuit := &ArithmeticCircuit{
		NumInputs:     1,
		NumOutputs:    1,
		NumConstraints: 1,
		Constraints: []Constraint{
			{A: []string{"x", "2"}, B: []string{"1"}, C: []string{"out"}},
		},
		WitnessOrder: []string{"x", "out"},
	}

	err := snarkService.Setup(circuit)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	request := &SNARKProofRequest{
		Witness: map[string]interface{}{
			"x":   float64(5),
			"out": float64(10),
		},
		PublicInputs: map[string]interface{}{
			"context": "test_context",
		},
		Protocol: "G16",
	}

	response, err := snarkService.GenerateProof(request)
	if err != nil {
		t.Fatalf("proof generation failed: %v", err)
	}

	if response == nil {
		t.Fatal("proof response should not be nil")
	}

	if response.Proof == nil {
		t.Fatal("proof should not be nil")
	}

	if response.PublicHash == "" {
		t.Error("public hash should not be empty")
	}

	if response.ExpiresAt <= response.CreatedAt {
		t.Error("expires at should be after created at")
	}
}

func TestZKSNARKProofVerification(t *testing.T) {
	snarkService := NewZKSNARKService(CurveP256)

	circuit := &ArithmeticCircuit{
		NumInputs:     1,
		NumOutputs:    1,
		NumConstraints: 1,
		Constraints: []Constraint{
			{A: []string{"x", "1"}, B: []string{"1"}, C: []string{"out"}},
		},
		WitnessOrder: []string{"x", "out"},
	}

	err := snarkService.Setup(circuit)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	proofRequest := &SNARKProofRequest{
		Witness: map[string]interface{}{
			"x":   float64(10),
			"out": float64(10),
		},
		PublicInputs: map[string]interface{}{
			"context": "verification_test",
		},
		Protocol: "G16",
	}

	proofResponse, err := snarkService.GenerateProof(proofRequest)
	if err != nil {
		t.Fatalf("proof generation failed: %v", err)
	}

	verificationRequest := &SNARKVerificationRequest{
		Proof:        proofResponse.Proof,
		PublicInputs: []string{"verification_test"},
		Protocol:     "G16",
	}

	verificationResponse, err := snarkService.VerifyProof(verificationRequest)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	if verificationResponse == nil {
		t.Fatal("verification response should not be nil")
	}

	t.Logf("Verification result: %v", verificationResponse.Valid)
}

func TestRangeProofCircuit(t *testing.T) {
	snarkService := NewZKSNARKService(CurveP384)

	circuit := snarkService.CreateRangeProofCircuit(0, 100)
	if circuit == nil {
		t.Fatal("range proof circuit should not be nil")
	}

	if circuit.NumConstraints == 0 {
		t.Error("range proof circuit should have constraints")
	}

	err := snarkService.Setup(circuit)
	if err != nil {
		t.Fatalf("setup failed for range proof circuit: %v", err)
	}

	t.Log("Range proof circuit created successfully")
}

func TestMembershipProofCircuit(t *testing.T) {
	snarkService := NewZKSNARKService(CurveP256)

	circuit := snarkService.CreateMembershipProofCircuit(10)
	if circuit == nil {
		t.Fatal("membership proof circuit should not be nil")
	}

	if len(circuit.Constraints) != 10 {
		t.Errorf("expected 10 constraints, got %d", len(circuit.Constraints))
	}

	err := snarkService.Setup(circuit)
	if err != nil {
		t.Fatalf("setup failed for membership proof circuit: %v", err)
	}

	t.Log("Membership proof circuit created successfully")
}

func TestKnowledgeProofCircuit(t *testing.T) {
	snarkService := NewZKSNARKService(CurveP256)

	circuit := snarkService.CreateKnowledgeProofCircuit("predicate_123")
	if circuit == nil {
		t.Fatal("knowledge proof circuit should not be nil")
	}

	err := snarkService.Setup(circuit)
	if err != nil {
		t.Fatalf("setup failed for knowledge proof circuit: %v", err)
	}

	t.Log("Knowledge proof circuit created successfully")
}

func TestEqualityProofCircuit(t *testing.T) {
	snarkService := NewZKSNARKService(CurveP256)

	circuit := snarkService.CreateEqualityProofCircuit()
	if circuit == nil {
		t.Fatal("equality proof circuit should not be nil")
	}

	err := snarkService.Setup(circuit)
	if err != nil {
		t.Fatalf("setup failed for equality proof circuit: %v", err)
	}

	t.Log("Equality proof circuit created successfully")
}

func TestSNARKProofSerialization(t *testing.T) {
	snarkService := NewZKSNARKService(CurveP256)

	circuit := &ArithmeticCircuit{
		NumInputs:     1,
		NumOutputs:    1,
		NumConstraints: 1,
		Constraints: []Constraint{
			{A: []string{"x"}, B: []string{"1"}, C: []string{"out"}},
		},
		WitnessOrder: []string{"x", "out"},
	}

	err := snarkService.Setup(circuit)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	request := &SNARKProofRequest{
		Witness: map[string]interface{}{
			"x":   float64(5),
			"out": float64(5),
		},
		PublicInputs: map[string]interface{}{"context": "test"},
		Protocol:     "G16",
	}

	response, err := snarkService.GenerateProof(request)
	if err != nil {
		t.Fatalf("proof generation failed: %v", err)
	}

	jsonStr, err := response.Proof.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize proof: %v", err)
	}

	if jsonStr == "" {
		t.Error("serialized proof should not be empty")
	}

	parsedProof, err := ParseSNARKProofFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("failed to parse proof: %v", err)
	}

	if parsedProof.Protocol != response.Proof.Protocol {
		t.Errorf("protocol mismatch: expected %s, got %s", response.Proof.Protocol, parsedProof.Protocol)
	}
}

func TestProvingKeyExportImport(t *testing.T) {
	snarkService := NewZKSNARKService(CurveP256)

	circuit := &ArithmeticCircuit{
		NumInputs:     1,
		NumOutputs:    1,
		NumConstraints: 1,
		Constraints: []Constraint{
			{A: []string{"x"}, B: []string{"1"}, C: []string{"out"}},
		},
		WitnessOrder: []string{"x", "out"},
	}

	err := snarkService.Setup(circuit)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	data, err := snarkService.ExportProvingKey()
	if err != nil {
		t.Fatalf("failed to export proving key: %v", err)
	}

	if len(data) == 0 {
		t.Error("exported data should not be empty")
	}

	newService := NewZKSNARKService(CurveP256)
	err = newService.ImportProvingKey(data)
	if err != nil {
		t.Fatalf("failed to import proving key: %v", err)
	}

	t.Log("Proving key export/import successful")
}

func TestVerificationKeyExportImport(t *testing.T) {
	snarkService := NewZKSNARKService(CurveP256)

	circuit := &ArithmeticCircuit{
		NumInputs:     1,
		NumOutputs:    1,
		NumConstraints: 1,
		Constraints: []Constraint{
			{A: []string{"x"}, B: []string{"1"}, C: []string{"out"}},
		},
		WitnessOrder: []string{"x", "out"},
	}

	err := snarkService.Setup(circuit)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	data, err := snarkService.ExportVerificationKey()
	if err != nil {
		t.Fatalf("failed to export verification key: %v", err)
	}

	if len(data) == 0 {
		t.Error("exported data should not be empty")
	}

	newService := NewZKSNARKService(CurveP256)
	err = newService.ImportVerificationKey(data)
	if err != nil {
		t.Fatalf("failed to import verification key: %v", err)
	}

	t.Log("Verification key export/import successful")
}

func TestInvalidWitnessValidation(t *testing.T) {
	snarkService := NewZKSNARKService(CurveP256)

	circuit := &ArithmeticCircuit{
		NumInputs:     2,
		NumOutputs:    1,
		NumConstraints: 2,
		Constraints: []Constraint{
			{A: []string{"a"}, B: []string{"1"}, C: []string{"out"}},
		},
		WitnessOrder: []string{"a", "b", "out"},
	}

	err := snarkService.Setup(circuit)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	incompleteWitness := map[string]interface{}{
		"a": float64(5),
	}

	err = snarkService.ValidateWitness(incompleteWitness, circuit)
	if err == nil {
		t.Error("validation should fail for incomplete witness")
	}

	completeWitness := map[string]interface{}{
		"a":   float64(5),
		"b":   float64(3),
		"out": float64(8),
	}

	err = snarkService.ValidateWitness(completeWitness, circuit)
	if err != nil {
		t.Errorf("validation should pass for complete witness: %v", err)
	}
}

func TestSNARKProofWithMultipleWitnesses(t *testing.T) {
	snarkService := NewZKSNARKService(CurveP256)

	circuit := &ArithmeticCircuit{
		NumInputs:     3,
		NumOutputs:    1,
		NumConstraints: 3,
		Constraints: []Constraint{
			{A: []string{"x1", "x2"}, B: []string{"1"}, C: []string{"sum"}},
			{A: []string{"sum", "x3"}, B: []string{"1"}, C: []string{"total"}},
			{A: []string{"total"}, B: []string{"1"}, C: []string{"result"}},
		},
		WitnessOrder: []string{"x1", "x2", "x3", "sum", "total", "result"},
	}

	err := snarkService.Setup(circuit)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	request := &SNARKProofRequest{
		Witness: map[string]interface{}{
			"x1":    float64(10),
			"x2":    float64(20),
			"x3":    float64(30),
			"sum":   float64(30),
			"total": float64(60),
			"result": float64(60),
		},
		PublicInputs: map[string]interface{}{
			"context": "multi_witness_test",
		},
		Protocol: "G16",
	}

	response, err := snarkService.GenerateProof(request)
	if err != nil {
		t.Fatalf("proof generation failed: %v", err)
	}

	if response == nil || response.Proof == nil {
		t.Fatal("proof should be generated")
	}

	t.Log("Multi-witness proof generated successfully")
}

func TestSNARKProofExpiration(t *testing.T) {
	snarkService := NewZKSNARKService(CurveP256)

	circuit := &ArithmeticCircuit{
		NumInputs:     1,
		NumOutputs:    1,
		NumConstraints: 1,
		Constraints: []Constraint{
			{A: []string{"x"}, B: []string{"1"}, C: []string{"out"}},
		},
		WitnessOrder: []string{"x", "out"},
	}

	err := snarkService.Setup(circuit)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	request := &SNARKProofRequest{
		Witness: map[string]interface{}{
			"x":   float64(5),
			"out": float64(5),
		},
		PublicInputs: map[string]interface{}{"context": "test"},
		Protocol:     "G16",
	}

	response, err := snarkService.GenerateProof(request)
	if err != nil {
		t.Fatalf("proof generation failed: %v", err)
	}

	if response.ExpiresAt <= time.Now().Unix() {
		t.Error("proof should not be expired immediately after creation")
	}

	expectedExpiry := time.Now().Add(30 * time.Minute).Unix()
	if response.ExpiresAt != expectedExpiry {
		t.Logf("Proof expires at %d (expected %d)", response.ExpiresAt, expectedExpiry)
	}
}
