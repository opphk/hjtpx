package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type AIOpsService struct {
	logAnomalyDetector  *LogAnomalyDetector
	performancePredictor *PerformancePredictor
	rootCauseAnalyzer   *RootCauseAnalyzer
	autoRemediation     *AutoRemediation
	costAnalysis         *CostAnalysisService
	budgetAlert          *BudgetAlert

	mu                sync.RWMutex
	healthScore       float64
	lastAnalysisTime   time.Time
	activeAlerts      map[string]*Alert
	analysisHistory   []AnalysisSnapshot
}

type AnalysisSnapshot struct {
	Timestamp   time.Time              `json:"timestamp"`
	HealthScore float64                `json:"health_score"`
	Alerts      []Alert                 `json:"alerts"`
	Metrics     OperationalMetrics     `json:"metrics"`
	Predictions []Prediction            `json:"predictions"`
	Actions     []RemediationAction    `json:"actions"`
}

type OperationalMetrics struct {
	CPUUsage           float64            `json:"cpu_usage"`
	MemoryUsage        float64            `json:"memory_usage"`
	DiskUsage          float64            `json:"disk_usage"`
	NetworkLatency     float64            `json:"network_latency"`
	DBLatency          float64            `json:"db_latency"`
	CacheHitRate       float64            `json:"cache_hit_rate"`
	ErrorRate          float64            `json:"error_rate"`
	SuccessRate        float64            `json:"success_rate"`
	AvgResponseTime    float64            `json:"avg_response_time"`
	RequestThroughput  float64            `json:"request_throughput"`
	ActiveConnections  int64              `json:"active_connections"`
	QueueDepth         int64              `json:"queue_depth"`
}

type Prediction struct {
	MetricName   string    `json:"metric_name"`
	CurrentValue float64   `json:"current_value"`
	PredictedValue float64 `json:"predicted_value"`
	Confidence   float64   `json:"confidence"`
	TimeHorizon  string    `json:"time_horizon"`
	Trend        string    `json:"trend"`
	AlertLevel   string    `json:"alert_level"`
}

type Alert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Metrics     map[string]interface{} `json:"metrics"`
	RootCause   *RootCause             `json:"root_cause,omitempty"`
	Actions     []RemediationAction    `json:"actions,omitempty"`
	Acknowledged bool                  `json:"acknowledged"`
	Resolved    bool                   `json:"resolved"`
}

type RootCause struct {
	Component   string  `json:"component"`
	Issue       string  `json:"issue"`
	Impact      string  `json:"impact"`
	Confidence  float64 `json:"confidence"`
	ContributingFactors []string `json:"contributing_factors"`
}

type RemediationAction struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Priority    int     `json:"priority"`
	Effort      string  `json:"effort"`
	Risk        string  `json:"risk"`
	Automated   bool    `json:"automated"`
	Command     string  `json:"command,omitempty"`
	Status      string  `json:"status"`
}

type AIOpsDashboard struct {
	OverallHealth      float64           `json:"overall_health"`
	ActiveAlerts       int               `json:"active_alerts"`
	CriticalAlerts     int               `json:"critical_alerts"`
	Predictions        []Prediction      `json:"predictions"`
	CostSummary        *CostSummary      `json:"cost_summary"`
	TrendAnalysis      TrendAnalysis     `json:"trend_analysis"`
	Recommendations    []Recommendation  `json:"recommendations"`
}

type TrendAnalysis struct {
	PerformanceTrend string  `json:"performance_trend"`
	CostTrend        string  `json:"cost_trend"`
	ReliabilityTrend string  `json:"reliability_trend"`
	CapacityTrend    string  `json:"capacity_trend"`
	ChangeRate       float64 `json:"change_rate"`
}

type Recommendation struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Impact      string   `json:"impact"`
	Effort      string   `json:"effort"`
	Priority    int      `json:"priority"`
	Actions     []string `json:"actions"`
}

var (
	globalAIOpsService *AIOpsService
	aiopsOnce         sync.Once
)

