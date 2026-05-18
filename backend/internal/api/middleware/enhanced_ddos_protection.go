package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

type EnhancedDDoSConfig struct {
	Enabled                   bool
	ExcludePaths              []string
	EnableWhitelist           bool
	EnableConnectionTracking   bool
	EnableTrafficAnalysis     bool
	EnableBehavioralAnalysis  bool
	EnableGeoBlocking         bool
	BlockedCountries          []string
	RequestsPerSecond         int
	RequestsPerMinute         int
	ConnectionLimitPerIP      int
	BlacklistDurationMinutes  int
	EnableMetrics             bool
}

var defaultEnhancedDDoSConfig = EnhancedDDoSConfig{
	Enabled:                   true,
	EnableWhitelist:           false,
	EnableConnectionTracking:   true,
	EnableTrafficAnalysis:     true,
	EnableBehavioralAnalysis:  true,
	EnableGeoBlocking:         false,
	BlockedCountries:          []string{},
	RequestsPerSecond:         10,
	RequestsPerMinute:         100,
	ConnectionLimitPerIP:      10,
	BlacklistDurationMinutes:  60,
	EnableMetrics:             true,
}

var (
	enhancedDDoSService *service.EnhancedDDoSProtectionService
	enhancedDDoSOnce    sync.Once
	ddosMetrics         = &DDoSMetrics{
		totalRequests:   0,
		blockedRequests: 0,
		activeConnections: make(map[string]int),
	}
)

type DDoSMetrics struct {
	mu                 sync.RWMutex
	totalRequests      int64
	blockedRequests    int64
	anomaliesDetected  int64
	blacklistedIPs     int64
	activeConnections  map[string]int
}

type DDOSProtectionMiddleware struct {
	config    EnhancedDDoSConfig
	service   *service.EnhancedDDoSProtectionService
	metrics   *DDoSMetrics
	excludePaths []string
}

func initEnhancedDDoSService(configs ...service.EnhancedDDoSProtectionConfig) {
	enhancedDDoSOnce.Do(func() {
		if len(configs) > 0 {
			enhancedDDoSService = service.NewEnhancedDDoSProtectionService(configs[0])
		} else {
			enhancedDDoSService = service.NewEnhancedDDoSProtectionService()
		}
	})
}

func EnhancedDDoSMiddleware(configs ...EnhancedDDoSConfig) gin.HandlerFunc {
	initEnhancedDDoSService()

	cfg := defaultEnhancedDDoSConfig
	if len(configs) > 0 {
		cfg = configs[0]
	}

	middleware := &DDOSProtectionMiddleware{
		config:       cfg,
		service:      enhancedDDoSService,
		metrics:      ddosMetrics,
		excludePaths: cfg.ExcludePaths,
	}

	return middleware.Handler()
}

func (m *DDOSProtectionMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.config.Enabled {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		for _, excluded := range m.excludePaths {
			if path == excluded || strings.HasPrefix(path, excluded+"/") {
				c.Next()
				return
			}
		}

		ip := m.getClientIP(c)

		if m.config.EnableWhitelist && m.isWhitelisted(ip) {
			c.Next()
			return
		}

		if m.config.EnableGeoBlocking && m.shouldBlockCountry(c) {
			m.recordBlocked(c, ip, "geo_block")
			m.abortWithResponse(c, http.StatusForbidden, "Access denied from your region", "geo_blocked")
			return
		}

		result := m.service.CheckRequest(c.Request)

		m.metrics.mu.Lock()
		m.metrics.totalRequests++
		if !result.Allowed {
			m.metrics.blockedRequests++
		}
		if result.IPStats != nil && result.IPStats.IsAnomaly {
			m.metrics.anomaliesDetected++
		}
		m.metrics.mu.Unlock()

		if m.config.EnableMetrics {
			c.Set("ddos_result", result)
			c.Set("ddos_threat_level", result.ThreatLevel)
			c.Set("ddos_severity", result.Severity)
		}

		if !result.Allowed {
			m.recordBlocked(c, ip, result.Reason)

			if m.config.EnableConnectionTracking {
				m.service.ReleaseConnection(ip)
			}

			if result.RecommendedAction == "block" {
				m.metrics.mu.Lock()
				m.metrics.blacklistedIPs++
				m.metrics.mu.Unlock()
			}

			m.abortWithResponse(c, m.getStatusCode(result), result.Reason, result.Severity)
			return
		}

		if m.config.EnableConnectionTracking {
			defer func() {
				m.service.ReleaseConnection(ip)
			}()
		}

		c.Next()
	}
}

