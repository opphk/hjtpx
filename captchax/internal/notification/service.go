package notification

import (
	"context"
	"fmt"

	"captchax/internal/model"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateNotification(ctx context.Context, userID uint, title, content, notifType string) (*model.Notification, error) {
	if notifType == "" {
		notifType = "info"
	}

	validTypes := map[string]bool{
		"info":    true,
		"success": true,
		"warning": true,
		"error":   true,
	}
	if !validTypes[notifType] {
		return nil, fmt.Errorf("invalid notification type: %s", notifType)
	}

	notif := &model.Notification{
		UserID:  userID,
		Title:   title,
		Content: content,
		Type:    notifType,
		IsRead:  false,
	}

	if err := s.db.WithContext(ctx).Create(notif).Error; err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	return notif, nil
}

func (s *Service) GetNotifications(ctx context.Context, userID uint, page, pageSize int) ([]model.Notification, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var notifications []model.Notification
	var total int64

	query := s.db.WithContext(ctx).Model(&model.Notification{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&notifications).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list notifications: %w", err)
	}

	return notifications, total, nil
}

func (s *Service) MarkAsRead(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Model(&model.Notification{}).Where("id = ?", id).Update("is_read", true)
	if result.Error != nil {
		return fmt.Errorf("failed to mark notification as read: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}
	return nil
}

func (s *Service) MarkAllAsRead(ctx context.Context, userID uint) error {
	result := s.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true)
	if result.Error != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", result.Error)
	}
	return nil
}

func (s *Service) DeleteNotification(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&model.Notification{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete notification: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}
	return nil
}

func (s *Service) GetUnreadCount(ctx context.Context, userID uint) (int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}
	return count, nil
}