namespace Hjtpx.Captcha.Exceptions;

public class AuthenticationException : CaptchaException
{
    public string? TokenType { get; }
    public DateTime? ExpiresAt { get; }

    public AuthenticationException(string message) : base(message)
    {
    }

    public AuthenticationException(string message, string? tokenType) : base(message)
    {
        TokenType = tokenType;
    }

    public AuthenticationException(string message, string? tokenType, DateTime? expiresAt) : base(message)
    {
        TokenType = tokenType;
        ExpiresAt = expiresAt;
    }

    public AuthenticationException(string message, Exception innerException) : base(message, innerException)
    {
    }

    public static AuthenticationException InvalidCredentials() =>
        new("Invalid credentials provided");

    public static AuthenticationException TokenExpired(string? tokenType = null, DateTime? expiresAt = null) =>
        new("Authentication token has expired", tokenType, expiresAt);

    public static AuthenticationException TokenMissing() =>
        new("Authentication token is missing");

    public static AuthenticationException InvalidTokenFormat() =>
        new("Invalid token format");

    public static AuthenticationException InsufficientPermissions(string requiredScope) =>
        new($"Insufficient permissions. Required scope: {requiredScope}");
}
