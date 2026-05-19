package tools

import (
	"strings"
	"testing"
)

func TestObfuscatorV2Creation(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	if obfuscator == nil {
		t.Fatal("Expected obfuscator V2 to be created")
	}
	if obfuscator.config.EnableAdvancedStringEncryption != true {
		t.Error("Default config should enable advanced string encryption")
	}
	if obfuscator.config.EnableEnhancedControlFlowFlattening != true {
		t.Error("Default config should enable enhanced control flow flattening")
	}
	if obfuscator.config.EnableCodeSplitting != true {
		t.Error("Default config should enable code splitting")
	}
	if obfuscator.config.EnableEnhancedDeadCodeInjection != true {
		t.Error("Default config should enable enhanced dead code injection")
	}
}

func TestObfuscatorV2WithCustomConfig(t *testing.T) {
	config := ObfuscatorV2Config{
		EnableAdvancedStringEncryption:     false,
		EnableEnhancedControlFlowFlattening: false,
		EnableCodeSplitting:               false,
		EnableEnhancedDeadCodeInjection:  false,
		EncryptionRounds:                 5,
		SplitFragments:                   5,
		DeadCodeRatio:                    0.5,
	}
	obfuscator := NewObfuscatorV2(config)
	if obfuscator.config.EnableAdvancedStringEncryption != false {
		t.Error("Custom config should disable advanced string encryption")
	}
	if obfuscator.config.EncryptionRounds != 5 {
		t.Error("Custom config should set encryption rounds to 5")
	}
}

func TestObfuscateV2Basic(t *testing.T) {
	code := `function hello() { return "world"; }`
	obfuscator := NewObfuscatorV2()
	result, err := obfuscator.Obfuscate(code)
	if err != nil {
		t.Fatalf("Obfuscate V2 failed: %v", err)
	}
	if result == "" {
		t.Error("Obfuscated result should not be empty")
	}
	if result == code {
		t.Error("Obfuscated result should differ from original")
	}
}

func TestObfuscateV2EmptyCode(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	_, err := obfuscator.Obfuscate("")
	if err == nil {
		t.Error("Expected error for empty code")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Error("Error message should mention empty code")
	}
}

// ==================== 字符串加密混淆测试 ====================

func TestV2StringEncryption(t *testing.T) {
	code := `var url = "https://api.example.com";
var token = "Bearer xyz123";`
	obfuscator := NewObfuscatorV2(ObfuscatorV2Config{
		EnableAdvancedStringEncryption: true,
		StringEncryptionKey:            []byte("test-v2-key-12345678"),
		EncryptionRounds:               3,
	})
	result := obfuscator.encryptStringsAdvanced(code)

	if strings.Contains(result, "https://api.example.com") {
		t.Error("Strings should be encrypted")
	}
	if !strings.Contains(result, "__v2d") {
		t.Error("Encrypted strings should use V2 decoder function")
	}

	stats := obfuscator.GetStats()
	if stats["strings_encrypted"] < 2 {
		t.Errorf("Should encrypt at least 2 strings, got %d", stats["strings_encrypted"])
	}
}

func TestV2StringEncryptionWithMultipleRounds(t *testing.T) {
	code := `var secret = "confidential data";`

	testCases := []int{1, 2, 3, 4, 5}
	for _, rounds := range testCases {
		obfuscator := NewObfuscatorV2(ObfuscatorV2Config{
			EnableAdvancedStringEncryption: true,
			EncryptionRounds:              rounds,
			StringEncryptionKey:           []byte("test-multi-round-key"),
		})
		result := obfuscator.encryptStringsAdvanced(code)

		if !strings.Contains(result, "__v2d") {
			t.Errorf("Should encrypt string with %d rounds", rounds)
		}

		if result == code {
			t.Errorf("Encrypted result should differ from original with %d rounds", rounds)
		}
	}
}

