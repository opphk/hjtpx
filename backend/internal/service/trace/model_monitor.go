package trace

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type MonitorAlertLevel string

const (
	AlertLevelInfo     MonitorAlertLevel = "info"
	AlertLevelWarning  MonitorAlertLevel = "warning"
	AlertLevelCritical MonitorAlertLevel = "critical"
	AlertLevelError    MonitorAlertLevel = "error"
)

type MonitorAlert struct {
	AlertID     string            `json:"alert_id"`
	Level       MonitorAlertLevel `json:"level"`
	ModelType   string            `json:"model_type"`
	MetricName  string            `json:"metric_name"`
	Message     string            `json:"message"`
	Threshold   float64           `json:"threshold"`
	CurrentValue float64          `json:"current_value"`
	Timestamp   time.Time         `json:"timestamp"`
	Resolved    bool              `json:"resolved"`
}

type MonitorMetric struct {
	Name           string      `json:"name"`
	ModelType      string      `json:"model_type"`
	Value          float64     `json:"value"`
	Min            float64     `json:"min"`
	Max            float64     `json:"max"`
	Avg            float64     `json:"avg"`
	StdDev         float64     `json:"std_dev"`
	Count          int64       `json:"count"`
	LastUpdated    time.Time   `json:"last_updated"`
	Unit           string      `json:"unit"`
	AlertThreshold float64     `json:"alert_threshold"`
	AlertLevel     MonitorAlertLevel `json:"alert_level"`
}

type MonitorStats struct {
	TotalPredictions    int64
	TotalErrors         int64
	TotalLatency        time.Duration
	ActiveModels        int
	AlertCount          int
	MemoryUsageBytes    int64
	CPUUsagePercent     float64
	LastMaintenanceTime time.Time
}

type ModelPerformanceHistory struct {
	Timestamp         time.Time
	ModelType         string
	Accuracy          float64
	Precision         float64
	Recall            float64
	F1Score           float64
	AvgResponseTimeMs float64
	Throughput        float64
}

type ModelHealthStatus struct {
	ModelType      string      `json:"model_type"`
	IsHealthy      bool        `json:"is_healthy"`
	Status         string      `json:"status"`
	LastCheckTime  time.Time   `json:"last_check_time"`
	Issues         []string    `json:"issues"`
	Performance    ModelPerformanceMetrics `json:"performance"`
}

type EnhancedModelMonitor struct {
	mu                  sync.RWMutex
	metrics             map[string]*MonitorMetric
	alerts              []MonitorAlert
	performanceHistory  []ModelPerformanceHistory
	stats               MonitorStats
	isRunning           bool
	lastCleanupTime     time.Time
	alertHistory        []MonitorAlert
	maxHistorySize      int
	maxAlertHistorySize int
}

func NewEnhancedModelMonitor() *EnhancedModelMonitor {
	return &EnhancedModelMonitor{
		metrics:             make(map[string]*MonitorMetric),
		alerts:              make([]MonitorAlert, 0, 100),
		performanceHistory:  make([]ModelPerformanceHistory, 0, 1000),
		stats:               MonitorStats{},
		isRunning:           false,
		maxHistorySize:      1000,
		maxAlertHistorySize: 500,
	}
}

func (m *EnhancedModelMonitor) Start() {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return
	}
	m.isRunning = true
	m.stats.LastMaintenanceTime = time.Now()
	m.mu.Unlock()

	go m.runCleanupLoop()
}

func (m *EnhancedModelMonitor) Stop() {
	m.mu.Lock()
	m.isRunning = false
	m.mu.Unlock()
}

func (m *EnhancedModelMonitor) runCleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupOldData()
		default:
			if !m.isRunning {
				return
			}
			time.Sleep(time.Second)
		}
	}
}

