const crypto = require('crypto');
const pool = require('../../../config/database/db');

const CSRF_SECRET = process.env.CSRF_SECRET || crypto.randomBytes(32).toString('hex');
const TOKEN_LENGTH = 32;
const TOKEN_EXPIRY = 3600000;

async function generateToken(userId, sessionId) {
  const token = crypto.randomBytes(TOKEN_LENGTH).toString('hex');
  const expiresAt = new Date(Date.now() + TOKEN_EXPIRY);
  const hashedToken = crypto.createHash('sha256').update(token).digest('hex');
  
  await pool.query(
    `INSERT INTO csrf_tokens (user_id, session_id, token_hash, expires_at)
     VALUES ($1, $2, $3, $4)
     ON CONFLICT (user_id, session_id) 
     DO UPDATE SET token_hash = $3, expires_at = $4, created_at = CURRENT_TIMESTAMP`,
    [userId, sessionId, hashedToken, expiresAt]
  );
  
  return { token, expiresAt };
}

async function validateToken(userId, sessionId, token) {
  if (!token) {
    return { valid: false, reason: 'No token provided' };
  }
  
  const hashedToken = crypto.createHash('sha256').update(token).digest('hex');
  
  const result = await pool.query(
    `SELECT * FROM csrf_tokens 
     WHERE user_id = $1 AND session_id = $2 AND token_hash = $3`,
    [userId, sessionId, hashedToken]
  );
  
  if (result.rows.length === 0) {
    return { valid: false, reason: 'Invalid token' };
  }
  
  const storedToken = result.rows[0];
  
  if (new Date(storedToken.expires_at) < new Date()) {
    await invalidateToken(userId, sessionId);
    return { valid: false, reason: 'Token expired' };
  }
  
  return { valid: true };
}

async function invalidateToken(userId, sessionId) {
  await pool.query(
    'DELETE FROM csrf_tokens WHERE user_id = $1 AND session_id = $2',
    [userId, sessionId]
  );
}

async function invalidateAllTokens(userId) {
  await pool.query('DELETE FROM csrf_tokens WHERE user_id = $1', [userId]);
}

async function cleanupExpiredTokens() {
  const result = await pool.query('DELETE FROM csrf_tokens WHERE expires_at < NOW()');
  return result.rowCount;
}

function createCsrfToken() {
  return crypto.randomBytes(TOKEN_LENGTH).toString('hex');
}

function verifyCsrfToken(token, secret) {
  try {
    const expected = crypto
      .createHmac('sha256', secret)
      .update(token.split('.')[0])
      .digest('hex');
    
    const received = token.split('.')[1];
    
    if (!received) {
      return false;
    }
    
    return crypto.timingSafeEqual(
      Buffer.from(expected),
      Buffer.from(received)
    );
  } catch (error) {
    return false;
  }
}

function generateSignedToken(secret = CSRF_SECRET) {
  const randomPart = crypto.randomBytes(TOKEN_LENGTH).toString('hex');
  const signature = crypto
    .createHmac('sha256', secret)
    .update(randomPart)
    .digest('hex');
  
  return `${randomPart}.${signature}`;
}

function csrfProtection(options = {}) {
  const {
    excludePaths = [],
    excludeMethods = ['GET', 'HEAD', 'OPTIONS'],
    tokenLocation = 'body',
    cookieOptions = {
      httpOnly: false,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'strict'
    }
  } = options;
  
  return async (req, res, next) => {
    if (excludePaths.some(path => req.path.startsWith(path))) {
      return next();
    }
    
    if (excludeMethods.includes(req.method)) {
      return next();
    }
    
    const token = tokenLocation === 'body' 
      ? req.body?._csrf || req.headers['x-csrf-token']
      : req.headers['x-csrf-token'];
    
    if (!token) {
      return res.status(403).json({
        success: false,
        error: 'CSRF token missing'
      });
    }
    
    if (req.user && req.session) {
      const validation = await validateToken(req.user.id, req.session.id, token);
      
      if (!validation.valid) {
        return res.status(403).json({
          success: false,
          error: `CSRF validation failed: ${validation.reason}`
        });
      }
      
      req.csrfValid = true;
    } else if (req.user) {
      const sessionId = req.user.id;
      const validation = await validateToken(req.user.id, sessionId, token);
      
      if (!validation.valid) {
        return res.status(403).json({
          success: false,
          error: `CSRF validation failed: ${validation.reason}`
        });
      }
      
      req.csrfValid = true;
    }
    
    next();
  };
}

