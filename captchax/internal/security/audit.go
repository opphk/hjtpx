package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type AuditLevel string

func (l AuditLevel) String() string {
	return string(l)
}

const (
	AuditLevelDebug   AuditLevel = "debug"
	AuditLevelInfo    AuditLevel = "info"
	AuditLevelWarning AuditLevel = "warning"
	AuditLevelError   AuditLevel = "error"
	AuditLevelCritical AuditLevel = "critical"
)

type AuditAction string

const (
	ActionLogin              AuditAction = "login"
	ActionLogout             AuditAction = "logout"
	ActionLoginFailed        AuditAction = "login_failed"
	ActionConfigChange       AuditAction = "config_change"
	ActionWhitelistAdd       AuditAction = "whitelist_add"
	ActionWhitelistRemove    AuditAction = "whitelist_remove"
	ActionBlacklistAdd       AuditAction = "blacklist_add"
	ActionBlacklistRemove    AuditAction = "blacklist_remove"
	ActionCaptchaCreate      AuditAction = "captcha_create"
	ActionCaptchaVerify      AuditAction = "captcha_verify"
	ActionCaptchaVerifyFailed AuditAction = "captcha_verify_failed"
	ActionSensitiveDelete    AuditAction = "sensitive_delete"
	ActionKeyRotate          AuditAction = "key_rotate"
	ActionAdminCreate        AuditAction = "admin_create"
	ActionAdminDelete        AuditAction = "admin_delete"
	ActionRateLimitExceeded  AuditAction = "rate_limit_exceeded"
	ActionSuspiciousActivity AuditAction = "suspicious_activity"
	ActionDDoSDetected       AuditAction = "ddos_detected"
	ActionSQLInjection       AuditAction = "sqli_detected"
	ActionSignatureInvalid   AuditAction = "signature_invalid"
)

