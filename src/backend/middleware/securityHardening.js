const helmet = require('helmet');

const securityHeaders = helmet({
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      scriptSrc: ["'self'", "'unsafe-inline'", "'unsafe-eval'"],
      styleSrc: ["'self'", "'unsafe-inline'", 'https://fonts.googleapis.com'],
      fontSrc: ["'self'", 'https://fonts.gstatic.com'],
      imgSrc: ["'self'", 'data:', 'https:', 'blob:'],
      connectSrc: ["'self'", 'wss:', 'https:'],
      mediaSrc: ["'self'"],
      objectSrc: ["'none'"],
      upgradeInsecureRequests: []
    }
  },
  crossOriginEmbedderPolicy: false,
  crossOriginResourcePolicy: { policy: 'cross-origin' },
  dnsPrefetchControl: { allow: false },
  frameguard: { action: 'deny' },
  hidePoweredBy: true,
  hsts: {
    maxAge: 31536000,
    includeSubDomains: true,
    preload: true
  },
  ieNoOpen: true,
  noSniff: true,
  originAgentCluster: true,
  permittedCrossDomainPolicies: { permittedPolicies: 'none' },
  referrerPolicy: { policy: 'strict-origin-when-cross-origin' },
  xssFilter: true
});

function sanitizeInput(req, res, next) {
  const escapeHtml = str => {
    if (typeof str !== 'string') return str;
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#x27;')
      .replace(/\//g, '&#x2F;');
  };

  const sanitizeObject = obj => {
    if (typeof obj === 'string') {
      return escapeHtml(obj);
    }
    if (Array.isArray(obj)) {
      return obj.map(item => sanitizeObject(item));
    }
    if (typeof obj === 'object' && obj !== null) {
      const sanitized = {};
      for (const [key, value] of Object.entries(obj)) {
        sanitized[escapeHtml(key)] = sanitizeObject(value);
      }
      return sanitized;
    }
    return obj;
  };

  if (req.body) {
    req.body = sanitizeObject(req.body);
  }
  if (req.query) {
    req.query = sanitizeObject(req.query);
  }
  if (req.params) {
    req.params = sanitizeObject(req.params);
  }

  next();
}

function validateContentType(req, res, next) {
  if (req.method === 'POST' || req.method === 'PUT' || req.method === 'PATCH') {
    const contentType = req.headers['content-type'];

    if (!contentType) {
      return res.status(415).json({
        success: false,
        error: {
          code: 'UNSUPPORTED_MEDIA_TYPE',
          message: 'Content-Type header is required'
        }
      });
    }

    const allowedTypes = [
      'application/json',
      'application/x-www-form-urlencoded',
      'multipart/form-data'
    ];
    const isAllowed = allowedTypes.some(type => contentType.includes(type));

    if (!isAllowed) {
      return res.status(415).json({
        success: false,
        error: {
          code: 'UNSUPPORTED_MEDIA_TYPE',
          message: `Content-Type must be one of: ${allowedTypes.join(', ')}`
        }
      });
    }
  }

  next();
}

function preventBruteForce(req, res, next) {
  const sensitiveEndpoints = [
    '/api/v1/auth/login',
    '/api/v1/auth/register',
    '/api/v1/password/reset'
  ];
  const isSensitive = sensitiveEndpoints.some(endpoint => req.path.includes(endpoint));

  if (!isSensitive) {
    return next();
  }

  const clientIP = req.ip || req.connection.remoteAddress;
  const key = `bruteforce:${clientIP}:${req.path}`;
  const attemptKey = `${key}:attempts`;
  const lockKey = `${key}:locked`;

  const redis = require('../../../config/redis/client');

  (async () => {
    try {
      const isLocked = await redis.get(lockKey);
      if (isLocked) {
        return res.status(429).json({
          success: false,
          error: {
            code: 'ACCOUNT_LOCKED',
            message: 'Too many failed attempts. Please try again later.'
          }
        });
      }

      const attempts = (await redis.get(attemptKey)) || '0';
      const newAttempts = parseInt(attempts) + 1;

      await redis.setEx(attemptKey, 900, newAttempts.toString());

      if (newAttempts >= 5) {
        await redis.setEx(lockKey, 900, '1');
        return res.status(429).json({
          success: false,
          error: {
            code: 'TOO_MANY_ATTEMPTS',
            message: 'Too many failed attempts. Account locked for 15 minutes.'
          }
        });
      }

      next();
    } catch (error) {
      console.error('Brute force check error:', error);
      next();
    }
  })();
}

function strictTransportSecurity(req, res, next) {
  if (process.env.NODE_ENV === 'production') {
    res.set({
      'Strict-Transport-Security': 'max-age=31536000; includeSubDomains; preload',
      'X-Content-Type-Options': 'nosniff',
      'X-Frame-Options': 'DENY',
      'X-XSS-Protection': '1; mode=block',
      'Referrer-Policy': 'strict-origin-when-cross-origin'
    });
  }
  next();
}

function requestSizeLimit(options = {}) {
  const { maxSize = '1mb', fieldSize = '1mb' } = options;

  return (req, res, next) => {
    const contentLength = req.headers['content-length'];

    if (contentLength) {
      const maxBytes = parseSize(maxSize);
      if (parseInt(contentLength) > maxBytes) {
        return res.status(413).json({
          success: false,
          error: {
            code: 'PAYLOAD_TOO_LARGE',
            message: `Request body too large. Maximum size is ${maxSize}`
          }
        });
      }
    }

    next();
  };
}

function parseSize(size) {
  const units = { b: 1, kb: 1024, mb: 1024 * 1024, gb: 1024 * 1024 * 1024 };
  const match = size.toLowerCase().match(/^(\d+)(b|kb|mb|gb)?$/);
  if (!match) return 1024 * 1024;
  return parseInt(match[1]) * (units[match[2]] || 1);
}

function methodRestriction(allowedMethods) {
  return (req, res, next) => {
    if (!allowedMethods.includes(req.method)) {
      return res.status(405).json({
        success: false,
        error: {
          code: 'METHOD_NOT_ALLOWED',
          message: `Method ${req.method} is not allowed for this endpoint`
        }
      });
    }
    next();
  };
}

function apiVersioning(req, res, next) {
  const version = req.headers['api-version'] || 'v1';
  req.apiVersion = version;

  res.set({
    'API-Version': version,
    'X-API-Version': version
  });

  next();
}

module.exports = {
  securityHeaders,
  sanitizeInput,
  validateContentType,
  preventBruteForce,
  strictTransportSecurity,
  requestSizeLimit,
  methodRestriction,
  apiVersioning
};
