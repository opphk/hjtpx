package examples

import (
	"fmt"
	"log"
	"time"

	"github.com/hjtpx/hjtpx/sdk/go"
)

func ExampleQuickStart() {
	client := sdk.NewClient(
		sdk.WithEndpoint("http://localhost:8080/api/v1"),
		sdk.WithAPIKey("your-api-key"),
		sdk.WithTimeout(30*time.Second),
	)
	_ = client
}

func ExampleSliderCaptcha() {
	client := sdk.NewClient(sdk.WithEndpoint("http://localhost:8080/api/v1"))

	slider, err := client.GetSliderCaptcha(&sdk.SliderCaptchaRequest{
		Width:     320,
		Height:    160,
		Tolerance: 8,
	})
	if err != nil {
		log.Fatalf("Failed to get slider captcha: %v", err)
	}

	fmt.Printf("Got slider captcha: %s\n", slider.ChallengeID)
	fmt.Printf("Background image size: %dx%d\n", slider.SliderWidth, slider.SliderHeight)

	verifyResp, err := client.VerifySliderCaptcha(slider.ChallengeID, "185")
	if err != nil {
		log.Fatalf("Failed to verify: %v", err)
	}

	fmt.Printf("Verification result: %v\n", verifyResp.Success)
}

func ExampleClickCaptcha() {
	client := sdk.NewClient(
		sdk.WithEndpoint("http://localhost:8080/api/v1"),
		sdk.WithAPIKey("your-api-key"),
		sdk.WithTimeout(30*time.Second),
	)

	click, err := client.GetClickCaptcha(&sdk.ClickCaptchaRequest{
		Width:     400,
		Height:    300,
		IconCount: 9,
		Mode:      "number",
	})
	if err != nil {
		log.Fatalf("Failed to get click captcha: %v", err)
	}

	fmt.Printf("Got click captcha: %s\n", click.ChallengeID)
	fmt.Printf("Hint: %s\n", click.Hint)
	fmt.Printf("Max points: %d\n", click.MaxPoints)

	clicks := []sdk.ClickData{
		{X: 100, Y: 150, Duration: 500},
		{X: 200, Y: 250, Duration: 300},
		{X: 300, Y: 100, Duration: 400},
	}

	verifyResp, err := client.VerifyClickCaptcha(click.ChallengeID, clicks)
	if err != nil {
		log.Fatalf("Failed to verify: %v", err)
	}

	fmt.Printf("Verification result: %v\n", verifyResp.Success)
}

func ExampleImageCaptcha() {
	client := sdk.NewClient(
		sdk.WithEndpoint("http://localhost:8080/api/v1"),
		sdk.WithAPIKey("your-api-key"),
	)

	image, err := client.GenerateImageCaptcha(&sdk.ImageCaptchaRequest{
		Type:  sdk.CaptchaTypeMixed,
		Count: 4,
	})
	if err != nil {
		log.Fatalf("Failed to generate image captcha: %v", err)
	}

	fmt.Printf("Got image captcha: %s\n", image.ChallengeID)

	base64Data, err := client.ExtractBase64Image(image.Image)
	if err != nil {
		log.Fatalf("Failed to extract image: %v", err)
	}
	fmt.Printf("Image size: %d bytes\n", len(base64Data))

	verifyResp, err := client.VerifyImageCaptcha(&sdk.VerifyImageCaptchaRequest{
		ChallengeID: image.ChallengeID,
		Answer:     "test",
	})
	if err != nil {
		log.Fatalf("Failed to verify: %v", err)
	}

	fmt.Printf("Verification result: %v\n", verifyResp.Success)
}

func ExampleAuth() {
	client := sdk.NewClient(sdk.WithEndpoint("http://localhost:8080/api/v1"))
	auth := client.Auth()

	resp, err := auth.Login(&sdk.LoginRequest{
		Username: "testuser",
		Password: "password123",
	})
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	fmt.Printf("Login success: %s\n", resp.User.Username)
	fmt.Printf("Access token: %s...\n", resp.AccessToken[:20])

	userClient := client.User()
	profile, err := userClient.GetProfile()
	if err != nil {
		log.Fatalf("Failed to get profile: %v", err)
	}
	fmt.Printf("User profile: %s <%s>\n", profile.Username, profile.Email)
}

