package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username           string         `gorm:"size:100;uniqueIndex:idx_users_username;not null" json:"username"`
	Email              string         `gorm:"uniqueIndex:idx_users_email;not null" json:"email"`
	PasswordHash       string         `gorm:"size:255;not null" json:"-"`
	Nickname           string         `gorm:"size:100" json:"nickname"`
	Avatar             string         `gorm:"size:500" json:"avatar"`
	Phone              string         `gorm:"size:20" json:"phone"`
	Bio                string         `gorm:"size:500" json:"bio"`
	IsVerified         bool           `gorm:"default:false;index:idx_users_verified" json:"is_verified"`
	VerifiedAt         *time.Time     `json:"verified_at,omitempty"`
	VerificationToken  string         `gorm:"size:100" json:"-"`
	PasswordResetToken string         `gorm:"size:100" json:"-"`
	PasswordResetAt    *time.Time     `json:"password_reset_at,omitempty"`
	LoginCount         int            `gorm:"default:0" json:"login_count"`
	LastLoginAt        *time.Time     `json:"last_login_at,omitempty"`
	LastLoginIP        string         `gorm:"size:50" json:"last_login_ip"`
	Status             string         `gorm:"size:20;default:active;index:idx_users_status" json:"status"`
	Applications       []Application  `gorm:"foreignKey:UserID" json:"applications,omitempty"`
	Verifications      []Verification `gorm:"foreignKey:UserID" json:"verifications,omitempty"`
}

type Admin struct {
	gorm.Model
	Username     string     `gorm:"size:100;uniqueIndex:idx_admins_username;not null" json:"username"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	Email        string     `gorm:"size:255;uniqueIndex:idx_admins_email" json:"email"`
	Status       string     `gorm:"size:20;default:active;index:idx_admins_status" json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP  string     `gorm:"size:50" json:"last_login_ip"`
	LoginCount   int        `gorm:"default:0" json:"login_count"`
	IsSuperAdmin bool       `gorm:"default:false" json:"is_super_admin"`
}

type AdminLoginLog struct {
	gorm.Model
	AdminID    uint   `gorm:"not null;index:idx_login_log_admin" json:"admin_id"`
	IPAddress  string `gorm:"size:50;index:idx_login_log_ip" json:"ip_address"`
	UserAgent  string `gorm:"size:500" json:"user_agent"`
	Status     string `gorm:"size:20;default:success;index:idx_login_log_status" json:"status"`
	FailReason string `gorm:"size:255" json:"fail_reason,omitempty"`
}

type Application struct {
	gorm.Model
	Name            string          `gorm:"size:255;not null;index:idx_app_name" json:"name"`
	UserID          uint            `gorm:"not null;index:idx_app_user_id;index:idx_app_user_active" json:"user_id"`
	Description     string          `gorm:"type:text" json:"description,omitempty"`
	APIKey          string          `gorm:"size:255;uniqueIndex:idx_app_api_key" json:"api_key"`
	Domain          string          `gorm:"size:255;index:idx_app_domain" json:"domain,omitempty"`
	Website         string          `gorm:"size:255" json:"website,omitempty"`
	IsActive        bool            `gorm:"default:true;index:idx_app_user_active" json:"is_active"`
	Config          string          `gorm:"type:text" json:"config,omitempty"`
	User            User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Verifications   []Verification  `gorm:"foreignKey:ApplicationID" json:"verifications,omitempty"`
	APIKeyHistories []APIKeyHistory `gorm:"foreignKey:ApplicationID" json:"api_key_histories,omitempty"`
}

