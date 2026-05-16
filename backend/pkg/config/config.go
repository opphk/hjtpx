package config

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Postgres   PostgresConfig   `yaml:"postgres"`
	Redis      RedisConfig      `yaml:"redis"`
	JWT        JWTConfig        `yaml:"jwt"`
	Security   SecurityConfig   `yaml:"security"`
	CSRF       CSRFConfig       `yaml:"csrf"`
	XSS        XSSConfig        `yaml:"xss"`
	SQLInjection SQLInjectionConfig `yaml:"sql_injection"`
	IPRisk     IPRiskConfig     `yaml:"ip_risk"`
	CORS       CORSConfig       `yaml:"cors"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
	Logging    LoggingConfig    `yaml:"logging"`
	SecurityLog SecurityLogConfig `yaml:"security_log"`
	Cache      CacheConfig      `yaml:"cache"`
	AsyncTasks AsyncTasksConfig `yaml:"async_tasks"`
	Blacklist BlacklistConfig  `yaml:"blacklist"`
	Whitelist WhitelistConfig `yaml:"whitelist"`
	Database   DatabaseConfig   `yaml:"database"`
	Metrics    MetricsConfig    `yaml:"metrics"`
}

type ServerConfig struct {
	Port           string `yaml:"port"`
	Mode           string `yaml:"mode"`
	ReadTimeout    int    `yaml:"read_timeout"`
	WriteTimeout   int    `yaml:"write_timeout"`
	IdleTimeout    int    `yaml:"idle_timeout"`
	MaxHeaderBytes int    `yaml:"max_header_bytes"`
}

type PostgresConfig struct {
	Host              string `yaml:"host"`
	Port              string `yaml:"port"`
	User              string `yaml:"user"`
	Password          string `yaml:"password"`
	DBName            string `yaml:"db_name"`
	SSLMode           string `yaml:"sslmode"`
	MaxOpenConns      int    `yaml:"max_open_conns"`
	MaxIdleConns      int    `yaml:"max_idle_conns"`
	ConnMaxLifetime   int    `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime   int    `yaml:"conn_max_idle_time"`
	WaitForPool       bool   `yaml:"wait_for_pool"`
}

type RedisConfig struct {
	Host           string `yaml:"host"`
	Port           string `yaml:"port"`
	Password       string `yaml:"password"`
	DB             int    `yaml:"db"`
	MaxRetries     int    `yaml:"max_retries"`
	PoolSize       int    `yaml:"pool_size"`
	MinIdleConns   int    `yaml:"min_idle_conns"`
	ReadTimeout    int    `yaml:"read_timeout"`
	WriteTimeout   int    `yaml:"write_timeout"`
}

type JWTConfig struct {
	Secret            string `yaml:"secret"`
	ExpireHours       int    `yaml:"expire_hours"`
	RefreshExpireHours int    `yaml:"refresh_expire_hours"`
	Issuer            string `yaml:"issuer"`
	Audience          string `yaml:"audience"`
}

type SecurityConfig struct {
	EnableCSRF         bool   `yaml:"enable_csrf"`
	EnableXSS          bool   `yaml:"enable_xss"`
	EnableSignature     bool   `yaml:"enable_signature"`
	EnableSQLInjection  bool   `yaml:"enable_sql_injection"`
	EnableIPRisk       bool   `yaml:"enable_ip_risk"`
	SignatureExpireSecs int    `yaml:"signature_expire_secs"`
	SignatureKey       string `yaml:"signature_key"`
	WhitelistIPs       []string `yaml:"whitelist_ips"`
}

type CSRFConfig struct {
	Enable            bool     `yaml:"enable"`
	TokenLength       int      `yaml:"token_length"`
	TokenExpirationHours int   `yaml:"token_expiration_hours"`
	HeaderName        string   `yaml:"header_name"`
	FormFieldName     string   `yaml:"form_field_name"`
	CookieName        string   `yaml:"cookie_name"`
	SafeMethods       []string `yaml:"safe_methods"`
	RequireValidation bool     `yaml:"require_validation"`
}

type XSSConfig struct {
	EnableLog       bool     `yaml:"enable_log"`
	BlockAttributes bool     `yaml:"block_attributes"`
	MaxLength       int      `yaml:"max_length"`
	EnableSafeHTML  bool     `yaml:"enable_safe_html"`
	AllowedTags     []string `yaml:"allowed_tags"`
}

