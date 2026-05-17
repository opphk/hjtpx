package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID    uint   `json:"user_id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	AppID     uint   `json:"app_id"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secretKey     []byte
	expirationTime time.Duration
	issuer        string
}

func NewJWTManager(secretKey string, expirationTime time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secretKey),
		expirationTime: expirationTime,
		issuer:        issuer,
	}
}

func (j *JWTManager) GenerateToken(userID uint, email, username string, appID uint, role string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Email:    email,
		Username: username,
		AppID:    appID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.expirationTime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    j.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (j *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	return j.GenerateToken(claims.UserID, claims.Email, claims.Username, claims.AppID, claims.Role)
}
