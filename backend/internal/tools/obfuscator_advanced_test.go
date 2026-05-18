package tools

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestFullObfuscation(t *testing.T) {
	code := `
document.addEventListener('DOMContentLoaded', function() {
    console.log('Test code loaded');

    const button = document.querySelector('.btn');
    if (button) {
        button.addEventListener('click', function() {
            console.log('Button clicked');
        });
    }
});
`

	config := ObfuscatorConfig{
		EnableVariableObfuscation:    true,
		EnableStringEncryption:      true,
		EnableCodeCompression:       true,
		EnableControlFlowFlattening: true,
		EnableFunctionWrapping:      true,
		EnableAdvancedAntiDebug:     true,
		EnableMemoryProtection:      true,
		EnableCodeIntegrity:        true,
		EnableAdvancedIntegrity:     true,
		StringEncryptionKey:        []byte("test-key-123"),
	}

	obfuscated, err := NewObfuscator(config).ApplyFullProtection(code)
	if err != nil {
		t.Fatalf("Failed to obfuscate code: %v", err)
	}

	if len(obfuscated) == 0 {
		t.Fatal("Obfuscated code is empty")
	}

	valid, errors := ValidateObfuscatedJS(obfuscated)
	if !valid {
		t.Errorf("Obfuscated JS validation failed: %v", errors)
	}

	fmt.Printf("Original code length: %d\n", len(code))
	fmt.Printf("Obfuscated code length: %d\n", len(obfuscated))
	fmt.Printf("Compression ratio: %.2f%%\n", float64(len(code)-len(obfuscated))/float64(len(code))*100)
}

func TestFileHashGeneration(t *testing.T) {
	code := `function test() { return "hello"; }`

	hash, err := GenerateFileHash(code, "sha256")
	if err != nil {
		t.Fatalf("Failed to generate SHA256 hash: %v", err)
	}

	if len(hash) != 64 {
		t.Errorf("Expected SHA256 hash length 64, got %d", len(hash))
	}

	hash384, err := GenerateFileHash(code, "sha384")
	if err != nil {
		t.Fatalf("Failed to generate SHA384 hash: %v", err)
	}

	if len(hash384) != 96 {
		t.Errorf("Expected SHA384 hash length 96, got %d", len(hash384))
	}

	hash512, err := GenerateFileHash(code, "sha512")
	if err != nil {
		t.Fatalf("Failed to generate SHA512 hash: %v", err)
	}

	if len(hash512) != 128 {
		t.Errorf("Expected SHA512 hash length 128, got %d", len(hash512))
	}

	md5Hash := GenerateMD5Hash(code)
	if len(md5Hash) != 32 {
		t.Errorf("Expected MD5 hash length 32, got %d", len(md5Hash))
	}
}

func TestIntegrityHashes(t *testing.T) {
	code := `var test = "integrity test";`

	hashes, err := GenerateIntegrityHashes(code)
	if err != nil {
		t.Fatalf("Failed to generate integrity hashes: %v", err)
	}

	if hashes.SHA256 == "" || hashes.SHA384 == "" || hashes.SHA512 == "" || hashes.MD5 == "" {
		t.Error("One or more hash values are empty")
	}

	fmt.Printf("SHA256: %s\n", hashes.SHA256)
	fmt.Printf("MD5: %s\n", hashes.MD5)
}

func TestCodeIntegrityVerifier(t *testing.T) {
	code := `console.log("test");`
	secret := "test-secret"

	verifier := GenerateCodeIntegrityVerifier(code, secret)
	if len(verifier) == 0 {
		t.Error("Generated verifier is empty")
	}

	valid, _ := ValidateObfuscatedJS(verifier)
	if !valid {
		t.Error("Verifier JS syntax is invalid")
	}
}

func TestFileHashReport(t *testing.T) {
	code := `
function example() {
    var x = 10;
    return x * 2;
}
`
	config := ObfuscatorConfig{
		StringEncryptionKey: []byte("report-test-key"),
	}

	report := GenerateFileHashReport(code, config)

	hashes, ok := report["hashes"].(map[string]string)
	if !ok {
		t.Fatal("Failed to get hashes from report")
	}

	if hashes["sha256"] == "" {
		t.Error("SHA256 hash is empty")
	}

	if hashes["crc32"] == "" {
		t.Error("CRC32 hash is empty")
	}

	reportJSON, _ := json.MarshalIndent(report, "", "  ")
	fmt.Printf("File Hash Report:\n%s\n", string(reportJSON))
}

