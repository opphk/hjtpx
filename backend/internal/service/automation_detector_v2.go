package service

import (
	"math"
	"strings"
	"time"
)

type AutomationDetectorV2 struct{}

type SeleniumDetection struct {
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

	automationVars := []string{"selenium", "webdriver", "callSelenium", "_selenium", "callPhantomJS"}
	for _, varName := range automationVars {
		if _, ok := navigator[varName]; ok {
			result.Detected = true
			result.Methods = append(result.Methods, "automation_var:"+varName)
			break
		}
	}

	result.Confidence = a.calculateConfidence(result.Methods)

	return result
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
