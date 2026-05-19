package handler

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
	"gorm.io/gorm"
)

type AnalyticsHandler struct {
	db           *gorm.DB
	cacheService interface {
		Get(key string) interface{}
		Set(key string, value interface{}, ttl time.Duration) error
	}
}

func NewAnalyticsHandler(db *gorm.DB, cacheService interface{}) *AnalyticsHandler {
	return &AnalyticsHandler{
		db:           db,
		cacheService: cacheService,
	}
}

func (h *AnalyticsHandler) GetAdvancedReport(c *gin.Context) {
	rangeType := c.DefaultQuery("range", "last7days")

	report := h.generateMockReport(rangeType)

	response.Success(c, report)
}

func (h *AnalyticsHandler) ExportReport(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")
	rangeType := c.DefaultQuery("range", "last7days")

	report := h.generateMockReport(rangeType)

	switch format {
	case "csv":
		csv := h.convertToCSV(report)
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=report_%s.csv", time.Now().Format("20060102")))
		c.String(200, csv)
	case "json":
		response.Success(c, report)
	default:
		response.Success(c, report)
	}
}

func (h *AnalyticsHandler) GetPredictions(c *gin.Context) {
	predictions := h.generateMockPredictions()
	response.Success(c, predictions)
}

func (h *AnalyticsHandler) GetTrendAnalysis(c *gin.Context) {
	trends := h.generateMockTrends()
	response.Success(c, trends)
}

func (h *AnalyticsHandler) GetGeoDistribution(c *gin.Context) {
	geoData := []map[string]interface{}{
		{"name": "北京", "value": 15000},
		{"name": "上海", "value": 12000},
		{"name": "广州", "value": 10000},
		{"name": "深圳", "value": 9000},
		{"name": "杭州", "value": 7000},
		{"name": "成都", "value": 6000},
		{"name": "武汉", "value": 5000},
		{"name": "西安", "value": 4000},
		{"name": "南京", "value": 3500},
		{"name": "重庆", "value": 3000},
	}
	response.Success(c, geoData)
}

func (h *AnalyticsHandler) GetDeviceDistribution(c *gin.Context) {
	deviceData := []map[string]interface{}{
		{"name": "桌面浏览器", "value": 55},
		{"name": "移动浏览器", "value": 35},
		{"name": "平板", "value": 7},
		{"name": "其他", "value": 3},
	}
	response.Success(c, deviceData)
}

func (h *AnalyticsHandler) GetTimeDistribution(c *gin.Context) {
	hours := make([]string, 24)
	values := make([]int, 24)
	for i := 0; i < 24; i++ {
		hours[i] = fmt.Sprintf("%d时", i)
		values[i] = int(math.Abs(math.Sin(float64(i)/3.0)*5000) + 1000)
	}
	response.Success(c, gin.H{
		"hours":  hours,
		"values": values,
	})
}

func (h *AnalyticsHandler) GetHeatmaps(c *gin.Context) {
	days := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	hours := make([]string, 24)
	for i := 0; i < 24; i++ {
		hours[i] = fmt.Sprintf("%d时", i)
	}

	values := make([][]int, 7)
	for i := 0; i < 7; i++ {
		values[i] = make([]int, 24)
		for j := 0; j < 24; j++ {
			values[i][j] = rand.Intn(500) + 50
		}
	}

	response.Success(c, gin.H{
		"days":   days,
		"hours":  hours,
		"values": values,
	})
}

func (h *AnalyticsHandler) GetCorrelations(c *gin.Context) {
	correlations := []map[string]interface{}{
		{"x": "响应时间", "y": "转化率", "correlation": -0.75},
		{"x": "验证复杂度", "y": "完成率", "correlation": -0.82},
		{"x": "用户活跃度", "y": "安全性", "correlation": 0.65},
		{"x": "API调用量", "y": "错误率", "correlation": 0.45},
		{"x": "移动端占比", "y": "用户体验", "correlation": 0.72},
	}
	response.Success(c, correlations)
}

func (h *AnalyticsHandler) GetForecasting(c *gin.Context) {
	labels := make([]string, 14)
	actual := make([]float64, 7)
	predicted := make([]float64, 14)
	upper := make([]float64, 14)
	lower := make([]float64, 14)

	for i := 0; i < 14; i++ {
		date := time.Now().AddDate(0, 0, i-13)
		labels[i] = date.Format("01-02")
	}

	for i := 0; i < 7; i++ {
		actual[i] = float64(rand.Intn(5000) + 3000)
	}

	for i := 0; i < 14; i++ {
		predicted[i] = float64(rand.Intn(3000) + 3000)
		upper[i] = predicted[i] * 1.2
		lower[i] = predicted[i] * 0.8
	}

	response.Success(c, gin.H{
		"labels":   labels,
		"actual":   actual,
		"predicted": predicted,
		"upper":    upper,
		"lower":    lower,
		"accuracy": 92.5,
	})
}

