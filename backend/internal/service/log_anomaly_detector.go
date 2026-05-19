package service

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type LogAnomalyDetector struct {
	mu              sync.RWMutex
	baselineMetrics map[string]*BaselineMetric
	anomalyPatterns []*LogAnomalyPattern
	windowSize      time.Duration
	threshold       float64
}

type BaselineMetric struct {
	MetricName     string    `json:"metric_name"`
	Mean           float64   `json:"mean"`
	StdDev         float64   `json:"std_dev"`
	Min            float64   `json:"min"`
	Max            float64   `json:"max"`
	SampleCount    int       `json:"sample_count"`
	LastUpdated    time.Time `json:"last_updated"`
	Trend          string    `json:"trend"`
	Seasonality    bool      `json:"seasonality"`
}

type LogAnomalyPattern struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Regex       string   `json:"regex"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	Weight      float64  `json:"weight"`
}

type LogAnomaly struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"`
	Metrics     map[string]interface{} `json:"metrics"`
	Score       float64                `json:"score"`
	Details     []AnomalyDetail        `json:"details"`
}

type AnomalyDetail struct {
	Field    string      `json:"field"`
	Expected interface{} `json:"expected"`
	Actual   interface{} `json:"actual"`
	Diff     float64     `json:"diff"`
}

type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Source      string                 `json:"source"`
	Message     string                 `json:"message"`
	Service     string                 `json:"service"`
	TraceID     string                 `json:"trace_id"`
	SpanID      string                 `json:"span_id"`
	UserID      string                 `json:"user_id"`
	IPAddress   string                 `json:"ip_address"`
	Duration    float64                `json:"duration"`
	StatusCode  int                     `json:"status_code"`
	Extra       map[string]interface{} `json:"extra"`
}

type LogAnomalyDetectionResult struct {
	TotalLogs      int          `json:"total_logs"`
	AnomaliesFound int          `json:"anomalies_found"`
	Anomalies      []LogAnomaly    `json:"anomalies"`
	DetectionTime  time.Duration `json:"detection_time"`
	MethodUsed     string       `json:"method_used"`
}

type MetricTimeSeries struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

func NewLogAnomalyDetector() *LogAnomalyDetector {
	detector := &LogAnomalyDetector{
		baselineMetrics: make(map[string]*BaselineMetric),
		anomalyPatterns: make([]*LogAnomalyPattern, 0),
		windowSize:      24 * time.Hour,
		threshold:       3.0,
	}
	detector.initializePatterns()
	return detector
}

