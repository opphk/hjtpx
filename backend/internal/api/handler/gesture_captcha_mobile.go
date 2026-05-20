package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var gesturePositions = map[int][2]int{
	1: {30, 30},
	2: {100, 30},
	3: {170, 30},
	4: {30, 100},
	5: {100, 100},
	6: {170, 100},
	7: {30, 170},
	8: {100, 170},
	9: {170, 170},
}

type GestureCaptchaSession struct {
	SessionID      string
	Pattern        string
	ShuffledPoints []int
	Status         string
	VerifyCount    int
	MaxAttempts    int
	CreatedAt      time.Time
	ExpiredAt      time.Time
	ClientIP       string
	UserAgent      string
	IsMobile       bool
	TouchData      *TouchVerificationData
}

type TouchVerificationData struct {
	TotalTouches    int
	TouchPressure   float64
	TouchDuration   int64
	TouchArea       float64
	VelocityProfile []float64
	IsMultiTouch    bool
}

var gestureSessions = make(map[string]*GestureCaptchaSession)
var gestureMutex sync.RWMutex

type VerifyGestureCaptchaRequest struct {
	ID          string `json:"id" binding:"required"`
	Pattern     string `json:"pattern" binding:"required"`
	TouchData   *TouchVerificationData `json:"touch_data,omitempty"`
	DeviceInfo  *DeviceInfo `json:"device_info,omitempty"`
}

type DeviceInfo struct {
	IsMobile      bool    `json:"is_mobile"`
	TouchCapable  bool    `json:"touch_capable"`
	MaxTouchPoints int    `json:"max_touch_points"`
	Platform      string  `json:"platform"`
	UserAgent     string  `json:"user_agent"`
	ScreenWidth   int     `json:"screen_width"`
	ScreenHeight  int     `json:"screen_height"`
}

type GestureCaptchaResponse struct {
	SessionID      string `json:"session_id"`
	Pattern        string `json:"pattern"`
	Hint           string `json:"hint"`
	GridSize       int    `json:"grid_size"`
	ExpiresIn      int64  `json:"expires_in"`
	ExpiresAt      int64  `json:"expires_at"`
	MobileOptimized bool  `json:"mobile_optimized"`
	PointPositions map[int][2]int `json:"point_positions"`
}

func GenerateGestureCaptcha(c *gin.Context) {
	sessionID := generateGestureSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	patternLength := 3 + rand.Intn(4)
	pattern := generateGesturePattern(patternLength)

	isMobile := detectMobileDevice(c)

	session := &GestureCaptchaSession{
		SessionID:      sessionID,
		Pattern:        pattern,
		Status:         "pending",
		VerifyCount:    0,
		MaxAttempts:    3,
		CreatedAt:      time.Now(),
		ExpiredAt:      expiresAt,
		ClientIP:       c.ClientIP(),
		UserAgent:      c.GetHeader("User-Agent"),
		IsMobile:       isMobile,
	}

	gestureMutex.Lock()
	gestureSessions[sessionID] = session
	gestureMutex.Unlock()

	hint := "Connect the dots in the pattern order"
	if isMobile {
		hint = "滑动连接圆点"
	}

	response.Success(c, GestureCaptchaResponse{
		SessionID:      sessionID,
		Pattern:        pattern,
		Hint:           hint,
		GridSize:       3,
		ExpiresIn:       int64(5 * time.Minute / time.Second),
		ExpiresAt:       expiresAt.Unix(),
		MobileOptimized: isMobile,
		PointPositions: gesturePositions,
	})
}

