package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CSRFConfig CSRF防护配置
type CSRFConfig struct {
	Secret          string        // 密钥
	TokenLength     int           // Token长度
	TokenExpiry     time.Duration // Token过期时间
	CookieName      string        // Cookie名称
	HeaderName      string        // 请求头名称
	FormFieldName   string        // 表单字段名称
	CookieSecure    bool          // Cookie是否安全
	CookieHTTPOnly  bool          // Cookie是否仅HTTP
	CookieSameSite  http.SameSite // Cookie SameSite策略
	ExcludePaths    []string      // 排除的路径
}

// DefaultCSRFConfig 默认CSRF配置
var DefaultCSRFConfig = &CSRFConfig{
	TokenLength:    32,
	TokenExpiry:    24 * time.Hour,
	CookieName:     "csrf_token",
	HeaderName:     "X-CSRF-Token",
	FormFieldName:  "_csrf",
	CookieSecure:   true,
	CookieHTTPOnly: false,
	CookieSameSite: http.SameSiteStrictMode,
	ExcludePaths:   []string{"/api/health", "/api/healthz", "/metrics"},
}

// CSRFProtection CSRF防护器
type CSRFProtection struct {
	config       *CSRFConfig
	tokenStore   map[string]*csrfToken
	mu           sync.RWMutex
	cookieDomain string
}

// csrfToken CSRF Token
type csrfToken struct {
	Token     string
	SessionID string
	CreatedAt time.Time
	ExpiresAt time.Time
	Used      bool
}

// NewCSRFProtection 创建CSRF防护器
func NewCSRFProtection(config *CSRFConfig) *CSRFProtection {
	if config == nil {
		config = DefaultCSRFConfig
	}

	if config.Secret == "" {
		config.Secret = "csrf-secret-key-change-in-production"
	}

	if config.TokenLength == 0 {
		config.TokenLength = 32
	}

	csrf := &CSRFProtection{
		config:     config,
		tokenStore: make(map[string]*csrfToken),
	}

	go csrf.cleanupExpiredTokens()

	return csrf
}

// SetCookieDomain 设置Cookie域
func (cs *CSRFProtection) SetCookieDomain(domain string) {
	cs.cookieDomain = domain
}

// GenerateToken 生成CSRF Token
func (cs *CSRFProtection) GenerateToken(sessionID string) (string, error) {
	tokenBytes := make([]byte, cs.config.TokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("生成随机token失败: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	now := time.Now()
	csrfToken := &csrfToken{
		Token:     token,
		SessionID: sessionID,
		CreatedAt: now,
		ExpiresAt: now.Add(cs.config.TokenExpiry),
		Used:      false,
	}

	cs.mu.Lock()
	cs.tokenStore[token] = csrfToken
	cs.mu.Unlock()

	signedToken := cs.signToken(token, sessionID)

	return signedToken, nil
}

// ValidateToken 验证CSRF Token
func (cs *CSRFProtection) ValidateToken(sessionID, token string) bool {
	if token == "" {
		return false
	}

	originalToken, valid := cs.unsignToken(token, sessionID)
	if !valid {
		return false
	}

	cs.mu.RLock()
	csrfToken, exists := cs.tokenStore[originalToken]
	cs.mu.RUnlock()

	if !exists {
		return false
	}

	if csrfToken.SessionID != sessionID {
		return false
	}

	if time.Now().After(csrfToken.ExpiresAt) {
		cs.mu.Lock()
		delete(cs.tokenStore, originalToken)
		cs.mu.Unlock()
		return false
	}

	if csrfToken.Used {
		return false
	}

	cs.mu.Lock()
	csrfToken.Used = true
	cs.mu.Unlock()

	return true
}

// RegenerateToken 重新生成Token
func (cs *CSRFProtection) RegenerateToken(sessionID, oldToken string) (string, error) {
	cs.mu.Lock()
	if oldToken != "" {
		originalToken, valid := cs.unsignToken(oldToken, sessionID)
		if valid {
			delete(cs.tokenStore, originalToken)
		}
	}
	cs.mu.Unlock()

	return cs.GenerateToken(sessionID)
}

// signToken 对Token进行签名
func (cs *CSRFProtection) signToken(token, sessionID string) string {
	data := fmt.Sprintf("%s:%s", token, sessionID)
	h := hmac.New(sha256.New, []byte(cs.config.Secret))
	h.Write([]byte(data))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s.%s", token, signature)
}

// unsignToken 验证并提取Token
func (cs *CSRFProtection) unsignToken(signedToken, sessionID string) (string, bool) {
	parts := strings.SplitN(signedToken, ".", 2)
	if len(parts) != 2 {
		return "", false
	}

	token := parts[0]
	signature := parts[1]

	data := fmt.Sprintf("%s:%s", token, sessionID)
	h := hmac.New(sha256.New, []byte(cs.config.Secret))
	h.Write([]byte(data))
	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) != 1 {
		return "", false
	}

	return token, true
}

// cleanupExpiredTokens 清理过期Token
func (cs *CSRFProtection) cleanupExpiredTokens() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cs.mu.Lock()
		now := time.Now()
		for token, csrfToken := range cs.tokenStore {
			if now.After(csrfToken.ExpiresAt) {
				delete(cs.tokenStore, token)
			}
		}
		cs.mu.Unlock()
	}
}

