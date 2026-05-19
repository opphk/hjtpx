package service

import (
	"testing"
)

func TestNewWASMSandboxEnhanced(t *testing.T) {
	sandbox := NewWASMSandboxEnhanced()
	if sandbox == nil {
		t.Fatal("Expected sandbox to be non-nil")
	}
	defer sandbox.Close()
}

func TestLoadModule(t *testing.T) {
	sandbox := NewWASMSandboxEnhanced()
	defer sandbox.Close()
	
	// 创建有效的 WASM 魔术数字
	moduleData := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	module, err := sandbox.LoadModule("test-module", "Test Module", moduleData, "default")
	
	if err != nil {
		t.Fatalf("LoadModule failed: %v", err)
	}
	if module == nil {
		t.Fatal("Expected module to be non-nil")
	}
	if module.ID != "test-module" {
		t.Error("Expected module ID to match")
	}
}

func TestUnloadModule(t *testing.T) {
	sandbox := NewWASMSandboxEnhanced()
	defer sandbox.Close()
	
	moduleData := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	sandbox.LoadModule("unload-me", "Test", moduleData, "default")
	
	err := sandbox.UnloadModule("unload-me")
	if err != nil {
		t.Fatalf("UnloadModule failed: %v", err)
	}
	
	// 确认模块已经删除
	modules := sandbox.ListModels()
	if len(modules) != 0 {
		t.Error("Expected no modules after unload")
	}
}

func TestExecuteModule(t *testing.T) {
	sandbox := NewWASMSandboxEnhanced()
	defer sandbox.Close()
	
	moduleData := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	sandbox.LoadModule("execute-test", "Execute Test", moduleData, "default")
	
	input := []byte("test input")
	result, err := sandbox.ExecuteModule("execute-test", input)
	
	if err != nil {
		t.Fatalf("ExecuteModule failed: %v", err)
	}
	
	// 模拟执行会原样返回输入
	if string(result) != string(input) {
		t.Errorf("Expected output to match input")
	}
}

func TestAddPolicy(t *testing.T) {
	sandbox := NewWASMSandboxEnhanced()
	defer sandbox.Close()
	
	policy := &WasmSecurityPolicy{
		ID:          "custom-policy",
		Name:        "Custom Policy",
		Level:       WasmSecurityLevelCritical,
		MemoryLimit: 512 * 1024 * 1024,
	}
	
	sandbox.AddPolicy(policy)
	
	retrieved, exists := sandbox.GetPolicy("custom-policy")
	if !exists {
		t.Fatal("Policy not found")
	}
	if retrieved.Name != "Custom Policy" {
		t.Error("Policy name doesn't match")
	}
}

func TestWhitelistBlacklist(t *testing.T) {
	sandbox := NewWASMSandboxEnhanced()
	defer sandbox.Close()
	
	sandbox.AddToWhitelist("safe-module")
	sandbox.AddToBlacklist("dangerous-module")
	
	if !sandbox.IsWhitelisted("safe-module") {
		t.Error("Expected safe-module to be whitelisted")
	}
	if !sandbox.IsBlacklisted("dangerous-module") {
		t.Error("Expected dangerous-module to be blacklisted")
	}
}

func TestModuleState(t *testing.T) {
	sandbox := NewWASMSandboxEnhanced()
	defer sandbox.Close()
	
	moduleData := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	sandbox.LoadModule("state-test", "State Test", moduleData, "default")
	
	module, _ := sandbox.GetModel("state-test")
	if module.State != WasmModuleStateLoaded {
		t.Error("Expected module to be in Loaded state")
	}
}

func TestAuditLog(t *testing.T) {
	sandbox := NewWASMSandboxEnhanced()
	defer sandbox.Close()
	
	moduleData := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	sandbox.LoadModule("audit-test", "Audit Test", moduleData, "default")
	
	logs := sandbox.GetAuditLog(10)
	if len(logs) == 0 {
		t.Error("Expected audit log entries")
	}
}

func TestCheckAPIAccess(t *testing.T) {
	sandbox := NewWASMSandboxEnhanced()
	defer sandbox.Close()
	
	moduleData := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	sandbox.LoadModule("api-test", "API Test", moduleData, "default")
	
	// 应该允许白名单内的 API
	allowed := sandbox.CheckAPIAccess("api-test", "math")
	if !allowed {
		t.Error("Expected 'math' API to be allowed")
	}
	
	// 应该拒绝黑名单内的 API
	denied := sandbox.CheckAPIAccess("api-test", "eval")
	if denied {
		t.Error("Expected 'eval' API to be denied")
	}
}

func TestCheckHostAccess(t *testing.T) {
	sandbox := NewWASMSandboxEnhanced()
	defer sandbox.Close()
	
	moduleData := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	sandbox.LoadModule("host-test", "Host Test", moduleData, "default")
	
	// 应该允许白名单内的主机
	allowed := sandbox.CheckHostAccess("host-test", "localhost")
	if !allowed {
		t.Error("Expected localhost to be allowed")
	}
	
	// 应该拒绝黑名单内的主机
	denied := sandbox.CheckHostAccess("host-test", "malicious.com")
	if denied {
		t.Error("Expected malicious.com to be denied")
	}
}

func TestWASMSandboxGetStats(t *testing.T) {
	sandbox := NewWASMSandboxEnhanced()
	defer sandbox.Close()
	
	stats := sandbox.GetStats()
	if stats["active_modules"] != 0 {
		t.Error("Expected 0 active modules initially")
	}
	
	moduleData := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	sandbox.LoadModule("stats-test", "Stats Test", moduleData, "default")
	
	stats = sandbox.GetStats()
	if stats["active_modules"] != 1 {
		t.Error("Expected 1 active module after loading")
	}
}

func TestValidateModule(t *testing.T) {
	sandbox := NewWASMSandboxEnhanced()
	defer sandbox.Close()
	
	// 无效的模块（太小）
	invalidModule := []byte{0x00}
	valid, _ := sandbox.ValidateModule(invalidModule)
	if valid {
		t.Error("Expected small module to be invalid")
	}
	
	// 无效的模块（魔术数字错误）
	wrongMagic := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	valid, _ = sandbox.ValidateModule(wrongMagic)
	if valid {
		t.Error("Expected module with wrong magic to be invalid")
	}
	
	// 有效的模块
	validModule := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	valid, _ = sandbox.ValidateModule(validModule)
	if !valid {
		t.Error("Expected valid WASM module to pass validation")
	}
}
