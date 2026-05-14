package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"captchax/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

type MockAdminRepo struct {
	admins map[string]*model.Admin
}

func NewMockAdminRepo() *MockAdminRepo {
	return &MockAdminRepo{
		admins: map[string]*model.Admin{},
	}
}

func (r *MockAdminRepo) GetByUsername(ctx context.Context, username string) (*model.Admin, error) {
	if admin, ok := r.admins[username]; ok {
		return admin, nil
	}
	return nil, nil
}

func (r *MockAdminRepo) Add(admin *model.Admin) {
	r.admins[admin.Username] = admin
}

func TestLogin(t *testing.T) {
	jwtSecret := "test-secret-key"
	tokenTTL := 1 * time.Hour

	t.Run("Login with valid credentials", func(t *testing.T) {
		mockRepo := NewMockAdminRepo()
		admin := &model.Admin{
			ID:           1,
			Username:     "admin",
			PasswordHash: "",
			Role:         "super",
			Status:       1,
		}
		admin.SetPassword("password123")
		mockRepo.Add(admin)

		authService := &AuthService{
			adminRepo: mockRepo,
			jwtSecret: []byte(jwtSecret),
			tokenTTL:   tokenTTL,
		}

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request, _ = http.NewRequest("POST", "/login", nil)

		req := &model.LoginRequest{
			Username: "admin",
			Password: "password123",
		}

		resp, err := authService.Login(c, req)
		if err != nil {
			t.Fatalf("Login() error = %v", err)
		}

		if resp.Token == "" {
			t.Error("Login() returned empty token")
		}

		if resp.Admin.Username != "admin" {
			t.Errorf("Admin username = %s, want 'admin'", resp.Admin.Username)
		}
	})

	t.Run("Login with invalid username", func(t *testing.T) {
		mockRepo := NewMockAdminRepo()

		authService := &AuthService{
			adminRepo: mockRepo,
			jwtSecret: []byte(jwtSecret),
			tokenTTL:   tokenTTL,
		}

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request, _ = http.NewRequest("POST", "/login", nil)

		req := &model.LoginRequest{
			Username: "nonexistent",
			Password: "password123",
		}

		_, err := authService.Login(c, req)
		if err == nil {
			t.Error("Login() expected error for invalid username")
		}
	})

	t.Run("Login with invalid password", func(t *testing.T) {
		mockRepo := NewMockAdminRepo()
		admin := &model.Admin{
			ID:           1,
			Username:     "admin",
			PasswordHash: "",
			Role:         "super",
			Status:       1,
		}
		admin.SetPassword("correctpassword")
		mockRepo.Add(admin)

		authService := &AuthService{
			adminRepo: mockRepo,
			jwtSecret: []byte(jwtSecret),
			tokenTTL:   tokenTTL,
		}

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request, _ = http.NewRequest("POST", "/login", nil)

		req := &model.LoginRequest{
			Username: "admin",
			Password: "wrongpassword",
		}

		_, err := authService.Login(c, req)
		if err == nil {
			t.Error("Login() expected error for invalid password")
		}
	})

	t.Run("Login with disabled account", func(t *testing.T) {
		mockRepo := NewMockAdminRepo()
		admin := &model.Admin{
			ID:           1,
			Username:     "disabled",
			PasswordHash: "",
			Role:         "admin",
			Status:       0,
		}
		admin.SetPassword("password123")
		mockRepo.Add(admin)

		authService := &AuthService{
			adminRepo: mockRepo,
			jwtSecret: []byte(jwtSecret),
			tokenTTL:   tokenTTL,
		}

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request, _ = http.NewRequest("POST", "/login", nil)

		req := &model.LoginRequest{
			Username: "disabled",
			Password: "password123",
		}

		_, err := authService.Login(c, req)
		if err == nil {
			t.Error("Login() expected error for disabled account")
		}

		if err.Error() != "account disabled" {
			t.Errorf("Error message = %s, want 'account disabled'", err.Error())
		}
	})
}

