package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type AIAttackDetectorService struct {
	attackModels         map[string]*AttackModel
	detectionRules       map[string]*DetectionRule
	requestHistory       map[string]*RequestSequence
	sessionProfiles      map[string]*SessionProfile
	attackIndicators     map[string]*AttackIndicator
	sequenceAnalyzers    map[string]*AISequenceAnalyzer
	mlClassifier        *MLAttackClassifier
	trainingData        []*TrainingSample
	isTraining          bool
	mu                  sync.RWMutex
	modelUpdateInterval time.Duration
	lastModelUpdate     time.Time
}

type AttackModel struct {
	ID          string
	Name        string
	Type        AttackCategory
	Version     string
	TrainedAt   time.Time
	Accuracy    float64
	Precision   float64
	Recall      float64
	F1Score     float64
	Features    []string
	IsActive    bool
	ConfusionMatrix *ConfusionMatrix
}

type ConfusionMatrix struct {
	TruePositives  int
	FalsePositives int
	TrueNegatives  int
	FalseNegatives int
}

type DetectionRule struct {
	ID           string
	Name         string
	Pattern      *regexp.Regexp
	AttackType   AttackCategory
	Severity     int
	Weight       float64
	Threshold    float64
	IsActive     bool
	MatchCount   int
	FalsePositives int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type RequestSequence struct {
	IP           string
	SessionID    string
	Requests     []*SequentialRequest
	SequenceHash string
	IsAttack     bool
	AttackType   AttackCategory
	Confidence   float64
	FirstSeen    time.Time
	LastSeen     time.Time
}

type SequentialRequest struct {
	Timestamp     time.Time
	Method        string
	Path          string
	UserAgent     string
	Headers       http.Header
	Body          string
	ResponseTime  time.Duration
	StatusCode    int
	SequenceOrder int
}

type SessionProfile struct {
	SessionID       string
	IP              string
	UserAgent       string
	StartTime       time.Time
	EndTime         time.Time
	RequestCount    int
	UniquePaths     map[string]bool
	Methods         map[string]int
	ErrorCount      int
	AvgResponseTime float64
	NavigationPath  []string
	IsBot           bool
	BotConfidence   float64
	ThreatScore     float64
}

type AttackIndicator struct {
	ID           string
	Type         string
	Value        string
	AttackType   AttackCategory
	FirstSeen    time.Time
	LastSeen     time.Time
	HitCount     int
	FalsePositiveRate float64
	IsActive     bool
}

type AISequenceAnalyzer struct {
	IP            string
	SequenceType  SequenceType
	DataPoints    []SequenceDataPoint
	Threshold     float64
	IsAnomalous   bool
	AnomalyScore  float64
	LastAnalyzed  time.Time
}

type SequenceDataPoint struct {
	Timestamp   time.Time
	SequenceLen int
	Entropy     float64
	Periodicity float64
	Complexity  float64
}

type MLAttackClassifier struct {
	ModelType    string
	FeatureWeights map[string]float64
	DecisionBoundary float64
	IsTrained    bool
	TrainingEpochs int
	LearningRate float64
}

type TrainingSample struct {
	Features   []float64
	Label      bool
	IsAttack   bool
	AttackType AttackCategory
	Timestamp  time.Time
}

type AttackCategory string

const (
	AttackCategorySQLInjection      AttackCategory = "sql_injection"
	AttackCategoryXSS              AttackCategory = "xss"
	AttackCategoryCSRF             AttackCategory = "csrf"
	AttackCategoryBruteForce       AttackCategory = "brute_force"
	AttackCategoryCredentialStuffing AttackCategory = "credential_stuffing"
	AttackCategoryDDoS             AttackCategory = "ddos"
	AttackCategoryWebShell         AttackCategory = "webshell"
	AttackCategoryAPIAbuse        AttackCategory = "api_abuse"
	AttackCategoryScraping         AttackCategory = "scraping"
	AttackCategoryAccountTakeover   AttackCategory = "account_takeover"
	AttackCategorySessionHijacking  AttackCategory = "session_hijacking"
	AttackCategoryZeroDay          AttackCategory = "zero_day"
	AttackCategoryBot              AttackCategory = "bot"
	AttackCategoryNormal           AttackCategory = "normal"
)

type SequenceType string

const (
	SequenceTypeNavigation SequenceType = "navigation"
	SequenceTypeTiming    SequenceType = "timing"
	SequenceTypePayload    SequenceType = "payload"
	SequenceTypeMixed      SequenceType = "mixed"
)

type AttackDetectionResult struct {
	IsAttack          bool
	AttackType        AttackCategory
	Confidence        float64
	Severity          int
	Indicators        []string
	MitigationActions []string
	AnalysisDetails   map[string]interface{}
}

type BehavioralAnalysisResult struct {
	IsAnomalous      bool
	AnomalyScore     float64
	AnomalyTypes     []string
	PatternMatch     string
	RiskLevel        string
	Recommendations  []string
}

type MLPredictionResult struct {
	AttackProbability float64
	Category          AttackCategory
	Confidence        float64
	Features          []string
	TopFeatures       []string
}

func NewAIAttackDetectorService() *AIAttackDetectorService {
	service := &AIAttackDetectorService{
		attackModels:       make(map[string]*AttackModel),
		detectionRules:     make(map[string]*DetectionRule),
		requestHistory:     make(map[string]*RequestSequence),
		sessionProfiles:    make(map[string]*SessionProfile),
		attackIndicators:   make(map[string]*AttackIndicator),
		sequenceAnalyzers:  make(map[string]*AISequenceAnalyzer),
		modelUpdateInterval: 5 * time.Minute,
	}

	service.mlClassifier = &MLAttackClassifier{
		ModelType:    "logistic_regression",
		FeatureWeights: make(map[string]float64),
		DecisionBoundary: 0.5,
		IsTrained:   true,
	}

	service.initializeDefaultModels()
	service.initializeDetectionRules()
	service.initializeFeatureWeights()
	return service
}

func (s *AIAttackDetectorService) initializeDefaultModels() {
	s.attackModels["sql_injection_v1"] = &AttackModel{
		ID:        "sql_injection_v1",
		Name:      "SQL Injection Detector",
		Type:      AttackCategorySQLInjection,
		Version:   "1.0",
		TrainedAt: time.Now(),
		Accuracy:  0.95,
		Precision: 0.93,
		Recall:    0.96,
		F1Score:   0.945,
		Features:  []string{"query_length", "special_chars", "sql_keywords", "union_pattern", "comment_markers"},
		IsActive:  true,
	}
	s.attackModels["xss_v1"] = &AttackModel{
		ID:        "xss_v1",
		Name:      "XSS Detector",
		Type:      AttackCategoryXSS,
		Version:   "1.0",
		TrainedAt: time.Now(),
		Accuracy:  0.94,
		Precision: 0.91,
		Recall:    0.95,
		F1Score:   0.93,
		Features:  []string{"script_tags", "event_handlers", "javascript_uri", "encoded_chars", "html_tags"},
		IsActive:  true,
	}
	s.attackModels["brute_force_v1"] = &AttackModel{
		ID:        "brute_force_v1",
		Name:      "Brute Force Detector",
		Type:      AttackCategoryBruteForce,
		Version:   "1.0",
		TrainedAt: time.Now(),
		Accuracy:  0.97,
		Precision: 0.98,
		Recall:    0.95,
		F1Score:   0.965,
		Features:  []string{"failed_attempts", "request_rate", "password_pattern", "username_variety", "timing_pattern"},
		IsActive:  true,
	}
	s.attackModels["bot_v1"] = &AttackModel{
		ID:        "bot_v1",
		Name:      "Bot Detector",
		Type:      AttackCategoryBot,
		Version:   "1.0",
		TrainedAt: time.Now(),
		Accuracy:  0.92,
		Precision: 0.89,
		Recall:    0.94,
		F1Score:   0.915,
		Features:  []string{"user_agent", "behavior_pattern", "headless_browser", "mouse_movement", "timing_human"},
		IsActive:  true,
	}
}

func (s *AIAttackDetectorService) initializeDetectionRules() {
	s.detectionRules["sqli_union"] = &DetectionRule{
		ID:         "sqli_union",
		Name:       "SQL Union Injection",
		Pattern:    regexp.MustCompile(`(?i)(union\s+(all\s+)?select|union\s+select)`),
		AttackType: AttackCategorySQLInjection,
		Severity:   5,
		Weight:     0.9,
		Threshold:  0.7,
		IsActive:   true,
		CreatedAt:  time.Now(),
	}
	s.detectionRules["sqli_or"] = &DetectionRule{
		ID:         "sqli_or",
		Name:       "SQL OR Injection",
		Pattern:    regexp.MustCompile(`(?i)(or\s+['"]?\w+['"]?\s*=\s*['"]?\w+['"]?|or\s+1\s*=\s*1)`),
		AttackType: AttackCategorySQLInjection,
		Severity:   5,
		Weight:     0.85,
		Threshold:  0.7,
		IsActive:   true,
		CreatedAt:  time.Now(),
	}
	s.detectionRules["xss_script"] = &DetectionRule{
		ID:         "xss_script",
		Name:       "XSS Script Tag",
		Pattern:    regexp.MustCompile(`(?i)(<script[^>]*>|</script|javascript:|on\w+\s*=)`),
		AttackType: AttackCategoryXSS,
		Severity:   5,
		Weight:     0.88,
		Threshold:  0.7,
		IsActive:   true,
		CreatedAt:  time.Now(),
	}
	s.detectionRules["xss_img"] = &DetectionRule{
		ID:         "xss_img",
		Name:       "XSS Image Tag",
		Pattern:    regexp.MustCompile(`(?i)(<img[^>]+src\s*=\s*["']?[^"']*["']?|onerror\s*=)`),
		AttackType: AttackCategoryXSS,
		Severity:   4,
		Weight:     0.8,
		Threshold:  0.7,
		IsActive:   true,
		CreatedAt:  time.Now(),
	}
	s.detectionRules["brute_force_pattern"] = &DetectionRule{
		ID:         "brute_force_pattern",
		Name:       "Brute Force Attempt",
		Pattern:    regexp.MustCompile(`(?i)(login|signin|auth|password|passwd)`),
		AttackType: AttackCategoryBruteForce,
		Severity:   3,
		Weight:     0.6,
		Threshold:  0.5,
		IsActive:   true,
		CreatedAt:  time.Now(),
	}
	s.detectionRules["path_traversal"] = &DetectionRule{
		ID:         "path_traversal",
		Name:       "Path Traversal",
		Pattern:    regexp.MustCompile(`(?i)(\.\.[\/\\]|%2e%2e%2f|%2e%2e%5c)`),
		AttackType: AttackCategorySQLInjection,
		Severity:   5,
		Weight:     0.92,
		Threshold:  0.7,
		IsActive:   true,
		CreatedAt:  time.Now(),
	}
	s.detectionRules["command_injection"] = &DetectionRule{
		ID:         "command_injection",
		Name:       "Command Injection",
		Pattern:    regexp.MustCompile(`(?i)(;|\||\`+"`"+`|\$\(|&&|\\n|\\r)`),
		AttackType: AttackCategoryZeroDay,
		Severity:   5,
		Weight:     0.9,
		Threshold:  0.7,
		IsActive:   true,
		CreatedAt:  time.Now(),
	}
}

func (s *AIAttackDetectorService) initializeFeatureWeights() {
	s.mlClassifier.FeatureWeights = map[string]float64{
		"request_rate":        0.15,
		"error_rate":           0.12,
		"unique_paths":         0.10,
		"session_duration":    0.08,
		"failed_auths":        0.18,
		"payload_complexity":   0.14,
		"geo_inconsistency":   0.05,
		"device_fingerprint":   0.10,
		"behavioral_pattern":  0.08,
		"timing_pattern":       0.10,
	}
}

func (s *AIAttackDetectorService) DetectAttack(ctx context.Context, r *http.Request, sessionID string) (*AttackDetectionResult, error) {
	result := &AttackDetectionResult{
		IsAttack:          false,
		AttackType:        AttackCategoryNormal,
		Confidence:        0.0,
		Severity:          0,
		Indicators:        []string{},
		MitigationActions: []string{},
		AnalysisDetails:   make(map[string]interface{}),
	}

	ip := getClientIP(r)

	ruleResults := s.evaluateRules(r)
	for _, ruleResult := range ruleResults {
		if ruleResult.Matched {
			result.IsAttack = true
			result.AttackType = ruleResult.AttackType
			result.Confidence += ruleResult.Weight
			result.Severity = max(result.Severity, ruleResult.Severity)
			result.Indicators = append(result.Indicators, ruleResult.RuleName)
		}
	}

	sequenceResult := s.analyzeSequence(ip, sessionID, r)
	if sequenceResult.IsAnomalous {
		result.Confidence += sequenceResult.Score * 0.3
		result.Indicators = append(result.Indicators, fmt.Sprintf("sequence:%s", sequenceResult.Type))
	}

	behaviorResult := s.performBehavioralAnalysis(ip, sessionID, r)
	if behaviorResult.IsAnomalous {
		result.Confidence += behaviorResult.AnomalyScore * 0.2
		result.Indicators = append(result.Indicators, behaviorResult.AnomalyTypes...)
	}

	mlResult := s.mlPredict(r, sequenceResult, behaviorResult)
	if mlResult.AttackProbability > 0.5 {
		result.IsAttack = true
		if result.Confidence < mlResult.AttackProbability {
			result.Confidence = mlResult.AttackProbability
			result.AttackType = mlResult.Category
		}
		result.Indicators = append(result.Indicators, mlResult.TopFeatures...)
	}

	result.Confidence = math.Min(result.Confidence, 1.0)
	result.MitigationActions = s.generateMitigationActions(result)

	s.recordDetection(ip, sessionID, r, result)

	return result, nil
}

type RuleMatchResult struct {
	RuleName  string
	Matched   bool
	AttackType AttackCategory
	Severity  int
	Weight    float64
}

func (s *AIAttackDetectorService) evaluateRules(r *http.Request) []*RuleMatchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*RuleMatchResult
	requestURI := r.RequestURI
	queryString := r.URL.RawQuery

	for _, rule := range s.detectionRules {
		if !rule.IsActive {
			continue
		}

		matched := rule.Pattern.MatchString(requestURI) || rule.Pattern.MatchString(queryString)
		if matched {
			rule.MatchCount++
			results = append(results, &RuleMatchResult{
				RuleName:  rule.Name,
				Matched:   true,
				AttackType: rule.AttackType,
				Severity:  rule.Severity,
				Weight:    rule.Weight,
			})
		}
	}

	return results
}

func (s *AIAttackDetectorService) analyzeSequence(ip, sessionID string, r *http.Request) *SequenceAnalysisResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", ip, sessionID)
	analyzer, exists := s.sequenceAnalyzers[key]
	if !exists {
		analyzer = &AISequenceAnalyzer{
			IP:           ip,
			SequenceType: SequenceTypeMixed,
			DataPoints:   make([]SequenceDataPoint, 0),
			Threshold:    0.8,
		}
		s.sequenceAnalyzers[key] = analyzer
	}

	dataPoint := SequenceDataPoint{
		Timestamp:   time.Now(),
		SequenceLen: len(analyzer.DataPoints) + 1,
		Entropy:     s.calculateSequenceEntropy(analyzer.DataPoints),
		Periodicity: s.calculatePeriodicity(analyzer.DataPoints),
		Complexity:  s.calculateSequenceComplexity(analyzer.DataPoints),
	}
	analyzer.DataPoints = append(analyzer.DataPoints, dataPoint)

	if len(analyzer.DataPoints) > 100 {
		analyzer.DataPoints = analyzer.DataPoints[len(analyzer.DataPoints)-100:]
	}

	result := &SequenceAnalysisResult{
		IsAnomalous: false,
		Score:       0.0,
		Type:        string(analyzer.SequenceType),
	}

	if len(analyzer.DataPoints) >= 10 {
		avgEntropy := s.calculateAverageEntropy(analyzer.DataPoints[:len(analyzer.DataPoints)-1])
		if dataPoint.Entropy < avgEntropy*0.5 {
			result.IsAnomalous = true
			result.Score = 0.7
			result.Type = "low_entropy_sequence"
		}

		avgPeriodicity := s.calculateAveragePeriodicity(analyzer.DataPoints[:len(analyzer.DataPoints)-1])
		if dataPoint.Periodicity > avgPeriodicity*1.5 && dataPoint.Periodicity > 0.8 {
			result.IsAnomalous = true
			result.Score = 0.8
			result.Type = "mechanical_periodicity"
		}
	}

	analyzer.IsAnomalous = result.IsAnomalous
	analyzer.AnomalyScore = result.Score
	analyzer.LastAnalyzed = time.Now()

	return result
}

