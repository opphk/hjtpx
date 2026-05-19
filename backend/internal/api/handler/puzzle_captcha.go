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

// PuzzleCaptchaSession 拼图验证码会话
type PuzzleCaptchaSession struct {
	SessionID   string
	TargetX     int
	TargetY     int
	PuzzleSize  int
	ImageWidth  int
	ImageHeight int
	Status      string
	VerifyCount int
	MaxAttempts int
	CreatedAt   time.Time
	ExpiredAt   time.Time
}

var puzzleSessions = make(map[string]*PuzzleCaptchaSession)
var puzzleMutex sync.RWMutex

// PuzzleCaptchaResponse 拼图验证码响应
type PuzzleCaptchaResponse struct {
	SessionID   string `json:"session_id"`
	ImageData   string `json:"image_data"`
	PuzzleData  string `json:"puzzle_data"`
	TargetX     int    `json:"target_x"`
	TargetY     int    `json:"target_y"`
	PuzzleSize  int    `json:"puzzle_size"`
	ImageWidth  int    `json:"image_width"`
	ImageHeight int    `json:"image_height"`
	ExpiresIn   int64  `json:"expires_in"`
	ExpiresAt   int64  `json:"expires_at"`
}

// PuzzleVerifyRequest 拼图验证码验证请求
type PuzzleVerifyRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	OffsetX   int    `json:"offset_x" binding:"required"`
	OffsetY   int    `json:"offset_y"`
}

// GeneratePuzzleCaptcha 生成拼图验证码
// @Summary 生成拼图验证码
// @Description 生成一个滑块拼图验证码
// @Tags 验证码
// @Accept json
// @Produce json
// @Success 200 {object} PuzzleCaptchaResponse "拼图验证码数据"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/captcha/puzzle/create [post]
func GeneratePuzzleCaptcha(c *gin.Context) {
	sessionID := generatePuzzleSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	// 拼图参数
	puzzleSize := 60 + rand.Intn(20) // 60-80px
	imageWidth := 300
	imageHeight := 150

	// 计算目标位置（确保拼图块不会超出边界）
	maxX := imageWidth - puzzleSize - 20
	maxY := imageHeight - puzzleSize - 20
	targetX := 20 + rand.Intn(maxX)
	targetY := 20 + rand.Intn(maxY)

	// 生成拼图图片（简化实现）
	imageData := generatePuzzleImage(imageWidth, imageHeight, targetX, targetY, puzzleSize)
	puzzleData := generatePuzzlePiece(targetX, targetY, puzzleSize)

	// 创建会话
	session := &PuzzleCaptchaSession{
		SessionID:   sessionID,
		TargetX:     targetX,
		TargetY:     targetY,
		PuzzleSize:  puzzleSize,
		ImageWidth:  imageWidth,
		ImageHeight: imageHeight,
		Status:      "pending",
		VerifyCount: 0,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
		ExpiredAt:   expiresAt,
	}

	// 存储会话
	puzzleMutex.Lock()
	puzzleSessions[sessionID] = session
	puzzleMutex.Unlock()

	response.Success(c, PuzzleCaptchaResponse{
		SessionID:   sessionID,
		ImageData:   imageData,
		PuzzleData:  puzzleData,
		TargetX:     targetX,
		TargetY:     targetY,
		PuzzleSize:  puzzleSize,
		ImageWidth:  imageWidth,
		ImageHeight: imageHeight,
		ExpiresIn:   int64(5 * time.Minute / time.Second),
		ExpiresAt:   expiresAt.Unix(),
	})
}

