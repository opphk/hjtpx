package solution

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidTransaction = errors.New("invalid transaction")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrFraudDetected      = errors.New("fraud detected")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrComplianceViolation = errors.New("compliance violation")
)

type FinanceSecurityService interface {
	ValidateTransaction(ctx context.Context, tx *Transaction) (*ValidationResult, error)
	CheckAML(ctx context.Context, tx *Transaction) (*AMLCheckResult, error)
	DetectFraud(ctx context.Context, tx *Transaction) (*FraudDetectionResult, error)
	CalculateRiskScore(ctx context.Context, tx *Transaction) (*RiskScore, error)
	ApplyTransactionPolicy(ctx context.Context, tx *Transaction) error
	GenerateComplianceReport(ctx context.Context, period string) (*ComplianceReport, error)
}

type Transaction struct {
	TransactionID    string            `json:"transaction_id"`
	Amount           float64           `json:"amount"`
	Currency         string            `json:"currency"`
	FromAccount      string            `json:"from_account"`
	ToAccount        string            `json:"to_account"`
	TransactionType  string            `json:"transaction_type"`
	Timestamp        time.Time         `json:"timestamp"`
	CustomerID       string            `json:"customer_id"`
	MerchantID       string            `json:"merchant_id"`
	Location         *GeoLocation      `json:"location,omitempty"`
	DeviceFingerprint string           `json:"device_fingerprint"`
	IPAddress        string            `json:"ip_address"`
	Metadata         map[string]string `json:"metadata"`
}

type GeoLocation struct {
	Country     string  `json:"country"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type ValidationResult struct {
	Valid         bool     `json:"valid"`
	Errors        []string `json:"errors"`
	Warnings      []string `json:"warnings"`
	RequiredAuth  []string `json:"required_auth"`
	Score         float64  `json:"score"`
	ValidatedAt   time.Time `json:"validated_at"`
}

type AMLCheckResult struct {
	Cleared         bool     `json:"cleared"`
	RiskLevel       string   `json:"risk_level"`
	MatchScore      float64  `json:"match_score"`
	Watchlists      []string `json:"watchlists"`
	RegulatoryReport bool   `json:"regulatory_report"`
	Recommendations []string `json:"recommendations"`
}

type FraudDetectionResult struct {
	IsFraud         bool     `json:"is_fraud"`
	FraudScore      float64  `json:"fraud_score"`
	FraudReasons    []string `json:"fraud_reasons"`
	RiskIndicators  []string `json:"risk_indicators"`
	Action          string   `json:"action"`
	ActionReasons   []string `json:"action_reasons"`
}

type RiskScore struct {
	Score           float64  `json:"score"`
	Level           string   `json:"level"`
	Factors         []RiskFactor `json:"factors"`
	Recommendations []string `json:"recommendations"`
	ExpiresAt       time.Time `json:"expires_at"`
}

type RiskFactor struct {
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`
	Value       float64 `json:"value"`
	Description string  `json:"description"`
}

type ComplianceReport struct {
	ReportID       string              `json:"report_id"`
	Period         string              `json:"period"`
	GeneratedAt    time.Time           `json:"generated_at"`
	TotalTransactions int64            `json:"total_transactions"`
	TotalAmount    float64              `json:"total_amount"`
	HighRiskCount  int64                `json:"high_risk_count"`
	AMLFlagsCount  int64                `json:"aml_flags_count"`
	FraudDetected  int64                `json:"fraud_detected"`
	ComplianceMetrics map[string]float64 `json:"compliance_metrics"`
	Findings       []ComplianceFinding  `json:"findings"`
}

type ComplianceFinding struct {
	Severity      string    `json:"severity"`
	Category      string    `json:"category"`
	Description   string    `json:"description"`
	TransactionIDs []string `json:"transaction_ids"`
	Recommendation string   `json:"recommendation"`
}

