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

func getAuditClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

type AuditCategory string

const (
	AuditCategoryOperation  AuditCategory = "operation"
	AuditCategorySecurity   AuditCategory = "security"
	AuditCategoryAnomaly   AuditCategory = "anomaly"
	AuditCategoryCompliance AuditCategory = "compliance"
)

type AuditSeverity string

const (
	AuditSeverityDebug     AuditSeverity = "debug"
	AuditSeverityInfo      AuditSeverity = "info"
	AuditSeverityWarning   AuditSeverity = "warning"
	AuditSeverityError     AuditSeverity = "error"
	AuditSeverityCritical  AuditSeverity = "critical"
)

type AuditStatus string

const (
	AuditStatusPending   AuditStatus = "pending"
	AuditStatusSuccess   AuditStatus = "success"
	AuditStatusFailure   AuditStatus = "failure"
	AuditStatusBlocked   AuditStatus = "blocked"
	AuditStatusWarning   AuditStatus = "warning"
)

type AuditRecord struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	Category        AuditCategory          `json:"category"`
	Severity        AuditSeverity         `json:"severity"`
	Status          AuditStatus            `json:"status"`
	SourceIP        string                 `json:"source_ip"`
	UserID          string                 `json:"user_id,omitempty"`
	SessionID       string                 `json:"session_id,omitempty"`
	Operation       string                 `json:"operation"`
	Resource        string                 `json:"resource"`
	Action          string                 `json:"action"`
	Result          string                 `json:"result,omitempty"`
	Details         map[string]interface{} `json:"details,omitempty"`
	UserAgent       string                 `json:"user_agent,omitempty"`
	RequestMethod   string                 `json:"request_method,omitempty"`
	RequestPath     string                 `json:"request_path,omitempty"`
	ResponseCode    int                    `json:"response_code,omitempty"`
	Duration        time.Duration          `json:"duration,omitempty"`
	ComplianceTags  []string               `json:"compliance_tags,omitempty"`
	RiskLevel       string                 `json:"risk_level,omitempty"`
}

type ComplianceRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Framework   string   `json:"framework"`
	Tags        []string `json:"tags"`
	Severity    AuditSeverity `json:"severity"`
	Enabled     bool     `json:"enabled"`
	Checks      []ComplianceCheck `json:"checks"`
}

type ComplianceCheck struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CheckType   string `json:"check_type"`
	Pattern     string `json:"pattern,omitempty"`
	Severity    AuditSeverity `json:"severity"`
	Action      string `json:"action"`
}

type ComplianceViolation struct {
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	Framework string    `json:"framework"`
	Violation string    `json:"violation"`
	Severity  AuditSeverity `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
	Evidence  map[string]interface{} `json:"evidence"`
	Remediation string `json:"remediation"`
}

type AnomalyPattern struct {
	PatternID    string    `json:"pattern_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Severity     AuditSeverity `json:"severity"`
	Threshold    float64   `json:"threshold"`
	WindowSize  time.Duration `json:"window_size"`
	MetricType  string    `json:"metric_type"`
	IsActive    bool      `json:"is_active"`
}

type SecurityEnhancedAuditService struct {
	records           []*AuditRecord
	anomalyPatterns  map[string]*AnomalyPattern
	complianceRules   map[string]*ComplianceRule
	violations        []*ComplianceViolation
	alertHandlers     []func(record *AuditRecord)
	anomalyHandlers   []func(pattern *AnomalyPattern, evidence map[string]interface{})

	mu               sync.RWMutex
	maxRecords       int
	asyncMode        bool
	eventBuffer      chan *AuditRecord
}

