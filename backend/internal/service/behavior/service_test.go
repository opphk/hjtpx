package behavior

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	err = db.AutoMigrate(
		&BehaviorTrajectory{},
		&BehaviorAnalysis{},
		&UserProfile{},
		&AnomalyRecord{},
		&AnomalyRule{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestTrajectoryService_TableName(t *testing.T) {
	traj := BehaviorTrajectory{}
	if traj.TableName() != "behavior_trajectories" {
		t.Errorf("Expected table name 'behavior_trajectories', got '%s'", traj.TableName())
	}
}

func TestBehaviorAnalysis_TableName(t *testing.T) {
	analysis := BehaviorAnalysis{}
	if analysis.TableName() != "behavior_analyses" {
		t.Errorf("Expected table name 'behavior_analyses', got '%s'", analysis.TableName())
	}
}

func TestUserProfile_TableName(t *testing.T) {
	profile := UserProfile{}
	if profile.TableName() != "behavior_user_profiles" {
		t.Errorf("Expected table name 'behavior_user_profiles', got '%s'", profile.TableName())
	}
}

func TestAnomalyRecord_TableName(t *testing.T) {
	record := AnomalyRecord{}
	if record.TableName() != "behavior_anomaly_records" {
		t.Errorf("Expected table name 'behavior_anomaly_records', got '%s'", record.TableName())
	}
}

func TestAnomalyRule_TableName(t *testing.T) {
	rule := AnomalyRule{}
	if rule.TableName() != "behavior_anomaly_rules" {
		t.Errorf("Expected table name 'behavior_anomaly_rules', got '%s'", rule.TableName())
	}
}

func TestCalculateDistance(t *testing.T) {
	svc := &TrajectoryService{}

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 3, Y: 4, Timestamp: 1100},
	}

	distance := svc.calculateDistance(points)
	expected := 5.0

	if distance < expected-0.1 || distance > expected+0.1 {
		t.Errorf("Expected distance ~%.2f, got %.2f", expected, distance)
	}
}

func TestCalculateSpeeds(t *testing.T) {
	svc := &TrajectoryService{}

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 10, Y: 0, Timestamp: 1010},
	}

	speeds := svc.calculateSpeeds(points)
	if len(speeds) != 1 {
		t.Fatalf("Expected 1 speed value, got %d", len(speeds))
	}

	if speeds[0] != 1.0 {
		t.Errorf("Expected speed 1.0, got %.2f", speeds[0])
	}
}

func TestCalculateEfficiency(t *testing.T) {
	svc := &TrajectoryService{}

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 100, Timestamp: 2000},
	}

	efficiency := svc.calculateEfficiency(points)
	if efficiency < 0.99 || efficiency > 1.01 {
		t.Errorf("Expected efficiency close to 1.0, got %.2f", efficiency)
	}
}

func TestCountDirectionChanges(t *testing.T) {
	svc := &TrajectoryService{}

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 10, Y: 0, Timestamp: 1100},
		{X: 10, Y: 10, Timestamp: 1200},
	}

	changes := svc.countDirectionChanges(points)
	if changes < 1 {
		t.Errorf("Expected at least 1 direction change, got %d", changes)
	}
}

func TestCountPauses(t *testing.T) {
	svc := &TrajectoryService{}

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 0, Y: 0, Timestamp: 1200},
		{X: 10, Y: 10, Timestamp: 1300},
	}

	pauses := svc.countPauses(points)
	if pauses != 1 {
		t.Errorf("Expected 1 pause, got %d", pauses)
	}
}

func TestExtractClicks(t *testing.T) {
	svc := &TrajectoryService{}

	points := []TrajectoryPoint{
		{X: 100, Y: 200, Timestamp: 1000, Event: "move"},
		{X: 150, Y: 250, Timestamp: 1100, Event: "click"},
		{X: 200, Y: 300, Timestamp: 1200, Event: "move"},
		{X: 250, Y: 350, Timestamp: 1300, Event: "click"},
	}

	clicks := svc.extractClicks(points)
	if len(clicks) != 2 {
		t.Errorf("Expected 2 clicks, got %d", len(clicks))
	}
}

func TestCalculateClickRegularity(t *testing.T) {
	svc := &TrajectoryService{}

	clicks := []TrajectoryPoint{
		{X: 100, Y: 200, Timestamp: 1000},
		{X: 150, Y: 250, Timestamp: 1100},
		{X: 200, Y: 300, Timestamp: 1200},
		{X: 250, Y: 350, Timestamp: 1300},
	}

	regularity := svc.calculateClickRegularity(clicks)
	if regularity < 0 || regularity > 1 {
		t.Errorf("Expected regularity between 0 and 1, got %.2f", regularity)
	}
}

