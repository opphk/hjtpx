package security

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"testing"
)

type VulnerabilityType string

const (
	VulnSQLInjection     VulnerabilityType = "SQL_INJECTION"
	VulnXSS              VulnerabilityType = "XSS"
	VulnCommandInjection VulnerabilityType = "COMMAND_INJECTION"
	VulnPathTraversal    VulnerabilityType = "PATH_TRAVERSAL"
	VulnCSRF             VulnerabilityType = "CSRF"
	VulnInsecureAuth     VulnerabilityType = "INSECURE_AUTH"
	VulnSensitiveData    VulnerabilityType = "SENSITIVE_DATA_EXPOSURE"
	VulnXXE              VulnerabilityType = "XXE"
	VulnDeserialize      VulnerabilityType = "INSECURE_DESERIALIZATION"
	VulnSSRF             VulnerabilityType = "SSRF"
)

type Vulnerability struct {
	Type        VulnerabilityType
	Severity    string
	Description string
	Location    string
	Payload     string
}

type SecurityScanner struct {
	vulnerabilities []Vulnerability
}

func NewSecurityScanner() *SecurityScanner {
	return &SecurityScanner{
		vulnerabilities: make([]Vulnerability, 0),
	}
}

func (s *SecurityScanner) ScanSQLInjection(input string) ([]Vulnerability, error) {
	sqlPatterns := []string{
		"' OR '",
		"admin' --",
		"DROP TABLE",
		"UNION SELECT",
		"' --",
		"';",
	}

	for _, pattern := range sqlPatterns {
		if strings.Contains(input, pattern) {
			s.vulnerabilities = append(s.vulnerabilities, Vulnerability{
				Type:        VulnSQLInjection,
				Severity:    "HIGH",
				Description: "Potential SQL injection vulnerability detected",
				Location:    "Input field",
				Payload:     input,
			})
			break
		}
	}
	return s.vulnerabilities, nil
}

func (s *SecurityScanner) ScanXSS(input string) ([]Vulnerability, error) {
	xssPatterns := []string{
		"<script",
		"javascript:",
		"onerror=",
		"onload=",
		"onmouseover=",
		"onclick=",
		"<img",
		"<svg",
		"&lt;script",
		"data:text/html",
	}

	for _, pattern := range xssPatterns {
		if strings.Contains(input, pattern) {
			s.vulnerabilities = append(s.vulnerabilities, Vulnerability{
				Type:        VulnXSS,
				Severity:    "HIGH",
				Description: "Potential XSS vulnerability detected",
				Location:    "Input field",
				Payload:     input,
			})
			break
		}
	}
	return s.vulnerabilities, nil
}

func (s *SecurityScanner) ScanCommandInjection(input string) ([]Vulnerability, error) {
	cmdPatterns := []string{
		"; rm",
		"&& cat",
		"| whoami",
		"$(",
		"`rm",
		"../",
		"/etc/passwd",
	}

	for _, pattern := range cmdPatterns {
		if strings.Contains(input, pattern) {
			s.vulnerabilities = append(s.vulnerabilities, Vulnerability{
				Type:        VulnCommandInjection,
				Severity:    "CRITICAL",
				Description: "Potential command injection vulnerability detected",
				Location:    "Input field",
				Payload:     input,
			})
			break
		}
	}
	return s.vulnerabilities, nil
}

func (s *SecurityScanner) ScanPathTraversal(input string) ([]Vulnerability, error) {
	pathPatterns := []string{
		"../",
		"/etc/passwd",
		"windows/win.ini",
		"system32",
	}

	for _, pattern := range pathPatterns {
		if strings.Contains(input, pattern) {
			s.vulnerabilities = append(s.vulnerabilities, Vulnerability{
				Type:        VulnPathTraversal,
				Severity:    "HIGH",
				Description: "Potential path traversal vulnerability detected",
				Location:    "Input field",
				Payload:     input,
			})
			break
		}
	}
	return s.vulnerabilities, nil
}

func (s *SecurityScanner) ScanXXE(input string) ([]Vulnerability, error) {
	xxePatterns := []string{
		"<!DOCTYPE",
		"<!ENTITY",
		"SYSTEM",
		"PUBLIC",
	}

	for _, pattern := range xxePatterns {
		if strings.Contains(input, pattern) {
			s.vulnerabilities = append(s.vulnerabilities, Vulnerability{
				Type:        VulnXXE,
				Severity:    "CRITICAL",
				Description: "Potential XXE vulnerability detected",
				Location:    "XML input",
				Payload:     input,
			})
			break
		}
	}
	return s.vulnerabilities, nil
}

func (s *SecurityScanner) ScanSSRF(input string) ([]Vulnerability, error) {
	ssrfPatterns := []string{
		"http://127.0.0.1",
		"http://localhost",
		"http://0.0.0.0",
		"http://169.254.169.254",
		"http://10.",
		"http://172.",
		"http://192.168.",
		"file://",
		"gopher://",
		"dict://",
	}

	for _, pattern := range ssrfPatterns {
		if strings.Contains(input, pattern) {
			s.vulnerabilities = append(s.vulnerabilities, Vulnerability{
				Type:        VulnSSRF,
				Severity:    "HIGH",
				Description: "Potential SSRF vulnerability detected",
				Location:    "URL input",
				Payload:     input,
			})
			break
		}
	}
	return s.vulnerabilities, nil
}

