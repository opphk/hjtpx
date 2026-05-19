package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlockchainService(t *testing.T) {
	svc := NewBlockchainService()
	assert.NotNil(t, svc)
}

func TestRecordVerification(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	record := &VerificationRecord{
		AppID:      "test-app",
		SessionID:  "session-123",
		EventType:  "login_attempt",
		EventData:  `{"user": "test@example.com"}`,
		RiskScore:  0.3,
		UserAgent:  "Mozilla/5.0",
		IPAddress:  "192.168.1.1",
		DeviceFinger: "fp-abc123",
	}

	proof, err := svc.RecordVerification(ctx, record)

	require.NoError(t, err)
	assert.NotNil(t, proof)
	assert.NotEmpty(t, proof.ProofID)
	assert.NotEmpty(t, proof.TxHash)
	assert.NotEmpty(t, proof.BlockHash)
	assert.Equal(t, "ethereum", proof.ChainID)
	assert.Equal(t, "confirmed", proof.Status)
	assert.Equal(t, record.RecordID, proof.RecordID)
}

func TestRecordVerification_NilRecord(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	proof, err := svc.RecordVerification(ctx, nil)

	assert.Error(t, err)
	assert.Nil(t, proof)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestVerifyProof(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	record := &VerificationRecord{
		AppID:      "test-app",
		SessionID:  "session-456",
		EventType:  "login_success",
		RiskScore:  0.1,
	}

	proof, err := svc.RecordVerification(ctx, record)
	require.NoError(t, err)

	verified, err := svc.VerifyProof(ctx, proof.ProofID)

	require.NoError(t, err)
	assert.Equal(t, "verified", verified.Status)
}

func TestVerifyProof_NotFound(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	proof, err := svc.VerifyProof(ctx, "non-existent-proof")

	assert.Error(t, err)
	assert.Nil(t, proof)
	assert.Equal(t, ErrProofNotFound, err)
}

func TestGetVerificationHistory(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		record := &VerificationRecord{
			AppID:     "test-app",
			SessionID: "session-" + string(rune('0'+i)),
			EventType: "login_attempt",
			RiskScore: 0.2,
		}
		_, err := svc.RecordVerification(ctx, record)
		require.NoError(t, err)
	}

	records, err := svc.GetVerificationHistory(ctx, "test-app", 10, 0)

	require.NoError(t, err)
	assert.Len(t, records, 5)
}

func TestGetVerificationHistory_WithPagination(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		record := &VerificationRecord{
			AppID:     "test-app-paginate",
			SessionID: "session-" + string(rune('0'+i)),
			EventType: "login_attempt",
			RiskScore: 0.2,
		}
		_, err := svc.RecordVerification(ctx, record)
		require.NoError(t, err)
	}

	page1, err := svc.GetVerificationHistory(ctx, "test-app-paginate", 3, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 3)

	page2, err := svc.GetVerificationHistory(ctx, "test-app-paginate", 3, 3)
	require.NoError(t, err)
	assert.Len(t, page2, 3)

	page3, err := svc.GetVerificationHistory(ctx, "test-app-paginate", 3, 6)
	require.NoError(t, err)
	assert.Len(t, page3, 3)

	page4, err := svc.GetVerificationHistory(ctx, "test-app-paginate", 3, 9)
	require.NoError(t, err)
	assert.Len(t, page4, 1)
}

func TestRegisterCrossChainIdentity(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	identity := &CrossChainIdentity{
		Identity:   "0x1234567890abcdef",
		ChainType:  "ethereum",
		PublicKey:  "0xpubkey123",
		LinkedIDs:  []string{"twitter:user1", "github:user1"},
		Verified:   true,
	}

	err := svc.RegisterCrossChainIdentity(ctx, identity)

	require.NoError(t, err)
	assert.NotEmpty(t, identity.IdentityID)
	assert.Greater(t, identity.TrustScore, 0.0)
}

