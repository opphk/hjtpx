using Hjtpx.Captcha.Client;
using Hjtpx.Captcha.Constants;
using Hjtpx.Captcha.Exceptions;
using Hjtpx.Captcha.Models;
using Hjtpx.Captcha.Pool;
using Hjtpx.Captcha.Retry;

namespace Hjtpx.Captcha.Examples;

class Program
{
    private const string BaseUrl = "http://localhost:8080";
    private const string ApiKey = "your-api-key";
    private const string SecretKey = "your-secret-key";

    static async Task Main(string[] args)
    {
        Console.WriteLine("HJTPX Captcha C# SDK Examples");
        Console.WriteLine("==============================");
        Console.WriteLine();

        await BasicUsageExample();

        await AdvancedConfigurationExample();

        await SliderCaptchaWithTrajectoryExample();

        await ClickCaptchaExample();

        await ErrorHandlingExample();

        await ConnectionPoolExample();

        await Console.Out.WriteLineAsync("\n所有示例运行完成！");
    }

    static async Task BasicUsageExample()
    {
        Console.WriteLine("1. 基础用法示例");
        Console.WriteLine("----------------");

        using var client = new CaptchaClient(BaseUrl, ApiKey, SecretKey);

        try
        {
            var sliderCaptcha = await client.GetSliderCaptchaAsync(320, 160, 8);
            Console.WriteLine($"   [滑块] SessionId: {sliderCaptcha.SessionId}");
            Console.WriteLine($"   [滑块] 图片: {sliderCaptcha.ImageUrl}");
            Console.WriteLine($"   [滑块] SecretY: {sliderCaptcha.SecretY}");
        }
        catch (ApiException ex)
        {
            Console.WriteLine($"   [错误] API异常: {ex.Message}");
        }
        catch (NetworkException ex)
        {
            Console.WriteLine($"   [错误] 网络异常: {ex.Message}");
        }

        Console.WriteLine();
    }

    static async Task AdvancedConfigurationExample()
    {
        Console.WriteLine("2. 高级配置示例");
        Console.WriteLine("----------------");

        var config = new CaptchaClientConfig(BaseUrl)
        {
            ApiKey = ApiKey,
            SecretKey = SecretKey,
            ConnectionPoolConfig = new ConnectionPoolConfig
            {
                MaxConnections = 100,
                MaxConnectionsPerRoute = 50,
                ConnectionTimeoutMs = 5000,
                SocketTimeoutMs = 30000,
                TimeToLiveMs = 60000,
                ValidateAfterInactivityMs = 2000
            },
            RetryConfig = new RetryConfig
            {
                MaxRetries = 3,
                InitialDelayMs = 100,
                MaxDelayMs = 10000,
                BackoffMultiplier = 2.0,
                RetryableStatusCodes = new List<int> { 429, 500, 502, 503, 504 }
            }
        };

        using var client = new CaptchaClient(config);

        Console.WriteLine("   配置已应用:");
        Console.WriteLine($"   - 最大连接数: {config.ConnectionPoolConfig.MaxConnections}");
        Console.WriteLine($"   - 最大重试次数: {config.RetryConfig.MaxRetries}");
        Console.WriteLine($"   - 初始延迟: {config.RetryConfig.InitialDelayMs}ms");

        Console.WriteLine();
    }

