package service

import (
	"context"
	"regexp"
	"testing"
)

func TestEnhancedSQLInjectionProtection(t *testing.T) {
	t.Run("DetectUNIONSELECT", func(t *testing.T) {
		protection := NewEnhancedSQLInjectionProtection(nil)

		testCases := []struct {
			input       string
			shouldBlock bool
		}{
			{"' OR '1'='1", true},
			{"admin' --", true},
			{"' UNION SELECT * FROM users--", true},
			{"1; DROP TABLE users", true},
			{"normal text input", false},
			{"user@example.com", false},
		}

		for _, tc := range testCases {
			valid, _, _ := protection.ValidateInput(tc.input)
			if valid == tc.shouldBlock {
				t.Errorf("Expected shouldBlock=%v for input %q, got valid=%v", tc.shouldBlock, tc.input, valid)
			}
		}
	})

	t.Run("ValidateAllInputs", func(t *testing.T) {
		protection := NewEnhancedSQLInjectionProtection(nil)

		inputs := map[string]string{
			"username": "normaluser",
			"password": "' OR '1'='1",
		}

		results := protection.ValidateAllInputs(inputs)

		if results["username"].Valid != true {
			t.Error("Expected username to be valid")
		}
		if results["password"].Valid != false {
			t.Error("Expected password to be blocked")
		}
	})

	t.Run("SanitizeInput", func(t *testing.T) {
		protection := NewEnhancedSQLInjectionProtection(nil)

		maliciousInput := "' OR '1'='1"
		sanitized := protection.SanitizeInput(maliciousInput)

		valid, _, _ := protection.ValidateInput(sanitized)
		if !valid {
			t.Errorf("Expected sanitized input to be valid, got: %s", sanitized)
		}
	})
}

func TestEnhancedRBACService(t *testing.T) {
	t.Run("HasPermission", func(t *testing.T) {
		rbac := NewEnhancedRBACService(nil)

		testCases := []struct {
			role        string
			permission  string
			shouldAllow bool
		}{
			{"admin", "user:read", true},
			{"admin", "user:delete", true},
			{"admin", "settings:update", true},
			{"moderator", "user:read", true},
			{"moderator", "user:delete", false},
			{"moderator", "content:moderate", true},
			{"user", "user:read:own", true},
			{"user", "user:read", false},
			{"user", "user:delete", false},
			{"guest", "content:read", true},
			{"guest", "user:read", false},
		}

		for _, tc := range testCases {
			result := rbac.HasPermission(tc.role, tc.permission)
			if result != tc.shouldAllow {
				t.Errorf("Role %s with permission %s: expected %v, got %v", tc.role, tc.permission, tc.shouldAllow, result)
			}
		}
	})

	t.Run("AddRole", func(t *testing.T) {
		rbac := NewEnhancedRBACService(nil)

		newRole := &Role{
			Name:        "testrole",
			Permissions: []string{"test:read", "test:write"},
		}

		err := rbac.AddRole(newRole)
		if err != nil {
			t.Errorf("Expected no error when adding new role, got: %v", err)
		}

		exists := rbac.HasPermission("testrole", "test:read")
		if !exists {
			t.Error("Expected new role to have permissions")
		}

		err = rbac.AddRole(newRole)
		if err == nil {
			t.Error("Expected error when adding duplicate role")
		}
	})

	t.Run("UpdateRolePermissions", func(t *testing.T) {
		rbac := NewEnhancedRBACService(nil)

		err := rbac.UpdateRolePermissions("user", []string{"user:read:own", "user:update:own", "new:permission"})
		if err != nil {
			t.Errorf("Expected no error when updating permissions, got: %v", err)
		}

		perms, _ := rbac.GetRolePermissions("user")
		found := false
		for _, p := range perms {
			if p == "new:permission" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected new permission to be in role permissions")
		}
	})

	t.Run("GetAllRoles", func(t *testing.T) {
		rbac := NewEnhancedRBACService(nil)

		roles := rbac.GetAllRoles()
		if len(roles) == 0 {
			t.Error("Expected at least one role")
		}

		roleNames := make(map[string]bool)
		for _, role := range roles {
			roleNames[role.Name] = true
		}

		expectedRoles := []string{"admin", "moderator", "user", "guest"}
		for _, expected := range expectedRoles {
			if !roleNames[expected] {
				t.Errorf("Expected role %s to exist", expected)
			}
		}
	})
}

