package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type BlacklistHandler struct {
	blacklistService *service.BlacklistService
}

func NewBlacklistHandler() *BlacklistHandler {
	return &BlacklistHandler{
		blacklistService: service.NewBlacklistService(),
	}
}

func GetBlacklistHandler() *BlacklistHandler {
	return NewBlacklistHandler()
}

// ListBlacklistQuery 黑名单列表查询参数
// @Description 黑名单列表查询参数
type ListBlacklistQuery struct {
	Page      int    `form:"page,default=1"`       // 页码
	Size      int    `form:"size,default=20"`      // 每页数量
	Type      string `form:"type"`                 // 类型
	Source    string `form:"source"`               // 来源
	Status    string `form:"status"`               // 状态
	Keyword   string `form:"keyword"`               // 关键词
	StartDate string `form:"start_date"`           // 开始日期
	EndDate   string `form:"end_date"`             // 结束日期
}

// CreateBlacklistRequest 创建黑名单请求
// @Description 创建黑名单请求参数
type CreateBlacklistRequest struct {
	Type           string   `json:"type" binding:"required"`            // 类型
	Target         string   `json:"target" binding:"required"`          // 目标
	Reason         string   `json:"reason"`                             // 原因
	Action         string   `json:"action"`                             // 操作
	ApplicationIDs []string `json:"application_ids"`                   // 应用ID列表
	Expiration     string   `json:"expiration"`                         // 过期时间
	Note           string   `json:"note"`                               // 备注
}

// UpdateBlacklistRequest 更新黑名单请求
// @Description 更新黑名单请求参数
type UpdateBlacklistRequest struct {
	Type           *string  `json:"type"`            // 类型
	Reason         *string  `json:"reason"`         // 原因
	Action         *string  `json:"action"`         // 操作
	ApplicationIDs []string `json:"application_ids"` // 应用ID列表
	Expiration     *string  `json:"expiration"`     // 过期时间
	Note           *string  `json:"note"`           // 备注
}

// ImportBlacklistRequest 导入黑名单请求
// @Description 批量导入黑名单请求参数
type ImportBlacklistRequest struct {
	Type      string   `json:"type" binding:"required"`       // 类型
	Targets   []string `json:"targets" binding:"required,min=1"` // 目标列表
	Reason    string   `json:"reason"`                        // 原因
	Action    string   `json:"action"`                        // 操作
	ExpiresAt string   `json:"expiration"`                    // 过期时间
}

// ListBlacklist 获取黑名单列表
// @Summary 获取黑名单列表
// @Description 分页获取黑名单记录列表
// @Tags 黑名单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，默认1"
// @Param size query int false "每页数量，默认20"
// @Param type query string false "类型：ip, user_id, device_id, email"
// @Param source query string false "来源"
// @Param status query string false "状态：active, expired, deleted"
// @Param keyword query string false "关键词搜索"
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} map[string]interface{} "黑名单列表"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/blacklist [get]
func ListBlacklist(c *gin.Context) {
	handler := GetBlacklistHandler()
	var query ListBlacklistQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, "无效的查询参数")
		return
	}

	startDate, _ := time.Parse("2006-01-02", query.StartDate)
	endDate, _ := time.Parse("2006-01-02", query.EndDate)
	if query.EndDate != "" {
		endDate = endDate.Add(24 * time.Hour)
	}

	filter := &service.ListBlacklistFilter{
		Page:      query.Page,
		PageSize:  query.Size,
		Type:      query.Type,
		Source:    query.Source,
		Status:    query.Status,
		Keyword:   query.Keyword,
		StartDate: startDate,
		EndDate:   endDate,
	}

	result, err := handler.blacklistService.ListBlacklist(filter)
	if err != nil {
		response.InternalServerError(c, "查询黑名单失败")
		return
	}

	response.Success(c, gin.H{
		"list":        result.Data,
		"total":       result.Total,
		"page":        result.Page,
		"page_size":   result.PageSize,
		"total_pages": result.TotalPages,
	})
}

