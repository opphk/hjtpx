package middleware

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type UserClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
}

type AdminClaims struct {
	AdminID  uint   `json:"admin_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type EnhancedAuthConfig struct {
	EnableRateLimit      bool
	RateLimitMaxAttempts int
	RateLimitWindow      time.Duration
	EnableIPWhitelist    bool
	AllowedIPs           []string
	EnableDeviceFingerprint bool
	EnableGeoBlock       bool
	BlockedCountries     []string
	RequireMFA           bool
	SessionTimeout       time.Duration
	MaxSessionsPerUser   int
}

var defaultEnhancedAuthConfig = EnhancedAuthConfig{
	EnableRateLimit:      true,
	RateLimitMaxAttempts: 5,
	RateLimitWindow:      15 * time.Minute,
	EnableIPWhitelist:    false,
	AllowedIPs:           []string{},
	EnableDeviceFingerprint: false,
	EnableGeoBlock:       false,
	BlockedCountries:     []string{},
	RequireMFA:           false,
	SessionTimeout:       24 * time.Hour,
	MaxSessionsPerUser:   3,
}

var (
	authRateLimitStore = &authRateLimitData{
		attempts: make(map[string]*authAttempt),
	}
	authRateLimitMu sync.RWMutex
)

type authAttempt struct {
	Count     int
	FirstTime time.Time
	Blocked   bool
	BlockEnd  time.Time
}

type authRateLimitData struct {
	attempts map[string]*authAttempt
	mu       sync.RWMutex
}

func (d *authRateLimitData) recordFailure(identifier string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	attempt, exists := d.attempts[identifier]

	if !exists || now.After(attempt.BlockEnd) {
		d.attempts[identifier] = &authAttempt{
			Count:     1,
			FirstTime: now,
			Blocked:   false,
		}
		return true
	}

	attempt.Count++
	if attempt.Count >= defaultEnhancedAuthConfig.RateLimitMaxAttempts {
		attempt.Blocked = true
		attempt.BlockEnd = now.Add(defaultEnhancedAuthConfig.RateLimitWindow)
		return false
	}

	return true
}

func (d *authRateLimitData) recordSuccess(identifier string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.attempts, identifier)
}

func (d *authRateLimitData) isBlocked(identifier string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	attempt, exists := d.attempts[identifier]
	if !exists {
		return false
	}

	if attempt.Blocked && time.Now().Before(attempt.BlockEnd) {
		return true
	}

	if time.Now().After(attempt.BlockEnd) {
		delete(d.attempts, identifier)
		return false
	}

	return false
}

func (d *authRateLimitData) cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	for identifier, attempt := range d.attempts {
		if now.After(attempt.BlockEnd) || now.After(attempt.FirstTime.Add(time.Hour)) {
			delete(d.attempts, identifier)
		}
	}
}

func init() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			authRateLimitStore.cleanup()
		}
	}()
}

func AuthMiddleware(configs ...EnhancedAuthConfig) gin.HandlerFunc {
	cfg := defaultEnhancedAuthConfig
	if len(configs) > 0 {
		cfg = configs[0]
	}

	return func(c *gin.Context) {
		identifier := getAuthIdentifier(c)

		if cfg.EnableRateLimit && authRateLimitStore.isBlocked(identifier) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many failed login attempts. Please try again later.",
				"retry_after": defaultEnhancedAuthConfig.RateLimitWindow.Seconds(),
			})
			return
		}

		if cfg.EnableIPWhitelist && len(cfg.AllowedIPs) > 0 {
			clientIP := c.ClientIP()
			if !isIPInWhitelist(clientIP, cfg.AllowedIPs) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "ip_not_allowed",
					"message": "Your IP address is not allowed to access this resource",
				})
				return
			}
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			if cfg.EnableRateLimit {
				authRateLimitStore.recordFailure(identifier)
			}
			response.Unauthorized(c)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			if cfg.EnableRateLimit {
				authRateLimitStore.recordFailure(identifier)
			}
			response.Unauthorized(c)
			c.Abort()
			return
		}

		token := parts[1]

		if redis.Client != nil {
			ctx := c.Request.Context()
			loggedOut, err := redis.Client.Get(ctx, "logout:"+token).Result()
			if err == nil && loggedOut == "1" {
				if cfg.EnableRateLimit {
					authRateLimitStore.recordFailure(identifier)
				}
				response.Unauthorized(c)
				c.Abort()
				return
			}
		}

		claims, err := jwt.ParseToken(token)
		if err != nil {
			if cfg.EnableRateLimit {
				authRateLimitStore.recordFailure(identifier)
			}
			response.Unauthorized(c)
			c.Abort()
			return
		}

		if redis.Client != nil && cfg.SessionTimeout > 0 {
			ctx := c.Request.Context()
			sessionKey := fmt.Sprintf("session:%d:%s", claims.AdminID, token[:min(16, len(token))])
			err := redis.Client.Set(ctx, sessionKey, "active", cfg.SessionTimeout).Err()
			if err != nil {
				fmt.Printf("[Auth] Warning: failed to update session: %v\n", err)
			}
		}

		if cfg.EnableRateLimit {
			authRateLimitStore.recordSuccess(identifier)
		}

		c.Set("admin_id", claims.AdminID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func AuthMiddlewareWithRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := getAuthIdentifier(c)

		if authRateLimitStore.isBlocked(identifier) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many failed login attempts. Please try again later.",
			})
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authRateLimitStore.recordFailure(identifier)
			response.Unauthorized(c)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			authRateLimitStore.recordFailure(identifier)
			response.Unauthorized(c)
			c.Abort()
			return
		}

		token := parts[1]

		if redis.Client != nil {
			ctx := c.Request.Context()
			loggedOut, err := redis.Client.Get(ctx, "logout:"+token).Result()
			if err == nil && loggedOut == "1" {
				authRateLimitStore.recordFailure(identifier)
				response.Unauthorized(c)
				c.Abort()
				return
			}
		}

		claims, err := jwt.ParseToken(token)
		if err != nil {
			authRateLimitStore.recordFailure(identifier)
			response.Unauthorized(c)
			c.Abort()
			return
		}

		if len(allowedRoles) > 0 {
			hasRole := false
			for _, role := range allowedRoles {
				if role == claims.Username {
					hasRole = true
					break
				}
			}
			if !hasRole {
				authRateLimitStore.recordFailure(identifier)
				response.Forbidden(c)
				c.Abort()
				return
			}
		}

		authRateLimitStore.recordSuccess(identifier)

		c.Set("admin_id", claims.AdminID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func UserAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := getAuthIdentifier(c)

		if authRateLimitStore.isBlocked(identifier) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many failed login attempts. Please try again later.",
			})
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authRateLimitStore.recordFailure(identifier)
			response.Unauthorized(c)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			authRateLimitStore.recordFailure(identifier)
			response.Unauthorized(c)
			c.Abort()
			return
		}

		token := parts[1]

		if redis.Client != nil {
			ctx := c.Request.Context()
			loggedOut, err := redis.Client.Get(ctx, "user_logout:"+token).Result()
			if err == nil && loggedOut == "1" {
				authRateLimitStore.recordFailure(identifier)
				response.Unauthorized(c)
				c.Abort()
				return
			}
		}

		claims, err := jwt.ParseUserToken(token)
		if err != nil {
			authRateLimitStore.recordFailure(identifier)
			response.Unauthorized(c)
			c.Abort()
			return
		}

		authRateLimitStore.recordSuccess(identifier)

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func getAuthIdentifier(c *gin.Context) string {
	components := []string{c.ClientIP()}

	if ua := c.GetHeader("User-Agent"); ua != "" {
		components = append(components, ua)
	}

	if sessionID := c.GetHeader("X-Session-ID"); sessionID != "" {
		components = append(components, sessionID)
	}

	return strings.Join(components, "|")
}

func isIPInWhitelist(ip string, whitelist []string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range whitelist {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			if ip == cidr {
				return true
			}
			continue
		}
		if ipnet.Contains(parsedIP) {
			return true
		}
	}

	return false
}

func GetUserID(c *gin.Context) uint {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(uint)
	}
	if adminID, exists := c.Get("admin_id"); exists {
		return adminID.(uint)
	}
	return 0
}

func GetUsername(c *gin.Context) string {
	if username, exists := c.Get("username"); exists {
		return username.(string)
	}
	return ""
}

func GetAdminID(c *gin.Context) uint {
	if adminID, exists := c.Get("admin_id"); exists {
		return adminID.(uint)
	}
	return 0
}

func IsSuperAdmin(c *gin.Context) bool {
	if isSuper, exists := c.Get("is_super_admin"); exists {
		return isSuper.(bool)
	}
	return false
}

func GetRole(c *gin.Context) string {
	if role, exists := c.Get("role"); exists {
		return role.(string)
	}
	return ""
}

type APIKeyAuthConfig struct {
	HeaderName    string
	QueryParam    string
	RedisPrefix   string
	MaxPerIP      int
	RequireOrigin bool
	OriginWhitelist []string
}

var defaultAPIKeyAuthConfig = APIKeyAuthConfig{
	HeaderName:    "X-API-Key",
	QueryParam:    "api_key",
	RedisPrefix:   "apikey:",
	MaxPerIP:      10,
	RequireOrigin: false,
	OriginWhitelist: []string{},
}

func APIKeyAuthMiddleware(configs ...APIKeyAuthConfig) gin.HandlerFunc {
	cfg := defaultAPIKeyAuthConfig
	if len(configs) > 0 {
		cfg = configs[0]
	}

	return func(c *gin.Context) {
		var apiKey string

		if header := c.GetHeader(cfg.HeaderName); header != "" {
			apiKey = header
		} else if query := c.Query(cfg.QueryParam); query != "" {
			apiKey = query
		}

		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "missing_api_key",
				"message": "API key is required",
			})
			return
		}

		if len(apiKey) < 32 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_api_key",
				"message": "Invalid API key format",
			})
			return
		}

		if cfg.RequireOrigin {
			origin := c.GetHeader("Origin")
			allowed := false
			for _, allowedOrigin := range cfg.OriginWhitelist {
				if subtle.ConstantTimeCompare([]byte(origin), []byte(allowedOrigin)) == 1 {
					allowed = true
					break
				}
			}
			if !allowed {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "origin_not_allowed",
					"message": "Origin not allowed",
				})
				return
			}
		}

		if redis.Client != nil {
			ctx := c.Request.Context()
			key := cfg.RedisPrefix + apiKey

			exists, err := redis.Client.Exists(ctx, key).Result()
			if err != nil || exists == 0 {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error":   "invalid_api_key",
					"message": "API key not found or revoked",
				})
				return
			}

			hashedKey := fmt.Sprintf("%x", apiKey)
			usageKey := fmt.Sprintf("apikey_usage:%s:%s", hashedKey[:16], c.ClientIP())
			err = redis.Client.Incr(ctx, usageKey).Err()
			if err == nil {
				count, _ := redis.Client.Get(ctx, usageKey).Int()
				if count > cfg.MaxPerIP {
					c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
						"error":   "rate_limit_exceeded",
						"message": "API key usage limit exceeded",
					})
					return
				}
				redis.Client.Expire(ctx, usageKey, 24*time.Hour)
			}
		}

		c.Set("api_key", apiKey)
		c.Set("api_key_hash", fmt.Sprintf("%x", apiKey)[:16])

		c.Next()
	}
}
