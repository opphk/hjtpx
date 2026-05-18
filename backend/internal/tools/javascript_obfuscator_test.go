package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestObfuscatorCreation(t *testing.T) {
	obfuscator := NewObfuscator()
	if obfuscator == nil {
		t.Fatal("Expected obfuscator to be created")
	}
	if obfuscator.config.EnableVariableObfuscation != true {
		t.Error("Default config should enable variable obfuscation")
	}
	if obfuscator.config.EnableStringEncryption != true {
		t.Error("Default config should enable string encryption")
	}
}

func TestObfuscatorWithCustomConfig(t *testing.T) {
	config := ObfuscatorConfig{
		EnableVariableObfuscation:   false,
		EnableStringEncryption:      false,
		EnableCodeCompression:       false,
		EnableControlFlowFlattening: false,
		StringEncryptionKey:         []byte("test-key-1234567890"),
	}
	obfuscator := NewObfuscator(config)
	if obfuscator.config.EnableVariableObfuscation != false {
		t.Error("Custom config should disable variable obfuscation")
	}
}

func TestObfuscateBasic(t *testing.T) {
	code := `function hello() { return "world"; }`
	obfuscator := NewObfuscator()
	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}
	if result == "" {
		t.Error("Obfuscated result should not be empty")
	}
	if result == code {
		t.Error("Obfuscated result should differ from original")
	}
}

func TestObfuscateEmptyCode(t *testing.T) {
	obfuscator := NewObfuscator()
	_, err := obfuscator.Obfuscate("")
	if err == nil {
		t.Error("Expected error for empty code")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Error("Error message should mention empty code")
	}
}

func TestRemoveComments(t *testing.T) {
	code := `// single line comment
function test() {
/* multi
line
comment */
return true;
}`
	obfuscator := NewObfuscator(ObfuscatorConfig{
		RemoveComments: true,
	})
	result := obfuscator.removeComments(code)

	if strings.Contains(result, "// single line comment") {
		t.Error("Single line comments should be removed")
	}
	if strings.Contains(result, "multi") && strings.Contains(result, "line") {
		t.Error("Multi line comments should be removed")
	}
}

func TestObfuscateVariables(t *testing.T) {
	code := `var myVariable = 10;
let anotherVar = "test";
const PI = 3.14;`
	obfuscator := NewObfuscator(ObfuscatorConfig{
		EnableVariableObfuscation: true,
	})
	result := obfuscator.obfuscateVariables(code)

	if strings.Contains(result, "myVariable") {
		t.Error("Variable names should be obfuscated")
	}
	if strings.Contains(result, "anotherVar") {
		t.Error("Variable names should be obfuscated")
	}
}

func TestReservedWordsNotObfuscated(t *testing.T) {
	code := `if (condition) { return true; }
for (var i = 0; i < 10; i++) { }
while (true) { break; }`
	obfuscator := NewObfuscator(ObfuscatorConfig{
		EnableVariableObfuscation: true,
	})
	result := obfuscator.obfuscateVariables(code)

	reservedWords := []string{"if", "return", "for", "while", "break"}
	for _, word := range reservedWords {
		if !strings.Contains(result, word) {
			t.Errorf("Reserved word '%s' should not be obfuscated", word)
		}
	}
}

func TestObfuscateStrings(t *testing.T) {
	code := `var url = "https://api.example.com";
var token = "Bearer xyz123";`
	obfuscator := NewObfuscator(ObfuscatorConfig{
		EnableStringEncryption: true,
		StringEncryptionKey:    []byte("test-key-1234567890"),
	})
	result := obfuscator.encryptStrings(code)

	if strings.Contains(result, "https://api.example.com") {
		t.Error("Strings should be encrypted")
	}
	if !strings.Contains(result, "__d") {
		t.Error("Encrypted strings should use decoder function")
	}
}

func TestEncryptString(t *testing.T) {
	plaintext := "Hello, World!"
	key := []byte("test-key-1234567890")

	encrypted, err := EncryptString(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptString failed: %v", err)
	}
	if encrypted == plaintext {
		t.Error("Encrypted string should differ from plaintext")
	}
}

func TestDecryptString(t *testing.T) {
	plaintext := "Hello, World!"
	key := []byte("test-key-1234567890")

	encrypted, err := EncryptString(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptString failed: %v", err)
	}

	decrypted, err := DecryptString(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptString failed: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("Decrypted string should match original, got %q", decrypted)
	}
}

