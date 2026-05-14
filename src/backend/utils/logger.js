const winston = require('winston');
const { v4: uuidv4 } = require('uuid');
const fs = require('fs');
const path = require('path');

const LOG_DIR = process.env.LOG_DIR || path.join(__dirname, '../../../logs');
const NODE_ENV = process.env.NODE_ENV || 'development';
const LOG_LEVEL = process.env.LOG_LEVEL || (NODE_ENV === 'production' ? 'info' : 'debug');

if (!fs.existsSync(LOG_DIR)) {
  fs.mkdirSync(LOG_DIR, { recursive: true });
}

const sensitiveFields = [
  'password',
  'passwordConfirm',
  'currentPassword',
  'newPassword',
  'token',
  'accessToken',
  'refreshToken',
  'apiKey',
  'secret',
  'privateKey',
  'authorization',
  'cookie',
  'set-cookie',
  'x-api-key',
  'ssn',
  'creditCard',
  'cardNumber',
  'cvv'
];

const sensitivePatterns = [
  /bearer\s+[a-zA-Z0-9\-_.]+/gi,
  /token["\s:=]+[a-zA-Z0-9\-_.]+/gi,
  /password["\s:=]+[^\s&]+/gi,
  /api[_-]?key["\s:=]+[^\s&]+/gi,
  /jwt["\s:=]+[a-zA-Z0-9\-_.]+/gi
];

function filterSensitiveData(data) {
  if (typeof data !== 'object' || data === null) {
    return data;
  }

  if (Array.isArray(data)) {
    return data.map(item => filterSensitiveData(item));
  }

  const filtered = {};

  for (const [key, value] of Object.entries(data)) {
    const lowerKey = key.toLowerCase();

    if (sensitiveFields.some(field => lowerKey.includes(field.toLowerCase()))) {
      filtered[key] = '[REDACTED]';
    } else if (typeof value === 'string') {
      let filteredValue = value;
      for (const pattern of sensitivePatterns) {
        filteredValue = filteredValue.replace(pattern, match => {
          const parts = match.split(/[:=]/);
          return parts.length > 1 ? `${parts[0]}: [REDACTED]` : '[REDACTED]';
        });
      }
      filtered[key] = filteredValue;
    } else {
      filtered[key] = filterSensitiveData(value);
    }
  }

  return filtered;
}

function sanitizeStackTrace(stack) {
  if (!stack) return undefined;

  return stack
    .split('\n')
    .filter(line => !line.includes('node_modules/') || process.env.NODE_ENV === 'development')
    .join('\n');
}

const logLevels = {
  error: 0,
  warn: 1,
  info: 2,
  http: 3,
  debug: 4,
  trace: 5
};

const logColors = {
  error: 'red',
  warn: 'yellow',
  info: 'green',
  http: 'magenta',
  debug: 'blue',
  trace: 'gray'
};

winston.addColors(logColors);

const jsonFormat = winston.format.combine(
  winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss.SSS' }),
  winston.format.errors({ stack: true }),
  winston.format((info) => {
    info.level = info.level.toUpperCase();
    return info;
  })(),
  winston.format.json()
);

const consoleFormat = winston.format.combine(
  winston.format.colorize({ all: true }),
  winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
  winston.format.printf(info => {
    const { level, message, timestamp, ...meta } = info;
    const metaStr = Object.keys(meta).length ? `\n${JSON.stringify(meta, null, 2)}` : '';
    return `${timestamp} [${level}]: ${message}${metaStr}`;
  })
);

const devFormat = winston.format.combine(
  winston.format.colorize({ all: true }),
  winston.format.timestamp({ format: 'HH:mm:ss' }),
  winston.format.printf(info => {
    const { level, message, timestamp, ...meta } = info;
    const metaStr = Object.keys(meta).length ? ` ${JSON.stringify(meta)}` : '';
    return `[${timestamp}] ${level}: ${message}${metaStr}`;
  })
);

const structuredFormat = winston.format.combine(
  winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss.SSS' }),
  winston.format.errors({ stack: true }),
  winston.format((info) => {
    const enriched = {
      '@timestamp': info.timestamp,
      '@version': '1',
      level: info.level.toUpperCase(),
      message: info.message,
      service: process.env.SERVICE_NAME || 'hjtpx-api',
      environment: NODE_ENV,
      hostname: process.env.HOSTNAME || require('os').hostname(),
      pid: process.pid
    };

    const { timestamp, level, message, ...rest } = info;
    Object.assign(enriched, filterSensitiveData(rest));

    return enriched;
  })(),
  winston.format.json()
);

const fileTransports = [];

fileTransports.push(
  new winston.transports.File({
    filename: path.join(LOG_DIR, 'error.log'),
    level: 'error',
    format: structuredFormat,
    maxsize: parseInt(process.env.LOG_MAX_SIZE) || 10485760,
    maxFiles: parseInt(process.env.LOG_MAX_FILES) || 5,
    tailable: true,
    zippedArchive: true
  })
);

fileTransports.push(
  new winston.transports.File({
    filename: path.join(LOG_DIR, 'warn.log'),
    level: 'warn',
    format: structuredFormat,
    maxsize: parseInt(process.env.LOG_MAX_SIZE) || 10485760,
    maxFiles: parseInt(process.env.LOG_MAX_FILES) || 5,
    tailable: true,
    zippedArchive: true
  })
);

fileTransports.push(
  new winston.transports.File({
    filename: path.join(LOG_DIR, 'combined.log'),
    format: structuredFormat,
    maxsize: parseInt(process.env.LOG_MAX_SIZE) || 10485760,
    maxFiles: parseInt(process.env.LOG_MAX_FILES) || 10,
    tailable: true,
    zippedArchive: true
  })
);

fileTransports.push(
  new winston.transports.File({
    filename: path.join(LOG_DIR, 'http.log'),
    level: 'http',
    format: structuredFormat,
    maxsize: parseInt(process.env.LOG_MAX_SIZE) || 10485760,
    maxFiles: parseInt(process.env.LOG_MAX_FILES) || 5,
    tailable: true,
    zippedArchive: true
  })
);

fileTransports.push(
  new winston.transports.File({
    filename: path.join(LOG_DIR, 'debug.log'),
    level: 'debug',
    format: structuredFormat,
    maxsize: parseInt(process.env.LOG_MAX_SIZE) || 10485760,
    maxFiles: parseInt(process.env.LOG_MAX_FILES) || 3,
    tailable: true,
    zippedArchive: true
  })
);

if (NODE_ENV === 'development') {
  fileTransports.push(
    new winston.transports.File({
      filename: path.join(LOG_DIR, 'development.log'),
      format: devFormat,
      maxsize: parseInt(process.env.LOG_MAX_SIZE) || 10485760,
      maxFiles: 3,
      tailable: true,
      zippedArchive: false
    })
  );
}

const transports = [
  new winston.transports.Console({
    format: NODE_ENV === 'production' ? jsonFormat : devFormat,
    silent: process.env.LOG_SILENT === 'true'
  })
];

transports.push(...fileTransports);

const Logger = winston.createLogger({
  level: LOG_LEVEL,
  levels: logLevels,
  format: structuredFormat,
  transports,
  exitOnError: false,
  handleExceptions: true,
  handleRejections: true
});

Logger.on('error', (err) => {
  console.error('Logger error:', err);
});

function generateRequestId() {
  return `req_${Date.now()}_${uuidv4().split('-')[0]}`;
}

function generateTraceId() {
  return uuidv4();
}

function generateCorrelationId() {
  return `corr_${Date.now()}_${uuidv4().split('-')[0]}`;
}

const createChildLogger = (context = {}) => {
  return Logger.child(context);
};

const createRequestLogger = (req) => {
  const requestId = req.requestId || generateRequestId();
  req.requestId = requestId;

  return {
    logRequest: (res, duration, additionalData = {}) => {
      const logData = filterSensitiveData({
        requestId,
        traceId: req.traceId,
        correlationId: req.correlationId,
        method: req.method,
        url: req.originalUrl || req.url,
        path: req.path,
        query: req.query,
        params: req.params,
        headers: {
          'user-agent': req.get('user-agent'),
          'content-type': req.get('content-type'),
          'x-forwarded-for': req.get('x-forwarded-for'),
          'x-real-ip': req.get('x-real-ip')
        },
        ip: req.ip || req.connection?.remoteAddress,
        userId: req.user?.id,
        userAgent: req.get('user-agent'),
        duration: `${duration}ms`,
        durationMs: duration,
        statusCode: res.statusCode,
        contentLength: res.get('content-length'),
        responseTime: parseFloat(res.get('x-response-time')) || duration,
        ...additionalData
      });

      if (res.statusCode >= 500) {
        Logger.error('Request failed', logData);
      } else if (res.statusCode >= 400) {
        Logger.warn('Request warning', logData);
      } else {
        Logger.http('Request completed', logData);
      }
    },

    logError: (error, additionalData = {}) => {
      const logData = filterSensitiveData({
        requestId,
        traceId: req.traceId,
        correlationId: req.correlationId,
        message: error.message,
        stack: NODE_ENV === 'development' ? sanitizeStackTrace(error.stack) : undefined,
        name: error.name,
        code: error.code,
        url: req.originalUrl || req.url,
        method: req.method,
        ip: req.ip || req.connection?.remoteAddress,
        userId: req.user?.id,
        ...additionalData
      });

      Logger.error('Request error', logData);
    },

    logInfo: (message, data = {}) => {
      Logger.info(message, filterSensitiveData({
        requestId,
        traceId: req.traceId,
        correlationId: req.correlationId,
        ...data
      }));
    },

    logDebug: (message, data = {}) => {
      Logger.debug(message, filterSensitiveData({
        requestId,
        traceId: req.traceId,
        correlationId: req.correlationId,
        ...data
      }));
    },

    logWarn: (message, data = {}) => {
      Logger.warn(message, filterSensitiveData({
        requestId,
        traceId: req.traceId,
        correlationId: req.correlationId,
        ...data
      }));
    }
  };
};

const logRequest = (req, res, duration) => {
  const requestId = req.requestId || generateRequestId();
  req.requestId = requestId;

  const logData = filterSensitiveData({
    requestId,
    traceId: req.traceId,
    correlationId: req.correlationId,
    method: req.method,
    url: req.originalUrl || req.url,
    path: req.path,
    query: req.query,
    params: req.params,
    headers: {
      'user-agent': req.get('user-agent'),
      'content-type': req.get('content-type'),
      'x-forwarded-for': req.get('x-forwarded-for'),
      'x-real-ip': req.get('x-real-ip')
    },
    ip: req.ip || req.connection?.remoteAddress,
    userId: req.user?.id,
    userAgent: req.get('user-agent'),
    duration: `${duration}ms`,
    durationMs: duration,
    statusCode: res.statusCode,
    contentLength: res.get('content-length'),
    ...res.metadata
  });

  if (res.statusCode >= 500) {
    Logger.error('Request failed', logData);
  } else if (res.statusCode >= 400) {
    Logger.warn('Request warning', logData);
  } else {
    Logger.http('Request completed', logData);
  }
};

const logError = (error, req = null, context = {}) => {
  const logData = filterSensitiveData({
    requestId: req?.requestId,
    traceId: req?.traceId,
    correlationId: req?.correlationId,
    message: error.message,
    stack: NODE_ENV === 'development' ? sanitizeStackTrace(error.stack) : undefined,
    name: error.name,
    code: error.code,
    url: req?.originalUrl || req?.url,
    method: req?.method,
    ip: req?.ip || req?.connection?.remoteAddress,
    ...context
  });

  Logger.error('Error occurred', logData);
};

const logInfo = (message, meta = {}) => {
  Logger.info(message, filterSensitiveData(meta));
};

const logWarn = (message, meta = {}) => {
  Logger.warn(message, filterSensitiveData(meta));
};

const logDebug = (message, meta = {}) => {
  Logger.debug(message, filterSensitiveData(meta));
};

const logTrace = (message, meta = {}) => {
  Logger.log('trace', message, filterSensitiveData(meta));
};

const logHttp = (message, meta = {}) => {
  Logger.http(message, filterSensitiveData(meta));
};

function createTransactionLogger(transactionId) {
  return {
    transactionId,
    log: (message, data = {}) => {
      Logger.info(`[Transaction: ${transactionId}] ${message}`, filterSensitiveData({
        transactionId,
        ...data
      }));
    },
    error: (message, data = {}) => {
      Logger.error(`[Transaction: ${transactionId}] ${message}`, filterSensitiveData({
        transactionId,
        ...data
      }));
    }
  };
}

function createComponentLogger(componentName) {
  return Logger.child({ component: componentName });
}

module.exports = {
  Logger,
  createChildLogger,
  createRequestLogger,
  createTransactionLogger,
  createComponentLogger,
  logRequest,
  logError,
  logInfo,
  logWarn,
  logDebug,
  logTrace,
  logHttp,
  generateRequestId,
  generateTraceId,
  generateCorrelationId,
  filterSensitiveData,
  sanitizeStackTrace,
  sensitiveFields
};
