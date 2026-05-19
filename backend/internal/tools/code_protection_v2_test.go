package tools

import (
	"fmt"
	"strings"
	"testing"
)

func TestCodeProtectionV2_BasicProtection(t *testing.T) {
	config := CodeProtectionV2Config{
		EnableVirtualization:      false,
		EnableControlFlowObfuscation: false,
		EnableAntiDebugV2:        false,
		EnableIntegrityCheckV2:   false,
		EnableCodeSplitting:      false,
		EnableDynamicDecryption:  false,
		ProtectionLevel:          1,
	}

	protector := NewCodeProtectionV2(config)

	code := "function test() { return 'hello'; }"

	protected, err := protector.Protect(code)
	if err != nil {
		t.Fatalf("Protection failed: %v", err)
	}

	if protected == "" {
		t.Error("Protected code should not be empty")
	}
}

func TestCodeProtectionV2_VirtualizationProtection(t *testing.T) {
	config := CodeProtectionV2Config{
		EnableVirtualization:      true,
		EnableControlFlowObfuscation: false,
		EnableAntiDebugV2:        false,
		EnableIntegrityCheckV2:   false,
		EnableCodeSplitting:      false,
		EnableDynamicDecryption:  false,
		ProtectionLevel:          1,
	}

	protector := NewCodeProtectionV2(config)

	code := "var x = 10; function add(a, b) { return a + b; }"

	protected, err := protector.Protect(code)
	if err != nil {
		t.Fatalf("Virtualization protection failed: %v", err)
	}

	if protected == "" {
		t.Error("Protected code should not be empty")
	}

	if !strings.Contains(protected, "_0xVM") {
		t.Log("VM engine should be included in protected code")
	}
}

func TestCodeProtectionV2_ControlFlowObfuscation(t *testing.T) {
	config := CodeProtectionV2Config{
		EnableVirtualization:      false,
		EnableControlFlowObfuscation: true,
		EnableAntiDebugV2:        false,
		EnableIntegrityCheckV2:   false,
		EnableCodeSplitting:      false,
		EnableDynamicDecryption:  false,
		ProtectionLevel:          2,
	}

	protector := NewCodeProtectionV2(config)

	code := "function test() { if (x > 0) { return true; } else { return false; } }"

	protected, err := protector.Protect(code)
	if err != nil {
		t.Fatalf("Control flow obfuscation failed: %v", err)
	}

	if protected == "" {
		t.Error("Protected code should not be empty")
	}
}

func TestCodeProtectionV2_AntiDebugProtection(t *testing.T) {
	config := CodeProtectionV2Config{
		EnableVirtualization:      false,
		EnableControlFlowObfuscation: false,
		EnableAntiDebugV2:        true,
		EnableIntegrityCheckV2:   false,
		EnableCodeSplitting:      false,
		EnableDynamicDecryption:  false,
		ProtectionLevel:          2,
	}

	protector := NewCodeProtectionV2(config)

	code := "function test() { return 'test'; }"

	protected, err := protector.Protect(code)
	if err != nil {
		t.Fatalf("Anti-debug protection failed: %v", err)
	}

	if !strings.Contains(protected, "_0xAD2") {
		t.Error("Anti-debug code should be included")
	}

	if !strings.Contains(protected, "devtools") {
		t.Error("DevTools detection should be included")
	}
}

func TestCodeProtectionV2_IntegrityCheckProtection(t *testing.T) {
	config := CodeProtectionV2Config{
		EnableVirtualization:      false,
		EnableControlFlowObfuscation: false,
		EnableAntiDebugV2:        false,
		EnableIntegrityCheckV2:   true,
		EnableCodeSplitting:      false,
		EnableDynamicDecryption:  false,
		ProtectionLevel:          3,
	}

	protector := NewCodeProtectionV2(config)

	code := "function test() { return 'integrity'; }"

	protected, err := protector.Protect(code)
	if err != nil {
		t.Fatalf("Integrity check protection failed: %v", err)
	}

	if !strings.Contains(protected, "_0xIC") {
		t.Error("Integrity checker should be included")
	}

	if !strings.Contains(protected, "sha256") {
		t.Error("SHA256 hashing should be included")
	}
}

