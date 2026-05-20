package service

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrivacyComputingService(t *testing.T) {
	service := NewPrivacyComputingService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.heEngine)
	assert.NotNil(t, service.mpcEngine)
	assert.NotNil(t, service.flEngine)
	assert.NotNil(t, service.pdsEngine)
	assert.NotNil(t, service.privacyBudget)
}

func TestHEKeyGeneration(t *testing.T) {
	he := NewHEEngine()

	pk, sk, err := he.GenerateKeyPair()
	require.NoError(t, err)
	assert.NotNil(t, pk)
	assert.NotNil(t, sk)
	assert.NotNil(t, pk.N)
	assert.NotNil(t, sk.Lambda)
}

func TestHEEncryptionDecryption(t *testing.T) {
	he := NewHEEngine()

	pk, sk, err := he.GenerateKeyPair()
	require.NoError(t, err)

	plaintext := big.NewInt(42)
	ct, err := he.Encrypt(plaintext, pk)
	require.NoError(t, err)
	assert.NotNil(t, ct)

	decrypted, err := he.Decrypt(ct, sk)
	require.NoError(t, err)
	assert.NotNil(t, decrypted)
}

func TestHEAddition(t *testing.T) {
	he := NewHEEngine()

	pk, sk, err := he.GenerateKeyPair()
	require.NoError(t, err)

	m1 := big.NewInt(10)
	m2 := big.NewInt(20)

	ct1, err := he.Encrypt(m1, pk)
	require.NoError(t, err)

	ct2, err := he.Encrypt(m2, pk)
	require.NoError(t, err)

	result, err := he.Add(ct1, ct2)
	require.NoError(t, err)
	assert.NotNil(t, result)

	decrypted, err := he.Decrypt(result, sk)
	require.NoError(t, err)
	assert.NotNil(t, decrypted)
}

func TestHEScalarMultiply(t *testing.T) {
	he := NewHEEngine()

	pk, _, err := he.GenerateKeyPair()
	require.NoError(t, err)

	m := big.NewInt(7)
	ct, err := he.Encrypt(m, pk)
	require.NoError(t, err)

	scalar := big.NewInt(3)
	result, err := he.ScalarMultiply(ct, scalar)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMPCSetupProtocol(t *testing.T) {
	mpc := NewMPCEngine()

	parties := []string{"party1", "party2", "party3"}
	err := mpc.SetupProtocol(MPCProtocolGMW, parties)
	require.NoError(t, err)

	err = mpc.SetupProtocol(MPCProtocolYao, parties)
	require.NoError(t, err)
}

func TestMPCShareSecret(t *testing.T) {
	mpc := NewMPCEngine()

	parties := []string{"party1", "party2", "party3"}
	err := mpc.SetupProtocol(MPCProtocolGMW, parties)
	require.NoError(t, err)

	secret := []byte("test-secret-data")
	shares, err := mpc.ShareSecret("party1", secret, 2)
	require.NoError(t, err)
	assert.NotEmpty(t, shares)
}

func TestMPCReconstructSecret(t *testing.T) {
	mpc := NewMPCEngine()

	parties := []string{"party1", "party2", "party3"}
	err := mpc.SetupProtocol(MPCProtocolGMW, parties)
	require.NoError(t, err)

	secret := []byte("test-secret-data")
	shares, err := mpc.ShareSecret("party1", secret, 2)
	require.NoError(t, err)

	reconstructed, err := mpc.ReconstructSecret(shares, 2)
	require.NoError(t, err)
	assert.NotNil(t, reconstructed)
}

func TestMPCReconstructSecretInsufficientShares(t *testing.T) {
	mpc := NewMPCEngine()

	parties := []string{"party1", "party2", "party3"}
	err := mpc.SetupProtocol(MPCProtocolGMW, parties)
	require.NoError(t, err)

	secret := []byte("test-secret-data")
	shares, err := mpc.ShareSecret("party1", secret, 3)
	require.NoError(t, err)

	_, err = mpc.ReconstructSecret(shares[:2], 3)
	assert.Error(t, err)
	assert.Equal(t, ErrMPCReconstruction, err)
}

func TestMPCCompute(t *testing.T) {
	mpc := NewMPCEngine()

	parties := []string{"party1", "party2", "party3"}
	err := mpc.SetupProtocol(MPCProtocolGMW, parties)
	require.NoError(t, err)

	result, err := mpc.ComputeMPC(parties, "sum")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, MPCProtocolGMW, result.Protocol)
}

