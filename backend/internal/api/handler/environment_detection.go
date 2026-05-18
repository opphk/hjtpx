package handler

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

type EnvironmentDetectionHandler struct {
	fingerprintAnalyzer *service.FingerprintAnalyzer
	proxyDetector       *service.ProxyDetectionService
}

func NewEnvironmentDetectionHandler() *EnvironmentDetectionHandler {
	return &EnvironmentDetectionHandler{
		fingerprintAnalyzer: service.NewFingerprintAnalyzer(),
		proxyDetector:       service.NewProxyDetectionService(),
	}
}

type EnvironmentDetectionRequest struct {
	Fingerprint         string                 `json:"fingerprint" binding:"required"`
	CanvasHash          string                 `json:"canvas_hash"`
	WebGLHash           string                 `json:"webgl_hash"`
	AudioHash           string                 `json:"audio_hash"`
	FontHash            string                 `json:"font_hash"`
	PluginHash          string                 `json:"plugin_hash"`
	ScreenResolution    string                 `json:"screen_resolution"`
	Timezone            string                 `json:"timezone"`
	Language            string                 `json:"language"`
	Platform            string                 `json:"platform"`
	UserAgent           string                 `json:"user_agent"`
	IPAddress           string                 `json:"ip_address"`
	Headers             map[string]string      `json:"headers"`
	WebRTCIPs           []string               `json:"webrtc_ips"`
	ConnectionType      string                 `json:"connection_type"`
	HardwareConcurrency int                    `json:"hardware_concurrency"`
	DeviceMemory        float64                `json:"device_memory"`
	RiskScore           float64                `json:"risk_score"`
	DetectionResults    map[string]interface{} `json:"detection_results"`
}

type EnvironmentDetectionResponse struct {
	Success         bool                    `json:"success"`
	FingerprintID   string                  `json:"fingerprint_id"`
	RiskLevel       string                  `json:"risk_level"`
	RiskScore       float64                 `json:"risk_score"`
	IsBot           bool                    `json:"is_bot"`
	IsVPN           bool                    `json:"is_vpn"`
	IsProxy         bool                    `json:"is_proxy"`
	IsTor           bool                    `json:"is_tor"`
	Confidence      float64                 `json:"confidence"`
	Indicators      []string                `json:"indicators"`
	Analysis        *DetectionAnalysis      `json:"analysis,omitempty"`
	ProxyResult     *service.ProxyDetection `json:"proxy_result,omitempty"`
	Recommendations []string                `json:"recommendations"`
	Timestamp       time.Time               `json:"timestamp"`
}

type DetectionAnalysis struct {
	AnomalyScore   float64                  `json:"anomaly_score"`
	IsAnomaly      bool                     `json:"is_anomaly"`
	AnomalyType    string                   `json:"anomaly_type"`
	Severity       string                   `json:"severity"`
	SimilarFingers []SimilarFingerprintInfo `json:"similar_fingerprints,omitempty"`
	ClusterInfo    *service.ClusterInfo     `json:"cluster_info,omitempty"`
}

type SimilarFingerprintInfo struct {
	FingerprintID string   `json:"fingerprint_id"`
	Similarity    float64  `json:"similarity"`
	CommonFields  []string `json:"common_fields"`
	DiffFields    []string `json:"diff_fields"`
}

type FingerprintAnalysisRequest struct {
	Fingerprint string  `form:"fingerprint" binding:"required"`
	Threshold   float64 `form:"threshold"`
}

type FingerprintAnalysisResponse struct {
	Success bool                     `json:"success"`
	Data    *FingerprintDataResponse `json:"data"`
}

type FingerprintDataResponse struct {
	Fingerprint   *service.FingerprintAnalysis `json:"fingerprint"`
	AnomalyResult *service.AnomalyResult       `json:"anomaly"`
	SimilarFps    []SimilarFingerprintInfo     `json:"similar_fingerprints"`
	Stats         *service.AnalysisStats       `json:"stats"`
	Clusters      []*service.ClusterInfo       `json:"clusters"`
}

type ProxyCheckRequest struct {
	IPAddress string `json:"ip_address" binding:"required"`
}

type ProxyCheckResponse struct {
	Success bool            `json:"success"`
	Data    *ProxyCheckData `json:"data"`
}

