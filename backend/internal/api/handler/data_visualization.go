package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type DataVisualizationHandler struct{}

type ChartDataRequest struct {
	Type     string `form:"type" binding:"required"`
	Period   string `form:"period" binding:"required"`
	Interval string `form:"interval,omitempty"`
	Metrics  []string `form:"metrics,omitempty"`
}

type ChartDataResponse struct {
	Labels   []string               `json:"labels"`
	Datasets []ChartDataset          `json:"datasets"`
	Metadata ChartMetadata           `json:"metadata"`
}

type ChartDataset struct {
	Label           string    `json:"label"`
	Data            []float64 `json:"data"`
	BackgroundColor []string  `json:"backgroundColor,omitempty"`
	BorderColor     []string  `json:"borderColor,omitempty"`
	Fill            bool      `json:"fill,omitempty"`
	Type            string    `json:"type,omitempty"`
}

type ChartMetadata struct {
	Total      float64 `json:"total"`
	Average    float64 `json:"average"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	ChangeRate float64 `json:"changeRate"`
	Period     string  `json:"period"`
	GeneratedAt string `json:"generatedAt"`
}

type MultiDimensionAnalysis struct {
	TimeSeries    ChartDataResponse `json:"timeSeries"`
	Geographic    []GeoData         `json:"geographic"`
	DeviceType    []DeviceData      `json:"deviceType"`
	BrowserType   []BrowserData     `json:"browserType"`
	BehaviorPattern []BehaviorData  `json:"behaviorPattern"`
}

type GeoData struct {
	Region  string  `json:"region"`
	Country string  `json:"country"`
	Count   int     `json:"count"`
	Rate    float64 `json:"rate"`
}

type DeviceData struct {
	Type  string  `json:"type"`
	Count int     `json:"count"`
	Rate  float64 `json:"rate"`
}

type BrowserData struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Rate  float64 `json:"rate"`
}

type BehaviorData struct {
	Pattern   string  `json:"pattern"`
	Count     int     `json:"count"`
	RiskLevel string  `json:"riskLevel"`
}

func NewDataVisualizationHandler() *DataVisualizationHandler {
	return &DataVisualizationHandler{}
}

func (h *DataVisualizationHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/visualization/chart/:type", h.GetChartData)
	router.GET("/visualization/analytics", h.GetMultiDimensionAnalysis)
	router.GET("/visualization/trends", h.GetTrends)
	router.GET("/visualization/distribution", h.GetDistribution)
	router.GET("/visualization/comparison", h.GetComparison)
	router.GET("/visualization/real-time", h.GetRealTimeMetrics)
}

func (h *DataVisualizationHandler) GetChartData(c *gin.Context) {
	var req ChartDataRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	var chartData ChartDataResponse

	switch req.Type {
	case "requests":
		chartData = h.generateRequestsChartData(req.Period)
	case "verification":
		chartData = h.generateVerificationChartData(req.Period)
	case "performance":
		chartData = h.generatePerformanceChartData(req.Period)
	case "users":
		chartData = h.generateUsersChartData(req.Period)
	case "risk":
		chartData = h.generateRiskChartData(req.Period)
	case "revenue":
		chartData = h.generateRevenueChartData(req.Period)
	default:
		chartData = h.generateDefaultChartData(req.Period)
	}

	response.Success(c, chartData)
}

func (h *DataVisualizationHandler) generateRequestsChartData(period string) ChartDataResponse {
	var labels []string
	var data []float64

	switch period {
	case "hour":
		labels = make([]string, 24)
		data = make([]float64, 24)
		for i := 0; i < 24; i++ {
			labels[i] = time.Now().Add(-time.Duration(23-i) * time.Hour).Format("15:04")
			data[i] = float64(1000 + i*137%5000)
		}
	case "day":
		labels = []string{"00:00", "04:00", "08:00", "12:00", "16:00", "20:00", "23:59"}
		data = []float64{1200, 800, 2500, 3800, 3200, 2800, 1500}
	case "week":
		labels = []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
		data = []float64{12000, 15000, 18000, 16000, 20000, 25000, 22000}
	case "month":
		labels = h.getLastNDays(30)
		data = make([]float64, 30)
		for i := 0; i < 30; i++ {
			data[i] = float64(85000 + i*1000%5000)
		}
	default:
		labels = h.getLastNDays(7)
		data = make([]float64, 7)
		for i := 0; i < 7; i++ {
			data[i] = float64(10000 + i*500%2000)
		}
	}

	datasets := []ChartDataset{
		{
			Label:           "请求量",
			Data:            data,
			BackgroundColor: []string{"rgba(54, 162, 235, 0.2)"},
			BorderColor:     []string{"rgba(54, 162, 235, 1)"},
			Fill:            true,
		},
	}

	return ChartDataResponse{
		Labels:   labels,
		Datasets: datasets,
		Metadata: h.calculateMetadata(data, period),
	}
}

func (h *DataVisualizationHandler) generateVerificationChartData(period string) ChartDataResponse {
	labels := []string{"成功", "失败", "待处理", "跳过"}
	data := []float64{8500, 1200, 300, 500}

	datasets := []ChartDataset{
		{
			Label:           "验证结果分布",
			Data:            data,
			BackgroundColor: []string{"rgba(75, 192, 192, 0.2)", "rgba(255, 99, 132, 0.2)", "rgba(255, 206, 86, 0.2)", "rgba(153, 102, 255, 0.2)"},
			BorderColor:     []string{"rgba(75, 192, 192, 1)", "rgba(255, 99, 132, 1)", "rgba(255, 206, 86, 1)", "rgba(153, 102, 255, 1)"},
			Fill:            false,
		},
	}

	return ChartDataResponse{
		Labels:   labels,
		Datasets: datasets,
		Metadata: h.calculateMetadata(data, period),
	}
}

func (h *DataVisualizationHandler) generatePerformanceChartData(period string) ChartDataResponse {
	labels := h.getLastNDays(24)
	data := make([]float64, 24)
	for i := 0; i < 24; i++ {
		data[i] = float64(50 + i*2%50)
	}

	datasets := []ChartDataset{
		{
			Label:           "平均响应时间 (ms)",
			Data:            data,
			BackgroundColor: []string{"rgba(255, 159, 64, 0.2)"},
			BorderColor:     []string{"rgba(255, 159, 64, 1)"},
			Fill:            true,
		},
	}

	return ChartDataResponse{
		Labels:   labels,
		Datasets: datasets,
		Metadata: h.calculateMetadata(data, period),
	}
}

func (h *DataVisualizationHandler) generateUsersChartData(period string) ChartDataResponse {
	labels := h.getLastNDays(7)
	newUsers := []float64{120, 150, 180, 200, 220, 250, 280}
	activeUsers := []float64{800, 850, 900, 950, 1000, 1100, 1200}

	datasets := []ChartDataset{
		{
			Label:           "新增用户",
			Data:            newUsers,
			BackgroundColor: []string{"rgba(54, 162, 235, 0.2)"},
			BorderColor:     []string{"rgba(54, 162, 235, 1)"},
			Fill:            true,
		},
		{
			Label:           "活跃用户",
			Data:            activeUsers,
			BackgroundColor: []string{"rgba(75, 192, 192, 0.2)"},
			BorderColor:     []string{"rgba(75, 192, 192, 1)"},
			Fill:            true,
		},
	}

	return ChartDataResponse{
		Labels:   labels,
		Datasets: datasets,
		Metadata: h.calculateMetadata(activeUsers, period),
	}
}

func (h *DataVisualizationHandler) generateRiskChartData(period string) ChartDataResponse {
	labels := []string{"低风险", "中风险", "高风险", "极高风险"}
	data := []float64{6000, 2000, 1500, 500}

	datasets := []ChartDataset{
		{
			Label:           "风险等级分布",
			Data:            data,
			BackgroundColor: []string{"rgba(75, 192, 192, 0.2)", "rgba(255, 206, 86, 0.2)", "rgba(255, 159, 64, 0.2)", "rgba(255, 99, 132, 0.2)"},
			BorderColor:     []string{"rgba(75, 192, 192, 1)", "rgba(255, 206, 86, 1)", "rgba(255, 159, 64, 1)", "rgba(255, 99, 132, 1)"},
			Fill:            false,
		},
	}

	return ChartDataResponse{
		Labels:   labels,
		Datasets: datasets,
		Metadata: h.calculateMetadata(data, period),
	}
}

func (h *DataVisualizationHandler) generateRevenueChartData(period string) ChartDataResponse {
	labels := h.getLastNDays(12)
	data := make([]float64, 12)
	for i := 0; i < 12; i++ {
		data[i] = float64(10000 + i*500%3000)
	}

	datasets := []ChartDataset{
		{
			Label:           "收入 (元)",
			Data:            data,
			BackgroundColor: []string{"rgba(153, 102, 255, 0.2)"},
			BorderColor:     []string{"rgba(153, 102, 255, 1)"},
			Fill:            true,
		},
	}

	return ChartDataResponse{
		Labels:   labels,
		Datasets: datasets,
		Metadata: h.calculateMetadata(data, period),
	}
}

func (h *DataVisualizationHandler) generateDefaultChartData(period string) ChartDataResponse {
	labels := h.getLastNDays(7)
	data := make([]float64, 7)
	for i := 0; i < 7; i++ {
		data[i] = float64(1000 + i*100%500)
	}

	datasets := []ChartDataset{
		{
			Label:           "数据",
			Data:            data,
			BackgroundColor: []string{"rgba(201, 203, 207, 0.2)"},
			BorderColor:     []string{"rgba(201, 203, 207, 1)"},
			Fill:            true,
		},
	}

	return ChartDataResponse{
		Labels:   labels,
		Datasets: datasets,
		Metadata: h.calculateMetadata(data, period),
	}
}

func (h *DataVisualizationHandler) calculateMetadata(data []float64, period string) ChartMetadata {
	if len(data) == 0 {
		return ChartMetadata{
			Period:      period,
			GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		}
	}

	var total, min, max float64
	total = data[0]
	min = data[0]
	max = data[0]

	for i := 1; i < len(data); i++ {
		total += data[i]
		if data[i] < min {
			min = data[i]
		}
		if data[i] > max {
			max = data[i]
		}
	}

	average := total / float64(len(data))

	var changeRate float64
	if len(data) > 1 {
		changeRate = ((data[len(data)-1] - data[0]) / data[0]) * 100
	}

	return ChartMetadata{
		Total:      total,
		Average:    average,
		Min:        min,
		Max:        max,
		ChangeRate: changeRate,
		Period:     period,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
}

func (h *DataVisualizationHandler) getLastNDays(n int) []string {
	labels := make([]string, n)
	for i := 0; i < n; i++ {
		labels[i] = time.Now().AddDate(0, 0, -n+i+1).Format("01-02")
	}
	return labels
}

func (h *DataVisualizationHandler) GetMultiDimensionAnalysis(c *gin.Context) {
	analysis := MultiDimensionAnalysis{
		TimeSeries:    h.generateRequestsChartData("week"),
		Geographic:    h.getGeographicData(),
		DeviceType:    h.getDeviceTypeData(),
		BrowserType:   h.getBrowserTypeData(),
		BehaviorPattern: h.getBehaviorPatternData(),
	}

	response.Success(c, analysis)
}

func (h *DataVisualizationHandler) getGeographicData() []GeoData {
	return []GeoData{
		{Country: "中国", Region: "亚洲", Count: 50000, Rate: 62.5},
		{Country: "美国", Region: "北美洲", Count: 15000, Rate: 18.75},
		{Country: "日本", Region: "亚洲", Count: 5000, Rate: 6.25},
		{Country: "德国", Region: "欧洲", Count: 3000, Rate: 3.75},
		{Country: "英国", Region: "欧洲", Count: 2000, Rate: 2.5},
		{Country: "其他", Region: "其他", Count: 5000, Rate: 6.25},
	}
}

func (h *DataVisualizationHandler) getDeviceTypeData() []DeviceData {
	return []DeviceData{
		{Type: "桌面端", Count: 45000, Rate: 56.25},
		{Type: "移动端", Count: 30000, Rate: 37.5},
		{Type: "平板", Count: 5000, Rate: 6.25},
	}
}

func (h *DataVisualizationHandler) getBrowserTypeData() []BrowserData {
	return []BrowserData{
		{Name: "Chrome", Count: 40000, Rate: 50.0},
		{Name: "Firefox", Count: 15000, Rate: 18.75},
		{Name: "Safari", Count: 12000, Rate: 15.0},
		{Name: "Edge", Count: 8000, Rate: 10.0},
		{Name: "其他", Count: 5000, Rate: 6.25},
	}
}

func (h *DataVisualizationHandler) getBehaviorPatternData() []BehaviorData {
	return []BehaviorData{
		{Pattern: "正常用户行为", Count: 65000, RiskLevel: "低"},
		{Pattern: "快速连续操作", Count: 8000, RiskLevel: "中"},
		{Pattern: "异常时间访问", Count: 3000, RiskLevel: "中"},
		{Pattern: "批量操作", Count: 2000, RiskLevel: "高"},
		{Pattern: "可疑行为", Count: 2000, RiskLevel: "极高"},
	}
}

type TrendsResponse struct {
	OverallTrend  string            `json:"overallTrend"`
	Metrics      map[string]TrendData `json:"metrics"`
	Predictions  []PredictionData  `json:"predictions"`
}

type TrendData struct {
	Current   float64 `json:"current"`
	Previous  float64 `json:"previous"`
	Change    float64 `json:"change"`
	ChangeRate float64 `json:"changeRate"`
	Trend     string  `json:"trend"`
}

type PredictionData struct {
	Date      string  `json:"date"`
	Predicted float64 `json:"predicted"`
	Lower     float64 `json:"lower"`
	Upper     float64 `json:"upper"`
}

func (h *DataVisualizationHandler) GetTrends(c *gin.Context) {
	metrics := map[string]TrendData{
		"requests": {
			Current:    15000,
			Previous:   12000,
			Change:     3000,
			ChangeRate: 25.0,
			Trend:      "up",
		},
		"successRate": {
			Current:    95.5,
			Previous:   94.2,
			Change:     1.3,
			ChangeRate: 1.38,
			Trend:      "up",
		},
		"avgResponseTime": {
			Current:    85.3,
			Previous:   92.1,
			Change:     -6.8,
			ChangeRate: -7.38,
			Trend:      "down",
		},
		"activeUsers": {
			Current:    8500,
			Previous:   7800,
			Change:     700,
			ChangeRate: 8.97,
			Trend:      "up",
		},
	}

	predictions := []PredictionData{}
	for i := 1; i <= 7; i++ {
		predictions = append(predictions, PredictionData{
			Date:      time.Now().AddDate(0, 0, i).Format("01-02"),
			Predicted: float64(15000 + i*500),
			Lower:     float64(15000 + i*500 - 1000),
			Upper:     float64(15000 + i*500 + 1000),
		})
	}

	response.Success(c, TrendsResponse{
		OverallTrend: "增长",
		Metrics:      metrics,
		Predictions:  predictions,
	})
}

type DistributionResponse struct {
	Type        string      `json:"type"`
	Data        []PieData   `json:"data"`
	Summary     DistributionSummary `json:"summary"`
}

type PieData struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Rate  float64 `json:"rate"`
}

type DistributionSummary struct {
	Total    int     `json:"total"`
	Distinct int     `json:"distinct"`
	TopItem  string  `json:"topItem"`
	Entropy  float64 `json:"entropy"`
}

func (h *DataVisualizationHandler) GetDistribution(c *gin.Context) {
	distType := c.DefaultQuery("type", "verification")

	var distribution DistributionResponse

	switch distType {
	case "verification":
		distribution = DistributionResponse{
			Type: "verification",
			Data: []PieData{
				{Label: "成功", Value: 8500, Rate: 85.0},
				{Label: "失败", Value: 1200, Rate: 12.0},
				{Label: "待处理", Value: 300, Rate: 3.0},
			},
			Summary: DistributionSummary{
				Total:    10000,
				Distinct: 3,
				TopItem:  "成功",
				Entropy:  0.56,
			},
		}
	case "device":
		distribution = DistributionResponse{
			Type: "device",
			Data: []PieData{
				{Label: "桌面端", Value: 56.25, Rate: 56.25},
				{Label: "移动端", Value: 37.5, Rate: 37.5},
				{Label: "平板", Value: 6.25, Rate: 6.25},
			},
			Summary: DistributionSummary{
				Total:    100,
				Distinct: 3,
				TopItem:  "桌面端",
				Entropy:  0.94,
			},
		}
	default:
		distribution = DistributionResponse{
			Type: "default",
			Data: []PieData{},
			Summary: DistributionSummary{
				Total:    0,
				Distinct: 0,
				TopItem:  "",
				Entropy:  0,
			},
		}
	}

	response.Success(c, distribution)
}

type ComparisonResponse struct {
	Items      []ComparisonItem `json:"items"`
	Metrics    map[string]ComparisonMetric `json:"metrics"`
}

type ComparisonItem struct {
	Name      string   `json:"name"`
	Values    []float64 `json:"values"`
	BestValue float64  `json:"bestValue"`
	Rank      int      `json:"rank"`
}

type ComparisonMetric struct {
	Name   string  `json:"name"`
	Unit   string  `json:"unit"`
	Weight float64 `json:"weight"`
}

func (h *DataVisualizationHandler) GetComparison(c *gin.Context) {
	items := []ComparisonItem{
		{
			Name:      "应用A",
			Values:    []float64{95.5, 85.3, 1200, 0.05},
			BestValue: 95.5,
			Rank:      1,
		},
		{
			Name:      "应用B",
			Values:    []float64{92.3, 78.5, 1500, 0.08},
			BestValue: 92.3,
			Rank:      2,
		},
		{
			Name:      "应用C",
			Values:    []float64{88.7, 92.1, 800, 0.03},
			BestValue: 88.7,
			Rank:      3,
		},
	}

	metrics := map[string]ComparisonMetric{
		"successRate":    {Name: "成功率", Unit: "%", Weight: 0.3},
		"responseTime":   {Name: "响应时间", Unit: "ms", Weight: 0.25},
		"requests":       {Name: "请求量", Unit: "", Weight: 0.25},
		"errorRate":      {Name: "错误率", Unit: "%", Weight: 0.2},
	}

	response.Success(c, ComparisonResponse{
		Items:   items,
		Metrics: metrics,
	})
}

type RealTimeMetricsResponse struct {
	Current   RealTimeData   `json:"current"`
	History   []RealTimeData `json:"history"`
	Status    SystemStatus   `json:"status"`
	Alerts    []Alert        `json:"alerts"`
}

type RealTimeData struct {
	Timestamp string  `json:"timestamp"`
	QPS       float64 `json:"qps"`
	Latency   float64 `json:"latency"`
	Errors    int     `json:"errors"`
	Users     int     `json:"users"`
}

type SystemStatus struct {
	Health     string `json:"health"`
	Uptime     int    `json:"uptime"`
	CPU        int    `json:"cpu"`
	Memory     int    `json:"memory"`
	Disk       int    `json:"disk"`
	NetworkIn  int    `json:"networkIn"`
	NetworkOut int    `json:"networkOut"`
}

type Alert struct {
	ID       uint   `json:"id"`
	Type     string `json:"type"`
	Message  string `json:"message"`
	Level    string `json:"level"`
	Time     string `json:"time"`
}

func (h *DataVisualizationHandler) GetRealTimeMetrics(c *gin.Context) {
	current := RealTimeData{
		Timestamp: time.Now().Format("15:04:05"),
		QPS:       1250.5,
		Latency:   85.3,
		Errors:    5,
		Users:     850,
	}

	history := []RealTimeData{}
	for i := 10; i >= 0; i-- {
		history = append(history, RealTimeData{
			Timestamp: time.Now().Add(-time.Duration(i) * time.Second).Format("15:04:05"),
			QPS:       1200 + float64(i*10),
			Latency:   80 + float64(i%20),
			Errors:    i % 10,
			Users:     800 + i*5,
		})
	}

	status := SystemStatus{
		Health:     "healthy",
		Uptime:     360000,
		CPU:        35,
		Memory:     55,
		Disk:       62,
		NetworkIn:  150000,
		NetworkOut: 250000,
	}

	alerts := []Alert{
		{
			ID:      1,
			Type:    "performance",
			Message: "响应时间略高于平均值",
			Level:   "info",
			Time:    time.Now().Add(-5 * time.Minute).Format("15:04"),
		},
	}

	response.Success(c, RealTimeMetricsResponse{
		Current:   current,
		History:   history,
		Status:    status,
		Alerts:    alerts,
	})
}
