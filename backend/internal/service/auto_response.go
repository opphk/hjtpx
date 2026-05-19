package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type AutoResponseService struct {
	responseRules     map[string]*ResponseRule
	activeActions     map[string]*ActiveAction
	actionHistory     []*ActionRecord
	escalationMatrix *EscalationMatrix
	notificationChannels map[string]*NotificationChannel
	automationWorkflows map[string]*AutomationWorkflow
	containmentPolicies map[string]*ContainmentPolicy
	isEnabled        bool
	mu               sync.RWMutex
	processedCount   int32
	failedCount      int32
	lastAction       time.Time
}

type ResponseRule struct {
	ID            string
	Name          string
	TriggerCondition *TriggerCondition
	Action        ResponseAction
	Priority      int
	IsActive      bool
	Cooldown      time.Duration
	LastTriggered time.Time
	TriggerCount  int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type TriggerCondition struct {
	ThreatType    []string
	ThreatLevel   []int
	SourceIP      string
	CountryCode   string
	ASNumber      int
	RequestCount  int
	ErrorCount    int
	TimeWindow    time.Duration
	RegexPattern  *regexp.Regexp
	MinConfidence float64
}

type ResponseAction struct {
	Type          ActionType
	Target        ActionTarget
	Parameters    map[string]interface{}
	Duration      time.Duration
	Notification  bool
	Escalation    bool
}

type ActionType string

const (
	ActionTypeBlockIP           ActionType = "block_ip"
	ActionTypeRateLimit        ActionType = "rate_limit"
	ActionTypeChallenge         ActionType = "challenge"
	ActionTypeCaptcha           ActionType = "captcha"
	ActionTypeNotify            ActionType = "notify"
	ActionTypeEscalate          ActionType = "escalate"
	ActionTypeRedirect          ActionType = "redirect"
	ActionTypeCustom            ActionType = "custom"
	ActionTypeQuarantine       ActionType = "quarantine"
	ActionTypeRevokeSession     ActionType = "revoke_session"
	ActionTypeLockAccount       ActionType = "lock_account"
	ActionTypeWebhook           ActionType = "webhook"
)

type ActionTarget struct {
	Type      string
	IP        string
	UserID    string
	SessionID string
	AccountID string
}

type ActiveAction struct {
	ID           string
	RuleID       string
	ActionType   ActionType
	Target       *ActionTarget
	StartTime    time.Time
	EndTime      time.Time
	IsActive     bool
	Reason       string
	TriggerCount int
}

type ActionRecord struct {
	ID           string
	RuleID       string
	RuleName     string
	ActionType   ActionType
	Target       *ActionTarget
	Timestamp    time.Time
	Duration     time.Duration
	Success      bool
	ErrorMessage string
	ThreatLevel  int
}

type EscalationMatrix struct {
	Levels        []EscalationLevel
	AutoEscalate  bool
	MaxEscalation int
}

type EscalationLevel struct {
	Level       int
	Name        string
	Criteria    *EscalationCriteria
	Actions     []ResponseAction
	NotifyTeam  bool
	Timeout     time.Duration
}

type EscalationCriteria struct {
	Duration     time.Duration
	RequestCount int
	ThreatScore  float64
	AttackTypes  []string
}

type NotificationChannel struct {
	ID       string
	Type     NotificationType
	Endpoint string
	IsActive bool
	Config   map[string]string
}

type NotificationType string

const (
	NotificationTypeEmail    NotificationType = "email"
	NotificationTypeSlack   NotificationType = "slack"
	NotificationTypePagerDuty NotificationType = "pagerduty"
	NotificationTypeWebhook NotificationType = "webhook"
	NotificationTypeSMS     NotificationType = "sms"
)

type AutomationWorkflow struct {
	ID          string
	Name        string
	Trigger     *WorkflowTrigger
	Steps       []WorkflowStep
	IsActive    bool
	LastRun     time.Time
	RunCount    int
	CreatedAt   time.Time
}

type WorkflowTrigger struct {
	Type        string
	Condition   *TriggerCondition
	ThreatLevel int
}

type WorkflowStep struct {
	Order       int
	Action      ResponseAction
	Condition   *StepCondition
	Timeout     time.Duration
	OnFailure   string
}

type StepCondition struct {
	Type      string
	Parameter string
	Operator  string
	Value     interface{}
}

type ContainmentPolicy struct {
	ID          string
	Name        string
	AttackType  string
	Actions     []ResponseAction
	IsActive    bool
	Priority    int
	CreatedAt   time.Time
}

type ResponseResult struct {
	ActionID   string
	Success    bool
	Message    string
	Details    map[string]interface{}
	ExecutedAt time.Time
}

type ThreatContext struct {
	IP           string
	UserID       string
	SessionID    string
	ThreatLevel  int
	ThreatTypes  []string
	Confidence   float64
	RequestCount int
	UserAgent    string
	CountryCode  string
	ASNNumber    int
	FirstSeen    time.Time
	LastSeen     time.Time
	AttackPatterns []string
}

type EscalationContext struct {
	CurrentLevel    int
	ThreatLevel     int
	Duration        time.Duration
	ActionCount     int
	ThreatTypes     []string
	AffectedTargets []string
}

func NewAutoResponseService() *AutoResponseService {
	service := &AutoResponseService{
		responseRules:      make(map[string]*ResponseRule),
		activeActions:      make(map[string]*ActiveAction),
		actionHistory:      make([]*ActionRecord, 0),
		escalationMatrix:   &EscalationMatrix{},
		notificationChannels: make(map[string]*NotificationChannel),
		automationWorkflows: make(map[string]*AutomationWorkflow),
		containmentPolicies: make(map[string]*ContainmentPolicy),
		isEnabled:          true,
	}

	service.initializeDefaultRules()
	service.initializeEscalationMatrix()
	service.initializeNotificationChannels()
	service.initializeContainmentPolicies()
	return service
}

func (s *AutoResponseService) initializeDefaultRules() {
	s.responseRules["critical_block"] = &ResponseRule{
		ID:       "critical_block",
		Name:     "Critical Threat Auto-Block",
		Priority: 100,
		TriggerCondition: &TriggerCondition{
			ThreatLevel:   []int{5},
			MinConfidence: 0.8,
		},
		Action: ResponseAction{
			Type:       ActionTypeBlockIP,
			Duration:   24 * time.Hour,
			Notification: true,
			Escalation: true,
		},
		IsActive:  true,
		Cooldown:  1 * time.Hour,
		CreatedAt: time.Now(),
	}

	s.responseRules["high_challenge"] = &ResponseRule{
		ID:       "high_challenge",
		Name:     "High Threat Challenge",
		Priority: 80,
		TriggerCondition: &TriggerCondition{
			ThreatLevel:   []int{4},
			MinConfidence: 0.6,
		},
		Action: ResponseAction{
			Type:        ActionTypeCaptcha,
			Duration:    1 * time.Hour,
			Notification: false,
		},
		IsActive:  true,
		Cooldown:  30 * time.Minute,
		CreatedAt: time.Now(),
	}

	s.responseRules["brute_force_block"] = &ResponseRule{
		ID:       "brute_force_block",
		Name:     "Brute Force Attack Block",
		Priority: 90,
		TriggerCondition: &TriggerCondition{
			ThreatType:   []string{"brute_force", "credential_stuffing"},
			RequestCount: 50,
			TimeWindow:   5 * time.Minute,
		},
		Action: ResponseAction{
			Type:       ActionTypeBlockIP,
			Duration:   2 * time.Hour,
			Notification: true,
		},
		IsActive:  true,
		Cooldown:  1 * time.Hour,
		CreatedAt: time.Now(),
	}

	s.responseRules["rate_limit_rule"] = &ResponseRule{
		ID:       "rate_limit_rule",
		Name:     "Rate Limit Excessive Requests",
		Priority: 60,
		TriggerCondition: &TriggerCondition{
			RequestCount: 200,
			TimeWindow:   1 * time.Minute,
		},
		Action: ResponseAction{
			Type:       ActionTypeRateLimit,
			Parameters: map[string]interface{}{"limit": 10, "window": "1m"},
			Duration:   15 * time.Minute,
		},
		IsActive:  true,
		Cooldown:  5 * time.Minute,
		CreatedAt: time.Now(),
	}

	s.responseRules["sql_injection_block"] = &ResponseRule{
		ID:       "sql_injection_block",
		Name:     "SQL Injection Block",
		Priority: 95,
		TriggerCondition: &TriggerCondition{
			ThreatType:   []string{"sql_injection"},
			MinConfidence: 0.7,
		},
		Action: ResponseAction{
			Type:       ActionTypeBlockIP,
			Duration:   6 * time.Hour,
			Notification: true,
		},
		IsActive:  true,
		Cooldown:  30 * time.Minute,
		CreatedAt: time.Now(),
	}

	s.responseRules["xss_challenge"] = &ResponseRule{
		ID:       "xss_challenge",
		Name:     "XSS Attack Challenge",
		Priority: 85,
		TriggerCondition: &TriggerCondition{
			ThreatType:   []string{"xss"},
			MinConfidence: 0.6,
		},
		Action: ResponseAction{
			Type:       ActionTypeChallenge,
			Duration:   30 * time.Minute,
		},
		IsActive:  true,
		Cooldown:  15 * time.Minute,
		CreatedAt: time.Now(),
	}

	s.responseRules["account_lock"] = &ResponseRule{
		ID:       "account_lock",
		Name:     "Account Lock on Suspicious Activity",
		Priority: 88,
		TriggerCondition: &TriggerCondition{
			ThreatType:   []string{"account_takeover", "credential_stuffing"},
			ErrorCount:   10,
			TimeWindow:   10 * time.Minute,
		},
		Action: ResponseAction{
			Type:       ActionTypeLockAccount,
			Duration:   1 * time.Hour,
			Notification: true,
		},
		IsActive:  true,
		Cooldown:  30 * time.Minute,
		CreatedAt: time.Now(),
	}
}

func (s *AutoResponseService) initializeEscalationMatrix() {
	s.escalationMatrix = &EscalationMatrix{
		AutoEscalate:  true,
		MaxEscalation: 5,
		Levels: []EscalationLevel{
			{
				Level:      1,
				Name:       "Initial Response",
				Criteria:   &EscalationCriteria{ThreatScore: 0.3},
				NotifyTeam: false,
				Timeout:    5 * time.Minute,
				Actions: []ResponseAction{
					{Type: ActionTypeNotify, Notification: true},
				},
			},
			{
				Level:      2,
				Name:       "Escalated",
				Criteria:   &EscalationCriteria{ThreatScore: 0.5, Duration: 5 * time.Minute},
				NotifyTeam: true,
				Timeout:    15 * time.Minute,
				Actions: []ResponseAction{
					{Type: ActionTypeBlockIP, Duration: 1 * time.Hour},
					{Type: ActionTypeNotify, Notification: true},
				},
			},
			{
				Level:      3,
				Name:       "Critical",
				Criteria:   &EscalationCriteria{ThreatScore: 0.7, Duration: 15 * time.Minute},
				NotifyTeam: true,
				Timeout:    1 * time.Hour,
				Actions: []ResponseAction{
					{Type: ActionTypeBlockIP, Duration: 24 * time.Hour},
					{Type: ActionTypeNotify, Notification: true},
					{Type: ActionTypeWebhook, Parameters: map[string]interface{}{"alert": "critical"}},
				},
			},
			{
				Level:      4,
				Name:       "Emergency",
				Criteria:   &EscalationCriteria{ThreatScore: 0.9},
				NotifyTeam: true,
				Timeout:    0,
				Actions: []ResponseAction{
					{Type: ActionTypeBlockIP, Duration: 72 * time.Hour},
					{Type: ActionTypeNotify, Notification: true},
					{Type: ActionTypeWebhook, Parameters: map[string]interface{}{"alert": "emergency"}},
				},
			},
		},
	}
}

func (s *AutoResponseService) initializeNotificationChannels() {
	s.notificationChannels["email"] = &NotificationChannel{
		ID:       "email",
		Type:     NotificationTypeEmail,
		Endpoint: "security@example.com",
		IsActive: true,
	}
	s.notificationChannels["slack"] = &NotificationChannel{
		ID:       "slack",
		Type:     NotificationTypeSlack,
		Endpoint: "https://hooks.slack.com/services/xxx",
		IsActive: true,
	}
	s.notificationChannels["webhook"] = &NotificationChannel{
		ID:       "webhook",
		Type:     NotificationTypeWebhook,
		Endpoint: "https://internal.example.com/security/webhook",
		IsActive: true,
	}
}

func (s *AutoResponseService) initializeContainmentPolicies() {
	s.containmentPolicies["ddos_containment"] = &ContainmentPolicy{
		ID:         "ddos_containment",
		Name:       "DDoS Attack Containment",
		AttackType: "ddos",
		Priority:   100,
		IsActive:   true,
		Actions: []ResponseAction{
			{Type: ActionTypeRateLimit, Parameters: map[string]interface{}{"limit": 1}},
			{Type: ActionTypeNotify, Notification: true},
		},
		CreatedAt: time.Now(),
	}
	s.containmentPolicies["data_exfil_containment"] = &ContainmentPolicy{
		ID:         "data_exfil_containment",
		Name:       "Data Exfiltration Containment",
		AttackType: "data_theft",
		Priority:   100,
		IsActive:   true,
		Actions: []ResponseAction{
			{Type: ActionTypeBlockIP, Duration: 24 * time.Hour},
			{Type: ActionTypeRevokeSession},
			{Type: ActionTypeNotify, Notification: true},
		},
		CreatedAt: time.Now(),
	}
}

func (s *AutoResponseService) ProcessThreat(ctx context.Context, threat *ThreatContext) (*ResponseResult, error) {
	if !s.isEnabled {
		return &ResponseResult{Success: false, Message: "Service disabled"}, nil
	}

	atomic.AddInt32(&s.processedCount, 1)

	matchedRules := s.findMatchingRules(threat)
	if len(matchedRules) == 0 {
		return &ResponseResult{Success: true, Message: "No matching rules"}, nil
	}

	var lastResult *ResponseResult
	for _, rule := range matchedRules {
		if s.isInCooldown(rule) {
			continue
		}

		result, err := s.executeAction(ctx, rule, threat)
		if err != nil {
			atomic.AddInt32(&s.failedCount, 1)
			s.recordAction(rule, threat, result, err.Error())
			continue
		}

		s.recordAction(rule, threat, result, "")
		lastResult = result

		rule.LastTriggered = time.Now()
		rule.TriggerCount++

		if rule.Action.Escalation {
			go s.handleEscalation(ctx, threat)
		}

		if rule.Action.Notification {
			go s.sendNotifications(rule, threat)
		}

		s.mu.Lock()
		s.lastAction = time.Now()
		s.mu.Unlock()

		if result.Success {
			break
		}
	}

	return lastResult, nil
}

func (s *AutoResponseService) findMatchingRules(threat *ThreatContext) []*ResponseRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matched []*ResponseRule

	for _, rule := range s.responseRules {
		if !rule.IsActive {
			continue
		}

		if s.matchesCondition(rule.TriggerCondition, threat) {
			matched = append(matched, rule)
		}
	}

	for i := 0; i < len(matched)-1; i++ {
		for j := i + 1; j < len(matched); j++ {
			if matched[j].Priority > matched[i].Priority {
				matched[i], matched[j] = matched[j], matched[i]
			}
		}
	}

	return matched
}

