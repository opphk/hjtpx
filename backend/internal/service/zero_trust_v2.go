package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

type ZeroTrustV2Service interface {
	ContinuousValidate(ctx context.Context, sessionID string, riskScore float64, behaviors []*BehaviorMetric) (*ContinuousValidationResult, error)
	UpdateValidationStatus(ctx context.Context, sessionID string, status *ValidationStatus) error
	GetValidationHistory(ctx context.Context, userID uint, limit, offset int) ([]*ValidationStatus, error)
	AnalyzeNetworkSegment(ctx context.Context, segment *NetworkSegment) (*SegmentAnalysis, error)
	CreateMicrosegment(ctx context.Context, segment *MicrosegmentV2) (string, error)
	UpdateMicrosegment(ctx context.Context, segmentID string, rules []*SegmentRule) error
	DeleteMicrosegment(ctx context.Context, segmentID string) error
	ValidateMicrosegmentAccess(ctx context.Context, segmentID, resourceID, userID string) (*AccessDecision, error)
	CalculateDynamicPermissions(ctx context.Context, userID uint, resource string, context *AccessContext) ([]string, error)
	RevokeDynamicPermission(ctx context.Context, userID uint, permission string) error
	EnrichSASEPolicy(ctx context.Context, policy *SASEPolicy) (*SASEPolicyEnriched, error)
	ValidateSASEPolicy(ctx context.Context, policyID string) (*SASEPolicyValidation, error)
	ProcessSASEEvent(ctx context.Context, event *SASEEvent) (*SASEEventResult, error)
	GetSASEMetrics(ctx context.Context) (*SASEMetrics, error)
}

type BehaviorMetric struct {
	MetricID   string                 `json:"metric_id"`
	MetricType string                 `json:"metric_type"`
	Value      float64                `json:"value"`
	Timestamp  time.Time              `json:"timestamp"`
	Context    map[string]interface{} `json:"context"`
}

type ContinuousValidationResult struct {
	SessionID      string             `json:"session_id"`
	IsValid        bool               `json:"is_valid"`
	RiskLevel      string             `json:"risk_level"`
	RiskScore      float64            `json:"risk_score"`
	Actions        []string           `json:"actions"`
	RequireReauth  bool               `json:"require_reauth"`
	Timestamp      time.Time          `json:"timestamp"`
	BehaviorScore  float64            `json:"behavior_score"`
	AnomalyScore   float64            `json:"anomaly_score"`
	ThreatDetected bool               `json:"threat_detected"`
	Factors        []*RiskFactorV2      `json:"factors"`
}

type RiskFactorV2 struct {
	FactorType    string  `json:"factor_type"`
	FactorName    string  `json:"factor_name"`
	Score         float64 `json:"score"`
	Weight        float64 `json:"weight"`
	Contributing  bool    `json:"contributing"`
	Description   string  `json:"description"`
}

type ValidationStatus struct {
	StatusID     string    `json:"status_id"`
	SessionID    string    `json:"session_id"`
	UserID       uint      `json:"user_id"`
	Status       string    `json:"status"`
	RiskScore    float64   `json:"risk_score"`
	RiskLevel    string    `json:"risk_level"`
	IsValid      bool      `json:"is_valid"`
	LastChecked  time.Time `json:"last_checked"`
	NextCheckAt  time.Time `json:"next_check_at"`
	Factors      []*RiskFactorV2 `json:"factors"`
}

type NetworkSegment struct {
	SegmentID      string            `json:"segment_id"`
	SegmentName    string            `json:"segment_name"`
	SegmentType    string            `json:"segment_type"`
	IPRange        string            `json:"ip_range"`
	VLAN           int               `json:"vlan"`
	TrustLevel     string            `json:"trust_level"`
	Devices        []*DeviceInfo     `json:"devices"`
	Connections    []*ConnectionInfo `json:"connections"`
	SecurityPolicy string            `json:"security_policy"`
}

type DeviceInfo struct {
	DeviceID     string `json:"device_id"`
	DeviceType   string `json:"device_type"`
	IPAddress    string `json:"ip_address"`
	MACAddress   string `json:"mac_address"`
	OS           string `json:"os"`
	LastSeen     time.Time `json:"last_seen"`
	ComplianceStatus string `json:"compliance_status"`
}

type ConnectionInfo struct {
	SourceID     string `json:"source_id"`
	TargetID     string `json:"target_id"`
	Protocol     string `json:"protocol"`
	Port         int    `json:"port"`
	IsAllowed    bool   `json:"is_allowed"`
	LastActivity time.Time `json:"last_activity"`
}

