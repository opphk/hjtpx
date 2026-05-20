package service

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

// ============================================
// 第一部分：行为预测服务核心（任务3.4）
// ============================================

type BehaviorPredictionService struct {
	intentPredictor     *IntentPredictor
	riskAssessor        *ProactiveRiskAssessor
	smartInterceptor    *SmartInterceptor
	sequenceAnalyzer    *SequenceAnalyzer
	neuralPredictor    *NeuralBehaviorPredictor
	onlineLearner       *OnlineLearningEngine
	mu                 sync.RWMutex
}

type IntentPredictor struct {
	userIntents map[string][]IntentSequence
	models      map[string]*IntentModel
	mu          sync.RWMutex
}

type IntentSequence struct {
	SequenceID    string
	UserID        string
	SessionID     string
	Actions       []UserAction
	PredictedIntent Intent
	Confidence    float64
	Timestamp     time.Time
}

type UserAction struct {
	ActionType string
	Target     string
	Timestamp  time.Time
	Duration   time.Duration
	Success    bool
	Metadata   map[string]interface{}
}

type Intent struct {
	Type           string
	Goal           string
	SubIntents     []Intent
	Urgency        float64
	Legitimacy     float64
	RelatedEntities []string
	Confidence     float64
}

type IntentModel struct {
	IntentType      string
	ActionPatterns  []ActionPattern
	SuccessRate     float64
	AvgDuration     float64
	CommonTargets   []string
	LastUpdated     time.Time
}

type ActionPattern struct {
	ActionType    string
	Frequency     float64
	AvgDuration   float64
	SuccessRate   float64
	NextActions   map[string]float64
}

type ProactiveRiskAssessor struct {
	userRiskProfiles map[string]*RiskProfile
	riskIndicators   map[string]*RiskIndicator
	globalThresholds *RiskThresholds
	mu               sync.RWMutex
}

type RiskProfile struct {
	UserID          string
	BaseRiskScore   float64
	CurrentRiskScore float64
	RiskTrend       string
	RiskFactors     []PredictionRiskFactor
	LastAssessment  time.Time
	TrustedDevices  []string
	KnownLocations  []string
}

type PredictionRiskFactor struct {
	FactorType    string
	Severity      float64
	Weight        float64
	Contributing  []string
	DetectedAt    time.Time
}

type RiskIndicator struct {
	IndicatorID    string
	IndicatorType  string
	CurrentValue   float64
	Threshold      float64
	IsActive       bool
	FalsePositiveRate float64
}

type RiskThresholds struct {
	LowRiskThreshold   float64
	MediumRiskThreshold float64
	HighRiskThreshold  float64
	CriticalThreshold  float64
}

type SmartInterceptor struct {
	interceptionRules map[string]*InterceptionRule
	whitelist        *WhitelistManager
	blacklist        *BlacklistManager
	mu               sync.RWMutex
}

type InterceptionRule struct {
	RuleID         string
	RuleType       string
	Priority       int
	Conditions     []RuleCondition
	Action         string
	IsEnabled      bool
	HitCount       int
	LastHit        time.Time
	Effectiveness  float64
}

type RuleCondition struct {
	Field     string
	Operator  string
	Value     interface{}
	Threshold float64
}

type WhitelistManager struct {
	entries map[string]*WhitelistEntry
	mu     sync.RWMutex
}

type WhitelistEntry struct {
	Identifier string
	Type       string
	AddedAt    time.Time
	ExpiresAt  *time.Time
	AddedBy    string
	Reason     string
}

type BlacklistManager struct {
	entries map[string]*BlacklistEntry
	mu     sync.RWMutex
}

type BlacklistEntry struct {
	Identifier   string
	Type         string
	AddedAt      time.Time
	ExpiresAt    *time.Time
	AddedBy      string
	Reason       string
	Severity     string
	HitCount     int
	LastHit      time.Time
}

type SequenceAnalyzer struct {
	sequences map[string]*ActionSequence
	patterns  map[string]*SequencePattern
	mu        sync.RWMutex
}

type ActionSequence struct {
	SequenceID  string
	UserID      string
	SessionID   string
	Actions     []SequenceAction
	StartTime   time.Time
	EndTime     time.Time
	IsComplete  bool
	Intention   string
}

type SequenceAction struct {
	ActionType   string
	Target       string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Result       string
	NextExpected string
}

type SequencePattern struct {
	PatternID     string
	PatternType   string
	ActionSequence []string
	Frequency     float64
	SuccessRate   float64
	IsSuspicious  bool
	Indicators    []string
}

type PredictionRequest struct {
	UserID          string
	SessionID       string
	CurrentAction   *UserAction
	RecentActions   []UserAction
	EnvironmentData map[string]interface{}
	HistoricalData  []models.BehaviorData
}

type PredictionResult struct {
	PredictedIntent   *Intent
	RiskAssessment    *RiskAssessment
	ShouldIntercept   bool
	InterceptionReason string
	RecommendedAction string
	Confidence        float64
	WarningLevel      string
	NextActions       []string
	DetailedAnalysis  string
}

type RiskAssessment struct {
	OverallRiskScore  float64
	RiskLevel         string
	RiskFactors       []PredictionRiskFactor
	ImmediateThreats  []ThreatInfo
	TrendAnalysis     string
	Recommendation    string
}

type ThreatInfo struct {
	ThreatType    string
	ThreatLevel   string
	Confidence     float64
	AffectedArea   string
	SuggestedAction string
}

