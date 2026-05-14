const winston = require('winston');

const logLevels = {
  error: 0,
  warn: 1,
  info: 2,
  http: 3,
  debug: 4
};

const logColors = {
  error: 'red',
  warn: 'yellow',
  info: 'green',
  http: 'magenta',
  debug: 'blue'
};

winston.addColors(logColors);

const format = winston.format.combine(
  winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
  winston.format.errors({ stack: true }),
  winston.format.splat(),
  winston.format.json()
);

const consoleFormat = winston.format.combine(
  winston.format.colorize({ all: true }),
  winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
  winston.format.printf(
    (info) => `${info.timestamp} ${info.level}: ${info.message}`
  )
);

const developFormat = winston.format.combine(
  winston.format.colorize({ all: true }),
  winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
  winston.format.printf((info) => {
    const { level, message, timestamp, ...meta } = info;
    const metaStr = Object.keys(meta).length ? JSON.stringify(meta, null, 2) : '';
    return `${timestamp} [${level}]: ${message} ${metaStr}`;
  })
);

const transports = [
  new winston.transports.Console({
    format: process.env.NODE_ENV === 'production' ? format : developFormat
  })
];

if (process.env.NODE_ENV === 'production') {
  transports.push(
    new winston.transports.File({
      filename: 'logs/error.log',
      level: 'error',
      maxsize: 5242880,
      maxFiles: 5
    }),
    new winston.transports.File({
      filename: 'logs/combined.log',
      maxsize: 5242880,
      maxFiles: 5
    }),
    new winston.transports.File({
      filename: 'logs/http.log',
      level: 'http',
      maxsize: 5242880,
      maxFiles: 5
    })
  );
}

const Logger = winston.createLogger({
  level: process.env.LOG_LEVEL || 'info',
  levels: logLevels,
  format,
  transports,
  exitOnError: false
});

const createChildLogger = (context) => {
  return Logger.child(context);
};

const logRequest = (req, res, duration) => {
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
    contentLength: res.get('content-length')
  };

  if (res.statusCode >= 500) {
    Logger.error('Request failed', logData);
  } else if (res.statusCode >= 400) {
    Logger.warn('Request warning', logData);
  } else {
    Logger.http('Request completed', logData);
  }
};

const logError = (error, req = null, context = {}) => {
  const logData = {
    requestId: req?.requestId,
    message: error.message,
    stack: process.env.NODE_ENV === 'development' ? error.stack : undefined,
    name: error.name,
    code: error.code,
    url: req?.originalUrl || req?.url,
    method: req?.method,
    ip: req?.ip || req?.connection?.remoteAddress,
    ...context
  };

  Logger.error('Error occurred', logData);
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

module.exports = {
  Logger,
  createChildLogger,
  logRequest,
  logError,
  logInfo,
  logWarn,
  logDebug
};
