package service

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

func getClientIPFromRequest(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

type CrawlerType string

const (
	CrawlerTypeUnknown    CrawlerType = "unknown"
	CrawlerTypeSearchBot CrawlerType = "search_bot"
	CrawlerTypeSocialBot CrawlerType = "social_bot"
	CrawlerTypeRSSReader CrawlerType = "rss_reader"
	CrawlerTypeAPIProxy  CrawlerType = "api_proxy"
	CrawlerTypeMalicious CrawlerType = "malicious"
	CrawlerTypeHeadless  CrawlerType = "headless_browser"
	CrawlerTypeAutomated CrawlerType = "automated_tool"
)

type CrawlerSignature struct {
	Type          CrawlerType
	Name          string
	Confidence    float64
	AllowSearch   bool
	AllowSocial   bool
	AllowRSS      bool
	AllowAPI      bool
	IsMalicious   bool
}

type BehaviorMetrics struct {
	RequestInterval   float64
	RequestVariance   float64
	PathEntropy       float64
	SessionDuration   time.Duration
	RequestCount      int
	UniquePaths       int
	PathDistribution  map[string]int
	MethodDistribution map[string]int
	IsRegularPattern  bool
	IsSequentialPath  bool
	HasJavaScriptUA   bool
}

type CrawlerDetectionResult struct {
	IsCrawler        bool
	CrawlerType      CrawlerType
	Confidence       float64
	Reasons          []string
	Signatures       []string
	BehaviorMetrics  *BehaviorMetrics
	RiskLevel        string
	RecommendedAction string
	ChallengeType    string
}

type CrawlerEnhancedDetectionService struct {
	mu              sync.RWMutex
	signatures      map[string]*CrawlerSignature
	behaviorCache   map[string]*BehaviorMetrics
	requestHistory  map[string][]*RequestRecord
	headlessPatterns []*regexp.Regexp
	cryptoPatterns  []*regexp.Regexp
	knownBots       map[string]*CrawlerSignature
	maxHistorySize  int
}

type RequestRecord struct {
	Timestamp   time.Time
	Path        string
	Method      string
	UserAgent   string
	Fingerprint string
	Interval    time.Duration
}

var knownBotSignatures = map[string]*CrawlerSignature{
	"googlebot": {
		Type:        CrawlerTypeSearchBot,
		Name:        "Googlebot",
		Confidence:  0.95,
		AllowSearch: true,
	},
	"bingbot": {
		Type:        CrawlerTypeSearchBot,
		Name:        "Bingbot",
		Confidence:  0.95,
		AllowSearch: true,
	},
	"yandex": {
		Type:        CrawlerTypeSearchBot,
		Name:        "Yandex Bot",
		Confidence:  0.90,
		AllowSearch: true,
	},
	"baiduspider": {
		Type:        CrawlerTypeSearchBot,
		Name:        "Baidu Spider",
		Confidence:  0.85,
		AllowSearch: true,
	},
	"duckduckbot": {
		Type:        CrawlerTypeSearchBot,
		Name:        "DuckDuckBot",
		Confidence:  0.95,
		AllowSearch: true,
	},
	"facebookexternalhit": {
		Type:        CrawlerTypeSocialBot,
		Name:        "Facebook Bot",
		Confidence:  0.90,
		AllowSocial: true,
	},
	"twitterbot": {
		Type:        CrawlerTypeSocialBot,
		Name:        "Twitter Bot",
		Confidence:  0.90,
		AllowSocial: true,
	},
	"linkedinbot": {
		Type:        CrawlerTypeSocialBot,
		Name:        "LinkedIn Bot",
		Confidence:  0.85,
		AllowSocial: true,
	},
	"redditbot": {
		Type:        CrawlerTypeSocialBot,
		Name:        "Reddit Bot",
		Confidence:  0.85,
		AllowSocial: true,
	},
	"curl": {
		Type:        CrawlerTypeAPIProxy,
		Name:        "cURL",
		Confidence:  0.70,
		AllowAPI:   true,
	},
	"wget": {
		Type:        CrawlerTypeAPIProxy,
		Name:        "Wget",
		Confidence:  0.70,
		AllowAPI:   true,
	},
	"python-requests": {
		Type:        CrawlerTypeAPIProxy,
		Name:        "Python Requests",
		Confidence:  0.65,
		AllowAPI:   true,
	},
	"scrapy": {
		Type:        CrawlerTypeMalicious,
		Name:        "Scrapy",
		Confidence:  0.80,
		IsMalicious: true,
	},
	"selenium": {
		Type:        CrawlerTypeHeadless,
		Name:        "Selenium",
		Confidence:  0.85,
	},
	"phantomjs": {
		Type:        CrawlerTypeHeadless,
		Name:        "PhantomJS",
		Confidence:  0.90,
	},
	"puppeteer": {
		Type:        CrawlerTypeHeadless,
		Name:        "Puppeteer",
		Confidence:  0.85,
	},
	"playwright": {
		Type:        CrawlerTypeHeadless,
		Name:        "Playwright",
		Confidence:  0.85,
	},
	"webdriver": {
		Type:        CrawlerTypeHeadless,
		Name:        "WebDriver",
		Confidence:  0.90,
	},
	"headless": {
		Type:        CrawlerTypeHeadless,
		Name:        "Headless Chrome",
		Confidence:  0.85,
	},
}

func NewCrawlerEnhancedDetectionService() *CrawlerEnhancedDetectionService {
	return &CrawlerEnhancedDetectionService{
		signatures:      make(map[string]*CrawlerSignature),
		behaviorCache:   make(map[string]*BehaviorMetrics),
		requestHistory:  make(map[string][]*RequestRecord),
		maxHistorySize:  1000,
		knownBots:       knownBotSignatures,
		headlessPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)headless`),
			regexp.MustCompile(`(?i)phantom`),
			regexp.MustCompile(`(?i)puppeteer`),
			regexp.MustCompile(`(?i)playwright`),
			regexp.MustCompile(`(?i)selenium`),
			regexp.MustCompile(`(?i)webdriver`),
			regexp.MustCompile(`(?i)chrome-lighthouse`),
		},
		cryptoPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)crypto`),
			regexp.MustCompile(`(?i)coinlib`),
			regexp.MustCompile(`(?i)coinhive`),
		},
	}
}

