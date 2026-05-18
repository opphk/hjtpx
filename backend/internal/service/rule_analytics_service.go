package service

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type RuleAnalyticsService struct {
	engine           *AdvancedRuleEngine
	history          []*RuleAnalyticsEntry
	maxHistorySize   int
	alertThresholds  *AlertThresholds
	notifications    chan *RuleAlert
	mu               sync.RWMutex
}

type RuleAnalyticsEntry struct {
	Timestamp       time.Time
	SessionID       string
	IPAddress       string
	TotalScore      float64
	IsBot           bool
	Confidence      float64
	TriggeredRules  []string
	CategoryScores  map[string]float64
	ProcessingTime   time.Duration
}

type AlertThresholds struct {
	BotRateThreshold       float64
	HighRiskScoreThreshold float64
	AnomalyCountThreshold  int
	FalsePositiveThreshold float64
}

type RuleAlert struct {
	AlertType   string
	Severity    string
	Message     string
	RuleName    string
	Timestamp  time.Time
	Details    map[string]interface{}
}

type RulePerformanceMetrics struct {
	RuleName           string
	TotalTriggers      int64
	TotalEvaluations   int64
	HitRate            float64
	FalsePositiveRate  float64
	TruePositiveRate   float64
	AverageScore       float64
	LastTriggered      time.Time
	Accuracy           float64
	Precision          float64
	Recall             float64
	F1Score            float64
}

type AnalyticsSummary struct {
	TotalEvaluations     int64
	TotalBots            int64
	BotRate              float64
	AverageConfidence    float64
	AverageProcessingTime time.Duration
	TopRules             []RulePerformanceMetrics
	CategoryBreakdown    map[string]CategoryAnalytics
	TrendData            []TrendEntry
	AnomalyAlerts        []RuleAlert
}

type CategoryAnalytics struct {
	TotalTriggers    int64
	AverageScore     float64
	TopRules         []string
	HitRate          float64
}

type TrendEntry struct {
	Timestamp   time.Time
	BotCount    int64
	HumanCount  int64
	TotalCount  int64
	BotRate     float64
	AvgScore    float64
}

func NewRuleAnalyticsService(engine *AdvancedRuleEngine) *RuleAnalyticsService {
	service := &RuleAnalyticsService{
		engine:         engine,
		history:        make([]*RuleAnalyticsEntry, 0),
		maxHistorySize: 10000,
		alertThresholds: &AlertThresholds{
			BotRateThreshold:       0.3,
			HighRiskScoreThreshold: 0.7,
			AnomalyCountThreshold:  100,
			FalsePositiveThreshold: 0.1,
		},
		notifications: make(chan *RuleAlert, 100),
	}

	go service.monitorAlerts()

	return service
}

func (ras *RuleAnalyticsService) RecordEvaluation(entry *RuleAnalyticsEntry) {
	ras.mu.Lock()
	defer ras.mu.Unlock()

	ras.history = append(ras.history, entry)

	if len(ras.history) > ras.maxHistorySize {
		ras.history = ras.history[1:]
	}

	ras.checkAlerts(entry)
}

func (ras *RuleAnalyticsService) checkAlerts(entry *RuleAnalyticsEntry) {
	if entry.IsBot && entry.TotalScore > ras.alertThresholds.HighRiskScoreThreshold {
		alert := &RuleAlert{
			AlertType:  "high_risk_bot",
			Severity:   "high",
			Message:    fmt.Sprintf("High risk bot detected: score=%.2f, confidence=%.2f", entry.TotalScore, entry.Confidence),
			RuleName:   "multiple",
			Timestamp:  time.Now(),
			Details: map[string]interface{}{
				"score":      entry.TotalScore,
				"confidence": entry.Confidence,
				"ip":         entry.IPAddress,
				"session":    entry.SessionID,
			},
		}
		select {
		case ras.notifications <- alert:
		default:
		}
	}
}

func (ras *RuleAnalyticsService) monitorAlerts() {
	for alert := range ras.notifications {
		ras.processAlert(alert)
	}
}

func (ras *RuleAnalyticsService) processAlert(alert *RuleAlert) {
	switch alert.Severity {
	case "critical":
		ras.handleCriticalAlert(alert)
	case "high":
		ras.handleHighAlert(alert)
	case "medium":
		ras.handleMediumAlert(alert)
	}
}

func (ras *RuleAnalyticsService) handleCriticalAlert(alert *RuleAlert) {
	fmt.Printf("[CRITICAL] %s: %s\n", alert.Timestamp.Format(time.RFC3339), alert.Message)
}

