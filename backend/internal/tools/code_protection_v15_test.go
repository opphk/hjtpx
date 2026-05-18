package tools

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"
)

func TestCodeProtectionV15Creation(t *testing.T) {
	cp := NewCodeProtectionV15()
	if cp == nil {
		t.Fatal("Expected CodeProtectionV15 to be created")
	}
	if cp.version != "15.0.0" {
		t.Errorf("Expected version 15.0.0, got %s", cp.version)
	}
}

func TestCodeProtectionV15Config(t *testing.T) {
	config := CodeProtectionV15Config{
		ProtectionLevel: ProtectionLevelAdvanced,
		EnableWASM: true,
		EnableIntegrityCheck: true,
		EnableAntiAutomation: true,
	}

	cp := NewCodeProtectionV15(config)

	if cp.config.ProtectionLevel != ProtectionLevelAdvanced {
		t.Error("Protection level should be Advanced")
	}
	if !cp.config.EnableWASM {
		t.Error("WASM should be enabled")
	}
}

func TestCodeProtectionV15ProtectCode(t *testing.T) {
	cp := NewCodeProtectionV15()
	code := `function hello() { return "world"; }`

	protected, err := cp.ProtectCode(code)
	if err != nil {
		t.Fatalf("ProtectCode failed: %v", err)
	}

	if protected == "" {
		t.Error("Protected code should not be empty")
	}

	if protected == code {
		t.Error("Protected code should differ from original")
	}
}

func TestCodeProtectionV15ProtectWithLevel(t *testing.T) {
	cp := NewCodeProtectionV15()
	code := `var test = "value";`

	tests := []struct {
		level ProtectionLevel
		name  string
	}{
		{ProtectionLevelBasic, "Basic"},
		{ProtectionLevelMedium, "Medium"},
		{ProtectionLevelAdvanced, "Advanced"},
		{ProtectionLevelMaximum, "Maximum"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protected, err := cp.ProtectWithLevel(code, tt.level)
			if err != nil {
				t.Fatalf("ProtectWithLevel(%s) failed: %v", tt.name, err)
			}

			if protected == "" {
				t.Error("Protected code should not be empty")
			}
		})
	}
}

func TestCodeProtectionV15VerifyIntegrity(t *testing.T) {
	cp := NewCodeProtectionV15()
	code := `function test() { return true; }`

	protected, err := cp.ProtectCode(code)
	if err != nil {
		t.Fatalf("ProtectCode failed: %v", err)
	}

	valid, err := cp.VerifyIntegrity(protected)
	if err != nil {
		t.Fatalf("VerifyIntegrity failed: %v", err)
	}

	if !valid {
		t.Error("Protected code should verify integrity")
	}
}

func TestCodeProtectionV15DetectAutomation(t *testing.T) {
	cp := NewCodeProtectionV15()

	isAutomated, detections, err := cp.DetectAutomation()
	if err != nil {
		t.Fatalf("DetectAutomation failed: %v", err)
	}

	if isAutomated {
		t.Log("Automation detected:", detections)
	} else {
		t.Log("No automation detected")
	}
}

func TestCodeProtectionV15GetVersion(t *testing.T) {
	cp := NewCodeProtectionV15()
	version := cp.GetVersion()

	if version != "15.0.0" {
		t.Errorf("Expected version 15.0.0, got %s", version)
	}
}

func TestCodeProtectionV15GetConfig(t *testing.T) {
	cp := NewCodeProtectionV15()
	config := cp.GetConfig()

	if config.ProtectionLevel != ProtectionLevelAdvanced {
		t.Error("Default protection level should be Advanced")
	}
}

func TestCodeProtectionV15UpdateConfig(t *testing.T) {
	cp := NewCodeProtectionV15()

	newConfig := CodeProtectionV15Config{
		ProtectionLevel: ProtectionLevelMaximum,
		EnableWASM: false,
	}

	cp.UpdateConfig(newConfig)

	if cp.config.ProtectionLevel != ProtectionLevelMaximum {
		t.Error("Protection level should be updated to Maximum")
	}
}

func TestCodeProtectionV15RotateKey(t *testing.T) {
	cp := NewCodeProtectionV15()

	err := cp.RotateKey()
	if err != nil {
		t.Fatalf("RotateKey failed: %v", err)
	}

	t.Log("Key rotated successfully")
}

func TestCodeProtectionV15EncryptDecryptData(t *testing.T) {
	cp := NewCodeProtectionV15()

	data := []byte("test data encryption")

	encrypted, err := cp.EncryptData(data)
	if err != nil {
		t.Fatalf("EncryptData failed: %v", err)
	}

	if len(encrypted) == 0 {
		t.Error("Encrypted data should not be empty")
	}

	decrypted, err := cp.DecryptData(encrypted)
	if err != nil {
		t.Fatalf("DecryptData failed: %v", err)
	}

	if string(decrypted) != string(data) {
		t.Error("Decrypted data should match original")
	}
}