func TestDynamicCodeLoader(t *testing.T) {
	loader := NewDynamicCodeLoader()

	loader.RegisterModule("module1", `function test1() { return "module1"; }`)
	loader.RegisterModule("module2", `function test2() { return "module2"; }`)

	loaderCode := loader.GenerateDynamicLoaderCode()
	if len(loaderCode) == 0 {
		t.Error("Generated loader code is empty")
	}

	valid, _ := ValidateObfuscatedJS(loaderCode)
	if !valid {
		t.Error("Loader JS syntax is invalid")
	}

	modules := map[string]string{
		"test": `console.log("dynamic module");`,
	}
	loaderWithModules, err := loader.GenerateLoaderWithModules(modules)
	if err != nil {
		t.Fatalf("Failed to generate loader with modules: %v", err)
	}

	valid = false
	valid, _ = ValidateObfuscatedJS(loaderWithModules)
	if !valid {
		t.Error("Loader with modules JS syntax is invalid")
	}

	fmt.Printf("Loader code length: %d\n", len(loaderCode))
	fmt.Printf("Loader with modules length: %d\n", len(loaderWithModules))
}

func TestEnhancedAntiDebug(t *testing.T) {
	antiDebug := GenerateEnhancedAntiDebug()
	if len(antiDebug) == 0 {
		t.Error("Generated anti-debug code is empty")
	}

	valid, _ := ValidateObfuscatedJS(antiDebug)
	if !valid {
		t.Error("Anti-debug JS syntax is invalid")
	}

	fmt.Printf("Anti-debug code length: %d\n", len(antiDebug))
}

func TestAdvancedCodeProtection(t *testing.T) {
	code := `
(function() {
    var secret = "protected";
    console.log(secret);
})();
`

	config := ObfuscatorConfig{
		EnableAdvancedAntiDebug: true,
		EnableAdvancedIntegrity: true,
		EnableDynamicLoading:    true,
		StringEncryptionKey:      []byte("protection-key"),
	}

	protected := GenerateAdvancedCodeProtection(code, config)
	if len(protected) == 0 {
		t.Error("Protected code is empty")
	}

	valid, _ := ValidateObfuscatedJS(protected)
	if !valid {
		t.Error("Protected JS syntax is invalid")
	}

	fmt.Printf("Protected code length: %d\n", len(protected))
}

func TestCodeObfuscationReport(t *testing.T) {
	code := `
function calculate(a, b) {
    var result = a + b;
    return result * 2;
}

var data = [1, 2, 3, 4, 5];
console.log(calculate(10, 20));
`

	config := ObfuscatorConfig{
		EnableVariableObfuscation:    true,
		EnableStringEncryption:      true,
		EnableCodeCompression:       true,
		EnableControlFlowFlattening: true,
		StringEncryptionKey:          []byte("report-key-123"),
	}

	report := GenerateCodeObfuscationReport(code, config)

	reportJSON, _ := json.MarshalIndent(report, "", "  ")
	fmt.Printf("Obfuscation Report:\n%s\n", string(reportJSON))

	qualityScore, ok := report["quality_score"].(float64)
	if !ok {
		t.Error("Failed to get quality score")
	}

	fmt.Printf("Quality Score: %.2f\n", qualityScore)
}

func TestMultipleHashAlgorithm(t *testing.T) {
	code := `const data = "test data for hashing";`

	algorithms := []string{"sha256", "sha384", "sha512", "md5"}

	for _, algo := range algorithms {
		hash, err := GenerateFileHash(code, algo)
		if err != nil {
			t.Errorf("Failed to generate %s hash: %v", algo, err)
		}

		if hash == "" {
			t.Errorf("Generated %s hash is empty", algo)
		}

		fmt.Printf("%s: %s\n", algo, hash)
	}
}

func TestObfuscatorOptions(t *testing.T) {
	opts := NewObfuscatorOptions()
	if opts == nil {
		t.Fatal("Expected obfuscator options to be created")
	}
	if opts.ObfuscationStrength != 3 {
		t.Errorf("Expected default strength 3, got %d", opts.ObfuscationStrength)
	}
	if !opts.EnableCompression {
		t.Error("Expected compression to be enabled by default")
	}
	if !opts.EnableAntiTamper {
		t.Error("Expected anti-tamper to be enabled by default")
	}
	if !opts.EnableRuntimeCheck {
		t.Error("Expected runtime check to be enabled by default")
	}
}