type SequenceAnalysisResult struct {
	IsAnomalous bool
	Score       float64
	Type        string
}

func (s *AIAttackDetectorService) calculateSequenceEntropy(points []SequenceDataPoint) float64 {
	if len(points) < 2 {
		return 0.5
	}
	sum := 0.0
	for _, p := range points {
		sum += p.Complexity
	}
	return sum / float64(len(points))
}

func (s *AIAttackDetectorService) calculatePeriodicity(points []SequenceDataPoint) float64 {
	if len(points) < 3 {
		return 0.0
	}
	timeDiffs := make([]float64, 0)
	for i := 1; i < len(points); i++ {
		diff := points[i].Timestamp.Sub(points[i-1].Timestamp).Seconds()
		timeDiffs = append(timeDiffs, diff)
	}

	avg := 0.0
	for _, d := range timeDiffs {
		avg += d
	}
	avg /= float64(len(timeDiffs))

	if avg == 0 {
		return 0.0
	}

	variance := 0.0
	for _, d := range timeDiffs {
		diff := d - avg
		variance += diff * diff
	}
	variance /= float64(len(timeDiffs))

	return 1.0 - (math.Sqrt(variance) / avg)
}

func (s *AIAttackDetectorService) calculateSequenceComplexity(points []SequenceDataPoint) float64 {
	if len(points) < 2 {
		return 0.0
	}

	uniqueTimestamps := make(map[int64]bool)
	for _, p := range points {
		uniqueTimestamps[p.Timestamp.Unix()] = true
	}

	uniqueRatio := float64(len(uniqueTimestamps)) / float64(len(points))
	return uniqueRatio
}

