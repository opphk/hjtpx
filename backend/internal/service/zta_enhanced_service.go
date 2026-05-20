package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

type ZTAContinuousValidationEngine interface {
	StartContinuousValidation(ctx context.Context, sessionID string, config *ValidationConfig) (*ValidationSession, error)
	StopContinuousValidation(ctx context.Context, sessionID string) error
	GetValidationStatus(ctx context.Context, sessionID string) (*ValidationStatus, error)
	ValidateAccessRequest(ctx context.Context, request *AccessRequest) (*AccessDecision, error)
	ProcessRiskSignal(ctx context.Context, sessionID string, signal *RiskSignal) error
	GetRiskScore(ctx context.Context, sessionID string) (*RiskScoreResult, error)
}

type ValidationConfig struct {
	SessionID            string        `json:"session_id"`
	ValidationInterval   time.Duration `json:"validation_interval"`
	RiskThresholds       RiskThresholds `json:"risk_thresholds"`
	EnabledFactors       []string      `json:"enabled_factors"`
	MaxGracePeriod       time.Duration `json:"max_grace_period"`
	AutoRevokeOnHighRisk bool          `json:"auto_revoke_on_high_risk"`
}

type RiskThresholds struct {
	Low      float64 `json:"low"`
	Medium   float64 `json:"medium"`
	High     float64 `json:"high"`
	Critical float64 `json:"critical"`
}

type ValidationSession struct {
	SessionID         string               `json:"session_id"`
	UserID            uint                 `json:"user_id"`
	StartTime         time.Time            `json:"start_time"`
	LastValidation    time.Time            `json:"last_validation"`
	Status            string               `json:"status"`
	CurrentRiskScore  float64              `json:"current_risk_score"`
	ValidationHistory []*ValidationResult  `json:"validation_history"`
	ActiveFactors     []string             `json:"active_factors"`
	GracePeriodEnd    *time.Time           `json:"grace_period_end"`
}

type ValidationResult struct {
	Timestamp      time.Time `json:"timestamp"`
	RiskScore      float64   `json:"risk_score"`
	Decision       string    `json:"decision"`
	FactorsChecked []string  `json:"factors_checked"`
	Details        string    `json:"details"`
}

type ValidationStatus struct {
	SessionID       string    `json:"session_id"`
	IsActive       bool      `json:"is_active"`
	CurrentRisk    float64   `json:"current_risk"`
	RiskLevel      string    `json:"risk_level"`
	LastCheck      time.Time `json:"last_check"`
	NextCheck      time.Time `json:"next_check"`
	FailedAttempts int      `json:"failed_attempts"`
	StatusMessage  string    `json:"status_message"`
}

type AccessRequest struct {
	RequestID    string            `json:"request_id"`
	UserID       uint              `json:"user_id"`
	SessionID    string            `json:"session_id"`
	Resource     string            `json:"resource"`
	Action       string            `json:"action"`
	Context      map[string]string `json:"context"`
	Timestamp    time.Time         `json:"timestamp"`
	SourceIP     string            `json:"source_ip"`
	UserAgent    string            `json:"user_agent"`
}

type AccessDecision struct {
	RequestID     string    `json:"request_id"`
	Decision      string    `json:"decision"`
	RiskScore     float64   `json:"risk_score"`
	RiskLevel     string    `json:"risk_level"`
	RequiredAuth  []string  `json:"required_auth"`
	Conditions    []string  `json:"conditions"`
	ExpiresAt     time.Time `json:"expires_at"`
	Reason        string    `json:"reason"`
}

type RiskSignal struct {
	SignalID   string                 `json:"signal_id"`
	SessionID  string                 `json:"session_id"`
	SignalType string                 `json:"signal_type"`
	Severity   string                 `json:"severity"`
	Source     string                 `json:"source"`
	Data       map[string]interface{} `json:"data"`
	Timestamp  time.Time              `json:"timestamp"`
	Processed  bool                   `json:"processed"`
}

type RiskScoreResult struct {
	SessionID       string             `json:"session_id"`
	TotalScore      float64            `json:"total_score"`
	RiskLevel       string             `json:"risk_level"`
	Breakdown       map[string]float64 `json:"breakdown"`
	Factors         []string           `json:"factors"`
	Timestamp       time.Time          `json:"timestamp"`
	Recommendations []string          `json:"recommendations"`
}

