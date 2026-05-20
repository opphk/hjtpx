package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	github.com/hjtpx/hjtpx/internal/model"
)

type DeviceFingerprintService struct {
	canvasService *CanvasFingerprintService
	webglService *WebGLFingerprintService
	audioService *AudioContextService
	config       *DeviceFingerprintConfig
	database     *FingerprintDatabase
	mu           sync.RWMutex
}

type DeviceFingerprintConfig struct {
	EnableCanvasFingerprint  bool
	EnableWebGLFingerprint   bool
	EnableAudioFingerprint   bool
	EnableMultiFingerprint   bool
	SimilarityThreshold     float64
	AnomalyThreshold        float64
	StabilityThreshold       float64
	FingerprintCacheTTL      time.Duration
}

type DeviceFingerprint struct {
	FingerprintID     string                    `json:"fingerprint_id"`
	Timestamp         time.Time                `json:"timestamp"`
	CanvasFingerprint *model.CanvasFingerprintResult `json:"canvas_fingerprint,omitempty"`
	WebGLFingerprint  *model.WebGLFingerprintResult `json:"webgl_fingerprint,omitempty"`
	AudioFingerprint  *model.AudioFingerprint  `json:"audio_fingerprint,omitempty"`
	CombinedHash      string                   `json:"combined_hash"`
	DeviceSignature   string                   `json:"device_signature"`
	RiskLevel        string                   `json:"risk_level"`
	RiskScore        float64                  `json:"risk_score"`
	Confidence        float64                 `json:"confidence"`
	IsTrusted        bool                     `json:"is_trusted"`
	Anomalies        []FingerprintAnomaly    `json:"anomalies,omitempty"`
	Recommendations  []string                `json:"recommendations,omitempty"`
}

