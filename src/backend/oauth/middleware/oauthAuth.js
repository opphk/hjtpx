const OAuthService = require('../services/oauthService');

async function requireOAuthToken(req, res, next) {
  try {
    const authHeader = req.headers.authorization;

    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      return res.status(401).json({
        success: false,
        error: 'No token provided',
        error_description: 'Bearer token is required'
      });
    }

    const token = authHeader.substring(7);

    const user = await OAuthService.verifyAccessToken(token);

    req.user = {
      id: user.sub,
      email: user.email,
      name: user.name,
      role: user.role,
      scope: user.scope
    };
    req.oauthToken = token;

    next();
  } catch (error) {
    return res.status(401).json({
      success: false,
      error: 'invalid_token',
      error_description: error.message
    });
  }
}

function requireScope(requiredScope) {
  return (req, res, next) => {
    if (!req.user || !req.user.scope) {
      return res.status(403).json({
        success: false,
        error: 'insufficient_scope',
        error_description: `Required scope: ${requiredScope}`
      });
    }

    const tokenScopes = req.user.scope.split(' ');
    if (!tokenScopes.includes(requiredScope)) {
      return res.status(403).json({
        success: false,
        error: 'insufficient_scope',
        error_description: `Required scope: ${requiredScope}`
      });
    }

    next();
  };
}

async function optionalOAuthToken(req, res, next) {
  const authHeader = req.headers.authorization;

  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return next();
  }

  try {
    const token = authHeader.substring(7);
    const user = await OAuthService.verifyAccessToken(token);

    req.user = {
      id: user.sub,
      email: user.email,
      name: user.name,
      role: user.role,
      scope: user.scope
    };
    req.oauthToken = token;
  } catch (error) {}

  next();
}

module.exports = {
  requireOAuthToken,
  requireScope,
  optionalOAuthToken
};
