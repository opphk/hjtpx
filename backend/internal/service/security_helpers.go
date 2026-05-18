package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"html"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/redis/go-redis/v9"
)

// CaptchaService 验证码服务
type CaptchaService struct {
	redis *redis.Client
	ctx   context.Context
}

// NewCaptchaService 创建验证码服务
func NewCaptchaService(redisClient *redis.Client) *CaptchaService {
	return &CaptchaService{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// ValidateCaptchaToken 验证验证码Token
func (s *CaptchaService) ValidateCaptchaToken(token string, expiration time.Duration) bool {
	if token == "" {
		return false
	}
	
	// 简单的token格式检查
	if len(token) < 10 {
		return false
	}
	
	// 检查token是否已被使用（在Redis中记录）
	if s.redis != nil {
		key := "captcha:used:" + token
		exists, _ := s.redis.Exists(s.ctx, key).Result()
		if exists > 0 {
			return false // Token已被使用
		}
	}
	
	return true
}

// MarkCaptchaTokenUsed 标记验证码Token已使用
func (s *CaptchaService) MarkCaptchaTokenUsed(token string, expiration time.Duration) error {
	if s.redis != nil {
		key := "captcha:used:" + token
		return s.redis.Set(s.ctx, key, "1", expiration).Err()
	}
	return nil
}

// MaskSensitiveData 屏蔽敏感数据
func MaskSensitiveData(input string) string {
	sensitivePatterns := []struct {
		pattern *regexp.Regexp
		mask   string
	}{
		{regexp.MustCompile(`(?i)(password|passwd|pwd)[\s:=]+[^\s,;]+`), "$1=***MASKED***"},
		{regexp.MustCompile(`(?i)(api[_-]?key|apikey)[\s:=]+[^\s,;]+`), "$1=***MASKED***"},
		{regexp.MustCompile(`(?i)(token|bearer)[\s:=]+[^\s,;]+`), "$1=***MASKED***"},
		{regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`), "****-****-****-****"},
		{regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`), "***-**-****"},
	}
	
	result := input
	for _, p := range sensitivePatterns {
		result = p.pattern.ReplaceAllString(result, p.mask)
	}
	
	return result
}

// SanitizeLogEntry 清理日志条目中的敏感信息
func SanitizeLogEntry(data map[string]string) map[string]string {
	sensitiveFields := []string{"password", "token", "api_key", "secret", "credential", "auth"}
	
	result := make(map[string]string)
	for key, value := range data {
		isSensitive := false
		lowerKey := strings.ToLower(key)
		for _, sensitive := range sensitiveFields {
			if strings.Contains(lowerKey, sensitive) {
				isSensitive = true
				break
			}
		}
		
		if isSensitive {
			result[key] = "***MASKED***"
		} else {
			result[key] = value
		}
	}
	
	return result
}

// IsStrongCaptcha 检查验证码强度
func IsStrongCaptcha(captcha string) bool {
	if len(captcha) < 4 {
		return false
	}
	
	// 检查是否包含重复字符
	hasRepeating := regexp.MustCompile(`(.)\1{2,}`).MatchString(captcha)
	if hasRepeating {
		return false
	}
	
	// 检查是否为弱验证码模式
	weakPatterns := []string{
		"1234", "0000", "1111", "2222", "3333", "4444", "5555", "6666", "7777", "8888", "9999",
		"abcd", "aaaa", "bbbb",
		"qwer", "asdf", "zxcv",
	}
	
	lowerCaptcha := strings.ToLower(captcha)
	for _, pattern := range weakPatterns {
		if lowerCaptcha == pattern || strings.Contains(lowerCaptcha, pattern) {
			return false
		}
	}
	
	return true
}

// GenerateSecureNonce 生成安全随机数
func GenerateSecureNonce(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// JWTClaims JWT声明
type JWTClaims struct {
	AdminID   uint   `json:"admin_id"`
	Username  string `json:"username"`
	ExpiresAt int64  `json:"exp"`
}

// GenerateJWT 生成JWT Token（包装pkg/jwt）
func GenerateJWT(redisClient *redis.Client, adminID uint, username string) (string, error) {
	return jwt.GenerateToken(adminID, username)
}

// ParseJWT 解析JWT Token（包装pkg/jwt）
func ParseJWT(redisClient *redis.Client, token string) (*JWTClaims, error) {
	claims, err := jwt.ParseToken(token)
	if err != nil {
		return nil, err
	}
	
	return &JWTClaims{
		AdminID:   claims.AdminID,
		Username:  claims.Username,
		ExpiresAt: claims.ExpiresAt.Unix(),
	}, nil
}

// SecuritySessionService 会话服务（与管理员会话服务区分）
type SecuritySessionService struct {
	redis *redis.Client
	ctx   context.Context
	sessions map[string]*SecuritySessionInfo
	mu      sync.RWMutex
}

type SecuritySessionInfo struct {
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// NewSecuritySessionService 创建会话服务
func NewSecuritySessionService(redisClient *redis.Client) *SecuritySessionService {
	return &SecuritySessionService{
		redis:    redisClient,
		ctx:      context.Background(),
		sessions: make(map[string]*SecuritySessionInfo),
	}
}

// CreateSecuritySession 创建新会话
func (s *SecuritySessionService) CreateSecuritySession(ctx context.Context, userID string, duration time.Duration) (string, error) {
	sessionID, err := GenerateSecureNonce(32)
	if err != nil {
		return "", err
	}
	
	sessionInfo := &SecuritySessionInfo{
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt:  time.Now().Add(duration),
	}
	
	if s.redis != nil {
		key := "session:" + sessionID
		data := map[string]interface{}{
			"user_id":    userID,
			"created_at": sessionInfo.CreatedAt.Unix(),
		}
		s.redis.HSet(ctx, key, data)
		s.redis.Expire(ctx, key, duration)
	} else {
		s.mu.Lock()
		s.sessions[sessionID] = sessionInfo
		s.mu.Unlock()
	}
	
	return sessionID, nil
}

// ValidateSecuritySession 验证会话
func (s *SecuritySessionService) ValidateSecuritySession(ctx context.Context, sessionID string) (bool, error) {
	if s.redis != nil {
		key := "session:" + sessionID
		exists, err := s.redis.Exists(ctx, key).Result()
		if err != nil {
			return false, err
		}
		return exists > 0, nil
	} else {
		s.mu.RLock()
		defer s.mu.RUnlock()
		
		session, exists := s.sessions[sessionID]
		if !exists {
			return false, nil
		}
		
		return time.Now().Before(session.ExpiresAt), nil
	}
}

// InvalidateSecuritySession 使会话失效
func (s *SecuritySessionService) InvalidateSecuritySession(ctx context.Context, sessionID string) error {
	if s.redis != nil {
		key := "session:" + sessionID
		return s.redis.Del(ctx, key).Err()
	} else {
		s.mu.Lock()
		defer s.mu.Unlock()
		
		delete(s.sessions, sessionID)
		return nil
	}
}

// XSSSecurity XSS安全服务
type XSSSecurity struct {
	redis *redis.Client
	ctx   context.Context
}

// NewXSSSecurity 创建XSS安全服务
func NewXSSSecurity(redisClient *redis.Client) *XSSSecurity {
	return &XSSSecurity{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// SanitizeInput 清理输入
func (s *XSSSecurity) SanitizeInput(input string) string {
	return SanitizeHTML(input)
}

// SanitizeHTML HTML转义和清理
func SanitizeHTML(input string) string {
	// 移除script标签（包括编码的）
	input = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)<script[^>]*>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)</script>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)<\/?script`).ReplaceAllString(input, "")
	
	// 移除iframe
	input = regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)<iframe[^>]*>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)</iframe>`).ReplaceAllString(input, "")
	
	// 移除img标签（可能有onerror）
	input = regexp.MustCompile(`(?i)<img[^>]*>`).ReplaceAllString(input, "")
	
	// 移除svg标签
	input = regexp.MustCompile(`(?i)<svg[^>]*>.*?</svg>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)<svg[^>]*>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)</svg>`).ReplaceAllString(input, "")
	
	// 移除embed标签
	input = regexp.MustCompile(`(?i)<embed[^>]*>`).ReplaceAllString(input, "")
	
	// 移除object标签
	input = regexp.MustCompile(`(?i)<object[^>]*>.*?</object>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)<object[^>]*>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)</object>`).ReplaceAllString(input, "")
	
	// 移除applet标签
	input = regexp.MustCompile(`(?i)<applet[^>]*>.*?</applet>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)<applet[^>]*>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)</applet>`).ReplaceAllString(input, "")
	
	// 移除form标签
	input = regexp.MustCompile(`(?i)<form[^>]*>.*?</form>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)<form[^>]*>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)</form>`).ReplaceAllString(input, "")
	
	// 移除link标签
	input = regexp.MustCompile(`(?i)<link[^>]*>`).ReplaceAllString(input, "")
	
	// 移除meta标签
	input = regexp.MustCompile(`(?i)<meta[^>]*>`).ReplaceAllString(input, "")
	
	// 移除base标签
	input = regexp.MustCompile(`(?i)<base[^>]*>`).ReplaceAllString(input, "")
	
	// 移除style标签
	input = regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)<style[^>]*>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)</style>`).ReplaceAllString(input, "")
	
	// 移除xml标签
	input = regexp.MustCompile(`(?i)<xml[^>]*>.*?</xml>`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)<xml[^>]*>`).ReplaceAllString(input, "")
	
	// 移除xss标签
	input = regexp.MustCompile(`(?i)<xss[^>]*>.*?</xss>`).ReplaceAllString(input, "")
	
	// 移除事件处理器
	input = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*(["'][^"']*["']|[^\s>]*)`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonload\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonclick\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonerror\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonmouseover\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonfocus\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonblur\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonchange\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonsubmit\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonkeydown\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonkeyup\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonkeypress\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonmouseout\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonmousedown\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonmouseup\b`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)\bonmouseover\b`).ReplaceAllString(input, "")
	
	// 移除javascript:协议
	input = regexp.MustCompile(`(?i)javascript\s*:`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)vbscript\s*:`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)data\s*:`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)blob\s*:`).ReplaceAllString(input, "")
	
	// 移除HTML注释（可能被用于绕过）
	input = regexp.MustCompile(`<!--.*?-->`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`<!--.*`).ReplaceAllString(input, "")
	
	// 移除CDATA节
	input = regexp.MustCompile(`(?i)<!\[CDATA\[.*?\]\]>`).ReplaceAllString(input, "")
	
	// 移除HTML实体编码
	input = regexp.MustCompile(`&#(\d+);`).ReplaceAllStringFunc(input, func(match string) string {
		return ""
	})
	input = regexp.MustCompile(`&#x([0-9a-f]+);`).ReplaceAllStringFunc(input, func(match string) string {
		return ""
	})
	
	// HTML转义
	return html.EscapeString(input)
}