type SQLInjectionConfig struct {
	Enable           bool     `yaml:"enable"`
	BlockMode        bool     `yaml:"block_mode"`
	EnableLog        bool     `yaml:"enable_log"`
	MaxQueryLength   int      `yaml:"max_query_length"`
	SeverityThreshold int     `yaml:"severity_threshold"`
	ExcludePaths     []string `yaml:"exclude_paths"`
	ExcludeParams    []string `yaml:"exclude_params"`
}

type IPRiskConfig struct {
	Enable               bool     `yaml:"enable"`
	EnableProxyDetection bool     `yaml:"enable_proxy_detection"`
	EnableVPNDetection   bool     `yaml:"enable_vpn_detection"`
	EnableTorDetection   bool     `yaml:"enable_tor_detection"`
	EnableHostingDetection bool   `yaml:"enable_hosting_detection"`
	EnableDatacenterCheck bool    `yaml:"enable_datacenter_check"`
	ProxyThreshold       int      `yaml:"proxy_threshold"`
	VPNThreshold         int      `yaml:"vpn_threshold"`
	CheckTimeoutSecs     int      `yaml:"check_timeout_secs"`
	CacheTTLSecs         int      `yaml:"cache_ttl_secs"`
	BlockHighRisk        bool     `yaml:"block_high_risk"`
	BlockCriticalRisk    bool     `yaml:"block_critical_risk"`
	WarnMediumRisk       bool     `yaml:"warn_medium_risk"`
	AllowedCountries     []string `yaml:"allowed_countries"`
	BlockedCountries     []string `yaml:"blocked_countries"`
	AllowedASNs          []string `yaml:"allowed_asns"`
	BlockedASNs          []string `yaml:"blocked_asns"`
	ExcludePaths         []string `yaml:"exclude_paths"`
}

type CORSConfig struct {
	AllowedOrigins      []string `yaml:"allowed_origins"`
	AllowCredentials   bool     `yaml:"allow_credentials"`
	AllowMethods       []string `yaml:"allow_methods"`
	AllowHeaders       []string `yaml:"allow_headers"`
	ExposeHeaders      []string `yaml:"expose_headers"`
	MaxAge             int      `yaml:"max_age"`
	AllowPrivateNetwork bool     `yaml:"allow_private_network"`
}

type RateLimitConfig struct {
	Enabled      bool             `yaml:"enabled"`
	DefaultLimit int              `yaml:"default_limit"`
	WindowSecs   int              `yaml:"window_secs"`
	BurstLimit   int              `yaml:"burst_limit"`
	CleanupIntervalMs int        `yaml:"cleanup_interval_ms"`
	IPLimits     IPLimitConfig    `yaml:"ip_limits"`
	UserLimits   UserLimitConfig  `yaml:"user_limits"`
	AppLimits    AppLimitConfig   `yaml:"app_limits"`
	RiskBased    RiskBasedConfig  `yaml:"risk_based"`
}

type IPLimitConfig struct {
	DefaultLimit int `yaml:"default_limit"`
	WindowSecs   int `yaml:"window_secs"`
	BurstLimit   int `yaml:"burst_limit"`
	Strict       LimitSettings `yaml:"strict"`
	Relaxed      LimitSettings `yaml:"relaxed"`
}

type UserLimitConfig struct {
	DefaultLimit int `yaml:"default_limit"`
	WindowSecs   int `yaml:"window_secs"`
	BurstLimit   int `yaml:"burst_limit"`
	Strict       LimitSettings `yaml:"strict"`
	Relaxed      LimitSettings `yaml:"relaxed"`
}

type AppLimitConfig struct {
	DefaultLimit int `yaml:"default_limit"`
	WindowSecs   int `yaml:"window_secs"`
	BurstLimit   int `yaml:"burst_limit"`
	Strict       LimitSettings `yaml:"strict"`
	Relaxed      LimitSettings `yaml:"relaxed"`
}

type LimitSettings struct {
	Limit      int `yaml:"limit"`
	WindowSecs int `yaml:"window_secs"`
}

type RiskBasedConfig struct {
	CriticalMaxRequests int `yaml:"critical_max_requests"`
	CriticalWindowSecs int `yaml:"critical_window_secs"`
	HighMaxRequests    int `yaml:"high_max_requests"`
	HighWindowSecs     int `yaml:"high_window_secs"`
	MediumMaxRequests   int `yaml:"medium_max_requests"`
	MediumWindowSecs    int `yaml:"medium_window_secs"`
}

