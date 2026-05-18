package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ValidationRule interface {
	Validate() error
	Name() string
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

type RequiredRule struct {
	FieldName string
	Value     interface{}
}

func (r *RequiredRule) Validate() error {
	if r.Value == nil || r.Value == "" {
		return &ValidationError{Field: r.FieldName, Message: "is required", Value: r.Value}
	}
	return nil
}

func (r *RequiredRule) Name() string {
	return "required"
}

type MinLengthRule struct {
	FieldName string
	Value     string
	Min       int
}

func (r *MinLengthRule) Validate() error {
	if len(r.Value) < r.Min {
		return &ValidationError{
			Field:   r.FieldName,
			Message: fmt.Sprintf("must be at least %d characters", r.Min),
			Value:   r.Value,
		}
	}
	return nil
}

func (r *MinLengthRule) Name() string {
	return "min_length"
}

type MaxLengthRule struct {
	FieldName string
	Value     string
	Max       int
}

func (r *MaxLengthRule) Validate() error {
	if len(r.Value) > r.Max {
		return &ValidationError{
			Field:   r.FieldName,
			Message: fmt.Sprintf("must be at most %d characters", r.Max),
			Value:   r.Value,
		}
	}
	return nil
}

func (r *MaxLengthRule) Name() string {
	return "max_length"
}

type RangeRule struct {
	FieldName string
	Value     int
	Min       int
	Max       int
}

func (r *RangeRule) Validate() error {
	if r.Value < r.Min || r.Value > r.Max {
		return &ValidationError{
			Field:   r.FieldName,
			Message: fmt.Sprintf("must be between %d and %d", r.Min, r.Max),
			Value:   r.Value,
		}
	}
	return nil
}

func (r *RangeRule) Name() string {
	return "range"
}

type EmailRule struct {
	FieldName string
	Value     string
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func (r *EmailRule) Validate() error {
	if r.Value != "" && !emailRegex.MatchString(r.Value) {
		return &ValidationError{
			Field:   r.FieldName,
			Message: "must be a valid email address",
			Value:   r.Value,
		}
	}
	return nil
}

func (r *EmailRule) Name() string {
	return "email"
}

type URLRule struct {
	FieldName string
	Value     string
}

func (r *URLRule) Validate() error {
	if r.Value == "" {
		return nil
	}
	_, err := url.ParseRequestURI(r.Value)
	if err != nil {
		return &ValidationError{
			Field:   r.FieldName,
			Message: "must be a valid URL",
			Value:   r.Value,
		}
	}
	return nil
}

func (r *URLRule) Name() string {
	return "url"
}

type IPAddressRule struct {
	FieldName string
	Value     string
}

var ipRegex = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)

func (r *IPAddressRule) Validate() error {
	if r.Value == "" {
		return nil
	}
	if !ipRegex.MatchString(r.Value) {
		return &ValidationError{
			Field:   r.FieldName,
			Message: "must be a valid IP address",
			Value:   r.Value,
		}
	}

	parts := strings.Split(r.Value, ".")
	for _, part := range parts {
		num, _ := strconv.Atoi(part)
		if num < 0 || num > 255 {
			return &ValidationError{
				Field:   r.FieldName,
				Message: "must be a valid IP address",
				Value:   r.Value,
			}
		}
	}
	return nil
}

func (r *IPAddressRule) Name() string {
	return "ip_address"
}

type PathExistsRule struct {
	FieldName string
	Value     string
	IsDir     bool
}

func (r *PathExistsRule) Validate() error {
	if r.Value == "" {
		return nil
	}

	info, err := os.Stat(r.Value)
	if err != nil {
		if os.IsNotExist(err) {
			return &ValidationError{
				Field:   r.FieldName,
				Message: "path does not exist",
				Value:   r.Value,
			}
		}
		return err
	}

	if r.IsDir && !info.IsDir() {
		return &ValidationError{
			Field:   r.FieldName,
			Message: "must be a directory",
			Value:   r.Value,
		}
	}

	if !r.IsDir && info.IsDir() {
		return &ValidationError{
			Field:   r.FieldName,
			Message: "must be a file",
			Value:   r.Value,
		}
	}

	return nil
}

func (r *PathExistsRule) Name() string {
	return "path_exists"
}

type InSliceRule struct {
	FieldName  string
	Value      string
	AllowedValues []string
}

func (r *InSliceRule) Validate() error {
	if r.Value == "" {
		return nil
	}

	for _, allowed := range r.AllowedValues {
		if r.Value == allowed {
			return nil
		}
	}

	return &ValidationError{
		Field:   r.FieldName,
		Message: fmt.Sprintf("must be one of: %v", r.AllowedValues),
		Value:   r.Value,
	}
}

func (r *InSliceRule) Name() string {
	return "in_slice"
}

type ConfigValidator struct {
	rules []ValidationRule
}

func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		rules: make([]ValidationRule, 0),
	}
}

