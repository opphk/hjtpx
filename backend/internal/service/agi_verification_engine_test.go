package service

import (
	"context"
	"testing"
	"time"
)

func TestAGIVerificationEngine_Initialize(t *testing.T) {
	engine := NewAGIVerificationEngine()

	if engine == nil {
		t.Fatal("Failed to create AGI verification engine")
	}

	if engine.initialized {
		t.Error("Engine should not be initialized before Initialize() call")
	}

	ctx := context.Background()
	err := engine.Initialize(ctx)

	if err != nil {
		t.Errorf("Initialize() returned error: %v", err)
	}

	if !engine.initialized {
		t.Error("Engine should be initialized after Initialize() call")
	}

	err = engine.Initialize(ctx)
	if err != nil {
		t.Error("Second Initialize() should not return error")
	}
}

func TestAGIVerificationEngine_RunComprehensiveTest(t *testing.T) {
	engine := NewAGIVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	config := &AGIEngineConfig{
		EnableCrossDomain: true,
		EnableReasoning:   true,
		EnableCreativity:  true,
		MinPassScore:       50.0,
	}

	record, err := engine.RunComprehensiveTest(ctx, config)

	if err != nil {
		t.Errorf("RunComprehensiveTest() returned error: %v", err)
	}

	if record == nil {
		t.Fatal("RunComprehensiveTest() returned nil record")
	}

	if record.ID == "" {
		t.Error("Record ID should not be empty")
	}

	if len(record.Results) == 0 {
		t.Error("Should have at least one test result")
	}

	if record.OverallScore < 0 || record.OverallScore > 100 {
		t.Errorf("Overall score %f is out of valid range [0, 100]", record.OverallScore)
	}

	if record.ProcessingTime < 0 {
		t.Error("Processing time should not be negative")
	}
}

func TestAGIVerificationEngine_RunComprehensiveTestWithNilConfig(t *testing.T) {
	engine := NewAGIVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	record, err := engine.RunComprehensiveTest(ctx, nil)

	if err != nil {
		t.Errorf("RunComprehensiveTest(nil) returned error: %v", err)
	}

	if record == nil {
		t.Fatal("RunComprehensiveTest(nil) returned nil record")
	}
}

func TestAGIVerificationEngine_GetVerificationRecord(t *testing.T) {
	engine := NewAGIVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	config := &AGIEngineConfig{
		EnableCrossDomain: true,
		EnableReasoning:   true,
		EnableCreativity:  true,
	}

	record, _ := engine.RunComprehensiveTest(ctx, config)

	retrieved, err := engine.GetVerificationRecord(ctx, record.ID)

	if err != nil {
		t.Errorf("GetVerificationRecord() returned error: %v", err)
	}

	if retrieved == nil {
		t.Fatal("GetVerificationRecord() returned nil")
	}

	if retrieved.ID != record.ID {
		t.Errorf("Retrieved record ID %s does not match original %s", retrieved.ID, record.ID)
	}
}

func TestAGIVerificationEngine_GetVerificationRecord_NotFound(t *testing.T) {
	engine := NewAGIVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	_, err := engine.GetVerificationRecord(ctx, "non_existent_id")

	if err == nil {
		t.Error("GetVerificationRecord() should return error for non-existent record")
	}
}

func TestAGIVerificationEngine_GetMetrics(t *testing.T) {
	engine := NewAGIVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	config := &AGIEngineConfig{
		EnableCrossDomain: true,
		EnableReasoning:   true,
		EnableCreativity:  true,
	}

	for i := 0; i < 3; i++ {
		engine.RunComprehensiveTest(ctx, config)
	}

	metrics, err := engine.GetMetrics(ctx)

	if err != nil {
		t.Errorf("GetMetrics() returned error: %v", err)
	}

	if metrics == nil {
		t.Fatal("GetMetrics() returned nil")
	}

	if metrics.TotalVerifications != 3 {
		t.Errorf("Expected TotalVerifications to be 3, got %d", metrics.TotalVerifications)
	}

	if metrics.CategoryScores == nil {
		t.Error("CategoryScores should not be nil")
	}
}