// getSessionID 获取会话ID
func (cs *CSRFProtection) getSessionID(c *gin.Context) string {
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID != "" {
		return sessionID
	}

	if session, exists := c.Get("session_id"); exists {
		return fmt.Sprintf("%v", session)
	}

	if cookie, err := c.Cookie("session_id"); err == nil && cookie != "" {
		return cookie
	}

	newSessionID := uuid.New().String()
	c.SetCookie("session_id", newSessionID, int(cs.config.TokenExpiry.Seconds()), "/", cs.cookieDomain, cs.config.CookieSecure, cs.config.CookieHTTPOnly)

	return newSessionID
}

// isExcludedPath 检查路径是否排除
func (cs *CSRFProtection) isExcludedPath(path string) bool {
	for _, exclude := range cs.config.ExcludePaths {
		if strings.HasPrefix(path, exclude) {
			return true
		}
	}
	return false
}

// Middleware 返回CSRF防护中间件
func (cs *CSRFProtection) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if cs.isExcludedPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			cs.handleTokenGeneration(c)
			c.Next()
			return
		}

		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch || c.Request.Method == http.MethodDelete {
			if !cs.validateRequest(c) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "CSRF验证失败",
					"error":   "invalid or missing CSRF token",
				})
				return
			}
		}

		c.Next()
	}
}

// handleTokenGeneration 处理Token生成
func (cs *CSRFProtection) handleTokenGeneration(c *gin.Context) {
	sessionID := cs.getSessionID(c)

	token, err := cs.GenerateToken(sessionID)
	if err != nil {
		return
	}

	c.SetCookie(
		cs.config.CookieName,
		token,
		int(cs.config.TokenExpiry.Seconds()),
		"/",
		cs.cookieDomain,
		cs.config.CookieSecure,
		cs.config.CookieHTTPOnly,
	)

	c.Header(cs.config.HeaderName, token)

	c.Set("csrf_token", token)
}

// validateRequest 验证请求
func (cs *CSRFProtection) validateRequest(c *gin.Context) bool {
	sessionID := cs.getSessionID(c)

	token := c.GetHeader(cs.config.HeaderName)
	if token == "" {
		token = c.Request.FormValue(cs.config.FormFieldName)
	}
	if token == "" {
		token = c.Query(cs.config.FormFieldName)
	}

	if token == "" {
		referer := c.GetHeader("Referer")
		origin := c.GetHeader("Origin")

		if referer == "" && origin == "" {
			return false
		}

		if referer != "" {
			allowedOrigin := cs.validateReferer(c, referer)
			if !allowedOrigin {
				return false
			}
		}

		if origin != "" {
			allowedOrigin := cs.validateOrigin(c, origin)
			if !allowedOrigin {
				return false
			}
		}
	}

	return cs.ValidateToken(sessionID, token)
}