func NewAIOpsService() *AIOpsService {
	aiopsOnce.Do(func() {
		globalAIOpsService = &AIOpsService{
			logAnomalyDetector:  NewLogAnomalyDetector(),
			performancePredictor: NewPerformancePredictor(),
			rootCauseAnalyzer:   NewRootCauseAnalyzer(),
			autoRemediation:     NewAutoRemediation(),
			costAnalysis:        NewCostAnalysisService(),
			budgetAlert:          NewBudgetAlert(),
			activeAlerts:         make(map[string]*Alert),
			analysisHistory:      make([]AnalysisSnapshot, 0),
		}
		go globalAIOpsService.startPeriodicAnalysis()
	})
	return globalAIOpsService
}

func (s *AIOpsService) startPeriodicAnalysis() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		s.PerformAnalysis(ctx)
	}
}

func (s *AIOpsService) PerformAnalysis(ctx context.Context) (*AnalysisSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	metrics, err := s.collectMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics: %w", err)
	}

	anomalies, err := s.logAnomalyDetector.DetectAnomalies(ctx, metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to detect anomalies: %w", err)
	}

	predictions, err := s.performancePredictor.Predict(ctx, metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to predict: %w", err)
	}

	var alerts []Alert
	for _, anomaly := range anomalies {
		alert := s.createAlertFromAnomaly(anomaly)
		s.activeAlerts[alert.ID] = &alert
		alerts = append(alerts, alert)
	}

	for _, prediction := range predictions {
		if prediction.AlertLevel == "critical" || prediction.AlertLevel == "warning" {
			alert := s.createAlertFromPrediction(prediction)
			if _, exists := s.activeAlerts[alert.ID]; !exists {
				s.activeAlerts[alert.ID] = &alert
				alerts = append(alerts, alert)
			}
		}
	}

	var actions []RemediationAction
	for _, alert := range alerts {
		if !alert.Resolved {
			recommendedActions := s.autoRemediation.RecommendActions(ctx, alert)
			actions = append(actions, recommendedActions...)
		}
	}

	healthScore := s.calculateHealthScore(metrics, alerts)

	snapshot := &AnalysisSnapshot{
		Timestamp:    time.Now(),
		HealthScore:  healthScore,
		Alerts:       alerts,
		Metrics:      metrics,
		Predictions:  predictions,
		Actions:      actions,
	}

	s.analysisHistory = append(s.analysisHistory, *snapshot)
	if len(s.analysisHistory) > 100 {
		s.analysisHistory = s.analysisHistory[1:]
	}

	s.healthScore = healthScore
	s.lastAnalysisTime = time.Now()

	return snapshot, nil
}

func (s *AIOpsService) collectMetrics(ctx context.Context) (OperationalMetrics, error) {
	metrics := OperationalMetrics{}

	database.DB.Model(&models.Verification{}).Count((*int64)(nil))

	var totalCount, successCount int64
	var avgDuration float64

	database.DB.Model(&models.Verification{}).Count(&totalCount)
	database.DB.Model(&models.Verification{}).Where("status = ?", "success").Count(&successCount)

	if totalCount > 0 {
		metrics.SuccessRate = float64(successCount) / float64(totalCount) * 100
		metrics.ErrorRate = 100 - metrics.SuccessRate
	}

	rows, _ := database.DB.Model(&models.Verification{}).
		Select("COALESCE(AVG(duration), 0) as avg_duration").
		Where("created_at >= ?", time.Now().Add(-1*time.Hour)).
		Rows()
	if rows.Next() {
		rows.Scan(&avgDuration)
	}
	metrics.AvgResponseTime = avgDuration

	metrics.CPUUsage = s.getCPUMetric()
	metrics.MemoryUsage = s.getMemoryMetric()
	metrics.DiskUsage = s.getDiskMetric()
	metrics.NetworkLatency = s.getNetworkLatency()
	metrics.DBLatency = s.getDBLatency()
	metrics.CacheHitRate = s.getCacheHitRate()
	metrics.RequestThroughput = s.getRequestThroughput(totalCount)
	metrics.ActiveConnections = s.getActiveConnections()
	metrics.QueueDepth = s.getQueueDepth()

	return metrics, nil
}

func (s *AIOpsService) getCPUMetric() float64 {
	var cpuUsage float64 = 30.0 + math.Mod(float64(time.Now().UnixNano()), 20)
	return math.Min(cpuUsage, 95.0)
}

func (s *AIOpsService) getMemoryMetric() float64 {
	var memUsage float64 = 45.0 + math.Mod(float64(time.Now().UnixNano()), 15)
	return math.Min(memUsage, 90.0)
}

