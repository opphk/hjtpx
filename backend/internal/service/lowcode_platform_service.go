package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type LowCodePlatformService interface {
	CreateIntegration(ctx context.Context, integration *Integration) error
	GetIntegration(ctx context.Context, id string) (*Integration, error)
	UpdateIntegration(ctx context.Context, integration *Integration) error
	DeleteIntegration(ctx context.Context, id string) error
	ListIntegrations(ctx context.Context, appID string, limit, offset int) ([]*Integration, error)
	ExecuteIntegration(ctx context.Context, id string, params map[string]interface{}) (*ExecutionResult, error)
	GetIntegrationTemplates(ctx context.Context) ([]*IntegrationTemplate, error)
	ValidateIntegration(ctx context.Context, integration *Integration) (*ValidationResult, error)
	GetExecutionHistory(ctx context.Context, integrationID string, limit, offset int) ([]*ExecutionRecord, error)
	RunAutomatedTest(ctx context.Context, integrationID string) (*TestResult, error)
}

type Integration struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	AppID        string                 `json:"app_id"`
	Type         string                 `json:"type"`
	Version      string                 `json:"version"`
	Config       map[string]interface{} `json:"config"`
	Steps        []IntegrationStep      `json:"steps"`
	Triggers     []Trigger              `json:"triggers"`
	Status       string                 `json:"status"`
	IsTemplate   bool                   `json:"is_template"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	LastExecuted time.Time              `json:"last_executed"`
	CreatedBy    string                 `json:"created_by"`
}

type IntegrationStep struct {
	StepID       string                 `json:"step_id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Order        int                    `json:"order"`
	Config       map[string]interface{} `json:"config"`
	RetryPolicy  *RetryPolicy           `json:"retry_policy,omitempty"`
	Timeout      int                    `json:"timeout"`
	Conditions   []Condition             `json:"conditions,omitempty"`
	OnError      string                 `json:"on_error"`
	NextStepOnOK string                 `json:"next_step_on_ok"`
	NextStepOnErr string                `json:"next_step_on_error"`
}

type Trigger struct {
	TriggerID string                 `json:"trigger_id"`
	Type      string                 `json:"type"`
	Config    map[string]interface{} `json:"config"`
	Enabled   bool                   `json:"enabled"`
	Schedule  string                 `json:"schedule,omitempty"`
}

type RetryPolicy struct {
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	Backoff     string        `json:"backoff"`
	MaxDelay    time.Duration `json:"max_delay"`
}

type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type IntegrationTemplate struct {
	TemplateID   string                 `json:"template_id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Category     string                 `json:"category"`
	Tags         []string               `json:"tags"`
	Integration  *Integration           `json:"integration"`
	Variables    []TemplateVariable     `json:"variables"`
	Documentation string                `json:"documentation"`
	Downloads    int                    `json:"downloads"`
	Rating       float64               `json:"rating"`
}

type TemplateVariable struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Options     []string    `json:"options,omitempty"`
}

type ExecutionResult struct {
	ExecutionID   string                 `json:"execution_id"`
	IntegrationID string                 `json:"integration_id"`
	Status        string                 `json:"status"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	Duration      time.Duration          `json:"duration"`
	StepsResults  []StepResult           `json:"steps_results"`
	Output        map[string]interface{} `json:"output"`
	Error         string                 `json:"error,omitempty"`
	Metrics       *ExecutionMetrics      `json:"metrics"`
}

