package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"sync"
	"sync/atomic"
	"time"
)

type DynamicDefenseService struct {
	defensePolicies     map[string]*DefensePolicy
	activeMitigations   map[string]*ActiveMitigation
	rateLimiters        map[string]*AdaptiveRateLimiter
	accessPatterns      map[string]*AccessPattern
	trafficProfiles     map[string]*TrafficProfile
	anomalyDetectors    map[string]*DefenseAnomalyDetector
	wafRules           []*WAFRule
	ipRanges           []*IPRangeConfig
	geoBlocking         *GeoBlockingConfig
	adaptiveThresholds  *AdaptiveThresholds
	defenseState        *DefenseState
	mu                  sync.RWMutex
	lastUpdate          time.Time
	updateInterval      time.Duration
	enabled             bool
}

type DefensePolicy struct {
	ID           string
	Name         string
	Priority     int
	Condition    *PolicyCondition
	Action       DefenseAction
	Duration     time.Duration
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	HitCount     int
}

type PolicyCondition struct {
	ThreatLevel   ThreatLevel
	IPReputation  float64
	UserAgent     string
	RequestPath   string
	CountryCode   string
	ASNNumber     int
	HourOfDay     []int
	DayOfWeek     []int
	RequestCount  int
	TimeWindow    time.Duration
}

type ThreatLevel int

const (
	ThreatLevelNone ThreatLevel = iota
	ThreatLevelLow
	ThreatLevelMedium
	ThreatLevelHigh
	ThreatLevelCritical
)

type DefenseAction string

const (
	ActionBlock          DefenseAction = "block"
	ActionChallenge      DefenseAction = "challenge"
	ActionRateLimit      DefenseAction = "rate_limit"
	ActionCaptcha         DefenseAction = "captcha"
	ActionRedirect       DefenseAction = "redirect"
	ActionMonitor        DefenseAction = "monitor"
	ActionThrottle       DefenseAction = "throttle"
	ActionAllow          DefenseAction = "allow"
	ActionCustom         DefenseAction = "custom"
)

type ActiveMitigation struct {
	ID           string
	PolicyID     string
	IP           string
	Action       DefenseAction
	StartTime    time.Time
	EndTime      time.Time
	Reason       string
	RequestCount int
	BlockedCount int
}

type AdaptiveRateLimiter struct {
	IP             string
	requests       []time.Time
	tokens         float64
	maxTokens      float64
	refillRate     float64
	lastRefill     time.Time
	blocked        bool
	blockUntil     time.Time
	mu             sync.Mutex
}

type AccessPattern struct {
	IP           string
	UserAgent    string
	FirstSeen    time.Time
	LastSeen     time.Time
	RequestCount int
	Paths        []string
	Methods      []string
	AvgInterval  float64
	UniqueDays   int
	IsSuspicious bool
	ThreatScore  float64
}

type TrafficProfile struct {
	IP             string
	BaselineRequests float64
	BaselineBytes   int64
	CurrentRequests float64
	CurrentBytes    int64
	PeakRequests    float64
	PeakBytes       int64
	AvgResponseTime float64
	IsAnomalous    bool
	LastUpdated    time.Time
}

type DefenseAnomalyDetector struct {
	IP             string
	DataPoints     []AnomalyDataPoint
	Threshold      float64
	IsAnomalous    bool
	AnomalyType    string
	Confidence     float64
	LastAnomaly    time.Time
}

type AnomalyDataPoint struct {
	Timestamp   time.Time
	RequestRate float64
	ErrorRate   float64
	Latency     float64
	ByteSize    int64
}

type WAFRule struct {
	ID           string
	Name         string
	Pattern      *regexp.Regexp
	Action       DefenseAction
	Severity     int
	IsActive     bool
	Description  string
	MatchCount   int
	CreatedAt    time.Time
}

type IPRangeConfig struct {
	CIDR         string
	Action       DefenseAction
	Priority     int
	Description  string
	IsWhitelist  bool
	IsBlacklist  bool
}

type GeoBlockingConfig struct {
	Enabled       bool
	BlockedCountries map[string]bool
	AllowedCountries map[string]bool
	BlockByDefault bool
}

type AdaptiveThresholds struct {
	RequestPerMinute int
	ErrorRate        float64
	LatencyMs        int
	BytesPerSecond   int64
	ConcurrentConns  int
}

