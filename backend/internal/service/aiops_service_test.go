package service

import (
	"context"
	"testing"
	"time"
)

func TestNewAIOpsService(t *testing.T) {
	service := NewAIOpsService()
	if service == nil {
		t.Fatal("NewAIOpsService returned nil")
	}
}

func TestDetectAnomalies(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	config := &AIOpsConfig{
		EnableRealTimeDetection:  true,
		EnableAutoLocalization:  true,
		SensitivityLevel:        "high",
	}

	metrics := []string{"requests", "success_rate", "latency_p99", "error_rate", "cpu_usage"}

	for _, metric := range metrics {
		t.Run(metric, func(t *testing.T) {
			result, err := service.DetectAnomalies(ctx, metric, config)
			if err != nil {
				t.Fatalf("DetectAnomalies failed for metric %s: %v", metric, err)
			}

			if result.AnomalyID == "" {
				t.Error("AnomalyID should not be empty")
			}

			if result.Timestamp.IsZero() {
				t.Error("Timestamp should not be zero")
			}

			if result.Metric != metric {
				t.Errorf("Expected metric '%s', got '%s'", metric, result.Metric)
			}

			if result.Score < 0 || result.Score > 1 {
				t.Errorf("Score should be between 0 and 1, got %f", result.Score)
			}

			if result.Confidence < 0 || result.Confidence > 1 {
				t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
			}

			if result.Description == "" {
				t.Error("Description should not be empty")
			}

			if result.RootCause == "" {
				t.Error("RootCause should not be empty")
			}

			if len(result.Recommendations) == 0 {
				t.Error("Recommendations should not be empty")
			}

			if result.Impact.Score < 0 || result.Impact.Score > 1 {
				t.Errorf("Impact Score should be between 0 and 1, got %f", result.Impact.Score)
			}

			if result.Impact.Level == "" {
				t.Error("Impact Level should not be empty")
			}
		})
	}
}

func TestDetectAnomaliesImpact(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	config := &AIOpsConfig{}

	result, err := service.DetectAnomalies(ctx, "success_rate", config)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	if result.Impact.Level != "high" {
		t.Errorf("Expected high impact for success_rate, got '%s'", result.Impact.Level)
	}

	if result.Impact.AffectedUsers <= 0 {
		t.Error("AffectedUsers should be positive")
	}
}

func TestDetectAnomaliesCorrelations(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	config := &AIOpsConfig{}

	result, err := service.DetectAnomalies(ctx, "requests", config)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	if len(result.Correlations) == 0 {
		t.Error("Correlations should not be empty")
	}

	for _, corr := range result.Correlations {
		if corr.Correlation < -1 || corr.Correlation > 1 {
			t.Errorf("Correlation should be between -1 and 1, got %f", corr.Correlation)
		}
	}
}

func TestDetectAnomaliesHistoricalSimilar(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	config := &AIOpsConfig{}

	result, err := service.DetectAnomalies(ctx, "error_rate", config)
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	if len(result.HistoricalSimilar) == 0 {
		t.Error("HistoricalSimilar should not be empty")
	}

	for _, similar := range result.HistoricalSimilar {
		if similar.Similarity < 0 || similar.Similarity > 1 {
			t.Errorf("Similarity should be between 0 and 1, got %f", similar.Similarity)
		}

		if similar.AnomalyID == "" {
			t.Error("AnomalyID should not be empty")
		}

		if similar.Resolution == "" {
			t.Error("Resolution should not be empty")
		}
	}
}

