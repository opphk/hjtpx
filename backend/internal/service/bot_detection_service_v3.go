package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type BotDetectionV3Config struct {
	EnableAIDetection    bool
	EnableDeviceAnalysis bool
	EnableDeepBrowserCheck bool
	EnableAdvancedAutomationDetection bool
	MLModelEnabled       bool
	NeuralNetworkEnabled bool
}

type BotDetectionV3Result struct {
	IsBot               bool
	ShouldBlock         bool
	RiskScore           float64
	Confidence          float64
	Reasons             []string
	ChallengeType       string
	DetectionMethods    []string
	DeviceFingerprint   string
	BehaviorScore       float64
	MLPrediction        float64
	NeuralNetworkOutput *NeuralNetworkResult
}

type NeuralNetworkResult struct {
	InputVector  []float64
	OutputVector []float64
	Prediction   float64
	Confidence   float64
	LayerOutputs [][]float64
}

type DeviceBehaviorData struct {
	DeviceType        string
	OS                string
	Browser           string
	ScreenResolution  string
	TouchPoints       int
	HasWebGL          bool
	WebGLRenderer     string
	HasCanvas         bool
	CanvasFingerprint string
	AudioFingerprint  string
	Fonts             []string
	Plugins           []string
}

type BotDetectionV3Service struct {
	config              BotDetectionV3Config
	fingerprints        map[string]*BotFingerprintV3
	behaviors           map[string]*BotBehaviorV3
	deviceData          map[string]*DeviceBehaviorData
	mlModel             *SimpleMLModel
	neuralNet          *FeedForwardNeuralNetwork
	knownBotSignatures  map[string]*BotSignature
	adaptiveThresholds *AdaptiveThresholds
	mu                  sync.RWMutex
}

type BotFingerprintV3 struct {
	FingerprintID string
	FirstSeen     time.Time
	LastSeen      time.Time
	RequestCount  int
	RiskScore     float64
	IsBlacklisted bool
	UserAgent     string
	IP            string
	DeviceData    *DeviceBehaviorData
	MLFeatures    []float64
}

type BotBehaviorV3 struct {
	IP              string
	RequestTimes    []time.Time
	RequestPaths    []string
	Methods         []string
	RequestCount    int
	LastActivity    time.Time
	AvgInterval     float64
	IsRegular       bool
	MouseMovements  []MouseMovement
	KeyboardPatterns []KeyboardPattern
	TouchEvents     []TouchEvent
}

type MouseMovement struct {
	X, Y         float64
	Timestamp     time.Time
	Velocity     float64
	Acceleration float64
}

type TouchEvent struct {
	X, Y        float64
	Timestamp   time.Time
	TouchType   string
	Pressure    float64
}

type BotSignature struct {
	Name        string
	Type        string
	Patterns    []*regexp.Regexp
	Indicators  []string
	Weight      float64
	MLFeatures  []float64
}

type AdaptiveThresholds struct {
	BaseThreshold   float64
	CurrentThreshold float64
	AdjustmentRate  float64
	LastAdjustment  time.Time
}

type SimpleMLModel struct {
	weights    [][]float64
	bias       []float64
	features   []string
	trained    bool
}

type FeedForwardNeuralNetwork struct {
	inputSize     int
	hiddenSize    int
	outputSize    int
	weightsInput  [][]float64
	weightsHidden [][]float64
	biasHidden    []float64
	biasOutput   []float64
	activated    bool
}

func NewBotDetectionV3Service(config BotDetectionV3Config) *BotDetectionV3Service {
	service := &BotDetectionV3Service{
		config:              config,
		fingerprints:        make(map[string]*BotFingerprintV3),
		behaviors:           make(map[string]*BotBehaviorV3),
		deviceData:          make(map[string]*DeviceBehaviorData),
		knownBotSignatures:  make(map[string]*BotSignature),
		adaptiveThresholds: &AdaptiveThresholds{
			BaseThreshold:    0.6,
			CurrentThreshold: 0.6,
			AdjustmentRate:   0.01,
			LastAdjustment:   time.Now(),
		},
	}

	if config.EnableAIDetection || config.MLModelEnabled {
		service.mlModel = NewSimpleMLModel()
	}

	if config.NeuralNetworkEnabled {
		service.neuralNet = NewFeedForwardNeuralNetwork(10, 20, 1)
	}

	service.initializeBotSignatures()
	return service
}

