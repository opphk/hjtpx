package handler

import (
	"strconv"

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

type CreateApplicationRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	UserID      uint   `json:"user_id" binding:"required"`
	Description string `json:"description" binding:"max=1000"`
	Domain      string `json:"domain" binding:"max=255"`
	Website     string `json:"website" binding:"max=255"`
}

type UpdateApplicationRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
	IsActive    *bool   `json:"is_active"`
	Domain      *string `json:"domain" binding:"omitempty,max=255"`
	Website     *string `json:"website" binding:"omitempty,max=255"`
}

type ListApplicationsQuery struct {
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=10"`
	Keyword   string `form:"keyword"`
	UserID    uint   `form:"user_id"`
	IsActive  *bool  `form:"is_active"`
	SortField string `form:"sort_field"`
	SortOrder string `form:"sort_order"`
}

type UpdateConfigRequest struct {
	CaptchaTypes         []string               `json:"captcha_types"`
	MaxVerifyPerMinute   int                    `json:"max_verify_per_minute"`
	MaxVerifyPerDay      int                    `json:"max_verify_per_day"`
	AllowedIPs           []string               `json:"allowed_ips"`
	BlockRefusedRequests bool                   `json:"block_refused_requests"`
	CustomSettings       map[string]interface{} `json:"custom_settings"`
}

func GetApplicationHandler() *ApplicationHandler {
	return NewApplicationHandler()
}

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
func SaveApplicationSearch(c *gin.Context) {
	var req struct {
		Name        string                      `json:"name" binding:"required"`
		Description string                      `json:"description"`
		Query       service.AdvancedSearchQuery `json:"query" binding:"required"`
	}
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
