package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "image/jpeg"
	_ "image/png"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"gorm.io/gorm"
)

const (
	whitelabelCacheKey = "whitelabel:config"
	uploadsDir         = "./uploads"
	logoDir            = "logos"
	defaultPrimary     = "#007bff"
	defaultSuccess     = "#28a745"
	defaultWarning     = "#ffc107"
	defaultDanger      = "#dc3545"
	defaultBrandName   = "HJTPX"
)

// WhitelabelConfig 白标配置结构
type WhitelabelConfig struct {
	BrandName    string `json:"brand_name"`
	PrimaryColor string `json:"primary_color"`
	SuccessColor string `json:"success_color"`
	WarningColor string `json:"warning_color"`
	DangerColor  string `json:"danger_color"`
	LogoURL      string `json:"logo_url"`
	FaviconURL   string `json:"favicon_url"`
	CustomCSS    string `json:"custom_css"`
	IsEnabled    bool   `json:"is_enabled"`
}

// WhitelabelService 白标主题服务
type WhitelabelService struct {
	ctx context.Context
}

// NewWhitelabelService 创建白标主题服务
func NewWhitelabelService() *WhitelabelService {
	if err := os.MkdirAll(filepath.Join(uploadsDir, logoDir), 0755); err != nil {
		fmt.Printf("Warning: failed to create uploads directory: %v\n", err)
	}

	return &WhitelabelService{
		ctx: context.Background(),
	}
}

// GetDefaultConfig 获取默认配置
func (s *WhitelabelService) GetDefaultConfig() WhitelabelConfig {
	return WhitelabelConfig{
		BrandName:    defaultBrandName,
		PrimaryColor: defaultPrimary,
		SuccessColor: defaultSuccess,
		WarningColor: defaultWarning,
		DangerColor:  defaultDanger,
		LogoURL:      "",
		FaviconURL:   "",
		CustomCSS:    "",
		IsEnabled:    false,
	}
}

// GetConfig 获取白标配置
func (s *WhitelabelService) GetConfig() (WhitelabelConfig, error) {
	if redis.Client != nil {
		cached, err := redis.Client.Get(s.ctx, whitelabelCacheKey).Result()
		if err == nil && cached != "" {
			var cachedConfig WhitelabelConfig
			if err := json.Unmarshal([]byte(cached), &cachedConfig); err == nil {
				return cachedConfig, nil
			}
		}
	}

	config := s.GetDefaultConfig()
	if database.DB == nil {
		return config, nil // 没有数据库连接时返回默认配置
	}

	var configs []models.Config
	if err := database.DB.Order("sort_order ASC, key ASC").Find(&configs).Error; err != nil {
		return config, err
	}

	for _, cfg := range configs {
		switch cfg.Key {
		case "whitelabel.brand_name":
			config.BrandName = cfg.Value
		case "whitelabel.primary_color":
			config.PrimaryColor = cfg.Value
		case "whitelabel.success_color":
			config.SuccessColor = cfg.Value
		case "whitelabel.warning_color":
			config.WarningColor = cfg.Value
		case "whitelabel.danger_color":
			config.DangerColor = cfg.Value
		case "whitelabel.logo_url":
			config.LogoURL = cfg.Value
		case "whitelabel.favicon_url":
			config.FaviconURL = cfg.Value
		case "whitelabel.custom_css":
			config.CustomCSS = cfg.Value
		case "whitelabel.is_enabled":
			config.IsEnabled = cfg.Value == "true"
		}
	}

	if redis.Client != nil {
		jsonData, _ := json.Marshal(config)
		redis.Client.Set(s.ctx, whitelabelCacheKey, jsonData, 0)
	}

	return config, nil
}

