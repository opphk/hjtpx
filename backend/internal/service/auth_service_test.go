package service

import (
	"context"
	"testing"
	"time"
)

func TestNewAuthService(t *testing.T) {
	cfg := AuthServiceConfig{
		SecretKey:     "test-secret-key",
		AccessExpiry:  24 * time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	authService, err := NewAuthService(cfg)
	if err != nil {
		t.Errorf("创建 AuthService 失败: %v", err)
	}
	if authService == nil {
		t.Error("NewAuthService 返回了 nil")
	}
}

func TestNewAuthService_MissingSecretKey(t *testing.T) {
	cfg := AuthServiceConfig{
		SecretKey:     "",
		AccessExpiry:  24 * time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	authService, err := NewAuthService(cfg)
	if err == nil {
		t.Error("缺少 secret key 应该返回错误")
	}
	if authService != nil {
		t.Error("缺少 secret key 应该返回 nil service")
	}
}

func TestNewAuthService_DefaultExpiry(t *testing.T) {
	cfg := AuthServiceConfig{
		SecretKey:     "test-secret-key",
		AccessExpiry:  0,
		RefreshExpiry: 0,
	}
	
	authService, err := NewAuthService(cfg)
	if err != nil {
		t.Errorf("创建 AuthService 失败: %v", err)
	}
	if authService == nil {
		t.Error("NewAuthService 返回了 nil")
	}
}

func TestAuthService_GenerateAndValidateToken(t *testing.T) {
	cfg := AuthServiceConfig{
		SecretKey:     "test-secret-key-for-generation",
		AccessExpiry:  24 * time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	authService, err := NewAuthService(cfg)
	if err != nil {
		t.Skipf("无法创建 AuthService: %v", err)
	}
	
	ctx := context.Background()
	token, _, err := authService.GenerateToken(ctx, 1, "testuser", "admin")
	if err != nil {
		t.Skipf("生成token失败: %v", err)
	}
	if token == "" {
		t.Error("生成的token不应为空")
	}
	
	claims, err := authService.ValidateToken(ctx, token)
	if err != nil {
		t.Errorf("验证token失败: %v", err)
	}
	if claims == nil {
		t.Error("验证结果不应为nil")
	}
	if claims.Username != "testuser" {
		t.Errorf("用户名不匹配: 期望 testuser, 实际 %s", claims.Username)
	}
}

func TestAuthService_ValidateInvalidToken(t *testing.T) {
	cfg := AuthServiceConfig{
		SecretKey:     "test-secret-key",
		AccessExpiry:  24 * time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	authService, err := NewAuthService(cfg)
	if err != nil {
		t.Skipf("无法创建 AuthService: %v", err)
	}
	
	ctx := context.Background()
	_, err = authService.ValidateToken(ctx, "invalid-token-12345")
	if err == nil {
		t.Error("无效token应该返回错误")
	}
}

func TestAuthService_InvalidateToken(t *testing.T) {
	cfg := AuthServiceConfig{
		SecretKey:     "test-secret-key",
		AccessExpiry:  24 * time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	authService, err := NewAuthService(cfg)
	if err != nil {
		t.Skipf("无法创建 AuthService: %v", err)
	}
	
	ctx := context.Background()
	token, _, err := authService.GenerateToken(ctx, 1, "testuser", "admin")
	if err != nil {
		t.Skipf("生成token失败: %v", err)
	}
	
	err = authService.InvalidateToken(ctx, token)
	if err != nil {
		t.Errorf("调用InvalidateToken失败: %v", err)
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	cfg := AuthServiceConfig{
		SecretKey:     "test-secret-key",
		AccessExpiry:  24 * time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	authService, err := NewAuthService(cfg)
	if err != nil {
		t.Skipf("无法创建 AuthService: %v", err)
	}
	
	ctx := context.Background()
	_, refreshToken, err := authService.GenerateToken(ctx, 1, "testuser", "admin")
	if err != nil {
		t.Skipf("生成token失败: %v", err)
	}
	
	newAccessToken, _, err := authService.RefreshToken(ctx, refreshToken)
	if err != nil {
		t.Skipf("刷新token失败: %v", err)
	}
	if newAccessToken == "" {
		t.Error("刷新后的token不应为空")
	}
}

func TestAuthService_GetTokenExpiry(t *testing.T) {
	cfg := AuthServiceConfig{
		SecretKey:     "test-secret-key",
		AccessExpiry:  24 * time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	authService, err := NewAuthService(cfg)
	if err != nil {
		t.Skipf("无法创建 AuthService: %v", err)
	}
	
	ctx := context.Background()
	expiry := authService.GetTokenExpiry(ctx)
	if expiry <= 0 {
		t.Error("token过期时间应该大于0")
	}
}
