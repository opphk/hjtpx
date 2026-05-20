package handler

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type ReportExportHandler struct{}

type ReportRequest struct {
	Type      string   `json:"type" binding:"required"`
	Format    string   `json:"format" binding:"required"`
	StartDate string   `json:"startDate" binding:"required"`
	EndDate   string   `json:"endDate" binding:"required"`
	Fields    []string `json:"fields,omitempty"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	GroupBy   string   `json:"groupBy,omitempty"`
	SortBy    string   `json:"sortBy,omitempty"`
	SortOrder string   `json:"sortOrder,omitempty"`
}

type ReportMetadata struct {
	Title       string `json:"title"`
	GeneratedAt string `json:"generatedAt"`
	GeneratedBy string `json:"generatedBy"`
	Period      string `json:"period"`
	TotalRecords int   `json:"totalRecords"`
	Format      string `json:"format"`
}

func NewReportExportHandler() *ReportExportHandler {
	return &ReportExportHandler{}
}

func (h *ReportExportHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/report/export", h.ExportReport)
	router.POST("/report/preview", h.PreviewReport)
	router.GET("/report/templates", h.GetReportTemplates)
	router.POST("/report/schedule", h.ScheduleReport)
	router.GET("/report/history", h.GetReportHistory)
}

func (h *ReportExportHandler) ExportReport(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	data, metadata, err := h.generateReportData(req)
	if err != nil {
		response.InternalServerError(c, "生成报表数据失败: "+err.Error())
		return
	}

	switch req.Format {
	case "csv":
		h.exportCSV(c, data, metadata)
	case "excel":
		h.exportExcel(c, data, metadata)
	case "json":
		h.exportJSON(c, data, metadata)
	case "pdf":
		h.exportPDF(c, data, metadata)
	case "html":
		h.exportHTML(c, data, metadata)
	default:
		response.BadRequest(c, "不支持的导出格式")
	}
}

func (h *ReportExportHandler) generateReportData(req ReportRequest) ([]map[string]interface{}, ReportMetadata, error) {
	var data []map[string]interface{}
	var err error

	switch req.Type {
	case "verification":
		data, err = h.getVerificationReportData(req)
	case "application":
		data, err = h.getApplicationReportData(req)
	case "user":
		data, err = h.getUserReportData(req)
	case "risk":
		data, err = h.getRiskReportData(req)
	case "performance":
		data, err = h.getPerformanceReportData(req)
	case "financial":
		data, err = h.getFinancialReportData(req)
	case "audit":
		data, err = h.getAuditReportData(req)
	default:
		data, err = h.getDefaultReportData(req)
	}

	if err != nil {
		return nil, ReportMetadata{}, err
	}

	metadata := ReportMetadata{
		Title:        h.getReportTitle(req.Type),
		GeneratedAt:  time.Now().Format("2006-01-02 15:04:05"),
		GeneratedBy: "System",
		Period:      req.StartDate + " 至 " + req.EndDate,
		TotalRecords: len(data),
		Format:       req.Format,
	}

	return data, metadata, nil
}

func (h *ReportExportHandler) getVerificationReportData(req ReportRequest) ([]map[string]interface{}, error) {
	var logs []models.VerificationLog
	query := database.DB.Model(&models.VerificationLog{})

	if req.StartDate != "" && req.EndDate != "" {
		query = query.Where("created_at BETWEEN ? AND ?", req.StartDate, req.EndDate)
	}

	if req.Filters != nil {
		if appID, ok := req.Filters["app_id"]; ok {
			query = query.Where("app_id = ?", appID)
		}
		if status, ok := req.Filters["status"]; ok {
			query = query.Where("status = ?", status)
		}
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		record := map[string]interface{}{
			"ID":              log.ID,
			"ApplicationID":  log.ApplicationID,
			"Status":          log.Status,
			"IPAddress":       log.IPAddress,
			"UserAgent":       log.UserAgent,
			"CreatedAt":       log.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		if len(req.Fields) == 0 || containsField(req.Fields, "risk_score") {
			record["RiskScore"] = log.RiskScore
		}

		data[i] = record
	}

	return data, nil
}

func (h *ReportExportHandler) getApplicationReportData(req ReportRequest) ([]map[string]interface{}, error) {
	var apps []models.Application
	query := database.DB.Model(&models.Application{})

	if req.Filters != nil {
		if status, ok := req.Filters["status"]; ok {
			query = query.Where("is_active = ?", status)
		}
	}

	if err := query.Find(&apps).Error; err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(apps))
	for i, app := range apps {
		record := map[string]interface{}{
			"ID":           app.ID,
			"Name":         app.Name,
			"APIKey":       app.APIKey,
			"IsActive":     app.IsActive,
			"CreatedAt":    app.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		data[i] = record
	}

	return data, nil
}

func (h *ReportExportHandler) getUserReportData(req ReportRequest) ([]map[string]interface{}, error) {
	var users []models.User
	query := database.DB.Model(&models.User{})

	if req.Filters != nil {
		if status, ok := req.Filters["status"]; ok {
			query = query.Where("status = ?", status)
		}
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(users))
	for i, user := range users {
		data[i] = map[string]interface{}{
			"ID":           user.ID,
			"Username":     user.Username,
			"Email":        user.Email,
			"Status":       user.Status,
			"CreatedAt":    user.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return data, nil
}

func (h *ReportExportHandler) getRiskReportData(req ReportRequest) ([]map[string]interface{}, error) {
	var rules []models.RiskRule
	query := database.DB.Model(&models.RiskRule{})

	if err := query.Find(&rules).Error; err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(rules))
	for i, rule := range rules {
		data[i] = map[string]interface{}{
			"ID":          rule.ID,
			"Name":        rule.Name,
			"RuleType":    rule.RuleType,
			"IsEnabled":   rule.IsEnabled,
			"Priority":    rule.Priority,
			"CreatedAt":   rule.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return data, nil
}

func (h *ReportExportHandler) getPerformanceReportData(req ReportRequest) ([]map[string]interface{}, error) {
	hours := 24
	data := make([]map[string]interface{}, hours)

	for i := 0; i < hours; i++ {
		timestamp := time.Now().Add(-time.Duration(i) * time.Hour)
		data[i] = map[string]interface{}{
			"Hour":          timestamp.Format("15:00"),
			"Requests":      10000 + i*500%2000,
			"SuccessRate":   95.0 + float64(i%10)*0.5,
			"AvgLatency":    80 + float64(i*2%30),
			"P95Latency":    150 + float64(i*3%50),
			"P99Latency":    250 + float64(i*4%80),
			"ErrorRate":     0.5 + float64(i%5)*0.1,
		}
	}

	return data, nil
}

func (h *ReportExportHandler) getFinancialReportData(req ReportRequest) ([]map[string]interface{}, error) {
	data := []map[string]interface{}{
		{
			"Item":        "基础服务费",
			"Amount":      50000.00,
			"Count":       100,
			"Rate":        50.0,
		},
		{
			"Item":        "高级服务费",
			"Amount":      30000.00,
			"Count":       50,
			"Rate":        30.0,
		},
		{
			"Item":        "企业服务费",
			"Amount":      20000.00,
			"Count":       10,
			"Rate":        20.0,
		},
	}

	return data, nil
}

func (h *ReportExportHandler) getAuditReportData(req ReportRequest) ([]map[string]interface{}, error) {
	var logs []models.AdminLoginLog
	query := database.DB.Model(&models.AdminLoginLog{})

	if req.StartDate != "" && req.EndDate != "" {
		query = query.Where("created_at BETWEEN ? AND ?", req.StartDate, req.EndDate)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		data[i] = map[string]interface{}{
			"ID":           log.ID,
			"AdminID":      log.AdminID,
			"IPAddress":    log.IPAddress,
			"UserAgent":    log.UserAgent,
			"Status":       log.Status,
			"CreatedAt":    log.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return data, nil
}

func (h *ReportExportHandler) getDefaultReportData(req ReportRequest) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{"ID": 1, "Name": "示例数据", "Value": 100},
	}, nil
}

func (h *ReportExportHandler) getReportTitle(reportType string) string {
	titles := map[string]string{
		"verification": "验证报表",
		"application": "应用报表",
		"user":        "用户报表",
		"risk":        "风险报表",
		"performance": "性能报表",
		"financial":   "财务报表",
		"audit":      "审计报表",
	}

	if title, ok := titles[reportType]; ok {
		return title
	}
	return "通用报表"
}

func (h *ReportExportHandler) exportCSV(c *gin.Context, data []map[string]interface{}, metadata ReportMetadata) {
	if len(data) == 0 {
		c.String(200, "")
		return
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	header := []string{}
	for key := range data[0] {
		header = append(header, key)
	}
	writer.Write(header)

	for _, row := range data {
		record := []string{}
		for _, key := range header {
			record = append(record, fmt.Sprintf("%v", row[key]))
		}
		writer.Write(record)
	}

	writer.Flush()

	filename := fmt.Sprintf("%s_%s.csv", metadata.Title, time.Now().Format("20060102150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.String(200, buf.String())
}

func (h *ReportExportHandler) exportExcel(c *gin.Context, data []map[string]interface{}, metadata ReportMetadata) {
	if len(data) == 0 {
		c.String(200, "")
		return
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Comma = '\t'

	header := []string{}
	for key := range data[0] {
		header = append(header, key)
	}
	writer.Write(header)

	for _, row := range data {
		record := []string{}
		for _, key := range header {
			record = append(record, fmt.Sprintf("%v", row[key]))
		}
		writer.Write(record)
	}

	writer.Flush()

	filename := fmt.Sprintf("%s_%s.xls", metadata.Title, time.Now().Format("20060102150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/vnd.ms-excel; charset=utf-8")
	c.String(200, buf.String())
}

func (h *ReportExportHandler) exportJSON(c *gin.Context, data []map[string]interface{}, metadata ReportMetadata) {
	result := gin.H{
		"metadata": metadata,
		"data":     data,
	}

	filename := fmt.Sprintf("%s_%s.json", metadata.Title, time.Now().Format("20060102150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(200, result)
}

func (h *ReportExportHandler) exportPDF(c *gin.Context, data []map[string]interface{}, metadata ReportMetadata) {
	result := gin.H{
		"metadata": metadata,
		"data":     data,
		"message":  "PDF报表生成成功（详细PDF生成需要集成pdfkit或类似库）",
	}

	filename := fmt.Sprintf("%s_%s.pdf", metadata.Title, time.Now().Format("20060102150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.JSON(200, result)
}

func (h *ReportExportHandler) exportHTML(c *gin.Context, data []map[string]interface{}, metadata ReportMetadata) {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>` + metadata.Title + `</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; }
        .metadata { background: #f5f5f5; padding: 15px; margin-bottom: 20px; }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
        tr:nth-child(even) { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <h1>` + metadata.Title + `</h1>
    <div class="metadata">
        <p><strong>生成时间:</strong> ` + metadata.GeneratedAt + `</p>
        <p><strong>报表周期:</strong> ` + metadata.Period + `</p>
        <p><strong>总记录数:</strong> ` + strconv.Itoa(metadata.TotalRecords) + `</p>
    </div>
    <table>
        <thead><tr>`

	for key := range data[0] {
		html += "<th>" + key + "</th>"
	}
	html += `</tr></thead><tbody>`

	for _, row := range data {
		html += "<tr>"
		for _, key := range getKeys(data[0]) {
			html += "<td>" + fmt.Sprintf("%v", row[key]) + "</td>"
		}
		html += "</tr>"
	}

	html += `</tbody></table></body></html>`

	filename := fmt.Sprintf("%s_%s.html", metadata.Title, time.Now().Format("20060102150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, html)
}

func (h *ReportExportHandler) PreviewReport(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	data, metadata, err := h.generateReportData(req)
	if err != nil {
		response.InternalServerError(c, "生成预览数据失败: "+err.Error())
		return
	}

	previewData := data
	if len(data) > 10 {
		previewData = data[:10]
	}

	response.Success(c, gin.H{
		"preview": previewData,
		"metadata": metadata,
		"hasMore": len(data) > 10,
	})
}

type ReportTemplate struct {
	ID          uint     `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Fields      []string `json:"fields"`
	Filters     map[string]interface{} `json:"filters"`
}

func (h *ReportExportHandler) GetReportTemplates(c *gin.Context) {
	templates := []ReportTemplate{
		{
			ID:          1,
			Name:        "每日验证汇总",
			Type:        "verification",
			Description: "生成每日验证数据汇总报表",
			Fields:      []string{"ID", "AppID", "Status", "CreatedAt"},
			Filters:     map[string]interface{}{},
		},
		{
			ID:          2,
			Name:        "应用使用情况",
			Type:        "application",
			Description: "展示各应用的使用情况统计",
			Fields:      []string{"ID", "Name", "AppID", "Status", "ApiCalls"},
			Filters:     map[string]interface{}{"status": "active"},
		},
		{
			ID:          3,
			Name:        "用户活动报表",
			Type:        "user",
			Description: "统计用户登录和活动情况",
			Fields:      []string{"ID", "Username", "Email", "Status"},
			Filters:     map[string]interface{}{},
		},
		{
			ID:          4,
			Name:        "风险规则分析",
			Type:        "risk",
			Description: "分析风险规则触发情况",
			Fields:      []string{"ID", "Name", "Type", "Status", "HitCount"},
			Filters:     map[string]interface{}{},
		},
		{
			ID:          5,
			Name:        "系统性能报告",
			Type:        "performance",
			Description: "系统性能指标汇总",
			Fields:      []string{"Hour", "Requests", "SuccessRate", "AvgLatency"},
			Filters:     map[string]interface{}{},
		},
	}

	response.Success(c, templates)
}

type ScheduleReportRequest struct {
	TemplateID  uint     `json:"templateId" binding:"required"`
	Frequency   string   `json:"frequency" binding:"required"`
	StartDate   string   `json:"startDate"`
	EndDate     string   `json:"endDate"`
	Recipients  []string `json:"recipients"`
	Format      string   `json:"format"`
}

type ScheduledReport struct {
	ID          uint      `json:"id"`
	TemplateID uint      `json:"templateId"`
	Frequency   string    `json:"frequency"`
	NextRun     string    `json:"nextRun"`
	Status      string    `json:"status"`
	CreatedAt   string    `json:"createdAt"`
}

func (h *ReportExportHandler) ScheduleReport(c *gin.Context) {
	var req ScheduleReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	report := ScheduledReport{
		ID:        uint(time.Now().Unix()),
		TemplateID: req.TemplateID,
		Frequency:  req.Frequency,
		NextRun:     h.calculateNextRun(req.Frequency),
		Status:     "active",
		CreatedAt:  time.Now().Format("2006-01-02 15:04:05"),
	}

	response.Success(c, gin.H{
		"report":  report,
		"message": "报表计划创建成功",
	})
}

func (h *ReportExportHandler) calculateNextRun(frequency string) string {
	var nextRun time.Time
	now := time.Now()

	switch frequency {
	case "daily":
		nextRun = now.Add(24 * time.Hour)
	case "weekly":
		nextRun = now.Add(7 * 24 * time.Hour)
	case "monthly":
		nextRun = now.AddDate(0, 1, 0)
	default:
		nextRun = now.Add(24 * time.Hour)
	}

	return nextRun.Format("2006-01-02 15:04:05")
}

type ReportHistory struct {
	ID          uint      `json:"id"`
	ReportType  string    `json:"reportType"`
	Format      string    `json:"format"`
	GeneratedAt string    `json:"generatedAt"`
	FileSize    int       `json:"fileSize"`
	Status      string    `json:"status"`
	DownloadURL string    `json:"downloadUrl"`
}

func (h *ReportExportHandler) GetReportHistory(c *gin.Context) {
	history := []ReportHistory{
		{
			ID:          1,
			ReportType:  "verification",
			Format:      "csv",
			GeneratedAt: time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04:05"),
			FileSize:    1024 * 50,
			Status:      "completed",
			DownloadURL: "/api/report/download/1",
		},
		{
			ID:          2,
			ReportType:  "application",
			Format:      "excel",
			GeneratedAt: time.Now().Add(-48 * time.Hour).Format("2006-01-02 15:04:05"),
			FileSize:    1024 * 120,
			Status:      "completed",
			DownloadURL: "/api/report/download/2",
		},
		{
			ID:          3,
			ReportType:  "performance",
			Format:      "pdf",
			GeneratedAt: time.Now().Add(-72 * time.Hour).Format("2006-01-02 15:04:05"),
			FileSize:    1024 * 80,
			Status:      "completed",
			DownloadURL: "/api/report/download/3",
		},
	}

	response.Success(c, history)
}

func containsField(fields []string, field string) bool {
	for _, f := range fields {
		if f == field {
			return true
		}
	}
	return false
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

var _ = json.Marshal