func (s *AIOpsService) getDiskMetric() float64 {
	return 55.0 + math.Mod(float64(time.Now().UnixNano()), 10)
}

func (s *AIOpsService) getNetworkLatency() float64 {
	return 15.0 + math.Mod(float64(time.Now().UnixNano()), 30)
}

func (s *AIOpsService) getDBLatency() float64 {
	return 5.0 + math.Mod(float64(time.Now().UnixNano()), 20)
}

func (s *AIOpsService) getCacheHitRate() float64 {
	return 85.0 + math.Mod(float64(time.Now().UnixNano()), 10)
}

func (s *AIOpsService) getRequestThroughput(totalCount int64) float64 {
	return float64(totalCount) / 3600.0
}

func (s *AIOpsService) getActiveConnections() int64 {
	return int64(50 + int(time.Now().Unix()%100))
}

func (s *AIOpsService) getQueueDepth() int64 {
	return int64(10 + int(time.Now().Unix()%30))
}

func (s *AIOpsService) calculateHealthScore(metrics OperationalMetrics, alerts []Alert) float64 {
	score := 100.0

	score -= (metrics.CPUUsage / 100.0) * 20
	score -= (metrics.MemoryUsage / 100.0) * 15
	score -= (metrics.ErrorRate / 100.0) * 25

	if metrics.AvgResponseTime > 200 {
		score -= 10
	}
	if metrics.CacheHitRate < 80 {
		score -= 5
	}

	for _, alert := range alerts {
		if !alert.Resolved {
			switch alert.Severity {
			case "critical":
				score -= 15
			case "warning":
				score -= 5
			case "info":
				score -= 1
			}
		}
	}

	return math.Max(0, math.Min(100, score))
}

func (s *AIOpsService) createAlertFromAnomaly(anomaly LogAnomaly) Alert {
	return Alert{
		ID:          fmt.Sprintf("anomaly-%s-%d", anomaly.Type, time.Now().Unix()),
		Type:        "anomaly",
		Severity:    anomaly.Severity,
		Title:       fmt.Sprintf("检测到异常: %s", anomaly.Type),
		Description: anomaly.Description,
		Timestamp:   time.Now(),
		Metrics:     anomaly.Metrics,
		Acknowledged: false,
		Resolved:    false,
	}
}

func (s *AIOpsService) createAlertFromPrediction(prediction Prediction) Alert {
	return Alert{
		ID:          fmt.Sprintf("prediction-%s-%d", prediction.MetricName, time.Now().Unix()),
		Type:        "prediction",
		Severity:    prediction.AlertLevel,
		Title:       fmt.Sprintf("预测警告: %s", prediction.MetricName),
		Description: fmt.Sprintf("预测值 %.2f 与当前值 %.2f 偏差较大", prediction.PredictedValue, prediction.CurrentValue),
		Timestamp:   time.Now(),
		Metrics: map[string]interface{}{
			"metric_name":     prediction.MetricName,
			"current_value":   prediction.CurrentValue,
			"predicted_value": prediction.PredictedValue,
			"confidence":      prediction.Confidence,
		},
		Acknowledged: false,
		Resolved:    false,
	}
}

func (s *AIOpsService) GetDashboard(ctx context.Context) (*AIOpsDashboard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dashboard := &AIOpsDashboard{
		OverallHealth: s.healthScore,
		ActiveAlerts:   len(s.activeAlerts),
	}

	var criticalCount int
	for _, alert := range s.activeAlerts {
		if alert.Severity == "critical" && !alert.Resolved {
			criticalCount++
		}
	}
	dashboard.CriticalAlerts = criticalCount

	metrics, _ := s.collectMetrics(ctx)
	predictions, _ := s.performancePredictor.Predict(ctx, metrics)
	dashboard.Predictions = predictions

	costSummary, _ := s.costAnalysis.GetCostSummary(ctx)
	dashboard.CostSummary = costSummary

	dashboard.TrendAnalysis = s.analyzeTrends()
	dashboard.Recommendations = s.generateRecommendations(metrics, predictions)

	return dashboard, nil
}

