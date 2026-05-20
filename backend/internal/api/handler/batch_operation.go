package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type BatchOperationHandler struct{}

type BatchOperationRequest struct {
	Operation string   `json:"operation" binding:"required"`
	IDs      []uint   `json:"ids" binding:"required,min=1"`
	Data     []string `json:"data,omitempty"`
}

type BatchOperationResponse struct {
	Total     int      `json:"total"`
	Success   int      `json:"success"`
	Failed    int      `json:"failed"`
	FailedIDs []uint   `json:"failedIds"`
	Errors    []string `json:"errors,omitempty"`
	Message   string  `json:"message"`
}

func NewBatchOperationHandler() *BatchOperationHandler {
	return &BatchOperationHandler{}
}

func (h *BatchOperationHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/batch/applications", h.BatchUpdateApplications)
	router.POST("/batch/users", h.BatchUpdateUsers)
	router.POST("/batch/logs", h.BatchDeleteLogs)
	router.POST("/batch/risk-rules", h.BatchUpdateRiskRules)
	router.POST("/batch/blacklist", h.BatchUpdateBlacklist)
	router.POST("/batch/export", h.BatchExport)
}

func (h *BatchOperationHandler) BatchUpdateApplications(c *gin.Context) {
	var req BatchOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	var result BatchOperationResponse
	result.Total = len(req.IDs)
	result.Success = 0
	result.Failed = 0
	result.FailedIDs = []uint{}
	result.Errors = []string{}

	switch req.Operation {
	case "enable":
		result = h.batchEnableApplications(req.IDs)
	case "disable":
		result = h.batchDisableApplications(req.IDs)
	case "delete":
		result = h.batchDeleteApplications(req.IDs)
	case "update":
		result = h.batchUpdateApplications(req.IDs, req.Data)
	default:
		response.BadRequest(c, "不支持的操作类型")
		return
	}

	response.Success(c, result)
}

func (h *BatchOperationHandler) batchEnableApplications(ids []uint) BatchOperationResponse {
	result := BatchOperationResponse{
		Total:     len(ids),
		Success:   0,
		Failed:    0,
		FailedIDs: []uint{},
		Errors:    []string{},
	}

	for _, id := range ids {
		if err := h.updateApplicationStatus(id, true); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, id)
			result.Errors = append(result.Errors, "ID "+strconv.Itoa(int(id))+": "+err.Error())
		} else {
			result.Success++
		}
	}

	if result.Failed == 0 {
		result.Message = "成功启用 " + strconv.Itoa(result.Success) + " 个应用"
	} else {
		result.Message = "启用完成：成功 " + strconv.Itoa(result.Success) + "，失败 " + strconv.Itoa(result.Failed)
	}

	return result
}

func (h *BatchOperationHandler) batchDisableApplications(ids []uint) BatchOperationResponse {
	result := BatchOperationResponse{
		Total:     len(ids),
		Success:   0,
		Failed:    0,
		FailedIDs: []uint{},
		Errors:    []string{},
	}

	for _, id := range ids {
		if err := h.updateApplicationStatus(id, false); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, id)
			result.Errors = append(result.Errors, "ID "+strconv.Itoa(int(id))+": "+err.Error())
		} else {
			result.Success++
		}
	}

	if result.Failed == 0 {
		result.Message = "成功禁用 " + strconv.Itoa(result.Success) + " 个应用"
	} else {
		result.Message = "禁用完成：成功 " + strconv.Itoa(result.Success) + "，失败 " + strconv.Itoa(result.Failed)
	}

	return result
}

func (h *BatchOperationHandler) batchDeleteApplications(ids []uint) BatchOperationResponse {
	result := BatchOperationResponse{
		Total:     len(ids),
		Success:   0,
		Failed:    0,
		FailedIDs: []uint{},
		Errors:    []string{},
	}

	for _, id := range ids {
		if err := h.deleteApplication(id); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, id)
			result.Errors = append(result.Errors, "ID "+strconv.Itoa(int(id))+": "+err.Error())
		} else {
			result.Success++
		}
	}

	if result.Failed == 0 {
		result.Message = "成功删除 " + strconv.Itoa(result.Success) + " 个应用"
	} else {
		result.Message = "删除完成：成功 " + strconv.Itoa(result.Success) + "，失败 " + strconv.Itoa(result.Failed)
	}

	return result
}