func ExampleAdmin() {
	client := sdk.NewClient(sdk.WithEndpoint("http://localhost:8080/api/v1"))
	admin := client.Admin("admin-token")

	stats, err := admin.GetDashboardStats()
	if err != nil {
		log.Fatalf("Failed to get stats: %v", err)
	}

	fmt.Printf("Total requests: %d\n", stats.TotalRequests)
	fmt.Printf("Total users: %d\n", stats.TotalUsers)
	fmt.Printf("Total errors: %d\n", stats.TotalErrors)

	realtime, err := admin.GetRealtimeStats()
	if err != nil {
		log.Fatalf("Failed to get realtime stats: %v", err)
	}
	fmt.Printf("Current QPS: %.2f\n", realtime.CurrentQPS)
	fmt.Printf("Success rate: %.2f%%\n", realtime.SuccessRate)
}

func ExampleEnvironmentDetection() {
	client := sdk.NewClient(sdk.WithEndpoint("http://localhost:8080/api/v1"))
	detect := client.Detect()

	result, err := detect.Check(&sdk.EnvironmentCheckRequest{
		Fingerprint: "user-unique-fingerprint-hash",
		CanvasHash:  "canvas-fingerprint-hash",
		WebGLVendor: "NVIDIA Corporation",
		WebGLRenderer: "GeForce GTX 1080",
		Fonts:       []string{"Arial", "Helvetica", "Times New Roman"},
		Plugins:     []string{"Chrome PDF Plugin"},
		ProxyDetected: false,
		Timezone:    "Asia/Shanghai",
		Language:    "zh-CN",
	})
	if err != nil {
		log.Fatalf("Environment check failed: %v", err)
	}

	fmt.Printf("Is bot: %v\n", result.IsBot)
	fmt.Printf("Risk level: %s\n", result.RiskLevel)
	fmt.Printf("Risk score: %.2f\n", result.RiskScore)
	fmt.Printf("Detected flags: %v\n", result.DetectedFlags)
}

func ExampleGestureCaptcha() {
	client := sdk.NewClient(sdk.WithEndpoint("http://localhost:8080/api/v1"))

	gesture, err := client.GetGestureCaptcha()
	if err != nil {
		log.Fatalf("Failed to get gesture captcha: %v", err)
	}

	fmt.Printf("Got gesture captcha: %s\n", gesture.ChallengeID)
	fmt.Printf("Pattern: %s\n", gesture.Pattern)
	fmt.Printf("Grid size: %d\n", gesture.GridSize)

	verifyResp, err := client.VerifyGestureCaptcha(&sdk.VerifyGestureRequest{
		ChallengeID: gesture.ChallengeID,
		Pattern:    []int{1, 3, 5, 7, 9},
	})
	if err != nil {
		log.Fatalf("Failed to verify: %v", err)
	}

	fmt.Printf("Verification result: %v\n", verifyResp.Success)
}

func ExampleWithRetry() {
	client := sdk.NewClient(
		sdk.WithEndpoint("http://localhost:8080/api/v1"),
		sdk.WithDebugMode(true),
	)

	slider, err := client.GetSliderCaptcha(nil)
	if err != nil {
		if sdk.IsSDKError(err) {
			code := sdk.GetSDKErrorCode(err)
			switch code {
			case 429:
				log.Println("Rate limited, please wait")
			case 500:
				log.Println("Server error, try again later")
			default:
				log.Printf("SDK Error %d: %v\n", code, err)
			}
		} else {
			log.Printf("Network error: %v\n", err)
		}
		return
	}

	fmt.Printf("Got captcha: %s\n", slider.ChallengeID)
}

func ExampleCaptchaClient() {
	client := sdk.NewCaptchaClient("app-id", "app-secret", &sdk.Config{
		BaseURL:     "http://localhost:8080/api/v1",
		HTTPTimeout: 30 * time.Second,
		MaxRetries:  3,
		DebugMode:   true,
	})

	slider, err := client.GenerateSliderCaptcha()
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	fmt.Printf("Slider captcha: %s\n", slider.ChallengeID)

	click, err := client.GenerateClickCaptcha()
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	fmt.Printf("Click captcha: %s\n", click.ChallengeID)

	image, err := client.GenerateImageCaptchaWithOptions(sdk.CaptchaTypeMixed, 4)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	fmt.Printf("Image captcha: %s\n", image.ChallengeID)

	stats := client.GetStats()
	fmt.Printf("Total requests: %d\n", stats.TotalRequests)
}
