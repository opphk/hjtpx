package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type ApplicationHandler struct {
	applicationService *service.ApplicationService
}

func NewApplicationHandler() *ApplicationHandler {
	return &ApplicationHandler{
		applicationService: service.NewApplicationService(),
	}
}

// CreateApplicationRequest 创建应用请求
// @Description 创建新应用的请求参数
type CreateApplicationRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`    // 应用名称
	UserID      uint   `json:"user_id" binding:"required"`                // 用户ID
	Description string `json:"description" binding:"max=1000"`           // 应用描述
	Domain      string `json:"domain" binding:"max=255"`                 // 域名
	Website     string `json:"website" binding:"max=255"`                // 网站URL
}

// UpdateApplicationRequest 更新应用请求
// @Description 更新应用的请求参数
type UpdateApplicationRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=255"`            // 应用名称
	Description *string `json:"description" binding:"omitempty,max=1000"`     // 应用描述
	IsActive    *bool   `json:"is_active"`                                   // 是否启用
	Domain      *string `json:"domain" binding:"omitempty,max=255"`           // 域名
	Website     *string `json:"website" binding:"omitempty,max=255"`          // 网站URL
}

// ListApplicationsQuery 应用列表查询参数
// @Description 应用列表查询参数
type ListApplicationsQuery struct {
	Page      int    `form:"page,default=1"`       // 页码
	PageSize  int    `form:"page_size,default=10"` // 每页数量
	Keyword   string `form:"keyword"`               // 关键词
	UserID    uint   `form:"user_id"`               // 用户ID
	IsActive  *bool  `form:"is_active"`             // 是否启用
	SortField string `form:"sort_field"`           // 排序字段
	SortOrder string `form:"sort_order"`           // 排序方式
}

// UpdateConfigRequest 更新配置请求
// @Description 更新应用配置的请求参数
type UpdateConfigRequest struct {
	CaptchaTypes         []string               `json:"captcha_types"`           // 验证码类型列表
	MaxVerifyPerMinute   int                    `json:"max_verify_per_minute"`   // 每分钟最大验证次数
	MaxVerifyPerDay      int                    `json:"max_verify_per_day"`      // 每天最大验证次数
	AllowedIPs           []string               `json:"allowed_ips"`             // 允许的IP地址列表
	BlockRefusedRequests bool                   `json:"block_refused_requests"`  // 阻止被拒绝的请求
	CustomSettings       map[string]interface{} `json:"custom_settings"`         // 自定义设置
}

// SaveSearchRequest 保存搜索请求
// @Description 保存搜索的请求参数
type SaveSearchRequest struct {
	Name        string                      `json:"name" binding:"required"`           // 搜索名称
	Description string                      `json:"description"`                       // 搜索描述
	Query       service.AdvancedSearchQuery `json:"query" binding:"required"`          // 搜索查询条件
}

// ImportApplicationRequest 导入应用请求
// @Description 导入应用配置的请求参数
type ImportApplicationRequest struct {
	Name        string                        `json:"name" binding:"required"`          // 应用名称
	Description string                        `json:"description"`                       // 应用描述
	Domain      string                        `json:"domain"`                          // 域名
	Website     string                        `json:"website"`                         // 网站URL
	Config      map[string]interface{}        `json:"config"`                          // 应用配置
	UserID      uint                          `json:"user_id"`                         // 用户ID
}

// CloneApplicationRequest 克隆应用请求
// @Description 克隆应用的请求参数
type CloneApplicationRequest struct {
	NewName string `json:"new_name" binding:"required"` // 新应用名称
}

// BatchDeleteRequest 批量删除请求
// @Description 批量删除的请求参数
type BatchDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"` // 应用ID列表
}

// BatchUpdateRequest 批量更新请求
// @Description 批量更新的请求参数
type BatchUpdateRequest struct {
	IDs      []uint               `json:"ids" binding:"required,min=1"`    // 应用ID列表
	IsActive *bool                 `json:"is_active"`                       // 是否启用
	Config   map[string]interface{} `json:"config"`                         // 应用配置
}

func GetApplicationHandler() *ApplicationHandler {
	return NewApplicationHandler()
}

