package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type FingerprintHandler struct {
	service *service.DeviceFingerprintService
}

func NewFingerprintHandler() *FingerprintHandler {
	return &FingerprintHandler{
		service: service.NewDeviceFingerprintService(),
	}
}

type CollectFingerprintRequest struct {
	Fingerprint service.FingerprintData `json:"fingerprint" binding:"required"`
}

type CollectFingerprintResponse struct {
	FingerprintID uint   `json:"fingerprint_id"`
	Hash          string `json:"hash"`
	RiskLevel     string `json:"risk_level"`
}

type VerifyFingerprintRequest struct {
	FingerprintID uint   `json:"fingerprint_id" binding:"required"`
	Hash          string `json:"hash" binding:"required"`
}

type VerifyFingerprintResponse struct {
	Valid           bool                        `json:"valid"`
	RiskLevel       string                      `json:"risk_level"`
	RiskScore       float64                     `json:"risk_score"`
	RiskFactors     []string                    `json:"risk_factors"`
	IsNewDevice     bool                        `json:"is_new_device"`
	SimilarDevices  []service.SimilarDevice     `json:"similar_devices"`
}

type DeviceListResponse struct {
	Devices []service.DeviceInfo `json:"devices"`
}

type TrustDeviceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type DeviceHistoryResponse struct {
	History []map[string]interface{} `json:"history"`
}

type AnomalyResponse struct {
	Anomalies []string `json:"anomalies"`
}

type ExportResponse struct {
	Data string `json:"data"`
}

func (h *FingerprintHandler) CollectFingerprint(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	var req CollectFingerprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	ipAddress := c.ClientIP()

	result, err := h.service.CollectFingerprint(userID.(uint), req.Fingerprint, ipAddress)
	if err != nil {
		response.InternalServerError(c, "failed to collect fingerprint")
		return
	}

	response.Success(c, CollectFingerprintResponse{
		FingerprintID: result.FingerprintID,
		Hash:          result.Hash,
		RiskLevel:     result.RiskLevel,
	})
}

func (h *FingerprintHandler) VerifyFingerprint(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	var req VerifyFingerprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	valid, assessment, similarDevices, err := h.service.VerifyFingerprint(
		userID.(uint),
		req.FingerprintID,
		req.Hash,
	)
	if err != nil {
		response.InternalServerError(c, "failed to verify fingerprint")
		return
	}

	response.Success(c, VerifyFingerprintResponse{
		Valid:          valid,
		RiskLevel:      assessment.Level,
		RiskScore:      assessment.Score,
		RiskFactors:    assessment.Factors,
		IsNewDevice:    assessment.IsNewDevice,
		SimilarDevices: similarDevices,
	})
}

func (h *FingerprintHandler) GetDevices(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	devices, err := h.service.GetUserDevices(userID.(uint))
	if err != nil {
		response.InternalServerError(c, "failed to get devices")
		return
	}

	response.Success(c, DeviceListResponse{
		Devices: devices,
	})
}

func (h *FingerprintHandler) TrustDevice(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	deviceIDStr := c.Param("id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return
	}

	err = h.service.TrustDevice(userID.(uint), uint(deviceID))
	if err != nil {
		response.InternalServerError(c, "failed to trust device")
		return
	}

	response.Success(c, TrustDeviceResponse{
		Success: true,
		Message: "device trusted successfully",
	})
}

func (h *FingerprintHandler) UntrustDevice(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	deviceIDStr := c.Param("id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return
	}

	err = h.service.UntrustDevice(userID.(uint), uint(deviceID))
	if err != nil {
		response.InternalServerError(c, "failed to untrust device")
		return
	}

	response.Success(c, TrustDeviceResponse{
		Success: true,
		Message: "device untrusted successfully",
	})
}

func (h *FingerprintHandler) GetDeviceHistory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	deviceIDStr := c.Param("id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	history, err := h.service.GetDeviceHistory(uint(deviceID), limit)
	if err != nil {
		response.InternalServerError(c, "failed to get device history")
		return
	}

	var historyMaps []map[string]interface{}
	for _, h := range history {
		historyMaps = append(historyMaps, map[string]interface{}{
			"id":            h.ID,
			"ip_address":    h.IPAddress,
			"location":      h.Location,
			"login_time":    h.LoginTime,
			"login_success": h.LoginSuccess,
			"user_agent":    h.UserAgent,
		})
	}

	_ = userID

	response.Success(c, DeviceHistoryResponse{
		History: historyMaps,
	})
}

func (h *FingerprintHandler) GetAnomalies(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	anomalies, err := h.service.CheckDeviceAnomalies(userID.(uint))
	if err != nil {
		response.InternalServerError(c, "failed to check anomalies")
		return
	}

	response.Success(c, AnomalyResponse{
		Anomalies: anomalies,
	})
}

func (h *FingerprintHandler) ExportData(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	data, err := h.service.ExportFingerprintData(userID.(uint))
	if err != nil {
		response.InternalServerError(c, "failed to export data")
		return
	}

	response.Success(c, ExportResponse{
		Data: data,
	})
}

func (h *FingerprintHandler) DeleteData(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}

	err := h.service.AnonymizeFingerprintData(userID.(uint))
	if err != nil {
		response.InternalServerError(c, "failed to delete data")
		return
	}

	response.Success(c, gin.H{
		"success": true,
		"message": "data anonymized successfully",
	})
}

func RegisterFingerprintRoutes(router *gin.Engine) {
	handler := NewFingerprintHandler()

	fingerprint := router.Group("/api/v1/fingerprint")
	{
		fingerprint.POST("/collect", middleware.AuthMiddleware(), handler.CollectFingerprint)
		fingerprint.POST("/verify", middleware.AuthMiddleware(), handler.VerifyFingerprint)
		fingerprint.GET("/devices", middleware.AuthMiddleware(), handler.GetDevices)
		fingerprint.POST("/devices/:id/trust", middleware.AuthMiddleware(), handler.TrustDevice)
		fingerprint.POST("/devices/:id/untrust", middleware.AuthMiddleware(), handler.UntrustDevice)
		fingerprint.GET("/devices/:id/history", middleware.AuthMiddleware(), handler.GetDeviceHistory)
		fingerprint.GET("/anomalies", middleware.AuthMiddleware(), handler.GetAnomalies)
		fingerprint.GET("/export", middleware.AuthMiddleware(), handler.ExportData)
		fingerprint.DELETE("/data", middleware.AuthMiddleware(), handler.DeleteData)
	}
}
