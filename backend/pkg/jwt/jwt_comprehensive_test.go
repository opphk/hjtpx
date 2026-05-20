package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestInitUserJWT(t *testing.T) {
	InitUserJWT("test-user-secret")
	assert.NotEmpty(t, userJwtSecret)
}

func TestGenerateToken(t *testing.T) {
	InitJWT("test-secret")
	
	token, err := GenerateToken(1, "testuser")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	
	claims, err := ParseToken(token)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), claims.AdminID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "hjtpx-admin", claims.Issuer)
}

func TestGenerateUserToken(t *testing.T) {
	InitUserJWT("user-secret")
	
	token, err := GenerateUserToken(1, "testuser")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	
	claims, err := ParseUserToken(token)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "hjtpx-user", claims.Issuer)
}

func TestGenerateUserToken_DefaultToAdminSecret(t *testing.T) {
	userJwtSecret = []byte{}
	InitJWT("admin-secret")
	
	token, err := GenerateUserToken(1, "testuser")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	
	userJwtSecret = []byte("user-secret")
}

func TestGenerateUserTokenWithRefresh(t *testing.T) {
	InitUserJWT("user-secret")
	
	accessToken, refreshToken, err := GenerateUserTokenWithRefresh(1, "testuser")
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.NotEqual(t, accessToken, refreshToken)
}

func TestGenerateUserTokenWithRefresh_DefaultSecret(t *testing.T) {
	userJwtSecret = []byte{}
	InitJWT("admin-secret")
	
	accessToken, refreshToken, err := GenerateUserTokenWithRefresh(1, "testuser")
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	
	userJwtSecret = []byte("user-secret")
}

func TestParseToken_Invalid(t *testing.T) {
	InitJWT("test-secret")
	
	claims, err := ParseToken("invalid-token")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestParseToken_WrongSecret(t *testing.T) {
	InitJWT("secret1")
	token, _ := GenerateToken(1, "user1")
	
	InitJWT("secret2")
	claims, err := ParseToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
	
	InitJWT("secret1")
}

func TestParseUserToken_Invalid(t *testing.T) {
	InitUserJWT("user-secret")
	
	claims, err := ParseUserToken("invalid-token")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateRefreshToken_Valid(t *testing.T) {
	InitUserJWT("user-secret")
	
	_, refreshToken, _ := GenerateUserTokenWithRefresh(1, "testuser")
	
	claims, err := ValidateRefreshToken(refreshToken)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, uint(1), claims.UserID)
	assert.Equal(t, "hjtpx-user-refresh", claims.Issuer)
}

func TestValidateRefreshToken_Invalid(t *testing.T) {
	InitUserJWT("user-secret")
	
	claims, err := ValidateRefreshToken("invalid-token")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateRefreshToken_WrongIssuer(t *testing.T) {
	InitUserJWT("user-secret")
	
	accessToken, _, _ := GenerateUserTokenWithRefresh(1, "testuser")
	
	claims, err := ValidateRefreshToken(accessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestParseToken_Expired(t *testing.T) {
	InitJWT("test-secret")
	
	nowTime := time.Now()
	expireTime := nowTime.Add(-1 * time.Hour)
	
	claims := Claims{
		AdminID:  1,
		Username: "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			NotBefore: jwt.NewNumericDate(nowTime),
			Issuer:    "hjtpx-admin",
		},
	}
	
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, _ := tokenClaims.SignedString(jwtSecret)
	
	parsedClaims, err := ParseToken(token)
	assert.Error(t, err)
	assert.Nil(t, parsedClaims)
}

func TestParseToken_NotYetValid(t *testing.T) {
	InitJWT("test-secret")
	
	nowTime := time.Now()
	notBeforeTime := nowTime.Add(1 * time.Hour)
	expireTime := nowTime.Add(2 * time.Hour)
	
	claims := Claims{
		AdminID:  1,
		Username: "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			NotBefore: jwt.NewNumericDate(notBeforeTime),
			Issuer:    "hjtpx-admin",
		},
	}
	
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, _ := tokenClaims.SignedString(jwtSecret)
	
	parsedClaims, err := ParseToken(token)
	assert.Error(t, err)
	assert.Nil(t, parsedClaims)
}

func TestClaims_Structure(t *testing.T) {
	claims := Claims{
		AdminID:  42,
		Username: "adminuser",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "test-issuer",
		},
	}
	
	assert.Equal(t, uint(42), claims.AdminID)
	assert.Equal(t, "adminuser", claims.Username)
	assert.Equal(t, "test-issuer", claims.Issuer)
}

func TestUserTokenClaims_Structure(t *testing.T) {
	claims := UserTokenClaims{
		UserID:   42,
		Username: "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "user-issuer",
		},
	}
	
	assert.Equal(t, uint(42), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "user-issuer", claims.Issuer)
}

func TestParseUserToken_DefaultToAdminSecret(t *testing.T) {
	userJwtSecret = []byte{}
	InitJWT("admin-secret")
	
	token, _ := GenerateToken(1, "admin")
	
	claims, err := ParseUserToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	
	userJwtSecret = []byte("user-secret")
}

func TestValidateRefreshToken_DefaultToAdminSecret(t *testing.T) {
	userJwtSecret = []byte{}
	InitJWT("admin-secret")
	
	token, _ := GenerateToken(1, "admin")
	
	claims, err := ValidateRefreshToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
	
	userJwtSecret = []byte("user-secret")
}

func TestGenerateToken_Concurrent(t *testing.T) {
	InitJWT("test-secret")
	
	tokens := make([]string, 100)
	
	for i := 0; i < 100; i++ {
		token, err := GenerateToken(uint(i), "user")
		assert.NoError(t, err)
		tokens[i] = token
	}
	
	for i, token := range tokens {
		claims, err := ParseToken(token)
		assert.NoError(t, err)
		assert.Equal(t, uint(i), claims.AdminID)
	}
}

func TestGenerateUserToken_Concurrent(t *testing.T) {
	InitUserJWT("user-secret")
	
	tokens := make([]string, 100)
	
	for i := 0; i < 100; i++ {
		token, err := GenerateUserToken(uint(i), "user")
		assert.NoError(t, err)
		tokens[i] = token
	}
	
	for i, token := range tokens {
		claims, err := ParseUserToken(token)
		assert.NoError(t, err)
		assert.Equal(t, uint(i), claims.UserID)
	}
}

func TestParseToken_Empty(t *testing.T) {
	claims, err := ParseToken("")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestParseUserToken_Empty(t *testing.T) {
	claims, err := ParseUserToken("")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateRefreshToken_Empty(t *testing.T) {
	claims, err := ValidateRefreshToken("")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestErrTokenInvalid(t *testing.T) {
	assert.Equal(t, "token is invalid", ErrTokenInvalid.Error())
}

func TestErrTokenExpired(t *testing.T) {
	assert.Equal(t, "token is expired", ErrTokenExpired.Error())
}

func BenchmarkGenerateToken(b *testing.B) {
	InitJWT("benchmark-secret")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateToken(1, "user")
	}
}

func BenchmarkParseToken(b *testing.B) {
	InitJWT("benchmark-secret")
	token, _ := GenerateToken(1, "user")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseToken(token)
	}
}

func BenchmarkGenerateUserToken(b *testing.B) {
	InitUserJWT("user-secret")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateUserToken(1, "user")
	}
}

func BenchmarkParseUserToken(b *testing.B) {
	InitUserJWT("user-secret")
	token, _ := GenerateUserToken(1, "user")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseUserToken(token)
	}
}
