package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type OWASPProtectionService struct {
	risks          map[string]*OWASPRisk
	validators     map[string]func(*http.Request) *OWASPCheckResult
	knownVulnerableVersions map[string][]string
	sensitiveHeaders        []string
	tamperPatterns          []*regexp.Regexp
	protectionConfig        *OWASPProtectionConfig
	auditLog               []*SecurityAuditEntry
	auditMu                sync.RWMutex
	reportGenerator         *ReportGenerator
}

type OWASPProtectionConfig struct {
	EnableA1BrokenAccessControl  bool
	EnableA2CryptographicFailures bool
	EnableA3Injection           bool
	EnableA4InsecureDesign      bool
	EnableA5SecurityMisconfig   bool
	EnableA6VulnerableComponents bool
	EnableA7AuthFailures       bool
	EnableA8DataIntegrityFailures bool
	EnableA9LoggingMonitoring    bool
	EnableA10SSRF              bool
	StrictMode                 bool
	LogViolations              bool
	BlockOnViolation           bool
}

type OWASPCheckResult struct {
	IsSafe     bool
	RiskID     string
	Severity   string
	Message    string
	Details    string
	Remediation string
}

type SecurityAuditEntry struct {
	Timestamp   time.Time
	RiskID      string
	ClientIP    string
	UserAgent   string
	RequestPath string
	Violation   string
	Blocked     bool
}

type ReportGenerator struct {
	reportData map[string]interface{}
	mu         sync.RWMutex
}

func NewOWASPProtectionService() *OWASPProtectionService {
	config := &OWASPProtectionConfig{
		EnableA1BrokenAccessControl:  true,
		EnableA2CryptographicFailures: true,
		EnableA3Injection:           true,
		EnableA4InsecureDesign:      true,
		EnableA5SecurityMisconfig:   true,
		EnableA6VulnerableComponents: true,
		EnableA7AuthFailures:       true,
		EnableA8DataIntegrityFailures: true,
		EnableA9LoggingMonitoring:    true,
		EnableA10SSRF:              true,
		StrictMode:                 false,
		LogViolations:             true,
		BlockOnViolation:           false,
	}

	service := &OWASPProtectionService{
		risks: map[string]*OWASPRisk{
			"A01": {
				ID:          "A01",
				Name:        "Broken Access Control",
				Description: "访问控制缺陷 - 限制不足或身份验证失败",
				Severity:    "Critical",
				Status:      "Active",
			},
			"A02": {
				ID:          "A02",
				Name:        "Cryptographic Failures",
				Description: "加密故障 - 敏感数据泄露",
				Severity:    "Critical",
				Status:      "Active",
			},
			"A03": {
				ID:          "A03",
				Name:        "Injection",
				Description: "注入攻击 - SQL、NoSQL、OS命令注入",
				Severity:    "Critical",
				Status:      "Active",
			},
			"A04": {
				ID:          "A04",
				Name:        "Insecure Design",
				Description: "不安全设计 - 架构和设计缺陷",
				Severity:    "High",
				Status:      "Active",
			},
			"A05": {
				ID:          "A05",
				Name:        "Security Misconfiguration",
				Description: "安全配置错误 - 默认配置、不完整配置",
				Severity:    "High",
				Status:      "Active",
			},
			"A06": {
				ID:          "A06",
				Name:        "Vulnerable and Outdated Components",
				Description: "脆弱和过期组件 - 使用已知漏洞的组件",
				Severity:    "High",
				Status:      "Active",
			},
			"A07": {
				ID:          "A07",
				Name:        "Identification and Authentication Failures",
				Description: "身份识别和认证失败 - 会话管理、凭证管理",
				Severity:    "Critical",
				Status:      "Active",
			},
			"A08": {
				ID:          "A08",
				Name:        "Software and Data Integrity Failures",
				Description: "软件和数据完整性故障 - 不安全的CI/CD、未验证更新",
				Severity:    "High",
				Status:      "Active",
			},
			"A09": {
				ID:          "A09",
				Name:        "Security Logging and Monitoring Failures",
				Description: "安全日志和监控故障 - 日志不足、响应延迟",
				Severity:    "Medium",
				Status:      "Active",
			},
			"A10": {
				ID:          "A10",
				Name:        "Server-Side Request Forgery",
				Description: "服务端请求伪造 - SSRF攻击",
				Severity:    "High",
				Status:      "Active",
			},
		},
		validators: make(map[string]func(*http.Request) *OWASPCheckResult),
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
		protectionConfig: config,
		auditLog:        make([]*SecurityAuditEntry, 0, 1000),
		reportGenerator: &ReportGenerator{
			reportData: make(map[string]interface{}),
		},
	}

	service.initValidators()

	return service
}