    static async Task SliderCaptchaWithTrajectoryExample()
    {
        Console.WriteLine("3. 滑块验证码（带轨迹）示例");
        Console.WriteLine("--------------------------");

        using var client = new CaptchaClient(BaseUrl, ApiKey);

        try
        {
            var sliderCaptcha = await client.GetSliderCaptchaAsync(320, 160, 5);
            Console.WriteLine($"   获取滑块验证码成功");
            Console.WriteLine($"   SessionId: {sliderCaptcha.SessionId}");

            var trajectory = GenerateHumanLikeTrajectory(sliderCaptcha.SecretY, 180);
            var result = await client.VerifySliderCaptchaAsync(
                sliderCaptcha.SessionId,
                180,
                sliderCaptcha.SecretY,
                trajectory
            );

            Console.WriteLine($"   验证结果: {(result.Success ? "成功" : "失败")}");
            Console.WriteLine($"   消息: {result.Message}");

            if (result.TrajectoryResult != null)
            {
                Console.WriteLine($"   轨迹得分: {result.TrajectoryResult.Score:F2}");
                Console.WriteLine($"   轨迹通过: {result.TrajectoryResult.Passed}");
            }
        }
        catch (CaptchaException ex)
        {
            Console.WriteLine($"   验证码异常: {ex.Message}");
        }

        Console.WriteLine();
    }

    static List<TrajectoryPoint> GenerateHumanLikeTrajectory(int targetY, int targetX)
    {
        var points = new List<TrajectoryPoint>();
        var random = new Random();
        long baseTime = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();

        int numPoints = random.Next(15, 25);
        for (int i = 0; i <= numPoints; i++)
        {
            double progress = (double)i / numPoints;

            int x = (int)(targetX * progress);

            double wobble = Math.Sin(progress * Math.PI * 4) * (3 + random.NextDouble() * 2);
            int y = targetY + (int)wobble;

            if (i == numPoints)
            {
                y = targetY;
            }

            long timestamp = baseTime + (i * (40 + random.Next(0, 20)));

            points.Add(new TrajectoryPoint(x, y, timestamp));
        }

        return points;
    }

    static async Task ClickCaptchaExample()
    {
        Console.WriteLine("4. 点击验证码示例");
        Console.WriteLine("-----------------");

        using var client = new CaptchaClient(BaseUrl, ApiKey);

        try
        {
            var clickCaptcha = await client.GetClickCaptchaAsync(
                mode: CaptchaTypes.Click,
                shuffle: false,
                points: 4
            );

            Console.WriteLine($"   获取点击验证码成功");
            Console.WriteLine($"   SessionId: {clickCaptcha.SessionId}");
            Console.WriteLine($"   模式: {clickCaptcha.Mode}");
            Console.WriteLine($"   最大点数: {clickCaptcha.MaxPoints}");

            if (clickCaptcha.Points != null && clickCaptcha.Points.Count > 0)
            {
                var clickSequence = Enumerable.Range(0, clickCaptcha.Points.Count).ToList();
                var result = await client.VerifyClickCaptchaAsync(
                    clickCaptcha.SessionId,
                    clickCaptcha.Points,
                    clickSequence
                );

                Console.WriteLine($"   验证结果: {(result.Success ? "成功" : "失败")}");
            }
        }
        catch (CaptchaException ex)
        {
            Console.WriteLine($"   验证码异常: {ex.Message}");
        }

        Console.WriteLine();
    }

    static async Task ErrorHandlingExample()
    {
        Console.WriteLine("5. 错误处理示例");
        Console.WriteLine("---------------");

        using var client = new CaptchaClient(BaseUrl, "invalid-api-key");

        try
        {
            var captcha = await client.GetSliderCaptchaAsync();
            Console.WriteLine($"   获取成功: {captcha.SessionId}");
        }
        catch (ApiException ex)
        {
            Console.WriteLine($"   [ApiException]");
            Console.WriteLine($"   - 状态码: {ex.StatusCode}");
            Console.WriteLine($"   - 消息: {ex.Message}");
            Console.WriteLine($"   - 可重试: {ex.IsRetryable}");
        }
        catch (ValidationException ex)
        {
            Console.WriteLine($"   [ValidationException]");
            Console.WriteLine($"   - 字段: {ex.FieldName}");
            Console.WriteLine($"   - 消息: {ex.Message}");
        }
        catch (NetworkException ex)
        {
            Console.WriteLine($"   [NetworkException]");
            Console.WriteLine($"   - 主机: {ex.Host}");
            Console.WriteLine($"   - 端口: {ex.Port}");
            Console.WriteLine($"   - 消息: {ex.Message}");
        }
        catch (Exception ex)
        {
            Console.WriteLine($"   [通用异常]");
            Console.WriteLine($"   - 类型: {ex.GetType().Name}");
            Console.WriteLine($"   - 消息: {ex.Message}");
        }

        Console.WriteLine();
    }