type LoggingConfig struct {
	Level       string `yaml:"level"`
	Format      string `yaml:"format"`
	OutputPath  string `yaml:"output_path"`
	MaxSizeMB   int    `yaml:"max_size_mb"`
	MaxBackups  int    `yaml:"max_backups"`
	MaxAgeDays  int    `yaml:"max_age_days"`
	Compress    bool   `yaml:"compress"`
}

type SecurityLogConfig struct {
	EnableConsole     bool     `yaml:"enable_console"`
	EnableFile        bool     `yaml:"enable_file"`
	EnableRedis       bool     `yaml:"enable_redis"`
	LogFilePath       string   `yaml:"log_file_path"`
	MaxFileSizeMB     int      `yaml:"max_file_size_mb"`
	MaxBackupFiles    int      `yaml:"max_backup_files"`
	RotateIntervalHours int    `yaml:"rotate_interval_hours"`
	MinLevel          string   `yaml:"min_level"`
	RetentionDays     int      `yaml:"retention_days"`
	ExcludePaths      []string `yaml:"exclude_paths"`
}

type CacheConfig struct {
	CaptchaResultTTLSecs int  `yaml:"captcha_result_ttl_secs"`
	StatsDataTTLSecs     int  `yaml:"stats_data_ttl_secs"`
	HotDataTTLSecs       int  `yaml:"hot_data_ttl_secs"`
	UserProfileTTLSecs   int  `yaml:"user_profile_ttl_secs"`
	WhitelistTTLSecs     int  `yaml:"whitelist_ttl_secs"`
	BlacklistTTLSecs     int  `yaml:"blacklist_ttl_secs"`
	DefaultTTLSecs       int  `yaml:"default_ttl_secs"`
	EnablePipeline       bool `yaml:"enable_pipeline"`
	EnableCompression     bool `yaml:"enable_compression"`
}

type AsyncTasksConfig struct {
	Enable         bool                    `yaml:"enable"`
	WorkerCount    int                     `yaml:"worker_count"`
	QueueSize     int                     `yaml:"queue_size"`
	MaxRetries    int                     `yaml:"max_retries"`
	DefaultTimeoutSecs int                 `yaml:"default_timeout_secs"`
	TaskTypes      map[string]TaskTypeConfig `yaml:"task_types"`
	PeriodicTasks  PeriodicTasksConfig     `yaml:"periodic_tasks"`
}

type TaskTypeConfig struct {
	Enable      bool `yaml:"enable"`
	TimeoutSecs int  `yaml:"timeout_secs"`
	Priority    int  `yaml:"priority"`
}

type PeriodicTasksConfig struct {
	CleanupExpired PeriodicTaskConfig `yaml:"cleanup_expired"`
	RefreshStats  PeriodicTaskConfig `yaml:"refresh_stats"`
	BackupData    PeriodicTaskConfig `yaml:"backup_data"`
}

type PeriodicTaskConfig struct {
	Enable        bool `yaml:"enable"`
	IntervalHours int  `yaml:"interval_hours"`
	IntervalMinutes int `yaml:"interval_minutes"`
}

type BlacklistConfig struct {
	AutoBanEnabled       bool             `yaml:"auto_ban_enabled"`
	ViolationThreshold   int              `yaml:"violation_threshold"`
	BanType             string           `yaml:"ban_type"`
	CleanupIntervalHours int             `yaml:"cleanup_interval_hours"`
	RetentionDays       int              `yaml:"retention_days"`
	BanDurations        BanDurationsConfig `yaml:"ban_durations"`
}

type BanDurationsConfig struct {
	Level1Count         int `yaml:"level1_count"`
	Level1DurationMins  int `yaml:"level1_duration_mins"`
	Level2Count         int `yaml:"level2_count"`
	Level2DurationMins  int `yaml:"level2_duration_mins"`
	Level3Count         int `yaml:"level3_count"`
	Level3DurationHours int `yaml:"level3_duration_hours"`
	Level4Count         int `yaml:"level4_count"`
	Level4DurationHours int `yaml:"level4_duration_hours"`
	Level5Count         int `yaml:"level5_count"`
	Level5DurationHours int `yaml:"level5_duration_hours"`
}