func NewSecurityEnhancedAuditService() *SecurityEnhancedAuditService {
	svc := &SecurityEnhancedAuditService{
		records:         make([]*AuditRecord, 0),
		anomalyPatterns: make(map[string]*AnomalyPattern),
		complianceRules: make(map[string]*ComplianceRule),
		violations:      make([]*ComplianceViolation, 0),
		alertHandlers:   make([]func(record *AuditRecord), 0),
		anomalyHandlers: make([]func(pattern *AnomalyPattern, evidence map[string]interface{}), 0),
		maxRecords:      100000,
		asyncMode:      true,
		eventBuffer:     make(chan *AuditRecord, 10000),
	}

	svc.initDefaultAnomalyPatterns()
	svc.initDefaultComplianceRules()

	go svc.processEvents()

	return svc
}

func (s *SecurityEnhancedAuditService) initDefaultAnomalyPatterns() {
	s.anomalyPatterns = map[string]*AnomalyPattern{
		"rapid_requests": {
			PatternID:   "rapid_requests",
			Name:        "Rapid Request Pattern",
			Description: "Detects abnormally fast request patterns",
			Severity:    AuditSeverityWarning,
			Threshold:   100,
			WindowSize:  time.Minute,
			MetricType:  "requests_per_second",
			IsActive:    true,
		},
		"unusual_hours": {
			PatternID:   "unusual_hours",
			Name:        "Unusual Access Hours",
			Description: "Detects access outside normal business hours",
			Severity:    AuditSeverityWarning,
			Threshold:   1,
			WindowSize:  time.Hour,
			MetricType:  "access_outside_hours",
			IsActive:    true,
		},
		"high_error_rate": {
			PatternID:   "high_error_rate",
			Name:        "High Error Rate",
			Description: "Detects abnormally high error rates",
			Severity:    AuditSeverityError,
			Threshold:   0.3,
			WindowSize:  5 * time.Minute,
			MetricType:  "error_rate",
			IsActive:    true,
		},
		"data_exfiltration": {
			PatternID:   "data_exfiltration",
			Name:        "Potential Data Exfiltration",
			Description: "Detects unusual bulk data access patterns",
			Severity:    AuditSeverityCritical,
			Threshold:   1000,
			WindowSize:  10 * time.Minute,
			MetricType:  "data_access_count",
			IsActive:    true,
		},
		"privilege_escalation": {
			PatternID:   "privilege_escalation",
			Name:        "Privilege Escalation Attempt",
			Description: "Detects attempts to escalate privileges",
			Severity:    AuditSeverityCritical,
			Threshold:   1,
			WindowSize:  time.Minute,
			MetricType:  "privilege_change_attempts",
			IsActive:    true,
		},
		"brute_force": {
			PatternID:   "brute_force",
			Name:        "Brute Force Attack",
			Description: "Detects rapid failed login attempts",
			Severity:    AuditSeverityCritical,
			Threshold:   10,
			WindowSize:  5 * time.Minute,
			MetricType:  "failed_login_attempts",
			IsActive:    true,
		},
	}
}

