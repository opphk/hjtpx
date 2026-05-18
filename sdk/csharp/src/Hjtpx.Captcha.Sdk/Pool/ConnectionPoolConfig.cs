namespace Hjtpx.Captcha.Pool;

public class ConnectionPoolConfig
{
    public int MaxConnections { get; set; } = 100;
    public int MaxConnectionsPerRoute { get; set; } = 50;
    public int ConnectionTimeoutMs { get; set; } = 5000;
    public int SocketTimeoutMs { get; set; } = 30000;
    public int ConnectionRequestTimeoutMs { get; set; } = 5000;
    public int TimeToLiveMs { get; set; } = 60000;
    public bool ValidateAfterInactivity { get; set; } = true;
    public int ValidateAfterInactivityMs { get; set; } = 2000;
}