func TestRegisterCrossChainIdentity_Duplicate(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	identity := &CrossChainIdentity{
		Identity:  "0xduplicate",
		ChainType: "ethereum",
	}

	err := svc.RegisterCrossChainIdentity(ctx, identity)
	require.NoError(t, err)

	err = svc.RegisterCrossChainIdentity(ctx, identity)
	assert.Error(t, err)
	assert.Equal(t, ErrIdentityExists, err)
}

func TestGetCrossChainIdentity(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	identity := &CrossChainIdentity{
		Identity:   "0xidentity123",
		ChainType:  "polygon",
		PublicKey:  "0xpubkey456",
	}

	err := svc.RegisterCrossChainIdentity(ctx, identity)
	require.NoError(t, err)

	retrieved, err := svc.GetCrossChainIdentity(ctx, identity.Identity)

	require.NoError(t, err)
	assert.Equal(t, identity.IdentityID, retrieved.IdentityID)
	assert.Equal(t, identity.Identity, retrieved.Identity)
}

func TestGetCrossChainIdentity_NotFound(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	identity, err := svc.GetCrossChainIdentity(ctx, "non-existent")

	assert.Error(t, err)
	assert.Nil(t, identity)
}

func TestVerifyCrossChain(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	identity := &CrossChainIdentity{
		Identity:  "0xcrosschain123",
		ChainType: "ethereum",
		Verified:  true,
	}

	err := svc.RegisterCrossChainIdentity(ctx, identity)
	require.NoError(t, err)

	request := &CrossChainVerifyRequest{
		SourceChain: "ethereum",
		TargetChain: "polygon",
		Identity:    identity.Identity,
	}

	response, err := svc.VerifyCrossChain(ctx, request)

	require.NoError(t, err)
	assert.True(t, response.Valid)
	assert.Equal(t, identity.Identity, response.Identity)
}

func TestVerifyCrossChain_ChainMismatch(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	identity := &CrossChainIdentity{
		Identity:  "0xmismatch123",
		ChainType: "polygon",
		Verified:  true,
	}

	err := svc.RegisterCrossChainIdentity(ctx, identity)
	require.NoError(t, err)

	request := &CrossChainVerifyRequest{
		SourceChain: "ethereum",
		TargetChain: "polygon",
		Identity:    identity.Identity,
	}

	response, err := svc.VerifyCrossChain(ctx, request)

	require.NoError(t, err)
	assert.False(t, response.Valid)
	assert.Contains(t, response.Message, "mismatch")
}

func TestVerifyCrossChain_IdentityNotFound(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	request := &CrossChainVerifyRequest{
		SourceChain: "ethereum",
		TargetChain: "polygon",
		Identity:    "0xnotfound",
	}

	response, err := svc.VerifyCrossChain(ctx, request)

	require.NoError(t, err)
	assert.False(t, response.Valid)
	assert.Equal(t, "identity not found", response.Message)
}

func TestVerifyCrossChain_MissingChains(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	request := &CrossChainVerifyRequest{
		SourceChain: "",
		TargetChain: "polygon",
		Identity:    "0x123",
	}

	response, err := svc.VerifyCrossChain(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestCreateAuditLog(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	log := &AuditLog{
		AppID:     "test-app",
		UserID:    "user-123",
		Action:    "login",
		Resource:  "/api/login",
		Details:   "User logged in successfully",
		IPAddress: "192.168.1.100",
		UserAgent: "Mozilla/5.0",
		Result:    "success",
	}

	err := svc.CreateAuditLog(ctx, log)

	require.NoError(t, err)
	assert.NotEmpty(t, log.LogID)
	assert.NotEmpty(t, log.Hash)
}

func TestGetAuditLogs(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		log := &AuditLog{
			AppID:  "test-app-audit",
			UserID: "user-" + string(rune('0'+i)),
			Action: "login",
		}
		err := svc.CreateAuditLog(ctx, log)
		require.NoError(t, err)
	}

	logs, err := svc.GetAuditLogs(ctx, "test-app-audit", time.Time{}, time.Now())

	require.NoError(t, err)
	assert.Len(t, logs, 3)
}

func TestVerifyAuditLog(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	log := &AuditLog{
		AppID:    "test-app-verify",
		UserID:   "user-verify",
		Action:   "login",
		Resource: "/api/login",
		Details:  "Test details",
	}

	err := svc.CreateAuditLog(ctx, log)
	require.NoError(t, err)

	valid, err := svc.VerifyAuditLog(ctx, log.LogID)

	require.NoError(t, err)
	assert.True(t, valid)
}

func TestVerifyAuditLog_Tampered(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	log := &AuditLog{
		AppID:    "test-app-tamper",
		UserID:   "user-tamper",
		Action:   "login",
		Resource: "/api/login",
		Details:  "Original details",
	}

	err := svc.CreateAuditLog(ctx, log)
	require.NoError(t, err)

	log.Details = "Tampered details"
	log.Hash = ""

	valid, err := svc.VerifyAuditLog(ctx, log.LogID)

	require.NoError(t, err)
	assert.False(t, valid)
}

func TestVerifyAuditLog_NotFound(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	valid, err := svc.VerifyAuditLog(ctx, "non-existent-log")

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestExportAuditTrail(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		log := &AuditLog{
			AppID:  "test-app-export",
			UserID: "user-export-" + string(rune('0'+i)),
			Action: "export_test",
		}
		err := svc.CreateAuditLog(ctx, log)
		require.NoError(t, err)
	}

	data, err := svc.ExportAuditTrail(ctx, "test-app-export")

	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), "export_test")
}