type DefenseState struct {
	CurrentLevel      ThreatLevel
	ActiveBlocks      int32
	ActiveChallenges  int32
	TotalMitigations  int32
	AvgResponseTime   float64
	LastThreatLevelChange time.Time
	ThreatTrend       []ThreatLevel
}

type DefenseResult struct {
	ShouldBlock    bool
	ShouldChallenge bool
	Action         DefenseAction
	RiskScore      float64
	ThreatLevel    ThreatLevel
	Recommendations []string
	MitigationID   string
	BlockDuration  time.Duration
}

type DynamicDefenseConfig struct {
	Enabled             bool
	UpdateInterval      time.Duration
	AdaptiveEnabled     bool
	GeoBlockingEnabled  bool
	WAFEnabled          bool
	RateLimitEnabled    bool
	AnomalyDetectionEnabled bool
}

func NewDynamicDefenseService() *DynamicDefenseService {
	service := &DynamicDefenseService{
		defensePolicies:    make(map[string]*DefensePolicy),
		activeMitigations:  make(map[string]*ActiveMitigation),
		rateLimiters:       make(map[string]*AdaptiveRateLimiter),
		accessPatterns:     make(map[string]*AccessPattern),
		trafficProfiles:   make(map[string]*TrafficProfile),
		anomalyDetectors:  make(map[string]*DefenseAnomalyDetector),
		updateInterval:    30 * time.Second,
		enabled:           true,
	}

	service.defenseState = &DefenseState{
		CurrentLevel:     ThreatLevelLow,
		ThreatTrend:      make([]ThreatLevel, 0),
	}
	service.geoBlocking = &GeoBlockingConfig{
		Enabled:        true,
		BlockedCountries: make(map[string]bool),
		AllowedCountries: make(map[string]bool),
		BlockByDefault: false,
	}
	service.adaptiveThresholds = &AdaptiveThresholds{
		RequestPerMinute: 100,
		ErrorRate:        0.1,
		LatencyMs:        1000,
		BytesPerSecond:   1024 * 1024,
		ConcurrentConns:  100,
	}

	service.initializeDefaultPolicies()
	service.initializeWAFRules()
	service.initializeIPRanges()
	return service
}

