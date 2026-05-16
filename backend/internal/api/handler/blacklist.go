package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
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

type ListBlacklistQuery struct {
	Page     int    `form:"page,default=1"`
	Size     int    `form:"size,default=20"`
	Type     string `form:"type"`
	Source   string `form:"source"`
	Status   string `form:"status"`
	Keyword  string `form:"keyword"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

type CreateBlacklistRequest struct {
	Type           string   `json:"type" binding:"required"`
	Target         string   `json:"target" binding:"required"`
	Reason         string   `json:"reason"`
	Action         string   `json:"action"`
	ApplicationIDs []string `json:"application_ids"`
	Expiration     string   `json:"expiration"`
	Note           string   `json:"note"`
}

type UpdateBlacklistRequest struct {
	Type           *string  `json:"type"`
	Reason         *string  `json:"reason"`
	Action         *string  `json:"action"`
	ApplicationIDs []string `json:"application_ids"`
	Expiration     *string  `json:"expiration"`
	Note           *string  `json:"note"`
}

type ImportBlacklistRequest struct {
	Type      string   `json:"type" binding:"required"`
	Targets   []string `json:"targets" binding:"required,min=1"`
	Reason    string   `json:"reason"`
	Action    string   `json:"action"`
	ExpiresAt string  `json:"expiration"`
}

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
		Source:        "manual",
		Reason:        req.Reason,
		Action:        req.Action,
		ApplicationIDs: req.ApplicationIDs,
		Expiration:    req.Expiration,
		Note:          req.Note,
		CreatedBy:     createdBy,
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
			Target:       target,
			Type:         req.Type,
			Source:      "import",
			Reason:       req.Reason,
			Action:       req.Action,
			Expiration:   req.ExpiresAt,
			CreatedBy:    createdBy,
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
