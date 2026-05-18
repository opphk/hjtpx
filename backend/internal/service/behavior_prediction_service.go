package service

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type BehaviorPredictionService struct {
	intentPredictor  *IntentPredictor
	riskAssessor     *ProactiveRiskAssessor
	smartInterceptor *SmartInterceptor
	sequenceAnalyzer *SequenceAnalyzer
	mu               sync.RWMutex
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
	Conditions     []InterceptionRuleCondition
	Action         string
	IsEnabled      bool
	HitCount       int
	LastHit        time.Time
	Effectiveness  float64
}

type InterceptionRuleCondition struct {
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
		Conditions: []InterceptionRuleCondition{
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
		Conditions: []InterceptionRuleCondition{
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

func (si *SmartInterceptor) evaluateCondition(condition InterceptionRuleCondition, req *PredictionRequest, assessment *RiskAssessment) bool {
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
