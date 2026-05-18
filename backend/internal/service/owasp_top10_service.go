package service

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type OWASPRisk struct {
	ID          string
	Name        string
	Description string
	Severity    string
	Status      string
	CVSSBase    float64
}

type OWASPComplianceResult struct {
	Checks           map[string]bool
	Score            float64
	Passed           int
	Total            int
	Compliant        bool
	RiskDetails      []*OWASPRiskDetail
	Recommendations  []string
}

type OWASPRiskDetail struct {
	RiskID        string
	RiskName      string
	Severity      string
	Detected      bool
	Evidence      string
	CVSSScore     float64
	Recommendation string
}

type OWASPService struct {
	risks                      map[string]*OWASPRisk
	validators                 map[string]func(*http.Request) (bool, string, float64)
	knownVulnerableVersions    map[string][]string
	sensitiveHeaders           []string
	tamperPatterns             []*regexp.Regexp
	sensitivePaths             []string
	blockedExtensions          []string
	allowedHTTPMethods         map[string]bool
	rateLimitConfigs           map[string]RateLimitConfig
	mu                         sync.RWMutex
	requestTimestamps          map[string]time.Time
	sessionTokens              map[string]bool
	blockedIPs                 map[string]time.Time
	allowedOrigins             map[string]bool
}