func TestDecryptStringWrongKey(t *testing.T) {
	plaintext := "Hello, World!"
	key1 := []byte("key1-1234567890ab")
	key2 := []byte("key2-1234567890ab")

	encrypted, _ := EncryptString(plaintext, key1)

	_, err := DecryptString(encrypted, key2)
	if err == nil {
		t.Error("Decryption with wrong key should fail")
	}
}

func TestCompressCode(t *testing.T) {
	code := `function  test()  {
    var   x   =   10;
    return  x;
}`
	obfuscator := NewObfuscator(ObfuscatorConfig{
		EnableCodeCompression: true,
		CompressWhitespace:    true,
	})
	result := obfuscator.compressCode(code)

	if strings.Contains(result, "  ") {
		t.Error("Multiple spaces should be compressed")
	}
}

func TestCompressCodeDisabled(t *testing.T) {
	code := "function  test()  {   return 1;  }"
	obfuscator := NewObfuscator(ObfuscatorConfig{
		EnableCodeCompression: false,
	})
	result := obfuscator.compressCode(code)
	if result != code {
		t.Error("Code should not be compressed when disabled")
	}
}

func TestWrapCode(t *testing.T) {
	code := `function test() { return 1; }`
	obfuscator := NewObfuscator(ObfuscatorConfig{
		EnableFunctionWrapping: true,
	})
	result := obfuscator.wrapCode(code)

	if !strings.Contains(result, "window.__d=") {
		t.Error("Wrapped code should include decoder function")
	}
	if !strings.HasPrefix(result, "(function") {
		t.Error("Wrapped code should start with IIFE")
	}
}

func TestFlattenControlFlow(t *testing.T) {
	code := `if (x > 0) { console.log("positive"); }
for (var i = 0; i < 10; i++) { sum += i; }`
	obfuscator := NewObfuscator(ObfuscatorConfig{
		EnableControlFlowFlattening: true,
	})
	result := obfuscator.flattenControlFlow(code)

	if strings.Contains(result, "if (x > 0)") && !strings.Contains(result, "_0xF1") {
		t.Error("Control flow should be flattened with obfuscated variables")
	}
}

func TestGenerateObfuscatedName(t *testing.T) {
	obfuscator := NewObfuscator()
	names := make(map[string]bool)

	for i := 0; i < 100; i++ {
		name := obfuscator.generateObfuscatedName()
		if names[name] {
			t.Errorf("Generated duplicate name: %s", name)
		}
		names[name] = true
		if !strings.HasPrefix(name, "_0x") {
			t.Errorf("Generated name should start with _0x, got %s", name)
		}
	}
}

func TestGenerateRandomString(t *testing.T) {
	obfuscator := NewObfuscator()
	length := 16

	str1 := obfuscator.generateRandomString(length)
	str2 := obfuscator.generateRandomString(length)

	if len(str1) != length {
		t.Errorf("String length should be %d, got %d", length, len(str1))
	}
	if str1 == str2 {
		t.Error("Generated strings should be unique")
	}
}

func TestInjectDeadCode(t *testing.T) {
	code := `function test() { return true; }`
	obfuscator := NewObfuscator(ObfuscatorConfig{
		EnableDeadCodeInjection: true,
	})
	result := obfuscator.injectDeadCode(code)

	if !strings.Contains(result, "(function(){") {
		t.Error("Dead code should be wrapped in IIFE")
	}
	if !strings.Contains(result, code) {
		t.Error("Original code should be preserved")
	}
}

func TestAnalyzeCode(t *testing.T) {
	code := `// comment
function hello() {
	var x = 10;
	var y = 20;
	if (x > y) { return true; }
	return false;
}`
	analyzer := AnalyzeCode(code)

	if analyzer.comments < 1 {
		t.Error("Should detect at least one comment")
	}
	if analyzer.functions < 1 {
		t.Error("Should detect at least one function")
	}
	if analyzer.variables < 2 {
		t.Error("Should detect at least two variables")
	}
	if analyzer.linesOfCode < 1 {
		t.Error("Should detect at least one line")
	}
}

func TestCodeAnalyzerMetrics(t *testing.T) {
	code := `function test() { return 1; }`
	analyzer := AnalyzeCode(code)
	metrics := analyzer.GetMetrics()

	if _, ok := metrics["lines_of_code"]; !ok {
		t.Error("Metrics should include lines_of_code")
	}
	if _, ok := metrics["functions"]; !ok {
		t.Error("Metrics should include functions")
	}
	if _, ok := metrics["variables"]; !ok {
		t.Error("Metrics should include variables")
	}
}