func TestTokenValidation(t *testing.T) {
	jwtSecret := []byte("test-secret-key")
	tokenTTL := 1 * time.Hour

	t.Run("Validate valid token", func(t *testing.T) {
		admin := &model.Admin{
			ID:       1,
			Username: "admin",
			Role:     "super",
		}

		now := time.Now()
		claims := &Claims{
			AdminID:  admin.ID,
			Username: admin.Username,
			Role:     admin.Role,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
				IssuedAt:  jwt.NewNumericDate(now),
				NotBefore: jwt.NewNumericDate(now),
				Issuer:    "captchax-admin",
				Subject:   admin.Username,
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString(jwtSecret)

		authService := &AuthService{
			jwtSecret: jwtSecret,
			tokenTTL:  tokenTTL,
		}

		validatedClaims, err := authService.ValidateToken(tokenString)
		if err != nil {
			t.Fatalf("ValidateToken() error = %v", err)
		}

		if validatedClaims.AdminID != admin.ID {
			t.Errorf("AdminID = %d, want %d", validatedClaims.AdminID, admin.ID)
		}

		if validatedClaims.Username != admin.Username {
			t.Errorf("Username = %s, want %s", validatedClaims.Username, admin.Username)
		}

		if validatedClaims.Role != admin.Role {
			t.Errorf("Role = %s, want %s", validatedClaims.Role, admin.Role)
		}
	})

	t.Run("Validate expired token", func(t *testing.T) {
		now := time.Now()
		claims := &Claims{
			AdminID:  1,
			Username: "admin",
			Role:     "admin",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
				NotBefore: jwt.NewNumericDate(now.Add(-2 * time.Hour)),
				Issuer:    "captchax-admin",
				Subject:   "admin",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString(jwtSecret)

		authService := &AuthService{
			jwtSecret: jwtSecret,
			tokenTTL:  tokenTTL,
		}

		_, err := authService.ValidateToken(tokenString)
		if err == nil {
			t.Error("ValidateToken() expected error for expired token")
		}
	})

	t.Run("Validate token with wrong signing method", func(t *testing.T) {
		now := time.Now()
		claims := &Claims{
			AdminID:  1,
			Username: "admin",
			Role:     "admin",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
				IssuedAt:  jwt.NewNumericDate(now),
				NotBefore: jwt.NewNumericDate(now),
				Issuer:    "captchax-admin",
				Subject:   "admin",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		tokenString, _ := token.SignedString(jwtSecret)

		authService := &AuthService{
			jwtSecret: jwtSecret,
			tokenTTL:  tokenTTL,
		}

		_, err := authService.ValidateToken(tokenString)
		if err == nil {
			t.Error("ValidateToken() expected error for wrong signing method")
		}
	})

	t.Run("Validate token with wrong secret", func(t *testing.T) {
		now := time.Now()
		claims := &Claims{
			AdminID:  1,
			Username: "admin",
			Role:     "admin",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
				IssuedAt:  jwt.NewNumericDate(now),
				NotBefore: jwt.NewNumericDate(now),
				Issuer:    "captchax-admin",
				Subject:   "admin",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("wrong-secret"))

		authService := &AuthService{
			jwtSecret: jwtSecret,
			tokenTTL:  tokenTTL,
		}

		_, err := authService.ValidateToken(tokenString)
		if err == nil {
			t.Error("ValidateToken() expected error for wrong secret")
		}
	})

	t.Run("Validate malformed token", func(t *testing.T) {
		authService := &AuthService{
			jwtSecret: jwtSecret,
			tokenTTL:  tokenTTL,
		}

		_, err := authService.ValidateToken("malformed.token.string")
		if err == nil {
			t.Error("ValidateToken() expected error for malformed token")
		}
	})

	t.Run("Validate empty token", func(t *testing.T) {
		authService := &AuthService{
			jwtSecret: jwtSecret,
			tokenTTL:  tokenTTL,
		}

		_, err := authService.ValidateToken("")
		if err == nil {
			t.Error("ValidateToken() expected error for empty token")
		}
	})
}

func TestAuthMiddleware(t *testing.T) {
	jwtSecret := []byte("test-secret-key")
	tokenTTL := 1 * time.Hour

	t.Run("Valid authorization header", func(t *testing.T) {
		now := time.Now()
		claims := &Claims{
			AdminID:  1,
			Username: "admin",
			Role:     "super",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
				IssuedAt:  jwt.NewNumericDate(now),
				NotBefore: jwt.NewNumericDate(now),
				Issuer:    "captchax-admin",
				Subject:   "admin",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString(jwtSecret)

		authService := &AuthService{
			jwtSecret: jwtSecret,
			tokenTTL:  tokenTTL,
		}

		router := gin.New()
		router.Use(authService.AuthMiddleware())
		router.GET("/protected", func(c *gin.Context) {
			adminID := authService.GetAdminID(c)
			username := authService.GetUsername(c)
			role := authService.GetRole(c)
			c.JSON(200, gin.H{
				"admin_id": adminID,
				"username": username,
				"role":     role,
			})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("Missing authorization header", func(t *testing.T) {
		authService := &AuthService{
			jwtSecret: jwtSecret,
			tokenTTL:  tokenTTL,
		}

		router := gin.New()
		router.Use(authService.AuthMiddleware())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("Invalid authorization header format", func(t *testing.T) {
		authService := &AuthService{
			jwtSecret: jwtSecret,
			tokenTTL:  tokenTTL,
		}

		router := gin.New()
		router.Use(authService.AuthMiddleware())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
		}

		if !strings.Contains(w.Body.String(), "invalid authorization header format") {
			t.Error("Response should contain 'invalid authorization header format' message")
		}
	})

	t.Run("Invalid token in authorization header", func(t *testing.T) {
		authService := &AuthService{
			jwtSecret: jwtSecret,
			tokenTTL:  tokenTTL,
		}

		router := gin.New()
		router.Use(authService.AuthMiddleware())
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})
}

func TestSuperAdminOnly(t *testing.T) {
	jwtSecret := []byte("test-secret-key")
	tokenTTL := 1 * time.Hour

	t.Run("Super admin access allowed", func(t *testing.T) {
		now := time.Now()
		claims := &Claims{
			AdminID:  1,
			Username: "superadmin",
			Role:     string(model.AdminRoleSuper),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
				IssuedAt:  jwt.NewNumericDate(now),
				NotBefore: jwt.NewNumericDate(now),
				Issuer:    "captchax-admin",
				Subject:   "superadmin",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString(jwtSecret)

		authService := &AuthService{
			jwtSecret: jwtSecret,
			tokenTTL:  tokenTTL,
		}

		router := gin.New()
		router.Use(authService.AuthMiddleware())
		router.Use(authService.SuperAdminOnly())
		router.GET("/admin/only", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "super admin access granted"})
		})

		req, _ := http.NewRequest("GET", "/admin/only", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("Non-super admin access forbidden", func(t *testing.T) {
		now := time.Now()
		claims := &Claims{
			AdminID:  2,
			Username: "regularadmin",
			Role:     string(model.AdminRoleAdmin),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
				IssuedAt:  jwt.NewNumericDate(now),
				NotBefore: jwt.NewNumericDate(now),
				Issuer:    "captchax-admin",
				Subject:   "regularadmin",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString(jwtSecret)

		authService := &AuthService{
			jwtSecret: jwtSecret,
			tokenTTL:  tokenTTL,
		}

		router := gin.New()
		router.Use(authService.AuthMiddleware())
		router.Use(authService.SuperAdminOnly())
		router.GET("/admin/only", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "super admin access granted"})
		})

		req, _ := http.NewRequest("GET", "/admin/only", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusForbidden)
		}
	})
}

func TestGenerateToken(t *testing.T) {
	authService := &AuthService{
		jwtSecret: []byte("test-secret"),
		tokenTTL:  1 * time.Hour,
	}

	t.Run("Generate token for admin", func(t *testing.T) {
		admin := &model.Admin{
			ID:       1,
			Username: "testadmin",
			Role:     "admin",
		}

		tokenString, err := authService.generateToken(admin)
		if err != nil {
			t.Fatalf("generateToken() error = %v", err)
		}

		if tokenString == "" {
			t.Error("generateToken() returned empty token")
		}

		if !strings.Contains(tokenString, ".") {
			t.Error("generateToken() returned invalid JWT format")
		}
	})

	t.Run("Generated token is valid", func(t *testing.T) {
		admin := &model.Admin{
			ID:       1,
			Username: "testadmin",
			Role:     "admin",
		}

		tokenString, _ := authService.generateToken(admin)

		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			t.Fatalf("ValidateToken() error = %v", err)
		}

		if claims.AdminID != admin.ID {
			t.Errorf("AdminID = %d, want %d", claims.AdminID, admin.ID)
		}
	})
}

func TestHelperMethods(t *testing.T) {
	authService := &AuthService{}

	t.Run("GetAdminID with value", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("admin_id", uint(123))

		id := authService.GetAdminID(c)
		if id != 123 {
			t.Errorf("GetAdminID() = %d, want 123", id)
		}
	})

	t.Run("GetAdminID without value", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		id := authService.GetAdminID(c)
		if id != 0 {
			t.Errorf("GetAdminID() = %d, want 0", id)
		}
	})

	t.Run("GetUsername with value", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("username", "testuser")

		username := authService.GetUsername(c)
		if username != "testuser" {
			t.Errorf("GetUsername() = %s, want 'testuser'", username)
		}
	})

	t.Run("GetUsername without value", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		username := authService.GetUsername(c)
		if username != "" {
			t.Errorf("GetUsername() = %s, want empty string", username)
		}
	})

	t.Run("GetRole with value", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("role", "admin")

		role := authService.GetRole(c)
		if role != "admin" {
			t.Errorf("GetRole() = %s, want 'admin'", role)
		}
	})

	t.Run("GetRole without value", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		role := authService.GetRole(c)
		if role != "" {
			t.Errorf("GetRole() = %s, want empty string", role)
		}
	})
}
