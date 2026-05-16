package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

type ProgressiveVerificationHandler struct {
	trustService *service.DeviceTrustService
}

type ProgressiveSession struct {
	SessionID     string      `json:"session_id"`
	Fingerprint   string      `json:"fingerprint"`
	CurrentLevel  int         `json:"current_level"`
	TrustScore    int         `json:"trust_score"`
	RiskScore     float64     `json:"risk_score"`
	Passed        bool        `json:"passed"`
	Challenges    []Challenge `json:"challenges"`
	CreatedAt     time.Time   `json:"created_at"`
	ExpiresAt     time.Time   `json:"expires_at"`
}

type Challenge struct {
	Type        string `json:"type"`
	Token       string `json:"token"`
	Data        string `json:"data,omitempty"`
	Completed   bool   `json:"completed"`
	CompletedAt int64  `json:"completed_at,omitempty"`
}

var (
	progressiveSessions = make(map[string]*ProgressiveSession)
	progressiveMutex   sync.RWMutex
)

type ProgressiveStartRequest struct {
	Fingerprint   string                 `json:"fingerprint" binding:"required"`
	Data          map[string]interface{} `json:"data,omitempty"`
	RequiredLevel int                    `json:"required_level,omitempty"`
}

type ProgressiveStartResponse struct {
	Success            bool       `json:"success"`
	SessionID          string     `json:"session_id"`
	Level              int        `json:"level"`
	LevelName          string     `json:"level_name"`
	TrustScore         int        `json:"trust_score"`
	RiskScore          float64    `json:"risk_score"`
	Challenges         []Challenge `json:"challenges,omitempty"`
	ShouldVerify       bool       `json:"should_verify"`
	Message            string     `json:"message"`
	SkipVerification   bool       `json:"skip_verification"`
}

type ProgressiveChallengeRequest struct {
	SessionID      string `json:"session_id" binding:"required"`
	ChallengeToken string `json:"challenge_token" binding:"required"`
	Result         string `json:"result"`
}

type ProgressiveChallengeResponse struct {
	Success      bool     `json:"success"`
	SessionID    string   `json:"session_id"`
	Passed       bool     `json:"passed"`
	CurrentLevel int      `json:"current_level"`
	TrustScore   int      `json:"trust_score"`
	RiskScore    float64  `json:"risk_score"`
	ShouldVerify bool     `json:"should_verify"`
	Message      string   `json:"message"`
}

func GetProgressiveVerificationHandler() *ProgressiveVerificationHandler {
	return &ProgressiveVerificationHandler{
		trustService: service.GetDeviceTrustService(),
	}
}

func (h *ProgressiveVerificationHandler) StartVerification(c *gin.Context) {
	var req ProgressiveStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	if req.Data == nil {
		req.Data = make(map[string]interface{})
	}

	if ua := c.GetHeader("User-Agent"); ua != "" {
		req.Data["user_agent"] = ua
	}
	if ip := c.ClientIP(); ip != "" {
		req.Data["ip_address"] = ip
	}

	skip, reason := h.trustService.ShouldSkipVerification(c.Request.Context(), req.Fingerprint)
	if skip {
		c.JSON(http.StatusOK, ProgressiveStartResponse{
			Success:            true,
			SessionID:          generateProgressiveSessionID(),
			Level:              0,
			LevelName:          "silent",
			TrustScore:         100,
			RiskScore:          0,
			ShouldVerify:       false,
			SkipVerification:   true,
			Message:            fmt.Sprintf("验证已跳过: %s", reason),
		})
		return
	}

	decision := h.trustService.EvaluateTrust(req.Fingerprint, req.Data)

	level, levelName := getProgressiveLevel(decision.TrustScore, decision.RiskScore, req.Fingerprint == "")

	if req.RequiredLevel > 0 && req.RequiredLevel > level {
		level = req.RequiredLevel
		levelName = getLevelName(level)
	}

	session := &ProgressiveSession{
		SessionID:    generateProgressiveSessionID(),
		Fingerprint:  req.Fingerprint,
		CurrentLevel: level,
		TrustScore:   decision.TrustScore,
		RiskScore:    decision.RiskScore,
		Passed:       false,
		Challenges:   generateChallenges(level),
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}

	progressiveMutex.Lock()
	progressiveSessions[session.SessionID] = session
	progressiveMutex.Unlock()

	response := ProgressiveStartResponse{
		Success:      true,
		SessionID:    session.SessionID,
		Level:        level,
		LevelName:    levelName,
		TrustScore:   decision.TrustScore,
		RiskScore:    decision.RiskScore,
		ShouldVerify: level > 0,
		Message:      getVerificationMessage(level, decision.TrustScore, decision.RiskScore),
	}

	if level > 0 {
		response.Challenges = session.Challenges
	}

	c.JSON(http.StatusOK, response)
}

