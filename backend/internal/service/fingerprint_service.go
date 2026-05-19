package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

type FingerprintService struct {
	fingerprints  map[string]*FingerprintData
	componentDB   map[string]*FingerprintComponent
	behaviorCache map[string]*BehaviorPatternV3
	anomalyDB     map[string]*AnomalyPattern
	mu            sync.RWMutex
	maxCacheSize  int
	mlModel       *BotDetectionModel
}

type FingerprintData struct {
	FingerprintID   string
	Components      map[string]string
	Hash            string
	FirstSeen       time.Time
	LastSeen        time.Time
	RequestCount    int
	DeviceType      string
	Browser         string
	OS              string
	RiskScore       float64
	AnomalyFlags    []string
	IsSuspicious    bool
	MLPrediction    *MLPredictionResult
	BehaviorHistory []*BehaviorSnapshot
}

type FingerprintComponent struct {
	Name      string
	Value     string
	Weight    float64
	Timestamp time.Time
}

type BehaviorPatternV3 struct {
	FingerprintID  string
	RequestTimes   []time.Time
	RequestPaths   []string
	AvgInterval    float64
	Variance       float64
	IsHumanLike    bool
	PatternType    string
	Confidence     float64
	MLFeatures     []float64
	LastUpdated    time.Time
}

type AnomalyPattern struct {
	Type          string
	Pattern       string
	Weight        float64
	Description   string
	MLThreshold   float64
}

type MLPredictionResult struct {
	IsBot          bool
	Confidence     float64
	BotType        string
	Features       map[string]float64
	AnomalyScore   float64
	Explanation    []string
}

type BehaviorSnapshot struct {
	Timestamp    time.Time
	RequestPath  string
	Interval     time.Duration
	IsAnomalous  bool
}

func NewFingerprintService() *FingerprintService {
	service := &FingerprintService{
		fingerprints:  make(map[string]*FingerprintData),
		componentDB:   make(map[string]*FingerprintComponent),
		behaviorCache: make(map[string]*BehaviorPatternV3),
		anomalyDB:     make(map[string]*AnomalyPattern),
		maxCacheSize:  50000,
		mlModel:       NewBotDetectionModel(),
	}
	service.initializeAnomalyPatterns()
	service.initializeMLModel()
	return service
}

func (s *FingerprintService) initializeAnomalyPatterns() {
	s.anomalyDB["missing_canvas"] = &AnomalyPattern{
		Type:        "missing_canvas",
		Pattern:     "",
		Weight:      25.0,
		Description: "Canvas指纹缺失",
	}
	s.anomalyDB["no_plugins"] = &AnomalyPattern{
		Type:        "no_plugins",
		Pattern:     "",
		Weight:      15.0,
		Description: "无可用插件",
	}
	s.anomalyDB["abnormal_languages"] = &AnomalyPattern{
		Type:        "abnormal_languages",
		Pattern:     "",
		Weight:      10.0,
		Description: "语言设置异常",
	}
	s.anomalyDB["software_renderer"] = &AnomalyPattern{
		Type:        "software_renderer",
		Pattern:     "swiftshader|llvmpipe|software",
		Weight:      40.0,
		Description: "软件渲染器检测",
	}
	s.anomalyDB["vm_renderer"] = &AnomalyPattern{
		Type:        "vm_renderer",
		Pattern:     "vmware|virtualbox|parallels",
		Weight:      45.0,
		Description: "虚拟机渲染器",
	}
	s.anomalyDB["headless_ua"] = &AnomalyPattern{
		Type:        "headless_ua",
		Pattern:     "headless|phantom",
		Weight:      35.0,
		Description: "Headless浏览器UA",
	}
	s.anomalyDB["automation_framework"] = &AnomalyPattern{
		Type:        "automation_framework",
		Pattern:     "selenium|puppeteer|playwright",
		Weight:      50.0,
		Description: "自动化框架检测",
	}
	s.anomalyDB["regular_timing"] = &AnomalyPattern{
		Type:        "regular_timing",
		Pattern:     "",
		Weight:      30.0,
		Description: "请求时间间隔过于规律",
	}
	s.anomalyDB["high_frequency"] = &AnomalyPattern{
		Type:        "high_frequency",
		Pattern:     "",
		Weight:      35.0,
		Description: "请求频率异常高",
	}
	s.anomalyDB["suspicious_ip"] = &AnomalyPattern{
		Type:        "suspicious_ip",
		Pattern:     "",
		Weight:      20.0,
		Description: "可疑IP地址",
	}
}

