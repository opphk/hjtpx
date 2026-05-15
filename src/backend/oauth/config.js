const OAUTH_CONFIG = {
  authorizationCodeLength: 64,
  authorizationCodeExpiry: 600,
  accessTokenExpiry: 3600,
  refreshTokenExpiry: 604800,
  defaultScope: 'openid profile email',
  supportedScopes: ['openid', 'profile', 'email', 'read', 'write'],
  tokenEndpointAuthMethods: ['none', 'client_secret_basic', 'client_secret_post'],
  responseTypes: ['code', 'token', 'id_token'],
  grantTypes: ['authorization_code', 'refresh_token', 'client_credentials', 'password'],
  pkceMethods: ['S256'],
  tokenRotation: true,
  reuseRefreshToken: false
};

const PROVIDER_CONFIG = {
  github: {
    name: 'GitHub',
    authorizationURL: 'https://github.com/login/oauth/authorize',
    tokenURL: 'https://github.com/login/oauth/access_token',
    apiURL: 'https://api.github.com',
    scopeSeparator: ' ',
    callbackURL: '/auth/github/callback'
  },
  google: {
    name: 'Google',
    authorizationURL: 'https://accounts.google.com/o/oauth2/v2/auth',
    tokenURL: 'https://oauth2.googleapis.com/token',
    apiURL: 'https://www.googleapis.com/oauth2/v2',
    scopeSeparator: ' ',
    callbackURL: '/auth/google/callback'
  }
};

module.exports = {
  OAUTH_CONFIG,
  PROVIDER_CONFIG
};
