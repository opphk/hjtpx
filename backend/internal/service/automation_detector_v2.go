package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type AutomationDetectorV2 struct {
	headlessPatterns   *HeadlessPatternDetector
	automationSignals  *AutomationSignalAnalyzer
	mouseAnalyzer      *AdvancedMouseAnalyzer
	keyboardAnalyzer   *AdvancedKeyboardAnalyzer
}

type HeadlessPatternDetector struct {
	KnownHeadlessIndicators []string
	SuspiciousRenders       []string
}

type AutomationSignalAnalyzer struct {
	AutomationTools map[string][]string
	TimingPatterns   map[string]float64
}

type AdvancedMouseAnalyzer struct {
	BehavioralThresholds *MouseThresholds
	PatternDetectors     []MousePatternDetector
}

type MouseThresholds struct {
	MinSpeedForHuman  float64
	MaxSpeedForHuman  float64
	MinPauseDuration  float64
	MaxLinearRatio    float64
}

type MousePatternDetector struct {
	PatternName   string
	DetectionFunc func([]MouseMovement) float64
}

type AdvancedKeyboardAnalyzer struct {
	TypingPatterns     map[string]*TypingPattern
	ErrorRateThresholds *ErrorRateThresholds
}

type TypingPattern struct {
	PatternName string
	IsBotIndicator bool
	MinVariance    float64
	MaxVariance    float64
}

type ErrorRateThresholds struct {
	HumanMinErrorRate   float64
	HumanMaxErrorRate   float64
	BotMinErrorRate     float64
	BotMaxErrorRate     float64
}

func NewAutomationDetectorV2() *AutomationDetectorV2 {
	return &AutomationDetectorV2{
		headlessPatterns: &HeadlessPatternDetector{
			KnownHeadlessIndicators: []string{
				"webdriver",
				"__webdriver_script_function",
				"__webdriver_script_func",
				"callSelenium",
				"_selenium",
				"callPhantomJS",
				"phantomjs",
				"slimerjs",
			},
			SuspiciousRenders: []string{
				"swiftshader",
				"llvmpipe",
				"software",
				"mesa",
				"virtualbox",
				"vmware",
			},
		},
		automationSignals: &AutomationSignalAnalyzer{
			AutomationTools: map[string][]string{
				"selenium": {"selenium", "webdriver", "chromedriver", "geckodriver"},
				"puppeteer": {"puppeteer", "headlesschrome", "chromium"},
				"playwright": {"playwright", "firefox", "webkit"},
				"phantomjs": {"phantomjs", "slimerjs"},
			},
			TimingPatterns: map[string]float64{
				"instant": 0.0,
				"fast": 0.05,
				"normal": 0.2,
				"slow": 0.5,
			},
		},
		mouseAnalyzer: &AdvancedMouseAnalyzer{
			BehavioralThresholds: &MouseThresholds{
				MinSpeedForHuman:  10.0,
				MaxSpeedForHuman:  1500.0,
				MinPauseDuration:  0.05,
				MaxLinearRatio:    0.95,
			},
			PatternDetectors: []MousePatternDetector{
				{PatternName: "perfect_linear", DetectionFunc: nil},
				{PatternName: "uniform_speed", DetectionFunc: nil},
				{PatternName: "excessive_pauses", DetectionFunc: nil},
			},
		},
		keyboardAnalyzer: &AdvancedKeyboardAnalyzer{
			TypingPatterns: map[string]*TypingPattern{
				"copy_paste": {
					PatternName: "copy_paste",
					IsBotIndicator: true,
					MinVariance: 0.0,
					MaxVariance: 0.001,
				},
				"mechanical": {
					PatternName: "mechanical",
					IsBotIndicator: true,
					MinVariance: 0.0,
					MaxVariance: 0.01,
				},
				"natural": {
					PatternName: "natural",
					IsBotIndicator: false,
					MinVariance: 0.05,
					MaxVariance: 2.0,
				},
			},
			ErrorRateThresholds: &ErrorRateThresholds{
				HumanMinErrorRate: 0.02,
				HumanMaxErrorRate: 0.15,
				BotMinErrorRate:   0.0,
				BotMaxErrorRate:   0.01,
			},
		},
	}
}

type SeleniumDetection struct {
	Detected   bool
	Methods    []string
	Confidence float64
}

type PhantomJSDetection struct {
	Detected   bool
	Methods    []string
	Confidence float64
}

type PuppeteerDetection struct {
	Detected   bool
	Methods    []string
	Confidence float64
}

type HeadlessDetection struct {
	Detected   bool
	Methods    []string
	Confidence float64
}

type MouseMovement struct {
	X         float64
	Y         float64
	Timestamp time.Time
}

type Pause struct {
	Start    time.Time
	Duration time.Duration
	Position MouseMovement
}

type MouseAnalysisResult struct {
	Suspicious       bool
	Reason           string
	AverageSpeed     float64
	Smoothness       float64
	DirectionChanges int
	PauseCount       int
	PausePattern     string
	RiskScore        float64
}

