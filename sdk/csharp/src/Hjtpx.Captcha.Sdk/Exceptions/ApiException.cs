namespace Hjtpx.Captcha.Exceptions;

public class ApiException : CaptchaException
{
    public ApiException(string message, string? code)
        : base(message, code, false)
    {
    }

    public ApiException(string message, string? code, bool isRetryable)
        : base(message, code, isRetryable)
    {
    }

    public ApiException(string message, string? code, Exception innerException)
        : base(message, code, false, innerException)
    {
    }
}
