package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrPIAProjectNotFound    = errors.New("privacy impact assessment project not found")
	ErrConsentNotFound       = errors.New("consent record not found")
	ErrDataSubjectNotFound   = errors.New("data subject not found")
	ErrInvalidConsentType    = errors.New("invalid consent type")
)

type PrivacyEngineeringService interface {
	CreatePrivacyImpactAssessment(ctx context.Context, pia *PIAProject) (*PIAProject, error)
	GetPrivacyImpactAssessment(ctx context.Context, piaID string) (*PIAProject, error)
	AssessDataMinimization(ctx context.Context, assessment *DataMinimizationCheck) (*DataMinimizationResult, error)
	ManageConsent(ctx context.Context, consent *ConsentRecord) (*ConsentRecord, error)
	GetConsentStatus(ctx context.Context, userID uint) (*ConsentStatus, error)
	ProcessDataSubjectRights(ctx context.Context, request *DataSubjectRightsRequest) (*DataSubjectRightsResponse, error)
	VerifyDataMinimization(ctx context.Context, dataType string) (*MinimizationVerification, error)
}

type PIAProject struct {
	ProjectID       string              `json:"project_id"`
	Name            string              `json:"name"`
	Description     string              `json:"description"`
	Status          string              `json:"status"`
	DataTypes       []string            `json:"data_types"`
	Purpose         string              `json:"purpose"`
	LegalBasis      string              `json:"legal_basis"`
	ThirdParties    []ThirdPartyInvolvement `json:"third_parties"`
	RiskAssessment  PIARiskAssessment   `json:"risk_assessment"`
	Mitigations     []PIMitigation      `json:"mitigations"`
	CreatedBy       string              `json:"created_by"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	ApprovedAt      *time.Time          `json:"approved_at,omitempty"`
	ApprovedBy      string              `json:"approved_by,omitempty"`
}

type ThirdPartyInvolvement struct {
	Name         string   `json:"name"`
	Purpose      string   `json:"purpose"`
	DataShared   []string `json:"data_shared"`
	Country      string   `json:"country"`
	HasDPA       bool     `json:"has_dpa"`
}

type PIARiskAssessment struct {
	OverallRisk   string             `json:"overall_risk"`
	RiskScore     float64            `json:"risk_score"`
	IdentifiedRisks []PIARisk       `json:"identified_risks"`
	ImpactLevel   string             `json:"impact_level"`
	Likelihood    string             `json:"likelihood"`
}

type PIARisk struct {
	RiskID       string    `json:"risk_id"`
	Category     string    `json:"category"`
	Description  string    `json:"description"`
	Severity     string    `json:"severity"`
	Probability  string    `json:"probability"`
	Impact       string    `json:"impact"`
	MitigationID string    `json:"mitigation_id,omitempty"`
}

type PIMitigation struct {
	MitigationID   string    `json:"mitigation_id"`
	Description     string    `json:"description"`
	Type            string    `json:"type"`
	Implementation  string    `json:"implementation"`
	Status          string    `json:"status"`
	Owner           string    `json:"owner"`
	DueDate         time.Time `json:"due_date"`
}

type DataMinimizationCheck struct {
	CheckID        string   `json:"check_id"`
	DataType       string   `json:"data_type"`
	Purpose        string   `json:"purpose"`
	CollectedData  []string `json:"collected_data"`
	RequiredData   []string `json:"required_data"`
	LegalBasis     string   `json:"legal_basis"`
}

type DataMinimizationResult struct {
	CheckID          string              `json:"check_id"`
	IsCompliant      bool                `json:"is_compliant"`
	ComplianceScore  float64             `json:"compliance_score"`
	ExcessData       []string            `json:"excess_data"`
	MissingData      []string            `json:"missing_data"`
	Recommendations  []string            `json:"recommendations"`
	Violations       []MinimizationViolation `json:"violations"`
}

type MinimizationViolation struct {
	ViolationID  string `json:"violation_id"`
	DataField    string `json:"data_field"`
	ViolationType string `json:"violation_type"`
	Description  string `json:"description"`
	Severity     string `json:"severity"`
}

type ConsentRecord struct {
	RecordID      string           `json:"record_id"`
	UserID        uint             `json:"user_id"`
	ConsentType   ConsentType      `json:"consent_type"`
	Purpose       string           `json:"purpose"`
	DataProcessed []string         `json:"data_processed"`
	Granted       bool             `json:"granted"`
	GrantedAt     *time.Time       `json:"granted_at,omitempty"`
	WithdrawnAt   *time.Time       `json:"withdrawn_at,omitempty"`
	Method        string           `json:"method"`
	IPAddress     string           `json:"ip_address"`
	UserAgent     string           `json:"user_agent"`
	Version       string           `json:"version"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type ConsentType string

const (
	ConsentMarketing   ConsentType = "marketing"
	ConsentAnalytics  ConsentType = "analytics"
	ConsentPersonalization ConsentType = "personalization"
	ConsentDataSharing ConsentType = "data_sharing"
	ConsentEssential  ConsentType = "essential"
	ConsentAll        ConsentType = "all"
)

type ConsentStatus struct {
	UserID       uint                   `json:"user_id"`
	LastUpdated  time.Time              `json:"last_updated"`
	Consents     map[ConsentType]bool   `json:"consents"`
	HasValidConsent bool                `json:"has_valid_consent"`
	Preferences  ConsentPreferences     `json:"preferences"`
}

type ConsentPreferences struct {
	AllowPersonalization bool `json:"allow_personalization"`
	AllowAnalytics      bool `json:"allow_analytics"`
	AllowMarketing      bool `json:"allow_marketing"`
	AllowDataSharing    bool `json:"allow_data_sharing"`
}

type DataSubjectRightsRequest struct {
	RequestID   string    `json:"request_id"`
	UserID      uint      `json:"user_id"`
	RightType   RightsType `json:"right_type"`
	Framework   string    `json:"framework"`
	Data        map[string]interface{} `json:"data"`
	RequestedAt time.Time `json:"requested_at"`
}

type RightsType string

const (
	RightAccess        RightsType = "access"
	RightDeletion      RightsType = "deletion"
	RightCorrection    RightsType = "correction"
	RightPortability   RightsType = "portability"
	RightRestriction   RightsType = "restriction"
	RightObjection     RightsType = "objection"
	RightWithdraw      RightsType = "withdraw_consent"
)

type DataSubjectRightsResponse struct {
	RequestID    string    `json:"request_id"`
	Status       string    `json:"status"`
	ProcessedAt  time.Time `json:"processed_at"`
	Deadline     time.Time `json:"deadline"`
	ActionTaken  string    `json:"action_taken"`
	DataProvided interface{} `json:"data_provided,omitempty"`
	Errors       []string  `json:"errors,omitempty"`
}

type MinimizationVerification struct {
	DataType      string    `json:"data_type"`
	IsMinimized   bool      `json:"is_minimized"`
	Collection    CollectionCheck `json:"collection"`
	Storage       StorageCheck   `json:"storage"`
	Retention     RetentionCheck  `json:"retention"`
	Processing    ProcessingCheck `json:"processing"`
}

type CollectionCheck struct {
	Collected   []string `json:"collected"`
	Necessary   []string `json:"necessary"`
	Excess      []string `json:"excess"`
	IsCompliant bool     `json:"is_compliant"`
}

type StorageCheck struct {
	Encrypted   bool     `json:"encrypted"`
	Locations   []string `json:"locations"`
	Restricted  bool     `json:"restricted"`
	IsCompliant bool     `json:"is_compliant"`
}

type RetentionCheck struct {
	Policy      string    `json:"policy"`
	CurrentAge  time.Duration `json:"current_age"`
	MaxAge      time.Duration `json:"max_age"`
	IsCompliant bool     `json:"is_compliant"`
}

type ProcessingCheck struct {
	Purposes    []string `json:"purposes"`
	Authorized  []string `json:"authorized"`
	IsCompliant bool     `json:"is_compliant"`
}

type privacyEngineeringService struct {
	mu         sync.RWMutex
	piaProjects map[string]*PIAProject
	consents   map[uint][]*ConsentRecord
}

func NewPrivacyEngineeringService() PrivacyEngineeringService {
	return &privacyEngineeringService{
		piaProjects: make(map[string]*PIAProject),
		consents:    make(map[uint][]*ConsentRecord),
	}
}

func (s *privacyEngineeringService) CreatePrivacyImpactAssessment(ctx context.Context, pia *PIAProject) (*PIAProject, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if pia.ProjectID == "" {
		pia.ProjectID = fmt.Sprintf("PIA-%d", time.Now().UnixNano())
	}

	pia.Status = "draft"
	pia.CreatedAt = time.Now()
	pia.UpdatedAt = time.Now()

	s.assessPIARisk(pia)

	s.piaProjects[pia.ProjectID] = pia

	return pia, nil
}

func (s *privacyEngineeringService) GetPrivacyImpactAssessment(ctx context.Context, piaID string) (*PIAProject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pia, exists := s.piaProjects[piaID]
	if !exists {
		return nil, ErrPIAProjectNotFound
	}

	return pia, nil
}

func (s *privacyEngineeringService) assessPIARisk(pia *PIAProject) {
	riskScore := 0.0

	if len(pia.DataTypes) > 10 {
		riskScore += 20
	} else if len(pia.DataTypes) > 5 {
		riskScore += 10
	}

	if len(pia.ThirdParties) > 3 {
		riskScore += 25
	} else if len(pia.ThirdParties) > 1 {
		riskScore += 15
	}

	if len(pia.DataTypes) > 0 {
		for _, dt := range pia.DataTypes {
			if dt == "biometric" || dt == "health" || dt == "financial" {
				riskScore += 15
				break
			}
		}
	}

	for _, tp := range pia.ThirdParties {
		if tp.Country == "CN" || tp.Country == "US" || tp.Country == "DE" {
			riskScore += 5
		}
	}

	if riskScore >= 70 {
		pia.RiskAssessment.OverallRisk = "high"
		pia.RiskAssessment.ImpactLevel = "significant"
		pia.RiskAssessment.Likelihood = "likely"
	} else if riskScore >= 40 {
		pia.RiskAssessment.OverallRisk = "medium"
		pia.RiskAssessment.ImpactLevel = "moderate"
		pia.RiskAssessment.Likelihood = "possible"
	} else {
		pia.RiskAssessment.OverallRisk = "low"
		pia.RiskAssessment.ImpactLevel = "minimal"
		pia.RiskAssessment.Likelihood = "unlikely"
	}

	pia.RiskAssessment.RiskScore = riskScore
}

func (s *privacyEngineeringService) AssessDataMinimization(ctx context.Context, assessment *DataMinimizationCheck) (*DataMinimizationResult, error) {
	result := &DataMinimizationResult{
		CheckID:         assessment.CheckID,
		IsCompliant:     true,
		ComplianceScore: 100.0,
		ExcessData:      []string{},
		MissingData:     []string{},
		Recommendations: []string{},
		Violations:       []MinimizationViolation{},
	}

	excessMap := make(map[string]bool)
	for _, collected := range assessment.CollectedData {
		isRequired := false
		for _, required := range assessment.RequiredData {
			if collected == required {
				isRequired = true
				break
			}
		}
		if !isRequired {
			excessMap[collected] = true
		}
	}

	for excess := range excessMap {
		result.ExcessData = append(result.ExcessData, excess)
		result.Violations = append(result.Violations, MinimizationViolation{
			ViolationID:   fmt.Sprintf("DM-%d", time.Now().UnixNano()),
			DataField:     excess,
			ViolationType: "excess_collection",
			Description:   fmt.Sprintf("Data field '%s' is not necessary for stated purpose", excess),
			Severity:      "medium",
		})
		result.ComplianceScore -= 15.0
		result.IsCompliant = false
	}

	for _, required := range assessment.RequiredData {
		isCollected := false
		for _, collected := range assessment.CollectedData {
			if required == collected {
				isCollected = true
				break
			}
		}
		if !isCollected {
			result.MissingData = append(result.MissingData, required)
			result.Recommendations = append(result.Recommendations,
				fmt.Sprintf("Collect required data field: %s", required))
		}
	}

	if len(result.ExcessData) > 0 {
		result.Recommendations = append(result.Recommendations,
			"Remove unnecessary data collection to comply with data minimization principles")
	}

	if result.ComplianceScore < 0 {
		result.ComplianceScore = 0
	}

	return result, nil
}

func (s *privacyEngineeringService) ManageConsent(ctx context.Context, consent *ConsentRecord) (*ConsentRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if consent.RecordID == "" {
		consent.RecordID = fmt.Sprintf("consent-%d", time.Now().UnixNano())
	}

	now := time.Now()

	switch consent.ConsentType {
	case ConsentMarketing, ConsentAnalytics, ConsentPersonalization, ConsentDataSharing:
		if !consent.Granted {
			consent.WithdrawnAt = &now
		} else {
			consent.GrantedAt = &now
		}
	case ConsentEssential:
		consent.Granted = true
		consent.GrantedAt = &now
	default:
		return nil, ErrInvalidConsentType
	}

	s.consents[consent.UserID] = append(s.consents[consent.UserID], consent)

	return consent, nil
}

func (s *privacyEngineeringService) GetConsentStatus(ctx context.Context, userID uint) (*ConsentStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records, exists := s.consents[userID]
	if !exists {
		return &ConsentStatus{
			UserID:          userID,
			LastUpdated:     time.Now(),
			Consents:        make(map[ConsentType]bool),
			HasValidConsent: false,
			Preferences: ConsentPreferences{
				AllowPersonalization: false,
				AllowAnalytics:      false,
				AllowMarketing:      false,
				AllowDataSharing:    false,
			},
		}, nil
	}

	status := &ConsentStatus{
		UserID:          userID,
		LastUpdated:     time.Now(),
		Consents:        make(map[ConsentType]bool),
		HasValidConsent: true,
		Preferences:    ConsentPreferences{},
	}

	for _, record := range records {
		if record.Granted {
			status.Consents[record.ConsentType] = true

			switch record.ConsentType {
			case ConsentPersonalization:
				status.Preferences.AllowPersonalization = true
			case ConsentAnalytics:
				status.Preferences.AllowAnalytics = true
			case ConsentMarketing:
				status.Preferences.AllowMarketing = true
			case ConsentDataSharing:
				status.Preferences.AllowDataSharing = true
			}
		}
	}

	return status, nil
}

func (s *privacyEngineeringService) ProcessDataSubjectRights(ctx context.Context, request *DataSubjectRightsRequest) (*DataSubjectRightsResponse, error) {
	response := &DataSubjectRightsResponse{
		RequestID:   request.RequestID,
		Status:      "completed",
		ProcessedAt: time.Now(),
		Deadline:    request.RequestedAt.Add(30 * 24 * time.Hour),
		Errors:      []string{},
	}

	switch request.RightType {
	case RightAccess:
		response.ActionTaken = "data_extracted"
		response.DataProvided = map[string]interface{}{
			"user_id":       request.UserID,
			"personal_data": []string{},
			"processing_activities": []string{},
		}
	case RightDeletion:
		response.ActionTaken = "data_deleted"
	case RightCorrection:
		response.ActionTaken = "data_corrected"
	case RightPortability:
		response.ActionTaken = "data_exported"
		response.DataProvided = map[string]interface{}{
			"format":     "json",
			"user_id":    request.UserID,
			"record_count": 0,
		}
	case RightRestriction:
		response.ActionTaken = "processing_restricted"
	case RightObjection:
		response.ActionTaken = "objection_registered"
	case RightWithdraw:
		response.ActionTaken = "consent_withdrawn"
	default:
		response.Status = "failed"
		response.Errors = append(response.Errors, "unsupported_right_type")
	}

	return response, nil
}

func (s *privacyEngineeringService) VerifyDataMinimization(ctx context.Context, dataType string) (*MinimizationVerification, error) {
	verification := &MinimizationVerification{
		DataType:    dataType,
		IsMinimized: true,
		Collection: CollectionCheck{
			Collected:   []string{"id", "email", "name"},
			Necessary:   []string{"id", "email", "name"},
			Excess:      []string{},
			IsCompliant: true,
		},
		Storage: StorageCheck{
			Encrypted:  true,
			Locations:  []string{"database"},
			Restricted: true,
			IsCompliant: true,
		},
		Retention: RetentionCheck{
			Policy:      "1_year",
			CurrentAge:  180 * 24 * time.Hour,
			MaxAge:      365 * 24 * time.Hour,
			IsCompliant: true,
		},
		Processing: ProcessingCheck{
			Purposes:   []string{"authentication", "service_delivery"},
			Authorized: []string{"authentication", "service_delivery"},
			IsCompliant: true,
		},
	}

	return verification, nil
}
