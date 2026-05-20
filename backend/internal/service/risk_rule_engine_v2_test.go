package service

import (
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&models.RiskRule{},
		&models.RiskRuleTriggerHistory{},
		&models.Blacklist{},
		&models.AlertRecord{},
		&models.WebhookConfig{},
		&models.VerificationLog{},
		&models.Workflow{},
		&models.WorkflowExecution{},
	)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func TestNewRiskRuleEngineV2(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	if engine == nil {
		t.Fatal("expected engine to be created")
	}

	if engine.db != db {
		t.Error("expected engine to have db connection")
	}

	if engine.compiledRules == nil {
		t.Error("expected compiledRules to be initialized")
	}

	if engine.ruleConditions == nil {
		t.Error("expected ruleConditions to be initialized")
	}

	if engine.ruleStats == nil {
		t.Error("expected ruleStats to be initialized")
	}

	if engine.eventEmitter == nil {
		t.Error("expected eventEmitter to be initialized")
	}
}

func TestRiskRuleEngineV2_CompileRule(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	rule := &models.RiskRule{
		Name:        "Test IP Frequency Rule",
		Description: "Test rule for IP frequency detection",
		RuleType:    "frequency",
		Condition:   "ip_frequency",
		Action:      "block",
		Params:      `{"threshold": 100, "window": 60}`,
		Severity:    "high",
		IsEnabled:   true,
	}

	compiled, err := engine.CompileRule(rule)
	if err != nil {
		t.Fatalf("failed to compile rule: %v", err)
	}

	if compiled == nil {
		t.Fatal("expected compiled rule to be returned")
	}

	if compiled.Rule != rule {
		t.Error("expected compiled rule to reference original rule")
	}

	if compiled.Condition == nil {
		t.Error("expected condition to be compiled")
	}

	if len(compiled.Actions) == 0 {
		t.Error("expected at least one action to be compiled")
	}
}

func TestRiskRuleEngineV2_EvaluateRules(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	rule := &models.RiskRule{
		Name:        "Test Risk Score Rule",
		Description: "Test rule for risk score",
		RuleType:    "risk_score",
		Condition:   "risk_score",
		Action:      "challenge",
		Params:      `{"score_threshold": 50}`,
		Severity:    "medium",
		IsEnabled:   true,
	}

	engine.CompileRule(rule)

	ctx := &model.RiskContext{
		SessionID: "test-session-123",
		IPAddress: "192.168.1.1",
		RiskScore: 75.0,
	}

	triggered, actions, err := engine.EvaluateRules(ctx)
	if err != nil {
		t.Fatalf("failed to evaluate rules: %v", err)
	}

	if len(triggered) == 0 {
		t.Error("expected at least one triggered rule")
	}

	if len(actions) == 0 {
		t.Error("expected at least one action")
	}

	if triggered[0].Rule.Action != "challenge" {
		t.Errorf("expected action to be 'challenge', got '%s'", triggered[0].Rule.Action)
	}
}

func TestRiskRuleEngineV2_TestRule(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	rule := &models.RiskRule{
		Name:        "Test Rule",
		Description: "Test rule",
		RuleType:    "test",
		Condition:   "risk_score",
		Action:      "block",
		Params:      `{"score_threshold": 50}`,
		IsEnabled:   true,
	}
	db.Create(rule)

	testCases := []*RuleTestCase{
		{
			Name:        "High Risk Score",
			Description: "Should trigger for high risk score",
			Context: &model.RiskContext{
				SessionID: "test-1",
				RiskScore: 80.0,
			},
			ExpectedHit: true,
		},
		{
			Name:        "Low Risk Score",
			Description: "Should not trigger for low risk score",
			Context: &model.RiskContext{
				SessionID: "test-2",
				RiskScore: 30.0,
			},
			ExpectedHit: false,
		},
	}

	suite, err := engine.TestRule(rule.ID, testCases)
	if err != nil {
		t.Fatalf("failed to test rule: %v", err)
	}

	if suite == nil {
		t.Fatal("expected test suite to be returned")
	}

	if suite.TotalTests != 2 {
		t.Errorf("expected 2 tests, got %d", suite.TotalTests)
	}

	if suite.PassedTests != 2 {
		t.Errorf("expected 2 passed tests, got %d", suite.PassedTests)
	}
}