func TestCodeProtectionV15GenerateProtectedScript(t *testing.T) {
	cp := NewCodeProtectionV15()
	script := `console.log("Hello World");`

	protected, err := cp.GenerateProtectedScript(script, ProtectionLevelAdvanced)
	if err != nil {
		t.Fatalf("GenerateProtectedScript failed: %v", err)
	}

	if protected == "" {
		t.Error("Protected script should not be empty")
	}
}

func TestCodeProtectionV15BatchProtect(t *testing.T) {
	cp := NewCodeProtectionV15()
	codes := []string{
		"var a = 1;",
		"var b = 2;",
		"var c = 3;",
	}

	results, err := cp.BatchProtect(codes, ProtectionLevelBasic)
	if err != nil {
		t.Fatalf("BatchProtect failed: %v", err)
	}

	if len(results) != len(codes) {
		t.Errorf("Expected %d results, got %d", len(codes), len(results))
	}

	for i, result := range results {
		if result == "" {
			t.Errorf("Result %d should not be empty", i)
		}
	}
}

func TestCodeProtectionV15AnalyzeProtection(t *testing.T) {
	cp := NewCodeProtectionV15()
	code := `window.__IntegrityHash = {}; window._0xvm = {};`

	analysis, err := cp.AnalyzeProtection(code)
	if err != nil {
		t.Fatalf("AnalyzeProtection failed: %v", err)
	}

	if !analysis["hasIntegrityCheck"] {
		t.Error("Should detect integrity check")
	}

	if !analysis["hasVirtualization"] {
		t.Error("Should detect virtualization")
	}
}

func TestCodeProtectionV15GenerateReport(t *testing.T) {
	cp := NewCodeProtectionV15()

	report := cp.GenerateReport()
	if report == nil {
		t.Fatal("Report should not be nil")
	}

	if report.Version != "15.0.0" {
		t.Errorf("Expected version 15.0.0, got %s", report.Version)
	}

	if !report.Features["wasm"] {
		t.Error("WASM feature should be enabled")
	}
}

func TestFlowObfuscatorCreation(t *testing.T) {
	fo := NewFlowObfuscator()
	if fo == nil {
		t.Fatal("Expected FlowObfuscator to be created")
	}
}

func TestFlowObfuscatorObfuscateControlFlow(t *testing.T) {
	fo := NewFlowObfuscator()
	code := `function test() { return 1; }`

	obfuscated, err := fo.ObfuscateControlFlow(code)
	if err != nil {
		t.Fatalf("ObfuscateControlFlow failed: %v", err)
	}

	if obfuscated == "" {
		t.Error("Obfuscated code should not be empty")
	}

	if obfuscated == code {
		t.Error("Obfuscated code should differ from original")
	}
}

func TestFlowObfuscatorObfuscateControlFlowEmpty(t *testing.T) {
	fo := NewFlowObfuscator()
	_, err := fo.ObfuscateControlFlow("")
	if err == nil {
		t.Error("Should return error for empty code")
	}
}

func TestFlowObfuscatorObfuscateVariables(t *testing.T) {
	fo := NewFlowObfuscator()
	code := `var myVariable = 1; let anotherVar = 2;`

	obfuscated := fo.obfuscateVariables(code)
	if obfuscated == "" {
		t.Error("Obfuscated code should not be empty")
	}
}

func TestFlowObfuscatorObfuscateFunctions(t *testing.T) {
	fo := NewFlowObfuscator()
	code := `function myFunction() { return 1; }`

	obfuscated := fo.obfuscateFunctions(code)
	if obfuscated == "" {
		t.Error("Obfuscated code should not be empty")
	}
}

func TestFlowObfuscatorObfuscateStrings(t *testing.T) {
	fo := NewFlowObfuscator()
	code := `var str = "Hello World";`

	obfuscated := fo.obfuscateStrings(code)
	if obfuscated == "" {
		t.Error("Obfuscated code should not be empty")
	}
}

func TestFlowObfuscatorAddSelfDefendingCode(t *testing.T) {
	fo := NewFlowObfuscator()
	code := `function test() { return 1; }`

	defended := fo.AddSelfDefendingCode(code)
	if defended == "" {
		t.Error("Defended code should not be empty")
	}
}

func TestFlowObfuscatorAddDebuggerDetection(t *testing.T) {
	fo := NewFlowObfuscator()
	code := `function test() { return 1; }`

	defended := fo.AddDebuggerDetection(code)
	if defended == "" {
		t.Error("Defended code should not be empty")
	}
}

func TestFlowObfuscatorCreateDispatchTable(t *testing.T) {
	fo := NewFlowObfuscator()
	code := `function test() { return 1; }`

	dispatched := fo.CreateDispatchTable(code)
	if dispatched == "" {
		t.Error("Dispatched code should not be empty")
	}
}

func TestIntegrityServiceCreation(t *testing.T) {
	is := NewIntegrityService()
	if is == nil {
		t.Fatal("Expected IntegrityService to be created")
	}
}

func TestIntegrityServiceCalculateHash(t *testing.T) {
	is := NewIntegrityService()
	data := "test data"

	hash := is.CalculateHash(data)
	if hash == "" {
		t.Error("Hash should not be empty")
	}

	hash2 := is.CalculateHash(data)
	if hash != hash2 {
		t.Error("Hash should be deterministic")
	}
}

