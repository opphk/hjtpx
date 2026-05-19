using Hjtpx.Captcha.Models;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Logging.Abstractions;
using System.Collections.Concurrent;
using System.Text.Json;

namespace Hjtpx.Captcha.Client;

public class BatchCaptchaClient
{
    private readonly CaptchaClient _client;
    private readonly int _concurrency;
    private readonly ILogger<BatchCaptchaClient> _logger;

    public BatchCaptchaClient(CaptchaClient client, int concurrency = 10)
        : this(client, concurrency, null)
    {
    }

    public BatchCaptchaClient(CaptchaClient client, int concurrency, ILogger<BatchCaptchaClient>? logger)
    {
        _client = client;
        _concurrency = concurrency > 0 ? concurrency : 10;
        _logger = logger ?? NullLogger<BatchCaptchaClient>.Instance;
    }

    public async Task<List<BatchCaptchaResult<T>>> BatchGenerateAsync<T>(
        List<CaptchaGenerationRequest> requests,
        CancellationToken cancellationToken = default)
    {
        var results = new ConcurrentBag<BatchCaptchaResult<T>>();
        var semaphore = new SemaphoreSlim(_concurrency);

        var tasks = requests.Select(async (req, index) =>
        {
            await semaphore.WaitAsync(cancellationToken);
            try
            {
                var result = await GenerateCaptchaAsync<T>(req, index, cancellationToken);
                results.Add(result);
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "Failed to generate captcha at index {Index}", index);
                results.Add(new BatchCaptchaResult<T>
                {
                    Index = index,
                    Success = false,
                    ErrorMessage = ex.Message
                });
            }
            finally
            {
                semaphore.Release();
            }
        });

        await Task.WhenAll(tasks);
        return results.OrderBy(r => r.Index).ToList();
    }

    private async Task<BatchCaptchaResult<T>> GenerateCaptchaAsync<T>(
        CaptchaGenerationRequest request,
        int index,
        CancellationToken cancellationToken)
    {
        object? result = null;

        switch (request.CaptchaType)
        {
            case CaptchaType.Slider:
                result = await _client.GetSliderCaptchaAsync(
                    request.Width, request.Height, request.Tolerance, cancellationToken);
                break;
            case CaptchaType.Click:
                result = await _client.GetClickCaptchaAsync(
                    request.Mode, request.Shuffle, request.Points, cancellationToken);
                break;
            case CaptchaType.Rotation:
                result = await _client.GetRotationCaptchaAsync(cancellationToken);
                break;
            case CaptchaType.Gesture:
                result = await _client.GetGestureCaptchaAsync(cancellationToken);
                break;
            case CaptchaType.Jigsaw:
                result = await _client.GetJigsawCaptchaAsync(
                    request.Width, request.Height, request.GridSize, cancellationToken);
                break;
            case CaptchaType.Voice:
                result = await _client.GetVoiceCaptchaAsync(request.Language, cancellationToken);
                break;
            case CaptchaType.Connect:
                result = await _client.GetConnectCaptchaAsync(cancellationToken);
                break;
            case CaptchaType.ThreeD:
                result = await _client.GetThreeDCaptchaAsync(cancellationToken);
                break;
            default:
                throw new ArgumentException($"Unsupported captcha type: {request.CaptchaType}");
        }

        return new BatchCaptchaResult<T>
        {
            Index = index,
            Success = true,
            SessionId = GetSessionId(result),
            Data = (T?)result
        };
    }

    public async Task<List<BatchVerifyResult>> BatchVerifyAsync(
        List<CaptchaVerifyRequest> requests,
        CancellationToken cancellationToken = default)
    {
        var results = new ConcurrentBag<BatchVerifyResult>();
        var semaphore = new SemaphoreSlim(_concurrency);

        var tasks = requests.Select(async (req, index) =>
        {
            await semaphore.WaitAsync(cancellationToken);
            try
            {
                var result = await VerifyCaptchaAsync(req, index, cancellationToken);
                results.Add(result);
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "Failed to verify captcha at index {Index}", index);
                results.Add(new BatchVerifyResult
                {
                    Index = index,
                    Success = false,
                    ErrorMessage = ex.Message
                });
            }
            finally
            {
                semaphore.Release();
            }
        });

        await Task.WhenAll(tasks);
        return results.OrderBy(r => r.Index).ToList();
    }

    private async Task<BatchVerifyResult> VerifyCaptchaAsync(
        CaptchaVerifyRequest request,
        int index,
        CancellationToken cancellationToken)
    {
        VerifyCaptchaResponse? response = null;

        switch (request.CaptchaType)
        {
            case CaptchaType.Slider:
                response = await _client.VerifySliderCaptchaAsync(
                    request.SessionId, request.X ?? 0, request.Y, request.Trajectory, cancellationToken);
                break;
            case CaptchaType.Click:
                response = await _client.VerifyClickCaptchaAsync(
                    request.SessionId, request.Points ?? new List<List<int>>(), request.ClickSequence, cancellationToken);
                break;
            case CaptchaType.Rotation:
                response = await _client.VerifyRotationCaptchaAsync(
                    request.SessionId, request.Angle ?? 0, cancellationToken);
                break;
            case CaptchaType.Gesture:
                response = await _client.VerifyGestureCaptchaAsync(
                    request.SessionId, request.Pattern ?? new List<int>(), cancellationToken);
                break;
            case CaptchaType.Jigsaw:
                response = await _client.VerifyJigsawCaptchaAsync(
                    request.SessionId, request.Pieces ?? new List<JigsawPiece>(), cancellationToken);
                break;
            case CaptchaType.Voice:
                response = await _client.VerifyVoiceCaptchaAsync(
                    request.SessionId, request.Answer ?? string.Empty, cancellationToken);
                break;
            case CaptchaType.Connect:
                response = await _client.VerifyConnectCaptchaAsync(
                    request.SessionId, request.Connections ?? new List<List<int>>(), cancellationToken);
                break;
            case CaptchaType.ThreeD:
                response = await _client.VerifyThreeDCaptchaAsync(
                    request.SessionId, request.TargetPosition ?? new List<double>(), cancellationToken);
                break;
            default:
                throw new ArgumentException($"Unsupported captcha type: {request.CaptchaType}");
        }

        return new BatchVerifyResult
        {
            Index = index,
            Success = response?.Success ?? false,
            Message = response?.Message,
            RemainingAttempts = response?.RemainingAttempts,
            RiskScore = response?.RiskScore,
            CaptchaPass = response?.CaptchaPass,
            FailReason = response?.FailReason
        };
    }

    private string? GetSessionId(object? result)
    {
        return result switch
        {
            SliderCaptchaResponse r => r.SessionId,
            ClickCaptchaResponse r => r.SessionId,
            RotationCaptchaResponse r => r.ChallengeId,
            GestureCaptchaResponse r => r.SessionId,
            JigsawCaptchaResponse r => r.SessionId,
            VoiceCaptchaResponse r => r.SessionId,
            ConnectCaptchaResponse r => r.SessionId,
            ThreeDCaptchaResponse r => r.SessionId,
            _ => null
        };
    }
}