type MicrosegmentNetworkService interface {
	CreateMicrosegment(ctx context.Context, segment *MicrosegmentDefinition) (*Microsegment, error)
	DeleteMicrosegment(ctx context.Context, segmentID string) error
	UpdateMicrosegment(ctx context.Context, segmentID string, updates *MicrosegmentUpdate) error
	ListMicrosegments(ctx context.Context, filters *MicrosegmentFilter) ([]*Microsegment, error)
	ValidateTraffic(ctx context.Context, sourceID, destID string, protocol string) (*TrafficDecision, error)
	IsolateWorkload(ctx context.Context, workloadID string, reason string) error
	RestoreWorkload(ctx context.Context, workloadID string) error
	GetWorkloadStatus(ctx context.Context, workloadID string) (*WorkloadStatus, error)
}

type MicrosegmentDefinition struct {
	SegmentID      string            `json:"segment_id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	SourceWorkload string           `json:"source_workload"`
	DestWorkload   string           `json:"dest_workload"`
	Port          int               `json:"port"`
	Protocol      string            `json:"protocol"`
	AllowedUsers  []string          `json:"allowed_users"`
	Tags          map[string]string `json:"tags"`
	Policy        *SegmentPolicy    `json:"policy"`
}

type SegmentPolicy struct {
	AllowByDefault    bool         `json:"allow_by_default"`
	AllowedSources   []string     `json:"allowed_sources"`
	DeniedSources    []string      `json:"denied_sources"`
	TimeRestrictions *TimeWindow   `json:"time_restrictions"`
	MaxConnections   int          `json:"max_connections"`
	InspectionLevel  string       `json:"inspection_level"`
}

type TimeWindow struct {
	StartHour int    `json:"start_hour"`
	EndHour   int    `json:"end_hour"`
	Days      []int  `json:"days"`
}

type MicrosegmentUpdate struct {
	Name         *string            `json:"name,omitempty"`
	AllowedUsers []string           `json:"allowed_users,omitempty"`
	Tags         map[string]string  `json:"tags,omitempty"`
	Policy       *SegmentPolicy     `json:"policy,omitempty"`
	IsActive     *bool              `json:"is_active,omitempty"`
}

type MicrosegmentFilter struct {
	WorkloadID string            `json:"workload_id"`
	Protocol   string            `json:"protocol"`
	Tags       map[string]string `json:"tags"`
	IsActive   *bool             `json:"is_active"`
}

type TrafficDecision struct {
	Allowed     bool      `json:"allowed"`
	Reason      string    `json:"reason"`
	MatchedRule string    `json:"matched_rule"`
	RiskLevel   string    `json:"risk_level"`
	Timestamp   time.Time `json:"timestamp"`
	Metadata    map[string]string `json:"metadata"`
}

type WorkloadStatus struct {
	WorkloadID       string     `json:"workload_id"`
	IsIsolated       bool       `json:"is_isolated"`
	IsolationReason  string     `json:"isolation_reason"`
	IsolatedAt       *time.Time `json:"isolated_at"`
	Connections      int        `json:"connections"`
	LastActivity     time.Time  `json:"last_activity"`
}

type LeastPrivilegeEngine interface {
	ComputePermissions(ctx context.Context, userID uint, resource string, context *PermissionContext) (*ComputedPermissions, error)
	ReviewPermissions(ctx context.Context, userID uint) (*PermissionReview, error)
	GrantTemporaryPermission(ctx context.Context, grant *PermissionGrant) error
	RevokePermission(ctx context.Context, userID uint, permission string) error
	GetPermissionAudit(ctx context.Context, userID uint, start, end time.Time) (*PermissionAudit, error)
}

type PermissionContext struct {
	TimeOfDay    time.Time `json:"time_of_day"`
	DayOfWeek    int       `json:"day_of_week"`
	Location     string    `json:"location"`
	DeviceType   string    `json:"device_type"`
	NetworkType  string    `json:"network_type"`
	AuthMethods  []string  `json:"auth_methods"`
	LastActivity time.Time `json:"last_activity"`
	RiskScore    float64   `json:"risk_score"`
}

type ComputedPermissions struct {
	UserID      uint      `json:"user_id"`
	Resource   string    `json:"resource"`
	Permissions []string `json:"permissions"`
	Conditions  []string `json:"conditions"`
	ExpiresAt   time.Time `json:"expires_at"`
	ComputedAt  time.Time `json:"computed_at"`
	Confidence  float64   `json:"confidence"`
	FactorsUsed []string  `json:"factors_used"`
}

type PermissionGrant struct {
	GrantID    string    `json:"grant_id"`
	UserID     uint      `json:"user_id"`
	Permission string    `json:"permission"`
	Resource   string    `json:"resource"`
	GrantedBy  string    `json:"granted_by"`
	GrantedAt  time.Time `json:"granted_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Reason     string    `json:"reason"`
	Revoked    bool      `json:"revoked"`
}

