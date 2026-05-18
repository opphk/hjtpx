package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type FingerprintData struct {
	FingerprintID   string    `json:"fingerprint_id"`
	DeviceID        string    `json:"device_id"`
	IP              string    `json:"ip"`
	UserAgent       string    `json:"user_agent"`
	Accept          string    `json:"accept"`
	AcceptLanguage  string    `json:"accept_language"`
	AcceptEncoding  string    `json:"accept_encoding"`
	Connection      string    `json:"connection"`
	ScreenInfo      string    `json:"screen_info"`
	Timezone        string    `json:"timezone"`
	CanvasHash      string    `json:"canvas_hash"`
	WebGLHash       string    `json:"webgl_hash"`
	AudioHash       string    `json:"audio_hash"`
	WebGLParamsHash string    `json:"webgl_params_hash"`
	FontHash        string    `json:"font_hash"`
	PluginHash      string    `json:"plugin_hash"`
	HardwareHash    string    `json:"hardware_hash"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
	RequestCount    int       `json:"request_count"`
	IsBlacklisted   bool      `json:"is_blacklisted"`
	BlacklistReason string    `json:"blacklist_reason"`
	RiskScore       float64   `json:"risk_score"`
}

type CanvasFingerprint struct {
	Canvas2DHash    string `json:"canvas_2d_hash"`
	CanvasWebGLHash string `json:"canvas_webgl_hash"`
	CanvasBitmapHash string `json:"canvas_bitmap_hash"`
	RenderingQuality string `json:"rendering_quality"`
}

type WebGLFingerprint struct {
	Renderer       string `json:"renderer"`
	Vendor         string `json:"vendor"`
	Version        string `json:"version"`
	ParametersHash string `json:"parameters_hash"`
	ExtensionsHash  string `json:"extensions_hash"`
	ShaderPrecision string `json:"shader_precision"`
}

type AudioFingerprint struct {
	Hash     string  `json:"hash"`
	Latency  float64 `json:"latency"`
	Channels int     `json:"channels"`
	SampleRate int   `json:"sample_rate"`
}

type BehaviorPattern struct {
	RequestTimes []time.Time `json:"request_times"`
	RequestPaths []string    `json:"request_paths"`
	Methods      []string    `json:"methods"`
	StartTime    time.Time   `json:"start_time"`
}

type FingerprintService struct {
	fingerprints map[string]*FingerprintData
	behaviors    map[string]*BehaviorPattern
	mu           sync.RWMutex
	blacklist    map[string]string
	blacklistMu  sync.RWMutex
}

func NewFingerprintService() *FingerprintService {
	return &FingerprintService{
		fingerprints: make(map[string]*FingerprintData),
		behaviors:    make(map[string]*BehaviorPattern),
		blacklist:    make(map[string]string),
	}
}

func (s *FingerprintService) GenerateFingerprint(r *http.Request, additionalData map[string]string) string {
	hasher := sha256.New()

	ip := s.getRealIP(r)
	hasher.Write([]byte(ip))

	userAgent := r.UserAgent()
	hasher.Write([]byte(userAgent))

	accept := r.Header.Get("Accept")
	hasher.Write([]byte(accept))

	acceptLang := r.Header.Get("Accept-Language")
	hasher.Write([]byte(acceptLang))

	acceptEnc := r.Header.Get("Accept-Encoding")
	hasher.Write([]byte(acceptEnc))

	connection := r.Header.Get("Connection")
	hasher.Write([]byte(connection))

	headers := []string{
		"DNT",
		"Upgrade-Insecure-Requests",
		"Sec-Fetch-Dest",
		"Sec-Fetch-Mode",
		"Sec-Fetch-Site",
		"Sec-Fetch-User",
		"Cache-Control",
	}
	for _, h := range headers {
		hasher.Write([]byte(r.Header.Get(h)))
	}

	for k, v := range additionalData {
		hasher.Write([]byte(k + ":" + v))
	}

	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash)
}

func (s *FingerprintService) getRealIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *FingerprintService) ExtractFingerprintData(r *http.Request, additionalData map[string]string) *FingerprintData {
	fingerprintID := s.GenerateFingerprint(r, additionalData)

	s.mu.Lock()
	defer s.mu.Unlock()

	if fp, exists := s.fingerprints[fingerprintID]; exists {
		fp.LastSeen = time.Now()
		fp.RequestCount++
		return fp
	}

	fp := &FingerprintData{
		FingerprintID:  fingerprintID,
		IP:             s.getRealIP(r),
		UserAgent:      r.UserAgent(),
		Accept:         r.Header.Get("Accept"),
		AcceptLanguage: r.Header.Get("Accept-Language"),
		AcceptEncoding: r.Header.Get("Accept-Encoding"),
		Connection:     r.Header.Get("Connection"),
		ScreenInfo:     additionalData["screen_info"],
		Timezone:       additionalData["timezone"],
		CanvasHash:     additionalData["canvas_hash"],
		WebGLHash:      additionalData["webgl_hash"],
		FirstSeen:      time.Now(),
		LastSeen:       time.Now(),
		RequestCount:   1,
		IsBlacklisted:  false,
		RiskScore:      0,
	}

	s.fingerprints[fingerprintID] = fp
	s.trackBehavior(fingerprintID, r)

	return fp
}

func (s *FingerprintService) trackBehavior(fingerprintID string, r *http.Request) {
	behavior, exists := s.behaviors[fingerprintID]
	if !exists {
		behavior = &BehaviorPattern{
			RequestTimes: make([]time.Time, 0),
			RequestPaths: make([]string, 0),
			Methods:      make([]string, 0),
			StartTime:    time.Now(),
		}
		s.behaviors[fingerprintID] = behavior
	}

	behavior.RequestTimes = append(behavior.RequestTimes, time.Now())
	behavior.RequestPaths = append(behavior.RequestPaths, r.URL.Path)
	behavior.Methods = append(behavior.Methods, r.Method)

	if len(behavior.RequestTimes) > 1000 {
		behavior.RequestTimes = behavior.RequestTimes[len(behavior.RequestTimes)-1000:]
		behavior.RequestPaths = behavior.RequestPaths[len(behavior.RequestPaths)-1000:]
		behavior.Methods = behavior.Methods[len(behavior.Methods)-1000:]
	}
}

func (s *FingerprintService) DetectAnomaly(fingerprintID string) (bool, string, float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	behavior, exists := s.behaviors[fingerprintID]
	if !exists || len(behavior.RequestTimes) < 5 {
		return false, "", 0
	}

	riskScore := 0.0
	indicators := []string{}

	requestFreq := s.analyzeRequestFrequency(behavior.RequestTimes)
	if requestFreq > 10 {
		riskScore += 20
		indicators = append(indicators, "请求频率异常")
	}

	if s.detectRegularIntervals(behavior.RequestTimes) {
		riskScore += 25
		indicators = append(indicators, "请求间隔过于规律")
	}

	if s.detectPathPattern(behavior.RequestPaths) {
		riskScore += 30
		indicators = append(indicators, "路径模式异常")
	}

	userAgentConsistency := s.checkUserAgentConsistency(fingerprintID)
	if !userAgentConsistency {
		riskScore += 15
		indicators = append(indicators, "User-Agent不一致")
	}

	return riskScore > 30, strings.Join(indicators, "; "), riskScore
}

func (s *FingerprintService) analyzeRequestFrequency(times []time.Time) float64 {
	if len(times) < 2 {
		return 0
	}

	window := 60 * time.Second
	count := 0
	now := times[len(times)-1]

	for i := len(times) - 1; i >= 0; i-- {
		if now.Sub(times[i]) <= window {
			count++
		} else {
			break
		}
	}

	return float64(count)
}

func (s *FingerprintService) detectRegularIntervals(times []time.Time) bool {
	if len(times) < 10 {
		return false
	}

	intervals := make([]float64, 0)
	for i := 1; i < len(times); i++ {
		intervals = append(intervals, float64(times[i].Sub(times[i-1]).Milliseconds()))
	}

	if len(intervals) < 5 {
		return false
	}

	avg := 0.0
	for _, interval := range intervals {
		avg += interval
	}
	avg /= float64(len(intervals))

	avgVariance := 0.0
	for _, interval := range intervals {
		avgVariance += (interval - avg) * (interval - avg)
	}
	avgVariance /= float64(len(intervals))

	if avg == 0 {
		return false
	}

	cv := avgVariance / (avg * avg)

	return cv < 0.01 && avg > 0
}

func (s *FingerprintService) detectPathPattern(paths []string) bool {
	if len(paths) < 5 {
		return false
	}

	pathCounts := make(map[string]int)
	for _, p := range paths {
		pathCounts[p]++
	}

	for _, count := range pathCounts {
		if count > len(paths)*3/4 {
			return true
		}
	}

	patternMatch := 0
	for i := 1; i < len(paths); i++ {
		if paths[i] == paths[i-1] {
			patternMatch++
		}
	}

	return float64(patternMatch) > float64(len(paths))*0.8
}

func (s *FingerprintService) checkUserAgentConsistency(fingerprintID string) bool {
	fp, exists := s.fingerprints[fingerprintID]
	if !exists {
		return true
	}

	return fp.RequestCount > 100 && fp.IsBlacklisted
}

func (s *FingerprintService) AddToBlacklist(fingerprintID, reason string) {
	s.blacklistMu.Lock()
	defer s.blacklistMu.Unlock()

	s.blacklist[fingerprintID] = reason

	s.mu.Lock()
	if fp, exists := s.fingerprints[fingerprintID]; exists {
		fp.IsBlacklisted = true
		fp.BlacklistReason = reason
	}
	s.mu.Unlock()
}

func (s *FingerprintService) RemoveFromBlacklist(fingerprintID string) {
	s.blacklistMu.Lock()
	defer s.blacklistMu.Unlock()

	delete(s.blacklist, fingerprintID)

	s.mu.Lock()
	if fp, exists := s.fingerprints[fingerprintID]; exists {
		fp.IsBlacklisted = false
		fp.BlacklistReason = ""
	}
	s.mu.Unlock()
}

func (s *FingerprintService) IsBlacklisted(fingerprintID string) (bool, string) {
	s.blacklistMu.RLock()
	defer s.blacklistMu.RUnlock()

	reason, exists := s.blacklist[fingerprintID]
	return exists, reason
}

func (s *FingerprintService) GetFingerprint(fingerprintID string) (*FingerprintData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fp, exists := s.fingerprints[fingerprintID]
	return fp, exists
}

func (s *FingerprintService) GetAllFingerprints() []*FingerprintData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*FingerprintData, 0, len(s.fingerprints))
	for _, fp := range s.fingerprints {
		result = append(result, fp)
	}
	return result
}

func (s *FingerprintService) CalculateRiskScore(fp *FingerprintData) float64 {
	riskScore := 0.0

	if fp.IsBlacklisted {
		return 100.0
	}

	if fp.RequestCount > 1000 {
		riskScore += 10
	}

	anomaly, _, _ := s.DetectAnomaly(fp.FingerprintID)
	if anomaly {
		riskScore += 40
	}

	return riskScore
}

func (s *FingerprintService) CleanupOldData() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)

	for id, fp := range s.fingerprints {
		if fp.LastSeen.Before(cutoff) {
			delete(s.fingerprints, id)
			delete(s.behaviors, id)
		}
	}

	s.blacklistMu.Lock()
	defer s.blacklistMu.Unlock()
}

func (s *FingerprintService) GenerateDeviceID(r *http.Request, additionalData map[string]string) string {
	hasher := sha256.New()

	ip := s.getRealIP(r)
	hasher.Write([]byte(ip))

	userAgent := r.UserAgent()
	hasher.Write([]byte(userAgent))

	if screen := additionalData["screen_info"]; screen != "" {
		hasher.Write([]byte(screen))
	}

	if timezone := additionalData["timezone"]; timezone != "" {
		hasher.Write([]byte(timezone))
	}

	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash)[:16]
}

func (s *FingerprintService) AnalyzeRequestPattern(fingerprintID string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	behavior, exists := s.behaviors[fingerprintID]
	if !exists {
		return nil
	}

	result := make(map[string]interface{})

	if len(behavior.RequestTimes) > 0 {
		intervals := make([]float64, 0)
		for i := 1; i < len(behavior.RequestTimes); i++ {
			intervals = append(intervals, behavior.RequestTimes[i].Sub(behavior.RequestTimes[i-1]).Seconds())
		}
		result["avg_interval"] = avg(intervals)
		result["min_interval"] = min(intervals)
		result["max_interval"] = max(intervals)
	}

	pathCounts := make(map[string]int)
	for _, path := range behavior.RequestPaths {
		pathCounts[path]++
	}
	result["path_distribution"] = pathCounts

	methodCounts := make(map[string]int)
	for _, method := range behavior.Methods {
		methodCounts[method]++
	}
	result["method_distribution"] = methodCounts

	result["total_requests"] = len(behavior.RequestTimes)
	result["duration_hours"] = time.Since(behavior.StartTime).Hours()

	return result
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values {
		if v < m {
			m = v
		}
	}
	return m
}

func max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values {
		if v > m {
			m = v
		}
	}
	return m
}

type FingerprintAnalysisResult struct {
	Fingerprint     *FingerprintData
	IsAnomaly       bool
	AnomalyReason   string
	RiskScore       float64
	PatternAnalysis map[string]interface{}
}

func (s *FingerprintService) AnalyzeFingerprint(r *http.Request, additionalData map[string]string) *FingerprintAnalysisResult {
	fp := s.ExtractFingerprintData(r, additionalData)
	isAnomaly, reason, riskScore := s.DetectAnomaly(fp.FingerprintID)
	patternAnalysis := s.AnalyzeRequestPattern(fp.FingerprintID)

	return &FingerprintAnalysisResult{
		Fingerprint:     fp,
		IsAnomaly:       isAnomaly,
		AnomalyReason:   reason,
		RiskScore:       riskScore,
		PatternAnalysis: patternAnalysis,
	}
}

func (s *FingerprintService) GetBlacklist() map[string]string {
	s.blacklistMu.RLock()
	defer s.blacklistMu.RUnlock()

	result := make(map[string]string)
	for k, v := range s.blacklist {
		result[k] = v
	}
	return result
}

func (s *FingerprintService) ClearBlacklist() {
	s.blacklistMu.Lock()
	defer s.blacklistMu.Unlock()
	s.blacklist = make(map[string]string)

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, fp := range s.fingerprints {
		fp.IsBlacklisted = false
		fp.BlacklistReason = ""
	}
}

func (s *FingerprintService) GetStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_fingerprints"] = len(s.fingerprints)
	stats["total_behaviors"] = len(s.behaviors)

	blacklisted := 0
	for _, fp := range s.fingerprints {
		if fp.IsBlacklisted {
			blacklisted++
		}
	}
	stats["blacklisted_count"] = blacklisted

	totalRequests := 0
	for _, fp := range s.fingerprints {
		totalRequests += fp.RequestCount
	}
	stats["total_requests"] = totalRequests

	return stats
}

func (s *FingerprintService) FindSimilarFingerprints(targetFP string, threshold float64) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, 0)

	target, exists := s.fingerprints[targetFP]
	if !exists {
		return result
	}

	for id, fp := range s.fingerprints {
		if id == targetFP {
			continue
		}

		similarity := 0.0
		if fp.IP == target.IP {
			similarity += 0.3
		}
		if fp.UserAgent == target.UserAgent {
			similarity += 0.3
		}
		if fp.AcceptLanguage == target.AcceptLanguage {
			similarity += 0.2
		}
		if fp.Accept == target.Accept {
			similarity += 0.2
		}

		if similarity >= threshold {
			result = append(result, id)
		}
	}

	return result
}

func (s *FingerprintService) ExtractEnhancedFingerprintData(r *http.Request, additionalData map[string]string) *FingerprintData {
	fingerprintID := s.GenerateEnhancedFingerprint(r, additionalData)

	s.mu.Lock()
	defer s.mu.Unlock()

	if fp, exists := s.fingerprints[fingerprintID]; exists {
		fp.LastSeen = time.Now()
		fp.RequestCount++
		return fp
	}

	fp := &FingerprintData{
		FingerprintID:   fingerprintID,
		IP:              s.getRealIP(r),
		UserAgent:       r.UserAgent(),
		Accept:          r.Header.Get("Accept"),
		AcceptLanguage:  r.Header.Get("Accept-Language"),
		AcceptEncoding:  r.Header.Get("Accept-Encoding"),
		Connection:      r.Header.Get("Connection"),
		ScreenInfo:      additionalData["screen_info"],
		Timezone:        additionalData["timezone"],
		CanvasHash:      additionalData["canvas_hash"],
		WebGLHash:       additionalData["webgl_hash"],
		AudioHash:       additionalData["audio_hash"],
		WebGLParamsHash: additionalData["webgl_params_hash"],
		FontHash:        additionalData["font_hash"],
		PluginHash:      additionalData["plugin_hash"],
		HardwareHash:    additionalData["hardware_hash"],
		FirstSeen:       time.Now(),
		LastSeen:        time.Now(),
		RequestCount:    1,
		IsBlacklisted:   false,
		RiskScore:       0,
	}

	s.fingerprints[fingerprintID] = fp
	s.trackBehavior(fingerprintID, r)

	return fp
}

func (s *FingerprintService) GenerateEnhancedFingerprint(r *http.Request, additionalData map[string]string) string {
	hasher := sha256.New()

	ip := s.getRealIP(r)
	hasher.Write([]byte(ip))

	userAgent := r.UserAgent()
	hasher.Write([]byte(userAgent))

	accept := r.Header.Get("Accept")
	hasher.Write([]byte(accept))

	acceptLang := r.Header.Get("Accept-Language")
	hasher.Write([]byte(acceptLang))

	acceptEnc := r.Header.Get("Accept-Encoding")
	hasher.Write([]byte(acceptEnc))

	connection := r.Header.Get("Connection")
	hasher.Write([]byte(connection))

	headers := []string{
		"DNT",
		"Upgrade-Insecure-Requests",
		"Sec-Fetch-Dest",
		"Sec-Fetch-Mode",
		"Sec-Fetch-Site",
		"Sec-Fetch-User",
		"Cache-Control",
		"Accept-CH",
		"Sec-CH-UA",
		"Sec-CH-UA-Mobile",
		"Sec-CH-UA-Platform",
	}
	for _, h := range headers {
		hasher.Write([]byte(r.Header.Get(h)))
	}

	for k, v := range additionalData {
		hasher.Write([]byte(k + ":" + v))
	}

	if canvasHash := additionalData["canvas_hash"]; canvasHash != "" {
		hasher.Write([]byte(canvasHash))
	}

	if webglHash := additionalData["webgl_hash"]; webglHash != "" {
		hasher.Write([]byte(webglHash))
	}

	if audioHash := additionalData["audio_hash"]; audioHash != "" {
		hasher.Write([]byte(audioHash))
	}

	if webglParamsHash := additionalData["webgl_params_hash"]; webglParamsHash != "" {
		hasher.Write([]byte(webglParamsHash))
	}

	if fontHash := additionalData["font_hash"]; fontHash != "" {
		hasher.Write([]byte(fontHash))
	}

	if pluginHash := additionalData["plugin_hash"]; pluginHash != "" {
		hasher.Write([]byte(pluginHash))
	}

	if hardwareHash := additionalData["hardware_hash"]; hardwareHash != "" {
		hasher.Write([]byte(hardwareHash))
	}

	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash)
}

func (s *FingerprintService) AnalyzeCanvasFingerprint(canvasData map[string]interface{}) *CanvasFingerprint {
	result := &CanvasFingerprint{}

	if canvas2D, ok := canvasData["canvas_2d"].(string); ok && canvas2D != "" {
		hasher := sha256.New()
		hasher.Write([]byte(canvas2D))
		result.Canvas2DHash = hex.EncodeToString(hasher.Sum(nil))
	}

	if canvasWebGL, ok := canvasData["canvas_webgl"].(string); ok && canvasWebGL != "" {
		hasher := sha256.New()
		hasher.Write([]byte(canvasWebGL))
		result.CanvasWebGLHash = hex.EncodeToString(hasher.Sum(nil))
	}

	if canvasBitmap, ok := canvasData["canvas_bitmap"].(string); ok && canvasBitmap != "" {
		hasher := sha256.New()
		hasher.Write([]byte(canvasBitmap))
		result.CanvasBitmapHash = hex.EncodeToString(hasher.Sum(nil))
	}

	if quality, ok := canvasData["rendering_quality"].(string); ok {
		result.RenderingQuality = quality
	}

	return result
}

func (s *FingerprintService) AnalyzeWebGLFingerprint(webglData map[string]interface{}) *WebGLFingerprint {
	result := &WebGLFingerprint{}

	if renderer, ok := webglData["renderer"].(string); ok {
		result.Renderer = renderer
	}

	if vendor, ok := webglData["vendor"].(string); ok {
		result.Vendor = vendor
	}

	if version, ok := webglData["version"].(string); ok {
		result.Version = version
	}

	if params, ok := webglData["parameters"].(string); ok && params != "" {
		hasher := sha256.New()
		hasher.Write([]byte(params))
		result.ParametersHash = hex.EncodeToString(hasher.Sum(nil))
	}

	if extensions, ok := webglData["extensions"].(string); ok && extensions != "" {
		hasher := sha256.New()
		hasher.Write([]byte(extensions))
		result.ExtensionsHash = hex.EncodeToString(hasher.Sum(nil))
	}

	if precision, ok := webglData["shader_precision"].(string); ok {
		result.ShaderPrecision = precision
	}

	return result
}

func (s *FingerprintService) AnalyzeAudioFingerprint(audioData map[string]interface{}) *AudioFingerprint {
	result := &AudioFingerprint{}

	if hash, ok := audioData["audio_hash"].(string); ok && hash != "" {
		result.Hash = hash
	}

	if latency, ok := audioData["latency"].(float64); ok {
		result.Latency = latency
	}

	if channels, ok := audioData["channels"].(int); ok {
		result.Channels = channels
	}

	if sampleRate, ok := audioData["sample_rate"].(int); ok {
		result.SampleRate = sampleRate
	}

	return result
}

func (s *FingerprintService) DetectFingerprintAnomalies(fp *FingerprintData) (bool, []string, float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	anomalies := []string{}
	riskScore := 0.0

	if fp.CanvasHash == "" && fp.WebGLHash == "" {
		anomalies = append(anomalies, "missing_advanced_fingerprints")
		riskScore += 15
	}

	if fp.AudioHash == "" {
		anomalies = append(anomalies, "missing_audio_fingerprint")
		riskScore += 10
	}

	if fp.FontHash == "" {
		anomalies = append(anomalies, "missing_font_fingerprint")
		riskScore += 8
	}

	if fp.HardwareHash == "" {
		anomalies = append(anomalies, "missing_hardware_fingerprint")
		riskScore += 5
	}

	similarFPs := s.findSimilarFingerprintsInternal(fp.FingerprintID)
	if len(similarFPs) > 3 {
		anomalies = append(anomalies, "too_many_similar_fingerprints")
		riskScore += 25
	}

	return len(anomalies) > 0, anomalies, riskScore
}

func (s *FingerprintService) findSimilarFingerprintsInternal(targetFP string) []string {
	result := make([]string, 0)

	target, exists := s.fingerprints[targetFP]
	if !exists {
		return result
	}

	for id, fp := range s.fingerprints {
		if id == targetFP {
			continue
		}

		similarity := 0.0
		if fp.IP == target.IP {
			similarity += 0.3
		}
		if fp.UserAgent == target.UserAgent {
			similarity += 0.3
		}
		if fp.CanvasHash != "" && fp.CanvasHash == target.CanvasHash {
			similarity += 0.2
		}
		if fp.WebGLHash != "" && fp.WebGLHash == target.WebGLHash {
			similarity += 0.2
		}

		if similarity >= 0.6 {
			result = append(result, id)
		}
	}

	return result
}

type EnhancedFingerprintAnalysis struct {
	Fingerprint           *FingerprintData
	CanvasAnalysis       *CanvasFingerprint
	WebGLAnalysis        *WebGLFingerprint
	AudioAnalysis        *AudioFingerprint
	IsAnomaly            bool
	AnomalyReasons       []string
	RiskScore            float64
	PatternAnalysis      map[string]interface{}
	SimilarFingerprints  []string
}

func (s *FingerprintService) PerformEnhancedAnalysis(r *http.Request, additionalData map[string]string) *EnhancedFingerprintAnalysis {
	fp := s.ExtractEnhancedFingerprintData(r, additionalData)

	canvasAnalysis := &CanvasFingerprint{}
	if canvasData, ok := additionalData["canvas_data"]; ok {
		var parsedData map[string]interface{}
		if err := json.Unmarshal([]byte(canvasData), &parsedData); err == nil {
			canvasAnalysis = s.AnalyzeCanvasFingerprint(parsedData)
		}
	}

	webglAnalysis := &WebGLFingerprint{}
	if webglData, ok := additionalData["webgl_data"]; ok {
		var parsedData map[string]interface{}
		if err := json.Unmarshal([]byte(webglData), &parsedData); err == nil {
			webglAnalysis = s.AnalyzeWebGLFingerprint(parsedData)
		}
	}

	audioAnalysis := &AudioFingerprint{}
	if audioData, ok := additionalData["audio_data"]; ok {
		var parsedData map[string]interface{}
		if err := json.Unmarshal([]byte(audioData), &parsedData); err == nil {
			audioAnalysis = s.AnalyzeAudioFingerprint(parsedData)
		}
	}

	isAnomaly, anomalyReasons, anomalyScore := s.DetectFingerprintAnomalies(fp)

	baseRiskScore := s.CalculateRiskScore(fp)
	totalRiskScore := baseRiskScore + anomalyScore

	patternAnalysis := s.AnalyzeRequestPattern(fp.FingerprintID)

	similarFPs := s.findSimilarFingerprintsInternal(fp.FingerprintID)

	return &EnhancedFingerprintAnalysis{
		Fingerprint:          fp,
		CanvasAnalysis:       canvasAnalysis,
		WebGLAnalysis:        webglAnalysis,
		AudioAnalysis:        audioAnalysis,
		IsAnomaly:            isAnomaly,
		AnomalyReasons:       anomalyReasons,
		RiskScore:            math.Min(totalRiskScore, 100),
		PatternAnalysis:      patternAnalysis,
		SimilarFingerprints: similarFPs,
	}
}