func (h *ProgressiveVerificationHandler) CompleteChallenge(c *gin.Context) {
	var req ProgressiveChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	progressiveMutex.RLock()
	session, exists := progressiveSessions[req.SessionID]
	progressiveMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "session not found or expired",
		})
		return
	}

	if time.Now().After(session.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "session expired",
		})
		return
	}

	challengeCompleted := false
	for i, challenge := range session.Challenges {
		if challenge.Token == req.ChallengeToken && !challenge.Completed {
			session.Challenges[i].Completed = true
			session.Challenges[i].CompletedAt = time.Now().UnixMilli()
			challengeCompleted = true
			break
		}
	}

	if !challengeCompleted {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "challenge not found or already completed",
		})
		return
	}

	allCompleted := true
	for _, challenge := range session.Challenges {
		if !challenge.Completed {
			allCompleted = false
			break
		}
	}

	if allCompleted {
		session.Passed = true
		session.TrustScore = minInt(100, session.TrustScore+10)
		h.trustService.UpdateTrustScore(session.Fingerprint, "verify", c.ClientIP(), c.GetHeader("User-Agent"))
	}

	progressiveMutex.Lock()
	progressiveSessions[session.SessionID] = session
	progressiveMutex.Unlock()

	c.JSON(http.StatusOK, ProgressiveChallengeResponse{
		Success:      true,
		SessionID:    session.SessionID,
		Passed:       session.Passed,
		CurrentLevel: session.CurrentLevel,
		TrustScore:   session.TrustScore,
		RiskScore:    session.RiskScore,
		ShouldVerify: !session.Passed && session.CurrentLevel > 0,
		Message:      getChallengeMessage(session.Passed),
	})
}

func (h *ProgressiveVerificationHandler) GetSessionStatus(c *gin.Context) {
	sessionID := c.Param("session_id")

	progressiveMutex.RLock()
	session, exists := progressiveSessions[sessionID]
	progressiveMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "session not found or expired",
		})
		return
	}

	allCompleted := true
	for _, challenge := range session.Challenges {
		if !challenge.Completed {
			allCompleted = false
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"session_id":    session.SessionID,
			"fingerprint":   session.Fingerprint,
			"current_level": session.CurrentLevel,
			"level_name":    getLevelName(session.CurrentLevel),
			"trust_score":   session.TrustScore,
			"risk_score":    session.RiskScore,
			"passed":        session.Passed,
			"challenges":    session.Challenges,
			"all_completed": allCompleted,
			"created_at":    session.CreatedAt,
			"expires_at":    session.ExpiresAt,
		},
	})
}

func (h *ProgressiveVerificationHandler) CancelVerification(c *gin.Context) {
	sessionID := c.Param("session_id")

	progressiveMutex.Lock()
	delete(progressiveSessions, sessionID)
	progressiveMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "verification cancelled",
	})
}

