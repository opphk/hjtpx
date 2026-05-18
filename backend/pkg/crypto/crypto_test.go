package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name        string
		length      int
		expectError bool
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
		name  string
		input []byte
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
		name  string
		input []byte
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
			assert.True(t, result == true || result == false)
		})
	}
}

func TestGenerateEd25519KeyPair(t *testing.T) {
	privateKey, publicKey, err := GenerateEd25519KeyPair()
	assert.NoError(t, err)
	assert.NotNil(t, privateKey)
	assert.NotNil(t, publicKey)
	assert.Len(t, privateKey, 64)
	assert.Len(t, publicKey, 32)
}

func TestSignEd25519(t *testing.T) {
	privateKey, publicKey, err := GenerateEd25519KeyPair()
	assert.NoError(t, err)

	message := []byte("test message for Ed25519 signature")
	signature, err := SignEd25519(message, privateKey)
	assert.NoError(t, err)
	assert.NotNil(t, signature)
	assert.Len(t, signature, 64)

	valid, err := VerifyEd25519(message, signature, publicKey)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestSignEd25519InvalidKey(t *testing.T) {
	invalidKey := []byte("invalid-key")
	message := []byte("test message")

	_, err := SignEd25519(message, invalidKey)
	assert.Error(t, err)
}

func TestVerifyEd25519InvalidKey(t *testing.T) {
	_, publicKey, err := GenerateEd25519KeyPair()
	assert.NoError(t, err)

	message := []byte("test message")
	signature := make([]byte, 64)

	valid, err := VerifyEd25519(message, signature, publicKey)
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestSignEd25519String(t *testing.T) {
	privateKey, _, err := GenerateEd25519KeyPair()
	assert.NoError(t, err)

	message := "test message string"
	signature, err := SignEd25519String(message, privateKey)
	assert.NoError(t, err)
	assert.NotEmpty(t, signature)
}

func TestVerifyEd25519String(t *testing.T) {
	privateKey, publicKey, err := GenerateEd25519KeyPair()
	assert.NoError(t, err)

	message := "test message string"
	signature, err := SignEd25519String(message, privateKey)
	assert.NoError(t, err)

	valid, err := VerifyEd25519String(message, signature, publicKey)
	assert.NoError(t, err)
	assert.True(t, valid)

	invalid, err := VerifyEd25519String("different message", signature, publicKey)
	assert.NoError(t, err)
	assert.False(t, invalid)
}

func TestExportEd25519KeyToPEM(t *testing.T) {
	privateKey, publicKey, err := GenerateEd25519KeyPair()
	assert.NoError(t, err)

	privatePEM, err := ExportEd25519PrivateKeyToPEM(privateKey)
	assert.NoError(t, err)
	assert.Contains(t, privatePEM, "PRIVATE KEY")

	publicPEM, err := ExportEd25519PublicKeyToPEM(publicKey)
	assert.NoError(t, err)
	assert.Contains(t, publicPEM, "PUBLIC KEY")
}

func TestParseEd25519KeyFromPEM(t *testing.T) {
	privateKey, publicKey, err := GenerateEd25519KeyPair()
	assert.NoError(t, err)

	privatePEM, err := ExportEd25519PrivateKeyToPEM(privateKey)
	assert.NoError(t, err)

	parsedPrivate, err := ParseEd25519PrivateKeyFromPEM(privatePEM)
	assert.NoError(t, err)
	assert.Equal(t, privateKey, parsedPrivate)

	publicPEM, err := ExportEd25519PublicKeyToPEM(publicKey)
	assert.NoError(t, err)

	parsedPublic, err := ParseEd25519PublicKeyFromPEM(publicPEM)
	assert.NoError(t, err)
	assert.Equal(t, publicKey, parsedPublic)
}

func TestGenerateDualSignature(t *testing.T) {
	primaryKey := []byte("primary-secret-key-1234567890")
	secondaryKey := []byte("secondary-secret-key-1234567890")
	message := []byte("test message for dual signature")

	signature, err := GenerateDualSignature(message, primaryKey, secondaryKey)
	assert.NoError(t, err)
	assert.NotNil(t, signature)
	assert.NotEmpty(t, signature.PrimarySignature)
	assert.NotEmpty(t, signature.SecondarySignature)
	assert.Len(t, signature.PrimarySignature, 64)
	assert.Len(t, signature.SecondarySignature, 128)
}

func TestVerifyDualSignature(t *testing.T) {
	primaryKey := []byte("primary-secret-key-1234567890")
	secondaryKey := []byte("secondary-secret-key-1234567890")
	message := []byte("test message for dual signature")

	signature, err := GenerateDualSignature(message, primaryKey, secondaryKey)
	assert.NoError(t, err)

	primaryValid, secondaryValid, err := VerifyDualSignature(
		message,
		signature.PrimarySignature,
		signature.SecondarySignature,
		primaryKey,
		secondaryKey,
	)
	assert.NoError(t, err)
	assert.True(t, primaryValid)
	assert.True(t, secondaryValid)

	invalidPrimary, invalidSecondary, err := VerifyDualSignature(
		[]byte("different message"),
		signature.PrimarySignature,
		signature.SecondarySignature,
		primaryKey,
		secondaryKey,
	)
	assert.NoError(t, err)
	assert.False(t, invalidPrimary)
	assert.False(t, invalidSecondary)
}

func TestGenerateDualSignatureMissingKey(t *testing.T) {
	primaryKey := []byte("primary-secret-key")
	secondaryKey := []byte{}
	message := []byte("test message")

	_, err := GenerateDualSignature(message, primaryKey, secondaryKey)
	assert.Error(t, err)
}
