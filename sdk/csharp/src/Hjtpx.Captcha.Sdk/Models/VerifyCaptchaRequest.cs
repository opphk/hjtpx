using System.Text.Json.Serialization;

namespace Hjtpx.Captcha.Models;

public class VerifyCaptchaRequest
{
    [JsonPropertyName("session_id")]
    public string SessionId { get; set; } = string.Empty;

    [JsonPropertyName("type")]
    public string Type { get; set; } = string.Empty;

    [JsonPropertyName("x")]
    public int? X { get; set; }

    [JsonPropertyName("y")]
    public int? Y { get; set; }

    [JsonPropertyName("trajectory")]
    public List<TrajectoryPoint>? Trajectory { get; set; }

    [JsonPropertyName("points")]
    public List<List<int>>? Points { get; set; }

    [JsonPropertyName("click_sequence")]
    public List<int>? ClickSequence { get; set; }

    [JsonPropertyName("angle")]
    public int? Angle { get; set; }

    [JsonPropertyName("pattern")]
    public List<int>? Pattern { get; set; }

    [JsonPropertyName("pieces")]
    public List<JigsawPiece>? Pieces { get; set; }

    [JsonPropertyName("answer")]
    public string? Answer { get; set; }

    [JsonPropertyName("connections")]
    public List<List<int>>? Connections { get; set; }

    [JsonPropertyName("target_position")]
    public List<double>? TargetPosition { get; set; }

    [JsonPropertyName("behavior_data")]
    public List<BehaviorDataPoint>? BehaviorData { get; set; }

    public class BehaviorDataPoint
    {
        [JsonPropertyName("x")]
        public int X { get; set; }

        [JsonPropertyName("y")]
        public int Y { get; set; }

        [JsonPropertyName("timestamp")]
        public long Timestamp { get; set; }

        [JsonPropertyName("event")]
        public string Event { get; set; } = string.Empty;
    }
}
