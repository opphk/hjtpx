namespace Hjtpx.Captcha.Exceptions;

public class AuthenticationException : CaptchaException
{
    public AuthenticationException(string message)
        : base(message, "AUTH_ERROR", false)
    {
    }

    public AuthenticationException(string message, Exception innerException)
        : base(message, "AUTH_ERROR", false, innerException)
    {
    }
}
