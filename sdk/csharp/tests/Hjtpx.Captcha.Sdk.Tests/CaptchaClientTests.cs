using Xunit;
using Hjtpx.Captcha.Client;
using Hjtpx.Captcha.Exceptions;
using Hjtpx.Captcha.Models;

namespace Hjtpx.Captcha.Sdk.Tests;

public class CaptchaClientTests : IDisposable
{
    private readonly CaptchaClient _client;

    public CaptchaClientTests()
    {
        _client = new CaptchaClient("http://localhost:8080");
    }

    public void Dispose()
    {
        _client.Dispose();
    }

    [Fact]
    public void Client_Created_Successfully()
    {
        Assert.NotNull(_client);
        Assert.Equal("http://localhost:8080", _client.Config.BaseUrl);
    }

    [Fact]
    public void ClientConfig_Created_WithDefaults()
    {
        var config = new CaptchaClientConfig();
        Assert.NotNull(config.ConnectionPoolConfig);
        Assert.NotNull(config.RetryConfig);
        Assert.Equal(100, config.ConnectionPoolConfig.MaxConnections);
        Assert.Equal(3, config.RetryConfig.MaxRetries);
    }

    [Fact]
    public void RetryConfig_CalculatesDelay_Correctly()
    {
        var config = new Retry.RetryConfig
        {
            InitialDelayMs = 100,
            MaxDelayMs = 10000,
            BackoffMultiplier = 2
        };

        var delay0 = config.CalculateDelay(0);
        var delay1 = config.CalculateDelay(1);
        var delay2 = config.CalculateDelay(2);

        Assert.Equal(100, delay0);
        Assert.Equal(200, delay1);
        Assert.Equal(400, delay2);
    }

    [Fact]
    public void RetryConfig_RetryableStatusCodes_ContainsDefaults()
    {
        var config = new Retry.RetryConfig();
        Assert.Contains(429, config.RetryableStatusCodes);
        Assert.Contains(500, config.RetryableStatusCodes);
        Assert.Contains(502, config.RetryableStatusCodes);
        Assert.Contains(503, config.RetryableStatusCodes);
        Assert.Contains(504, config.RetryableStatusCodes);
    }

    [Fact]
    public void HmacSigner_SignAndVerify_Works()
    {
        var signer = new Signer.HmacSigner("test-secret-key");
        var data = "test-data-to-sign";
        var signature = signer.Sign(data);
        Assert.True(signer.Verify(data, signature));
    }

    [Fact]
    public void CaptchaException_Creates_WithMessage()
    {
        var ex = new CaptchaException("Test exception");
        Assert.Equal("Test exception", ex.Message);
        Assert.Null(ex.Code);
        Assert.False(ex.IsRetryable);
    }

    [Fact]
    public void ApiException_Creates_WithCodeAndMessage()
    {
        var ex = new ApiException("API error", 400);
        Assert.Equal("API error", ex.Message);
        Assert.Equal("400", ex.Code);
    }

    [Fact]
    public void Models_CanBeCreated_WithoutErrors()
    {
        var sliderResponse = new SliderCaptchaResponse { SessionId = "test-session" };
        var verifyRequest = new VerifyCaptchaRequest { SessionId = "test-session", Type = "slider" };
        var verifyResponse = new VerifyCaptchaResponse { Success = true };

        Assert.NotNull(sliderResponse);
        Assert.NotNull(verifyRequest);
        Assert.NotNull(verifyResponse);
    }
}
