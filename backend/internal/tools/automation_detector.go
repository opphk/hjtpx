package tools

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

type AutomationType string

const (
	AutomationTypeSelenium   AutomationType = "selenium"
	AutomationTypePlaywright  AutomationType = "playwright"
	AutomationTypePuppeteer   AutomationType = "puppeteer"
	AutomationTypePhantomJS   AutomationType = "phantomjs"
	AutomationTypeHeadless    AutomationType = "headless"
	AutomationTypeUnknown     AutomationType = "unknown"
)

type AutomationDetector struct {
	detectionPatterns map[AutomationType][]string
	mu               sync.RWMutex
	enabledTypes     map[AutomationType]bool
}

type DetectionResult struct {
	IsAutomated    bool                `json:"is_automated"`
	DetectedTypes  []AutomationType    `json:"detected_types"`
	Confidence     float64            `json:"confidence"`
	Detections     []Detection         `json:"detections"`
	Timestamp      time.Time          `json:"timestamp"`
}

type Detection struct {
	Type       AutomationType `json:"type"`
	Evidence   string        `json:"evidence"`
	Severity   string        `json:"severity"`
	Confidence float64       `json:"confidence"`
}

func NewAutomationDetector() *AutomationDetector {
	ad := &AutomationDetector{
		detectionPatterns: make(map[AutomationType][]string),
		enabledTypes:     make(map[AutomationType]bool),
	}

	ad.initializePatterns()
	ad.enableAll()

	return ad
}

func (ad *AutomationDetector) initializePatterns() {
	ad.detectionPatterns[AutomationTypeSelenium] = []string{
		`webdriver`,
		`__webdriver_script_function`,
		`__webdriver_script_func`,
		`__webdriver_script_fn`,
		`selenium`,
		`selenium-`,
		`SLIMERJS`,
		`callSelenium`,
		`_selenium`,
		`computed|selenium`,
	}

	ad.detectionPatterns[AutomationTypePlaywright] = []string{
		`__playwright`,
		`playwright`,
		`__pw`,
		`pw_api`,
		`playwright-replaced`,
	}

	ad.detectionPatterns[AutomationTypePuppeteer] = []string{
		`puppeteer`,
		`__puppeteer`,
		`puppeteer_replaced`,
		`chrome-pdf`,
	}

	ad.detectionPatterns[AutomationTypePhantomJS] = []string{
		`phantomjs`,
		`__phantomjs`,
		`callPhantom`,
		`_phantom`,
		`phantom`,
	}

	ad.detectionPatterns[AutomationTypeHeadless] = []string{
		`headless`,
		`HeadlessChrome`,
		`Headless`,
		`navigator.webdriver`,
		`navigator.plugins`,
	}
}

func (ad *AutomationDetector) enableAll() {
	for at := range ad.detectionPatterns {
		ad.enabledTypes[at] = true
	}
}

func (ad *AutomationDetector) EnableDetection(at AutomationType) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	ad.enabledTypes[at] = true
}

func (ad *AutomationDetector) DisableDetection(at AutomationType) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	ad.enabledTypes[at] = false
}

func (ad *AutomationDetector) DetectAutomation() (bool, map[string]interface{}, error) {
	result := ad.performDetection()

	response := make(map[string]interface{})
	response["is_automated"] = result.IsAutomated
	response["confidence"] = result.Confidence
	response["detected_types"] = result.DetectedTypes
	response["timestamp"] = result.Timestamp.Unix()

	if result.IsAutomated {
		detections := make([]map[string]interface{}, len(result.Detections))
		for i, d := range result.Detections {
			detections[i] = map[string]interface{}{
				"type":       d.Type,
				"evidence":   d.Evidence,
				"severity":   d.Severity,
				"confidence": d.Confidence,
			}
		}
		response["detections"] = detections
	}

	return result.IsAutomated, response, nil
}

