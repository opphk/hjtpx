package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type MonitoringMetric struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	MetricType  string    `json:"metric_type" gorm:"size:50;index"`
	MetricName  string    `json:"metric_name" gorm:"size:100;index"`
	Dimension   string    `json:"dimension" gorm:"size:100;index"`
	DimensionValue string `json:"dimension_value" gorm:"size:100"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit" gorm:"size:20"`
	Timestamp   time.Time `json:"timestamp" gorm:"index"`
	Metadata    string    `json:"metadata" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
}

type MonitoringAlert struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	AlertType     string    `json:"alert_type" gorm:"size:50;index"`
	AlertName     string    `json:"alert_name" gorm:"size:100"`
	Severity      string    `json:"severity" gorm:"size:20;index"`
	Message       string    `json:"message" gorm:"type:text"`
	MetricName    string    `json:"metric_name" gorm:"size:100"`
	Threshold     float64   `json:"threshold"`
	CurrentValue  float64   `json:"current_value"`
	Operator      string    `json:"operator" gorm:"size:10"`
	Status        string    `json:"status" gorm:"size:20;index"`
	AcknowledgedAt *time.Time `json:"acknowledged_at"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	AcknowledgedBy *uint     `json:"acknowledged_by"`
	ResolvedBy    *uint     `json:"resolved_by"`
	CreatedAt     time.Time `json:"created_at" gorm:"index"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type MonitoringDashboard struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:100"`
	Description string    `json:"description" gorm:"type:text"`
	Widgets     string    `json:"widgets" gorm:"type:text"`
	IsDefault   bool      `json:"is_default"`
	UserID      uint      `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MonitoringReport struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	ReportType    string    `json:"report_type" gorm:"size:50"`
	ReportName    string    `json:"report_name" gorm:"size:100"`
	PeriodStart   time.Time `json:"period_start"`
	PeriodEnd     time.Time `json:"period_end"`
	Summary       string    `json:"summary" gorm:"type:text"`
	Metrics       string    `json:"metrics" gorm:"type:text"`
	Recommendations string  `json:"recommendations" gorm:"type:text"`
	GeneratedAt   time.Time `json:"generated_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type MonitoringService struct {
	mu           sync.RWMutex
	alertThresholds map[string]map[string]float64
	metricsBuffer  []MonitoringMetric
	bufferSize     int
	flushInterval  time.Duration
}

var monitoringInstance *MonitoringService
var monitoringOnce sync.Once

func NewMonitoringService() *MonitoringService {
	monitoringOnce.Do(func() {
		monitoringInstance = &MonitoringService{
			alertThresholds: map[string]map[string]float64{
				"risk_score": {
					"critical": 20.0,
					"high":     40.0,
					"medium":   60.0,
				},
				"response_time": {
					"critical": 100.0,
					"high":     50.0,
					"medium":   20.0,
				},
				"block_rate": {
					"critical": 0.5,
					"high":     0.3,
					"medium":   0.15,
				},
				"false_positive_rate": {
					"critical": 0.1,
					"high":     0.05,
					"medium":   0.02,
				},
				"false_negative_rate": {
					"critical": 0.1,
					"high":     0.05,
					"medium":   0.02,
				},
			},
			metricsBuffer: make([]MonitoringMetric, 0, 1000),
			bufferSize:    1000,
			flushInterval: 30 * time.Second,
		}
		go monitoringInstance.startFlushWorker()
	})
	return monitoringInstance
}

func (s *MonitoringService) RecordMetric(ctx context.Context, metricType string, metricName string, value float64, dimension string, dimensionValue string, unit string, metadata map[string]interface{}) error {
	metric := MonitoringMetric{
		MetricType:     metricType,
		MetricName:      metricName,
		Dimension:       dimension,
		DimensionValue:  dimensionValue,
		Value:           value,
		Unit:            unit,
		Timestamp:       time.Now(),
	}

	if metadata != nil {
		metadataJSON, err := json.Marshal(metadata)
		if err == nil {
			metric.Metadata = string(metadataJSON)
		}
	}

	s.mu.Lock()
	s.metricsBuffer = append(s.metricsBuffer, metric)
	if len(s.metricsBuffer) >= s.bufferSize {
		s.flushMetrics()
	}
	s.mu.Unlock()

	s.cacheMetric(ctx, &metric)

	s.checkThresholds(ctx, metricName, value)

	return nil
}

