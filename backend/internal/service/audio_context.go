package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"sync"
	"time"

	github.com/hjtpx/hjtpx/internal/model"
)

type AudioContextService struct {
	database            *AudioFingerprintDB
	config              *AudioContextConfig
	analyzer            *AudioFingerprintAnalyzer
	similarityCalculator *AudioSimilarityCalculator
	cache               *AudioFingerprintCache
}

type AudioFingerprintDB struct {
	fingerprints map[string]*model.AudioFingerprint
	mu          sync.RWMutex
}

type AudioContextConfig struct {
	EnableDetailedAnalysis bool
	AnalysisTimeout       time.Duration
	MaxFingerprintAge     time.Duration
	SimilarityThreshold  float64
	CacheEnabled         bool
}

type AudioFingerprintAnalyzer struct {
	database *model.AudioFingerprintDatabase
	mu       sync.RWMutex
}

type AudioSimilarityCalculator struct {
	weights map[string]float64
	mu      sync.RWMutex
}

type AudioFingerprintCache struct {
	cache map[string]*CachedAudioFingerprint
	mu    sync.RWMutex
	ttl   time.Duration
}

type CachedAudioFingerprint struct {
	Fingerprint *model.AudioFingerprint
	Timestamp  time.Time
}

type AudioContextMetrics struct {
	SampleRate           int     `json:"sample_rate"`
	State                string  `json:"state"`
	ChannelCount         int     `json:"channel_count"`
	Latency              float64 `json:"latency"`
	IsSupported          bool    `json:"is_supported"`
	MaxChannelCount      int     `json:"max_channel_count"`
	OutputTimestamp      float64 `json:"output_timestamp"`
	CurrentTime          float64 `json:"current_time"`
}

type AudioProcessingMetrics struct {
	FrequencyData          []float64 `json:"frequency_data"`
	TimeDomainData        []float64 `json:"time_domain_data"`
	PeakFrequency         float64   `json:"peak_frequency"`
	PeakAmplitude         float64   `json:"peak_amplitude"`
	RMSAmplitude          float64   `json:"rms_amplitude"`
	SpectralCentroid      float64   `json:"spectral_centroid"`
	SpectralFlatness      float64   `json:"spectral_flatness"`
	ZeroCrossingRate      float64   `json:"zero_crossing_rate"`
	TotalHarmonicDistortion float64 `json:"total_harmonic_distortion"`
	ProcessingTime        float64   `json:"processing_time"`
}

type EnhancedAudioAnalysisResult struct {
	Success              bool                     `json:"success"`
	AudioAnalysis        *AudioAnalysisMetrics   `json:"audio_analysis"`
	ProcessingAnalysis   []AudioProcessingFeature `json:"processing_analysis"`
	Anomalies           []AudioAnomalyDetail     `json:"anomalies"`
	RiskScore           float64                  `json:"risk_score"`
	RiskLevel           string                   `json:"risk_level"`
	Recommendations     []string                `json:"recommendations"`
	Confidence          float64                 `json:"confidence"`
}

type AudioAnalysisMetrics struct {
	SampleRate    int    `json:"sample_rate"`
	ChannelCount int    `json:"channel_count"`
	State        string `json:"state"`
	Latency      float64 `json:"latency"`
	IsHardware   bool   `json:"is_hardware"`
}

type AudioProcessingFeature struct {
	FeatureName  string  `json:"feature_name"`
	IsSupported  bool    `json:"is_supported"`
	Confidence   float64 `json:"confidence"`
	Details      string  `json:"details"`
}

type AudioAnomalyDetail struct {
	Type            string `json:"type"`
	Severity        string `json:"severity"`
	Description     string `json:"description"`
	DetectionMethod string `json:"detection_method"`
	Evidence        string `json:"evidence,omitempty"`
}

func (r *EnhancedAudioAnalysisResult) calculateOverallRisk() {
	r.RiskScore = 0.0

	for _, anomaly := range r.Anomalies {
		switch anomaly.Severity {
		case "high":
			r.RiskScore += 30
		case "medium":
			r.RiskScore += 15
		case "low":
			r.RiskScore += 5
		}
	}

	r.RiskScore = math.Min(r.RiskScore, 100)

	if r.RiskScore >= 70 {
		r.RiskLevel = "high"
		r.Confidence = 0.95
	} else if r.RiskScore >= 40 {
		r.RiskLevel = "medium"
		r.Confidence = 0.75
	} else {
		r.RiskLevel = "low"
		r.Confidence = 0.85
	}
}

func (r *EnhancedAudioAnalysisResult) generateRecommendations() {
	if r.RiskLevel == "high" {
		r.Recommendations = append(r.Recommendations, "建议进行额外验证")
		r.Recommendations = append(r.Recommendations, "考虑阻止或标记该请求")
	}

	if !r.AudioAnalysis.IsHardware {
		r.Recommendations = append(r.Recommendations, "检测到软件音频渲染")
	}

	if len(r.Anomalies) > 3 {
		r.Recommendations = append(r.Recommendations, "检测到多个异常,建议深入调查")
	}
}

type EnhancedAudioAnomalyResult struct {
	Success        bool                  `json:"success"`
	Error         string                `json:"error,omitempty"`
	FingerprintID  string                `json:"fingerprint_id"`
	Anomalies     []AudioAnomalyDetail  `json:"anomalies"`
	RiskScore     float64               `json:"risk_score"`
	RiskLevel     string                `json:"risk_level"`
	Recommendations []string              `json:"recommendations"`
}

func NewAudioContextService() *AudioContextService {
	return &AudioContextService{
		database: NewAudioFingerprintDB(),
		config: &AudioContextConfig{
			EnableDetailedAnalysis: true,
			AnalysisTimeout:       5 * time.Second,
			MaxFingerprintAge:     24 * time.Hour,
			SimilarityThreshold:   70.0,
			CacheEnabled:          true,
		},
		analyzer: NewAudioFingerprintAnalyzer(),
		similarityCalculator: NewAudioSimilarityCalculator(),
		cache: NewAudioFingerprintCache(5 * time.Minute),
	}
}

func NewAudioFingerprintDB() *AudioFingerprintDB {
	return &AudioFingerprintDB{
		fingerprints: make(map[string]*model.AudioFingerprint),
	}
}

func NewAudioFingerprintAnalyzer() *AudioFingerprintAnalyzer {
	return &AudioFingerprintAnalyzer{
		database: model.NewAudioFingerprintDatabase(),
	}
}

func NewAudioSimilarityCalculator() *AudioSimilarityCalculator {
	return &AudioSimilarityCalculator{
		weights: map[string]float64{
			"sample_rate":       0.15,
			"channel_count":      0.10,
			"channel_mode":      0.10,
			"peak_frequency":    0.15,
			"rms_amplitude":    0.10,
			"spectral_centroid": 0.10,
			"oscillator_type":   0.10,
			"fft_size":          0.10,
			"rendering_mode":    0.05,
			"hardware":          0.05,
		},
	}
}

func NewAudioFingerprintCache(ttl time.Duration) *AudioFingerprintCache {
	return &AudioFingerprintCache{
		cache: make(map[string]*CachedAudioFingerprint),
		ttl:   ttl,
	}
}

func (s *AudioContextService) GenerateFingerprint(data map[string]interface{}) (*model.AudioFingerprint, error) {
	if data == nil {
		data = make(map[string]interface{})
	}

	fingerprint := &model.AudioFingerprint{
		FingerprintID: generateAudioFingerprintID(),
		Timestamp:     time.Now(),
	}

	s.extractContextProperties(fingerprint, data)
	s.extractDestinationInfo(fingerprint, data)
	s.extractProcessingData(fingerprint, data)
	s.extractOscillatorConfig(fingerprint, data)
	s.extractGainNodeConfig(fingerprint, data)
	s.extractAnalyserConfig(fingerprint, data)
	s.extractCapabilities(fingerprint, data)
	s.analyzeRenderingMode(fingerprint)
	s.analyzeHardwareAcceleration(fingerprint)

	fingerprint.AudioHash = fingerprint.GenerateHash()

	if s.config.CacheEnabled {
		s.cache.Store(fingerprint.FingerprintID, fingerprint)
	}

	s.database.Add(fingerprint)
	s.analyzer.database.AddFingerprint(fingerprint)

	return fingerprint, nil
}

