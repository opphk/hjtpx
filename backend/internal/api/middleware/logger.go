package middleware

import (
	"encoding/json"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/metrics"
)

var (
	logBuffer   = make([]LogEntry, 0, 1000)
	logBufferMu sync.Mutex
	logFile     *os.File
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type LogEntry struct {
	Timestamp   string                 `json:"timestamp"`
	Level       LogLevel               `json:"level"`
	RequestID   string                 `json:"request_id"`
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	StatusCode  int                    `json:"status_code"`
	Latency     string                 `json:"latency"`
	LatencyMs   float64                `json:"latency_ms"`
	ClientIP    string                 `json:"client_ip"`
	UserAgent   string                 `json:"user_agent"`
	BodySize    int                    `json:"body_size"`
	Query       string                 `json:"query,omitempty"`
	Error       string                 `json:"error,omitempty"`
	ErrorType   string                 `json:"error_type,omitempty"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	Service     string                 `json:"service"`
	Environment string                 `json:"environment"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func InitLogger(logPath string) error {
	if logPath == "" {
		logPath = "/var/log/hjtpx/app.log"
	}

	dir := "/var/log/hjtpx"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	logFile = file
	return nil
}

func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		bodySize := c.Writer.Size()

		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("X-Request-ID", requestID)
		c.Header("X-Request-ID", requestID)

		entry := LogEntry{
			Timestamp:   start.Format(time.RFC3339Nano),
			RequestID:   requestID,
			Method:      method,
			Path:        path,
			StatusCode:  statusCode,
			Latency:     latency.String(),
			LatencyMs:   latency.Seconds() * 1000,
			ClientIP:    clientIP,
			UserAgent:   c.Request.UserAgent(),
			BodySize:    bodySize,
			Query:       query,
			Service:     "hjtpx",
			Environment: getEnv("GIN_MODE", "production"),
		}

		if len(c.Errors) > 0 {
			entry.Error = c.Errors.String()
			entry.Level = LogLevelError
		} else if statusCode >= 500 {
			entry.Level = LogLevelError
		} else if statusCode >= 400 {
			entry.Level = LogLevelWarn
		} else {
			entry.Level = LogLevelInfo
		}

		metrics.IncrementRequestCount()
		if statusCode >= 500 {
			metrics.IncrementFailureCount()
		} else if statusCode >= 200 && statusCode < 300 {
			metrics.IncrementSuccessCount()
		}

		writeLog(entry)
	}
}

func writeLog(entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	if logFile != nil {
		logFile.Write(append(data, '\n'))
	}

	if len(os.Getenv("LOG_STDOUT")) != 0 {
		println(string(data))
	}
}

func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

var randomSrc = rand.New(rand.NewSource(time.Now().UnixNano()))

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[randomSrc.Intn(len(letters))]
	}
	return string(b)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

type AuditLog struct {
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resource_id"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
	RequestID  string    `json:"request_id"`
	Success    bool      `json:"success"`
	ErrorMsg   string    `json:"error_msg,omitempty"`
	OldValue   string    `json:"old_value,omitempty"`
	NewValue   string    `json:"new_value,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

func WriteAuditLog(log AuditLog) {
	log.Timestamp = time.Now()

	data, err := json.Marshal(log)
	if err != nil {
		return
	}

	if logFile != nil {
		logFile.Write(append(data, '\n'))
	}
}