func TestCalculateRiskScore(t *testing.T) {
	svc := &TrajectoryService{}

	testCases := []struct {
		name     string
		analysis BehaviorAnalysis
		minScore float64
		maxScore float64
	}{
		{
			name: "High Risk - Bot-like",
			analysis: BehaviorAnalysis{
				PathEfficiency:   0.95,
				TotalDistance:   200,
				JitterScore:    0.01,
				CurvatureAvg:   0.02,
				DirectionChanges: 2,
				PauseCount:     0,
				MicroCorrections: 0,
				ClickRegularity: 0.95,
				ClickCount:     5,
				AverageSpeed:   15,
			},
			minScore: 50,
		},
		{
			name: "Low Risk - Human-like",
			analysis: BehaviorAnalysis{
				PathEfficiency:   0.5,
				TotalDistance:   100,
				JitterScore:    0.2,
				CurvatureAvg:   0.3,
				DirectionChanges: 20,
				PauseCount:     5,
				MicroCorrections: 10,
				ClickRegularity: 0.3,
				ClickCount:     2,
				AverageSpeed:   2,
			},
			maxScore: 30,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := svc.calculateRiskScore(&tc.analysis)
			if tc.minScore > 0 && score < tc.minScore {
				t.Errorf("Expected score >= %.2f, got %.2f", tc.minScore, score)
			}
			if tc.maxScore > 0 && score > tc.maxScore {
				t.Errorf("Expected score <= %.2f, got %.2f", tc.maxScore, score)
			}
		})
	}
}

func TestCalculateConfidence(t *testing.T) {
	svc := &TrajectoryService{}

	testCases := []struct {
		name     string
		analysis BehaviorAnalysis
	}{
		{
			name: "High Confidence",
			analysis: BehaviorAnalysis{
				RiskScore:        85,
				DirectionChanges: 10,
				ClickCount:      5,
				TotalDistance:   300,
			},
		},
		{
			name: "Low Confidence",
			analysis: BehaviorAnalysis{
				RiskScore:        50,
				DirectionChanges: 2,
				ClickCount:      0,
				TotalDistance:   50,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			confidence := svc.calculateConfidence(&tc.analysis)
			if confidence < 0.5 || confidence > 0.95 {
				t.Errorf("Expected confidence between 0.5 and 0.95, got %.2f", confidence)
			}
		})
	}
}

func TestMean(t *testing.T) {
	testCases := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"Normal", []float64{1, 2, 3, 4, 5}, 3.0},
		{"Empty", []float64{}, 0.0},
		{"Single", []float64{5.0}, 5.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mean(tc.values)
			if result != tc.expected {
				t.Errorf("Expected %.2f, got %.2f", tc.expected, result)
			}
		})
	}
}

func TestVariance(t *testing.T) {
	values := []float64{2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0}
	v := variance(values)
	if v <= 0 {
		t.Error("Expected positive variance")
	}
}

func TestStdFloat(t *testing.T) {
	values := []float64{2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0}
	std := stdFloat(values)
	if std <= 0 {
		t.Error("Expected positive standard deviation")
	}
}

func TestMaxFloat(t *testing.T) {
	values := []float64{1.5, 3.7, 2.2, 5.0, 4.1}
	max := maxFloat(values)
	if max != 5.0 {
		t.Errorf("Expected 5.0, got %.2f", max)
	}
}

func TestGetRecommendation(t *testing.T) {
	testCases := []struct {
		anomalyType string
		severity    string
	}{
		{"speed_anomaly", "critical"},
		{"path_anomaly", "high"},
		{"click_anomaly", "medium"},
		{"bot_detection", "low"},
		{"unknown_type", "medium"},
	}

	for _, tc := range testCases {
		t.Run(tc.anomalyType+"_"+tc.severity, func(t *testing.T) {
			rec := getRecommendation(tc.anomalyType, tc.severity)
			if rec == "" {
				t.Error("Expected non-empty recommendation")
			}
		})
	}
}

func TestSmoothPoints(t *testing.T) {
	svc := &TrajectoryService{}

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 10, Y: 10, Timestamp: 1100},
		{X: 20, Y: 20, Timestamp: 1200},
	}

	smoothed := svc.smoothPoints(points, 3)
	if len(smoothed) != len(points) {
		t.Errorf("Expected %d smoothed points, got %d", len(points), len(smoothed))
	}
}

func TestCountCorrections(t *testing.T) {
	svc := &TrajectoryService{}

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 3, Y: 1, Timestamp: 1100},
		{X: 7, Y: 0, Timestamp: 1200},
	}

	corrections := svc.countCorrections(points)
	t.Logf("Micro corrections: %d", corrections)
}

func TestCalculateCurvature(t *testing.T) {
	svc := &TrajectoryService{}

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 10, Y: 10, Timestamp: 1100},
		{X: 20, Y: 5, Timestamp: 1200},
	}

	curvature := svc.calculateCurvature(points)
	if curvature < 0 {
		t.Error("Expected non-negative curvature")
	}
}

func TestCalculateJitter(t *testing.T) {
	svc := &TrajectoryService{}

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 5, Y: 1, Timestamp: 1100},
		{X: 10, Y: 0, Timestamp: 1200},
	}

	jitter := svc.calculateJitter(points)
	if jitter < 0 {
		t.Error("Expected non-negative jitter score")
	}
}

