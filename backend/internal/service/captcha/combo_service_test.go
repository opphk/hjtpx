package captcha

import (
	"context"
	"testing"
	"time"
)

func TestComboService_Create(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	req := &CreateComboRequest{
		Types:       []string{"slider", "click", "video"},
		Strategy:    ComboStrategyMajority,
		Difficulty:  3,
		RiskScore:   0.5,
		MaxSteps:    3,
		ClientIP:    "127.0.0.1",
		UserAgent:   "test-agent",
		Fingerprint: "test-fingerprint",
	}

	resp, err := service.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if resp.SessionID == "" {
		t.Error("SessionID should not be empty")
	}

	if len(resp.Steps) == 0 {
		t.Error("Steps should not be empty")
	}

	if resp.TotalRequired <= 0 {
		t.Error("TotalRequired should be positive")
	}

	if resp.ExpiresIn <= 0 {
		t.Error("ExpiresIn should be positive")
	}

	if resp.EstimatedTime <= 0 {
		t.Error("EstimatedTime should be positive")
	}
}

func TestComboService_Create_DefaultValues(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	req := &CreateComboRequest{}

	resp, err := service.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create with default values failed: %v", err)
	}

	if resp.Strategy != ComboStrategyMajority {
		t.Errorf("Expected default strategy 'majority', got '%s'", resp.Strategy)
	}

	if resp.TotalRequired <= 0 {
		t.Error("TotalRequired should be set to a positive value")
	}
}

func TestComboService_Create_MaxDifficulty(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	req := &CreateComboRequest{
		Difficulty: 10,
		MaxSteps:   10,
	}

	resp, err := service.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create with max values failed: %v", err)
	}

	if resp.Steps[0].Difficulty > 5 {
		t.Errorf("Step difficulty should be capped at 5, got %d", resp.Steps[0].Difficulty)
	}

	if len(resp.Steps) > 5 {
		t.Errorf("Steps count should be capped at 5, got %d", len(resp.Steps))
	}
}

func TestComboService_Create_MaxSteps(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	req := &CreateComboRequest{
		MaxSteps: 2,
	}

	resp, err := service.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create with max steps failed: %v", err)
	}

	if len(resp.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(resp.Steps))
	}
}

func TestComboService_Verify_Success(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	createReq := &CreateComboRequest{
		Types:      []string{"slider", "click"},
		Strategy:   ComboStrategyAll,
		Difficulty: 2,
		MaxSteps:   2,
	}

	createResp, err := service.Create(context.Background(), createReq)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stepResults := make([]StepResult, len(createResp.Steps))
	for i := range createResp.Steps {
		stepResults[i] = StepResult{
			StepIndex:     i,
			StepType:      createResp.Steps[i].Type,
			StepSessionID: createResp.Steps[i].SessionID,
			Success:       true,
			Score:         100,
			TimeSpent:     10.0,
		}
	}

	verifyReq := &VerifyComboRequest{
		SessionID:    createResp.SessionID,
		StepResults:  stepResults,
		BehaviorData: map[string]interface{}{"total_moves": 20.0},
	}

	verifyResp, err := service.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !verifyResp.Success {
		t.Error("Verification should succeed when all steps pass")
	}

	if verifyResp.Score <= 0 {
		t.Error("Score should be positive")
	}

	if verifyResp.PassedSteps != len(createResp.Steps) {
		t.Errorf("Expected %d passed steps, got %d", len(createResp.Steps), verifyResp.PassedSteps)
	}

	if verifyResp.CanRetry {
		t.Error("Should not allow retry when all steps passed")
	}
}

