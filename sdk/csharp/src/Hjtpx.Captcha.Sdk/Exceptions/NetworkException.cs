using System.Net;

namespace Hjtpx.Captcha.Exceptions;

public class NetworkException : CaptchaException
{
    public string? Host { get; }
    public int? Port { get; }
    public string? Endpoint { get; }
    public HttpRequestException? HttpException { get; }

    public NetworkException(string message) : base(message)
    {
    }

    public NetworkException(string message, Exception innerException) : base(message, innerException)
    {
        if (innerException is HttpRequestException httpEx)
        {
            HttpException = httpEx;
            Host = httpEx.Host;
            Port = httpEx.Port;
        }
    }

    public NetworkException(string message, string host, int? port = null, string? endpoint = null)
        : base(message)
    {
        Host = host;
        Port = port;
        Endpoint = endpoint;
    }

    public NetworkException(string message, Exception innerException, string host, int? port = null)
        : base(message, innerException)
    {
        Host = host;
        Port = port;

        if (innerException is HttpRequestException httpEx)
        {
            HttpException = httpEx;
        }
    }

    public static NetworkException ConnectionTimeout(string host, int? timeoutMs = null) =>
        new($"Connection to {host} timed out after {timeoutMs}ms", host, timeoutMs);

    public static NetworkException ConnectionRefused(string host, int? port = null) =>
        new($"Connection to {host}:{port} was refused", host, port);

    public static NetworkException ConnectionReset(string host) =>
        new($"Connection to {host} was reset", host);

    public static NetworkException HostUnreachable(string host) =>
        new($"Host {host} is unreachable", host);

    public static NetworkException DnsResolutionFailed(string host) =>
        new($"DNS resolution failed for host: {host}", host);

    public static NetworkException SslHandshakeFailed(string host) =>
        new($"SSL/TLS handshake failed for host: {host}", host);

    public static NetworkException RequestAborted(string endpoint) =>
        new($"Request to {endpoint} was aborted", endpoint);

    public override string ToString()
    {
        var sb = new System.Text.StringBuilder();
        sb.AppendLine($"Network Exception: {Message}");

        if (!string.IsNullOrEmpty(Host))
        {
            sb.AppendLine($"Host: {Host}");
        }

        if (Port.HasValue)
        {
            sb.AppendLine($"Port: {Port}");
        }

        if (!string.IsNullOrEmpty(Endpoint))
        {
            sb.AppendLine($"Endpoint: {Endpoint}");
        }

        if (HttpException?.StatusCode != null)
        {
            sb.AppendLine($"HTTP Status: {HttpException.StatusCode}");
        }

        return sb.ToString();
    }
}
