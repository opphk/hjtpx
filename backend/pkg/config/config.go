package config

import (
	"fmt"
	"os"
)

type Config struct {
	Server    ServerConfig
	Postgres  PostgresConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Database  DatabaseConfig
	Alert     AlertConfig
	I18n      I18nConfig
	Backup    BackupConfig
	Edge      EdgeConfig
}

type DatabaseConfig struct {
	SlowQueryThresholdMs int                       `yaml:"slow_query_threshold_ms"`
	QueryOptimization    QueryOptimizationConfig   `yaml:"query_optimization"`
	ReadWriteSeparation  ReadWriteSeparationConfig `yaml:"read_write_separation"`
	ConnectionPool       ConnectionPoolConfig      `yaml:"connection_pool"`
	DataArchiving        DataArchivingConfig       `yaml:"data_archiving"`
	IndexOptimization    IndexOptimizationConfig   `yaml:"index_optimization"`
	Monitoring           MonitoringConfig          `yaml:"monitoring"`
}

type QueryOptimizationConfig struct {
	EnablePreparedStatements bool `yaml:"enable_prepared_statements"`
	EnableQueryCache         bool `yaml:"enable_query_cache"`
	QueryCacheTTLSecs        int  `yaml:"query_cache_ttl_secs"`
	MaxQueryCacheSize        int  `yaml:"max_query_cache_size"`
}

type ReadWriteSeparationConfig struct {
	Enabled             bool           `yaml:"enabled"`
	Master              DatabaseNode   `yaml:"master"`
	Slaves              []DatabaseNode `yaml:"slaves"`
	LoadBalanceStrategy string         `yaml:"load_balance_strategy"`
	AutoFailover        bool           `yaml:"auto_failover"`
}

