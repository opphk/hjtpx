package service

import (
	"math"
	"regexp"
	"sync"
	"time"
)

type TrafficPattern struct {
	RequestTimes []time.Time
	RequestSizes []int
	Methods      []string
	Paths        []string
	UserAgents   []string
}

type AnomalyDetectionResult struct {
	IsAnomaly   bool
	AnomalyType string
	Score       float64
	Details     map[string]interface{}
}

type AnomalyDetectionService struct {
	patterns map[string]*TrafficPattern
	mu       sync.RWMutex
}

func NewAnomalyDetectionService() *AnomalyDetectionService {
	return &AnomalyDetectionService{
		patterns: make(map[string]*TrafficPattern),
	}
}

func (s *AnomalyDetectionService) RecordTraffic(clientID string, size int, method, path, userAgent string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pattern, exists := s.patterns[clientID]
	if !exists {
		pattern = &TrafficPattern{
			RequestTimes: make([]time.Time, 0),
			RequestSizes: make([]int, 0),
			Methods:      make([]string, 0),
			Paths:        make([]string, 0),
			UserAgents:   make([]string, 0),
		}
		s.patterns[clientID] = pattern
	}

	now := time.Now()
	pattern.RequestTimes = append(pattern.RequestTimes, now)
	pattern.RequestSizes = append(pattern.RequestSizes, size)
	pattern.Methods = append(pattern.Methods, method)
	pattern.Paths = append(pattern.Paths, path)
	pattern.UserAgents = append(pattern.UserAgents, userAgent)

	s.cleanOldData(pattern)
}

func (s *AnomalyDetectionService) cleanOldData(pattern *TrafficPattern) {
	cutoff := time.Now().Add(-1 * time.Hour)
	filteredTimes := make([]time.Time, 0)
	filteredSizes := make([]int, 0)
	filteredMethods := make([]string, 0)
	filteredPaths := make([]string, 0)
	filteredUserAgents := make([]string, 0)

	for i, t := range pattern.RequestTimes {
		if t.After(cutoff) {
			filteredTimes = append(filteredTimes, t)
			filteredSizes = append(filteredSizes, pattern.RequestSizes[i])
			filteredMethods = append(filteredMethods, pattern.Methods[i])
			filteredPaths = append(filteredPaths, pattern.Paths[i])
			filteredUserAgents = append(filteredUserAgents, pattern.UserAgents[i])
		}
	}

	pattern.RequestTimes = filteredTimes
	pattern.RequestSizes = filteredSizes
	pattern.Methods = filteredMethods
	pattern.Paths = filteredPaths
	pattern.UserAgents = filteredUserAgents
}

func (s *AnomalyDetectionService) DetectAnomaly(clientID string) *AnomalyDetectionResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pattern, exists := s.patterns[clientID]
	if !exists || len(pattern.RequestTimes) < 10 {
		return &AnomalyDetectionResult{IsAnomaly: false, Score: 0}
	}

	result := &AnomalyDetectionResult{
		Details: make(map[string]interface{}),
	}

	requestFreqScore := s.analyzeRequestFrequency(pattern)
	sizeAnomalyScore := s.analyzeRequestSizes(pattern)
	pathRepetitionScore := s.analyzePathRepetition(pattern)
	userAgentScore := s.analyzeUserAgents(pattern)

	totalScore := (requestFreqScore + sizeAnomalyScore + pathRepetitionScore + userAgentScore) / 4
	result.Score = totalScore
	result.IsAnomaly = totalScore > 0.7

	if requestFreqScore > 0.8 {
		result.AnomalyType = "high_frequency"
		result.Details["frequency_score"] = requestFreqScore
	} else if sizeAnomalyScore > 0.8 {
		result.AnomalyType = "unusual_size"
		result.Details["size_score"] = sizeAnomalyScore
	} else if pathRepetitionScore > 0.8 {
		result.AnomalyType = "path_repetition"
		result.Details["path_score"] = pathRepetitionScore
	}

	return result
}

func (s *AnomalyDetectionService) analyzeRequestFrequency(pattern *TrafficPattern) float64 {
	if len(pattern.RequestTimes) < 2 {
		return 0
	}

	intervals := make([]float64, 0)
	for i := 1; i < len(pattern.RequestTimes); i++ {
		interval := pattern.RequestTimes[i].Sub(pattern.RequestTimes[i-1]).Milliseconds()
		intervals = append(intervals, float64(interval))
	}

	avgInterval := average(intervals)
	if avgInterval < 100 {
		return 1.0
	} else if avgInterval < 500 {
		return 0.8
	} else if avgInterval < 1000 {
		return 0.5
	}
	return 0
}

