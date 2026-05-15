const express = require('express');

const authMiddleware = require('../../middleware/auth');
const githubProvider = require('../providers/github');
const googleProvider = require('../providers/google');
const OAuthService = require('../services/oauthService');

const router = express.Router();

router.get('/github', async (req, res) => {
  try {
    const { state } = req.query;

    const savedState = req.session?.oauthState || req.cookies?.oauthState;
    if (state && savedState && state !== savedState) {
      return res.status(400).json({
        success: false,
        error: 'Invalid OAuth state'
      });
    }

    const authUrl = githubProvider.getAuthorizationURL(state || githubProvider.generateState());

    res.redirect(authUrl);
  } catch (error) {
    console.error('GitHub auth initiation error:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to initiate GitHub authentication'
    });
  }
});

router.get('/github/callback', async (req, res) => {
  try {
    const { code, state, error } = req.query;

    if (error) {
      return res.redirect(`/login?error=${encodeURIComponent(error)}`);
    }

    if (!code) {
      return res.redirect('/login?error=no_code');
    }

    const tokenData = await githubProvider.getAccessToken(code);
    const profile = await githubProvider.getUserProfile(tokenData.access_token);

    const user = await OAuthService.registerOAuthUser(profile.provider, profile.providerId, {
      ...profile,
      refreshToken: tokenData.refresh_token,
      expiresAt: tokenData.expires_at
    });

    const accessToken = OAuthService.generateAccessToken(user);
    const refreshToken = OAuthService.generateRefreshToken(user, 'github');

    const redirectUrl = process.env.OAUTH_SUCCESS_REDIRECT_URL || '/dashboard';
    res.redirect(`${redirectUrl}?access_token=${accessToken}&refresh_token=${refreshToken}`);
  } catch (error) {
    console.error('GitHub callback error:', error);
    res.redirect('/login?error=oauth_failed');
  }
});

router.get('/google', async (req, res) => {
  try {
    const { state } = req.query;

    const savedState = req.session?.oauthState || req.cookies?.oauthState;
    if (state && savedState && state !== savedState) {
      return res.status(400).json({
        success: false,
        error: 'Invalid OAuth state'
      });
    }

    const authUrl = googleProvider.getAuthorizationURL(state || googleProvider.generateState());

    res.redirect(authUrl);
  } catch (error) {
    console.error('Google auth initiation error:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to initiate Google authentication'
    });
  }
});

router.get('/google/callback', async (req, res) => {
  try {
    const { code, state, error, error_description } = req.query;

    if (error) {
      return res.redirect(`/login?error=${encodeURIComponent(error_description || error)}`);
    }

    if (!code) {
      return res.redirect('/login?error=no_code');
    }

    const tokenData = await googleProvider.getAccessToken(code);
    const profile = await googleProvider.getUserProfile(tokenData.access_token, tokenData.id_token);

    const user = await OAuthService.registerOAuthUser(profile.provider, profile.providerId, {
      ...profile,
      refreshToken: tokenData.refresh_token,
      expiresAt: tokenData.expires_at
    });

    const accessToken = OAuthService.generateAccessToken(user);
    const refreshToken = OAuthService.generateRefreshToken(user, 'google');

    const redirectUrl = process.env.OAUTH_SUCCESS_REDIRECT_URL || '/dashboard';
    res.redirect(`${redirectUrl}?access_token=${accessToken}&refresh_token=${refreshToken}`);
  } catch (error) {
    console.error('Google callback error:', error);
    res.redirect('/login?error=oauth_failed');
  }
});

router.post('/disconnect/:provider', authMiddleware, async (req, res) => {
  try {
    const { provider } = req.params;

    if (!['github', 'google'].includes(provider)) {
      return res.status(400).json({
        success: false,
        error: 'Unsupported provider'
      });
    }

    const result = await OAuthService.disconnectOAuthProvider(req.user.id, provider);

    res.json({
      success: true,
      message: `Disconnected from ${provider}`
    });
  } catch (error) {
    console.error('OAuth disconnect error:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to disconnect OAuth provider'
    });
  }
});

router.get('/providers', authMiddleware, async (req, res) => {
  try {
    const result = await OAuthService.getConnectedProviders(req.user.id);

    res.json({
      success: true,
      data: {
        connected: result.connected,
        available: ['github', 'google'].filter(p => !result.connected.includes(p))
      }
    });
  } catch (error) {
    console.error('Get providers error:', error);
    res.status(500).json({
      success: false,
      error: 'Failed to get OAuth providers'
    });
  }
});

module.exports = router;
