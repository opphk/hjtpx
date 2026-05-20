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
	assert.NotNil(t, service.registries)
	assert.NotNil(t, service.zkProver)
	assert.NotNil(t, service.zkVerifier)
	assert.NotNil(t, service.crossChain)
	assert.NotNil(t, service.issuerService)
}

func TestCreateDID(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	req := &DIDCreateRequest{
		Method:            DIDMethodWeb,
		MethodSpecificID:  "example.com:user1",
		PublicKey:         []byte("test-public-key"),
		ServiceEndpoints:  []ServiceEndpoint{},
	}

	resp, err := service.CreateDID(ctx, req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.DID)
	assert.NotNil(t, resp.Document)
	assert.Contains(t, resp.DID, "did:web")
}

func TestCreateDIDAlreadyExists(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	req := &DIDCreateRequest{
		Method:           DIDMethodWeb,
		MethodSpecificID: "example.com:user2",
		PublicKey:        []byte("test-public-key"),
	}

	resp1, err := service.CreateDID(ctx, req)
	require.NoError(t, err)
	assert.True(t, resp1.Success)

	resp2, err := service.CreateDID(ctx, req)
	assert.False(t, resp2.Success)
	assert.Error(t, err)
}

func TestResolveDID(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	createReq := &DIDCreateRequest{
		Method:           DIDMethodWeb,
		MethodSpecificID: "example.com:user3",
		PublicKey:        []byte("test-public-key"),
	}

	createResp, err := service.CreateDID(ctx, createReq)
	require.NoError(t, err)

	doc, err := service.ResolveDID(ctx, createResp.DID)
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, createResp.DID, doc.ID)
}

func TestResolveDIDNotFound(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	_, err := service.ResolveDID(ctx, "did:web:nonexistent")
	assert.Error(t, err)
	assert.Equal(t, ErrDIDNotFound, err)
}

func TestIssueCredential(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	issuerDID := "did:web:issuer.example.com"
	service.RegisterIssuer(&IssuerConfig{
		DID:             issuerDID,
		Name:            "Test Issuer",
		PublicKey:       []byte("issuer-public-key"),
		CredentialTypes: []string{"IdentityCredential"},
		TrustLevel:      5,
		RevocationList:  make(map[string]bool),
	})

	holderDID := "did:web:holder.example.com"
	holderReq := &DIDCreateRequest{
		Method:           DIDMethodWeb,
		MethodSpecificID: "holder.example.com",
		PublicKey:        []byte("holder-public-key"),
	}
	_, err := service.CreateDID(ctx, holderReq)
	require.NoError(t, err)

	expTime := time.Now().Add(24 * time.Hour)
	issueReq := &VCCreateRequest{
		IssuerDID:      issuerDID,
		HolderDID:      holderDID,
		CredentialType: []string{"IdentityCredential"},
		Claims: map[string]interface{}{
			"name":    "John Doe",
			"email":   "john@example.com",
			"age":     30,
		},
		ExpirationDate: &expTime,
	}

	resp, err := service.IssueCredential(ctx, issueReq)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Credential)
	assert.NotEmpty(t, resp.Credential.ID)
	assert.Equal(t, issuerDID, resp.Credential.Issuer)
}

