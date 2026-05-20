package model

type DifficultyLevel string

const (
	DifficultyEasy    DifficultyLevel = "easy"
	DifficultyMedium  DifficultyLevel = "medium"
	DifficultyHard    DifficultyLevel = "hard"
	DifficultyExpert  DifficultyLevel = "expert"
)

type CaptchaType string

const (
	CaptchaTypeSlider      CaptchaType = "slider"
	CaptchaTypeEmoji       CaptchaType = "emoji"
	CaptchaTypeVoice       CaptchaType = "voice"
	CaptchaType3D          CaptchaType = "3d"
	CaptchaTypeGesture     CaptchaType = "gesture"
	CaptchaTypeMultisensory CaptchaType = "multisensory"
	CaptchaTypeSpatial     CaptchaType = "spatial"
	CaptchaTypeQuantum     CaptchaType = "quantum"
)

type EcosystemStatus string

const (
	EcosystemStatusInitializing EcosystemStatus = "initializing"
	EcosystemStatusActive       EcosystemStatus = "active"
	EcosystemStatusEvolving     EcosystemStatus = "evolving"
	EcosystemStatusOptimizing   EcosystemStatus = "optimizing"
	EcosystemStatusDegraded     EcosystemStatus = "degraded"
)

type UserProfile struct {
	UserID             string                   `json:"user_id"`
	SuccessRate        float64                  `json:"success_rate"`
	AvgResponseTime    float64                  `json:"avg_response_time"`
	AttemptsPerCaptcha float64                  `json:"attempts_per_captcha"`
	PreferredTypes     []CaptchaType            `json:"preferred_types"`
	PreferredDifficulty DifficultyLevel         `json:"preferred_difficulty"`
	LastCaptchaTime    int64                    `json:"last_captcha_time"`
	TotalCaptchas      int                      `json:"total_captchas"`
	SuccessCaptchas    int                      `json:"success_captchas"`
	BehaviorPatterns   []BehaviorPattern        `json:"behavior_patterns"`
	RiskProfile        *RiskProfile             `json:"risk_profile"`
	AdaptationLevel    float64                  `json:"adaptation_level"`
	LearningRate       float64                  `json:"learning_rate"`
	LastUpdated        int64                    `json:"last_updated"`
}

type BehaviorPattern struct {
	PatternID    string                 `json:"pattern_id"`
	PatternType  string                 `json:"pattern_type"`
	Features     map[string]float64     `json:"features"`
	Confidence   float64                `json:"confidence"`
	Frequency    float64                `json:"frequency"`
	FirstSeen    int64                  `json:"first_seen"`
	LastSeen     int64                  `json:"last_seen"`
	SuccessCount int                    `json:"success_count"`
	FailCount    int                    `json:"fail_count"`
}

type RiskProfile struct {
	BaseScore         float64          `json:"base_score"`
	EnvironmentScore  float64          `json:"environment_score"`
	BehaviorScore     float64          `json:"behavior_score"`
	HistoryScore      float64          `json:"history_score"`
	CompositeScore    float64          `json:"composite_score"`
	RiskLevel         string           `json:"risk_level"`
	ThreatIndicators  []string         `json:"threat_indicators"`
	MitigationActions []string         `json:"mitigation_actions"`
	LastAssessed      int64            `json:"last_assessed"`
}

type AttackHistory struct {
	AttackID       string                 `json:"attack_id"`
	AttackType     string                 `json:"attack_type"`
	Timestamp      int64                  `json:"timestamp"`
	TargetCaptcha  CaptchaType            `json:"target_captcha"`
	Method         string                 `json:"method"`
	Success        bool                   `json:"success"`
	DefenseAction  string                 `json:"defense_action"`
	IPAddress      string                 `json:"ip_address"`
	UserAgent      string                 `json:"user_agent"`
	Fingerprint    string                 `json:"fingerprint"`
	Details        map[string]interface{} `json:"details"`
}

type DifficultyAdjustment struct {
	AdjustmentID    string           `json:"adjustment_id"`
	UserID          string           `json:"user_id"`
	PreviousLevel   DifficultyLevel `json:"previous_level"`
	NewLevel        DifficultyLevel `json:"new_level"`
	Reason          string           `json:"reason"`
	SuccessImpact   float64          `json:"success_impact"`
	Timestamp       int64            `json:"timestamp"`
}

type CaptchaConfig struct {
	ConfigID         string           `json:"config_id"`
	CaptchaType      CaptchaType      `json:"captcha_type"`
	DifficultyLevel  DifficultyLevel  `json:"difficulty_level"`
	TimeLimit        int              `json:"time_limit"`
	MaxAttempts      int              `json:"max_attempts"`
	SuccessThreshold float64          `json:"success_threshold"`
	Features         map[string]interface{} `json:"features"`
	Enabled          bool             `json:"enabled"`
}

type EcosystemMetrics struct {
	MetricsID        string                 `json:"metrics_id"`
	Timestamp        int64                  `json:"timestamp"`
	TotalCaptchas    int                    `json:"total_captchas"`
	SuccessRate      float64                `json:"success_rate"`
	AvgResponseTime  float64                `json:"avg_response_time"`
	AttackCount      int                    `json:"attack_count"`
	AttackSuccessRate float64               `json:"attack_success_rate"`
	ActiveUsers      int                    `json:"active_users"`
	ModelVersion     string                 `json:"model_version"`
	HealthScore      float64                `json:"health_score"`
	EvolutionStage   int                    `json:"evolution_stage"`
	OptimizationScore float64              `json:"optimization_score"`
}