type ProxyCheckData struct {
	IPAddress        string    `json:"ip_address"`
	IsProxy          bool      `json:"is_proxy"`
	IsVPN            bool      `json:"is_vpn"`
	IsTor            bool      `json:"is_tor"`
	IsDataCenter     bool      `json:"is_data_center"`
	RiskLevel        string    `json:"risk_level"`
	Score            float64   `json:"score"`
	Confidence       float64   `json:"confidence"`
	Country          string    `json:"country"`
	ISP              string    `json:"isp"`
	ASN              string    `json:"asn"`
	DetectionMethods []string  `json:"detection_methods"`
	Hosing           bool      `json:"hosting"`
	Mobile           bool      `json:"mobile"`
	LastChecked      time.Time `json:"last_checked"`
}

func (h *EnvironmentDetectionHandler) DetectEnvironment(c *gin.Context) {
	var req EnvironmentDetectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request: " + err.Error(),
		})
		return
	}

	if req.RiskScore < 0 || req.RiskScore > 100 || math.IsNaN(req.RiskScore) || math.IsInf(req.RiskScore, 0) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid risk_score: must be between 0 and 100",
		})
		return
	}

	clientIP := c.ClientIP()

	headers := make(map[string]string)
	for _, header := range []string{
		"X-Forwarded-For", "X-Real-IP", "Via", "X-ProxyChain",
		"Forwarded", "CF-Connecting-IP", "True-Client-IP",
	} {
		if val := c.GetHeader(header); val != "" {
			headers[header] = val
		}
	}

	if req.IPAddress == "" {
		req.IPAddress = clientIP
	}

	analysisData := map[string]interface{}{
		"canvas_hash":          req.CanvasHash,
		"webgl_hash":           req.WebGLHash,
		"audio_hash":           req.AudioHash,
		"font_hash":            req.FontHash,
		"plugin_hash":          req.PluginHash,
		"screen_resolution":    req.ScreenResolution,
		"timezone":             req.Timezone,
		"language":             req.Language,
		"platform":             req.Platform,
		"user_agent":           req.UserAgent,
		"webrtc_ips":           req.WebRTCIPs,
		"connection_type":      req.ConnectionType,
		"hardware_concurrency": req.HardwareConcurrency,
		"device_memory":        req.DeviceMemory,
	}

	fpAnalysis, anomalyResult, err := h.fingerprintAnalyzer.AnalyzeFingerprint(analysisData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "fingerprint analysis failed",
		})
		return
	}

	proxyResult, err := h.proxyDetector.DetectProxy(req.IPAddress, headers)
	if err != nil {
		proxyResult = &service.ProxyDetection{
			IPAddress:  req.IPAddress,
			IsProxy:    false,
			IsVPN:      false,
			IsTor:      false,
			RiskLevel:  "unknown",
			Score:      0,
			Confidence: 0,
		}
	}

	riskScore := calculateCombinedRiskScore(req.RiskScore, fpAnalysis, anomalyResult, proxyResult)

	response := &EnvironmentDetectionResponse{
		Success:         true,
		FingerprintID:   fpAnalysis.FingerprintID,
		RiskScore:       riskScore,
		Confidence:      fpAnalysis.Confidence,
		Indicators:      fpAnalysis.RiskIndicators,
		Recommendations: generateRecommendations(riskScore, proxyResult, anomalyResult),
		Timestamp:       time.Now(),
	}

	if riskScore > 70 {
		response.RiskLevel = "high"
		response.IsBot = true
	} else if riskScore > 40 {
		response.RiskLevel = "medium"
	} else {
		response.RiskLevel = "low"
	}

	response.IsVPN = proxyResult.IsVPN
	response.IsProxy = proxyResult.IsProxy
	response.IsTor = proxyResult.IsTor

	analysis := &DetectionAnalysis{
		AnomalyScore: anomalyResult.Score,
		IsAnomaly:    anomalyResult.IsAnomaly,
		AnomalyType:  anomalyResult.AnomalyType,
		Severity:     anomalyResult.Severity,
	}

	similarFps := h.fingerprintAnalyzer.GetSimilarFingerprints(fpAnalysis.FingerprintID, 70)
	if len(similarFps) > 0 {
		analysis.SimilarFingers = make([]SimilarFingerprintInfo, 0)
		for _, sim := range similarFps {
			if len(analysis.SimilarFingers) >= 5 {
				break
			}
			analysis.SimilarFingers = append(analysis.SimilarFingers, SimilarFingerprintInfo{
				FingerprintID: sim.FingerprintID,
				Similarity:    sim.Similarity,
				CommonFields:  sim.CommonFields,
				DiffFields:    sim.DiffFields,
			})
		}
	}

	response.Analysis = analysis
	response.ProxyResult = proxyResult

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func (h *EnvironmentDetectionHandler) GetFingerprintAnalysis(c *gin.Context) {
	var req FingerprintAnalysisRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "fingerprint parameter is required: " + err.Error(),
		})
		return
	}

	if req.Threshold <= 0 {
		req.Threshold = 70
	}

	fp, exists := h.fingerprintAnalyzer.GetFingerprint(req.Fingerprint)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "fingerprint not found",
		})
		return
	}

	anomaly := h.fingerprintAnalyzer.GetAnomaly(req.Fingerprint)
	similarFps := h.fingerprintAnalyzer.GetSimilarFingerprints(req.Fingerprint, req.Threshold)
	stats := h.fingerprintAnalyzer.GetStats()
	clusters := h.fingerprintAnalyzer.GetClusters()

	similarFpInfos := make([]SimilarFingerprintInfo, 0)
	for _, sim := range similarFps {
		similarFpInfos = append(similarFpInfos, SimilarFingerprintInfo{
			FingerprintID: sim.FingerprintID,
			Similarity:    sim.Similarity,
			CommonFields:  sim.CommonFields,
			DiffFields:    sim.DiffFields,
		})
	}

	response := &FingerprintDataResponse{
		Fingerprint:   fp,
		AnomalyResult: anomaly,
		SimilarFps:    similarFpInfos,
		Stats:         stats,
		Clusters:      clusters,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func (h *EnvironmentDetectionHandler) CheckProxy(c *gin.Context) {
	var req ProxyCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ip_address is required: " + err.Error(),
		})
		return
	}

	headers := make(map[string]string)
	for _, header := range []string{
		"X-Forwarded-For", "X-Real-IP", "Via", "X-ProxyChain",
		"Forwarded", "CF-Connecting-IP",
	} {
		if val := c.GetHeader(header); val != "" {
			headers[header] = val
		}
	}

	proxyResult, err := h.proxyDetector.DetectProxy(req.IPAddress, headers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "proxy detection failed: " + err.Error(),
		})
		return
	}

	response := &ProxyCheckData{
		IPAddress:        proxyResult.IPAddress,
		IsProxy:          proxyResult.IsProxy,
		IsVPN:            proxyResult.IsVPN,
		IsTor:            proxyResult.IsTor,
		IsDataCenter:     proxyResult.IsDataCenter,
		RiskLevel:        proxyResult.RiskLevel,
		Score:            proxyResult.Score,
		Confidence:       proxyResult.Confidence,
		Country:          proxyResult.Country,
		ISP:              proxyResult.ISP,
		ASN:              proxyResult.ASN,
		DetectionMethods: proxyResult.DetectionMethods,
		Hosing:           proxyResult.Hosting,
		Mobile:           proxyResult.Mobile,
		LastChecked:      proxyResult.LastChecked,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

func (h *EnvironmentDetectionHandler) GetDetectionStats(c *gin.Context) {
	stats := h.fingerprintAnalyzer.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_fingerprints": stats.TotalFingerprints,
			"bot_fingerprints":   stats.BotFingerprints,
			"vpn_fingerprints":   stats.VPNFingerprints,
			"avg_anomaly_score":  stats.AvgAnomalyScore,
			"high_risk_count":    stats.HighRiskCount,
			"medium_risk_count":  stats.MediumRiskCount,
			"low_risk_count":     stats.LowRiskCount,
			"clusters_count":     stats.ClustersCount,
		},
	})
}