func (s *DynamicDefenseService) initializeDefaultPolicies() {
	s.defensePolicies["critical_block"] = &DefensePolicy{
		ID:        "critical_block",
		Name:      "Critical Threat Blocking",
		Priority:  100,
		Condition: &PolicyCondition{ThreatLevel: ThreatLevelCritical},
		Action:    ActionBlock,
		Duration:  24 * time.Hour,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	s.defensePolicies["high_challenge"] = &DefensePolicy{
		ID:        "high_challenge",
		Name:      "High Threat Challenge",
		Priority:  80,
		Condition: &PolicyCondition{ThreatLevel: ThreatLevelHigh},
		Action:    ActionCaptcha,
		Duration:  1 * time.Hour,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	s.defensePolicies["medium_rate_limit"] = &DefensePolicy{
		ID:        "medium_rate_limit",
		Name:      "Medium Threat Rate Limiting",
		Priority:  60,
		Condition: &PolicyCondition{ThreatLevel: ThreatLevelMedium},
		Action:    ActionRateLimit,
		Duration:  30 * time.Minute,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	s.defensePolicies["low_monitor"] = &DefensePolicy{
		ID:        "low_monitor",
		Name:      "Low Threat Monitoring",
		Priority:  40,
		Condition: &PolicyCondition{ThreatLevel: ThreatLevelLow},
		Action:    ActionMonitor,
		Duration:  15 * time.Minute,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
}

func (s *DynamicDefenseService) initializeWAFRules() {
	s.wafRules = append(s.wafRules, &WAFRule{
		ID:          "waf_sqli",
		Name:        "SQL Injection Protection",
		Pattern:     regexp.MustCompile(`(?i)(union\s+select|or\s+1\s*=\s*1|drop\s+table|insert\s+into|exec\s*\()`),
		Action:      ActionBlock,
		Severity:    5,
		IsActive:    true,
		Description: "Blocks SQL injection attempts",
		CreatedAt:   time.Now(),
	})
	s.wafRules = append(s.wafRules, &WAFRule{
		ID:          "waf_xss",
		Name:        "XSS Protection",
		Pattern:     regexp.MustCompile(`(?i)(<script|javascript:|onerror=|onload=|alert\s*\()`),
		Action:      ActionBlock,
		Severity:    4,
		IsActive:    true,
		Description: "Blocks cross-site scripting attempts",
		CreatedAt:   time.Now(),
	})
	s.wafRules = append(s.wafRules, &WAFRule{
		ID:          "waf_path",
		Name:        "Path Traversal Protection",
		Pattern:     regexp.MustCompile(`(?i)(\.\.\/|\.\.\\|%2e%2e%2f)`),
		Action:      ActionBlock,
		Severity:    4,
		IsActive:    true,
		Description: "Blocks path traversal attempts",
		CreatedAt:   time.Now(),
	})
}

func (s *DynamicDefenseService) initializeIPRanges() {
	s.ipRanges = append(s.ipRanges, &IPRangeConfig{
		CIDR:        "10.0.0.0/8",
		Action:      ActionAllow,
		Priority:    100,
		IsWhitelist: true,
		Description: "Private network",
	})
	s.ipRanges = append(s.ipRanges, &IPRangeConfig{
		CIDR:        "192.168.0.0/16",
		Action:      ActionAllow,
		Priority:    100,
		IsWhitelist: true,
		Description: "Private network",
	})
	s.ipRanges = append(s.ipRanges, &IPRangeConfig{
		CIDR:        "172.16.0.0/12",
		Action:      ActionAllow,
		Priority:    100,
		IsWhitelist: true,
		Description: "Private network",
	})
}

func (s *DynamicDefenseService) EvaluateRequest(ctx context.Context, r *http.Request) (*DefenseResult, error) {
	ip := getClientIP(r)
	userAgent := r.UserAgent()

	result := &DefenseResult{
		RiskScore:      0.0,
		ThreatLevel:    ThreatLevelNone,
		Recommendations: []string{},
	}

	if s.checkIPRangeWhitelist(ip) {
		return &DefenseResult{
			ShouldBlock:     false,
			ShouldChallenge: false,
			Action:         ActionAllow,
			RiskScore:      0.0,
			ThreatLevel:    ThreatLevelNone,
		}, nil
	}

	if s.checkIPRangeBlacklist(ip) {
		return &DefenseResult{
			ShouldBlock:     true,
			ShouldChallenge: false,
			Action:         ActionBlock,
			RiskScore:      100.0,
			ThreatLevel:    ThreatLevelCritical,
			BlockDuration: 24 * time.Hour,
		}, nil
	}

	if s.geoBlocking.Enabled {
		if s.isGeoBlocked(r) {
			result.RiskScore += 30
			result.ThreatLevel = ThreatLevelMedium
			result.Recommendations = append(result.Recommendations, "Geo-blocking applied")
		}
	}

	wafResult := s.evaluateWAFRules(r)
	if wafResult.Matched {
		result.RiskScore += float64(wafResult.Severity * 10)
		result.ThreatLevel = s.calculateThreatLevel(result.RiskScore)
		result.Recommendations = append(result.Recommendations, fmt.Sprintf("WAF rule matched: %s", wafResult.RuleName))
	}

	rateLimitResult := s.evaluateRateLimit(ip)
	if rateLimitResult.ShouldLimit {
		result.RiskScore += rateLimitResult.AdditionalScore
		result.ThreatLevel = s.calculateThreatLevel(result.RiskScore)
		result.Recommendations = append(result.Recommendations, "Rate limit exceeded")
	}

	patternResult := s.evaluateAccessPattern(ip, userAgent, r.URL.Path)
	result.RiskScore += patternResult.Score
	result.ThreatLevel = s.calculateThreatLevel(result.RiskScore)

	if patternResult.IsAnomalous {
		result.Recommendations = append(result.Recommendations, "Anomalous access pattern detected")
	}

	anomalyResult := s.detectAnomalies(ip)
	if anomalyResult.IsAnomalous {
		result.RiskScore += anomalyResult.Score
		result.ThreatLevel = s.calculateThreatLevel(result.RiskScore)
		result.Recommendations = append(result.Recommendations, fmt.Sprintf("Anomaly detected: %s", anomalyResult.Type))
	}

	result.ShouldBlock = result.RiskScore >= 80 || result.ThreatLevel >= ThreatLevelCritical
	result.ShouldChallenge = result.RiskScore >= 40 && result.RiskScore < 80
	result.Action = s.determineAction(result.ThreatLevel, result.RiskScore)

	if result.ShouldBlock {
		result.BlockDuration = s.calculateBlockDuration(result.ThreatLevel)
	}

	s.updateDefenseState(result)
	s.recordAccess(ip, userAgent, r.URL.Path)

	return result, nil
}

func (s *DynamicDefenseService) checkIPRangeWhitelist(ip string) bool {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}

	for _, config := range s.ipRanges {
		if !config.IsWhitelist {
			continue
		}
		_, cidr, err := net.ParseCIDR(config.CIDR)
		if err != nil {
			continue
		}
		if cidr.Contains(ipAddr) {
			return true
		}
	}
	return false
}

func (s *DynamicDefenseService) checkIPRangeBlacklist(ip string) bool {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}

	for _, config := range s.ipRanges {
		if !config.IsBlacklist {
			continue
		}
		_, cidr, err := net.ParseCIDR(config.CIDR)
		if err != nil {
			continue
		}
		if cidr.Contains(ipAddr) {
			return true
		}
	}
	return false
}

func (s *DynamicDefenseService) isGeoBlocked(r *http.Request) bool {
	if !s.geoBlocking.BlockByDefault {
		return s.geoBlocking.BlockedCountries["XX"]
	}
	return !s.geoBlocking.AllowedCountries["US"]
}

type WAFResult struct {
	Matched   bool
	RuleName  string
	Severity  int
	RuleID    string
}

func (s *DynamicDefenseService) evaluateWAFRules(r *http.Request) *WAFResult {
	result := &WAFResult{Matched: false}

	requestURI := r.RequestURI
	queryString := r.URL.RawQuery

	for _, rule := range s.wafRules {
		if !rule.IsActive {
			continue
		}

		if rule.Pattern.MatchString(requestURI) || rule.Pattern.MatchString(queryString) {
			result.Matched = true
			result.RuleName = rule.Name
			result.Severity = rule.Severity
			result.RuleID = rule.ID
			rule.MatchCount++
			return result
		}
	}

	return result
}

type DefenseRateLimitResult struct {
	ShouldLimit    bool
	CurrentRate    float64
	MaxRate        float64
	AdditionalScore float64
}

func (s *DynamicDefenseService) evaluateRateLimit(ip string) *RateLimitResult {
	limiter := s.getOrCreateRateLimiter(ip)

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	if limiter.blocked && time.Now().Before(limiter.blockUntil) {
		return &DefenseRateLimitResult{
			ShouldLimit:    true,
			CurrentRate:    float64(len(limiter.requests)),
			MaxRate:        limiter.maxTokens,
			AdditionalScore: 50,
		}
	}

	now := time.Now()
	limiter.requests = append(limiter.requests, now)

	windowStart := now.Add(-1 * time.Minute)
	var recentRequests int
	var validRequests []time.Time
	for _, t := range limiter.requests {
		if t.After(windowStart) {
			recentRequests++
			validRequests = append(validRequests, t)
		}
	}
	limiter.requests = validRequests

	if recentRequests > int(limiter.maxTokens) {
		limiter.blocked = true
		limiter.blockUntil = now.Add(5 * time.Minute)
		return &DefenseRateLimitResult{
			ShouldLimit:    true,
			CurrentRate:    float64(recentRequests),
			MaxRate:        limiter.maxTokens,
			AdditionalScore: 40,
		}
	}

	return &DefenseRateLimitResult{
		ShouldLimit:    false,
		CurrentRate:    float64(recentRequests),
		MaxRate:        limiter.maxTokens,
		AdditionalScore: 0,
	}
}

func (s *DynamicDefenseService) getOrCreateRateLimiter(ip string) *AdaptiveRateLimiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limiter, exists := s.rateLimiters[ip]; exists {
		return limiter
	}

	limiter := &AdaptiveRateLimiter{
		IP:         ip,
		requests:   make([]time.Time, 0),
		tokens:     100,
		maxTokens:  100,
		refillRate: 10,
		lastRefill: time.Now(),
	}
	s.rateLimiters[ip] = limiter
	return limiter
}

type PatternResult struct {
	Score      float64
	IsAnomalous bool
	PatternType string
}

func (s *DynamicDefenseService) evaluateAccessPattern(ip, userAgent, path string) *PatternResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	pattern, exists := s.accessPatterns[ip]
	if !exists {
		pattern = &AccessPattern{
			IP:        ip,
			UserAgent: userAgent,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
			Paths:     make([]string, 0),
			Methods:   make([]string, 0),
		}
		s.accessPatterns[ip] = pattern
	}

	pattern.LastSeen = time.Now()
	pattern.RequestCount++
	pattern.Paths = append(pattern.Paths, path)

	if len(pattern.Paths) > 100 {
		pattern.Paths = pattern.Paths[len(pattern.Paths)-100:]
	}

	result := &PatternResult{Score: 0}

	if pattern.RequestCount > 1000 {
		result.Score += 20
		result.IsAnomalous = true
		result.PatternType = "high_volume"
	}

	uniquePaths := make(map[string]bool)
	for _, p := range pattern.Paths {
		uniquePaths[p] = true
	}
	if len(uniquePaths) == 1 && pattern.RequestCount > 50 {
		result.Score += 25
		result.IsAnomalous = true
		result.PatternType = "single_path_focus"
	}

	return result
}