func (s *MonitoringService) RecordRiskMetric(ctx context.Context, fingerprint string, ipAddress string, riskScore float64, action string, latency time.Duration) error {
	metadata := map[string]interface{}{
		"fingerprint": fingerprint,
		"ip_address":  ipAddress,
		"action":      action,
		"latency_ms":  latency.Milliseconds(),
	}

	s.RecordMetric(ctx, "risk", "risk_score", riskScore, "fingerprint", fingerprint, "score", metadata)
	s.RecordMetric(ctx, "risk", "risk_action", 1, "action", action, "count", metadata)
	s.RecordMetric(ctx, "performance", "risk_response_time", float64(latency.Milliseconds()), "fingerprint", fingerprint, "ms", nil)

	if riskScore < 40 {
		s.RecordMetric(ctx, "risk", "low_risk_requests", 1, "action", action, "count", nil)
	} else if riskScore < 60 {
		s.RecordMetric(ctx, "risk", "medium_risk_requests", 1, "action", action, "count", nil)
	} else {
		s.RecordMetric(ctx, "risk", "high_risk_requests", 1, "action", action, "count", nil)
	}

	return nil
}

func (s *MonitoringService) RecordBlockEvent(ctx context.Context, fingerprint string, ipAddress string, blockReason string) error {
	metadata := map[string]interface{}{
		"fingerprint":  fingerprint,
		"ip_address":   ipAddress,
		"block_reason": blockReason,
	}

	s.RecordMetric(ctx, "block", "blocked_requests", 1, "reason", blockReason, "count", metadata)

	var count int64
	database.DB.Model(&struct {
		tableName struct{} `gorm:"table:risk_events"`
	}{}).Where("created_at > ?", time.Now().Add(-1*time.Hour)).Count(&count)

	s.RecordMetric(ctx, "block", "hourly_block_rate", float64(count), "period", "1h", "count", nil)

	return nil
}

func (s *MonitoringService) RecordFalsePositive(ctx context.Context, fingerprint string, ipAddress string, actualOutcome string) error {
	metadata := map[string]interface{}{
		"fingerprint":     fingerprint,
		"ip_address":       ipAddress,
		"actual_outcome":   actualOutcome,
	}

	s.RecordMetric(ctx, "quality", "false_positive", 1, "type", "false_positive", "count", metadata)

	return nil
}

func (s *MonitoringService) RecordFalseNegative(ctx context.Context, fingerprint string, ipAddress string, attackType string) error {
	metadata := map[string]interface{}{
		"fingerprint": fingerprint,
		"ip_address":   ipAddress,
		"attack_type":  attackType,
	}

	s.RecordMetric(ctx, "quality", "false_negative", 1, "type", "false_negative", "count", metadata)

	return nil
}

func (s *MonitoringService) RecordModelPerformance(ctx context.Context, modelType string, accuracy float64, precision float64, recall float64, f1Score float64, latency time.Duration) error {
	metadata := map[string]interface{}{
		"accuracy":   accuracy,
		"precision": precision,
		"recall":    recall,
		"f1_score":  f1Score,
	}

	s.RecordMetric(ctx, "model", "accuracy", accuracy, "model", modelType, "percent", metadata)
	s.RecordMetric(ctx, "model", "precision", precision, "model", modelType, "percent", metadata)
	s.RecordMetric(ctx, "model", "recall", recall, "model", modelType, "percent", metadata)
	s.RecordMetric(ctx, "model", "f1_score", f1Score, "model", modelType, "percent", nil)
	s.RecordMetric(ctx, "performance", "model_latency", float64(latency.Milliseconds()), "model", modelType, "ms", nil)

	return nil
}

func (s *MonitoringService) RecordStrategyEffectiveness(ctx context.Context, strategyName string, action string, hitCount int64, blockCount int64, falsePositiveCount int64) error {
	effectiveness := float64(0)
	if hitCount > 0 {
		effectiveness = float64(blockCount) / float64(hitCount)
	}

	fpRate := float64(0)
	if blockCount > 0 {
		fpRate = float64(falsePositiveCount) / float64(blockCount)
	}

	metadata := map[string]interface{}{
		"action":               action,
		"hit_count":            hitCount,
		"block_count":          blockCount,
		"false_positive_count": falsePositiveCount,
	}

	s.RecordMetric(ctx, "strategy", "effectiveness", effectiveness, "strategy", strategyName, "rate", metadata)
	s.RecordMetric(ctx, "strategy", "false_positive_rate", fpRate, "strategy", strategyName, "rate", nil)

	return nil
}