func GetBlacklistSummary(c *gin.Context) {
	handler := GetBlacklistHandler()

	summary, err := handler.blacklistService.GetBlacklistSummary()
	if err != nil {
		response.InternalServerError(c, "获取黑名单统计失败")
		return
	}

	response.Success(c, summary)
}

// GetBlacklistByID 获取黑名单详情
// @Summary 获取黑名单详情
// @Description 根据ID获取黑名单详细信息
// @Tags 黑名单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "黑名单ID"
// @Success 200 {object} map[string]interface{} "黑名单详情"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 404 {object} map[string]interface{} "黑名单记录不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/blacklist/{id} [get]
func GetBlacklistByID(c *gin.Context) {
	handler := GetBlacklistHandler()
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的黑名单ID")
		return
	}

	item, err := handler.blacklistService.GetBlacklistByID(uint(id))
	if err != nil {
		if err == service.ErrBlacklistNotFound {
			response.NotFound(c, "黑名单记录不存在")
			return
		}
		response.InternalServerError(c, "获取黑名单详情失败")
		return
	}

	response.Success(c, item)
}

func CreateBlacklist(c *gin.Context) {
	handler := GetBlacklistHandler()
	var req CreateBlacklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	adminID, _ := c.Get("admin_id")
	var createdBy uint
	if id, ok := adminID.(uint); ok {
		createdBy = id
	}

	input := &service.CreateBlacklistInput{
		Target:         req.Target,
		Type:           req.Type,
		Source:         "manual",
		Reason:         req.Reason,
		Action:         req.Action,
		ApplicationIDs: req.ApplicationIDs,
		Expiration:     req.Expiration,
		Note:           req.Note,
		CreatedBy:      createdBy,
	}

	item, err := handler.blacklistService.CreateBlacklist(input)
	if err != nil {
		if err == service.ErrInvalidInput {
			response.BadRequest(c, "无效的输入参数")
			return
		}
		response.InternalServerError(c, "创建黑名单失败")
		return
	}

	response.Success(c, item)
}

// UpdateBlacklist 更新黑名单
// @Summary 更新黑名单
// @Description 更新指定黑名单记录
// @Tags 黑名单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "黑名单ID"
// @Param body body UpdateBlacklistRequest true "更新黑名单请求"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 404 {object} map[string]interface{} "黑名单记录不存在"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/blacklist/{id} [put]
func UpdateBlacklist(c *gin.Context) {
	handler := GetBlacklistHandler()
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的黑名单ID")
		return
	}

	var req UpdateBlacklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	input := &service.UpdateBlacklistInput{
		Type:           req.Type,
		Reason:         req.Reason,
		Action:         req.Action,
		ApplicationIDs: req.ApplicationIDs,
		Expiration:     req.Expiration,
		Note:           req.Note,
	}

	item, err := handler.blacklistService.UpdateBlacklist(uint(id), input)
	if err != nil {
		if err == service.ErrBlacklistNotFound {
			response.NotFound(c, "黑名单记录不存在")
			return
		}
		response.InternalServerError(c, "更新黑名单失败")
		return
	}

	response.Success(c, item)
}

func DeleteBlacklist(c *gin.Context) {
	handler := GetBlacklistHandler()
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的黑名单ID")
		return
	}

	err = handler.blacklistService.DeleteBlacklist(uint(id))
	if err != nil {
		if err == service.ErrBlacklistNotFound {
			response.NotFound(c, "黑名单记录不存在")
			return
		}
		response.InternalServerError(c, "删除黑名单失败")
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

func UnblockBlacklist(c *gin.Context) {
	handler := GetBlacklistHandler()
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的黑名单ID")
		return
	}

	item, err := handler.blacklistService.UnblockBlacklist(uint(id))
	if err != nil {
		if err == service.ErrBlacklistNotFound {
			response.NotFound(c, "黑名单记录不存在")
			return
		}
		response.InternalServerError(c, "解封失败")
		return
	}

	response.Success(c, item)
}

