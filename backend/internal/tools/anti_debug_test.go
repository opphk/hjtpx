package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestAntiDebug_NewAntiDebug(t *testing.T) {
	ad := NewAntiDebug()

	if ad == nil {
		t.Fatal("NewAntiDebug should not return nil")
	}

	if ad.IsCompromised() {
		t.Error("NewAntiDebug should not be compromised initially")
	}

	if ad.GetDetectionCount() != 0 {
		t.Errorf("Expected 0 detection count, got %d", ad.GetDetectionCount())
	}
}

func TestAntiDebug_Config(t *testing.T) {
	config := AntiDebugConfig{
		EnableDevToolsDetection:   true,
		EnableMemoryProtection:    true,
		EnableIntegrityCheck:      true,
		EnableSelfDestruct:        true,
		EnableAntiTampering:       true,
		EnableAutomationDetection: true,
		CheckInterval:             1000 * time.Millisecond,
	}

	ad := NewAntiDebug(config)

	if ad == nil {
		t.Fatal("NewAntiDebug with config should not return nil")
	}
}

func TestAntiDebug_GenerateIntegrityHash(t *testing.T) {
	ad := NewAntiDebug()

	code := "console.log('test');"
	hash := ad.GenerateIntegrityHash(code)

	if hash == "" {
		t.Error("GenerateIntegrityHash should not return empty string")
	}

	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}

	hash2 := ad.GenerateIntegrityHash(code)
	if hash != hash2 {
		t.Error("Same code should produce same hash")
	}

	code2 := "console.log('different');"
	hash3 := ad.GenerateIntegrityHash(code2)
	if hash == hash3 {
		t.Error("Different codes should produce different hashes")
	}
}

func TestAntiDebug_SetAndVerifyIntegrity(t *testing.T) {
	ad := NewAntiDebug()

	code := "function test() { return 42; }"
	hash := ad.GenerateIntegrityHash(code)

	ad.SetIntegrityHash(hash)

	if !ad.VerifyIntegrity(code) {
		t.Error("VerifyIntegrity should return true for original code")
	}

	modifiedCode := "function test() { return 0; }"
	if ad.VerifyIntegrity(modifiedCode) {
		t.Error("VerifyIntegrity should return false for modified code")
	}
}

func TestAntiDebug_RecordDetection(t *testing.T) {
	ad := NewAntiDebug()

	ad.RecordDetection()
	if ad.GetDetectionCount() != 1 {
		t.Errorf("Expected detection count 1, got %d", ad.GetDetectionCount())
	}

	ad.RecordDetection()
	ad.RecordDetection()

	if ad.GetDetectionCount() != 3 {
		t.Errorf("Expected detection count 3, got %d", ad.GetDetectionCount())
	}
}

func TestAntiDebug_MarkCompromised(t *testing.T) {
	ad := NewAntiDebug()

	if ad.IsCompromised() {
		t.Error("Should not be compromised initially")
	}

	ad.MarkCompromised()

	if !ad.IsCompromised() {
		t.Error("Should be compromised after MarkCompromised")
	}
}

func TestAntiDebug_GenerateChallenge(t *testing.T) {
	ad := NewAntiDebug()

	challenge, err := ad.GenerateChallenge()
	if err != nil {
		t.Fatalf("GenerateChallenge failed: %v", err)
	}

	if challenge == "" {
		t.Error("Challenge should not be empty")
	}

	if len(challenge) < 20 {
		t.Errorf("Challenge seems too short: %s", challenge)
	}

	challenge2, _ := ad.GenerateChallenge()
	if challenge == challenge2 {
		t.Error("Challenges should be unique")
	}
}

func TestAntiDebug_TokenGeneration(t *testing.T) {
	ad := NewAntiDebug()

	token, err := ad.GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if token == "" {
		t.Error("Token should not be empty")
	}

	if !ad.ValidateToken(token) {
		t.Error("Valid token should pass validation")
	}
}

func TestAntiDebug_InvalidToken(t *testing.T) {
	ad := NewAntiDebug()

	if ad.ValidateToken("invalid-token") {
		t.Error("Invalid token should fail validation")
	}

	if ad.ValidateToken("") {
		t.Error("Empty token should fail validation")
	}
}