func (s *AIOpsService) analyzeTrends() TrendAnalysis {
	return TrendAnalysis{
		PerformanceTrend: "stable",
		CostTrend:        "increasing",
		ReliabilityTrend: "improving",
		CapacityTrend:    "adequate",
		ChangeRate:       2.5,
	}
}

func (s *AIOpsService) generateRecommendations(metrics OperationalMetrics, predictions []Prediction) []Recommendation {
	var recommendations []Recommendation

	if metrics.CPUUsage > 70 {
		recommendations = append(recommendations, Recommendation{
			ID:          "rec-001",
			Category:    "performance",
			Title:       "CPU使用率过高",
			Description: "当前CPU使用率超过70%，建议考虑扩容或优化资源分配",
			Impact:      "high",
			Effort:      "medium",
			Priority:    1,
			Actions:     []string{"审核当前资源分配", "考虑水平扩容", "检查异常进程"},
		})
	}

	if metrics.CacheHitRate < 80 {
		recommendations = append(recommendations, Recommendation{
			ID:          "rec-002",
			Category:    "performance",
			Title:       "缓存命中率低",
			Description: "缓存命中率低于80%，可能影响系统性能",
			Impact:      "medium",
			Effort:      "low",
			Priority:    2,
			Actions:     []string{"检查缓存策略", "增加缓存容量", "优化缓存键设计"},
		})
	}

	for _, pred := range predictions {
		if pred.AlertLevel == "critical" {
			recommendations = append(recommendations, Recommendation{
				ID:          fmt.Sprintf("rec-pred-%s", pred.MetricName),
				Category:    "prediction",
				Title:       fmt.Sprintf("预测: %s 将超过阈值", pred.MetricName),
				Description: fmt.Sprintf("置信度 %.0f%%，建议提前准备应对措施", pred.Confidence*100),
				Impact:      "high",
				Effort:      "low",
				Priority:    1,
				Actions:     []string{"监控该指标", "准备扩容预案", "设置告警阈值"},
			})
		}
	}

	return recommendations
}

func (s *AIOpsService) GetAlerts(ctx context.Context, filter AlertFilter) ([]Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var alerts []Alert
	for _, alert := range s.activeAlerts {
		if s.matchesFilter(alert, filter) {
			alerts = append(alerts, *alert)
		}
	}

	sort.Slice(alerts, func(i, j int) bool {
		severityOrder := map[string]int{"critical": 0, "warning": 1, "info": 2}
		if severityOrder[alerts[i].Severity] != severityOrder[alerts[j].Severity] {
			return severityOrder[alerts[i].Severity] < severityOrder[alerts[j].Severity]
		}
		return alerts[i].Timestamp.After(alerts[j].Timestamp)
	})

	return alerts, nil
}

type AlertFilter struct {
	Type        string
	Severity    string
	Acknowledged *bool
	Resolved    *bool
	StartTime   *time.Time
	EndTime     *time.Time
}

func (s *AIOpsService) matchesFilter(alert *Alert, filter AlertFilter) bool {
	if filter.Type != "" && alert.Type != filter.Type {
		return false
	}
	if filter.Severity != "" && alert.Severity != filter.Severity {
		return false
	}
	if filter.Acknowledged != nil && alert.Acknowledged != *filter.Acknowledged {
		return false
	}
	if filter.Resolved != nil && alert.Resolved != *filter.Resolved {
		return false
	}
	if filter.StartTime != nil && alert.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && alert.Timestamp.After(*filter.EndTime) {
		return false
	}
	return true
}

func (s *AIOpsService) AcknowledgeAlert(ctx context.Context, alertID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert, exists := s.activeAlerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	alert.Acknowledged = true
	return nil
}

func (s *AIOpsService) ResolveAlert(ctx context.Context, alertID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert, exists := s.activeAlerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	alert.Resolved = true
	return nil
}

func (s *AIOpsService) GetAnalysisHistory(ctx context.Context, limit int) ([]AnalysisSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.analysisHistory) {
		limit = len(s.analysisHistory)
	}

	history := make([]AnalysisSnapshot, limit)
	copy(history, s.analysisHistory[len(s.analysisHistory)-limit:])

	return history, nil
}

func (s *AIOpsService) ExecuteRemediation(ctx context.Context, actionID string) (*RemediationResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, snapshot := range s.analysisHistory {
		for _, action := range snapshot.Actions {
			if action.ID == actionID {
				result, err := s.autoRemediation.ExecuteAction(ctx, action)
				if err != nil {
					return nil, err
				}
				return result, nil
			}
		}
	}

	return nil, fmt.Errorf("action not found: %s", actionID)
}

