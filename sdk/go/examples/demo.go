package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	captchago "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
	fmt.Println("======================================")
	fmt.Println("  HJT Captcha SDK Demo")
	fmt.Println("======================================")
	fmt.Println()

	demoBasicUsage()
	fmt.Println()

	demoWithAPIKeys()
	fmt.Println()

	demoImageCaptchaWorkflow()
	fmt.Println()

	demoSliderCaptchaWorkflow()
	fmt.Println()

	demoClickCaptchaWorkflow()
	fmt.Println()

	demoCustomConfiguration()
	fmt.Println()

	demoErrorHandling()
	fmt.Println()

	demoAdvancedFeatures()
	fmt.Println()

	demoTimeoutConfiguration()
	fmt.Println()

	demoBatchGeneration()
	fmt.Println()

	demoSignatureDemo()
	fmt.Println()

	demoRetryConfiguration()
	fmt.Println()

	demoContextCancellation()
	fmt.Println()

	demoMockServer()
	fmt.Println()

	fmt.Println("======================================")
	fmt.Println("  Demo completed successfully!")
	fmt.Println("======================================")
}

func demoBasicUsage() {
	fmt.Println("[Basic Usage]")

	client := captchago.NewClient()

	resp, err := client.GenerateImageCaptcha(nil)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	fmt.Printf("  Challenge ID: %s\n", resp.ChallengeID)
	fmt.Printf("  Image length: %d bytes (base64)\n", len(resp.Image))

	imageData, err := client.ExtractBase64Image(resp.Image)
	if err != nil {
		fmt.Printf("  Error extracting image: %v\n", err)
		return
	}
	fmt.Printf("  Image decoded: %d bytes\n", len(imageData))
}

func demoWithAPIKeys() {
	fmt.Println("[With API Keys]")

	client := captchago.NewClient(
		captchago.WithAPIKey("your-api-key"),
		captchago.WithAPISecret("your-api-secret"),
		captchago.WithEndpoint("http://localhost:8080"),
		captchago.WithDebugMode(true),
	)

	resp, err := client.GenerateImageCaptcha(nil)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	fmt.Printf("  Generated with auth: %s\n", resp.ChallengeID)
}

func demoImageCaptchaWorkflow() {
	fmt.Println("[Image Captcha Workflow]")

	client := captchago.NewClient(
		captchago.WithEndpoint("http://localhost:8080"),
	)

	req := &captchago.ImageCaptchaRequest{
		Type:  captchago.CaptchaTypeMixed,
		Count: 6,
	}

	genResp, err := client.GenerateImageCaptcha(req)
	if err != nil {
		fmt.Printf("  Generation error: %v\n", err)
		return
	}

	fmt.Printf("  Generated captcha ID: %s\n", genResp.ChallengeID)

	imageData, err := client.ExtractBase64Image(genResp.Image)
	if err != nil {
		fmt.Printf("  Error extracting image: %v\n", err)
		return
	}
	_ = imageData
	fmt.Printf("  Image decoded: %d bytes\n", len(imageData))

	fmt.Printf("  Simulating user input: \"a1b2c3\"\n")
	verifyReq := &captchago.VerifyImageCaptchaRequest{
		ChallengeID: genResp.ChallengeID,
		Answer:     "a1b2c3",
	}

	verifyResp, err := client.VerifyImageCaptcha(verifyReq)
	if err != nil {
		fmt.Printf("  Verification error: %v\n", err)
		return
	}

	fmt.Printf("  Verification result: %v\n", verifyResp.Success)
}

