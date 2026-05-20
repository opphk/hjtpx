package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDIDVerificationService(t *testing.T) {
	service := NewDIDVerificationService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.didRegistry)
	assert.NotNil(t, service.vcService)
	assert.NotNil(t, service.zkService)
	assert.NotNil(t, service.bridge)
	assert.NotNil(t, service.auditLog)
}

func TestCreateDID(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	publicKey := []byte("test-public-key-1234567890")
	services := []ServiceEndpoint{}

	doc, err := service.CreateDID(ctx, DIDMethodWeb, "example.com:user1", publicKey, services)
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Contains(t, doc.ID, "did:web")
	assert.Equal(t, "did:web:example.com:user1", doc.ID)
	assert.Len(t, doc.VerificationMethod, 1)
	assert.False(t, doc.Created.IsZero())
}

func TestCreateDIDAlreadyExists(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	publicKey := []byte("test-public-key")
	services := []ServiceEndpoint{}

	doc1, err := service.CreateDID(ctx, DIDMethodWeb, "example.com:user2", publicKey, services)
	require.NoError(t, err)

	doc2, err := service.CreateDID(ctx, DIDMethodWeb, "example.com:user2", publicKey, services)
	assert.Error(t, err)
	assert.Nil(t, doc2)
}

func TestResolveDID(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	publicKey := []byte("test-public-key-123")
	services := []ServiceEndpoint{}

	created, err := service.CreateDID(ctx, DIDMethodWeb, "example.com:user3", publicKey, services)
	require.NoError(t, err)

	resolved, err := service.ResolveDID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, resolved.ID)
	assert.Equal(t, created.Context, resolved.Context)
}

func TestResolveDIDNotFound(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	_, err := service.ResolveDID(ctx, "did:web:nonexistent")
	assert.Error(t, err)
}

func TestUpdateDID(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	publicKey := []byte("test-public-key")
	services := []ServiceEndpoint{}

	created, err := service.CreateDID(ctx, DIDMethodWeb, "example.com:user4", publicKey, services)
	require.NoError(t, err)

	newServices := []ServiceEndpoint{
		{
			ID:              created.ID + "#service-1",
			Type:            "LinkedDomains",
			ServiceEndpoint: "https://example.com",
		},
	}

	updates := &DIDDocument{
		Service: newServices,
	}

	err = service.UpdateDID(ctx, created.ID, updates)
	require.NoError(t, err)

	resolved, err := service.ResolveDID(ctx, created.ID)
	require.NoError(t, err)
	assert.Len(t, resolved.Service, 1)
	assert.Equal(t, "https://example.com", resolved.Service[0].ServiceEndpoint)
}

func TestDeactivateDID(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	publicKey := []byte("test-public-key")
	services := []ServiceEndpoint{}

	created, err := service.CreateDID(ctx, DIDMethodWeb, "example.com:user5", publicKey, services)
	require.NoError(t, err)

	err = service.DeactivateDID(ctx, created.ID, "User requested deactivation")
	require.NoError(t, err)

	resolved, err := service.ResolveDID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, resolved.Proof)
}

func TestListDIDs(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	method := DIDMethodWeb

	for i := 0; i < 5; i++ {
		_, err := service.CreateDID(ctx, method, "example.com:user"+string(rune('a'+i)), []byte("key"), []ServiceEndpoint{})
		require.NoError(t, err)
	}

	dids := service.ListDIDs(ctx, method)
	assert.Len(t, dids, 5)
}

