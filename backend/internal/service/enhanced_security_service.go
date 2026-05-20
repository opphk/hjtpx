package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type EnhancedSQLInjectionProtection struct {
	redis      *redis.Client
	ctx        context.Context
	validators []SQLInjectionValidator
	mu         sync.RWMutex
}

type SQLInjectionValidator struct {
	Name        string
	Pattern     *regexp.Regexp
	Severity    string
	Description string
}

func NewEnhancedSQLInjectionProtection(redisClient *redis.Client) *EnhancedSQLInjectionProtection {
	protection := &EnhancedSQLInjectionProtection{
		redis: redisClient,
		ctx:   context.Background(),
		validators: []SQLInjectionValidator{
			{
				Name:        "union_select",
				Pattern:     regexp.MustCompile(`(?i)\b(union\s+(all\s+)?select|union\s+select)\b`),
				Severity:    "HIGH",
				Description: "Detected UNION SELECT pattern",
			},
			{
				Name:        "or_always_true",
				Pattern:     regexp.MustCompile(`(?i)('\s*(or|and)\s*'?[\d\w]+'?\s*[=<>]|1\s*=\s*1|--|;--)`),
				Severity:    "HIGH",
				Description: "Detected always-true SQL condition",
			},
			{
				Name:        "comment_injection",
				Pattern:     regexp.MustCompile(`(?i)(--|#|\/\*|\*\/)`),
				Severity:    "MEDIUM",
				Description: "Detected SQL comment pattern",
			},
			{
				Name:        "stacked_queries",
				Pattern:     regexp.MustCompile(`(?i);\s*(select|insert|update|delete|drop|alter|exec|execute|create|truncate)`),
				Severity:    "CRITICAL",
				Description: "Detected stacked queries pattern",
			},
			{
				Name:        "information_schema",
				Pattern:     regexp.MustCompile(`(?i)(information_schema|sys\.(tables|columns)|mysql\.(user|db)|pg_catalog|sqlite_master)`),
				Severity:    "HIGH",
				Description: "Detected system table access attempt",
			},
			{
				Name:        "time_based_blind",
				Pattern:     regexp.MustCompile(`(?i)(sleep\s*\(|benchmark\s*\(|pg_sleep\s*\(|waitfor\s+delay\b|get_lock\s*\()`),
				Severity:    "HIGH",
				Description: "Detected time-based blind injection pattern",
			},
			{
				Name:        "file_operations",
				Pattern:     regexp.MustCompile(`(?i)(load_file\s*\(|into\s+(out|dump)file|outfile\s*=)`),
				Severity:    "CRITICAL",
				Description: "Detected file operation pattern",
			},
			{
				Name:        "hex_encoding",
				Pattern:     regexp.MustCompile(`(?i)0x[0-9a-f]+`),
				Severity:    "MEDIUM",
				Description: "Detected hex-encoded data",
			},
			{
				Name:        "char_function",
				Pattern:     regexp.MustCompile(`(?i)char\s*\(\s*\d+(\s*,\s*\d+)*\s*\)`),
				Severity:    "MEDIUM",
				Description: "Detected CHAR function usage",
			},
			{
				Name:        "concat_function",
				Pattern:     regexp.MustCompile(`(?i)concat\s*\(`),
				Severity:    "MEDIUM",
				Description: "Detected CONCAT function usage",
			},
		},
	}

	return protection
}

func (s *EnhancedSQLInjectionProtection) ValidateInput(input string) (bool, string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, validator := range s.validators {
		if validator.Pattern.MatchString(input) {
			return false, validator.Name, validator.Severity
		}
	}

	return true, "", ""
}

func (s *EnhancedSQLInjectionProtection) ValidateAllInputs(inputs map[string]string) map[string]struct {
	Valid    bool
	Pattern  string
	Severity string
} {
	results := make(map[string]struct {
		Valid    bool
		Pattern  string
		Severity string
	})

	for key, value := range inputs {
		valid, pattern, severity := s.ValidateInput(value)
		results[key] = struct {
			Valid    bool
			Pattern  string
			Severity string
		}{
			Valid:    valid,
			Pattern:  pattern,
			Severity: severity,
		}
	}

	return results
}