func (s *SecurityEnhancedAuditService) initDefaultComplianceRules() {
	s.complianceRules = map[string]*ComplianceRule{
		"gdpr_data_access": {
			ID:          "gdpr_data_access",
			Name:        "GDPR Data Access Logging",
			Description: "Ensures all personal data access is logged",
			Framework:   "GDPR",
			Tags:        []string{"privacy", "data-access", "personal-data"},
			Severity:    AuditSeverityWarning,
			Enabled:     true,
			Checks: []ComplianceCheck{
				{
					ID:          "gdpr_001",
					Name:        "Data access event logging",
					Description: "Verify data access events are logged",
					CheckType:   "event_logging",
					Severity:    AuditSeverityWarning,
					Action:      "log",
				},
			},
		},
		"pci_data_protection": {
			ID:          "pci_data_protection",
			Name:        "PCI DSS Data Protection",
			Description: "PCI DSS compliance for cardholder data",
			Framework:   "PCI-DSS",
			Tags:        []string{"payment", "card-data", "pii"},
			Severity:    AuditSeverityCritical,
			Enabled:     true,
			Checks: []ComplianceCheck{
				{
					ID:          "pci_001",
					Name:        "Card data access logging",
					Description: "All access to cardholder data must be logged",
					CheckType:   "event_logging",
					Severity:    AuditSeverityCritical,
					Action:      "block",
				},
				{
					ID:          "pci_002",
					Name:        "Sensitive data masking",
					Description: "Card numbers must be masked in logs",
					CheckType:   "data_masking",
					Severity:    AuditSeverityError,
					Action:      "alert",
				},
			},
		},
		"hipaa_audit_controls": {
			ID:          "hipaa_audit_controls",
			Name:        "HIPAA Audit Controls",
			Description: "Implement and maintain audit controls",
			Framework:   "HIPAA",
			Tags:        []string{"healthcare", "phi", "audit"},
			Severity:    AuditSeverityCritical,
			Enabled:     true,
			Checks: []ComplianceCheck{
				{
					ID:          "hipaa_001",
					Name:        "PHI access logging",
					Description: "Log all access to Protected Health Information",
					CheckType:   "event_logging",
					Severity:    AuditSeverityCritical,
					Action:      "enforce",
				},
			},
		},
		"sox_access_control": {
			ID:          "sox_access_control",
			Name:        "SOX Access Control",
			Description: "SOX compliance for financial data access",
			Framework:   "SOX",
			Tags:        []string{"financial", "access-control", "segregation-of-duties"},
			Severity:    AuditSeverityError,
			Enabled:     true,
			Checks: []ComplianceCheck{
				{
					ID:          "sox_001",
					Name:        "Financial data access audit",
					Description: "All financial data access must be audited",
					CheckType:   "event_logging",
					Severity:    AuditSeverityError,
					Action:      "log",
				},
			},
		},
	}
}

func (s *SecurityEnhancedAuditService) LogOperation(operation string, resource string, action string, r *http.Request, details map[string]interface{}) *AuditRecord {
	record := s.createAuditRecord(AuditCategoryOperation, AuditSeverityInfo, r)
	record.Operation = operation
	record.Resource = resource
	record.Action = action
	record.Details = details

	return s.saveRecord(record)
}

func (s *SecurityEnhancedAuditService) LogSecurityEvent(severity AuditSeverity, operation string, r *http.Request, details map[string]interface{}) *AuditRecord {
	record := s.createAuditRecord(AuditCategorySecurity, severity, r)
	record.Operation = operation
	record.Details = details

	return s.saveRecord(record)
}

func (s *SecurityEnhancedAuditService) LogAccessDenied(reason string, r *http.Request, details map[string]interface{}) *AuditRecord {
	record := s.createAuditRecord(AuditCategorySecurity, AuditSeverityWarning, r)
	record.Operation = "access_denied"
	record.Result = reason
	record.Status = AuditStatusBlocked
	record.Details = details

	return s.saveRecord(record)
}

func (s *SecurityEnhancedAuditService) LogAnomaly(patternID string, r *http.Request, evidence map[string]interface{}) *AuditRecord {
	pattern, exists := s.anomalyPatterns[patternID]
	severity := AuditSeverityWarning
	if exists {
		severity = pattern.Severity
	}

	record := s.createAuditRecord(AuditCategoryAnomaly, severity, r)
	record.Operation = "anomaly_detected"
	record.Result = patternID
	record.Details = evidence
	record.RiskLevel = string(severity)

	auditRecord := s.saveRecord(record)

	if exists && s.asyncMode {
		go func() {
			for _, handler := range s.anomalyHandlers {
				handler(pattern, evidence)
			}
		}()
	} else if exists {
		for _, handler := range s.anomalyHandlers {
			handler(pattern, evidence)
		}
	}

	return auditRecord
}

func (s *SecurityEnhancedAuditService) LogComplianceViolation(violation *ComplianceViolation) *AuditRecord {
	record := &AuditRecord{
		ID:             generateAuditID(),
		Timestamp:      time.Now(),
		Category:       AuditCategoryCompliance,
		Severity:       violation.Severity,
		Status:         AuditStatusFailure,
		Operation:      "compliance_violation",
		Resource:       violation.RuleName,
		Result:         violation.Violation,
		ComplianceTags: []string{violation.Framework},
		Details:        violation.Evidence,
		RiskLevel:      string(violation.Severity),
	}

	return s.saveRecord(record)
}