func NewBehaviorPredictionService() *BehaviorPredictionService {
	service := &BehaviorPredictionService{
		intentPredictor: &IntentPredictor{
			userIntents: make(map[string][]IntentSequence),
			models:      make(map[string]*IntentModel),
		},
		riskAssessor: &ProactiveRiskAssessor{
			userRiskProfiles: make(map[string]*RiskProfile),
			riskIndicators:   make(map[string]*RiskIndicator),
			globalThresholds: &RiskThresholds{
				LowRiskThreshold:    20.0,
				MediumRiskThreshold: 50.0,
				HighRiskThreshold:   75.0,
				CriticalThreshold:   90.0,
			},
		},
		smartInterceptor: &SmartInterceptor{
			interceptionRules: make(map[string]*InterceptionRule),
			whitelist: &WhitelistManager{
				entries: make(map[string]*WhitelistEntry),
			},
			blacklist: &BlacklistManager{
				entries: make(map[string]*BlacklistEntry),
			},
		},
		sequenceAnalyzer: &SequenceAnalyzer{
			sequences: make(map[string]*ActionSequence),
			patterns:  make(map[string]*SequencePattern),
		},
		neuralPredictor: NewNeuralBehaviorPredictor(),
		onlineLearner:   NewOnlineLearningEngine(),
	}
	
	service.initializeDefaultModels()
	service.initializeDefaultRules()
	service.initializeDefaultPatterns()
	
	return service
}

func (s *BehaviorPredictionService) PredictUserBehavior(req *PredictionRequest) *PredictionResult {
	
	intent := s.intentPredictor.predictIntent(req)
	
	riskAssessment := s.riskAssessor.assessRisk(req)
	
	intercept, reason := s.smartInterceptor.shouldIntercept(req, riskAssessment)
	
	recommendedAction := s.determineRecommendedAction(intent, riskAssessment, intercept)
	
	nextActions := s.sequenceAnalyzer.predictNextActions(req)
	
	result := &PredictionResult{
		PredictedIntent:     intent,
		RiskAssessment:     riskAssessment,
		ShouldIntercept:     intercept,
		InterceptionReason:  reason,
		RecommendedAction:   recommendedAction,
		Confidence:          intent.Confidence,
		WarningLevel:        s.determineWarningLevel(riskAssessment),
		NextActions:        nextActions,
	}
	
	result.DetailedAnalysis = s.generateAnalysisText(result)
	
	s.updateModels(req, result)
	
	return result
}

func (s *BehaviorPredictionService) initializeDefaultModels() {
	s.intentPredictor.models["login_attempt"] = &IntentModel{
		IntentType:     "login_attempt",
		ActionPatterns: []ActionPattern{
			{ActionType: "click_username_field", Frequency: 1.0, AvgDuration: 1.5},
			{ActionType: "type_username", Frequency: 0.9, AvgDuration: 2.0},
			{ActionType: "click_password_field", Frequency: 0.95, AvgDuration: 0.5},
			{ActionType: "type_password", Frequency: 0.9, AvgDuration: 2.5},
			{ActionType: "click_submit", Frequency: 1.0, AvgDuration: 0.3},
		},
		SuccessRate: 0.85,
		AvgDuration: 7.0,
	}
	
	s.intentPredictor.models["account_registration"] = &IntentModel{
		IntentType:     "account_registration",
		ActionPatterns: []ActionPattern{
			{ActionType: "fill_form", Frequency: 1.0, AvgDuration: 15.0},
			{ActionType: "upload_file", Frequency: 0.3, AvgDuration: 5.0},
			{ActionType: "accept_terms", Frequency: 1.0, AvgDuration: 0.5},
			{ActionType: "submit_form", Frequency: 1.0, AvgDuration: 0.5},
		},
		SuccessRate: 0.75,
		AvgDuration: 25.0,
	}
	
	s.intentPredictor.models["suspicious_activity"] = &IntentModel{
		IntentType:     "suspicious_activity",
		ActionPatterns: []ActionPattern{
			{ActionType: "rapid_clicks", Frequency: 1.0, AvgDuration: 2.0},
			{ActionType: "pattern_repetition", Frequency: 0.9, AvgDuration: 10.0},
			{ActionType: "unusual_timing", Frequency: 0.8, AvgDuration: 0.0},
		},
		SuccessRate: 0.95,
		AvgDuration: 5.0,
	}
}

func (s *BehaviorPredictionService) initializeDefaultRules() {
	s.smartInterceptor.interceptionRules["rapid_fire"] = &InterceptionRule{
		RuleID:    "rapid_fire",
		RuleType:  "rate_limit",
		Priority:  1,
		Conditions: []RuleCondition{
			{Field: "action_count", Operator: ">", Value: 10, Threshold: 10},
			{Field: "time_window", Operator: "<", Value: 5, Threshold: 5},
		},
		Action:        "challenge",
		IsEnabled:    true,
		Effectiveness: 0.85,
	}
	
	s.smartInterceptor.interceptionRules["suspicious_pattern"] = &InterceptionRule{
		RuleID:    "suspicious_pattern",
		RuleType:  "pattern_match",
		Priority:  2,
		Conditions: []RuleCondition{
			{Field: "pattern_match", Operator: "==", Value: "suspicious", Threshold: 0.8},
		},
		Action:        "block",
		IsEnabled:    true,
		Effectiveness: 0.92,
	}
}

func (s *BehaviorPredictionService) initializeDefaultPatterns() {
	s.sequenceAnalyzer.patterns["credential_stuffing"] = &SequencePattern{
		PatternID:     "credential_stuffing",
		PatternType:   "attack",
		ActionSequence: []string{"rapid_login_attempts", "failed_authentication", "ip_change"},
		Frequency:     0.05,
		SuccessRate:   0.02,
		IsSuspicious:  true,
		Indicators:    []string{"rapid_repeated_attempts", "credential_variation"},
	}
	
	s.sequenceAnalyzer.patterns["normal_user"] = &SequencePattern{
		PatternID:     "normal_user",
		PatternType:   "normal",
		ActionSequence: []string{"page_load", "form_filling", "submission", "success"},
		Frequency:     0.85,
		SuccessRate:   0.90,
		IsSuspicious:  false,
		Indicators:    []string{"natural_pacing", "typical_time_durations"},
	}
}

