package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/export"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

// ExportHandler 导出处理器
type ExportHandler struct {
	logService              *service.LogService
	scheduledExportService  *service.ScheduledExportService
	reportTemplateService   *service.ReportTemplateService
	exportHistoryService    *service.ExportHistoryService
}

// NewExportHandler 创建导出处理器
func NewExportHandler() *ExportHandler {
	return &ExportHandler{
		logService:             service.NewLogService(),
		scheduledExportService: service.NewScheduledExportService(),
		reportTemplateService:  service.NewReportTemplateService(),
		exportHistoryService:   service.NewExportHistoryService(),
	}
}

// EnhancedExportLogs 增强的导出日志功能
func (h *ExportHandler) EnhancedExportLogs(c *gin.Context) {
	var req struct {
		ApplicationID uint   `form:"application_id"`
		Status        string `form:"status"`
		CaptchaType   string `form:"captcha_type"`
		StartDate     string `form:"start_date"`
		EndDate       string `form:"end_date"`
		Format        string `form:"format" binding:"required"`
		Title         string `form:"title"`
	}
	
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid parameters")
		return
	}
	
	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)
	if req.EndDate != "" {
		endDate = endDate.Add(24 * time.Hour)
	}
	
	// 查询日志
	params := service.LogQueryParams{
		ApplicationID: req.ApplicationID,
		Status:        req.Status,
		CaptchaType:   req.CaptchaType,
		StartDate:     startDate,
		EndDate:       endDate,
		PageSize:      10000, // 导出时限制数量
	}
	
	result, err := h.logService.QueryLogs(params)
	if err != nil {
		response.InternalServerError(c, "Query logs failed")
		return
	}
	
	title := req.Title
	if title == "" {
		title = "Verification Logs Export"
	}
	
	// 根据格式导出
	switch req.Format {
	case "xlsx", "excel":
		h.exportAsExcel(c, result.Logs, title)
	case "pdf":
		h.exportAsPDF(c, result.Logs, title)
	case "json":
		h.exportAsJSON(c, result.Logs, title)
	case "csv":
		h.exportAsCSV(c, result.Logs, title)
	case "html", "visualization":
		h.exportAsVisualization(c, result.Logs, title)
	default:
		response.BadRequest(c, "Unsupported export format")
	}
}

