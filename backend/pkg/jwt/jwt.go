package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenInvalid = errors.New("token is invalid")
	ErrTokenExpired = errors.New("token is expired")
)

type Claims struct {
	AdminID   uint   `json:"admin_id"`
	Username  string `json:"username"`
	jwt.RegisteredClaims
}

type UserTokenClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var jwtSecret []byte
var userJwtSecret []byte
var jwtExpireHours = 24
var userJwtExpireMinutes = 60

func InitJWT(secret string) {
	jwtSecret = []byte(secret)
}

func InitUserJWT(secret string) {
	userJwtSecret = []byte(secret)
}

func GenerateToken(adminID uint, username string) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(time.Duration(jwtExpireHours) * time.Hour)

	claims := Claims{
		AdminID:  adminID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			NotBefore: jwt.NewNumericDate(nowTime),
			Issuer:    "hjtpx-admin",
		},
	}

	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString(jwtSecret)
	return token, err
}

func GenerateUserToken(userID uint, username string) (string, error) {
	if len(userJwtSecret) == 0 {
		userJwtSecret = jwtSecret
	}

	nowTime := time.Now()
	accessExpireTime := nowTime.Add(time.Duration(userJwtExpireMinutes) * time.Minute)

	claims := UserTokenClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpireTime),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			NotBefore: jwt.NewNumericDate(nowTime),
			Issuer:    "hjtpx-user",
		},
	}

	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString(userJwtSecret)
	return token, err
}

func GenerateUserTokenWithRefresh(userID uint, username string) (accessToken string, refreshToken string, err error) {
	if len(userJwtSecret) == 0 {
		userJwtSecret = jwtSecret
	}

	nowTime := time.Now()
	accessExpireTime := nowTime.Add(15 * time.Minute)
	refreshExpireTime := nowTime.Add(7 * 24 * time.Hour)

	accessClaims := UserTokenClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpireTime),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			NotBefore: jwt.NewNumericDate(nowTime),
			Issuer:    "hjtpx-user",
		},
	}

	refreshClaims := UserTokenClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpireTime),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			NotBefore: jwt.NewNumericDate(nowTime),
			Issuer:    "hjtpx-user-refresh",
		},
	}

	accessTokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenClaims.SignedString(userJwtSecret)
	if err != nil {
		return "", "", err
	}

	refreshTokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshTokenClaims.SignedString(userJwtSecret)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func ParseToken(token string) (*Claims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

func ParseUserToken(token string) (*UserTokenClaims, error) {
	if len(userJwtSecret) == 0 {
		userJwtSecret = jwtSecret
	}

	tokenClaims, err := jwt.ParseWithClaims(token, &UserTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return userJwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := tokenClaims.Claims.(*UserTokenClaims); ok && tokenClaims.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

func ValidateRefreshToken(token string) (*UserTokenClaims, error) {
	if len(userJwtSecret) == 0 {
		userJwtSecret = jwtSecret
	}

	tokenClaims, err := jwt.ParseWithClaims(token, &UserTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return userJwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := tokenClaims.Claims.(*UserTokenClaims)
	if !ok || !tokenClaims.Valid {
		return nil, ErrTokenInvalid
	}

	if claims.Issuer != "hjtpx-user-refresh" {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
