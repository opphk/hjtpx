const winston = require('winston');
const DailyRotateFile = require('winston-daily-rotate-file');
const { v4: uuidv4 } = require('uuid');
const loggingConfig = require('../config/logging').logging;

const { combine, timestamp, printf, errors, json, colorize, splat } = winston.format;

const LOG_DIR = loggingConfig.logDir;
const NODE_ENV = loggingConfig.environment;
const SERVICE_NAME = loggingConfig.serviceName;
const APP_VERSION = loggingConfig.appVersion;

const logLevels = {
  error: 0,
  warn: 1,
  info: 2,
  http: 3,
  debug: 4,
  performance: 5,
  security: 6
};

const logColors = {
  error: 'red',
  warn: 'yellow',
  info: 'green',
  http: 'magenta',
  debug: 'blue',
  performance: 'cyan',
  security: 'red bold'
};

winston.addColors(logColors);

const sensitiveDataMasker = (data, sensitiveFields = loggingConfig.sensitiveFields) => {
  if (!data || typeof data !== 'object') {
    return data;
  }

  if (Array.isArray(data)) {
    return data.map(item => sensitiveDataMasker(item, sensitiveFields));
  }

  const masked = {};

  for (const [key, value] of Object.entries(data)) {
    const lowerKey = key.toLowerCase();
    const isSensitive = sensitiveFields.some(field => 
      lowerKey.includes(field.toLowerCase())
    );

    if (isSensitive) {
      masked[key] = '***MASKED***';
    } else if (typeof value === 'object' && value !== null) {
      masked[key] = sensitiveDataMasker(value, sensitiveFields);
    } else {
      masked[key] = value;
    }
  }

  return masked;
};

const maskSensitiveData = winston.format(info => {
  if (info.meta) {
    info.meta = sensitiveDataMasker(info.meta);
  }
  if (info.query) {
    info.query = sensitiveDataMasker(info.query);
  }
  if (info.body) {
    info.body = sensitiveDataMasker(info.body);
  }
  if (info.headers) {
    info.headers = sensitiveDataMasker(info.headers);
  }
  return info;
});

const logFormat = combine(
  errors({ stack: true }),
  splat(),
  timestamp({ format: 'YYYY-MM-DD HH:mm:ss.SSS' }),
  maskSensitiveData(),
  json()
);

const consoleFormat = combine(
  colorize({ all: true }),
  timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
  printf(info => {
    const { level, message, timestamp, ...meta } = info;
    const maskedMeta = sensitiveDataMasker(meta);
    const metaStr = Object.keys(maskedMeta).length ? `\n${JSON.stringify(maskedMeta, null, 2)}` : '';
    return `${timestamp} [${level}]: ${message}${metaStr}`;
  })
);

const createTransports = () => {
  const transports = [];

  if (loggingConfig.enableFile) {
    const maxSize = loggingConfig.maxFileSize;
    const maxFiles = loggingConfig.maxFiles;
    const zippedArchive = loggingConfig.zippedArchive;

    transports.push(
      new DailyRotateFile({
        filename: `${LOG_DIR}/error-%DATE%.log`,
        datePattern: loggingConfig.datePattern,
        level: 'error',
        maxSize,
        maxFiles,
        format: logFormat,
        handleExceptions: true,
        handleRejections: true,
        zippedArchive,
        eol: '\n'
      })
    );

    transports.push(
      new DailyRotateFile({
        filename: `${LOG_DIR}/warn-%DATE%.log`,
        datePattern: loggingConfig.datePattern,
        level: 'warn',
        maxSize,
        maxFiles,
        format: logFormat,
        zippedArchive,
        eol: '\n'
      })
    );

    transports.push(
      new DailyRotateFile({
        filename: `${LOG_DIR}/combined-%DATE%.log`,
        datePattern: loggingConfig.datePattern,
        maxSize,
        maxFiles,
        format: logFormat,
        zippedArchive,
        eol: '\n'
      })
    );

    transports.push(
      new DailyRotateFile({
        filename: `${LOG_DIR}/http-%DATE%.log`,
        datePattern: loggingConfig.datePattern,
        level: 'http',
        maxSize,
        maxFiles: '14d',
        format: logFormat,
        zippedArchive,
        eol: '\n'
      })
    );

    transports.push(
      new DailyRotateFile({
        filename: `${LOG_DIR}/performance-%DATE%.log`,
        datePattern: loggingConfig.datePattern,
        level: 'performance',
        maxSize,
        maxFiles: '7d',
        format: logFormat,
        zippedArchive,
        eol: '\n'
      })
    );

    transports.push(
      new DailyRotateFile({
        filename: `${LOG_DIR}/security-%DATE%.log`,
        datePattern: loggingConfig.datePattern,
        level: 'security',
        maxSize,
        maxFiles: '30d',
        format: logFormat,
        zippedArchive,
        eol: '\n'
      })
    );
  }

  if (loggingConfig.enableConsole && NODE_ENV !== 'test') {
    transports.push(
      new winston.transports.Console({
        format: NODE_ENV === 'production' ? logFormat : consoleFormat,
        handleExceptions: true,
        handleRejections: true
      })
    );
  }

  return transports;
};