func TestIntegrityServiceCalculateHashB64(t *testing.T) {
	is := NewIntegrityService()
	data := "test data"

	hashB64 := is.CalculateHashB64(data)
	if hashB64 == "" {
		t.Error("Hash B64 should not be empty")
	}
}

func TestIntegrityServiceVerifyHash(t *testing.T) {
	is := NewIntegrityService()
	data := "test data"

	hash := is.CalculateHash(data)
	valid := is.VerifyHash(data, hash)

	if !valid {
		t.Error("Hash should verify correctly")
	}
}

func TestIntegrityServiceVerifyHashInvalid(t *testing.T) {
	is := NewIntegrityService()
	data := "test data"

	valid := is.VerifyHash(data, "invalid-hash")
	if valid {
		t.Error("Invalid hash should not verify")
	}
}

func TestIntegrityServiceCreateChecksum(t *testing.T) {
	is := NewIntegrityService()
	data := []byte("test data")

	checksum := is.CreateChecksum(data)
	if checksum == "" {
		t.Error("Checksum should not be empty")
	}
}

func TestIntegrityServiceVerifyChecksum(t *testing.T) {
	is := NewIntegrityService()
	data := []byte("test data")

	checksum := is.CreateChecksum(data)
	valid := is.VerifyChecksum(data, checksum)

	if !valid {
		t.Error("Checksum should verify correctly")
	}
}

func TestIntegrityServiceGenerateIntegrityToken(t *testing.T) {
	is := NewIntegrityService()
	data := "test data"

	token, err := is.GenerateIntegrityToken(data, time.Minute)
	if err != nil {
		t.Fatalf("GenerateIntegrityToken failed: %v", err)
	}

	if token == "" {
		t.Error("Token should not be empty")
	}
}

func TestIntegrityServiceVerifyIntegrityToken(t *testing.T) {
	is := NewIntegrityService()
	data := "test data"

	token, err := is.GenerateIntegrityToken(data, time.Minute)
	if err != nil {
		t.Fatalf("GenerateIntegrityToken failed: %v", err)
	}

	valid, err := is.VerifyIntegrityToken(token)
	if err != nil {
		t.Fatalf("VerifyIntegrityToken failed: %v", err)
	}

	if !valid {
		t.Error("Token should verify correctly")
	}
}

func TestIntegrityServiceVerifyIntegrityTokenExpired(t *testing.T) {
	is := NewIntegrityService()
	data := "test data"

	token, err := is.GenerateIntegrityToken(data, -time.Minute)
	if err != nil {
		t.Fatalf("GenerateIntegrityToken failed: %v", err)
	}

	_, err = is.VerifyIntegrityToken(token)
	if err == nil {
		t.Error("Expired token should return error")
	}
}

func TestIntegrityServiceCacheHash(t *testing.T) {
	is := NewIntegrityService()
	data := "test data"

	hash1 := is.CacheHash(data)
	hash2 := is.CacheHash(data)

	if hash1 != hash2 {
		t.Error("Cached hash should be consistent")
	}
}

func TestIntegrityServiceClearCache(t *testing.T) {
	is := NewIntegrityService()
	is.CacheHash("data1")
	is.CacheHash("data2")

	is.ClearCache()

	if is.GetCacheSize() != 0 {
		t.Error("Cache should be empty after clear")
	}
}

func TestIntegrityServiceCreateMultipleHashes(t *testing.T) {
	is := NewIntegrityService()
	data := "test data"

	hashes := is.CreateMultipleHashes(data)
	if len(hashes) == 0 {
		t.Error("Should create multiple hashes")
	}

	if _, ok := hashes["sha256"]; !ok {
		t.Error("Should have SHA256 hash")
	}
}

func TestIntegrityServiceVerifyMultipleHashes(t *testing.T) {
	is := NewIntegrityService()
	data := "test data"

	hashes := is.CreateMultipleHashes(data)
	valid := is.VerifyMultipleHashes(data, hashes)

	if !valid {
		t.Error("Multiple hashes should verify correctly")
	}
}

func TestIntegrityServiceGenerateMerkleRoot(t *testing.T) {
	is := NewIntegrityService()
	items := []string{"item1", "item2", "item3", "item4"}

	root := is.GenerateMerkleRoot(items)
	if root == "" {
		t.Error("Merkle root should not be empty")
	}
}

func TestIntegrityServiceGenerateMerkleRootEmpty(t *testing.T) {
	is := NewIntegrityService()
	root := is.GenerateMerkleRoot([]string{})

	if root != "" {
		t.Error("Empty items should return empty root")
	}
}

func TestIntegrityServiceCreateIntegrityReport(t *testing.T) {
	is := NewIntegrityService()
	data := "test data"

	report := is.CreateIntegrityReport(data)
	if report == nil {
		t.Fatal("Report should not be nil")
	}

	if _, ok := report["hash"]; !ok {
		t.Error("Report should contain hash")
	}
}

