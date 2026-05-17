package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurityService(t *testing.T) {
	t.Run("SanitizeInput removes SQL keywords", func(t *testing.T) {
		service := NewSecurityService(nil)

		testCases := []struct {
			input    string
			expected string
		}{
			{"' OR '1'='1", "  1 1"},
			{"'; DROP TABLE users;--", " DROP TABLE users "},
			{"UNION SELECT * FROM", "  * FROM"},
			{"normal input", "normal input"},
			{"<script>alert(1)</script>", "<script>alert(1)</script>"},
		}

		for _, tc := range testCases {
			result := service.SanitizeInput(tc.input)
			assert.NotContains(t, strings.ToLower(result), "union")
			assert.NotContains(t, strings.ToLower(result), "select")
			assert.NotContains(t, strings.ToLower(result), "drop")
			assert.NotContains(t, strings.ToLower(result), "delete")
			assert.NotContains(t, strings.ToLower(result), "insert")
			assert.NotContains(t, strings.ToLower(result), "update")
			assert.NotContains(t, strings.ToLower(result), "exec")
		}
	})

	t.Run("SanitizeHTML removes script tags", func(t *testing.T) {
		service := NewSecurityService(nil)

		testCases := []struct {
			input    string
			contains []string
		}{
			{
				input:    "<script>alert('xss')</script>Hello",
				contains: []string{"script", "alert"},
			},
			{
				input:    "<img src=x onerror=alert(1)>",
				contains: []string{"onerror"},
			},
			{
				input:    "Normal text without HTML",
				contains: []string{},
			},
			{
				input:    "<iframe src='evil.com'></iframe>",
				contains: []string{"iframe"},
			},
		}

		for _, tc := range testCases {
			result := service.SanitizeHTML(tc.input)
			for _, shouldNotContain := range tc.contains {
				assert.NotContains(t, strings.ToLower(result), shouldNotContain)
			}
		}
	})

	t.Run("ValidatePasswordStrength", func(t *testing.T) {
		service := NewSecurityService(nil)

		testCases := []struct {
			name     string
			password string
			wantErr  bool
		}{
			{"valid password", "Pass@word1", false},
			{"too short", "Pass@1", true},
			{"no uppercase", "pass@word1", true},
			{"no lowercase", "PASS@WORD1", true},
			{"no digit", "Pass@word", true},
			{"no special", "Password1", true},
			{"common password", "password123", true},
			{"empty password", "", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := service.ValidatePasswordStrength(tc.password)
				if tc.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("HashPassword and CheckPassword", func(t *testing.T) {
		service := NewSecurityService(nil)

		password := "Secure@Pass123"
		hash, err := service.HashPassword(password)
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)

		assert.True(t, service.CheckPassword(password, hash))
		assert.False(t, service.CheckPassword("WrongPassword", hash))
	})

	t.Run("UseParameterizedQueries", func(t *testing.T) {
		service := NewSecurityService(nil)
		assert.True(t, service.UseParameterizedQueries())
	})
}

func TestConfigEncryptor(t *testing.T) {
	t.Run("Encrypt and Decrypt", func(t *testing.T) {
		encryptor := NewConfigEncryptor("test-encryption-key")

		plaintext := "sensitive-data-123"
		encrypted, err := encryptor.Encrypt(plaintext)
		assert.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, plaintext, encrypted)

		decrypted, err := encryptor.Decrypt(encrypted)
		assert.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("Encrypt empty string", func(t *testing.T) {
		encryptor := NewConfigEncryptor("test-key")

		encrypted, err := encryptor.Encrypt("")
		assert.NoError(t, err)
		assert.Empty(t, encrypted)

		decrypted, err := encryptor.Decrypt("")
		assert.NoError(t, err)
		assert.Empty(t, decrypted)
	})

	t.Run("Decrypt invalid data", func(t *testing.T) {
		encryptor := NewConfigEncryptor("test-key")

		_, err := encryptor.Decrypt("invalid-base64!")
		assert.Error(t, err)
	})

	t.Run("Different keys produce different ciphertexts", func(t *testing.T) {
		encryptor1 := NewConfigEncryptor("key1")
		encryptor2 := NewConfigEncryptor("key2")

		plaintext := "test data"
		encrypted1, _ := encryptor1.Encrypt(plaintext)
		encrypted2, _ := encryptor2.Encrypt(plaintext)

		assert.NotEqual(t, encrypted1, encrypted2)
	})

	t.Run("EncryptConfig and DecryptConfig", func(t *testing.T) {
		encryptor := NewConfigEncryptor("config-key")

		type Config struct {
			Username string
			Password string
		}

		original := Config{
			Username: "admin",
			Password: "secret123",
		}

		encrypted, err := encryptor.EncryptConfig(original)
		assert.NoError(t, err)

		var decrypted Config
		err = encryptor.DecryptConfig(encrypted, &decrypted)
		assert.NoError(t, err)
		assert.Equal(t, original.Username, decrypted.Username)
		assert.Equal(t, original.Password, decrypted.Password)
	})
}

func TestJWTSecurity(t *testing.T) {
	t.Run("CreateTokenPair", func(t *testing.T) {
		security := NewJWTSecurity(nil, "test-secret")

		userID := int64(123)
		tokenPair, err := security.CreateTokenPair(userID)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenPair.AccessToken)
		assert.NotEmpty(t, tokenPair.RefreshToken)
		assert.Equal(t, 900, tokenPair.ExpiresIn)
		assert.Equal(t, "Bearer", tokenPair.TokenType)
	})

	t.Run("ValidateToken with valid token", func(t *testing.T) {
		security := NewJWTSecurity(nil, "test-secret")

		tokenPair, _ := security.CreateTokenPair(123)
		claims, err := security.ValidateToken(tokenPair.AccessToken)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
	})

	t.Run("ValidateToken with invalid token", func(t *testing.T) {
		security := NewJWTSecurity(nil, "test-secret")

		_, err := security.ValidateToken("invalid-token")
		assert.Error(t, err)
	})

	t.Run("IsTokenBlacklisted without Redis", func(t *testing.T) {
		security := NewJWTSecurity(nil, "test-secret")

		assert.False(t, security.IsTokenBlacklisted("some-token"))
	})

	t.Run("RevokeToken", func(t *testing.T) {
		security := NewJWTSecurity(nil, "test-secret")

		tokenPair, _ := security.CreateTokenPair(123)
		err := security.RevokeToken(tokenPair.AccessToken)
		assert.NoError(t, err)
	})
}

func TestCSRFSecurity(t *testing.T) {
	t.Run("GenerateToken", func(t *testing.T) {
		security := NewCSRFSecurity(nil)

		token, err := security.GenerateToken()
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Greater(t, len(token), 20)
	})

	t.Run("Generate unique tokens", func(t *testing.T) {
		security := NewCSRFSecurity(nil)

		token1, _ := security.GenerateToken()
		token2, _ := security.GenerateToken()

		assert.NotEqual(t, token1, token2)
	})
}

func TestRequestValidator(t *testing.T) {
	validator := NewRequestValidator()

	t.Run("Validate email", func(t *testing.T) {
		validEmails := []string{
			"test@example.com",
			"user.name@domain.org",
			"user+tag@example.co.uk",
		}
		invalidEmails := []string{
			"invalid",
			"@example.com",
			"user@",
			"user@.com",
		}

		for _, email := range validEmails {
			err := validator.Validate("email", email, "email")
			assert.NoError(t, err, "Expected %s to be valid", email)
		}

		for _, email := range invalidEmails {
			err := validator.Validate("email", email, "email")
			assert.Error(t, err, "Expected %s to be invalid", email)
		}
	})

	t.Run("Validate phone", func(t *testing.T) {
		validPhones := []string{
			"13812345678",
			"15912345678",
			"18612345678",
		}
		invalidPhones := []string{
			"12345",
			"123456789012",
			"abc12345678",
		}

		for _, phone := range validPhones {
			err := validator.Validate("phone", phone, "phone")
			assert.NoError(t, err, "Expected %s to be valid", phone)
		}

		for _, phone := range invalidPhones {
			err := validator.Validate("phone", phone, "phone")
			assert.Error(t, err, "Expected %s to be invalid", phone)
		}
	})

	t.Run("Validate required", func(t *testing.T) {
		err := validator.Validate("name", "", "required")
		assert.Error(t, err)

		err = validator.Validate("name", "John", "required")
		assert.NoError(t, err)
	})

	t.Run("Validate numeric", func(t *testing.T) {
		err := validator.Validate("count", "123", "numeric")
		assert.NoError(t, err)

		err = validator.Validate("count", "abc", "numeric")
		assert.Error(t, err)
	})

	t.Run("Validate URL", func(t *testing.T) {
		validURLs := []string{
			"http://example.com",
			"https://example.com",
			"http://example.com/path",
		}

		for _, url := range validURLs {
			err := validator.Validate("url", url, "url")
			assert.NoError(t, err, "Expected %s to be valid", url)
		}
	})

	t.Run("ValidateMap", func(t *testing.T) {
		data := map[string]string{
			"email": "test@example.com",
			"phone": "13812345678",
			"name":  "John",
		}

		rules := map[string]string{
			"email": "email",
			"phone": "phone",
			"name":  "required",
		}

		errors := validator.ValidateMap(data, rules)
		assert.Empty(t, errors)
	})

	t.Run("ValidateMap with errors", func(t *testing.T) {
		data := map[string]string{
			"email": "invalid",
			"phone": "12345",
			"name":  "",
		}

		rules := map[string]string{
			"email": "email",
			"phone": "phone",
			"name":  "required",
		}

		errors := validator.ValidateMap(data, rules)
		assert.Len(t, errors, 3)
		assert.Contains(t, errors, "email")
		assert.Contains(t, errors, "phone")
		assert.Contains(t, errors, "name")
	})
}

func TestSecurityMetrics(t *testing.T) {
	metrics := GetSecurityMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.TotalRequests)
}

func TestSecurityPolicy(t *testing.T) {
	policy := DefaultSecurityPolicy

	assert.Equal(t, 5, policy.MaxLoginAttempts)
	assert.Equal(t, 8, policy.PasswordMinLength)
	assert.True(t, policy.PasswordRequireUpper)
	assert.True(t, policy.PasswordRequireLower)
	assert.True(t, policy.PasswordRequireDigit)
	assert.True(t, policy.PasswordRequireSpecial)
}
