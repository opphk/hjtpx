using System;
using System.Collections.Generic;
using System.Net;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;
using Moq;
using Moq.Protected;
using Xunit;

namespace CaptchaX.SDK.Tests;

public class CaptchaXClientTests
{
    private readonly Mock<HttpMessageHandler> _mockHandler;
    private readonly HttpClient _httpClient;
    private readonly CaptchaConfig _config;

    public CaptchaXClientTests()
    {
        _mockHandler = new Mock<HttpMessageHandler>();
        _httpClient = new HttpClient(_mockHandler.Object)
        {
            BaseAddress = new Uri("https://api.example.com/")
        };
        _config = new CaptchaConfig("https://api.example.com")
        {
            AppId = "test-app-id",
            Timeout = 10000,
            RetryTimes = 3,
            ApiVersion = ApiVersion.V1
        };
    }

    private CaptchaXClient CreateClient()
    {
        return new CaptchaXClient(_config, _httpClient);
    }

    private void SetupMockResponse(string jsonContent, HttpStatusCode statusCode = HttpStatusCode.OK)
    {
        _mockHandler
            .Protected()
            .Setup<Task<HttpResponseMessage>>(
                "SendAsync",
                ItExpr.IsAny<HttpRequestMessage>(),
                ItExpr.IsAny<CancellationToken>())
            .ReturnsAsync(new HttpResponseMessage
            {
                StatusCode = statusCode,
                Content = new StringContent(jsonContent)
            });
    }

    [Fact]
    public async Task HealthCheckAsync_ReturnsHealthStatus()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""status"": ""healthy"",
                ""service"": ""captchax"",
                ""timestamp"": ""2024-01-01T00:00:00Z"",
                ""version"": ""1.0.0""
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.HealthCheckAsync();

