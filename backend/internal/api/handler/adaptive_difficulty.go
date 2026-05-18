package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
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

var adaptiveHandler = NewAdaptiveDifficultyHandler(service.NewAdaptiveDifficultyService())

func GetAdaptiveDifficultyHandler() *AdaptiveDifficultyHandler {
	return adaptiveHandler
}

// GetUserDifficulty 获取用户难度
// @Summary 获取用户难度
// @Description 根据用户ID获取当前自适应难度级别和用户档案信息
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Param user_id query string false "用户ID，不提供则自动生成匿名ID"
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/adaptive/difficulty [get]
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

// UpdateUserResultRequest 更新用户验证结果请求
type UpdateUserResultRequest struct {
	UserID  string `json:"user_id" binding:"required" example:"user123"`
	Success bool   `json:"success" binding:"required" example:"true"`
	Time    int64  `json:"time" example:"1500"`
}

// UpdateUserResult 更新用户验证结果
// @Summary 更新用户验证结果
// @Description 根据验证结果更新用户档案，自动调整难度级别
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Param body body UpdateUserResultRequest true "更新用户验证结果请求"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Router /api/v1/adaptive/result [post]
func (h *AdaptiveDifficultyHandler) UpdateUserResult(c *gin.Context) {
	var req UpdateUserResultRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	h.adaptiveService.UpdateProfile(req.UserID, req.Success, time.Duration(req.Time))

	difficulty := h.adaptiveService.GetDifficulty(req.UserID)

	response.Success(c, gin.H{
		"new_difficulty": difficulty,
	})
}

// GetConfig 获取配置
// @Summary 获取自适应难度配置
// @Description 获取自适应难度系统的当前配置参数
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=service.DifficultyConfig} "获取成功"
// @Router /api/v1/adaptive/config [get]
func (h *AdaptiveDifficultyHandler) GetConfig(c *gin.Context) {
	config := h.adaptiveService.GetConfig()
	response.Success(c, config)
}

// UpdateConfig 更新配置
// @Summary 更新自适应难度配置
// @Description 更新自适应难度系统的配置参数
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Param body body service.DifficultyConfig true "难度配置"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Router /api/v1/adaptive/config [put]
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
// @Summary 获取所有用户档案
// @Description 获取系统中所有用户的自适应难度档案（管理端）
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/adaptive/profiles [get]
func (h *AdaptiveDifficultyHandler) GetAllProfiles(c *gin.Context) {
	profiles := h.adaptiveService.GetAllProfiles()
	response.Success(c, profiles)
}

// AddBehaviorFlagRequest 添加行为标记请求
type AddBehaviorFlagRequest struct {
	UserID string `json:"user_id" binding:"required" example:"user123"`
	Flag   string `json:"flag" binding:"required" example:"suspicious_behavior"`
}

// AddBehaviorFlag 添加行为标记
// @Summary 添加行为标记
// @Description 为用户添加行为标记，用于调整难度评估
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Param body body AddBehaviorFlagRequest true "添加行为标记请求"
// @Success 200 {object} response.Response "添加成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Router /api/v1/adaptive/flag [post]
func (h *AdaptiveDifficultyHandler) AddBehaviorFlag(c *gin.Context) {
	var req AddBehaviorFlagRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	h.adaptiveService.AddBehaviorFlag(req.UserID, req.Flag)
	response.Success(c, gin.H{"message": "Flag added"})
}

// GetDifficultyForCaptcha 获取验证码难度
// @Summary 获取验证码难度
// @Description 根据用户ID获取适合的验证码难度级别，支持AB测试模式
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Param user_id query string false "用户ID，不提供则自动生成匿名ID"
// @Param ab_test query string false "是否启用AB测试，默认false"
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/adaptive/captcha-difficulty [get]
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

// Top-level functions

// GetUserDifficulty 获取用户难度
// @Summary 获取用户难度
// @Description 根据用户ID获取当前自适应难度级别和用户档案信息
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Param user_id query string false "用户ID，不提供则自动生成匿名ID"
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/adaptive/difficulty [get]
func GetUserDifficulty(c *gin.Context) {
	GetAdaptiveDifficultyHandler().GetUserDifficulty(c)
}

// UpdateUserResult 更新用户验证结果
// @Summary 更新用户验证结果
// @Description 根据验证结果更新用户档案，自动调整难度级别
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Param body body UpdateUserResultRequest true "更新用户验证结果请求"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Router /api/v1/adaptive/result [post]
func UpdateUserResult(c *gin.Context) {
	GetAdaptiveDifficultyHandler().UpdateUserResult(c)
}

// GetAdaptiveConfig 获取自适应难度配置
// @Summary 获取自适应难度配置
// @Description 获取自适应难度系统的当前配置参数
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=service.DifficultyConfig} "获取成功"
// @Router /api/v1/admin/adaptive/config [get]
func GetAdaptiveConfig(c *gin.Context) {
	GetAdaptiveDifficultyHandler().GetConfig(c)
}

// UpdateAdaptiveConfig 更新自适应难度配置
// @Summary 更新自适应难度配置
// @Description 更新自适应难度系统的配置参数
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Param body body service.DifficultyConfig true "难度配置"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Router /api/v1/admin/adaptive/config [put]
func UpdateAdaptiveConfig(c *gin.Context) {
	GetAdaptiveDifficultyHandler().UpdateConfig(c)
}

// GetAllAdaptiveProfiles 获取所有用户档案
// @Summary 获取所有用户档案
// @Description 获取系统中所有用户的自适应难度档案（管理端）
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/admin/adaptive/profiles [get]
func GetAllAdaptiveProfiles(c *gin.Context) {
	GetAdaptiveDifficultyHandler().GetAllProfiles(c)
}

// AddBehaviorFlag 添加行为标记
// @Summary 添加行为标记
// @Description 为用户添加行为标记，用于调整难度评估
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Param body body AddBehaviorFlagRequest true "添加行为标记请求"
// @Success 200 {object} response.Response "添加成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Router /api/v1/adaptive/flag [post]
func AddBehaviorFlag(c *gin.Context) {
	GetAdaptiveDifficultyHandler().AddBehaviorFlag(c)
}

// GetDifficultyForCaptcha 获取验证码难度
// @Summary 获取验证码难度
// @Description 根据用户ID获取适合的验证码难度级别，支持AB测试模式
// @Tags 自适应难度
// @Accept json
// @Produce json
// @Param user_id query string false "用户ID，不提供则自动生成匿名ID"
// @Param ab_test query string false "是否启用AB测试，默认false"
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/adaptive/captcha-difficulty [get]
func GetDifficultyForCaptcha(c *gin.Context) {
	GetAdaptiveDifficultyHandler().GetDifficultyForCaptcha(c)
}