func (s *MonitoringService) GetRealTimeMetrics(ctx context.Context, metricType string, timeRange time.Duration) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	keys, _ := redis.GetClient().Keys(ctx, fmt.Sprintf("metric:%s:*", metricType)).Result()

	for _, key := range keys {
		if data, err := redis.GetClient().Get(ctx, key).Result(); err == nil {
			var metric MonitoringMetric
			if json.Unmarshal([]byte(data), &metric) == nil {
				if time.Since(metric.Timestamp) <= timeRange {
					metrics[metric.MetricName] = metric.Value
				}
			}
		}
	}

	return metrics, nil
}

func (s *MonitoringService) GetRiskMetrics(ctx context.Context, startTime time.Time, endTime time.Time) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	var avgRiskScore float64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND timestamp BETWEEN ? AND ?", "risk", "risk_score", startTime, endTime).
		Select("AVG(value)").Scan(&avgRiskScore)
	metrics["avg_risk_score"] = avgRiskScore

	var totalRequests int64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND timestamp BETWEEN ? AND ?", "risk", "risk_action", startTime, endTime).
		Select("SUM(value)").Scan(&totalRequests)
	metrics["total_requests"] = totalRequests

	var blockedRequests int64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND dimension_value = ? AND timestamp BETWEEN ? AND ?", "block", "blocked_requests", "blacklist", startTime, endTime).
		Select("SUM(value)").Scan(&blockedRequests)
	metrics["blocked_requests"] = blockedRequests

	blockRate := float64(0)
	if totalRequests > 0 {
		blockRate = float64(blockedRequests) / float64(totalRequests)
	}
	metrics["block_rate"] = blockRate

	var falsePositives int64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND timestamp BETWEEN ? AND ?", "quality", startTime, endTime).
		Count(&falsePositives)
	metrics["false_positives"] = falsePositives

	fpRate := float64(0)
	if blockedRequests > 0 {
		fpRate = float64(falsePositives) / float64(blockedRequests)
	}
	metrics["false_positive_rate"] = fpRate

	var avgResponseTime float64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND timestamp BETWEEN ? AND ?", "performance", "risk_response_time", startTime, endTime).
		Select("AVG(value)").Scan(&avgResponseTime)
	metrics["avg_response_time_ms"] = avgResponseTime

	return metrics, nil
}

func (s *MonitoringService) GetStrategyPerformance(ctx context.Context, strategyName string, startTime time.Time, endTime time.Time) (map[string]interface{}, error) {
	performance := make(map[string]interface{})

	var effectiveness float64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND dimension_value = ? AND timestamp BETWEEN ? AND ?", "strategy", "effectiveness", strategyName, startTime, endTime).
		Select("AVG(value)").Scan(&effectiveness)
	performance["effectiveness"] = effectiveness

	var fpRate float64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND dimension_value = ? AND timestamp BETWEEN ? AND ?", "strategy", "false_positive_rate", strategyName, startTime, endTime).
		Select("AVG(value)").Scan(&fpRate)
	performance["false_positive_rate"] = fpRate

	var hitCount int64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND dimension_value = ? AND timestamp BETWEEN ? AND ?", "strategy", strategyName, startTime, endTime).
		Count(&hitCount)
	performance["total_hits"] = hitCount

	return performance, nil
}

func (s *MonitoringService) GetModelPerformance(ctx context.Context, modelType string, startTime time.Time, endTime time.Time) (map[string]interface{}, error) {
	performance := make(map[string]interface{})

	var accuracy float64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND dimension_value = ? AND timestamp BETWEEN ? AND ?", "model", "accuracy", modelType, startTime, endTime).
		Select("AVG(value)").Scan(&accuracy)
	performance["accuracy"] = accuracy

	var precision float64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND dimension_value = ? AND timestamp BETWEEN ? AND ?", "model", "precision", modelType, startTime, endTime).
		Select("AVG(value)").Scan(&precision)
	performance["precision"] = precision

	var recall float64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND dimension_value = ? AND timestamp BETWEEN ? AND ?", "model", "recall", modelType, startTime, endTime).
		Select("AVG(value)").Scan(&recall)
	performance["recall"] = recall

	var f1Score float64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND dimension_value = ? AND timestamp BETWEEN ? AND ?", "model", "f1_score", modelType, startTime, endTime).
		Select("AVG(value)").Scan(&f1Score)
	performance["f1_score"] = f1Score

	var avgLatency float64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND dimension_value = ? AND timestamp BETWEEN ? AND ?", "performance", "model_latency", modelType, startTime, endTime).
		Select("AVG(value)").Scan(&avgLatency)
	performance["avg_latency_ms"] = avgLatency

	return performance, nil
}

