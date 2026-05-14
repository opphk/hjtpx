package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/opphk/captcha-system/pkg/captcha"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	service *captcha.AdminService
	jwt     *captcha.JWTService
}

func NewAdminHandler(service *captcha.AdminService, jwt *captcha.JWTService) *AdminHandler {
	return &AdminHandler{
		service: service,
		jwt:     jwt,
	}
}

func (h *AdminHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "Invalid request body")
		return
	}

	user, err := h.service.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil || user == nil {
		Error(c, 401, "Invalid credentials")
		return
	}

	if !user.IsActive {
		Error(c, 401, "Account is disabled")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		Error(c, 401, "Invalid credentials")
		return
	}

	tokenPair, err := h.jwt.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		Error(c, 500, "Failed to generate token")
		return
	}

	h.service.UpdateLastLogin(c.Request.Context(), user.ID)

	Success(c, map[string]interface{}{
		"token":        tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_at":   tokenPair.ExpiresAt,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

func (h *AdminHandler) Logout(c *gin.Context) {
	Success(c, map[string]interface{}{
		"message": "Logged out successfully",
	})
}

func (h *AdminHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		Error(c, 500, "Failed to get stats")
		return
	}

	Success(c, stats)
}

func (h *AdminHandler) GetChallenges(c *gin.Context) {
	page := 1
	size := 20

	if p := c.Query("page"); p != "" {
		if _, err := c.GetQuery("page"); err {
			page = 1
		}
	}
	if s := c.Query("size"); s != "" {
		if _, err := c.GetQuery("size"); err {
			size = 20
		}
	}

	var challengeType string
	if t := c.Query("type"); t != "" {
		challengeType = t
	}

	challenges, total, err := h.service.GetChallenges(c.Request.Context(), page, size, challengeType)
	if err != nil {
		Error(c, 500, "Failed to get challenges")
		return
	}

	Paginated(c, challenges, total, page, size)
}

func (h *AdminHandler) GetAttempts(c *gin.Context) {
	page := 1
	size := 20

	attempts, total, err := h.service.GetAttempts(c.Request.Context(), page, size)
	if err != nil {
		Error(c, 500, "Failed to get attempts")
		return
	}

	Paginated(c, attempts, total, page, size)
}

func (h *AdminHandler) UpdateConfig(c *gin.Context) {
	var req struct {
		Key   string      `json:"key" binding:"required"`
		Value interface{} `json:"value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "Invalid request body")
		return
	}

	if err := h.service.UpdateConfig(c.Request.Context(), req.Key, req.Value); err != nil {
		Error(c, 500, "Failed to update config")
		return
	}

	Success(c, map[string]interface{}{
		"message": "Config updated successfully",
	})
}

func (h *AdminHandler) GetLogs(c *gin.Context) {
	level := c.DefaultQuery("level", "info")
	page := 1
	size := 50

	logs, total, err := h.service.GetLogs(c.Request.Context(), level, page, size)
	if err != nil {
		Error(c, 500, "Failed to get logs")
		return
	}

	Paginated(c, logs, total, page, size)
}

type JWTAuthConfig struct {
	Secret       string
	ExpiresHours int
}

func JWTAuth(jwtConfig *JWTAuthConfig) gin.HandlerFunc {
	jwtService := captcha.NewJWTService(jwtConfig.Secret, jwtConfig.ExpiresHours)

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			ErrorWithStatus(c, http.StatusUnauthorized, 401, "Authorization header required")
			c.Abort()
			return
		}

		tokenStr := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenStr = authHeader[7:]
		} else {
			ErrorWithStatus(c, http.StatusUnauthorized, 401, "Invalid authorization format")
			c.Abort()
			return
		}

		claims, err := jwtService.ValidateToken(tokenStr)
		if err != nil {
			ErrorWithStatus(c, http.StatusUnauthorized, 401, "Invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func (h *AdminHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "Invalid request body")
		return
	}

	tokenPair, err := h.jwt.RefreshToken(req.RefreshToken)
	if err != nil {
		Error(c, 401, "Invalid refresh token")
		return
	}

	Success(c, map[string]interface{}{
		"token":        tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_at":   tokenPair.ExpiresAt,
	})
}

func (h *AdminHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		Error(c, 401, "User not authenticated")
		return
	}

	username, _ := c.Get("username")
	role, _ := c.Get("role")

	Success(c, map[string]interface{}{
		"id":       userID,
		"username": username,
		"role":     role,
	})
}
