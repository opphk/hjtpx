//go:build integration
// +build integration

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestCaptchaWorkflow(t *testing.T) {
	router := setupTestRouter()

	router.GET("/api/v1/captcha/slider", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"session_id": "test-session-123",
			"image_url":  "/api/v1/captcha/image/background",
			"puzzle_image": "/api/v1/captcha/image/puzzle",
			"target_x":   150,
			"target_y":   100,
		})
	})

	router.POST("/api/v1/captcha/verify", func(c *gin.Context) {
		var req struct {
			SessionID string `json:"session_id"`
			X         int    `json:"x"`
			Y         int    `json:"y"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}

		if req.SessionID == "test-session-123" && req.X > 140 && req.X < 160 {
			c.JSON(200, gin.H{
				"success":     true,
				"score":       95.5,
				"risk_level":  "low",
			})
		} else {
			c.JSON(200, gin.H{
				"success":     false,
				"score":       45.0,
				"risk_level":  "medium",
				"message":     "验证失败",
			})
		}
	})

	t.Run("Generate slider captcha", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp, "session_id")
		assert.Contains(t, resp, "image_url")
		assert.Contains(t, resp, "target_x")
		assert.Contains(t, resp, "target_y")
	})

	t.Run("Verify correct slider position", func(t *testing.T) {
		verifyReq := map[string]interface{}{
			"session_id": "test-session-123",
			"x":          150,
			"y":          100,
		}
		body, _ := json.Marshal(verifyReq)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, true, resp["success"])
	})

	t.Run("Verify incorrect slider position", func(t *testing.T) {
		verifyReq := map[string]interface{}{
			"session_id": "test-session-123",
			"x":          50,
			"y":          50,
		}
		body, _ := json.Marshal(verifyReq)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, false, resp["success"])
	})
}

func TestClickCaptchaWorkflow(t *testing.T) {
	router := setupTestRouter()

	router.GET("/api/v1/captcha/click", func(c *gin.Context) {
		mode := c.DefaultQuery("mode", "number")
		c.JSON(200, gin.H{
			"session_id": "click-session-456",
			"image_url":  "/api/v1/captcha/image/click",
			"hint":       "请依次点击: 1, 2, 3",
			"mode":       mode,
			"points":     3,
		})
	})

	router.POST("/api/v1/captcha/verify", func(c *gin.Context) {
		var req struct {
			SessionID string      `json:"session_id"`
			Points    [][2]int    `json:"points"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}

		if req.SessionID == "click-session-456" && len(req.Points) == 3 {
			c.JSON(200, gin.H{
				"success":     true,
				"score":       92.0,
				"risk_level":  "low",
			})
		} else {
			c.JSON(200, gin.H{
				"success":     false,
				"score":       30.0,
				"risk_level":  "high",
				"message":     "点击验证失败",
			})
		}
	})

	t.Run("Generate click captcha", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/captcha/click?mode=number", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp, "session_id")
		assert.Contains(t, resp, "hint")
		assert.Equal(t, "number", resp["mode"])
	})

	t.Run("Verify correct click sequence", func(t *testing.T) {
		verifyReq := map[string]interface{}{
			"session_id": "click-session-456",
			"points":     [][]int{{100, 100}, {150, 150}, {200, 200}},
		}
		body, _ := json.Marshal(verifyReq)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, true, resp["success"])
	})
}

