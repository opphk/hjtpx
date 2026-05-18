package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type UpdateWhitelabelConfigRequest struct {
	BrandName       string `json:"brand_name"`
	PrimaryColor    string `json:"primary_color"`
	SuccessColor    string `json:"success_color"`
	WarningColor    string `json:"warning_color"`
	DangerColor     string `json:"danger_color"`
	CustomCSS       string `json:"custom_css"`
	IsEnabled       bool   `json:"is_enabled"`
}

func GetWhitelabelConfig(c *gin.Context) {
	whitelabelService := service.NewWhitelabelService()
	config, err := whitelabelService.GetConfig()
	if err != nil {
		response.InternalServerError(c, "failed to get whitelabel config: "+err.Error())
		return
	}
	response.Success(c, config)
}

func UpdateWhitelabelConfig(c *gin.Context) {
	var req UpdateWhitelabelConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}
	
	whitelabelService := service.NewWhitelabelService()
	
	// Get current config to preserve logo and favicon
	currentConfig, _ := whitelabelService.GetConfig()
	
	config := service.WhitelabelConfig{
		BrandName:    req.BrandName,
		PrimaryColor: req.PrimaryColor,
		SuccessColor: req.SuccessColor,
		WarningColor: req.WarningColor,
		DangerColor:  req.DangerColor,
		LogoURL:      currentConfig.LogoURL,
		FaviconURL:   currentConfig.FaviconURL,
		CustomCSS:    req.CustomCSS,
		IsEnabled:    req.IsEnabled,
	}
	
	if err := whitelabelService.UpdateConfig(config); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	
	response.Success(c, gin.H{
		"message": "whitelabel config updated successfully",
		"config":  config,
	})
}

func GetWhitelabelCSS(c *gin.Context) {
	whitelabelService := service.NewWhitelabelService()
	css := whitelabelService.GenerateCSS()
	
	c.Header("Content-Type", "text/css; charset=utf-8")
	c.String(200, css)
}

func UploadLogo(c *gin.Context) {
	logoType := c.Param("type") // "logo" or "favicon"
	if logoType != "logo" && logoType != "favicon" {
		response.BadRequest(c, "invalid logo type, must be 'logo' or 'favicon'")
		return
	}
	
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "no file uploaded: "+err.Error())
		return
	}
	
	// Open the file
	src, err := file.Open()
	if err != nil {
		response.InternalServerError(c, "failed to open uploaded file: "+err.Error())
		return
	}
	defer src.Close()
	
	whitelabelService := service.NewWhitelabelService()
	
	// Upload the logo
	url, err := whitelabelService.UploadLogo(src, file.Filename, logoType)
	if err != nil {
		response.InternalServerError(c, "failed to upload logo: "+err.Error())
		return
	}
	
	// Update the config
	config, _ := whitelabelService.GetConfig()
	if logoType == "logo" {
		config.LogoURL = url
	} else {
		config.FaviconURL = url
	}
	
	if err := whitelabelService.UpdateConfig(config); err != nil {
		response.InternalServerError(c, "failed to update config: "+err.Error())
		return
	}
	
	response.Success(c, gin.H{
		"message": "logo uploaded successfully",
		"url":     url,
		"type":    logoType,
	})
}

func ResetWhitelabelConfig(c *gin.Context) {
	whitelabelService := service.NewWhitelabelService()
	
	if err := whitelabelService.ResetConfig(); err != nil {
		response.InternalServerError(c, "failed to reset whitelabel config: "+err.Error())
		return
	}
	
	defaultConfig := whitelabelService.GetDefaultConfig()
	response.Success(c, gin.H{
		"message": "whitelabel config reset to defaults",
		"config":  defaultConfig,
	})
}

func DeleteLogo(c *gin.Context) {
	logoType := c.Param("type") // "logo" or "favicon"
	if logoType != "logo" && logoType != "favicon" {
		response.BadRequest(c, "invalid logo type, must be 'logo' or 'favicon'")
		return
	}
	
	whitelabelService := service.NewWhitelabelService()
	config, _ := whitelabelService.GetConfig()
	
	var url string
	if logoType == "logo" {
		url = config.LogoURL
		config.LogoURL = ""
	} else {
		url = config.FaviconURL
		config.FaviconURL = ""
	}
	
	// Delete file
	if url != "" {
		whitelabelService.DeleteLogo(url)
	}
	
	// Update config
	if err := whitelabelService.UpdateConfig(config); err != nil {
		response.InternalServerError(c, "failed to update config: "+err.Error())
		return
	}
	
	response.Success(c, gin.H{
		"message": "logo deleted successfully",
		"type":    logoType,
	})
}
