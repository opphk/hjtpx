using System.Text.Json.Serialization;

namespace Hjtpx.Captcha.Models;

public class RotationCaptchaResponse
{
    [JsonPropertyName("challenge_id")]
    public string ChallengeId { get; set; } = string.Empty;

    [JsonPropertyName("image_url")]
    public string ImageUrl { get; set; } = string.Empty;
}