func TestComboService_Verify_PartialSuccess(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	createReq := &CreateComboRequest{
		Types:      []string{"slider", "click", "gesture"},
		Strategy:   ComboStrategyMajority,
		Difficulty: 2,
		MaxSteps:   3,
	}

	createResp, err := service.Create(context.Background(), createReq)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stepResults := []StepResult{
		{
			StepIndex:     0,
			StepType:      "slider",
			Success:       true,
			Score:         100,
			TimeSpent:     10.0,
		},
		{
			StepIndex:     1,
			StepType:      "click",
			Success:       true,
			Score:         80,
			TimeSpent:     15.0,
		},
		{
			StepIndex:     2,
			StepType:      "gesture",
			Success:       false,
			Score:         0,
			TimeSpent:     5.0,
		},
	}

	verifyReq := &VerifyComboRequest{
		SessionID:    createResp.SessionID,
		StepResults:  stepResults,
		BehaviorData: nil,
	}

	verifyResp, err := service.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !verifyResp.Success {
		t.Error("Verification should succeed with majority (2/3)")
	}

	if verifyResp.PassedSteps != 2 {
		t.Errorf("Expected 2 passed steps, got %d", verifyResp.PassedSteps)
	}

	if verifyResp.FailedSteps != 1 {
		t.Errorf("Expected 1 failed step, got %d", verifyResp.FailedSteps)
	}
}

func TestComboService_Verify_Failure(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	createReq := &CreateComboRequest{
		Types:      []string{"slider", "click"},
		Strategy:   ComboStrategyAll,
		Difficulty: 2,
		MaxSteps:   2,
	}

	createResp, err := service.Create(context.Background(), createReq)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	stepResults := []StepResult{
		{
			StepIndex: 0,
			Success:   false,
			Score:     0,
			TimeSpent: 5.0,
		},
		{
			StepIndex: 1,
			Success:   true,
			Score:     80,
			TimeSpent: 10.0,
		},
	}

	verifyReq := &VerifyComboRequest{
		SessionID:    createResp.SessionID,
		StepResults:  stepResults,
		BehaviorData: nil,
	}

	verifyResp, err := service.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if verifyResp.Success {
		t.Error("Verification should fail when not all steps pass (strategy=all)")
	}

	if verifyResp.CanRetry {
		t.Error("Should not allow retry when strategy is 'all' and some steps failed")
	}
}

func TestComboService_Verify_SessionNotFound(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	verifyReq := &VerifyComboRequest{
		SessionID:   "nonexistent_session",
		StepResults: []StepResult{},
	}

	verifyResp, err := service.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify should not return error: %v", err)
	}

	if verifyResp.Success {
		t.Error("Verification should fail for nonexistent session")
	}

	if verifyResp.CanRetry {
		t.Error("Should not allow retry for nonexistent session")
	}
}

func TestComboService_GetAvailableTypes(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	req := &CreateComboRequest{}

	types := service.getAvailableTypes(req)

	if len(types) == 0 {
		t.Error("Should return at least one available type")
	}

	for _, t := range types {
		if !service.isTypeAvailable(t) {
			t.Errorf("Type '%s' should be available", t)
		}
	}
}

func TestComboService_GetAvailableTypes_WithPreference(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	req := &CreateComboRequest{
		Types: []string{"video", "ar"},
	}

	types := service.getAvailableTypes(req)

	if len(types) == 0 {
		t.Error("Should return at least one type from preference")
	}
}

func TestComboService_CreateSmartSelector(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	req := &CreateComboRequest{
		PreferFast:   true,
		PreferSecure: false,
		RiskScore:    0.3,
	}

	selector := service.createSmartSelector(req)

	if selector.weights["slider"] < 0.9 {
		t.Error("Slider weight should be high when PreferFast is true")
	}
}

func TestComboService_CreateSmartSelector_PreferSecure(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	req := &CreateComboRequest{
		PreferFast:   false,
		PreferSecure: true,
		RiskScore:    0.5,
	}

	selector := service.createSmartSelector(req)

	if selector.weights["video"] < 0.8 {
		t.Error("Video weight should be high when PreferSecure is true")
	}
}

