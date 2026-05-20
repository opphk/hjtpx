package service

import (
	"context"
	"testing"
)

func TestAGIVerificationService(t *testing.T) {
	service := NewAGIVerificationService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	if !service.initialized {
		t.Error("Service should be initialized")
	}
}

func TestAGIVerificationService_VerifyModel(t *testing.T) {
	service := NewAGIVerificationService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	tests := []struct {
		name        string
		req         *AGIVerificationRequest
		expectError bool
	}{
		{
			name: "Basic verification with all test types",
			req: &AGIVerificationRequest{
				ModelID:    "test-model-1",
				TestTypes:  []string{"reasoning", "knowledge", "creativity"},
				Difficulty: 2,
			},
			expectError: false,
		},
		{
			name: "Verification with single test type",
			req: &AGIVerificationRequest{
				ModelID:    "test-model-2",
				TestTypes:  []string{"reasoning"},
				Difficulty: 1,
			},
			expectError: false,
		},
		{
			name: "Verification with difficulty 5",
			req: &AGIVerificationRequest{
				ModelID:    "test-model-3",
				TestTypes:  []string{"knowledge"},
				Difficulty: 5,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.VerifyModel(ctx, tt.req)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if result == nil {
					t.Fatal("Result should not be nil")
				}

				if !result.Success {
					t.Error("Verification should succeed")
				}

				if result.RecordID == "" {
					t.Error("RecordID should not be empty")
				}

				if result.OverallScore < 0 || result.OverallScore > 100 {
					t.Errorf("OverallScore should be between 0 and 100, got %f", result.OverallScore)
				}

				if len(result.Tests) != len(tt.req.TestTypes) {
					t.Errorf("Expected %d tests, got %d", len(tt.req.TestTypes), len(result.Tests))
				}
			}
		})
	}
}

func TestReasoningTestEngine(t *testing.T) {
	engine := NewReasoningTestEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	if !engine.initialized {
		t.Error("Engine should be initialized")
	}
}

func TestReasoningTestEngine_RunTest(t *testing.T) {
	engine := NewReasoningTestEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	for difficulty := 1; difficulty <= 5; difficulty++ {
		t.Run("Difficulty_"+string(rune('0'+difficulty)), func(t *testing.T) {
			result, err := engine.RunTest(ctx, difficulty)

			if err != nil {
				t.Fatalf("Failed to run test at difficulty %d: %v", difficulty, err)
			}

			if result == nil {
				t.Fatal("Result should not be nil")
			}

			if result.Score < 0 {
				t.Errorf("Score should be non-negative, got %f", result.Score)
			}

			if result.MaxScore <= 0 {
				t.Errorf("MaxScore should be positive, got %f", result.MaxScore)
			}

			if result.Duration < 0 {
				t.Errorf("Duration should be non-negative, got %d", result.Duration)
			}

			if result.Difficulty != difficulty {
				t.Errorf("Expected difficulty %d, got %d", difficulty, result.Difficulty)
			}
		})
	}
}

func TestKnowledgeValidator(t *testing.T) {
	validator := NewKnowledgeValidator()
	ctx := context.Background()

	if err := validator.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize validator: %v", err)
	}

	if !validator.initialized {
		t.Error("Validator should be initialized")
	}
}

func TestKnowledgeValidator_RunTest(t *testing.T) {
	validator := NewKnowledgeValidator()
	ctx := context.Background()

	if err := validator.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize validator: %v", err)
	}

	for difficulty := 1; difficulty <= 3; difficulty++ {
		t.Run("Difficulty_"+string(rune('0'+difficulty)), func(t *testing.T) {
			result, err := validator.RunTest(ctx, difficulty)

			if err != nil {
				t.Fatalf("Failed to run test at difficulty %d: %v", difficulty, err)
			}

			if result == nil {
				t.Fatal("Result should not be nil")
			}

			if result.Score < 0 {
				t.Errorf("Score should be non-negative, got %f", result.Score)
			}

			if result.MaxScore <= 0 {
				t.Errorf("MaxScore should be positive, got %f", result.MaxScore)
			}
		})
	}
}

