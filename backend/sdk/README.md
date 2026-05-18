# HJTPX SDK 文档

## 概述

HJTPX SDK 是用于访问 HJTPX 验证服务的客户端库，提供简洁的 API 接口用于集成验证码验证功能。

## 支持语言

- Python
- JavaScript/TypeScript
- Go
- Java

## 安装

### Python

```bash
pip install hjtpx-sdk
```

### JavaScript/TypeScript

```bash
npm install hjtpx-sdk
# 或
yarn add hjtpx-sdk
```

### Go

```bash
go get github.com/hjtpx/hjtpx-go-sdk
```

### Java

```xml
<dependency>
    <groupId>com.hjtpx</groupId>
    <artifactId>hjtpx-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

## 快速开始

### Python 示例

```python
from hjtpx import HJTPXClient

# 初始化客户端
client = HJTPXClient(
    api_key="your-api-key",
    api_secret="your-api-secret",
    base_url="https://api.hjtpx.com"
)

# 创建验证码
response = client.create_captcha(
    app_id="your-app-id",
    captcha_type="text",
    options={
        "length": 4,
        "timeout": 300
    }
)
print(response)

# 验证验证码
result = client.verify_captcha(
    app_id="your-app-id",
    challenge=response['challenge'],
    response="user-input"
)
print(result)
```

### JavaScript/TypeScript 示例

```typescript
import { HJTPXClient } from 'hjtpx-sdk';

// 初始化客户端
const client = new HJTPXClient({
    apiKey: 'your-api-key',
    apiSecret: 'your-api-secret',
    baseUrl: 'https://api.hjtpx.com'
});

// 创建验证码
const response = await client.createCaptcha({
    appId: 'your-app-id',
    captchaType: 'text',
    options: {
        length: 4,
        timeout: 300
    }
});
console.log(response);

// 验证验证码
const result = await client.verifyCaptcha({
    appId: 'your-app-id',
    challenge: response.challenge,
    response: 'user-input'
});
console.log(result);
```

### Go 示例

```go
package main

import (
    "fmt"
    "github.com/hjtpx/hjtpx-go-sdk"
)

func main() {
    // 初始化客户端
    client := hjtpx.NewClient(
        hjtpx.WithAPIKey("your-api-key"),
        hjtpx.WithAPISecret("your-api-secret"),
        hjtpx.WithBaseURL("https://api.hjtpx.com"),
    )

    // 创建验证码
    response, err := client.CreateCaptcha(&hjtpx.CreateCaptchaRequest{
        AppID:       "your-app-id",
        CaptchaType: "text",
        Options: map[string]interface{}{
            "length":  4,
            "timeout": 300,
        },
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(response)

    // 验证验证码
    result, err := client.VerifyCaptcha(&hjtpx.VerifyCaptchaRequest{
        AppID:      "your-app-id",
        Challenge:  response.Challenge,
        Response:   "user-input",
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(result)
}
```

### Java 示例

```java
import com.hjtpx.sdk.HJTPXClient;
import com.hjtpx.sdk.request.CreateCaptchaRequest;
import com.hjtpx.sdk.request.VerifyCaptchaRequest;
import com.hjtpx.sdk.response.CreateCaptchaResponse;
import com.hjtpx.sdk.response.VerifyCaptchaResponse;

public class Main {
    public static void main(String[] args) {
        // 初始化客户端
        HJTPXClient client = new HJTPXClient.Builder()
            .apiKey("your-api-key")
            .apiSecret("your-api-secret")
            .baseUrl("https://api.hjtpx.com")
            .build();

        // 创建验证码
        CreateCaptchaRequest createRequest = new CreateCaptchaRequest();
        createRequest.setAppId("your-app-id");
        createRequest.setCaptchaType("text");
        
        CreateCaptchaResponse createResponse = client.createCaptcha(createRequest);
        System.out.println(createResponse);

        // 验证验证码
        VerifyCaptchaRequest verifyRequest = new VerifyCaptchaRequest();
        verifyRequest.setAppId("your-app-id");
        verifyRequest.setChallenge(createResponse.getChallenge());
        verifyRequest.setResponse("user-input");
        
        VerifyCaptchaResponse verifyResponse = client.verifyCaptcha(verifyRequest);
        System.out.println(verifyResponse);
    }
}
```

## API 参考

### 创建验证码

**方法**: `createCaptcha` / `CreateCaptcha`

**参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| appId | string | 是 | 应用ID |
| captchaType | string | 是 | 验证码类型: text, slider, click, math, icon |
| options | object | 否 | 可选配置 |
| options.length | int | 否 | 验证码长度，默认4 |
| options.timeout | int | 否 | 过期时间（秒），默认300 |
| options.difficulty | string | 否 | 难度: easy, medium, hard |

**响应**:

```json
{
    "success": true,
    "challenge": "abc123",
    "image": "base64-encoded-image",
    "expire_at": "2024-01-15T10:35:00Z"
}
```

### 验证验证码

**方法**: `verifyCaptcha` / `VerifyCaptcha`

**参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| appId | string | 是 | 应用ID |
| challenge | string | 是 | 挑战ID |
| response | string | 是 | 用户输入的验证码 |

**响应**:

```json
{
    "success": true,
    "verified": true,
    "risk_score": 0.15,
    "message": "验证成功"
}
```

### 获取验证码状态

**方法**: `getCaptchaStatus` / `GetCaptchaStatus`

**参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| appId | string | 是 | 应用ID |
| challenge | string | 是 | 挑战ID |

**响应**:

```json
{
    "success": true,
    "status": "verified",
    "expired": false,
    "verified_at": "2024-01-15T10:32:00Z"
}
```

### 获取应用统计

**方法**: `getAppStats` / `GetAppStats`

**参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| appId | string | 是 | 应用ID |
| startDate | string | 否 | 开始日期 (YYYY-MM-DD) |
| endDate | string | 否 | 结束日期 (YYYY-MM-DD) |

**响应**:

```json
{
    "success": true,
    "total_requests": 1000,
    "success_rate": 0.95,
    "avg_response_time": 120,
    "data": [...]
}
```

## 错误处理

SDK 会抛出统一的异常类型，包含错误码和错误信息：

```python
try:
    result = client.verify_captcha(...)
except HJTPXError as e:
    print(f"错误码: {e.code}")
    print(f"错误信息: {e.message}")
    print(f"详情: {e.details}")
```

## 配置选项

| 配置项 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| apiKey | string | - | API密钥 |
| apiSecret | string | - | API密钥 |
| baseUrl | string | https://api.hjtpx.com | API基础URL |
| timeout | int | 30 | 请求超时时间（秒） |
| retryCount | int | 3 | 重试次数 |
| debug | bool | false | 是否启用调试模式 |

## 最佳实践

1. **安全存储密钥**: 不要将 API 密钥硬编码在代码中，使用环境变量或配置文件。

2. **错误重试**: 对于网络错误，SDK 会自动重试，建议设置合理的重试次数。

3. **异步调用**: 对于高并发场景，建议使用异步客户端。

4. **日志记录**: 记录关键操作的日志，便于问题排查。

## 更新日志

### v1.0.0 (2025-07-01)
- 初始版本发布
- 支持 Python、JavaScript、Go、Java
- 提供验证码创建和验证功能

## 许可证

MIT License

---

*文档版本: 1.0*  
*最后更新: 2025年7月*