func (s *OWASPProtectionService) initValidators() {
	s.validators["A01"] = s.checkBrokenAccessControlEnhanced
	s.validators["A02"] = s.checkCryptographicFailuresEnhanced
	s.validators["A03"] = s.checkInjectionEnhanced
	s.validators["A04"] = s.checkInsecureDesignEnhanced
	s.validators["A05"] = s.checkSecurityMisconfigurationEnhanced
	s.validators["A06"] = s.checkVulnerableComponentsEnhanced
	s.validators["A07"] = s.checkAuthFailuresEnhanced
	s.validators["A08"] = s.checkDataIntegrityEnhanced
	s.validators["A09"] = s.checkLoggingFailuresEnhanced
	s.validators["A10"] = s.checkSSRFEnhanced
}

func (s *OWASPProtectionService) CheckAllRisks(r *http.Request) map[string]*OWASPCheckResult {
	results := make(map[string]*OWASPCheckResult)

	for id, validator := range s.validators {
		if s.shouldCheck(id) {
			result := validator(r)
			results[id] = result

			if !result.IsSafe {
				s.logViolation(r, id, result)
			}
		}
	}

	return results
}

func (s *OWASPProtectionService) shouldCheck(riskID string) bool {
	config := s.protectionConfig

	switch riskID {
	case "A01":
		return config.EnableA1BrokenAccessControl
	case "A02":
		return config.EnableA2CryptographicFailures
	case "A03":
		return config.EnableA3Injection
	case "A04":
		return config.EnableA4InsecureDesign
	case "A05":
		return config.EnableA5SecurityMisconfig
	case "A06":
		return config.EnableA6VulnerableComponents
	case "A07":
		return config.EnableA7AuthFailures
	case "A08":
		return config.EnableA8DataIntegrityFailures
	case "A09":
		return config.EnableA9LoggingMonitoring
	case "A10":
		return config.EnableA10SSRF
	default:
		return true
	}
}

func (s *OWASPProtectionService) logViolation(r *http.Request, riskID string, result *OWASPCheckResult) {
	if !s.protectionConfig.LogViolations {
		return
	}

	s.auditMu.Lock()
	defer s.auditMu.Unlock()

	entry := &SecurityAuditEntry{
		Timestamp:   time.Now(),
		RiskID:      riskID,
		ClientIP:    getClientIPFromRequest(r),
		UserAgent:   r.UserAgent(),
		RequestPath: r.URL.Path,
		Violation:   result.Message,
		Blocked:     s.protectionConfig.BlockOnViolation,
	}

	s.auditLog = append(s.auditLog, entry)

	if len(s.auditLog) > 10000 {
		s.auditLog = s.auditLog[len(s.auditLog)-10000:]
	}

	s.reportGenerator.mu.Lock()
	s.reportGenerator.reportData[riskID] = map[string]int{
		"violations": len(s.auditLog),
	}
	s.reportGenerator.mu.Unlock()
}

