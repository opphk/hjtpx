package handler

import (
	"fmt"
	"math"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var sliderGeneratorService *captcha.GeneratorService
var sliderVerifierService *captcha.VerifierService
var sliderAnalyzer *service.SliderAnalyzer

func InitSliderCaptchaHandler(gen *captcha.GeneratorService, ver *captcha.VerifierService) {
	sliderGeneratorService = gen
	sliderVerifierService = ver
	sliderAnalyzer = service.NewSliderAnalyzer()
}

// SliderCaptchaRequest 滑动验证码创建请求
type SliderCaptchaRequest struct {
	Width        int `json:"width"`         // 验证码图片宽度
	Height       int `json:"height"`        // 验证码图片高度
	SliderWidth  int `json:"slider_width"`  // 滑动块宽度
	SliderHeight int `json:"slider_height"` // 滑动块高度
}

// SliderVerifyRequest 滑动验证码验证请求
type SliderVerifyRequest struct {
	SessionID          string                    `json:"session_id" binding:"required"`    // 会话 ID
	PositionX          float64                   `json:"position_x" binding:"required"`    // X 坐标
	PositionY          float64                   `json:"position_y" binding:"required"`    // Y 坐标
	RiskScore          float64                   `json:"risk_score"`                       // 风险评分
	TraceScore         float64                   `json:"trace_score"`                      // 轨迹评分
	EnvScore           float64                   `json:"env_score"`                        // 环境评分
	Trajectory         []TrajectoryPoint          `json:"trajectory"`                       // 轨迹数据
	BehaviorData       SliderBehaviorData         `json:"behavior_data"`                    // 行为数据
	TrajectoryMetadata  *TrajectoryMetadata       `json:"trajectory_metadata,omitempty"`    // 轨迹元数据
	DeviceInfo         *DeviceInfoRequest         `json:"device_info,omitempty"`            // 设备信息
	EncryptedPayload   string                    `json:"encrypted_payload,omitempty"`      // 加密载荷
}

type TrajectoryMetadata struct {
	Version         string  `json:"version"`
	PointCount      int     `json:"point_count"`
	StartTime       float64 `json:"start_time"`
	EndTime         float64 `json:"end_time"`
	Duration        int64   `json:"duration"`
	SamplingQuality float64 `json:"sampling_quality"`
}

type DeviceInfoRequest struct {
	TouchCapable     bool    `json:"touch_capable"`
	MaxTouchPoints   int     `json:"max_touch_points"`
	DevicePixelRatio float64 `json:"device_pixel_ratio"`
	ScreenWidth      int     `json:"screen_width"`
	ScreenHeight     int     `json:"screen_height"`
	ColorDepth       int     `json:"color_depth"`
	Language         string  `json:"language"`
	Platform         string  `json:"platform"`
	Timezone         string  `json:"timezone"`
}

type TrajectoryPoint struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Timestamp int64 `json:"timestamp"`
	Pressure  int   `json:"pressure"`  // 压力值（移动端）
	VelocityX int   `json:"velocity_x"` // X方向速度
	VelocityY int   `json:"velocity_y"` // Y方向速度
}

type SliderBehaviorData struct {
	StartTime       int64   `json:"start_time"`        // 开始时间
	EndTime         int64   `json:"end_time"`          // 结束时间
	Duration        int64   `json:"duration"`          // 总耗时(ms)
	RetryCount      int     `json:"retry_count"`       // 重试次数
	IsMobile        bool    `json:"is_mobile"`         // 是否移动端
	DeviceType      string  `json:"device_type"`       // 设备类型
	OsType          string  `json:"os_type"`           // 操作系统
	BrowserType     string  `json:"browser_type"`      // 浏览器类型
	ScreenWidth     int     `json:"screen_width"`      // 屏幕宽度
	ScreenHeight    int     `json:"screen_height"`     // 屏幕高度
	PixelRatio      float64 `json:"pixel_ratio"`       // 像素比
	NetworkType     string  `json:"network_type"`      // 网络类型
	Latency         int     `json:"latency"`          // 网络延迟(ms)
	ClickCount      int     `json:"click_count"`      // 点击次数
	MouseDownCount  int     `json:"mouse_down_count"` // 鼠标按下次数
	MouseUpCount    int     `json:"mouse_up_count"`   // 鼠标松开次数
	PathLength      float64 `json:"path_length"`      // 路径长度
	MaxVelocity     float64 `json:"max_velocity"`     // 最大速度
	AvgVelocity     float64 `json:"avg_velocity"`     // 平均速度
	Acceleration    float64 `json:"acceleration"`     // 加速度
	DirectionChanges int    `json:"direction_changes"` // 方向改变次数
	IsTouchDevice   bool    `json:"is_touch_device"`   // 是否触摸设备
	TouchPoints     int     `json:"touch_points"`     // 触摸点数
}

