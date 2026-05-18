namespace Hjtpx.Captcha.Retry;

public class RetryConfig
{
    public int MaxRetries { get; set; } = 3;
    public long InitialDelayMs { get; set; } = 100;
    public long MaxDelayMs { get; set; } = 10000;
    public double BackoffMultiplier { get; set; } = 2.0;
    public List<Type> RetryableExceptions { get; set; } = new List<Type>();
    public List<int> RetryableStatusCodes { get; set; } = new List<int> { 429, 500, 502, 503, 504 };

    public bool IsRetryableStatusCode(int statusCode)
    {
        return RetryableStatusCodes.Contains(statusCode);
    }

    public bool IsRetryableException(Exception e)
    {
        return RetryableExceptions.Any(t => t.IsInstanceOfType(e));
    }

    public long CalculateDelay(int attempt)
    {
        long delay = (long)(InitialDelayMs * Math.Pow(BackoffMultiplier, attempt));
        return Math.Min(delay, MaxDelayMs);
    }
}
