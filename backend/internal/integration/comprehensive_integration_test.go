package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/jwt"
	"github.com/hjtpx/hjtpx/pkg/response"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestHealthEndpoint(t *testing.T) {
	r := setupTestRouter()
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "success", resp["message"])
	assert.Equal(t, "healthy", resp["data"].(map[string]interface{})["status"])
}

func TestJWTAuthentication(t *testing.T) {
	jwt.InitJWT("test-secret-key")

	token, err := jwt.GenerateToken(1, "testuser")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := jwt.ParseToken(token)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), claims.AdminID)
	assert.Equal(t, "testuser", claims.Username)
}

func TestUserTokenGeneration(t *testing.T) {
	jwt.InitUserJWT("user-secret-key")

	token, err := jwt.GenerateUserToken(1, "testuser")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := jwt.ParseUserToken(token)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
}

func TestRefreshTokenGeneration(t *testing.T) {
	jwt.InitUserJWT("user-secret-key")

	accessToken, refreshToken, err := jwt.GenerateUserTokenWithRefresh(1, "testuser")
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	claims, err := jwt.ValidateRefreshToken(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, uint(1), claims.UserID)
}

func TestResponseFormat(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		response.Success(c, gin.H{"key": "value"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.NotNil(t, resp.Data)
}

func TestErrorResponse(t *testing.T) {
	r := setupTestRouter()
	r.GET("/error", func(c *gin.Context) {
		response.BadRequest(c, "Invalid request")
	})

	req, _ := http.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.Code)
	assert.Equal(t, "Invalid request", resp.Message)
}

func TestUnauthorizedResponse(t *testing.T) {
	r := setupTestRouter()
	r.GET("/protected", func(c *gin.Context) {
		response.Unauthorized(c)
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 401, resp.Code)
}

func TestNotFoundResponse(t *testing.T) {
	r := setupTestRouter()
	r.GET("/notfound", func(c *gin.Context) {
		response.NotFound(c, "Resource not found")
	})

	req, _ := http.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.Code)
}

func TestInternalServerErrorResponse(t *testing.T) {
	r := setupTestRouter()
	r.GET("/servererror", func(c *gin.Context) {
		response.InternalServerError(c, "Internal error")
	})

	req, _ := http.NewRequest("GET", "/servererror", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 500, resp.Code)
}

func TestTooManyRequestsResponse(t *testing.T) {
	r := setupTestRouter()
	r.GET("/ratelimit", func(c *gin.Context) {
		response.TooManyRequests(c, "Rate limit exceeded")
	})

	req, _ := http.NewRequest("GET", "/ratelimit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 429, resp.Code)
}

func TestJSONRequest(t *testing.T) {
	r := setupTestRouter()
	r.POST("/json", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			response.BadRequest(c, "Invalid JSON")
			return
		}
		response.Success(c, data)
	})

	body := map[string]string{"key": "value"}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/json", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCORSHeaders(t *testing.T) {
	r := setupTestRouter()
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Next()
	})
	r.GET("/cors", func(c *gin.Context) {
		response.Success(c, nil)
	})

	req, _ := http.NewRequest("GET", "/cors", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
}

func TestPreflightRequest(t *testing.T) {
	r := setupTestRouter()
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Next()
	})
	r.OPTIONS("/preflight", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req, _ := http.NewRequest("OPTIONS", "/preflight", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestMultipleEndpoints(t *testing.T) {
	r := setupTestRouter()
	
	endpoints := []struct {
		path   string
		method string
	}{
		{"/api/users", "GET"},
		{"/api/applications", "POST"},
		{"/api/captchas", "GET"},
		{"/api/stats", "POST"},
	}

	for _, ep := range endpoints {
		handler := func(c *gin.Context) {
			response.Success(c, gin.H{"path": c.Request.URL.Path})
		}

		switch ep.method {
		case "GET":
			r.GET(ep.path, handler)
		case "POST":
			r.POST(ep.path, handler)
		}

		req, _ := http.NewRequest(ep.method, ep.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestResponseConstants(t *testing.T) {
	assert.Equal(t, 0, response.CodeSuccess)
	assert.Equal(t, 400, response.CodeInvalidParams)
	assert.Equal(t, 401, response.CodeUnauthorized)
	assert.Equal(t, 403, response.CodeForbidden)
	assert.Equal(t, 404, response.CodeNotFound)
	assert.Equal(t, 500, response.CodeServerError)
	assert.Equal(t, 429, response.CodeTooManyRequests)
}

func TestFailResponse(t *testing.T) {
	r := setupTestRouter()
	r.GET("/fail", func(c *gin.Context) {
		response.Fail(c, 1001, "Custom error")
	})

	req, _ := http.NewRequest("GET", "/fail", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 1001, resp.Code)
	assert.Equal(t, "Custom error", resp.Message)
}