func VerifyGestureCaptcha(c *gin.Context) {
	var req VerifyGestureCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	gestureMutex.RLock()
	session, exists := gestureSessions[req.ID]
	gestureMutex.RUnlock()

	if !exists {
		response.Fail(c, response.CodeNotFound, "会话不存在或已过期")
		return
	}

	if time.Now().After(session.ExpiredAt) {
		gestureMutex.Lock()
		delete(gestureSessions, req.ID)
		gestureMutex.Unlock()
		response.Fail(c, response.CodeNotFound, "会话已过期")
		return
	}

	if session.VerifyCount >= session.MaxAttempts {
		response.Fail(c, response.CodeTooManyRequests, "验证次数已用完")
		return
	}

	gestureMutex.Lock()
	session.VerifyCount++
	gestureMutex.Unlock()

	var verificationScore float64 = 100
	var riskIndicators []string

	if req.TouchData != nil {
		riskIndicators = analyzeTouchData(req.TouchData)
		verificationScore = calculateTouchVerificationScore(req.TouchData)
	}

	if req.DeviceInfo != nil && req.DeviceInfo.IsMobile {
		if !session.IsMobile {
			riskIndicators = append(riskIndicators, "设备类型不匹配")
			verificationScore -= 20
		}
	}

	success, reason := verifyGesturePattern(session.Pattern, req.Pattern)

	if success && verificationScore >= 60 {
		gestureMutex.Lock()
		session.Status = "verified"
		delete(gestureSessions, req.ID)
		gestureMutex.Unlock()

		response.Success(c, gin.H{
			"success":           true,
			"message":           "验证成功",
			"score":             verificationScore,
			"risk_indicators":   riskIndicators,
		})
	} else {
		if !success {
			reason = "手势模式不匹配"
		} else {
			reason = "触摸特征异常"
		}
		
		response.Success(c, gin.H{
			"success":           false,
			"message":           "验证失败",
			"reason":            reason,
			"remaining_tries":   session.MaxAttempts - session.VerifyCount,
			"risk_indicators":   riskIndicators,
		})
	}
}

func analyzeTouchData(data *TouchVerificationData) []string {
	indicators := make([]string, 0)

	if data.TotalTouches == 0 {
		indicators = append(indicators, "缺少触摸数据")
	}

	if data.IsMultiTouch {
		indicators = append(indicators, "检测到多点触控")
	}

	if data.TouchDuration > 0 && data.TouchDuration < 50 {
		indicators = append(indicators, "触摸持续时间过短")
	}

	if data.TouchPressure > 0 && data.TouchPressure > 1.0 {
		indicators = append(indicators, "触摸压力异常")
	}

	if len(data.VelocityProfile) > 0 {
		avgVelocity := calculateAverage(data.VelocityProfile)
		if avgVelocity > 1000 {
			indicators = append(indicators, "触摸速度过快")
		}
	}

	return indicators
}

func calculateTouchVerificationScore(data *TouchVerificationData) float64 {
	score := 100.0

	if data.TotalTouches == 0 {
		score -= 30
	}

	if data.TouchDuration < 100 {
		score -= 15
	}

	if data.TouchPressure > 1.5 || data.TouchPressure < 0.1 {
		score -= 10
	}

	if len(data.VelocityProfile) > 0 {
		variance := calculateVariance(data.VelocityProfile)
		if variance < 0.01 {
			score -= 20
		}
	}

	return math.Max(0, math.Min(100, score))
}

func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateVariance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := calculateAverage(values)
	sum := 0.0
	for _, v := range values {
		sum += (v - mean) * (v - mean)
	}
	return sum / float64(len(values))
}

func generateGesturePattern(length int) string {
	points := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	
	for i := len(points) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		points[i], points[j] = points[j], points[i]
	}

	selected := points[:length]

	pattern := fmt.Sprintf("%d", selected[0])
	for i := 1; i < length; i++ {
		pattern += fmt.Sprintf("-%d", selected[i])
	}

	return pattern
}

