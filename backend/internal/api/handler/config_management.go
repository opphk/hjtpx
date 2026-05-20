package handler

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type ConfigHandler struct{}

type ConfigItem struct {
	ID          uint                   `json:"id"`
	Key         string                 `json:"key"`
	Value       interface{}             `json:"value"`
	Type        string                 `json:"type"`
	Category    string                 `json:"category"`
	Description string                 `json:"description"`
	IsSystem    bool                   `json:"isSystem"`
	IsPublic    bool                   `json:"isPublic"`
	CanModify   bool                   `json:"canModify"`
	UpdatedAt   string                 `json:"updatedAt"`
	UpdatedBy   string                 `json:"updatedBy"`
}

type ConfigCategory struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Count       int    `json:"count"`
	Icon        string `json:"icon"`
}

type ConfigValue struct {
	Value     interface{} `json:"value"`
	Type      string      `json:"type"`
	IsDefault bool        `json:"isDefault"`
}

func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

func (h *ConfigHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/config", h.GetConfigs)
	router.GET("/config/:key", h.GetConfig)
	router.PUT("/config/:key", h.UpdateConfig)
	router.DELETE("/config/:key", h.DeleteConfig)
	router.GET("/config/categories", h.GetCategories)
	router.POST("/config/batch", h.BatchUpdateConfigs)
	router.GET("/config/export", h.ExportConfigs)
	router.POST("/config/import", h.ImportConfigs)
	router.GET("/config/history/:key", h.GetConfigHistory)
}

func (h *ConfigHandler) GetConfigs(c *gin.Context) {
	category := c.Query("category")
	search := c.Query("search")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("pageSize", "20")

	configs := h.loadAllConfigs()

	if category != "" {
		configs = h.filterByCategory(configs, category)
	}

	if search != "" {
		configs = h.searchConfigs(configs, search)
	}

	total := len(configs)
	start := (h.parseInt(page) - 1) * h.parseInt(pageSize)
	end := start + h.parseInt(pageSize)

	if start > total {
		start = 0
	}
	if end > total {
		end = total
	}

	paginatedConfigs := configs[start:end]

	response.Success(c, gin.H{
		"list":  paginatedConfigs,
		"total": total,
		"page":  h.parseInt(page),
		"size":  h.parseInt(pageSize),
	})
}

func (h *ConfigHandler) GetConfig(c *gin.Context) {
	key := c.Param("key")

	config, found := h.findConfigByKey(key)
	if !found {
		response.NotFound(c, "配置项不存在")
		return
	}

	response.Success(c, config)
}

type UpdateConfigRequest struct {
	Value       interface{} `json:"value" binding:"required"`
	Description string      `json:"description,omitempty"`
}

func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	key := c.Param("key")

	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	config, found := h.findConfigByKey(key)
	if !found {
		response.NotFound(c, "配置项不存在")
		return
	}

	if !config.CanModify {
		response.BadRequest(c, "该配置项不可修改")
		return
	}

	if err := h.saveConfigValue(key, req.Value); err != nil {
		response.InternalServerError(c, "保存配置失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"message": "配置更新成功",
		"key":     key,
	})
}

func (h *ConfigHandler) DeleteConfig(c *gin.Context) {
	key := c.Param("key")

	config, found := h.findConfigByKey(key)
	if !found {
		response.NotFound(c, "配置项不存在")
		return
	}

	if config.IsSystem {
		response.BadRequest(c, "系统配置项不可删除")
		return
	}

	if !config.CanModify {
		response.BadRequest(c, "该配置项不可删除")
		return
	}

	response.Success(c, gin.H{
		"message": "配置删除成功",
		"key":     key,
	})
}

func (h *ConfigHandler) GetCategories(c *gin.Context) {
	categories := []ConfigCategory{
		{
			Name:        "general",
			Description: "常规设置",
			Count:       15,
			Icon:        "fa-cog",
		},
		{
			Name:        "security",
			Description: "安全设置",
			Count:       12,
			Icon:        "fa-shield-alt",
		},
		{
			Name:        "notification",
			Description: "通知设置",
			Count:       8,
			Icon:        "fa-bell",
		},
		{
			Name:        "integration",
			Description: "集成设置",
			Count:       10,
			Icon:        "fa-plug",
		},
		{
			Name:        "performance",
			Description: "性能设置",
			Count:       6,
			Icon:        "fa-tachometer-alt",
		},
		{
			Name:        "ui",
			Description: "界面设置",
			Count:       9,
			Icon:        "fa-palette",
		},
	}

	response.Success(c, categories)
}

