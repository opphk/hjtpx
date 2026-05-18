# HJTPX Captcha C# SDK

HJTPX 验证码系统的 C# 软件开发工具包，支持所有主流验证码类型。

## 功能特性

- 支持多种验证码类型：
  - 滑块验证
  - 点击验证
  - 旋转验证
  - 手势验证
  - 拼图验证
  - 语音验证
  - 连连看验证
  - 3D验证
- API 签名验证（HMAC-SHA256）
- 连接池管理
- 自动重试机制
- 完整的异步 API
- 支持 .NET 配置系统
- 完善的错误处理

## 安装

### NuGet 包

```bash
dotnet add package Hjtpx.Captcha.Sdk
```

### 或通过 .csproj 引用

```xml
<PackageReference Include="Hjtpx.Captcha.Sdk" Version="1.0.0" />
```

## 快速开始

### 基础使用

```csharp
using Hjtpx.Captcha.Client;
using Hjtpx.Captcha.Models;

// 创建客户端
var client = new CaptchaClient("http://localhost:8080", "your-api-key", "your-secret-key");

try
{
    // 获取滑块验证码
    var captcha = await client.GetSliderCaptchaAsync(320, 160, 8);
    Console.WriteLine($"Session ID: {captcha.SessionId}");

    // 验证验证码
    var result = await client.VerifySliderCaptchaAsync(
        captcha.SessionId,
        185,
        captcha.SecretY,
        new List<TrajectoryPoint>
        {
            new TrajectoryPoint(0, captcha.SecretY, DateTimeOffset.UtcNow.ToUnixTimeMilliseconds() - 1000),
            new TrajectoryPoint(185, captcha.SecretY, DateTimeOffset.UtcNow.ToUnixTimeMilliseconds())
        }
    );

    Console.WriteLine($"验证成功: {result.Success}");
}
catch (Exception ex)
{
    Console.WriteLine($"错误: {ex.Message}");
}
finally
{
    client.Dispose();
}
```

### 使用配置

```csharp
var config = new CaptchaClientConfig("http://localhost:8080")
{
    ApiKey = "your-api-key",
    SecretKey = "your-secret-key"
};

// 配置连接池
config.ConnectionPoolConfig.MaxConnections = 200;
config.ConnectionPoolConfig.MaxConnectionsPerRoute = 50;

// 配置重试
config.RetryConfig.MaxRetries = 5;
config.RetryConfig.InitialDelayMs = 200;

var client = new CaptchaClient(config);
```

## 验证码类型

### 滑块验证码

```csharp
// 获取滑块验证码
var slider = await client.GetSliderCaptchaAsync(320, 160, 8);

// 生成轨迹
var trajectory = new List<TrajectoryPoint>
{
    new TrajectoryPoint(0, slider.SecretY, DateTimeOffset.UtcNow.ToUnixTimeMilliseconds() - 1000),
    new TrajectoryPoint(50, slider.SecretY + 2, DateTimeOffset.UtcNow.ToUnixTimeMilliseconds() - 800),
    new TrajectoryPoint(100, slider.SecretY - 1, DateTimeOffset.UtcNow.ToUnixTimeMilliseconds() - 600),
    new TrajectoryPoint(150, slider.SecretY, DateTimeOffset.UtcNow.ToUnixTimeMilliseconds())
};

// 验证
var result = await client.VerifySliderCaptchaAsync(
    slider.SessionId,
    150,
    slider.SecretY,
    trajectory
);
```

### 点选验证码

```csharp
// 获取点选验证码
var click = await client.GetClickCaptchaAsync("number", true, 3);
Console.WriteLine($"提示: {click.Hint}");

// 用户点击
var clicks = new List<ClickData>
{
    new ClickData { X = 100, Y = 100, Duration = 500 },
    new ClickData { X = 200, Y = 150, Duration = 300 },
    new ClickData { X = 300, Y = 200, Duration = 400 }
};

// 验证
var result = await client.VerifyClickCaptchaAsync(
    click.SessionId,
    clicks,
    new List<int> { 0, 1, 2 }
);
```

### 手势验证码

```csharp
// 获取手势验证码
var gesture = await client.GetGestureCaptchaAsync();

// 用户绘制的手势
var pattern = new List<int> { 0, 1, 2, 4, 8 };

// 验证
var result = await client.VerifyGestureCaptchaAsync(gesture.SessionId, pattern);
```

### 旋转验证码

```csharp
// 获取旋转验证码
var rotation = await client.GetRotationCaptchaAsync();

// 用户旋转角度
var result = await client.VerifyRotationCaptchaAsync(rotation.ChallengeId, 87);
```

### 拼图验证码

