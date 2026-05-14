const { sanitizeObject, escapeHtml, removeXssPatterns } = require('../utils/xssSanitizer');

const CSP_POLICIES = {
  default: [
    "default-src 'self'",
    "script-src 'self' 'unsafe-inline' 'unsafe-eval'",
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data: https:",
    "font-src 'self' data:",
    "connect-src 'self' wss: https:",
    "frame-src 'none'",
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "frame-ancestors 'none'",
    "upgrade-insecure-requests"
  ],
  strict: [
    "default-src 'self'",
    "script-src 'self'",
    "style-src 'self'",
    "img-src 'self' data:",
    "font-src 'self'",
    "connect-src 'self'",
    "frame-src 'none'",
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "frame-ancestors 'none'",
    "upgrade-insecure-requests"
  ],
  relaxed: [
    "default-src 'self' 'unsafe-inline' 'unsafe-eval'",
    "script-src 'self' 'unsafe-inline' 'unsafe-eval'",
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data: https: blob:",
    "font-src 'self' data:",
    "connect-src 'self' wss: https:",
    "frame-src 'self'",
    "object-src 'self'",
    "base-uri 'self'",
    "form-action 'self'"
  ]
};

function xssProtection(options = {}) {
  const {
    enabled = true,
    sanitizeBody = true,
    sanitizeQuery = true,
    sanitizeParams = true,
    sanitizeHeaders = false,
    stripAllTags = false,
    maxBodySize = '10mb',
    excludedPaths = [],
    cspMode = 'default'
  } = options;
  
  return (req, res, next) => {
    if (!enabled) {
      return next();
    }
    
    if (excludedPaths.some(path => req.path.startsWith(path))) {
      return next();
    }
    
    try {
      if (sanitizeBody && req.body) {
        req.body = sanitizeObject(req.body, { stripAllTags });
      }
      
      if (sanitizeQuery && req.query) {
        req.query = sanitizeObject(req.query, { stripAllTags });
      }
      
      if (sanitizeParams && req.params) {
        req.params = sanitizeObject(req.params, { stripAllTags });
      }
      
      if (sanitizeHeaders && req.headers) {
        const sanitizedHeaders = {};
        for (const [key, value] of Object.entries(req.headers)) {
          if (typeof value === 'string') {
            sanitizedHeaders[key] = sanitizeObject(value, { stripAllTags });
          } else if (Array.isArray(value)) {
            sanitizedHeaders[key] = value.map(v => 
              typeof v === 'string' ? sanitizeObject(v, { stripAllTags }) : v
            );
          }
        }
        req.headers = sanitizedHeaders;
      }
      
      req.xssSanitized = true;
    } catch (error) {
      console.error('XSS sanitization error:', error);
      return res.status(400).json({
        success: false,
        error: 'Invalid request data'
      });
    }
    
    next();
  };
}

function setCSPHeaders(options = {}) {
  const mode = options.mode || 'default';
  const policies = CSP_POLICIES[mode] || CSP_POLICIES.default;
  const reportUri = options.reportUri;
  
  return (req, res, next) => {
    let policyString = policies.join('; ');
    
    if (reportUri) {
      policyString += `; report-uri ${reportUri}`;
    }
    
    res.setHeader('Content-Security-Policy', policyString);
    res.setHeader('X-Content-Security-Policy', policyString);
    res.setHeader('X-WebKit-CSP', policyString);
    
    next();
  };
}