func (s *AnomalyDetectionService) analyzeRequestSizes(pattern *TrafficPattern) float64 {
	if len(pattern.RequestSizes) < 5 {
		return 0
	}

	avgSize := averageInt(pattern.RequestSizes)
	stdDev := stdDevInt(pattern.RequestSizes)
	variation := stdDev / math.Max(1, avgSize)

	if variation > 2 {
		return 0.9
	} else if variation > 1 {
		return 0.6
	}
	return 0
}

func (s *AnomalyDetectionService) analyzePathRepetition(pattern *TrafficPattern) float64 {
	if len(pattern.Paths) < 5 {
		return 0
	}

	pathCounts := make(map[string]int)
	for _, path := range pattern.Paths {
		pathCounts[path]++
	}

	maxCount := 0
	for _, count := range pathCounts {
		if count > maxCount {
			maxCount = count
		}
	}

	ratio := float64(maxCount) / float64(len(pattern.Paths))
	if ratio > 0.9 {
		return 0.95
	} else if ratio > 0.7 {
		return 0.7
	} else if ratio > 0.5 {
		return 0.4
	}
	return 0
}

func (s *AnomalyDetectionService) analyzeUserAgents(pattern *TrafficPattern) float64 {
	if len(pattern.UserAgents) < 5 {
		return 0
	}

	uniqueUserAgents := make(map[string]bool)
	for _, ua := range pattern.UserAgents {
		uniqueUserAgents[ua] = true
	}

	if len(uniqueUserAgents) > 5 {
		return 0.8
	} else if len(uniqueUserAgents) > 3 {
		return 0.4
	}
	return 0
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func averageInt(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

func stdDevInt(values []int) float64 {
	if len(values) < 2 {
		return 0
	}
	avg := averageInt(values)
	variance := 0.0
	for _, v := range values {
		diff := float64(v) - avg
		variance += diff * diff
	}
	variance /= float64(len(values))
	return math.Sqrt(variance)
}

type InputValidationResult struct {
	IsValid bool
	Errors  []string
}

type InputValidator struct {
	sqlPatterns     []*regexp.Regexp
	xssPatterns     []*regexp.Regexp
	cmdInjPatterns  []*regexp.Regexp
}

func NewInputValidator() *InputValidator {
	return &InputValidator{
		sqlPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|alter|exec|execute|script|javascript)`),
			regexp.MustCompile(`(?i)(['";\\\/\-\(\)])`),
		},
		xssPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(<script|javascript:|on\w+\s*=|data:text/html)`),
			regexp.MustCompile(`(?i)(<iframe|<img|<svg|<link|<meta)`),
		},
		cmdInjPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(\|\||&&|;|` + "`" + `|\$\(|\$\{)`),
			regexp.MustCompile(`(?i)(system|exec|shell|passthru|popen)`),
		},
	}
}

func (v *InputValidator) ValidateInput(input string) *InputValidationResult {
	result := &InputValidationResult{IsValid: true, Errors: make([]string, 0)}

	for _, pattern := range v.sqlPatterns {
		if pattern.MatchString(input) {
			result.IsValid = false
			result.Errors = append(result.Errors, "Potential SQL injection detected")
			break
		}
	}

	for _, pattern := range v.xssPatterns {
		if pattern.MatchString(input) {
			result.IsValid = false
			result.Errors = append(result.Errors, "Potential XSS attack detected")
			break
		}
	}

	for _, pattern := range v.cmdInjPatterns {
		if pattern.MatchString(input) {
			result.IsValid = false
			result.Errors = append(result.Errors, "Potential command injection detected")
			break
		}
	}

	return result
}

func (v *InputValidator) SanitizeInput(input string) string {
	sanitized := input
	sanitized = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(sanitized, "")
	sanitized = regexp.MustCompile(`['";\\\/]`).ReplaceAllString(sanitized, "")
	return sanitized
}

type SecurityHeadersConfig struct {
	CSP               string
	HSTS              string
	XFrameOptions     string
	XContentTypeOptions string
	XXSSProtection    string
	ReferrerPolicy    string
}

var DefaultSecurityHeaders = SecurityHeadersConfig{
	CSP: "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
	HSTS: "max-age=31536000; includeSubDomains; preload",
	XFrameOptions: "DENY",
	XContentTypeOptions: "nosniff",
	XXSSProtection: "1; mode=block",
	ReferrerPolicy: "strict-origin-when-cross-origin",
}
