package service

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"net/http"
	"regexp"
	"sync"
	"time"
)

var (
	botUserAgentPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)bot`),
		regexp.MustCompile(`(?i)crawler`),
		regexp.MustCompile(`(?i)spider`),
		regexp.MustCompile(`(?i)scraper`),
		regexp.MustCompile(`(?i)curl`),
		regexp.MustCompile(`(?i)wget`),
		regexp.MustCompile(`(?i)python-requests`),
		regexp.MustCompile(`(?i)scrapy`),
		regexp.MustCompile(`(?i)selenium`),
		regexp.MustCompile(`(?i)headless`),
		regexp.MustCompile(`(?i)phantom`),
		regexp.MustCompile(`(?i)puppeteer`),
		regexp.MustCompile(`(?i)playwright`),
		regexp.MustCompile(`(?i)googlebot`),
		regexp.MustCompile(`(?i)bingbot`),
		regexp.MustCompile(`(?i)slurp`),
		regexp.MustCompile(`(?i)duckduckbot`),
		regexp.MustCompile(`(?i)baiduspider`),
		regexp.MustCompile(`(?i)yandexbot`),
		regexp.MustCompile(`(?i)sogou`),
		regexp.MustCompile(`(?i)exabot`),
		regexp.MustCompile(`(?i)facebot`),
		regexp.MustCompile(`(?i)ia_archiver`),
	}

	suspiciousHeaderNames = []string{
		"X-Scanner",
		"X-Forwarded-For",
		"Via",
		"X-ProxyUser-Ip",
		"X-Originating-IP",
		"X-Remote-IP",
		"X-Proxy-IP",
		"X-Client-IP",
		"X-Real-IP",
		"X-Forwarded",
		"X-Forwarded-Host",
		"Forwarded-For",
		"X-Cluster-Client-IP",
	}

	automationIndicatorPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)webdriver`),
		regexp.MustCompile(`(?i)selenium`),
		regexp.MustCompile(`(?i)chrome-automation`),
		regexp.MustCompile(`(?i)automation`),
		regexp.MustCompile(`(?i)test`),
	}
)

type BotFingerprintData struct {
	FingerprintID string
	FirstSeen     time.Time
	LastSeen      time.Time
	RequestCount  int
	RiskScore     float64
	IsBlacklisted bool
	UserAgent     string
	IP            string
}

type BotBehaviorData struct {
	IP             string
	RequestTimes   []time.Time
	RequestPaths   []string
	Methods        []string
	RequestCount   int
	LastActivity   time.Time
	AvgInterval    float64
	IsRegular      bool
}

type BotDetectionResult struct {
	IsBot          bool
	ShouldBlock    bool
	RiskScore      float64
	Reasons        []string
	ChallengeType  string
	Confidence     float64
}

type BotDetectionService struct {
	fingerprints    map[string]*BotFingerprintData
	behaviors       map[string]*BotBehaviorData
	mu              sync.RWMutex
	botPatterns     []*regexp.Regexp
	headerPatterns  []string
	autoIndicators  []*regexp.Regexp
	maxFingerprints int
	maxBehaviors    int
}

func NewBotDetectionService() *BotDetectionService {
	return &BotDetectionService{
		fingerprints:    make(map[string]*BotFingerprintData),
		behaviors:       make(map[string]*BotBehaviorData),
		botPatterns:     botUserAgentPatterns,
		headerPatterns:  suspiciousHeaderNames,
		autoIndicators:  automationIndicatorPatterns,
		maxFingerprints: 10000,
		maxBehaviors:    10000,
	}
}

func (s *BotDetectionService) DetectBot(r *http.Request, additionalData map[string]string) *BotDetectionResult {
	ip := getClientIP(r)
	userAgent := r.UserAgent()

	result := &BotDetectionResult{
		IsBot:         false,
		ShouldBlock:   false,
		RiskScore:     0.0,
		Reasons:       []string{},
		ChallengeType: "",
		Confidence:    0.0,
	}

	score := 0.0
	confidence := 0.0

	if s.checkUserAgent(userAgent, result) {
		score += 0.5
		confidence += 0.4
	}

	if s.checkHeaders(r, result) {
		score += 0.2
		confidence += 0.2
	}

	if s.checkBehavior(ip, r, result) {
		score += 0.2
		confidence += 0.3
	}

	if s.checkFingerprint(ip, userAgent, additionalData, result) {
		score += 0.1
		confidence += 0.1
	}

	// Check if fingerprint is blacklisted
	s.mu.RLock()
	fingerprintID := s.generateFingerprintID(ip, userAgent, additionalData)
	if fp, exists := s.fingerprints[fingerprintID]; exists && fp.IsBlacklisted {
		result.IsBot = true
		result.ShouldBlock = true
		result.RiskScore = 1.0
		result.Confidence = 0.9
		result.Reasons = append(result.Reasons, "Fingerprint blacklisted")
	}
	s.mu.RUnlock()

	// If not already blocked by blacklist, apply threshold
	if !result.IsBot {
		result.RiskScore = math.Min(score, 1.0)
		result.Confidence = math.Min(confidence, 1.0)

		if result.RiskScore >= 0.7 {
			result.IsBot = true
			result.ShouldBlock = true
			result.ChallengeType = "captcha"
		} else if result.RiskScore >= 0.4 {
			result.IsBot = true
			result.ChallengeType = "js_challenge"
		}
	}

	return result
}

