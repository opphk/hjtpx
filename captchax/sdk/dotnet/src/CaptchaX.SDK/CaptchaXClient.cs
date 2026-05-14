using System;
using System.Collections.Generic;
using System.Net.Http;
using System.Net.Http.Json;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Threading;
using System.Threading.Tasks;

namespace CaptchaX.SDK;

public class CaptchaXClient : IDisposable
{
    private readonly HttpClient _httpClient;
    private readonly CaptchaConfig _config;
    private bool _disposed;

    public CaptchaXClient(CaptchaConfig config)
    {
        _config = config ?? throw new ArgumentNullException(nameof(config));
        _httpClient = new HttpClient
        {
            BaseAddress = new Uri(config.BaseUrl.TrimEnd('/') + "/"),
            Timeout = TimeSpan.FromMilliseconds(config.Timeout)
        };
        _httpClient.DefaultRequestHeaders.Add("Accept", "application/json");
        if (!string.IsNullOrEmpty(config.AppId))
        {
            _httpClient.DefaultRequestHeaders.Add("X-App-Id", config.AppId);
        }
    }

    public CaptchaXClient(CaptchaConfig config, HttpClient httpClient)
    {
        _config = config ?? throw new ArgumentNullException(nameof(config));
        _httpClient = httpClient ?? throw new ArgumentNullException(nameof(httpClient));
    }

    private async Task<T> ExecuteWithRetryAsync<T>(
        Func<Task<T>> action,
        CancellationToken cancellationToken = default)
    {
        var attempts = 0;
        var delay = TimeSpan.FromMilliseconds(500);

        while (true)
        {
            try
            {
                return await action().ConfigureAwait(false);
            }
            catch (Exception ex) when (attempts < _config.RetryTimes && IsRetryableException(ex))
            {
                attempts++;
                if (attempts >= _config.RetryTimes)
                {
                    throw new CaptchaXException($"Request failed after {_config.RetryTimes} attempts", ex);
                }
                await Task.Delay(delay, cancellationToken).ConfigureAwait(false);
                delay = TimeSpan.FromMilliseconds(delay.TotalMilliseconds * 2);
            }
        }
    }

    private static bool IsRetryableException(Exception ex)
    {
        return ex is HttpRequestException ||
               ex is TaskCanceledException ||
               ex is OperationCanceledException;
    }

    private string GetApiVersionPrefix()
    {
        return _config.ApiVersion == ApiVersion.V2 ? "v2" : "v1";
    }

    private async Task<ApiResponse<T>> SendRequestAsync<T>(
        HttpMethod method,
        string endpoint,
        object? body = null,
        Dictionary<string, string>? queryParams = null,
        CancellationToken cancellationToken = default)
    {
        var url = $"{GetApiVersionPrefix()}/{endpoint.TrimStart('/')}";

        if (queryParams != null && queryParams.Count > 0)
        {
            var queryString = string.Join("&", queryParams);
            url = $"{url}?{queryString}";
        }

        var request = new HttpRequestMessage(method, url);

        if (body != null)
        {
            var jsonOptions = new JsonSerializerOptions
            {
                PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower,
                DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
            };
            request.Content = JsonContent.Create(body, options: jsonOptions);
        }

        HttpResponseMessage response;
        try
        {
            response = await _httpClient.SendAsync(request, cancellationToken).ConfigureAwait(false);
        }
        catch (Exception ex)
        {
            throw new CaptchaXException($"HTTP request failed: {ex.Message}", ex);
        }

        if (!response.IsSuccessStatusCode)
        {
            var content = await response.Content.ReadAsStringAsync(cancellationToken).ConfigureAwait(false);
            throw new CaptchaXException($"HTTP {(int)response.StatusCode}: {content}", null);
        }

        var jsonOptions2 = new JsonSerializerOptions
        {
            PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower,
            PropertyNameCaseInsensitive = true
        };

        return await response.Content.ReadFromJsonAsync<ApiResponse<T>>(jsonOptions2, cancellationToken).ConfigureAwait(false)
               ?? throw new CaptchaXException("Empty response received");
    }

