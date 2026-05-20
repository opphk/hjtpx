package handler

import (
	"errors"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
"github.com/hjtpx/hjtpx/internal/api/middleware"
"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type GDPRHandler struct {
	gdprService *service.GDPRService
}

func NewGDPRHandler() *GDPRHandler {
	return &GDPRHandler{
		gdprService: service.NewGDPRService(),
	}
}

var gdprHandler *GDPRHandler

func GetGDPRHandler() *GDPRHandler {
	if gdprHandler == nil {
		gdprHandler = NewGDPRHandler()
	}
	return gdprHandler
}

// UpdateConsentRequest 更新同意设置的请求结构
type UpdateConsentRequest struct {
	ConsentMarketing       bool `json:"consent_marketing"`
	ConsentAnalytics       bool `json:"consent_analytics"`
	ConsentPersonalization bool `json:"consent_personalization"`
	ConsentDataSharing     bool `json:"consent_data_sharing"`
}

// DataExportRequest 数据导出请求结构
type DataExportRequest struct {
	Format string `json:"format" binding:"required,oneof=json csv"` // 必须是json或csv
}

// DataDeletionRequest 数据删除请求结构
type DataDeletionRequest struct {
	Reason string `json:"reason"`
}

// GetConsent 获取用户同意设置
// @Summary 获取用户同意设置
// @Description 获取当前用户的GDPR同意设置
// @Tags GDPR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.Response "获取成功"
// @Failure 401 {object} response.Response "未授权"
// @Router /api/v1/gdpr/consent [get]
func (h *GDPRHandler) GetConsent(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c)
		return
	}

	consent, err := h.gdprService.GetConsent(userID)
	if err != nil {
		response.InternalServerError(c, "获取同意设置失败")
		return
	}

	response.Success(c, consent)
}

// UpdateConsent 更新用户同意设置
// @Summary 更新用户同意设置
// @Description 更新当前用户的GDPR同意设置
// @Tags GDPR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param consent body UpdateConsentRequest true "同意设置"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Router /api/v1/gdpr/consent [put]
func (h *GDPRHandler) UpdateConsent(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c)
		return
	}

	var req UpdateConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	consent := &models.UserConsent{
		ConsentMarketing:       req.ConsentMarketing,
		ConsentAnalytics:       req.ConsentAnalytics,
		ConsentPersonalization: req.ConsentPersonalization,
		ConsentDataSharing:     req.ConsentDataSharing,
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	updatedConsent, err := h.gdprService.UpdateConsent(userID, consent, clientIP, userAgent)
	if err != nil {
		response.InternalServerError(c, "更新同意设置失败")
		return
	}

	response.Success(c, updatedConsent)
}

// RequestDataExport 请求数据导出
// @Summary 请求数据导出
// @Description 请求导出当前用户的数据
// @Tags GDPR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body DataExportRequest true "导出请求"
// @Success 200 {object} response.Response "请求成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Router /api/v1/gdpr/data-export [post]
func (h *GDPRHandler) RequestDataExport(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c)
		return
	}

	var req DataExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	exportRequest, err := h.gdprService.RequestDataExport(userID, req.Format)
	if err != nil {
		if errors.Is(err, service.ErrInvalidExportFormat) {
			response.BadRequest(c, "无效的导出格式，仅支持json和csv")
			return
		}
		if errors.Is(err, service.ErrExportProcessing) {
			response.BadRequest(c, "已有导出请求正在处理中")
			return
		}
		response.InternalServerError(c, "创建导出请求失败")
		return
	}

	response.Success(c, exportRequest)
}

// GetExportStatus 获取导出状态
func (h *GDPRHandler) GetExportStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c)
		return
	}

	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseUint(requestIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的请求ID")
		return
	}

	exportRequest, err := h.gdprService.GetExportRequest(uint(requestID))
	if err != nil {
		if errors.Is(err, service.ErrExportRequestNotFound) {
			response.NotFound(c, "导出请求未找到")
			return
		}
		response.InternalServerError(c, "获取导出状态失败")
		return
	}

	// 确保请求属于当前用户
	if exportRequest.UserID != userID {
		response.Forbidden(c)
		return
	}

	response.Success(c, exportRequest)
}

// DownloadExport 下载导出文件
func (h *GDPRHandler) DownloadExport(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c)
		return
	}

	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseUint(requestIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的请求ID")
		return
	}

	exportRequest, err := h.gdprService.GetExportRequest(uint(requestID))
	if err != nil {
		if errors.Is(err, service.ErrExportRequestNotFound) {
			response.NotFound(c, "导出请求未找到")
			return
		}
		response.InternalServerError(c, "获取导出状态失败")
		return
	}

	// 确保请求属于当前用户
	if exportRequest.UserID != userID {
		response.Forbidden(c)
		return
	}

	if exportRequest.Status != "completed" {
		response.BadRequest(c, "导出尚未完成")
		return
	}

	if _, err := os.Stat(exportRequest.FilePath); os.IsNotExist(err) {
		response.NotFound(c, "导出文件不存在")
		return
	}

	c.FileAttachment(exportRequest.FilePath, "user_data."+exportRequest.ExportFormat)
}

// RequestDataDeletion 请求数据删除
// @Summary 请求数据删除
// @Description 请求删除当前用户的数据
// @Tags GDPR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body DataDeletionRequest true "删除请求"
// @Success 200 {object} response.Response "请求成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Router /api/v1/gdpr/data-deletion [post]
func (h *GDPRHandler) RequestDataDeletion(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c)
		return
	}

	var req DataDeletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	deletionRequest, err := h.gdprService.RequestDataDeletion(userID, req.Reason)
	if err != nil {
		if errors.Is(err, service.ErrDeletionProcessing) {
			response.BadRequest(c, "已有删除请求正在处理中")
			return
		}
		response.InternalServerError(c, "创建删除请求失败")
		return
	}

	response.Success(c, deletionRequest)
}

// GetDeletionStatus 获取删除请求状态
func (h *GDPRHandler) GetDeletionStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		response.Unauthorized(c)
		return
	}

	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseUint(requestIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的请求ID")
		return
	}

	deletionRequest, err := h.gdprService.GetDeletionRequest(uint(requestID))
	if err != nil {
		if errors.Is(err, service.ErrDeletionRequestNotFound) {
			response.NotFound(c, "删除请求未找到")
			return
		}
		response.InternalServerError(c, "获取删除状态失败")
		return
	}

	// 确保请求属于当前用户
	if deletionRequest.UserID != userID {
		response.Forbidden(c)
		return
	}

	response.Success(c, deletionRequest)
}
