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

var jwtSecret []byte
var jwtExpireHours = 24

// InitJWT 初始化JWT配置
func InitJWT(secret string) {
	jwtSecret = []byte(secret)
}

// GenerateToken 生成JWT token
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

// ParseToken 解析JWT token
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
