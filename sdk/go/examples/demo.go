package main

import (
	"fmt"
	"time"

	captchago "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
	fmt.Println("======================================")
	fmt.Println("  HJT Captcha SDK Demo")
	fmt.Println("======================================")
	fmt.Println()

	cfg := &captchago.Config{
		BaseURL:        "http://localhost:8080",
		MaxIdleConns:   10,
		MaxOpenConns:   100,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
		HTTPTimeout:     10 * time.Second,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		MaxRetries:      3,
		RetryDelay:      100 * time.Millisecond,
		DebugMode:       true,
	}

	client := captchago.NewCaptchaClient("your-app-id", "your-app-secret", cfg)
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Printf("Error closing client: %v\n", err)
		}
	}()

	fmt.Println("Client created successfully with configuration:")
	fmt.Printf("  - MaxIdleConns: %d\n", cfg.MaxIdleConns)
	fmt.Printf("  - MaxOpenConns: %d\n", cfg.MaxOpenConns)
	fmt.Printf("  - HTTPTimeout: %v\n", cfg.HTTPTimeout)
	fmt.Printf("  - MaxRetries: %d\n", cfg.MaxRetries)
	fmt.Println()

	sliderDemo(client)
	fmt.Println()

	clickDemo(client)
	fmt.Println()

	imageDemo(client)
	fmt.Println()

	errorHandlingDemo(client)
	fmt.Println()

	statsDemo(client)
	fmt.Println()

	poolConfigDemo(client)
	fmt.Println()

	fmt.Println("======================================")
	fmt.Println("  Demo completed successfully!")
	fmt.Println("======================================")
}

func sliderDemo(client *captchago.CaptchaClient) {
	fmt.Println("[Slider Captcha Demo]")

	sliderResp, err := client.GenerateSliderCaptcha()
	if err != nil {
		fmt.Printf("  Error generating slider captcha: %v\n", err)
		return
	}

	fmt.Printf("  Challenge ID: %s\n", sliderResp.ChallengeID)
	fmt.Printf("  Slider size: %dx%d\n", sliderResp.SliderWidth, sliderResp.SliderHeight)

	if sliderResp.BackgroundImage != "" {
		bgData, err := client.ExtractBase64Image(sliderResp.BackgroundImage)
		if err != nil {
			fmt.Printf("  Error extracting background image: %v\n", err)
		} else {
			fmt.Printf("  Background image size: %d bytes\n", len(bgData))
		}
	}

	if sliderResp.SliderImage != "" {
		sliderData, err := client.ExtractBase64Image(sliderResp.SliderImage)
		if err != nil {
			fmt.Printf("  Error extracting slider image: %v\n", err)
		} else {
			fmt.Printf("  Slider image size: %d bytes\n", len(sliderData))
		}
	}

	fmt.Printf("  Simulating user sliding to position: 120\n")
	verifyResp, err := client.VerifySliderCaptcha(sliderResp.ChallengeID, "120")
	if err != nil {
		fmt.Printf("  Error verifying slider: %v\n", err)
		return
	}

	fmt.Printf("  Verification result: %v\n", verifyResp.Success)
	if verifyResp.Score > 0 {
		fmt.Printf("  Score: %.2f\n", verifyResp.Score)
	}
	if verifyResp.RiskLevel != "" {
		fmt.Printf("  Risk level: %s\n", verifyResp.RiskLevel)
	}
}