func (s *AudioContextService) extractContextProperties(fp *model.AudioFingerprint, data map[string]interface{}) {
	if sampleRate, ok := data["sample_rate"].(float64); ok {
		fp.ContextProperties.SampleRate = int(sampleRate)
	}
	if state, ok := data["state"].(string); ok {
		fp.ContextProperties.State = state
	}
	if numberOfInputs, ok := data["number_of_inputs"].(float64); ok {
		fp.ContextProperties.NumberOfInputs = int(numberOfInputs)
	}
	if numberOfOutputs, ok := data["number_of_outputs"].(float64); ok {
		fp.ContextProperties.NumberOfOutputs = int(numberOfOutputs)
	}
	if channelCount, ok := data["channel_count"].(float64); ok {
		fp.ContextProperties.ChannelCount = int(channelCount)
	}
	if channelCountMode, ok := data["channel_count_mode"].(string); ok {
		fp.ContextProperties.ChannelCountMode = channelCountMode
	}
	if channelInterpretation, ok := data["channel_interpretation"].(string); ok {
		fp.ContextProperties.ChannelInterpretation = channelInterpretation
	}
	if latencyHint, ok := data["latency_hint"].(string); ok {
		fp.ContextProperties.LatencyHint = latencyHint
	}
	if baseLatency, ok := data["base_latency"].(float64); ok {
		fp.ContextProperties.BaseLatency = baseLatency
	}
	if outputTimestamp, ok := data["output_timestamp"].(float64); ok {
		fp.ContextProperties.OutputTimestamp = outputTimestamp
	}
	if currentTime, ok := data["current_time"].(float64); ok {
		fp.ContextProperties.CurrentTime = currentTime
	}
}

func (s *AudioContextService) extractDestinationInfo(fp *model.AudioFingerprint, data map[string]interface{}) {
	if destType, ok := data["destination_type"].(string); ok {
		fp.DestinationInfo.Type = destType
	}
	if context, ok := data["destination_context"].(string); ok {
		fp.DestinationInfo.Context = context
	}
	if numInputs, ok := data["destination_number_of_inputs"].(float64); ok {
		fp.DestinationInfo.NumberOfInputs = int(numInputs)
	}
	if numOutputs, ok := data["destination_number_of_outputs"].(float64); ok {
		fp.DestinationInfo.NumberOfOutputs = int(numOutputs)
	}
	if destChannelCount, ok := data["destination_channel_count"].(float64); ok {
		fp.DestinationInfo.ChannelCount = int(destChannelCount)
	}
	if destChannelMode, ok := data["destination_channel_count_mode"].(string); ok {
		fp.DestinationInfo.ChannelCountMode = destChannelMode
	}
}

func (s *AudioContextService) extractProcessingData(fp *model.AudioFingerprint, data map[string]interface{}) {
	if frequencyData, ok := data["frequency_data"].([]interface{}); ok {
		for _, val := range frequencyData {
			if fv, ok := val.(float64); ok {
				fp.ProcessingData.FrequencyData = append(fp.ProcessingData.FrequencyData, fv)
			}
		}
	}

	if timeDomainData, ok := data["time_domain_data"].([]interface{}); ok {
		for _, val := range timeDomainData {
			if fv, ok := val.(float64); ok {
				fp.ProcessingData.TimeDomainData = append(fp.ProcessingData.TimeDomainData, fv)
			}
		}
	}

	if peakFreq, ok := data["peak_frequency"].(float64); ok {
		fp.ProcessingData.PeakFrequency = peakFreq
	}
	if peakAmp, ok := data["peak_amplitude"].(float64); ok {
		fp.ProcessingData.PeakAmplitude = peakAmp
	}
	if rmsAmp, ok := data["rms_amplitude"].(float64); ok {
		fp.ProcessingData.RMSAmplitude = rmsAmp
	}
	if spectralCentroid, ok := data["spectral_centroid"].(float64); ok {
		fp.ProcessingData.SpectralCentroid = spectralCentroid
	}
	if spectralFlatness, ok := data["spectral_flatness"].(float64); ok {
		fp.ProcessingData.SpectralFlatness = spectralFlatness
	}
	if zcr, ok := data["zero_crossing_rate"].(float64); ok {
		fp.ProcessingData.ZeroCrossingRate = zcr
	}
	if thd, ok := data["total_harmonic_distortion"].(float64); ok {
		fp.ProcessingData.TotalHarmonicDistortion = thd
	}
}

func (s *AudioContextService) extractOscillatorConfig(fp *model.AudioFingerprint, data map[string]interface{}) {
	if oscType, ok := data["oscillator_type"].(string); ok {
		fp.OscillatorConfig.Type = oscType
	}
	if frequency, ok := data["oscillator_frequency"].(float64); ok {
		fp.OscillatorConfig.Frequency = frequency
	}
	if detune, ok := data["oscillator_detune"].(float64); ok {
		fp.OscillatorConfig.Detune = detune
	}
}

func (s *AudioContextService) extractGainNodeConfig(fp *model.AudioFingerprint, data map[string]interface{}) {
	if gain, ok := data["gain_value"].(float64); ok {
		fp.GainNodeConfig.Gain = gain
	}
}

func (s *AudioContextService) extractAnalyserConfig(fp *model.AudioFingerprint, data map[string]interface{}) {
	if fftSize, ok := data["fft_size"].(float64); ok {
		fp.AnalyserConfig.FFTSize = int(fftSize)
	}
	if freqBinCount, ok := data["frequency_bin_count"].(float64); ok {
		fp.AnalyserConfig.FrequencyBinCount = int(freqBinCount)
	}
	if minDecibels, ok := data["min_decibels"].(float64); ok {
		fp.AnalyserConfig.MinDecibels = minDecibels
	}
	if maxDecibels, ok := data["max_decibels"].(float64); ok {
		fp.AnalyserConfig.MaxDecibels = maxDecibels
	}
	if smoothing, ok := data["smoothing_time_constant"].(float64); ok {
		fp.AnalyserConfig.SmoothingTimeConstant = smoothing
	}
}

func (s *AudioContextService) extractCapabilities(fp *model.AudioFingerprint, data map[string]interface{}) {
	if supportedFormats, ok := data["supported_formats"].([]interface{}); ok {
		for _, format := range supportedFormats {
			if formatStr, ok := format.(string); ok {
				fp.SupportedFormats = append(fp.SupportedFormats, formatStr)
			}
		}
	}

	if decodingCapabilities, ok := data["decoding_capabilities"].([]interface{}); ok {
		for _, cap := range decodingCapabilities {
			if capStr, ok := cap.(string); ok {
				fp.DecodingCapabilities = append(fp.DecodingCapabilities, capStr)
			}
		}
	}

	if isAudioContextSupported, ok := data["is_audio_context_supported"].(bool); ok {
		fp.IsAudioContextSupported = isAudioContextSupported
	}
	if isOscillatorSupported, ok := data["is_oscillator_supported"].(bool); ok {
		fp.IsOscillatorSupported = isOscillatorSupported
	}
	if isAnalyserSupported, ok := data["is_analyser_supported"].(bool); ok {
		fp.IsAnalyserSupported = isAnalyserSupported
	}
	if isGainNodeSupported, ok := data["is_gain_node_supported"].(bool); ok {
		fp.IsGainNodeSupported = isGainNodeSupported
	}
	if isStereoPannerSupported, ok := data["is_stereo_panner_supported"].(bool); ok {
		fp.IsStereoPannerSupported = isStereoPannerSupported
	}
	if isOfflineContext, ok := data["is_offline_context"].(bool); ok {
		fp.IsOfflineContext = isOfflineContext
	}
	if offlineDuration, ok := data["offline_duration"].(float64); ok {
		fp.OfflineDuration = offlineDuration
	}
	if maxChannelCount, ok := data["max_channel_count"].(float64); ok {
		fp.MaxChannelCount = int(maxChannelCount)
	}
	if renderingConsistency, ok := data["rendering_consistency"].(float64); ok {
		fp.RenderingConsistency = renderingConsistency
	}
	if noiseLevel, ok := data["noise_level"].(float64); ok {
		fp.NoiseLevel = noiseLevel
	}
	if isSoftwareRenderer, ok := data["is_software_renderer"].(bool); ok {
		fp.IsSoftwareRenderer = isSoftwareRenderer
	}
	if isHardwareAccelerated, ok := data["is_hardware_accelerated"].(bool); ok {
		fp.IsHardwareAccelerated = isHardwareAccelerated
	}
	if apiVersion, ok := data["browser_audio_api_version"].(string); ok {
		fp.BrowserAudioAPIVersion = apiVersion
	}
	if analysisDuration, ok := data["analysis_duration"].(float64); ok {
		fp.AnalysisDuration = analysisDuration
	}
}