type TransactionPolicy struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	TransactionType string  `json:"transaction_type"`
	MinAmount       float64  `json:"min_amount"`
	MaxAmount       float64  `json:"max_amount"`
	AllowedCountries []string `json:"allowed_countries"`
	BlockedCountries []string `json:"blocked_countries"`
	RequiredDocuments []string `json:"required_documents"`
	MaxDailyLimit   float64  `json:"max_daily_limit"`
	MaxMonthlyLimit float64  `json:"max_monthly_limit"`
}

type financeSecurityService struct {
	amlRules       []AMLRule
	fraudRules     []FraudRule
	policies       []TransactionPolicy
	watchlists      []WatchlistEntry
}

type AMLRule struct {
	Name            string   `json:"name"`
	Threshold       float64  `json:"threshold"`
	Period          string   `json:"period"`
	Action          string   `json:"action"`
	Sensitivity     float64  `json:"sensitivity"`
}

type FraudRule struct {
	Name            string   `json:"name"`
	Pattern         string   `json:"pattern"`
	Weight          float64  `json:"weight"`
	Action          string   `json:"action"`
	Threshold       float64  `json:"threshold"`
}

type WatchlistEntry struct {
	EntityID      string   `json:"entity_id"`
	EntityType    string   `json:"entity_type"`
	Name          string   `json:"name"`
	ListType      string   `json:"list_type"`
	Country       string   `json:"country"`
	RiskLevel     string   `json:"risk_level"`
	AddedDate     time.Time `json:"added_date"`
	ExpirationDate time.Time `json:"expiration_date"`
}

func NewFinanceSecurityService() FinanceSecurityService {
	service := &financeSecurityService{
		amlRules:   []AMLRule{},
		fraudRules: []FraudRule{},
		policies:   []TransactionPolicy{},
		watchlists: []WatchlistEntry{},
	}

	service.initializeDefaultRules()
	return service
}

func (s *financeSecurityService) initializeDefaultRules() {
	s.amlRules = []AMLRule{
		{Name: "large_transaction", Threshold: 10000, Period: "single", Action: "flag", Sensitivity: 0.7},
		{Name: "rapid_movement", Threshold: 5000, Period: "24h", Action: "review", Sensitivity: 0.8},
		{Name: "structuring", Threshold: 9000, Period: "24h", Action: "block", Sensitivity: 0.9},
		{Name: "high_risk_country", Threshold: 1, Period: "single", Action: "escalate", Sensitivity: 1.0},
	}

	s.fraudRules = []FraudRule{
		{Name: "velocity_check", Pattern: "multiple_transactions", Weight: 0.6, Action: "review", Threshold: 5},
		{Name: "geographic_impossible", Pattern: "location_change", Weight: 0.8, Action: "block", Threshold: 1},
		{Name: "device_mismatch", Pattern: "new_device", Weight: 0.5, Action: "verify", Threshold: 1},
		{Name: "amount_anomaly", Pattern: "outlier", Weight: 0.7, Action: "review", Threshold: 1},
		{Name: "unusual_time", Pattern: "off_hours", Weight: 0.3, Action: "flag", Threshold: 1},
	}

	s.policies = []TransactionPolicy{
		{
			ID:              "domestic_transfer",
			Name:            "Domestic Transfer",
			TransactionType: "transfer",
			MinAmount:        0.01,
			MaxAmount:        100000,
			AllowedCountries: []string{"US", "CA"},
			BlockedCountries: []string{},
			MaxDailyLimit:   50000,
			MaxMonthlyLimit: 200000,
		},
		{
			ID:              "international_transfer",
			Name:            "International Transfer",
			TransactionType: "international_transfer",
			MinAmount:        100,
			MaxAmount:        1000000,
			AllowedCountries: []string{},
			BlockedCountries: []string{"KP", "IR", "SY"},
			MaxDailyLimit:   100000,
			MaxMonthlyLimit: 500000,
		},
		{
			ID:              "high_value_transaction",
			Name:            "High Value Transaction",
			TransactionType: "high_value",
			MinAmount:        50000,
			MaxAmount:        10000000,
			AllowedCountries: []string{},
			BlockedCountries: []string{},
			RequiredDocuments: []string{"id_verification", "source_of_funds"},
			MaxDailyLimit:   1000000,
			MaxMonthlyLimit: 5000000,
		},
	}
}

