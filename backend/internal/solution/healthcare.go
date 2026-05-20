package solution

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type HealthcareComplianceService interface {
	ValidatePHI(ctx context.Context, data *PHIData) (*PHIValidationResult, error)
	AnonymizeData(ctx context.Context, data interface{}, rules *AnonymizationRules) (interface{}, error)
	CheckHIPAACompliance(ctx context.Context, operation *HealthcareOperation) (*ComplianceResult, error)
	ManageConsent(ctx context.Context, consent *PatientConsent) error
	AuditAccess(ctx context.Context, access *PHIAccess) error
	GeneratePrivacyReport(ctx context.Context, patientID string) (*PrivacyReport, error)
}

type PHIData struct {
	PatientID      string                 `json:"patient_id"`
	PatientName    string                 `json:"patient_name"`
	DateOfBirth    time.Time              `json:"date_of_birth"`
	SSN            string                 `json:"ssn"`
	MedicalRecordNumber string             `json:"mrn"`
	Diagnosis      []Diagnosis            `json:"diagnosis"`
	Medications    []Medication            `json:"medications"`
	TreatmentPlans []TreatmentPlan        `json:"treatment_plans"`
	LabResults     []LabResult            `json:"lab_results"`
	ContactInfo    *ContactInfo           `json:"contact_info"`
	InsuranceInfo  *InsuranceInfo         `json:"insurance_info"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type Diagnosis struct {
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Provider    string    `json:"provider"`
	Severity    string    `json:"severity"`
}

type Medication struct {
	Name         string    `json:"name"`
	Dosage       string    `json:"dosage"`
	Frequency    string    `json:"frequency"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	Prescriber   string    `json:"prescriber"`
}

type TreatmentPlan struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	Goals       []string  `json:"goals"`
	Interventions []string `json:"interventions"`
}

type LabResult struct {
	TestName     string    `json:"test_name"`
	TestCode     string    `json:"test_code"`
	Value        string    `json:"value"`
	Unit         string    `json:"unit"`
	ReferenceRange string  `json:"reference_range"`
	Date         time.Time `json:"date"`
	LabName      string    `json:"lab_name"`
	ResultStatus string    `json:"result_status"`
}

