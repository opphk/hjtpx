package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type SecurityEventType string

const (
	EventLoginSuccess       SecurityEventType = "login_success"
	EventLoginFailed        SecurityEventType = "login_failed"
	EventLogout             SecurityEventType = "logout"
	EventPasswordChanged    SecurityEventType = "password_changed"
	EventCSRFViolation      SecurityEventType = "csrf_violation"
	EventXSSAttempt         SecurityEventType = "xss_attempt"
	EventSignatureFailed    SecurityEventType = "signature_failed"
	EventRateLimitExceeded  SecurityEventType = "rate_limit_exceeded"
	EventSuspiciousActivity SecurityEventType = "suspicious_activity"
	EventUnauthorizedAccess SecurityEventType = "unauthorized_access"
	EventPermissionDenied   SecurityEventType = "permission_denied"
	EventBruteForce         SecurityEventType = "brute_force"
	EventSessionHijack      SecurityEventType = "session_hijack"
	EventDataTampering      SecurityEventType = "data_tampering"
	EventInjectionAttempt   SecurityEventType = "injection_attempt"
	EventPathTraversal      SecurityEventType = "path_traversal"
	EventSensitiveExposure  SecurityEventType = "sensitive_exposure"
)

type SecurityLevel string

const (
	LevelLow      SecurityLevel = "low"
	LevelMedium   SecurityLevel = "medium"
	LevelHigh     SecurityLevel = "high"
	LevelCritical SecurityLevel = "critical"
)

