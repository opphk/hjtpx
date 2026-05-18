using Xunit;
using Hjtpx.Captcha.Client;
using Hjtpx.Captcha.Constants;
using Hjtpx.Captcha.Exceptions;
using Hjtpx.Captcha.Models;
using Hjtpx.Captcha.Pool;
using Hjtpx.Captcha.Retry;
using Hjtpx.Captcha.Signer;

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
    public void ClientConfig_Created_WithAllParameters()
    {
        var config = new CaptchaClientConfig("http://localhost:8080", "api-key", "secret-key");
        Assert.Equal("http://localhost:8080", config.BaseUrl);
        Assert.Equal("api-key", config.ApiKey);
        Assert.Equal("secret-key", config.SecretKey);
    }

    [Theory]
    [InlineData("http://localhost:8080")]
    [InlineData("https://api.example.com")]
    [InlineData("http://localhost:8080/")]
    public void Client_AcceptsVariousBaseUrls(string baseUrl)
    {
        using var client = new CaptchaClient(baseUrl);
        Assert.NotNull(client);
    }

    [Fact]
    public void Client_AccessToken_CanBeSetAndRetrieved()
    {
        _client.AccessToken = "test-token";
        Assert.Equal("test-token", _client.AccessToken);
    }
}

public class RetryConfigTests
{
    [Fact]
    public void RetryConfig_CalculatesDelay_Correctly()
    {
        var config = new RetryConfig
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
    public void RetryConfig_Delay_RespectsMaxDelay()
    {
        var config = new RetryConfig
        {
            InitialDelayMs = 1000,
            MaxDelayMs = 5000,
            BackoffMultiplier = 2
        };

        var delay5 = config.CalculateDelay(5);
        Assert.Equal(32000, delay5);
        Assert.True(delay5 > config.MaxDelayMs);
        Assert.Equal(config.MaxDelayMs, Math.Min(delay5, config.MaxDelayMs));
    }

    [Fact]
    public void RetryConfig_RetryableStatusCodes_ContainsDefaults()
    {
        var config = new RetryConfig();
        Assert.Contains(429, config.RetryableStatusCodes);
        Assert.Contains(500, config.RetryableStatusCodes);
        Assert.Contains(502, config.RetryableStatusCodes);
        Assert.Contains(503, config.RetryableStatusCodes);
        Assert.Contains(504, config.RetryableStatusCodes);
    }

    [Theory]
    [InlineData(429, true)]
    [InlineData(500, true)]
    [InlineData(502, true)]
    [InlineData(503, true)]
    [InlineData(504, true)]
    [InlineData(400, false)]
    [InlineData(401, false)]
    [InlineData(404, false)]
    public void RetryConfig_IsRetryableStatusCode_Works(int statusCode, bool expected)
    {
        var config = new RetryConfig();
        Assert.Equal(expected, config.IsRetryableStatusCode(statusCode));
    }

    [Fact]
    public void RetryConfig_DefaultValues_AreCorrect()
    {
        var config = new RetryConfig();
        Assert.Equal(3, config.MaxRetries);
        Assert.Equal(100, config.InitialDelayMs);
        Assert.Equal(10000, config.MaxDelayMs);
        Assert.Equal(2.0, config.BackoffMultiplier);
    }
}

public class HmacSignerTests
{
    [Fact]
    public void HmacSigner_SignAndVerify_Works()
    {
        var signer = new HmacSigner("test-secret-key");
        var data = "test-data-to-sign";
        var signature = signer.Sign(data);
        Assert.True(signer.Verify(data, signature));
    }

    [Fact]
    public void HmacSigner_DifferentKeys_ProduceDifferentSignatures()
    {
        var signer1 = new HmacSigner("secret-key-1");
        var signer2 = new HmacSigner("secret-key-2");
        var data = "test-data";

        var sig1 = signer1.Sign(data);
        var sig2 = signer2.Sign(data);

        Assert.NotEqual(sig1, sig2);
    }

    [Fact]
    public void HmacSigner_SameInput_ProducesSameSignature()
    {
        var signer = new HmacSigner("secret-key");
        var data = "test-data";

        var sig1 = signer.Sign(data);
        var sig2 = signer.Sign(data);

        Assert.Equal(sig1, sig2);
    }

    [Fact]
    public void HmacSigner_TamperedData_FailsVerification()
    {
        var signer = new HmacSigner("secret-key");
        var data = "original-data";
        var signature = signer.Sign(data);

        var tamperedData = "tampered-data";
        Assert.False(signer.Verify(tamperedData, signature));
    }

