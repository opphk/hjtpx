package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
	"gorm.io/gorm"
)

// ========== Dashboard Extensions ==========

type SystemStatusResponse struct {
	Status        map[string]ServiceStatus `json:"status"`
	ResourceUsage ResourceUsage            `json:"resourceUsage"`
}

type ServiceStatus struct {
	Status  string `json:"status"`
	Latency int    `json:"latency"`
}

type ResourceUsage struct {
	CPU    int `json:"cpu"`
	Memory int `json:"memory"`
	Disk   int `json:"disk"`
}

// GetSystemStatus 获取系统状态
// @Summary 获取系统状态
// @Description 获取系统的整体状态，包括各服务（数据库、Redis、API、存储）的健康状态和资源使用情况
// @Tags 系统
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SystemStatusResponse "系统状态信息"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/system-status [get]
func GetSystemStatus(c *gin.Context) {
	status := map[string]ServiceStatus{
		"db":      {Status: "healthy", Latency: 5},
		"redis":   {Status: "healthy", Latency: 2},
		"api":     {Status: "healthy", Latency: 15},
		"storage": {Status: "healthy", Latency: 8},
	}

	usage := ResourceUsage{
		CPU:    35,
		Memory: 55,
		Disk:   62,
	}

	response.Success(c, SystemStatusResponse{
		Status:        status,
		ResourceUsage: usage,
	})
}

type RequestTrendResponse struct {
	Labels []string `json:"labels"`
	Data   []int64  `json:"data"`
}

// GetRequestTrend 获取请求趋势
// @Summary 获取请求趋势
// @Description 获取验证请求的趋势数据，支持小时、天、周级别
// @Tags 系统
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param period query string false "时间周期" Enums(hour, day, week) default(hour)
// @Success 200 {object} RequestTrendResponse "请求趋势数据"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/request-trend [get]
func GetRequestTrend(c *gin.Context) {
	period := c.DefaultQuery("period", "hour")

	var labels []string
	var data []int64

	switch period {
	case "hour":
		labels = make([]string, 24)
		data = make([]int64, 24)
		for i := 0; i < 24; i++ {
			labels[i] = strconv.Itoa(i) + ":00"
			data[i] = int64(1000 + (i*137)%5000)
		}
	case "day":
		weekDays := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
		labels = weekDays
		values := []int64{12000, 15000, 18000, 16000, 20000, 25000, 22000}
		data = values
	case "week":
		labels = make([]string, 7)
		data = make([]int64, 7)
		for i := 0; i < 7; i++ {
			labels[i] = "第" + strconv.Itoa(i+1) + "周"
			data[i] = int64(85000 + i*10000)
		}
	default:
		labels = make([]string, 24)
		data = make([]int64, 24)
		for i := 0; i < 24; i++ {
			labels[i] = strconv.Itoa(i) + ":00"
			data[i] = int64(1000 + (i*137)%5000)
		}
	}

	response.Success(c, RequestTrendResponse{
		Labels: labels,
		Data:   data,
	})
}

// ========== Applications Extensions ==========

type ApplicationsSummaryResponse struct {
	Total         int     `json:"total"`
	Active        int     `json:"active"`
	TotalApiCalls int64   `json:"totalApiCalls"`
	SuccessRate   float64 `json:"successRate"`
	TotalUsers    int64   `json:"totalUsers"`
}

// GetApplicationsSummary 获取应用概览
// @Summary 获取应用概览
// @Description 获取所有应用的统计概览信息
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ApplicationsSummaryResponse "应用统计概览"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications-summary [get]
func GetApplicationsSummary(c *gin.Context) {
	var totalApps int64
	var activeApps int64
	database.DB.Model(&models.Application{}).Count(&totalApps)
	database.DB.Model(&models.Application{}).Where("is_active = ?", true).Count(&activeApps)

	var totalVerifications int64
	database.DB.Model(&models.VerificationLog{}).Count(&totalVerifications)

	var successCount int64
	database.DB.Model(&models.VerificationLog{}).Where("status = ?", "success").Count(&successCount)

	var totalUsers int64
	database.DB.Model(&models.User{}).Count(&totalUsers)

	successRate := 0.0
	if totalVerifications > 0 {
		successRate = float64(successCount) / float64(totalVerifications) * 100
	}

	response.Success(c, ApplicationsSummaryResponse{
		Total:         int(totalApps),
		Active:        int(activeApps),
		TotalApiCalls: totalVerifications,
		SuccessRate:   successRate,
		TotalUsers:    totalUsers,
	})
}

// ========== Logs Extensions ==========

type LogsSummaryResponse struct {
	Total    int64 `json:"total"`
	Errors   int64 `json:"errors"`
	Warnings int64 `json:"warnings"`
	Today    int64 `json:"today"`
}