func TestCreativityAssessor(t *testing.T) {
	assessor := NewCreativityAssessor()
	ctx := context.Background()

	if err := assessor.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize assessor: %v", err)
	}

	if !assessor.initialized {
		t.Error("Assessor should be initialized")
	}
}

func TestCreativityAssessor_RunTest(t *testing.T) {
	assessor := NewCreativityAssessor()
	ctx := context.Background()

	if err := assessor.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize assessor: %v", err)
	}

	for difficulty := 1; difficulty <= 5; difficulty++ {
		t.Run("Difficulty_"+string(rune('0'+difficulty)), func(t *testing.T) {
			result, err := assessor.RunTest(ctx, difficulty)

			if err != nil {
				t.Fatalf("Failed to run test at difficulty %d: %v", difficulty, err)
			}

			if result == nil {
				t.Fatal("Result should not be nil")
			}

			if result.Score < 0 || result.Score > 100 {
				t.Errorf("Score should be between 0 and 100, got %f", result.Score)
			}

			if result.MaxScore != 100 {
				t.Errorf("MaxScore should be 100, got %f", result.MaxScore)
			}
		})
	}
}

func TestAGIVerificationService_GetVerificationRecord(t *testing.T) {
	service := NewAGIVerificationService()
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	req := &AGIVerificationRequest{
		ModelID:    "test-record",
		TestTypes:  []string{"reasoning"},
		Difficulty: 2,
	}

	result, err := service.VerifyModel(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create verification: %v", err)
	}

	record, err := service.GetVerificationRecord(ctx, result.RecordID)
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}

	if record == nil {
		t.Fatal("Record should not be nil")
	}

	if record.ID != result.RecordID {
		t.Errorf("Expected record ID %s, got %s", result.RecordID, record.ID)
	}

	_, err = service.GetVerificationRecord(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent record")
	}
}

func TestAGIAppendUnique(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected []string
	}{
		{
			name:     "Append to empty slice",
			slice:    []string{},
			item:     "test",
			expected: []string{"test"},
		},
		{
			name:     "Append new item",
			slice:    []string{"a", "b"},
			item:     "c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Don't append duplicate",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendUnique(tt.slice, tt.item)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}

			for i := range tt.expected {
				if result[i] != tt.expected[i] {
					t.Errorf("Expected %s at index %d, got %s", tt.expected[i], i, result[i])
				}
			}
		})
	}
}

func TestAGICalculateOverallScore(t *testing.T) {
	service := NewAGIVerificationService()

	tests := []struct {
		name        string
		testResults []*AGITestResult
		expected    float64
	}{
		{
			name:        "Empty tests",
			testResults: []*AGITestResult{},
			expected:    0,
		},
		{
			name: "Single test",
			testResults: []*AGITestResult{
				{Score: 75, MaxScore: 100},
			},
			expected: 75,
		},
		{
			name: "Multiple tests",
			testResults: []*AGITestResult{
				{Score: 80, MaxScore: 100},
				{Score: 60, MaxScore: 100},
				{Score: 100, MaxScore: 100},
			},
			expected: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateOverallScore(tt.testResults)

			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestAGIDetermineStatus(t *testing.T) {
	service := NewAGIVerificationService()

	tests := []struct {
		name     string
		score    float64
		expected string
	}{
		{name: "Excellent score", score: 95, expected: "excellent"},
		{name: "Good score", score: 80, expected: "good"},
		{name: "Pass score", score: 65, expected: "pass"},
		{name: "Fail score", score: 50, expected: "fail"},
		{name: "Excellent boundary", score: 90, expected: "excellent"},
		{name: "Good boundary", score: 75, expected: "good"},
		{name: "Pass boundary", score: 60, expected: "pass"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := service.determineStatus(tt.score)

			if status != tt.expected {
				t.Errorf("Expected status %s, got %s", tt.expected, status)
			}
		})
	}
}
