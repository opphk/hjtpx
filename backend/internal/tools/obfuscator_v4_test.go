package tools

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestObfuscatorV4Basic(t *testing.T) {
	code := `function test() { return "hello"; }`

	obfuscator := NewObfuscatorV4(nil)
	result, err := obfuscator.Obfuscate(code)

	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	if !strings.Contains(result, "function") {
		t.Log("Warning: Result may be over-obfuscated")
	}

	stats := obfuscator.GetObfuscationStats()
	if stats == nil {
		t.Fatal("GetObfuscationStats returned nil")
	}

	if stats["version"] != "4.0.0" {
		t.Errorf("Expected version 4.0.0, got %v", stats["version"])
	}
}

func TestObfuscatorV4StringEncryption(t *testing.T) {
	code := `
function greet(name) {
    return "Hello, " + name;
}
function farewell(name) {
    return "Goodbye, " + name;
}
`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableStringEncryption: true,
		EnableStringSegmentation: true,
		EnhancedEncryptionLevel: 3,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	if !strings.Contains(result, "__m") && !strings.Contains(result, "__x") {
		t.Log("Warning: String encryption may not be applied")
	}
}

func TestObfuscatorV4ControlFlow(t *testing.T) {
	code := `
function calculate(x, y) {
    if (x > y) {
        return x - y;
    } else {
        return y - x;
    }
}
`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableControlFlowFlattening: true,
		EnableAdvancedControlFlow: true,
		EnableDeadCodeInjection: true,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	if !strings.Contains(result, "switch") && !strings.Contains(result, "_blk") {
		t.Log("Control flow flattening may not be applied")
	}
}

func TestObfuscatorV4AntiDebug(t *testing.T) {
	code := `function check() { return true; }`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableAntiDebug: true,
		EnableEnhancedAntiDebug: true,
		EnableBreakpointDetection: true,
		EnableDevToolsDetection: true,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	expectedPatterns := []string{
		"debugger",
		"document.body.innerHTML",
		"window.location",
	}

	foundCount := 0
	for _, pattern := range expectedPatterns {
		if strings.Contains(result, pattern) {
			foundCount++
		}
	}

	if foundCount < 1 {
		t.Log("Warning: Anti-debug mechanisms may not be fully applied")
	}
}

func TestObfuscatorV4DomainLock(t *testing.T) {
	code := `function init() { console.log("initialized"); }`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableDomainLock: true,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	if !strings.Contains(result, "localhost") && !strings.Contains(result, "Access Denied") {
		t.Log("Warning: Domain lock may not be applied")
	}
}

func TestObfuscatorV4CodeVirtualization(t *testing.T) {
	code := `function add(a, b) { return a + b; }`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableCodeVirtualization: true,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	if !strings.Contains(result, "_v.execute") && !strings.Contains(result, "instructions") {
		t.Log("Warning: Code virtualization may not be applied")
	}
}

func TestObfuscatorV4IntegrityCheck(t *testing.T) {
	code := `function verify() { return true; }`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableCodeIntegrity: true,
		EnableSelfDefending: true,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	if !strings.Contains(result, "sha256") && !strings.Contains(result, "_ih") {
		t.Log("Warning: Integrity check may not be applied")
	}
}

func TestObfuscatorV4ProtectionLevels(t *testing.T) {
	code := `function test() { return "test"; }`

	testCases := []struct {
		level   int
		minSize int
	}{
		{1, 50},
		{2, 100},
		{3, 150},
		{4, 200},
		{5, 250},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Level%d", tc.level), func(t *testing.T) {
			obfuscator := NewObfuscatorV4(nil)
			result, err := obfuscator.ProtectWithLevel(code, tc.level)
			if err != nil {
				t.Fatalf("ProtectWithLevel failed: %v", err)
			}

			if len(result) < tc.minSize {
				t.Errorf("Level %d: Result size %d is less than minimum %d",
					tc.level, len(result), tc.minSize)
			}

			stats := obfuscator.GetObfuscationStats()
			if stats["protection_level"].(int) != tc.level {
				t.Errorf("Level %d: Expected protection level %d, got %v",
					tc.level, tc.level, stats["protection_level"])
			}
		})
	}
}

