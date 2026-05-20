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
	ErrDIDAlreadyExists      = errors.New("DID already exists")
	ErrVCInvalid            = errors.New("invalid verifiable credential")
	ErrVCExpired             = errors.New("verifiable credential expired")
	ErrVCRevoked            = errors.New("verifiable credential revoked")
	ErrVPInvalid            = errors.New("invalid verifiable presentation")
	ErrZKProofInvalid       = errors.New("invalid ZK proof")
	ErrZKProofVerification  = errors.New("ZK proof verification failed")
	ErrChainNotSupported    = errors.New("chain not supported")
	ErrDIDDocNotValid       = errors.New("DID document not valid")
)

type DIDMethod string

const (
	DIDMethodWeb     DIDMethod = "did:web"
	DIDMethodEthr    DIDMethod = "did:ethr"
	DIDMethodIon     DIDMethod = "did:ion"
	DIDMethodSolana  DIDMethod = "did:sol"
)

type DID struct {
	Method   DIDMethod
	MethodSpecificID string
	Path     string
}

type DIDDocument struct {
	Context           []string                 `json:"@context"`
	ID                string                   `json:"id"`
	Controller        []string                 `json:"controller,omitempty"`
	VerificationMethod []VerificationMethod     `json:"verificationMethod,omitempty"`
	Authentication    []interface{}            `json:"authentication,omitempty"`
	AssertionMethod   []interface{}            `json:"assertionMethod,omitempty"`
	KeyAgreement      []KeyAgreementMethod     `json:"keyAgreement,omitempty"`
	CapabilityInvocation []interface{}         `json:"capabilityInvocation,omitempty"`
	CapabilityDelegation []interface{}         `json:"capabilityDelegation,omitempty"`
	Service           []ServiceEndpoint        `json:"service,omitempty"`
	Created           time.Time                `json:"created"`
	Updated           time.Time                `json:"updated"`
	Proof             *DIDProof                 `json:"proof,omitempty"`
}

type VerificationMethod struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Controller   string `json:"controller"`
	PublicKeyJwk *JWK  `json:"publicKeyJwk,omitempty"`
	PublicKeyHex string `json:"publicKeyHex,omitempty"`
}

type KeyAgreementMethod struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Controller   string `json:"controller"`
	PublicKeyJwk *JWK  `json:"publicKeyJwk,omitempty"`
}

type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Alg string `json:"alg"`
}

type ServiceEndpoint struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`
	ServiceEndpoint string            `json:"serviceEndpoint"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
}

type DIDProof struct {
	Type               string    `json:"type"`
	Created            time.Time `json:"created"`
	ProofPurpose       string    `json:"proofPurpose"`
	VerificationMethod string    `json:"verificationMethod"`
	ProofValue         string    `json:"proofValue"`
}

type CredentialStatus struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Revoker   string `json:"revoker"`
	Timestamp time.Time `json:"timestamp"`
}

type CredentialSubject struct {
	ID    string                 `json:"id"`
	Type  string                 `json:"type"`
	Claims map[string]interface{} `json:"claims"`
}

type VerifiableCredential struct {
	Context          []string              `json:"@context"`
	ID               string                `json:"id"`
	Type             []string              `json:"type"`
	Issuer           string                `json:"issuer"`
	IssuanceDate     time.Time             `json:"issuanceDate"`
	ExpirationDate   *time.Time            `json:"expirationDate,omitempty"`
	CredentialSubject []CredentialSubject  `json:"credentialSubject"`
	CredentialStatus *CredentialStatus     `json:"credentialStatus,omitempty"`
	Proof            *VCProof              `json:"proof,omitempty"`
}

type VCProof struct {
	Type               string    `json:"type"`
	Created            time.Time `json:"created"`
	ProofPurpose       string    `json:"proofPurpose"`
	VerificationMethod string    `json:"verificationMethod"`
	ProofValue         string    `json:"proofValue"`
	ProofType          string    `json:"proofType"`
}

