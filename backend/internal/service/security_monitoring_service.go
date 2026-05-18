package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type SecurityAlert struct {
	ID          string
	Timestamp   time.Time
	AlertType   SecurityAlertType
	Severity    AlertSeverity
	Source      string
	Description string
	Details     map[string]interface{}
	Status      AlertStatus
	AcknowledgedBy string
	AcknowledgedAt time.Time
}

type SecurityAlertType string

const (
	AlertTypeAuthFailure     SecurityAlertType = "auth_failure"
	AlertTypeBruteForce      SecurityAlertType = "brute_force"
	AlertTypeSQLInjection    SecurityAlertType = "sql_injection"
	AlertTypeXSSAttack       SecurityAlertType = "xss_attack"
	AlertTypeCSRFDetected    SecurityAlertType = "csrf_detected"
	AlertTypeRateLimitExceeded SecurityAlertType = "rate_limit_exceeded"
	AlertTypeDDoSDetected    SecurityAlertType = "ddos_detected"
	AlertTypeSuspiciousIP    SecurityAlertType = "suspicious_ip"
	AlertTypePrivilegeEscalation SecurityAlertType = "privilege_escalation"
	AlertTypeDataBreach      SecurityAlertType = "data_breach"
	AlertTypeSessionHijack   SecurityAlertType = "session_hijack"
	AlertTypeCredentialStuffing SecurityAlertType = "credential_stuffing"
)

type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityHigh     AlertSeverity = "high"
	SeverityMedium   AlertSeverity = "medium"
	SeverityLow      AlertSeverity = "low"
	SeverityInfo     AlertSeverity = "info"
)

type AlertStatus string

const (
	AlertStatusNew          AlertStatus = "new"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
	AlertStatusDismissed    AlertStatus = "dismissed"
)

type AlertRule struct {
	ID              string
	Name            string
	AlertType       SecurityAlertType
	Severity        AlertSeverity
	Condition       AlertCondition
	Threshold       int
	TimeWindow      time.Duration
	Enabled         bool
	Actions         []AlertAction
	Cooldown        time.Duration
	lastAlertTime   map[string]time.Time
}

type AlertCondition struct {
	Field    string
	Operator string
	Value    interface{}
}

type AlertAction struct {
	Type      string
	Target    string
	Template  string
	Priority  int
}

type SecurityMonitoringService struct {
	alerts         []*SecurityAlert
	rules          []*AlertRule
	alertHandlers   []AlertHandler
	ipTracking     map[string]*IPTrackingData
	mu             sync.RWMutex
	maxAlerts      int
	retentionDays  int
	enableAutoBlock bool
	autoBlockThreshold int
	autoBlockDuration time.Duration
}

type IPTrackingData struct {
	IP              string
	RequestCount    int
	AuthFailures    int
	SQLInjections   int
	XSSAttempts     int
	RateLimitHits   int
	LastSeen        time.Time
	FirstSeen       time.Time
	IsBlocked       bool
	BlockExpiresAt   time.Time
	Country         string
	ASN             string
	Reputation      float64
	ThreatCategories []string
}

type AlertHandler func(alert *SecurityAlert)

func NewSecurityMonitoringService() *SecurityMonitoringService {
	service := &SecurityMonitoringService{
		alerts:           make([]*SecurityAlert, 0),
		rules:             make([]*AlertRule, 0),
		alertHandlers:     make([]AlertHandler, 0),
		ipTracking:       make(map[string]*IPTrackingData),
		maxAlerts:         10000,
		retentionDays:    30,
		enableAutoBlock:  true,
		autoBlockThreshold: 100,
		autoBlockDuration: 30 * time.Minute,
	}
	
	service.initDefaultRules()
	
	return service
}

