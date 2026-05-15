class CSPManager {
  static getPolicy() {
    return {
      'default-src': ["'self'"],
      'script-src': ["'self'", "'strict-dynamic'", 'https://cdn.jsdelivr.net'],
      'style-src': ["'self'", "'unsafe-inline'", 'https://fonts.googleapis.com'],
      'img-src': ["'self'", 'data:', 'https:', 'blob:'],
      'connect-src': ["'self'", 'https://api.captchax.com', 'https://api.hjtpx.com'],
      'font-src': ["'self'", 'https://fonts.gstatic.com'],
      'frame-ancestors': ["'none'"],
      'form-action': ["'self'"],
      'base-uri': ["'self'"],
      'object-src': ["'none'"],
      'upgrade-insecure-requests': []
    };
  }

  static buildDirective(policy) {
    return Object.entries(policy)
      .map(([directive, values]) => {
        if (directive === 'upgrade-insecure-requests' || directive === 'block-all-mixed-content') {
          return directive;
        }
        return `${directive} ${values.join(' ')}`;
      })
      .join('; ');
  }

  static applyPolicy(res, customPolicy = null) {
    const policy = customPolicy || this.getPolicy();
    const csp = this.buildDirective(policy);
    
    res.setHeader('Content-Security-Policy', csp);
    res.setHeader('X-Content-Security-Policy', csp);
    res.setHeader('X-WebKit-CSP', csp);
  }

  static getReportOnlyPolicy() {
    return {
      'default-src': ["'self'"],
      'script-src': ["'self'"],
      'report-uri': ['/csp-violation-report']
    };
  }

  static applyReportOnly(res) {
    const policy = this.getReportOnlyPolicy();
    res.setHeader('Content-Security-Policy-Report-Only', this.buildDirective(policy));
  }

  static validateDirective(directive) {
    const validDirectives = [
      'default-src', 'script-src', 'style-src', 'img-src', 'font-src',
      'connect-src', 'media-src', 'object-src', 'frame-ancestors',
      'form-action', 'base-uri', 'child-src', 'worker-src', 'manifest-src',
      'report-uri', 'upgrade-insecure-requests', 'block-all-mixed-content'
    ];
    return validDirectives.includes(directive);
  }

  static addDirective(basePolicy, directive, values) {
    if (!this.validateDirective(directive)) {
      throw new Error(`Invalid CSP directive: ${directive}`);
    }
    return { ...basePolicy, [directive]: values };
  }

  static getNonce() {
    const crypto = require('crypto');
    return crypto.randomBytes(16).toString('base64');
  }

  static getPolicyWithNonce(basePolicy, nonce) {
    const policy = { ...basePolicy };
    if (policy['script-src']) {
      policy['script-src'] = [...policy['script-src'], `'nonce-${nonce}'`];
    }
    return policy;
  }

  static createMiddleware(customPolicy = null) {
    return (req, res, next) => {
      this.applyPolicy(res, customPolicy);
      next();
    };
  }

  static createReportOnlyMiddleware() {
    return (req, res, next) => {
      this.applyReportOnly(res);
      next();
    };
  }
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = CSPManager;
}
