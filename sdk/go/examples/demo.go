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
	fmt.Printf("  Nil request error: %v\n", err)

	_, err = client.VerifyImageCaptcha(&captchago.VerifyImageCaptchaRequest{})
	fmt.Printf("  Missing challenge_id error: %v\n", err)

	_, err = client.VerifyImageCaptcha(&captchago.VerifyImageCaptchaRequest{
		ChallengeID: "test-id",
	})
	fmt.Printf("  Missing answer error: %v\n", err)

	_, err = client.ExtractBase64Image("")
	fmt.Printf("  Invalid image data error: %v\n", err)
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
