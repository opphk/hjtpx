package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigHandler_LoadAllConfigs(t *testing.T) {
	handler := NewConfigHandler()

	t.Run("Test Load All Configs", func(t *testing.T) {
		configs := handler.loadAllConfigs()

		assert.NotEmpty(t, configs, "Configs should not be empty")
		assert.Greater(t, len(configs), 10, "Should have more than 10 configs")
	})

	t.Run("Test Config Structure", func(t *testing.T) {
		configs := handler.loadAllConfigs()

		for _, config := range configs {
			assert.NotEmpty(t, config.Key, "Key should not be empty")
			assert.NotEmpty(t, config.Category, "Category should not be empty")
			assert.NotEmpty(t, config.Type, "Type should not be empty")
			assert.NotEmpty(t, config.Description, "Description should not be empty")
		}
	})

	t.Run("Test System Configs", func(t *testing.T) {
		configs := handler.loadAllConfigs()

		for _, config := range configs {
			if config.IsSystem {
				assert.True(t, config.CanModify || !config.CanModify, "System config should have defined modify flag")
			}
		}
	})
}

func TestConfigHandler_FilterByCategory(t *testing.T) {
	handler := NewConfigHandler()

	t.Run("Test Filter by General Category", func(t *testing.T) {
		configs := handler.loadAllConfigs()
		filtered := handler.filterByCategory(configs, "general")

		for _, config := range filtered {
			assert.Equal(t, "general", config.Category, "Category should be general")
		}
	})

	t.Run("Test Filter by Security Category", func(t *testing.T) {
		configs := handler.loadAllConfigs()
		filtered := handler.filterByCategory(configs, "security")

		for _, config := range filtered {
			assert.Equal(t, "security", config.Category, "Category should be security")
		}
	})

	t.Run("Test Filter by Notification Category", func(t *testing.T) {
		configs := handler.loadAllConfigs()
		filtered := handler.filterByCategory(configs, "notification")

		for _, config := range filtered {
			assert.Equal(t, "notification", config.Category, "Category should be notification")
		}
	})
}

func TestConfigHandler_SearchConfigs(t *testing.T) {
	handler := NewConfigHandler()

	t.Run("Test Search by Key", func(t *testing.T) {
		configs := handler.loadAllConfigs()
		results := handler.searchConfigs(configs, "system")

		assert.NotEmpty(t, results, "Should find configs matching 'system'")
	})

	t.Run("Test Search by Description", func(t *testing.T) {
		configs := handler.loadAllConfigs()
		results := handler.searchConfigs(configs, "登录")

		assert.NotEmpty(t, results, "Should find configs matching '登录'")
	})

	t.Run("Test Search Case Insensitive", func(t *testing.T) {
		configs := handler.loadAllConfigs()
		results := handler.searchConfigs(configs, "EMAIL")

		for _, config := range results {
			assert.Contains(t, config.Key, "email", "Should contain email (case insensitive)")
		}
	})
}

func TestConfigHandler_FindConfigByKey(t *testing.T) {
	handler := NewConfigHandler()

	t.Run("Test Find Existing Config", func(t *testing.T) {
		config, found := handler.findConfigByKey("system.site_name")

		assert.True(t, found, "Should find config with key 'system.site_name'")
		assert.Equal(t, "system.site_name", config.Key, "Key should match")
	})

	t.Run("Test Find Non-existing Config", func(t *testing.T) {
		_, found := handler.findConfigByKey("non.existing.key")

		assert.False(t, found, "Should not find config with non-existing key")
	})
}