func (s *AutoResponseService) matchesCondition(condition *TriggerCondition, threat *ThreatContext) bool {
	if len(condition.ThreatLevel) > 0 {
		found := false
		for _, level := range condition.ThreatLevel {
			if threat.ThreatLevel == level {
				found = true
				break
			}
		}
		if !found && condition.ThreatLevel[0] < threat.ThreatLevel {
		} else if !found {
			return false
		}
	}

	if len(condition.ThreatType) > 0 {
		found := false
		for _, t := range condition.ThreatType {
			for _, threatType := range threat.ThreatTypes {
				if strings.Contains(strings.ToLower(threatType), strings.ToLower(t)) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	if condition.RequestCount > 0 && threat.RequestCount < condition.RequestCount {
		return false
	}

	if condition.MinConfidence > 0 && threat.Confidence < condition.MinConfidence {
		return false
	}

	if condition.CountryCode != "" && threat.CountryCode != condition.CountryCode {
		return false
	}

	return true
}

func (s *AutoResponseService) isInCooldown(rule *ResponseRule) bool {
	if rule.Cooldown == 0 {
		return false
	}
	return time.Since(rule.LastTriggered) < rule.Cooldown
}

func (s *AutoResponseService) executeAction(ctx context.Context, rule *ResponseRule, threat *ThreatContext) (*ResponseResult, error) {
	target := &ActionTarget{
		IP: threat.IP,
	}

	actionID := fmt.Sprintf("action_%d", time.Now().UnixNano())

	activeAction := &ActiveAction{
		ID:         actionID,
		RuleID:     rule.ID,
		ActionType: rule.Action.Type,
		Target:     target,
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(rule.Action.Duration),
		IsActive:   true,
		Reason:     rule.Name,
	}

	s.mu.Lock()
	s.activeActions[actionID] = activeAction
	s.mu.Unlock()

	var success bool
	var message string

	switch rule.Action.Type {
	case ActionTypeBlockIP:
		success = s.blockIP(threat.IP, rule.Action.Duration)
		message = fmt.Sprintf("Blocked IP %s for %v", threat.IP, rule.Action.Duration)
	case ActionTypeRateLimit:
		success = s.applyRateLimit(threat.IP, rule.Action)
		message = fmt.Sprintf("Rate limit applied to IP %s", threat.IP)
	case ActionTypeChallenge:
		success = true
		message = "Challenge initiated"
	case ActionTypeCaptcha:
		success = true
		message = "CAPTCHA required"
	case ActionTypeLockAccount:
		success = s.lockAccount(threat.UserID)
		message = fmt.Sprintf("Account locked: %s", threat.UserID)
	case ActionTypeRevokeSession:
		success = s.revokeSession(threat.SessionID)
		message = fmt.Sprintf("Session revoked: %s", threat.SessionID)
	case ActionTypeNotify:
		success = true
		message = "Notification sent"
	case ActionTypeWebhook:
		success = s.triggerWebhook(rule.Action)
		message = "Webhook triggered"
	default:
		success = false
		message = "Unknown action type"
	}

	return &ResponseResult{
		ActionID:   actionID,
		Success:    success,
		Message:    message,
		ExecutedAt: time.Now(),
	}, nil
}

func (s *AutoResponseService) blockIP(ip string, duration time.Duration) bool {
	time.Sleep(10 * time.Millisecond)
	return true
}

func (s *AutoResponseService) applyRateLimit(ip string, action ResponseAction) bool {
	limit := 10
	window := "1m"

	if action.Parameters != nil {
		if l, ok := action.Parameters["limit"].(int); ok {
			limit = l
		}
		if w, ok := action.Parameters["window"].(string); ok {
			window = w
		}
	}

	_ = limit
	_ = window
	return true
}

func (s *AutoResponseService) lockAccount(userID string) bool {
	return userID != ""
}

func (s *AutoResponseService) revokeSession(sessionID string) bool {
	return sessionID != ""
}

func (s *AutoResponseService) triggerWebhook(action ResponseAction) bool {
	return true
}

func (s *AutoResponseService) recordAction(rule *ResponseRule, threat *ThreatContext, result *ResponseResult, errorMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := &ActionRecord{
		ID:          fmt.Sprintf("record_%d", time.Now().UnixNano()),
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		ActionType:  rule.Action.Type,
		Target: &ActionTarget{
			IP: threat.IP,
		},
		Timestamp:    time.Now(),
		Success:      result.Success,
		ErrorMessage: errorMsg,
		ThreatLevel:  threat.ThreatLevel,
	}

	s.actionHistory = append(s.actionHistory, record)

	if len(s.actionHistory) > 10000 {
		s.actionHistory = s.actionHistory[len(s.actionHistory)-10000:]
	}
}

func (s *AutoResponseService) handleEscalation(ctx context.Context, threat *ThreatContext) {
	escalationCtx := &EscalationContext{
		CurrentLevel:    1,
		ThreatLevel:     threat.ThreatLevel,
		Duration:        time.Since(threat.FirstSeen),
		ThreatTypes:     threat.ThreatTypes,
		AffectedTargets: []string{threat.IP},
	}

	if !s.escalationMatrix.AutoEscalate {
		return
	}

	for _, level := range s.escalationMatrix.Levels {
		if s.shouldEscalate(level, escalationCtx) {
			go s.executeEscalationActions(ctx, level, threat)
			escalationCtx.CurrentLevel = level.Level
		}
	}
}

func (s *AutoResponseService) shouldEscalate(level EscalationLevel, ctx *EscalationContext) bool {
	if ctx.CurrentLevel >= level.Level {
		return false
	}

	if level.Criteria == nil {
		return true
	}

	if level.Criteria.ThreatScore > 0 {
		if float64(ctx.ThreatLevel)/10.0 < level.Criteria.ThreatScore {
			return false
		}
	}

	if level.Criteria.Duration > 0 {
		if ctx.Duration < level.Criteria.Duration {
			return false
		}
	}

	if level.Criteria.RequestCount > 0 && ctx.ThreatLevel < level.Criteria.RequestCount/10 {
		return false
	}

	return true
}

func (s *AutoResponseService) executeEscalationActions(ctx context.Context, level EscalationLevel, threat *ThreatContext) {
	for _, action := range level.Actions {
		go s.executeAction(ctx, &ResponseRule{Action: action}, threat)
	}

	if level.NotifyTeam {
		go s.notifySecurityTeam(level.Name, threat)
	}
}

func (s *AutoResponseService) notifySecurityTeam(levelName string, threat *ThreatContext) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, channel := range s.notificationChannels {
		if !channel.IsActive {
			continue
		}
		s.sendToChannel(channel, levelName, threat)
	}
}

func (s *AutoResponseService) sendToChannel(channel *NotificationChannel, levelName string, threat *ThreatContext) {
	_ = levelName
	_ = threat
}

func (s *AutoResponseService) sendNotifications(rule *ResponseRule, threat *ThreatContext) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, channel := range s.notificationChannels {
		if !channel.IsActive {
			continue
		}
		s.sendNotification(channel, rule, threat)
	}
}

func (s *AutoResponseService) sendNotification(channel *NotificationChannel, rule *ResponseRule, threat *ThreatContext) {
	notification := map[string]interface{}{
		"rule_name":    rule.Name,
		"threat_level": threat.ThreatLevel,
		"ip":           threat.IP,
		"threat_types": threat.ThreatTypes,
		"timestamp":    time.Now(),
	}

	_ = notification
}

func (s *AutoResponseService) GetActiveActions() []*ActiveAction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var actions []*ActiveAction
	now := time.Now()
	for _, action := range s.activeActions {
		if action.IsActive && now.Before(action.EndTime) {
			actions = append(actions, action)
		}
	}
	return actions
}

