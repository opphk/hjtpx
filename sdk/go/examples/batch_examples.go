package captcha

import (
	"context"
	"fmt"
	"log"
	"time"
)

func ExampleBatchVerify() {
	fmt.Println("=== Go SDK 批量请求示例 ===")

	client := NewClient("http://localhost:8080")

	batchClient := NewBatchClient(client, 5, 3)

	requests := []BatchRequest{
		{
			SessionID: "session-001",
			Type:      "slider",
			Data: map[string]interface{}{
				"x": 150,
				"y": 50,
			},
		},
		{
			SessionID: "session-002",
			Type:      "slider",
			Data: map[string]interface{}{
				"x": 180,
				"y": 60,
			},
		},
		{
			SessionID: "session-003",
			Type:      "slider",
			Data: map[string]interface{}{
				"x": 120,
				"y": 45,
			},
		},
	}

	ctx := context.Background()
	result := batchClient.BatchVerify(ctx, requests)

	fmt.Printf("总请求数: %d\n", result.Total)
	fmt.Printf("成功数: %d\n", result.Successful)
	fmt.Printf("失败数: %d\n", result.Failed)

	for i, resp := range result.Results {
		if resp.Success {
			fmt.Printf("请求 %d: 成功\n", i)
		} else {
			fmt.Printf("请求 %d: 失败 - %v\n", i, resp.Error)
		}
	}
}

func ExampleBulkGetCaptcha() {
	fmt.Println("=== Go SDK 批量获取验证码示例 ===")

	client := NewClient("http://localhost:8080")

	requests := []BulkCaptchaRequest{
		{Type: "slider", Width: 320, Height: 160, Tolerance: 8},
		{Type: "slider", Width: 400, Height: 200, Tolerance: 10},
		{Type: "click", MaxPoints: 4, Mode: "number"},
		{Type: "click", MaxPoints: 3, Mode: "chinese"},
	}

	ctx := context.Background()
	responses := client.BulkGetCaptcha(ctx, requests)

	for _, resp := range responses {
		if resp.Error != nil {
			fmt.Printf("获取验证码 %d 失败: %v\n", resp.Index, resp.Error)
		} else {
			fmt.Printf("验证码 %d: SessionID=%s, ImageURL=%s\n",
				resp.Index, resp.SessionID, resp.ImageURL)
		}
	}
}

func ExampleBatchVerifyCaptcha() {
	fmt.Println("=== Go SDK 批量验证API示例 ===")

	client := NewClient("http://localhost:8080")

	req := &BatchVerifyRequest{
		Items: []BatchVerifyItem{
			{
				Index:     0,
				SessionID: "session-001",
				Type:      "slider",
				X:         150,
				Y:         50,
				Trajectory: []TrajectoryPoint{
					{X: 0, Y: 50, T: time.Now().UnixMilli() - 1000},
					{X: 50, Y: 52, T: time.Now().UnixMilli() - 800},
					{X: 100, Y: 48, T: time.Now().UnixMilli() - 500},
					{X: 150, Y: 50, T: time.Now().UnixMilli()},
				},
			},
			{
				Index:     1,
				SessionID: "session-002",
				Type:      "slider",
				X:         180,
				Y:         60,
			},
		},
	}

	resp, err := client.BatchVerifyCaptcha(req)
	if err != nil {
		log.Printf("批量验证失败: %v", err)
		return
	}

	fmt.Println("批量验证结果:")
	for _, item := range resp.Results {
		status := "失败"
		if item.Success {
			status = "成功"
		}
		fmt.Printf("  索引 %d: %s - %s (剩余尝试: %d)\n",
			item.Index, status, item.Message, item.Remaining)
	}
}

func ExampleErrorHandling() {
	fmt.Println("=== Go SDK 错误处理示例 ===")

	client := NewClient("http://localhost:8080")

	verifyReq := &VerifyCaptchaRequest{
		SessionID: "invalid-session",
		X:         100,
	}

	result, err := client.VerifyCaptcha(verifyReq)
	if err != nil {
		fmt.Printf("验证失败: %v\n", err)

		if IsSDKError(err) {
			code := GetSDKErrorCode(err)
			fmt.Printf("错误码: %d\n", code)

			switch code {
			case StatusUnauthorized:
				fmt.Println("处理: API密钥无效或已过期")
			case StatusRateLimited:
				fmt.Println("处理: 请求频率超限，需要降速")
			case StatusInternalError:
				fmt.Println("处理: 服务器内部错误，稍后重试")
			default:
				fmt.Println("处理: 其他错误")
			}
		}

		if IsRetryableError(err) {
			fmt.Println("该错误可重试")
			delay := RetryStrategy(1, 100*time.Millisecond)
			fmt.Printf("建议等待: %v\n", delay)
		}
	} else {
		fmt.Printf("验证成功: %v\n", result.Success)
	}
}

