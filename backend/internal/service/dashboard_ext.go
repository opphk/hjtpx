package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

// DashboardExtService 仪表盘扩展服务
type DashboardExtService struct {
	db *gorm.DB
}

// NewDashboardExtService 创建新的仪表盘扩展服务
func NewDashboardExtService() *DashboardExtService {
	return &DashboardExtService{
		db: database.DB,
	}
}

// GetDashboardConfig 获取管理员的仪表盘配置
func (s *DashboardExtService) GetDashboardConfig(adminID uint) (*models.DashboardConfig, error) {
	var config models.DashboardConfig
	err := s.db.Where("admin_id = ? AND is_active = ?", adminID, true).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建默认配置
			return s.createDefaultDashboardConfig(adminID)
		}
		return nil, err
	}
	return &config, nil
}

// createDefaultDashboardConfig 创建默认的仪表盘配置
func (s *DashboardExtService) createDefaultDashboardConfig(adminID uint) (*models.DashboardConfig, error) {
	defaultWidgets := []map[string]interface{}{
		{
			"widget_type": "stat",
			"title":       "今日验证",
			"position_x":  0,
			"position_y":  0,
			"width":        3,
			"height":       1,
		},
		{
			"widget_type": "stat",
			"title":       "通过率",
			"position_x":  3,
			"position_y":  0,
			"width":        3,
			"height":       1,
		},
		{
			"widget_type": "stat",
			"title":       "拦截率",
			"position_x":  6,
			"position_y":  0,
			"width":        3,
			"height":       1,
		},
		{
			"widget_type": "stat",
			"title":       "平均响应时间",
			"position_x":  9,
			"position_y":  0,
			"width":        3,
			"height":       1,
		},
		{
			"widget_type": "chart",
			"title":       "24小时趋势",
			"position_x":  0,
			"position_y":  1,
			"width":        8,
			"height":       3,
		},
		{
			"widget_type": "list",
			"title":       "最近验证记录",
			"position_x":  8,
			"position_y":  1,
			"width":        4,
			"height":       3,
		},
	}

	widgetsJSON, _ := json.Marshal(defaultWidgets)

	config := models.DashboardConfig{
		AdminID:      adminID,
		LayoutConfig: string(widgetsJSON),
		Theme:        "default",
		IsActive:     true,
	}

	if err := s.db.Create(&config).Error; err != nil {
		return nil, err
	}

	// 创建默认组件
	for _, widgetData := range defaultWidgets {
		widgetJSON, _ := json.Marshal(widgetData)
		widget := models.DashboardWidget{
			DashboardConfigID: config.ID,
			WidgetType:        widgetData["widget_type"].(string),
			Title:             widgetData["title"].(string),
			PositionX:         widgetData["position_x"].(int),
			PositionY:         widgetData["position_y"].(int),
			Width:             widgetData["width"].(int),
			Height:            widgetData["height"].(int),
			Config:            string(widgetJSON),
			IsVisible:         true,
		}
		if err := s.db.Create(&widget).Error; err != nil {
			return nil, err
		}
	}

	return &config, nil
}

// SaveDashboardConfig 保存仪表盘配置
func (s *DashboardExtService) SaveDashboardConfig(adminID uint, config *models.DashboardConfig) (*models.DashboardConfig, error) {
	existingConfig, err := s.GetDashboardConfig(adminID)
	if err != nil {
		return nil, err
	}

	existingConfig.LayoutConfig = config.LayoutConfig
	existingConfig.Theme = config.Theme
	if err := s.db.Save(existingConfig).Error; err != nil {
		return nil, err
	}

	return existingConfig, nil
}

// UpdateDashboardTheme 更新主题
func (s *DashboardExtService) UpdateDashboardTheme(adminID uint, theme string) error {
	config, err := s.GetDashboardConfig(adminID)
	if err != nil {
		return err
	}
	config.Theme = theme
	return s.db.Save(config).Error
}