func (s *AutoResponseService) GetActionHistory(filter *ActionHistoryFilter) ([]*ActionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var records []*ActionRecord
	for _, record := range s.actionHistory {
		if filter != nil && !s.matchesHistoryFilter(record, filter) {
			continue
		}
		records = append(records, record)
	}
	return records, nil
}

func (s *AutoResponseService) matchesHistoryFilter(record *ActionRecord, filter *ActionHistoryFilter) bool {
	if filter.ActionType != "" && record.ActionType != ActionType(filter.ActionType) {
		return false
	}
	if filter.RuleID != "" && record.RuleID != filter.RuleID {
		return false
	}
	if !filter.StartTime.IsZero() && record.Timestamp.Before(filter.StartTime) {
		return false
	}
	if !filter.EndTime.IsZero() && record.Timestamp.After(filter.EndTime) {
		return false
	}
	return true
}

type ActionHistoryFilter struct {
	ActionType string
	RuleID     string
	StartTime  time.Time
	EndTime    time.Time
	TargetIP   string
	Success    *bool
}

func (s *AutoResponseService) CreateResponseRule(rule *ResponseRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	rule.IsActive = true
	s.responseRules[rule.ID] = rule
	return nil
}

func (s *AutoResponseService) UpdateResponseRule(ruleID string, updates *ResponseRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule, exists := s.responseRules[ruleID]
	if !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	if updates.Name != "" {
		rule.Name = updates.Name
	}
	if updates.Priority > 0 {
		rule.Priority = updates.Priority
	}
	rule.IsActive = updates.IsActive
	if updates.Cooldown > 0 {
		rule.Cooldown = updates.Cooldown
	}
	rule.UpdatedAt = time.Now()

	return nil
}