// VerifyPuzzleCaptcha 验证拼图验证码
// @Summary 验证拼图验证码
// @Description 验证用户拖动拼图的位置是否正确
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body PuzzleVerifyRequest true "拼图验证请求"
// @Success 200 {object} map[string]interface{} "验证结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "会话不存在"
// @Failure 429 {object} map[string]interface{} "验证次数超限"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/captcha/puzzle/verify [post]
func VerifyPuzzleCaptcha(c *gin.Context) {
	var req PuzzleVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	// 获取会话
	puzzleMutex.RLock()
	session, exists := puzzleSessions[req.SessionID]
	puzzleMutex.RUnlock()

	if !exists {
		response.Fail(c, response.CodeNotFound, "会话不存在或已过期")
		return
	}

	// 检查会话是否过期
	if time.Now().After(session.ExpiredAt) {
		puzzleMutex.Lock()
		delete(puzzleSessions, req.SessionID)
		puzzleMutex.Unlock()
		response.Fail(c, response.CodeNotFound, "会话已过期")
		return
	}

	// 检查验证次数
	if session.VerifyCount >= session.MaxAttempts {
		response.Fail(c, response.CodeTooManyRequests, "验证次数已用完")
		return
	}

	// 更新验证次数
	puzzleMutex.Lock()
	session.VerifyCount++
	puzzleMutex.Unlock()

	// 计算位置偏差（主要检查X坐标，Y坐标允许较大偏差）
	xDiff := abs(req.OffsetX - session.TargetX)
	yDiff := abs(req.OffsetY - session.TargetY)

	// 设置容差范围
	xTolerance := 15
	yTolerance := 20

	success := xDiff <= xTolerance && yDiff <= yTolerance

	if success {
		// 验证成功
		puzzleMutex.Lock()
		session.Status = "verified"
		delete(puzzleSessions, req.SessionID)
		puzzleMutex.Unlock()

		response.Success(c, gin.H{
			"success":    true,
			"message":    "验证成功",
			"score":      100 - float64(xDiff)*2,
			"offset_x":   req.OffsetX,
			"target_x":   session.TargetX,
			"offset_y":   req.OffsetY,
			"target_y":   session.TargetY,
			"x_diff":     xDiff,
			"y_diff":     yDiff,
		})
	} else {
		response.Success(c, gin.H{
			"success":         false,
			"message":         "验证失败",
			"offset_x":        req.OffsetX,
			"target_x":        session.TargetX,
			"offset_y":        req.OffsetY,
			"target_y":        session.TargetY,
			"x_diff":          xDiff,
			"y_diff":          yDiff,
			"x_tolerance":     xTolerance,
			"y_tolerance":     yTolerance,
			"remaining_tries": session.MaxAttempts - session.VerifyCount,
		})
	}
}

// GetPuzzleCaptchaStatus 获取拼图验证码状态
// @Summary 获取拼图验证码状态
// @Description 通过 session_id 获取拼图验证码会话的当前状态
// @Tags 验证码
// @Accept json
// @Produce json
// @Param session_id path string true "会话 ID"
// @Success 200 {object} map[string]interface{} "会话状态"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "会话不存在"
// @Router /api/v1/captcha/puzzle/status/{session_id} [get]
func GetPuzzleCaptchaStatus(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.Fail(c, response.CodeInvalidParams, "session_id不能为空")
		return
	}

	puzzleMutex.RLock()
	session, exists := puzzleSessions[sessionID]
	puzzleMutex.RUnlock()

	if !exists {
		response.Fail(c, response.CodeNotFound, "会话不存在")
		return
	}

	response.Success(c, gin.H{
		"session_id":   session.SessionID,
		"status":       session.Status,
		"verify_count": session.VerifyCount,
		"max_attempts": session.MaxAttempts,
		"created_at":   session.CreatedAt.Unix(),
		"expires_at":   session.ExpiredAt.Unix(),
		"expires_in":   int64(time.Until(session.ExpiredAt).Seconds()),
	})
}

func generatePuzzleSessionID() string {
	return fmt.Sprintf("puzzle_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func generatePuzzleImage(width, height, targetX, targetY, puzzleSize int) string {
	// 简化实现：返回一个简单的拼图验证码图片占位符
	return fmt.Sprintf("data:image/svg+xml;base64,PHN2ZyB3aWR0aD0i%dIiBoZWlnaHQ9Ii%dIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjxyZWN0IHdpZHRoPSIxMDAlIiBoZWlnaHQ9IjEwMCUiIGZpbGw9IiNmZmYiLz48cGF0aCBkPSJNMjAgMzBoMjYwdjkwSDB2LTkwem0zMCAzMHY1MHYxMGg0MHYtMTB2LTUwSDB2LTUwSDB2LTUwSDB2LTUwSDMwem0xMDAgMHY1MHYxMGg0MHYtMTB2LTUwSDB2LTUwSDB2LTUwSDB2LTUwSDMwem0xMDAgMHY1MHYxMGg0MHYtMTB2LTUwSDB2LTUwSDB2LTUwSDB2LTUwSDMwem0tMTMwIDEwdjMwSDB2LTMwem0wIDUwdiMwSDB2LTMwem0xMDAgMHYzMDBoLTgwdi0zMDBoODB6IiBmaWxsPSIjNjY2Ii8+PC9zdmc+", width, height)
}

func generatePuzzlePiece(targetX, targetY, puzzleSize int) string {
	// 简化实现：返回拼图块数据
	data := map[string]interface{}{
		"x":     targetX,
		"y":     targetY,
		"width": puzzleSize,
		"height": puzzleSize,
	}
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

// 定期清理过期会话
func init() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			cleanupExpiredPuzzleSessions()
		}
	}()
}

func cleanupExpiredPuzzleSessions() {
	now := time.Now()
	puzzleMutex.Lock()
	defer puzzleMutex.Unlock()
	for id, session := range puzzleSessions {
		if now.After(session.ExpiredAt) {
			delete(puzzleSessions, id)
		}
	}
}