func TestIntegrityChecker_New(t *testing.T) {
	ic := NewIntegrityChecker()

	if ic == nil {
		t.Fatal("NewIntegrityChecker should not return nil")
	}
}

func TestIntegrityChecker_RegisterAndVerify(t *testing.T) {
	ic := NewIntegrityChecker()

	code := "function test() { return true; }"
	err := ic.RegisterCode("test_func", code)
	if err != nil {
		t.Fatalf("RegisterCode failed: %v", err)
	}

	valid, err := ic.VerifyCode("test_func", code)
	if err != nil {
		t.Fatalf("VerifyCode failed: %v", err)
	}

	if !valid {
		t.Error("Original code should verify correctly")
	}

	modifiedCode := "function test() { return false; }"
	valid, err = ic.VerifyCode("test_func", modifiedCode)
	if err != nil {
		t.Fatalf("VerifyCode failed: %v", err)
	}

	if valid {
		t.Error("Modified code should not verify")
	}
}

func TestIntegrityChecker_NonexistentCode(t *testing.T) {
	ic := NewIntegrityChecker()

	_, err := ic.VerifyCode("nonexistent", "some code")
	if err == nil {
		t.Error("Should return error for unregistered code")
	}
}

func TestMemoryProtection_ProtectFunction(t *testing.T) {
	mp := NewMemoryProtection()

	mp.ProtectFunction("testFunc", "function testFunc() { return 1; }")

	if !mp.IsFunctionProtected("testFunc") {
		t.Error("Function should be marked as protected")
	}

	if mp.IsFunctionProtected("otherFunc") {
		t.Error("Other function should not be protected")
	}
}

func TestMemoryProtection_DetectModification(t *testing.T) {
	mp := NewMemoryProtection()

	originalCode := "function test() { return 1; }"
	mp.ProtectFunction("test", originalCode)

	if mp.DetectModification("test", originalCode) {
		t.Error("Original code should not be detected as modified")
	}

	modifiedCode := "function test() { return 2; }"
	if !mp.DetectModification("test", modifiedCode) {
		t.Error("Modified code should be detected")
	}

	if mp.DetectModification("unknown", originalCode) {
		t.Error("Unknown function should return false")
	}
}

func TestRuntimeMonitor_Events(t *testing.T) {
	rm := NewRuntimeMonitor()

	rm.LogEvent("test_event", "Test description", "info")

	events := rm.GetRecentEvents(10)
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if events[0].EventType != "test_event" {
		t.Errorf("Expected event type 'test_event', got '%s'", events[0].EventType)
	}

	if events[0].Severity != "info" {
		t.Errorf("Expected severity 'info', got '%s'", events[0].Severity)
	}
}

func TestRuntimeMonitor_GetStatus(t *testing.T) {
	rm := NewRuntimeMonitor()

	status := rm.GetStatus()

	if status["is_compromised"] != false {
		t.Error("Should not be compromised initially")
	}

	if status["detection_count"].(int) != 0 {
		t.Error("Detection count should be 0")
	}

	rm.antiDebug.RecordDetection()
	rm.antiDebug.RecordDetection()

	status = rm.GetStatus()
	if status["detection_count"].(int) != 2 {
		t.Errorf("Expected detection count 2, got %v", status["detection_count"])
	}
}

func TestAntiDebugGenerator_GenerateJavaScriptProtection(t *testing.T) {
	gen := NewAntiDebugGenerator()

	code := gen.GenerateJavaScriptProtection()
	if code == "" {
		t.Error("Generated code should not be empty")
	}

	if !strings.Contains(code, "_0xAD") {
		t.Error("Generated code should contain anti-debug object")
	}

	if !strings.Contains(code, "registerDetector") {
		t.Error("Generated code should contain registerDetector")
	}
}