func (m *EnhancedModelMonitor) cleanupOldData() {
	m.mu.Lock()
	defer m.mu.Unlock()

	ageThreshold := time.Now().Add(-24 * time.Hour)

	for len(m.performanceHistory) > 0 && m.performanceHistory[0].Timestamp.Before(ageThreshold) {
		m.performanceHistory = m.performanceHistory[1:]
	}

	for len(m.alerts) > 0 && m.alerts[0].Timestamp.Before(ageThreshold) {
		m.alerts = m.alerts[1:]
	}
}

func (m *EnhancedModelMonitor) RecordPrediction(modelType string, predicted, actual bool, responseTime time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.TotalPredictions++
	m.stats.TotalLatency += responseTime

	metricKey := modelType + "_predictions"
	if _, exists := m.metrics[metricKey]; !exists {
		m.metrics[metricKey] = &MonitorMetric{
			Name:      "predictions",
			ModelType: modelType,
			Unit:      "count",
		}
	}
	m.metrics[metricKey].Count++

	latencyKey := modelType + "_latency"
	if _, exists := m.metrics[latencyKey]; !exists {
		m.metrics[latencyKey] = &MonitorMetric{
			Name:      "latency",
			ModelType: modelType,
			Unit:      "ms",
		}
	}
	latencyMs := float64(responseTime.Milliseconds())
	m.updateMetric(latencyKey, latencyMs)
}

func (m *EnhancedModelMonitor) RecordError(modelType string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.TotalErrors++

	errorKey := modelType + "_errors"
	if _, exists := m.metrics[errorKey]; !exists {
		m.metrics[errorKey] = &MonitorMetric{
			Name:      "errors",
			ModelType: modelType,
			Unit:      "count",
		}
	}
	m.metrics[errorKey].Count++

	alert := MonitorAlert{
		AlertID:     generateAlertID(),
		Level:       AlertLevelError,
		ModelType:   modelType,
		MetricName:  "errors",
		Message:     err.Error(),
		Timestamp:   time.Now(),
		Resolved:    false,
	}
	m.alerts = append(m.alerts, alert)
	m.stats.AlertCount++
}

