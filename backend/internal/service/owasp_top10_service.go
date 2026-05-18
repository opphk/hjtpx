package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type OWASPRisk struct {
	ID          string
	Name        string
	Description string
	Severity    string
	Status      string
}

type OWASPService struct {
	risks          map[string]*OWASPRisk
	validators     map[string]func(*http.Request) (bool, string)
	knownVulnerableVersions map[string][]string
	sensitiveHeaders        []string
	tamperPatterns          []*regexp.Regexp
}

func NewOWASPService() *OWASPService {
	service := &OWASPService{
		risks: map[string]*OWASPRisk{
			"A01": {"A01", "Broken Access Control", "访问控制缺陷", "Critical", "Active"},
			"A02": {"A02", "Cryptographic Failures", "加密故障", "Critical", "Active"},
			"A03": {"A03", "Injection", "注入攻击", "Critical", "Active"},
			"A04": {"A04", "Insecure Design", "不安全设计", "High", "Active"},
			"A05": {"A05", "Security Misconfiguration", "安全配置错误", "High", "Active"},
			"A06": {"A06", "Vulnerable and Outdated Components", "脆弱和过期组件", "High", "Active"},
			"A07": {"A07", "Identification and Authentication Failures", "身份识别和认证失败", "Critical", "Active"},
			"A08": {"A08", "Software and Data Integrity Failures", "软件和数据完整性故障", "High", "Active"},
			"A09": {"A09", "Security Logging and Monitoring Failures", "安全日志和监控故障", "Medium", "Active"},
			"A10": {"A10", "Server-Side Request Forgery", "服务端请求伪造", "High", "Active"},
		},
		validators: make(map[string]func(*http.Request) (bool, string)),
		knownVulnerableVersions: map[string][]string{
			"WordPress": {"2.0", "2.1", "2.2", "3.0", "3.1", "4.0"},
			"jQuery":    {"1.0", "1.1", "1.2", "1.3", "1.4", "1.5"},
			"Apache":    {"1.3", "2.0", "2.2"},
			"nginx":     {"0.6", "0.7", "0.8", "1.0", "1.2"},
			"PHP":       {"4.0", "4.1", "4.2", "4.3", "5.0", "5.1", "5.2"},
			"Node.js":   {"0.10", "0.12", "4.0", "5.0", "6.0"},
			"OpenSSL":   {"0.9.8", "1.0.0", "1.0.1"},
		},
		sensitiveHeaders: []string{
			"Authorization", "X-Api-Key", "X-Token", "Cookie",
			"Proxy-Authorization", "X-Csrf-Token", "X-Secret",
		},
		tamperPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(\.\./|\.\.)`),
			regexp.MustCompile(`(?i)(%2e%2e|%2e)`),
			regexp.MustCompile(`(?i)(null|0x00)`),
		},
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
		safe, _ := validator(r)
		results[id] = safe
	}
	return results
}

func (s *OWASPService) checkBrokenAccessControl(r *http.Request) (bool, string) {
	path := r.URL.Path
	
	sensitivePaths := []string{
		"/admin", "/config", "/backup", "/.env", "/.git",
		"/api/v1/admin", "/api/admin", "/management", "/system",
		"/private", "/secret", "/keys", "/token", "/oauth",
		"/.well-known/acme-challenge", "/cgi-bin", "/phpmyadmin",
	}
	
	for _, p := range sensitivePaths {
		if strings.Contains(path, p) {
			if !s.hasValidAccessControl(r) {
				return false, "Access to sensitive path without proper authorization: " + p
			}
		}
	}
	
	if strings.HasPrefix(path, "/api/") && r.Header.Get("Authorization") == "" {
		return false, "Missing authorization header for API endpoint"
	}
	
	return true, ""
}

func (s *OWASPService) hasValidAccessControl(r *http.Request) bool {
	if r.Header.Get("Authorization") != "" {
		return true
	}
	if r.Header.Get("X-Admin-Token") != "" {
		return true
	}
	return false
}

func (s *OWASPService) checkCryptographicFailures(r *http.Request) (bool, string) {
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
		return false, "Insecure connection (not HTTPS)"
	}
	
	if r.TLS != nil && r.TLS.CipherSuite != 0 {
		if s.isWeakCipher(r.TLS.CipherSuite) {
			return false, "Weak TLS cipher suite detected"
		}
	}
	
	if r.Header.Get("X-Content-Type-Options") == "" {
		return false, "Missing X-Content-Type-Options header"
	}
	
	if r.Header.Get("Strict-Transport-Security") == "" {
		return false, "Missing HSTS header"
	}
	
	if r.Header.Get("X-Frame-Options") == "" {
		return false, "Missing X-Frame-Options header"
	}
	
	if r.Header.Get("X-XSS-Protection") == "" {
		return false, "Missing X-XSS-Protection header"
	}
	
	return true, ""
}

func (s *OWASPService) isWeakCipher(cipher uint16) bool {
	weakCiphers := []uint16{
		0x0004, 0x0005, 0x000A, 0x0009, 0x0015, 0x0016,
		0x0013, 0x0014, 0x002F, 0x0035, 0x003D, 0x003C,
		0x003B, 0x003A, 0xC007, 0xC008, 0xC011, 0xC012,
		0x003D, 0x003C, 0x009C, 0x009D, 0x009E, 0x009F,
	}
	for _, weak := range weakCiphers {
		if cipher == weak {
			return true
		}
	}
	return false
}

func (s *OWASPService) checkInjection(r *http.Request) (bool, string) {
	query := r.URL.RawQuery
	path := r.URL.Path
	body := s.getRequestBody(r)
	combined := query + path + body
	
	sqlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union\s+select|select\s+.*from|insert\s+into|update\s+.*set|delete\s+from|drop\s+table|alter\s+table)`),
		regexp.MustCompile(`(?i)(exec\s+\(|\sproc\s+|execute\s+)`),
		regexp.MustCompile(`(?i)(sleep\s*\(|benchmark\(|pg_sleep\(|waitfor\s+delay)`),
		regexp.MustCompile(`(?i)(load_file\(|into\s+outfile|into\s+dumpfile|outfile\s*=)`),
		regexp.MustCompile(`(?i)(\'\s*or\s*1\s*=\s*1|\'\s*and\s*1\s*=\s*1|1\s*=\s*1\s*--)`),
		regexp.MustCompile(`(?i)(information_schema|sys\.tables|pg_catalog)`),
		regexp.MustCompile(`(?i)(and\s+\d+\s*=\s*\d+|or\s+\d+\s*=\s*\d+)`),
		regexp.MustCompile(`(?i)(having\s+\d+\s*=\s*\d+)`),
		regexp.MustCompile(`(?im)(union\s+all\s+select|union\s+select)`),
		regexp.MustCompile(`(?i)(\bor\b\s*\d+\s*=\s*\d+|\band\b\s*\d+\s*=\s*\d+)`),
		regexp.MustCompile(`(?i)(;\s*drop\s+table|;\s*delete\s+from|;\s*update\s+)`),
		regexp.MustCompile(`(?i)(0x[0-9a-f]+)`),
		regexp.MustCompile(`(?i)(char\s*\(\s*\d+|concat\s*\()`),
	}
	
	for _, pattern := range sqlPatterns {
		if pattern.MatchString(combined) {
			return false, "SQL injection attempt detected"
		}
	}
	
	xssPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(<script[^>]*>.*?<\/script>|<script\b)`),
		regexp.MustCompile(`(?i)(javascript:|vbscript:|data:|blob:)`),
		regexp.MustCompile(`(?i)(onclick\s*=|onload\s*=|onerror\s*=|onfocus\s*=|onmouseover\s*=|onmouseout\s*=|onblur\s*=|onchange\s*=|onsubmit\s*=|onkeydown\s*=|onkeyup\s*=|onkeypress\s*=)`),
		regexp.MustCompile(`(?i)(alert\s*\(|prompt\s*\(|confirm\s*\()`),
		regexp.MustCompile(`(?i)(<iframe[^>]*>|<svg[^>]*>|<embed[^>]*>)`),
		regexp.MustCompile(`(?i)(<object[^>]*>|<applet[^>]*>|<form[^>]*>)`),
		regexp.MustCompile(`(?i)(<meta[^>]*>|expression\s*\(|behavior\s*:)`),
		regexp.MustCompile(`(?i)(<xml[^>]*>|<xss>.*?<\/xss>)`),
		regexp.MustCompile(`(?i)(document\.|window\.|parent\.|top\.)`),
		regexp.MustCompile(`(?i)(src\s*=\s*["']?\s*javascript:|href\s*=\s*["']?\s*javascript:)`),
		regexp.MustCompile(`(?i)(<img[^>]+onerror\s*=|<img[^>]+src\s*=\s*["']x)`),
		regexp.MustCompile(`(?i)(\btarget\s*=\s*["']?\s*_blank\s*.*\bopener\b)`),
	}
	
	for _, pattern := range xssPatterns {
		if pattern.MatchString(combined) {
			return false, "XSS attack attempt detected"
		}
	}
	
	cmdPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(;\s*|&&\s*|\|\|\s*|\|\s*)`),
		regexp.MustCompile(`(?i)(` + "`" + `|\$\(|\$\{|\beval\b)`),
		regexp.MustCompile(`(?i)(wget\s+|curl\s+|nc\s+|netcat\s+|telnet\s+|ssh\s+|ftp\s+)`),
		regexp.MustCompile(`(?i)(chmod\s+|chown\s+|useradd\s+|passwd\s+|sudo\s+|su\s+)`),
		regexp.MustCompile(`(?i)(/bin/bash|/bin/sh|/usr/bin/perl|/usr/bin/python)`),
		regexp.MustCompile(`(?i)(cat\s+|grep\s+|awk\s+|sed\s+|tail\s+|head\s+|more\s+|less\s+)`),
		regexp.MustCompile(`(?i)(;\s*sh\b|;\s*bash\b|;\s*python\b|;\s*perl\b)`),
	}
	
	for _, pattern := range cmdPatterns {
		if pattern.MatchString(query) || pattern.MatchString(path) {
			return false, "Command injection attempt detected"
		}
	}
	
	filePathPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(/etc/passwd|/etc/shadow|/root/\.ssh|/\.git/config)`),
		regexp.MustCompile(`(?i)(/var/log/|/tmp/|/var/tmp/|/dev/null)`),
		regexp.MustCompile(`(?i)(\.\./|\.\.\\|\.\.%2f|\.%5c\.)`),
		regexp.MustCompile(`(?i)(%2e%2e|%2e|%2f|%5c|%252e|%252f)`),
		regexp.MustCompile(`(?i)(file:///etc/|file:///c:\\|\\\\UNC\\\\|\\\\127\\)`),
	}
	
	for _, pattern := range filePathPatterns {
		if pattern.MatchString(query) || pattern.MatchString(path) {
			return false, "Path traversal attempt detected"
		}
	}
	
	return true, ""
}

func (s *OWASPService) getRequestBody(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return string(bodyBytes)
}

func (s *OWASPService) checkSecurityMisconfiguration(r *http.Request) (bool, string) {
	serverHeader := r.Header.Get("Server")
	if serverHeader != "" && (strings.Contains(serverHeader, "Apache") || strings.Contains(serverHeader, "nginx")) {
		if strings.Contains(serverHeader, "/") {
			return false, "Server version exposed in header"
		}
	}
	return true, ""
}

func (s *OWASPService) checkAuthFailures(r *http.Request) (bool, string) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" && !strings.HasSuffix(r.URL.Path, "/login") && !strings.HasSuffix(r.URL.Path, "/public") {
		return false, "Missing authentication for protected resource"
	}
	return true, ""
}

func (s *OWASPService) checkSSRF(r *http.Request) (bool, string) {
	query := r.URL.RawQuery
	ssrfPatterns := []string{
		"http://127.0.0.1", "http://localhost", "http://0.0.0.0",
		"http://[::]", "file://", "gopher://", "ftp://",
		"http://169.254.", "http://localhost:", "http://127.1.",
		"http://2130706433", "https://127.0.0.1", "https://localhost",
	}
	
	ssrfRegex := regexp.MustCompile(`(?i)(192\.168\.|10\.|172\.(1[6-9]|2[0-9]|3[01])\.|169\.254\.|127\.|0\.0\.0\.0)`)
	
	for _, pattern := range ssrfPatterns {
		if strings.Contains(query, pattern) {
			return false, "Potential SSRF attempt: " + pattern
		}
	}
	
	if ssrfRegex.MatchString(query) {
		return false, "Potential SSRF attempt: internal network access"
	}
	
	ipv6Patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\[::1\]`),
		regexp.MustCompile(`(?i)\[::ffff:`),
		regexp.MustCompile(`(?i)\[0:0:0:0:0:ffff:`),
	}
	
	for _, pattern := range ipv6Patterns {
		if pattern.MatchString(query) {
			return false, "Potential SSRF attempt: IPv6 localhost"
		}
	}
	
	metadataPatterns := []string{
		"metadata.google.internal",
		"metadata.azure.com",
		"169.254.169.254",
		"metadata.openstack.org",
		"metadata.your-cloud-provider.com",
	}
	
	for _, pattern := range metadataPatterns {
		if strings.Contains(query, pattern) {
			return false, "Potential SSRF attempt: cloud metadata endpoint"
		}
	}
	
	protocolPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(dict://|ldap://|sftp://|imap://|pop3://|telnet://|tftp://)`),
		regexp.MustCompile(`(?i)(gopher://|ftp://|file://)`),
	}
	
	for _, pattern := range protocolPatterns {
		if pattern.MatchString(query) {
			return false, "Potential SSRF attempt: dangerous protocol"
		}
	}
	
	ipEncodingPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(0x[a-f0-9]{8})`),
		regexp.MustCompile(`(?i)(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`),
	}
	
	for _, pattern := range ipEncodingPatterns {
		if pattern.MatchString(query) {
			ip := pattern.FindString(query)
			if isPrivateOrLocalIP(ip) {
				return false, "Potential SSRF attempt: private IP encoded"
			}
		}
	}
	
	return true, ""
}

func isPrivateOrLocalIP(ip string) bool {
	privateRanges := []string{
		"10.", "172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.",
		"172.25.", "172.26.", "172.27.", "172.28.", "172.29.",
		"172.30.", "172.31.", "192.168.", "127.", "0.",
		"169.254.", "localhost",
	}
	
	for _, range_ := range privateRanges {
		if strings.Contains(ip, range_) {
			return true
		}
	}
	return false
}

func (s *OWASPService) CheckBrokenAccessControl(r *http.Request) (bool, string) {
	return s.checkBrokenAccessControl(r)
}

func (s *OWASPService) CheckCryptographicFailures(r *http.Request) (bool, string) {
	return s.checkCryptographicFailures(r)
}

func (s *OWASPService) CheckInjection(r *http.Request) (bool, string) {
	return s.checkInjection(r)
}

func (s *OWASPService) CheckSecurityMisconfiguration(r *http.Request) (bool, string) {
	return s.checkSecurityMisconfiguration(r)
}

func (s *OWASPService) CheckAuthFailures(r *http.Request) (bool, string) {
	return s.checkAuthFailures(r)
}

func (s *OWASPService) CheckSSRF(r *http.Request) (bool, string) {
	return s.checkSSRF(r)
}

func (s *OWASPService) SanitizeInput(input string) string {
	sanitized := html.EscapeString(input)
	sanitized = regexp.MustCompile(`[^\w\s\-.,!?:;@#$%^&*()_+=\[\]{}|<>/]`).ReplaceAllString(sanitized, "")
	sanitized = regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|alter|exec|execute)`).ReplaceAllString(sanitized, "")
	sanitized = regexp.MustCompile(`(?i)(eval|base64_decode|system|shell_exec|passthru)`).ReplaceAllString(sanitized, "")
	return sanitized
}

func (s *OWASPService) GetAllRisks() []*OWASPRisk {
	risks := make([]*OWASPRisk, 0, len(s.risks))
	for _, risk := range s.risks {
		risks = append(risks, risk)
	}
	return risks
}

func (s *OWASPService) checkInsecureDesign(r *http.Request) (bool, string) {
	path := r.URL.Path
	
	insecurePaths := []string{
		"/login", "/register", "/api/login", "/api/register",
	}
	
	for _, p := range insecurePaths {
		if strings.Contains(path, p) {
			if r.Method != "POST" {
				return false, "Security critical endpoint should use POST method"
			}
			
			contentType := r.Header.Get("Content-Type")
			if contentType != "application/json" && !strings.Contains(contentType, "application/x-www-form-urlencoded") {
				return false, "Missing proper content type for authentication endpoint"
			}
		}
	}
	
	if r.Header.Get("Referer") == "" && !strings.HasSuffix(path, "/public") {
		return false, "Missing Referer header"
	}
	
	return true, ""
}

func (s *OWASPService) checkVulnerableComponents(r *http.Request) (bool, string) {
	userAgent := r.UserAgent()
	
	for component, versions := range s.knownVulnerableVersions {
		if strings.Contains(userAgent, component) {
			for _, version := range versions {
				if strings.Contains(userAgent, component+"/"+version) || 
				   strings.Contains(userAgent, version) && strings.Contains(userAgent, component) {
					return false, "Vulnerable component detected: " + component + " " + version
				}
			}
		}
	}
	
	xPoweredBy := r.Header.Get("X-Powered-By")
	if xPoweredBy != "" {
		return false, "X-Powered-By header exposes technology stack"
	}
	
	return true, ""
}

func (s *OWASPService) checkDataIntegrity(r *http.Request) (bool, string) {
	query := r.URL.RawQuery
	body := s.getRequestBody(r)
	
	for _, pattern := range s.tamperPatterns {
		if pattern.MatchString(query) || pattern.MatchString(body) {
			return false, "Potential data tampering attempt detected"
		}
	}
	
	if r.Header.Get("Content-Length") == "" && body != "" {
		return false, "Missing Content-Length header for request body"
	}
	
	if r.Header.Get("Host") == "" {
		return false, "Missing Host header"
	}
	
	return true, ""
}

func (s *OWASPService) checkLoggingFailures(r *http.Request) (bool, string) {
	path := r.URL.Path
	
	criticalPaths := []string{"/login", "/logout", "/api/auth", "/admin"}
	isCriticalPath := false
	for _, p := range criticalPaths {
		if strings.Contains(path, p) {
			isCriticalPath = true
			break
		}
	}
	
	if isCriticalPath {
		if r.Header.Get("X-Request-ID") == "" {
			return false, "Missing X-Request-ID header for critical operation"
		}
	}
	
	return true, ""
}

func (s *OWASPService) CheckInsecureDesign(r *http.Request) (bool, string) {
	return s.checkInsecureDesign(r)
}

func (s *OWASPService) CheckVulnerableComponents(r *http.Request) (bool, string) {
	return s.checkVulnerableComponents(r)
}

func (s *OWASPService) CheckDataIntegrity(r *http.Request) (bool, string) {
	return s.checkDataIntegrity(r)
}

func (s *OWASPService) CheckLoggingFailures(r *http.Request) (bool, string) {
	return s.checkLoggingFailures(r)
}

func (s *OWASPService) GenerateRequestHash(r *http.Request) string {
	data := r.Method + r.URL.Path + r.URL.RawQuery + s.getRequestBody(r)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (s *OWASPService) CheckCompliance(r *http.Request) map[string]interface{} {
	result := make(map[string]interface{})
	checks := s.CheckRequest(r)

	total := len(checks)
	passed := 0
	for _, passedCheck := range checks {
		if passedCheck {
			passed++
		}
	}

	result["checks"] = checks
	result["score"] = float64(passed) / float64(total) * 100
	result["passed"] = passed
	result["total"] = total
	result["compliant"] = float64(passed)/float64(total) >= 0.9

	return result
}