func TestAntiDebugGenerator_WithConfig(t *testing.T) {
	config := AntiDebugConfig{
		EnableDevToolsDetection:   true,
		EnableMemoryProtection:    true,
		EnableSelfDestruct:        true,
		EnableAntiTampering:       true,
		EnableAutomationDetection: true,
		CheckInterval:             3000 * time.Millisecond,
	}

	gen := NewAntiDebugGenerator(config)

	code := gen.GenerateJavaScriptProtection()
	if code == "" {
		t.Error("Generated code should not be empty")
	}
}

func TestGenerateAntiDebugCode_Basic(t *testing.T) {
	code := GenerateAntiDebugCode(nil)
	if code == "" {
		t.Error("Generated code should not be empty")
	}

	if !strings.Contains(code, "_0xDT") {
		t.Error("Code should contain DevTools detector")
	}

	if !strings.Contains(code, "_0xMP") {
		t.Error("Code should contain Memory Protection")
	}
}

func TestGenerateAntiDebugCode_WithOptions(t *testing.T) {
	options := map[string]interface{}{
		"enableDevTools":           false,
		"enableMemoryProtection":   true,
		"enableSelfDestruct":      true,
		"enableAntiTampering":      false,
		"enableAutomationDetection": true,
		"checkInterval":           5000,
	}

	code := GenerateAntiDebugCode(options)
	if code == "" {
		t.Error("Generated code should not be empty")
	}
}

func TestGenerateIntegrityCheck(t *testing.T) {
	codeHash := "abc123def456"
	code := GenerateIntegrityCheck(codeHash)

	if code == "" {
		t.Error("Generated code should not be empty")
	}

	if !strings.Contains(code, codeHash) {
		t.Error("Generated code should contain hash")
	}

	if !strings.Contains(code, "__integrityHash") && !strings.Contains(code, "_0xH") {
		t.Error("Generated code should reference integrity hash")
	}
}

func TestGenerateSecureToken(t *testing.T) {
	secret := []byte("test-secret-key")
	data := []byte("test data")

	token, err := GenerateSecureToken(secret, data)
	if err != nil {
		t.Fatalf("GenerateSecureToken failed: %v", err)
	}

	if token == "" {
		t.Error("Token should not be empty")
	}

	verifiedData, ok := VerifySecureToken(secret, token)
	if !ok {
		t.Error("Valid token should verify successfully")
	}

	if string(verifiedData) != string(data) {
		t.Error("Verified data should match original")
	}
}

func TestVerifySecureToken_WrongSecret(t *testing.T) {
	secret1 := []byte("secret-key-1")
	secret2 := []byte("secret-key-2")
	data := []byte("test data")

	token, _ := GenerateSecureToken(secret1, data)

	_, ok := VerifySecureToken(secret2, token)
	if ok {
		t.Error("Token signed with different secret should not verify")
	}
}

func TestVerifySecureToken_TamperedToken(t *testing.T) {
	secret := []byte("test-secret")
	data := []byte("original data")

	token, _ := GenerateSecureToken(secret, data)

	tamperedToken := token + "extra"

	_, ok := VerifySecureToken(secret, tamperedToken)
	if ok {
		t.Error("Tampered token should not verify")
	}
}

func TestGenerateIntegrityReport(t *testing.T) {
	monitor := NewRuntimeMonitor()

	report := GenerateIntegrityReport(monitor)

	if report == nil {
		t.Fatal("Report should not be nil")
	}

	if !report.IsValid {
		t.Error("Fresh monitor should report as valid (no compromise)")
	}

	monitor.antiDebug.MarkCompromised()

	report = GenerateIntegrityReport(monitor)
	if report.IsValid {
		t.Error("Compromised monitor should report as invalid")
	}

	if len(report.Violations) == 0 {
		t.Error("Compromised monitor should have violations")
	}
}

func TestCodeObfuscator_New(t *testing.T) {
	co := NewCodeObfuscator()

	if co == nil {
		t.Fatal("NewCodeObfuscator should not return nil")
	}
}

func TestCodeObfuscator_Protect(t *testing.T) {
	co := NewCodeObfuscator()

	code := "function test() { console.log('hello'); }"
	protected, err := co.Protect(code)

	if err != nil {
		t.Fatalf("Protect failed: %v", err)
	}

	if protected == "" {
		t.Error("Protected code should not be empty")
	}

	if protected == code {
		t.Error("Protected code should be different from original")
	}
}

