package service

import (
	"testing"
	"time"
)

func TestNewRuleAnalyticsService(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	if service == nil {
		t.Fatal("Expected non-nil service")
	}
	if service.engine != engine {
		t.Error("Expected engine to be set")
	}
	if service.history == nil {
		t.Error("Expected history to be initialized")
	}
	if service.alertThresholds == nil {
		t.Error("Expected alert thresholds to be initialized")
	}
}

func TestRecordEvaluation(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	entry := &RuleAnalyticsEntry{
		Timestamp:      time.Now(),
		SessionID:      "test_session_1",
		IPAddress:      "192.168.1.1",
		TotalScore:     0.8,
		IsBot:          true,
		Confidence:     0.9,
		TriggeredRules: []string{"trajectory_speed_too_fast"},
		CategoryScores: map[string]float64{
			"speed": 0.9,
		},
		ProcessingTime: 10 * time.Millisecond,
	}

	service.RecordEvaluation(entry)

	if len(service.history) != 1 {
		t.Errorf("Expected 1 entry in history, got %d", len(service.history))
	}
}

func TestRecordMultipleEvaluations(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	for i := 0; i < 10; i++ {
		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "test_session",
			IPAddress:      "192.168.1.1",
			TotalScore:     0.5 + float64(i)*0.05,
			IsBot:          i%2 == 0,
			Confidence:     0.8,
			TriggeredRules: []string{"trajectory_speed_too_fast"},
			ProcessingTime: 5 * time.Millisecond,
		}
		service.RecordEvaluation(entry)
	}

	if len(service.history) != 10 {
		t.Errorf("Expected 10 entries in history, got %d", len(service.history))
	}
}

func TestGetAnalyticsSummary(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	for i := 0; i < 20; i++ {
		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "session_" + string(rune(i)),
			IPAddress:      "192.168.1.1",
			TotalScore:     0.6,
			IsBot:          i%2 == 0,
			Confidence:     0.75,
			TriggeredRules: []string{"trajectory_speed_too_fast"},
			CategoryScores: map[string]float64{
				"speed": 0.8,
			},
			ProcessingTime: 10 * time.Millisecond,
		}
		service.RecordEvaluation(entry)
	}

	summary := service.GetAnalyticsSummary(24 * time.Hour)

	if summary.TotalEvaluations != 20 {
		t.Errorf("Expected 20 evaluations, got %d", summary.TotalEvaluations)
	}

	if summary.TotalBots != 10 {
		t.Errorf("Expected 10 bots, got %d", summary.TotalBots)
	}

	if summary.BotRate != 0.5 {
		t.Errorf("Expected bot rate 0.5, got %f", summary.BotRate)
	}
}

func TestGetAnalyticsSummaryTimeRange(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	oldEntry := &RuleAnalyticsEntry{
		Timestamp:      time.Now().Add(-48 * time.Hour),
		SessionID:      "old_session",
		IPAddress:      "192.168.1.1",
		TotalScore:     0.9,
		IsBot:          true,
		Confidence:     0.9,
		TriggeredRules: []string{"trajectory_speed_too_fast"},
		ProcessingTime: 10 * time.Millisecond,
	}
	service.RecordEvaluation(oldEntry)

	newEntry := &RuleAnalyticsEntry{
		Timestamp:      time.Now(),
		SessionID:      "new_session",
		IPAddress:      "192.168.1.2",
		TotalScore:     0.3,
		IsBot:          false,
		Confidence:     0.7,
		TriggeredRules: []string{},
		ProcessingTime: 5 * time.Millisecond,
	}
	service.RecordEvaluation(newEntry)

	summary24h := service.GetAnalyticsSummary(24 * time.Hour)
	if summary24h.TotalEvaluations != 1 {
		t.Errorf("Expected 1 evaluation in 24h range, got %d", summary24h.TotalEvaluations)
	}

	summary48h := service.GetAnalyticsSummary(48 * time.Hour)
	if summary48h.TotalEvaluations != 2 {
		t.Errorf("Expected 2 evaluations in 48h range, got %d", summary48h.TotalEvaluations)
	}
}

