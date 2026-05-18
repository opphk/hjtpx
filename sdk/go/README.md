# hjtpx Go SDK

A comprehensive Go SDK for hjtpx captcha services, providing support for image captcha, slider captcha, click captcha with advanced features like connection pooling, automatic retries, and detailed error handling.

## Features

- **Multiple Captcha Types**: Support for image captcha, slider captcha, and click captcha
- **Two Client Modes**: Simple client for basic use and advanced client with connection pooling for production
- **Connection Pool Management**: Configurable HTTP connection pooling for optimal performance (advanced client)
- **Automatic Retries**: Built-in retry mechanism with exponential backoff for failed requests
- **Comprehensive Error Handling**: Detailed error types and error code extraction utilities
- **Statistics & Monitoring**: Track request statistics including success rate, retry count, and error tracking
- **Thread-Safe**: Safe for concurrent use with proper mutex protection
- **Configurable**: Extensive configuration options for timeouts, pool sizes, and retry behavior

## Installation

```bash
go get github.com/hjtpx/hjtpx/sdk/go
```

## Quick Start

### Simple Client (Recommended for Basic Use)

```go
package main

import (
    "fmt"
    "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
    client := sdk.NewClient(
        sdk.WithAPIKey("your-api-key"),
        sdk.WithAPISecret("your-api-secret"),
        sdk.WithEndpoint("http://localhost:8080"),
        sdk.WithTimeout(30*time.Second),
    )
    defer client.Close()

    captcha, err := client.GenerateImageCaptcha(nil)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Captcha ID: %s\n", captcha.ChallengeID)

    verifyResult, err := client.VerifyImageCaptcha(&sdk.VerifyImageCaptchaRequest{
        ChallengeID: captcha.ChallengeID,
        Answer:     "user-input-answer",
    })
    if err != nil {
        fmt.Printf("Verification error: %v\n", err)
        return
    }

    fmt.Printf("Verification success: %v\n", verifyResult.Success)
}
```

### Advanced Client with Connection Pooling

```go
package main

import (
    "fmt"
    "time"
    "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
    cfg := &sdk.Config{
        BaseURL:        "http://localhost:8080",
        MaxRetries:     3,
        HTTPTimeout:    10 * time.Second,
        MaxIdleConns:   10,
        MaxOpenConns:   100,
    }

    client := sdk.NewCaptchaClient("your-app-id", "your-app-secret", cfg)
    defer client.Close()

    sliderResult, err := client.GenerateSliderCaptcha()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Slider Captcha ID: %s\n", sliderResult.ChallengeID)

    verifyResult, err := client.VerifySliderCaptcha(sliderResult.ChallengeID, "120")
    if err != nil {
        fmt.Printf("Verification error: %v\n", err)
        return
    }

    fmt.Printf("Verification success: %v\n", verifyResult.Success)
}
```

## Client Comparison

| Feature | Simple Client | Advanced Client |
|---------|--------------|-----------------|
| Connection Pool | No | Yes |
| Automatic Retries | No | Yes |
| Statistics Tracking | No | Yes |
| Custom HTTP Client | No | Yes |
| Best For | Low traffic, testing | Production, high traffic |

## Simple Client Options

```go
// WithAPIKey sets the API key for authentication
func WithAPIKey(apiKey string) Option

// WithAPISecret sets the API secret for authentication
func WithAPISecret(apiSecret string) Option

// WithEndpoint sets the base URL for API requests
func WithEndpoint(endpoint string) Option

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) Option

// WithDebugMode enables debug logging
func WithDebugMode(debug bool) Option

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) Option
```

## Advanced Client Configuration

The `Config` struct provides extensive customization options:

```go
type Config struct {
    // Connection pool configuration
    MaxIdleConns    int           // Maximum idle connections (default: 10)
    MaxOpenConns    int           // Maximum open connections (default: 100)
    ConnMaxLifetime time.Duration // Connection max lifetime (default: 30 minutes)
    ConnMaxIdleTime time.Duration // Idle connection timeout (default: 5 minutes)

    // Timeout configuration
    HTTPTimeout  time.Duration // Total HTTP timeout (default: 30 seconds)
    DialTimeout  time.Duration // Dial timeout (default: 10 seconds)
    ReadTimeout  time.Duration // Read timeout (default: 15 seconds)
    WriteTimeout time.Duration // Write timeout (default: 15 seconds)

    // Retry configuration
    MaxRetries int           // Maximum retry attempts (default: 3)
    RetryDelay time.Duration // Base retry delay (default: 100ms)

    // Basic configuration
    BaseURL   string // API endpoint (default: "http://localhost:8080")
    AppID     string // Application ID
    AppSecret string // Application Secret
    DebugMode bool   // Enable debug logging
}
```

