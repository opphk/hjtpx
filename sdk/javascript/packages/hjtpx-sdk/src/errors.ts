export class SDKError extends Error {
  public readonly code: number;
  public readonly message: string;
  public readonly retryAfter?: number;

  constructor(code: number, message: string, retryAfter?: number) {
    super(`SDKError(code=${code}, message=${message})`);
    this.name = 'SDKError';
    this.code = code;
    this.message = message;
    this.retryAfter = retryAfter;

    Object.setPrototypeOf(this, SDKError.prototype);
  }

  isRateLimited(): boolean {
    return this.code === 429;
  }

  isUnauthorized(): boolean {
    return this.code === 401;
  }

  isServerError(): boolean {
    return this.code >= 500;
  }

  isInvalidParams(): boolean {
    return this.code === 400;
  }
}

export class NetworkError extends SDKError {
  constructor(message: string) {
    super(0, message);
    this.name = 'NetworkError';
    Object.setPrototypeOf(this, NetworkError.prototype);
  }
}

export class TimeoutError extends SDKError {
  constructor(message: string = 'Request timeout') {
    super(408, message);
    this.name = 'TimeoutError';
    Object.setPrototypeOf(this, TimeoutError.prototype);
  }
}

export class InvalidParamsError extends SDKError {
  constructor(message: string) {
    super(400, message);
    this.name = 'InvalidParamsError';
    Object.setPrototypeOf(this, InvalidParamsError.prototype);
  }
}

export class RateLimitedError extends SDKError {
  constructor(message: string = 'Rate limited', retryAfter?: number) {
    super(429, message, retryAfter);
    this.name = 'RateLimitedError';
    Object.setPrototypeOf(this, RateLimitedError.prototype);
  }
}

export class UnauthorizedError extends SDKError {
  constructor(message: string = 'Unauthorized') {
    super(401, message);
    this.name = 'UnauthorizedError';
    Object.setPrototypeOf(this, UnauthorizedError.prototype);
  }
}

export class ServerError extends SDKError {
  constructor(code: number, message: string = 'Server error') {
    super(code, message);
    this.name = 'ServerError';
    Object.setPrototypeOf(this, ServerError.prototype);
  }
}

export function isSDKError(error: unknown): error is SDKError {
  return error instanceof SDKError;
}

export function getErrorCode(error: unknown): number {
  if (error instanceof SDKError) {
    return error.code;
  }
  return 0;
}
