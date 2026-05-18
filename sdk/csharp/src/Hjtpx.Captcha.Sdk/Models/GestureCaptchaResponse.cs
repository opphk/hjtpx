using System.Text.Json.Serialization;

namespace Hjtpx.Captcha.Models;

public class GestureCaptchaResponse
{
    [JsonPropertyName("session_id")]
    public string SessionId { get; set; } = string.Empty;

    [JsonPropertyName("pattern")]
    public string Pattern { get; set; } = string.Empty;

    [JsonPropertyName("grid_size")]
    public int GridSize { get; set; }

    [JsonPropertyName("hint")]
    public string Hint { get; set; } = string.Empty;
}