func (s *SecurityMonitoringService) initDefaultRules() {
	s.rules = append(s.rules, &AlertRule{
		ID:        "rule-001",
		Name:      "连续认证失败",
		AlertType: AlertTypeBruteForce,
		Severity:  SeverityHigh,
		Condition:  AlertCondition{Field: "auth_failures", Operator: ">=", Value: 5},
		Threshold: 5,
		TimeWindow: 10 * time.Minute,
		Enabled:   true,
		Actions: []AlertAction{
			{Type: "log", Target: "security_log", Priority: 1},
			{Type: "notify", Target: "admin", Priority: 2},
		},
		Cooldown: 5 * time.Minute,
		lastAlertTime: make(map[string]time.Time),
	})

	s.rules = append(s.rules, &AlertRule{
		ID:        "rule-002",
		Name:      "SQL注入检测",
		AlertType: AlertTypeSQLInjection,
		Severity:  SeverityCritical,
		Condition:  AlertCondition{Field: "sql_injections", Operator: ">=", Value: 1},
		Threshold: 1,
		TimeWindow: 1 * time.Minute,
		Enabled:   true,
		Actions: []AlertAction{
			{Type: "block", Target: "ip", Priority: 1},
			{Type: "log", Target: "security_log", Priority: 2},
			{Type: "notify", Target: "admin", Priority: 3},
		},
		Cooldown: 15 * time.Minute,
		lastAlertTime: make(map[string]time.Time),
	})

	s.rules = append(s.rules, &AlertRule{
		ID:        "rule-003",
		Name:      "XSS攻击检测",
		AlertType: AlertTypeXSSAttack,
		Severity:  SeverityHigh,
		Condition:  AlertCondition{Field: "xss_attempts", Operator: ">=", Value: 1},
		Threshold: 1,
		TimeWindow: 1 * time.Minute,
		Enabled:   true,
		Actions: []AlertAction{
			{Type: "log", Target: "security_log", Priority: 1},
			{Type: "notify", Target: "admin", Priority: 2},
		},
		Cooldown: 10 * time.Minute,
		lastAlertTime: make(map[string]time.Time),
	})

	s.rules = append(s.rules, &AlertRule{
		ID:        "rule-004",
		Name:      "速率限制触发",
		AlertType: AlertTypeRateLimitExceeded,
		Severity:  SeverityMedium,
		Condition:  AlertCondition{Field: "rate_limit_hits", Operator: ">=", Value: 50},
		Threshold: 50,
		TimeWindow: 1 * time.Minute,
		Enabled:   true,
		Actions: []AlertAction{
			{Type: "log", Target: "security_log", Priority: 1},
		},
		Cooldown: 5 * time.Minute,
		lastAlertTime: make(map[string]time.Time),
	})

	s.rules = append(s.rules, &AlertRule{
		ID:        "rule-005",
		Name:      "可疑IP",
		AlertType: AlertTypeSuspiciousIP,
		Severity:  SeverityHigh,
		Condition:  AlertCondition{Field: "request_count", Operator: ">", Value: 1000},
		Threshold: 1000,
		TimeWindow: 1 * time.Hour,
		Enabled:   true,
		Actions: []AlertAction{
			{Type: "log", Target: "security_log", Priority: 1},
			{Type: "notify", Target: "admin", Priority: 2},
		},
		Cooldown: 30 * time.Minute,
		lastAlertTime: make(map[string]time.Time),
	})
}

func (s *SecurityMonitoringService) TrackRequest(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tracking, exists := s.ipTracking[ip]
	if !exists {
		tracking = &IPTrackingData{
			IP:              ip,
			FirstSeen:       time.Now(),
			ThreatCategories: make([]string, 0),
		}
		s.ipTracking[ip] = tracking
	}

	tracking.RequestCount++
	tracking.LastSeen = time.Now()

	s.evaluateRules(ip, tracking)
}

func (s *SecurityMonitoringService) TrackAuthFailure(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tracking, exists := s.ipTracking[ip]
	if !exists {
		tracking = &IPTrackingData{
			IP:        ip,
			FirstSeen: time.Now(),
		}
		s.ipTracking[ip] = tracking
	}

	tracking.AuthFailures++
	tracking.LastSeen = time.Now()

	s.evaluateRules(ip, tracking)
}

func (s *SecurityMonitoringService) TrackSQLInjection(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tracking, exists := s.ipTracking[ip]
	if !exists {
		tracking = &IPTrackingData{
			IP:        ip,
			FirstSeen: time.Now(),
		}
		s.ipTracking[ip] = tracking
	}

	tracking.SQLInjections++
	tracking.LastSeen = time.Now()
	tracking.ThreatCategories = append(tracking.ThreatCategories, "sql_injection")

	s.evaluateRules(ip, tracking)
}

func (s *SecurityMonitoringService) TrackXSSAttempt(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tracking, exists := s.ipTracking[ip]
	if !exists {
		tracking = &IPTrackingData{
			IP:        ip,
			FirstSeen: time.Now(),
		}
		s.ipTracking[ip] = tracking
	}

	tracking.XSSAttempts++
	tracking.LastSeen = time.Now()
	tracking.ThreatCategories = append(tracking.ThreatCategories, "xss")

	s.evaluateRules(ip, tracking)
}

func (s *SecurityMonitoringService) TrackRateLimitHit(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tracking, exists := s.ipTracking[ip]
	if !exists {
		tracking = &IPTrackingData{
			IP:        ip,
			FirstSeen: time.Now(),
		}
		s.ipTracking[ip] = tracking
	}

	tracking.RateLimitHits++
	tracking.LastSeen = time.Now()

	s.evaluateRules(ip, tracking)
}