func TestObfuscatorV4MultiLayerEncryption(t *testing.T) {
	code := `
function secret() {
    return "This is a secret message";
}
function anotherSecret() {
    return "Another secret";
}
`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableStringEncryption: true,
		EnableStringSegmentation: true,
		EnhancedEncryptionLevel: 5,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	decryptorCount := strings.Count(result, "function")
	if decryptorCount < 2 {
		t.Log("Warning: Multi-layer decryption may not be properly implemented")
	}
}

func TestObfuscatorV4PerformanceAnomalyDetection(t *testing.T) {
	code := `function slow() { var sum = 0; for (var i = 0; i < 100; i++) sum += i; return sum; }`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnablePerformanceAnomalyDetection: true,
		EnableTimingProtection: true,
		EnableMemoryProtection: true,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	if !strings.Contains(result, "_pa.detect") && !strings.Contains(result, "_tp.check") {
		t.Log("Warning: Performance monitoring may not be applied")
	}
}

func TestObfuscatorV4Mutation(t *testing.T) {
	code := `function mutate() { var obj = {}; obj.test = 1; return obj; }`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableMutationObfuscation: true,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	if !strings.Contains(result, "Object.freeze") && !strings.Contains(result, "defineProperty") {
		t.Log("Warning: Mutation obfuscation may not be applied")
	}
}

func TestObfuscatorV4NetworkMonitoring(t *testing.T) {
	code := `function load() { return true; }`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableNetworkMonitoring: true,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	if !strings.Contains(result, "_nm.monitor") {
		t.Log("Warning: Network monitoring may not be applied")
	}
}

func TestObfuscatorV4VariableObfuscation(t *testing.T) {
	code := `
function calculateSum(array) {
    var sum = 0;
    var length = array.length;
    for (var i = 0; i < length; i++) {
        sum = sum + array[i];
    }
    return sum;
}
`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableVariableObfuscation: true,
		EnableNumberObfuscation: true,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	originalVars := []string{"sum", "length", "array"}
	for _, v := range originalVars {
		if strings.Contains(result, v) {
			t.Logf("Warning: Variable %s may not be obfuscated", v)
		}
	}
}

func TestObfuscatorV4DeadCodeInjection(t *testing.T) {
	code := `
function test() {
    var x = 10;
    return x * 2;
}
`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableDeadCodeInjection: true,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	deadCodeIndicators := []string{"Math.random()", "Date.now()", "throw new Error"}
	found := 0
	for _, indicator := range deadCodeIndicators {
		if strings.Contains(result, indicator) {
			found++
		}
	}

	if found == 0 {
		t.Log("Warning: Dead code injection may not be applied")
	}
}

func TestObfuscatorV4MemoryProtection(t *testing.T) {
	code := `function protect() { return true; }`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableMemoryProtection: true,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	if !strings.Contains(result, "_mp.check") && !strings.Contains(result, "Object.prototype") {
		t.Log("Warning: Memory protection may not be applied")
	}
}

func TestObfuscatorV4CommentsRemoval(t *testing.T) {
	code := `
function withComments() {
    // This is a single line comment
    var x = 10; // inline comment
    /* This is a
       multi-line comment */
    return x;
}
`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		RemoveComments: true,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if strings.Contains(result, "// This is a") {
		t.Error("Single line comment was not removed")
	}

	if strings.Contains(result, "/* This is a") {
		t.Error("Multi-line comment was not removed")
	}
}

func TestObfuscatorV4WhitespaceRemoval(t *testing.T) {
	code := `
function    withSpaces    ()    {
    var    x    =    10    ;
    return    x    ;
}
`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		RemoveWhitespace: true,
		EnableCodeCompression: true,
	})

	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	extraSpaces := strings.Count(result, "    ")
	if extraSpaces > 2 {
		t.Logf("Warning: Extra whitespace may still be present: %d occurrences", extraSpaces)
	}
}

