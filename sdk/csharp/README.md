# HJTPX Captcha C# SDK

HJTPX 验证码系统的 C# 软件开发工具包，提供简洁、安全、可靠的验证码解决方案。

## 功能特性

### 支持的验证码类型

| 类型 | 描述 | 验证方式 |
|------|------|---------|
| 滑块验证码 | 拖动滑块完成验证 | 位置 + 轨迹 |
| 点击验证码 | 按顺序点击指定区域 | 坐标点 |
| 旋转验证码 | 将图片旋转到正确角度 | 角度 |
| 手势验证码 | 按顺序绘制指定手势 | 手势路径 |
| 拼图验证码 | 将拼图块拖动到正确位置 | 拼图块位置 |
| 语音验证码 | 听音识码验证 | 音频答案 |
| 连连看验证码 | 按顺序连接指定点 | 连接关系 |
| 3D验证码 | 在3D空间中选择目标 | 3D坐标 |

### 核心能力

- **HMAC-SHA256 签名验证**：确保请求安全性和完整性
- **连接池管理**：高效的 HTTP 连接复用，降低资源消耗
- **自动重试机制**：智能应对网络波动和临时故障
- **完整异步 API**：基于 Task 的现代化异步编程模式
- **丰富的异常体系**：精准的错误定位和处理
- **全面的日志支持**：集成 Microsoft.Extensions.Logging

## 系统要求

- .NET Standard 2.1 及以上
- .NET Core 3.1 及以上
- .NET 5/6/7/8 及以上

## 安装

### NuGet 包管理器

```bash
dotnet add package Hjtpx.Captcha.Sdk
```

### Package Reference

```xml
<PackageReference Include="Hjtpx.Captcha.Sdk" Version="1.0.0" />
```

### 通过 dotnet CLI

```bash
dotnet add package Hjtpx.Captcha.Sdk --version 1.0.0
```

## 快速开始

### 基础用法

```csharp
using Hjtpx.Captcha.Client;
using Hjtpx.Captcha.Models;

// 创建客户端
var client = new CaptchaClient("http://localhost:8080", "your-api-key");

try
{
    // 获取滑块验证码
    var sliderCaptcha = await client.GetSliderCaptchaAsync(320, 160, 8);
    Console.WriteLine($"Session ID: {sliderCaptcha.SessionId}");

    // 模拟用户滑动
    var result = await client.VerifySliderCaptchaAsync(
        sliderCaptcha.SessionId,
        sliderCaptcha.SecretY + 2,  // 允许容差
        sliderCaptcha.SecretY,
        GenerateTrajectory(sliderCaptcha.SecretY)
    );

    if (result.Success)
    {
        Console.WriteLine("验证通过！");
    }
}
catch (Exception ex)
{
    Console.WriteLine($"错误: {ex.Message}");
}
finally
{
    client.Dispose();
}

// 生成模拟轨迹
List<TrajectoryPoint> GenerateTrajectory(int targetY)
{
    var points = new List<TrajectoryPoint>();
    var random = new Random();
    long baseTime = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();

    for (int i = 0; i <= 20; i++)
    {
        int x = i * 9;  // 模拟滑动
        int y = targetY + random.Next(-3, 4);  // 轻微上下抖动
        long t = baseTime + (i * 50);  // 50ms 间隔
        points.Add(new TrajectoryPoint(x, y, t));
    }

    return points;
}
```

### 使用配置文件

```csharp
var config = new CaptchaClientConfig("http://localhost:8080")
{
    ApiKey = "your-api-key",
    SecretKey = "your-secret-key",
    ConnectionPoolConfig = new ConnectionPoolConfig
    {
        MaxConnections = 100,
        MaxConnectionsPerRoute = 50,
        ConnectionTimeoutMs = 5000,
        SocketTimeoutMs = 30000
    },
    RetryConfig = new RetryConfig
    {
        MaxRetries = 3,
        InitialDelayMs = 100,
        MaxDelayMs = 10000,
        BackoffMultiplier = 2.0
    }
};

var client = new CaptchaClient(config);
```

### 使用依赖注入

```csharp
// Program.cs
builder.Services.AddSingleton<CaptchaClient>(sp =>
{
    var config = new CaptchaClientConfig("http://localhost:8080")
    {
        ApiKey = builder.Configuration["Captcha:ApiKey"],
        SecretKey = builder.Configuration["Captcha:SecretKey"]
    };
    return new CaptchaClient(config);
});

// Controller
public class CaptchaController : ControllerBase
{
    private readonly CaptchaClient _captchaClient;

    public CaptchaController(CaptchaClient captchaClient)
    {
        _captchaClient = captchaClient;
    }
}
```

