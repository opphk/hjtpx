const express = require('express');

const OAuthService = require('../services/oauthService');

const router = express.Router();

router.post('/revoke', async (req, res) => {
  try {
    const { token, token_type_hint } = req.body;

    if (!token) {
      return res.status(400).json({
        error: 'invalid_request',
        error_description: 'token is required'
      });
    }

    let success = false;
    let hint = token_type_hint;

    if (!hint) {
      const decoded = decodeToken(token);
      hint = decoded?.type === 'refresh_token' ? 'refresh_token' : 'access_token';
    }

    success = await OAuthService.revokeToken(token, hint);

    res.json({
      revoked: success,
      hint
    });
  } catch (error) {
    console.error('Token revocation error:', error);
    res.status(200).json({
      revoked: false,
      error: 'The token may have already been revoked or is invalid'
    });
  }
});

router.post('/introspect', async (req, res) => {
  try {
    const { token, token_type_hint } = req.body;

    if (!token) {
      return res.status(400).json({
        error: 'invalid_request',
        error_description: 'token is required'
      });
    }

    let isActive = false;
    let tokenData = null;
    let hint = token_type_hint;

    if (!hint) {
      const decoded = decodeToken(token);
      hint = decoded?.type === 'refresh_token' ? 'refresh_token' : 'access_token';
    }

    try {
      if (hint === 'access_token') {
        tokenData = await OAuthService.verifyAccessToken(token);
        isActive = true;
      } else if (hint === 'refresh_token') {
        tokenData = await OAuthService.verifyRefreshToken(token);
        isActive = true;
      }
    } catch (error) {
      isActive = false;
    }

    res.json({
      active: isActive,
      token_type: hint,
      ...(tokenData && {
        sub: tokenData.sub,
        client_id: tokenData.clientId,
        scope: tokenData.scope,
        exp: tokenData.exp
      })
    });
  } catch (error) {
    console.error('Token introspection error:', error);
    res.status(200).json({
      active: false
    });
  }
});

function decodeToken(token) {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = Buffer.from(parts[1], 'base64').toString('utf-8');
    return JSON.parse(payload);
  } catch {
    return null;
  }
}

module.exports = router;
