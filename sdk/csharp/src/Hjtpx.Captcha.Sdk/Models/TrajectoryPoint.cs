using System.Text.Json.Serialization;

namespace Hjtpx.Captcha.Models;

public class TrajectoryPoint
{
    [JsonPropertyName("x")]
    public int X { get; set; }

    [JsonPropertyName("y")]
    public int Y { get; set; }

    [JsonPropertyName("t")]
    public long Timestamp { get; set; }

    public TrajectoryPoint()
    {
    }

    public TrajectoryPoint(int x, int y, long timestamp)
    {
        X = x;
        Y = y;
        Timestamp = timestamp;
    }
}