func TestGetRulePerformance(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	for i := 0; i < 10; i++ {
		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "session_" + string(rune(i)),
			IPAddress:      "192.168.1.1",
			TotalScore:     0.7,
			IsBot:          true,
			Confidence:     0.85,
			TriggeredRules: []string{"trajectory_speed_too_fast", "click_interval_too_short"},
			ProcessingTime: 8 * time.Millisecond,
		}
		service.RecordEvaluation(entry)
	}

	performance := service.GetRulePerformance("trajectory_speed_too_fast")

	if performance == nil {
		t.Fatal("Expected non-nil performance")
	}

	if performance.TotalTriggers != 10 {
		t.Errorf("Expected 10 triggers, got %d", performance.TotalTriggers)
	}

	if performance.HitRate == 0 {
		t.Error("Expected non-zero hit rate")
	}
}

func TestGetRulePerformanceNonExistent(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	performance := service.GetRulePerformance("non_existent_rule")

	if performance == nil {
		t.Fatal("Expected non-nil performance for non-existent rule")
	}

	if performance.TotalTriggers != 0 {
		t.Errorf("Expected 0 triggers for non-existent rule, got %d", performance.TotalTriggers)
	}
}

func TestGetCategoryAnalytics(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	for i := 0; i < 15; i++ {
		var rules []string
		var category string

		if i%3 == 0 {
			rules = []string{"trajectory_speed_too_fast"}
			category = "speed"
		} else if i%3 == 1 {
			rules = []string{"click_interval_too_short"}
			category = "click"
		} else {
			rules = []string{"captcha_failure_rate_too_high"}
			category = "captcha"
		}

		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "session_" + string(rune(i)),
			IPAddress:      "192.168.1.1",
			TotalScore:     0.6,
			IsBot:          true,
			Confidence:     0.8,
			TriggeredRules: rules,
			CategoryScores: map[string]float64{
				category: 0.7,
			},
			ProcessingTime: 10 * time.Millisecond,
		}
		service.RecordEvaluation(entry)
	}

	categories := service.GetCategoryAnalytics()

	if len(categories) == 0 {
		t.Error("Expected categories to be populated")
	}

	if _, exists := categories["speed"]; !exists {
		t.Error("Expected speed category to exist")
	}

	if categories["speed"].TotalTriggers != 5 {
		t.Errorf("Expected 5 triggers in speed category, got %d", categories["speed"].TotalTriggers)
	}
}

func TestGetAlerts(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	for i := 0; i < 5; i++ {
		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "session_" + string(rune(i)),
			IPAddress:      "192.168.1.1",
			TotalScore:     0.85,
			IsBot:          true,
			Confidence:     0.9,
			TriggeredRules: []string{"trajectory_speed_too_fast"},
			ProcessingTime: 10 * time.Millisecond,
		}
		service.RecordEvaluation(entry)
	}

	alerts := service.GetAlerts(24 * time.Hour)

	if len(alerts) != 5 {
		t.Errorf("Expected 5 alerts, got %d", len(alerts))
	}

	for _, alert := range alerts {
		if alert.Severity != "high" {
			t.Errorf("Expected high severity for bot alert, got %s", alert.Severity)
		}
	}
}