func TestCreateCodeGuard(t *testing.T) {
	code := "function test() {}"
	key := "secret-key"

	guard := CreateCodeGuard(code, key)

	if guard == "" {
		t.Error("Guard code should not be empty")
	}

	if !strings.Contains(guard, key) {
		t.Error("Guard should contain the key")
	}
}

func TestValidateCodeSafety(t *testing.T) {
	safeCode := "function test() { return 42; }"
	valid, msg := ValidateCodeSafety(safeCode)
	if !valid {
		t.Errorf("Safe code should be valid: %s", msg)
	}

	dangerousCode := "eval('alert(1)');"
	valid, _ = ValidateCodeSafety(dangerousCode)
	if valid {
		t.Error("Code with eval should not be safe")
	}

	dangerousCode2 := "document.write('<script>alert(1)</script>');"
	valid, _ = ValidateCodeSafety(dangerousCode2)
	if valid {
		t.Error("Code with document.write should not be safe")
	}
}

func TestVirtualizationEngine(t *testing.T) {
	ve := NewVirtualizationEngine()

	if ve == nil {
		t.Fatal("NewVirtualizationEngine should not return nil")
	}

	code := []byte{0x00, 0xFF}

	err := ve.Execute(code)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
}

func TestCreateVirtualizedCode(t *testing.T) {
	code := "return 42"
	virtualized := CreateVirtualizedCode(code)

	if virtualized == "" {
		t.Error("Virtualized code should not be empty")
	}

	if !strings.Contains(virtualized, "_0xVM") {
		t.Error("Virtualized code should contain VM reference")
	}

	if !strings.Contains(virtualized, "_0xBC") {
		t.Error("Virtualized code should contain bytecode array")
	}
}

func TestAntiDebugMiddleware(t *testing.T) {
	m := NewAntiDebugMiddleware()

	if m == nil {
		t.Fatal("NewAntiDebugMiddleware should not return nil")
	}

	if !m.IsEnabled() {
		t.Error("Middleware should be enabled by default")
	}

	m.SetEnabled(false)
	if m.IsEnabled() {
		t.Error("Middleware should be disabled after SetEnabled(false)")
	}

	script := m.GenerateProtectionScript()
	if script == "" {
		t.Error("Generated protection script should not be empty")
	}

	monitor := m.GetMonitor()
	if monitor == nil {
		t.Error("GetMonitor should not return nil")
	}
}

func TestAntiDebug_ConcurrentAccess(t *testing.T) {
	ad := NewAntiDebug()

	done := make(chan bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		go func() {
			ad.RecordDetection()
			_ = ad.IsCompromised()
			ad.GetSignatureKey()
			done <- true
		}()
	}

	for i := 0; i < iterations; i++ {
		<-done
	}
}

func TestAntiDebug_JSONSerialization(t *testing.T) {
	ad := NewAntiDebug()

	ad.SetIntegrityHash("test-hash-123")

	data, err := json.Marshal(ad)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
}

func TestIntegrityChecker_MultipleCodes(t *testing.T) {
	ic := NewIntegrityChecker()

	codes := map[string]string{
		"func1": "function a() {}",
		"func2": "function b() {}",
		"func3": "function c() {}",
	}

	for name, code := range codes {
		if err := ic.RegisterCode(name, code); err != nil {
			t.Fatalf("RegisterCode(%s) failed: %v", name, err)
		}
	}

	for name, code := range codes {
		valid, err := ic.VerifyCode(name, code)
		if err != nil {
			t.Fatalf("VerifyCode(%s) failed: %v", name, err)
		}
		if !valid {
			t.Errorf("Code %s should verify correctly", name)
		}
	}
}

func TestMemoryProtection_MultipleFunctions(t *testing.T) {
	mp := NewMemoryProtection()

	functions := map[string]string{
		"func1": "function a() {}",
		"func2": "function b() {}",
		"func3": "function c() {}",
	}

	for name, code := range functions {
		mp.ProtectFunction(name, code)
	}

	for name, code := range functions {
		if !mp.IsFunctionProtected(name) {
			t.Errorf("Function %s should be protected", name)
		}

		if mp.DetectModification(name, code) {
			t.Errorf("Original code for %s should not be detected as modified", name)
		}

		modified := code + " modified"
		if !mp.DetectModification(name, modified) {
			t.Errorf("Modified code for %s should be detected", name)
		}
	}
}

