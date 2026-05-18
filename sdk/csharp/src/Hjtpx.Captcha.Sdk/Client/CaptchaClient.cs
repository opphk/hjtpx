using System.Text;
using System.Text.Json;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Logging.Abstractions;
using Hjtpx.Captcha.Models;
using Hjtpx.Captcha.Exceptions;
using Hjtpx.Captcha.Signer;
using Hjtpx.Captcha.Retry;

namespace Hjtpx.Captcha.Client;

public class CaptchaClient : IDisposable
{
    private readonly CaptchaClientConfig _config;
    private readonly HttpClient _httpClient;
    private readonly RetryManager _retryManager;
    private readonly HmacSigner? _signer;
    private readonly ILogger<CaptchaClient> _logger;
    private string? _accessToken;
    private bool _disposed;

    public string? AccessToken
    {
        get => _accessToken;
        set => _accessToken = value;
    }

    public CaptchaClientConfig Config => _config;

    public CaptchaClient(string baseUrl, ILogger<CaptchaClient>? logger = null)
        : this(new CaptchaClientConfig(baseUrl), logger)
    {
    }

    public CaptchaClient(string baseUrl, string apiKey, ILogger<CaptchaClient>? logger = null)
        : this(new CaptchaClientConfig(baseUrl, apiKey), logger)
    {
    }

    public CaptchaClient(CaptchaClientConfig config, ILogger<CaptchaClient>? logger = null)
    {
        _config = config;
        _logger = logger ?? NullLogger<CaptchaClient>.Instance;
        
        var handler = new SocketsHttpHandler
        {
            PooledConnectionLifetime = TimeSpan.FromMilliseconds(config.ConnectionPoolConfig.TimeToLiveMs),
            PooledConnectionIdleTimeout = TimeSpan.FromMilliseconds(config.ConnectionPoolConfig.ValidateAfterInactivityMs),
            MaxConnectionsPerServer = config.ConnectionPoolConfig.MaxConnectionsPerRoute,
            ConnectTimeout = TimeSpan.FromMilliseconds(config.ConnectionPoolConfig.ConnectionTimeoutMs)
        };
        
        _httpClient = new HttpClient(handler)
        {
            Timeout = TimeSpan.FromMilliseconds(config.ConnectionPoolConfig.SocketTimeoutMs),
            BaseAddress = new Uri(config.BaseUrl)
        };

        _retryManager = new RetryManager(config.RetryConfig, _logger as ILogger<RetryManager>);

        if (!string.IsNullOrEmpty(config.SecretKey))
        {
            _signer = new HmacSigner(config.SecretKey);
        }
    }

    #region Captcha Methods

    public async Task<SliderCaptchaResponse> GetSliderCaptchaAsync(int? width = null, int? height = null, int? tolerance = null, CancellationToken cancellationToken = default)
    {
        var queryParams = new Dictionary<string, string>();
        if (width.HasValue) queryParams["width"] = width.Value.ToString();
        if (height.HasValue) queryParams["height"] = height.Value.ToString();
        if (tolerance.HasValue) queryParams["tolerance"] = tolerance.Value.ToString();

        return await ExecuteGetAsync<SliderCaptchaResponse>("/api/v1/captcha/slider", queryParams, cancellationToken);
    }

    public async Task<ClickCaptchaResponse> GetClickCaptchaAsync(string? mode = null, bool? shuffle = null, int? points = null, CancellationToken cancellationToken = default)
    {
        var queryParams = new Dictionary<string, string>();
        if (!string.IsNullOrEmpty(mode)) queryParams["mode"] = mode;
        if (shuffle.HasValue) queryParams["shuffle"] = shuffle.Value.ToString().ToLowerInvariant();
        if (points.HasValue) queryParams["points"] = points.Value.ToString();

        return await ExecuteGetAsync<ClickCaptchaResponse>("/api/v1/captcha/click", queryParams, cancellationToken);
    }