func (m *DDOSProtectionMiddleware) getClientIP(c *gin.Context) string {
	ip := c.GetHeader("X-Forwarded-For")
	if ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}

	ip = c.GetHeader("X-Real-IP")
	if ip != "" {
		return ip
	}

	ip = c.GetHeader("CF-Connecting-IP")
	if ip != "" {
		return ip
	}

	return c.ClientIP()
}

func (m *DDOSProtectionMiddleware) isWhitelisted(ip string) bool {
	stats := m.service.GetIPStats(ip)
	if stats == nil {
		return false
	}
	return stats.ThreatScore < -0.5
}

func (m *DDOSProtectionMiddleware) shouldBlockCountry(c *gin.Context) bool {
	country := c.GetHeader("CF-IPCountry")
	if country == "" {
		country = c.GetHeader("X-Geo-Country")
	}

	if country == "" {
		return false
	}

	for _, blocked := range m.config.BlockedCountries {
		if strings.EqualFold(country, blocked) {
			return true
		}
	}

	return false
}

func (m *DDOSProtectionMiddleware) recordBlocked(c *gin.Context, ip string, reason string) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	if count, exists := m.metrics.activeConnections[ip]; exists && count > 0 {
		m.metrics.activeConnections[ip] = count - 1
	}

	blockedKey := fmt.Sprintf("blocked:%s:%s", ip, reason)
	_ = blockedKey
}

func (m *DDOSProtectionMiddleware) getStatusCode(result *service.EnhancedDDoSCheckResult) int {
	switch result.Severity {
	case "critical":
		return http.StatusForbidden
	case "high":
		return http.StatusTooManyRequests
	case "medium":
		return http.StatusTooManyRequests
	default:
		return http.StatusTooManyRequests
	}
}

func (m *DDOSProtectionMiddleware) abortWithResponse(c *gin.Context, statusCode int, reason string, severity string) {
	retryAfter := 60
	if result, ok := c.Get("ddos_result"); ok {
		if r, ok := result.(*service.EnhancedDDoSCheckResult); ok {
			retryAfter = r.RetryAfter
		}
	}

	c.Header("Retry-After", strconv.Itoa(retryAfter))
	c.Header("X-DDoS-Reason", reason)
	c.Header("X-DDoS-Severity", severity)
	c.Header("X-DDoS-Protected", "true")

	c.AbortWithStatusJSON(statusCode, gin.H{
		"error":       "request_blocked",
		"code":        statusCode,
		"reason":      reason,
		"retry_after": retryAfter,
	})
}

func GetEnhancedDDoSService() *service.EnhancedDDoSProtectionService {
	initEnhancedDDoSService()
	return enhancedDDoSService
}

func GetDDoSMetrics() map[string]interface{} {
	ddosMetrics.mu.RLock()
	defer ddosMetrics.mu.RUnlock()

	blockedRate := float64(0)
	if ddosMetrics.totalRequests > 0 {
		blockedRate = float64(ddosMetrics.blockedRequests) / float64(ddosMetrics.totalRequests)
	}

	return map[string]interface{}{
		"total_requests":     ddosMetrics.totalRequests,
		"blocked_requests":   ddosMetrics.blockedRequests,
		"blocked_rate":       blockedRate,
		"anomalies_detected": ddosMetrics.anomaliesDetected,
		"blacklisted_ips":    ddosMetrics.blacklistedIPs,
		"active_connections": len(ddosMetrics.activeConnections),
	}
}

func ResetDDoSMetrics() {
	ddosMetrics.mu.Lock()
	defer ddosMetrics.mu.Unlock()

	ddosMetrics.totalRequests = 0
	ddosMetrics.blockedRequests = 0
	ddosMetrics.anomaliesDetected = 0
	ddosMetrics.blacklistedIPs = 0
	ddosMetrics.activeConnections = make(map[string]int)
}

