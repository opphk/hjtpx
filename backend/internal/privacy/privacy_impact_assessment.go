package privacy

import (
	"sync"
	"time"
)

type PrivacyImpactAssessment struct {
	ID                string
	ProjectName       string
	Description       string
	AssessmentDate    time.Time
	DataTypes         []DataType
	ProcessingPurpose string
	LegalBasis        string
	Stakeholders      []Stakeholder
	AssessmentLevel   AssessmentLevel
	Findings          []PIAFinding
	Recommendations   []Recommendation
	RiskScore         float64
	Status            AssessmentStatus
	Approver          string
	ExpiryDate        time.Time
	mu                sync.RWMutex
}

type DataType struct {
	Name           string
	Category       DataCategory
	Sensitivity    SensitivityLevel
	IsPersonal     bool
	IsSensitive    bool
	RetentionPeriod string
}

type DataCategory int

const (
	BasicIdentity DataCategory = iota
	ContactInfo
	FinancialInfo
	HealthInfo
	BiometricInfo
	LocationData
	BehavioralData
	TechnicalData
	GeneticData
	OtherData
)

type SensitivityLevel int

const (
	LowSensitivity SensitivityLevel = iota
	MediumSensitivity
	HighSensitivity
	CriticalSensitivity
)

type Stakeholder struct {
	Name         string
	Role         string
	ContactEmail string
	Involvement  InvolvementLevel
}

type InvolvementLevel int

const (
	Minimal InvolvementLevel = iota
	Moderate
	Significant
	Primary
)

type AssessmentLevel int

const (
	Level1Screen AssessmentLevel = iota
	Level2Standard
	Level3Comprehensive
)

type AssessmentStatus int

const (
	StatusDraft AssessmentStatus = iota
	StatusInReview
	StatusApproved
	StatusExpired
	StatusSuperseded
)

type PIAFinding struct {
	ID          string
	Category    FindingCategory
	Description string
	Severity    SeverityLevel
	Evidence    []string
	Impact      string
	Likelihood  LikelihoodLevel
}

type FindingCategory int

const (
	CategoryDataCollection FindingCategory = iota
	CategoryDataProcessing
	CategoryDataStorage
	CategoryDataSharing
	CategoryDataRetention
	CategoryAccessControl
	CategoryConsent
	CategoryTransparency
	CategorySecurity
	CategoryThirdParty
)

type SeverityLevel int

const (
	SeverityNegligible SeverityLevel = iota
	SeverityMinor
	SeverityModerate
	SeverityMajor
	SeverityCritical
)

type LikelihoodLevel int

const (
	LikelihoodRare LikelihoodLevel = iota
	LikelihoodUnlikely
	LikelihoodPossible
	LikelihoodLikely
	LikelihoodAlmostCertain
)

type Recommendation struct {
	ID          string
	FindingID   string
	Description string
	Priority    PriorityLevel
	Status      RecommendationStatus
	DueDate     time.Time
}

type PriorityLevel int