type KeyPress struct {
	Key       string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

type KeyboardAnalysisResult struct {
	Suspicious      bool
	Reason          string
	AverageInterval float64
	ErrorRate       float64
	RhythmPattern   string
	ShiftUsage      float64
	RiskScore       float64
}

type AutomationData struct {
	UserAgent      string
	Navigator      map[string]interface{}
	Headers        map[string]string
	Screen         map[string]interface{}
	MouseMovements []MouseMovement
	KeyPresses     []KeyPress
}

type EnvAutomationResult struct {
	AutomationDetected bool
	Detections        []interface{}
	MouseAnalysis     *MouseAnalysisResult
	KeyboardAnalysis  *KeyboardAnalysisResult
	RiskScore         float64
}

type AutomationDetectorResult struct {
	AutomationDetected bool
	Detections        []interface{}
	MouseAnalysis     *MouseAnalysisResult
	KeyboardAnalysis  *KeyboardAnalysisResult
	RiskScore         float64
}

func (a *AutomationDetectorV2) DetectSelenium(userAgent string, navigator map[string]interface{}) SeleniumDetection {
	result := SeleniumDetection{}

	if strings.Contains(strings.ToLower(userAgent), "selenium") {
		result.Detected = true
		result.Methods = append(result.Methods, "user_agent")
	}

	if webdriver, ok := navigator["webdriver"].(bool); ok && webdriver {
		result.Detected = true
		result.Methods = append(result.Methods, "webdriver")
	}

	if cdp, ok := navigator["__webdriver_script_function"].(bool); ok && cdp {
		result.Detected = true
		result.Methods = append(result.Methods, "cdp_function")
	}

	if dom, ok := navigator["__webdriver_script_func"].(bool); ok && dom {
		result.Detected = true
		result.Methods = append(result.Methods, "dom_func")
	}

	seleniumIndicators := []string{
		"selenium",
		"webdriver",
		"callSelenium",
		"_selenium",
		"callPhantomJS",
		"__webdriver_script_function",
		"__webdriver_script_func",
		"__webdriver_evaluate",
		"__webdriver_script_element",
		"__selenium_evaluate",
		"_driver",
		"driver_",
	}

	detectedCount := 0
	for _, varName := range seleniumIndicators {
		if _, ok := navigator[varName]; ok {
			result.Detected = true
			result.Methods = append(result.Methods, "selenium_var:"+varName)
			detectedCount++
		}
	}

	if chromeDriver, ok := navigator["chromedriver"].(string); ok && chromeDriver != "" {
		result.Detected = true
		result.Methods = append(result.Methods, "chromedriver_detected:"+chromeDriver)
	}

	if geckoDriver, ok := navigator["geckodriver"].(string); ok && geckoDriver != "" {
		result.Detected = true
		result.Methods = append(result.Methods, "geckodriver_detected:"+geckoDriver)
	}

	seleniumUAIndicators := []string{
		" chromedriver",
		" geckodriver",
		" webdriver",
		" selenium",
	}
	uaLower := strings.ToLower(userAgent)
	for _, indicator := range seleniumUAIndicators {
		if strings.Contains(uaLower, indicator) {
			result.Detected = true
			result.Methods = append(result.Methods, "selenium_ua_pattern:"+indicator)
			break
		}
	}

	result.Confidence = a.calculateSeleniumConfidence(result.Methods, detectedCount)

	return result
}

func (a *AutomationDetectorV2) calculateSeleniumConfidence(methods []string, indicatorCount int) float64 {
	baseConfidence := math.Min(float64(len(methods))*0.25, 0.8)

	if indicatorCount > 3 {
		baseConfidence = math.Min(baseConfidence+0.2, 1.0)
	}

	criticalIndicators := []string{"webdriver", "chromedriver", "geckodriver"}
	for _, method := range methods {
		for _, critical := range criticalIndicators {
			if strings.Contains(method, critical) {
				baseConfidence = math.Min(baseConfidence+0.1, 1.0)
				break
			}
		}
	}

	return baseConfidence
}

func (a *AutomationDetectorV2) DetectPhantomJS(userAgent string, navigator map[string]interface{}) PhantomJSDetection {
	result := PhantomJSDetection{}

	phantomUAIndicators := []string{
		"phantomjs",
		"slimerjs",
		"phantom",
	}
	uaLower := strings.ToLower(userAgent)
	for _, indicator := range phantomUAIndicators {
		if strings.Contains(uaLower, indicator) {
			result.Detected = true
			result.Methods = append(result.Methods, "phantom_user_agent:"+indicator)
			break
		}
	}

	if phantom, ok := navigator["phantom"].(map[string]interface{}); ok {
		result.Detected = true
		if version, exists := phantom["version"]; exists {
			result.Methods = append(result.Methods, fmt.Sprintf("phantom_version:%v", version))
		}
		if page, exists := phantom["page"]; exists {
			if pageMap, ok := page.(map[string]interface{}); ok {
				if _, hasContent := pageMap["content"]; hasContent {
					result.Methods = append(result.Methods, "phantom_page_content")
				}
			}
		}
	}

	phantomIndicators := []string{
		"page",
		"phantomjs",
		"slimerjs",
		"_phantom",
		"callPhantomJS",
	}
	for _, indicator := range phantomIndicators {
		if _, ok := navigator[indicator]; ok {
			result.Detected = true
			result.Methods = append(result.Methods, "phantom_indicator:"+indicator)
			break
		}
	}

	if callbacks, ok := navigator["callbacks"]; ok {
		if cbMap, ok := callbacks.(map[string]interface{}); ok {
			if len(cbMap) == 0 {
				result.Detected = true
				result.Methods = append(result.Methods, "phantom_empty_callbacks")
			}
		}
	}

	result.Confidence = a.calculatePhantomConfidence(result.Methods)

	return result
}

func (a *AutomationDetectorV2) calculatePhantomConfidence(methods []string) float64 {
	if len(methods) == 0 {
		return 0.0
	}

	baseScore := 0.0
	for _, method := range methods {
		if strings.Contains(method, "user_agent") {
			baseScore += 0.3
		} else if strings.Contains(method, "phantom") && !strings.Contains(method, "indicator") {
			baseScore += 0.5
		} else {
			baseScore += 0.2
		}
	}

	return math.Min(baseScore, 1.0)
}

func (a *AutomationDetectorV2) DetectPuppeteer(userAgent string, headers map[string]string) PuppeteerDetection {
	result := PuppeteerDetection{}

	if strings.Contains(strings.ToLower(userAgent), "headless") {
		result.Detected = true
		result.Methods = append(result.Methods, "headless_ua")
	}

	if strings.Contains(strings.ToLower(userAgent), "chrome") &&
		!strings.Contains(strings.ToLower(userAgent), "safari") {
		result.Detected = true
		result.Methods = append(result.Methods, "chrome_without_safari")
	}

	puppeteerHeaders := []string{"puppeteer", "headlesschrome", "chromium"}
	for header, value := range headers {
		lowerHeader := strings.ToLower(header)
		lowerValue := strings.ToLower(value)
		for _, pup := range puppeteerHeaders {
			if strings.Contains(lowerHeader, pup) || strings.Contains(lowerValue, pup) {
				result.Detected = true
				result.Methods = append(result.Methods, "header:"+header)
				break
			}
		}
	}

	result.Confidence = a.calculateConfidence(result.Methods)

	return result
}

func (a *AutomationDetectorV2) DetectHeadlessChrome(navigator map[string]interface{}, screen map[string]interface{}) HeadlessDetection {
	result := HeadlessDetection{}

	if width, ok := screen["width"].(float64); ok {
		if width < 100 || width > 10000 {
			result.Detected = true
			result.Methods = append(result.Methods, "abnormal_screen_width")
		}
	}

	if languages, ok := navigator["languages"].([]string); ok && len(languages) == 0 {
		result.Detected = true
		result.Methods = append(result.Methods, "no_languages")
	}

	if plugins, ok := navigator["plugins"].([]string); ok && len(plugins) == 0 {
		result.Detected = true
		result.Methods = append(result.Methods, "no_plugins")
	}

	if touchSupport, ok := navigator["maxTouchPoints"].(float64); ok {
		if touchSupport == 0 && strings.Contains(strings.ToLower(a.userAgentFromNavigator(navigator)), "mobile") {
			result.Detected = true
			result.Methods = append(result.Methods, "mobile_without_touch")
		}
	}

	if webgl, ok := navigator["webgl"].(string); ok {
		if strings.Contains(strings.ToLower(webgl), "swiftshader") ||
			strings.Contains(strings.ToLower(webgl), "llvmpipe") {
			result.Detected = true
			result.Methods = append(result.Methods, "software_renderer")
		}
	}

	headlessResult := a.detectHeadlessWebdriver(navigator)
	if headlessResult.Detected {
		result.Detected = true
		result.Methods = append(result.Methods, headlessResult.Methods...)
	}

	windowResult := a.detectWindowPropertyAnomalies(navigator)
	if windowResult.Detected {
		result.Detected = true
		result.Methods = append(result.Methods, windowResult.Methods...)
	}

	result.Confidence = a.calculateConfidence(result.Methods)

	return result
}

func (a *AutomationDetectorV2) AnalyzeMouseMovement(movements []MouseMovement) MouseAnalysisResult {
	result := MouseAnalysisResult{}

	if len(movements) < 2 {
		result.Suspicious = true
		result.Reason = "insufficient_data"
		return result
	}

	speeds := a.calculateMovementSpeeds(movements)
	avgSpeed := a.calculateAverage(speeds)
	result.AverageSpeed = avgSpeed

	speedVariance := a.calculateVariance(speeds, avgSpeed)
	if speedVariance < 0.01 || avgSpeed > 2000 {
		result.Suspicious = true
		result.Reason = "abnormal_speed"
	}

	smoothness := a.calculateSmoothness(movements)
	result.Smoothness = smoothness

	if smoothness < 0.5 {
		result.Suspicious = true
		result.Reason = "unnatural_movement"
	}

	directionChanges := a.countDirectionChanges(movements)
	result.DirectionChanges = directionChanges

	if directionChanges < len(movements)/10 {
		result.Suspicious = true
		result.Reason = "too_linear"
	}

	mechanicalAnalysis := a.detectMechanicalMovement(movements)
	if mechanicalAnalysis.IsMechanical {
		result.Suspicious = true
		result.Reason = "mechanical_pattern_detected"
	}

	pauses := a.detectPauses(movements)
	result.PauseCount = len(pauses)
	result.PausePattern = a.analyzePausePattern(pauses)

	if len(pauses) == 0 {
		result.Suspicious = true
		result.Reason = "no_natural_pauses"
	}

	result.RiskScore = a.calculateRiskScore(result)

	return result
}

type MechanicalMovementResult struct {
	IsMechanical     bool
	MechanicalScore  float64
	LinearRatio      float64
	UniformSpeedRatio float64
	PatternType      string
}

func (a *AutomationDetectorV2) detectMechanicalMovement(movements []MouseMovement) MechanicalMovementResult {
	result := MechanicalMovementResult{}

	if len(movements) < 3 {
		return result
	}

	linearAnalysis := a.analyzeLinearMovement(movements)
	result.LinearRatio = linearAnalysis.LinearRatio
	if linearAnalysis.LinearRatio > 0.95 {
		result.MechanicalScore += 0.5
		result.PatternType = "linear"
	}

	speedUniformity := a.analyzeSpeedUniformity(movements)
	result.UniformSpeedRatio = speedUniformity
	if speedUniformity > 0.95 {
		result.MechanicalScore += 0.4
		if result.PatternType == "linear" {
			result.PatternType = "linear_uniform"
		} else {
			result.PatternType = "uniform_speed"
		}
	}

	angleConsistency := a.analyzeAngleConsistency(movements)
	if angleConsistency > 0.95 {
		result.MechanicalScore += 0.3
		result.PatternType += "_angle_consistent"
	}

	timeConsistency := a.analyzeTimeConsistency(movements)
	if timeConsistency > 0.95 {
		result.MechanicalScore += 0.2
		result.PatternType += "_time_consistent"
	}

	result.IsMechanical = result.MechanicalScore > 0.6

	return result
}

type LinearAnalysisResult struct {
	LinearRatio     float64
	DeviationPoints int
	TotalPoints     int
}

func (a *AutomationDetectorV2) analyzeLinearMovement(movements []MouseMovement) LinearAnalysisResult {
	result := LinearAnalysisResult{}

	if len(movements) < 3 {
		return result
	}

	start := movements[0]
	end := movements[len(movements)-1]

	totalDX := end.X - start.X
	totalDY := end.Y - start.Y
	totalDist := math.Sqrt(totalDX*totalDX + totalDY*totalDY)

	if totalDist < 10 {
		result.LinearRatio = 1.0
		return result
	}

	result.TotalPoints = len(movements) - 2

	for i := 1; i < len(movements)-1; i++ {
		p := movements[i]
		dx := p.X - start.X
		dy := p.Y - start.Y

		t := (dx*totalDX + dy*totalDY) / (totalDist * totalDist)

		if t < 0 || t > 1 {
			result.DeviationPoints++
			continue
		}

		projX := start.X + t*totalDX
		projY := start.Y + t*totalDY

		deviation := math.Sqrt(math.Pow(p.X-projX, 2) + math.Pow(p.Y-projY, 2))

		if deviation > 5 {
			result.DeviationPoints++
		}
	}

	if result.TotalPoints > 0 {
		result.LinearRatio = 1.0 - (float64(result.DeviationPoints) / float64(result.TotalPoints))
	}

	return result
}

func (a *AutomationDetectorV2) analyzeSpeedUniformity(movements []MouseMovement) float64 {
	speeds := a.calculateMovementSpeeds(movements)
	if len(speeds) < 2 {
		return 0.0
	}

	avg := a.calculateAverage(speeds)
	if avg < 1.0 {
		return 1.0
	}

	variance := a.calculateVariance(speeds, avg)
	coefficientOfVariation := math.Sqrt(variance) / avg

	return math.Max(0, 1.0-coefficientOfVariation)
}

func (a *AutomationDetectorV2) analyzeAngleConsistency(movements []MouseMovement) float64 {
	if len(movements) < 3 {
		return 0.0
	}

	angles := []float64{}
	for i := 2; i < len(movements); i++ {
		dx1 := movements[i-1].X - movements[i-2].X
		dy1 := movements[i-1].Y - movements[i-2].Y
		dx2 := movements[i].X - movements[i-1].X
		dy2 := movements[i].Y - movements[i-1].Y

		dot := dx1*dx2 + dy1*dy2
		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if mag1 > 0 && mag2 > 0 {
			cosAngle := dot / (mag1 * mag2)
			angle := math.Acos(math.Max(-1, math.Min(1, cosAngle)))
			angles = append(angles, angle)
		}
	}

	if len(angles) == 0 {
		return 1.0
	}

	avgAngle := a.calculateAverage(angles)
	variance := a.calculateVariance(angles, avgAngle)

	consistency := 1.0 / (1.0 + math.Sqrt(variance)*10)

	return consistency
}

func (a *AutomationDetectorV2) analyzeTimeConsistency(movements []MouseMovement) float64 {
	if len(movements) < 2 {
		return 0.0
	}

	intervals := []float64{}
	for i := 1; i < len(movements); i++ {
		interval := movements[i].Timestamp.Sub(movements[i-1].Timestamp).Seconds()
		intervals = append(intervals, interval)
	}

	if len(intervals) < 2 {
		return 0.0
	}

	avg := a.calculateAverage(intervals)
	if avg < 0.001 {
		return 1.0
	}

	variance := a.calculateVariance(intervals, avg)
	coefficientOfVariation := math.Sqrt(variance) / avg

	return math.Max(0, 1.0-coefficientOfVariation)
}

func (a *AutomationDetectorV2) AnalyzeKeyboardInput(keypresses []KeyPress) KeyboardAnalysisResult {
	result := KeyboardAnalysisResult{}

	if len(keypresses) < 5 {
		result.Suspicious = true
		result.Reason = "insufficient_data"
		return result
	}

	intervals := a.calculateKeyIntervals(keypresses)
	avgInterval := a.calculateAverage(intervals)
	result.AverageInterval = avgInterval

	errorCount := 0
	for _, kp := range keypresses {
		if kp.Key == "Backspace" || kp.Key == "Delete" {
			errorCount++
		}
	}
	errorRate := float64(errorCount) / float64(len(keypresses))
	result.ErrorRate = errorRate

	rhythm := a.analyzeKeyRhythm(intervals)
	result.RhythmPattern = rhythm

	rhythmVariance := a.calculateVariance(intervals, avgInterval)
	if rhythmVariance < 0.01 && avgInterval < 0.05 {
		result.Suspicious = true
		result.Reason = "mechanical_input"
	}

	pasteResult := a.detectPasteBehavior(keypresses)
	if pasteResult.IsPasted {
		result.Suspicious = true
		result.Reason = "paste_detected"
	}

	speedAnalysis := a.analyzeTypingSpeed(keypresses)
	if speedAnalysis.IsSuspicious {
		result.Suspicious = true
		result.Reason = speedAnalysis.SuspiciousReason
	}

	shiftCount := 0
	for _, kp := range keypresses {
		if kp.Key == "Shift" {
			shiftCount++
		}
	}
	result.ShiftUsage = float64(shiftCount) / float64(len(keypresses))

	result.RiskScore = a.calculateKeyboardRiskScore(result)

	return result
}

type PasteDetectionResult struct {
	IsPasted          bool
	PasteProbability float64
	PasteIndicators  []string
}

func (a *AutomationDetectorV2) detectPasteBehavior(keypresses []KeyPress) PasteDetectionResult {
	result := PasteDetectionResult{}

	if len(keypresses) < 10 {
		return result
	}

	intervals := a.calculateKeyIntervals(keypresses)
	avgInterval := a.calculateAverage(intervals)

	if avgInterval < 0.005 {
		result.PasteProbability = 0.8
		result.PasteIndicators = append(result.PasteIndicators, "instant_input")
	}

	intervalsWithoutErrors := []float64{}
	for i, kp := range keypresses {
		if kp.Key != "Backspace" && kp.Key != "Delete" && kp.Key != "Control" && kp.Key != "Meta" {
			if i > 0 {
				prevkp := keypresses[i-1]
				if prevkp.Key != "Backspace" && prevkp.Key != "Delete" && prevkp.Key != "Control" && prevkp.Key != "Meta" {
					interval := keypresses[i].Timestamp.Sub(keypresses[i-1].Timestamp).Seconds()
					intervalsWithoutErrors = append(intervalsWithoutErrors, interval)
				}
			}
		}
	}

	if len(intervalsWithoutErrors) > 5 {
		avgClean := a.calculateAverage(intervalsWithoutErrors)
		varianceClean := a.calculateVariance(intervalsWithoutErrors, avgClean)

		if varianceClean < 0.0001 && avgClean < 0.01 {
			result.PasteProbability += 0.6
			result.PasteIndicators = append(result.PasteIndicators, "uniform_micro_timing")
		}

		medianClean := a.calculateMedian(intervalsWithoutErrors)
		if math.Abs(avgClean-medianClean) < 0.001 {
			result.PasteProbability += 0.4
			result.PasteIndicators = append(result.PasteIndicators, "perfect_timing_regularity")
		}
	}

	keyLengthVariance := 0.0
	for i := 1; i < len(intervalsWithoutErrors); i++ {
		diff := math.Abs(intervalsWithoutErrors[i] - intervalsWithoutErrors[i-1])
		keyLengthVariance += diff * diff
	}
	if len(intervalsWithoutErrors) > 1 {
		keyLengthVariance /= float64(len(intervalsWithoutErrors) - 1)
	}

	if keyLengthVariance < 0.00001 {
		result.PasteProbability += 0.5
		result.PasteIndicators = append(result.PasteIndicators, "identical_timing")
	}

	modifierCount := 0
	for _, kp := range keypresses {
		if kp.Key == "Control" || kp.Key == "Meta" || kp.Key == "Alt" {
			modifierCount++
		}
	}
	modifierRatio := float64(modifierCount) / float64(len(keypresses))

	if modifierRatio < 0.05 && len(keypresses) > 20 {
		result.PasteProbability += 0.3
		result.PasteIndicators = append(result.PasteIndicators, "no_modifiers")
	}

	result.IsPasted = result.PasteProbability > 0.6

	return result
}

type TypingSpeedResult struct {
	AverageSpeed      float64
	IsSuspicious      bool
	SuspiciousReason  string
	SpeedPattern      string
}

func (a *AutomationDetectorV2) analyzeTypingSpeed(keypresses []KeyPress) TypingSpeedResult {
	result := TypingSpeedResult{}

	if len(keypresses) < 3 {
		return result
	}

	intervals := a.calculateKeyIntervals(keypresses)
	avgInterval := a.calculateAverage(intervals)

	charsPerSecond := 1.0 / (avgInterval + 0.001)
	result.AverageSpeed = charsPerSecond

	if charsPerSecond > 30 {
		result.IsSuspicious = true
		result.SuspiciousReason = "extremely_fast_typing"
		result.SpeedPattern = "machine_gun"
	} else if charsPerSecond > 15 {
		variance := a.calculateVariance(intervals, avgInterval)
		if variance < 0.01 {
			result.IsSuspicious = true
			result.SuspiciousReason = "fast_mechanical_typing"
			result.SpeedPattern = "mechanical_fast"
		} else {
			result.SpeedPattern = "fast_human"
		}
	} else if charsPerSecond < 1 {
		result.SpeedPattern = "very_slow"
	} else {
		result.SpeedPattern = "normal"
	}

	speedChanges := 0
	for i := 1; i < len(intervals); i++ {
		ratio := intervals[i] / (intervals[i-1] + 0.001)
		if ratio < 0.5 || ratio > 2.0 {
			speedChanges++
		}
	}

	speedChangeRatio := float64(speedChanges) / float64(len(intervals))
	if speedChangeRatio > 0.8 && result.SpeedPattern == "normal" {
		result.IsSuspicious = true
		result.SuspiciousReason = "erratic_speed_pattern"
		result.SpeedPattern = "erratic"
	}

	return result
}

func (a *AutomationDetectorV2) DetectAutomation(data AutomationData) AutomationDetectorResult {
	result := AutomationDetectorResult{}

	if data.UserAgent != "" && data.Navigator != nil {
		selenium := a.DetectSelenium(data.UserAgent, data.Navigator)
		if selenium.Detected {
			result.AutomationDetected = true
			result.Detections = append(result.Detections, selenium)
		}
	}

	if data.UserAgent != "" && data.Headers != nil {
		puppeteer := a.DetectPuppeteer(data.UserAgent, data.Headers)
		if puppeteer.Detected {
			result.AutomationDetected = true
			result.Detections = append(result.Detections, puppeteer)
		}
	}

	if data.Navigator != nil && data.Screen != nil {
		headless := a.DetectHeadlessChrome(data.Navigator, data.Screen)
		if headless.Detected {
			result.AutomationDetected = true
			result.Detections = append(result.Detections, headless)
		}
	}

	if len(data.MouseMovements) > 0 {
		mouseAnalysis := a.AnalyzeMouseMovement(data.MouseMovements)
		result.MouseAnalysis = &mouseAnalysis
		if mouseAnalysis.Suspicious {
			result.RiskScore += mouseAnalysis.RiskScore
		}
	}

	if len(data.KeyPresses) > 0 {
		keyboardAnalysis := a.AnalyzeKeyboardInput(data.KeyPresses)
		result.KeyboardAnalysis = &keyboardAnalysis
		if keyboardAnalysis.Suspicious {
			result.RiskScore += keyboardAnalysis.RiskScore
		}
	}

	result.RiskScore = a.normalizeScore(result.RiskScore)

	return result
}

func (a *AutomationDetectorV2) calculateConfidence(methods []string) float64 {
	return math.Min(float64(len(methods))*0.3, 1.0)
}

func (a *AutomationDetectorV2) calculateMovementSpeeds(movements []MouseMovement) []float64 {
	speeds := make([]float64, 0, len(movements)-1)
	for i := 1; i < len(movements); i++ {
		dx := movements[i].X - movements[i-1].X
		dy := movements[i].Y - movements[i-1].Y
		dt := movements[i].Timestamp.Sub(movements[i-1].Timestamp).Seconds()
		if dt > 0 {
			speed := math.Sqrt(dx*dx+dy*dy) / dt
			speeds = append(speeds, speed)
		}
	}
	return speeds
}

func (a *AutomationDetectorV2) calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (a *AutomationDetectorV2) calculateVariance(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

func (a *AutomationDetectorV2) calculateSmoothness(movements []MouseMovement) float64 {
	if len(movements) < 3 {
		return 1.0
	}

	smoothness := 0.0
	count := 0
	for i := 2; i < len(movements); i++ {
		angle1 := math.Atan2(movements[i-1].Y-movements[i-2].Y, movements[i-1].X-movements[i-2].X)
		angle2 := math.Atan2(movements[i].Y-movements[i-1].Y, movements[i].X-movements[i-1].X)
		angleDiff := math.Abs(angle2 - angle1)
		if angleDiff > math.Pi {
			angleDiff = 2*math.Pi - angleDiff
		}
		smoothness += 1.0 - angleDiff/math.Pi
		count++
	}

	if count == 0 {
		return 1.0
	}
	return smoothness / float64(count)
}

func (a *AutomationDetectorV2) countDirectionChanges(movements []MouseMovement) int {
	if len(movements) < 2 {
		return 0
	}

	changes := 0
	for i := 2; i < len(movements); i++ {
		dx1 := movements[i-1].X - movements[i-2].X
		dy1 := movements[i-1].Y - movements[i-2].Y
		dx2 := movements[i].X - movements[i-1].X
		dy2 := movements[i].Y - movements[i-1].Y

		cross := dx1*dy2 - dy1*dx2
		if math.Abs(cross) > 10 {
			changes++
		}
	}

	return changes
}

func (a *AutomationDetectorV2) detectPauses(movements []MouseMovement) []Pause {
	pauses := []Pause{}
	threshold := 50 * time.Millisecond

	for i := 1; i < len(movements); i++ {
		duration := movements[i].Timestamp.Sub(movements[i-1].Timestamp)
		if duration > threshold {
			distance := math.Sqrt(
				math.Pow(movements[i].X-movements[i-1].X, 2) +
					math.Pow(movements[i].Y-movements[i-1].Y, 2))
			if distance < 5 {
				pauses = append(pauses, Pause{
					Start:    movements[i-1].Timestamp,
					Duration: duration,
					Position: movements[i-1],
				})
			}
		}
	}

	return pauses
}

func (a *AutomationDetectorV2) analyzePausePattern(pauses []Pause) string {
	if len(pauses) == 0 {
		return "none"
	}

	intervals := make([]float64, len(pauses))
	for i := 1; i < len(pauses); i++ {
		intervals[i-1] = pauses[i].Start.Sub(pauses[i-1].Start).Seconds()
	}

	variance := a.calculateVariance(intervals, a.calculateAverage(intervals))

	if variance < 0.1 {
		return "regular"
	} else if variance < 1.0 {
		return "natural"
	}
	return "random"
}

func (a *AutomationDetectorV2) calculateKeyIntervals(keypresses []KeyPress) []float64 {
	intervals := make([]float64, 0, len(keypresses)-1)
	for i := 1; i < len(keypresses); i++ {
		interval := keypresses[i].Timestamp.Sub(keypresses[i-1].Timestamp).Seconds()
		intervals = append(intervals, interval)
	}
	return intervals
}

func (a *AutomationDetectorV2) analyzeKeyRhythm(intervals []float64) string {
	if len(intervals) == 0 {
		return "unknown"
	}

	avg := a.calculateAverage(intervals)
	variance := a.calculateVariance(intervals, avg)

	if variance < 0.001 {
		return "mechanical"
	} else if variance < 0.1 {
		return "regular"
	}
	return "natural"
}

func (a *AutomationDetectorV2) calculateRiskScore(result MouseAnalysisResult) float64 {
	score := 0.0

	if result.Suspicious {
		score += 0.4
	}

	if result.Smoothness < 0.5 {
		score += 0.3
	}

	if result.AverageSpeed > 1500 {
		score += 0.3
	}

	return math.Min(score, 1.0)
}

func (a *AutomationDetectorV2) calculateKeyboardRiskScore(result KeyboardAnalysisResult) float64 {
	score := 0.0

	if result.Suspicious {
		score += 0.3
	}

	if result.ErrorRate < 0.01 {
		score += 0.2
	}

	if result.RhythmPattern == "mechanical" {
		score += 0.5
	}

	return math.Min(score, 1.0)
}

func (a *AutomationDetectorV2) normalizeScore(score float64) float64 {
	return math.Min(score, 1.0)
}

func (a *AutomationDetectorV2) userAgentFromNavigator(navigator map[string]interface{}) string {
	if ua, ok := navigator["userAgent"].(string); ok {
		return ua
	}
	return ""
}

func (a *AutomationDetectorV2) detectHeadlessWebdriver(navigator map[string]interface{}) HeadlessDetection {
	result := HeadlessDetection{}

	if webdriver, ok := navigator["webdriver"].(bool); ok && webdriver {
		result.Detected = true
		result.Methods = append(result.Methods, "navigator_webdriver_true")
	}

	if chrome, ok := navigator["chrome"].(map[string]interface{}); ok {
		if runtime, exists := chrome["runtime"]; exists {
			if runtimeMap, ok := runtime.(map[string]interface{}); ok {
				if _, hasConnect := runtimeMap["connect"]; !hasConnect {
					result.Detected = true
					result.Methods = append(result.Methods, "chrome_runtime_missing_connect")
				}
			}
		}
	}

	automationIndicators := []string{
		"__webdriver_script_function",
		"__webdriver_script_func",
		"__webdriver_script_element",
		"__webdriver_evaluate",
		"_selenium",
		"callSelenium",
		"_callPhantomJS",
		"fb_xd_fragment",
		"phantom",
	}

	for _, indicator := range automationIndicators {
		if _, ok := navigator[indicator]; ok {
			result.Detected = true
			result.Methods = append(result.Methods, "automation_indicator:"+indicator)
			break
		}
	}

	if permissions, ok := navigator["permissions"].(string); ok {
		if strings.ToLower(permissions) == "denied" {
			result.Detected = true
			result.Methods = append(result.Methods, "permissions_denied")
		}
	}

	result.Confidence = a.calculateConfidence(result.Methods)
	return result
}

func (a *AutomationDetectorV2) detectWindowPropertyAnomalies(navigator map[string]interface{}) HeadlessDetection {
	result := HeadlessDetection{}

	requiredWindowProps := []string{"location", "history", "navigator", "screen", "performance"}
	missingProps := []string{}

	for _, prop := range requiredWindowProps {
		if _, ok := navigator[prop]; !ok {
			missingProps = append(missingProps, prop)
		}
	}

	if len(missingProps) > 2 {
		result.Detected = true
		result.Methods = append(result.Methods, fmt.Sprintf("missing_window_props:%s", strings.Join(missingProps, ",")))
	}

	if innerWidth, ok := navigator["innerWidth"].(float64); ok {
		if innerWidth == 0 || innerWidth > 10000 {
			result.Detected = true
			result.Methods = append(result.Methods, "abnormal_innerWidth")
		}
	}

	if innerHeight, ok := navigator["innerHeight"].(float64); ok {
		if innerHeight == 0 || innerHeight > 10000 {
			result.Detected = true
			result.Methods = append(result.Methods, "abnormal_innerHeight")
		}
	}

	if outerWidth, ok := navigator["outerWidth"].(float64); ok {
		if outerWidth == 0 {
			result.Detected = true
			result.Methods = append(result.Methods, "zero_outerWidth")
		}
	}

	if devicePixelRatio, ok := navigator["devicePixelRatio"].(float64); ok {
		if devicePixelRatio == 0 || devicePixelRatio > 5 {
			result.Detected = true
			result.Methods = append(result.Methods, "abnormal_devicePixelRatio")
		}
	}

	if statusBar, ok := navigator["statusbar"]; ok {
		if status, ok := statusBar.(map[string]interface{}); ok {
			if visible, exists := status["visible"]; exists {
				if vis, ok := visible.(bool); ok && !vis {
					result.Detected = true
					result.Methods = append(result.Methods, "statusbar_not_visible")
				}
			}
		}
	}

	if toolbar, ok := navigator["toolbar"]; ok {
		if tool, ok := toolbar.(map[string]interface{}); ok {
			if visible, exists := tool["visible"]; exists {
				if vis, ok := visible.(bool); ok && !vis {
					result.Detected = true
					result.Methods = append(result.Methods, "toolbar_not_visible")
				}
			}
		}
	}

	result.Confidence = a.calculateConfidence(result.Methods)
	return result
}

func (a *AutomationDetectorV2) DetectAdvancedHeadless(navigator map[string]interface{}, screen map[string]interface{}) AdvancedHeadlessResult {
	result := AdvancedHeadlessResult{}

	result.PerformanceTiming = a.analyzePerformanceTiming(navigator)
	result.MediaDevices = a.analyzeMediaDevices(navigator)
	result.WebGLAnalysis = a.analyzeWebGLDeep(navigator)
	result.CanvasFingerprint = a.analyzeCanvasFingerprint(navigator)
	result.PluginsAnalysis = a.analyzePluginsDeep(navigator)
	result.BehavioralSignals = a.analyzeBehavioralSignals(navigator)

	result.TotalRiskScore = a.calculateAdvancedRiskScore(result)

	return result
}

func (a *AutomationDetectorV2) analyzePerformanceTiming(navigator map[string]interface{}) float64 {
	if timing, ok := navigator["performanceTiming"].(map[string]interface{}); ok {
		if navigationStart, ok := timing["navigationStart"].(float64); ok {
			if loadEventEnd, ok := timing["loadEventEnd"].(float64); ok {
				loadTime := loadEventEnd - navigationStart
				if loadTime < 100 {
					return 0.7
				} else if loadTime < 500 {
					return 0.3
				}
			}
		}
	}
	return 0.0
}

func (a *AutomationDetectorV2) analyzeMediaDevices(navigator map[string]interface{}) float64 {
	if devices, ok := navigator["mediaDevices"].([]string); ok {
		if len(devices) == 0 {
			return 0.5
		}

		if len(devices) == 1 && strings.Contains(strings.ToLower(devices[0]), "default") {
			return 0.4
		}
	}

	if enumerateDevices, ok := navigator["enumerateDevices"].(bool); ok && !enumerateDevices {
		return 0.6
	}

	return 0.0
}

func (a *AutomationDetectorV2) analyzeWebGLDeep(navigator map[string]interface{}) float64 {
	score := 0.0

	if renderer, ok := navigator["webglRenderer"].(string); ok {
		rendererLower := strings.ToLower(renderer)
		for _, suspicious := range a.headlessPatterns.SuspiciousRenders {
			if strings.Contains(rendererLower, strings.ToLower(suspicious)) {
				score += 0.6
				break
			}
		}
	}

	if vendor, ok := navigator["webglVendor"].(string); ok {
		if vendor == "" {
			score += 0.3
		}
	}

	if debugInfo, ok := navigator["webglDebugInfo"].(map[string]interface{}); ok {
		if rendererInfo, exists := debugInfo["rendererInfo"]; exists {
			if infoStr, ok := rendererInfo.(string); ok {
				if strings.Contains(strings.ToLower(infoStr), "google") ||
					strings.Contains(strings.ToLower(infoStr), "apple") {
					score += 0.1
				}
			}
		}
	}

	return math.Min(score, 1.0)
}

func (a *AutomationDetectorV2) analyzeCanvasFingerprint(navigator map[string]interface{}) float64 {
	if canvas, ok := navigator["canvasFingerprint"].(string); ok {
		if len(canvas) < 100 {
			return 0.6
		}

		if canvas == "" {
			return 0.4
		}
	}

	if canvasContext, ok := navigator["canvasContext"].(string); ok {
		if strings.Contains(strings.ToLower(canvasContext), "2d") &&
			!strings.Contains(strings.ToLower(canvasContext), "webgl") {
			return 0.2
		}
	}

	return 0.0
}

func (a *AutomationDetectorV2) analyzePluginsDeep(navigator map[string]interface{}) float64 {
	if plugins, ok := navigator["plugins"].([]string); ok {
		if len(plugins) == 0 {
			return 0.4
		}

		if len(plugins) < 3 {
			return 0.2
		}

		commonPlugins := 0
		for _, plugin := range plugins {
			pluginLower := strings.ToLower(plugin)
			if strings.Contains(pluginLower, "pdf") ||
				strings.Contains(pluginLower, "flash") ||
				strings.Contains(pluginLower, "silverlight") {
				commonPlugins++
			}
		}

		if commonPlugins == 0 {
			return 0.3
		}
	}

	return 0.0
}

func (a *AutomationDetectorV2) analyzeBehavioralSignals(navigator map[string]interface{}) float64 {
	score := 0.0

	if connectionType, ok := navigator["connectionType"].(string); ok {
		if connectionType == "unknown" || connectionType == "" {
			score += 0.2
		}
	}

	if doNotTrack, ok := navigator["doNotTrack"].(string); ok {
		if doNotTrack == "unspecified" {
			score += 0.1
		}
	}

	if battery, ok := navigator["battery"].(map[string]interface{}); ok {
		if charging, ok := battery["charging"].(bool); ok && !charging {
			score += 0.1
		}
	}

	if permissions, ok := navigator["permissions"].(map[string]string); ok {
		if geolocation, exists := permissions["geolocation"]; exists && geolocation == "denied" {
			score += 0.1
		}
	}

	return math.Min(score, 1.0)
}

func (a *AutomationDetectorV2) calculateAdvancedRiskScore(result AdvancedHeadlessResult) float64 {
	totalScore := result.PerformanceTiming +
		result.MediaDevices +
		result.WebGLAnalysis +
		result.CanvasFingerprint +
		result.PluginsAnalysis +
		result.BehavioralSignals

	factorCount := 6.0

	return math.Min(totalScore/factorCount, 1.0)
}

func (a *AutomationDetectorV2) AnalyzeMousePatternAdvanced(movements []MouseMovement) AdvancedMouseResult {
	result := AdvancedMouseResult{}

	if len(movements) < 3 {
		result.BasicResult = MouseAnalysisResult{
			Suspicious: true,
			Reason:    "insufficient_data",
		}
		return result
	}

	result.SpeedAnalysis = a.analyzeSpeedDistribution(movements)
	result.TrajectoryPattern = a.analyzeTrajectoryPattern(movements)
	result.BezierQuality = a.analyzeBezierCurveQuality(movements)
	result.MicroMovement = a.analyzeMicroMovements(movements)
	result.AccelerationPattern = a.analyzeAccelerationPattern(movements)

	result.Confidence = a.calculateMouseConfidence(result)
	result.BehaviorSignature = a.generateMouseSignature(result)

	return result
}

func (a *AutomationDetectorV2) analyzeSpeedDistribution(movements []MouseMovement) SpeedDistributionResult {
	result := SpeedDistributionResult{}

	speeds := a.calculateMovementSpeeds(movements)
	if len(speeds) == 0 {
		return result
	}

	result.Average = a.calculateAverage(speeds)
	result.Maximum = a.calculateMax(speeds)
	result.Minimum = a.calculateMin(speeds)
	result.Variance = a.calculateVariance(speeds, result.Average)
	result.Median = a.calculateMedian(speeds)

	result.IsUniform = result.Variance < (result.Average * 0.1)
	result.IsExtreme = result.Maximum > 2000 || result.Minimum < 5

	return result
}

func (a *AutomationDetectorV2) calculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func (a *AutomationDetectorV2) calculateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func (a *AutomationDetectorV2) calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	return sorted[len(sorted)/2]
}

func (a *AutomationDetectorV2) analyzeTrajectoryPattern(movements []MouseMovement) TrajectoryPatternResult {
	result := TrajectoryPatternResult{}

	totalDist := a.calculateTotalDistance(movements)
	if totalDist == 0 {
		return result
	}

	directDist := math.Sqrt(
		math.Pow(movements[len(movements)-1].X-movements[0].X, 2) +
			math.Pow(movements[len(movements)-1].Y-movements[0].Y, 2))

	result.PathRatio = directDist / totalDist

	if result.PathRatio > 0.99 {
		result.Pattern = "perfect_straight"
		result.BotScore = 0.8
	} else if result.PathRatio > 0.95 {
		result.Pattern = "almost_straight"
		result.BotScore = 0.5
	} else if result.PathRatio < 0.5 {
		result.Pattern = "complex"
		result.BotScore = 0.2
	} else {
		result.Pattern = "normal"
		result.BotScore = 0.3
	}

	result.TurnCount = a.countTurns(movements)
	result.CurvatureVariance = a.calculateCurvatureVariance(movements)

	return result
}

func (a *AutomationDetectorV2) calculateTotalDistance(movements []MouseMovement) float64 {
	total := 0.0
	for i := 1; i < len(movements); i++ {
		dx := movements[i].X - movements[i-1].X
		dy := movements[i].Y - movements[i-1].Y
		total += math.Sqrt(dx*dx + dy*dy)
	}
	return total
}

func (a *AutomationDetectorV2) countTurns(movements []MouseMovement) int {
	turns := 0
	for i := 2; i < len(movements); i++ {
		dx1 := movements[i-1].X - movements[i-2].X
		dy1 := movements[i-1].Y - movements[i-2].Y
		dx2 := movements[i].X - movements[i-1].X
		dy2 := movements[i].Y - movements[i-1].Y

		dot := dx1*dx2 + dy1*dy2
		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if mag1 > 0 && mag2 > 0 {
			cosAngle := dot / (mag1 * mag2)
			if cosAngle < 0.5 {
				turns++
			}
		}
	}
	return turns
}

func (a *AutomationDetectorV2) calculateCurvatureVariance(movements []MouseMovement) float64 {
	if len(movements) < 3 {
		return 0.0
	}

	curvatures := []float64{}
	for i := 1; i < len(movements)-1; i++ {
		dx1 := movements[i].X - movements[i-1].X
		dy1 := movements[i].Y - movements[i-1].Y
		dx2 := movements[i+1].X - movements[i].X
		dy2 := movements[i+1].Y - movements[i].Y

		cross := dx1*dy2 - dy1*dx2
		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if mag1 > 0 && mag2 > 0 {
			curvature := math.Abs(cross) / (mag1 * mag2)
			curvatures = append(curvatures, curvature)
		}
	}

	if len(curvatures) == 0 {
		return 0.0
	}

	avg := a.calculateAverage(curvatures)
	return a.calculateVariance(curvatures, avg)
}

func (a *AutomationDetectorV2) analyzeBezierCurveQuality(movements []MouseMovement) float64 {
	if len(movements) < 10 {
		return 0.0
	}

	speeds := a.calculateMovementSpeeds(movements)
	speedVariance := a.calculateVariance(speeds, a.calculateAverage(speeds))

	if speedVariance < 10 {
		return 0.8
	} else if speedVariance < 100 {
		return 0.5
	}

	return 0.2
}

func (a *AutomationDetectorV2) analyzeMicroMovements(movements []MouseMovement) MicroMovementResult {
	result := MicroMovementResult{}

	if len(movements) < 3 {
		return result
	}

	microCount := 0
	totalMicroDist := 0.0

	for i := 1; i < len(movements); i++ {
		dx := movements[i].X - movements[i-1].X
		dy := movements[i].Y - movements[i-1].Y
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist < 3 {
			microCount++
			totalMicroDist += dist
		}
	}

	result.Count = microCount
	result.Ratio = float64(microCount) / float64(len(movements))
	result.AverageDistance = totalMicroDist / math.Max(1.0, float64(microCount))

	if result.Ratio > 0.8 {
		result.IsSuspicious = true
		result.SuspicionReason = "excessive_micro_movements"
	} else if result.Ratio < 0.1 {
		result.IsSuspicious = true
		result.SuspicionReason = "no_micro_movements"
	}

	return result
}

func (a *AutomationDetectorV2) analyzeAccelerationPattern(movements []MouseMovement) AccelerationPatternResult {
	result := AccelerationPatternResult{}

	if len(movements) < 3 {
		return result
	}

	speeds := a.calculateMovementSpeeds(movements)
	if len(speeds) < 2 {
		return result
	}

	accelerations := []float64{}
	for i := 1; i < len(speeds); i++ {
		dt := movements[i+1].Timestamp.Sub(movements[i].Timestamp).Seconds()
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}

	if len(accelerations) > 0 {
		result.Average = a.calculateAverage(accelerations)
		result.Maximum = a.calculateMax(accelerations)
		result.Minimum = a.calculateMin(accelerations)
		result.Variance = a.calculateVariance(accelerations, result.Average)
	}

	if math.Abs(result.Variance) < 0.1 {
		result.Pattern = "constant"
		result.BotScore = 0.7
	} else if math.Abs(result.Variance) > 10 {
		result.Pattern = "erratic"
		result.BotScore = 0.5
	} else {
		result.Pattern = "natural"
		result.BotScore = 0.2
	}

	return result
}

func (a *AutomationDetectorV2) calculateMouseConfidence(result AdvancedMouseResult) float64 {
	confidence := 0.0

	confidence += (1.0 - result.SpeedAnalysis.Variance/1000.0) * 0.2
	confidence += (1.0 - result.TrajectoryPattern.BotScore) * 0.3
	confidence += (1.0 - result.BezierQuality) * 0.2

	if !result.MicroMovement.IsSuspicious {
		confidence += 0.2
	}

	confidence += (1.0 - result.AccelerationPattern.BotScore) * 0.1

	return math.Max(0, math.Min(1, confidence))
}

func (a *AutomationDetectorV2) generateMouseSignature(result AdvancedMouseResult) string {
	signature := ""

	signature += fmt.Sprintf("S:%.2f_", result.SpeedAnalysis.Average)
	signature += fmt.Sprintf("P:%.2f_", result.TrajectoryPattern.PathRatio)
	signature += fmt.Sprintf("M:%d_", result.MicroMovement.Count)
	signature += fmt.Sprintf("A:%.2f", result.AccelerationPattern.Average)

	return signature
}

func (a *AutomationDetectorV2) AnalyzeKeyboardPatternAdvanced(keypresses []KeyPress) AdvancedKeyboardResult {
	result := AdvancedKeyboardResult{}

	if len(keypresses) < 3 {
		result.BasicResult = KeyboardAnalysisResult{
			Suspicious: true,
			Reason:    "insufficient_data",
		}
		return result
	}

	result.TimingAnalysis = a.analyzeTypingTiming(keypresses)
	result.ErrorAnalysis = a.analyzeTypingErrors(keypresses)
	result.KeyDistribution = a.analyzeKeyDistribution(keypresses)
	result.RhythmPattern = a.analyzeTypingRhythm(keypresses)
	result.ForceAnalysis = a.analyzeTypingForce(keypresses)

	result.Confidence = a.calculateKeyboardConfidence(result)
	result.BehaviorSignature = a.generateKeyboardSignature(result)

	return result
}

func (a *AutomationDetectorV2) analyzeTypingTiming(keypresses []KeyPress) TypingTimingResult {
	result := TypingTimingResult{}

	intervals := a.calculateKeyIntervals(keypresses)
	if len(intervals) == 0 {
		return result
	}

	result.Average = a.calculateAverage(intervals)
	result.Median = a.calculateMedian(intervals)
	result.Variance = a.calculateVariance(intervals, result.Average)
	result.StdDev = math.Sqrt(result.Variance)
	result.MinInterval = a.calculateMin(intervals)
	result.MaxInterval = a.calculateMax(intervals)

	result.CoefficientOfVariation = result.StdDev / (result.Average + 0.001)

	if result.CoefficientOfVariation < 0.05 {
		result.Pattern = "mechanical"
		result.BotScore = 0.9
	} else if result.CoefficientOfVariation < 0.2 {
		result.Pattern = "regular"
		result.BotScore = 0.5
	} else {
		result.Pattern = "natural"
		result.BotScore = 0.2
	}

	return result
}

func (a *AutomationDetectorV2) analyzeTypingErrors(keypresses []KeyPress) TypingErrorResult {
	result := TypingErrorResult{}

	errorCount := 0
	for _, kp := range keypresses {
		if kp.Key == "Backspace" || kp.Key == "Delete" || kp.Key == "Escape" {
			errorCount++
		}
	}

	result.ErrorCount = errorCount
	result.ErrorRate = float64(errorCount) / float64(len(keypresses))

	if result.ErrorRate == 0 && len(keypresses) > 20 {
		result.IsSuspicious = true
		result.SuspicionReason = "no_errors_long_input"
	} else if result.ErrorRate > 0.3 {
		result.IsSuspicious = true
		result.SuspicionReason = "too_many_errors"
	}

	return result
}

func (a *AutomationDetectorV2) analyzeKeyDistribution(keypresses []KeyPress) KeyDistributionResult {
	result := KeyDistributionResult{}

	keyCounts := make(map[string]int)
	for _, kp := range keypresses {
		keyCounts[kp.Key]++
	}

	result.UniqueKeys = len(keyCounts)
	result.TotalKeys = len(keypresses)

	letterCount := 0
	for kp := range keyCounts {
		if len(kp) == 1 && kp >= "a" && kp <= "z" || kp >= "A" && kp <= "Z" {
			letterCount++
		}
	}
	result.LetterRatio = float64(letterCount) / float64(result.TotalKeys)

	shiftCount := keyCounts["Shift"]
	result.ShiftRatio = float64(shiftCount) / float64(result.TotalKeys)

	spaceCount := keyCounts[" "]
	result.SpaceRatio = float64(spaceCount) / float64(result.TotalKeys)

	return result
}

func (a *AutomationDetectorV2) analyzeTypingRhythm(keypresses []KeyPress) RhythmPatternResult {
	result := RhythmPatternResult{}

	intervals := a.calculateKeyIntervals(keypresses)
	if len(intervals) < 3 {
		return result
	}

	bursts := []int{}
	currentBurst := 1

	for i := 1; i < len(intervals); i++ {
		if intervals[i] < 0.1 {
			currentBurst++
		} else {
			if currentBurst > 1 {
				bursts = append(bursts, currentBurst)
			}
			currentBurst = 1
		}
	}

	result.BurstCount = len(bursts)
	if len(bursts) > 0 {
		result.AverageBurstLength = a.calculateAverageFloat(bursts)
	}

	consecutiveFast := 0
	maxConsecutiveFast := 0
	for _, interval := range intervals {
		if interval < 0.05 {
			consecutiveFast++
			if consecutiveFast > maxConsecutiveFast {
				maxConsecutiveFast = consecutiveFast
			}
		} else {
			consecutiveFast = 0
		}
	}

	result.MaxConsecutiveFast = maxConsecutiveFast

	if maxConsecutiveFast > 10 {
		result.Pattern = "machine_gun"
		result.BotScore = 0.8
	} else if result.BurstCount == 0 && len(intervals) > 5 {
		result.Pattern = "uniform"
		result.BotScore = 0.6
	} else {
		result.Pattern = "natural"
		result.BotScore = 0.2
	}

	return result
}

func (a *AutomationDetectorV2) calculateAverageFloat(values []int) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += float64(v)
	}
	return sum / float64(len(values))
}