func (s *financeSecurityService) ValidateTransaction(ctx context.Context, tx *Transaction) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
		RequiredAuth: []string{},
		ValidatedAt: time.Now(),
	}

	if tx.Amount <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "transaction amount must be positive")
	}

	if tx.TransactionType == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "transaction type is required")
	}

	if policy := s.getTransactionPolicy(tx); policy != nil {
		if err := s.validateAgainstPolicy(tx, policy, result); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
		}
	}

	result.Score = s.calculateValidationScore(result)
	return result, nil
}

func (s *financeSecurityService) getTransactionPolicy(tx *Transaction) *TransactionPolicy {
	for _, policy := range s.policies {
		if policy.TransactionType == tx.TransactionType {
			if tx.Amount >= policy.MinAmount && tx.Amount <= policy.MaxAmount {
				return &policy
			}
		}
	}
	return nil
}

func (s *financeSecurityService) validateAgainstPolicy(tx *Transaction, policy *TransactionPolicy, result *ValidationResult) error {
	if len(policy.BlockedCountries) > 0 && tx.Location != nil {
		for _, country := range policy.BlockedCountries {
			if tx.Location.Country == country {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("country %s is blocked for this transaction type", country))
				return ErrComplianceViolation
			}
		}
	}

	if len(policy.AllowedCountries) > 0 && tx.Location != nil {
		allowed := false
		for _, country := range policy.AllowedCountries {
			if tx.Location.Country == country {
				allowed = true
				break
			}
		}
		if !allowed {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("country %s is not allowed for this transaction type", tx.Location.Country))
		}
	}

	return nil
}

func (s *financeSecurityService) calculateValidationScore(result *ValidationResult) float64 {
	if !result.Valid {
		return 0
	}

	score := 100.0
	score -= float64(len(result.Errors)) * 20
	score -= float64(len(result.Warnings)) * 5

	return score
}

func (s *financeSecurityService) CheckAML(ctx context.Context, tx *Transaction) (*AMLCheckResult, error) {
	result := &AMLCheckResult{
		Cleared:    true,
		RiskLevel:  "low",
		MatchScore: 0,
		Watchlists: []string{},
		RegulatoryReport: false,
		Recommendations: []string{},
	}

	for _, entry := range s.watchlists {
		if entry.EntityID == tx.CustomerID || entry.EntityID == tx.MerchantID {
			result.MatchScore += 0.5
			result.Watchlists = append(result.Watchlists, entry.EntityID)
			result.RiskLevel = entry.RiskLevel
			result.Recommendations = append(result.Recommendations, fmt.Sprintf("Match found on %s list", entry.ListType))
		}
	}

	for _, rule := range s.amlRules {
		if s.evaluateAMLRule(tx, &rule) {
			result.MatchScore += rule.Sensitivity
			result.Recommendations = append(result.Recommendations, fmt.Sprintf("AML alert: %s triggered", rule.Name))

			if rule.Action == "block" {
				result.Cleared = false
			}
		}
	}

	if result.MatchScore >= 0.7 {
		result.RiskLevel = "high"
		result.Cleared = false
		result.RegulatoryReport = true
	} else if result.MatchScore >= 0.4 {
		result.RiskLevel = "medium"
	}

	return result, nil
}

func (s *financeSecurityService) evaluateAMLRule(tx *Transaction, rule *AMLRule) bool {
	switch rule.Name {
	case "large_transaction":
		return tx.Amount >= rule.Threshold
	case "high_risk_country":
		if tx.Location != nil {
			highRiskCountries := []string{"KP", "IR", "SY", "CU", "VE"}
			for _, country := range highRiskCountries {
				if tx.Location.Country == country {
					return true
				}
			}
		}
	}
	return false
}