func (v *ConfigValidator) AddRule(rule ValidationRule) *ConfigValidator {
	v.rules = append(v.rules, rule)
	return v
}

func (v *ConfigValidator) Validate() error {
	var validationErrors ValidationErrors
	for _, rule := range v.rules {
		if err := rule.Validate(); err != nil {
			validationErrors = append(validationErrors, *err.(*ValidationError))
		}
	}

	if validationErrors.HasErrors() {
		return validationErrors
	}
	return nil
}

func ValidateServerConfig(cfg *ServerConfig) error {
	v := NewConfigValidator()

	if _, err := strconv.Atoi(cfg.Port); err != nil {
		v.AddRule(&RangeRule{FieldName: "server.port", Value: 0, Min: 1, Max: 65535})
	}

	v.AddRule(&InSliceRule{
		FieldName: "server.mode",
		Value:     cfg.Mode,
		AllowedValues: []string{"debug", "release", "test"},
	})

	return v.Validate()
}

func ValidatePostgresConfig(cfg *PostgresConfig) error {
	v := NewConfigValidator()

	v.AddRule(&RequiredRule{FieldName: "postgres.host", Value: cfg.Host})
	v.AddRule(&RequiredRule{FieldName: "postgres.port", Value: cfg.Port})

	port, _ := strconv.Atoi(cfg.Port)
	v.AddRule(&RangeRule{FieldName: "postgres.port", Value: port, Min: 1, Max: 65535})

	v.AddRule(&InSliceRule{
		FieldName: "postgres.sslmode",
		Value:     cfg.SSLMode,
		AllowedValues: []string{"disable", "require", "verify-full", "verify-ca"},
	})

	return v.Validate()
}

func ValidateRedisConfig(cfg *RedisConfig) error {
	v := NewConfigValidator()

	v.AddRule(&RequiredRule{FieldName: "redis.host", Value: cfg.Host})

	port, _ := strconv.Atoi(cfg.Port)
	v.AddRule(&RangeRule{FieldName: "redis.port", Value: port, Min: 1, Max: 65535})

	return v.Validate()
}

func ValidateJWTConfig(cfg *JWTConfig) error {
	v := NewConfigValidator()

	v.AddRule(&RequiredRule{FieldName: "jwt.secret", Value: cfg.Secret})
	v.AddRule(&MinLengthRule{FieldName: "jwt.secret", Value: cfg.Secret, Min: 32})

	return v.Validate()
}

func ValidateDatabaseConfig(cfg *DatabaseConfig) error {
	v := NewConfigValidator()

	if cfg.ConnectionPool.MaxOpenConns < 1 {
		v.AddRule(&RangeRule{FieldName: "database.connection_pool.max_open_conns", Value: 1, Min: 1, Max: 10000})
	}

	if cfg.ConnectionPool.MaxIdleConns < 0 {
		v.AddRule(&RangeRule{FieldName: "database.connection_pool.max_idle_conns", Value: 0, Min: 0, Max: 1000})
	}

	return v.Validate()
}