func (ras *RuleAnalyticsService) handleHighAlert(alert *RuleAlert) {
	fmt.Printf("[HIGH] %s: %s\n", alert.Timestamp.Format(time.RFC3339), alert.Message)
}

func (ras *RuleAnalyticsService) handleMediumAlert(alert *RuleAlert) {
	fmt.Printf("[MEDIUM] %s: %s\n", alert.Timestamp.Format(time.RFC3339), alert.Message)
}

func (ras *RuleAnalyticsService) GetAnalyticsSummary(timeRange time.Duration) *AnalyticsSummary {
	ras.mu.RLock()
	defer ras.mu.RUnlock()

	cutoff := time.Now().Add(-timeRange)
	recentEntries := make([]*RuleAnalyticsEntry, 0)

	for _, entry := range ras.history {
		if entry.Timestamp.After(cutoff) {
			recentEntries = append(recentEntries, entry)
		}
	}

	summary := &AnalyticsSummary{
		CategoryBreakdown: make(map[string]CategoryAnalytics),
		TrendData:         ras.generateTrendData(recentEntries),
		AnomalyAlerts:     make([]RuleAlert, 0),
	}

	if len(recentEntries) == 0 {
		return summary
	}

	summary.TotalEvaluations = int64(len(recentEntries))

	var totalScore float64
	var totalConfidence float64
	var totalProcessingTime int64

	for _, entry := range recentEntries {
		if entry.IsBot {
			summary.TotalBots++
		}
		totalScore += entry.TotalScore
		totalConfidence += entry.Confidence
		totalProcessingTime += entry.ProcessingTime.Nanoseconds()

		for category, score := range entry.CategoryScores {
			if _, exists := summary.CategoryBreakdown[category]; !exists {
				summary.CategoryBreakdown[category] = CategoryAnalytics{
					TopRules: make([]string, 0),
				}
			}
			cat := summary.CategoryBreakdown[category]
			cat.AverageScore += score
			cat.TotalTriggers++
			summary.CategoryBreakdown[category] = cat
		}
	}

	for category := range summary.CategoryBreakdown {
		cat := summary.CategoryBreakdown[category]
		if cat.TotalTriggers > 0 {
			cat.AverageScore /= float64(cat.TotalTriggers)
			cat.HitRate = float64(cat.TotalTriggers) / float64(summary.TotalEvaluations)
		}
		summary.CategoryBreakdown[category] = cat
	}

	if summary.TotalEvaluations > 0 {
		summary.BotRate = float64(summary.TotalBots) / float64(summary.TotalEvaluations)
		summary.AverageConfidence = totalConfidence / float64(summary.TotalEvaluations)
		summary.AverageProcessingTime = time.Duration(totalProcessingTime / summary.TotalEvaluations)
	}

	summary.TopRules = ras.calculateTopRules(recentEntries)

	return summary
}

func (ras *RuleAnalyticsService) generateTrendData(entries []*RuleAnalyticsEntry) []TrendEntry {
	if len(entries) == 0 {
		return []TrendEntry{}
	}

	buckets := 24
	if len(entries) < 24 {
		buckets = len(entries)
	}

	bucketSize := len(entries) / buckets
	if bucketSize == 0 {
		bucketSize = 1
	}

	trends := make([]TrendEntry, 0, buckets)

	for i := 0; i < len(entries); i += bucketSize {
		end := i + bucketSize
		if end > len(entries) {
			end = len(entries)
		}

		bucket := entries[i:end]
		trend := TrendEntry{
			Timestamp: bucket[0].Timestamp,
		}

		var totalScore float64
		for _, entry := range bucket {
			trend.TotalCount++
			if entry.IsBot {
				trend.BotCount++
			} else {
				trend.HumanCount++
			}
			totalScore += entry.TotalScore
		}

		if trend.TotalCount > 0 {
			trend.BotRate = float64(trend.BotCount) / float64(trend.TotalCount)
			trend.AvgScore = totalScore / float64(trend.TotalCount)
		}

		trends = append(trends, trend)
	}

	return trends
}

