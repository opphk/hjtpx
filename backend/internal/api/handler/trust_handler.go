package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

type WhitelistHandler struct {
	trustService *service.DeviceTrustService
}

type WhitelistRequest struct {
	Target       string     `json:"target" binding:"required"`
	Type         string     `json:"type" binding:"required"`
	Reason       string     `json:"reason,omitempty"`
	ApplicationID *uint     `json:"application_id,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

type WhitelistResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func GetWhitelistHandler() *WhitelistHandler {
	return &WhitelistHandler{
		trustService: service.GetDeviceTrustService(),
	}
}

func (h *WhitelistHandler) AddWhitelist(c *gin.Context) {
	var req WhitelistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, WhitelistResponse{
			Success: false,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	validTypes := map[string]bool{
		"fingerprint": true,
		"ip":         true,
		"user":       true,
		"application": true,
	}
	if !validTypes[req.Type] {
		c.JSON(http.StatusBadRequest, WhitelistResponse{
			Success: false,
			Message: "invalid type: must be fingerprint, ip, user, or application",
		})
		return
	}

	if req.Type == "fingerprint" {
		if len(req.Target) < 16 || len(req.Target) > 64 {
			c.JSON(http.StatusBadRequest, WhitelistResponse{
				Success: false,
				Message: "fingerprint must be 16-64 characters",
			})
			return
		}
		h.trustService.MarkAsVerified(req.Target, 365*24*time.Hour)
	}

	c.JSON(http.StatusOK, WhitelistResponse{
		Success: true,
		Message: fmt.Sprintf("successfully added %s to whitelist", req.Type),
		Data: map[string]interface{}{
			"target":       req.Target,
			"type":         req.Type,
			"reason":       req.Reason,
			"expires_at":   req.ExpiresAt,
			"created_at":   time.Now(),
		},
	})
}

func (h *WhitelistHandler) RemoveWhitelist(c *gin.Context) {
	target := c.Param("target")
	whitelistType := c.Query("type")

	if target == "" || whitelistType == "" {
		c.JSON(http.StatusBadRequest, WhitelistResponse{
			Success: false,
			Message: "target and type are required",
		})
		return
	}

	validTypes := map[string]bool{
		"fingerprint": true,
		"ip":         true,
		"user":       true,
		"application": true,
	}
	if !validTypes[whitelistType] {
		c.JSON(http.StatusBadRequest, WhitelistResponse{
			Success: false,
			Message: "invalid type",
		})
		return
	}

	if whitelistType == "fingerprint" {
		h.trustService.RemoveFingerprint(target)
	}

	c.JSON(http.StatusOK, WhitelistResponse{
		Success: true,
		Message: fmt.Sprintf("successfully removed %s from whitelist", whitelistType),
	})
}

func (h *WhitelistHandler) ListWhitelist(c *gin.Context) {
	whitelistType := c.DefaultQuery("type", "fingerprint")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	stats := h.trustService.GetStatistics()

	var items []map[string]interface{}
	if whitelistType == "fingerprint" {
		_ = stats["trust_distribution"].(map[string]int)
		for fp := range getFingerprintsFromCache() {
			trustInfo := h.trustService.GetTrustInfo(fp)
			if trustInfo != nil && trustInfo.IsVerified {
				items = append(items, map[string]interface{}{
					"target":       fp,
					"type":         "fingerprint",
					"trust_score":  trustInfo.TrustScore,
					"trust_level":  trustInfo.TrustLevel,
					"visit_count":  trustInfo.VisitCount,
					"last_visit":   trustInfo.LastVisit,
				})
			}
		}
	}

	total := len(items)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		items = []map[string]interface{}{}
	} else if end > total {
		items = items[start:]
	} else {
		items = items[start:end]
	}

	c.JSON(http.StatusOK, WhitelistResponse{
		Success: true,
		Data: map[string]interface{}{
			"items":      items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"stats":     stats,
		},
	})
}

func (h *WhitelistHandler) CheckWhitelist(c *gin.Context) {
	target := c.Query("target")
	whitelistType := c.Query("type")

	if target == "" {
		c.JSON(http.StatusBadRequest, WhitelistResponse{
			Success: false,
			Message: "target is required",
		})
		return
	}

	if whitelistType == "" {
		whitelistType = "fingerprint"
	}

	isWhitelisted := false
	details := map[string]interface{}{}

	switch whitelistType {
	case "fingerprint":
		info := h.trustService.GetTrustInfo(target)
		if info != nil {
			isWhitelisted = info.IsVerified
			details["trust_score"] = info.TrustScore
			details["trust_level"] = info.TrustLevel
			details["verified"] = info.IsVerified
			if info.ExpiresAt != nil {
				details["expires_at"] = info.ExpiresAt
			}
		}
	case "ip":
		isWhitelisted = false
		details["checked"] = true
	}

	c.JSON(http.StatusOK, WhitelistResponse{
		Success: true,
		Data: map[string]interface{}{
			"target":        target,
			"type":          whitelistType,
			"is_whitelisted": isWhitelisted,
			"details":       details,
		},
	})
}

func (h *WhitelistHandler) UpdateWhitelist(c *gin.Context) {
	target := c.Param("target")
	var req WhitelistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, WhitelistResponse{
			Success: false,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	if req.Type == "fingerprint" && req.ExpiresAt != nil {
		h.trustService.MarkAsVerified(target, time.Until(*req.ExpiresAt))
	}

	c.JSON(http.StatusOK, WhitelistResponse{
		Success: true,
		Message: "whitelist entry updated",
		Data: map[string]interface{}{
			"target":     target,
			"type":       req.Type,
			"reason":     req.Reason,
			"expires_at": req.ExpiresAt,
		},
	})
}

func (h *WhitelistHandler) GetWhitelistStats(c *gin.Context) {
	stats := h.trustService.GetStatistics()

	c.JSON(http.StatusOK, WhitelistResponse{
		Success: true,
		Data:    stats,
	})
}

func (h *WhitelistHandler) BulkAddWhitelist(c *gin.Context) {
	var requests struct {
		Entries []WhitelistRequest `json:"entries" binding:"required"`
	}
	if err := c.ShouldBindJSON(&requests); err != nil {
		c.JSON(http.StatusBadRequest, WhitelistResponse{
			Success: false,
			Message: "invalid request: " + err.Error(),
		})
		return
	}

	if len(requests.Entries) > 100 {
		c.JSON(http.StatusBadRequest, WhitelistResponse{
			Success: false,
			Message: "maximum 100 entries per request",
		})
		return
	}

	results := make([]map[string]interface{}, 0, len(requests.Entries))
	successCount := 0
	failCount := 0

	for _, entry := range requests.Entries {
		result := map[string]interface{}{
			"target": entry.Target,
			"type":   entry.Type,
			"status": "failed",
		}

		if entry.Type == "fingerprint" {
			h.trustService.MarkAsVerified(entry.Target, 365*24*time.Hour)
			result["status"] = "success"
			successCount++
		} else {
			result["error"] = "unsupported type"
			failCount++
		}

		results = append(results, result)
	}

	c.JSON(http.StatusOK, WhitelistResponse{
		Success: true,
		Message: fmt.Sprintf("processed %d entries", len(requests.Entries)),
		Data: map[string]interface{}{
			"total":        len(requests.Entries),
			"success":      successCount,
			"failed":       failCount,
			"results":      results,
		},
	})
}

func (h *WhitelistHandler) ExportWhitelist(c *gin.Context) {
	whitelistType := c.DefaultQuery("type", "fingerprint")

	var data []map[string]interface{}

	if whitelistType == "fingerprint" {
		for fp := range getFingerprintsFromCache() {
			info := h.trustService.GetTrustInfo(fp)
			if info != nil && info.IsVerified {
				data = append(data, map[string]interface{}{
					"target":          fp,
					"type":            "fingerprint",
					"trust_score":     info.TrustScore,
					"trust_level":     info.TrustLevel,
					"visit_count":     info.VisitCount,
					"success_count":   info.SuccessCount,
					"first_visit":     info.FirstVisit,
					"last_visit":      info.LastVisit,
				})
			}
		}
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=whitelist_%s_%s.json",
		whitelistType, time.Now().Format("20060102_150405")))
	c.Data(http.StatusOK, "application/json", jsonData)
}

func getFingerprintsFromCache() map[string]bool {
	fps := make(map[string]bool)
	stats := service.GetDeviceTrustService().GetStatistics()
	if stats != nil {
		_ = stats
	}
	return fps
}

type TrustHandler struct {
	trustService *service.DeviceTrustService
}

type TrustEvaluateRequest struct {
	Fingerprint string                 `json:"fingerprint" binding:"required"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

type TrustVerifyRequest struct {
	Fingerprint string `json:"fingerprint" binding:"required"`
	Token       string `json:"token,omitempty"`
	Duration    int    `json:"duration,omitempty"`
}

func GetTrustHandler() *TrustHandler {
	return &TrustHandler{
		trustService: service.GetDeviceTrustService(),
	}
}

func (h *TrustHandler) EvaluateTrust(c *gin.Context) {
	var req TrustEvaluateRequest
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

	decision := h.trustService.EvaluateTrust(req.Fingerprint, req.Data)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    decision,
	})
}

