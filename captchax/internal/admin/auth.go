package admin

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"captchax/internal/model"
	"captchax/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AdminRepoInterface interface {
	GetByUsername(ctx context.Context, username string) (*model.Admin, error)
}

type AuthService struct {
	adminRepo  AdminRepoInterface
	jwtSecret  []byte
	tokenTTL    time.Duration
}

func NewAuthService(adminRepo AdminRepoInterface, jwtSecret string, tokenTTL time.Duration) *AuthService {
	return &AuthService{
		adminRepo: adminRepo,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  tokenTTL,
	}
}

type Claims struct {
	AdminID   uint   `json:"admin_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

func (s *AuthService) Login(ctx *gin.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
	admin, err := s.adminRepo.GetByUsername(ctx.Request.Context(), req.Username)
	if err != nil {
		return nil, err
	}
	if admin == nil {
		return nil, errors.New("invalid credentials")
	}

	if !admin.CheckPassword(req.Password) {
		return nil, errors.New("invalid credentials")
	}

	if admin.Status != 1 {
		return nil, errors.New("account disabled")
	}

	token, err := s.generateToken(admin)
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		Token: token,
		Admin: *admin,
	}, nil
}

func (s *AuthService) generateToken(admin *model.Admin) (string, error) {
	now := time.Now()
	claims := &Claims{
		AdminID:  admin.ID,
		Username: admin.Username,
		Role:     admin.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "captchax-admin",
			Subject:   admin.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (s *AuthService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Error(c, http.StatusBadRequest, "invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := s.ValidateToken(tokenString)
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set("admin_id", claims.AdminID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func (s *AuthService) SuperAdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}

		if role.(string) != string(model.AdminRoleSuper) {
			response.Forbidden(c, "super admin access required")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (s *AuthService) GetAdminID(c *gin.Context) uint {
	if id, exists := c.Get("admin_id"); exists {
		return id.(uint)
	}
	return 0
}

func (s *AuthService) GetUsername(c *gin.Context) string {
	if username, exists := c.Get("username"); exists {
		return username.(string)
	}
	return ""
}

func (s *AuthService) GetRole(c *gin.Context) string {
	if role, exists := c.Get("role"); exists {
		return role.(string)
	}
	return ""
}
