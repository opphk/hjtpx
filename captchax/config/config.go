package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Log      LogConfig      `yaml:"log"`
	Captcha  CaptchaConfig  `yaml:"captcha"`
	Admin    AdminConfig    `yaml:"admin"`
	User     UserConfig     `yaml:"user"`
	Sentry   SentryConfig   `yaml:"sentry"`
	AI       AIConfig       `yaml:"ai_model"`
}

type SentryConfig struct {
	DSN              string  `yaml:"dsn"`
	Environment      string  `yaml:"environment"`
	Release          string  `yaml:"release"`
	TracesSampleRate float64 `yaml:"traces_sample_rate"`
	Debug            bool    `yaml:"debug"`
}

type AdminConfig struct {
	JWTSecret        string `yaml:"jwt_secret"`
	TokenTTLSeconds  int    `yaml:"token_ttl_seconds"`
	CookieName       string `yaml:"cookie_name"`
}

type UserConfig struct {
	JWTSecret       string `yaml:"jwt_secret"`
	TokenTTLSeconds int    `yaml:"token_ttl_seconds"`
}

type ServerConfig struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	Mode   string `yaml:"mode"`
	Redis  RedisConfig `yaml:"redis"`
	Captcha CaptchaConfig `yaml:"captcha"`
}

type DatabaseConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"password"`
	DBName          string `yaml:"dbname"`
	SSLMode         string `yaml:"sslmode"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

type CaptchaConfig struct {
	ExpireMinutes int `yaml:"expire_minutes"`
	MaxAttempts   int `yaml:"max_attempts"`
	Width         int `yaml:"width"`
	Height        int `yaml:"height"`
	Length        int `yaml:"length"`
	SliderSize    int `yaml:"slider_size"`
	Tolerance     int `yaml:"tolerance"`
}

var (
	cfg  *Config
	once sync.Once
)

func Load(configPath string) (*Config, error) {
	var loadErr error
	once.Do(func() {
		if configPath == "" {
			configPath = getDefaultConfigPath()
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			loadErr = fmt.Errorf("failed to read config file: %w", err)
			return
		}

		cfg = &Config{}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			loadErr = fmt.Errorf("failed to parse config file: %w", err)
			cfg = nil
			return
		}

		if err := cfg.Validate(); err != nil {
			loadErr = fmt.Errorf("config validation failed: %w", err)
			cfg = nil
			return
		}
	})

	if loadErr != nil {
		return nil, loadErr
	}
	return cfg, nil
}

func getDefaultConfigPath() string {
	execPath, err := os.Executable()
	if err != nil {
		return "config/config.yaml"
	}
	dir := filepath.Dir(execPath)
	return filepath.Join(dir, "config", "config.yaml")
}

func Get() *Config {
	if cfg == nil {
		panic("config not loaded, call Load() first")
	}
	return cfg
}

func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}
	if c.Captcha.ExpireMinutes <= 0 {
		return fmt.Errorf("captcha expire_minutes must be positive")
	}
	if c.Captcha.MaxAttempts <= 0 {
		return fmt.Errorf("captcha max_attempts must be positive")
	}
	return nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type RiskConfig struct {
	SlideSpeedThresholdFast   time.Duration
	SlideSpeedThresholdSlow   time.Duration
	SmoothnessThreshold       float64
	JitterThreshold           float64
	MaxFailureCount           int
	CriticalFailureCount      int
	BlockDuration             time.Duration
	HighFrequencyThreshold    int64
	RiskScoreThresholds       RiskScoreThresholds
}

type RiskScoreThresholds struct {
	Low      int
	Medium   int
	High     int
	Critical int
}

func DefaultRiskConfig() *RiskConfig {
	return &RiskConfig{
		SlideSpeedThresholdFast:   1 * time.Second,
		SlideSpeedThresholdSlow:   30 * time.Second,
		SmoothnessThreshold:       0.95,
		JitterThreshold:           0.1,
		MaxFailureCount:           3,
		CriticalFailureCount:      5,
		BlockDuration:             30 * time.Minute,
		HighFrequencyThreshold:    100,
		RiskScoreThresholds: RiskScoreThresholds{
			Low:      0,
			Medium:   25,
			High:     50,
			Critical: 80,
		},
	}
}

func (c *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type AIConfig struct {
	Enable            bool     `yaml:"enable"`
	ModelPath         string   `yaml:"model_path"`
	HiddenDims        []int    `yaml:"hidden_dims"`
	LearningRate      float64  `yaml:"learning_rate"`
	WeightDecay       float64  `yaml:"weight_decay"`
	BatchSize         int      `yaml:"batch_size"`
	TrainingEpochs    int      `yaml:"training_epochs"`
	ValidationSplit   float64  `yaml:"validation_split"`
	DropoutRate       float64  `yaml:"dropout_rate"`
}

func DefaultAIConfig() *AIConfig {
	return &AIConfig{
		Enable:           true,
		ModelPath:        "models/risk_model.json",
		HiddenDims:       []int{64, 32, 16},
		LearningRate:     0.001,
		WeightDecay:      0.0001,
		BatchSize:        32,
		TrainingEpochs:   100,
		ValidationSplit:  0.2,
		DropoutRate:      0.0,
	}
}
