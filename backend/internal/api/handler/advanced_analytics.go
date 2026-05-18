package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

// ========== 用户行为分析 ==========

type UserBehaviorAnalysis struct {
	TotalVerifications      int64         `json:"totalVerifications"`
	SuccessRate             float64       `json:"successRate"`
	AvgVerificationTime     float64       `json:"avgVerificationTime"`
	CompletionRateTrend     []TrendPoint  `json:"completionRateTrend"`
	VerificationTimeStats   TimeStats     `json:"verificationTimeStats"`
	CaptchaTypePreference   []TypeCount   `json:"captchaTypePreference"`
	TimeDistributionHeatmap [][]int       `json:"timeDistributionHeatmap"`
	ActiveUserDistribution  []HourlyCount `json:"activeUserDistribution"`
}

type TrendPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type TimeStats struct {
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Average float64 `json:"average"`
	Median  float64 `json:"median"`
	P95     float64 `json:"p95"`
}

type TypeCount struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

type HourlyCount struct {
	Hour  int   `json:"hour"`
	Count int64 `json:"count"`
}

func GetUserBehaviorAnalysis(c *gin.Context) {
	mockData := UserBehaviorAnalysis{
		TotalVerifications:  854321,
		SuccessRate:         94.7,
		AvgVerificationTime: 3.2,
		CompletionRateTrend: generateCompletionRateTrend(),
		VerificationTimeStats: TimeStats{
			Min:     0.8,
			Max:     15.6,
			Average: 3.2,
			Median:  2.8,
			P95:     8.5,
		},
		CaptchaTypePreference: []TypeCount{
			{"滑块验证", 425632},
			{"点选验证", 215678},
			{"旋转验证", 112345},
			{"拼图验证", 75234},
			{"文字识别", 25432},
		},
		TimeDistributionHeatmap: generateHeatmap(),
		ActiveUserDistribution:  generateHourlyDistribution(),
	}
	response.Success(c, mockData)
}

func generateCompletionRateTrend() []TrendPoint {
	trend := make([]TrendPoint, 30)
	now := time.Now()
	for i := 29; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		value := 90 + float64(i)*0.15 + float64(i%5)*0.5
		trend[29-i] = TrendPoint{
			Date:  date.Format("2006-01-02"),
			Value: value,
		}
	}
	return trend
}

func generateHeatmap() [][]int {
	heatmap := make([][]int, 7)
	for day := 0; day < 7; day++ {
		heatmap[day] = make([]int, 24)
		for hour := 0; hour < 24; hour++ {
			base := 50 + (hour-12)*(hour-12)/20
			if hour >= 9 && hour <= 18 {
				base += 100
			}
			if day >= 5 {
				base -= 30
			}
			heatmap[day][hour] = base + int(time.Now().UnixNano()%30)
		}
	}
	return heatmap
}

func generateHourlyDistribution() []HourlyCount {
	distribution := make([]HourlyCount, 24)
	for hour := 0; hour < 24; hour++ {
		count := int64(100 + (hour-12)*(hour-12))
		if hour >= 9 && hour <= 18 {
			count += 500
		}
		distribution[hour] = HourlyCount{
			Hour:  hour,
			Count: count,
		}
	}
	return distribution
}

// ========== 攻击趋势分析 ==========

type AttackTrendAnalysis struct {
	DetectionRateTrend     []TrendPoint   `json:"detectionRateTrend"`
	AttackTypeDistribution []TypeCount    `json:"attackTypeDistribution"`
	GeoDistribution        []GeoCount     `json:"geoDistribution"`
	TimePatternAnalysis    TimePattern    `json:"timePatternAnalysis"`
	RiskScoreDistribution  []ScoreBin     `json:"riskScoreDistribution"`
	RecentAttacks          []AttackRecord `json:"recentAttacks"`
	Alerts                 []AlertItem    `json:"alerts"`
}

type GeoCount struct {
	Region string `json:"region"`
	Count  int64  `json:"count"`
}

type TimePattern struct {
	PeakHours    []int   `json:"peakHours"`
	WeekdayRatio float64 `json:"weekdayRatio"`
	WeekendRatio float64 `json:"weekendRatio"`
}

type ScoreBin struct {
	Range string `json:"range"`
	Count int64  `json:"count"`
}

type AttackRecord struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	Type      string    `json:"type"`
	Time      time.Time `json:"time"`
	RiskScore float64   `json:"riskScore"`
	Status    string    `json:"status"`
}

