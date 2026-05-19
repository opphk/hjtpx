package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type BotFingerprintV2 struct {
	FingerprintID string                 `json:"fingerprint_id"`
	CollectedAt   time.Time             `json:"collected_at"`
	Features      BotFingerprintFeatures `json:"features"`
	Calculated    BotCalculatedMetrics  `json:"calculated"`
}

type BotFingerprintFeatures struct {
	UserAgent            string   `json:"user_agent"`
	UserAgentParsed     UAInfo   `json:"ua_parsed"`
	AcceptLanguage      string   `json:"accept_language"`
	AcceptEncoding      string   `json:"accept_encoding"`
	Accept              string   `json:"accept"`
	Referer             string   `json:"referer"`
	Origin              string   `json:"origin"`
	CanvasFingerprint   string   `json:"canvas_fingerprint,omitempty"`
	WebGLFingerprint    string   `json:"webgl_fingerprint,omitempty"`
	AudioFingerprint    string   `json:"audio_fingerprint,omitempty"`
	ScreenResolution    string   `json:"screen_resolution"`
	ColorDepth          int      `json:"color_depth"`
	Timezone            string   `json:"timezone"`
	Language            string   `json:"language"`
	Platform            string   `json:"platform"`
	HardwareConcurrency int      `json:"hardware_concurrency"`
	DeviceMemory       int      `json:"device_memory"`
	TouchPoints         int      `json:"touch_points"`
	MaxTouchPoints      int      `json:"max_touch_points"`
	HasWebGL            bool     `json:"has_webgl"`
	HasWebGL2           bool     `json:"has_webgl2"`
	HasCanvas           bool     `json:"has_canvas"`
	HasWebRTC           bool     `json:"has_webrtc"`
	HasIndexedDB        bool     `json:"has_indexed_db"`
	HasLocalStorage      bool     `json:"has_local_storage"`
	HasSessionStorage    bool     `json:"has_session_storage"`
	HasCookie           bool     `json:"has_cookie"`
	Plugins             []string `json:"plugins"`
	Fonts               []string `json:"fonts"`
	ConnectionType      string   `json:"connection_type"`
	EffectiveType       string   `json:"effective_type"`
	Downlink            float64  `json:"downlink"`
	RTT                 int      `json:"rtt"`
	RequestHeaders      map[string]string `json:"request_headers"`
}

type UAInfo struct {
	Browser     string `json:"browser"`
	BrowserVersion string `json:"browser_version"`
	OS          string `json:"os"`
	OSVersion   string `json:"os_version"`
	Device      string `json:"device"`
	IsMobile    bool   `json:"is_mobile"`
	IsBot       bool   `json:"is_bot"`
	Engine      string `json:"engine"`
	EngineVersion string `json:"engine_version"`
}

type BotCalculatedMetrics struct {
	Entropy           float64            `json:"entropy"`
	SuspicionScore    float64            `json:"suspicion_score"`
	LegitimacyScore   float64            `json:"legitimacy_score"`
	AnomalyFlags     []string           `json:"anomaly_flags"`
	MatchedPatterns  []string           `json:"matched_patterns"`
	ConfidenceLevel   string              `json:"confidence_level"`
	Recommendation   string             `json:"recommendation"`
}

type BotFingerprintMatcherV2 struct {
	knownBotPatterns   []*regexp.Regexp
	automationPatterns  []*regexp.Regexp
	proxyPatterns      []*regexp.Regexp
	vpnPatterns        []*regexp.Regexp
	humanPatterns      []*regexp.Regexp
	featureWeights     map[string]float64
}

func NewBotFingerprintMatcherV2() *BotFingerprintMatcherV2 {
	return &BotFingerprintMatcherV2{
		knownBotPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)bot|crawler|spider|scraper|curl|wget|python|java|go-http`),
			regexp.MustCompile(`(?i)Googlebot|Bingbot|Yahoo|Baidu|Sogou|Yandex`),
			regexp.MustCompile(`(?i)Ahrefs|Semrush|Moz|Majestic|Seznam`),
			regexp.MustCompile(`(?i)LinkedInBot|Facebook|Twitterbot`),
		},
		automationPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)webdriver|selenium|phantom|headless`),
			regexp.MustCompile(`(?i)chrome-automation|automation-extension`),
			regexp.MustCompile(`(?i)puppeteer|playwright|nightmare`),
		},
		proxyPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)proxy| vpn |hide.my|anonymizer`),
		},
		vpnPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)nordvpn|expressvpn|cyberghost|hotspotshield`),
			regexp.MustCompile(`(?i)protonvpn|surfshark|ipvanish| mullvad`),
		},
		humanPatterns: []*regexp.Regexp{
			regexp.MustCompile(`Mozilla/5\.0 \(Windows NT 10\.0`),
			regexp.MustCompile(`Mozilla/5\.0 \(Macintosh; Intel Mac OS X`),
			regexp.MustCompile(`Mozilla/5\.0 \(iPhone; CPU iPhone`),
			regexp.MustCompile(`Mozilla/5\.0 \(Linux; Android`),
		},
		featureWeights: map[string]float64{
			"user_agent":              0.15,
			"canvas_fingerprint":       0.10,
			"webgl_fingerprint":       0.10,
			"screen_resolution":       0.08,
			"timezone":                0.05,
			"language":                0.05,
			"platform":                0.05,
			"plugins":                0.08,
			"fonts":                   0.10,
			"automation_indicators":   0.12,
			"network_characteristics": 0.12,
		},
	}
}