func (s *AutoResponseService) DeleteResponseRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.responseRules, ruleID)
	return nil
}

func (s *AutoResponseService) GetResponseRules() []*ResponseRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]*ResponseRule, 0, len(s.responseRules))
	for _, rule := range s.responseRules {
		rules = append(rules, rule)
	}
	return rules
}

func (s *AutoResponseService) AddNotificationChannel(channel *NotificationChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	channel.ID = fmt.Sprintf("channel_%d", time.Now().UnixNano())
	channel.IsActive = true
	s.notificationChannels[channel.ID] = channel
	return nil
}

func (s *AutoResponseService) GetNotificationChannels() []*NotificationChannel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	channels := make([]*NotificationChannel, 0, len(s.notificationChannels))
	for _, channel := range s.notificationChannels {
		channels = append(channels, channel)
	}
	return channels
}

func (s *AutoResponseService) CreateAutomationWorkflow(workflow *AutomationWorkflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workflow.ID = fmt.Sprintf("workflow_%d", time.Now().UnixNano())
	workflow.CreatedAt = time.Now()
	workflow.IsActive = true
	s.automationWorkflows[workflow.ID] = workflow
	return nil
}

func (s *AutoResponseService) ExecuteWorkflow(ctx context.Context, workflowID string, threat *ThreatContext) error {
	s.mu.RLock()
	workflow, exists := s.automationWorkflows[workflowID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	if !workflow.IsActive {
		return fmt.Errorf("workflow is not active: %s", workflowID)
	}

	for _, step := range workflow.Steps {
		if !s.evaluateStepCondition(step.Condition, threat) {
			continue
		}

		ctx, cancel := context.WithTimeout(ctx, step.Timeout)
		defer cancel()

		result, err := s.executeAction(ctx, &ResponseRule{Action: step.Action}, threat)
		if err != nil {
			if step.OnFailure == "abort" {
				return err
			}
		}

		_ = result
	}

	s.mu.Lock()
	workflow.LastRun = time.Now()
	workflow.RunCount++
	s.mu.Unlock()

	return nil
}

func (s *AutoResponseService) evaluateStepCondition(condition *StepCondition, threat *ThreatContext) bool {
	if condition == nil {
		return true
	}

	switch condition.Type {
	case "threat_level":
		return s.compareValues(threat.ThreatLevel, condition.Operator, condition.Value)
	case "threat_type":
		for _, t := range threat.ThreatTypes {
			if strings.Contains(strings.ToLower(t), strings.ToLower(condition.Value.(string))) {
				return true
			}
		}
		return false
	case "confidence":
		return s.compareValues(threat.Confidence, condition.Operator, condition.Value)
	}

	return true
}

func (s *AutoResponseService) compareValues(actual interface{}, operator string, expected interface{}) bool {
	switch operator {
	case ">":
		return toFloat(actual) > toFloat(expected)
	case ">=":
		return toFloat(actual) >= toFloat(expected)
	case "<":
		return toFloat(actual) < toFloat(expected)
	case "<=":
		return toFloat(actual) <= toFloat(expected)
	case "==":
		return actual == expected
	case "!=":
		return actual != expected
	}
	return false
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case float64:
		return val
	case float32:
		return float64(val)
	}
	return 0
}

