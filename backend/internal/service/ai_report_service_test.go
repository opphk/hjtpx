package service

import (
	"context"
	"testing"
	"time"
)

func TestNewAIReportService(t *testing.T) {
	service := NewAIReportService()
	if service == nil {
		t.Fatal("NewAIReportService returned nil")
	}
}

func TestGenerateTrendForecast(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	config := ForecastConfig{
		Metric:       "requests",
		Horizon:      24 * time.Hour,
		Confidence:   0.95,
		Seasonality:  true,
		Outlieraware: true,
	}

	forecast, err := service.GenerateTrendForecast(ctx, "requests", config)
	if err != nil {
		t.Fatalf("GenerateTrendForecast failed: %v", err)
	}

	if forecast.Metric != "requests" {
		t.Errorf("Expected metric 'requests', got '%s'", forecast.Metric)
	}

	if forecast.CurrentValue <= 0 {
		t.Error("CurrentValue should be positive")
	}

	if forecast.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", forecast.Confidence)
	}

	if len(forecast.ForecastPoints) == 0 {
		t.Error("ForecastPoints should not be empty")
	}

	if len(forecast.ForecastPoints) != 24 {
		t.Errorf("Expected 24 forecast points for 24-hour horizon, got %d", len(forecast.ForecastPoints))
	}

	for i, point := range forecast.ForecastPoints {
		if point.Timestamp.Before(time.Now()) {
			t.Errorf("Point %d timestamp is in the past: %v", i, point.Timestamp)
		}

		if point.LowerBound > point.Value {
			t.Errorf("Point %d: LowerBound (%f) > Value (%f)", i, point.LowerBound, point.Value)
		}

		if point.UpperBound < point.Value {
			t.Errorf("Point %d: UpperBound (%f) < Value (%f)", i, point.UpperBound, point.Value)
		}
	}

	if forecast.PredictedValue <= 0 {
		t.Error("PredictedValue should be positive")
	}

	if forecast.Trend == "" {
		t.Error("Trend should not be empty")
	}

	if len(forecast.Recommendations) == 0 {
		t.Error("Recommendations should not be empty")
	}
}

func TestGenerateTrendForecastWithDifferentMetrics(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	metrics := []string{"requests", "success_rate", "latency_p99", "error_rate", "active_users"}

	for _, metric := range metrics {
		t.Run(metric, func(t *testing.T) {
			config := ForecastConfig{
				Metric:     metric,
				Horizon:    12 * time.Hour,
				Confidence: 0.90,
			}

			forecast, err := service.GenerateTrendForecast(ctx, metric, config)
			if err != nil {
				t.Fatalf("GenerateTrendForecast for %s failed: %v", metric, err)
			}

			if forecast.Metric != metric {
				t.Errorf("Expected metric '%s', got '%s'", metric, forecast.Metric)
			}
		})
	}
}

func TestGenerateTrendForecastSeasonality(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	config := ForecastConfig{
		Metric:      "requests",
		Horizon:     48 * time.Hour,
		Seasonality: true,
	}

	forecast, err := service.GenerateTrendForecast(ctx, "requests", config)
	if err != nil {
		t.Fatalf("GenerateTrendForecast failed: %v", err)
	}

	if forecast.Seasonality == nil {
		t.Error("Seasonality should not be nil when enabled")
	}

	if forecast.Seasonality["detected"] != true {
		t.Error("Seasonality should be detected")
	}

	_, hasHourly := forecast.Seasonality["hourly_pattern"]
	_, hasDaily := forecast.Seasonality["daily_pattern"]

	if !hasHourly || !hasDaily {
		t.Error("Seasonality should contain both hourly and daily patterns")
	}
}