func (ras *RuleAnalyticsService) calculateTopRules(entries []*RuleAnalyticsEntry) []RulePerformanceMetrics {
	ruleStats := make(map[string]*RulePerformanceMetrics)

	for _, entry := range entries {
		for _, ruleName := range entry.TriggeredRules {
			if _, exists := ruleStats[ruleName]; !exists {
				ruleStats[ruleName] = &RulePerformanceMetrics{
					RuleName: ruleName,
				}
			}

			ruleStats[ruleName].TotalTriggers++
			ruleStats[ruleName].AverageScore += entry.TotalScore

			if entry.IsBot {
				ruleStats[ruleName].TruePositiveRate++
			} else {
				ruleStats[ruleName].FalsePositiveRate++
			}
		}
	}

	metrics := make([]RulePerformanceMetrics, 0, len(ruleStats))
	for _, metric := range ruleStats {
		if metric.TotalTriggers > 0 {
			metric.HitRate = float64(metric.TotalTriggers) / float64(len(entries))
			metric.AverageScore /= float64(metric.TotalTriggers)

			totalClassified := metric.TruePositiveRate + metric.FalsePositiveRate
			if totalClassified > 0 {
				metric.Precision = metric.TruePositiveRate / totalClassified
			}

			if metric.TotalTriggers > 0 {
				metric.Recall = metric.TruePositiveRate / float64(metric.TotalTriggers)
			}

			if metric.Precision+metric.Recall > 0 {
				metric.F1Score = 2 * (metric.Precision * metric.Recall) / (metric.Precision + metric.Recall)
			}

			metric.Accuracy = ras.calculateRuleAccuracy(metric)
		}
		metrics = append(metrics, *metric)
	}

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].TotalTriggers > metrics[j].TotalTriggers
	})

	if len(metrics) > 20 {
		metrics = metrics[:20]
	}

	return metrics
}

func (ras *RuleAnalyticsService) calculateRuleAccuracy(metric *RulePerformanceMetrics) float64 {
	if metric.TotalTriggers == 0 {
		return 0
	}

	truePositives := metric.TruePositiveRate
	falsePositives := metric.FalsePositiveRate

	trueNegatives := float64(ras.getTotalEvaluations()) - float64(metric.TotalTriggers) - falsePositives

	accuracy := (truePositives + trueNegatives) / float64(ras.getTotalEvaluations())
	return accuracy
}

func (ras *RuleAnalyticsService) getTotalEvaluations() int64 {
	ras.mu.RLock()
	defer ras.mu.RUnlock()
	return int64(len(ras.history))
}

func (ras *RuleAnalyticsService) GetRulePerformance(ruleName string) *RulePerformanceMetrics {
	ras.mu.RLock()
	defer ras.mu.RUnlock()

	metric := &RulePerformanceMetrics{
		RuleName: ruleName,
	}

	for _, entry := range ras.history {
		for _, triggeredRule := range entry.TriggeredRules {
			if triggeredRule == ruleName {
				metric.TotalTriggers++
				metric.AverageScore += entry.TotalScore

				if entry.IsBot {
					metric.TruePositiveRate++
				} else {
					metric.FalsePositiveRate++
				}

				if metric.LastTriggered.IsZero() || entry.Timestamp.After(metric.LastTriggered) {
					metric.LastTriggered = entry.Timestamp
				}
			}
		}
	}

	if metric.TotalTriggers > 0 {
		metric.HitRate = float64(metric.TotalTriggers) / float64(len(ras.history))
		metric.AverageScore /= float64(metric.TotalTriggers)

		totalClassified := metric.TruePositiveRate + metric.FalsePositiveRate
		if totalClassified > 0 {
			metric.Precision = metric.TruePositiveRate / totalClassified
		}

		if metric.TotalTriggers > 0 {
			metric.Recall = metric.TruePositiveRate / float64(metric.TotalTriggers)
		}

		if metric.Precision+metric.Recall > 0 {
			metric.F1Score = 2 * (metric.Precision * metric.Recall) / (metric.Precision + metric.Recall)
		}
	}

	return metric
}

func (ras *RuleAnalyticsService) GetCategoryAnalytics() map[string]CategoryAnalytics {
	ras.mu.RLock()
	defer ras.mu.RUnlock()

	analytics := make(map[string]CategoryAnalytics)

	for _, entry := range ras.history {
		for _, ruleName := range entry.TriggeredRules {
			rule, exists := ras.engine.GetRule(ruleName)
			if !exists {
				continue
			}

			category := rule.Category
			if _, exists := analytics[category]; !exists {
				analytics[category] = CategoryAnalytics{
					TopRules: make([]string, 0),
				}
			}

			cat := analytics[category]
			cat.TotalTriggers++
			cat.AverageScore += entry.TotalScore

			found := false
			for _, r := range cat.TopRules {
				if r == ruleName {
					found = true
					break
				}
			}
			if !found {
				cat.TopRules = append(cat.TopRules, ruleName)
			}
			analytics[category] = cat
		}
	}

	for category := range analytics {
		cat := analytics[category]
		if cat.TotalTriggers > 0 {
			cat.AverageScore /= float64(cat.TotalTriggers)
			cat.HitRate = float64(cat.TotalTriggers) / float64(len(ras.history))
		}

		if len(cat.TopRules) > 5 {
			cat.TopRules = cat.TopRules[:5]
		}
		analytics[category] = cat
	}

	return analytics
}