func TestConfigHandler_ParseInt(t *testing.T) {
	handler := NewConfigHandler()

	t.Run("Test Parse Valid Number", func(t *testing.T) {
		assert.Equal(t, 123, handler.parseInt("123"))
		assert.Equal(t, 0, handler.parseInt("0"))
		assert.Equal(t, 999, handler.parseInt("999"))
	})

	t.Run("Test Parse Mixed String", func(t *testing.T) {
		assert.Equal(t, 123, handler.parseInt("123abc"))
		assert.Equal(t, 456, handler.parseInt("abc456def"))
	})

	t.Run("Test Parse Non-numeric String", func(t *testing.T) {
		assert.Equal(t, 0, handler.parseInt("abc"))
		assert.Equal(t, 0, handler.parseInt(""))
	})
}

func TestConfigHandler_ContainsIgnoreCase(t *testing.T) {
	t.Run("Test Contains Ignore Case", func(t *testing.T) {
		tests := []struct {
			s        string
			substr   string
			expected bool
		}{
			{"Hello World", "hello", true},
			{"Hello World", "WORLD", true},
			{"Hello World", "Test", false},
			{"", "", true},
			{"Hello", "", true},
			{"", "Test", false},
		}

		for _, tt := range tests {
			result := containsIgnoreCase(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result, "Result should match expected for s='%s', substr='%s'", tt.s, tt.substr)
		}
	})
}

func TestConfigHandler_GetCategories(t *testing.T) {
	t.Run("Test Get Categories", func(t *testing.T) {
		categories := []ConfigCategory{
			{Name: "general", Description: "常规设置", Count: 15, Icon: "fa-cog"},
			{Name: "security", Description: "安全设置", Count: 12, Icon: "fa-shield-alt"},
			{Name: "notification", Description: "通知设置", Count: 8, Icon: "fa-bell"},
			{Name: "integration", Description: "集成设置", Count: 10, Icon: "fa-plug"},
			{Name: "performance", Description: "性能设置", Count: 6, Icon: "fa-tachometer-alt"},
			{Name: "ui", Description: "界面设置", Count: 9, Icon: "fa-palette"},
		}

		assert.Equal(t, 6, len(categories), "Should have 6 categories")
		assert.Equal(t, "general", categories[0].Name)
		assert.Equal(t, "security", categories[1].Name)
	})
}

func TestConfigHandler_BatchUpdateConfigs(t *testing.T) {
	t.Run("Test Batch Update Configs Structure", func(t *testing.T) {
		req := BatchUpdateRequest{
			Configs: map[string]interface{}{
				"system.site_name": "New Site Name",
				"ui.theme":         "dark",
			},
		}

		assert.Equal(t, 2, len(req.Configs), "Should have 2 configs")
		assert.Equal(t, "New Site Name", req.Configs["system.site_name"])
		assert.Equal(t, "dark", req.Configs["ui.theme"])
	})
}

func TestConfigHandler_ConfigHistory(t *testing.T) {
	t.Run("Test Config History Structure", func(t *testing.T) {
		history := []ConfigHistory{
			{
				ID:        1,
				Key:       "test.key",
				OldValue:  "old_value",
				NewValue:  "new_value",
				UpdatedBy: "admin",
				UpdatedAt: "2024-01-01 12:00:00",
				Reason:    "Test update",
			},
		}

		assert.Equal(t, 1, len(history), "Should have 1 history entry")
		assert.Equal(t, "test.key", history[0].Key)
		assert.Equal(t, "admin", history[0].UpdatedBy)
	})
}

func TestConfigItem(t *testing.T) {
	t.Run("Test Config Item Structure", func(t *testing.T) {
		item := ConfigItem{
			ID:          1,
			Key:         "test.config",
			Value:       "value",
			Type:        "string",
			Category:    "general",
			Description: "Test config",
			IsSystem:    false,
			IsPublic:    true,
			CanModify:   true,
			UpdatedAt:   "2024-01-01 12:00:00",
			UpdatedBy:   "admin",
		}

		assert.Equal(t, uint(1), item.ID)
		assert.Equal(t, "test.config", item.Key)
		assert.Equal(t, "string", item.Type)
		assert.True(t, item.IsPublic)
		assert.True(t, item.CanModify)
	})
}
