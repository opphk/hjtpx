package jwt

import (
	"testing"
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