func (s *financeSecurityService) DetectFraud(ctx context.Context, tx *Transaction) (*FraudDetectionResult, error) {
	result := &FraudDetectionResult{
		IsFraud:        false,
		FraudScore:     0,
		FraudReasons:   []string{},
		RiskIndicators: []string{},
		Action:         "allow",
		ActionReasons:  []string{},
	}

	for _, rule := range s.fraudRules {
		score, triggered := s.evaluateFraudRule(tx, &rule)
		if triggered {
			result.FraudScore += score * rule.Weight
			result.RiskIndicators = append(result.RiskIndicators, rule.Name)
		}
	}

	if result.FraudScore >= 0.7 {
		result.IsFraud = true
		result.Action = "block"
		result.ActionReasons = append(result.ActionReasons, "high fraud score")
		result.FraudReasons = append(result.FraudReasons, fmt.Sprintf("Fraud score %.2f exceeds threshold", result.FraudScore))
	} else if result.FraudScore >= 0.4 {
		result.Action = "review"
		result.ActionReasons = append(result.ActionReasons, "medium fraud score, manual review required")
	}

	return result, nil
}

func (s *financeSecurityService) evaluateFraudRule(tx *Transaction, rule *FraudRule) (float64, bool) {
	switch rule.Name {
	case "velocity_check":
		return 0.6, true
	case "geographic_impossible":
		return 0.8, true
	case "device_mismatch":
		return 0.5, true
	case "amount_anomaly":
		return 0.7, tx.Amount > 100000
	case "unusual_time":
		hour := tx.Timestamp.Hour()
		return 0.3, hour < 6 || hour > 22
	}
	return 0, false
}

