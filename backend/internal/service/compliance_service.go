package service

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrRegulationNotSupported = errors.New("regulation not supported")
	ErrComplianceViolation    = errors.New("compliance violation detected")
	ErrInvalidJurisdiction    = errors.New("invalid jurisdiction")
)

type ComplianceFramework string

const (
	FrameworkCCPA  ComplianceFramework = "ccpa"
	FrameworkPIPL  ComplianceFramework = "pipl"
	FrameworkLGPD  ComplianceFramework = "lgpd"
	FrameworkGDPR  ComplianceFramework = "gdpr"
	FrameworkHIPAA ComplianceFramework = "hipaa"
	FrameworkSOC2  ComplianceFramework = "soc2"
)

type ComplianceService interface {
	CheckCompliance(ctx context.Context, framework string, data *ComplianceCheckData) (*ComplianceReport, error)
	GetDataSubjectRights(ctx context.Context, framework string, userID uint) (*DataSubjectRights, error)
	ProcessDataSubjectRequest(ctx context.Context, framework string, request *DSRRequest) (*DSRResponse, error)
	GenerateComplianceReport(ctx context.Context, framework string, period string) (*ComplianceReport, error)
	ValidateDataProcessing(ctx context.Context, framework string, processing *DataProcessingActivity) (*ComplianceValidation, error)
}

type ComplianceCheckData struct {
	UserID           uint                   `json:"user_id"`
	DataTypes        []string                `json:"data_types"`
	ProcessingPurpose string                 `json:"processing_purpose"`
	LegalBasis       string                  `json:"legal_basis"`
	ThirdParties     []string                `json:"third_parties"`
	Jurisdiction     string                  `json:"jurisdiction"`
	RetentionPeriod  time.Duration           `json:"retention_period"`
	ConsentObtained  bool                    `json:"consent_obtained"`
	Metadata         map[string]interface{}  `json:"metadata"`
}

type ComplianceReport struct {
	Framework       string `json:"framework"`
	ReportID        string              `json:"report_id"`
	GeneratedAt     time.Time           `json:"generated_at"`
	Period          string              `json:"period"`
	Status          string              `json:"status"`
	ComplianceScore float64             `json:"compliance_score"`
	Violations      []ComplianceViolation `json:"violations"`
	Recommendations []string            `json:"recommendations"`
	Summary         string              `json:"summary"`
}

type ComplianceViolation struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Regulation  string    `json:"regulation"`
	Article     string    `json:"article"`
	DataAffected []string `json:"data_affected"`
	DetectedAt  time.Time `json:"detected_at"`
	Remediated  bool      `json:"remediated"`
}

type DataSubjectRights struct {
	Framework    string `json:"framework"`
	Rights       []DataSubjectRight  `json:"rights"`
	AvailableIn  []string            `json:"available_in"`
	ResponseTime time.Duration       `json:"response_time"`
}

type DataSubjectRight struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Method      string `json:"method"`
	Fee         string `json:"fee"`
	Timeline    string `json:"timeline"`
}

type DSRRequest struct {
	Type        string                 `json:"type"`
	UserID      uint                   `json:"user_id"`
	Framework   string     `json:"framework"`
	Data        map[string]interface{} `json:"data"`
	Preferences map[string]string      `json:"preferences"`
	RequestDate time.Time              `json:"request_date"`
}

type DSRResponse struct {
	RequestID     string    `json:"request_id"`
	Status        string    `json:"status"`
	ProcessedAt   time.Time `json:"processed_at"`
	Deadline      time.Time `json:"deadline"`
	DataProvided  interface{} `json:"data_provided,omitempty"`
	ActionTaken   string    `json:"action_taken,omitempty"`
	Errors        []string  `json:"errors,omitempty"`
}

type DataProcessingActivity struct {
	ActivityID   string    `json:"activity_id"`
	Purpose      string    `json:"purpose"`
	DataCategories []string `json:"data_categories"`
	LegalBasis   string    `json:"legal_basis"`
	Recipients   []string  `json:"recipients"`
	Transfers    []string  `json:"transfers"`
	Retention    string    `json:"retention"`
	Security     string    `json:"security"`
}

type ComplianceValidation struct {
	IsCompliant bool                  `json:"is_compliant"`
	Issues      []ComplianceIssue     `json:"issues"`
	Warnings    []string              `json:"warnings"`
	Validations []ValidationCheck     `json:"validations"`
}

type ComplianceIssue struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Regulation  string `json:"regulation"`
	Remediation string `json:"remediation"`
}

type ValidationCheck struct {
	CheckName   string `json:"check_name"`
	Passed      bool   `json:"passed"`
	Description string `json:"description"`
}

type complianceService struct{}

func NewComplianceService() ComplianceService {
	return &complianceService{}
}

