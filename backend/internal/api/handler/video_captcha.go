package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var (
	videoGeneratorService *captcha.VideoGeneratorService
	videoVerifierService *captcha.VideoVerifierService
)

func InitVideoCaptchaHandler(gen *captcha.VideoGeneratorService, ver *captcha.VideoVerifierService) {
	videoGeneratorService = gen
	videoVerifierService = ver
}

func initVideoServices() {
	if videoGeneratorService == nil {
		videoGeneratorService = captcha.NewVideoGeneratorServiceSimple()
	}
	if videoVerifierService == nil {
		videoVerifierService = captcha.NewVideoVerifierServiceSimple()
	}
}

type VideoCaptchaGenerateRequest struct {
	Width      int `json:"width"`
	Height     int `json:"height"`
	Difficulty int `json:"difficulty"`
}

type VideoCaptchaVerifyRequest struct {
	SessionID   string                    `json:"session_id" binding:"required"`
	Answer      string                    `json:"answer" binding:"required"`
	BehaviorData captcha.VideoBehaviorData `json:"behavior_data"`
}

func VideoCaptchaGenerate(c *gin.Context) {
	initVideoServices()
	
	var req VideoCaptchaGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = VideoCaptchaGenerateRequest{}
	}

	createReq := &captcha.VideoCaptchaRequest{
		Width:       req.Width,
		Height:      req.Height,
		Difficulty:  req.Difficulty,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := videoGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成视频验证码失败")
		return
	}

	response.Success(c, result)
}

func VideoCaptchaVerify(c *gin.Context) {
	initVideoServices()
	
	var req VideoCaptchaVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VerifyVideoCaptchaRequest{
		SessionID:   req.SessionID,
		Answer:      req.Answer,
		BehaviorData: req.BehaviorData,
	}

	result, err := videoVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func VideoCaptchaOptions(c *gin.Context) {
	initVideoServices()
	
	options := gin.H{
		"difficulty_options": []gin.H{
			{"value": 1, "label": "简单", "description": "基础动作识别"},
			{"value": 2, "label": "中等", "description": "复杂动作识别"},
			{"value": 3, "label": "困难", "description": "快速动作识别"},
		},
		"video_formats":     []string{"mp4", "webm", "ogg"},
		"supported_actions": []string{
			"举手", "挥手", "点头", "摇头",
			"眨眼", "张嘴", "抬手", "放下",
			"向左看", "向右看", "向上看", "向下看",
		},
		"features": []string{
			"动作识别验证",
			"行为分析",
			"多难度模式",
			"实时风险评估",
		},
		"duration_options": gin.H{
			"simple":   5,
			"medium":   8,
			"hard":     12,
		},
		"max_attempts": 3,
		"expires_in":   300,
	}

	response.Success(c, options)
}

func VideoCaptchaStatus(c *gin.Context) {
	initVideoServices()
	
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	valid, message := videoVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}