func TestVerifyCredential(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	issuerDID := "did:web:issuer.example.com"
	service.RegisterIssuer(&IssuerConfig{
		DID:             issuerDID,
		Name:            "Test Issuer",
		PublicKey:       []byte("issuer-public-key"),
		CredentialTypes: []string{"IdentityCredential"},
		TrustLevel:      5,
		RevocationList:  make(map[string]bool),
	})

	holderDID := "did:web:holder.example.com"
	holderReq := &DIDCreateRequest{
		Method:           DIDMethodWeb,
		MethodSpecificID: "holder.example.com",
		PublicKey:        []byte("holder-public-key"),
	}
	_, err := service.CreateDID(ctx, holderReq)
	require.NoError(t, err)

	expTime := time.Now().Add(24 * time.Hour)
	issueReq := &VCCreateRequest{
		IssuerDID:      issuerDID,
		HolderDID:      holderDID,
		CredentialType: []string{"IdentityCredential"},
		Claims: map[string]interface{}{
			"name": "John Doe",
		},
		ExpirationDate: &expTime,
	}

	issueResp, err := service.IssueCredential(ctx, issueReq)
	require.NoError(t, err)

	verifyReq := &VCVerifyRequest{
		Credential:   issueResp.Credential,
		CheckExpired: true,
		CheckRevoked: true,
		CheckProof:   true,
	}

	verifyResp, err := service.VerifyCredential(ctx, verifyReq)
	require.NoError(t, err)
	assert.True(t, verifyResp.Valid)
}

func TestVerifyExpiredCredential(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	issuerDID := "did:web:issuer.example.com"
	service.RegisterIssuer(&IssuerConfig{
		DID:             issuerDID,
		Name:            "Test Issuer",
		PublicKey:       []byte("issuer-public-key"),
		CredentialTypes: []string{"IdentityCredential"},
		TrustLevel:      5,
		RevocationList:  make(map[string]bool),
	})

	holderDID := "did:web:holder.example.com"
	holderReq := &DIDCreateRequest{
		Method:           DIDMethodWeb,
		MethodSpecificID: "holder.example.com",
		PublicKey:        []byte("holder-public-key"),
	}
	_, err := service.CreateDID(ctx, holderReq)
	require.NoError(t, err)

	expTime := time.Now().Add(-1 * time.Hour)
	issueReq := &VCCreateRequest{
		IssuerDID:      issuerDID,
		HolderDID:      holderDID,
		CredentialType: []string{"IdentityCredential"},
		Claims: map[string]interface{}{
			"name": "John Doe",
		},
		ExpirationDate: &expTime,
	}

	issueResp, err := service.IssueCredential(ctx, issueReq)
	require.NoError(t, err)

	verifyReq := &VCVerifyRequest{
		Credential:   issueResp.Credential,
		CheckExpired: true,
	}

	verifyResp, err := service.VerifyCredential(ctx, verifyReq)
	require.NoError(t, err)
	assert.False(t, verifyResp.Valid)
	assert.Contains(t, verifyResp.Errors[0], "expired")
}

func TestCreateAndVerifyPresentation(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	issuerDID := "did:web:issuer.example.com"
	service.RegisterIssuer(&IssuerConfig{
		DID:             issuerDID,
		Name:            "Test Issuer",
		PublicKey:       []byte("issuer-public-key"),
		CredentialTypes: []string{"IdentityCredential"},
		TrustLevel:      5,
		RevocationList:  make(map[string]bool),
	})

	holderDID := "did:web:holder.example.com"
	holderReq := &DIDCreateRequest{
		Method:           DIDMethodWeb,
		MethodSpecificID: "holder.example.com",
		PublicKey:        []byte("holder-public-key"),
	}
	_, err := service.CreateDID(ctx, holderReq)
	require.NoError(t, err)

	expTime := time.Now().Add(24 * time.Hour)
	issueReq := &VCCreateRequest{
		IssuerDID:      issuerDID,
		HolderDID:      holderDID,
		CredentialType: []string{"IdentityCredential"},
		Claims: map[string]interface{}{
			"name": "John Doe",
		},
		ExpirationDate: &expTime,
	}

	issueResp, err := service.IssueCredential(ctx, issueReq)
	require.NoError(t, err)

	vp, err := service.CreatePresentation(ctx, holderDID, []*VerifiableCredential{issueResp.Credential}, "challenge123", "example.com")
	require.NoError(t, err)
	assert.NotNil(t, vp)
	assert.Equal(t, holderDID, vp.Holder)

	vpReq := &VPVerifyRequest{
		Presentation:         vp,
		Challenge:            "challenge123",
		Domain:               "example.com",
		VerifyCredentials:    true,
	}

	vpResp, err := service.VerifyPresentation(ctx, vpReq)
	require.NoError(t, err)
	assert.True(t, vpResp.Valid)
}