func TestPerformAnomalyAttribution(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	config := AttributionConfig{
		AnomalyTime: time.Now().Add(-1 * time.Hour),
		Metric:      "requests",
		WindowSize:  24 * time.Hour,
		CausalDepth: 3,
	}

	attribution, err := service.PerformAnomalyAttribution(ctx, "anomaly_001", config)
	if err != nil {
		t.Fatalf("PerformAnomalyAttribution failed: %v", err)
	}

	if attribution.AnomalyID != "anomaly_001" {
		t.Errorf("Expected anomaly ID 'anomaly_001', got '%s'", attribution.AnomalyID)
	}

	if attribution.Metric != "requests" {
		t.Errorf("Expected metric 'requests', got '%s'", attribution.Metric)
	}

	if len(attribution.Contributing) == 0 {
		t.Error("Contributing factors should not be empty")
	}

	totalContribution := 0.0
	for _, factor := range attribution.Contributing {
		if factor.Factor == "" {
			t.Error("Factor name should not be empty")
		}
		if factor.Weight < 0 || factor.Weight > 1 {
			t.Errorf("Factor weight should be between 0 and 1, got %f", factor.Weight)
		}
		totalContribution += factor.Value * factor.Weight
	}

	if attribution.ImpactScore != totalContribution {
		t.Errorf("ImpactScore mismatch: expected %f, got %f", totalContribution, attribution.ImpactScore)
	}

	if attribution.Confidence < 0 || attribution.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", attribution.Confidence)
	}

	if attribution.Explanation == "" {
		t.Error("Explanation should not be empty")
	}

	if len(attribution.RecommendedActions) == 0 {
		t.Error("RecommendedActions should not be empty")
	}
}

func TestPerformAnomalyAttributionForErrorRate(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	config := AttributionConfig{
		AnomalyTime: time.Now().Add(-2 * time.Hour),
		Metric:      "error_rate",
		WindowSize:  12 * time.Hour,
		CausalDepth: 2,
	}

	attribution, err := service.PerformAnomalyAttribution(ctx, "anomaly_002", config)
	if err != nil {
		t.Fatalf("PerformAnomalyAttribution for error_rate failed: %v", err)
	}

	if attribution.Metric != "error_rate" {
		t.Errorf("Expected metric 'error_rate', got '%s'", attribution.Metric)
	}

	foundUpstream := false
	for _, factor := range attribution.Contributing {
		if factor.Factor == "upstream_errors" {
			foundUpstream = true
			break
		}
	}

	if !foundUpstream {
		t.Error("error_rate attribution should include upstream_errors factor")
	}
}

func TestGenerateNLReport(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	request := NLReportRequest{
		ReportType: "executive",
		TimeRange: TimeRange{
			Start: time.Now().Add(-24 * time.Hour),
			End:   time.Now(),
		},
		Metrics:    []string{"requests", "success_rate"},
		Dimensions: []string{"hour"},
		Filters:    map[string]interface{}{},
		Language:   "zh-CN",
		Format:     "full",
	}

	report, err := service.GenerateNLReport(ctx, request)
	if err != nil {
		t.Fatalf("GenerateNLReport failed: %v", err)
	}

	if report.ReportID == "" {
		t.Error("ReportID should not be empty")
	}

	if report.Title == "" {
		t.Error("Title should not be empty")
	}

	if report.Summary == "" {
		t.Error("Summary should not be empty")
	}

	if len(report.Sections) == 0 {
		t.Error("Sections should not be empty")
	}

	for i, section := range report.Sections {
		if section.Title == "" {
			t.Errorf("Section %d should have a title", i)
		}
		if section.Order != i+1 {
			t.Errorf("Section %d should have order %d, got %d", i, i+1, section.Order)
		}
	}

	if len(report.Charts) == 0 {
		t.Error("Charts should not be empty")
	}

	for _, chart := range report.Charts {
		if chart.ID == "" {
			t.Error("Chart ID should not be empty")
		}
		if chart.Type == "" {
			t.Error("Chart Type should not be empty")
		}
	}

	if len(report.Tables) == 0 {
		t.Error("Tables should not be empty")
	}

	for _, table := range report.Tables {
		if len(table.Headers) == 0 {
			t.Error("Table headers should not be empty")
		}
		if len(table.Rows) == 0 {
			t.Error("Table rows should not be empty")
		}
	}

	if len(report.Insights) == 0 {
		t.Error("Insights should not be empty")
	}

	if len(report.KeyMetrics) == 0 {
		t.Error("KeyMetrics should not be empty")
	}

	if len(report.Comparisons) == 0 {
		t.Error("Comparisons should not be empty")
	}

	if report.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should not be zero")
	}

	if report.ModelVersion == "" {
		t.Error("ModelVersion should not be empty")
	}
}