func (s *FingerprintService) initializeMLModel() {
	s.mlModel.weights = map[string]float64{
		"canvas_consistency":   0.15,
		"webgl_renderer":       0.12,
		"request_timing":       0.18,
		"navigation_pattern":   0.14,
		"device_properties":    0.10,
		"behavior_entropy":     0.16,
		"session_regularity":   0.15,
	}
}

func (s *FingerprintService) GenerateFingerprint(userAgent string, headers map[string]string) (string, error) {
	components := s.collectComponents(userAgent, headers)
	fingerprintID := s.generateFingerprintID(components)
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	fingerprint := &FingerprintData{
		FingerprintID:   fingerprintID,
		Components:      components,
		Hash:            fingerprintID,
		FirstSeen:       time.Now(),
		LastSeen:        time.Now(),
		RequestCount:    1,
		RiskScore:       0.0,
		AnomalyFlags:    []string{},
		IsSuspicious:    false,
		BehaviorHistory: []*BehaviorSnapshot{},
	}
	
	s.fingerprints[fingerprintID] = fingerprint
	s.extractDeviceInfo(fingerprint)
	
	if len(s.fingerprints) > s.maxCacheSize {
		s.cleanupOldFingerprints()
	}
	
	return fingerprintID, nil
}

func (s *FingerprintService) collectComponents(userAgent string, headers map[string]string) map[string]string {
	components := make(map[string]string)
	
	components["user_agent"] = userAgent
	components["user_agent_hash"] = s.hashString(userAgent)
	
	if headers != nil {
		if lang := headers["Accept-Language"]; lang != "" {
			components["accept_language"] = lang
		}
		if encoding := headers["Accept-Encoding"]; encoding != "" {
			components["accept_encoding"] = encoding
		}
		if charset := headers["Accept-Charset"]; charset != "" {
			components["accept_charset"] = charset
		}
		if secChUa := headers["Sec-Ch-Ua"]; secChUa != "" {
			components["sec_ch_ua"] = secChUa
		}
		if secChPlatform := headers["Sec-Ch-Ua-Platform"]; secChPlatform != "" {
			components["sec_ch_platform"] = secChPlatform
		}
	}
	
	browser, version := s.parseBrowser(userAgent)
	components["browser"] = browser
	components["browser_version"] = version
	
	os := s.parseOS(userAgent)
	components["os"] = os
	
	deviceType := s.detectDeviceType(userAgent)
	components["device_type"] = deviceType
	
	return components
}

func (s *FingerprintService) generateFingerprintID(components map[string]string) string {
	hasher := sha256.New()
	keys := make([]string, 0, len(components))
	for k := range components {
		keys = append(keys, k)
	}
	for _, k := range keys {
		hasher.Write([]byte(k + ":" + components[k]))
	}
	return hex.EncodeToString(hasher.Sum(nil))[:32]
}

func (s *FingerprintService) hashString(str string) string {
	hasher := sha256.New()
	hasher.Write([]byte(str))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *FingerprintService) parseBrowser(userAgent string) (browser string, version string) {
	ua := strings.ToLower(userAgent)
	
	browserPatterns := []struct {
		Name    string
		Pattern *regexp.Regexp
	}{
		{"Edge", regexp.MustCompile(`edg[e]?/(\d+)`)},
		{"Chrome", regexp.MustCompile(`chrome/(\d+)`)},
		{"Firefox", regexp.MustCompile(`firefox/(\d+)`)},
		{"Safari", regexp.MustCompile(`safari/(\d+)`)},
		{"Opera", regexp.MustCompile(`opr/(\d+)`)},
		{"IE", regexp.MustCompile(`msie\s(\d+)`)},
	}
	
	for _, bp := range browserPatterns {
		if matches := bp.Pattern.FindStringSubmatch(ua); len(matches) > 1 {
			return bp.Name, matches[1]
		}
	}
	
	return "Unknown", "0"
}

