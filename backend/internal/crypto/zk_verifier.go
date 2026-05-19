package crypto

import (
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
	ErrVerificationTimeout    = errors.New("verification timeout")
	ErrInvalidVerificationKey = errors.New("invalid verification key")
	ErrProofReplay           = errors.New("proof replay detected")
	ErrInvalidProofFormat    = errors.New("invalid proof format")
)

type VerificationMode string

const (
	ModeStrict   VerificationMode = "strict"
	ModeStandard VerificationMode = "standard"
	ModeFast     VerificationMode = "fast"
)

type VerificationResult struct {
	Valid        bool      `json:"valid"`
	Error        string    `json:"error,omitempty"`
	ProofID      string    `json:"proof_id"`
	VerifiedAt   time.Time `json:"verified_at"`
	Duration     time.Duration `json:"duration_ms"`
	Mode         VerificationMode `json:"mode"`
	Score        float64   `json:"score,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type VerificationConfig struct {
	Mode             VerificationMode
	Timeout          time.Duration
	EnableReplayCheck bool
	MaxProofAge      time.Duration
	StrictValidation bool
	EnableProofCaching bool
	CacheTTL         time.Duration
}

type ProofCache struct {
	proofs map[string]*CachedProof
	mu     sync.RWMutex
	ttl    time.Duration
}

type CachedProof struct {
	Proof      *ZKProof
	VerifiedAt time.Time
	Result     bool
}

type ZKVerifier struct {
	mu             sync.RWMutex
	verKey         *VerificationKey
	curveType      CurveType
	config         *VerificationConfig
	proofCache     *ProofCache
	stats          *VerificationStats
}

type VerificationStats struct {
	TotalVerifications    int64     `json:"total_verifications"`
	SuccessfulVerifications int64    `json:"successful_verifications"`
	FailedVerifications    int64     `json:"failed_verifications"`
	ReplayAttacksDetected  int64     `json:"replay_attacks_detected"`
	AverageDuration        int64     `json:"average_duration_ms"`
	LastVerification       time.Time `json:"last_verification"`
	mu                    sync.Mutex
}

type VerificationRequest struct {
	Proof        *ZKProof `json:"proof"`
	PublicInputs map[string]interface{} `json:"public_inputs"`
	SessionID    string   `json:"session_id,omitempty"`
	ClientNonce  string   `json:"client_nonce,omitempty"`
}

type BatchVerificationRequest struct {
	Proofs []VerificationRequest `json:"proofs"`
	Mode   VerificationMode       `json:"mode"`
}

type BatchVerificationResult struct {
	Results    []VerificationResult `json:"results"`
	TotalValid int                  `json:"total_valid"`
	TotalInvalid int                `json:"total_invalid"`
	TotalDuration int64             `json:"total_duration_ms"`
}

type ProofChallenge struct {
	Challenge  string    `json:"challenge"`
	Timestamp  int64     `json:"timestamp"`
	VerifierID string    `json:"verifier_id"`
	ExpiresAt  int64     `json:"expires_at"`
}

type VerificationPolicy struct {
	PolicyID       string       `json:"policy_id"`
	AllowedTypes   []StatementType `json:"allowed_types"`
	MaxProofAge    time.Duration `json:"max_proof_age"`
	RequireFreshness bool        `json:"require_freshness"`
	RequireSignature bool       `json:"require_signature"`
}

type VerifierService struct {
	verifier *ZKVerifier
	policies map[string]*VerificationPolicy
	mu       sync.RWMutex
}

func NewZKVerifier(curveType CurveType, config *VerificationConfig) (*ZKVerifier, error) {
	if config == nil {
		config = &VerificationConfig{
			Mode:              ModeStandard,
			Timeout:           30 * time.Second,
			EnableReplayCheck: true,
			MaxProofAge:       30 * time.Minute,
			StrictValidation:  true,
			EnableProofCaching: true,
			CacheTTL:          5 * time.Minute,
		}
	}

	zkVerifier := &ZKVerifier{
		curveType: curveType,
		config:    config,
		proofCache: &ProofCache{
			proofs: make(map[string]*CachedProof),
			ttl:    config.CacheTTL,
		},
		stats: &VerificationStats{},
	}

	if err := zkVerifier.initializeVerificationKey(); err != nil {
		return nil, err
	}

	return zkVerifier, nil
}

func (v *ZKVerifier) initializeVerificationKey() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.verKey = &VerificationKey{
		IC: []*Point{
			{X: big.NewInt(1), Y: big.NewInt(2)},
			{X: big.NewInt(3), Y: big.NewInt(4)},
		},
		VC: []*Point{
			{X: big.NewInt(5), Y: big.NewInt(6)},
			{X: big.NewInt(7), Y: big.NewInt(8)},
		},
		Alpha:    &Point{X: big.NewInt(9), Y: big.NewInt(10)},
		Beta:     &Point{X: big.NewInt(11), Y: big.NewInt(12)},
		Gamma:   &Point{X: big.NewInt(13), Y: big.NewInt(14)},
		Delta:   &Point{X: big.NewInt(15), Y: big.NewInt(16)},
		Protocol: "G16",
		CurveType: v.curveType,
		CreatedAt: time.Now().Unix(),
	}

	return nil
}

func (v *ZKVerifier) Verify(request *VerificationRequest) (*VerificationResult, error) {
	startTime := time.Now()

	result := &VerificationResult{
		ProofID:    generateProofID(),
		VerifiedAt: startTime,
		Mode:       v.config.Mode,
	}

	if request == nil || request.Proof == nil {
		result.Valid = false
		result.Error = "invalid request"
		result.Duration = time.Since(startTime)
		return result, ErrInvalidProof
	}

	if v.config.EnableReplayCheck {
		if v.isProofReplayed(request.Proof) {
			result.Valid = false
			result.Error = "proof replay detected"
			result.Duration = time.Since(startTime)
			v.recordFailedVerification()
			v.stats.mu.Lock()
			v.stats.ReplayAttacksDetected++
			v.stats.mu.Unlock()
			return result, ErrProofReplay
		}
	}

	if v.config.EnableProofCaching {
		if cachedResult := v.getCachedResult(request.Proof); cachedResult != nil {
			result.Valid = cachedResult.Result
			result.Duration = time.Since(startTime)
			result.Metadata = map[string]interface{}{
				"cached": true,
			}
			v.recordSuccessfulVerification(result.Duration)
			return result, nil
		}
	}

	if v.config.MaxProofAge > 0 {
		if request.Proof.ExpiresAt > 0 && time.Now().Unix() > request.Proof.ExpiresAt {
			result.Valid = false
			result.Error = "proof expired"
			result.Duration = time.Since(startTime)
			v.recordFailedVerification()
			return result, ErrProofExpired
		}
	}

	valid, err := v.performVerification(request)
	if err != nil {
		result.Valid = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		v.recordFailedVerification()
		return result, err
	}

	result.Valid = valid
	result.Duration = time.Since(startTime)
	result.Score = v.calculateVerificationScore(valid, result.Duration)

	if v.config.EnableProofCaching {
		v.cacheResult(request.Proof, valid)
	}

	if valid {
		v.markProofAsUsed(request.Proof)
		v.recordSuccessfulVerification(result.Duration)
	} else {
		v.recordFailedVerification()
	}

	return result, nil
}

func (v *ZKVerifier) performVerification(request *VerificationRequest) (bool, error) {
	proof := request.Proof

	if len(proof.ProofData) == 0 {
		return false, ErrInvalidProofFormat
	}

	switch v.config.Mode {
	case ModeStrict:
		return v.strictVerification(proof, request.PublicInputs)
	case ModeStandard:
		return v.standardVerification(proof, request.PublicInputs)
	case ModeFast:
		return v.fastVerification(proof)
	default:
		return v.standardVerification(proof, request.PublicInputs)
	}
}

func (v *ZKVerifier) strictVerification(proof *ZKProof, publicInputs map[string]interface{}) (bool, error) {
	if !v.validateProofStructure(proof) {
		return false, ErrInvalidProofFormat
	}

	if !v.validateStatementType(proof.Statement) {
		return false, ErrInvalidStatement
	}

	if publicInputs != nil {
		if !v.validatePublicInputs(proof, publicInputs) {
			return false, ErrInvalidPublicInput
		}
	}

	return v.verifyProofCryptographically(proof)
}

func (v *ZKVerifier) standardVerification(proof *ZKProof, publicInputs map[string]interface{}) (bool, error) {
	if !v.validateProofStructure(proof) {
		return false, ErrInvalidProofFormat
	}

	return v.verifyProofCryptographically(proof)
}

func (v *ZKVerifier) fastVerification(proof *ZKProof) (bool, error) {
	if len(proof.ProofData) < 64 {
		return false, ErrInvalidProofFormat
	}
	return true, nil
}

func (v *ZKVerifier) validateProofStructure(proof *ZKProof) bool {
	if proof == nil {
		return false
	}
	if proof.ProofData == nil || len(proof.ProofData) == 0 {
		return false
	}
	if proof.Statement == "" {
		return false
	}
	if proof.CurveType == "" {
		return false
	}
	return true
}

func (v *ZKVerifier) validateStatementType(statement StatementType) bool {
	validTypes := []StatementType{
		StatementRangeProof,
		StatementSetMembership,
		StatementKnowledge,
		StatementEquality,
	}

	for _, t := range validTypes {
		if t == statement {
			return true
		}
	}
	return false
}

func (v *ZKVerifier) validatePublicInputs(proof *ZKProof, publicInputs map[string]interface{}) bool {
	if len(proof.PublicInputs) == 0 && len(publicInputs) > 0 {
		return false
	}

	for key, expectedVal := range proof.PublicInputs {
		if actualVal, ok := publicInputs[key]; ok {
			if fmt.Sprintf("%v", expectedVal) != fmt.Sprintf("%v", actualVal) {
				return false
			}
		}
	}

	return true
}

func (v *ZKVerifier) verifyProofCryptographically(proof *ZKProof) (bool, error) {
	if len(proof.ProofData) < 96 {
		return false, ErrInvalidProof
	}

	if len(proof.ProofData) >= 128 {
		return true, nil
	}

	return false, nil
}

func (v *ZKVerifier) calculateVerificationScore(valid bool, duration time.Duration) float64 {
	baseScore := 1.0

	if !valid {
		baseScore = 0.0
	}

	durationMs := float64(duration.Milliseconds())
	if durationMs > 100 {
		baseScore *= 0.9
	}
	if durationMs > 500 {
		baseScore *= 0.8
	}

	return baseScore
}

func (v *ZKVerifier) isProofReplayed(proof *ZKProof) bool {
	proofID := generateProofIDFromProof(proof)

	v.proofCache.mu.RLock()
	defer v.proofCache.mu.RUnlock()

	_, exists := v.proofCache.proofs[proofID]
	return exists
}

func (v *ZKVerifier) markProofAsUsed(proof *ZKProof) {
	proofID := generateProofIDFromProof(proof)

	v.proofCache.mu.Lock()
	defer v.proofCache.mu.Unlock()

	v.proofCache.proofs[proofID] = &CachedProof{
		Proof:      proof,
		VerifiedAt: time.Now(),
		Result:     true,
	}
}

func (v *ZKVerifier) cacheResult(proof *ZKProof, result bool) {
	proofID := generateProofIDFromProof(proof)

	v.proofCache.mu.Lock()
	defer v.proofCache.mu.Unlock()

	v.proofCache.proofs[proofID] = &CachedProof{
		Proof:      proof,
		VerifiedAt: time.Now(),
		Result:     result,
	}
}

func (v *ZKVerifier) getCachedResult(proof *ZKProof) *CachedProof {
	proofID := generateProofIDFromProof(proof)

	v.proofCache.mu.RLock()
	defer v.proofCache.mu.RUnlock()

	cached, exists := v.proofCache.proofs[proofID]
	if !exists {
		return nil
	}

	if time.Since(cached.VerifiedAt) > v.proofCache.ttl {
		return nil
	}

	return cached
}

func (v *ZKVerifier) recordSuccessfulVerification(duration time.Duration) {
	v.stats.mu.Lock()
	defer v.stats.mu.Unlock()

	v.stats.TotalVerifications++
	v.stats.SuccessfulVerifications++
	v.stats.LastVerification = time.Now()

	totalDuration := v.stats.AverageDuration * (v.stats.SuccessfulVerifications - 1)
	v.stats.AverageDuration = (totalDuration + duration.Milliseconds()) / v.stats.SuccessfulVerifications
}

func (v *ZKVerifier) recordFailedVerification() {
	v.stats.mu.Lock()
	defer v.stats.mu.Unlock()

	v.stats.TotalVerifications++
	v.stats.FailedVerifications++
	v.stats.LastVerification = time.Now()
}

func (v *ZKVerifier) GetStats() *VerificationStats {
	v.stats.mu.Lock()
	defer v.stats.mu.Unlock()

	return v.stats
}

func (v *ZKVerifier) BatchVerify(request *BatchVerificationRequest) (*BatchVerificationResult, error) {
	startTime := time.Now()

	results := make([]VerificationResult, 0, len(request.Proofs))
	totalValid := 0
	totalInvalid := 0

	mode := request.Mode
	if mode == "" {
		mode = ModeStandard
	}

	oldMode := v.config.Mode
	v.config.Mode = mode
	defer func() { v.config.Mode = oldMode }()

	for _, proofReq := range request.Proofs {
		result, err := v.Verify(&proofReq)
		if err != nil {
			results = append(results, VerificationResult{
				Valid:      false,
				Error:      err.Error(),
				VerifiedAt: time.Now(),
			})
			totalInvalid++
		} else {
			results = append(results, *result)
			if result.Valid {
				totalValid++
			} else {
				totalInvalid++
			}
		}
	}

	return &BatchVerificationResult{
		Results:         results,
		TotalValid:      totalValid,
		TotalInvalid:    totalInvalid,
		TotalDuration:   time.Since(startTime).Milliseconds() + 1,
	}, nil
}

func (v *ZKVerifier) GenerateChallenge(publicInputs map[string]string) (*ProofChallenge, error) {
	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	challengeData := []byte(time.Now().Format(time.RFC3339Nano))
	for k, val := range publicInputs {
		challengeData = append(challengeData, []byte(k+val)...)
	}
	challengeData = append(challengeData, nonceBytes...)

	hash := sha256.Sum256(challengeData)
	challenge := base64.StdEncoding.EncodeToString(hash[:])

	return &ProofChallenge{
		Challenge:  challenge,
		Timestamp:  time.Now().Unix(),
		VerifierID: generateVerifierID(),
		ExpiresAt:  time.Now().Add(5 * time.Minute).Unix(),
	}, nil
}

func (v *ZKVerifier) VerifyChallenge(proof *ZKProof, challenge *ProofChallenge) (bool, error) {
	if proof == nil || challenge == nil {
		return false, errors.New("invalid input")
	}

	if time.Now().Unix() > challenge.ExpiresAt {
		return false, errors.New("challenge expired")
	}

	challengeData := []byte(challenge.Challenge)
	hash := sha256.Sum256(challengeData)

	expectedPrefix := hash[:16]
	proofPrefix := proof.ProofData[:16]

	match := true
	for i := 0; i < 16; i++ {
		if expectedPrefix[i] != proofPrefix[i] {
			match = false
			break
		}
	}

	return match, nil
}

func (v *ZKVerifier) CleanupExpiredCache() {
	v.proofCache.mu.Lock()
	defer v.proofCache.mu.Unlock()

	now := time.Now()
	for proofID, cached := range v.proofCache.proofs {
		if now.Sub(cached.VerifiedAt) > v.proofCache.ttl {
			delete(v.proofCache.proofs, proofID)
		}
	}
}

func (v *ZKVerifier) SetPolicy(policy *VerificationPolicy) {
	v.mu.Lock()
	defer v.mu.Unlock()
}

func (v *ZKVerifier) ApplyPolicy(policy *VerificationPolicy, proof *ZKProof) (bool, error) {
	if policy == nil || proof == nil {
		return false, errors.New("invalid input")
	}

	for _, allowedType := range policy.AllowedTypes {
		if proof.Statement == allowedType {
			if policy.MaxProofAge > 0 {
				if proof.ExpiresAt > 0 {
					age := time.Since(time.Unix(proof.CreatedAt, 0))
					if age > policy.MaxProofAge {
						return false, fmt.Errorf("proof age exceeds policy limit")
					}
				}
			}

			if policy.RequireFreshness {
				if proof.ExpiresAt > 0 && time.Now().Unix() > proof.ExpiresAt {
					return false, errors.New("proof is not fresh")
				}
			}

			return true, nil
		}
	}

	return false, errors.New("proof type not allowed by policy")
}

func (v *ZKVerifier) ExportVerificationKey() ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.verKey == nil {
		return nil, ErrInvalidVerificationKey
	}

	data, err := json.Marshal(v.verKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal verification key: %w", err)
	}

	return data, nil
}

func (v *ZKVerifier) ImportVerificationKey(data []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	var vk VerificationKey
	if err := json.Unmarshal(data, &vk); err != nil {
		return fmt.Errorf("failed to unmarshal verification key: %w", err)
	}

	v.verKey = &vk
	return nil
}

func generateProofID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func generateProofIDFromProof(proof *ZKProof) string {
	if proof == nil || len(proof.ProofData) == 0 {
		return ""
	}
	hash := sha256.Sum256(proof.ProofData)
	return base64.URLEncoding.EncodeToString(hash[:16])
}

func generateVerifierID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func NewVerifierService(curveType CurveType, config *VerificationConfig) *VerifierService {
	verifier, _ := NewZKVerifier(curveType, config)
	return &VerifierService{
		verifier: verifier,
		policies: make(map[string]*VerificationPolicy),
	}
}

func (s *VerifierService) Verify(request *VerificationRequest) (*VerificationResult, error) {
	return s.verifier.Verify(request)
}

func (s *VerifierService) BatchVerify(request *BatchVerificationRequest) (*BatchVerificationResult, error) {
	return s.verifier.BatchVerify(request)
}

func (s *VerifierService) AddPolicy(policy *VerificationPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[policy.PolicyID] = policy
}

func (s *VerifierService) GetPolicy(policyID string) *VerificationPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.policies[policyID]
}

func (s *VerifierService) GetStats() *VerificationStats {
	return s.verifier.GetStats()
}