func TestGenerateNLReportWithDifferentTypes(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	reportTypes := []string{"executive", "operational", "security", "technical", "compliance"}

	for _, reportType := range reportTypes {
		t.Run(reportType, func(t *testing.T) {
			request := NLReportRequest{
				ReportType: reportType,
				TimeRange: TimeRange{
					Start: time.Now().Add(-24 * time.Hour),
					End:   time.Now(),
				},
			}

			report, err := service.GenerateNLReport(ctx, request)
			if err != nil {
				t.Fatalf("GenerateNLReport for type %s failed: %v", reportType, err)
			}

			expectedTitle := map[string]string{
				"executive":   "Executive Summary Report",
				"operational": "Operational Performance Report",
				"security":    "Security Analysis Report",
				"technical":   "Technical Deep-Dive Report",
				"compliance":  "Compliance Status Report",
			}

			if report.Title != expectedTitle[reportType] {
				t.Errorf("Expected title '%s', got '%s'", expectedTitle[reportType], report.Title)
			}
		})
	}
}

func TestProcessInteractiveQuery(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	query := InteractiveQuery{
		Query:  "Show requests trend for the last 7 days",
		Intent: QueryIntent{
			Type:         "trend",
			TargetMetrics: []string{"requests"},
			Aggregation:  "sum",
			Comparison:   false,
			DrillDown:    []string{"hour", "day"},
		},
		TimeRange: TimeRange{
			Start: time.Now().Add(-7 * 24 * time.Hour),
			End:   time.Now(),
		},
		Dimensions: []string{"day"},
	}

	result, err := service.ProcessInteractiveQuery(ctx, query)
	if err != nil {
		t.Fatalf("ProcessInteractiveQuery failed: %v", err)
	}

	if result.QueryID == "" {
		t.Error("QueryID should not be empty")
	}

	if len(result.Results) == 0 {
		t.Error("Results should not be empty")
	}

	if len(result.Visualizations) == 0 {
		t.Error("Visualizations should not be empty for trend query")
	}

	if result.Explanation == "" {
		t.Error("Explanation should not be empty")
	}

	if len(result.RelatedQueries) == 0 {
		t.Error("RelatedQueries should not be empty")
	}

	if len(result.Metadata) == 0 {
		t.Error("Metadata should not be empty")
	}
}

func TestProcessInteractiveQueryComparison(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	query := InteractiveQuery{
		Query:  "Compare success rate with last week",
		Intent: QueryIntent{
			Type:         "comparison",
			TargetMetrics: []string{"success_rate"},
			Comparison:   true,
		},
		TimeRange: TimeRange{
			Start: time.Now().Add(-7 * 24 * time.Hour),
			End:   time.Now(),
		},
	}

	result, err := service.ProcessInteractiveQuery(ctx, query)
	if err != nil {
		t.Fatalf("ProcessInteractiveQuery comparison failed: %v", err)
	}

	foundComparison := false
	for _, viz := range result.Visualizations {
		if viz.Type == "bar" {
			foundComparison = true
			break
		}
	}

	if !foundComparison {
		t.Error("Comparison query should produce bar chart visualization")
	}
}

func TestProcessInteractiveQueryDistribution(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	query := InteractiveQuery{
		Query:  "Show distribution by type",
		Intent: QueryIntent{
			Type:         "distribution",
			TargetMetrics: []string{"requests"},
		},
		TimeRange: TimeRange{
			Start: time.Now().Add(-24 * time.Hour),
			End:   time.Now(),
		},
	}

	result, err := service.ProcessInteractiveQuery(ctx, query)
	if err != nil {
		t.Fatalf("ProcessInteractiveQuery distribution failed: %v", err)
	}

	foundDistribution := false
	for _, viz := range result.Visualizations {
		if viz.Type == "pie" {
			foundDistribution = true
			break
		}
	}

	if !foundDistribution {
		t.Error("Distribution query should produce pie chart visualization")
	}
}

func TestCreateExplorationSession(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	userID := uint(123)

	session, err := service.CreateExplorationSession(ctx, userID)
	if err != nil {
		t.Fatalf("CreateExplorationSession failed: %v", err)
	}

	if session.SessionID == "" {
		t.Error("SessionID should not be empty")
	}

	if session.UserID != userID {
		t.Errorf("Expected UserID %d, got %d", userID, session.UserID)
	}

	if len(session.Queries) != 0 {
		t.Error("Initial Queries should be empty")
	}

	if len(session.Bookmarks) != 0 {
		t.Error("Initial Bookmarks should be empty")
	}

	if session.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	if session.LastActivity.IsZero() {
		t.Error("LastActivity should not be zero")
	}
}