func (s *FingerprintService) parseOS(userAgent string) string {
	ua := strings.ToLower(userAgent)
	
	if strings.Contains(ua, "windows") {
		return "Windows"
	}
	if strings.Contains(ua, "mac os") || strings.Contains(ua, "macos") {
		return "macOS"
	}
	if strings.Contains(ua, "linux") && !strings.Contains(ua, "android") {
		return "Linux"
	}
	if strings.Contains(ua, "android") {
		return "Android"
	}
	if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ios") {
		return "iOS"
	}
	
	return "Unknown"
}

func (s *FingerprintService) detectDeviceType(userAgent string) string {
	ua := strings.ToLower(userAgent)
	
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
			return "tablet"
		}
		return "mobile"
	}
	
	return "desktop"
}

func (s *FingerprintService) extractDeviceInfo(fp *FingerprintData) {
	if ua, ok := fp.Components["user_agent"]; ok {
		fp.Browser, _ = s.parseBrowser(ua)
		fp.OS = s.parseOS(ua)
		fp.DeviceType = s.detectDeviceType(ua)
	}
}

func (s *FingerprintService) ValidateFingerprint(fingerprint string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if len(fingerprint) < 16 || len(fingerprint) > 64 {
		return false
	}
	
	_, exists := s.fingerprints[fingerprint]
	return exists
}

func (s *FingerprintService) CompareFingerprints(fp1, fp2 string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	data1, exists1 := s.fingerprints[fp1]
	data2, exists2 := s.fingerprints[fp2]
	
	if !exists1 || !exists2 {
		return 0.0
	}
	
	similarity := 0.0
	totalComponents := 0
	
	for key, val1 := range data1.Components {
		totalComponents++
		if val2, ok := data2.Components[key]; ok {
			if val1 == val2 {
				similarity += 1.0
			}
		}
	}
	
	if totalComponents == 0 {
		return 0.0
	}
	
	return (similarity / float64(totalComponents)) * 100.0
}

func (s *FingerprintService) GetFingerprintComponents(userAgent string, headers map[string]string) (map[string]string, error) {
	return s.collectComponents(userAgent, headers), nil
}

func (s *FingerprintService) AnalyzeFingerprint(fingerprint string) (*FingerprintAnalysisV3, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	data, exists := s.fingerprints[fingerprint]
	if !exists {
		return nil, fmt.Errorf("fingerprint not found")
	}
	
	analysis := &FingerprintAnalysisV3{
		FingerprintID:   fingerprint,
		RiskScore:       data.RiskScore,
		IsSuspicious:    data.IsSuspicious,
		AnomalyFlags:    data.AnomalyFlags,
		DeviceInfo:      DeviceInfo{
			Browser:    data.Browser,
			OS:         data.OS,
			DeviceType: data.DeviceType,
		},
		ComponentCount: len(data.Components),
		LastSeen:       data.LastSeen,
		RequestCount:   data.RequestCount,
		MLPrediction:   data.MLPrediction,
		Recommendations: []string{},
	}
	
	if data.MLPrediction != nil {
		analysis.MLConfidence = data.MLPrediction.Confidence
	}
	
	return analysis, nil
}

type FingerprintAnalysisV3 struct {
	FingerprintID      string
	RiskScore          float64
	IsSuspicious       bool
	AnomalyFlags       []string
	DeviceInfo         DeviceInfo
	ComponentCount     int
	LastSeen           time.Time
	RequestCount       int
	MLPrediction       *MLPredictionResult
	MLConfidence       float64
	Recommendations    []string
}

type DeviceInfo struct {
	Browser    string
	OS         string
	DeviceType string
}