// ListApplications 获取应用列表
// @Summary 获取应用列表
// @Description 分页获取应用列表
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认10"
// @Param keyword query string false "关键词搜索"
// @Param user_id query int false "用户ID"
// @Param is_active query bool false "是否启用"
// @Param sort_field query string false "排序字段：name, created_at"
// @Param sort_order query string false "排序方式：asc, desc"
// @Success 200 {object} map[string]interface{} "应用列表"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications [get]
func ListApplications(c *gin.Context) {
	var query ListApplicationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, "invalid query parameters: "+err.Error())
		return
	}

	filter := &service.ListApplicationsFilter{
		Page:      query.Page,
		PageSize:  query.PageSize,
		Keyword:   query.Keyword,
		UserID:    query.UserID,
		IsActive:  query.IsActive,
		SortField: query.SortField,
		SortOrder: query.SortOrder,
	}

	result, err := service.NewApplicationService().ListApplications(filter)
	if err != nil {
		response.InternalServerError(c, "failed to list applications: "+err.Error())
		return
	}

	response.Success(c, result)
}

// CreateApplication 创建应用
// @Summary 创建应用
// @Description 创建新的应用
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateApplicationRequest true "创建应用请求"
// @Success 200 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 404 {object} map[string]interface{} "用户不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications [post]
func CreateApplication(c *gin.Context) {
	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}

	input := &service.CreateApplicationInput{
		Name:        req.Name,
		UserID:      req.UserID,
		Description: req.Description,
		Domain:      req.Domain,
		Website:     req.Website,
	}

	app, err := service.NewApplicationService().CreateApplication(input)
	if err != nil {
		if err == service.ErrUserNotFoundApp {
			response.NotFound(c, "user not found")
			return
		}
		if err == service.ErrInvalidInput {
			response.BadRequest(c, "invalid application name")
			return
		}
		response.InternalServerError(c, "failed to create application: "+err.Error())
		return
	}

	response.Success(c, service.ToApplicationResponse(app))
}

func UpdateApplication(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	var req UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}

	input := &service.UpdateApplicationInput{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
		Domain:      req.Domain,
		Website:     req.Website,
	}

	app, err := service.NewApplicationService().UpdateApplication(uint(id), input)
	if err != nil {
		if err == service.ErrApplicationNotFound {
			response.NotFound(c, "application not found")
			return
		}
		response.InternalServerError(c, "failed to update application: "+err.Error())
		return
	}

	response.Success(c, service.ToApplicationResponse(app))
}

// DeleteApplication 删除应用
// @Summary 删除应用
// @Description 删除指定的应用
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "应用ID"
// @Success 200 {object} map[string]interface{} "删除成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 404 {object} map[string]interface{} "应用不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications/{id} [delete]
func DeleteApplication(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	err = service.NewApplicationService().DeleteApplication(uint(id))
	if err != nil {
		if err == service.ErrApplicationNotFound {
			response.NotFound(c, "application not found")
			return
		}
		response.InternalServerError(c, "failed to delete application: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "application deleted successfully"})
}

func RegenerateApplicationKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	app, oldKey, err := service.NewApplicationService().RegenerateAPIKey(uint(id))
	if err != nil {
		if err == service.ErrApplicationNotFound {
			response.NotFound(c, "application not found")
			return
		}
		response.InternalServerError(c, "failed to regenerate API key: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"application": service.ToApplicationResponse(app),
		"old_key":     oldKey,
		"warning":     "please save the new API key securely, the old key has been invalidated",
	})
}

// GetApplicationConfig 获取应用配置
// @Summary 获取应用配置
// @Description 获取指定应用的配置信息
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "应用ID"
// @Success 200 {object} map[string]interface{} "应用配置"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 404 {object} map[string]interface{} "应用不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications/{id}/config [get]
func GetApplicationConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	config, err := service.NewApplicationService().GetApplicationConfig(uint(id))
	if err != nil {
		if err == service.ErrApplicationNotFound {
			response.NotFound(c, "application not found")
			return
		}
		response.InternalServerError(c, "failed to get application config: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"application_id": id,
		"config":         config,
	})
}