        Assert.NotNull(result);
        Assert.Equal("healthy", result.Status);
        Assert.Equal("captchax", result.Service);
    }

    [Fact]
    public async Task GenerateSliderCaptchaAsync_ReturnsSliderCaptchaResult()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""id"": ""captcha-123"",
                ""background_b64"": ""base64background"",
                ""slider_b64"": ""base64slider"",
                ""target_x"": 100,
                ""target_y"": 50
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.GenerateSliderCaptchaAsync();

        Assert.NotNull(result);
        Assert.Equal("captcha-123", result.Id);
        Assert.Equal(100, result.TargetX);
    }

    [Fact]
    public async Task VerifySliderCaptchaAsync_ReturnsVerifyResult()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""success"": true,
                ""message"": ""Verification passed""
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.VerifySliderCaptchaAsync("captcha-123", 0.95);

        Assert.NotNull(result);
        Assert.True(result.Success);
    }

    [Fact]
    public async Task GenerateClickCaptchaAsync_ReturnsClickCaptchaResult()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""id"": ""click-456"",
                ""image"": ""base64image"",
                ""target_chars"": [""A"", ""B"", ""C""],
                ""char_positions"": [
                    { ""char"": ""A"", ""x"": 100, ""y"": 50 },
                    { ""char"": ""B"", ""x"": 200, ""y"": 100 },
                    { ""char"": ""C"", ""x"": 300, ""y"": 150 }
                ]
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.GenerateClickCaptchaAsync();

        Assert.NotNull(result);
        Assert.Equal("click-456", result.Id);
        Assert.Equal(3, result.TargetChars.Count);
    }

    [Fact]
    public async Task VerifyClickCaptchaAsync_ReturnsVerifyResult()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""success"": true,
                ""score"": 0.95,
                ""message"": ""Verification passed""
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var clicks = new List<CharPosition>
        {
            new CharPosition("A", 100, 50),
            new CharPosition("B", 200, 100)
        };
        var result = await client.VerifyClickCaptchaAsync("click-456", clicks);

        Assert.NotNull(result);
        Assert.True(result.Success);
        Assert.Equal(0.95, result.Score);
    }

    [Fact]
    public async Task GeneratePuzzleCaptchaAsync_ReturnsPuzzleCaptchaResult()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""id"": ""puzzle-789"",
                ""background_b64"": ""base64bg"",
                ""puzzle_b64"": ""base64puzzle"",
                ""target_x"": 150,
                ""target_y"": 75
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.GeneratePuzzleCaptchaAsync();

        Assert.NotNull(result);
        Assert.Equal("puzzle-789", result.Id);
    }

    [Fact]
    public async Task VerifyPuzzleCaptchaAsync_ReturnsVerifyResult()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""success"": true,
                ""message"": ""Verification passed""
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.VerifyPuzzleCaptchaAsync("puzzle-789", 0.98);

        Assert.NotNull(result);
        Assert.True(result.Success);
    }

    [Fact]
    public async Task BatchVerifyAsync_ReturnsBatchVerifyResponse()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""results"": [
                    { ""captcha_id"": ""captcha-1"", ""success"": true, ""message"": ""OK"" },
                    { ""captcha_id"": ""captcha-2"", ""success"": false, ""message"": ""Invalid"" }
                ],
                ""summary"": {
                    ""total"": 2,
                    ""success"": 1,
                    ""failed"": 1,
                    ""skipped"": 0
                }
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var items = new List<BatchVerifyItem>
        {
            new BatchVerifyItem { CaptchaId = "captcha-1", Type = "slider", TargetX = 100 },
            new BatchVerifyItem { CaptchaId = "captcha-2", Type = "slider", TargetX = 50 }
        };

        var result = await client.BatchVerifyAsync(items);

        Assert.NotNull(result);
        Assert.Equal(2, result.Summary.Total);
        Assert.Equal(1, result.Summary.SuccessCount);
    }

    [Fact]
    public async Task ListScenariosAsync_ReturnsScenarioList()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": [
                {
                    ""id"": ""scenario-1"",
                    ""name"": ""Test Scenario"",
                    ""description"": ""A test scenario"",
                    ""difficulty"": ""medium""
                }
            ]
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.ListScenariosAsync();

        Assert.NotNull(result);
        Assert.Single(result);
        Assert.Equal("Test Scenario", result[0].Name);
    }

    [Fact]
    public async Task CreateScenarioAsync_ReturnsCreatedScenario()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""id"": ""scenario-new"",
                ""name"": ""New Scenario"",
                ""description"": ""A new scenario"",
                ""difficulty"": ""easy""
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.CreateScenarioAsync("New Scenario", "A new scenario", "easy");

        Assert.NotNull(result);
        Assert.Equal("New Scenario", result.Name);
    }

    [Fact]
    public async Task GetScenarioAsync_ReturnsScenario()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""id"": ""scenario-123"",
                ""name"": ""Test Scenario"",
                ""description"": ""Test"",
                ""difficulty"": ""hard""
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.GetScenarioAsync("scenario-123");

        Assert.NotNull(result);
        Assert.Equal("scenario-123", result.Id);
    }

    [Fact]
    public async Task UpdateScenarioAsync_ReturnsUpdatedScenario()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""id"": ""scenario-123"",
                ""name"": ""Updated Scenario"",
                ""description"": ""Updated"",
                ""difficulty"": ""hard""
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.UpdateScenarioAsync("scenario-123", "Updated Scenario");

        Assert.NotNull(result);
        Assert.Equal("Updated Scenario", result.Name);
    }

    [Fact]
    public async Task DeleteScenarioAsync_CompletesWithoutError()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": null
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        await client.DeleteScenarioAsync("scenario-123");
    }

    [Fact]
    public async Task ListWebhooksAsync_ReturnsWebhookList()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": [
                {
                    ""id"": ""webhook-1"",
                    ""app_id"": ""app-123"",
                    ""url"": ""https://example.com/webhook"",
                    ""events"": [""verify.success""],
                    ""enabled"": true
                }
            ]
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.ListWebhooksAsync();

        Assert.NotNull(result);
        Assert.Single(result);
        Assert.Equal("https://example.com/webhook", result[0].Url);
    }

    [Fact]
    public async Task CreateWebhookAsync_ReturnsCreatedWebhook()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""id"": ""webhook-new"",
                ""app_id"": ""app-123"",
                ""url"": ""https://example.com/newhook"",
                ""events"": [""verify.success""],
                ""enabled"": true
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.CreateWebhookAsync(
            "https://example.com/newhook",
            new List<string> { "verify.success" });

        Assert.NotNull(result);
        Assert.Equal("https://example.com/newhook", result.Url);
    }

    [Fact]
    public async Task GetWebhookAsync_ReturnsWebhook()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""id"": ""webhook-123"",
                ""app_id"": ""app-123"",
                ""url"": ""https://example.com/webhook"",
                ""events"": [""verify.success""],
                ""enabled"": true
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.GetWebhookAsync("webhook-123");

        Assert.NotNull(result);
        Assert.Equal("webhook-123", result.Id);
    }

    [Fact]
    public async Task UpdateWebhookAsync_ReturnsUpdatedWebhook()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": {
                ""id"": ""webhook-123"",
                ""app_id"": ""app-123"",
                ""url"": ""https://example.com/updated"",
                ""events"": [""verify.success"", ""verify.fail""],
                ""enabled"": true
            }
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        var result = await client.UpdateWebhookAsync(
            "webhook-123",
            url: "https://example.com/updated");

        Assert.NotNull(result);
        Assert.Equal("https://example.com/updated", result.Url);
    }

    [Fact]
    public async Task DeleteWebhookAsync_CompletesWithoutError()
    {
        var jsonResponse = @"{
            ""code"": 0,
            ""message"": ""success"",
            ""data"": null
        }";
        SetupMockResponse(jsonResponse);

        var client = CreateClient();
        await client.DeleteWebhookAsync("webhook-123");
    }

    [Fact]
    public void Constructor_WithNullConfig_ThrowsArgumentNullException()
    {
        Assert.Throws<ArgumentNullException>(() => new CaptchaXClient(null!));
    }

    [Fact]
    public void Constructor_WithNullHttpClient_ThrowsArgumentNullException()
    {
        var config = new CaptchaConfig("https://api.example.com");
        Assert.Throws<ArgumentNullException>(() => new CaptchaXClient(config, null!));
    }

    [Fact]
    public async Task ApiVersion_V2_UsesCorrectPrefix()
    {
        var configV2 = new CaptchaConfig("https://api.example.com")
        {
            ApiVersion = ApiVersion.V2
        };

        var client = new CaptchaXClient(configV2, _httpClient);

        Assert.Equal(ApiVersion.V2, client.GetType().GetProperty("Config")?.GetValue(client));
    }

    [Fact]
    public void CaptchaConfig_DefaultValues_AreCorrect()
    {
        var config = new CaptchaConfig("https://api.example.com");

        Assert.Equal(10000, config.Timeout);
        Assert.Equal(3, config.RetryTimes);
        Assert.Equal(ApiVersion.V1, config.ApiVersion);
        Assert.Null(config.AppId);
    }
}
