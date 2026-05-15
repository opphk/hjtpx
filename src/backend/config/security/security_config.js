const CSPManager = require('../../frontend/src/utils/csp_manager');
const { SENSITIVE_FIELDS } = require('../utils/security/data_encryption');

const securityConfig = {
  apiSignature: {
    enabled: process.env.API_SIGNATURE_ENABLED !== 'false',
    algorithm: 'hmac-sha256',
    timestampWindow: 300,
    excludePaths: ['/health', '/public', '/api/health', '/api/v1/health']
  },

  ipControl: {
    enabled: process.env.IP_CONTROL_ENABLED !== 'false',
    whitelist: process.env.IP_WHITELIST
      ? process.env.IP_WHITELIST.split(',')
      : ['127.0.0.1', '::1'],
    blacklist: process.env.IP_BLACKLIST ? process.env.IP_BLACKLIST.split(',') : [],
    defaultAction: process.env.IP_DEFAULT_ACTION || 'allow'
  },

  geoRestriction: {
    enabled: process.env.GEO_RESTRICTION_ENABLED === 'true',
    allowedCountries: process.env.ALLOWED_COUNTRIES
      ? process.env.ALLOWED_COUNTRIES.split(',')
      : ['CN', 'US', 'HK', 'TW', 'SG', 'JP', 'KR'],
    blockedCountries: process.env.BLOCKED_COUNTRIES ? process.env.BLOCKED_COUNTRIES.split(',') : []
  },

  rateLimiting: {
    enabled: process.env.RATE_LIMIT_ENABLED !== 'false',
    global: {
      windowMs: parseInt(process.env.RATE_LIMIT_GLOBAL_WINDOW) || 60000,
      max: parseInt(process.env.RATE_LIMIT_GLOBAL_MAX) || 100
    },
    api: {
      windowMs: parseInt(process.env.RATE_LIMIT_API_WINDOW) || 1000,
      max: parseInt(process.env.RATE_LIMIT_API_MAX) || 10
    },
    auth: {
      windowMs: parseInt(process.env.RATE_LIMIT_AUTH_WINDOW) || 300000,
      max: parseInt(process.env.RATE_LIMIT_AUTH_MAX) || 5
    },
    upload: {
      windowMs: parseInt(process.env.RATE_LIMIT_UPLOAD_WINDOW) || 60000,
      max: parseInt(process.env.RATE_LIMIT_UPLOAD_MAX) || 5
    }
  },

  csrf: {
    enabled: process.env.CSRF_ENABLED !== 'false',
    cookieName: '_csrf',
    headerName: 'x-csrf-token',
    bodyName: '_csrf',
    tokenLength: 32,
    safeMethods: ['GET', 'HEAD', 'OPTIONS'],
    excludedPaths: ['/api/health']
  },

  csp: {
    enabled: process.env.CSP_ENABLED !== 'false',
    reportOnly: process.env.CSP_REPORT_ONLY === 'true',
    reportUri: '/csp-violation-report',
    policy: CSPManager.getPolicy()
  },

  encryption: {
    enabled: process.env.ENCRYPTION_ENABLED !== 'false',
    algorithm: 'aes-256-gcm',
    keyRotation: parseInt(process.env.ENCRYPTION_KEY_ROTATION_DAYS) || 30,
    encryptedFields: SENSITIVE_FIELDS,
    masterKey: process.env.ENCRYPTION_MASTER_KEY
  },

  xss: {
    enabled: process.env.XSS_PROTECTION_ENABLED !== 'false',
    escapeHtml: true,
    removeScriptTags: true,
    removeEventHandlers: true,
    blockJavascriptUrls: true,
    sanitizationRules: {
      maxLength: 10000,
      allowedTags: [],
      allowedAttributes: []
    }
  },

  sqlInjection: {
    enabled: process.env.SQL_INJECTION_PROTECTION_ENABLED !== 'false',
    useParameterizedQueries: true,
    escapeCharacters: true,
    validatePatterns: true,
    dangerousPatterns: [
      /\b(SELECT|INSERT|UPDATE|DELETE|DROP)\b/gi,
      /\b(UNION|EXEC|EXECUTE)\b/gi,
      /(--|#|\/\*)/g,
      /;\s*(DROP|INSERT|UPDATE|DELETE)/gi
    ]
  },

  headers: {
    enabled: process.env.SECURITY_HEADERS_ENABLED !== 'false',
    hsts: {
      enabled: process.env.NODE_ENV === 'production',
      maxAge: 31536000,
      includeSubDomains: true,
      preload: true
    },
    xFrameOptions: 'DENY',
    xContentTypeOptions: 'nosniff',
    xXSSProtection: '1; mode=block',
    referrerPolicy: 'strict-origin-when-cross-origin',
    permissionsPolicy: 'geolocation=(), microphone=(), camera=()'
  },

  authentication: {
    maxLoginAttempts: parseInt(process.env.MAX_LOGIN_ATTEMPTS) || 5,
    lockoutDuration: parseInt(process.env.LOCKOUT_DURATION) || 900,
    sessionTimeout: parseInt(process.env.SESSION_TIMEOUT) || 3600,
    requireTwoFactor: process.env.REQUIRE_2FA === 'true',
    passwordMinLength: parseInt(process.env.PASSWORD_MIN_LENGTH) || 8,
    passwordRequireSpecial: true,
    passwordRequireNumber: true,
    passwordRequireUppercase: true
  },

  audit: {
    enabled: process.env.AUDIT_LOG_ENABLED !== 'false',
    logDir: process.env.AUDIT_LOG_DIR || 'logs/audit',
    retentionDays: parseInt(process.env.AUDIT_RETENTION_DAYS) || 365,
    logSensitiveOperations: true,
    logFailedAttempts: true,
    logDataAccess: true,
    actions: {
      critical: ['DELETE_ACCOUNT', 'PERMISSION_CHANGE', 'SECURITY_BREACH'],
      high: ['LOGIN', 'PASSWORD_CHANGE', 'API_KEY_CREATE'],
      medium: ['UPDATE_PROFILE', 'PASSWORD_RESET'],
      low: ['VIEW', 'LIST', 'SEARCH']
    }
  },

  monitoring: {
    enabled: process.env.SECURITY_MONITORING_ENABLED !== 'false',
    alertThreshold: {
      failedLogin: parseInt(process.env.ALERT_FAILED_LOGIN) || 5,
      suspiciousIP: parseInt(process.env.ALERT_SUSPICIOUS_IP) || 10,
      xssAttempt: parseInt(process.env.ALERT_XSS_ATTEMPT) || 3,
      sqlInjection: parseInt(process.env.ALERT_SQL_INJECTION) || 1
    },
    alertMethods: process.env.SECURITY_ALERT_METHODS
      ? process.env.SECURITY_ALERT_METHODS.split(',')
      : ['email', 'slack'],
    alertRecipients: process.env.SECURITY_ALERT_RECIPIENTS
      ? process.env.SECURITY_ALERT_RECIPIENTS.split(',')
      : []
  },

  cors: {
    enabled: true,
    origin: process.env.CORS_ORIGIN || '*',
    methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
    allowedHeaders: ['Content-Type', 'Authorization', 'X-Requested-With'],
    exposedHeaders: ['X-RateLimit-Limit', 'X-RateLimit-Remaining'],
    credentials: true,
    maxAge: 86400
  },

  contentTypeValidation: {
    enabled: true,
    maxUploadSize: parseInt(process.env.MAX_UPLOAD_SIZE) || 10 * 1024 * 1024,
    allowedMimeTypes: [
      'image/jpeg',
      'image/png',
      'image/gif',
      'application/pdf',
      'application/json'
    ]
  },

  requestValidation: {
    maxBodySize: parseInt(process.env.MAX_BODY_SIZE) || 1024 * 1024,
    maxQueryParams: 100,
    maxHeaderSize: 8192,
    jsonDepthLimit: 10
  }
};

module.exports = securityConfig;