    public async Task<RotationCaptchaResponse> GetRotationCaptchaAsync(CancellationToken cancellationToken = default)
    {
        return await ExecuteGetAsync<RotationCaptchaResponse>("/api/v1/captcha/rotation", null, cancellationToken);
    }

    public async Task<GestureCaptchaResponse> GetGestureCaptchaAsync(CancellationToken cancellationToken = default)
    {
        return await ExecuteGetAsync<GestureCaptchaResponse>("/api/v1/captcha/gesture", null, cancellationToken);
    }

    public async Task<JigsawCaptchaResponse> GetJigsawCaptchaAsync(int? width = null, int? height = null, int? gridSize = null, CancellationToken cancellationToken = default)
    {
        var queryParams = new Dictionary<string, string>();
        if (width.HasValue) queryParams["width"] = width.Value.ToString();
        if (height.HasValue) queryParams["height"] = height.Value.ToString();
        if (gridSize.HasValue) queryParams["grid_size"] = gridSize.Value.ToString();

        return await ExecuteGetAsync<JigsawCaptchaResponse>("/api/v1/captcha/jigsaw", queryParams, cancellationToken);
    }

    public async Task<VoiceCaptchaResponse> GetVoiceCaptchaAsync(string? language = null, CancellationToken cancellationToken = default)
    {
        var queryParams = new Dictionary<string, string>();
        if (!string.IsNullOrEmpty(language)) queryParams["language"] = language;

        return await ExecuteGetAsync<VoiceCaptchaResponse>("/api/v1/captcha/voice", queryParams, cancellationToken);
    }

    public async Task<ConnectCaptchaResponse> GetConnectCaptchaAsync(CancellationToken cancellationToken = default)
    {
        return await ExecuteGetAsync<ConnectCaptchaResponse>("/api/v1/captcha/connect", null, cancellationToken);
    }

    public async Task<ThreeDCaptchaResponse> GetThreeDCaptchaAsync(CancellationToken cancellationToken = default)
    {
        return await ExecuteGetAsync<ThreeDCaptchaResponse>("/api/v1/captcha/3d", null, cancellationToken);
    }

    public async Task<VerifyCaptchaResponse> VerifyCaptchaAsync(VerifyCaptchaRequest request, CancellationToken cancellationToken = default)
    {
        return await ExecutePostAsync<VerifyCaptchaResponse>("/api/v1/captcha/verify", request, cancellationToken);
    }

    public async Task<VerifyCaptchaResponse> VerifySliderCaptchaAsync(string sessionId, int x, int? y = null, List<TrajectoryPoint>? trajectory = null, CancellationToken cancellationToken = default)
    {
        var request = new VerifyCaptchaRequest
        {
            SessionId = sessionId,
            Type = "slider",
            X = x,
            Y = y,
            Trajectory = trajectory
        };
        return await VerifyCaptchaAsync(request, cancellationToken);
    }

    public async Task<VerifyCaptchaResponse> VerifyClickCaptchaAsync(string sessionId, List<List<int>> points, List<int>? clickSequence = null, CancellationToken cancellationToken = default)
    {
        var request = new VerifyCaptchaRequest
        {
            SessionId = sessionId,
            Type = "click",
            Points = points,
            ClickSequence = clickSequence
        };
        return await VerifyCaptchaAsync(request, cancellationToken);
    }

    public async Task<VerifyCaptchaResponse> VerifyRotationCaptchaAsync(string challengeId, int angle, CancellationToken cancellationToken = default)
    {
        var request = new VerifyCaptchaRequest
        {
            SessionId = challengeId,
            Type = "rotation",
            Angle = angle
        };
        return await ExecutePostAsync<VerifyCaptchaResponse>("/api/v1/captcha/rotation/verify", request, cancellationToken);
    }