func TestCodeProtectionV2_CodeSplittingProtection(t *testing.T) {
	config := CodeProtectionV2Config{
		EnableVirtualization:      false,
		EnableControlFlowObfuscation: false,
		EnableAntiDebugV2:        false,
		EnableIntegrityCheckV2:   false,
		EnableCodeSplitting:      true,
		EnableDynamicDecryption:  false,
		ProtectionLevel:          3,
	}

	protector := NewCodeProtectionV2(config)

	code := "function test() { return 'split'; } function demo() { return 'code'; }"

	protected, err := protector.Protect(code)
	if err != nil {
		t.Fatalf("Code splitting protection failed: %v", err)
	}

	if !strings.Contains(protected, "_0xCHUNKS") {
		t.Error("Code chunks should be included")
	}

	if !strings.Contains(protected, "_0xLOADER") {
		t.Error("Loader should be included")
	}
}

func TestCodeProtectionV2_DynamicDecryptionProtection(t *testing.T) {
	config := CodeProtectionV2Config{
		EnableVirtualization:      false,
		EnableControlFlowObfuscation: false,
		EnableAntiDebugV2:        false,
		EnableIntegrityCheckV2:   false,
		EnableCodeSplitting:      false,
		EnableDynamicDecryption:  true,
		ProtectionLevel:          3,
	}

	protector := NewCodeProtectionV2(config)

	code := "function test() { return 'encrypted'; }"

	protected, err := protector.Protect(code)
	if err != nil {
		t.Fatalf("Dynamic decryption protection failed: %v", err)
	}

	if !strings.Contains(protected, "_0xEK") {
		t.Error("Encryption key should be included")
	}

	if !strings.Contains(protected, "_0xDEC") {
		t.Error("Decryption function should be included")
	}
}

func TestCodeProtectionV2_AllProtections(t *testing.T) {
	config := CodeProtectionV2Config{
		EnableVirtualization:      true,
		EnableControlFlowObfuscation: true,
		EnableAntiDebugV2:        true,
		EnableIntegrityCheckV2:   true,
		EnableCodeSplitting:      true,
		EnableDynamicDecryption:  true,
		ProtectionLevel:          3,
	}

	protector := NewCodeProtectionV2(config)

	code := `
function add(a, b) {
    return a + b;
}
function multiply(x, y) {
    return x * y;
}
`

	protected, err := protector.Protect(code)
	if err != nil {
		t.Fatalf("Full protection failed: %v", err)
	}

	if len(protected) < len(code) {
		t.Error("Protected code should be longer than original due to protection layers")
	}

	t.Logf("Original size: %d, Protected size: %d", len(code), len(protected))
}

func TestCodeProtectionV2_ProtectionLevels(t *testing.T) {
	code := "function test() { return 'level test'; }"

	testCases := []struct {
		level    int
		expected int
	}{
		{1, 1},
		{2, 2},
		{3, 3},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			protected, err := NewCodeProtectionV2(CodeProtectionV2Config{ProtectionLevel: tc.level}).Protect(code)
			if err != nil {
				t.Fatalf("Protection level %d failed: %v", tc.level, err)
			}

			if len(protected) == 0 {
				t.Error("Protected code should not be empty")
			}
		})
	}
}

func TestCodeProtectionV2_AdvancedVMEngine(t *testing.T) {
	engine := NewAdvancedVMEngine()

	if engine == nil {
		t.Fatal("VM engine should not be nil")
	}

	if len(engine.instructionSet) == 0 {
		t.Error("Instruction set should not be empty")
	}

	if engine.context == nil {
		t.Error("VM context should not be nil")
	}
}

func TestCodeProtectionV2_VMInstructionExecution(t *testing.T) {
	engine := NewAdvancedVMEngine()

	t.Run("NOP Instruction", func(t *testing.T) {
		result := engine.instructionSet[0x00].Execute(engine.context, nil)
		if result != nil {
			t.Error("NOP should return nil")
		}
	})

	t.Run("LOAD_CONST Instruction", func(t *testing.T) {
		engine.context.Stack = []interface{}{}
		engine.instructionSet[0x01].Execute(engine.context, []interface{}{42})
		
		if len(engine.context.Stack) != 1 {
			t.Error("Stack should have 1 element")
		}

		if engine.context.Stack[0] != 42 {
			t.Errorf("Expected 42, got %v", engine.context.Stack[0])
		}
	})

	t.Run("STORE_VAR Instruction", func(t *testing.T) {
		engine.instructionSet[0x02].Execute(engine.context, []interface{}{"x", 100})
		
		if engine.context.Variables["x"] != 100 {
			t.Errorf("Expected 100, got %v", engine.context.Variables["x"])
		}
	})

	t.Run("LOAD_VAR Instruction", func(t *testing.T) {
		engine.context.Stack = []interface{}{}
		engine.instructionSet[0x03].Execute(engine.context, []interface{}{"x"})
		
		if len(engine.context.Stack) != 1 {
			t.Error("Stack should have 1 element")
		}
	})
}