```csharp
// 获取拼图验证码
var jigsaw = await client.GetJigsawCaptchaAsync(300, 300, 3);

// 用户调整的碎片
var pieces = jigsaw.Pieces.Select(p => new JigsawPiece
{
    Index = p.Index,
    OriginalX = p.OriginalX,
    OriginalY = p.OriginalY,
    CurrentX = p.OriginalX + 5,
    CurrentY = p.OriginalY + 5,
    Width = p.Width,
    Height = p.Height,
    Rotation = 0
}).ToList();

// 验证
var result = await client.VerifyJigsawCaptchaAsync(jigsaw.SessionId, pieces);
```

### 语音验证码

```csharp
// 获取语音验证码
var voice = await client.GetVoiceCaptchaAsync("zh-CN");

// 用户听到的答案
var result = await client.VerifyVoiceCaptchaAsync(voice.SessionId, voice.Text ?? "123456");
```

## 用户认证

### 登录

```csharp
var login = await client.LoginAsync("username", "password");
Console.WriteLine($"Token: {login.AccessToken}");
Console.WriteLine($"User: {login.User.Username}");
```

### 注册

```csharp
var register = await client.RegisterAsync("newuser", "new@example.com", "password");
Console.WriteLine($"User ID: {register.UserId}");
```

### 登出

```csharp
await client.LogoutAsync();
```

## 环境检测

### 获取检测脚本

```csharp
var script = await client.GetDetectionScriptAsync();
```

### 提交检测数据

```csharp
var detectionData = new Dictionary<string, object>
{
    ["fingerprint"] = "browser-fingerprint",
    ["canvas_hash"] = "canvas-fingerprint",
    ["webgl_vendor"] = "WebGL Vendor",
    ["timezone"] = "Asia/Shanghai",
    ["language"] = "zh-CN"
};

var result = await client.SubmitDetectionAsync(detectionData);
Console.WriteLine($"Risk Level: {result["risk_level"]}");
```

## 错误处理

SDK 提供以下异常类型：

```csharp
try
{
    var captcha = await client.GetSliderCaptchaAsync();
}
catch (ApiException ex)
{
    // API 返回错误
    Console.WriteLine($"API 错误: {ex.Code} - {ex.Message}");
}
catch (NetworkException ex)
{
    // 网络错误
    Console.WriteLine($"网络错误: {ex.Message}");
}
catch (CaptchaException ex)
{
    // 通用验证码错误
    Console.WriteLine($"验证码错误: {ex.Message}");
}
catch (ValidationException ex)
{
    // 参数验证错误
    Console.WriteLine($"验证错误: {ex.Message}");
}
catch (AuthenticationException ex)
{
    // 认证错误
    Console.WriteLine($"认证错误: {ex.Message}");
}
```

## 配置选项

### CaptchaClientConfig

| 属性 | 类型 | 说明 | 默认值 |
|-----|------|-----|-------|
| BaseUrl | string | API 基础地址 | 必填 |
| ApiKey | string? | API 密钥 | null |
| SecretKey | string? | 用于签名的密钥 | null |
| ConnectionPoolConfig | ConnectionPoolConfig | 连接池配置 | 见下方 |
| RetryConfig | RetryConfig | 重试配置 | 见下方 |

### ConnectionPoolConfig

| 属性 | 类型 | 说明 | 默认值 |
|-----|------|-----|-------|
| MaxConnections | int | 最大总连接数 | 100 |
| MaxConnectionsPerRoute | int | 每个路由最大连接数 | 50 |
| ConnectionTimeoutMs | int | 连接超时（毫秒） | 5000 |
| SocketTimeoutMs | int | Socket 超时（毫秒） | 30000 |
| TimeToLiveMs | int | 连接生存时间（毫秒） | 60000 |

### RetryConfig

| 属性 | 类型 | 说明 | 默认值 |
|-----|------|-----|-------|
| MaxRetries | int | 最大重试次数 | 3 |
| InitialDelayMs | long | 初始延迟（毫秒） | 100 |
| MaxDelayMs | long | 最大延迟（毫秒） | 10000 |
| BackoffMultiplier | double | 退避乘数 | 2.0 |
| RetryableStatusCodes | List<int> | 可重试的状态码 | 429,500,502,503,504 |

## 运行测试

```bash
# 运行单元测试
cd tests/Hjtpx.Captcha.Sdk.Tests
dotnet test

# 运行示例
cd examples
dotnet run
```

## 项目结构

```
csharp/
├── src/
│   └── Hjtpx.Captcha.Sdk/
│       ├── Client/          # 客户端实现
│       ├── Exceptions/      # 异常类型
│       ├── Models/          # 数据模型
│       ├── Pool/            # 连接池
│       ├── Retry/           # 重试机制
│       └── Signer/          # 签名实现
├── tests/
│   └── Hjtpx.Captcha.Sdk.Tests/  # 单元测试
├── examples/                # 示例代码
└── README.md                # 本文档
```

## 兼容性

- .NET Standard 2.1+
- .NET Core 3.1+
- .NET 5/6/7/8+

## 许可证

MIT License

## 注意事项

本 SDK 目前处于开发阶段，虽然在常见场景下可以正常使用，但可能还有未发现的问题。建议在生产环境使用前进行充分测试。