func TestAGIVerificationEngine_NotInitialized(t *testing.T) {
	engine := NewAGIVerificationEngine()
	ctx := context.Background()

	_, err := engine.RunComprehensiveTest(ctx, nil)

	if err == nil {
		t.Error("RunComprehensiveTest() should return error when not initialized")
	}
}

func TestCrossDomainValidator_Initialize(t *testing.T) {
	validator := NewCrossDomainValidator()

	if validator.initialized {
		t.Error("Validator should not be initialized before Initialize() call")
	}

	ctx := context.Background()
	err := validator.Initialize(ctx)

	if err != nil {
		t.Errorf("Initialize() returned error: %v", err)
	}

	if !validator.initialized {
		t.Error("Validator should be initialized after Initialize() call")
	}
}

func TestCrossDomainValidator_Validate(t *testing.T) {
	validator := NewCrossDomainValidator()
	ctx := context.Background()

	if err := validator.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize validator: %v", err)
	}

	result := validator.Validate(ctx)

	if result == nil {
		t.Fatal("Validate() returned nil result")
	}

	if result.Score < 0 || result.Score > result.MaxScore {
		t.Errorf("Score %f is out of valid range [0, %f]", result.Score, result.MaxScore)
	}

	if result.Category != "cross_domain" {
		t.Errorf("Expected category 'cross_domain', got '%s'", result.Category)
	}

	if result.Metrics == nil {
		t.Error("Metrics should not be nil")
	}

	if result.Metrics["domains_tested"] == 0 {
		t.Error("Should have tested at least one domain")
	}
}

func TestAdvancedReasoningTester_Initialize(t *testing.T) {
	tester := NewAdvancedReasoningTester()

	if tester.initialized {
		t.Error("Tester should not be initialized before Initialize() call")
	}

	ctx := context.Background()
	err := tester.Initialize(ctx)

	if err != nil {
		t.Errorf("Initialize() returned error: %v", err)
	}

	if !tester.initialized {
		t.Error("Tester should be initialized after Initialize() call")
	}
}

func TestAdvancedReasoningTester_Test(t *testing.T) {
	tester := NewAdvancedReasoningTester()
	ctx := context.Background()

	if err := tester.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize tester: %v", err)
	}

	result := tester.Test(ctx)

	if result == nil {
		t.Fatal("Test() returned nil result")
	}

	if result.Score < 0 || result.Score > 100 {
		t.Errorf("Score %f is out of valid range [0, 100]", result.Score)
	}

	if result.Category != "reasoning" {
		t.Errorf("Expected category 'reasoning', got '%s'", result.Category)
	}

	expectedTypes := []string{"deductive", "inductive", "abductive", "causal", "analogical"}
	for _, rt := range expectedTypes {
		key := rt + "_score"
		if _, ok := result.Metrics[key]; !ok {
			t.Errorf("Expected metric key '%s' not found", key)
		}
	}
}

func TestCreativeThinkingEngine_Initialize(t *testing.T) {
	engine := NewCreativeThinkingEngine()

	if engine.initialized {
		t.Error("Engine should not be initialized before Initialize() call")
	}

	ctx := context.Background()
	err := engine.Initialize(ctx)

	if err != nil {
		t.Errorf("Initialize() returned error: %v", err)
	}

	if !engine.initialized {
		t.Error("Engine should be initialized after Initialize() call")
	}
}

func TestCreativeThinkingEngine_Evaluate(t *testing.T) {
	engine := NewCreativeThinkingEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	result := engine.Evaluate(ctx)

	if result == nil {
		t.Fatal("Evaluate() returned nil result")
	}

	if result.Score < 0 || result.Score > 100 {
		t.Errorf("Score %f is out of valid range [0, 100]", result.Score)
	}

	if result.Category != "creativity" {
		t.Errorf("Expected category 'creativity', got '%s'", result.Category)
	}

	expectedDimensions := []string{"fluency", "flexibility", "originality", "elaboration"}
	for _, dim := range expectedDimensions {
		key := dim + "_score"
		if _, ok := result.Metrics[key]; !ok {
			t.Errorf("Expected metric key '%s' not found", key)
		}
	}
}

