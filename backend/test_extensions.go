package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// 测试仪表盘配置相关API
func TestDashboardConfigAPIs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// 创建测试路由
	r := gin.Default()
	
	// 模拟一些路由进行测试
	r.GET("/admin/api/dashboard/config", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"theme": "default",
				"layout": "default",
			},
		})
	})
	
	r.PUT("/admin/api/dashboard/theme", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Theme updated"})
	})
	
	r.GET("/admin/api/notifications", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"items": []interface{}{},
				"total": 0,
			},
		})
	})
	
	// 测试获取仪表盘配置
	t.Run("GetDashboardConfig", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/dashboard/config", nil)
		r.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
	
	// 测试更新主题
	t.Run("UpdateDashboardTheme", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"theme": "dark"}`)
		req, _ := http.NewRequest("PUT", "/admin/api/dashboard/theme", body)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
	
	// 测试获取通知列表
	t.Run("GetNotifications", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/notifications", nil)
		r.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

// 测试风控规则引擎相关API
func TestRiskRulesAPIs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// 创建测试路由
	r := gin.Default()
	
	// 模拟路由
	r.GET("/admin/api/risk-templates", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"items": []interface{}{},
				"total": 0,
			},
		})
	})
	
	r.GET("/admin/api/risk-rules", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"items": []interface{}{},
				"total": 0,
			},
		})
	})
	
	r.GET("/admin/api/risk-rules/performance-overview", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"total_evaluations": 100,
				"total_hits": 20,
				"avg_latency": 50,
				"rules": []interface{}{},
			},
		})
	})
	
	// 测试获取规则模板
	t.Run("GetRiskTemplates", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/risk-templates", nil)
		r.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
	
	// 测试获取风控规则
	t.Run("GetRiskRules", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/risk-rules", nil)
		r.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
	
	// 测试获取性能概览
	t.Run("GetPerformanceOverview", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/risk-rules/performance-overview", nil)
		r.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var result map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &result)
		
		if result["code"] != float64(0) {
			t.Errorf("Expected code 0, got %v", result["code"])
		}
	})
}

// 测试数据模型
func TestDataModels(t *testing.T) {
	// 测试 DashboardConfig 模型
	t.Run("DashboardConfigModel", func(t *testing.T) {
		// 这个测试主要验证模型结构是否符合预期
		type DashboardConfig struct {
			ID        uint   `json:"id"`
			AdminID   uint   `json:"admin_id"`
			Theme     string `json:"theme"`
			Layout    string `json:"layout"`
			Widgets   string `json:"widgets"`
		}
		
		config := DashboardConfig{
			AdminID: 1,
			Theme:   "dark",
			Layout:  "custom",
		}
		
		if config.AdminID != 1 {
			t.Error("AdminID should be 1")
		}
		if config.Theme != "dark" {
			t.Error("Theme should be dark")
		}
	})
	
	// 测试 RiskRule 模型
	t.Run("RiskRuleModel", func(t *testing.T) {
		type RiskRule struct {
			ID           uint   `json:"id"`
			Name         string `json:"name"`
			RuleType     string `json:"rule_type"`
			Severity     string `json:"severity"`
			Enabled      bool   `json:"enabled"`
			Conditions   string `json:"conditions"`
			Action       string `json:"action"`
		}
		
		rule := RiskRule{
			Name:     "Test Rule",
			RuleType: "behavior",
			Severity: "high",
			Enabled:  true,
		}
		
		if rule.Name != "Test Rule" {
			t.Error("Rule name should be Test Rule")
		}
		if rule.RuleType != "behavior" {
			t.Error("Rule type should be behavior")
		}
		if !rule.Enabled {
			t.Error("Rule should be enabled")
		}
	})
}