func (ras *RuleAnalyticsService) GetAlerts(timeRange time.Duration) []RuleAlert {
	ras.mu.RLock()
	defer ras.mu.RUnlock()

	cutoff := time.Now().Add(-timeRange)
	alerts := make([]RuleAlert, 0)

	for _, entry := range ras.history {
		if entry.Timestamp.After(cutoff) {
			if entry.IsBot && entry.TotalScore > ras.alertThresholds.HighRiskScoreThreshold {
				alert := RuleAlert{
					AlertType:  "high_risk_bot",
					Severity:   "high",
					Message:    fmt.Sprintf("Bot detected: score=%.2f, confidence=%.2f", entry.TotalScore, entry.Confidence),
					Timestamp:  entry.Timestamp,
					Details: map[string]interface{}{
						"score":      entry.TotalScore,
						"confidence": entry.Confidence,
						"ip":         entry.IPAddress,
						"session":    entry.SessionID,
					},
				}
				alerts = append(alerts, alert)
			}
		}
	}

	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].Timestamp.After(alerts[j].Timestamp)
	})

	return alerts
}

func (ras *RuleAnalyticsService) SetAlertThreshold(thresholdType string, value float64) error {
	ras.mu.Lock()
	defer ras.mu.Unlock()

	switch thresholdType {
	case "bot_rate":
		ras.alertThresholds.BotRateThreshold = value
	case "high_risk_score":
		ras.alertThresholds.HighRiskScoreThreshold = value
	case "anomaly_count":
		ras.alertThresholds.AnomalyCountThreshold = int(value)
	case "false_positive":
		ras.alertThresholds.FalsePositiveThreshold = value
	default:
		return fmt.Errorf("unknown threshold type: %s", thresholdType)
	}

	return nil
}

func (ras *RuleAnalyticsService) ExportAnalyticsReport(format string) string {
	ras.mu.RLock()
	defer ras.mu.RUnlock()

	summary := ras.GetAnalyticsSummary(24 * time.Hour)

	var sb strings.Builder

	switch format {
	case "json":
		sb.WriteString("{\n")
		sb.WriteString(fmt.Sprintf(`  "total_evaluations": %d,`+"\n", summary.TotalEvaluations))
		sb.WriteString(fmt.Sprintf(`  "total_bots": %d,`+"\n", summary.TotalBots))
		sb.WriteString(fmt.Sprintf(`  "bot_rate": %.4f,`+"\n", summary.BotRate))
		sb.WriteString(fmt.Sprintf(`  "average_confidence": %.4f,`+"\n", summary.AverageConfidence))
		sb.WriteString(fmt.Sprintf(`  "average_processing_time_ms": %.2f,`+"\n", float64(summary.AverageProcessingTime.Milliseconds())))
		sb.WriteString(`  "top_rules": [` + "\n")

		for i, rule := range summary.TopRules {
			if i > 0 {
				sb.WriteString(",\n")
			}
			sb.WriteString(fmt.Sprintf(`    {"name": "%s", "triggers": %d, "accuracy": %.4f, "f1_score": %.4f}`,
				rule.RuleName, rule.TotalTriggers, rule.Accuracy, rule.F1Score))
		}

		sb.WriteString("\n  ]\n")
		sb.WriteString("}")

	case "text", "markdown":
		sb.WriteString("# 机器人检测规则分析报告\n\n")
		sb.WriteString(fmt.Sprintf("**生成时间**: %s\n\n", time.Now().Format(time.RFC3339)))
		sb.WriteString("## 总体统计\n\n")
		sb.WriteString(fmt.Sprintf("- 总评估次数: %d\n", summary.TotalEvaluations))
		sb.WriteString(fmt.Sprintf("- 检测到机器人: %d\n", summary.TotalBots))
		sb.WriteString(fmt.Sprintf("- 机器人比例: %.2f%%\n", summary.BotRate*100))
		sb.WriteString(fmt.Sprintf("- 平均置信度: %.2f%%\n", summary.AverageConfidence*100))
		sb.WriteString(fmt.Sprintf("- 平均处理时间: %.2fms\n\n", float64(summary.AverageProcessingTime.Milliseconds())))

		sb.WriteString("## 规则性能排名\n\n")
		sb.WriteString("| 规则名称 | 触发次数 | 命中率 | 准确率 | F1分数 |\n")
		sb.WriteString("|---------|---------|------|------|------|\n")

		for _, rule := range summary.TopRules {
			sb.WriteString(fmt.Sprintf("| %s | %d | %.2f%% | %.2f%% | %.4f |\n",
				rule.RuleName, rule.TotalTriggers, rule.HitRate*100, rule.Accuracy*100, rule.F1Score))
		}

		sb.WriteString("\n## 分类统计\n\n")
		for category, cat := range summary.CategoryBreakdown {
			sb.WriteString(fmt.Sprintf("### %s\n", strings.ToUpper(category)))
			sb.WriteString(fmt.Sprintf("- 总触发: %d\n", cat.TotalTriggers))
			sb.WriteString(fmt.Sprintf("- 平均分数: %.4f\n", cat.AverageScore))
			sb.WriteString(fmt.Sprintf("- 命中率: %.2f%%\n\n", cat.HitRate*100))
		}

	default:
		sb.WriteString("Unsupported format. Use 'json', 'text', or 'markdown'.\n")
	}

	return sb.String()
}

