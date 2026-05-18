package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/redis"
)

type EnhancedCSRFSecurityConfig struct {
	TokenLength          int
	TokenExpiration      time.Duration
	DoubleSubmitCookie   bool
	RequireEncryption    bool
	RotateOnVerification bool
	EnableSessionBinding bool
	SameSiteCookie       string
	SecureCookie         bool
	HttpOnlyCookie       bool
}

type EnhancedCSRFSecurity struct {
	config     EnhancedCSRFSecurityConfig
	tokenStore *CSRFTokenStore
	secretKey  []byte
}

type CSRFTokenStore interface {
	Store(sessionID, token, hashedToken string, expiration time.Time) error
	Verify(sessionID, hashedToken string) (bool, error)
	Delete(sessionID string) error
	Get(sessionID string) (string, error)
}

type CSRFRedisStore struct {
	expiration time.Duration
}

type CSRFMemoryStore struct {
	tokens    map[string]map[string]time.Time
	mu        sync.RWMutex
	cleanupCh chan struct{}
}

type CSRFToken struct {
	Token       string
	HashedToken string
	SessionID   string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Used        bool
}

var defaultCSRFSecurityConfig = EnhancedCSRFSecurityConfig{
	TokenLength:          32,
	TokenExpiration:      1 * time.Hour,
	DoubleSubmitCookie:   true,
	RequireEncryption:    false,
	RotateOnVerification: true,
	EnableSessionBinding: true,
	SameSiteCookie:       "Strict",
	SecureCookie:         true,
	HttpOnlyCookie:       false,
}

func NewCSRFSecurity(config *EnhancedCSRFSecurityConfig) *EnhancedCSRFSecurity {
	cfg := defaultCSRFSecurityConfig
	if config != nil {
		cfg = *config
	}

	csrf := &EnhancedCSRFSecurity{
		config:    cfg,
		secretKey: []byte("csrf-secret-key-change-in-production"),
	}

	if redis.Client != nil {
		csrf.tokenStore = &CSRFRedisStore{expiration: cfg.TokenExpiration}
	} else {
		csrf.tokenStore = NewCSRFMemoryStoreEx(cfg.TokenExpiration)
	}

	return csrf
}

func NewCSRFMemoryStoreEx(expiration time.Duration) *CSRFMemoryStore {
	store := &CSRFMemoryStore{
		tokens:    make(map[string]map[string]time.Time),
		cleanupCh: make(chan struct{}),
	}

	go store.cleanupRoutine(expiration)

	return store
}

func (s *CSRFMemoryStore) cleanupRoutine(expiration time.Duration) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup(expiration)
		case <-s.cleanupCh:
			return
		}
	}
}

func (s *CSRFMemoryStore) cleanup(expiration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for sessionID, tokenMap := range s.tokens {
		for token, expiry := range tokenMap {
			if now.After(expiry) {
				delete(tokenMap, token)
			}
		}
		if len(tokenMap) == 0 {
			delete(s.tokens, sessionID)
		}
	}
}

func (s *CSRFMemoryStore) Store(sessionID, token, hashedToken string, expiration time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tokens[sessionID] == nil {
		s.tokens[sessionID] = make(map[string]time.Time)
	}

	hashedKey := hashCSRFToken(token)
	s.tokens[sessionID][hashedKey] = expiration

	return nil
}