// DetectXSS 检测XSS攻击
func (s *XSSSecurity) DetectXSS(input string) (bool, string) {
	xssPatterns := []struct {
		pattern string
		name    string
	}{
		{`<script`, "script_tag"},
		{`javascript:`, "javascript_protocol"},
		{`vbscript:`, "vbscript_protocol"},
		{`data:`, "data_protocol"},
		{`onerror`, "onerror_event"},
		{`onload`, "onload_event"},
		{`onclick`, "onclick_event"},
		{`onmouseover`, "onmouseover_event"},
		{`onfocus`, "onfocus_event"},
		{`onblur`, "onblur_event"},
		{`onchange`, "onchange_event"},
		{`onsubmit`, "onsubmit_event"},
		{`onkeydown`, "onkeydown_event"},
		{`onkeyup`, "onkeyup_event"},
		{`onkeypress`, "onkeypress_event"},
		{`<iframe`, "iframe_tag"},
		{`<img`, "img_tag"},
		{`<svg`, "svg_tag"},
		{`<embed`, "embed_tag"},
		{`<object`, "object_tag"},
		{`<applet`, "applet_tag"},
		{`<form`, "form_tag"},
		{`<link`, "link_tag"},
		{`<meta`, "meta_tag"},
		{`<base`, "base_tag"},
		{`<style`, "style_tag"},
		{`<xml`, "xml_tag"},
		{`expression(`, "css_expression"},
		{`behavior:`, "css_behavior"},
		{`alert(`, "alert_function"},
		{`prompt(`, "prompt_function"},
		{`confirm(`, "confirm_function"},
		{`document.`, "document_object"},
		{`window.`, "window_object"},
		{`parent.`, "parent_object"},
		{`top.`, "top_object"},
		{`&#`, "html_entity"},
		{`\\x`, "hex_escape"},
		{`\\u`, "unicode_escape"},
	}
	
	lowerInput := strings.ToLower(input)
	for _, p := range xssPatterns {
		if strings.Contains(lowerInput, p.pattern) {
			return true, p.name
		}
	}
	
	return false, ""
}