func (s *AudioContextService) analyzeRenderingMode(fp *model.AudioFingerprint) {
	if fp.ContextProperties.LatencyHint == "interactive" {
		fp.RenderingMode = "interactive"
		fp.RenderingLatency = 0.02
	} else if fp.ContextProperties.LatencyHint == "balanced" {
		fp.RenderingMode = "balanced"
		fp.RenderingLatency = 0.1
	} else if fp.ContextProperties.LatencyHint == "playback" {
		fp.RenderingMode = "playback"
		fp.RenderingLatency = 0.2
	} else {
		fp.RenderingMode = "default"
		fp.RenderingLatency = fp.ContextProperties.BaseLatency
	}
}

func (s *AudioContextService) EnhancedAudioAnalysis(data map[string]interface{}) *EnhancedAudioAnalysisResult {
	result := &EnhancedAudioAnalysisResult{
		Success: true,
		AudioAnalysis: &AudioAnalysisMetrics{
			SampleRate: s.getIntValue(data, "sample_rate"),
			ChannelCount: s.getIntValue(data, "channel_count"),
			State: s.getStringValue(data, "state"),
		},
		ProcessingAnalysis: make([]AudioProcessingFeature, 0),
		Anomalies: make([]AudioAnomalyDetail, 0),
		RiskScore: 0.0,
		Recommendations: make([]string, 0),
	}

	result.analyzeAudioCapabilities(data)
	result.analyzeProcessingFeatures(data)
	result.detectAudioAnomaliesEnhanced(data)
	result.calculateOverallRisk()
	result.generateRecommendations()

	return result
}

func (s *AudioContextService) analyzeAudioCapabilities(data map[string]interface{}) {
}

func (s *AudioContextService) analyzeProcessingFeatures(data map[string]interface{}) {
}

func (s *AudioContextService) detectAudioAnomaliesEnhanced(data map[string]interface{}) {
}

func (s *AudioContextService) getIntValue(data map[string]interface{}, key string) int {
	if val, ok := data[key].(float64); ok {
		return int(val)
	}
	return 0
}