func (m *BotFingerprintMatcherV2) Match(fingerprint *BotFingerprintV2) BotDetectionResultV2 {
	result := BotDetectionResultV2{
		IsBot:       false,
		ShouldBlock: false,
		RiskScore:   0,
		Reasons:     make([]string, 0),
		Features:    make([]MatchedFeature, 0),
		Confidence:  0,
	}

	if fingerprint == nil {
		return result
	}

	m.matchUserAgent(fingerprint, &result)
	m.matchAutomationIndicators(fingerprint, &result)
	m.matchFingerprints(fingerprint, &result)
	m.matchNetworkCharacteristics(fingerprint, &result)
	m.matchHumanPatterns(fingerprint, &result)

	result.calculateFinalScore()
	result.calculateConfidence()
	result.determineRecommendation()

	return result
}

func (m *BotFingerprintMatcherV2) matchUserAgent(fingerprint *BotFingerprintV2, result *BotDetectionResultV2) {
	ua := fingerprint.Features.UserAgent

	for _, pattern := range m.knownBotPatterns {
		if pattern.MatchString(ua) {
			result.RiskScore += 0.40
			result.Reasons = append(result.Reasons, fmt.Sprintf("Known bot pattern: %s", pattern.String()))
			result.Features = append(result.Features, MatchedFeature{
				Name:        "user_agent",
				MatchType:   "known_bot",
				Weight:      0.40,
				Description: "User agent matches known bot pattern",
			})
			result.IsBot = true
			return
		}
	}

	for _, pattern := range m.automationPatterns {
		if pattern.MatchString(ua) {
			result.RiskScore += 0.50
			result.Reasons = append(result.Reasons, "Automation indicator in User-Agent")
			result.Features = append(result.Features, MatchedFeature{
				Name:        "user_agent",
				MatchType:   "automation",
				Weight:      0.50,
				Description: "User agent contains automation indicators",
			})
			result.ShouldBlock = true
		}
	}

	if ua == "" || len(ua) < 20 {
		result.RiskScore += 0.25
		result.Reasons = append(result.Reasons, "Suspiciously short or empty User-Agent")
		result.Features = append(result.Features, MatchedFeature{
			Name:        "user_agent",
			MatchType:   "suspicious",
			Weight:      0.25,
			Description: "User-Agent is too short or empty",
		})
	}

	if fingerprint.Features.UserAgentParsed.IsBot {
		result.RiskScore += 0.35
		result.Reasons = append(result.Reasons, "Parsed as bot from User-Agent")
		result.Features = append(result.Features, MatchedFeature{
			Name:        "user_agent",
			MatchType:   "parsed_bot",
			Weight:      0.35,
			Description: "User-Agent parsing detected bot",
		})
	}
}

func (m *BotFingerprintMatcherV2) matchAutomationIndicators(fingerprint *BotFingerprintV2, result *BotDetectionResultV2) {
	features := fingerprint.Features

	if !features.HasCanvas || features.CanvasFingerprint == "" {
		result.RiskScore += 0.15
		result.Reasons = append(result.Reasons, "Missing Canvas fingerprint")
		result.Features = append(result.Features, MatchedFeature{
			Name:        "canvas_fingerprint",
			MatchType:   "missing",
			Weight:      0.15,
			Description: "Canvas fingerprint not detected",
		})
	}

	if !features.HasWebGL || features.WebGLFingerprint == "" {
		result.RiskScore += 0.10
		result.Reasons = append(result.Reasons, "Missing WebGL fingerprint")
		result.Features = append(result.Features, MatchedFeature{
			Name:        "webgl_fingerprint",
			MatchType:   "missing",
			Weight:      0.10,
			Description: "WebGL fingerprint not detected",
		})
	}

	if features.HasWebRTC && features.HasWebGL2 && features.HasCanvas {
		result.RiskScore -= 0.10
	}

	if features.Platform == "" || features.Platform == "unknown" {
		result.RiskScore += 0.10
		result.Reasons = append(result.Reasons, "Unknown platform")
		result.Features = append(result.Features, MatchedFeature{
			Name:        "platform",
			MatchType:   "unknown",
			Weight:      0.10,
			Description: "Platform not detected",
		})
	}

	if features.Timezone == "" || features.Timezone == "unknown" {
		result.RiskScore += 0.08
		result.Reasons = append(result.Reasons, "Missing timezone information")
		result.Features = append(result.Features, MatchedFeature{
			Name:        "timezone",
			MatchType:   "missing",
			Weight:      0.08,
			Description: "Timezone not detected",
		})
	}
}

