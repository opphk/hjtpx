package security

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SecurityLogLevel 安全日志级别
type SecurityLogLevel string

const (
	LogLevelInfo     SecurityLogLevel = "info"
	LogLevelWarning SecurityLogLevel = "warning"
	LogLevelCritical SecurityLogLevel = "critical"
)

// SecurityLog 安全日志
type SecurityLog struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       SecurityLogLevel      `json:"level"`
	Event       string                `json:"event"`
	IPAddress   string                `json:"ip_address"`
	UserID      uint                  `json:"user_id,omitempty"`
	UserAgent   string                `json:"user_agent,omitempty"`
	RequestPath string                `json:"request_path,omitempty"`
	Method      string                `json:"method,omitempty"`
	StatusCode  int                   `json:"status_code,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	RiskScore   float64               `json:"risk_score,omitempty"`
}

// SecurityEventType 安全事件类型
type SecurityEventType string

const (
	EventLoginSuccess        SecurityEventType = "login_success"
	EventLoginFailure        SecurityEventType = "login_failure"
	EventLogout              SecurityEventType = "logout"
	EventPasswordChange     SecurityEventType = "password_change"
	EventPasswordReset      SecurityEventType = "password_reset"
	EventAccountLocked      SecurityEventType = "account_locked"
	EventAccountUnlocked    SecurityEventType = "account_unlocked"
	EventBruteForce        SecurityEventType = "brute_force_attack"
	EventXSSAttempt         SecurityEventType = "xss_attempt"
	EventSQLInjection      SecurityEventType = "sql_injection_attempt"
	EventCSRFViolation     SecurityEventType = "csrf_violation"
	EventRateLimitExceeded SecurityEventType = "rate_limit_exceeded"
	EventIPBlocked         SecurityEventType = "ip_blocked"
	EventIPUnblocked       SecurityEventType = "ip_unblocked"
	EventPermissionDenied  SecurityEventType = "permission_denied"
	EventSensitiveAccess  SecurityEventType = "sensitive_data_access"
	EventAdminAction       SecurityEventType = "admin_action"
	EventConfigChange      SecurityEventType = "config_change"
	EventAPIKeyGenerated   SecurityEventType = "api_key_generated"
	EventAPIKeyRevoked     SecurityEventType = "api_key_revoked"
)

// AuditLogger 审计日志记录器
type AuditLogger struct {
	logs       []SecurityLog
	maxSize    int
	mu         sync.RWMutex
	filePath   string
	enableFile bool
	enableJSON bool
}

// NewAuditLogger 创建审计日志记录器
func NewAuditLogger(filePath string, maxSize int) *AuditLogger {
	if maxSize <= 0 {
		maxSize = 10000
	}

	logger := &AuditLogger{
		logs:       make([]SecurityLog, 0, maxSize),
		maxSize:    maxSize,
		filePath:   filePath,
		enableFile: filePath != "",
		enableJSON: true,
	}

	if logger.enableFile {
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.enableFile = false
		}
	}

	return logger
}

// Log 记录日志
func (l *AuditLogger) Log(log SecurityLog) {
	log.Timestamp = time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.logs = append(l.logs, log)

	if len(l.logs) > l.maxSize {
		l.logs = l.logs[1:]
	}

	if l.enableFile {
		l.writeToFile(log)
	}
}

// LogEvent 记录安全事件
func (l *AuditLogger) LogEvent(eventType SecurityEventType, ip string, details map[string]interface{}) {
	level := LogLevelInfo

	switch eventType {
	case EventBruteForce, EventSQLInjection, EventXSSAttempt, EventCSRFViolation:
		level = LogLevelCritical
	case EventAccountLocked, EventIPBlocked, EventPermissionDenied:
		level = LogLevelWarning
	}

	l.Log(SecurityLog{
		Level:     level,
		Event:     string(eventType),
		IPAddress: ip,
		Details:   details,
	})
}

// LogLogin 记录登录事件
func (l *AuditLogger) LogLogin(success bool, username, ip string, userID uint) {
	eventType := EventLoginSuccess
	if !success {
		eventType = EventLoginFailure
	}

	l.Log(SecurityLog{
		Level:     LogLevelInfo,
		Event:     string(eventType),
		IPAddress: ip,
		UserID:    userID,
		Details: map[string]interface{}{
			"username": username,
		},
	})
}

// LogSecurityViolation 记录安全违规
func (l *AuditLogger) LogSecurityViolation(violationType string, ip, path, userAgent string, details map[string]interface{}) {
	l.Log(SecurityLog{
		Level:       LogLevelCritical,
		Event:       violationType,
		IPAddress:   ip,
		RequestPath: path,
		UserAgent:   userAgent,
		Details:     details,
	})
}

// GetLogs 获取日志
func (l *AuditLogger) GetLogs(start, end time.Time, level SecurityLogLevel, event string) []SecurityLog {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]SecurityLog, 0)

	for _, log := range l.logs {
		if !log.Timestamp.After(start) || !log.Timestamp.Before(end) {
			continue
		}

		if level != "" && log.Level != level {
			continue
		}

		if event != "" && log.Event != event {
			continue
		}

		result = append(result, log)
	}

	return result
}

// GetRecentLogs 获取最近的日志
func (l *AuditLogger) GetRecentLogs(count int) []SecurityLog {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if count <= 0 {
		count = 100
	}

	if count > len(l.logs) {
		count = len(l.logs)
	}

	start := len(l.logs) - count
	result := make([]SecurityLog, count)
	copy(result, l.logs[start:])

	return result
}

// GetCriticalLogs 获取关键日志
func (l *AuditLogger) GetCriticalLogs(since time.Duration) []SecurityLog {
	l.mu.RLock()
	defer l.mu.RUnlock()

	cutoff := time.Now().Add(-since)
	result := make([]SecurityLog, 0)

	for _, log := range l.logs {
		if log.Timestamp.After(cutoff) && (log.Level == LogLevelCritical || log.Level == LogLevelWarning) {
			result = append(result, log)
		}
	}

	return result
}

// GetIPLogs 获取IP的日志
func (l *AuditLogger) GetIPLogs(ip string, limit int) []SecurityLog {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]SecurityLog, 0)
	count := 0

	for i := len(l.logs) - 1; i >= 0; i-- {
		log := l.logs[i]
		if log.IPAddress == ip {
			result = append(result, log)
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}

	return result
}

// GetUserLogs 获取用户的日志
func (l *AuditLogger) GetUserLogs(userID uint, limit int) []SecurityLog {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]SecurityLog, 0)
	count := 0

	for i := len(l.logs) - 1; i >= 0; i-- {
		log := l.logs[i]
		if log.UserID == userID {
			result = append(result, log)
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}

	return result
}

// GetStats 获取统计信息
func (l *AuditLogger) GetStats(since time.Duration) map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	cutoff := time.Now().Add(-since)

	stats := map[string]interface{}{
		"total_events":      0,
		"critical_events":   0,
		"warning_events":    0,
		"info_events":       0,
		"unique_ips":        0,
		"unique_users":      0,
		"event_counts":      map[string]int{},
	}

	ipSet := make(map[string]bool)
	userSet := make(map[uint]bool)
	eventCounts := make(map[string]int)

	for _, log := range l.logs {
		if log.Timestamp.Before(cutoff) {
			continue
		}

		stats["total_events"] = stats["total_events"].(int) + 1

		switch log.Level {
		case LogLevelCritical:
			stats["critical_events"] = stats["critical_events"].(int) + 1
		case LogLevelWarning:
			stats["warning_events"] = stats["warning_events"].(int) + 1
		case LogLevelInfo:
			stats["info_events"] = stats["info_events"].(int) + 1
		}

		if log.IPAddress != "" {
			ipSet[log.IPAddress] = true
		}

		if log.UserID > 0 {
			userSet[log.UserID] = true
		}

		eventCounts[log.Event]++
	}

	stats["unique_ips"] = len(ipSet)
	stats["unique_users"] = len(userSet)
	stats["event_counts"] = eventCounts

	return stats
}

// writeToFile 写入文件
func (l *AuditLogger) writeToFile(log SecurityLog) {
	data, err := json.Marshal(log)
	if err != nil {
		return
	}

	file, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	file.Write(data)
	file.WriteString("\n")
}

// Clear 清除日志
func (l *AuditLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = make([]SecurityLog, 0, l.maxSize)
}

// Export 导出日志
func (l *AuditLogger) Export() []SecurityLog {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]SecurityLog, len(l.logs))
	copy(result, l.logs)

	return result
}

// GlobalAuditLogger 全局审计日志记录器
var GlobalAuditLogger *AuditLogger

// InitGlobalAuditLogger 初始化全局审计日志记录器
func InitGlobalAuditLogger(filePath string) {
	GlobalAuditLogger = NewAuditLogger(filePath, 10000)
}

// LogSecurityEvent 记录安全事件的便捷函数
func LogSecurityEvent(eventType string, ip string, details map[string]interface{}) {
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogEvent(SecurityEventType(eventType), ip, details)
	}
}

// LogSecurityViolation 记录安全违规的便捷函数
func LogSecurityViolation(violationType, ip, path, userAgent string, details map[string]interface{}) {
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogSecurityViolation(violationType, ip, path, userAgent, details)
	}
}

// LoginSecurityLog 登录安全日志
func LoginSecurityLog(success bool, username, ip string, userID uint) {
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogLogin(success, username, ip, userID)
	}
}

// GetSecurityStats 获取安全统计
func GetSecurityStats(since time.Duration) map[string]interface{} {
	if GlobalAuditLogger != nil {
		return GlobalAuditLogger.GetStats(since)
	}
	return map[string]interface{}{}
}

// GetRecentSecurityLogs 获取最近的安全日志
func GetRecentSecurityLogs(count int) []SecurityLog {
	if GlobalAuditLogger != nil {
		return GlobalAuditLogger.GetRecentLogs(count)
	}
	return []SecurityLog{}
}

// GetCriticalSecurityLogs 获取关键安全日志
func GetCriticalSecurityLogs(since time.Duration) []SecurityLog {
	if GlobalAuditLogger != nil {
		return GlobalAuditLogger.GetCriticalLogs(since)
	}
	return []SecurityLog{}
}