public enum CaptchaType
{
    Slider,
    Click,
    Rotation,
    Gesture,
    Jigsaw,
    Voice,
    Connect,
    ThreeD
}

public class CaptchaGenerationRequest
{
    public CaptchaType CaptchaType { get; set; }
    public int? Width { get; set; }
    public int? Height { get; set; }
    public int? Tolerance { get; set; }
    public int? Points { get; set; }
    public string? Mode { get; set; }
    public bool? Shuffle { get; set; }
    public int? GridSize { get; set; }
    public string? Language { get; set; }
}

public class CaptchaVerifyRequest
{
    public CaptchaType CaptchaType { get; set; }
    public string SessionId { get; set; } = string.Empty;
    public int? X { get; set; }
    public int? Y { get; set; }
    public int? Angle { get; set; }
    public string? Answer { get; set; }
    public List<int>? Pattern { get; set; }
    public List<List<int>>? Points { get; set; }
    public List<JigsawPiece>? Pieces { get; set; }
    public List<List<int>>? Connections { get; set; }
    public List<double>? TargetPosition { get; set; }
    public List<TrajectoryPoint>? Trajectory { get; set; }
    public List<int>? ClickSequence { get; set; }
}

public class BatchCaptchaResult<T>
{
    public int Index { get; set; }
    public bool Success { get; set; }
    public string? SessionId { get; set; }
    public T? Data { get; set; }
    public string? ErrorMessage { get; set; }
}

public class BatchVerifyResult
{
    public int Index { get; set; }
    public bool Success { get; set; }
    public string? Message { get; set; }
    public int? RemainingAttempts { get; set; }
    public double? RiskScore { get; set; }
    public bool? CaptchaPass { get; set; }
    public string? FailReason { get; set; }
    public string? ErrorMessage { get; set; }
}
