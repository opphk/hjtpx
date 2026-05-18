package service

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type SecurityEventType string

const (
	EventLoginAttempt            SecurityEventType = "login_attempt"
	EventLoginSuccess            SecurityEventType = "login_success"
	EventLoginFailure            SecurityEventType = "login_failure"
	EventLogout                  SecurityEventType = "logout"
	EventAccessDenied            SecurityEventType = "access_denied"
	EventCSRFDetected            SecurityEventType = "csrf_detected"
	EventSQLInjection            SecurityEventType = "sql_injection"
	EventXSSAttempt              SecurityEventType = "xss_attempt"
	EventCommandInjection        SecurityEventType = "command_injection"
	EventPathTraversal           SecurityEventType = "path_traversal"
	EventSSRFAttempt             SecurityEventType = "ssrf_attempt"
	EventRateLimitHit            SecurityEventType = "rate_limit_hit"
	EventBotDetected             SecurityEventType = "bot_detected"
	EventDDoSAttempt             SecurityEventType = "ddos_attempt"
	EventSuspiciousActivity      SecurityEventType = "suspicious_activity"
	EventPrivilegeEscalation     SecurityEventType = "privilege_escalation"
	EventDataAccess              SecurityEventType = "data_access"
	EventConfigChange            SecurityEventType = "config_change"
	EventAccountCreated          SecurityEventType = "account_created"
	EventAccountDeleted          SecurityEventType = "account_deleted"
	EventAccountLocked           SecurityEventType = "account_locked"
	EventAccountUnlocked         SecurityEventType = "account_unlocked"
	EventPasswordChange          SecurityEventType = "password_change"
	EventPasswordReset           SecurityEventType = "password_reset"
	EventPasswordResetRequest    SecurityEventType = "password_reset_request"
	EventAPIKeyGenerated         SecurityEventType = "api_key_generated"
	EventAPIKeyRevoked           SecurityEventType = "api_key_revoked"
	EventSessionCreated          SecurityEventType = "session_created"
	EventSessionExpired          SecurityEventType = "session_expired"
	EventSessionInvalidated      SecurityEventType = "session_invalidated"
	EventSessionHijackAttempt    SecurityEventType = "session_hijack_attempt"
	EventDataExport              SecurityEventType = "data_export"
	EventDataImport              SecurityEventType = "data_import"
	EventBackupCreated           SecurityEventType = "backup_created"
	EventBackupRestored          SecurityEventType = "backup_restored"
	EventBackupDeleted           SecurityEventType = "backup_deleted"
	EventFirewallBlock           SecurityEventType = "firewall_block"
	EventIPReputationWarning     SecurityEventType = "ip_reputation_warning"
	EventIPBlacklisted           SecurityEventType = "ip_blacklisted"
	EventIPWhitelisted           SecurityEventType = "ip_whitelisted"
	EventAnomalyDetected         SecurityEventType = "anomaly_detected"
	EventViolationDetected       SecurityEventType = "violation_detected"
	EventMFAEnabled              SecurityEventType = "mfa_enabled"
	EventMFADisabled             SecurityEventType = "mfa_disabled"
	EventMFAFailed               SecurityEventType = "mfa_failed"
	EventMFASuccess              SecurityEventType = "mfa_success"
	EventCertificateExpiry      SecurityEventType = "certificate_expiry"
	EventDependencyVulnerability SecurityEventType = "dependency_vulnerability"
	EventSecurityScan            SecurityEventType = "security_scan"
	EventPenetrationTest         SecurityEventType = "penetration_test"
	EventInputValidationFailed   SecurityEventType = "input_validation_failed"
	EventInvalidRequest          SecurityEventType = "invalid_request"
	EventPayloadTooLarge         SecurityEventType = "payload_too_large"
	EventUploadBlocked           SecurityEventType = "upload_blocked"
	EventFileInclusion           SecurityEventType = "file_inclusion"
	EventLDAPInjection           SecurityEventType = "ldap_injection"
	EventXMLInjection            SecurityEventType = "xml_injection"
	EventJSONInjection           SecurityEventType = "json_injection"
	EventXXEAttack               SecurityEventType = "xxe_attack"
	EventHTTPParameterPollution  SecurityEventType = "http_parameter_pollution"
	EventOpenRedirect            SecurityEventType = "open_redirect"
	EventClickjackingAttempt     SecurityEventType = "clickjacking_attempt"
)