func TestLocalizeFault(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	config := &AIOpsConfig{
		EnableAutoLocalization: true,
	}

	symptoms := []Symptom{
		{
			Type:        "performance_degradation",
			Description: "High latency observed",
			Timestamp:   time.Now(),
			Severity:    "high",
			Entity:      "api-gateway",
		},
		{
			Type:        "error_spike",
			Description: "Error rate increased",
			Timestamp:   time.Now(),
			Severity:    "critical",
			Entity:      "core-api",
		},
	}

	result, err := service.LocalizeFault(ctx, symptoms, config)
	if err != nil {
		t.Fatalf("LocalizeFault failed: %v", err)
	}

	if result.FaultID == "" {
		t.Error("FaultID should not be empty")
	}

	if len(result.Symptoms) != len(symptoms) {
		t.Errorf("Expected %d symptoms, got %d", len(symptoms), len(result.Symptoms))
	}

	if len(result.Candidates) == 0 {
		t.Error("Candidates should not be empty")
	}

	for _, candidate := range result.Candidates {
		if candidate.Component == "" {
			t.Error("Candidate component should not be empty")
		}

		if candidate.Probability < 0 || candidate.Probability > 1 {
			t.Errorf("Probability should be between 0 and 1, got %f", candidate.Probability)
		}

		if len(candidate.Evidence) == 0 {
			t.Error("Evidence should not be empty")
		}

		for _, evidence := range candidate.Evidence {
			if evidence.Type == "" {
				t.Error("Evidence type should not be empty")
			}

			if evidence.Weight < 0 || evidence.Weight > 1 {
				t.Errorf("Evidence weight should be between 0 and 1, got %f", evidence.Weight)
			}
		}
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
	}

	if result.RootCause != nil {
		if result.RootCause.Probability <= 0.6 {
			t.Error("Root cause should have probability > 0.6")
		}

		if len(result.PropagationPath) == 0 {
			t.Error("PropagationPath should not be empty for root cause")
		}
	}
}

func TestLocalizeFaultPropagationPath(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	config := &AIOpsConfig{}

	symptoms := []Symptom{
		{
			Type:        "latency",
			Description: "High latency",
			Timestamp:   time.Now(),
			Severity:    "high",
			Entity:      "api-gateway",
		},
	}

	result, err := service.LocalizeFault(ctx, symptoms, config)
	if err != nil {
		t.Fatalf("LocalizeFault failed: %v", err)
	}

	if len(result.PropagationPath) == 0 {
		t.Error("PropagationPath should not be empty")
	}

	for i, node := range result.PropagationPath {
		if node == "" {
			t.Errorf("Node %d in propagation path should not be empty", i)
		}
	}
}

func TestPredictMaintenance(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	config := &AIOpsConfig{
		EnablePredictiveMaintenance: true,
	}

	components := []string{"database", "cache", "load_balancer", "api_gateway"}

	for _, component := range components {
		t.Run(component, func(t *testing.T) {
			result, err := service.PredictMaintenance(ctx, component, config)
			if err != nil {
				t.Fatalf("PredictMaintenance failed for component %s: %v", component, err)
			}

			if result.PredictionID == "" {
				t.Error("PredictionID should not be empty")
			}

			if result.Component != component {
				t.Errorf("Expected component '%s', got '%s'", component, result.Component)
			}

			if result.Probability < 0 || result.Probability > 1 {
				t.Errorf("Probability should be between 0 and 1, got %f", result.Probability)
			}

			if result.TimeToFailure <= 0 {
				t.Error("TimeToFailure should be positive")
			}

			if result.Confidence < 0 || result.Confidence > 1 {
				t.Errorf("Confidence should be between 0 and 1, got %f", result.Confidence)
			}

			if result.RiskLevel == "" {
				t.Error("RiskLevel should not be empty")
			}

			validRiskLevels := map[string]bool{
				"low":      true,
				"medium":   true,
				"high":     true,
				"critical": true,
			}

			if !validRiskLevels[result.RiskLevel] {
				t.Errorf("Invalid RiskLevel '%s'", result.RiskLevel)
			}

			if len(result.Indicators) == 0 {
				t.Error("Indicators should not be empty")
			}

			for _, indicator := range result.Indicators {
				if indicator.Name == "" {
					t.Error("Indicator Name should not be empty")
				}

				if indicator.Status == "" {
					t.Error("Indicator Status should not be empty")
				}

				validStatuses := map[string]bool{
					"normal": true,
					"warning": true,
					"critical": true,
				}

				if !validStatuses[indicator.Status] {
					t.Errorf("Invalid Indicator Status '%s'", indicator.Status)
				}
			}

			if len(result.RecommendedActions) == 0 {
				t.Error("RecommendedActions should not be empty")
			}

			if result.MaintenanceWindow.EarliestStart.IsZero() {
				t.Error("MaintenanceWindow.EarliestStart should not be zero")
			}

			if result.MaintenanceWindow.LatestEnd.IsZero() {
				t.Error("MaintenanceWindow.LatestEnd should not be zero")
			}

			if !result.MaintenanceWindow.LatestEnd.After(result.MaintenanceWindow.EarliestStart) {
				t.Error("LatestEnd should be after EarliestStart")
			}
		})
	}
}