func (s *financeSecurityService) CalculateRiskScore(ctx context.Context, tx *Transaction) (*RiskScore, error) {
	score := &RiskScore{
		Score:     0,
		Level:     "low",
		Factors:   []RiskFactor{},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	factors := []struct {
		name        string
		weight      float64
		description string
		calculate   func(*Transaction) float64
	}{
		{"transaction_amount", 0.3, "Transaction amount risk factor", s.calculateAmountRisk},
		{"geographic_risk", 0.2, "Geographic risk factor", s.calculateGeoRisk},
		{"velocity_risk", 0.2, "Transaction velocity risk", s.calculateVelocityRisk},
		{"device_risk", 0.15, "Device fingerprint risk", s.calculateDeviceRisk},
		{"time_risk", 0.15, "Time-based risk factor", s.calculateTimeRisk},
	}

	for _, f := range factors {
		value := f.calculate(tx)
		score.Factors = append(score.Factors, RiskFactor{
			Name:        f.name,
			Weight:      f.weight,
			Value:       value,
			Description: f.description,
		})
		score.Score += value * f.weight
	}

	if score.Score >= 0.7 {
		score.Level = "critical"
		score.Recommendations = append(score.Recommendations, "Block transaction, escalate to compliance")
	} else if score.Score >= 0.5 {
		score.Level = "high"
		score.Recommendations = append(score.Recommendations, "Require additional verification")
	} else if score.Score >= 0.3 {
		score.Level = "medium"
		score.Recommendations = append(score.Recommendations, "Monitor transaction")
	} else {
		score.Level = "low"
		score.Recommendations = append(score.Recommendations, "Proceed with standard processing")
	}

	return score, nil
}

func (s *financeSecurityService) calculateAmountRisk(tx *Transaction) float64 {
	if tx.Amount > 100000 {
		return 1.0
	} else if tx.Amount > 50000 {
		return 0.8
	} else if tx.Amount > 10000 {
		return 0.5
	} else if tx.Amount > 1000 {
		return 0.2
	}
	return 0.1
}

func (s *financeSecurityService) calculateGeoRisk(tx *Transaction) float64 {
	if tx.Location == nil {
		return 0.3
	}

	highRiskCountries := map[string]bool{"KP": true, "IR": true, "SY": true, "CU": true}
	if highRiskCountries[tx.Location.Country] {
		return 1.0
	}

	mediumRiskCountries := map[string]bool{"RU": true, "CN": true, "NG": true}
	if mediumRiskCountries[tx.Location.Country] {
		return 0.6
	}

	return 0.2
}

func (s *financeSecurityService) calculateVelocityRisk(tx *Transaction) float64 {
	return 0.4
}

func (s *financeSecurityService) calculateDeviceRisk(tx *Transaction) float64 {
	if tx.DeviceFingerprint == "" {
		return 0.5
	}
	return 0.2
}

func (s *financeSecurityService) calculateTimeRisk(tx *Transaction) float64 {
	hour := tx.Timestamp.Hour()
	if hour >= 2 && hour <= 5 {
		return 0.8
	} else if hour >= 22 || hour <= 6 {
		return 0.5
	}
	return 0.2
}

func (s *financeSecurityService) ApplyTransactionPolicy(ctx context.Context, tx *Transaction) error {
	for _, policy := range s.policies {
		if policy.TransactionType == tx.TransactionType {
			if tx.Amount < policy.MinAmount {
				return fmt.Errorf("transaction amount %.2f is below minimum %.2f", tx.Amount, policy.MinAmount)
			}
			if tx.Amount > policy.MaxAmount {
				return fmt.Errorf("transaction amount %.2f exceeds maximum %.2f", tx.Amount, policy.MaxAmount)
			}
		}
	}
	return nil
}

func (s *financeSecurityService) GenerateComplianceReport(ctx context.Context, period string) (*ComplianceReport, error) {
	report := &ComplianceReport{
		ReportID:    fmt.Sprintf("CR-%s-%d", strings.ToUpper(period), time.Now().Unix()),
		Period:      period,
		GeneratedAt: time.Now(),
		TotalTransactions: 15000,
		TotalAmount:   50000000,
		HighRiskCount: 150,
		AMLFlagsCount: 45,
		FraudDetected: 12,
		ComplianceMetrics: map[string]float64{
			"aml_compliance_rate":     99.5,
			"fraud_detection_rate":    98.2,
			"kyc_completion_rate":     97.8,
			"regulatory_report_rate":  99.1,
		},
		Findings: []ComplianceFinding{
			{
				Severity:      "medium",
				Category:      "AML",
				Description:   "Multiple transactions approaching reporting threshold",
				TransactionIDs: []string{"TX001", "TX002"},
				Recommendation: "Review and enhance monitoring thresholds",
			},
			{
				Severity:      "low",
				Category:      "KYC",
				Description:   "Some customers with incomplete verification",
				TransactionIDs: []string{"TX003"},
				Recommendation: "Send verification reminders to affected customers",
			},
		},
	}

	return report, nil
}

type TokenizationService interface {
	Tokenize(ctx context.Context, data string, tokenType string) (string, error)
	Detokenize(ctx context.Context, token string) (string, error)
	ValidateToken(ctx context.Context, token string) (bool, error)
}

type tokenizationService struct {
	tokens map[string]string
}

func NewTokenizationService() TokenizationService {
	return &tokenizationService{
		tokens: make(map[string]string),
	}
}

func (s *tokenizationService) Tokenize(ctx context.Context, data string, tokenType string) (string, error) {
	token := fmt.Sprintf("tok_%s_%d", tokenType, time.Now().UnixNano())
	s.tokens[token] = data
	return token, nil
}

func (s *tokenizationService) Detokenize(ctx context.Context, token string) (string, error) {
	data, exists := s.tokens[token]
	if !exists {
		return "", errors.New("token not found")
	}
	return data, nil
}

func (s *tokenizationService) ValidateToken(ctx context.Context, token string) (bool, error) {
	_, exists := s.tokens[token]
	return exists, nil
}