func (h *BatchOperationHandler) batchUpdateApplications(ids []uint, data []string) BatchOperationResponse {
	result := BatchOperationResponse{
		Total:     len(ids),
		Success:   0,
		Failed:    0,
		FailedIDs: []uint{},
		Errors:    []string{},
	}

	for _, id := range ids {
		if err := h.updateApplicationConfig(id, data); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, id)
			result.Errors = append(result.Errors, "ID "+strconv.Itoa(int(id))+": "+err.Error())
		} else {
			result.Success++
		}
	}

	if result.Failed == 0 {
		result.Message = "成功更新 " + strconv.Itoa(result.Success) + " 个应用"
	} else {
		result.Message = "更新完成：成功 " + strconv.Itoa(result.Success) + "，失败 " + strconv.Itoa(result.Failed)
	}

	return result
}

func (h *BatchOperationHandler) updateApplicationStatus(id uint, enabled bool) error {
	if err := database.DB.Model(&models.Application{}).Where("id = ?", id).Update("enabled", enabled).Error; err != nil {
		return err
	}
	return nil
}

func (h *BatchOperationHandler) deleteApplication(id uint) error {
	if err := database.DB.Delete(&models.Application{}, id).Error; err != nil {
		return err
	}
	return nil
}

func (h *BatchOperationHandler) updateApplicationConfig(id uint, data []string) error {
	if len(data) > 0 {
		if err := database.DB.Model(&models.Application{}).Where("id = ?", id).Update("config", data[0]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (h *BatchOperationHandler) BatchUpdateUsers(c *gin.Context) {
	var req BatchOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	var result BatchOperationResponse
	result.Total = len(req.IDs)
	result.Success = 0
	result.Failed = 0
	result.FailedIDs = []uint{}
	result.Errors = []string{}

	switch req.Operation {
	case "enable":
		result = h.batchEnableUsers(req.IDs)
	case "disable":
		result = h.batchDisableUsers(req.IDs)
	case "delete":
		result = h.batchDeleteUsers(req.IDs)
	default:
		response.BadRequest(c, "不支持的操作类型")
		return
	}

	response.Success(c, result)
}

func (h *BatchOperationHandler) batchEnableUsers(ids []uint) BatchOperationResponse {
	result := BatchOperationResponse{
		Total:     len(ids),
		Success:   0,
		Failed:    0,
		FailedIDs: []uint{},
		Errors:    []string{},
	}

	for _, id := range ids {
		if err := h.updateUserStatus(id, "active"); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, id)
			result.Errors = append(result.Errors, "ID "+strconv.Itoa(int(id))+": "+err.Error())
		} else {
			result.Success++
		}
	}

	if result.Failed == 0 {
		result.Message = "成功启用 " + strconv.Itoa(result.Success) + " 个用户"
	} else {
		result.Message = "启用完成：成功 " + strconv.Itoa(result.Success) + "，失败 " + strconv.Itoa(result.Failed)
	}

	return result
}

func (h *BatchOperationHandler) batchDisableUsers(ids []uint) BatchOperationResponse {
	result := BatchOperationResponse{
		Total:     len(ids),
		Success:   0,
		Failed:    0,
		FailedIDs: []uint{},
		Errors:    []string{},
	}

	for _, id := range ids {
		if err := h.updateUserStatus(id, "disabled"); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, id)
			result.Errors = append(result.Errors, "ID "+strconv.Itoa(int(id))+": "+err.Error())
		} else {
			result.Success++
		}
	}

	if result.Failed == 0 {
		result.Message = "成功禁用 " + strconv.Itoa(result.Success) + " 个用户"
	} else {
		result.Message = "禁用完成：成功 " + strconv.Itoa(result.Success) + "，失败 " + strconv.Itoa(result.Failed)
	}

	return result
}

func (h *BatchOperationHandler) batchDeleteUsers(ids []uint) BatchOperationResponse {
	result := BatchOperationResponse{
		Total:     len(ids),
		Success:   0,
		Failed:    0,
		FailedIDs: []uint{},
		Errors:    []string{},
	}

	for _, id := range ids {
		if err := h.deleteUser(id); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, id)
			result.Errors = append(result.Errors, "ID "+strconv.Itoa(int(id))+": "+err.Error())
		} else {
			result.Success++
		}
	}

	if result.Failed == 0 {
		result.Message = "成功删除 " + strconv.Itoa(result.Success) + " 个用户"
	} else {
		result.Message = "删除完成：成功 " + strconv.Itoa(result.Success) + "，失败 " + strconv.Itoa(result.Failed)
	}

	return result
}

