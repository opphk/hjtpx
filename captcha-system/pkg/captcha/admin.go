package captcha

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secretKey    []byte
	expiresHours int
}

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

func NewJWTService(secret string, expiresHours int) *JWTService {
	return &JWTService{
		secretKey:    []byte(secret),
		expiresHours: expiresHours,
	}
}

func (s *JWTService) GenerateToken(userID int64, username, role string) (*TokenPair, error) {
	now := time.Now()

	accessClaims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.expiresHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "captcha-system",
			Subject:   username,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString(s.secretKey)
	if err != nil {
		return nil, err
	}

	refreshClaims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.expiresHours*7) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "captcha-system-refresh",
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString(s.secretKey)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshTokenStr,
		ExpiresAt:    accessClaims.ExpiresAt.Unix(),
	}, nil
}

func (s *JWTService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (s *JWTService) RefreshToken(refreshTokenStr string) (*TokenPair, error) {
	claims, err := s.ValidateToken(refreshTokenStr)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	return s.GenerateToken(claims.UserID, claims.Username, claims.Role)
}

func (s *JWTService) GetSecretKey() []byte {
	return s.secretKey
}