type VerifiablePresentation struct {
	Context        []string           `json:"@context"`
	ID             string             `json:"id,omitempty"`
	Type           []string           `json:"type"`
	Holder         string             `json:"holder"`
	VerifiableCredential []*VerifiableCredential `json:"verifiableCredential,omitempty"`
	Proof          *VPProof           `json:"proof,omitempty"`
}

type VPProof struct {
	Type               string    `json:"type"`
	Created            time.Time `json:"created"`
	Challenge          string    `json:"challenge,omitempty"`
	Domain             string    `json:"domain,omitempty"`
	ProofPurpose       string    `json:"proofPurpose"`
	VerificationMethod string    `json:"verificationMethod"`
	ProofValue         string    `json:"proofValue"`
}

type ZKProof struct {
	ID               string                 `json:"id"`
	Type             []string               `json:"type"`
	VerifiableCredential *VerifiableCredential `json:"verifiableCredential"`
	Proof            *ZKProofValue          `json:"proof"`
	GeneratedAt      time.Time              `json:"generatedAt"`
}

type ZKProofValue struct {
	ProofType         string   `json:"proofType"`
	CircuitIdentifier string   `json:"circuitIdentifier"`
	ProofInputs       []string `json:"proofInputs"`
	ProofOutputs      []string `json:"proofOutputs"`
	PublicSignals     []string `json:"publicSignals"`
	ProofData         string   `json:"proofData"`
}

type ZKProofRequest struct {
	RequestID      string                 `json:"requestId"`
	Challenge      string                 `json:"challenge"`
	Domain         string                 `json:"domain"`
	ClaimTypes     []string               `json:"claimTypes"`
	SchemaHash     string                 `json:"schemaHash"`
	CircuitVersion string                 `json:"circuitVersion"`
	Nonce          string                 `json:"nonce"`
}

type ZKProofResponse struct {
	Success       bool          `json:"success"`
	Proof         *ZKProof     `json:"proof,omitempty"`
	ErrorMessage  string        `json:"errorMessage,omitempty"`
	ProofDuration time.Duration `json:"proofDuration"`
}

type DIDRegistry struct {
	mu      sync.RWMutex
	dids    map[string]*DIDDocument
	method  DIDMethod
}

type CrossChainIdentity struct {
	mu            sync.RWMutex
	chains        map[string]*ChainDIDState
	bridgeEnabled bool
}

type ChainDIDState struct {
	ChainID     string
	DID         string
	PublicKey   []byte
	State       string
	BlockNumber uint64
	Timestamp   time.Time
}

type DIDVerificationService struct {
	mu          sync.RWMutex
	registries  map[DIDMethod]*DIDRegistry
	zkProver    *ZKProver
	zkVerifier  *ZKVerifier
	crossChain  *CrossChainIdentity
	issuerService *DIDIssuerService
}

type DIDIssuerService struct {
	mu     sync.RWMutex
	issuers map[string]*IssuerConfig
}

type IssuerConfig struct {
	DID            string
	Name           string
	PublicKey      []byte
	CredentialTypes []string
	TrustLevel     int
	RevocationList map[string]bool
}

type ZKProver struct {
	mu         sync.RWMutex
	circuits   map[string]*ZKCircuit
	keys       map[string]*ProvingKey
}

type ZKCircuit struct {
	CircuitID     string
	Version       string
	ABI           []interface{}
	Constraints   int
	MaxInputs     int
}

type ProvingKey struct {
	Curve   string
	Points  [][]byte
}

type ZKVerifier struct {
	mu      sync.RWMutex
	vKeys   map[string]*VerificationKey
	circuits map[string]*ZKCircuit
}

type VerificationKey struct {
	Curve   string
	Points  [][]byte
}

type DIDCreateRequest struct {
	Method          DIDMethod
	MethodSpecificID string
	PublicKey        []byte
	ServiceEndpoints []ServiceEndpoint
}

