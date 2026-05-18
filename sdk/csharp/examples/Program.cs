using Hjtpx.Captcha.Client;
using Hjtpx.Captcha.Models;

namespace Hjtpx.Captcha.Examples;

class Program
{
    static async Task Main(string[] args)
    {
        Console.WriteLine("HJTPX Captcha C# SDK Examples");
        Console.WriteLine("============================");

        // 1. 基础用法：创建客户端
        Console.WriteLine("\n1. 创建 CaptchaClient");
        var client = new CaptchaClient("http://localhost:8080", "your-api-key", "your-secret-key");
        Console.WriteLine("   客户端创建成功");

        try
        {
            // 2. 获取滑块验证码
            Console.WriteLine("\n2. 获取滑块验证码");
            var sliderCaptcha = await client.GetSliderCaptchaAsync(320, 160, 8);
            Console.WriteLine($"   SessionId: {sliderCaptcha.SessionId}");
            Console.WriteLine($"   ImageUrl: {sliderCaptcha.ImageUrl}");

            // 3. 验证滑块验证码（假设用户滑动到 x=185 位置）
            Console.WriteLine("\n3. 验证滑块验证码");
            var verifyResult = await client.VerifySliderCaptchaAsync(
                sliderCaptcha.SessionId,
                185,
                sliderCaptcha.SecretY,
                new List<TrajectoryPoint>
                {
                    new TrajectoryPoint(0, sliderCaptcha.SecretY, DateTimeOffset.UtcNow.ToUnixTimeMilliseconds() - 1000),
                    new TrajectoryPoint(50, sliderCaptcha.SecretY + 5, DateTimeOffset.UtcNow.ToUnixTimeMilliseconds() - 800),
                    new TrajectoryPoint(100, sliderCaptcha.SecretY - 3, DateTimeOffset.UtcNow.ToUnixTimeMilliseconds() - 500),
                    new TrajectoryPoint(150, sliderCaptcha.SecretY + 2, DateTimeOffset.UtcNow.ToUnixTimeMilliseconds() - 200),
                    new TrajectoryPoint(185, sliderCaptcha.SecretY, DateTimeOffset.UtcNow.ToUnixTimeMilliseconds())
                }
            );
            Console.WriteLine($"   验证成功: {verifyResult.Success}");
            Console.WriteLine($"   消息: {verifyResult.Message}");

            // 4. 其他验证码类型示例
            Console.WriteLine("\n4. 其他验证码类型");

            // 点击验证码
            var clickCaptcha = await client.GetClickCaptchaAsync();
            Console.WriteLine($"   点击验证码 SessionId: {clickCaptcha.SessionId}");

            // 旋转验证码
            var rotationCaptcha = await client.GetRotationCaptchaAsync();
            Console.WriteLine($"   旋转验证码 ChallengeId: {rotationCaptcha.ChallengeId}");

            // 手势验证码
            var gestureCaptcha = await client.GetGestureCaptchaAsync();
            Console.WriteLine($"   手势验证码 SessionId: {gestureCaptcha.SessionId}");

            // 拼图验证码
            var jigsawCaptcha = await client.GetJigsawCaptchaAsync();
            Console.WriteLine($"   拼图验证码 SessionId: {jigsawCaptcha.SessionId}");

            // 语音验证码
            var voiceCaptcha = await client.GetVoiceCaptchaAsync();
            Console.WriteLine($"   语音验证码 SessionId: {voiceCaptcha.SessionId}");

            // 连连看验证码
            var connectCaptcha = await client.GetConnectCaptchaAsync();
            Console.WriteLine($"   连连看验证码 SessionId: {connectCaptcha.SessionId}");

            // 3D 验证码
            var threeDCaptcha = await client.GetThreeDCaptchaAsync();
            Console.WriteLine($"   3D 验证码 SessionId: {threeDCaptcha.SessionId}");

            // 5. 用户认证示例
            Console.WriteLine("\n5. 用户认证");
            Console.WriteLine("   注意：请根据实际需要使用");
            // var loginResult = await client.LoginAsync("username", "password", "captcha-token");
            // Console.WriteLine($"   登录成功，Access Token: {loginResult.AccessToken}");
            // await client.LogoutAsync();

            // 6. 环境检测示例
            Console.WriteLine("\n6. 环境检测");
            var detectionScript = await client.GetDetectionScriptAsync();
            Console.WriteLine($"   检测脚本长度: {detectionScript.Length} 字符");

            Console.WriteLine("\n示例运行完成！");
        }
        catch (Exception ex)
        {
            Console.WriteLine($"\n发生错误: {ex.Message}");
            Console.WriteLine($"堆栈跟踪: {ex.StackTrace}");
        }
        finally
        {
            client.Dispose();
        }
    }
}