    static async Task ConnectionPoolExample()
    {
        Console.WriteLine("6. 连接池管理示例");
        Console.WriteLine("-----------------");

        using var pool = new ManagedConnectionPool(new ConnectionPoolConfig
        {
            MaxConnections = 100,
            MaxConnectionsPerRoute = 50
        });

        var tasks = new List<Task>();

        for (int i = 0; i < 10; i++)
        {
            int index = i;
            tasks.Add(Task.Run(async () =>
            {
                var client = pool.GetClient();
                try
                {
                    var httpClient = pool.GetClient();
                    await Task.Delay(100);
                    pool.ReturnClient(httpClient);
                    Console.WriteLine($"   任务 {index}: 获取连接成功");
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"   任务 {index}: 错误 - {ex.Message}");
                }
            }));
        }

        await Task.WhenAll(tasks);

        var stats = pool.GetStatistics();
        Console.WriteLine($"   连接池统计:");
        Console.WriteLine($"   - 总连接数: {stats.TotalConnections}");
        Console.WriteLine($"   - 活跃连接: {stats.ActiveConnections}");
        Console.WriteLine($"   - 可用连接: {stats.AvailableConnections}");
        Console.WriteLine($"   - 总请求数: {stats.TotalRequests}");
        Console.WriteLine($"   - 失败请求: {stats.FailedRequests}");

