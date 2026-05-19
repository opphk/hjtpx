package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type ComplianceHandler struct {
	service *service.ComplianceService
}

func NewComplianceHandler() *ComplianceHandler {
	return &ComplianceHandler{
		service: service.NewComplianceService(),
	}
}

func (h *ComplianceHandler) GetComplianceStatus(c *gin.Context) {
	status, err := h.service.GetComplianceStatus()
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, status)
}

func (h *ComplianceHandler) GenerateGDPRReport(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Query("user_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "user_id is required")
		return
	}

	report, err := h.service.GenerateGDPRReport(uint(userID))
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, report)
}

func (h *ComplianceHandler) GenerateSOC2Report(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Query("tenant_id"), 10, 64)
	if err != nil {
		tenantID = 0
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		startDate = time.Now().AddDate(0, -1, 0)
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		endDate = time.Now()
	}

	report, err := h.service.GenerateSOC2Report(uint(tenantID), startDate, endDate)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, report)
}

func (h *ComplianceHandler) GenerateSecurityReport(c *gin.Context) {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		startDate = time.Now().AddDate(0, -1, 0)
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		endDate = time.Now()
	}

	report, err := h.service.GenerateSecurityComplianceReport(startDate, endDate)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, report)
}

func (h *ComplianceHandler) GenerateDataProtectionReport(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Query("tenant_id"), 10, 64)
	if err != nil {
		tenantID = 0
	}

	report, err := h.service.GenerateDataProtectionReport(uint(tenantID))
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, report)
}

func (h *ComplianceHandler) ExportReport(c *gin.Context) {
	reportType := c.Query("type")
	if reportType == "" {
		response.BadRequest(c, "report type is required")
		return
	}

	params := make(map[string]interface{})

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseFloat(userIDStr, 64); err == nil {
			params["user_id"] = userID
		}
	}

	if tenantIDStr := c.Query("tenant_id"); tenantIDStr != "" {
		if tenantID, err := strconv.ParseFloat(tenantIDStr, 64); err == nil {
			params["tenant_id"] = tenantID
		}
	}

	if startDate := c.Query("start_date"); startDate != "" {
		params["start_date"] = startDate
	}

	if endDate := c.Query("end_date"); endDate != "" {
		params["end_date"] = endDate
	}

	data, err := h.service.ExportComplianceReport(reportType, params)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=compliance_report_"+reportType+".json")
	c.Data(200, "application/json", data)
}

func (h *ComplianceHandler) GetReportTypes(c *gin.Context) {
	reportTypes := []map[string]interface{}{
		{
			"type":        "gdpr",
			"name":        "GDPR Report",
			"description": "General Data Protection Regulation compliance report",
			"parameters": []string{"user_id"},
		},
		{
			"type":        "soc2",
			"name":        "SOC 2 Report",
			"description": "Service Organization Control 2 Type II compliance report",
			"parameters": []string{"tenant_id", "start_date", "end_date"},
		},
		{
			"type":        "security",
			"name":        "Security Compliance Report",
			"description": "Security compliance and audit report",
			"parameters": []string{"start_date", "end_date"},
		},
		{
			"type":        "dataprotection",
			"name":        "Data Protection Report",
			"description": "Data protection and privacy report",
			"parameters": []string{"tenant_id"},
		},
	}

	response.Success(c, reportTypes)
}

func RegisterComplianceRoutes(r *gin.RouterGroup) {
	handler := NewComplianceHandler()

	compliance := r.Group("/compliance")
	{
		compliance.GET("/status", handler.GetComplianceStatus)
		compliance.GET("/report-types", handler.GetReportTypes)

		compliance.GET("/reports/gdpr", handler.GenerateGDPRReport)
		compliance.GET("/reports/soc2", handler.GenerateSOC2Report)
		compliance.GET("/reports/security", handler.GenerateSecurityReport)
		compliance.GET("/reports/dataprotection", handler.GenerateDataProtectionReport)

		compliance.GET("/reports/export", handler.ExportReport)
	}
}