func (m *BotFingerprintMatcherV2) matchFingerprints(fingerprint *BotFingerprintV2, result *BotDetectionResultV2) {
	canvasFP := fingerprint.Features.CanvasFingerprint
	webglFP := fingerprint.Features.WebGLFingerprint

	if canvasFP != "" && webglFP != "" {
		if canvasFP == webglFP {
			result.RiskScore += 0.30
			result.Reasons = append(result.Reasons, "Canvas and WebGL fingerprints are identical")
			result.Features = append(result.Features, MatchedFeature{
				Name:        "fingerprint_consistency",
				MatchType:   "anomaly",
				Weight:      0.30,
				Description: "Canvas and WebGL fingerprints match - likely spoofed",
			})
		}

		if len(canvasFP) < 20 || len(webglFP) < 20 {
			result.RiskScore += 0.20
			result.Reasons = append(result.Reasons, "Suspiciously short fingerprints")
			result.Features = append(result.Features, MatchedFeature{
				Name:        "fingerprint_length",
				MatchType:   "suspicious",
				Weight:      0.20,
				Description: "Fingerprints are unusually short",
			})
		}
	}

	if len(fingerprint.Features.Plugins) == 0 {
		result.RiskScore += 0.12
		result.Reasons = append(result.Reasons, "No browser plugins detected")
		result.Features = append(result.Features, MatchedFeature{
			Name:        "plugins",
			MatchType:   "missing",
			Weight:      0.12,
			Description: "No plugins detected - unusual for real browser",
		})
	}

	if len(fingerprint.Features.Fonts) < 5 {
		result.RiskScore += 0.15
		result.Reasons = append(result.Reasons, "Limited font detection - possible headless browser")
		result.Features = append(result.Features, MatchedFeature{
			Name:        "fonts",
			MatchType:   "limited",
			Weight:      0.15,
			Description: "Fewer fonts detected than expected",
		})
	}
}

func (m *BotFingerprintMatcherV2) matchNetworkCharacteristics(fingerprint *BotFingerprintV2, result *BotDetectionResultV2) {
	features := fingerprint.Features

	if features.EffectiveType == "slow-2g" || features.EffectiveType == "2g" {
		result.RiskScore += 0.05
		result.Reasons = append(result.Reasons, "Very slow network connection")
	}

	if features.RTT == 0 {
		result.RiskScore += 0.08
		result.Reasons = append(result.Reasons, "No round-trip time detected")
		result.Features = append(result.Features, MatchedFeature{
			Name:        "network_rtt",
			MatchType:   "missing",
			Weight:      0.08,
			Description: "RTT is 0 - unusual for real network",
		})
	}

	if features.HardwareConcurrency == 0 || features.HardwareConcurrency > 32 {
		result.RiskScore += 0.10
		result.Reasons = append(result.Reasons, "Suspicious hardware concurrency value")
		result.Features = append(result.Features, MatchedFeature{
			Name:        "hardware_concurrency",
			MatchType:   "suspicious",
			Weight:      0.10,
			Description: fmt.Sprintf("Hardware concurrency is %d", features.HardwareConcurrency),
		})
	}

	if features.DeviceMemory == 0 {
		result.RiskScore += 0.05
		result.Reasons = append(result.Reasons, "Device memory not detected")
	}

	if features.TouchPoints == 0 && features.MaxTouchPoints > 0 {
		result.RiskScore += 0.05
		result.Reasons = append(result.Reasons, "Inconsistent touch point detection")
	}
}

func (m *BotFingerprintMatcherV2) matchHumanPatterns(fingerprint *BotFingerprintV2, result *BotDetectionResultV2) {
	ua := fingerprint.Features.UserAgent

	for _, pattern := range m.humanPatterns {
		if pattern.MatchString(ua) {
			result.RiskScore -= 0.15
			result.Features = append(result.Features, MatchedFeature{
				Name:        "user_agent",
				MatchType:   "human_pattern",
				Weight:      -0.15,
				Description: "Matches common human browser pattern",
			})
			break
		}
	}

	if fingerprint.Features.HasCookie && 
	   fingerprint.Features.HasLocalStorage && 
	   fingerprint.Features.HasSessionStorage {
		result.RiskScore -= 0.10
		result.Features = append(result.Features, MatchedFeature{
			Name:        "storage_capabilities",
			MatchType:   "human_indicator",
			Weight:      -0.10,
			Description: "All storage capabilities present - human browser",
		})
	}
}