func clickDemo(client *captchago.CaptchaClient) {
	fmt.Println("[Click Captcha Demo]")

	clickResp, err := client.GenerateClickCaptcha()
	if err != nil {
		fmt.Printf("  Error generating click captcha: %v\n", err)
		return
	}

	fmt.Printf("  Challenge ID: %s\n", clickResp.ChallengeID)
	fmt.Printf("  Target index: %d\n", clickResp.TargetIndex)
	fmt.Printf("  Number of icons: %d\n", len(clickResp.IconPositions))

	if len(clickResp.TargetPosition) == 2 {
		fmt.Printf("  Target position: [%d, %d]\n", clickResp.TargetPosition[0], clickResp.TargetPosition[1])
	}

	targetX := clickResp.IconPositions[clickResp.TargetIndex][0]
	targetY := clickResp.IconPositions[clickResp.TargetIndex][1]
	fmt.Printf("  Simulating click at target icon [%d, %d]\n", targetX, targetY)

	clicks := []captchago.ClickData{
		{X: targetX, Y: targetY, Duration: 500},
		{X: targetX + 5, Y: targetY + 5, Duration: 200},
	}

	verifyResp, err := client.VerifyClickCaptcha(clickResp.ChallengeID, clicks)
	if err != nil {
		fmt.Printf("  Error verifying click: %v\n", err)
		return
	}

	fmt.Printf("  Verification result: %v\n", verifyResp.Success)
	if verifyResp.Score > 0 {
		fmt.Printf("  Score: %.2f\n", verifyResp.Score)
	}
}

func imageDemo(client *captchago.CaptchaClient) {
	fmt.Println("[Image Captcha Demo]")

	testCases := []struct {
		name string
		req  *captchago.ImageCaptchaRequest
	}{
		{
			name: "Default",
			req:  nil,
		},
		{
			name: "Number only (4 chars)",
			req: &captchago.ImageCaptchaRequest{
				Type:  captchago.CaptchaTypeNumber,
				Count: 4,
			},
		},
		{
			name: "Letter only (5 chars)",
			req: &captchago.ImageCaptchaRequest{
				Type:  captchago.CaptchaTypeLetter,
				Count: 5,
			},
		},
		{
			name: "Mixed with noise",
			req: &captchago.ImageCaptchaRequest{
				Type:      captchago.CaptchaTypeMixed,
				Count:     6,
				NoiseMode: 3,
				LineMode:  2,
			},
		},
		{
			name: "Custom charset",
			req: &captchago.ImageCaptchaRequest{
				CustomSet: "ABCDEF12345!@#$%",
				Count:     5,
			},
		},
	}

	for _, tc := range testCases {
		resp, err := client.GenerateImageCaptcha(tc.req)
		if err != nil {
			fmt.Printf("  [%s] Error: %v\n", tc.name, err)
			continue
		}

		fmt.Printf("  [%s] Generated: %s\n", tc.name, resp.ChallengeID)

		imageData, err := client.ExtractBase64Image(resp.Image)
		if err != nil {
			fmt.Printf("  [%s] Error extracting image: %v\n", tc.name, err)
			continue
		}
		fmt.Printf("  [%s] Image size: %d bytes\n", tc.name, len(imageData))

		verifyReq := &captchago.VerifyImageCaptchaRequest{
			ChallengeID: resp.ChallengeID,
			Answer:     "demo",
		}
		verifyResp, err := client.VerifyImageCaptcha(verifyReq.ChallengeID, verifyReq.Answer)
		if err != nil {
			fmt.Printf("  [%s] Verification error: %v\n", tc.name, err)
			continue
		}
		fmt.Printf("  [%s] Verification result: %v\n", tc.name, verifyResp.Success)
	}
}

