package main

import (
	"fmt"
	"time"

	captchago "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
	fmt.Println("======================================")
	fmt.Println("  HJT Captcha Advanced SDK Demo")
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

	advancedSliderDemo(client)
	fmt.Println()

	advancedClickDemo(client)
	fmt.Println()

	advancedImageDemo(client)
	fmt.Println()

	errorHandlingAdvancedDemo(client)
	fmt.Println()

	statsAndMonitoringDemo(client)
	fmt.Println()

	connectionPoolDemo(client)
	fmt.Println()

	fmt.Println("======================================")
	fmt.Println("  Advanced Demo completed!")
	fmt.Println("======================================")
}

func advancedSliderDemo(client *captchago.CaptchaClient) {
	fmt.Println("[Advanced Slider Captcha Demo]")

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

	positions := []string{"80", "100", "120", "140", "160", "180"}
	for _, pos := range positions {
		fmt.Printf("  Testing slider position: %s\n", pos)
		verifyResp, err := client.VerifySliderCaptcha(sliderResp.ChallengeID, pos)
		if err != nil {
			fmt.Printf("    Error: %v\n", err)
			continue
		}
		fmt.Printf("    Result: %v, Score: %.2f\n", verifyResp.Success, verifyResp.Score)
	}

	stats := client.GetStats()
	fmt.Printf("  Total requests so far: %d\n", stats.TotalRequests)
	fmt.Printf("  Success rate: %.2f%%\n", stats.SuccessRate)
}

func advancedClickDemo(client *captchago.CaptchaClient) {
	fmt.Println("[Advanced Click Captcha Demo]")

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

	fmt.Println("  Testing various click patterns...")

	patterns := [][]captchago.ClickData{
		{{X: clickResp.IconPositions[clickResp.TargetIndex][0], Y: clickResp.IconPositions[clickResp.TargetIndex][1], Duration: 100}},
		{{X: clickResp.IconPositions[clickResp.TargetIndex][0], Y: clickResp.IconPositions[clickResp.TargetIndex][1], Duration: 200}, {X: clickResp.IconPositions[clickResp.TargetIndex][0] + 5, Y: clickResp.IconPositions[clickResp.TargetIndex][1] + 5, Duration: 150}},
		{{X: 50, Y: 50, Duration: 300}},
	}

	for i, clicks := range patterns {
		fmt.Printf("  Pattern %d:\n", i+1)
		verifyResp, err := client.VerifyClickCaptcha(clickResp.ChallengeID, clicks)
		if err != nil {
			fmt.Printf("    Error: %v\n", err)
			continue
		}
		fmt.Printf("    Result: %v, Score: %.2f\n", verifyResp.Success, verifyResp.Score)
	}
}

func advancedImageDemo(client *captchago.CaptchaClient) {
	fmt.Println("[Advanced Image Captcha Demo]")

	testCases := []struct {
		name string
		req  *captchago.ImageCaptchaRequest
	}{
		{
			name: "Default (Mixed, 4 chars)",
			req:  nil,
		},
		{
			name: "Numbers only (4 chars)",
			req: &captchago.ImageCaptchaRequest{
				Type:  captchago.CaptchaTypeNumber,
				Count: 4,
			},
		},
		{
			name: "Letters only (5 chars)",
			req: &captchago.ImageCaptchaRequest{
				Type:  captchago.CaptchaTypeLetter,
				Count: 5,
			},
		},
		{
			name: "Mixed with noise (6 chars)",
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
		{
			name: "High noise level",
			req: &captchago.ImageCaptchaRequest{
				Type:      captchago.CaptchaTypeMixed,
				Count:     4,
				NoiseMode: 8,
				LineMode:  6,
			},
		},
	}

	successCount := 0
	totalTests := 0

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

		testAnswers := []string{"demo", "1234", "ABCD", "test"}
		for _, answer := range testAnswers {
			totalTests++
			verifyResp, err := client.VerifyImageCaptcha(resp.ChallengeID, answer)
			if err != nil {
				fmt.Printf("  [%s] Verification error: %v\n", tc.name, err)
				continue
			}
			fmt.Printf("  [%s] Answer '%s' verification: %v\n", tc.name, answer, verifyResp.Success)
			if verifyResp.Success {
				successCount++
			}
		}
	}

	fmt.Printf("  Total verification tests: %d\n", totalTests)
	fmt.Printf("  Successful verifications: %d\n", successCount)
	if totalTests > 0 {
		fmt.Printf("  Success rate: %.2f%%\n", float64(successCount)/float64(totalTests)*100)
	}
}

