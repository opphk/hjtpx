const express = require('express');

const { OAUTH_CONFIG } = require('../config');
const PKCE = require('../pkce');
const OAuthService = require('../services/oauthService');

const router = express.Router();

router.get('/authorize', async (req, res) => {
  try {
    const {
      response_type,
      client_id,
      redirect_uri,
      scope,
      state,
      code_challenge,
      code_challenge_method,
      nonce
    } = req.query;

    if (response_type !== 'code') {
      return res.status(400).json({
        error: 'unsupported_response_type',
        error_description: 'Only "code" response type is supported'
      });
    }

    if (!client_id) {
      return res.status(400).json({
        error: 'invalid_request',
        error_description: 'client_id is required'
      });
    }

    if (!redirect_uri) {
      return res.status(400).json({
        error: 'invalid_request',
        error_description: 'redirect_uri is required'
      });
    }

    const validRedirectUris = (process.env.OAUTH_REDIRECT_URIS || '').split(',');
    if (!validRedirectUris.includes(redirect_uri)) {
      return res.status(400).json({
        error: 'invalid_request',
        error_description: 'Invalid redirect_uri'
      });
    }

    if (code_challenge && !code_challenge_method) {
      return res.status(400).json({
        error: 'invalid_request',
        error_description: 'code_challenge_method is required when code_challenge is present'
      });
    }

    if (code_challenge_method && !OAUTH_CONFIG.pkceMethods.includes(code_challenge_method)) {
      return res.status(400).json({
        error: 'invalid_request',
        error_description: 'Only S256 code_challenge_method is supported'
      });
    }

    if (code_challenge && !PKCE.validateCodeChallenge(code_challenge)) {
      return res.status(400).json({
        error: 'invalid_request',
        error_description: 'Invalid code_challenge format'
      });
    }

    const requestedScope = scope || OAUTH_CONFIG.defaultScope;
    const scopes = requestedScope.split(' ');
    const validScopes = scopes.every(s => OAUTH_CONFIG.supportedScopes.includes(s));
    if (!validScopes) {
      return res.status(400).json({
        error: 'invalid_scope',
        error_description: 'One or more requested scopes are not supported'
      });
    }

    if (!req.user) {
      return res.redirect(`/login?redirect_uri=${encodeURIComponent(req.originalUrl)}`);
    }

    const authCode = OAuthService.generateAuthorizationCode();
    const codeData = {
      user: {
        id: req.user.id,
        email: req.user.email,
        name: req.user.name,
        role: req.user.role
      },
      clientId: client_id,
      redirectUri: redirect_uri,
      scope: requestedScope,
      codeChallenge: code_challenge,
      codeChallengeMethod: code_challenge_method,
      nonce,
      createdAt: Date.now()
    };

    await OAuthService.storeAuthorizationCode(authCode, codeData);

    const separator = redirect_uri.includes('?') ? '&' : '?';
    const redirectUrl = `${redirect_uri}${separator}code=${authCode}${state ? `&state=${state}` : ''}`;

    res.redirect(redirectUrl);
  } catch (error) {
    console.error('Authorization error:', error);
    res.status(500).json({
      error: 'server_error',
      error_description: 'An error occurred during authorization'
    });
  }
});

router.post('/authorize', async (req, res) => {
  const { grant_type } = req.body;

  if (grant_type !== 'authorization_code') {
    return res.status(400).json({
      error: 'unsupported_grant_type',
      error_description: 'Only authorization_code grant is supported at this endpoint'
    });
  }

  return res.status(405).json({
    error: 'method_not_allowed',
    error_description: 'POST authorization is not supported'
  });
});

module.exports = router;