type SecurityEvent struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	EventType     SecurityEventType      `json:"event_type"`
	Severity      string                 `json:"severity"`
	SourceIP      string                 `json:"source_ip"`
	UserAgent     string                 `json:"user_agent"`
	UserID        string                 `json:"user_id,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	RequestPath   string                 `json:"request_path"`
	RequestMethod string                 `json:"request_method"`
	Details       map[string]interface{} `json:"details,omitempty"`
	Status        string                 `json:"status"`
	GeoLocation   map[string]string      `json:"geo_location,omitempty"`
	ThreatScore   float64                `json:"threat_score,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	RelatedEvents []string               `json:"related_events,omitempty"`
	ActionTaken   string                 `json:"action_taken,omitempty"`
	RawData       string                 `json:"raw_data,omitempty"`
}

type ThreatIntelEntry struct {
	IP              string
	ThreatType      string
	Confidence      float64
	Source          string
	LastSeen        time.Time
	Description     string
}

type SecurityAuditService struct {
	events              []*SecurityEvent
	eventBuffer         chan *SecurityEvent
	alertHandlers       []func(event *SecurityEvent)
	threatIntel         map[string][]*ThreatIntelEntry
	ipEventCounts       map[string]int
	geoIPCache          map[string]map[string]string
	mu                  sync.RWMutex
	maxEvents           int
	severityLevels      map[SecurityEventType]string
	asyncMode           bool // 控制是否使用异步模式，测试时可以设置为false
	enableThreatIntel   bool
	maxThreatScore      float64
	retentionDays       int
	sensitivePatterns   []*regexp.Regexp
}

