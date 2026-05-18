using System.Collections.Concurrent;
using System.Net;
using System.Net.Http;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Logging.Abstractions;

namespace Hjtpx.Captcha.Pool;

public interface IConnectionPool : IDisposable
{
    HttpClient GetClient();
    void ReturnClient(HttpClient client);
    PoolStatistics GetStatistics();
    void Reset();
}

public class PoolStatistics
{
    public int TotalConnections { get; set; }
    public int ActiveConnections { get; set; }
    public int AvailableConnections { get; set; }
    public int ConnectionErrors { get; set; }
    public long TotalRequests { get; set; }
    public long FailedRequests { get; set; }
    public DateTime LastResetTime { get; set; }
}

public class ConnectionPool : IConnectionPool
{
    private readonly ConnectionPoolConfig _config;
    private readonly ILogger<ConnectionPool> _logger;
    private readonly ConcurrentBag<HttpClient> _availableClients;
    private readonly HashSet<HttpClient> _allClients;
    private readonly object _lockObject = new();
    private int _totalCreated;
    private int _connectionErrors;
    private long _totalRequests;
    private long _failedRequests;
    private DateTime _lastResetTime;
    private bool _disposed;

    public ConnectionPool(ConnectionPoolConfig? config = null, ILogger<ConnectionPool>? logger = null)
    {
        _config = config ?? new ConnectionPoolConfig();
        _logger = logger ?? NullLogger<ConnectionPool>.Instance;
        _availableClients = new ConcurrentBag<HttpClient>();
        _allClients = new HashSet<HttpClient>();
        _lastResetTime = DateTime.UtcNow;

        PreWarmConnections();
    }

    private void PreWarmConnections()
    {
        var preWarmCount = Math.Min(_config.MaxConnectionsPerRoute, 5);
        
        for (int i = 0; i < preWarmCount; i++)
        {
            var client = CreateHttpClient();
            _availableClients.Add(client);
            lock (_lockObject)
            {
                _allClients.Add(client);
                _totalCreated++;
            }
        }

        _logger.LogInformation("Connection pool pre-warmed with {Count} connections", preWarmCount);
    }

    private HttpClient CreateHttpClient()
    {
        var handler = new SocketsHttpHandler
        {
            PooledConnectionLifetime = TimeSpan.FromMilliseconds(_config.TimeToLiveMs),
            PooledConnectionIdleTimeout = TimeSpan.FromMilliseconds(_config.ValidateAfterInactivityMs),
            MaxConnectionsPerServer = _config.MaxConnectionsPerRoute,
            ConnectTimeout = TimeSpan.FromMilliseconds(_config.ConnectionTimeoutMs),
            AutomaticDecompression = DecompressionMethods.All,
            AllowAutoRedirect = true,
            MaxAutomaticRedirections = 5,
            KeepAlivePingDelay = TimeSpan.FromSeconds(30),
            KeepAlivePingTimeout = TimeSpan.FromSeconds(10),
            ResponseDrainTimeout = TimeSpan.FromSeconds(5)
        };

        var client = new HttpClient(handler)
        {
            Timeout = TimeSpan.FromMilliseconds(_config.SocketTimeoutMs)
        };

        client.DefaultRequestHeaders.Add("User-Agent", "HJTPX-Captcha-CSharp-SDK/1.0.0");
        client.DefaultRequestHeaders.Add("Accept", "application/json, text/plain, */*");
        client.DefaultRequestHeaders.Add("Accept-Encoding", "gzip, deflate, br");

        return client;
    }

    public HttpClient GetClient()
    {
        if (_disposed)
        {
            throw new ObjectDisposedException(nameof(ConnectionPool));
        }

        Interlocked.Increment(ref _totalRequests);

        if (_availableClients.TryTake(out var client))
        {
            _logger.LogDebug("Reusing existing connection. Available: {Available}, Total: {Total}", 
                _availableClients.Count, _allClients.Count);
            return client;
        }

        lock (_lockObject)
        {
            if (_allClients.Count < _config.MaxConnections)
            {
                var newClient = CreateHttpClient();
                _allClients.Add(newClient);
                _totalCreated++;
                _logger.LogDebug("Created new connection. Total: {Total}", _allClients.Count);
                return newClient;
            }
        }

        while (!_availableClients.TryTake(out client))
        {
            if (_disposed)
            {
                throw new ObjectDisposedException(nameof(ConnectionPool));
            }
            Thread.SpinWait(100);
        }

        return client;
    }