type AlertItem struct {
	ID       string    `json:"id"`
	Severity string    `json:"severity"`
	Message  string    `json:"message"`
	Time     time.Time `json:"time"`
	Resolved bool      `json:"resolved"`
}

func GetAttackTrendAnalysis(c *gin.Context) {
	mockData := AttackTrendAnalysis{
		DetectionRateTrend: generateDetectionRateTrend(),
		AttackTypeDistribution: []TypeCount{
			{"暴力破解", 4523},
			{"爬虫攻击", 3214},
			{"IP欺诈", 2876},
			{"设备指纹异常", 1987},
			{"会话劫持", 876},
			{"其他", 432},
		},
		GeoDistribution: []GeoCount{
			{"北京", 2845},
			{"上海", 2134},
			{"广东", 1876},
			{"浙江", 1234},
			{"海外", 987},
			{"其他", 654},
		},
		TimePatternAnalysis: TimePattern{
			PeakHours:    []int{2, 3, 4, 23},
			WeekdayRatio: 65.4,
			WeekendRatio: 34.6,
		},
		RiskScoreDistribution: []ScoreBin{
			{"0-20", 1234},
			{"21-40", 2345},
			{"41-60", 3456},
			{"61-80", 2876},
			{"81-100", 1234},
		},
		RecentAttacks: generateRecentAttacks(),
		Alerts:        generateAlerts(),
	}
	response.Success(c, mockData)
}

func generateDetectionRateTrend() []TrendPoint {
	trend := make([]TrendPoint, 30)
	now := time.Now()
	for i := 29; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		value := 95 - float64(i)*0.1 + float64(i%3)*2
		trend[29-i] = TrendPoint{
			Date:  date.Format("2006-01-02"),
			Value: value,
		}
	}
	return trend
}

func generateRecentAttacks() []AttackRecord {
	attacks := make([]AttackRecord, 10)
	now := time.Now()
	attackTypes := []string{"暴力破解", "爬虫攻击", "IP欺诈", "设备指纹异常", "会话劫持"}
	statuses := []string{"blocked", "flagged", "reviewed"}

	for i := 0; i < 10; i++ {
		attacks[i] = AttackRecord{
			ID:        "ATTACK-" + strconv.Itoa(1000+i),
			IP:        "192.168." + strconv.Itoa(i) + "." + strconv.Itoa(100+i),
			Type:      attackTypes[i%5],
			Time:      now.Add(-time.Duration(i*30) * time.Minute),
			RiskScore: 60 + float64(i*4),
			Status:    statuses[i%3],
		}
	}
	return attacks
}

func generateAlerts() []AlertItem {
	alerts := make([]AlertItem, 5)
	now := time.Now()
	severities := []string{"critical", "high", "medium", "low"}
	messages := []string{
		"检测到异常高频IP访问",
		"发现批量注册尝试",
		"设备指纹异常率上升",
		"验证失败率异常波动",
		"新风险规则触发",
	}

	for i := 0; i < 5; i++ {
		alerts[i] = AlertItem{
			ID:       "ALERT-" + strconv.Itoa(2000+i),
			Severity: severities[i%4],
			Message:  messages[i],
			Time:     now.Add(-time.Duration(i*60) * time.Minute),
			Resolved: i >= 3,
		}
	}
	return alerts
}

// ========== 风险报告生成 ==========

type ReportRequest struct {
	ReportType string `json:"report_type" binding:"required"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Format     string `json:"format"`
}

type ReportResponse struct {
	ReportID        string        `json:"reportId"`
	ReportType      string        `json:"reportType"`
	GeneratedAt     time.Time     `json:"generatedAt"`
	StartDate       string        `json:"startDate"`
	EndDate         string        `json:"endDate"`
	Summary         ReportSummary `json:"summary"`
	KeyMetrics      []MetricItem  `json:"keyMetrics"`
	Anomalies       []Anomaly     `json:"anomalies"`
	Recommendations []string      `json:"recommendations"`
	DownloadURL     string        `json:"downloadUrl"`
}

type ReportSummary struct {
	TotalRequests  int64   `json:"totalRequests"`
	SuccessRate    float64 `json:"successRate"`
	AttackDetected int64   `json:"attackDetected"`
	RiskReduction  float64 `json:"riskReduction"`
}

type MetricItem struct {
	Name       string  `json:"name"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
	Change     float64 `json:"change"`
	IsPositive bool    `json:"isPositive"`
}

