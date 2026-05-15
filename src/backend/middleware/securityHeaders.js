const crypto = require('crypto');
const helmet = require('helmet');

const isProduction = process.env.NODE_ENV === 'production';
const isStaging = process.env.NODE_ENV === 'staging';

const nonceMiddleware = (req, res, next) => {
  res.locals.nonce = crypto.randomBytes(16).toString('base64');
  next();
};

const getCSPDirectives = (nonce) => {
  const directives = [
    `default-src 'self'`,
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic' cdn.jsdelivr.net cdnjs.cloudflare.com unpkg.com`,
    `style-src 'self' 'unsafe-inline' fonts.googleapis.com cdn.jsdelivr.net cdnjs.cloudflare.com`,
    `font-src 'self' fonts.gstatic.com cdn.jsdelivr.net cdnjs.cloudflare.com`,
    `img-src 'self' data: https: blob:`,
    `connect-src 'self' wss: https: https://api.captchax.com https://api.hjtpx.com`,
    `media-src 'self'`,
    `object-src 'none'`,
    `frame-src 'none'`,
    `frame-ancestors 'none'`,
    `form-action 'self'`,
    `base-uri 'self'`,
    `worker-src 'self' blob:`,
    `manifest-src 'self'`,
    `child-src 'none'`,
    `report-uri /api/v1/security/csp-report`
  ];

  if (isProduction || isStaging) {
    directives.push('upgrade-insecure-requests');
    directives.push('block-all-mixed-content');
  }

  return directives.join('; ');
};

const helmetMiddleware = (() => {
  const cspDirectives = {
    defaultSrc: ["'self'"],
    scriptSrc: ["'self'", "'unsafe-inline'", "'unsafe-eval'", 'cdn.jsdelivr.net', 'cdnjs.cloudflare.com', 'unpkg.com'],
    styleSrc: ["'self'", "'unsafe-inline'", 'fonts.googleapis.com', 'cdn.jsdelivr.net', 'cdnjs.cloudflare.com'],
    fontSrc: ["'self'", 'fonts.gstatic.com', 'cdn.jsdelivr.net', 'cdnjs.cloudflare.com'],
    imgSrc: ["'self'", 'data:', 'https:', 'blob:'],
    connectSrc: ["'self'", 'wss:', 'https:', 'https://api.captchax.com', 'https://api.hjtpx.com'],
    mediaSrc: ["'self'"],
    objectSrc: ["'none'"],
    frameSrc: ["'none'"],
    frameAncestors: ["'none'"],
    formAction: ["'self'"],
    baseUri: ["'self'"],
    workerSrc: ["'self'", 'blob:'],
    manifestSrc: ["'self'"],
    childSrc: ["'none'"]
  };

  if (isProduction || isStaging) {
    cspDirectives.upgradeInsecureRequests = [];
  }

  const helmetInstance = helmet({
    contentSecurityPolicy: {
      directives: cspDirectives,
      reportUri: '/api/v1/security/csp-report',
      setAllHeaders: true
    },
    crossOriginEmbedderPolicy: {
      policy: 'require-corp'
    },
    crossOriginResourcePolicy: {
      policy: 'same-origin'
    },
    crossOriginOpenerPolicy: {
      policy: 'same-origin'
    },
    dnsPrefetchControl: {
      allow: false
    },
    frameguard: {
      action: 'deny'
    },
    hidePoweredBy: true,
    hsts: {
      maxAge: 31536000,
      includeSubDomains: true,
      preload: true
    },
    ieNoOpen: true,
    noSniff: true,
    originAgentCluster: true,
    permittedCrossDomainPolicies: {
      permittedPolicies: 'none'
    },
    referrerPolicy: {
      policy: 'strict-origin-when-cross-origin'
    },
    xssFilter: true
  });

  return (req, res, next) => {
    helmetInstance(req, res, () => {
      res.set('Permissions-Policy', 'accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=(), fullscreen=(self), picture-in-picture=(self)');
      next();
    });
  };
})();

const securityHeaders = (req, res, next) => {
  const nonce = crypto.randomBytes(16).toString('base64');
  res.locals.cspNonce = nonce;

  const cspHeader = getCSPDirectives(nonce);

  res.set({
    'Content-Security-Policy': cspHeader,
    'Content-Security-Policy-Report-Only': process.env.CSP_REPORT_ONLY === 'true' ? cspHeader : undefined,
    'X-Content-Type-Options': 'nosniff',
    'X-Frame-Options': isProduction ? 'DENY' : 'SAMEORIGIN',
    'X-XSS-Protection': '1; mode=block',
    'Strict-Transport-Security': 'max-age=31536000; includeSubDomains; preload',
    'Referrer-Policy': 'strict-origin-when-cross-origin',
    'Permissions-Policy': 'accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=(), fullscreen=(self), picture-in-picture=(self)',
    'X-Download-Options': 'noopen',
    'X-Permitted-Cross-Domain-Policies': 'none',
    'Cache-Control': 'no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0',
    'Pragma': 'no-cache',
    'X-Request-ID': req.requestId || `req_${Date.now()}_${crypto.randomBytes(8).toString('hex')}`,
    'Cross-Origin-Opener-Policy': 'same-origin',
    'Cross-Origin-Resource-Policy': 'same-origin',
    'Cross-Origin-Embedder-Policy': 'require-corp',
    'X-DNS-Prefetch-Control': 'off',
    'Origin-Agent-Cluster': '?1',
    'X-Permitted-Cross-Domain-Policies': 'none'
  });

  if (isProduction) {
    res.set({
      'Vary': 'Origin, X-Requested-With, Content-Type, Accept',
      'X-Powered-By': ''
    });
  }

  next();
};

const additionalSecurityHeaders = (req, res, next) => {
  res.set({
    'X-Content-Type-Options': 'nosniff',
    'X-Frame-Options': isProduction ? 'DENY' : 'SAMEORIGIN',
    'X-XSS-Protection': '1; mode=block',
    'Strict-Transport-Security': 'max-age=31536000; includeSubDomains; preload',
    'Referrer-Policy': 'strict-origin-when-cross-origin',
    'Permissions-Policy': 'accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=(), fullscreen=(self), picture-in-picture=(self)',
    'X-Download-Options': 'noopen',
    'X-Permitted-Cross-Domain-Policies': 'none',
    'Cache-Control': 'no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0',
    'Pragma': 'no-cache',
    'X-Request-ID': req.requestId || `req_${Date.now()}_${crypto.randomBytes(8).toString('hex')}`
  });

  if (isProduction) {
    res.set({
      'Vary': 'Origin, X-Requested-With, Content-Type, Accept'
    });
  }

  next();
};

const createCSPReportEndpoint = () => (req, res) => {
  const cspReport = req.body || req['csp-report'];
  console.warn('CSP Violation Report:', JSON.stringify(cspReport, null, 2));
  res.status(204).send();
};

module.exports = {
  securityHeaders,
  additionalSecurityHeaders,
  nonceMiddleware,
  createCSPReportEndpoint,
  helmetMiddleware
};
