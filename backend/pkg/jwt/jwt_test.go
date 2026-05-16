package jwt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitJWT(t *testing.T) {
	InitJWT("test-secret-key")
}

func TestGenerateToken(t *testing.T) {
	InitJWT("test-secret-key")

	token, err := GenerateToken(1, "testuser")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	if token == "" {
		t.Fatal("Generated token is empty")
	}
}

func TestParseToken(t *testing.T) {
	InitJWT("test-secret-key")

	originalAdminID := uint(1)
	originalUsername := "testuser"

	token, err := GenerateToken(originalAdminID, originalUsername)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if claims.AdminID != originalAdminID {
		t.Errorf("Expected admin ID %d, got %d", originalAdminID, claims.AdminID)
	}
	if claims.Username != originalUsername {
		t.Errorf("Expected username %s, got %s", originalUsername, claims.Username)
	}
}

func TestParseInvalidToken(t *testing.T) {
	InitJWT("test-secret-key")

	_, err := ParseToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
}

func TestParseTokenWithWrongSecret(t *testing.T) {
	InitJWT("secret-1")

	token, err := GenerateToken(1, "testuser")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	InitJWT("secret-2")

	_, err = ParseToken(token)
	if err == nil {
		t.Error("Expected error for token signed with wrong secret, got nil")
	}
}

func TestGenerateTokenDifferentUsers(t *testing.T) {
	InitJWT("test-secret-key")

	token1, err := GenerateToken(1, "user1")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	token2, err := GenerateToken(2, "user2")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	assert.NotEqual(t, token1, token2)

	claims1, _ := ParseToken(token1)
	claims2, _ := ParseToken(token2)

	assert.Equal(t, uint(1), claims1.AdminID)
	assert.Equal(t, "user1", claims1.Username)

	assert.Equal(t, uint(2), claims2.AdminID)
	assert.Equal(t, "user2", claims2.Username)
}

func TestClaimsStructure(t *testing.T) {
	claims := Claims{
		AdminID:  123,
		Username: "testadmin",
	}

	assert.Equal(t, uint(123), claims.AdminID)
	assert.Equal(t, "testadmin", claims.Username)
}

func TestTokenExpiration(t *testing.T) {
	InitJWT("test-secret-key")

	token, err := GenerateToken(1, "testuser")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.NotBefore)
	assert.Equal(t, "hjtpx-admin", claims.Issuer)
}

func TestEmptyToken(t *testing.T) {
	InitJWT("test-secret-key")

	_, err := ParseToken("")
	if err == nil {
		t.Error("Expected error for empty token, got nil")
	}
}

func TestMalformedToken(t *testing.T) {
	InitJWT("test-secret-key")

	testTokens := []string{
		"not-a-jwt-token",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		"abc.def.ghi",
		"HEADER.PAYLOAD.SIGNATURE",
	}

	for _, token := range testTokens {
		_, err := ParseToken(token)
		if err == nil {
			t.Errorf("Expected error for malformed token '%s', got nil", token)
		}
	}
}

func TestTokenWithMissingClaims(t *testing.T) {
	InitJWT("test-secret-key")

	token, err := GenerateToken(1, "testuser")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("Failed to parse valid token: %v", err)
	}

	assert.Greater(t, claims.ExpiresAt.Unix(), claims.IssuedAt.Unix())
	assert.GreaterOrEqual(t, claims.NotBefore.Unix(), claims.IssuedAt.Unix()-1)
	assert.LessOrEqual(t, claims.NotBefore.Unix(), claims.IssuedAt.Unix()+1)
}