type DefenseAnomalyResult struct {
	IsAnomalous bool
	Score       float64
	Type        string
	Confidence  float64
}

func (s *DynamicDefenseService) detectAnomalies(ip string) *DefenseAnomalyResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	detector, exists := s.anomalyDetectors[ip]
	if !exists {
		detector = &DefenseAnomalyDetector{
			IP:         ip,
			DataPoints: make([]AnomalyDataPoint, 0),
			Threshold:  0.8,
		}
		s.anomalyDetectors[ip] = detector
	}

	dataPoint := AnomalyDataPoint{
		Timestamp:   time.Now(),
		RequestRate: float64(len(detector.DataPoints)),
		ByteSize:    int64(len(detector.DataPoints) * 1000),
	}
	detector.DataPoints = append(detector.DataPoints, dataPoint)

	if len(detector.DataPoints) > 100 {
		detector.DataPoints = detector.DataPoints[len(detector.DataPoints)-100:]
	}

	result := &DefenseAnomalyResult{IsAnomalous: false}

	if len(detector.DataPoints) >= 10 {
		avgRate := s.calculateAverageRequestRate(detector.DataPoints)
		currentRate := detector.DataPoints[len(detector.DataPoints)-1].RequestRate

		if currentRate > avgRate*3 {
			result.IsAnomalous = true
			result.Score = 40
			result.Type = "request_rate_spike"
			result.Confidence = 0.85
			detector.IsAnomalous = true
			detector.AnomalyType = result.Type
		}
	}

	return result
}