func (h *BatchOperationHandler) updateUserStatus(id uint, status string) error {
	if err := database.DB.Model(&models.User{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return err
	}
	return nil
}

func (h *BatchOperationHandler) deleteUser(id uint) error {
	if err := database.DB.Delete(&models.User{}, id).Error; err != nil {
		return err
	}
	return nil
}

func (h *BatchOperationHandler) BatchDeleteLogs(c *gin.Context) {
	var req BatchOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if req.Operation != "delete" {
		response.BadRequest(c, "日志仅支持删除操作")
		return
	}

	var result BatchOperationResponse
	result.Total = len(req.IDs)
	result.Success = 0
	result.Failed = 0
	result.FailedIDs = []uint{}
	result.Errors = []string{}

	for _, id := range req.IDs {
		if err := h.deleteLog(id); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, id)
			result.Errors = append(result.Errors, "ID "+strconv.Itoa(int(id))+": "+err.Error())
		} else {
			result.Success++
		}
	}

	if result.Failed == 0 {
		result.Message = "成功删除 " + strconv.Itoa(result.Success) + " 条日志"
	} else {
		result.Message = "删除完成：成功 " + strconv.Itoa(result.Success) + "，失败 " + strconv.Itoa(result.Failed)
	}

	response.Success(c, result)
}

func (h *BatchOperationHandler) deleteLog(id uint) error {
	if err := database.DB.Delete(&models.VerificationLog{}, id).Error; err != nil {
		return err
	}
	return nil
}

func (h *BatchOperationHandler) BatchUpdateRiskRules(c *gin.Context) {
	var req BatchOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	var result BatchOperationResponse
	result.Total = len(req.IDs)
	result.Success = 0
	result.Failed = 0
	result.FailedIDs = []uint{}
	result.Errors = []string{}

	switch req.Operation {
	case "enable":
		result = h.batchEnableRiskRules(req.IDs)
	case "disable":
		result = h.batchDisableRiskRules(req.IDs)
	case "delete":
		result = h.batchDeleteRiskRules(req.IDs)
	default:
		response.BadRequest(c, "不支持的操作类型")
		return
	}

	response.Success(c, result)
}

func (h *BatchOperationHandler) batchEnableRiskRules(ids []uint) BatchOperationResponse {
	result := BatchOperationResponse{
		Total:     len(ids),
		Success:   0,
		Failed:    0,
		FailedIDs: []uint{},
		Errors:    []string{},
	}

	for _, id := range ids {
		if err := h.updateRiskRuleStatus(id, true); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, id)
			result.Errors = append(result.Errors, "ID "+strconv.Itoa(int(id))+": "+err.Error())
		} else {
			result.Success++
		}
	}

	if result.Failed == 0 {
		result.Message = "成功启用 " + strconv.Itoa(result.Success) + " 条规则"
	} else {
		result.Message = "启用完成：成功 " + strconv.Itoa(result.Success) + "，失败 " + strconv.Itoa(result.Failed)
	}

	return result
}

func (h *BatchOperationHandler) batchDisableRiskRules(ids []uint) BatchOperationResponse {
	result := BatchOperationResponse{
		Total:     len(ids),
		Success:   0,
		Failed:    0,
		FailedIDs: []uint{},
		Errors:    []string{},
	}

	for _, id := range ids {
		if err := h.updateRiskRuleStatus(id, false); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, id)
			result.Errors = append(result.Errors, "ID "+strconv.Itoa(int(id))+": "+err.Error())
		} else {
			result.Success++
		}
	}

	if result.Failed == 0 {
		result.Message = "成功禁用 " + strconv.Itoa(result.Success) + " 条规则"
	} else {
		result.Message = "禁用完成：成功 " + strconv.Itoa(result.Success) + "，失败 " + strconv.Itoa(result.Failed)
	}

	return result
}

func (h *BatchOperationHandler) batchDeleteRiskRules(ids []uint) BatchOperationResponse {
	result := BatchOperationResponse{
		Total:     len(ids),
		Success:   0,
		Failed:    0,
		FailedIDs: []uint{},
		Errors:    []string{},
	}

	for _, id := range ids {
		if err := h.deleteRiskRule(id); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, id)
			result.Errors = append(result.Errors, "ID "+strconv.Itoa(int(id))+": "+err.Error())
		} else {
			result.Success++
		}
	}

	if result.Failed == 0 {
		result.Message = "成功删除 " + strconv.Itoa(result.Success) + " 条规则"
	} else {
		result.Message = "删除完成：成功 " + strconv.Itoa(result.Success) + "，失败 " + strconv.Itoa(result.Failed)
	}

	return result
}

