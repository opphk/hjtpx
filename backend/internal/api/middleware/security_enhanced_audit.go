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
	securityEnhancedAuditService *service.SecurityEnhancedAuditService
	securityEnhancedAuditOnce   = &sync.Once{}
)

func initSecurityEnhancedAudit() {
	securityEnhancedAuditOnce.Do(func() {
		securityEnhancedAuditService = service.NewSecurityEnhancedAuditService()
	})
}

type SecurityEnhancedAuditConfig struct {
	Enabled          bool
	ExcludePaths     []string
	LogRequestBody   bool
	LogResponseBody  bool
	LogHeaders       bool
	EnableCompliance bool
	EnableAnomaly   bool
}

var DefaultSecurityEnhancedAuditConfig = SecurityEnhancedAuditConfig{
	Enabled:          true,
	LogHeaders:       true,
	EnableCompliance: true,
	EnableAnomaly:    true,
}

func SecurityEnhancedAuditMiddleware(config ...SecurityEnhancedAuditConfig) gin.HandlerFunc {
	initSecurityEnhancedAudit()

	cfg := DefaultSecurityEnhancedAuditConfig
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
		requestDetails := make(map[string]interface{})

		if cfg.LogHeaders {
			headers := make(map[string]string)
			for key, values := range c.Request.Header {
				if len(values) > 0 {
					headers[key] = values[0]
				}
			}
			requestDetails["headers"] = headers
		}

		var requestBody []byte
		if cfg.LogRequestBody && c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			if len(requestBody) > 0 {
				requestDetails["body"] = string(requestBody)
			}
		}

		blw := &enhancedBodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		duration := time.Since(start)
		requestDetails["duration_ms"] = duration.Milliseconds()
		requestDetails["status_code"] = c.Writer.Status()

		if cfg.LogResponseBody && blw.body.Len() > 0 {
			requestDetails["response_body"] = blw.body.String()
		}

		operation := determineOperation(c)
		resource := determineResource(c)
		action := c.Request.Method

		securityEnhancedAuditService.LogOperation(operation, resource, action, c.Request, requestDetails)

		if cfg.EnableAnomaly {
			go func() {
				anomalies := securityEnhancedAuditService.DetectAnomalies(c.ClientIP(), 5*time.Minute)
				for _, anomaly := range anomalies {
					evidence := map[string]interface{}{
						"pattern_id": anomaly.PatternID,
						"pattern_name": anomaly.Name,
						"request_path": c.Request.URL.Path,
					}
					securityEnhancedAuditService.LogAnomaly(anomaly.PatternID, c.Request, evidence)
				}
			}()
		}

		if cfg.EnableCompliance {
			go func() {
				context := map[string]interface{}{
					"logged":      true,
					"authorized":  c.Writer.Status() < 400,
					"user_id":     getUserIDFromContext(c),
				}
				violations := securityEnhancedAuditService.CheckCompliance(c.Request, context)
				for _, violation := range violations {
					securityEnhancedAuditService.LogComplianceViolation(violation)
				}
			}()
		}

		if c.Writer.Status() >= 400 {
			securityEnhancedAuditService.LogAccessDenied(
				http.StatusText(c.Writer.Status()),
				c.Request,
				requestDetails,
			)
		}

		c.Set("security_audit_record", true)
	}
}

func determineOperation(c *gin.Context) string {
	path := c.Request.URL.Path
	method := c.Request.Method

	switch {
	case path == "/api/login" || path == "/login":
		return "user_login"
	case path == "/api/logout" || path == "/logout":
		return "user_logout"
	case path == "/api/register" || path == "/register":
		return "user_registration"
	case path == "/api/admin":
		return "admin_access"
	case path == "/api/captcha":
		return "captcha_verification"
	case path == "/api/data" || stringsHasPrefix(path, "/api/data/"):
		return "data_access"
	case path == "/api/upload" || stringsHasPrefix(path, "/api/upload/"):
		return "file_upload"
	case path == "/api/settings":
		return "settings_change"
	default:
		return method + "_" + path
	}
}

func determineResource(c *gin.Context) string {
	path := c.Request.URL.Path

	switch {
	case stringsHasPrefix(path, "/api/admin"):
		return "admin_panel"
	case stringsHasPrefix(path, "/api/user"):
		return "user_data"
	case stringsHasPrefix(path, "/api/captcha"):
		return "captcha_service"
	case stringsHasPrefix(path, "/api/stats"):
		return "statistics"
	case stringsHasPrefix(path, "/api/data"):
		return "application_data"
	default:
		return "api_endpoint"
	}
}