## API 参考

### CaptchaClient

验证码客户端，提供所有验证码操作的入口。

#### 构造函数

```csharp
// 使用基础 URL
public CaptchaClient(string baseUrl);

// 使用 URL 和 API Key
public CaptchaClient(string baseUrl, string apiKey);

// 使用 URL、API Key 和 Secret Key
public CaptchaClient(string baseUrl, string apiKey, string secretKey);

// 使用配置对象
public CaptchaClient(CaptchaClientConfig config);

// 使用配置对象和日志器
public CaptchaClient(CaptchaClientConfig config, ILogger<CaptchaClient> logger);
```

#### 属性

| 属性 | 类型 | 说明 |
|------|------|------|
| AccessToken | string? | 访问令牌（登录后自动设置） |
| Config | CaptchaClientConfig | 客户端配置 |

### 验证码获取方法

#### GetSliderCaptchaAsync

获取滑块验证码。

```csharp
Task<SliderCaptchaResponse> GetSliderCaptchaAsync(
    int? width = null,
    int? height = null,
    int? tolerance = null,
    CancellationToken cancellationToken = default
);
```

**参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| width | int? | 图片宽度，默认 320 |
| height | int? | 图片高度，默认 160 |
| tolerance | int? | 容差值，默认 5 |
| cancellationToken | CancellationToken | 取消令牌 |

**返回：** `SliderCaptchaResponse`

```csharp
public class SliderCaptchaResponse
{
    public string SessionId { get; set; }      // 会话 ID
    public string ImageUrl { get; set; }        // 背景图 URL
    public string PuzzleUrl { get; set; }       // 拼图块 URL
    public string HintUrl { get; set; }         // 提示图 URL
    public int Shape { get; set; }              // 拼图形状
    public int SecretY { get; set; }            // 正确 Y 坐标
    public int ImageWidth { get; set; }          // 图片宽度
    public int ImageHeight { get; set; }        // 图片高度
    public int Tolerance { get; set; }          // 容差值
}
```

#### GetClickCaptchaAsync

获取点击验证码。

```csharp
Task<ClickCaptchaResponse> GetClickCaptchaAsync(
    string? mode = null,
    bool? shuffle = null,
    int? points = null,
    CancellationToken cancellationToken = default
);
```

**参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| mode | string? | 模式："order" 或 "multi" |
| shuffle | bool? | 是否打乱顺序 |
| points | int? | 点击点数量 |

**返回：** `ClickCaptchaResponse`

```csharp
public class ClickCaptchaResponse
{
    public string SessionId { get; set; }
    public string ImageUrl { get; set; }
    public string Hint { get; set; }
    public List<int>? HintOrder { get; set; }
    public int MaxPoints { get; set; }
    public string Mode { get; set; }
    public bool AllowShuffle { get; set; }
    public List<List<int>>? Points { get; set; }
}
```

#### GetRotationCaptchaAsync

获取旋转验证码。

```csharp
Task<RotationCaptchaResponse> GetRotationCaptchaAsync(
    CancellationToken cancellationToken = default
);
```

#### GetGestureCaptchaAsync

获取手势验证码。

```csharp
Task<GestureCaptchaResponse> GetGestureCaptchaAsync(
    CancellationToken cancellationToken = default
);
```

#### GetJigsawCaptchaAsync

获取拼图验证码。

```csharp
Task<JigsawCaptchaResponse> GetJigsawCaptchaAsync(
    int? width = null,
    int? height = null,
    int? gridSize = null,
    CancellationToken cancellationToken = default
);
```

#### GetVoiceCaptchaAsync

获取语音验证码。

```csharp
Task<VoiceCaptchaResponse> GetVoiceCaptchaAsync(
    string? language = null,
    CancellationToken cancellationToken = default
);
```

#### GetConnectCaptchaAsync

获取连连看验证码。

```csharp
Task<ConnectCaptchaResponse> GetConnectCaptchaAsync(
    CancellationToken cancellationToken = default
);
```

#### GetThreeDCaptchaAsync

获取 3D 验证码。

```csharp
Task<ThreeDCaptchaResponse> GetThreeDCaptchaAsync(
    CancellationToken cancellationToken = default
);
```

### 验证码验证方法

#### VerifySliderCaptchaAsync

验证滑块验证码。

```csharp
Task<VerifyCaptchaResponse> VerifySliderCaptchaAsync(
    string sessionId,
    int x,
    int? y = null,
    List<TrajectoryPoint>? trajectory = null,
    CancellationToken cancellationToken = default
);
```

#### VerifyClickCaptchaAsync

验证点击验证码。