func (m *EnhancedModelMonitor) updateMetric(key string, value float64) {
	metric := m.metrics[key]
	
	if metric.Count == 0 {
		metric.Min = value
		metric.Max = value
		metric.Avg = value
	} else {
		metric.Min = minFloat(metric.Min, value)
		metric.Max = maxFloat(metric.Max, value)
		metric.Avg = (metric.Avg*float64(metric.Count) + value) / float64(metric.Count+1)
	}
	
	metric.Value = value
	metric.Count++
	metric.LastUpdated = time.Now()

	if metric.AlertThreshold > 0 && value > metric.AlertThreshold {
		m.triggerAlert(metric.ModelType, metric.Name, value, metric.AlertThreshold, metric.AlertLevel)
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func (m *EnhancedModelMonitor) SetAlertThreshold(modelType, metricName string, threshold float64, level MonitorAlertLevel) error {
	key := modelType + "_" + metricName
	
	m.mu.Lock()
	defer m.mu.Unlock()

	if metric, exists := m.metrics[key]; exists {
		metric.AlertThreshold = threshold
		metric.AlertLevel = level
		return nil
	}

	return errors.New("metric not found")
}

func (m *EnhancedModelMonitor) triggerAlert(modelType, metricName string, currentValue, threshold float64, level MonitorAlertLevel) {
	alert := MonitorAlert{
		AlertID:     generateAlertID(),
		Level:       level,
		ModelType:   modelType,
		MetricName:  metricName,
		Message:     buildAlertMessage(modelType, metricName, currentValue, threshold),
		Threshold:   threshold,
		CurrentValue: currentValue,
		Timestamp:   time.Now(),
		Resolved:    false,
	}

	m.alerts = append(m.alerts, alert)
	m.stats.AlertCount++

	if len(m.alerts) > m.maxAlertHistorySize {
		m.alerts = m.alerts[len(m.alerts)-m.maxAlertHistorySize:]
	}
}

func generateAlertID() string {
	return "alert_" + time.Now().Format("20060102150405") + "_" + randString(6)
}

func buildAlertMessage(modelType, metricName string, current, threshold float64) string {
	return "Alert: " + modelType + " " + metricName + " exceeded threshold. Current: " + 
		floatToString(current) + ", Threshold: " + floatToString(threshold)
}

func floatToString(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func (m *EnhancedModelMonitor) RecordPerformance(modelType string, accuracy, precision, recall, f1 float64, avgLatencyMs float64, throughput float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	history := ModelPerformanceHistory{
		Timestamp:         time.Now(),
		ModelType:         modelType,
		Accuracy:          accuracy,
		Precision:         precision,
		Recall:            recall,
		F1Score:           f1,
		AvgResponseTimeMs: avgLatencyMs,
		Throughput:        throughput,
	}

	m.performanceHistory = append(m.performanceHistory, history)

	if len(m.performanceHistory) > m.maxHistorySize {
		m.performanceHistory = m.performanceHistory[len(m.performanceHistory)-m.maxHistorySize:]
	}

	accKey := modelType + "_accuracy"
	if _, exists := m.metrics[accKey]; !exists {
		m.metrics[accKey] = &MonitorMetric{
			Name:      "accuracy",
			ModelType: modelType,
			Unit:      "%",
		}
	}
	m.updateMetric(accKey, accuracy*100)

	f1Key := modelType + "_f1_score"
	if _, exists := m.metrics[f1Key]; !exists {
		m.metrics[f1Key] = &MonitorMetric{
			Name:      "f1_score",
			ModelType: modelType,
			Unit:      "",
		}
	}
	m.updateMetric(f1Key, f1)

	latencyKey := modelType + "_avg_latency"
	if _, exists := m.metrics[latencyKey]; !exists {
		m.metrics[latencyKey] = &MonitorMetric{
			Name:      "avg_latency",
			ModelType: modelType,
			Unit:      "ms",
		}
	}
	m.updateMetric(latencyKey, avgLatencyMs)

	throughputKey := modelType + "_throughput"
	if _, exists := m.metrics[throughputKey]; !exists {
		m.metrics[throughputKey] = &MonitorMetric{
			Name:      "throughput",
			ModelType: modelType,
			Unit:      "req/s",
		}
	}
	m.updateMetric(throughputKey, throughput)
}

func (m *EnhancedModelMonitor) GetAlerts(level MonitorAlertLevel) []MonitorAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if level == "" {
		return append([]MonitorAlert(nil), m.alerts...)
	}

	result := make([]MonitorAlert, 0)
	for _, alert := range m.alerts {
		if alert.Level == level && !alert.Resolved {
			result = append(result, alert)
		}
	}
	return result
}

func (m *EnhancedModelMonitor) ResolveAlert(alertID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, alert := range m.alerts {
		if alert.AlertID == alertID {
			m.alerts[i].Resolved = true
			return nil
		}
	}

	return errors.New("alert not found")
}

func (m *EnhancedModelMonitor) GetMetrics() map[string]*MonitorMetric {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*MonitorMetric)
	for k, v := range m.metrics {
		result[k] = v
	}
	return result
}

func (m *EnhancedModelMonitor) GetMetric(modelType, metricName string) (*MonitorMetric, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := modelType + "_" + metricName
	if metric, exists := m.metrics[key]; exists {
		return metric, nil
	}

	return nil, errors.New("metric not found")
}

func (m *EnhancedModelMonitor) GetStats() MonitorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

func (m *EnhancedModelMonitor) GetPerformanceHistory(modelType string, start, end time.Time) []ModelPerformanceHistory {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ModelPerformanceHistory, 0)
	for _, history := range m.performanceHistory {
		if history.ModelType == modelType && 
			!history.Timestamp.Before(start) && 
			!history.Timestamp.After(end) {
			result = append(result, history)
		}
	}
	return result
}

