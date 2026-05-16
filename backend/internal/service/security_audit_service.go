package service

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sync"
	"time"
)

type SecurityEventType string

const (
	EventLoginAttempt       SecurityEventType = "login_attempt"
	EventLoginSuccess       SecurityEventType = "login_success"
	EventLoginFailure       SecurityEventType = "login_failure"
	EventAccessDenied       SecurityEventType = "access_denied"
	EventCSRFDetected       SecurityEventType = "csrf_detected"
	EventSQLInjection       SecurityEventType = "sql_injection"
	EventXSSAttempt         SecurityEventType = "xss_attempt"
	EventRateLimitHit       SecurityEventType = "rate_limit_hit"
	EventBotDetected        SecurityEventType = "bot_detected"
	EventDDoSAttempt        SecurityEventType = "ddos_attempt"
	EventSuspiciousActivity SecurityEventType = "suspicious_activity"
	EventPrivilegeEscalation SecurityEventType = "privilege_escalation"
	EventDataAccess         SecurityEventType = "data_access"
	EventConfigChange       SecurityEventType = "config_change"
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
}

type SecurityAuditService struct {
	events        []*SecurityEvent
	eventBuffer   chan *SecurityEvent
	alertHandlers []func(event *SecurityEvent)
	mu            sync.RWMutex
	maxEvents     int
	severityLevels map[SecurityEventType]string
	asyncMode     bool // 控制是否使用异步模式，测试时可以设置为false
}

func NewSecurityAuditService() *SecurityAuditService {
	service := &SecurityAuditService{
		events:        make([]*SecurityEvent, 0),
		eventBuffer:   make(chan *SecurityEvent, 1000),
		alertHandlers: make([]func(event *SecurityEvent), 0),
		maxEvents:     10000,
		asyncMode:     true,
		severityLevels: map[SecurityEventType]string{
			EventLoginAttempt:       "info",
			EventLoginSuccess:       "info",
			EventLoginFailure:       "warning",
			EventAccessDenied:       "warning",
			EventCSRFDetected:       "high",
			EventSQLInjection:       "critical",
			EventXSSAttempt:         "high",
			EventRateLimitHit:       "warning",
			EventBotDetected:        "medium",
			EventDDoSAttempt:        "critical",
			EventSuspiciousActivity: "medium",
			EventPrivilegeEscalation: "critical",
			EventDataAccess:         "info",
			EventConfigChange:       "high",
		},
	}
	go service.processEvents()
	return service
}

func (s *SecurityAuditService) LogEvent(eventType SecurityEventType, r *http.Request, details map[string]interface{}) *SecurityEvent {
	event := &SecurityEvent{
		ID:            generateEventID(),
		Timestamp:     time.Now(),
		EventType:     eventType,
		Severity:      s.severityLevels[eventType],
		SourceIP:      getClientIP(r),
		UserAgent:     r.UserAgent(),
		RequestPath:   r.URL.Path,
		RequestMethod: r.Method,
		Details:       details,
		Status:        "new",
	}

	if s.asyncMode {
		s.eventBuffer <- event
	} else {
		s.storeEvent(event)
		s.checkAlerts(event)
	}
	return event
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
		`(\%27)|(\')|(\-\-)|(\%23)|(#)`: EventSQLInjection,
		`(\%3C)|<((\%2F)|\/)*[a-z0-9\%]+`: EventXSSAttempt,
		`(alert|script|javascript)`: EventXSSAttempt,
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
