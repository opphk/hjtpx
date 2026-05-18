package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEnvManager(t *testing.T) {
	manager := GetEnvManager()

	os.Setenv("TEST_VAR", "test_value")
	result := manager.Get("TEST_VAR", "default")
	assert.Equal(t, "test_value", result)
	os.Unsetenv("TEST_VAR")

	result = manager.Get("NON_EXISTENT", "default_value")
	assert.Equal(t, "default_value", result)
}

func TestEnvManagerGetInt(t *testing.T) {
	manager := GetEnvManager()

	os.Setenv("TEST_INT", "42")
	result := manager.GetInt("TEST_INT", 0)
	assert.Equal(t, 42, result)
	os.Unsetenv("TEST_INT")

	result = manager.GetInt("NON_EXISTENT", 100)
	assert.Equal(t, 100, result)
}

func TestEnvManagerGetBool(t *testing.T) {
	manager := GetEnvManager()

	os.Setenv("TEST_BOOL", "true")
	result := manager.GetBool("TEST_BOOL", false)
	assert.True(t, result)
	os.Unsetenv("TEST_BOOL")

	os.Setenv("TEST_BOOL_1", "1")
	result = manager.GetBool("TEST_BOOL_1", false)
	assert.True(t, result)
	os.Unsetenv("TEST_BOOL_1")

	result = manager.GetBool("NON_EXISTENT", true)
	assert.True(t, result)
}

func TestEnvManagerGetDuration(t *testing.T) {
	manager := GetEnvManager()

	os.Setenv("TEST_DURATION", "1h30m")
	result := manager.GetDuration("TEST_DURATION", 0)
	assert.Equal(t, 90*time.Minute, result)
	os.Unsetenv("TEST_DURATION")

	result = manager.GetDuration("NON_EXISTENT", 5*time.Minute)
	assert.Equal(t, 5*time.Minute, result)
}

func TestEnvManagerGetSlice(t *testing.T) {
	manager := GetEnvManager()

	os.Setenv("TEST_SLICE", "a,b,c")
	result := manager.GetSlice("TEST_SLICE", nil, ",")
	assert.Equal(t, []string{"a", "b", "c"}, result)
	os.Unsetenv("TEST_SLICE")

	defaultSlice := []string{"x", "y"}
	result = manager.GetSlice("NON_EXISTENT", defaultSlice, ",")
	assert.Equal(t, defaultSlice, result)
}

func TestEnvManagerSet(t *testing.T) {
	manager := GetEnvManager()

	manager.Set("NEW_VAR", "new_value")
	result := manager.Get("NEW_VAR", "")
	assert.Equal(t, "new_value", result)
	os.Unsetenv("NEW_VAR")
}

func TestEnvManagerMustGet(t *testing.T) {
	manager := GetEnvManager()

	os.Setenv("REQUIRED_VAR", "required_value")
	result, err := manager.MustGet("REQUIRED_VAR")
	assert.NoError(t, err)
	assert.Equal(t, "required_value", result)
	os.Unsetenv("REQUIRED_VAR")

	_, err = manager.MustGet("NON_EXISTENT_VAR_12345")
	assert.Error(t, err)
}

func TestEnvManagerIsProduction(t *testing.T) {
	manager := GetEnvManager()

	os.Setenv("GIN_MODE", "release")
	assert.True(t, manager.IsProduction())
	os.Unsetenv("GIN_MODE")

	os.Setenv("APP_ENV", "production")
	assert.True(t, manager.IsProduction())
	os.Unsetenv("APP_ENV")

	os.Setenv("GIN_MODE", "debug")
	os.Setenv("APP_ENV", "development")
	assert.False(t, manager.IsProduction())
	os.Unsetenv("GIN_MODE")
	os.Unsetenv("APP_ENV")
}

func TestConfigValidator(t *testing.T) {
	v := NewConfigValidator()

	v.AddRule(&RequiredRule{FieldName: "test_field", Value: ""})
	err := v.Validate()
	assert.Error(t, err)

	v2 := NewConfigValidator()
	v2.AddRule(&RequiredRule{FieldName: "test_field", Value: "has_value"})
	err = v2.Validate()
	assert.NoError(t, err)
}

func TestMinLengthRule(t *testing.T) {
	rule := &MinLengthRule{
		FieldName: "password",
		Value:     "short",
		Min:       10,
	}

	err := rule.Validate()
	assert.Error(t, err)

	rule.Value = "this_is_long_enough"
	err = rule.Validate()
	assert.NoError(t, err)
}

func TestMaxLengthRule(t *testing.T) {
	rule := &MaxLengthRule{
		FieldName: "username",
		Value:     "this_is_too_long",
		Max:       5,
	}

	err := rule.Validate()
	assert.Error(t, err)

	rule.Value = "short"
	err = rule.Validate()
	assert.NoError(t, err)
}

