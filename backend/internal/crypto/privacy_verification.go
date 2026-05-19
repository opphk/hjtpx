package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrInvalidPrivacyRequest  = errors.New("invalid privacy request")
	ErrDataLeakageDetected    = errors.New("data leakage detected")
	ErrBudgetExceeded         = errors.New("privacy budget exceeded")
	ErrInvalidDataFormat      = errors.New("invalid data format")
	ErrConsentRequired        = errors.New("user consent required")
)

type PrivacyLevel int

const (
	PrivacyLevelNone    PrivacyLevel = 0
	PrivacyLevelBasic   PrivacyLevel = 1
	PrivacyLevelMedium  PrivacyLevel = 2
	PrivacyLevelHigh    PrivacyLevel = 3
	PrivacyLevelMaximum PrivacyLevel = 4
)

type ConsentType string

const (
	ConsentDataCollection  ConsentType = "data_collection"
	ConsentDataProcessing   ConsentType = "data_processing"
	ConsentDataSharing      ConsentType = "data_sharing"
	ConsentDataRetention    ConsentType = "data_retention"
	ConsentThirdParty       ConsentType = "third_party"
)

type ConsentRecord struct {
	UserID       string                 `json:"user_id"`
	ConsentType  ConsentType            `json:"consent_type"`
	Granted      bool                   `json:"granted"`
	GrantedAt    time.Time              `json:"granted_at"`
	RevokedAt    *time.Time             `json:"revoked_at,omitempty"`
	Version      string                 `json:"version"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Metadata     map[string]interface{}  `json:"metadata,omitempty"`
}

type PrivacyPolicy struct {
	PolicyID        string                 `json:"policy_id"`
	UserID          string                 `json:"user_id"`
	DataCategories  []string               `json:"data_categories"`
	RetentionPeriod time.Duration          `json:"retention_period"`
	SharingAllowed  bool                   `json:"sharing_allowed"`
	ThirdParties    []string               `json:"third_parties,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Version         string                 `json:"version"`
}