func TestComboSelector_SelectTypes(t *testing.T) {
	selector := &ComboSelector{
		weights: map[string]float64{
			"slider":  0.9,
			"click":   0.8,
			"gesture": 0.7,
			"video":   0.6,
			"ar":      0.5,
		},
		difficultyCaps: map[string]int{
			"slider":  5,
			"click":   5,
			"gesture": 5,
			"video":   3,
			"ar":      3,
		},
	}

	testCases := []struct {
		name         string
		count        int
		difficulty   int
		expectMin    int
	}{
		{"select 2 types", 2, 3, 2},
		{"select 3 types", 3, 3, 3},
		{"select 5 types", 5, 3, 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			selected := selector.SelectTypes(tc.count, tc.difficulty, 0.3)
			if len(selected) < tc.expectMin {
				t.Errorf("Expected at least %d types, got %d", tc.expectMin, len(selected))
			}
		})
	}
}

func TestComboService_CalculateStepDifficulty(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	testCases := []struct {
		name        string
		difficulty  int
		stepIndex   int
		totalSteps  int
		expectMin   int
		expectMax   int
	}{
		{"first step easy", 2, 0, 3, 2, 3},
		{"middle step", 3, 1, 3, 3, 4},
		{"last step hard", 4, 2, 3, 4, 5},
		{"capped at 5", 5, 4, 5, 5, 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.calculateStepDifficulty(tc.difficulty, tc.stepIndex, tc.totalSteps)
			if result < tc.expectMin || result > tc.expectMax {
				t.Errorf("Expected difficulty between %d and %d, got %d", tc.expectMin, tc.expectMax, result)
			}
		})
	}
}

func TestComboService_CalculateRequiredSteps(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	testCases := []struct {
		name        string
		strategy    ComboStrategy
		totalSteps  int
		expected    int
	}{
		{"all strategy", ComboStrategyAll, 3, 3},
		{"any strategy", ComboStrategyAny, 3, 1},
		{"majority strategy", ComboStrategyMajority, 3, 2},
		{"majority even", ComboStrategyMajority, 4, 3},
		{"weighted strategy", ComboStrategyWeighted, 3, 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.calculateRequiredSteps(tc.strategy, tc.totalSteps)
			if result != tc.expected {
				t.Errorf("Expected %d required steps for %s, got %d", tc.expected, tc.strategy, result)
			}
		})
	}
}