func (s *FingerprintService) DetectFingerprintAnomaly(fingerprint string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	data, exists := s.fingerprints[fingerprint]
	if !exists {
		return false, fmt.Errorf("fingerprint not found")
	}
	
	s.mu.RUnlock()
	anomalies := s.analyzeAnomalies(data)
	s.mu.RLock()
	
	return len(anomalies) > 0, nil
}

func (s *FingerprintService) analyzeAnomalies(data *FingerprintData) []string {
	anomalies := []string{}
	
	if data.Components["canvas_consistency"] == "" {
		anomalies = append(anomalies, "missing_canvas")
	}
	
	if data.Components["plugins"] == "" {
		anomalies = append(anomalies, "no_plugins")
	}
	
	if data.Components["accept_language"] == "" {
		anomalies = append(anomalies, "abnormal_languages")
	}
	
	ua := strings.ToLower(data.Components["user_agent"])
	if strings.Contains(ua, "headless") || strings.Contains(ua, "phantom") {
		anomalies = append(anomalies, "headless_ua")
	}
	
	if strings.Contains(ua, "selenium") || strings.Contains(ua, "puppeteer") || strings.Contains(ua, "playwright") {
		anomalies = append(anomalies, "automation_framework")
	}
	
	return anomalies
}

func (s *FingerprintService) UpdateFingerprintCache(fingerprint, userID string) error {
	return nil
}

func (s *FingerprintService) GetFingerprintFromCache(userID string) (string, error) {
	return "", nil
}

func (s *FingerprintService) DeleteFingerprintCache(userID string) error {
	return nil
}

func (s *FingerprintService) ClearAllFingerprints() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fingerprints = make(map[string]*FingerprintData)
	return nil
}

func (s *FingerprintService) GetFingerprintStats() (*FingerprintStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	stats := &FingerprintStats{
		TotalFingerprints: len(s.fingerprints),
		DeviceDistribution: make(map[string]int),
		BrowserDistribution: make(map[string]int),
		OSDistribution: make(map[string]int),
	}
	
	for _, fp := range s.fingerprints {
		stats.DeviceDistribution[fp.DeviceType]++
		stats.BrowserDistribution[fp.Browser]++
		stats.OSDistribution[fp.OS]++
	}
	
	return stats, nil
}

type FingerprintStats struct {
	TotalFingerprints   int
	DeviceDistribution   map[string]int
	BrowserDistribution map[string]int
	OSDistribution      map[string]int
}

func (s *FingerprintService) GenerateFingerprintString(seed string) string {
	hasher := sha256.New()
	hasher.Write([]byte(seed + time.Now().String()))
	return hex.EncodeToString(hasher.Sum(nil))[:32]
}

type BotDetectionModel struct {
	weights       map[string]float64
	trainingData  []TrainingSample
	threshold     float64
}

type TrainingSample struct {
	Features    map[string]float64
	Label       bool
	BotType     string
}

func NewBotDetectionModel() *BotDetectionModel {
	return &BotDetectionModel{
		weights:      make(map[string]float64),
		trainingData: []TrainingSample{},
		threshold:    0.7,
	}
}

func (m *BotDetectionModel) Predict(features map[string]float64) *MLPredictionResult {
	result := &MLPredictionResult{
		Features:     features,
		AnomalyScore: 0.0,
		Explanation:  []string{},
	}
	
	score := 0.0
	totalWeight := 0.0
	
	for feature, value := range features {
		if weight, exists := m.weights[feature]; exists {
			score += value * weight
			totalWeight += weight
		}
	}
	
	if totalWeight > 0 {
		normalizedScore := score / totalWeight
		result.Confidence = math.Min(normalizedScore, 1.0)
		result.IsBot = normalizedScore > m.threshold
		
		if result.IsBot {
			result.BotType = m.classifyBotType(features)
		}
	}
	
	return result
}

func (m *BotDetectionModel) classifyBotType(features map[string]float64) string {
	if features["automation_indicators"] > 0.8 {
		return "automation_tool"
	}
	if features["headless_browser"] > 0.7 {
		return "headless_browser"
	}
	if features["vm_indicators"] > 0.6 {
		return "virtual_machine"
	}
	if features["proxy_indicators"] > 0.5 {
		return "proxy_bot"
	}
	return "unknown_bot"
}