func (s *MonitoringService) GetRiskDistribution(ctx context.Context, startTime time.Time, endTime time.Time) (map[string]int64, error) {
	distribution := make(map[string]int64)

	var lowRisk int64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND value >= 80 AND timestamp BETWEEN ? AND ?", "risk", "risk_score", startTime, endTime).
		Count(&lowRisk)
	distribution["low"] = lowRisk

	var mediumRisk int64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND value >= 60 AND value < 80 AND timestamp BETWEEN ? AND ?", "risk", "risk_score", startTime, endTime).
		Count(&mediumRisk)
	distribution["medium"] = mediumRisk

	var highRisk int64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND value >= 40 AND value < 60 AND timestamp BETWEEN ? AND ?", "risk", "risk_score", startTime, endTime).
		Count(&highRisk)
	distribution["high"] = highRisk

	var criticalRisk int64
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND value < 40 AND timestamp BETWEEN ? AND ?", "risk", "risk_score", startTime, endTime).
		Count(&criticalRisk)
	distribution["critical"] = criticalRisk

	return distribution, nil
}

func (s *MonitoringService) GetActionDistribution(ctx context.Context, startTime time.Time, endTime time.Time) (map[string]int64, error) {
	distribution := make(map[string]int64)

	var actions []MonitoringMetric
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND timestamp BETWEEN ? AND ?", "risk", "risk_action", startTime, endTime).
		Find(&actions)

	for _, action := range actions {
		distribution[action.DimensionValue]++
	}

	return distribution, nil
}

func (s *MonitoringService) GetTrendData(ctx context.Context, metricType string, metricName string, startTime time.Time, endTime time.Time, interval time.Duration) ([]map[string]interface{}, error) {
	var metrics []MonitoringMetric
	database.DB.Model(&MonitoringMetric{}).
		Where("metric_type = ? AND metric_name = ? AND timestamp BETWEEN ? AND ?", metricType, metricName, startTime, endTime).
		Order("timestamp ASC").
		Find(&metrics)

	trends := make([]map[string]interface{}, 0)
	for _, metric := range metrics {
		trends = append(trends, map[string]interface{}{
			"timestamp": metric.Timestamp,
			"value":     metric.Value,
			"unit":      metric.Unit,
		})
	}

	return trends, nil
}

func (s *MonitoringService) CreateAlert(ctx context.Context, alertType string, alertName string, severity string, message string, metricName string, threshold float64, operator string) (*MonitoringAlert, error) {
	alert := &MonitoringAlert{
		AlertType:    alertType,
		AlertName:    alertName,
		Severity:     severity,
		Message:      message,
		MetricName:   metricName,
		Threshold:    threshold,
		Operator:     operator,
		Status:       "active",
		CurrentValue: 0,
	}

	if err := database.DB.Create(alert).Error; err != nil {
		return nil, err
	}

	s.notifyAlert(ctx, alert)

	return alert, nil
}

func (s *MonitoringService) GetActiveAlerts(ctx context.Context) ([]MonitoringAlert, error) {
	var alerts []MonitoringAlert
	err := database.DB.Where("status IN ?", []string{"active", "acknowledged"}).Order("severity DESC, created_at DESC").Find(&alerts).Error
	return alerts, err
}

func (s *MonitoringService) GetAlertHistory(ctx context.Context, startTime time.Time, endTime time.Time, severity string) ([]MonitoringAlert, error) {
	var alerts []MonitoringAlert
	query := database.DB.Where("created_at BETWEEN ? AND ?", startTime, endTime)

	if severity != "" {
		query = query.Where("severity = ?", severity)
	}

	err := query.Order("created_at DESC").Find(&alerts).Error
	return alerts, err
}

func (s *MonitoringService) AcknowledgeAlert(ctx context.Context, alertID uint, userID uint) error {
	now := time.Now()
	return database.DB.Model(&MonitoringAlert{}).Where("id = ?", alertID).Updates(map[string]interface{}{
		"status":           "acknowledged",
		"acknowledged_at":  now,
		"acknowledged_by":  userID,
	}).Error
}

func (s *MonitoringService) ResolveAlert(ctx context.Context, alertID uint, userID uint) error {
	now := time.Now()
	return database.DB.Model(&MonitoringAlert{}).Where("id = ?", alertID).Updates(map[string]interface{}{
		"status":        "resolved",
		"resolved_at":   now,
		"resolved_by":   userID,
	}).Error
}

