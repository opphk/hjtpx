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