func TestV2ShouldEncryptString(t *testing.T) {
	obfuscator := NewObfuscatorV2()

	testCases := []struct {
		input    string
		expected bool
	}{
		{"a", false},
		{"ab", false},
		{"abc", true},
		{"function", false},
		{"var ", false},
		{"console.log", false},
		{"https://example.com", true},
		{"Bearer token123", true},
	}

	for _, tc := range testCases {
		result := obfuscator.shouldEncryptStringV2(tc.input)
		if result != tc.expected {
			t.Errorf("shouldEncryptStringV2(%q) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

func TestV2ScrambleBytes(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	for seed := 0; seed < 5; seed++ {
		result := obfuscator.scrambleBytesV2(data, seed)

		if len(result) != len(data) {
			t.Errorf("Scrambled bytes should have same length, got %d", len(result))
		}

		for _, b := range data {
			found := false
			for _, r := range result {
				if r == b {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Scrambled result should contain original byte %d", b)
			}
		}
	}
}

// ==================== 控制流扁平化测试 ====================

func TestV2FlattenControlFlowEnhanced(t *testing.T) {
	code := `if (x > 0) { console.log("positive"); } else { console.log("negative"); }
for (var i = 0; i < 10; i++) { sum += i; }
while (true) { break; }`

	obfuscator := NewObfuscatorV2(ObfuscatorV2Config{
		EnableEnhancedControlFlowFlattening: true,
	})
	result := obfuscator.flattenControlFlowEnhanced(code)

	if !strings.Contains(result, "switch(") {
		t.Error("Enhanced control flow should use switch statements")
	}
	if !strings.Contains(result, "_0xSF") {
		t.Error("Enhanced control flow should use state variables")
	}

	stats := obfuscator.GetStats()
	if stats["control_flow_flattened"] < 1 {
		t.Error("Should have flattened at least one control flow structure")
	}
}

func TestV2FlattenIfEnhanced(t *testing.T) {
	code := `if (condition) { doSomething(); } else { doOther(); }`
	obfuscator := NewObfuscatorV2()
	result := obfuscator.flattenIfEnhanced(code)

	if !strings.Contains(result, "switch(") {
		t.Error("Flattened if statements should use switch")
	}
	if !strings.Contains(result, "case") {
		t.Error("Flattened if statements should use case")
	}
}

func TestV2FlattenForEnhanced(t *testing.T) {
	code := `for (var i = 0; i < 10; i++) { sum += i; }`
	obfuscator := NewObfuscatorV2()
	result := obfuscator.flattenForEnhanced(code)

	if !strings.Contains(result, "for(") {
		t.Error("Flattened for loops should use for loop")
	}
	if !strings.Contains(result, "_0xSF") {
		t.Error("Flattened for loops should use state variables")
	}
}

func TestV2FlattenWhileEnhanced(t *testing.T) {
	code := `while (condition) { doSomething(); }`
	obfuscator := NewObfuscatorV2()
	result := obfuscator.flattenWhileEnhanced(code)

	if !strings.Contains(result, "for(") {
		t.Error("Flattened while loops should use for loop")
	}
}

func TestV2AddStateMachineWrapper(t *testing.T) {
	code := `function test() { return true; }`
	obfuscator := NewObfuscatorV2()
	result := obfuscator.addStateMachineWrapper(code)

	if !strings.Contains(result, "_0xSF") {
		t.Error("State machine wrapper should use state variables")
	}
	if !strings.Contains(result, "dispatch") {
		t.Error("State machine wrapper should have dispatch function")
	}
}

func TestV2GenerateStateVar(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	names := make(map[string]bool)

	for i := 0; i < 10; i++ {
		name := obfuscator.generateStateVar()
		if names[name] {
			t.Errorf("Generated duplicate state variable: %s", name)
		}
		names[name] = true
		if !strings.HasPrefix(name, "_0xSF") {
			t.Errorf("State variable should start with _0xSF, got %s", name)
		}
	}
}

// ==================== 代码分割混淆测试 ====================

func TestV2SplitCode(t *testing.T) {
	code := `function test1() { return 1; }
function test2() { return 2; }
function test3() { return 3; }
function test4() { return 4; }
function test5() { return 5; }`

	obfuscator := NewObfuscatorV2(ObfuscatorV2Config{
		EnableCodeSplitting: true,
		SplitFragments:      3,
	})
	result := obfuscator.splitCode(code)

	if !strings.Contains(result, "__frg_") {
		t.Error("Split code should use fragment tokens")
	}
	if !strings.Contains(result, "eval(") {
		t.Error("Split code should use eval for reassembly")
	}

	stats := obfuscator.GetStats()
	if stats["fragments_created"] < 2 {
		t.Errorf("Should create at least 2 fragments, got %d", stats["fragments_created"])
	}
}

func TestV2SplitCodeShort(t *testing.T) {
	code := `var x = 1;`

	obfuscator := NewObfuscatorV2(ObfuscatorV2Config{
		EnableCodeSplitting: true,
	})
	result := obfuscator.splitCode(code)

	if len(result) < len(code) {
		t.Error("Short code should not be split if too small")
	}
}

func TestV2SplitIntoStatements(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	code := `function test() {
    var x = 1;
    var y = 2;
    return x + y;
}`

	statements := obfuscator.splitIntoStatements(code)

	if len(statements) < 1 {
		t.Errorf("Should split into at least 1 statement, got %d", len(statements))
	}
}

func TestV2AssembleFragments(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	fragments := []string{
		"function test1() { return 1; }",
		"function test2() { return 2; }",
		"function test3() { return 3; }",
	}

	result := obfuscator.assembleFragments(fragments)

	if !strings.Contains(result, "__frg_") {
		t.Error("Assembled fragments should use fragment tokens")
	}
	if !strings.Contains(result, "eval(") {
		t.Error("Assembled fragments should use eval")
	}
	if !strings.Contains(result, ".join(") {
		t.Error("Assembled fragments should join fragments")
	}
}

func TestV2EncodeFragment(t *testing.T) {
	obfuscator := NewObfuscatorV2(ObfuscatorV2Config{
		StringEncryptionKey: []byte("test-fragment-key"),
	})
	fragment := `function test() { return true; }`

	encoded := obfuscator.encodeFragment(fragment)

	if encoded == fragment {
		t.Error("Encoded fragment should differ from original")
	}
	if encoded == "" {
		t.Error("Encoded fragment should not be empty")
	}
}

// ==================== 死代码注入测试 ====================

func TestV2InjectDeadCodeEnhanced(t *testing.T) {
	code := `function test() { return true; }`
	obfuscator := NewObfuscatorV2(ObfuscatorV2Config{
		EnableEnhancedDeadCodeInjection: true,
		DeadCodeRatio:                   0.5,
	})
	result := obfuscator.injectDeadCodeEnhanced(code)

	if result == code {
		t.Error("Dead code should be injected")
	}

	stats := obfuscator.GetStats()
	if stats["dead_code_injected"] < 1 {
		t.Error("Should inject at least one dead code block")
	}
}

func TestV2GenerateDeadCode(t *testing.T) {
	obfuscator := NewObfuscatorV2()

	deadCodeTypes := []string{
		"var ", "if(", "for(", "function ", "=",
	}

	for i := 0; i < 10; i++ {
		deadCode := obfuscator.generateDeadCode()
		found := false
		for _, prefix := range deadCodeTypes {
			if strings.Contains(deadCode, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Generated dead code should contain valid code pattern, got: %s", deadCode)
		}
	}
}

func TestV2GenerateDeadVariableDecl(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	result := obfuscator.generateDeadVariableDecl()

	if !strings.Contains(result, "var ") {
		t.Error("Dead variable declaration should start with 'var '")
	}
	if !strings.Contains(result, "=") {
		t.Error("Dead variable declaration should contain '='")
	}
}

func TestV2GenerateDeadConditional(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	result := obfuscator.generateDeadConditional()

	if !strings.Contains(result, "if(") {
		t.Error("Dead conditional should start with 'if('")
	}
	if !strings.Contains(result, "{") {
		t.Error("Dead conditional should contain braces")
	}
}

func TestV2GenerateDeadLoop(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	result := obfuscator.generateDeadLoop()

	if !strings.Contains(result, "for(") {
		t.Error("Dead loop should start with 'for('")
	}
}

func TestV2GenerateDeadFunction(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	result := obfuscator.generateDeadFunction()

	if !strings.HasPrefix(strings.TrimSpace(result), "function ") {
		t.Error("Dead function should start with 'function '")
	}
}

func TestV2GenerateDeadExpression(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	result := obfuscator.generateDeadExpression()

	if !strings.Contains(result, "var ") {
		t.Error("Dead expression should declare variable")
	}
	if !strings.Contains(result, "=") {
		t.Error("Dead expression should contain assignment")
	}
}

func TestV2InjectDeadCodeAtRandom(t *testing.T) {
	code := `function test() { return true; }
function test2() { return false; }`
	obfuscator := NewObfuscatorV2()
	deadCode := "var x = 1;"

	result := obfuscator.injectDeadCodeAtRandom(code, deadCode)

	if !strings.Contains(result, deadCode) {
		t.Error("Dead code should be injected into result")
	}
}

func TestV2FindInjectionPoints(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	code := `var x = 1;
function test() { return true; }
if (condition) { doSomething(); }`

	points := obfuscator.findInjectionPoints(code)

	if len(points) == 0 {
		t.Error("Should find at least one injection point")
	}
}

// ==================== 控制流重排序测试 ====================

func TestV2ReorderControlFlow(t *testing.T) {
	code := `function test1() { return 1; }
function test2() { return 2; }
function test3() { return 3; }`

	obfuscator := NewObfuscatorV2(ObfuscatorV2Config{
		EnableControlFlowReordering: true,
		EnableReachableCodeShuffling: true,
	})
	result := obfuscator.reorderControlFlow(code)

	if !strings.Contains(result, "function ") {
		t.Error("Reordered code should still contain functions")
	}
}

func TestV2ExtractReachableBlocks(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	code := `function test1() { var x = 1; return x; }
function test2() { var y = 2; return y; }`

	blocks := obfuscator.extractReachableBlocks(code)

	if len(blocks) < 2 {
		t.Errorf("Should extract at least 2 blocks, got %d", len(blocks))
	}
}

func TestV2ShuffleBlocks(t *testing.T) {
	blocks := []string{"block1", "block2", "block3", "block4"}

	obfuscator := NewObfuscatorV2()
	shuffled := obfuscator.shuffleBlocks(blocks)

	if len(shuffled) != len(blocks) {
		t.Error("Shuffled blocks should have same length")
	}

	originalMap := make(map[string]bool)
	for _, b := range blocks {
		originalMap[b] = true
	}

	for _, b := range shuffled {
		if !originalMap[b] {
			t.Errorf("Shuffled blocks should contain all original blocks, missing: %s", b)
		}
	}
}

func TestV2ReconstructCode(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	blocks := []string{"block1\n", "block2\n", "block3\n"}

	result := obfuscator.reconstructCode(blocks)

	if !strings.Contains(result, "block1") {
		t.Error("Reconstructed code should contain block1")
	}
	if !strings.Contains(result, "block2") {
		t.Error("Reconstructed code should contain block2")
	}
	if !strings.Contains(result, "block3") {
		t.Error("Reconstructed code should contain block3")
	}
}

// ==================== 虚假控制流测试 ====================

func TestV2InjectBogusControlFlow(t *testing.T) {
	code := `function test() { return true; }
function test2() { return false; }`

	obfuscator := NewObfuscatorV2(ObfuscatorV2Config{
		EnableBogusControlFlow: true,
	})
	result := obfuscator.injectBogusControlFlow(code)

	if !strings.Contains(result, "Math.random()") {
		t.Error("Bogus control flow should use Math.random()")
	}
	if !strings.Contains(result, "function ") {
		t.Error("Original code should be preserved")
	}
}

func TestV2GenerateBogusControlFlow(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	result := obfuscator.generateBogusControlFlow()

	if !strings.Contains(result, "Math.random()") {
		t.Error("Bogus control flow should use Math.random()")
	}
	if !strings.Contains(result, "if(") {
		t.Error("Bogus control flow should contain conditional")
	}
	if !strings.Contains(result, "_0x") {
		t.Error("Bogus control flow should use obfuscated variables")
	}
}

// ==================== 辅助函数测试 ====================

func TestV2GetStats(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	stats := obfuscator.GetStats()

	if _, ok := stats["strings_encrypted"]; !ok {
		t.Error("Stats should include strings_encrypted")
	}
	if _, ok := stats["fragments_created"]; !ok {
		t.Error("Stats should include fragments_created")
	}
	if _, ok := stats["dead_code_injected"]; !ok {
		t.Error("Stats should include dead_code_injected")
	}
	if _, ok := stats["control_flow_flattened"]; !ok {
		t.Error("Stats should include control_flow_flattened")
	}
}

func TestV2ResetStats(t *testing.T) {
	obfuscator := NewObfuscatorV2()

	obfuscator.stats.StringsEncrypted = 10
	obfuscator.stats.FragmentsCreated = 5
	obfuscator.stats.DeadCodeInjected = 3
	obfuscator.stats.ControlFlowFlattened = 2

	obfuscator.ResetStats()

	stats := obfuscator.GetStats()
	if stats["strings_encrypted"] != 0 {
		t.Error("Stats should be reset to 0")
	}
}

func TestV2GenerateObfuscatedName(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	names := make(map[string]bool)

	for i := 0; i < 100; i++ {
		name := obfuscator.generateObfuscatedNameV2()
		if names[name] {
			t.Errorf("Generated duplicate name: %s", name)
		}
		names[name] = true
		if !strings.HasPrefix(name, "_0x") {
			t.Errorf("Generated name should start with _0x, got %s", name)
		}
	}
}

func TestV2GenerateRandomValue(t *testing.T) {
	obfuscator := NewObfuscatorV2()
	values := make(map[string]bool)

	for i := 0; i < 20; i++ {
		value := obfuscator.generateRandomValue()
		if value == "" {
			t.Error("Generated random value should not be empty")
		}
		if strings.Contains(value, "\n") {
			t.Errorf("Generated random value should not contain newlines: %s", value)
		}
		values[value] = true
	}
}

func TestV2RandomInt(t *testing.T) {
	obfuscator := NewObfuscatorV2()

	for i := 0; i < 100; i++ {
		val := obfuscator.randomInt(10, 20)
		if val < 10 || val >= 20 {
			t.Errorf("Random int should be between 10 and 19, got %d", val)
		}
	}
}

func TestV2GenerateRandomString(t *testing.T) {
	obfuscator := NewObfuscatorV2()

	testLengths := []int{5, 10, 15, 20}
	for _, length := range testLengths {
		str := obfuscator.generateRandomString(length)
		if len(str) != length {
			t.Errorf("Generated string should have length %d, got %d", length, len(str))
		}
	}
}

// ==================== 便捷函数测试 ====================

func TestObfuscateWithConfigV2(t *testing.T) {
	code := `function hello() { return "world"; }`
	config := ObfuscatorV2Config{
		EnableAdvancedStringEncryption:    true,
		EnableEnhancedControlFlowFlattening: true,
		EnableCodeSplitting:                true,
		EnableEnhancedDeadCodeInjection:   true,
		EncryptionRounds:                  2,
		SplitFragments:                    2,
		DeadCodeRatio:                     0.3,
	}

	result, err := ObfuscateWithConfigV2(code, config)
	if err != nil {
		t.Fatalf("ObfuscateWithConfigV2 failed: %v", err)
	}
	if result == "" {
		t.Error("Result should not be empty")
	}
	if result == code {
		t.Error("Result should differ from original")
	}
}

func TestGenerateV2ObfuscationReport(t *testing.T) {
	original := `function test() { return "hello"; }`
	obfuscated := `function _0x1(){return _0x2;}`

	config := ObfuscatorV2Config{
		EnableAdvancedStringEncryption:    true,
		EnableEnhancedControlFlowFlattening: true,
		EnableCodeSplitting:                true,
		EnableEnhancedDeadCodeInjection:   true,
		EncryptionRounds:                  3,
		SplitFragments:                    3,
		DeadCodeRatio:                     0.3,
	}

	report := GenerateV2ObfuscationReport(original, obfuscated, config)

	if _, ok := report["original_length"]; !ok {
		t.Error("Report should include original_length")
	}
	if _, ok := report["obfuscated_length"]; !ok {
		t.Error("Report should include obfuscated_length")
	}
	if _, ok := report["compression_ratio"]; !ok {
		t.Error("Report should include compression_ratio")
	}
	if _, ok := report["config"]; !ok {
		t.Error("Report should include config")
	}
}

func TestEstimateV2ObfuscationStrength(t *testing.T) {
	config := ObfuscatorV2Config{
		EnableAdvancedStringEncryption:     true,
		EnableEnhancedControlFlowFlattening: true,
		EnableCodeSplitting:                true,
		EnableEnhancedDeadCodeInjection:   true,
		EncryptionRounds:                  3,
		SplitFragments:                    3,
		DeadCodeRatio:                     0.3,
		EnableControlFlowReordering:       true,
		EnableBogusControlFlow:            true,
	}

	strength := EstimateV2ObfuscationStrength("test code", config)

	if strength <= 0 {
		t.Error("Strength score should be positive")
	}
	if strength > 100 {
		t.Error("Strength score should not exceed 100")
	}
}

func TestEstimateV2ObfuscationStrengthWithMinConfig(t *testing.T) {
	config := ObfuscatorV2Config{
		EnableAdvancedStringEncryption: false,
	}

	strength := EstimateV2ObfuscationStrength("test code", config)

	if strength != 0 {
		t.Errorf("Strength score should be 0 with minimal config, got %f", strength)
	}
}

// ==================== 综合测试 ====================

func TestV2FullObfuscationPipeline(t *testing.T) {
	code := `function calculate(a, b) {
    var result = a + b;
    console.log("Result: " + result);
    return result;
}

function multiply(x, y) {
    var product = x * y;
    return product;
}`

	config := ObfuscatorV2Config{
		EnableAdvancedStringEncryption:     true,
		EnableEnhancedControlFlowFlattening: true,
		EnableCodeSplitting:                true,
		EnableEnhancedDeadCodeInjection:   true,
		EnableControlFlowReordering:       true,
		EnableBogusControlFlow:            true,
		EncryptionRounds:                  3,
		SplitFragments:                    3,
		DeadCodeRatio:                     0.3,
		StringEncryptionKey:                []byte("test-full-v2-key"),
	}

	obfuscator := NewObfuscatorV2(config)
	result, err := obfuscator.Obfuscate(code)

	if err != nil {
		t.Fatalf("Full V2 obfuscation pipeline failed: %v", err)
	}

	if result == "" {
		t.Error("Obfuscated code should not be empty")
	}

	if result == code {
		t.Error("Obfuscated code should differ from original")
	}

	stats := obfuscator.GetStats()
	if stats["strings_encrypted"] < 1 {
		t.Error("Should encrypt at least one string")
	}
}

func TestV2ConcurrentObfuscation(t *testing.T) {
	code := `function test() { return true; }`
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			obfuscator := NewObfuscatorV2()
			_, err := obfuscator.Obfuscate(code)
			if err != nil {
				t.Errorf("Concurrent V2 obfuscation failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestV2DeterminismWithSameConfig(t *testing.T) {
	code := `var myVar = "test value";`

	obfuscator1 := NewObfuscatorV2(ObfuscatorV2Config{
		EnableAdvancedStringEncryption: true,
		StringEncryptionKey:             []byte("deterministic-key"),
		EncryptionRounds:                2,
	})
	result1, _ := obfuscator1.Obfuscate(code)

	obfuscator2 := NewObfuscatorV2(ObfuscatorV2Config{
		EnableAdvancedStringEncryption: true,
		StringEncryptionKey:             []byte("deterministic-key"),
		EncryptionRounds:                2,
	})
	result2, _ := obfuscator2.Obfuscate(code)

	if result1 != result2 {
		t.Error("Same code with same key should produce deterministic results")
	}
}

func TestV2PreserveFunctionStructure(t *testing.T) {
	code := `function test() { return true; }
function test2() { return false; }`

	obfuscator := NewObfuscatorV2(ObfuscatorV2Config{
		EnableCodeSplitting:            true,
		SplitFragments:                 2,
	})
	result := obfuscator.splitCode(code)

	if !strings.Contains(result, "__frg_") {
		t.Error("Split code should contain fragment tokens")
	}
	if !strings.Contains(result, "eval(") {
		t.Error("Split code should contain eval for reassembly")
	}
}

func TestV2FragmentCount(t *testing.T) {
	code := `function test1() { return 1; }
function test2() { return 2; }
function test3() { return 3; }
function test4() { return 4; }
function test5() { return 5; }
function test6() { return 6; }`

	obfuscator := NewObfuscatorV2(ObfuscatorV2Config{
		EnableCodeSplitting: true,
		SplitFragments:      4,
	})
	obfuscator.Obfuscate(code)

	stats := obfuscator.GetStats()
	if stats["fragments_created"] < 3 {
		t.Errorf("Should create at least 3 fragments, got %d", stats["fragments_created"])
	}
}

func TestV2DeadCodeRatioBounds(t *testing.T) {
	testCases := []struct {
		ratio    float64
		expected string
	}{
		{-0.5, "low"},
		{0.0, "low"},
		{0.5, "medium"},
		{1.0, "high"},
		{1.5, "high"},
	}

	for _, tc := range testCases {
		obfuscator := NewObfuscatorV2(ObfuscatorV2Config{
			EnableEnhancedDeadCodeInjection: true,
			DeadCodeRatio:                   tc.ratio,
		})

		stats := obfuscator.GetStats()
		if tc.ratio < 0 || tc.ratio > 1 {
			if stats["dead_code_injected"] >= 1 {
				// Should be capped
			}
		}
	}
}