func NewSimpleMLModel() *SimpleMLModel {
	return &SimpleMLModel{
		weights:  make([][]float64, 10),
		bias:     make([]float64, 1),
		features: []string{"request_rate", "session_duration", "mouse_variance", "keyboard_speed", "touch_accuracy", "canvas_unique", "webgl_software", "timezone_offset", "language_variance", "automation_flags"},
	}
}

func (m *SimpleMLModel) Train(features [][]float64, labels []float64) error {
	if len(features) != len(labels) {
		return fmt.Errorf("features and labels length mismatch")
	}

	for i := range m.weights {
		m.weights[i] = make([]float64, 1)
		for j := range m.weights[i] {
			m.weights[i][j] = (float64(i) + 1) / float64(len(m.weights))
		}
	}

	for i := range m.bias {
		m.bias[i] = 0.5
	}

	m.trained = true
	return nil
}

func (m *SimpleMLModel) Predict(features []float64) float64 {
	if !m.trained || len(features) == 0 {
		return 0.5
	}

	var sum float64
	for i := 0; i < len(features) && i < len(m.weights); i++ {
		sum += features[i] * m.weights[i][0]
	}
	sum += m.bias[0]

	prediction := 1.0 / (1.0 + math.Exp(-sum))
	return math.Max(0, math.Min(1, prediction))
}

func NewFeedForwardNeuralNetwork(inputSize, hiddenSize, outputSize int) *FeedForwardNeuralNetwork {
	nn := &FeedForwardNeuralNetwork{
		inputSize:   inputSize,
		hiddenSize:  hiddenSize,
		outputSize:  outputSize,
		activated:   false,
	}

	nn.weightsInput = make([][]float64, inputSize)
	for i := range nn.weightsInput {
		nn.weightsInput[i] = make([]float64, hiddenSize)
		for j := range nn.weightsInput[i] {
			nn.weightsInput[i][j] = (float64(i*j) + 1) / float64(inputSize*hiddenSize)
		}
	}

	nn.weightsHidden = make([][]float64, hiddenSize)
	for i := range nn.weightsHidden {
		nn.weightsHidden[i] = make([]float64, outputSize)
		for j := range nn.weightsHidden[i] {
			nn.weightsHidden[i][j] = (float64(i) + 1) / float64(hiddenSize*outputSize)
		}
	}

	nn.biasHidden = make([]float64, hiddenSize)
	nn.biasOutput = make([]float64, outputSize)

	nn.activated = true
	return nn
}

func (nn *FeedForwardNeuralNetwork) Forward(input []float64) *NeuralNetworkResult {
	if !nn.activated || len(input) != nn.inputSize {
		return &NeuralNetworkResult{Prediction: 0.5, Confidence: 0.0}
	}

	hiddenLayer := make([]float64, nn.hiddenSize)
	for j := 0; j < nn.hiddenSize; j++ {
		sum := nn.biasHidden[j]
		for i := 0; i < nn.inputSize; i++ {
			sum += input[i] * nn.weightsInput[i][j]
		}
		hiddenLayer[j] = nn.relu(sum)
	}

	outputLayer := make([]float64, nn.outputSize)
	for k := 0; k < nn.outputSize; k++ {
		sum := nn.biasOutput[k]
		for j := 0; j < nn.hiddenSize; j++ {
			sum += hiddenLayer[j] * nn.weightsHidden[j][k]
		}
		outputLayer[k] = nn.sigmoid(sum)
	}

	prediction := outputLayer[0]
	if len(outputLayer) > 0 {
		prediction = outputLayer[0]
	}

	return &NeuralNetworkResult{
		InputVector:  input,
		OutputVector: outputLayer,
		Prediction:   prediction,
		Confidence:   1.0 - math.Abs(prediction-0.5)*2,
		LayerOutputs: [][]float64{hiddenLayer},
	}
}

func (nn *FeedForwardNeuralNetwork) relu(x float64) float64 {
	if x > 0 {
		return x
	}
	return 0
}