func (s *DynamicDefenseService) calculateAverageRequestRate(points []AnomalyDataPoint) float64 {
	if len(points) == 0 {
		return 0
	}
	var sum float64
	for _, p := range points {
		sum += p.RequestRate
	}
	return sum / float64(len(points))
}

func (s *DynamicDefenseService) calculateThreatLevel(score float64) ThreatLevel {
	switch {
	case score >= 80:
		return ThreatLevelCritical
	case score >= 60:
		return ThreatLevelHigh
	case score >= 40:
		return ThreatLevelMedium
	case score >= 20:
		return ThreatLevelLow
	default:
		return ThreatLevelNone
	}
}

func (s *DynamicDefenseService) determineAction(level ThreatLevel, score float64) DefenseAction {
	switch level {
	case ThreatLevelCritical:
		return ActionBlock
	case ThreatLevelHigh:
		return ActionCaptcha
	case ThreatLevelMedium:
		return ActionRateLimit
	case ThreatLevelLow:
		return ActionMonitor
	default:
		return ActionAllow
	}
}

func (s *DynamicDefenseService) calculateBlockDuration(level ThreatLevel) time.Duration {
	switch level {
	case ThreatLevelCritical:
		return 24 * time.Hour
	case ThreatLevelHigh:
		return 1 * time.Hour
	case ThreatLevelMedium:
		return 30 * time.Minute
	case ThreatLevelLow:
		return 15 * time.Minute
	default:
		return 5 * time.Minute
	}
}

