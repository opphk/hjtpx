class CSRFTokenManager {
  static generateToken(length = 32) {
    const crypto = require('crypto');
    return crypto.randomBytes(length).toString('hex');
  }

  static setCookieToken(res, token, options = {}) {
    const cookieOptions = {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'strict',
      maxAge: 3600000,
      path: '/',
      ...options
    };

    res.cookie('_csrf', token, cookieOptions);
    res.header('x-csrf-token', token);
  }

  static validateToken(req) {
    const safeMethods = ['GET', 'HEAD', 'OPTIONS'];
    if (safeMethods.includes(req.method)) {
      return true;
    }

    const headerToken = req.headers['x-csrf-token'];
    const cookieToken = req.cookies && req.cookies._csrf;
    const bodyToken = req.body && req.body._csrf;

    const providedToken = headerToken || bodyToken || cookieToken;

    if (!providedToken) {
      return false;
    }

    const storedToken = cookieToken || (req.session && req.session.csrfToken);

    if (!storedToken) {
      return false;
    }

    return providedToken === storedToken;
  }

  static createTokenValidator(options = {}) {
    const {
      cookieName = '_csrf',
      headerName = 'x-csrf-token',
      bodyName = '_csrf',
      safeMethods = ['GET', 'HEAD', 'OPTIONS'],
      excludedPaths = []
    } = options;

    return (req, res, next) => {
      if (excludedPaths.includes(req.path)) {
        return next();
      }

      if (safeMethods.includes(req.method)) {
        return next();
      }

      const headerToken = req.headers[headerName];
      const bodyToken = req.body && req.body[bodyName];
      const providedToken = headerToken || bodyToken;

      if (!providedToken) {
        return res.status(403).json({
          error: 'CSRF token missing',
          message: 'CSRF token is required for this request'
        });
      }

      if (!this.validateToken(req)) {
        return res.status(403).json({
          error: 'CSRF token invalid',
          message: 'CSRF token validation failed'
        });
      }

      next();
    };
  }
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = CSRFTokenManager;
}