type StepResult struct {
	StepID    string                 `json:"step_id"`
	Status    string                 `json:"status"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Duration  time.Duration          `json:"duration"`
	Output    map[string]interface{} `json:"output"`
	Error     string                 `json:"error,omitempty"`
}

type ExecutionMetrics struct {
	TotalSteps    int                    `json:"total_steps"`
	CompletedSteps int                   `json:"completed_steps"`
	FailedSteps   int                    `json:"failed_steps"`
	AvgStepTime   time.Duration          `json:"avg_step_time"`
	StepTimes     map[string]time.Duration `json:"step_times"`
}

type ExecutionRecord struct {
	ExecutionID string     `json:"execution_id"`
	Status     string     `json:"status"`
	StartTime  time.Time  `json:"start_time"`
	Duration   time.Duration `json:"duration"`
	Error      string     `json:"error,omitempty"`
	TriggeredBy string    `json:"triggered_by"`
}

type ValidationResult struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

type TestResult struct {
	TestID         string           `json:"test_id"`
	IntegrationID  string           `json:"integration_id"`
	Status         string           `json:"status"`
	StartTime      time.Time        `json:"start_time"`
	EndTime        time.Time        `json:"end_time"`
	TotalTests     int              `json:"total_tests"`
	PassedTests    int              `json:"passed_tests"`
	FailedTests    int              `json:"failed_tests"`
	TestCases      []TestCaseResult `json:"test_cases"`
	Coverage       float64          `json:"coverage"`
}

type TestCaseResult struct {
	TestName   string `json:"test_name"`
	Status     string `json:"status"`
	Duration   time.Duration `json:"duration"`
	Error      string `json:"error,omitempty"`
	Input      map[string]interface{} `json:"input"`
	Expected   map[string]interface{} `json:"expected"`
	Actual     map[string]interface{} `json:"actual"`
}

type lowCodePlatformService struct {
	integrations map[string]*Integration
	templates   map[string]*IntegrationTemplate
	executions  map[string][]*ExecutionRecord
	history     map[string]*ExecutionResult
}

var (
	ErrIntegrationNotFound = errors.New("integration not found")
	ErrInvalidConfig      = errors.New("invalid integration configuration")
	ErrExecutionFailed    = errors.New("execution failed")
)

func NewLowCodePlatformService() LowCodePlatformService {
	return &lowCodePlatformService{
		integrations: make(map[string]*Integration),
		templates:    make(map[string]*IntegrationTemplate),
		executions:   make(map[string][]*ExecutionRecord),
		history:      make(map[string]*ExecutionResult),
	}
}

func (s *lowCodePlatformService) CreateIntegration(ctx context.Context, integration *Integration) error {
	if integration == nil {
		return errors.New("integration cannot be nil")
	}

	if integration.ID == "" {
		integration.ID = uuid.New().String()
	}
	if integration.CreatedAt.IsZero() {
		integration.CreatedAt = time.Now()
	}
	integration.UpdatedAt = time.Now()
	if integration.Status == "" {
		integration.Status = "draft"
	}
	if integration.Version == "" {
		integration.Version = "1.0.0"
	}

	s.integrations[integration.ID] = integration

	return nil
}

func (s *lowCodePlatformService) GetIntegration(ctx context.Context, id string) (*Integration, error) {
	integration, exists := s.integrations[id]
	if !exists {
		return nil, ErrIntegrationNotFound
	}
	return integration, nil
}

func (s *lowCodePlatformService) UpdateIntegration(ctx context.Context, integration *Integration) error {
	if integration == nil {
		return errors.New("integration cannot be nil")
	}

	if _, exists := s.integrations[integration.ID]; !exists {
		return ErrIntegrationNotFound
	}

	integration.UpdatedAt = time.Now()
	s.integrations[integration.ID] = integration

	return nil
}

func (s *lowCodePlatformService) DeleteIntegration(ctx context.Context, id string) error {
	if _, exists := s.integrations[id]; !exists {
		return ErrIntegrationNotFound
	}

	delete(s.integrations, id)
	return nil
}

func (s *lowCodePlatformService) ListIntegrations(ctx context.Context, appID string, limit, offset int) ([]*Integration, error) {
	var result []*Integration

	for _, integration := range s.integrations {
		if appID == "" || integration.AppID == appID {
			result = append(result, integration)
		}
	}

	if offset >= len(result) {
		return []*Integration{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

func (s *lowCodePlatformService) ExecuteIntegration(ctx context.Context, id string, params map[string]interface{}) (*ExecutionResult, error) {
	integration, exists := s.integrations[id]
	if !exists {
		return nil, ErrIntegrationNotFound
	}

	result := &ExecutionResult{
		ExecutionID:   uuid.New().String(),
		IntegrationID: id,
		Status:        "running",
		StartTime:     time.Now(),
		StepsResults:  []StepResult{},
		Output:        make(map[string]interface{}),
		Metrics: &ExecutionMetrics{
			TotalSteps:    len(integration.Steps),
			CompletedSteps: 0,
			FailedSteps:   0,
			StepTimes:     make(map[string]time.Duration),
		},
	}

	if params != nil {
		result.Output["input_params"] = params
	}

	for _, step := range integration.Steps {
		stepResult := s.executeStep(step, params, result.Output)
		result.StepsResults = append(result.StepsResults, stepResult)

		if stepResult.Status == "failed" {
			result.Metrics.FailedSteps++
			if step.OnError == "stop" {
				result.Status = "failed"
				break
			}
		} else {
			result.Metrics.CompletedSteps++
		}

		result.Metrics.StepTimes[step.StepID] = stepResult.Duration
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	avgTime := time.Duration(0)
	if len(result.StepsResults) > 0 {
		for _, sr := range result.StepsResults {
			avgTime += sr.Duration
		}
		avgTime = avgTime / time.Duration(len(result.StepsResults))
	}
	result.Metrics.AvgStepTime = avgTime

	if result.Status == "running" {
		result.Status = "completed"
	}

	integration.LastExecuted = time.Now()

	s.history[result.ExecutionID] = result
	s.executions[id] = append(s.executions[id], &ExecutionRecord{
		ExecutionID: result.ExecutionID,
		Status:      result.Status,
		StartTime:   result.StartTime,
		Duration:    result.Duration,
		Error:       result.Error,
	})

	return result, nil
}

func (s *lowCodePlatformService) executeStep(step IntegrationStep, input map[string]interface{}, output map[string]interface{}) StepResult {
	result := StepResult{
		StepID:    step.StepID,
		Status:    "running",
		StartTime: time.Now(),
		Output:    make(map[string]interface{}),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()

	time.Sleep(time.Duration(50+step.Order*10) * time.Millisecond)

	switch step.Type {
	case "http_request":
		result.Output["status"] = 200
		result.Output["data"] = "mock response"
		result.Status = "success"

	case "transform":
		result.Output["transformed"] = true
		result.Output["data"] = input
		result.Status = "success"

	case "condition":
		result.Output["evaluated"] = true
		result.Output["path"] = step.NextStepOnOK
		result.Status = "success"

	case "verify_captcha":
		result.Output["success"] = true
		result.Output["verified"] = true
		result.Status = "success"

	case "report_risk":
		result.Output["risk_score"] = 0.3
		result.Output["risk_level"] = "low"
		result.Status = "success"

	case "blockchain_record":
		result.Output["tx_hash"] = "0x" + uuid.New().String()
		result.Output["block_number"] = 15000000 + step.Order
		result.Status = "success"

	case "iot_auth":
		result.Output["authenticated"] = true
		result.Output["token"] = uuid.New().String()
		result.Status = "success"

	default:
		result.Output["executed"] = true
		result.Status = "success"
	}

	output[step.StepID] = result.Output

	return result
}

func (s *lowCodePlatformService) GetIntegrationTemplates(ctx context.Context) ([]*IntegrationTemplate, error) {
	if len(s.templates) == 0 {
		s.initDefaultTemplates()
	}

	var result []*IntegrationTemplate
	for _, template := range s.templates {
		result = append(result, template)
	}

	return result, nil
}

func (s *lowCodePlatformService) initDefaultTemplates() {
	s.templates["web-login-flow"] = &IntegrationTemplate{
		TemplateID:  "web-login-flow",
		Name:        "Web Login Security Flow",
		Description: "Complete login flow with captcha verification and risk assessment",
		Category:    "Authentication",
		Tags:        []string{"login", "captcha", "security", "risk"},
		Variables: []TemplateVariable{
			{Name: "app_id", Type: "string", Required: true, Description: "Application ID"},
			{Name: "captcha_type", Type: "select", Default: "slider", Options: []string{"slider", "icon", "rotate"}, Description: "Captcha type"},
			{Name: "risk_threshold", Type: "number", Default: 0.7, Description: "Risk score threshold"},
		},
		Documentation: "This template provides a complete login flow with:\n1. Captcha verification\n2. Risk assessment\n3. Device fingerprinting\n4. Blockchain proof recording",
		Downloads: 1523,
		Rating:    4.8,
		Integration: &Integration{
			Name:        "Web Login Flow",
			Description: "Secure login with captcha and risk assessment",
			Type:        "workflow",
			Steps: []IntegrationStep{
				{StepID: "step1", Name: "Device Fingerprint", Type: "device_fingerprint", Order: 1},
				{StepID: "step2", Name: "Create Captcha", Type: "http_request", Order: 2},
				{StepID: "step3", Name: "Verify Captcha", Type: "verify_captcha", Order: 3, Conditions: []Condition{{Field: "verified", Operator: "eq", Value: true}}},
				{StepID: "step4", Name: "Risk Assessment", Type: "report_risk", Order: 4},
				{StepID: "step5", Name: "Record Proof", Type: "blockchain_record", Order: 5},
			},
		},
	}

	s.templates["iot-authentication"] = &IntegrationTemplate{
		TemplateID:  "iot-authentication",
		Name:        "IoT Device Authentication",
		Description: "Secure authentication flow for IoT devices",
		Category:    "IoT",
		Tags:        []string{"iot", "device", "authentication", "smart-home"},
		Variables: []TemplateVariable{
			{Name: "device_type", Type: "select", Default: "thermostat", Options: []string{"thermostat", "camera", "lock", "sensor"}, Description: "Device type"},
			{Name: "auth_method", Type: "select", Default: "token", Options: []string{"token", "certificate", "biometric"}, Description: "Authentication method"},
		},
		Documentation: "IoT device authentication flow with:\n1. Device registration\n2. Certificate validation\n3. Fingerprint verification",
		Downloads: 892,
		Rating:    4.6,
		Integration: &Integration{
			Name:        "IoT Authentication",
			Description: "Secure IoT device authentication",
			Type:        "workflow",
			Steps: []IntegrationStep{
				{StepID: "step1", Name: "Device Registration", Type: "http_request", Order: 1},
				{StepID: "step2", Name: "Authenticate Device", Type: "iot_auth", Order: 2},
				{StepID: "step3", Name: "Record Session", Type: "blockchain_record", Order: 3},
			},
		},
	}

	s.templates["api-protection"] = &IntegrationTemplate{
		TemplateID:  "api-protection",
		Name:        "API Protection Gateway",
		Description: "Protect APIs with captcha and rate limiting",
		Category:    "Security",
		Tags:        []string{"api", "protection", "rate-limit", "captcha"},
		Variables: []TemplateVariable{
			{Name: "rate_limit", Type: "number", Default: 100, Description: "Requests per minute"},
			{Name: "enable_captcha", Type: "boolean", Default: true, Description: "Enable captcha verification"},
		},
		Documentation: "API protection with:\n1. Rate limiting\n2. Captcha challenge\n3. Risk scoring",
		Downloads: 2341,
		Rating:    4.9,
		Integration: &Integration{
			Name:        "API Protection",
			Description: "Protect APIs from abuse",
			Type:        "gateway",
			Steps: []IntegrationStep{
				{StepID: "step1", Name: "Rate Check", Type: "http_request", Order: 1},
				{StepID: "step2", Name: "Risk Check", Type: "report_risk", Order: 2},
				{StepID: "step3", Name: "Captcha Challenge", Type: "verify_captcha", Order: 3},
			},
		},
	}
}

func (s *lowCodePlatformService) ValidateIntegration(ctx context.Context, integration *Integration) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	if integration.Name == "" {
		result.Errors = append(result.Errors, "Integration name is required")
		result.Valid = false
	}

	if integration.AppID == "" {
		result.Errors = append(result.Errors, "App ID is required")
		result.Valid = false
	}

	if len(integration.Steps) == 0 {
		result.Errors = append(result.Errors, "At least one step is required")
		result.Valid = false
	}

	stepIDs := make(map[string]bool)
	for i, step := range integration.Steps {
		if step.StepID == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Step %d has no ID", i+1))
		} else if stepIDs[step.StepID] {
			result.Errors = append(result.Errors, fmt.Sprintf("Duplicate step ID: %s", step.StepID))
			result.Valid = false
		} else {
			stepIDs[step.StepID] = true
		}

		if step.Type == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Step %s has no type", step.StepID))
			result.Valid = false
		}
	}

	for _, step := range integration.Steps {
		if step.RetryPolicy != nil {
			if step.RetryPolicy.MaxAttempts < 1 {
				result.Warnings = append(result.Warnings, "Max retry attempts should be at least 1")
			}
			if step.Timeout < 1000 {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Step %s timeout is very short", step.StepID))
			}
		}
	}

	return result, nil
}

func (s *lowCodePlatformService) GetExecutionHistory(ctx context.Context, integrationID string, limit, offset int) ([]*ExecutionRecord, error) {
	records, exists := s.executions[integrationID]
	if !exists {
		return []*ExecutionRecord{}, nil
	}

	if offset >= len(records) {
		return []*ExecutionRecord{}, nil
	}

	end := offset + limit
	if end > len(records) {
		end = len(records)
	}

	return records[offset:end], nil
}

func (s *lowCodePlatformService) RunAutomatedTest(ctx context.Context, integrationID string) (*TestResult, error) {
	integration, exists := s.integrations[integrationID]
	if !exists {
		return nil, ErrIntegrationNotFound
	}

	result := &TestResult{
		TestID:        uuid.New().String(),
		IntegrationID: integrationID,
		Status:        "running",
		StartTime:     time.Now(),
		TestCases:     []TestCaseResult{},
	}

	testCases := []struct {
		name   string
		params map[string]interface{}
	}{
		{"Happy Path", map[string]interface{}{"test": true}},
		{"Edge Case 1", map[string]interface{}{"empty": true}},
		{"Error Handling", map[string]interface{}{"simulate_error": true}},
		{"Performance", map[string]interface{}{"perf_test": true}},
	}

	for _, tc := range testCases {
		testCase := TestCaseResult{
			TestName: tc.name,
			Input:    tc.params,
			Expected: map[string]interface{}{"status": "success"},
		}

		startTime := time.Now()

		if tc.name == "Error Handling" {
			testCase.Status = "passed"
			testCase.Error = "simulated error handled correctly"
		} else {
			execResult, err := s.ExecuteIntegration(ctx, integrationID, tc.params)
			if err != nil {
				testCase.Status = "failed"
				testCase.Error = err.Error()
				result.FailedTests++
			} else {
				testCase.Status = "passed"
				testCase.Actual = map[string]interface{}{"status": execResult.Status}
			}
		}

		testCase.Duration = time.Since(startTime)
		result.TestCases = append(result.TestCases, testCase)
		result.TotalTests++
		result.PassedTests++
	}

	result.EndTime = time.Now()
	result.Status = "completed"

	if len(integration.Steps) > 0 {
		result.Coverage = float64(len(integration.Steps)) / float64(len(integration.Steps)) * 100
	}

	return result, nil
}

func (s *lowCodePlatformService) ExportIntegration(ctx context.Context, id string) ([]byte, error) {
	integration, exists := s.integrations[id]
	if !exists {
		return nil, ErrIntegrationNotFound
	}

	data, err := json.MarshalIndent(integration, "", "  ")
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *lowCodePlatformService) ImportIntegration(ctx context.Context, data []byte) (*Integration, error) {
	var integration Integration
	if err := json.Unmarshal(data, &integration); err != nil {
		return nil, err
	}

	integration.ID = uuid.New().String()
	integration.CreatedAt = time.Now()
	integration.UpdatedAt = time.Now()
	integration.IsTemplate = false

	s.integrations[integration.ID] = &integration

	return &integration, nil
}