func TestQueryKnowledgeGraph(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	query := KnowledgeGraphQuery{
		QueryType:     "dependencies",
		Entities:      []string{"service", "database"},
		Relationships: []string{"depends_on", "connects_to"},
		Depth:         2,
		Filters:       map[string]interface{}{"status": "healthy"},
	}

	result, err := service.QueryKnowledgeGraph(ctx, query)
	if err != nil {
		t.Fatalf("QueryKnowledgeGraph failed: %v", err)
	}

	if result.QueryID == "" {
		t.Error("QueryID should not be empty")
	}

	if len(result.Nodes) == 0 {
		t.Error("Nodes should not be empty")
	}

	for _, node := range result.Nodes {
		if node.ID == "" {
			t.Error("Node ID should not be empty")
		}

		if node.Type == "" {
			t.Error("Node Type should not be empty")
		}

		if node.Name == "" {
			t.Error("Node Name should not be empty")
		}
	}

	if len(result.Edges) == 0 {
		t.Error("Edges should not be empty")
	}

	for _, edge := range result.Edges {
		if edge.Source == "" {
			t.Error("Edge Source should not be empty")
		}

		if edge.Target == "" {
			t.Error("Edge Target should not be empty")
		}

		if edge.Relationship == "" {
			t.Error("Edge Relationship should not be empty")
		}

		if edge.Weight < 0 || edge.Weight > 1 {
			t.Errorf("Edge Weight should be between 0 and 1, got %f", edge.Weight)
		}
	}

	for _, path := range result.Paths {
		if len(path.Nodes) < 2 {
			t.Error("Path should have at least 2 nodes")
		}

		if path.TotalWeight < 0 {
			t.Error("TotalWeight should not be negative")
		}
	}

	if result.Summary == "" {
		t.Error("Summary should not be empty")
	}

	if result.Metadata == nil {
		t.Error("Metadata should not be nil")
	}

	if nodeCount, ok := result.Metadata["node_count"].(int); !ok || nodeCount != len(result.Nodes) {
		t.Error("Metadata node_count should match Nodes length")
	}
}

func TestGetIncidentContext(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	incidentID := "INC-001"

	context, err := service.GetIncidentContext(ctx, incidentID)
	if err != nil {
		t.Fatalf("GetIncidentContext failed: %v", err)
	}

	if context.IncidentID != incidentID {
		t.Errorf("Expected IncidentID '%s', got '%s'", incidentID, context.IncidentID)
	}

	if len(context.Timeline) == 0 {
		t.Error("Timeline should not be empty")
	}

	for i, event := range context.Timeline {
		if event.Timestamp.IsZero() {
			t.Errorf("Timeline event %d: Timestamp should not be zero", i)
		}

		if event.Type == "" {
			t.Errorf("Timeline event %d: Type should not be empty", i)
		}

		if event.Description == "" {
			t.Errorf("Timeline event %d: Description should not be empty", i)
		}
	}

	if len(context.RelatedChanges) == 0 {
		t.Error("RelatedChanges should not be empty")
	}

	if len(context.RelatedAlerts) == 0 {
		t.Error("RelatedAlerts should not be empty")
	}

	if len(context.Metrics) == 0 {
		t.Error("Metrics should not be empty")
	}

	if len(context.Logs) == 0 {
		t.Error("Logs should not be empty")
	}
}