func (s *OWASPProtectionService) checkBrokenAccessControlEnhanced(r *http.Request) *OWASPCheckResult {
	path := r.URL.Path

	sensitivePaths := []string{
		"/admin", "/config", "/backup", "/.env", "/.git",
		"/api/v1/admin", "/api/admin", "/management", "/system",
		"/private", "/secret", "/keys", "/token", "/oauth",
		"/.well-known/acme-challenge", "/cgi-bin", "/phpmyadmin",
	}

	for _, p := range sensitivePaths {
		if strings.Contains(path, p) {
			if !s.hasValidAuthorization(r) {
				return &OWASPCheckResult{
					IsSafe:     false,
					RiskID:     "A01",
					Severity:   "Critical",
					Message:    "Access to sensitive path without proper authorization",
					Details:    fmt.Sprintf("Path: %s", p),
					Remediation: "Implement proper access control checks and require authentication",
				}
			}

			if !s.hasProperHTTPMethod(r) {
				return &OWASPCheckResult{
					IsSafe:     false,
					RiskID:     "A01",
					Severity:   "High",
					Message:    "HTTP method not allowed for sensitive endpoint",
					Details:    fmt.Sprintf("Path: %s, Method: %s", p, r.Method),
					Remediation: "Restrict HTTP methods to only those necessary",
				}
			}
		}
	}

	if strings.HasPrefix(path, "/api/") && r.Header.Get("Authorization") == "" {
		return &OWASPCheckResult{
			IsSafe:     false,
			RiskID:     "A01",
			Severity:   "High",
			Message:    "Missing authorization header for API endpoint",
			Details:    fmt.Sprintf("Path: %s", path),
			Remediation: "Require Authorization header for all API requests",
		}
	}

	return &OWASPCheckResult{
		IsSafe:   true,
		RiskID:   "A01",
		Severity: "None",
		Message:  "No access control violations detected",
	}
}

func (s *OWASPProtectionService) checkCryptographicFailuresEnhanced(r *http.Request) *OWASPCheckResult {
	issues := []string{}

	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
		issues = append(issues, "Insecure connection (not HTTPS)")
	}

	if r.TLS != nil && r.TLS.CipherSuite != 0 {
		if s.isWeakCipher(r.TLS.CipherSuite) {
			issues = append(issues, "Weak TLS cipher suite detected")
		}
	}

	if r.Header.Get("X-Content-Type-Options") == "" {
		issues = append(issues, "Missing X-Content-Type-Options header")
	}

	if r.Header.Get("Strict-Transport-Security") == "" {
		issues = append(issues, "Missing HSTS header")
	}

	if r.Header.Get("X-Frame-Options") == "" {
		issues = append(issues, "Missing X-Frame-Options header")
	}

	if r.Header.Get("X-XSS-Protection") == "" {
		issues = append(issues, "Missing X-XSS-Protection header")
	}

	if r.Header.Get("Content-Security-Policy") == "" {
		issues = append(issues, "Missing Content-Security-Policy header")
	}

	if len(issues) > 0 {
		return &OWASPCheckResult{
			IsSafe:     false,
			RiskID:     "A02",
			Severity:   "Critical",
			Message:    fmt.Sprintf("Cryptographic failures detected: %s", strings.Join(issues, "; ")),
			Details:    strings.Join(issues, "\n"),
			Remediation: "Implement proper TLS configuration and security headers",
		}
	}

	return &OWASPCheckResult{
		IsSafe:   true,
		RiskID:   "A02",
		Severity: "None",
		Message:  "No cryptographic failures detected",
	}
}