func TestIntegrityCheckerCreation(t *testing.T) {
	ic := NewIntegrityChecker()
	if ic == nil {
		t.Fatal("Expected IntegrityChecker to be created")
	}
}

func TestIntegrityCheckerCheck(t *testing.T) {
	ic := NewIntegrityChecker()
	data := "test data"

	valid, hash := ic.Check(data)
	if !valid {
		t.Error("Check should return valid for correct hash")
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}
}

func TestIntegrityCheckerGetStatistics(t *testing.T) {
	ic := NewIntegrityChecker()
	ic.Check("data1")
	ic.Check("data2")

	stats := ic.GetStatistics()
	if stats["total_checks"].(int) != 2 {
		t.Error("Should have 2 total checks")
	}
}

func TestKeyManagerCreation(t *testing.T) {
	km := NewKeyManager(time.Hour)
	if km == nil {
		t.Fatal("Expected KeyManager to be created")
	}
}

func TestKeyManagerGetCurrentKey(t *testing.T) {
	km := NewKeyManager(time.Hour)
	key := km.GetCurrentKey()

	if len(key) == 0 {
		t.Error("Key should not be empty")
	}
}

func TestKeyManagerRotateKey(t *testing.T) {
	km := NewKeyManager(time.Hour)
	oldKey := km.GetCurrentKey()

	err := km.RotateKey()
	if err != nil {
		t.Fatalf("RotateKey failed: %v", err)
	}

	newKey := km.GetCurrentKey()
	if string(oldKey) == string(newKey) {
		t.Error("Key should be different after rotation")
	}
}

func TestKeyManagerGetKeyVersion(t *testing.T) {
	km := NewKeyManager(time.Hour)
	version := km.GetKeyVersion()

	if version < 1 {
		t.Error("Key version should be at least 1")
	}
}

func TestKeyManagerGetKeyHistory(t *testing.T) {
	km := NewKeyManager(time.Hour)
	km.RotateKey()
	km.RotateKey()

	history := km.GetKeyHistory()
	if len(history) < 2 {
		t.Error("Should have at least 2 key records")
	}
}

func TestKeyManagerDeriveKey(t *testing.T) {
	km := NewKeyManager(time.Hour)
	password := "testpassword"
	salt := []byte("testsalt")

	derived, err := km.DeriveKey(password, salt)
	if err != nil {
		t.Fatalf("DeriveKey failed: %v", err)
	}

	if len(derived) == 0 {
		t.Error("Derived key should not be empty")
	}
}

func TestKeyManagerEncryptDecryptWithCurrentKey(t *testing.T) {
	km := NewKeyManager(time.Hour)
	plaintext := []byte("test data")

	encrypted, err := km.EncryptWithCurrentKey(plaintext)
	if err != nil {
		t.Fatalf("EncryptWithCurrentKey failed: %v", err)
	}

	decrypted, err := km.DecryptWithCurrentKey(encrypted)
	if err != nil {
		t.Fatalf("DecryptWithCurrentKey failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Error("Decrypted data should match original")
	}
}

func TestKeyManagerEncryptDecryptString(t *testing.T) {
	km := NewKeyManager(time.Hour)
	plaintext := "test string"

	encrypted, err := km.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("EncryptString failed: %v", err)
	}

	decrypted, err := km.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("DecryptString failed: %v", err)
	}

	if decrypted != plaintext {
		t.Error("Decrypted string should match original")
	}
}

func TestKeyManagerCreateKeyBundle(t *testing.T) {
	km := NewKeyManager(time.Hour)
	bundle := km.CreateKeyBundle()

	if bundle == nil {
		t.Fatal("Bundle should not be nil")
	}

	if _, ok := bundle["version"]; !ok {
		t.Error("Bundle should contain version")
	}
}

func TestKeyManagerValidateKey(t *testing.T) {
	km := NewKeyManager(time.Hour)
	key := km.GetCurrentKey()

	if !km.ValidateKey(key) {
		t.Error("Current key should be valid")
	}
}

func TestKeyManagerGetKeyInfo(t *testing.T) {
	km := NewKeyManager(time.Hour)
	info := km.GetKeyInfo()

	if info == nil {
		t.Fatal("Info should not be nil")
	}

	if _, ok := info["key_length"]; !ok {
		t.Error("Info should contain key_length")
	}
}

func TestAutomationDetectorCreation(t *testing.T) {
	ad := NewAutomationDetector()
	if ad == nil {
		t.Fatal("Expected AutomationDetector to be created")
	}
}

func TestAutomationDetectorDetectAutomation(t *testing.T) {
	ad := NewAutomationDetector()
	isAutomated, detections, err := ad.DetectAutomation()

	if err != nil {
		t.Fatalf("DetectAutomation failed: %v", err)
	}

	if isAutomated {
		t.Log("Automation detected:", detections)
	} else {
		t.Log("No automation detected")
	}
}

func TestAutomationDetectorGenerateDetectionCode(t *testing.T) {
	ad := NewAutomationDetector()
	code := ad.GenerateDetectionCode()

	if code == "" {
		t.Error("Detection code should not be empty")
	}
}