```csharp
Task<VerifyCaptchaResponse> VerifyClickCaptchaAsync(
    string sessionId,
    List<List<int>> points,
    List<int>? clickSequence = null,
    CancellationToken cancellationToken = default
);
```

#### VerifyRotationCaptchaAsync

验证旋转验证码。

```csharp
Task<VerifyCaptchaResponse> VerifyRotationCaptchaAsync(
    string challengeId,
    int angle,
    CancellationToken cancellationToken = default
);
```

#### VerifyGestureCaptchaAsync

验证手势验证码。

```csharp
Task<VerifyCaptchaResponse> VerifyGestureCaptchaAsync(
    string sessionId,
    List<int> pattern,
    CancellationToken cancellationToken = default
);
```

#### VerifyJigsawCaptchaAsync

验证拼图验证码。

```csharp
Task<VerifyCaptchaResponse> VerifyJigsawCaptchaAsync(
    string sessionId,
    List<JigsawPiece> pieces,
    CancellationToken cancellationToken = default
);
```

#### VerifyVoiceCaptchaAsync

验证语音验证码。

```csharp
Task<VerifyCaptchaResponse> VerifyVoiceCaptchaAsync(
    string sessionId,
    string answer,
    CancellationToken cancellationToken = default
);
```

#### VerifyConnectCaptchaAsync

验证连连看验证码。

```csharp
Task<VerifyCaptchaResponse> VerifyConnectCaptchaAsync(
    string sessionId,
    List<List<int>> connections,
    CancellationToken cancellationToken = default
);
```

#### VerifyThreeDCaptchaAsync

验证 3D 验证码。

```csharp
Task<VerifyCaptchaResponse> VerifyThreeDCaptchaAsync(
    string sessionId,
    List<double> targetPosition,
    CancellationToken cancellationToken = default
);
```

#### 通用验证方法

```csharp
Task<VerifyCaptchaResponse> VerifyCaptchaAsync(
    VerifyCaptchaRequest request,
    CancellationToken cancellationToken = default
);
```

### 认证方法

#### LoginAsync

用户登录。

```csharp
Task<LoginResponse> LoginAsync(
    string username,
    string password,
    string? captchaToken = null,
    CancellationToken cancellationToken = default
);
```

#### LogoutAsync

用户登出。

```csharp
Task LogoutAsync(CancellationToken cancellationToken = default);
```

### 环境检测方法

#### GetDetectionScriptAsync

获取环境检测脚本。

```csharp
Task<string> GetDetectionScriptAsync(
    string? callback = null,
    CancellationToken cancellationToken = default
);
```

#### SubmitDetectionAsync

提交环境检测数据。

```csharp
Task<Dictionary<string, object>> SubmitDetectionAsync(
    Dictionary<string, object> data,
    CancellationToken cancellationToken = default
);
```

#### CheckEnvironmentAsync

检查环境状态。

```csharp
Task<Dictionary<string, object>> CheckEnvironmentAsync(
    Dictionary<string, object> data,
    CancellationToken cancellationToken = default
);
```

## 数据模型

### TrajectoryPoint

轨迹点，用于滑块验证。

```csharp
public class TrajectoryPoint
{
    public int X { get; set; }          // X 坐标
    public int Y { get; set; }          // Y 坐标
    public long Timestamp { get; set; }  // 时间戳（毫秒）

    public TrajectoryPoint() { }

    public TrajectoryPoint(int x, int y, long timestamp)
    {
        X = x;
        Y = y;
        Timestamp = timestamp;
    }
}
```

### JigsawPiece

拼图块，用于拼图验证。

```csharp
public class JigsawPiece
{
    [JsonPropertyName("x")]
    public int X { get; set; }

    [JsonPropertyName("y")]
    public int Y { get; set; }

    [JsonPropertyName("width")]
    public int Width { get; set; }

    [JsonPropertyName("height")]
    public int Height { get; set; }

    [JsonPropertyName("rotation")]
    public int Rotation { get; set; }
}
```

### VerifyCaptchaResponse

验证结果。

```csharp
public class VerifyCaptchaResponse
{
    public bool Success { get; set; }
    public string Message { get; set; }
    public int? RemainingAttempts { get; set; }
    public double? RiskScore { get; set; }
    public bool? CaptchaPass { get; set; }
    public string? FailReason { get; set; }
    public TrajectoryResult? TrajectoryResult { get; set; }

    public class TrajectoryResult
    {
        public double Score { get; set; }
        public bool Passed { get; set; }
        public List<string>? Reasons { get; set; }
    }
}
```

## 配置选项

### CaptchaClientConfig