func (s *AudioContextService) getStringValue(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func (s *AudioContextService) ExtractAudioFeatures(data map[string]interface{}) *AudioProcessingMetrics {
	metrics := &AudioProcessingMetrics{}

	if frequencyData, ok := data["frequency_data"].([]interface{}); ok {
		for _, val := range frequencyData {
			if fv, ok := val.(float64); ok {
				metrics.FrequencyData = append(metrics.FrequencyData, fv)
			}
		}
	}

	if timeDomainData, ok := data["time_domain_data"].([]interface{}); ok {
		for _, val := range timeDomainData {
			if fv, ok := val.(float64); ok {
				metrics.TimeDomainData = append(metrics.TimeDomainData, fv)
			}
		}
	}

	metrics.PeakFrequency = calculatePeakFrequency(metrics.FrequencyData)
	metrics.PeakAmplitude = calculatePeakAmplitude(metrics.TimeDomainData)
	metrics.RMSAmplitude = calculateRMSAmplitude(metrics.TimeDomainData)
	metrics.SpectralCentroid = calculateSpectralCentroid(metrics.FrequencyData)
	metrics.SpectralFlatness = calculateSpectralFlatness(metrics.FrequencyData)
	metrics.ZeroCrossingRate = calculateZeroCrossingRate(metrics.TimeDomainData)

	if processingTime, ok := data["processing_time"].(float64); ok {
		metrics.ProcessingTime = processingTime
	}

	metrics.TotalHarmonicDistortion = s.calculateTHD(metrics.FrequencyData)

	return metrics
}

func (s *AudioContextService) calculateTHD(frequencyData []float64) float64 {
	if len(frequencyData) < 4 {
		return 0
	}

	harmonics := make([]float64, 0)
	peakIdx := 0
	peakVal := 0.0

	for i, val := range frequencyData {
		if val > peakVal {
			peakVal = val
			peakIdx = i
		}
	}

	for i := peakIdx * 2; i < len(frequencyData) && len(harmonics) < 5; i++ {
		harmonics = append(harmonics, frequencyData[i])
	}

	if len(harmonics) == 0 {
		return 0
	}

	var harmonicSum float64
	for _, h := range harmonics {
		harmonicSum += h * h
	}

	if peakVal == 0 {
		return 0
	}

	thd := math.Sqrt(harmonicSum / float64(len(harmonics))) / peakVal
	return thd * 100
}

func (s *AudioContextService) DetectAudioAnomaliesEnhanced(fpID string) *EnhancedAudioAnomalyResult {
	fp, exists := s.database.Get(fpID)
	if !exists {
		return &EnhancedAudioAnomalyResult{
			Success: false,
			Error: "fingerprint_not_found",
		}
	}

	result := &EnhancedAudioAnomalyResult{
		Success: true,
		FingerprintID: fpID,
		Anomalies: make([]AudioAnomalyDetail, 0),
		RiskScore: 0.0,
		RiskLevel: "low",
	}

	result.detectRenderingAnomalies(fp)
	result.detectProcessingAnomalies(fp)
	result.detectHardwareAnomalies(fp)
	result.detectSpoofingIndicators(fp)

	result.calculateRiskScore()

	return result
}

func (r *EnhancedAudioAnomalyResult) detectRenderingAnomalies(fp *model.AudioFingerprint) {
	if fp.RenderingConsistency > 0.999 && len(fp.ProcessingData.FrequencyData) > 100 {
		r.Anomalies = append(r.Anomalies, AudioAnomalyDetail{
			Type: "suspicious_consistency",
			Severity: "medium",
			Description: "渲染一致性异常高",
			DetectionMethod: "consistency_check",
		})
		r.RiskScore += 20
	}

	if fp.RenderingLatency == 0 && fp.ContextProperties.BaseLatency == 0 {
		r.Anomalies = append(r.Anomalies, AudioAnomalyDetail{
			Type: "zero_latency",
			Severity: "low",
			Description: "延迟为零",
			DetectionMethod: "latency_check",
		})
		r.RiskScore += 10
	}

	if fp.IsSoftwareRenderer && fp.ContextProperties.SampleRate > 48000 {
		r.Anomalies = append(r.Anomalies, AudioAnomalyDetail{
			Type: "software_high_sample_rate",
			Severity: "medium",
			Description: "软件渲染器使用高采样率",
			DetectionMethod: "rendering_check",
		})
		r.RiskScore += 15
	}
}

func (r *EnhancedAudioAnomalyResult) detectProcessingAnomalies(fp *model.AudioFingerprint) {
	if len(fp.ProcessingData.FrequencyData) == 0 {
		r.Anomalies = append(r.Anomalies, AudioAnomalyDetail{
			Type: "missing_frequency_data",
			Severity: "high",
			Description: "缺少频率数据",
			DetectionMethod: "data_check",
		})
		r.RiskScore += 30
	}

	if fp.ProcessingData.SpectralFlatness < 0.001 && len(fp.ProcessingData.FrequencyData) > 0 {
		r.Anomalies = append(r.Anomalies, AudioAnomalyDetail{
			Type: "flat_spectrum",
			Severity: "high",
			Description: "频谱过于平坦",
			DetectionMethod: "spectral_analysis",
		})
		r.RiskScore += 25
	}

	if fp.ProcessingData.RMSAmplitude == 0 && len(fp.ProcessingData.TimeDomainData) > 0 {
		r.Anomalies = append(r.Anomalies, AudioAnomalyDetail{
			Type: "zero_amplitude",
			Severity: "medium",
			Description: "振幅为零",
			DetectionMethod: "amplitude_check",
		})
		r.RiskScore += 20
	}
}

func (r *EnhancedAudioAnomalyResult) detectHardwareAnomalies(fp *model.AudioFingerprint) {
	if !fp.IsAudioContextSupported && !fp.IsOscillatorSupported {
		r.Anomalies = append(r.Anomalies, AudioAnomalyDetail{
			Type: "no_audio_support",
			Severity: "high",
			Description: "不支持音频API",
			DetectionMethod: "capability_check",
		})
		r.RiskScore += 40
	}

	if fp.MaxChannelCount > 32 {
		r.Anomalies = append(r.Anomalies, AudioAnomalyDetail{
			Type: "unusual_channel_count",
			Severity: "low",
			Description: "声道数异常高",
			DetectionMethod: "channel_check",
		})
		r.RiskScore += 10
	}

	if fp.ContextProperties.SampleRate == 0 {
		r.Anomalies = append(r.Anomalies, AudioAnomalyDetail{
			Type: "missing_sample_rate",
			Severity: "high",
			Description: "缺少采样率",
			DetectionMethod: "sample_rate_check",
		})
		r.RiskScore += 30
	}
}

func (r *EnhancedAudioAnomalyResult) detectSpoofingIndicators(fp *model.AudioFingerprint) {
	patterns := detectSuspiciousPatterns(fp)
	for _, pattern := range patterns {
		r.Anomalies = append(r.Anomalies, AudioAnomalyDetail{
			Type: "spoofing_indicator",
			Severity: "medium",
			Description: fmt.Sprintf("检测到可疑模式: %s", pattern),
			DetectionMethod: "pattern_analysis",
		})
		r.RiskScore += 15
	}
}

func (r *EnhancedAudioAnomalyResult) calculateRiskScore() {
	r.RiskScore = math.Min(r.RiskScore, 100)

	if r.RiskScore >= 70 {
		r.RiskLevel = "high"
	} else if r.RiskScore >= 40 {
		r.RiskLevel = "medium"
	} else {
		r.RiskLevel = "low"
	}
}

func (r *EnhancedAudioAnomalyResult) generateRecommendations() {
	if r.RiskLevel == "high" {
		r.Recommendations = append(r.Recommendations, "建议进行额外验证")
		r.Recommendations = append(r.Recommendations, "考虑阻止或标记该请求")
	}

	if len(r.Anomalies) > 3 {
		r.Recommendations = append(r.Recommendations, "检测到多个异常,建议深入调查")
	}
}

func (s *AudioContextService) analyzeHardwareAcceleration(fp *model.AudioFingerprint) {
	if fp.ContextProperties.SampleRate > 44100 && fp.MaxChannelCount >= 2 {
		fp.IsHardwareAccelerated = true
	}

	if fp.RenderingConsistency > 0.95 && fp.NoiseLevel < 0.01 {
		fp.IsSoftwareRenderer = true
	}
}

func (s *AudioContextService) AnalyzeFingerprint(fpID string) (*model.AudioFingerprintAnalysis, error) {
	fp, exists := s.database.Get(fpID)
	if !exists {
		return nil, fmt.Errorf("指纹不存在: %s", fpID)
	}

	analysis := s.analyzer.Analyze(fp)

	return analysis, nil
}

func (s *AudioContextService) CompareFingerprints(fpID1, fpID2 string) (*model.AudioComparisonResult, error) {
	fp1, exists1 := s.database.Get(fpID1)
	fp2, exists2 := s.database.Get(fpID2)

	if !exists1 {
		return nil, fmt.Errorf("第一个指纹不存在: %s", fpID1)
	}
	if !exists2 {
		return nil, fmt.Errorf("第二个指纹不存在: %s", fpID2)
	}

	comparison := s.similarityCalculator.Calculate(fp1, fp2)

	return comparison, nil
}

func (s *AudioContextService) DetectAnomalies(fpID string) (*AudioAnomalyDetection, error) {
	fp, exists := s.database.Get(fpID)
	if !exists {
		return nil, fmt.Errorf("指纹不存在: %s", fpID)
	}

	detection := &AudioAnomalyDetection{
		Timestamp:   time.Now(),
		Fingerprint: fp,
	}

	s.detectRenderingAnomalies(detection, fp)
	s.detectProcessingAnomalies(detection, fp)
	s.detectHardwareAnomalies(detection, fp)

	detection.CalculateOverallRiskScore()

	return detection, nil
}

func (s *AudioContextService) detectRenderingAnomalies(detection *AudioAnomalyDetection, fp *model.AudioFingerprint) {
	if fp.RenderingConsistency > 0.999 && len(fp.ProcessingData.FrequencyData) > 100 {
		detection.AnomalyIndicators = append(detection.AnomalyIndicators, "suspiciously_consistent_rendering")
		detection.RiskScore += 25
	}

	if fp.RenderingLatency == 0 {
		detection.AnomalyIndicators = append(detection.AnomalyIndicators, "zero_latency")
		detection.RiskScore += 15
	}

	if fp.IsSoftwareRenderer && fp.ContextProperties.SampleRate > 48000 {
		detection.AnomalyIndicators = append(detection.AnomalyIndicators, "high_sample_rate_software")
		detection.RiskScore += 20
	}
}

func (s *AudioContextService) detectProcessingAnomalies(detection *AudioAnomalyDetection, fp *model.AudioFingerprint) {
	if len(fp.ProcessingData.FrequencyData) == 0 {
		detection.AnomalyIndicators = append(detection.AnomalyIndicators, "missing_frequency_data")
		detection.RiskScore += 30
	}

	if fp.ProcessingData.SpectralFlatness == 0 {
		detection.AnomalyIndicators = append(detection.AnomalyIndicators, "perfectly_flat_spectrum")
		detection.RiskScore += 35
	}

	if fp.ProcessingData.RMSAmplitude == 0 {
		detection.AnomalyIndicators = append(detection.AnomalyIndicators, "zero_amplitude")
		detection.RiskScore += 20
	}

	if fp.ProcessingData.ZeroCrossingRate == 0 && len(fp.ProcessingData.TimeDomainData) > 0 {
		detection.AnomalyIndicators = append(detection.AnomalyIndicators, "no_zero_crossings")
		detection.RiskScore += 25
	}
}

func (s *AudioContextService) detectHardwareAnomalies(detection *AudioAnomalyDetection, fp *model.AudioFingerprint) {
	if !fp.IsAudioContextSupported && !fp.IsOscillatorSupported {
		detection.AnomalyIndicators = append(detection.AnomalyIndicators, "no_audio_support")
		detection.RiskScore += 40
	}

	if fp.MaxChannelCount > 32 {
		detection.AnomalyIndicators = append(detection.AnomalyIndicators, "unusually_high_channel_count")
		detection.RiskScore += 15
	}

	if fp.ContextProperties.SampleRate == 0 {
		detection.AnomalyIndicators = append(detection.AnomalyIndicators, "missing_sample_rate")
		detection.RiskScore += 30
	}
}

func (s *AudioContextService) GetSimilarFingerprints(fpID string, threshold float64) []*model.AudioFingerprint {
	return s.analyzer.database.FindSimilarFingerprints(fpID, threshold)
}

func (s *AudioContextService) ValidateFingerprint(fpID string) (bool, []string) {
	fp, exists := s.database.Get(fpID)
	if !exists {
		return false, []string{"fingerprint_not_found"}
	}

	return fp.ValidateFingerprint()
}

func (s *AudioContextService) GetFingerprint(fpID string) (*model.AudioFingerprint, bool) {
	return s.database.Get(fpID)
}

func (s *AudioContextService) GetAllFingerprints() []*model.AudioFingerprint {
	return s.database.GetAll()
}

func (s *AudioContextService) RemoveFingerprint(fpID string) {
	s.database.Remove(fpID)
	s.analyzer.database.RemoveFingerprint(fpID)
}

func (s *AudioContextService) GetStatistics() *model.AudioFingerprintStats {
	return s.analyzer.database.CalculateStats()
}

func (s *AudioContextService) ExportFingerprints() ([]byte, error) {
	return s.analyzer.database.ExportData()
}

func (s *AudioContextService) ImportFingerprints(data []byte) error {
	return s.analyzer.database.ImportData(data)
}

type AudioAnomalyDetection struct {
	Fingerprint         *model.AudioFingerprint `json:"fingerprint"`
	Timestamp           time.Time               `json:"timestamp"`
	AnomalyIndicators   []string                `json:"anomaly_indicators"`
	RiskScore           float64                 `json:"risk_score"`
	IsSuspicious        bool                    `json:"is_suspicious"`
	Severity            string                  `json:"severity"`
	Confidence          float64                 `json:"confidence"`
	Recommendations     []string                `json:"recommendations"`
}

func (d *AudioAnomalyDetection) CalculateOverallRiskScore() {
	d.RiskScore = math.Min(d.RiskScore, 100)

	if d.RiskScore >= 70 {
		d.IsSuspicious = true
		d.Severity = "high"
	} else if d.RiskScore >= 40 {
		d.IsSuspicious = true
		d.Severity = "medium"
	} else {
		d.Severity = "low"
	}

	d.Confidence = math.Min(1.0, float64(len(d.AnomalyIndicators))/10.0+0.3)

	d.generateRecommendations()
}

func (d *AudioAnomalyDetection) generateRecommendations() {
	if d.RiskScore >= 70 {
		d.Recommendations = append(d.Recommendations, "建议进行额外验证")
		d.Recommendations = append(d.Recommendations, "考虑阻止或标记该请求")
	}

	if len(d.AnomalyIndicators) > 3 {
		d.Recommendations = append(d.Recommendations, "检测到多个异常指标，需要深入调查")
	}

	if d.Fingerprint != nil && !d.Fingerprint.IsAudioContextSupported {
		d.Recommendations = append(d.Recommendations, "AudioContext不支持，可能为自动化脚本")
	}
}

func (db *AudioFingerprintDB) Add(fp *model.AudioFingerprint) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if fp.FingerprintID == "" {
		fp.FingerprintID = generateAudioFingerprintID()
	}

	db.fingerprints[fp.FingerprintID] = fp
}