type DataSubject struct {
	SubjectID    string                 `json:"subject_id"`
	Pseudonym    string                 `json:"pseudonym"`
	PrivacyLevel PrivacyLevel          `json:"privacy_level"`
	Consents     map[ConsentType]bool   `json:"consents"`
	CreatedAt    time.Time              `json:"created_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type PrivacyVerificationRequest struct {
	RequestID      string                 `json:"request_id"`
	DataSubject    *DataSubject            `json:"data_subject"`
	DataToVerify   map[string]interface{} `json:"data_to_verify"`
	StatementType  StatementType          `json:"statement_type"`
	PrivacyLevel   PrivacyLevel           `json:"privacy_level"`
	ProofRequest   *SNARKProofRequest     `json:"proof_request,omitempty"`
	Timestamp      int64                  `json:"timestamp"`
	SessionID      string                 `json:"session_id"`
	ClientNonce    string                 `json:"client_nonce,omitempty"`
}

type PrivacyVerificationResult struct {
	RequestID     string                 `json:"request_id"`
	Valid         bool                   `json:"valid"`
	ProofValid    bool                   `json:"proof_valid"`
	ConsentValid  bool                   `json:"consent_valid"`
	PolicyValid   bool                   `json:"policy_valid"`
	PrivacyPreserved bool                `json:"privacy_preserved"`
	Error         string                 `json:"error,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
	VerifiedAt    time.Time              `json:"verified_at"`
	Duration      int64                  `json:"duration_ms"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type PrivacyBudget struct {
	UserID          string    `json:"user_id"`
	TotalBudget     int64     `json:"total_budget"`
	UsedBudget      int64     `json:"used_budget"`
	RemainingBudget int64     `json:"remaining_budget"`
	ResetAt         time.Time `json:"reset_at"`
	LastResetAt     time.Time `json:"last_reset_at"`
	Transactions    int64     `json:"transactions"`
}

type PrivacyVerificationService struct {
	mu          sync.RWMutex
	budgets     map[string]*PrivacyBudget
	policies    map[string]*PrivacyPolicy
	consents    map[string]*ConsentRecord
	subjects    map[string]*DataSubject
	config      *PrivacyConfig
	stats       *PrivacyStats
}

type PrivacyConfig struct {
	DefaultPrivacyLevel   PrivacyLevel
	MaxBudget             int64
	BudgetResetPeriod     time.Duration
	RequireConsent        bool
	EnableBudgetTracking  bool
	EnableAuditLog        bool
	StrictMode            bool
}

type PrivacyStats struct {
	TotalVerifications int64     `json:"total_verifications"`
	SuccessfulVerifs   int64     `json:"successful_verifications"`
	FailedVerifs       int64     `json:"failed_verifications"`
	BudgetExceeded     int64     `json:"budget_exceeded"`
	DataLeakageAttempts int64    `json:"data_leakage_attempts"`
	LastVerification   time.Time `json:"last_verification"`
	mu                 sync.Mutex
}

type DataRedactionRule struct {
	RuleID       string       `json:"rule_id"`
	FieldName    string       `json:"field_name"`
	RedactionType RedactionType `json:"redaction_type"`
	Pattern      string       `json:"pattern,omitempty"`
	Replacement  string       `json:"replacement,omitempty"`
}

type RedactionType string

const (
	RedactFull    RedactionType = "full"
	RedactPartial RedactionType = "partial"
	RedactHash    RedactionType = "hash"
	RedactMask    RedactionType = "mask"
)

type DataMaskingRule struct {
	RuleID      string       `json:"rule_id"`
	FieldName   string       `json:"field_name"`
	MaskType    MaskType     `json:"mask_type"`
	VisibleChars int         `json:"visible_chars"`
	MaskChar    string       `json:"mask_char"`
	Regex       string       `json:"regex,omitempty"`
}

type MaskType string

const (
	MaskTypeCreditCard  MaskType = "credit_card"
	MaskTypeEmail       MaskType = "email"
	MaskTypePhone       MaskType = "phone"
	MaskTypeSSN         MaskType = "ssn"
	MaskTypeCustom      MaskType = "custom"
)

type PrivacyAuditLog struct {
	LogID        string    `json:"log_id"`
	RequestID    string    `json:"request_id"`
	UserID       string    `json:"user_id"`
	Action       string    `json:"action"`
	Result       string    `json:"result"`
	PrivacyLevel PrivacyLevel `json:"privacy_level"`
	DataCategories []string `json:"data_categories"`
	Timestamp    time.Time `json:"timestamp"`
	IPAddress    string    `json:"ip_address,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

func NewPrivacyVerificationService(config *PrivacyConfig) *PrivacyVerificationService {
	if config == nil {
		config = &PrivacyConfig{
			DefaultPrivacyLevel:  PrivacyLevelMedium,
			MaxBudget:            1000,
			BudgetResetPeriod:    24 * time.Hour,
			RequireConsent:       true,
			EnableBudgetTracking: true,
			EnableAuditLog:       true,
			StrictMode:           false,
		}
	}

	return &PrivacyVerificationService{
		budgets:   make(map[string]*PrivacyBudget),
		policies:  make(map[string]*PrivacyPolicy),
		consents:  make(map[string]*ConsentRecord),
		subjects:  make(map[string]*DataSubject),
		config:    config,
		stats:     &PrivacyStats{},
	}
}

func (s *PrivacyVerificationService) Verify(request *PrivacyVerificationRequest) (*PrivacyVerificationResult, error) {
	startTime := time.Now()

	result := &PrivacyVerificationResult{
		VerifiedAt: startTime,
		Warnings:   make([]string, 0),
	}

	if request == nil {
		result.Valid = false
		result.Error = ErrInvalidPrivacyRequest.Error()
		result.Duration = time.Since(startTime).Milliseconds()
		s.recordFailedVerification()
		return result, ErrInvalidPrivacyRequest
	}

	result.RequestID = request.RequestID

	if err := s.validateRequest(request); err != nil {
		result.Valid = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime).Milliseconds()
		s.recordFailedVerification()
		return result, err
	}

	if s.config.RequireConsent {
		consentValid := s.validateConsents(request)
		result.ConsentValid = consentValid
		if !consentValid {
			result.Warnings = append(result.Warnings, "user consent not fully granted")
			if s.config.StrictMode {
				result.Valid = false
				result.Error = ErrConsentRequired.Error()
				result.Duration = time.Since(startTime).Milliseconds()
				s.recordFailedVerification()
				return result, ErrConsentRequired
			}
		}
	}

	if s.config.EnableBudgetTracking {
		budgetValid := s.validateBudget(request.DataSubject.SubjectID)
		if !budgetValid {
			result.Valid = false
			result.Error = ErrBudgetExceeded.Error()
			result.Duration = time.Since(startTime).Milliseconds()
			s.recordFailedVerification()
			s.stats.mu.Lock()
			s.stats.BudgetExceeded++
			s.stats.mu.Unlock()
			return result, ErrBudgetExceeded
		}
	}

	if request.ProofRequest != nil {
		proofValid, err := s.verifyProof(request)
		result.ProofValid = proofValid
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("proof verification issue: %s", err.Error()))
		}
	} else {
		result.ProofValid = true
	}

	dataLeakage := s.checkDataLeakage(request)
	if dataLeakage {
		result.Valid = false
		result.Error = ErrDataLeakageDetected.Error()
		result.PrivacyPreserved = false
		result.Duration = time.Since(startTime).Milliseconds()
		s.recordFailedVerification()
		s.stats.mu.Lock()
		s.stats.DataLeakageAttempts++
		s.stats.mu.Unlock()
		return result, ErrDataLeakageDetected
	}

	policyValid := s.validatePolicy(request)
	result.PolicyValid = policyValid

	result.PrivacyPreserved = s.verifyPrivacyPreservation(request)

	if s.config.EnableBudgetTracking {
		s.updateBudget(request.DataSubject.SubjectID)
	}

	result.Valid = result.ProofValid && result.ConsentValid && result.PolicyValid
	result.Duration = time.Since(startTime).Milliseconds()

	if result.Valid {
		s.recordSuccessfulVerification()
	} else {
		s.recordFailedVerification()
	}

	if s.config.EnableAuditLog {
		s.logVerification(request, result)
	}

	return result, nil
}