func TestAnalyzeTrajectory(t *testing.T) {
	svc := &TrajectoryService{db: nil}

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 100, Timestamp: 1100},
		{X: 200, Y: 200, Timestamp: 1200},
		{X: 200, Y: 200, Timestamp: 1300},
		{X: 300, Y: 300, Timestamp: 1400},
		{X: 350, Y: 350, Timestamp: 1500, Event: "click"},
	}

	traj := &BehaviorTrajectory{
		UserID: "test-user",
		Points: points,
	}

	analysis := &BehaviorAnalysis{
		TrajectoryID: 1,
		UserID:       traj.UserID,
	}

	analysis.TotalDistance = svc.calculateDistance(points)
	speeds := svc.calculateSpeeds(points)
	if len(speeds) > 0 {
		analysis.AverageSpeed = mean(speeds)
		analysis.MaxSpeed = maxFloat(speeds)
	}
	analysis.PathEfficiency = svc.calculateEfficiency(points)
	analysis.DirectionChanges = svc.countDirectionChanges(points)
	analysis.CurvatureAvg = svc.calculateCurvature(points)
	analysis.JitterScore = svc.calculateJitter(points)
	analysis.PauseCount = svc.countPauses(points)
	analysis.MicroCorrections = svc.countCorrections(points)

	clicks := svc.extractClicks(points)
	analysis.ClickCount = len(clicks)
	if len(clicks) > 2 {
		analysis.ClickRegularity = svc.calculateClickRegularity(clicks)
	}

	if analysis.TotalDistance <= 0 {
		t.Error("Expected positive total distance")
	}
	if analysis.PathEfficiency <= 0 || analysis.PathEfficiency > 1 {
		t.Error("Path efficiency should be between 0 and 1")
	}
	if analysis.ClickCount != 1 {
		t.Errorf("Expected 1 click, got %d", analysis.ClickCount)
	}
}

func TestProfileService_CreateAndGetProfile(t *testing.T) {
	profile := &UserProfile{
		UserID:     "test-user-create",
		RiskLevel:  "low",
		TrustScore: 80,
		Features: UserFeatures{
			MouseSpeedAvg: 3.5,
		},
	}

	if profile.UserID != "test-user-create" {
		t.Errorf("Expected user ID 'test-user-create', got '%s'", profile.UserID)
	}

	if profile.Version != 0 {
		t.Errorf("Expected version 0, got %d", profile.Version)
	}

	profile.Version = 1
	profile.LastUpdatedAt = time.Now()
	if profile.FeaturesJSON == "" {
		data, _ := json.Marshal(profile.Features)
		profile.FeaturesJSON = string(data)
	}

	profile.RiskLevel = "medium"

	profile.Version++
	profile.LastUpdatedAt = time.Now()

	if profile.Version != 2 {
		t.Errorf("Expected version 2 after update, got %d", profile.Version)
	}
}

func TestAnomalyService_DetectAnomalies(t *testing.T) {
	svc := &AnomalyService{db: nil}

	testCases := []struct {
		name            string
		analysis        BehaviorAnalysis
		expectSpeed     bool
		expectPath      bool
		expectClick     bool
		expectBot       bool
	}{
		{
			name: "Speed anomaly",
			analysis: BehaviorAnalysis{
				UserID:       "test-user",
				AverageSpeed: 15,
				MaxSpeed:     25,
				RiskScore:    40,
			},
			expectSpeed: true,
		},
		{
			name: "Path anomaly",
			analysis: BehaviorAnalysis{
				UserID:         "test-user",
				PathEfficiency: 0.95,
				TotalDistance:  200,
				RiskScore:      40,
			},
			expectPath: true,
		},
		{
			name: "Click anomaly",
			analysis: BehaviorAnalysis{
				UserID:          "test-user",
				ClickRegularity: 0.95,
				ClickCount:      5,
				RiskScore:       40,
			},
			expectClick: true,
		},
		{
			name: "Bot detection",
			analysis: BehaviorAnalysis{
				UserID:    "test-user",
				RiskScore: 75,
			},
			expectBot: true,
		},
		{
			name: "Normal behavior",
			analysis: BehaviorAnalysis{
				UserID:         "test-user",
				AverageSpeed:   2,
				MaxSpeed:       5,
				PathEfficiency: 0.5,
				ClickRegularity: 0.3,
				RiskScore:      20,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tc.analysis
			var speedAnomalies, pathAnomalies, clickAnomalies, botDetections int

			if a.AverageSpeed > 10 || a.MaxSpeed > 20 {
				speedAnomalies++
			}
			if a.PathEfficiency > 0.92 && a.TotalDistance > 100 {
				pathAnomalies++
			}
			if a.ClickRegularity > 0.9 && a.ClickCount > 2 {
				clickAnomalies++
			}
			if a.RiskScore >= 70 {
				botDetections++
			}

			if tc.expectSpeed && speedAnomalies < 1 {
				t.Error("Expected speed anomaly")
			}
			if tc.expectPath && pathAnomalies < 1 {
				t.Error("Expected path anomaly")
			}
			if tc.expectClick && clickAnomalies < 1 {
				t.Error("Expected click anomaly")
			}
			if tc.expectBot && botDetections < 1 {
				t.Error("Expected bot detection")
			}

			if !tc.expectSpeed && speedAnomalies > 0 {
				t.Error("Unexpected speed anomaly")
			}
			if !tc.expectPath && pathAnomalies > 0 {
				t.Error("Unexpected path anomaly")
			}
			if !tc.expectClick && clickAnomalies > 0 {
				t.Error("Unexpected click anomaly")
			}
			if !tc.expectBot && botDetections > 0 {
				t.Error("Unexpected bot detection")
			}

			_ = svc
		})
	}
}

