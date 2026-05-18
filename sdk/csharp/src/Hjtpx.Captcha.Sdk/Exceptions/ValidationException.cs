namespace Hjtpx.Captcha.Exceptions;

public class ValidationException : CaptchaException
{
    public ValidationException(string message)
        : base(message, "VALIDATION_ERROR", false)
    {
    }

    public ValidationException(string message, Exception innerException)
        : base(message, "VALIDATION_ERROR", false, innerException)
    {
    }
}
