# SDK 使用指南

本文档详细介绍了各语言SDK的功能、使用方法和最佳实践。

## 目录

- [Go SDK](#go-sdk)
- [Python SDK](#python-sdk)
- [JavaScript SDK](#javascript-sdk)
- [Node.js SDK](#nodejs-sdk)
- [最佳实践](#最佳实践)

---

## Go SDK

### 安装

```bash
go get github.com/hjtpx/hjtpx/sdk/go
```

### 基础用法

```go
package main

import (
    "fmt"
    "github.com/hjtpx/hjtpx/sdk/go"
)

func main() {
    client := captcha.NewClient("http://localhost:8080")

    captcha, err := client.GetSliderCaptcha(320, 160, 8)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Session ID: %s\n", captcha.SessionID)

    verifyReq := &captcha.VerifyCaptchaRequest{
        SessionID: captcha.SessionID,
        X:         185,
        Y:         captcha.SecretY,
    }

    result, err := client.VerifyCaptcha(verifyReq)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Success: %v\n", result.Success)
}
```

### 批量请求

```go
batchClient := captcha.NewBatchClient(client, 5, 3)

requests := []captcha.BatchRequest{
    {
        SessionID: "session-001",
        Type:      "slider",
        Data: map[string]interface{}{"x": 150, "y": 50},
    },
    {
        SessionID: "session-002",
        Type:      "slider",
        Data: map[string]interface{}{"x": 180, "y": 60},
    },
}

ctx := context.Background()
result := batchClient.BatchVerify(ctx, requests)

fmt.Printf("成功: %d/%d\n", result.Successful, result.Total)
```

### API批量验证

```go
req := &captcha.BatchVerifyRequest{
    Items: []captcha.BatchVerifyItem{
        {
            Index:     0,
            SessionID: "session-001",
            Type:      "slider",
            X:         150,
            Y:         50,
        },
    },
}

resp, err := client.BatchVerifyCaptcha(req)
if err != nil {
    log.Fatal(err)
}

for _, item := range resp.Results {
    fmt.Printf("索引 %d: %s\n", item.Index, item.Message)
}
```

### 错误处理

```go
verifyReq := &captcha.VerifyCaptchaRequest{
    SessionID: "invalid-session",
    X:         100,
}

result, err := client.VerifyCaptcha(verifyReq)
if err != nil {
    if captcha.IsSDKError(err) {
        code := captcha.GetSDKErrorCode(err)
        switch code {
        case captcha.StatusUnauthorized:
            fmt.Println("API密钥无效")
        case captcha.StatusRateLimited:
            fmt.Println("请求频率超限")
        case captcha.StatusInternalError:
            fmt.Println("服务器错误")
        }

        if captcha.IsRetryableError(err) {
            delay := captcha.RetryStrategy(1, 100*time.Millisecond)
            fmt.Printf("建议等待: %v\n", delay)
        }
    }
}
```

### 完整示例

参考 [examples/batch_examples.go](examples/batch_examples.go)

---

## Python SDK

### 安装

```bash
pip install aiohttp
```

### 同步客户端

```python
from captcha import CaptchaClient, TrajectoryPoint

client = CaptchaClient(
    base_url="http://localhost:8080",
    api_key="your-api-key",
    timeout=30,
    max_retries=3,
)

slider = client.get_slider_captcha(width=320, height=160, tolerance=8)
print(f"Session ID: {slider.session_id}")

result = client.verify_slider_captcha(
    session_id=slider.session_id,
    x=185,
    y=slider.secret_y,
    trajectory=[
        TrajectoryPoint(x=0, y=slider.secret_y, t=1000),
        TrajectoryPoint(x=50, y=slider.secret_y + 2, t=1200),
        TrajectoryPoint(x=185, y=slider.secret_y, t=1500),
    ]
)

print(f"Success: {result.success}")
```

### 异步客户端

```python
import asyncio
from async_captcha import AsyncCaptchaClient

async def main():
    async with AsyncCaptchaClient(
        base_url="http://localhost:8080",
        timeout=30,
        max_retries=3,
    ) as client:
        slider = await client.get_slider_captcha()
        print(f"Session ID: {slider.session_id}")

        result = await client.verify_slider_captcha(
            session_id=slider.session_id,
            x=185,
            y=slider.secret_y,
        )
        print(f"Success: {result.success}")

asyncio.run(main())
```

### 异步批量请求

```python
import asyncio
from async_captcha import AsyncCaptchaClient

async def batch_example():
    async with AsyncCaptchaClient("http://localhost:8080") as client:
        tasks = [
            client.get_slider_captcha()
            for _ in range(10)
        ]
        sliders = await asyncio.gather(*tasks)

        verify_tasks = [
            client.verify_slider_captcha(
                session_id=slider.session_id,
                x=150,
                y=slider.secret_y,
            )
            for slider in sliders
        ]

        results = await asyncio.gather(*verify_tasks, return_exceptions=True)

        success_count = sum(
            1 for r in results
            if not isinstance(r, Exception) and r.success
        )

        print(f"成功率: {success_count}/{len(results)}")

asyncio.run(batch_example())
```

### 完整示例

参考 [async_examples.py](async_examples.py)

---

## JavaScript SDK

### 浏览器端使用

```html
<script src="captcha.js"></script>
<script>
    const client = new CaptchaClient('http://localhost:8080', {
        apiKey: 'your-api-key',
        timeout: 30000,
        retryCount: 3,
    });

    async function initCaptcha() {
        try {
            const slider = await client.getSliderCaptcha({
                width: 320,
                height: 160,
                tolerance: 8
            });
            console.log('Session ID:', slider.session_id);

            const result = await client.verifySliderCaptcha({
                session_id: slider.session_id,
                x: 185,
                y: slider.secret_y,
            });
            console.log('Success:', result.success);
        } catch (error) {
            console.error('Error:', error);
        }
    }

    initCaptcha();
</script>
```

### 使用UI组件

```html
<div id="captcha-container"></div>

<script src="captcha.js"></script>
<script>
    const client = new CaptchaClient('http://localhost:8080');

    const widget = new SliderCaptchaWidget(
        document.getElementById('captcha-container'),
        client,
        {
            width: 320,
            height: 160,
            tolerance: 8,
            onSuccess: (result) => {
                console.log('Verification successful:', result);
                document.getElementById('token').value = result.token;
            },
            onFail: (message) => {
                console.error('Verification failed:', message);
            }
        }
    );
</script>
```

### 批量请求

```javascript
const { client, batchVerify } = createBatchClient('http://localhost:8080', {
    concurrency: 5,
    maxRetries: 3,
});

const requests = [
    { session_id: 'session-1', type: 'slider', x: 150 },
    { session_id: 'session-2', type: 'slider', x: 180 },
    { session_id: 'session-3', type: 'slider', x: 200 },
];

const results = await batchVerify(requests);
console.log(`成功率: ${(results.successRate * 100).toFixed(1)}%`);
```

### 错误处理

```javascript
try {
    const result = await client.verifySliderCaptcha({
        session_id: 'invalid-session',
        x: 100,
    });
} catch (error) {
    if (error instanceof CaptchaAPIError) {
        console.error(`API错误 ${error.code}: ${error.message}`);
    } else if (error instanceof CaptchaTimeoutError) {
        console.error('请求超时');
    } else if (error instanceof CaptchaNetworkError) {
        console.error('网络错误');
    }
}
```

### TypeScript类型

```typescript
import { SliderCaptchaResponse, VerifyCaptchaResponse } from './types';

const client = new CaptchaClient('http://localhost:8080');

const slider: SliderCaptchaResponse = await client.getSliderCaptcha({
    width: 320,
    height: 160,
});

const result: VerifyCaptchaResponse = await client.verifySliderCaptcha({
    session_id: slider.session_id,
    x: 185,
    y: slider.secret_y,
});
```

### 完整示例

参考 [examples/browser-examples.js](examples/browser-examples.js)

---

## Node.js SDK

Node.js SDK位于 `sdk/nodejs/` 目录，支持完整的TypeScript类型定义。

### 安装

```bash
npm install @hjtpx/captcha-sdk
```

### 基础用法

```typescript
import { CaptchaClient, SliderCaptchaResponse } from '@hjtpx/captcha-sdk';

const client = new CaptchaClient({
    baseUrl: 'http://localhost:8080',
    apiKey: 'your-api-key',
    timeout: 30000,
    retryConfig: {
        maxRetries: 3,
        initialDelayMs: 100,
    },
});

const slider: SliderCaptchaResponse = await client.getSliderCaptcha({
    width: 320,
    height: 160,
    tolerance: 8,
});

const result = await client.verifySliderCaptcha({
    session_id: slider.session_id,
    x: 185,
    y: slider.secret_y,
    trajectory: [
        { x: 0, y: slider.secret_y, t: 1000 },
        { x: 50, y: slider.secret_y + 2, t: 1200 },
        { x: 185, y: slider.secret_y, t: 1500 },
    ],
});

console.log('Success:', result.success);
```

---

## 最佳实践

### 1. 超时配置

根据实际网络环境调整超时时间：

```go
client := captcha.NewClient(
    "http://localhost:8080",
    captcha.WithTimeout(10 * time.Second),
)
```

```python
client = CaptchaClient(
    base_url="http://localhost:8080",
    timeout=10,
)
```

```javascript
const client = new CaptchaClient('http://localhost:8080', {
    timeout: 10000,
});
```

### 2. 重试机制

所有SDK都内置重试机制：

```go
batchClient := captcha.NewBatchClient(client, 5, 3)
```

```python
async with AsyncCaptchaClient(
    base_url="http://localhost:8080",
    max_retries=3,
    retry_backoff_factor=0.5,
) as client:
```

```javascript
const { client, batchManager } = createBatchClient('http://localhost:8080', {
    maxRetries: 3,
});
```

### 3. 并发控制

批量请求时控制并发数：

```go
batchClient := captcha.NewBatchClient(client, 10, 3)
```

```python
async with AsyncCaptchaClient(
    base_url="http://localhost:8080",
    max_connections=20,
) as client:
```

```javascript
const { client, batchManager } = createBatchClient('http://localhost:8080', {
    concurrency: 10,
});
```

### 4. 错误处理

根据错误类型进行不同处理：

```go
if captcha.IsRetryableError(err) {
    delay := captcha.RetryStrategy(attempt, 100*time.Millisecond)
    time.Sleep(delay)
}
```

```python
try:
    result = await client.verify_slider_captcha(...)
except AsyncCaptchaTimeoutError:
    print("请求超时，需要重试")
except AsyncCaptchaAPIError as e:
    print(f"API错误: {e.code}, {e.message}")
```

```javascript
if (isRetryableError(error)) {
    console.log('该错误可重试');
}
```

### 5. 资源管理

使用上下文管理器确保资源正确释放：

```python
async with AsyncCaptchaClient("http://localhost:8080") as client:
    slider = await client.get_slider_captcha()
```

```go
// Client不需要手动关闭，但BatchClient的结果应该及时处理
defer func() {
    result := batchClient.BatchVerify(ctx, requests)
    fmt.Printf("成功率: %v\n", float64(result.Successful)/float64(result.Total))
}()
```

### 6. 轨迹数据

提供真实的人类轨迹可以显著提高通过率：

```go
trajectory := []captcha.TrajectoryPoint{
    {X: 0, Y: secretY, T: now - 1000},
    {X: 50, Y: secretY + 2, T: now - 800},
    {X: 100, Y: secretY - 1, T: now - 500},
    {X: 150, Y: secretY + 1, T: now - 200},
    {X: 185, Y: secretY, T: now},
}
```

```python
trajectory = [
    TrajectoryPoint(x=0, y=secret_y, t=1000),
    TrajectoryPoint(x=50, y=secret_y + 2, t=1200),
    TrajectoryPoint(x=185, y=secret_y, t=1500),
]
```

### 7. 安全建议

- 永远不要在前端代码中硬编码API密钥
- 使用环境变量或安全的密钥管理服务
- 在服务器端验证验证码结果，不要仅依赖前端验证

```go
if !result.Success {
    return c.JSON(400, gin.H{"error": "验证码验证失败"})
}
```

---

## 技术支持

如遇问题，请参考：

- [Go SDK README](go/README.md)
- [Python SDK README](python/README.md)
- [JavaScript SDK README](javascript/README.md)
- [Node.js SDK README](nodejs/README.md)

---

本文档版本: 2.0.0
最后更新: 2026-05-18
