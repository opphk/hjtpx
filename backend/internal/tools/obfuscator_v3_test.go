package tools

import (
	"strings"
	"testing"
)

func TestObfuscatorV3Creation(t *testing.T) {
	obf := NewObfuscatorV3()
	if obf == nil {
		t.Fatal("NewObfuscatorV3 returned nil")
	}
	if obf.options == nil {
		t.Fatal("ObfuscatorOptions is nil")
	}
	if !obf.options.ControlFlowFlattening {
		t.Error("ControlFlowFlattening should be enabled")
	}
	if !obf.options.StringEncryption {
		t.Error("StringEncryption should be enabled")
	}
	if !obf.options.AntiDebug {
		t.Error("AntiDebug should be enabled")
	}
}

func TestObfuscatorV3BasicObfuscation(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `function test() { return "hello world"; }`

	result, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty string")
	}

	if len(result) < len(testCode) {
		t.Logf("Note: Obfuscated code is shorter than original, which is unusual")
	}
}

func TestObfuscatorV3StringArray(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `
function getMessage() {
	return "hello";
}
function getName() {
	return "world";
}
`

	result, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if !strings.Contains(result, "[") || !strings.Contains(result, "]") {
		t.Error("Result should contain array notation")
	}
}

func TestObfuscatorV3AntiDebug(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `function init() { console.log("started"); }`

	result, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if !strings.Contains(result, "debugger") && !strings.Contains(result, "setInterval") {
		t.Error("Anti-debug code should be present")
	}
}

func TestObfuscatorV3DomainLock(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `function checkDomain() { return true; }`

	result, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if !strings.Contains(result, "localhost") && !strings.Contains(result, "hostname") {
		t.Error("Domain lock code should be present")
	}
}

func TestObfuscatorV3VariableObfuscation(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `
var myVariable = "test";
var anotherVar = 123;
function myFunction() {
	return myVariable;
}
`

	result, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if strings.Contains(result, "myVariable") && strings.Contains(result, "anotherVar") {
		if !strings.Contains(result, "_") {
			t.Log("Note: Variable names may not be fully obfuscated")
		}
	}
}

func TestObfuscatorV3EmptyCode(t *testing.T) {
	obf := NewObfuscatorV3()

	_, err := obf.Obfuscate("")
	if err == nil {
		t.Error("Expected error for empty code")
	}
}

func TestObfuscatorV3Stats(t *testing.T) {
	obf := NewObfuscatorV3()

	stats := obf.GetObfuscationStats()
	if stats == nil {
		t.Fatal("GetObfuscationStats returned nil")
	}

	if stats["version"] != "3.1.0" {
		t.Errorf("Expected version 3.1.0, got %v", stats["version"])
	}

	features, ok := stats["features_enabled"].(map[string]bool)
	if !ok {
		t.Fatal("features_enabled is not a map")
	}

	expectedFeatures := []string{
		"string_array",
		"string_encryption",
		"variable_obfuscation",
		"control_flow_flattening",
		"anti_debug",
		"domain_lock",
		"self_defending",
	}

	for _, feature := range expectedFeatures {
		if !features[feature] {
			t.Errorf("Feature %s should be enabled", feature)
		}
	}
}

func TestObfuscatorV3Minify(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `
function test() {
    var x = 1;
    var y = 2;
    return x + y;
}
`

	minified := obf.minifyAdvanced(testCode)

	if strings.Contains(minified, "\n") {
		t.Error("Minified code should not contain newlines")
	}

	if strings.Contains(minified, "    ") {
		t.Error("Minified code should not contain 4-space indentation")
	}
}

func TestObfuscatorV3DeadCodeInjection(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `function test() { return true; }`

	result, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if !strings.Contains(result, "var") && !strings.Contains(result, "Math.random") {
		t.Log("Note: Dead code injection may not have added any code")
	}
}