func TestEnhancedCSRFService(t *testing.T) {
	t.Run("GenerateAndValidateToken", func(t *testing.T) {
		csrf := NewEnhancedCSRFService(nil)
		ctx := context.Background()

		token, err := csrf.GenerateToken(ctx, "user123", "session456", "192.168.1.1", "Mozilla/5.0")
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		if token == "" {
			t.Error("Generated token should not be empty")
		}

		valid, err := csrf.ValidateToken(ctx, token, "192.168.1.1", "Mozilla/5.0")
		if err != nil {
			t.Errorf("Expected valid token, got error: %v", err)
		}
		if !valid {
			t.Error("Expected token to be valid")
		}
	})

	t.Run("TokenIPMismatch", func(t *testing.T) {
		csrf := NewEnhancedCSRFService(nil)
		ctx := context.Background()

		token, _ := csrf.GenerateToken(ctx, "user123", "session456", "192.168.1.1", "Mozilla/5.0")

		valid, err := csrf.ValidateToken(ctx, token, "192.168.1.2", "Mozilla/5.0")
		if err == nil && valid {
			t.Error("Expected token validation to fail due to IP mismatch")
		}
	})

	t.Run("ReuseToken", func(t *testing.T) {
		csrf := NewEnhancedCSRFService(nil)
		ctx := context.Background()

		token, _ := csrf.GenerateToken(ctx, "user123", "session456", "192.168.1.1", "Mozilla/5.0")

		csrf.InvalidateToken(ctx, token)

		valid, _ := csrf.ValidateToken(ctx, token, "192.168.1.1", "Mozilla/5.0")
		if valid {
			t.Error("Expected reused token to be invalid")
		}
	})

	t.Run("InvalidateUserTokens", func(t *testing.T) {
		csrf := NewEnhancedCSRFService(nil)
		ctx := context.Background()

		token1, _ := csrf.GenerateToken(ctx, "user123", "session1", "192.168.1.1", "Mozilla/5.0")
		token2, _ := csrf.GenerateToken(ctx, "user123", "session2", "192.168.1.1", "Mozilla/5.0")
		token3, _ := csrf.GenerateToken(ctx, "otheruser", "session3", "192.168.1.1", "Mozilla/5.0")

		csrf.InvalidateUserTokens(ctx, "user123")

		valid1, _ := csrf.ValidateToken(ctx, token1, "192.168.1.1", "Mozilla/5.0")
		valid2, _ := csrf.ValidateToken(ctx, token2, "192.168.1.1", "Mozilla/5.0")
		valid3, _ := csrf.ValidateToken(ctx, token3, "192.168.1.1", "Mozilla/5.0")

		if valid1 || valid2 {
			t.Error("Expected user123 tokens to be invalidated")
		}
		if !valid3 {
			t.Error("Expected otheruser token to still be valid")
		}
	})
}

func TestEnhancedXSSProtection(t *testing.T) {
	t.Run("SanitizeHTML", func(t *testing.T) {
		xss := NewEnhancedXSSProtection(nil)

		testCases := []struct {
			input        string
			shouldBlockXSS bool
		}{
			{"<script>alert('xss')</script>", true},
			{"<img src=x onerror=alert(1)>", true},
			{"javascript:alert(1)", true},
			{"<p>normal text</p>", false},
			{"<b>bold text</b>", false},
			{"Hello &amp; World", false},
			{"plain text only", false},
		}

		for _, tc := range testCases {
			detected, _, _ := xss.DetectXSS(tc.input)
			if detected != tc.shouldBlockXSS {
				t.Errorf("Input %q: expected XSS detection=%v, got %v", tc.input, tc.shouldBlockXSS, detected)
			}
		}
	})

	t.Run("DetectXSS", func(t *testing.T) {
		xss := NewEnhancedXSSProtection(nil)

		testCases := []struct {
			input       string
			shouldDetect bool
			expectedType string
		}{
			{"<script>alert(1)</script>", true, "script_tag"},
			{"javascript:alert(1)", true, "javascript_protocol"},
			{"<img src=x onerror=alert(1)>", true, "onerror_event"},
			{"<iframe src='evil.com'>", true, "iframe_tag"},
			{"<svg onload=alert(1)>", true, "onload_event"},
			{"normal text", false, ""},
		}

		for _, tc := range testCases {
			detected, pattern, _ := xss.DetectXSS(tc.input)
			if detected != tc.shouldDetect {
				t.Errorf("Input %q: expected detected=%v, got %v", tc.input, tc.shouldDetect, detected)
			}
			if tc.shouldDetect && pattern != tc.expectedType {
				t.Errorf("Input %q: expected pattern=%s, got %s", tc.input, tc.expectedType, pattern)
			}
		}
	})

	t.Run("ValidateURL", func(t *testing.T) {
		xss := NewEnhancedXSSProtection(nil)

		testCases := []struct {
			url         string
			shouldValid bool
		}{
			{"https://example.com", true},
			{"http://example.com/path?query=1", true},
			{"mailto:user@example.com", true},
			{"tel:+1234567890", true},
			{"javascript:alert(1)", false},
			{"file:///etc/passwd", false},
		}

		for _, tc := range testCases {
			valid, _ := xss.ValidateURL(tc.url)
			if valid != tc.shouldValid {
				t.Errorf("URL %q: expected valid=%v, got %v", tc.url, tc.shouldValid, valid)
			}
		}
	})
}