func errorHandlingAdvancedDemo(client *captchago.CaptchaClient) {
	fmt.Println("[Advanced Error Handling Demo]")

	fmt.Println("Testing parameter validation...")

	paramTests := []struct {
		name      string
		testFunc  func() error
	}{
		{
			name: "Empty captchaID (slider)",
			testFunc: func() error {
				_, err := client.VerifySliderCaptcha("", "120")
				return err
			},
		},
		{
			name: "Empty answer (slider)",
			testFunc: func() error {
				_, err := client.VerifySliderCaptcha("test-id", "")
				return err
			},
		},
		{
			name: "Empty captchaID (click)",
			testFunc: func() error {
				_, err := client.VerifyClickCaptcha("", []captchago.ClickData{{X: 100, Y: 100}})
				return err
			},
		},
		{
			name: "Empty clicks (click)",
			testFunc: func() error {
				_, err := client.VerifyClickCaptcha("test-id", []captchago.ClickData{})
				return err
			},
		},
		{
			name: "Empty captchaID (image)",
			testFunc: func() error {
				_, err := client.VerifyImageCaptcha("", "answer")
				return err
			},
		},
		{
			name: "Empty answer (image)",
			testFunc: func() error {
				_, err := client.VerifyImageCaptcha("test-id", "")
				return err
			},
		},
	}

	errorCount := 0
	for _, test := range paramTests {
		err := test.testFunc()
		if err != nil {
			fmt.Printf("  [%s] Error caught: %v\n", test.name, err)
			if captchago.IsSDKError(err) {
				fmt.Printf("    Error code: %d\n", captchago.GetSDKErrorCode(err))
			}
			errorCount++
		} else {
			fmt.Printf("  [%s] No error (unexpected)\n", test.name)
		}
	}

	fmt.Printf("Total validation errors caught: %d/%d\n", errorCount, len(paramTests))

	fmt.Println("Testing image extraction edge cases...")
	extractionTests := []struct {
		name    string
		dataURI string
	}{
		{"Empty string", ""},
		{"Invalid format", "invalid-data"},
		{"Valid PNG", "data:image/png;base64,SGVsbG8gV29ybGQ="},
		{"Valid JPEG", "data:image/jpeg;base64,SGVsbG8gV29ybGQ="},
	}

	for _, test := range extractionTests {
		_, err := client.ExtractBase64Image(test.dataURI)
		if err != nil {
			fmt.Printf("  [%s] Error: %v\n", test.name, err)
		} else {
			fmt.Printf("  [%s] Success\n", test.name)
		}
	}

	fmt.Println("Error handling demo completed")
}

func statsAndMonitoringDemo(client *captchago.CaptchaClient) {
	fmt.Println("[Statistics and Monitoring Demo]")

	initialStats := client.GetStats()
	fmt.Println("Initial statistics:")
	fmt.Printf("  Total requests: %d\n", initialStats.TotalRequests)
	fmt.Printf("  Successful requests: %d\n", initialStats.SuccessfulRequests)
	fmt.Printf("  Failed requests: %d\n", initialStats.FailedRequests)
	fmt.Printf("  Retried requests: %d\n", initialStats.RetriedRequests)
	fmt.Printf("  Success rate: %.2f%%\n", initialStats.SuccessRate)
	fmt.Printf("  Active connections: %d\n", initialStats.ActiveConnections)
	fmt.Printf("  Idle connections: %d\n", initialStats.IdleConnections)

	if initialStats.LastError != nil {
		fmt.Printf("  Last error: %v\n", initialStats.LastError)
		fmt.Printf("  Last error time: %v\n", initialStats.LastErrorTime)
	}

	fmt.Println("Performing some test requests...")
	for i := 0; i < 5; i++ {
		_, err := client.GenerateImageCaptcha(nil)
		if err != nil {
			fmt.Printf("  Request %d error: %v\n", i+1, err)
		} else {
			fmt.Printf("  Request %d success\n", i+1)
		}
	}

	updatedStats := client.GetStats()
	fmt.Println("Updated statistics:")
	fmt.Printf("  Total requests: %d (+%d)\n", updatedStats.TotalRequests, updatedStats.TotalRequests-initialStats.TotalRequests)
	fmt.Printf("  Successful requests: %d (+%d)\n", updatedStats.SuccessfulRequests, updatedStats.SuccessfulRequests-initialStats.SuccessfulRequests)
	fmt.Printf("  Success rate: %.2f%%\n", updatedStats.SuccessRate)

	statsDiff := updatedStats.TotalRequests - initialStats.TotalRequests
	successDiff := updatedStats.SuccessfulRequests - initialStats.SuccessfulRequests
	if statsDiff > 0 {
		actualRate := float64(successDiff) / float64(statsDiff) * 100
		fmt.Printf("  Actual success rate: %.2f%%\n", actualRate)
	}
}

func connectionPoolDemo(client *captchago.CaptchaClient) {
	fmt.Println("[Connection Pool Configuration Demo]")

	fmt.Println("Current pool configuration:")
	stats := client.GetStats()
	fmt.Printf("  Active connections: %d\n", stats.ActiveConnections)
	fmt.Printf("  Idle connections: %d\n", stats.IdleConnections)

	configurations := []*captchago.Config{
		{
			MaxIdleConns:  5,
			MaxOpenConns:  50,
			MaxRetries:    1,
			RetryDelay:    50 * time.Millisecond,
		},
		{
			MaxIdleConns:  20,
			MaxOpenConns:  200,
			MaxRetries:    3,
			RetryDelay:    100 * time.Millisecond,
		},
		{
			MaxIdleConns:  50,
			MaxOpenConns:  500,
			MaxRetries:    5,
			RetryDelay:    200 * time.Millisecond,
		},
	}

	for i, cfg := range configurations {
		fmt.Printf("\nConfiguration %d:\n", i+1)
		fmt.Printf("  MaxIdleConns: %d\n", cfg.MaxIdleConns)
		fmt.Printf("  MaxOpenConns: %d\n", cfg.MaxOpenConns)
		fmt.Printf("  MaxRetries: %d\n", cfg.MaxRetries)

		if err := client.SetPoolConfig(cfg); err != nil {
			fmt.Printf("  Error updating config: %v\n", err)
			continue
		}

		fmt.Println("  Pool configuration updated successfully")

		for j := 0; j < 3; j++ {
			_, err := client.GenerateImageCaptcha(nil)
			if err != nil {
				fmt.Printf("  Request %d error: %v\n", j+1, err)
			}
		}

		updatedStats := client.GetStats()
		fmt.Printf("  Active connections: %d\n", updatedStats.ActiveConnections)
		fmt.Printf("  Idle connections: %d\n", updatedStats.IdleConnections)
	}

	fmt.Println("\nNote: Configuration changes take effect for new requests")
	fmt.Println("Connection pool demo completed")
}