func (s *OWASPProtectionService) checkInjectionEnhanced(r *http.Request) *OWASPCheckResult {
	query := r.URL.RawQuery
	path := r.URL.Path
	body := s.getRequestBody(r)
	combined := query + path + body

	sqlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union\s+select|select\s+.*\s+from|insert\s+into|update\s+.*\s+set|delete\s+from|drop\s+table|alter\s+table)`),
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
			return &OWASPCheckResult{
				IsSafe:     false,
				RiskID:     "A03",
				Severity:   "Critical",
				Message:    "SQL injection attempt detected",
				Details:    fmt.Sprintf("Pattern matched: %s", pattern.String()),
				Remediation: "Use parameterized queries and input validation",
			}
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
	}

	for _, pattern := range xssPatterns {
		if pattern.MatchString(combined) {
			return &OWASPCheckResult{
				IsSafe:     false,
				RiskID:     "A03",
				Severity:   "Critical",
				Message:    "XSS attack attempt detected",
				Details:    fmt.Sprintf("Pattern matched: %s", pattern.String()),
				Remediation: "Sanitize and escape HTML output, use Content-Security-Policy",
			}
		}
	}

	cmdPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(;\s*|&&\s*|\|\|\s*|\|\s*)`),
		regexp.MustCompile(`(?i)(` + "`" + `|\$\(|\$\{|\beval\b)`),
		regexp.MustCompile(`(?i)(wget\s+|curl\s+|nc\s+|netcat\s+|telnet\s+|ssh\s+|ftp\s+)`),
		regexp.MustCompile(`(?i)(chmod\s+|chown\s+|useradd\s+|passwd\s+|sudo\s+|su\s+)`),
		regexp.MustCompile(`(?i)(/bin/bash|/bin/sh|/usr/bin/perl|/usr/bin/python)`),
		regexp.MustCompile(`(?i)(cat\s+|grep\s+|awk\s+|sed\s+|tail\s+|head\s+|more\s+|less\s+)`),
	}

	for _, pattern := range cmdPatterns {
		if pattern.MatchString(query) || pattern.MatchString(path) {
			return &OWASPCheckResult{
				IsSafe:     false,
				RiskID:     "A03",
				Severity:   "Critical",
				Message:    "Command injection attempt detected",
				Details:    fmt.Sprintf("Pattern matched: %s", pattern.String()),
				Remediation: "Validate and sanitize all user input, avoid shell commands",
			}
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
			return &OWASPCheckResult{
				IsSafe:     false,
				RiskID:     "A03",
				Severity:   "High",
				Message:    "Path traversal attempt detected",
				Details:    fmt.Sprintf("Pattern matched: %s", pattern.String()),
				Remediation: "Validate and sanitize file paths, use allowlists",
			}
		}
	}

	return &OWASPCheckResult{
		IsSafe:   true,
		RiskID:   "A03",
		Severity: "None",
		Message:  "No injection attacks detected",
	}
}

func (s *OWASPProtectionService) checkInsecureDesignEnhanced(r *http.Request) *OWASPCheckResult {
	path := r.URL.Path

	insecurePaths := []string{
		"/login", "/register", "/api/login", "/api/register",
	}

	for _, p := range insecurePaths {
		if strings.Contains(path, p) {
			if r.Method != "POST" {
				return &OWASPCheckResult{
					IsSafe:     false,
					RiskID:     "A04",
					Severity:   "High",
					Message:    "Security critical endpoint should use POST method",
					Details:    fmt.Sprintf("Path: %s, Method: %s", p, r.Method),
					Remediation: "Use POST method for authentication endpoints",
				}
			}

			contentType := r.Header.Get("Content-Type")
			if contentType != "application/json" && !strings.Contains(contentType, "application/x-www-form-urlencoded") {
				return &OWASPCheckResult{
					IsSafe:     false,
					RiskID:     "A04",
					Severity:   "Medium",
					Message:    "Missing proper content type for authentication endpoint",
					Details:    fmt.Sprintf("Path: %s, Content-Type: %s", p, contentType),
					Remediation: "Set proper Content-Type header (application/json)",
				}
			}
		}
	}

	return &OWASPCheckResult{
		IsSafe:   true,
		RiskID:   "A04",
		Severity: "None",
		Message:  "No insecure design issues detected",
	}
}