type AuditEvent struct {
	ID          string                 `json:"id"`
	Timestamp   string                 `json:"timestamp"`
	Level       AuditLevel             `json:"level"`
	Action      AuditAction            `json:"action"`
	UserID      string                 `json:"user_id,omitempty"`
	Username    string                 `json:"username,omitempty"`
	IPAddress   string                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Resource    string                 `json:"resource,omitempty"`
	ResourceID  string                 `json:"resource_id,omitempty"`
	Method      string                 `json:"method,omitempty"`
	Path        string                 `json:"path,omitempty"`
	StatusCode  int                    `json:"status_code,omitempty"`
	Success     bool                   `json:"success"`
	Message     string                 `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	RiskScore   float64                `json:"risk_score,omitempty"`
	ExtraData   map[string]string      `json:"extra_data,omitempty"`
}

type AuditLogger interface {
	Log(event *AuditEvent)
	LogAsync(event *AuditEvent)
	Query(filter *AuditFilter) ([]*AuditEvent, error)
	GetStats() (*AuditStats, error)
}

type AuditFilter struct {
	StartTime   time.Time
	EndTime     time.Time
	Level       AuditLevel
	Action      AuditAction
	UserID      string
	IPAddress   string
	Success     *bool
	Limit       int
	Offset      int
}

type AuditStats struct {
	TotalEvents      int64            `json:"total_events"`
	ByLevel          map[string]int64 `json:"by_level"`
	ByAction         map[string]int64 `json:"by_action"`
	FailedAttempts   int64            `json:"failed_attempts"`
	SuspiciousEvents int64            `json:"suspicious_events"`
	TopIPAddresses   map[string]int64 `json:"top_ip_addresses"`
	TopUsers         map[string]int64 `json:"top_users"`
}

type AuditConfig struct {
	Enabled           bool
	RetentionDays     int
	BufferSize        int
	FlushInterval     time.Duration
	AsyncEnabled      bool
	IncludeBody       bool
	IncludeHeaders    bool
	MaskSensitiveData bool
	SensitiveFields   []string
}

var defaultAuditConfig = &AuditConfig{
	Enabled:           true,
	RetentionDays:     90,
	BufferSize:        1000,
	FlushInterval:     5 * time.Second,
	AsyncEnabled:      true,
	IncludeBody:       false,
	IncludeHeaders:    false,
	MaskSensitiveData: true,
	SensitiveFields:   []string{"password", "token", "secret", "key", "authorization", "cookie"},
}

type AuditLoggerV2 struct {
	config    *AuditConfig
	buffer    chan *AuditEvent
	events    []*AuditEvent
	mu        sync.RWMutex
	maxEvents int
	stats     *AuditStats
	statsMu   sync.RWMutex
}

func NewAuditLogger(config *AuditConfig) *AuditLoggerV2 {
	if config == nil {
		config = defaultAuditConfig
	}

	logger := &AuditLoggerV2{
		config:    config,
		buffer:    make(chan *AuditEvent, config.BufferSize),
		events:    make([]*AuditEvent, 0, 10000),
		maxEvents: 100000,
		stats: &AuditStats{
			ByLevel:        make(map[string]int64),
			ByAction:       make(map[string]int64),
			TopIPAddresses: make(map[string]int64),
			TopUsers:       make(map[string]int64),
		},
	}

	if config.AsyncEnabled {
		go logger.processBuffer()
	}

	return logger
}

func (l *AuditLoggerV2) processBuffer() {
	ticker := time.NewTicker(l.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case event := <-l.buffer:
			l.appendEvent(event)
		case <-ticker.C:
			l.flush()
		}
	}
}

func (l *AuditLoggerV2) appendEvent(event *AuditEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	l.events = append(l.events, event)

	if len(l.events) > l.maxEvents {
		l.events = l.events[len(l.events)-l.maxEvents:]
	}

	l.updateStats(event)
}

func (l *AuditLoggerV2) flush() {
	l.mu.Lock()
	defer l.mu.Unlock()
}

func (l *AuditLoggerV2) updateStats(event *AuditEvent) {
	l.statsMu.Lock()
	defer l.statsMu.Unlock()

	l.stats.TotalEvents++
	l.stats.ByLevel[string(event.Level)]++
	l.stats.ByAction[string(event.Action)]++

	if event.IPAddress != "" {
		l.stats.TopIPAddresses[event.IPAddress]++
	}
	if event.UserID != "" {
		l.stats.TopUsers[event.UserID]++
	}

	if !event.Success || event.Level == AuditLevelWarning || event.Level == AuditLevelError || event.Level == AuditLevelCritical {
		l.stats.SuspiciousEvents++
	}
}

func (l *AuditLoggerV2) Log(event *AuditEvent) {
	if !l.config.Enabled {
		return
	}

	event.ID = generateEventID()
	event.Timestamp = time.Now().UTC().Format(time.RFC3339)

	if l.config.MaskSensitiveData {
		l.maskSensitiveData(event)
	}

	if l.config.AsyncEnabled {
		select {
		case l.buffer <- event:
		default:
			l.appendEvent(event)
		}
	} else {
		l.appendEvent(event)
	}
}

func (l *AuditLoggerV2) LogAsync(event *AuditEvent) {
	l.Log(event)
}

func (l *AuditLoggerV2) maskSensitiveData(event *AuditEvent) {
	if event.Details == nil {
		return
	}

	for _, field := range l.config.SensitiveFields {
		if val, exists := event.Details[field]; exists {
			event.Details[field] = maskValue(fmt.Sprintf("%v", val))
		}
	}

	if event.ExtraData != nil {
		for _, field := range l.config.SensitiveFields {
			if val, exists := event.ExtraData[field]; exists {
				event.ExtraData[field] = maskValue(val)
			}
		}
	}
}

func maskValue(value string) string {
	if len(value) <= 4 {
		return "***"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

func generateEventID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (l *AuditLoggerV2) Query(filter *AuditFilter) ([]*AuditEvent, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []*AuditEvent
	for _, event := range l.events {
		if !l.matchesFilter(event, filter) {
			continue
		}
		results = append(results, event)
	}

	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}
	if filter.Offset > 0 && filter.Offset < len(results) {
		results = results[filter.Offset:]
	}

	return results, nil
}

func (l *AuditLoggerV2) matchesFilter(event *AuditEvent, filter *AuditFilter) bool {
	if filter.StartTime.Unix() > 0 {
		t, _ := time.Parse(time.RFC3339, event.Timestamp)
		if t.Before(filter.StartTime) {
			return false
		}
	}
	if filter.EndTime.Unix() > 0 {
		t, _ := time.Parse(time.RFC3339, event.Timestamp)
		if t.After(filter.EndTime) {
			return false
		}
	}
	if filter.Level != "" && event.Level != filter.Level {
		return false
	}
	if filter.Action != "" && event.Action != filter.Action {
		return false
	}
	if filter.UserID != "" && event.UserID != filter.UserID {
		return false
	}
	if filter.IPAddress != "" && event.IPAddress != filter.IPAddress {
		return false
	}
	if filter.Success != nil && event.Success != *filter.Success {
		return false
	}
	return true
}

func (l *AuditLoggerV2) GetStats() (*AuditStats, error) {
	l.statsMu.RLock()
	defer l.statsMu.RUnlock()

	statsCopy := &AuditStats{
		TotalEvents:      l.stats.TotalEvents,
		ByLevel:          make(map[string]int64),
		ByAction:         make(map[string]int64),
		FailedAttempts:   l.stats.FailedAttempts,
		SuspiciousEvents: l.stats.SuspiciousEvents,
		TopIPAddresses:   make(map[string]int64),
		TopUsers:         make(map[string]int64),
	}

	for k, v := range l.stats.ByLevel {
		statsCopy.ByLevel[k] = v
	}
	for k, v := range l.stats.ByAction {
		statsCopy.ByAction[k] = v
	}
	for k, v := range l.stats.TopIPAddresses {
		statsCopy.TopIPAddresses[k] = v
	}
	for k, v := range l.stats.TopUsers {
		statsCopy.TopUsers[k] = v
	}

	return statsCopy, nil
}

type AuditMiddleware struct {
	logger  *AuditLoggerV2
	config  *AuditConfig
	exempts map[string]bool
}

func NewAuditMiddleware(config *AuditConfig) *AuditMiddleware {
	if config == nil {
		config = defaultAuditConfig
	}
	return &AuditMiddleware{
		logger:  NewAuditLogger(config),
		config:  config,
		exempts: make(map[string]bool),
	}
}

func (m *AuditMiddleware) AddExempt(path string) {
	m.exempts[path] = true
}

func (m *AuditMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.exempts[c.Request.URL.Path] {
			c.Next()
			return
		}

		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateEventID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		start := time.Now()

		c.Next()

		duration := time.Since(start)

		action := m.detectAction(c)
		level := m.determineLevel(c, action)

		event := &AuditEvent{
			Level:      level,
			Action:     action,
			IPAddress:  getClientIP(c),
			UserAgent:  c.GetHeader("User-Agent"),
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			StatusCode: c.Writer.Status(),
			Success:    c.Writer.Status() < 400,
			Message:    fmt.Sprintf("%s %s - %d (%v)", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration),
			RequestID:  requestID,
			Details: map[string]interface{}{
				"duration_ms": duration.Milliseconds(),
				"query":       c.Request.URL.RawQuery,
			},
		}

		if userID, exists := c.Get("user_id"); exists {
			event.UserID = fmt.Sprintf("%v", userID)
		}
		if username, exists := c.Get("username"); exists {
			event.Username = fmt.Sprintf("%v", username)
		}
		if sessionID, exists := c.Get("session_id"); exists {
			event.SessionID = fmt.Sprintf("%v", sessionID)
		}

		if m.config.IncludeBody && c.Request.Body != nil {
			body, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(strings.NewReader(string(body)))
			if len(body) > 0 && len(body) < 1024 {
				event.Details["body"] = string(body)
			}
		}

		if m.config.IncludeHeaders {
			headers := make(map[string]string)
			for _, h := range []string{"Authorization", "Content-Type", "Origin", "Referer"} {
				if v := c.GetHeader(h); v != "" {
					headers[h] = v
				}
			}
			event.Details["headers"] = headers
		}

		m.logger.Log(event)
	}
}

func (m *AuditMiddleware) detectAction(c *gin.Context) AuditAction {
	path := c.Request.URL.Path
	method := c.Request.Method

	switch {
	case strings.HasSuffix(path, "/login") && method == http.MethodPost:
		if c.Writer.Status() >= 400 {
			return ActionLoginFailed
		}
		return ActionLogin
	case strings.HasSuffix(path, "/logout") && method == http.MethodPost:
		return ActionLogout
	case strings.HasSuffix(path, "/config") && method == http.MethodPost:
		return ActionConfigChange
	case strings.HasSuffix(path, "/whitelist") && method == http.MethodPost:
		return ActionWhitelistAdd
	case strings.HasSuffix(path, "/whitelist") && method == http.MethodDelete:
		return ActionWhitelistRemove
	case strings.HasSuffix(path, "/blacklist") && method == http.MethodPost:
		return ActionBlacklistAdd
	case strings.HasSuffix(path, "/blacklist") && method == http.MethodDelete:
		return ActionBlacklistRemove
	case strings.Contains(path, "/captcha/") && method == http.MethodPost:
		if strings.HasSuffix(path, "/verify") {
			if c.Writer.Status() >= 400 {
				return ActionCaptchaVerifyFailed
			}
			return ActionCaptchaVerify
		}
		return ActionCaptchaCreate
	case strings.Contains(path, "/delete") || strings.Contains(path, "/remove"):
		return ActionSensitiveDelete
	default:
		return AuditAction("request:" + method + ":" + path)
	}
}

func (m *AuditMiddleware) determineLevel(c *gin.Context, action AuditAction) AuditLevel {
	status := c.Writer.Status()

	switch {
	case status >= 500:
		return AuditLevelError
	case status >= 400:
		switch action {
		case ActionLoginFailed:
			return AuditLevelWarning
		case ActionSQLInjection:
			return AuditLevelCritical
		default:
			return AuditLevelInfo
		}
	case action == ActionKeyRotate || action == ActionAdminCreate || action == ActionAdminDelete:
		return AuditLevelWarning
	default:
		return AuditLevelInfo
	}
}

func getClientIP(c *gin.Context) string {
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return xri
	}
	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	if ip != "" {
		return ip
	}
	return c.ClientIP()
}

func LogSecurityEvent(logger *AuditLoggerV2, level AuditLevel, action AuditAction, message string, details map[string]interface{}) {
	event := &AuditEvent{
		Level:   level,
		Action:  action,
		Success: level != AuditLevelError && level != AuditLevelCritical,
		Message: message,
		Details: details,
	}
	logger.Log(event)
}

func LogLoginAttempt(logger *AuditLoggerV2, username, ip, userAgent string, success bool, reason string) {
	action := ActionLogin
	if !success {
		action = ActionLoginFailed
	}
	LogSecurityEvent(logger, AuditLevelInfo, action, reason, map[string]interface{}{
		"username":   username,
		"ip_address": ip,
		"user_agent": userAgent,
		"success":    success,
	})
}

func LogConfigChange(logger *AuditLoggerV2, userID, username, ip string, changes map[string]interface{}) {
	LogSecurityEvent(logger, AuditLevelWarning, ActionConfigChange, "configuration changed", map[string]interface{}{
		"user_id": userID,
		"username": username,
		"ip_address": ip,
		"changes": changes,
	})
}

func LogKeyRotation(logger *AuditLoggerV2, triggeredBy string) {
	LogSecurityEvent(logger, AuditLevelWarning, ActionKeyRotate, "signing key rotated", map[string]interface{}{
		"triggered_by": triggeredBy,
	})
}

func LogSuspiciousActivity(logger *AuditLoggerV2, ip, activity string, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["ip_address"] = ip
	LogSecurityEvent(logger, AuditLevelWarning, ActionSuspiciousActivity, activity, details)
}

var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(union\s+select|select\s+\*\s+from)`),
	regexp.MustCompile(`(?i)(drop\s+table|delete\s+from)`),
	regexp.MustCompile(`(?i)(exec\s*\(|execute\s*\()`),
	regexp.MustCompile(`--`),
	regexp.MustCompile(`/\*.*\*/`),
	regexp.MustCompile(`(?i)(or\s+1\s*=\s*1|and\s+1\s*=\s*1)`),
}

