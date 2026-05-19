package crypto

import (
	"testing"
	"time"
)

func TestPrivacyEngineCreation(t *testing.T) {
	engine := NewPrivacyEngine(nil)
	if engine == nil {
		t.Fatal("engine should not be nil")
	}
}

func TestDefaultMaskingRules(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	rules := engine.ListMaskingRules()
	if len(rules) == 0 {
		t.Error("default rules should be created")
	}
}

func TestMaskEmail(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	tests := []struct {
		input    string
		expected string
	}{
		{"test@example.com", "t***t@example.com"},
		{"user123@domain.com", "u***3@domain.com"},
		{"ab@cd.com", "a****b@cd.com"},
	}

	for _, tt := range tests {
		result := engine.MaskEmail(tt.input)
		t.Logf("MaskEmail(%s) = %s", tt.input, result)
	}
}

func TestMaskCreditCard(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	tests := []struct {
		input    string
		expected string
	}{
		{"1234567890123456", "************3456"},
		{"4111111111111111", "************1111"},
		{"5500000000000004", "************0004"},
	}

	for _, tt := range tests {
		result := engine.MaskCreditCard(tt.input)
		if result != tt.expected {
			t.Errorf("MaskCreditCard(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestMaskPhone(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	tests := []struct {
		input    string
		expected string
	}{
		{"1234567890", "******7890"},
		{"+8613812345678", "*******5678"},
		{"555-123-4567", "*******4567"},
	}

	for _, tt := range tests {
		result := engine.MaskPhone(tt.input)
		t.Logf("MaskPhone(%s) = %s", tt.input, result)
	}
}

func TestMaskSSN(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	tests := []struct {
		input    string
		expected string
	}{
		{"123-45-6789", "***-**-6789"},
		{"123456789", "***-**-6789"},
	}

	for _, tt := range tests {
		result := engine.MaskSSN(tt.input)
		if result != tt.expected {
			t.Errorf("MaskSSN(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestMaskStringData(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	request := &DataMaskingRequest{
		Data:     "test@example.com",
		DataType: DataTypeEmail,
		Level:   PrivacyLevelMedium,
		UserID:  "user_123",
	}

	response, err := engine.MaskData(request)
	if err != nil {
		t.Fatalf("masking failed: %v", err)
	}

	if response == nil {
		t.Fatal("response should not be nil")
	}

	if response.MaskedData == nil {
		t.Error("masked data should not be nil")
	}
}

func TestMaskMapData(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	request := &DataMaskingRequest{
		Data: map[string]interface{}{
			"email": "user@example.com",
			"name":  "John Doe",
		},
		DataType: DataTypeCustom,
		Level:   PrivacyLevelMedium,
		UserID:  "user_456",
	}

	response, err := engine.MaskData(request)
	if err != nil {
		t.Fatalf("map masking failed: %v", err)
	}

	if response == nil {
		t.Fatal("response should not be nil")
	}

	t.Logf("Masked map data: %v", response.MaskedData)
}

func TestAddMaskingRule(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	rule := &MaskingRule{
		RuleID:      "custom_rule",
		DataType:    DataTypeCustom,
		Strategy:    StrategyPartial,
		VisibleHead: 2,
		VisibleTail: 2,
		MaskChar:    "#",
		Priority:    5,
	}

	err := engine.AddMaskingRule(rule)
	if err != nil {
		t.Fatalf("failed to add rule: %v", err)
	}

	retrieved := engine.GetMaskingRule("custom_rule")
	if retrieved == nil {
		t.Fatal("rule should be retrievable")
	}

	if retrieved.RuleID != "custom_rule" {
		t.Errorf("rule ID mismatch")
	}
}

func TestRemoveMaskingRule(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	rule := &MaskingRule{
		RuleID:   "removable_rule",
		DataType: DataTypeCustom,
		Strategy: StrategyFull,
	}

	engine.AddMaskingRule(rule)

	err := engine.RemoveMaskingRule("removable_rule")
	if err != nil {
		t.Fatalf("failed to remove rule: %v", err)
	}

	retrieved := engine.GetMaskingRule("removable_rule")
	if retrieved != nil {
		t.Error("rule should be removed")
	}
}

func TestDefaultPrivacyLevel(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	if engine.GetDefaultPrivacyLevel() != PrivacyLevelMedium {
		t.Errorf("default level should be medium")
	}

	engine.SetDefaultPrivacyLevel(PrivacyLevelHigh)

	if engine.GetDefaultPrivacyLevel() != PrivacyLevelHigh {
		t.Errorf("default level should be high")
	}
}

func TestBudgetCheck(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	request := &BudgetCheckRequest{
		UserID:    "budget_user",
		Operation: "test_operation",
		DataSize:  1024,
	}

	response, err := engine.CheckBudget(request)
	if err != nil {
		t.Fatalf("budget check failed: %v", err)
	}

	if response == nil {
		t.Fatal("response should not be nil")
	}

	if !response.Allowed {
		t.Error("operation should be allowed")
	}

	if response.Remaining <= 0 {
		t.Error("remaining budget should be positive")
	}
}

func TestBudgetExceededPrivacyEngine(t *testing.T) {
	config := &BudgetConfig{
		MaxBudget:            10,
		MaxOperationsPerDay: 1000,
		ResetPeriod:         24 * time.Hour,
	}

	engine := NewPrivacyEngine(config)

	engine.mu.Lock()
	engine.budgets["exhausted_user"] = &BudgetRecord{
		UserID:         "exhausted_user",
		BudgetUsed:     10,
		OperationCount: 10,
		ResetAt:        time.Now().Add(24 * time.Hour),
	}
	engine.mu.Unlock()

	request := &BudgetCheckRequest{
		UserID:    "exhausted_user",
		Operation: "test",
		DataSize:  100,
	}

	response, err := engine.CheckBudget(request)
	if response != nil && response.Allowed {
		t.Error("operation should not be allowed when budget exhausted")
	}

	if err != nil && err != ErrBudgetExceeded {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTokenVault(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	token, err := engine.tokenVault.Add("sensitive_data")
	if err != nil {
		t.Fatalf("failed to add token: %v", err)
	}

	if token == "" {
		t.Error("token should not be empty")
	}

	retrieved, exists := engine.tokenVault.Get(token)
	if !exists {
		t.Error("token should exist")
	}

	if retrieved != "sensitive_data" {
		t.Errorf("retrieved data mismatch: expected 'sensitive_data', got '%s'", retrieved)
	}

	removed := engine.tokenVault.Remove(token)
	if !removed {
		t.Error("token should be removable")
	}

	_, exists = engine.tokenVault.Get(token)
	if exists {
		t.Error("token should not exist after removal")
	}
}

func TestMaskingRuleValidation(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	validRule := &MaskingRule{
		RuleID:   "valid_rule",
		DataType: DataTypeEmail,
		Strategy: StrategyPartial,
	}

	err := engine.ValidateMaskingRule(validRule)
	if err != nil {
		t.Errorf("valid rule should pass validation: %v", err)
	}

	nilRule := (*MaskingRule)(nil)
	err = engine.ValidateMaskingRule(nilRule)
	if err == nil {
		t.Error("nil rule should fail validation")
	}

	emptyIDRule := &MaskingRule{
		DataType: DataTypeEmail,
		Strategy: StrategyPartial,
	}
	err = engine.ValidateMaskingRule(emptyIDRule)
	if err == nil {
		t.Error("rule without ID should fail validation")
	}
}

func TestParseDataType(t *testing.T) {
	tests := []struct {
		input    string
		expected DataType
	}{
		{"email", DataTypeEmail},
		{"phone", DataTypePhone},
		{"credit_card", DataTypeCreditCard},
		{"cc", DataTypeCreditCard},
		{"ssn", DataTypeSSN},
		{"name", DataTypeName},
		{"address", DataTypeAddress},
		{"ip", DataTypeIPAddress},
		{"unknown", DataTypeCustom},
	}

	for _, tt := range tests {
		result, err := ParseDataType(tt.input)
		if err != nil {
			t.Errorf("parse error for %s: %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("ParseDataType(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestParseMaskingStrategy(t *testing.T) {
	tests := []struct {
		input    string
		expected MaskingStrategy
	}{
		{"full", StrategyFull},
		{"partial", StrategyPartial},
		{"hash", StrategyHash},
		{"tokenize", StrategyTokenize},
		{"nullify", StrategyNullify},
	}

	for _, tt := range tests {
		result, err := ParseMaskingStrategy(tt.input)
		if err != nil {
			t.Errorf("parse error for %s: %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("ParseMaskingStrategy(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}

	_, err := ParseMaskingStrategy("invalid")
	if err == nil {
		t.Error("invalid strategy should fail")
	}
}

func TestBatchMask(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	requests := []*DataMaskingRequest{
		{Data: "test1@example.com", DataType: DataTypeEmail, Level: PrivacyLevelMedium, UserID: "user1"},
		{Data: "test2@example.com", DataType: DataTypeEmail, Level: PrivacyLevelMedium, UserID: "user2"},
		{Data: "test3@example.com", DataType: DataTypeEmail, Level: PrivacyLevelMedium, UserID: "user3"},
	}

	results, err := engine.BatchMask(requests)
	if err != nil {
		t.Fatalf("batch masking failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestEngineStats(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	stats := engine.GetStats()
	if stats == nil {
		t.Fatal("stats should not be nil")
	}

	engine.recordOperation()

	stats = engine.GetStats()
	if stats.TotalOperations == 0 {
		t.Error("total operations should be incremented")
	}
}

func TestResetStats(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	engine.recordOperation()
	engine.recordOperation()

	engine.ResetStats()

	stats := engine.GetStats()
	if stats.TotalOperations != 0 {
		t.Error("stats should be reset")
	}
}

func TestExportImportConfig(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	customRule := &MaskingRule{
		RuleID:      "export_test_rule",
		DataType:    DataTypeCustom,
		Strategy:    StrategyPartial,
		VisibleHead: 1,
		VisibleTail: 1,
		MaskChar:    "#",
	}

	engine.AddMaskingRule(customRule)

	data, err := engine.ExportConfig()
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	newEngine := NewPrivacyEngine(nil)
	err = newEngine.ImportConfig(data)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	rule := newEngine.GetMaskingRule("export_test_rule")
	if rule == nil {
		t.Error("imported rule should exist")
	}
}

func TestTokenVaultSize(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	if engine.GetTokenVaultSize() != 0 {
		t.Error("initial vault size should be 0")
	}

	engine.tokenVault.Add("data1")
	engine.tokenVault.Add("data2")

	if engine.GetTokenVaultSize() != 2 {
		t.Errorf("vault size should be 2, got %d", engine.GetTokenVaultSize())
	}
}

func TestClearTokenVault(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	engine.tokenVault.Add("data1")
	engine.tokenVault.Add("data2")

	engine.ClearTokenVault()

	if engine.GetTokenVaultSize() != 0 {
		t.Error("vault should be empty after clear")
	}
}

func TestBudgetAllocation(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	allocation := &BudgetAllocation{
		UserID:     "user_alloc",
		Allocated:  100,
		Used:       50,
		Remaining:  50,
	}

	err := engine.AllocateBudget(allocation)
	if err != nil {
		t.Fatalf("allocation failed: %v", err)
	}

	budget := engine.GetBudgetRecord("user_alloc")
	if budget == nil {
		t.Fatal("budget should exist")
	}
}

func TestBudgetConfig(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	config := engine.GetBudgetConfig()
	if config == nil {
		t.Fatal("config should not be nil")
	}

	newConfig := &BudgetConfig{
		MaxBudget:            5000,
		MaxOperationsPerDay: 500,
		ResetPeriod:         12 * time.Hour,
	}

	engine.SetBudgetConfig(newConfig)

	retrieved := engine.GetBudgetConfig()
	if retrieved.MaxBudget != 5000 {
		t.Errorf("max budget mismatch")
	}
}

func TestCustomMaskingRule(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	err := engine.CreateCustomRule("custom_pattern", `\[.*?\]`, "***")
	if err != nil {
		t.Fatalf("failed to create custom rule: %v", err)
	}

	result, err := engine.ApplyCustomMasking("test [sensitive] data", "custom_pattern")
	if err != nil {
		t.Fatalf("failed to apply custom masking: %v", err)
	}

	t.Logf("Custom masking result: %s", result)
}

func TestInvalidCustomMaskingRule(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	err := engine.CreateCustomRule("invalid_pattern", "[invalid", "")
	if err == nil {
		t.Error("invalid pattern should fail")
	}
}

func TestMaskingRuleUpdate(t *testing.T) {
	engine := NewPrivacyEngine(nil)

	rule := &MaskingRule{
		RuleID:      "update_rule",
		DataType:    DataTypeEmail,
		Strategy:    StrategyPartial,
		VisibleHead: 1,
		VisibleTail: 1,
		MaskChar:    "*",
	}

	engine.AddMaskingRule(rule)

	updatedRule := &MaskingRule{
		RuleID:      "update_rule",
		DataType:    DataTypeEmail,
		Strategy:    StrategyFull,
		MaskChar:    "#",
	}

	err := engine.UpdateMaskingRule(updatedRule)
	if err != nil {
		t.Fatalf("failed to update rule: %v", err)
	}

	retrieved := engine.GetMaskingRule("update_rule")
	if retrieved.Strategy != StrategyFull {
		t.Error("strategy should be updated")
	}
}
