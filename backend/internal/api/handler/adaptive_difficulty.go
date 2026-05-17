package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"hjtpx/backend/internal/service"
	"hjtpx/backend/pkg/response"
)

// AdaptiveDifficultyHandler 自适应难度处理器
type AdaptiveDifficultyHandler struct {
	adaptiveService *service.AdaptiveDifficultyService
}

// NewAdaptiveDifficultyHandler 创建自适应难度处理器
func NewAdaptiveDifficultyHandler(adaptiveService *service.AdaptiveDifficultyService) *AdaptiveDifficultyHandler {
	return &AdaptiveDifficultyHandler{
		adaptiveService: adaptiveService,
	}
}

// GetUserDifficulty 获取用户难度
func (h *AdaptiveDifficultyHandler) GetUserDifficulty(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = "anonymous_" + strconv.FormatInt(time.Now().UnixNano()%1000000, 10)
	}

	difficulty := h.adaptiveService.GetDifficulty(userID)
	profile := h.adaptiveService.GetOrCreateProfile(userID)

	response.Success(c, gin.H{
		"difficulty": difficulty,
		"profile":    profile,
	})
}

// UpdateUserResult 更新用户验证结果
func (h *AdaptiveDifficultyHandler) UpdateUserResult(c *gin.Context) {
	var req struct {
		UserID  string        `json:"user_id" binding:"required"`
		Success bool          `json:"success" binding:"required"`
		Time    time.Duration `json:"time"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	h.adaptiveService.UpdateProfile(req.UserID, req.Success, req.Time)

	difficulty := h.adaptiveService.GetDifficulty(req.UserID)

	response.Success(c, gin.H{
		"new_difficulty": difficulty,
	})
}

// GetConfig 获取配置
func (h *AdaptiveDifficultyHandler) GetConfig(c *gin.Context) {
	config := h.adaptiveService.GetConfig()
	response.Success(c, config)
}

// UpdateConfig 更新配置
func (h *AdaptiveDifficultyHandler) UpdateConfig(c *gin.Context) {
	var config service.DifficultyConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.BadRequest(c, "Invalid config")
		return
	}

	h.adaptiveService.UpdateConfig(&config)
	response.Success(c, gin.H{"message": "Config updated"})
}

// GetAllProfiles 获取所有用户档案（管理端）
func (h *AdaptiveDifficultyHandler) GetAllProfiles(c *gin.Context) {
	profiles := h.adaptiveService.GetAllProfiles()
	response.Success(c, profiles)
}

// AddBehaviorFlag 添加行为标记
func (h *AdaptiveDifficultyHandler) AddBehaviorFlag(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		Flag   string `json:"flag" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	h.adaptiveService.AddBehaviorFlag(req.UserID, req.Flag)
	response.Success(c, gin.H{"message": "Flag added"})
}

// GetDifficultyForCaptcha 获取验证码难度
func (h *AdaptiveDifficultyHandler) GetDifficultyForCaptcha(c *gin.Context) {
	userID := c.Query("user_id")
	abTestStr := c.Query("ab_test")
	abTest := abTestStr == "true"

	if userID == "" {
		userID = "anonymous_" + strconv.FormatInt(time.Now().UnixNano()%1000000, 10)
	}

	difficulty := h.adaptiveService.GetDifficultyForCaptcha(userID, abTest)

	response.Success(c, gin.H{
		"difficulty": difficulty,
		"user_id":    userID,
	})
}
