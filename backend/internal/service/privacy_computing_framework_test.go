package service

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrivacyComputingFramework(t *testing.T) {
	framework := NewPrivacyComputingService()
	assert.NotNil(t, framework)
	assert.NotNil(t, framework.heEngine)
	assert.NotNil(t, framework.mpcEngine)
	assert.NotNil(t, framework.flEngine)
	assert.NotNil(t, framework.pdsEngine)
}

func TestHEContextCreation(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	algorithms := []HEAlgorithm{HEAlgorithmPaillier, HEAlgorithmBGV, HEAlgorithmCKKS, HEAlgorithmBFV}

	for _, algo := range algorithms {
		err := framework.CreateHEContext(ctx, "test-context-"+string(algo), algo)
		require.NoError(t, err)
	}

	info, err := framework.GetHEContextInfo(ctx, "test-context-"+string(HEAlgorithmPaillier))
	require.NoError(t, err)
	assert.Equal(t, HEAlgorithmPaillier, info.Algorithm)
}

func TestHEKeyGeneration(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	err := framework.CreateHEContext(ctx, "test-keygen", HEAlgorithmPaillier)
	require.NoError(t, err)

	pk, sk, err := framework.GenerateHEKeyPair(ctx, "test-keygen")
	require.NoError(t, err)
	assert.NotNil(t, pk)
	assert.NotNil(t, sk)
}

func TestHEEncryptionDecryption(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	err := framework.CreateHEContext(ctx, "test-enc", HEAlgorithmPaillier)
	require.NoError(t, err)

	pk, sk, err := framework.GenerateHEKeyPair(ctx, "test-enc")
	require.NoError(t, err)

	plaintext := big.NewInt(42)
	ct, err := framework.HEEncrypt(ctx, "test-enc", plaintext, pk)
	require.NoError(t, err)
	assert.NotNil(t, ct)

	decrypted, err := framework.HEDecrypt(ctx, "test-enc", ct, sk)
	require.NoError(t, err)
	assert.Equal(t, plaintext.Int64(), decrypted.Int64())
}

func TestHEAddition(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	err := framework.CreateHEContext(ctx, "test-add", HEAlgorithmPaillier)
	require.NoError(t, err)

	pk, sk, err := framework.GenerateHEKeyPair(ctx, "test-add")
	require.NoError(t, err)

	m1 := big.NewInt(10)
	m2 := big.NewInt(20)

	ct1, err := framework.HEEncrypt(ctx, "test-add", m1, pk)
	require.NoError(t, err)

	ct2, err := framework.HEEncrypt(ctx, "test-add", m2, pk)
	require.NoError(t, err)

	result, err := framework.HEAdd(ctx, "test-add", ct1, ct2)
	require.NoError(t, err)

	decrypted, err := framework.HEDecrypt(ctx, "test-add", result, sk)
	require.NoError(t, err)
	assert.Equal(t, int64(30), decrypted.Int64())
}

func TestHESubtraction(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	err := framework.CreateHEContext(ctx, "test-sub", HEAlgorithmPaillier)
	require.NoError(t, err)

	pk, sk, err := framework.GenerateHEKeyPair(ctx, "test-sub")
	require.NoError(t, err)

	m1 := big.NewInt(50)
	m2 := big.NewInt(30)

	ct1, err := framework.HEEncrypt(ctx, "test-sub", m1, pk)
	require.NoError(t, err)

	ct2, err := framework.HEEncrypt(ctx, "test-sub", m2, pk)
	require.NoError(t, err)

	result, err := framework.HESubtract(ctx, "test-sub", ct1, ct2)
	require.NoError(t, err)

	decrypted, err := framework.HEDecrypt(ctx, "test-sub", result, sk)
	require.NoError(t, err)
	assert.Equal(t, int64(20), decrypted.Int64())
}