func TestGetAlertsTimeRange(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	oldEntry := &RuleAnalyticsEntry{
		Timestamp:      time.Now().Add(-48 * time.Hour),
		SessionID:      "old_session",
		IPAddress:      "192.168.1.1",
		TotalScore:     0.8,
		IsBot:          true,
		Confidence:     0.85,
		TriggeredRules: []string{"trajectory_speed_too_fast"},
		ProcessingTime: 10 * time.Millisecond,
	}
	service.RecordEvaluation(oldEntry)

	newEntry := &RuleAnalyticsEntry{
		Timestamp:      time.Now(),
		SessionID:      "new_session",
		IPAddress:      "192.168.1.2",
		TotalScore:     0.8,
		IsBot:          true,
		Confidence:     0.85,
		TriggeredRules: []string{"click_interval_too_short"},
		ProcessingTime: 10 * time.Millisecond,
	}
	service.RecordEvaluation(newEntry)

	alerts24h := service.GetAlerts(24 * time.Hour)
	if len(alerts24h) != 1 {
		t.Errorf("Expected 1 alert in 24h range, got %d", len(alerts24h))
	}

	alerts48h := service.GetAlerts(48 * time.Hour)
	if len(alerts48h) != 2 {
		t.Errorf("Expected 2 alerts in 48h range, got %d", len(alerts48h))
	}
}

func TestSetAlertThreshold(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	testCases := []struct {
		thresholdType string
		value         float64
		expectedErr   bool
	}{
		{"bot_rate", 0.5, false},
		{"high_risk_score", 0.8, false},
		{"anomaly_count", 50, false},
		{"false_positive", 0.15, false},
		{"invalid_type", 0.5, true},
	}

	for _, tc := range testCases {
		err := service.SetAlertThreshold(tc.thresholdType, tc.value)
		if tc.expectedErr && err == nil {
			t.Errorf("Expected error for threshold type %s", tc.thresholdType)
		}
		if !tc.expectedErr && err != nil {
			t.Errorf("Unexpected error for threshold type %s: %v", tc.thresholdType, err)
		}
	}
}

func TestExportAnalyticsReportJSON(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	for i := 0; i < 10; i++ {
		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "session_" + string(rune(i)),
			IPAddress:      "192.168.1.1",
			TotalScore:     0.6,
			IsBot:          i%2 == 0,
			Confidence:     0.8,
			TriggeredRules: []string{"trajectory_speed_too_fast"},
			CategoryScores: map[string]float64{
				"speed": 0.7,
			},
			ProcessingTime: 10 * time.Millisecond,
		}
		service.RecordEvaluation(entry)
	}

	report := service.ExportAnalyticsReport("json")

	if report == "" {
		t.Error("Expected non-empty JSON report")
	}

	if len(report) < 50 {
		t.Error("Expected detailed JSON report")
	}
}

func TestExportAnalyticsReportMarkdown(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	for i := 0; i < 10; i++ {
		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "session_" + string(rune(i)),
			IPAddress:      "192.168.1.1",
			TotalScore:     0.6,
			IsBot:          i%2 == 0,
			Confidence:     0.8,
			TriggeredRules: []string{"trajectory_speed_too_fast"},
			ProcessingTime: 10 * time.Millisecond,
		}
		service.RecordEvaluation(entry)
	}

	report := service.ExportAnalyticsReport("markdown")

	if report == "" {
		t.Error("Expected non-empty markdown report")
	}

	if len(report) < 100 {
		t.Error("Expected detailed markdown report")
	}
}

func TestExportAnalyticsReportText(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	for i := 0; i < 10; i++ {
		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "session_" + string(rune(i)),
			IPAddress:      "192.168.1.1",
			TotalScore:     0.6,
			IsBot:          i%2 == 0,
			Confidence:     0.8,
			TriggeredRules: []string{"trajectory_speed_too_fast"},
			ProcessingTime: 10 * time.Millisecond,
		}
		service.RecordEvaluation(entry)
	}

	report := service.ExportAnalyticsReport("text")

	if report == "" {
		t.Error("Expected non-empty text report")
	}
}