func (db *AudioFingerprintDB) Get(fpID string) (*model.AudioFingerprint, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	fp, exists := db.fingerprints[fpID]
	return fp, exists
}

func (db *AudioFingerprintDB) GetAll() []*model.AudioFingerprint {
	db.mu.RLock()
	defer db.mu.RUnlock()

	result := make([]*model.AudioFingerprint, 0, len(db.fingerprints))
	for _, fp := range db.fingerprints {
		result = append(result, fp)
	}

	return result
}

func (db *AudioFingerprintDB) Remove(fpID string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	delete(db.fingerprints, fpID)
}

func (a *AudioFingerprintAnalyzer) Analyze(fp *model.AudioFingerprint) *model.AudioFingerprintAnalysis {
	a.mu.Lock()
	defer a.mu.Unlock()

	analysis := &model.AudioFingerprintAnalysis{
		FingerprintID:     fp.FingerprintID,
		AnomalyIndicators: make([]string, 0),
		SuspiciousReasons: make([]string, 0),
	}

	a.analyzeRendering(analysis, fp)
	a.analyzeProcessing(analysis, fp)
	a.analyzeUniqueness(analysis, fp)
	a.calculateConsistency(analysis, fp)

	analysis.DetermineSuspicious()
	analysis.CalculateFinalScore()

	return analysis
}

func (a *AudioFingerprintAnalyzer) analyzeRendering(analysis *model.AudioFingerprintAnalysis, fp *model.AudioFingerprint) {
	analysis.RenderingAnalysis = &model.RenderingAnalysis{
		RenderingMode:        fp.RenderingMode,
		RenderingLatency:    fp.RenderingLatency,
		IsHardwareAccelerated: fp.IsHardwareAccelerated,
		IsSoftwareRenderer:   fp.IsSoftwareRenderer,
		RenderingArtifacts:   make([]string, 0),
	}

	if fp.RenderingConsistency > 0.999 {
		analysis.AddAnomalyIndicator("suspiciously_consistent_rendering", 30)
	}

	if fp.IsSoftwareRenderer && fp.ContextProperties.SampleRate > 44100 {
		analysis.AddAnomalyIndicator("software_renderer_high_sample_rate", 25)
	}

	if fp.RenderingLatency == 0 {
		analysis.AddAnomalyIndicator("zero_rendering_latency", 20)
	}

	analysis.RenderingAnalysis.ConsistencyScore = fp.RenderingConsistency
	analysis.RenderingAnalysis.IsConsistent = fp.RenderingConsistency > 0.9
}

func (a *AudioFingerprintAnalyzer) analyzeProcessing(analysis *model.AudioFingerprintAnalysis, fp *model.AudioFingerprint) {
	analysis.ProcessingAnalysis = &model.ProcessingAnalysis{
		ProcessingWarnings: make([]string, 0),
	}

	if len(fp.ProcessingData.FrequencyData) == 0 {
		analysis.AddAnomalyIndicator("missing_processing_data", 35)
		analysis.ProcessingAnalysis.ProcessingWarnings = append(
			analysis.ProcessingAnalysis.ProcessingWarnings,
			"频率数据缺失",
		)
	}

	if fp.ProcessingData.SpectralFlatness < 0.01 {
		analysis.AddAnomalyIndicator("suspiciously_flat_spectrum", 30)
	}

	analysis.ProcessingAnalysis.ProcessingTime = fp.AnalysisDuration
	analysis.ProcessingAnalysis.IsStable = fp.ProcessingData.SpectralCentroid > 0
}

func (a *AudioFingerprintAnalyzer) analyzeUniqueness(analysis *model.AudioFingerprintAnalysis, fp *model.AudioFingerprint) {
	similarFps := a.database.FindSimilarFingerprints(fp.FingerprintID, 80)

	if len(similarFps) == 0 {
		analysis.UniquenessScore = 95.0
		analysis.AddAnomalyIndicator("unique_fingerprint", 40)
	} else if len(similarFps) > 10 {
		analysis.UniquenessScore = 20.0
	} else {
		analysis.UniquenessScore = 100.0 - float64(len(similarFps)*5)
	}

	if analysis.UniquenessScore > 95 {
		analysis.SuspiciousReasons = append(analysis.SuspiciousReasons, "异常独特的指纹")
	}
}

func (a *AudioFingerprintAnalyzer) calculateConsistency(analysis *model.AudioFingerprintAnalysis, fp *model.AudioFingerprint) {
	analysis.ConsistencyScore = fp.RenderingConsistency

	if fp.ContextProperties.SampleRate > 0 && fp.MaxChannelCount > 0 {
		analysis.ConsistencyScore += 0.1
	}

	if len(fp.ProcessingData.FrequencyData) > 0 {
		analysis.ConsistencyScore += 0.1
	}

	analysis.ConsistencyScore = math.Min(analysis.ConsistencyScore, 1.0)
}

func (c *AudioSimilarityCalculator) Calculate(fp1, fp2 *model.AudioFingerprint) *model.AudioComparisonResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := &model.AudioComparisonResult{
		MatchDetails: &model.MatchDetails{
			MatchingProperties: make([]string, 0),
			SimilarValues:      make(map[string]float64),
			CloseMatches:       make([]string, 0),
		},
		MismatchDetails: &model.MismatchDetails{
			MismatchedProperties: make([]string, 0),
			SignificantDiffs:    make(map[string]float64),
			MinorDiffs:          make([]string, 0),
		},
	}

	c.compareContextProperties(result, fp1, fp2)
	c.compareProcessingData(result, fp1, fp2)
	c.compareHash(result, fp1, fp2)

	result.ContextMatchScore = c.calculateContextScore(result)
	result.ProcessingMatchScore = c.calculateProcessingScore(result)
	result.HashMatchScore = c.calculateHashScore(result, fp1, fp2)

	result.SimilarityScore = result.CalculateOverallSimilarity()
	result.OverallMatch = result.SimilarityScore >= 70
	result.Confidence = c.calculateConfidence(result)
	result.Recommendation = result.GenerateRecommendation()

	return result
}

func (c *AudioSimilarityCalculator) compareContextProperties(result *model.AudioComparisonResult, fp1, fp2 *model.AudioFingerprint) {
	if fp1.ContextProperties.SampleRate == fp2.ContextProperties.SampleRate && fp1.ContextProperties.SampleRate > 0 {
		result.MatchDetails.MatchingProperties = append(result.MatchDetails.MatchingProperties, "sample_rate")
	} else if fp1.ContextProperties.SampleRate > 0 && fp2.ContextProperties.SampleRate > 0 {
		diff := math.Abs(float64(fp1.ContextProperties.SampleRate - fp2.ContextProperties.SampleRate))
		result.MatchDetails.SimilarValues["sample_rate"] = 1.0 - (diff / float64(fp1.ContextProperties.SampleRate))
		if result.MatchDetails.SimilarValues["sample_rate"] > 0.9 {
			result.MatchDetails.CloseMatches = append(result.MatchDetails.CloseMatches, "sample_rate")
		}
	}

	if fp1.ContextProperties.ChannelCount == fp2.ContextProperties.ChannelCount && fp1.ContextProperties.ChannelCount > 0 {
		result.MatchDetails.MatchingProperties = append(result.MatchDetails.MatchingProperties, "channel_count")
	}

	if fp1.ContextProperties.ChannelCountMode == fp2.ContextProperties.ChannelCountMode && fp1.ContextProperties.ChannelCountMode != "" {
		result.MatchDetails.MatchingProperties = append(result.MatchDetails.MatchingProperties, "channel_count_mode")
	}
}

func (c *AudioSimilarityCalculator) compareProcessingData(result *model.AudioComparisonResult, fp1, fp2 *model.AudioFingerprint) {
	if fp1.ProcessingData.PeakFrequency > 0 && fp2.ProcessingData.PeakFrequency > 0 {
		diff := math.Abs(fp1.ProcessingData.PeakFrequency - fp2.ProcessingData.PeakFrequency)
		avgFreq := (fp1.ProcessingData.PeakFrequency + fp2.ProcessingData.PeakFrequency) / 2
		similarity := 1.0 - (diff / avgFreq)

		result.MatchDetails.SimilarValues["peak_frequency"] = math.Max(0, similarity)
	}

	if fp1.ProcessingData.RMSAmplitude > 0 && fp2.ProcessingData.RMSAmplitude > 0 {
		diff := math.Abs(fp1.ProcessingData.RMSAmplitude - fp2.ProcessingData.RMSAmplitude)
		avgAmp := (fp1.ProcessingData.RMSAmplitude + fp2.ProcessingData.RMSAmplitude) / 2
		similarity := 1.0 - (diff / avgAmp)

		result.MatchDetails.SimilarValues["rms_amplitude"] = math.Max(0, similarity)
	}

	if fp1.ProcessingData.SpectralCentroid > 0 && fp2.ProcessingData.SpectralCentroid > 0 {
		diff := math.Abs(fp1.ProcessingData.SpectralCentroid - fp2.ProcessingData.SpectralCentroid)
		avgCentroid := (fp1.ProcessingData.SpectralCentroid + fp2.ProcessingData.SpectralCentroid) / 2
		similarity := 1.0 - (diff / avgCentroid)

		result.MatchDetails.SimilarValues["spectral_centroid"] = math.Max(0, similarity)
	}
}