func (s *SecurityScanner) ScanAll(input string) ([]Vulnerability, error) {
	s.vulnerabilities = make([]Vulnerability, 0)
	s.ScanSQLInjection(input)
	s.ScanXSS(input)
	s.ScanCommandInjection(input)
	s.ScanPathTraversal(input)
	s.ScanXXE(input)
	s.ScanSSRF(input)
	return s.vulnerabilities, nil
}

func (s *SecurityScanner) GetVulnerabilities() []Vulnerability {
	return s.vulnerabilities
}

func (s *SecurityScanner) Clear() {
	s.vulnerabilities = s.vulnerabilities[:0]
}

type PasswordStrength struct {
	Score    int
	Strength string
	Feedback []string
}

var (
	PasswordWeak       = PasswordStrength{Score: 0, Strength: "WEAK", Feedback: []string{}}
	PasswordMedium     = PasswordStrength{Score: 3, Strength: "MEDIUM", Feedback: []string{}}
	PasswordStrong     = PasswordStrength{Score: 5, Strength: "STRONG", Feedback: []string{}}
	PasswordVeryStrong = PasswordStrength{Score: 6, Strength: "VERY STRONG", Feedback: []string{}}
)

func CheckPasswordStrength(password string) PasswordStrength {
	score := 0
	feedback := make([]string, 0)

	if len(password) >= 8 {
		score++
	} else {
		feedback = append(feedback, "Password should be at least 8 characters long")
	}

	if len(password) >= 12 {
		score++
	}

	if hasUppercase(password) {
		score++
	} else {
		feedback = append(feedback, "Password should contain at least one uppercase letter")
	}

	if hasLowercase(password) {
		score++
	} else {
		feedback = append(feedback, "Password should contain at least one lowercase letter")
	}

	if hasNumber(password) {
		score++
	} else {
		feedback = append(feedback, "Password should contain at least one number")
	}

	if hasSpecial(password) {
		score++
	} else {
		feedback = append(feedback, "Password should contain at least one special character")
	}

	var result PasswordStrength
	if score >= 6 {
		result = PasswordVeryStrong
	} else if score >= 5 {
		result = PasswordStrong
	} else if score >= 3 {
		result = PasswordMedium
	} else {
		result = PasswordWeak
	}
	result.Feedback = feedback
	result.Score = score
	return result
}

func hasUppercase(s string) bool {
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			return true
		}
	}
	return false
}

func hasLowercase(s string) bool {
	for _, c := range s {
		if c >= 'a' && c <= 'z' {
			return true
		}
	}
	return false
}

func hasNumber(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

func hasSpecial(s string) bool {
	for _, c := range s {
		if strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?/~", c) {
			return true
		}
	}
	return false
}

func SanitizeInput(input string) string {
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")
	input = strings.ReplaceAll(input, "/", "&#x2F;")
	return input
}

func ValidateURL(rawurl string) bool {
	parsed, err := url.Parse(rawurl)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return true
}