// UpdateApplicationConfig 更新应用配置
// @Summary 更新应用配置
// @Description 更新指定应用的配置信息
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "应用ID"
// @Param body body UpdateConfigRequest true "更新配置请求"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 404 {object} map[string]interface{} "应用不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications/{id}/config [put]
func UpdateApplicationConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}

	config := &service.ApplicationConfig{
		CaptchaTypes:         req.CaptchaTypes,
		MaxVerifyPerMinute:   req.MaxVerifyPerMinute,
		MaxVerifyPerDay:      req.MaxVerifyPerDay,
		AllowedIPs:           req.AllowedIPs,
		BlockRefusedRequests: req.BlockRefusedRequests,
		CustomSettings:       req.CustomSettings,
	}

	app, err := service.NewApplicationService().UpdateApplicationConfig(uint(id), config)
	if err != nil {
		if err == service.ErrApplicationNotFound {
			response.NotFound(c, "application not found")
			return
		}
		response.InternalServerError(c, "failed to update application config: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":     "configuration updated successfully",
		"application": service.ToApplicationResponse(app),
	})
}

func GetApplicationStatistics(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	stats, err := service.NewApplicationService().GetApplicationStatistics(uint(id))
	if err != nil {
		if err == service.ErrApplicationNotFound {
			response.NotFound(c, "application not found")
			return
		}
		response.InternalServerError(c, "failed to get application statistics: "+err.Error())
		return
	}

	response.Success(c, stats)
}

// GetApplication 获取应用详情
// @Summary 获取应用详情
// @Description 获取指定应用的详细信息
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "应用ID"
// @Success 200 {object} map[string]interface{} "应用详情"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 404 {object} map[string]interface{} "应用不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications/{id} [get]
func GetApplication(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	app, err := service.NewApplicationService().GetApplicationByID(uint(id))
	if err != nil {
		if err == service.ErrApplicationNotFound {
			response.NotFound(c, "application not found")
			return
		}
		response.InternalServerError(c, "failed to get application: "+err.Error())
		return
	}

	response.Success(c, service.ToApplicationResponse(app))
}

// AdvancedSearchApplications 高级搜索应用
func AdvancedSearchApplications(c *gin.Context) {
	var query service.AdvancedSearchQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		response.BadRequest(c, "无效的查询参数")
		return
	}

	searchService := service.NewAdvancedSearchService()
	result, err := searchService.SearchApplications(query)
	if err != nil {
		response.InternalServerError(c, "搜索失败")
		return
	}

	response.Success(c, result)
}

// SaveApplicationSearch 保存应用搜索
// @Summary 保存应用搜索
// @Description 保存当前的搜索条件以便后续使用
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body SaveSearchRequest true "保存搜索请求"
// @Success 200 {object} map[string]interface{} "保存结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications/save-search [post]
func SaveApplicationSearch(c *gin.Context) {
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
	savedSearch, err := searchService.SaveSearch(req.Name, "applications", req.Query, req.Description, createdBy)
	if err != nil {
		response.InternalServerError(c, "保存搜索失败")
		return
	}

	response.Success(c, savedSearch)
}

// GetSavedApplicationSearches 获取保存的应用搜索
func GetSavedApplicationSearches(c *gin.Context) {
	adminID, _ := c.Get("admin_id")
	var createdBy uint
	if id, ok := adminID.(uint); ok {
		createdBy = id
	}

	searchService := service.NewAdvancedSearchService()
	searches, err := searchService.GetSavedSearches("applications", createdBy)
	if err != nil {
		response.InternalServerError(c, "获取保存的搜索失败")
		return
	}

	response.Success(c, searches)
}

// DeleteSavedApplicationSearch 删除保存的应用搜索
// @Summary 删除保存的应用搜索
// @Description 删除指定保存的搜索条件
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "搜索ID"
// @Success 200 {object} map[string]interface{} "删除结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications/saved-searches/{id} [delete]
func DeleteSavedApplicationSearch(c *gin.Context) {
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

// ExportApplication 导出应用配置
// @Summary 导出应用配置
// @Description 导出指定应用的配置信息为JSON格式
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "应用ID"
// @Success 200 {object} map[string]interface{} "应用配置JSON"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 404 {object} map[string]interface{} "应用不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications/{id}/export [get]
func ExportApplication(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的应用ID")
		return
	}

	app, config, err := service.NewApplicationService().ExportApplicationConfig(uint(id))
	if err != nil {
		if err == service.ErrApplicationNotFound {
			response.NotFound(c, "应用不存在")
			return
		}
		response.InternalServerError(c, "导出应用配置失败: "+err.Error())
		return
	}

	exportData := gin.H{
		"application": service.ToApplicationResponse(app),
		"config":      config,
		"export_time": time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
	}

	response.Success(c, exportData)
}