type DIDCreateResponse struct {
	Success       bool
	DID           string
	Document      *DIDDocument
	ErrorMessage  string
}

type VCCreateRequest struct {
	IssuerDID      string
	HolderDID      string
	CredentialType []string
	Claims         map[string]interface{}
	ExpirationDate *time.Time
}

type VCCreateResponse struct {
	Success       bool
	Credential    *VerifiableCredential
	ErrorMessage  string
}

type VCVerifyRequest struct {
	Credential    *VerifiableCredential
	CheckExpired  bool
	CheckRevoked  bool
	CheckProof    bool
}

type VCVerifyResponse struct {
	Valid         bool
	Errors        []string
	Warnings      []string
	CredentialID  string
}

type VPVerifyRequest struct {
	Presentation  *VerifiablePresentation
	Challenge     string
	Domain        string
	VerifyCredentials bool
}

type VPVerifyResponse struct {
	Valid           bool
	Errors          []string
	Holder          string
	CredentialCount int
}

type CrossChainVerifyRequest struct {
	DID            string
	TargetChain    string
	ProofOfOwnership []byte
}

type CrossChainVerifyResponse struct {
	Valid         bool
	ChainStates   map[string]*ChainDIDState
	ErrorMessage  string
}

func NewDIDVerificationService() *DIDVerificationService {
	return &DIDVerificationService{
		registries:  make(map[DIDMethod]*DIDRegistry),
		zkProver:    NewZKProver(),
		zkVerifier:  NewZKVerifier(),
		crossChain:  NewCrossChainIdentity(),
		issuerService: NewDIDIssuerService(),
	}
}

func NewDIDRegistry(method DIDMethod) *DIDRegistry {
	return &DIDRegistry{
		dids:   make(map[string]*DIDDocument),
		method: method,
	}
}

func NewZKProver() *ZKProver {
	return &ZKProver{
		circuits: make(map[string]*ZKCircuit),
		keys:    make(map[string]*ProvingKey),
	}
}

func NewZKVerifier() *ZKVerifier {
	return &ZKVerifier{
		vKeys:    make(map[string]*VerificationKey),
		circuits: make(map[string]*ZKCircuit),
	}
}

func NewCrossChainIdentity() *CrossChainIdentity {
	return &CrossChainIdentity{
		chains:        make(map[string]*ChainDIDState),
		bridgeEnabled: true,
	}
}

func NewDIDIssuerService() *DIDIssuerService {
	return &DIDIssuerService{
		issuers: make(map[string]*IssuerConfig),
	}
}

func (s *DIDVerificationService) CreateDID(ctx context.Context, req *DIDCreateRequest) (*DIDCreateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.registries[req.Method]; !exists {
		s.registries[req.Method] = NewDIDRegistry(req.Method)
	}

	did := &DID{
		Method:          req.Method,
		MethodSpecificID: req.MethodSpecificID,
	}
	didString := formatDID(did)

	if _, exists := s.registries[req.Method].dids[didString]; exists {
		return &DIDCreateResponse{
			Success:      false,
			ErrorMessage: ErrDIDAlreadyExists.Error(),
		}, ErrDIDAlreadyExists
	}

	vm := &VerificationMethod{
		ID:           didString + "#keys-1",
		Type:         "EcdsaSecp256k1VerificationKey2019",
		Controller:   didString,
		PublicKeyHex: base64.StdEncoding.EncodeToString(req.PublicKey),
	}

	doc := &DIDDocument{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/v1",
		},
		ID:                 didString,
		VerificationMethod: []VerificationMethod{*vm},
		Authentication:     []interface{}{didString + "#keys-1"},
		AssertionMethod:    []interface{}{didString + "#keys-1"},
		Created:            time.Now(),
		Updated:            time.Now(),
	}

	if len(req.ServiceEndpoints) > 0 {
		doc.Service = req.ServiceEndpoints
	}

	s.registries[req.Method].dids[didString] = doc

	return &DIDCreateResponse{
		Success:  true,
		DID:      didString,
		Document: doc,
	}, nil
}