func TestFLRegisterNode(t *testing.T) {
	fl := NewFLEngine()

	node := &FLNode{
		NodeID:      "node-1",
		DatasetSize: 1000,
		Status:      "active",
		Reputation:  1.0,
	}

	err := fl.RegisterNode(node)
	require.NoError(t, err)

	assert.NotNil(t, fl.nodes["node-1"])
}

func TestFLInitializeModel(t *testing.T) {
	fl := NewFLEngine()

	initialWeights := []float64{0.1, 0.2, 0.3, 0.4}
	model, err := fl.InitializeModel("model-1", initialWeights)
	require.NoError(t, err)
	assert.NotNil(t, model)
	assert.Equal(t, "model-1", model.ModelID)
	assert.Len(t, model.Weights, 4)
}

func TestFLAggregateUpdates(t *testing.T) {
	fl := NewFLEngine()

	initialWeights := []float64{0.1, 0.2, 0.3, 0.4}
	_, err := fl.InitializeModel("model-1", initialWeights)
	require.NoError(t, err)

	request := &FLAggregationRequest{
		NodeID:      "node-1",
		ModelID:     "model-1",
		Weights:     []float64{0.05, 0.05, 0.05, 0.05},
		Gradient:    []float64{0.01, 0.01, 0.01, 0.01},
		DatasetSize: 100,
	}

	result, err := fl.AggregateUpdates(request)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Len(t, result.AggregatedWeights, 4)
}

func TestFLApplyDifferentialPrivacy(t *testing.T) {
	fl := NewFLEngine()

	gradients := []float64{1.0, 2.0, 3.0, 4.0}
	noised := fl.ApplyDifferentialPrivacy(gradients, 1.0, 1e-5)
	assert.Len(t, noised, 4)
}

func TestPDSRegisterData(t *testing.T) {
	pds := NewPDSEngine()

	he := NewHEEngine()
	pk, _, _ := he.GenerateKeyPair()
	ct, _ := he.Encrypt(big.NewInt(123), pk)

	err := pds.RegisterData("data-1", "sensitive", "owner-1", ct)
	require.NoError(t, err)

	data, err := pds.GetData("data-1")
	require.NoError(t, err)
	assert.Equal(t, "data-1", data.DataID)
}

func TestPDSSetAccessPolicy(t *testing.T) {
	pds := NewPDSEngine()

	he := NewHEEngine()
	pk, _, _ := he.GenerateKeyPair()
	ct, _ := he.Encrypt(big.NewInt(123), pk)

	err := pds.RegisterData("data-1", "sensitive", "owner-1", ct)
	require.NoError(t, err)

	policy := &DataAccessPolicy{
		RequiredZKProof:  true,
		MinTrustLevel:    5,
		AllowedPurposes: []string{"research", "analytics"},
	}

	err = pds.SetAccessPolicy("data-1", policy)
	require.NoError(t, err)
}

