using System.Text.Json.Serialization;

namespace Hjtpx.Captcha.Models;

public class ClickCaptchaResponse
{
    [JsonPropertyName("session_id")]
    public string SessionId { get; set; } = string.Empty;

    [JsonPropertyName("image_url")]
    public string ImageUrl { get; set; } = string.Empty;

    [JsonPropertyName("hint")]
    public string Hint { get; set; } = string.Empty;

    [JsonPropertyName("hint_order")]
    public List<int>? HintOrder { get; set; }

    [JsonPropertyName("max_points")]
    public int MaxPoints { get; set; }

    [JsonPropertyName("mode")]
    public string Mode { get; set; } = string.Empty;

    [JsonPropertyName("allow_shuffle")]
    public bool AllowShuffle { get; set; }

    [JsonPropertyName("points")]
    public List<List<int>>? Points { get; set; }
}