func verifyGesturePattern(expected, actual string) (bool, string) {
	if expected == actual {
		return true, ""
	}

	expectedPoints := parsePattern(expected)
	actualPoints := parsePattern(actual)

	if len(expectedPoints) != len(actualPoints) {
		return false, fmt.Sprintf("点数量不匹配: 期望%d个点, 实际%d个点", len(expectedPoints), len(actualPoints))
	}

	for i, ep := range expectedPoints {
		if i >= len(actualPoints) {
			return false, fmt.Sprintf("第%d个点缺失", i+1)
		}
		if ep != actualPoints[i] {
			return false, fmt.Sprintf("第%d个点不匹配: 期望%d, 实际%d", i+1, ep, actualPoints[i])
		}
	}

	return true, ""
}

func parsePattern(pattern string) []int {
	if pattern == "" {
		return []int{}
	}

	var result []int
	var current int
	
	for _, ch := range pattern {
		if ch == '-' {
			if current > 0 {
				result = append(result, current)
				current = 0
			}
		} else if ch >= '0' && ch <= '9' {
			current = current*10 + int(ch-'0')
		}
	}
	
	if current > 0 {
		result = append(result, current)
	}

	return result
}

func generateGestureSessionID() string {
	return fmt.Sprintf("gesture_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func GetGestureCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	gestureMutex.RLock()
	session, exists := gestureSessions[sessionID]
	gestureMutex.RUnlock()

	if !exists {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, gin.H{
		"session_id":      session.SessionID,
		"status":          session.Status,
		"verify_count":    session.VerifyCount,
		"max_attempts":    session.MaxAttempts,
		"created_at":      session.CreatedAt.Unix(),
		"expires_at":      session.ExpiredAt.Unix(),
		"expires_in":      int64(time.Until(session.ExpiredAt).Seconds()),
		"is_mobile":      session.IsMobile,
	})
}

func init() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			cleanupExpiredGestureSessions()
		}
	}()
}

func cleanupExpiredGestureSessions() {
	now := time.Now()
	gestureMutex.Lock()
	defer gestureMutex.Unlock()
	for id, session := range gestureSessions {
		if now.After(session.ExpiredAt) {
			delete(gestureSessions, id)
		}
	}
}

func GetGestureGridPoints(c *gin.Context) {
	isMobile := detectMobileDevice(c)
	
	positions := gesturePositions
	if isMobile {
		scaleFactor := calculateScaleFactor(c)
		scaledPositions := make(map[int][2]int)
		for k, v := range positions {
			scaledPositions[k] = [2]int{
				int(float64(v[0]) * scaleFactor),
				int(float64(v[1]) * scaleFactor),
			}
		}
		positions = scaledPositions
	}

	response.Success(c, gin.H{
		"grid_size": 3,
		"points":    positions,
		"hint":      "按顺序连接数字点",
		"is_mobile": isMobile,
	})
}

func detectMobileDevice(c *gin.Context) bool {
	userAgent := c.GetHeader("User-Agent")
	mobileKeywords := []string{"Android", "iPhone", "iPad", "iPod", "BlackBerry", "Windows Phone", "Mobile"}
	
	for _, keyword := range mobileKeywords {
		if contains(userAgent, keyword) {
			return true
		}
	}
	
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func calculateScaleFactor(c *gin.Context) float64 {
	return 0.8
}

type RotateCaptchaRequest struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type RotateCaptchaResponse struct {
	SessionID   string `json:"session_id"`
	ImageData   string `json:"image_data"`
	TargetAngle int    `json:"target_angle"`
	CurrentAngle int   `json:"current_angle"`
	ExpiresIn   int64  `json:"expires_in"`
	ExpiresAt   int64  `json:"expires_at"`
}

type RotateVerifyRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Angle     int    `json:"angle" binding:"required"`
	TouchData *TouchVerificationData `json:"touch_data,omitempty"`
}