func (s *DynamicDefenseService) updateDefenseState(result *DefenseResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if result.ShouldBlock {
		atomic.AddInt32(&s.defenseState.ActiveBlocks, 1)
	}
	if result.ShouldChallenge {
		atomic.AddInt32(&s.defenseState.ActiveChallenges, 1)
	}
	atomic.AddInt32(&s.defenseState.TotalMitigations, 1)

	s.defenseState.ThreatTrend = append(s.defenseState.ThreatTrend, result.ThreatLevel)
	if len(s.defenseState.ThreatTrend) > 100 {
		s.defenseState.ThreatTrend = s.defenseState.ThreatTrend[len(s.defenseState.ThreatTrend)-100:]
	}

	newLevel := s.calculateOverallThreatLevel()
	if newLevel != s.defenseState.CurrentLevel {
		s.defenseState.LastThreatLevelChange = time.Now()
		s.defenseState.CurrentLevel = newLevel
	}
}

func (s *DynamicDefenseService) calculateOverallThreatLevel() ThreatLevel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalScore float64
	count := 0

	for _, pattern := range s.accessPatterns {
		totalScore += pattern.ThreatScore
		count++
	}

	if count == 0 {
		return ThreatLevelLow
	}

	avgScore := totalScore / float64(count)
	return s.calculateThreatLevel(avgScore)
}

func (s *DynamicDefenseService) recordAccess(ip, userAgent, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pattern, exists := s.accessPatterns[ip]
	if !exists {
		return
	}

	if len(pattern.Methods) > 100 {
		pattern.Methods = pattern.Methods[len(pattern.Methods)-100:]
	}
}

func (s *DynamicDefenseService) GetDefenseState() *DefenseState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stateCopy := *s.defenseState
	return &stateCopy
}

func (s *DynamicDefenseService) ApplyPolicy(ctx context.Context, policyID string, targetIP string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	policy, exists := s.defensePolicies[policyID]
	if !exists {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	mitigation := &ActiveMitigation{
		ID:           fmt.Sprintf("mit_%d", time.Now().UnixNano()),
		PolicyID:     policy.ID,
		IP:           targetIP,
		Action:       policy.Action,
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(policy.Duration),
		Reason:       fmt.Sprintf("Policy: %s", policy.Name),
		RequestCount: 0,
		BlockedCount: 0,
	}

	s.activeMitigations[targetIP] = mitigation
	policy.HitCount++
	policy.UpdatedAt = time.Now()

	return nil
}

func (s *DynamicDefenseService) RemoveMitigation(targetIP string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.activeMitigations, targetIP)
	return nil
}

func (s *DynamicDefenseService) GetActiveMitigations() []*ActiveMitigation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mitigations := make([]*ActiveMitigation, 0)
	for _, m := range s.activeMitigations {
		if time.Now().Before(m.EndTime) {
			mitigations = append(mitigations, m)
		}
	}
	return mitigations
}

func (s *DynamicDefenseService) CreatePolicy(policy *DefensePolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	policy.ID = fmt.Sprintf("policy_%d", time.Now().UnixNano())
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	s.defensePolicies[policy.ID] = policy
	return nil
}

func (s *DynamicDefenseService) UpdatePolicy(policyID string, updates *DefensePolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	policy, exists := s.defensePolicies[policyID]
	if !exists {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	if updates.Name != "" {
		policy.Name = updates.Name
	}
	if updates.Priority > 0 {
		policy.Priority = updates.Priority
	}
	if updates.Condition != nil {
		policy.Condition = updates.Condition
	}
	if updates.Action != "" {
		policy.Action = updates.Action
	}
	if updates.Duration > 0 {
		policy.Duration = updates.Duration
	}
	policy.IsActive = updates.IsActive
	policy.UpdatedAt = time.Now()

	return nil
}

func (s *DynamicDefenseService) DeletePolicy(policyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.defensePolicies, policyID)
	return nil
}

func (s *DynamicDefenseService) GetPolicies() []*DefensePolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policies := make([]*DefensePolicy, 0)
	for _, p := range s.defensePolicies {
		policies = append(policies, p)
	}
	return policies
}