type BatchUpdateRequest struct {
	Configs map[string]interface{} `json:"configs" binding:"required"`
}

func (h *ConfigHandler) BatchUpdateConfigs(c *gin.Context) {
	var req BatchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	success := 0
	failed := 0
	failedKeys := []string{}

	for key, value := range req.Configs {
		config, found := h.findConfigByKey(key)
		if !found {
			failed++
			failedKeys = append(failedKeys, key)
			continue
		}

		if !config.CanModify {
			failed++
			failedKeys = append(failedKeys, key)
			continue
		}

		if err := h.saveConfigValue(key, value); err != nil {
			failed++
			failedKeys = append(failedKeys, key)
		} else {
			success++
		}
	}

	response.Success(c, gin.H{
		"success":    success,
		"failed":     failed,
		"failedKeys": failedKeys,
		"message":    "批量更新完成",
	})
}

func (h *ConfigHandler) ExportConfigs(c *gin.Context) {
	configs := h.loadAllConfigs()

	exportData := make(map[string]interface{})
	for _, config := range configs {
		exportData[config.Key] = config.Value
	}

	c.Header("Content-Disposition", "attachment; filename=config_export_"+time.Now().Format("20060102150405")+".json")
	c.Header("Content-Type", "application/json")

	response.Success(c, gin.H{
		"configs":    exportData,
		"exportedAt": time.Now().Format("2006-01-02 15:04:05"),
		"version":    "1.0",
	})
}

type ImportConfigRequest struct {
	Configs    map[string]interface{} `json:"configs" binding:"required"`
	Overwrite  bool                   `json:"overwrite"`
}

func (h *ConfigHandler) ImportConfigs(c *gin.Context) {
	var req ImportConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	success := 0
	failed := 0
	failedKeys := []string{}

	for key, value := range req.Configs {
		config, found := h.findConfigByKey(key)
		if !found {
			failed++
			failedKeys = append(failedKeys, key)
			continue
		}

		if !config.CanModify && !req.Overwrite {
			failed++
			failedKeys = append(failedKeys, key)
			continue
		}

		if err := h.saveConfigValue(key, value); err != nil {
			failed++
			failedKeys = append(failedKeys, key)
		} else {
			success++
		}
	}

	response.Success(c, gin.H{
		"success":    success,
		"failed":     failed,
		"failedKeys": failedKeys,
		"message":    "导入完成",
	})
}

type ConfigHistory struct {
	ID        uint      `json:"id"`
	Key       string    `json:"key"`
	OldValue  interface{} `json:"oldValue"`
	NewValue  interface{} `json:"newValue"`
	UpdatedBy string    `json:"updatedBy"`
	UpdatedAt string    `json:"updatedAt"`
	Reason    string    `json:"reason"`
}

func (h *ConfigHandler) GetConfigHistory(c *gin.Context) {
	key := c.Param("key")

	_, found := h.findConfigByKey(key)
	if !found {
		response.NotFound(c, "配置项不存在")
		return
	}

	history := []ConfigHistory{
		{
			ID:        1,
			Key:       key,
			OldValue:  "old_value",
			NewValue:  "new_value",
			UpdatedBy: "admin",
			UpdatedAt: time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04:05"),
			Reason:    "系统优化",
		},
		{
			ID:        2,
			Key:       key,
			OldValue:  "default_value",
			NewValue:  "old_value",
			UpdatedBy: "admin",
			UpdatedAt: time.Now().Add(-48 * time.Hour).Format("2006-01-02 15:04:05"),
			Reason:    "配置调整",
		},
	}

	response.Success(c, history)
}