func TestRangeRule(t *testing.T) {
	rule := &RangeRule{
		FieldName: "age",
		Value:     150,
		Min:       0,
		Max:       120,
	}

	err := rule.Validate()
	assert.Error(t, err)

	rule.Value = 25
	err = rule.Validate()
	assert.NoError(t, err)
}

func TestEmailRule(t *testing.T) {
	rule := &EmailRule{
		FieldName: "email",
		Value:     "invalid-email",
	}

	err := rule.Validate()
	assert.Error(t, err)

	rule.Value = "valid@example.com"
	err = rule.Validate()
	assert.NoError(t, err)
}

func TestIPAddressRule(t *testing.T) {
	rule := &IPAddressRule{
		FieldName: "ip",
		Value:     "invalid-ip",
	}

	err := rule.Validate()
	assert.Error(t, err)

	rule.Value = "192.168.1.1"
	err = rule.Validate()
	assert.NoError(t, err)

	rule.Value = "999.999.999.999"
	err = rule.Validate()
	assert.Error(t, err)
}

func TestInSliceRule(t *testing.T) {
	rule := &InSliceRule{
		FieldName:      "status",
		Value:          "pending",
		AllowedValues: []string{"pending", "approved", "rejected"},
	}

	err := rule.Validate()
	assert.NoError(t, err)

	rule.Value = "unknown"
	err = rule.Validate()
	assert.Error(t, err)
}

func TestValidateServerConfig(t *testing.T) {
	cfg := &ServerConfig{
		Port: "8080",
		Mode: "release",
	}

	err := ValidateServerConfig(cfg)
	assert.NoError(t, err)

	cfg.Port = "invalid"
	err = ValidateServerConfig(cfg)
	assert.Error(t, err)

	cfg.Port = "8080"
	cfg.Mode = "invalid"
	err = ValidateServerConfig(cfg)
	assert.Error(t, err)
}

func TestValidatePostgresConfig(t *testing.T) {
	cfg := &PostgresConfig{
		Host:    "localhost",
		Port:    "5432",
		SSLMode: "disable",
	}

	err := ValidatePostgresConfig(cfg)
	assert.NoError(t, err)

	cfg.Host = ""
	err = ValidatePostgresConfig(cfg)
	assert.Error(t, err)
}

func TestValidateJWTConfig(t *testing.T) {
	cfg := &JWTConfig{
		Secret:      "this_is_a_very_long_secret_key_for_testing",
		ExpireHours: 24,
	}

	err := ValidateJWTConfig(cfg)
	assert.NoError(t, err)

	cfg.Secret = "short"
	err = ValidateJWTConfig(cfg)
	assert.Error(t, err)
}

func TestValidateFullConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: "8080",
			Mode: "release",
		},
		Postgres: PostgresConfig{
			Host:    "localhost",
			Port:    "5432",
			SSLMode: "disable",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: "6379",
		},
		JWT: JWTConfig{
			Secret:      "this_is_a_very_long_secret_key_for_testing",
			ExpireHours: 24,
		},
		Database: DatabaseConfig{
			ConnectionPool: ConnectionPoolConfig{
				MaxOpenConns: 100,
				MaxIdleConns: 10,
			},
		},
	}

	err := ValidateFullConfig(cfg)
	assert.NoError(t, err)
}

func TestValidateConfigPerformance(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: "8080",
			Mode: "release",
		},
		Postgres: PostgresConfig{
			Host:    "localhost",
			Port:    "5432",
			SSLMode: "disable",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: "6379",
		},
		JWT: JWTConfig{
			Secret:      "this_is_a_very_long_secret_key_for_testing",
			ExpireHours: 24,
		},
		Database: DatabaseConfig{
			ConnectionPool: ConnectionPoolConfig{
				MaxOpenConns: 100,
				MaxIdleConns: 10,
			},
		},
	}

	valid, duration := ValidateConfigPerformance(cfg)
	assert.True(t, valid)
	assert.Less(t, duration.Milliseconds(), int64(100))
}

func TestGetConfigIssues(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: "",
		},
		Postgres: PostgresConfig{
			Host: "",
		},
		Redis: RedisConfig{
			Host: "",
		},
		JWT: JWTConfig{
			Secret: "",
		},
		Database: DatabaseConfig{
			ConnectionPool: ConnectionPoolConfig{
				MaxOpenConns: 5,
			},
		},
	}

	issues := GetConfigIssues(cfg)
	assert.NotEmpty(t, issues)
	assert.Contains(t, issues, "Server port is not configured")
	assert.Contains(t, issues, "PostgreSQL host is not configured")
	assert.Contains(t, issues, "Redis host is not configured")
	assert.Contains(t, issues, "JWT secret is not properly configured (change in production)")
	assert.Contains(t, issues, "Database connection pool max open connections is too low")
}

func TestValidatorWithTimeout(t *testing.T) {
	validator := NewValidatorWithTimeout(100 * time.Millisecond)

	err := validator.Validate(func() error {
		return nil
	})
	assert.NoError(t, err)

	err = validator.Validate(func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}