func (c *AudioSimilarityCalculator) compareHash(result *model.AudioComparisonResult, fp1, fp2 *model.AudioFingerprint) {
	if fp1.AudioHash != "" && fp2.AudioHash != "" {
		if fp1.AudioHash == fp2.AudioHash {
			result.MatchDetails.MatchingProperties = append(result.MatchDetails.MatchingProperties, "audio_hash")
		} else {
			result.MismatchDetails.MismatchedProperties = append(result.MismatchDetails.MismatchedProperties, "audio_hash")
		}
	}
}

func (c *AudioSimilarityCalculator) calculateContextScore(result *model.AudioComparisonResult) float64 {
	if len(result.MatchDetails.MatchingProperties) == 0 {
		return 0
	}

	matchCount := float64(len(result.MatchDetails.MatchingProperties))
	closeMatchCount := float64(len(result.MatchDetails.CloseMatches))

	similaritySum := 0.0
	for _, sim := range result.MatchDetails.SimilarValues {
		similaritySum += sim
	}

	if len(result.MatchDetails.SimilarValues) > 0 {
		return ((matchCount + closeMatchCount + similaritySum) / (matchCount + closeMatchCount + float64(len(result.MatchDetails.SimilarValues)))) * 100
	}

	return (matchCount / (matchCount + 1)) * 100
}

func (c *AudioSimilarityCalculator) calculateProcessingScore(result *model.AudioComparisonResult) float64 {
	if len(result.MatchDetails.SimilarValues) == 0 {
		return 0
	}

	sum := 0.0
	for _, sim := range result.MatchDetails.SimilarValues {
		sum += sim
	}

	return (sum / float64(len(result.MatchDetails.SimilarValues))) * 100
}

func (c *AudioSimilarityCalculator) calculateHashScore(result *model.AudioComparisonResult, fp1, fp2 *model.AudioFingerprint) float64 {
	for _, prop := range result.MatchDetails.MatchingProperties {
		if prop == "audio_hash" {
			return 100.0
		}
	}

	return 0.0
}

func (c *AudioSimilarityCalculator) calculateConfidence(result *model.AudioComparisonResult) float64 {
	confidence := 0.5

	if len(result.MatchDetails.MatchingProperties) > 3 {
		confidence += 0.2
	}

	if len(result.MatchDetails.SimilarValues) > 2 {
		confidence += 0.2
	}

	return math.Min(confidence, 1.0)
}

func (c *AudioFingerprintCache) Store(key string, fp *model.AudioFingerprint) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = &CachedAudioFingerprint{
		Fingerprint: fp,
		Timestamp:   time.Now(),
	}
}

func (c *AudioFingerprintCache) Get(key string) (*model.AudioFingerprint, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cached, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	if time.Since(cached.Timestamp) > c.ttl {
		delete(c.cache, key)
		return nil, false
	}

	return cached.Fingerprint, true
}

func (c *AudioFingerprintCache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, key)
}

func (c *AudioFingerprintCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CachedAudioFingerprint)
}

func generateAudioFingerprintID() string {
	hasher := sha256.New()
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)

	hasher.Write([]byte(fmt.Sprintf("%d", timestamp)))
	hasher.Write(randomBytes)

	hash := hasher.Sum(nil)
	return "afp_" + hex.EncodeToString(hash)[:16]
}

func (s *AudioContextService) ExtractAudioFeatures(data map[string]interface{}) *AudioProcessingMetrics {
	metrics := &AudioProcessingMetrics{}

	if frequencyData, ok := data["frequency_data"].([]interface{}); ok {
		for _, val := range frequencyData {
			if fv, ok := val.(float64); ok {
				metrics.FrequencyData = append(metrics.FrequencyData, fv)
			}
		}
	}

	if timeDomainData, ok := data["time_domain_data"].([]interface{}); ok {
		for _, val := range timeDomainData {
			if fv, ok := val.(float64); ok {
				metrics.TimeDomainData = append(metrics.TimeDomainData, fv)
			}
		}
	}

	metrics.PeakFrequency = calculatePeakFrequency(metrics.FrequencyData)
	metrics.PeakAmplitude = calculatePeakAmplitude(metrics.TimeDomainData)
	metrics.RMSAmplitude = calculateRMSAmplitude(metrics.TimeDomainData)
	metrics.SpectralCentroid = calculateSpectralCentroid(metrics.FrequencyData)
	metrics.SpectralFlatness = calculateSpectralFlatness(metrics.FrequencyData)
	metrics.ZeroCrossingRate = calculateZeroCrossingRate(metrics.TimeDomainData)

	if processingTime, ok := data["processing_time"].(float64); ok {
		metrics.ProcessingTime = processingTime
	}

	return metrics
}

func calculatePeakFrequency(freqData []float64) float64 {
	if len(freqData) == 0 {
		return 0
	}

	maxVal := 0.0
	peakIdx := 0
	for i, val := range freqData {
		if val > maxVal {
			maxVal = val
			peakIdx = i
		}
	}

	return float64(peakIdx)
}

func calculatePeakAmplitude(timeData []float64) float64 {
	if len(timeData) == 0 {
		return 0
	}

	maxVal := 0.0
	for _, val := range timeData {
		if math.Abs(val) > maxVal {
			maxVal = math.Abs(val)
		}
	}

	return maxVal
}

func calculateRMSAmplitude(timeData []float64) float64 {
	if len(timeData) == 0 {
		return 0
	}

	sumSquares := 0.0
	for _, val := range timeData {
		sumSquares += val * val
	}

	return math.Sqrt(sumSquares / float64(len(timeData)))
}

func calculateSpectralCentroid(freqData []float64) float64 {
	if len(freqData) == 0 {
		return 0
	}

	weightedSum := 0.0
	magnitudeSum := 0.0

	for i, val := range freqData {
		weightedSum += float64(i) * val
		magnitudeSum += val
	}

	if magnitudeSum == 0 {
		return 0
	}

	return weightedSum / magnitudeSum
}

func calculateSpectralFlatness(freqData []float64) float64 {
	if len(freqData) == 0 {
		return 0
	}

	geometricMean := 1.0
	arithmeticMean := 0.0
	positiveCount := 0

	for _, val := range freqData {
		if val > 0 {
			geometricMean *= val
			positiveCount++
		}
		arithmeticMean += val
	}

	if positiveCount == 0 || arithmeticMean == 0 {
		return 0
	}

	geometricMean = math.Pow(geometricMean, 1.0/float64(positiveCount))
	arithmeticMean /= float64(len(freqData))

	return geometricMean / arithmeticMean
}

func calculateZeroCrossingRate(timeData []float64) float64 {
	if len(timeData) < 2 {
		return 0
	}

	crossings := 0
	for i := 1; i < len(timeData); i++ {
		if (timeData[i] >= 0 && timeData[i-1] < 0) || (timeData[i] < 0 && timeData[i-1] >= 0) {
			crossings++
		}
	}

	return float64(crossings) / float64(len(timeData)-1)
}

func (s *AudioContextService) DetectAudioSpoofing(data map[string]interface{}) *AudioSpoofingDetection {
	detection := &AudioSpoofingDetection{
		Timestamp: time.Now(),
		Indicators: make([]string, 0),
	}

	fp, _ := s.GenerateFingerprint(data)

	if fp != nil {
		detection.Fingerprint = fp

		if len(fp.ProcessingData.FrequencyData) == 0 {
			detection.Indicators = append(detection.Indicators, "no_audio_processing_data")
			detection.RiskScore += 30
		}

		if fp.ProcessingData.SpectralFlatness < 0.01 {
			detection.Indicators = append(detection.Indicators, "artificially_flat_spectrum")
			detection.RiskScore += 35
		}

		if fp.RenderingConsistency > 0.999 {
			detection.Indicators = append(detection.Indicators, "perfect_rendering_consistency")
			detection.RiskScore += 25
		}

		if !fp.IsAudioContextSupported && !fp.IsOscillatorSupported {
			detection.Indicators = append(detection.Indicators, "no_native_audio_support")
			detection.RiskScore += 40
		}
	}

	detection.RiskScore = math.Min(detection.RiskScore, 100)

	if detection.RiskScore >= 60 {
		detection.IsSuspicious = true
	}

	return detection
}

