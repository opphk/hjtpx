package service

import (
	"bytes"
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
	risks      map[string]*OWASPRisk
	validators map[string]func(*http.Request) (bool, string)
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
	}

	service.validators["A01"] = service.checkBrokenAccessControl
	service.validators["A02"] = service.checkCryptographicFailures
	service.validators["A03"] = service.checkInjection
	service.validators["A05"] = service.checkSecurityMisconfiguration
	service.validators["A07"] = service.checkAuthFailures
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
	suspiciousPaths := []string{"/admin", "/config", "/backup", "/.env", "/.git"}
	for _, p := range suspiciousPaths {
		if strings.Contains(path, p) {
			return false, "Access to sensitive path: " + p
		}
	}
	return true, ""
}

func (s *OWASPService) checkCryptographicFailures(r *http.Request) (bool, string) {
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
		return false, "Insecure connection (not HTTPS)"
	}
	return true, ""
}

func (s *OWASPService) checkInjection(r *http.Request) (bool, string) {
	query := r.URL.RawQuery
	path := r.URL.Path
	body := s.getRequestBody(r)
	combined := query + path + body
	
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|alter)`),
		regexp.MustCompile(`(?i)(<script|javascript:|on\w+\s*=)`),
		regexp.MustCompile(`(?i)(exec|system|shell_exec|passthru|eval|base64_decode)`),
		regexp.MustCompile(`(?i)(/outtmp|/etc/passwd|/etc/shadow|/root/.ssh|/.git/config)`),
		regexp.MustCompile(`(?i)(sleep\s*\(|benchmark\(|pg_sleep|waitfor\s+delay)`),
		regexp.MustCompile(`(?i)(load_file|into\s+outfile|into\s+dumpfile)`),
	}
	
	for _, pattern := range patterns {
		if pattern.MatchString(combined) {
			return false, "Potential injection detected"
		}
	}
	
	cmdPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(;|\|\||&&)`),
		regexp.MustCompile(`(?i)(` + "`" + `|\$\(|\$\{)`),
		regexp.MustCompile(`(?i)(wget|curl|nc|netcat|telnet|ssh|ftp)`),
		regexp.MustCompile(`(?i)(chmod|chown|useradd|passwd|sudo|su\s)`),
	}
	
	for _, pattern := range cmdPatterns {
		if pattern.MatchString(query) || pattern.MatchString(path) {
			return false, "Potential command injection detected"
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
	}
	
	ssrfRegex := regexp.MustCompile(`(?i)(192\.168\.|10\.|172\.(1[6-9]|2[0-9]|3[01])\.|169\.254\.|127\.)`)
	
	for _, pattern := range ssrfPatterns {
		if strings.Contains(query, pattern) {
			return false, "Potential SSRF attempt"
		}
	}
	
	if ssrfRegex.MatchString(query) {
		return false, "Potential SSRF attempt: internal network access"
	}
	
	ipv6Patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\[::1\]`),
		regexp.MustCompile(`(?i)\[::ffff:`),
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
	}
	
	for _, pattern := range metadataPatterns {
		if strings.Contains(query, pattern) {
			return false, "Potential SSRF attempt: cloud metadata endpoint"
		}
	}
	
	return true, ""
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