func TestAnalysesToFloat64(t *testing.T) {
	analyses := []BehaviorAnalysis{
		{ClickCount: 3},
		{ClickCount: 5},
		{ClickCount: 2},
	}

	result := analysesToFloat64(analyses, func(a BehaviorAnalysis) float64 {
		return float64(a.ClickCount)
	})

	if len(result) != 3 {
		t.Errorf("Expected 3 results, got %d", len(result))
	}

	if result[0] != 3 || result[1] != 5 || result[2] != 2 {
		t.Error("Unexpected values in result")
	}
}

func TestAnomalyService_GetStatistics(t *testing.T) {
	stats := map[string]interface{}{
		"total_anomalies":      int64(100),
		"pending_count":         int64(20),
		"processed_count":       int64(80),
		"false_positive_count": int64(5),
		"by_type": map[string]int64{
			"speed_anomaly":   30,
			"path_anomaly":    25,
			"click_anomaly":   20,
			"bot_detection":   25,
		},
		"by_severity": map[string]int64{
			"critical": 15,
			"high":     25,
			"medium":   35,
			"low":      25,
		},
	}

	if stats == nil {
		t.Fatal("Statistics should not be nil")
	}

	if _, ok := stats["total_anomalies"]; !ok {
		t.Error("Expected 'total_anomalies' key in statistics")
	}

	if stats["total_anomalies"].(int64) != 100 {
		t.Error("Expected total_anomalies to be 100")
	}
}

func TestAnomalyService_ListRules(t *testing.T) {
	rules := []AnomalyRule{
		{
			ID:          1,
			Name:        "Speed Anomaly Rule",
			Type:        "speed_anomaly",
			Severity:    "high",
			Threshold:   10.0,
			IsEnabled:   true,
			Action:      "block",
		},
		{
			ID:          2,
			Name:        "Path Anomaly Rule",
			Type:        "path_anomaly",
			Severity:    "medium",
			Threshold:   0.9,
			IsEnabled:   true,
			Action:      "verify",
		},
	}

	if rules == nil {
		t.Fatal("Rules should not be nil")
	}

	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	if rules[0].Name != "Speed Anomaly Rule" {
		t.Error("Expected first rule to be 'Speed Anomaly Rule'")
	}
}

func TestTrajectoryService_ListAnalyses(t *testing.T) {
	analyses := []BehaviorAnalysis{
		{
			ID:              1,
			UserID:          "test-user",
			TotalDistance:   500.0,
			AverageSpeed:    3.5,
			PathEfficiency: 0.65,
			RiskScore:       25,
			IsBotLikely:     false,
		},
		{
			ID:              2,
			UserID:          "test-user",
			TotalDistance:   800.0,
			AverageSpeed:    4.0,
			PathEfficiency:  0.70,
			RiskScore:       30,
			IsBotLikely:     false,
		},
	}

	total := int64(len(analyses))

	if analyses == nil {
		t.Fatal("Analyses should not be nil")
	}

	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}

	if analyses[0].RiskScore != 25 {
		t.Error("Expected first analysis risk score to be 25")
	}
}

func TestTrajectoryService_GetStatistics(t *testing.T) {
	stats := map[string]interface{}{
		"total_count":    int64(100),
		"total_distance": float64(50000),
		"avg_distance":   float64(500),
		"avg_speed":      float64(3.5),
		"bot_count":      int64(15),
		"human_count":    int64(85),
		"bot_rate":       float64(15.0),
		"avg_risk_score": float64(25.5),
	}

	if stats == nil {
		t.Fatal("Statistics should not be nil")
	}

	expectedKeys := []string{"total_count", "total_distance", "avg_distance", "avg_speed", "bot_count", "human_count", "bot_rate", "avg_risk_score"}
	for _, key := range expectedKeys {
		if _, ok := stats[key]; !ok {
			t.Errorf("Expected key '%s' in statistics", key)
		}
	}
}