func (h *TrustHandler) GetTrustInfo(c *gin.Context) {
	fingerprint := c.Param("fingerprint")
	if fingerprint == "" {
		fingerprint = c.Query("fingerprint")
	}

	if fingerprint == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "fingerprint is required",
		})
		return
	}

	info := h.trustService.GetTrustInfo(fingerprint)
	if info == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "fingerprint not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    info,
	})
}

func (h *TrustHandler) VerifyDevice(c *gin.Context) {
	var req TrustVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	duration := time.Duration(req.Duration) * 24 * time.Hour
	if duration <= 0 {
		duration = 24 * 7 * time.Hour
	}

	h.trustService.MarkAsVerified(req.Fingerprint, duration)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "device verified successfully",
		"data": map[string]interface{}{
			"fingerprint": req.Fingerprint,
			"expires_at":  time.Now().Add(duration),
		},
	})
}

func (h *TrustHandler) RecordEvent(c *gin.Context) {
	var req struct {
		Fingerprint string `json:"fingerprint" binding:"required"`
		Event       string `json:"event" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	validEvents := map[string]bool{
		"login":            true,
		"login_success":    true,
		"login_failed":     true,
		"trust_increase":   true,
		"trust_decrease":   true,
		"verify":           true,
		"risk_detected":    true,
	}
	if !validEvents[req.Event] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid event type",
		})
		return
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")

	h.trustService.UpdateTrustScore(req.Fingerprint, req.Event, ip, ua)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "event recorded",
	})
}

func (h *TrustHandler) GetTrustHistory(c *gin.Context) {
	fingerprint := c.Param("fingerprint")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	history := h.trustService.GetTrustHistory(fingerprint, limit)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"fingerprint": fingerprint,
			"history":     history,
			"count":       len(history),
		},
	})
}

func (h *TrustHandler) AnalyzeFingerprint(c *gin.Context) {
	var req struct {
		Fingerprint string                 `json:"fingerprint" binding:"required"`
		Data        map[string]interface{} `json:"data,omitempty"`
	}

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

	analysis := h.trustService.AnalyzeFingerprint(req.Fingerprint, req.Data)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    analysis,
	})
}

func (h *TrustHandler) SetRiskScore(c *gin.Context) {
	var req struct {
		Fingerprint string    `json:"fingerprint" binding:"required"`
		RiskScore   float64   `json:"risk_score" binding:"required"`
		RiskFactors []string  `json:"risk_factors,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	if req.RiskScore < 0 || req.RiskScore > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "risk_score must be between 0 and 100",
		})
		return
	}

	h.trustService.SetRiskScore(req.Fingerprint, req.RiskScore, req.RiskFactors)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "risk score updated",
	})
}