const (
	PriorityLow PriorityLevel = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

type RecommendationStatus int

const (
	RecStatusPending RecommendationStatus = iota
	RecStatusInProgress
	RecStatusImplemented
	RecStatusRejected
	RecStatusDeferred
)

func NewPrivacyImpactAssessment(projectName, description string) *PrivacyImpactAssessment {
	return &PrivacyImpactAssessment{
		ID:                generateUUID(),
		ProjectName:       projectName,
		Description:       description,
		AssessmentDate:    time.Now(),
		DataTypes:         make([]DataType, 0),
		Findings:          make([]PIAFinding, 0),
		Recommendations:   make([]Recommendation, 0),
		AssessmentLevel:   Level1Screen,
		Status:            StatusDraft,
		Stakeholders:      make([]Stakeholder, 0),
	}
}

func (pia *PrivacyImpactAssessment) AddDataType(dataType DataType) {
	pia.mu.Lock()
	defer pia.mu.Unlock()
	pia.DataTypes = append(pia.DataTypes, dataType)
	pia.updateAssessmentLevel()
}

func (pia *PrivacyImpactAssessment) AddFinding(finding PIAFinding) {
	pia.mu.Lock()
	defer pia.mu.Unlock()
	pia.Findings = append(pia.Findings, finding)
	pia.calculateRiskScore()
}

func (pia *PrivacyImpactAssessment) AddRecommendation(recommendation Recommendation) {
	pia.mu.Lock()
	defer pia.mu.Unlock()
	pia.Recommendations = append(pia.Recommendations, recommendation)
}

func (pia *PrivacyImpactAssessment) AddStakeholder(stakeholder Stakeholder) {
	pia.mu.Lock()
	defer pia.mu.Unlock()
	pia.Stakeholders = append(pia.Stakeholders, stakeholder)
}

func (pia *PrivacyImpactAssessment) updateAssessmentLevel() {
	hasSensitiveData := false
	hasHighSensitivity := false

	for _, dt := range pia.DataTypes {
		if dt.IsSensitive {
			hasSensitiveData = true
		}
		if dt.Sensitivity == CriticalSensitivity || dt.Sensitivity == HighSensitivity {
			hasHighSensitivity = true
		}
	}

	if hasHighSensitivity {
		pia.AssessmentLevel = Level3Comprehensive
	} else if hasSensitiveData {
		pia.AssessmentLevel = Level2Standard
	} else {
		pia.AssessmentLevel = Level1Screen
	}
}

func (pia *PrivacyImpactAssessment) calculateRiskScore() {
	totalRisk := 0.0
	maxRisk := 0.0

	for _, finding := range pia.Findings {
		severityScore := float64(finding.Severity)
		likelihoodScore := float64(finding.Likelihood)
		risk := severityScore * likelihoodScore
		totalRisk += risk
		maxRisk += 4.0 * 4.0
	}

	if maxRisk > 0 {
		pia.RiskScore = (totalRisk / maxRisk) * 100
	}
}

func (pia *PrivacyImpactAssessment) GetRiskScore() float64 {
	pia.mu.RLock()
	defer pia.mu.RUnlock()
	return pia.RiskScore
}

func (pia *PrivacyImpactAssessment) GetFindingsByCategory(category FindingCategory) []PIAFinding {
	pia.mu.RLock()
	defer pia.mu.RUnlock()

	results := make([]PIAFinding, 0)
	for _, f := range pia.Findings {
		if f.Category == category {
			results = append(results, f)
		}
	}
	return results
}

func (pia *PrivacyImpactAssessment) GetFindingsBySeverity(severity SeverityLevel) []PIAFinding {
	pia.mu.RLock()
	defer pia.mu.RUnlock()

	results := make([]PIAFinding, 0)
	for _, f := range pia.Findings {
		if f.Severity == severity {
			results = append(results, f)
		}
	}
	return results
}

func (pia *PrivacyImpactAssessment) GetPendingRecommendations() []Recommendation {
	pia.mu.RLock()
	defer pia.mu.RUnlock()

	results := make([]Recommendation, 0)
	for _, r := range pia.Recommendations {
		if r.Status == RecStatusPending || r.Status == RecStatusInProgress {
			results = append(results, r)
		}
	}
	return results
}

func (pia *PrivacyImpactAssessment) Approve(approver string) {
	pia.mu.Lock()
	defer pia.mu.Unlock()
	pia.Status = StatusApproved
	pia.Approver = approver
	pia.ExpiryDate = time.Now().AddDate(1, 0, 0)
}

func (pia *PrivacyImpactAssessment) IsExpired() bool {
	pia.mu.RLock()
	defer pia.mu.RUnlock()
	if pia.Status != StatusApproved {
		return false
	}
	return time.Now().After(pia.ExpiryDate)
}

func (pia *PrivacyImpactAssessment) GetHighPriorityRecommendations() []Recommendation {
	pia.mu.RLock()
	defer pia.mu.RUnlock()

	results := make([]Recommendation, 0)
	for _, r := range pia.Recommendations {
		if r.Priority == PriorityHigh || r.Priority == PriorityCritical {
			results = append(results, r)
		}
	}
	return results
}

type PIAReporter struct {
	assessments map[string]*PrivacyImpactAssessment
	mu          sync.RWMutex
}

func NewPIAReporter() *PIAReporter {
	return &PIAReporter{
		assessments: make(map[string]*PrivacyImpactAssessment),
	}
}

func (pr *PIAReporter) AddAssessment(pia *PrivacyImpactAssessment) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.assessments[pia.ID] = pia
}