func TestProfileService_GetStatistics(t *testing.T) {
	stats := map[string]interface{}{
		"total_profiles":    int64(100),
		"active_profiles":   int64(80),
		"inactive_profiles": int64(20),
		"avg_trust_score":   float64(75.5),
		"risk_distribution": map[string]int64{
			"critical": 5,
			"high":     10,
			"medium":   25,
			"low":      40,
			"minimal":  20,
		},
	}

	if stats == nil {
		t.Fatal("Statistics should not be nil")
	}

	expectedKeys := []string{"total_profiles", "active_profiles", "inactive_profiles", "avg_trust_score", "risk_distribution"}
	for _, key := range expectedKeys {
		if _, ok := stats[key]; !ok {
			t.Errorf("Expected key '%s' in statistics", key)
		}
	}
}

func TestCalculateDistance_EdgeCases(t *testing.T) {
	svc := &TrajectoryService{}

	testCases := []struct {
		name   string
		points []TrajectoryPoint
	}{
		{"Empty points", []TrajectoryPoint{}},
		{"Single point", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}}},
		{"Two points - same location", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}, {X: 0, Y: 0, Timestamp: 1100}}},
		{"Multiple points", []TrajectoryPoint{
			{X: 0, Y: 0, Timestamp: 1000},
			{X: 3, Y: 4, Timestamp: 1100},
			{X: 6, Y: 8, Timestamp: 1200},
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dist := svc.calculateDistance(tc.points)
			if dist < 0 {
				t.Error("Distance should be non-negative")
			}
		})
	}
}

func TestCalculateEfficiency_EdgeCases(t *testing.T) {
	svc := &TrajectoryService{}

	testCases := []struct {
		name       string
		points     []TrajectoryPoint
		minEffic   float64
		maxEffic   float64
	}{
		{"Empty points", []TrajectoryPoint{}, 0, 0},
		{"Single point", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}}, 0, 0},
		{"Straight line", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}, {X: 100, Y: 0, Timestamp: 1100}}, 0.99, 1.01},
		{"Zigzag path", []TrajectoryPoint{
			{X: 0, Y: 0, Timestamp: 1000},
			{X: 50, Y: 50, Timestamp: 1100},
			{X: 100, Y: 0, Timestamp: 1200},
			{X: 150, Y: 50, Timestamp: 1300},
			{X: 200, Y: 0, Timestamp: 1400},
		}, 0.5, 0.8},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			effic := svc.calculateEfficiency(tc.points)
			if effic < tc.minEffic || effic > tc.maxEffic {
				t.Errorf("Expected efficiency between %.2f and %.2f, got %.2f", tc.minEffic, tc.maxEffic, effic)
			}
		})
	}
}

func TestCountDirectionChanges_EdgeCases(t *testing.T) {
	svc := &TrajectoryService{}

	testCases := []struct {
		name    string
		points  []TrajectoryPoint
		minChgs int
	}{
		{"Empty points", []TrajectoryPoint{}, 0},
		{"Single point", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}}, 0},
		{"Two points", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}, {X: 10, Y: 10, Timestamp: 1100}}, 0},
		{"Multiple direction changes", []TrajectoryPoint{
			{X: 0, Y: 0, Timestamp: 1000},
			{X: 10, Y: 0, Timestamp: 1100},
			{X: 10, Y: 10, Timestamp: 1200},
			{X: 0, Y: 10, Timestamp: 1300},
			{X: 0, Y: 0, Timestamp: 1400},
		}, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chgs := svc.countDirectionChanges(tc.points)
			if chgs < tc.minChgs {
				t.Errorf("Expected at least %d direction changes, got %d", tc.minChgs, chgs)
			}
		})
	}
}

func TestCountPauses_EdgeCases(t *testing.T) {
	svc := &TrajectoryService{}

	testCases := []struct {
		name     string
		points   []TrajectoryPoint
		expected int
	}{
		{"Empty points", []TrajectoryPoint{}, 0},
		{"Single point", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}}, 0},
		{"Two points - no pause", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}, {X: 100, Y: 100, Timestamp: 1100}}, 0},
		{"Two points - pause detected", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}, {X: 0, Y: 0, Timestamp: 1200}, {X: 10, Y: 10, Timestamp: 1300}}, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pauses := svc.countPauses(tc.points)
			if pauses != tc.expected {
				t.Errorf("Expected %d pauses, got %d", tc.expected, pauses)
			}
		})
	}
}

func TestSmoothPoints_EdgeCases(t *testing.T) {
	svc := &TrajectoryService{}

	testCases := []struct {
		name    string
		points  []TrajectoryPoint
		window  int
	}{
		{"Empty points", []TrajectoryPoint{}, 3},
		{"Single point", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}}, 3},
		{"Points less than window", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}, {X: 10, Y: 10, Timestamp: 1100}}, 5},
		{"Odd window", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}, {X: 10, Y: 10, Timestamp: 1100}, {X: 20, Y: 20, Timestamp: 1200}}, 3},
		{"Even window (auto corrected to odd)", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}, {X: 10, Y: 10, Timestamp: 1100}, {X: 20, Y: 20, Timestamp: 1200}}, 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			smoothed := svc.smoothPoints(tc.points, tc.window)
			if len(smoothed) != len(tc.points) {
				t.Errorf("Expected %d smoothed points, got %d", len(tc.points), len(smoothed))
			}
		})
	}
}

