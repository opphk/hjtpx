package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/captchax/sdk/go/captchax"
	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	CaptchaToken string `json:"captchaToken" binding:"required"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
}

type VerifyResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Score   float64 `json:"score,omitempty"`
}

func main() {
	r := gin.Default()

	client := captchax.NewClient(
		captchax.WithAppID("example-app"),
		captchax.WithServerURL("http://localhost:8080"),
		captchax.WithTimeout(10*time.Second),
	)

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Captcha-Token")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "captcha-example-backend",
		})
	})

	r.POST("/api/captcha/create", func(c *gin.Context) {
		var req struct {
			Type string `json:"type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
			return
		}

		ctx := context.Background()

		switch req.Type {
		case "slider":
			result, err := client.CreateSliderCaptcha(ctx, &captchax.SliderCaptchaRequest{
				Width:  200,
				Height: 80,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"id":         result.ID,
				"background": result.BackgroundB64,
				"slider":     result.SliderB64,
				"targetX":    result.TargetX,
				"targetY":    result.TargetY,
			})

		case "click":
			result, err := client.CreateClickCaptcha(ctx, &captchax.ClickCaptchaRequest{
				CharCount: 4,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"id":          result.ID,
				"image":       result.Image,
				"targetChars": result.TargetChars,
			})

		case "puzzle":
			result, err := client.CreatePuzzleCaptcha(ctx, &captchax.PuzzleCaptchaRequest{
				Width:  300,
				Height: 150,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"id":         result.ID,
				"background": result.BackgroundB64,
				"puzzle":     result.PuzzleB64,
				"targetX":    result.TargetX,
				"targetY":    result.TargetY,
			})

		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的验证码类型"})
		}
	})

	r.POST("/api/captcha/verify", func(c *gin.Context) {
		var req struct {
			Type     string  `json:"type" binding:"required"`
			CaptchaID string `json:"captchaId" binding:"required"`
			TargetX  int     `json:"targetX"`
			TargetY  int     `json:"targetY"`
			Clicks   []struct {
				Char string `json:"char"`
				X    int    `json:"x"`
				Y    int    `json:"y"`
			} `json:"clicks"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
			return
		}

		ctx := context.Background()

		var result *captchax.VerifyResponse
		var err error

		switch req.Type {
		case "slider":
			result, err = client.VerifySlider(ctx, &captchax.VerifyRequest{
				CaptchaID: req.CaptchaID,
				TargetX:   req.TargetX,
				TargetY:   req.TargetY,
			})

		case "click":
			var clicks []captchax.ClickItem
			for _, click := range req.Clicks {
				clicks = append(clicks, captchax.ClickItem{
					Char: click.Char,
					X:    click.X,
					Y:    click.Y,
				})
			}
			result, err = client.VerifyClick(ctx, &captchax.ClickVerifyRequest{
				CaptchaID: req.CaptchaID,
				Clicks:    clicks,
			})

		case "puzzle":
			result, err = client.VerifyPuzzle(ctx, &captchax.PuzzleVerifyRequest{
				CaptchaID: req.CaptchaID,
				TargetX:  req.TargetX,
				TargetY:  req.TargetY,
			})

		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的验证码类型"})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, VerifyResponse{
			Success: result.Success,
			Message: result.Message,
			Score:   result.Score,
		})
	})

	r.POST("/api/login", func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
			return
		}

		ctx := context.Background()
		valid, err := client.VerifyToken(ctx, req.CaptchaToken)
		if err != nil {
			log.Printf("验证失败: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "验证码验证失败",
			})
			return
		}

		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "验证码无效或已过期",
			})
			return
		}

		if req.Username == "admin" && req.Password == "admin123" {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "登录成功",
				"token":   "mock-jwt-token",
			})
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "用户名或密码错误",
		})
	})

	r.POST("/api/captcha/token/verify", func(c *gin.Context) {
		token := c.GetHeader("X-Captcha-Token")
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Token 不能为空",
			})
			return
		}

		ctx := context.Background()
		valid, err := client.VerifyToken(ctx, token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "验证服务错误",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": valid,
			"message": ifThen(valid, "验证成功", "验证失败"),
		})
	})

	log.Println("服务器启动在 http://localhost:8081")
	log.Fatal(r.Run(":8081"))
}

func ifThen(cond bool, thenStr, elseStr string) string {
	if cond {
		return thenStr
	}
	return elseStr
}

func postJSON(url string, data interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.Bytes(), nil
}