func TestAutoResolveIncident(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	incidentID := "INC-001"

	result, err := service.AutoResolveIncident(ctx, incidentID)
	if err != nil {
		t.Fatalf("AutoResolveIncident failed: %v", err)
	}

	if result.IncidentID != incidentID {
		t.Errorf("Expected IncidentID '%s', got '%s'", incidentID, result.IncidentID)
	}

	if result.ResolutionID == "" {
		t.Error("ResolutionID should not be empty")
	}

	if len(result.Actions) == 0 {
		t.Error("Actions should not be empty")
	}

	for _, action := range result.Actions {
		if action.ActionID == "" {
			t.Error("Action ActionID should not be empty")
		}

		if action.Type == "" {
			t.Error("Action Type should not be empty")
		}

		if action.Description == "" {
			t.Error("Action Description should not be empty")
		}

		if action.Target == "" {
			t.Error("Action Target should not be empty")
		}

		validRiskLevels := map[string]bool{
			"low":    true,
			"medium": true,
			"high":   true,
		}

		if !validRiskLevels[action.RiskLevel] {
			t.Errorf("Invalid RiskLevel '%s'", action.RiskLevel)
		}

		if action.EstimatedDuration <= 0 {
			t.Error("EstimatedDuration should be positive")
		}
	}
}

func TestDetermineSeverity(t *testing.T) {
	service := NewAIOpsService(nil)

	tests := []struct {
		metric     string
		expected   string
	}{
		{"requests", "high"},
		{"success_rate", "critical"},
		{"latency_p99", "medium"},
		{"error_rate", "high"},
		{"cpu_usage", "medium"},
		{"disk_usage", "low"},
		{"unknown_metric", "medium"},
	}

	for _, tt := range tests {
		t.Run(tt.metric, func(t *testing.T) {
			severity := service.determineSeverity(tt.metric)
			if severity != tt.expected {
				t.Errorf("Expected severity '%s' for metric '%s', got '%s'", tt.expected, tt.metric, severity)
			}
		})
	}
}