func (ip *IntentPredictor) predictIntent(req *PredictionRequest) *Intent {
	
	if len(req.RecentActions) < 2 {
		return &Intent{
			Type:        "unknown",
			Confidence:  0.3,
			Legitimacy: 0.5,
		}
	}
	
	actionSequence := ip.extractActionSequence(req.RecentActions)
	
	matchedModel := ip.matchSequenceToModel(actionSequence)
	
	var intentType string
	var urgency float64
	var legitimacy float64
	
	switch {
	case ip.matchesPattern(actionSequence, "login_attempt"):
		intentType = "login"
		urgency = 0.7
		legitimacy = 0.8
	case ip.matchesPattern(actionSequence, "account_registration"):
		intentType = "registration"
		urgency = 0.5
		legitimacy = 0.9
	case ip.detectSuspiciousBehavior(req):
		intentType = "suspicious"
		urgency = 0.9
		legitimacy = 0.1
	default:
		intentType = "general"
		urgency = 0.5
		legitimacy = 0.7
	}
	
	intent := &Intent{
		Type:        intentType,
		Goal:        ip.determineGoal(actionSequence),
		Urgency:     urgency,
		Legitimacy:  legitimacy,
		Confidence:  ip.calculateConfidence(actionSequence, matchedModel),
	}
	
	return intent
}

func (ip *IntentPredictor) extractActionSequence(actions []UserAction) []string {
	sequence := make([]string, 0, len(actions))
	for _, action := range actions {
		sequence = append(sequence, action.ActionType)
	}
	return sequence
}

func (ip *IntentPredictor) matchSequenceToModel(sequence []string) *IntentModel {
	
	for _, model := range ip.models {
		patternLen := len(model.ActionPatterns)
		if patternLen == 0 || len(sequence) < patternLen/2 {
			continue
		}
		
		matchCount := 0
		for _, seqAction := range sequence {
			for _, pattern := range model.ActionPatterns {
				if seqAction == pattern.ActionType {
					matchCount++
					break
				}
			}
		}
		
		matchRatio := float64(matchCount) / float64(len(sequence))
		if matchRatio > 0.6 {
			return model
		}
	}
	
	return nil
}

func (ip *IntentPredictor) matchesPattern(sequence []string, patternType string) bool {
	model := ip.models[patternType]
	if model == nil {
		return false
	}
	
	patternActions := make(map[string]bool)
	for _, pattern := range model.ActionPatterns {
		patternActions[pattern.ActionType] = true
	}
	
	matchCount := 0
	for _, action := range sequence {
		if patternActions[action] {
			matchCount++
		}
	}
	
	return float64(matchCount) / float64(len(sequence)) > 0.5
}

func (ip *IntentPredictor) detectSuspiciousBehavior(req *PredictionRequest) bool {
	if len(req.RecentActions) < 3 {
		return false
	}
	
	avgDuration := 0.0
	for _, action := range req.RecentActions {
		avgDuration += action.Duration.Seconds()
	}
	avgDuration /= float64(len(req.RecentActions))
	
	if avgDuration < 0.5 {
		return true
	}
	
	rapidClicks := 0
	for i := 1; i < len(req.RecentActions); i++ {
		if req.RecentActions[i].Timestamp.Sub(req.RecentActions[i-1].Timestamp) < 200*time.Millisecond {
			rapidClicks++
		}
	}
	
	if rapidClicks > len(req.RecentActions)/2 {
		return true
	}
	
	return false
}

func (ip *IntentPredictor) determineGoal(sequence []string) string {
	if len(sequence) == 0 {
		return "unknown"
	}
	
	lastAction := sequence[len(sequence)-1]
	switch lastAction {
	case "click_submit", "submit_form":
		return "form_submission"
	case "type_input":
		return "data_entry"
	case "click_navigation":
		return "navigation"
	case "rapid_clicks":
		return "automated_activity"
	default:
		return "general_interaction"
	}
}

func (ip *IntentPredictor) calculateConfidence(sequence []string, model *IntentModel) float64 {
	if model == nil {
		return 0.5
	}
	
	baseConfidence := 0.7
	
	matchRatio := 0.0
	for _, action := range sequence {
		for _, pattern := range model.ActionPatterns {
			if action == pattern.ActionType {
				matchRatio += pattern.Frequency
				break
			}
		}
	}
	
	if len(sequence) > 0 {
		matchRatio /= float64(len(sequence))
	}
	
	return math.Min(1.0, baseConfidence+matchRatio*0.3)
}

func (pra *ProactiveRiskAssessor) assessRisk(req *PredictionRequest) *RiskAssessment {
	profile := pra.getOrCreateRiskProfile(req.UserID)
	
	riskScore := profile.BaseRiskScore
	var riskFactors []PredictionRiskFactor
	
	if len(req.RecentActions) > 0 {
		avgDuration := 0.0
		for _, action := range req.RecentActions {
			avgDuration += action.Duration.Seconds()
		}
		avgDuration /= float64(len(req.RecentActions))
		
		if avgDuration < 0.3 {
			factor := PredictionRiskFactor{
				FactorType: "rapid_actions",
				Severity:   0.8,
				Weight:     0.3,
				DetectedAt: time.Now(),
			}
			riskFactors = append(riskFactors, factor)
			riskScore += factor.Severity * factor.Weight * 20
		}
	}
	
	if len(req.EnvironmentData) > 0 {
		envRisk := pra.assessEnvironmentRisk(req.EnvironmentData)
		if envRisk > 0 {
			factor := PredictionRiskFactor{
				FactorType: "environment",
				Severity:   envRisk,
				Weight:     0.25,
				DetectedAt: time.Now(),
			}
			riskFactors = append(riskFactors, factor)
			riskScore += factor.Severity * factor.Weight * 30
		}
	}
	
	riskScore = math.Max(0, math.Min(100, riskScore))
	
	threats := pra.identifyThreats(riskScore, riskFactors)
	
	assessment := &RiskAssessment{
		OverallRiskScore: riskScore,
		RiskLevel:       pra.determineRiskLevel(riskScore),
		RiskFactors:     riskFactors,
		ImmediateThreats: threats,
		TrendAnalysis:   pra.analyzeRiskTrend(profile),
		Recommendation:  pra.generateRecommendation(riskScore, riskFactors),
	}
	
	profile.CurrentRiskScore = riskScore
	profile.RiskTrend = assessment.TrendAnalysis
	profile.LastAssessment = time.Now()
	
	return assessment
}

