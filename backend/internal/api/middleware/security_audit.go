package middleware

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

var (
	securityAudit *service.SecurityAuditService
	auditOnce     = &sync.Once{}
)

func initSecurityAudit() {
	auditOnce.Do(func() {
		securityAudit = service.NewSecurityAuditService()
	})
}

type SecurityAuditConfig struct {
	Enabled         bool
	ExcludePaths    []string
	LogRequestBody  bool
	LogResponseBody bool
}

var DefaultSecurityAuditConfig = SecurityAuditConfig{
	Enabled:         true,
	LogRequestBody:  false,
	LogResponseBody: false,
}

func SecurityAuditMiddleware(config ...SecurityAuditConfig) gin.HandlerFunc {
	initSecurityAudit()

	cfg := DefaultSecurityAuditConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		for _, excluded := range cfg.ExcludePaths {
			if path == excluded || pathHasPrefix(path, excluded+"/") {
				c.Next()
				return
			}
		}

		start := time.Now()
		details := make(map[string]interface{})

		if cfg.LogRequestBody && c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			if len(bodyBytes) > 0 {
				details["request_body"] = string(bodyBytes)
			}
		}

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		intrusionEvents := securityAudit.DetectIntrusionAttempts(c.Request)
		for _, event := range intrusionEvents {
			_ = event
		}

		duration := time.Since(start)
		details["duration_ms"] = duration.Milliseconds()
		details["status_code"] = c.Writer.Status()

		if cfg.LogResponseBody {
			details["response_body"] = blw.body.String()
		}

		if c.Writer.Status() >= 400 {
			eventType := service.EventAccessDenied
			if c.Writer.Status() == http.StatusTooManyRequests {
				eventType = service.EventRateLimitHit
			}
			securityAudit.LogEvent(eventType, c.Request, details)
		}

		c.Set("security_audit", securityAudit)
	}
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func GetSecurityAuditService() *service.SecurityAuditService {
	initSecurityAudit()
	return securityAudit
}