const Logger = winston.createLogger({
  level: loggingConfig.level,
  levels: logLevels,
  format: logFormat,
  transports: createTransports(),
  exitOnError: false,
  silent: NODE_ENV === 'test',
  defaultMeta: {
    service: SERVICE_NAME,
    environment: NODE_ENV,
    version: APP_VERSION,
    timestamp: new Date().toISOString()
  }
});

const generateRequestId = () => {
  return `req_${Date.now()}_${uuidv4().substr(0, 8)}`;
};

const createChildLogger = (context, meta = {}) => {
  return Logger.child({
    ...context,
    ...meta,
    timestamp: new Date().toISOString()
  });
};

const logRequest = (req, res, duration, meta = {}) => {
  const logData = {
    requestId: req.requestId,
    traceId: req.traceId || req.requestId,
    method: req.method,
    url: req.originalUrl || req.url,
    path: req.path,
    query: sensitiveDataMasker(req.query),
    params: sensitiveDataMasker(req.params),
    body: sensitiveDataMasker(req.body),
    headers: sensitiveDataMasker(
      Object.fromEntries(
        Object.entries(req.headers || {}).filter(([key]) => 
          !['authorization', 'cookie'].includes(key.toLowerCase())
        )
      )
    ),
    ip: req.ip || req.connection?.remoteAddress,
    userAgent: req.get('user-agent'),
    userId: req.user?.id,
    duration: `${duration}ms`,
    statusCode: res.statusCode,
    contentLength: res.get('content-length'),
    referer: req.get('referer'),
    ...meta
  };

  if (res.statusCode >= 500) {
    Logger.error('Request failed', logData);
  } else if (res.statusCode >= 400) {
    Logger.warn('Request warning', logData);
  } else if (duration > 1000) {
    Logger.performance('Slow request', logData);
  } else {
    Logger.http('Request completed', logData);
  }
};

const logError = (error, req = null, context = {}) => {
  const logData = {
    requestId: req?.requestId,
    traceId: req?.traceId || req?.requestId,
    message: error.message,
    stack: NODE_ENV === 'development' ? error.stack : undefined,
    name: error.name,
    code: error.code,
    statusCode: error.statusCode,
    url: req?.originalUrl || req?.url,
    method: req?.method,
    ip: req?.ip || req?.connection?.remoteAddress,
    userAgent: req?.get?.('user-agent'),
    userId: req?.user?.id,
    ...sensitiveDataMasker(context)
  };

  Logger.error('Error occurred', logData);
};

const logSecurity = (event, data = {}) => {
  Logger.security(event, {
    ...sensitiveDataMasker(data),
    timestamp: new Date().toISOString(),
    source: 'security-middleware'
  });
};

const logPerformance = (metric, data = {}) => {
  Logger.performance(metric, {
    ...sensitiveDataMasker(data),
    timestamp: new Date().toISOString(),
    source: 'performance-monitor'
  });
};

const logInfo = (message, meta = {}) => {
  Logger.info(message, sensitiveDataMasker(meta));
};

const logWarn = (message, meta = {}) => {
  Logger.warn(message, sensitiveDataMasker(meta));
};

const logDebug = (message, meta = {}) => {
  Logger.debug(message, sensitiveDataMasker(meta));
};

const logHttp = (message, meta = {}) => {
  Logger.http(message, sensitiveDataMasker(meta));
};

const logPerformanceMetric = (name, value, unit, tags = {}) => {
  Logger.performance(`${name}`, {
    metric: name,
    value,
    unit,
    tags: sensitiveDataMasker(tags),
    timestamp: new Date().toISOString()
  });
};

module.exports = {
  Logger,
  createChildLogger,
  logRequest,
  logError,
  logSecurity,
  logPerformance,
  logInfo,
  logWarn,
  logDebug,
  logHttp,
  logPerformanceMetric,
  generateRequestId,
  sensitiveDataMasker
};
