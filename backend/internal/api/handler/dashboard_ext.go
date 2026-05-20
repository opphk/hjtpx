package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var dashboardExtService = service.NewDashboardExtService()
var notificationService = service.NewNotificationService()

// GetDashboardConfig 获取仪表盘配置
func GetDashboardConfig(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	if adminID == 0 {
		adminID = 1 // 默认管理员ID
	}
	
	config, err := dashboardExtService.GetDashboardConfig(adminID)
	if err != nil {
		response.InternalServerError(c, "获取仪表盘配置失败")
		return
	}
	
	response.Success(c, config)
}

// SaveDashboardConfig 保存仪表盘配置
func SaveDashboardConfig(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	if adminID == 0 {
		adminID = 1
	}
	
	var config models.DashboardConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}
	
	result, err := dashboardExtService.SaveDashboardConfig(adminID, &config)
	if err != nil {
		response.InternalServerError(c, "保存仪表盘配置失败")
		return
	}
	
	response.Success(c, result)
}

// UpdateDashboardTheme 更新主题
func UpdateDashboardTheme(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	if adminID == 0 {
		adminID = 1
	}
	
	var req struct {
		Theme string `json:"theme" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}
	
	if err := dashboardExtService.UpdateDashboardTheme(adminID, req.Theme); err != nil {
		response.InternalServerError(c, "更新主题失败")
		return
	}
	
	response.Success(c, nil)
}

// GetDashboardWidgets 获取仪表盘组件
func GetDashboardWidgets(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	if adminID == 0 {
		adminID = 1
	}
	
	widgets, err := dashboardExtService.GetDashboardWidgets(adminID)
	if err != nil {
		response.InternalServerError(c, "获取仪表盘组件失败")
		return
	}
	
	response.Success(c, widgets)
}

// SaveDashboardWidgets 保存仪表盘组件
func SaveDashboardWidgets(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	if adminID == 0 {
		adminID = 1
	}
	
	var widgets []models.DashboardWidget
	if err := c.ShouldBindJSON(&widgets); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}
	
	if err := dashboardExtService.SaveDashboardWidgets(adminID, widgets); err != nil {
		response.InternalServerError(c, "保存仪表盘组件失败")
		return
	}
	
	response.Success(c, nil)
}

// ==================== 通知相关 ====================

// GetNotifications 获取通知列表
func GetNotifications(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	if adminID == 0 {
		adminID = 1
	}
	
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	onlyUnread := c.Query("only_unread") == "true"
	
	notifications, total, err := notificationService.GetNotifications(adminID, page, pageSize, onlyUnread)
	if err != nil {
		response.InternalServerError(c, "获取通知列表失败")
		return
	}
	
	response.Success(c, gin.H{
		"items":      notifications,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// MarkNotificationRead 标记通知为已读
func MarkNotificationRead(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	if adminID == 0 {
		adminID = 1
	}
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的通知ID")
		return
	}
	
	if err := notificationService.MarkAsRead(uint(id), adminID); err != nil {
		response.InternalServerError(c, "标记已读失败")
		return
	}
	
	response.Success(c, nil)
}

// MarkAllNotificationsRead 标记所有通知为已读
func MarkAllNotificationsRead(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	if adminID == 0 {
		adminID = 1
	}
	
	if err := notificationService.MarkAllAsRead(adminID); err != nil {
		response.InternalServerError(c, "标记失败")
		return
	}
	
	response.Success(c, nil)
}

// DeleteNotification 删除通知
func DeleteNotification(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	if adminID == 0 {
		adminID = 1
	}
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的通知ID")
		return
	}
	
	if err := notificationService.DeleteNotification(uint(id), adminID); err != nil {
		response.InternalServerError(c, "删除通知失败")
		return
	}
	
	response.Success(c, nil)
}

// GetUnreadNotificationCount 获取未读通知数量
func GetUnreadNotificationCount(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	if adminID == 0 {
		adminID = 1
	}
	
	count, err := notificationService.GetUnreadCount(adminID)
	if err != nil {
		response.InternalServerError(c, "获取未读数量失败")
		return
	}
	
	response.Success(c, gin.H{"count": count})
}

// BroadcastNotification 广播通知
func BroadcastNotification(c *gin.Context) {
	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Type    string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}
	
	if req.Type == "" {
		req.Type = "info"
	}
	
	if err := notificationService.BroadcastNotification(req.Title, req.Content, req.Type); err != nil {
		response.InternalServerError(c, "广播通知失败")
		return
	}
	
	response.Success(c, nil)
}