func (s *DynamicDefenseService) AddWAFRule(rule *WAFRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule.ID = fmt.Sprintf("waf_%d", time.Now().UnixNano())
	rule.CreatedAt = time.Now()
	rule.MatchCount = 0
	rule.IsActive = true
	s.wafRules = append(s.wafRules, rule)
	return nil
}

func (s *DynamicDefenseService) GetWAFRules() []*WAFRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]*WAFRule, len(s.wafRules))
	copy(rules, s.wafRules)
	return rules
}

func (s *DynamicDefenseService) EnableGeoBlocking(countries []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.geoBlocking.Enabled = true
	for _, country := range countries {
		s.geoBlocking.BlockedCountries[country] = true
	}
}

func (s *DynamicDefenseService) DisableGeoBlocking(countries []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, country := range countries {
		delete(s.geoBlocking.BlockedCountries, country)
	}
}

func (s *DynamicDefenseService) AddIPToWhitelist(cidr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ipRanges = append(s.ipRanges, &IPRangeConfig{
		CIDR:        cidr,
		Action:      ActionAllow,
		Priority:    100,
		IsWhitelist: true,
		Description: "User added whitelist",
	})
	return nil
}

func (s *DynamicDefenseService) AddIPToBlacklist(cidr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ipRanges = append(s.ipRanges, &IPRangeConfig{
		CIDR:        cidr,
		Action:      ActionBlock,
		Priority:    100,
		IsBlacklist: true,
		Description: "User added blacklist",
	})
	return nil
}

func (s *DynamicDefenseService) AdjustAdaptiveThresholds(thresholds *AdaptiveThresholds) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if thresholds.RequestPerMinute > 0 {
		s.adaptiveThresholds.RequestPerMinute = thresholds.RequestPerMinute
	}
	if thresholds.ErrorRate > 0 {
		s.adaptiveThresholds.ErrorRate = thresholds.ErrorRate
	}
	if thresholds.LatencyMs > 0 {
		s.adaptiveThresholds.LatencyMs = thresholds.LatencyMs
	}
}

func (s *DynamicDefenseService) GetDefenseStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_policies":     len(s.defensePolicies),
		"active_policies":    0,
		"total_mitigations":  atomic.LoadInt32(&s.defenseState.TotalMitigations),
		"active_blocks":      atomic.LoadInt32(&s.defenseState.ActiveBlocks),
		"active_challenges":  atomic.LoadInt32(&s.defenseState.ActiveChallenges),
		"waf_rules":          len(s.wafRules),
		"ip_ranges":          len(s.ipRanges),
		"current_threat_level": s.defenseState.CurrentLevel,
		"geo_blocking_enabled": s.geoBlocking.Enabled,
		"blocked_countries":  len(s.geoBlocking.BlockedCountries),
		"thresholds":         s.adaptiveThresholds,
	}

	activePolicies := 0
	for _, p := range s.defensePolicies {
		if p.IsActive {
			activePolicies++
		}
	}
	stats["active_policies"] = activePolicies

	var totalWAFMatches int
	for _, rule := range s.wafRules {
		totalWAFMatches += rule.MatchCount
	}
	stats["total_waf_matches"] = totalWAFMatches

	return stats
}

func (s *DynamicDefenseService) PerformSelfTuning(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats := s.GetDefenseStatistics()
	totalMitigations := atomic.LoadInt32(&s.defenseState.TotalMitigations)

	if totalMitigations > 1000 {
		s.adaptiveThresholds.RequestPerMinute = int(float64(s.adaptiveThresholds.RequestPerMinute) * 0.8)
	}

	if totalMitigations < 10 {
		s.adaptiveThresholds.RequestPerMinute = int(float64(s.adaptiveThresholds.RequestPerMinute) * 1.2)
	}

	_ = stats
	return nil
}

func (s *DynamicDefenseService) ExportConfiguration() (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config := map[string]interface{}{
		"policies":           s.defensePolicies,
		"waf_rules":          s.wafRules,
		"ip_ranges":          s.ipRanges,
		"geo_blocking":       s.geoBlocking,
		"adaptive_thresholds": s.adaptiveThresholds,
		"enabled":            s.enabled,
	}

	return config, nil
}