func (s *CrawlerEnhancedDetectionService) DetectCrawler(r *http.Request, additionalData map[string]string) *CrawlerDetectionResult {
	ip := getClientIPFromRequest(r)
	userAgent := r.UserAgent()
	fingerprint := s.generateFingerprint(ip, userAgent, additionalData)

	result := &CrawlerDetectionResult{
		Reasons:   []string{},
		Signatures: []string{},
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.recordRequest(ip, r, fingerprint)

	signatureScore := s.checkSignatures(userAgent, r, result)
	behaviorScore := s.analyzeBehavior(ip, r, result)
	automationScore := s.detectAutomation(userAgent, r, result)
	patternScore := s.analyzeRequestPattern(ip, r, result)

	totalScore := (signatureScore + behaviorScore + automationScore + patternScore) / 4
	result.Confidence = math.Min(totalScore, 1.0)

	if totalScore >= 0.7 {
		result.IsCrawler = true
		result.RiskLevel = "high"
		result.RecommendedAction = "block"
		result.ChallengeType = "captcha"
	} else if totalScore >= 0.5 {
		result.IsCrawler = true
		result.RiskLevel = "medium"
		result.RecommendedAction = "challenge"
		result.ChallengeType = "js_challenge"
	} else if totalScore >= 0.3 {
		result.IsCrawler = true
		result.RiskLevel = "low"
		result.RecommendedAction = "allow_with_log"
	} else {
		result.IsCrawler = false
		result.RiskLevel = "none"
		result.RecommendedAction = "allow"
	}

	result.CrawlerType = s.determineCrawlerType(result)

	return result
}

func (s *CrawlerEnhancedDetectionService) checkSignatures(userAgent string, r *http.Request, result *CrawlerDetectionResult) float64 {
	score := 0.0

	userAgentLower := strings.ToLower(userAgent)

	for botName, signature := range s.knownBots {
		if strings.Contains(userAgentLower, botName) {
			score += signature.Confidence * 0.4
			result.Signatures = append(result.Signatures, "Known bot: "+signature.Name)
			result.Reasons = append(result.Reasons, "Matches known bot signature: "+signature.Name)

			if signature.IsMalicious {
				score += 0.3
				result.Reasons = append(result.Reasons, "Malicious bot detected: "+signature.Name)
			}
		}
	}

	headers := r.Header
	requiredBotHeaders := []string{"Accept", "Accept-Language", "Accept-Encoding"}
	missingHeaders := 0
	for _, header := range requiredBotHeaders {
		if headers.Get(header) == "" {
			missingHeaders++
		}
	}
	if missingHeaders >= 2 {
		score += 0.2
		result.Reasons = append(result.Reasons, "Missing standard HTTP headers")
	}

	acceptHeader := headers.Get("Accept")
	if acceptHeader == "" || acceptHeader == "*/*" {
		score += 0.1
		result.Reasons = append(result.Reasons, "Suspicious Accept header")
	}

	if headers.Get("Accept-Language") == "" {
		score += 0.05
		result.Reasons = append(result.Reasons, "Missing Accept-Language header")
	}

	encodingHeader := headers.Get("Accept-Encoding")
	if encodingHeader == "" {
		score += 0.05
	} else if strings.Contains(encodingHeader, "gzip") && strings.Contains(encodingHeader, "deflate") {
		score -= 0.1
	}

	return math.Min(score, 1.0)
}

func (s *CrawlerEnhancedDetectionService) analyzeBehavior(ip string, r *http.Request, result *CrawlerDetectionResult) float64 {
	score := 0.0

	history, exists := s.requestHistory[ip]
	if !exists || len(history) < 5 {
		return 0.0
	}

	metrics := s.calculateBehaviorMetrics(history)
	result.BehaviorMetrics = metrics

	if metrics.RequestInterval > 0 && metrics.RequestInterval < 100 {
		score += 0.3
		result.Reasons = append(result.Reasons, "Unusually fast request interval")
	}

	if metrics.RequestVariance < 50 && metrics.RequestCount > 20 {
		score += 0.25
		result.Reasons = append(result.Reasons, "Suspiciously regular request pattern")
	}

	if metrics.UniquePaths > 0 {
		uniqueRatio := float64(metrics.UniquePaths) / float64(metrics.RequestCount)
		if uniqueRatio < 0.1 && metrics.RequestCount > 50 {
			score += 0.2
			result.Reasons = append(result.Reasons, "Low path diversity - possible scraping")
		}
	}

	if metrics.IsSequentialPath {
		score += 0.15
		result.Reasons = append(result.Reasons, "Sequential path access pattern")
	}

	allowedMethods := map[string]bool{"GET": true, "POST": true, "HEAD": true}
	for method := range metrics.MethodDistribution {
		if !allowedMethods[method] {
			score += 0.1
			result.Reasons = append(result.Reasons, "Unusual HTTP method: "+method)
		}
	}

	return math.Min(score, 1.0)
}

func (s *CrawlerEnhancedDetectionService) detectAutomation(userAgent string, r *http.Request, result *CrawlerDetectionResult) float64 {
	score := 0.0

	userAgentLower := strings.ToLower(userAgent)

	for _, pattern := range s.headlessPatterns {
		if pattern.MatchString(userAgentLower) {
			score += 0.4
			result.Signatures = append(result.Signatures, "Headless browser detected")
			result.Reasons = append(result.Reasons, "Headless browser automation detected")
			break
		}
	}

	for _, pattern := range s.cryptoPatterns {
		if pattern.MatchString(userAgentLower) {
			score += 0.5
			result.Signatures = append(result.Signatures, "Cryptocurrency mining detected")
			result.Reasons = append(result.Reasons, "Potential cryptojacking activity")
		}
	}

	headers := r.Header

	if headers.Get("Webdriver") != "" {
		score += 0.5
		result.Signatures = append(result.Signatures, "Webdriver header present")
		result.Reasons = append(result.Reasons, "Webdriver automation detected")
	}

	if headers.Get("Driver") != "" {
		score += 0.4
		result.Signatures = append(result.Signatures, "Driver header present")
	}

	if headers.Get("秒表") != "" || headers.Get("seconds") != "" {
		score += 0.3
		result.Reasons = append(result.Reasons, "Automation timing indicators")
	}

	if strings.Contains(userAgentLower, "navigator.webdriver") {
		score += 0.4
		result.Signatures = append(result.Signatures, "Navigator webdriver property")
	}

	if r.URL.Query().Get("_phantom") != "" || r.URL.Query().Get("__webdriver_evaluate") != "" {
		score += 0.5
		result.Reasons = append(result.Reasons, "PhantomJS/Selenium query parameters")
	}

	return math.Min(score, 1.0)
}

func (s *CrawlerEnhancedDetectionService) analyzeRequestPattern(ip string, r *http.Request, result *CrawlerDetectionResult) float64 {
	score := 0.0

	history, exists := s.requestHistory[ip]
	if !exists || len(history) < 10 {
		return 0.0
	}

	pathCounts := make(map[string]int)
	for _, record := range history {
		pathCounts[record.Path]++
	}

	totalRequests := len(history)
	for path, count := range pathCounts {
		frequency := float64(count) / float64(totalRequests)
		if frequency > 0.8 && totalRequests > 30 {
			score += 0.3
			result.Reasons = append(result.Reasons, "High frequency access to path: "+path)
			break
		}
	}

	paths := make([]string, len(history))
	for i, record := range history {
		paths[i] = record.Path
	}

	if s.isSequentialPattern(paths) {
		score += 0.2
		result.Reasons = append(result.Reasons, "Sequential path access detected")
	}

	methodCounts := make(map[string]int)
	for _, record := range history {
		methodCounts[record.Method]++
	}

	if methodCounts["GET"] > 0 && methodCounts["POST"] == 0 && methodCounts["PUT"] == 0 {
		if totalRequests > 100 {
			score += 0.15
			result.Reasons = append(result.Reasons, "Only GET requests - possible crawler")
		}
	}

	if strings.HasPrefix(r.URL.Path, "/api/") && r.Method == "GET" {
		score += 0.1
		result.Reasons = append(result.Reasons, "High-frequency API access")
	}

	return math.Min(score, 1.0)
}

func (s *CrawlerEnhancedDetectionService) calculateBehaviorMetrics(history []*RequestRecord) *BehaviorMetrics {
	metrics := &BehaviorMetrics{
		PathDistribution:   make(map[string]int),
		MethodDistribution: make(map[string]int),
	}

	if len(history) < 2 {
		return metrics
	}

	metrics.RequestCount = len(history)

	intervals := make([]float64, 0)
	paths := make([]string, 0)
	uniquePaths := make(map[string]bool)

	for i, record := range history {
		metrics.PathDistribution[record.Path]++
		metrics.MethodDistribution[record.Method]++
		paths = append(paths, record.Path)
		uniquePaths[record.Path] = true

		if i > 0 {
			interval := record.Timestamp.Sub(history[i-1].Timestamp).Milliseconds()
			intervals = append(intervals, float64(interval))
		}
	}

	metrics.UniquePaths = len(uniquePaths)

	if len(intervals) > 0 {
		sum := 0.0
		for _, interval := range intervals {
			sum += interval
		}
		metrics.RequestInterval = sum / float64(len(intervals))

		mean := metrics.RequestInterval
		varianceSum := 0.0
		for _, interval := range intervals {
			varianceSum += math.Pow(interval-mean, 2)
		}
		metrics.RequestVariance = math.Sqrt(varianceSum / float64(len(intervals)))
	}

	metrics.IsRegularPattern = metrics.RequestVariance < metrics.RequestInterval*0.2
	metrics.IsSequentialPath = s.isSequentialPattern(paths)

	if len(history) > 0 {
		metrics.SessionDuration = history[len(history)-1].Timestamp.Sub(history[0].Timestamp)
	}

	return metrics
}

func (s *CrawlerEnhancedDetectionService) isSequentialPattern(paths []string) bool {
	if len(paths) < 5 {
		return false
	}

	sequentialCount := 0
	for i := 1; i < len(paths); i++ {
		if paths[i] > paths[i-1] {
			sequentialCount++
		}
	}

	ratio := float64(sequentialCount) / float64(len(paths)-1)
	return ratio > 0.8
}

func (s *CrawlerEnhancedDetectionService) recordRequest(ip string, r *http.Request, fingerprint string) {
	record := &RequestRecord{
		Timestamp:   time.Now(),
		Path:        r.URL.Path,
		Method:      r.Method,
		UserAgent:   r.UserAgent(),
		Fingerprint: fingerprint,
	}

	history, exists := s.requestHistory[ip]
	if !exists {
		history = make([]*RequestRecord, 0, s.maxHistorySize)
	}

	history = append(history, record)

	if len(history) > s.maxHistorySize {
		history = history[len(history)-s.maxHistorySize:]
	}

	s.requestHistory[ip] = history
}

func (s *CrawlerEnhancedDetectionService) generateFingerprint(ip string, userAgent string, additionalData map[string]string) string {
	hasher := sha256.New()
	hasher.Write([]byte(ip))
	hasher.Write([]byte(userAgent))
	for key, value := range additionalData {
		hasher.Write([]byte(key + ":" + value))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *CrawlerEnhancedDetectionService) determineCrawlerType(result *CrawlerDetectionResult) CrawlerType {
	typeScores := make(map[CrawlerType]float64)

	for _, reason := range result.Reasons {
		reasonLower := strings.ToLower(reason)
		if strings.Contains(reasonLower, "search") || strings.Contains(reasonLower, "google") || strings.Contains(reasonLower, "bing") {
			typeScores[CrawlerTypeSearchBot] += 0.2
		}
		if strings.Contains(reasonLower, "social") || strings.Contains(reasonLower, "facebook") || strings.Contains(reasonLower, "twitter") {
			typeScores[CrawlerTypeSocialBot] += 0.2
		}
		if strings.Contains(reasonLower, "headless") || strings.Contains(reasonLower, "selenium") || strings.Contains(reasonLower, "puppeteer") {
			typeScores[CrawlerTypeHeadless] += 0.3
		}
		if strings.Contains(reasonLower, "scraping") || strings.Contains(reasonLower, "scrapy") {
			typeScores[CrawlerTypeMalicious] += 0.3
		}
		if strings.Contains(reasonLower, "automation") {
			typeScores[CrawlerTypeAutomated] += 0.2
		}
		if strings.Contains(reasonLower, "crypto") || strings.Contains(reasonLower, "mining") {
			typeScores[CrawlerTypeMalicious] += 0.4
		}
	}

	maxScore := 0.0
	var maxType CrawlerType = CrawlerTypeUnknown

	for crawlerType, score := range typeScores {
		if score > maxScore {
			maxScore = score
			maxType = crawlerType
		}
	}

	return maxType
}

func (s *CrawlerEnhancedDetectionService) GetKnownBotSignature(botName string) *CrawlerSignature {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if signature, exists := s.knownBots[botName]; exists {
		return signature
	}
	return nil
}

func (s *CrawlerEnhancedDetectionService) AddKnownBotSignature(botName string, signature *CrawlerSignature) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.knownBots[botName] = signature
}

func (s *CrawlerEnhancedDetectionService) GetRequestHistory(ip string) []*RequestRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if history, exists := s.requestHistory[ip]; exists {
		result := make([]*RequestRecord, len(history))
		copy(result, history)
		return result
	}
	return nil
}

func (s *CrawlerEnhancedDetectionService) ClearRequestHistory(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.requestHistory, ip)
	delete(s.behaviorCache, ip)
}

func (s *CrawlerEnhancedDetectionService) GetCrawlerStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalHistorySize := 0
	for _, history := range s.requestHistory {
		totalHistorySize += len(history)
	}

	return map[string]interface{}{
		"tracked_ips":       len(s.requestHistory),
		"total_records":     totalHistorySize,
		"known_bot_types":   len(s.knownBots),
		"behavior_cache_size": len(s.behaviorCache),
	}
}