| 属性 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| BaseUrl | string | 是 | - | API 基础地址 |
| ApiKey | string? | 否 | null | API 密钥 |
| SecretKey | string? | 否 | null | 签名密钥 |
| ConnectionPoolConfig | ConnectionPoolConfig | 否 | 新建实例 | 连接池配置 |
| RetryConfig | RetryConfig | 否 | 新建实例 | 重试配置 |

### ConnectionPoolConfig

| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| MaxConnections | int | 100 | 最大总连接数 |
| MaxConnectionsPerRoute | int | 50 | 每个路由最大连接数 |
| ConnectionTimeoutMs | int | 5000 | 连接超时（毫秒） |
| SocketTimeoutMs | int | 30000 | Socket 超时（毫秒） |
| TimeToLiveMs | int | 60000 | 连接生存时间（毫秒） |
| ValidateAfterInactivityMs | int | 2000 | 空闲后验证间隔（毫秒） |

### RetryConfig

| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| MaxRetries | int | 3 | 最大重试次数 |
| InitialDelayMs | long | 100 | 初始延迟（毫秒） |
| MaxDelayMs | long | 10000 | 最大延迟（毫秒） |
| BackoffMultiplier | double | 2.0 | 退避乘数 |
| RetryableExceptions | List<Type> | 空列表 | 可重试的异常类型 |
| RetryableStatusCodes | List<int> | [429,500,502,503,504] | 可重试的状态码 |

## 异常处理

SDK 提供层次化的异常体系，便于精确捕获和处理错误。

### 异常层次

```
CaptchaException (基类)
├── ApiException          // API 错误
├── AuthenticationException  // 认证错误
├── NetworkException      // 网络错误
└── ValidationException   // 参数验证错误
```

### 使用示例

```csharp
try
{
    var captcha = await client.GetSliderCaptchaAsync();
    var result = await client.VerifySliderCaptchaAsync(captcha.SessionId, 100);
}
catch (ApiException ex)
{
    // API 返回错误
    Console.WriteLine($"API 错误 [{ex.StatusCode}]: {ex.Message}");
    Console.WriteLine($"可重试: {ex.IsRetryable}");

    if (ex.StatusCode == 401)
    {
        // 处理认证失败
        await HandleAuthenticationFailure();
    }
}
catch (NetworkException ex)
{
    // 网络错误
    Console.WriteLine($"网络错误: {ex.Message}");
    Console.WriteLine($"主机: {ex.Host}:{ex.Port}");

    if (ex.Message.Contains("timeout"))
    {
        // 处理超时
        await RetryWithBackoff();
    }
}
catch (ValidationException ex)
{
    // 参数验证错误
    Console.WriteLine($"验证错误: {ex.Message}");
    Console.WriteLine($"字段: {ex.FieldName}");
}
catch (CaptchaException ex)
{
    // 其他验证码错误
    Console.WriteLine($"验证码错误: {ex.Message}");
}
```

### 静态工厂方法

各异常类提供了静态工厂方法，便于快速创建常见错误：

```csharp
// ValidationException
ValidationException.EmptySessionId();
ValidationException.Required("fieldName");
ValidationException.InvalidFormat("fieldName", "email");
ValidationException.OutOfRange("age", 150, 0, 120);

// AuthenticationException
AuthenticationException.InvalidCredentials();
AuthenticationException.TokenExpired();
AuthenticationException.TokenMissing();

// NetworkException
NetworkException.ConnectionTimeout("api.example.com", 5000);
NetworkException.ConnectionRefused("api.example.com", 8080);
```

## 高级用法

### 使用连接池管理器

```csharp
using var pool = new ManagedConnectionPool(new ConnectionPoolConfig
{
    MaxConnections = 200
});

var client = pool.GetClient();
// 使用 client...
pool.ReturnClient(client);

var stats = pool.GetStatistics();
Console.WriteLine($"活跃连接: {stats.ActiveConnections}");
```

### 自定义重试策略

```csharp
var config = new RetryConfig
{
    MaxRetries = 5,
    InitialDelayMs = 200,
    MaxDelayMs = 30000,
    BackoffMultiplier = 1.5,
    RetryableStatusCodes = new List<int> { 429, 500, 502, 503, 504 },
    RetryableExceptions = new List<Type> { typeof(HttpRequestException) }
};
```

### HMAC 签名

```csharp
using Hjtpx.Captcha.Signer;

var signer = new HmacSigner("your-secret-key");
string dataToSign = $"{timestamp}:/api/v1/captcha/slider";
string signature = signer.Sign(dataToSign);
```

## 示例项目

### 基础示例