## API Reference

### Simple Client Methods

#### GenerateImageCaptcha

```go
func (c *Client) GenerateImageCaptcha(req *ImageCaptchaRequest) (*ImageCaptchaResponse, error)
```

Generates a new image captcha challenge.

```go
type ImageCaptchaRequest struct {
    Type  CaptchaType // number, letter, or mixed
    Count int         // Number of characters (default: 4)
}
```

Returns:
- `ImageCaptchaResponse`: Contains challenge_id and base64-encoded image

#### GetSliderCaptcha

```go
func (c *Client) GetSliderCaptcha(req *SliderCaptchaRequest) (*SliderCaptchaResponse, error)
```

```go
type SliderCaptchaRequest struct {
    Width      int // Image width (default: 300)
    Height     int // Image height (default: 150)
    Tolerance  int // Sliding tolerance (default: 8)
}
```

#### GetClickCaptcha

```go
func (c *Client) GetClickCaptcha(req *ClickCaptchaRequest) (*ClickCaptchaResponse, error)
```

```go
type ClickCaptchaRequest struct {
    Width     int // Image width (default: 400)
    Height    int // Image height (default: 300)
    IconCount int // Number of click targets (default: 9)
}
```

#### VerifyImageCaptcha

```go
func (c *Client) VerifyImageCaptcha(req *VerifyImageCaptchaRequest) (*VerifyImageCaptchaResponse, error)
```

```go
type VerifyImageCaptchaRequest struct {
    ChallengeID string // The challenge ID
    Answer     string // User's answer
}
```

#### VerifyCaptcha

```go
func (c *Client) VerifyCaptcha(req *VerifyCaptchaRequest) (*VerifyCaptchaResponse, error)
```

```go
type VerifyCaptchaRequest struct {
    ChallengeID string
    Action     string // "click", "slide", etc.
    Data       map[string]interface{}
}
```

### Advanced Client Methods

#### Constructor

```go
func NewCaptchaClient(appID, appSecret string, cfg *Config) *CaptchaClient
```

Creates a new captcha client with the specified credentials and configuration. If `cfg` is nil, default values will be used.

#### Methods

##### Close

```go
func (c *CaptchaClient) Close() error
```

Closes the client and releases all resources. Safe to call multiple times.

##### SetPoolConfig

```go
func (c *CaptchaClient) SetPoolConfig(cfg *Config) error
```

Updates the connection pool configuration at runtime. Returns an error if the client is closed.

##### GetStats

```go
func (c *CaptchaClient) GetStats() PoolStats
```

Returns current pool statistics:

```go
type PoolStats struct {
    ActiveConnections  int
    IdleConnections   int
    TotalRequests     int64
    FailedRequests    int64
    SuccessfulRequests int64
    RetriedRequests   int64
    SuccessRate       float64
    LastError         error
    LastErrorTime     time.Time
}
```

### Captcha Generation Methods

#### GenerateSliderCaptcha

```go
func (c *CaptchaClient) GenerateSliderCaptcha() (*SliderCaptchaResponse, error)
```

Generates a new slider captcha challenge.

Returns:
- `SliderCaptchaResponse`: Contains challenge_id, background_image, slider_image, slider_width, slider_height

#### GenerateClickCaptcha

```go
func (c *CaptchaClient) GenerateClickCaptcha() (*ClickCaptchaResponse, error)
```

Generates a new click captcha challenge.

Returns:
- `ClickCaptchaResponse`: Contains challenge_id, background_image, target_position, target_index, icon_positions

#### GenerateImageCaptcha

```go
func (c *CaptchaClient) GenerateImageCaptcha(req *ImageCaptchaRequest) (*ImageCaptchaResponse, error)
```