func (s *PrivacyVerificationService) validateRequest(request *PrivacyVerificationRequest) error {
	if request == nil {
		return ErrInvalidPrivacyRequest
	}

	if request.DataSubject == nil {
		return fmt.Errorf("%w: missing data subject", ErrInvalidPrivacyRequest)
	}

	if request.DataSubject.SubjectID == "" {
		return fmt.Errorf("%w: missing subject ID", ErrInvalidPrivacyRequest)
	}

	if request.DataToVerify == nil || len(request.DataToVerify) == 0 {
		return fmt.Errorf("%w: missing data to verify", ErrInvalidPrivacyRequest)
	}

	if request.StatementType == "" {
		return fmt.Errorf("%w: missing statement type", ErrInvalidPrivacyRequest)
	}

	return nil
}

func (s *PrivacyVerificationService) validateConsents(request *PrivacyVerificationRequest) bool {
	subject := request.DataSubject

	if len(subject.Consents) == 0 {
		if s.config.RequireConsent {
			return false
		}
		return true
	}

	requiredConsents := []ConsentType{
		ConsentDataCollection,
		ConsentDataProcessing,
	}

	for _, consentType := range requiredConsents {
		if granted, ok := subject.Consents[consentType]; !ok || !granted {
			return false
		}
	}

	return true
}

func (s *PrivacyVerificationService) validateBudget(userID string) bool {
	if !s.config.EnableBudgetTracking {
		return true
	}

	budget, exists := s.budgets[userID]
	if !exists {
		return true
	}

	if budget.RemainingBudget <= 0 {
		return false
	}

	if time.Now().After(budget.ResetAt) {
		s.resetBudget(userID)
	}

	budget, _ = s.budgets[userID]
	return budget.RemainingBudget > 0
}

func (s *PrivacyVerificationService) updateBudget(userID string) {
	if !s.config.EnableBudgetTracking {
		return
	}

	budget, exists := s.budgets[userID]
	if !exists {
		budget = &PrivacyBudget{
			UserID:       userID,
			TotalBudget:  s.config.MaxBudget,
			UsedBudget:   0,
			ResetAt:      time.Now().Add(s.config.BudgetResetPeriod),
			LastResetAt:  time.Now(),
		}
		s.budgets[userID] = budget
	}

	budget.UsedBudget++
	budget.RemainingBudget = budget.TotalBudget - budget.UsedBudget
	budget.Transactions++
}

func (s *PrivacyVerificationService) resetBudget(userID string) {
	if budget, exists := s.budgets[userID]; exists {
		budget.UsedBudget = 0
		budget.RemainingBudget = budget.TotalBudget
		budget.LastResetAt = time.Now()
		budget.ResetAt = time.Now().Add(s.config.BudgetResetPeriod)
	}
}

