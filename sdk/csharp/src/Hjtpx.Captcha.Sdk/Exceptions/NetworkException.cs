namespace Hjtpx.Captcha.Exceptions;

public class NetworkException : CaptchaException
{
    public NetworkException(string message)
        : base(message, "NETWORK_ERROR", true)
    {
    }

    public NetworkException(string message, Exception innerException)
        : base(message, "NETWORK_ERROR", true, innerException)
    {
    }
}