func TestRiskRuleEngineV2_SimulateRuleExecution(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	rule := &models.RiskRule{
		Name:        "Test Simulation Rule",
		Description: "Test rule for simulation",
		RuleType:    "simulation",
		Condition:   "risk_score",
		Action:      "block",
		Params:      `{"score_threshold": 60}`,
		IsEnabled:   true,
	}
	db.Create(rule)
	engine.CompileRule(rule)

	ctx := &model.RiskContext{
		SessionID: "sim-session-123",
		IPAddress: "10.0.0.1",
		RiskScore: 70.0,
	}

	result, err := engine.SimulateRuleExecution(rule.ID, ctx)
	if err != nil {
		t.Fatalf("failed to simulate rule execution: %v", err)
	}

	if result == nil {
		t.Fatal("expected simulation result to be returned")
	}

	if result.RuleID != rule.ID {
		t.Errorf("expected rule ID %d, got %d", rule.ID, result.RuleID)
	}

	if result.TotalExecutionTime < 0 {
		t.Error("expected non-negative execution time")
	}
}

func TestRiskRuleEngineV2_ValidateRule(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	testCases := []struct {
		name        string
		rule        *models.RiskRule
		expectError bool
	}{
		{
			name: "Valid Rule",
			rule: &models.RiskRule{
				Name:      "Valid Rule",
				RuleType:  "test",
				Condition: "risk_score",
				Action:    "block",
				Severity:  "high",
			},
			expectError: false,
		},
		{
			name: "Missing Name",
			rule: &models.RiskRule{
				RuleType:  "test",
				Condition: "risk_score",
				Action:    "block",
			},
			expectError: true,
		},
		{
			name: "Invalid Action",
			rule: &models.RiskRule{
				Name:      "Invalid Action Rule",
				RuleType:  "test",
				Condition: "risk_score",
				Action:    "invalid_action",
			},
			expectError: true,
		},
		{
			name: "Invalid Severity",
			rule: &models.RiskRule{
				Name:     "Invalid Severity Rule",
				RuleType: "test",
				Condition: "risk_score",
				Action:   "block",
				Severity: "invalid",
			},
			expectError: true,
		},
		{
			name: "Invalid Params JSON",
			rule: &models.RiskRule{
				Name:      "Invalid JSON Rule",
				RuleType:  "test",
				Condition: "risk_score",
				Action:    "block",
				Params:    "{invalid json}",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errors, err := engine.ValidateRule(tc.rule)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.expectError && len(errors) == 0 {
				t.Error("expected validation errors, got none")
			}

			if !tc.expectError && len(errors) > 0 {
				t.Errorf("expected no validation errors, got %d: %v", len(errors), errors)
			}
		})
	}
}

func TestRiskRuleEngineV2_GetStats(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	stats := engine.GetStats()

	if stats.TotalEvaluations != 0 {
		t.Errorf("expected 0 evaluations, got %d", stats.TotalEvaluations)
	}

	if stats.TotalHits != 0 {
		t.Errorf("expected 0 hits, got %d", stats.TotalHits)
	}

	if stats.TotalMisses != 0 {
		t.Errorf("expected 0 misses, got %d", stats.TotalMisses)
	}
}

func TestRiskRuleEngineV2_Metrics(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	engine.RecordMetric("test_metric", 42.0, MetricTypeCounter, map[string]string{"label": "value"})

	metric, ok := engine.GetMetric("test_metric", map[string]string{"label": "value"})
	if !ok {
		t.Fatal("expected metric to be found")
	}

	if metric.Value != 42.0 {
		t.Errorf("expected metric value 42.0, got %f", metric.Value)
	}

	allMetrics := engine.GetAllMetrics()
	if len(allMetrics) == 0 {
		t.Error("expected at least one metric")
	}
}

func TestRiskRuleEngineV2_EventEmitter(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	eventReceived := false

	handler := func(event *RuleEvent) {
		eventReceived = true
		if event.RuleID != 123 {
			t.Errorf("expected rule ID 123, got %d", event.RuleID)
		}
	}

	engine.Subscribe("test.event", handler)
	engine.Subscribe("*", handler)

	engine.emitEvent(&RuleEvent{
		EventType: "test.event",
		RuleID:    123,
		Timestamp: time.Now(),
	})

	time.Sleep(100 * time.Millisecond)

	if !eventReceived {
		t.Error("expected event to be received")
	}

	engine.Unsubscribe("test.event", handler)
}