func (pra *ProactiveRiskAssessor) getOrCreateRiskProfile(userID string) *RiskProfile {
	pra.mu.Lock()
	defer pra.mu.Unlock()
	
	if profile, exists := pra.userRiskProfiles[userID]; exists {
		return profile
	}
	
	profile := &RiskProfile{
		UserID:          userID,
		BaseRiskScore:   30.0,
		CurrentRiskScore: 30.0,
		RiskTrend:      "stable",
		RiskFactors:    []PredictionRiskFactor{},
		LastAssessment: time.Now(),
		TrustedDevices:  []string{},
		KnownLocations:  []string{},
	}
	pra.userRiskProfiles[userID] = profile
	return profile
}

func (pra *ProactiveRiskAssessor) assessEnvironmentRisk(env map[string]interface{}) float64 {
	risk := 0.0
	
	if ip, ok := env["ip_address"].(string); ok {
		if pra.isKnownSuspiciousIP(ip) {
			risk += 0.6
		}
		if pra.isProxyIP(ip) {
			risk += 0.3
		}
	}
	
	if ua, ok := env["user_agent"].(string); ok {
		if pra.isSuspiciousUserAgent(ua) {
			risk += 0.4
		}
	}
	
	if fingerprint, ok := env["fingerprint"].(string); ok {
		if pra.isNewDevice(fingerprint) {
			risk += 0.2
		}
	}
	
	return math.Min(1.0, risk)
}

func (pra *ProactiveRiskAssessor) isKnownSuspiciousIP(ip string) bool {
	return false
}

func (pra *ProactiveRiskAssessor) isProxyIP(ip string) bool {
	return false
}

func (pra *ProactiveRiskAssessor) isSuspiciousUserAgent(ua string) bool {
	suspicious := []string{"curl", "wget", "python-requests", "scrapy", "bot"}
	for _, s := range suspicious {
		if len(ua) > len(s) && (ua[:len(s)] == s || len(ua) > 10 && containsSubstr(ua, s)) {
			return true
		}
	}
	return false
}

func (pra *ProactiveRiskAssessor) isNewDevice(fingerprint string) bool {
	return false
}

func (pra *ProactiveRiskAssessor) identifyThreats(score float64, factors []PredictionRiskFactor) []ThreatInfo {
	threats := []ThreatInfo{}
	
	if score > pra.globalThresholds.HighRiskThreshold {
		threats = append(threats, ThreatInfo{
			ThreatType:    "high_risk_activity",
			ThreatLevel:   "high",
			Confidence:    0.85,
			AffectedArea:  "authentication",
			SuggestedAction: "require_additional_verification",
		})
	}
	
	for _, factor := range factors {
		if factor.Severity > 0.7 {
			threats = append(threats, ThreatInfo{
				ThreatType:    factor.FactorType,
				ThreatLevel:   "medium",
				Confidence:    factor.Severity,
				AffectedArea:  "behavior_pattern",
				SuggestedAction: "monitor_closely",
			})
		}
	}
	
	return threats
}

func (pra *ProactiveRiskAssessor) determineRiskLevel(score float64) string {
	if score < pra.globalThresholds.LowRiskThreshold {
		return "low"
	} else if score < pra.globalThresholds.MediumRiskThreshold {
		return "medium"
	} else if score < pra.globalThresholds.HighRiskThreshold {
		return "high"
	}
	return "critical"
}

func (pra *ProactiveRiskAssessor) analyzeRiskTrend(profile *RiskProfile) string {
	if profile.CurrentRiskScore < profile.BaseRiskScore-10 {
		return "decreasing"
	} else if profile.CurrentRiskScore > profile.BaseRiskScore+10 {
		return "increasing"
	}
	return "stable"
}

func (pra *ProactiveRiskAssessor) generateRecommendation(score float64, factors []PredictionRiskFactor) string {
	if score < pra.globalThresholds.LowRiskThreshold {
		return "allow_with_normal_monitoring"
	} else if score < pra.globalThresholds.MediumRiskThreshold {
		return "allow_with_increased_monitoring"
	} else if score < pra.globalThresholds.HighRiskThreshold {
		return "require_additional_verification"
	}
	return "block_and_review"
}

func (si *SmartInterceptor) shouldIntercept(req *PredictionRequest, assessment *RiskAssessment) (bool, string) {
	
	if si.isWhitelisted(req) {
		return false, ""
	}
	
	if si.isBlacklisted(req) {
		return true, "blacklisted_entity"
	}
	
	for _, rule := range si.interceptionRules {
		if !rule.IsEnabled {
			continue
		}
		
		if si.evaluateRule(rule, req, assessment) {
			rule.HitCount++
			rule.LastHit = time.Now()
			return true, fmt.Sprintf("rule_%s_triggered", rule.RuleID)
		}
	}
	
	if assessment.OverallRiskScore > 85 {
		return true, "critical_risk_level"
	}
	
	return false, ""
}

func (si *SmartInterceptor) isWhitelisted(req *PredictionRequest) bool {
	si.whitelist.mu.RLock()
	defer si.whitelist.mu.RUnlock()
	
	key := fmt.Sprintf("%s:%s", req.UserID, req.SessionID)
	if entry, exists := si.whitelist.entries[key]; exists {
		if entry.ExpiresAt == nil || entry.ExpiresAt.After(time.Now()) {
			return true
		}
	}
	
	return false
}

func (si *SmartInterceptor) isBlacklisted(req *PredictionRequest) bool {
	si.blacklist.mu.RLock()
	defer si.blacklist.mu.RUnlock()
	
	key := fmt.Sprintf("%s:%s", req.UserID, req.SessionID)
	if entry, exists := si.blacklist.entries[key]; exists {
		if entry.ExpiresAt == nil || entry.ExpiresAt.After(time.Now()) {
			entry.HitCount++
			entry.LastHit = time.Now()
			return true
		}
	}
	
	return false
}

func (si *SmartInterceptor) evaluateRule(rule *InterceptionRule, req *PredictionRequest, assessment *RiskAssessment) bool {
	for _, condition := range rule.Conditions {
		if !si.evaluateCondition(condition, req, assessment) {
			return false
		}
	}
	return true
}