func (s *EnhancedSQLInjectionProtection) SanitizeInput(input string) string {
	sanitized := input

	dangerousPatterns := []struct {
		pattern    *regexp.Regexp
		replacement string
	}{
		{regexp.MustCompile(`(?i)('\s*(or|and)\s*'?[\d\w]+'?\s*[=<>])`), ""},
		{regexp.MustCompile(`(?i)(union\s+(all\s+)?select)`), ""},
		{regexp.MustCompile(`(?i)(select\s+\*\s+from)`), ""},
		{regexp.MustCompile(`(?i)(insert\s+into)`), ""},
		{regexp.MustCompile(`(?i)(update\s+.*\s+set)`), ""},
		{regexp.MustCompile(`(?i)(delete\s+from)`), ""},
		{regexp.MustCompile(`(?i)(drop\s+(table|database))`), ""},
		{regexp.MustCompile(`(?i)(exec\s*\(|\bexecute\s*\()`), ""},
		{regexp.MustCompile(`(?i)(--|#|\/\*|\*\/)`), ""},
	}

	for _, dp := range dangerousPatterns {
		sanitized = dp.pattern.ReplaceAllString(sanitized, dp.replacement)
	}

	return strings.TrimSpace(sanitized)
}

func (s *EnhancedSQLInjectionProtection) LogInjectionAttempt(ctx context.Context, input string, pattern string, severity string) error {
	if s.redis == nil {
		return nil
	}

	key := fmt.Sprintf("security:sql_injection:%s", time.Now().Format("2006-01-02"))
	entry := fmt.Sprintf("{\"timestamp\":\"%s\",\"input\":\"%s\",\"pattern\":\"%s\",\"severity\":\"%s\"}",
		time.Now().Format(time.RFC3339),
		html.EscapeString(input),
		pattern,
		severity,
	)

	return s.redis.LPush(ctx, key, entry).Err()
}

func (s *EnhancedSQLInjectionProtection) AddCustomValidator(name string, pattern *regexp.Regexp, severity string, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.validators = append(s.validators, SQLInjectionValidator{
		Name:        name,
		Pattern:     pattern,
		Severity:    severity,
		Description: description,
	})
}

type EnhancedRBACService struct {
	redis       *redis.Client
	ctx         context.Context
	roles       map[string]*Role
	permissions map[string]*Permission
	mu          sync.RWMutex
}

type Role struct {
	Name        string
	Permissions []string
	Parent      string
}

type Permission struct {
	Name        string
	Resource    string
	Action      string
	Description string
}

func NewEnhancedRBACService(redisClient *redis.Client) *EnhancedRBACService {
	rbac := &EnhancedRBACService{
		redis:       redisClient,
		ctx:         context.Background(),
		roles:       make(map[string]*Role),
		permissions: make(map[string]*Permission),
	}

	rbac.initializeDefaultRoles()
	rbac.initializeDefaultPermissions()

	return rbac
}

func (s *EnhancedRBACService) initializeDefaultRoles() {
	s.roles = map[string]*Role{
		"admin": {
			Name:        "admin",
			Permissions: []string{"*"},
			Parent:      "",
		},
		"moderator": {
			Name:        "moderator",
			Permissions: []string{"user:read", "user:update", "content:read", "content:moderate", "report:read"},
			Parent:      "",
		},
		"user": {
			Name:        "user",
			Permissions: []string{"user:read:own", "user:update:own", "content:read", "content:create"},
			Parent:      "",
		},
		"guest": {
			Name:        "guest",
			Permissions: []string{"content:read"},
			Parent:      "",
		},
	}
}

