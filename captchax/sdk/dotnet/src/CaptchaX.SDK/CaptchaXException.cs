namespace CaptchaX.SDK;

public class CaptchaXException : Exception
{
    public int Code { get; }
    public int StatusCode { get; }

    public CaptchaXException(string message, int code = 500, int statusCode = 500)
        : base(message)
    {
        Code = code;
        StatusCode = statusCode;
    }

    public CaptchaXException(string message, Exception innerException)
        : base(message, innerException)
    {
        Code = 500;
        StatusCode = 500;
    }
}