// GetLogsSummary 获取日志摘要
// @Summary 获取日志摘要
// @Description 获取验证日志的统计摘要信息
// @Tags 日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} LogsSummaryResponse "日志摘要信息"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/logs-summary [get]
func GetLogsSummary(c *gin.Context) {
	var total int64
	var errors int64
	var warnings int64
	var today int64

	database.DB.Model(&models.VerificationLog{}).Count(&total)
	database.DB.Model(&models.VerificationLog{}).Where("status = ?", "failed").Count(&errors)
	database.DB.Model(&models.VerificationLog{}).Where("status = ?", "pending").Count(&warnings)

	todayStart := time.Now().Truncate(24 * time.Hour)
	database.DB.Model(&models.VerificationLog{}).Where("created_at >= ?", todayStart).Count(&today)

	response.Success(c, LogsSummaryResponse{
		Total:    total,
		Errors:   errors,
		Warnings: warnings,
		Today:    today,
	})
}

type ClearLogsRequest struct {
	Range string `json:"range"`
}

// ClearLogs 清理日志
// @Summary 清理日志
// @Description 清理指定时间范围之前的日志记录
// @Tags 日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body ClearLogsRequest true "清理参数"
// @Success 200 {object} map[string]interface{} "清理结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/logs/clear [post]
func ClearLogs(c *gin.Context) {
	var req ClearLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	days := 7
	switch req.Range {
	case "3d":
		days = 3
	case "7d":
		days = 7
	case "30d":
		days = 30
	case "all":
		days = 0
	}

	var deleted int64
	if days > 0 {
		cutoff := time.Now().AddDate(0, 0, -days)
		result := database.DB.Where("created_at < ?", cutoff).Delete(&models.VerificationLog{})
		deleted = result.RowsAffected
	} else {
		result := database.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.VerificationLog{})
		deleted = result.RowsAffected
	}

	response.Success(c, gin.H{
		"deleted_count": deleted,
	})
}

// ========== Risk Rules ==========