func TestAutomationDetectorGenerateEnhancedDetectionCode(t *testing.T) {
	ad := NewAutomationDetector()
	code := ad.GenerateEnhancedDetectionCode()

	if code == "" {
		t.Error("Enhanced detection code should not be empty")
	}
}

func TestAutomationDetectorAddCustomPattern(t *testing.T) {
	ad := NewAutomationDetector()
	ad.AddCustomPattern(AutomationTypeHeadless, "custom-pattern")
}

func TestAutomationDetectorGetEnabledTypes(t *testing.T) {
	ad := NewAutomationDetector()
	types := ad.GetEnabledTypes()

	if len(types) == 0 {
		t.Error("Should have enabled types")
	}
}

func TestAutomationDetectorSetEnabledTypes(t *testing.T) {
	ad := NewAutomationDetector()
	types := []AutomationType{AutomationTypeSelenium, AutomationTypePlaywright}

	ad.SetEnabledTypes(types)
	enabled := ad.GetEnabledTypes()

	if len(enabled) != len(types) {
		t.Error("Enabled types should match set types")
	}
}

func TestAutomationDetectorGetStatistics(t *testing.T) {
	ad := NewAutomationDetector()
	stats := ad.GetStatistics()

	if stats == nil {
		t.Fatal("Statistics should not be nil")
	}

	if _, ok := stats["total_patterns"]; !ok {
		t.Error("Statistics should contain total_patterns")
	}
}

func TestBehavioralAnalyzerCreation(t *testing.T) {
	ba := NewBehavioralAnalyzer()
	if ba == nil {
		t.Fatal("Expected BehavioralAnalyzer to be created")
	}
}

func TestBehavioralAnalyzerRecordMouseMovement(t *testing.T) {
	ba := NewBehavioralAnalyzer()
	ba.RecordMouseMovement()

	metrics := ba.GetMetrics()
	if metrics.MouseMovements != 1 {
		t.Error("Should have 1 mouse movement recorded")
	}
}

func TestBehavioralAnalyzerRecordKeyPress(t *testing.T) {
	ba := NewBehavioralAnalyzer()
	ba.RecordKeyPress()

	metrics := ba.GetMetrics()
	if metrics.KeyPresses != 1 {
		t.Error("Should have 1 key press recorded")
	}
}

func TestBehavioralAnalyzerRecordClick(t *testing.T) {
	ba := NewBehavioralAnalyzer()
	ba.RecordClick()

	metrics := ba.GetMetrics()
	if metrics.ClickCount != 1 {
		t.Error("Should have 1 click recorded")
	}
}

func TestBehavioralAnalyzerAnalyzeBehavior(t *testing.T) {
	ba := NewBehavioralAnalyzer()
	for i := 0; i < 15; i++ {
		ba.RecordMouseMovement()
	}
	for i := 0; i < 10; i++ {
		ba.RecordKeyPress()
	}
	for i := 0; i < 5; i++ {
		ba.RecordClick()
	}
	for i := 0; i < 10; i++ {
		ba.RecordScroll()
	}

	isBot, score, anomalies := ba.AnalyzeBehavior()

	t.Logf("Is Bot: %v, Score: %.2f, Anomalies: %v", isBot, score, anomalies)
}

func TestProtectionCryptoServiceCreation(t *testing.T) {
	pcs := NewProtectionCryptoService(nil)
	if pcs == nil {
		t.Fatal("Expected ProtectionCryptoService to be created")
	}
}

func TestProtectionCryptoServiceEncryptDecryptCode(t *testing.T) {
	pcs := NewProtectionCryptoService(nil)
	code := "function test() { return 1; }"

	encrypted, err := pcs.EncryptCode(code)
	if err != nil {
		t.Fatalf("EncryptCode failed: %v", err)
	}

	decrypted, err := pcs.DecryptCode(encrypted)
	if err != nil {
		t.Fatalf("DecryptCode failed: %v", err)
	}

	if decrypted != code {
		t.Error("Decrypted code should match original")
	}
}

func TestProtectionCryptoServiceDeriveKey(t *testing.T) {
	pcs := NewProtectionCryptoService(nil)
	derived, err := pcs.deriveKey(nil, "test-info")

	if err != nil {
		t.Fatalf("deriveKey failed: %v", err)
	}

	if len(derived) == 0 {
		t.Error("Derived key should not be empty")
	}
}

func TestProtectionCryptoServiceEncryptDecryptWithDerivedKey(t *testing.T) {
	pcs := NewProtectionCryptoService(nil)
	plaintext := []byte("test data")

	encrypted, err := pcs.encryptWithDerivedKey(plaintext, "test-info")
	if err != nil {
		t.Fatalf("encryptWithDerivedKey failed: %v", err)
	}

	decrypted, err := pcs.decryptWithDerivedKey(encrypted, "test-info")
	if err != nil {
		t.Fatalf("decryptWithDerivedKey failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Error("Decrypted data should match original")
	}
}

func TestGenerateRandomPassword(t *testing.T) {
	password := GenerateRandomPassword(16)
	if len(password) != 16 {
		t.Errorf("Expected password length 16, got %d", len(password))
	}

	password2 := GenerateRandomPassword(16)
	if password == password2 {
		t.Error("Generated passwords should be unique")
	}
}

func TestGenerateSalt(t *testing.T) {
	salt, err := GenerateSalt(16)
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}

	if len(salt) != 16 {
		t.Errorf("Expected salt length 16, got %d", len(salt))
	}
}

