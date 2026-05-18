package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

type AIRecommendationHandler struct {
	recommendationService *service.AIRecommendationService
}

func NewAIRecommendationHandler() *AIRecommendationHandler {
	return &AIRecommendationHandler{
		recommendationService: service.NewAIRecommendationService(),
	}
}

type RecommendCaptchaRequest struct {
	UserID          string                     `json:"user_id"`
	Fingerprint     string                     `json:"fingerprint"`
	IPAddress       string                     `json:"ip_address"`
	SessionID       string                     `json:"session_id"`
	ApplicationID   uint                       `json:"application_id"`
	RiskScore       float64                    `json:"risk_score"`
	AccessFrequency float64                    `json:"access_frequency"`
	DeviceTrust     float64                    `json:"device_trust"`
	TimeOfDay       int                        `json:"time_of_day"`
	EnvInfo         *EnvInfoRequest            `json:"env_info"`
	BehaviorData    []BehaviorPointRequest     `json:"behavior_data"`
}

type EnvInfoRequest struct {
	UserAgent            string   `json:"user_agent"`
	Platform             string   `json:"platform"`
	Language             string   `json:"language"`
	Languages            []string `json:"languages"`
	ScreenWidth          int      `json:"screen_width"`
	ScreenHeight         int      `json:"screen_height"`
	ColorDepth           int      `json:"color_depth"`
	PixelRatio           float64  `json:"pixel_ratio"`
	Timezone             string   `json:"timezone"`
	TimezoneOffset       int      `json:"timezone_offset"`
	CanvasFingerprint    string   `json:"canvas_fingerprint"`
	WebGLRenderer        string   `json:"webgl_renderer"`
	WebGLVendor          string   `json:"webgl_vendor"`
	Plugins              []string `json:"plugins"`
	Fonts                []string `json:"fonts"`
	TouchSupport         bool     `json:"touch_support"`
	MaxTouchPoints       int      `json:"max_touch_points"`
	HardwareConcurrency   int      `json:"hardware_concurrency"`
	Fingerprint          string   `json:"fingerprint"`
}

type BehaviorPointRequest struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Timestamp int64 `json:"timestamp"`
	Event     string `json:"event"`
}

type RecommendCaptchaResponse struct {
	Success           bool                         `json:"success"`
	Data              *CaptchaRecommendationResponse `json:"data,omitempty"`
	Error             string                       `json:"error,omitempty"`
	RecommendedType   string                       `json:"recommended_type"`
	Confidence        float64                      `json:"confidence"`
	Alternatives      []AlternativeResponse        `json:"alternatives"`
	Difficulty        *DifficultyResponse          `json:"difficulty"`
	Factors           []FactorResponse             `json:"factors"`
	EstimatedDuration int64                        `json:"estimated_duration_ms"`
	Reason            string                       `json:"reason"`
}

type CaptchaRecommendationResponse struct {
	RecommendedType   string                `json:"recommended_type"`
	Confidence        float64               `json:"confidence"`
	Alternatives      []AlternativeResponse `json:"alternatives"`
	Difficulty        *DifficultyResponse   `json:"difficulty"`
	Factors           []FactorResponse      `json:"factors"`
	EstimatedDuration int64                 `json:"estimated_duration_ms"`
	Reason           string                `json:"reason"`
}

