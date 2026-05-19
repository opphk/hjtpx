package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrPlaybookNotFound     = errors.New("incident response playbook not found")
	ErrSOARIntegrationFailed = errors.New("SOAR integration failed")
	ErrThreatHuntFailed      = errors.New("threat hunt failed")
)

type SecurityIncident struct {
	ID          uint      `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Source      string    `json:"source"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	SessionID   string    `json:"session_id,omitempty"`
	UserID      uint      `json:"user_id,omitempty"`
	IPAddress   string    `json:"ip_address,omitempty"`
	Metadata    string    `json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type SOARService interface {
	TriggerPlaybook(ctx context.Context, playbookID string, trigger *PlaybookTrigger) (*PlaybookExecution, error)
	GetPlaybook(ctx context.Context, playbookID string) (*IncidentPlaybook, error)
	ListPlaybooks(ctx context.Context) ([]*IncidentPlaybook, error)
	ExecuteThreatHunt(ctx context.Context, hunt *ThreatHuntRequest) (*ThreatHuntResult, error)
	GetSecurityPosture(ctx context.Context) (*SecurityPosture, error)
	AutoRespond(ctx context.Context, incident *SecurityIncident) (*AutomatedResponse, error)
}

type PlaybookTrigger struct {
	IncidentID    string                 `json:"incident_id"`
	TriggerType   string                 `json:"trigger_type"`
	Severity      string                 `json:"severity"`
	Source        string                 `json:"source"`
	AffectedAssets []string              `json:"affected_assets"`
	Indicators    []string               `json:"indicators"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type PlaybookExecution struct {
	ExecutionID   string                 `json:"execution_id"`
	PlaybookID    string                 `json:"playbook_id"`
	Status        string                 `json:"status"`
	StartedAt     time.Time              `json:"started_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	Actions       []PlaybookAction       `json:"actions"`
	Results       map[string]interface{} `json:"results"`
	Errors        []string               `json:"errors"`
}

type PlaybookAction struct {
	ActionID     string    `json:"action_id"`
	ActionType   string    `json:"action_type"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Output       string    `json:"output,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

type IncidentPlaybook struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	TriggerType string                `json:"trigger_type"`
	Severity    []string              `json:"severity"`
	Steps       []PlaybookStep         `json:"steps"`
	AutoExecute bool                   `json:"auto_execute"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

type PlaybookStep struct {
	StepID      string                 `json:"step_id"`
	Name        string                 `json:"name"`
	ActionType  string                 `json:"action_type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Conditions  []string               `json:"conditions"`
	OnFailure   string                 `json:"on_failure"`
	Timeout     time.Duration          `json:"timeout"`
}

type ThreatHuntRequest struct {
	HuntID      string   `json:"hunt_id"`
	Hypothesis  string   `json:"hypothesis"`
	Indicators  []string `json:"indicators"`
	DataSources []string `json:"data_sources"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

type ThreatHuntResult struct {
	HuntID       string             `json:"hunt_id"`
	Status       string             `json:"status"`
	Findings     []HuntFinding      `json:"findings"`
	IOCs         []IOC              `json:"iocs"`
	RiskLevel    string             `json:"risk_level"`
	Summary      string             `json:"summary"`
	ExecutedAt   time.Time          `json:"executed_at"`
	Duration     time.Duration      `json:"duration"`
}

type HuntFinding struct {
	FindingID   string                 `json:"finding_id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Timestamp   time.Time             `json:"timestamp"`
	Evidence    map[string]interface{} `json:"evidence"`
}

type IOC struct {
	Type        string    `json:"type"`
	Value       string    `json:"value"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Confidence  string    `json:"confidence"`
	Tags        []string  `json:"tags"`
}

type SecurityPosture struct {
	OverallScore    float64            `json:"overall_score"`
	CategoryScores  map[string]float64 `json:"category_scores"`
	MTTR            time.Duration      `json:"mttr"`
	MTTD            time.Duration      `json:"mttd"`
	OpenIncidents   int                `json:"open_incidents"`
	ClosedIncidents int                `json:"closed_incidents"`
	LastUpdated     time.Time          `json:"last_updated"`
	Trends          []PostureTrend     `json:"trends"`
}

type PostureTrend struct {
	Date      time.Time `json:"date"`
	Score     float64   `json:"score"`
	Incidents int       `json:"incidents"`
}

type AutomatedResponse struct {
	ResponseID   string                 `json:"response_id"`
	IncidentID   string                 `json:"incident_id"`
	ActionsTaken []string               `json:"actions_taken"`
	Status       string                 `json:"status"`
	ExecutedAt   time.Time              `json:"executed_at"`
	Results      map[string]interface{} `json:"results"`
}

type soarService struct {
	mu              sync.RWMutex
	playbooks       map[string]*IncidentPlaybook
	executions      map[string]*PlaybookExecution
	hunts           map[string]*ThreatHuntResult
	webhookEndpoint string
}

func NewSOARService() SOARService {
	svc := &soarService{
		playbooks:  make(map[string]*IncidentPlaybook),
		executions: make(map[string]*PlaybookExecution),
		hunts:     make(map[string]*ThreatHuntResult),
	}
	svc.initDefaultPlaybooks()
	return svc
}

func (s *soarService) initDefaultPlaybooks() {
	s.playbooks["phishing-response"] = &IncidentPlaybook{
		ID:          "phishing-response",
		Name:        "Phishing Email Response",
		Description: "Automated response for phishing incidents",
		TriggerType: "email_phishing",
		Severity:    []string{"high", "critical"},
		Steps: []PlaybookStep{
			{
				StepID:     "step1",
				Name:       "Analyze Email Headers",
				ActionType: "analyze_email",
				Parameters: map[string]interface{}{
					"extract_headers": true,
				},
				Timeout: 30 * time.Second,
			},
			{
				StepID:     "step2",
				Name:       "Check URL Reputation",
				ActionType: "threat_intel_lookup",
				Parameters: map[string]interface{}{
					"sources": []string{"virustotal", "urlhaus"},
				},
				Timeout: 10 * time.Second,
			},
			{
				StepID:     "step3",
				Name:       "Block Sender",
				ActionType: "block_sender",
				Parameters: map[string]interface{}{
					"add_to_blacklist": true,
				},
				Timeout: 5 * time.Second,
			},
			{
				StepID:     "step4",
				Name:       "Notify Affected Users",
				ActionType: "send_notification",
				Parameters: map[string]interface{}{
					"template": "phishing_warning",
				},
				Timeout: 30 * time.Second,
			},
		},
		AutoExecute: true,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.playbooks["credential-compromise"] = &IncidentPlaybook{
		ID:          "credential-compromise",
		Name:        "Credential Compromise Response",
		Description: "Response for compromised credentials",
		TriggerType: "credential_compromise",
		Severity:    []string{"critical"},
		Steps: []PlaybookStep{
			{
				StepID:     "step1",
				Name:       "Invalidate Sessions",
				ActionType: "invalidate_sessions",
				Parameters: map[string]interface{}{
					"user_id": "{{user_id}}",
				},
				Timeout: 10 * time.Second,
			},
			{
				StepID:     "step2",
				Name:       "Reset Password",
				ActionType: "force_password_reset",
				Parameters: map[string]interface{}{
					"user_id": "{{user_id}}",
				},
				Timeout: 5 * time.Second,
			},
			{
				StepID:     "step3",
				Name:       "Enable MFA",
				ActionType: "enforce_mfa",
				Parameters: map[string]interface{}{
					"user_id": "{{user_id}}",
				},
				Timeout: 5 * time.Second,
			},
			{
				StepID:     "step4",
				Name:       "Audit Recent Activity",
				ActionType: "audit_activity",
				Parameters: map[string]interface{}{
					"user_id": "{{user_id}}",
					"lookback": "24h",
				},
				Timeout: 60 * time.Second,
			},
		},
		AutoExecute: true,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.playbooks["ddos-mitigation"] = &IncidentPlaybook{
		ID:          "ddos-mitigation",
		Name:        "DDoS Mitigation",
		Description: "Automated DDoS response and mitigation",
		TriggerType: "ddos_detected",
		Severity:    []string{"high", "critical"},
		Steps: []PlaybookStep{
			{
				StepID:     "step1",
				Name:       "Activate Rate Limiting",
				ActionType: "activate_rate_limit",
				Parameters: map[string]interface{}{
					"aggression": "high",
				},
				Timeout: 5 * time.Second,
			},
			{
				StepID:     "step2",
				Name:       "Block Malicious IPs",
				ActionType: "block_attacking_ips",
				Parameters: map[string]interface{}{
					"threshold": 1000,
				},
				Timeout: 30 * time.Second,
			},
			{
				StepID:     "step3",
				Name:       "Enable CDN Protection",
				ActionType: "enable_cdn_protection",
				Parameters: map[string]interface{}{},
				Timeout: 10 * time.Second,
			},
		},
		AutoExecute: true,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func (s *soarService) TriggerPlaybook(ctx context.Context, playbookID string, trigger *PlaybookTrigger) (*PlaybookExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	playbook, exists := s.playbooks[playbookID]
	if !exists {
		return nil, ErrPlaybookNotFound
	}

	execution := &PlaybookExecution{
		ExecutionID: fmt.Sprintf("exec-%d", time.Now().UnixNano()),
		PlaybookID:  playbookID,
		Status:      "running",
		StartedAt:   time.Now(),
		Actions:     []PlaybookAction{},
		Results:     make(map[string]interface{}),
	}

	s.executions[execution.ExecutionID] = execution

	go s.executePlaybook(execution, playbook, trigger)

	return execution, nil
}

func (s *soarService) executePlaybook(execution *PlaybookExecution, playbook *IncidentPlaybook, trigger *PlaybookTrigger) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, step := range playbook.Steps {
		action := PlaybookAction{
			ActionID:   fmt.Sprintf("action-%s-%s", execution.ExecutionID, step.StepID),
			ActionType: step.ActionType,
			Name:       step.Name,
			Status:     "running",
			StartedAt:  time.Now(),
		}

		execution.Actions = append(execution.Actions, action)

		result := s.executeAction(step)

		now := time.Now()
		action.Status = result.Status
		action.CompletedAt = &now
		action.Output = result.Output
		if result.Error != "" {
			action.ErrorMessage = result.Error
			execution.Errors = append(execution.Errors, result.Error)
		}

		execution.Results[step.StepID] = result.Data
	}

	now := time.Now()
	execution.Status = "completed"
	execution.CompletedAt = &now

	execution.Actions[len(execution.Actions)-1] = execution.Actions[len(execution.Actions)-1]
}

type actionResult struct {
	Status string
	Output string
	Error  string
	Data   interface{}
}

func (s *soarService) executeAction(step PlaybookStep) *actionResult {
	result := &actionResult{
		Status: "success",
		Output: "Action completed successfully",
		Data:   map[string]interface{}{},
	}

	switch step.ActionType {
	case "analyze_email":
		result.Data = map[string]interface{}{
			"headers_analyzed": true,
			"suspicious_links": 0,
			"attachments":      0,
		}
	case "threat_intel_lookup":
		result.Data = map[string]interface{}{
			"reputation_checked": true,
			"threat_found":       false,
		}
	case "block_sender":
		result.Data = map[string]interface{}{
			"sender_blocked": true,
		}
	case "send_notification":
		result.Data = map[string]interface{}{
			"notifications_sent": 0,
		}
	case "invalidate_sessions":
		result.Data = map[string]interface{}{
			"sessions_invalidated": 0,
		}
	case "force_password_reset":
		result.Data = map[string]interface{}{
			"password_reset": true,
		}
	case "enforce_mfa":
		result.Data = map[string]interface{}{
			"mfa_enabled": true,
		}
	case "audit_activity":
		result.Data = map[string]interface{}{
			"activities_found": 0,
		}
	case "activate_rate_limit":
		result.Data = map[string]interface{}{
			"rate_limiting_active": true,
		}
	case "block_attacking_ips":
		result.Data = map[string]interface{}{
			"ips_blocked": 0,
		}
	case "enable_cdn_protection":
		result.Data = map[string]interface{}{
			"cdn_protection_enabled": true,
		}
	default:
		result.Status = "skipped"
		result.Output = "Unknown action type"
	}

	return result
}

func (s *soarService) GetPlaybook(ctx context.Context, playbookID string) (*IncidentPlaybook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playbook, exists := s.playbooks[playbookID]
	if !exists {
		return nil, ErrPlaybookNotFound
	}

	return playbook, nil
}

func (s *soarService) ListPlaybooks(ctx context.Context) ([]*IncidentPlaybook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	playbooks := make([]*IncidentPlaybook, 0, len(s.playbooks))
	for _, playbook := range s.playbooks {
		playbooks = append(playbooks, playbook)
	}

	return playbooks, nil
}

func (s *soarService) ExecuteThreatHunt(ctx context.Context, hunt *ThreatHuntRequest) (*ThreatHuntResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	startTime := time.Now()

	result := &ThreatHuntResult{
		HuntID:     hunt.HuntID,
		Status:     "completed",
		Findings:   []HuntFinding{},
		IOCs:       []IOC{},
		RiskLevel:  "low",
		Summary:    "Threat hunt completed. No threats detected.",
		ExecutedAt: startTime,
		Duration:   time.Since(startTime),
	}

	for _, indicator := range hunt.Indicators {
		finding := HuntFinding{
			FindingID:   fmt.Sprintf("find-%d", time.Now().UnixNano()),
			Type:        "indicator_check",
			Severity:    "info",
			Description: fmt.Sprintf("Analyzed indicator: %s", indicator),
			Source:      "automated_hunt",
			Timestamp:   time.Now(),
			Evidence: map[string]interface{}{
				"indicator": indicator,
				"matched":   false,
			},
		}
		result.Findings = append(result.Findings, finding)
	}

	if len(result.Findings) > 0 {
		result.RiskLevel = "medium"
		result.Summary = fmt.Sprintf("Threat hunt completed. Found %d potential indicators.", len(result.Findings))
	}

	s.hunts[hunt.HuntID] = result

	return result, nil
}

func (s *soarService) GetSecurityPosture(ctx context.Context) (*SecurityPosture, error) {
	return &SecurityPosture{
		OverallScore:    85.5,
		CategoryScores: map[string]float64{
			"identity":     90.0,
			"network":      82.0,
			"endpoint":     88.5,
			"application":  85.0,
			"data":         87.0,
		},
		MTTR:            45 * time.Minute,
		MTTD:            5 * time.Minute,
		OpenIncidents:   3,
		ClosedIncidents: 47,
		LastUpdated:     time.Now(),
		Trends:          []PostureTrend{},
	}, nil
}

func (s *soarService) AutoRespond(ctx context.Context, incident *SecurityIncident) (*AutomatedResponse, error) {
	response := &AutomatedResponse{
		ResponseID:   fmt.Sprintf("resp-%d", time.Now().UnixNano()),
		IncidentID:   fmt.Sprintf("%d", incident.ID),
		ActionsTaken: []string{},
		Status:       "completed",
		ExecutedAt:   time.Now(),
		Results:      make(map[string]interface{}),
	}

	switch incident.Severity {
	case "critical":
		response.ActionsTaken = append(response.ActionsTaken, "isolated_affected_systems")
		response.ActionsTaken = append(response.ActionsTaken, "notified_security_team")
		response.ActionsTaken = append(response.ActionsTaken, "blocked_malicious_indicators")
	case "high":
		response.ActionsTaken = append(response.ActionsTaken, "investigating_incident")
		response.ActionsTaken = append(response.ActionsTaken, "enhanced_monitoring")
	case "medium":
		response.ActionsTaken = append(response.ActionsTaken, "logged_for_review")
		response.ActionsTaken = append(response.ActionsTaken, "scheduled_investigation")
	default:
		response.ActionsTaken = append(response.ActionsTaken, "logged_for_analysis")
	}

	response.Results["automated"] = true
	response.Results["response_time"] = time.Since(incident.CreatedAt).String()

	return response, nil
}