func (s *AIAttackDetectorService) calculateAverageEntropy(points []SequenceDataPoint) float64 {
	if len(points) == 0 {
		return 0.5
	}
	sum := 0.0
	for _, p := range points {
		sum += p.Entropy
	}
	return sum / float64(len(points))
}

func (s *AIAttackDetectorService) calculateAveragePeriodicity(points []SequenceDataPoint) float64 {
	if len(points) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, p := range points {
		sum += p.Periodicity
	}
	return sum / float64(len(points))
}

func (s *AIAttackDetectorService) performBehavioralAnalysis(ip, sessionID string, r *http.Request) *BehavioralAnalysisResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", ip, sessionID)
	profile, exists := s.sessionProfiles[key]
	if !exists {
		profile = &SessionProfile{
			SessionID:     sessionID,
			IP:            ip,
			UserAgent:     r.UserAgent(),
			StartTime:     time.Now(),
			UniquePaths:   make(map[string]bool),
			Methods:       make(map[string]int),
			ThreatScore:   0,
		}
		s.sessionProfiles[key] = profile
	}

	profile.RequestCount++
	profile.UniquePaths[r.URL.Path] = true
	profile.Methods[r.Method]++
	profile.NavigationPath = append(profile.NavigationPath, r.URL.Path)

	if len(profile.NavigationPath) > 100 {
		profile.NavigationPath = profile.NavigationPath[len(profile.NavigationPath)-100:]
	}

	result := &BehavioralAnalysisResult{
		AnomalyTypes:    []string{},
		Recommendations: []string{},
	}

	if profile.RequestCount > 500 {
		result.AnomalyTypes = append(result.AnomalyTypes, "excessive_requests")
		result.RiskLevel = "high"
		result.IsAnomalous = true
	}

	if float64(len(profile.UniquePaths))/float64(profile.RequestCount) < 0.1 && profile.RequestCount > 50 {
		result.AnomalyTypes = append(result.AnomalyTypes, "low_path_diversity")
		result.RiskLevel = "medium"
		result.IsAnomalous = true
	}

	suspiciousPaths := []string{"/admin", "/login", "/api", "/config"}
	focusCount := 0
	for _, path := range suspiciousPaths {
		for _, visitedPath := range profile.NavigationPath {
			if strings.Contains(visitedPath, path) {
				focusCount++
			}
		}
	}
	if focusCount > profile.RequestCount/2 && profile.RequestCount > 20 {
		result.AnomalyTypes = append(result.AnomalyTypes, "path_focus_attack")
		result.RiskLevel = "high"
		result.IsAnomalous = true
	}

	if result.IsAnomalous {
		profile.ThreatScore = 0.8
	}

	if result.RiskLevel == "" {
		result.RiskLevel = "low"
	}

	return result
}

