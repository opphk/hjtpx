using System.Text.Json.Serialization;

namespace Hjtpx.Captcha.Models;

public class JigsawCaptchaResponse
{
    [JsonPropertyName("session_id")]
    public string SessionId { get; set; } = string.Empty;

    [JsonPropertyName("image_url")]
    public string ImageUrl { get; set; } = string.Empty;

    [JsonPropertyName("pieces")]
    public List<JigsawPiece>? Pieces { get; set; }

    [JsonPropertyName("piece_images")]
    public List<string>? PieceImages { get; set; }

    [JsonPropertyName("grid_size")]
    public int GridSize { get; set; }

    [JsonPropertyName("piece_width")]
    public int PieceWidth { get; set; }

    [JsonPropertyName("piece_height")]
    public int PieceHeight { get; set; }

    [JsonPropertyName("image_width")]
    public int ImageWidth { get; set; }

    [JsonPropertyName("image_height")]
    public int ImageHeight { get; set; }
}