        Console.WriteLine();
    }

    static async Task RotationCaptchaExample()
    {
        Console.WriteLine("7. 旋转验证码示例");
        Console.WriteLine("-----------------");

        using var client = new CaptchaClient(BaseUrl, ApiKey);

        try
        {
            var rotationCaptcha = await client.GetRotationCaptchaAsync();
            Console.WriteLine($"   获取旋转验证码成功");
            Console.WriteLine($"   ChallengeId: {rotationCaptcha.ChallengeId}");
            Console.WriteLine($"   图片URL: {rotationCaptcha.ImageUrl}");

            var correctAngle = rotationCaptcha.CorrectAngle;
            var result = await client.VerifyRotationCaptchaAsync(
                rotationCaptcha.ChallengeId,
                correctAngle
            );

            Console.WriteLine($"   验证结果: {(result.Success ? "成功" : "失败")}");
        }
        catch (CaptchaException ex)
        {
            Console.WriteLine($"   验证码异常: {ex.Message}");
        }

        Console.WriteLine();
    }

    static async Task GestureCaptchaExample()
    {
        Console.WriteLine("8. 手势验证码示例");
        Console.WriteLine("----------------");

        using var client = new CaptchaClient(BaseUrl, ApiKey);

        try
        {
            var gestureCaptcha = await client.GetGestureCaptchaAsync();
            Console.WriteLine($"   获取手势验证码成功");
            Console.WriteLine($"   SessionId: {gestureCaptcha.SessionId}");

            if (gestureCaptcha.Pattern != null)
            {
                var result = await client.VerifyGestureCaptchaAsync(
                    gestureCaptcha.SessionId,
                    gestureCaptcha.Pattern
                );

                Console.WriteLine($"   验证结果: {(result.Success ? "成功" : "失败")}");
            }
        }
        catch (CaptchaException ex)
        {
            Console.WriteLine($"   验证码异常: {ex.Message}");
        }

        Console.WriteLine();
    }

    static async Task JigsawCaptchaExample()
    {
        Console.WriteLine("9. 拼图验证码示例");
        Console.WriteLine("----------------");

        using var client = new CaptchaClient(BaseUrl, ApiKey);

        try
        {
            var jigsawCaptcha = await client.GetJigsawCaptchaAsync(320, 160, 3);
            Console.WriteLine($"   获取拼图验证码成功");
            Console.WriteLine($"   SessionId: {jigsawCaptcha.SessionId}");
            Console.WriteLine($"   网格大小: {jigsawCaptcha.GridSize}");

            if (jigsawCaptcha.Pieces != null)
            {
                var result = await client.VerifyJigsawCaptchaAsync(
                    jigsawCaptcha.SessionId,
                    jigsawCaptcha.Pieces
                );

                Console.WriteLine($"   验证结果: {(result.Success ? "成功" : "失败")}");
            }
        }
        catch (CaptchaException ex)
        {
            Console.WriteLine($"   验证码异常: {ex.Message}");
        }

        Console.WriteLine();
    }

    static async Task VoiceCaptchaExample()
    {
        Console.WriteLine("10. 语音验证码示例");
        Console.WriteLine("-----------------");

        using var client = new CaptchaClient(BaseUrl, ApiKey);

        try
        {
            var voiceCaptcha = await client.GetVoiceCaptchaAsync("zh-CN");
            Console.WriteLine($"   获取语音验证码成功");
            Console.WriteLine($"   SessionId: {voiceCaptcha.SessionId}");
            Console.WriteLine($"   音频URL: {voiceCaptcha.AudioUrl}");
            Console.WriteLine($"   语言: {voiceCaptcha.Language}");

            Console.WriteLine($"   提示: 请听音频并使用 Text 值进行验证");
            var result = await client.VerifyVoiceCaptchaAsync(
                voiceCaptcha.SessionId,
                voiceCaptcha.Text
            );

            Console.WriteLine($"   验证结果: {(result.Success ? "成功" : "失败")}");
        }
        catch (CaptchaException ex)
        {
            Console.WriteLine($"   验证码异常: {ex.Message}");
        }

        Console.WriteLine();
    }

    static async Task ConnectCaptchaExample()
    {
        Console.WriteLine("11. 连连看验证码示例");
        Console.WriteLine("-------------------");

        using var client = new CaptchaClient(BaseUrl, ApiKey);

        try
        {
            var connectCaptcha = await client.GetConnectCaptchaAsync();
            Console.WriteLine($"   获取连连看验证码成功");
            Console.WriteLine($"   SessionId: {connectCaptcha.SessionId}");

            if (connectCaptcha.Nodes != null && connectCaptcha.Connections != null)
            {
                var result = await client.VerifyConnectCaptchaAsync(
                    connectCaptcha.SessionId,
                    connectCaptcha.Connections
                );

                Console.WriteLine($"   验证结果: {(result.Success ? "成功" : "失败")}");
            }
        }
        catch (CaptchaException ex)
        {
            Console.WriteLine($"   验证码异常: {ex.Message}");
        }

        Console.WriteLine();
    }

    static async Task ThreeDCaptchaExample()
    {
        Console.WriteLine("12. 3D验证码示例");
        Console.WriteLine("---------------");

        using var client = new CaptchaClient(BaseUrl, ApiKey);

        try
        {
            var threeDCaptcha = await client.GetThreeDCaptchaAsync();
            Console.WriteLine($"   获取3D验证码成功");
            Console.WriteLine($"   SessionId: {threeDCaptcha.SessionId}");
            Console.WriteLine($"   场景URL: {threeDCaptcha.SceneUrl}");

            if (threeDCaptcha.TargetPosition != null)
            {
                var result = await client.VerifyThreeDCaptchaAsync(
                    threeDCaptcha.SessionId,
                    threeDCaptcha.TargetPosition
                );

                Console.WriteLine($"   验证结果: {(result.Success ? "成功" : "失败")}");
            }
        }
        catch (CaptchaException ex)
        {
            Console.WriteLine($"   验证码异常: {ex.Message}");
        }

        Console.WriteLine();
    }

    static async Task AuthenticationExample()
    {
        Console.WriteLine("13. 用户认证示例");
        Console.WriteLine("---------------");

        using var client = new CaptchaClient(BaseUrl, ApiKey);

        try
        {
            var sliderCaptcha = await client.GetSliderCaptchaAsync();
            var captchaToken = $"token_{sliderCaptcha.SessionId}";

            var loginResult = await client.LoginAsync("username", "password", captchaToken);
            Console.WriteLine($"   登录成功");
            Console.WriteLine($"   AccessToken: {loginResult.AccessToken[..20]}...");

            if (client.AccessToken != null)
            {
                var logoutResult = await client.LogoutAsync();
                Console.WriteLine($"   登出成功");
            }
        }
        catch (AuthenticationException ex)
        {
            Console.WriteLine($"   认证异常: {ex.Message}");
        }
        catch (CaptchaException ex)
        {
            Console.WriteLine($"   验证码异常: {ex.Message}");
        }

        Console.WriteLine();
    }

    static async Task EnvironmentDetectionExample()
    {
        Console.WriteLine("14. 环境检测示例");
        Console.WriteLine("----------------");

        using var client = new CaptchaClient(BaseUrl, ApiKey);

        try
        {
            var script = await client.GetDetectionScriptAsync();
            Console.WriteLine($"   获取检测脚本成功");
            Console.WriteLine($"   脚本长度: {script.Length} 字符");

            var detectionData = new Dictionary<string, object>
            {
                ["userAgent"] = "Mozilla/5.0...",
                ["platform"] = "Win32",
                ["language"] = "zh-CN",
                ["screenResolution"] = "1920x1080",
                ["timezone"] = "Asia/Shanghai",
                ["plugins"] = new List<string> { "plugin1", "plugin2" }
            };

            var submitResult = await client.SubmitDetectionAsync(detectionData);
            Console.WriteLine($"   提交检测数据: 成功");

            var checkResult = await client.CheckEnvironmentAsync(detectionData);
            Console.WriteLine($"   检查环境: 成功");
        }
        catch (CaptchaException ex)
        {
            Console.WriteLine($"   异常: {ex.Message}");
        }

        Console.WriteLine();
    }

    static async Task AllCaptchaTypesExample()
    {
        Console.WriteLine("15. 所有验证码类型快速获取");
        Console.WriteLine("-------------------------");

        using var client = new CaptchaClient(BaseUrl, ApiKey);

        var captchaMethods = new Dictionary<string, Func<Task<string>>>
        {
            ["滑块"] = async () =>
            {
                var captcha = await client.GetSliderCaptchaAsync();
                return captcha.SessionId;
            },
            ["点击"] = async () =>
            {
                var captcha = await client.GetClickCaptchaAsync();
                return captcha.SessionId;
            },
            ["旋转"] = async () =>
            {
                var captcha = await client.GetRotationCaptchaAsync();
                return captcha.ChallengeId;
            },
            ["手势"] = async () =>
            {
                var captcha = await client.GetGestureCaptchaAsync();
                return captcha.SessionId;
            },
            ["拼图"] = async () =>
            {
                var captcha = await client.GetJigsawCaptchaAsync();
                return captcha.SessionId;
            },
            ["语音"] = async () =>
            {
                var captcha = await client.GetVoiceCaptchaAsync();
                return captcha.SessionId;
            },
            ["连连看"] = async () =>
            {
                var captcha = await client.GetConnectCaptchaAsync();
                return captcha.SessionId;
            },
            ["3D"] = async () =>
            {
                var captcha = await client.GetThreeDCaptchaAsync();
                return captcha.SessionId;
            }
        };

        foreach (var (name, method) in captchaMethods)
        {
            try
            {
                var sessionId = await method();
                Console.WriteLine($"   [{name}] SessionId: {sessionId}");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"   [{name}] 错误: {ex.Message}");
            }
        }

        Console.WriteLine();
    }
}