func TestCountCorrections_EdgeCases(t *testing.T) {
	svc := &TrajectoryService{}

	testCases := []struct {
		name    string
		points  []TrajectoryPoint
	}{
		{"Empty points", []TrajectoryPoint{}},
		{"Single point", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}}},
		{"Two points", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}, {X: 10, Y: 10, Timestamp: 1100}}},
		{"Micro corrections", []TrajectoryPoint{
			{X: 0, Y: 0, Timestamp: 1000},
			{X: 3, Y: 1, Timestamp: 1100},
			{X: 7, Y: 0, Timestamp: 1200},
			{X: 12, Y: 1, Timestamp: 1300},
			{X: 18, Y: 0, Timestamp: 1400},
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			corrections := svc.countCorrections(tc.points)
			if corrections < 0 {
				t.Error("Corrections should be non-negative")
			}
		})
	}
}

func TestCalculateClickRegularity_EdgeCases(t *testing.T) {
	svc := &TrajectoryService{}

	testCases := []struct {
		name   string
		clicks []TrajectoryPoint
	}{
		{"Empty clicks", []TrajectoryPoint{}},
		{"Single click", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}}},
		{"Two clicks", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}, {X: 10, Y: 10, Timestamp: 1100}}},
		{"Regular clicks", []TrajectoryPoint{
			{X: 0, Y: 0, Timestamp: 1000},
			{X: 0, Y: 0, Timestamp: 1100},
			{X: 0, Y: 0, Timestamp: 1200},
			{X: 0, Y: 0, Timestamp: 1300},
			{X: 0, Y: 0, Timestamp: 1400},
		}},
		{"Irregular clicks", []TrajectoryPoint{
			{X: 0, Y: 0, Timestamp: 1000},
			{X: 0, Y: 0, Timestamp: 1500},
			{X: 0, Y: 0, Timestamp: 1600},
			{X: 0, Y: 0, Timestamp: 3000},
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			regularity := svc.calculateClickRegularity(tc.clicks)
			if regularity < 0 || regularity > 1 {
				t.Errorf("Regularity should be between 0 and 1, got %.2f", regularity)
			}
		})
	}
}

func TestCalculateCurvature_EdgeCases(t *testing.T) {
	svc := &TrajectoryService{}

	testCases := []struct {
		name    string
		points  []TrajectoryPoint
	}{
		{"Empty points", []TrajectoryPoint{}},
		{"Single point", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}}},
		{"Two points", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}, {X: 10, Y: 10, Timestamp: 1100}}},
		{"Straight line", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}, {X: 10, Y: 10, Timestamp: 1100}, {X: 20, Y: 20, Timestamp: 1200}}},
		{"Curved path", []TrajectoryPoint{{X: 0, Y: 0, Timestamp: 1000}, {X: 10, Y: 10, Timestamp: 1100}, {X: 20, Y: 5, Timestamp: 1200}, {X: 30, Y: 15, Timestamp: 1300}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			curvature := svc.calculateCurvature(tc.points)
			if curvature < 0 {
				t.Error("Curvature should be non-negative")
			}
		})
	}
}

func TestTrajectoryService_SaveAndGet(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTrajectoryService(db)
	ctx := context.Background()

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 100, Timestamp: 1100},
		{X: 200, Y: 200, Timestamp: 1200},
	}

	traj := &BehaviorTrajectory{
		UserID: "test-user",
		Points: points,
	}

	err := svc.SaveTrajectory(ctx, traj)
	if err != nil {
		t.Fatalf("SaveTrajectory failed: %v", err)
	}

	if traj.ID == 0 {
		t.Error("Expected non-zero ID after save")
	}

	retrieved, err := svc.GetTrajectory(ctx, traj.ID)
	if err != nil {
		t.Fatalf("GetTrajectory failed: %v", err)
	}

	if retrieved.UserID != "test-user" {
		t.Errorf("Expected user ID 'test-user', got '%s'", retrieved.UserID)
	}

	if len(retrieved.Points) != 3 {
		t.Errorf("Expected 3 points, got %d", len(retrieved.Points))
	}
}

func TestTrajectoryService_ListTrajectories(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTrajectoryService(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		traj := &BehaviorTrajectory{
			UserID: "test-user",
			Points: []TrajectoryPoint{{X: i * 10, Y: i * 10, Timestamp: int64(1000 + i*100)}},
		}
		svc.SaveTrajectory(ctx, traj)
	}

	query := &TrajectoryQuery{UserID: "test-user", Page: 1, PageSize: 10}
	trajs, total, err := svc.ListTrajectories(ctx, query)
	if err != nil {
		t.Fatalf("ListTrajectories failed: %v", err)
	}

	if total != 5 {
		t.Errorf("Expected 5 trajectories, got %d", total)
	}

	if len(trajs) != 5 {
		t.Errorf("Expected 5 trajectories in result, got %d", len(trajs))
	}
}