func (m *BotDetectionModel) Train(samples []TrainingSample) {
	m.trainingData = append(m.trainingData, samples...)
	m.updateWeights()
}

func (m *BotDetectionModel) updateWeights() {
	featureCounts := make(map[string]int)
	featurePositiveCounts := make(map[string]int)
	
	for _, sample := range m.trainingData {
		for feature := range sample.Features {
			featureCounts[feature]++
			if sample.Label {
				featurePositiveCounts[feature]++
			}
		}
	}
	
	for feature, count := range featureCounts {
		if count > 0 {
			m.weights[feature] = float64(featurePositiveCounts[feature]) / float64(count)
		}
	}
}

func (s *FingerprintService) CleanupExpiredCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cutoff := time.Now().Add(-24 * time.Hour)
	for id, fp := range s.fingerprints {
		if fp.LastSeen.Before(cutoff) {
			delete(s.fingerprints, id)
		}
	}
	
	for id, bp := range s.behaviorCache {
		if bp.LastUpdated.Before(cutoff) {
			delete(s.behaviorCache, id)
		}
	}
}

func (s *FingerprintService) cleanupOldFingerprints() {
	cutoff := time.Now().Add(-24 * time.Hour)
	count := 0
	targetDelete := len(s.fingerprints) - s.maxCacheSize + 1000
	
	for id, fp := range s.fingerprints {
		if fp.LastSeen.Before(cutoff) && count < targetDelete {
			delete(s.fingerprints, id)
			count++
		}
	}
}

func (s *FingerprintService) RecordBehavior(fingerprintID string, path string, interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	bp, exists := s.behaviorCache[fingerprintID]
	if !exists {
		bp = &BehaviorPatternV3{
			FingerprintID: fingerprintID,
			RequestTimes: []time.Time{},
			RequestPaths: []string{},
			MLFeatures:   make([]float64, 8),
			LastUpdated:  time.Now(),
		}
		s.behaviorCache[fingerprintID] = bp
	}
	
	bp.RequestTimes = append(bp.RequestTimes, time.Now())
	bp.RequestPaths = append(bp.RequestPaths, path)
	bp.LastUpdated = time.Now()
	
	if len(bp.RequestTimes) > 1000 {
		bp.RequestTimes = bp.RequestTimes[len(bp.RequestTimes)-500:]
		bp.RequestPaths = bp.RequestPaths[len(bp.RequestPaths)-500:]
	}
	
	if len(bp.RequestTimes) >= 2 {
		bp.AvgInterval = s.calculateAverageInterval(bp.RequestTimes)
		bp.Variance = s.calculateIntervalVariance(bp.RequestTimes, time.Duration(bp.AvgInterval))
		bp.IsHumanLike = s.checkHumanLikeness(bp)
		bp.MLFeatures = s.extractMLFeatures(bp)
	}
	
	s.updateFingerprintRiskScore(fingerprintID)
}

func (s *FingerprintService) calculateAverageInterval(times []time.Time) float64 {
	if len(times) < 2 {
		return 0
	}
	
	total := 0.0
	for i := 1; i < len(times); i++ {
		total += float64(times[i].Sub(times[i-1]).Milliseconds())
	}
	
	return total / float64(len(times)-1)
}

func (s *FingerprintService) calculateIntervalVariance(times []time.Time, avg time.Duration) float64 {
	if len(times) < 2 {
		return 0
	}
	
	var sumSq float64
	for i := 1; i < len(times); i++ {
		diff := float64(times[i].Sub(times[i-1])) - float64(avg)
		sumSq += diff * diff
	}
	
	return math.Sqrt(sumSq / float64(len(times)-1))
}

func (s *FingerprintService) checkHumanLikeness(bp *BehaviorPatternV3) bool {
	if bp.AvgInterval < 100 {
		return false
	}
	
	if bp.Variance < 50 && bp.AvgInterval < 2000 {
		return false
	}
	
	return bp.AvgInterval > 1000 && bp.Variance > 100
}