func (si *SmartInterceptor) evaluateCondition(condition RuleCondition, req *PredictionRequest, assessment *RiskAssessment) bool {
	switch condition.Field {
	case "action_count":
		actionCount := len(req.RecentActions)
		return si.compareValues(float64(actionCount), condition.Operator, condition.Threshold)
	case "time_window":
		if len(req.RecentActions) > 1 {
			window := req.RecentActions[len(req.RecentActions)-1].Timestamp.Sub(req.RecentActions[0].Timestamp).Seconds()
			return si.compareValues(window, condition.Operator, condition.Threshold)
		}
	case "risk_score":
		return si.compareValues(assessment.OverallRiskScore, condition.Operator, condition.Threshold)
	}
	return false
}

func (si *SmartInterceptor) compareValues(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case "==":
		return value == threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	}
	return false
}

func (sa *SequenceAnalyzer) predictNextActions(req *PredictionRequest) []string {
	actionSequence := make([]string, 0, len(req.RecentActions))
	for _, action := range req.RecentActions {
		actionSequence = append(actionSequence, action.ActionType)
	}
	
	predictions := make([]string, 0)
	
	for _, pattern := range sa.patterns {
		if !pattern.IsSuspicious && pattern.SuccessRate > 0.8 {
			nextActions := sa.findNextActionsInPattern(actionSequence, pattern)
			predictions = append(predictions, nextActions...)
		}
	}
	
	return predictions
}

func (sa *SequenceAnalyzer) findNextActionsInPattern(current []string, pattern *SequencePattern) []string {
	for i := 0; i <= len(current)-len(pattern.ActionSequence); i++ {
		match := true
		for j := 0; j < len(pattern.ActionSequence) && i+j < len(current); j++ {
			if current[i+j] != pattern.ActionSequence[j] {
				match = false
				break
			}
		}
		
		if match && i+len(pattern.ActionSequence) < len(current)+2 {
			return pattern.ActionSequence[len(pattern.ActionSequence):]
		}
	}
	
	return []string{}
}

func (s *BehaviorPredictionService) determineRecommendedAction(
	intent *Intent,
	assessment *RiskAssessment,
	shouldIntercept bool,
) string {
	
	if shouldIntercept {
		if assessment.OverallRiskScore > 90 {
			return "block"
		}
		return "challenge"
	}
	
	if intent.Type == "suspicious" || assessment.OverallRiskScore > 70 {
		return "additional_verification"
	}
	
	if intent.Legitimacy > 0.8 && assessment.OverallRiskScore < 30 {
		return "allow"
	}
	
	return "monitor"
}

func (s *BehaviorPredictionService) determineWarningLevel(assessment *RiskAssessment) string {
	if assessment.OverallRiskScore > 85 {
		return "critical"
	} else if assessment.OverallRiskScore > 70 {
		return "high"
	} else if assessment.OverallRiskScore > 50 {
		return "medium"
	} else if assessment.OverallRiskScore > 30 {
		return "low"
	}
	return "none"
}

func (s *BehaviorPredictionService) generateAnalysisText(result *PredictionResult) string {
	text := fmt.Sprintf("行为分析：检测到用户意图为 %s，可信度 %.1f%%。\n",
		result.PredictedIntent.Type, result.PredictedIntent.Confidence*100)
	
	text += fmt.Sprintf("风险评估：当前风险等级为 %s，风险评分 %.1f。\n",
		result.RiskAssessment.RiskLevel, result.RiskAssessment.OverallRiskScore)
	
	if len(result.RiskAssessment.RiskFactors) > 0 {
		text += "风险因素："
		for _, factor := range result.RiskAssessment.RiskFactors {
			text += fmt.Sprintf("%s(严重度%.2f) ", factor.FactorType, factor.Severity)
		}
		text += "\n"
	}
	
	if result.ShouldIntercept {
		text += fmt.Sprintf("拦截原因：%s\n", result.InterceptionReason)
	}
	
	text += fmt.Sprintf("建议操作：%s\n", result.RecommendedAction)
	
	return text
}

func (s *BehaviorPredictionService) updateModels(req *PredictionRequest, result *PredictionResult) {
	
	if req.UserID != "" && len(req.RecentActions) > 0 {
		sequence := IntentSequence{
			SequenceID:     fmt.Sprintf("%s_%d", req.SessionID, time.Now().Unix()),
			UserID:         req.UserID,
			SessionID:      req.SessionID,
			Actions:        req.RecentActions,
			PredictedIntent: *result.PredictedIntent,
			Confidence:     result.Confidence,
			Timestamp:      time.Now(),
		}
		
		s.intentPredictor.mu.Lock()
		s.intentPredictor.userIntents[req.UserID] = append(
			s.intentPredictor.userIntents[req.UserID], sequence)
		
		if len(s.intentPredictor.userIntents[req.UserID]) > 100 {
			s.intentPredictor.userIntents[req.UserID] = 
				s.intentPredictor.userIntents[req.UserID][len(s.intentPredictor.userIntents[req.UserID])-100:]
		}
		s.intentPredictor.mu.Unlock()
	}
}

func (s *BehaviorPredictionService) AddToWhitelist(identifier string, identifierType string, duration time.Duration, reason string) {
	entry := &WhitelistEntry{
		Identifier: identifier,
		Type:       identifierType,
		AddedAt:    time.Now(),
		AddedBy:    "system",
		Reason:     reason,
	}
	
	if duration > 0 {
		expires := time.Now().Add(duration)
		entry.ExpiresAt = &expires
	}
	
	s.smartInterceptor.whitelist.mu.Lock()
	defer s.smartInterceptor.whitelist.mu.Unlock()
	
	key := fmt.Sprintf("%s:%s", identifierType, identifier)
	s.smartInterceptor.whitelist.entries[key] = entry
}

func (s *BehaviorPredictionService) AddToBlacklist(identifier string, identifierType string, duration time.Duration, reason string, severity string) {
	entry := &BlacklistEntry{
		Identifier: identifier,
		Type:       identifierType,
		AddedAt:    time.Now(),
		AddedBy:    "system",
		Reason:     reason,
		Severity:   severity,
	}
	
	if duration > 0 {
		expires := time.Now().Add(duration)
		entry.ExpiresAt = &expires
	}
	
	s.smartInterceptor.blacklist.mu.Lock()
	defer s.smartInterceptor.blacklist.mu.Unlock()
	
	key := fmt.Sprintf("%s:%s", identifierType, identifier)
	s.smartInterceptor.blacklist.entries[key] = entry
}