func (s *AIAttackDetectorService) mlPredict(r *http.Request, seqResult *SequenceAnalysisResult, behResult *BehavioralAnalysisResult) *MLPredictionResult {
	result := &MLPredictionResult{
		Features:     []string{},
		TopFeatures:  []string{},
	}

	features := make(map[string]float64)

	features["payload_complexity"] = s.calculatePayloadComplexity(r.URL.RawQuery)
	features["request_rate"] = s.estimateRequestRate(r)
	features["unique_paths"] = float64(len(strings.Split(r.URL.Path, "/")))

	if seqResult.IsAnomalous {
		features["sequence_anomaly"] = seqResult.Score
	} else {
		features["sequence_anomaly"] = 0.0
	}

	if behResult.IsAnomalous {
		features["behavior_anomaly"] = behResult.AnomalyScore
	} else {
		features["behavior_anomaly"] = 0.0
	}

	var weightedSum float64
	var totalWeight float64
	for feature, value := range features {
		weight := s.mlClassifier.FeatureWeights[feature]
		if weight == 0 {
			weight = 0.1
		}
		weightedSum += value * weight
		totalWeight += weight
		result.Features = append(result.Features, fmt.Sprintf("%s:%.2f", feature, value))
	}

	rawProbability := weightedSum / totalWeight
	result.AttackProbability = math.Min(math.Max(rawProbability, 0), 1)

	if result.AttackProbability > 0.7 {
		result.Category = AttackCategorySQLInjection
		result.TopFeatures = append(result.TopFeatures, "sql_keywords")
	} else if result.AttackProbability > 0.5 {
		result.Category = AttackCategoryXSS
		result.TopFeatures = append(result.TopFeatures, "script_tags")
	} else if result.AttackProbability > 0.3 {
		result.Category = AttackCategoryBot
		result.TopFeatures = append(result.TopFeatures, "behavior_pattern")
	} else {
		result.Category = AttackCategoryNormal
	}

	result.Confidence = result.AttackProbability

	return result
}

