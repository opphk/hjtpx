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
}

type DatabaseConfig struct {
	SlowQueryThresholdMs int                    `yaml:"slow_query_threshold_ms"`
	QueryOptimization    QueryOptimizationConfig `yaml:"query_optimization"`
}

type QueryOptimizationConfig struct {
	EnablePreparedStatements bool `yaml:"enable_prepared_statements"`
	EnableQueryCache        bool `yaml:"enable_query_cache"`
	QueryCacheTTLSecs       int  `yaml:"query_cache_ttl_secs"`
}

var globalConfig *Config

func GetConfig() *Config {
	if globalConfig == nil {
		globalConfig = LoadConfig()
	}
	return globalConfig
}

type JWTConfig struct {
	Secret     string
	ExpireHours int
}

type ServerConfig struct {
	Port    string
	Mode    string
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
	Host            string
	Port            string
	Password        string
	DB              int
	MaxRetries      int
}

func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:    getEnv("SERVER_PORT", "8080"),
			Mode:    getEnv("GIN_MODE", "debug"),
		},
		Postgres: PostgresConfig{
			Host:            getEnv("POSTGRES_HOST", "localhost"),
			Port:            getEnv("POSTGRES_PORT", "5432"),
			User:            getEnv("POSTGRES_USER", "postgres"),
			Password:        getEnv("POSTGRES_PASSWORD", "postgres"),
			DBName:          getEnv("POSTGRES_DB", "verification"),
			SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
			MaxOpenConns:    25,
			MaxIdleConns:    10,
			ConnMaxLifetime: 5,
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
			SlowQueryThresholdMs: getEnvAsInt("SLOW_QUERY_THRESHOLD_MS", 200),
			QueryOptimization: QueryOptimizationConfig{
				EnablePreparedStatements: true,
				EnableQueryCache:        true,
				QueryCacheTTLSecs:       300,
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

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
