# hjtpx Go SDK

A comprehensive Go SDK for hjtpx captcha and behavior verification services, providing support for multiple captcha types, authentication, environment detection, and admin management with advanced features like automatic retries and detailed error handling.

## Features

- **Multiple Captcha Types**: Support for image captcha, slider captcha, click captcha, and gesture captcha
- **Authentication**: User registration, login, token management
- **Environment Detection**: Browser fingerprinting, automation detection, proxy detection
- **Admin Management**: Dashboard stats, application management, logs, blacklists, risk rules
- **Automatic Retries**: Built-in retry mechanism with exponential backoff for failed requests
- **Comprehensive Error Handling**: Detailed error types and error code extraction utilities
- **Thread-Safe**: Safe for concurrent use
- **Configurable**: Extensive configuration options for timeouts, pool sizes, and retry behavior

## Installation

```bash
go get github.com/hjtpx/hjtpx/sdk/go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
    client := sdk.NewClient(
        sdk.WithEndpoint("http://localhost:8080/api/v1"),
        sdk.WithAPIKey("your-api-key"),
        sdk.WithTimeout(30*time.Second),
    )
    defer func() {}()

    slider, err := client.GetSliderCaptcha(&sdk.SliderCaptchaRequest{
        Width:     320,
        Height:    160,
        Tolerance: 8,
    })
    if err != nil {
        log.Fatalf("Failed to get slider captcha: %v", err)
    }

    fmt.Printf("Got slider captcha: %s\n", slider.ChallengeID)

    verifyResp, err := client.VerifySliderCaptcha(slider.ChallengeID, "185")
    if err != nil {
        log.Fatalf("Failed to verify: %v", err)
    }

    fmt.Printf("Verification success: %v\n", verifyResp.Success)
}
```

## Configuration Options

### Client Options

```go
client := sdk.NewClient(
    sdk.WithAPIKey("your-api-key"),           // API Key
    sdk.WithAPISecret("your-api-secret"),     // API Secret
    sdk.WithEndpoint("http://localhost:8080/api/v1"), // API Endpoint
    sdk.WithTimeout(30*time.Second),           // HTTP Timeout
    sdk.WithDebugMode(true),                   // Enable debug logging
    sdk.WithHTTPClient(customHTTPClient),       // Custom HTTP client
)
```

### Config Struct

```go
cfg := &sdk.Config{
    MaxIdleConns:    10,           // Maximum idle connections
    MaxOpenConns:   100,           // Maximum open connections
    ConnMaxLifetime: 30*time.Minute, // Connection max lifetime
    ConnMaxIdleTime:  5*time.Minute,  // Idle connection timeout
    HTTPTimeout:      30*time.Second, // Total HTTP timeout
    DialTimeout:      10*time.Second, // Dial timeout
    ReadTimeout:      15*time.Second, // Read timeout
    WriteTimeout:     15*time.Second, // Write timeout
    MaxRetries:       3,            // Maximum retry attempts
    RetryDelay:       100*time.Millisecond, // Base retry delay
    BaseURL:          "http://localhost:8080/api/v1",
    DebugMode:        false,
}
```

## Captcha API

### Slider Captcha

```go
slider, err := client.GetSliderCaptcha(&sdk.SliderCaptchaRequest{
    Width:     320,
    Height:    160,
    Tolerance: 8,
})
if err != nil {
    log.Fatal(err)
}

verifyResp, err := client.VerifySliderCaptcha(slider.ChallengeID, "185")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Success: %v\n", verifyResp.Success)
```

### Click Captcha

```go
click, err := client.GetClickCaptcha(&sdk.ClickCaptchaRequest{
    Width:     400,
    Height:    300,
    IconCount: 9,
    Mode:      "number",
})
if err != nil {
    log.Fatal(err)
}

clicks := []sdk.ClickData{
    {X: 100, Y: 150, Duration: 500},
    {X: 200, Y: 250, Duration: 300},
    {X: 300, Y: 100, Duration: 400},
}

verifyResp, err := client.VerifyClickCaptcha(click.ChallengeID, clicks)
```

### Image Captcha

```go
image, err := client.GenerateImageCaptcha(&sdk.ImageCaptchaRequest{
    Type:  sdk.CaptchaTypeMixed,
    Count: 4,
})
if err != nil {
    log.Fatal(err)
}

verifyResp, err := client.VerifyImageCaptcha(&sdk.VerifyImageCaptchaRequest{
    ChallengeID: image.ChallengeID,
    Answer:     "a1b2",
})
```

### Gesture Captcha

```go
gesture, err := client.GetGestureCaptcha()
if err != nil {
    log.Fatal(err)
}

verifyResp, err := client.VerifyGestureCaptcha(&sdk.VerifyGestureRequest{
    ChallengeID: gesture.ChallengeID,
    Pattern:    []int{1, 3, 5, 7, 9},
})
```

### Universal Verify

```go
req := &sdk.VerifyCaptchaRequest{
    ChallengeID: "challenge-id",
    Action:     "slide",
    X:          185,
    Y:          100,
}
resp, err := client.VerifyCaptcha(req)
```

