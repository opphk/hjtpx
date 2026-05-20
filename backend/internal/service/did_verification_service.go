package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrDIDInvalid           = errors.New("invalid DID")
	ErrDIDNotFound          = errors.New("DID not found")
	ErrDIDAlreadyExists     = errors.New("DID already exists")
	ErrVCInvalid            = errors.New("invalid verifiable credential")
	ErrVCExpired            = errors.New("verifiable credential expired")
	ErrVCRevoked            = errors.New("verifiable credential revoked")
	ErrVPInvalid            = errors.New("invalid verifiable presentation")
	ErrZKProofInvalid       = errors.New("invalid ZK proof")
	ErrZKProofVerification  = errors.New("ZK proof verification failed")
	ErrChainNotSupported    = errors.New("chain not supported")
	ErrDIDDocNotValid       = errors.New("DID document not valid")
)

type DIDMethod string

const (
	DIDMethodWeb    DIDMethod = "did:web"
	DIDMethodEthr   DIDMethod = "did:ethr"
	DIDMethodIon    DIDMethod = "did:ion"
	DIDMethodSolana DIDMethod = "did:sol"
)

type DIDVerificationService struct {
	mu           sync.RWMutex
	didRegistry  *DIDRegistry
	vcService    *VerifiableCredentialService
	zkService    *ZKProofService
	bridge       *CrossChainBridge
	auditLog     *DIDAuditLog
}

type DIDRegistry struct {
	mu      sync.RWMutex
	dids    map[string]*DIDDocument
	methods map[DIDMethod]*MethodRegistry
}

type MethodRegistry struct {
	Method DIDMethod
	DIDs   map[string]*DIDDocument
}

type DIDAuditLog struct {
	mu    sync.RWMutex
	logs  []DIDAuditEntry
}

type DIDAuditEntry struct {
	DID        string    `json:"did"`
	Action     string    `json:"action"`
	ActorID    string    `json:"actor_id"`
	Details    string    `json:"details"`
	Timestamp  time.Time `json:"timestamp"`
	IPAddress  string    `json:"ip_address"`
	Result     string    `json:"result"`
}

type CrossChainBridge struct {
	mu      sync.RWMutex
	chains  map[string]*ChainConfig
	bridges map[string]*BridgeConnection
}

type ChainConfig struct {
	ChainID      string
	Name         string
	Protocol     string
	RPCEndpoint  string
	BlockExplorer string
	Status       string
}

type BridgeConnection struct {
	SourceChain  string
	TargetChain  string
	Protocol     string
	Status       string
	LastSyncTime time.Time
}

type ZKProofService struct {
	mu       sync.RWMutex
	circuits map[string]*ZKProofCircuit
	provers  map[string]*ZKProver
	verifiers map[string]*ZKVerifier
}

type ZKProofCircuit struct {
	CircuitID   string
	Name        string
	Version     string
	Constraints int
	ABI         []byte
}

type ZKProver struct {
	ProverID    string
	CircuitID   string
	ProvingKey  []byte
	ProofFormat string
}

type ZKVerifier struct {
	VerifierID  string
	CircuitID   string
	VerifyKey   []byte
}

func NewDIDVerificationService() *DIDVerificationService {
	return &DIDVerificationService{
		didRegistry: NewDIDRegistry(),
		vcService:   NewVerifiableCredentialService(),
		zkService:  NewZKProofService(),
		bridge:     NewCrossChainBridge(),
		auditLog:   NewDIDAuditLog(),
	}
}

func NewDIDRegistry() *DIDRegistry {
	return &DIDRegistry{
		dids:    make(map[string]*DIDDocument),
		methods: make(map[DIDMethod]*MethodRegistry),
	}
}

func NewDIDAuditLog() *DIDAuditLog {
	return &DIDAuditLog{
		logs: make([]DIDAuditEntry, 0),
	}
}

func NewCrossChainBridge() *CrossChainBridge {
	return &CrossChainBridge{
		chains:  make(map[string]*ChainConfig),
		bridges: make(map[string]*BridgeConnection),
	}
}

func NewZKProofService() *ZKProofService {
	return &ZKProofService{
		circuits:  make(map[string]*ZKProofCircuit),
		provers:   make(map[string]*ZKProver),
		verifiers: make(map[string]*ZKVerifier),
	}
}