func NewOWASPService() *OWASPService {
	service := &OWASPService{
		risks: map[string]*OWASPRisk{
			"A01": {"A01", "Broken Access Control", "访问控制缺陷", "Critical", "Active", 9.8},
			"A02": {"A02", "Cryptographic Failures", "加密故障", "Critical", "Active", 9.1},
			"A03": {"A03", "Injection", "注入攻击", "Critical", "Active", 8.8},
			"A04": {"A04", "Insecure Design", "不安全设计", "High", "Active", 8.2},
			"A05": {"A05", "Security Misconfiguration", "安全配置错误", "High", "Active", 7.5},
			"A06": {"A06", "Vulnerable and Outdated Components", "脆弱和过期组件", "High", "Active", 7.3},
			"A07": {"A07", "Identification and Authentication Failures", "身份识别和认证失败", "Critical", "Active", 9.8},
			"A08": {"A08", "Software and Data Integrity Failures", "软件和数据完整性故障", "High", "Active", 8.2},
			"A09": {"A09", "Security Logging and Monitoring Failures", "安全日志和监控故障", "Medium", "Active", 5.3},
			"A10": {"A10", "Server-Side Request Forgery", "服务端请求伪造", "High", "Active", 8.1},
		},
		validators: make(map[string]func(*http.Request) (bool, string, float64)),
		knownVulnerableVersions: map[string][]string{
			"WordPress":   {"2.0", "2.1", "2.2", "3.0", "3.1", "4.0", "4.1", "4.2", "4.3"},
			"jQuery":      {"1.0", "1.1", "1.2", "1.3", "1.4", "1.5", "1.6", "1.7", "1.8"},
			"Apache":      {"1.3", "2.0", "2.2", "2.4.0", "2.4.1"},
			"nginx":       {"0.6", "0.7", "0.8", "1.0", "1.2", "1.4"},
			"PHP":         {"4.0", "4.1", "4.2", "4.3", "5.0", "5.1", "5.2", "5.3"},
			"Node.js":     {"0.10", "0.12", "4.0", "5.0", "6.0", "7.0", "8.0"},
			"OpenSSL":     {"0.9.8", "1.0.0", "1.0.1", "1.0.2"},
			"MySQL":       {"5.0", "5.1", "5.5", "5.6"},
			"PostgreSQL":  {"8.0", "8.1", "8.2", "8.3", "8.4"},
			"Redis":       {"2.0", "2.2", "2.4", "2.6", "2.8"},
			"Elasticsearch": {"1.0", "1.1", "1.2", "1.3", "1.4", "1.5"},
		},
		sensitiveHeaders: []string{
			"Authorization", "X-Api-Key", "X-Token", "Cookie",
			"Proxy-Authorization", "X-Csrf-Token", "X-Secret",
			"X-Session-Token", "X-Auth-Token", "X-User-Token",
			"X-Client-Secret", "X-App-Secret", "X-Access-Token",
		},
		tamperPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(\.\./|\.\.)`),
			regexp.MustCompile(`(?i)(%2e%2e|%2e)`),
			regexp.MustCompile(`(?i)(null|0x00|\0)`),
			regexp.MustCompile(`(?i)(%00|%0d|%0a)`),
			regexp.MustCompile(`(?i)(%2f|%5c)`),
		},
		sensitivePaths: []string{
			"/admin", "/config", "/backup", "/.env", "/.git",
			"/api/v1/admin", "/api/admin", "/management", "/system",
			"/private", "/secret", "/keys", "/token", "/oauth",
			"/.well-known/acme-challenge", "/cgi-bin", "/phpmyadmin",
			"/wp-admin", "/wp-login", "/xmlrpc.php", "/adminer",
			"/server-status", "/status", "/healthz", "/readyz",
			"/swagger", "/docs", "/api-docs", "/openapi",
			"/graphql", "/graphiql", "/playground",
		},
		blockedExtensions: []string{
			".php", ".asp", ".aspx", ".jsp", ".jspx", ".cgi",
			".pl", ".py", ".rb", ".sh", ".bat", ".cmd",
			".exe", ".dll", ".so", ".dylib", ".jar", ".war",
			".sql", ".bak", ".backup", ".old", ".tmp", ".log",
		},
		allowedHTTPMethods: map[string]bool{
			"GET":     true,
			"POST":    true,
			"PUT":     true,
			"DELETE":  true,
			"PATCH":   true,
			"OPTIONS": true,
			"HEAD":    true,
			"CONNECT": false,
			"TRACE":   false,
		},
		rateLimitConfigs: map[string]RateLimitConfig{
			"/login":    {MaxRequests: 5, WindowSecs: 300},
			"/register": {MaxRequests: 3, WindowSecs: 600},
			"/api/auth": {MaxRequests: 10, WindowSecs: 60},
			"/admin":    {MaxRequests: 20, WindowSecs: 60},
		},
		requestTimestamps: make(map[string]time.Time),
		sessionTokens:     make(map[string]bool),
		blockedIPs:        make(map[string]time.Time),
		allowedOrigins:    make(map[string]bool),
	}

	service.validators["A01"] = service.checkBrokenAccessControl
	service.validators["A02"] = service.checkCryptographicFailures
	service.validators["A03"] = service.checkInjection
	service.validators["A04"] = service.checkInsecureDesign
	service.validators["A05"] = service.checkSecurityMisconfiguration
	service.validators["A06"] = service.checkVulnerableComponents
	service.validators["A07"] = service.checkAuthFailures
	service.validators["A08"] = service.checkDataIntegrity
	service.validators["A09"] = service.checkLoggingFailures
	service.validators["A10"] = service.checkSSRF

	return service
}

func (s *OWASPService) CheckRequest(r *http.Request) map[string]bool {
	results := make(map[string]bool)
	for id, validator := range s.validators {
		safe, _, _ := validator(r)
		results[id] = safe
	}
	return results
}

func (s *OWASPService) checkBrokenAccessControl(r *http.Request) (bool, string, float64) {
	path := r.URL.Path
	method := r.Method

	if !s.allowedHTTPMethods[method] {
		return false, "Disallowed HTTP method: " + method, 9.8
	}

	for _, p := range s.sensitivePaths {
		if strings.Contains(path, p) {
			if !s.hasValidAccessControl(r) {
				return false, "Access to sensitive path without proper authorization: " + p, 9.8
			}
		}
	}

	for _, ext := range s.blockedExtensions {
		if strings.HasSuffix(path, ext) {
			return false, "Access to blocked file extension: " + ext, 7.5
		}
	}

	if strings.HasPrefix(path, "/api/") && r.Header.Get("Authorization") == "" {
		authPaths := []string{"/api/public", "/api/health", "/api/status"}
		isPublic := false
		for _, publicPath := range authPaths {
			if strings.HasPrefix(path, publicPath) {
				isPublic = true
				break
			}
		}
		if !isPublic {
			return false, "Missing authorization header for API endpoint", 9.8
		}
	}

	if s.isRateLimited(r) {
		return false, "Rate limit exceeded for sensitive endpoint", 5.0
	}

	return true, "", 0.0
}

func (s *OWASPService) hasValidAccessControl(r *http.Request) bool {
	if r.Header.Get("Authorization") != "" {
		return true
	}
	if r.Header.Get("X-Admin-Token") != "" {
		return true
	}
	if r.Header.Get("X-API-Key") != "" {
		return true
	}
	cookie, err := r.Cookie("session")
	if err == nil && cookie.Value != "" {
		return true
	}
	return false
}

func (s *OWASPService) isRateLimited(r *http.Request) bool {
	path := r.URL.Path
	ip := getClientIP(r)

	for prefix, config := range s.rateLimitConfigs {
		if strings.HasPrefix(path, prefix) {
			s.mu.Lock()
			key := ip + ":" + prefix
			lastTime, exists := s.requestTimestamps[key]
			if exists && time.Since(lastTime) < time.Duration(config.WindowSecs)*time.Second {
				s.mu.Unlock()
				return true
			}
			s.requestTimestamps[key] = time.Now()
			s.mu.Unlock()
		}
	}
	return false
}

func (s *OWASPService) checkCryptographicFailures(r *http.Request) (bool, string, float64) {
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
		return false, "Insecure connection (not HTTPS)", 9.1
	}

	if r.TLS != nil {
		if s.isWeakCipher(r.TLS.CipherSuite) {
			return false, "Weak TLS cipher suite detected: " + strconv.Itoa(int(r.TLS.CipherSuite)), 7.5
		}

		if !s.isValidCertificate(r.TLS) {
			return false, "Invalid TLS certificate", 8.1
		}

		if r.TLS.Version < 0x0303 {
			return false, "Outdated TLS version: " + strconv.Itoa(int(r.TLS.Version)), 7.5
		}
	}

	if r.Header.Get("X-Content-Type-Options") == "" {
		return false, "Missing X-Content-Type-Options header", 5.3
	}

	if r.Header.Get("Strict-Transport-Security") == "" {
		return false, "Missing HSTS header", 5.3
	}

	if r.Header.Get("X-Frame-Options") == "" {
		return false, "Missing X-Frame-Options header", 5.3
	}

	if r.Header.Get("X-XSS-Protection") == "" {
		return false, "Missing X-XSS-Protection header", 5.3
	}

	if r.Header.Get("Content-Security-Policy") == "" {
		return false, "Missing Content-Security-Policy header", 5.3
	}

	return true, "", 0.0
}

func (s *OWASPService) isWeakCipher(cipher uint16) bool {
	weakCiphers := []uint16{
		0x0004, 0x0005, 0x000A, 0x0009, 0x0015, 0x0016,
		0x0013, 0x0014, 0x002F, 0x0035, 0x003D, 0x003C,
		0x003B, 0x003A, 0xC007, 0xC008, 0xC011, 0xC012,
		0x003D, 0x003C, 0x009C, 0x009D, 0x009E, 0x009F,
		0x006B, 0x006C, 0x006D, 0x006E, 0x0084, 0x0085,
		0x0086, 0x0087, 0xC013, 0xC014, 0xC015, 0xC016,
		0x0040, 0x0041, 0x0042, 0x0043, 0x0067, 0x0068,
	}
	for _, weak := range weakCiphers {
		if cipher == weak {
			return true
		}
	}
	return false
}

func (s *OWASPService) isValidCertificate(tls *tls.ConnectionState) bool {
	if len(tls.PeerCertificates) == 0 {
		return false
	}
	cert := tls.PeerCertificates[0]
	if cert.NotAfter.Before(time.Now()) {
		return false
	}
	if cert.NotBefore.After(time.Now()) {
		return false
	}
	return true
}

func (s *OWASPService) checkInjection(r *http.Request) (bool, string, float64) {
	query := r.URL.RawQuery
	path := r.URL.Path
	body := s.getRequestBody(r)
	combined := query + path + body

	sqlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union\s+select|select\s+.*from|insert\s+into|update\s+.*set|delete\s+from|drop\s+table|alter\s+table)`),
		regexp.MustCompile(`(?i)(exec\s+\(|\sproc\s+|execute\s+|sp_executesql)`),
		regexp.MustCompile(`(?i)(sleep\s*\(|benchmark\(|pg_sleep\(|waitfor\s+delay|dbms_lock.sleep)`),
		regexp.MustCompile(`(?i)(load_file\(|into\s+outfile|into\s+dumpfile|outfile\s*=|xp_cmdshell)`),
		regexp.MustCompile(`(?i)(\'\s*or\s*1\s*=\s*1|\'\s*and\s*1\s*=\s*1|1\s*=\s*1\s*--|--.*|#.*)`),
		regexp.MustCompile(`(?i)(information_schema|sys\.tables|pg_catalog|mysql\.user)`),
		regexp.MustCompile(`(?i)(concat\(|concat_ws\(|group_concat|union\s+all)`),
		regexp.MustCompile(`(?i)(case\s+when|if\s+\(|decode\(|substr\()`),
		regexp.MustCompile(`(?i)(@\@version|@@hostname|@@datadir)`),
		regexp.MustCompile(`(?i)(binary\s+|cast\s*\(|convert\s*\()`),
	}

	for _, pattern := range sqlPatterns {
		if pattern.MatchString(combined) {
			return false, "SQL injection attempt detected: " + pattern.String(), 8.8
		}
	}

	xssPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(<script[^>]*>.*<\/script>|<script\b)`),
		regexp.MustCompile(`(?i)(javascript:|vbscript:|data:|blob:|livescript:|mocha:)`),
		regexp.MustCompile(`(?i)(onclick\s*=|onload\s*=|onerror\s*=|onfocus\s*=|onmouseover\s*=|onkeydown\s*=|onchange\s*=)`),
		regexp.MustCompile(`(?i)(alert\s*\(|prompt\s*\(|confirm\s*\(|eval\s*\(|setTimeout\s*\(|setInterval\s*\()`),
		regexp.MustCompile(`(?i)(<iframe[^>]*>|<svg[^>]*>|<embed[^>]*>|<object[^>]*>|<applet[^>]*>)`),
		regexp.MustCompile(`(?i)(<form[^>]*>|<input[^>]*>|<textarea[^>]*>)`),
		regexp.MustCompile(`(?i)(<img[^>]*src\s*=|src\s*=.*javascript:)`),
		regexp.MustCompile(`(?i)(document\.cookie|document\.write|location\.href)`),
		regexp.MustCompile(`(?i)(<style[^>]*>.*<\/style>|expression\s*\()`),
		regexp.MustCompile(`(?i)(\]\]\s*>|<\/?[^>]+>`),
	}

	for _, pattern := range xssPatterns {
		if pattern.MatchString(combined) {
			return false, "XSS attack attempt detected: " + pattern.String(), 8.4
		}
	}

	cmdPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(;\s*|&&\s*|\|\|\s*|\|\s*|\$\(|\$\{)`),
		regexp.MustCompile(`(?i)(` + "`" + `|\beval\b|\bexec\b|\bsystem\b)`),
		regexp.MustCompile(`(?i)(wget\s+|curl\s+|nc\s+|netcat\s+|telnet\s+|ssh\s+|ftp\s+)`),
		regexp.MustCompile(`(?i)(chmod\s+|chown\s+|useradd\s+|passwd\s+|sudo\s+|su\s+)`),
		regexp.MustCompile(`(?i)(/bin/bash|/bin/sh|/usr/bin/perl|/usr/bin/python|/usr/bin/ruby)`),
		regexp.MustCompile(`(?i)(powershell|cmd\.exe|command\.com|cscript|wscript)`),
		regexp.MustCompile(`(?i)(rm\s+-rf|del\s+/f|format\s+|mkfs\s+)`),
	}

	for _, pattern := range cmdPatterns {
		if pattern.MatchString(query) || pattern.MatchString(path) {
			return false, "Command injection attempt detected: " + pattern.String(), 8.8
		}
	}

	filePathPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(/etc/passwd|/etc/shadow|/root/\.ssh|/\.git/config|/var/log/|/tmp/)`),
		regexp.MustCompile(`(?i)(\.\./|\.\.)`),
		regexp.MustCompile(`(?i)(%2e%2e|%2e|%2f|%5c|%252e)`),
		regexp.MustCompile(`(?i)(/proc/self/environ|/dev/null|/dev/zero|/dev/random)`),
	}

	for _, pattern := range filePathPatterns {
		if pattern.MatchString(query) || pattern.MatchString(path) {
			return false, "Path traversal attempt detected: " + pattern.String(), 7.5
		}
	}

	xmlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(<!DOCTYPE|<!ENTITY|xmlns:|xml:base)`),
		regexp.MustCompile(`(?i)(SYSTEM\s+|PUBLIC\s+)"?[^"]*"`),
	}

	for _, pattern := range xmlPatterns {
		if pattern.MatchString(body) {
			return false, "XXE injection attempt detected", 8.1
		}
	}

	return true, "", 0.0
}

func (s *OWASPService) getRequestBody(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return string(bodyBytes)
}

func (s *OWASPService) checkSecurityMisconfiguration(r *http.Request) (bool, string, float64) {
	serverHeader := r.Header.Get("Server")
	if serverHeader != "" {
		if strings.Contains(serverHeader, "/") && (strings.Contains(serverHeader, "Apache") || strings.Contains(serverHeader, "nginx") || strings.Contains(serverHeader, "Microsoft")) {
			return false, "Server version exposed in header: " + serverHeader, 7.5
		}
	}

	xPoweredBy := r.Header.Get("X-Powered-By")
	if xPoweredBy != "" {
		return false, "X-Powered-By header exposes technology stack: " + xPoweredBy, 5.3
	}

	xAspNetVersion := r.Header.Get("X-AspNet-Version")
	if xAspNetVersion != "" {
		return false, "X-AspNet-Version header exposed", 5.3
	}

	xAspNetMvcVersion := r.Header.Get("X-AspNetMvc-Version")
	if xAspNetMvcVersion != "" {
		return false, "X-AspNetMvc-Version header exposed", 5.3
	}

	return true, "", 0.0
}

func (s *OWASPService) checkAuthFailures(r *http.Request) (bool, string, float64) {
	path := r.URL.Path
	authHeader := r.Header.Get("Authorization")

	publicPaths := []string{"/login", "/register", "/public", "/api/public", "/health", "/status"}
	isPublic := false
	for _, publicPath := range publicPaths {
		if strings.HasPrefix(path, publicPath) || strings.HasSuffix(path, publicPath) {
			isPublic = true
			break
		}
	}

	if !isPublic && authHeader == "" {
		return false, "Missing authentication for protected resource", 9.8
	}

	if strings.HasSuffix(path, "/login") || strings.HasSuffix(path, "/register") {
		if r.Method != http.MethodPost {
			return false, "Authentication endpoints should use POST method", 7.5
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" && !strings.Contains(contentType, "application/x-www-form-urlencoded") {
			return false, "Missing proper content type for authentication endpoint", 5.3
		}
	}

	if authHeader != "" && !strings.HasPrefix(authHeader, "Bearer ") && !strings.HasPrefix(authHeader, "Basic ") {
		return false, "Invalid authentication scheme", 7.5
	}

	return true, "", 0.0
}

func (s *OWASPService) checkSSRF(r *http.Request) (bool, string, float64) {
	query := r.URL.RawQuery
	body := s.getRequestBody(r)
	combined := query + body

	ssrfPatterns := []string{
		"http://127.0.0.1", "http://localhost", "http://0.0.0.0",
		"http://[::]", "file://", "gopher://", "ftp://",
		"http://169.254.", "http://localhost:", "http://127.1.",
		"http://0:80", "http://0000:80", "http://[::1]",
	}

	for _, pattern := range ssrfPatterns {
		if strings.Contains(combined, pattern) {
			return false, "Potential SSRF attempt: " + pattern, 8.1
		}
	}

	privateIPRegex := regexp.MustCompile(`(?i)(192\.168\.|10\.|172\.(1[6-9]|2[0-9]|3[01])\.|169\.254\.|127\.|100\.(6[4-9]|[7-9][0-9]|1[0-1][0-9]|12[0-7])\.)`)
	if privateIPRegex.MatchString(combined) {
		return false, "Potential SSRF attempt: internal network access", 8.1
	}

	ipv6Patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\[::1\]`),
		regexp.MustCompile(`(?i)\[::ffff:`),
		regexp.MustCompile(`(?i)\[fe80:`),
		regexp.MustCompile(`(?i)\[fc00:`),
	}

	for _, pattern := range ipv6Patterns {
		if pattern.MatchString(combined) {
			return false, "Potential SSRF attempt: IPv6 localhost or local network", 8.1
		}
	}

	metadataPatterns := []string{
		"metadata.google.internal",
		"metadata.azure.com",
		"169.254.169.254",
		"metadata.openstack.org",
		"169.254.169.254/latest/meta-data",
		"instance-data.ec2.internal",
		"vcap.me",
	}

	for _, pattern := range metadataPatterns {
		if strings.Contains(combined, pattern) {
			return false, "Potential SSRF attempt: cloud metadata endpoint", 8.1
		}
	}

	hostHeader := r.Header.Get("Host")
	if hostHeader != "" {
		if privateIPRegex.MatchString(hostHeader) {
			return false, "Host header contains private IP address", 7.5
		}
	}

	return true, "", 0.0
}

func (s *OWASPService) CheckBrokenAccessControl(r *http.Request) (bool, string, float64) {
	return s.checkBrokenAccessControl(r)
}

func (s *OWASPService) CheckCryptographicFailures(r *http.Request) (bool, string, float64) {
	return s.checkCryptographicFailures(r)
}

func (s *OWASPService) CheckInjection(r *http.Request) (bool, string, float64) {
	return s.checkInjection(r)
}

func (s *OWASPService) CheckSecurityMisconfiguration(r *http.Request) (bool, string, float64) {
	return s.checkSecurityMisconfiguration(r)
}

func (s *OWASPService) CheckAuthFailures(r *http.Request) (bool, string, float64) {
	return s.checkAuthFailures(r)
}

func (s *OWASPService) CheckSSRF(r *http.Request) (bool, string, float64) {
	return s.checkSSRF(r)
}

func (s *OWASPService) SanitizeInput(input string) string {
	sanitized := html.EscapeString(input)
	sanitized = regexp.MustCompile(`[^\w\s\-.,!?:;@#$%^&*()_+=\[\]{}|<>/]`).ReplaceAllString(sanitized, "")
	sanitized = regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|alter|exec|execute)`).ReplaceAllString(sanitized, "")
	sanitized = regexp.MustCompile(`(?i)(eval|base64_decode|system|shell_exec|passthru)`).ReplaceAllString(sanitized, "")
	sanitized = regexp.MustCompile(`(?i)(javascript:|vbscript:)`).ReplaceAllString(sanitized, "")
	return sanitized
}

func (s *OWASPService) GetAllRisks() []*OWASPRisk {
	risks := make([]*OWASPRisk, 0, len(s.risks))
	for _, risk := range s.risks {
		risks = append(risks, risk)
	}
	return risks
}

func (s *OWASPService) checkInsecureDesign(r *http.Request) (bool, string, float64) {
	path := r.URL.Path
	method := r.Method

	securityCriticalPaths := []string{"/login", "/register", "/api/login", "/api/register", "/api/auth"}
	for _, p := range securityCriticalPaths {
		if strings.Contains(path, p) {
			if method != http.MethodPost {
				return false, "Security critical endpoint should use POST method", 8.2
			}

			contentType := r.Header.Get("Content-Type")
			if contentType != "application/json" && !strings.Contains(contentType, "application/x-www-form-urlencoded") {
				return false, "Missing proper content type for authentication endpoint", 5.3
			}
		}
	}

	referrer := r.Header.Get("Referer")
	if referrer == "" && !strings.HasSuffix(path, "/public") {
		return false, "Missing Referer header", 5.3
	}

	contentType := r.Header.Get("Content-Type")
	if method == http.MethodPost && contentType == "" {
		return false, "Missing Content-Type header for POST request", 5.3
	}

	return true, "", 0.0
}

func (s *OWASPService) checkVulnerableComponents(r *http.Request) (bool, string, float64) {
	userAgent := r.UserAgent()

	for component, versions := range s.knownVulnerableVersions {
		if strings.Contains(userAgent, component) {
			for _, version := range versions {
				if strings.Contains(userAgent, component+"/"+version) ||
					strings.Contains(userAgent, version) && strings.Contains(userAgent, component) {
					return false, "Vulnerable component detected: " + component + " " + version, 7.3
				}
			}
		}
	}

	xPoweredBy := r.Header.Get("X-Powered-By")
	if xPoweredBy != "" {
		return false, "X-Powered-By header exposes technology stack: " + xPoweredBy, 5.3
	}

	return true, "", 0.0
}

func (s *OWASPService) checkDataIntegrity(r *http.Request) (bool, string, float64) {
	query := r.URL.RawQuery
	body := s.getRequestBody(r)

	for _, pattern := range s.tamperPatterns {
		if pattern.MatchString(query) || pattern.MatchString(body) {
			return false, "Potential data tampering attempt detected: " + pattern.String(), 8.2
		}
	}

	if r.Header.Get("Content-Length") == "" && body != "" {
		return false, "Missing Content-Length header for request body", 5.3
	}

	if r.Header.Get("Host") == "" {
		return false, "Missing Host header", 5.3
	}

	contentLengthStr := r.Header.Get("Content-Length")
	if contentLengthStr != "" {
		contentLength, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			return false, "Invalid Content-Length header", 5.3
		}
		if contentLength != len(body) {
			return false, "Content-Length mismatch", 8.2
		}
	}

	return true, "", 0.0
}

func (s *OWASPService) checkLoggingFailures(r *http.Request) (bool, string, float64) {
	path := r.URL.Path

	criticalPaths := []string{"/login", "/logout", "/api/auth", "/admin", "/password", "/token"}
	isCriticalPath := false
	for _, p := range criticalPaths {
		if strings.Contains(path, p) {
			isCriticalPath = true
			break
		}
	}

	if isCriticalPath {
		if r.Header.Get("X-Request-ID") == "" {
			return false, "Missing X-Request-ID header for critical operation", 5.3
		}
	}

	return true, "", 0.0
}

func (s *OWASPService) CheckInsecureDesign(r *http.Request) (bool, string, float64) {
	return s.checkInsecureDesign(r)
}

func (s *OWASPService) CheckVulnerableComponents(r *http.Request) (bool, string, float64) {
	return s.checkVulnerableComponents(r)
}

func (s *OWASPService) CheckDataIntegrity(r *http.Request) (bool, string, float64) {
	return s.checkDataIntegrity(r)
}

func (s *OWASPService) CheckLoggingFailures(r *http.Request) (bool, string, float64) {
	return s.checkLoggingFailures(r)
}

func (s *OWASPService) GenerateRequestHash(r *http.Request) string {
	data := r.Method + r.URL.Path + r.URL.RawQuery + s.getRequestBody(r)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *OWASPService) CheckCompliance(r *http.Request) map[string]interface{} {
	result := make(map[string]interface{})
	checks := make(map[string]bool)
	riskDetails := make([]*OWASPRiskDetail, 0)
	recommendations := make([]string, 0)
	totalScore := 0.0

	total := len(s.validators)
	passed := 0

	for id, validator := range s.validators {
		safe, evidence, cvss := validator(r)
		checks[id] = safe

		if !safe {
			risk := s.risks[id]
			riskDetails = append(riskDetails, &OWASPRiskDetail{
				RiskID:        id,
				RiskName:      risk.Name,
				Severity:      risk.Severity,
				Detected:      true,
				Evidence:      evidence,
				CVSSScore:     cvss,
				Recommendation: s.getRecommendation(id),
			})
			totalScore += cvss
			recommendations = append(recommendations, s.getRecommendation(id))
		} else {
			passed++
		}
	}

	result["checks"] = checks
	result["score"] = float64(passed) / float64(total) * 100
	result["passed"] = passed
	result["total"] = total
	result["compliant"] = float64(passed)/float64(total) >= 0.9
	result["risk_details"] = riskDetails
	result["recommendations"] = recommendations
	result["overall_risk_score"] = totalScore

	return result
}

func (s *OWASPService) getRecommendation(riskID string) string {
	recommendations := map[string]string{
		"A01": "实施严格的访问控制策略，使用角色基础访问控制(RBAC)，最小权限原则",
		"A02": "强制使用HTTPS，禁用弱加密套件，实施HSTS策略",
		"A03": "使用参数化查询，实施输入验证和输出编码，使用WAF防护",
		"A04": "实施安全设计原则，进行威胁建模，使用安全开发生命周期",
		"A05": "移除不必要的服务和功能，禁用默认账户，定期更新配置",
		"A06": "定期更新依赖组件，使用依赖检查工具，实施软件供应链安全",
		"A07": "实施多因素认证，使用强密码策略，实施会话管理最佳实践",
		"A08": "使用数字签名验证数据完整性，实施代码审查，使用CI/CD安全检查",
		"A09": "实施全面的日志记录，配置安全监控告警，定期审查日志",
		"A10": "实施输入验证白名单，禁止访问内部服务，使用SSRF防护工具",
	}
	return recommendations[riskID]
}

func (s *OWASPService) ValidateRequest(r *http.Request) *OWASPComplianceResult {
	result := &OWASPComplianceResult{
		Checks:          make(map[string]bool),
		RiskDetails:     make([]*OWASPRiskDetail, 0),
		Recommendations: make([]string, 0),
	}

	total := len(s.validators)
	passed := 0

	for id, validator := range s.validators {
		safe, evidence, cvss := validator(r)
		result.Checks[id] = safe

		if !safe {
			risk := s.risks[id]
			result.RiskDetails = append(result.RiskDetails, &OWASPRiskDetail{
				RiskID:        id,
				RiskName:      risk.Name,
				Severity:      risk.Severity,
				Detected:      true,
				Evidence:      evidence,
				CVSSScore:     cvss,
				Recommendation: s.getRecommendation(id),
			})
			result.Recommendations = append(result.Recommendations, s.getRecommendation(id))
		} else {
			passed++
		}
	}

	result.Passed = passed
	result.Total = total
	result.Score = float64(passed) / float64(total) * 100
	result.Compliant = result.Score >= 90

	return result
}

func (s *OWASPService) AddToBlockedIPs(ip string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blockedIPs[ip] = time.Now().Add(duration)
}

func (s *OWASPService) IsIPBlocked(ip string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	expiry, exists := s.blockedIPs[ip]
	if !exists {
		return false
	}
	return time.Now().Before(expiry)
}

func (s *OWASPService) AddAllowedOrigin(origin string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allowedOrigins[origin] = true
}

func (s *OWASPService) IsOriginAllowed(origin string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.allowedOrigins[origin]
}