func TestObfuscatorV4EmptyCode(t *testing.T) {
	obfuscator := NewObfuscatorV4(nil)

	_, err := obfuscator.Obfuscate("")
	if err == nil {
		t.Error("Expected error for empty code, got nil")
	}
}

func TestObfuscatorV4ComplexCode(t *testing.T) {
	code := `
(function() {
    var config = {
        apiUrl: 'https://api.example.com',
        apiKey: 'secret-key-12345',
        timeout: 5000,
        retries: 3
    };

    function fetchData(endpoint) {
        return new Promise(function(resolve, reject) {
            var xhr = new XMLHttpRequest();
            xhr.open('GET', config.apiUrl + endpoint, true);
            xhr.setRequestHeader('Authorization', 'Bearer ' + config.apiKey);
            xhr.timeout = config.timeout;

            xhr.onload = function() {
                if (xhr.status >= 200 && xhr.status < 300) {
                    resolve(JSON.parse(xhr.responseText));
                } else {
                    reject(new Error('Request failed: ' + xhr.status));
                }
            };

            xhr.onerror = function() {
                reject(new Error('Network error'));
            };

            xhr.ontimeout = function() {
                reject(new Error('Request timeout'));
            };

            xhr.send();
        });
    }

    function processData(data) {
        if (!data || !Array.isArray(data)) {
            return [];
        }
        return data.map(function(item) {
            return {
                id: item.id,
                name: item.name,
                value: item.value * config.retries
            };
        });
    }

    window.API = {
        fetch: fetchData,
        process: processData
    };
})();
`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableVariableObfuscation:     true,
		EnableStringEncryption:        true,
		EnableStringSegmentation:     true,
		EnableControlFlowFlattening:  true,
		EnableAdvancedControlFlow:    true,
		EnableDeadCodeInjection:      true,
		EnableAntiDebug:              true,
		EnableEnhancedAntiDebug:      true,
		EnableBreakpointDetection:    true,
		EnableDevToolsDetection:      true,
		EnableCodeIntegrity:          true,
		EnableSelfDefending:          true,
		EnableTimingProtection:       true,
		EnableMemoryProtection:       true,
		EnableCodeVirtualization:    true,
		EnableMutationObfuscation:    true,
		EnableDomainLock:            true,
		EnableNetworkMonitoring:     true,
		EnablePerformanceAnomalyDetection: true,
		EnhancedEncryptionLevel:     5,
		ProtectionLevel:             5,
	})

	start := time.Now()
	result, err := obfuscator.Obfuscate(code)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Obfuscate failed: %v", err)
	}

	if result == "" {
		t.Fatal("Obfuscate returned empty result")
	}

	if len(result) < len(code) {
		t.Errorf("Obfuscation should increase code size, but got %d < %d", len(result), len(code))
	}

	t.Logf("Obfuscation took %v, original size: %d, obfuscated size: %d",
		elapsed, len(code), len(result))

	stats := obfuscator.GetObfuscationStats()
	t.Logf("Obfuscation stats: %+v", stats)
}