type WhitelistConfig struct {
	Enable               bool `yaml:"enable"`
	BypassRateLimit      bool `yaml:"bypass_rate_limit"`
	BypassIPRisk         bool `yaml:"bypass_ip_risk"`
	CleanupIntervalHours int  `yaml:"cleanup_interval_hours"`
}

type DatabaseConfig struct {
	EnableQueryLog         bool              `yaml:"enable_query_log"`
	SlowQueryThresholdMs   int               `yaml:"slow_query_threshold_ms"`
	Indexes               IndexesConfig     `yaml:"indexes"`
	QueryOptimization     QueryOptimizationConfig `yaml:"query_optimization"`
}

type IndexesConfig struct {
	VerificationTable  []string               `yaml:"verification_table"`
	ApplicationTable  []string               `yaml:"application_table"`
	UserTable         []string               `yaml:"user_table"`
	BlacklistTable    []string               `yaml:"blacklist_table"`
}

type QueryOptimizationConfig struct {
	EnablePreparedStatements bool `yaml:"enable_prepared_statements"`
	ConnectionPoolSize      int  `yaml:"connection_pool_size"`
	IdleConnectionPoolSize  int  `yaml:"idle_connection_pool_size"`
	MaxConnectionLifetimeMins int `yaml:"max_connection_lifetime_mins"`
	EnableQueryCache        bool `yaml:"enable_query_cache"`
	QueryCacheTTLSecs       int  `yaml:"query_cache_ttl_secs"`
}

type MetricsConfig struct {
	Enable    bool   `yaml:"enable"`
	Path      string `yaml:"path"`
	EnablePprof bool `yaml:"enable_pprof"`
	PprofPath string `yaml:"pprof_path"`
}

var (
	configInstance *Config
	configOnce     sync.Once
	configMu       sync.RWMutex
)

