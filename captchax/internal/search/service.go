package search

import (
	"context"
	"fmt"
	"strings"

	"captchax/internal/model"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type UserSearchResult struct {
	Users    []UserItem `json:"users"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

type UserItem struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}

type CaptchaLogSearchResult struct {
	Logs     []CaptchaLogItem `json:"logs"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type CaptchaLogItem struct {
	ID        int64  `json:"id"`
	Type      string `json:"captcha_type"`
	ClientID  string `json:"client_id"`
	IP        string `json:"ip"`
	Result    bool   `json:"result"`
	RiskScore int    `json:"risk_score"`
	CreatedAt string `json:"created_at"`
}

func (s *Service) SearchUsers(ctx context.Context, keyword string, page, pageSize int) (*UserSearchResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	if strings.TrimSpace(keyword) == "" {
		return &UserSearchResult{Users: []UserItem{}, Page: page, PageSize: pageSize}, nil
	}

	pattern := "%" + keyword + "%"
	var admins []model.Admin
	var total int64

	query := s.db.WithContext(ctx).Model(&model.Admin{}).Where(
		"username ILIKE ? OR email ILIKE ? OR nickname ILIKE ?",
		pattern, pattern, pattern,
	)

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := query.Order("id ASC").Offset(offset).Limit(pageSize).Find(&admins).Error; err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	items := make([]UserItem, 0, len(admins))
	for _, a := range admins {
		items = append(items, UserItem{
			ID:       a.ID,
			Username: a.Username,
			Email:    a.Email,
			Nickname: a.Nickname,
		})
	}

	return &UserSearchResult{
		Users:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) SearchCaptchaLogs(ctx context.Context, keyword string, page, pageSize int) (*CaptchaLogSearchResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	if strings.TrimSpace(keyword) == "" {
		return &CaptchaLogSearchResult{Logs: []CaptchaLogItem{}, Page: page, PageSize: pageSize}, nil
	}

	pattern := "%" + keyword + "%"
	var logs []model.CaptchaLog
	var total int64

	query := s.db.WithContext(ctx).Model(&model.CaptchaLog{}).Where(
		"client_id ILIKE ? OR ip ILIKE ? OR captcha_type ILIKE ?",
		pattern, pattern, pattern,
	)

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count captcha logs: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to search captcha logs: %w", err)
	}

	items := make([]CaptchaLogItem, 0, len(logs))
	for _, l := range logs {
		items = append(items, CaptchaLogItem{
			ID:        l.ID,
			Type:      l.Type,
			ClientID:  l.ClientID,
			IP:        l.IP,
			Result:    l.Result,
			RiskScore: l.RiskScore,
			CreatedAt: l.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &CaptchaLogSearchResult{
		Logs:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

type GlobalSearchResult struct {
	Users       []UserItem        `json:"users,omitempty"`
	CaptchaLogs []CaptchaLogItem  `json:"captcha_logs,omitempty"`
}

func (s *Service) GlobalSearch(ctx context.Context, keyword, searchType string, page, pageSize int) (interface{}, error) {
	switch searchType {
	case "users":
		return s.SearchUsers(ctx, keyword, page, pageSize)
	case "captcha_logs", "logs":
		return s.SearchCaptchaLogs(ctx, keyword, page, pageSize)
	case "":
		users, err := s.SearchUsers(ctx, keyword, 1, 10)
		if err != nil {
			return nil, err
		}
		logs, err := s.SearchCaptchaLogs(ctx, keyword, 1, 10)
		if err != nil {
			return nil, err
		}
		return &GlobalSearchResult{
			Users:       users.Users,
			CaptchaLogs: logs.Logs,
		}, nil
	default:
		return s.SearchUsers(ctx, keyword, page, pageSize)
	}
}