type AlternativeResponse struct {
	Type   string  `json:"type"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

type DifficultyResponse struct {
	Level         string `json:"level"`
	Score        float64 `json:"score"`
	SliderOffset  int    `json:"slider_offset,omitempty"`
	JigsawPieces  int    `json:"jigsaw_pieces,omitempty"`
	ClickCount    int    `json:"click_count,omitempty"`
	TimeLimit     int    `json:"time_limit_seconds"`
}

type FactorResponse struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
	Impact float64 `json:"impact"`
	Score  float64 `json:"score"`
}

type RecommendDifficultyRequest struct {
	UserID       string  `json:"user_id"`
	Fingerprint  string  `json:"fingerprint"`
	CaptchaType  string  `json:"captcha_type"`
	RiskScore    float64 `json:"risk_score"`
	TimeOfDay    int     `json:"time_of_day"`
	SuccessRate  float64 `json:"success_rate"`
	AvgDuration  int64   `json:"avg_duration_ms"`
	FailureCount int     `json:"failure_count"`
}

type RecommendDifficultyResponse struct {
	Success           bool                       `json:"success"`
	Data              *DifficultyDetail          `json:"data,omitempty"`
	Error             string                     `json:"error,omitempty"`
	RecommendedLevel  string                     `json:"recommended_level"`
	Difficulty        *DifficultyResponse        `json:"difficulty"`
	Confidence        float64                    `json:"confidence"`
	Factors           []FactorDetailResponse    `json:"factors"`
	AdjustmentReason  string                     `json:"adjustment_reason"`
}

type DifficultyDetail struct {
	RecommendedLevel string                `json:"recommended_level"`
	Difficulty      *DifficultyResponse  `json:"difficulty"`
	Confidence      float64              `json:"confidence"`
	Factors         []FactorDetailResponse `json:"factors"`
	AdjustmentReason string              `json:"adjustment_reason"`
}

type FactorDetailResponse struct {
	Name     string  `json:"name"`
	Impact   float64 `json:"impact"`
	NewValue float64 `json:"new_value"`
	OldValue float64 `json:"old_value"`
}

type UpdateHistoryRequest struct {
	UserID      string `json:"user_id" binding:"required"`
	Fingerprint string `json:"fingerprint"`
	CaptchaType string `json:"captcha_type" binding:"required"`
	Success     bool   `json:"success"`
	Duration    int64  `json:"duration"`
}

func (h *AIRecommendationHandler) GetCaptchaRecommendation(c *gin.Context) {
	var req RecommendCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request: " + err.Error(),
		})
		return
	}

	if req.RiskScore < 0 || req.RiskScore > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "risk_score must be between 0 and 100",
		})
		return
	}

	if req.TimeOfDay < 0 {
		req.TimeOfDay = time.Now().Hour()
	}

	var envInfo *service.EnvInfo
	if req.EnvInfo != nil {
		envInfo = &service.EnvInfo{
			UserAgent:            req.EnvInfo.UserAgent,
			Platform:             req.EnvInfo.Platform,
			Language:             req.EnvInfo.Language,
			Languages:            req.EnvInfo.Languages,
			ScreenWidth:          req.EnvInfo.ScreenWidth,
			ScreenHeight:         req.EnvInfo.ScreenHeight,
			ColorDepth:           req.EnvInfo.ColorDepth,
			PixelRatio:           req.EnvInfo.PixelRatio,
			Timezone:             req.EnvInfo.Timezone,
			TimezoneOffset:       req.EnvInfo.TimezoneOffset,
			CanvasFingerprint:    req.EnvInfo.CanvasFingerprint,
			WebGLRenderer:        req.EnvInfo.WebGLRenderer,
			WebGLVendor:         req.EnvInfo.WebGLVendor,
			Plugins:              req.EnvInfo.Plugins,
			Fonts:                req.EnvInfo.Fonts,
			TouchSupport:         req.EnvInfo.TouchSupport,
			MaxTouchPoints:       req.EnvInfo.MaxTouchPoints,
			HardwareConcurrency:   req.EnvInfo.HardwareConcurrency,
			Fingerprint:          req.EnvInfo.Fingerprint,
		}
	}

	var behaviorData []service.BehaviorDataPoint
	for _, point := range req.BehaviorData {
		behaviorData = append(behaviorData, service.BehaviorDataPoint{
			X:         point.X,
			Y:         point.Y,
			Timestamp: point.Timestamp,
			Event:     point.Event,
		})
	}

	recommendationReq := &service.CaptchaRecommendationRequest{
		UserID:          req.UserID,
		Fingerprint:     req.Fingerprint,
		IPAddress:       req.IPAddress,
		SessionID:       req.SessionID,
		ApplicationID:   req.ApplicationID,
		EnvInfo:        envInfo,
		RiskScore:       req.RiskScore,
		TimeOfDay:       req.TimeOfDay,
		AccessFrequency: req.AccessFrequency,
		DeviceTrust:     req.DeviceTrust,
		BehaviorData:    behaviorData,
	}

	recommendation, err := h.recommendationService.GetRecommendation(c.Request.Context(), recommendationReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to get recommendation: " + err.Error(),
		})
		return
	}

	response := h.buildRecommendResponse(recommendation)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func (h *AIRecommendationHandler) GetDifficultyRecommendation(c *gin.Context) {
	var req RecommendDifficultyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request: " + err.Error(),
		})
		return
	}

	if req.RiskScore < 0 || req.RiskScore > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "risk_score must be between 0 and 100",
		})
		return
	}

	captchaType := service.CaptchaType(req.CaptchaType)
	if !isValidCaptchaType(captchaType) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid captcha_type",
		})
		return
	}

	if req.TimeOfDay < 0 {
		req.TimeOfDay = time.Now().Hour()
	}

	difficultyReq := &service.DifficultyRequest{
		UserID:       req.UserID,
		Fingerprint:  req.Fingerprint,
		CaptchaType:  captchaType,
		RiskScore:    req.RiskScore,
		TimeOfDay:    req.TimeOfDay,
		SuccessRate:  req.SuccessRate,
		AvgDuration:  req.AvgDuration,
		FailureCount: req.FailureCount,
	}

	difficultyResp, err := h.recommendationService.GetDifficultyRecommendation(c.Request.Context(), difficultyReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to get difficulty recommendation: " + err.Error(),
		})
		return
	}

	response := h.buildDifficultyResponse(difficultyResp)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func (h *AIRecommendationHandler) UpdateUserHistory(c *gin.Context) {
	var req UpdateHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request: " + err.Error(),
		})
		return
	}

	captchaType := service.CaptchaType(req.CaptchaType)
	if !isValidCaptchaType(captchaType) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid captcha_type",
		})
		return
	}

	clientIP := c.ClientIP()
	h.recommendationService.UpdateUserHistory(
		req.UserID,
		req.Fingerprint,
		clientIP,
		captchaType,
		req.Success,
		req.Duration,
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "user history updated successfully",
	})
}

func (h *AIRecommendationHandler) GetCaptchaTypeStats(c *gin.Context) {
	stats := h.recommendationService.GetCaptchaTypeStats()

	typeStats := make([]map[string]interface{}, 0)
	for _, stat := range stats {
		typeStats = append(typeStats, map[string]interface{}{
			"type":           stat.Type,
			"success_rate":   stat.SuccessRate,
			"avg_duration":   stat.AvgDuration,
			"total_attempts": stat.TotalAttempts,
			"failure_rate":   stat.FailureRate,
			"comfort_score":  stat.ComfortScore,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    typeStats,
	})
}

func (h *AIRecommendationHandler) GetUserStats(c *gin.Context) {
	userID := c.Query("user_id")
	fingerprint := c.Query("fingerprint")

	if userID == "" && fingerprint == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "user_id or fingerprint is required",
		})
		return
	}

	history := h.recommendationService.GetUserStats(userID, fingerprint)
	if history == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    nil,
			"message": "no history found",
		})
		return
	}

	preferredTypes := make(map[string]int)
	for t, count := range history.PreferredTypes {
		preferredTypes[string(t)] = count
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"user_id":           history.UserID,
			"fingerprint":       history.Fingerprint,
			"total_attempts":    history.TotalAttempts,
			"success_count":     history.SuccessCount,
			"failure_count":     history.FailureCount,
			"success_rate":      history.SuccessRate,
			"avg_duration":      history.AvgDuration,
			"preferred_types":   preferredTypes,
			"last_captcha_type": string(history.LastCaptchaType),
			"last_verified_at": history.LastVerifiedAt,
			"created_at":        history.CreatedAt,
		},
	})
}

func (h *AIRecommendationHandler) buildRecommendResponse(rec *service.CaptchaRecommendation) *RecommendCaptchaResponse {
	alternatives := make([]AlternativeResponse, len(rec.Alternatives))
	for i, alt := range rec.Alternatives {
		alternatives[i] = AlternativeResponse{
			Type:   string(alt.Type),
			Score:  alt.Score,
			Reason: alt.Reason,
		}
	}

	factors := make([]FactorResponse, len(rec.Factors))
	for i, f := range rec.Factors {
		factors[i] = FactorResponse{
			Name:   f.Name,
			Weight: f.Weight,
			Impact: f.Impact,
			Score:  f.Score,
		}
	}

	return &RecommendCaptchaResponse{
		Success:           true,
		RecommendedType:   string(rec.RecommendedType),
		Confidence:        rec.Confidence,
		Alternatives:      alternatives,
		Difficulty: &DifficultyResponse{
			Level:        rec.Difficulty.Level,
			Score:        rec.Difficulty.Score,
			SliderOffset: rec.Difficulty.SliderOffset,
			JigsawPieces: rec.Difficulty.JigsawPieces,
			ClickCount:   rec.Difficulty.ClickCount,
			TimeLimit:    rec.Difficulty.TimeLimit,
		},
		Factors:           factors,
		EstimatedDuration: rec.EstimatedDuration,
		Reason:            rec.Reason,
	}
}

func (h *AIRecommendationHandler) buildDifficultyResponse(resp *service.DifficultyResponse) *RecommendDifficultyResponse {
	factors := make([]FactorDetailResponse, len(resp.Factors))
	for i, f := range resp.Factors {
		factors[i] = FactorDetailResponse{
			Name:     f.Name,
			Impact:   f.Impact,
			NewValue: f.NewValue,
			OldValue: f.OldValue,
		}
	}

	return &RecommendDifficultyResponse{
		Success:           true,
		RecommendedLevel:  resp.RecommendedLevel,
		Difficulty: &DifficultyResponse{
			Level:        resp.Difficulty.Level,
			Score:        resp.Difficulty.Score,
			SliderOffset: resp.Difficulty.SliderOffset,
			JigsawPieces: resp.Difficulty.JigsawPieces,
			ClickCount:   resp.Difficulty.ClickCount,
			TimeLimit:    resp.Difficulty.TimeLimit,
		},
		Confidence:       resp.Confidence,
		Factors:          factors,
		AdjustmentReason: resp.AdjustmentReason,
	}
}

func isValidCaptchaType(captchaType service.CaptchaType) bool {
	validTypes := []service.CaptchaType{
		service.CaptchaTypeSlider,
		service.CaptchaTypeClick,
		service.CaptchaTypeGesture,
		service.CaptchaTypeLianLianKan,
		service.CaptchaTypeVoice,
		service.CaptchaType3D,
		service.CaptchaTypeSeamless,
	}

	for _, t := range validTypes {
		if t == captchaType {
			return true
		}
	}
	return false
}

func (h *AIRecommendationHandler) GetRecommendationByQuery(c *gin.Context) {
	userID := c.Query("user_id")
	fingerprint := c.Query("fingerprint")
	riskScoreStr := c.DefaultQuery("risk_score", "50")
	captchaType := c.DefaultQuery("captcha_type", "")

	riskScore, err := strconv.ParseFloat(riskScoreStr, 64)
	if err != nil || riskScore < 0 || riskScore > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid risk_score",
		})
		return
	}

	if captchaType != "" {
		req := RecommendDifficultyRequest{
			UserID:      userID,
			Fingerprint: fingerprint,
			CaptchaType: captchaType,
			RiskScore:   riskScore,
			TimeOfDay:   time.Now().Hour(),
		}

		difficultyReq := &service.DifficultyRequest{
			UserID:       req.UserID,
			Fingerprint:  req.Fingerprint,
			CaptchaType:  service.CaptchaType(req.CaptchaType),
			RiskScore:    req.RiskScore,
			TimeOfDay:    req.TimeOfDay,
			SuccessRate:  req.SuccessRate,
			AvgDuration:  req.AvgDuration,
			FailureCount: req.FailureCount,
		}

		difficultyResp, err := h.recommendationService.GetDifficultyRecommendation(c.Request.Context(), difficultyReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "failed to get difficulty recommendation",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":           true,
			"recommended_level": difficultyResp.RecommendedLevel,
			"difficulty": &DifficultyResponse{
				Level:        difficultyResp.Difficulty.Level,
				Score:        difficultyResp.Difficulty.Score,
				SliderOffset: difficultyResp.Difficulty.SliderOffset,
				JigsawPieces: difficultyResp.Difficulty.JigsawPieces,
				ClickCount:   difficultyResp.Difficulty.ClickCount,
				TimeLimit:    difficultyResp.Difficulty.TimeLimit,
			},
			"confidence":        difficultyResp.Confidence,
			"adjustment_reason": difficultyResp.AdjustmentReason,
		})
		return
	}

	req := &service.CaptchaRecommendationRequest{
		UserID:      userID,
		Fingerprint: fingerprint,
		RiskScore:   riskScore,
		TimeOfDay:   time.Now().Hour(),
		IPAddress:   c.ClientIP(),
	}

	recommendation, err := h.recommendationService.GetRecommendation(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to get recommendation",
		})
		return
	}

	response := h.buildRecommendResponse(recommendation)
	c.JSON(http.StatusOK, response)
}