func (s *AutoResponseService) CancelAction(actionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	action, exists := s.activeActions[actionID]
	if !exists {
		return fmt.Errorf("action not found: %s", actionID)
	}

	action.IsActive = false
	return nil
}

func (s *AutoResponseService) GetResponseStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_rules":         len(s.responseRules),
		"active_rules":        0,
		"total_actions":       len(s.activeActions),
		"processed_threats":   atomic.LoadInt32(&s.processedCount),
		"failed_actions":      atomic.LoadInt32(&s.failedCount),
		"notification_channels": len(s.notificationChannels),
		"automation_workflows": len(s.automationWorkflows),
		"containment_policies": len(s.containmentPolicies),
		"is_enabled":          s.isEnabled,
		"last_action":         s.lastAction,
	}

	activeRules := 0
	actionTypeCount := make(map[string]int)

	for _, rule := range s.responseRules {
		if rule.IsActive {
			activeRules++
		}
	}

	for _, action := range s.activeActions {
		actionTypeCount[string(action.ActionType)]++
	}

	stats["active_rules"] = activeRules
	stats["action_type_distribution"] = actionTypeCount

	var totalTriggers int
	for _, rule := range s.responseRules {
		totalTriggers += rule.TriggerCount
	}
	stats["total_rule_triggers"] = totalTriggers

	return stats
}