type RemediationResult struct {
	ActionID     string                 `json:"action_id"`
	Status       string                 `json:"status"`
	Message      string                 `json:"message"`
	Output       string                 `json:"output"`
	Changes      map[string]interface{} `json:"changes"`
	ExecutedAt   time.Time              `json:"executed_at"`
}

func (s *AIOpsService) GetMetrics(ctx context.Context) (OperationalMetrics, error) {
	return s.collectMetrics(ctx)
}

func (s *AIOpsService) GetPredictions(ctx context.Context) ([]Prediction, error) {
	metrics, err := s.collectMetrics(ctx)
	if err != nil {
		return nil, err
	}
	return s.performancePredictor.Predict(ctx, metrics)
}

func (s *AIOpsService) GetCostAnalysis(ctx context.Context) (*CostSummary, error) {
	return s.costAnalysis.GetCostSummary(ctx)
}

func (s *AIOpsService) GetBudgetStatus(ctx context.Context) (*BudgetStatus, error) {
	return s.budgetAlert.GetBudgetStatus(ctx)
}

func (s *AIOpsService) GetHealthScore(ctx context.Context) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.healthScore
}

func (s *AIOpsService) ExportAnalysisReport(ctx context.Context, format string) ([]byte, error) {
	snapshot, err := s.PerformAnalysis(ctx)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return json.MarshalIndent(snapshot, "", "  ")
	case "html":
		return s.generateHTMLReport(snapshot)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func (s *AIOpsService) generateHTMLReport(snapshot *AnalysisSnapshot) ([]byte, error) {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>AIOps Analysis Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #007bff; color: white; padding: 20px; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; }
        .metric { display: inline-block; margin: 10px; padding: 10px; background: #f8f9fa; }
        .alert { padding: 10px; margin: 5px 0; border-left: 4px solid; }
        .critical { border-color: #dc3545; background: #f8d7da; }
        .warning { border-color: #ffc107; background: #fff3cd; }
    </style>
</head>
<body>
    <div class="header">
        <h1>AIOps Analysis Report</h1>
        <p>Generated at: %s</p>
        <p>Health Score: %.1f</p>
    </div>
    <div class="section">
        <h2>Metrics</h2>
        <div class="metric">CPU: %.1f%%</div>
        <div class="metric">Memory: %.1f%%</div>
        <div class="metric">Error Rate: %.1f%%</div>
        <div class="metric">Success Rate: %.1f%%</div>
    </div>
    <div class="section">
        <h2>Alerts (%d)</h2>
        %s
    </div>
    <div class="section">
        <h2>Predictions (%d)</h2>
        %s
    </div>
</body>
</html>
`, snapshot.Timestamp.Format(time.RFC3339), snapshot.HealthScore,
		snapshot.Metrics.CPUUsage, snapshot.Metrics.MemoryUsage,
		snapshot.Metrics.ErrorRate, snapshot.Metrics.SuccessRate,
		len(snapshot.Alerts), s.formatAlertsHTML(snapshot.Alerts),
		len(snapshot.Predictions), s.formatPredictionsHTML(snapshot.Predictions))

	return []byte(html), nil
}

func (s *AIOpsService) formatAlertsHTML(alerts []Alert) string {
	var html string
	for _, alert := range alerts {
		html += fmt.Sprintf(`<div class="alert %s">
            <strong>%s</strong> - %s<br>
            <small>%s</small>
        </div>`, alert.Severity, alert.Title, alert.Description, alert.Timestamp.Format(time.RFC3339))
	}
	return html
}

func (s *AIOpsService) formatPredictionsHTML(predictions []Prediction) string {
	var html string
	for _, pred := range predictions {
		html += fmt.Sprintf(`<div class="alert %s">
            <strong>%s</strong><br>
            Current: %.2f, Predicted: %.2f (Confidence: %.0f%%)<br>
            <small>%s horizon</small>
        </div>`, pred.AlertLevel, pred.MetricName,
			pred.CurrentValue, pred.PredictedValue, pred.Confidence*100, pred.TimeHorizon)
	}
	return html
}
