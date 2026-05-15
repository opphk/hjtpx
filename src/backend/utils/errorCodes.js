class ErrorCode {
  static AUTHENTICATION_ERRORS = {
    AUTH_001: 'INVALID_CREDENTIALS',
    AUTH_002: 'TOKEN_EXPIRED',
    AUTH_003: 'TOKEN_INVALID',
    AUTH_004: 'ACCOUNT_LOCKED',
    AUTH_005: 'ACCOUNT_DISABLED',
    AUTH_006: 'SESSION_EXPIRED',
    AUTH_007: 'UNAUTHORIZED_ACCESS',
    AUTH_008: 'INSUFFICIENT_PERMISSIONS'
  };

  static VALIDATION_ERRORS = {
    VAL_001: 'VALIDATION_FAILED',
    VAL_002: 'INVALID_INPUT',
    VAL_003: 'MISSING_REQUIRED_FIELD',
    VAL_004: 'INVALID_FORMAT',
    VAL_005: 'OUT_OF_RANGE',
    VAL_006: 'DUPLICATE_ENTRY'
  };

  static DATABASE_ERRORS = {
    DB_001: 'CONNECTION_FAILED',
    DB_002: 'QUERY_FAILED',
    DB_003: 'TRANSACTION_FAILED',
    DB_004: 'RECORD_NOT_FOUND',
    DB_005: 'CONSTRAINT_VIOLATION'
  };

  static SERVER_ERRORS = {
    SRV_001: 'INTERNAL_ERROR',
    SRV_002: 'SERVICE_UNAVAILABLE',
    SRV_003: 'TIMEOUT',
    SRV_004: 'RATE_LIMIT_EXCEEDED',
    SRV_005: 'MAINTENANCE_MODE'
  };

  static SECURITY_ERRORS = {
    SEC_001: 'XSS_ATTACK_DETECTED',
    SEC_002: 'SQL_INJECTION_DETECTED',
    SEC_003: 'CSRF_TOKEN_INVALID',
    SEC_004: 'RATE_LIMIT_VIOLATION',
    SEC_005: 'INVALID_CSRF_TOKEN',
    SEC_006: 'BRUTE_FORCE_DETECTED'
  };

  static FILE_ERRORS = {
    FILE_001: 'FILE_TOO_LARGE',
    FILE_002: 'INVALID_FILE_TYPE',
    FILE_003: 'UPLOAD_FAILED',
    FILE_004: 'FILE_NOT_FOUND',
    FILE_005: 'STORAGE_ERROR'
  };

  static getCategory(code) {
    const prefix = code.substring(0, 3);
    const categories = {
      'AUTH': 'Authentication',
      'VAL': 'Validation',
      'DB': 'Database',
      'SRV': 'Server',
      'SEC': 'Security',
      'FILE': 'File'
    };
    return categories[prefix] || 'Unknown';
  }

  static getHttpStatus(code) {
    const statusMap = {
      'AUTH': 401,
      'VAL': 400,
      'DB': 500,
      'SRV': 500,
      'SEC': 403,
      'FILE': 400
    };
    const prefix = code.substring(0, 3);
    return statusMap[prefix] || 500;
  }
}

module.exports = ErrorCode;