func TestExportAnalyticsReportUnsupported(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	report := service.ExportAnalyticsReport("xml")

	if report == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestAlertGeneration(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	entry := &RuleAnalyticsEntry{
		Timestamp:      time.Now(),
		SessionID:      "test_session",
		IPAddress:      "192.168.1.1",
		TotalScore:     0.75,
		IsBot:          true,
		Confidence:     0.85,
		TriggeredRules: []string{"trajectory_speed_too_fast", "click_interval_too_short"},
		ProcessingTime: 10 * time.Millisecond,
	}

	service.RecordEvaluation(entry)
}

func TestTrendDataGeneration(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	for i := 0; i < 50; i++ {
		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "session_" + string(rune(i)),
			IPAddress:      "192.168.1.1",
			TotalScore:     0.5 + float64(i)*0.01,
			IsBot:          i%2 == 0,
			Confidence:     0.75,
			TriggeredRules: []string{"trajectory_speed_too_fast"},
			ProcessingTime: 5 * time.Millisecond,
		}
		service.RecordEvaluation(entry)
	}

	summary := service.GetAnalyticsSummary(24 * time.Hour)

	if len(summary.TrendData) == 0 {
		t.Error("Expected trend data to be generated")
	}
}

func TestCategoryBreakdown(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	categories := []string{"speed", "click", "slider", "captcha"}

	for i := 0; i < 20; i++ {
		category := categories[i%len(categories)]
		var rules []string

		switch category {
		case "speed":
			rules = []string{"trajectory_speed_too_fast"}
		case "click":
			rules = []string{"click_interval_too_short"}
		case "slider":
			rules = []string{"slider_release_precision_too_high"}
		case "captcha":
			rules = []string{"captcha_failure_rate_too_high"}
		}

		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "session_" + string(rune(i)),
			IPAddress:      "192.168.1.1",
			TotalScore:     0.6,
			IsBot:          true,
			Confidence:     0.8,
			TriggeredRules: rules,
			CategoryScores: map[string]float64{
				category: 0.7,
			},
			ProcessingTime: 10 * time.Millisecond,
		}
		service.RecordEvaluation(entry)
	}

	summary := service.GetAnalyticsSummary(24 * time.Hour)

	if len(summary.CategoryBreakdown) != len(categories) {
		t.Errorf("Expected %d categories, got %d", len(categories), len(summary.CategoryBreakdown))
	}
}

func TestTopRulesCalculation(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	ruleFrequency := map[string]int{
		"trajectory_speed_too_fast": 20,
		"click_interval_too_short": 15,
		"slider_release_precision_too_high": 10,
		"captcha_failure_rate_too_high": 5,
	}

	for i := 0; i < 50; i++ {
		rules := []string{}

		for rule, count := range ruleFrequency {
			if i < count {
				rules = append(rules, rule)
			}
		}

		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "session_" + string(rune(i)),
			IPAddress:      "192.168.1.1",
			TotalScore:     0.6,
			IsBot:          true,
			Confidence:     0.8,
			TriggeredRules: rules,
			ProcessingTime: 10 * time.Millisecond,
		}
		service.RecordEvaluation(entry)
	}

	summary := service.GetAnalyticsSummary(24 * time.Hour)

	if len(summary.TopRules) == 0 {
		t.Error("Expected top rules to be calculated")
	}

	if len(summary.TopRules) > 0 && summary.TopRules[0].TotalTriggers < summary.TopRules[1].TotalTriggers {
		t.Error("Expected top rules to be sorted by trigger count")
	}
}

func TestAverageProcessingTime(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	processingTimes := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
	}

	for i, pt := range processingTimes {
		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "session_" + string(rune(i)),
			IPAddress:      "192.168.1.1",
			TotalScore:     0.5,
			IsBot:          false,
			Confidence:     0.7,
			TriggeredRules: []string{},
			ProcessingTime: pt,
		}
		service.RecordEvaluation(entry)
	}

	summary := service.GetAnalyticsSummary(24 * time.Hour)

	expectedAvg := 30 * time.Millisecond
	if summary.AverageProcessingTime != expectedAvg {
		t.Errorf("Expected average processing time %v, got %v", expectedAvg, summary.AverageProcessingTime)
	}
}

