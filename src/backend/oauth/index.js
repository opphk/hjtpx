const express = require('express');

const authRoutes = require('./routes/auth');
const authorizationRoutes = require('./routes/authorization');
const revokeRoutes = require('./routes/revoke');
const tokenRoutes = require('./routes/token');
const OAuthService = require('./services/oauthService');

const router = express.Router();

router.use('/', authorizationRoutes);
router.use('/', tokenRoutes);
router.use('/', revokeRoutes);
router.use('/auth', authRoutes);

router.get('/.well-known/openid-configuration', (req, res) => {
  const baseUrl = process.env.OAUTH_BASE_URL || `${req.protocol}://${req.get('host')}`;

  res.json({
    issuer: process.env.OAUTH_ISSUER || baseUrl,
    authorization_endpoint: `${baseUrl}/oauth/authorize`,
    token_endpoint: `${baseUrl}/oauth/token`,
    token_revocation_endpoint: `${baseUrl}/oauth/revoke`,
    token_introspection_endpoint: `${baseUrl}/oauth/introspect`,
    userinfo_endpoint: `${baseUrl}/oauth/userinfo`,
    jwks_uri: `${baseUrl}/.well-known/jwks.json`,
    response_types_supported: ['code'],
    grant_types_supported: ['authorization_code', 'refresh_token', 'client_credentials'],
    subject_types_supported: ['public'],
    id_token_signing_alg_values_supported: ['RS256'],
    code_challenge_methods_supported: ['S256'],
    scopes_supported: ['openid', 'profile', 'email', 'read', 'write'],
    token_endpoint_auth_methods_supported: ['none', 'client_secret_basic', 'client_secret_post'],
    claims_supported: ['sub', 'name', 'email', 'email_verified', 'picture', 'role']
  });
});

router.get('/userinfo', async (req, res) => {
  try {
    const authHeader = req.headers.authorization;

    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      return res.status(401).json({
        error: 'invalid_token',
        error_description: 'Bearer token is required'
      });
    }

    const token = authHeader.substring(7);
    const userInfo = await OAuthService.getUserInfo(token);

    res.json(userInfo);
  } catch (error) {
    res.status(401).json({
      error: 'invalid_token',
      error_description: error.message
    });
  }
});

router.get('/.well-known/jwks.json', (req, res) => {
  const publicKey = process.env.OAUTH_PUBLIC_KEY || '';
  const keyId = process.env.OAUTH_KEY_ID || 'default-key-1';

  if (!publicKey) {
    return res.status(501).json({
      error: 'not_implemented',
      error_description: 'JWKS not configured'
    });
  }

  res.json({
    keys: [
      {
        kty: 'RSA',
        use: 'sig',
        kid: keyId,
        alg: 'RS256',
        n: publicKey
      }
    ]
  });
});

module.exports = router;
