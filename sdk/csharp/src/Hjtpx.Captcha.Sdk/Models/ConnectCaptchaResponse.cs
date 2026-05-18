using System.Text.Json.Serialization;

namespace Hjtpx.Captcha.Models;

public class ConnectCaptchaResponse
{
    [JsonPropertyName("session_id")]
    public string SessionId { get; set; } = string.Empty;

    [JsonPropertyName("image_url")]
    public string ImageUrl { get; set; } = string.Empty;

    [JsonPropertyName("pairs")]
    public List<PairItem>? Pairs { get; set; }

    public class PairItem
    {
        [JsonPropertyName("left")]
        public int Left { get; set; }

        [JsonPropertyName("right")]
        public int Right { get; set; }
    }
}
