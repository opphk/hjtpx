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
	Total      int64                    `json:"total"`
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalPages int                      `json:"total_pages"`
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

// GetVerificationLogs 获取验证日志列表
// @Summary 获取验证日志列表
// @Description 分页获取验证码验证日志
// @Tags 验证日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认20"
// @Param application_id query int false "应用ID"
// @Param status query string false "状态：success, failed, pending"
// @Param captcha_type query string false "验证码类型：slider, click, voice"
// @Param session_id query string false "会话ID"
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Param min_risk_score query number false "最小风险评分"
// @Param max_risk_score query number false "最大风险评分"
// @Param ip_address query string false "IP地址"
// @Param risk_level query string false "风险等级：low, medium, high, critical"
// @Success 200 {object} LogListMapResponse "日志列表"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/logs [get]
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

// GetLogDetail 获取日志详情
// @Summary 获取日志详情
// @Description 根据ID获取验证日志详细信息
// @Tags 验证日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "日志ID"
// @Success 200 {object} map[string]interface{} "日志详情"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 404 {object} map[string]interface{} "日志不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/logs/{id} [get]
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
	IPAddress     string `form:"ip_address"`
	MinRiskScore  float64 `form:"min_risk_score"`
	MaxRiskScore  float64 `form:"max_risk_score"`
}

// ExportLogs 导出日志
// @Summary 导出验证日志
// @Description 导出符合筛选条件的验证日志
// @Tags 验证日志
// @Accept json
// @Produce json/csv
// @Security BearerAuth
// @Param application_id query int false "应用ID"
// @Param status query string false "状态：success, failed, pending"
// @Param captcha_type query string false "验证码类型：slider, click, voice"
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Param risk_level query string false "风险等级：low, medium, high, critical"
// @Param format query string false "导出格式：csv, json，默认csv"
// @Success 200 {file} file "CSV文件"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/logs/export [get]
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
		IPAddress:     req.IPAddress,
		MinRiskScore:  req.MinRiskScore,
		MaxRiskScore:  req.MaxRiskScore,
	}

	data, contentType, err := handler.logService.ExportLogs(params)
	if err != nil {
		response.InternalServerError(c, "导出失败")
		return
	}

	filename := fmt.Sprintf("verification_logs_%s", time.Now().Format("20060102150405"))
	switch req.Format {
	case "xlsx", "excel":
		filename += ".xlsx"
	case "pdf":
		filename += ".pdf"
	case "json":
		filename += ".json"
	default:
		filename += ".csv"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(200, contentType, data)
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

// GetLogsBySession 获取会话日志
// @Summary 获取会话日志
// @Description 根据会话ID获取所有相关日志
// @Tags 验证日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param session_id path string true "会话ID"
// @Success 200 {object} map[string]interface{} "日志列表"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/logs/session/{session_id} [get]
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

// GetLogStatistics 获取日志统计
// @Summary 获取日志统计
// @Description 获取验证日志的统计数据
// @Tags 验证日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "统计数据"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/logs/statistics [get]
func GetLogStatistics(c *gin.Context) {
	handler := GetLogHandler()

	stats, err := handler.statsService.GetLogStatistics()
	if err != nil {
		response.InternalServerError(c, "获取统计失败")
		return
	}

	response.Success(c, stats)
}

// AdvancedSearchLogs 高级搜索日志
// @Summary 高级搜索日志
// @Description 使用高级查询条件搜索验证日志
// @Tags 验证日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body service.AdvancedSearchQuery true "高级搜索查询"
// @Success 200 {object} map[string]interface{} "搜索结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/logs/search [post]
func AdvancedSearchLogs(c *gin.Context) {
	var query service.AdvancedSearchQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		response.BadRequest(c, "无效的查询参数")
		return
	}

	searchService := service.NewAdvancedSearchService()
	result, err := searchService.SearchLogs(query)
	if err != nil {
		response.InternalServerError(c, "搜索失败")
		return
	}

	// 转换为带风险等级的格式
	logs, ok := result.Data.([]models.VerificationLog)
	if ok {
		logMaps := make([]map[string]interface{}, len(logs))
		for i, log := range logs {
			logMaps[i] = logToMap(log)
		}
		result.Data = logMaps
	}

	response.Success(c, result)
}

// SaveLogSearch 保存日志搜索
// @Summary 保存日志搜索
// @Description 保存当前的搜索条件以便后续使用
// @Tags 验证日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body SaveSearchRequest true "保存搜索请求"
// @Success 200 {object} map[string]interface{} "保存结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/logs/save-search [post]
func SaveLogSearch(c *gin.Context) {
	var req SaveSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	adminID, _ := c.Get("admin_id")
	var createdBy uint
	if id, ok := adminID.(uint); ok {
		createdBy = id
	}

	searchService := service.NewAdvancedSearchService()
	savedSearch, err := searchService.SaveSearch(req.Name, "logs", req.Query, req.Description, createdBy)
	if err != nil {
		response.InternalServerError(c, "保存搜索失败")
		return
	}

	response.Success(c, savedSearch)
}

// GetSavedLogSearches 获取保存的日志搜索
// @Summary 获取保存的日志搜索列表
// @Description 获取当前用户保存的所有日志搜索条件
// @Tags 验证日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "搜索列表"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/logs/saved-searches [get]
func GetSavedLogSearches(c *gin.Context) {
	adminID, _ := c.Get("admin_id")
	var createdBy uint
	if id, ok := adminID.(uint); ok {
		createdBy = id
	}

	searchService := service.NewAdvancedSearchService()
	searches, err := searchService.GetSavedSearches("logs", createdBy)
	if err != nil {
		response.InternalServerError(c, "获取保存的搜索失败")
		return
	}

	response.Success(c, searches)
}

// DeleteSavedLogSearch 删除保存的日志搜索
// @Summary 删除保存的日志搜索
// @Description 删除指定保存的搜索条件
// @Tags 验证日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "搜索ID"
// @Success 200 {object} map[string]interface{} "删除结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/logs/saved-searches/{id} [delete]
func DeleteSavedLogSearch(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的搜索ID")
		return
	}

	searchService := service.NewAdvancedSearchService()
	if err := searchService.DeleteSavedSearch(uint(id)); err != nil {
		response.InternalServerError(c, "删除搜索失败")
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

type LogAnalysisRequest struct {
	Days       int  `form:"days,default=7"`
	GroupByApp bool `form:"group_by_app"`
}

func GetLogAnalysisStats(c *gin.Context) {
	handler := GetLogHandler()
	var req LogAnalysisRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req.Days = 7
		req.GroupByApp = false
	}

	stats, err := handler.statsService.GetLogAnalysisStats(req.Days, req.GroupByApp)
	if err != nil {
		response.InternalServerError(c, "获取统计失败")
		return
	}

	response.Success(c, stats)
}

type LogTrendRequest struct {
	Date string `form:"date"`
}

func GetLogTrendStats(c *gin.Context) {
	handler := GetLogHandler()
	var req LogTrendRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req.Date = time.Now().Format("2006-01-02")
	}

	stats, err := handler.statsService.GetLogTrendStats(req.Date)
	if err != nil {
		response.InternalServerError(c, "获取趋势统计失败")
		return
	}

	response.Success(c, stats)
}