func errorHandlingDemo(client *captchago.CaptchaClient) {
	fmt.Println("[Error Handling Demo]")

	fmt.Println("Testing parameter validation...")

	_, err := client.VerifySliderCaptcha("", "120")
	if err != nil {
		fmt.Printf("  Empty captchaID: %v\n", err)
		if captchago.IsSDKError(err) {
			fmt.Printf("    Error code: %d\n", captchago.GetSDKErrorCode(err))
		}
	}

	_, err = client.VerifySliderCaptcha("test-id", "")
	if err != nil {
		fmt.Printf("  Empty answer: %v\n", err)
		if captchago.IsSDKError(err) {
			fmt.Printf("    Error code: %d\n", captchago.GetSDKErrorCode(err))
		}
	}

	_, err = client.VerifyClickCaptcha("", []captchago.ClickData{{X: 100, Y: 100}})
	if err != nil {
		fmt.Printf("  Empty captchaID (click): %v\n", err)
		if captchago.IsSDKError(err) {
			fmt.Printf("    Error code: %d\n", captchago.GetSDKErrorCode(err))
		}
	}

	_, err = client.VerifyClickCaptcha("test-id", []captchago.ClickData{})
	if err != nil {
		fmt.Printf("  Empty clicks: %v\n", err)
		if captchago.IsSDKError(err) {
			fmt.Printf("    Error code: %d\n", captchago.GetSDKErrorCode(err))
		}
	}

	_, err = client.VerifyImageCaptcha("", "answer")
	if err != nil {
		fmt.Printf("  Empty captchaID (image): %v\n", err)
		if captchago.IsSDKError(err) {
			fmt.Printf("    Error code: %d\n", captchago.GetSDKErrorCode(err))
		}
	}

	_, err = client.VerifyImageCaptcha("test-id", "")
	if err != nil {
		fmt.Printf("  Empty answer (image): %v\n", err)
		if captchago.IsSDKError(err) {
			fmt.Printf("    Error code: %d\n", captchago.GetSDKErrorCode(err))
		}
	}

	fmt.Println("Testing image extraction...")
	_, err = client.ExtractBase64Image("")
	if err != nil {
		fmt.Printf("  Empty data URI: %v\n", err)
	}

	_, err = client.ExtractBase64Image("invalid-data")
	if err != nil {
		fmt.Printf("  Invalid data format: %v\n", err)
	}

	fmt.Println("Error handling demo completed")
}

func statsDemo(client *captchago.CaptchaClient) {
	fmt.Println("[Statistics Demo]")

	stats := client.GetStats()
	fmt.Printf("  Total requests: %d\n", stats.TotalRequests)
	fmt.Printf("  Successful requests: %d\n", stats.SuccessfulRequests)
	fmt.Printf("  Failed requests: %d\n", stats.FailedRequests)
	fmt.Printf("  Retried requests: %d\n", stats.RetriedRequests)
	fmt.Printf("  Success rate: %.2f%%\n", stats.SuccessRate)
	fmt.Printf("  Active connections: %d\n", stats.ActiveConnections)
	fmt.Printf("  Idle connections: %d\n", stats.IdleConnections)

	if stats.LastError != nil {
		fmt.Printf("  Last error: %v\n", stats.LastError)
		fmt.Printf("  Last error time: %v\n", stats.LastErrorTime)
	}
}

func poolConfigDemo(client *captchago.CaptchaClient) {
	fmt.Println("[Pool Configuration Demo]")

	fmt.Println("Current pool configuration:")
	stats := client.GetStats()
	fmt.Printf("  Active connections: %d\n", stats.ActiveConnections)
	fmt.Printf("  Idle connections: %d\n", stats.IdleConnections)

	fmt.Println("Updating pool configuration...")
	newCfg := &captchago.Config{
		MaxIdleConns:  20,
		MaxOpenConns: 200,
		MaxRetries:   5,
		RetryDelay:   200 * time.Millisecond,
	}

	if err := client.SetPoolConfig(newCfg); err != nil {
		fmt.Printf("  Error updating config: %v\n", err)
		return
	}

	fmt.Println("Pool configuration updated successfully")
	fmt.Println("Note: Configuration changes take effect for new requests")
}

func retryDemo(client *captchago.CaptchaClient) {
	fmt.Println("[Retry Mechanism Demo]")

	fmt.Printf("Max retries configured: %d\n", 3)
	fmt.Println("The SDK will automatically retry on:")
	fmt.Println("  - HTTP 5xx errors")
	fmt.Println("  - HTTP 429 (rate limited)")
	fmt.Println("  - Network timeouts")
	fmt.Println("  - Connection failures")
	fmt.Println()
	fmt.Println("Retry delay uses exponential backoff")
	fmt.Println("Example: 100ms, 200ms, 400ms for attempts 1, 2, 3")
}