func (s *EnhancedRBACService) initializeDefaultPermissions() {
	permissions := []Permission{
		{Name: "user:read", Resource: "user", Action: "read"},
		{Name: "user:read:own", Resource: "user", Action: "read:own"},
		{Name: "user:create", Resource: "user", Action: "create"},
		{Name: "user:update", Resource: "user", Action: "update"},
		{Name: "user:update:own", Resource: "user", Action: "update:own"},
		{Name: "user:delete", Resource: "user", Action: "delete"},
		{Name: "content:read", Resource: "content", Action: "read"},
		{Name: "content:create", Resource: "content", Action: "create"},
		{Name: "content:update", Resource: "content", Action: "update"},
		{Name: "content:delete", Resource: "content", Action: "delete"},
		{Name: "content:moderate", Resource: "content", Action: "moderate"},
		{Name: "report:read", Resource: "report", Action: "read"},
		{Name: "report:create", Resource: "report", Action: "create"},
		{Name: "settings:read", Resource: "settings", Action: "read"},
		{Name: "settings:update", Resource: "settings", Action: "update"},
	}

	for _, p := range permissions {
		s.permissions[p.Name] = &p
	}
}

func (s *EnhancedRBACService) HasPermission(role string, permission string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	roleData, exists := s.roles[role]
	if !exists {
		return false
	}

	for _, p := range roleData.Permissions {
		if p == "*" {
			return true
		}
		if p == permission {
			return true
		}
		if strings.HasPrefix(permission, strings.TrimSuffix(p, "*")) && strings.HasSuffix(p, "*") {
			return true
		}
	}

	return false
}

func (s *EnhancedRBACService) CheckPermission(ctx context.Context, userID string, permission string) (bool, error) {
	userRole, err := s.GetUserRole(ctx, userID)
	if err != nil {
		return false, err
	}

	return s.HasPermission(userRole, permission), nil
}

func (s *EnhancedRBACService) GetUserRole(ctx context.Context, userID string) (string, error) {
	if s.redis != nil {
		role, err := s.redis.HGet(ctx, fmt.Sprintf("user:%s", userID), "role").Result()
		if err == nil {
			return role, nil
		}
	}

	return "guest", nil
}

func (s *EnhancedRBACService) SetUserRole(ctx context.Context, userID string, role string) error {
	if s.redis != nil {
		return s.redis.HSet(ctx, fmt.Sprintf("user:%s", userID), "role", role).Err()
	}
	return nil
}

func (s *EnhancedRBACService) AddRole(role *Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.roles[role.Name]; exists {
		return errors.New("role already exists")
	}

	s.roles[role.Name] = role
	return nil
}

func (s *EnhancedRBACService) UpdateRolePermissions(roleName string, permissions []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	role, exists := s.roles[roleName]
	if !exists {
		return errors.New("role not found")
	}

	role.Permissions = permissions
	return nil
}

func (s *EnhancedRBACService) GetRolePermissions(roleName string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	role, exists := s.roles[roleName]
	if !exists {
		return nil, errors.New("role not found")
	}

	return role.Permissions, nil
}

func (s *EnhancedRBACService) GetAllRoles() []*Role {
	s.mu.RLock()
	defer s.mu.RUnlock()

	roles := make([]*Role, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, role)
	}

	return roles
}

type EnhancedCSRFService struct {
	redis         *redis.Client
	ctx           context.Context
	tokenLength   int
	tokenExpiry   time.Duration
	tokens        map[string]*CSRFToken
	mu            sync.RWMutex
}

type CSRFToken struct {
	Token       string
	UserID      string
	SessionID   string
	IPAddress   string
	UserAgent   string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Used        bool
}

func NewEnhancedCSRFService(redisClient *redis.Client) *EnhancedCSRFService {
	return &EnhancedCSRFService{
		redis:       redisClient,
		ctx:         context.Background(),
		tokenLength: 32,
		tokenExpiry: 1 * time.Hour,
		tokens:      make(map[string]*CSRFToken),
	}
}