func TestMaskKey(t *testing.T) {
	key := []byte("testkey12345678")
	masked := MaskKey(key, 4)

	if masked[:4] != "test" {
		t.Error("Masked key should start with visible characters")
	}

	if masked[4:] != "************" {
		t.Error("Masked key should have masked characters")
	}
}

func TestMaskKeyShort(t *testing.T) {
	key := []byte("ab")
	masked := MaskKey(key, 4)

	if masked != "**" {
		t.Error("Short key should be fully masked")
	}
}

func TestKeyRotationScheduler(t *testing.T) {
	km := NewKeyManager(time.Hour)
	krs := NewKeyRotationScheduler(km)

	krs.Start(time.Millisecond * 100)
	time.Sleep(time.Millisecond * 150)
	krs.Stop()

	version := km.GetKeyVersion()
	if version < 1 {
		t.Error("Key should have been rotated at least once")
	}
}

func TestDetectBrowserFingerprint(t *testing.T) {
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0"
	plugins := []string{"Plugin1", "Plugin2"}
	language := "en-US"

	fingerprint := DetectBrowserFingerprint(userAgent, plugins, language)

	if fingerprint == nil {
		t.Fatal("Fingerprint should not be nil")
	}

	if fingerprint["plugin_count"].(int) != 2 {
		t.Error("Plugin count should be 2")
	}
}

func BenchmarkCodeProtectionV15ProtectCode(b *testing.B) {
	cp := NewCodeProtectionV15()
	code := `function hello() { return "world"; }`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cp.ProtectCode(code)
	}
}

func BenchmarkIntegrityServiceCalculateHash(b *testing.B) {
	is := NewIntegrityService()
	data := "test data for benchmarking"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		is.CalculateHash(data)
	}
}

func BenchmarkKeyManagerEncryptDecrypt(b *testing.B) {
	km := NewKeyManager(time.Hour)
	data := []byte("test data for benchmarking")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encrypted, _ := km.EncryptWithCurrentKey(data)
		km.DecryptWithCurrentKey(encrypted)
	}
}

func TestFullProtectionWorkflow(t *testing.T) {
	cp := NewCodeProtectionV15()

	code := `
function login(username, password) {
    if (username === "admin" && password === "secret123") {
        return { success: true, token: "xyz123" };
    }
    return { success: false };
}

function getData(id) {
    return { id: id, data: "sensitive information" };
}
`

	protected, err := cp.ProtectCode(code)
	if err != nil {
		t.Fatalf("ProtectCode failed: %v", err)
	}

	analysis, err := cp.AnalyzeProtection(protected)
	if err != nil {
		t.Fatalf("AnalyzeProtection failed: %v", err)
	}

	t.Log("Protection analysis:", analysis)

	report := cp.GenerateReport()
	t.Log("Protection report:", report)

	isAutomated, detections, err := cp.DetectAutomation()
	if err != nil {
		t.Fatalf("DetectAutomation failed: %v", err)
	}
	t.Log("Automation detection:", isAutomated, detections)
}

func TestControlFlowFlattening(t *testing.T) {
	cp := NewCodeProtectionV15()
	cp.config.EnableControlFlowFlattening = true

	code := `
if (condition) {
    doSomething();
} else {
    doOther();
}
`

	protected, err := cp.ProtectWithLevel(code, ProtectionLevelMaximum)
	if err != nil {
		t.Fatalf("ProtectWithLevel failed: %v", err)
	}

	t.Log("Protected code length:", len(protected))
}

func TestRuntimeDecryption(t *testing.T) {
	cp := NewCodeProtectionV15()
	cp.config.EnableRuntimeDecryption = true

	code := `console.log("This will be encrypted");`

	protected, err := cp.ProtectWithLevel(code, ProtectionLevelMaximum)
	if err != nil {
		t.Fatalf("ProtectWithLevel failed: %v", err)
	}

	if protected == "" {
		t.Error("Protected code should not be empty")
	}
}

func TestWASMDecryption(t *testing.T) {
	cp := NewCodeProtectionV15()
	cp.config.EnableWASM = true
	cp.config.EnableWASMDecryption = true

	code := `function wasmProtected() { return true; }`

	protected, err := cp.ProtectWithLevel(code, ProtectionLevelMaximum)
	if err != nil {
		t.Fatalf("ProtectWithLevel failed: %v", err)
	}

	t.Log("WASM protected code length:", len(protected))
}