func TestComboService_EvaluateSuccess(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	testCases := []struct {
		name        string
		config      *ComboCaptchaConfig
		passed      int
		failed      int
		expected    bool
	}{
		{
			"all pass",
			&ComboCaptchaConfig{Strategy: ComboStrategyAll, Steps: []*ComboStep{{}, {}, {}}},
			3, 0, true,
		},
		{
			"all fail",
			&ComboCaptchaConfig{Strategy: ComboStrategyAll, Steps: []*ComboStep{{}, {}, {}}},
			0, 3, false,
		},
		{
			"any pass",
			&ComboCaptchaConfig{Strategy: ComboStrategyAny, Steps: []*ComboStep{{}}},
			1, 0, true,
		},
		{
			"majority pass",
			&ComboCaptchaConfig{Strategy: ComboStrategyMajority, TotalRequired: 2, Steps: []*ComboStep{{}, {}, {}}},
			2, 1, true,
		},
		{
			"majority fail",
			&ComboCaptchaConfig{Strategy: ComboStrategyMajority, TotalRequired: 2, Steps: []*ComboStep{{}, {}, {}}},
			1, 2, false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.evaluateSuccess(tc.config, tc.passed, tc.failed)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestComboService_GenerateResultMessage(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	testCases := []struct {
		name        string
		config      *ComboCaptchaConfig
		success     bool
		passed      int
		failed      int
	}{
		{"success message", &ComboCaptchaConfig{Strategy: ComboStrategyAll, Steps: []*ComboStep{{}, {}}}, true, 2, 0},
		{"fail message", &ComboCaptchaConfig{Strategy: ComboStrategyAll, Steps: []*ComboStep{{}, {}}}, false, 1, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := service.generateResultMessage(tc.config, tc.success, tc.passed, tc.failed)
			if msg == "" {
				t.Error("Message should not be empty")
			}
		})
	}
}

func TestComboService_GetStep(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	createReq := &CreateComboRequest{
		Types:      []string{"slider", "click"},
		Difficulty: 2,
		MaxSteps:   2,
	}

	createResp, err := service.Create(context.Background(), createReq)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	step, err := service.GetStep(context.Background(), createResp.SessionID, 0)
	if err != nil {
		t.Fatalf("GetStep failed: %v", err)
	}

	if step.Index != 0 {
		t.Errorf("Expected step index 0, got %d", step.Index)
	}

	if step.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", step.Status)
	}

	if step.StartedAt == nil {
		t.Error("StartedAt should be set after GetStep")
	}
}

func TestComboService_GetStatus(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	createReq := &CreateComboRequest{
		Types:      []string{"slider", "click"},
		Difficulty: 2,
		MaxSteps:   2,
	}

	createResp, err := service.Create(context.Background(), createReq)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	status, err := service.GetStatus(context.Background(), createResp.SessionID)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if status["session_id"] != createResp.SessionID {
		t.Errorf("Expected session_id '%s', got '%v'", createResp.SessionID, status["session_id"])
	}

	if status["status"] != "pending" {
		t.Errorf("Expected status 'pending', got '%v'", status["status"])
	}

	if status["total_steps"].(int) != 2 {
		t.Errorf("Expected 2 total steps, got %v", status["total_steps"])
	}

	if status["remaining_time"].(float64) <= 0 {
		t.Error("Remaining time should be positive")
	}
}

func TestComboService_CreateStep_Video(t *testing.T) {
	service := NewComboService(nil, nil, NewVideoGeneratorService(nil, nil), nil)

	step := service.createStep(context.Background(), "video", 0, 3, &CreateComboRequest{})

	if step.Type != "video" {
		t.Errorf("Expected type 'video', got '%s'", step.Type)
	}

	if step.SubType == "" {
		t.Error("SubType should be set for video step")
	}

	if step.SessionID == "" {
		t.Error("SessionID should be generated for video step")
	}
}

func TestComboService_CreateStep_AR(t *testing.T) {
	service := NewComboService(nil, nil, nil, NewARGeneratorService(nil, nil))

	step := service.createStep(context.Background(), "ar", 0, 3, &CreateComboRequest{})

	if step.Type != "ar" {
		t.Errorf("Expected type 'ar', got '%s'", step.Type)
	}

	if step.SubType == "" {
		t.Error("SubType should be set for ar step")
	}

	if step.SessionID == "" {
		t.Error("SessionID should be generated for ar step")
	}
}

func TestComboService_CreateStep_OtherTypes(t *testing.T) {
	service := NewComboService(nil, nil, nil, nil)

	types := []string{"slider", "click", "gesture", "3d", "emoji", "semantic"}

	for _, captchaType := range types {
		step := service.createStep(context.Background(), captchaType, 0, 2, &CreateComboRequest{})

		if step.Type != captchaType {
			t.Errorf("Expected type '%s', got '%s'", captchaType, step.Type)
		}

		if step.SessionID == "" {
			t.Errorf("SessionID should be generated for %s step", captchaType)
		}

		if step.MaxAttempts != 3 {
			t.Errorf("Expected MaxAttempts 3 for %s, got %d", captchaType, step.MaxAttempts)
		}

		if step.TimeLimit <= 0 {
			t.Errorf("TimeLimit should be positive for %s", captchaType)
		}
	}
}

func TestComboStep_Structure(t *testing.T) {
	step := &ComboStep{
		Index:        0,
		Type:         "slider",
		Difficulty:   3,
		Mandatory:    true,
		Status:       "pending",
		Score:        0,
		MaxScore:     100,
		Attempts:     0,
		MaxAttempts:  3,
		SessionID:    "test_session",
		TimeLimit:    120,
	}

	if step.Index != 0 {
		t.Errorf("Expected Index 0, got %d", step.Index)
	}

	if step.Type != "slider" {
		t.Errorf("Expected Type 'slider', got '%s'", step.Type)
	}

	if step.Difficulty != 3 {
		t.Errorf("Expected Difficulty 3, got %d", step.Difficulty)
	}

	if !step.Mandatory {
		t.Error("Step should be mandatory")
	}

	if step.Status != "pending" {
		t.Errorf("Expected Status 'pending', got '%s'", step.Status)
	}
}

func TestComboCaptchaConfig_Structure(t *testing.T) {
	now := time.Now()
	config := &ComboCaptchaConfig{
		SessionID:     "test_combo_session",
		Steps:         []*ComboStep{},
		Strategy:      ComboStrategyMajority,
		TotalRequired: 2,
		Status:        "pending",
		CurrentStep:   0,
		VerifiedCount: 0,
		FailedCount:   0,
		TotalScore:    0,
		RiskScore:     0.5,
		CreatedAt:     now,
		ExpiredAt:     now.Add(10 * time.Minute),
		ClientIP:      "127.0.0.1",
		UserAgent:     "test",
		Fingerprint:   "test_fp",
	}

	if config.SessionID != "test_combo_session" {
		t.Errorf("Expected SessionID 'test_combo_session', got '%s'", config.SessionID)
	}

	if config.Strategy != ComboStrategyMajority {
		t.Errorf("Expected Strategy 'majority', got '%s'", config.Strategy)
	}

	if config.TotalRequired != 2 {
		t.Errorf("Expected TotalRequired 2, got %d", config.TotalRequired)
	}

	if config.Status != "pending" {
		t.Errorf("Expected Status 'pending', got '%s'", config.Status)
	}
}

func TestComboStrategy_Constants(t *testing.T) {
	if ComboStrategyAll != "all" {
		t.Errorf("Expected ComboStrategyAll 'all', got '%s'", ComboStrategyAll)
	}

	if ComboStrategyAny != "any" {
		t.Errorf("Expected ComboStrategyAny 'any', got '%s'", ComboStrategyAny)
	}

	if ComboStrategyMajority != "majority" {
		t.Errorf("Expected ComboStrategyMajority 'majority', got '%s'", ComboStrategyMajority)
	}

	if ComboStrategyWeighted != "weighted" {
		t.Errorf("Expected ComboStrategyWeighted 'weighted', got '%s'", ComboStrategyWeighted)
	}
}

func BenchmarkComboService_Create(b *testing.B) {
	service := NewComboService(nil, nil, nil, nil)

	req := &CreateComboRequest{
		Types:      []string{"slider", "click", "video"},
		Strategy:   ComboStrategyMajority,
		Difficulty: 3,
		MaxSteps:   3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Create(context.Background(), req)
	}
}

func BenchmarkComboService_GenerateIntelligentSteps(b *testing.B) {
	service := NewComboService(nil, nil, nil, nil)

	req := &CreateComboRequest{
		Types:      []string{"slider", "click", "video", "ar"},
		RiskScore:  0.5,
		PreferFast: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.generateIntelligentSteps(context.Background(), req, 3, 3)
	}
}

func BenchmarkComboSelector_SelectTypes(b *testing.B) {
	selector := &ComboSelector{
		weights: map[string]float64{
			"slider":  0.9,
			"click":   0.8,
			"gesture": 0.7,
			"video":   0.6,
			"ar":      0.5,
		},
		difficultyCaps: map[string]int{
			"slider":  5,
			"click":   5,
			"gesture": 5,
			"video":   3,
			"ar":      3,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector.SelectTypes(3, 3, 0.3)
	}
}

func BenchmarkComboService_CalculateStepDifficulty(b *testing.B) {
	service := NewComboService(nil, nil, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.calculateStepDifficulty(3, i%3, 3)
	}
}