func (s *PrivacyVerificationService) verifyProof(request *PrivacyVerificationRequest) (bool, error) {
	if request.ProofRequest == nil {
		return true, nil
	}

	snarkService := NewZKSNARKService(request.DataSubject.PrivacyLevel.GetCurveType())
	circuit := snarkService.CreateKnowledgeProofCircuit("predicate")

	circuitInput := &CircuitInput{
		Public: make([]string, 0),
		Secret: make([]string, 0),
	}

	for k, v := range request.DataToVerify {
		circuitInput.Public = append(circuitInput.Public, k)
		circuitInput.Secret = append(circuitInput.Secret, fmt.Sprintf("%v", v))
	}

	if err := snarkService.Setup(circuit); err != nil {
		return false, err
	}

	publicInputs := make(map[string]interface{})
	for _, key := range circuitInput.Public {
		if val, ok := request.DataToVerify[key]; ok {
			publicInputs[key] = val
		}
	}

	witness := make(map[string]interface{})
	for i, key := range circuitInput.Secret {
		if i < len(circuitInput.Secret) {
			witness[key] = circuitInput.Secret[i]
		}
	}

	response, err := snarkService.GenerateProof(&SNARKProofRequest{
		Witness:      witness,
		PublicInputs: publicInputs,
		Protocol:     "G16",
	})

	if err != nil {
		return false, err
	}

	if response == nil || response.Proof == nil {
		return false, errors.New("proof generation failed")
	}

	return true, nil
}

func (s *PrivacyVerificationService) checkDataLeakage(request *PrivacyVerificationRequest) bool {
	sensitiveFields := []string{"password", "secret", "token", "key", "credential", "ssn", "credit_card"}

	for fieldName := range request.DataToVerify {
		for _, sensitive := range sensitiveFields {
			if containsSensitive(fieldName, sensitive) {
				return true
			}
		}
	}

	return false
}

func containsSensitive(fieldName, sensitive string) bool {
	fieldLower := toLower(fieldName)
	sensitiveLower := toLower(sensitive)
	return len(fieldLower) >= len(sensitiveLower) && containsSubstring(fieldLower, sensitiveLower)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func (s *PrivacyVerificationService) validatePolicy(request *PrivacyVerificationRequest) bool {
	if request.DataSubject == nil {
		return true
	}

	return request.DataSubject.PrivacyLevel >= s.config.DefaultPrivacyLevel
}

func (s *PrivacyVerificationService) verifyPrivacyPreservation(request *PrivacyVerificationRequest) bool {
	if request.DataSubject == nil {
		return false
	}

	switch request.DataSubject.PrivacyLevel {
	case PrivacyLevelMaximum:
		return s.verifyMaximumPrivacy(request)
	case PrivacyLevelHigh:
		return s.verifyHighPrivacy(request)
	case PrivacyLevelMedium:
		return s.verifyMediumPrivacy(request)
	case PrivacyLevelBasic:
		return s.verifyBasicPrivacy(request)
	default:
		return true
	}
}

func (s *PrivacyVerificationService) verifyMaximumPrivacy(request *PrivacyVerificationRequest) bool {
	return true
}

func (s *PrivacyVerificationService) verifyHighPrivacy(request *PrivacyVerificationRequest) bool {
	return true
}

func (s *PrivacyVerificationService) verifyMediumPrivacy(request *PrivacyVerificationRequest) bool {
	return true
}

func (s *PrivacyVerificationService) verifyBasicPrivacy(request *PrivacyVerificationRequest) bool {
	return true
}

func (s *PrivacyVerificationService) recordSuccessfulVerification() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	s.stats.TotalVerifications++
	s.stats.SuccessfulVerifs++
	s.stats.LastVerification = time.Now()
}

func (s *PrivacyVerificationService) recordFailedVerification() {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	s.stats.TotalVerifications++
	s.stats.FailedVerifs++
	s.stats.LastVerification = time.Now()
}

func (s *PrivacyVerificationService) logVerification(request *PrivacyVerificationRequest, result *PrivacyVerificationResult) {
	log := &PrivacyAuditLog{
		LogID:          generateLogID(),
		RequestID:      request.RequestID,
		UserID:         request.DataSubject.SubjectID,
		Action:         "privacy_verification",
		Result:         fmt.Sprintf("%v", result.Valid),
		PrivacyLevel:   request.PrivacyLevel,
		DataCategories: []string{},
		Timestamp:      time.Now(),
	}

	if result.Valid {
		log.Result = "success"
	} else {
		log.Result = "failed"
	}

	_ = log
}

func (s *PrivacyVerificationService) RecordConsent(consent *ConsentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if consent == nil || consent.UserID == "" {
		return errors.New("invalid consent record")
	}

	key := fmt.Sprintf("%s:%s", consent.UserID, consent.ConsentType)
	s.consents[key] = consent

	return nil
}

func (s *PrivacyVerificationService) GetConsent(userID string, consentType ConsentType) *ConsentRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", userID, consentType)
	consent, exists := s.consents[key]
	if !exists {
		return nil
	}

	if consent.RevokedAt != nil {
		return nil
	}

	return consent
}