func (a *AutomationDetectorV2) analyzeTypingForce(keypresses []KeyPress) ForcePatternResult {
	result := ForcePatternResult{}

	pressureData := []float64{}
	for _, kp := range keypresses {
		if pressure, ok := kp.KeyPressure(); ok {
			pressureData = append(pressureData, pressure)
		}
	}

	if len(pressureData) == 0 {
		result.HasData = false
		return result
	}

	result.HasData = true
	result.AveragePressure = a.calculateAverage(pressureData)
	result.Variance = a.calculateVariance(pressureData, result.AveragePressure)

	if result.Variance < 0.01 {
		result.Pattern = "constant_force"
		result.BotScore = 0.7
	} else if result.Variance > 1.0 {
		result.Pattern = "natural_variation"
		result.BotScore = 0.2
	} else {
		result.Pattern = "normal"
		result.BotScore = 0.4
	}

	return result
}

func (a *AutomationDetectorV2) calculateKeyboardConfidence(result AdvancedKeyboardResult) float64 {
	confidence := 0.0

	confidence += (1.0 - result.TimingAnalysis.BotScore) * 0.3

	if !result.ErrorAnalysis.IsSuspicious {
		confidence += 0.2
	}

	confidence += (1.0 - result.RhythmPattern.BotScore) * 0.3

	if result.ForceAnalysis.HasData {
		confidence += (1.0 - result.ForceAnalysis.BotScore) * 0.2
	}

	return math.Max(0, math.Min(1, confidence))
}