// GetDashboardWidgets 获取仪表盘组件
func (s *DashboardExtService) GetDashboardWidgets(adminID uint) ([]models.DashboardWidget, error) {
	config, err := s.GetDashboardConfig(adminID)
	if err != nil {
		return nil, err
	}

	var widgets []models.DashboardWidget
	err = s.db.Where("dashboard_config_id = ?", config.ID).Order("position_y, position_x").Find(&widgets).Error
	return widgets, err
}

// SaveDashboardWidgets 保存仪表盘组件
func (s *DashboardExtService) SaveDashboardWidgets(adminID uint, widgets []models.DashboardWidget) error {
	config, err := s.GetDashboardConfig(adminID)
	if err != nil {
		return err
	}

	// 删除旧组件
	if err := s.db.Where("dashboard_config_id = ?", config.ID).Delete(&models.DashboardWidget{}).Error; err != nil {
		return err
	}

	// 创建新组件
	for _, widget := range widgets {
		widget.DashboardConfigID = config.ID
		if err := s.db.Create(&widget).Error; err != nil {
			return err
		}
	}

	return nil
}

// NotificationService 通知服务
type NotificationService struct {
	db *gorm.DB
}

// NewNotificationService 创建新的通知服务
func NewNotificationService() *NotificationService {
	return &NotificationService{
		db: database.DB,
	}
}

// CreateNotification 创建通知
func (s *NotificationService) CreateNotification(adminID uint, title, content, notificationType, link string, meta interface{}) (*models.Notification, error) {
	metaJSON, _ := json.Marshal(meta)
	notification := models.Notification{
		AdminID: adminID,
		Title:   title,
		Content: content,
		Type:    notificationType,
		Link:    link,
		Meta:    string(metaJSON),
		IsRead:  false,
	}
	if err := s.db.Create(&notification).Error; err != nil {
		return nil, err
	}
	return &notification, nil
}

// GetNotifications 获取管理员的通知列表
func (s *NotificationService) GetNotifications(adminID uint, page, pageSize int, onlyUnread bool) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	var total int64

	query := s.db.Model(&models.Notification{}).Where("admin_id = ?", adminID)
	if onlyUnread {
		query = query.Where("is_read = ?", false)
	}

	query.Count(&total)
	err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&notifications).Error
	return notifications, total, err
}

// MarkAsRead 标记通知为已读
func (s *NotificationService) MarkAsRead(notificationID uint, adminID uint) error {
	return s.db.Model(&models.Notification{}).Where("id = ? AND admin_id = ?", notificationID, adminID).Updates(map[string]interface{}{
		"is_read": true,
		"read_at": time.Now(),
	}).Error
}

// MarkAllAsRead 标记所有通知为已读
func (s *NotificationService) MarkAllAsRead(adminID uint) error {
	return s.db.Model(&models.Notification{}).Where("admin_id = ?", adminID).Updates(map[string]interface{}{
		"is_read": true,
		"read_at": time.Now(),
	}).Error
}

// DeleteNotification 删除通知
func (s *NotificationService) DeleteNotification(notificationID uint, adminID uint) error {
	return s.db.Where("id = ? AND admin_id = ?", notificationID, adminID).Delete(&models.Notification{}).Error
}

// GetUnreadCount 获取未读通知数量
func (s *NotificationService) GetUnreadCount(adminID uint) (int64, error) {
	var count int64
	err := s.db.Model(&models.Notification{}).Where("admin_id = ? AND is_read = ?", adminID, false).Count(&count).Error
	return count, err
}

// BroadcastNotification 广播通知给所有管理员
func (s *NotificationService) BroadcastNotification(title, content, notificationType string) error {
	var admins []models.Admin
	if err := s.db.Find(&admins).Error; err != nil {
		return err
	}

	for _, admin := range admins {
		if _, err := s.CreateNotification(admin.ID, title, content, notificationType, "", nil); err != nil {
			fmt.Printf("Failed to send notification to admin %d: %v\n", admin.ID, err)
		}
	}
	return nil
}