func TestRuleAnalyticsHandler(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	for i := 0; i < 10; i++ {
		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "session_" + string(rune(i)),
			IPAddress:      "192.168.1.1",
			TotalScore:     0.6,
			IsBot:          i%2 == 0,
			Confidence:     0.8,
			TriggeredRules: []string{"trajectory_speed_too_fast"},
			ProcessingTime: 10 * time.Millisecond,
		}
		service.RecordEvaluation(entry)
	}

	handler := NewRuleAnalyticsHandler(service)

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}

	if handler.analyticsService != service {
		t.Error("Expected handler to have correct service")
	}
}

func TestHistorySizeLimit(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)
	service.maxHistorySize = 5

	for i := 0; i < 10; i++ {
		entry := &RuleAnalyticsEntry{
			Timestamp:      time.Now(),
			SessionID:      "session_" + string(rune(i)),
			IPAddress:      "192.168.1.1",
			TotalScore:     0.5,
			IsBot:          false,
			Confidence:     0.7,
			TriggeredRules: []string{},
			ProcessingTime: 5 * time.Millisecond,
		}
		service.RecordEvaluation(entry)
	}

	if len(service.history) > service.maxHistorySize {
		t.Errorf("Expected history size <= %d, got %d", service.maxHistorySize, len(service.history))
	}
}

func TestEmptyHistory(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	summary := service.GetAnalyticsSummary(24 * time.Hour)

	if summary.TotalEvaluations != 0 {
		t.Errorf("Expected 0 evaluations for empty history, got %d", summary.TotalEvaluations)
	}

	alerts := service.GetAlerts(24 * time.Hour)
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts for empty history, got %d", len(alerts))
	}
}

func TestAlertThresholdDefaults(t *testing.T) {
	engine := NewAdvancedRuleEngine()
	service := NewRuleAnalyticsService(engine)

	if service.alertThresholds.BotRateThreshold != 0.3 {
		t.Errorf("Expected default BotRateThreshold 0.3, got %f", service.alertThresholds.BotRateThreshold)
	}

	if service.alertThresholds.HighRiskScoreThreshold != 0.7 {
		t.Errorf("Expected default HighRiskScoreThreshold 0.7, got %f", service.alertThresholds.HighRiskScoreThreshold)
	}

	if service.alertThresholds.AnomalyCountThreshold != 100 {
		t.Errorf("Expected default AnomalyCountThreshold 100, got %d", service.alertThresholds.AnomalyCountThreshold)
	}
}

func TestRuleStatStruct(t *testing.T) {
	stat := RuleStat{
		Name:        "test_rule",
		HitCount:    100,
		HitRate:     0.5,
		Accuracy:    0.85,
	}

	if stat.Name != "test_rule" {
		t.Errorf("Expected name test_rule, got %s", stat.Name)
	}

	if stat.HitCount != 100 {
		t.Errorf("Expected HitCount 100, got %d", stat.HitCount)
	}
}

func TestTrendEntry(t *testing.T) {
	trend := TrendEntry{
		Timestamp: time.Now(),
		BotCount:  10,
		HumanCount: 90,
		TotalCount: 100,
		BotRate:   0.1,
		AvgScore:  0.45,
	}

	if trend.TotalCount != 100 {
		t.Errorf("Expected TotalCount 100, got %d", trend.TotalCount)
	}

	if trend.BotRate != 0.1 {
		t.Errorf("Expected BotRate 0.1, got %f", trend.BotRate)
	}
}

func TestCategoryAnalytics(t *testing.T) {
	cat := CategoryAnalytics{
		TotalTriggers: 50,
		AverageScore:  0.65,
		TopRules:     []string{"rule1", "rule2"},
		HitRate:      0.25,
	}

	if cat.TotalTriggers != 50 {
		t.Errorf("Expected TotalTriggers 50, got %d", cat.TotalTriggers)
	}

	if cat.AverageScore != 0.65 {
		t.Errorf("Expected AverageScore 0.65, got %f", cat.AverageScore)
	}

	if len(cat.TopRules) != 2 {
		t.Errorf("Expected 2 top rules, got %d", len(cat.TopRules))
	}
}
