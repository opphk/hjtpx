namespace CaptchaX.SDK;

public class CaptchaConfig
{
    public string BaseUrl { get; set; }
    public string? AppId { get; set; }
    public int Timeout { get; set; } = 10000;
    public int RetryTimes { get; set; } = 3;
    public ApiVersion ApiVersion { get; set; } = ApiVersion.V1;

    public CaptchaConfig(string baseUrl)
    {
        BaseUrl = baseUrl;
    }
}

public enum ApiVersion
{
    V1,
    V2
}

public class CharPosition
{
    [System.Text.Json.Serialization.JsonPropertyName("char")]
    public string Char { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("x")]
    public int X { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("y")]
    public int Y { get; set; }

    public CharPosition() { }

    public CharPosition(string charValue, int x, int y)
    {
        Char = charValue;
        X = x;
        Y = y;
    }
}

public class SliderCaptchaResult
{
    [System.Text.Json.Serialization.JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("background_b64")]
    public string BackgroundB64 { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("slider_b64")]
    public string SliderB64 { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("target_x")]
    public int TargetX { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("target_y")]
    public int TargetY { get; set; }
}

public class SliderVerifyResult
{
    [System.Text.Json.Serialization.JsonPropertyName("success")]
    public bool Success { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("message")]
    public string Message { get; set; } = string.Empty;
}

public class ClickCaptchaResult
{
    [System.Text.Json.Serialization.JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("image")]
    public string Image { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("target_chars")]
    public List<string> TargetChars { get; set; } = new();

    [System.Text.Json.Serialization.JsonPropertyName("char_positions")]
    public List<CharPosition> CharPositions { get; set; } = new();
}

public class ClickVerifyResult
{
    [System.Text.Json.Serialization.JsonPropertyName("success")]
    public bool Success { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("score")]
    public double Score { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("message")]
    public string Message { get; set; } = string.Empty;
}

public class PuzzleCaptchaResult
{
    [System.Text.Json.Serialization.JsonPropertyName("id")]
    public string Id { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("background_b64")]
    public string BackgroundB64 { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("puzzle_b64")]
    public string PuzzleB64 { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("target_x")]
    public int TargetX { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("target_y")]
    public int TargetY { get; set; }
}

public class PuzzleVerifyResult
{
    [System.Text.Json.Serialization.JsonPropertyName("success")]
    public bool Success { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("message")]
    public string Message { get; set; } = string.Empty;
}

public class Scenario
{
    [System.Text.Json.Serialization.JsonPropertyName("id")]
    public string? Id { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("name")]
    public string Name { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("description")]
    public string? Description { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("difficulty")]
    public string? Difficulty { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("config")]
    public Dictionary<string, object>? Config { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("updated_at")]
    public string? UpdatedAt { get; set; }
}

public class Webhook
{
    [System.Text.Json.Serialization.JsonPropertyName("id")]
    public string? Id { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("app_id")]
    public string AppId { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("url")]
    public string Url { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("secret")]
    public string? Secret { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("events")]
    public List<string> Events { get; set; } = new();

    [System.Text.Json.Serialization.JsonPropertyName("headers")]
    public Dictionary<string, string>? Headers { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("enabled")]
    public bool Enabled { get; set; } = true;

    [System.Text.Json.Serialization.JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("updated_at")]
    public string? UpdatedAt { get; set; }
}

public class BatchVerifyItem
{
    [System.Text.Json.Serialization.JsonPropertyName("captcha_id")]
    public string CaptchaId { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("type")]
    public string Type { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("target_x")]
    public int TargetX { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("target_y")]
    public int? TargetY { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("clicks")]
    public List<CharPosition>? Clicks { get; set; }
}

public class BatchVerifyResult
{
    [System.Text.Json.Serialization.JsonPropertyName("captcha_id")]
    public string CaptchaId { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("success")]
    public bool Success { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("message")]
    public string Message { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("score")]
    public double? Score { get; set; }
}

public class BatchVerifySummary
{
    [System.Text.Json.Serialization.JsonPropertyName("total")]
    public int Total { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("success")]
    public int SuccessCount { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("failed")]
    public int Failed { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("skipped")]
    public int Skipped { get; set; }
}

public class BatchVerifyResponse
{
    [System.Text.Json.Serialization.JsonPropertyName("results")]
    public List<BatchVerifyResult> Results { get; set; } = new();

    [System.Text.Json.Serialization.JsonPropertyName("summary")]
    public BatchVerifySummary Summary { get; set; } = new();
}

public class HealthStatus
{
    [System.Text.Json.Serialization.JsonPropertyName("status")]
    public string Status { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("service")]
    public string Service { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("timestamp")]
    public string Timestamp { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("version")]
    public string Version { get; set; } = string.Empty;
}

internal class ApiResponse<T>
{
    [System.Text.Json.Serialization.JsonPropertyName("code")]
    public int Code { get; set; }

    [System.Text.Json.Serialization.JsonPropertyName("message")]
    public string Message { get; set; } = string.Empty;

    [System.Text.Json.Serialization.JsonPropertyName("data")]
    public T? Data { get; set; }
}