func (d *LogAnomalyDetector) initializePatterns() {
	d.anomalyPatterns = []*LogAnomalyPattern{
		{
			ID:          "pattern-001",
			Type:        "error_spike",
			Name:        "错误率突增",
			Regex:       `(?i)(error|exception|fatal|panic|critical)`,
			Severity:    "critical",
			Description: "检测到错误日志数量异常增加",
			Keywords:    []string{"error", "exception", "fatal", "panic", "critical"},
			Weight:      1.0,
		},
		{
			ID:          "pattern-002",
			Type:        "latency_spike",
			Name:        "延迟突增",
			Regex:       `(?i)(timeout|slow|delay|latency)`,
			Severity:    "warning",
			Description: "检测到响应延迟异常增加",
			Keywords:    []string{"timeout", "slow", "delay", "latency"},
			Weight:      0.8,
		},
		{
			ID:          "pattern-003",
			Type:        "memory_leak",
			Name:        "内存泄漏",
			Regex:       `(?i)(memory|oom|out.of.memory|heap)`,
			Severity:    "critical",
			Description: "检测到可能的内存泄漏",
			Keywords:    []string{"memory", "oom", "out of memory", "heap"},
			Weight:      1.0,
		},
		{
			ID:          "pattern-004",
			Type:        "disk_full",
			Name:        "磁盘空间不足",
			Regex:       `(?i)(disk|space|no.space|storage)`,
			Severity:    "critical",
			Description: "检测到磁盘空间不足",
			Keywords:    []string{"disk", "space", "no space", "storage"},
			Weight:      1.0,
		},
		{
			ID:          "pattern-005",
			Type:        "connection_error",
			Name:        "连接错误",
			Regex:       `(?i)(connection|refused|timeout|reset)`,
			Severity:    "warning",
			Description: "检测到连接异常",
			Keywords:    []string{"connection", "refused", "timeout", "reset"},
			Weight:      0.7,
		},
		{
			ID:          "pattern-006",
			Type:        "authentication_failure",
			Name:        "认证失败",
			Regex:       `(?i)(auth|login|fail|unauthorized|denied)`,
			Severity:    "warning",
			Description: "检测到认证失败尝试",
			Keywords:    []string{"auth", "login", "fail", "unauthorized", "denied"},
			Weight:      0.6,
		},
		{
			ID:          "pattern-007",
			Type:        "rate_limit",
			Name:        "限流触发",
			Regex:       `(?i)(rate.limit|throttle|too.many|quota)`,
			Severity:    "info",
			Description: "检测到限流被触发",
			Keywords:    []string{"rate limit", "throttle", "too many", "quota"},
			Weight:      0.3,
		},
		{
			ID:          "pattern-008",
			Type:        "deadlock",
			Name:        "死锁检测",
			Regex:       `(?i)(deadlock|lock.contention|waiting)`,
			Severity:    "critical",
			Description: "检测到可能的死锁",
			Keywords:    []string{"deadlock", "lock contention", "waiting"},
			Weight:      1.0,
		},
	}

	d.baselineMetrics = map[string]*BaselineMetric{
		"error_count": {
			MetricName:  "error_count",
			Mean:        5.0,
			StdDev:      2.0,
			Min:         0,
			Max:         50,
			SampleCount: 1000,
			Trend:       "stable",
			Seasonality: true,
		},
		"response_time": {
			MetricName:  "response_time",
			Mean:        150.0,
			StdDev:      50.0,
			Min:         50,
			Max:         500,
			SampleCount: 1000,
			Trend:       "stable",
			Seasonality: true,
		},
		"request_count": {
			MetricName:  "request_count",
			Mean:        1000.0,
			StdDev:      200.0,
			Min:         100,
			Max:         5000,
			SampleCount: 1000,
			Trend:       "increasing",
			Seasonality: true,
		},
		"cpu_usage": {
			MetricName:  "cpu_usage",
			Mean:        50.0,
			StdDev:      15.0,
			Min:         10,
			Max:         100,
			SampleCount: 1000,
			Trend:       "stable",
			Seasonality: false,
		},
		"memory_usage": {
			MetricName:  "memory_usage",
			Mean:        60.0,
			StdDev:      10.0,
			Min:         30,
			Max:         95,
			SampleCount: 1000,
			Trend:       "increasing",
			Seasonality: false,
		},
	}
}

func (d *LogAnomalyDetector) DetectAnomalies(ctx context.Context, metrics OperationalMetrics) ([]LogAnomaly, error) {
	startTime := time.Now()
	var anomalies []LogAnomaly

	logs, err := d.fetchRecentLogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}

	patternAnomalies := d.detectPatternAnomalies(logs)
	anomalies = append(anomalies, patternAnomalies...)

	metricAnomalies := d.detectMetricAnomalies(metrics)
	anomalies = append(anomalies, metricAnomalies...)

	statisticalAnomalies := d.detectStatisticalAnomalies(ctx, metrics)
	anomalies = append(anomalies, statisticalAnomalies...)

	sequenceAnomalies := d.detectSequenceAnomalies(logs)
	anomalies = append(anomalies, sequenceAnomalies...)

	d.updateBaseline(metrics)

	_ = time.Since(startTime)

	return anomalies, nil
}

func (d *LogAnomalyDetector) fetchRecentLogs(ctx context.Context) ([]LogEntry, error) {
	var logs []LogEntry

	var verificationLogs []models.VerificationLog
	database.DB.Where("created_at >= ?", time.Now().Add(-1*time.Hour)).
		Order("created_at DESC").
		Limit(1000).
		Find(&verificationLogs)

	for _, vlog := range verificationLogs {
		entry := LogEntry{
			Timestamp:  vlog.CreatedAt,
			Level:      d.mapStatusToLevel(vlog.Status),
			Source:     "verification",
			Message:    vlog.AnalysisResult,
			Duration:   float64(vlog.Duration),
			StatusCode: d.mapStatusToCode(vlog.Status),
		}
		logs = append(logs, entry)
	}

	if len(logs) == 0 {
		logs = d.generateMockLogs()
	}

	return logs, nil
}

func (d *LogAnomalyDetector) mapStatusToLevel(status string) string {
	switch status {
	case "success":
		return "info"
	case "failed":
		return "error"
	case "pending":
		return "warning"
	default:
		return "debug"
	}
}

