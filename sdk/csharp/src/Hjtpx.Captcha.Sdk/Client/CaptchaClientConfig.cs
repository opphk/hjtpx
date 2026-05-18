using Hjtpx.Captcha.Pool;
using Hjtpx.Captcha.Retry;

namespace Hjtpx.Captcha.Client;

public class CaptchaClientConfig
{
    public string BaseUrl { get; set; } = string.Empty;
    public string? ApiKey { get; set; }
    public string? SecretKey { get; set; }
    public ConnectionPoolConfig ConnectionPoolConfig { get; set; } = new ConnectionPoolConfig();
    public RetryConfig RetryConfig { get; set; } = new RetryConfig();

    public CaptchaClientConfig()
    {
    }

    public CaptchaClientConfig(string baseUrl)
    {
        BaseUrl = baseUrl;
    }

    public CaptchaClientConfig(string baseUrl, string apiKey)
    {
        BaseUrl = baseUrl;
        ApiKey = apiKey;
    }

    public CaptchaClientConfig(string baseUrl, string apiKey, string secretKey)
    {
        BaseUrl = baseUrl;
        ApiKey = apiKey;
        SecretKey = secretKey;
    }
}
