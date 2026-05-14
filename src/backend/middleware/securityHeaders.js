const helmet = require('helmet');

const isProduction = process.env.NODE_ENV === 'production';

const securityHeaders = helmet({
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      scriptSrc: ["'self'", "'unsafe-inline'", 'cdn.jsdelivr.net'],
      styleSrc: ["'self'", "'unsafe-inline'", 'fonts.googleapis.com'],
      fontSrc: ["'self'", 'fonts.gstatic.com'],
      imgSrc: ["'self'", 'data:', 'https:'],
      connectSrc: ["'self'"],
      mediaSrc: ["'self'"],
      objectSrc: ["'none'"],
      upgradeInsecureRequests: isProduction ? [] : null
    }
  },
  crossOriginEmbedderPolicy: isProduction,
  crossOriginOpenerPolicy: isProduction,
  crossOriginResourcePolicy: { policy: 'same-origin' },
  dnsPrefetchControl: { allow: false },
  frameguard: { action: isProduction ? 'deny' : 'allowfrom' },
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

const corsOrigins = process.env.CORS_ORIGIN
  ? process.env.CORS_ORIGIN.split(',')
  : process.env.ALLOWED_ORIGINS
  ? process.env.ALLOWED_ORIGINS.split(',')
  : [
      'http://localhost:3000',
      'http://localhost:3001',
      'http://127.0.0.1:3000',
      'http://127.0.0.1:3001'
    ];

const additionalSecurityHeaders = (req, res, next) => {
  res.set({
    'X-Content-Type-Options': 'nosniff',
    'X-Frame-Options': isProduction ? 'DENY' : 'SAMEORIGIN',
    'X-XSS-Protection': '1; mode=block',
    'Strict-Transport-Security': 'max-age=31536000; includeSubDomains; preload',
    'Referrer-Policy': 'strict-origin-when-cross-origin',
    'Permissions-Policy': 'accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()',
    'X-Download-Options': 'noopen',
    'Cache-Control': 'no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0',
    'Pragma': 'no-cache',
    'X-Request-ID': req.requestId || `req_${Date.now()}`
  });

  if (isProduction) {
    res.set({
      'Vary': 'Origin',
      'X-Permitted-Cross-Domain-Policies': 'none'
    });
  }

  next();
};

module.exports = {
  securityHeaders,
  additionalSecurityHeaders
};
