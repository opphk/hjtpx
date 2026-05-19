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
	fmt.Println("  Go SDK 所有验证码类型示例")
	fmt.Println("======================================")
	fmt.Println()

	cfg := &captchago.Config{
		BaseURL:      "http://localhost:8080",
		MaxRetries:   3,
		HTTPTimeout:  10 * time.Second,
		DebugMode:    true,
	}

	client := captchago.NewCaptchaClient("demo-app-id", "demo-app-secret", cfg)
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Printf("Error closing client: %v\n", err)
		}
	}()

	fmt.Println("1. 滑块验证码示例")
	fmt.Println("-----------------------------------")
	demonstrateSliderCaptcha(client)
	fmt.Println()

	fmt.Println("2. 点击验证码示例")
	fmt.Println("-----------------------------------")
	demonstrateClickCaptcha(client)
	fmt.Println()

	fmt.Println("3. 图形验证码示例")
	fmt.Println("-----------------------------------")
	demonstrateImageCaptcha(client)
	fmt.Println()

	fmt.Println("======================================")
	fmt.Println("  所有验证码类型示例完成")
	fmt.Println("======================================")
}

func demonstrateSliderCaptcha(client *captchago.CaptchaClient) {
	ctx := context.Background()

	slider, err := client.GenerateSliderCaptchaWithContext(ctx)
	if err != nil {
		log.Printf("Error generating slider: %v", err)
		return
	}

	fmt.Printf("✓ Challenge ID: %s\n", slider.ChallengeID)
	fmt.Printf("✓ Slider size: %dx%d\n", slider.SliderWidth, slider.SliderHeight)
	fmt.Printf("✓ Secret position: (%d, %d)\n", slider.SecretX, slider.SecretY)

	result, err := client.VerifySliderCaptchaWithContext(ctx, slider.ChallengeID, fmt.Sprintf("%d", slider.SecretX))
	if err != nil {
		fmt.Printf("✗ Verification error: %v\n", err)
		return
	}

	fmt.Printf("✓ Verification success: %v\n", result.Success)
	if result.Score > 0 {
		fmt.Printf("✓ Score: %.2f\n", result.Score)
	}
}

func demonstrateClickCaptcha(client *captchago.CaptchaClient) {
	ctx := context.Background()

	click, err := client.GenerateClickCaptchaWithContext(ctx)
	if err != nil {
		log.Printf("Error generating click captcha: %v", err)
		return
	}

	fmt.Printf("✓ Challenge ID: %s\n", click.ChallengeID)
	fmt.Printf("✓ Total icons: %d\n", click.TotalIcons)
	fmt.Printf("✓ Target index: %d\n", click.TargetIndex)

	if len(click.IconPositions) > click.TargetIndex {
		targetX := click.IconPositions[click.TargetIndex][0]
		targetY := click.IconPositions[click.TargetIndex][1]
		fmt.Printf("✓ Target position: [%d, %d]\n", targetX, targetY)

		clicks := []captchago.ClickData{
			{X: targetX, Y: targetY, Duration: 500},
		}

		result, err := client.VerifyClickCaptchaWithContext(ctx, click.ChallengeID, clicks)
		if err != nil {
			fmt.Printf("✗ Verification error: %v\n", err)
			return
		}

		fmt.Printf("✓ Verification success: %v\n", result.Success)
	}
}

func demonstrateImageCaptcha(client *captchago.CaptchaClient) {
	ctx := context.Background()

	testCases := []struct {
		name string
		req  *captchago.ImageCaptchaRequest
	}{
		{
			name: "数字验证码 (4位)",
			req: &captchago.ImageCaptchaRequest{
				Type:  captchago.CaptchaTypeNumber,
				Count: 4,
			},
		},
		{
			name: "字母验证码 (4位)",
			req: &captchago.ImageCaptchaRequest{
				Type:  captchago.CaptchaTypeLetter,
				Count: 4,
			},
		},
		{
			name: "混合验证码 (4位)",
			req: &captchago.ImageCaptchaRequest{
				Type:  captchago.CaptchaTypeMixed,
				Count: 4,
			},
		},
		{
			name: "中文验证码 (3位)",
			req: &captchago.ImageCaptchaRequest{
				Type:  captchago.CaptchaTypeChinese,
				Count: 3,
			},
		},
	}

	for _, tc := range testCases {
		image, err := client.GenerateImageCaptchaWithContext(ctx, tc.req)
		if err != nil {
			fmt.Printf("  ✗ %s: Error - %v\n", tc.name, err)
			continue
		}

		fmt.Printf("  ✓ %s: Challenge ID = %s\n", tc.name, image.ChallengeID)
	}
}