## Authentication API

```go
auth := client.Auth()

resp, err := auth.Login(&sdk.LoginRequest{
    Username: "testuser",
    Password: "password123",
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Access token: %s\n", resp.AccessToken)

userClient := client.User()
profile, err := userClient.GetProfile()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("User: %s\n", profile.Username)
```

### Token Manager

```go
tm := sdk.NewTokenManager(client)
tm.SetTokens(resp.AccessToken, resp.RefreshToken, resp.ExpiresIn)

if tm.IsTokenExpired() {
    if err := tm.Refresh(); err != nil {
        log.Fatal(err)
    }
}
```

## Environment Detection API

```go
detect := client.Detect()

result, err := detect.Check(&sdk.EnvironmentCheckRequest{
    Fingerprint:   "user-unique-fingerprint-hash",
    CanvasHash:    "canvas-fingerprint-hash",
    WebGLVendor:   "NVIDIA Corporation",
    WebGLRenderer: "GeForce GTX 1080",
    Fonts:         []string{"Arial", "Helvetica"},
    ProxyDetected: false,
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Is bot: %v\n", result.IsBot)
fmt.Printf("Risk level: %s\n", result.RiskLevel)
fmt.Printf("Risk score: %.2f\n", result.RiskScore)
```

### Submit Detection Data

```go
submitResp, err := detect.Submit(&sdk.DetectionSubmitRequest{
    DetectionID: "detection-id",
    RiskScore:   15.5,
    Chain:       []string{"webgl", "canvas", "fonts"},
    Fingerprint: "base64-encoded-fingerprint",
})
```

## Admin API

```go
admin := client.Admin("admin-token")

stats, err := admin.GetDashboardStats()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Total requests: %d\n", stats.TotalRequests)

realtime, err := admin.GetRealtimeStats()
fmt.Printf("Current QPS: %.2f\n", realtime.CurrentQPS)
```

### Admin Methods

```go
admin.GetDashboardStats()           // Dashboard statistics
admin.GetRecentActivity(10)          // Recent activity
admin.GetSystemStatus()            // System status
admin.GetRequestTrend(...)          // Request trend
admin.GetVerificationStats(...)    // Verification statistics
admin.GetChartData("verification", "week") // Chart data
admin.GetTrendData(7)             // Trend data
admin.GetHourlyStats("2024-01-15") // Hourly stats
admin.GetRealtimeStats()          // Real-time stats
admin.GetRiskDistribution()       // Risk distribution
admin.GetTopIPs(10, "all")       // Top IPs
admin.GetApplicationStats(10)      // Application stats
admin.GetCaptchaTypeStats()       // Captcha type stats
admin.GenerateReport(...)         // Generate report
```

## Error Handling

```go
slider, err := client.GetSliderCaptcha(nil)
if err != nil {
    if sdk.IsSDKError(err) {
        code := sdk.GetSDKErrorCode(err)
        switch code {
        case 401:
            fmt.Println("Unauthorized - check credentials")
        case 429:
            fmt.Println("Rate limited - wait before retry")
        case 500:
            fmt.Println("Server error - try again later")
        default:
            fmt.Printf("SDK Error %d: %v\n", code, err)
        }
    } else {
        fmt.Printf("Network error: %v\n", err)
    }
    return
}
```

### Error Types

```go
var (
    sdk.ErrNetworkError        = errors.New("network error")
    sdk.ErrTimeout            = errors.New("request timeout")
    sdk.ErrInvalidResponse    = errors.New("invalid response")
    sdk.ErrServerError        = errors.New("server error")
    sdk.ErrInvalidParams      = errors.New("invalid parameters")
    sdk.ErrVerificationFailed = errors.New("verification failed")
    sdk.ErrRateLimited        = errors.New("rate limited")
    sdk.ErrUnauthorized       = errors.New("unauthorized")
    sdk.ErrInternalError      = errors.New("internal error")
)
```

## CaptchaClient (Simple Wrapper)

For simple use cases, use `CaptchaClient`:

```go
client := sdk.NewCaptchaClient("app-id", "app-secret", &sdk.Config{
    BaseURL:    "http://localhost:8080/api/v1",
    HTTPTimeout: 30 * time.Second,
    MaxRetries: 3,
})

slider, err := client.GenerateSliderCaptcha()
click, err := client.GenerateClickCaptcha()
image, err := client.GenerateImageCaptchaWithOptions(sdk.CaptchaTypeMixed, 4)

stats := client.GetStats()
```

## Utility Functions

### Extract Base64 Image

```go
base64Data, err := client.ExtractBase64Image(imageResp.Image)
// base64Data contains raw image bytes
```

### Query Parameters

```go
params := map[string]interface{}{
    "width": 320,
    "height": 160,
    "mode": "number",
}
queryString := sdk.ParseQueryParams(params)
```

## Testing

Run SDK tests:

```bash
cd sdk/go
go test -v ./...
```

Run benchmarks:

```bash
go test -bench=. -benchmem
```

## Examples

See `examples/` directory for complete examples:

- `examples/quickstart/` - Basic usage
- More examples coming soon

## License

MIT License