func (h *TrustHandler) GetStatistics(c *gin.Context) {
	stats := h.trustService.GetStatistics()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

func (h *TrustHandler) BatchEvaluate(c *gin.Context) {
	var requests struct {
		Items []TrustEvaluateRequest `json:"items" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requests); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	if len(requests.Items) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "maximum 100 items per request",
		})
		return
	}

	results := make([]map[string]interface{}, 0, len(requests.Items))

	for _, item := range requests.Items {
		decision := h.trustService.EvaluateTrust(item.Fingerprint, item.Data)
		results = append(results, map[string]interface{}{
			"fingerprint": item.Fingerprint,
			"decision":    decision,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"total":   len(requests.Items),
			"results": results,
		},
	})
}

func shouldSkipVerification(fingerprint string, trustService *service.DeviceTrustService) (bool, string) {
	if fingerprint == "" {
		return false, "missing_fingerprint"
	}

	info := trustService.GetTrustInfo(fingerprint)
	if info == nil {
		return false, "new_device"
	}

	if info.IsVerified && info.ExpiresAt != nil {
		if time.Now().Before(*info.ExpiresAt) {
			return true, "verified_device"
		}
	}

	if info.TrustScore >= 90 && info.RiskScore < 10 {
		return true, "high_trust"
	}

	return false, ""
}

func getProgressiveVerificationLevel(trustScore int, riskScore float64, isNewDevice bool) (int, string) {
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

func determineVerificationAction(trustScore int, riskScore float64, verificationLevel int) string {
	switch {
	case trustScore >= 80 && riskScore < 20:
		return "allow"
	case trustScore >= 60 && riskScore < 40:
		return "allow_with_monitoring"
	case trustScore >= 40:
		return "challenge"
	default:
		return "block"
	}
}

func calculateTrustScoreFromRisk(riskScore float64, factors []string) int {
	baseScore := 100 - int(riskScore)

	if len(factors) > 0 {
		penalty := len(factors) * 5
		baseScore -= penalty
	}

	if baseScore < 0 {
		baseScore = 0
	}
	if baseScore > 100 {
		baseScore = 100
	}

	return baseScore
}

func parseRiskFactors(data string) []string {
	if data == "" {
		return []string{}
	}

	var factors []string
	if err := json.Unmarshal([]byte(data), &factors); err != nil {
		return strings.Split(data, ",")
	}
	return factors
}
