# Hjtpx Captcha Go SDK v15.0

## 特性

- 同步和异步验证码验证
- 批量验证支持
- 异步验证（服务端处理）
- 自动重试机制
- 速率限制
- 完整的错误处理
- 支持所有验证码类型

## 安装

```bash
go get github.com/hjtpx/hjtpx/sdk/go
```

## 快速开始

### 基本使用

```go
package main

import (
    "fmt"
    "time"
    captcha "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
    // 创建客户端
    client := captcha.NewClient("http://localhost:8080",
        captcha.WithAPIKey("your-api-key"),
        captcha.WithTimeout(30*time.Second),
    )

    // 获取滑块验证码
    slider, err := client.GetSliderCaptcha(320, 160, 8)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Session ID: %s\n", slider.SessionID)

    // 验证验证码
    result, err := client.VerifyCaptcha(&captcha.VerifyCaptchaRequest{
        SessionID: slider.SessionID,
        X:         185,
        Y:         slider.SecretY,
        Trajectory: []captcha.TrajectoryPoint{
            {X: 0, Y: slider.SecretY, T: time.Now().UnixMilli() - 1000},
            {X: 50, Y: slider.SecretY + 5, T: time.Now().UnixMilli() - 800},
            {X: 100, Y: slider.SecretY - 3, T: time.Now().UnixMilli() - 500},
            {X: 150, Y: slider.SecretY + 2, T: time.Now().UnixMilli() - 200},
            {X: 185, Y: slider.SecretY, T: time.Now().UnixMilli()},
        },
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("Success: %v, Message: %s\n", result.Success, result.Message)
}
```

### 批量验证

```go
// 创建多个验证请求
requests := []captcha.VerifyCaptchaRequest{
    {SessionID: "session-1", X: 100},
    {SessionID: "session-2", X: 150},
    {SessionID: "session-3", X: 200},
}

// 批量验证
result, err := client.BatchVerify(requests)
if err != nil {
    panic(err)
}

fmt.Printf("Success: %d, Failed: %d, Time: %dms\n",
    result.Success, result.Failed, result.TotalTime)

for _, r := range result.Results {
    fmt.Printf("Session %s: Success=%v\n", r.SessionID, r.Success)
}
```

### 异步验证

```go
// 发起异步验证
asyncReq := &captcha.AsyncVerifyRequest{
    SessionID:  "session-async-1",
    X:          150,
    CallbackURL: "https://example.com/callback",
}

asyncResp, err := client.AsyncVerify(asyncReq)
if err != nil {
    panic(err)
}
fmt.Printf("Task ID: %s, Status: %s\n", asyncResp.TaskID, asyncResp.Status)

// 等待结果
result, err := client.WaitAsyncResult(asyncResp.TaskID, 30*time.Second)
if err != nil {
    panic(err)
}
fmt.Printf("Result: Success=%v\n", result.Result.Success)
```

### 带上下文的验证

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

result, err := client.VerifyCaptchaWithContext(ctx, &captcha.VerifyCaptchaRequest{
    SessionID: "session-1",
    X:         100,
})
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        fmt.Println("Request timed out")
    }
}
```

### 配置重试和限流

```go
client := captcha.NewClient("http://localhost:8080",
    captcha.WithRetryConfig(&captcha.RetryConfig{
        MaxRetries: 5,
        BaseDelay:  100 * time.Millisecond,
        MaxDelay:   5 * time.Second,
    }),
    captcha.WithRateLimiter(100, time.Second), // 每秒100请求
)
```

## API 参考

### NewClient

创建新的验证码客户端。

```go
func NewClient(baseURL string, options ...Option) *Client
```

### WithAPIKey

设置API密钥。

```go
func WithAPIKey(apiKey string) Option
```

### WithTimeout

设置请求超时时间。

```go
func WithTimeout(timeout time.Duration) Option
```

### WithRetryConfig

配置重试策略。

```go
func WithRetryConfig(config *RetryConfig) Option
```

### WithRateLimiter

配置速率限制。

```go
func WithRateLimiter(requestsPerSecond int, window time.Duration) Option
```

### GetSliderCaptcha

获取滑块验证码。

```go
func (c *Client) GetSliderCaptcha(width, height, tolerance int) (*SliderCaptchaResponse, error)
```

### VerifyCaptcha

验证验证码。

```go
func (c *Client) VerifyCaptcha(req *VerifyCaptchaRequest) (*VerifyCaptchaResponse, error)
```

### BatchVerify

批量验证。

```go
func (c *Client) BatchVerify(requests []VerifyCaptchaRequest) (*BatchVerifyResponse, error)
```

### AsyncVerify

异步验证。

```go
func (c *Client) AsyncVerify(req *AsyncVerifyRequest) (*AsyncVerifyResponse, error)
```

### GetAsyncResult

获取异步验证结果。

```go
func (c *Client) GetAsyncResult(taskID string) (*AsyncResultResponse, error)
```

### WaitAsyncResult

等待异步验证结果。

```go
func (c *Client) WaitAsyncResult(taskID string, timeout time.Duration) (*AsyncResultResponse, error)
```

## 错误处理

```go
import "errors"

// 检查是否为SDK错误
var sdkErr *captcha.SDKError
if errors.As(err, &sdkErr) {
    fmt.Printf("SDK Error %d: %s\n", sdkErr.Code, sdkErr.Message)
}

// 判断是否可重试
if captcha.IsRetryableError(err) {
    fmt.Println("Error is retryable")
}

// 错误分类处理
captcha.HandleError(err)
```

## 许可

MIT License
