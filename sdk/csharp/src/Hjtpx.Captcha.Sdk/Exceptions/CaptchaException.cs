namespace Hjtpx.Captcha.Exceptions;

public class CaptchaException : Exception
{
    public string? Code { get; }
    public bool IsRetryable { get; }

    public CaptchaException(string message)
        : base(message)
    {
    }

    public CaptchaException(string message, string? code)
        : base(message)
    {
        Code = code;
    }

    public CaptchaException(string message, string? code, bool isRetryable)
        : base(message)
    {
        Code = code;
        IsRetryable = isRetryable;
    }

    public CaptchaException(string message, Exception innerException)
        : base(message, innerException)
    {
    }

    public CaptchaException(string message, string? code, Exception innerException)
        : base(message, innerException)
    {
        Code = code;
    }

    public CaptchaException(string message, string? code, bool isRetryable, Exception innerException)
        : base(message, innerException)
    {
        Code = code;
        IsRetryable = isRetryable;
    }
}