type DatabaseNode struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"db_name"`
	SSLMode  string `yaml:"sslmode"`
	Weight   int    `yaml:"weight"`
}

type ConnectionPoolConfig struct {
	MaxOpenConns        int `yaml:"max_open_conns"`
	MaxIdleConns        int `yaml:"max_idle_conns"`
	ConnMaxLifetimeSecs int `yaml:"conn_max_lifetime_secs"`
	ConnMaxIdleTimeSecs int `yaml:"conn_max_idle_time_secs"`
	HealthCheckInterval int `yaml:"health_check_interval_secs"`
	MinIdleConns        int `yaml:"min_idle_conns"`
	WaitTimeoutSecs     int `yaml:"wait_timeout_secs"`
	MaxWaitCount        int `yaml:"max_wait_count"`
	EnableWarmup        bool `yaml:"enable_warmup"`
	WarmupConns         int `yaml:"warmup_conns"`
	EnableAutoTuning    bool `yaml:"enable_auto_tuning"`
	HighLoadThreshold   int  `yaml:"high_load_threshold"`
	LowLoadThreshold    int  `yaml:"low_load_threshold"`
}

type DataArchivingConfig struct {
	Enabled              bool   `yaml:"enabled"`
	ArchiveThresholdDays int    `yaml:"archive_threshold_days"`
	ArchiveTablePrefix   string `yaml:"archive_table_prefix"`
	AutoCleanupEnabled   bool   `yaml:"auto_cleanup_enabled"`
	CleanupThresholdDays int    `yaml:"cleanup_threshold_days"`
}

type IndexOptimizationConfig struct {
	AutoAnalyzeEnabled       bool `yaml:"auto_analyze_enabled"`
	AutoAnalyzeIntervalHours int  `yaml:"auto_analyze_interval_hours"`
}

type MonitoringConfig struct {
	EnableQueryMetrics      bool `yaml:"enable_query_metrics"`
	EnableConnectionMetrics bool `yaml:"enable_connection_metrics"`
	MetricsIntervalSecs     int  `yaml:"metrics_interval_secs"`
}

type AlertConfig struct {
	Enabled        bool `yaml:"enabled"`
	DefaultTimeout int  `yaml:"default_timeout_secs"`
	MaxAlertCount  int  `yaml:"max_alert_count"`
	SlackEnabled   bool `yaml:"slack_enabled"`
	WebhookEnabled bool `yaml:"webhook_enabled"`
}

type I18nConfig struct {
	DefaultLang        string   `yaml:"default_lang"`
	SupportedLangs     []string `yaml:"supported_langs"`
	TranslationsDir    string   `yaml:"translations_dir"`
	DefaultTimezone    string   `yaml:"default_timezone"`
	SupportedTimezones []string `yaml:"supported_timezones"`
}

type BackupConfig struct {
	Enabled                 bool   `yaml:"enabled"`
	BackupDir               string `yaml:"backup_dir"`
	AutoBackupEnabled       bool   `yaml:"auto_backup_enabled"`
	AutoBackupIntervalHours int    `yaml:"auto_backup_interval_hours"`
	IncrementalEnabled      bool   `yaml:"incremental_enabled"`
	IncrementalIntervalMins int    `yaml:"incremental_interval_mins"`
	RemoteBackupEnabled     bool   `yaml:"remote_backup_enabled"`
	RemoteBackupType        string `yaml:"remote_backup_type"`
	RemoteBackupPath        string `yaml:"remote_backup_path"`
	RemoteBackupEndpoint    string `yaml:"remote_backup_endpoint"`
	RemoteBackupAccessKey   string `yaml:"remote_backup_access_key"`
	RemoteBackupSecretKey   string `yaml:"remote_backup_secret_key"`
	RetentionDays           int    `yaml:"retention_days"`
	CompressionEnabled      bool   `yaml:"compression_enabled"`
	EncryptionEnabled       bool   `yaml:"encryption_enabled"`
	EncryptionKey           string `yaml:"encryption_key"`
}

var globalConfig *Config

func GetConfig() *Config {
	if globalConfig == nil {
		globalConfig = LoadConfig()
	}
	return globalConfig
}

type JWTConfig struct {
	Secret      string
	ExpireHours int
}

type ServerConfig struct {
	Port string
	Mode string
}

type PostgresConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int
}

type RedisConfig struct {
	Host       string
	Port       string
	Password   string
	DB         int
	MaxRetries int
}

func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		Postgres: PostgresConfig{
			Host:            getEnv("POSTGRES_HOST", "localhost"),
			Port:            getEnv("POSTGRES_PORT", "5432"),
			User:            getEnv("POSTGRES_USER", "postgres"),
			Password:        getEnv("POSTGRES_PASSWORD", "postgres"),
			DBName:          getEnv("POSTGRES_DB", "verification"),
			SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("POSTGRES_MAX_OPEN_CONNS", 100),
			MaxIdleConns:    getEnvAsInt("POSTGRES_MAX_IDLE_CONNS", 20),
			ConnMaxLifetime: getEnvAsInt("POSTGRES_CONN_MAX_LIFETIME", 5),
		},
		Redis: RedisConfig{
			Host:       getEnv("REDIS_HOST", "localhost"),
			Port:       getEnv("REDIS_PORT", "6379"),
			Password:   getEnv("REDIS_PASSWORD", ""),
			DB:         0,
			MaxRetries: 3,
		},
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			ExpireHours: getEnvAsInt("JWT_EXPIRE_HOURS", 24),
		},
		Database: DatabaseConfig{
			SlowQueryThresholdMs: getEnvAsInt("SLOW_QUERY_THRESHOLD_MS", 50),
			QueryOptimization: QueryOptimizationConfig{
				EnablePreparedStatements: getEnvAsBool("DB_ENABLE_PREPARED_STATEMENTS", true),
				EnableQueryCache:         getEnvAsBool("DB_ENABLE_QUERY_CACHE", true),
				QueryCacheTTLSecs:        getEnvAsInt("DB_QUERY_CACHE_TTL_SECS", 300),
				MaxQueryCacheSize:        getEnvAsInt("DB_MAX_QUERY_CACHE_SIZE", 10000),
			},
			ReadWriteSeparation: ReadWriteSeparationConfig{
				Enabled:             getEnvAsBool("DB_RW_SEPARATION_ENABLED", false),
				LoadBalanceStrategy: getEnv("DB_RW_LOAD_BALANCE_STRATEGY", "round_robin"),
				AutoFailover:        getEnvAsBool("DB_RW_AUTO_FAILOVER", true),
				Master: DatabaseNode{
					Host:     getEnv("DB_MASTER_HOST", "localhost"),
					Port:     getEnv("DB_MASTER_PORT", "5432"),
					User:     getEnv("DB_MASTER_USER", "postgres"),
					Password: getEnv("DB_MASTER_PASSWORD", "postgres"),
					DBName:   getEnv("DB_MASTER_DB_NAME", "verification"),
					SSLMode:  getEnv("DB_MASTER_SSLMODE", "disable"),
					Weight:   100,
				},
			},
			ConnectionPool: ConnectionPoolConfig{
				MaxOpenConns:        getEnvAsInt("DB_POOL_MAX_OPEN", 150),
				MaxIdleConns:        getEnvAsInt("DB_POOL_MAX_IDLE", 50),
				ConnMaxLifetimeSecs: getEnvAsInt("DB_POOL_MAX_LIFETIME", 3600),
				ConnMaxIdleTimeSecs: getEnvAsInt("DB_POOL_MAX_IDLE_TIME", 600),
				HealthCheckInterval: getEnvAsInt("DB_POOL_HEALTH_CHECK", 30),
				MinIdleConns:        getEnvAsInt("DB_POOL_MIN_IDLE", 10),
				WaitTimeoutSecs:     getEnvAsInt("DB_POOL_WAIT_TIMEOUT", 30),
				MaxWaitCount:        getEnvAsInt("DB_POOL_MAX_WAIT", 500),
				EnableWarmup:        getEnvAsBool("DB_POOL_WARMUP_ENABLED", true),
				WarmupConns:         getEnvAsInt("DB_POOL_WARMUP_CONNS", 10),
				EnableAutoTuning:    getEnvAsBool("DB_POOL_AUTO_TUNING", true),
				HighLoadThreshold:   getEnvAsInt("DB_POOL_HIGH_LOAD", 80),
				LowLoadThreshold:    getEnvAsInt("DB_POOL_LOW_LOAD", 20),
			},
			DataArchiving: DataArchivingConfig{
				Enabled:              getEnvAsBool("DB_ARCHIVING_ENABLED", true),
				ArchiveThresholdDays: getEnvAsInt("DB_ARCHIVE_THRESHOLD_DAYS", 30),
				ArchiveTablePrefix:   getEnv("DB_ARCHIVE_TABLE_PREFIX", "archive_"),
				AutoCleanupEnabled:   getEnvAsBool("DB_AUTO_CLEANUP_ENABLED", true),
				CleanupThresholdDays: getEnvAsInt("DB_CLEANUP_THRESHOLD_DAYS", 365),
			},
			IndexOptimization: IndexOptimizationConfig{
				AutoAnalyzeEnabled:       getEnvAsBool("DB_AUTO_ANALYZE_ENABLED", true),
				AutoAnalyzeIntervalHours: getEnvAsInt("DB_AUTO_ANALYZE_INTERVAL", 24),
			},
			Monitoring: MonitoringConfig{
				EnableQueryMetrics:      getEnvAsBool("DB_MONITOR_QUERIES", true),
				EnableConnectionMetrics: getEnvAsBool("DB_MONITOR_CONNECTIONS", true),
				MetricsIntervalSecs:     getEnvAsInt("DB_METRICS_INTERVAL", 60),
			},
		},
		Alert: AlertConfig{
			Enabled:        getEnvAsBool("ALERT_ENABLED", true),
			DefaultTimeout: getEnvAsInt("ALERT_DEFAULT_TIMEOUT", 300),
			MaxAlertCount:  getEnvAsInt("ALERT_MAX_COUNT", 1000),
			SlackEnabled:   getEnvAsBool("ALERT_SLACK_ENABLED", true),
			WebhookEnabled: getEnvAsBool("ALERT_WEBHOOK_ENABLED", true),
		},
		I18n: I18nConfig{
			DefaultLang:        getEnv("DEFAULT_LANG", "zh-CN"),
			SupportedLangs:     []string{"zh-CN", "en-US", "ja-JP", "ko-KR", "fr-FR", "de-DE", "es-ES", "pt-BR", "it-IT", "ru-RU", "ar-SA"},
			TranslationsDir:    getEnv("TRANSLATIONS_DIR", "translations"),
			DefaultTimezone:    getEnv("DEFAULT_TIMEZONE", "Asia/Shanghai"),
			SupportedTimezones: []string{"Asia/Shanghai", "America/New_York", "America/Los_Angeles", "Europe/London", "Europe/Paris", "Europe/Berlin", "Asia/Tokyo", "Asia/Seoul", "Australia/Sydney", "Pacific/Auckland"},
		},
		Backup: BackupConfig{
			Enabled:                 getEnvAsBool("BACKUP_ENABLED", true),
			BackupDir:               getEnv("BACKUP_DIR", "./backups"),
			AutoBackupEnabled:       getEnvAsBool("BACKUP_AUTO_ENABLED", true),
			AutoBackupIntervalHours: getEnvAsInt("BACKUP_AUTO_INTERVAL_HOURS", 24),
			IncrementalEnabled:      getEnvAsBool("BACKUP_INCREMENTAL_ENABLED", true),
			IncrementalIntervalMins: getEnvAsInt("BACKUP_INCREMENTAL_INTERVAL_MINS", 60),
			RemoteBackupEnabled:     getEnvAsBool("BACKUP_REMOTE_ENABLED", false),
			RemoteBackupType:        getEnv("BACKUP_REMOTE_TYPE", "s3"),
			RemoteBackupPath:        getEnv("BACKUP_REMOTE_PATH", ""),
			RemoteBackupEndpoint:    getEnv("BACKUP_REMOTE_ENDPOINT", ""),
			RemoteBackupAccessKey:   getEnv("BACKUP_REMOTE_ACCESS_KEY", ""),
			RemoteBackupSecretKey:   getEnv("BACKUP_REMOTE_SECRET_KEY", ""),
			RetentionDays:           getEnvAsInt("BACKUP_RETENTION_DAYS", 30),
			CompressionEnabled:      getEnvAsBool("BACKUP_COMPRESSION_ENABLED", true),
			EncryptionEnabled:       getEnvAsBool("BACKUP_ENCRYPTION_ENABLED", false),
			EncryptionKey:           getEnv("BACKUP_ENCRYPTION_KEY", ""),
		},
		Edge: EdgeConfig{
			Enabled:                 getEnvAsBool("EDGE_ENABLED", false),
			NodeID:                  getEnv("EDGE_NODE_ID", "edge-node-001"),
			NodeName:                getEnv("EDGE_NODE_NAME", "Edge Node 001"),
			NodeType:                getEnv("EDGE_NODE_TYPE", "edge"),
			Region:                  getEnv("EDGE_REGION", "cn-east-1"),
			Zone:                    getEnv("EDGE_ZONE", "zone-a"),
			LoadBalanceStrategy:     getEnv("EDGE_LOAD_BALANCE_STRATEGY", "least_load"),
			SyncIntervalSecs:        getEnvAsInt("EDGE_SYNC_INTERVAL_SECS", 60),
			HealthCheckIntervalSecs: getEnvAsInt("EDGE_HEALTH_CHECK_INTERVAL_SECS", 30),
			HeartbeatIntervalSecs:   getEnvAsInt("EDGE_HEARTBEAT_INTERVAL_SECS", 10),
			MaxSyncBatchSize:        getEnvAsInt("EDGE_MAX_SYNC_BATCH_SIZE", 1000),
			CloudEndpoint:           getEnv("EDGE_CLOUD_ENDPOINT", "https://api.example.com"),
			CloudAPIKey:             getEnv("EDGE_CLOUD_API_KEY", ""),
			LocalCacheTTLMinutes:    getEnvAsInt("EDGE_LOCAL_CACHE_TTL_MINUTES", 60),
			EnableLocalVerification: getEnvAsBool("EDGE_ENABLE_LOCAL_VERIFICATION", true),
			MetricsEnabled:          getEnvAsBool("EDGE_METRICS_ENABLED", true),
			Capacity: EdgeCapacity{
				MaxRequestsPerSecond:   getEnvAsInt("EDGE_CAPACITY_MAX_RPS", 10000),
				MaxConcurrentRequests:  getEnvAsInt("EDGE_CAPACITY_MAX_CONCURRENT", 1000),
				MemoryLimitMB:          getEnvAsInt("EDGE_CAPACITY_MEMORY_LIMIT_MB", 4096),
				CPUCores:               getEnvAsInt("EDGE_CAPACITY_CPU_CORES", 4),
			},
		},
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
		return value == "true" || value == "1"
	}
	return defaultValue
}

type EdgeConfig struct {
	Enabled                bool              `yaml:"enabled"`
	NodeID                 string            `yaml:"node_id"`
	NodeName               string            `yaml:"node_name"`
	NodeType               string            `yaml:"node_type"`
	Region                 string            `yaml:"region"`
	Zone                   string            `yaml:"zone"`
	Capacity               EdgeCapacity      `yaml:"capacity"`
	LoadBalanceStrategy    string            `yaml:"load_balance_strategy"`
	SyncIntervalSecs       int               `yaml:"sync_interval_secs"`
	HealthCheckIntervalSecs int              `yaml:"health_check_interval_secs"`
	HeartbeatIntervalSecs  int               `yaml:"heartbeat_interval_secs"`
	MaxSyncBatchSize       int               `yaml:"max_sync_batch_size"`
	CloudEndpoint          string            `yaml:"cloud_endpoint"`
	CloudAPIKey            string            `yaml:"cloud_api_key"`
	LocalCacheTTLMinutes   int               `yaml:"local_cache_ttl_minutes"`
	EnableLocalVerification bool             `yaml:"enable_local_verification"`
	MetricsEnabled         bool              `yaml:"metrics_enabled"`
}

type EdgeCapacity struct {
	MaxRequestsPerSecond   int `yaml:"max_requests_per_second"`
	MaxConcurrentRequests  int `yaml:"max_concurrent_requests"`
	MemoryLimitMB          int `yaml:"memory_limit_mb"`
	CPUCores               int `yaml:"cpu_cores"`
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