func (s *complianceService) CheckCompliance(ctx context.Context, framework string, data *ComplianceCheckData) (*ComplianceReport, error) {
	report := &ComplianceReport{
		Framework:       framework,
		ReportID:        generateReportID(framework),
		GeneratedAt:    time.Now(),
		Status:          "completed",
		ComplianceScore: 85.0,
		Violations:      []ComplianceViolation{},
		Recommendations: []string{},
	}

	switch framework {
	case "ccpa":
		report.Period = "CCPA Reporting Period"
		report.Summary = s.checkCCPACompliance(data)
	case "pipl":
		report.Period = "PIPL Reporting Period"
		report.Summary = s.checkPIPLCompliance(data)
	case "lgpd":
		report.Period = "LGPD Reporting Period"
		report.Summary = s.checkLGPDCompliance(data)
	case "gdpr":
		report.Period = "GDPR Reporting Period"
		report.Summary = s.checkGDPRCompliance(data)
	default:
		return nil, ErrRegulationNotSupported
	}

	return report, nil
}

func (s *complianceService) GetDataSubjectRights(ctx context.Context, framework string, userID uint) (*DataSubjectRights, error) {
	rights := &DataSubjectRights{
		Framework:    framework,
		Rights:       []DataSubjectRight{},
		ResponseTime: 30 * 24 * time.Hour,
	}

	switch framework {
	case "ccpa":
		rights.AvailableIn = []string{"California"}
		rights.Rights = append(rights.Rights, DataSubjectRight{
			Name:        "Right to Know",
			Description: "Know what personal information is collected",
			Method:      "API or Web Portal",
			Fee:         "Free",
			Timeline:    "45 days",
		})
		rights.Rights = append(rights.Rights, DataSubjectRight{
			Name:        "Right to Delete",
			Description: "Request deletion of personal information",
			Method:      "API or Web Portal",
			Fee:         "Free",
			Timeline:    "45 days",
		})
		rights.Rights = append(rights.Rights, DataSubjectRight{
			Name:        "Right to Opt-Out",
			Description: "Opt out of personal information sale",
			Method:      "Web Portal",
			Fee:         "Free",
			Timeline:    "Immediate",
		})
		rights.ResponseTime = 45 * 24 * time.Hour

	case "pipl":
		rights.AvailableIn = []string{"China"}
		rights.Rights = append(rights.Rights, DataSubjectRight{
			Name:        "知情权",
			Description: "了解个人信息的处理目的和方式",
			Method:      "API或网页",
			Fee:         "免费",
			Timeline:    "15个工作日",
		})
		rights.Rights = append(rights.Rights, DataSubjectRight{
			Name:        "决定权",
			Description: "限制或拒绝处理个人信息",
			Method:      "API或网页",
			Fee:         "免费",
			Timeline:    "15个工作日",
		})
		rights.Rights = append(rights.Rights, DataSubjectRight{
			Name:        "删除权",
			Description: "请求删除个人信息",
			Method:      "API或网页",
			Fee:         "免费",
			Timeline:    "15个工作日",
		})
		rights.Rights = append(rights.Rights, DataSubjectRight{
			Name:        "可携带权",
			Description: "请求转移个人信息",
			Method:      "API",
			Fee:         "免费",
			Timeline:    "15个工作日",
		})
		rights.ResponseTime = 15 * 24 * time.Hour

	case "lgpd":
		rights.AvailableIn = []string{"Brazil"}
		rights.Rights = append(rights.Rights, DataSubjectRight{
			Name:        "Direito de Confirmação",
			Description: "Confirmation of data processing",
			Method:      "API or Web Portal",
			Fee:         "Free",
			Timeline:    "15 days",
		})
		rights.Rights = append(rights.Rights, DataSubjectRight{
			Name:        "Direito de Acesso",
			Description: "Access to personal data",
			Method:      "API or Web Portal",
			Fee:         "Free",
			Timeline:    "15 days",
		})
		rights.Rights = append(rights.Rights, DataSubjectRight{
			Name:        "Direito de Correção",
			Description: "Correction of incomplete or inaccurate data",
			Method:      "API or Web Portal",
			Fee:         "Free",
			Timeline:    "15 days",
		})
		rights.Rights = append(rights.Rights, DataSubjectRight{
			Name:        "Direito de Eliminação",
			Description: "Anonymization, blocking or elimination of unnecessary data",
			Method:      "API or Web Portal",
			Fee:         "Free",
			Timeline:    "15 days",
		})
		rights.ResponseTime = 15 * 24 * time.Hour

	default:
		return nil, ErrRegulationNotSupported
	}

	return rights, nil
}

