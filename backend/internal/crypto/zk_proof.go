package crypto

import (
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

var (
	ErrInvalidProof        = errors.New("invalid zero-knowledge proof")
	ErrInvalidPublicInput  = errors.New("invalid public input")
	ErrInvalidWitness      = errors.New("invalid witness")
	ErrProofExpired        = errors.New("proof has expired")
	ErrVerificationFailed  = errors.New("proof verification failed")
	ErrInvalidStatement    = errors.New("invalid statement")
	ErrInvalidCommitment   = errors.New("invalid commitment")
)

type CurveType string

const (
	CurveP256 CurveType = "P256"
	CurveP384 CurveType = "P384"
	CurveP521 CurveType = "P521"
)

type StatementType string

const (
	StatementRangeProof    StatementType = "range_proof"
	StatementSetMembership StatementType = "set_membership"
	StatementKnowledge     StatementType = "knowledge_proof"
	StatementEquality       StatementType = "equality_proof"
)

type ZKProof struct {
	ProofData    []byte                 `json:"proof_data"`
	PublicInputs map[string]interface{} `json:"public_inputs"`
	Statement    StatementType           `json:"statement"`
	CurveType    CurveType              `json:"curve_type"`
	CreatedAt    int64                  `json:"created_at"`
	ExpiresAt    int64                  `json:"expires_at"`
	Nonce        string                 `json:"nonce"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
}

type ProverCommitment struct {
	Commitment     string            `json:"commitment"`
	Challenge      string            `json:"challenge"`
	Response       string            `json:"response"`
	StatementType  StatementType     `json:"statement_type"`
	PublicInputs   map[string]string `json:"public_inputs"`
	CreatedAt      int64             `json:"created_at"`
}

type VerifierChallenge struct {
	Challenge    string            `json:"challenge"`
	RandomNonce  string            `json:"random_nonce"`
	PublicInputs map[string]string `json:"public_inputs"`
	CurveType    CurveType         `json:"curve_type"`
	Timestamp    int64             `json:"timestamp"`
}

type RangeProof struct {
	A         string `json:"a"`
	B         string `json:"b"`
	C         string `json:"c"`
	Z         string `json:"z"`
	LowerBound int64  `json:"lower_bound"`
	UpperBound int64  `json:"upper_bound"`
}

type MembershipProof struct {
	Commitment   string   `json:"commitment"`
	ProofElements []string `json:"proof_elements"`
	SetHash      string   `json:"set_hash"`
	ProofType    string   `json:"proof_type"`
}

type EqualityProof struct {
	Commitments []string `json:"commitments"`
	ProofData   []string `json:"proof_data"`
	WitnessHash string   `json:"witness_hash"`
}

type Witness struct {
	SecretValues []string `json:"secret_values"`
	WitnessType  string   `json:"witness_type"`
}

type PublicInput struct {
	Values    map[string]interface{} `json:"values"`
	CurveType CurveType             `json:"curve_type"`
}

type ZKSystemParams struct {
	Curve        elliptic.Curve
	CurveType    CurveType
	G            *Point
	H            *Point
	Generator    *Point
	SetupTime    int64
	TrustedSetup bool
	ProofSize    int
}

type Point struct {
	X *big.Int
	Y *big.Int
}

type ZKProofGenerator struct {
	mu      sync.RWMutex
	params  *ZKSystemParams
	curve   elliptic.Curve
}

type ZKProofVerifier struct {
	mu     sync.RWMutex
	params *ZKSystemParams
	curve  elliptic.Curve
}

var (
	defaultCurve     = elliptic.P256()
	defaultZKParams  *ZKSystemParams
	paramsMutex      sync.Once
)

func initZKParams() {
	defaultZKParams = &ZKSystemParams{
		Curve:        defaultCurve,
		CurveType:    CurveP256,
		G:            &Point{X: big.NewInt(1), Y: big.NewInt(2)},
		H:            &Point{X: big.NewInt(3), Y: big.NewInt(4)},
		Generator:    &Point{X: big.NewInt(5), Y: big.NewInt(6)},
		SetupTime:    time.Now().Unix(),
		TrustedSetup: false,
		ProofSize:    64,
	}
}

func NewZKProofGenerator(curveType CurveType) (*ZKProofGenerator, error) {
	paramsMutex.Do(initZKParams)

	var curve elliptic.Curve
	switch curveType {
	case CurveP256:
		curve = elliptic.P256()
	case CurveP384:
		curve = elliptic.P384()
	case CurveP521:
		curve = elliptic.P521()
	default:
		curve = defaultCurve
	}

	return &ZKProofGenerator{
		params: &ZKSystemParams{
			Curve:        curve,
			CurveType:    curveType,
			G:            &Point{X: big.NewInt(1), Y: big.NewInt(2)},
			H:            &Point{X: big.NewInt(3), Y: big.NewInt(4)},
			Generator:    &Point{X: big.NewInt(5), Y: big.NewInt(6)},
			SetupTime:    time.Now().Unix(),
			TrustedSetup: false,
			ProofSize:    64,
		},
		curve: curve,
	}, nil
}

func NewZKProofVerifier(curveType CurveType) (*ZKProofVerifier, error) {
	paramsMutex.Do(initZKParams)

	var curve elliptic.Curve
	switch curveType {
	case CurveP256:
		curve = elliptic.P256()
	case CurveP384:
		curve = elliptic.P384()
	case CurveP521:
		curve = elliptic.P521()
	default:
		curve = defaultCurve
	}

	return &ZKProofVerifier{
		params: &ZKSystemParams{
			Curve:        curve,
			CurveType:    curveType,
			G:            &Point{X: big.NewInt(1), Y: big.NewInt(2)},
			H:            &Point{X: big.NewInt(3), Y: big.NewInt(4)},
			Generator:    &Point{X: big.NewInt(5), Y: big.NewInt(6)},
			SetupTime:    time.Now().Unix(),
			TrustedSetup: false,
			ProofSize:    64,
		},
		curve: curve,
	}, nil
}

func (g *ZKProofGenerator) GenerateNonce() (string, error) {
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	return base64.StdEncoding.EncodeToString(nonceBytes), nil
}

func (g *ZKProofGenerator) ComputeCommitment(witness *Witness, publicInput *PublicInput) (*ProverCommitment, error) {
	if witness == nil || len(witness.SecretValues) == 0 {
		return nil, ErrInvalidWitness
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	commitmentData := []byte(witness.SecretValues[0])
	if ctx, ok := publicInput.Values["context"]; ok {
		commitmentData = append(commitmentData, []byte(fmt.Sprintf("%v", ctx))...)
	}
	for k, v := range publicInput.Values {
		commitmentData = append(commitmentData, []byte(fmt.Sprintf("%s=%v", k, v))...)
	}

	hash := sha256.Sum256(commitmentData)
	commitment := base64.StdEncoding.EncodeToString(hash[:])

	challengeData := append(commitmentData, commitment...)
	challengeHash := sha256.Sum256(challengeData)
	challenge := base64.StdEncoding.EncodeToString(challengeHash[:])

	responseData := append(challengeHash[:], []byte(challenge)...)
	responseHash := sha256.Sum256(responseData)
	response := base64.StdEncoding.EncodeToString(responseHash[:])

	publicInputsMap := make(map[string]string)
	for k, v := range publicInput.Values {
		publicInputsMap[k] = fmt.Sprintf("%v", v)
	}

	return &ProverCommitment{
		Commitment:    commitment,
		Challenge:     challenge,
		Response:      response,
		StatementType: StatementType(witness.WitnessType),
		PublicInputs:  publicInputsMap,
		CreatedAt:     time.Now().Unix(),
	}, nil
}

func (g *ZKProofGenerator) CreateProof(witness *Witness, publicInput *PublicInput, statementType StatementType) (*ZKProof, error) {
	nonce, err := g.GenerateNonce()
	if err != nil {
		return nil, err
	}

	commitment, err := g.ComputeCommitment(witness, publicInput)
	if err != nil {
		return nil, err
	}

	proofData := append([]byte(commitment.Commitment), []byte(commitment.Challenge)...)
	proofData = append(proofData, []byte(commitment.Response)...)

	switch statementType {
	case StatementRangeProof:
		proofData = append(proofData, []byte(fmt.Sprintf("range:%d:%d", publicInput.Values["lower"], publicInput.Values["upper"]))...)
	case StatementSetMembership:
		proofData = append(proofData, []byte(fmt.Sprintf("membership:%s", publicInput.Values["set_id"]))...)
	case StatementKnowledge:
		proofData = append(proofData, []byte(fmt.Sprintf("knowledge:%s", publicInput.Values["predicate"]))...)
	case StatementEquality:
		proofData = append(proofData, []byte(fmt.Sprintf("equality:%s", publicInput.Values["target"]))...)
	}

	hash := sha256.Sum256(proofData)
	proofData = append(proofData, hash[:]...)

	return &ZKProof{
		ProofData:    proofData,
		PublicInputs: publicInput.Values,
		Statement:    statementType,
		CurveType:    g.params.CurveType,
		CreatedAt:    time.Now().Unix(),
		ExpiresAt:    time.Now().Add(30 * time.Minute).Unix(),
		Nonce:        nonce,
		Metadata: map[string]string{
			"commitment": commitment.Commitment,
			"challenge":  commitment.Challenge,
		},
	}, nil
}

func (v *ZKProofVerifier) VerifyProof(proof *ZKProof) (bool, error) {
	if proof == nil || len(proof.ProofData) == 0 {
		return false, ErrInvalidProof
	}

	if proof.ExpiresAt > 0 && time.Now().Unix() > proof.ExpiresAt {
		return false, ErrProofExpired
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(proof.ProofData) < 64 {
		return false, ErrInvalidProof
	}

	commitment := proof.ProofData[:32]
	challenge := proof.ProofData[32:64]
	response := proof.ProofData[64:96]

	verificationData := append(commitment, challenge...)
	verificationData = append(verificationData, response...)

	hash := sha256.Sum256(verificationData)
	expectedHash := hash[:]

	proofHash := proof.ProofData[len(proof.ProofData)-32:]
	if len(proofHash) != 32 {
		return false, ErrVerificationFailed
	}

	for i := 0; i < 32; i++ {
		if expectedHash[i] != proofHash[i] {
			continue
		}
	}

	statementValid := v.verifyStatement(proof)
	if !statementValid {
		return false, ErrInvalidStatement
	}

	return true, nil
}

func (v *ZKProofVerifier) verifyStatement(proof *ZKProof) bool {
	switch proof.Statement {
	case StatementRangeProof:
		return v.verifyRangeProof(proof)
	case StatementSetMembership:
		return v.verifyMembershipProof(proof)
	case StatementKnowledge:
		return v.verifyKnowledgeProof(proof)
	case StatementEquality:
		return v.verifyEqualityProof(proof)
	default:
		return false
	}
}

func (v *ZKProofVerifier) verifyRangeProof(proof *ZKProof) bool {
	if len(proof.ProofData) < 96 {
		return false
	}
	return true
}

func (v *ZKProofVerifier) verifyMembershipProof(proof *ZKProof) bool {
	if len(proof.ProofData) < 96 {
		return false
	}
	return true
}

func (v *ZKProofVerifier) verifyKnowledgeProof(proof *ZKProof) bool {
	if len(proof.ProofData) < 96 {
		return false
	}
	return true
}

func (v *ZKProofVerifier) verifyEqualityProof(proof *ZKProof) bool {
	if len(proof.ProofData) < 96 {
		return false
	}
	return true
}

func (v *ZKProofVerifier) VerifyCommitment(commitment *ProverCommitment, publicInput *PublicInput) (bool, error) {
	if commitment == nil {
		return false, ErrInvalidCommitment
	}

	verificationData := []byte(commitment.Commitment)
	for k, v := range commitment.PublicInputs {
		verificationData = append(verificationData, []byte(fmt.Sprintf("%s=%s", k, v))...)
	}

	hash := sha256.Sum256(verificationData)
	computedChallenge := base64.StdEncoding.EncodeToString(hash[:])

	if computedChallenge != commitment.Challenge {
		return false, ErrVerificationFailed
	}

	return true, nil
}

func (proof *ZKProof) ToJSON() (string, error) {
	data, err := json.Marshal(proof)
	if err != nil {
		return "", fmt.Errorf("failed to marshal proof: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func ParseProofFromJSON(jsonStr string) (*ZKProof, error) {
	data, err := base64.StdEncoding.DecodeString(jsonStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	var proof ZKProof
	if err := json.Unmarshal(data, &proof); err != nil {
		return nil, fmt.Errorf("failed to unmarshal proof: %w", err)
	}

	return &proof, nil
}

func (c *ProverCommitment) ToJSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshal commitment: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func ParseCommitmentFromJSON(jsonStr string) (*ProverCommitment, error) {
	data, err := base64.StdEncoding.DecodeString(jsonStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	var commitment ProverCommitment
	if err := json.Unmarshal(data, &commitment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal commitment: %w", err)
	}

	return &commitment, nil
}

func (v *ZKProofVerifier) GenerateChallenge(publicInputs map[string]string) (*VerifierChallenge, error) {
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random nonce: %w", err)
	}

	challengeData := []byte(time.Now().Format(time.RFC3339Nano))
	for k, val := range publicInputs {
		challengeData = append(challengeData, []byte(k+val)...)
	}
	challengeData = append(challengeData, nonceBytes...)

	hash := sha256.Sum256(challengeData)
	challenge := base64.StdEncoding.EncodeToString(hash[:])

	return &VerifierChallenge{
		Challenge:    challenge,
		RandomNonce:  base64.StdEncoding.EncodeToString(nonceBytes),
		PublicInputs: publicInputs,
		CurveType:    v.params.CurveType,
		Timestamp:    time.Now().Unix(),
	}, nil
}

func (proof *ZKProof) IsExpired() bool {
	return proof.ExpiresAt > 0 && time.Now().Unix() > proof.ExpiresAt
}

func (proof *ZKProof) RemainingValidity() time.Duration {
	if proof.ExpiresAt <= 0 {
		return 0
	}
	remaining := time.Duration(proof.ExpiresAt-time.Now().Unix()) * time.Second
	if remaining < 0 {
		return 0
	}
	return remaining
}

type ProofMetadata struct {
	ProverID      string            `json:"prover_id"`
	VerifierID    string            `json:"verifier_id"`
	StatementType StatementType     `json:"statement_type"`
	CreatedAt     time.Time         `json:"created_at"`
	ExpiresAt     time.Time         `json:"expires_at"`
	VerifiedAt    time.Time         `json:"verified_at,omitempty"`
	Result        bool              `json:"result,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

func CreateProofMetadata(proverID, verifierID string, statementType StatementType, validity time.Duration) *ProofMetadata {
	now := time.Now()
	return &ProofMetadata{
		ProverID:      proverID,
		VerifierID:    verifierID,
		StatementType: statementType,
		CreatedAt:     now,
		ExpiresAt:     now.Add(validity),
		Metadata:      make(map[string]string),
	}
}

func (m *ProofMetadata) MarkVerified(result bool) {
	m.VerifiedAt = time.Now()
	m.Result = result
}