func (s *SecurityEnhancedAuditService) createAuditRecord(category AuditCategory, severity AuditSeverity, r *http.Request) *AuditRecord {
	record := &AuditRecord{
		ID:        generateAuditID(),
		Timestamp: time.Now(),
		Category:  category,
		Severity:  severity,
		Status:    AuditStatusSuccess,
	}

	if r != nil {
		record.SourceIP = getAuditClientIP(r)
		record.UserAgent = r.UserAgent()
		record.RequestMethod = r.Method
		record.RequestPath = r.URL.Path
	}

	return record
}

func (s *SecurityEnhancedAuditService) saveRecord(record *AuditRecord) *AuditRecord {
	if s.asyncMode {
		select {
		case s.eventBuffer <- record:
		default:
		}
	} else {
		s.storeRecord(record)
		s.checkAlerts(record)
	}

	return record
}

func (s *SecurityEnhancedAuditService) processEvents() {
	for record := range s.eventBuffer {
		s.storeRecord(record)
		s.checkAlerts(record)
	}
}

func (s *SecurityEnhancedAuditService) storeRecord(record *AuditRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = append(s.records, record)

	if len(s.records) > s.maxRecords {
		s.records = s.records[len(s.records)-s.maxRecords:]
	}
}

func (s *SecurityEnhancedAuditService) checkAlerts(record *AuditRecord) {
	if record.Severity == AuditSeverityCritical || record.Severity == AuditSeverityError {
		for _, handler := range s.alertHandlers {
			if s.asyncMode {
				go handler(record)
			} else {
				handler(record)
			}
		}
	}
}

func (s *SecurityEnhancedAuditService) RegisterAlertHandler(handler func(record *AuditRecord)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertHandlers = append(s.alertHandlers, handler)
}

func (s *SecurityEnhancedAuditService) RegisterAnomalyHandler(handler func(pattern *AnomalyPattern, evidence map[string]interface{})) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.anomalyHandlers = append(s.anomalyHandlers, handler)
}

func (s *SecurityEnhancedAuditService) CheckCompliance(r *http.Request, context map[string]interface{}) []*ComplianceViolation {
	violations := make([]*ComplianceViolation, 0)

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, rule := range s.complianceRules {
		if !rule.Enabled {
			continue
		}

		for _, check := range rule.Checks {
			violation := s.runComplianceCheck(check, r, context)
			if violation != nil {
				violations = append(violations, violation)
			}
		}
	}

	return violations
}