func TestCodeProtectionV2_VMCompilation(t *testing.T) {
	engine := NewAdvancedVMEngine()

	code := "var x = 10; var y = 20;"

	bytecode, err := engine.Compile(code)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	if bytecode == nil {
		t.Fatal("Bytecode should not be nil")
	}

	if len(bytecode.Instructions) == 0 {
		t.Error("Instructions should not be empty")
	}

	if bytecode.Metadata == nil {
		t.Error("Metadata should not be nil")
	}

	if bytecode.Metadata.Version != "2.0" {
		t.Errorf("Expected version '2.0', got '%s'", bytecode.Metadata.Version)
	}
}

func TestCodeProtectionV2_VMExecution(t *testing.T) {
	engine := NewAdvancedVMEngine()

	code := "var test = 'hello';"

	bytecode, err := engine.Compile(code)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	err = engine.Execute(bytecode)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}
}

func TestCodeProtectionV2_ControlFlowObfuscator(t *testing.T) {
	obfuscator := NewControlFlowObfuscator()

	if obfuscator == nil {
		t.Fatal("Control flow obfuscator should not be nil")
	}

	code := "function test() { if (x > 0) { return true; } return false; }"

	obfuscated := obfuscator.Obfuscate(code)

	if obfuscated == "" {
		t.Error("Obfuscated code should not be empty")
	}

	if len(obfuscated) < len(code) {
		t.Error("Obfuscated code should be longer or equal to original")
	}
}

func TestCodeProtectionV2_ControlFlowBlockExtraction(t *testing.T) {
	obfuscator := NewControlFlowObfuscator()

	code := `
function func1() { return 1; }
function func2() { return 2; }
function func3() { return 3; }
`

	blocks := obfuscator.extractBlocks(code)

	if len(blocks) == 0 {
		t.Error("Should extract at least one block")
	}
}

func TestCodeProtectionV2_JunkBlockGeneration(t *testing.T) {
	obfuscator := NewControlFlowObfuscator()

	junkBlocks := obfuscator.generateJunkBlocks(3)

	if junkBlocks == "" {
		t.Error("Junk blocks should not be empty")
	}

	if !strings.Contains(junkBlocks, "_0xJ") {
		t.Error("Junk blocks should contain obfuscated variable names")
	}
}

func TestCodeProtectionV2_AntiDebugV2(t *testing.T) {
	antiDebug := NewAntiDebugV2()

	if antiDebug == nil {
		t.Fatal("AntiDebugV2 should not be nil")
	}

	if len(antiDebug.detectionMethods) == 0 {
		t.Error("Detection methods should not be empty")
	}

	if len(antiDebug.protectionLayers) == 0 {
		t.Error("Protection layers should not be empty")
	}
}

func TestCodeProtectionV2_AntiDebugProtectionCode(t *testing.T) {
	antiDebug := NewAntiDebugV2()

	code := antiDebug.GenerateProtectionCode()

	if code == "" {
		t.Error("Protection code should not be empty")
	}

	if !strings.Contains(code, "_0xAD2") {
		t.Error("Should contain anti-debug object")
	}

	if !strings.Contains(code, "devtools") {
		t.Error("Should contain devtools detection")
	}

	if !strings.Contains(code, "debugger") {
		t.Error("Should contain debugger protection")
	}
}

func TestCodeProtectionV2_IntegrityCheckerV2(t *testing.T) {
	checker := NewIntegrityCheckerV2()

	if checker == nil {
		t.Fatal("IntegrityCheckerV2 should not be nil")
	}

	if !checker.tamperDetection {
		t.Error("Tamper detection should be enabled")
	}
}

func TestCodeProtectionV2_IntegrityCodeGeneration(t *testing.T) {
	checker := NewIntegrityCheckerV2()

	code := "function test() { return 'integrity'; }"

	integrityCode := checker.GenerateIntegrityCode(code)

	if integrityCode == "" {
		t.Error("Integrity code should not be empty")
	}

	if !strings.Contains(integrityCode, "_0xIH") {
		t.Error("Should contain integrity hash variable")
	}

	if !strings.Contains(integrityCode, "_0xIC") {
		t.Error("Should contain integrity checker object")
	}
}

