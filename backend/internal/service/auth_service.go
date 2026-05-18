package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenInvalid     = errors.New("token is invalid")
	ErrTokenExpiredAuth = errors.New("token is expired")
	ErrTokenNotReady    = errors.New("token service not ready")
)

type AuthService interface {
	GenerateToken(ctx context.Context, adminID uint, username, role string) (string, string, error)
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
	RefreshToken(ctx context.Context, refreshToken string) (string, string, error)
	InvalidateToken(ctx context.Context, token string) error
	GetTokenExpiry(ctx context.Context) time.Duration
}

type TokenClaims struct {
	AdminID   uint   `json:"admin_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

type authService struct {
	secretKey     []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	issuer        string
	refreshIssuer string
}

type AuthServiceConfig struct {
	SecretKey     string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

func NewAuthService(cfg AuthServiceConfig) (AuthService, error) {
	if cfg.SecretKey == "" {
		return nil, errors.New("secret key cannot be empty")
	}
	if cfg.AccessExpiry <= 0 {
		cfg.AccessExpiry = 24 * time.Hour
	}
	if cfg.RefreshExpiry <= 0 {
		cfg.RefreshExpiry = 7 * 24 * time.Hour
	}

	return &authService{
		secretKey:     []byte(cfg.SecretKey),
		accessExpiry:  cfg.AccessExpiry,
		refreshExpiry: cfg.RefreshExpiry,
		issuer:        "hjtpx-admin",
		refreshIssuer: "hjtpx-admin-refresh",
	}, nil
}

func (s *authService) GenerateToken(ctx context.Context, adminID uint, username, role string) (string, string, error) {
	now := time.Now()
	accessExpiry := now.Add(s.accessExpiry)
	refreshExpiry := now.Add(s.refreshExpiry)

	accessClaims := TokenClaims{
		AdminID:   adminID,
		Username:  username,
		Role:      role,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
			Subject:   username,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString(s.secretKey)
	if err != nil {
		return "", "", err
	}

	refreshClaims := TokenClaims{
		AdminID:   adminID,
		Username:  username,
		Role:      role,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.refreshIssuer,
			Subject:   username,
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString(s.secretKey)
	if err != nil {
		return "", "", err
	}

	return accessTokenStr, refreshTokenStr, nil
}

func (s *authService) ValidateToken(ctx context.Context, tokenStr string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpiredAuth
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	if claims.TokenType != "access" {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshTokenStr string) (string, string, error) {
	token, err := jwt.ParseWithClaims(refreshTokenStr, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", "", ErrTokenExpiredAuth
		}
		return "", "", ErrTokenInvalid
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return "", "", ErrTokenInvalid
	}

	if claims.TokenType != "refresh" {
		return "", "", ErrTokenInvalid
	}

	return s.GenerateToken(ctx, claims.AdminID, claims.Username, claims.Role)
}

func (s *authService) InvalidateToken(ctx context.Context, token string) error {
	return nil
}

func (s *authService) GetTokenExpiry(ctx context.Context) time.Duration {
	return s.accessExpiry
}
