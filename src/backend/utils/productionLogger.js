const winston = require('winston');
const DailyRotateFile = require('winston-daily-rotate-file');

const { combine, timestamp, printf, errors, json, colorize, splat } = winston.format;

const LOG_DIR = process.env.LOG_DIR || 'logs';
const NODE_ENV = process.env.NODE_ENV || 'development';

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

const logFormat = combine(
  errors({ stack: true }),
  splat(),
  timestamp({ format: 'YYYY-MM-DD HH:mm:ss.SSS' }),
  json()
);

const consoleFormat = combine(
  colorize({ all: true }),
  timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
  printf(info => {
    const { level, message, timestamp, ...meta } = info;
    const metaStr = Object.keys(meta).length ? `\n${JSON.stringify(meta, null, 2)}` : '';
    return `${timestamp} [${level}]: ${message}${metaStr}`;
  })
);

const createTransports = () => {
  const transports = [];

  if (NODE_ENV === 'production') {
    const maxSize = process.env.LOG_MAX_SIZE || '20m';
    const maxFiles = process.env.LOG_MAX_FILES || '30d';

    transports.push(
      new DailyRotateFile({
        filename: `${LOG_DIR}/error-%DATE%.log`,
        datePattern: 'YYYY-MM-DD',
        level: 'error',
        maxSize,
        maxFiles,
        format: logFormat,
        handleExceptions: true,
        handleRejections: true,
        zippedArchive: true,
        eol: '\n'
      })
    );

    transports.push(
      new DailyRotateFile({
        filename: `${LOG_DIR}/warn-%DATE%.log`,
        datePattern: 'YYYY-MM-DD',
        level: 'warn',
        maxSize,
        maxFiles,
        format: logFormat,
        zippedArchive: true,
        eol: '\n'
      })
    );

    transports.push(
      new DailyRotateFile({
        filename: `${LOG_DIR}/combined-%DATE%.log`,
        datePattern: 'YYYY-MM-DD',
        maxSize,
        maxFiles,
        format: logFormat,
        zippedArchive: true,
        eol: '\n'
      })
    );

    transports.push(
      new DailyRotateFile({
        filename: `${LOG_DIR}/http-%DATE%.log`,
        datePattern: 'YYYY-MM-DD',
        level: 'http',
        maxSize,
        maxFiles: '14d',
        format: logFormat,
        zippedArchive: true,
        eol: '\n'
      })
    );

    transports.push(
      new DailyRotateFile({
        filename: `${LOG_DIR}/performance-%DATE%.log`,
        datePattern: 'YYYY-MM-DD',
        level: 'performance',
        maxSize,
        maxFiles: '7d',
        format: logFormat,
        zippedArchive: true,
        eol: '\n'
      })
    );

    transports.push(
      new DailyRotateFile({
        filename: `${LOG_DIR}/security-%DATE%.log`,
        datePattern: 'YYYY-MM-DD',
        level: 'security',
        maxSize,
        maxFiles: '30d',
        format: logFormat,
        zippedArchive: true,
        eol: '\n'
      })
    );
  }

  if (NODE_ENV !== 'test') {
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
  level: process.env.LOG_LEVEL || 'info',
  levels: logLevels,
  format: logFormat,
  transports: createTransports(),
  exitOnError: false,
  silent: NODE_ENV === 'test',
  defaultMeta: {
    service: 'hjtpx-api',
    environment: NODE_ENV,
    version: process.env.APP_VERSION || '1.0.0'
  }
});

const createChildLogger = (context, meta = {}) => {
  return Logger.child(context, {
    ...meta,
    timestamp: new Date().toISOString()
  });
};

const logRequest = (req, res, duration, meta = {}) => {
  const logData = {
    requestId: req.requestId,
    method: req.method,
    url: req.originalUrl || req.url,
    path: req.path,
    query: req.query,
    params: req.params,
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
    ...context
  };

  Logger.error('Error occurred', logData);
};

const logSecurity = (event, data = {}) => {
  Logger.security(event, {
    ...data,
    timestamp: new Date().toISOString(),
    source: 'security-middleware'
  });
};

const logPerformance = (metric, data = {}) => {
  Logger.performance(metric, {
    ...data,
    timestamp: new Date().toISOString(),
    source: 'performance-monitor'
  });
};

const logInfo = (message, meta = {}) => {
  Logger.info(message, meta);
};

const logWarn = (message, meta = {}) => {
  Logger.warn(message, meta);
};

const logDebug = (message, meta = {}) => {
  Logger.debug(message, meta);
};

const logHttp = (message, meta = {}) => {
  Logger.http(message, meta);
};

const logPerformanceMetric = (name, value, unit, tags = {}) => {
  Logger.performance(`${name}`, {
    metric: name,
    value,
    unit,
    tags,
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
  logPerformanceMetric
};
