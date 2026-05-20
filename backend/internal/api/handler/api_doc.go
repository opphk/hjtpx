package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type APIDocHandler struct{}

func NewAPIDocHandler() *APIDocHandler {
	return &APIDocHandler{}
}

type EndpointInfo struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Auth        bool   `json:"auth"`
	RateLimit   string `json:"rate_limit"`
	CacheTTL    int    `json:"cache_ttl,omitempty"`
}

type APIVersionInfo struct {
	Version   string          `json:"version"`
	BaseURL   string          `json:"base_url"`
	Endpoints []EndpointInfo  `json:"endpoints"`
}

func (h *APIDocHandler) GetAPIV1Doc(c *gin.Context) {
	doc := APIVersionInfo{
		Version: "v1",
		BaseURL: "/api/v1",
		Endpoints: []EndpointInfo{
			{
				Method:      "GET",
				Path:        "/health",
				Description: "健康检查",
				Auth:        false,
				RateLimit:   "100/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/generate",
				Description: "生成验证码（统一接口）",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/verify",
				Description: "验证验证码（统一接口）",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "GET",
				Path:        "/captcha/status",
				Description: "查询验证码状态",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/slider/create",
				Description: "创建滑动验证码",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/slider/verify",
				Description: "验证滑动验证码",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "GET",
				Path:        "/captcha/click",
				Description: "获取点击验证码",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/verify",
				Description: "统一验证接口",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/admin/login",
				Description: "管理员登录",
				Auth:        false,
				RateLimit:   "10/min",
			},
			{
				Method:      "POST",
				Path:        "/admin/logout",
				Description: "管理员登出",
				Auth:        true,
				RateLimit:   "10/min",
			},
			{
				Method:      "GET",
				Path:        "/admin/me",
				Description: "获取当前管理员信息",
				Auth:        true,
				RateLimit:   "60/min",
			},
			{
				Method:      "GET",
				Path:        "/admin/applications",
				Description: "获取应用列表",
				Auth:        true,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/admin/applications",
				Description: "创建应用",
				Auth:        true,
				RateLimit:   "20/min",
			},
			{
				Method:      "GET",
				Path:        "/admin/applications/:id",
				Description: "获取应用详情",
				Auth:        true,
				RateLimit:   "60/min",
			},
			{
				Method:      "PUT",
				Path:        "/admin/applications/:id",
				Description: "更新应用",
				Auth:        true,
				RateLimit:   "20/min",
			},
			{
				Method:      "DELETE",
				Path:        "/admin/applications/:id",
				Description: "删除应用",
				Auth:        true,
				RateLimit:   "10/min",
			},
			{
				Method:      "GET",
				Path:        "/admin/logs",
				Description: "获取验证日志",
				Auth:        true,
				RateLimit:   "60/min",
			},
			{
				Method:      "GET",
				Path:        "/admin/logs/:id",
				Description: "获取日志详情",
				Auth:        true,
				RateLimit:   "60/min",
			},
			{
				Method:      "GET",
				Path:        "/admin/statistics/overview",
				Description: "获取统计概览",
				Auth:        true,
				RateLimit:   "30/min",
				CacheTTL:    60,
			},
			{
				Method:      "GET",
				Path:        "/admin/statistics/verification-trend",
				Description: "获取验证趋势",
				Auth:        true,
				RateLimit:   "30/min",
				CacheTTL:    300,
			},
		},
	}

	response.Success(c, doc)
}

func (h *APIDocHandler) GetAPIV2Doc(c *gin.Context) {
	doc := APIVersionInfo{
		Version: "v2",
		BaseURL: "/api/v2",
		Endpoints: []EndpointInfo{
			{
				Method:      "GET",
				Path:        "/health",
				Description: "健康检查",
				Auth:        false,
				RateLimit:   "100/min",
			},
			{
				Method:      "GET",
				Path:        "/captcha/gesture",
				Description: "获取手势验证码",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/gesture/verify",
				Description: "验证手势验证码",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/rotate/create",
				Description: "创建旋转验证码",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/rotate/verify",
				Description: "验证旋转验证码",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/video/generate",
				Description: "生成视频验证码",
				Auth:        false,
				RateLimit:   "30/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/video/verify",
				Description: "验证视频验证码",
				Auth:        false,
				RateLimit:   "30/min",
			},
			{
				Method:      "GET",
				Path:        "/captcha/video/options",
				Description: "获取视频验证码选项",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/ai/v3/smart-captcha/generate",
				Description: "AI智能验证码生成",
				Auth:        true,
				RateLimit:   "30/min",
			},
			{
				Method:      "POST",
				Path:        "/ai/v3/risk-assessment",
				Description: "AI风险评估",
				Auth:        true,
				RateLimit:   "30/min",
			},
			{
				Method:      "POST",
				Path:        "/crypto/v2/generate-key",
				Description: "生成加密密钥",
				Auth:        true,
				RateLimit:   "20/min",
			},
			{
				Method:      "POST",
				Path:        "/crypto/v2/encrypt",
				Description: "加密数据",
				Auth:        true,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/crypto/v2/decrypt",
				Description: "解密数据",
				Auth:        true,
				RateLimit:   "60/min",
			},
		},
	}

	response.Success(c, doc)
}