func (s *AutoResponseService) Enable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isEnabled = true
}

func (s *AutoResponseService) Disable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isEnabled = false
}

func (s *AutoResponseService) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isEnabled
}

func (s *AutoResponseService) ExportConfiguration() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config := map[string]interface{}{
		"response_rules":      s.responseRules,
		"escalation_matrix":   s.escalationMatrix,
		"notification_channels": s.notificationChannels,
		"containment_policies": s.containmentPolicies,
		"is_enabled":          s.isEnabled,
		"export_time":         time.Now(),
	}

	return json.MarshalIndent(config, "", "  ")
}

func (s *AutoResponseService) ImportConfiguration(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	if enabled, ok := config["is_enabled"].(bool); ok {
		s.isEnabled = enabled
	}

	return nil
}

func (s *AutoResponseService) CreateContainmentPolicy(policy *ContainmentPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	policy.ID = fmt.Sprintf("policy_%d", time.Now().UnixNano())
	policy.CreatedAt = time.Now()
	policy.IsActive = true
	s.containmentPolicies[policy.ID] = policy
	return nil
}

func (s *AutoResponseService) GetContainmentPolicies() []*ContainmentPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policies := make([]*ContainmentPolicy, 0, len(s.containmentPolicies))
	for _, policy := range s.containmentPolicies {
		policies = append(policies, policy)
	}
	return policies
}

