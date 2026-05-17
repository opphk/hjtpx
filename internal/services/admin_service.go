package services

import (
	"errors"
	"time"

	"hjtpx/internal/models"
	"hjtpx/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrAppNotFound        = errors.New("app not found")
	ErrAppExists          = errors.New("app already exists")
)

type AdminService struct {
	userRepo            *repository.UserRepository
	appRepo             *repository.AppRepository
	captchaRepo         *repository.CaptchaRepository
	verificationLogRepo *repository.VerificationLogRepository
}

func NewAdminService(
	userRepo *repository.UserRepository,
	appRepo *repository.AppRepository,
	captchaRepo *repository.CaptchaRepository,
	verificationLogRepo *repository.VerificationLogRepository,
) *AdminService {
	return &AdminService{
		userRepo:            userRepo,
		appRepo:             appRepo,
		captchaRepo:         captchaRepo,
		verificationLogRepo: verificationLogRepo,
	}
}

type DashboardStats struct {
	TotalUsers       int64            `json:"total_users"`
	TotalApps        int64            `json:"total_apps"`
	TotalCaptchas    int64            `json:"total_captchas"`
	TotalVerifications int64         `json:"total_verifications"`
	PendingCaptchas  int64            `json:"pending_captchas"`
	VerifiedCaptchas  int64            `json:"verified_captchas"`
	ExpiredCaptchas  int64            `json:"expired_captchas"`
	FailedCaptchas   int64            `json:"failed_captchas"`
	SuccessRate      float64          `json:"success_rate"`
	RecentUsers      []models.User    `json:"recent_users"`
	RecentCaptchas   []models.Captcha `json:"recent_captchas"`
}

func (s *AdminService) GetDashboardStats() (*DashboardStats, error) {
	stats := &DashboardStats{}

	userCount, err := s.userRepo.Count()
	if err != nil {
		return nil, err
	}
	stats.TotalUsers = userCount

	appCount, err := s.appRepo.Count()
	if err != nil {
		return nil, err
	}
	stats.TotalApps = appCount

	captchaStats, err := s.captchaRepo.GetStats()
	if err != nil {
		return nil, err
	}

	if total, ok := captchaStats["total"].(int64); ok {
		stats.TotalCaptchas = total
	}
	if pending, ok := captchaStats["pending"].(int64); ok {
		stats.PendingCaptchas = pending
	}
	if verified, ok := captchaStats["verified"].(int64); ok {
		stats.VerifiedCaptchas = verified
	}
	if expired, ok := captchaStats["expired"].(int64); ok {
		stats.ExpiredCaptchas = expired
	}
	if failed, ok := captchaStats["failed"].(int64); ok {
		stats.FailedCaptchas = failed
	}

	if stats.TotalCaptchas > 0 {
		stats.SuccessRate = float64(stats.VerifiedCaptchas) / float64(stats.TotalCaptchas) * 100
	}

	users, _, err := s.userRepo.FindAll(1, 5)
	if err == nil {
		stats.RecentUsers = users
	}

	captchas, _, err := s.captchaRepo.FindAll(1, 5, "")
	if err == nil {
		stats.RecentCaptchas = captchas
	}

	return stats, nil
}

func (s *AdminService) CreateApp(req *models.CreateAppRequest) (*models.App, error) {
	existingApp, err := s.appRepo.GetByAppKey(req.AppKey)
	if err == nil && existingApp != nil {
		return nil, ErrAppExists
	}

	app := &models.App{
		Name:      req.Name,
		AppKey:    req.AppKey,
		AppSecret: req.AppSecret,
		Domain:    req.Domain,
		OwnerID:   req.OwnerID,
		Status:    1,
	}

	if err := s.appRepo.Create(app); err != nil {
		return nil, err
	}

	return app, nil
}

func (s *AdminService) UpdateApp(appID uint, req *models.UpdateAppRequest) (*models.App, error) {
	app, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, ErrAppNotFound
	}

	if req.Name != "" {
		app.Name = req.Name
	}
	if req.AppSecret != "" {
		app.AppSecret = req.AppSecret
	}
	if req.Status != 0 {
		app.Status = req.Status
	}
	if req.Domain != "" {
		app.Domain = req.Domain
	}

	if err := s.appRepo.Update(app); err != nil {
		return nil, err
	}

	return app, nil
}

func (s *AdminService) DeleteApp(appID uint) error {
	return s.appRepo.Delete(appID)
}

func (s *AdminService) ListApps(page, pageSize int) ([]models.App, int64, error) {
	return s.appRepo.FindAll(page, pageSize)
}

func (s *AdminService) GetApp(appID uint) (*models.App, error) {
	app, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, ErrAppNotFound
	}
	return app, nil
}

func (s *AdminService) CreateUser(req *models.CreateUserRequest) (*models.User, error) {
	existingUser, err := s.userRepo.FindByEmail(req.Email)
	if err == nil && existingUser != nil {
		return nil, ErrUserNotFound
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:    req.Email,
		Password: string(hashedPassword),
		Username: req.Username,
		AppID:    req.AppID,
		Status:   1,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AdminService) UpdateUser(userID uint, req *models.UpdateUserRequest) (*models.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Status != 0 {
		user.Status = req.Status
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AdminService) DeleteUser(userID uint) error {
	return s.userRepo.Delete(userID)
}

func (s *AdminService) ListUsers(page, pageSize int) ([]models.User, int64, error) {
	return s.userRepo.FindAll(page, pageSize)
}

func (s *AdminService) GetUser(userID uint) (*models.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *AdminService) AdminLogin(email, password string) (*models.User, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	user.LastLogin = time.Now()
	s.userRepo.Update(user)

	return user, nil
}

func (s *AdminService) GetCaptchaStats() (map[string]interface{}, error) {
	return s.captchaRepo.GetStats()
}

func (s *AdminService) ListCaptchas(page, pageSize int, status string) ([]models.Captcha, int64, error) {
	return s.captchaRepo.FindAll(page, pageSize, status)
}