// CreateSliderCaptcha 创建滑动验证码
// @Summary 创建滑动验证码
// @Description 生成一个新的滑动验证码
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body SliderCaptchaRequest false "验证码参数"
// @Success 200 {object} map[string]interface{} "成功返回验证码数据"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "生成失败"
// @Router /api/v1/captcha/slider/create [post]
func CreateSliderCaptcha(c *gin.Context) {
	var req SliderCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数解析失败")
		return
	}

	createReq := &captcha.CreateCaptchaRequest{
		Width:        req.Width,
		Height:       req.Height,
		SliderWidth:  req.SliderWidth,
		SliderHeight: req.SliderHeight,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		Fingerprint:  c.GetHeader("X-Fingerprint"),
	}

	result, err := sliderGeneratorService.Create(c.Request.Context(), createReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "生成验证码失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// VerifySliderCaptcha 验证滑动验证码
// @Summary 验证滑动验证码
// @Description 验证用户对滑动验证码的操作，包含行为分析
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body SliderVerifyRequest true "验证请求"
// @Success 200 {object} map[string]interface{} "验证结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "会话不存在"
// @Failure 500 {object} map[string]interface{} "验证失败"
// @Router /api/v1/captcha/slider/verify [post]
func VerifySliderCaptcha(c *gin.Context) {
	var req SliderVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	if err := validateSliderVerifyRequest(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, err.Error())
		return
	}

	session, err := sliderVerifierService.GetSessionStatus(c.Request.Context(), req.SessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在或已过期")
		return
	}

	var analysisResult *service.SliderAnalysisResult
	if len(req.Trajectory) >= 3 && sliderAnalyzer != nil {
		sliderPoints := make([]service.SliderPoint, len(req.Trajectory))
		for i, p := range req.Trajectory {
			sliderPoints[i] = service.SliderPoint{
				X:         p.X,
				Y:         p.Y,
				Timestamp: p.Timestamp,
			}
		}
		analysisResult, err = sliderAnalyzer.AnalyzeWithHighSamplingSupport(sliderPoints, req.PositionX)
		if err != nil {
			analysisResult = nil
		}
	}

	adjustedRiskScore := req.RiskScore
	if analysisResult != nil {
		if analysisResult.IsBot {
			adjustedRiskScore = 0.9
		} else {
			adjustedRiskScore = 0.1
		}
	}

	verifyReq := &captcha.VerifyRequest{
		SessionID:  req.SessionID,
		PositionX:  req.PositionX,
		PositionY:  req.PositionY,
		RiskScore:  adjustedRiskScore,
		TraceScore: req.TraceScore,
		EnvScore:   req.EnvScore,
	}

	result, err := sliderVerifierService.Verify(c.Request.Context(), verifyReq)
	if err != nil {
		response.Fail(c, response.CodeServerError, "验证失败: "+err.Error())
		return
	}

	resultData := map[string]interface{}{
		"success":             result.Success,
		"message":             result.Message,
		"score":               result.Score,
		"position_diff":       result.PositionDiff,
		"trajectory_analysis":  nil,
		"behavior_summary":    nil,
	}

	if analysisResult != nil {
		resultData["trajectory_analysis"] = map[string]interface{}{
			"is_bot":             analysisResult.IsBot,
			"confidence":         analysisResult.Confidence,
			"anomaly_score":      analysisResult.AnomalyScore,
			"risk_indicators":    analysisResult.RiskIndicators,
			"overall_score":      analysisResult.OverallRiskScore,
			"pattern":            analysisResult.TrajectoryPattern,
			"speed_profile":      analysisResult.SpeedProfile,
			"sampling_rate":      calculateSamplingRate(req.Trajectory),
			"direction_changes":  countDirectionChanges(req.Trajectory),
			"path_length":        calculatePathLength(req.Trajectory),
			"path_efficiency":    calculatePathEfficiency(req.Trajectory),
		}
	}

	if session.GapX > 0 && analysisResult != nil {
		resultData["trajectory_analysis"].(map[string]interface{})["sampling_rate"] = calculateSamplingRate(req.Trajectory)
	}

	if req.BehaviorData.Duration > 0 {
		resultData["behavior_summary"] = map[string]interface{}{
			"duration_ms":   req.BehaviorData.Duration,
			"retry_count":   req.BehaviorData.RetryCount,
			"is_mobile":     req.BehaviorData.IsMobile,
			"device_type":   req.BehaviorData.DeviceType,
			"avg_velocity":  req.BehaviorData.AvgVelocity,
			"max_velocity":  req.BehaviorData.MaxVelocity,
			"path_length":   req.BehaviorData.PathLength,
			"network_type":  req.BehaviorData.NetworkType,
			"latency_ms":    req.BehaviorData.Latency,
		}
	}

	if req.TrajectoryMetadata != nil {
		resultData["trajectory_metadata"] = req.TrajectoryMetadata
	}

	if req.DeviceInfo != nil {
		resultData["device_info"] = req.DeviceInfo
	}

	response.Success(c, resultData)
}