func (ad *AutomationDetector) performDetection() *DetectionResult {
	result := &DetectionResult{
		DetectedTypes: make([]AutomationType, 0),
		Detections:   make([]Detection, 0),
		Timestamp:    time.Now(),
	}

	ad.mu.RLock()
	enabledTypes := make(map[AutomationType]bool)
	for k, v := range ad.enabledTypes {
		enabledTypes[k] = v
	}
	patterns := make(map[AutomationType][]string)
	for k, v := range ad.detectionPatterns {
		patterns[k] = v
	}
	ad.mu.RUnlock()

	totalConfidence := 0.0
	detectionCount := 0

	for at, detectionPatterns := range patterns {
		if !enabledTypes[at] {
			continue
		}

		detection := ad.detectType(at, detectionPatterns)
		if detection != nil {
			result.Detections = append(result.Detections, *detection)
			result.DetectedTypes = append(result.DetectedTypes, at)
			totalConfidence += detection.Confidence
			detectionCount++
		}
	}

	if detectionCount > 0 {
		result.IsAutomated = true
		result.Confidence = totalConfidence / float64(detectionCount)
		if result.Confidence < 0.5 {
			result.Confidence = 0.5
		}
	}

	return result
}

func (ad *AutomationDetector) detectType(at AutomationType, patterns []string) *Detection {
	detector := ad.getDetectorFunc(at)
	if detector != nil {
		evidence, confidence := detector()
		if confidence > 0.5 {
			return &Detection{
				Type:       at,
				Evidence:   evidence,
				Severity:   ad.getSeverity(at),
				Confidence: confidence,
			}
		}
	}

	for _, pattern := range patterns {
		evidence, confidence := ad.checkPattern(pattern)
		if confidence > 0.7 {
			return &Detection{
				Type:       at,
				Evidence:   evidence,
				Severity:   ad.getSeverity(at),
				Confidence: confidence,
			}
		}
	}

	return nil
}

func (ad *AutomationDetector) getDetectorFunc(at AutomationType) func() (string, float64) {
	switch at {
	case AutomationTypeSelenium:
		return func() (string, float64) {
			return "webdriver detection", 0.85
		}
	case AutomationTypePlaywright:
		return func() (string, float64) {
			return "playwright detection", 0.85
		}
	case AutomationTypePuppeteer:
		return func() (string, float64) {
			return "puppeteer detection", 0.85
		}
	case AutomationTypeHeadless:
		return func() (string, float64) {
			return "headless browser detection", 0.75
		}
	}
	return nil
}

func (ad *AutomationDetector) checkPattern(pattern string) (string, float64) {
	return fmt.Sprintf("matched pattern: %s", pattern), 0.75
}

func (ad *AutomationDetector) getSeverity(at AutomationType) string {
	switch at {
	case AutomationTypeSelenium:
		return "high"
	case AutomationTypePlaywright:
		return "high"
	case AutomationTypePuppeteer:
		return "high"
	case AutomationTypePhantomJS:
		return "medium"
	case AutomationTypeHeadless:
		return "low"
	default:
		return "medium"
	}
}

func (ad *AutomationDetector) GenerateDetectionCode() string {
	detectionCode := `
(function(){
	var _0xauto = {
		detected: false,
		types: [],
		check: function() {
			var _0xd = [];
			
			if(navigator.webdriver) {
				_d.push('webdriver');
			}
			
			if(window.navigator.plugins && window.navigator.plugins.length < 3) {
				_d.push('headless');
			}
			
			if(window.callPhantom || window._phantom) {
				_d.push('phantomjs');
			}
			
			if(window.__selenium || window.__webdriver) {
				_d.push('selenium');
			}
			
			if(window.__playwright || window.__pw) {
				_d.push('playwright');
			}
			
			if(window.__puppeteer) {
				_d.push('puppeteer');
			}
			
			if(_d.length > 0) {
				this.detected = true;
				this.types = _d;
				document.documentElement.style.display = 'none';
				document.body.innerHTML = '<div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#000;color:#fff;display:flex;justify-content:center;align-items:center;font-family:Arial,sans-serif;z-index:2147483647;"><h1>Automated Access Denied</h1></div>';
			}
			
			return !this.detected;
		}
	};
	
	_0xauto.check();
	
	document.addEventListener('DOMContentLoaded', function() {
		setTimeout(function() {
			_0xauto.check();
		}, 1000);
	});
	
	window._0xauto = _0xauto;
})();
`
	return detectionCode
}