func demoSliderCaptchaWorkflow() {
	fmt.Println("[Slider Captcha Workflow]")

	client := captchago.NewClient(
		captchago.WithEndpoint("http://localhost:8080"),
	)

	req := &captchago.SliderCaptchaRequest{
		Width:  300,
		Height: 200,
	}

	resp, err := client.GetSliderCaptcha(req)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	fmt.Printf("  Slider captcha ID: %s\n", resp.ChallengeID)
	fmt.Printf("  Slider size: %dx%d\n", resp.SliderWidth, resp.SliderHeight)

	if resp.BackgroundImage != "" {
		bgData, _ := client.ExtractBase64Image(resp.BackgroundImage)
		_ = bgData
		fmt.Printf("  Background image decoded: %d bytes\n", len(bgData))
	}

	if resp.SliderImage != "" {
		sliderData, _ := client.ExtractBase64Image(resp.SliderImage)
		_ = sliderData
		fmt.Printf("  Slider piece decoded: %d bytes\n", len(sliderData))
	}

	verifyReq := &captchago.VerifyCaptchaRequest{
		ChallengeID: resp.ChallengeID,
		Action:     "slide",
		Data: map[string]interface{}{
			"offset": 120,
		},
	}

	verifyResp, err := client.VerifyCaptcha(verifyReq)
	if err != nil {
		fmt.Printf("  Verification error: %v\n", err)
		return
	}

	fmt.Printf("  Slider verification result: %v\n", verifyResp.Success)
	if verifyResp.Score > 0 {
		fmt.Printf("  Verification score: %.2f\n", verifyResp.Score)
	}
}

func demoClickCaptchaWorkflow() {
	fmt.Println("[Click Captcha Workflow]")

	client := captchago.NewClient(
		captchago.WithEndpoint("http://localhost:8080"),
	)

	req := &captchago.ClickCaptchaRequest{
		Width:     400,
		Height:    300,
		IconCount: 9,
	}

	resp, err := client.GetClickCaptcha(req)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	fmt.Printf("  Click captcha ID: %s\n", resp.ChallengeID)
	fmt.Printf("  Target index: %d\n", resp.TargetIndex)
	fmt.Printf("  Number of icons: %d\n", len(resp.IconPositions))

	verifyReq := &captchago.VerifyCaptchaRequest{
		ChallengeID: resp.ChallengeID,
		Action:     "click",
		Data: map[string]interface{}{
			"click_index": resp.TargetIndex,
		},
	}

	verifyResp, err := client.VerifyCaptcha(verifyReq)
	if err != nil {
		fmt.Printf("  Verification error: %v\n", err)
		return
	}

	fmt.Printf("  Click verification result: %v\n", verifyResp.Success)
}

func demoCustomConfiguration() {
	fmt.Println("[Custom Configuration]")

	testCases := []struct {
		name string
		req  *captchago.ImageCaptchaRequest
	}{
		{
			name: "Number only",
			req: &captchago.ImageCaptchaRequest{
				Type:  captchago.CaptchaTypeNumber,
				Count: 4,
			},
		},
		{
			name: "Letter only",
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
				CustomSet: "ABCDEF12345",
				Count:     5,
			},
		},
	}

	client := captchago.NewClient()

	for _, tc := range testCases {
		resp, err := client.GenerateImageCaptcha(tc.req)
		if err != nil {
			fmt.Printf("  [%s] Error: %v\n", tc.name, err)
			continue
		}
		fmt.Printf("  [%s] Generated: %s\n", tc.name, resp.ChallengeID)
	}
}