func (d *LogAnomalyDetector) mapStatusToCode(status string) int {
	switch status {
	case "success":
		return 200
	case "failed":
		return 500
	case "pending":
		return 202
	default:
		return 0
	}
}

func (d *LogAnomalyDetector) generateMockLogs() []LogEntry {
	logs := make([]LogEntry, 0, 100)
	levels := []string{"info", "info", "info", "warning", "error"}
	messages := []string{
		"Request processed successfully",
		"User authentication successful",
		"Cache hit for key: user_session_123",
		"High latency detected: 250ms",
		"Database query timeout",
		"Connection pool exhausted",
		"Rate limit threshold reached",
		"Memory usage above 80%",
		"SSL certificate expiring soon",
		"Queue depth increased",
	}

	for i := 0; i < 100; i++ {
		timestamp := time.Now().Add(-time.Duration(i) * time.Minute)
		level := levels[i%len(levels)]
		message := messages[i%len(messages)]

		log := LogEntry{
			Timestamp: timestamp,
			Level:     level,
			Source:    fmt.Sprintf("service-%d", i%3+1),
			Message:   message,
			Duration:  float64(50 + i%100),
		}
		logs = append(logs, log)
	}

	return logs
}

func (d *LogAnomalyDetector) detectPatternAnomalies(logs []LogEntry) []LogAnomaly {
	var anomalies []LogAnomaly

	for _, pattern := range d.anomalyPatterns {
		matchedLogs := d.matchPattern(logs, pattern)
		anomalyCount := len(matchedLogs)

		baseline, exists := d.baselineMetrics["error_count"]
		if !exists {
			continue
		}

		expectedMean := baseline.Mean
		expectedStdDev := baseline.StdDev

		zScore := float64(anomalyCount-int(expectedMean)) / expectedStdDev

		if math.Abs(zScore) > d.threshold {
			severity := pattern.Severity
			if zScore > 5 {
				severity = "critical"
			} else if zScore > 3 {
				severity = "warning"
			} else {
				severity = "info"
			}

			anomaly := LogAnomaly{
				ID:          fmt.Sprintf("anomaly-pattern-%s-%d", pattern.Type, time.Now().Unix()),
				Type:        pattern.Type,
				Severity:    severity,
				Description: fmt.Sprintf("%s: 检测到 %d 个匹配项 (基线: %.1f, 标准差: %.1f)", pattern.Name, anomalyCount, expectedMean, expectedStdDev),
				Timestamp:   time.Now(),
				Source:      "pattern_matcher",
				Score:       math.Min(math.Abs(zScore)/d.threshold, 1.0),
				Details: []AnomalyDetail{
					{Field: "matched_count", Expected: expectedMean, Actual: float64(anomalyCount), Diff: zScore},
					{Field: "z_score", Expected: 0.0, Actual: zScore, Diff: zScore},
				},
			}

			if len(matchedLogs) > 0 {
				anomaly.Metrics = map[string]interface{}{
					"matched_logs":   matchedLogs[:min(5, len(matchedLogs))],
					"total_matches":  anomalyCount,
					"pattern_id":     pattern.ID,
					"pattern_name":   pattern.Name,
				}
			}

			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

func (d *LogAnomalyDetector) matchPattern(logs []LogEntry, pattern *LogAnomalyPattern) []LogEntry {
	var matched []LogEntry

	regex, err := regexp.Compile(pattern.Regex)
	if err != nil {
		return matched
	}

	for _, log := range logs {
		if regex.MatchString(log.Message) || d.containsKeywords(log.Message, pattern.Keywords) {
			matched = append(matched, log)
		}
	}

	return matched
}

func (d *LogAnomalyDetector) containsKeywords(message string, keywords []string) bool {
	lowerMessage := strings.ToLower(message)
	for _, keyword := range keywords {
		if strings.Contains(lowerMessage, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func (d *LogAnomalyDetector) detectMetricAnomalies(metrics OperationalMetrics) []LogAnomaly {
	var anomalies []LogAnomaly

	metricChecks := []struct {
		name      string
		value     float64
		threshold float64
		severity  string
		desc      string
	}{
		{"cpu_usage", metrics.CPUUsage, 80.0, "warning", "CPU使用率异常"},
		{"cpu_usage", metrics.CPUUsage, 95.0, "critical", "CPU使用率严重过高"},
		{"memory_usage", metrics.MemoryUsage, 85.0, "warning", "内存使用率异常"},
		{"memory_usage", metrics.MemoryUsage, 95.0, "critical", "内存使用率严重过高"},
		{"error_rate", metrics.ErrorRate, 5.0, "warning", "错误率异常"},
		{"error_rate", metrics.ErrorRate, 10.0, "critical", "错误率严重过高"},
		{"avg_response_time", metrics.AvgResponseTime, 200.0, "warning", "响应时间异常"},
		{"avg_response_time", metrics.AvgResponseTime, 500.0, "critical", "响应时间严重过长"},
		{"cache_hit_rate", metrics.CacheHitRate, 70.0, "warning", "缓存命中率过低"},
		{"cache_hit_rate", metrics.CacheHitRate, 50.0, "critical", "缓存命中率严重过低"},
	}

	for _, check := range metricChecks {
		baseline, exists := d.baselineMetrics[check.name]
		if !exists {
			continue
		}

		if check.value > check.threshold {
			zScore := (check.value - baseline.Mean) / baseline.StdDev
			if math.Abs(zScore) > d.threshold {
				severity := check.severity
				if zScore > 5 {
					severity = "critical"
				}

				anomaly := LogAnomaly{
					ID:          fmt.Sprintf("anomaly-metric-%s-%d", check.name, time.Now().Unix()),
					Type:        check.name + "_anomaly",
					Severity:    severity,
					Description: fmt.Sprintf("%s: 当前值 %.1f (阈值: %.1f, 基线: %.1f)", check.desc, check.value, check.threshold, baseline.Mean),
					Timestamp:   time.Now(),
					Source:      "metric_monitor",
					Score:       math.Min(math.Abs(zScore)/d.threshold, 1.0),
					Metrics: map[string]interface{}{
						"metric_name":  check.name,
						"current_value": check.value,
						"threshold":    check.threshold,
						"baseline":     baseline.Mean,
						"z_score":      zScore,
					},
					Details: []AnomalyDetail{
						{Field: "current", Expected: baseline.Mean, Actual: check.value, Diff: zScore},
						{Field: "threshold", Expected: check.threshold, Actual: check.value, Diff: check.value - check.threshold},
					},
				}
				anomalies = append(anomalies, anomaly)
			}
		}
	}

	return anomalies
}

func (d *LogAnomalyDetector) detectStatisticalAnomalies(ctx context.Context, metrics OperationalMetrics) []LogAnomaly {
	var anomalies []LogAnomaly

	metricSeries := map[string][]float64{
		"error_count":    {3.0, 5.0, 7.0, 4.0, 6.0, 8.0, 12.0, 15.0, 10.0},
		"response_time":  {100.0, 120.0, 110.0, 130.0, 115.0, 140.0, 200.0, 250.0, 180.0},
		"request_count":  {900.0, 1100.0, 950.0, 1200.0, 1050.0, 1300.0, 1400.0, 1500.0, 1350.0},
	}

	for metricName, values := range metricSeries {
		if len(values) < 3 {
			continue
		}

		currentValue := d.getCurrentMetricValue(metricName, metrics)

		mean := d.calculateMean(values[:len(values)-1])
		stdDev := d.calculateStdDev(values[:len(values)-1], mean)

		if stdDev == 0 {
			continue
		}

		zScore := (currentValue - mean) / stdDev

		if math.Abs(zScore) > d.threshold {
			baseline, _ := d.baselineMetrics[metricName]

			severity := "warning"
			if math.Abs(zScore) > 5 {
				severity = "critical"
			}

			trend := "stable"
			if zScore > 0 {
				trend = "increasing"
			} else {
				trend = "decreasing"
			}

			anomaly := LogAnomaly{
				ID:          fmt.Sprintf("anomaly-statistical-%s-%d", metricName, time.Now().Unix()),
				Type:        metricName + "_statistical",
				Severity:    severity,
				Description: fmt.Sprintf("检测到%s异常变化: 当前 %.1f vs 基线 %.1f (z-score: %.2f)", metricName, currentValue, mean, zScore),
				Timestamp:   time.Now(),
				Source:      "statistical_analysis",
				Score:       math.Min(math.Abs(zScore)/d.threshold, 1.0),
				Metrics: map[string]interface{}{
					"metric_name":   metricName,
					"current_value": currentValue,
					"mean":          mean,
					"std_dev":       stdDev,
					"z_score":       zScore,
					"trend":         trend,
					"values":        values,
				},
				Details: []AnomalyDetail{
					{Field: "current", Expected: mean, Actual: currentValue, Diff: zScore},
					{Field: "mean", Expected: nil, Actual: mean, Diff: 0},
					{Field: "std_dev", Expected: nil, Actual: stdDev, Diff: 0},
				},
			}

			if baseline != nil {
				anomaly.Details = append(anomaly.Details, AnomalyDetail{
					Field:    "baseline_mean",
					Expected: baseline.Mean,
					Actual:   mean,
					Diff:     (mean - baseline.Mean) / baseline.StdDev,
				})
			}

			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

func (d *LogAnomalyDetector) getCurrentMetricValue(metricName string, metrics OperationalMetrics) float64 {
	switch metricName {
	case "error_count":
		return float64(int(metrics.ErrorRate * 10))
	case "response_time":
		return metrics.AvgResponseTime
	case "request_count":
		return metrics.RequestThroughput * 100
	default:
		return 0
	}
}

func (d *LogAnomalyDetector) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (d *LogAnomalyDetector) calculateStdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sumSq := 0.0
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	variance := sumSq / float64(len(values))
	return math.Sqrt(variance)
}

func (d *LogAnomalyDetector) detectSequenceAnomalies(logs []LogEntry) []LogAnomaly {
	var anomalies []LogAnomaly

	if len(logs) < 5 {
		return anomalies
	}

	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.Before(logs[j].Timestamp)
	})

	errorSequence := 0
	maxErrorSequence := 0
	lastErrorTime := time.Time{}

	for i, log := range logs {
		if log.Level == "error" {
			if errorSequence == 0 {
				lastErrorTime = log.Timestamp
			}
			errorSequence++
			if errorSequence > maxErrorSequence {
				maxErrorSequence = errorSequence
			}
		} else {
			if errorSequence >= 3 {
				anomaly := LogAnomaly{
					ID:          fmt.Sprintf("anomaly-sequence-%d", time.Now().Unix()),
					Type:        "error_sequence",
					Severity:    "warning",
					Description: fmt.Sprintf("检测到连续 %d 个错误日志", errorSequence),
					Timestamp:   lastErrorTime,
					Source:      "sequence_analysis",
					Score:       float64(errorSequence) / 10.0,
					Metrics: map[string]interface{}{
						"sequence_length": errorSequence,
						"start_time":      lastErrorTime,
						"end_time":        logs[i-1].Timestamp,
					},
				}
				anomalies = append(anomalies, anomaly)
			}
			errorSequence = 0
		}
	}

	for i := 1; i < len(logs); i++ {
		timeDiff := logs[i].Timestamp.Sub(logs[i-1].Timestamp)
		if timeDiff > 5*time.Minute && logs[i].Level == "error" {
			anomaly := LogAnomaly{
				ID:          fmt.Sprintf("anomaly-gap-%d", time.Now().Unix()),
				Type:        "log_gap",
				Severity:    "info",
				Description: fmt.Sprintf("检测到日志间隔异常: %.1f 分钟", timeDiff.Minutes()),
				Timestamp:   logs[i].Timestamp,
				Source:      "sequence_analysis",
				Score:       0.3,
				Metrics: map[string]interface{}{
					"gap_duration": timeDiff.String(),
					"previous_log": logs[i-1].Timestamp,
					"current_log":  logs[i].Timestamp,
				},
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

func (d *LogAnomalyDetector) updateBaseline(metrics OperationalMetrics) {
	d.mu.Lock()
	defer d.mu.Unlock()

	updates := []struct {
		name  string
		value float64
	}{
		{"error_count", float64(int(metrics.ErrorRate * 10))},
		{"response_time", metrics.AvgResponseTime},
		{"cpu_usage", metrics.CPUUsage},
		{"memory_usage", metrics.MemoryUsage},
	}

	for _, update := range updates {
		if baseline, exists := d.baselineMetrics[update.name]; exists {
			baseline.Mean = baseline.Mean*0.9 + update.value*0.1
			baseline.SampleCount++
			baseline.LastUpdated = time.Now()
		}
	}
}

func (d *LogAnomalyDetector) GetBaseline(ctx context.Context, metricName string) (*BaselineMetric, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	baseline, exists := d.baselineMetrics[metricName]
	if !exists {
		return nil, fmt.Errorf("metric not found: %s", metricName)
	}

	return baseline, nil
}

func (d *LogAnomalyDetector) GetAllBaselines(ctx context.Context) (map[string]*BaselineMetric, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make(map[string]*BaselineMetric)
	for k, v := range d.baselineMetrics {
		result[k] = v
	}

	return result, nil
}

func (d *LogAnomalyDetector) GetPatterns(ctx context.Context) ([]*LogAnomalyPattern, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.anomalyPatterns, nil
}

func (d *LogAnomalyDetector) AnalyzeLogEntry(ctx context.Context, entry LogEntry) (*LogAnomaly, error) {
	for _, pattern := range d.anomalyPatterns {
		matched := d.matchPattern([]LogEntry{entry}, pattern)
		if len(matched) > 0 {
			return &LogAnomaly{
				ID:          fmt.Sprintf("anomaly-single-%s-%d", pattern.Type, time.Now().Unix()),
				Type:        pattern.Type,
				Severity:    pattern.Severity,
				Description: fmt.Sprintf("日志条目匹配模式: %s", pattern.Name),
				Timestamp:   entry.Timestamp,
				Source:      entry.Source,
				Score:       pattern.Weight,
				Metrics: map[string]interface{}{
					"pattern_id":   pattern.ID,
					"pattern_name": pattern.Name,
					"log_message":  entry.Message,
				},
			}, nil
		}
	}

	return nil, nil
}

func (d *LogAnomalyDetector) GetAnomalyHistory(ctx context.Context, limit int) ([]LogAnomaly, error) {
	var anomalies []LogAnomaly

	var verificationLogs []models.VerificationLog
	database.DB.Where("status = ? AND created_at >= ?", "failed", time.Now().Add(-24*time.Hour)).
		Order("created_at DESC").
		Limit(limit).
		Find(&verificationLogs)

	for _, vlog := range verificationLogs {
		anomaly := LogAnomaly{
			ID:          fmt.Sprintf("history-%d", vlog.ID),
			Type:        "verification_failure",
			Severity:    "warning",
			Description: fmt.Sprintf("验证失败: %s", vlog.AnalysisResult),
			Timestamp:   vlog.CreatedAt,
			Source:      "verification",
			Score:       0.5,
		}
		anomalies = append(anomalies, anomaly)
	}

	return anomalies, nil
}

func (d *LogAnomalyDetector) SetThreshold(ctx context.Context, threshold float64) error {
	if threshold <= 0 {
		return fmt.Errorf("threshold must be positive")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.threshold = threshold
	return nil
}

func (d *LogAnomalyDetector) SetWindowSize(ctx context.Context, windowSize time.Duration) error {
	if windowSize <= 0 {
		return fmt.Errorf("window size must be positive")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.windowSize = windowSize
	return nil
}

func (d *LogAnomalyDetector) ExportAnomalies(ctx context.Context, format string) ([]byte, error) {
	anomalies, err := d.GetAnomalyHistory(ctx, 1000)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return d.exportJSON(anomalies)
	case "csv":
		return d.exportCSV(anomalies)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func (d *LogAnomalyDetector) exportJSON(anomalies []LogAnomaly) ([]byte, error) {
	_ = map[string]interface{}{
		"export_time":  time.Now(),
		"total_count":  len(anomalies),
		"anomalies":    anomalies,
		"threshold":    d.threshold,
		"window_size": d.windowSize.String(),
	}

	return []byte(fmt.Sprintf(`{"export_time":"%s","total_count":%d,"anomalies":[%s],"threshold":%.1f}`,
		time.Now().Format(time.RFC3339), len(anomalies), formatAnomaliesJSON(anomalies), d.threshold)), nil
}

func formatAnomaliesJSON(anomalies []LogAnomaly) string {
	if len(anomalies) == 0 {
		return ""
	}
	var parts []string
	for _, a := range anomalies {
		parts = append(parts, fmt.Sprintf(`{"id":"%s","type":"%s","severity":"%s","description":"%s","timestamp":"%s","score":%.2f}`,
			a.ID, a.Type, a.Severity, a.Description, a.Timestamp.Format(time.RFC3339), a.Score))
	}
	return strings.Join(parts, ",")
}

func (d *LogAnomalyDetector) exportCSV(anomalies []LogAnomaly) ([]byte, error) {
	var csv strings.Builder
	csv.WriteString("ID,Type,Severity,Description,Timestamp,Score\n")

	for _, a := range anomalies {
		csv.WriteString(fmt.Sprintf("%s,%s,%s,\"%s\",%s,%.2f\n",
			a.ID, a.Type, a.Severity, a.Description, a.Timestamp.Format(time.RFC3339), a.Score))
	}

	return []byte(csv.String()), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