type ConnectionTrackingMiddleware struct {
	enabled           bool
	maxConnections    int
	connectionTimeout int
	activeConnections map[string][]time.Time
	mu                sync.RWMutex
}

import "time"

func NewConnectionTrackingMiddleware(maxConnections int, timeoutSeconds int) *ConnectionTrackingMiddleware {
	ct := &ConnectionTrackingMiddleware{
		enabled:           true,
		maxConnections:    maxConnections,
		connectionTimeout: timeoutSeconds,
		activeConnections: make(map[string][]time.Time),
	}

	go ct.cleanupRoutine()

	return ct
}

func (ct *ConnectionTrackingMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ct.enabled {
			c.Next()
			return
		}

		ip := c.ClientIP()
		now := time.Now()

		ct.mu.Lock()
		times := ct.activeConnections[ip]
		validTimes := make([]time.Time, 0)
		cutoff := now.Add(-time.Duration(ct.connectionTimeout) * time.Second)

		for _, t := range times {
			if t.After(cutoff) {
				validTimes = append(validTimes, t)
			}
		}

		if len(validTimes) >= ct.maxConnections {
			ct.mu.Unlock()
			c.Header("Retry-After", strconv.Itoa(ct.connectionTimeout))
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":       "too_many_connections",
				"message":    "Maximum connections exceeded",
				"retry_after": ct.connectionTimeout,
			})
			return
		}

		validTimes = append(validTimes, now)
		ct.activeConnections[ip] = validTimes
		ct.mu.Unlock()

		defer func() {
			ct.mu.Lock()
			defer ct.mu.Unlock()
			if times, ok := ct.activeConnections[ip]; ok {
				newTimes := make([]time.Time, 0)
				for _, t := range times {
					if !t.Equal(now) {
						newTimes = append(newTimes, t)
					}
				}
				if len(newTimes) > 0 {
					ct.activeConnections[ip] = newTimes
				} else {
					delete(ct.activeConnections, ip)
				}
			}
		}()

		c.Next()
	}
}

func (ct *ConnectionTrackingMiddleware) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ct.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-time.Duration(ct.connectionTimeout*2) * time.Second)

		for ip, times := range ct.activeConnections {
			validTimes := make([]time.Time, 0)
			for _, t := range times {
				if t.After(cutoff) {
					validTimes = append(validTimes, t)
				}
			}
			if len(validTimes) > 0 {
				ct.activeConnections[ip] = validTimes
			} else {
				delete(ct.activeConnections, ip)
			}
		}
		ct.mu.Unlock()
	}
}

func (ct *ConnectionTrackingMiddleware) GetConnectionCount(ip string) int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return len(ct.activeConnections[ip])
}

func ConnectionTrackingMiddlewareHandler(maxConnections int, timeoutSeconds int) gin.HandlerFunc {
	ct := NewConnectionTrackingMiddleware(maxConnections, timeoutSeconds)
	return ct.Handler()
}

type TrafficAnalysisMiddleware struct {
	enabled              bool
	sampleRate           float64
	enableSizeAnalysis   bool
	enableMethodAnalysis bool
	enablePathAnalysis   bool
	enableHeaderAnalysis bool
}

var defaultTrafficAnalysisConfig = TrafficAnalysisMiddleware{
	enabled:              true,
	sampleRate:           1.0,
	enableSizeAnalysis:   true,
	enableMethodAnalysis: true,
	enablePathAnalysis:   true,
	enableHeaderAnalysis: true,
}

func TrafficAnalysisMiddlewareHandler(config ...TrafficAnalysisMiddleware) gin.HandlerFunc {
	cfg := defaultTrafficAnalysisConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		if !cfg.enabled {
			c.Next()
			return
		}

		requestInfo := map[string]interface{}{
			"ip":        c.ClientIP(),
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"size":      c.Request.ContentLength,
			"headers":   c.Request.Header,
			"timestamp": time.Now(),
		}

		c.Set("traffic_analysis", requestInfo)

		if cfg.enableMethodAnalysis {
			if !isValidHTTPMethod(c.Request.Method) {
				c.AbortWithStatusJSON(http.StatusMethodNotAllowed, gin.H{
					"error":   "invalid_method",
					"message": "HTTP method not allowed",
				})
				return
			}
		}

		if cfg.enablePathAnalysis {
			if isSuspiciousPath(c.Request.URL.Path) {
				c.Set("suspicious_path", true)
			}
		}

		if cfg.enableHeaderAnalysis {
			if missingRequiredHeaders(c.Request) {
				c.Set("missing_headers", true)
			}
		}

		c.Next()
	}
}