func demoErrorHandling() {
	fmt.Println("[Error Handling]")

	client := captchago.NewClient(
		captchago.WithEndpoint("http://localhost:8080"),
	)

	_, err := client.VerifyImageCaptcha(nil)
	if sdkErr, ok := err.(*captchago.SDKError); ok {
		fmt.Printf("  Nil request error: code=%d, message=%s\n", sdkErr.Code, sdkErr.Message)
	} else {
		fmt.Printf("  Nil request error: %v\n", err)
	}

	_, err = client.VerifyImageCaptcha(&captchago.VerifyImageCaptchaRequest{})
	if sdkErr, ok := err.(*captchago.SDKError); ok {
		fmt.Printf("  Missing challenge_id error: code=%d, message=%s\n", sdkErr.Code, sdkErr.Message)
	} else {
		fmt.Printf("  Missing challenge_id error: %v\n", err)
	}

	_, err = client.VerifyImageCaptcha(&captchago.VerifyImageCaptchaRequest{
		ChallengeID: "test-id",
	})
	if sdkErr, ok := err.(*captchago.SDKError); ok {
		fmt.Printf("  Missing answer error: code=%d, message=%s\n", sdkErr.Code, sdkErr.Message)
	} else {
		fmt.Printf("  Missing answer error: %v\n", err)
	}

	_, err = client.ExtractBase64Image("")
	if sdkErr, ok := err.(*captchago.SDKError); ok {
		fmt.Printf("  Invalid image data error: code=%d, message=%s\n", sdkErr.Code, sdkErr.Message)
	} else {
		fmt.Printf("  Invalid image data error: %v\n", err)
	}
}

func demoAdvancedFeatures() {
	fmt.Println("[Advanced Features]")

	client := captchago.NewClient(
		captchago.WithEndpoint("http://localhost:8080"),
		captchago.WithAppID("app-12345"),
		captchago.WithAppSecret("app-secret-67890"),
		captchago.WithSignatureKey("signature-key-abcdef"),
		captchago.WithDebugMode(true),
	)

	client.EnableSignature(true)

	fmt.Printf("  App ID configured: %s\n", "app-12345")
	fmt.Printf("  Signature enabled: %v\n", true)

	signature := client.GenerateSignature("GET", "/api/v1/captcha/image", nil, nil)
	fmt.Printf("  Generated signature: %s\n", signature)

	resp, err := client.GenerateImageCaptcha(nil)
	if err != nil {
		fmt.Printf("  Error with signature: %v\n", err)
		return
	}

	fmt.Printf("  Generated with advanced features: %s\n", resp.ChallengeID)
}

func demoTimeoutConfiguration() {
	fmt.Println("[Timeout Configuration]")

	client := captchago.NewClient(
		captchago.WithTimeout(5*time.Second),
	)

	fmt.Printf("  Timeout configured: %v\n", 5*time.Second)
	_ = client
}

func demoBatchGeneration() {
	fmt.Println("[Batch Generation]")

	client := captchago.NewClient()

	for i := 1; i <= 3; i++ {
		resp, err := client.GenerateImageCaptcha(nil)
		if err != nil {
			fmt.Printf("  [%d] Error: %v\n", i, err)
			continue
		}
		fmt.Printf("  [%d] Generated: %s\n", i, resp.ChallengeID)
	}
}

func demoSignatureDemo() {
	fmt.Println("[Signature Demo]")

	client := captchago.NewClient(
		captchago.WithSignatureKey("my-secret-signature-key"),
		captchago.WithDebugMode(true),
	)

	signature := client.GenerateSignature("POST", "/api/v1/captcha/verify",
		map[string]string{"key": "value"},
		[]byte(`{"test":"data"}`))
	fmt.Printf("  Generated signature: %s\n", signature)

	signature2 := client.GenerateSignature("POST", "/api/v1/captcha/verify",
		map[string]string{"key": "value"},
		[]byte(`{"test":"data"}`))
	fmt.Printf("  Signature verification: %v\n", signature == signature2)

	params := map[string]string{"a": "1", "b": "2", "c": "3"}
	sig3 := client.GenerateSignature("GET", "/test", params, nil)
	fmt.Printf("  Signature with sorted params: %s\n", sig3)
}