func (s *DynamicDefenseService) ImportConfiguration(config map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if policies, ok := config["policies"].(map[string]*DefensePolicy); ok {
		s.defensePolicies = policies
	}

	if enabled, ok := config["enabled"].(bool); ok {
		s.enabled = enabled
	}

	return nil
}

func (s *DynamicDefenseService) AnalyzeAttackPattern(ctx context.Context, ip string) (*AttackPatternAnalysis, error) {
	s.mu.RLock()
	pattern, exists := s.accessPatterns[ip]
	detector, detectorExists := s.anomalyDetectors[ip]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no pattern data for IP: %s", ip)
	}

	analysis := &AttackPatternAnalysis{
		IP:             ip,
		RequestCount:   pattern.RequestCount,
		UniquePaths:    len(pattern.Paths),
		FirstSeen:      pattern.FirstSeen,
		LastSeen:       pattern.LastSeen,
		IsSuspicious:   pattern.IsSuspicious,
		ThreatScore:    pattern.ThreatScore,
		PatternTypes:   []string{},
		Recommendations: []string{},
	}

	if detectorExists && detector.IsAnomalous {
		analysis.PatternTypes = append(analysis.PatternTypes, detector.AnomalyType)
		analysis.Recommendations = append(analysis.Recommendations, fmt.Sprintf("Detected anomaly: %s", detector.AnomalyType))
	}

	if pattern.RequestCount > 1000 {
		analysis.PatternTypes = append(analysis.PatternTypes, "high_volume")
		analysis.Recommendations = append(analysis.Recommendations, "Consider blocking due to excessive requests")
	}

	return analysis, nil
}

type AttackPatternAnalysis struct {
	IP              string
	RequestCount    int
	UniquePaths     int
	FirstSeen       time.Time
	LastSeen        time.Time
	IsSuspicious    bool
	ThreatScore     float64
	PatternTypes    []string
	Recommendations []string
}

func (s *DynamicDefenseService) GetTrafficBaseline(ip string) *TrafficBaseline {
	s.mu.RLock()
	profile, exists := s.trafficProfiles[ip]
	s.mu.RUnlock()

	if !exists {
		return &TrafficBaseline{
			IP:             ip,
			BaselineRequests: 10,
			BaselineBytes:   1000,
		}
	}

	return &TrafficBaseline{
		IP:               ip,
		BaselineRequests: int(profile.BaselineRequests),
		BaselineBytes:    profile.BaselineBytes,
		CurrentRequests:  int(profile.CurrentRequests),
		CurrentBytes:     profile.CurrentBytes,
		PeakRequests:     int(profile.PeakRequests),
		PeakBytes:        profile.PeakBytes,
		DeviationPercent: s.calculateDeviation(profile),
	}
}

type TrafficBaseline struct {
	IP               string
	BaselineRequests int
	BaselineBytes    int64
	CurrentRequests  int
	CurrentBytes     int64
	PeakRequests     int
	PeakBytes        int64
	DeviationPercent float64
}

func (s *DynamicDefenseService) calculateDeviation(profile *TrafficProfile) float64 {
	if profile.BaselineRequests == 0 {
		return 0
	}
	return float64(profile.CurrentRequests-profile.BaselineRequests) / profile.BaselineRequests * 100
}

func (s *DynamicDefenseService) UpdateTrafficProfile(ip string, requests int, bytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, exists := s.trafficProfiles[ip]
	if !exists {
		profile = &TrafficProfile{
			IP:               ip,
			BaselineRequests: float64(requests),
			BaselineBytes:    bytes,
		}
		s.trafficProfiles[ip] = profile
	}

	profile.CurrentRequests = float64(requests)
	profile.CurrentBytes = bytes
	profile.LastUpdated = time.Now()

	if float64(requests) > profile.PeakRequests {
		profile.PeakRequests = float64(requests)
	}
	if bytes > profile.PeakBytes {
		profile.PeakBytes = bytes
	}
}

func (s *DynamicDefenseService) ResetTrafficProfile(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.trafficProfiles, ip)
	delete(s.accessPatterns, ip)
	delete(s.anomalyDetectors, ip)
	delete(s.rateLimiters, ip)
}

func (s *DynamicDefenseService) EnableDefense() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = true
}

func (s *DynamicDefenseService) DisableDefense() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = false
}

func (s *DynamicDefenseService) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}