func TestCalculateAnomalyScore(t *testing.T) {
	service := NewAIOpsService(nil)

	metrics := []string{"requests", "success_rate", "latency_p99", "error_rate"}

	for _, metric := range metrics {
		t.Run(metric, func(t *testing.T) {
			score := service.calculateAnomalyScore(metric)

			if score < 0 || score > 1 {
				t.Errorf("Score should be between 0 and 1, got %f", score)
			}
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	service := NewAIOpsService(nil)

	metrics := []string{"requests", "success_rate", "latency_p99", "error_rate"}

	for _, metric := range metrics {
		t.Run(metric, func(t *testing.T) {
			recs := service.generateRecommendations(metric)

			if len(recs) == 0 {
				t.Error("Recommendations should not be empty")
			}

			for _, rec := range recs {
				if rec == "" {
					t.Error("Recommendation should not be empty")
				}
			}
		})
	}
}

func TestIdentifyAffectedEntities(t *testing.T) {
	service := NewAIOpsService(nil)

	metrics := []string{"requests", "success_rate", "latency_p99"}

	for _, metric := range metrics {
		t.Run(metric, func(t *testing.T) {
			entities := service.identifyAffectedEntities(metric)

			if len(entities) == 0 {
				t.Error("Affected entities should not be empty")
			}

			for _, entity := range entities {
				if entity == "" {
					t.Error("Entity should not be empty")
				}
			}
		})
	}
}

func TestAssessImpact(t *testing.T) {
	service := NewAIOpsService(nil)

	tests := []struct {
		metric         string
		expectedLevel  string
		minScore       float64
	}{
		{"success_rate", "high", 0.8},
		{"error_rate", "high", 0.7},
		{"latency_p99", "medium", 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.metric, func(t *testing.T) {
			impact := service.assessImpact(tt.metric)

			if impact.Level != tt.expectedLevel {
				t.Errorf("Expected level '%s' for metric '%s', got '%s'", tt.expectedLevel, tt.metric, impact.Level)
			}

			if impact.Score < tt.minScore {
				t.Errorf("Expected score >= %f for metric '%s', got %f", tt.minScore, tt.metric, impact.Score)
			}

			if impact.AffectedUsers <= 0 {
				t.Error("AffectedUsers should be positive")
			}

			if impact.Description == "" {
				t.Error("Description should not be empty")
			}
		})
	}
}

func TestIdentifyFaultCandidates(t *testing.T) {
	service := NewAIOpsService(nil)

	symptoms := []Symptom{
		{
			Type:        "latency",
			Description: "High latency",
			Timestamp:   time.Now(),
			Severity:    "high",
			Entity:      "api",
		},
	}

	candidates := service.identifyFaultCandidates(symptoms)

	if len(candidates) == 0 {
		t.Error("Candidates should not be empty")
	}

	totalProbability := 0.0
	for _, candidate := range candidates {
		totalProbability += candidate.Probability

		if len(candidate.SupportingMetrics) == 0 {
			t.Error("SupportingMetrics should not be empty")
		}
	}

	if totalProbability > 1.5 {
		t.Errorf("Total probability seems too high: %f", totalProbability)
	}
}

func TestGetEvidence(t *testing.T) {
	service := NewAIOpsService(nil)

	dbEvidence := service.getDatabaseEvidence()
	if len(dbEvidence) == 0 {
		t.Error("Database evidence should not be empty")
	}

	cacheEvidence := service.getCacheEvidence()
	if len(cacheEvidence) == 0 {
		t.Error("Cache evidence should not be empty")
	}

	networkEvidence := service.getNetworkEvidence()
	if len(networkEvidence) == 0 {
		t.Error("Network evidence should not be empty")
	}
}

func TestTracePropagationPath(t *testing.T) {
	service := NewAIOpsService(nil)

	components := []string{"database", "cache", "network"}

	for _, component := range components {
		t.Run(component, func(t *testing.T) {
			candidate := FaultCandidate{Component: component}
			path := service.tracePropagationPath(candidate)

			if len(path) == 0 {
				t.Error("Propagation path should not be empty")
			}

			if path[0] != component {
				t.Errorf("First node in path should be '%s'", component)
			}
		})
	}
}

func TestGenerateFaultActions(t *testing.T) {
	service := NewAIOpsService(nil)

	result := &FaultLocalizationResult{
		Candidates: []FaultCandidate{
			{Component: "database", Probability: 0.75},
		},
		RootCause: &FaultCandidate{Component: "database", Probability: 0.8},
	}

	actions := service.generateFaultActions(result)

	if len(actions) == 0 {
		t.Error("Actions should not be empty")
	}

	for _, action := range actions {
		if action == "" {
			t.Error("Action should not be empty")
		}
	}
}

func TestEstimateResolutionTime(t *testing.T) {
	service := NewAIOpsService(nil)

	tests := []struct {
		component   string
		minDuration time.Duration
	}{
		{"database", 30 * time.Minute},
		{"cache", 10 * time.Minute},
		{"network", 30 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.component, func(t *testing.T) {
			result := &FaultLocalizationResult{
				RootCause: &FaultCandidate{Component: tt.component, Probability: 0.8},
			}

			duration := service.estimateResolutionTime(result)

			if duration < tt.minDuration {
				t.Errorf("Expected duration >= %v for component '%s', got %v", tt.minDuration, tt.component, duration)
			}
		})
	}
}

func TestAnalyzeMaintenanceIndicators(t *testing.T) {
	service := NewAIOpsService(nil)

	components := []string{"database", "cache", "load_balancer", "api_gateway"}

	for _, component := range components {
		t.Run(component, func(t *testing.T) {
			indicators := service.analyzeMaintenanceIndicators(component)

			if len(indicators) == 0 {
				t.Error("Indicators should not be empty")
			}

			for _, indicator := range indicators {
				if indicator.Name == "" {
					t.Error("Indicator name should not be empty")
				}

				if indicator.Unit == "" {
					t.Error("Indicator unit should not be empty")
				}
			}
		})
	}
}

func TestGenerateMaintenanceActions(t *testing.T) {
	service := NewAIOpsService(nil)

	prediction := &PredictiveMaintenanceResult{
		Component:  "database",
		RiskLevel: "high",
	}

	actions := service.generateMaintenanceActions(prediction)

	if len(actions) == 0 {
		t.Error("Actions should not be empty")
	}
}

func TestCalculateMaintenanceWindow(t *testing.T) {
	service := NewAIOpsService(nil)

	tests := []struct {
		riskLevel      string
		maxStartDelay  time.Duration
	}{
		{"critical", 24 * time.Hour},
		{"high", 72 * time.Hour},
		{"medium", 168 * time.Hour},
		{"low", 168 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.riskLevel, func(t *testing.T) {
			prediction := &PredictiveMaintenanceResult{
				RiskLevel: tt.riskLevel,
			}

			window := service.calculateMaintenanceWindow(prediction)

			delay := window.EarliestStart.Sub(time.Now())
			if delay > tt.maxStartDelay {
				t.Errorf("Expected start delay <= %v for risk '%s', got %v", tt.maxStartDelay, tt.riskLevel, delay)
			}

			if window.Duration <= 0 {
				t.Error("Duration should be positive")
			}
		})
	}
}

func TestBuildIncidentTimeline(t *testing.T) {
	service := NewAIOpsService(nil)

	events := service.buildIncidentTimeline("INC-001")

	if len(events) == 0 {
		t.Error("Timeline events should not be empty")
	}

	for i := 1; i < len(events); i++ {
		if events[i].Timestamp.Before(events[i-1].Timestamp) {
			t.Error("Timeline events should be in chronological order")
		}
	}
}

func TestFindRelatedChanges(t *testing.T) {
	service := NewAIOpsService(nil)

	changes := service.findRelatedChanges("INC-001")

	if len(changes) == 0 {
		t.Error("Related changes should not be empty")
	}

	for _, change := range changes {
		if change.ChangeID == "" {
			t.Error("ChangeID should not be empty")
		}

		if change.Type == "" {
			t.Error("Type should not be empty")
		}
	}
}

func TestCaptureMetricSnapshots(t *testing.T) {
	service := NewAIOpsService(nil)

	metrics := service.captureMetricSnapshots("INC-001")

	if len(metrics) == 0 {
		t.Error("Metric snapshots should not be empty")
	}

	for _, metric := range metrics {
		if metric.Metric == "" {
			t.Error("Metric name should not be empty")
		}

		if metric.Unit == "" {
			t.Error("Unit should not be empty")
		}

		if metric.Timestamp.IsZero() {
			t.Error("Timestamp should not be zero")
		}
	}
}

func TestRetrieveLogSnippets(t *testing.T) {
	service := NewAIOpsService(nil)

	logs := service.retrieveLogSnippets("INC-001")

	if len(logs) == 0 {
		t.Error("Log snippets should not be empty")
	}

	validLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
	}

	for _, log := range logs {
		if log.Level == "" {
			t.Error("Log level should not be empty")
		}

		if !validLevels[log.Level] {
			t.Errorf("Invalid log level '%s'", log.Level)
		}

		if log.Message == "" {
			t.Error("Log message should not be empty")
		}
	}
}

func TestSuggestResolutionActions(t *testing.T) {
	service := NewAIOpsService(nil)

	actions := service.suggestResolutionActions("INC-001")

	if len(actions) == 0 {
		t.Error("Resolution actions should not be empty")
	}

	for _, action := range actions {
		if action.Type == "" {
			t.Error("Action type should not be empty")
		}

		if action.Parameters == nil {
			t.Error("Parameters should not be nil")
		}
	}
}

func TestGenerateRollbackPlan(t *testing.T) {
	service := NewAIOpsService(nil)

	actions := []ResolutionAction{
		{
			Type:        "rollback",
			Description: "Rollback database configuration",
			Target:      "database",
		},
		{
			Type:        "scale",
			Description: "Scale up service",
			Target:      "service",
		},
	}

	plan := service.generateRollbackPlan(actions)

	if plan == "" {
		t.Error("Rollback plan should not be empty")
	}
}

func TestGenerateShortID(t *testing.T) {
	service := NewAIOpsService(nil)

	id1 := service.generateShortID()
	id2 := service.generateShortID()

	if len(id1) != 8 {
		t.Errorf("Expected ID length 8, got %d", len(id1))
	}

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}
}

func TestExportAIOpsReport(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	formats := []string{"json", "pdf", "csv"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			data, err := service.ExportAIOpsReport(ctx, format)
			if err != nil {
				t.Errorf("ExportAIOpsReport failed for format %s: %v", format, err)
			}

			if len(data) == 0 {
				t.Error("Exported data should not be empty")
			}
		})
	}
}

