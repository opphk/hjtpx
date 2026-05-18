namespace Hjtpx.Captcha.Exceptions;

public class CaptchaException : Exception
{
    public int? ErrorCode { get; }

    public CaptchaException(string message) : base(message)
    {
    }

    public CaptchaException(string message, Exception innerException) : base(message, innerException)
    {
    }

    public CaptchaException(string message, int errorCode) : base(message)
    {
        ErrorCode = errorCode;
    }

    public CaptchaException(string message, int errorCode, Exception innerException) : base(message, innerException)
    {
        ErrorCode = errorCode;
    }
}
