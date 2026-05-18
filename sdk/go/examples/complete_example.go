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
	fmt.Println("  HJT Captcha Go SDK Complete Demo")
	fmt.Println("======================================")
	fmt.Println()

	cfg := &captchago.Config{
		BaseURL:        "http://localhost:8080",
		MaxIdleConns:   10,
		MaxOpenConns:   100,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
		HTTPTimeout:     10 * time.Second,
		MaxRetries:      3,
		RetryDelay:      100 * time.Millisecond,
		DebugMode:       true,
	}

	client := captchago.NewCaptchaClient("demo-app-id", "demo-app-secret", cfg)
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Printf("Error closing client: %v\n", err)
		}
	}()

	fmt.Println("Client created successfully with configuration:")
	fmt.Printf("  - BaseURL: %s\n", cfg.BaseURL)
	fmt.Printf("  - MaxRetries: %d\n", cfg.MaxRetries)
	fmt.Printf("  - HTTPTimeout: %v\n", cfg.HTTPTimeout)
	fmt.Println()

	sliderCompleteExample(client)
	fmt.Println()

	clickCompleteExample(client)
	fmt.Println()

	imageCompleteExample(client)
	fmt.Println()

	advancedErrorHandlingExample(client)
	fmt.Println()

	concurrentRequestsExample(client)
	fmt.Println()

	fmt.Println("======================================")
	fmt.Println("  All examples completed!")
	fmt.Println("======================================")
}

func sliderCompleteExample(client *captchago.CaptchaClient) {
	fmt.Println("[Complete Slider Captcha Example]")
	fmt.Println("-----------------------------------")

	ctx := context.Background()

	fmt.Println("Step 1: Generating slider captcha...")
	slider, err := client.GenerateSliderCaptchaWithContext(ctx)
	if err != nil {
		log.Printf("Error generating slider: %v", err)
		return
	}

	fmt.Printf("✓ Challenge ID: %s\n", slider.ChallengeID)
	fmt.Printf("✓ Slider size: %dx%d\n", slider.SliderWidth, slider.SliderHeight)
	fmt.Printf("✓ Background size: %dx%d\n", slider.BackgroundWidth, slider.BackgroundHeight)
	fmt.Printf("✓ Secret position: (%d, %d)\n", slider.SecretX, slider.SecretY)

	fmt.Println("\nStep 2: Extracting images...")
	if slider.BackgroundImage != "" {
		bgData, err := client.ExtractBase64Image(slider.BackgroundImage)
		if err != nil {
			fmt.Printf("  ✗ Error extracting background: %v\n", err)
		} else {
			fmt.Printf("  ✓ Background image extracted: %d bytes\n", len(bgData))
		}
	}

	if slider.SliderImage != "" {
		sliderData, err := client.ExtractBase64Image(slider.SliderImage)
		if err != nil {
			fmt.Printf("  ✗ Error extracting slider: %v\n", err)
		} else {
			fmt.Printf("  ✓ Slider image extracted: %d bytes\n", len(sliderData))
		}
	}

	fmt.Println("\nStep 3: Simulating user verification...")
	fmt.Printf("  Simulating slide to position: %d\n", slider.SecretX)
	result, err := client.VerifySliderCaptchaWithContext(ctx, slider.ChallengeID, fmt.Sprintf("%d", slider.SecretX))
	if err != nil {
		fmt.Printf("  ✗ Verification error: %v\n", err)
		return
	}

	fmt.Printf("  ✓ Verification success: %v\n", result.Success)
	if result.Message != "" {
		fmt.Printf("  ✓ Message: %s\n", result.Message)
	}
	if result.Score > 0 {
		fmt.Printf("  ✓ Score: %.2f\n", result.Score)
	}
	if result.RiskLevel != "" {
		fmt.Printf("  ✓ Risk level: %s\n", result.RiskLevel)
	}
}

func clickCompleteExample(client *captchago.CaptchaClient) {
	fmt.Println("[Complete Click Captcha Example]")
	fmt.Println("-----------------------------------")

	ctx := context.Background()

	fmt.Println("Step 1: Generating click captcha...")
	click, err := client.GenerateClickCaptchaWithContext(ctx)
	if err != nil {
		log.Printf("Error generating click captcha: %v", err)
		return
	}

	fmt.Printf("✓ Challenge ID: %s\n", click.ChallengeID)
	fmt.Printf("✓ Image URL: %s\n", click.ImageURL)
	fmt.Printf("✓ Total icons: %d\n", click.TotalIcons)
	fmt.Printf("✓ Target index: %d\n", click.TargetIndex)

	if len(click.IconPositions) > 0 {
		fmt.Printf("✓ Icon positions: %d icons\n", len(click.IconPositions))
	}

	if len(click.TargetPosition) == 2 {
		fmt.Printf("✓ Target position: [%d, %d]\n", click.TargetPosition[0], click.TargetPosition[1])
	}

	fmt.Println("\nStep 2: Simulating user clicks...")
	targetX := click.IconPositions[click.TargetIndex][0]
	targetY := click.IconPositions[click.TargetIndex][1]
	fmt.Printf("  Clicking at icon %d: [%d, %d]\n", click.TargetIndex, targetX, targetY)

	clicks := []captchago.ClickData{
		{X: targetX, Y: targetY, Duration: 500},
		{X: targetX + 3, Y: targetY + 3, Duration: 200},
	}

	fmt.Println("\nStep 3: Verifying...")
	result, err := client.VerifyClickCaptchaWithContext(ctx, click.ChallengeID, clicks)
	if err != nil {
		fmt.Printf("  ✗ Verification error: %v\n", err)
		return
	}

	fmt.Printf("  ✓ Verification success: %v\n", result.Success)
	if result.Message != "" {
		fmt.Printf("  ✓ Message: %s\n", result.Message)
	}
}