func (s *EnhancedCSRFService) GenerateToken(ctx context.Context, userID string, sessionID string, ipAddress string, userAgent string) (string, error) {
	bytes := make([]byte, s.tokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)

	csrfToken := &CSRFToken{
		Token:     token,
		UserID:    userID,
		SessionID: sessionID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.tokenExpiry),
		Used:      false,
	}

	s.mu.Lock()
	s.tokens[token] = csrfToken
	s.mu.Unlock()

	if s.redis != nil {
		key := fmt.Sprintf("csrf:token:%s", token)
		data := map[string]interface{}{
			"user_id":    userID,
			"session_id": sessionID,
			"ip_address": ipAddress,
			"user_agent": userAgent,
			"created_at": csrfToken.CreatedAt.Unix(),
			"expires_at": csrfToken.ExpiresAt.Unix(),
			"used":       false,
		}
		if err := s.redis.HSet(ctx, key, data).Err(); err != nil {
			return "", err
		}
		if err := s.redis.Expire(ctx, key, s.tokenExpiry).Err(); err != nil {
			return "", err
		}
	}

	return token, nil
}

func (s *EnhancedCSRFService) ValidateToken(ctx context.Context, token string, ipAddress string, userAgent string) (bool, error) {
	s.mu.RLock()
	tokenData, exists := s.tokens[token]
	s.mu.RUnlock()

	if exists {
		if time.Now().After(tokenData.ExpiresAt) {
			return false, errors.New("token expired")
		}

		if tokenData.Used {
			return false, errors.New("token already used")
		}

		if ipAddress != "" && tokenData.IPAddress != "" && tokenData.IPAddress != ipAddress {
			return false, errors.New("IP address mismatch")
		}

		return true, nil
	}

	if s.redis != nil {
		key := fmt.Sprintf("csrf:token:%s", token)
		data, err := s.redis.HGetAll(ctx, key).Result()
		if err != nil || len(data) == 0 {
			return false, errors.New("token not found")
		}

		expiresAt, _ := strconv.ParseInt(data["expires_at"], 10, 64)
		if time.Now().Unix() > expiresAt {
			return false, errors.New("token expired")
		}

		used, _ := strconv.ParseBool(data["used"])
		if used {
			return false, errors.New("token already used")
		}

		return true, nil
	}

	return false, errors.New("token not found")
}

func (s *EnhancedCSRFService) InvalidateToken(ctx context.Context, token string) error {
	s.mu.Lock()
	if tokenData, exists := s.tokens[token]; exists {
		tokenData.Used = true
	}
	s.mu.Unlock()

	if s.redis != nil {
		key := fmt.Sprintf("csrf:token:%s", token)
		return s.redis.HSet(ctx, key, "used", true).Err()
	}

	return nil
}

func (s *EnhancedCSRFService) InvalidateUserTokens(ctx context.Context, userID string) error {
	s.mu.Lock()
	for _, tokenData := range s.tokens {
		if tokenData.UserID == userID {
			tokenData.Used = true
		}
	}
	s.mu.Unlock()

	if s.redis != nil {
		pattern := fmt.Sprintf("csrf:token:*")
		keys, err := s.redis.Keys(ctx, pattern).Result()
		if err != nil {
			return err
		}

		for _, key := range keys {
			userIDFromToken, err := s.redis.HGet(ctx, key, "user_id").Result()
			if err == nil && userIDFromToken == userID {
				s.redis.HSet(ctx, key, "used", true)
			}
		}
	}

	return nil
}

func (s *EnhancedCSRFService) CleanupExpiredTokens() {
	s.mu.Lock()
	for token, tokenData := range s.tokens {
		if time.Now().After(tokenData.ExpiresAt) {
			delete(s.tokens, token)
		}
	}
	s.mu.Unlock()
}

type EnhancedXSSProtection struct {
	redis          *redis.Client
	ctx            context.Context
	allowedTags    map[string]bool
	allowedAttrs   map[string]bool
	dangerousTags  []*regexp.Regexp
	dangerousAttrs []*regexp.Regexp
	mu             sync.RWMutex
}