func GenerateRotateCaptcha(c *gin.Context) {
	var req RotateCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = RotateCaptchaRequest{Width: 200, Height: 200}
	}

	isMobile := detectMobileDevice(c)
	if isMobile {
		req.Width = 300
		req.Height = 300
	}

	sessionID := generateRotateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	targetAngle := rand.Intn(360)
	currentAngle := rand.Intn(360)

	imageData := generateRotateImage(targetAngle, req.Width, req.Height)

	rotateSessionData := map[string]interface{}{
		"target_angle": targetAngle,
		"created_at":   time.Now().Unix(),
		"expires_at":   expiresAt.Unix(),
		"is_mobile":    isMobile,
	}
	dataJSON, _ := json.Marshal(rotateSessionData)
	
	gestureMutex.Lock()
	gestureSessions[sessionID] = &GestureCaptchaSession{
		SessionID: sessionID,
		Pattern:   string(dataJSON),
		Status:    "pending",
		CreatedAt: time.Now(),
		ExpiredAt: expiresAt,
		IsMobile:  isMobile,
	}
	gestureMutex.Unlock()

	response.Success(c, RotateCaptchaResponse{
		SessionID:   sessionID,
		ImageData:   imageData,
		TargetAngle: targetAngle,
		CurrentAngle: currentAngle,
		ExpiresIn:   int64(5 * time.Minute / time.Second),
		ExpiresAt:   expiresAt.Unix(),
	})
}

func VerifyRotateCaptcha(c *gin.Context) {
	var req RotateVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	gestureMutex.RLock()
	session, exists := gestureSessions[req.SessionID]
	gestureMutex.RUnlock()

	if !exists {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(session.Pattern), &sessionData); err != nil {
		response.Fail(c, response.CodeServerError, "会话数据解析失败")
		return
	}

	targetAngle := int(sessionData["target_angle"].(float64))
	
	angleDiff := abs(req.Angle - targetAngle)
	if angleDiff > 180 {
		angleDiff = 360 - angleDiff
	}

	tolerance := 15
	success := angleDiff <= tolerance

	var riskIndicators []string
	verificationScore := float64(100)

	if req.TouchData != nil {
		riskIndicators = analyzeTouchData(req.TouchData)
		verificationScore = calculateTouchVerificationScore(req.TouchData)
	}

	if success && verificationScore >= 60 {
		gestureMutex.Lock()
		delete(gestureSessions, req.SessionID)
		gestureMutex.Unlock()

		response.Success(c, gin.H{
			"success":          true,
			"message":          "验证成功",
			"angle_diff":       angleDiff,
			"target_angle":      targetAngle,
			"your_angle":        req.Angle,
			"score":             100 - float64(angleDiff)*2,
			"risk_indicators":   riskIndicators,
		})
	} else {
		if !success {
			riskIndicators = append(riskIndicators, "旋转角度不正确")
		} else {
			riskIndicators = append(riskIndicators, "触摸特征异常")
		}
		
		response.Success(c, gin.H{
			"success":        false,
			"message":        "验证失败",
			"angle_diff":     angleDiff,
			"target_angle":   targetAngle,
			"your_angle":     req.Angle,
			"tolerance":      tolerance,
			"risk_indicators": riskIndicators,
		})
	}
}

func generateRotateSessionID() string {
	return fmt.Sprintf("rotate_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func generateRotateImage(targetAngle, width, height int) string {
	return fmt.Sprintf("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0i%dIiBoZWlnaHQ9Ii%dIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjxyZWN0IHdpZHRoPSIxMDAlIiBoZWlnaHQ9IjEwMCUiIGZpbGw9IiNmZmYiLz48cGF0aCBkPSJNMjAgNTBoNjB2LTJoLTYwem0wIDJ2NGgxMHYtNGgtMTB6bTIwIDB2NGgxMHYtNGgtMTB6bTIwIDB2NGgxMHYtNGgtMTB6bTAtMTB2NGgxMHYtNGgtMTB6bTIwIDB2NGgxMHYtNGgtMTB6bTIwIDB2NGgxMHYtNGgtMTB6IiBmaWxsPSIjMjIyIi8+PC9zdmc+", width, height)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