public class AdvancedExamples
{
    private readonly CaptchaClient _client;

    public AdvancedExamples(CaptchaClient client)
    {
        _client = client;
    }

    public async Task RunAdvancedExamplesAsync()
    {
        await SliderCaptchaWithRetryAsync();
        await ClickCaptchaExampleAsync();
        await GestureCaptchaExampleAsync();
        await JigsawCaptchaExampleAsync();
        await UserLoginExampleAsync();
        await EnvironmentDetectionExampleAsync();
    }

    private async Task SliderCaptchaWithRetryAsync()
    {
        Console.WriteLine("\n滑块验证码示例（含重试）");

        for (int attempt = 1; attempt <= 3; attempt++)
        {
            try
            {
                var slider = await _client.GetSliderCaptchaAsync(320, 160, 8);
                Console.WriteLine($"获取成功，SessionId: {slider.SessionId}");

                var verifyResult = await _client.VerifySliderCaptchaAsync(
                    slider.SessionId,
                    185,
                    slider.SecretY,
                    GenerateTrajectory(slider.SecretY)
                );

                if (verifyResult.Success)
                {
                    Console.WriteLine("验证成功！");
                    break;
                }
                else
                {
                    Console.WriteLine($"验证失败: {verifyResult.Message}");
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"第 {attempt} 次尝试失败: {ex.Message}");
                if (attempt == 3)
                {
                    Console.WriteLine("所有尝试均失败");
                }
            }
        }
    }

    private List<TrajectoryPoint> GenerateTrajectory(int secretY)
    {
        var trajectory = new List<TrajectoryPoint>();
        long baseTime = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();

        for (int i = 0; i <= 10; i++)
        {
            int x = i * 20;
            int y = secretY + (i % 2 == 0 ? 2 : -2);
            long t = baseTime - (1000 - i * 100);
            trajectory.Add(new TrajectoryPoint(x, y, t));
        }

        return trajectory;
    }

    private async Task ClickCaptchaExampleAsync()
    {
        Console.WriteLine("\n点击验证码示例");

        var clickCaptcha = await _client.GetClickCaptchaAsync("number", shuffle: true, points: 3);
        Console.WriteLine($"获取点击验证码，SessionId: {clickCaptcha.SessionId}");

        var verifyResult = await _client.VerifyClickCaptchaAsync(
            clickCaptcha.SessionId,
            new List<List<int>>
            {
                new List<int> { 100, 100 },
                new List<int> { 200, 100 },
                new List<int> { 150, 200 }
            },
            new List<int> { 0, 1, 2 }
        );

        Console.WriteLine($"验证结果: {verifyResult.Success}");
    }

    private async Task GestureCaptchaExampleAsync()
    {
        Console.WriteLine("\n手势验证码示例");

        var gestureCaptcha = await _client.GetGestureCaptchaAsync();
        Console.WriteLine($"获取手势验证码，SessionId: {gestureCaptcha.SessionId}");
        Console.WriteLine($"网格大小: {gestureCaptcha.GridSize}");

        var verifyResult = await _client.VerifyGestureCaptchaAsync(
            gestureCaptcha.SessionId,
            new List<int> { 1, 2, 3, 4, 5, 6 }
        );

        Console.WriteLine($"验证结果: {verifyResult.Success}");
    }

