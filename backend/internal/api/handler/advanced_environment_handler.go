package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

type AdvancedEnvironmentHandler struct {
	detector *service.AdvancedEnvDetectorService
}

func NewAdvancedEnvironmentHandler() *AdvancedEnvironmentHandler {
	return &AdvancedEnvironmentHandler{
		detector: service.NewAdvancedEnvDetectorService(),
	}
}

type DetectEnvironmentRequest struct {
	DetectionID   string                 `json:"detection_id"`
	RiskScore     float64                `json:"risk_score"`
	RiskLevel     string                 `json:"risk_level"`
	AllDetections []string               `json:"all_detections"`
	Timestamp     int64                  `json:"timestamp"`
	ClientResults map[string]interface{} `json:"client_results"`
	Summary       *DetectionSummary      `json:"summary"`
	Fingerprint   string                 `json:"fingerprint"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
}

type DetectionSummary struct {
	TotalChecks       int                       `json:"total_checks"`
	HighRiskChecks    int                       `json:"high_risk_checks"`
	MediumRiskChecks  int                       `json:"medium_risk_checks"`
	LowRiskChecks     int                       `json:"low_risk_checks"`
	Categories        map[string]CategoryResult `json:"categories"`
}

type CategoryResult struct {
	Score      float64  `json:"score"`
	Detections []string `json:"detections"`
}

type DetectEnvironmentResponse struct {
	Success         bool                              `json:"success"`
	DetectionID     string                            `json:"detection_id"`
	RiskScore       float64                           `json:"risk_score"`
	RiskLevel       string                            `json:"risk_level"`
	IsBot           bool                              `json:"is_bot"`
	IsVPN           bool                              `json:"is_vpn"`
	IsProxy         bool                              `json:"is_proxy"`
	IsTor           bool                              `json:"is_tor"`
	IsVM            bool                              `json:"is_vm"`
	IsDarkWeb       bool                              `json:"is_dark_web"`
	Confidence      float64                           `json:"confidence"`
	Indicators      []string                          `json:"indicators"`
	WebGLAnalysis   *WebGLAnalysisData                `json:"webgl_analysis,omitempty"`
	TorAnalysis     *TorAnalysisData                  `json:"tor_analysis,omitempty"`
	VMAnalysis      *VMAnalysisData                   `json:"vm_analysis,omitempty"`
	Recommendations []string                           `json:"recommendations"`
	Timestamp       time.Time                         `json:"timestamp"`
}

type WebGLAnalysisData struct {
	IsSoftwareRenderer bool     `json:"is_software_renderer"`
	IsVMRenderer      bool     `json:"is_vm_renderer"`
	IsAnonymized      bool     `json:"is_anonymized"`
	Anomalies         []string `json:"anomalies"`
	Score             float64  `json:"score"`
}

type TorAnalysisData struct {
	IsTorNode       bool     `json:"is_tor_node"`
	IsTorExitNode   bool     `json:"is_tor_exit_node"`
	IsDarkWebAccess bool     `json:"is_dark_web_access"`
	ExitNodeCountry string   `json:"exit_node_country"`
	ExitNodeISP     string   `json:"exit_node_isp"`
	ExitNodeASN     string   `json:"exit_node_asn"`
	Indicators      []string `json:"indicators"`
	Score           float64  `json:"score"`
}

type VMAnalysisData struct {
	IsVM            bool     `json:"is_vm"`
	VMType          string   `json:"vm_type"`
	CPUDetected     bool     `json:"cpu_detected"`
	GPUDetected     bool     `json:"gpu_detected"`
	MemoryDetected  bool     `json:"memory_detected"`
	ProcessDetected bool     `json:"process_detected"`
	BiosDetected    bool     `json:"bios_detected"`
	Indicators      []string `json:"indicators"`
	Score           float64  `json:"score"`
}

type TorCheckResponse struct {
	Success       bool      `json:"success"`
	IsTor         bool      `json:"is_tor"`
	IsTorExitNode bool      `json:"is_tor_exit_node"`
	IPAddress     string    `json:"ip_address"`
	Country       string    `json:"country"`
	ISP           string    `json:"isp"`
	ASN           string    `json:"asn"`
	Hoting        bool      `json:"hosting"`
	Proxy         bool      `json:"proxy"`
	RiskLevel     string    `json:"risk_level"`
	Score         float64   `json:"score"`
	Confidence    float64   `json:"confidence"`
	Indicators    []string  `json:"indicators"`
	CheckedAt     time.Time `json:"checked_at"`
}

type EnvironmentStatsResponse struct {
	Success           bool      `json:"success"`
	TotalDetections   int64     `json:"total_detections"`
	HighRiskCount     int64     `json:"high_risk_count"`
	MediumRiskCount   int64     `json:"medium_risk_count"`
	LowRiskCount      int64     `json:"low_risk_count"`
	TorDetections     int64     `json:"tor_detections"`
	VMDetections      int64     `json:"vm_detections"`
	BotDetections     int64     `json:"bot_detections"`
	AvgRiskScore      float64   `json:"avg_risk_score"`
	TopIndicators     []string  `json:"top_indicators"`
	TopVMTypes        []string  `json:"top_vm_types"`
	TopCountries      []string  `json:"top_countries"`
	LastUpdated       time.Time `json:"last_updated"`
}

func (h *AdvancedEnvironmentHandler) DetectEnvironment(c *gin.Context) {
	var req DetectEnvironmentRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	if req.RiskScore < 0 || req.RiskScore > 100 {
		req.RiskScore = 50
	}

	clientIP := c.ClientIP()
	
	if req.IPAddress == "" {
		req.IPAddress = clientIP
	}

	headers := h.extractProxyHeaders(c)

	var ipAddress string
	if forwarded := headers["X-Forwarded-For"]; forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ipAddress = strings.TrimSpace(ips[0])
		}
	} else if realIP := headers["X-Real-IP"]; realIP != "" {
		ipAddress = realIP
	} else {
		ipAddress = clientIP
	}

	ctx := c.Request.Context()

	serviceReq := &service.AdvancedEnvDetectionRequest{
		DetectionID:   req.DetectionID,
		RiskScore:     req.RiskScore,
		RiskLevel:     req.RiskLevel,
		AllDetections: req.AllDetections,
		Timestamp:     req.Timestamp,
		ClientResults: req.ClientResults,
		Summary:       convertSummary(req.Summary),
		Fingerprint:   req.Fingerprint,
		IPAddress:     ipAddress,
		UserAgent:     req.UserAgent,
	}

	result, err := h.detector.DetectEnvironment(ctx, serviceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "environment detection failed: " + err.Error(),
		})
		return
	}

	response := h.convertToResponse(result)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func (h *AdvancedEnvironmentHandler) CheckTorNetwork(c *gin.Context) {
	clientIP := c.ClientIP()
	
	ip := c.DefaultQuery("ip", clientIP)
	
	if ip == "" {
		ip = clientIP
	}

	headers := h.extractProxyHeaders(c)
	
	if forwarded := headers["X-Forwarded-For"]; forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip = strings.TrimSpace(ips[0])
		}
	} else if realIP := headers["X-Real-IP"]; realIP != "" {
		ip = realIP
	}

	ctx := c.Request.Context()

	result, err := h.detector.CheckTorNetwork(ctx, ip)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "tor check failed: " + err.Error(),
		})
		return
	}

	response := &TorCheckResponse{
		Success:       result.Success,
		IsTor:         result.IsTor,
		IsTorExitNode: result.IsTorExitNode,
		IPAddress:     result.IPAddress,
		Country:       result.Country,
		ISP:           result.ISP,
		ASN:           result.ASN,
		Hoting:        result.Hosting,
		Proxy:         result.Proxy,
		RiskLevel:     result.RiskLevel,
		Score:         result.Score,
		Confidence:    result.Confidence,
		Indicators:    result.Indicators,
		CheckedAt:     result.CheckedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func (h *AdvancedEnvironmentHandler) GetEnvironmentStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": EnvironmentStatsResponse{
			Success:         true,
			TotalDetections: 0,
			HighRiskCount:   0,
			MediumRiskCount: 0,
			LowRiskCount:    0,
			TorDetections:   0,
			VMDetections:    0,
			BotDetections:   0,
			AvgRiskScore:    0,
			TopIndicators:   []string{},
			TopVMTypes:      []string{},
			TopCountries:    []string{},
			LastUpdated:     time.Now(),
		},
	})
}

func (h *AdvancedEnvironmentHandler) GetCachedDetection(c *gin.Context) {
	detectionID := c.Param("id")
	
	if detectionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "detection id is required",
		})
		return
	}

	result, found := h.detector.GetCachedResult(detectionID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "detection not found",
		})
		return
	}

	response := h.convertToResponse(result)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func (h *AdvancedEnvironmentHandler) extractProxyHeaders(c *gin.Context) map[string]string {
	headers := make(map[string]string)
	
	proxyHeaders := []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"Via",
		"X-ProxyChain",
		"Forwarded",
		"CF-Connecting-IP",
		"True-Client-IP",
	}
	
	for _, header := range proxyHeaders {
		if val := c.GetHeader(header); val != "" {
			headers[header] = val
		}
	}
	
	return headers
}

func (h *AdvancedEnvironmentHandler) convertToResponse(result *service.AdvancedEnvResult) *DetectEnvironmentResponse {
	response := &DetectEnvironmentResponse{
		Success:         true,
		DetectionID:     result.DetectionID,
		RiskScore:       result.RiskScore,
		RiskLevel:       result.RiskLevel,
		IsBot:           result.IsBot,
		IsVPN:           result.IsVPN,
		IsProxy:         result.IsProxy,
		IsTor:           result.IsTor,
		IsVM:            result.IsVM,
		IsDarkWeb:       result.IsDarkWeb,
		Confidence:      result.Confidence,
		Indicators:      result.Indicators,
		Recommendations: result.Recommendations,
		Timestamp:       result.Timestamp,
	}

	if result.WebGLAnalysis != nil {
		response.WebGLAnalysis = &WebGLAnalysisData{
			IsSoftwareRenderer: result.WebGLAnalysis.IsSoftwareRenderer,
			IsVMRenderer:      result.WebGLAnalysis.IsVMRenderer,
			IsAnonymized:      result.WebGLAnalysis.IsAnonymized,
			Anomalies:         result.WebGLAnalysis.Anomalies,
			Score:             result.WebGLAnalysis.Score,
		}
	}

	if result.TorAnalysis != nil {
		response.TorAnalysis = &TorAnalysisData{
			IsTorNode:       result.TorAnalysis.IsTorNode,
			IsTorExitNode:   result.TorAnalysis.IsTorExitNode,
			IsDarkWebAccess: result.TorAnalysis.IsDarkWebAccess,
			ExitNodeCountry: result.TorAnalysis.ExitNodeCountry,
			ExitNodeISP:     result.TorAnalysis.ExitNodeISP,
			ExitNodeASN:     result.TorAnalysis.ExitNodeASN,
			Indicators:      result.TorAnalysis.Indicators,
			Score:           result.TorAnalysis.Score,
		}
	}

	if result.VMAnalysis != nil {
		response.VMAnalysis = &VMAnalysisData{
			IsVM:            result.VMAnalysis.IsVM,
			VMType:          result.VMAnalysis.VMType,
			CPUDetected:     result.VMAnalysis.CPUDetected,
			GPUDetected:     result.VMAnalysis.GPUDetected,
			MemoryDetected:  result.VMAnalysis.MemoryDetected,
			ProcessDetected: result.VMAnalysis.ProcessDetected,
			BiosDetected:    result.VMAnalysis.BiosDetected,
			Indicators:      result.VMAnalysis.Indicators,
			Score:           result.VMAnalysis.Score,
		}
	}

	return response
}

func convertSummary(summary *DetectionSummary) *service.DetectionSummary {
	if summary == nil {
		return nil
	}

	serviceSummary := &service.DetectionSummary{
		TotalChecks:      summary.TotalChecks,
		HighRiskChecks:   summary.HighRiskChecks,
		MediumRiskChecks: summary.MediumRiskChecks,
		LowRiskChecks:    summary.LowRiskChecks,
		Categories:       make(map[string]service.CategoryResult),
	}

	if summary.Categories != nil {
		for k, v := range summary.Categories {
			serviceSummary.Categories[k] = service.CategoryResult{
				Score:      v.Score,
				Detections: v.Detections,
			}
		}
	}

	return serviceSummary
}

func (h *AdvancedEnvironmentHandler) RegisterRoutes(router *gin.RouterGroup) {
	environment := router.Group("/environment")
	{
		environment.POST("/detect", h.DetectEnvironment)
		environment.GET("/tor-check", h.CheckTorNetwork)
		environment.GET("/stats", h.GetEnvironmentStats)
		environment.GET("/detection/:id", h.GetCachedDetection)
	}
}