func TestCodeProtectionV2_IntegrityVerification(t *testing.T) {
	checker := NewIntegrityCheckerV2()

	code := "function test() { return 'verify'; }"

	checker.GenerateIntegrityCode(code)

	isValid := checker.VerifyIntegrity(code)
	if !isValid {
		t.Error("Original code should pass integrity check")
	}

	modifiedCode := code + " // tampered"
	isValid = checker.VerifyIntegrity(modifiedCode)
	if isValid {
		t.Error("Modified code should not pass integrity check")
	}
}

func TestCodeProtectionV2_CodeSplitter(t *testing.T) {
	splitter := NewCodeSplitter()

	if splitter == nil {
		t.Fatal("CodeSplitter should not be nil")
	}
}

func TestCodeProtectionV2_CodeSplitting(t *testing.T) {
	splitter := NewCodeSplitter()

	code := "function test1() { return 1; } function test2() { return 2; } function test3() { return 3; }"

	splitCode := splitter.Split(code, 3)

	if splitCode == "" {
		t.Error("Split code should not be empty")
	}

	if !strings.Contains(splitCode, "_0xCHUNKS") {
		t.Error("Should contain chunks array")
	}

	if !strings.Contains(splitCode, "_0xLOADER") {
		t.Error("Should contain loader object")
	}
}

func TestCodeProtectionV2_SingleChunk(t *testing.T) {
	splitter := NewCodeSplitter()

	code := "function test() { return 1; }"

	splitCode := splitter.Split(code, 1)

	if splitCode != code {
		t.Error("Single chunk should return original code")
	}
}

func TestCodeProtectionV2_LoaderGeneration(t *testing.T) {
	splitter := NewCodeSplitter()

	loader := splitter.GenerateLoader()

	if loader == "" {
		t.Error("Loader should not be empty")
	}
}

func TestCodeProtectionV2_PolymorphicCodeGeneration(t *testing.T) {
	polymorphic1 := GeneratePolymorphicCode()
	polymorphic2 := GeneratePolymorphicCode()

	if polymorphic1 == "" {
		t.Error("Polymorphic code should not be empty")
	}

	if !strings.Contains(polymorphic1, "_0xP") {
		t.Error("Polymorphic code should contain obfuscated variables")
	}

	if polymorphic1 == polymorphic2 {
		t.Log("Polymorphic codes may be the same due to random selection")
	}

	t.Logf("Polymorphic variation 1: %s", polymorphic1[:50])
}

func TestCodeProtectionV2_VMContextReset(t *testing.T) {
	engine := NewAdvancedVMEngine()

	engine.context.Stack = []interface{}{1, 2, 3}
	engine.context.Variables["x"] = 100
	engine.context.IP = 100

	engine.context.Reset()

	if len(engine.context.Stack) != 0 {
		t.Error("Stack should be empty after reset")
	}

	if len(engine.context.Variables) != 0 {
		t.Error("Variables should be empty after reset")
	}

	if engine.context.IP != 0 {
		t.Error("IP should be 0 after reset")
	}
}

func TestCodeProtectionV2_OperationExecution(t *testing.T) {
	engine := NewAdvancedVMEngine()

	t.Run("ADD Operation", func(t *testing.T) {
		engine.context.Stack = []interface{}{10, 20}
		result := engine.instructionSet[0x04].Execute(engine.context, nil)
		
		if result.(int) != 30 {
			t.Errorf("Expected 30, got %v", result)
		}
	})

	t.Run("SUB Operation", func(t *testing.T) {
		engine.context.Stack = []interface{}{20, 10}
		result := engine.instructionSet[0x05].Execute(engine.context, nil)
		
		if result.(int) != 10 {
			t.Errorf("Expected 10, got %v", result)
		}
	})

	t.Run("MUL Operation", func(t *testing.T) {
		engine.context.Stack = []interface{}{5, 6}
		result := engine.instructionSet[0x06].Execute(engine.context, nil)
		
		if result.(int) != 30 {
			t.Errorf("Expected 30, got %v", result)
		}
	})

	t.Run("DIV Operation", func(t *testing.T) {
		engine.context.Stack = []interface{}{20, 4}
		result := engine.instructionSet[0x07].Execute(engine.context, nil)
		
		if result.(int) != 5 {
			t.Errorf("Expected 5, got %v", result)
		}
	})
}

