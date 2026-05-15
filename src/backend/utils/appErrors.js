const ErrorCode = require('./errorCodes');

class AppError extends Error {
  constructor(code, message, statusCode = null, details = null) {
    super(message);
    this.code = code;
    this.statusCode = statusCode || ErrorCode.getHttpStatus(code);
    this.details = details;
    this.timestamp = new Date().toISOString();
    this.isOperational = true;

    Error.captureStackTrace(this, this.constructor);
  }

  toJSON() {
    return {
      success: false,
      error: {
        code: this.code,
        message: this.message,
        category: ErrorCode.getCategory(this.code),
        statusCode: this.statusCode,
        details: this.details,
        timestamp: this.timestamp,
        stack: process.env.NODE_ENV === 'development' ? this.stack : undefined
      }
    };
  }

  toResponse() {
    return {
      success: false,
      error: {
        code: this.code,
        message: this.message,
        details: this.details
      }
    };
  }
}

class AuthenticationError extends AppError {
  constructor(code = 'AUTH_001', message = 'Authentication failed', details = null) {
    super(code, message, 401, details);
  }
}

class ValidationError extends AppError {
  constructor(message = 'Validation failed', details = null) {
    super('VAL_001', message, 400, details);
  }
}

class NotFoundError extends AppError {
  constructor(resource = 'Resource', details = null) {
    super('DB_004', `${resource} not found`, 404, details);
  }
}

class DatabaseError extends AppError {
  constructor(message = 'Database operation failed', details = null) {
    super('DB_002', message, 500, details);
  }
}

class SecurityError extends AppError {
  constructor(code = 'SEC_001', message = 'Security violation detected', details = null) {
    super(code, message, 403, details);
  }
}

class RateLimitError extends AppError {
  constructor(message = 'Rate limit exceeded', details = null) {
    super('SRV_004', message, 429, details);
  }
}

module.exports = {
  AppError,
  AuthenticationError,
  ValidationError,
  NotFoundError,
  DatabaseError,
  SecurityError,
  RateLimitError
};