func TestRiskRuleEngineV2_EvalConditions(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	t.Run("evalIPFrequency", func(t *testing.T) {
		ctx := &model.RiskContext{
			IPAddress: "192.168.1.100",
		}

		params := map[string]interface{}{
			"threshold": float64(10),
			"window":    float64(60),
		}

		result := engine.evalIPFrequency(ctx, params)
		if result {
			t.Error("expected no IP frequency match for new IP")
		}
	})

	t.Run("evalVelocity", func(t *testing.T) {
		ctx := &model.RiskContext{
			MouseSpeed: 5000.0,
		}

		params := map[string]interface{}{
			"speed_threshold": float64(2000),
		}

		result := engine.evalVelocity(ctx, params)
		if !result {
			t.Error("expected velocity match for high mouse speed")
		}
	})

	t.Run("evalRiskScore", func(t *testing.T) {
		ctx := &model.RiskContext{
			RiskScore: 85.0,
		}

		params := map[string]interface{}{
			"score_threshold": float64(70),
		}

		result := engine.evalRiskScore(ctx, params)
		if !result {
			t.Error("expected risk score match for high risk")
		}
	})

	t.Run("evalGeoAnomaly", func(t *testing.T) {
		ctx := &model.RiskContext{
			LastKnownLocation: "Beijing",
			CurrentLocation:   "Shanghai",
		}

		params := map[string]interface{}{
			"allowed_regions": []interface{}{"Beijing"},
		}

		result := engine.evalGeoAnomaly(ctx, params)
		if !result {
			t.Error("expected geo anomaly for different location")
		}
	})

	t.Run("evalDeviceReputation", func(t *testing.T) {
		ctx := &model.RiskContext{
			DeviceReputationScore: 30.0,
		}

		params := map[string]interface{}{
			"min_reputation": float64(50),
		}

		result := engine.evalDeviceReputation(ctx, params)
		if !result {
			t.Error("expected device reputation match for low reputation")
		}
	})

	t.Run("evalTimeWindow", func(t *testing.T) {
		ctx := &model.RiskContext{}

		params := map[string]interface{}{
			"start_hour": float64(9),
			"end_hour":   float64(17),
		}

		result := engine.evalTimeWindow(ctx, params)
		currentHour := time.Now().Hour()
		expected := currentHour < 9 || currentHour > 17

		if result != expected {
			t.Errorf("expected time window result %v for current hour %d", expected, currentHour)
		}
	})
}

func TestRiskRuleEngineV2_ExecuteAutomatedResponse(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	ctx := &model.RiskContext{
		SessionID: "response-test-session",
		IPAddress: "172.16.0.1",
		RiskScore: 90.0,
	}

	actions := []ActionConfig{
		{
			Type:   "block",
			Target: "user",
			Params: map[string]interface{}{
				"reason": "High risk score detected",
			},
			Priority: 100,
		},
	}

	response, err := engine.ExecuteAutomatedResponse(ctx, actions)
	if err != nil {
		t.Fatalf("failed to execute automated response: %v", err)
	}

	if response == nil {
		t.Fatal("expected response to be returned")
	}

	if response.Decision != "block" {
		t.Errorf("expected decision 'block', got '%s'", response.Decision)
	}

	if len(response.Actions) == 0 {
		t.Error("expected at least one action result")
	}
}

func TestRiskRuleEngineV2_CalculatePathEfficiency(t *testing.T) {
	db := setupTestDB(t)
	engine := NewRiskRuleEngineV2(db)

	t.Run("Normal Path", func(t *testing.T) {
		traceData := []model.TracePoint{
			{X: 0, Y: 0},
			{X: 50, Y: 50},
			{X: 100, Y: 100},
		}

		efficiency := engine.calculatePathEfficiency(traceData)
		if efficiency <= 0 || efficiency > 1 {
			t.Errorf("expected efficiency between 0 and 1, got %f", efficiency)
		}
	})

	t.Run("Short Path", func(t *testing.T) {
		traceData := []model.TracePoint{
			{X: 0, Y: 0},
		}

		efficiency := engine.calculatePathEfficiency(traceData)
		if efficiency != 0 {
			t.Errorf("expected 0 for single point, got %f", efficiency)
		}
	})
}
