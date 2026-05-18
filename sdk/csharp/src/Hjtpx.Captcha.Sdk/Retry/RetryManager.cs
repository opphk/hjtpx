using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Logging.Abstractions;

namespace Hjtpx.Captcha.Retry;

public class RetryManager
{
    private readonly RetryConfig _config;
    private readonly ILogger<RetryManager> _logger;

    public RetryManager(RetryConfig? config = null, ILogger<RetryManager>? logger = null)
    {
        _config = config ?? new RetryConfig();
        _logger = logger ?? NullLogger<RetryManager>.Instance;
    }

    public async Task<T> ExecuteAsync<T>(Func<Task<T>> func, CancellationToken cancellationToken = default)
    {
        int attempt = 0;
        Exception? lastException = null;

        while (attempt <= _config.MaxRetries)
        {
            try
            {
                return await func();
            }
            catch (Exception e)
            {
                lastException = e;
                attempt++;

                if (attempt > _config.MaxRetries)
                {
                    _logger.LogWarning("Max retries ({MaxRetries}) reached, giving up", _config.MaxRetries);
                    break;
                }

                if (!ShouldRetry(e))
                {
                    _logger.LogWarning("Exception not retryable, giving up: {Message}", e.Message);
                    break;
                }

                long delay = _config.CalculateDelay(attempt - 1);
                _logger.LogInformation("Retry attempt {Attempt}/{MaxRetries}, waiting {Delay}ms", attempt, _config.MaxRetries, delay);

                await Task.Delay((int)delay, cancellationToken);
            }
        }

        throw lastException!;
    }

    public T Execute<T>(Func<T> func)
    {
        int attempt = 0;
        Exception? lastException = null;

        while (attempt <= _config.MaxRetries)
        {
            try
            {
                return func();
            }
            catch (Exception e)
            {
                lastException = e;
                attempt++;

                if (attempt > _config.MaxRetries)
                {
                    _logger.LogWarning("Max retries ({MaxRetries}) reached, giving up", _config.MaxRetries);
                    break;
                }

                if (!ShouldRetry(e))
                {
                    _logger.LogWarning("Exception not retryable, giving up: {Message}", e.Message);
                    break;
                }

                long delay = _config.CalculateDelay(attempt - 1);
                _logger.LogInformation("Retry attempt {Attempt}/{MaxRetries}, waiting {Delay}ms", attempt, _config.MaxRetries, delay);

                Thread.Sleep((int)delay);
            }
        }

        throw lastException!;
    }

    private bool ShouldRetry(Exception e)
    {
        return _config.IsRetryableException(e) || IsNetworkException(e);
    }

    private bool IsNetworkException(Exception e)
    {
        string? message = e.Message?.ToLowerInvariant();
        if (string.IsNullOrEmpty(message))
        {
            return false;
        }

        return message.Contains("timeout") ||
               message.Contains("connection reset") ||
               message.Contains("connection refused") ||
               message.Contains("socket hang up") ||
               message.Contains("network");
    }
}