func (ad *AutomationDetector) GenerateEnhancedDetectionCode() string {
	enhancedCode := `
(function(){
	var _0xenh = {
		checks: [],
		violations: 0,
		maxViolations: 3,
		addCheck: function(_0xfn, _0xname) {
			this.checks.push({fn: _0xfn, name: _0xname});
		},
		runChecks: function() {
			var _0xres = true;
			for(var _0xi = 0; _0xi < this.checks.length; _0xi++) {
				try {
					if(!this.checks[_0xi].fn()) {
						this.violations++;
						_0xres = false;
					}
				} catch(_0xe) {
					this.violations++;
					_0xres = false;
				}
			}
			if(this.violations >= this.maxViolations) {
				this.block();
			}
			return _0xres;
		},
		block: function() {
			document.documentElement.style.display = 'none';
			document.body.innerHTML = '<div style="position:fixed;top:0;left:0;width:100%;height:100%;background:#000;color:#fff;display:flex;justify-content:center;align-items:center;font-family:Arial,sans-serif;z-index:2147483647;"><h1>Access Denied</h1></div>';
		}
	};
	
	_0xenh.addCheck(function() {
		return !(navigator.webdriver === true);
	}, 'webdriver');
	
	_0xenh.addCheck(function() {
		var _0xw = window.outerWidth - window.innerWidth;
		var _0xh = window.outerHeight - window.innerHeight;
		return !(_0xw > 160 || _0xh > 160);
	}, 'devtools-size');
	
	_0xenh.addCheck(function() {
		var _0xs = new Date().getTime();
		debugger;
		var _0xe = new Date().getTime();
		return (_0xe - _0xs) < 50;
	}, 'debugger');
	
	_0xenh.addCheck(function() {
		var _0fn = function() {};
		_0fn.toString = function() { return 'function()'; };
		console.log(_0fn);
		var _0xcc = console.clear;
		return true;
	}, 'console-check');
	
	window._0xenh = _0xenh;
	
	setInterval(function() {
		_0xenh.runChecks();
	}, 3000);
})();
`
	return enhancedCode
}

func (ad *AutomationDetector) AddCustomPattern(at AutomationType, pattern string) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	ad.detectionPatterns[at] = append(ad.detectionPatterns[at], pattern)
}

func (ad *AutomationDetector) RemovePattern(at AutomationType, pattern string) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	
	patterns := ad.detectionPatterns[at]
	for i, p := range patterns {
		if p == pattern {
			ad.detectionPatterns[at] = append(patterns[:i], patterns[i+1:]...)
			break
		}
	}
}

func (ad *AutomationDetector) GetEnabledTypes() []AutomationType {
	ad.mu.RLock()
	defer ad.mu.RUnlock()
	
	types := make([]AutomationType, 0)
	for at, enabled := range ad.enabledTypes {
		if enabled {
			types = append(types, at)
		}
	}
	return types
}

func (ad *AutomationDetector) SetEnabledTypes(types []AutomationType) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	
	ad.enabledTypes = make(map[AutomationType]bool)
	for _, at := range types {
		ad.enabledTypes[at] = true
	}
}

func (ad *AutomationDetector) GetStatistics() map[string]interface{} {
	ad.mu.RLock()
	defer ad.mu.RUnlock()
	
	stats := make(map[string]interface{})
	stats["total_patterns"] = 0
	stats["enabled_types"] = make([]AutomationType, 0)
	
	for at, patterns := range ad.detectionPatterns {
		stats["total_patterns"] = stats["total_patterns"].(int) + len(patterns)
		if ad.enabledTypes[at] {
			stats["enabled_types"] = append(stats["enabled_types"].([]AutomationType), at)
		}
	}
	
	return stats
}

type BehavioralAnalyzer struct {
	metrics     BehavioralMetrics
	thresholds  map[string]float64
	mu          sync.RWMutex
}

type BehavioralMetrics struct {
	MouseMovements   int       `json:"mouse_movements"`
	KeyPresses       int       `json:"key_presses"`
	ClickCount       int       `json:"click_count"`
	ScrollEvents     int       `json:"scroll_events"`
	FocusEvents      int       `json:"focus_events"`
	LastActivityTime time.Time `json:"last_activity_time"`
}

func NewBehavioralAnalyzer() *BehavioralAnalyzer {
	ba := &BehavioralAnalyzer{
		thresholds: map[string]float64{
			"min_mouse_movements":  10,
			"min_key_presses":       5,
			"min_click_count":      2,
			"min_scroll_events":    5,
			"activity_timeout":      300,
		},
	}
	ba.resetMetrics()
	return ba
}

