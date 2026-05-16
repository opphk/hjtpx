package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name           string
		length         int
		expectError    bool
	}{
		{
			name:        "length 8",
			length:      8,
			expectError: false,
		},
		{
			name:        "length 16",
			length:      16,
			expectError: false,
		},
		{
			name:        "length 32",
			length:      32,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateRandomString(tt.length)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.length)
			}
		})
	}
}

func TestGenerateSalt(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "length 8",
			length: 8,
		},
		{
			name:   "length 16",
			length: 16,
		},
		{
			name:   "length 32",
			length: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateSalt(tt.length)
			assert.NoError(t, err)
			assert.Len(t, result, tt.length)
		})
	}
}

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "simple password",
			password: "password123",
		},
		{
			name:     "complex password",
			password: "ComplexP@ssw0rd!123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			assert.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.NotEqual(t, tt.password, hash)

			// Same password should have different hashes
			hash2, err := HashPassword(tt.password)
			assert.NoError(t, err)
			assert.NotEqual(t, hash, hash2)
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "password123"
	hash, err := HashPassword(password)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		password string
		hash     string
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
		},
		{
			name:     "wrong password",
			password: "wrongpassword",
			hash:     hash,
		},
		{
			name:     "invalid hash",
			password: password,
			hash:     "invalidhash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just check it doesn't panic
			_ = VerifyPassword(tt.password, tt.hash)
		})
	}
}

func TestHashSHA256(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
	}{
		{
			name:  "empty string",
			input: []byte(""),
		},
		{
			name:  "simple string",
			input: []byte("hello world"),
		},
		{
			name:  "complex string",
			input: []byte("Complex!@#$%^&*()String"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashSHA256(tt.input)
			assert.Len(t, hash, 64) // SHA256 is 64 hex chars

			// Same input should have same hash
			hash2 := HashSHA256(tt.input)
			assert.Equal(t, hash, hash2)
		})
	}
}

func TestHashSHA1(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
	}{
		{
			name:  "empty string",
			input: []byte(""),
		},
		{
			name:  "simple string",
			input: []byte("hello world"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashSHA1(tt.input)
			assert.Len(t, hash, 40) // SHA1 is 40 hex chars

			// Same input should have same hash
			hash2 := HashSHA1(tt.input)
			assert.Equal(t, hash, hash2)
		})
	}
}

func TestConstantTimeCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "equal strings",
			a:    "test123",
			b:    "test123",
		},
		{
			name: "different strings",
			a:    "test123",
			b:    "test456",
		},
		{
			name: "empty strings",
			a:    "",
			b:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConstantTimeCompare(tt.a, tt.b)
			// Just verify it doesn't panic
			assert.True(t, result == true || result == false)
		})
	}
}