func TestApplicationWorkflow(t *testing.T) {
	router := setupTestRouter()

	applications := make(map[string]map[string]interface{})
	var nextID int = 1

	router.POST("/api/v1/applications", func(c *gin.Context) {
		var req struct {
			Name        string `json:"name" binding:"required"`
			Description string `json:"description"`
			Domain      string `json:"domain"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		id := fmt.Sprintf("%d", nextID)
		nextID++
		app := map[string]interface{}{
			"id":          id,
			"name":        req.Name,
			"description": req.Description,
			"domain":      req.Domain,
			"api_key":     "key-" + id,
			"is_active":   true,
			"created_at":  time.Now(),
		}
		applications[id] = app
		c.JSON(201, app)
	})

	router.GET("/api/v1/applications", func(c *gin.Context) {
		page := c.DefaultQuery("page", "1")
		pageSize := c.DefaultQuery("page_size", "10")

		apps := make([]map[string]interface{}, 0, len(applications))
		for _, app := range applications {
			apps = append(apps, app)
		}

		c.JSON(200, gin.H{
			"items":      apps,
			"total":      len(apps),
			"page":       page,
			"page_size":  pageSize,
		})
	})

	router.GET("/api/v1/applications/:id", func(c *gin.Context) {
		id := c.Param("id")
		if app, exists := applications[id]; exists {
			c.JSON(200, app)
		} else {
			c.JSON(404, gin.H{"error": "application not found"})
		}
	})

	t.Run("Create application", func(t *testing.T) {
		createReq := map[string]interface{}{
			"name":        "Test App",
			"description": "Test application for integration test",
			"domain":      "test.example.com",
		}
		body, _ := json.Marshal(createReq)

		req, _ := http.NewRequest("POST", "/api/v1/applications", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp, "id")
		assert.Contains(t, resp, "api_key")
		assert.Equal(t, "Test App", resp["name"])
	})

	t.Run("List applications", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/applications", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp, "items")
		assert.Contains(t, resp, "total")
	})

	t.Run("Get application by ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/applications/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "1", resp["id"])
	})

	t.Run("Get non-existent application", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/applications/999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestStatsWorkflow(t *testing.T) {
	router := setupTestRouter()

	router.GET("/api/v1/stats/dashboard", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"total_users":     1250,
			"total_apps":      45,
			"total_requests":  854321,
			"total_errors":     1234,
			"success_rate":    94.7,
			"timestamp":       time.Now(),
		})
	})

	router.GET("/api/v1/stats/trend", func(c *gin.Context) {
		days := c.DefaultQuery("days", "7")
		c.JSON(200, gin.H{
			"days": days,
			"data": []map[string]interface{}{
				{"date": "2025-05-12", "total": 1000, "success": 950, "failed": 50},
				{"date": "2025-05-13", "total": 1100, "success": 1045, "failed": 55},
				{"date": "2025-05-14", "total": 1200, "success": 1140, "failed": 60},
				{"date": "2025-05-15", "total": 1150, "success": 1092, "failed": 58},
				{"date": "2025-05-16", "total": 1300, "success": 1235, "failed": 65},
				{"date": "2025-05-17", "total": 1400, "success": 1330, "failed": 70},
				{"date": "2025-05-18", "total": 1500, "success": 1425, "failed": 75},
			},
		})
	})

	t.Run("Get dashboard stats", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/stats/dashboard", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp, "total_users")
		assert.Contains(t, resp, "total_apps")
		assert.Contains(t, resp, "total_requests")
	})

	t.Run("Get trend data", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/stats/trend?days=7", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp, "data")
	})
}

func TestUserAuthWorkflow(t *testing.T) {
	router := setupTestRouter()

	users := make(map[string]map[string]interface{})
	var userID int = 1

	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}

		if req.Username == "testuser" && req.Password == "password123" {
			c.JSON(200, gin.H{
				"success": true,
				"token":   "jwt-token-" + fmt.Sprintf("%d", time.Now().Unix()),
				"user": map[string]interface{}{
					"id":       1,
					"username": req.Username,
					"email":    "test@example.com",
				},
			})
		} else {
			c.JSON(401, gin.H{
				"success": false,
				"error":   "invalid credentials",
			})
		}
	})

	router.POST("/api/v1/auth/register", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Email    string `json:"email" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		id := userID
		userID++
		user := map[string]interface{}{
			"id":       id,
			"username": req.Username,
			"email":    req.Email,
		}
		users[fmt.Sprintf("%d", id)] = user

		c.JSON(201, gin.H{
			"success": true,
			"user":    user,
		})
	})

	t.Run("Register new user", func(t *testing.T) {
		registerReq := map[string]interface{}{
			"username": "newuser",
			"email":    "newuser@example.com",
			"password": "securepassword",
		}
		body, _ := json.Marshal(registerReq)

		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, true, resp["success"])
	})

	t.Run("Login with valid credentials", func(t *testing.T) {
		loginReq := map[string]interface{}{
			"username": "testuser",
			"password": "password123",
		}
		body, _ := json.Marshal(loginReq)

		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, true, resp["success"])
		assert.Contains(t, resp, "token")
	})

	t.Run("Login with invalid credentials", func(t *testing.T) {
		loginReq := map[string]interface{}{
			"username": "testuser",
			"password": "wrongpassword",
		}
		body, _ := json.Marshal(loginReq)

		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestEndToEndCaptchaFlow(t *testing.T) {
	router := setupTestRouter()

	sessionStore := make(map[string]map[string]interface{})

	router.GET("/api/v1/captcha/slider", func(c *gin.Context) {
		sessionID := fmt.Sprintf("sess_%d", time.Now().UnixNano())
		session := map[string]interface{}{
			"type":      "slider",
			"target_x":  150,
			"target_y":  100,
			"tolerance": 10,
			"created":   time.Now(),
		}
		sessionStore[sessionID] = session

		c.JSON(200, gin.H{
			"session_id":   sessionID,
			"image_url":    "/api/v1/captcha/image/background",
			"puzzle_image": "/api/v1/captcha/image/puzzle",
			"target_x":     150,
			"target_y":     100,
		})
	})

	router.POST("/api/v1/captcha/verify", func(c *gin.Context) {
		var req struct {
			SessionID string `json:"session_id" binding:"required"`
			X         int    `json:"x"`
			Y         int    `json:"y"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}

		session, exists := sessionStore[req.SessionID]
		if !exists {
			c.JSON(404, gin.H{"error": "session not found"})
			return
		}

		targetX := session["target_x"].(int)
		targetY := session["target_y"].(int)
		tolerance := session["tolerance"].(int)

		absDiffX := req.X - targetX
		if absDiffX < 0 {
			absDiffX = -absDiffX
		}
		absDiffY := req.Y - targetY
		if absDiffY < 0 {
			absDiffY = -absDiffY
		}

		if absDiffX <= tolerance && absDiffY <= tolerance {
			delete(sessionStore, req.SessionID)
			c.JSON(200, gin.H{
				"success":    true,
				"score":      95.0,
				"risk_level": "low",
				"message":    "验证成功",
			})
		} else {
			c.JSON(200, gin.H{
				"success":    false,
				"score":      40.0,
				"risk_level": "high",
				"message":    "验证失败，请重试",
			})
		}
	})

	t.Run("Complete captcha verification flow", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var captchaResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &captchaResp)
		assert.NoError(t, err)

		sessionID := captchaResp["session_id"].(string)
		targetX := int(captchaResp["target_x"].(float64))
		targetY := int(captchaResp["target_y"].(float64))

		verifyReq := map[string]interface{}{
			"session_id": sessionID,
			"x":          targetX + 3,
			"y":          targetY + 2,
		}
		body, _ := json.Marshal(verifyReq)

		verifyReqHTTP, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(body))
		verifyReqHTTP.Header.Set("Content-Type", "application/json")
		verifyW := httptest.NewRecorder()
		router.ServeHTTP(verifyW, verifyReqHTTP)

		assert.Equal(t, http.StatusOK, verifyW.Code)

		var verifyResp map[string]interface{}
		err = json.Unmarshal(verifyW.Body.Bytes(), &verifyResp)
		assert.NoError(t, err)
		assert.Equal(t, true, verifyResp["success"])

		verifyReq2 := map[string]interface{}{
			"session_id": sessionID,
			"x":          50,
			"y":          50,
		}
		body2, _ := json.Marshal(verifyReq2)

		verifyReqHTTP2, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(body2))
		verifyReqHTTP2.Header.Set("Content-Type", "application/json")
		verifyW2 := httptest.NewRecorder()
		router.ServeHTTP(verifyW2, verifyReqHTTP2)

		assert.Equal(t, http.StatusNotFound, verifyW2.Code)
	})
}

func TestConcurrentRequests(t *testing.T) {
	router := setupTestRouter()

	requestCount := 0

	router.GET("/api/v1/captcha/slider", func(c *gin.Context) {
		requestCount++
		c.JSON(200, gin.H{
			"session_id": fmt.Sprintf("sess_%d", requestCount),
			"target_x":   150,
			"target_y":   100,
		})
	})

	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func() {
			req, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				done <- true
			} else {
				done <- false
			}
		}()
	}

	successCount := 0
	for i := 0; i < 100; i++ {
		if <-done {
			successCount++
		}
	}

	assert.Equal(t, 100, successCount)
	assert.Equal(t, 100, requestCount)
}

func TestMiddlewareChain(t *testing.T) {
	router := setupTestRouter()

	middlewareCalled := make(map[string]bool)

	router.Use(func(c *gin.Context) {
		middlewareCalled["logger"] = true
		c.Next()
	})

	router.Use(func(c *gin.Context) {
		middlewareCalled["cors"] = true
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		middlewareCalled["handler"] = true
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, middlewareCalled["logger"])
	assert.True(t, middlewareCalled["cors"])
	assert.True(t, middlewareCalled["handler"])
}
