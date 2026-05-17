package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type LogHandler struct {
	logService   *service.LogService
	statsService *service.StatsService
}

func NewLogHandler() *LogHandler {
	return &LogHandler{
		logService:   service.NewLogService(),
		statsService: service.NewStatsService(),
	}
}

func GetLogHandler() *LogHandler {
	return NewLogHandler()
}

type GetVerificationLogsRequest struct {
	Page          int     `form:"page,default=1"`
	PageSize      int     `form:"page_size,default=20"`
	ApplicationID uint    `form:"application_id"`
	Status        string  `form:"status"`
	CaptchaType   string  `form:"captcha_type"`
	SessionID     string  `form:"session_id"`
	StartDate     string  `form:"start_date"`
	EndDate       string  `form:"end_date"`
	MinRiskScore  float64 `form:"min_risk_score"`
	MaxRiskScore  float64 `form:"max_risk_score"`
	IPAddress     string  `form:"ip_address"`
	RiskLevel     string  `form:"risk_level"`
}

type LogListResponse struct {
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
	Logs       []models.VerificationLog `json:"logs"`
}

type LogListMapResponse struct {
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
	Logs       []map[string]interface{} `json:"logs"`
}

func calculateRiskLevel(riskScore float64) string {
	if riskScore >= 80 {
		return "critical"
	} else if riskScore >= 60 {
		return "high"
	} else if riskScore >= 30 {
		return "medium"
	}
	return "low"
}

func logToMap(log models.VerificationLog) map[string]interface{} {
	return map[string]interface{}{
		"id":           log.ID,
		"session_id":   log.SessionID,
		"captcha_type": log.CaptchaType,
		"status":       log.Status,
		"ip_address":   log.IPAddress,
		"user_agent":   log.UserAgent,
		"risk_score":   log.RiskScore,
		"risk_level":   calculateRiskLevel(log.RiskScore),
		"duration":     log.Duration,
		"created_at":   log.CreatedAt,
	}
}

func GetVerificationLogs(c *gin.Context) {
	handler := GetLogHandler()
	var req GetVerificationLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "无效的查询参数")
		return
	}

	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)
	if req.EndDate != "" {
		endDate = endDate.Add(24 * time.Hour)
	}

	params := service.LogQueryParams{
		Page:          req.Page,
		PageSize:      req.PageSize,
		ApplicationID: req.ApplicationID,
		Status:        req.Status,
		CaptchaType:   req.CaptchaType,
		SessionID:     req.SessionID,
		StartDate:     startDate,
		EndDate:       endDate,
		MinRiskScore:  req.MinRiskScore,
		MaxRiskScore:  req.MaxRiskScore,
		IPAddress:     req.IPAddress,
	}

	result, err := handler.logService.QueryLogs(params)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	if req.RiskLevel != "" {
		filteredLogs := make([]map[string]interface{}, 0, len(result.Logs))
		for _, log := range result.Logs {
			logMap := logToMap(log)
			if logMap["risk_level"] == req.RiskLevel {
				filteredLogs = append(filteredLogs, logMap)
			}
		}

		total := int64(len(filteredLogs))
		totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))
		if totalPages < 1 {
			totalPages = 1
		}

		start := (req.Page - 1) * req.PageSize
		end := start + req.PageSize
		pageLogs := filteredLogs
		if start >= len(filteredLogs) {
			pageLogs = []map[string]interface{}{}
		} else if end > len(filteredLogs) {
			pageLogs = filteredLogs[start:]
		} else {
			pageLogs = filteredLogs[start:end]
		}

		response.Success(c, LogListMapResponse{
			Total:      total,
			Page:       req.Page,
			PageSize:   req.PageSize,
			TotalPages: totalPages,
			Logs:       pageLogs,
		})
		return
	}

	logMaps := make([]map[string]interface{}, len(result.Logs))
	for i, log := range result.Logs {
		logMaps[i] = logToMap(log)
	}

	response.Success(c, LogListMapResponse{
		Total:      result.Total,
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
		Logs:       logMaps,
	})
}

func GetLogDetail(c *gin.Context) {
	handler := GetLogHandler()
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的日志ID")
		return
	}

	log, err := handler.logService.GetLogByID(uint(id))
	if err != nil {
		response.NotFound(c, "日志不存在")
		return
	}

	logMap := map[string]interface{}{
		"id":              log.ID,
		"verification_id": log.VerificationID,
		"session_id":      log.SessionID,
		"application_id":  log.ApplicationID,
		"captcha_type":    log.CaptchaType,
		"status":          log.Status,
		"ip_address":      log.IPAddress,
		"user_agent":      log.UserAgent,
		"risk_score":      log.RiskScore,
		"risk_level":      calculateRiskLevel(log.RiskScore),
		"analysis_result": log.AnalysisResult,
		"duration":        log.Duration,
		"created_at":      log.CreatedAt,
	}

	response.Success(c, logMap)
}

type ExportLogsRequest struct {
	ApplicationID uint   `form:"application_id"`
	Status        string `form:"status"`
	CaptchaType   string `form:"captcha_type"`
	StartDate     string `form:"start_date"`
	EndDate       string `form:"end_date"`
	RiskLevel     string `form:"risk_level"`
	Format        string `form:"format,default=csv"`
}

func ExportLogs(c *gin.Context) {
	handler := GetLogHandler()
	var req ExportLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "无效的查询参数")
		return
	}

	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)
	if req.EndDate != "" {
		endDate = endDate.Add(24 * time.Hour)
	}

	if req.Format == "" {
		req.Format = "csv"
	}

	params := service.LogExportParams{
		ApplicationID: req.ApplicationID,
		Status:        req.Status,
		CaptchaType:   req.CaptchaType,
		StartDate:     startDate,
		EndDate:       endDate,
		Format:        req.Format,
	}

	data, err := handler.logService.ExportLogs(params)
	if err != nil {
		response.InternalServerError(c, "导出失败")
		return
	}

	filename := fmt.Sprintf("verification_logs_%s.csv", time.Now().Format("20060102150405"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(200, "text/csv", data)
}

func DeleteOldLogs(c *gin.Context) {
	handler := GetLogHandler()
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 {
		response.BadRequest(c, "无效的天数参数")
		return
	}

	deleted, err := handler.logService.DeleteOldLogs(days)
	if err != nil {
		response.InternalServerError(c, "删除失败")
		return
	}

	response.Success(c, gin.H{
		"deleted_count": deleted,
	})
}

func GetLogsBySession(c *gin.Context) {
	handler := GetLogHandler()
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.BadRequest(c, "session_id不能为空")
		return
	}

	logs, err := handler.logService.GetLogsBySessionID(sessionID)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	response.Success(c, logs)
}

func GetLogStatistics(c *gin.Context) {
	handler := GetLogHandler()

	stats, err := handler.statsService.GetLogStatistics()
	if err != nil {
		response.InternalServerError(c, "获取统计失败")
		return
	}

	response.Success(c, stats)
}