func (s *MonitoringService) GenerateReport(ctx context.Context, reportType string, startTime time.Time, endTime time.Time) (*MonitoringReport, error) {
	report := &MonitoringReport{
		ReportType:  reportType,
		ReportName:  fmt.Sprintf("%s报告_%s_%s", reportType, startTime.Format("20060102"), endTime.Format("20060102")),
		PeriodStart: startTime,
		PeriodEnd:   endTime,
		GeneratedAt: time.Now(),
	}

	metrics, _ := s.GetRiskMetrics(ctx, startTime, endTime)
	metricsJSON, _ := json.Marshal(metrics)
	report.Metrics = string(metricsJSON)

	var summaryBuilder string
	if avgScore, ok := metrics["avg_risk_score"].(float64); ok {
		summaryBuilder += fmt.Sprintf("平均风险评分: %.2f; ", avgScore)
	}
	if blockRate, ok := metrics["block_rate"].(float64); ok {
		summaryBuilder += fmt.Sprintf("拦截率: %.2f%%; ", blockRate*100)
	}
	if fpRate, ok := metrics["false_positive_rate"].(float64); ok {
		summaryBuilder += fmt.Sprintf("误报率: %.2f%%; ", fpRate*100)
	}
	report.Summary = summaryBuilder

	var recommendations []string
	if blockRate, ok := metrics["block_rate"].(float64); ok && blockRate > 0.3 {
		recommendations = append(recommendations, "拦截率过高，建议检查风控规则是否过于严格")
	}
	if fpRate, ok := metrics["false_positive_rate"].(float64); ok && fpRate > 0.05 {
		recommendations = append(recommendations, "误报率较高，建议优化风控策略以减少误报")
	}
	if recommendationsJSON, _ := json.Marshal(recommendations); recommendationsJSON != nil {
		report.Recommendations = string(recommendationsJSON)
	}

	database.DB.Create(report)

	return report, nil
}

func (s *MonitoringService) GetReports(ctx context.Context, reportType string, limit int) ([]MonitoringReport, error) {
	var reports []MonitoringReport
	query := database.DB.Order("generated_at DESC")

	if reportType != "" {
		query = query.Where("report_type = ?", reportType)
	}

	err := query.Limit(limit).Find(&reports).Error
	return reports, err
}

func (s *MonitoringService) cacheMetric(ctx context.Context, metric *MonitoringMetric) {
	key := fmt.Sprintf("metric:%s:%s:%d", metric.MetricType, metric.MetricName, metric.Timestamp.Unix())
	data, _ := json.Marshal(metric)
	redis.GetClient().Set(ctx, key, data, 24*time.Hour)
}

func (s *MonitoringService) checkThresholds(ctx context.Context, metricName string, value float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if thresholds, exists := s.alertThresholds[metricName]; exists {
		severity := ""
		threshold := float64(0)

		if value < thresholds["critical"] {
			severity = "critical"
			threshold = thresholds["critical"]
		} else if value < thresholds["high"] {
			severity = "high"
			threshold = thresholds["high"]
		} else if value < thresholds["medium"] {
			severity = "medium"
			threshold = thresholds["medium"]
		}

		if severity != "" {
			s.CreateAlert(ctx, "threshold", fmt.Sprintf("%s告警", metricName), severity,
				fmt.Sprintf("%s指标低于阈值: 当前值%.2f, 阈值%.2f", metricName, value, threshold),
				metricName, threshold, "<")
		}
	}
}

func (s *MonitoringService) flushMetrics() {
	if len(s.metricsBuffer) == 0 {
		return
	}

	database.DB.Create(&s.metricsBuffer)

	s.metricsBuffer = make([]MonitoringMetric, 0, s.bufferSize)
}

func (s *MonitoringService) startFlushWorker() {
	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		s.flushMetrics()
		s.mu.Unlock()
	}
}

func (s *MonitoringService) notifyAlert(ctx context.Context, alert *MonitoringAlert) {
	alertJSON, _ := json.Marshal(alert)
	redis.GetClient().Publish(ctx, "monitoring:alerts", alertJSON)

	redis.GetClient().LPush(ctx, fmt.Sprintf("alerts:%s", alert.Severity), alertJSON)
	redis.GetClient().LTrim(ctx, fmt.Sprintf("alerts:%s", alert.Severity), 0, 999)
}

func (s *MonitoringService) UpdateAlertThreshold(metricName string, severity string, threshold float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.alertThresholds[metricName]; !exists {
		s.alertThresholds[metricName] = make(map[string]float64)
	}

	s.alertThresholds[metricName][severity] = threshold

	ctx := context.Background()
	thresholdsJSON, _ := json.Marshal(s.alertThresholds)
	redis.GetClient().Set(ctx, "monitoring:thresholds", thresholdsJSON, 0)

	return nil
}

func (s *MonitoringService) GetAlertThresholds() map[string]map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]map[string]float64)
	for k, v := range s.alertThresholds {
		result[k] = v
	}
	return result
}