func (s *BotDetectionService) checkUserAgent(userAgent string, result *BotDetectionResult) bool {
	for _, pattern := range s.botPatterns {
		if pattern.MatchString(userAgent) {
			result.Reasons = append(result.Reasons, "Suspicious user agent")
			return true
		}
	}
	return false
}

func (s *BotDetectionService) checkHeaders(r *http.Request, result *BotDetectionResult) bool {
	suspicious := false
	for _, header := range s.headerPatterns {
		if r.Header.Get(header) != "" {
			suspicious = true
			result.Reasons = append(result.Reasons, "Suspicious header: "+header)
			break
		}
	}
	return suspicious
}

func (s *BotDetectionService) checkBehavior(ip string, r *http.Request, result *BotDetectionResult) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	behavior, exists := s.behaviors[ip]
	if !exists {
		behavior = &BotBehaviorData{
			IP:           ip,
			RequestTimes: []time.Time{},
			RequestPaths: []string{},
			Methods:      []string{},
			RequestCount: 0,
			LastActivity: now,
		}
		s.behaviors[ip] = behavior
		if len(s.behaviors) > s.maxBehaviors {
			s.cleanupOldBehaviors()
		}
	}

	behavior.RequestTimes = append(behavior.RequestTimes, now)
	behavior.RequestPaths = append(behavior.RequestPaths, r.URL.Path)
	behavior.Methods = append(behavior.Methods, r.Method)
	behavior.RequestCount++
	behavior.LastActivity = now

	if len(behavior.RequestTimes) > 100 {
		behavior.RequestTimes = behavior.RequestTimes[len(behavior.RequestTimes)-100:]
		behavior.RequestPaths = behavior.RequestPaths[len(behavior.RequestPaths)-100:]
		behavior.Methods = behavior.Methods[len(behavior.Methods)-100:]
	}

	if behavior.RequestCount > 50 {
		avgInterval := s.calculateAvgInterval(behavior.RequestTimes)
		if avgInterval > 0 && avgInterval < 100 {
			result.Reasons = append(result.Reasons, "Unusually frequent requests")
			return true
		}
	}

	return false
}

func (s *BotDetectionService) checkFingerprint(ip string, userAgent string, additionalData map[string]string, result *BotDetectionResult) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	fingerprintID := s.generateFingerprintID(ip, userAgent, additionalData)
	fingerprint, exists := s.fingerprints[fingerprintID]
	if !exists {
		fingerprint = &BotFingerprintData{
			FingerprintID: fingerprintID,
			FirstSeen:     time.Now(),
			LastSeen:      time.Now(),
			RequestCount:  0,
			RiskScore:     0.0,
			IsBlacklisted: false,
			UserAgent:     userAgent,
			IP:            ip,
		}
		s.fingerprints[fingerprintID] = fingerprint
		if len(s.fingerprints) > s.maxFingerprints {
			s.cleanupOldFingerprints()
		}
	}

	fingerprint.RequestCount++
	fingerprint.LastSeen = time.Now()

	if fingerprint.IsBlacklisted {
		result.Reasons = append(result.Reasons, "Fingerprint blacklisted")
		return true
	}

	return false
}

func (s *BotDetectionService) generateFingerprintID(ip string, userAgent string, additionalData map[string]string) string {
	hasher := sha256.New()
	hasher.Write([]byte(ip))
	hasher.Write([]byte(userAgent))
	for k, v := range additionalData {
		hasher.Write([]byte(k + ":" + v))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *BotDetectionService) calculateAvgInterval(times []time.Time) float64 {
	if len(times) < 2 {
		return 0
	}
	total := 0.0
	for i := 1; i < len(times); i++ {
		total += float64(times[i].Sub(times[i-1]).Milliseconds())
	}
	return total / float64(len(times)-1)
}

func (s *BotDetectionService) cleanupOldFingerprints() {
	cutoff := time.Now().Add(-24 * time.Hour)
	for id, fp := range s.fingerprints {
		if fp.LastSeen.Before(cutoff) {
			delete(s.fingerprints, id)
		}
	}
}

func (s *BotDetectionService) cleanupOldBehaviors() {
	cutoff := time.Now().Add(-24 * time.Hour)
	for ip, bh := range s.behaviors {
		if bh.LastActivity.Before(cutoff) {
			delete(s.behaviors, ip)
		}
	}
}

func (s *BotDetectionService) AddToBlacklist(ip string, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, fp := range s.fingerprints {
		if fp.IP == ip {
			fp.IsBlacklisted = true
		}
	}
}

func (s *BotDetectionService) RemoveFromBlacklist(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, fp := range s.fingerprints {
		if fp.IP == ip {
			fp.IsBlacklisted = false
		}
	}
}