type FingerprintAnomaly struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Source      string  `json:"source"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}

type DeviceFingerprintAnalysis struct {
	Fingerprint        *DeviceFingerprint
	CanvasAnalysis     *model.CanvasFingerprintResult
	WebGLAnalysis      *model.WebGLFingerprintResult
	AudioAnalysis      *model.AudioFingerprint
	OverallRiskScore   float64
	OverallRiskLevel  string
	Confidence         float64
	Anomalies         []FingerprintAnomaly
	Recommendations   []string
}

func NewDeviceFingerprintService() *DeviceFingerprintService {
	return &DeviceFingerprintService{
		canvasService: NewCanvasFingerprintService(),
		webglService:  NewWebGLFingerprintService(),
		audioService:  NewAudioContextService(),
		config: &DeviceFingerprintConfig{
			EnableCanvasFingerprint: true,
			EnableWebGLFingerprint:  true,
			EnableAudioFingerprint:  true,
			EnableMultiFingerprint:  true,
			SimilarityThreshold:     85.0,
			AnomalyThreshold:         40.0,
			StabilityThreshold:       0.8,
			FingerprintCacheTTL:     24 * time.Hour,
		},
		database: NewFingerprintDatabase(),
	}
}

func (s *DeviceFingerprintService) GenerateDeviceFingerprint(info *model.EnvInfo) *DeviceFingerprint {
	s.mu.Lock()
	defer s.mu.Unlock()

	fingerprint := &DeviceFingerprint{
		FingerprintID: s.generateFingerprintID(info),
		Timestamp:    time.Now(),
		Anomalies:    make([]FingerprintAnomaly, 0),
		Recommendations: make([]string, 0),
	}

	if s.config.EnableCanvasFingerprint && info.CanvasFingerprint != "" {
		fingerprint.CanvasFingerprint = s.canvasService.GenerateEnhancedFingerprint(info)
		if fingerprint.CanvasFingerprint.RiskScore > 0 {
			s.addAnomaliesFromCanvas(fingerprint)
		}
	}

	if s.config.EnableWebGLFingerprint && (info.WebGLRenderer != "" || info.WebGLVendor != "") {
		fingerprint.WebGLFingerprint = s.webglService.GenerateEnhancedFingerprint(info)
		if fingerprint.WebGLFingerprint.RiskScore > 0 {
			s.addAnomaliesFromWebGL(fingerprint)
		}
	}

	if s.config.EnableAudioFingerprint {
		audioData := s.extractAudioDataFromEnvInfo(info)
		if len(audioData) > 0 {
			fp, _ := s.audioService.GenerateFingerprint(audioData)
			fingerprint.AudioFingerprint = fp
		}
	}

	fingerprint.CombinedHash = s.computeCombinedHash(fingerprint)
	fingerprint.DeviceSignature = s.generateDeviceSignature(fingerprint)

	s.calculateRiskAndConfidence(fingerprint)
	s.generateRecommendations(fingerprint)

	return fingerprint
}

func (s *DeviceFingerprintService) AnalyzeDeviceFingerprint(info *model.EnvInfo) *DeviceFingerprintAnalysis {
	analysis := &DeviceFingerprintAnalysis{
		Fingerprint:       s.GenerateDeviceFingerprint(info),
		Anomalies:       make([]FingerprintAnomaly, 0),
		Recommendations: make([]string, 0),
	}

	analysis.CanvasAnalysis = analysis.Fingerprint.CanvasFingerprint
	analysis.WebGLAnalysis = analysis.Fingerprint.WebGLFingerprint
	analysis.AudioAnalysis = analysis.Fingerprint.AudioFingerprint

	s.mergeAnomalies(analysis)
	s.calculateOverallRisk(analysis)
	s.generateAnalysisRecommendations(analysis)

	analysis.Confidence = s.calculateConfidence(analysis)

	return analysis
}

func (s *DeviceFingerprintService) CompareDeviceFingerprints(fp1, fp2 *DeviceFingerprint) *DeviceFingerprintComparison {
	comparison := &DeviceFingerprintComparison{
		Fingerprint1: fp1.FingerprintID,
		Fingerprint2: fp2.FingerprintID,
		SimilarityScores: make(map[string]float64),
		CommonFeatures:   make([]string, 0),
		DiffFeatures:    make([]string, 0),
	}

	if fp1 == nil || fp2 == nil {
		comparison.OverallSimilarity = 0
		return comparison
	}

	if fp1.CanvasFingerprint != nil && fp2.CanvasFingerprint != nil {
		canvasSim := s.compareCanvasFingerprints(fp1.CanvasFingerprint, fp2.CanvasFingerprint)
		comparison.SimilarityScores["canvas"] = canvasSim
		if canvasSim >= 80 {
			comparison.CommonFeatures = append(comparison.CommonFeatures, "canvas")
		} else {
			comparison.DiffFeatures = append(comparison.DiffFeatures, "canvas")
		}
	}

	if fp1.WebGLFingerprint != nil && fp2.WebGLFingerprint != nil {
		webglSim := s.compareWebGLFingerprints(fp1.WebGLFingerprint, fp2.WebGLFingerprint)
		comparison.SimilarityScores["webgl"] = webglSim
		if webglSim >= 80 {
			comparison.CommonFeatures = append(comparison.CommonFeatures, "webgl")
		} else {
			comparison.DiffFeatures = append(comparison.DiffFeatures, "webgl")
		}
	}

	if fp1.CombinedHash != "" && fp2.CombinedHash != "" {
		hashSim := s.calculateHashSimilarity(fp1.CombinedHash, fp2.CombinedHash)
		comparison.SimilarityScores["combined_hash"] = hashSim
	}

	comparison.OverallSimilarity = s.calculateOverallSimilarity(comparison.SimilarityScores)
	comparison.IsSameDevice = comparison.OverallSimilarity >= s.config.SimilarityThreshold

	comparison.Confidence = comparison.OverallSimilarity / 100.0

	return comparison
}

func (s *DeviceFingerprintService) DetectDeviceAnomalies(info *model.EnvInfo) *DeviceAnomalyDetection {
	detection := &DeviceAnomalyDetection{
		Timestamp:    time.Now(),
		Anomalies:   make([]DeviceAnomalyDetail, 0),
		RiskScore:   0.0,
		RiskLevel:   "low",
		Indicators:  make([]string, 0),
	}

	if info.CanvasFingerprint == "" {
		detection.Anomalies = append(detection.Anomalies, DeviceAnomalyDetail{
			Type:        "missing_canvas",
			Severity:    "medium",
			Description: "Canvas指纹缺失",
			Score:       20,
		})
		detection.Indicators = append(detection.Indicators, "missing_canvas_fingerprint")
	}

	if info.WebGLRenderer == "" || info.WebGLVendor == "" {
		detection.Anomalies = append(detection.Anomalies, DeviceAnomalyDetail{
			Type:        "missing_webgl",
			Severity:    "medium",
			Description: "WebGL指纹缺失",
			Score:       20,
		})
		detection.Indicators = append(detection.Indicators, "missing_webgl_fingerprint")
	}

	if info.CanvasFingerprint != "" {
		canvasResult := s.canvasService.GenerateEnhancedFingerprint(info)
		if len(canvasResult.Anomalies) > 0 {
			detection.RiskScore += float64(len(canvasResult.Anomalies)) * 10
			detection.Indicators = append(detection.Indicators, "canvas_anomalies_detected")
		}
	}

	if info.WebGLRenderer != "" {
		rendererLower := strings.ToLower(info.WebGLRenderer)
		softwareIndicators := []string{"swiftshader", "llvmpipe", "mesa", "software"}
		for _, indicator := range softwareIndicators {
			if strings.Contains(rendererLower, indicator) {
				detection.Anomalies = append(detection.Anomalies, DeviceAnomalyDetail{
					Type:        "software_renderer",
					Severity:    "high",
					Description: fmt.Sprintf("检测到软件渲染器: %s", indicator),
					Score:       40,
				})
				detection.Indicators = append(detection.Indicators, "software_renderer_detected")
				break
			}
		}
	}

	detection.RiskScore = math.Min(detection.RiskScore, 100)

	if detection.RiskScore >= 70 {
		detection.RiskLevel = "high"
	} else if detection.RiskScore >= 40 {
		detection.RiskLevel = "medium"
	}

	return detection
}

func (s *DeviceFingerprintService) OptimizeFingerprintWeights() *FingerprintWeights {
	weights := &FingerprintWeights{
		Canvas: 0.30,
		WebGL:  0.35,
		Audio:  0.20,
		Font:   0.15,
	}

	return weights
}

func (s *DeviceFingerprintService) generateFingerprintID(info *model.EnvInfo) string {
	hasher := sha256.New()

	if info.CanvasFingerprint != "" {
		hasher.Write([]byte(info.CanvasFingerprint))
	}
	if info.WebGLRenderer != "" {
		hasher.Write([]byte(info.WebGLRenderer))
	}
	if info.WebGLVendor != "" {
		hasher.Write([]byte(info.WebGLVendor))
	}

	hash := hasher.Sum(nil)
	return "dfp_" + hex.EncodeToString(hash)[:16]
}

func (s *DeviceFingerprintService) extractAudioDataFromEnvInfo(info *model.EnvInfo) map[string]interface{} {
	data := make(map[string]interface{})

	if info.Fingerprint != "" {
		parts := strings.Split(info.Fingerprint, ";")
		for _, part := range parts {
			if strings.HasPrefix(part, "audio:") {
				audioInfo := strings.TrimPrefix(part, "audio:")
				data["audio_info"] = audioInfo
			}
		}
	}

	if info.WebGLRenderer != "" {
		data["renderer"] = info.WebGLRenderer
	}

	return data
}

func (s *DeviceFingerprintService) computeCombinedHash(fp *DeviceFingerprint) string {
	hasher := sha256.New()

	if fp.CanvasFingerprint != nil && fp.CanvasFingerprint.Fingerprint != "" {
		hasher.Write([]byte(fp.CanvasFingerprint.Fingerprint))
	}

	if fp.WebGLFingerprint != nil && fp.WebGLFingerprint.Fingerprint != "" {
		hasher.Write([]byte(fp.WebGLFingerprint.Fingerprint))
	}

	if fp.AudioFingerprint != nil && fp.AudioFingerprint.AudioHash != "" {
		hasher.Write([]byte(fp.AudioFingerprint.AudioHash))
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *DeviceFingerprintService) generateDeviceSignature(fp *DeviceFingerprint) string {
	components := make([]string, 0)

	if fp.CanvasFingerprint != nil && fp.CanvasFingerprint.Fingerprint != "" {
		components = append(components, "canvas:"+fp.CanvasFingerprint.Fingerprint[:8])
	}

	if fp.WebGLFingerprint != nil && fp.WebGLFingerprint.Fingerprint != "" {
		components = append(components, "webgl:"+fp.WebGLFingerprint.Fingerprint[:8])
	}

	return strings.Join(components, "|")
}

func (s *DeviceFingerprintService) calculateRiskAndConfidence(fp *DeviceFingerprint) {
	totalScore := 0.0
	count := 0

	if fp.CanvasFingerprint != nil {
		totalScore += fp.CanvasFingerprint.RiskScore
		count++
	}

	if fp.WebGLFingerprint != nil {
		totalScore += fp.WebGLFingerprint.RiskScore
		count++
	}

	if count > 0 {
		fp.RiskScore = totalScore / float64(count)
	} else {
		fp.RiskScore = 50.0
	}

	if fp.RiskScore >= 70 {
		fp.RiskLevel = "high"
		fp.Confidence = 0.9
	} else if fp.RiskScore >= 40 {
		fp.RiskLevel = "medium"
		fp.Confidence = 0.75
	} else {
		fp.RiskLevel = "low"
		fp.Confidence = 0.85
	}

	fp.IsTrusted = fp.RiskScore < 30 && fp.Confidence > 0.8
}

func (s *DeviceFingerprintService) generateRecommendations(fp *DeviceFingerprint) {
	if fp.RiskLevel == "high" {
		fp.Recommendations = append(fp.Recommendations, "建议进行额外验证")
		fp.Recommendations = append(fp.Recommendations, "考虑阻止或标记该请求")
	}

	if len(fp.Anomalies) > 3 {
		fp.Recommendations = append(fp.Recommendations, "检测到多个异常,建议深入调查")
	}
}

func (s *DeviceFingerprintService) addAnomaliesFromCanvas(fp *DeviceFingerprint) {
	if fp.CanvasFingerprint == nil {
		return
	}

	for _, anomaly := range fp.CanvasFingerprint.Anomalies {
		fp.Anomalies = append(fp.Anomalies, FingerprintAnomaly{
			Type:        anomaly.Type,
			Severity:    anomaly.Severity,
			Source:      "canvas",
			Description: anomaly.Description,
			Score:       10,
		})
	}
}

func (s *DeviceFingerprintService) addAnomaliesFromWebGL(fp *DeviceFingerprint) {
	if fp.WebGLFingerprint == nil {
		return
	}

	for _, anomaly := range fp.WebGLFingerprint.Anomalies {
		fp.Anomalies = append(fp.Anomalies, FingerprintAnomaly{
			Type:        anomaly.Type,
			Severity:    anomaly.Severity,
			Source:      "webgl",
			Description: anomaly.Description,
			Score:       10,
		})
	}
}

func (s *DeviceFingerprintService) mergeAnomalies(analysis *DeviceFingerprintAnalysis) {
	analysis.Anomalies = append(analysis.Anomalies, analysis.Fingerprint.Anomalies...)
}

func (s *DeviceFingerprintService) calculateOverallRisk(analysis *DeviceFingerprintAnalysis) {
	analysis.OverallRiskScore = analysis.Fingerprint.RiskScore

	if analysis.CanvasAnalysis != nil && analysis.CanvasAnalysis.RiskScore > 50 {
		analysis.OverallRiskScore += 10
	}

	if analysis.WebGLAnalysis != nil && analysis.WebGLAnalysis.RiskScore > 50 {
		analysis.OverallRiskScore += 15
	}

	analysis.OverallRiskScore = math.Min(analysis.OverallRiskScore, 100)

	if analysis.OverallRiskScore >= 70 {
		analysis.OverallRiskLevel = "high"
	} else if analysis.OverallRiskScore >= 40 {
		analysis.OverallRiskLevel = "medium"
	} else {
		analysis.OverallRiskLevel = "low"
	}
}

func (s *DeviceFingerprintService) generateAnalysisRecommendations(analysis *DeviceFingerprintAnalysis) {
	if analysis.OverallRiskLevel == "high" {
		analysis.Recommendations = append(analysis.Recommendations, "建议进行额外验证")
		analysis.Recommendations = append(analysis.Recommendations, "考虑阻止或标记该请求")
	}

	if len(analysis.Anomalies) > 5 {
		analysis.Recommendations = append(analysis.Recommendations, "检测到多个异常,建议深入调查")
	}

	for _, anomaly := range analysis.Anomalies {
		if anomaly.Source == "canvas" && anomaly.Severity == "high" {
			analysis.Recommendations = append(analysis.Recommendations, "Canvas指纹异常,可能为自动化环境")
		}
		if anomaly.Source == "webgl" && anomaly.Severity == "high" {
			analysis.Recommendations = append(analysis.Recommendations, "WebGL指纹异常,可能为虚拟机或软件渲染")
		}
	}
}

func (s *DeviceFingerprintService) calculateConfidence(analysis *DeviceFingerprintAnalysis) float64 {
	confidence := 0.5

	if analysis.Fingerprint.CanvasFingerprint != nil {
		confidence += 0.15
	}

	if analysis.Fingerprint.WebGLFingerprint != nil {
		confidence += 0.2
	}

	if analysis.Fingerprint.AudioFingerprint != nil {
		confidence += 0.15
	}

	return math.Min(confidence, 1.0)
}

func (s *DeviceFingerprintService) compareCanvasFingerprints(fp1, fp2 *model.CanvasFingerprintResult) float64 {
	if fp1 == nil || fp2 == nil || fp1.Fingerprint == "" || fp2.Fingerprint == "" {
		return 0
	}

	if fp1.Fingerprint == fp2.Fingerprint {
		return 100
	}

	return s.calculateHashSimilarity(fp1.Fingerprint, fp2.Fingerprint)
}

func (s *DeviceFingerprintService) compareWebGLFingerprints(fp1, fp2 *model.WebGLFingerprintResult) float64 {
	if fp1 == nil || fp2 == nil || fp1.Fingerprint == "" || fp2.Fingerprint == "" {
		return 0
	}

	if fp1.Fingerprint == fp2.Fingerprint {
		return 100
	}

	return s.calculateHashSimilarity(fp1.Fingerprint, fp2.Fingerprint)
}

func (s *DeviceFingerprintService) calculateHashSimilarity(hash1, hash2 string) float64 {
	if hash1 == hash2 {
		return 100.0
	}

	if len(hash1) != len(hash2) {
		return 0.0
	}

	if len(hash1) == 0 {
		return 0.0
	}

	matches := 0
	for i := 0; i < len(hash1) && i < len(hash2); i++ {
		if hash1[i] == hash2[i] {
			matches++
		}
	}

	return float64(matches) / float64(len(hash1)) * 100.0
}

func (s *DeviceFingerprintService) calculateOverallSimilarity(scores map[string]float64) float64 {
	if len(scores) == 0 {
		return 0.0
	}

	weights := map[string]float64{
		"canvas":        0.30,
		"webgl":         0.35,
		"audio":         0.20,
		"combined_hash": 0.15,
	}

	totalWeight := 0.0
	weightedSum := 0.0

	for key, weight := range weights {
		if score, ok := scores[key]; ok {
			weightedSum += score * weight
			totalWeight += weight
		}
	}

	if totalWeight == 0 {
		return 0.0
	}

	return weightedSum / totalWeight
}

type DeviceFingerprintComparison struct {
	Fingerprint1      string             `json:"fingerprint1"`
	Fingerprint2      string             `json:"fingerprint2"`
	SimilarityScores  map[string]float64 `json:"similarity_scores"`
	OverallSimilarity float64            `json:"overall_similarity"`
	CommonFeatures    []string           `json:"common_features"`
	DiffFeatures      []string           `json:"diff_features"`
	IsSameDevice      bool               `json:"is_same_device"`
	Confidence        float64            `json:"confidence"`
	Recommendations   []string           `json:"recommendations,omitempty"`
}

type DeviceAnomalyDetection struct {
	Timestamp    time.Time              `json:"timestamp"`
	Anomalies   []DeviceAnomalyDetail `json:"anomalies"`
	RiskScore   float64               `json:"risk_score"`
	RiskLevel   string                `json:"risk_level"`
	Indicators  []string              `json:"indicators"`
	Recommendations []string          `json:"recommendations,omitempty"`
}

type DeviceAnomalyDetail struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}

type FingerprintWeights struct {
	Canvas float64 `json:"canvas"`
	WebGL  float64 `json:"webgl"`
	Audio  float64 `json:"audio"`
	Font   float64 `json:"font"`
}

func (s *DeviceFingerprintService) ExportFingerprint(fpID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fp, exists := s.database.GetFingerprint(fpID)
	if !exists {
		return "", fmt.Errorf("fingerprint not found: %s", fpID)
	}

	data, err := json.MarshalIndent(fp, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *DeviceFingerprintService) GetFingerprintStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_fingerprints"] = len(s.database.fingerprints)

	highRisk := 0
	mediumRisk := 0
	lowRisk := 0

	for _, fp := range s.database.fingerprints {
		if fp.RiskLevel == "high" {
			highRisk++
		} else if fp.RiskLevel == "medium" {
			mediumRisk++
		} else {
			lowRisk++
		}
	}

	stats["high_risk_count"] = highRisk
	stats["medium_risk_count"] = mediumRisk
	stats["low_risk_count"] = lowRisk

	return stats
}

func (s *DeviceFingerprintService) UpdateConfig(config *DeviceFingerprintConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config.EnableCanvasFingerprint {
		s.config.EnableCanvasFingerprint = config.EnableCanvasFingerprint
	}
	if config.EnableWebGLFingerprint {
		s.config.EnableWebGLFingerprint = config.EnableWebGLFingerprint
	}
	if config.EnableAudioFingerprint {
		s.config.EnableAudioFingerprint = config.EnableAudioFingerprint
	}
	if config.EnableMultiFingerprint {
		s.config.EnableMultiFingerprint = config.EnableMultiFingerprint
	}
	if config.SimilarityThreshold > 0 {
		s.config.SimilarityThreshold = config.SimilarityThreshold
	}
	if config.AnomalyThreshold > 0 {
		s.config.AnomalyThreshold = config.AnomalyThreshold
	}
	if config.StabilityThreshold > 0 {
		s.config.StabilityThreshold = config.StabilityThreshold
	}
}
