package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

var (
	owaspService *service.OWASPService
	owaspOnce    = &sync.Once{}
)

type OWASPConfig struct {
	Enabled           bool
	EnforceHeaders    bool
	EnforceHTTPS     bool
	BlockNonCompliant bool
	EnableSQLInjection bool
	EnableXSSProtection bool
	EnableSSRFProtection bool
	EnablePathTraversal bool
	EnableCmdInjection  bool
	EnableLDAPInjection bool
	EnableNoSQLInjection bool
	EnableXMLInjection   bool
	EnableHeaderValidation bool
	EnableBodyValidation   bool
}

var DefaultOWASPConfig = OWASPConfig{
	Enabled:               true,
	EnforceHeaders:        true,
	EnforceHTTPS:          false,
	BlockNonCompliant:     false,
	EnableSQLInjection:    true,
	EnableXSSProtection:   true,
	EnableSSRFProtection:  true,
	EnablePathTraversal:   true,
	EnableCmdInjection:    true,
	EnableLDAPInjection:   true,
	EnableNoSQLInjection:  true,
	EnableXMLInjection:    true,
	EnableHeaderValidation: true,
	EnableBodyValidation:  true,
}

func initOWASPService() {
	owaspOnce.Do(func() {
		owaspService = service.NewOWASPService()
	})
}

func OWASPSecurityMiddleware(config ...OWASPConfig) gin.HandlerFunc {
	initOWASPService()

	cfg := DefaultOWASPConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		if cfg.EnforceHeaders {
			setSecurityHeaders(c.Writer)
		}

		if cfg.EnforceHTTPS {
			if c.Request.TLS == nil && c.GetHeader("X-Forwarded-Proto") != "https" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "HTTPS required",
					"code":  http.StatusForbidden,
				})
				return
			}
		}

		securityResult := checkOWASPCompliance(c, cfg)

		c.Set("owasp_compliance", securityResult)
		c.Set("security_scan_id", generateSecurityScanID())

		if securityResult["blocked"].(bool) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":      securityResult["error"].(string),
				"code":       "SECURITY_VIOLATION",
				"category":   securityResult["category"].(string),
				"scan_id":    securityResult["scan_id"].(string),
			})
			return
		}

		if cfg.BlockNonCompliant && !securityResult["compliant"].(bool) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":      "OWASP compliance check failed",
				"code":       http.StatusBadRequest,
				"compliance": securityResult,
			})
			return
		}

		c.Set("owasp_service", owaspService)
		c.Next()
	}
}

func checkOWASPCompliance(c *gin.Context, cfg OWASPConfig) map[string]interface{} {
	result := map[string]interface{}{
		"compliant": true,
		"blocked":   false,
		"scan_id":   generateSecurityScanID(),
		"timestamp": time.Now().Unix(),
	}
	checks := make(map[string]bool)

	req := c.Request

	if cfg.EnableSQLInjection {
		if blocked, reason := checkSQLInjection(req); blocked {
			result["compliant"] = false
			result["blocked"] = true
			result["error"] = reason
			result["category"] = "A03:2021-Injection"
			return result
		}
		checks["sql_injection"] = true
	}

	if cfg.EnableXSSProtection {
		if blocked, reason := checkXSSAttack(req); blocked {
			result["compliant"] = false
			result["blocked"] = true
			result["error"] = reason
			result["category"] = "A03:2021-Injection"
			return result
		}
		checks["xss"] = true
	}

	if cfg.EnableSSRFProtection {
		if blocked, reason := checkSSRFAttack(req); blocked {
			result["compliant"] = false
			result["blocked"] = true
			result["error"] = reason
			result["category"] = "A10:2021-SSRF"
			return result
		}
		checks["ssrf"] = true
	}

	if cfg.EnablePathTraversal {
		if blocked, reason := checkPathTraversal(req); blocked {
			result["compliant"] = false
			result["blocked"] = true
			result["error"] = reason
			result["category"] = "A01:2021-Broken Access Control"
			return result
		}
		checks["path_traversal"] = true
	}

	if cfg.EnableCmdInjection {
		if blocked, reason := checkCommandInjection(req); blocked {
			result["compliant"] = false
			result["blocked"] = true
			result["error"] = reason
			result["category"] = "A03:2021-Injection"
			return result
		}
		checks["cmd_injection"] = true
	}

	if cfg.EnableLDAPInjection {
		if blocked, reason := checkLDAPInjection(req); blocked {
			result["compliant"] = false
			result["blocked"] = true
			result["error"] = reason
			result["category"] = "A03:2021-Injection"
			return result
		}
		checks["ldap_injection"] = true
	}

	if cfg.EnableNoSQLInjection {
		if blocked, reason := checkNoSQLInjection(req); blocked {
			result["compliant"] = false
			result["blocked"] = true
			result["error"] = reason
			result["category"] = "A03:2021-Injection"
			return result
		}
		checks["nosql_injection"] = true
	}

	if cfg.EnableXMLInjection {
		if blocked, reason := checkXMLInjection(req); blocked {
			result["compliant"] = false
			result["blocked"] = true
			result["error"] = reason
			result["category"] = "A03:2021-Injection"
			return result
		}
		checks["xml_injection"] = true
	}

	if cfg.EnableHeaderValidation {
		if blocked, _ := checkSecurityHeaders(req); blocked {
			checks["security_headers"] = false
		} else {
			checks["security_headers"] = true
		}
	}

	compliance := owaspService.CheckCompliance(req)
	if compliance["compliant"] != nil {
		if compl, ok := compliance["compliant"].(bool); ok {
			checks["owasp_service"] = compl
		}
	}

	passedChecks := 0
	totalChecks := len(checks)
	for _, v := range checks {
		if v {
			passedChecks++
		}
	}
	result["checks"] = checks
	result["score"] = float64(passedChecks) / float64(totalChecks) * 100
	result["passed"] = passedChecks
	result["total"] = totalChecks

	return result
}