func demoRetryConfiguration() {
	fmt.Println("[Retry Configuration]")

	retryConfig := captchago.DefaultRetryConfig()
	fmt.Printf("  Default max retries: %d\n", retryConfig.MaxRetries)
	fmt.Printf("  Default initial delay: %v\n", retryConfig.InitialDelay)
	fmt.Printf("  Default max delay: %v\n", retryConfig.MaxDelay)
	fmt.Printf("  Default backoff factor: %.2f\n", retryConfig.BackoffFactor)

	customConfig := &captchago.RetryConfig{
		MaxRetries:     5,
		InitialDelay:   200 * time.Millisecond,
		MaxDelay:       10 * time.Second,
		BackoffFactor:  2.5,
		RetryableCodes: []int{429, 500, 502, 503, 504},
	}
	fmt.Printf("  Custom max retries: %d\n", customConfig.MaxRetries)

	_ = captchago.NewClient(
		captchago.WithRetryConfig(customConfig),
	)

	fmt.Printf("  Should retry 500: %v\n", customConfig.ShouldRetry(500))
	fmt.Printf("  Should retry 400: %v\n", customConfig.ShouldRetry(400))
	fmt.Printf("  Should retry 429: %v\n", customConfig.ShouldRetry(429))

	delay := customConfig.NextDelay(0)
	fmt.Printf("  Delay for attempt 0: %v\n", delay)
	delay = customConfig.NextDelay(1)
	fmt.Printf("  Delay for attempt 1: %v\n", delay)
	delay = customConfig.NextDelay(2)
	fmt.Printf("  Delay for attempt 2: %v\n", delay)
}

func demoContextCancellation() {
	fmt.Println("[Context Cancellation]")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	customClient := &http.Client{
		Timeout: 50 * time.Millisecond,
	}

	client := captchago.NewClient()
	client.SetHTTPClient(customClient)

	done := make(chan bool, 1)

	go func() {
		_, err := client.GenerateImageCaptcha(nil)
		if err != nil {
			fmt.Printf("  Context cancelled or timeout: %v\n", err)
		}
		done <- true
	}()

	select {
	case <-done:
		fmt.Printf("  Request completed\n")
	case <-ctx.Done():
		fmt.Printf("  Context timeout: %v\n", ctx.Err())
	}
}

func demoMockServer() {
	fmt.Println("[Mock Server Demo]")

	mock := captchago.NewMockServer(18080)
	if err := mock.Start(); err != nil {
		fmt.Printf("  Failed to start mock server: %v\n", err)
		return
	}
	defer mock.Stop()

	time.Sleep(100 * time.Millisecond)

	client := captchago.NewClient(
		captchago.WithEndpoint("http://localhost:18080"),
		captchago.WithDebugMode(false),
	)

	resp, err := client.GenerateImageCaptcha(nil)
	if err != nil {
		fmt.Printf("  Mock server test failed: %v\n", err)
		return
	}
	fmt.Printf("  Mock server returned: %s\n", resp.ChallengeID)

	mock.SetCorrectAnswer("test-answer")

	verifyResp, err := client.VerifyImageCaptcha(&captchago.VerifyImageCaptchaRequest{
		ChallengeID: "test-id",
		Answer:     "test-answer",
	})
	if err != nil {
		fmt.Printf("  Verification failed: %v\n", err)
		return
	}
	fmt.Printf("  Mock verification result: %v\n", verifyResp.Success)
	fmt.Printf("  Total verify calls: %d\n", mock.VerifyCalls)
}

func demoProductionReady() {
	fmt.Println("[Production Ready Configuration]")

	endpoint := os.Getenv("CAPTCHA_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}

	apiKey := os.Getenv("CAPTCHA_API_KEY")
	apiSecret := os.Getenv("CAPTCHA_API_SECRET")

	_ = captchago.NewClient(
		captchago.WithEndpoint(endpoint),
		captchago.WithAPIKey(apiKey),
		captchago.WithAPISecret(apiSecret),
		captchago.WithTimeout(30*time.Second),
		captchago.WithRetryConfig(&captchago.RetryConfig{
			MaxRetries:     3,
			InitialDelay:   100 * time.Millisecond,
			MaxDelay:       5 * time.Second,
			BackoffFactor:  2.0,
			RetryableCodes: []int{429, 500, 502, 503, 504},
		}),
	)

	if apiKey == "" || apiSecret == "" {
		log.Println("Warning: API credentials not configured")
	}
}