type SegmentAnalysis struct {
	SegmentID     string   `json:"segment_id"`
	TotalDevices  int      `json:"total_devices"`
	TotalConnections int   `json:"total_connections"`
	TrustScore    float64  `json:"trust_score"`
	RiskScore     float64  `json:"risk_score"`
	Threats         []*ThreatInfoV3 `json:"threats"`
	Vulnerabilities []*VulnerabilityInfo `json:"vulnerabilities"`
	Recommendations []string `json:"recommendations"`
	AnalyzedAt    time.Time `json:"analyzed_at"`
}

type ThreatInfoV3 struct {
	ThreatID    string `json:"threat_id"`
	ThreatType  string `json:"threat_type"`
	Severity    string `json:"severity"`
	SourceIP    string `json:"source_ip"`
	Description string `json:"description"`
	DetectedAt  time.Time `json:"detected_at"`
}

type VulnerabilityInfo struct {
	VulnID       string `json:"vuln_id"`
	VulnType     string `json:"vuln_type"`
	Severity     string `json:"severity"`
	AffectedDevice string `json:"affected_device"`
	Description  string `json:"description"`
	CVSS         float64 `json:"cvss"`
}

type MicrosegmentV2 struct {
	SegmentID      string          `json:"segment_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	SourceIP       string          `json:"source_ip"`
	DestIP         string          `json:"dest_ip"`
	Port           int             `json:"port"`
	Protocol       string          `json:"protocol"`
	Application    string          `json:"application"`
	AllowedUsers   []string        `json:"allowed_users"`
	AllowedRoles   []string        `json:"allowed_roles"`
	Rules          []*SegmentRule  `json:"rules"`
	IsActive       bool            `json:"is_active"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Metadata       map[string]interface{} `json:"metadata"`
	TrustLevel     string          `json:"trust_level"`
}