    public async Task<HealthStatus> HealthCheckAsync(CancellationToken cancellationToken = default)
    {
        var result = await ExecuteWithRetryAsync(async () =>
        {
            var response = await SendRequestAsync<HealthStatus>(
                HttpMethod.Get,
                "health",
                cancellationToken: cancellationToken).ConfigureAwait(false);
            return response;
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Health check returned null");
    }

    public async Task<SliderCaptchaResult> GenerateSliderCaptchaAsync(
        string? scenarioId = null,
        string? extraData = null,
        CancellationToken cancellationToken = default)
    {
        var body = new Dictionary<string, object?>();
        if (!string.IsNullOrEmpty(scenarioId)) body["scenario_id"] = scenarioId;
        if (!string.IsNullOrEmpty(extraData)) body["extra_data"] = extraData;

        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<SliderCaptchaResult>(
                HttpMethod.Post,
                "captcha/slider",
                body,
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Slider captcha generation failed");
    }

    public async Task<SliderVerifyResult> VerifySliderCaptchaAsync(
        string captchaId,
        double distance,
        CancellationToken cancellationToken = default)
    {
        var body = new Dictionary<string, object>
        {
            ["captcha_id"] = captchaId,
            ["distance"] = distance
        };

        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<SliderVerifyResult>(
                HttpMethod.Post,
                "captcha/slider/verify",
                body,
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Slider captcha verification failed");
    }

    public async Task<ClickCaptchaResult> GenerateClickCaptchaAsync(
        string? scenarioId = null,
        string? extraData = null,
        CancellationToken cancellationToken = default)
    {
        var body = new Dictionary<string, object?>();
        if (!string.IsNullOrEmpty(scenarioId)) body["scenario_id"] = scenarioId;
        if (!string.IsNullOrEmpty(extraData)) body["extra_data"] = extraData;

        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<ClickCaptchaResult>(
                HttpMethod.Post,
                "captcha/click",
                body,
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Click captcha generation failed");
    }

    public async Task<ClickVerifyResult> VerifyClickCaptchaAsync(
        string captchaId,
        List<CharPosition> clicks,
        CancellationToken cancellationToken = default)
    {
        var body = new Dictionary<string, object>
        {
            ["captcha_id"] = captchaId,
            ["clicks"] = clicks
        };

        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<ClickVerifyResult>(
                HttpMethod.Post,
                "captcha/click/verify",
                body,
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Click captcha verification failed");
    }

    public async Task<PuzzleCaptchaResult> GeneratePuzzleCaptchaAsync(
        string? scenarioId = null,
        string? extraData = null,
        CancellationToken cancellationToken = default)
    {
        var body = new Dictionary<string, object?>();
        if (!string.IsNullOrEmpty(scenarioId)) body["scenario_id"] = scenarioId;
        if (!string.IsNullOrEmpty(extraData)) body["extra_data"] = extraData;

        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<PuzzleCaptchaResult>(
                HttpMethod.Post,
                "captcha/puzzle",
                body,
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Puzzle captcha generation failed");
    }

    public async Task<PuzzleVerifyResult> VerifyPuzzleCaptchaAsync(
        string captchaId,
        double distance,
        CancellationToken cancellationToken = default)
    {
        var body = new Dictionary<string, object>
        {
            ["captcha_id"] = captchaId,
            ["distance"] = distance
        };

        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<PuzzleVerifyResult>(
                HttpMethod.Post,
                "captcha/puzzle/verify",
                body,
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Puzzle captcha verification failed");
    }

    public async Task<BatchVerifyResponse> BatchVerifyAsync(
        List<BatchVerifyItem> items,
        CancellationToken cancellationToken = default)
    {
        var body = new Dictionary<string, object>
        {
            ["items"] = items
        };

        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<BatchVerifyResponse>(
                HttpMethod.Post,
                "captcha/batch/verify",
                body,
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Batch verify failed");
    }

    public async Task<List<Scenario>> ListScenariosAsync(CancellationToken cancellationToken = default)
    {
        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<List<Scenario>>(
                HttpMethod.Get,
                "scenarios",
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? new List<Scenario>();
    }

    public async Task<Scenario> CreateScenarioAsync(
        string name,
        string? description = null,
        string? difficulty = null,
        Dictionary<string, object>? config = null,
        CancellationToken cancellationToken = default)
    {
        var body = new Dictionary<string, object?>
        {
            ["name"] = name,
            ["description"] = description,
            ["difficulty"] = difficulty,
            ["config"] = config
        };

        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<Scenario>(
                HttpMethod.Post,
                "scenarios",
                body,
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Scenario creation failed");
    }

    public async Task<Scenario> GetScenarioAsync(string scenarioId, CancellationToken cancellationToken = default)
    {
        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<Scenario>(
                HttpMethod.Get,
                $"scenarios/{scenarioId}",
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Get scenario failed");
    }

    public async Task<Scenario> UpdateScenarioAsync(
        string scenarioId,
        string? name = null,
        string? description = null,
        string? difficulty = null,
        Dictionary<string, object>? config = null,
        CancellationToken cancellationToken = default)
    {
        var body = new Dictionary<string, object?>
        {
            ["name"] = name,
            ["description"] = description,
            ["difficulty"] = difficulty,
            ["config"] = config
        };

        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<Scenario>(
                HttpMethod.Put,
                $"scenarios/{scenarioId}",
                body,
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Scenario update failed");
    }

    public async Task DeleteScenarioAsync(string scenarioId, CancellationToken cancellationToken = default)
    {
        await ExecuteWithRetryAsync(async () =>
        {
            await SendRequestAsync<object>(
                HttpMethod.Delete,
                $"scenarios/{scenarioId}",
                cancellationToken: cancellationToken).ConfigureAwait(false);
            return true;
        }, cancellationToken).ConfigureAwait(false);
    }

    public async Task<List<Webhook>> ListWebhooksAsync(CancellationToken cancellationToken = default)
    {
        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<List<Webhook>>(
                HttpMethod.Get,
                "webhooks",
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? new List<Webhook>();
    }

    public async Task<Webhook> CreateWebhookAsync(
        string url,
        List<string> events,
        string? secret = null,
        Dictionary<string, string>? headers = null,
        bool enabled = true,
        CancellationToken cancellationToken = default)
    {
        var body = new Dictionary<string, object?>
        {
            ["url"] = url,
            ["events"] = events,
            ["secret"] = secret,
            ["headers"] = headers,
            ["enabled"] = enabled
        };

        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<Webhook>(
                HttpMethod.Post,
                "webhooks",
                body,
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Webhook creation failed");
    }

    public async Task<Webhook> GetWebhookAsync(string webhookId, CancellationToken cancellationToken = default)
    {
        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<Webhook>(
                HttpMethod.Get,
                $"webhooks/{webhookId}",
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Get webhook failed");
    }

    public async Task<Webhook> UpdateWebhookAsync(
        string webhookId,
        string? url = null,
        List<string>? events = null,
        string? secret = null,
        Dictionary<string, string>? headers = null,
        bool? enabled = null,
        CancellationToken cancellationToken = default)
    {
        var body = new Dictionary<string, object?>
        {
            ["url"] = url,
            ["events"] = events,
            ["secret"] = secret,
            ["headers"] = headers,
            ["enabled"] = enabled
        };

        var result = await ExecuteWithRetryAsync(async () =>
        {
            return await SendRequestAsync<Webhook>(
                HttpMethod.Put,
                $"webhooks/{webhookId}",
                body,
                cancellationToken: cancellationToken).ConfigureAwait(false);
        }, cancellationToken).ConfigureAwait(false);

        return result.Data ?? throw new CaptchaXException("Webhook update failed");
    }

    public async Task DeleteWebhookAsync(string webhookId, CancellationToken cancellationToken = default)
    {
        await ExecuteWithRetryAsync(async () =>
        {
            await SendRequestAsync<object>(
                HttpMethod.Delete,
                $"webhooks/{webhookId}",
                cancellationToken: cancellationToken).ConfigureAwait(false);
            return true;
        }, cancellationToken).ConfigureAwait(false);
    }

    public void Dispose()
    {
        Dispose(true);
        GC.SuppressFinalize(this);
    }

    protected virtual void Dispose(bool disposing)
    {
        if (!_disposed)
        {
            if (disposing)
            {
                _httpClient.Dispose();
            }
            _disposed = true;
        }
    }
}
