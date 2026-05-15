const crypto = require('crypto');

const isProduction = process.env.NODE_ENV === 'production';
const isStaging = process.env.NODE_ENV === 'staging';

const generateNonce = () => {
  return crypto.randomBytes(16).toString('base64');
};

const getCSPDirectives = (nonce = null) => {
  const directives = {
    'default-src': ["'self'"],
    'script-src': ["'self'"],
    'style-src': ["'self'", "'unsafe-inline'", 'fonts.googleapis.com'],
    'font-src': ["'self'", 'fonts.gstatic.com'],
    'img-src': ["'self'", 'data:', 'https:', 'blob:'],
    'connect-src': ["'self'", 'wss:', 'https:'],
    'media-src': ["'self'"],
    'object-src': ["'none'"],
    'frame-src': ["'none'"],
    'frame-ancestors': ["'none'"],
    'form-action': ["'self'"],
    'base-uri': ["'self'"],
    'worker-src': ["'self'", 'blob:'],
    'manifest-src': ["'self'"],
    'child-src': ["'none'"],
    'report-uri': '/api/v1/security/csp-report'
  };

  if (nonce) {
    directives['script-src'].push(`'nonce-${nonce}'`, "'strict-dynamic'");
  } else {
    directives['script-src'].push("'unsafe-inline'", "'unsafe-eval'");
  }

  if (isProduction || isStaging) {
    directives['upgrade-insecure-requests'] = [];
    directives['block-all-mixed-content'] = [];
  }

  return directives;
};

const formatCSPHeader = (directives) => {
  return Object.entries(directives)
    .map(([directive, values]) => {
      if (Array.isArray(values) && values.length > 0) {
        return `${directive} ${values.join(' ')}`;
      } else if (values === []) {
        return directive;
      }
      return null;
    })
    .filter(Boolean)
    .join('; ');
};

const cspMiddleware = (req, res, next) => {
  const nonce = generateNonce();
  res.locals.cspNonce = nonce;

  const directives = getCSPDirectives(nonce);
  const cspHeader = formatCSPHeader(directives);

  res.setHeader('Content-Security-Policy', cspHeader);

  if (process.env.CSP_REPORT_ONLY === 'true') {
    res.setHeader('Content-Security-Policy-Report-Only', cspHeader);
  }

  next();
};

const createCSPReportHandler = () => {
  return (req, res) => {
    const cspReport = req.body['csp-report'] || req.body;
    
    if (process.env.NODE_ENV !== 'production') {
      console.warn('CSP Violation Report:', JSON.stringify(cspReport, null, 2));
    }

    try {
      const logService = require('../services/logService');
      if (logService && logService.logSecurityEvent) {
        logService.logSecurityEvent('csp_violation', {
          report: cspReport,
          ip: req.ip,
          userAgent: req.get('user-agent'),
          path: req.path
        });
      }
    } catch (error) {
      console.warn('Failed to log CSP violation:', error.message);
    }

    res.status(204).send();
  };
};

const validateCSPConfiguration = (config) => {
  const errors = [];

  if (!config.directives || typeof config.directives !== 'object') {
    errors.push('CSP configuration must include directives object');
  }

  const requiredDirectives = ['default-src', 'script-src', 'object-src'];
  requiredDirectives.forEach(directive => {
    if (!config.directives || !config.directives[directive]) {
      errors.push(`Missing required directive: ${directive}`);
    }
  });

  if (config.directives?.['object-src']?.[0] !== "'none'" && 
      config.directives?.['object-src']?.[0] !== 'none') {
    errors.push('object-src should be set to "none" for security');
  }

  if (config.directives?.['frame-ancestors']?.[0] !== "'none'" &&
      config.directives?.['frame-ancestors']?.[0] !== 'none') {
    errors.push('frame-ancestors should be set to "none" for security');
  }

  return {
    valid: errors.length === 0,
    errors
  };
};

const createStrictCSP = () => {
  const nonce = generateNonce();
  const directives = getCSPDirectives(nonce);
  
  directives['script-src'] = ["'self'", `nonce-${nonce}`, "'strict-dynamic'"];
  directives['style-src'] = ["'self'", "'unsafe-inline'"];
  directives['img-src'] = ["'self'", 'data:', 'blob:'];
  directives['connect-src'] = ["'self'", 'wss:', 'https:'];
  
  if (isProduction || isStaging) {
    directives['upgrade-insecure-requests'] = [];
  }

  return formatCSPHeader(directives);
};

const createPermissiveCSP = () => {
  const directives = {
    'default-src': ["'self'"],
    'script-src': ["'self'", "'unsafe-inline'", "'unsafe-eval'"],
    'style-src': ["'self'", "'unsafe-inline'"],
    'font-src': ["'self'", 'https:'],
    'img-src': ["'self'", 'data:', 'https:'],
    'connect-src': ["'self'", 'wss:', 'https:'],
    'media-src': ["'self'"],
    'object-src': ["'none'"],
    'frame-src': ["'none'"],
    'frame-ancestors': ["'none'"],
    'form-action': ["'self'"],
    'base-uri': ["'self'"],
    'worker-src': ["'self'", 'blob:', 'https:'],
    'manifest-src': ["'self'"],
    'child-src': ["'none'"]
  };

  return formatCSPHeader(directives);
};

module.exports = {
  cspMiddleware,
  generateNonce,
  getCSPDirectives,
  formatCSPHeader,
  createCSPReportHandler,
  validateCSPConfiguration,
  createStrictCSP,
  createPermissiveCSP
};