func countDirectionChanges(trajectory []TrajectoryPoint) int {
	if len(trajectory) < 3 {
		return 0
	}
	count := 0
	for i := 1; i < len(trajectory)-1; i++ {
		dx1 := trajectory[i].X - trajectory[i-1].X
		dy1 := trajectory[i].Y - trajectory[i-1].Y
		dx2 := trajectory[i+1].X - trajectory[i].X
		dy2 := trajectory[i+1].Y - trajectory[i].Y

		if (dx1 >= 0 && dx2 < 0) || (dx1 < 0 && dx2 >= 0) {
			count++
		}
		if (dy1 >= 0 && dy2 < 0) || (dy1 < 0 && dy2 >= 0) {
			count++
		}
	}
	return count
}

func calculatePathLength(trajectory []TrajectoryPoint) float64 {
	if len(trajectory) < 2 {
		return 0
	}
	var length float64
	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		length += float64(time.Duration(trajectory[i].Timestamp-trajectory[i-1].Timestamp).Seconds()) * (dx*dx + dy*dy)
	}
	return length
}

func validateSliderVerifyRequest(req *SliderVerifyRequest) error {
	if req.SessionID == "" {
		return fmt.Errorf("session_id不能为空")
	}

	if req.PositionX < 0 || req.PositionX > 1000 {
		return fmt.Errorf("position_x超出有效范围")
	}

	if req.PositionY < 0 || req.PositionY > 1000 {
		return fmt.Errorf("position_y超出有效范围")
	}

	if len(req.Trajectory) > 1000 {
		return fmt.Errorf("轨迹点数量过多，最多支持1000个点")
	}

	if req.TrajectoryMetadata != nil {
		if req.TrajectoryMetadata.Duration < 0 || req.TrajectoryMetadata.Duration > 60000 {
			return fmt.Errorf("轨迹持续时间超出有效范围")
		}
	}

	return nil
}

func calculatePathEfficiency(trajectory []TrajectoryPoint) float64 {
	if len(trajectory) < 2 {
		return 0
	}

	directDistance := math.Sqrt(
		math.Pow(float64(trajectory[len(trajectory)-1].X-trajectory[0].X), 2) +
		math.Pow(float64(trajectory[len(trajectory)-1].Y-trajectory[0].Y), 2),
	)

	if directDistance == 0 {
		return 0
	}

	var pathLength float64
	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		pathLength += math.Sqrt(dx*dx + dy*dy)
	}

	if pathLength == 0 {
		return 0
	}

	return directDistance / pathLength
}

func analyzeTrajectory(trajectory []TrajectoryPoint, targetPosition int) *service.SliderAnalysisResult {
	if len(trajectory) < 3 || sliderAnalyzer == nil {
		return nil
	}

	sliderPoints := make([]service.SliderPoint, len(trajectory))
	for i, p := range trajectory {
		sliderPoints[i] = service.SliderPoint{
			X:         p.X,
			Y:         p.Y,
			Timestamp: p.Timestamp,
		}
	}

	result, err := sliderAnalyzer.AnalyzeWithHighSamplingSupport(sliderPoints, targetPosition)
	if err != nil {
		return nil
	}

	return result
}

func calculateSamplingRate(trajectory []TrajectoryPoint) float64 {
	if len(trajectory) < 2 {
		return 0
	}

	totalDuration := float64(trajectory[len(trajectory)-1].Timestamp - trajectory[0].Timestamp)
	if totalDuration <= 0 {
		return 0
	}

	return float64(len(trajectory)-1) / totalDuration * 1000
}

// GetSliderCaptchaStatus 获取验证码会话状态
// @Summary 获取验证码会话状态
// @Description 通过 session_id 获取验证码会话的当前状态
// @Tags 验证码
// @Accept json
// @Produce json
// @Param session_id path string true "会话 ID"
// @Success 200 {object} map[string]interface{} "会话状态"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "会话不存在"
// @Router /api/v1/captcha/slider/status/{session_id} [get]
func GetSliderCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	session, err := sliderVerifierService.GetSessionStatus(c.Request.Context(), sessionID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, session)
}

// CheckSliderCaptchaValid 检查验证码有效性
// @Summary 检查验证码是否有效
// @Description 检查验证码会话是否仍然有效
// @Tags 验证码
// @Accept json
// @Produce json
// @Param session_id path string true "会话 ID"
// @Success 200 {object} map[string]interface{} "检查结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /api/v1/captcha/slider/check/{session_id} [get]
func CheckSliderCaptchaValid(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	valid, message := sliderVerifierService.CheckSessionValid(c.Request.Context(), sessionID)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":   valid,
			"message": message,
		},
	})
}