type APIKeyHistory struct {
	gorm.Model
	ApplicationID uint        `gorm:"not null;index:idx_api_key_app_id;index:idx_api_key_app_changed" json:"application_id"`
	OldAPIKey     string      `gorm:"size:255" json:"old_api_key"`
	NewAPIKey     string      `gorm:"size:255" json:"new_api_key"`
	ChangedAt     time.Time   `gorm:"index:idx_api_key_app_changed" json:"changed_at"`
	Application   Application `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
}

type Verification struct {
	gorm.Model
	ApplicationID *uint          `gorm:"index:idx_verification_app_id;index:idx_verification_app_status;index:idx_verification_app_created" json:"application_id,omitempty"`
	UserID        *uint          `gorm:"index:idx_verification_user_id;index:idx_verification_user_created" json:"user_id,omitempty"`
	SessionID     string         `gorm:"size:100;index:idx_verification_session" json:"session_id"`
	CaptchaType   string         `gorm:"size:50;index:idx_verification_type" json:"captcha_type"`
	Status        string         `gorm:"size:50;not null;default:pending;index:idx_verification_status;index:idx_verification_app_status" json:"status"`
	IPAddress     string         `gorm:"size:50;index:idx_verification_ip" json:"ip_address"`
	UserAgent     string         `gorm:"size:500" json:"user_agent"`
	RiskScore     float64        `gorm:"default:0;index:idx_verification_risk" json:"risk_score"`
	Duration      int64          `gorm:"comment:'验证耗时(毫秒)'" json:"duration"`
	CreatedAt     time.Time      `gorm:"index:idx_verification_created;index:idx_verification_app_created;index:idx_verification_user_created" json:"created_at"`
	BehaviorData  []BehaviorData `gorm:"foreignKey:VerificationID" json:"behavior_data,omitempty"`
	Application   *Application   `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
	User          *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type BehaviorData struct {
	gorm.Model
	VerificationID uint         `gorm:"not null;index:idx_behavior_verification" json:"verification_id"`
	Data           string       `gorm:"type:text" json:"data"`
	DataType       string       `gorm:"size:100" json:"data_type"`
	Timestamp      time.Time    `json:"timestamp"`
	Verification   Verification `gorm:"foreignKey:VerificationID" json:"verification,omitempty"`
}

type Blacklist struct {
	gorm.Model
	Target         string `gorm:"size:255;not null;index:idx_blacklist_target;index:idx_blacklist_target_type" json:"target"`
	Type           string `gorm:"size:50;not null;index:idx_blacklist_type;index:idx_blacklist_target_type;index:idx_blacklist_type_status" json:"type"`
	Source         string `gorm:"size:50;default:manual" json:"source"`
	Reason         string `gorm:"type:text" json:"reason,omitempty"`
	Action         string `gorm:"size:50;default:block" json:"action"`
	Status         string `gorm:"size:50;default:active;index:idx_blacklist_status;index:idx_blacklist_type_status" json:"status"`
	Note           string `gorm:"type:text" json:"note,omitempty"`
	CreatedBy      uint   `gorm:"default:0" json:"created_by"`
	HitCount       int    `gorm:"default:0" json:"hit_count"`
	ApplicationIDs string `gorm:"type:text" json:"application_ids,omitempty"`
	Expiration     string `gorm:"size:50" json:"expiration,omitempty"`
}

type VerificationLog struct {
	gorm.Model
	VerificationID uint         `gorm:"not null;index:idx_verification_log_verification" json:"verification_id"`
	SessionID      string       `gorm:"size:100;index:idx_verification_log_session" json:"session_id"`
	ApplicationID  uint         `gorm:"not null;index:idx_verification_log_app;index:idx_verification_log_app_created" json:"application_id"`
	CaptchaType    string       `gorm:"size:50" json:"captcha_type"`
	Status         string       `gorm:"size:50;not null;index:idx_verification_log_status" json:"status"`
	IPAddress      string       `gorm:"size:50" json:"ip_address"`
	UserAgent      string       `gorm:"size:500" json:"user_agent"`
	RiskScore      float64      `gorm:"default:0" json:"risk_score"`
	AnalysisResult string       `gorm:"type:text" json:"analysis_result"`
	Duration       int64        `gorm:"comment:'验证耗时(毫秒)'" json:"duration"`
	CreatedAt      time.Time    `gorm:"index:idx_verification_log_created;index:idx_verification_log_app_created" json:"created_at"`
	Verification   Verification `gorm:"foreignKey:VerificationID" json:"verification,omitempty"`
	Application    Application  `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
}

type DeviceFingerprint struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Fingerprint   string    `gorm:"uniqueIndex:idx_device_fingerprint;size:64" json:"fingerprint"`
	CanvasHash    string    `gorm:"size:64" json:"canvas_hash"`
	WebGLVendor   string    `gorm:"size:100" json:"webgl_vendor"`
	WebGLRenderer string    `gorm:"size:100" json:"webgl_renderer"`
	UserAgent     string    `gorm:"size:500" json:"user_agent"`
	IPAddress     string    `gorm:"size:45;index:idx_device_ip" json:"ip_address"`
	ScreenInfo    string    `gorm:"size:100" json:"screen_info"`
	Timezone      string    `gorm:"size:100" json:"timezone"`
	Language      string    `gorm:"size:50" json:"language"`
	Fonts         string    `gorm:"size:500" json:"fonts"`
	Plugins       string    `gorm:"size:500" json:"plugins"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `gorm:"index:idx_device_last_seen" json:"last_seen"`
	VisitCount    int       `gorm:"default:1" json:"visit_count"`
	IsBot         bool      `gorm:"default:false;index:idx_device_is_bot" json:"is_bot"`
	RiskLevel     string    `gorm:"size:20;default:low;index:idx_device_risk_level" json:"risk_level"`
	RiskScore     float64   `gorm:"default:0" json:"risk_score"`
	ProxyDetected bool      `gorm:"default:false" json:"proxy_detected"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ABTest struct {
	gorm.Model
	Name          string          `gorm:"size:255;not null;index:idx_abtest_name" json:"name"`
	Description   string          `gorm:"type:text" json:"description"`
	ApplicationID uint            `gorm:"not null;index:idx_abtest_app" json:"application_id"`
	Status        string          `gorm:"size:50;default:draft;index:idx_abtest_status" json:"status"`
	StartDate     *time.Time      `json:"start_date"`
	EndDate       *time.Time      `json:"end_date"`
	TrafficSplit  string          `gorm:"type:text" json:"traffic_split"`
	Config        string          `gorm:"type:text" json:"config"`
	CreatedBy     uint            `json:"created_by"`
	Application   Application     `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
	Variants      []ABTestVariant `gorm:"foreignKey:ABTestID" json:"variants,omitempty"`
}

type ABTestVariant struct {
	gorm.Model
	ABTestID       uint   `gorm:"not null;index:idx_variant_abtest" json:"ab_test_id"`
	Name           string `gorm:"size:255;not null" json:"name"`
	IsControl      bool   `gorm:"default:false" json:"is_control"`
	TrafficPercent int    `gorm:"default:0" json:"traffic_percent"`
	Config         string `gorm:"type:text" json:"config"`
	Description    string `gorm:"type:text" json:"description"`
	ABTest         ABTest `gorm:"foreignKey:ABTestID" json:"ab_test,omitempty"`
}

type ABTestEvent struct {
	gorm.Model
	ABTestID     uint          `gorm:"not null;index:idx_event_abtest" json:"ab_test_id"`
	VariantID    uint          `gorm:"not null;index:idx_event_variant" json:"variant_id"`
	SessionID    string        `gorm:"size:100;index:idx_event_session" json:"session_id"`
	EventName    string        `gorm:"size:100;index:idx_event_name" json:"event_name"`
	EventType    string        `gorm:"size:50;index:idx_event_type" json:"event_type"`
	IsConversion bool          `gorm:"default:false;index:idx_event_conversion" json:"is_conversion"`
	Value        float64       `json:"value"`
	Metadata     string        `gorm:"type:text" json:"metadata"`
	Timestamp    time.Time     `gorm:"index:idx_event_timestamp" json:"timestamp"`
	ABTest       ABTest        `gorm:"foreignKey:ABTestID" json:"ab_test,omitempty"`
	Variant      ABTestVariant `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
}

type ABTestAssignment struct {
	gorm.Model
	ABTestID   uint          `gorm:"not null;index:idx_assign_abtest" json:"ab_test_id"`
	VariantID  uint          `gorm:"not null;index:idx_assign_variant" json:"variant_id"`
	SessionID  string        `gorm:"size:100;index:idx_assign_session;uniqueIndex:idx_assign_abtest_session" json:"session_id"`
	UserID     *uint         `gorm:"index:idx_assign_user" json:"user_id"`
	DeviceID   string        `gorm:"size:64;index:idx_assign_device" json:"device_id"`
	IPAddress  string        `gorm:"size:45" json:"ip_address"`
	AssignedAt time.Time     `gorm:"index:idx_assign_time" json:"assigned_at"`
	ABTest     ABTest        `gorm:"foreignKey:ABTestID" json:"ab_test,omitempty"`
	Variant    ABTestVariant `gorm:"foreignKey:VariantID" json:"variant,omitempty"`
}

type AlertChannel struct {
	gorm.Model
	Name        string `gorm:"size:255;not null;index:idx_alert_channel_name" json:"name"`
	Type        string `gorm:"size:50;not null;index:idx_alert_channel_type" json:"type"`
	Config      string `gorm:"type:text" json:"config"`
	IsEnabled   bool   `gorm:"default:true;index:idx_alert_channel_enabled" json:"is_enabled"`
	Description string `gorm:"type:text" json:"description,omitempty"`
}

type AlertRule struct {
	gorm.Model
	Name              string `gorm:"size:255;not null;index:idx_alert_rule_name" json:"name"`
	EventType         string `gorm:"size:100;not null;index:idx_alert_rule_event" json:"event_type"`
	Condition         string `gorm:"type:text" json:"condition"`
	Severity          string `gorm:"size:20;not null;index:idx_alert_rule_severity" json:"severity"`
	ChannelIDs        string `gorm:"type:text" json:"channel_ids"`
	IsEnabled         bool   `gorm:"default:true;index:idx_alert_rule_enabled" json:"is_enabled"`
	AggregationWindow int    `gorm:"default:300" json:"aggregation_window"`
	Threshold         int    `gorm:"default:1" json:"threshold"`
	Description       string `gorm:"type:text" json:"description,omitempty"`
}

type AlertRecord struct {
	gorm.Model
	RuleID           uint       `gorm:"not null;index:idx_alert_record_rule;index:idx_alert_record_rule_time" json:"rule_id"`
	RuleName         string     `gorm:"size:255" json:"rule_name"`
	EventType        string     `gorm:"size:100;index:idx_alert_record_event;index:idx_alert_record_event_time" json:"event_type"`
	Severity         string     `gorm:"size:20;index:idx_alert_record_severity" json:"severity"`
	Message          string     `gorm:"type:text" json:"message"`
	Context          string     `gorm:"type:text" json:"context"`
	Status           string     `gorm:"size:50;default:triggered;index:idx_alert_record_status" json:"status"`
	AggregationKey   string     `gorm:"size:255;index:idx_alert_record_agg_key" json:"aggregation_key"`
	Count            int        `gorm:"default:1" json:"count"`
	FirstTriggeredAt *time.Time `gorm:"index:idx_alert_record_first_triggered" json:"first_triggered_at"`
	LastTriggeredAt  *time.Time `gorm:"index:idx_alert_record_last_triggered" json:"last_triggered_at"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
	Rule             *AlertRule `gorm:"foreignKey:RuleID" json:"rule,omitempty"`
}

type AlertHistory struct {
	gorm.Model
	AlertID     uint         `gorm:"not null;index:idx_alert_history_alert" json:"alert_id"`
	Action      string       `gorm:"size:50;not null" json:"action"`
	OldStatus   string       `gorm:"size:50" json:"old_status"`
	NewStatus   string       `gorm:"size:50" json:"new_status"`
	Note        string       `gorm:"type:text" json:"note"`
	PerformedBy uint         `json:"performed_by"`
	Alert       *AlertRecord `gorm:"foreignKey:AlertID" json:"alert,omitempty"`
}

// ScheduledExport 定时导出任务模型
type ScheduledExport struct {
	gorm.Model
	Name             string     `gorm:"size:255;not null;index:idx_scheduled_name" json:"name"`
	Description      string     `gorm:"type:text" json:"description,omitempty"`
	CronExpression   string     `gorm:"size:100;not null" json:"cron_expression"`
	ExportFormat     string     `gorm:"size:20;default:xlsx" json:"export_format"`
	ExportType       string     `gorm:"size:50;default:logs" json:"export_type"`
	Filters          string     `gorm:"type:text" json:"filters,omitempty"`          // JSON格式的过滤条件
	EmailRecipients  string     `gorm:"type:text" json:"email_recipients,omitempty"` // 收件人，逗号分隔
	IsEnabled        bool       `gorm:"default:true;index:idx_scheduled_enabled" json:"is_enabled"`
	LastRunAt        *time.Time `json:"last_run_at,omitempty"`
	NextRunAt        *time.Time `json:"next_run_at,omitempty"`
	LastStatus       string     `gorm:"size:20;default:pending" json:"last_status"`
	LastErrorMessage string     `gorm:"type:text" json:"last_error_message,omitempty"`
	CreatedBy        uint       `json:"created_by"`
}

// ExportHistory 导出历史记录模型
type ExportHistory struct {
	gorm.Model
	ScheduledExportID *uint            `gorm:"index:idx_export_history_scheduled" json:"scheduled_export_id,omitempty"`
	Name              string           `gorm:"size:255" json:"name"`
	ExportType        string           `gorm:"size:50" json:"export_type"`
	ExportFormat      string           `gorm:"size:20" json:"export_format"`
	FileSize          int64            `json:"file_size"`
	RecordCount       int              `json:"record_count"`
	FilePath          string           `gorm:"size:500" json:"file_path"`
	Status            string           `gorm:"size:20;default:completed" json:"status"`
	ErrorMessage      string           `gorm:"type:text" json:"error_message,omitempty"`
	TriggeredBy       string           `gorm:"size:100" json:"triggered_by"`
	ScheduledExport   *ScheduledExport `gorm:"foreignKey:ScheduledExportID" json:"scheduled_export,omitempty"`
}

// ReportTemplate 报表模板模型
type ReportTemplate struct {
	gorm.Model
	Name        string `gorm:"size:255;not null;index:idx_template_name" json:"name"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	ReportType  string `gorm:"size:50;not null" json:"report_type"`
	Layout      string `gorm:"type:text" json:"layout,omitempty"`  // 布局配置JSON
	Columns     string `gorm:"type:text" json:"columns,omitempty"` // 列配置JSON
	Filters     string `gorm:"type:text" json:"filters,omitempty"` // 默认过滤条件JSON
	Styles      string `gorm:"type:text" json:"styles,omitempty"`  // 样式配置JSON
	IsPublic    bool   `gorm:"default:false" json:"is_public"`
	CreatedBy   uint   `json:"created_by"`
}

// VisualizationChart 可视化图表配置
type VisualizationChart struct {
	gorm.Model
	Name             string          `gorm:"size:255;not null" json:"name"`
	ChartType        string          `gorm:"size:50;not null" json:"chart_type"`      // line, bar, pie, etc.
	DataConfig       string          `gorm:"type:text" json:"data_config,omitempty"`  // 数据配置JSON
	StyleConfig      string          `gorm:"type:text" json:"style_config,omitempty"` // 样式配置JSON
	ReportTemplateID uint            `gorm:"index:idx_chart_template" json:"report_template_id"`
	ReportTemplate   *ReportTemplate `gorm:"foreignKey:ReportTemplateID" json:"report_template,omitempty"`
}

// SeamlessVerification 无感验证记录
type SeamlessVerification struct {
	gorm.Model
	SessionID         string       `gorm:"size:100;index:idx_seamless_session" json:"session_id"`
	ApplicationID     *uint        `gorm:"index:idx_seamless_app" json:"application_id,omitempty"`
	UserID            *uint        `gorm:"index:idx_seamless_user" json:"user_id,omitempty"`
	DeviceFingerprint string       `gorm:"size:64;index:idx_seamless_fingerprint" json:"device_fingerprint"`
	Decision          string       `gorm:"size:50;not null;index:idx_seamless_decision" json:"decision"` // allow, challenge, block
	RiskScore         float64      `gorm:"default:0" json:"risk_score"`
	Reason            string       `gorm:"type:text" json:"reason,omitempty"`
	IPAddress         string       `gorm:"size:50" json:"ip_address"`
	UserAgent         string       `gorm:"size:500" json:"user_agent"`
	Duration          int64        `gorm:"comment:'验证耗时(毫秒)'" json:"duration"`
	Application       *Application `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
	User              *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TrustedDevice 信任设备记录
type TrustedDevice struct {
	gorm.Model
	UserID            uint       `gorm:"not null;index:idx_trusted_user" json:"user_id"`
	DeviceFingerprint string     `gorm:"size:64;index:idx_trusted_fingerprint" json:"device_fingerprint"`
	DeviceName        string     `gorm:"size:255" json:"device_name,omitempty"`
	IsTrusted         bool       `gorm:"default:true;index:idx_trusted_is_trusted" json:"is_trusted"`
	TrustedAt         *time.Time `json:"trusted_at,omitempty"`
	LastUsedAt        time.Time  `gorm:"index:idx_trusted_last_used" json:"last_used_at"`
	UseCount          int        `gorm:"default:1" json:"use_count"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	User              User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// SeamlessConfig 无感验证配置
type SeamlessConfig struct {
	gorm.Model
	ApplicationID              uint        `gorm:"not null;uniqueIndex:idx_seamless_config_app" json:"application_id"`
	Enabled                    bool        `gorm:"default:true" json:"enabled"`
	AutoTrustAfter             int         `gorm:"default:3" json:"auto_trust_after"`     // 连续验证成功次数后自动信任
	TrustDurationDays          int         `gorm:"default:30" json:"trust_duration_days"` // 信任设备有效期(天)
	TrustExpiryDays            int         `gorm:"default:30" json:"trust_expiry_days"`  // 兼容旧字段
	ChallengeThreshold         float64     `gorm:"default:30" json:"challenge_threshold"` // 风险分高于此值需要挑战
	BlockThreshold             float64     `gorm:"default:70" json:"block_threshold"`     // 风险分高于此值直接阻止
	WhitelistEnabled           bool        `gorm:"default:false" json:"whitelist_enabled"`
	RequireBehaviorAnalysis    bool        `gorm:"default:true" json:"require_behavior_analysis"`
	ForceVerificationThreshold float64     `gorm:"default:50" json:"force_verification_threshold"`
	EnableAutoTrust            bool        `gorm:"default:true" json:"enable_auto_trust"`
	MinTrustLevel              float64     `gorm:"default:70" json:"min_trust_level"`
	MaxDevicePerUser           int         `gorm:"default:5" json:"max_device_per_user"`
	Application                Application `gorm:"foreignKey:ApplicationID" json:"application,omitempty"`
}

type TraceRecord struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SessionID      string    `gorm:"size:100;index:idx_trace_session" json:"session_id"`
	VerificationID *uint     `gorm:"index:idx_trace_verification" json:"verification_id,omitempty"`
	ApplicationID  *uint     `gorm:"index:idx_trace_app" json:"application_id,omitempty"`
	RawData        string    `gorm:"type:text" json:"raw_data"`
	FeaturesData   string    `gorm:"type:text" json:"features_data"`
	ScoreData      string    `gorm:"type:text" json:"score_data"`
	TotalTime      int64     `json:"total_time"`
	TotalScore     float64   `json:"total_score"`
	MoveCount      int       `json:"move_count"`
	AvgSpeed       float64   `json:"avg_speed"`
	MaxSpeed       float64   `json:"max_speed"`
	RiskFactors    string    `gorm:"type:text" json:"risk_factors"`
	IPAddress      string    `gorm:"size:50" json:"ip_address"`
	UserAgent      string    `gorm:"size:500" json:"user_agent"`
	DeviceInfo     string    `gorm:"size:255" json:"device_info"`
	CreatedAt      time.Time `json:"created_at"`
}

func (TraceRecord) TableName() string {
	return "trace_records"
}

type CaptchaSession struct {
	ID            int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID     string     `gorm:"size:100;uniqueIndex;not null" json:"session_id"`
	BackgroundURL string     `gorm:"size:500" json:"background_url"`
	SliderURL     string     `gorm:"size:500" json:"slider_url"`
	GapX          int        `json:"gap_x"`
	GapY          int        `json:"gap_y"`
	Status        string     `gorm:"size:50;default:pending" json:"status"`
	VerifyCount   int        `gorm:"default:0" json:"verify_count"`
	MaxAttempts   int        `gorm:"default:3" json:"max_attempts"`
	RiskScore     float64    `gorm:"default:0" json:"risk_score"`
	TraceScore    float64    `gorm:"default:0" json:"trace_score"`
	EnvScore      float64    `gorm:"default:0" json:"env_score"`
	CreatedAt     time.Time  `json:"created_at"`
	ExpiredAt     time.Time  `json:"expired_at"`
	VerifiedAt    *time.Time `json:"verified_at"`
	ClientIP      string     `gorm:"size:50" json:"client_ip"`
	UserAgent     string     `gorm:"size:500" json:"user_agent"`
	Fingerprint   string     `gorm:"size:255" json:"fingerprint"`
}

func (CaptchaSession) TableName() string {
	return "captcha_sessions"
}

type VoiceCaptchaSession struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID   string     `gorm:"size:100;uniqueIndex;not null" json:"session_id"`
	Code        string     `gorm:"size:20;not null" json:"code"`
	Language    string     `gorm:"size:20;default:zh-CN" json:"language"`
	Status      string     `gorm:"size:50;default:pending" json:"status"`
	VerifyCount int        `gorm:"default:0" json:"verify_count"`
	MaxAttempts int        `gorm:"default:3" json:"max_attempts"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiredAt   time.Time  `json:"expired_at"`
	VerifiedAt  *time.Time `json:"verified_at"`
	ClientIP    string     `gorm:"size:50" json:"client_ip"`
	UserAgent   string     `gorm:"size:500" json:"user_agent"`
	Fingerprint string     `gorm:"size:255" json:"fingerprint"`
}

func (VoiceCaptchaSession) TableName() string {
	return "voice_captcha_sessions"
}

// UserMFA 用户多因素认证配置
type UserMFA struct {
	gorm.Model
	UserID         uint       `gorm:"not null;uniqueIndex:idx_user_mfa_user" json:"user_id"`
	MFAType        string     `gorm:"size:20;not null;index:idx_user_mfa_type" json:"mfa_type"` // totp, sms, email
	Secret         string     `gorm:"size:255" json:"-"`
	Phone          string     `gorm:"size:20" json:"phone,omitempty"`
	Email          string     `gorm:"size:255" json:"email,omitempty"`
	IsEnabled      bool       `gorm:"default:false;index:idx_user_mfa_enabled" json:"is_enabled"`
	BackupCodes    string     `gorm:"type:text" json:"-"` // JSON数组存储备份码
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty"`
	User           User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (UserMFA) TableName() string {
	return "user_mfa"
}

// AdminMFA 管理员多因素认证配置
type AdminMFA struct {
	gorm.Model
	AdminID        uint       `gorm:"not null;uniqueIndex:idx_admin_mfa_admin" json:"admin_id"`
	MFAType        string     `gorm:"size:20;not null;index:idx_admin_mfa_type" json:"mfa_type"` // totp, sms, email
	Secret         string     `gorm:"size:255" json:"-"`
	Phone          string     `gorm:"size:20" json:"phone,omitempty"`
	Email          string     `gorm:"size:255" json:"email,omitempty"`
	IsEnabled      bool       `gorm:"default:false;index:idx_admin_mfa_enabled" json:"is_enabled"`
	BackupCodes    string     `gorm:"type:text" json:"-"` // JSON数组存储备份码
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty"`
	Admin          Admin      `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
}

func (AdminMFA) TableName() string {
	return "admin_mfa"
}

// MFACode 临时验证码记录（用于短信和邮箱）
type MFACode struct {
	gorm.Model
	TargetType  string    `gorm:"size:20;not null;index:idx_mfa_code_target_type" json:"target_type"` // user, admin
	TargetID    uint      `gorm:"not null;index:idx_mfa_code_target" json:"target_id"`
	MFAType     string    `gorm:"size:20;not null" json:"mfa_type"` // sms, email
	Code        string    `gorm:"size:20;not null" json:"code"`
	Destination string    `gorm:"size:255;not null" json:"destination"`
	ExpiresAt   time.Time `gorm:"not null;index:idx_mfa_code_expires" json:"expires_at"`
	IsUsed      bool      `gorm:"default:false" json:"is_used"`
}

func (MFACode) TableName() string {
	return "mfa_codes"
}

// UserConsent 用户同意设置模型
type UserConsent struct {
	gorm.Model
	UserID                 uint      `gorm:"not null;uniqueIndex:idx_consent_user" json:"user_id"`
	ConsentMarketing       bool      `gorm:"default:false" json:"consent_marketing"`      // 营销信息同意
	ConsentAnalytics       bool      `gorm:"default:true" json:"consent_analytics"`       // 分析数据收集同意
	ConsentPersonalization bool      `gorm:"default:true" json:"consent_personalization"` // 个性化同意
	ConsentDataSharing     bool      `gorm:"default:false" json:"consent_data_sharing"`   // 数据共享同意
	ConsentUpdatedAt       time.Time `json:"consent_updated_at"`                          // 同意更新时间
	ConsentIP              string    `gorm:"size:50" json:"consent_ip"`                   // 同意时的IP地址
	ConsentUserAgent       string    `gorm:"size:500" json:"consent_user_agent"`          // 同意时的User Agent
	User                   User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (UserConsent) TableName() string {
	return "user_consents"
}

// DataExportRequest 数据导出请求模型
type DataExportRequest struct {
	gorm.Model
	UserID       uint       `gorm:"not null;index:idx_export_user" json:"user_id"`
	ExportFormat string     `gorm:"size:20;default:json" json:"export_format"`                     // json, csv
	Status       string     `gorm:"size:50;default:pending;index:idx_export_status" json:"status"` // pending, processing, completed, failed
	FilePath     string     `gorm:"size:500" json:"file_path,omitempty"`                           // 导出文件路径
	RequestedAt  time.Time  `json:"requested_at"`                                                  // 请求时间
	CompletedAt  *time.Time `json:"completed_at,omitempty"`                                        // 完成时间
	Error        string     `gorm:"type:text" json:"error,omitempty"`                              // 错误信息
	User         User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (DataExportRequest) TableName() string {
	return "data_export_requests"
}

// DataDeletionRequest 数据删除请求模型
type DataDeletionRequest struct {
	gorm.Model
	UserID      uint       `gorm:"not null;index:idx_deletion_user" json:"user_id"`
	Status      string     `gorm:"size:50;default:pending;index:idx_deletion_status" json:"status"` // pending, processing, completed, rejected
	RequestedAt time.Time  `json:"requested_at"`                                                    // 请求时间
	ProcessedAt *time.Time `json:"processed_at,omitempty"`                                          // 处理时间
	Reason      string     `gorm:"type:text" json:"reason,omitempty"`                               // 删除原因
	AuditLog    string     `gorm:"type:text" json:"audit_log,omitempty"`                            // 审计日志
	ProcessedBy *uint      `json:"processed_by,omitempty"`                                          // 处理人ID
	User        User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (DataDeletionRequest) TableName() string {
	return "data_deletion_requests"
}

// UserDataSnapshot 用户数据快照（用于软删除后保留数据）
type UserDataSnapshot struct {
	gorm.Model
	UserID       uint      `gorm:"not null;uniqueIndex:idx_snapshot_user" json:"user_id"`
	UserData     string    `gorm:"type:text" json:"user_data"` // 用户数据JSON
	DeletedAt    time.Time `json:"deleted_at"`                 // 删除时间
	RetentionEnd time.Time `json:"retention_end"`              // 保留结束时间
}

func (UserDataSnapshot) TableName() string {
	return "user_data_snapshots"
}