func (s *BehaviorPredictionService) GetRiskProfile(userID string) *RiskProfile {
	s.riskAssessor.mu.RLock()
	defer s.riskAssessor.mu.RUnlock()
	return s.riskAssessor.userRiskProfiles[userID]
}

func (s *BehaviorPredictionService) UpdateRiskThresholds(thresholds *RiskThresholds) {
	s.riskAssessor.mu.Lock()
	defer s.riskAssessor.mu.Unlock()
	s.riskAssessor.globalThresholds = thresholds
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================
// 第二部分：神经网络行为预测器（任务3.4）
// ============================================

// NeuralBehaviorPredictor 神经网络行为预测器
type NeuralBehaviorPredictor struct {
	inputSize     int
	hiddenSize    int
	outputSize    int
	learningRate  float64
	weights       *NeuralWeights
	activationHist []float64
	mu            sync.RWMutex
}

// NeuralWeights 神经网络权重
type NeuralWeights struct {
	InputHidden  [][]float64  // 输入层到隐藏层权重
	HiddenOutput [][]float64  // 隐藏层到输出层权重
	HiddenBias   []float64    // 隐藏层偏置
	OutputBias   []float64    // 输出层偏置
}

// NeuralPredictionResult 神经网络预测结果
type NeuralPredictionResult struct {
	BotProbability   float64            // 机器人概率
	HumanLikelihood  float64            // 人类可能性
	IntentPrediction *IntentPrediction   // 意图预测
	Confidence       float64            // 置信度
	FeatureAnalysis  map[string]float64 // 特征分析
	RiskIndicators   []string           // 风险指标
}

// IntentPrediction 意图预测
type IntentPrediction struct {
	PrimaryIntent   string
	SecondaryIntent string
	IntentConfidence float64
	IntentSequence  []string
}

// NewNeuralBehaviorPredictor 创建神经网络预测器
func NewNeuralBehaviorPredictor() *NeuralBehaviorPredictor {
	return &NeuralBehaviorPredictor{
		inputSize:     100,
		hiddenSize:    50,
		outputSize:    3,
		learningRate:  0.01,
		weights:       nil,
		activationHist: make([]float64, 0),
	}
}

// InitializeWeights 初始化神经网络权重
func (n *NeuralBehaviorPredictor) InitializeWeights() {
	n.weights = &NeuralWeights{
		InputHidden:  make([][]float64, n.inputSize),
		HiddenOutput: make([][]float64, n.hiddenSize),
		HiddenBias:   make([]float64, n.hiddenSize),
		OutputBias:   make([]float64, n.outputSize),
	}

	// 初始化输入层到隐藏层权重
	for i := 0; i < n.inputSize; i++ {
		n.weights.InputHidden[i] = make([]float64, n.hiddenSize)
		for j := 0; j < n.hiddenSize; j++ {
			n.weights.InputHidden[i][j] = (math.rand.Float64() - 0.5) * 0.5
		}
	}

	// 初始化隐藏层到输出层权重
	for i := 0; i < n.hiddenSize; i++ {
		n.weights.HiddenOutput[i] = make([]float64, n.outputSize)
		for j := 0; j < n.outputSize; j++ {
			n.weights.HiddenOutput[i][j] = (math.rand.Float64() - 0.5) * 0.5
		}
	}

	// 初始化偏置
	for i := 0; i < n.hiddenSize; i++ {
		n.weights.HiddenBias[i] = (math.rand.Float64() - 0.5) * 0.1
	}
	for i := 0; i < n.outputSize; i++ {
		n.weights.OutputBias[i] = (math.rand.Float64() - 0.5) * 0.1
	}
}

// Predict 使用神经网络进行行为预测
func (n *NeuralBehaviorPredictor) Predict(features []float64) *NeuralPredictionResult {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.weights == nil {
		n.InitializeWeights()
	}

	// 确保特征数量匹配
	if len(features) < n.inputSize {
		padded := make([]float64, n.inputSize)
		copy(padded, features)
		features = padded
	} else if len(features) > n.inputSize {
		features = features[:n.inputSize]
	}

	// 前向传播
	hiddenActivations := n.forwardHiddenLayer(features)
	outputActivations := n.forwardOutputLayer(hiddenActivations)

	// 解析输出
	result := &NeuralPredictionResult{
		BotProbability:  outputActivations[0],
		HumanLikelihood: outputActivations[1],
		Confidence:      outputActivations[2],
		IntentPrediction: &IntentPrediction{
			PrimaryIntent:   n.classifyIntent(outputActivations),
			IntentConfidence: outputActivations[2],
		},
		FeatureAnalysis: make(map[string]float64),
		RiskIndicators: []string{},
	}

	// 分析关键特征
	result.FeatureAnalysis = n.analyzeKeyFeatures(features)

	// 生成风险指标
	result.RiskIndicators = n.generateRiskIndicators(result.BotProbability, features)

	return result
}

// forwardHiddenLayer 前向传播到隐藏层
func (n *NeuralBehaviorPredictor) forwardHiddenLayer(inputs []float64) []float64 {
	hidden := make([]float64, n.hiddenSize)

	for j := 0; j < n.hiddenSize; j++ {
		sum := n.weights.HiddenBias[j]
		for i := 0; i < n.inputSize; i++ {
			sum += inputs[i] * n.weights.InputHidden[i][j]
		}
		// ReLU激活函数
		hidden[j] = math.Max(0, sum)
	}

	return hidden
}

// forwardOutputLayer 前向传播到输出层
func (n *NeuralBehaviorPredictor) forwardOutputLayer(hidden []float64) []float64 {
	output := make([]float64, n.outputSize)

	for k := 0; k < n.outputSize; k++ {
		sum := n.weights.OutputBias[k]
		for j := 0; j < n.hiddenSize; j++ {
			sum += hidden[j] * n.weights.HiddenOutput[j][k]
		}
		// Sigmoid激活函数
		output[k] = 1.0 / (1.0 + math.Exp(-sum))
	}

	return output
}

// classifyIntent 根据输出分类意图
func (n *NeuralBehaviorPredictor) classifyIntent(outputs []float64) string {
	maxIdx := 0
	maxVal := outputs[0]
	for i := 1; i < len(outputs); i++ {
		if outputs[i] > maxVal {
			maxVal = outputs[i]
			maxIdx = i
		}
	}

	intents := []string{"normal_user", "suspicious", "bot"}
	if maxIdx < len(intents) {
		return intents[maxIdx]
	}
	return "unknown"
}

// analyzeKeyFeatures 分析关键特征
func (n *NeuralBehaviorPredictor) analyzeKeyFeatures(features []float64) map[string]float64 {
	analysis := make(map[string]float64)

	// 分析速度特征（前20个特征）
	if len(features) >= 20 {
		speedSum := 0.0
		for i := 0; i < 20; i++ {
			speedSum += features[i]
		}
		analysis["avg_speed"] = speedSum / 20.0
	}

	// 分析加速度特征（20-40）
	if len(features) >= 40 {
		accelSum := 0.0
		for i := 20; i < 40; i++ {
			accelSum += features[i]
		}
		analysis["avg_acceleration"] = accelSum / 20.0
	}

	// 分析轨迹特征（40-60）
	if len(features) >= 60 {
		trajSum := 0.0
		for i := 40; i < 60; i++ {
			trajSum += features[i]
		}
		analysis["avg_trajectory"] = trajSum / 20.0
	}

	// 分析时间特征（60-80）
	if len(features) >= 80 {
		timeSum := 0.0
		for i := 60; i < 80; i++ {
			timeSum += features[i]
		}
		analysis["avg_timing"] = timeSum / 20.0
	}

	// 分析模式特征（80-100）
	if len(features) >= 100 {
		patternSum := 0.0
		for i := 80; i < 100; i++ {
			patternSum += features[i]
		}
		analysis["avg_pattern"] = patternSum / 20.0
	}

	return analysis
}

// generateRiskIndicators 生成风险指标
func (n *NeuralBehaviorPredictor) generateRiskIndicators(botProb float64, features []float64) []string {
	indicators := []string{}

	if botProb > 0.7 {
		indicators = append(indicators, "high_bot_probability")
	}

	if len(features) >= 20 {
		// 检查异常速度
		speedVariance := 0.0
		speedSum := 0.0
		for i := 0; i < 20; i++ {
			speedSum += features[i]
		}
		speedMean := speedSum / 20.0
		for i := 0; i < 20; i++ {
			speedVariance += (features[i] - speedMean) * (features[i] - speedMean)
		}
		if speedVariance < 0.01 {
			indicators = append(indicators, "abnormal_speed_consistency")
		}
	}

	if len(features) >= 100 {
		// 检查异常模式
		patternVariance := 0.0
		patternSum := 0.0
		for i := 80; i < 100; i++ {
			patternSum += features[i]
		}
		patternMean := patternSum / 20.0
		for i := 80; i < 100; i++ {
			patternVariance += (features[i] - patternMean) * (features[i] - patternMean)
		}
		if patternVariance < 0.005 {
			indicators = append(indicators, "mechanical_pattern_detected")
		}
	}

	return indicators
}

// Train 训练神经网络
func (n *NeuralBehaviorPredictor) Train(features []float64, expectedOutput []float64) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.weights == nil {
		n.InitializeWeights()
	}

	// 确保特征数量匹配
	if len(features) < n.inputSize {
		padded := make([]float64, n.inputSize)
		copy(padded, features)
		features = padded
	} else if len(features) > n.inputSize {
		features = features[:n.inputSize]
	}

	// 前向传播
	hiddenActivations := n.forwardHiddenLayer(features)
	outputActivations := n.forwardOutputLayer(hiddenActivations)

	// 反向传播
	n.backpropagate(features, hiddenActivations, outputActivations, expectedOutput)
}