func (s *PrivacyVerificationService) RevokeConsent(userID string, consentType ConsentType) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", userID, consentType)
	consent, exists := s.consents[key]
	if !exists {
		return errors.New("consent not found")
	}

	now := time.Now()
	consent.RevokedAt = &now

	return nil
}

func (s *PrivacyVerificationService) CreateDataSubject(subjectID string, privacyLevel PrivacyLevel) *DataSubject {
	s.mu.Lock()
	defer s.mu.Unlock()

	subject := &DataSubject{
		SubjectID:    subjectID,
		Pseudonym:    generatePseudonym(subjectID),
		PrivacyLevel: privacyLevel,
		Consents:     make(map[ConsentType]bool),
		CreatedAt:    time.Now(),
	}

	s.subjects[subjectID] = subject

	return subject
}

func (s *PrivacyVerificationService) GetDataSubject(subjectID string) *DataSubject {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.subjects[subjectID]
}

func (s *PrivacyVerificationService) SetPrivacyLevel(subjectID string, level PrivacyLevel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	subject, exists := s.subjects[subjectID]
	if !exists {
		return errors.New("data subject not found")
	}

	subject.PrivacyLevel = level
	return nil
}

func (s *PrivacyVerificationService) GetBudget(userID string) *PrivacyBudget {
	s.mu.RLock()
	defer s.mu.RUnlock()

	budget, exists := s.budgets[userID]
	if !exists {
		return &PrivacyBudget{
			UserID:          userID,
			TotalBudget:     s.config.MaxBudget,
			UsedBudget:      0,
			RemainingBudget: s.config.MaxBudget,
			ResetAt:         time.Now().Add(s.config.BudgetResetPeriod),
			LastResetAt:     time.Now(),
		}
	}

	return budget
}

func (s *PrivacyVerificationService) ResetBudget(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resetBudget(userID)
}

func (s *PrivacyVerificationService) GetStats() *PrivacyStats {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	return s.stats
}

func (s *PrivacyVerificationService) ExportAuditLogs(start, end time.Time) ([]*PrivacyAuditLog, error) {
	return make([]*PrivacyAuditLog, 0), nil
}

func (level PrivacyLevel) GetCurveType() CurveType {
	switch level {
	case PrivacyLevelMaximum, PrivacyLevelHigh:
		return CurveP521
	case PrivacyLevelMedium:
		return CurveP384
	default:
		return CurveP256
	}
}

func (level PrivacyLevel) String() string {
	switch level {
	case PrivacyLevelNone:
		return "none"
	case PrivacyLevelBasic:
		return "basic"
	case PrivacyLevelMedium:
		return "medium"
	case PrivacyLevelHigh:
		return "high"
	case PrivacyLevelMaximum:
		return "maximum"
	default:
		return "unknown"
	}
}

func generateLogID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func generatePseudonym(subjectID string) string {
	hash := sha256.Sum256([]byte(subjectID + time.Now().Format("2006-01-02")))
	return base64.URLEncoding.EncodeToString(hash[:16])
}

func (s *PrivacyVerificationService) BatchVerify(requests []*PrivacyVerificationRequest) ([]*PrivacyVerificationResult, error) {
	results := make([]*PrivacyVerificationResult, 0, len(requests))

	for _, request := range requests {
		result, err := s.Verify(request)
		if err != nil {
			results = append(results, &PrivacyVerificationResult{
				RequestID: request.RequestID,
				Valid:     false,
				Error:     err.Error(),
				VerifiedAt: time.Now(),
			})
		} else {
			results = append(results, result)
		}
	}

	return results, nil
}

func (s *PrivacyVerificationService) CreatePolicy(policy *PrivacyPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if policy == nil || policy.UserID == "" {
		return errors.New("invalid policy")
	}

	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	policy.Version = "1.0"

	s.policies[policy.PolicyID] = policy
	return nil
}

func (s *PrivacyVerificationService) GetPolicy(policyID string) *PrivacyPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.policies[policyID]
}

func (s *PrivacyVerificationService) UpdatePolicy(policy *PrivacyPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.policies[policy.PolicyID]
	if !exists {
		return errors.New("policy not found")
	}

	policy.CreatedAt = existing.CreatedAt
	policy.UpdatedAt = time.Now()

	parts := parseVersion(existing.Version)
	parts[2]++
	policy.Version = formatVersion(parts)

	s.policies[policy.PolicyID] = policy
	return nil
}

func parseVersion(v string) [3]int {
	var parts [3]int
	fmt.Sscanf(v, "%d.%d.%d", &parts[0], &parts[1], &parts[2])
	return parts
}

func formatVersion(parts [3]int) string {
	return fmt.Sprintf("%d.%d.%d", parts[0], parts[1], parts[2])
}
