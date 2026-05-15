const { sensitiveDataMasker } = require('../utils/logger');
const loggingConfig = require('../config/logging').logging;

const SENSITIVE_FIELDS = loggingConfig.sensitiveFields;

const sanitizeLogsMiddleware = (req, res, next) => {
  if (req.body && typeof req.body === 'object') {
    req.body = sensitiveDataMasker(req.body, SENSITIVE_FIELDS);
  }

  if (req.query && typeof req.query === 'object') {
    req.query = sensitiveDataMasker(req.query, SENSITIVE_FIELDS);
  }

  if (req.params && typeof req.params === 'object') {
    req.params = sensitiveDataMasker(req.params, SENSITIVE_FIELDS);
  }

  if (req.headers && typeof req.headers === 'object') {
    const sanitizedHeaders = {};
    for (const [key, value] of Object.entries(req.headers)) {
      const lowerKey = key.toLowerCase();
      const isSensitive = SENSITIVE_FIELDS.some(field => 
        lowerKey.includes(field.toLowerCase())
      );
      
      if (isSensitive) {
        sanitizedHeaders[key] = '***MASKED***';
      } else {
        sanitizedHeaders[key] = value;
      }
    }
    req.headers = sanitizedHeaders;
  }

  if (req.cookies && typeof req.cookies === 'object') {
    req.cookies = sensitiveDataMasker(req.cookies, SENSITIVE_FIELDS);
  }

  next();
};

module.exports = sanitizeLogsMiddleware;