func LoadConfig() *Config {
	configOnce.Do(func() {
		configInstance = &Config{
			Server: ServerConfig{
				Port:           getEnv("SERVER_PORT", "8080"),
				Mode:           getEnv("GIN_MODE", "debug"),
				ReadTimeout:    getEnvAsInt("SERVER_READ_TIMEOUT", 30),
				WriteTimeout:   getEnvAsInt("SERVER_WRITE_TIMEOUT", 30),
				IdleTimeout:    getEnvAsInt("SERVER_IDLE_TIMEOUT", 120),
				MaxHeaderBytes: getEnvAsInt("SERVER_MAX_HEADER_BYTES", 1048576),
			},
			Postgres: PostgresConfig{
				Host:            getEnv("POSTGRES_HOST", "localhost"),
				Port:            getEnv("POSTGRES_PORT", "5432"),
				User:            getEnv("POSTGRES_USER", "postgres"),
				Password:        getEnv("POSTGRES_PASSWORD", "postgres"),
				DBName:          getEnv("POSTGRES_DB", "app_db"),
				SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
				MaxOpenConns:    getEnvAsInt("POSTGRES_MAX_OPEN_CONNS", 25),
				MaxIdleConns:    getEnvAsInt("POSTGRES_MAX_IDLE_CONNS", 10),
				ConnMaxLifetime: getEnvAsInt("POSTGRES_CONN_MAX_LIFETIME", 300),
				ConnMaxIdleTime: getEnvAsInt("POSTGRES_CONN_MAX_IDLE_TIME", 60),
				WaitForPool:     getEnvAsBool("POSTGRES_WAIT_FOR_POOL", true),
			},
			Redis: RedisConfig{
				Host:         getEnv("REDIS_HOST", "localhost"),
				Port:         getEnv("REDIS_PORT", "6379"),
				Password:     getEnv("REDIS_PASSWORD", ""),
				DB:           getEnvAsInt("REDIS_DB", 0),
				MaxRetries:   getEnvAsInt("REDIS_MAX_RETRIES", 3),
				PoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 20),
				MinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 5),
				ReadTimeout:  getEnvAsInt("REDIS_READ_TIMEOUT", 3),
				WriteTimeout: getEnvAsInt("REDIS_WRITE_TIMEOUT", 3),
			},
			JWT: JWTConfig{
				Secret:            getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
				ExpireHours:       getEnvAsInt("JWT_EXPIRE_HOURS", 24),
				RefreshExpireHours: getEnvAsInt("JWT_REFRESH_EXPIRE_HOURS", 168),
				Issuer:            getEnv("JWT_ISSUER", "hjtpx"),
				Audience:          getEnv("JWT_AUDIENCE", "hjtpx-api"),
			},
			Security: SecurityConfig{
				EnableCSRF:         getEnvAsBool("SECURITY_ENABLE_CSRF", true),
				EnableXSS:          getEnvAsBool("SECURITY_ENABLE_XSS", true),
				EnableSignature:    getEnvAsBool("SECURITY_ENABLE_SIGNATURE", true),
				EnableSQLInjection:  getEnvAsBool("SECURITY_ENABLE_SQL_INJECTION", true),
				EnableIPRisk:       getEnvAsBool("SECURITY_ENABLE_IP_RISK", true),
				SignatureExpireSecs: getEnvAsInt("SIGNATURE_EXPIRE_SECS", 300),
				SignatureKey:       getEnv("SIGNATURE_KEY", ""),
			},
			CSRF: CSRFConfig{
				Enable:              getEnvAsBool("CSRF_ENABLE", true),
				TokenLength:         getEnvAsInt("CSRF_TOKEN_LENGTH", 32),
				TokenExpirationHours: getEnvAsInt("CSRF_TOKEN_EXPIRATION_HOURS", 1),
				HeaderName:         getEnv("CSRF_HEADER_NAME", "X-CSRF-Token"),
				FormFieldName:      getEnv("CSRF_FORM_FIELD_NAME", "csrf_token"),
				CookieName:         getEnv("CSRF_COOKIE_NAME", "csrf_token"),
				SafeMethods:        getEnvAsSlice("CSRF_SAFE_METHODS", "GET,HEAD,OPTIONS", ","),
				RequireValidation:  getEnvAsBool("CSRF_REQUIRE_VALIDATION", true),
			},
			XSS: XSSConfig{
				EnableLog:       getEnvAsBool("XSS_ENABLE_LOG", true),
				BlockAttributes: getEnvAsBool("XSS_BLOCK_ATTRIBUTES", false),
				MaxLength:       getEnvAsInt("XSS_MAX_LENGTH", 10000),
				EnableSafeHTML:  getEnvAsBool("XSS_ENABLE_SAFE_HTML", true),
				AllowedTags:     getEnvAsSlice("XSS_ALLOWED_TAGS", "p,br,strong,em,u,h1,h2,h3,h4,h5,h6,ul,ol,li,a,img", ","),
			},
			SQLInjection: SQLInjectionConfig{
				Enable:           getEnvAsBool("SQL_INJECTION_ENABLE", true),
				BlockMode:        getEnvAsBool("SQL_INJECTION_BLOCK_MODE", true),
				EnableLog:        getEnvAsBool("SQL_INJECTION_ENABLE_LOG", true),
				MaxQueryLength:   getEnvAsInt("SQL_INJECTION_MAX_QUERY_LENGTH", 10000),
				SeverityThreshold: getEnvAsInt("SQL_INJECTION_SEVERITY_THRESHOLD", 3),
				ExcludePaths:     getEnvAsSlice("SQL_INJECTION_EXCLUDE_PATHS", "/health,/api/health", ","),
			},
			IPRisk: IPRiskConfig{
				Enable:                 getEnvAsBool("IP_RISK_ENABLE", true),
				EnableProxyDetection:   getEnvAsBool("IP_RISK_ENABLE_PROXY_DETECTION", true),
				EnableVPNDetection:     getEnvAsBool("IP_RISK_ENABLE_VPN_DETECTION", true),
				EnableTorDetection:     getEnvAsBool("IP_RISK_ENABLE_TOR_DETECTION", true),
				EnableHostingDetection: getEnvAsBool("IP_RISK_ENABLE_HOSTING_DETECTION", true),
				EnableDatacenterCheck: getEnvAsBool("IP_RISK_ENABLE_DATACENTER_CHECK", true),
				ProxyThreshold:         getEnvAsInt("IP_RISK_PROXY_THRESHOLD", 50),
				VPNThreshold:           getEnvAsInt("IP_RISK_VPN_THRESHOLD", 30),
				CheckTimeoutSecs:       getEnvAsInt("IP_RISK_CHECK_TIMEOUT_SECS", 5),
				CacheTTLSecs:           getEnvAsInt("IP_RISK_CACHE_TTL_SECS", 3600),
				BlockHighRisk:         getEnvAsBool("IP_RISK_BLOCK_HIGH_RISK", false),
				BlockCriticalRisk:     getEnvAsBool("IP_RISK_BLOCK_CRITICAL_RISK", true),
				WarnMediumRisk:        getEnvAsBool("IP_RISK_WARN_MEDIUM_RISK", true),
			},
			CORS: CORSConfig{
				AllowedOrigins:    getEnvAsSlice("CORS_ALLOWED_ORIGINS", "*", ","),
				AllowCredentials:  getEnvAsBool("CORS_ALLOW_CREDENTIALS", true),
				AllowMethods:      getEnvAsSlice("CORS_ALLOW_METHODS", "GET,POST,PUT,DELETE,PATCH,OPTIONS,HEAD", ","),
				AllowHeaders:      getEnvAsSlice("CORS_ALLOW_HEADERS", "Origin,Content-Type,Accept,Authorization,X-CSRF-Token,X-Signature,X-Timestamp,X-Nonce", ","),
				ExposeHeaders:     getEnvAsSlice("CORS_EXPOSE_HEADERS", "Content-Length,X-Request-ID,X-RateLimit-Limit,X-RateLimit-Remaining,X-RateLimit-Reset", ","),
				MaxAge:            getEnvAsInt("CORS_MAX_AGE", 86400),
				AllowPrivateNetwork: getEnvAsBool("CORS_ALLOW_PRIVATE_NETWORK", false),
			},
			RateLimit: RateLimitConfig{
				Enabled:           getEnvAsBool("RATE_LIMIT_ENABLED", true),
				DefaultLimit:      getEnvAsInt("RATE_LIMIT_DEFAULT_LIMIT", 100),
				WindowSecs:        getEnvAsInt("RATE_LIMIT_WINDOW_SECS", 60),
				BurstLimit:        getEnvAsInt("RATE_LIMIT_BURST_LIMIT", 200),
				CleanupIntervalMs: getEnvAsInt("RATE_LIMIT_CLEANUP_INTERVAL_MS", 60000),
				IPLimits: IPLimitConfig{
					DefaultLimit: getEnvAsInt("RATE_LIMIT_IP_DEFAULT", 100),
					WindowSecs:  getEnvAsInt("RATE_LIMIT_IP_WINDOW", 60),
					BurstLimit:  getEnvAsInt("RATE_LIMIT_IP_BURST", 200),
				},
				UserLimits: UserLimitConfig{
					DefaultLimit: getEnvAsInt("RATE_LIMIT_USER_DEFAULT", 200),
					WindowSecs:  getEnvAsInt("RATE_LIMIT_USER_WINDOW", 60),
					BurstLimit:  getEnvAsInt("RATE_LIMIT_USER_BURST", 400),
				},
				AppLimits: AppLimitConfig{
					DefaultLimit: getEnvAsInt("RATE_LIMIT_APP_DEFAULT", 500),
					WindowSecs:  getEnvAsInt("RATE_LIMIT_APP_WINDOW", 60),
					BurstLimit:  getEnvAsInt("RATE_LIMIT_APP_BURST", 1000),
				},
				RiskBased: RiskBasedConfig{
					CriticalMaxRequests: getEnvAsInt("RATE_LIMIT_RISK_CRITICAL_MAX", 5),
					CriticalWindowSecs: getEnvAsInt("RATE_LIMIT_RISK_CRITICAL_WINDOW", 300),
					HighMaxRequests:    getEnvAsInt("RATE_LIMIT_RISK_HIGH_MAX", 20),
					HighWindowSecs:     getEnvAsInt("RATE_LIMIT_RISK_HIGH_WINDOW", 120),
					MediumMaxRequests:  getEnvAsInt("RATE_LIMIT_RISK_MEDIUM_MAX", 50),
					MediumWindowSecs:   getEnvAsInt("RATE_LIMIT_RISK_MEDIUM_WINDOW", 60),
				},
			},
			Logging: LoggingConfig{
				Level:      getEnv("LOG_LEVEL", "info"),
				Format:     getEnv("LOG_FORMAT", "json"),
				OutputPath: getEnv("LOG_OUTPUT_PATH", "/var/log/hjtpx/app.log"),
				MaxSizeMB:  getEnvAsInt("LOG_MAX_SIZE_MB", 100),
				MaxBackups: getEnvAsInt("LOG_MAX_BACKUPS", 7),
				MaxAgeDays: getEnvAsInt("LOG_MAX_AGE_DAYS", 30),
				Compress:   getEnvAsBool("LOG_COMPRESS", true),
			},
			SecurityLog: SecurityLogConfig{
				EnableConsole:      getEnvAsBool("SECURITY_LOG_ENABLE_CONSOLE", true),
				EnableFile:         getEnvAsBool("SECURITY_LOG_ENABLE_FILE", true),
				EnableRedis:        getEnvAsBool("SECURITY_LOG_ENABLE_REDIS", true),
				LogFilePath:        getEnv("SECURITY_LOG_FILE_PATH", "/var/log/hjtpx/security.log"),
				MaxFileSizeMB:      getEnvAsInt("SECURITY_LOG_MAX_FILE_SIZE_MB", 10),
				MaxBackupFiles:     getEnvAsInt("SECURITY_LOG_MAX_BACKUP_FILES", 30),
				RotateIntervalHours: getEnvAsInt("SECURITY_LOG_ROTATE_INTERVAL_HOURS", 24),
				MinLevel:           getEnv("SECURITY_LOG_MIN_LEVEL", "low"),
				RetentionDays:       getEnvAsInt("SECURITY_LOG_RETENTION_DAYS", 90),
			},
			Cache: CacheConfig{
				CaptchaResultTTLSecs: getEnvAsInt("CACHE_CAPTCHA_RESULT_TTL_SECS", 300),
				StatsDataTTLSecs:     getEnvAsInt("CACHE_STATS_DATA_TTL_SECS", 60),
				HotDataTTLSecs:       getEnvAsInt("CACHE_HOT_DATA_TTL_SECS", 1800),
				UserProfileTTLSecs:   getEnvAsInt("CACHE_USER_PROFILE_TTL_SECS", 3600),
				WhitelistTTLSecs:     getEnvAsInt("CACHE_WHITELIST_TTL_SECS", 3600),
				BlacklistTTLSecs:     getEnvAsInt("CACHE_BLACKLIST_TTL_SECS", 86400),
				DefaultTTLSecs:       getEnvAsInt("CACHE_DEFAULT_TTL_SECS", 300),
				EnablePipeline:       getEnvAsBool("CACHE_ENABLE_PIPELINE", true),
				EnableCompression:    getEnvAsBool("CACHE_ENABLE_COMPRESSION", false),
			},
			AsyncTasks: AsyncTasksConfig{
				Enable:             getEnvAsBool("ASYNC_TASKS_ENABLE", true),
				WorkerCount:        getEnvAsInt("ASYNC_TASKS_WORKER_COUNT", 4),
				QueueSize:          getEnvAsInt("ASYNC_TASKS_QUEUE_SIZE", 1000),
				MaxRetries:         getEnvAsInt("ASYNC_TASKS_MAX_RETRIES", 3),
				DefaultTimeoutSecs:  getEnvAsInt("ASYNC_TASKS_DEFAULT_TIMEOUT_SECS", 300),
			},
			Blacklist: BlacklistConfig{
				AutoBanEnabled:       getEnvAsBool("BLACKLIST_AUTO_BAN_ENABLED", true),
				ViolationThreshold:   getEnvAsInt("BLACKLIST_VIOLATION_THRESHOLD", 5),
				BanType:             getEnv("BLACKLIST_BAN_TYPE", "ip"),
				CleanupIntervalHours: getEnvAsInt("BLACKLIST_CLEANUP_INTERVAL_HOURS", 24),
				RetentionDays:       getEnvAsInt("BLACKLIST_RETENTION_DAYS", 90),
				BanDurations: BanDurationsConfig{
					Level1Count:         2,
					Level1DurationMins:  30,
					Level2Count:         3,
					Level2DurationMins:  60,
					Level3Count:         5,
					Level3DurationHours: 2,
					Level4Count:         7,
					Level4DurationHours: 6,
					Level5Count:         10,
					Level5DurationHours: 24,
				},
			},
			Whitelist: WhitelistConfig{
				Enable:               getEnvAsBool("WHITELIST_ENABLE", true),
				BypassRateLimit:      getEnvAsBool("WHITELIST_BYPASS_RATE_LIMIT", true),
				BypassIPRisk:         getEnvAsBool("WHITELIST_BYPASS_IP_RISK", false),
				CleanupIntervalHours: getEnvAsInt("WHITELIST_CLEANUP_INTERVAL_HOURS", 24),
			},
			Database: DatabaseConfig{
				EnableQueryLog:       getEnvAsBool("DATABASE_ENABLE_QUERY_LOG", false),
				SlowQueryThresholdMs: getEnvAsInt("DATABASE_SLOW_QUERY_THRESHOLD_MS", 1000),
				QueryOptimization: QueryOptimizationConfig{
					EnablePreparedStatements:  getEnvAsBool("DB_ENABLE_PREPARED_STATEMENTS", true),
					ConnectionPoolSize:       getEnvAsInt("DB_CONNECTION_POOL_SIZE", 25),
					IdleConnectionPoolSize:   getEnvAsInt("DB_IDLE_CONNECTION_POOL_SIZE", 10),
					MaxConnectionLifetimeMins: getEnvAsInt("DB_MAX_CONNECTION_LIFETIME_MINS", 5),
					EnableQueryCache:          getEnvAsBool("DB_ENABLE_QUERY_CACHE", true),
					QueryCacheTTLSecs:         getEnvAsInt("DB_QUERY_CACHE_TTL_SECS", 60),
				},
			},
			Metrics: MetricsConfig{
				Enable:     getEnvAsBool("METRICS_ENABLE", true),
				Path:       getEnv("METRICS_PATH", "/metrics"),
				EnablePprof: getEnvAsBool("METRICS_ENABLE_PPROF", true),
				PprofPath:  getEnv("METRICS_PPROF_PATH", "/debug/pprof"),
			},
		}
	})

	return configInstance
}