func TestRuntimeMonitor_ManyEvents(t *testing.T) {
	rm := NewRuntimeMonitor()

	eventTypes := []string{"event1", "event2", "event3", "event4", "event5"}
	severities := []string{"info", "warning", "error", "critical", "debug"}

	for i := 0; i < 100; i++ {
		rm.LogEvent(
			eventTypes[i%len(eventTypes)],
			fmt.Sprintf("Event %d description", i),
			severities[i%len(severities)],
		)
	}

	events := rm.GetRecentEvents(10)
	if len(events) != 10 {
		t.Errorf("Expected 10 recent events, got %d", len(events))
	}
}

func TestGenerateAntiDebugCode_VariousOptions(t *testing.T) {
	testCases := []struct {
		name    string
		options map[string]interface{}
	}{
		{
			name:    "all_disabled",
			options: map[string]interface{}{"enableDevTools": false, "enableMemoryProtection": false},
		},
		{
			name:    "all_enabled",
			options: map[string]interface{}{"enableDevTools": true, "enableSelfDestruct": true},
		},
		{
			name:    "custom_interval",
			options: map[string]interface{}{"checkInterval": 10000},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code := GenerateAntiDebugCode(tc.options)
			if code == "" {
				t.Errorf("Test case %s: generated code should not be empty", tc.name)
			}
		})
	}
}

func TestIntegrityReport_JSON(t *testing.T) {
	monitor := NewRuntimeMonitor()
	monitor.antiDebug.MarkCompromised()

	report := GenerateIntegrityReport(monitor)

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var result IntegrityReport
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if result.IsValid {
		t.Error("Compromised report should not be valid")
	}

	if len(result.Violations) == 0 {
		t.Error("Compromised report should have violations")
	}
}

func BenchmarkAntiDebug_GenerateIntegrityHash(b *testing.B) {
	ad := NewAntiDebug()
	code := "function test() { return 42; }"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ad.GenerateIntegrityHash(code)
	}
}

func BenchmarkAntiDebug_VerifyIntegrity(b *testing.B) {
	ad := NewAntiDebug()
	code := "function test() { return 42; }"
	hash := ad.GenerateIntegrityHash(code)
	ad.SetIntegrityHash(hash)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ad.VerifyIntegrity(code)
	}
}

func BenchmarkIntegrityChecker_VerifyCode(b *testing.B) {
	ic := NewIntegrityChecker()
	code := "function test() { return 42; }"
	ic.RegisterCode("test", code)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ic.VerifyCode("test", code)
	}
}

func BenchmarkGenerateAntiDebugCode(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateAntiDebugCode(nil)
	}
}

func TestAntiDebug_CodeSignature(t *testing.T) {
	ad := NewAntiDebug()

	code := "function test() { return 42; }"
	timestamp := time.Now().Unix()

	signature := ad.GenerateCodeSignature(code, timestamp)
	if signature == "" {
		t.Error("Signature should not be empty")
	}

	if !ad.VerifyCodeSignature(code, timestamp, signature) {
		t.Error("Valid signature should verify")
	}

	modifiedCode := "function test() { return 0; }"
	if ad.VerifyCodeSignature(modifiedCode, timestamp, signature) {
		t.Error("Signature for modified code should not verify")
	}
}

func TestAntiDebug_CodeSignature_Expired(t *testing.T) {
	ad := NewAntiDebug()

	code := "function test() { return 42; }"
	oldTimestamp := time.Now().Add(-10 * time.Minute).Unix()

	signature := ad.GenerateCodeSignature(code, oldTimestamp)
	if signature == "" {
		t.Error("Signature should not be empty")
	}

	newTimestamp := time.Now().Unix()
	if ad.VerifyCodeSignature(code, newTimestamp, signature) {
		t.Error("Expired timestamp should not verify")
	}
}