func (a *AutomationDetectorV2) generateKeyboardSignature(result AdvancedKeyboardResult) string {
	signature := ""

	signature += fmt.Sprintf("A:%.3f_", result.TimingAnalysis.Average)
	signature += fmt.Sprintf("CV:%.3f_", result.TimingAnalysis.CoefficientOfVariation)
	signature += fmt.Sprintf("E:%.2f_", result.ErrorAnalysis.ErrorRate)
	signature += fmt.Sprintf("B:%d", result.RhythmPattern.BurstCount)

	return signature
}

type AdvancedHeadlessResult struct {
	PerformanceTiming  float64
	MediaDevices       float64
	WebGLAnalysis      float64
	CanvasFingerprint  float64
	PluginsAnalysis    float64
	BehavioralSignals  float64
	TotalRiskScore     float64
}

type AdvancedMouseResult struct {
	SpeedAnalysis      SpeedDistributionResult
	TrajectoryPattern  TrajectoryPatternResult
	BezierQuality      float64
	MicroMovement      MicroMovementResult
	AccelerationPattern AccelerationPatternResult
	Confidence         float64
	BehaviorSignature  string
	BasicResult        MouseAnalysisResult
}

type SpeedDistributionResult struct {
	Average  float64
	Maximum  float64
	Minimum  float64
	Variance float64
	Median   float64
	IsUniform bool
	IsExtreme bool
}

