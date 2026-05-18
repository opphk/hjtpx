namespace Hjtpx.Captcha.Constants;

public static class CaptchaTypes
{
    public const string Slider = "slider";
    public const string Click = "click";
    public const string Rotation = "rotation";
    public const string Gesture = "gesture";
    public const string Jigsaw = "jigsaw";
    public const string Voice = "voice";
    public const string Connect = "connect";
    public const string ThreeD = "3d";

    public static readonly string[] AllTypes = new[]
    {
        Slider,
        Click,
        Rotation,
        Gesture,
        Jigsaw,
        Voice,
        Connect,
        ThreeD
    };

    public static bool IsValid(string? type)
    {
        return type != null && AllTypes.Contains(type);
    }

    public static string GetDisplayName(string type)
    {
        return type switch
        {
            Slider => "滑块验证码",
            Click => "点击验证码",
            Rotation => "旋转验证码",
            Gesture => "手势验证码",
            Jigsaw => "拼图验证码",
            Voice => "语音验证码",
            Connect => "连连看验证码",
            ThreeD => "3D验证码",
            _ => type
        };
    }
}

public enum CaptchaType
{
    Slider,
    Click,
    Rotation,
    Gesture,
    Jigsaw,
    Voice,
    Connect,
    ThreeD
}

public static class ApiConstants
{
    public const string DefaultBaseUrl = "http://localhost:8080";
    public const string DefaultUserAgent = "HJTPX-Captcha-CSharp-SDK";

    public static class Headers
    {
        public const string ApiKey = "X-API-Key";
        public const string Timestamp = "X-Timestamp";
        public const string Signature = "X-Signature";
        public const string Authorization = "Authorization";
        public const string UserAgent = "User-Agent";
    }

    public static class Paths
    {
        public const string SliderCaptcha = "/api/v1/captcha/slider";
        public const string ClickCaptcha = "/api/v1/captcha/click";
        public const string RotationCaptcha = "/api/v1/captcha/rotation";
        public const string GestureCaptcha = "/api/v1/captcha/gesture";
        public const string JigsawCaptcha = "/api/v1/captcha/jigsaw";
        public const string VoiceCaptcha = "/api/v1/captcha/voice";
        public const string ConnectCaptcha = "/api/v1/captcha/connect";
        public const string ThreeDCaptcha = "/api/v1/captcha/3d";
        public const string VerifyCaptcha = "/api/v1/captcha/verify";
        public const string RotationVerify = "/api/v1/captcha/rotation/verify";
        public const string GestureVerify = "/api/v1/captcha/gesture/verify";
        public const string JigsawVerify = "/api/v1/captcha/jigsaw/verify";
        public const string Login = "/api/v1/auth/login";
        public const string Logout = "/api/v1/auth/logout";
        public const string DetectionScript = "/api/v1/detect/script";
        public const string DetectionSubmit = "/api/v1/detect/submit";
        public const string DetectionCheck = "/api/v1/detect/check";
    }

    public static class StatusCodes
    {
        public const int Success = 0;
        public const int InvalidApiKey = 1001;
        public const int InvalidSignature = 1002;
        public const int SessionExpired = 1003;
        public const int CaptchaExpired = 1004;
        public const int CaptchaVerified = 1005;
        public const int InvalidCaptcha = 1006;
        public const int RateLimited = 1007;
        public const int InvalidParameter = 1008;
        public const int InternalError = 5000;
    }
}