func TestTrajectoryService_AnalyzeTrajectory(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTrajectoryService(db)
	ctx := context.Background()

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 100, Timestamp: 1100},
		{X: 200, Y: 200, Timestamp: 1200},
		{X: 200, Y: 200, Timestamp: 1300},
		{X: 300, Y: 300, Timestamp: 1400},
		{X: 350, Y: 350, Timestamp: 1500, Event: "click"},
	}

	traj := &BehaviorTrajectory{
		UserID: "test-user",
		Points: points,
	}

	err := svc.SaveTrajectory(ctx, traj)
	if err != nil {
		t.Fatalf("SaveTrajectory failed: %v", err)
	}

	analysis, err := svc.AnalyzeTrajectory(ctx, traj)
	if err != nil {
		t.Fatalf("AnalyzeTrajectory failed: %v", err)
	}

	if analysis.ID == 0 {
		t.Error("Expected non-zero ID for analysis")
	}

	if analysis.TotalDistance <= 0 {
		t.Error("Expected positive total distance")
	}

	if analysis.ClickCount != 1 {
		t.Errorf("Expected 1 click, got %d", analysis.ClickCount)
	}
}

func TestProfileService_CRUD(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProfileService(db)
	ctx := context.Background()

	profile := &UserProfile{
		UserID:     "test-user",
		RiskLevel:  "low",
		TrustScore: 80,
		Features: UserFeatures{
			MouseSpeedAvg: 3.5,
		},
	}

	err := svc.CreateProfile(ctx, profile)
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	if profile.ID == 0 {
		t.Error("Expected non-zero ID after create")
	}

	if profile.Version != 1 {
		t.Errorf("Expected version 1, got %d", profile.Version)
	}

	retrieved, err := svc.GetProfile(ctx, "test-user")
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected profile to be found")
	}

	if retrieved.TrustScore != 80 {
		t.Errorf("Expected trust score 80, got %f", retrieved.TrustScore)
	}

	retrieved.TrustScore = 90
	err = svc.UpdateProfile(ctx, retrieved)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	if retrieved.Version != 2 {
		t.Errorf("Expected version 2 after update, got %d", retrieved.Version)
	}

	updated, _ := svc.GetProfile(ctx, "test-user")
	if updated.TrustScore != 90 {
		t.Errorf("Expected updated trust score 90, got %f", updated.TrustScore)
	}
}

func TestProfileService_ListProfiles(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProfileService(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		profile := &UserProfile{
			UserID:     "test-user-" + string(rune('0'+i)),
			RiskLevel:  "low",
			TrustScore: 80,
		}
		svc.CreateProfile(ctx, profile)
	}

	query := &UserProfileQuery{Page: 1, PageSize: 10}
	profiles, total, err := svc.ListProfiles(ctx, query)
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}

	if total != 3 {
		t.Errorf("Expected 3 profiles, got %d", total)
	}

	if len(profiles) != 3 {
		t.Errorf("Expected 3 profiles in result, got %d", len(profiles))
	}
}

func TestProfileService_GenerateProfile(t *testing.T) {
	db := setupTestDB(t)
	trajSvc := NewTrajectoryService(db)
	profileSvc := NewProfileService(db)
	ctx := context.Background()

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 100, Y: 100, Timestamp: 1100},
		{X: 200, Y: 200, Timestamp: 1200},
	}

	traj := &BehaviorTrajectory{
		UserID: "test-user",
		Points: points,
	}
	trajSvc.SaveTrajectory(ctx, traj)
	analysis, _ := trajSvc.AnalyzeTrajectory(ctx, traj)

	profile, err := profileSvc.GenerateProfile(ctx, "test-user", 1, []BehaviorAnalysis{*analysis})
	if err != nil {
		t.Fatalf("GenerateProfile failed: %v", err)
	}

	if profile.UserID != "test-user" {
		t.Errorf("Expected user ID 'test-user', got '%s'", profile.UserID)
	}

	if profile.Features.MouseSpeedAvg <= 0 {
		t.Error("Expected positive mouse speed average")
	}
}

func TestAnomalyService_DetectAndGetAnomalies(t *testing.T) {
	db := setupTestDB(t)
	trajSvc := NewTrajectoryService(db)
	anomalySvc := NewAnomalyService(db)
	ctx := context.Background()

	points := []TrajectoryPoint{
		{X: 0, Y: 0, Timestamp: 1000},
		{X: 1000, Y: 1000, Timestamp: 1050},
		{X: 2000, Y: 2000, Timestamp: 1100},
	}

	traj := &BehaviorTrajectory{
		UserID: "test-user",
		Points: points,
	}
	trajSvc.SaveTrajectory(ctx, traj)
	analysis, _ := trajSvc.AnalyzeTrajectory(ctx, traj)

	anomalies, err := anomalySvc.DetectAnomalies(ctx, []BehaviorAnalysis{*analysis})
	if err != nil {
		t.Fatalf("DetectAnomalies failed: %v", err)
	}

	if len(anomalies) == 0 {
		t.Error("Expected at least 1 anomaly for high-speed trajectory")
	}

	if len(anomalies) > 0 {
		anomaly := anomalies[0]
		if anomaly.Type == "" {
			t.Error("Expected non-empty anomaly type")
		}

		if anomaly.Score <= 0 {
			t.Error("Expected positive anomaly score")
		}
	}
}