func (s *SecurityMonitoringService) evaluateRules(ip string, tracking *IPTrackingData) {
	for _, rule := range s.rules {
		if !rule.Enabled {
			continue
		}

		if s.shouldSuppressAlert(rule, ip) {
			continue
		}

		if s.checkCondition(rule, tracking) {
			alert := s.createAlert(rule, tracking)
			s.alerts = append(s.alerts, alert)
			s.notifyHandlers(alert)

			if s.enableAutoBlock && s.shouldAutoBlock(tracking) {
				s.blockIP(ip)
			}

			rule.lastAlertTime[ip] = time.Now()
		}
	}
}

func (s *SecurityMonitoringService) shouldSuppressAlert(rule *AlertRule, ip string) bool {
	if lastTime, exists := rule.lastAlertTime[ip]; exists {
		if time.Since(lastTime) < rule.Cooldown {
			return true
		}
	}
	return false
}

func (s *SecurityMonitoringService) checkCondition(rule *AlertRule, tracking *IPTrackingData) bool {
	fieldValue := s.getFieldValue(rule.Condition.Field, tracking)
	thresholdValue := rule.Threshold

	switch rule.Condition.Operator {
	case ">=":
		return fieldValue >= thresholdValue
	case ">":
		return fieldValue > thresholdValue
	case "==":
		return fieldValue == thresholdValue
	case "<=":
		return fieldValue <= thresholdValue
	case "<":
		return fieldValue < thresholdValue
	}
	return false
}

func (s *SecurityMonitoringService) getFieldValue(field string, tracking *IPTrackingData) int {
	switch field {
	case "auth_failures":
		return tracking.AuthFailures
	case "sql_injections":
		return tracking.SQLInjections
	case "xss_attempts":
		return tracking.XSSAttempts
	case "rate_limit_hits":
		return tracking.RateLimitHits
	case "request_count":
		return tracking.RequestCount
	}
	return 0
}

func (s *SecurityMonitoringService) shouldAutoBlock(tracking *IPTrackingData) bool {
	return tracking.AuthFailures >= s.autoBlockThreshold ||
		   tracking.SQLInjections >= 5 ||
		   tracking.XSSAttempts >= 10
}

func (s *SecurityMonitoringService) blockIP(ip string) {
	if tracking, exists := s.ipTracking[ip]; exists {
		tracking.IsBlocked = true
		tracking.BlockExpiresAt = time.Now().Add(s.autoBlockDuration)
	}
}

func (s *SecurityMonitoringService) createAlert(rule *AlertRule, tracking *IPTrackingData) *SecurityAlert {
	return &SecurityAlert{
		ID:          fmt.Sprintf("alert-%s-%d", rule.ID, time.Now().Unix()),
		Timestamp:   time.Now(),
		AlertType:   rule.AlertType,
		Severity:    rule.Severity,
		Source:      tracking.IP,
		Description: rule.Name,
		Details: map[string]interface{}{
			"request_count":   tracking.RequestCount,
			"auth_failures":   tracking.AuthFailures,
			"sql_injections":  tracking.SQLInjections,
			"xss_attempts":    tracking.XSSAttempts,
			"rate_limit_hits": tracking.RateLimitHits,
			"threat_categories": tracking.ThreatCategories,
		},
		Status: AlertStatusNew,
	}
}

func (s *SecurityMonitoringService) notifyHandlers(alert *SecurityAlert) {
	for _, handler := range s.alertHandlers {
		handler(alert)
	}
}

func (s *SecurityMonitoringService) RegisterAlertHandler(handler AlertHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertHandlers = append(s.alertHandlers, handler)
}