func TestObfuscatorOptionsStrength(t *testing.T) {
	opts := NewObfuscatorOptions()

	opts.ObfuscationStrength = 0
	if opts.GetStrength() != 1 {
		t.Error("Strength 0 should be clamped to 1")
	}

	opts.ObfuscationStrength = 10
	if opts.GetStrength() != 5 {
		t.Error("Strength 10 should be clamped to 5")
	}

	opts.ObfuscationStrength = 3
	if opts.GetStrength() != 3 {
		t.Error("Strength 3 should remain 3")
	}
}

func TestObfuscatorOptionsHelpers(t *testing.T) {
	opts := NewObfuscatorOptions()

	opts.EnableCompression = true
	if !opts.IsCompressionEnabled() {
		t.Error("Compression should be enabled")
	}

	opts.EnableAntiTamper = true
	if !opts.IsAntiTamperEnabled() {
		t.Error("Anti-tamper should be enabled")
	}

	opts.EnableRuntimeCheck = true
	if !opts.IsRuntimeCheckEnabled() {
		t.Error("Runtime check should be enabled")
	}
}

func TestEnhancedVariableNameGeneration(t *testing.T) {
	obfuscator := NewObfuscator()
	names := make(map[string]bool)

	for i := 0; i < 200; i++ {
		name := obfuscator.generateObfuscatedName()
		if names[name] {
			t.Errorf("Generated duplicate name at iteration %d: %s", i, name)
		}
		names[name] = true

		if len(name) < 2 {
			t.Errorf("Generated name too short: %s", name)
		}
	}

	fmt.Printf("Generated %d unique names\n", len(names))
}

func TestAntiTamperConfig(t *testing.T) {
	config := &AntiTamperConfig{
		EnableChecksum:      true,
		EnableMarkers:       true,
		EnableIntegrityTest: true,
		EnableMutationTest:  true,
		EnableTimeLock:     true,
		CheckInterval:      5000,
		MarkerCount:        10,
	}

	if !config.EnableChecksum {
		t.Error("Checksum should be enabled")
	}
	if config.CheckInterval != 5000 {
		t.Errorf("Expected interval 5000, got %d", config.CheckInterval)
	}
	if config.MarkerCount != 10 {
		t.Errorf("Expected marker count 10, got %d", config.MarkerCount)
	}
}

func TestGenerateAntiTamperProtection(t *testing.T) {
	code := `function test() { return true; }`
	secret := "test-secret-123"

	result := GenerateAntiTamperProtection(code, secret, nil)
	if len(result) == 0 {
		t.Error("Generated anti-tamper protection should not be empty")
	}

	if !strings.Contains(result, code) {
		t.Error("Original code should be preserved")
	}
}

func TestGenerateChecksumProtection(t *testing.T) {
	hash := "abc123def456"
	interval := 5000

	result := GenerateChecksumProtection(hash, interval)
	if !strings.Contains(result, "window.__CP") {
		t.Error("Should contain window.__CP")
	}
	if !strings.Contains(result, hash) {
		t.Error("Should contain the hash")
	}
}

func TestGenerateMarkerProtection(t *testing.T) {
	hash := "testhash123"
	count := 5

	result := GenerateMarkerProtection(hash, count)
	if !strings.Contains(result, "_0xMP_m0") {
		t.Error("Should contain marker m0")
	}
	if !strings.Contains(result, "_0xMP_m4") {
		t.Error("Should contain marker m4 for count 5")
	}
}

func TestGenerateIntegrityTest(t *testing.T) {
	hash := "integrityhash"
	interval := 8000

	result := GenerateIntegrityTest(hash, interval)
	if !strings.Contains(result, "window.__IT") {
		t.Error("Should contain window.__IT")
	}
}

func TestGenerateMutationTest(t *testing.T) {
	result := GenerateMutationTest()
	if !strings.Contains(result, "MutationObserver") {
		t.Error("Should contain MutationObserver")
	}
	if !strings.Contains(result, "window.__MT") {
		t.Error("Should contain window.__MT")
	}
}