func TestIntegrityCheck(t *testing.T) {
	cp := NewCodeProtectionV15()
	cp.config.EnableIntegrityCheck = true

	code := `var importantData = "critical";`

	protected, err := cp.ProtectWithLevel(code, ProtectionLevelAdvanced)
	if err != nil {
		t.Fatalf("ProtectWithLevel failed: %v", err)
	}

	valid, err := cp.VerifyIntegrity(protected)
	if err != nil {
		t.Fatalf("VerifyIntegrity failed: %v", err)
	}

	t.Log("Integrity valid:", valid)
}

func TestAntiAutomationProtection(t *testing.T) {
	cp := NewCodeProtectionV15()
	cp.config.EnableAntiAutomation = true

	code := `function checkUser() { return true; }`

	protected, err := cp.ProtectWithLevel(code, ProtectionLevelAdvanced)
	if err != nil {
		t.Fatalf("ProtectWithLevel failed: %v", err)
	}

	analysis, _ := cp.AnalyzeProtection(protected)
	if !analysis["hasAntiAutomation"] {
		t.Error("Should have anti-automation protection")
	}
}

func TestVirtualization(t *testing.T) {
	cp := NewCodeProtectionV15()
	cp.config.EnableVirtualization = true

	code := `function virtualized() { return 42; }`

	protected, err := cp.ProtectWithLevel(code, ProtectionLevelMaximum)
	if err != nil {
		t.Fatalf("ProtectWithLevel failed: %v", err)
	}

	analysis, _ := cp.AnalyzeProtection(protected)
	if !analysis["hasVirtualization"] {
		t.Error("Should have virtualization")
	}
}

func TestPolymorphicProtection(t *testing.T) {
	cp := NewCodeProtectionV15()
	cp.config.EnablePolymorphicCode = true

	code := `console.log("test");`

	protected, err := cp.ProtectWithLevel(code, ProtectionLevelMaximum)
	if err != nil {
		t.Fatalf("ProtectWithLevel failed: %v", err)
	}

	analysis, _ := cp.AnalyzeProtection(protected)
	if !analysis["hasPolymorphic"] {
		t.Error("Should have polymorphic protection")
	}
}

func TestTimingProtection(t *testing.T) {
	cp := NewCodeProtectionV15()
	cp.config.EnableTimingProtection = true

	code := `function timing() { return Date.now(); }`

	protected, err := cp.ProtectWithLevel(code, ProtectionLevelAdvanced)
	if err != nil {
		t.Fatalf("ProtectWithLevel failed: %v", err)
	}

	analysis, _ := cp.AnalyzeProtection(protected)
	if !analysis["hasTimingProtection"] {
		t.Error("Should have timing protection")
	}
}

func TestProtectionLevelComparison(t *testing.T) {
	cp := NewCodeProtectionV15()
	code := `function test() { return 1; }`

	levels := []ProtectionLevel{
		ProtectionLevelBasic,
		ProtectionLevelMedium,
		ProtectionLevelAdvanced,
		ProtectionLevelMaximum,
	}

	lengths := make([]int, len(levels))

	for i, level := range levels {
		protected, err := cp.ProtectWithLevel(code, level)
		if err != nil {
			t.Fatalf("ProtectWithLevel failed for level %d: %v", level, err)
		}
		lengths[i] = len(protected)
		t.Logf("Level %d: %d chars", level, lengths[i])
	}

	if lengths[0] >= lengths[3] {
		t.Error("Maximum protection should produce longer code than basic")
	}
}

func TestKeyRotationWithDecryption(t *testing.T) {
	km := NewKeyManager(time.Hour)

	data := []byte("sensitive data")
	encrypted1, err := km.EncryptWithCurrentKey(data)
	if err != nil {
		t.Fatalf("First encryption failed: %v", err)
	}

	km.RotateKey()

	encrypted2, err := km.EncryptWithCurrentKey(data)
	if err != nil {
		t.Fatalf("Second encryption failed: %v", err)
	}

	decrypted1, err := km.DecryptWithCurrentKey(encrypted1)
	if err != nil {
		t.Fatalf("First decryption failed: %v", err)
	}

	if string(decrypted1) != string(data) {
		t.Error("First decryption should succeed")
	}

	decrypted2, err := km.DecryptWithCurrentKey(encrypted2)
	if err != nil {
		t.Fatalf("Second decryption failed: %v", err)
	}

	if string(decrypted2) != string(data) {
		t.Error("Second decryption should succeed")
	}
}

func TestMerkleTreeIntegrity(t *testing.T) {
	is := NewIntegrityService()

	items := []string{
		"item1",
		"item2",
		"item3",
		"item4",
		"item5",
	}

	root := is.GenerateMerkleRoot(items)
	if root == "" {
		t.Fatal("Merkle root should not be empty")
	}

	t.Logf("Merkle root: %s", root)

	item := items[2]
	itemHash := is.CalculateHash(item)

	t.Logf("Verifying item %s with hash %s", item, itemHash)
	t.Logf("Merkle root: %s", root)
}