func TestVerifyDIDAuthentication(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	publicKey := []byte("test-public-key-12345")
	services := []ServiceEndpoint{}

	created, err := service.CreateDID(ctx, DIDMethodWeb, "example.com:user6", publicKey, services)
	require.NoError(t, err)

	valid, err := service.VerifyDIDAuthentication(ctx, created.ID, "challenge123", []byte("signature"))
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestAddChainConfiguration(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	config := &ChainConfig{
		ChainID:       "ethereum",
		Name:          "Ethereum",
		Protocol:      "ethr",
		RPCEndpoint:   "https://mainnet.infura.io",
		BlockExplorer: "https://etherscan.io",
		Status:        "active",
	}

	err := service.AddChainConfiguration(ctx, config)
	require.NoError(t, err)
}

func TestCreateBridgeConnection(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	bridge, err := service.CreateBridgeConnection(ctx, "ethereum", "polygon")
	require.NoError(t, err)
	assert.NotNil(t, bridge)
	assert.Equal(t, "ethereum", bridge.SourceChain)
	assert.Equal(t, "polygon", bridge.TargetChain)
	assert.Equal(t, "active", bridge.Status)
}

func TestSyncCrossChainDID(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	did := "did:web:example.com:user7"

	err := service.SyncCrossChainDID(ctx, did, "ethereum", "polygon")
	require.Error(t, err)

	_, err = service.CreateBridgeConnection(ctx, "ethereum", "polygon")
	require.NoError(t, err)

	err = service.SyncCrossChainDID(ctx, did, "ethereum", "polygon")
	require.NoError(t, err)
}

func TestGetCrossChainState(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	_, err := service.CreateBridgeConnection(ctx, "ethereum", "polygon")
	require.NoError(t, err)

	did := "did:web:example.com:user8"

	states, err := service.GetCrossChainState(ctx, did)
	require.NoError(t, err)
	assert.NotEmpty(t, states)
}

func TestRegisterCircuit(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	circuit := &ZKProofCircuit{
		CircuitID:   "test-circuit",
		Name:        "Test Circuit",
		Version:     "1.0",
		Constraints: 100,
		ABI:         []byte("circuit-abi"),
	}

	err := service.RegisterCircuit(ctx, circuit)
	require.NoError(t, err)
}

func TestGenerateZKProof(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	publicKey := []byte("test-public-key")
	services := []ServiceEndpoint{}

	created, err := service.CreateDID(ctx, DIDMethodWeb, "example.com:user9", publicKey, services)
	require.NoError(t, err)

	proof, err := service.GenerateZKProof(ctx, created.ID, []string{"name", "age"}, "challenge123")
	require.NoError(t, err)
	assert.NotNil(t, proof)
	assert.NotEmpty(t, proof.ID)
	assert.Contains(t, proof.ID, "urn:zkproof:")
}

func TestVerifyZKProof(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	proof := &ZKProof{
		ID:   "urn:zkproof:test",
		Type: []string{"BbsBlsProof2020"},
		Proof: &ZKProofValue{
			ProofType:         "Groth16",
			CircuitIdentifier: "test-circuit",
		},
		GeneratedAt: time.Now(),
	}

	valid, err := service.VerifyZKProof(ctx, proof)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestVerifyZKProofInvalid(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	proof := &ZKProof{
		ID:   "urn:zkproof:test",
		Type: []string{"BbsBlsProof2020"},
		Proof: nil,
	}

	valid, err := service.VerifyZKProof(ctx, proof)
	assert.False(t, valid)
	assert.Error(t, err)
}

func TestGetAuditLogs(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	_, _ = service.CreateDID(ctx, DIDMethodWeb, "example.com:user10", []byte("key"), []ServiceEndpoint{})

	logs := service.GetAuditLogs(ctx, "did:web:example.com:user10", 10)
	assert.Len(t, logs, 1)
}

func TestExportDIDDocument(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	publicKey := []byte("test-public-key")
	services := []ServiceEndpoint{}

	created, err := service.CreateDID(ctx, DIDMethodWeb, "example.com:user11", publicKey, services)
	require.NoError(t, err)

	jsonData, err := service.ExportDIDDocument(ctx, created.ID, "json")
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	jsonldData, err := service.ExportDIDDocument(ctx, created.ID, "jsonld")
	require.NoError(t, err)
	assert.NotEmpty(t, jsonldData)
}

func TestExportDIDDocumentNotFound(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	_, err := service.ExportDIDDocument(ctx, "did:web:nonexistent", "json")
	assert.Error(t, err)
}

func TestGetDIDCount(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	initialCount := service.GetDIDCount(ctx)

	_, _ = service.CreateDID(ctx, DIDMethodWeb, "example.com:user12", []byte("key"), []ServiceEndpoint{})

	newCount := service.GetDIDCount(ctx)
	assert.Equal(t, initialCount+1, newCount)
}

func TestGetChainCount(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	count := service.GetChainCount(ctx)
	assert.Equal(t, 0, count)

	_, _ = service.CreateBridgeConnection(ctx, "ethereum", "polygon")

	newCount := service.GetChainCount(ctx)
	assert.Equal(t, count, newCount)
}

func TestGetCircuitCount(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	count := service.GetCircuitCount(ctx)
	assert.Equal(t, 0, count)

	circuit := &ZKProofCircuit{
		CircuitID: "test-circuit-2",
	}
	_ = service.RegisterCircuit(ctx, circuit)

	newCount := service.GetCircuitCount(ctx)
	assert.Equal(t, 1, newCount)
}

func TestIssueCredential(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	issuerDID := "did:web:issuer.example.com"
	holderDID := "did:web:holder.example.com"

	_ = service.vcService.RegisterIssuer(ctx, &IssuerInfo{
		DID:            issuerDID,
		Name:           "Test Issuer",
		CredentialTypes: []string{"IdentityCredential"},
		TrustLevel:     5,
	})

	expiration := time.Now().Add(24 * time.Hour)
	credential, err := service.vcService.IssueCredential(ctx, issuerDID, holderDID, []string{"IdentityCredential"}, map[string]interface{}{
		"name": "John Doe",
		"age": 30,
	}, &expiration)
	require.NoError(t, err)
	assert.NotNil(t, credential)
	assert.NotEmpty(t, credential.ID)
	assert.Equal(t, issuerDID, credential.Issuer)
}

func TestVerifyCredential(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	issuerDID := "did:web:issuer.example.com"
	holderDID := "did:web:holder.example.com"

	_ = service.vcService.RegisterIssuer(ctx, &IssuerInfo{
		DID:            issuerDID,
		Name:           "Test Issuer",
		CredentialTypes: []string{"IdentityCredential"},
		TrustLevel:     5,
	})

	expiration := time.Now().Add(24 * time.Hour)
	credential, err := service.vcService.IssueCredential(ctx, issuerDID, holderDID, []string{"IdentityCredential"}, map[string]interface{}{
		"name": "John Doe",
	}, &expiration)
	require.NoError(t, err)

	result, err := service.vcService.VerifyCredential(ctx, credential.ID, true, true, true)
	require.NoError(t, err)
	assert.True(t, result.Valid)
}

func TestVerifyExpiredCredential(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	issuerDID := "did:web:issuer.example.com"
	holderDID := "did:web:holder.example.com"

	_ = service.vcService.RegisterIssuer(ctx, &IssuerInfo{
		DID:            issuerDID,
		Name:           "Test Issuer",
		CredentialTypes: []string{"IdentityCredential"},
		TrustLevel:     5,
	})

	expiration := time.Now().Add(-1 * time.Hour)
	credential, err := service.vcService.IssueCredential(ctx, issuerDID, holderDID, []string{"IdentityCredential"}, map[string]interface{}{
		"name": "John Doe",
	}, &expiration)
	require.NoError(t, err)

	result, err := service.vcService.VerifyCredential(ctx, credential.ID, true, true, true)
	require.NoError(t, err)
	assert.False(t, result.Valid)
}

func TestRevokeCredential(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	issuerDID := "did:web:issuer.example.com"
	holderDID := "did:web:holder.example.com"

	_ = service.vcService.RegisterIssuer(ctx, &IssuerInfo{
		DID:            issuerDID,
		Name:           "Test Issuer",
		CredentialTypes: []string{"IdentityCredential"},
		TrustLevel:     5,
	})

	expiration := time.Now().Add(24 * time.Hour)
	credential, err := service.vcService.IssueCredential(ctx, issuerDID, holderDID, []string{"IdentityCredential"}, map[string]interface{}{
		"name": "John Doe",
	}, &expiration)
	require.NoError(t, err)

	err = service.vcService.RevokeCredential(ctx, credential.ID, issuerDID, "User requested")
	require.NoError(t, err)
}

func TestCreatePresentation(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	issuerDID := "did:web:issuer.example.com"
	holderDID := "did:web:holder.example.com"

	_ = service.vcService.RegisterIssuer(ctx, &IssuerInfo{
		DID:            issuerDID,
		Name:           "Test Issuer",
		CredentialTypes: []string{"IdentityCredential"},
		TrustLevel:     5,
	})

	expiration := time.Now().Add(24 * time.Hour)
	credential, err := service.vcService.IssueCredential(ctx, issuerDID, holderDID, []string{"IdentityCredential"}, map[string]interface{}{
		"name": "John Doe",
	}, &expiration)
	require.NoError(t, err)

	vp, err := service.vcService.CreatePresentation(ctx, holderDID, []*VerifiableCredential{credential}, "challenge123", "example.com")
	require.NoError(t, err)
	assert.NotNil(t, vp)
	assert.Equal(t, holderDID, vp.Holder)
}

func TestVerifyPresentation(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	issuerDID := "did:web:issuer.example.com"
	holderDID := "did:web:holder.example.com"

	_ = service.vcService.RegisterIssuer(ctx, &IssuerInfo{
		DID:            issuerDID,
		Name:           "Test Issuer",
		CredentialTypes: []string{"IdentityCredential"},
		TrustLevel:     5,
	})

	expiration := time.Now().Add(24 * time.Hour)
	credential, err := service.vcService.IssueCredential(ctx, issuerDID, holderDID, []string{"IdentityCredential"}, map[string]interface{}{
		"name": "John Doe",
	}, &expiration)
	require.NoError(t, err)

	vp, err := service.vcService.CreatePresentation(ctx, holderDID, []*VerifiableCredential{credential}, "challenge123", "example.com")
	require.NoError(t, err)

	result, err := service.vcService.VerifyPresentation(ctx, vp, "challenge123", "example.com", true)
	require.NoError(t, err)
	assert.True(t, result.Valid)
}