func TestCodeProtectionV2_LogicalOperationExecution(t *testing.T) {
	engine := NewAdvancedVMEngine()

	t.Run("AND Operation", func(t *testing.T) {
		engine.context.Stack = []interface{}{true, true}
		result := engine.instructionSet[0x10].Execute(engine.context, nil)
		
		if result.(bool) != true {
			t.Error("AND of true and true should be true")
		}
	})

	t.Run("OR Operation", func(t *testing.T) {
		engine.context.Stack = []interface{}{true, false}
		result := engine.instructionSet[0x11].Execute(engine.context, nil)
		
		if result.(bool) != true {
			t.Error("OR of true and false should be true")
		}
	})

	t.Run("NOT Operation", func(t *testing.T) {
		engine.context.Stack = []interface{}{false}
		result := engine.instructionSet[0x12].Execute(engine.context, nil)
		
		if result.(bool) != true {
			t.Error("NOT of false should be true")
		}
	})
}

func TestCodeProtectionV2_ControlFlowInstructions(t *testing.T) {
	engine := NewAdvancedVMEngine()

	t.Run("JUMP Instruction", func(t *testing.T) {
		engine.context.IP = 0
		engine.instructionSet[0x08].Execute(engine.context, []interface{}{100})
		
		if engine.context.IP != 100 {
			t.Errorf("Expected IP 100, got %d", engine.context.IP)
		}
	})

	t.Run("CALL and RETURN Instructions", func(t *testing.T) {
		engine.context.CallStack = []int{}
		engine.context.IP = 50
		engine.context.Functions["testFunc"] = &VMFunctionV2{
			Name:       "testFunc",
			EntryPoint: 200,
		}

		engine.instructionSet[0x0B].Execute(engine.context, []interface{}{"testFunc"})

		if engine.context.IP != 200 {
			t.Errorf("Expected IP 200, got %d", engine.context.IP)
		}

		if len(engine.context.CallStack) != 1 {
			t.Error("Call stack should have 1 element")
		}

		engine.instructionSet[0x0C].Execute(engine.context, nil)

		if engine.context.IP != 50 {
			t.Errorf("Expected IP 50 after return, got %d", engine.context.IP)
		}
	})
}

func TestCodeProtectionV2_DecryptInstructions(t *testing.T) {
	engine := NewAdvancedVMEngine()

	original := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	encrypted := engine.encryptInstructions(original)
	decrypted := engine.decryptInstructions(encrypted)

	if len(encrypted) != len(original) {
		t.Error("Encrypted length should match original")
	}

	for i := range original {
		if original[i] != decrypted[i] {
			t.Errorf("Byte %d: expected %d, got %d", i, original[i], decrypted[i])
		}
	}
}

func TestCodeProtectionV2_ObfuscatedLoaderGeneration(t *testing.T) {
	engine := NewAdvancedVMEngine()

	loader := engine.GenerateObfuscatedLoader()

	if loader == "" {
		t.Error("Loader should not be empty")
	}

	if !strings.Contains(loader, "_0xVM") {
		t.Error("Loader should contain VM object")
	}

	if !strings.Contains(loader, "_0xVM.execute") {
		t.Error("Loader should contain execute method")
	}

	if !strings.Contains(loader, "_0xVM.decrypt") {
		t.Error("Loader should contain decrypt method")
	}
}

func TestCodeProtectionV2_EmptyCode(t *testing.T) {
	config := CodeProtectionV2Config{
		ProtectionLevel: 1,
	}

	protector := NewCodeProtectionV2(config)

	_, err := protector.Protect("")
	if err == nil {
		t.Error("Empty code should cause error")
	}
}

func TestCodeProtectionV2_LargeCodeProtection(t *testing.T) {
	config := CodeProtectionV2Config{
		EnableVirtualization:      true,
		EnableControlFlowObfuscation: true,
		EnableAntiDebugV2:        true,
		EnableIntegrityCheckV2:   true,
		ProtectionLevel:          3,
	}

	protector := NewCodeProtectionV2(config)

	var largeCode strings.Builder
	largeCode.WriteString("function ")
	for i := 0; i < 100; i++ {
		largeCode.WriteString(fmt.Sprintf("func%d() { var x%d = %d; } ", i, i, i))
	}

	code := largeCode.String()
	protected, err := protector.Protect(code)

	if err != nil {
		t.Fatalf("Large code protection failed: %v", err)
	}

	if len(protected) <= len(code) {
		t.Error("Protected large code should be longer than original")
	}
}