type RuleAnalyticsHandler struct {
	analyticsService *RuleAnalyticsService
}

func NewRuleAnalyticsHandler(analyticsService *RuleAnalyticsService) *RuleAnalyticsHandler {
	return &RuleAnalyticsHandler{
		analyticsService: analyticsService,
	}
}

func (rah *RuleAnalyticsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/api/analytics/summary"):
		rah.handleSummary(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/analytics/rules"):
		rah.handleRules(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/analytics/categories"):
		rah.handleCategories(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/analytics/alerts"):
		rah.handleAlerts(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/analytics/report"):
		rah.handleReport(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (rah *RuleAnalyticsHandler) handleSummary(w http.ResponseWriter, r *http.Request) {
	duration := 24 * time.Hour

	summary := rah.analyticsService.GetAnalyticsSummary(duration)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"total_evaluations": %d, "total_bots": %d, "bot_rate": %.4f, "average_confidence": %.4f}`,
		summary.TotalEvaluations, summary.TotalBots, summary.BotRate, summary.AverageConfidence)
}

func (rah *RuleAnalyticsHandler) handleRules(w http.ResponseWriter, r *http.Request) {
	summary := rah.analyticsService.GetAnalyticsSummary(24 * time.Hour)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, "{\"rules\": [")

	for i, rule := range summary.TopRules {
		if i > 0 {
			fmt.Fprint(w, ",")
		}
		fmt.Fprintf(w, `{"name": "%s", "triggers": %d, "accuracy": %.4f}`,
			rule.RuleName, rule.TotalTriggers, rule.Accuracy)
	}

	fmt.Fprint(w, "]}")
}

func (rah *RuleAnalyticsHandler) handleCategories(w http.ResponseWriter, r *http.Request) {
	categories := rah.analyticsService.GetCategoryAnalytics()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, "{\"categories\": {")

	first := true
	for name, cat := range categories {
		if !first {
			fmt.Fprint(w, ",")
		}
		first = false
		fmt.Fprintf(w, `"%s": {"triggers": %d, "avg_score": %.4f, "hit_rate": %.4f}`,
			name, cat.TotalTriggers, cat.AverageScore, cat.HitRate)
	}

	fmt.Fprint(w, "}}")
}

func (rah *RuleAnalyticsHandler) handleAlerts(w http.ResponseWriter, r *http.Request) {
	duration := 24 * time.Hour
	alerts := rah.analyticsService.GetAlerts(duration)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"count": %d, "alerts": [`, len(alerts))

	for i, alert := range alerts {
		if i > 0 {
			fmt.Fprint(w, ",")
		}
		fmt.Fprintf(w, `{"type": "%s", "severity": "%s", "message": "%s", "timestamp": "%s"}`,
			alert.AlertType, alert.Severity, alert.Message, alert.Timestamp.Format(time.RFC3339))
	}

	fmt.Fprint(w, "]}")
}

func (rah *RuleAnalyticsHandler) handleReport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "markdown"
	}

	report := rah.analyticsService.ExportAnalyticsReport(format)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, report)
}