func (s *DIDVerificationService) ResolveDID(ctx context.Context, didString string) (*DIDDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	did, err := parseDID(didString)
	if err != nil {
		return nil, err
	}

	registry, exists := s.registries[did.Method]
	if !exists {
		return nil, ErrDIDNotFound
	}

	doc, exists := registry.dids[didString]
	if !exists {
		return nil, ErrDIDNotFound
	}

	return doc, nil
}

func (s *DIDVerificationService) IssueCredential(ctx context.Context, req *VCCreateRequest) (*VCCreateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	issuerConfig, exists := s.issuerService.issuers[req.IssuerDID]
	if !exists {
		return nil, ErrDIDNotFound
	}

	subject := CredentialSubject{
		ID:     req.HolderDID,
		Type:   "Person",
		Claims: req.Claims,
	}

	vc := &VerifiableCredential{
		Context: []string{
			"https://www.w3.org/2018/credentials/v1",
			"https://www.w3.org/2018/credentials/examples/v1",
		},
		ID:               "urn:uuid:" + generateUUID(),
		Type:             append([]string{"VerifiableCredential"}, req.CredentialType...),
		Issuer:           req.IssuerDID,
		IssuanceDate:     time.Now(),
		ExpirationDate:   req.ExpirationDate,
		CredentialSubject: []CredentialSubject{subject},
		CredentialStatus: &CredentialStatus{
			ID:        issuerConfig.DID + "/credentials/status/1",
			Type:      "RevocationList2020Entry",
			Revoker:   issuerConfig.DID,
			Timestamp: time.Now(),
		},
	}

	proof := &VCProof{
		Type:               "EcdsaSecp256k1Signature2019",
		Created:            time.Now(),
		ProofPurpose:       "assertionMethod",
		VerificationMethod: issuerConfig.DID + "#keys-1",
		ProofValue:         base64.StdEncoding.EncodeToString(generateSignature(vc)),
		ProofType:          "BbsBlsSignature2020",
	}

	vc.Proof = proof

	return &VCCreateResponse{
		Success:    true,
		Credential: vc,
	}, nil
}

func (s *DIDVerificationService) VerifyCredential(ctx context.Context, req *VCVerifyRequest) (*VCVerifyResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	errors := make([]string, 0)
	warnings := make([]string, 0)

	if req.Credential == nil {
		return &VCVerifyResponse{
			Valid: false,
			Errors: []string{"credential is nil"},
		}, ErrVCInvalid
	}

	if req.CheckExpired && req.Credential.ExpirationDate != nil {
		if time.Now().After(*req.Credential.ExpirationDate) {
			errors = append(errors, "credential has expired")
		}
	}

	if req.CheckRevoked && req.Credential.CredentialStatus != nil {
		revoked, err := s.checkRevocation(req.Credential.CredentialStatus)
		if err == nil && revoked {
			errors = append(errors, "credential has been revoked")
		}
	}

	if req.CheckProof && req.Credential.Proof != nil {
		valid, err := s.verifyVCProof(req.Credential)
		if err != nil || !valid {
			errors = append(errors, "proof verification failed")
		}
	}

	if len(errors) == 0 {
		return &VCVerifyResponse{
			Valid:        true,
			Errors:       errors,
			Warnings:     warnings,
			CredentialID: req.Credential.ID,
		}, nil
	}

	return &VCVerifyResponse{
		Valid:        false,
		Errors:       errors,
		Warnings:     warnings,
		CredentialID: req.Credential.ID,
	}, nil
}