func (s *OWASPProtectionService) checkSecurityMisconfigurationEnhanced(r *http.Request) *OWASPCheckResult {
	issues := []string{}

	serverHeader := r.Header.Get("Server")
	if serverHeader != "" {
		if strings.Contains(serverHeader, "Apache") || strings.Contains(serverHeader, "nginx") {
			if strings.Contains(serverHeader, "/") {
				issues = append(issues, "Server version exposed in header")
			}
		}
	}

	xPoweredBy := r.Header.Get("X-Powered-By")
	if xPoweredBy != "" {
		issues = append(issues, "X-Powered-By header exposes technology stack")
	}

	if r.Header.Get("X-Debug") == "true" {
		issues = append(issues, "Debug mode enabled in production")
	}

	if r.Header.Get("Server") == "" && r.Header.Get("X-Powered-By") == "" {
		issues = append(issues, "No server identification headers (may be intentional)")
	}

	if len(issues) > 0 {
		return &OWASPCheckResult{
			IsSafe:     false,
			RiskID:     "A05",
			Severity:   "High",
			Message:    fmt.Sprintf("Security misconfigurations detected: %s", strings.Join(issues, "; ")),
			Details:    strings.Join(issues, "\n"),
			Remediation: "Remove version disclosure headers, disable debug mode in production",
		}
	}

	return &OWASPCheckResult{
		IsSafe:   true,
		RiskID:   "A05",
		Severity: "None",
		Message:  "No security misconfigurations detected",
	}
}

func (s *OWASPProtectionService) checkVulnerableComponentsEnhanced(r *http.Request) *OWASPCheckResult {
	userAgent := r.UserAgent()

	for component, versions := range s.knownVulnerableVersions {
		if strings.Contains(userAgent, component) {
			for _, version := range versions {
				if strings.Contains(userAgent, component+"/"+version) ||
					strings.Contains(userAgent, version) && strings.Contains(userAgent, component) {
					return &OWASPCheckResult{
						IsSafe:     false,
						RiskID:     "A06",
						Severity:   "High",
						Message:    fmt.Sprintf("Vulnerable component detected: %s %s", component, version),
						Details:    fmt.Sprintf("User-Agent: %s", userAgent),
						Remediation: "Update component to latest secure version",
					}
				}
			}
		}
	}

	return &OWASPCheckResult{
		IsSafe:   true,
		RiskID:   "A06",
		Severity: "None",
		Message:  "No vulnerable components detected",
	}
}

func (s *OWASPProtectionService) checkAuthFailuresEnhanced(r *http.Request) *OWASPCheckResult {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" && !strings.HasSuffix(r.URL.Path, "/login") && !strings.HasSuffix(r.URL.Path, "/public") {
		return &OWASPCheckResult{
			IsSafe:     false,
			RiskID:     "A07",
			Severity:   "Critical",
			Message:    "Missing authentication for protected resource",
			Details:    fmt.Sprintf("Path: %s", r.URL.Path),
			Remediation: "Require proper authentication for protected endpoints",
		}
	}

	return &OWASPCheckResult{
		IsSafe:   true,
		RiskID:   "A07",
		Severity: "None",
		Message:  "No authentication failures detected",
	}
}

func (s *OWASPProtectionService) checkDataIntegrityEnhanced(r *http.Request) *OWASPCheckResult {
	query := r.URL.RawQuery
	body := s.getRequestBody(r)

	for _, pattern := range s.tamperPatterns {
		if pattern.MatchString(query) || pattern.MatchString(body) {
			return &OWASPCheckResult{
				IsSafe:     false,
				RiskID:     "A08",
				Severity:   "High",
				Message:    "Potential data tampering attempt detected",
				Details:    fmt.Sprintf("Pattern: %s", pattern.String()),
				Remediation: "Implement integrity checks and input validation",
			}
		}
	}

	return &OWASPCheckResult{
		IsSafe:   true,
		RiskID:   "A08",
		Severity: "None",
		Message:  "No data integrity issues detected",
	}
}

