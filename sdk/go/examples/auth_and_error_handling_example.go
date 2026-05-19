package main

import (
	"context"
	"fmt"
	"log"
	"time"

	captchago "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
	fmt.Println("======================================")
	fmt.Println("  Go SDK 认证和错误处理示例")
	fmt.Println("======================================")
	fmt.Println()

	cfg := &captchago.Config{
		BaseURL:     "http://localhost:8080",
		MaxRetries:  3,
		HTTPTimeout: 10 * time.Second,
		DebugMode:   true,
	}

	client := captchago.NewCaptchaClient("demo-app-id", "demo-app-secret", cfg)
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Printf("Error closing client: %v\n", err)
		}
	}()

	fmt.Println("1. 错误处理示例")
	fmt.Println("-----------------------------------")
	demonstrateErrorHandling(client)
	fmt.Println()

	fmt.Println("2. 认证服务示例")
	fmt.Println("-----------------------------------")
	demonstrateAuthService(client)
	fmt.Println()

	fmt.Println("======================================")
	fmt.Println("  认证和错误处理示例完成")
	fmt.Println("======================================")
}

func demonstrateErrorHandling(client *captchago.CaptchaClient) {
	fmt.Println("测试各种错误场景...")

	fmt.Println("\n1.1 测试空 challenge ID...")
	err := client.VerifySliderCaptcha("", "100")
	handleError(err, "空 challenge ID")

	fmt.Println("\n1.2 测试空 answer...")
	err = client.VerifySliderCaptcha("test-id", "")
	handleError(err, "空 answer")

	fmt.Println("\n1.3 测试空点击数据...")
	err = client.VerifyClickCaptcha("test-id", []captchago.ClickData{})
	handleError(err, "空点击数据")

	fmt.Println("\n1.4 测试无效 data URI...")
	_, err = client.ExtractBase64Image("invalid-data")
	handleError(err, "无效 data URI")

	fmt.Println("\n1.5 测试空 data URI...")
	_, err = client.ExtractBase64Image("")
	handleError(err, "空 data URI")

	fmt.Println("\n1.6 错误重试性检查...")
	testRetryableErrors()
}

func handleError(err error, scenario string) {
	if err != nil {
		fmt.Printf("  ✓ %s - 捕获到错误: %v\n", scenario, err)

		if captchago.IsSDKError(err) {
			code := captchago.GetSDKErrorCode(err)
			msg := captchago.GetSDKErrorMessage(err)
			fmt.Printf("    - Error code: %d\n", code)
			fmt.Printf("    - Error message: %s\n", msg)
		}
	} else {
		fmt.Printf("  ✗ %s - 未捕获到错误\n", scenario)
	}
}

func testRetryableErrors() {
	testErrors := []struct {
		err      error
		desc     string
	}{
		{captchago.ErrNetworkError, "网络错误"},
		{captchago.ErrTimeout, "超时错误"},
		{captchago.ErrRateLimited, "限流错误"},
		{fmt.Errorf("connection refused"), "连接拒绝"},
		{fmt.Errorf("timeout error"), "超时"},
	}

	for _, te := range testErrors {
		retryable := captchago.IsRetryableError(te.err)
		fmt.Printf("  %s -> 可重试: %v\n", te.desc, retryable)
	}
}

func demonstrateAuthService(client *captchago.CaptchaClient) {
	ctx := context.Background()
	authService := client.Auth()

	fmt.Println("2.1 用户登录...")
	loginReq := &captchago.LoginRequest{
		Username:     "testuser",
		Password:     "testpassword",
		CaptchaToken: "",
	}

	loginResp, err := authService.Login(ctx, loginReq)
	if err != nil {
		log.Printf("登录失败: %v", err)
		fmt.Println("  (可能服务器未运行或配置错误)")
	} else {
		fmt.Printf("  ✓ 登录成功!\n")
		fmt.Printf("    - Access Token: %s...\n", truncateString(loginResp.AccessToken, 20))
		fmt.Printf("    - User ID: %d\n", loginResp.User.ID)
		fmt.Printf("    - Username: %s\n", loginResp.User.Username)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
