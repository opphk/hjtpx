using System.Text.Json.Serialization;

namespace Hjtpx.Captcha.Models;

public class LoginRequest
{
    [JsonPropertyName("username")]
    public string Username { get; set; } = string.Empty;

    [JsonPropertyName("password")]
    public string Password { get; set; } = string.Empty;

    [JsonPropertyName("captcha_token")]
    public string? CaptchaToken { get; set; }
}