func (s *OWASPProtectionService) checkLoggingFailuresEnhanced(r *http.Request) *OWASPCheckResult {
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
			return &OWASPCheckResult{
				IsSafe:     false,
				RiskID:     "A09",
				Severity:   "Medium",
				Message:    "Missing X-Request-ID header for critical operation",
				Details:    fmt.Sprintf("Path: %s", path),
				Remediation: "Add request tracking ID for security audit trail",
			}
		}
	}

	return &OWASPCheckResult{
		IsSafe:   true,
		RiskID:   "A09",
		Severity: "None",
		Message:  "No logging issues detected",
	}
}

func (s *OWASPProtectionService) checkSSRFEnhanced(r *http.Request) *OWASPCheckResult {
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
			return &OWASPCheckResult{
				IsSafe:     false,
				RiskID:     "A10",
				Severity:   "High",
				Message:    fmt.Sprintf("Potential SSRF attempt: %s", pattern),
				Details:    fmt.Sprintf("Query: %s", query),
				Remediation: "Validate and sanitize URLs, use allowlists for allowed domains",
			}
		}
	}

	if ssrfRegex.MatchString(query) {
		return &OWASPCheckResult{
			IsSafe:     false,
			RiskID:     "A10",
			Severity:   "High",
			Message:    "Potential SSRF attempt: internal network access",
			Details:    fmt.Sprintf("Query: %s", query),
			Remediation: "Block access to internal network ranges",
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
			return &OWASPCheckResult{
				IsSafe:     false,
				RiskID:     "A10",
				Severity:   "Critical",
				Message:    fmt.Sprintf("Potential SSRF attempt: cloud metadata endpoint (%s)", pattern),
				Details:    fmt.Sprintf("Query: %s", query),
				Remediation: "Block access to cloud metadata endpoints",
			}
		}
	}

	return &OWASPCheckResult{
		IsSafe:   true,
		RiskID:   "A10",
		Severity: "None",
		Message:  "No SSRF attempts detected",
	}
}

func (s *OWASPProtectionService) hasValidAuthorization(r *http.Request) bool {
	if r.Header.Get("Authorization") != "" {
		return true
	}
	if r.Header.Get("X-Admin-Token") != "" {
		return true
	}
	return false
}

func (s *OWASPProtectionService) hasProperHTTPMethod(r *http.Request) bool {
	sensitivePaths := []string{"/admin", "/config", "/backup", "/.env"}
	for _, p := range sensitivePaths {
		if strings.Contains(r.URL.Path, p) {
			return r.Method == "GET" || r.Method == "POST"
		}
	}
	return true
}

func (s *OWASPProtectionService) isWeakCipher(cipher uint16) bool {
	weakCiphers := []uint16{
		0x0004, 0x0005, 0x000A, 0x0009, 0x0015, 0x0016,
		0x0013, 0x0014, 0x002F, 0x0035, 0x003D, 0x003C,
		0x003B, 0x003A, 0xC007, 0xC008, 0xC011, 0xC012,
	}
	for _, weak := range weakCiphers {
		if cipher == weak {
			return true
		}
	}
	return false
}

func (s *OWASPProtectionService) getRequestBody(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return string(bodyBytes)
}

func getClientIPFromRequest(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	return r.RemoteAddr
}

func (s *OWASPProtectionService) GenerateComplianceReport() *OWASPComplianceReport {
	s.auditMu.RLock()
	defer s.auditMu.RUnlock()

	report := &OWASPComplianceReport{
		GeneratedAt: time.Now(),
		TotalChecks:  len(s.validators),
		Checks:       make(map[string]*RiskCheck),
	}

	violationsByRisk := make(map[string]int)
	for _, entry := range s.auditLog {
		violationsByRisk[entry.RiskID]++
	}

	for id, risk := range s.risks {
		report.Checks[id] = &RiskCheck{
			Risk:       risk,
			Violations: violationsByRisk[id],
			IsCompliant: violationsByRisk[id] == 0,
		}
	}

	report.ComplianceScore = s.calculateComplianceScore(report)
	report.Recommendations = s.generateRecommendations(report)

	return report
}