func (nn *FeedForwardNeuralNetwork) sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func (s *BotDetectionV3Service) initializeBotSignatures() {
	s.knownBotSignatures["selenium"] = &BotSignature{
		Name: "Selenium",
		Type: "automation",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)selenium`),
			regexp.MustCompile(`(?i)webdriver.*selenium`),
		},
		Indicators: []string{"__selenium_evaluate", "__webdriver_script_fn"},
		Weight:     0.85,
		MLFeatures: []float64{0.9, 0.1, 0.8, 0.9, 0.2, 0.3, 0.7, 0.5, 0.3, 0.95},
	}

	s.knownBotSignatures["puppeteer"] = &BotSignature{
		Name: "Puppeteer",
		Type: "automation",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)puppeteer`),
			regexp.MustCompile(`(?i)headless.*chrome`),
		},
		Indicators: []string{"$cdc_asdjflasutopfhvcZLmcfl_", "__puppeteer_evaluation_script"},
		Weight:     0.88,
		MLFeatures: []float64{0.95, 0.05, 0.9, 0.95, 0.1, 0.2, 0.8, 0.4, 0.2, 0.98},
	}

	s.knownBotSignatures["playwright"] = &BotSignature{
		Name: "Playwright",
		Type: "automation",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)playwright`),
		},
		Indicators: []string{"__playwright__", "__pw_api_hooks__"},
		Weight:     0.87,
		MLFeatures: []float64{0.92, 0.08, 0.85, 0.9, 0.15, 0.25, 0.75, 0.45, 0.25, 0.96},
	}

	s.knownBotSignatures["headless"] = &BotSignature{
		Name: "Headless Browser",
		Type: "automation",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)headless`),
		},
		Indicators: []string{"navigator.webdriver", "headless_detected"},
		Weight:     0.75,
		MLFeatures: []float64{0.8, 0.2, 0.7, 0.8, 0.3, 0.4, 0.6, 0.5, 0.4, 0.85},
	}

	s.knownBotSignatures["webdriver"] = &BotSignature{
		Name: "WebDriver",
		Type: "automation",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)webdriver`),
		},
		Indicators: []string{"webdriver", "__webdriver_evaluate"},
		Weight:     0.82,
		MLFeatures: []float64{0.85, 0.15, 0.75, 0.85, 0.25, 0.35, 0.65, 0.5, 0.35, 0.88},
	}
}

func (s *BotDetectionV3Service) DetectBotV3(r *http.Request, additionalData map[string]interface{}) *BotDetectionV3Result {
	result := &BotDetectionV3Result{
		IsBot:              false,
		ShouldBlock:        false,
		RiskScore:          0.0,
		Confidence:         0.0,
		Reasons:            []string{},
		DetectionMethods:   []string{},
	}

	ip := s.getClientIP(r)
	userAgent := r.UserAgent()
	fingerprintID := s.generateFingerprintIDV3(ip, userAgent, additionalData)

	s.mu.Lock()
	deviceData := s.extractDeviceData(r, additionalData)
	s.deviceData[fingerprintID] = deviceData

	s.recordRequestV3(ip, r)

	if s.config.EnableAIDetection || s.config.MLModelEnabled {
		s.detectWithML(ip, userAgent, additionalData, result)
	}

	if s.config.NeuralNetworkEnabled && s.neuralNet != nil {
		s.detectWithNeuralNetwork(ip, userAgent, additionalData, result)
	}

	if s.config.EnableDeviceAnalysis {
		s.analyzeDeviceBehavior(deviceData, result)
	}

	if s.config.EnableDeepBrowserCheck {
		s.performDeepBrowserAnalysis(r, additionalData, result)
	}

	if s.config.EnableAdvancedAutomationDetection {
		s.detectAdvancedAutomation(r, additionalData, result)
	}

	s.analyzeBehaviorPatternsV3(ip, result)

	s.updateFingerprintV3(fingerprintID, userAgent, ip, deviceData, result)

	s.applyAdaptiveThreshold(result)

	s.mu.Unlock()

	return result
}

func (s *BotDetectionV3Service) extractDeviceData(r *http.Request, data map[string]interface{}) *DeviceBehaviorData {
	deviceData := &DeviceBehaviorData{}

	deviceData.DeviceType = "unknown"
	if strings.Contains(strings.ToLower(r.UserAgent()), "mobile") {
		deviceData.DeviceType = "mobile"
	} else if strings.Contains(strings.ToLower(r.UserAgent()), "tablet") {
		deviceData.DeviceType = "tablet"
	} else {
		deviceData.DeviceType = "desktop"
	}

	if data != nil {
		if screen, ok := data["screen_resolution"].(string); ok {
			deviceData.ScreenResolution = screen
		}
		if touch, ok := data["touch_points"].(float64); ok {
			deviceData.TouchPoints = int(touch)
		}
		if webgl, ok := data["webgl_renderer"].(string); ok {
			deviceData.WebGLRenderer = webgl
			deviceData.HasWebGL = true
		}
		if canvas, ok := data["canvas_fingerprint"].(string); ok {
			deviceData.CanvasFingerprint = canvas
			deviceData.HasCanvas = true
		}
		if audio, ok := data["audio_fingerprint"].(string); ok {
			deviceData.AudioFingerprint = audio
		}
	}

	return deviceData
}

func (s *BotDetectionV3Service) recordRequestV3(ip string, r *http.Request) {
	behavior, exists := s.behaviors[ip]
	if !exists {
		behavior = &BotBehaviorV3{
			IP:           ip,
			RequestTimes: []time.Time{},
			RequestPaths: []string{},
			Methods:      []string{},
		}
		s.behaviors[ip] = behavior
	}

	now := time.Now()
	behavior.RequestTimes = append(behavior.RequestTimes, now)
	behavior.RequestPaths = append(behavior.RequestPaths, r.URL.Path)
	behavior.Methods = append(behavior.Methods, r.Method)
	behavior.RequestCount++
	behavior.LastActivity = now

	if len(behavior.RequestTimes) > 100 {
		behavior.RequestTimes = behavior.RequestTimes[len(behavior.RequestTimes)-100:]
	}
}

func (s *BotDetectionV3Service) detectWithML(ip string, userAgent string, data map[string]interface{}, result *BotDetectionV3Result) {
	result.DetectionMethods = append(result.DetectionMethods, "ml_model")

	features := s.extractMLFeatures(ip, userAgent, data)
	result.MLPrediction = s.mlModel.Predict(features)

	if result.MLPrediction > 0.7 {
		result.RiskScore += result.MLPrediction * 0.3
		result.Reasons = append(result.Reasons, "ML model detected bot-like behavior")
	}
}

func (s *BotDetectionV3Service) extractMLFeatures(ip string, userAgent string, data map[string]interface{}) []float64 {
	features := make([]float64, 10)

	s.mu.RLock()
	behavior := s.behaviors[ip]
	s.mu.RUnlock()

	if behavior != nil && len(behavior.RequestTimes) > 1 {
		var totalInterval float64
		for i := 1; i < len(behavior.RequestTimes); i++ {
			totalInterval += behavior.RequestTimes[i].Sub(behavior.RequestTimes[i-1]).Seconds()
		}
		avgInterval := totalInterval / float64(len(behavior.RequestTimes)-1)
		features[0] = math.Min(1.0, 1.0/avgInterval)

		if len(behavior.RequestTimes) > 0 {
			sessionDuration := time.Since(behavior.RequestTimes[0]).Seconds()
			features[1] = math.Min(1.0, sessionDuration/3600)
		}
	}

	if data != nil {
		if variance, ok := data["mouse_variance"].(float64); ok {
			features[2] = math.Min(1.0, variance/1000)
		}
		if speed, ok := data["keyboard_speed"].(float64); ok {
			features[3] = math.Min(1.0, speed/500)
		}
		if accuracy, ok := data["touch_accuracy"].(float64); ok {
			features[4] = accuracy
		}
		if unique, ok := data["canvas_unique"].(float64); ok {
			features[5] = unique
		}
		if software, ok := data["webgl_software"].(float64); ok {
			features[6] = software
		}
		if tz, ok := data["timezone_offset"].(float64); ok {
			features[7] = math.Min(1.0, math.Abs(tz)/720)
		}
		if langVar, ok := data["language_variance"].(float64); ok {
			features[8] = langVar
		}
		if auto, ok := data["automation_flags"].(float64); ok {
			features[9] = auto
		}
	}

	return features
}

func (s *BotDetectionV3Service) detectWithNeuralNetwork(ip string, userAgent string, data map[string]interface{}, result *BotDetectionV3Result) {
	result.DetectionMethods = append(result.DetectionMethods, "neural_network")

	features := s.extractMLFeatures(ip, userAgent, data)
	nnResult := s.neuralNet.Forward(features)
	result.NeuralNetworkOutput = nnResult

	if nnResult.Prediction > 0.7 {
		result.RiskScore += nnResult.Prediction * 0.25
		result.Reasons = append(result.Reasons, "Neural network detected bot-like behavior")
		result.Confidence += nnResult.Confidence * 0.2
	}
}

func (s *BotDetectionV3Service) analyzeDeviceBehavior(deviceData *DeviceBehaviorData, result *BotDetectionV3Result) {
	result.DetectionMethods = append(result.DetectionMethods, "device_analysis")

	if deviceData.HasWebGL && deviceData.WebGLRenderer != "" {
		lowerRenderer := strings.ToLower(deviceData.WebGLRenderer)
		if strings.Contains(lowerRenderer, "swiftshader") ||
			strings.Contains(lowerRenderer, "llvmpipe") ||
			strings.Contains(lowerRenderer, "software") {
			result.RiskScore += 0.15
			result.Reasons = append(result.Reasons, "Software WebGL renderer detected")
		}
	}

	if deviceData.HasCanvas && deviceData.CanvasFingerprint != "" {
		hash := sha256.Sum256([]byte(deviceData.CanvasFingerprint))
		hashStr := base64.StdEncoding.EncodeToString(hash[:])[:12]
		if s.isCommonCanvasHash(hashStr) {
			result.RiskScore += 0.1
			result.Reasons = append(result.Reasons, "Common canvas fingerprint")
		}
	}

	if deviceData.TouchPoints == 0 && deviceData.DeviceType == "mobile" {
		result.RiskScore += 0.05
		result.Reasons = append(result.Reasons, "Mobile device without touch support")
	}
}

func (s *BotDetectionV3Service) isCommonCanvasHash(hash string) bool {
	commonHashes := map[string]bool{
		"a1b2c3d4e5f6": true,
		"1234567890ab": true,
		"ffffffffffff": true,
		"000000000000": true,
	}
	return commonHashes[hash]
}

func (s *BotDetectionV3Service) performDeepBrowserAnalysis(r *http.Request, data map[string]interface{}, result *BotDetectionV3Result) {
	result.DetectionMethods = append(result.DetectionMethods, "deep_browser_check")

	deepIndicators := map[string]float64{
		"navigator.webdriver": 0.9,
		"chrome.runtime":      0.1,
		"__webdriver_evaluate": 0.85,
		"__selenium_evaluate":  0.85,
		"__fxdriver_evaluate":  0.8,
		"$cdc_asdjflasutopfhvcZLmcfl_": 0.88,
		"$chrome_asyncScriptInfo": 0.85,
	}

	if data != nil {
		for indicator, weight := range deepIndicators {
			if val, ok := data[indicator]; ok {
				if boolVal, ok := val.(bool); ok && boolVal {
					result.RiskScore += weight
					result.Reasons = append(result.Reasons, fmt.Sprintf("Deep browser indicator: %s", indicator))
				}
			}
		}
	}

	if r.Header.Get("Sec-Ch-Ua-Platform") == "" && r.Header.Get("Sec-Ch-Ua") != "" {
		result.RiskScore += 0.1
		result.Reasons = append(result.Reasons, "Missing platform header")
	}

	if r.Header.Get("Sec-Fetch-Site") == "" {
		result.RiskScore += 0.05
		result.Reasons = append(result.Reasons, "Missing fetch site header")
	}

	if r.Header.Get("Accept-Language") == "" {
		result.RiskScore += 0.05
		result.Reasons = append(result.Reasons, "Missing accept language header")
	}
}

func (s *BotDetectionV3Service) detectAdvancedAutomation(r *http.Request, data map[string]interface{}, result *BotDetectionV3Result) {
	result.DetectionMethods = append(result.DetectionMethods, "advanced_automation")

	for sigName, signature := range s.knownBotSignatures {
		for _, pattern := range signature.Patterns {
			if pattern.MatchString(r.UserAgent()) {
				result.RiskScore += signature.Weight
				result.Reasons = append(result.Reasons, fmt.Sprintf("Detected %s automation tool", sigName))
				break
			}
		}
	}

	if data != nil {
		for sigName, signature := range s.knownBotSignatures {
			for _, indicator := range signature.Indicators {
				if val, ok := data[indicator]; ok {
					if boolVal, ok := val.(bool); ok && boolVal {
						result.RiskScore += signature.Weight * 0.8
						result.Reasons = append(result.Reasons, fmt.Sprintf("Automation indicator from %s", sigName))
						break
					}
				}
			}
		}
	}

	s.checkTimingAnomalies(r, result)
}

func (s *BotDetectionV3Service) checkTimingAnomalies(r *http.Request, result *BotDetectionV3Result) {
	if timingData, ok := r.Header["X-Request-Timing"]; ok && len(timingData) > 0 {
		var loadTime float64
		fmt.Sscanf(timingData[0], "%f", &loadTime)

		if loadTime < 50 {
			result.RiskScore += 0.2
			result.Reasons = append(result.Reasons, "Suspiciously fast page load")
		}
	}
}

func (s *BotDetectionV3Service) analyzeBehaviorPatternsV3(ip string, result *BotDetectionV3Result) {
	s.mu.RLock()
	behavior := s.behaviors[ip]
	s.mu.RUnlock()

	if behavior == nil || len(behavior.RequestTimes) < 5 {
		return
	}

	result.DetectionMethods = append(result.DetectionMethods, "behavior_patterns")

	var intervals []float64
	for i := 1; i < len(behavior.RequestTimes); i++ {
		interval := behavior.RequestTimes[i].Sub(behavior.RequestTimes[i-1]).Seconds() * 1000
		intervals = append(intervals, interval)
	}

	if len(intervals) > 1 {
		mean := 0.0
		for _, v := range intervals {
			mean += v
		}
		mean /= float64(len(intervals))

		variance := 0.0
		for _, v := range intervals {
			variance += (v - mean) * (v - mean)
		}
		variance /= float64(len(intervals))
		stdDev := math.Sqrt(variance)

		cv := stdDev / (mean + 1)
		if cv < 0.05 && mean < 500 {
			result.RiskScore += 0.25
			result.Reasons = append(result.Reasons, "Uniform request timing detected")
			result.BehaviorScore = 0.9
		}

		if mean < 100 && len(behavior.RequestTimes) > 30 {
			result.RiskScore += 0.3
			result.Reasons = append(result.Reasons, "Extremely high request rate")
			result.BehaviorScore = 0.95
		}
	}

	if behavior.RequestCount > 100 && result.BehaviorScore < 0.5 {
		result.RiskScore += 0.15
		result.Reasons = append(result.Reasons, "High request count with regular patterns")
		result.BehaviorScore = 0.7
	}
}

func (s *BotDetectionV3Service) updateFingerprintV3(fingerprintID, userAgent, ip string, deviceData *DeviceBehaviorData, result *BotDetectionV3Result) {
	fingerprint, exists := s.fingerprints[fingerprintID]
	if !exists {
		fingerprint = &BotFingerprintV3{
			FingerprintID: fingerprintID,
			FirstSeen:     time.Now(),
			LastSeen:     time.Now(),
			RequestCount: 0,
			RiskScore:    0.0,
			UserAgent:    userAgent,
			IP:           ip,
			DeviceData:   deviceData,
			MLFeatures:   []float64{},
		}
		s.fingerprints[fingerprintID] = fingerprint
	}

	fingerprint.RequestCount++
	fingerprint.LastSeen = time.Now()
	fingerprint.RiskScore = result.RiskScore

	if fingerprint.RequestCount > 50 {
		fingerprint.MLFeatures = s.extractMLFeatures(ip, userAgent, nil)
	}
}

func (s *BotDetectionV3Service) applyAdaptiveThreshold(result *BotDetectionV3Result) {
	threshold := s.adaptiveThresholds.CurrentThreshold

	result.RiskScore = math.Min(1.0, result.RiskScore)
	result.Confidence = math.Min(1.0, result.Confidence)

	if result.RiskScore >= threshold {
		result.IsBot = true

		if result.RiskScore >= threshold*1.2 {
			result.ShouldBlock = true
			result.ChallengeType = "block"
		} else if result.RiskScore >= threshold*1.1 {
			result.ChallengeType = "captcha"
		} else {
			result.ChallengeType = "js_challenge"
		}
	}

	s.adjustThreshold(result.RiskScore)
}

func (s *BotDetectionV3Service) adjustThreshold(currentRisk float64) {
	now := time.Now()
	if now.Sub(s.adaptiveThresholds.LastAdjustment) < 5*time.Minute {
		return
	}

	if currentRisk > 0.8 {
		s.adaptiveThresholds.CurrentThreshold -= s.adaptiveThresholds.AdjustmentRate
	} else if currentRisk < 0.3 {
		s.adaptiveThresholds.CurrentThreshold += s.adaptiveThresholds.AdjustmentRate
	}

	if s.adaptiveThresholds.CurrentThreshold < 0.4 {
		s.adaptiveThresholds.CurrentThreshold = 0.4
	}
	if s.adaptiveThresholds.CurrentThreshold > 0.8 {
		s.adaptiveThresholds.CurrentThreshold = 0.8
	}

	s.adaptiveThresholds.LastAdjustment = now
}

func (s *BotDetectionV3Service) getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}

	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	return r.RemoteAddr
}

func (s *BotDetectionV3Service) generateFingerprintIDV3(ip string, userAgent string, data map[string]interface{}) string {
	hasher := sha256.New()
	hasher.Write([]byte(ip))
	hasher.Write([]byte(userAgent))

	if data != nil {
		if deviceType, ok := data["device_type"].(string); ok {
			hasher.Write([]byte(deviceType))
		}
		if screen, ok := data["screen_resolution"].(string); ok {
			hasher.Write([]byte(screen))
		}
	}

	hash := hasher.Sum(nil)
	return base64.StdEncoding.EncodeToString(hash)
}

func (s *BotDetectionV3Service) GetStatistics() BotDetectionStatistics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := BotDetectionStatistics{
		TotalFingerprints: len(s.fingerprints),
		ActiveBehaviors:   len(s.behaviors),
		ActiveDevices:     len(s.deviceData),
		DetectionMethods:  make(map[string]int),
	}

	for _, fp := range s.fingerprints {
		if fp.RiskScore > 0.5 {
			stats.HighRiskCount++
		}
		if fp.IsBlacklisted {
			stats.BlacklistedCount++
		}
	}

	for method, count := range stats.DetectionMethods {
		_ = method
		_ = count
	}

	return stats
}

type BotDetectionStatistics struct {
	TotalFingerprints int            `json:"total_fingerprints"`
	ActiveBehaviors   int            `json:"active_behaviors"`
	ActiveDevices     int            `json:"active_devices"`
	HighRiskCount     int            `json:"high_risk_count"`
	BlacklistedCount  int            `json:"blacklisted_count"`
	DetectionMethods  map[string]int `json:"detection_methods"`
}

func (s *BotDetectionV3Service) ExportModel(ctx context.Context) ([]byte, error) {
	modelData := struct {
		Weights    [][]float64   `json:"weights"`
		Bias      []float64     `json:"bias"`
		Features  []string       `json:"features"`
		Threshold float64       `json:"threshold"`
	}{
		Weights:   s.mlModel.weights,
		Bias:      s.mlModel.bias,
		Features:  s.mlModel.features,
		Threshold: s.adaptiveThresholds.CurrentThreshold,
	}

	return json.Marshal(modelData)
}

func (s *BotDetectionV3Service) ImportModel(ctx context.Context, data []byte) error {
	var modelData struct {
		Weights    [][]float64 `json:"weights"`
		Bias      []float64   `json:"bias"`
		Features  []string     `json:"features"`
		Threshold float64     `json:"threshold"`
	}

	if err := json.Unmarshal(data, &modelData); err != nil {
		return err
	}

	s.mlModel.weights = modelData.Weights
	s.mlModel.bias = modelData.Bias
	s.mlModel.features = modelData.Features
	s.mlModel.trained = true

	s.adaptiveThresholds.CurrentThreshold = modelData.Threshold

	return nil
}