type PermissionReview struct {
	UserID           uint             `json:"user_id"`
	ReviewDate       time.Time        `json:"review_date"`
	Permissions      []*PermissionItem `json:"permissions"`
	UnusedCount      int              `json:"unused_count"`
	Overprivileged   bool             `json:"overprivileged"`
	Recommendations  []string         `json:"recommendations"`
}

type PermissionItem struct {
	Permission string    `json:"permission"`
	Resource   string    `json:"resource"`
	LastUsed   time.Time `json:"last_used"`
	UseCount   int       `json:"use_count"`
	RiskLevel  string    `json:"risk_level"`
}

type PermissionAudit struct {
	UserID       uint          `json:"user_id"`
	StartDate    time.Time     `json:"start_date"`
	EndDate      time.Time     `json:"end_date"`
	Entries      []*AuditEntry `json:"entries"`
	TotalGrants  int           `json:"total_grants"`
	TotalRevokes int           `json:"total_revokes"`
}

type AuditEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	Action     string    `json:"action"`
	Permission string    `json:"permission"`
	Resource   string    `json:"resource"`
	GrantedBy  string    `json:"granted_by"`
	Reason     string    `json:"reason"`
}

type SASEService interface {
	RegisterEdgeLocation(ctx context.Context, location *EdgeLocation) error
	GetNearestEdge(ctx context.Context, coordinates *Coordinates) (*EdgeLocation, error)
	RouteTraffic(ctx context.Context, request *TrafficRouteRequest) (*TrafficRouteResponse, error)
	SyncSecurityPolicy(ctx context.Context, policy *SecurityPolicy) error
	GetEdgeStatus(ctx context.Context, edgeID string) (*EdgeStatus, error)
	ProcessEdgeEvent(ctx context.Context, event *EdgeEvent) error
}

type EdgeLocation struct {
	EdgeID        string       `json:"edge_id"`
	Name          string       `json:"name"`
	Region        string       `json:"region"`
	Coordinates   *Coordinates `json:"coordinates"`
	Capacity      int          `json:"capacity"`
	CurrentLoad   int          `json:"current_load"`
	Services      []string     `json:"services"`
	IsActive      bool         `json:"is_active"`
	LastHeartbeat time.Time    `json:"last_heartbeat"`
}

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type TrafficRouteRequest struct {
	RequestID    string       `json:"request_id"`
	UserLocation *Coordinates `json:"user_location"`
	Destination  string       `json:"destination"`
	ServiceType  string       `json:"service_type"`
	Priority     string       `json:"priority"`
}

type TrafficRouteResponse struct {
	RequestID        string        `json:"request_id"`
	SelectedEdge     *EdgeLocation `json:"selected_edge"`
	RoutePath        []string      `json:"route_path"`
	EstimatedLatency int           `json:"estimated_latency"`
	SecurityLevel    string        `json:"security_level"`
	Decision         string        `json:"decision"`
}

type SecurityPolicy struct {
	PolicyID       string        `json:"policy_id"`
	Version        int           `json:"version"`
	Rules          []*PolicyRule `json:"rules"`
	EffectiveFrom  time.Time     `json:"effective_from"`
	CreatedBy      string        `json:"created_by"`
	LastUpdated    time.Time     `json:"last_updated"`
}

type PolicyRule struct {
	RuleID     string   `json:"rule_id"`
	Priority   int      `json:"priority"`
	Conditions []string `json:"conditions"`
	Action     string   `json:"action"`
	Effect     string   `json:"effect"`
}