    private async Task JigsawCaptchaExampleAsync()
    {
        Console.WriteLine("\n拼图验证码示例");

        var jigsawCaptcha = await _client.GetJigsawCaptchaAsync(300, 300, 3);
        Console.WriteLine($"获取拼图验证码，SessionId: {jigsawCaptcha.SessionId}");
        Console.WriteLine($"碎片数量: {jigsawCaptcha.Pieces.Count}");

        var correctPieces = jigsawCaptcha.Pieces.Select(p => new JigsawPiece
        {
            Index = p.Index,
            OriginalX = p.OriginalX,
            OriginalY = p.OriginalY,
            CurrentX = p.OriginalX,
            CurrentY = p.OriginalY,
            Width = p.Width,
            Height = p.Height,
            Rotation = 0
        }).ToList();

        var verifyResult = await _client.VerifyJigsawCaptchaAsync(
            jigsawCaptcha.SessionId,
            correctPieces
        );

        Console.WriteLine($"验证结果: {verifyResult.Success}");
    }

    private async Task UserLoginExampleAsync()
    {
        Console.WriteLine("\n用户登录示例");

        try
        {
            var loginResult = await _client.LoginAsync("testuser", "password123", null);
            Console.WriteLine($"登录成功！");
            Console.WriteLine($"Access Token: {loginResult.AccessToken}");
            Console.WriteLine($"刷新 Token: {loginResult.RefreshToken}");
            Console.WriteLine($"用户信息: {loginResult.User.Username} ({loginResult.User.Email})");

            await _client.LogoutAsync();
            Console.WriteLine("已登出");
        }
        catch (Exception ex)
        {
            Console.WriteLine($"登录失败: {ex.Message}");
        }
    }

    private async Task EnvironmentDetectionExampleAsync()
    {
        Console.WriteLine("\n环境检测示例");

        var script = await _client.GetDetectionScriptAsync("onDetectionComplete");
        Console.WriteLine($"获取检测脚本，长度: {script.Length} 字符");

        var submitResult = await _client.SubmitDetectionAsync(new Dictionary<string, object>
        {
            { "fingerprint", "abc123" },
            { "canvas_hash", "canvas456" },
            { "webgl_vendor", "WebGL Vendor" },
            { "timezone", "Asia/Shanghai" },
            { "language", "zh-CN" }
        });

        Console.WriteLine($"提交检测结果: {submitResult}");

        var checkResult = await _client.CheckEnvironmentAsync(new Dictionary<string, object>
        {
            { "fingerprint", "abc123" },
            { "risk_score", 0.1 }
        });

        Console.WriteLine($"环境检查结果: {checkResult}");
    }
}

public class CustomConfigurationExample
{
    public CaptchaClient CreateCustomClient()
    {
        var config = new CaptchaClientConfig("http://localhost:8080")
        {
            ApiKey = "your-api-key",
            SecretKey = "your-secret-key"
        };

        config.ConnectionPoolConfig.MaxConnections = 200;
        config.ConnectionPoolConfig.MaxConnectionsPerRoute = 50;
        config.ConnectionPoolConfig.ConnectionTimeoutMs = 10000;
        config.ConnectionPoolConfig.SocketTimeoutMs = 30000;

        config.RetryConfig.MaxRetries = 5;
        config.RetryConfig.InitialDelayMs = 200;
        config.RetryConfig.MaxDelayMs = 10000;
        config.RetryConfig.BackoffMultiplier = 2.0;
        config.RetryConfig.RetryableStatusCodes = new List<int> { 429, 500, 502, 503, 504 };

        return new CaptchaClient(config);
    }
}