// backpropagate 反向传播算法
func (n *NeuralBehaviorPredictor) backpropagate(inputs, hidden, output, expected []float64) {
	// 计算输出层误差
	outputErrors := make([]float64, n.outputSize)
	for k := 0; k < n.outputSize; k++ {
		if k < len(expected) {
			outputErrors[k] = (expected[k] - output[k]) * output[k] * (1.0 - output[k])
		}
	}

	// 计算隐藏层误差
	hiddenErrors := make([]float64, n.hiddenSize)
	for j := 0; j < n.hiddenSize; j++ {
		errorSum := 0.0
		for k := 0; k < n.outputSize; k++ {
			errorSum += outputErrors[k] * n.weights.HiddenOutput[j][k]
		}
		hiddenErrors[j] = hidden[j] * (1.0 - hidden[j]) * errorSum
	}

	// 更新隐藏层到输出层权重
	for j := 0; j < n.hiddenSize; j++ {
		for k := 0; k < n.outputSize; k++ {
			n.weights.HiddenOutput[j][k] += n.learningRate * outputErrors[k] * hidden[j]
		}
	}

	// 更新输入层到隐藏层权重
	for i := 0; i < n.inputSize; i++ {
		for j := 0; j < n.hiddenSize; j++ {
			n.weights.InputHidden[i][j] += n.learningRate * hiddenErrors[j] * inputs[i]
		}
	}

	// 更新偏置
	for k := 0; k < n.outputSize; k++ {
		n.weights.OutputBias[k] += n.learningRate * outputErrors[k]
	}
	for j := 0; j < n.hiddenSize; j++ {
		n.weights.HiddenBias[j] += n.learningRate * hiddenErrors[j]
	}
}

// ============================================
// 第三部分：在线学习引擎（任务3.4）
// ============================================

// OnlineLearningEngine 在线学习引擎
type OnlineLearningEngine struct {
	trainingBuffer   []TrainingSample
	bufferSize       int
	updateFrequency  int
	modelVersion     int
	learningRate     float64
	mu               sync.RWMutex
}

// TrainingSample 训练样本
type TrainingSample struct {
	Features       []float64
	Label         float64
	Timestamp     time.Time
	Confidence    float64
	Source        string
}