type AudioSpoofingDetection struct {
	Fingerprint  *model.AudioFingerprint `json:"fingerprint"`
	Timestamp   time.Time              `json:"timestamp"`
	Indicators  []string               `json:"indicators"`
	RiskScore   float64                `json:"risk_score"`
	IsSuspicious bool                   `json:"is_suspicious"`
}

func (s *AudioContextService) GetAudioContextMetrics(data map[string]interface{}) *AudioContextMetrics {
	metrics := &AudioContextMetrics{}

	if sampleRate, ok := data["sample_rate"].(float64); ok {
		metrics.SampleRate = int(sampleRate)
	}
	if state, ok := data["state"].(string); ok {
		metrics.State = state
	}
	if channelCount, ok := data["channel_count"].(float64); ok {
		metrics.ChannelCount = int(channelCount)
	}
	if latency, ok := data["latency"].(float64); ok {
		metrics.Latency = latency
	}
	if isSupported, ok := data["is_supported"].(bool); ok {
		metrics.IsSupported = isSupported
	}
	if maxChannelCount, ok := data["max_channel_count"].(float64); ok {
		metrics.MaxChannelCount = int(maxChannelCount)
	}
	if outputTimestamp, ok := data["output_timestamp"].(float64); ok {
		metrics.OutputTimestamp = outputTimestamp
	}
	if currentTime, ok := data["current_time"].(float64); ok {
		metrics.CurrentTime = currentTime
	}

	return metrics
}

func (s *AudioContextService) AnalyzeAudioRendering(data map[string]interface{}) *AudioRenderingAnalysis {
	analysis := &AudioRenderingAnalysis{
		Timestamp: time.Now(),
		Metrics:   make(map[string]interface{}),
	}

	if renderingMode, ok := data["rendering_mode"].(string); ok {
		analysis.RenderingMode = renderingMode
		analysis.Metrics["mode"] = renderingMode
	}

	if latency, ok := data["latency"].(float64); ok {
		analysis.Latency = latency
		analysis.Metrics["latency"] = latency
	}

	if consistency, ok := data["consistency"].(float64); ok {
		analysis.Consistency = consistency
		analysis.Metrics["consistency"] = consistency

		if consistency > 0.999 {
			analysis.Warnings = append(analysis.Warnings, "suspiciously_consistent_rendering")
		}
	}

	if hardwareAccel, ok := data["hardware_accelerated"].(bool); ok {
		analysis.IsHardwareAccelerated = hardwareAccel
		analysis.Metrics["hardware"] = hardwareAccel
	}

	return analysis
}

type AudioRenderingAnalysis struct {
	Timestamp             time.Time
	RenderingMode         string
	Latency              float64
	Consistency          float64
	IsHardwareAccelerated bool
	Warnings             []string
	Metrics              map[string]interface{}
}

func (s *AudioContextService) CompareAudioRendering(fp1ID, fp2ID string) *AudioRenderingComparison {
	fp1, exists1 := s.database.Get(fp1ID)
	fp2, exists2 := s.database.Get(fp2ID)

	if !exists1 || !exists2 {
		return nil
	}

	comparison := &AudioRenderingComparison{
		RenderingModeMatch: fp1.RenderingMode == fp2.RenderingMode,
		LatencyDiff:        math.Abs(fp1.RenderingLatency - fp2.RenderingLatency),
		ConsistencyDiff:    math.Abs(fp1.RenderingConsistency - fp2.RenderingConsistency),
	}

	if fp1.IsHardwareAccelerated == fp2.IsHardwareAccelerated {
		comparison.HardwareMatch = true
	}

	comparison.Similarity = (comparison.RenderingModeMatchScore() + comparison.LatencyScore() + comparison.ConsistencyScore()) / 3.0

	return comparison
}

type AudioRenderingComparison struct {
	RenderingModeMatch  bool
	LatencyDiff        float64
	ConsistencyDiff    float64
	HardwareMatch      bool
	Similarity         float64
}

func (c *AudioRenderingComparison) RenderingModeMatchScore() float64 {
	if c.RenderingModeMatch {
		return 100.0
	}
	return 0.0
}

func (c *AudioRenderingComparison) LatencyScore() float64 {
	if c.LatencyDiff < 0.01 {
		return 100.0
	}
	return math.Max(0, 100.0-c.LatencyDiff*1000)
}

func (c *AudioRenderingComparison) ConsistencyScore() float64 {
	return (1.0 - c.ConsistencyDiff) * 100.0
}

func (s *AudioContextService) GenerateAudioReport(fpID string) (string, error) {
	fp, exists := s.database.Get(fpID)
	if !exists {
		return "", fmt.Errorf("指纹不存在: %s", fpID)
	}

	report := &AudioFingerprintReport{
		FingerprintID: fp.FingerprintID,
		GeneratedAt:   time.Now(),
		ContextInfo:   fp.ContextProperties,
		ProcessingInfo: fp.ProcessingData,
		RenderingInfo: AudioRenderingInfo{
			Mode:      fp.RenderingMode,
			Latency:   fp.RenderingLatency,
			Consistency: fp.RenderingConsistency,
			Hardware:   fp.IsHardwareAccelerated,
		},
		Capabilities: AudioCapabilities{
			AudioContext: fp.IsAudioContextSupported,
			Oscillator:   fp.IsOscillatorSupported,
			Analyser:     fp.IsAnalyserSupported,
			GainNode:     fp.IsGainNodeSupported,
		},
	}

	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}

	return string(reportJSON), nil
}

type AudioFingerprintReport struct {
	FingerprintID   string
	GeneratedAt    time.Time
	ContextInfo    model.AudioContextProperty
	ProcessingInfo model.AudioProcessingData
	RenderingInfo  AudioRenderingInfo
	Capabilities   AudioCapabilities
}

type AudioRenderingInfo struct {
	Mode         string
	Latency      float64
	Consistency  float64
	Hardware     bool
}

type AudioCapabilities struct {
	AudioContext bool
	Oscillator   bool
	Analyser     bool
	GainNode     bool
}

func (s *AudioContextService) ValidateAudioContextSupport(data map[string]interface{}) *AudioSupportValidation {
	validation := &AudioSupportValidation{
		Timestamp: time.Now(),
		Checks:    make([]AudioSupportCheck, 0),
	}

	if isSupported, ok := data["is_audio_context_supported"].(bool); ok {
		validation.Checks = append(validation.Checks, AudioSupportCheck{
			Name:    "AudioContext",
			Passed:  isSupported,
			Message: "AudioContext支持",
		})
		validation.Supported = isSupported
	}

	if isSupported, ok := data["is_oscillator_supported"].(bool); ok {
		validation.Checks = append(validation.Checks, AudioSupportCheck{
			Name:    "OscillatorNode",
			Passed:  isSupported,
			Message: "OscillatorNode支持",
		})
	}

	if isSupported, ok := data["is_analyser_supported"].(bool); ok {
		validation.Checks = append(validation.Checks, AudioSupportCheck{
			Name:    "AnalyserNode",
			Passed:  isSupported,
			Message: "AnalyserNode支持",
		})
	}

	if isSupported, ok := data["is_gain_node_supported"].(bool); ok {
		validation.Checks = append(validation.Checks, AudioSupportCheck{
			Name:    "GainNode",
			Passed:  isSupported,
			Message: "GainNode支持",
		})
	}

	if isSupported, ok := data["is_stereo_panner_supported"].(bool); ok {
		validation.Checks = append(validation.Checks, AudioSupportCheck{
			Name:    "StereoPannerNode",
			Passed:  isSupported,
			Message: "StereoPannerNode支持",
		})
	}

	validation.CalculateSupportScore()

	return validation
}

type AudioSupportValidation struct {
	Timestamp   time.Time
	Checks      []AudioSupportCheck
	Supported   bool
	SupportScore float64
}

type AudioSupportCheck struct {
	Name    string
	Passed  bool
	Message string
}

func (v *AudioSupportValidation) CalculateSupportScore() {
	if len(v.Checks) == 0 {
		v.SupportScore = 0
		return
	}

	passedCount := 0
	for _, check := range v.Checks {
		if check.Passed {
			passedCount++
		}
	}

	v.SupportScore = (float64(passedCount) / float64(len(v.Checks))) * 100
}

