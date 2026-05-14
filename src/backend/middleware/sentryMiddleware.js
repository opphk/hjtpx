const Sentry = require('@sentry/node');
const { captureException, captureMessage, addBreadcrumb, setContext } = require('../services/sentryService');

function sentryMiddleware() {
  return (err, req, res, next) => {
    addBreadcrumb(`${req.method} ${req.path}`, 'http', {
      status: res.statusCode,
      userId: req.user?.id,
    });

    setContext('request', {
      method: req.method,
      url: req.originalUrl,
      headers: {
        'user-agent': req.headers['user-agent'],
        'content-type': req.headers['content-type'],
      },
      query: req.query,
      params: req.params,
      requestId: req.requestId,
    });

    if (req.user) {
      setContext('user', {
        id: req.user.id,
        username: req.user.username,
        email: req.user.email,
        role: req.user.role,
      });
    }

    if (err) {
      const errorInfo = {
        user: req.user,
        request: req,
        context: 'Express middleware',
      };

      if (err.statusCode && err.statusCode < 500) {
        captureMessage(`${err.statusCode} Error: ${err.message}`, 'warning', {
          tags: { type: 'client-error' },
          extra: { statusCode: err.statusCode, path: req.path },
        });
      } else {
        captureException(err, errorInfo);
      }
    }

    next(err);
  };
}

function sentryRequestHandler() {
  return Sentry.Handlers.requestHandler({
    transaction: 'request',
    user: ['id', 'username', 'email'],
    request: ['method', 'url', 'headers', 'query'],
  });
}

function sentryErrorHandler() {
  return Sentry.Handlers.errorHandler({
    shouldHandleError(error) {
      if (error.statusCode && error.statusCode < 500) {
        return false;
      }
      return true;
    },
  });
}

function performanceMonitoringMiddleware() {
  return (req, res, next) => {
    const startTime = process.hrtime.bigint();

    res.on('finish', () => {
      const endTime = process.hrtime.bigint();
      const duration = Number(endTime - startTime) / 1e6;

      const transaction = Sentry.startTransaction({
        op: 'http',
        name: `${req.method} ${req.path}`,
      });

      transaction.setTag('http.method', req.method);
      transaction.setTag('http.status_code', res.statusCode);
      transaction.setTag('http.url', req.originalUrl);

      if (duration > 1000) {
        transaction.setTag('performance', 'slow');
        captureMessage(`Slow request detected: ${req.method} ${req.path}`, 'warning', {
          tags: { type: 'slow-request' },
          extra: { duration, statusCode: res.statusCode },
        });
      }

      transaction.finish();

      addBreadcrumb(`${req.method} ${req.path} completed`, 'performance', {
        duration: `${duration.toFixed(2)}ms`,
        statusCode: res.statusCode,
      });
    });

    next();
  };
}

module.exports = {
  sentryMiddleware,
  sentryRequestHandler,
  sentryErrorHandler,
  performanceMonitoringMiddleware,
};