func (m *EnhancedModelMonitor) GetHealthStatus(modelType string) *ModelHealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := &ModelHealthStatus{
		ModelType:     modelType,
		IsHealthy:     true,
		Status:        "healthy",
		LastCheckTime: time.Now(),
		Issues:        make([]string, 0),
	}

	accMetric, exists := m.metrics[modelType+"_accuracy"]
	if exists && accMetric.Avg < 70 {
		status.IsHealthy = false
		status.Status = "degraded"
		status.Issues = append(status.Issues, "accuracy below 70%")
	}

	errorMetric, exists := m.metrics[modelType+"_errors"]
	if exists && errorMetric.Count > 100 {
		status.IsHealthy = false
		status.Status = "error"
		status.Issues = append(status.Issues, "high error rate")
	}

	latencyMetric, exists := m.metrics[modelType+"_avg_latency"]
	if exists && latencyMetric.Avg > 1000 {
		status.IsHealthy = false
		status.Status = "slow"
		status.Issues = append(status.Issues, "latency exceeds 1s")
	}

	return status
}

func (m *EnhancedModelMonitor) GetComprehensiveReport() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report := make(map[string]interface{})

	report["stats"] = map[string]interface{}{
		"total_predictions": m.stats.TotalPredictions,
		"total_errors":      m.stats.TotalErrors,
		"total_latency_ms":  m.stats.TotalLatency.Milliseconds(),
		"active_models":     m.stats.ActiveModels,
		"alert_count":       m.stats.AlertCount,
		"memory_usage_bytes": m.stats.MemoryUsageBytes,
		"cpu_usage_percent":  m.stats.CPUUsagePercent,
		"last_maintenance":   m.stats.LastMaintenanceTime,
	}

	report["metrics"] = m.metrics

	activeAlerts := make([]MonitorAlert, 0)
	for _, alert := range m.alerts {
		if !alert.Resolved {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	report["active_alerts"] = activeAlerts

	report["model_health"] = make(map[string]interface{})
	modelTypes := []string{"lstm", "transformer", "yolo"}
	for _, modelType := range modelTypes {
		health := m.GetHealthStatusUnsafe(modelType)
		report["model_health"].(map[string]interface{})[modelType] = health
	}

	return report
}

func (m *EnhancedModelMonitor) GetHealthStatusUnsafe(modelType string) *ModelHealthStatus {
	status := &ModelHealthStatus{
		ModelType:     modelType,
		IsHealthy:     true,
		Status:        "healthy",
		LastCheckTime: time.Now(),
		Issues:        make([]string, 0),
	}

	accMetric, exists := m.metrics[modelType+"_accuracy"]
	if exists && accMetric.Avg < 70 {
		status.IsHealthy = false
		status.Status = "degraded"
		status.Issues = append(status.Issues, "accuracy below 70%")
	}

	errorMetric, exists := m.metrics[modelType+"_errors"]
	if exists && errorMetric.Count > 100 {
		status.IsHealthy = false
		status.Status = "error"
		status.Issues = append(status.Issues, "high error rate")
	}

	return status
}

func (m *EnhancedModelMonitor) UpdateResourceUsage(memoryBytes int64, cpuPercent float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.MemoryUsageBytes = memoryBytes
	m.stats.CPUUsagePercent = cpuPercent
}

func (m *EnhancedModelMonitor) UpdateActiveModels(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.ActiveModels = count
}

func (m *EnhancedModelMonitor) ResetStats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats = MonitorStats{
		LastMaintenanceTime: time.Now(),
	}
}

func (m *EnhancedModelMonitor) GetAlertHistory(modelType string) []MonitorAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]MonitorAlert, 0)
	for _, alert := range m.alerts {
		if alert.ModelType == modelType {
			result = append(result, alert)
		}
	}
	return result
}

func (m *EnhancedModelMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

func (m *EnhancedModelMonitor) GetAlertCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.alerts)
}

func (m *EnhancedModelMonitor) GetActiveAlertCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, alert := range m.alerts {
		if !alert.Resolved {
			count++
		}
	}
	return count
}