    [Theory]
    [InlineData("")]
    [InlineData("short")]
    [InlineData("a very long string that contains many characters for testing")]
    public void HmacSigner_WorksWithVariousInputLengths(string data)
    {
        var signer = new HmacSigner("secret-key");
        var signature = signer.Sign(data);
        Assert.NotEmpty(signature);
        Assert.True(signer.Verify(data, signature));
    }
}

public class CaptchaTypesTests
{
    [Theory]
    [InlineData("slider", true)]
    [InlineData("click", true)]
    [InlineData("rotation", true)]
    [InlineData("gesture", true)]
    [InlineData("jigsaw", true)]
    [InlineData("voice", true)]
    [InlineData("connect", true)]
    [InlineData("3d", true)]
    [InlineData("invalid", false)]
    [InlineData("", false)]
    [InlineData(null, false)]
    public void CaptchaTypes_IsValid_Works(string? type, bool expected)
    {
        Assert.Equal(expected, CaptchaTypes.IsValid(type));
    }

    [Fact]
    public void CaptchaTypes_AllTypes_ContainsAllTypes()
    {
        Assert.Equal(8, CaptchaTypes.AllTypes.Length);
        Assert.Contains("slider", CaptchaTypes.AllTypes);
        Assert.Contains("click", CaptchaTypes.AllTypes);
        Assert.Contains("rotation", CaptchaTypes.AllTypes);
        Assert.Contains("gesture", CaptchaTypes.AllTypes);
        Assert.Contains("jigsaw", CaptchaTypes.AllTypes);
        Assert.Contains("voice", CaptchaTypes.AllTypes);
        Assert.Contains("connect", CaptchaTypes.AllTypes);
        Assert.Contains("3d", CaptchaTypes.AllTypes);
    }

    [Theory]
    [InlineData("slider", "滑块验证码")]
    [InlineData("click", "点击验证码")]
    [InlineData("rotation", "旋转验证码")]
    [InlineData("gesture", "手势验证码")]
    [InlineData("jigsaw", "拼图验证码")]
    [InlineData("voice", "语音验证码")]
    [InlineData("connect", "连连看验证码")]
    [InlineData("3d", "3D验证码")]
    public void CaptchaTypes_GetDisplayName_ReturnsCorrectName(string type, string expected)
    {
        Assert.Equal(expected, CaptchaTypes.GetDisplayName(type));
    }
}

public class ExceptionTests
{
    [Fact]
    public void CaptchaException_Creates_WithMessage()
    {
        var ex = new CaptchaException("Test exception");
        Assert.Equal("Test exception", ex.Message);
        Assert.Null(ex.ErrorCode);
    }

    [Fact]
    public void CaptchaException_Creates_WithMessageAndCode()
    {
        var ex = new CaptchaException("Test exception", 1001);
        Assert.Equal("Test exception", ex.Message);
        Assert.Equal(1001, ex.ErrorCode);
    }

    [Fact]
    public void CaptchaException_Creates_WithInnerException()
    {
        var inner = new InvalidOperationException("Inner");
        var ex = new CaptchaException("Outer", inner);
        Assert.Same(inner, ex.InnerException);
    }

    [Fact]
    public void ApiException_Creates_WithCodeAndMessage()
    {
        var ex = new ApiException("API error", 400, false);
        Assert.Equal("API error", ex.Message);
        Assert.Equal(400, ex.StatusCode);
        Assert.False(ex.IsRetryable);
    }

    [Fact]
    public void ApiException_RetryableStatusCodes_AreCorrect()
    {
        var ex500 = new ApiException("Server error", 500, true);
        var ex429 = new ApiException("Rate limited", 429, true);
        var ex400 = new ApiException("Bad request", 400, false);

        Assert.True(ex500.IsRetryable);
        Assert.True(ex429.IsRetryable);
        Assert.False(ex400.IsRetryable);
    }

    [Fact]
    public void ApiException_FromStatusCode_CreatesCorrectException()
    {
        var ex = ApiException.FromStatusCode(401);
        Assert.Equal(401, ex.StatusCode);
        Assert.Contains("Unauthorized", ex.Message);
    }

    [Fact]
    public void ValidationException_FactoryMethods_Work()
    {
        var ex1 = ValidationException.EmptySessionId();
        Assert.Contains("Session ID", ex1.Message);

        var ex2 = ValidationException.Required("email");
        Assert.Contains("email", ex2.Message);

        var ex3 = ValidationException.InvalidFormat("phone", "digits only");
        Assert.Contains("phone", ex3.Message);
    }

    [Fact]
    public void AuthenticationException_FactoryMethods_Work()
    {
        var ex1 = AuthenticationException.InvalidCredentials();
        Assert.Contains("Invalid credentials", ex1.Message);

        var ex2 = AuthenticationException.TokenExpired();
        Assert.Contains("expired", ex2.Message);

        var ex3 = AuthenticationException.TokenMissing();
        Assert.Contains("missing", ex3.Message);
    }

