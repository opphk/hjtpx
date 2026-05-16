package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token expired")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrUserNotVerified   = errors.New("user not verified")
	ErrUserDisabled      = errors.New("user is disabled")
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

type RegisterInput struct {
	Username     string                `json:"username" binding:"required,min=3,max=50"`
	Email        string                `json:"email" binding:"required,email"`
	Password     string                `json:"password" binding:"required,min=6,max=128"`
	BehaviorData []models.BehaviorData `json:"behavior_data,omitempty"`
}

type LoginInput struct {
	Username     string                `json:"username" binding:"required"`
	Password     string                `json:"password" binding:"required"`
	CaptchaToken string                `json:"captcha_token,omitempty"`
	BehaviorData []models.BehaviorData `json:"behavior_data,omitempty"`
}

type UserResponse struct {
	ID          uint       `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Nickname    string     `json:"nickname"`
	Avatar      string     `json:"avatar"`
	Phone       string     `json:"phone"`
	Bio         string     `json:"bio"`
	IsVerified  bool       `json:"is_verified"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	LoginCount  int        `json:"login_count"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP string     `json:"last_login_ip"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
}

type UpdateProfileInput struct {
	Nickname string `json:"nickname" binding:"max=100"`
	Avatar   string `json:"avatar" binding:"max=500"`
	Phone    string `json:"phone" binding:"max=20"`
	Bio      string `json:"bio" binding:"max=500"`
}

type ChangePasswordInput struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=128"`
}

type ResetPasswordInput struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=128"`
}

func (s *UserService) Register(input *RegisterInput, riskScore float64) (*models.User, string, error) {
	if riskScore > 70 {
		return nil, "", errors.New("risk score too high, registration blocked")
	}

	var existing models.User
	if err := database.DB.Where("username = ? OR email = ?", input.Username, input.Email).First(&existing).Error; err == nil {
		return nil, "", ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	verificationToken := generateToken(32)

	user := &models.User{
		Username:          input.Username,
		Email:             input.Email,
		PasswordHash:      string(hashedPassword),
		VerificationToken: verificationToken,
		Status:            "active",
	}

	if err := database.DB.Create(user).Error; err != nil {
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	return user, verificationToken, nil
}

func (s *UserService) Login(input *LoginInput, clientIP string) (*models.User, string, error) {
	var user models.User
	if err := database.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrUserNotFound
		}
		return nil, "", err
	}

	if user.Status != "active" {
		return nil, "", ErrUserDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, "", ErrInvalidPassword
	}

	now := time.Now()
	loginCount := user.LoginCount + 1

	updates := map[string]interface{}{
		"login_count":   loginCount,
		"last_login_at": now,
		"last_login_ip": clientIP,
	}

	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		return nil, "", fmt.Errorf("failed to update login info: %w", err)
	}

	user.LoginCount = loginCount
	user.LastLoginAt = &now
	user.LastLoginIP = clientIP

	return &user, "", nil
}

func (s *UserService) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (s *UserService) UpdateProfile(userID uint, input *UpdateProfileInput) (*models.User, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, ErrUserNotFound
	}

	updates := make(map[string]interface{})
	if input.Nickname != "" {
		updates["nickname"] = input.Nickname
	}
	if input.Avatar != "" {
		updates["avatar"] = input.Avatar
	}
	if input.Phone != "" {
		updates["phone"] = input.Phone
	}
	if input.Bio != "" {
		updates["bio"] = input.Bio
	}

	if len(updates) == 0 {
		return &user, nil
	}

	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	database.DB.First(&user, userID)
	return &user, nil
}

func (s *UserService) ChangePassword(userID uint, input *ChangePasswordInput) error {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.OldPassword)); err != nil {
		return ErrInvalidPassword
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := database.DB.Model(&user).Update("password_hash", string(newHash)).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

func (s *UserService) RequestPasswordReset(email string) (string, error) {
	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}

	resetToken := generateToken(32)
	resetAt := time.Now().Add(1 * time.Hour)

	if err := database.DB.Model(&user).Updates(map[string]interface{}{
		"password_reset_token": resetToken,
		"password_reset_at":    resetAt,
	}).Error; err != nil {
		return "", fmt.Errorf("failed to set reset token: %w", err)
	}

	return resetToken, nil
}

func (s *UserService) ResetPassword(input *ResetPasswordInput) error {
	var user models.User
	if err := database.DB.Where("password_reset_token = ?", input.Token).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidToken
		}
		return err
	}

	if user.PasswordResetAt == nil || user.PasswordResetAt.Before(time.Now()) {
		return ErrTokenExpired
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := database.DB.Model(&user).Updates(map[string]interface{}{
		"password_hash":        string(newHash),
		"password_reset_token": "",
		"password_reset_at":    nil,
	}).Error; err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	return nil
}

func (s *UserService) VerifyEmail(token string) error {
	var user models.User
	if err := database.DB.Where("verification_token = ?", token).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidToken
		}
		return err
	}

	now := time.Now()
	if err := database.DB.Model(&user).Updates(map[string]interface{}{
		"is_verified":        true,
		"verified_at":        now,
		"verification_token": "",
	}).Error; err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	return nil
}

func (s *UserService) GenerateEmailVerificationToken(userID uint) (string, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return "", ErrUserNotFound
	}

	token := generateToken(32)
	if err := database.DB.Model(&user).Update("verification_token", token).Error; err != nil {
		return "", fmt.Errorf("failed to set verification token: %w", err)
	}

	return token, nil
}

func (s *UserService) ChangeEmail(userID uint, newEmail string) error {
	var existing models.User
	if err := database.DB.Where("email = ? AND id != ?", newEmail, userID).First(&existing).Error; err == nil {
		return ErrUserAlreadyExists
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return ErrUserNotFound
	}

	verificationToken := generateToken(32)
	if err := database.DB.Model(&user).Updates(map[string]interface{}{
		"email":              newEmail,
		"is_verified":        false,
		"verification_token": verificationToken,
	}).Error; err != nil {
		return fmt.Errorf("failed to change email: %w", err)
	}

	return nil
}

func generateToken(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}

func ToUserResponse(user *models.User) *UserResponse {
	return &UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		Phone:       user.Phone,
		Bio:         user.Bio,
		IsVerified:  user.IsVerified,
		VerifiedAt:  user.VerifiedAt,
		LoginCount:  user.LoginCount,
		LastLoginAt: user.LastLoginAt,
		LastLoginIP: user.LastLoginIP,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
	}
}
