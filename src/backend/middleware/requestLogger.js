const { logRequest } = require('../utils/logger');

const requestLogger = (req, res, next) => {
  const startTime = Date.now();
  const requestId = generateRequestId();

  req.requestId = requestId;

  if (process.env.NODE_ENV === 'development') {
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

const generateRequestId = () => {
  return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
};

const sensitiveDataMasker = (data, sensitiveFields = ['password', 'token', 'authorization', 'cookie', 'secret']) => {
  if (!data || typeof data !== 'object') {
    return data;
  }

  const masked = { ...data };

  for (const field of sensitiveFields) {
    if (masked[field]) {
      masked[field] = '***MASKED***';
    }
  }

  return masked;
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

const logSlowRequests = (threshold = 3000) => {
  return (req, res, next) => {
    const startTime = Date.now();

    res.on('finish', () => {
      const duration = Date.now() - startTime;

      if (duration > threshold) {
        console.warn(
          JSON.stringify({
            level: 'warn',
            type: 'slow_request',
            requestId: req.requestId,
            method: req.method,
            path: req.path,
            duration: `${duration}ms`,
            threshold: `${threshold}ms`
          })
        );
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