    [Fact]
    public void NetworkException_FactoryMethods_Work()
    {
        var ex1 = NetworkException.ConnectionTimeout("example.com", 5000);
        Assert.Contains("timed out", ex1.Message);
        Assert.Equal("example.com", ex1.Host);

        var ex2 = NetworkException.ConnectionRefused("example.com", 8080);
        Assert.Contains("refused", ex2.Message);
    }
}

public class ConnectionPoolConfigTests
{
    [Fact]
    public void ConnectionPoolConfig_DefaultValues_AreCorrect()
    {
        var config = new ConnectionPoolConfig();
        Assert.Equal(100, config.MaxConnections);
        Assert.Equal(50, config.MaxConnectionsPerRoute);
        Assert.Equal(5000, config.ConnectionTimeoutMs);
        Assert.Equal(30000, config.SocketTimeoutMs);
        Assert.Equal(60000, config.TimeToLiveMs);
        Assert.Equal(2000, config.ValidateAfterInactivityMs);
    }

    [Fact]
    public void ConnectionPoolConfig_CanSetCustomValues()
    {
        var config = new ConnectionPoolConfig
        {
            MaxConnections = 200,
            MaxConnectionsPerRoute = 100,
            ConnectionTimeoutMs = 10000
        };

        Assert.Equal(200, config.MaxConnections);
        Assert.Equal(100, config.MaxConnectionsPerRoute);
        Assert.Equal(10000, config.ConnectionTimeoutMs);
    }
}

public class ModelsTests
{
    [Fact]
    public void SliderCaptchaResponse_CanBeCreated()
    {
        var response = new SliderCaptchaResponse
        {
            SessionId = "test-session",
            ImageUrl = "http://example.com/image.jpg",
            PuzzleUrl = "http://example.com/puzzle.jpg",
            SecretY = 50,
            ImageWidth = 320,
            ImageHeight = 160,
            Tolerance = 5
        };

        Assert.Equal("test-session", response.SessionId);
        Assert.Equal(50, response.SecretY);
    }

    [Fact]
    public void VerifyCaptchaRequest_CanBeCreated()
    {
        var request = new VerifyCaptchaRequest
        {
            SessionId = "test-session",
            Type = "slider",
            X = 100,
            Y = 50
        };

        Assert.Equal("test-session", request.SessionId);
        Assert.Equal("slider", request.Type);
        Assert.Equal(100, request.X);
    }

    [Fact]
    public void VerifyCaptchaResponse_CanBeCreated()
    {
        var response = new VerifyCaptchaResponse
        {
            Success = true,
            Message = "Verification passed",
            RemainingAttempts = 3,
            RiskScore = 0.1
        };

        Assert.True(response.Success);
        Assert.Equal("Verification passed", response.Message);
        Assert.Equal(3, response.RemainingAttempts);
    }

    [Fact]
    public void TrajectoryPoint_CanBeCreated_WithConstructor()
    {
        var point = new TrajectoryPoint(100, 50, 1234567890);
        Assert.Equal(100, point.X);
        Assert.Equal(50, point.Y);
        Assert.Equal(1234567890, point.Timestamp);
    }

    [Fact]
    public void TrajectoryPoint_CanBeCreated_WithDefaultConstructor()
    {
        var point = new TrajectoryPoint
        {
            X = 100,
            Y = 50,
            Timestamp = 1234567890
        };

        Assert.Equal(100, point.X);
        Assert.Equal(50, point.Y);
    }

    [Fact]
    public void ApiResponse_IsSuccess_Works()
    {
        var successResponse = new ApiResponse<string> { Code = 0, Data = "test" };
        var errorResponse = new ApiResponse<string> { Code = 1001, Message = "Error" };

        Assert.True(successResponse.IsSuccess);
        Assert.False(errorResponse.IsSuccess);
    }
}

public class ApiConstantsTests
{
    [Fact]
    public void ApiConstants_DefaultValues_AreCorrect()
    {
        Assert.Equal("http://localhost:8080", ApiConstants.DefaultBaseUrl);
        Assert.Equal("HJTPX-Captcha-CSharp-SDK", ApiConstants.DefaultUserAgent);
    }

    [Fact]
    public void ApiConstants_Headers_AreCorrect()
    {
        Assert.Equal("X-API-Key", ApiConstants.Headers.ApiKey);
        Assert.Equal("X-Timestamp", ApiConstants.Headers.Timestamp);
        Assert.Equal("X-Signature", ApiConstants.Headers.Signature);
        Assert.Equal("Authorization", ApiConstants.Headers.Authorization);
    }

    [Fact]
    public void ApiConstants_StatusCodes_AreCorrect()
    {
        Assert.Equal(0, ApiConstants.StatusCodes.Success);
        Assert.Equal(1001, ApiConstants.StatusCodes.InvalidApiKey);
        Assert.Equal(1002, ApiConstants.StatusCodes.InvalidSignature);
        Assert.Equal(1003, ApiConstants.StatusCodes.SessionExpired);
        Assert.Equal(1004, ApiConstants.StatusCodes.CaptchaExpired);
    }
}