type ContactInfo struct {
	Address     string `json:"address"`
	City        string `json:"city"`
	State       string `json:"state"`
	ZipCode     string `json:"zip_code"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
}

type InsuranceInfo struct {
	Provider       string `json:"provider"`
	PolicyNumber   string `json:"policy_number"`
	GroupNumber    string `json:"group_number"`
	SubscriberID   string `json:"subscriber_id"`
	Relationship   string `json:"relationship"`
}

type PHIValidationResult struct {
	Valid        bool     `json:"valid"`
	Errors       []string `json:"errors"`
	Warnings     []string `json:"warnings"`
	PHIElements  []string `json:"phi_elements"`
	SensitivityLevel string `json:"sensitivity_level"`
}

type AnonymizationRules struct {
	DirectIdentifiers []string           `json:"direct_identifiers"`
	QuasiIdentifiers  []string           `json:"quasi_identifiers"`
	KAnonymity        int                `json:"k_anonymity"`
	LDiversity        int                `json:"l_diversity"`
	Generalization    map[string]string  `json:"generalization"`
	Suppression       []string           `json:"suppression"`
	DateShift         bool               `json:"date_shift"`
	DateShiftRange    int                `json:"date_shift_range_days"`
}

type HealthcareOperation struct {
	OperationType string    `json:"operation_type"`
	PatientID     string    `json:"patient_id"`
	OperatorID    string    `json:"operator_id"`
	Department    string    `json:"department"`
	Purpose       string    `json:"purpose"`
	Timestamp     time.Time `json:"timestamp"`
	DataAccessed  []string  `json:"data_accessed"`
	ResourceID    string    `json:"resource_id"`
}

type ComplianceResult struct {
	Compliant      bool      `json:"compliant"`
	Violations     []Violation `json:"violations"`
	Recommendations []string `json:"recommendations"`
	AuditRequired   bool      `json:"audit_required"`
}

type Violation struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Regulation  string `json:"regulation"`
}

type PatientConsent struct {
	ConsentID       string    `json:"consent_id"`
	PatientID       string    `json:"patient_id"`
	ConsentType     string    `json:"consent_type"`
	Granted         bool      `json:"granted"`
	EffectiveDate   time.Time `json:"effective_date"`
	ExpirationDate  time.Time `json:"expiration_date"`
	Scope           []string  `json:"scope"`
	Revocable       bool      `json:"revocable"`
	LastModified    time.Time `json:"last_modified"`
	WitnessRequired bool      `json:"witness_required"`
	WitnessID       string    `json:"witness_id,omitempty"`
}

type PHIAccess struct {
	AccessID     string    `json:"access_id"`
	PatientID    string    `json:"patient_id"`
	UserID       string    `json:"user_id"`
	AccessType   string    `json:"access_type"`
	Timestamp    time.Time `json:"timestamp"`
	DataElements []string  `json:"data_elements"`
	Purpose      string    `json:"purpose"`
	IPAddress    string    `json:"ip_address"`
	Success      bool      `json:"success"`
	DeniedReason string    `json:"denied_reason,omitempty"`
}

type PrivacyReport struct {
	ReportID       string           `json:"report_id"`
	PatientID      string           `json:"patient_id"`
	GeneratedAt    time.Time        `json:"generated_at"`
	PeriodStart    time.Time        `json:"period_start"`
	PeriodEnd     time.Time        `json:"period_end"`
	TotalAccesses int              `json:"total_accesses"`
	AccessBreakdown map[string]int  `json:"access_breakdown"`
	Consents      []PatientConsent `json:"consents"`
	Violations    []Violation      `json:"violations"`
}

type healthcareComplianceService struct {
	consents      map[string]*PatientConsent
	auditLog      []PHIAccess
	phiRules      []PHIRule
}

type PHIRule struct {
	Name           string   `json:"name"`
	IdentifierType string   `json:"identifier_type"`
	Required       bool     `json:"required"`
	SensitivityLevel string `json:"sensitivity_level"`
}

func NewHealthcareComplianceService() HealthcareComplianceService {
	service := &healthcareComplianceService{
		consents: make(map[string]*PatientConsent),
		auditLog: []PHIAccess{},
		phiRules: []PHIRule{},
	}

	service.initializePHIRules()
	return service
}

func (s *healthcareComplianceService) initializePHIRules() {
	s.phiRules = []PHIRule{
		{Name: "patient_name", IdentifierType: "direct", Required: true, SensitivityLevel: "high"},
		{Name: "ssn", IdentifierType: "direct", Required: false, SensitivityLevel: "critical"},
		{Name: "date_of_birth", IdentifierType: "quasi", Required: true, SensitivityLevel: "medium"},
		{Name: "address", IdentifierType: "quasi", Required: false, SensitivityLevel: "medium"},
		{Name: "phone", IdentifierType: "direct", Required: false, SensitivityLevel: "high"},
		{Name: "email", IdentifierType: "direct", Required: false, SensitivityLevel: "high"},
		{Name: "mrn", IdentifierType: "direct", Required: true, SensitivityLevel: "medium"},
		{Name: "insurance_number", IdentifierType: "direct", Required: false, SensitivityLevel: "high"},
	}
}

func (s *healthcareComplianceService) ValidatePHI(ctx context.Context, data *PHIData) (*PHIValidationResult, error) {
	result := &PHIValidationResult{
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
		PHIElements: []string{},
	}

	if data.PatientID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "patient ID is required")
	}

	for _, rule := range s.phiRules {
		result.PHIElements = append(result.PHIElements, rule.Name)

		if rule.Required {
			switch rule.Name {
			case "patient_name":
				if data.PatientName == "" {
					result.Valid = false
					result.Errors = append(result.Errors, "patient name is required")
				}
			case "date_of_birth":
				if data.DateOfBirth.IsZero() {
					result.Warnings = append(result.Warnings, "date of birth not provided")
				}
			case "mrn":
				if data.MedicalRecordNumber == "" {
					result.Warnings = append(result.Warnings, "medical record number not provided")
				}
			}
		}
	}

	result.SensitivityLevel = s.calculateSensitivityLevel(data)

	return result, nil
}

func (s *healthcareComplianceService) calculateSensitivityLevel(data *PHIData) string {
	sensitiveCount := 0
	totalFields := 8

	if data.SSN != "" {
		sensitiveCount++
	}
	if data.InsuranceInfo != nil {
		sensitiveCount++
	}
	if len(data.Diagnosis) > 0 {
		sensitiveCount++
	}

	sensitivity := float64(sensitiveCount) / float64(totalFields)

	if sensitivity >= 0.7 {
		return "critical"
	} else if sensitivity >= 0.4 {
		return "high"
	} else if sensitivity >= 0.2 {
		return "medium"
	}
	return "low"
}

func (s *healthcareComplianceService) AnonymizeData(ctx context.Context, data interface{}, rules *AnonymizationRules) (interface{}, error) {
	if rules == nil {
		rules = &AnonymizationRules{
			KAnonymity:       5,
			LDiversity:       2,
			DateShift:        true,
			DateShiftRange:   30,
		}
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(dataBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	result = s.applyAnonymizationRules(result, rules)

	return result, nil
}

func (s *healthcareComplianceService) applyAnonymizationRules(data map[string]interface{}, rules *AnonymizationRules) map[string]interface{} {
	for _, identifier := range rules.DirectIdentifiers {
		if _, exists := data[identifier]; exists {
			data[identifier] = "[REDACTED]"
		}
	}

	for _, field := range rules.Suppression {
		if _, exists := data[field]; exists {
			data[field] = nil
		}
	}

	for field, level := range rules.Generalization {
		if _, exists := data[field]; exists {
			data[field] = s.generalizeValue(field, level)
		}
	}

	return data
}

func (s *healthcareComplianceService) generalizeValue(field string, level string) string {
	switch level {
	case "city":
		return "[CITY]"
	case "state":
		return "[STATE]"
	case "zip":
		return "[ZIP]"
	case "age":
		return "[AGE_RANGE]"
	case "date":
		return "[DATE]"
	default:
		return "[GENERALIZED]"
	}
}

func (s *healthcareComplianceService) CheckHIPAACompliance(ctx context.Context, operation *HealthcareOperation) (*ComplianceResult, error) {
	result := &ComplianceResult{
		Compliant:      true,
		Violations:     []Violation{},
		Recommendations: []string{},
	}

	validPurposes := []string{
		"treatment",
		"payment",
		"healthcare_operations",
		"public_health",
		"research",
		"legal",
	}

	purposeValid := false
	for _, validPurpose := range validPurposes {
		if operation.Purpose == validPurpose {
			purposeValid = true
			break
		}
	}

	if !purposeValid {
		result.Compliant = false
		result.Violations = append(result.Violations, Violation{
			Type:        "invalid_purpose",
			Severity:    "high",
			Description: fmt.Sprintf("Invalid purpose: %s", operation.Purpose),
			Regulation:  "HIPAA Privacy Rule",
		})
	}

	consentKey := fmt.Sprintf("%s_%s", operation.PatientID, operation.Purpose)
	if consent, exists := s.consents[consentKey]; exists {
		if !consent.Granted {
			result.Compliant = false
			result.Violations = append(result.Violations, Violation{
				Type:        "consent_missing",
				Severity:    "critical",
				Description: "Required patient consent not obtained",
				Regulation:  "HIPAA Privacy Rule",
			})
		}

		if consent.ExpirationDate.Before(operation.Timestamp) {
			result.Compliant = false
			result.Violations = append(result.Violations, Violation{
				Type:        "consent_expired",
				Severity:    "high",
				Description: "Patient consent has expired",
				Regulation:  "HIPAA Privacy Rule",
			})
		}
	}

	if operation.OperationType == "disclosure" {
		result.AuditRequired = true
		result.Recommendations = append(result.Recommendations, "Document this disclosure for audit trail")
	}

	return result, nil
}

func (s *healthcareComplianceService) ManageConsent(ctx context.Context, consent *PatientConsent) error {
	if consent.ConsentID == "" {
		return fmt.Errorf("consent ID is required")
	}

	if consent.PatientID == "" {
		return fmt.Errorf("patient ID is required")
	}

	consent.LastModified = time.Now()
	s.consents[consent.ConsentID] = consent

	return nil
}

func (s *healthcareComplianceService) AuditAccess(ctx context.Context, access *PHIAccess) error {
	if access.AccessID == "" {
		access.AccessID = fmt.Sprintf("ACC-%d", time.Now().UnixNano())
	}

	s.auditLog = append(s.auditLog, *access)

	return nil
}

func (s *healthcareComplianceService) GeneratePrivacyReport(ctx context.Context, patientID string) (*PrivacyReport, error) {
	report := &PrivacyReport{
		ReportID:    fmt.Sprintf("PR-%s-%d", patientID, time.Now().Unix()),
		PatientID:   patientID,
		GeneratedAt: time.Now(),
		PeriodStart: time.Now().AddDate(0, -1, 0),
		PeriodEnd:   time.Now(),
		AccessBreakdown: make(map[string]int),
		Consents:    []PatientConsent{},
		Violations:  []Violation{},
	}

	for _, consent := range s.consents {
		if consent.PatientID == patientID {
			report.Consents = append(report.Consents, *consent)
		}
	}

	for _, access := range s.auditLog {
		if access.PatientID == patientID {
			report.TotalAccesses++
			report.AccessBreakdown[access.AccessType]++
		}
	}

	return report, nil
}

type DataDeidentificationService interface {
	Deidentify(ctx context.Context, data *PHIData, method string) (*DeidentifiedData, error)
	Reidentify(ctx context.Context, deidentifiedData *DeidentifiedData) (*PHIData, error)
}

type DeidentifiedData struct {
	DeidentifiedID string                 `json:"deidentified_id"`
	Data           map[string]interface{} `json:"data"`
	Method         string                 `json:"method"`
	LinkingKey     string                 `json:"linking_key"`
	CreatedAt      time.Time              `json:"created_at"`
}

type dataDeidentificationService struct {
	mappingTable map[string]string
}

func NewDataDeidentificationService() DataDeidentificationService {
	return &dataDeidentificationService{
		mappingTable: make(map[string]string),
	}
}

func (s *dataDeidentificationService) Deidentify(ctx context.Context, data *PHIData, method string) (*DeidentifiedData, error) {
	deidData := &DeidentifiedData{
		DeidentifiedID: fmt.Sprintf("DEID-%d", time.Now().UnixNano()),
		Data:           make(map[string]interface{}),
		Method:         method,
		LinkingKey:     fmt.Sprintf("KEY-%d", time.Now().UnixNano()),
		CreatedAt:      time.Now(),
	}

	phiBytes, _ := json.Marshal(data)
	json.Unmarshal(phiBytes, &deidData.Data)

	switch method {
	case "safe_harbor":
		deidData.Data["patient_name"] = "[REDACTED]"
		deidData.Data["ssn"] = "[REDACTED]"
		deidData.Data["date_of_birth"] = "[SUPPRESSED]"
	case "limited_dataset":
		deidData.Data["patient_name"] = "[CODED]"
		deidData.Data["ssn"] = "[REMOVED]"
	case "statistical":
		if dob := deidData.Data["date_of_birth"]; dob != nil {
			deidData.Data["age_group"] = "[35-40]"
		}
	}

	return deidData, nil
}

func (s *dataDeidentificationService) Reidentify(ctx context.Context, deidentifiedData *DeidentifiedData) (*PHIData, error) {
	return nil, fmt.Errorf("re-identification not permitted")
}
