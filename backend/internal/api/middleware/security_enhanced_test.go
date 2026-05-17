package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDDoSEnhancedProtectionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("DefaultConfig", func(t *testing.T) {
		router := gin.New()
		router.Use(DDoSEnhancedProtectionMiddleware())

		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get("X-IP-Trust-Level"))
	})

	t.Run("ExcludedPath", func(t *testing.T) {
		router := gin.New()
		router.Use(DDoSEnhancedProtectionMiddleware(DDoSEnhancedConfig{
			ExcludePaths: []string{"/health"},
		}))

		router.GET("/health", func(c *gin.Context) {
			c.String(http.StatusOK, "healthy")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Disabled", func(t *testing.T) {
		router := gin.New()
		router.Use(DDoSEnhancedProtectionMiddleware(DDoSEnhancedConfig{
			Enabled: false,
		}))

		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestDDoSStatsHandler_GetStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &DDoSStatsHandler{}
	router.GET("/stats", handler.GetStats)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "total_requests")
	assert.Contains(t, response, "active_ips")
	assert.Contains(t, response, "timestamp")
}

func TestDDoSStatsHandler_GetIPReputation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &DDoSStatsHandler{}
	router.GET("/reputation", handler.GetIPReputation)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/reputation?ip=192.168.1.1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.1", response["ip"])
	assert.Contains(t, response, "score")
	assert.Contains(t, response, "tier")
}

func TestDDoSStatsHandler_SetIPWhitelist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &DDoSStatsHandler{}
	router.POST("/whitelist", handler.SetIPWhitelist)

	body := `{"ip": "192.168.1.100", "whitelisted": true}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/whitelist", nil)
	req.Body = createRequestBody(body)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.100", response["ip"])
	assert.Equal(t, true, response["whitelisted"])
}

func TestDDoSStatsHandler_SetIPBlacklist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &DDoSStatsHandler{}
	router.POST("/blacklist", handler.SetIPBlacklist)

	body := `{"ip": "192.168.1.101", "blacklisted": true}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/blacklist", nil)
	req.Body = createRequestBody(body)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDDoSStatsHandler_GetRateLimitStrategy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &DDoSStatsHandler{}
	router.GET("/strategy", handler.GetRateLimitStrategy)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/strategy", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "name")
	assert.Contains(t, response, "requests_per_sec")
	assert.Contains(t, response, "burst_size")
}

func TestDDoSStatsHandler_UpdateRateLimitStrategy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &DDoSStatsHandler{}
	router.POST("/strategy", handler.UpdateRateLimitStrategy)

	body := `{"name": "strict", "requests_per_sec": 50, "burst_size": 100, "adaptive": false, "block_duration_secs": 600}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/strategy", nil)
	req.Body = createRequestBody(body)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetDDoSEnhancedService(t *testing.T) {
	service := GetDDoSEnhancedService()
	assert.NotNil(t, service)
}

func TestCrawlerEnhancedDetectionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("DefaultConfig", func(t *testing.T) {
		router := gin.New()
		router.Use(CrawlerEnhancedDetectionMiddleware())

		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get("X-Crawler-Risk-Level"))
		assert.NotEmpty(t, w.Header().Get("X-Crawler-Confidence"))
	})

	t.Run("ExcludedPath", func(t *testing.T) {
		router := gin.New()
		router.Use(CrawlerEnhancedDetectionMiddleware(CrawlerEnhancedConfig{
			ExcludePaths: []string{"/public"},
		}))

		router.GET("/public", func(c *gin.Context) {
			c.String(http.StatusOK, "public")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/public", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Disabled", func(t *testing.T) {
		router := gin.New()
		router.Use(CrawlerEnhancedDetectionMiddleware(CrawlerEnhancedConfig{
			Enabled: false,
		}))

		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("BlockHighRiskCrawler", func(t *testing.T) {
		router := gin.New()
		router.Use(CrawlerEnhancedDetectionMiddleware(CrawlerEnhancedConfig{
			ChallengeMode: false,
		}))

		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "curl/7.68.0")
		req.Header.Set("Webdriver", "true")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestCrawlerStatsHandler_GetStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &CrawlerStatsHandler{}
	router.GET("/crawler/stats", handler.GetStats)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/crawler/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "tracked_ips")
}

func TestCrawlerStatsHandler_GetIPHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &CrawlerStatsHandler{}
	router.GET("/crawler/history", handler.GetIPHistory)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/crawler/history?ip=192.168.1.1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.1", response["ip"])
}

func TestCrawlerStatsHandler_ClearIPHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &CrawlerStatsHandler{}
	router.DELETE("/crawler/history", handler.ClearIPHistory)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/crawler/history?ip=192.168.1.1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCrawlerStatsHandler_AddKnownBot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &CrawlerStatsHandler{}
	router.POST("/crawler/bot", handler.AddKnownBot)

	body := `{"bot_name": "testbot", "bot_type": "api_proxy", "allow_api": true, "confidence": 0.8}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/crawler/bot", nil)
	req.Body = createRequestBody(body)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetCrawlerEnhancedService(t *testing.T) {
	service := GetCrawlerEnhancedService()
	assert.NotNil(t, service)
}

func TestSecurityEnhancedAuditMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("DefaultConfig", func(t *testing.T) {
		router := gin.New()
		router.Use(SecurityEnhancedAuditMiddleware())

		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("ExcludedPath", func(t *testing.T) {
		router := gin.New()
		router.Use(SecurityEnhancedAuditMiddleware(SecurityEnhancedAuditConfig{
			ExcludePaths: []string{"/health"},
		}))

		router.GET("/health", func(c *gin.Context) {
			c.String(http.StatusOK, "healthy")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Disabled", func(t *testing.T) {
		router := gin.New()
		router.Use(SecurityEnhancedAuditMiddleware(SecurityEnhancedAuditConfig{
			Enabled: false,
		}))

		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("LogHeaders", func(t *testing.T) {
		router := gin.New()
		router.Use(SecurityEnhancedAuditMiddleware(SecurityEnhancedAuditConfig{
			LogHeaders: true,
		}))

		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Custom-Header", "test")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSecurityAuditHandler_GetRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &SecurityAuditHandler{}
	router.GET("/audit/records", handler.GetRecords)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit/records", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "records")
	assert.Contains(t, response, "count")
}

func TestSecurityAuditHandler_GetStatistics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &SecurityAuditHandler{}
	router.GET("/audit/stats", handler.GetStatistics)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "total_records")
	assert.Contains(t, response, "by_category")
	assert.Contains(t, response, "by_severity")
}

func TestSecurityAuditHandler_GetAnomalyPatterns(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &SecurityAuditHandler{}
	router.GET("/audit/anomalies", handler.GetAnomalyPatterns)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit/anomalies", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "patterns")
}

func TestSecurityAuditHandler_UpdateAnomalyPattern(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &SecurityAuditHandler{}
	router.PUT("/audit/anomalies", handler.UpdateAnomalyPattern)

	body := `{"pattern_id": "rapid_requests", "enabled": false}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/audit/anomalies", nil)
	req.Body = createRequestBody(body)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSecurityAuditHandler_GetComplianceRules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &SecurityAuditHandler{}
	router.GET("/audit/compliance", handler.GetComplianceRules)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit/compliance", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "rules")
}

func TestSecurityAuditHandler_UpdateComplianceRule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &SecurityAuditHandler{}
	router.PUT("/audit/compliance", handler.UpdateComplianceRule)

	body := `{"rule_id": "gdpr_data_access", "enabled": false}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/audit/compliance", nil)
	req.Body = createRequestBody(body)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSecurityAuditHandler_CheckCompliance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &SecurityAuditHandler{}
	router.POST("/audit/check", handler.CheckCompliance)

	body := `{"path": "/api/user/123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/audit/check", nil)
	req.Body = createRequestBody(body)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "violations")
	assert.Contains(t, response, "is_compliant")
}

func TestSecurityAuditHandler_ExportRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &SecurityAuditHandler{}
	router.GET("/audit/export", handler.ExportRecords)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit/export?format=json", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.Bytes())
}

func TestSecurityAuditHandler_ExportRecordsCSV(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := &SecurityAuditHandler{}
	router.GET("/audit/export/csv", handler.ExportRecords)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit/export/csv?format=csv", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.Bytes())
}

func TestGetSecurityEnhancedAuditService(t *testing.T) {
	service := GetSecurityEnhancedAuditService()
	assert.NotNil(t, service)
}

func TestFormatRetryAfter(t *testing.T) {
	result := formatRetryAfter(300)
	assert.Contains(t, result, "s")
}

func TestFormatInt(t *testing.T) {
	result := formatInt(5)
	assert.NotEmpty(t, result)
}

func TestStringsHasPrefix(t *testing.T) {
	assert.True(t, stringsHasPrefix("/api/test", "/api"))
	assert.False(t, stringsHasPrefix("/other/test", "/api"))
}

func TestGetUserIDFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("WithUserID", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", "12345")

		userID := getUserIDFromContext(c)
		assert.Equal(t, "12345", userID)
	})

	t.Run("WithUserIDAlternative", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("userID", "67890")

		userID := getUserIDFromContext(c)
		assert.Equal(t, "67890", userID)
	})

	t.Run("WithoutUserID", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		userID := getUserIDFromContext(c)
		assert.Empty(t, userID)
	})
}

func createRequestBody(content string) *stringReader {
	return &stringReader{content: content}
}

type stringReader struct {
	content string
	pos    int
}

func (r *stringReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.content) {
		return 0, nil
	}
	n = copy(p, r.content[r.pos:])
	r.pos += n
	return n, nil
}

func (r *stringReader) Close() error {
	return nil
}
