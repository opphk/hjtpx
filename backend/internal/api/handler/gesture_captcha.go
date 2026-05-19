package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

// 手势验证码的数字点阵位置定义
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

// GestureCaptchaSession 手势验证码会话
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
}

var gestureSessions = make(map[string]*GestureCaptchaSession)
var gestureMutex sync.RWMutex

// VerifyGestureCaptchaRequest 手势验证码验证请求
type VerifyGestureCaptchaRequest struct {
	ID      string `json:"id" binding:"required"`       // 验证码ID
	Pattern string `json:"pattern" binding:"required"` // 手势模式
}

// GestureCaptchaResponse 手势验证码响应
type GestureCaptchaResponse struct {
	SessionID      string `json:"session_id"`
	Pattern        string `json:"pattern"`
	Hint           string `json:"hint"`
	GridSize       int    `json:"grid_size"`
	ExpiresIn      int64  `json:"expires_in"`
	ExpiresAt      int64  `json:"expires_at"`
}

// GenerateGestureCaptcha 生成手势验证码
// @Summary 生成手势验证码
// @Description 生成一个新的手势点连验证码，包含3x3点阵
// @Tags 验证码
// @Accept json
// @Produce json
// @Success 200 {object} GestureCaptchaResponse "手势验证码数据"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/captcha/gesture [get]
func GenerateGestureCaptcha(c *gin.Context) {
	sessionID := generateGestureSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	// 生成目标手势模式 (3-6个点)
	patternLength := 3 + rand.Intn(4)
	pattern := generateGesturePattern(patternLength)

	// 创建会话
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
	}

	// 存储会话
	gestureMutex.Lock()
	gestureSessions[sessionID] = session
	gestureMutex.Unlock()

	response.Success(c, GestureCaptchaResponse{
		SessionID: sessionID,
		Pattern:   pattern,
		Hint:      "Connect the dots in the pattern order",
		GridSize:  3,
		ExpiresIn: int64(5 * time.Minute / time.Second),
		ExpiresAt: expiresAt.Unix(),
	})
}

// VerifyGestureCaptcha 验证手势验证码
// @Summary 验证手势验证码
// @Description 验证用户绘制的手势是否正确，支持容差检测
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body VerifyGestureCaptchaRequest true "手势验证请求"
// @Success 200 {object} map[string]interface{} "验证结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "会话不存在"
// @Failure 429 {object} map[string]interface{} "验证次数超限"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/captcha/gesture/verify [post]
func VerifyGestureCaptcha(c *gin.Context) {
	var req VerifyGestureCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	// 获取会话
	gestureMutex.RLock()
	session, exists := gestureSessions[req.ID]
	gestureMutex.RUnlock()

	if !exists {
		response.Fail(c, response.CodeNotFound, "会话不存在或已过期")
		return
	}

	// 检查会话是否过期
	if time.Now().After(session.ExpiredAt) {
		gestureMutex.Lock()
		delete(gestureSessions, req.ID)
		gestureMutex.Unlock()
		response.Fail(c, response.CodeNotFound, "会话已过期")
		return
	}

	// 检查验证次数
	if session.VerifyCount >= session.MaxAttempts {
		response.Fail(c, response.CodeTooManyRequests, "验证次数已用完")
		return
	}

	// 更新验证次数
	gestureMutex.Lock()
	session.VerifyCount++
	gestureMutex.Unlock()

	// 验证手势模式
	success, reason := verifyGesturePattern(session.Pattern, req.Pattern)

	if success {
		// 验证成功，标记会话为已验证并删除
		gestureMutex.Lock()
		session.Status = "verified"
		delete(gestureSessions, req.ID)
		gestureMutex.Unlock()

		response.Success(c, gin.H{
			"success": true,
			"message": "验证成功",
			"score":   100,
		})
	} else {
		response.Success(c, gin.H{
			"success":      false,
			"message":      "验证失败",
			"reason":       reason,
			"remaining_tries": session.MaxAttempts - session.VerifyCount,
		})
	}
}

// generateGesturePattern 生成随机手势模式
func generateGesturePattern(length int) string {
	points := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	
	// Fisher-Yates 打乱
	for i := len(points) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		points[i], points[j] = points[j], points[i]
	}

	// 取前length个点
	selected := points[:length]

	// 转换为字符串格式
	pattern := fmt.Sprintf("%d", selected[0])
	for i := 1; i < length; i++ {
		pattern += fmt.Sprintf("-%d", selected[i])
	}

	return pattern
}

// verifyGesturePattern 验证手势模式是否匹配
func verifyGesturePattern(expected, actual string) (bool, string) {
	if expected == actual {
		return true, ""
	}

	// 解析模式
	expectedPoints := parsePattern(expected)
	actualPoints := parsePattern(actual)

	if len(expectedPoints) != len(actualPoints) {
		return false, fmt.Sprintf("点数量不匹配: 期望%d个点, 实际%d个点", len(expectedPoints), len(actualPoints))
	}

	// 检查点序列是否相同（允许一定的顺序容错）
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

// parsePattern 解析手势模式字符串
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

// generateGestureSessionID 生成会话ID
func generateGestureSessionID() string {
	return fmt.Sprintf("gesture_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

// GetGestureCaptchaStatus 获取手势验证码状态
// @Summary 获取手势验证码状态
// @Description 通过 session_id 获取手势验证码会话的当前状态
// @Tags 验证码
// @Accept json
// @Produce json
// @Param session_id path string true "会话 ID"
// @Success 200 {object} map[string]interface{} "会话状态"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "会话不存在"
// @Router /api/v1/captcha/gesture/status/{session_id} [get]
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
	})
}