func TestCalculateObfuscationRatio(t *testing.T) {
	analyzer := &CodeAnalyzer{}
	original := "very long original string here"
	obfuscated := "s"

	ratio := analyzer.CalculateObfuscationRatio(original, obfuscated)
	if ratio <= 0 {
		t.Error("Ratio should be positive for smaller obfuscated code")
	}

	ratio = analyzer.CalculateObfuscationRatio("", obfuscated)
	if ratio != 0 {
		t.Error("Ratio should be 0 for empty original")
	}
}

func TestValidateObfuscatedCode(t *testing.T) {
	validCode := `function test() { return true; }`
	valid, msg := ValidateObfuscatedCode(validCode)
	if !valid {
		t.Errorf("Valid code should pass validation: %s", msg)
	}

	todoCode := `function test() { TODO: fix this; }`
	valid, _ = ValidateObfuscatedCode(todoCode)
	if valid {
		t.Error("Code with TODO should fail validation")
	}

	unbalancedCode := `function test() { return true;`
	valid, _ = ValidateObfuscatedCode(unbalancedCode)
	if valid {
		t.Error("Unbalanced braces should fail validation")
	}
}

func TestGenerateObfuscationReport(t *testing.T) {
	original := `function hello() { return "world"; }`
	obfuscated := `function _0x1(){return"world";}`
	config := ObfuscatorConfig{
		EnableVariableObfuscation: true,
	}

	report := GenerateObfuscationReport(original, obfuscated, config)

	if _, ok := report["original"]; !ok {
		t.Error("Report should include original section")
	}
	if _, ok := report["obfuscated"]; !ok {
		t.Error("Report should include obfuscated section")
	}
	if _, ok := report["compression_ratio"]; !ok {
		t.Error("Report should include compression_ratio")
	}
}

func TestObfuscateFile(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "input.js")
	outputFile := filepath.Join(tmpDir, "output.js")

	code := `function hello() { return "world"; }`
	if err := os.WriteFile(inputFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := ObfuscateFile(inputFile, outputFile)
	if err != nil {
		t.Fatalf("ObfuscateFile failed: %v", err)
	}

	output, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(output) == 0 {
		t.Error("Output file should not be empty")
	}
}

func TestObfuscateFileNotFound(t *testing.T) {
	err := ObfuscateFile("/nonexistent/input.js", "/tmp/output.js")
	if err == nil {
		t.Error("Should return error for nonexistent file")
	}
}

func TestStringDecoder(t *testing.T) {
	key := []byte("test-key-1234567890")
	decoder := NewStringDecoder(key)

	encrypted, _ := EncryptString("hello world", key)

	err := decoder.RegisterDecoder(1, encrypted)
	if err != nil {
		t.Fatalf("RegisterDecoder failed: %v", err)
	}

	decoded, ok := decoder.GetDecoded(1)
	if !ok {
		t.Error("Should find registered decoder")
	}
	if decoded != "hello world" {
		t.Errorf("Decoded value should match, got %s", decoded)
	}
}

func TestStringDecoderDecode(t *testing.T) {
	key := []byte("test-key-1234567890")
	decoder := NewStringDecoder(key)

	encrypted, _ := EncryptString("test data", key)

	decoded, err := decoder.Decode(encrypted)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if decoded != "test data" {
		t.Errorf("Decoded should match original, got %s", decoded)
	}
}

func TestGenerateRandomKey(t *testing.T) {
	key1, err := GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("GenerateRandomKey failed: %v", err)
	}
	if len(key1) != 32 {
		t.Errorf("Key length should be 32, got %d", len(key1))
	}

	key2, _ := GenerateRandomKey(32)
	if string(key1) == string(key2) {
		t.Error("Generated keys should be unique")
	}
}

func TestGenerateRandomKeyLengthValidation(t *testing.T) {
	key, _ := GenerateRandomKey(8)
	if len(key) != 32 {
		t.Errorf("Key length should default to 32, got %d", len(key))
	}

	key, _ = GenerateRandomKey(128)
	if len(key) != 64 {
		t.Errorf("Key length should be capped at 64, got %d", len(key))
	}
}

func TestGenerateHexKey(t *testing.T) {
	hexKey, err := GenerateHexKey(32)
	if err != nil {
		t.Fatalf("GenerateHexKey failed: %v", err)
	}
	if len(hexKey) != 64 {
		t.Errorf("Hex key should be 64 chars, got %d", len(hexKey))
	}
}

