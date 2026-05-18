using System.Text.Json.Serialization;

namespace Hjtpx.Captcha.Models;

public class ApiResponse<T>
{
    [JsonPropertyName("code")]
    public int Code { get; set; }

    [JsonPropertyName("message")]
    public string Message { get; set; } = string.Empty;

    [JsonPropertyName("data")]
    public T? Data { get; set; }

    public bool IsSuccess => Code == 0;
}