func TestPDSRequestAccessOwner(t *testing.T) {
	pds := NewPDSEngine()

	he := NewHEEngine()
	pk, _, _ := he.GenerateKeyPair()
	ct, _ := he.Encrypt(big.NewInt(123), pk)

	err := pds.RegisterData("data-1", "sensitive", "owner-1", ct)
	require.NoError(t, err)

	request := &DataAccessRequest{
		DataID:      "data-1",
		RequesterID: "owner-1",
		Purpose:     "management",
	}

	resp, err := pds.RequestAccess(request)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestPDSRequestAccessWithZKProof(t *testing.T) {
	pds := NewPDSEngine()

	he := NewHEEngine()
	pk, _, _ := he.GenerateKeyPair()
	ct, _ := he.Encrypt(big.NewInt(123), pk)

	err := pds.RegisterData("data-1", "sensitive", "owner-1", ct)
	require.NoError(t, err)

	policy := &DataAccessPolicy{
		RequiredZKProof:  true,
		MinTrustLevel:    5,
		AllowedPurposes: []string{"research"},
	}

	err = pds.SetAccessPolicy("data-1", policy)
	require.NoError(t, err)

	request := &DataAccessRequest{
		DataID:      "data-1",
		RequesterID: "user-1",
		Purpose:     "research",
		ZKProof:     []byte("zk-proof-data"),
	}

	resp, err := pds.RequestAccess(request)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestPrivacyComputingServiceHETO(t *testing.T) {
	service := NewPrivacyComputingService()

	pk, sk, err := service.HEGenerateKeyPair()
	require.NoError(t, err)

	plaintext := big.NewInt(100)
	ct, err := service.HEEncrypt(plaintext, pk)
	require.NoError(t, err)

	decrypted, err := service.HEDecrypt(ct, sk)
	require.NoError(t, err)
	assert.Equal(t, plaintext.Int64(), decrypted.Int64())
}

func TestPrivacyComputingServiceFL(t *testing.T) {
	service := NewPrivacyComputingService()

	node := &FLNode{
		NodeID:      "node-1",
		DatasetSize: 1000,
		Status:      "active",
		Reputation:  1.0,
	}

	err := service.FLRegisterNode(node)
	require.NoError(t, err)

	weights := []float64{0.1, 0.2, 0.3}
	model, err := service.FLInitializeModel("model-1", weights)
	require.NoError(t, err)
	assert.NotNil(t, model)

	request := &FLAggregationRequest{
		NodeID:      "node-1",
		ModelID:     "model-1",
		Weights:     []float64{0.01, 0.01, 0.01},
		DatasetSize: 100,
	}

	result, err := service.FLAggregateUpdates(request)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestPrivacyComputingServiceMPC(t *testing.T) {
	service := NewPrivacyComputingService()

	parties := []string{"party1", "party2", "party3"}
	err := service.MPCSetupProtocol(MPCProtocolGMW, parties)
	require.NoError(t, err)

	secret := []byte("shared-secret")
	shares, err := service.MPCShareSecret("party1", secret, 2)
	require.NoError(t, err)
	assert.NotEmpty(t, shares)
}

func TestPrivacyComputingServicePDS(t *testing.T) {
	service := NewPrivacyComputingService()

	he := NewHEEngine()
	pk, _, _ := he.GenerateKeyPair()
	ct, _ := he.Encrypt(big.NewInt(456), pk)

	err := service.PDSRegisterData("data-1", "sensitive", "owner-1", ct)
	require.NoError(t, err)

	policy := &DataAccessPolicy{
		RequiredZKProof:  false,
		MinTrustLevel:    1,
		AllowedPurposes: []string{"any"},
	}

	err = service.PDSSetAccessPolicy("data-1", policy)
	require.NoError(t, err)
}

func TestPrivacyComputingServicePrivacyBudget(t *testing.T) {
	service := NewPrivacyComputingService()

	budget := &FLPrivacyBudget{
		Epsilon:     0.5,
		Delta:       1e-6,
		NoiseScale:  0.5,
		MaxGradient: 3.0,
	}

	service.SetPrivacyBudget("user-1", budget)

	retrieved := service.GetPrivacyBudget("user-1")
	assert.NotNil(t, retrieved)
	assert.Equal(t, 0.5, retrieved.Epsilon)
}

func TestPrivacyComputingServicePrivacyCheck(t *testing.T) {
	service := NewPrivacyComputingService()

	budget := &FLPrivacyBudget{
		Epsilon:     1.0,
		Delta:       1e-5,
		NoiseScale:  1.0,
		MaxGradient: 5.0,
	}

	data := []float64{1.0, 2.0, 3.0}
	valid := service.PerformPrivacyCheck(data, budget)
	assert.True(t, valid)

	invalidData := []float64{10.0, 20.0, 30.0}
	invalid := service.PerformPrivacyCheck(invalidData, budget)
	assert.False(t, invalid)
}

func TestPrivacyComputingServiceHEOperation(t *testing.T) {
	service := NewPrivacyComputingService()
	ctx := context.Background()

	pk, sk, err := service.HEGenerateKeyPair()
	require.NoError(t, err)

	m1 := big.NewInt(50)
	m2 := big.NewInt(30)

	ct1, err := service.HEEncrypt(m1, pk)
	require.NoError(t, err)

	ct2, err := service.HEEncrypt(m2, pk)
	require.NoError(t, err)

	req := &HEOperationRequest{
		Operation: "add",
		Operand1:  ct1,
		Operand2:  ct2,
	}

	resp, err := service.PerformHEOperation(ctx, req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Result)

	decrypted, err := service.HEDecrypt(resp.Result, sk)
	require.NoError(t, err)
	assert.NotNil(t, decrypted)
}

func TestPrivacyComputingServiceMPCComputation(t *testing.T) {
	service := NewPrivacyComputingService()
	ctx := context.Background()

	parties := []string{"party1", "party2"}

	req := &MPCComputationRequest{
		Protocol:       MPCProtocolGMW,
		PartyIDs:       parties,
		ComputationType: "average",
		Inputs:         make(map[string][]byte),
	}

	resp, err := service.ExecuteMPCComputation(ctx, req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestPrivacyComputingServiceMPCShareDistribution(t *testing.T) {
	service := NewPrivacyComputingService()
	ctx := context.Background()

	parties := []string{"party1", "party2", "party3"}
	err := service.MPCSetupProtocol(MPCProtocolGMW, parties)
	require.NoError(t, err)

	secret := []byte("secret-for-distribution")
	dist, err := service.DistributeMPCShares(ctx, "party1", secret, 2, 3)
	require.NoError(t, err)
	assert.NotNil(t, dist)
	assert.Equal(t, "party1", dist.PartyID)
}

func TestPrivacyComputingServiceVerifyPrivacyCompliance(t *testing.T) {
	service := NewPrivacyComputingService()
	ctx := context.Background()

	req := &PrivacyVerificationRequest{
		DataID:  "data-1",
		ZKProof: []byte("zk-proof"),
		Purpose: "research",
	}

	resp, err := service.VerifyPrivacyCompliance(ctx, req)
	require.NoError(t, err)
	assert.True(t, resp.Valid)
}

func TestPrivacyComputingServiceVerifyPrivacyComplianceMissingProof(t *testing.T) {
	service := NewPrivacyComputingService()
	ctx := context.Background()

	req := &PrivacyVerificationRequest{
		DataID:  "data-1",
		ZKProof: nil,
		Purpose: "research",
	}

	resp, err := service.VerifyPrivacyCompliance(ctx, req)
	require.NoError(t, err)
	assert.False(t, resp.Valid)
}

func TestPrivacyComputingServiceExportEncryptedData(t *testing.T) {
	service := NewPrivacyComputingService()
	ctx := context.Background()

	he := NewHEEngine()
	pk, _, _ := he.GenerateKeyPair()
	ct, _ := he.Encrypt(big.NewInt(789), pk)

	err := service.PDSRegisterData("data-export-1", "sensitive", "owner-1", ct)
	require.NoError(t, err)

	exported, err := service.ExportEncryptedData(ctx, "data-export-1", "json")
	require.NoError(t, err)
	assert.NotEmpty(t, exported)
}

func TestPrivacyComputingServiceGetMPCPartyCount(t *testing.T) {
	service := NewPrivacyComputingService()

	parties := []string{"party1", "party2", "party3"}
	err := service.MPCSetupProtocol(MPCProtocolGMW, parties)
	require.NoError(t, err)

	count := service.GetMPCPartyCount()
	assert.Equal(t, 3, count)
}

func TestPrivacyComputingServiceGetFLNodeCount(t *testing.T) {
	service := NewPrivacyComputingService()

	node1 := &FLNode{NodeID: "node-1", DatasetSize: 100}
	node2 := &FLNode{NodeID: "node-2", DatasetSize: 200}

	service.FLRegisterNode(node1)
	service.FLRegisterNode(node2)

	count := service.GetFLNodeCount()
	assert.Equal(t, 2, count)
}

func TestPrivacyComputingServiceGetPDSDataCount(t *testing.T) {
	service := NewPrivacyComputingService()

	he := NewHEEngine()
	pk, _, _ := he.GenerateKeyPair()
	ct, _ := he.Encrypt(big.NewInt(123), pk)

	service.PDSRegisterData("data-1", "type1", "owner1", ct)
	service.PDSRegisterData("data-2", "type2", "owner2", ct)

	count := service.GetPDSDataCount()
	assert.Equal(t, 2, count)
}

func TestPrivacyComputingServiceFLModels(t *testing.T) {
	service := NewPrivacyComputingService()

	weights := []float64{0.1, 0.2, 0.3}
	_, err := service.FLInitializeModel("model-1", weights)
	require.NoError(t, err)

	_, err = service.FLInitializeModel("model-2", weights)
	require.NoError(t, err)

	models := service.GetFLModels()
	assert.Len(t, models, 2)
}

func TestPrivacyComputingServiceGetFLNodes(t *testing.T) {
	service := NewPrivacyComputingService()

	node := &FLNode{NodeID: "node-1", DatasetSize: 100, Status: "active"}
	service.FLRegisterNode(node)

	nodes := service.GetFLNodes()
	assert.Len(t, nodes, 1)
}

func TestPrivacyComputingServiceGetRegisteredData(t *testing.T) {
	service := NewPrivacyComputingService()

	he := NewHEEngine()
	pk, _, _ := he.GenerateKeyPair()
	ct, _ := he.Encrypt(big.NewInt(123), pk)

	service.PDSRegisterData("data-1", "type1", "owner1", ct)

	data := service.GetRegisteredData()
	assert.Len(t, data, 1)
}

func TestFLVerifyUpdate(t *testing.T) {
	fl := NewFLEngine()

	node := &FLNode{
		NodeID:      "node-1",
		DatasetSize: 1000,
		Status:      "active",
		Reputation:  1.0,
	}
	fl.RegisterNode(node)

	validWeights := []float64{1.0, 2.0, 3.0}
	valid, err := fl.VerifyUpdate("node-1", validWeights)
	require.NoError(t, err)
	assert.True(t, valid)

	invalidWeights := []float64{10.0, 20.0, 30.0}
	invalid, err := fl.VerifyUpdate("node-1", invalidWeights)
	require.NoError(t, err)
	assert.False(t, invalid)
}

func TestFLVerifyUpdateInvalidNode(t *testing.T) {
	fl := NewFLEngine()

	_, err := fl.VerifyUpdate("nonexistent-node", []float64{1.0, 2.0})
	assert.Error(t, err)
}

func TestMPCEvaluateGarbledCircuit(t *testing.T) {
	mpc := NewMPCEngine()

	input := []byte("test-input")
	output, err := mpc.EvaluateGarbledCircuit(input)
	require.NoError(t, err)
	assert.NotNil(t, output)
}

func TestGenerateRandomBytes(t *testing.T) {
	bytes, err := GenerateRandomBytes(32)
	require.NoError(t, err)
	assert.Len(t, bytes, 32)
}

func TestEncodeDecodeBase64(t *testing.T) {
	original := []byte("test-data-for-encoding")
	encoded := EncodeToBase64(original)
	assert.NotEmpty(t, encoded)

	decoded, err := DecodeFromBase64(encoded)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}