// validateReferer 验证Referer
func (cs *CSRFProtection) validateReferer(c *gin.Context, referer string) bool {
	if referer == "" {
		return false
	}

	origin := c.Request.URL.Scheme + "://" + c.Request.URL.Host
	return strings.HasPrefix(referer, origin)
}

// validateOrigin 验证Origin
func (cs *CSRFProtection) validateOrigin(c *gin.Context, origin string) bool {
	if origin == "" {
		return false
	}

	expectedOrigin := c.Request.URL.Scheme + "://" + c.Request.URL.Host
	return origin == expectedOrigin
}

// SetExcludedPaths 设置排除路径
func (cs *CSRFProtection) SetExcludedPaths(paths []string) {
	cs.config.ExcludePaths = paths
}

// AddExcludedPath 添加排除路径
func (cs *CSRFProtection) AddExcludedPath(path string) {
	cs.config.ExcludePaths = append(cs.config.ExcludePaths, path)
}

// GlobalCSRFProtection 全局CSRF防护器
var GlobalCSRFProtection *CSRFProtection

// InitGlobalCSRFProtection 初始化全局CSRF防护器
func InitGlobalCSRFProtection(config *CSRFConfig) {
	GlobalCSRFProtection = NewCSRFProtection(config)
}

// CSRFProtectionMiddleware CSRF防护中间件包装器
func CSRFProtectionMiddleware(config *CSRFConfig) gin.HandlerFunc {
	csrf := NewCSRFProtection(config)
	return csrf.Middleware()
}

// RequireCSRFToken 要求CSRF Token的中间件
func RequireCSRFToken() gin.HandlerFunc {
	if GlobalCSRFProtection == nil {
		GlobalCSRFProtection = NewCSRFProtection(DefaultCSRFConfig)
	}

	return func(c *gin.Context) {
		sessionID := GlobalCSRFProtection.getSessionID(c)
		token := c.GetHeader(DefaultCSRFConfig.HeaderName)
		if token == "" {
			token = c.Request.FormValue(DefaultCSRFConfig.FormFieldName)
		}

		if !GlobalCSRFProtection.ValidateToken(sessionID, token) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "CSRF验证失败",
				"error":   "invalid or missing CSRF token",
			})
			return
		}

		c.Next()
	}
}

// GetCSRFToken 获取CSRF Token的处理函数
func GetCSRFToken() gin.HandlerFunc {
	if GlobalCSRFProtection == nil {
		GlobalCSRFProtection = NewCSRFProtection(DefaultCSRFConfig)
	}

	return func(c *gin.Context) {
		sessionID := GlobalCSRFProtection.getSessionID(c)
		token, err := GlobalCSRFProtection.GenerateToken(sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "生成CSRF token失败",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": gin.H{
				"token": token,
				"name":  DefaultCSRFConfig.HeaderName,
				"header": DefaultCSRFConfig.HeaderName,
				"form_field": DefaultCSRFConfig.FormFieldName,
			},
		})
	}
}

// DoubleSubmitCookie 双重提交Cookie验证
func (cs *CSRFProtection) DoubleSubmitCookie(c *gin.Context) bool {
	cookieToken := c.GetHeader(cs.config.HeaderName)
	if cookieToken == "" {
		if cookie, err := c.Cookie(cs.config.CookieName); err == nil {
			cookieToken = cookie
		}
	}

	formToken := c.Request.FormValue(cs.config.FormFieldName)
	if formToken == "" {
		formToken = c.Query(cs.config.FormFieldName)
	}

	if cookieToken == "" || formToken == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(cookieToken), []byte(formToken)) == 1
}

// DoubleSubmitCookieMiddleware 双重提交Cookie中间件
func DoubleSubmitCookieMiddleware() gin.HandlerFunc {
	csrf := NewCSRFProtection(DefaultCSRFConfig)

	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			csrf.handleTokenGeneration(c)
			c.Next()
			return
		}

		if !csrf.DoubleSubmitCookie(c) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "CSRF验证失败",
				"error":   "cookie and form token mismatch",
			})
			return
		}

		c.Next()
	}
}