// 定期清理过期会话
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

// GetGestureGridPoints 获取手势点阵位置
// @Summary 获取手势点阵位置
// @Description 获取3x3手势点阵的坐标位置
// @Tags 验证码
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "点阵位置"
// @Router /api/v1/captcha/gesture/grid [get]
func GetGestureGridPoints(c *gin.Context) {
	response.Success(c, gin.H{
		"grid_size": 3,
		"points":    gesturePositions,
		"hint":      "按顺序连接数字点",
	})
}

// RotateCaptchaRequest 旋转验证码请求
type RotateCaptchaRequest struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// RotateCaptchaResponse 旋转验证码响应
type RotateCaptchaResponse struct {
	SessionID   string `json:"session_id"`
	ImageData   string `json:"image_data"`
	TargetAngle int    `json:"target_angle"`
	CurrentAngle int   `json:"current_angle"`
	ExpiresIn   int64  `json:"expires_in"`
	ExpiresAt   int64  `json:"expires_at"`
}

// RotateVerifyRequest 旋转验证码验证请求
type RotateVerifyRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Angle     int    `json:"angle" binding:"required"`
}

// GenerateRotateCaptcha 生成旋转验证码
// @Summary 生成旋转验证码
// @Description 生成一个需要用户旋转到正确角度的验证码
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body RotateCaptchaRequest false "验证码参数"
// @Success 200 {object} RotateCaptchaResponse "旋转验证码数据"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/captcha/rotate/create [post]
func GenerateRotateCaptcha(c *gin.Context) {
	var req RotateCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = RotateCaptchaRequest{Width: 200, Height: 200}
	}

	sessionID := generateRotateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	// 生成随机目标角度 (0-359度)
	targetAngle := rand.Intn(360)
	// 当前显示的旋转角度（随机偏移）
	currentAngle := rand.Intn(360)

	// 生成旋转验证码图片（简化实现）
	imageData := generateRotateImage(targetAngle, req.Width, req.Height)

	// 存储会话数据
	rotateSessionData := map[string]interface{}{
		"target_angle": targetAngle,
		"created_at":   time.Now().Unix(),
		"expires_at":   expiresAt.Unix(),
	}
	dataJSON, _ := json.Marshal(rotateSessionData)
	
	// 使用内存存储（实际项目应使用Redis）
	gestureMutex.Lock()
	gestureSessions[sessionID] = &GestureCaptchaSession{
		SessionID: sessionID,
		Pattern:   string(dataJSON),
		Status:    "pending",
		CreatedAt: time.Now(),
		ExpiredAt: expiresAt,
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

// VerifyRotateCaptcha 验证旋转验证码
// @Summary 验证旋转验证码
// @Description 验证用户旋转的角度是否正确
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body RotateVerifyRequest true "旋转验证请求"
// @Success 200 {object} map[string]interface{} "验证结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "会话不存在"
// @Failure 500 {object} map[string]interface{} "验证失败"
// @Router /api/v1/captcha/rotate/verify [post]
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

	// 解析会话数据
	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(session.Pattern), &sessionData); err != nil {
		response.Fail(c, response.CodeServerError, "会话数据解析失败")
		return
	}

	targetAngle := int(sessionData["target_angle"].(float64))
	
	// 计算角度差（考虑360度循环）
	angleDiff := abs(req.Angle - targetAngle)
	if angleDiff > 180 {
		angleDiff = 360 - angleDiff
	}

	// 容差范围为15度
	tolerance := 15
	success := angleDiff <= tolerance

	if success {
		gestureMutex.Lock()
		delete(gestureSessions, req.SessionID)
		gestureMutex.Unlock()

		response.Success(c, gin.H{
			"success":      true,
			"message":      "验证成功",
			"angle_diff":   angleDiff,
			"target_angle": targetAngle,
			"your_angle":   req.Angle,
			"score":        100 - float64(angleDiff)*2,
		})
	} else {
		response.Success(c, gin.H{
			"success":      false,
			"message":      "验证失败",
			"angle_diff":   angleDiff,
			"target_angle": targetAngle,
			"your_angle":   req.Angle,
			"tolerance":    tolerance,
		})
	}
}

func generateRotateSessionID() string {
	return fmt.Sprintf("rotate_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func generateRotateImage(targetAngle, width, height int) string {
	// 简化实现：返回一个简单的旋转验证码图片占位符
	return fmt.Sprintf("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0i%dIiBoZWlnaHQ9Ii%dIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjxyZWN0IHdpZHRoPSIxMDAlIiBoZWlnaHQ9IjEwMCUiIGZpbGw9IiNmZmYiLz48cGF0aCBkPSJNMjAgNTBoNjB2LTJoLTYwem0wIDJ2NGgxMHYtNGgtMTB6bTIwIDB2NGgxMHYtNGgtMTB6bTIwIDB2NGgxMHYtNGgtMTB6bTAtMTB2NGgxMHYtNGgtMTB6bTIwIDB2NGgxMHYtNGgtMTB6bTIwIDB2NGgxMHYtNGgtMTB6IiBmaWxsPSIjMjIyIi8+PC9zdmc+", width, height)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}