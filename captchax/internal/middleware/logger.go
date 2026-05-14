package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type LogEntry struct {
	Timestamp    string                 `json:"timestamp"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	StatusCode   int                    `json:"status_code"`
	Latency     string                 `json:"latency"`
	ClientIP     string                 `json:"client_ip"`
	UserAgent    string                 `json:"user_agent"`
	BodySize     int                    `json:"body_size"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	Fields       map[string]interface{} `json:"fields,omitempty"`
}

var (
	multiSpaceRegex = regexp.MustCompile(`\s+`)
	requestIDRegex  = regexp.MustCompile(`^[a-zA-Z0-9\-_]{8,64}$`)
)

func Logger() gin.HandlerFunc {
	return LoggerWithConfig(&LoggerConfig{
		SkipPaths: []string{"/health", "/metrics"},
	})
}

type LoggerConfig struct {
	SkipPaths     []string
	RequestIDFunc func(*gin.Context) string
}

func LoggerWithConfig(config *LoggerConfig) gin.HandlerFunc {
	skipPathsMap := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPathsMap[path] = true
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		if skipPathsMap[path] {
			c.Next()
			return
		}

		requestID := ""
		if config != nil && config.RequestIDFunc != nil {
			requestID = config.RequestIDFunc(c)
		}
		if requestID == "" {
			requestID = c.GetHeader("X-Request-ID")
			if requestID != "" && !requestIDRegex.MatchString(requestID) {
				requestID = ""
			}
		}
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()
		method := c.Request.Method
		bodySize := c.Writer.Size()

		errorMessage := ""
		if len(c.Errors) > 0 {
			errorMessage = c.Errors.String()
		}

		entry := LogEntry{
			Timestamp:    start.UTC().Format(time.RFC3339),
			Method:       method,
			Path:         path,
			StatusCode:   statusCode,
			Latency:     latency.String(),
			ClientIP:    clientIP,
			UserAgent:   truncateString(userAgent, 200),
			BodySize:    bodySize,
			ErrorMessage: errorMessage,
			RequestID:    requestID,
			Fields:      collectLogFields(c),
		}

		logJSON, _ := json.Marshal(entry)
		fmt.Printf("%s\n", logJSON)

		if statusCode >= 500 {
			logRequestDetails(c, requestBody, entry)
		}
	}
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID, _ := c.Get("request_id")
				entry := LogEntry{
					Timestamp:    time.Now().UTC().Format(time.RFC3339),
					Method:       c.Request.Method,
					Path:         c.Request.URL.Path,
					StatusCode:   500,
					ClientIP:     c.ClientIP(),
					UserAgent:    c.Request.UserAgent(),
					ErrorMessage: fmt.Sprintf("panic recovered: %v", err),
					RequestID:    fmt.Sprintf("%v", requestID),
				}
				logJSON, _ := json.Marshal(entry)
				fmt.Printf("%s\n", logJSON)

				c.AbortWithStatusJSON(500, gin.H{
					"error":       "internal server error",
					"request_id":  requestID,
				})
			}
		}()
		c.Next()
	}
}

func collectLogFields(c *gin.Context) map[string]interface{} {
	fields := make(map[string]interface{})

	if appID := c.GetHeader("X-App-ID"); appID != "" {
		fields["app_id"] = appID
	}

	if domain := c.Query("domain"); domain != "" {
		fields["domain"] = domain
	}

	if latency := c.GetHeader("X-Response-Time"); latency != "" {
		fields["response_time_header"] = latency
	}

	return fields
}

func logRequestDetails(c *gin.Context, requestBody []byte, entry LogEntry) {
	if len(requestBody) > 0 {
		sanitizedBody := sanitizeBody(requestBody)
		if len(sanitizedBody) > 0 {
			fmt.Printf("[REQUEST-BODY] RequestID: %s, Body: %s\n", entry.RequestID, sanitizedBody)
		}
	}
}

func sanitizeBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return truncateString(string(body), 500)
	}

	sensitiveFields := []string{"password", "token", "secret", "key", "authorization", "credential"}
	for _, field := range sensitiveFields {
		if val, ok := data[field]; ok {
			data[field] = "[REDACTED]"
			_ = val
		}
	}

	result, _ := json.Marshal(data)
	return truncateString(string(result), 500)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func generateRequestID() string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%x", timestamp)
}

func SanitizeLogPath(path string) string {
	path = strings.TrimSpace(path)
	path = multiSpaceRegex.ReplaceAllString(path, " ")
	return path
}