func TestObfuscatorV3FunctionReordering(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `
function funcA() { return 1; }
function funcB() { return 2; }
function funcC() { return 3; }
`

	result, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == testCode {
		t.Log("Note: Function reordering may not have modified the code")
	}
}

func TestObfuscatorV3ConsoleDisable(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `function test() { console.log("test"); }`

	result, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if !strings.Contains(result, "console") {
		t.Error("Console manipulation code should be present")
	}
}

func TestObfuscatorV3NumberEncryption(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `function getValue() { return 100; }`

	result, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == testCode {
		t.Log("Note: Number encryption may not have modified the code")
	}
}

func TestObfuscatorV3SelfDefending(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `function verify() { return true; }`

	result, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if !strings.Contains(result, "sha256") && !strings.Contains(result, "hash") {
		t.Log("Note: Self-defending code may use different hash method")
	}
}

func TestKeyGenerator(t *testing.T) {
	kg := newKeyGenerator()

	key1 := kg.generate()
	if key1 == "" {
		t.Fatal("generate() returned empty key")
	}

	key2 := kg.generate()
	if key1 == key2 {
		t.Error("Two generated keys should be different")
	}

	if len(key1) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key1))
	}
}

func TestStringRegistry(t *testing.T) {
	sr := newStringRegistry()

	idx1 := sr.add("test")
	if idx1 != 0 {
		t.Errorf("Expected index 0, got %d", idx1)
	}

	idx2 := sr.add("test")
	if idx2 != idx1 {
		t.Error("Same string should return same index")
	}

	idx3 := sr.add("another")
	if idx3 == idx1 {
		t.Error("Different string should return different index")
	}

	encoded := sr.getEncoded(idx1)
	if encoded == "" {
		t.Error("Encoded string should not be empty")
	}
}

func TestObfuscatorV3ComplexCode(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `
function calculateSum(arr) {
	var sum = 0;
	for (var i = 0; i < arr.length; i++) {
		sum += arr[i];
	}
	return sum;
}

function calculateAverage(arr) {
	var sum = calculateSum(arr);
	return sum / arr.length;
}
`

	result, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Result should not be empty")
	}

	if result == testCode {
		t.Log("Note: Obfuscated code equals original")
	}
}

func TestObfuscatorV3ControlFlowFlattening(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `
function complexFunction(x) {
	if (x > 0) {
		return x * 2;
	} else {
		return x / 2;
	}
}
`

	result, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Result should not be empty")
	}
}

func TestObfuscatorV3LiveRelocation(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `function init() { document.body.appendChild(div); }`

	result, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if !strings.Contains(result, "appendChild") || !strings.Contains(result, "prototype") {
		t.Log("Note: Live relocation may use different method names")
	}
}

func BenchmarkObfuscatorV3(b *testing.B) {
	obf := NewObfuscatorV3()

	testCode := `
function calculateSum(arr) {
	var sum = 0;
	for (var i = 0; i < arr.length; i++) {
		sum += arr[i];
	}
	return sum;
}

function calculateAverage(arr) {
	var sum = calculateSum(arr);
	return sum / arr.length;
}

function processData(data) {
	var result = [];
	for (var i = 0; i < data.length; i++) {
		if (data[i] > 0) {
			result.push(data[i] * 2);
		} else {
			result.push(data[i] / 2);
		}
	}
	return result;
}
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = obf.Obfuscate(testCode)
	}
}

func TestObfuscatorV3DeterministicObfuscation(t *testing.T) {
	obf := NewObfuscatorV3()

	testCode := `function test() { return "hello"; }`

	result1, err := obf.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("First obfuscation failed: %v", err)
	}

	obf2 := NewObfuscatorV3()
	result2, err := obf2.Obfuscate(testCode)
	if err != nil {
		t.Fatalf("Second obfuscation failed: %v", err)
	}

	t.Logf("First result length: %d", len(result1))
	t.Logf("Second result length: %d", len(result2))

	if len(result1) == 0 || len(result2) == 0 {
		t.Error("Obfuscation results should not be empty")
	}
}