func (h *ExportHandler) exportAsExcel(c *gin.Context, logs []models.VerificationLog, title string) {
	exportData := export.ConvertLogsToExportData(logs, title)
	exporter := export.NewExcelExporter()
	data, err := exporter.Export(exportData)
	if err != nil {
		response.InternalServerError(c, "Export failed")
		return
	}
	
	filename := fmt.Sprintf("verification_logs_%s.xlsx", time.Now().Format("20060102150405"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func (h *ExportHandler) exportAsPDF(c *gin.Context, logs []models.VerificationLog, title string) {
	exportData := export.ConvertLogsToExportData(logs, title)
	exporter := export.NewPDFExporter()
	data, err := exporter.Export(exportData)
	if err != nil {
		response.InternalServerError(c, "Export failed")
		return
	}
	
	filename := fmt.Sprintf("verification_logs_%s.pdf", time.Now().Format("20060102150405"))
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(200, "application/pdf", data)
}

func (h *ExportHandler) exportAsJSON(c *gin.Context, logs []models.VerificationLog, title string) {
	exportData := export.ConvertLogsToExportData(logs, title)
	exporter := export.NewJSONExporter()
	data, err := exporter.Export(exportData)
	if err != nil {
		response.InternalServerError(c, "Export failed")
		return
	}
	
	filename := fmt.Sprintf("verification_logs_%s.json", time.Now().Format("20060102150405"))
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(200, "application/json", data)
}

func (h *ExportHandler) exportAsCSV(c *gin.Context, logs []models.VerificationLog, title string) {
	exportData := export.ConvertLogsToExportData(logs, title)
	exporter := export.NewCSVExporter()
	data, err := exporter.Export(exportData)
	if err != nil {
		response.InternalServerError(c, "Export failed")
		return
	}
	
	filename := fmt.Sprintf("verification_logs_%s.csv", time.Now().Format("20060102150405"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(200, "text/csv", data)
}

func (h *ExportHandler) exportAsVisualization(c *gin.Context, logs []models.VerificationLog, title string) {
	vizData := export.GenerateLogVisualization(logs, title)
	exporter := export.NewVisualizationExporter()
	data, err := exporter.ExportHTML(vizData)
	if err != nil {
		response.InternalServerError(c, "Export failed")
		return
	}
	
	filename := fmt.Sprintf("verification_logs_visualization_%s.html", time.Now().Format("20060102150405"))
	c.Header("Content-Type", "text/html")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(200, "text/html", data)
}

// ScheduledExportHandlers 定时导出相关处理器
func (h *ExportHandler) CreateScheduledExport(c *gin.Context) {
	var task models.ScheduledExport
	if err := c.ShouldBindJSON(&task); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	
	if err := h.scheduledExportService.CreateScheduledExport(&task); err != nil {
		response.InternalServerError(c, "Create scheduled export failed")
		return
	}
	
	response.Success(c, task)
}

func (h *ExportHandler) ListScheduledExports(c *gin.Context) {
	tasks, err := h.scheduledExportService.ListScheduledExports()
	if err != nil {
		response.InternalServerError(c, "List scheduled exports failed")
		return
	}
	
	response.Success(c, tasks)
}

func (h *ExportHandler) GetScheduledExport(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	task, err := h.scheduledExportService.GetScheduledExport(uint(id))
	if err != nil {
		response.NotFound(c, "Scheduled export not found")
		return
	}
	
	response.Success(c, task)
}

func (h *ExportHandler) UpdateScheduledExport(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	
	var task models.ScheduledExport
	if err := c.ShouldBindJSON(&task); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	
	task.ID = uint(id)
	if err := h.scheduledExportService.UpdateScheduledExport(&task); err != nil {
		response.InternalServerError(c, "Update scheduled export failed")
		return
	}
	
	response.Success(c, task)
}

func (h *ExportHandler) DeleteScheduledExport(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.scheduledExportService.DeleteScheduledExport(uint(id)); err != nil {
		response.InternalServerError(c, "Delete scheduled export failed")
		return
	}
	
	response.Success(c, gin.H{"message": "Deleted successfully"})
}

func (h *ExportHandler) ExecuteScheduledExport(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.scheduledExportService.ExecuteScheduledExport(uint(id)); err != nil {
		response.InternalServerError(c, "Execute scheduled export failed")
		return
	}
	
	response.Success(c, gin.H{"message": "Executed successfully"})
}

// ReportTemplateHandlers 报表模板相关处理器
func (h *ExportHandler) CreateReportTemplate(c *gin.Context) {
	var template models.ReportTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	
	if err := h.reportTemplateService.CreateReportTemplate(&template); err != nil {
		response.InternalServerError(c, "Create report template failed")
		return
	}
	
	response.Success(c, template)
}

func (h *ExportHandler) ListReportTemplates(c *gin.Context) {
	templates, err := h.reportTemplateService.ListReportTemplates()
	if err != nil {
		response.InternalServerError(c, "List report templates failed")
		return
	}
	
	response.Success(c, templates)
}

func (h *ExportHandler) GetReportTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	template, err := h.reportTemplateService.GetReportTemplate(uint(id))
	if err != nil {
		response.NotFound(c, "Report template not found")
		return
	}
	
	response.Success(c, template)
}

func (h *ExportHandler) UpdateReportTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	
	var template models.ReportTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	
	template.ID = uint(id)
	if err := h.reportTemplateService.UpdateReportTemplate(&template); err != nil {
		response.InternalServerError(c, "Update report template failed")
		return
	}
	
	response.Success(c, template)
}

func (h *ExportHandler) DeleteReportTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.reportTemplateService.DeleteReportTemplate(uint(id)); err != nil {
		response.InternalServerError(c, "Delete report template failed")
		return
	}
	
	response.Success(c, gin.H{"message": "Deleted successfully"})
}

// ExportHistoryHandlers 导出历史相关处理器
func (h *ExportHandler) ListExportHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	
	histories, total, err := h.exportHistoryService.ListExportHistory(page, pageSize)
	if err != nil {
		response.InternalServerError(c, "List export history failed")
		return
	}
	
	response.Success(c, gin.H{
		"total": total,
		"page": page,
		"page_size": pageSize,
		"items": histories,
	})
}