func TestAGIEngineConfig_Defaults(t *testing.T) {
	engine := NewAGIVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	record, err := engine.RunComprehensiveTest(ctx, nil)

	if err != nil {
		t.Errorf("RunComprehensiveTest(nil) returned error: %v", err)
	}

	if record == nil {
		t.Fatal("RunComprehensiveTest(nil) returned nil record")
	}

	if len(record.Results) < 3 {
		t.Errorf("Expected at least 3 test results with default config, got %d", len(record.Results))
	}
}

func TestEngineMetrics_ScoreRange(t *testing.T) {
	engine := NewAGIVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	config := &AGIEngineConfig{
		EnableCrossDomain: true,
		EnableReasoning:   true,
		EnableCreativity:  true,
	}

	for i := 0; i < 10; i++ {
		record, _ := engine.RunComprehensiveTest(ctx, config)

		if record.OverallScore < 0 || record.OverallScore > 100 {
			t.Errorf("Run %d: Overall score %f is out of valid range [0, 100]", i, record.OverallScore)
		}
	}

	metrics, _ := engine.GetMetrics(ctx)

	if metrics.AverageScore < 0 || metrics.AverageScore > 100 {
		t.Errorf("Average score %f is out of valid range [0, 100]", metrics.AverageScore)
	}

	if metrics.SuccessRate < 0 || metrics.SuccessRate > 1 {
		t.Errorf("Success rate %f is out of valid range [0, 1]", metrics.SuccessRate)
	}
}

func TestVerificationRecord_ProcessingTime(t *testing.T) {
	engine := NewAGIVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	config := &AGIEngineConfig{
		EnableCrossDomain: true,
		EnableReasoning:   true,
		EnableCreativity:  true,
	}

	record, _ := engine.RunComprehensiveTest(ctx, config)

	if record.ProcessingTime < 0 {
		t.Error("Processing time should not be negative")
	}

	for _, result := range record.Results {
		if result.Duration < 0 {
			t.Errorf("Test '%s' duration is negative", result.TestName)
		}
	}
}

func TestCrossDomainValidator_MultipleDomains(t *testing.T) {
	validator := NewCrossDomainValidator()
	ctx := context.Background()

	if err := validator.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize validator: %v", err)
	}

	for i := 0; i < 5; i++ {
		result := validator.Validate(ctx)

		if result.Metrics["domains_tested"] < 1 {
			t.Error("Should test at least one domain")
		}
	}
}

func TestReasoningTester_MultipleRuns(t *testing.T) {
	tester := NewAdvancedReasoningTester()
	ctx := context.Background()

	if err := tester.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize tester: %v", err)
	}

	for i := 0; i < 5; i++ {
		result := tester.Test(ctx)

		if result.Score < 0 || result.Score > 100 {
			t.Errorf("Run %d: Score %f is out of valid range [0, 100]", i, result.Score)
		}
	}
}

func TestCreativityEngine_MultipleRuns(t *testing.T) {
	engine := NewCreativeThinkingEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	for i := 0; i < 5; i++ {
		result := engine.Evaluate(ctx)

		if result.Score < 0 || result.Score > 100 {
			t.Errorf("Run %d: Score %f is out of valid range [0, 100]", i, result.Score)
		}
	}
}

func TestEngineVerificationRecord_Timestamp(t *testing.T) {
	engine := NewAGIVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	before := time.Now()
	record, _ := engine.RunComprehensiveTest(ctx, nil)
	after := time.Now()

	if record.Timestamp.Before(before) || record.Timestamp.After(after) {
		t.Error("Record timestamp should be between before and after test execution")
	}
}

func TestEngineTestResult_Passed(t *testing.T) {
	engine := NewAGIVerificationEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	config := &AGIEngineConfig{
		EnableCrossDomain: true,
		EnableReasoning:   true,
		EnableCreativity:  true,
		MinPassScore:      50.0,
	}

	record, _ := engine.RunComprehensiveTest(ctx, config)

	for _, result := range record.Results {
		expectedPassed := result.Score >= result.MaxScore*0.6
		if result.Passed != expectedPassed {
			t.Errorf("Test '%s': Passed=%v but expected %v (score=%f, max=%f)",
				result.TestName, result.Passed, expectedPassed, result.Score, result.MaxScore)
		}
	}
}