// DetectSQLInjection 检测SQL注入攻击
func (s *XSSSecurity) DetectSQLInjection(input string) (bool, string) {
	sqlPatterns := []struct {
		pattern string
		name    string
	}{
		{`union select`, "union_select"},
		{`union all select`, "union_all_select"},
		{`select * from`, "select_all"},
		{`insert into`, "insert_statement"},
		{`update set`, "update_statement"},
		{`delete from`, "delete_statement"},
		{`drop table`, "drop_table"},
		{`alter table`, "alter_table"},
		{`exec(`, "exec_statement"},
		{`execute(`, "execute_statement"},
		{`';--`, "sql_comment"},
		{`' or '1'='1`, "or_always_true"},
		{`' or 1=1`, "or_numeric_true"},
		{`1=1--`, "numeric_true_comment"},
		{`information_schema`, "information_schema"},
		{`sys.tables`, "system_tables"},
		{`pg_catalog`, "postgres_catalog"},
		{`sleep(`, "time_based"},
		{`benchmark(`, "benchmark_function"},
		{`waitfor delay`, "mssql_delay"},
		{`load_file(`, "file_read"},
		{`into outfile`, "file_write"},
		{`0x`, "hex_literal"},
		{`char(`, "char_function"},
		{`concat(`, "concat_function"},
	}
	
	lowerInput := strings.ToLower(input)
	for _, p := range sqlPatterns {
		if strings.Contains(lowerInput, p.pattern) {
			return true, p.name
		}
	}
	
	return false, ""
}

// DetectCommandInjection 检测命令注入攻击
func (s *XSSSecurity) DetectCommandInjection(input string) (bool, string) {
	cmdPatterns := []struct {
		pattern string
		name    string
	}{
		{`&&`, "and_operator"},
		{`||`, "or_operator"},
		{";", "semicolon"},
		{"|", "pipe"},
		{"`", "backtick"},
		{"$(", "subshell"},
		{"${", "variable_expansion"},
		{"eval(", "eval_function"},
		{"system(", "system_function"},
		{"exec(", "exec_function"},
		{"passthru(", "passthru_function"},
		{"shell_exec(", "shell_exec_function"},
		{"popen(", "popen_function"},
	}
	
	lowerInput := strings.ToLower(input)
	for _, p := range cmdPatterns {
		if strings.Contains(lowerInput, p.pattern) {
			return true, p.name
		}
	}
	
	return false, ""
}