Generates a new image captcha challenge. If `req` is nil, default parameters are used.

```go
type ImageCaptchaRequest struct {
    Type      CaptchaType // number, letter, or mixed
    Count     int         // Number of characters (default: 4)
    CustomSet string      // Custom character set
    NoiseMode int         // Noise level (0-10)
    LineMode  int         // Line interference level (0-10)
}
```

Returns:
- `ImageCaptchaResponse`: Contains challenge_id and base64-encoded image

### Captcha Verification Methods

#### VerifySliderCaptcha

```go
func (c *CaptchaClient) VerifySliderCaptcha(captchaID, answer string) (*VerifyCaptchaResponse, error)
```

Verifies a slider captcha solution.

Parameters:
- `captchaID`: The challenge ID from GenerateSliderCaptcha
- `answer`: The slider offset position (string representation)

Returns:
- `VerifyCaptchaResponse`: Contains success, score, and risk_level

#### VerifyClickCaptcha

```go
func (c *CaptchaClient) VerifyClickCaptcha(captchaID string, clicks []ClickData) (*VerifyCaptchaResponse, error)
```

Verifies a click captcha solution.

Parameters:
- `captchaID`: The challenge ID from GenerateClickCaptcha
- `clicks`: Array of ClickData with X, Y coordinates and Duration

```go
type ClickData struct {
    X        int
    Y        int
    Duration int64 // milliseconds
}
```

Returns:
- `VerifyCaptchaResponse`: Contains success, score, and risk_level

#### VerifyImageCaptcha

```go
func (c *CaptchaClient) VerifyImageCaptcha(captchaID, answer string) (*VerifyImageCaptchaResponse, error)
```

Verifies an image captcha answer.

Parameters:
- `captchaID`: The challenge ID from GenerateImageCaptcha
- `answer`: The user's input answer (e.g., "a1b2c3")

Returns:
- `VerifyImageCaptchaResponse`: Contains success boolean

### Utility Methods

#### ExtractBase64Image

```go
func ExtractBase64Image(dataURI string) ([]byte, error)
```

Extracts raw image bytes from a base64 data URI.

## Error Handling

The SDK provides comprehensive error handling utilities:

### Error Types

```go
var (
    ErrNetworkError        = errors.New("network error")
    ErrTimeout            = errors.New("request timeout")
    ErrInvalidResponse    = errors.New("invalid response")
    ErrServerError        = errors.New("server error")
    ErrInvalidParams      = errors.New("invalid parameters")
    ErrVerificationFailed = errors.New("verification failed")
    ErrRateLimited        = errors.New("rate limited")
    ErrUnauthorized       = errors.New("unauthorized")
    ErrInternalError      = errors.New("internal error")
)
```

### Error Utilities

```go
// Check if an error is an SDKError
func IsSDKError(err error) bool

// Get the error code from an SDKError
func GetSDKErrorCode(err error) int
```

### Example

```go
// Using simple client
client := sdk.NewClient(sdk.WithAPIKey("app-id"), sdk.WithAPISecret("app-secret"))
defer client.Close()

result, err := client.GenerateImageCaptcha(nil)
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

// Using advanced client
client := sdk.NewCaptchaClient("app-id", "app-secret", cfg)
defer client.Close()

result, err := client.GenerateSliderCaptcha()
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

## Connection Pool Management (Advanced Client)

The advanced SDK manages HTTP connections efficiently with configurable pool settings:

### Default Configuration

```go
cfg := &sdk.Config{
    MaxIdleConns:    10,
    MaxOpenConns:   100,
    ConnMaxLifetime: 30 * time.Minute,
    ConnMaxIdleTime: 5 * time.Minute,
}
```

### Monitoring

```go
client := sdk.NewCaptchaClient("app-id", "app-secret", cfg)

stats := client.GetStats()
fmt.Printf("Total Requests: %d\n", stats.TotalRequests)
fmt.Printf("Success Rate: %.2f%%\n", stats.SuccessRate)
fmt.Printf("Retried Requests: %d\n", stats.RetriedRequests)
fmt.Printf("Active Connections: %d\n", stats.ActiveConnections)
```

### Dynamic Configuration

```go
newCfg := &sdk.Config{
    MaxIdleConns:  50,
    MaxOpenConns: 200,
    MaxRetries:   5,
}