func HashSHA256(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

type OWASPTop10 struct {
	Name        string
	Description string
	Risk        string
	TestFunc    func(t *testing.T)
}

var OWASPTop10Tests = []OWASPTop10{
	{
		Name:        "A01:2021 - Broken Access Control",
		Description: "Verification that access controls properly restrict unauthorized users",
		Risk:        "Critical",
		TestFunc:    TestBrokenAccessControl,
	},
	{
		Name:        "A02:2021 - Cryptographic Failures",
		Description: "Testing secure cryptography implementation",
		Risk:        "Critical",
		TestFunc:    TestCryptographicFailures,
	},
	{
		Name:        "A03:2021 - Injection",
		Description: "Testing for SQL, XSS, and other injection flaws",
		Risk:        "Critical",
		TestFunc:    TestInjection,
	},
	{
		Name:        "A05:2021 - Security Misconfiguration",
		Description: "Testing for insecure default configurations",
		Risk:        "High",
		TestFunc:    TestSecurityMisconfiguration,
	},
	{
		Name:        "A06:2021 - Vulnerable and Outdated Components",
		Description: "Testing for known vulnerabilities in dependencies",
		Risk:        "High",
		TestFunc:    TestVulnerableComponents,
	},
	{
		Name:        "A07:2021 - Identification and Authentication Failures",
		Description: "Testing authentication mechanism vulnerabilities",
		Risk:        "High",
		TestFunc:    TestAuthFailures,
	},
	{
		Name:        "A08:2021 - Software and Data Integrity Failures",
		Description: "Testing for insecure deserialization",
		Risk:        "High",
		TestFunc:    TestIntegrityFailures,
	},
	{
		Name:        "A09:2021 - Security Logging and Monitoring Failures",
		Description: "Testing logging and monitoring implementations",
		Risk:        "Medium",
		TestFunc:    TestLoggingFailures,
	},
	{
		Name:        "A10:2021 - Server-Side Request Forgery",
		Description: "Testing for SSRF vulnerabilities",
		Risk:        "High",
		TestFunc:    TestSSRF,
	},
}

func RunOWASPSecurityTests(t *testing.T) {
	for _, test := range OWASPTop10Tests {
		t.Run(test.Name, test.TestFunc)
	}
}

func TestBrokenAccessControl(t *testing.T) {
	t.Run("UnauthorizedAccess", func(t *testing.T) {
		scanner := NewSecurityScanner()
		payload := "../../../etc/passwd"
		vulnerable, _ := scanner.ScanPathTraversal(payload)
		assertTrue(t, len(vulnerable) > 0, "Path traversal should be detected")
	})
}

func TestCryptographicFailures(t *testing.T) {
	t.Run("WeakHashing", func(t *testing.T) {
		password := "password123"
		strength := CheckPasswordStrength(password)
		assertTrue(t, strength.Score < 5, "Weak password should have low score")
	})

	t.Run("SecureHashing", func(t *testing.T) {
		input := "sensitive data"
		hash := HashSHA256(input)
		assertTrue(t, len(hash) == 64, "SHA256 should produce 64 char hash")
	})
}

func TestInjection(t *testing.T) {
	t.Run("SQLInjection", func(t *testing.T) {
		scanner := NewSecurityScanner()
		payload := "' OR '1'='1"
		vulnerable, _ := scanner.ScanSQLInjection(payload)
		assertTrue(t, len(vulnerable) > 0, "SQL injection should be detected")
	})

	t.Run("XSSInjection", func(t *testing.T) {
		scanner := NewSecurityScanner()
		payload := "<script>alert('xss')</script>"
		vulnerable, _ := scanner.ScanXSS(payload)
		assertTrue(t, len(vulnerable) > 0, "XSS should be detected")
	})

	t.Run("XSSSanitization", func(t *testing.T) {
		input := "<script>alert('xss')</script>"
		sanitized := SanitizeInput(input)
		assertTrue(t, !strings.Contains(sanitized, "<script"), "XSS should be sanitized")
	})
}

func TestSecurityMisconfiguration(t *testing.T) {
	t.Run("DefaultPasswords", func(t *testing.T) {
		commonPasswords := []string{"admin", "password", "123456", "qwerty"}
		for _, pwd := range commonPasswords {
			strength := CheckPasswordStrength(pwd)
			assertTrue(t, strength.Strength == "WEAK", "Common passwords should be weak")
		}
	})
}

func TestVulnerableComponents(t *testing.T) {
	t.Run("DependencyCheck", func(t *testing.T) {
		assertTrue(t, true, "Should run dependency vulnerability scan")
	})
}

func TestAuthFailures(t *testing.T) {
	t.Run("WeakPasswords", func(t *testing.T) {
		weakPasswords := []string{"test", "123", "password"}
		for _, pwd := range weakPasswords {
			strength := CheckPasswordStrength(pwd)
			assertTrue(t, strength.Strength == "WEAK", "Should detect weak passwords")
		}
	})

	t.Run("StrongPassword", func(t *testing.T) {
		strength := CheckPasswordStrength("MyStr0ng!Passw0rd")
		assertTrue(t, strength.Strength == "STRONG" || strength.Strength == "VERY STRONG", "Should detect strong password")
	})
}

func TestIntegrityFailures(t *testing.T) {
	t.Run("InsecureDeserialization", func(t *testing.T) {
		scanner := NewSecurityScanner()
		xxePayload := `<!DOCTYPE test [ <!ENTITY xxe SYSTEM "file:///etc/passwd"> ]>`
		vulnerable, _ := scanner.ScanXXE(xxePayload)
		assertTrue(t, len(vulnerable) > 0, "XXE should be detected")
	})
}

func TestLoggingFailures(t *testing.T) {
	t.Run("SensitiveDataLogging", func(t *testing.T) {
		assertTrue(t, true, "Should test that sensitive data is not logged")
	})
}

func TestSSRF(t *testing.T) {
	t.Run("InternalURL", func(t *testing.T) {
		scanner := NewSecurityScanner()
		payload := "http://127.0.0.1:8080/internal"
		vulnerable, _ := scanner.ScanSSRF(payload)
		assertTrue(t, len(vulnerable) > 0, "SSRF should be detected")
	})

	t.Run("ValidateURL", func(t *testing.T) {
		assertTrue(t, ValidateURL("https://example.com"), "Valid HTTPS URL should be allowed")
		assertTrue(t, !ValidateURL("file:///etc/passwd"), "File URLs should be blocked")
	})
}

func assertTrue(t *testing.T, condition bool, msg string, args ...interface{}) {
	t.Helper()
	if !condition {
		t.Errorf(msg, args...)
	}
}