// UpdateConfig 更新白标配置
func (s *WhitelabelService) UpdateConfig(config WhitelabelConfig) error {
	if err := s.ValidateConfig(config); err != nil {
		return err
	}

	if database.DB == nil {
		return errors.New("database not initialized")
	}

	updates := map[string]string{
		"whitelabel.brand_name":    config.BrandName,
		"whitelabel.primary_color": config.PrimaryColor,
		"whitelabel.success_color": config.SuccessColor,
		"whitelabel.warning_color": config.WarningColor,
		"whitelabel.danger_color":  config.DangerColor,
		"whitelabel.logo_url":      config.LogoURL,
		"whitelabel.favicon_url":   config.FaviconURL,
		"whitelabel.custom_css":    config.CustomCSS,
		"whitelabel.is_enabled":    boolToString(config.IsEnabled),
	}

	// Batch update
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for key, value := range updates {
			var existing models.Config
			result := tx.Where("`key` = ?", key).First(&existing)

			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// Create new config
				group := "whitelabel"
				if len(key) > 0 {
					parts := strings.Split(key, ".")
					if len(parts) > 1 {
						group = parts[0]
					}
				}
				newConfig := models.Config{
					Key:       key,
					Value:     value,
					Group:     group,
					IsVisible: true,
				}
				if err := tx.Create(&newConfig).Error; err != nil {
					return err
				}
			} else if result.Error != nil {
				return result.Error
			} else {
				// Update existing
				existing.Value = value
				existing.UpdatedAt = time.Now()
				if err := tx.Save(&existing).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	if redis.Client != nil {
		redis.Client.Del(s.ctx, whitelabelCacheKey)
		jsonData, _ := json.Marshal(config)
		redis.Client.Set(s.ctx, whitelabelCacheKey, jsonData, 0)
	}

	return nil
}

// ValidateConfig 验证配置
func (s *WhitelabelService) ValidateConfig(config WhitelabelConfig) error {
	if config.BrandName == "" {
		return errors.New("品牌名称不能为空")
	}

	colorPattern := regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)
	colors := []string{config.PrimaryColor, config.SuccessColor, config.WarningColor, config.DangerColor}
	colorNames := []string{"主色", "成功色", "警告色", "危险色"}

	for i, color := range colors {
		if !colorPattern.MatchString(color) {
			return fmt.Errorf("%s格式不正确，应为 #FFFFFF 或 #FFF", colorNames[i])
		}
	}

	return nil
}

// GenerateCSS 生成动态CSS
func (s *WhitelabelService) GenerateCSS() string {
	config, _ := s.GetConfig()

	if !config.IsEnabled {
		return "/* Whitelabel theme is disabled */"
	}

	css := fmt.Sprintf(`
:root {
    --primary: %s;
    --success: %s;
    --warning: %s;
    --danger: %s;
}

.btn-primary {
    background-color: %s !important;
    border-color: %s !important;
}

.btn-primary:hover {
    background-color: %s !important;
    border-color: %s !important;
}

.btn-success {
    background-color: %s !important;
    border-color: %s !important;
}

.btn-success:hover {
    background-color: %s !important;
    border-color: %s !important;
}

.btn-warning {
    background-color: %s !important;
    border-color: %s !important;
}

.btn-danger {
    background-color: %s !important;
    border-color: %s !important;
}

.text-primary {
    color: %s !important;
}

.text-success {
    color: %s !important;
}

.text-warning {
    color: %s !important;
}

.text-danger {
    color: %s !important;
}

.bg-primary {
    background-color: %s !important;
}

.bg-success {
    background-color: %s !important;
}

.bg-warning {
    background-color: %s !important;
}

.bg-danger {
    background-color: %s !important;
}

.sidebar-dark-primary {
    background-color: %s !important;
}

.navbar-light {
    border-bottom-color: %s !important;
}

.nav-pills .nav-link.active {
    background-color: %s !important;
}
`,
		config.PrimaryColor, config.SuccessColor, config.WarningColor, config.DangerColor,
		config.PrimaryColor, config.PrimaryColor,
		darkenColor(config.PrimaryColor, 10), darkenColor(config.PrimaryColor, 10),
		config.SuccessColor, config.SuccessColor,
		darkenColor(config.SuccessColor, 10), darkenColor(config.SuccessColor, 10),
		config.WarningColor, config.WarningColor,
		config.DangerColor, config.DangerColor,
		config.PrimaryColor, config.SuccessColor, config.WarningColor, config.DangerColor,
		config.PrimaryColor, config.SuccessColor, config.WarningColor, config.DangerColor,
		config.PrimaryColor, config.PrimaryColor, config.PrimaryColor,
	)

	if config.CustomCSS != "" {
		css += "\n\n/* Custom CSS */\n" + config.CustomCSS
	}

	return css
}