func TestGenerateTimeLock(t *testing.T) {
	result := GenerateTimeLock()
	if !strings.Contains(result, "Time lock") {
		t.Error("Should contain Time lock message")
	}
	if !strings.Contains(result, "window.__TL") {
		t.Error("Should contain window.__TL")
	}
}

func TestRuntimeValidator(t *testing.T) {
	validator := NewRuntimeValidator()
	if validator == nil {
		t.Fatal("Runtime validator should be created")
	}

	validator.RegisterCriticalFunction("testFn", "function testFn() { return true; }")
	validator.RegisterHash("code", "somehash123")

	if len(validator.criticalFunctions) != 1 {
		t.Error("Should have 1 critical function registered")
	}
	if len(validator.hashValues) != 1 {
		t.Error("Should have 1 hash value registered")
	}

	code := validator.GenerateValidatorCode()
	if !strings.Contains(code, "window.__RV") {
		t.Error("Generated validator should contain window.__RV")
	}
}

func TestGenerateRuntimeValidationCode(t *testing.T) {
	code := `function test() { return true; }`
	criticalFns := map[string]string{
		"test": "function test() { return true; }",
	}
	secret := "secret123"

	result := GenerateRuntimeValidationCode(code, criticalFns, secret)
	if !strings.Contains(result, code) {
		t.Error("Original code should be preserved")
	}
	if !strings.Contains(result, "window.__RV") {
		t.Error("Should contain window.__RV")
	}
}

func TestGenerateCodeTamperDetection(t *testing.T) {
	code := `function protected() { return true; }`
	secret := "tamper-secret"

	result := GenerateCodeTamperDetection(code, secret)
	if !strings.Contains(result, code) {
		t.Error("Original code should be preserved")
	}
	if !strings.Contains(result, "window.__CTD") {
		t.Error("Should contain window.__CTD")
	}
}

func TestApplyEnhancedObfuscation(t *testing.T) {
	code := `function hello() { return "world"; }`

	opts := NewObfuscatorOptions()
	opts.ObfuscationStrength = 3
	opts.EnableAntiTamper = true
	opts.EnableRuntimeCheck = true

	result, err := ApplyEnhancedObfuscation(code, opts)
	if err != nil {
		t.Fatalf("Enhanced obfuscation failed: %v", err)
	}

	if result == "" {
		t.Error("Result should not be empty")
	}
	if result == code {
		t.Error("Code should be obfuscated")
	}

	valid, errors := ValidateObfuscatedJS(result)
	if !valid {
		t.Errorf("Obfuscated JS validation failed: %v", errors)
	}
}

func TestApplyEnhancedObfuscationWithNilOptions(t *testing.T) {
	code := `function test() { return true; }`

	result, err := ApplyEnhancedObfuscation(code, nil)
	if err != nil {
		t.Fatalf("Enhanced obfuscation with nil options failed: %v", err)
	}

	if result == "" {
		t.Error("Result should not be empty")
	}
}

func TestApplyEnhancedObfuscationLevels(t *testing.T) {
	code := `function calculate(a, b) { return a + b; }`

	for level := 1; level <= 5; level++ {
		opts := NewObfuscatorOptions()
		opts.ObfuscationStrength = level

		result, err := ApplyEnhancedObfuscation(code, opts)
		if err != nil {
			t.Errorf("Level %d failed: %v", level, err)
		}

		if result == "" {
			t.Errorf("Level %d produced empty result", level)
		}
	}
}

func TestGenerateSelfDefendingCode(t *testing.T) {
	code := `function defend() { return true; }`
	secret := "defend-secret"

	result := GenerateSelfDefendingCode(code, secret)
	if !strings.Contains(result, code) {
		t.Error("Original code should be preserved")
	}
	if !strings.Contains(result, "window.__SDC") {
		t.Error("Should contain window.__SDC")
	}
}

func TestEnhancedObfuscationWithCompression(t *testing.T) {
	code := `function  test()  {
    var   x   =   10;
    return  x;
}`

	opts := NewObfuscatorOptions()
	opts.EnableCompression = true

	result, _ := ApplyEnhancedObfuscation(code, opts)

	if strings.Contains(result, "  ") {
		t.Error("Whitespace should be compressed")
	}
}