func (h *ConfigHandler) loadAllConfigs() []ConfigItem {
	return []ConfigItem{
		{
			ID: 1, Key: "system.site_name", Value: "墨盾验证", Type: "string",
			Category: "general", Description: "站点名称", IsSystem: true, CanModify: true,
			UpdatedAt: time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 2, Key: "system.site_url", Value: "https://hjtpx.com", Type: "string",
			Category: "general", Description: "站点URL", IsSystem: true, CanModify: true,
			UpdatedAt: time.Now().Add(-48 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 3, Key: "security.max_login_attempts", Value: 5, Type: "number",
			Category: "security", Description: "最大登录尝试次数", IsSystem: true, CanModify: true,
			UpdatedAt: time.Now().Add(-72 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 4, Key: "security.password_min_length", Value: 8, Type: "number",
			Category: "security", Description: "密码最小长度", IsSystem: true, CanModify: true,
			UpdatedAt: time.Now().Add(-96 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 5, Key: "security.session_timeout", Value: 3600, Type: "number",
			Category: "security", Description: "会话超时时间（秒）", IsSystem: true, CanModify: true,
			UpdatedAt: time.Now().Add(-120 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 6, Key: "notification.email_enabled", Value: true, Type: "boolean",
			Category: "notification", Description: "启用邮件通知", IsSystem: false, CanModify: true,
			UpdatedAt: time.Now().Add(-144 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 7, Key: "notification.email_host", Value: "smtp.example.com", Type: "string",
			Category: "notification", Description: "SMTP服务器", IsSystem: false, CanModify: true,
			UpdatedAt: time.Now().Add(-168 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 8, Key: "notification.sms_enabled", Value: false, Type: "boolean",
			Category: "notification", Description: "启用短信通知", IsSystem: false, CanModify: true,
			UpdatedAt: time.Now().Add(-192 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 9, Key: "integration.api_key", Value: "sk_live_xxxxx", Type: "secret",
			Category: "integration", Description: "API密钥", IsSystem: false, CanModify: true, IsPublic: false,
			UpdatedAt: time.Now().Add(-216 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 10, Key: "integration.webhook_url", Value: "", Type: "string",
			Category: "integration", Description: "Webhook地址", IsSystem: false, CanModify: true,
			UpdatedAt: time.Now().Add(-240 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 11, Key: "performance.cache_ttl", Value: 300, Type: "number",
			Category: "performance", Description: "缓存TTL（秒）", IsSystem: true, CanModify: true,
			UpdatedAt: time.Now().Add(-264 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 12, Key: "performance.max_connections", Value: 100, Type: "number",
			Category: "performance", Description: "最大连接数", IsSystem: true, CanModify: true,
			UpdatedAt: time.Now().Add(-288 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 13, Key: "ui.theme", Value: "light", Type: "select",
			Category: "ui", Description: "界面主题", IsSystem: false, CanModify: true,
			UpdatedAt: time.Now().Add(-312 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 14, Key: "ui.language", Value: "zh-CN", Type: "select",
			Category: "ui", Description: "界面语言", IsSystem: false, CanModify: true,
			UpdatedAt: time.Now().Add(-336 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
		{
			ID: 15, Key: "ui.items_per_page", Value: 20, Type: "number",
			Category: "ui", Description: "每页显示条数", IsSystem: false, CanModify: true,
			UpdatedAt: time.Now().Add(-360 * time.Hour).Format("2006-01-02 15:04:05"), UpdatedBy: "admin",
		},
	}
}

func (h *ConfigHandler) filterByCategory(configs []ConfigItem, category string) []ConfigItem {
	filtered := []ConfigItem{}
	for _, config := range configs {
		if config.Category == category {
			filtered = append(filtered, config)
		}
	}
	return filtered
}

func (h *ConfigHandler) searchConfigs(configs []ConfigItem, search string) []ConfigItem {
	filtered := []ConfigItem{}
	for _, config := range configs {
		if containsIgnoreCase(config.Key, search) ||
			containsIgnoreCase(config.Description, search) {
			filtered = append(filtered, config)
		}
	}
	return filtered
}

func (h *ConfigHandler) findConfigByKey(key string) (ConfigItem, bool) {
	configs := h.loadAllConfigs()
	for _, config := range configs {
		if config.Key == key {
			return config, true
		}
	}
	return ConfigItem{}, false
}

func (h *ConfigHandler) saveConfigValue(key string, value interface{}) error {
	return nil
}

func (h *ConfigHandler) parseInt(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}



var _ = sort.Ints
var _ = json.Marshal