func (s *AIAttackDetectorService) calculatePayloadComplexity(query string) float64 {
	if query == "" {
		return 0.0
	}

	specialChars := 0
	for _, c := range query {
		if strings.ContainsRune("!@#$%^&*()_+-=[]{}|;':\",./<>?`~", c) {
			specialChars++
		}
	}

	length := len(query)
	uniqueChars := len(map[rune]struct{}{})
	for _, c := range query {
		uniqueChars++
	}

	complexity := (float64(specialChars) / float64(length) * 0.4) +
		(float64(uniqueChars) / float64(length) * 0.3) +
		(math.Min(float64(length)/100.0, 1.0) * 0.3)

	return math.Min(complexity, 1.0)
}

func (s *AIAttackDetectorService) estimateRequestRate(r *http.Request) float64 {
	ip := getClientIP(r)
	s.mu.RLock()
	analyzer, exists := s.sequenceAnalyzers[ip]
	s.mu.RUnlock()

	if !exists || len(analyzer.DataPoints) < 2 {
		return 0.1
	}

	now := time.Now()
	windowStart := now.Add(-1 * time.Minute)
	recentCount := 0
	for _, dp := range analyzer.DataPoints {
		if dp.Timestamp.After(windowStart) {
			recentCount++
		}
	}

	rate := float64(recentCount) / 60.0
	return math.Min(rate/10.0, 1.0)
}