func TestRecordHash_Uniqueness(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	record1 := &VerificationRecord{
		AppID:      "hash-test",
		SessionID:  "session-1",
		EventType:  "login",
		RiskScore:  0.5,
		IPAddress:  "10.0.0.1",
	}

	record2 := &VerificationRecord{
		AppID:      "hash-test",
		SessionID:  "session-2",
		EventType:  "login",
		RiskScore:  0.5,
		IPAddress:  "10.0.0.2",
	}

	proof1, err := svc.RecordVerification(ctx, record1)
	require.NoError(t, err)

	proof2, err := svc.RecordVerification(ctx, record2)
	require.NoError(t, err)

	assert.NotEqual(t, proof1.TxHash, proof2.TxHash)
	assert.NotEqual(t, proof1.BlockHash, proof2.BlockHash)
}

func TestTrustScoreCalculation(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	identity := &CrossChainIdentity{
		Identity:   "0xtrusttest",
		ChainType:  "ethereum",
		PublicKey:  "0xpub",
		LinkedIDs:  []string{"id1", "id2", "id3"},
		Verified:   true,
	}

	err := svc.RegisterCrossChainIdentity(ctx, identity)
	require.NoError(t, err)

	baseScore := 50.0
	publicKeyBonus := 20.0
	linkedBonus := 15.0

	expectedMin := baseScore + publicKeyBonus + linkedBonus
	assert.GreaterOrEqual(t, identity.TrustScore, expectedMin)
}

func TestChainState_Updates(t *testing.T) {
	svc := NewBlockchainService()
	ctx := context.Background()

	record := &VerificationRecord{
		AppID:     "state-test",
		SessionID: "session-state",
		EventType: "login",
		RiskScore: 0.3,
	}

	_, err := svc.RecordVerification(ctx, record)
	require.NoError(t, err)

	proof, err := svc.RecordVerification(ctx, record)
	require.NoError(t, err)

	assert.Greater(t, proof.BlockNumber, uint64(0))
}

func TestMerkleTree_Rebuild(t *testing.T) {
	mt := newMerkleTree()

	mt.addLeaf("leaf1")
	assert.Len(t, mt.Leaves, 1)
	assert.NotEmpty(t, mt.Root)

	mt.addLeaf("leaf2")
	assert.Len(t, mt.Leaves, 2)
	assert.NotEmpty(t, mt.Root)

	mt.addLeaf("leaf3")
	assert.Len(t, mt.Leaves, 3)
}

func TestNilContext(t *testing.T) {
	svc := NewBlockchainService()

	record := &VerificationRecord{
		AppID:     "nil-ctx",
		SessionID: "session",
		EventType: "test",
		RiskScore: 0.5,
	}

	proof, err := svc.RecordVerification(nil, record)

	require.NoError(t, err)
	assert.NotNil(t, proof)
}
