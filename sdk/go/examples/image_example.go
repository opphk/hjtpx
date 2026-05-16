package main

import (
	"fmt"
	"time"

	sdk "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
	cfg := &sdk.Config{
		BaseURL:     "http://localhost:8080",
		HTTPTimeout: 10 * time.Second,
		DebugMode:   true,
	}

	client := sdk.NewCaptchaClient("", "", cfg)
	defer client.Close()

	fmt.Println("=== Image Captcha Example ===")

	req := &sdk.ImageCaptchaRequest{
		Type:      sdk.CaptchaTypeMixed,
		Count:     4,
		NoiseMode: 2,
		LineMode:  1,
	}

	captcha, err := client.GenerateImageCaptcha(req)
	if err != nil {
		fmt.Printf("✗ Image generation failed: %v\n", err)
		return
	}
	fmt.Printf("✓ Image Captcha ID: %s\n", captcha.ChallengeID)

	result, err := client.VerifyImageCaptcha(captcha.ChallengeID, "1234")
	if err != nil {
		fmt.Printf("✗ Image verification failed: %v\n", err)
		return
	}
	fmt.Printf("✓ Verification success: %v\n", result.Success)

	stats := client.GetStats()
	fmt.Printf("\n📊 Statistics:\n")
	fmt.Printf("  Total Requests: %d\n", stats.TotalRequests)
	fmt.Printf("  Success Rate: %.2f%%\n", stats.SuccessRate)
}
