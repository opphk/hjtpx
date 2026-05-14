package captcha

import (
	"context"
	"encoding/json"
	"time"

	"github.com/opphk/captcha-system/internal/model"
	"github.com/opphk/captcha-system/internal/repository"
)

type AdminService struct {
	adminRepo *repository.AdminRepository
	statsRepo *repository.StatsRepository
}

func NewAdminService(adminRepo *repository.AdminRepository, statsRepo *repository.StatsRepository) *AdminService {
	return &AdminService{
		adminRepo: adminRepo,
		statsRepo: statsRepo,
	}
}

func (s *AdminService) GetUserByUsername(ctx context.Context, username string) (*model.AdminUser, error) {
	return s.adminRepo.GetByUsername(ctx, username)
}

func (s *AdminService) UpdateLastLogin(ctx context.Context, userID int64) error {
	return s.adminRepo.UpdateLastLogin(ctx, userID)
}

func (s *AdminService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	today := time.Now().Truncate(24 * time.Hour)

	todayStats, err := s.statsRepo.GetByDate(ctx, today)
	if err != nil {
		return nil, err
	}

	weekAgo := today.AddDate(0, 0, -7)
	weekStats, err := s.statsRepo.GetRange(ctx, weekAgo, today)
	if err != nil {
		return nil, err
	}

	totalStats, err := s.statsRepo.GetTotal(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"today":        todayStats,
		"week":         weekStats,
		"total":        totalStats,
		"generated_at": time.Now(),
	}, nil
}

func (s *AdminService) GetChallenges(ctx context.Context, page, size int, challengeType string) ([]*model.Challenge, int64, error) {
	return s.adminRepo.GetChallenges(ctx, page, size, challengeType)
}

func (s *AdminService) GetAttempts(ctx context.Context, page, size int) ([]*model.Attempt, int64, error) {
	return s.adminRepo.GetAttempts(ctx, page, size)
}

func (s *AdminService) UpdateConfig(ctx context.Context, key string, value interface{}) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.adminRepo.UpdateConfig(ctx, key, valueJSON)
}

func (s *AdminService) GetLogs(ctx context.Context, level string, page, size int) ([]*model.Log, int64, error) {
	return s.adminRepo.GetLogs(ctx, level, page, size)
}