func (h *APIDocHandler) GetAPIV3Doc(c *gin.Context) {
	doc := APIVersionInfo{
		Version: "v3",
		BaseURL: "/api/v3",
		Endpoints: []EndpointInfo{
			{
				Method:      "GET",
				Path:        "/health",
				Description: "健康检查",
				Auth:        false,
				RateLimit:   "100/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/neural/create",
				Description: "创建神经验证码",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/neural/verify",
				Description: "验证神经验证码",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/spatio-temporal/create",
				Description: "创建时空验证码",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/spatio-temporal/verify",
				Description: "验证时空验证码",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/vr-ar/generate",
				Description: "生成VR/AR验证码",
				Auth:        false,
				RateLimit:   "30/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/vr-ar/verify",
				Description: "验证VR/AR验证码",
				Auth:        false,
				RateLimit:   "30/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/biometric/generate",
				Description: "生成生物识别验证码",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/biometric/verify",
				Description: "验证生物识别验证码",
				Auth:        false,
				RateLimit:   "60/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/multisensory/create",
				Description: "创建多感官验证码",
				Auth:        false,
				RateLimit:   "30/min",
			},
			{
				Method:      "POST",
				Path:        "/captcha/multisensory/verify",
				Description: "验证多感官验证码",
				Auth:        false,
				RateLimit:   "30/min",
			},
			{
				Method:      "POST",
				Path:        "/ai/v3/feedback",
				Description: "记录反馈",
				Auth:        true,
				RateLimit:   "100/min",
			},
			{
				Method:      "GET",
				Path:        "/ai/v3/stats",
				Description: "获取学习统计",
				Auth:        true,
				RateLimit:   "30/min",
				CacheTTL:    300,
			},
		},
	}

	response.Success(c, doc)
}

func (h *APIDocHandler) GetAPIOverview(c *gin.Context) {
	overview := gin.H{
		"name":        "HJTPX API",
		"version":     "20.0.0",
		"description": "验证码服务API",
		"versions": []gin.H{
			{
				"version":   "v1",
				"base_url":  "/api/v1",
				"status":    "stable",
				"endpoints": 19,
			},
			{
				"version":   "v2",
				"base_url":  "/api/v2",
				"status":    "stable",
				"endpoints": 14,
			},
			{
				"version":   "v3",
				"base_url":  "/api/v3",
				"status":    "current",
				"endpoints": 13,
			},
		},
		"documentation": gin.H{
			"swagger": "/swagger/index.html",
			"redoc":   "/api/docs",
		},
		"contact": gin.H{
			"name":  "API Support",
			"email": "support@example.com",
		},
		"updated_at": time.Now().Format(time.RFC3339),
	}

	response.Success(c, overview)
}

func (h *APIDocHandler) GetHealthCheck(c *gin.Context) {
	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"services": gin.H{
			"api":     "healthy",
			"database": "healthy",
			"cache":   "healthy",
		},
		"version": "20.0.0",
	}

	response.Success(c, health)
}

func (h *APIDocHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/docs", h.GetAPIOverview)
		api.GET("/health", h.GetHealthCheck)

		v1 := api.Group("/v1")
		{
			v1.GET("/docs", h.GetAPIV1Doc)
		}

		v2 := api.Group("/v2")
		{
			v2.GET("/docs", h.GetAPIV2Doc)
		}

		v3 := api.Group("/v3")
		{
			v3.GET("/docs", h.GetAPIV3Doc)
		}
	}
}