type Anomaly struct {
	Time        string `json:"time"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

func GenerateRiskReport(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if req.StartDate == "" {
		req.StartDate = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	}
	if req.EndDate == "" {
		req.EndDate = time.Now().Format("2006-01-02")
	}
	if req.Format == "" {
		req.Format = "pdf"
	}

	report := ReportResponse{
		ReportID:    "REPORT-" + strconv.FormatInt(time.Now().Unix(), 10),
		ReportType:  req.ReportType,
		GeneratedAt: time.Now(),
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Summary: ReportSummary{
			TotalRequests:  854321,
			SuccessRate:    94.7,
			AttackDetected: 12345,
			RiskReduction:  45.2,
		},
		KeyMetrics: []MetricItem{
			{"验证成功率", 94.7, "%", 2.3, true},
			{"平均响应时间", 3.2, "s", -15.4, true},
			{"攻击拦截率", 98.5, "%", 1.2, true},
			{"活跃用户数", 12456, "", 8.7, true},
			{"风险评分", 72.3, "", -5.4, true},
		},
		Anomalies: []Anomaly{
			{
				Time:        "2024-01-15 02:30",
				Type:        "spike",
				Description: "验证请求量异常增加",
				Severity:    "high",
			},
			{
				Time:        "2024-01-14 18:45",
				Type:        "pattern",
				Description: "检测到批量注册模式",
				Severity:    "medium",
			},
		},
		Recommendations: []string{
			"建议在凌晨时段增加风控规则敏感度",
			"考虑对高频IP段实施更严格的验证策略",
			"建议更新设备指纹识别库",
			"优化滑块验证码难度参数",
		},
		DownloadURL: "/api/v1/admin/analytics/report/download/" + strconv.FormatInt(time.Now().Unix(), 10),
	}

	response.Success(c, report)
}

// ========== 自定义报表 ==========

type ReportConfig struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Metrics       []string               `json:"metrics"`
	TimeRange     TimeRangeConfig        `json:"timeRange"`
	Filters       map[string]interface{} `json:"filters"`
	Visualization string                 `json:"visualization"`
	Schedule      ScheduleConfig         `json:"schedule"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

type TimeRangeConfig struct {
	Type  string `json:"type"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type ScheduleConfig struct {
	Enabled   bool   `json:"enabled"`
	Frequency string `json:"frequency"`
	Email     string `json:"email"`
}

var reportConfigs []ReportConfig

func ListReportConfigs(c *gin.Context) {
	if len(reportConfigs) == 0 {
		reportConfigs = []ReportConfig{
			{
				ID:            "config-1",
				Name:          "日常监控报表",
				Description:   "每日系统运行状态监控",
				Metrics:       []string{"totalRequests", "successRate", "avgResponseTime", "attackCount"},
				TimeRange:     TimeRangeConfig{Type: "daily", Start: "", End: ""},
				Filters:       map[string]interface{}{},
				Visualization: "dashboard",
				Schedule:      ScheduleConfig{Enabled: true, Frequency: "daily", Email: "admin@example.com"},
				CreatedAt:     time.Now().AddDate(0, 0, -7),
				UpdatedAt:     time.Now().AddDate(0, 0, -1),
			},
			{
				ID:            "config-2",
				Name:          "安全分析报表",
				Description:   "安全攻击趋势分析",
				Metrics:       []string{"attackCount", "detectionRate", "riskScore", "blockedIPs"},
				TimeRange:     TimeRangeConfig{Type: "weekly", Start: "", End: ""},
				Filters:       map[string]interface{}{"severity": "high"},
				Visualization: "charts",
				Schedule:      ScheduleConfig{Enabled: false, Frequency: "", Email: ""},
				CreatedAt:     time.Now().AddDate(0, 0, -14),
				UpdatedAt:     time.Now().AddDate(0, 0, -3),
			},
		}
	}
	response.Success(c, gin.H{"list": reportConfigs, "total": len(reportConfigs)})
}

func CreateReportConfig(c *gin.Context) {
	var config ReportConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	config.ID = "config-" + strconv.Itoa(len(reportConfigs)+1)
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()
	reportConfigs = append(reportConfigs, config)

	response.Success(c, config)
}

func GetReportConfig(c *gin.Context) {
	id := c.Param("id")
	for _, config := range reportConfigs {
		if config.ID == id {
			response.Success(c, config)
			return
		}
	}
	response.NotFound(c, "报表配置不存在")
}

func UpdateReportConfig(c *gin.Context) {
	id := c.Param("id")
	var config ReportConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	for i, existing := range reportConfigs {
		if existing.ID == id {
			config.ID = id
			config.CreatedAt = existing.CreatedAt
			config.UpdatedAt = time.Now()
			reportConfigs[i] = config
			response.Success(c, config)
			return
		}
	}
	response.NotFound(c, "报表配置不存在")
}

func DeleteReportConfig(c *gin.Context) {
	id := c.Param("id")
	for i, config := range reportConfigs {
		if config.ID == id {
			reportConfigs = append(reportConfigs[:i], reportConfigs[i+1:]...)
			response.Success(c, gin.H{"message": "删除成功"})
			return
		}
	}
	response.NotFound(c, "报表配置不存在")
}

// ========== 数据可视化数据 ==========

type VisualizationData struct {
	PieChart     PieChartData     `json:"pieChart"`
	BarChart     BarChartData     `json:"barChart"`
	LineChart    LineChartData    `json:"lineChart"`
	RadarChart   RadarChartData   `json:"radarChart"`
	FunnelChart  FunnelChartData  `json:"funnelChart"`
	ScatterChart ScatterChartData `json:"scatterChart"`
}

type PieChartData struct {
	Labels []string `json:"labels"`
	Data   []int64  `json:"data"`
}

type BarChartData struct {
	Labels   []string  `json:"labels"`
	Datasets []Dataset `json:"datasets"`
}

type Dataset struct {
	Label string  `json:"label"`
	Data  []int64 `json:"data"`
}

type LineChartData struct {
	Labels   []string      `json:"labels"`
	Datasets []LineDataset `json:"datasets"`
}

type LineDataset struct {
	Label string    `json:"label"`
	Data  []float64 `json:"data"`
}

type RadarChartData struct {
	Labels []string  `json:"labels"`
	Data   []float64 `json:"data"`
}

type FunnelChartData struct {
	Stages []FunnelStage `json:"stages"`
}

type FunnelStage struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type ScatterChartData struct {
	Points []ScatterPoint `json:"points"`
}

type ScatterPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func GetVisualizationData(c *gin.Context) {
	data := VisualizationData{
		PieChart: PieChartData{
			Labels: []string{"滑块验证", "点选验证", "旋转验证", "拼图验证", "文字识别"},
			Data:   []int64{425632, 215678, 112345, 75234, 25432},
		},
		BarChart: BarChartData{
			Labels: []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"},
			Datasets: []Dataset{
				{"成功", []int64{12000, 15000, 18000, 16000, 20000, 25000, 22000}},
				{"失败", []int64{800, 1200, 900, 1100, 1500, 2000, 1800}},
			},
		},
		LineChart: LineChartData{
			Labels: generateLast30DaysLabels(),
			Datasets: []LineDataset{
				{"请求量", generateRandomFloats(30, 10000, 25000)},
				{"成功率", generateRandomFloats(30, 90, 98)},
			},
		},
		RadarChart: RadarChartData{
			Labels: []string{"安全性", "可用性", "性能", "准确性", "用户体验", "可靠性"},
			Data:   []float64{95, 92, 88, 94, 85, 93},
		},
		FunnelChart: FunnelChartData{
			Stages: []FunnelStage{
				{"访问页面", 100000},
				{"开始验证", 85000},
				{"完成验证", 78000},
				{"验证成功", 74000},
				{"继续操作", 68000},
			},
		},
		ScatterChart: ScatterChartData{
			Points: generateScatterPoints(),
		},
	}
	response.Success(c, data)
}

func generateLast30DaysLabels() []string {
	labels := make([]string, 30)
	now := time.Now()
	for i := 29; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		labels[29-i] = date.Format("01-02")
	}
	return labels
}

func generateRandomFloats(n int, min, max float64) []float64 {
	data := make([]float64, n)
	for i := range data {
		data[i] = min + float64(i)*(max-min)/float64(n) + float64(time.Now().UnixNano()%100)/100*(max-min)/5
	}
	return data
}

func generateScatterPoints() []ScatterPoint {
	points := make([]ScatterPoint, 50)
	for i := range points {
		points[i] = ScatterPoint{
			X: float64(i) * 2,
			Y: float64(i*3) + float64(time.Now().UnixNano()%20),
		}
	}
	return points
}
