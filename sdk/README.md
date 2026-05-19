# HJTPX SDK 生态系统

完整的验证码系统SDK集合，支持多种编程语言和框架。

## 支持的语言和框架

| SDK | 语言 | 框架集成 | 文档 |
|-----|------|---------|------|
| [Go SDK](go/README.md) | Go 1.16+ | Gin, Echo, 标准库 | [文档](go/README.md) |
| [Python SDK](python/README.md) | Python 3.7+ | Django, Flask, 异步 | [文档](python/README.md) |
| [Node.js SDK](nodejs/README.md) | Node.js 16+, TypeScript | Express, Koa | [文档](nodejs/README.md) |
| [PHP SDK](php/README.md) | PHP 7.4+ | Laravel, Symfony | [文档](php/README.md) |
| [Java SDK](java/README.md) | Java 11+ | Spring Boot | [文档](java/README.md) |
| [C# SDK](csharp/README.md) | C# 8.0+ | ASP.NET Core | [文档](csharp/README.md) |

## 功能特性

所有SDK都提供以下功能：

- **多种验证码类型**：滑块、点击、图形、旋转、手势、拼图、语音、连连看、3D验证码
- **连接池管理**：高效的HTTP连接复用
- **自动重试机制**：指数退避算法
- **完善的错误处理**：针对不同错误类型提供专用异常类
- **用户认证**：注册、登录、令牌刷新
- **环境检测**：浏览器指纹、环境安全检测

## API 一致性

所有SDK遵循统一的API设计模式：

### 验证码获取

```go
// Go
captcha := client.GenerateSliderCaptcha()
```

```python
# Python
captcha = client.get_slider_captcha()
```

```typescript
// TypeScript/Node.js
const captcha = await client.getSliderCaptcha();
```

```php
// PHP
$captcha = $client->getSliderCaptcha();
```

```java
// Java
SliderCaptchaResponse captcha = client.getSliderCaptcha();
```

```csharp
// C#
var captcha = await client.GetSliderCaptchaAsync();
```

### 验证码验证

```go
// Go
result := client.VerifySliderCaptcha(challengeID, "120")
```

```python
# Python
result = client.verify_slider_captcha(session_id, x=150)
```

```typescript
// TypeScript/Node.js
const result = await client.verifySliderCaptcha(sessionId, x);
```

```php
// PHP
$result = $client->verifySliderCaptcha($sessionId, $x);
```

```java
// Java
VerifyCaptchaResponse result = client.verifySliderCaptcha(sessionId, x);
```

```csharp
// C#
var result = await client.VerifySliderCaptchaAsync(sessionId, x);
```

## 验证码类型支持矩阵

| 类型 | Go | Python | Node.js | PHP | Java | C# |
|------|:--:|:------:|:-------:|:---:|:----:|:--:|
| 滑块验证码 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 点击验证码 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 图形验证码 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 旋转验证码 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 手势验证码 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 拼图验证码 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 语音验证码 | - | - | ✓ | ✓ | ✓ | ✓ |
| 连连看验证码 | - | - | ✓ | ✓ | ✓ | ✓ |
| 3D验证码 | - | - | ✓ | ✓ | ✓ | ✓ |

## 快速开始

### Go SDK

```bash
go get github.com/hjtpx/hjtpx/sdk/go
```

```go
client := captcha.NewCaptchaClient("app-id", "app-secret", nil)
defer client.Close()

slider, _ := client.GenerateSliderCaptcha()
result, _ := client.VerifySliderCaptcha(slider.ChallengeID, "120")
```

详细文档：[go/README.md](go/README.md)

### Python SDK

```bash
pip install requests
```

```python
client = CaptchaClient(base_url="http://localhost:8080")

captcha = client.get_slider_captcha()
result = client.verify_slider_captcha(
    session_id=captcha.session_id,
    x=150
)
```

详细文档：[python/README.md](python/README.md)

### Node.js/TypeScript SDK

```bash
npm install hjtpx-sdk
# 或
yarn add hjtpx-sdk
```

```typescript
const client = new CaptchaClient({
  baseUrl: 'http://localhost:8080',
});

const captcha = await client.getSliderCaptcha();
const result = await client.verifySliderCaptcha(
  captcha.session_id,
  150
);
```

详细文档：[nodejs/README.md](nodejs/README.md)

### PHP SDK

```bash
composer require hjtpx/captcha-sdk
```

```php
$client = new CaptchaClient('http://localhost:8080');

$captcha = $client->getSliderCaptcha();
$result = $client->verifySliderCaptcha($captcha->sessionId, 150);
```

详细文档：[php/README.md](php/README.md)

### Java SDK

```xml
<dependency>
    <groupId>com.hjtpx</groupId>
    <artifactId>captcha-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

```java
try (CaptchaClient client = new CaptchaClient(baseUrl, apiKey)) {
    SliderCaptchaResponse captcha = client.getSliderCaptcha(320, 160, 5);
    VerifyCaptchaResponse result = client.verifySliderCaptcha(
        captcha.getSessionId(),
        180
    );
}
```

详细文档：[java/README.md](java/README.md)

### C# SDK

```bash
dotnet add package Hjtpx.Captcha.Sdk
```

```csharp
var client = new CaptchaClient(baseUrl, apiKey);

var captcha = await client.GetSliderCaptchaAsync();
var result = await client.VerifySliderCaptchaAsync(
    captcha.SessionId,
    150
);
```

详细文档：[csharp/README.md](csharp/README.md)

## 错误处理

所有SDK都提供了类似的错误处理机制：

```go
// Go
if sdk.IsSDKError(err) {
    code := sdk.GetSDKErrorCode(err)
    fmt.Printf("Error code: %d\n", code)
}
```

```python
# Python
try:
    result = client.verify_slider_captcha(session_id, x)
except CaptchaTimeoutError:
    print("Request timeout")
except CaptchaAPIError as e:
    print(f"API error: {e.code}")
```

```typescript
// TypeScript/Node.js
try {
  const result = await client.verifySliderCaptcha(sessionId, x);
} catch (error) {
  if (error instanceof ValidationError) {
    console.error('Validation error');
  } else if (error instanceof RateLimitError) {
    console.error('Rate limit exceeded');
  }
}
```

```php
// PHP
try {
    $result = $client->verifySliderCaptcha($sessionId, $x);
} catch (ApiException $e) {
    echo "API Error: " . $e->getMessage();
}
```

```java
// Java
try {
    VerifyCaptchaResponse result = client.verifySliderCaptcha(sessionId, x);
} catch (ApiException e) {
    System.err.println("API Error: " + e.getMessage());
}
```

```csharp
// C#
try {
    var result = await client.VerifySliderCaptchaAsync(sessionId, x);
} catch (ApiException e) {
    Console.WriteLine($"API Error: {e.Message}");
}
```

## 框架集成示例

### Go

- [Gin集成示例](go/examples/gin_integration_example.go)
- [标准库HTTP服务器示例](go/examples/standard_http_example.go)

### Python

- [Django集成示例](python/django_examples.py)
- [Flask集成示例](python/flask_examples.py)
- [异步客户端示例](python/async_captcha.py)

### Node.js/TypeScript

- [Express集成示例](nodejs/examples/express-integration.ts)
- [Koa集成示例](nodejs/examples/koa-integration.ts)

### Java

- [Spring Boot集成示例](java/examples/spring-integration/)

### PHP

- Laravel和Symfony示例请参考 [php/README.md](php/README.md)

### C#

- ASP.NET Core示例请参考 [csharp/README.md](csharp/README.md)

## 配置选项

所有SDK都支持以下配置选项：

| 选项 | 描述 | 默认值 |
|------|------|--------|
| baseUrl | API服务器地址 | http://localhost:8080 |
| timeout | 请求超时时间（毫秒） | 30000 |
| maxRetries | 最大重试次数 | 3 |
| retryDelay | 重试延迟（毫秒） | 100 |
| apiKey | API密钥 | - |
| secretKey | 签名密钥 | - |

## 注意事项

1. 所有SDK为基本可用版本，可能存在未发现的问题
2. 请根据实际API接口调整使用方式
3. 生产环境使用前请充分测试
4. 建议合理配置重试次数和超时时间，避免对服务器造成压力
5. 存储密钥时请遵循安全最佳实践

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！