func (s *CSRFMemoryStore) Verify(sessionID, hashedToken string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if tokenMap, ok := s.tokens[sessionID]; ok {
		if expiry, exists := tokenMap[hashedToken]; exists {
			if time.Now().Before(expiry) {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *CSRFMemoryStore) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tokens, sessionID)
	return nil
}

func (s *CSRFMemoryStore) Get(sessionID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if tokenMap, ok := s.tokens[sessionID]; ok {
		for token := range tokenMap {
			return token, nil
		}
	}

	return "", fmt.Errorf("token not found")
}

func (s *CSRFRedisStore) Store(sessionID, token, hashedToken string, expiration time.Time) error {
	if redis.Client == nil {
		return fmt.Errorf("redis client not available")
	}

	ctx := context.Background()
	key := fmt.Sprintf("csrf:token:%s", sessionID)

	data := CSRFToken{
		Token:       token,
		HashedToken: hashedToken,
		SessionID:   sessionID,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiration,
		Used:        false,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return redis.Client.Set(ctx, key, jsonData, s.expiration).Err()
}

func (s *CSRFRedisStore) Verify(sessionID, hashedToken string) (bool, error) {
	if redis.Client == nil {
		return false, fmt.Errorf("redis client not available")
	}

	ctx := context.Background()
	key := fmt.Sprintf("csrf:token:%s", sessionID)

	data, err := redis.Client.Get(ctx, key).Result()
	if err != nil {
		return false, err
	}

	var token CSRFToken
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return false, err
	}

	if token.HashedToken != hashedToken {
		return false, nil
	}

	if time.Now().After(token.ExpiresAt) {
		return false, fmt.Errorf("token expired")
	}

	if token.Used {
		return false, fmt.Errorf("token already used")
	}

	return true, nil
}

func (s *CSRFRedisStore) Delete(sessionID string) error {
	if redis.Client == nil {
		return nil
	}

	ctx := context.Background()
	key := fmt.Sprintf("csrf:token:%s", sessionID)
	return redis.Client.Del(ctx, key).Err()
}

func (s *CSRFRedisStore) Get(sessionID string) (string, error) {
	if redis.Client == nil {
		return "", fmt.Errorf("redis client not available")
	}

	ctx := context.Background()
	key := fmt.Sprintf("csrf:token:%s", sessionID)

	data, err := redis.Client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}

	var token CSRFToken
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return "", err
	}

	return token.Token, nil
}