    public async Task<VerifyCaptchaResponse> VerifyGestureCaptchaAsync(string sessionId, List<int> pattern, CancellationToken cancellationToken = default)
    {
        var request = new VerifyCaptchaRequest
        {
            SessionId = sessionId,
            Type = "gesture",
            Pattern = pattern
        };
        return await ExecutePostAsync<VerifyCaptchaResponse>("/api/v1/captcha/gesture/verify", request, cancellationToken);
    }

    public async Task<VerifyCaptchaResponse> VerifyJigsawCaptchaAsync(string sessionId, List<JigsawPiece> pieces, CancellationToken cancellationToken = default)
    {
        var request = new VerifyCaptchaRequest
        {
            SessionId = sessionId,
            Type = "jigsaw",
            Pieces = pieces
        };
        return await ExecutePostAsync<VerifyCaptchaResponse>("/api/v1/captcha/jigsaw/verify", request, cancellationToken);
    }

    public async Task<VerifyCaptchaResponse> VerifyVoiceCaptchaAsync(string sessionId, string answer, CancellationToken cancellationToken = default)
    {
        var request = new VerifyCaptchaRequest
        {
            SessionId = sessionId,
            Type = "voice",
            Answer = answer
        };
        return await VerifyCaptchaAsync(request, cancellationToken);
    }

    public async Task<VerifyCaptchaResponse> VerifyConnectCaptchaAsync(string sessionId, List<List<int>> connections, CancellationToken cancellationToken = default)
    {
        var request = new VerifyCaptchaRequest
        {
            SessionId = sessionId,
            Type = "connect",
            Connections = connections
        };
        return await VerifyCaptchaAsync(request, cancellationToken);
    }

    public async Task<VerifyCaptchaResponse> VerifyThreeDCaptchaAsync(string sessionId, List<double> targetPosition, CancellationToken cancellationToken = default)
    {
        var request = new VerifyCaptchaRequest
        {
            SessionId = sessionId,
            Type = "3d",
            TargetPosition = targetPosition
        };
        return await VerifyCaptchaAsync(request, cancellationToken);
    }

    #endregion

    #region Auth Methods

    public async Task<LoginResponse> LoginAsync(string username, string password, string? captchaToken = null, CancellationToken cancellationToken = default)
    {
        var request = new LoginRequest
        {
            Username = username,
            Password = password,
            CaptchaToken = captchaToken
        };

        var response = await ExecutePostAsync<LoginResponse>("/api/v1/auth/login", request, cancellationToken);
        _accessToken = response.AccessToken;
        return response;
    }

    public async Task LogoutAsync(CancellationToken cancellationToken = default)
    {
        await ExecutePostAsync<object>("/api/v1/auth/logout", null, cancellationToken);
        _accessToken = null;
    }

    #endregion

    #region Detection Methods

    public async Task<string> GetDetectionScriptAsync(string? callback = null, CancellationToken cancellationToken = default)
    {
        var queryParams = new Dictionary<string, string>();
        if (!string.IsNullOrEmpty(callback)) queryParams["callback"] = callback;

        return await _retryManager.ExecuteAsync(async () =>
        {
            var request = BuildRequest(HttpMethod.Get, "/api/v1/detect/script", queryParams);
            
            using var response = await _httpClient.SendAsync(request, cancellationToken);
            
            if (!response.IsSuccessStatusCode)
            {
                var content = await response.Content.ReadAsStringAsync(cancellationToken);
                throw new ApiException($"Failed to get detection script: {content}", (int)response.StatusCode);
            }

            return await response.Content.ReadAsStringAsync(cancellationToken);
        }, cancellationToken);
    }

    public async Task<Dictionary<string, object>> SubmitDetectionAsync(Dictionary<string, object> data, CancellationToken cancellationToken = default)
    {
        var response = await ExecutePostAsync<Dictionary<string, object>>("/api/v1/detect/submit", data, cancellationToken);
        return response;
    }

    public async Task<Dictionary<string, object>> CheckEnvironmentAsync(Dictionary<string, object> data, CancellationToken cancellationToken = default)
    {
        var response = await ExecutePostAsync<Dictionary<string, object>>("/api/v1/detect/check", data, cancellationToken);
        return response;
    }

