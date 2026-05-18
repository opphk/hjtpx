package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var enhancedSliderGenerator *captcha.EnhancedSliderGenerator
var enhancedSliderVerifier *captcha.EnhancedSliderVerifier

func InitEnhancedSliderCaptchaHandler(gen *captcha.EnhancedSliderGenerator, ver *captcha.EnhancedSliderVerifier) {
	enhancedSliderGenerator = gen
	enhancedSliderVerifier = ver
}

type EnhancedSliderCaptchaRequest struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	SliderWidth  int    `json:"slider_width"`
	SliderHeight int    `json:"slider_height"`
	Difficulty   int    `json:"difficulty"`
	Mode         string `json:"mode"`
}

type EnhancedSliderVerifyRequest struct {
	SessionID       string                             `json:"session_id" binding:"required"`
	PositionX       int                                `json:"position_x" binding:"required"`
	PositionY       int                                `json:"position_y" binding:"required"`
	Trajectory      []EnhancedTrajectoryPointRequest  `json:"trajectory"`
	DragDuration    int64                              `json:"drag_duration"`
	ResistanceLevel int                                `json:"resistance_level"`
	Difficulty      int                                `json:"difficulty"`
	Obstacles       []ObstacleInfoRequest              `json:"obstacles"`
	TrackMode       string                             `json:"track_mode"`
}

type EnhancedTrajectoryPointRequest struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Timestamp int64   `json:"timestamp"`
	Pressure  float64 `json:"pressure,omitempty"`
	TiltX     float64 `json:"tilt_x,omitempty"`
	TiltY     float64 `json:"tilt_y,omitempty"`
}

type ObstacleInfoRequest struct {
	Type     string `json:"type"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Rotation int    `json:"rotation"`
}

func CreateEnhancedSliderCaptcha(c *gin.Context) {
	var req EnhancedSliderCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = EnhancedSliderCaptchaRequest{}
	}

	createReq := &captcha.EnhancedCreateRequest{
		Width:        req.Width,
		Height:       req.Height,
		SliderWidth:  req.SliderWidth,
		SliderHeight: req.SliderHeight,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Fingerprint:  c.GetHeader("X-Fingerprint"),
		Difficulty:   req.Difficulty,
		Mode:         req.Mode,
	}

	result, err := enhancedSliderGenerator.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成增强验证码失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

func VerifyEnhancedSliderCaptcha(c *gin.Context) {
	var req EnhancedSliderVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	trajectory := make([]captcha.EnhancedTrajectoryPoint, len(req.Trajectory))
	for i, p := range req.Trajectory {
		trajectory[i] = captcha.EnhancedTrajectoryPoint{
			X:         p.X,
			Y:         p.Y,
			Timestamp: p.Timestamp,
			Pressure:  p.Pressure,
			TiltX:     p.TiltX,
			TiltY:     p.TiltY,
		}
	}

	obstacles := make([]captcha.ObstacleInfo, len(req.Obstacles))
	for i, o := range req.Obstacles {
		obstacles[i] = captcha.ObstacleInfo{
			Type:     o.Type,
			X:        o.X,
			Y:        o.Y,
			Width:    o.Width,
			Height:   o.Height,
			Rotation: o.Rotation,
		}
	}

	verifyReq := &captcha.EnhancedVerifyRequest{
		SessionID:       req.SessionID,
		PositionX:       req.PositionX,
		PositionY:       req.PositionY,
		Trajectory:      trajectory,
		DragDuration:    req.DragDuration,
		ResistanceLevel: req.ResistanceLevel,
		Difficulty:      req.Difficulty,
		Obstacles:       obstacles,
		TrackMode:       req.TrackMode,
	}

	result, err := enhancedSliderVerifier.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

func GetEnhancedSliderCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := enhancedSliderVerifier.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

func CheckEnhancedSliderCaptchaValid(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	valid, message := enhancedSliderVerifier.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}

func GetEnhancedSliderDifficulty(c *gin.Context) {
	fingerprint := c.GetHeader("X-Fingerprint")
	
	difficulty := enhancedSliderVerifier.GetRecommendedDifficulty(fingerprint)
	
	response.Success(c, gin.H{
		"difficulty": difficulty,
		"fingerprint": fingerprint,
	})
}