func (s *AIAttackDetectorService) generateMitigationActions(result *AttackDetectionResult) []string {
	var actions []string

	switch result.AttackType {
	case AttackCategorySQLInjection:
		actions = append(actions, "启用WAF SQL注入防护规则")
		actions = append(actions, "记录完整请求日志")
		actions = append(actions, "考虑阻止该IP")
	case AttackCategoryXSS:
		actions = append(actions, "启用WAF XSS防护规则")
		actions = append(actions, "对请求参数进行HTML编码")
		actions = append(actions, "启用内容安全策略")
	case AttackCategoryBruteForce:
		actions = append(actions, "临时封禁该IP")
		actions = append(actions, "启用账户锁定机制")
		actions = append(actions, "要求验证码验证")
	case AttackCategoryBot:
		actions = append(actions, "返回验证码挑战")
		actions = append(actions, "启用Bot检测")
		actions = append(actions, "添加JavaScript挑战")
	default:
		actions = append(actions, "记录并监控")
		actions = append(actions, "根据置信度调整响应")
	}

	if result.Severity >= 5 {
		actions = append(actions, "立即阻止请求")
		actions = append(actions, "通知安全团队")
	}

	return actions
}

func (s *AIAttackDetectorService) recordDetection(ip, sessionID string, r *http.Request, result *AttackDetectionResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s", ip, sessionID)
	sequence, exists := s.requestHistory[key]
	if !exists {
		sequence = &RequestSequence{
			IP:           ip,
			SessionID:    sessionID,
			Requests:     make([]*SequentialRequest, 0),
			FirstSeen:    time.Now(),
		}
		s.requestHistory[key] = sequence
	}

	req := &SequentialRequest{
		Timestamp:     time.Now(),
		Method:        r.Method,
		Path:          r.URL.Path,
		UserAgent:     r.UserAgent(),
		Headers:       r.Header,
		SequenceOrder: len(sequence.Requests),
	}
	sequence.Requests = append(sequence.Requests, req)

	if len(sequence.Requests) > 1000 {
		sequence.Requests = sequence.Requests[len(sequence.Requests)-1000:]
	}

	sequence.LastSeen = time.Now()
	sequence.IsAttack = result.IsAttack
	sequence.AttackType = result.AttackType
	sequence.Confidence = result.Confidence
}