type AdaptiveEcosystemRequest struct {
	UserID           string                 `json:"user_id"`
	IPAddress        string                 `json:"ip_address"`
	UserAgent        string                 `json:"user_agent"`
	Fingerprint      string                 `json:"fingerprint"`
	Context          map[string]interface{} `json:"context"`
	PreferredType    CaptchaType            `json:"preferred_type"`
	RequestPurpose   string                 `json:"request_purpose"`
}

type AdaptiveEcosystemResponse struct {
	SessionID        string                 `json:"session_id"`
	CaptchaConfig    *CaptchaConfig         `json:"captcha_config"`
	CaptchaData      interface{}            `json:"captcha_data"`
	AdaptiveHints    []string               `json:"adaptive_hints"`
	RiskAssessment   *RiskAssessment        `json:"risk_assessment"`
	EcosystemStatus  EcosystemStatus        `json:"ecosystem_status"`
	ExpiresIn        int64                  `json:"expires_in"`
	ExpiresAt        int64                  `json:"expires_at"`
	ModelVersion     string                 `json:"model_version"`
}

type RiskAssessment struct {
	AssessmentID   string                 `json:"assessment_id"`
	RiskLevel      string                 `json:"risk_level"`
	RiskScore      float64                `json:"risk_score"`
	ThreatFactors  []ThreatFactor          `json:"threat_factors"`
	Recommendations []string               `json:"recommendations"`
	Confidence     float64                `json:"confidence"`
	ModelUsed      string                 `json:"model_used"`
}

type ThreatFactor struct {
	FactorID     string  `json:"factor_id"`
	FactorName   string  `json:"factor_name"`
	Weight       float64 `json:"weight"`
	Score        float64 `json:"score"`
	Contribution float64 `json:"contribution"`
	Description  string  `json:"description"`
}

type AdaptiveVerifyRequest struct {
	SessionID      string                 `json:"session_id"`
	UserID         string                 `json:"user_id"`
	Answer         interface{}            `json:"answer"`
	ResponseTime   int64                  `json:"response_time"`
	BehaviorData   map[string]interface{} `json:"behavior_data"`
	EnvironmentData map[string]interface{} `json:"environment_data"`
}

type AdaptiveVerifyResponse struct {
	Success         bool                   `json:"success"`
	Score           float64                `json:"score"`
	Message         string                 `json:"message"`
	NextDifficulty  DifficultyLevel        `json:"next_difficulty"`
	LearningUpdate  *LearningUpdate        `json:"learning_update"`
	Feedback        *VerificationFeedback  `json:"feedback"`
}

type LearningUpdate struct {
	UpdateID       string                 `json:"update_id"`
	PatternUpdated string                 `json:"pattern_updated"`
	Changes        map[string]float64     `json:"changes"`
	ConfidenceDelta float64              `json:"confidence_delta"`
	Effectiveness  float64                `json:"effectiveness"`
}

type AdaptiveFeedback struct {
	IsCorrect      bool                   `json:"is_correct"`
	TimeTaken      int64                  `json:"time_taken"`
	DifficultyHit  DifficultyLevel        `json:"difficulty_hit"`
	HintUsed       bool                   `json:"hint_used"`
	UserStruggled  bool                   `json:"user_struggled"`
}

type EcosystemEvolution struct {
	EvolutionID    string           `json:"evolution_id"`
	Stage          int              `json:"stage"`
	PreviousState  *EcosystemMetrics `json:"previous_state"`
	NewState       *EcosystemMetrics `json:"new_state"`
	Changes        []EcosystemChange `json:"changes"`
	PerformanceGain float64         `json:"performance_gain"`
	Timestamp      int64            `json:"timestamp"`
}

type EcosystemChange struct {
	ChangeType   string                 `json:"change_type"`
	Target       string                 `json:"target"`
	OldValue     interface{}            `json:"old_value"`
	NewValue     interface{}            `json:"new_value"`
	Reason       string                 `json:"reason"`
}

type SelfOptimizationRequest struct {
	OptimizationGoal string                 `json:"optimization_goal"`
	MetricsSnapshot  *EcosystemMetrics      `json:"metrics_snapshot"`
	Constraints      map[string]interface{} `json:"constraints"`
}

type SelfOptimizationResponse struct {
	OptimizationID  string                 `json:"optimization_id"`
	RecommendedChanges []EcosystemChange  `json:"recommended_changes"`
	PredictedImpact float64               `json:"predicted_impact"`
	Confidence      float64               `json:"confidence"`
	ImplementationPlan []string           `json:"implementation_plan"`
}

type UserLearningData struct {
	UserID           string                   `json:"user_id"`
	SessionHistory   []LearningSession        `json:"session_history"`
	PerformanceTrend []PerformanceDataPoint  `json:"performance_trend"`
	AdaptationRate   float64                  `json:"adaptation_rate"`
	CurrentLevel     int                      `json:"current_level"`
}

type LearningSession struct {
	SessionID   string           `json:"session_id"`
	Timestamp   int64            `json:"timestamp"`
	CaptchaType CaptchaType      `json:"captcha_type"`
	Difficulty DifficultyLevel  `json:"difficulty"`
	Success    bool             `json:"success"`
	TimeTaken  int64             `json:"time_taken"`
	Attempts   int               `json:"attempts"`
}

type PerformanceDataPoint struct {
	Timestamp     int64   `json:"timestamp"`
	SuccessRate   float64 `json:"success_rate"`
	AvgTimeTaken  float64 `json:"avg_time_taken"`
	DifficultyAvg float64 `json:"difficulty_avg"`
}

type AttackDefensePolicy struct {
	PolicyID       string           `json:"policy_id"`
	AttackType     string           `json:"attack_type"`
	Severity       string           `json:"severity"`
	DefenseAction  string           `json:"defense_action"`
	Threshold      float64          `json:"threshold"`
	CooldownPeriod  int              `json:"cooldown_period"`
	IsActive       bool             `json:"is_active"`
}
