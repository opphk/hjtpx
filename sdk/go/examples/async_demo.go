package main

import (
	"fmt"
	"time"

	captchago "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
	fmt.Println("======================================")
	fmt.Println("  HJT Captcha SDK Async Demo")
	fmt.Println("======================================")
	fmt.Println()

	// 创建异步客户端
	asyncClient := captchago.NewAsyncClient("http://localhost:8080")

	fmt.Println("1. 异步获取单个验证码")
	asyncExampleSingle(asyncClient)
	fmt.Println()

	fmt.Println("2. 异步批量获取验证码")
	asyncExampleBatch(asyncClient)
	fmt.Println()

	fmt.Println("3. 异步验证验证码")
	asyncExampleVerify(asyncClient)
	fmt.Println()

	fmt.Println("======================================")
	fmt.Println("  Async Demo completed successfully!")
	fmt.Println("======================================")
}

func asyncExampleSingle(client *captchago.AsyncClient) {
	fmt.Println("[Single Async Request]")

	// 异步获取滑块验证码
	resultChan := client.GetSliderCaptchaAsync(320, 160, 8)

	// 可以在此执行其他操作...
	fmt.Println("  Fetching captcha asynchronously...")
	time.Sleep(100 * time.Millisecond)
	fmt.Println("  Doing other work while waiting...")

	// 等待结果
	result := <-resultChan
	if result.Error != nil {
		fmt.Printf("  Error: %v\n", result.Error)
		return
	}

	fmt.Printf("  Session ID: %s\n", result.Data.SessionID)
	fmt.Printf("  Image URL: %s\n", result.Data.ImageURL)
}

func asyncExampleBatch(client *captchago.AsyncClient) {
	fmt.Println("[Batch Async Requests]")

	// 批量异步获取5个验证码
	channels := client.BatchGetSliderCaptcha(5, 320, 160, 8)
	fmt.Printf("  Started %d concurrent requests\n", len(channels))

	// 等待所有请求完成
	results, err := captchago.WaitAll(channels)
	if err != nil {
		fmt.Printf("  Error occurred: %v\n", err)
	}

	successCount := 0
	for i, result := range results {
		if result.Error != nil {
			fmt.Printf("  Request %d failed: %v\n", i+1, result.Error)
		} else {
			fmt.Printf("  Request %d success: %s\n", i+1, result.Data.SessionID[:10]+"...")
			successCount++
		}
	}

	fmt.Printf("  Success rate: %d/%d\n", successCount, len(results))
}

func asyncExampleVerify(client *captchago.AsyncClient) {
	fmt.Println("[Async Verification]")

	// 先获取验证码
	getResult := <-client.GetSliderCaptchaAsync(320, 160, 8)
	if getResult.Error != nil {
		fmt.Printf("  Failed to get captcha: %v\n", getResult.Error)
		return
	}

	sessionID := getResult.Data.SessionID
	fmt.Printf("  Got captcha: %s\n", sessionID[:10]+"...")

	// 异步验证
	verifyReq := &captchago.VerifyCaptchaRequest{
		SessionID: sessionID,
		X:         185,
		Y:         getResult.Data.SecretY,
		Trajectory: []captchago.TrajectoryPoint{
			{X: 0, Y: getResult.Data.SecretY, T: time.Now().UnixMilli() - 1000},
			{X: 50, Y: getResult.Data.SecretY + 5, T: time.Now().UnixMilli() - 800},
			{X: 100, Y: getResult.Data.SecretY - 3, T: time.Now().UnixMilli() - 500},
			{X: 150, Y: getResult.Data.SecretY + 2, T: time.Now().UnixMilli() - 200},
			{X: 185, Y: getResult.Data.SecretY, T: time.Now().UnixMilli()},
		},
	}

	verifyChan := client.VerifyCaptchaAsync(verifyReq)
	fmt.Println("  Verifying asynchronously...")

	// 等待验证结果
	verifyResult := <-verifyChan
	if verifyResult.Error != nil {
		fmt.Printf("  Verification failed: %v\n", verifyResult.Error)
		return
	}

	fmt.Printf("  Success: %v\n", verifyResult.Data.Success)
	fmt.Printf("  Message: %s\n", verifyResult.Data.Message)
}