func checkSQLInjection(req *http.Request) (bool, string) {
	combined := getRequestContent(req)

	sqlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(\bUNION\b.*\bSELECT\b|\bSELECT\b.*\bFROM\b|\bINSERT\b.*\bINTO\b|\bUPDATE\b.*\bSET\b|\bDELETE\b.*\bFROM\b|\bDROP\b.*\bTABLE\b)`),
		regexp.MustCompile(`(?i)(\bEXEC\b|\bEXECUTE\b|\bXP_\b|\bSP_\b|\bEXEC\s*\()`),
		regexp.MustCompile(`(?i)(\bSLEEP\b|\bBENCHMARK\b|\bPG_SLEEP\b|\bWAITFOR\b|\bDELAY\b)`),
		regexp.MustCompile(`(?i)(\bLOAD_FILE\b|\bINTO\s+OUTFILE\b|\bINTO\s+DUMPFILE\b|\bOUTFILE\b)`),
		regexp.MustCompile(`(?i)(['"]\s*OR\s*['"]?\s*\d+\s*=\s*\d|\bOR\b\s+1\s*=\s*1|\bAND\b\s+1\s*=\s*1)`),
		regexp.MustCompile(`(?i)(\bINFORMATION_SCHEMA\b|\bSYS\.TABLES\b|\bPG_CATALOG\b|\bSYSCAT\b)`),
		regexp.MustCompile(`(?i)(--\s*$|/\*.*\*/|#\s*$)`),
		regexp.MustCompile(`(?i)(\bHAVING\b.*\b=\b|\bWHERE\b.*['"]\s*\bOR\b\s*['"])`),
		regexp.MustCompile(`(?i)(\bCASE\b.*\bWHEN\b.*\bTHEN\b)`),
		regexp.MustCompile(`(?i)(\bEXTRACTVALUE\b|\bUPDATEXML\b|\bXMLTYPE\b)`),
	}

	for _, pattern := range sqlPatterns {
		if pattern.MatchString(combined) {
			return true, "SQL injection attempt detected: " + pattern.String()
		}
	}

	return false, ""
}

func checkXSSAttack(req *http.Request) (bool, string) {
	combined := getRequestContent(req)

	xssPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)<script[^>]*\/?>`),
		regexp.MustCompile(`(?i)javascript\s*:`),
		regexp.MustCompile(`(?i)vbscript\s*:`),
		regexp.MustCompile(`(?i)data\s*:`),
		regexp.MustCompile(`(?i)\bon\w+\s*=`),
		regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
		regexp.MustCompile(`(?i)<object[^>]*>.*?</object>`),
		regexp.MustCompile(`(?i)<embed[^>]*>`),
		regexp.MustCompile(`(?i)<applet[^>]*>.*?</applet>`),
		regexp.MustCompile(`(?i)<svg[^>]*>.*?</svg>`),
		regexp.MustCompile(`(?i)<math[^>]*>.*?</math>`),
		regexp.MustCompile(`(?i)<details[^>]*.*?</details>`),
		regexp.MustCompile(`(?i)<marquee[^>]*>.*?</marquee>`),
		regexp.MustCompile(`(?i)<xmp[^>]*>.*?</xmp>`),
		regexp.MustCompile(`(?i)<plaintext[^>]*>.*?</plaintext>`),
		regexp.MustCompile(`(?i)\$\{.*?\}`),
		regexp.MustCompile(`(?i)\{.*?_.*?\}`),
		regexp.MustCompile(`(?i)eval\s*\(`),
		regexp.MustCompile(`(?i)setTimeout\s*\(`),
		regexp.MustCompile(`(?i)setInterval\s*\(`),
		regexp.MustCompile(`(?i)document\.cookie`),
		regexp.MustCompile(`(?i)document\.write`),
		regexp.MustCompile(`(?i)innerHTML\s*=`),
		regexp.MustCompile(`(?i)outerHTML\s*=`),
	}

	for _, pattern := range xssPatterns {
		if pattern.MatchString(combined) {
			return true, "XSS attack attempt detected"
		}
	}

	return false, ""
}

func checkSSRFAttack(req *http.Request) (bool, string) {
	query := req.URL.RawQuery
	body := getRequestBody(req)
	combined := query + body

	ssrfPatterns := []string{
		"http://127.0.0.1",
		"http://localhost",
		"http://0.0.0.0",
		"http://[::]",
		"http://[::1]",
		"file://",
		"gopher://",
		"dict://",
		"ftp://",
		"http://169.254.",
		"http://127.1.",
	}

	ssrfRegex := regexp.MustCompile(`(?i)(192\.168\.|10\.|172\.(1[6-9]|2[0-9]|3[01])\.|169\.254\.|127\.|0\.0\.0\.0|224\.|240\.)`)

	for _, pattern := range ssrfPatterns {
		if strings.Contains(combined, pattern) {
			return true, "Potential SSRF attempt: internal network access"
		}
	}

	if ssrfRegex.MatchString(combined) {
		return true, "Potential SSRF attempt: private IP range detected"
	}

	metadataEndpoints := []string{
		"metadata.google.internal",
		"metadata.azure.com",
		"169.254.169.254",
		"metadata.openstack.org",
		"metadata.youstruct.com",
	}

	for _, endpoint := range metadataEndpoints {
		if strings.Contains(combined, endpoint) {
			return true, "Potential SSRF attempt: cloud metadata endpoint"
		}
	}

	return false, ""
}

func checkPathTraversal(req *http.Request) (bool, string) {
	combined := req.URL.Path + req.URL.RawQuery

	pathTraversalPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(\.\./|\.\.)`),
		regexp.MustCompile(`(?i)(%2e%2e|%2e)`),
		regexp.MustCompile(`(?i)(%2f%2f|%2f|%5c|%5c)`),
		regexp.MustCompile(`(?i)(\.\.%2f|\.\.%5c)`),
		regexp.MustCompile(`(?i)(/etc/passwd|/etc/shadow|/root/\.ssh)`),
		regexp.MustCompile(`(?i)(/\.git/config|\.git\/HEAD|\.git\/config)`),
		regexp.MustCompile(`(?i)(win.ini|boot.ini|autoexec.bat)`),
		regexp.MustCompile(`(?i)(/var/log/|/var/tmp/|/tmp/)`),
	}

	for _, pattern := range pathTraversalPatterns {
		if pattern.MatchString(combined) {
			return true, "Path traversal attempt detected"
		}
	}

	return false, ""
}

func checkCommandInjection(req *http.Request) (bool, string) {
	combined := req.URL.Path + req.URL.RawQuery

	cmdPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(;\s*|&&\s*|\|\|\s*|\|\s*|>\s*|<\s*)`),
		regexp.MustCompile(`(?i)(` + "`" + `|\$\(|\$\{|\beval\b|\bexec\b)`),
		regexp.MustCompile(`(?i)(wget\s+|curl\s+|nc\s+|netcat\s+|telnet\s+|ssh\s+|ftp\s+)`),
		regexp.MustCompile(`(?i)(chmod\s+|chown\s+|useradd\s+|passwd\s+|sudo\s+|su\b)`),
		regexp.MustCompile(`(?i)(/bin/bash|/bin/sh|/usr/bin/perl|/usr/bin/python)`),
		regexp.MustCompile(`(?i)(rm\s+-rf|mkfs\.|dd\s+if=)`),
		regexp.MustCompile(`(?i)(nohup\s+|fork\s+|system\s*\()`),
		regexp.MustCompile(`(?i)(curl\s+-s\s+|wget\s+-O\s+)`),
		regexp.MustCompile(`(?i)(&&|\|\|)\s*rm\s+`),
	}

	for _, pattern := range cmdPatterns {
		if pattern.MatchString(combined) {
			return true, "Command injection attempt detected"
		}
	}

	return false, ""
}

func checkLDAPInjection(req *http.Request) (bool, string) {
	combined := getRequestContent(req)

	ldapPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(\*|\(|\)||\bn\b|\bOr\b|\bAnd\b|\bNot\b|\bMemberOf\b)`),
		regexp.MustCompile(`(?i)(\buserPassword\b.*\{)`),
		regexp.MustCompile(`(?i)(\.\.|\+|\||\&|\!|~)`),
		regexp.MustCompile(`(?i)(\badmin\b.*\bpwd\b|\broot\b.*\bpass\b)`),
		regexp.MustCompile(`(?i)(\*\)`),
		regexp.MustCompile(`(?i)(\(\)|\(\*\)`),
	}

	for _, pattern := range ldapPatterns {
		if pattern.MatchString(combined) {
			return true, "Potential LDAP injection detected"
		}
	}

	return false, ""
}

func checkNoSQLInjection(req *http.Request) (bool, string) {
	combined := getRequestContent(req)

	nosqlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(\$where|\$eval|\$function)`),
		regexp.MustCompile(`(?i)(\$ne|\$eq|\$lt|\$gt|\$lte|\$gte)`),
		regexp.MustCompile(`(?i)(\$in|\$nin|\$exists|\$regex)`),
		regexp.MustCompile(`(?i)(\$or|\$and|\$not|\$nor)`),
		regexp.MustCompile(`(?i)(\$regex.*\$options)`),
		regexp.MustCompile(`(?i)(\bsleep\b|\btimeout\b|\bdelay\b)`),
		regexp.MustCompile(`(?i)(\$where.*function|\$where.*\(\))`),
	}

	for _, pattern := range nosqlPatterns {
		if pattern.MatchString(combined) {
			return true, "Potential NoSQL injection detected"
		}
	}

	return false, ""
}

func checkXMLInjection(req *http.Request) (bool, string) {
	body := getRequestBody(req)

	if !strings.Contains(body, "<") && !strings.Contains(body, "xml") {
		return false, ""
	}

	xmlPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)<!DOCTYPE\s+html`),
		regexp.MustCompile(`(?i)<!ENTITY\s+\w+\s+SYSTEM`),
		regexp.MustCompile(`(?i)<!ENTITY\s+\w+\s+PUBLIC`),
		regexp.MustCompile(`(?i)<!\[CDATA\[`),
		regexp.MustCompile(`(?i)SYSTEM\s*"file:`),
		regexp.MustCompile(`(?i)PUBLIC\s*"\s*"file:`),
		regexp.MustCompile(`(?i)xmlns\s*=\s*"http://`),
		regexp.MustCompile(`(?i)<?xml-stylesheet`),
		regexp.MustCompile(`(?i)<\?xml\s+version.*\?>`),
	}

	for _, pattern := range xmlPatterns {
		if pattern.MatchString(body) {
			return true, "Potential XML injection detected"
		}
	}

	return false, ""
}

func checkSecurityHeaders(req *http.Request) (bool, string) {
	missingHeaders := []string{}

	if req.TLS != nil && req.Header.Get("Strict-Transport-Security") == "" {
		missingHeaders = append(missingHeaders, "Strict-Transport-Security")
	}

	if req.Header.Get("X-Content-Type-Options") == "" {
		missingHeaders = append(missingHeaders, "X-Content-Type-Options")
	}

	if req.Header.Get("X-Frame-Options") == "" {
		missingHeaders = append(missingHeaders, "X-Frame-Options")
	}

	if len(missingHeaders) > 2 {
		return true, "Missing critical security headers"
	}

	return false, ""
}

func getRequestContent(req *http.Request) string {
	query := req.URL.RawQuery
	path := req.URL.Path
	body := getRequestBody(req)
	return query + path + body
}

func getRequestBody(req *http.Request) string {
	if req.Body == nil {
		return ""
	}
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return ""
	}
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return string(bodyBytes)
}

func generateSecurityScanID() string {
	timestamp := time.Now().UnixNano()
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d-%d", timestamp, time.Now().Unix())))
	return fmt.Sprintf("scan_%s", hex.EncodeToString(hash[:])[:16])
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Content-Security-Policy", "default-src 'self'")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
}

func GetOWASPService() *service.OWASPService {
	initOWASPService()
	return owaspService
}
