package service

import (
	"testing"
)

func TestNewConfigService(t *testing.T) {
	configService := NewConfigService()
	if configService == nil {
		t.Error("NewConfigService 返回了 nil")
	}
}

func TestGetConfigConfig(t *testing.T) {
	configService := NewConfigService()
	
	config, err := configService.GetConfig()
	if err != nil {
		t.Errorf("获取配置失败: %v", err)
	}
	if config == nil {
		t.Error("配置不应为 nil")
	}
}

func TestUpdateConfigConfig(t *testing.T) {
	configService := NewConfigService()
	
	currentConfig, err := configService.GetConfig()
	if err != nil {
		t.Skipf("无法获取当前配置，跳过测试: %v", err)
	}
	
	err = configService.UpdateConfig(currentConfig)
	if err != nil {
		t.Errorf("更新配置失败: %v", err)
	}
}

func TestGetConfigValue(t *testing.T) {
	configService := NewConfigService()
	
	value := configService.GetConfigValue("app.name")
	if value == nil {
		t.Error("应该能获取配置值")
	}
}

func TestGetConfigValue_NotFound(t *testing.T) {
	configService := NewConfigService()
	
	value := configService.GetConfigValue("nonexistent.key")
	if value != nil {
		t.Error("不存在的配置键应该返回 nil")
	}
}

func TestSetConfigValue(t *testing.T) {
	configService := NewConfigService()
	
	err := configService.SetConfigValue("test.key", "test-value")
	if err != nil {
		t.Errorf("设置配置值失败: %v", err)
	}
	
	value := configService.GetConfigValue("test.key")
	if value == nil || value != "test-value" {
		t.Error("配置值设置不正确")
	}
}

func TestReloadConfig(t *testing.T) {
	configService := NewConfigService()
	
	err := configService.ReloadConfig()
	if err != nil {
		t.Errorf("重新加载配置失败: %v", err)
	}
}

func TestValidateConfigConfig(t *testing.T) {
	configService := NewConfigService()
	
	validConfig := map[string]interface{}{
		"app.name": "hjtpx",
		"app.port": 8080,
	}
	
	err := configService.ValidateConfig(validConfig)
	if err != nil {
		t.Errorf("有效配置验证失败: %v", err)
	}
	
	invalidConfig := map[string]interface{}{
		"app.port": "not-a-number",
	}
	
	err = configService.ValidateConfig(invalidConfig)
	if err == nil {
		t.Error("无效配置应该验证失败")
	}
}

func TestExportConfig(t *testing.T) {
	configService := NewConfigService()
	
	export, err := configService.ExportConfig()
	if err != nil {
		t.Errorf("导出配置失败: %v", err)
	}
	if export == "" {
		t.Error("导出的配置不应为空")
	}
}

func TestImportConfig(t *testing.T) {
	configService := NewConfigService()
	
	configJSON := `{"app.name":"hjtpx","app.port":8080}`
	
	err := configService.ImportConfig(configJSON)
	if err != nil {
		t.Errorf("导入配置失败: %v", err)
	}
	
	value := configService.GetConfigValue("app.name")
	if value == nil || value != "hjtpx" {
		t.Error("导入的配置值不正确")
	}
}