func TestGenerateZKProof(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	issuerDID := "did:web:issuer.example.com"
	service.RegisterIssuer(&IssuerConfig{
		DID:             issuerDID,
		Name:            "Test Issuer",
		PublicKey:       []byte("issuer-public-key"),
		CredentialTypes: []string{"IdentityCredential"},
		TrustLevel:      5,
		RevocationList:  make(map[string]bool),
	})

	holderDID := "did:web:holder.example.com"
	holderReq := &DIDCreateRequest{
		Method:           DIDMethodWeb,
		MethodSpecificID: "holder.example.com",
		PublicKey:        []byte("holder-public-key"),
	}
	_, err := service.CreateDID(ctx, holderReq)
	require.NoError(t, err)

	expTime := time.Now().Add(24 * time.Hour)
	issueReq := &VCCreateRequest{
		IssuerDID:      issuerDID,
		HolderDID:      holderDID,
		CredentialType: []string{"IdentityCredential"},
		Claims: map[string]interface{}{
			"name": "John Doe",
		},
		ExpirationDate: &expTime,
	}

	issueResp, err := service.IssueCredential(ctx, issueReq)
	require.NoError(t, err)

	zkReq := &ZKProofRequest{
		RequestID:      "req-123",
		Challenge:      "zk-challenge",
		Domain:         "example.com",
		ClaimTypes:     []string{"name", "age"},
		SchemaHash:     "schema-hash",
		CircuitVersion: "v1",
		Nonce:          "nonce-123",
	}

	zkResp, err := service.GenerateZKProof(ctx, issueResp.Credential, zkReq)
	require.NoError(t, err)
	assert.True(t, zkResp.Success)
	assert.NotNil(t, zkResp.Proof)
}