func hashCSRFToken(token string) string {
	h := hmac.New(sha256.New, []byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *EnhancedCSRFSecurity) GenerateToken() (string, error) {
	bytes := make([]byte, s.config.TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	token := base64.URLEncoding.EncodeToString(bytes)
	hashedToken := hashCSRFToken(token)

	return token, nil
}

func (s *EnhancedCSRFSecurity) StoreToken(sessionID, token string) error {
	hashedToken := hashCSRFToken(token)
	expiration := time.Now().Add(s.config.TokenExpiration)
	return s.tokenStore.Store(sessionID, token, hashedToken, expiration)
}

func (s *EnhancedCSRFSecurity) VerifyToken(sessionID, token string) (bool, error) {
	if token == "" {
		return false, fmt.Errorf("token is required")
	}

	if len(token) < 16 {
		return false, fmt.Errorf("token too short")
	}

	hashedToken := hashCSRFToken(token)
	valid, err := s.tokenStore.Verify(sessionID, hashedToken)
	if err != nil {
		return false, err
	}

	if valid && s.config.RotateOnVerification {
		_ = s.tokenStore.Delete(sessionID)
	}

	return valid, nil
}

func (s *EnhancedCSRFSecurity) ValidateRequest(sessionID, token, cookieToken string) (bool, error) {
	if s.config.DoubleSubmitCookie && cookieToken != "" {
		if token != cookieToken {
			return false, fmt.Errorf("double submit validation failed")
		}
	}

	return s.VerifyToken(sessionID, token)
}

func (s *EnhancedCSRFSecurity) GetConfig() EnhancedCSRFSecurityConfig {
	return s.config
}

type XSSSecurityConfig struct {
	EnableHTMLSanitization    bool
	EnableAttributeFiltering  bool
	EnableURLValidation       bool
	EnableJSRemoval           bool
	AllowedTags               []string
	AllowedAttrs              []string
	AllowedSchemes            []string
	MaxInputLength            int
	EnableCSRFProtection      bool
	EnableClickjackingProtection bool
}

type XSSSecurity struct {
	config XSSSecurityConfig
}

var defaultXSSSecurityConfig = XSSSecurityConfig{
	EnableHTMLSanitization:    true,
	EnableAttributeFiltering:  true,
	EnableURLValidation:       true,
	EnableJSRemoval:           true,
	AllowedTags:              []string{"p", "br", "b", "i", "em", "strong", "a", "ul", "ol", "li", "h1", "h2", "h3", "h4", "h5", "h6"},
	AllowedAttrs:             []string{"href", "title", "class"},
	AllowedSchemes:           []string{"http", "https", "mailto"},
	MaxInputLength:           10000,
	EnableCSRFProtection:      true,
	EnableClickjackingProtection: true,
}

var (
	xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)<script[^>]*>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)on\w+\s*=`),
		regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
		regexp.MustCompile(`(?i)<object[^>]*>.*?</object>`),
		regexp.MustCompile(`(?i)<embed[^>]*>`),
		regexp.MustCompile(`(?i)<applet[^>]*>.*?</applet>`),
		regexp.MustCompile(`(?i)expression\s*\(`),
		regexp.MustCompile(`(?i)url\s*\(`),
		regexp.MustCompile(`(?i)data\s*:`),
		regexp.MustCompile(`(?i)<link[^>]*>`),
		regexp.MustCompile(`(?i)<meta[^>]*>`),
		regexp.MustCompile(`(?i)<base[^>]*>`),
		regexp.MustCompile(`(?i)<form[^>]*>`),
		regexp.MustCompile(`(?i)<input[^>]*>`),
		regexp.MustCompile(`(?i)<button[^>]*>`),
		regexp.MustCompile(`(?i)<textarea[^>]*>`),
		regexp.MustCompile(`(?i)<select[^>]*>`),
		regexp.MustCompile(`(?i)<option[^>]*>`),
		regexp.MustCompile(`(?i)<svg[^>]*>.*?</svg>`),
		regexp.MustCompile(`(?i)<math[^>]*>.*?</math>`),
		regexp.MustCompile(`(?i)<xml[^>]*>`),
		regexp.MustCompile(`(?i)\$\{.*?\}`),
		regexp.MustCompile(`(?i)\{.*?\}`),
	}

	htmlTagPattern = regexp.MustCompile(`<[^>]+>`)
	htmlAttrPattern = regexp.MustCompile(`\s+([a-zA-Z-]+)(\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>]+))?`)
	scriptProtocolPattern = regexp.MustCompile(`(?i)\b(javascript|vbscript|livescript)\s*:`)
	eventHandlerPattern = regexp.MustCompile(`(?i)\bon(mouse|load|click|dblclick|error|focus|blur|change|reset|submit|key|drag|drop)\s*=`)
)

func NewXSSSecurity(config *XSSSecurityConfig) *XSSSecurity {
	cfg := defaultXSSSecurityConfig
	if config != nil {
		cfg = *config
	}
	return &XSSSecurity{config: cfg}
}

func (x *XSSSecurity) SanitizeInput(input string) string {
	if input == "" {
		return ""
	}

	if len(input) > x.config.MaxInputLength {
		input = input[:x.config.MaxInputLength]
	}

	for _, pattern := range xssPatterns {
		input = pattern.ReplaceAllString(input, "")
	}

	input = x.removeEventHandlers(input)

	input = x.validateURLs(input)

	if x.config.EnableHTMLSanitization {
		input = x.sanitizeHTML(input)
	}

	return input
}

func (x *XSSSecurity) removeEventHandlers(input string) string {
	input = eventHandlerPattern.ReplaceAllString(input, "")
	return input
}

func (x *XSSSecurity) validateURLs(input string) string {
	if !x.config.EnableURLValidation {
		return input
	}

	urlPattern := regexp.MustCompile(`(?i)(href|src|action)\s*=\s*["']([^"']+)["']`)
	matches := urlPattern.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		if len(match) == 3 {
			original := match[0]
			attrName := match[1]
			urlValue := match[2]

			if !x.isAllowedURL(urlValue) {
				replacement := attrName + `="javascript:void(0)"` + ` data-blocked="true"`
				input = strings.Replace(input, original, replacement, 1)
			}
		}
	}

	return input
}

func (x *XSSSecurity) isAllowedURL(urlStr string) bool {
	if urlStr == "" || urlStr == "#" || strings.HasPrefix(urlStr, "javascript:void(0)") {
		return true
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	scheme := strings.ToLower(u.Scheme)
	for _, allowed := range x.config.AllowedSchemes {
		if scheme == strings.ToLower(allowed) {
			return true
		}
	}

	return false
}

func (x *XSSSecurity) sanitizeHTML(input string) string {
	if !x.config.EnableHTMLSanitization {
		return html.EscapeString(input)
	}

	var result strings.Builder
	var inTag bool

	for i := 0; i < len(input); i++ {
		c := input[i]

		if c == '<' {
			inTag = true
			result.WriteByte(c)
			continue
		}

		if c == '>' {
			inTag = false
			result.WriteByte(c)
			continue
		}

		if !inTag {
			result.WriteByte(c)
			continue
		}

		result.WriteByte(c)
	}

	output := result.String()

	output = htmlTagPattern.ReplaceAllStringFunc(output, func(tag string) string {
		return x.validateTag(tag)
	})

	return output
}

func (x *XSSSecurity) validateTag(tag string) string {
	tagLower := strings.ToLower(tag)

	if strings.HasPrefix(tagLower, "</") {
		tagName := strings.TrimPrefix(tagLower, "</")
		tagName = strings.TrimSuffix(tagName, ">")
		tagName = strings.TrimSpace(tagName)

		for _, allowed := range x.config.AllowedTags {
			if strings.EqualFold(tagName, allowed) {
				return tag
			}
		}
		return ""
	}

	if strings.HasPrefix(tagLower, "<!") || strings.HasPrefix(tagLower, "<?") {
		return ""
	}

	tagName := tag
	if idx := strings.IndexAny(tag, " \n\r\t"); idx > 0 {
		tagName = tag[:idx]
	}
	tagName = strings.TrimPrefix(strings.ToLower(tagName), "<")

	for _, allowed := range x.config.AllowedTags {
		if strings.EqualFold(tagName, allowed) {
			return x.validateAttributes(tag)
		}
	}

	return ""
}

func (x *XSSSecurity) validateAttributes(tag string) string {
	if !x.config.EnableAttributeFiltering {
		return tag
	}

	var result strings.Builder
	result.WriteString(strings.SplitN(tag, " ", 2)[0])
	result.WriteString(" ")

	matches := htmlAttrPattern.FindAllStringSubmatch(tag, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		attrName := match[1]
		attrValue := match[2]

		isAllowed := false
		for _, allowed := range x.config.AllowedAttrs {
			if strings.EqualFold(attrName, allowed) {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			continue
		}

		if attrValue != "" {
			attrValue = x.sanitizeAttributeValue(attrValue)
		}

		result.WriteString(attrName)
		if attrValue != "" {
			result.WriteString(`="` + attrValue + `"`)
		}
		result.WriteString(" ")
	}

	output := result.String()
	output = strings.TrimRight(output, " ")

	if !strings.HasSuffix(output, "/>") && !strings.Contains(output, "</") {
		output = strings.TrimSuffix(output, ">") + ">"
	}

	return output
}

func (x *XSSSecurity) sanitizeAttributeValue(value string) string {
	value = strings.Trim(value, `"'`)

	value = scriptProtocolPattern.ReplaceAllString(value, "")

	value = eventHandlerPattern.ReplaceAllString(value, "")

	value = template.HTMLEscape(value)

	return value
}

