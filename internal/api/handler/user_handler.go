package handler

import (
	"time"

	"hjtpx/internal/api/middleware"
	"hjtpx/internal/models"
	"hjtpx/internal/repository"
	"hjtpx/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	userRepo    *repository.UserRepository
	jwtManager *utils.JWTManager
}

func NewUserHandler(userRepo *repository.UserRepository, jwtManager *utils.JWTManager) *UserHandler {
	return &UserHandler{
		userRepo:    userRepo,
		jwtManager: jwtManager,
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	AppID    uint   `json:"app_id"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Username string `json:"username"`
	AppID    uint   `json:"app_id"`
}

func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if !utils.ValidateParams(c, &req) {
		return
	}

	existingUser, _ := h.userRepo.FindByEmail(req.Email)
	if existingUser != nil {
		utils.BadRequest(c, "Email already registered")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.InternalError(c, "Failed to hash password")
		return
	}

	user := &models.User{
		Email:    req.Email,
		Password: string(hashedPassword),
		Username: req.Username,
		AppID:    req.AppID,
		Status:   1,
	}

	if err := h.userRepo.Create(user); err != nil {
		utils.InternalError(c, "Failed to create user")
		return
	}

	utils.Created(c, gin.H{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
	})
}

func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if !utils.ValidateParams(c, &req) {
		return
	}

	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		utils.Unauthorized(c, "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		utils.Unauthorized(c, "Invalid email or password")
		return
	}

	if user.Status != 1 {
		utils.Forbidden(c, "Account is disabled")
		return
	}

	user.LastLogin = time.Now()
	h.userRepo.Update(user)

	token, err := h.jwtManager.GenerateToken(user.ID, user.Email, user.Username, user.AppID, "user")
	if err != nil {
		utils.InternalError(c, "Failed to generate token")
		return
	}

	utils.Success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"username":   user.Username,
			"created_at": user.CreatedAt,
		},
	})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		utils.Unauthorized(c, "User not authenticated")
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		utils.NotFound(c, "User not found")
		return
	}

	utils.Success(c, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"username":   user.Username,
		"app_id":     user.AppID,
		"status":     user.Status,
		"last_login": user.LastLogin,
		"created_at": user.CreatedAt,
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		utils.Unauthorized(c, "User not authenticated")
		return
	}

	var req models.UpdateUserRequest
	if !utils.ValidateParams(c, &req) {
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		utils.NotFound(c, "User not found")
		return
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

	if err := h.userRepo.Update(user); err != nil {
		utils.InternalError(c, "Failed to update user")
		return
	}

	utils.Success(c, gin.H{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
	})
}

func (h *UserHandler) Logout(c *gin.Context) {
	utils.Success(c, gin.H{
		"message": "Logged out successfully",
	})
}