func TestGetAIOpsDashboard(t *testing.T) {
	service := NewAIOpsService()
	ctx := context.Background()

	dashboard, err := service.GetAIOpsDashboard(ctx)
	if err != nil {
		t.Fatalf("GetAIOpsDashboard failed: %v", err)
	}

	if dashboard.ActiveIncidents < 0 {
		t.Error("ActiveIncidents should not be negative")
	}

	if dashboard.AnomaliesDetected < 0 {
		t.Error("AnomaliesDetected should not be negative")
	}

	if dashboard.HealthScore < 0 || dashboard.HealthScore > 100 {
		t.Errorf("HealthScore should be between 0 and 100, got %f", dashboard.HealthScore)
	}

	if len(dashboard.Metrics) == 0 {
		t.Error("Metrics should not be empty")
	}

	for _, metric := range dashboard.Metrics {
		if metric.Name == "" {
			t.Error("Metric name should not be empty")
		}
	}

	if len(dashboard.RecentEvents) == 0 {
		t.Error("RecentEvents should not be empty")
	}

	if len(dashboard.TopIssues) == 0 {
		t.Error("TopIssues should not be empty")
	}
}

func TestAIOpsDashboard(t *testing.T) {
	dashboard := &AIOpsDashboard{
		ActiveIncidents:  2,
		AnomaliesDetected: 5,
		Predictions:       10,
		HealthScore:       92.5,
		Metrics:           []MetricData{},
		RecentEvents:      []EventData{},
		TopIssues:         []IssueData{},
	}

	if dashboard.ActiveIncidents != 2 {
		t.Errorf("Expected ActiveIncidents 2, got %d", dashboard.ActiveIncidents)
	}

	if dashboard.HealthScore != 92.5 {
		t.Errorf("Expected HealthScore 92.5, got %f", dashboard.HealthScore)
	}
}