func (s *AIAttackDetectorService) GetAttackStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_models":         len(s.attackModels),
		"active_models":         0,
		"total_rules":           len(s.detectionRules),
		"active_rules":         0,
		"total_sequences":      len(s.requestHistory),
		"total_sessions":       len(s.sessionProfiles),
		"total_indicators":     len(s.attackIndicators),
		"last_model_update":    s.lastModelUpdate,
	}

	activeModels := 0
	activeRules := 0
	totalMatches := 0

	for _, model := range s.attackModels {
		if model.IsActive {
			activeModels++
		}
	}
	for _, rule := range s.detectionRules {
		if rule.IsActive {
			activeRules++
		}
		totalMatches += rule.MatchCount
	}

	stats["active_models"] = activeModels
	stats["active_rules"] = activeRules
	stats["total_rule_matches"] = totalMatches

	var totalAnomalies int
	for _, analyzer := range s.sequenceAnalyzers {
		if analyzer.IsAnomalous {
			totalAnomalies++
		}
	}
	stats["anomalous_sequences"] = totalAnomalies

	return stats
}

func (s *AIAttackDetectorService) TrainModel(ctx context.Context, modelID string, samples []*TrainingSample) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	model, exists := s.attackModels[modelID]
	if !exists {
		return fmt.Errorf("model not found: %s", modelID)
	}

	s.isTraining = true
	defer func() { s.isTraining = false }()

	var truePositives, falsePositives, trueNegatives, falseNegatives int
	var totalPrecision, totalRecall float64

	for _, sample := range samples {
		prediction := s.predictSample(sample)
		if prediction && sample.IsAttack {
			truePositives++
		} else if prediction && !sample.IsAttack {
			falsePositives++
		} else if !prediction && sample.IsAttack {
			falseNegatives++
		} else {
			trueNegatives++
		}
	}

	if truePositives+falsePositives > 0 {
		model.Precision = float64(truePositives) / float64(truePositives+falsePositives)
	}
	if truePositives+falseNegatives > 0 {
		model.Recall = float64(truePositives) / float64(truePositives+falseNegatives)
	}
	if model.Precision+model.Recall > 0 {
		model.F1Score = 2 * (model.Precision * model.Recall) / (model.Precision + model.Recall)
	}
	if truePositives+falsePositives+falseNegatives+trueNegatives > 0 {
		model.Accuracy = float64(truePositives+trueNegatives) / float64(truePositives+falsePositives+falseNegatives+trueNegatives)
	}

	model.ConfusionMatrix = &ConfusionMatrix{
		TruePositives:  truePositives,
		FalsePositives: falsePositives,
		TrueNegatives:  trueNegatives,
		FalseNegatives: falseNegatives,
	}
	model.TrainedAt = time.Now()
	model.Version = fmt.Sprintf("v%d.%d", time.Now().Year(), time.Now().Unix())

	s.trainingData = append(s.trainingData, samples...)

	return nil
}

func (s *AIAttackDetectorService) predictSample(sample *TrainingSample) bool {
	if len(sample.Features) == 0 {
		return false
	}

	var sum float64
	for _, f := range sample.Features {
		sum += f
	}
	avg := sum / float64(len(sample.Features))

	return avg > s.mlClassifier.DecisionBoundary
}

func (s *AIAttackDetectorService) AddDetectionRule(rule *DetectionRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	rule.MatchCount = 0
	rule.IsActive = true
	s.detectionRules[rule.ID] = rule
	return nil
}

func (s *AIAttackDetectorService) UpdateDetectionRule(ruleID string, updates *DetectionRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule, exists := s.detectionRules[ruleID]
	if !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	if updates.Pattern != nil {
		rule.Pattern = updates.Pattern
	}
	if updates.Severity > 0 {
		rule.Severity = updates.Severity
	}
	if updates.Weight > 0 {
		rule.Weight = updates.Weight
	}
	if updates.Threshold > 0 {
		rule.Threshold = updates.Threshold
	}
	rule.IsActive = updates.IsActive
	rule.UpdatedAt = time.Now()

	return nil
}

func (s *AIAttackDetectorService) GetDetectionRules() []*DetectionRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]*DetectionRule, 0)
	for _, rule := range s.detectionRules {
		rules = append(rules, rule)
	}
	return rules
}

func (s *AIAttackDetectorService) GetAttackIndicators(ctx context.Context, attackType AttackCategory) []*AttackIndicator {
	s.mu.RLock()
	defer s.mu.RUnlock()

	indicators := make([]*AttackIndicator, 0)
	for _, indicator := range s.attackIndicators {
		if indicator.AttackType == attackType && indicator.IsActive {
			indicators = append(indicators, indicator)
		}
	}
	return indicators
}