func TestEnhancedObfuscationWithoutCompression(t *testing.T) {
	code := `function test() { return true; }`

	opts := NewObfuscatorOptions()
	opts.EnableCompression = false

	result, _ := ApplyEnhancedObfuscation(code, opts)
	if result == "" {
		t.Error("Result should not be empty")
	}
}

func TestEnhancedObfuscationWithoutAntiTamper(t *testing.T) {
	code := `function test() { return true; }`

	opts := NewObfuscatorOptions()
	opts.EnableAntiTamper = false

	result, _ := ApplyEnhancedObfuscation(code, opts)

	valid, _ := ValidateObfuscatedJS(result)
	if !valid {
		t.Error("Result should be valid JS without anti-tamper")
	}
}

func TestEnhancedObfuscationWithoutRuntimeCheck(t *testing.T) {
	code := `function test() { return true; }`

	opts := NewObfuscatorOptions()
	opts.EnableRuntimeCheck = false

	result, _ := ApplyEnhancedObfuscation(code, opts)
	if result == "" {
		t.Error("Result should not be empty")
	}
}

func TestChecksumVerification(t *testing.T) {
	code := `var test = "checksum test";`
	hash := HashCode(code)

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if !VerifyCodeIntegrity(hash, code) {
		t.Error("Code should verify against its hash")
	}
}

func TestMarkerProtectionUniqueness(t *testing.T) {
	code := `function test() { return true; }`
	secret := "marker-secret"

	result1 := GenerateMarkerProtection(HashCode(code), 5)
	result2 := GenerateMarkerProtection(HashCode(code), 5)

	hash1 := HashCode(result1)
	hash2 := HashCode(result2)

	if hash1 != hash2 {
		t.Error("Same inputs should produce same marker protection")
	}
}

func TestIntegrityTestSnapshot(t *testing.T) {
	code := `function snapshot() { return true; }`
	hash := HashCode(code)

	result := GenerateIntegrityTest(hash, 10000)
	if !strings.Contains(result, "snapshot") {
		t.Error("Should contain snapshot functionality")
	}
	if !strings.Contains(result, "window.__IT") {
		t.Error("Should contain window.__IT")
	}
}

func TestTamperDetectionSegments(t *testing.T) {
	code := `function detect() { return true; }`
	secret := "detection-secret"

	result := GenerateCodeTamperDetection(code, secret)

	if !strings.Contains(result, "_0xCTD") {
		t.Error("Should contain _0xCTD")
	}

	if !strings.Contains(result, "segments") {
		t.Error("Should contain segments for hash partitioning")
	}
}

func TestSelfDefendingVerification(t *testing.T) {
	code := `function defend() { return true; }`
	secret := "defend-secret"

	result := GenerateSelfDefendingCode(code, secret)

	if !strings.Contains(result, "verifyLength") {
		t.Error("Should contain verifyLength function")
	}
	if !strings.Contains(result, "criticalPoints") {
		t.Error("Should contain criticalPoints array")
	}
}

func TestVariableNameUniqueness(t *testing.T) {
	obfuscator := NewObfuscator()
	seen := make(map[string]bool)

	for i := 0; i < 500; i++ {
		name := obfuscator.generateObfuscatedName()
		if seen[name] {
			t.Errorf("Name collision at iteration %d: %s", i, name)
		}
		seen[name] = true
	}
}

func TestVariableNameLengthDistribution(t *testing.T) {
	obfuscator := NewObfuscator()
	lengths := make(map[int]int)

	for i := 0; i < 300; i++ {
		name := obfuscator.generateObfuscatedName()
		lengths[len(name)]++
	}

	minLen := 0
	maxLen := 0
	for l := range lengths {
		if l < minLen || minLen == 0 {
			minLen = l
		}
		if l > maxLen {
			maxLen = l
		}
	}

	fmt.Printf("Name length distribution: min=%d, max=%d\n", minLen, maxLen)

	if maxLen < 2 {
		t.Error("Names should have reasonable length")
	}
}

func TestObfuscatorMutexSafety(t *testing.T) {
	code := `function test() { return true; }`
	done := make(chan bool, 20)

	for i := 0; i < 20; i++ {
		go func() {
			obfuscator := NewObfuscator()
			_, err := obfuscator.Obfuscate(code)
			if err != nil {
				t.Errorf("Concurrent obfuscation error: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}