func NewSecurityAuditService() *SecurityAuditService {
	service := &SecurityAuditService{
		events:            make([]*SecurityEvent, 0),
		eventBuffer:       make(chan *SecurityEvent, 1000),
		alertHandlers:     make([]func(event *SecurityEvent), 0),
		threatIntel:       make(map[string][]*ThreatIntelEntry),
		ipEventCounts:     make(map[string]int),
		geoIPCache:        make(map[string]map[string]string),
		maxEvents:         10000,
		asyncMode:         true,
		enableThreatIntel: true,
		maxThreatScore:    100.0,
		retentionDays:     30,
		sensitivePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(password|secret|token|api[_-]key|credentials)`),
			regexp.MustCompile(`(?i)(ssn|credit[_-]card|cvv|bank[_-]account)`),
			regexp.MustCompile(`(?i)(private[_-]key|ssh[_-]key|certificate)`),
		},
		severityLevels: map[SecurityEventType]string{
			EventLoginAttempt:            "info",
			EventLoginSuccess:            "info",
			EventLoginFailure:            "warning",
			EventLogout:                  "info",
			EventAccessDenied:            "warning",
			EventCSRFDetected:           "high",
			EventSQLInjection:           "critical",
			EventXSSAttempt:             "high",
			EventCommandInjection:        "critical",
			EventPathTraversal:           "high",
			EventSSRFAttempt:            "high",
			EventRateLimitHit:           "warning",
			EventBotDetected:            "medium",
			EventDDoSAttempt:            "critical",
			EventSuspiciousActivity:     "medium",
			EventPrivilegeEscalation:    "critical",
			EventDataAccess:             "info",
			EventConfigChange:           "high",
			EventAccountCreated:         "info",
			EventAccountDeleted:         "warning",
			EventAccountLocked:          "high",
			EventAccountUnlocked:        "warning",
			EventPasswordChange:         "info",
			EventPasswordReset:          "warning",
			EventPasswordResetRequest:   "warning",
			EventAPIKeyGenerated:        "high",
			EventAPIKeyRevoked:          "warning",
			EventSessionCreated:         "info",
			EventSessionExpired:         "info",
			EventSessionInvalidated:     "warning",
			EventSessionHijackAttempt:   "critical",
			EventDataExport:             "high",
			EventDataImport:             "high",
			EventBackupCreated:          "info",
			EventBackupRestored:         "warning",
			EventBackupDeleted:         "warning",
			EventFirewallBlock:         "medium",
			EventIPReputationWarning:   "warning",
			EventIPBlacklisted:         "high",
			EventIPWhitelisted:         "info",
			EventAnomalyDetected:       "high",
			EventViolationDetected:     "high",
			EventMFAEnabled:            "info",
			EventMFADisabled:           "warning",
			EventMFAFailed:             "warning",
			EventMFASuccess:            "info",
			EventCertificateExpiry:     "high",
			EventDependencyVulnerability: "high",
			EventSecurityScan:          "info",
			EventPenetrationTest:       "warning",
			EventInputValidationFailed: "warning",
			EventInvalidRequest:        "warning",
			EventPayloadTooLarge:       "warning",
			EventUploadBlocked:         "high",
			EventFileInclusion:         "critical",
			EventLDAPInjection:         "critical",
			EventXMLInjection:          "critical",
			EventJSONInjection:         "critical",
			EventXXEAttack:             "critical",
			EventHTTPParameterPollution: "medium",
			EventOpenRedirect:          "high",
			EventClickjackingAttempt:   "medium",
		},
	}
	go service.processEvents()
	go service.cleanupOldEvents()
	return service
}

func (s *SecurityAuditService) LogEvent(eventType SecurityEventType, r *http.Request, details map[string]interface{}) *SecurityEvent {
	ip := getClientIP(r)
	event := &SecurityEvent{
		ID:            generateEventID(),
		Timestamp:     time.Now(),
		EventType:     eventType,
		Severity:      s.severityLevels[eventType],
		SourceIP:      ip,
		UserAgent:     r.UserAgent(),
		RequestPath:   r.URL.Path,
		RequestMethod: r.Method,
		Details:       details,
		Status:        "new",
		GeoLocation:   s.getGeoIP(ip),
		ThreatScore:   s.calculateThreatScore(eventType, ip, r),
		Tags:          s.generateTags(eventType, r),
	}

	if s.asyncMode {
		s.eventBuffer <- event
	} else {
		s.storeEvent(event)
		s.checkAlerts(event)
	}

	s.updateIPEventCount(ip)
	s.checkThreatIntel(ip, event)

	return event
}

func (s *SecurityAuditService) calculateThreatScore(eventType SecurityEventType, ip string, r *http.Request) float64 {
	score := 0.0

	severityWeights := map[string]float64{
		"info":     5.0,
		"medium":   25.0,
		"warning":  50.0,
		"high":     75.0,
		"critical": 100.0,
	}

	if weight, exists := severityWeights[s.severityLevels[eventType]]; exists {
		score += weight
	}

	if intel, exists := s.threatIntel[ip]; exists {
		for _, entry := range intel {
			score += entry.Confidence * 10
		}
	}

	if s.ipEventCounts[ip] > 100 {
		score += 20.0
	}

	for _, pattern := range s.sensitivePatterns {
		if pattern.MatchString(r.URL.Path) || pattern.MatchString(r.URL.RawQuery) {
			score += 15.0
			break
		}
	}

	if score > s.maxThreatScore {
		score = s.maxThreatScore
	}

	return score
}

func (s *SecurityAuditService) generateTags(eventType SecurityEventType, r *http.Request) []string {
	tags := []string{string(eventType)}

	if strings.HasPrefix(r.URL.Path, "/admin") {
		tags = append(tags, "admin")
	}
	if strings.HasPrefix(r.URL.Path, "/api") {
		tags = append(tags, "api")
	}
	if strings.HasSuffix(r.URL.Path, "/login") {
		tags = append(tags, "authentication")
	}

	severity := s.severityLevels[eventType]
	if severity == "high" || severity == "critical" {
		tags = append(tags, "priority")
	}

	return tags
}

func (s *SecurityAuditService) getGeoIP(ip string) map[string]string {
	if geo, exists := s.geoIPCache[ip]; exists {
		return geo
	}

	geo := map[string]string{
		"country": "Unknown",
		"city":    "Unknown",
		"region":  "Unknown",
	}

	if strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168.") {
		geo["country"] = "Private"
	}

	s.geoIPCache[ip] = geo
	return geo
}

func (s *SecurityAuditService) updateIPEventCount(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ipEventCounts[ip]++
}

func (s *SecurityAuditService) checkThreatIntel(ip string, event *SecurityEvent) {
	if !s.enableThreatIntel {
		return
	}

	if intel, exists := s.threatIntel[ip]; exists {
		for _, entry := range intel {
			if entry.Confidence > 0.7 {
				event.RelatedEvents = append(event.RelatedEvents, entry.ThreatType)
				event.ThreatScore += entry.Confidence * 20
			}
		}
	}
}

func (s *SecurityAuditService) AddThreatIntel(ip, threatType, source, description string, confidence float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := &ThreatIntelEntry{
		IP:          ip,
		ThreatType:  threatType,
		Confidence:  confidence,
		Source:      source,
		LastSeen:    time.Now(),
		Description: description,
	}

	s.threatIntel[ip] = append(s.threatIntel[ip], entry)
}

func (s *SecurityAuditService) cleanupOldEvents() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-time.Duration(s.retentionDays) * 24 * time.Hour)
		newEvents := make([]*SecurityEvent, 0)
		for _, event := range s.events {
			if event.Timestamp.After(cutoff) {
				newEvents = append(newEvents, event)
			}
		}
		s.events = newEvents
		s.mu.Unlock()
	}
}

func (s *SecurityAuditService) processEvents() {
	for event := range s.eventBuffer {
		s.storeEvent(event)
		s.checkAlerts(event)
	}
}

func (s *SecurityAuditService) storeEvent(event *SecurityEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, event)

	if len(s.events) > s.maxEvents {
		s.events = s.events[len(s.events)-s.maxEvents:]
	}
}

func (s *SecurityAuditService) checkAlerts(event *SecurityEvent) {
	for _, handler := range s.alertHandlers {
		if s.asyncMode {
			go handler(event)
		} else {
			handler(event)
		}
	}
}

func (s *SecurityAuditService) RegisterAlertHandler(handler func(event *SecurityEvent)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertHandlers = append(s.alertHandlers, handler)
}

func (s *SecurityAuditService) GetRecentEvents(limit int) []*SecurityEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.events) {
		limit = len(s.events)
	}

	start := len(s.events) - limit
	return s.events[start:]
}

func (s *SecurityAuditService) GetEventsByType(eventType SecurityEventType, limit int) []*SecurityEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]*SecurityEvent, 0)
	for i := len(s.events) - 1; i >= 0 && len(filtered) < limit; i-- {
		if s.events[i].EventType == eventType {
			filtered = append(filtered, s.events[i])
		}
	}
	return filtered
}

func (s *SecurityAuditService) GetEventsByIP(ip string, limit int) []*SecurityEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]*SecurityEvent, 0)
	for i := len(s.events) - 1; i >= 0 && len(filtered) < limit; i-- {
		if s.events[i].SourceIP == ip {
			filtered = append(filtered, s.events[i])
		}
	}
	return filtered
}

func (s *SecurityAuditService) GetSecurityStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_events": len(s.events),
		"by_severity": map[string]int{
			"info":     0,
			"medium":   0,
			"warning":  0,
			"high":     0,
			"critical": 0,
		},
		"by_type": make(map[string]int),
	}

	for _, event := range s.events {
		stats["by_severity"].(map[string]int)[event.Severity]++
		stats["by_type"].(map[string]int)[string(event.EventType)]++
	}

	return stats
}

func (s *SecurityAuditService) DetectIntrusionAttempts(r *http.Request) []*SecurityEvent {
	events := make([]*SecurityEvent, 0)

	path := r.URL.Path
	query := r.URL.RawQuery

	suspiciousPatterns := map[string]SecurityEventType{
		`(\%27)|(\')|(\-\-)|(\%23)|(#)`:       EventSQLInjection,
		`(\%3C)|<((\%2F)|\/)*[a-z0-9\%]+`:     EventXSSAttempt,
		`(alert|script|javascript)`:           EventXSSAttempt,
		`(union|select|insert|update|delete)`: EventSQLInjection,
	}

	for pattern, eventType := range suspiciousPatterns {
		if matchPattern(path+query, pattern) {
			event := s.LogEvent(eventType, r, map[string]interface{}{
				"pattern": pattern,
				"matched": true,
			})
			events = append(events, event)
		}
	}

	return events
}

func (s *SecurityAuditService) ExportEvents(format string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch format {
	case "json":
		return json.Marshal(s.events)
	default:
		return json.Marshal(s.events)
	}
}

func generateEventID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

func matchPattern(input, pattern string) bool {
	matched, _ := regexp.MatchString(pattern, input)
	return matched
}