func TestValidateKey(t *testing.T) {
	validKey := []byte("Abcdefgh12345678")
	if !ValidateKey(validKey) {
		t.Error("Valid key should pass validation")
	}

	shortKey := []byte("short")
	if ValidateKey(shortKey) {
		t.Error("Short key should fail validation")
	}

	noLowerKey := []byte("ABCDEFGH12345678")
	if ValidateKey(noLowerKey) {
		t.Error("Key without lowercase should fail validation")
	}

	noUpperKey := []byte("abcdefgh12345678")
	if ValidateKey(noUpperKey) {
		t.Error("Key without uppercase should fail validation")
	}

	noDigitKey := []byte("Abcdefghijklmnop")
	if ValidateKey(noDigitKey) {
		t.Error("Key without digit should fail validation")
	}
}

func TestHashCode(t *testing.T) {
	code := "test code"
	hash1 := HashCode(code)
	hash2 := HashCode(code)

	if hash1 != hash2 {
		t.Error("Same code should produce same hash")
	}

	hash3 := HashCode("different code")
	if hash1 == hash3 {
		t.Error("Different code should produce different hash")
	}
}

func TestVerifyCodeIntegrity(t *testing.T) {
	code := "original code"
	hash := HashCode(code)

	if !VerifyCodeIntegrity(hash, code) {
		t.Error("Original code should verify against its hash")
	}

	if VerifyCodeIntegrity(hash, "modified code") {
		t.Error("Modified code should not verify against original hash")
	}
}

func TestCreateIntegrityCheck(t *testing.T) {
	code := "test code"
	check := CreateIntegrityCheck(code)

	if !strings.Contains(check, "window.__h=") {
		t.Error("Integrity check should include window.__h")
	}
}

func TestExtractIntegrityHash(t *testing.T) {
	code := "var hash = 'abc123';"
	code += "window.__h='" + code + "';"

	hash, found := ExtractIntegrityHash(code)
	if !found {
		t.Error("Should find integrity hash in code")
	}
	if hash == "" {
		t.Error("Extracted hash should not be empty")
	}
}

func TestGenerateCodeSignature(t *testing.T) {
	code := "test code"
	secret := "secret key"

	sig1 := GenerateCodeSignature(code, secret)
	sig2 := GenerateCodeSignature(code, secret)

	if sig1 != sig2 {
		t.Error("Same inputs should produce same signature")
	}

	sig3 := GenerateCodeSignature("different code", secret)
	if sig1 == sig3 {
		t.Error("Different code should produce different signature")
	}
}

func TestVerifyCodeSignature(t *testing.T) {
	code := "test code"
	secret := "secret key"
	signature := GenerateCodeSignature(code, secret)

	if !VerifyCodeSignature(code, signature, secret) {
		t.Error("Valid signature should verify")
	}

	if VerifyCodeSignature("modified", signature, secret) {
		t.Error("Modified code should not verify")
	}
}

func TestObfuscationOptions(t *testing.T) {
	opts := NewObfuscationOptions()
	if opts.Seed != 12345 {
		t.Error("Default seed should be 12345")
	}
	if opts.TargetObfuscationRate != 0.7 {
		t.Error("Default target obfuscation rate should be 0.7")
	}
}

func TestGetRandomInt(t *testing.T) {
	for i := 0; i < 100; i++ {
		val := GetRandomInt(1, 10)
		if val < 1 || val > 10 {
			t.Errorf("Random int should be between 1 and 10, got %d", val)
		}
	}
}

func TestGetRandomFloat(t *testing.T) {
	for i := 0; i < 100; i++ {
		val := GetRandomFloat()
		if val < 0 || val >= 1 {
			t.Errorf("Random float should be between 0 and 1, got %f", val)
		}
	}
}

func TestInjectAntiDebug(t *testing.T) {
	code := `function test() { return true; }`
	result := InjectAntiDebug(code)

	if !strings.Contains(result, "window.outerHeight") {
		t.Error("Anti-debug code should include window size check")
	}
	if !strings.Contains(result, code) {
		t.Error("Original code should be preserved")
	}
}

func TestCreateCodeIntegrityModule(t *testing.T) {
	code := `function test() { return true; }`
	key := []byte("test-key-1234567890")

	result := CreateCodeIntegrityModule(code, key)

	if !strings.Contains(result, "_0xI") {
		t.Error("Integrity module should include hash variable")
	}
	if !strings.Contains(result, code) {
		t.Error("Original code should be preserved")
	}
}

func TestOptimizeObfuscationLevel1(t *testing.T) {
	code := `var myVariable = "test value";`
	result := OptimizeObfuscation(code, 1)

	if result == code {
		t.Error("Code should be obfuscated at level 1")
	}
}

func TestOptimizeObfuscationLevel2(t *testing.T) {
	code := `var url = "https://example.com";`
	result := OptimizeObfuscation(code, 2)

	if strings.Contains(result, "https://example.com") {
		t.Error("Strings should be encrypted at level 2")
	}
}