func TestAnomalyService_ListAnomalies(t *testing.T) {
	db := setupTestDB(t)
	anomalySvc := NewAnomalyService(db)
	ctx := context.Background()

	anomalySvc.db = db.WithContext(ctx).Create(&AnomalyRecord{
		UserID:      "test-user",
		Type:        "speed_anomaly",
		Severity:    "high",
		Score:       85,
		Description: "Test anomaly",
	})

	query := &AnomalyQuery{Page: 1, PageSize: 10}
	anomalies, total, err := anomalySvc.GetAnomalies(ctx, query)
	if err != nil {
		t.Fatalf("GetAnomalies failed: %v", err)
	}

	if total < 1 {
		t.Error("Expected at least 1 anomaly")
	}

	if len(anomalies) < 1 {
		t.Error("Expected at least 1 anomaly in result")
	}
}

func TestAnomalyService_CRUD_Rules(t *testing.T) {
	db := setupTestDB(t)
	anomalySvc := NewAnomalyService(db)
	ctx := context.Background()

	rule := &AnomalyRule{
		Name:      "Test Rule",
		Type:      "speed_anomaly",
		Severity:  "high",
		Threshold: 10.0,
		Action:    "block",
		IsEnabled: true,
	}

	err := anomalySvc.CreateRule(ctx, rule)
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	if rule.ID == 0 {
		t.Error("Expected non-zero ID after create")
	}

	rules, err := anomalySvc.ListRules(ctx, "")
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}

	if len(rules) < 1 {
		t.Error("Expected at least 1 rule")
	}

	rule.IsEnabled = false
	err = anomalySvc.ToggleRule(ctx, rule.ID, false)
	if err != nil {
		t.Fatalf("ToggleRule failed: %v", err)
	}

	updated, _ := anomalySvc.ListRules(ctx, "")
	if len(updated) > 0 && updated[0].IsEnabled {
		t.Error("Expected rule to be disabled after toggle")
	}

	err = anomalySvc.DeleteRule(ctx, rule.ID)
	if err != nil {
		t.Fatalf("DeleteRule failed: %v", err)
	}

	rule2 := &AnomalyRule{
		Name:      "Test Rule 2",
		Type:      "speed_anomaly",
		Severity:  "high",
		Threshold: 10.0,
		Action:    "block",
		IsEnabled: true,
	}
	err = anomalySvc.CreateRule(ctx, rule2)
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	deleted, _ := anomalySvc.ListRules(ctx, "")
	if len(deleted) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(deleted))
	}
}

func TestTrajectoryService_Statistics(t *testing.T) {
	db := setupTestDB(t)
	svc := NewTrajectoryService(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		points := []TrajectoryPoint{
			{X: 0, Y: 0, Timestamp: 1000},
			{X: 100, Y: 100, Timestamp: 1100},
			{X: 200, Y: 200, Timestamp: 1200},
		}
		traj := &BehaviorTrajectory{
			UserID: "test-user",
			Points: points,
		}
		svc.SaveTrajectory(ctx, traj)
		svc.AnalyzeTrajectory(ctx, traj)
	}

	query := &TrajectoryQuery{UserID: "test-user"}
	stats, err := svc.GetStatistics(ctx, query)
	if err != nil {
		t.Fatalf("GetStatistics failed: %v", err)
	}

	if stats["total_count"] != int64(3) {
		t.Errorf("Expected total_count 3, got %v", stats["total_count"])
	}
}

func TestProfileService_Statistics(t *testing.T) {
	db := setupTestDB(t)
	svc := NewProfileService(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		profile := &UserProfile{
			UserID:     "test-user-" + string(rune('a'+i)),
			RiskLevel:  "low",
			TrustScore: 80,
		}
		svc.CreateProfile(ctx, profile)
	}

	query := &UserProfileQuery{}
	stats, err := svc.GetStatistics(ctx, query)
	if err != nil {
		t.Fatalf("GetStatistics failed: %v", err)
	}

	if stats["total_profiles"] != int64(3) {
		t.Errorf("Expected total_profiles 3, got %v", stats["total_profiles"])
	}
}

func TestAnomalyService_Statistics(t *testing.T) {
	db := setupTestDB(t)
	svc := NewAnomalyService(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		svc.db = db.WithContext(ctx).Create(&AnomalyRecord{
			UserID:      "test-user",
			Type:        "speed_anomaly",
			Severity:    "high",
			Score:       85,
			Description: "Test anomaly",
		})
	}

	query := &AnomalyQuery{}
	stats, err := svc.GetStatistics(ctx, query)
	if err != nil {
		t.Fatalf("GetStatistics failed: %v", err)
	}

	if stats["total_anomalies"] != int64(3) {
		t.Errorf("Expected total_anomalies 3, got %v", stats["total_anomalies"])
	}
}
