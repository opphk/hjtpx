package config

import (
	"fmt"
	"os"
)

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Database DatabaseConfig
	Alert    AlertConfig
}

type DatabaseConfig struct {
	SlowQueryThresholdMs int                     `yaml:"slow_query_threshold_ms"`
	QueryOptimization    QueryOptimizationConfig `yaml:"query_optimization"`
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
	Enabled              bool           `yaml:"enabled"`
	Master               DatabaseNode   `yaml:"master"`
	Slaves               []DatabaseNode `yaml:"slaves"`
	LoadBalanceStrategy  string         `yaml:"load_balance_strategy"`
	AutoFailover         bool           `yaml:"auto_failover"`
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
	MaxOpenConns        int  `yaml:"max_open_conns"`
	MaxIdleConns        int  `yaml:"max_idle_conns"`
	ConnMaxLifetimeSecs int  `yaml:"conn_max_lifetime_secs"`
	ConnMaxIdleTimeSecs int  `yaml:"conn_max_idle_time_secs"`
	HealthCheckInterval int  `yaml:"health_check_interval_secs"`
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
	Enabled          bool `yaml:"enabled"`
	DefaultTimeout   int  `yaml:"default_timeout_secs"`
	MaxAlertCount    int  `yaml:"max_alert_count"`
	SlackEnabled     bool `yaml:"slack_enabled"`
	WebhookEnabled   bool `yaml:"webhook_enabled"`
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
				MaxOpenConns:        getEnvAsInt("DB_POOL_MAX_OPEN", 100),
				MaxIdleConns:        getEnvAsInt("DB_POOL_MAX_IDLE", 20),
				ConnMaxLifetimeSecs: getEnvAsInt("DB_POOL_MAX_LIFETIME", 1800),
				ConnMaxIdleTimeSecs: getEnvAsInt("DB_POOL_MAX_IDLE_TIME", 600),
				HealthCheckInterval: getEnvAsInt("DB_POOL_HEALTH_CHECK", 30),
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

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
