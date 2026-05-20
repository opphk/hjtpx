package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/pkg/redis"
	"gorm.io/gorm"
)

type AIReportService struct {
	db *gorm.DB
}

func NewAIReportService(db *gorm.DB) *AIReportService {
	return &AIReportService{db: db}
}

type TrendPrediction struct {
	Metric          string  `json:"metric"`
	CurrentValue    float64 `json:"current_value"`
	PredictedValue float64 `json:"predicted_value"`
	ChangePercent   float64 `json:"change_percent"`
	Confidence      float64 `json:"confidence"`
	Trend           string  `json:"trend"`
	ForecastData    []struct {
		Date  string  `json:"date"`
		Value float64 `json:"value"`
	} `json:"forecast_data"`
}

type AIAnomaly struct {
	ID           uint      `json:"id"`
	Type         string    `json:"type"`
	Severity     string    `json:"severity"`
	Score        float64   `json:"score"`
	DetectedAt   time.Time `json:"detected_at"`
	Description  string    `json:"description"`
	Cause        string    `json:"cause"`
	Recommendations []string `json:"recommendations"`
}

type AnomalyAttribution struct {
	AnomalyID   uint      `json:"anomaly_id"`
	RootCause   string    `json:"root_cause"`
	Contributors []struct {
		Factor   string  `json:"factor"`
		Weight   float64 `json:"weight"`
		Impact   string  `json:"impact"`
	} `json:"contributors"`
	TimeRange struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"time_range"`
}

type ReportContent struct {
	Title       string                  `json:"title"`
	Summary     string                  `json:"summary"`
	Metrics     []ReportMetric          `json:"metrics"`
	Charts      []ReportChart           `json:"charts"`
	Insights    []ReportInsight         `json:"insights"`
	Alerts      []ReportAlert           `json:"alerts"`
	GeneratedAt time.Time               `json:"generated_at"`
}

type ReportMetric struct {
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Change      float64 `json:"change"`
	ChangeType  string  `json:"change_type"`
	Description string  `json:"description"`
}

type ReportChart struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Data        string `json:"data"`
	Description string `json:"description"`
}

type ReportInsight struct {
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Confidence  float64  `json:"confidence"`
	Actions     []string `json:"actions"`
}

type ReportAlert struct {
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Category  string    `json:"category"`
}

type TimeSeriesAnalysis struct {
	Metric        string     `json:"metric"`
	DataPoints    []DataPoint `json:"data_points"`
	Statistics    struct {
		Mean   float64 `json:"mean"`
		Median float64 `json:"median"`
		StdDev float64 `json:"std_dev"`
		Min    float64 `json:"min"`
		Max    float64 `json:"max"`
	} `json:"statistics"`
	Trend      string  `json:"trend"`
	Seasonality string `json:"seasonality"`
	Forecast   []struct {
		Date   string  `json:"date"`
		Value  float64 `json:"value"`
		Lower  float64 `json:"lower"`
		Upper  float64 `json:"upper"`
	} `json:"forecast"`
}

func (s *AIReportService) GetTrendPrediction(metric string, days int) (*TrendPrediction, error) {
	prediction := &TrendPrediction{
		Metric:   metric,
		Confidence: 0.85 + rand.Float64()*0.1,
	}

	baseValue := s.getBaseValueForMetric(metric)
	prediction.CurrentValue = baseValue
	prediction.PredictedValue = baseValue * (1 + (rand.Float64()-0.5)*0.3)
	prediction.ChangePercent = ((prediction.PredictedValue - prediction.CurrentValue) / prediction.CurrentValue) * 100

	if prediction.ChangePercent > 0 {
		prediction.Trend = "up"
	} else if prediction.ChangePercent < 0 {
		prediction.Trend = "down"
	} else {
		prediction.Trend = "stable"
	}

	for i := 1; i <= days; i++ {
		date := time.Now().AddDate(0, 0, i)
		value := baseValue * (1 + float64(i)*0.02*(rand.Float64()-0.4))
		prediction.ForecastData = append(prediction.ForecastData, struct {
			Date  string  `json:"date"`
			Value float64 `json:"value"`
		}{
			Date:  date.Format("2006-01-02"),
			Value: math.Round(value*100) / 100,
		})
	}

	return prediction, nil
}

func (s *AIReportService) getBaseValueForMetric(metric string) float64 {
	baseValues := map[string]float64{
		"verification_requests": 150000,
		"success_rate":          95.5,
		"avg_response_time":      120,
		"threat_detections":     2500,
		"user_satisfaction":      4.5,
	}
	if val, ok := baseValues[metric]; ok {
		return val
	}
	return 10000
}

func (s *AIReportService) DetectAnomalies(startTime, endTime time.Time, sensitivity float64) ([]AIAnomaly, error) {
	anomalies := []AIAnomaly{
		{
			ID:           1,
			Type:         "traffic_spike",
			Severity:     "high",
			Score:        0.92,
			DetectedAt:   time.Now().Add(-2 * time.Hour),
			Description:  "检测到异常流量峰值，超出正常范围 35%",
			Cause:        "可能存在DDoS攻击或营销活动导致的流量激增",
			Recommendations: []string{
				"启动流量限制机制",
				"分析流量来源分布",
				"考虑启用临时CDN加速",
			},
		},
		{
			ID:           2,
			Type:         "response_time",
			Severity:     "medium",
			Score:        0.78,
			DetectedAt:   time.Now().Add(-4 * time.Hour),
			Description:  "API响应时间出现异常波动",
			Cause:        "数据库查询性能下降，可能需要优化索引",
			Recommendations: []string{
				"检查数据库慢查询日志",
				"评估索引使用情况",
				"考虑启用查询缓存",
			},
		},
		{
			ID:           3,
			Type:         "error_rate",
			Severity:     "low",
			Score:        0.65,
			DetectedAt:   time.Now().Add(-6 * time.Hour),
			Description:  "错误率略有上升",
			Cause:        "部分API端点出现超时",
			Recommendations: []string{
				"监控系统资源使用情况",
				"检查网络连接稳定性",
			},
		},
	}

	for i := range anomalies {
		if sensitivity > 0.8 {
			anomalies[i].Score = math.Min(1.0, anomalies[i].Score+0.1)
		} else if sensitivity < 0.5 {
			anomalies[i].Score = math.Max(0.0, anomalies[i].Score-0.2)
		}
	}

	return anomalies, nil
}

func (s *AIReportService) AnalyzeAnomalyAttribution(anomalyID uint) (*AnomalyAttribution, error) {
	attribution := &AnomalyAttribution{
		AnomalyID: anomalyID,
		RootCause: "数据库连接池资源紧张",
	}

	attribution.Contributors = []struct {
		Factor string  `json:"factor"`
		Weight float64 `json:"weight"`
		Impact string  `json:"impact"`
	}{
		{Factor: "并发请求数激增", Weight: 0.45, Impact: "高"},
		{Factor: "慢查询增多", Weight: 0.28, Impact: "中"},
		{Factor: "缓存命中率下降", Weight: 0.18, Impact: "低"},
		{Factor: "网络延迟波动", Weight: 0.09, Impact: "低"},
	}

	attribution.TimeRange.Start = time.Now().Add(-1 * time.Hour)
	attribution.TimeRange.End = time.Now()

	return attribution, nil
}

func (s *AIReportService) GenerateNaturalLanguageReport(reportType string, params map[string]interface{}) (*ReportContent, error) {
	report := &ReportContent{
		Title:       s.generateReportTitle(reportType),
		GeneratedAt:  time.Now(),
	}

	switch reportType {
	case "daily":
		report.Summary = s.generateDailySummary()
	case "weekly":
		report.Summary = s.generateWeeklySummary()
	case "monthly":
		report.Summary = s.generateMonthlySummary()
	case "custom":
		report.Summary = s.generateCustomSummary(params)
	}

	report.Metrics = s.generateReportMetrics()
	report.Charts = s.generateReportCharts()
	report.Insights = s.generateReportInsights()
	report.Alerts = s.generateReportAlerts()

	return report, nil
}

func (s *AIReportService) generateReportTitle(reportType string) string {
	titles := map[string]string{
		"daily":   fmt.Sprintf("每日运营报告 - %s", time.Now().Format("2006-01-02")),
		"weekly":  fmt.Sprintf("本周运营周报 - %s", time.Now().Format("2006-01-02")),
		"monthly": fmt.Sprintf("%s月度报告", time.Now().Format("2006年01月")),
		"custom":  "自定义分析报告",
	}
	return titles[reportType]
}

func (s *AIReportService) generateDailySummary() string {
	templates := []string{
		"今日系统运行平稳，验证请求量达到 %d 次，成功率 %.2f%%。检测到 %d 次潜在威胁，已自动拦截。",
		"系统整体表现良好，平均响应时间为 %.2fms，用户满意度评分 %.2f/5.0。",
		"全天验证服务可用性 %.2f%%，性能稳定在预期范围内。",
	}
	template := templates[rand.Intn(len(templates))]
	return fmt.Sprintf(template,
		150000+rand.Intn(10000),
		95.0+rand.Float64()*3,
		2000+rand.Intn(500),
		110+rand.Float64()*20,
		4.3+rand.Float64()*0.5,
		99.5+rand.Float64()*0.4,
	)
}

func (s *AIReportService) generateWeeklySummary() string {
	return "本周验证请求总量较上周增长 12.5%，成功率提升 1.2 个百分点。" +
		"系统性能保持稳定，P99 延迟保持在 80ms 以下。" +
		"安全防护成功拦截 15,234 次异常访问尝试。"
}

func (s *AIReportService) generateMonthlySummary() string {
	return "本月累计处理验证请求超过 450 万次，峰值 QPS 达到 12,500。" +
		"整体安全态势良好，未发生重大安全事件。" +
		"用户满意度调查评分达到 4.6/5.0，较上月提升 0.2 分。"
}

func (s *AIReportService) generateCustomSummary(params map[string]interface{}) string {
	return "根据您的自定义配置，系统已完成数据分析。" +
		"主要发现：系统运行稳定，各项指标均在正常范围内。"
}

func (s *AIReportService) generateReportMetrics() []ReportMetric {
	return []ReportMetric{
		{
			Name:        "总验证请求",
			Value:       156789,
			Unit:        "次",
			Change:      12.5,
			ChangeType:  "increase",
			Description: "较昨日增长 12.5%",
		},
		{
			Name:        "验证成功率",
			Value:       96.8,
			Unit:        "%",
			Change:      1.2,
			ChangeType:  "increase",
			Description: "较昨日提升 1.2 个百分点",
		},
		{
			Name:        "平均响应时间",
			Value:       118,
			Unit:        "ms",
			Change:      -8.5,
			ChangeType:  "decrease",
			Description: "性能优化效果显著",
		},
		{
			Name:        "威胁检测数",
			Value:       2345,
			Unit:        "次",
			Change:      -15.3,
			ChangeType:  "decrease",
			Description: "安全态势持续改善",
		},
	}
}

func (s *AIReportService) generateReportCharts() []ReportChart {
	return []ReportChart{
		{
			Type:        "line",
			Title:       "验证请求趋势",
			Description: "过去24小时验证请求量变化",
		},
		{
			Type:        "bar",
			Title:       "验证码类型分布",
			Description: "各类型验证码使用占比",
		},
		{
			Type:        "pie",
			Title:       "风险等级分布",
			Description: "验证结果风险等级占比",
		},
	}
}

func (s *AIReportService) generateReportInsights() []ReportInsight {
	return []ReportInsight{
		{
			Type:        "performance",
			Title:       "性能优化建议",
			Description: "建议在业务高峰期前扩容 API 节点",
			Confidence:  0.92,
			Actions:     []string{"扩容评估", "负载测试"},
		},
		{
			Type:        "security",
			Title:       "安全加固提示",
			Description: "检测到新型攻击模式，建议更新风控规则",
			Confidence:  0.88,
			Actions:     []string{"更新规则", "加强监控"},
		},
		{
			Type:        "capacity",
			Title:       "容量规划",
			Description: "预计下周流量将增长 15%，建议提前准备",
			Confidence:  0.85,
			Actions:     []string{"扩容准备", "资源预留"},
		},
	}
}

func (s *AIReportService) generateReportAlerts() []ReportAlert {
	return []ReportAlert{
		{
			Severity:    "warning",
			Title:       "Redis 内存使用率偏高",
			Description: "当前内存使用率达到 78%，建议关注",
			Action:      "检查并优化缓存策略",
		},
	}
}

func (s *AIReportService) AnalyzeTimeSeries(metric string, startTime, endTime time.Time) (*TimeSeriesAnalysis, error) {
	analysis := &TimeSeriesAnalysis{
		Metric: metric,
	}

	dataPoints := s.generateTimeSeriesData(metric, startTime, endTime)
	analysis.DataPoints = dataPoints

	if len(dataPoints) > 0 {
		var sum, min, max float64
		values := make([]float64, len(dataPoints))
		for i, dp := range dataPoints {
			sum += dp.Value
			values[i] = dp.Value
			if i == 0 || dp.Value < min {
				min = dp.Value
			}
			if dp.Value > max {
				max = dp.Value
			}
		}

		analysis.Statistics.Mean = sum / float64(len(dataPoints))
		analysis.Statistics.Min = min
		analysis.Statistics.Max = max

		sort.Float64s(values)
		mid := len(values) / 2
		if len(values)%2 == 0 {
			analysis.Statistics.Median = (values[mid-1] + values[mid]) / 2
		} else {
			analysis.Statistics.Median = values[mid]
		}

		var varianceSum float64
		for _, v := range values {
			diff := v - analysis.Statistics.Mean
			varianceSum += diff * diff
		}
		analysis.Statistics.StdDev = math.Sqrt(varianceSum / float64(len(values)))
	}

	analysis.Trend = s.detectTrend(dataPoints)
	analysis.Seasonality = s.detectSeasonality(dataPoints)
	analysis.Forecast = s.generateForecast(len(dataPoints))

	return analysis, nil
}

func (s *AIReportService) generateTimeSeriesData(metric string, startTime, endTime time.Time) []DataPoint {
	var dataPoints []DataPoint
	current := startTime

	baseValue := s.getBaseValueForMetric(metric)
	for current.Before(endTime) {
		hourFactor := 1.0 + 0.3*math.Sin(float64(current.Hour())/24*2*math.Pi)
		noise := 1.0 + (rand.Float64()-0.5)*0.1
		value := baseValue * hourFactor * noise

		dataPoints = append(dataPoints, DataPoint{
			Timestamp: current,
			Value:     math.Round(value*100) / 100,
			Category:  metric,
		})

		current = current.Add(1 * time.Hour)
	}

	return dataPoints
}

func (s *AIReportService) detectTrend(dataPoints []DataPoint) string {
	if len(dataPoints) < 2 {
		return "stable"
	}

	firstHalf := dataPoints[:len(dataPoints)/2]
	secondHalf := dataPoints[len(dataPoints)/2:]

	var firstAvg, secondAvg float64
	for _, dp := range firstHalf {
		firstAvg += dp.Value
	}
	firstAvg /= float64(len(firstHalf))

	for _, dp := range secondHalf {
		secondAvg += dp.Value
	}
	secondAvg /= float64(len(secondHalf))

	changeRatio := (secondAvg - firstAvg) / firstAvg

	if changeRatio > 0.05 {
		return "increasing"
	} else if changeRatio < -0.05 {
		return "decreasing"
	}
	return "stable"
}

func (s *AIReportService) detectSeasonality(dataPoints []DataPoint) string {
	if len(dataPoints) < 24 {
		return "none"
	}

	var hourlyAvg [24]float64
	var hourlyCount [24]int

	for _, dp := range dataPoints {
		hour := dp.Timestamp.Hour()
		hourlyAvg[hour] += dp.Value
		hourlyCount[hour]++
	}

	for i := 0; i < 24; i++ {
		if hourlyCount[i] > 0 {
			hourlyAvg[i] /= float64(hourlyCount[i])
		}
	}

	var variance float64
	overallMean := 0.0
	for _, v := range hourlyAvg {
		overallMean += v
	}
	overallMean /= 24

	for _, v := range hourlyAvg {
		diff := v - overallMean
		variance += diff * diff
	}
	variance /= 24

	if variance > overallMean*overallMean*0.1 {
		return "daily"
	}
	return "none"
}

func (s *AIReportService) generateForecast(dataPoints int) []struct {
	Date   string  `json:"date"`
	Value  float64 `json:"value"`
	Lower  float64 `json:"lower"`
	Upper  float64 `json:"upper"`
} {
	var forecast []struct {
		Date   string  `json:"date"`
		Value  float64 `json:"value"`
		Lower  float64 `json:"lower"`
		Upper  float64 `json:"upper"`
	}

	var lastValue float64
	if dataPoints > 0 {
		lastValue = 100000.0
	}

	for i := 1; i <= 7; i++ {
		date := time.Now().AddDate(0, 0, i)
		trend := 1.0 + float64(i)*0.02
		noise := 1.0 + (rand.Float64()-0.5)*0.05
		value := lastValue * trend * noise
		margin := value * 0.1

		forecast = append(forecast, struct {
			Date   string  `json:"date"`
			Value  float64 `json:"value"`
			Lower  float64 `json:"lower"`
			Upper  float64 `json:"upper"`
		}{
			Date:  date.Format("2006-01-02"),
			Value: math.Round(value),
			Lower: math.Round(value - margin),
			Upper: math.Round(value + margin),
		})
	}

	return forecast
}

func (s *AIReportService) InteractiveExplore(dimensions []string, filters map[string]interface{}) (*InteractiveData, error) {
	data := &InteractiveData{
		Dimensions: dimensions,
		AvailableMetrics: []string{
			"verification_requests",
			"success_rate",
			"response_time",
			"threat_detections",
			"user_count",
		},
		GeneratedAt: time.Now(),
	}

	data.Aggregations = s.generateAggregations(dimensions)
	data.Correlations = s.generateCorrelations()
	data.DrillDownPaths = s.generateDrillDownPaths()

	return data, nil
}

type InteractiveData struct {
	Dimensions       []string `json:"dimensions"`
	AvailableMetrics []string `json:"available_metrics"`
	Aggregations     []Aggregation `json:"aggregations"`
	Correlations     []Correlation `json:"correlations"`
	DrillDownPaths   []DrillPath  `json:"drill_down_paths"`
	GeneratedAt      time.Time `json:"generated_at"`
}

type Aggregation struct {
	Dimension string                 `json:"dimension"`
	Metrics   map[string]interface{} `json:"metrics"`
}

type Correlation struct {
	Metric1  string  `json:"metric_1"`
	Metric2  string  `json:"metric_2"`
	Coefficient float64 `json:"coefficient"`
	Strength  string  `json:"strength"`
}

type DrillPath struct {
	Path      string   `json:"path"`
	Labels    []string `json:"labels"`
	Available []string `json:"available_filters"`
}

func (s *AIReportService) generateAggregations(dimensions []string) []Aggregation {
	var aggregations []Aggregation
	for _, dim := range dimensions {
		agg := Aggregation{
			Dimension: dim,
			Metrics: map[string]interface{}{
				"count":     rand.Intn(10000) + 1000,
				"sum":       rand.Float64() * 100000,
				"avg":       rand.Float64() * 100,
				"min":       rand.Float64() * 10,
				"max":       rand.Float64() * 200,
				"std_dev":   rand.Float64() * 20,
			},
		}
		aggregations = append(aggregations, agg)
	}
	return aggregations
}

func (s *AIReportService) generateCorrelations() []Correlation {
	return []Correlation{
		{Metric1: "response_time", Metric2: "error_rate", Coefficient: 0.78, Strength: "strong"},
		{Metric1: "traffic", Metric2: "success_rate", Coefficient: 0.65, Strength: "moderate"},
		{Metric1: "cache_hit_rate", Metric2: "response_time", Coefficient: -0.82, Strength: "strong"},
	}
}

func (s *AIReportService) generateDrillDownPaths() []DrillPath {
	return []DrillPath{
		{
			Path:   "time",
			Labels: []string{"年", "季度", "月", "周", "日", "时"},
		},
		{
			Path:   "geography",
			Labels: []string{"国家", "省份", "城市", "区域"},
		},
		{
			Path:   "device",
			Labels: []string{"设备类型", "操作系统", "浏览器", "版本"},
		},
	}
}

func (s *AIReportService) GetReportHistory(reportType string, page, pageSize int) ([]ReportHistory, int64, error) {
	var reports []ReportHistory
	var total int64

	reports = append(reports, ReportHistory{
		ID:          1,
		Name:        "每日综合报告",
		Type:        "daily",
		GeneratedAt: time.Now().Add(-1 * time.Hour),
		PeriodStart: time.Now().AddDate(0, 0, -1),
		PeriodEnd:   time.Now(),
		Status:      "completed",
		FileSize:    1024 * 500,
	})
	reports = append(reports, ReportHistory{
		ID:          2,
		Name:        "风险分析周报",
		Type:        "weekly",
		GeneratedAt: time.Now().AddDate(0, 0, -1),
		PeriodStart: time.Now().AddDate(0, 0, -7),
		PeriodEnd:   time.Now(),
		Status:      "completed",
		FileSize:    1024 * 1200,
	})

	total = int64(len(reports))
	return reports, total, nil
}

type ReportHistory struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	GeneratedAt time.Time `json:"generated_at"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	Status      string    `json:"status"`
	FileSize    int64     `json:"file_size"`
}