type OWASPComplianceReport struct {
	GeneratedAt      time.Time
	TotalChecks      int
	Checks           map[string]*RiskCheck
	ComplianceScore  float64
	Recommendations  []string
}

type RiskCheck struct {
	Risk       *OWASPRisk
	Violations int
	IsCompliant bool
}

func (s *OWASPProtectionService) calculateComplianceScore(report *OWASPComplianceReport) float64 {
	compliantCount := 0
	for _, check := range report.Checks {
		if check.IsCompliant {
			compliantCount++
		}
	}
	return float64(compliantCount) / float64(report.TotalChecks) * 100
}

func (s *OWASPProtectionService) generateRecommendations(report *OWASPComplianceReport) []string {
	var recommendations []string

	for id, check := range report.Checks {
		if !check.IsCompliant {
			recommendations = append(recommendations,
				fmt.Sprintf("Address %s violations in %s (%s)",
					check.Risk.Name, id, check.Risk.Description))
		}
	}

	if report.ComplianceScore < 100 {
		recommendations = append(recommendations,
			"Review and remediate all security violations to achieve full compliance")
	}

	return recommendations
}

func (s *OWASPProtectionService) GetAuditLog() []*SecurityAuditEntry {
	s.auditMu.RLock()
	defer s.auditMu.RUnlock()

	logCopy := make([]*SecurityAuditEntry, len(s.auditLog))
	copy(logCopy, s.auditLog)
	return logCopy
}

func (s *OWASPProtectionService) ExportReport(format string) ([]byte, error) {
	report := s.GenerateComplianceReport()

	switch format {
	case "json":
		return json.MarshalIndent(report, "", "  ")
	case "html":
		return s.generateHTMLReport(report)
	default:
		return json.MarshalIndent(report, "", "  ")
	}
}

func (s *OWASPProtectionService) generateHTMLReport(report *OWASPComplianceReport) ([]byte, error) {
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>OWASP Top 10 Security Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; }
        .score { font-size: 24px; font-weight: bold; color: {{if gt .ComplianceScore 80}}green{{else}}red{{end}}; }
        .risk { border: 1px solid #ddd; padding: 10px; margin: 10px 0; border-radius: 5px; }
        .critical { border-left: 5px solid red; }
        .high { border-left: 5px solid orange; }
        .medium { border-left: 5px solid yellow; }
        .low { border-left: 5px solid green; }
    </style>
</head>
<body>
    <h1>OWASP Top 10 Security Report</h1>
    <p>Generated: {{.GeneratedAt}}</p>
    <p>Compliance Score: <span class="score">{{printf "%.2f" .ComplianceScore}}%</span></p>
    <h2>Risk Checks</h2>
    {{range .Checks}}
    <div class="risk {{.Risk.Severity}}">
        <h3>{{.Risk.ID}}: {{.Risk.Name}}</h3>
        <p>{{.Risk.Description}}</p>
        <p>Violations: {{.Violations}}</p>
        <p>Status: {{if .IsCompliant}}<strong>Compliant</strong>{{else}}<strong>Non-Compliant</strong>{{end}}</p>
    </div>
    {{end}}
</body>
</html>`

	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, report)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *OWASPProtectionService) SetProtectionConfig(config *OWASPProtectionConfig) {
	s.protectionConfig = config
}

func (s *OWASPProtectionService) EnableStrictMode() {
	s.protectionConfig.StrictMode = true
	s.protectionConfig.BlockOnViolation = true
}

func (s *OWASPProtectionService) DisableStrictMode() {
	s.protectionConfig.StrictMode = false
	s.protectionConfig.BlockOnViolation = false
}