type BotDetectionResultV2 struct {
	IsBot         bool             `json:"is_bot"`
	ShouldBlock   bool             `json:"should_block"`
	RiskScore     float64          `json:"risk_score"`
	Reasons       []string         `json:"reasons"`
	Features      []MatchedFeature `json:"matched_features"`
	Confidence    float64          `json:"confidence"`
	Recommendation string          `json:"recommendation"`
}

type MatchedFeature struct {
	Name        string  `json:"name"`
	MatchType   string  `json:"match_type"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
}

func (r *BotDetectionResultV2) calculateFinalScore() {
	r.RiskScore = math.Max(0, math.Min(100, r.RiskScore*100))
}

func (r *BotDetectionResultV2) calculateConfidence() {
	if len(r.Features) == 0 {
		r.Confidence = 30
		return
	}

	highWeightFeatures := 0
	for _, f := range r.Features {
		if math.Abs(f.Weight) >= 0.20 {
			highWeightFeatures++
		}
	}

	r.Confidence = math.Min(100, float64(len(r.Features)*10)+float64(highWeightFeatures*15))
}

func (r *BotDetectionResultV2) determineRecommendation() {
	switch {
	case r.RiskScore >= 80:
		r.Recommendation = "BLOCK"
		r.ShouldBlock = true
		r.IsBot = true
	case r.RiskScore >= 60:
		r.Recommendation = "CHALLENGE"
		r.ShouldBlock = false
		r.IsBot = true
	case r.RiskScore >= 40:
		r.Recommendation = "VERIFY"
		r.ShouldBlock = false
	case r.RiskScore >= 20:
		r.Recommendation = "MONITOR"
		r.ShouldBlock = false
	default:
		r.Recommendation = "ALLOW"
		r.ShouldBlock = false
		r.IsBot = false
	}
}

func (s *BotDetectionService) DetectBotV2(fingerprint *BotFingerprintV2) *BotDetectionResultV2 {
	matcher := NewBotFingerprintMatcherV2()
	result := matcher.Match(fingerprint)
	
	fingerprint.Calculated.SuspicionScore = result.RiskScore
	fingerprint.Calculated.ConfidenceLevel = fmt.Sprintf("%.0f%%", result.Confidence)
	fingerprint.Calculated.Recommendation = result.Recommendation
	
	return &result
}

func CalculateFingerprintEntropy(fingerprint string) float64 {
	if len(fingerprint) == 0 {
		return 0
	}

	charCounts := make(map[rune]int)
	totalChars := 0

	for _, char := range fingerprint {
		charCounts[char]++
		totalChars++
	}

	entropy := 0.0
	for _, count := range charCounts {
		if count > 0 {
			p := float64(count) / float64(totalChars)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (s *BotDetectionService) UpdateFingerprintLibraryV2(fingerprintID string, features *BotFingerprintFeatures) error {
	fpKey := s.generateFingerprintKey(features)
	
	s.mu.Lock()
	defer s.mu.Unlock()

	s.fingerprintLibrary[fpKey] = &FingerprintEntry{
		FingerprintID: fingerprintID,
		Features:     features,
		FirstSeen:    time.Now(),
		LastSeen:     time.Now(),
		HitCount:     1,
		IsVerified:   false,
	}
	
	return nil
}

func (s *BotDetectionService) GenerateFingerprintReport() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	report := map[string]interface{}{
		"total_fingerprints": len(s.fingerprintLibrary),
		"generated_at":        time.Now(),
		"bot_fingerprints":    0,
		"human_fingerprints":   0,
		"unknown_fingerprints": 0,
		"top_indicators":      []string{},
		"average_confidence":   0.0,
	}
	
	botCount := 0
	humanCount := 0
	totalConfidence := 0.0
	
	for _, entry := range s.fingerprintLibrary {
		if entry.IsVerified {
			if entry.RiskScore >= 50 {
				botCount++
			} else {
				humanCount++
			}
			totalConfidence += entry.RiskScore
		} else {
			report["unknown_fingerprints"] = report["unknown_fingerprints"].(int) + 1
		}
	}
	
	report["bot_fingerprints"] = botCount
	report["human_fingerprints"] = humanCount
	
	if botCount+humanCount > 0 {
		report["average_confidence"] = totalConfidence / float64(botCount+humanCount)
	}
	
	return report
}