func (h *EnvironmentDetectionHandler) GetClusters(c *gin.Context) {
	clusters := h.fingerprintAnalyzer.GetClusters()

	clusterInfos := make([]map[string]interface{}, 0)
	for _, cluster := range clusters {
		clusterInfos = append(clusterInfos, map[string]interface{}{
			"cluster_id":      cluster.ClusterID,
			"size":            cluster.Size,
			"common_features": cluster.CommonFeatures,
			"risk_level":      cluster.RiskLevel,
			"first_seen":      cluster.FirstSeen,
			"last_seen":       cluster.LastSeen,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    clusterInfos,
	})
}

func calculateCombinedRiskScore(clientScore float64, fp *service.FingerprintAnalysis, anomaly *service.AnomalyResult, proxy *service.ProxyDetection) float64 {
	score := clientScore * 0.4

	if fp != nil {
		score += fp.AnomalyScore * 0.3
	}

	if anomaly != nil {
		score += anomaly.Score * 0.15
	}

	if proxy != nil {
		score += proxy.Score * 0.15
	}

	if fp != nil && fp.IsKnownBot {
		score = math.Min(score*1.5+20, 100)
	}

	if proxy != nil && (proxy.IsProxy || proxy.IsVPN || proxy.IsTor) {
		score = math.Min(score*1.3+15, 100)
	}

	return math.Round(math.Min(math.Max(score, 0), 100)*100) / 100
}

func generateRecommendations(riskScore float64, proxy *service.ProxyDetection, anomaly *service.AnomalyResult) []string {
	recommendations := make([]string, 0)

	if riskScore > 80 {
		recommendations = append(recommendations, "高风险，建议阻止访问或要求额外验证")
	} else if riskScore > 60 {
		recommendations = append(recommendations, "中高风险，建议启用验证码或短信验证")
	} else if riskScore > 40 {
		recommendations = append(recommendations, "中风险，建议记录日志并监控")
	} else {
		recommendations = append(recommendations, "低风险，允许正常访问")
	}

	if proxy != nil && proxy.IsTor {
		recommendations = append(recommendations, "检测到Tor网络，Tor常被用于绕过限制，请谨慎处理")
	}

	if proxy != nil && proxy.IsVPN {
		recommendations = append(recommendations, "检测到VPN连接，VPN可能用于隐私保护，需结合其他指标判断")
	}

	if proxy != nil && proxy.IsProxy {
		recommendations = append(recommendations, "检测到代理服务器，代理可能用于隐藏真实IP")
	}

	if anomaly != nil && anomaly.IsAnomaly {
		if anomaly.Severity == "high" {
			recommendations = append(recommendations, "检测到异常行为模式，建议触发安全告警")
		}
	}

	return recommendations
}

func (h *EnvironmentDetectionHandler) BatchDetectProxy(c *gin.Context) {
	var req struct {
		IPs []string `json:"ips" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "ips parameter is required: " + err.Error(),
		})
		return
	}

	if len(req.IPs) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "maximum 100 IPs allowed per request",
		})
		return
	}

	headers := make(map[string]string)
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		if val := c.GetHeader(header); val != "" {
			headers[header] = val
		}
	}

	results := make(map[string]*service.ProxyDetection)
	for _, ip := range req.IPs {
		result, err := h.proxyDetector.DetectProxy(ip, headers)
		if err == nil {
			results[ip] = result
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
		"count":   len(results),
	})
}

func (h *EnvironmentDetectionHandler) ExportFingerprintData(c *gin.Context) {
	db := h.fingerprintAnalyzer.GetDatabase()
	data, err := db.ExportData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "export failed: " + err.Error(),
		})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=fingerprint_data.json")
	c.Data(http.StatusOK, "application/json", data)
}

func (h *EnvironmentDetectionHandler) ImportFingerprintData(c *gin.Context) {
	var req struct {
		Data json.RawMessage `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "data parameter is required: " + err.Error(),
		})
		return
	}

	db := h.fingerprintAnalyzer.GetDatabase()
	if err := db.ImportData(req.Data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "import failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "data imported successfully",
	})
}