func (s *DIDVerificationService) CreatePresentation(ctx context.Context, did string, credentials []*VerifiableCredential, challenge string, domain string) (*VerifiablePresentation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vp := &VerifiablePresentation{
		Context: []string{
			"https://www.w3.org/2018/credentials/v1",
		},
		ID:      "urn:uuid:" + generateUUID(),
		Type:    []string{"VerifiablePresentation"},
		Holder:  did,
		VerifiableCredential: credentials,
	}

	proof := &VPProof{
		Type:               "EcdsaSecp256k1Signature2019",
		Created:            time.Now(),
		Challenge:          challenge,
		Domain:             domain,
		ProofPurpose:       "authentication",
		VerificationMethod: did + "#keys-1",
		ProofValue:         base64.StdEncoding.EncodeToString(generateSignature(vp)),
	}

	vp.Proof = proof

	return vp, nil
}

func (s *DIDVerificationService) VerifyPresentation(ctx context.Context, req *VPVerifyRequest) (*VPVerifyResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	errors := make([]string, 0)

	if req.Presentation == nil {
		return &VPVerifyResponse{
			Valid:  false,
			Errors: []string{"presentation is nil"},
		}, ErrVPInvalid
	}

	if req.Challenge != "" && req.Presentation.Proof != nil {
		if req.Presentation.Proof.Challenge != req.Challenge {
			errors = append(errors, "challenge mismatch")
		}
	}

	if req.Domain != "" && req.Presentation.Proof != nil {
		if req.Presentation.Proof.Domain != req.Domain {
			errors = append(errors, "domain mismatch")
		}
	}

	if req.VerifyCredentials && len(req.Presentation.VerifiableCredential) > 0 {
		for _, vc := range req.Presentation.VerifiableCredential {
			verifyReq := &VCVerifyRequest{
				Credential:   vc,
				CheckExpired: true,
				CheckRevoked: true,
				CheckProof:   true,
			}
			result, _ := s.VerifyCredential(ctx, verifyReq)
			if result != nil && !result.Valid {
				errors = append(errors, result.Errors...)
			}
		}
	}

	if len(errors) == 0 {
		return &VPVerifyResponse{
			Valid:           true,
			Errors:          errors,
			Holder:          req.Presentation.Holder,
			CredentialCount: len(req.Presentation.VerifiableCredential),
		}, nil
	}

	return &VPVerifyResponse{
		Valid:           false,
		Errors:          errors,
		Holder:          req.Presentation.Holder,
		CredentialCount: len(req.Presentation.VerifiableCredential),
	}, nil
}

func (s *DIDVerificationService) GenerateZKProof(ctx context.Context, credential *VerifiableCredential, req *ZKProofRequest) (*ZKProofResponse, error) {
	start := time.Now()

	zkProof := &ZKProof{
		ID:                     "urn:zkproof:" + generateUUID(),
		Type:                   []string{"BbsBlsProof2020", "CLSignature2023"},
		VerifiableCredential: credential,
		Proof: &ZKProofValue{
			ProofType:         "Groth16",
			CircuitIdentifier: "credential-circuit-v1",
			ProofInputs:       []string{},
			ProofOutputs:      []string{},
			PublicSignals:     []string{},
			ProofData:         base64.StdEncoding.EncodeToString(generateZKProofData()),
		},
		GeneratedAt: time.Now(),
	}

	return &ZKProofResponse{
		Success:       true,
		Proof:         zkProof,
		ProofDuration: time.Since(start),
	}, nil
}

func (s *DIDVerificationService) VerifyZKProof(ctx context.Context, proof *ZKProof) (bool, error) {
	if proof == nil || proof.Proof == nil {
		return false, ErrZKProofInvalid
	}

	if proof.GeneratedAt.IsZero() {
		return false, ErrZKProofInvalid
	}

	return true, nil
}

func (s *DIDVerificationService) RegisterCrossChainDID(ctx context.Context, did string, chainID string, publicKey []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := &ChainDIDState{
		ChainID:     chainID,
		DID:         did,
		PublicKey:   publicKey,
		State:       "active",
		BlockNumber: uint64(time.Now().Unix()),
		Timestamp:   time.Now(),
	}

	key := chainID + ":" + did
	s.crossChain.chains[key] = state

	return nil
}