func TestCreateBookmark(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	query := InteractiveQuery{
		Query: "Show peak hours analysis",
		Intent: QueryIntent{
			Type:         "trend",
			TargetMetrics: []string{"requests"},
		},
	}

	bookmark, err := service.CreateBookmark(ctx, "session_123", query, "Peak Hours Analysis")
	if err != nil {
		t.Fatalf("CreateBookmark failed: %v", err)
	}

	if bookmark.ID == "" {
		t.Error("Bookmark ID should not be empty")
	}

	if bookmark.Name != "Peak Hours Analysis" {
		t.Errorf("Expected name 'Peak Hours Analysis', got '%s'", bookmark.Name)
	}

	if bookmark.Query == nil {
		t.Error("Query should not be nil")
	}

	if len(bookmark.Annotations) != 0 {
		t.Error("Initial Annotations should be empty")
	}

	if bookmark.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestExportReport(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	testCases := []struct {
		format string
	}{
		{"json"},
		{"pdf"},
		{"csv"},
		{"html"},
		{"JSON"},
		{"PDF"},
	}

	for _, tc := range testCases {
		t.Run(tc.format, func(t *testing.T) {
			data, err := service.ExportReport(ctx, "report_001", tc.format)
			if err != nil {
				t.Fatalf("ExportReport failed for format %s: %v", tc.format, err)
			}

			if len(data) == 0 {
				t.Error("Exported data should not be empty")
			}
		})
	}
}

func TestForecastAccuracy(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	config := ForecastConfig{
		Metric:     "requests",
		Horizon:    24 * time.Hour,
		Confidence: 0.95,
	}

	forecast, err := service.GenerateTrendForecast(ctx, "requests", config)
	if err != nil {
		t.Fatalf("GenerateTrendForecast failed: %v", err)
	}

	for _, point := range forecast.ForecastPoints {
		if point.Value < 0 {
			t.Errorf("Forecast value should not be negative, got %f", point.Value)
		}

		if point.LowerBound < 0 {
			t.Errorf("Lower bound should not be negative, got %f", point.LowerBound)
		}

		if point.UpperBound < point.LowerBound {
			t.Errorf("Upper bound (%f) should be >= lower bound (%f)", point.UpperBound, point.LowerBound)
		}
	}
}

func TestAnomalyDetection(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	config := ForecastConfig{
		Metric:     "requests",
		Horizon:    48 * time.Hour,
		Confidence: 0.95,
	}

	forecast, err := service.GenerateTrendForecast(ctx, "requests", config)
	if err != nil {
		t.Fatalf("GenerateTrendForecast failed: %v", err)
	}

	anomalyCount := 0
	for _, point := range forecast.ForecastPoints {
		if point.IsAnomaly {
			anomalyCount++
		}
	}

	t.Logf("Detected %d anomalies out of %d forecast points", anomalyCount, len(forecast.ForecastPoints))
}

func TestAnomalyAttributionCausality(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	config := AttributionConfig{
		AnomalyTime: time.Now().Add(-1 * time.Hour),
		Metric:      "latency_p99",
		WindowSize:  24 * time.Hour,
		CausalDepth: 3,
	}

	attribution, err := service.PerformAnomalyAttribution(ctx, "anomaly_latency", config)
	if err != nil {
		t.Fatalf("PerformAnomalyAttribution failed: %v", err)
	}

	if len(attribution.Contributing) == 0 {
		t.Fatal("Should have contributing factors")
	}

	totalWeight := 0.0
	for _, factor := range attribution.Contributing {
		totalWeight += factor.Weight
	}

	if totalWeight != 1.0 {
		t.Errorf("Total weight should be 1.0, got %f", totalWeight)
	}

	if len(attribution.Correlated) > 0 {
		for _, corr := range attribution.Correlated {
			if corr.Correlation < -1 || corr.Correlation > 1 {
				t.Errorf("Correlation should be between -1 and 1, got %f", corr.Correlation)
			}
		}
	}
}

func TestReportConsistency(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	request := NLReportRequest{
		ReportType: "operational",
		TimeRange: TimeRange{
			Start: time.Now().Add(-48 * time.Hour),
			End:   time.Now(),
		},
		Metrics:    []string{"requests", "success_rate", "latency_p99"},
		Dimensions: []string{"hour"},
	}

	report, err := service.GenerateNLReport(ctx, request)
	if err != nil {
		t.Fatalf("GenerateNLReport failed: %v", err)
	}

	for _, section := range report.Sections {
		if section.Order < 1 {
			t.Errorf("Section order should be >= 1, got %d", section.Order)
		}
	}

	sectionOrders := make(map[int]bool)
	for _, section := range report.Sections {
		if sectionOrders[section.Order] {
			t.Errorf("Duplicate section order: %d", section.Order)
		}
		sectionOrders[section.Order] = true
	}

	for _, chart := range report.Charts {
		if chart.Data == nil {
			t.Errorf("Chart %s should have data", chart.ID)
		}
	}

	for _, table := range report.Tables {
		if len(table.Headers) != len(table.Rows[0]) {
			t.Errorf("Table %s: headers count (%d) should match row columns (%d)",
				table.ID, len(table.Headers), len(table.Rows[0]))
		}
	}
}

func TestQueryMetadataIntegrity(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	query := InteractiveQuery{
		Query: "Show recent metrics",
		Intent: QueryIntent{
			Type:          "trend",
			TargetMetrics: []string{"requests", "success_rate"},
		},
	}

	result, err := service.ProcessInteractiveQuery(ctx, query)
	if err != nil {
		t.Fatalf("ProcessInteractiveQuery failed: %v", err)
	}

	if result.Metadata["execution_time_ms"] == nil {
		t.Error("Metadata should include execution_time_ms")
	}

	if result.Metadata["data_points"] == nil {
		t.Error("Metadata should include data_points")
	}

	if result.Metadata["query_complexity"] == nil {
		t.Error("Metadata should include query_complexity")
	}
}

func TestReportGenerationPerformance(t *testing.T) {
	service := NewAIReportService()
	ctx := context.Background()

	request := NLReportRequest{
		ReportType: "comprehensive",
		TimeRange: TimeRange{
			Start: time.Now().Add(-30 * 24 * time.Hour),
			End:   time.Now(),
		},
		Metrics: []string{
			"requests", "success_rate", "latency_p50", "latency_p99",
			"error_rate", "blocked_rate", "active_users",
		},
		Dimensions: []string{"hour", "day", "week"},
	}

	start := time.Now()
	report, err := service.GenerateNLReport(ctx, request)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("GenerateNLReport failed: %v", err)
	}

	if elapsed > 5*time.Second {
		t.Logf("Warning: Report generation took %v, which is longer than expected", elapsed)
	}

	if len(report.Sections) == 0 {
		t.Error("Report should have sections")
	}

	if len(report.Charts) == 0 {
		t.Error("Report should have charts")
	}

	if len(report.Insights) == 0 {
		t.Error("Report should have insights")
	}
}

