package security

import (
	"testing"
)

func TestSecurityScannerCreation(t *testing.T) {
	scanner := NewSecurityScanner()

	if scanner == nil {
		t.Fatal("Expected security scanner to be created, got nil")
	}
}

func TestSQLInjectionScan(t *testing.T) {
	scanner := NewSecurityScanner()

	// Test cases that should be flagged
	testCases := []string{
		"' OR '1'='1",
		"admin' --",
		"DROP TABLE users;",
		"' UNION SELECT * FROM users",
	}

	for _, tc := range testCases {
		vulnerabilities, err := scanner.ScanSQLInjection(tc)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(vulnerabilities) == 0 {
			t.Errorf("Expected SQL injection vulnerability for input: %q", tc)
		}
		scanner.Clear()
	}

	// Test cases that should not be flagged
	safeCases := []string{
		"normal input",
		"hello world",
		"user@example.com",
		"123456",
	}

	for _, tc := range safeCases {
		vulnerabilities, err := scanner.ScanSQLInjection(tc)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(vulnerabilities) > 0 {
			t.Errorf("Unexpected SQL injection flag for safe input: %q", tc)
		}
		scanner.Clear()
	}
}

func TestXSSScan(t *testing.T) {
	scanner := NewSecurityScanner()

	testCases := []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert(1)>",
		"javascript:alert('xss')",
		"<svg onload=alert(1)>",
	}

	for _, tc := range testCases {
		vulnerabilities, err := scanner.ScanXSS(tc)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(vulnerabilities) == 0 {
			t.Errorf("Expected XSS vulnerability for input: %q", tc)
		}
		scanner.Clear()
	}
}

func TestPathTraversalScan(t *testing.T) {
	scanner := NewSecurityScanner()

	testCases := []string{
		"../../etc/passwd",
		"../../../../etc/passwd",
		"./../file.txt",
		"/etc/passwd",
	}

	for _, tc := range testCases {
		vulnerabilities, err := scanner.ScanPathTraversal(tc)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(vulnerabilities) == 0 {
			t.Errorf("Expected path traversal vulnerability for input: %q", tc)
		}
		scanner.Clear()
	}
}

func TestCommandInjectionScan(t *testing.T) {
	scanner := NewSecurityScanner()

	testCases := []string{
		"; rm -rf /",
		"&& cat /etc/passwd",
		"| whoami",
		"$(echo hello)",
	}

	for _, tc := range testCases {
		vulnerabilities, err := scanner.ScanCommandInjection(tc)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(vulnerabilities) == 0 {
			t.Errorf("Expected command injection vulnerability for input: %q", tc)
		}
		scanner.Clear()
	}
}

func TestFullScan(t *testing.T) {
	scanner := NewSecurityScanner()

	input := "'; <script>alert('xss')</script> ../../etc/passwd"

	vulnerabilities, err := scanner.ScanAll(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(vulnerabilities) == 0 {
		t.Error("Expected vulnerabilities from full scan")
	}
}

func TestCheckPasswordStrength(t *testing.T) {
	tests := []struct {
		password    string
		wantStrength string
	}{
		{"", "WEAK"},
		{"123456", "WEAK"},
		{"abc123", "WEAK"},
		{"Password123", "MEDIUM"},
		{"P@ssw0rd123!", "VERY STRONG"},
		{"My$tr0ngP@ssw0rd!2024", "VERY STRONG"},
	}

	for _, tt := range tests {
		strength := CheckPasswordStrength(tt.password)
		if strength.Strength != tt.wantStrength {
			t.Errorf("CheckPasswordStrength(%q) = %s, want %s", tt.password, strength.Strength, tt.wantStrength)
		}
	}
}

func TestSanitizeInput(t *testing.T) {
	input := "<script>alert('xss')</script> hello <b>world</b>"
	sanitized := SanitizeInput(input)

	if sanitized == input {
		t.Error("Expected input to be sanitized")
	}
}

func TestValidateURL(t *testing.T) {
	validURLs := []string{
		"https://example.com",
		"http://sub.example.org/path",
		"https://test.co.uk:8080",
	}

	invalidURLs := []string{
		"javascript:alert(1)",
		"file:///etc/passwd",
		"data:text/html",
		"not-a-url",
	}

	for _, url := range validURLs {
		if !ValidateURL(url) {
			t.Errorf("Expected valid URL rejected: %q", url)
		}
	}

	for _, url := range invalidURLs {
		if ValidateURL(url) {
			t.Errorf("Expected invalid URL accepted: %q", url)
		}
	}
}

func TestHashSHA256(t *testing.T) {
	input := "test data"
	hash1 := HashSHA256(input)
	hash2 := HashSHA256(input)
	hash3 := HashSHA256("different data")

	if hash1 == "" {
		t.Error("Expected non-empty hash")
	}

	if hash1 != hash2 {
		t.Error("Same input should produce same hash")
	}

	if hash1 == hash3 {
		t.Error("Different inputs should produce different hashes")
	}
}
