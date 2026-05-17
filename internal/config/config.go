package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App       AppConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Log       LogConfig
	JWT       JWTConfig
	RateLimit RateLimitConfig
	Signature SignatureConfig
}

type AppConfig struct {
	Host string
	Port string
	Mode string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type LogConfig struct {
	Level      string
	Format     string
	OutputPath string
}

type JWTConfig struct {
	Secret          string
	ExpirationHours int
	Issuer          string
}

type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int
	Enabled           bool
}

type SignatureConfig struct {
	SecretKey                string
	TimestampToleranceSeconds int
	NonceExpirationSeconds   int
	Enabled                  bool
}

func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("env")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetDefault("APP.HOST", "0.0.0.0")
	viper.SetDefault("APP.PORT", "8080")
	viper.SetDefault("APP.MODE", "debug")

	viper.SetDefault("DATABASE.HOST", "localhost")
	viper.SetDefault("DATABASE.PORT", "5432")
	viper.SetDefault("DATABASE.USER", "postgres")
	viper.SetDefault("DATABASE.SSLMODE", "disable")

	viper.SetDefault("REDIS.PORT", "6379")
	viper.SetDefault("REDIS.DB", 0)

	viper.SetDefault("LOG.LEVEL", "debug")
	viper.SetDefault("LOG.FORMAT", "json")
	viper.SetDefault("LOG.OUTPUTPATH", "stdout")

	viper.SetDefault("JWT.SECRET", "your-secret-key-change-in-production")
	viper.SetDefault("JWT.EXPIRATIONHOURS", 24)
	viper.SetDefault("JWT.ISSUER", "hjtpx")

	viper.SetDefault("RATELIMIT.REQUESTSPERMINUTE", 100)
	viper.SetDefault("RATELIMIT.BURSTSIZE", 200)
	viper.SetDefault("RATELIMIT.ENABLED", true)

	viper.SetDefault("SIGNATURE.SECRETKEY", "your-signature-secret-key-change-in-production")
	viper.SetDefault("SIGNATURE.TIMESTAMPTOLERANCESECONDS", 300)
	viper.SetDefault("SIGNATURE.NONCEEXPIRATIONSECONDS", 600)
	viper.SetDefault("SIGNATURE.ENABLED", true)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

func (a *AppConfig) Addr() string {
	return fmt.Sprintf("%s:%s", a.Host, a.Port)
}

func (j *JWTConfig) ExpirationTime() time.Duration {
	return time.Duration(j.ExpirationHours) * time.Hour
}

func (s *SignatureConfig) TimestampTolerance() time.Duration {
	return time.Duration(s.TimestampToleranceSeconds) * time.Second
}

func (s *SignatureConfig) NonceExpiration() time.Duration {
	return time.Duration(s.NonceExpirationSeconds) * time.Second
}