```csharp
// examples/Program.cs
using Hjtpx.Captcha.Client;
using Hjtpx.Captcha.Models;

var client = new CaptchaClient("http://localhost:8080", "your-api-key");

// 滑块验证码
var slider = await client.GetSliderCaptchaAsync();
Console.WriteLine($"滑块: {slider.SessionId}");

var verifyResult = await client.VerifySliderCaptchaAsync(
    slider.SessionId,
    slider.SecretY,
    slider.SecretY,
    GenerateHumanLikeTrajectory(slider.SecretY)
);

Console.WriteLine($"验证结果: {verifyResult.Success}");

// 点击验证码
var click = await client.GetClickCaptchaAsync(mode: "order");
Console.WriteLine($"点击: {click.SessionId}");

var clickResult = await client.VerifyClickCaptchaAsync(
    click.SessionId,
    click.Points!,
    new List<int> { 0, 1, 2, 3 }
);

Console.WriteLine($"点击验证: {clickResult.Success}");
```

### Web API 集成

```csharp
[ApiController]
[Route("api/[controller]")]
public class CaptchaController : ControllerBase
{
    private readonly CaptchaClient _captchaClient;

    public CaptchaController(CaptchaClient captchaClient)
    {
        _captchaClient = captchaClient;
    }

    [HttpGet("slider")]
    public async Task<IActionResult> GetSliderCaptcha()
    {
        var captcha = await _captchaClient.GetSliderCaptchaAsync(320, 160, 5);
        return Ok(captcha);
    }

    [HttpPost("verify")]
    public async Task<IActionResult> VerifyCaptcha([FromBody] VerifyCaptchaRequest request)
    {
        try
        {
            var result = await _captchaClient.VerifyCaptchaAsync(request);
            return Ok(result);
        }
        catch (ApiException ex)
        {
            return StatusCode(500, new { error = ex.Message });
        }
    }
}
```

## 项目结构

```
csharp/
├── src/
│   └── Hjtpx.Captcha.Sdk/
│       ├── Client/              # 客户端核心
│       │   ├── CaptchaClient.cs
│       │   └── CaptchaClientConfig.cs
│       ├── Constants/           # 常量和枚举
│       │   └── CaptchaTypes.cs
│       ├── Exceptions/          # 异常定义
│       │   ├── CaptchaException.cs
│       │   ├── ApiException.cs
│       │   ├── AuthenticationException.cs
│       │   ├── NetworkException.cs
│       │   └── ValidationException.cs
│       ├── Models/              # 数据模型
│       │   ├── ApiResponse.cs
│       │   ├── SliderCaptchaResponse.cs
│       │   ├── ClickCaptchaResponse.cs
│       │   └── ...
│       ├── Pool/                # 连接池管理
│       │   ├── ConnectionPoolConfig.cs
│       │   └── ConnectionPoolManager.cs
│       ├── Retry/               # 重试机制
│       │   ├── RetryConfig.cs
│       │   └── RetryManager.cs
│       └── Signer/              # 签名实现
│           └── HmacSigner.cs
├── tests/
│   └── Hjtpx.Captcha.Sdk.Tests/
│       └── CaptchaClientTests.cs
├── examples/
│   ├── Hjtpx.Captcha.Examples.csproj
│   └── Program.cs
├── Hjtpx.Captcha.sln
├── Hjtpx.Captcha.Sdk.nuspec
└── README.md
```

## 构建和发布

### 本地构建

```bash
# 还原依赖
dotnet restore

# 构建
dotnet build

# 运行测试
dotnet test

# 发布 NuGet 包
dotnet pack
```

### CI/CD 发布

```yaml
# azure-pipelines.yml
- task: DotNetCoreCLI@2
  displayName: 'Pack NuGet package'
  inputs:
    command: 'pack'
    packagesToPack: 'src/Hjtpx.Captcha.Sdk/Hjtpx.Captcha.Sdk.csproj'
    versioningScheme: 'byPrereleaseTag'
    packDestination: '$(Build.ArtifactStagingDirectory)'

- task: NuGetPush@2
  displayName: 'Push to NuGet'
  inputs:
    packageLocation: '$(Build.ArtifactStagingDirectory)/*.nupkg'
    apiKey: $(NUGET_API_KEY)
```

## 许可证

MIT License - 详见 [LICENSE](../../LICENSE) 文件

## 贡献

欢迎提交 Issue 和 Pull Request！

## 注意事项

1. 请妥善保管 API Key 和 Secret Key，切勿泄露
2. 生产环境建议配置合理的超时和重试策略
3. 建议使用依赖注入管理客户端生命周期
4. 验证时建议同时发送轨迹数据以提高安全性
