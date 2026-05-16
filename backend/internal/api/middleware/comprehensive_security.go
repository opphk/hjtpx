package middleware

import "github.com/gin-gonic/gin"

type ComprehensiveSecurityConfig struct {
	Fingerprint      FingerprintConfig
	RateLimit        SmartRateLimitConfig
	ReplayProtection ReplayProtectionConfig
	AnomalyDetection AnomalyDetectionConfig
	InputValidation  InputValidationConfig
	SecurityHeaders  SecurityHeadersConfig
	SecurityLogger   SecurityLoggerConfig
	Enabled          bool
	ExcludePaths     []string
}

var DefaultComprehensiveSecurityConfig = ComprehensiveSecurityConfig{
	Enabled: true,
	Fingerprint: FingerprintConfig{
		Enabled:        true,
		CheckBlacklist: true,
		BlockAnomaly:   false,
	},
	RateLimit: SmartRateLimitConfig{
		Enabled: true,
	},
	ReplayProtection: ReplayProtectionConfig{
		Enabled:        true,
		ErrorOnFailure: false,
	},
	AnomalyDetection: AnomalyDetectionConfig{
		Enabled:      true,
		BlockAnomaly: false,
	},
	InputValidation: InputValidationConfig{
		Enabled:       true,
		ValidateQuery: true,
		ValidateForm:  true,
	},
	SecurityHeaders: SecurityHeadersConfig{
		Enabled: true,
	},
	SecurityLogger: SecurityLoggerConfig{
		Enabled:    true,
		LogHeaders: true,
	},
}

func ComprehensiveSecurityMiddleware(config ...ComprehensiveSecurityConfig) gin.HandlerFunc {
	cfg := DefaultComprehensiveSecurityConfig
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

		handlers := make([]gin.HandlerFunc, 0)

		if cfg.SecurityHeaders.Enabled {
			handlers = append(handlers, SecurityHeadersMiddleware(cfg.SecurityHeaders))
		}

		if cfg.Fingerprint.Enabled {
			handlers = append(handlers, FingerprintMiddleware(cfg.Fingerprint))
		}

		if cfg.InputValidation.Enabled {
			handlers = append(handlers, InputValidationMiddleware(cfg.InputValidation))
		}

		if cfg.ReplayProtection.Enabled {
			handlers = append(handlers, ReplayProtectionMiddleware(cfg.ReplayProtection))
		}

		if cfg.AnomalyDetection.Enabled {
			handlers = append(handlers, AnomalyDetectionMiddleware(cfg.AnomalyDetection))
		}

		if cfg.RateLimit.Enabled {
			handlers = append(handlers, SmartRateLimitMiddleware(cfg.RateLimit))
		}

		if cfg.SecurityLogger.Enabled {
			handlers = append(handlers, SecurityLoggerMiddleware(cfg.SecurityLogger))
		}

		for _, handler := range handlers {
			handler(c)
			if c.IsAborted() {
				return
			}
		}

		c.Next()
	}
}

func OWASPTop10SecurityMiddleware() gin.HandlerFunc {
	config := DefaultComprehensiveSecurityConfig
	config.Fingerprint.CheckBlacklist = true
	config.Fingerprint.BlockAnomaly = true
	config.InputValidation.Enabled = true
	config.SecurityHeaders.Enabled = true
	config.RateLimit.Enabled = true
	config.AnomalyDetection.Enabled = true
	config.ReplayProtection.Enabled = true
	config.SecurityLogger.Enabled = true
	config.InputValidation.ValidateQuery = true
	config.InputValidation.ValidateForm = true
	config.InputValidation.ValidateJSON = true

	return ComprehensiveSecurityMiddleware(config)
}