// NewOnlineLearningEngine 创建在线学习引擎
func NewOnlineLearningEngine() *OnlineLearningEngine {
	return &OnlineLearningEngine{
		trainingBuffer:  make([]TrainingSample, 0),
		bufferSize:      1000,
		updateFrequency: 100,
		modelVersion:    1,
		learningRate:    0.01,
	}
}

// AddSample 添加训练样本
func (o *OnlineLearningEngine) AddSample(features []float64, label float64, confidence float64, source string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	sample := TrainingSample{
		Features:    features,
		Label:       label,
		Timestamp:   time.Now(),
		Confidence:  confidence,
		Source:      source,
	}

	o.trainingBuffer = append(o.trainingBuffer, sample)

	// 保持缓冲区大小
	if len(o.trainingBuffer) > o.bufferSize {
		o.trainingBuffer = o.trainingBuffer[len(o.trainingBuffer)-o.bufferSize:]
	}

	// 检查是否需要更新模型
	if len(o.trainingBuffer)%o.updateFrequency == 0 {
		go o.triggerModelUpdate()
	}
}

// triggerModelUpdate 触发模型更新
func (o *OnlineLearningEngine) triggerModelUpdate() {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 清理过期样本（超过24小时的样本）
	cutoff := time.Now().Add(-24 * time.Hour)
	validSamples := make([]TrainingSample, 0)
	for _, sample := range o.trainingBuffer {
		if sample.Timestamp.After(cutoff) {
			validSamples = append(validSamples, sample)
		}
	}
	o.trainingBuffer = validSamples

	// 更新模型版本
	o.modelVersion++
}

// GetRecentSamples 获取最近的样本
func (o *OnlineLearningEngine) GetRecentSamples(count int) []TrainingSample {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if count >= len(o.trainingBuffer) {
		return o.trainingBuffer
	}

	return o.trainingBuffer[len(o.trainingBuffer)-count:]
}

// GetModelVersion 获取当前模型版本
func (o *OnlineLearningEngine) GetModelVersion() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.modelVersion
}

// GetBufferSize 获取缓冲区大小
func (o *OnlineLearningEngine) GetBufferSize() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.trainingBuffer)
}

// CalculateStatistics 计算样本统计信息
func (o *OnlineLearningEngine) CalculateStatistics() map[string]interface{} {
	o.mu.RLock()
	defer o.mu.RUnlock()

	stats := make(map[string]interface{})

	if len(o.trainingBuffer) == 0 {
		return stats
	}

	// 计算标签分布
	positiveCount := 0
	negativeCount := 0
	totalConfidence := 0.0

	for _, sample := range o.trainingBuffer {
		if sample.Label > 0.5 {
			positiveCount++
		} else {
			negativeCount++
		}
		totalConfidence += sample.Confidence
	}

	stats["total_samples"] = len(o.trainingBuffer)
	stats["positive_samples"] = positiveCount
	stats["negative_samples"] = negativeCount
	stats["avg_confidence"] = totalConfidence / float64(len(o.trainingBuffer))
	stats["positive_ratio"] = float64(positiveCount) / float64(len(o.trainingBuffer))

	// 计算特征统计
	if len(o.trainingBuffer) > 0 {
		featureMeans := make([]float64, 0)
		featureStds := make([]float64, 0)

		if len(o.trainingBuffer[0].Features) > 0 {
			nFeatures := len(o.trainingBuffer[0].Features)
			featureMeans = make([]float64, nFeatures)
			featureStds = make([]float64, nFeatures)

			// 计算均值
			for _, sample := range o.trainingBuffer {
				for i := 0; i < nFeatures && i < len(sample.Features); i++ {
					featureMeans[i] += sample.Features[i]
				}
			}
			for i := range featureMeans {
				featureMeans[i] /= float64(len(o.trainingBuffer))
			}

			// 计算标准差
			for _, sample := range o.trainingBuffer {
				for i := 0; i < nFeatures && i < len(sample.Features); i++ {
					diff := sample.Features[i] - featureMeans[i]
					featureStds[i] += diff * diff
				}
			}
			for i := range featureStds {
				featureStds[i] = math.Sqrt(featureStds[i] / float64(len(o.trainingBuffer)))
			}

			stats["feature_means"] = featureMeans
			stats["feature_stds"] = featureStds
		}
	}

	return stats
}

// OptimizeLearningRate 自适应优化学习率
func (o *OnlineLearningEngine) OptimizeLearningRate() float64 {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 根据样本数量调整学习率
	if len(o.trainingBuffer) < 100 {
		o.learningRate = 0.1
	} else if len(o.trainingBuffer) < 500 {
		o.learningRate = 0.05
	} else if len(o.trainingBuffer) < 1000 {
		o.learningRate = 0.01
	} else {
		o.learningRate = 0.005
	}

	return o.learningRate
}

// ClearBuffer 清除缓冲区
func (o *OnlineLearningEngine) ClearBuffer() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.trainingBuffer = make([]TrainingSample, 0)
	o.modelVersion++
}

// GetTrainingData 获取训练数据
func (o *OnlineLearningEngine) GetTrainingData() ([][]float64, []float64) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	features := make([][]float64, len(o.trainingBuffer))
	labels := make([]float64, len(o.trainingBuffer))

	for i, sample := range o.trainingBuffer {
		features[i] = sample.Features
		labels[i] = sample.Label
	}

	return features, labels
}

// CalculateLabelDistribution 计算标签分布
func (o *OnlineLearningEngine) CalculateLabelDistribution() map[string]float64 {
	o.mu.RLock()
	defer o.mu.RUnlock()

	distribution := make(map[string]float64)
	total := float64(len(o.trainingBuffer))

	if total == 0 {
		return distribution
	}

	positiveCount := 0
	negativeCount := 0
	uncertainCount := 0

	for _, sample := range o.trainingBuffer {
		if sample.Label > 0.7 {
			positiveCount++
		} else if sample.Label < 0.3 {
			negativeCount++
		} else {
			uncertainCount++
		}
	}

	distribution["positive"] = float64(positiveCount) / total
	distribution["negative"] = float64(negativeCount) / total
	distribution["uncertain"] = float64(uncertainCount) / total

	return distribution
}
