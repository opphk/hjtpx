const promClient = require('prom-client');

const register = new promClient.Registry();

promClient.collectDefaultMetrics({ register });

const httpRequestDuration = new promClient.Histogram({
  name: 'http_request_duration_seconds',
  help: 'Duration of HTTP requests in seconds',
  labelNames: ['method', 'route', 'status_code'],
  buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10]
});

const httpRequestTotal = new promClient.Counter({
  name: 'http_requests_total',
  help: 'Total number of HTTP requests',
  labelNames: ['method', 'route', 'status_code']
});

const httpRequestErrors = new promClient.Counter({
  name: 'http_request_errors_total',
  help: 'Total number of HTTP request errors',
  labelNames: ['method', 'route', 'error_type']
});

const databaseQueryDuration = new promClient.Histogram({
  name: 'database_query_duration_seconds',
  help: 'Duration of database queries in seconds',
  labelNames: ['query_type', 'table'],
  buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5]
});

const databaseQueryErrors = new promClient.Counter({
  name: 'database_query_errors_total',
  help: 'Total number of database query errors',
  labelNames: ['query_type', 'error_code']
});

const redisOperationDuration = new promClient.Histogram({
  name: 'redis_operation_duration_seconds',
  help: 'Duration of Redis operations in seconds',
  labelNames: ['operation', 'status'],
  buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5]
});

const activeConnections = new promClient.Gauge({
  name: 'http_active_connections',
  help: 'Number of active HTTP connections'
});

const websocketConnections = new promClient.Gauge({
  name: 'websocket_connections_active',
  help: 'Number of active WebSocket connections'
});

const authenticationAttempts = new promClient.Counter({
  name: 'authentication_attempts_total',
  help: 'Total number of authentication attempts',
  labelNames: ['type', 'result']
});

const rateLimitHits = new promClient.Counter({
  name: 'rate_limit_hits_total',
  help: 'Total number of rate limit hits',
  labelNames: ['type']
});

const cacheHits = new promClient.Counter({
  name: 'cache_hits_total',
  help: 'Total number of cache hits',
  labelNames: ['cache_type', 'status']
});

const cacheMisses = new promClient.Counter({
  name: 'cache_misses_total',
  help: 'Total number of cache misses',
  labelNames: ['cache_type']
});

const businessMetrics = new promClient.Gauge({
  name: 'business_metric_current',
  help: 'Current value of business metrics',
  labelNames: ['metric_name']
});

const businessMetricEvents = new promClient.Counter({
  name: 'business_metric_events_total',
  help: 'Total number of business metric events',
  labelNames: ['event_type']
});

const memoryUsage = new promClient.Gauge({
  name: 'process_memory_bytes',
  help: 'Process memory usage in bytes',
  labelNames: ['type']
});

const cpuUsage = new promClient.Gauge({
  name: 'process_cpu_usage_percent',
  help: 'Process CPU usage percentage'
});

register.registerMetric(httpRequestDuration);
register.registerMetric(httpRequestTotal);
register.registerMetric(httpRequestErrors);
register.registerMetric(databaseQueryDuration);
register.registerMetric(databaseQueryErrors);
register.registerMetric(redisOperationDuration);
register.registerMetric(activeConnections);
register.registerMetric(websocketConnections);
register.registerMetric(authenticationAttempts);
register.registerMetric(rateLimitHits);
register.registerMetric(cacheHits);
register.registerMetric(cacheMisses);
register.registerMetric(businessMetrics);
register.registerMetric(businessMetricEvents);
register.registerMetric(memoryUsage);
register.registerMetric(cpuUsage);

const metricsMiddleware = (req, res, next) => {
  const start = process.hrtime.bigint();

  activeConnections.inc();

  res.on('finish', () => {
    activeConnections.dec();

    const end = process.hrtime.bigint();
    const durationInSeconds = Number(end - start) / 1e9;

    const route = req.route?.path || req.path || 'unknown';
    const labels = {
      method: req.method,
      route: route,
      status_code: res.statusCode
    };

    httpRequestDuration.observe(labels, durationInSeconds);
    httpRequestTotal.inc(labels);

    if (res.statusCode >= 400) {
      httpRequestErrors.inc({
        method: req.method,
        route: route,
        error_type: res.statusCode >= 500 ? 'server_error' : 'client_error'
      });
    }

    memoryUsage.set({ type: 'heap_used' }, process.memoryUsage().heapUsed);
    memoryUsage.set({ type: 'heap_total' }, process.memoryUsage().heapTotal);
    memoryUsage.set({ type: 'rss' }, process.memoryUsage().rss);
  });

  next();
};

const getMetrics = async () => {
  return await register.metrics();
};

const getContentType = () => {
  return register.contentType;
};

const recordHttpRequest = (method, route, statusCode, duration) => {
  httpRequestDuration.observe({ method, route, status_code: statusCode }, duration);
  httpRequestTotal.inc({ method, route, status_code: statusCode });
};

const recordDatabaseQuery = (queryType, table, duration, error = null) => {
  databaseQueryDuration.observe({ query_type: queryType, table }, duration);
  if (error) {
    databaseQueryErrors.inc({
      query_type: queryType,
      error_code: error.code || 'unknown'
    });
  }
};

const recordRedisOperation = (operation, status, duration) => {
  redisOperationDuration.observe({ operation, status }, duration);
};

const recordAuthAttempt = (type, result) => {
  authenticationAttempts.inc({ type, result });
};

const recordRateLimitHit = (type) => {
  rateLimitHits.inc({ type });
};

const recordCacheHit = (cacheType) => {
  cacheHits.inc({ cache_type: cacheType, status: 'hit' });
};

const recordCacheMiss = (cacheType) => {
  cacheMisses.inc({ cache_type: cacheType });
};

const setBusinessMetric = (name, value) => {
  businessMetrics.set({ metric_name: name }, value);
};

const recordBusinessEvent = (eventType) => {
  businessMetricEvents.inc({ event_type: eventType });
};

const updateConnectionMetrics = (wsCount) => {
  websocketConnections.set(wsCount);
};

const updateSystemMetrics = () => {
  const memUsage = process.memoryUsage();
  memoryUsage.set({ type: 'heap_used' }, memUsage.heapUsed);
  memoryUsage.set({ type: 'heap_total' }, memUsage.heapTotal);
  memoryUsage.set({ type: 'external' }, memUsage.external);
  memoryUsage.set({ type: 'rss' }, memUsage.rss);
};

module.exports = {
  register,
  metricsMiddleware,
  getMetrics,
  getContentType,
  recordHttpRequest,
  recordDatabaseQuery,
  recordRedisOperation,
  recordAuthAttempt,
  recordRateLimitHit,
  recordCacheHit,
  recordCacheMiss,
  setBusinessMetric,
  recordBusinessEvent,
  updateConnectionMetrics,
  updateSystemMetrics
};