func TestOptimizeObfuscationLevel3(t *testing.T) {
	code := `function test() { return true; }`
	result := OptimizeObfuscation(code, 3)

	if result == code {
		t.Error("Code should be obfuscated at level 3")
	}
}

func TestOptimizeObfuscationInvalidLevel(t *testing.T) {
	code := `var myVariable = "test value";`

	result := OptimizeObfuscation(code, 0)
	if result == code {
		t.Error("Level 0 should be treated as level 1")
	}

	result = OptimizeObfuscation(code, 5)
	if result == code {
		t.Error("Level 5 should be treated as level 3")
	}
}

func TestGetObfuscationLevel(t *testing.T) {
	simpleCode := `function test() { return true; }`
	level := GetObfuscationLevel(simpleCode)
	if level < 1 || level > 3 {
		t.Errorf("Simple code should return level 1-3, got %d", level)
	}

	complexCode := `if(a){if(b){if(c){if(d){if(e){if(f){if(g){if(h){if(i){if(j){}}}}} }}}} }`
	level = GetObfuscationLevel(complexCode)
	if level < 1 || level > 3 {
		t.Errorf("Complex code should return level 1-3, got %d", level)
	}
}

func TestEstimateObfuscationTime(t *testing.T) {
	time := EstimateObfuscationTime(100)
	if !strings.HasSuffix(time, "ms") {
		t.Error("Small code should return time in milliseconds")
	}

	time = EstimateObfuscationTime(50000)
	if !strings.HasSuffix(time, "s") && !strings.HasSuffix(time, "m") {
		t.Error("Large code should return time in seconds or minutes")
	}
}

func TestObfuscatorGetStats(t *testing.T) {
	code := `var a = 1; var b = "test";`
	obfuscator := NewObfuscator()
	obfuscator.Obfuscate(code)

	stats := obfuscator.GetStats()
	if stats["variables_obfuscated"] < 0 {
		t.Error("Variables obfuscated count should be non-negative")
	}
}

func TestObfuscateWithConfig(t *testing.T) {
	code := `function hello() { return "world"; }`
	config := ObfuscatorConfig{
		EnableVariableObfuscation: true,
		EnableStringEncryption:    false,
		RemoveComments:            true,
	}

	result, err := ObfuscateWithConfig(code, config)
	if err != nil {
		t.Fatalf("ObfuscateWithConfig failed: %v", err)
	}
	if result == "" {
		t.Error("Result should not be empty")
	}
}

func TestFullObfuscationPipeline(t *testing.T) {
	originalCode := `// This is a test function
function calculateSum(a, b) {
    var result = a + b;
    console.log("Sum is: " + result);
    return result;
}

// Another function
function multiply(x, y) {
    var product = x * y;
    return product;
}`

	config := ObfuscatorConfig{
		EnableVariableObfuscation:   true,
		EnableStringEncryption:      true,
		EnableCodeCompression:       true,
		EnableControlFlowFlattening: true,
		EnableFunctionWrapping:      true,
		RemoveComments:              true,
		PreserveConsole:             true,
		StringEncryptionKey:         []byte("test-key-1234567890"),
	}

	obfuscator := NewObfuscator(config)
	obfuscated, err := obfuscator.Obfuscate(originalCode)
	if err != nil {
		t.Fatalf("Full obfuscation pipeline failed: %v", err)
	}

	if obfuscated == originalCode {
		t.Error("Code should be obfuscated")
	}

	if obfuscated == "" {
		t.Error("Obfuscated code should not be empty")
	}

	report := GenerateObfuscationReport(originalCode, obfuscated, config)
	if report == nil {
		t.Error("Report should not be nil")
	}
}