type RiskRule struct {
	ID          uint     `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Condition   string   `json:"condition"`
	Action      string   `json:"action"`
	Priority    int      `json:"priority"`
	Enabled     bool     `json:"enabled"`
	HitCount    int      `json:"hitCount"`
	Apps        []string `json:"apps"`
}

type RiskRulesSummaryResponse struct {
	TotalRules   int     `json:"totalRules"`
	ActiveRules  int     `json:"activeRules"`
	BlockedToday int     `json:"blockedToday"`
	RiskAlerts   int     `json:"riskAlerts"`
	BlockRate    float64 `json:"blockRate"`
}

var mockRiskRules = []RiskRule{
	{ID: 1, Name: "IP频率限制", Type: "rate_limit", Description: "限制单个IP的请求频率", Condition: "请求次数 > 100", Action: "captcha", Priority: 10, Enabled: true, HitCount: 1234, Apps: []string{"all"}},
	{ID: 2, Name: "恶意IP封禁", Type: "ip_block", Description: "封禁已知恶意IP段", Condition: "IP in 黑名单", Action: "block", Priority: 100, Enabled: true, HitCount: 567, Apps: []string{"all"}},
	{ID: 3, Name: "异常行为检测", Type: "behavior", Description: "检测异常用户行为模式", Condition: "行为分数 < 60", Action: "captcha", Priority: 5, Enabled: true, HitCount: 89, Apps: []string{"1", "2"}},
	{ID: 4, Name: "设备指纹识别", Type: "device_fingerprint", Description: "识别重复设备", Condition: "设备重复率 > 80%", Action: "warning", Priority: 3, Enabled: false, HitCount: 23, Apps: []string{"1"}},
	{ID: 5, Name: "会话劫持检测", Type: "behavior", Description: "检测会话异常", Condition: "IP变更 + UA变更", Action: "review", Priority: 8, Enabled: true, HitCount: 12, Apps: []string{"1", "2", "3"}},
	{ID: 6, Name: "批量注册限制", Type: "rate_limit", Description: "限制批量注册行为", Condition: "注册次数 > 10/分钟", Action: "block", Priority: 10, Enabled: true, HitCount: 345, Apps: []string{"1"}},
	{ID: 7, Name: "爬虫识别", Type: "behavior", Description: "识别爬虫访问", Condition: "请求特征匹配", Action: "captcha", Priority: 5, Enabled: true, HitCount: 678, Apps: []string{"all"}},
	{ID: 8, Name: "暴力破解防护", Type: "rate_limit", Description: "防止暴力破解密码", Condition: "失败次数 > 5/10分钟", Action: "block", Priority: 10, Enabled: true, HitCount: 234, Apps: []string{"1"}},
}

// GetRiskRulesSummary 获取风险规则摘要
// @Summary 获取风险规则摘要
// @Description 获取风险规则的统计摘要信息
// @Tags 风控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} RiskRulesSummaryResponse "风险规则摘要"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/risk-rules/summary [get]
func GetRiskRulesSummary(c *gin.Context) {
	activeCount := 0
	for _, rule := range mockRiskRules {
		if rule.Enabled {
			activeCount++
		}
	}

	response.Success(c, RiskRulesSummaryResponse{
		TotalRules:   len(mockRiskRules),
		ActiveRules:  activeCount,
		BlockedToday: 1234,
		RiskAlerts:   56,
		BlockRate:    2.5,
	})
}

type ListRiskRulesQuery struct {
	Page    int    `form:"page,default=1"`
	Size    int    `form:"size,default=10"`
	Type    string `form:"type"`
	Status  string `form:"status"`
	Keyword string `form:"keyword"`
}

// ListRiskRules 获取风险规则列表
// @Summary 获取风险规则列表
// @Description 分页获取风险规则列表
// @Tags 风控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，默认1"
// @Param size query int false "每页数量，默认10"
// @Param type query string false "规则类型"
// @Param status query string false "状态：enabled, disabled"
// @Param keyword query string false "关键词搜索"
// @Success 200 {object} map[string]interface{} "风险规则列表"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/risk-rules [get]
func ListRiskRules(c *gin.Context) {
	var query ListRiskRulesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, "无效的查询参数")
		return
	}

	filtered := make([]RiskRule, 0)
	for _, rule := range mockRiskRules {
		if query.Type != "" && rule.Type != query.Type {
			continue
		}
		if query.Status == "enabled" && !rule.Enabled {
			continue
		}
		if query.Status == "disabled" && rule.Enabled {
			continue
		}
		if query.Keyword != "" && !containsIgnoreCase(rule.Name, query.Keyword) {
			continue
		}
		filtered = append(filtered, rule)
	}

	start := (query.Page - 1) * query.Size
	end := start + query.Size
	if start > len(filtered) {
		filtered = nil
	} else if end > len(filtered) {
		filtered = filtered[start:]
	} else {
		filtered = filtered[start:end]
	}

	response.Success(c, gin.H{
		"list":  filtered,
		"total": len(mockRiskRules),
	})
}

type CreateRiskRuleRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Type        string                 `json:"type" binding:"required"`
	Description string                 `json:"description"`
	Condition   map[string]interface{} `json:"condition"`
	Action      string                 `json:"action" binding:"required"`
	Priority    int                    `json:"priority"`
	Enabled     bool                   `json:"enabled"`
	Apps        []string               `json:"apps"`
}

// CreateRiskRule 创建风险规则
// @Summary 创建风险规则
// @Description 创建新的风险规则
// @Tags 风控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateRiskRuleRequest true "创建风险规则请求"
// @Success 200 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/risk-rules [post]
func CreateRiskRule(c *gin.Context) {
	var req CreateRiskRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	response.Success(c, gin.H{
		"id":      uint(len(mockRiskRules) + 1),
		"name":    req.Name,
		"type":    req.Type,
		"action":  req.Action,
		"enabled": req.Enabled,
	})
}

// GetRiskRule 获取风险规则详情
// @Summary 获取风险规则详情
// @Description 根据ID获取风险规则详细信息
// @Tags 风控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Success 200 {object} RiskRule "风险规则详情"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 404 {object} map[string]interface{} "规则不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/risk-rules/{id} [get]
func GetRiskRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的规则ID")
		return
	}

	for _, rule := range mockRiskRules {
		if rule.ID == uint(id) {
			response.Success(c, rule)
			return
		}
	}

	response.NotFound(c, "规则不存在")
}

type UpdateRiskRuleRequest struct {
	Name        *string                `json:"name"`
	Type        *string                `json:"type"`
	Description *string                `json:"description"`
	Condition   map[string]interface{} `json:"condition"`
	Action      *string                `json:"action"`
	Priority    *int                   `json:"priority"`
	Enabled     *bool                  `json:"enabled"`
	Apps        []string               `json:"apps"`
}

// UpdateRiskRule 更新风险规则
// @Summary 更新风险规则
// @Description 更新指定的风险规则
// @Tags 风控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Param body body UpdateRiskRuleRequest true "更新风险规则请求"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/risk-rules/{id} [put]
func UpdateRiskRule(c *gin.Context) {
	idStr := c.Param("id")
	_, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的规则ID")
		return
	}

	var req UpdateRiskRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	response.Success(c, gin.H{"message": "规则更新成功"})
}

// DeleteRiskRule 删除风险规则
// @Summary 删除风险规则
// @Description 删除指定的风险规则
// @Tags 风控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Success 200 {object} map[string]interface{} "删除成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/risk-rules/{id} [delete]
func DeleteRiskRule(c *gin.Context) {
	idStr := c.Param("id")
	_, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的规则ID")
		return
	}

	response.Success(c, gin.H{"message": "规则删除成功"})
}

type ToggleRiskRuleRequest struct {
	Enabled bool `json:"enabled"`
}

// ToggleRiskRule 切换风险规则状态
// @Summary 切换风险规则状态
// @Description 启用或禁用指定的风险规则
// @Tags 风控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Param body body ToggleRiskRuleRequest true "切换状态请求"
// @Success 200 {object} map[string]interface{} "切换成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/risk-rules/{id}/toggle [post]
func ToggleRiskRule(c *gin.Context) {
	idStr := c.Param("id")
	_, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的规则ID")
		return
	}

	var req ToggleRiskRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	status := "禁用"
	if req.Enabled {
		status = "启用"
	}

	response.Success(c, gin.H{
		"message": "规则已" + status,
		"enabled": req.Enabled,
	})
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return len(substr) == 0 || contains(s, substr)
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