func TestEmptyContextHandling(t *testing.T) {
	service := NewAIReportService()

	config := ForecastConfig{
		Metric:    "requests",
		Horizon:   12 * time.Hour,
	}

	forecast, err := service.GenerateTrendForecast(context.Background(), "requests", config)
	if err != nil {
		t.Fatalf("GenerateTrendForecast with empty context failed: %v", err)
	}

	if forecast == nil {
		t.Fatal("Forecast should not be nil")
	}
}

func TestContributingFactorsNormalization(t *testing.T) {
	service := NewAIReportService()

	factors := service.identifyContributingFactors("requests", 24*time.Hour)

	totalContribution := 0.0
	for _, factor := range factors {
		totalContribution += factor.Value
	}

	if totalContribution == 0 {
		t.Error("Total contribution should not be zero")
	}
}

func TestSeasonalityAnalysis(t *testing.T) {
	service := NewAIReportService()

	seasonality := service.analyzeSeasonality("requests")

	if seasonality == nil {
		t.Fatal("Seasonality should not be nil")
	}

	if seasonality["detected"] != true {
		t.Error("Seasonality should be detected")
	}

	hourly, ok := seasonality["hourly_pattern"].([]float64)
	if !ok {
		t.Fatal("hourly_pattern should be []float64")
	}

	if len(hourly) != 24 {
		t.Errorf("hourly_pattern should have 24 elements, got %d", len(hourly))
	}

	for i, val := range hourly {
		if val <= 0 {
			t.Errorf("hourly_pattern[%d] should be positive, got %f", i, val)
		}
	}

	daily, ok := seasonality["daily_pattern"].([]float64)
	if !ok {
		t.Fatal("daily_pattern should be []float64")
	}

	if len(daily) != 7 {
		t.Errorf("daily_pattern should have 7 elements, got %d", len(daily))
	}
}

