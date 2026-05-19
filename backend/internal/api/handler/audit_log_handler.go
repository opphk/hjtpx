package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type AuditLogHandler struct {
	service *service.AuditLogService
}

func NewAuditLogHandler() *AuditLogHandler {
	return &AuditLogHandler{
		service: service.NewAuditLogService(),
	}
}

func (h *AuditLogHandler) QueryLogs(c *gin.Context) {
	var query service.AuditLogQuery

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseUint(userIDStr, 10, 64); err == nil {
			query.UserID = uint(userID)
		}
	}

	query.Username = c.Query("username")
	query.LogType = c.Query("log_type")
	query.Level = c.Query("level")
	query.Action = c.Query("action")
	query.ResourceType = c.Query("resource_type")
	query.ResourceID = c.Query("resource_id")
	query.Status = c.Query("status")
	query.IPAddress = c.Query("ip_address")

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			query.StartTime = &startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			query.EndTime = &endTime
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			query.Offset = offset
		}
	} else {
		query.Offset = 0
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			query.Limit = limit
		}
	} else {
		query.Limit = 20
	}

	logs, total, err := h.service.QueryLogs(query)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"logs":     logs,
		"total":    total,
		"offset":   query.Offset,
		"limit":    query.Limit,
	})
}

func (h *AuditLogHandler) GetLogByID(c *gin.Context) {
	logID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid log ID")
		return
	}

	logEntry, err := h.service.GetLogByID(uint(logID))
	if err != nil {
		response.NotFound(c)
		return
	}

	response.Success(c, logEntry)
}

func (h *AuditLogHandler) GetSecurityEvents(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Query("user_id"), 10, 64)
	if err != nil {
		userID = 0
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil {
		limit = 50
	}

	events, err := h.service.GetSecurityEvents(uint(userID), limit)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, events)
}

func (h *AuditLogHandler) GetAuthenticationHistory(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		response.BadRequest(c, "username is required")
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil {
		limit = 50
	}

	history, err := h.service.GetAuthenticationHistory(username, limit)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, history)
}

func (h *AuditLogHandler) GetAccessSummary(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Query("user_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "user_id is required")
		return
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		startDate = time.Now().AddDate(0, 0, -30)
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		endDate = time.Now()
	}

	summary, err := h.service.GetAccessSummary(uint(userID), startDate, endDate)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, summary)
}

func (h *AuditLogHandler) GetSystemAccessReport(c *gin.Context) {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		startDate = time.Now().AddDate(0, 0, -30)
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		endDate = time.Now()
	}

	report, err := h.service.GetSystemAccessReport(startDate, endDate)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, report)
}

func (h *AuditLogHandler) ExportLogs(c *gin.Context) {
	var query service.AuditLogQuery

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseUint(userIDStr, 10, 64); err == nil {
			query.UserID = uint(userID)
		}
	}

	query.LogType = c.Query("log_type")
	query.Level = c.Query("level")

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			query.StartTime = &startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			query.EndTime = &endTime
		}
	}

	data, err := h.service.ExportLogs(query)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=audit_logs.json")
	c.Data(200, "application/json", data)
}

func (h *AuditLogHandler) PurgeOldLogs(c *gin.Context) {
	days, err := strconv.Atoi(c.Query("days"))
	if err != nil || days <= 0 {
		response.BadRequest(c, "days must be a positive integer")
		return
	}

	olderThan := time.Now().AddDate(0, 0, -days)
	rowsAffected, err := h.service.PurgeOldLogs(olderThan)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":        "Old logs purged successfully",
		"rows_affected": rowsAffected,
		"older_than":     olderThan.Format(time.RFC3339),
	})
}

func RegisterAuditLogRoutes(r *gin.RouterGroup) {
	handler := NewAuditLogHandler()

	audit := r.Group("/audit")
	{
		audit.GET("/logs", handler.QueryLogs)
		audit.GET("/logs/:id", handler.GetLogByID)
		audit.GET("/logs/export", handler.ExportLogs)
		audit.DELETE("/logs/purge", handler.PurgeOldLogs)

		audit.GET("/security-events", handler.GetSecurityEvents)
		audit.GET("/authentication-history", handler.GetAuthenticationHistory)
		audit.GET("/access-summary", handler.GetAccessSummary)
		audit.GET("/system-report", handler.GetSystemAccessReport)
	}
}