function xssDetector() {
  const suspiciousPatterns = [
    /<script/i,
    /javascript:/i,
    /on\w+\s*=/i,
    /<iframe/i,
    /<embed/i,
    /<object/i,
    /<link/i,
    /<meta/i,
    /expression\s*\(/i,
    /url\s*\(/i,
    /data\s*:/i,
    /vbscript:/i
  ];
  
  return (req, res, next) => {
    const checkValue = (key, value, path = '') => {
      if (typeof value !== 'string') return false;
      
      for (const pattern of suspiciousPatterns) {
        if (pattern.test(value)) {
          console.warn(`Suspicious XSS pattern detected in ${path}${key}:`, value.substring(0, 100));
          return true;
        }
      }
      return false;
    };
    
    const checkObject = (obj, path = '') => {
      for (const [key, value] of Object.entries(obj)) {
        const currentPath = path ? `${path}.${key}` : key;
        
        if (typeof value === 'string') {
          if (checkValue(key, value, currentPath)) {
            return true;
          }
        } else if (Array.isArray(value)) {
          for (let i = 0; i < value.length; i++) {
            if (typeof value[i] === 'string' && checkValue(key, value[i], `${currentPath}[${i}]`)) {
              return true;
            }
          }
        } else if (value && typeof value === 'object') {
          if (checkObject(value, currentPath)) {
            return true;
          }
        }
      }
      return false;
    };
    
    try {
      if (req.body && checkObject(req.body, 'body')) {
        req.xssAttempt = true;
      }
      
      if (req.query && checkObject(req.query, 'query')) {
        req.xssAttempt = true;
      }
    } catch (error) {
      console.error('XSS detection error:', error);
    }
    
    next();
  };
}

function strictXssValidation() {
  return (req, res, next) => {
    const dangerousPatterns = [
      { pattern: /<script[\s\S]*?>[\s\S]*?<\/script>/gi, name: 'script_tag' },
      { pattern: /<iframe[\s\S]*?>[\s\S]*?<\/iframe>/gi, name: 'iframe_tag' },
      { pattern: /javascript\s*:/gi, name: 'javascript_protocol' },
      { pattern: /on\w+\s*=\s*["'][^"']*["']/gi, name: 'event_handler' },
      { pattern: /on\w+\s*=\s*[^\s>]+/gi, name: 'event_handler_short' },
      { pattern: /<embed[\s\S]*?>/gi, name: 'embed_tag' },
      { pattern: /<object[\s\S]*?>/gi, name: 'object_tag' },
      { pattern: /data\s*:\s*text\/html/gi, name: 'data_url_html' },
      { pattern: /expression\s*\([^)]*\)/gi, name: 'css_expression' },
      { pattern: /url\s*\([^)]*\)/gi, name: 'css_url' },
      { pattern: /vbscript\s*:/gi, name: 'vbscript_protocol' }
    ];
    
    const validateValue = (value) => {
      if (typeof value !== 'string') return true;
      
      for (const { pattern } of dangerousPatterns) {
        if (pattern.test(value)) {
          return false;
        }
      }
      return true;
    };
    
    const validateObject = (obj) => {
      for (const value of Object.values(obj)) {
        if (typeof value === 'string') {
          if (!validateValue(value)) {
            return false;
          }
        } else if (Array.isArray(value)) {
          for (const item of value) {
            if (typeof item === 'string' && !validateValue(item)) {
              return false;
            }
          }
        } else if (value && typeof value === 'object') {
          if (!validateObject(value)) {
            return false;
          }
        }
      }
      return true;
    };
    
    try {
      if (req.body && !validateObject(req.body)) {
        return res.status(400).json({
          success: false,
          error: 'Malicious content detected'
        });
      }
    } catch (error) {
      console.error('XSS validation error:', error);
    }
    
    next();
  };
}

function addSecurityHeaders() {
  return (req, res, next) => {
    res.setHeader('X-XSS-Protection', '1; mode=block');
    res.setHeader('X-Content-Type-Options', 'nosniff');
    res.setHeader('X-Frame-Options', 'DENY');
    res.setHeader('X-Permitted-Cross-Domain-Policies', 'none');
    res.setHeader('Referrer-Policy', 'strict-origin-when-cross-origin');
    res.setHeader('Permissions-Policy', 'geolocation=(), microphone=(), camera=()');
    
    next();
  };
}

function createXssMiddleware(options = {}) {
  return [
    addSecurityHeaders(),
    xssProtection(options),
    xssDetector(),
    strictXssValidation(),
    setCSPHeaders(options.csp ? { mode: options.csp } : {})
  ].filter(Boolean);
}

module.exports = {
  xssProtection,
  setCSPHeaders,
  xssDetector,
  strictXssValidation,
  addSecurityHeaders,
  createXssMiddleware,
  CSP_POLICIES
};