func TestSecurityAuditService(t *testing.T) {
	t.Run("LogSecurityEvent", func(t *testing.T) {
		audit := NewSecurityAuditService(nil)

		entry := &AuditLogEntry{
			UserID:    "user123",
			Action:    "login",
			Resource:  "auth",
			Result:    "success",
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
			RiskScore: 0.1,
		}

		err := audit.LogSecurityEvent(entry)
		if err != nil {
			t.Errorf("Expected no error logging event, got: %v", err)
		}
	})

	t.Run("EnableDisableChecks", func(t *testing.T) {
		audit := NewSecurityAuditService(nil)

		if !audit.IsCheckEnabled("sql_injection") {
			t.Error("Expected sql_injection check to be enabled by default")
		}

		audit.DisableCheck("sql_injection")
		if audit.IsCheckEnabled("sql_injection") {
			t.Error("Expected sql_injection check to be disabled")
		}

		audit.EnableCheck("sql_injection")
		if !audit.IsCheckEnabled("sql_injection") {
			t.Error("Expected sql_injection check to be enabled")
		}
	})
}

func TestIsPrivateIP(t *testing.T) {
	testCases := []struct {
		ip       string
		isPrivate bool
	}{
		{"127.0.0.1", true},
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"[::1]", true},
		{"::1", true},
	}

	for _, tc := range testCases {
		result := IsPrivateIP(tc.ip)
		if result != tc.isPrivate {
			t.Errorf("IP %s: expected private=%v, got %v", tc.ip, tc.isPrivate, result)
		}
	}
}

func TestGenerateSecureToken(t *testing.T) {
	t.Run("GenerateDifferentTokens", func(t *testing.T) {
		token1, err := GenerateSecureToken(32)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		token2, err := GenerateSecureToken(32)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		if token1 == token2 {
			t.Error("Generated tokens should be unique")
		}

		if len(token1) != 64 {
			t.Errorf("Expected token length 64, got %d", len(token1))
		}
	})

	t.Run("TokenLength", func(t *testing.T) {
		testLengths := []int{16, 32, 64}

		for _, length := range testLengths {
			token, err := GenerateSecureToken(length)
			if err != nil {
				t.Errorf("Failed to generate token of length %d: %v", length, err)
			}
			expectedLen := length * 2
			if len(token) != expectedLen {
				t.Errorf("Expected token length %d, got %d", expectedLen, len(token))
			}
		}
	})
}

func TestGenerateUUID(t *testing.T) {
	t.Run("GenerateDifferentUUIDs", func(t *testing.T) {
		uuid1 := GenerateUUID()
		uuid2 := GenerateUUID()

		if uuid1 == uuid2 {
			t.Error("Generated UUIDs should be unique")
		}

		if len(uuid1) != 36 {
			t.Errorf("Expected UUID length 36, got %d", len(uuid1))
		}
	})
}

func TestEnhancedSQLInjectionValidators(t *testing.T) {
	protection := NewEnhancedSQLInjectionProtection(nil)

	t.Run("AddCustomValidator", func(t *testing.T) {
		customPattern := regexp.MustCompile(`(?i)custom_pattern`)

		protection.AddCustomValidator("custom", customPattern, "HIGH", "Custom injection pattern")

		valid, pattern, _ := protection.ValidateInput("test custom_pattern value")
		if valid {
			t.Error("Expected custom pattern to be detected")
		}
		if pattern != "custom" {
			t.Errorf("Expected pattern name 'custom', got %s", pattern)
		}
	})
}