// UploadLogo 上传Logo
func (s *WhitelabelService) UploadLogo(file io.Reader, filename string, logoType string) (string, error) {
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".png"
	}

	allowedExts := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".svg":  true,
	}

	ext = strings.ToLower(ext)
	if !allowedExts[ext] {
		return "", errors.New("不支持的图片格式，仅支持 PNG, JPG, GIF, SVG")
	}

	newFilename := fmt.Sprintf("%s_%d%s", logoType, getTimestamp(), ext)
	filepath := filepath.Join(uploadsDir, logoDir, newFilename)

	dst, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("保存文件失败: %v", err)
	}

	return fmt.Sprintf("/uploads/%s/%s", logoDir, newFilename), nil
}

// DeleteLogo 删除Logo
func (s *WhitelabelService) DeleteLogo(logoURL string) error {
	if logoURL == "" {
		return nil
	}

	filename := filepath.Base(logoURL)
	filepath := filepath.Join(uploadsDir, logoDir, filename)

	if _, err := os.Stat(filepath); err == nil {
		return os.Remove(filepath)
	}

	return nil
}

// ResetConfig 重置为默认配置
func (s *WhitelabelService) ResetConfig() error {
	defaultConfig := s.GetDefaultConfig()

	if database.DB != nil {
		keysToDelete := []string{
			"whitelabel.brand_name",
			"whitelabel.primary_color",
			"whitelabel.success_color",
			"whitelabel.warning_color",
			"whitelabel.danger_color",
			"whitelabel.logo_url",
			"whitelabel.favicon_url",
			"whitelabel.custom_css",
			"whitelabel.is_enabled",
		}

		for _, key := range keysToDelete {
			database.DB.Where("`key` = ?", key).Delete(&models.Config{})
		}
	}

	if redis.Client != nil {
		redis.Client.Del(s.ctx, whitelabelCacheKey)
	}

	// 保存默认配置
	return s.UpdateConfig(defaultConfig)
}

// InitializeDefaults 初始化默认配置
func (s *WhitelabelService) InitializeDefaults() error {
	existingConfig, err := s.GetConfig()
	if err != nil {
		return err
	}

	defaultConfig := s.GetDefaultConfig()

	if existingConfig.BrandName == defaultBrandName {
		return s.UpdateConfig(defaultConfig)
	}

	return nil
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func getTimestamp() int64 {
	return time.Now().UnixNano()
}

func darkenColor(hex string, percent int) string {
	if len(hex) == 4 {
		hex = "#" + string(hex[1]) + string(hex[1]) + string(hex[2]) + string(hex[2]) + string(hex[3]) + string(hex[3])
	}

	r := parseIntFromHex(hex[1:3])
	g := parseIntFromHex(hex[3:5])
	b := parseIntFromHex(hex[5:7])

	factor := float64(100-percent) / 100

	r = int(float64(r) * factor)
	g = int(float64(g) * factor)
	b = int(float64(b) * factor)

	if r < 0 {
		r = 0
	}
	if g < 0 {
		g = 0
	}
	if b < 0 {
		b = 0
	}
	if r > 255 {
		r = 255
	}
	if g > 255 {
		g = 255
	}
	if b > 255 {
		b = 255
	}

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func parseIntFromHex(s string) int {
	var result int
	fmt.Sscanf(s, "%x", &result)
	return result
}

func isValidImage(file io.Reader, ext string) bool {
	_, format, err := image.DecodeConfig(file)
	if err != nil {
		return false
	}

	switch ext {
	case ".png":
		return format == "png"
	case ".jpg", ".jpeg":
		return format == "jpeg"
	case ".gif":
		return format == "gif"
	default:
		return true
	}
}