func (s *DIDVerificationService) CreateDID(ctx context.Context, method DIDMethod, methodSpecificID string, publicKey []byte, services []ServiceEndpoint) (*DIDDocument, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	didString := fmt.Sprintf("did:%s:%s", method, methodSpecificID)

	if _, exists := s.didRegistry.dids[didString]; exists {
		return nil, ErrDIDAlreadyExists
	}

	vm := &VerificationMethod{
		ID:           didString + "#keys-1",
		Type:         "EcdsaSecp256k1VerificationKey2019",
		Controller:   didString,
		PublicKeyHex: base64.StdEncoding.EncodeToString(publicKey),
	}

	doc := &DIDDocument{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/v1",
		},
		ID:                   didString,
		VerificationMethod:   []VerificationMethod{*vm},
		Authentication:       []interface{}{didString + "#keys-1"},
		AssertionMethod:      []interface{}{didString + "#keys-1"},
		KeyAgreement:         []KeyAgreementMethod{},
		Service:              services,
		Created:              time.Now(),
		Updated:              time.Now(),
	}

	s.didRegistry.dids[didString] = doc

	if _, exists := s.didRegistry.methods[method]; !exists {
		s.didRegistry.methods[method] = &MethodRegistry{
			Method: method,
			DIDs:   make(map[string]*DIDDocument),
		}
	}
	s.didRegistry.methods[method].DIDs[didString] = doc

	s.auditLog.logs = append(s.auditLog.logs, DIDAuditEntry{
		DID:       didString,
		Action:    "create",
		ActorID:   methodSpecificID,
		Details:   "DID created successfully",
		Timestamp: time.Now(),
		IPAddress: "127.0.0.1",
		Result:    "success",
	})

	return doc, nil
}

func (s *DIDVerificationService) ResolveDID(ctx context.Context, didString string) (*DIDDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, exists := s.didRegistry.dids[didString]
	if !exists {
		return nil, ErrDIDNotFound
	}

	return doc, nil
}

func (s *DIDVerificationService) UpdateDID(ctx context.Context, didString string, updates *DIDDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, exists := s.didRegistry.dids[didString]
	if !exists {
		return ErrDIDNotFound
	}

	if updates.VerificationMethod != nil {
		doc.VerificationMethod = updates.VerificationMethod
	}
	if updates.Service != nil {
		doc.Service = updates.Service
	}
	if updates.Authentication != nil {
		doc.Authentication = updates.Authentication
	}
	if updates.AssertionMethod != nil {
		doc.AssertionMethod = updates.AssertionMethod
	}

	doc.Updated = time.Now()

	s.auditLog.logs = append(s.auditLog.logs, DIDAuditEntry{
		DID:       didString,
		Action:    "update",
		ActorID:   "system",
		Details:   "DID updated successfully",
		Timestamp: time.Now(),
		IPAddress: "127.0.0.1",
		Result:    "success",
	})

	return nil
}

func (s *DIDVerificationService) DeactivateDID(ctx context.Context, didString string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, exists := s.didRegistry.dids[didString]
	if !exists {
		return ErrDIDNotFound
	}

	proof := &DIDProof{
		Type:               "EcdsaSecp256k1Signature2019",
		Created:            time.Now(),
		ProofPurpose:       "assertionMethod",
		VerificationMethod: didString + "#keys-1",
		ProofValue:         base64.StdEncoding.EncodeToString([]byte("deactivation-proof")),
	}

	doc.Proof = proof
	doc.Updated = time.Now()

	s.auditLog.logs = append(s.auditLog.logs, DIDAuditEntry{
		DID:       didString,
		Action:    "deactivate",
		ActorID:   "system",
		Details:   reason,
		Timestamp: time.Now(),
		IPAddress: "127.0.0.1",
		Result:    "success",
	})

	return nil
}

func (s *DIDVerificationService) ListDIDs(ctx context.Context, method DIDMethod) []*DIDDocument {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if method != "" {
		if registry, exists := s.didRegistry.methods[method]; exists {
			docs := make([]*DIDDocument, 0, len(registry.DIDs))
			for _, doc := range registry.DIDs {
				docs = append(docs, doc)
			}
			return docs
		}
		return []*DIDDocument{}
	}

	docs := make([]*DIDDocument, 0, len(s.didRegistry.dids))
	for _, doc := range s.didRegistry.dids {
		docs = append(docs, doc)
	}
	return docs
}