func isValidHTTPMethod(method string) bool {
	validMethods := map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"DELETE":  true,
		"PATCH":   true,
		"HEAD":    true,
		"OPTIONS": true,
	}
	return validMethods[method]
}

func isSuspiciousPath(path string) bool {
	suspiciousPatterns := []string{
		"..",
		"etc/passwd",
		"c:\\windows",
		".env",
		".git/",
		".htaccess",
		"wp-admin",
		"admin-console",
		"phpmyadmin",
		"xmlrpc.php",
	}

	lowerPath := strings.ToLower(path)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	return false
}

func missingRequiredHeaders(r *http.Request) bool {
	requiredHeaders := []string{"User-Agent"}
	missingCount := 0

	for _, header := range requiredHeaders {
		if r.Header.Get(header) == "" {
			missingCount++
		}
	}

	return missingCount >= len(requiredHeaders)
}

type BehavioralAnalysisMiddleware struct {
	enabled           bool
	minRequestInterval time.Duration
	maxErrorRate       float64
	trackBehavior      bool
}

var defaultBehavioralAnalysisConfig = BehavioralAnalysisMiddleware{
	enabled:           true,
	minRequestInterval: 50 * time.Millisecond,
	maxErrorRate:       0.5,
	trackBehavior:      true,
}

func BehavioralAnalysisMiddlewareHandler(config ...BehavioralAnalysisMiddleware) gin.HandlerFunc {
	cfg := defaultBehavioralAnalysisConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	type clientBehavior struct {
		mu              sync.RWMutex
		lastRequestTime time.Time
		requestCount    int
		errorCount      int
		suspiciousFlags []string
	}

	behaviors := make(map[string]*clientBehavior)
	var mu sync.RWMutex

	return func(c *gin.Context) {
		if !cfg.enabled {
			c.Next()
			return
		}

		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		behavior, exists := behaviors[ip]
		if !exists {
			behavior = &clientBehavior{
				lastRequestTime: now,
				requestCount:    0,
				errorCount:      0,
				suspiciousFlags: make([]string, 0),
			}
			behaviors[ip] = behavior
		}
		mu.Unlock()

		behavior.mu.Lock()

		interval := now.Sub(behavior.lastRequestTime)
		if interval < cfg.minRequestInterval && behavior.requestCount > 10 {
			behavior.suspiciousFlags = append(behavior.suspiciousFlags, "rapid_requests")
		}

		errorRate := float64(0)
		if behavior.requestCount > 0 {
			errorRate = float64(behavior.errorCount) / float64(behavior.requestCount)
		}

		if errorRate > cfg.maxErrorRate && behavior.requestCount > 5 {
			behavior.suspiciousFlags = append(behavior.suspiciousFlags, "high_error_rate")
		}

		behavior.requestCount++
		behavior.lastRequestTime = now
		behavior.mu.Unlock()

		if len(behavior.suspiciousFlags) > 0 {
			c.Set("behavior_flags", behavior.suspiciousFlags)
			c.Set("behavior_score", float64(len(behavior.suspiciousFlags))/10.0)
		}

		c.Next()

		if c.Writer.Status() >= 400 {
			behavior.mu.Lock()
			behavior.errorCount++
			behavior.mu.Unlock()
		}
	}
}

func SetupDDoSProtectionMiddleware(r *gin.Engine) {
	r.Use(EnhancedDDoSMiddleware())

	r.Use(ConnectionTrackingMiddlewareHandler(100, 60))

	r.Use(TrafficAnalysisMiddlewareHandler())

	r.Use(BehavioralAnalysisMiddlewareHandler())
}