func TestMetricData(t *testing.T) {
	metric := MetricData{
		Name:   "Success Rate",
		Value:  99.5,
		Unit:   "%",
		Change: 0.5,
		Trend:  "up",
	}

	if metric.Name == "" {
		t.Error("Name should not be empty")
	}

	if metric.Value <= 0 {
		t.Error("Value should be positive")
	}

	validTrends := map[string]bool{"up": true, "down": true, "stable": true}
	if !validTrends[metric.Trend] {
		t.Errorf("Invalid trend '%s'", metric.Trend)
	}
}

func TestEventData(t *testing.T) {
	event := EventData{
		Timestamp:   time.Now(),
		Type:        "incident",
		Description: "Error rate spike detected",
		Severity:    "critical",
	}

	if event.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	if event.Type == "" {
		t.Error("Type should not be empty")
	}

	if event.Description == "" {
		t.Error("Description should not be empty")
	}
}

func TestIssueData(t *testing.T) {
	issue := IssueData{
		ID:            "issue_001",
		Title:         "Database connection pool exhausted",
		Priority:      "high",
		AffectedCount: 1000,
		Status:        "investigating",
	}

	if issue.ID == "" {
		t.Error("ID should not be empty")
	}

	if issue.Title == "" {
		t.Error("Title should not be empty")
	}

	validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	if !validPriorities[issue.Priority] {
		t.Errorf("Invalid priority '%s'", issue.Priority)
	}
}
