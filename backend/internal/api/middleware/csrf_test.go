package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCSRFProtection_GenerateToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection())
	router.GET("/test", func(c *gin.Context) {
		token := c.GetHeader("X-CSRF-Token")
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-CSRF-Token") == "" {
		t.Error("应该生成CSRF token")
	}
}

func TestCSRFProtection_VerifyValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection())
	
	var capturedToken string
	router.GET("/test", func(c *gin.Context) {
		capturedToken = c.GetHeader("X-CSRF-Token")
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
	
	req1, _ := http.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	
	token := w1.Header().Get("X-CSRF-Token")
	
	req2, _ := http.NewRequest("POST", "/test", nil)
	req2.Header.Set("X-CSRF-Token", token)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	
	if w2.Code != http.StatusOK {
		t.Errorf("有效token应该通过验证, 实际状态码: %d", w2.Code)
	}
}

func TestCSRFProtection_VerifyInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("POST", "/test", nil)
	req.Header.Set("X-CSRF-Token", "invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("无效token应该返回403, 实际状态码: %d", w.Code)
	}
}

func TestCSRFProtection_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("缺少token应该返回403, 实际状态码: %d", w.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Frame-Options") == "" {
		t.Error("应该设置X-Frame-Options header")
	}
	if w.Header().Get("X-Content-Type-Options") == "" {
		t.Error("应该设置X-Content-Type-Options header")
	}
	if w.Header().Get("X-XSS-Protection") == "" {
		t.Error("应该设置X-XSS-Protection header")
	}
	if w.Header().Get("Strict-Transport-Security") == "" {
		t.Error("应该设置Strict-Transport-Security header")
	}
}

func TestRequestIDGenerator(t *testing.T) {
	id1 := RequestIDGenerator()
	id2 := RequestIDGenerator()
	
	if id1 == "" {
		t.Error("生成的request ID不应为空")
	}
	if id1 == id2 {
		t.Error("两次生成的request ID应该不同")
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("应该设置X-Request-ID header")
	}
}

func TestRequestIDMiddleware_WithExistingID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "existing-id")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")
	if requestID != "existing-id" {
		t.Errorf("应该保留已存在的request ID, 期望 existing-id, 实际 %s", requestID)
	}
}

func TestErrorHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ErrorHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("正常请求应该返回200, 实际状态码: %d", w.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RecoveryMiddleware())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req, _ := http.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("panic应该返回500, 实际状态码: %d", w.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("应该设置Access-Control-Allow-Origin header")
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("preflight请求应该返回204, 实际状态码: %d", w.Code)
	}
}