func (x *XSSSecurity) SanitizeHTML(input string) string {
	return x.sanitizeHTML(input)
}

func (x *XSSSecurity) SanitizeURL(input string) string {
	if input == "" {
		return ""
	}

	u, err := url.Parse(input)
	if err != nil {
		return ""
	}

	if !x.isAllowedURL(u.String()) {
		return "javascript:void(0)"
	}

	return u.String()
}

func (x *XSSSecurity) GetConfig() XSSSecurityConfig {
	return x.config
}

func (x *XSSSecurity) UpdateConfig(config XSSSecurityConfig) {
	x.config = config
}

func (x *XSSSecurity) DetectXSS(input string) (bool, string) {
	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return true, pattern.String()
		}
	}

	if eventHandlerPattern.MatchString(input) {
		return true, "event handler detected"
	}

	if scriptProtocolPattern.MatchString(input) {
		return true, "script protocol detected"
	}

	return false, ""
}

type ContentSecurityPolicyConfig struct {
	DefaultSrc         []string
	ScriptSrc          []string
	StyleSrc           []string
	ImgSrc             []string
	FontSrc            []string
	ConnectSrc         []string
	FrameSrc           []string
	ObjectSrc          []string
	MediaSrc           []string
	WorkerSrc          []string
	ManifestSrc        []string
	BaseURI            []string
	FormAction         []string
	FrameAncestors     []string
	ReportURI          string
	ReportOnly         bool
	EnableNonce        bool
	BlockAllMixedContent bool
	UpgradeInsecureRequests bool
}

