const sentryService = require('../services/sentryService');

const sentryMiddleware = {
  errorHandler(err, req, res, next) {
    const errorContext = {
      userId: req.user?.id,
      requestId: req.requestId,
      endpoint: req.path,
      method: req.method,
      context: 'Express Error Handler Middleware',
      tags: {
        error_type: err.name || 'Error',
        error_code: err.code || 'UNKNOWN',
        http_status: err.statusCode || 500,
        user_agent: req.get('User-Agent'),
        referer: req.get('Referer'),
        origin: req.get('Origin'),
      },
      extra: {
        query: req.query,
        params: req.params,
        body: req.body && Object.keys(req.body).length > 0 ? req.body : undefined,
        ip: req.ip || req.connection?.remoteAddress,
        hostname: req.hostname,
        protocol: req.protocol,
        url: req.originalUrl,
        headers: {
          contentType: req.get('Content-Type'),
          accept: req.get('Accept'),
          authorization: req.headers.authorization ? '[PRESENT]' : '[ABSENT]',
        },
      },
    };

    sentryService.captureException(err, errorContext);

    if (err.statusCode && err.statusCode < 500) {
      return next(err);
    }

    next(err);
  },

  requestHandler() {
    return (req, res, next) => {
      const startTime = Date.now();
      
      sentryService.addBreadcrumb(
        `Incoming request: ${req.method} ${req.path}`,
        'http',
        {
          method: req.method,
          url: req.path,
          query: req.query,
        },
        'info'
      );

      const transaction = sentryService.startTransaction(
        `${req.method} ${req.path}`,
        'http.server',
        {
          description: `${req.method} ${req.path}`,
          data: {
            method: req.method,
            url: req.originalUrl,
            query: req.query,
          },
        }
      );

      transaction.setTag('http.method', req.method);
      transaction.setTag('http.url', req.hostname);
      transaction.setTag('request.id', req.requestId);

      if (req.user?.id) {
        transaction.setTag('user.id', String(req.user.id));
      }

      res.on('finish', () => {
        transaction.setTag('http.status_code', res.statusCode);
        transaction.setData('response_time', Date.now() - startTime);
        transaction.setData('response_size', res.get('Content-Length') || 0);
        
        if (res.statusCode >= 400) {
          transaction.setStatus('server_error');
        } else {
          transaction.setStatus('ok');
        }
        
        transaction.finish();
      });

      next();
    };
  },

  tracingMiddleware() {
    return (req, res, next) => {
      const startSpan = sentryService.startTransaction(
        `Express: ${req.method} ${req.path}`,
        'middleware',
        { description: 'Request processing' }
      );

      startSpan.setTag('middleware.type', 'tracing');
      startSpan.setTag('middleware.name', 'sentry.tracing');

      const originalEnd = res.end;
      const startTime = Date.now();

      res.end = function(...args) {
        const duration = Date.now() - startTime;
        
        startSpan.setData('response.duration_ms', duration);
        startSpan.setData('response.status_code', res.statusCode);
        startSpan.setStatus(res.statusCode >= 400 ? 'server_error' : 'ok');
        
        startSpan.finish();

        if (duration > 1000) {
          sentryService.captureMessage(
            `Slow request detected: ${req.method} ${req.path} took ${duration}ms`,
            'warning',
            {
              tags: {
                slow_request: true,
                duration_ms: duration,
                threshold_ms: 1000,
              },
              extra: {
                method: req.method,
                path: req.path,
                duration_ms: duration,
              },
            }
          );
        }

        return originalEnd.apply(this, args);
      };

      next();
    };
  },

  performanceMonitor() {
    return (req, res, next) => {
      const startMemory = process.memoryUsage();
      const startTime = Date.now();

      res.on('finish', () => {
        const duration = Date.now() - startTime;
        const endMemory = process.memoryUsage();
        const memoryDelta = {
          heapUsed: endMemory.heapUsed - startMemory.heapUsed,
          heapTotal: endMemory.heapTotal - startMemory.heapTotal,
          rss: endMemory.rss - startMemory.rss,
        };

        if (duration > 2000 || memoryDelta.heapUsed > 50 * 1024 * 1024) {
          sentryService.captureMessage(
            'Performance threshold exceeded',
            'warning',
            {
              tags: {
                performance_alert: true,
                slow_endpoint: duration > 2000,
                memory_leak_suspect: memoryDelta.heapUsed > 50 * 1024 * 1024,
              },
              extra: {
                duration_ms: duration,
                memory_delta_bytes: memoryDelta,
                memory_end: endMemory,
                endpoint: req.path,
                method: req.method,
              },
            }
          );
        }
      });

      next();
    };
  },
};

module.exports = sentryMiddleware;