function csrfTokenGenerator(options = {}) {
  const {
    secret = CSRF_SECRET,
    expiresIn = TOKEN_EXPIRY,
    tokenLength = TOKEN_LENGTH
  } = options;
  
  return (req, res, next) => {
    if (!req.csrfToken) {
      const randomPart = crypto.randomBytes(tokenLength).toString('hex');
      const signature = crypto
        .createHmac('sha256', secret)
        .update(randomPart)
        .digest('hex');
      
      req.csrfToken = `${randomPart}.${signature}`;
      req.csrfTokenExpiry = Date.now() + expiresIn;
      
      res.cookie('csrf-token', req.csrfToken, {
        httpOnly: false,
        secure: process.env.NODE_ENV === 'production',
        sameSite: 'strict',
        maxAge: expiresIn
      });
    }
    
    next();
  };
}

function checkReferer(options = {}) {
  const {
    allowedOrigins = [],
    allowEmpty = false
  } = options;
  
  return (req, res, next) => {
    const referer = req.headers.referer || req.headers.referrer;
    
    if (!referer && !allowEmpty) {
      return res.status(403).json({
        success: false,
        error: 'Missing referer header'
      });
    }
    
    if (referer) {
      try {
        const refererUrl = new URL(referer);
        const requestUrl = new URL(req.originalUrl, `https://${req.headers.host}`);
        
        const isAllowed = allowedOrigins.length === 0 || 
          allowedOrigins.some(origin => {
            try {
              return new URL(origin).hostname === refererUrl.hostname;
            } catch {
              return false;
            }
          });
        
        if (!isAllowed && refererUrl.hostname !== requestUrl.hostname) {
          return res.status(403).json({
            success: false,
            error: 'Invalid referer'
          });
        }
      } catch (error) {
        return res.status(403).json({
          success: false,
          error: 'Invalid referer format'
        });
      }
    }
    
    next();
  };
}

function checkOrigin(options = {}) {
  const {
    allowedOrigins = [],
    allowEmpty = false
  } = options;
  
  return (req, res, next) => {
    const origin = req.headers.origin;
    
    if (!origin && !allowEmpty) {
      return res.status(403).json({
        success: false,
        error: 'Missing origin header'
      });
    }
    
    if (origin) {
      try {
        const originUrl = new URL(origin);
        
        const isAllowed = allowedOrigins.length === 0 ||
          allowedOrigins.includes(origin) ||
          allowedOrigins.includes(originUrl.hostname);
        
        if (!isAllowed && process.env.NODE_ENV === 'production') {
          return res.status(403).json({
            success: false,
            error: 'Invalid origin'
          });
        }
      } catch (error) {
        return res.status(403).json({
          success: false,
          error: 'Invalid origin format'
        });
      }
    }
    
    next();
  };
}

function doubleSubmitCookie() {
  return (req, res, next) => {
    const cookieToken = req.cookies?.csrf_token;
    const headerToken = req.headers['x-csrf-token'] || req.body?._csrf;
    
    if (req.method === 'GET' || req.method === 'HEAD' || req.method === 'OPTIONS') {
      const newToken = createCsrfToken();
      res.cookie('csrf_token', newToken, {
        httpOnly: false,
        secure: process.env.NODE_ENV === 'production',
        sameSite: 'strict'
      });
      req.csrfToken = newToken;
      return next();
    }
    
    if (!cookieToken || !headerToken) {
      return res.status(403).json({
        success: false,
        error: 'CSRF token missing'
      });
    }
    
    if (cookieToken !== headerToken) {
      return res.status(403).json({
        success: false,
        error: 'CSRF token mismatch'
      });
    }
    
    req.csrfValid = true;
    next();
  };
}

function createCsrfMiddleware(options = {}) {
  return [
    csrfTokenGenerator(options),
    csrfProtection(options),
    checkReferer(options.referer),
    checkOrigin(options.origin)
  ].filter(Boolean);
}

module.exports = {
  generateToken,
  validateToken,
  invalidateToken,
  invalidateAllTokens,
  cleanupExpiredTokens,
  createCsrfToken,
  verifyCsrfToken,
  generateSignedToken,
  csrfProtection,
  csrfTokenGenerator,
  checkReferer,
  checkOrigin,
  doubleSubmitCookie,
  createCsrfMiddleware,
  CSRF_SECRET,
  TOKEN_LENGTH,
  TOKEN_EXPIRY
};
