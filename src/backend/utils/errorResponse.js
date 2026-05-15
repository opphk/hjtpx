const { ErrorCode } = require('./errorCodes');
const { ErrorLogService } = require('./errorLogger');

const ErrorResponse = {
  format(error) {
    return {
      success: false,
      error: {
        code: error.code || 'UNKNOWN',
        message: error.message || 'An error occurred',
        timestamp: new Date().toISOString()
      }
    };
  },

  validation(message, details) {
    return {
      success: false,
      error: {
        code: ErrorCode.VALIDATION_ERRORS.VALIDATION_FAILED,
        message,
        details,
        timestamp: new Date().toISOString()
      }
    };
  },

  unauthorized(message = 'Unauthorized') {
    return {
      success: false,
      error: {
        code: ErrorCode.AUTHENTICATION_ERRORS.AUTH_007,
        message,
        timestamp: new Date().toISOString()
      }
    };
  },

  forbidden(message = 'Forbidden') {
    return {
      success: false,
      error: {
        code: ErrorCode.SECURITY_ERRORS.SEC_001,
        message,
        timestamp: new Date().toISOString()
      }
    };
  },

  notFound(resource = 'Resource') {
    return {
      success: false,
      error: {
        code: ErrorCode.DATABASE_ERRORS.RECORD_NOT_FOUND,
        message: `${resource} not found`,
        timestamp: new Date().toISOString()
      }
    };
  },

  server(message = 'Internal server error') {
    ErrorLogService.logWarning(message);
    return {
      success: false,
      error: {
        code: ErrorCode.SERVER_ERRORS.INTERNAL_ERROR,
        message: process.env.NODE_ENV === 'production' ? 'Internal server error' : message,
        timestamp: new Date().toISOString()
      }
    };
  },

  rateLimit(message = 'Too many requests') {
    return {
      success: false,
      error: {
        code: ErrorCode.SERVER_ERRORS.RATE_LIMIT_EXCEEDED,
        message,
        timestamp: new Date().toISOString()
      }
    };
  }
};

module.exports = ErrorResponse;