func TestHEScalarMultiply(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	err := framework.CreateHEContext(ctx, "test-scalar", HEAlgorithmPaillier)
	require.NoError(t, err)

	pk, _, err := framework.GenerateHEKeyPair(ctx, "test-scalar")
	require.NoError(t, err)

	m := big.NewInt(7)
	ct, err := framework.HEEncrypt(ctx, "test-scalar", m, pk)
	require.NoError(t, err)

	scalar := big.NewInt(3)
	result, err := framework.HEScalarMultiply(ctx, "test-scalar", ct, scalar)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMPCProtocolSetup(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	protocols := []MPCProtocol{MPCProtocolGMW, MPCProtocolBGW, MPCProtocolYao, MPCProtocolSPDZ, MPCProtocolABY}

	for _, protocol := range protocols {
		session, err := framework.CreateMPCSession(ctx, protocol, []string{"party1", "party2", "party3"})
		require.NoError(t, err)
		assert.NotNil(t, session)
		assert.Equal(t, protocol, session.Protocol)
	}
}

func TestMPCSecretSharing(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	session, err := framework.CreateMPCSession(ctx, MPCProtocolGMW, []string{"party1", "party2", "party3"})
	require.NoError(t, err)

	secret := []byte("test-secret-data-12345")
	dist, err := framework.MPCShareSecret(ctx, session.SessionID, "party1", secret, 2)
	require.NoError(t, err)
	assert.NotNil(t, dist)
	assert.Equal(t, 3, dist.TotalParties)
	assert.Equal(t, 2, dist.Threshold)
}

func TestMPCSecretReconstruction(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	session, err := framework.CreateMPCSession(ctx, MPCProtocolGMW, []string{"party1", "party2", "party3"})
	require.NoError(t, err)

	secret := []byte("test-secret-for-reconstruction")
	dist, err := framework.MPCShareSecret(ctx, session.SessionID, "party1", secret, 2)
	require.NoError(t, err)

	shares := make([]*MPCShare, len(dist.Shares))
	for i, share := range dist.Shares {
		shares[i] = share
	}

	result, err := framework.MPCReconstructSecret(ctx, session.SessionID, shares, 2)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMPCComputation(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	session, err := framework.CreateMPCSession(ctx, MPCProtocolGMW, []string{"party1", "party2"})
	require.NoError(t, err)

	inputs := map[string][]byte{
		"party1": []byte("input1"),
		"party2": []byte("input2"),
	}

	result, err := framework.ExecuteMPCComputationV2(ctx, &MPCComputationRequestV2{
		SessionID:       session.SessionID,
		ComputationType: "sum",
		Inputs:          inputs,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestFLNodeRegistration(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	node := &FLNode{
		NodeID:      "node-1",
		DatasetSize: 1000,
		ModelVersion: "1.0.0",
		Status:      "active",
		Reputation:  1.0,
	}

	err := framework.FLRegisterNode(ctx, node)
	require.NoError(t, err)
}

func TestFLModelInitialization(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	initialWeights := []float64{0.1, 0.2, 0.3, 0.4}
	model, err := framework.FLInitializeModel(ctx, "model-1", initialWeights)
	require.NoError(t, err)
	assert.NotNil(t, model)
	assert.Equal(t, "model-1", model.ModelID)
	assert.Len(t, model.Weights, 4)
}

func TestFLAggregation(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	initialWeights := []float64{0.1, 0.2, 0.3, 0.4}
	_, err := framework.FLInitializeModel(ctx, "model-2", initialWeights)
	require.NoError(t, err)

	request := &FLAggregationRequest{
		NodeID:      "node-1",
		ModelID:     "model-2",
		Weights:     []float64{0.05, 0.05, 0.05, 0.05},
		Gradient:    []float64{0.01, 0.01, 0.01, 0.01},
		DatasetSize: 100,
	}

	result, err := framework.FLAggregateUpdates(ctx, request)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestFLDifferentialPrivacy(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	gradients := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	noised := framework.FLApplyDifferentialPrivacy(ctx, gradients, 1.0, 1e-5)
	assert.Len(t, noised, 5)

	for i := range gradients {
		assert.NotEqual(t, gradients[i], noised[i])
	}
}

func TestPDSDataRegistration(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	err := framework.CreateHEContext(ctx, "test-pds", HEAlgorithmPaillier)
	require.NoError(t, err)

	pk, _, err := framework.GenerateHEKeyPair(ctx, "test-pds")
	require.NoError(t, err)

	plaintext := big.NewInt(123)
	ct, err := framework.HEEncrypt(ctx, "test-pds", plaintext, pk)
	require.NoError(t, err)

	err = framework.PDSRegisterData(ctx, "data-1", "sensitive", "owner-1", ct)
	require.NoError(t, err)
}

func TestPDSAccessPolicy(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	err := framework.CreateHEContext(ctx, "test-policy", HEAlgorithmPaillier)
	require.NoError(t, err)

	pk, _, err := framework.GenerateHEKeyPair(ctx, "test-policy")
	require.NoError(t, err)

	plaintext := big.NewInt(456)
	ct, err := framework.HEEncrypt(ctx, "test-policy", plaintext, pk)
	require.NoError(t, err)

	err = framework.PDSRegisterData(ctx, "data-policy-1", "sensitive", "owner-1", ct)
	require.NoError(t, err)

	policy := &DataAccessPolicy{
		RequiredZKProof: true,
		MinTrustLevel:   5,
		AllowedPurposes: []string{"research", "analytics"},
	}

	err = framework.PDSSetAccessPolicy(ctx, "data-policy-1", policy)
	require.NoError(t, err)
}

func TestPDSAccessRequest(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	err := framework.CreateHEContext(ctx, "test-access", HEAlgorithmPaillier)
	require.NoError(t, err)

	pk, _, err := framework.GenerateHEKeyPair(ctx, "test-access")
	require.NoError(t, err)

	plaintext := big.NewInt(789)
	ct, err := framework.HEEncrypt(ctx, "test-access", plaintext, pk)
	require.NoError(t, err)

	err = framework.PDSRegisterData(ctx, "data-access-1", "sensitive", "owner-1", ct)
	require.NoError(t, err)

	request := &DataAccessRequest{
		DataID:      "data-access-1",
		RequesterID: "owner-1",
		Purpose:     "management",
	}

	resp, err := framework.PDSRequestAccess(ctx, request)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestPrivacyBudget(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	budget := &FLPrivacyBudget{
		Epsilon:     0.5,
		Delta:       1e-6,
		NoiseScale:  0.5,
		MaxGradient: 3.0,
	}

	framework.SetPrivacyBudget(ctx, "user-1", budget)

	retrieved := framework.GetPrivacyBudget(ctx, "user-1")
	assert.NotNil(t, retrieved)
	assert.Equal(t, 0.5, retrieved.Epsilon)
}

func TestPrivacyCheck(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	budget := &FLPrivacyBudget{
		Epsilon:     1.0,
		Delta:       1e-5,
		NoiseScale:  1.0,
		MaxGradient: 5.0,
	}

	validData := []float64{1.0, 2.0, 3.0}
	valid := framework.PerformPrivacyCheck(ctx, validData, budget)
	assert.True(t, valid)

	invalidData := []float64{10.0, 20.0, 30.0}
	invalid := framework.PerformPrivacyCheck(ctx, invalidData, budget)
	assert.False(t, invalid)
}

func TestHEBatchOperations(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	err := framework.CreateHEContext(ctx, "test-batch", HEAlgorithmPaillier)
	require.NoError(t, err)

	pk, sk, err := framework.GenerateHEKeyPair(ctx, "test-batch")
	require.NoError(t, err)

	ciphertexts := make([]*HECiphertext, 5)
	for i := 0; i < 5; i++ {
		plaintext := big.NewInt(int64(i + 1))
		ct, err := framework.HEEncrypt(ctx, "test-batch", plaintext, pk)
		require.NoError(t, err)
		ciphertexts[i] = ct
	}

	batchResult, err := framework.HEBatchAdd(ctx, "test-batch", ciphertexts)
	require.NoError(t, err)
	assert.True(t, batchResult.Success)

	decrypted, err := framework.HEDecrypt(ctx, "test-batch", batchResult.Results[0], sk)
	require.NoError(t, err)
	assert.Equal(t, int64(15), decrypted.Int64())
}

func TestMPCMultipleParties(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	session, err := framework.CreateMPCSession(ctx, MPCProtocolGMW, []string{"party1", "party2", "party3", "party4", "party5"})
	require.NoError(t, err)

	secret := []byte("multi-party-secret-key")
	dist, err := framework.MPCShareSecret(ctx, session.SessionID, "party1", secret, 3)
	require.NoError(t, err)
	assert.Equal(t, 5, dist.TotalParties)
	assert.Equal(t, 3, dist.Threshold)
}

func TestFLMultipleNodes(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		node := &FLNode{
			NodeID:      "node-" + string(rune('a'+i)),
			DatasetSize: 1000 + i*100,
			Status:      "active",
			Reputation:  1.0 - float64(i)*0.1,
		}
		err := framework.FLRegisterNode(ctx, node)
		require.NoError(t, err)
	}

	nodeCount := framework.GetFLNodeCount(ctx)
	assert.Equal(t, 5, nodeCount)
}

func TestPrivacyVerification(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	req := &PrivacyVerificationRequest{
		DataID:  "data-verification-1",
		ZKProof: []byte("zk-proof-data"),
		Purpose: "research",
	}

	resp, err := framework.VerifyPrivacyCompliance(ctx, req)
	require.NoError(t, err)
	assert.True(t, resp.Valid)
}

func TestPrivacyVerificationMissingProof(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	req := &PrivacyVerificationRequest{
		DataID:  "data-verification-2",
		ZKProof: nil,
		Purpose: "research",
	}

	resp, err := framework.VerifyPrivacyCompliance(ctx, req)
	require.NoError(t, err)
	assert.False(t, resp.Valid)
}

func TestGetMPCPartyCount(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	session, err := framework.CreateMPCSession(ctx, MPCProtocolGMW, []string{"party1", "party2", "party3"})
	require.NoError(t, err)

	count := framework.GetMPCPartyCount(ctx)
	assert.Equal(t, 3, count)

	_ = session
}

func TestGetFLNodeCount(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	node1 := &FLNode{NodeID: "node-count-1", DatasetSize: 100}
	node2 := &FLNode{NodeID: "node-count-2", DatasetSize: 200}

	framework.FLRegisterNode(ctx, node1)
	framework.FLRegisterNode(ctx, node2)

	count := framework.GetFLNodeCount(ctx)
	assert.Equal(t, 2, count)
}

func TestGetPDSDataCount(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	err := framework.CreateHEContext(ctx, "test-count", HEAlgorithmPaillier)
	require.NoError(t, err)

	pk, _, err := framework.GenerateHEKeyPair(ctx, "test-count")
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		plaintext := big.NewInt(int64(i))
		ct, err := framework.HEEncrypt(ctx, "test-count", plaintext, pk)
		require.NoError(t, err)

		err = framework.PDSRegisterData(ctx, "data-count-"+string(rune('0'+i)), "type", "owner", ct)
		require.NoError(t, err)
	}

	count := framework.GetPDSDataCount(ctx)
	assert.Equal(t, 3, count)
}

func TestExportEncryptedData(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	err := framework.CreateHEContext(ctx, "test-export", HEAlgorithmPaillier)
	require.NoError(t, err)

	pk, _, err := framework.GenerateHEKeyPair(ctx, "test-export")
	require.NoError(t, err)

	plaintext := big.NewInt(999)
	ct, err := framework.HEEncrypt(ctx, "test-export", plaintext, pk)
	require.NoError(t, err)

	err = framework.PDSRegisterData(ctx, "data-export-1", "sensitive", "owner-1", ct)
	require.NoError(t, err)

	exported, err := framework.ExportEncryptedData(ctx, "data-export-1", "json")
	require.NoError(t, err)
	assert.NotEmpty(t, exported)
}

func TestMPCDotProduct(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	err := framework.CreateHEContext(ctx, "test-dot", HEAlgorithmPaillier)
	require.NoError(t, err)

	pk, _, err := framework.GenerateHEKeyPair(ctx, "test-dot")
	require.NoError(t, err)

	vec1 := []*HECiphertext{}
	vec2 := []*HECiphertext{}

	for i := 0; i < 4; i++ {
		plaintext := big.NewInt(int64(i + 1))
		ct1, err := framework.HEEncrypt(ctx, "test-dot", plaintext, pk)
		require.NoError(t, err)
		vec1 = append(vec1, ct1)

		ct2, err := framework.HEEncrypt(ctx, "test-dot", plaintext, pk)
		require.NoError(t, err)
		vec2 = append(vec2, ct2)
	}

	dotProduct, err := framework.HEComputeDotProduct(ctx, "test-dot", vec1, vec2)
	require.NoError(t, err)
	assert.NotNil(t, dotProduct)
}

func TestMPCGarbledCircuit(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	session, err := framework.CreateMPCSession(ctx, MPCProtocolYao, []string{"party1", "party2"})
	require.NoError(t, err)

	garbledTables := map[int][]byte{
		0: []byte("garbled-table-0"),
		1: []byte("garbled-table-1"),
	}

	inputs := map[int]byte{
		0: 1,
		1: 0,
	}

	output, err := framework.EvaluateGarbledCircuit(ctx, session.SessionID, garbledTables, inputs)
	require.NoError(t, err)
	assert.NotNil(t, output)
}

func TestMPCSessionManagement(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	session, err := framework.CreateMPCSession(ctx, MPCProtocolGMW, []string{"party1", "party2"})
	require.NoError(t, err)

	retrievedSession, err := framework.GetMPCSession(ctx, session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, session.SessionID, retrievedSession.SessionID)

	err = framework.TerminateMPCSession(ctx, session.SessionID)
	require.NoError(t, err)
}

func TestFLModelAggregation(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	initialWeights := []float64{0.1, 0.2, 0.3}
	_, err := framework.FLInitializeModel(ctx, "model-agg", initialWeights)
	require.NoError(t, err)

	models := framework.GetFLModels(ctx)
	assert.Len(t, models, 1)
}

func TestGetAvailableMPCProtocols(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	protocols := framework.GetAllMPCProtocols(ctx)
	assert.NotEmpty(t, protocols)
	assert.GreaterOrEqual(t, len(protocols), 3)
}

func TestHEMultiplication(t *testing.T) {
	framework := NewPrivacyComputingService()
	ctx := context.Background()

	err := framework.CreateHEContext(ctx, "test-mul", HEAlgorithmPaillier)
	require.NoError(t, err)

	pk, sk, err := framework.GenerateHEKeyPair(ctx, "test-mul")
	require.NoError(t, err)

	m1 := big.NewInt(6)
	m2 := big.NewInt(7)

	ct1, err := framework.HEEncrypt(ctx, "test-mul", m1, pk)
	require.NoError(t, err)

	ct2, err := framework.HEEncrypt(ctx, "test-mul", m2, pk)
	require.NoError(t, err)

	result, err := framework.HEMultiply(ctx, "test-mul", ct1, ct2)
	require.NoError(t, err)
	assert.NotNil(t, result)

	_ = sk
}
