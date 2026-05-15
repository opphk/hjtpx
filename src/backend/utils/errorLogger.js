const winston = require('winston');
const path = require('path');
const fs = require('fs');

const errorLogDir = process.env.ERROR_LOG_DIR || path.join(process.cwd(), 'logs', 'errors');

if (!fs.existsSync(errorLogDir)) {
  fs.mkdirSync(errorLogDir, { recursive: true });
}

const errorLogger = winston.createLogger({
  level: 'error',
  format: winston.format.combine(
    winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
    winston.format.errors({ stack: true }),
    winston.format.json()
  ),
  defaultMeta: { service: 'hjtpx-api' },
  transports: [
    new winston.transports.File({
      filename: path.join(errorLogDir, 'error.log'),
      maxsize: 5242880,
      maxFiles: 10
    }),
    new winston.transports.File({
      filename: path.join(errorLogDir, 'combined.log'),
      maxsize: 5242880,
      maxFiles: 5
    })
  ]
});

if (process.env.NODE_ENV !== 'production') {
  errorLogger.add(new winston.transports.Console({
    format: winston.format.combine(
      winston.format.colorize(),
      winston.format.simple()
    )
  }));
}

class ErrorLogService {
  static log(error, context = {}) {
    const logEntry = {
      message: error.message,
      code: error.code || 'UNKNOWN',
      stack: error.stack,
      context: {
        ...context,
        timestamp: new Date().toISOString()
      }
    };

    errorLogger.error(logEntry);

    return logEntry;
  }

  static logWarning(message, context = {}) {
    errorLogger.warn({
      message,
      context: {
        ...context,
        timestamp: new Date().toISOString()
      }
    });
  }

  static async getErrorLogs(options = {}) {
    const { startDate, endDate, limit = 100, level = 'error' } = options;
    return [];
  }
}

module.exports = { errorLogger, ErrorLogService };
