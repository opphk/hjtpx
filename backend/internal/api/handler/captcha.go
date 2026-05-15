package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type CaptchaSession struct {
	ID         string
	Type       string
	TargetX    int
	TargetY    int
	Points     [][2]int
	Hint       string
	MaxPoints  int
	CreatedAt  time.Time
}

var (
	captchaSessions = make(map[string]*CaptchaSession)
	sessionMutex    sync.RWMutex
	behaviorService = service.NewBehaviorAnalysisService()
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func generateSliderImage() (string, int, int) {
	targetX := 150 + rand.Intn(100)
	targetY := 50 + rand.Intn(100)

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="360" height="220">
		<defs>
			<linearGradient id="bg" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
				<stop offset="0%%" style="stop-color:#667eea"/>
				<stop offset="100%%" style="stop-color:#764ba2"/>
			</linearGradient>
		</defs>
		<rect width="100%%" height="100%%" fill="url(#bg)"/>
		<text x="180" y="40" text-anchor="middle" fill="white" font-size="18" font-family="Arial">
			拖动滑块完成验证
		</text>
		<rect x="%d" y="%d" width="50" height="50" fill="rgba(255,255,255,0.25)" stroke="white" stroke-width="2"/>
	</svg>`, targetX, targetY)

	encoded := base64.StdEncoding.EncodeToString([]byte(svg))
	return "data:image/svg+xml;base64," + encoded, targetX, targetY
}

func generateClickImage() (string, string, [][2]int, int) {
	points := [][2]int{
		{60, 110},
		{180, 110},
		{300, 110},
	}

	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="360" height="220">
		<defs>
			<linearGradient id="bg2" x1="0%" y1="0%" x2="100%" y2="100%">
				<stop offset="0%" style="stop-color:#f093fb"/>
				<stop offset="100%" style="stop-color:#f5576c"/>
			</linearGradient>
		</defs>
		<rect width="100%" height="100%" fill="url(#bg2)"/>
		<text x="60" y="110" text-anchor="middle" fill="white" font-size="32" font-family="Arial" font-weight="bold">1</text>
		<text x="180" y="110" text-anchor="middle" fill="white" font-size="32" font-family="Arial" font-weight="bold">2</text>
		<text x="300" y="110" text-anchor="middle" fill="white" font-size="32" font-family="Arial" font-weight="bold">3</text>
	</svg>`

	encoded := base64.StdEncoding.EncodeToString([]byte(svg))
	return "data:image/svg+xml;base64," + encoded, "请依次点击: 1, 2, 3", points, 3
}

func GetSliderCaptcha(c *gin.Context) {
	sessionID := generateSessionID()
	imageURL, targetX, targetY := generateSliderImage()

	session := &CaptchaSession{
		ID:        sessionID,
		Type:      "slider",
		TargetX:   targetX,
		TargetY:   targetY,
		CreatedAt: time.Now(),
	}

	sessionMutex.Lock()
	captchaSessions[sessionID] = session
	sessionMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"image_url":  imageURL,
		"puzzle_y":   targetY,
	})
}

func GetClickCaptcha(c *gin.Context) {
	sessionID := generateSessionID()
	imageURL, hint, points, maxPoints := generateClickImage()

	session := &CaptchaSession{
		ID:        sessionID,
		Type:      "click",
		Points:    points,
		Hint:      hint,
		MaxPoints: maxPoints,
		CreatedAt: time.Now(),
	}

	sessionMutex.Lock()
	captchaSessions[sessionID] = session
	sessionMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"image_url":  imageURL,
		"hint":       hint,
		"max_points": maxPoints,
	})
}

type VerifyRequest struct {
	SessionID     string                  `json:"session_id" binding:"required"`
	Type          string                  `json:"type" binding:"required"`
	X             int                     `json:"x"`
	Y             int                     `json:"y"`
	Points        [][2]int                `json:"points"`
	BehaviorData  []BehaviorDataPoint     `json:"behavior_data"`
	ApplicationID uint                    `json:"application_id"`
}

type BehaviorDataPoint struct {
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Timestamp int64   `json:"timestamp"`
	Event     string  `json:"event"`
}

func VerifyCaptcha(c *gin.Context) {
	startTime := time.Now()
	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的请求参数",
		})
		return
	}

	sessionMutex.RLock()
	session, exists := captchaSessions[req.SessionID]
	sessionMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "会话不存在或已过期",
		})
		return
	}

	if session.Type != req.Type {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "验证类型不匹配",
		})
		return
	}

	var captchaSuccess bool

	if req.Type == "slider" {
		tolerance := 10
		captchaSuccess = abs(req.X - session.TargetX) <= tolerance && abs(req.Y - session.TargetY) <= tolerance
	} else if req.Type == "click" {
		if len(req.Points) != session.MaxPoints {
			captchaSuccess = false
		} else {
			captchaSuccess = true
			tolerance := 30
			for i := 0; i < session.MaxPoints; i++ {
				if abs(req.Points[i][0]-session.Points[i][0]) > tolerance ||
					abs(req.Points[i][1]-session.Points[i][1]) > tolerance {
					captchaSuccess = false
					break
				}
			}
		}
	}

	db := database.GetDB()

	behaviorDataList := make([]models.BehaviorData, 0, len(req.BehaviorData))
	for _, dp := range req.BehaviorData {
		dataJSON, _ := json.Marshal(dp)
		behaviorDataList = append(behaviorDataList, models.BehaviorData{
			Data:      string(dataJSON),
			DataType:  dp.Event,
			Timestamp: time.UnixMilli(dp.Timestamp),
		})
	}

	finalSuccess, riskScore, analysisReport := behaviorService.VerifyWithBehaviorAnalysis(
		captchaSuccess,
		behaviorDataList,
	)

	status := "failed"
	if finalSuccess {
		status = "success"
		sessionMutex.Lock()
		delete(captchaSessions, req.SessionID)
		sessionMutex.Unlock()
	}

	duration := time.Since(startTime).Milliseconds()

	verification := &models.Verification{
		SessionID:    req.SessionID,
		CaptchaType:  req.Type,
		ApplicationID: req.ApplicationID,
		UserID:       0,
		Status:       status,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		RiskScore:    riskScore,
		BehaviorData: behaviorDataList,
	}

	if err := db.Create(verification).Error; err != nil {
		fmt.Printf("Failed to save verification: %v\n", err)
	}

	logEntry := &models.VerificationLog{
		VerificationID: verification.ID,
		SessionID:      req.SessionID,
		ApplicationID:  req.ApplicationID,
		CaptchaType:    req.Type,
		Status:         status,
		IPAddress:      c.ClientIP(),
		UserAgent:      c.GetHeader("User-Agent"),
		RiskScore:      riskScore,
		AnalysisResult: analysisReport,
		Duration:       duration,
	}

	if err := db.Create(logEntry).Error; err != nil {
		fmt.Printf("Failed to save verification log: %v\n", err)
	}

	message := "验证失败"
	if finalSuccess {
		message = "验证成功"
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      finalSuccess,
		"message":      message,
		"risk_score":   riskScore,
		"captcha_pass": captchaSuccess,
	})
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