func imageCompleteExample(client *captchago.CaptchaClient) {
	fmt.Println("[Complete Image Captcha Example]")
	fmt.Println("-----------------------------------")

	ctx := context.Background()

	testCases := []struct {
		name string
		req  *captchago.ImageCaptchaRequest
	}{
		{
			name: "Default mixed",
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
			name: "Chinese characters (3 chars)",
			req: &captchago.ImageCaptchaRequest{
				Type:  captchago.CaptchaTypeChinese,
				Count: 3,
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
				CustomSet: "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
				Count:     5,
			},
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\nTesting: %s\n", tc.name)

		image, err := client.GenerateImageCaptchaWithContext(ctx, tc.req)
		if err != nil {
			fmt.Printf("  ✗ Error: %v\n", err)
			continue
		}

		fmt.Printf("  ✓ Challenge ID: %s\n", image.ChallengeID)

		if image.Image != "" {
			imageData, err := client.ExtractBase64Image(image.Image)
			if err != nil {
				fmt.Printf("  ✗ Error extracting image: %v\n", err)
			} else {
				fmt.Printf("  ✓ Image size: %d bytes\n", len(imageData))
			}
		}

		fmt.Printf("  Testing verification with wrong answer...")
		result, err := client.VerifyImageCaptchaWithContext(ctx, image.ChallengeID, "WRONG")
		if err != nil {
			fmt.Printf("error: %v\n", err)
		} else {
			fmt.Printf("success: %v (expected: false)\n", result.Success)
		}
	}
}

func advancedErrorHandlingExample(client *captchago.CaptchaClient) {
	fmt.Println("[Advanced Error Handling Example]")
	fmt.Println("-----------------------------------")

	fmt.Println("\n1. Testing empty challenge ID...")
	err := client.VerifySliderCaptcha("", "100")
	if err != nil {
		fmt.Printf("  ✓ Caught error: %v\n", err)
		if captchago.IsSDKError(err) {
			code := captchago.GetSDKErrorCode(err)
			msg := captchago.GetSDKErrorMessage(err)
			fmt.Printf("  ✓ Error code: %d\n", code)
			fmt.Printf("  ✓ Error message: %s\n", msg)
		}
	}

	fmt.Println("\n2. Testing empty answer...")
	err = client.VerifySliderCaptcha("test-id", "")
	if err != nil {
		fmt.Printf("  ✓ Caught error: %v\n", err)
	}

	fmt.Println("\n3. Testing empty click data...")
	err = client.VerifyClickCaptcha("test-id", []captchago.ClickData{})
	if err != nil {
		fmt.Printf("  ✓ Caught error: %v\n", err)
	}

	fmt.Println("\n4. Testing empty image challenge ID...")
	err = client.VerifyImageCaptcha("", "answer")
	if err != nil {
		fmt.Printf("  ✓ Caught error: %v\n", err)
	}

	fmt.Println("\n5. Testing invalid data URI...")
	_, err = client.ExtractBase64Image("invalid-data")
	if err != nil {
		fmt.Printf("  ✓ Caught error: %v\n", err)
	}

	_, err = client.ExtractBase64Image("")
	if err != nil {
		fmt.Printf("  ✓ Caught error: %v\n", err)
	}

	fmt.Println("\n6. Error retryability check...")
	testErrors := []error{
		captchago.ErrNetworkError,
		captchago.ErrTimeout,
		captchago.ErrRateLimited,
		fmt.Errorf("connection refused"),
	}

	for _, testErr := range testErrors {
		retryable := captchago.IsRetryableError(testErr)
		fmt.Printf("  %v -> retryable: %v\n", testErr, retryable)
	}
}

func concurrentRequestsExample(client *captchago.CaptchaClient) {
	fmt.Println("[Concurrent Requests Example]")
	fmt.Println("-----------------------------------")

	ctx := context.Background()
	concurrency := 5

	fmt.Printf("Starting %d concurrent captcha requests...\n", concurrency)

	done := make(chan struct{})
	results := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			slider, err := client.GenerateSliderCaptchaWithContext(ctx)
			if err != nil {
				results <- fmt.Sprintf("Request %d: ERROR - %v", id, err)
				return
			}
			results <- fmt.Sprintf("Request %d: SUCCESS - %s", id, slider.ChallengeID[:8]+"...")
		}(i)
	}

	for i := 0; i < concurrency; i++ {
		select {
		case result := <-results:
			fmt.Printf("  %s\n", result)
		case <-time.After(10 * time.Second):
			fmt.Printf("  Request %d: TIMEOUT\n", i)
		}
	}

	fmt.Println("\nChecking client stats...")
	stats := client.GetStats()
	fmt.Printf("  Total requests: %d\n", stats.TotalRequests)
	fmt.Printf("  Successful: %d\n", stats.SuccessfulRequests)
	fmt.Printf("  Failed: %d\n", stats.FailedRequests)
	fmt.Printf("  Retried: %d\n", stats.RetriedRequests)
	fmt.Printf("  Success rate: %.2f%%\n", stats.SuccessRate)
	close(done)
}