func (s *DIDVerificationService) VerifyDIDAuthentication(ctx context.Context, didString string, challenge string, signature []byte) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, exists := s.didRegistry.dids[didString]
	if !exists {
		return false, ErrDIDNotFound
	}

	for _, vm := range doc.VerificationMethod {
		if vm.ID == didString+"#keys-1" {
			return true, nil
		}
	}

	return false, nil
}

func (s *DIDVerificationService) AddChainConfiguration(ctx context.Context, config *ChainConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bridge.chains[config.ChainID] = config

	return nil
}

func (s *DIDVerificationService) CreateBridgeConnection(ctx context.Context, sourceChain, targetChain string) (*BridgeConnection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bridgeKey := fmt.Sprintf("%s->%s", sourceChain, targetChain)

	bridge := &BridgeConnection{
		SourceChain:  sourceChain,
		TargetChain:  targetChain,
		Protocol:     "cross-chain-did",
		Status:       "active",
		LastSyncTime: time.Now(),
	}

	s.bridge.bridges[bridgeKey] = bridge

	return bridge, nil
}

func (s *DIDVerificationService) SyncCrossChainDID(ctx context.Context, did string, sourceChain, targetChain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bridgeKey := fmt.Sprintf("%s->%s", sourceChain, targetChain)

	bridge, exists := s.bridge.bridges[bridgeKey]
	if !exists {
		return errors.New("bridge connection not found")
	}

	bridge.LastSyncTime = time.Now()

	return nil
}

func (s *DIDVerificationService) GetCrossChainState(ctx context.Context, did string) (map[string]*ChainDIDState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	states := make(map[string]*ChainDIDState)

	for chainID, chainConfig := range s.bridge.chains {
		states[chainID] = &ChainDIDState{
			ChainID:     chainID,
			DID:         did,
			PublicKey:   []byte{},
			State:       "synced",
			BlockNumber: uint64(time.Now().Unix()),
			Timestamp:   time.Now(),
		}
		_ = chainConfig
	}

	return states, nil
}

func (s *DIDVerificationService) RegisterCircuit(ctx context.Context, circuit *ZKProofCircuit) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.zkService.circuits[circuit.CircuitID] = circuit

	return nil
}

func (s *DIDVerificationService) GenerateZKProof(ctx context.Context, did string, claimTypes []string, challenge string) (*ZKProof, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, exists := s.didRegistry.dids[did]
	if !exists {
		return nil, ErrDIDNotFound
	}

	proof := &ZKProof{
		ID:                     "urn:zkproof:" + generateUUID(),
		Type:                   []string{"BbsBlsProof2020", "CLSignature2023"},
		VerifiableCredential:   nil,
		Proof: &ZKProofValue{
			ProofType:         "Groth16",
			CircuitIdentifier: "credential-circuit-v1",
			ProofInputs:       []string{},
			ProofOutputs:      []string{},
			PublicSignals:     []string{},
			ProofData:         base64.StdEncoding.EncodeToString([]byte("zk-proof-data")),
		},
		GeneratedAt: time.Now(),
	}

	_ = doc
	_ = claimTypes
	_ = challenge

	return proof, nil
}

func (s *DIDVerificationService) VerifyZKProof(ctx context.Context, proof *ZKProof) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if proof == nil || proof.Proof == nil {
		return false, ErrZKProofInvalid
	}

	return true, nil
}

func (s *DIDVerificationService) GetAuditLogs(ctx context.Context, did string, limit int) []*DIDAuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs := make([]DIDAuditEntry, 0)
	for _, entry := range s.auditLog.logs {
		if entry.DID == did {
			logs = append(logs, entry)
		}
	}

	if limit > 0 && len(logs) > limit {
		return logs[len(logs)-limit:]
	}

	return logs
}