func (h *BatchOperationHandler) updateRiskRuleStatus(id uint, enabled bool) error {
	if err := database.DB.Model(&models.RiskRule{}).Where("id = ?", id).Update("enabled", enabled).Error; err != nil {
		return err
	}
	return nil
}

func (h *BatchOperationHandler) deleteRiskRule(id uint) error {
	if err := database.DB.Delete(&models.RiskRule{}, id).Error; err != nil {
		return err
	}
	return nil
}

func (h *BatchOperationHandler) BatchUpdateBlacklist(c *gin.Context) {
	var req BatchOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	var result BatchOperationResponse
	result.Total = len(req.IDs)
	result.Success = 0
	result.Failed = 0
	result.FailedIDs = []uint{}
	result.Errors = []string{}

	switch req.Operation {
	case "delete":
		result = h.batchDeleteBlacklist(req.IDs)
	default:
		response.BadRequest(c, "不支持的操作类型")
		return
	}

	response.Success(c, result)
}

func (h *BatchOperationHandler) batchDeleteBlacklist(ids []uint) BatchOperationResponse {
	result := BatchOperationResponse{
		Total:     len(ids),
		Success:   0,
		Failed:    0,
		FailedIDs: []uint{},
		Errors:    []string{},
	}

	for _, id := range ids {
		if err := h.deleteBlacklist(id); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, id)
			result.Errors = append(result.Errors, "ID "+strconv.Itoa(int(id))+": "+err.Error())
		} else {
			result.Success++
		}
	}

	if result.Failed == 0 {
		result.Message = "成功删除 " + strconv.Itoa(result.Success) + " 条黑名单"
	} else {
		result.Message = "删除完成：成功 " + strconv.Itoa(result.Success) + "，失败 " + strconv.Itoa(result.Failed)
	}

	return result
}

func (h *BatchOperationHandler) deleteBlacklist(id uint) error {
	if err := database.DB.Delete(&models.Blacklist{}, id).Error; err != nil {
		return err
	}
	return nil
}

type BatchExportRequest struct {
	Type      string   `json:"type" binding:"required"`
	Format    string   `json:"format" binding:"required"`
	IDs       []uint   `json:"ids,omitempty"`
	StartDate string   `json:"startDate,omitempty"`
	EndDate   string   `json:"endDate,omitempty"`
	Fields    []string `json:"fields,omitempty"`
}

func (h *BatchOperationHandler) BatchExport(c *gin.Context) {
	var req BatchExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	exportData, err := h.generateExportData(req)
	if err != nil {
		response.InternalServerError(c, "生成导出数据失败: "+err.Error())
		return
	}

	filename := h.generateExportFilename(req.Type, req.Format)

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", h.getContentType(req.Format))

	switch req.Format {
	case "csv":
		h.exportCSV(c, exportData)
	case "excel":
		h.exportExcel(c, exportData)
	case "json":
		h.exportJSON(c, exportData)
	case "pdf":
		h.exportPDF(c, exportData)
	default:
		response.BadRequest(c, "不支持的导出格式")
	}
}

func (h *BatchOperationHandler) generateExportData(req BatchExportRequest) ([]map[string]interface{}, error) {
	switch req.Type {
	case "applications":
		return h.getApplicationsExportData(req)
	case "users":
		return h.getUsersExportData(req)
	case "logs":
		return h.getLogsExportData(req)
	case "risk-rules":
		return h.getRiskRulesExportData(req)
	case "blacklist":
		return h.getBlacklistExportData(req)
	default:
		return nil, nil
	}
}

func (h *BatchOperationHandler) getApplicationsExportData(req BatchExportRequest) ([]map[string]interface{}, error) {
	var apps []models.Application
	query := database.DB.Model(&models.Application{})

	if len(req.IDs) > 0 {
		query = query.Where("id IN ?", req.IDs)
	}

	if err := query.Find(&apps).Error; err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(apps))
	for i, app := range apps {
		data[i] = map[string]interface{}{
			"ID":           app.ID,
			"Name":         app.Name,
			"APIKey":       app.APIKey,
			"IsActive":     app.IsActive,
			"CreatedAt":    app.CreatedAt,
			"UpdatedAt":    app.UpdatedAt,
		}
	}

	return data, nil
}