func (h *ProgressiveVerificationHandler) GetVerificationLevels(c *gin.Context) {
	levels := []map[string]interface{}{
		{
			"level":              0,
			"name":               "silent",
			"description":        "静默验证，无需用户交互",
			"trust_score_min":    80,
			"risk_score_max":     20,
			"requires_challenge": false,
		},
		{
			"level":              1,
			"name":               "light",
			"description":        "轻度验证，可能需要简单操作",
			"trust_score_min":    60,
			"risk_score_max":     40,
			"requires_challenge": true,
			"challenge_types":    []string{"slider", "simple_click"},
		},
		{
			"level":              2,
			"name":               "moderate",
			"description":        "中等验证，需要完成图形验证",
			"trust_score_min":    40,
			"risk_score_max":     60,
			"requires_challenge": true,
			"challenge_types":    []string{"slider", "image_select", "rotation"},
		},
		{
			"level":              3,
			"name":               "strict",
			"description":        "严格验证，需要完成多重验证",
			"trust_score_min":    0,
			"risk_score_max":     100,
			"requires_challenge": true,
			"challenge_types":    []string{"slider", "image_select", "rotation", "biometric"},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    levels,
	})
}

func (h *ProgressiveVerificationHandler) RecordVerificationResult(c *gin.Context) {
	var req struct {
		Fingerprint string  `json:"fingerprint" binding:"required"`
		SessionID   string  `json:"session_id"`
		Passed      bool    `json:"passed"`
		Level       int     `json:"level"`
		Duration    int64   `json:"duration"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	event := "verify"
	if req.Passed {
		event = "login_success"
	}

	h.trustService.UpdateTrustScore(req.Fingerprint, event, c.ClientIP(), c.GetHeader("User-Agent"))

	if req.Passed {
		duration := 24 * time.Hour
		if req.Level >= 3 {
			duration = 24 * 7 * 24 * time.Hour
		}
		h.trustService.MarkAsVerified(req.Fingerprint, duration)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "verification result recorded",
	})
}

func generateProgressiveSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "pvs_" + hex.EncodeToString(bytes)
}

func generateChallenges(level int) []Challenge {
	challenges := make([]Challenge, 0)

	switch level {
	case 0:
		return challenges
	case 1:
		challenges = append(challenges, Challenge{
			Type:      "simple_interaction",
			Token:     generateChallengeToken(),
			Completed: false,
		})
	case 2:
		challenges = append(challenges, Challenge{
			Type:      "slider_captcha",
			Token:     generateChallengeToken(),
			Completed: false,
		}, Challenge{
			Type:      "image_select",
			Token:     generateChallengeToken(),
			Completed: false,
		})
	case 3:
		challenges = append(challenges, Challenge{
			Type:      "slider_captcha",
			Token:     generateChallengeToken(),
			Completed: false,
		}, Challenge{
			Type:      "image_select",
			Token:     generateChallengeToken(),
			Completed: false,
		}, Challenge{
			Type:      "rotation_puzzle",
			Token:     generateChallengeToken(),
			Completed: false,
		})
	}

	return challenges
}

func generateChallengeToken() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "ch_" + hex.EncodeToString(bytes)
}

func getLevelName(level int) string {
	names := map[int]string{
		0: "silent",
		1: "light",
		2: "moderate",
		3: "strict",
	}
	if name, ok := names[level]; ok {
		return name
	}
	return "unknown"
}

func getProgressiveLevel(trustScore int, riskScore float64, isNewDevice bool) (int, string) {
	if trustScore >= 80 && riskScore < 20 && !isNewDevice {
		return 0, "silent"
	}
	if trustScore >= 60 && riskScore < 40 {
		return 1, "light"
	}
	if trustScore >= 40 {
		return 2, "moderate"
	}
	return 3, "strict"
}

func getVerificationMessage(level int, trustScore int, riskScore float64) string {
	switch level {
	case 0:
		return "验证通过，无需额外操作"
	case 1:
		return fmt.Sprintf("需要完成轻度验证 (信任分: %d, 风险分: %.1f)", trustScore, riskScore)
	case 2:
		return fmt.Sprintf("需要完成中等验证 (信任分: %d, 风险分: %.1f)", trustScore, riskScore)
	case 3:
		return fmt.Sprintf("需要完成严格验证 (信任分: %d, 风险分: %.1f)", trustScore, riskScore)
	default:
		return "验证级别未知"
	}
}

func getChallengeMessage(passed bool) string {
	if passed {
		return "挑战完成，验证通过"
	}
	return "挑战未完成"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
