export const ErrorCodes = {
  AUTHENTICATION_ERRORS: {
    AUTH_001: 'INVALID_CREDENTIALS',
    AUTH_002: 'TOKEN_EXPIRED',
    AUTH_003: 'TOKEN_INVALID',
    AUTH_004: 'ACCOUNT_LOCKED',
    AUTH_005: 'ACCOUNT_DISABLED',
    AUTH_006: 'SESSION_EXPIRED',
    AUTH_007: 'UNAUTHORIZED_ACCESS',
    AUTH_008: 'INSUFFICIENT_PERMISSIONS'
  },
  VALIDATION_ERRORS: {
    VAL_001: 'VALIDATION_FAILED',
    VAL_002: 'INVALID_INPUT',
    VAL_003: 'MISSING_REQUIRED_FIELD',
    VAL_004: 'INVALID_FORMAT',
    VAL_005: 'OUT_OF_RANGE',
    VAL_006: 'DUPLICATE_ENTRY'
  },
  SERVER_ERRORS: {
    SRV_001: 'INTERNAL_ERROR',
    SRV_002: 'SERVICE_UNAVAILABLE',
    SRV_003: 'TIMEOUT',
    SRV_004: 'RATE_LIMIT_EXCEEDED',
    SRV_005: 'MAINTENANCE_MODE'
  },
  SECURITY_ERRORS: {
    SEC_001: 'XSS_ATTACK_DETECTED',
    SEC_002: 'SQL_INJECTION_DETECTED',
    SEC_003: 'CSRF_TOKEN_INVALID',
    SEC_004: 'RATE_LIMIT_VIOLATION',
    SEC_005: 'INVALID_CSRF_TOKEN',
    SEC_006: 'BRUTE_FORCE_DETECTED'
  }
};

export const ErrorCategories = {
  AUTH: 'Authentication',
  VAL: 'Validation',
  DB: 'Database',
  SRV: 'Server',
  SEC: 'Security',
  FILE: 'File'
};

export const HttpStatusCodes = {
  400: 'Bad Request',
  401: 'Unauthorized',
  403: 'Forbidden',
  404: 'Not Found',
  409: 'Conflict',
  422: 'Unprocessable Entity',
  429: 'Too Many Requests',
  500: 'Internal Server Error',
  502: 'Bad Gateway',
  503: 'Service Unavailable'
};

export function getErrorCategory(code) {
  if (!code) return 'Unknown';
  const prefix = code.substring(0, 3);
  return ErrorCategories[prefix] || 'Unknown';
}

export function getErrorMessage(code, defaultMessage) {
  const messages = {
    AUTH_001: 'Invalid credentials',
    AUTH_002: 'Your session has expired',
    AUTH_003: 'Invalid authentication token',
    AUTH_004: 'Your account has been locked',
    AUTH_005: 'Your account has been disabled',
    AUTH_006: 'Session expired',
    AUTH_007: 'You are not authorized to perform this action',
    AUTH_008: 'You do not have permission to perform this action',
    VAL_001: 'Validation failed',
    VAL_002: 'Invalid input provided',
    VAL_003: 'Required field is missing',
    VAL_004: 'Invalid format',
    VAL_005: 'Value is out of range',
    VAL_006: 'Duplicate entry found',
    SRV_001: 'Something went wrong',
    SRV_002: 'Service is temporarily unavailable',
    SRV_003: 'Request timed out',
    SRV_004: 'Too many requests, please try again later',
    SRV_005: 'System is under maintenance',
    SEC_001: 'Security violation detected',
    SEC_002: 'Potential security threat blocked',
    SEC_003: 'Invalid security token',
    SEC_004: 'Rate limit exceeded',
    SEC_005: 'Invalid CSRF token',
    SEC_006: 'Suspicious activity detected'
  };
  return messages[code] || defaultMessage || 'An error occurred';
}

export function getHttpStatus(code) {
  const statusMap = {
    'AUTH': 401,
    'VAL': 400,
    'DB': 500,
    'SRV': 500,
    'SEC': 403,
    'FILE': 400
  };
  if (!code) return 500;
  const prefix = code.substring(0, 3);
  return statusMap[prefix] || 500;
}