func NewEnhancedXSSProtection(redisClient *redis.Client) *EnhancedXSSProtection {
	protection := &EnhancedXSSProtection{
		redis:        redisClient,
		ctx:          context.Background(),
		allowedTags:  make(map[string]bool),
		allowedAttrs: make(map[string]bool),
		dangerousTags: []*regexp.Regexp{
			regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
			regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
			regexp.MustCompile(`(?i)<object[^>]*>.*?</object>`),
			regexp.MustCompile(`(?i)<embed[^>]*>`),
			regexp.MustCompile(`(?i)<applet[^>]*>.*?</applet>`),
			regexp.MustCompile(`(?i)<form[^>]*>.*?</form>`),
			regexp.MustCompile(`(?i)<link[^>]*>`),
			regexp.MustCompile(`(?i)<meta[^>]*>`),
			regexp.MustCompile(`(?i)<base[^>]*>`),
			regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`),
			regexp.MustCompile(`(?i)<svg[^>]*>.*?</svg>`),
			regexp.MustCompile(`(?i)<math[^>]*>.*?</math>`),
			regexp.MustCompile(`(?i)<xss[^>]*>.*?</xss>`),
		},
		dangerousAttrs: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\s+on\w+\s*=\s*(["'][^"']*["']|[^\s>]*)`),
			regexp.MustCompile(`(?i)\bhref\s*=\s*["']?\s*javascript:`),
			regexp.MustCompile(`(?i)\bsrc\s*=\s*["']?\s*javascript:`),
			regexp.MustCompile(`(?i)\baction\s*=\s*["']?\s*javascript:`),
			regexp.MustCompile(`(?i)\bdata\s*=\s*["']?\s*text/html`),
			regexp.MustCompile(`(?i)\bformaction\s*=\s*["']?\s*javascript:`),
			regexp.MustCompile(`(?i)expression\s*\(`),
			regexp.MustCompile(`(?i)behavior\s*:`),
		},
	}

	protection.allowedTags = map[string]bool{
		"p": true, "br": true, "b": true, "i": true, "em": true,
		"strong": true, "a": true, "ul": true, "ol": true, "li": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"table": true, "tr": true, "td": true, "th": true, "thead": true, "tbody": true,
		"div": true, "span": true, "blockquote": true, "pre": true, "code": true,
		"img": true, "hr": true,
	}

	protection.allowedAttrs = map[string]bool{
		"href": true, "title": true, "class": true, "id": true, "style": true,
		"alt": true, "src": true, "width": true, "height": true, "target": true,
	}

	return protection
}

func (s *EnhancedXSSProtection) SanitizeHTML(input string) string {
	sanitized := input

	for _, pattern := range s.dangerousTags {
		sanitized = pattern.ReplaceAllString(sanitized, "")
	}

	for _, pattern := range s.dangerousAttrs {
		sanitized = pattern.ReplaceAllString(sanitized, "")
	}

	sanitized = regexp.MustCompile(`(?i)&#(\d+);`).ReplaceAllStringFunc(sanitized, func(match string) string {
		re := regexp.MustCompile(`&#(\d+);`)
		matches := re.FindStringSubmatch(match)
		if len(matches) > 1 {
			num, _ := strconv.Atoi(matches[1])
			if num == 60 || num == 62 || num == 34 || num == 39 || num < 32 || num > 126 {
				return ""
			}
		}
		return match
	})

	sanitized = regexp.MustCompile(`&#x([0-9a-fA-F]+);`).ReplaceAllStringFunc(sanitized, func(match string) string {
		re := regexp.MustCompile(`&#x([0-9a-fA-F]+);`)
		matches := re.FindStringSubmatch(match)
		if len(matches) > 1 {
			num, _ := strconv.ParseInt(matches[1], 16, 64)
			if num == 0x3C || num == 0x3E || num == 0x22 || num == 0x27 || num < 0x20 || num > 0x7E {
				return ""
			}
		}
		return match
	})

	sanitized = regexp.MustCompile(`(?i)<!--.*?-->`).ReplaceAllString(sanitized, "")

	sanitized = regexp.MustCompile(`(?i)<!\[CDATA\[.*?\]\]>`).ReplaceAllString(sanitized, "")

	if !strings.ContainsAny(sanitized, "<>") {
		return html.EscapeString(sanitized)
	}

	allowedPattern := regexp.MustCompile(`<(/?)(p|br|b|i|em|strong|a|ul|ol|li|h[1-6]|table|tr|td|th|thead|tbody|div|span|blockquote|pre|code|img|hr)([^>]*)>`)
	sanitized = allowedPattern.ReplaceAllStringFunc(sanitized, func(match string) string {
		lowerMatch := strings.ToLower(match)
		for attrPattern := range map[string]bool{
			"on": true, "javascript": true, "vbscript": true, "data:text": true,
		} {
			if strings.Contains(lowerMatch, attrPattern+":") || strings.HasPrefix(lowerMatch, attrPattern+":") {
				return ""
			}
		}
		return match
	})

	return sanitized
}