func (s *AutoResponseService) TestResponseAction(ctx context.Context, actionType ActionType, target *ActionTarget) (*ResponseResult, error) {
	rule := &ResponseRule{
		ID:   "test_rule",
		Name: "Test Rule",
		Action: ResponseAction{
			Type:     actionType,
			Target:   target,
			Duration: 1 * time.Minute,
		},
	}

	threat := &ThreatContext{
		IP:          target.IP,
		UserID:      target.UserID,
		SessionID:   target.SessionID,
		ThreatLevel: 5,
	}

	return s.executeAction(ctx, rule, threat)
}

func (s *AutoResponseService) GetEscalationStatus(threatID string) (*EscalationStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &EscalationStatus{
		ThreatID:     threatID,
		CurrentLevel: 1,
		MaxLevel:     s.escalationMatrix.MaxEscalation,
		IsEscalated:  false,
		History:      []EscalationEvent{},
	}, nil
}

type EscalationStatus struct {
	ThreatID     string
	CurrentLevel int
	MaxLevel     int
	IsEscalated  bool
	History      []EscalationEvent
}

type EscalationEvent struct {
	Level     int
	Timestamp time.Time
	Action    string
	Success   bool
}

func (s *AutoResponseService) AnalyzeResponseEffectiveness() *ResponseEffectivenessReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	report := &ResponseEffectivenessReport{
		GeneratedAt: time.Now(),
	}

	var totalActions int
	var successfulActions int
	actionTypeStats := make(map[string]*ActionStats)
	threatTypeStats := make(map[string]*ThreatStats)

	for _, record := range s.actionHistory {
		totalActions++
		if record.Success {
			successfulActions++
		}

		if _, exists := actionTypeStats[string(record.ActionType)]; !exists {
			actionTypeStats[string(record.ActionType)] = &ActionStats{}
		}
		actionTypeStats[string(record.ActionType)].Total++
		if record.Success {
			actionTypeStats[string(record.ActionType)].Successful++
		}
	}

	for _, record := range s.actionHistory {
		if _, exists := threatTypeStats[record.RuleName]; !exists {
			threatTypeStats[record.RuleName] = &ThreatStats{}
		}
		threatTypeStats[record.RuleName].Count++
	}

	if totalActions > 0 {
		report.OverallEffectiveness = float64(successfulActions) / float64(totalActions)
	}
	report.ActionTypeStats = actionTypeStats
	report.ThreatTypeStats = threatTypeStats

	return report
}

type ResponseEffectivenessReport struct {
	GeneratedAt          time.Time
	OverallEffectiveness float64
	ActionTypeStats      map[string]*ActionStats
	ThreatTypeStats      map[string]*ThreatStats
}

type ActionStats struct {
	Total      int
	Successful int
}

type ThreatStats struct {
	Count int
}