func (s *AudioContextService) DetectAudioFingerprintingPatterns(data map[string]interface{}) *AudioFingerprintingPattern {
	pattern := &AudioFingerprintingPattern{
		Timestamp: time.Now(),
		Patterns:  make([]string, 0),
		RiskLevel: "low",
	}

	if patterns, ok := data["patterns"].([]interface{}); ok {
		for _, p := range patterns {
			if pStr, ok := p.(string); ok {
				pattern.Patterns = append(pattern.Patterns, pStr)
			}
		}
	}

	if len(pattern.Patterns) > 5 {
		pattern.RiskLevel = "high"
		pattern.IsSuspicious = true
	}

	return pattern
}

type AudioFingerprintingPattern struct {
	Timestamp    time.Time
	Patterns     []string
	RiskLevel    string
	IsSuspicious bool
}

func (s *AudioContextService) AnalyzeAudioFrequencySpectrum(data map[string]interface{}) *AudioFrequencySpectrum {
	spectrum := &AudioFrequencySpectrum{
		Timestamp: time.Now(),
		Bands:     make([]FrequencyBand, 0),
	}

	if frequencyData, ok := data["frequency_data"].([]interface{}); ok {
		bandSize := len(frequencyData) / 10
		if bandSize == 0 {
			bandSize = 1
		}

		for i := 0; i < 10; i++ {
			start := i * bandSize
			end := start + bandSize
			if end > len(frequencyData) {
				end = len(frequencyData)
			}

			var sum float64
			var count int
			for j := start; j < end; j++ {
				if val, ok := frequencyData[j].(float64); ok {
					sum += val
					count++
				}
			}

			if count > 0 {
				spectrum.Bands = append(spectrum.Bands, FrequencyBand{
					Index: i,
					AverageMagnitude: sum / float64(count),
					FrequencyRange: fmt.Sprintf("%d-%d", start, end),
				})
			}
		}
	}

	return spectrum
}

type AudioFrequencySpectrum struct {
	Timestamp time.Time
	Bands     []FrequencyBand
}

type FrequencyBand struct {
	Index             int
	AverageMagnitude float64
	FrequencyRange    string
}

func (s *AudioContextService) MatchAudioFingerprints(fp1ID, fp2ID string) float64 {
	fp1, exists1 := s.database.Get(fp1ID)
	fp2, exists2 := s.database.Get(fp2ID)

	if !exists1 || !exists2 {
		return 0
	}

	return s.similarityCalculator.Calculate(fp1, fp2).SimilarityScore
}

func (s *AudioContextService) DetectAudioAnomalies(fpID string) *AudioAnomalyResult {
	detection, err := s.DetectAnomalies(fpID)
	if err != nil {
		return &AudioAnomalyResult{
			Error: err.Error(),
		}
	}

	return &AudioAnomalyResult{
		FingerprintID:  detection.Fingerprint.FingerprintID,
		IsSuspicious:   detection.IsSuspicious,
		Severity:      detection.Severity,
		RiskScore:     detection.RiskScore,
		Indicators:    detection.AnomalyIndicators,
		Recommendations: detection.Recommendations,
	}
}

type AudioAnomalyResult struct {
	FingerprintID    string
	IsSuspicious     bool
	Severity         string
	RiskScore        float64
	Indicators       []string
	Recommendations  []string
	Error            string
}

func (s *AudioContextService) CleanupOldFingerprints(maxAge time.Duration) int {
	count := 0
	cutoff := time.Now().Add(-maxAge)

	for _, fp := range s.database.GetAll() {
		if fp.Timestamp.Before(cutoff) {
			s.RemoveFingerprint(fp.FingerprintID)
			count++
		}
	}

	return count
}

func (s *AudioContextService) GetFingerprintCount() int {
	return len(s.database.GetAll())
}

func (s *AudioContextService) ClearCache() {
	s.cache.Clear()
}

func (s *AudioContextService) SetConfig(config *AudioContextConfig) {
	s.config = config
}

func (s *AudioContextService) GetConfig() *AudioContextConfig {
	return s.config
}

func validateAudioFingerprintData(data map[string]interface{}) error {
	if data == nil {
		return fmt.Errorf("数据不能为空")
	}

	if _, ok := data["sample_rate"]; !ok {
		return fmt.Errorf("缺少必需字段: sample_rate")
	}

	return nil
}

func isValidSampleRate(rate int) bool {
	validRates := []int{8000, 11025, 12000, 16000, 22050, 24000, 32000, 44100, 48000, 96000}
	for _, r := range validRates {
		if r == rate {
			return true
		}
	}
	return false
}

func isValidChannelCount(count int) bool {
	return count >= 1 && count <= 32
}

func isValidFFTSize(size int) bool {
	sizes := []int{32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768}
	for _, s := range sizes {
		if s == size {
			return true
		}
	}
	return false
}

func detectSuspiciousPatterns(fp *model.AudioFingerprint) []string {
	patterns := make([]string, 0)

	if fp.ProcessingData.SpectralFlatness < 0.01 && len(fp.ProcessingData.FrequencyData) > 0 {
		patterns = append(patterns, "suspiciously_flat_spectrum")
	}

	if fp.RenderingConsistency > 0.999 {
		patterns = append(patterns, "perfect_rendering_consistency")
	}

	if fp.ContextProperties.SampleRate > 48000 && fp.IsSoftwareRenderer {
		patterns = append(patterns, "high_sample_rate_software_renderer")
	}

	re := regexp.MustCompile(`^[a-f0-9]+$`)
	if len(fp.AudioHash) > 0 && !re.MatchString(fp.AudioHash) {
		patterns = append(patterns, "invalid_hash_format")
	}

	return patterns
}

func calculateAudioEntropy(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	bins := 16
	histogram := make([]int, bins)
	maxVal := 0.0

	for _, val := range data {
		if math.Abs(val) > maxVal {
			maxVal = math.Abs(val)
		}
	}

	if maxVal == 0 {
		return 0
	}

	for _, val := range data {
		bin := int((math.Abs(val) / maxVal) * float64(bins-1))
		if bin >= bins {
			bin = bins - 1
		}
		histogram[bin]++
	}

	entropy := 0.0
	total := float64(len(data))

	for _, count := range histogram {
		if count > 0 {
			prob := float64(count) / total
			entropy -= prob * math.Log2(prob)
		}
	}

	return entropy / math.Log2(float64(bins))
}

func analyzeAudioQuality(fp *model.AudioFingerprint) float64 {
	quality := 0.0

	if fp.ContextProperties.SampleRate >= 44100 {
		quality += 25
	} else if fp.ContextProperties.SampleRate >= 22050 {
		quality += 15
	}

	if fp.ContextProperties.ChannelCount >= 2 {
		quality += 20
	}

	if len(fp.ProcessingData.FrequencyData) > 0 {
		quality += 25
	}

	if fp.ProcessingData.RMSAmplitude > 0 {
		quality += 15
	}

	if isValidFFTSize(fp.AnalyserConfig.FFTSize) {
		quality += 15
	}

	return quality
}

type AudioQualityAnalysis struct {
	QualityScore    float64
	FrequencyQuality float64
	TemporalQuality float64
	OverallQuality float64
}

func (s *AudioContextService) AnalyzeAudioQuality(fpID string) *AudioQualityAnalysis {
	fp, exists := s.database.Get(fpID)
	if !exists {
		return nil
	}

	analysis := &AudioQualityAnalysis{
		FrequencyQuality: analyzeFrequencyQuality(fp.ProcessingData.FrequencyData),
		TemporalQuality:  analyzeTemporalQuality(fp.ProcessingData.TimeDomainData),
	}

	analysis.QualityScore = analyzeAudioQuality(fp)

	analysis.OverallQuality = (analysis.QualityScore + analysis.FrequencyQuality + analysis.TemporalQuality) / 3.0

	return analysis
}

func analyzeFrequencyQuality(freqData []float64) float64 {
	if len(freqData) == 0 {
		return 0
	}

	entropy := calculateAudioEntropy(freqData)

	if entropy > 0.7 && entropy < 0.95 {
		return 100.0
	}

	return entropy * 100
}

func analyzeTemporalQuality(timeData []float64) float64 {
	if len(timeData) == 0 {
		return 0
	}

	quality := 0.0

	peakAmp := calculatePeakAmplitude(timeData)
	if peakAmp > 0 {
		quality += 50
	}

	rmsAmp := calculateRMSAmplitude(timeData)
	if rmsAmp > 0 {
		quality += 30
	}

	zcr := calculateZeroCrossingRate(timeData)
	if zcr > 0.01 && zcr < 0.5 {
		quality += 20
	}

	return quality
}