type SecurityEvent struct {
	EventID      string           `json:"event_id"`
	Timestamp    time.Time        `json:"timestamp"`
	EventType    SecurityEventType `json:"event_type"`
	Level        SecurityLevel    `json:"level"`
	ClientIP     string           `json:"client_ip"`
	UserID       string           `json:"user_id,omitempty"`
	Username     string           `json:"username,omitempty"`
	Path         string           `json:"path"`
	Method       string           `json:"method"`
	UserAgent    string           `json:"user_agent"`
	SessionID    string           `json:"session_id,omitempty"`
	Description  string           `json:"description"`
	Details      string           `json:"details,omitempty"`
	RequestID    string           `json:"request_id,omitempty"`
	Duration     int64            `json:"duration_ms,omitempty"`
	StatusCode   int              `json:"status_code,omitempty"`
	IsBlocked    bool             `json:"is_blocked"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

type SecurityLogConfig struct {
	EnableConsole    bool
	EnableFile       bool
	EnableRedis      bool
	LogFilePath      string
	MaxFileSize      int64
	MaxBackupFiles   int
	RotateInterval   time.Duration
	MinLevel         SecurityLevel
	ExcludePaths     []string
	ExcludeEventTypes []SecurityEventType
	RetentionDays    int
}

var defaultSecurityLogConfig = SecurityLogConfig{
	EnableConsole:    true,
	EnableFile:       true,
	EnableRedis:      true,
	LogFilePath:      "./logs/security.log",
	MaxFileSize:      10 * 1024 * 1024,
	MaxBackupFiles:   30,
	RotateInterval:   24 * time.Hour,
	MinLevel:        LevelLow,
	ExcludePaths:    []string{"/health", "/api/health", "/metrics", "/api/metrics"},
	ExcludeEventTypes: []SecurityEventType{},
	RetentionDays:   90,
}

var (
	securityLogInstance *SecurityLog
	securityLogOnce     sync.Once
	securityLogMutex    sync.Mutex
)

type SecurityLog struct {
	config   SecurityLogConfig
	file     *os.File
	fileMu   sync.Mutex
	eventCh  chan SecurityEvent
	wg       sync.WaitGroup
	stopCh   chan struct{}
}

func getSecurityLevel(eventType SecurityEventType) SecurityLevel {
	levelMap := map[SecurityEventType]SecurityLevel{
		EventLoginSuccess:       LevelLow,
		EventLogout:             LevelLow,
		EventPasswordChanged:    LevelMedium,
		EventLoginFailed:        LevelMedium,
		EventCSRFViolation:      LevelHigh,
		EventXSSAttempt:         LevelHigh,
		EventSignatureFailed:    LevelHigh,
		EventSuspiciousActivity: LevelHigh,
		EventUnauthorizedAccess: LevelHigh,
		EventRateLimitExceeded:  LevelMedium,
		EventPermissionDenied:   LevelMedium,
		EventBruteForce:         LevelCritical,
		EventSessionHijack:      LevelCritical,
		EventDataTampering:      LevelCritical,
		EventInjectionAttempt:   LevelCritical,
		EventPathTraversal:      LevelHigh,
		EventSensitiveExposure:  LevelCritical,
	}

	if level, ok := levelMap[eventType]; ok {
		return level
	}
	return LevelMedium
}

func getEventDescription(eventType SecurityEventType) string {
	descriptions := map[SecurityEventType]string{
		EventLoginSuccess:       "User login successful",
		EventLoginFailed:        "User login failed",
		EventLogout:             "User logout",
		EventPasswordChanged:    "User password changed",
		EventCSRFViolation:      "CSRF token validation failed",
		EventXSSAttempt:         "XSS attack attempt detected",
		EventSignatureFailed:    "API signature verification failed",
		EventRateLimitExceeded:  "Rate limit exceeded",
		EventSuspiciousActivity: "Suspicious activity detected",
		EventUnauthorizedAccess: "Unauthorized access attempt",
		EventPermissionDenied:   "Permission denied",
		EventBruteForce:         "Brute force attack detected",
		EventSessionHijack:      "Session hijack attempt",
		EventDataTampering:      "Data tampering detected",
		EventInjectionAttempt:   "Injection attack attempt",
		EventPathTraversal:      "Path traversal attempt",
		EventSensitiveExposure: "Sensitive data exposure",
	}

	if desc, ok := descriptions[eventType]; ok {
		return desc
	}
	return "Unknown security event"
}

func GetSecurityLog(config ...SecurityLogConfig) *SecurityLog {
	securityLogOnce.Do(func() {
		cfg := defaultSecurityLogConfig
		if len(config) > 0 {
			cfg = config[0]
		}
		securityLogInstance = newSecurityLog(cfg)
	})
	return securityLogInstance
}

func newSecurityLog(config SecurityLogConfig) *SecurityLog {
	log := &SecurityLog{
		config:  config,
		eventCh: make(chan SecurityEvent, 1000),
		stopCh:  make(chan struct{}),
	}

	if config.EnableFile {
		log.initFile()
	}

	log.wg.Add(1)
	go log.processEvents()

	log.wg.Add(1)
	go log.rotateLogFile()

	return log
}

func (l *SecurityLog) initFile() {
	l.fileMu.Lock()
	defer l.fileMu.Unlock()

	dir := filepath.Dir(l.config.LogFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("[SecurityLog] Failed to create log directory: %v\n", err)
		return
	}

	file, err := os.OpenFile(l.config.LogFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("[SecurityLog] Failed to open log file: %v\n", err)
		return
	}
	l.file = file
}

func (l *SecurityLog) processEvents() {
	defer l.wg.Done()

	for {
		select {
		case event := <-l.eventCh:
			l.writeEvent(event)
		case <-l.stopCh:
			l.drainEvents()
			return
		}
	}
}

func (l *SecurityLog) drainEvents() {
	for {
		select {
		case event := <-l.eventCh:
			l.writeEvent(event)
		default:
			return
		}
	}
}

func (l *SecurityLog) rotateLogFile() {
	defer l.wg.Done()

	ticker := time.NewTicker(l.config.RotateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.rotate()
		case <-l.stopCh:
			return
		}
	}
}

func (l *SecurityLog) rotate() {
	l.fileMu.Lock()
	defer l.fileMu.Unlock()

	if l.file == nil {
		return
	}

	l.file.Close()

	timestamp := time.Now().Format("2006-01-02_150405")
	rotatedPath := fmt.Sprintf("%s.%s", l.config.LogFilePath, timestamp)

	if err := os.Rename(l.config.LogFilePath, rotatedPath); err != nil {
		fmt.Printf("[SecurityLog] Failed to rotate log file: %v\n", err)
	}

	l.cleanOldFiles()

	file, err := os.OpenFile(l.config.LogFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("[SecurityLog] Failed to open new log file: %v\n", err)
		return
	}
	l.file = file
}

func (l *SecurityLog) cleanOldFiles() {
	dir := filepath.Dir(l.config.LogFilePath)
	base := filepath.Base(l.config.LogFilePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var logFiles []os.FileInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil && filepath.Base(info.Name()) != base {
				logFiles = append(logFiles, info)
			}
		}
	}

	if len(logFiles) > l.config.MaxBackupFiles {
		cutoff := len(logFiles) - l.config.MaxBackupFiles
		for i := 0; i < cutoff; i++ {
			os.Remove(filepath.Join(dir, logFiles[i].Name()))
		}
	}
}

func (l *SecurityLog) writeEvent(event SecurityEvent) {
	if l.config.EnableConsole {
		l.writeConsole(event)
	}

	if l.config.EnableFile && l.file != nil {
		l.writeFile(event)
	}

	if l.config.EnableRedis && redis.Client != nil {
		l.writeRedis(event)
	}
}

func (l *SecurityLog) writeConsole(event SecurityEvent) {
	levelStr := string(event.Level)
	colorCode := ""

	switch event.Level {
	case LevelCritical:
		colorCode = "\033[31m"
	case LevelHigh:
		colorCode = "\033[33m"
	case LevelMedium:
		colorCode = "\033[34m"
	case LevelLow:
		colorCode = "\033[32m"
	}

	resetCode := "\033[0m"
	timestamp := event.Timestamp.Format("2006/01/02 15:04:05")

	message := fmt.Sprintf("%s[SECURITY %s] %s | %s | %s | %s | %s | %s%s",
		colorCode,
		levelStr,
		timestamp,
		event.EventType,
		event.ClientIP,
		event.Path,
		event.Description,
		resetCode,
	)

	if event.Username != "" {
		message += fmt.Sprintf(" | User: %s", event.Username)
	}

	if event.Details != "" {
		message += fmt.Sprintf(" | Details: %s", event.Details)
	}

	fmt.Println(message)
}

func (l *SecurityLog) writeFile(event SecurityEvent) {
	l.fileMu.Lock()
	defer l.fileMu.Unlock()

	if l.file == nil {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("[SecurityLog] Failed to marshal event: %v\n", err)
		return
	}

	l.file.Write(data)
	l.file.WriteString("\n")

	info, _ := l.file.Stat()
	if info.Size() >= l.config.MaxFileSize {
		go l.rotate()
	}
}

func (l *SecurityLog) writeRedis(event SecurityEvent) {
	if redis.Client == nil {
		return
	}

	ctx := context.Background()

	key := fmt.Sprintf("security:event:%s", event.EventID)
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	redis.Client.Set(ctx, key, data, time.Duration(l.config.RetentionDays)*24*time.Hour)

	redis.Client.Set(ctx, key+"_ts", event.Timestamp.Unix(), time.Duration(l.config.RetentionDays)*24*time.Hour)
	redis.Client.SAdd(ctx, "security:events:by_time", event.EventID).Result()

	redis.Client.Incr(ctx, fmt.Sprintf("security:count:type:%s", event.EventType))
	redis.Client.Incr(ctx, fmt.Sprintf("security:count:level:%s", event.Level))
	redis.Client.Incr(ctx, "security:count:total")
}

func (l *SecurityLog) Log(event SecurityEvent) {
	event.Timestamp = time.Now()
	event.EventID = fmt.Sprintf("%d-%s", event.Timestamp.UnixNano(), generateRandomID(8))

	if event.Level == "" {
		event.Level = getSecurityLevel(event.EventType)
	}

	if event.Description == "" {
		event.Description = getEventDescription(event.EventType)
	}

	l.eventCh <- event
}

func (l *SecurityLog) LogLoginSuccess(c *gin.Context, userID uint, username string) {
	event := SecurityEvent{
		EventType:   EventLoginSuccess,
		Level:       LevelLow,
		ClientIP:    c.ClientIP(),
		UserID:      fmt.Sprintf("%d", userID),
		Username:    username,
		Path:        c.Request.URL.Path,
		Method:      c.Request.Method,
		UserAgent:   c.GetHeader("User-Agent"),
		Description: "User login successful",
		IsBlocked:   false,
	}
	l.Log(event)
}

func (l *SecurityLog) LogLoginFailed(c *gin.Context, username, reason string) {
	event := SecurityEvent{
		EventType:   EventLoginFailed,
		Level:       LevelMedium,
		ClientIP:    c.ClientIP(),
		Username:    username,
		Path:        c.Request.URL.Path,
		Method:      c.Request.Method,
		UserAgent:   c.GetHeader("User-Agent"),
		Description: "User login failed",
		Details:     reason,
		IsBlocked:   false,
	}
	l.Log(event)
}

func (l *SecurityLog) LogCSRFViolation(c *gin.Context) {
	event := SecurityEvent{
		EventType:   EventCSRFViolation,
		Level:       LevelHigh,
		ClientIP:    c.ClientIP(),
		Path:        c.Request.URL.Path,
		Method:      c.Request.Method,
		UserAgent:   c.GetHeader("User-Agent"),
		Description: "CSRF token validation failed",
		IsBlocked:   true,
	}
	l.Log(event)
}

func (l *SecurityLog) LogXSSAttempt(c *gin.Context, field, value string) {
	event := SecurityEvent{
		EventType:   EventXSSAttempt,
		Level:       LevelHigh,
		ClientIP:    c.ClientIP(),
		Path:        c.Request.URL.Path,
		Method:      c.Request.Method,
		UserAgent:   c.GetHeader("User-Agent"),
		Description: "XSS attack attempt detected",
		Details:     fmt.Sprintf("Field: %s", field),
		Extra: map[string]interface{}{
			"field": field,
			"sanitized_value": SanitizeString(value),
		},
		IsBlocked: true,
	}
	l.Log(event)
}

func (l *SecurityLog) LogSignatureFailure(c *gin.Context) {
	event := SecurityEvent{
		EventType:   EventSignatureFailed,
		Level:       LevelHigh,
		ClientIP:    c.ClientIP(),
		Path:        c.Request.URL.Path,
		Method:      c.Request.Method,
		UserAgent:   c.GetHeader("User-Agent"),
		Description: "API signature verification failed",
		IsBlocked:   true,
	}
	l.Log(event)
}

func (l *SecurityLog) LogSuspiciousActivity(c *gin.Context, reason string) {
	event := SecurityEvent{
		EventType:   EventSuspiciousActivity,
		Level:       LevelHigh,
		ClientIP:    c.ClientIP(),
		Path:        c.Request.URL.Path,
		Method:      c.Request.Method,
		UserAgent:   c.GetHeader("User-Agent"),
		Description: "Suspicious activity detected",
		Details:     reason,
		IsBlocked:   true,
	}
	l.Log(event)
}

func (l *SecurityLog) LogUnauthorizedAccess(c *gin.Context) {
	event := SecurityEvent{
		EventType:   EventUnauthorizedAccess,
		Level:       LevelHigh,
		ClientIP:    c.ClientIP(),
		Path:        c.Request.URL.Path,
		Method:      c.Request.Method,
		UserAgent:   c.GetHeader("User-Agent"),
		Description: "Unauthorized access attempt",
		IsBlocked:   true,
	}
	l.Log(event)
}

func (l *SecurityLog) LogBruteForce(c *gin.Context, attemptCount int) {
	event := SecurityEvent{
		EventType:   EventBruteForce,
		Level:       LevelCritical,
		ClientIP:    c.ClientIP(),
		Path:        c.Request.URL.Path,
		Method:      c.Request.Method,
		UserAgent:   c.GetHeader("User-Agent"),
		Description: "Brute force attack detected",
		Details:     fmt.Sprintf("Attempt count: %d", attemptCount),
		IsBlocked:   true,
	}
	l.Log(event)
}

func (l *SecurityLog) LogInjectionAttempt(c *gin.Context, injectionType string) {
	event := SecurityEvent{
		EventType:   EventInjectionAttempt,
		Level:       LevelCritical,
		ClientIP:    c.ClientIP(),
		Path:        c.Request.URL.Path,
		Method:      c.Request.Method,
		UserAgent:   c.GetHeader("User-Agent"),
		Description: "Injection attack attempt detected",
		Details:     fmt.Sprintf("Type: %s", injectionType),
		IsBlocked:   true,
	}
	l.Log(event)
}

func (l *SecurityLog) Close() {
	close(l.stopCh)
	l.wg.Wait()

	l.fileMu.Lock()
	defer l.fileMu.Unlock()

	if l.file != nil {
		l.file.Close()
	}
}

func SecurityEventLogger() gin.HandlerFunc {
	log := GetSecurityLog()

	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Milliseconds()
		statusCode := c.Writer.Status()

		if statusCode >= 400 && statusCode < 500 {
			if statusCode == 403 {
				event := SecurityEvent{
					EventType:  EventCSRFViolation,
					Level:      LevelHigh,
					ClientIP:   c.ClientIP(),
					Path:       c.Request.URL.Path,
					Method:     c.Request.Method,
					UserAgent:  c.GetHeader("User-Agent"),
					Duration:   duration,
					StatusCode: statusCode,
					IsBlocked:  true,
				}
				log.Log(event)
			}
		}
	}
}

func generateRandomID(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(time.Nanosecond)
	}
	return string(result)
}

func GetSecurityStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	stats["total_events"] = 0
	stats["by_level"] = map[string]int{
		"low":      0,
		"medium":  0,
		"high":    0,
		"critical": 0,
	}
	stats["by_type"] = map[string]int{}
	stats["recent_24h"] = 0
	stats["blocked_count"] = 0

	if redis.Client != nil {
		ctx := context.Background()

		total, err := redis.Client.Get(ctx, "security:count:total").Int()
		if err == nil {
			stats["total_events"] = total
		}

		for _, level := range []string{"low", "medium", "high", "critical"} {
			count, err := redis.Client.Get(ctx, fmt.Sprintf("security:count:level:%s", level)).Int()
			if err == nil {
				stats["by_level"].(map[string]int)[level] = count
			}
		}

		cutoff := time.Now().Add(-24 * time.Hour).Unix()
		allEvents, err := redis.Client.SMembers(ctx, "security:events:by_time").Result()
		if err == nil {
			count := 0
			for _, eventID := range allEvents {
				eventKey := fmt.Sprintf("security:event:%s", eventID)
				ts, err := redis.Client.Get(ctx, eventKey+"_ts").Int64()
				if err == nil && ts >= cutoff {
					count++
				}
			}
			stats["recent_24h"] = count
		}
	}

	return stats, nil
}

func LogSecurityEvent(event SecurityEvent) {
	log := GetSecurityLog()
	log.Log(event)
}

func LogSecurityEventAsync(event SecurityEvent) {
	go func() {
		log := GetSecurityLog()
		log.Log(event)
	}()
}

func SecurityEventHandler() gin.HandlerFunc {
	log := GetSecurityLog()

	return func(c *gin.Context) {
		c.Next()

		statusCode := c.Writer.Status()

		if statusCode >= 400 {
			eventType := EventUnauthorizedAccess
			level := LevelMedium

			switch statusCode {
			case 401:
				eventType = EventUnauthorizedAccess
				level = LevelHigh
			case 403:
				eventType = EventCSRFViolation
				level = LevelHigh
			case 429:
				eventType = EventRateLimitExceeded
				level = LevelMedium
			}

			event := SecurityEvent{
				EventType:   eventType,
				Level:       level,
				ClientIP:    c.ClientIP(),
				Path:        c.Request.URL.Path,
				Method:      c.Request.Method,
				UserAgent:   c.GetHeader("User-Agent"),
				StatusCode:  statusCode,
				IsBlocked:   true,
			}

			log.Log(event)
		}
	}
}