func getUserIDFromContext(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	if userID, exists := c.Get("userID"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

type enhancedBodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *enhancedBodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

type SecurityAuditHandler struct{}

func (h *SecurityAuditHandler) GetRecords(c *gin.Context) {
	filter := &service.AuditFilter{}

	if category := c.Query("category"); category != "" {
		filter.Category = service.AuditCategory(category)
	}
	if severity := c.Query("severity"); severity != "" {
		filter.Severity = service.AuditSeverity(severity)
	}
	if sourceIP := c.Query("ip"); sourceIP != "" {
		filter.SourceIP = sourceIP
	}
	if userID := c.Query("user_id"); userID != "" {
		filter.UserID = userID
	}
	if operation := c.Query("operation"); operation != "" {
		filter.Operation = operation
	}
	if tag := c.Query("compliance_tag"); tag != "" {
		filter.ComplianceTag = tag
	}
	if limit := c.Query("limit"); limit != "" {
		var l int
		if _, err := parseInt(limit, &l); err == nil {
			filter.Limit = l
		}
	}

	records := securityEnhancedAuditService.GetRecords(filter)

	c.JSON(http.StatusOK, gin.H{
		"records": records,
		"count":   len(records),
	})
}

func (h *SecurityAuditHandler) GetStatistics(c *gin.Context) {
	stats := securityEnhancedAuditService.GetStatistics()

	c.JSON(http.StatusOK, gin.H{
		"total_records":            stats.TotalRecords,
		"by_category":              stats.ByCategory,
		"by_severity":              stats.BySeverity,
		"by_status":                stats.ByStatus,
		"compliance_violations":    stats.ComplianceViolations,
	})
}

func (h *SecurityAuditHandler) GetAnomalyPatterns(c *gin.Context) {
	patterns := securityEnhancedAuditService.GetAnomalyPatterns()

	c.JSON(http.StatusOK, gin.H{
		"patterns": patterns,
		"count":    len(patterns),
	})
}

func (h *SecurityAuditHandler) UpdateAnomalyPattern(c *gin.Context) {
	var req struct {
		PatternID string `json:"pattern_id" binding:"required"`
		Enabled   bool   `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := securityEnhancedAuditService.UpdateAnomalyPattern(req.PatternID, req.Enabled)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pattern_id": req.PatternID,
		"enabled":   req.Enabled,
		"message":   "Anomaly pattern updated",
	})
}

func (h *SecurityAuditHandler) GetComplianceRules(c *gin.Context) {
	rules := securityEnhancedAuditService.GetComplianceRules()

	c.JSON(http.StatusOK, gin.H{
		"rules": rules,
		"count": len(rules),
	})
}

func (h *SecurityAuditHandler) UpdateComplianceRule(c *gin.Context) {
	var req struct {
		RuleID  string `json:"rule_id" binding:"required"`
		Enabled bool   `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := securityEnhancedAuditService.UpdateComplianceRule(req.RuleID, req.Enabled)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rule_id": req.RuleID,
		"enabled": req.Enabled,
		"message": "Compliance rule updated",
	})
}

func (h *SecurityAuditHandler) CheckCompliance(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Path == "" {
		req.Path = c.Request.URL.Path
	}

	r, _ := http.NewRequest(c.Request.Method, req.Path, nil)
	r.Header = c.Request.Header.Clone()

	context := map[string]interface{}{
		"logged":     true,
		"authorized": true,
	}

	violations := securityEnhancedAuditService.CheckCompliance(r, context)

	c.JSON(http.StatusOK, gin.H{
		"path":        req.Path,
		"violations":  violations,
		"is_compliant": len(violations) == 0,
	})
}

func (h *SecurityAuditHandler) ExportRecords(c *gin.Context) {
	format := c.DefaultQuery("format", "json")

	filter := &service.AuditFilter{}

	if category := c.Query("category"); category != "" {
		filter.Category = service.AuditCategory(category)
	}
	if severity := c.Query("severity"); severity != "" {
		filter.Severity = service.AuditSeverity(severity)
	}
	if sourceIP := c.Query("ip"); sourceIP != "" {
		filter.SourceIP = sourceIP
	}

	data, err := securityEnhancedAuditService.ExportRecords(format, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if format == "csv" {
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=audit_records.csv")
	} else {
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=audit_records.json")
	}

	c.Data(http.StatusOK, "application/octet-stream", data)
}

func GetSecurityEnhancedAuditService() *service.SecurityEnhancedAuditService {
	initSecurityEnhancedAudit()
	return securityEnhancedAuditService
}

func parseInt(s string, result *int) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, &parseError{s}
		}
		n = n*10 + int(c-'0')
	}
	*result = n
	return n, nil
}

type parseError struct {
	s string
}

func (e *parseError) Error() string {
	return "invalid integer: " + e.s
}
