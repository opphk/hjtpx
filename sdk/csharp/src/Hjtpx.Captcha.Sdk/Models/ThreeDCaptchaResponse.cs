using System.Text.Json.Serialization;

namespace Hjtpx.Captcha.Models;

public class ThreeDCaptchaResponse
{
    [JsonPropertyName("session_id")]
    public string SessionId { get; set; } = string.Empty;

    [JsonPropertyName("scene_url")]
    public string SceneUrl { get; set; } = string.Empty;

    [JsonPropertyName("target_id")]
    public string TargetId { get; set; } = string.Empty;

    [JsonPropertyName("hint")]
    public string Hint { get; set; } = string.Empty;
}
