const { logRequest, generateRequestId, sensitiveDataMasker, logWarn } = require('../utils/logger');
const loggingConfig = require('../config/logging').logging;

const requestLogger = (req, res, next) => {
  const startTime = Date.now();
  const requestId = generateRequestId();
  const traceId = req.headers['x-trace-id'] || requestId;

  req.requestId = requestId;
  req.traceId = traceId;

  if (loggingConfig.environment === 'development') {
    console.log(`\n[${requestId}] --> ${req.method} ${req.path}`);
  }

  const originalEnd = res.end;
  res.end = function (chunk, encoding) {
    const duration = Date.now() - startTime;
    res.end = originalEnd;
    res.end(chunk, encoding);
    logRequest(req, res, duration);
  };

  res.on('close', () => {
    if (!res.writableFinished) {
      const duration = Date.now() - startTime;
      logRequest(req, res, duration);
    }
  });

  next();
};

const logRequestBody = (req, maxLength = 1000) => {
  if (!req.body || Object.keys(req.body).length === 0) {
    return null;
  }

  const maskedBody = sensitiveDataMasker(req.body);
  const bodyString = JSON.stringify(maskedBody);

  if (bodyString.length > maxLength) {
    return bodyString.substring(0, maxLength) + '... [TRUNCATED]';
  }

  return bodyString;
};

const logSlowRequests = (threshold = loggingConfig.slowRequestThreshold || 3000) => {
  return (req, res, next) => {
    const startTime = Date.now();

    res.on('finish', () => {
      const duration = Date.now() - startTime;

      if (duration > threshold) {
        logWarn('Slow request detected', {
          type: 'slow_request',
          requestId: req.requestId,
          traceId: req.traceId,
          method: req.method,
          path: req.path,
          duration: `${duration}ms`,
          threshold: `${threshold}ms`
        });
      }
    });

    next();
  };
};

module.exports = {
  requestLogger,
  sensitiveDataMasker,
  logRequestBody,
  logSlowRequests
};