func (s *EnhancedXSSProtection) DetectXSS(input string) (bool, string, string) {
	lowerInput := strings.ToLower(input)

	xssPatterns := []struct {
		pattern    string
		name       string
		severity   string
	}{
		{`<script`, "script_tag", "CRITICAL"},
		{`javascript:`, "javascript_protocol", "CRITICAL"},
		{`vbscript:`, "vbscript_protocol", "HIGH"},
		{`data:`, "data_protocol", "HIGH"},
		{`onerror`, "onerror_event", "CRITICAL"},
		{`onload`, "onload_event", "HIGH"},
		{`onclick`, "onclick_event", "HIGH"},
		{`onmouseover`, "onmouseover_event", "MEDIUM"},
		{`<iframe`, "iframe_tag", "HIGH"},
		{`<svg`, "svg_tag", "HIGH"},
		{`<embed`, "embed_tag", "HIGH"},
		{`<object`, "object_tag", "HIGH"},
		{`<form`, "form_tag", "MEDIUM"},
		{`expression(`, "css_expression", "HIGH"},
		{`alert(`, "alert_function", "HIGH"},
		{`prompt(`, "prompt_function", "HIGH"},
		{`confirm(`, "confirm_function", "HIGH"},
	}

	for _, xp := range xssPatterns {
		if strings.Contains(lowerInput, xp.pattern) {
			return true, xp.name, xp.severity
		}
	}

	return false, "", ""
}

func (s *EnhancedXSSProtection) ValidateURL(urlStr string) (bool, string) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return false, "Invalid URL format"
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" && parsed.Scheme != "mailto" && parsed.Scheme != "tel" {
		return false, fmt.Sprintf("Disallowed URL scheme: %s", parsed.Scheme)
	}

	if parsed.Host != "" {
		host := parsed.Hostname()
		if net.ParseIP(host) != nil {
			if IsPrivateIP(host) {
				return false, "URL points to private IP address"
			}
		}
	}

	return true, ""
}

func (s *EnhancedXSSProtection) LogXSSAttempt(ctx context.Context, input string, pattern string, severity string) error {
	if s.redis == nil {
		return nil
	}

	key := fmt.Sprintf("security:xss:%s", time.Now().Format("2006-01-02"))
	entry := fmt.Sprintf("{\"timestamp\":\"%s\",\"input\":\"%s\",\"pattern\":\"%s\",\"severity\":\"%s\"}",
		time.Now().Format(time.RFC3339),
		html.EscapeString(input),
		pattern,
		severity,
	)

	return s.redis.LPush(ctx, key, entry).Err()
}

type SecurityAuditService struct {
	redis            *redis.Client
	ctx              context.Context
	auditLogPrefix   string
	maxLogAge        time.Duration
	enabledChecks    map[string]bool
	mu               sync.RWMutex
}

type AuditLogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	UserID      string    `json:"user_id"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	Result      string    `json:"result"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	Details     string    `json:"details"`
	RiskScore   float64   `json:"risk_score"`
}

func NewSecurityAuditService(redisClient *redis.Client) *SecurityAuditService {
	return &SecurityAuditService{
		redis:          redisClient,
		ctx:            context.Background(),
		auditLogPrefix: "security:audit:",
		maxLogAge:      30 * 24 * time.Hour,
		enabledChecks: map[string]bool{
			"sql_injection":   true,
			"xss":             true,
			"csrf":            true,
			"path_traversal":  true,
			"ssrf":            true,
			"auth_failures":   true,
			"rate_limit":      true,
		},
	}
}

func (s *SecurityAuditService) LogSecurityEvent(entry *AuditLogEntry) error {
	if s.redis == nil {
		return nil
	}

	entry.Timestamp = time.Now()
	key := fmt.Sprintf("%s%s", s.auditLogPrefix, entry.Timestamp.Format("2006-01-02"))

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return s.redis.LPush(s.ctx, key, data).Err()
}

func (s *SecurityAuditService) GetSecurityEvents(startDate time.Time, endDate time.Time, eventType string) ([]*AuditLogEntry, error) {
	if s.redis == nil {
		return nil, nil
	}

	var entries []*AuditLogEntry
	currentDate := startDate

	for currentDate.Before(endDate) || currentDate.Equal(endDate) {
		key := fmt.Sprintf("%s%s", s.auditLogPrefix, currentDate.Format("2006-01-02"))
		data, err := s.redis.LRange(s.ctx, key, 0, -1).Result()
		if err != nil {
			continue
		}

		for _, item := range data {
			var entry AuditLogEntry
			if err := json.Unmarshal([]byte(item), &entry); err != nil {
				continue
			}

			if eventType == "" || entry.Action == eventType {
				entries = append(entries, &entry)
			}
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return entries, nil
}

func (s *SecurityAuditService) AnalyzeSecurityEvents(startDate time.Time, endDate time.Time) (map[string]interface{}, error) {
	events, err := s.GetSecurityEvents(startDate, endDate, "")
	if err != nil {
		return nil, err
	}

	highRiskCount := 0
	successCount := 0
	failedCount := 0

	eventTypes := make(map[string]int)
	topUsers := make(map[string]int)
	topIPs := make(map[string]int)

	for _, event := range events {
		eventTypes[event.Action]++
		topUsers[event.UserID]++
		topIPs[event.IPAddress]++

		if event.RiskScore > 0.7 {
			highRiskCount++
		}

		if event.Result == "success" {
			successCount++
		} else if event.Result == "failure" {
			failedCount++
		}
	}

	analysis := map[string]interface{}{
		"total_events":      len(events),
		"event_types":       eventTypes,
		"top_users":         topUsers,
		"top_ips":           topIPs,
		"high_risk_events":  highRiskCount,
		"successful_events": successCount,
		"failed_events":     failedCount,
	}

	return analysis, nil
}

func (s *SecurityAuditService) EnableCheck(checkName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabledChecks[checkName] = true
}

func (s *SecurityAuditService) DisableCheck(checkName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabledChecks[checkName] = false
}

func (s *SecurityAuditService) IsCheckEnabled(checkName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabledChecks[checkName]
}

func IsPrivateIP(ip string) bool {
	privateRanges := []string{
		"10.",
		"172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.",
		"172.25.", "172.26.", "172.27.", "172.28.", "172.29.",
		"172.30.", "172.31.",
		"192.168.",
		"127.",
		"0.",
		"169.254.",
		"[::1]",
		"[::ffff:",
	}

	for _, range_ := range privateRanges {
		if strings.Contains(ip, range_) {
			return true
		}
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP != nil {
		if parsedIP.IsLoopback() || parsedIP.IsPrivate() || parsedIP.IsUnspecified() {
			return true
		}
	}

	return false
}

func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GenerateUUID() string {
	return uuid.New().String()
}