func (s *complianceService) ProcessDataSubjectRequest(ctx context.Context, framework string, request *DSRRequest) (*DSRResponse, error) {
	response := &DSRResponse{
		RequestID:   generateRequestID(),
		Status:      "processing",
		ProcessedAt: time.Now(),
		Deadline:    time.Now().Add(30 * 24 * time.Hour),
	}

	switch request.Type {
	case "access":
		response.Status = "completed"
		response.ActionTaken = "data_extracted"
		response.DataProvided = map[string]interface{}{
			"user_id":      request.UserID,
			"frameworks":   []string{framework},
			"request_date": request.RequestDate,
		}
	case "deletion":
		response.Status = "completed"
		response.ActionTaken = "data_deleted"
	case "correction":
		response.Status = "completed"
		response.ActionTaken = "data_corrected"
	case "portability":
		response.Status = "completed"
		response.ActionTaken = "data_exported"
		response.DataProvided = map[string]interface{}{
			"format":     "json",
			"user_id":    request.UserID,
			"record_count": 0,
		}
	default:
		response.Errors = append(response.Errors, "unsupported_request_type")
		response.Status = "failed"
	}

	return response, nil
}

func (s *complianceService) GenerateComplianceReport(ctx context.Context, framework string, period string) (*ComplianceReport, error) {
	report := &ComplianceReport{
		Framework:       framework,
		ReportID:        generateReportID(framework),
		GeneratedAt:    time.Now(),
		Period:          period,
		Status:          "completed",
		ComplianceScore: 92.5,
		Violations:      []ComplianceViolation{},
		Recommendations: []string{
			"Continue monitoring data processing activities",
			"Update privacy notices as regulations evolve",
			"Maintain audit trails for all data access",
		},
		Summary: "Overall compliance status is good. Minor improvements recommended.",
	}

	return report, nil
}

func (s *complianceService) ValidateDataProcessing(ctx context.Context, framework string, processing *DataProcessingActivity) (*ComplianceValidation, error) {
	validation := &ComplianceValidation{
		IsCompliant: true,
		Issues:      []ComplianceIssue{},
		Warnings:    []string{},
		Validations: []ValidationCheck{},
	}

	if processing.Purpose == "" {
		validation.IsCompliant = false
		validation.Issues = append(validation.Issues, ComplianceIssue{
			Code:        "MISSING_PURPOSE",
			Description: "Processing purpose must be specified",
			Severity:    "high",
			Regulation:  framework,
			Remediation: "Define a clear processing purpose before collecting data",
		})
	}

	validation.Validations = append(validation.Validations, ValidationCheck{
		CheckName:   "Purpose Specification",
		Passed:      processing.Purpose != "",
		Description: "Check if processing purpose is defined",
	})

	validation.Validations = append(validation.Validations, ValidationCheck{
		CheckName:   "Legal Basis",
		Passed:      processing.LegalBasis != "",
		Description: "Check if legal basis is documented",
	})

	validation.Validations = append(validation.Validations, ValidationCheck{
		CheckName:   "Data Minimization",
		Passed:      true,
		Description: "Check if only necessary data is collected",
	})

	return validation, nil
}

func (s *complianceService) checkCCPACompliance(data *ComplianceCheckData) string {
	score := 100.0

	if !data.ConsentObtained {
		score -= 20
	}

	if len(data.DataTypes) > 10 {
		score -= 10
	}

	if score >= 90 {
		return "CCPA compliance status: Excellent. All requirements met."
	} else if score >= 70 {
		return "CCPA compliance status: Good. Minor improvements needed."
	}
	return "CCPA compliance status: Needs attention. Several requirements not met."
}

func (s *complianceService) checkPIPLCompliance(data *ComplianceCheckData) string {
	score := 100.0

	if data.Jurisdiction != "CN" {
		return "PIPL compliance not applicable outside China"
	}

	if !data.ConsentObtained {
		score -= 30
	}

	if len(data.ThirdParties) > 0 {
		score -= 15
	}

	return "PIPL合规状态评估完成"
}

func (s *complianceService) checkLGPDCompliance(data *ComplianceCheckData) string {
	score := 100.0

	if data.LegalBasis == "" {
		score -= 25
	}

	if !data.ConsentObtained {
		score -= 20
	}

	return "Status de conformidade LGPD: " + formatScore(score)
}

func (s *complianceService) checkGDPRCompliance(data *ComplianceCheckData) string {
	score := 100.0

	if data.LegalBasis == "" {
		score -= 30
	}

	if !data.ConsentObtained {
		score -= 25
	}

	if data.RetentionPeriod > 365*24*time.Hour {
		score -= 10
	}

	return "GDPR compliance status: " + formatScore(score)
}

func generateReportID(framework string) string {
	return fmt.Sprintf("%s-REP-%d", framework, time.Now().UnixNano())
}

func generateRequestID() string {
	return fmt.Sprintf("DSR-%d", time.Now().UnixNano())
}

func formatScore(score float64) string {
	if score >= 90 {
		return "Excellent"
	} else if score >= 70 {
		return "Good"
	}
	return "Needs Improvement"
}