func (h *BatchOperationHandler) getUsersExportData(req BatchExportRequest) ([]map[string]interface{}, error) {
	var users []models.User
	query := database.DB.Model(&models.User{})

	if len(req.IDs) > 0 {
		query = query.Where("id IN ?", req.IDs)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(users))
	for i, user := range users {
		data[i] = map[string]interface{}{
			"ID":           user.ID,
			"Username":     user.Username,
			"Email":         user.Email,
			"Status":       user.Status,
			"CreatedAt":    user.CreatedAt,
			"UpdatedAt":    user.UpdatedAt,
		}
	}

	return data, nil
}

func (h *BatchOperationHandler) getLogsExportData(req BatchExportRequest) ([]map[string]interface{}, error) {
	var logs []models.VerificationLog
	query := database.DB.Model(&models.VerificationLog{})

	if len(req.IDs) > 0 {
		query = query.Where("id IN ?", req.IDs)
	}

	if req.StartDate != "" && req.EndDate != "" {
		query = query.Where("created_at BETWEEN ? AND ?", req.StartDate, req.EndDate)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		data[i] = map[string]interface{}{
			"ID":               log.ID,
			"VerificationID":   log.VerificationID,
			"ApplicationID":  log.ApplicationID,
			"Status":           log.Status,
			"CreatedAt":        log.CreatedAt,
		}
	}

	return data, nil
}

func (h *BatchOperationHandler) getRiskRulesExportData(req BatchExportRequest) ([]map[string]interface{}, error) {
	var rules []models.RiskRule
	query := database.DB.Model(&models.RiskRule{})

	if len(req.IDs) > 0 {
		query = query.Where("id IN ?", req.IDs)
	}

	if err := query.Find(&rules).Error; err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(rules))
	for i, rule := range rules {
		data[i] = map[string]interface{}{
			"ID":           rule.ID,
			"Name":         rule.Name,
			"RuleType":     rule.RuleType,
			"IsEnabled":    rule.IsEnabled,
			"Priority":     rule.Priority,
			"CreatedAt":    rule.CreatedAt,
		}
	}

	return data, nil
}

func (h *BatchOperationHandler) getBlacklistExportData(req BatchExportRequest) ([]map[string]interface{}, error) {
	var entries []models.Blacklist
	query := database.DB.Model(&models.Blacklist{})

	if len(req.IDs) > 0 {
		query = query.Where("id IN ?", req.IDs)
	}

	if err := query.Find(&entries).Error; err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(entries))
	for i, entry := range entries {
		data[i] = map[string]interface{}{
			"ID":           entry.ID,
			"Target":       entry.Target,
			"Type":         entry.Type,
			"Reason":       entry.Reason,
			"CreatedAt":    entry.CreatedAt,
		}
	}

	return data, nil
}

func (h *BatchOperationHandler) generateExportFilename(dataType, format string) string {
	return dataType + "_export_" + strconv.FormatInt(time.Now().Unix(), 10) + "." + format
}

func (h *BatchOperationHandler) getContentType(format string) string {
	switch format {
	case "csv":
		return "text/csv"
	case "excel":
		return "application/vnd.ms-excel"
	case "json":
		return "application/json"
	case "pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

func (h *BatchOperationHandler) exportCSV(c *gin.Context, data []map[string]interface{}) {
	if len(data) == 0 {
		c.String(200, "")
		return
	}

	csv := "ID"
	for key := range data[0] {
		csv += "," + key
	}
	csv += "\n"

	for _, row := range data {
		line := ""
		for _, value := range row {
			line += "," + fmt.Sprintf("%v", value)
		}
		csv += line[1:] + "\n"
	}

	c.String(200, csv)
}

func (h *BatchOperationHandler) exportExcel(c *gin.Context, data []map[string]interface{}) {
	if len(data) == 0 {
		c.String(200, "")
		return
	}

	excel := "ID"
	for key := range data[0] {
		excel += "\t" + key
	}
	excel += "\n"

	for _, row := range data {
		line := ""
		for _, value := range row {
			line += "\t" + fmt.Sprintf("%v", value)
		}
		excel += line[1:] + "\n"
	}

	c.String(200, excel)
}

func (h *BatchOperationHandler) exportJSON(c *gin.Context, data []map[string]interface{}) {
	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data":    data,
	})
}

func (h *BatchOperationHandler) exportPDF(c *gin.Context, data []map[string]interface{}) {
	c.JSON(200, gin.H{
		"code":    0,
		"message": "PDF导出功能开发中",
		"data":    data,
	})
}