    #endregion

    #region Helper Methods

    private async Task<T> ExecuteGetAsync<T>(string path, Dictionary<string, string>? queryParams, CancellationToken cancellationToken = default)
    {
        return await _retryManager.ExecuteAsync(async () =>
        {
            var request = BuildRequest(HttpMethod.Get, path, queryParams);
            return await ExecuteRequestAsync<T>(request, cancellationToken);
        }, cancellationToken);
    }

    private async Task<T> ExecutePostAsync<T>(string path, object? body, CancellationToken cancellationToken = default)
    {
        return await _retryManager.ExecuteAsync(async () =>
        {
            var request = BuildRequest(HttpMethod.Post, path, null);
            
            if (body != null)
            {
                var json = JsonSerializer.Serialize(body);
                request.Content = new StringContent(json, Encoding.UTF8, "application/json");
            }
            
            return await ExecuteRequestAsync<T>(request, cancellationToken);
        }, cancellationToken);
    }

    private HttpRequestMessage BuildRequest(HttpMethod method, string path, Dictionary<string, string>? queryParams)
    {
        var uriBuilder = new UriBuilder(new Uri(_httpClient.BaseAddress!, path));
        
        if (queryParams != null && queryParams.Count > 0)
        {
            var query = new StringBuilder();
            foreach (var param in queryParams)
            {
                if (query.Length > 0)
                    query.Append('&');
                query.Append($"{Uri.EscapeDataString(param.Key)}={Uri.EscapeDataString(param.Value)}");
            }
            uriBuilder.Query = query.ToString();
        }

        var request = new HttpRequestMessage(method, uriBuilder.Uri);
        AddHeaders(request, uriBuilder.Path);
        return request;
    }

    private void AddHeaders(HttpRequestMessage request, string path)
    {
        request.Headers.Add("User-Agent", "HJTPX-Captcha-CSharp-SDK/1.0.0");

        if (!string.IsNullOrEmpty(_config.ApiKey))
        {
            request.Headers.Add("X-API-Key", _config.ApiKey);
        }

        if (!string.IsNullOrEmpty(_accessToken))
        {
            request.Headers.Add("Authorization", $"Bearer {_accessToken}");
        }

        if (_signer != null)
        {
            long timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();
            string dataToSign = $"{timestamp}:{path}";
            string signature = _signer.Sign(dataToSign);
            request.Headers.Add("X-Timestamp", timestamp.ToString());
            request.Headers.Add("X-Signature", signature);
        }
    }

    private async Task<T> ExecuteRequestAsync<T>(HttpRequestMessage request, CancellationToken cancellationToken = default)
    {
        using var response = await _httpClient.SendAsync(request, cancellationToken);
        var content = await response.Content.ReadAsStringAsync(cancellationToken);

        if (!response.IsSuccessStatusCode)
        {
            throw new ApiException($"API request failed: {content}", (int)response.StatusCode, _config.RetryConfig.IsRetryableStatusCode((int)response.StatusCode));
        }

        if (typeof(T) == typeof(string))
        {
            return (T)(object)content;
        }

        var apiResponse = JsonSerializer.Deserialize<ApiResponse<T>>(content, new JsonSerializerOptions
        {
            PropertyNameCaseInsensitive = true
        });

        if (apiResponse == null)
        {
            throw new CaptchaException("Failed to deserialize API response");
        }

        if (!apiResponse.IsSuccess)
        {
            throw new ApiException(apiResponse.Message, apiResponse.Code);
        }

        return apiResponse.Data!;
    }

    #endregion

    #region IDisposable

    public void Dispose()
    {
        Dispose(true);
        GC.SuppressFinalize(this);
    }

    protected virtual void Dispose(bool disposing)
    {
        if (_disposed)
            return;

        if (disposing)
        {
            _httpClient.Dispose();
        }

        _disposed = true;
    }

    #endregion
}