var defaultCSPConfig = ContentSecurityPolicyConfig{
	DefaultSrc:         []string{"'self'"},
	ScriptSrc:          []string{"'self'"},
	StyleSrc:           []string{"'self'", "'unsafe-inline'"},
	ImgSrc:             []string{"'self'", "data:", "https:"},
	FontSrc:            []string{"'self'"},
	ConnectSrc:         []string{"'self'"},
	FrameSrc:           []string{"'none'"},
	ObjectSrc:          []string{"'none'"},
	MediaSrc:           []string{"'self'"},
	WorkerSrc:          []string{"'self'"},
	ManifestSrc:        []string{"'self'"},
	BaseURI:            []string{"'self'"},
	FormAction:         []string{"'self'"},
	FrameAncestors:     []string{"'none'"},
	BlockAllMixedContent: true,
	UpgradeInsecureRequests: true,
}

func (c *ContentSecurityPolicyConfig) BuildPolicy() string {
	var directives []string

	if len(c.DefaultSrc) > 0 {
		directives = append(directives, fmt.Sprintf("default-src %s", strings.Join(c.DefaultSrc, " ")))
	}

	if len(c.ScriptSrc) > 0 {
		directives = append(directives, fmt.Sprintf("script-src %s", strings.Join(c.ScriptSrc, " ")))
	}

	if len(c.StyleSrc) > 0 {
		directives = append(directives, fmt.Sprintf("style-src %s", strings.Join(c.StyleSrc, " ")))
	}

	if len(c.ImgSrc) > 0 {
		directives = append(directives, fmt.Sprintf("img-src %s", strings.Join(c.ImgSrc, " ")))
	}

	if len(c.FontSrc) > 0 {
		directives = append(directives, fmt.Sprintf("font-src %s", strings.Join(c.FontSrc, " ")))
	}

	if len(c.ConnectSrc) > 0 {
		directives = append(directives, fmt.Sprintf("connect-src %s", strings.Join(c.ConnectSrc, " ")))
	}

	if len(c.FrameSrc) > 0 {
		directives = append(directives, fmt.Sprintf("frame-src %s", strings.Join(c.FrameSrc, " ")))
	}

	if len(c.ObjectSrc) > 0 {
		directives = append(directives, fmt.Sprintf("object-src %s", strings.Join(c.ObjectSrc, " ")))
	}

	if len(c.MediaSrc) > 0 {
		directives = append(directives, fmt.Sprintf("media-src %s", strings.Join(c.MediaSrc, " ")))
	}

	if len(c.WorkerSrc) > 0 {
		directives = append(directives, fmt.Sprintf("worker-src %s", strings.Join(c.WorkerSrc, " ")))
	}

	if len(c.ManifestSrc) > 0 {
		directives = append(directives, fmt.Sprintf("manifest-src %s", strings.Join(c.ManifestSrc, " ")))
	}

	if len(c.BaseURI) > 0 {
		directives = append(directives, fmt.Sprintf("base-uri %s", strings.Join(c.BaseURI, " ")))
	}

	if len(c.FormAction) > 0 {
		directives = append(directives, fmt.Sprintf("form-action %s", strings.Join(c.FormAction, " ")))
	}

	if len(c.FrameAncestors) > 0 {
		directives = append(directives, fmt.Sprintf("frame-ancestors %s", strings.Join(c.FrameAncestors, " ")))
	}

	if c.BlockAllMixedContent {
		directives = append(directives, "block-all-mixed-content")
	}

	if c.UpgradeInsecureRequests {
		directives = append(directives, "upgrade-insecure-requests")
	}

	return strings.Join(directives, "; ")
}

func GenerateCSPNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

type SecurityHeadersConfig struct {
	EnableCSP            bool
	CSP                  ContentSecurityPolicyConfig
	EnableHSTS           bool
	HSTSMaxAge            int
	HSTSIncludeSubdomains bool
	HSTSPreload          bool
	EnableXFrameOptions  bool
	XFrameOptions        string
	EnableXContentType   bool
	XContentTypeOptions  string
	EnableXSSProtection  bool
	XSSProtectionMode    string
	EnableReferrerPolicy  bool
	ReferrerPolicy       string
	EnablePermissionsPolicy bool
	PermissionsPolicy    string
	EnableOtherHeaders   bool
}

var defaultSecurityHeadersConfig = SecurityHeadersConfig{
	EnableCSP:            true,
	EnableHSTS:           true,
	HSTSMaxAge:           31536000,
	HSTSIncludeSubdomains: true,
	HSTSPreload:          true,
	EnableXFrameOptions:  true,
	XFrameOptions:        "DENY",
	EnableXContentType:   true,
	XContentTypeOptions:  "nosniff",
	EnableXSSProtection:  true,
	XSSProtectionMode:    "block",
	EnableReferrerPolicy: true,
	ReferrerPolicy:       "strict-origin-when-cross-origin",
	EnablePermissionsPolicy: true,
	PermissionsPolicy:    "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()",
	EnableOtherHeaders:   true,
}

func BuildSecurityHeaders(config SecurityHeadersConfig) map[string]string {
	headers := make(map[string]string)

	if config.EnableCSP {
		csp := config.CSP.BuildPolicy()
		headers["Content-Security-Policy"] = csp
	}

	if config.EnableHSTS {
		hsts := fmt.Sprintf("max-age=%d", config.HSTSMaxAge)
		if config.HSTSIncludeSubdomains {
			hsts += "; includeSubDomains"
		}
		if config.HSTSPreload {
			hsts += "; preload"
		}
		headers["Strict-Transport-Security"] = hsts
	}

	if config.EnableXFrameOptions {
		headers["X-Frame-Options"] = config.XFrameOptions
	}

	if config.EnableXContentType {
		headers["X-Content-Type-Options"] = config.XContentTypeOptions
	}

	if config.EnableXSSProtection {
		headers["X-XSS-Protection"] = fmt.Sprintf("1; mode=%s", config.XSSProtectionMode)
	}

	if config.EnableReferrerPolicy {
		headers["Referrer-Policy"] = config.ReferrerPolicy
	}

	if config.EnablePermissionsPolicy {
		headers["Permissions-Policy"] = config.PermissionsPolicy
	}

	if config.EnableOtherHeaders {
		headers["X-Permitted-Cross-Domain-Policies"] = "none"
		headers["X-Download-Options"] = "noopen"
	}

	return headers
}