type EdgeStatus struct {
	EdgeID             string    `json:"edge_id"`
	IsHealthy          bool      `json:"is_healthy"`
	CPUUsage           float64   `json:"cpu_usage"`
	MemoryUsage        float64   `json:"memory_usage"`
	NetworkIn          int64     `json:"network_in"`
	NetworkOut         int64     `json:"network_out"`
	ActiveConnections  int       `json:"active_connections"`
	LastHeartbeat      time.Time `json:"last_heartbeat"`
}

type EdgeEvent struct {
	EventID     string                 `json:"event_id"`
	EdgeID      string                 `json:"edge_id"`
	EventType   string                 `json:"event_type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
}

type ztaEnhancedService struct {
	mu                    sync.RWMutex
	validationSessions    map[string]*ValidationSession
	microsegments         map[string]*Microsegment
	workloads             map[string]*WorkloadStatus
	permissions           map[uint][]*ComputedPermissions
	permissionGrants      map[string]*PermissionGrant
	edgeLocations         map[string]*EdgeLocation
	securityPolicies      map[string]*SecurityPolicy
	riskSignals           map[string][]*RiskSignal
	maxRiskScore          float64
	criticalRiskScore    float64
}

var (
	ErrSessionNotActive    = errors.New("session not active")
	ErrSegmentNotFound     = errors.New("microsegment not found")
	ErrWorkloadNotFound    = errors.New("workload not found")
	ErrPermissionDenied    = errors.New("permission denied")
	ErrEdgeNotFound        = errors.New("edge location not found")
	ErrInvalidRiskSignal   = errors.New("invalid risk signal")
)

func NewZTAEnhancedService() ZTAContinuousValidationEngine {
	svc := &ztaEnhancedService{
		validationSessions: make(map[string]*ValidationSession),
		microsegments:     make(map[string]*Microsegment),
		workloads:          make(map[string]*WorkloadStatus),
		permissions:        make(map[uint][]*ComputedPermissions),
		permissionGrants:   make(map[string]*PermissionGrant),
		edgeLocations:      make(map[string]*EdgeLocation),
		securityPolicies:   make(map[string]*SecurityPolicy),
		riskSignals:        make(map[string][]*RiskSignal),
		maxRiskScore:        100.0,
		criticalRiskScore:   80.0,
	}
	svc.initEdgeLocations()
	return svc
}

func (s *ztaEnhancedService) initEdgeLocations() {
	s.edgeLocations["edge-us-east"] = &EdgeLocation{
		EdgeID:        "edge-us-east",
		Name:          "US East",
		Region:        "us-east-1",
		Coordinates:   &Coordinates{Latitude: 37.4316, Longitude: -78.6569},
		Capacity:      10000,
		CurrentLoad:   2500,
		Services:      []string{"auth", "validation", "analytics"},
		IsActive:      true,
		LastHeartbeat: time.Now(),
	}
}

func (s *ztaEnhancedService) StartContinuousValidation(ctx context.Context, sessionID string, config *ValidationConfig) (*ValidationSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config == nil {
		config = &ValidationConfig{
			ValidationInterval: 5 * time.Minute,
			RiskThresholds: RiskThresholds{
				Low:      20,
				Medium:   40,
				High:     60,
				Critical: 80,
			},
			EnabledFactors:      []string{"device", "location", "behavior", "time"},
			MaxGracePeriod:      30 * time.Minute,
			AutoRevokeOnHighRisk: true,
		}
	}

	session := &ValidationSession{
		SessionID:         sessionID,
		StartTime:         time.Now(),
		LastValidation:    time.Now(),
		Status:            "active",
		CurrentRiskScore:  0,
		ValidationHistory: []*ValidationResult{},
		ActiveFactors:     config.EnabledFactors,
	}

	s.validationSessions[sessionID] = session

	return session, nil
}

func (s *ztaEnhancedService) StopContinuousValidation(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.validationSessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	session.Status = "stopped"

	return nil
}

func (s *ztaEnhancedService) GetValidationStatus(ctx context.Context, sessionID string) (*ValidationStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.validationSessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	status := &ValidationStatus{
		SessionID:      sessionID,
		IsActive:       session.Status == "active",
		CurrentRisk:    session.CurrentRiskScore,
		RiskLevel:      s.calculateRiskLevel(session.CurrentRiskScore),
		LastCheck:      session.LastValidation,
		NextCheck:      session.LastValidation.Add(5 * time.Minute),
		FailedAttempts: 0,
		StatusMessage:  "Validation in progress",
	}

	return status, nil
}

func (s *ztaEnhancedService) ValidateAccessRequest(ctx context.Context, request *AccessRequest) (*AccessDecision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if request.RequestID == "" {
		request.RequestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	request.Timestamp = time.Now()

	session, exists := s.validationSessions[request.SessionID]
	if !exists {
		return &AccessDecision{
			RequestID: request.RequestID,
			Decision:  "deny",
			Reason:    "No active validation session",
		}, nil
	}

	riskScore := s.computeAccessRiskScore(request, session)

	decision := &AccessDecision{
		RequestID:    request.RequestID,
		RiskScore:    riskScore,
		RiskLevel:    s.calculateRiskLevel(riskScore),
		RequiredAuth: []string{},
		Conditions:   []string{},
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}

	if riskScore >= s.criticalRiskScore {
		decision.Decision = "deny"
		decision.Reason = "Critical risk level detected"
		decision.RequiredAuth = []string{"step_up_auth", "identity_verification"}
	} else if riskScore >= 60 {
		decision.Decision = "challenge"
		decision.Reason = "Elevated risk - additional verification required"
		decision.RequiredAuth = []string{"mfa"}
	} else if riskScore >= 40 {
		decision.Decision = "allow_with_monitoring"
		decision.Reason = "Medium risk - allowed with enhanced monitoring"
	} else {
		decision.Decision = "allow"
		decision.Reason = "Risk within acceptable threshold"
	}

	session.CurrentRiskScore = riskScore
	session.LastValidation = time.Now()

	validationResult := &ValidationResult{
		Timestamp:      time.Now(),
		RiskScore:      riskScore,
		Decision:       decision.Decision,
		FactorsChecked: session.ActiveFactors,
		Details:        decision.Reason,
	}
	session.ValidationHistory = append(session.ValidationHistory, validationResult)

	return decision, nil
}

func (s *ztaEnhancedService) ProcessRiskSignal(ctx context.Context, sessionID string, signal *RiskSignal) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if signal.SignalID == "" {
		signal.SignalID = fmt.Sprintf("sig-%d", time.Now().UnixNano())
	}
	signal.Timestamp = time.Now()
	signal.Processed = false

	session, exists := s.validationSessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	session.CurrentRiskScore += s.calculateSignalImpact(signal)

	if session.CurrentRiskScore > s.maxRiskScore {
		session.CurrentRiskScore = s.maxRiskScore
	}

	if signal.Severity == "critical" || signal.Severity == "high" {
		session.Status = "escalated"
	}

	s.riskSignals[sessionID] = append(s.riskSignals[sessionID], signal)

	return nil
}

func (s *ztaEnhancedService) GetRiskScore(ctx context.Context, sessionID string) (*RiskScoreResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.validationSessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	breakdown := s.calculateRiskBreakdown(session)

	result := &RiskScoreResult{
		SessionID:  sessionID,
		TotalScore: session.CurrentRiskScore,
		RiskLevel:  s.calculateRiskLevel(session.CurrentRiskScore),
		Breakdown:  breakdown,
		Factors:    session.ActiveFactors,
		Timestamp:  time.Now(),
		Recommendations: s.generateRiskRecommendations(session.CurrentRiskScore, breakdown),
	}

	return result, nil
}

func (s *ztaEnhancedService) computeAccessRiskScore(request *AccessRequest, session *ValidationSession) float64 {
	baseScore := session.CurrentRiskScore

	if request.SourceIP != "" {
		if s.isKnownSuspiciousIP(request.SourceIP) {
			baseScore += 20
		}
	}

	if request.UserAgent != "" && s.isSuspiciousUserAgent(request.UserAgent) {
		baseScore += 15
	}

	if time.Now().Hour() < 6 || time.Now().Hour() > 22 {
		baseScore += 10
	}

	if baseScore > s.maxRiskScore {
		baseScore = s.maxRiskScore
	}

	return baseScore
}

func (s *ztaEnhancedService) calculateSignalImpact(signal *RiskSignal) float64 {
	switch signal.Severity {
	case "critical":
		return 30
	case "high":
		return 20
	case "medium":
		return 10
	case "low":
		return 5
	default:
		return 0
	}
}

func (s *ztaEnhancedService) calculateRiskLevel(score float64) string {
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

func (s *ztaEnhancedService) calculateRiskBreakdown(session *ValidationSession) map[string]float64 {
	return map[string]float64{
		"device":   session.CurrentRiskScore * 0.2,
		"location": session.CurrentRiskScore * 0.25,
		"behavior": session.CurrentRiskScore * 0.35,
		"time":    session.CurrentRiskScore * 0.2,
	}
}

func (s *ztaEnhancedService) generateRiskRecommendations(score float64, breakdown map[string]float64) []string {
	var recommendations []string

	if score >= 80 {
		recommendations = append(recommendations, "Immediate action required - consider session termination")
		recommendations = append(recommendations, "Enable step-up authentication")
		recommendations = append(recommendations, "Notify security team")
	} else if score >= 60 {
		recommendations = append(recommendations, "Implement enhanced monitoring")
		recommendations = append(recommendations, "Require MFA for all sensitive operations")
	}

	if breakdown["behavior"] > 25 {
		recommendations = append(recommendations, "Review behavior patterns for anomalies")
	}

	if breakdown["location"] > 20 {
		recommendations = append(recommendations, "Verify user location")
	}

	return recommendations
}

func (s *ztaEnhancedService) isKnownSuspiciousIP(ip string) bool {
	suspiciousRanges := []string{"192.0.2.0/24", "203.0.113.0/24"}
	for _, rangeIP := range suspiciousRanges {
		if ip == rangeIP {
			return true
		}
	}
	return false
}

func (s *ztaEnhancedService) isSuspiciousUserAgent(ua string) bool {
	suspiciousAgents := []string{"curl", "wget", "python-requests", "scraper"}
	for _, agent := range suspiciousAgents {
		if len(ua) > len(agent) && ua[:len(agent)] == agent {
			return true
		}
	}
	return false
}

func (s *ztaEnhancedService) CreateMicrosegment(ctx context.Context, segment *MicrosegmentDefinition) (*Microsegment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if segment.SegmentID == "" {
		segment.SegmentID = fmt.Sprintf("seg-%d", time.Now().UnixNano())
	}

	microseg := &Microsegment{
		ID:           uint(time.Now().Unix()),
		Name:         segment.Name,
		SourceIP:     segment.SourceWorkload,
		DestIP:       segment.DestWorkload,
		Port:         segment.Port,
		Protocol:     segment.Protocol,
		AllowedUsers: segment.AllowedUsers,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}

	s.microsegments[segment.SegmentID] = microseg

	return microseg, nil
}

func (s *ztaEnhancedService) DeleteMicrosegment(ctx context.Context, segmentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.microsegments[segmentID]; !exists {
		return ErrSegmentNotFound
	}

	delete(s.microsegments, segmentID)

	return nil
}

func (s *ztaEnhancedService) UpdateMicrosegment(ctx context.Context, segmentID string, updates *MicrosegmentUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	segment, exists := s.microsegments[segmentID]
	if !exists {
		return ErrSegmentNotFound
	}

	if updates.Name != nil {
		segment.Name = *updates.Name
	}
	if updates.AllowedUsers != nil {
		segment.AllowedUsers = updates.AllowedUsers
	}
	if updates.IsActive != nil {
		segment.IsActive = *updates.IsActive
	}

	return nil
}

func (s *ztaEnhancedService) ListMicrosegments(ctx context.Context, filters *MicrosegmentFilter) ([]*Microsegment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Microsegment

	for _, segment := range s.microsegments {
		if filters != nil {
			if filters.WorkloadID != "" && segment.SourceIP != filters.WorkloadID && segment.DestIP != filters.WorkloadID {
				continue
			}
			if filters.Protocol != "" && segment.Protocol != filters.Protocol {
				continue
			}
			if filters.IsActive != nil && segment.IsActive != *filters.IsActive {
				continue
			}
		}
		result = append(result, segment)
	}

	return result, nil
}

func (s *ztaEnhancedService) ValidateTraffic(ctx context.Context, sourceID, destID string, protocol string) (*TrafficDecision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, segment := range s.microsegments {
		if segment.SourceIP == sourceID && segment.DestIP == destID && segment.Protocol == protocol {
			if segment.IsActive {
				return &TrafficDecision{
					Allowed:     true,
					Reason:      "Allowed by microsegment policy",
					MatchedRule: segment.Name,
					RiskLevel:   "low",
					Timestamp:   time.Now(),
					Metadata: map[string]string{
						"segment_id": fmt.Sprintf("%d", segment.ID),
					},
				}, nil
			}
		}
	}

	return &TrafficDecision{
		Allowed:   false,
		Reason:    "No matching microsegment policy",
		RiskLevel: "medium",
		Timestamp: time.Now(),
	}, nil
}

func (s *ztaEnhancedService) IsolateWorkload(ctx context.Context, workloadID string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	status := &WorkloadStatus{
		WorkloadID:       workloadID,
		IsIsolated:       true,
		IsolationReason:  reason,
		IsolatedAt:       &now,
		LastActivity:     time.Now(),
	}

	s.workloads[workloadID] = status

	return nil
}

func (s *ztaEnhancedService) RestoreWorkload(ctx context.Context, workloadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, exists := s.workloads[workloadID]
	if !exists {
		return ErrWorkloadNotFound
	}

	status.IsIsolated = false
	status.IsolationReason = ""
	status.IsolatedAt = nil

	return nil
}

func (s *ztaEnhancedService) GetWorkloadStatus(ctx context.Context, workloadID string) (*WorkloadStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, exists := s.workloads[workloadID]
	if !exists {
		return &WorkloadStatus{
			WorkloadID:   workloadID,
			IsIsolated:   false,
			LastActivity: time.Now(),
		}, nil
	}

	return status, nil
}

func (s *ztaEnhancedService) ComputePermissions(ctx context.Context, userID uint, resource string, context *PermissionContext) (*ComputedPermissions, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if context == nil {
		context = &PermissionContext{
			TimeOfDay: time.Now(),
			DayOfWeek: int(time.Now().Weekday()),
			RiskScore: 0,
		}
	}

	permissions := []string{"read"}

	if context.RiskScore < 40 {
		permissions = append(permissions, "write", "execute")
	}

	if context.AuthMethods != nil && len(context.AuthMethods) > 1 {
		permissions = append(permissions, "admin")
	}

	computed := &ComputedPermissions{
		UserID:      userID,
		Resource:    resource,
		Permissions: permissions,
		Conditions:  s.generatePermissionConditions(context),
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		ComputedAt:   time.Now(),
		Confidence:   0.95,
		FactorsUsed:  []string{"time", "risk_score", "auth_methods"},
	}

	s.permissions[userID] = append(s.permissions[userID], computed)

	return computed, nil
}

func (s *ztaEnhancedService) ReviewPermissions(ctx context.Context, userID uint) (*PermissionReview, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var items []*PermissionItem

	perms, exists := s.permissions[userID]
	if exists {
		for _, perm := range perms {
			items = append(items, &PermissionItem{
				Permission: perm.Permissions[0],
				Resource:   perm.Resource,
				LastUsed:   perm.ComputedAt,
				UseCount:   0,
				RiskLevel:  "low",
			})
		}
	}

	return &PermissionReview{
		UserID:          userID,
		ReviewDate:     time.Now(),
		Permissions:    items,
		UnusedCount:    0,
		Overprivileged: false,
		Recommendations: []string{"Current permissions are appropriate"},
	}, nil
}

func (s *ztaEnhancedService) GrantTemporaryPermission(ctx context.Context, grant *PermissionGrant) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if grant.GrantID == "" {
		grant.GrantID = fmt.Sprintf("grant-%d", time.Now().UnixNano())
	}
	grant.GrantedAt = time.Now()
	grant.Revoked = false

	s.permissionGrants[grant.GrantID] = grant

	return nil
}

func (s *ztaEnhancedService) RevokePermission(ctx context.Context, userID uint, permission string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for grantID, grant := range s.permissionGrants {
		if grant.UserID == userID && grant.Permission == permission && !grant.Revoked {
			grant.Revoked = true
			s.permissionGrants[grantID] = grant
			return nil
		}
	}

	return ErrPermissionDenied
}

func (s *ztaEnhancedService) GetPermissionAudit(ctx context.Context, userID uint, start, end time.Time) (*PermissionAudit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var entries []*AuditEntry

	for _, grant := range s.permissionGrants {
		if grant.UserID == userID {
			entries = append(entries, &AuditEntry{
				Timestamp:  grant.GrantedAt,
				Action:     "grant",
				Permission: grant.Permission,
				Resource:   grant.Resource,
				GrantedBy:  grant.GrantedBy,
				Reason:     grant.Reason,
			})
		}
	}

	return &PermissionAudit{
		UserID:       userID,
		StartDate:   start,
		EndDate:     end,
		Entries:     entries,
		TotalGrants: len(entries),
	}, nil
}

func (s *ztaEnhancedService) generatePermissionConditions(context *PermissionContext) []string {
	var conditions []string

	if context.Location != "" {
		conditions = append(conditions, fmt.Sprintf("location=%s", context.Location))
	}

	if context.DeviceType != "" {
		conditions = append(conditions, fmt.Sprintf("device=%s", context.DeviceType))
	}

	if context.RiskScore < 30 {
		conditions = append(conditions, "risk_level=low")
	}

	return conditions
}

func (s *ztaEnhancedService) RegisterEdgeLocation(ctx context.Context, location *EdgeLocation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	location.LastHeartbeat = time.Now()
	s.edgeLocations[location.EdgeID] = location

	return nil
}

func (s *ztaEnhancedService) GetNearestEdge(ctx context.Context, coordinates *Coordinates) (*EdgeLocation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var nearest *EdgeLocation
	minDistance := float64(0)

	for _, edge := range s.edgeLocations {
		if !edge.IsActive {
			continue
		}

		dist := s.calculateDistance(coordinates, edge.Coordinates)
		if nearest == nil || dist < minDistance {
			nearest = edge
			minDistance = dist
		}
	}

	if nearest == nil {
		return nil, ErrEdgeNotFound
	}

	return nearest, nil
}

func (s *ztaEnhancedService) calculateDistance(coord1, coord2 *Coordinates) float64 {
	latDiff := coord1.Latitude - coord2.Latitude
	lonDiff := coord1.Longitude - coord2.Longitude
	return (latDiff * latDiff) + (lonDiff * lonDiff)
}

func (s *ztaEnhancedService) RouteTraffic(ctx context.Context, request *TrafficRouteRequest) (*TrafficRouteResponse, error) {
	edge, err := s.GetNearestEdge(ctx, request.UserLocation)
	if err != nil {
		return nil, err
	}

	return &TrafficRouteResponse{
		RequestID:        request.RequestID,
		SelectedEdge:     edge,
		RoutePath:        []string{"client", edge.EdgeID, request.Destination},
		EstimatedLatency: int(s.calculateDistance(request.UserLocation, edge.Coordinates)),
		SecurityLevel:   "high",
		Decision:         "allow",
	}, nil
}

func (s *ztaEnhancedService) SyncSecurityPolicy(ctx context.Context, policy *SecurityPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	policy.LastUpdated = time.Now()
	s.securityPolicies[policy.PolicyID] = policy

	return nil
}

func (s *ztaEnhancedService) GetEdgeStatus(ctx context.Context, edgeID string) (*EdgeStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	edge, exists := s.edgeLocations[edgeID]
	if !exists {
		return nil, ErrEdgeNotFound
	}

	return &EdgeStatus{
		EdgeID:            edgeID,
		IsHealthy:         edge.IsActive,
		CPUUsage:          float64(edge.CurrentLoad) / float64(edge.Capacity) * 100,
		MemoryUsage:       45.0,
		NetworkIn:         int64(edge.CurrentLoad * 1000),
		NetworkOut:        int64(edge.CurrentLoad * 800),
		ActiveConnections: edge.CurrentLoad / 10,
		LastHeartbeat:     edge.LastHeartbeat,
	}, nil
}

func (s *ztaEnhancedService) ProcessEdgeEvent(ctx context.Context, event *EdgeEvent) error {
	if event.EventID == "" {
		event.EventID = fmt.Sprintf("event-%d", time.Now().UnixNano())
	}
	event.Timestamp = time.Now()

	return nil
}

func (s *ztaEnhancedService) hashString(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