func TestConcurrentObfuscation(t *testing.T) {
	code := `function test() { return true; }`
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			obfuscator := NewObfuscator()
			_, err := obfuscator.Obfuscate(code)
			if err != nil {
				t.Errorf("Concurrent obfuscation failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestObfuscationDeterminism(t *testing.T) {
	code := `var myVar = "test value";`

	obfuscator1 := NewObfuscator(ObfuscatorConfig{
		StringEncryptionKey: []byte("fixed-key-12345678"),
	})
	result1, _ := obfuscator1.Obfuscate(code)

	obfuscator2 := NewObfuscator(ObfuscatorConfig{
		StringEncryptionKey: []byte("fixed-key-12345678"),
	})
	result2, _ := obfuscator2.Obfuscate(code)

	if result1 != result2 {
		t.Error("Same code with same key should produce deterministic results")
	}
}

func TestEncryptStringsDynamic(t *testing.T) {
	code := `var url = "https://api.example.com";`

	obfuscator := NewObfuscator(ObfuscatorConfig{
		EnableStringEncryption: true,
	})
	result := obfuscator.encryptStringsDynamic(code)

	if strings.Contains(result, "https://api.example.com") {
		t.Error("Dynamic string encryption should encrypt strings")
	}
}

func TestGenerateDynamicDecryptor(t *testing.T) {
	obfuscator := NewObfuscator()
	decoderVar := "_0xTest"
	result := obfuscator.generateDynamicDecryptor(decoderVar)

	if !strings.Contains(result, "atob") {
		t.Error("Dynamic decryptor should use atob for base64 decoding")
	}
	if !strings.Contains(result, decoderVar) {
		t.Error("Dynamic decryptor should use the provided decoder variable name")
	}
}

func TestInjectEnhancedAntiDebug(t *testing.T) {
	code := `function test() { return true; }`

	obfuscator := NewObfuscator()
	result := obfuscator.InjectEnhancedAntiDebug(code)

	if !strings.Contains(result, "keydown") {
		t.Error("Enhanced anti-debug should listen for keydown events")
	}
	if !strings.Contains(result, "e.key==='F12'") {
		t.Error("Enhanced anti-debug should detect F12 key")
	}
}

func TestInjectSelfDestruct(t *testing.T) {
	code := `function test() { return true; }`

	obfuscator := NewObfuscator()
	err := obfuscator.InjectSelfDestruct(code)

	if err != nil {
		t.Error("Self destruct injection should not return error")
	}
}

func TestAddMemoryProtection(t *testing.T) {
	code := `function test() { return true; }`

	obfuscator := NewObfuscator()
	result := obfuscator.AddMemoryProtection(code)

	if !strings.Contains(result, "Object.defineProperty") {
		t.Error("Memory protection should use Object.defineProperty")
	}
}

func TestApplyAdvancedObfuscation(t *testing.T) {
	code := `function hello() { return "world"; }`

	obfuscator := NewObfuscator(ObfuscatorConfig{
		EnableVariableObfuscation:   true,
		EnableStringEncryption:      true,
		EnableControlFlowFlattening: true,
		EnableDeadCodeInjection:     true,
		EnableFunctionWrapping:      true,
	})

	result, err := obfuscator.ApplyAdvancedObfuscation(code)
	if err != nil {
		t.Fatalf("Advanced obfuscation failed: %v", err)
	}

	if result == code {
		t.Error("Advanced obfuscation should modify code")
	}
}

func TestWrapCodeAdvanced(t *testing.T) {
	code := `function test() { return true; }`

	obfuscator := NewObfuscator()
	result := obfuscator.wrapCodeAdvanced(code)

	if !strings.Contains(result, "window") {
		t.Error("Advanced wrapping should use window object")
	}
}

func TestInjectDeadCodeAdvanced(t *testing.T) {
	code := `function test() { return true; }`

	obfuscator := NewObfuscator()
	result := obfuscator.injectDeadCodeAdvanced(code)

	if !strings.Contains(result, "Math.random()") {
		t.Error("Advanced dead code should use random values")
	}
}

func TestGenerateRandomIntExpr(t *testing.T) {
	obfuscator := NewObfuscator()
	result := obfuscator.generateRandomIntExpr()

	if result == "" {
		t.Error("Random int expression should not be empty")
	}
}

func TestGenerateRandomBoolExpr(t *testing.T) {
	obfuscator := NewObfuscator()
	result := obfuscator.generateRandomBoolExpr()

	if !strings.Contains(result, ">") {
		t.Error("Random bool expression should use comparison operator")
	}
}

func TestCompressCodeAdvanced(t *testing.T) {
	code := `function  test()  {
    var   x   =   10;
}`

	obfuscator := NewObfuscator(ObfuscatorConfig{
		CompressWhitespace: true,
	})
	result := obfuscator.compressCodeAdvanced(code)

	if strings.Contains(result, "  ") {
		t.Error("Advanced compression should remove multiple spaces")
	}
}

func TestCalculateObfuscationEntropy(t *testing.T) {
	code := `function test() { return true; }`

	entropy := CalculateObfuscationEntropy(code)
	if entropy <= 0 {
		t.Error("Entropy should be positive for non-empty code")
	}

	emptyEntropy := CalculateObfuscationEntropy("")
	if emptyEntropy != 0 {
		t.Error("Entropy should be 0 for empty code")
	}
}

func TestEstimateObfuscationQuality(t *testing.T) {
	original := `function test() { return "hello world"; }`
	obfuscated := `function _0x1(){return _0x2;}`

	quality := EstimateObfuscationQuality(original, obfuscated)

	if _, ok := quality["entropy_original"]; !ok {
		t.Error("Quality estimate should include entropy_original")
	}
	if _, ok := quality["entropy_obfuscated"]; !ok {
		t.Error("Quality estimate should include entropy_obfuscated")
	}
	if _, ok := quality["overall_quality"]; !ok {
		t.Error("Quality estimate should include overall_quality")
	}
}

func TestGenerateObfuscationCertificate(t *testing.T) {
	original := `function test() { return true; }`
	obfuscated := `function _0x1(){return!1;}`

	config := ObfuscatorConfig{
		EnableVariableObfuscation: true,
		EnableStringEncryption:    true,
		EnableCodeCompression:     true,
	}

	cert := GenerateObfuscationCertificate(original, obfuscated, config)

	if !strings.Contains(cert, "代码混淆证书") {
		t.Error("Certificate should contain title")
	}
}

func TestCreateSelfCheckingCode(t *testing.T) {
	code := `function test() { return true; }`
	key := []byte("test-key-1234567890")

	result := CreateSelfCheckingCode(code, key)

	if !strings.Contains(result, "data-hash") {
		t.Error("Self-checking code should include hash attribute")
	}
	if !strings.Contains(result, code) {
		t.Error("Original code should be preserved")
	}
}

// ========== 增强控制流平坦化测试 ==========

func TestFlattenControlFlowAdvanced(t *testing.T) {
	code := `if (x > 0) { console.log("positive"); } else { console.log("negative"); }
for (var i = 0; i < 10; i++) { sum += i; }
while (true) { break; }`

	obfuscator := NewObfuscator(ObfuscatorConfig{
		EnableControlFlowFlattening: true,
	})
	result := obfuscator.flattenControlFlowAdvanced(code)

	if !strings.Contains(result, "switch(") {
		t.Error("Advanced control flow flattening should use switch statements")
	}
	if !strings.Contains(result, "case") {
		t.Error("Advanced control flow flattening should use case statements")
	}
}

func TestFlattenIfStatements(t *testing.T) {
	code := `if (condition) { doSomething(); } else { doOther(); }`
	obfuscator := NewObfuscator()
	result := obfuscator.flattenIfStatements(code)

	if !strings.Contains(result, "switch(") {
		t.Error("Flattened if statements should use switch")
	}
}

func TestFlattenForLoops(t *testing.T) {
	code := `for (var i = 0; i < 10; i++) { sum += i; }`
	obfuscator := NewObfuscator()
	result := obfuscator.flattenForLoops(code)

	if !strings.Contains(result, "for(;;)") {
		t.Error("Flattened for loops should use infinite loop")
	}
}

func TestFlattenWhileLoops(t *testing.T) {
	code := `while (condition) { doSomething(); }`
	obfuscator := NewObfuscator()
	result := obfuscator.flattenWhileLoops(code)

	if !strings.Contains(result, "for(;;)") {
		t.Error("Flattened while loops should use infinite loop")
	}
}

func TestAddOpaquePredicates(t *testing.T) {
	code := `var x = 10;`
	obfuscator := NewObfuscator()
	result := obfuscator.addOpaquePredicates(code)

	if !strings.Contains(result, "Math.random()") {
		t.Error("Opaque predicates should use random values")
	}
}

func TestAddMultiLevelStateMachine(t *testing.T) {
	code := `var x = 1;`
	obfuscator := NewObfuscator()
	result := obfuscator.addMultiLevelStateMachine(code)

	if !strings.Contains(result, "states") {
		t.Error("Multi-level state machine should have states array")
	}
}

// ========== 增强字符串加密测试 ==========

func TestEncryptStringMultiRound(t *testing.T) {
	obfuscator := NewObfuscator(ObfuscatorConfig{
		StringEncryptionKey: []byte("test-multi-key"),
	})
	result := obfuscator.encryptStringMultiRound("hello world")

	if !strings.Contains(result, "__mr") {
		t.Error("Multi-round encryption should use __mr decoder")
	}
	if result == "hello world" {
		t.Error("Multi-round encryption should modify the string")
	}
}

func TestEncryptStringCustomTable(t *testing.T) {
	obfuscator := NewObfuscator()
	result := obfuscator.encryptStringCustomTable("test string")

	if !strings.Contains(result, "__ct") {
		t.Error("Custom table encryption should use __ct decoder")
	}
	if result == "test string" {
		t.Error("Custom table encryption should modify the string")
	}
}

func TestEncryptStringAESBase64(t *testing.T) {
	obfuscator := NewObfuscator(ObfuscatorConfig{
		StringEncryptionKey: []byte("test-aes-base64-key"),
	})
	result := obfuscator.encryptStringAESBase64("secret data")

	if !strings.Contains(result, "__ab") {
		t.Error("AES+Base64 encryption should use __ab decoder")
	}
	if result == "secret data" {
		t.Error("AES+Base64 encryption should modify the string")
	}
}

func TestScrambleBytes(t *testing.T) {
	obfuscator := NewObfuscator()
	data := []byte{1, 2, 3, 4, 5}
	result := obfuscator.scrambleBytes(data, 1)

	if len(result) != len(data) {
		t.Error("Scrambled bytes should have same length")
	}
}

// ========== 增强代码虚拟化测试 ==========

func TestCreateVirtualization(t *testing.T) {
	code := `function test() { return true; }`
	obfuscator := NewObfuscator(ObfuscatorConfig{
		EnableCodeVirtualization: true,
		StringEncryptionKey:      []byte("test-vm-key"),
	})
	result := obfuscator.createVirtualization(code)

	if !strings.Contains(result, "_0xVM") {
		t.Error("Virtualization should include VM object")
	}
	if !strings.Contains(result, "atob") {
		t.Error("Virtualization should use base64 decoding")
	}
}

func TestCreateAdvancedVirtualization(t *testing.T) {
	code := `console.log("test");`
	obfuscator := NewObfuscator()
	result := obfuscator.createAdvancedVirtualization(code)

	if !strings.Contains(result, "_0xAVM") {
		t.Error("Advanced virtualization should include AVM object")
	}
	if !strings.Contains(result, "DECODE") {
		t.Error("Advanced virtualization should have DECODE function")
	}
}

// ========== 增强反调试机制测试 ==========

func TestInjectEnhancedAntiDebugWithSixChecks(t *testing.T) {
	code := `function test() { return true; }`
	obfuscator := NewObfuscator()
	result := obfuscator.InjectEnhancedAntiDebug(code)

	checks := []string{
		"checkWindowSize",
		"checkDebuggerTiming",
		"checkConsoleAPI",
		"checkFunctionToString",
		"checkFirebug",
		"checkExceptionHandler",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("Enhanced anti-debug should contain %s check", check)
		}
	}
}

func TestAntiDebugHasSixDetectionMethods(t *testing.T) {
	result := GenerateEnhancedAntiDebug()

	checks := []string{
		"detectDevTools",
		"window_size",
		"webkitDebuggerAPI",
		"firebug",
		"_commandLineAPI",
		"function_debugger",
	}

	count := 0
	for _, check := range checks {
		if strings.Contains(result, check) {
			count++
		}
	}

	if count != 6 {
		t.Errorf("Expected 6 detection methods, found %d", count)
	}
}

// ========== 综合测试 ==========

func TestFullAdvancedObfuscation(t *testing.T) {
	code := `function calculate(a, b) {
    var result = a + b;
    console.log("Result: " + result);
    return result;
}`

	config := ObfuscatorConfig{
		EnableVariableObfuscation:   true,
		EnableStringEncryption:      true,
		StringEncryptionMethod:      "multi-enc",
		EnableControlFlowFlattening: true,
		EnableCodeCompression:       true,
		EnableFunctionWrapping:      true,
		EnableAdvancedAntiDebug:     true,
		EnableCodeVirtualization:    true,
		StringEncryptionKey:         []byte("test-full-key"),
	}

	obfuscator := NewObfuscator(config)
	result, err := obfuscator.ApplyFullProtection(code)

	if err != nil {
		t.Fatalf("Full obfuscation failed: %v", err)
	}

	if len(result) == 0 {
		t.Error("Obfuscated code should not be empty")
	}

	if result == code {
		t.Error("Obfuscated code should differ from original")
	}
}

func TestMultipleStringEncryptionMethods(t *testing.T) {
	code := `var msg = "secret message";`

	methods := []string{"aes-gcm", "rc4", "chacha20", "xor", "multi-enc", "custom-table", "aes-base64"}

	for _, method := range methods {
		obfuscator := NewObfuscator(ObfuscatorConfig{
			EnableStringEncryption:   true,
			StringEncryptionMethod:  method,
			StringEncryptionKey:     []byte("test-method-key"),
		})
		result := obfuscator.encryptStrings(code)

		if !strings.Contains(result, "__") {
			t.Errorf("Method %s should produce encrypted strings with decoder function", method)
		}
	}
}