func (pr *PIAReporter) GetAssessment(id string) *PrivacyImpactAssessment {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.assessments[id]
}

func (pr *PIAReporter) GetAllAssessments() []*PrivacyImpactAssessment {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	results := make([]*PrivacyImpactAssessment, 0, len(pr.assessments))
	for _, pia := range pr.assessments {
		results = append(results, pia)
	}
	return results
}

func (pr *PIAReporter) GetExpiringAssessments(days int) []*PrivacyImpactAssessment {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	cutoff := time.Now().AddDate(0, 0, days)
	results := make([]*PrivacyImpactAssessment, 0)

	for _, pia := range pr.assessments {
		if pia.Status == StatusApproved && pia.ExpiryDate.Before(cutoff) {
			results = append(results, pia)
		}
	}
	return results
}

func (pr *PIAReporter) GenerateSummary() PIASummary {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	summary := PIASummary{
		TotalAssessments: len(pr.assessments),
	}

	for _, pia := range pr.assessments {
		switch pia.Status {
		case StatusDraft:
			summary.DraftCount++
		case StatusInReview:
			summary.InReviewCount++
		case StatusApproved:
			summary.ApprovedCount++
		case StatusExpired:
			summary.ExpiredCount++
		}

		summary.TotalRiskScore += pia.RiskScore
		summary.TotalFindings += len(pia.Findings)
		summary.TotalRecommendations += len(pia.Recommendations)

		pending := 0
		for _, r := range pia.Recommendations {
			if r.Status == RecStatusPending || r.Status == RecStatusInProgress {
				pending++
			}
		}
		summary.PendingRecommendations += pending
	}

	if summary.TotalAssessments > 0 {
		summary.AverageRiskScore = summary.TotalRiskScore / float64(summary.TotalAssessments)
	}

	return summary
}

type PIASummary struct {
	TotalAssessments       int
	DraftCount              int
	InReviewCount           int
	ApprovedCount           int
	ExpiredCount            int
	TotalFindings           int
	TotalRecommendations    int
	PendingRecommendations  int
	TotalRiskScore          float64
	AverageRiskScore        float64
}

func generateUUID() string {
	return time.Now().Format("20060102150405.000000000")
}

type PIATemplate struct {
	Name            string
	Description     string
	RequiredDataTypes []DataCategory
	AssessmentLevel AssessmentLevel
	CommonFindings  []string
	RequiredDocuments []string
}

func NewPIATemplate(name string) *PIATemplate {
	return &PIATemplate{
		Name:              name,
		RequiredDataTypes: make([]DataCategory, 0),
		CommonFindings:    make([]string, 0),
		RequiredDocuments: make([]string, 0),
	}
}

func (t *PIATemplate) Apply(pia *PrivacyImpactAssessment) {
	pia.mu.Lock()
	defer pia.mu.Unlock()

	pia.AssessmentLevel = t.AssessmentLevel
}

type PIAChecklist struct {
	Items     []ChecklistItem
	Completed int
	Total     int
	mu        sync.RWMutex
}

type ChecklistItem struct {
	ID          string
	Description string
	IsChecked   bool
	Category    string
	Required    bool
}

func NewPIAChecklist() *PIAChecklist {
	return &PIAChecklist{
		Items:     make([]ChecklistItem, 0),
		Completed: 0,
		Total:     0,
	}
}

func (cl *PIAChecklist) AddItem(item ChecklistItem) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.Items = append(cl.Items, item)
	cl.Total++
}

func (cl *PIAChecklist) CheckItem(id string) bool {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	for i, item := range cl.Items {
		if item.ID == id && !item.IsChecked {
			cl.Items[i].IsChecked = true
			cl.Completed++
			return true
		}
	}
	return false
}

func (cl *PIAChecklist) GetCompletionPercentage() float64 {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	if cl.Total == 0 {
		return 0
	}
	return float64(cl.Completed) / float64(cl.Total) * 100
}

func (cl *PIAChecklist) IsComplete() bool {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	for _, item := range cl.Items {
		if item.Required && !item.IsChecked {
			return false
		}
	}
	return true
}