func (s *AIAttackDetectorService) AddAttackIndicator(indicator *AttackIndicator) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	indicator.ID = fmt.Sprintf("indicator_%d", time.Now().UnixNano())
	indicator.FirstSeen = time.Now()
	indicator.LastSeen = time.Now()
	indicator.HitCount = 0
	indicator.IsActive = true
	s.attackIndicators[indicator.ID] = indicator
	return nil
}

func (s *AIAttackDetectorService) GetSequenceAnalysis(ip, sessionID string) (*AISequenceAnalyzer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", ip, sessionID)
	analyzer, exists := s.sequenceAnalyzers[key]
	if !exists {
		return nil, fmt.Errorf("no sequence data for %s", key)
	}
	return analyzer, nil
}

func (s *AIAttackDetectorService) GetSessionProfile(sessionID string) (*SessionProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, profile := range s.sessionProfiles {
		if profile.SessionID == sessionID {
			return profile, nil
		}
	}
	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (s *AIAttackDetectorService) ExportModelConfig() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config := map[string]interface{}{
		"models":           s.attackModels,
		"rules":            s.detectionRules,
		"feature_weights":  s.mlClassifier.FeatureWeights,
		"decision_boundary": s.mlClassifier.DecisionBoundary,
		"export_time":       time.Now(),
	}

	return json.MarshalIndent(config, "", "  ")
}

func (s *AIAttackDetectorService) ImportModelConfig(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	if weights, ok := config["feature_weights"].(map[string]float64); ok {
		s.mlClassifier.FeatureWeights = weights
	}

	if boundary, ok := config["decision_boundary"].(float64); ok {
		s.mlClassifier.DecisionBoundary = boundary
	}

	return nil
}

func (s *AIAttackDetectorService) UpdateFeatureWeights(weights map[string]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range weights {
		s.mlClassifier.FeatureWeights[k] = v
	}
}

func (s *AIAttackDetectorService) GetModelPerformance(modelID string) (*ModelPerformance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	model, exists := s.attackModels[modelID]
	if !exists {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}

	return &ModelPerformance{
		ModelID:    model.ID,
		Accuracy:   model.Accuracy,
		Precision:  model.Precision,
		Recall:     model.Recall,
		F1Score:    model.F1Score,
		ConfusionMatrix: model.ConfusionMatrix,
		TrainedAt:  model.TrainedAt,
		Version:    model.Version,
	}, nil
}

type ModelPerformance struct {
	ModelID        string
	Accuracy       float64
	Precision      float64
	Recall         float64
	F1Score        float64
	ConfusionMatrix *ConfusionMatrix
	TrainedAt      time.Time
	Version        string
}

func (s *AIAttackDetectorService) PerformIncrementalLearning(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.trainingData) < 10 {
		return nil
	}

	recentSamples := s.trainingData
	if len(recentSamples) > 1000 {
		recentSamples = recentSamples[len(recentSamples)-1000:]
	}

	featureStats := make(map[string]struct{ sum, count float64 })
	for _, sample := range recentSamples {
		for i, feature := range sample.Features {
			key := fmt.Sprintf("feature_%d", i)
			featureStats[key].sum += feature
			featureStats[key].count++
		}
	}

	for key, stats := range featureStats {
		if stats.count > 0 {
			avg := stats.sum / stats.count
			s.mlClassifier.FeatureWeights[key] = math.Min(avg*1.1, 1.0)
		}
	}

	s.mlClassifier.DecisionBoundary = 0.5

	return nil
}

func (s *AIAttackDetectorService) ClearDetectionHistory() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requestHistory = make(map[string]*RequestSequence)
	s.sequenceAnalyzers = make(map[string]*AISequenceAnalyzer)
}

func (s *AIAttackDetectorService) GetActiveThreats() []*ActiveThreat {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var threats []*ActiveThreat
	for _, sequence := range s.requestHistory {
		if sequence.IsAttack && time.Since(sequence.LastSeen) < 1*time.Hour {
			threat := &ActiveThreat{
				IP:         sequence.IP,
				SessionID:  sequence.SessionID,
				AttackType: sequence.AttackType,
				Confidence: sequence.Confidence,
				FirstSeen:  sequence.FirstSeen,
				LastSeen:   sequence.LastSeen,
				RequestCount: len(sequence.Requests),
			}
			threats = append(threats, threat)
		}
	}
	return threats
}

type ActiveThreat struct {
	IP           string
	SessionID    string
	AttackType   AttackCategory
	Confidence   float64
	FirstSeen    time.Time
	LastSeen     time.Time
	RequestCount int
}
