package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hjtpx/hjtpx/internal/service"
)

var (
	anomalyService    *service.AnomalyDetectionService
	inputValidator    *service.InputValidator
)

func init() {
	anomalyService = service.NewAnomalyDetectionService()
	inputValidator = service.NewInputValidator()
}

type AnomalyDetectionConfig struct {
	Enabled        bool
	KeyFunc        func(c *gin.Context) string
	BlockAnomaly   bool
	ExcludePaths   []string
}

var defaultAnomalyConfig = AnomalyDetectionConfig{
	Enabled:      true,
	BlockAnomaly: false,
	KeyFunc: func(c *gin.Context) string {
		return c.ClientIP()
	},
}

func AnomalyDetectionMiddleware(config ...AnomalyDetectionConfig) gin.HandlerFunc {
	cfg := defaultAnomalyConfig
	if len(config) > 0 {
		cfg = config[0]
		if cfg.KeyFunc == nil {
			cfg.KeyFunc = defaultAnomalyConfig.KeyFunc
		}
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

		clientID := cfg.KeyFunc(c)
		userAgent := c.GetHeader("User-Agent")

		var bodySize int
		if c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			bodySize = len(bodyBytes)
			c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		anomalyService.RecordTraffic(clientID, bodySize, c.Request.Method, path, userAgent)
		result := anomalyService.DetectAnomaly(clientID)

		c.Set("anomaly_result", result)
		c.Header("X-Anomaly-Score", string(rune(result.Score*100)))

		if result.IsAnomaly && cfg.BlockAnomaly {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "anomaly_detected",
				"message": "Suspicious traffic pattern detected",
				"type":    result.AnomalyType,
			})
			return
		}

		c.Next()
	}
}

type InputValidationConfig struct {
	Enabled      bool
	ValidateQuery bool
	ValidateForm  bool
	ValidateJSON  bool
	ExcludePaths []string
}

var defaultInputValidationConfig = InputValidationConfig{
	Enabled:      true,
	ValidateQuery: true,
	ValidateForm:  true,
	ValidateJSON:  true,
}

func InputValidationMiddleware(config ...InputValidationConfig) gin.HandlerFunc {
	cfg := defaultInputValidationConfig
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

		if cfg.ValidateQuery {
			for key, values := range c.Request.URL.Query() {
				for _, value := range values {
					result := inputValidator.ValidateInput(value)
					if !result.IsValid {
						c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
							"error":   "invalid_input",
							"message": "Invalid query parameter: " + key,
							"details": result.Errors,
						})
						return
					}
				}
			}
		}

		if cfg.ValidateForm && c.Request.Method == http.MethodPost {
			if err := c.Request.ParseForm(); err == nil {
				for key, values := range c.Request.PostForm {
					for _, value := range values {
						result := inputValidator.ValidateInput(value)
						if !result.IsValid {
							c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
								"error":   "invalid_input",
								"message": "Invalid form parameter: " + key,
								"details": result.Errors,
							})
							return
						}
					}
				}
			}
		}

		c.Next()
	}
}

type SecurityHeadersConfig struct {
	Enabled        bool
	CSP            string
	HSTS           string
	XFrameOptions  string
	ContentTypeOptions string
	XSSProtection  string
	ReferrerPolicy string
}

var defaultSecurityHeadersConfig = SecurityHeadersConfig{
	Enabled:          true,
	CSP:              service.DefaultSecurityHeaders.CSP,
	HSTS:             service.DefaultSecurityHeaders.HSTS,
	XFrameOptions:    service.DefaultSecurityHeaders.XFrameOptions,
	ContentTypeOptions: service.DefaultSecurityHeaders.XContentTypeOptions,
	XSSProtection:    service.DefaultSecurityHeaders.XXSSProtection,
	ReferrerPolicy:   service.DefaultSecurityHeaders.ReferrerPolicy,
}

