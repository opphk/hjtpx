using System.Security.Cryptography;
using System.Text;

namespace Hjtpx.Captcha.Signer;

public class HmacSigner
{
    private readonly byte[] _secretKeyBytes;

    public HmacSigner(string secretKey)
    {
        _secretKeyBytes = Encoding.UTF8.GetBytes(secretKey);
    }

    public string Sign(string data)
    {
        using var hmac = new HMACSHA256(_secretKeyBytes);
        byte[] hashBytes = hmac.ComputeHash(Encoding.UTF8.GetBytes(data));
        return Convert.ToBase64String(hashBytes);
    }

    public bool Verify(string data, string signature)
    {
        string expectedSignature = Sign(data);
        return expectedSignature == signature;
    }
}