// ImportApplication 导入应用配置
// @Summary 导入应用配置
// @Description 从JSON配置导入创建新应用
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body ImportApplicationRequest true "导入应用请求"
// @Success 200 {object} map[string]interface{} "导入结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications/import [post]
func ImportApplication(c *gin.Context) {
	var req ImportApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if req.Name == "" {
		response.BadRequest(c, "应用名称不能为空")
		return
	}

	app, err := service.NewApplicationService().ImportApplication(&service.ImportApplicationInput{
		Name:        req.Name,
		Description: req.Description,
		Domain:      req.Domain,
		Website:     req.Website,
		Config:      req.Config,
		UserID:      req.UserID,
	})
	if err != nil {
		response.InternalServerError(c, "导入应用失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":     "应用导入成功",
		"application": service.ToApplicationResponse(app),
	})
}

// CloneApplication 克隆应用
// @Summary 克隆应用
// @Description 克隆指定应用创建一个新应用
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "源应用ID"
// @Param body body CloneApplicationRequest true "克隆请求"
// @Success 200 {object} map[string]interface{} "克隆结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 404 {object} map[string]interface{} "应用不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications/{id}/clone [post]
func CloneApplication(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的应用ID")
		return
	}

	var req CloneApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if req.NewName == "" {
		response.BadRequest(c, "新应用名称不能为空")
		return
	}

	newApp, err := service.NewApplicationService().CloneApplication(uint(id), req.NewName)
	if err != nil {
		if err == service.ErrApplicationNotFound {
			response.NotFound(c, "源应用不存在")
			return
		}
		response.InternalServerError(c, "克隆应用失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":     "应用克隆成功",
		"original_id": id,
		"new_app":     service.ToApplicationResponse(newApp),
	})
}

// BatchDeleteApplications 批量删除应用
// @Summary 批量删除应用
// @Description 批量删除指定的应用
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body BatchDeleteRequest true "批量删除请求"
// @Success 200 {object} map[string]interface{} "删除结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications/batch-delete [post]
func BatchDeleteApplications(c *gin.Context) {
	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if len(req.IDs) == 0 {
		response.BadRequest(c, "请选择要删除的应用")
		return
	}

	if len(req.IDs) > 100 {
		response.BadRequest(c, "单次最多删除100个应用")
		return
	}

	result, err := service.NewApplicationService().BatchDeleteApplications(req.IDs)
	if err != nil {
		response.InternalServerError(c, "批量删除失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":       "批量删除完成",
		"total":         len(req.IDs),
		"deleted":       result.Deleted,
		"not_found":     result.NotFound,
		"failed_ids":    result.FailedIDs,
	})
}

// BatchUpdateApplications 批量更新应用
// @Summary 批量更新应用
// @Description 批量更新应用的配置信息
// @Tags 应用管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body BatchUpdateRequest true "批量更新请求"
// @Success 200 {object} map[string]interface{} "更新结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/applications/batch-update [post]
func BatchUpdateApplications(c *gin.Context) {
	var req BatchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if len(req.IDs) == 0 {
		response.BadRequest(c, "请选择要更新的应用")
		return
	}

	if len(req.IDs) > 100 {
		response.BadRequest(c, "单次最多更新100个应用")
		return
	}

	result, err := service.NewApplicationService().BatchUpdateApplications(req.IDs, &service.BatchUpdateInput{
		IsActive: req.IsActive,
		Config:   req.Config,
	})
	if err != nil {
		response.InternalServerError(c, "批量更新失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":     "批量更新完成",
		"total":       len(req.IDs),
		"updated":     result.Updated,
		"not_found":   result.NotFound,
		"failed_ids":  result.FailedIDs,
	})
}
