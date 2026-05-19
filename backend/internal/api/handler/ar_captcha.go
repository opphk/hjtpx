package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var arGeneratorService *captcha.ARGeneratorService
var arVerifierService *captcha.ARVerifierService

func InitARCaptchaHandler(
	gen *captcha.ARGeneratorService,
	ver *captcha.ARVerifierService,
) {
	arGeneratorService = gen
	arVerifierService = ver
}

type ARCaptchaGenerateRequest struct {
	SceneType  string `json:"sceneType"`
	Difficulty string `json:"difficulty"`
}

type ARCaptchaVerifyRequest struct {
	SessionID    string                    `json:"sessionID" binding:"required"`
	UserGesture  *captcha.UserGesture      `json:"userGesture" binding:"required"`
	ObjectID     int                       `json:"objectID"`
	PositionX    float64                   `json:"positionX"`
	PositionY    float64                   `json:"positionY"`
	PositionZ    float64                   `json:"positionZ"`
	RiskScore    float64                  `json:"riskScore"`
}

func ARCaptchaGenerate(c *gin.Context) {
	var req ARCaptchaGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = ARCaptchaGenerateRequest{}
	}

	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}

	createReq := &captcha.CreateARRequest{
		SceneType:   req.SceneType,
		Difficulty:  req.Difficulty,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.GetHeader("User-Agent"),
		Fingerprint: c.GetHeader("X-Fingerprint"),
	}

	result, err := arGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成验证码失败")
		return
	}

	response.Success(c, result)
}

func ARCaptchaVerify(c *gin.Context) {
	var req ARCaptchaVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误")
		return
	}

	verifyReq := &captcha.VerifyARRequest{
		SessionID:   req.SessionID,
		UserGesture: req.UserGesture,
		ObjectID:    req.ObjectID,
		PositionX:   req.PositionX,
		PositionY:   req.PositionY,
		PositionZ:   req.PositionZ,
		RiskScore:   req.RiskScore,
	}

	result, err := arVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败")
		return
	}

	response.Success(c, result)
}

func ARCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "sessionID不能为空")
		return
	}

	session, err := arVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

func ARCaptchaOptions(c *gin.Context) {
	response.Success(c, gin.H{
		"scene_types": []gin.H{
			{"value": "object_placement", "label": "物体放置"},
			{"value": "gesture_recognition", "label": "手势识别"},
			{"value": "spatial_puzzle", "label": "空间拼图"},
			{"value": "object_tracking", "label": "物体追踪"},
			{"value": "depth_estimation", "label": "深度估计"},
		},
		"difficulty_options": []gin.H{
			{"value": "easy", "label": "简单"},
			{"value": "medium", "label": "中等"},
			{"value": "hard", "label": "困难"},
			{"value": "expert", "label": "专家"},
		},
		"gesture_types": []string{
			"tap", "swipe_left", "swipe_right", "swipe_up", "swipe_down",
			"circle", "triangle", "square", "pinch", "rotate",
		},
		"object_types": []string{
			"cube", "sphere", "pyramid", "cylinder", "cone",
			"torus", "star", "heart", "diamond", "ring",
		},
		"features": []string{
			"3D渲染",
			"WebXR支持",
			"行为分析",
			"AI检测",
		},
	})
}