func (ba *BehavioralAnalyzer) resetMetrics() {
	ba.metrics = BehavioralMetrics{
		LastActivityTime: time.Now(),
	}
}

func (ba *BehavioralAnalyzer) RecordMouseMovement() {
	ba.mu.Lock()
	defer ba.mu.Unlock()
	ba.metrics.MouseMovements++
	ba.metrics.LastActivityTime = time.Now()
}

func (ba *BehavioralAnalyzer) RecordKeyPress() {
	ba.mu.Lock()
	defer ba.mu.Unlock()
	ba.metrics.KeyPresses++
	ba.metrics.LastActivityTime = time.Now()
}

func (ba *BehavioralAnalyzer) RecordClick() {
	ba.mu.Lock()
	defer ba.mu.Unlock()
	ba.metrics.ClickCount++
	ba.metrics.LastActivityTime = time.Now()
}

func (ba *BehavioralAnalyzer) RecordScroll() {
	ba.mu.Lock()
	defer ba.mu.Unlock()
	ba.metrics.ScrollEvents++
	ba.metrics.LastActivityTime = time.Now()
}

func (ba *BehavioralAnalyzer) RecordFocus() {
	ba.mu.Lock()
	defer ba.mu.Unlock()
	ba.metrics.FocusEvents++
	ba.metrics.LastActivityTime = time.Now()
}

func (ba *BehavioralAnalyzer) AnalyzeBehavior() (bool, float64, []string) {
	ba.mu.RLock()
	defer ba.mu.RUnlock()

	anomalyScore := 0.0
	anomalies := make([]string, 0)

	if ba.metrics.MouseMovements < int(ba.thresholds["min_mouse_movements"]) {
		anomalyScore += 0.3
		anomalies = append(anomalies, "low_mouse_activity")
	}

	if ba.metrics.KeyPresses < int(ba.thresholds["min_key_presses"]) {
		anomalyScore += 0.2
		anomalies = append(anomalies, "low_keyboard_activity")
	}

	if ba.metrics.ClickCount < int(ba.thresholds["min_click_count"]) {
		anomalyScore += 0.2
		anomalies = append(anomalies, "low_click_activity")
	}

	if ba.metrics.ScrollEvents < int(ba.thresholds["min_scroll_events"]) {
		anomalyScore += 0.2
		anomalies = append(anomalies, "low_scroll_activity")
	}

	timeSinceActivity := time.Since(ba.metrics.LastActivityTime).Seconds()
	if timeSinceActivity > ba.thresholds["activity_timeout"] {
		anomalyScore += 0.1
		anomalies = append(anomalies, "inactivity_timeout")
	}

	isBot := anomalyScore >= 0.5
	return isBot, anomalyScore, anomalies
}

func (ba *BehavioralAnalyzer) GetMetrics() BehavioralMetrics {
	ba.mu.RLock()
	defer ba.mu.RUnlock()
	return ba.metrics
}

func (ba *BehavioralAnalyzer) SetThreshold(key string, value float64) {
	ba.mu.Lock()
	defer ba.mu.Unlock()
	ba.thresholds[key] = value
}

func GenerateAntiAutomationJavaScript() string {
	ad := NewAutomationDetector()
	return ad.GenerateEnhancedDetectionCode()
}

func DetectBrowserFingerprint(userAgent string, plugins []string, language string) map[string]interface{} {
	fingerprint := make(map[string]interface{})

	userAgentLower := strings.ToLower(userAgent)

	fingerprint["is_headless"] = strings.Contains(userAgentLower, "headless") ||
		strings.Contains(userAgentLower, "phantom") ||
		strings.Contains(userAgentLower, "slimer")

	fingerprint["is_mobile"] = regexp.MustCompile(`mobile|android|iphone`).MatchString(userAgentLower)

	fingerprint["has_webdriver"] = strings.Contains(userAgentLower, "webdriver")

	fingerprint["plugin_count"] = len(plugins)

	fingerprint["has_unusual_plugins"] = len(plugins) < 2

	fingerprint["language_match"] = regexp.MustCompile(`^[a-z]{2}-[A-Z]{2}$`).MatchString(language)

	return fingerprint
}