func (s *FingerprintService) extractMLFeatures(bp *BehaviorPatternV3) []float64 {
	features := make([]float64, 8)
	
	if len(bp.RequestTimes) >= 2 {
		features[0] = bp.AvgInterval / 1000.0
		features[1] = bp.Variance / 1000.0
		
		uniquePaths := make(map[string]bool)
		for _, path := range bp.RequestPaths {
			uniquePaths[path] = true
		}
		features[2] = float64(len(uniquePaths)) / float64(len(bp.RequestPaths))
		
		features[3] = s.calculateEntropy(bp.RequestPaths)
		
		features[4] = float64(bp.RequestCount()) / (time.Since(bp.LastUpdated).Minutes() + 1)
	}
	
	return features
}

func (s *FingerprintService) calculateEntropy(paths []string) float64 {
	if len(paths) == 0 {
		return 0
	}
	
	freq := make(map[string]int)
	for _, path := range paths {
		freq[path]++
	}
	
	entropy := 0.0
	for _, count := range freq {
		p := float64(count) / float64(len(paths))
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	
	return entropy
}

func (bp *BehaviorPatternV3) RequestCount() int {
	return len(bp.RequestTimes)
}

func (s *FingerprintService) updateFingerprintRiskScore(fingerprintID string) {
	fp, exists := s.fingerprints[fingerprintID]
	if !exists {
		return
	}
	
	bp, hasBehavior := s.behaviorCache[fingerprintID]
	
	riskScore := 0.0
	anomalies := []string{}
	
	if !bp.IsHumanLike && hasBehavior {
		riskScore += 25.0
		anomalies = append(anomalies, "inhuman_behavior")
	}
	
	if bp.AvgInterval > 0 && bp.AvgInterval < 500 {
		riskScore += 30.0
		anomalies = append(anomalies, "high_frequency_requests")
	}
	
	if bp.Variance > 0 && bp.Variance < 100 && bp.AvgInterval < 3000 {
		riskScore += 20.0
		anomalies = append(anomalies, "regular_timing_pattern")
	}
	
	features := make(map[string]float64)
	if hasBehavior {
		features["request_timing"] = bp.AvgInterval / 10000.0
		features["behavior_entropy"] = s.calculateEntropy(bp.RequestPaths) / 10.0
		features["session_regularity"] = bp.Variance / 1000.0
		
		mlResult := s.mlModel.Predict(features)
		if mlResult.IsBot {
			riskScore += mlResult.Confidence * 30
			fp.MLPrediction = mlResult
		}
	}
	
	fp.RiskScore = math.Min(riskScore, 100.0)
	fp.AnomalyFlags = anomalies
	fp.IsSuspicious = riskScore > 30
}

func (s *FingerprintService) AnalyzeBehaviorPattern(fingerprintID string) (*BehaviorAnalysisResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	bp, exists := s.behaviorCache[fingerprintID]
	if !exists {
		return nil, fmt.Errorf("behavior pattern not found")
	}
	
	result := &BehaviorAnalysisResult{
		FingerprintID:  fingerprintID,
		AvgInterval:   bp.AvgInterval,
		Variance:      bp.Variance,
		IsHumanLike:   bp.IsHumanLike,
		PatternType:   bp.PatternType,
		Confidence:    bp.Confidence,
		RequestCount:  len(bp.RequestTimes),
		UniquePaths:   s.countUniquePaths(bp.RequestPaths),
		Entropy:       s.calculateEntropy(bp.RequestPaths),
	}
	
	return result, nil
}

type BehaviorAnalysisResult struct {
	FingerprintID string
	AvgInterval   float64
	Variance      float64
	IsHumanLike   bool
	PatternType   string
	Confidence    float64
	RequestCount  int
	UniquePaths   int
	Entropy       float64
}

func (s *FingerprintService) countUniquePaths(paths []string) int {
	unique := make(map[string]bool)
	for _, path := range paths {
		unique[path] = true
	}
	return len(unique)
}

func (s *FingerprintService) DetectAnomalousBehavior(fingerprintID string) (bool, []string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	bp, exists := s.behaviorCache[fingerprintID]
	if !exists {
		return false, nil, nil
	}
	
	anomalies := []string{}
	
	if !bp.IsHumanLike {
		anomalies = append(anomalies, "non_human_timing")
	}
	
	if bp.AvgInterval > 0 && bp.AvgInterval < 200 {
		anomalies = append(anomalies, "extremely_fast_requests")
	}
	
	if bp.Variance > 0 && bp.Variance < 50 && bp.AvgInterval < 3000 {
		anomalies = append(anomalies, "mechanical_regularity")
	}
	
	if len(bp.RequestTimes) > 100 && bp.AvgInterval < 500 {
		anomalies = append(anomalies, "sustained_high_frequency")
	}
	
	entropy := s.calculateEntropy(bp.RequestPaths)
	if entropy < 1.0 && len(bp.RequestPaths) > 10 {
		anomalies = append(anomalies, "low_navigation_entropy")
	}
	
	return len(anomalies) > 0, anomalies, nil
}

func (s *FingerprintService) GetDeviceFingerprintV3(userAgent string, envData map[string]interface{}) (string, *DeviceFingerprintV3, error) {
	components := s.collectComponents(userAgent, nil)
	
	if envData != nil {
		if canvasHash, ok := envData["canvas_hash"].(string); ok {
			components["canvas_hash"] = canvasHash
		}
		if webglRenderer, ok := envData["webgl_renderer"].(string); ok {
			components["webgl_renderer"] = webglRenderer
		}
		if webglVendor, ok := envData["webgl_vendor"].(string); ok {
			components["webgl_vendor"] = webglVendor
		}
		if fonts, ok := envData["fonts"].([]string); ok {
			components["fonts_hash"] = s.hashString(strings.Join(fonts, ","))
		}
		if audioHash, ok := envData["audio_hash"].(string); ok {
			components["audio_hash"] = audioHash
		}
	}
	
	fingerprintID := s.generateFingerprintID(components)
	
	v3 := &DeviceFingerprintV3{
		FingerprintID:  fingerprintID,
		Components:     components,
		Version:        "3.0",
		GenerationTime:  time.Now(),
		HashAlgorithm:  "SHA256",
	}
	
	s.mu.Lock()
	s.fingerprints[fingerprintID] = &FingerprintData{
		FingerprintID: fingerprintID,
		Components:    components,
		Hash:          fingerprintID,
		FirstSeen:     time.Now(),
		LastSeen:      time.Now(),
	}
	s.mu.Unlock()
	
	return fingerprintID, v3, nil
}

type DeviceFingerprintV3 struct {
	FingerprintID string
	Components    map[string]string
	Version       string
	GenerationTime time.Time
	HashAlgorithm string
}

func (s *FingerprintService) RecognizeBehaviorPattern(fingerprintID string) (string, float64, error) {
	s.mu.RLock()
	bp, exists := s.behaviorCache[fingerprintID]
	s.mu.RUnlock()
	
	if !exists {
		return "", 0, nil
	}
	
	patternType := "unknown"
	confidence := 0.0
	
	if bp.AvgInterval > 0 && bp.AvgInterval < 300 && len(bp.RequestTimes) > 20 {
		patternType = "automated_scraping"
		confidence = 0.9
	} else if bp.Variance < 100 && bp.AvgInterval > 500 && bp.AvgInterval < 2000 {
		patternType = "rate_limited_bot"
		confidence = 0.85
	} else if bp.IsHumanLike {
		patternType = "human"
		confidence = 0.95
	} else if len(bp.RequestPaths) == 1 && len(bp.RequestTimes) > 50 {
		patternType = "single_endpoint_abuse"
		confidence = 0.8
	} else if s.calculateEntropy(bp.RequestPaths) < 2.0 {
		patternType = "low_diversity_access"
		confidence = 0.75
	}
	
	s.mu.Lock()
	bp.PatternType = patternType
	bp.Confidence = confidence
	s.mu.Unlock()
	
	return patternType, confidence, nil
}