type TrajectoryPatternResult struct {
	Pattern            string
	PathRatio          float64
	TurnCount          int
	CurvatureVariance  float64
	BotScore           float64
}

type MicroMovementResult struct {
	Count            int
	Ratio            float64
	AverageDistance  float64
	IsSuspicious     bool
	SuspicionReason  string
}

type AccelerationPatternResult struct {
	Pattern   string
	Average   float64
	Maximum   float64
	Minimum   float64
	Variance  float64
	BotScore  float64
}

type AdvancedKeyboardResult struct {
	TimingAnalysis     TypingTimingResult
	ErrorAnalysis      TypingErrorResult
	KeyDistribution    KeyDistributionResult
	RhythmPattern      RhythmPatternResult
	ForceAnalysis      ForcePatternResult
	Confidence         float64
	BehaviorSignature  string
	BasicResult        KeyboardAnalysisResult
}

type TypingTimingResult struct {
	Pattern              string
	Average              float64
	Median               float64
	Variance             float64
	StdDev               float64
	MinInterval          float64
	MaxInterval          float64
	CoefficientOfVariation float64
	BotScore             float64
}

type TypingErrorResult struct {
	ErrorCount         int
	ErrorRate          float64
	IsSuspicious       bool
	SuspicionReason    string
}

type KeyDistributionResult struct {
	UniqueKeys    int
	TotalKeys     int
	LetterRatio   float64
	ShiftRatio    float64
	SpaceRatio    float64
}

type RhythmPatternResult struct {
	Pattern               string
	BurstCount            int
	AverageBurstLength    float64
	MaxConsecutiveFast    int
	BotScore              float64
}

type ForcePatternResult struct {
	HasData          bool
	Pattern          string
	AveragePressure  float64
	Variance         float64
	BotScore         float64
}

func (kp KeyPress) KeyPressure() (float64, bool) {
	if pressure, ok := kp.Metadata["pressure"].(float64); ok {
		return pressure, true
	}
	return 0, false
}