func ImportBlacklist(c *gin.Context) {
	handler := GetBlacklistHandler()
	var req ImportBlacklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	adminID, _ := c.Get("admin_id")
	var createdBy uint
	if id, ok := adminID.(uint); ok {
		createdBy = id
	}

	var inputs []service.CreateBlacklistInput
	for _, target := range req.Targets {
		inputs = append(inputs, service.CreateBlacklistInput{
			Target:     target,
			Type:       req.Type,
			Source:     "import",
			Reason:     req.Reason,
			Action:     req.Action,
			Expiration: req.ExpiresAt,
			CreatedBy:  createdBy,
		})
	}

	count, err := handler.blacklistService.BatchCreateBlacklist(inputs)
	if err != nil {
		response.InternalServerError(c, "导入失败")
		return
	}

	response.Success(c, gin.H{
		"imported": count,
		"total":    len(req.Targets),
	})
}

// CheckBlacklist 检查黑名单
// @Summary 检查目标是否在黑名单中
// @Description 检查指定目标是否已被加入黑名单
// @Tags 黑名单
// @Accept json
// @Produce json
// @Param target query string true "目标值"
// @Param type query string false "类型，默认ip"
// @Success 200 {object} map[string]interface{} "检查结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/blacklist/check [get]
func CheckBlacklist(c *gin.Context) {
	handler := GetBlacklistHandler()
	target := c.Query("target")
	blType := c.DefaultQuery("type", "ip")

	if target == "" {
		response.BadRequest(c, "target参数不能为空")
		return
	}

	isBlacklisted, err := handler.blacklistService.CheckBlacklist(target, blType)
	if err != nil {
		response.InternalServerError(c, "检查失败")
		return
	}

	response.Success(c, gin.H{
		"is_blacklisted": isBlacklisted,
		"target":         target,
		"type":           blType,
	})
}

// AdvancedSearchBlacklist 高级搜索黑名单
// @Summary 高级搜索黑名单
// @Description 使用高级查询条件搜索黑名单
// @Tags 黑名单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body service.AdvancedSearchQuery true "高级搜索查询"
// @Success 200 {object} map[string]interface{} "搜索结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/admin/blacklist/search [post]
func AdvancedSearchBlacklist(c *gin.Context) {
	var query service.AdvancedSearchQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		response.BadRequest(c, "无效的查询参数")
		return
	}

	searchService := service.NewAdvancedSearchService()
	result, err := searchService.SearchBlacklist(query)
	if err != nil {
		response.InternalServerError(c, "搜索失败")
		return
	}

	response.Success(c, result)
}

// SaveBlacklistSearch 保存黑名单搜索
func SaveBlacklistSearch(c *gin.Context) {
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
	savedSearch, err := searchService.SaveSearch(req.Name, "blacklist", req.Query, req.Description, createdBy)
	if err != nil {
		response.InternalServerError(c, "保存搜索失败")
		return
	}

	response.Success(c, savedSearch)
}

// GetSavedBlacklistSearches 获取保存的黑名单搜索
func GetSavedBlacklistSearches(c *gin.Context) {
	adminID, _ := c.Get("admin_id")
	var createdBy uint
	if id, ok := adminID.(uint); ok {
		createdBy = id
	}

	searchService := service.NewAdvancedSearchService()
	searches, err := searchService.GetSavedSearches("blacklist", createdBy)
	if err != nil {
		response.InternalServerError(c, "获取保存的搜索失败")
		return
	}

	response.Success(c, searches)
}

// DeleteSavedBlacklistSearch 删除保存的黑名单搜索
func DeleteSavedBlacklistSearch(c *gin.Context) {
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
