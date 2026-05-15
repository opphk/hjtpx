package jwt

import (
	"testing"

	"github.com/hjtpx/hjtpx/pkg/config"
)

func TestInitJWT(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:     "test-secret-key",
		ExpireHours: 24,
	}
	InitJWT(cfg)
}

func TestGenerateToken(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:     "test-secret-key",
		ExpireHours: 24,
	}
	InitJWT(cfg)

	token, err := GenerateToken(1, "testuser")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	if token == "" {
		t.Fatal("Generated token is empty")
	}
}

func TestParseToken(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:     "test-secret-key",
		ExpireHours: 24,
	}
	InitJWT(cfg)

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
	cfg := &config.JWTConfig{
		Secret:     "test-secret-key",
		ExpireHours: 24,
	}
	InitJWT(cfg)

	_, err := ParseToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
}

func TestParseTokenWithWrongSecret(t *testing.T) {
	cfg1 := &config.JWTConfig{
		Secret:     "secret-1",
		ExpireHours: 24,
	}
	InitJWT(cfg1)

	token, err := GenerateToken(1, "testuser")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	cfg2 := &config.JWTConfig{
		Secret:     "secret-2",
		ExpireHours: 24,
	}
	InitJWT(cfg2)

	_, err = ParseToken(token)
	if err == nil {
		t.Error("Expected error for token signed with wrong secret, got nil")
	}
}