func (s *SecurityMonitoringService) GetAlerts(filter AlertFilter) []*SecurityAlert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alerts := make([]*SecurityAlert, 0)
	for _, alert := range s.alerts {
		if filter.Match(alert) {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

type AlertFilter struct {
	Type     SecurityAlertType
	Severity AlertSeverity
	Status   AlertStatus
	Source   string
	StartTime time.Time
	EndTime  time.Time
	Limit    int
}

func (f AlertFilter) Match(alert *SecurityAlert) bool {
	if f.Type != "" && alert.AlertType != f.Type {
		return false
	}
	if f.Severity != "" && alert.Severity != f.Severity {
		return false
	}
	if f.Status != "" && alert.Status != f.Status {
		return false
	}
	if f.Source != "" && alert.Source != f.Source {
		return false
	}
	if !f.StartTime.IsZero() && alert.Timestamp.Before(f.StartTime) {
		return false
	}
	if !f.EndTime.IsZero() && alert.Timestamp.After(f.EndTime) {
		return false
	}
	return true
}

func (s *SecurityMonitoringService) AcknowledgeAlert(alertID, acknowledgedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, alert := range s.alerts {
		if alert.ID == alertID {
			alert.Status = AlertStatusAcknowledged
			alert.AcknowledgedBy = acknowledgedBy
			alert.AcknowledgedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("alert not found: %s", alertID)
}

func (s *SecurityMonitoringService) ResolveAlert(alertID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, alert := range s.alerts {
		if alert.ID == alertID {
			alert.Status = AlertStatusResolved
			return nil
		}
	}
	return fmt.Errorf("alert not found: %s", alertID)
}

func (s *SecurityMonitoringService) GetIPStatus(ip string) *IPTrackingData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if tracking, exists := s.ipTracking[ip]; exists {
		if time.Now().After(tracking.BlockExpiresAt) {
			tracking.IsBlocked = false
		}
		return tracking
	}
	return nil
}

func (s *SecurityMonitoringService) GetSecurityStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_alerts":       len(s.alerts),
		"active_rules":       len(s.rules),
		"tracked_ips":        len(s.ipTracking),
		"by_severity":        make(map[string]int),
		"by_type":            make(map[string]int),
		"by_status":          make(map[string]int),
	}

	for _, alert := range s.alerts {
		stats["by_severity"].(map[string]int)[string(alert.Severity)]++
		stats["by_type"].(map[string]int)[string(alert.AlertType)]++
		stats["by_status"].(map[string]int)[string(alert.Status)]++
	}

	return stats
}

func (s *SecurityMonitoringService) CleanupOldData() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-time.Duration(s.retentionDays) * 24 * time.Hour)

	newAlerts := make([]*SecurityAlert, 0)
	for _, alert := range s.alerts {
		if alert.Timestamp.After(cutoff) {
			newAlerts = append(newAlerts, alert)
		}
	}
	s.alerts = newAlerts

	for ip, tracking := range s.ipTracking {
		if tracking.LastSeen.Before(cutoff) && !tracking.IsBlocked {
			delete(s.ipTracking, ip)
		}
	}

	if len(s.alerts) > s.maxAlerts {
		s.alerts = s.alerts[len(s.alerts)-s.maxAlerts:]
	}
}

func (s *SecurityMonitoringService) ExportAlerts(format string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch format {
	case "json":
		return json.MarshalIndent(s.alerts, "", "  ")
	case "csv":
		return s.exportAlertsCSV()
	default:
		return json.Marshal(s.alerts)
	}
}

func (s *SecurityMonitoringService) exportAlertsCSV() ([]byte, error) {
	var sb strings.Builder
	sb.WriteString("ID,Timestamp,Type,Severity,Source,Description,Status\n")

	for _, alert := range s.alerts {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s\n",
			alert.ID,
			alert.Timestamp.Format(time.RFC3339),
			alert.AlertType,
			alert.Severity,
			alert.Source,
			alert.Description,
			alert.Status,
		))
	}

	return []byte(sb.String()), nil
}

func (s *SecurityMonitoringService) AddRule(rule *AlertRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = append(s.rules, rule)
}

func (s *SecurityMonitoringService) RemoveRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, rule := range s.rules {
		if rule.ID == ruleID {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("rule not found: %s", ruleID)
}

func (s *SecurityMonitoringService) EnableRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rule := range s.rules {
		if rule.ID == ruleID {
			rule.Enabled = true
			return nil
		}
	}
	return fmt.Errorf("rule not found: %s", ruleID)
}

func (s *SecurityMonitoringService) DisableRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rule := range s.rules {
		if rule.ID == ruleID {
			rule.Enabled = false
			return nil
		}
	}
	return fmt.Errorf("rule not found: %s", ruleID)
}

func ValidateIPAddress(ip string) bool {
	ipPattern := regexp.MustCompile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
	return ipPattern.MatchString(ip)
}

func ValidateDomain(domain string) bool {
	domainPattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]?\.[a-zA-Z]{2,}$`)
	return domainPattern.MatchString(domain)
}

func ValidateURL(url string) bool {
	urlPattern := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return urlPattern.MatchString(url)
}

func DetectSuspiciousUserAgent(ua string) bool {
	suspiciousPatterns := []string{
		"curl",
		"wget",
		"python-requests",
		"scrapy",
		"bot",
		"crawler",
		"spider",
		"scan",
		"test",
		"scan",
	}

	lowerUA := strings.ToLower(ua)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerUA, pattern) {
			return true
		}
	}
	return false
}
