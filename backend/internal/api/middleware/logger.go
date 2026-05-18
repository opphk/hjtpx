package middleware

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/metrics"
)

var (
	logFile     *os.File
	logStdout   bool
	logBuffer   []LogEntry
	logBufferMu sync.Mutex
	logChan     chan LogEntry
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

var logEntryPool = sync.Pool{
	New: func() interface{} {
		return &LogEntry{
			Metadata: make(map[string]interface{}, 4),
		}
	},
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

var randomSrc = rand.New(rand.NewSource(time.Now().UnixNano()))

func init() {
	logBuffer = make([]LogEntry, 0, 100)
	logChan = make(chan LogEntry, 1000)
	logStdout = os.Getenv("LOG_STDOUT") != ""
	go logWriter()
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
	close(logChan)
	if logFile != nil {
		logFile.Close()
	}
}

func logWriter() {
	for entry := range logChan {
		writeLog(entry)
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

		entry := logEntryPool.Get().(*LogEntry)
		entry.Timestamp = start.Format(time.RFC3339Nano)
		entry.RequestID = requestID
		entry.Method = method
		entry.Path = path
		entry.StatusCode = statusCode
		entry.Latency = latency.String()
		entry.LatencyMs = latency.Seconds() * 1000
		entry.ClientIP = clientIP
		entry.UserAgent = c.Request.UserAgent()
		entry.BodySize = bodySize
		entry.Query = query
		entry.Service = "hjtpx"
		entry.Environment = getEnv("GIN_MODE", "production")
		entry.Error = ""
		entry.ErrorType = ""
		entry.TraceID = ""
		entry.SpanID = ""
		for k := range entry.Metadata {
			delete(entry.Metadata, k)
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

		select {
		case logChan <- *entry:
		default:
			logBufferMu.Lock()
			logBuffer = append(logBuffer, *entry)
			if len(logBuffer) >= cap(logBuffer) {
				flushLogBuffer()
			}
			logBufferMu.Unlock()
		}

		logEntryPool.Put(entry)
	}
}

func flushLogBuffer() {
	for _, entry := range logBuffer {
		writeLog(entry)
	}
	logBuffer = logBuffer[:0]
}

func writeLog(entry LogEntry) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(entry); err != nil {
		bufPool.Put(buf)
		return
	}

	data := buf.Bytes()

	if logFile != nil {
		logFile.Write(data)
	}

	if logStdout {
		os.Stdout.Write(data)
	}

	bufPool.Put(buf)
}

func generateRequestID() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"[randomSrc.Intn(62)]
	}
	return time.Now().Format("20060102150405") + "-" + string(b)
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