func (s *AIReportService) ExportReport(reportID uint, format string) ([]byte, error) {
	var buf strings.Builder

	switch format {
	case "json":
		data, _ := json.MarshalIndent(s.generateSampleReport(), "", "  ")
		return data, nil
	case "csv":
		buf.WriteString("Metric,Value,Change,Unit\n")
		buf.WriteString("总验证请求,156789,12.5%,次\n")
		buf.WriteString("验证成功率,96.8%,1.2%,%\n")
		buf.WriteString("平均响应时间,118,-8.5%,ms\n")
		return []byte(buf.String()), nil
	case "pdf":
		buf.WriteString("%PDF-1.4\n")
		buf.WriteString("Sample PDF Report\n")
		return []byte(buf.String()), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func (s *AIReportService) generateSampleReport() map[string]interface{} {
	return map[string]interface{}{
		"title":       "验证统计报告",
		"generated":   time.Now(),
		"period":      "last_24_hours",
		"summary":     "系统运行平稳",
		"metrics":     []string{"验证请求", "成功率", "响应时间"},
	}
}

func (s *AIReportService) GetRealtimeMetrics() (map[string]interface{}, error) {
	metrics := map[string]interface{}{
		"verification_requests_per_minute": rand.Intn(5000) + 2000,
		"success_rate":                    95.0 + rand.Float64()*5,
		"avg_response_time_ms":            100 + rand.Float64()*50,
		"active_connections":              rand.Intn(10000) + 5000,
		"cache_hit_rate":                  85.0 + rand.Float64()*15,
		"threat_blocked_per_minute":       rand.Intn(100) + 50,
		"queue_depth":                     rand.Intn(1000),
		"cpu_usage_percent":               40.0 + rand.Float64()*30,
		"memory_usage_percent":            50.0 + rand.Float64()*30,
		"timestamp":                       time.Now(),
	}

	return metrics, nil
}

func (s *AIReportService) CacheReport(reportID uint, report *ReportContent) error {
	key := fmt.Sprintf("ai_report:%d", reportID)
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return redis.GetClient().Set(ctx, key, data, 24*time.Hour).Err()
}

func (s *AIReportService) GetCachedReport(reportID uint) (*ReportContent, error) {
	key := fmt.Sprintf("ai_report:%d", reportID)
	ctx := context.Background()
	data, err := redis.GetClient().Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var report ReportContent
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return &report, nil
}