func (h *EnvironmentDetectionHandler) DeleteFingerprint(c *gin.Context) {
	fingerprintID := c.Param("id")
	if fingerprintID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "fingerprint id is required",
		})
		return
	}

	db := h.fingerprintAnalyzer.GetDatabase()
	db.RemoveFingerprint(fingerprintID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "fingerprint deleted successfully",
	})
}

func (h *EnvironmentDetectionHandler) CleanupOldData(c *gin.Context) {
	maxAgeStr := c.DefaultQuery("max_age", "24h")

	maxAge, err := parseDuration(maxAgeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid max_age format: " + err.Error(),
		})
		return
	}

	db := h.fingerprintAnalyzer.GetDatabase()
	removed := db.CleanupOldData(maxAge)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"removed": removed,
		"max_age": maxAgeStr,
	})
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	if strings.HasSuffix(s, "h") {
		hours, err := strconv.Atoi(strings.TrimSuffix(s, "h"))
		if err != nil {
			return 0, err
		}
		return time.Duration(hours) * time.Hour, nil
	}

	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	if strings.HasSuffix(s, "m") {
		mins, err := strconv.Atoi(strings.TrimSuffix(s, "m"))
		if err != nil {
			return 0, err
		}
		return time.Duration(mins) * time.Minute, nil
	}

	return time.ParseDuration(s)
}

func (h *EnvironmentDetectionHandler) GetVPNPatterns(c *gin.Context) {
	patterns := h.proxyDetector.GetVPNPatterns()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    patterns,
	})
}

func (h *EnvironmentDetectionHandler) ValidateHeaders(c *gin.Context) {
	var req struct {
		Headers map[string]string `json:"headers" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "headers parameter is required: " + err.Error(),
		})
		return
	}

	isFlagged, flagged := h.proxyDetector.ValidateHeaders(req.Headers)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"is_flagged": isFlagged,
		"flagged":    flagged,
	})
}
