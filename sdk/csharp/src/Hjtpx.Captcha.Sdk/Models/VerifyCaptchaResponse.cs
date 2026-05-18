using System.Text.Json.Serialization;

namespace Hjtpx.Captcha.Models;

public class VerifyCaptchaResponse
{
    [JsonPropertyName("success")]
    public bool Success { get; set; }

    [JsonPropertyName("message")]
    public string Message { get; set; } = string.Empty;

    [JsonPropertyName("remaining_attempts")]
    public int? RemainingAttempts { get; set; }

    [JsonPropertyName("risk_score")]
    public double? RiskScore { get; set; }

    [JsonPropertyName("captcha_pass")]
    public bool? CaptchaPass { get; set; }

    [JsonPropertyName("fail_reason")]
    public string? FailReason { get; set; }

    [JsonPropertyName("trajectory_result")]
    public TrajectoryResult? TrajectoryResult { get; set; }

    public class TrajectoryResult
    {
        [JsonPropertyName("score")]
        public double Score { get; set; }

        [JsonPropertyName("passed")]
        public bool Passed { get; set; }

        [JsonPropertyName("reasons")]
        public List<string>? Reasons { get; set; }
    }
}