func (h *AnalyticsHandler) CreateCustomReport(c *gin.Context) {
	var req struct {
		Name      string   `json:"name"`
		Type      string   `json:"type"`
		StartDate string   `json:"start_date"`
		EndDate   string   `json:"end_date"`
		Charts    []string `json:"charts"`
		Format    string   `json:"format"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "invalid request", err.Error())
		return
	}

	report := map[string]interface{}{
		"id":         strconv.FormatInt(time.Now().UnixNano(), 10),
		"name":       req.Name,
		"type":       req.Type,
		"start_date": req.StartDate,
		"end_date":   req.EndDate,
		"charts":     req.Charts,
		"format":     req.Format,
		"created_at": time.Now(),
	}

	response.Success(c, report)
}

func (h *AnalyticsHandler) GetCustomReports(c *gin.Context) {
	reports := []map[string]interface{}{
		{
			"id":         "1",
			"name":       "周验证趋势",
			"type":       "verification",
			"created_at": time.Now().AddDate(0, 0, -7),
		},
		{
			"id":         "2",
			"name":       "月度安全报告",
			"type":       "risk",
			"created_at": time.Now().AddDate(0, -1, 0),
		},
		{
			"id":         "3",
			"name":       "性能分析",
			"type":       "performance",
			"created_at": time.Now().AddDate(0, 0, -3),
		},
	}

	response.Success(c, gin.H{
		"items":      reports,
		"total":      len(reports),
		"page":       1,
		"page_size":  20,
		"total_pages": 1,
	})
}

func (h *AnalyticsHandler) generateMockReport(rangeType string) map[string]interface{} {
	summary := map[string]interface{}{
		"total_verifications": rand.Intn(50000) + 50000,
		"pass_rate":           fmt.Sprintf("%.1f", rand.Float64()*20+75),
		"block_rate":          fmt.Sprintf("%.1f", rand.Float64()*10+5),
		"avg_response_time":   rand.Intn(50) + 30,
		"active_users":        rand.Intn(5000) + 1000,
		"revenue":             rand.Intn(50000) + 10000,
	}

	trend := h.generateTrendData(rangeType)
	riskDistribution := map[string]interface{}{
		"low":      rand.Intn(30000) + 20000,
		"medium":   rand.Intn(15000) + 5000,
		"high":     rand.Intn(5000) + 1000,
		"critical": rand.Intn(1000) + 100,
	}

	captchaTypes := []map[string]interface{}{
		{"type": "滑动验证", "count": 35000, "rate": 92.5},
		{"type": "点选验证", "count": 25000, "rate": 88.3},
		{"type": "图形验证", "count": 15000, "rate": 85.7},
		{"type": "语义验证", "count": 8000, "rate": 90.2},
		{"type": "3D验证", "count": 5000, "rate": 95.1},
	}

	return map[string]interface{}{
		"summary":          summary,
		"trend":            trend,
		"risk_distribution": riskDistribution,
		"captcha_types":    captchaTypes,
		"geo":              h.generateGeoData(),
		"device":           h.generateDeviceData(),
		"time_distribution": h.generateTimeDistribution(),
		"prediction":       h.generatePredictionData(),
		"heatmap":          h.generateHeatmapData(),
		"insights":         h.generateInsights(),
	}
}

func (h *AnalyticsHandler) generateTrendData(rangeType string) []map[string]interface{} {
	days := 7
	switch rangeType {
	case "last7days":
		days = 7
	case "last30days":
		days = 30
	case "thisMonth":
		days = time.Now().Day()
	case "lastMonth":
		days = 30
	}

	data := make([]map[string]interface{}, days)
	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -days+i+1)
		total := rand.Intn(10000) + 5000
		passed := int(float64(total) * (rand.Float64()*0.2 + 0.8))
		data[i] = map[string]interface{}{
			"time":        date.Format("01-02"),
			"total":       total,
			"passed":      passed,
			"failed":      total - passed,
			"pass_rate":   fmt.Sprintf("%.1f", float64(passed)/float64(total)*100),
			"response_time": rand.Intn(50) + 30,
			"risk_score":  fmt.Sprintf("%.1f", rand.Float64()*30+20),
		}
	}

	return data
}

func (h *AnalyticsHandler) generateGeoData() []map[string]interface{} {
	locations := []string{"北京", "上海", "广州", "深圳", "杭州", "成都", "武汉", "西安", "南京", "重庆"}
	data := make([]map[string]interface{}, len(locations))
	for i, loc := range locations {
		data[i] = map[string]interface{}{
			"name":  loc,
			"value": rand.Intn(10000) + 3000,
		}
	}
	return data
}

func (h *AnalyticsHandler) generateDeviceData() []map[string]interface{} {
	return []map[string]interface{}{
		{"name": "桌面浏览器", "value": 55},
		{"name": "移动浏览器", "value": 35},
		{"name": "平板", "value": 7},
		{"name": "其他", "value": 3},
	}
}

func (h *AnalyticsHandler) generateTimeDistribution() map[string]interface{} {
	hours := make([]string, 24)
	values := make([]int, 24)
	for i := 0; i < 24; i++ {
		hours[i] = fmt.Sprintf("%d时", i)
		values[i] = rand.Intn(5000) + 500
	}
	return map[string]interface{}{
		"hours":  hours,
		"values": values,
	}
}

func (h *AnalyticsHandler) generatePredictionData() map[string]interface{} {
	labels := make([]string, 14)
	actual := make([]float64, 7)
	predicted := make([]float64, 14)

	for i := 0; i < 14; i++ {
		date := time.Now().AddDate(0, 0, i-13)
		labels[i] = date.Format("01-02")
	}

	for i := 0; i < 7; i++ {
		actual[i] = float64(rand.Intn(5000) + 3000)
	}

	for i := 0; i < 14; i++ {
		predicted[i] = float64(rand.Intn(3000) + 3000)
	}

	return map[string]interface{}{
		"labels":    labels,
		"actual":    actual,
		"predicted": predicted,
		"accuracy":  92.5,
	}
}

func (h *AnalyticsHandler) generateHeatmapData() map[string]interface{} {
	days := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	hours := make([]string, 24)
	for i := 0; i < 24; i++ {
		hours[i] = fmt.Sprintf("%d时", i)
	}

	values := make([][]int, 7)
	for i := 0; i < 7; i++ {
		values[i] = make([]int, 24)
		for j := 0; j < 24; j++ {
			values[i][j] = rand.Intn(500) + 50
		}
	}

	return map[string]interface{}{
		"days":   days,
		"hours":  hours,
		"values": values,
	}
}

func (h *AnalyticsHandler) generateInsights() []map[string]interface{} {
	return []map[string]interface{}{
		{"type": "info", "message": "滑动验证成功率较上周提升了5.2%"},
		{"type": "warning", "message": "检测到异常流量来源，建议加强风控"},
		{"type": "success", "message": "系统性能稳定，平均响应时间低于目标值"},
		{"type": "info", "message": "移动端验证量增长显著，建议优化移动端体验"},
	}
}

func (h *AnalyticsHandler) generateMockPredictions() map[string]interface{} {
	labels := make([]string, 7)
	values := make([]float64, 7)
	confidence := make([]float64, 7)

	for i := 0; i < 7; i++ {
		date := time.Now().AddDate(0, 0, i+1)
		labels[i] = date.Format("01-02")
		values[i] = float64(rand.Intn(5000) + 3000)
		confidence[i] = rand.Float64()*20 + 80
	}

	return map[string]interface{}{
		"labels":    labels,
		"values":    values,
		"confidence": confidence,
	}
}

func (h *AnalyticsHandler) generateMockTrends() map[string]interface{} {
	days := 30
	labels := make([]string, days)
	trendValues := make([]float64, days)

	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -days+i+1)
		labels[i] = date.Format("01-02")
		trendValues[i] = float64(rand.Intn(5000) + 3000)
	}

	trend := 0.0
	if len(trendValues) > 1 {
		trend = (trendValues[len(trendValues)-1] - trendValues[0]) / trendValues[0] * 100
	}

	return map[string]interface{}{
		"labels": labels,
		"values": trendValues,
		"trend":  fmt.Sprintf("%.2f%%", trend),
	}
}

func (h *AnalyticsHandler) convertToCSV(report map[string]interface{}) string {
	csv := "时间,总验证数,通过数,失败数,通过率,平均响应时间,风险分\n"

	if trend, ok := report["summary"].(map[string]interface{}); ok {
		csv += fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s\n",
			"汇总",
			fmt.Sprintf("%v", trend["total_verifications"]),
			"-",
			"-",
			fmt.Sprintf("%s%%", trend["pass_rate"]),
			fmt.Sprintf("%sms", trend["avg_response_time"]),
			"-",
		)
	}

	return csv
}
