package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	captchago "github.com/hjtpx/hjtpx/sdk/go"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Success   bool   `json:"success"`
	Token     string `json:"token,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message"`
}

func main() {
	fmt.Println("======================================")
	fmt.Println("  Gin Framework Integration Demo")
	fmt.Println("======================================")
	fmt.Println()

	r := gin.Default()

	cfg := &captchago.Config{
		BaseURL:     "http://localhost:8080",
		MaxRetries:  3,
		HTTPTimeout: 10 * time.Second,
		DebugMode:   true,
	}

	captchaClient := captchago.NewCaptchaClient("app-id", "app-secret", cfg)
	defer captchaClient.Close()

	r.Use(func(c *gin.Context) {
		c.Set("captcha_client", captchaClient)
		c.Next()
	})

	r.GET("/captcha/slider", handleGetSliderCaptcha)
	r.GET("/captcha/click", handleGetClickCaptcha)
	r.GET("/captcha/image", handleGetImageCaptcha)
	r.POST("/captcha/verify/slider", handleVerifySlider)
	r.POST("/captcha/verify/click", handleVerifyClick)
	r.POST("/captcha/verify/image", handleVerifyImage)
	r.POST("/login", handleLogin)

	r.Run(":8081")
}

func handleGetSliderCaptcha(c *gin.Context) {
	client := c.MustGet("captcha_client").(*captchago.CaptchaClient)

	slider, err := client.GenerateSliderCaptcha()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Failed to generate slider captcha: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"session_id":   slider.ChallengeID,
		"background":   slider.BackgroundImage,
		"slider":       slider.SliderImage,
		"secret_x":     slider.SecretX,
		"secret_y":     slider.SecretY,
	})
}

func handleGetClickCaptcha(c *gin.Context) {
	client := c.MustGet("captcha_client").(*captchago.CaptchaClient)

	click, err := client.GenerateClickCaptcha()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Failed to generate click captcha: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"session_id":   click.ChallengeID,
		"image_url":    click.ImageURL,
		"target_index": click.TargetIndex,
		"icon_count":   click.TotalIcons,
	})
}

func handleGetImageCaptcha(c *gin.Context) {
	client := c.MustGet("captcha_client").(*captchago.CaptchaClient)

	image, err := client.GenerateImageCaptcha(nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Failed to generate image captcha: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"session_id":   image.ChallengeID,
		"image":        image.Image,
	})
}

func handleVerifySlider(c *gin.Context) {
	client := c.MustGet("captcha_client").(*captchago.CaptchaClient)

	var req struct {
		SessionID string `json:"session_id" binding:"required"`
		Position  string `json:"position" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request",
		})
		return
	}

	result, err := client.VerifySliderCaptcha(req.SessionID, req.Position)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Verification failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      result.Success,
		"score":        result.Score,
		"risk_level":   result.RiskLevel,
		"message":      result.Message,
	})
}

func handleVerifyClick(c *gin.Context) {
	client := c.MustGet("captcha_client").(*captchago.CaptchaClient)

	var req struct {
		SessionID string               `json:"session_id" binding:"required"`
		Clicks    []captchago.ClickData `json:"clicks" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request",
		})
		return
	}

	result, err := client.VerifyClickCaptcha(req.SessionID, req.Clicks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Verification failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      result.Success,
		"score":        result.Score,
		"risk_level":   result.RiskLevel,
		"message":      result.Message,
	})
}

func handleVerifyImage(c *gin.Context) {
	client := c.MustGet("captcha_client").(*captchago.CaptchaClient)

	var req struct {
		SessionID string `json:"session_id" binding:"required"`
		Answer    string `json:"answer" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request",
		})
		return
	}

	result, err := client.VerifyImageCaptcha(req.SessionID, req.Answer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Verification failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": result.Success,
		"message": result.Message,
	})
}

func handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request",
		})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Success: true,
		Token:   "mock-token-" + req.Username,
		Message: "Login successful",
	})
}

func exampleWithContext() {
	fmt.Println("\nContext Timeout Example:")
	fmt.Println("-----------------------------------")

	cfg := &captchago.Config{
		BaseURL:     "http://localhost:8080",
		HTTPTimeout: 5 * time.Second,
	}

	client := captchago.NewCaptchaClient("app-id", "app-secret", cfg)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	slider, err := client.GenerateSliderCaptchaWithContext(ctx)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Request timed out")
		} else {
			log.Printf("Error: %v", err)
		}
		return
	}

	log.Printf("Got slider: %s", slider.ChallengeID)
}