err := client.SetPoolConfig(newCfg)
```

## Automatic Retries (Advanced Client)

The advanced SDK automatically retries failed requests with configurable behavior:

### Retry Conditions

- HTTP 5xx errors (server errors)
- HTTP 429 (rate limited) - with Retry-After header support
- Network timeouts
- Connection failures

### Configuration

```go
cfg := &sdk.Config{
    MaxRetries: 3,           // Maximum retry attempts
    RetryDelay: 100 * time.Millisecond, // Base delay
}
```

The SDK uses exponential backoff: delay = baseDelay * retryAttempt

### Disabling Retries

```go
cfg := &sdk.Config{
    MaxRetries: 0, // Disable retries
}
```

## Complete Examples

### Example 1: Simple Image Captcha Workflow

```go
package main

import (
    "fmt"
    "time"

    "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
    client := sdk.NewClient(
        sdk.WithAPIKey("your-api-key"),
        sdk.WithAPISecret("your-api-secret"),
        sdk.WithEndpoint("http://localhost:8080"),
        sdk.WithTimeout(30*time.Second),
    )
    defer client.Close()

    captcha, err := client.GenerateImageCaptcha(&sdk.ImageCaptchaRequest{
        Type:  sdk.CaptchaTypeMixed,
        Count: 4,
    })
    if err != nil {
        fmt.Printf("Generation failed: %v\n", err)
        return
    }
    fmt.Printf("Image Captcha: %s\n", captcha.ChallengeID)

    verifyResp, err := client.VerifyImageCaptcha(&sdk.VerifyImageCaptchaRequest{
        ChallengeID: captcha.ChallengeID,
        Answer:     "user-input",
    })
    if err != nil {
        fmt.Printf("Verification failed: %v\n", err)
        return
    }
    fmt.Printf("Verification: %v\n", verifyResp.Success)
}
```

### Example 2: Slider Captcha with Advanced Client

```go
package main

import (
    "fmt"
    "time"

    "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
    cfg := &sdk.Config{
        BaseURL:        "http://localhost:8080",
        MaxIdleConns:   10,
        MaxOpenConns:   100,
        HTTPTimeout:    10 * time.Second,
        MaxRetries:     3,
        RetryDelay:     100 * time.Millisecond,
        DebugMode:      true,
    }

    client := sdk.NewCaptchaClient("your-app-id", "your-app-secret", cfg)
    defer client.Close()

    slider, err := client.GenerateSliderCaptcha()
    if err != nil {
        fmt.Printf("Slider generation failed: %v\n", err)
        return
    }
    fmt.Printf("Slider Captcha: %s\n", slider.ChallengeID)

    verifyResult, err := client.VerifySliderCaptcha(slider.ChallengeID, "120")
    if err != nil {
        fmt.Printf("Slider verification failed: %v\n", err)
        return
    }
    fmt.Printf("Slider verification: %v\n", verifyResult.Success)

    stats := client.GetStats()
    fmt.Printf("Total Requests: %d\n", stats.TotalRequests)
    fmt.Printf("Success Rate: %.2f%%\n", stats.SuccessRate)
}
```

### Example 3: Click Captcha

```go
package main

import (
    "fmt"
    "time"

    "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
    client := sdk.NewCaptchaClient("your-app-id", "your-app-secret", nil)
    defer client.Close()

    click, err := client.GenerateClickCaptcha()
    if err != nil {
        fmt.Printf("Click generation failed: %v\n", err)
        return
    }
    fmt.Printf("Click Captcha: %s\n", click.ChallengeID)

    clicks := []sdk.ClickData{
        {X: click.IconPositions[click.TargetIndex][0], Y: click.IconPositions[click.TargetIndex][1], Duration: 500},
    }

    verifyResult, err := client.VerifyClickCaptcha(click.ChallengeID, clicks)
    if err != nil {
        fmt.Printf("Click verification failed: %v\n", err)
        return
    }
    fmt.Printf("Click verification: %v\n", verifyResult.Success)
}
```

## Testing

Run the SDK tests:

```bash
cd sdk/go
go test -v ./...
```

Run benchmarks:

```bash
go test -bench=. -benchmem
```

## License

MIT License
