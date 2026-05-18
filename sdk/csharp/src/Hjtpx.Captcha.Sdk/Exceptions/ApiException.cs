namespace Hjtpx.Captcha.Exceptions;

public class ApiException : CaptchaException
{
    public int StatusCode { get; }
    public bool IsRetryable { get; }
    public string? ResponseBody { get; }
    public string? RequestPath { get; }

    public ApiException(string message, int statusCode, bool isRetryable = false)
        : base(message, statusCode)
    {
        StatusCode = statusCode;
        IsRetryable = isRetryable;
    }

    public ApiException(string message, int statusCode, bool isRetryable, string? responseBody)
        : base(message, statusCode)
    {
        StatusCode = statusCode;
        IsRetryable = isRetryable;
        ResponseBody = responseBody;
    }

    public ApiException(string message, int statusCode, string? responseBody, string? requestPath)
        : base(message, statusCode)
    {
        StatusCode = statusCode;
        ResponseBody = responseBody;
        RequestPath = requestPath;
    }

    public ApiException(string message, int statusCode, bool isRetryable, Exception innerException)
        : base(message, statusCode, innerException)
    {
        StatusCode = statusCode;
        IsRetryable = isRetryable;
    }

    public override string ToString()
    {
        var sb = new System.Text.StringBuilder();
        sb.AppendLine($"API Exception: {Message}");
        sb.AppendLine($"Status Code: {StatusCode}");
        sb.AppendLine($"Is Retryable: {IsRetryable}");

        if (!string.IsNullOrEmpty(RequestPath))
        {
            sb.AppendLine($"Request Path: {RequestPath}");
        }

        if (!string.IsNullOrEmpty(ResponseBody))
        {
            sb.AppendLine($"Response Body: {ResponseBody}");
        }

        if (InnerException != null)
        {
            sb.AppendLine($"Inner Exception: {InnerException.Message}");
        }

        return sb.ToString();
    }

    public static ApiException FromStatusCode(int statusCode, string? responseBody = null, string? requestPath = null)
    {
        var (message, isRetryable) = statusCode switch
        {
            400 => ("Bad Request - Invalid request parameters", false),
            401 => ("Unauthorized - Invalid or missing authentication", false),
            403 => ("Forbidden - Insufficient permissions", false),
            404 => ("Not Found - Resource not found", false),
            429 => ("Too Many Requests - Rate limit exceeded", true),
            500 => ("Internal Server Error", true),
            502 => ("Bad Gateway", true),
            503 => ("Service Unavailable", true),
            504 => ("Gateway Timeout", true),
            _ => ("Unknown error occurred", statusCode >= 500)
        };

        return new ApiException(message, statusCode, isRetryable, responseBody)
        {
            RequestPath = requestPath
        };
    }
}
