package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/stretchr/testify/assert"
)

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func(c *gin.Context)
		expected uint
	}{
		{
			name: "user_id from context",
			setupCtx: func(c *gin.Context) {
				c.Set("user_id", uint(123))
			},
			expected: 123,
		},
		{
			name: "admin_id from context when user_id not set",
			setupCtx: func(c *gin.Context) {
				c.Set("admin_id", uint(456))
			},
			expected: 456,
		},
		{
			name: "user_id takes precedence over admin_id",
			setupCtx: func(c *gin.Context) {
				c.Set("user_id", uint(789))
				c.Set("admin_id", uint(999))
			},
			expected: 789,
		},
		{
			name:     "no id set returns 0",
			setupCtx: func(c *gin.Context) {},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			result := GetUserID(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUsername(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func(c *gin.Context)
		expected string
	}{
		{
			name: "username from context",
			setupCtx: func(c *gin.Context) {
				c.Set("username", "testuser")
			},
			expected: "testuser",
		},
		{
			name:     "no username set returns empty string",
			setupCtx: func(c *gin.Context) {},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			result := GetUsername(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthMiddleware_WithValidToken(t *testing.T) {
	jwt.InitJWT("test-secret-key")

	token, err := jwt.GenerateToken(1, "testadmin")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	r := setupTestRouter()
	r.Use(AuthMiddleware())
	handlerCalled := false
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		username := GetUsername(c)
		userID := GetUserID(c)
		c.JSON(http.StatusOK, gin.H{
			"username": username,
			"user_id":  userID,
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
}

func TestUserAuthMiddleware_WithValidToken(t *testing.T) {
	jwt.InitUserJWT("test-secret-key")

	token, err := jwt.GenerateUserToken(1, "testuser")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	r := setupTestRouter()
	r.Use(UserAuthMiddleware())
	handlerCalled := false
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		username := GetUsername(c)
		userID := GetUserID(c)
		c.JSON(http.StatusOK, gin.H{
			"username": username,
			"user_id":  userID,
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
}

func TestRecordViolationMiddleware(t *testing.T) {
	r := setupTestRouter()
	r.Use(RecordViolationMiddleware("test_violation"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRecordFailedAttemptMiddleware(t *testing.T) {
	r := setupTestRouter()
	r.Use(RecordFailedAttemptMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClearFailedAttemptsMiddleware(t *testing.T) {
	r := setupTestRouter()
	r.Use(ClearFailedAttemptsMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
