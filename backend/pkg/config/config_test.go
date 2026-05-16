package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetConfig(t *testing.T) {
	// First call - should initialize config
	config := GetConfig()
	assert.NotNil(t, config)

	// Second call - should return same instance
	config2 := GetConfig()
	assert.Equal(t, config, config2)
}

func TestLoadConfig(t *testing.T) {
	// Just verify the function doesn't panic
	config := LoadConfig()
	assert.NotNil(t, config)
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		defaultValue   string
		expectedResult string
		checkNonEmpty  bool
	}{
		{
			name:          "non-existent env var",
			key:           "NON_EXISTENT_ENV_VAR_123456",
			defaultValue:  "default_value",
			checkNonEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEnv(tt.key, tt.defaultValue)
			if tt.checkNonEmpty {
				assert.NotEmpty(t, result)
			} else {
				assert.Equal(t, tt.defaultValue, result)
			}
		})
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		defaultValue   int
		expectedResult int
	}{
		{
			name:           "non-existent env var",
			key:            "NON_EXISTENT_ENV_VAR_123456",
			defaultValue:   42,
			expectedResult: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEnvAsInt(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	config := LoadConfig()

	// Check default values
	assert.Equal(t, "8080", config.Server.Port)
	assert.Equal(t, "debug", config.Server.Mode)
	assert.Equal(t, "localhost", config.Postgres.Host)
	assert.Equal(t, "5432", config.Postgres.Port)
	assert.Equal(t, "verification", config.Postgres.DBName)
	assert.Equal(t, "localhost", config.Redis.Host)
	assert.Equal(t, "6379", config.Redis.Port)
	assert.Equal(t, "your-secret-key-change-in-production", config.JWT.Secret)
	assert.Equal(t, 24, config.JWT.ExpireHours)
}