func DetectSensitivePattern(input string) (bool, string) {
	for _, pattern := range sensitivePatterns {
		if pattern.MatchString(input) {
			return true, pattern.String()
		}
	}
	return false, ""
}

func LogSecurityViolation(logger *AuditLoggerV2, violationType string, ip string, details map[string]interface{}) {
	var action AuditAction
	var level AuditLevel

	switch violationType {
	case "sqli":
		action = ActionSQLInjection
		level = AuditLevelCritical
	case "ddos":
		action = ActionDDoSDetected
		level = AuditLevelCritical
	case "rate_limit":
		action = ActionRateLimitExceeded
		level = AuditLevelWarning
	case "invalid_signature":
		action = ActionSignatureInvalid
		level = AuditLevelWarning
	default:
		action = ActionSuspiciousActivity
		level = AuditLevelWarning
	}

	LogSecurityEvent(logger, level, action, fmt.Sprintf("security violation: %s", violationType), details)
}

type AuditContext struct {
	UserID    string
	Username  string
	IPAddress string
	UserAgent string
	RequestID string
	SessionID string
}

func (ctx *AuditContext) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"user_id":    ctx.UserID,
		"username":   ctx.Username,
		"ip_address": ctx.IPAddress,
		"user_agent": ctx.UserAgent,
		"request_id": ctx.RequestID,
		"session_id": ctx.SessionID,
	}
}

func ContextFromGin(c *gin.Context) *AuditContext {
	ctx := &AuditContext{
		IPAddress: getClientIP(c),
		UserAgent: c.GetHeader("User-Agent"),
	}

	if v, exists := c.Get("user_id"); exists {
		ctx.UserID = fmt.Sprintf("%v", v)
	}
	if v, exists := c.Get("username"); exists {
		ctx.Username = fmt.Sprintf("%v", v)
	}
	if v, exists := c.Get("request_id"); exists {
		ctx.RequestID = fmt.Sprintf("%v", v)
	}
	if v, exists := c.Get("session_id"); exists {
		ctx.SessionID = fmt.Sprintf("%v", v)
	}

	return ctx
}