func SecurityHeadersMiddleware(config ...SecurityHeadersConfig) gin.HandlerFunc {
	cfg := defaultSecurityHeadersConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		if cfg.CSP != "" {
			c.Header("Content-Security-Policy", cfg.CSP)
		}

		if cfg.HSTS != "" {
			c.Header("Strict-Transport-Security", cfg.HSTS)
		}

		if cfg.XFrameOptions != "" {
			c.Header("X-Frame-Options", cfg.XFrameOptions)
		}

		if cfg.ContentTypeOptions != "" {
			c.Header("X-Content-Type-Options", cfg.ContentTypeOptions)
		}

		if cfg.XSSProtection != "" {
			c.Header("X-XSS-Protection", cfg.XSSProtection)
		}

		if cfg.ReferrerPolicy != "" {
			c.Header("Referrer-Policy", cfg.ReferrerPolicy)
		}

		c.Header("X-Permitted-Cross-Domain-Policies", "none")
		c.Header("X-Download-Options", "noopen")

		c.Next()
	}
}

type SecurityLog struct {
	Timestamp    string                 `json:"timestamp"`
	ClientIP     string                 `json:"client_ip"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	StatusCode   int                    `json:"status_code"`
	UserAgent    string                 `json:"user_agent"`
	Fingerprint  string                 `json:"fingerprint,omitempty"`
	AnomalyScore float64                `json:"anomaly_score,omitempty"`
	IsAnomaly    bool                   `json:"is_anomaly,omitempty"`
	RateLimited  bool                   `json:"rate_limited,omitempty"`
	RiskScore    float64                `json:"risk_score,omitempty"`
	Headers      map[string]string      `json:"headers,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

type SecurityLoggerConfig struct {
	Enabled        bool
	LogHeaders     bool
	LogBody        bool
	ExcludePaths   []string
}

var defaultSecurityLoggerConfig = SecurityLoggerConfig{
	Enabled:    true,
	LogHeaders: true,
}

func SecurityLoggerMiddleware(config ...SecurityLoggerConfig) gin.HandlerFunc {
	cfg := defaultSecurityLoggerConfig
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

		c.Next()

		log := &SecurityLog{
			Timestamp:  c.GetHeader("Date"),
			ClientIP:   c.ClientIP(),
			Method:     c.Request.Method,
			Path:       path,
			StatusCode: c.Writer.Status(),
			UserAgent:  c.GetHeader("User-Agent"),
		}

		if fp, exists := c.Get("fingerprint_id"); exists {
			log.Fingerprint = fp.(string)
		}

		if score, exists := c.Get("risk_score"); exists {
			log.RiskScore = score.(float64)
		}

		if anomalyResult, exists := c.Get("anomaly_result"); exists {
			if result, ok := anomalyResult.(interface{ GetScore() float64; IsAnomaly() bool }); ok {
				log.AnomalyScore = result.GetScore()
				log.IsAnomaly = result.IsAnomaly()
			}
		}

		if rateResult, exists := c.Get("rate_limit_result"); exists {
			if result, ok := rateResult.(*service.SmartRateLimitResult); ok {
				log.RateLimited = !result.Allowed
			}
		}

		if cfg.LogHeaders {
			log.Headers = make(map[string]string)
			for key, values := range c.Request.Header {
				if len(values) > 0 {
					log.Headers[key] = values[0]
				}
			}
		}

		c.Set("security_log", log)
	}
}

func GetAnomalyService() *service.AnomalyDetectionService {
	return anomalyService
}

func GetInputValidator() *service.InputValidator {
	return inputValidator
}

func GetAnomalyResult(c *gin.Context) interface{} {
	if result, exists := c.Get("anomaly_result"); exists {
		return result
	}
	return nil
}

func GetSecurityLog(c *gin.Context) *SecurityLog {
	if log, exists := c.Get("security_log"); exists {
		return log.(*SecurityLog)
	}
	return nil
}