    public void ReturnClient(HttpClient client)
    {
        if (_disposed || client == null)
        {
            return;
        }

        if (!_allClients.Contains(client))
        {
            _logger.LogWarning("Attempted to return unknown client to pool");
            return;
        }

        if (_availableClients.Count < _config.MaxConnections)
        {
            _availableClients.Add(client);
            _logger.LogDebug("Returned connection to pool. Available: {Available}", _availableClients.Count);
        }
        else
        {
            client.Dispose();
            lock (_lockObject)
            {
                _allClients.Remove(client);
            }
            _logger.LogDebug("Pool full, disposed excess connection. Available: {Available}", _availableClients.Count);
        }
    }

    public PoolStatistics GetStatistics()
    {
        lock (_lockObject)
        {
            return new PoolStatistics
            {
                TotalConnections = _allClients.Count,
                ActiveConnections = _allClients.Count - _availableClients.Count,
                AvailableConnections = _availableClients.Count,
                ConnectionErrors = _connectionErrors,
                TotalRequests = _totalRequests,
                FailedRequests = _failedRequests,
                LastResetTime = _lastResetTime
            };
        }
    }

    public void RecordError()
    {
        Interlocked.Increment(ref _connectionErrors);
        Interlocked.Increment(ref _failedRequests);
    }

    public void Reset()
    {
        lock (_lockObject)
        {
            foreach (var client in _allClients)
            {
                client.Dispose();
            }
            _allClients.Clear();
            _availableClients.Clear();
            _totalCreated = 0;
            _connectionErrors = 0;
            _totalRequests = 0;
            _failedRequests = 0;
            _lastResetTime = DateTime.UtcNow;
        }

        PreWarmConnections();
        _logger.LogInformation("Connection pool reset and re-warmed");
    }

    public void Dispose()
    {
        Dispose(true);
        GC.SuppressFinalize(this);
    }

    protected virtual void Dispose(bool disposing)
    {
        if (_disposed)
        {
            return;
        }

        if (disposing)
        {
            lock (_lockObject)
            {
                foreach (var client in _allClients)
                {
                    client.Dispose();
                }
                _allClients.Clear();
                _availableClients.Clear();
            }
        }

        _disposed = true;
        _logger.LogInformation("Connection pool disposed");
    }
}

public class ManagedConnectionPool : IConnectionPool
{
    private readonly ConnectionPool _pool;
    private readonly ConcurrentDictionary<HttpClient, PooledClientState> _clientStates;
    private readonly TimeSpan _clientLifetime;
    private readonly Timer _cleanupTimer;

    private class PooledClientState
    {
        public DateTime CreatedAt { get; set; }
        public DateTime LastUsedAt { get; set; }
        public int UseCount { get; set; }
        public bool IsHealthy { get; set; }
    }

    public ManagedConnectionPool(ConnectionPoolConfig? config = null, ILogger<ConnectionPool>? logger = null)
    {
        _pool = new ConnectionPool(config, logger);
        _clientStates = new ConcurrentDictionary<HttpClient, PooledClientState>();
        _clientLifetime = TimeSpan.FromMinutes(5);
        _cleanupTimer = new Timer(CleanupInactiveClients, null, TimeSpan.FromMinutes(1), TimeSpan.FromMinutes(1));
    }

    public HttpClient GetClient()
    {
        var client = _pool.GetClient();
        
        var state = _clientStates.GetOrAdd(client, _ => new PooledClientState
        {
            CreatedAt = DateTime.UtcNow,
            LastUsedAt = DateTime.UtcNow,
            UseCount = 0,
            IsHealthy = true
        });

        state.LastUsedAt = DateTime.UtcNow;
        state.UseCount++;

        return client;
    }

    public void ReturnClient(HttpClient client)
    {
        if (_clientStates.TryGetValue(client, out var state))
        {
            if (!state.IsHealthy || DateTime.UtcNow - state.CreatedAt > _clientLifetime)
            {
                _pool.RecordError();
                state.IsHealthy = false;
            }
        }

        _pool.ReturnClient(client);
    }

    public PoolStatistics GetStatistics()
    {
        return _pool.GetStatistics();
    }

    public void Reset()
    {
        _clientStates.Clear();
        _pool.Reset();
    }

    private void CleanupInactiveClients(object? state)
    {
        var now = DateTime.UtcNow;
        var staleClients = _clientStates
            .Where(kvp => now - kvp.Value.LastUsedAt > _clientLifetime)
            .Select(kvp => kvp.Key)
            .ToList();

        foreach (var client in staleClients)
        {
            if (_clientStates.TryRemove(client, out var clientState))
            {
                clientState.IsHealthy = false;
                _pool.ReturnClient(client);
            }
        }

        if (staleClients.Count > 0)
        {
            _pool.GetStatistics(); 
        }
    }

    public void Dispose()
    {
        _cleanupTimer.Dispose();
        _clientStates.Clear();
        _pool.Dispose();
    }
}