func (s *DIDVerificationService) ExportDIDDocument(ctx context.Context, did string, format string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, exists := s.didRegistry.dids[did]
	if !exists {
		return nil, ErrDIDNotFound
	}

	switch format {
	case "json":
		return json.MarshalIndent(doc, "", "  ")
	case "jsonld":
		jsonBytes, _ := json.Marshal(doc)
		jsonld := map[string]interface{}{
			"@context": "https://www.w3.org/ns/did/v1",
			"document": jsonBytes,
		}
		return json.MarshalIndent(jsonld, "", "  ")
	default:
		return nil, errors.New("unsupported format")
	}
}

func (s *DIDVerificationService) GetDIDCount(ctx context.Context) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.didRegistry.dids)
}

func (s *DIDVerificationService) GetChainCount(ctx context.Context) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.bridge.chains)
}

func (s *DIDVerificationService) GetCircuitCount(ctx context.Context) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.zkService.circuits)
}

type VerifiableCredentialService struct {
	mu          sync.RWMutex
	credentials map[string]*VerifiableCredential
	schemas     map[string]*VCSchema
	issuers     map[string]*IssuerInfo
	revocations map[string]*RevocationEntry
}

type VCSchema struct {
	SchemaID    string                 `json:"schema_id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Fields      []string               `json:"fields"`
	Constraints map[string]interface{} `json:"constraints"`
}

type IssuerInfo struct {
	DID            string   `json:"did"`
	Name           string   `json:"name"`
	URL            string   `json:"url"`
	CredentialTypes []string `json:"credential_types"`
	TrustLevel     int      `json:"trust_level"`
	LogoURL        string   `json:"logo_url"`
}

type RevocationEntry struct {
	CredentialID string    `json:"credential_id"`
	RevokedAt    time.Time `json:"revoked_at"`
	RevokedBy    string    `json:"revoked_by"`
	Reason       string    `json:"reason"`
}

func NewVerifiableCredentialService() *VerifiableCredentialService {
	return &VerifiableCredentialService{
		credentials: make(map[string]*VerifiableCredential),
		schemas:     make(map[string]*VCSchema),
		issuers:     make(map[string]*IssuerInfo),
		revocations: make(map[string]*RevocationEntry),
	}
}

func (s *VerifiableCredentialService) IssueCredential(ctx context.Context, issuerDID string, holderDID string, credentialType []string, claims map[string]interface{}, expirationDate *time.Time) (*VerifiableCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	issuer, exists := s.issuers[issuerDID]
	if !exists {
		return nil, errors.New("issuer not registered")
	}

	credential := &VerifiableCredential{
		Context: []string{
			"https://www.w3.org/2018/credentials/v1",
			"https://www.w3.org/2018/credentials/examples/v1",
		},
		ID:               "urn:uuid:" + generateUUID(),
		Type:             append([]string{"VerifiableCredential"}, credentialType...),
		Issuer:           issuerDID,
		IssuanceDate:     time.Now(),
		ExpirationDate:   expirationDate,
		CredentialSubject: []CredentialSubject{
			{
				ID:     holderDID,
				Type:   "Person",
				Claims: claims,
			},
		},
		CredentialStatus: &CredentialStatus{
			ID:        issuerDID + "/credentials/status/1",
			Type:      "RevocationList2020Entry",
			Revoker:   issuerDID,
			Timestamp: time.Now(),
		},
	}

	proof := &VCProof{
		Type:               "EcdsaSecp256k1Signature2019",
		Created:            time.Now(),
		ProofPurpose:       "assertionMethod",
		VerificationMethod: issuerDID + "#keys-1",
		ProofValue:         base64.StdEncoding.EncodeToString(generateSignature(credential)),
		ProofType:          "BbsBlsSignature2020",
	}

	credential.Proof = proof

	s.credentials[credential.ID] = credential

	_ = issuer

	return credential, nil
}

func (s *VerifiableCredentialService) VerifyCredential(ctx context.Context, credentialID string, checkExpired bool, checkRevoked bool, checkProof bool) (*VCVerifyResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	credential, exists := s.credentials[credentialID]
	if !exists {
		return &VCVerifyResponse{
			Valid:       false,
			Errors:      []string{"credential not found"},
			CredentialID: credentialID,
		}, ErrVCInvalid
	}

	errors := make([]string, 0)
	warnings := make([]string, 0)

	if checkExpired && credential.ExpirationDate != nil {
		if time.Now().After(*credential.ExpirationDate) {
			errors = append(errors, "credential has expired")
		}
	}

	if checkRevoked {
		if revocation, exists := s.revocations[credentialID]; exists {
			errors = append(errors, fmt.Sprintf("credential was revoked at %s by %s: %s",
				revocation.RevokedAt.Format(time.RFC3339), revocation.RevokedBy, revocation.Reason))
		}
	}

	if checkProof && credential.Proof != nil {
		valid, err := s.verifyProof(credential)
		if err != nil || !valid {
			errors = append(errors, "proof verification failed")
		}
	}

	valid := len(errors) == 0

	return &VCVerifyResponse{
		Valid:        valid,
		Errors:       errors,
		Warnings:     warnings,
		CredentialID: credentialID,
	}, nil
}

func (s *VerifiableCredentialService) verifyProof(vc *VerifiableCredential) (bool, error) {
	if vc.Proof == nil {
		return false, ErrVCInvalid
	}

	return true, nil
}

func (s *VerifiableCredentialService) RevokeCredential(ctx context.Context, credentialID string, revokedBy string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.credentials[credentialID]; !exists {
		return ErrVCInvalid
	}

	s.revocations[credentialID] = &RevocationEntry{
		CredentialID: credentialID,
		RevokedAt:    time.Now(),
		RevokedBy:    revokedBy,
		Reason:       reason,
	}

	return nil
}

func (s *VerifiableCredentialService) RegisterIssuer(ctx context.Context, issuer *IssuerInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.issuers[issuer.DID] = issuer

	return nil
}

func (s *VerifiableCredentialService) RegisterSchema(ctx context.Context, schema *VCSchema) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.schemas[schema.SchemaID] = schema

	return nil
}

func (s *VerifiableCredentialService) GetCredential(ctx context.Context, credentialID string) (*VerifiableCredential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	credential, exists := s.credentials[credentialID]
	if !exists {
		return nil, ErrVCInvalid
	}

	return credential, nil
}

func (s *VerifiableCredentialService) CreatePresentation(ctx context.Context, holderDID string, credentials []*VerifiableCredential, challenge string, domain string) (*VerifiablePresentation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vp := &VerifiablePresentation{
		Context: []string{
			"https://www.w3.org/2018/credentials/v1",
		},
		ID:                       "urn:uuid:" + generateUUID(),
		Type:                     []string{"VerifiablePresentation"},
		Holder:                   holderDID,
		VerifiableCredential:     credentials,
	}

	proof := &VPProof{
		Type:               "EcdsaSecp256k1Signature2019",
		Created:            time.Now(),
		Challenge:          challenge,
		Domain:             domain,
		ProofPurpose:       "authentication",
		VerificationMethod: holderDID + "#keys-1",
		ProofValue:         base64.StdEncoding.EncodeToString(generateSignature(vp)),
	}

	vp.Proof = proof

	return vp, nil
}

func (s *VerifiableCredentialService) VerifyPresentation(ctx context.Context, vp *VerifiablePresentation, challenge string, domain string, verifyCredentials bool) (*VPVerifyResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	errors := make([]string, 0)

	if vp.Proof != nil {
		if challenge != "" && vp.Proof.Challenge != challenge {
			errors = append(errors, "challenge mismatch")
		}
		if domain != "" && vp.Proof.Domain != domain {
			errors = append(errors, "domain mismatch")
		}
	}

	if verifyCredentials {
		for _, vc := range vp.VerifiableCredential {
			_, err := s.VerifyCredential(ctx, vc.ID, true, true, true)
			if err != nil {
				errors = append(errors, fmt.Sprintf("credential %s verification failed: %v", vc.ID, err))
			}
		}
	}

	valid := len(errors) == 0

	return &VPVerifyResponse{
		Valid:           valid,
		Errors:          errors,
		Holder:          vp.Holder,
		CredentialCount: len(vp.VerifiableCredential),
	}, nil
}

func generateUUID() string {
	hash := sha256.Sum256([]byte(time.Now().String()))
	return base64.URLEncoding.EncodeToString(hash[:16])
}

func generateSignature(data interface{}) []byte {
	jsonBytes, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonBytes)
	return hash[:]
}
