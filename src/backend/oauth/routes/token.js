const express = require('express');

const { OAUTH_CONFIG } = require('../config');
const PKCE = require('../pkce');
const OAuthService = require('../services/oauthService');

const router = express.Router();

router.post('/token', async (req, res) => {
  try {
    const { grant_type, code, redirect_uri, client_id, code_verifier, refresh_token } = req.body;

    if (!grant_type) {
      return res.status(400).json({
        error: 'invalid_request',
        error_description: 'grant_type is required'
      });
    }

    switch (grant_type) {
      case 'authorization_code':
        return handleAuthorizationCodeGrant(req, res, {
          code,
          redirect_uri,
          client_id,
          code_verifier
        });

      case 'refresh_token':
        return handleRefreshTokenGrant(req, res, { refresh_token });

      case 'client_credentials':
        return handleClientCredentialsGrant(req, res);

      default:
        return res.status(400).json({
          error: 'unsupported_grant_type',
          error_description: `Grant type "${grant_type}" is not supported`
        });
    }
  } catch (error) {
    console.error('Token endpoint error:', error);
    res.status(500).json({
      error: 'server_error',
      error_description: 'An error occurred during token issuance'
    });
  }
});

async function handleAuthorizationCodeGrant(req, res, params) {
  const { code, redirect_uri, client_id, code_verifier } = params;

  if (!code) {
    return res.status(400).json({
      error: 'invalid_request',
      error_description: 'code is required for authorization_code grant'
    });
  }

  if (!redirect_uri) {
    return res.status(400).json({
      error: 'invalid_request',
      error_description: 'redirect_uri is required for authorization_code grant'
    });
  }

  try {
    const authCodeData = await OAuthService.getAuthorizationCode(code);

    if (!authCodeData) {
      return res.status(400).json({
        error: 'invalid_grant',
        error_description: 'Invalid or expired authorization code'
      });
    }

    if (authCodeData.redirectUri !== redirect_uri) {
      return res.status(400).json({
        error: 'invalid_grant',
        error_description: 'redirect_uri mismatch'
      });
    }

    if (authCodeData.clientId !== client_id) {
      return res.status(400).json({
        error: 'invalid_grant',
        error_description: 'client_id mismatch'
      });
    }

    if (authCodeData.codeChallenge) {
      if (!code_verifier) {
        return res.status(400).json({
          error: 'invalid_request',
          error_description: 'code_verifier is required for PKCE flow'
        });
      }

      if (!PKCE.validateCodeVerifier(code_verifier)) {
        return res.status(400).json({
          error: 'invalid_request',
          error_description: 'Invalid code_verifier format'
        });
      }

      const isValid = await PKCE.verifyCodeChallenge(code_verifier, authCodeData.codeChallenge);

      if (!isValid) {
        return res.status(400).json({
          error: 'invalid_grant',
          error_description: 'PKCE verification failed'
        });
      }
    }

    const tokens = await OAuthService.exchangeCodeForTokens(
      code,
      code_verifier,
      redirect_uri,
      client_id
    );

    res.json({
      access_token: tokens.access_token,
      token_type: tokens.token_type,
      expires_in: tokens.expires_in,
      refresh_token: tokens.refresh_token,
      scope: tokens.scope,
      ...(tokens.id_token && { id_token: tokens.id_token })
    });
  } catch (error) {
    console.error('Authorization code grant error:', error);
    res.status(400).json({
      error: 'invalid_grant',
      error_description: error.message
    });
  }
}

async function handleRefreshTokenGrant(req, res, params) {
  const { refresh_token } = params;

  if (!refresh_token) {
    return res.status(400).json({
      error: 'invalid_request',
      error_description: 'refresh_token is required for refresh_token grant'
    });
  }

  try {
    const tokens = await OAuthService.refreshAccessToken(refresh_token);

    res.json({
      access_token: tokens.access_token,
      token_type: tokens.token_type,
      expires_in: tokens.expires_in,
      refresh_token: tokens.refresh_token
    });
  } catch (error) {
    console.error('Refresh token grant error:', error);
    res.status(400).json({
      error: 'invalid_grant',
      error_description: error.message
    });
  }
}

async function handleClientCredentialsGrant(req, res) {
  const { client_id, client_secret } = req.body;

  const validClientId = process.env.OAUTH_CLIENT_ID || 'hjtpx-client';
  const validClientSecret = process.env.OAUTH_CLIENT_SECRET;

  if (!validClientSecret) {
    return res.status(501).json({
      error: 'server_error',
      error_description: 'Client credentials grant is not configured'
    });
  }

  if (client_id !== validClientId || client_secret !== validClientSecret) {
    return res.status(401).json({
      error: 'invalid_client',
      error_description: 'Invalid client credentials'
    });
  }

  const user = {
    id: 'service-account',
    email: 'service@hjtpx.local',
    name: 'Service Account',
    role: 'service'
  };

  const accessToken = OAuthService.generateAccessToken(user, 'read write');

  res.json({
    access_token: accessToken,
    token_type: 'Bearer',
    expires_in: 3600,
    scope: 'read write'
  });
}

module.exports = router;
