export class CaptchaError extends Error {
  public readonly code: string;
  public readonly statusCode?: number;
  public readonly retryable: boolean;

  constructor(
    message: string,
    code: string = 'UNKNOWN_ERROR',
    statusCode?: number,
    retryable: boolean = false
  ) {
    super(message);
    this.name = 'CaptchaError';
    this.code = code;
    this.statusCode = statusCode;
    this.retryable = retryable;
    Object.setPrototypeOf(this, CaptchaError.prototype);
  }
}

export class ValidationError extends CaptchaError {
  constructor(message: string) {
    super(message, 'VALIDATION_ERROR', 400, false);
    this.name = 'ValidationError';
    Object.setPrototypeOf(this, ValidationError.prototype);
  }
}

export class AuthenticationError extends CaptchaError {
  constructor(message: string = 'Authentication failed') {
    super(message, 'AUTHENTICATION_ERROR', 401, false);
    this.name = 'AuthenticationError';
    Object.setPrototypeOf(this, AuthenticationError.prototype);
  }
}

export class NotFoundError extends CaptchaError {
  constructor(message: string = 'Resource not found') {
    super(message, 'NOT_FOUND', 404, false);
    this.name = 'NotFoundError';
    Object.setPrototypeOf(this, NotFoundError.prototype);
  }
}

export class RateLimitError extends CaptchaError {
  public readonly retryAfter?: number;

  constructor(message: string = 'Rate limit exceeded', retryAfter?: number) {
    super(message, 'RATE_LIMIT_ERROR', 429, true);
    this.name = 'RateLimitError';
    this.retryAfter = retryAfter;
    Object.setPrototypeOf(this, RateLimitError.prototype);
  }
}

export class ServerError extends CaptchaError {
  constructor(message: string = 'Server error', statusCode: number = 500) {
    super(message, 'SERVER_ERROR', statusCode, true);
    this.name = 'ServerError';
    Object.setPrototypeOf(this, ServerError.prototype);
  }
}

export class NetworkError extends CaptchaError {
  public readonly cause?: Error;

  constructor(message: string = 'Network error', cause?: Error) {
    super(message, 'NETWORK_ERROR', undefined, true);
    this.name = 'NetworkError';
    this.cause = cause;
    Object.setPrototypeOf(this, NetworkError.prototype);
  }
}