type SegmentRule struct {
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Conditions  []*RuleConditionV2       `json:"conditions"`
	Action      string                 `json:"action"`
	Priority    int                    `json:"priority"`
	IsEnabled   bool                   `json:"is_enabled"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

type RuleConditionV2 struct {
	Field      string `json:"field"`
	Operator   string `json:"operator"`
	Value      string `json:"value"`
}

type AccessDecision struct {
	Allowed        bool      `json:"allowed"`
	SegmentID      string    `json:"segment_id"`
	ResourceID     string    `json:"resource_id"`
	UserID         string    `json:"user_id"`
	Reason         string    `json:"reason"`
	MatchingRules  []string  `json:"matching_rules,omitempty"`
	DeniedRules    []string  `json:"denied_rules,omitempty"`
	ConditionsMet  []*RuleCondition `json:"conditions_met,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

type AccessContext struct {
	UserID       uint                   `json:"user_id"`
	Resource     string                 `json:"resource"`
	Action       string                 `json:"action"`
	Environment  map[string]interface{} `json:"environment"`
	Time         time.Time              `json:"time"`
	Location     string                 `json:"location"`
	DeviceStatus string                 `json:"device_status"`
}

type Permission struct {
	PermissionID   string    `json:"permission_id"`
	PermissionName string    `json:"permission_name"`
	Resource       string    `json:"resource"`
	Actions        []string  `json:"actions"`
	Conditions     []*RuleConditionV2 `json:"conditions"`
	GrantedAt      time.Time `json:"granted_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	GrantedBy      string    `json:"granted_by"`
	IsDynamic      bool      `json:"is_dynamic"`
}

type SASEPolicy struct {
	PolicyID      string                 `json:"policy_id"`
	PolicyName    string                 `json:"policy_name"`
	PolicyType    string                 `json:"policy_type"`
	Priority      int                    `json:"priority"`
	Conditions    []*PolicyCondition     `json:"conditions"`
	Actions       []string               `json:"actions"`
	Targets       []*PolicyTarget        `json:"targets"`
	IsEnabled     bool                   `json:"is_enabled"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type PolicyCondition struct {
	ConditionType string                 `json:"condition_type"`
	Field         string                 `json:"field"`
	Operator      string                 `json:"operator"`
	Value        interface{}            `json:"value"`
}

type PolicyTarget struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}

type SASEPolicyEnriched struct {
	Policy        *SASEPolicy `json:"policy"`
	ComputedRisk  float64     `json:"computed_risk"`
	MatchingUsers int         `json:"matching_users"`
	EstimatedImpact string    `json:"estimated_impact"`
	Conflicts     []*PolicyConflict `json:"conflicts,omitempty"`
	Recommendations []string  `json:"recommendations"`
}

type PolicyConflict struct {
	ConflictType string `json:"conflict_type"`
	Policy1ID    string `json:"policy_1_id"`
	Policy2ID    string `json:"policy_2_id"`
	Description  string `json:"description"`
}

type SASEPolicyValidation struct {
	PolicyID     string   `json:"policy_id"`
	IsValid      bool     `json:"is_valid"`
	Errors       []string `json:"errors"`
	Warnings     []string `json:"warnings"`
	ValidatedAt  time.Time `json:"validated_at"`
}

type SASEEvent struct {
	EventID     string                 `json:"event_id"`
	EventType   string                 `json:"event_type"`
	SourceIP    string                 `json:"source_ip"`
	DestIP      string                 `json:"dest_ip"`
	UserID      uint                   `json:"user_id"`
	DeviceID    string                 `json:"device_id"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource"`
	Result      string                 `json:"result"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
}

type SASEEventResult struct {
	EventID     string                 `json:"event_id"`
	Processed   bool                   `json:"processed"`
	Actions     []string               `json:"actions"`
	Policies    []*PolicyMatch         `json:"policies"`
	RiskScore   float64                `json:"risk_score"`
	ThreatDetected bool                 `json:"threat_detected"`
	Alerted     bool                   `json:"alerted"`
	ProcessedAt time.Time              `json:"processed_at"`
}

type PolicyMatch struct {
	PolicyID   string   `json:"policy_id"`
	PolicyName string   `json:"policy_name"`
	MatchScore float64  `json:"match_score"`
	Actions    []string `json:"actions"`
}

type SASEMetrics struct {
	TotalPolicies    int                    `json:"total_policies"`
	ActivePolicies   int                    `json:"active_policies"`
	TotalEvents      int64                  `json:"total_events"`
	ThreatsDetected  int64                  `json:"threats_detected"`
	AlertsTriggered  int64                  `json:"alerts_triggered"`
	BlockedAccess    int64                  `json:"blocked_access"`
	AllowedAccess    int64                  `json:"allowed_access"`
	AvgRiskScore     float64                `json:"avg_risk_score"`
	TopThreats       []*ThreatInfoV2          `json:"top_threats"`
	MetricsByType    map[string]int64       `json:"metrics_by_type"`
}

type zeroTrustV2Service struct {
	mu               sync.RWMutex
	sessions         map[string]*SessionV2
	microsegments    map[string]*MicrosegmentV2
	permissions      map[uint][]*Permission
	sasePolicies     map[string]*SASEPolicy
	validationHistory map[string][]*ValidationStatus
	saseEvents       []*SASEEvent
	metrics          *SASEMetrics
	maxRiskScore     float64
	criticalRiskScore float64
}

type SessionV2 struct {
	SessionID     string
	UserID        uint
	RiskScore     float64
	RiskLevel     string
	IsValid       bool
	LastChecked   time.Time
	Behaviors     []*BehaviorMetric
	ThreatLevel   string
	AnomalyScore  float64
	BehaviorScore float64
}

var (
	ErrSessionNotFoundV2    = errors.New("session not found")
	ErrSegmentNotFound      = errors.New("segment not found")
	ErrPolicyNotFound       = errors.New("policy not found")
	ErrPermissionDenied     = errors.New("permission denied")
	ErrInvalidRiskScoreV2   = errors.New("invalid risk score")
	ErrSegmentConflict      = errors.New("segment conflict detected")
)

func NewZeroTrustV2Service() ZeroTrustV2Service {
	svc := &zeroTrustV2Service{
		sessions:         make(map[string]*SessionV2),
		microsegments:    make(map[string]*MicrosegmentV2),
		permissions:     make(map[uint][]*Permission),
		sasePolicies:    make(map[string]*SASEPolicy),
		validationHistory: make(map[string][]*ValidationStatus),
		saseEvents:      []*SASEEvent{},
		metrics: &SASEMetrics{
			MetricsByType: make(map[string]int64),
		},
		maxRiskScore:     100.0,
		criticalRiskScore: 75.0,
	}
	svc.initDefaultMicrosegments()
	svc.initDefaultPolicies()
	return svc
}

func (s *zeroTrustV2Service) initDefaultMicrosegments() {
	s.microsegments["internal_services"] = &MicrosegmentV2{
		SegmentID:   "internal_services",
		Name:        "Internal Services",
		Description: "Internal microservices network segment",
		SourceIP:    "10.0.0.0/8",
		TrustLevel:  "high",
		IsActive:    true,
		CreatedAt:   time.Now(),
		Rules: []*SegmentRule{
			{
				RuleID:   "internal_rule_1",
				RuleName: "Allow internal services",
				Conditions: []*RuleConditionV2{
					{Field: "source_ip", Operator: "in", Value: "10.0.0.0/8"},
				},
				Action:    "allow",
				Priority:  100,
				IsEnabled: true,
			},
		},
	}

	s.microsegments["dmz"] = &MicrosegmentV2{
		SegmentID:   "dmz",
		Name:        "DMZ",
		Description: "Demilitarized zone segment",
		SourceIP:    "192.168.1.0/24",
		TrustLevel:  "medium",
		IsActive:    true,
		CreatedAt:   time.Now(),
		Rules: []*SegmentRule{
			{
				RuleID:   "dmz_rule_1",
				RuleName: "Allow HTTPS traffic",
				Conditions: []*RuleConditionV2{
					{Field: "protocol", Operator: "eq", Value: "tcp"},
					{Field: "port", Operator: "in", Value: "443,80"},
				},
				Action:    "allow",
				Priority:  90,
				IsEnabled: true,
			},
			{
				RuleID:   "dmz_rule_2",
				RuleName: "Default deny",
				Conditions: []*RuleConditionV2{},
				Action:    "deny",
				Priority:  10,
				IsEnabled: true,
			},
		},
	}
}

func (s *zeroTrustV2Service) initDefaultPolicies() {
	s.sasePolicies["block_tor"] = &SASEPolicy{
		PolicyID:   "block_tor",
		PolicyName: "Block Tor Exit Nodes",
		PolicyType: "security",
		Priority:   100,
		Conditions: []*PolicyCondition{
			{ConditionType: "ip_list", Field: "source_ip", Operator: "in", Value: "tor_exit_nodes"},
		},
		Actions:    []string{"block", "alert"},
		IsEnabled:  true,
		CreatedAt:  time.Now(),
	}

	s.sasePolicies["require_mfa_high_risk"] = &SASEPolicy{
		PolicyID:   "require_mfa_high_risk",
		PolicyName: "Require MFA for High Risk",
		PolicyType: "access",
		Priority:   95,
		Conditions: []*PolicyCondition{
			{ConditionType: "risk", Field: "risk_score", Operator: "gt", Value: 70},
		},
		Actions:    []string{"step_up_auth", "log"},
		IsEnabled:  true,
		CreatedAt:  time.Now(),
	}
}

func (s *zeroTrustV2Service) ContinuousValidate(ctx context.Context, sessionID string, riskScore float64, behaviors []*BehaviorMetric) (*ContinuousValidationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if riskScore < 0 || riskScore > s.maxRiskScore {
		return nil, ErrInvalidRiskScoreV2
	}

	session := s.getOrCreateSession(sessionID)
	session.RiskScore = riskScore
	session.RiskLevel = s.calculateRiskLevel(riskScore)
	session.LastChecked = time.Now()
	session.Behaviors = behaviors

	behaviorScore := s.calculateBehaviorScore(behaviors)
	anomalyScore := s.calculateAnomalyScore(behaviors, riskScore)

	session.BehaviorScore = behaviorScore
	session.AnomalyScore = anomalyScore

	factors := s.analyzeRiskFactors(riskScore, behaviorScore, anomalyScore, behaviors)

	result := &ContinuousValidationResult{
		SessionID:      sessionID,
		RiskScore:      riskScore,
		RiskLevel:      session.RiskLevel,
		BehaviorScore:  behaviorScore,
		AnomalyScore:   anomalyScore,
		Timestamp:      time.Now(),
		Factors:        factors,
		ThreatDetected: anomalyScore > 60,
		Actions:        []string{},
	}

	if riskScore >= s.criticalRiskScore || anomalyScore > 80 {
		result.IsValid = false
		result.RequireReauth = true
		result.Actions = append(result.Actions, "block_access", "force_reauth", "alert_security")
		session.IsValid = false
	} else if riskScore >= s.criticalRiskScore*0.6 {
		result.IsValid = true
		result.RequireReauth = true
		result.Actions = append(result.Actions, "step_up_auth", "log_enhanced")
	} else {
		result.IsValid = true
		result.RequireReauth = false
		result.Actions = append(result.Actions, "allow_access", "continue_monitoring")
	}

	if anomalyScore > 50 {
		result.Actions = append(result.Actions, "anomaly_detected")
	}

	return result, nil
}

func (s *zeroTrustV2Service) getOrCreateSession(sessionID string) *SessionV2 {
	session, exists := s.sessions[sessionID]
	if !exists {
		session = &SessionV2{
			SessionID:   sessionID,
			RiskScore:   0,
			RiskLevel:   "minimal",
			IsValid:     true,
			LastChecked: time.Now(),
			Behaviors:   []*BehaviorMetric{},
		}
		s.sessions[sessionID] = session
	}
	return session
}

func (s *zeroTrustV2Service) calculateRiskLevel(score float64) string {
	switch {
	case score >= 80:
		return "critical"
	case score >= 60:
		return "high"
	case score >= 40:
		return "medium"
	case score >= 20:
		return "low"
	default:
		return "minimal"
	}
}

func (s *zeroTrustV2Service) calculateBehaviorScore(behaviors []*BehaviorMetric) float64 {
	if len(behaviors) == 0 {
		return 50.0
	}

	totalScore := 0.0
	for _, b := range behaviors {
		totalScore += b.Value
	}

	return totalScore / float64(len(behaviors))
}

func (s *zeroTrustV2Service) calculateAnomalyScore(behaviors []*BehaviorMetric, riskScore float64) float64 {
	anomalyScore := 0.0

	for _, b := range behaviors {
		switch b.MetricType {
		case "unusual_time":
			anomalyScore += 30.0
		case "unusual_location":
			anomalyScore += 40.0
		case "rapid_actions":
			anomalyScore += 20.0
		case "suspicious_pattern":
			anomalyScore += 50.0
		case "credential_stuffing":
			anomalyScore += 60.0
		}
	}

	anomalyScore += riskScore * 0.3

	if anomalyScore > 100 {
		anomalyScore = 100
	}

	return anomalyScore
}

func (s *zeroTrustV2Service) analyzeRiskFactors(riskScore, behaviorScore, anomalyScore float64, behaviors []*BehaviorMetric) []*RiskFactorV2 {
	factors := []*RiskFactorV2{}

	factors = append(factors, &RiskFactorV2{
		FactorType:   "risk_score",
		FactorName:   "Base Risk Score",
		Score:        riskScore,
		Weight:       0.4,
		Contributing: riskScore > 50,
		Description:  "Overall risk score based on user behavior",
	})

	factors = append(factors, &RiskFactorV2{
		FactorType:   "behavior",
		FactorName:   "Behavior Analysis",
		Score:        behaviorScore,
		Weight:       0.3,
		Contributing: behaviorScore < 40,
		Description:  "Score based on behavioral biometrics",
	})

	factors = append(factors, &RiskFactorV2{
		FactorType:   "anomaly",
		FactorName:   "Anomaly Detection",
		Score:        anomalyScore,
		Weight:       0.3,
		Contributing: anomalyScore > 50,
		Description:  "Score based on anomalous patterns",
	})

	for _, b := range behaviors {
		if b.Value < 30 {
			factors = append(factors, &RiskFactorV2{
				FactorType:  "metric",
				FactorName:  b.MetricType,
				Score:       b.Value,
				Weight:      0.2,
				Contributing: true,
				Description: fmt.Sprintf("Low score for metric: %s", b.MetricType),
			})
		}
	}

	return factors
}

func (s *zeroTrustV2Service) UpdateValidationStatus(ctx context.Context, sessionID string, status *ValidationStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if status == nil {
		return errors.New("status cannot be nil")
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		session = &SessionV2{
			SessionID: sessionID,
		}
		s.sessions[sessionID] = session
	}

	session.RiskScore = status.RiskScore
	session.RiskLevel = status.RiskLevel
	session.IsValid = status.IsValid
	session.LastChecked = time.Now()

	status.StatusID = fmt.Sprintf("vs_%d", time.Now().UnixNano())
	status.SessionID = sessionID
	status.LastChecked = time.Now()

	if status.NextCheckAt.IsZero() {
		status.NextCheckAt = time.Now().Add(5 * time.Minute)
	}

	s.validationHistory[sessionID] = append(s.validationHistory[sessionID], status)

	return nil
}

func (s *zeroTrustV2Service) GetValidationHistory(ctx context.Context, userID uint, limit, offset int) ([]*ValidationStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ValidationStatus

	for _, history := range s.validationHistory {
		for _, status := range history {
			if userID == 0 || status.UserID == userID {
				result = append(result, status)
			}
		}
	}

	if offset >= len(result) {
		return []*ValidationStatus{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

func (s *zeroTrustV2Service) AnalyzeNetworkSegment(ctx context.Context, segment *NetworkSegment) (*SegmentAnalysis, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if segment == nil {
		return nil, errors.New("segment cannot be nil")
	}

	analysis := &SegmentAnalysis{
		SegmentID:       segment.SegmentID,
		TotalDevices:    len(segment.Devices),
		TotalConnections: len(segment.Connections),
		AnalyzedAt:      time.Now(),
		Threats:        []*ThreatInfoV3{},
		Vulnerabilities: []*VulnerabilityInfo{},
		Recommendations: []string{},
	}

	analysis.TrustScore = s.calculateSegmentTrustScore(segment)
	analysis.RiskScore = 100 - analysis.TrustScore

	for _, device := range segment.Devices {
		if device.ComplianceStatus == "non_compliant" {
			analysis.Vulnerabilities = append(analysis.Vulnerabilities, &VulnerabilityInfo{
				VulnID:          fmt.Sprintf("vuln_%s", device.DeviceID),
				VulnType:        "compliance",
				Severity:        "high",
				AffectedDevice:  device.DeviceID,
				Description:     "Device does not meet compliance requirements",
				CVSS:            7.5,
			})
		}
	}

	for _, conn := range segment.Connections {
		if !conn.IsAllowed {
			analysis.Threats = append(analysis.Threats, &ThreatInfoV3{
				ThreatID:    fmt.Sprintf("threat_%s", conn.SourceID),
				ThreatType:  "unauthorized_access",
				Severity:    "medium",
				SourceIP:    "unknown",
				Description: fmt.Sprintf("Unauthorized connection attempt to %s", conn.TargetID),
				DetectedAt:  time.Now(),
			})
		}
	}

	if analysis.TrustScore < 70 {
		analysis.Recommendations = append(analysis.Recommendations, "Implement stricter access controls")
	}
	if len(analysis.Vulnerabilities) > 0 {
		analysis.Recommendations = append(analysis.Recommendations, "Address device compliance issues")
	}
	if len(analysis.Threats) > 0 {
		analysis.Recommendations = append(analysis.Recommendations, "Investigate unauthorized connection attempts")
	}

	return analysis, nil
}

func (s *zeroTrustV2Service) calculateSegmentTrustScore(segment *NetworkSegment) float64 {
	trustScore := 80.0

	switch segment.TrustLevel {
	case "critical":
		trustScore = 95.0
	case "high":
		trustScore = 85.0
	case "medium":
		trustScore = 70.0
	case "low":
		trustScore = 50.0
	}

	nonCompliantDevices := 0
	for _, device := range segment.Devices {
		if device.ComplianceStatus != "compliant" {
			nonCompliantDevices++
		}
	}

	if len(segment.Devices) > 0 {
		complianceRatio := float64(len(segment.Devices)-nonCompliantDevices) / float64(len(segment.Devices))
		trustScore *= complianceRatio
	}

	blockedConnections := 0
	for _, conn := range segment.Connections {
		if !conn.IsAllowed {
			blockedConnections++
		}
	}
	if len(segment.Connections) > 0 {
		trustScore -= float64(blockedConnections) * 2
	}

	if trustScore < 0 {
		trustScore = 0
	}

	return trustScore
}

func (s *zeroTrustV2Service) CreateMicrosegment(ctx context.Context, segment *MicrosegmentV2) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if segment == nil {
		return "", errors.New("segment cannot be nil")
	}

	if segment.SegmentID == "" {
		segment.SegmentID = fmt.Sprintf("seg_%d", time.Now().UnixNano())
	}

	if segment.CreatedAt.IsZero() {
		segment.CreatedAt = time.Now()
	}
	segment.UpdatedAt = time.Now()

	segment.IsActive = true

	s.microsegments[segment.SegmentID] = segment

	return segment.SegmentID, nil
}

func (s *zeroTrustV2Service) UpdateMicrosegment(ctx context.Context, segmentID string, rules []*SegmentRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	segment, exists := s.microsegments[segmentID]
	if !exists {
		return ErrSegmentNotFound
	}

	segment.Rules = rules
	segment.UpdatedAt = time.Now()

	return nil
}

func (s *zeroTrustV2Service) DeleteMicrosegment(ctx context.Context, segmentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.microsegments[segmentID]; !exists {
		return ErrSegmentNotFound
	}

	delete(s.microsegments, segmentID)

	return nil
}

func (s *zeroTrustV2Service) ValidateMicrosegmentAccess(ctx context.Context, segmentID, resourceID, userID string) (*AccessDecision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	segment, exists := s.microsegments[segmentID]
	if !exists {
		return &AccessDecision{
			Allowed:    false,
			Reason:     "segment_not_found",
			Timestamp:  time.Now(),
		}, ErrSegmentNotFound
	}

	if !segment.IsActive {
		return &AccessDecision{
			Allowed:    false,
			SegmentID:  segmentID,
			Reason:     "segment_inactive",
			Timestamp:  time.Now(),
		}, nil
	}

	userAllowed := false
	for _, allowedUser := range segment.AllowedUsers {
		if allowedUser == userID {
			userAllowed = true
			break
		}
	}

	if !userAllowed && len(segment.AllowedUsers) > 0 {
		return &AccessDecision{
			Allowed:    false,
			SegmentID:  segmentID,
			ResourceID: resourceID,
			UserID:     userID,
			Reason:     "user_not_in_allowed_list",
			Timestamp:  time.Now(),
		}, nil
	}

	matchingRules := []string{}
	deniedRules := []string{}

	for _, rule := range segment.Rules {
		if !rule.IsEnabled {
			continue
		}

		if rule.ExpiresAt != nil && time.Now().After(*rule.ExpiresAt) {
			continue
		}

		matched := s.evaluateRuleConditions(rule.Conditions, userID, resourceID)

		if matched {
			if rule.Action == "allow" {
				matchingRules = append(matchingRules, rule.RuleID)
				userAllowed = true
			} else if rule.Action == "deny" {
				deniedRules = append(deniedRules, rule.RuleID)
				userAllowed = false
			}
		}
	}

	decision := &AccessDecision{
		Allowed:       userAllowed,
		SegmentID:     segmentID,
		ResourceID:    resourceID,
		UserID:        userID,
		Reason:        "policy_evaluated",
		MatchingRules: matchingRules,
		DeniedRules:   deniedRules,
		Timestamp:     time.Now(),
	}

	if userAllowed {
		decision.Reason = "access_allowed_by_policy"
	} else {
		decision.Reason = "access_denied_by_policy"
	}

	return decision, nil
}

func (s *zeroTrustV2Service) evaluateRuleConditions(conditions []*RuleConditionV2, userID, resourceID string) bool {
	if len(conditions) == 0 {
		return true
	}

	for _, cond := range conditions {
		switch cond.Field {
		case "user_id":
			if cond.Operator == "eq" && userID != cond.Value {
				return false
			}
			if cond.Operator == "neq" && userID == cond.Value {
				return false
			}
		case "resource_id":
			if cond.Operator == "eq" && resourceID != cond.Value {
				return false
			}
		}
	}

	return true
}

func (s *zeroTrustV2Service) CalculateDynamicPermissions(ctx context.Context, userID uint, resource string, accessCtx *AccessContext) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if accessCtx == nil {
		accessCtx = &AccessContext{}
	}

	permissions := []string{}

	basePermissions := s.getBasePermissions(userID, resource)
	permissions = append(permissions, basePermissions...)

	if accessCtx.DeviceStatus == "compliant" {
		permissions = append(permissions, "device_access")
	}

	if accessCtx.Location != "" && accessCtx.Location != "unknown" {
		permissions = append(permissions, "location_verified")
	}

	if accessCtx.Time.Hour() >= 9 && accessCtx.Time.Hour() <= 17 {
		permissions = append(permissions, "business_hours")
	}

	if len(permissions) > 0 {
		permissions = append(permissions, "basic_access")
	}

	return permissions, nil
}

func (s *zeroTrustV2Service) getBasePermissions(userID uint, resource string) []string {
	perms := s.permissions[userID]
	basePerms := []string{}

	for _, perm := range perms {
		if perm.Resource == resource || perm.Resource == "*" {
			if perm.ExpiresAt.IsZero() || time.Now().Before(perm.ExpiresAt) {
				basePerms = append(basePerms, perm.PermissionName)
			}
		}
	}

	if len(basePerms) == 0 {
		basePerms = []string{"read"}
	}

	return basePerms
}

func (s *zeroTrustV2Service) RevokeDynamicPermission(ctx context.Context, userID uint, permission string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	perms := s.permissions[userID]
	for i, perm := range perms {
		if perm.PermissionName == permission {
			perm.ExpiresAt = time.Now()
			s.permissions[userID][i] = perm
			return nil
		}
	}

	return ErrPermissionDenied
}

func (s *zeroTrustV2Service) EnrichSASEPolicy(ctx context.Context, policy *SASEPolicy) (*SASEPolicyEnriched, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if policy == nil {
		return nil, errors.New("policy cannot be nil")
	}

	enriched := &SASEPolicyEnriched{
		Policy:          policy,
		ComputedRisk:    50.0,
		MatchingUsers:   0,
		Conflicts:       []*PolicyConflict{},
		Recommendations: []string{},
	}

	switch policy.PolicyType {
	case "security":
		enriched.ComputedRisk = 70.0
		enriched.EstimatedImpact = "high"
	case "access":
		enriched.ComputedRisk = 40.0
		enriched.EstimatedImpact = "medium"
	case "compliance":
		enriched.ComputedRisk = 60.0
		enriched.EstimatedImpact = "medium"
	}

	enriched.MatchingUsers = len(policy.Targets) * 10

	for _, existingPolicy := range s.sasePolicies {
		if existingPolicy.PolicyID == policy.PolicyID {
			continue
		}

		if existingPolicy.Priority == policy.Priority && existingPolicy.PolicyType == policy.PolicyType {
			conflict := &PolicyConflict{
				ConflictType: "priority_conflict",
				Policy1ID:    policy.PolicyID,
				Policy2ID:    existingPolicy.PolicyID,
				Description:  "Policies have same priority and type",
			}
			enriched.Conflicts = append(enriched.Conflicts, conflict)
		}
	}

	if enriched.ComputedRisk >= 70 {
		enriched.Recommendations = append(enriched.Recommendations, "Consider reducing scope of high-risk policy")
	}
	if len(enriched.Conflicts) > 0 {
		enriched.Recommendations = append(enriched.Recommendations, "Resolve policy conflicts before deployment")
	}

	return enriched, nil
}

func (s *zeroTrustV2Service) ValidateSASEPolicy(ctx context.Context, policyID string) (*SASEPolicyValidation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policy, exists := s.sasePolicies[policyID]
	if !exists {
		return nil, ErrPolicyNotFound
	}

	validation := &SASEPolicyValidation{
		PolicyID:    policyID,
		IsValid:     true,
		Errors:      []string{},
		Warnings:    []string{},
		ValidatedAt: time.Now(),
	}

	if len(policy.Conditions) == 0 {
		validation.Warnings = append(validation.Warnings, "Policy has no conditions")
	}

	if len(policy.Actions) == 0 {
		validation.Errors = append(validation.Errors, "Policy must have at least one action")
		validation.IsValid = false
	}

	if policy.Priority <= 0 {
		validation.Warnings = append(validation.Warnings, "Policy priority should be positive")
	}

	for _, condition := range policy.Conditions {
		if condition.Field == "" {
			validation.Errors = append(validation.Errors, "Condition field cannot be empty")
			validation.IsValid = false
		}
		if condition.Operator == "" {
			validation.Errors = append(validation.Errors, "Condition operator cannot be empty")
			validation.IsValid = false
		}
	}

	return validation, nil
}

func (s *zeroTrustV2Service) ProcessSASEEvent(ctx context.Context, event *SASEEvent) (*SASEEventResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if event == nil {
		return nil, errors.New("event cannot be nil")
	}

	if event.EventID == "" {
		event.EventID = fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	s.saseEvents = append(s.saseEvents, event)
	s.metrics.TotalEvents++

	result := &SASEEventResult{
		EventID:       event.EventID,
		Processed:     true,
		Actions:       []string{},
		Policies:      []*PolicyMatch{},
		RiskScore:     0,
		ThreatDetected: false,
		Alerted:      false,
		ProcessedAt:   time.Now(),
	}

	riskScore := 0.0

	for _, policy := range s.sasePolicies {
		if !policy.IsEnabled {
			continue
		}

		matchScore := s.calculatePolicyMatchScore(policy, event)
		if matchScore > 0 {
			policyMatch := &PolicyMatch{
				PolicyID:   policy.PolicyID,
				PolicyName: policy.PolicyName,
				MatchScore: matchScore,
				Actions:    policy.Actions,
			}
			result.Policies = append(result.Policies, policyMatch)

			riskScore += matchScore * 0.3

			for _, action := range policy.Actions {
				result.Actions = append(result.Actions, action)
				if action == "block" {
					s.metrics.BlockedAccess++
				}
				if action == "alert" {
					result.Alerted = true
					s.metrics.AlertsTriggered++
				}
			}
		}
	}

	if riskScore > 50 {
		result.ThreatDetected = true
		s.metrics.ThreatsDetected++
	}

	result.RiskScore = riskScore
	s.metrics.AvgRiskScore = (s.metrics.AvgRiskScore + riskScore) / 2

	s.metrics.MetricsByType[event.EventType]++
	s.metrics.AllowedAccess++

	return result, nil
}

func (s *zeroTrustV2Service) calculatePolicyMatchScore(policy *SASEPolicy, event *SASEEvent) float64 {
	score := 0.0

	for _, condition := range policy.Conditions {
		switch condition.ConditionType {
		case "ip_list":
			if event.SourceIP != "" {
				score += 80.0
			}
		case "risk":
			if event.Metadata != nil {
				if riskScore, ok := event.Metadata["risk_score"].(float64); ok {
					threshold := 50.0
					if riskScore > threshold {
						score += 90.0
					}
				}
			}
		case "user":
			if event.UserID > 0 {
				score += 70.0
			}
		case "device":
			if event.DeviceID != "" {
				score += 60.0
			}
		}
	}

	if len(policy.Targets) > 0 {
		for _, target := range policy.Targets {
			if target.TargetType == "resource" && target.TargetID == event.Resource {
				score += 50.0
			}
		}
	}

	if score > 100 {
		score = 100
	}

	return score
}

func (s *zeroTrustV2Service) GetSASEMetrics(ctx context.Context) (*SASEMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := &SASEMetrics{
		TotalPolicies:   len(s.sasePolicies),
		ActivePolicies: 0,
		TotalEvents:    s.metrics.TotalEvents,
		ThreatsDetected: s.metrics.ThreatsDetected,
		AlertsTriggered: s.metrics.AlertsTriggered,
		BlockedAccess:  s.metrics.BlockedAccess,
		AllowedAccess:  s.metrics.AllowedAccess,
		AvgRiskScore:   s.metrics.AvgRiskScore,
		TopThreats:     []*ThreatInfoV2{},
		MetricsByType:  make(map[string]int64),
	}

	for _, policy := range s.sasePolicies {
		if policy.IsEnabled {
			metrics.ActivePolicies++
		}
	}

	for k, v := range s.metrics.MetricsByType {
		metrics.MetricsByType[k] = v
	}

	return metrics, nil
}

func computeHash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