func TestVerifyZKProof(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	zkProof := &ZKProof{
		ID:     "urn:zkproof:test",
		Type:   []string{"BbsBlsProof2020"},
		Proof:  &ZKProofValue{
			ProofType:         "Groth16",
			CircuitIdentifier: "test-circuit",
		},
		GeneratedAt: time.Now(),
	}

	valid, err := service.VerifyZKProof(ctx, zkProof)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestVerifyZKProofInvalid(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	zkProof := &ZKProof{
		ID:          "urn:zkproof:test",
		Type:        []string{"BbsBlsProof2020"},
		Proof:       nil,
		GeneratedAt: time.Now(),
	}

	valid, err := service.VerifyZKProof(ctx, zkProof)
	assert.False(t, valid)
	assert.Error(t, err)
}

func TestRegisterCrossChainDID(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	did := "did:web:user.example.com"
	chainID := "ethereum"
	publicKey := []byte("cross-chain-public-key")

	err := service.RegisterCrossChainDID(ctx, did, chainID, publicKey)
	require.NoError(t, err)
}

func TestVerifyCrossChainIdentity(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	did := "did:web:user.example.com"
	chainID := "ethereum"
	publicKey := []byte("cross-chain-public-key")

	err := service.RegisterCrossChainDID(ctx, did, chainID, publicKey)
	require.NoError(t, err)

	req := &CrossChainVerifyRequest{
		DID:         did,
		TargetChain: chainID,
	}

	resp, err := service.VerifyCrossChainIdentity(ctx, req)
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.NotEmpty(t, resp.ChainStates)
}

func TestVerifyCrossChainIdentityNotFound(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	req := &CrossChainVerifyRequest{
		DID:         "did:web:nonexistent",
		TargetChain: "ethereum",
	}

	resp, err := service.VerifyCrossChainIdentity(ctx, req)
	require.NoError(t, err)
	assert.False(t, resp.Valid)
}

func TestListDIDs(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	method := DIDMethodWeb

	for i := 0; i < 3; i++ {
		req := &DIDCreateRequest{
			Method:           method,
			MethodSpecificID: "example.com:user" + string(rune('0'+i)),
			PublicKey:        []byte("test-key"),
		}
		_, err := service.CreateDID(ctx, req)
		require.NoError(t, err)
	}

	dids := service.ListDIDs(ctx, method)
	assert.Len(t, dids, 3)
}

func TestUpdateDID(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	req := &DIDCreateRequest{
		Method:           DIDMethodWeb,
		MethodSpecificID: "example.com:user",
		PublicKey:        []byte("original-key"),
	}

	createResp, err := service.CreateDID(ctx, req)
	require.NoError(t, err)

	updates := &DIDDocument{
		Service: []ServiceEndpoint{
			{
				ID:              createResp.DID + "#service-1",
				Type:            "LinkedDomains",
				ServiceEndpoint: "https://example.com",
			},
		},
	}

	err = service.UpdateDID(ctx, createResp.DID, updates)
	require.NoError(t, err)

	doc, err := service.ResolveDID(ctx, createResp.DID)
	require.NoError(t, err)
	assert.Len(t, doc.Service, 1)
}

func TestDeleteDID(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	req := &DIDCreateRequest{
		Method:           DIDMethodWeb,
		MethodSpecificID: "example.com:user-delete",
		PublicKey:        []byte("test-key"),
	}

	createResp, err := service.CreateDID(ctx, req)
	require.NoError(t, err)

	err = service.DeleteDID(ctx, createResp.DID)
	require.NoError(t, err)

	_, err = service.ResolveDID(ctx, createResp.DID)
	assert.Error(t, err)
}

func TestRegisterIssuer(t *testing.T) {
	service := NewDIDVerificationService()

	config := &IssuerConfig{
		DID:             "did:web:issuer.example.com",
		Name:            "Test Issuer",
		PublicKey:       []byte("issuer-public-key"),
		CredentialTypes: []string{"IdentityCredential"},
		TrustLevel:      5,
		RevocationList:  make(map[string]bool),
	}

	err := service.RegisterIssuer(config)
	require.NoError(t, err)

	assert.NotNil(t, service.issuerService.issuers[config.DID])
}

func TestVerifyCredentialNilCredential(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	req := &VCVerifyRequest{
		Credential:   nil,
		CheckExpired: true,
	}

	resp, err := service.VerifyCredential(ctx, req)
	assert.False(t, resp.Valid)
	assert.Error(t, err)
}

func TestVerifyPresentationNilPresentation(t *testing.T) {
	service := NewDIDVerificationService()
	ctx := context.Background()

	req := &VPVerifyRequest{
		Presentation:  nil,
		VerifyCredentials: true,
	}

	resp, err := service.VerifyPresentation(ctx, req)
	assert.False(t, resp.Valid)
	assert.Error(t, err)
}

func TestParseDID(t *testing.T) {
	testCases := []struct {
		name      string
		didString string
		wantErr   bool
	}{
		{
			name:      "Valid Web DID",
			didString: "did:web:example.com",
			wantErr:   false,
		},
		{
			name:      "Valid Ethr DID",
			didString: "did:ethr:0x1234567890123456789012345678901234567890",
			wantErr:   false,
		},
		{
			name:      "Invalid DID",
			didString: "invalid:did",
			wantErr:   true,
		},
		{
			name:      "Unsupported Method",
			didString: "did:unsupported:123",
			wantErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseDID(tc.didString)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetServices(t *testing.T) {
	service := NewDIDVerificationService()

	zkVerifier := service.GetZKVerifier()
	assert.NotNil(t, zkVerifier)

	crossChain := service.GetCrossChainBridge()
	assert.NotNil(t, crossChain)
	assert.True(t, crossChain.bridgeEnabled)
}