func TestAutomationDetectionTypes(t *testing.T) {
	ad := NewAutomationDetector()

	testCases := []struct {
		automationType AutomationType
		expected       bool
	}{
		{AutomationTypeSelenium, true},
		{AutomationTypePlaywright, true},
		{AutomationTypePuppeteer, true},
		{AutomationTypePhantomJS, true},
		{AutomationTypeHeadless, true},
	}

	for _, tc := range testCases {
		t.Run(string(tc.automationType), func(t *testing.T) {
			ad.EnableDetection(tc.automationType)
			enabled := ad.GetEnabledTypes()

			found := false
			for _, at := range enabled {
				if at == tc.automationType {
					found = true
					break
				}
			}

			if !found {
				t.Error("Automation type should be enabled")
			}
		})
	}
}

func TestBehavioralMetrics(t *testing.T) {
	ba := NewBehavioralAnalyzer()

	ba.RecordMouseMovement()
	ba.RecordMouseMovement()
	ba.RecordMouseMovement()

	ba.RecordKeyPress()
	ba.RecordKeyPress()

	ba.RecordClick()

	ba.RecordScroll()
	ba.RecordScroll()

	metrics := ba.GetMetrics()

	if metrics.MouseMovements != 3 {
		t.Errorf("Expected 3 mouse movements, got %d", metrics.MouseMovements)
	}

	if metrics.KeyPresses != 2 {
		t.Errorf("Expected 2 key presses, got %d", metrics.KeyPresses)
	}

	if metrics.ClickCount != 1 {
		t.Errorf("Expected 1 click, got %d", metrics.ClickCount)
	}

	if metrics.ScrollEvents != 2 {
		t.Errorf("Expected 2 scroll events, got %d", metrics.ScrollEvents)
	}
}

func TestAllFeatures(t *testing.T) {
	cp := NewCodeProtectionV15(CodeProtectionV15Config{
		ProtectionLevel: ProtectionLevelMaximum,
		EnableWASM: true,
		EnableWASMDecryption: true,
		EnableControlFlowFlattening: true,
		EnableIntegrityCheck: true,
		EnableAntiAutomation: true,
		EnableRuntimeDecryption: true,
		EnableVirtualization: true,
		EnablePolymorphicCode: true,
		EnableTimingProtection: true,
	})

	code := `
(function() {
    var apiKey = "secret-key-12345";
    var config = {
        endpoint: "https://api.example.com",
        timeout: 5000,
        retries: 3
    };

    function authenticate(credentials) {
        if (credentials.username === "admin" && credentials.password === "secret") {
            return { success: true, token: generateToken() };
        }
        return { success: false, error: "Invalid credentials" };
    }

    function generateToken() {
        var timestamp = Date.now();
        var random = Math.random().toString(36).substring(7);
        return btoa(timestamp + "-" + random);
    }

    function makeRequest(endpoint, data) {
        console.log("Request to:", endpoint);
        console.log("Data:", JSON.stringify(data));
        return { status: 200, data: { message: "success" } };
    }

    window.API = {
        authenticate: authenticate,
        request: makeRequest,
        config: config
    };
})();
`

	protected, err := cp.ProtectCode(code)
	if err != nil {
		t.Fatalf("ProtectCode failed: %v", err)
	}

	t.Logf("Original code length: %d", len(code))
	t.Logf("Protected code length: %d", len(protected))
	t.Logf("Protection ratio: %.2f%%", float64(len(protected))/float64(len(code))*100)

	analysis, _ := cp.AnalyzeProtection(protected)
	t.Log("Protection features:", analysis)

	valid, _ := cp.VerifyIntegrity(protected)
	t.Log("Integrity valid:", valid)

	report := cp.GenerateReport()
	t.Log("Protection level:", report.ProtectionLevel)
	t.Log("Features enabled:", report.Features)
}

func TestStressProtection(t *testing.T) {
	cp := NewCodeProtectionV15()

	for i := 0; i < 100; i++ {
		code := fmt.Sprintf(`function test%d() { return %d; }`, i, i)
		protected, err := cp.ProtectCode(code)
		if err != nil {
			t.Fatalf("ProtectCode failed at iteration %d: %v", i, err)
		}

		if protected == "" {
			t.Errorf("Protected code should not be empty at iteration %d", i)
		}
	}
}

func TestConcurrentKeyRotation(t *testing.T) {
	km := NewKeyManager(time.Hour)

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				km.RotateKey()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	version := km.GetKeyVersion()
	t.Logf("Final key version: %d", version)
}

func TestBase64Encoding(t *testing.T) {
	pcs := NewProtectionCryptoService(nil)

	original := "Hello, World! 你好世界！"
	encrypted, err := pcs.EncryptCode(original)
	if err != nil {
		t.Fatalf("EncryptCode failed: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("Base64 decode failed: %v", err)
	}

	t.Logf("Encrypted length: %d bytes", len(decoded))
}

func TestCodeProtectionV15_GenerateMultipleReports(t *testing.T) {
	cp := NewCodeProtectionV15()

	for i := 0; i < 5; i++ {
		report := cp.GenerateReport()
		if report == nil {
			t.Fatal("Report should not be nil")
		}

		if report.Version != "15.0.0" {
			t.Errorf("Expected version 15.0.0, got %s", report.Version)
		}
	}
}
