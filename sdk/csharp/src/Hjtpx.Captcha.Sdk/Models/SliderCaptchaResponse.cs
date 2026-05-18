using System.Text.Json.Serialization;

namespace Hjtpx.Captcha.Models;

public class SliderCaptchaResponse
{
    [JsonPropertyName("session_id")]
    public string SessionId { get; set; } = string.Empty;

    [JsonPropertyName("image_url")]
    public string ImageUrl { get; set; } = string.Empty;

    [JsonPropertyName("puzzle_url")]
    public string PuzzleUrl { get; set; } = string.Empty;

    [JsonPropertyName("hint_url")]
    public string HintUrl { get; set; } = string.Empty;

    [JsonPropertyName("shape")]
    public int Shape { get; set; }

    [JsonPropertyName("secret_y")]
    public int SecretY { get; set; }

    [JsonPropertyName("image_width")]
    public int ImageWidth { get; set; }

    [JsonPropertyName("image_height")]
    public int ImageHeight { get; set; }

    [JsonPropertyName("tolerance")]
    public int Tolerance { get; set; }
}