func ExampleAdvancedErrorHandling() {
	fmt.Println("=== Go SDK 高级错误处理示例 ===")

	client := NewClient("http://localhost:8080")

	requests := []BatchRequest{
		{SessionID: "session-1", Type: "slider", Data: map[string]interface{}{"x": 100}},
		{SessionID: "session-2", Type: "slider", Data: map[string]interface{}{"x": 150}},
		{SessionID: "session-3", Type: "slider", Data: map[string]interface{}{"x": 200}},
	}

	ctx := context.Background()
	batchClient := NewBatchClient(client, 5, 2)
	result := batchClient.BatchVerify(ctx, requests)

	errorCounts := make(map[string]int)
	for _, resp := range result.Results {
		if !resp.Success && resp.Error != nil {
			errType := fmt.Sprintf("%T", resp.Error)
			errorCounts[errType]++
		}
	}

	fmt.Println("错误统计:")
	for errType, count := range errorCounts {
		fmt.Printf("  %s: %d次\n", errType, count)
	}

	fmt.Printf("\n成功率: %.2f%%\n", float64(result.Successful)/float64(result.Total)*100)
}

func ExampleSDKErrorCreation() {
	fmt.Println("=== Go SDK 错误创建示例 ===")

	err1 := NewSDKError(400, "Invalid parameters")
	fmt.Printf("简单错误: %v\n", err1)

	err2 := NewSDKErrorWithCause(500, "Internal error", fmt.Errorf("database connection failed"))
	fmt.Printf("带原因的错误: %v\n", err2)
	fmt.Printf("  原因: %v\n", err2.Unwrap())

	err3 := NewSDKErrorFromResponse(401, "Unauthorized")
	fmt.Printf("从响应创建的错误: %v\n", err3)

	wrapped := WrapError(404, "Session not found", ErrServerError)
	fmt.Printf("包装的错误: %v\n", wrapped)

	if IsSDKError(wrapped) {
		fmt.Println("是SDK错误")
	}
}

func ExampleClientWithTimeout() {
	fmt.Println("=== Go SDK 超时配置示例 ===")

	client := NewClient(
		"http://localhost:8080",
		WithTimeout(10*time.Second),
		WithAPIKey("your-api-key"),
	)

	captcha, err := client.GetSliderCaptcha(320, 160, 8)
	if err != nil {
		log.Printf("获取验证码失败: %v", err)
		return
	}

	fmt.Printf("获取成功: %s\n", captcha.SessionID)
}

func ExampleConcurrentBatchProcessing() {
	fmt.Println("=== Go SDK 并发批量处理示例 ===")

	client := NewClient("http://localhost:8080")
	batchClient := NewBatchClient(client, 10, 3)

	batchCount := 5
	requestsPerBatch := 10

	for batch := 0; batch < batchCount; batch++ {
		requests := make([]BatchRequest, requestsPerBatch)
		for i := 0; i < requestsPerBatch; i++ {
			requests[i] = BatchRequest{
				SessionID: fmt.Sprintf("batch-%d-session-%d", batch, i),
				Type:      "slider",
				Data: map[string]interface{}{
					"x": 100 + (i * 10),
				},
			}
		}

		ctx := context.Background()
		result := batchClient.BatchVerify(ctx, requests)

		fmt.Printf("批次 %d: 成功 %d/%d (成功率: %.1f%%)\n",
			batch,
			result.Successful,
			result.Total,
			float64(result.Successful)/float64(result.Total)*100)
	}
}

func main() {
	fmt.Println("======================================")
	fmt.Println("Go SDK 完整示例")
	fmt.Println("======================================")

	ExampleBatchVerify()
	fmt.Println()

	ExampleBulkGetCaptcha()
	fmt.Println()

	ExampleBatchVerifyCaptcha()
	fmt.Println()

	ExampleErrorHandling()
	fmt.Println()

	ExampleAdvancedErrorHandling()
	fmt.Println()

	ExampleSDKErrorCreation()
	fmt.Println()

	ExampleClientWithTimeout()
	fmt.Println()

	ExampleConcurrentBatchProcessing()
	fmt.Println()

	fmt.Println("======================================")
	fmt.Println("示例执行完成")
	fmt.Println("======================================")
}