func GetConfig() *Config {
	configMu.RLock()
	defer configMu.RUnlock()
	if configInstance == nil {
		return LoadConfig()
	}
	return configInstance
}

func UpdateConfig(cfg *Config) {
	configMu.Lock()
	defer configMu.Unlock()
	configInstance = cfg
}

func GetCacheConfig() *CacheConfig {
	cfg := GetConfig()
	return &cfg.Cache
}

func GetCacheTTL(name string) time.Duration {
	cfg := GetCacheConfig()
	switch name {
	case "captcha_result":
		return time.Duration(cfg.CaptchaResultTTLSecs) * time.Second
	case "stats_data":
		return time.Duration(cfg.StatsDataTTLSecs) * time.Second
	case "hot_data":
		return time.Duration(cfg.HotDataTTLSecs) * time.Second
	case "user_profile":
		return time.Duration(cfg.UserProfileTTLSecs) * time.Second
	case "whitelist":
		return time.Duration(cfg.WhitelistTTLSecs) * time.Second
	case "blacklist":
		return time.Duration(cfg.BlacklistTTLSecs) * time.Second
	default:
		return time.Duration(cfg.DefaultTTLSecs) * time.Second
	}
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		var result int
		_, err := fmt.Sscanf(value, "%d", &result)
		if err == nil {
			return result
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		result, err := fmt.Sscanf(value, "%t", &defaultValue)
		if result == 1 && err == nil {
			return defaultValue
		}
		if value == "true" || value == "1" || value == "yes" {
			return true
		}
		if value == "false" || value == "0" || value == "no" {
			return false
		}
	}
	return defaultValue
}

func getEnvAsSlice(key, defaultValue, sep string) []string {
	if value := os.Getenv(key); value != "" {
		return splitAndTrim(value, sep)
	}
	return splitAndTrim(defaultValue, sep)
}

func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, part := range split(s, sep) {
		trimmed := trim(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func split(s, sep string) []string {
	if s == "" {
		return nil
	}
	result := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func (c *Config) Validate() error {
	if c.Security.SignatureKey == "" || c.Security.SignatureKey == "change-me-in-production" {
		return fmt.Errorf("SECURITY_SIGNATURE_KEY must be set in production")
	}
	if c.JWT.Secret == "" || c.JWT.Secret == "your-secret-key-change-in-production" || c.JWT.Secret == "change-me-in-production" {
		return fmt.Errorf("JWT_SECRET must be set in production")
	}
	if c.Server.Port == "" {
		return fmt.Errorf("SERVER_PORT is required")
	}
	return nil
}

func (c *Config) IsProductionReady() bool {
	if err := c.Validate(); err != nil {
		return false
	}
	if c.Logging.Level != "info" && c.Logging.Level != "warn" && c.Logging.Level != "error" {
		return false
	}
	if c.Metrics.EnablePprof {
		return false
	}
	return true
}