func ValidateSecurityConfig(cfg *SecurityConfig) error {
	v := NewConfigValidator()

	if cfg.CSRF.TokenLength < 16 {
		v.AddRule(&RangeRule{FieldName: "security.csrf.token_length", Value: cfg.CSRF.TokenLength, Min: 16, Max: 128})
	}

	if cfg.Signature.SecretKey == "" || cfg.Signature.SecretKey == "default-secret-key-change-in-production" {
		v.AddRule(&RequiredRule{FieldName: "security.signature.secret_key", Value: cfg.Signature.SecretKey})
	}

	if cfg.Crypto.AESKeySize != 16 && cfg.Crypto.AESKeySize != 24 && cfg.Crypto.AESKeySize != 32 {
		v.AddRule(&RangeRule{FieldName: "security.crypto.aes_key_size", Value: 0, Min: 16, Max: 32})
	}

	if cfg.Password.MinLength < 6 {
		v.AddRule(&RangeRule{FieldName: "security.password.min_length", Value: cfg.Password.MinLength, Min: 6, Max: 128})
	}

	return v.Validate()
}

func ValidateFullConfig(cfg *Config) error {
	var allErrors ValidationErrors

	if err := ValidateServerConfig(&cfg.Server); err != nil {
		allErrors = append(allErrors, err.(ValidationErrors)...)
	}

	if err := ValidatePostgresConfig(&cfg.Postgres); err != nil {
		allErrors = append(allErrors, err.(ValidationErrors)...)
	}

	if err := ValidateRedisConfig(&cfg.Redis); err != nil {
		allErrors = append(allErrors, err.(ValidationErrors)...)
	}

	if err := ValidateJWTConfig(&cfg.JWT); err != nil {
		allErrors = append(allErrors, err.(ValidationErrors)...)
	}

	if err := ValidateDatabaseConfig(&cfg.Database); err != nil {
		allErrors = append(allErrors, err.(ValidationErrors)...)
	}

	if allErrors.HasErrors() {
		return allErrors
	}

	return nil
}

func ValidateConfigWithDefaults(cfg *Config) error {
	v := NewConfigValidator()

	v.AddRule(&RequiredRule{FieldName: "server.port", Value: cfg.Server.Port})
	v.AddRule(&RequiredRule{FieldName: "postgres.host", Value: cfg.Postgres.Host})
	v.AddRule(&RequiredRule{FieldName: "redis.host", Value: cfg.Redis.Host})

	return v.Validate()
}

type ValidatorWithTimeout struct {
	timeout time.Duration
}

func NewValidatorWithTimeout(timeout time.Duration) *ValidatorWithTimeout {
	return &ValidatorWithTimeout{timeout: timeout}
}

func (v *ValidatorWithTimeout) Validate(fn func() error) error {
	done := make(chan error, 1)

	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(v.timeout):
		return errors.New("validation timeout")
	}
}

func ValidateConfigPerformance(cfg *Config) (bool, time.Duration) {
	start := time.Now()
	err := ValidateFullConfig(cfg)
	duration := time.Since(start)

	return err == nil, duration
}

func GetConfigIssues(cfg *Config) []string {
	var issues []string

	if cfg.Server.Port == "" {
		issues = append(issues, "Server port is not configured")
	}

	if cfg.Postgres.Host == "" {
		issues = append(issues, "PostgreSQL host is not configured")
	}

	if cfg.Redis.Host == "" {
		issues = append(issues, "Redis host is not configured")
	}

	if cfg.JWT.Secret == "" || cfg.JWT.Secret == "your-secret-key-change-in-production" {
		issues = append(issues, "JWT secret is not properly configured (change in production)")
	}

	if cfg.Database.ConnectionPool.MaxOpenConns < 10 {
		issues = append(issues, "Database connection pool max open connections is too low")
	}

	return issues
}