func TestTrendDetermination(t *testing.T) {
	service := NewAIReportService()

	metrics := []string{"requests", "success_rate", "latency_p99", "error_rate", "active_users"}

	for _, metric := range metrics {
		trend := service.determineTrend(metric)
		validTrends := map[string]bool{
			"increasing":   true,
			"decreasing":   true,
			"stable":       true,
			"fluctuating":   true,
		}

		if !validTrends[trend] {
			t.Errorf("Invalid trend '%s' for metric '%s'", trend, metric)
		}
	}
}

func TestChangePercentCalculation(t *testing.T) {
	service := NewAIReportService()

	metrics := []string{"requests", "success_rate", "latency_p99", "error_rate"}

	for _, metric := range metrics {
		change := service.calculateChangePercent(metric)

		t.Logf("Metric %s: change percent = %.2f%%", metric, change)
	}
}

func TestRecommendationGeneration(t *testing.T) {
	service := NewAIReportService()

	forecast := &TrendForecast{
		Metric:         "requests",
		CurrentValue:   100000,
		PredictedValue: 120000,
		Trend:          "increasing",
		ChangePercent:  20.0,
		Anomalies: []AnomalyPoint{
			{
				Timestamp:     time.Now().Add(-24 * time.Hour),
				Value:         150000,
				ExpectedValue: 100000,
				Deviation:     0.5,
				Severity:      "high",
			},
		},
	}

	recommendations := service.generateRecommendations(forecast)

	if len(recommendations) == 0 {
		t.Error("Should generate recommendations for significant change")
	}

	hasScalingRec := false
	for _, rec := range recommendations {
		if rec != "" {
			hasScalingRec = true
			break
		}
	}

	if !hasScalingRec {
		t.Error("Should have non-empty recommendations")
	}
}

func TestAttributionExplanationGeneration(t *testing.T) {
	service := NewAIReportService()

	attribution := &AnomalyAttribution{
		Metric: "requests",
		Contributing: []ContributionFactor{
			{Factor: "network_latency", Contribution: 0.35, Weight: 0.4},
			{Factor: "database_load", Contribution: 0.25, Weight: 0.3},
		},
		Correlated: []CorrelationInfo{
			{Type: "deployment", Description: "Recent deployment"},
		},
		ImpactScore: 0.235,
	}

	explanation := service.generateAttributionExplanation(attribution)

	if explanation == "" {
		t.Error("Explanation should not be empty")
	}

	if len(explanation) < 50 {
		t.Error("Explanation should be meaningful")
	}
}

func TestQueryResultExplanation(t *testing.T) {
	service := NewAIReportService()

	query := InteractiveQuery{
		Query: "Show requests trend",
		Intent: QueryIntent{
			Type:          "trend",
			TargetMetrics: []string{"requests"},
		},
	}

	result := &QueryResult{
		QueryID: "q_123",
		Results: map[string]interface{}{
			"count": 1000,
		},
	}

	explanation := service.explainQueryResult(query, result)

	if explanation == "" {
		t.Error("Explanation should not be empty")
	}
}

func TestRelatedQueriesSuggestion(t *testing.T) {
	service := NewAIReportService()

	query := InteractiveQuery{
		Query: "Show requests",
		Intent: QueryIntent{
			Type:          "trend",
			TargetMetrics: []string{"requests"},
		},
	}

	related := service.suggestRelatedQueries(query)

	if len(related) == 0 {
		t.Error("Should suggest related queries")
	}

	for _, q := range related {
		if q == "" {
			t.Error("Related query should not be empty")
		}
	}
}