func (s *SecurityEnhancedAuditService) runComplianceCheck(check ComplianceCheck, r *http.Request, context map[string]interface{}) *ComplianceViolation {
	sensitivePatterns := map[string]string{
		"card_number":   `\b\d{13,19}\b`,
		"ssn":          `\b\d{3}-\d{2}-\d{4}\b`,
		"email":        `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
		"phone":        `\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`,
	}

	switch check.CheckType {
	case "event_logging":
		if context["logged"] == nil || context["logged"] == false {
			return &ComplianceViolation{
				RuleID:      check.ID,
				RuleName:    check.Name,
				Framework:   "compliance",
				Violation:   "Event not logged as required",
				Severity:    check.Severity,
				Timestamp:   time.Now(),
				Evidence:    map[string]interface{}{"request": r.URL.Path},
				Remediation: "Ensure all sensitive operations are logged",
			}
		}

	case "data_masking":
		path := r.URL.Path
		query := r.URL.RawQuery

		for patternName, pattern := range sensitivePatterns {
			matched, _ := regexp.MatchString(pattern, path+query)
			if matched {
				return &ComplianceViolation{
					RuleID:      check.ID,
					RuleName:    check.Name,
					Framework:   "PCI-DSS",
					Violation:   fmt.Sprintf("Unmasked %s detected", patternName),
					Severity:    check.Severity,
					Timestamp:   time.Now(),
					Evidence:    map[string]interface{}{"pattern": patternName},
					Remediation: "Mask sensitive data before logging or displaying",
				}
			}
		}

	case "access_control":
		if context["authorized"] == nil || context["authorized"] == false {
			return &ComplianceViolation{
				RuleID:      check.ID,
				RuleName:    check.Name,
				Framework:   "SOX",
				Violation:   "Unauthorized access attempted",
				Severity:    check.Severity,
				Timestamp:   time.Now(),
				Evidence:    map[string]interface{}{"path": r.URL.Path},
				Remediation: "Implement proper access controls",
			}
		}
	}

	return nil
}

func (s *SecurityEnhancedAuditService) DetectAnomalies(ip string, window time.Duration) []*AnomalyPattern {
	anomalies := make([]*AnomalyPattern, 0)

	s.mu.RLock()
	defer s.mu.RUnlock()

	cutoff := time.Now().Add(-window)
	records := make([]*AuditRecord, 0)

	for _, record := range s.records {
		if record.SourceIP == ip && record.Timestamp.After(cutoff) {
			records = append(records, record)
		}
	}

	for _, pattern := range s.anomalyPatterns {
		if !pattern.IsActive {
			continue
		}

		detected := s.evaluateAnomalyPattern(pattern, records)
		if detected {
			anomalies = append(anomalies, pattern)
		}
	}

	return anomalies
}

func (s *SecurityEnhancedAuditService) evaluateAnomalyPattern(pattern *AnomalyPattern, records []*AuditRecord) bool {
	switch pattern.PatternID {
	case "rapid_requests":
		if len(records) > int(pattern.Threshold) {
			return true
		}

	case "high_error_rate":
		if len(records) == 0 {
			return false
		}
		errorCount := 0
		for _, record := range records {
			if record.Status == AuditStatusFailure || record.Status == AuditStatusBlocked {
				errorCount++
			}
		}
		errorRate := float64(errorCount) / float64(len(records))
		return errorRate > pattern.Threshold

	case "brute_force":
		failedLogins := 0
		for _, record := range records {
			if record.Operation == "login" && record.Status == AuditStatusFailure {
				failedLogins++
			}
		}
		return float64(failedLogins) > pattern.Threshold

	case "unusual_hours":
		hour := time.Now().Hour()
		if hour < 6 || hour > 22 {
			return true
		}

	case "privilege_escalation":
		for _, record := range records {
			if strings.Contains(record.Operation, "privilege") || strings.Contains(record.Operation, "admin") {
				if record.Status == AuditStatusFailure {
					return true
				}
			}
		}
	}

	return false
}

func (s *SecurityEnhancedAuditService) GetRecords(filter *AuditFilter) []*AuditRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := s.records

	if filter != nil {
		filtered := make([]*AuditRecord, 0)
		for _, record := range records {
			if s.matchesFilter(record, filter) {
				filtered = append(filtered, record)
			}
		}
		records = filtered
	}

	if filter != nil && filter.Limit > 0 && len(records) > filter.Limit {
		records = records[len(records)-filter.Limit:]
	}

	return records
}

func (s *SecurityEnhancedAuditService) matchesFilter(record *AuditRecord, filter *AuditFilter) bool {
	if filter == nil {
		return true
	}

	if filter.Category != "" && record.Category != filter.Category {
		return false
	}

	if filter.Severity != "" && record.Severity != filter.Severity {
		return false
	}

	if filter.SourceIP != "" && record.SourceIP != filter.SourceIP {
		return false
	}

	if filter.UserID != "" && record.UserID != filter.UserID {
		return false
	}

	if filter.Operation != "" && record.Operation != filter.Operation {
		return false
	}

	if filter.StartTime != nil && record.Timestamp.Before(*filter.StartTime) {
		return false
	}

	if filter.EndTime != nil && record.Timestamp.After(*filter.EndTime) {
		return false
	}

	if filter.ComplianceTag != "" {
		found := false
		for _, tag := range record.ComplianceTags {
			if tag == filter.ComplianceTag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

type AuditFilter struct {
	Category       AuditCategory
	Severity       AuditSeverity
	SourceIP       string
	UserID         string
	Operation      string
	StartTime      *time.Time
	EndTime        *time.Time
	Limit          int
	ComplianceTag  string
}

func (s *SecurityEnhancedAuditService) GetStatistics() *AuditStatistics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &AuditStatistics{
		TotalRecords:   len(s.records),
		ByCategory:     make(map[string]int),
		BySeverity:     make(map[string]int),
		ByStatus:       make(map[string]int),
		ComplianceViolations: len(s.violations),
	}

	categoryCounts := make(map[AuditCategory]int)
	severityCounts := make(map[AuditSeverity]int)
	statusCounts := make(map[AuditStatus]int)

	for _, record := range s.records {
		categoryCounts[record.Category]++
		severityCounts[record.Severity]++
		statusCounts[record.Status]++
	}

	for cat, count := range categoryCounts {
		stats.ByCategory[string(cat)] = count
	}
	for sev, count := range severityCounts {
		stats.BySeverity[string(sev)] = count
	}
	for status, count := range statusCounts {
		stats.ByStatus[string(status)] = count
	}

	return stats
}

type AuditStatistics struct {
	TotalRecords          int            `json:"total_records"`
	ByCategory            map[string]int `json:"by_category"`
	BySeverity            map[string]int `json:"by_severity"`
	ByStatus              map[string]int `json:"by_status"`
	ComplianceViolations  int            `json:"compliance_violations"`
}

func (s *SecurityEnhancedAuditService) GetAnomalyPatterns() []*AnomalyPattern {
	s.mu.RLock()
	defer s.mu.RUnlock()

	patterns := make([]*AnomalyPattern, 0, len(s.anomalyPatterns))
	for _, pattern := range s.anomalyPatterns {
		patterns = append(patterns, pattern)
	}

	return patterns
}

func (s *SecurityEnhancedAuditService) UpdateAnomalyPattern(patternID string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pattern, exists := s.anomalyPatterns[patternID]
	if !exists {
		return fmt.Errorf("pattern not found: %s", patternID)
	}

	pattern.IsActive = enabled
	return nil
}

func (s *SecurityEnhancedAuditService) GetComplianceRules() []*ComplianceRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]*ComplianceRule, 0, len(s.complianceRules))
	for _, rule := range s.complianceRules {
		rules = append(rules, rule)
	}

	return rules
}

func (s *SecurityEnhancedAuditService) UpdateComplianceRule(ruleID string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule, exists := s.complianceRules[ruleID]
	if !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	rule.Enabled = enabled
	return nil
}

func (s *SecurityEnhancedAuditService) ExportRecords(format string, filter *AuditFilter) ([]byte, error) {
	records := s.GetRecords(filter)

	switch format {
	case "json":
		return json.MarshalIndent(records, "", "  ")
	case "csv":
		return s.exportToCSV(records)
	default:
		return json.Marshal(records)
	}
}

func (s *SecurityEnhancedAuditService) exportToCSV(records []*AuditRecord) ([]byte, error) {
	var builder strings.Builder

	builder.WriteString("ID,Timestamp,Category,Severity,Status,SourceIP,UserID,Operation,Resource,Action,Result,RequestPath,ResponseCode\n")

	for _, record := range records {
		builder.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%d\n",
			record.ID,
			record.Timestamp.Format(time.RFC3339),
			record.Category,
			record.Severity,
			record.Status,
			record.SourceIP,
			record.UserID,
			record.Operation,
			record.Resource,
			record.Action,
			record.Result,
			record.RequestPath,
			record.ResponseCode,
		))
	}

	return []byte(builder.String()), nil
}

func generateAuditID() string {
	return fmt.Sprintf("AUD-%s-%s", time.Now().Format("20060102150405"), randomString(8))
}
