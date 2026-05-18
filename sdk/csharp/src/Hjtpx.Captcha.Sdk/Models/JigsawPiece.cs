using System.Text.Json.Serialization;

namespace Hjtpx.Captcha.Models;

public class JigsawPiece
{
    [JsonPropertyName("index")]
    public int Index { get; set; }

    [JsonPropertyName("original_x")]
    public int OriginalX { get; set; }

    [JsonPropertyName("original_y")]
    public int OriginalY { get; set; }

    [JsonPropertyName("current_x")]
    public int CurrentX { get; set; }

    [JsonPropertyName("current_y")]
    public int CurrentY { get; set; }

    [JsonPropertyName("width")]
    public int Width { get; set; }

    [JsonPropertyName("height")]
    public int Height { get; set; }

    [JsonPropertyName("rotation")]
    public int Rotation { get; set; }

    public JigsawPiece()
    {
    }

    public JigsawPiece(int index, int originalX, int originalY, int currentX, int currentY, int width, int height)
    {
        Index = index;
        OriginalX = originalX;
        OriginalY = originalY;
        CurrentX = currentX;
        CurrentY = currentY;
        Width = width;
        Height = height;
        Rotation = 0;
    }
}
