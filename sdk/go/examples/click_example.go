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

	fmt.Println("=== Click Captcha Example ===")

	click, err := client.GenerateClickCaptcha()
	if err != nil {
		fmt.Printf("✗ Click generation failed: %v\n", err)
		return
	}
	fmt.Printf("✓ Click Captcha ID: %s\n", click.ChallengeID)
	fmt.Printf("  Target Index: %d\n", click.TargetIndex)
	fmt.Printf("  Icon Positions: %v\n", click.IconPositions)

	clicks := []sdk.ClickData{
		{
			X:        click.IconPositions[click.TargetIndex][0],
			Y:        click.IconPositions[click.TargetIndex][1],
			Duration: 500,
		},
	}

	verifyResult, err := client.VerifyClickCaptcha(click.ChallengeID, clicks)
	if err != nil {
		fmt.Printf("✗ Click verification failed: %v\n", err)
		return
	}

	fmt.Printf("✓ Verification success: %v\n", verifyResult.Success)
	fmt.Printf("  Score: %.2f\n", verifyResult.Score)
	fmt.Printf("  Risk Level: %s\n", verifyResult.RiskLevel)

	stats := client.GetStats()
	fmt.Printf("\n📊 Statistics:\n")
	fmt.Printf("  Total Requests: %d\n", stats.TotalRequests)
	fmt.Printf("  Success Rate: %.2f%%\n", stats.SuccessRate)
}