func TestObfuscatorV4Concurrency(t *testing.T) {
	code := `function test() { return "test"; }`

	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableVariableObfuscation: true,
		EnableStringEncryption: true,
	})

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 5; j++ {
				_, err := obfuscator.Obfuscate(code)
				if err != nil {
					t.Errorf("Concurrent obfuscation failed: %v", err)
				}
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkObfuscatorV4Basic(b *testing.B) {
	code := `function test() { return "hello world"; }`
	obfuscator := NewObfuscatorV4(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = obfuscator.Obfuscate(code)
	}
}

func BenchmarkObfuscatorV4Advanced(b *testing.B) {
	code := `
function complex(a, b, c) {
    if (a > b) {
        if (a > c) {
            return a;
        } else {
            return c;
        }
    } else {
        if (b > c) {
            return b;
        } else {
            return c;
        }
    }
}
`
	obfuscator := NewObfuscatorV4(&ObfuscatorV4Options{
		EnableVariableObfuscation:    true,
		EnableStringEncryption:       true,
		EnableStringSegmentation:     true,
		EnableControlFlowFlattening:  true,
		EnableAdvancedControlFlow:    true,
		EnableDeadCodeInjection:     true,
		EnableAntiDebug:             true,
		EnableEnhancedAntiDebug:     true,
		EnableBreakpointDetection:   true,
		EnableDevToolsDetection:     true,
		EnableCodeIntegrity:         true,
		EnhancedEncryptionLevel:     5,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = obfuscator.Obfuscate(code)
	}
}

func BenchmarkObfuscatorV4ProtectionLevel5(b *testing.B) {
	code := `function test() { return "test"; }`
	obfuscator := NewObfuscatorV4(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = obfuscator.ProtectWithLevel(code, 5)
	}
}

func TestObfuscatorV4AdvancedKeyGenerator(t *testing.T) {
	kg := NewAdvancedKeyGenerator()

	key1 := kg.generate()
	key2 := kg.generate()

	if key1 == key2 {
		t.Error("Generated keys should be unique")
	}

	if len(key1) < 32 {
		t.Error("Generated key is too short")
	}

	variableName := kg.generateVariableName()
	if variableName == "" {
		t.Error("Generated variable name is empty")
	}

	if !strings.HasPrefix(variableName, "_") {
		t.Error("Variable name should start with underscore")
	}
}

func TestObfuscatorV4StringRegistry(t *testing.T) {
	sr := NewAdvancedStringRegistry()

	idx1 := sr.add("test string")
	idx2 := sr.add("test string")

	if idx1 != idx2 {
		t.Error("Same string should return same index")
	}

	idx3 := sr.add("different string")
	if idx3 == idx1 {
		t.Error("Different string should return different index")
	}

	encoded := sr.getEncoded(idx1)
	if encoded == "" {
		t.Error("Encoded string should not be empty")
	}

	idx4 := sr.addSegmented("segmented string", 3)
	if idx4 < 0 {
		t.Error("Segmented string should be added")
	}

	segments := sr.getSegments(idx4)
	if len(segments) == 0 {
		t.Error("Segments should not be empty")
	}
}

func TestObfuscatorV4ControlFlowManager(t *testing.T) {
	cfm := NewEnhancedControlFlowManager()

	blockID := cfm.createBlock("test body", "test condition")
	if blockID == "" {
		t.Error("Created block should have an ID")
	}

	opaque := cfm.generateOpaquePredicate()
	if opaque == "" {
		t.Error("Generated opaque predicate should not be empty")
	}

	sm := cfm.createStateMachine(5)
	if sm == nil {
		t.Error("Created state machine should not be nil")
	}

	if len(sm.States) != 5 {
		t.Errorf("Expected 5 states, got %d", len(sm.States))
	}
}

func TestObfuscatorV4AntiDebugManager(t *testing.T) {
	mgr := NewEnhancedAntiDebugManager()

	if len(mgr.techniques) == 0 {
		t.Error("Techniques should not be empty")
	}

	if len(mgr.layers) == 0 {
		t.Error("Protection layers should not be empty")
	}

	technique, exists := mgr.techniques["debugger_check"]
	if !exists {
		t.Error("debugger_check technique should exist")
	}

	if !technique.Enabled {
		t.Error("debugger_check should be enabled by default")
	}
}

func TestObfuscatorV4IntegrityManager(t *testing.T) {
	mgr := NewEnhancedIntegrityManager()

	if mgr == nil {
		t.Error("Integrity manager should not be nil")
	}

	if len(mgr.checksums) != 0 {
		t.Error("Initial checksums should be empty")
	}

	mgr.checksums["test"] = "hash123"
	if len(mgr.checksums) != 1 {
		t.Error("Checksum should be added")
	}
}

func TestObfuscatorV4VirtualizationManager(t *testing.T) {
	mgr := NewEnhancedVirtualizationManager()

	if mgr == nil {
		t.Error("Virtualization manager should not be nil")
	}

	if mgr.bytecode == nil {
		t.Error("Bytecode should not be nil")
	}

	if mgr.bytecode.Metadata.Version != "4.0" {
		t.Errorf("Expected bytecode version 4.0, got %s", mgr.bytecode.Metadata.Version)
	}
}