func (s *DIDVerificationService) VerifyCrossChainIdentity(ctx context.Context, req *CrossChainVerifyRequest) (*CrossChainVerifyResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chainStates := make(map[string]*ChainDIDState)

	for key, state := range s.crossChain.chains {
		if state.DID == req.DID {
			chainStates[key] = state
		}
	}

	if len(chainStates) == 0 {
		return &CrossChainVerifyResponse{
			Valid:        false,
			ChainStates:  chainStates,
			ErrorMessage: "no cross-chain state found",
		}, nil
	}

	return &CrossChainVerifyResponse{
		Valid:       true,
		ChainStates: chainStates,
	}, nil
}

func (s *DIDVerificationService) RegisterIssuer(config *IssuerConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.issuerService.issuers[config.DID] = config
	return nil
}

func (s *DIDVerificationService) RevokeCredential(ctx context.Context, credentialID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, registry := range s.registries {
		for _, doc := range registry.dids {
			for _, vm := range doc.VerificationMethod {
				if vm.ID == credentialID {
					return nil
				}
			}
		}
	}

	return ErrVCInvalid
}

func (s *DIDVerificationService) checkRevocation(status *CredentialStatus) (bool, error) {
	if status == nil {
		return false, nil
	}

	issuerConfig, exists := s.issuerService.issuers[status.Revoker]
	if !exists {
		return false, nil
	}

	return issuerConfig.RevocationList[status.ID], nil
}

func (s *DIDVerificationService) verifyVCProof(vc *VerifiableCredential) (bool, error) {
	if vc.Proof == nil {
		return false, ErrVCInvalid
	}

	return true, nil
}

func formatDID(did *DID) string {
	result := "did:" + string(did.Method) + ":" + did.MethodSpecificID
	if did.Path != "" {
		result += did.Path
	}
	return result
}

func parseDID(didString string) (*DID, error) {
	var did DID
	n, err := fmt.Sscanf(didString, "did:%s:%s", &did.Method, &did.MethodSpecificID)
	if err != nil || n < 2 {
		return nil, ErrDIDInvalid
	}

	validMethods := map[DIDMethod]bool{
		DIDMethodWeb:     true,
		DIDMethodEthr:    true,
		DIDMethodIon:     true,
		DIDMethodSolana:  true,
	}

	if !validMethods[did.Method] {
		return nil, ErrDIDInvalid
	}

	return &did, nil
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

func generateZKProofData() []byte {
	hash := sha256.Sum256([]byte(time.Now().String() + "zk-proof"))
	return hash[:]
}

func (s *DIDVerificationService) ListDIDs(ctx context.Context, method DIDMethod) []*DIDDocument {
	s.mu.RLock()
	defer s.mu.RUnlock()

	registry, exists := s.registries[method]
	if !exists {
		return nil
	}

	result := make([]*DIDDocument, 0, len(registry.dids))
	for _, doc := range registry.dids {
		result = append(result, doc)
	}

	return result
}

func (s *DIDVerificationService) UpdateDID(ctx context.Context, didString string, updates *DIDDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	did, err := parseDID(didString)
	if err != nil {
		return err
	}

	registry, exists := s.registries[did.Method]
	if !exists {
		return ErrDIDNotFound
	}

	if _, exists := registry.dids[didString]; !exists {
		return ErrDIDNotFound
	}

	updates.ID = didString
	updates.Updated = time.Now()

	registry.dids[didString] = updates

	return nil
}

func (s *DIDVerificationService) DeleteDID(ctx context.Context, didString string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	did, err := parseDID(didString)
	if err != nil {
		return err
	}

	registry, exists := s.registries[did.Method]
	if !exists {
		return ErrDIDNotFound
	}

	delete(registry.dids, didString)

	return nil
}

func (s *DIDVerificationService) GetZKVerifier() *ZKVerifier {
	return s.zkVerifier
}

func (s *DIDVerificationService) GetCrossChainBridge() *CrossChainIdentity {
	return s.crossChain
}
