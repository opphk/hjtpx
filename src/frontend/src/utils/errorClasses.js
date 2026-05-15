import { getErrorMessage, getErrorCategory, getHttpStatus } from './errorCodes';

export class AppError extends Error {
  constructor(code, message, details = null, statusCode = null) {
    super(message);
    this.code = code;
    this.statusCode = statusCode || getHttpStatus(code);
    this.details = details;
    this.category = getErrorCategory(code);
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
        category: this.category,
        statusCode: this.statusCode,
        details: this.details,
        timestamp: this.timestamp
      }
    };
  }
}

export class NetworkError extends AppError {
  constructor(message = 'Network error occurred', details = null) {
    super('SRV_002', message, details, 503);
    this.name = 'NetworkError';
  }
}

export class TimeoutError extends AppError {
  constructor(message = 'Request timed out', details = null) {
    super('SRV_003', message, details, 408);
    this.name = 'TimeoutError';
  }
}

export class AuthenticationError extends AppError {
  constructor(code = 'AUTH_001', message = 'Authentication failed', details = null) {
    super(code, message, details, 401);
    this.name = 'AuthenticationError';
  }
}

export class ValidationError extends AppError {
  constructor(message = 'Validation failed', details = null) {
    super('VAL_001', message, details, 400);
    this.name = 'ValidationError';
  }
}

export class NotFoundError extends AppError {
  constructor(resource = 'Resource', details = null) {
    super('DB_004', `${resource} not found`, details, 404);
    this.name = 'NotFoundError';
  }
}

export function parseErrorResponse(response) {
  if (!response || !response.data) {
    return new AppError('SRV_001', 'An unexpected error occurred');
  }

  const { error } = response.data;

  if (error) {
    const code = error.code || 'UNKNOWN';
    const message = error.message || getErrorMessage(code);
    return new AppError(code, message, error.details, error.statusCode);
  }

  return new AppError('SRV_001', 'An unexpected error occurred');
}

export function isAuthError(error) {
  if (error instanceof AppError) {
    return error.code && error.code.startsWith('AUTH');
  }
  return false;
}

export function isNetworkError(error) {
  if (error instanceof NetworkError || error instanceof TimeoutError) {
    return true;
  }
  if (error.code === 'ECONNREFUSED' || error.code === 'NETWORK_ERROR') {
    return true;
  }
  return false;
}

export function shouldRetry(error) {
  if (error instanceof TimeoutError) return true;
  if (error instanceof NetworkError) return true;
  if (error.code === 'SRV_004') return true;
  return false;
}
