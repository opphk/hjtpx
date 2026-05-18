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
