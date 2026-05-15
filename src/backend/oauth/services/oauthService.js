const crypto = require('crypto');

const bcrypt = require('bcryptjs');
const jwt = require('jsonwebtoken');

const pool = require('../../../config/database/db');
const cacheService = require('../../services/cacheService');
const { OAUTH_CONFIG } = require('../config');

const JWT_SECRET = process.env.JWT_SECRET || 'hjtpx-oauth-secret-key-change-in-production';
const JWT_EXPIRES_IN = process.env.JWT_EXPIRES_IN || '1h';
const REFRESH_TOKEN_EXPIRES_IN = process.env.REFRESH_TOKEN_EXPIRES_IN || '7d';

class OAuthService {
  static generateAuthorizationCode() {
    return crypto.randomBytes(OAUTH_CONFIG.authorizationCodeLength).toString('hex');
  }

  static async storeAuthorizationCode(code, data) {
    const key = `oauth:auth_code:${code}`;
    const cacheData = {
      ...data,
      expiresAt: Date.now() + OAUTH_CONFIG.authorizationCodeExpiry * 1000
    };
    await cacheService.set(key, JSON.stringify(cacheData), OAUTH_CONFIG.authorizationCodeExpiry);
  }

  static async getAuthorizationCode(code) {
    const key = `oauth:auth_code:${code}`;
    const data = await cacheService.get(key);
    if (!data) return null;
    return JSON.parse(data);
  }

  static async deleteAuthorizationCode(code) {
    const key = `oauth:auth_code:${code}`;
    await cacheService.delete(key);
  }

  static generateAccessToken(user, scope = OAUTH_CONFIG.defaultScope) {
    const payload = {
      sub: user.id || user.userId,
      email: user.email,
      name: user.name,
      role: user.role,
      scope,
      type: 'access_token',
      iat: Math.floor(Date.now() / 1000)
    };

    return jwt.sign(payload, JWT_SECRET, { expiresIn: JWT_EXPIRES_IN });
  }

  static generateRefreshToken(user, clientId = 'default') {
    const payload = {
      sub: user.id || user.userId,
      email: user.email,
      clientId,
      type: 'refresh_token',
      jti: crypto.randomBytes(16).toString('hex'),
      iat: Math.floor(Date.now() / 1000)
    };

    return jwt.sign(payload, JWT_SECRET, { expiresIn: REFRESH_TOKEN_EXPIRES_IN });
  }

  static generateIdToken(user) {
    const payload = {
      sub: user.id || user.userId,
      email: user.email,
      name: user.name,
      picture: user.picture,
      aud: process.env.OAUTH_CLIENT_ID || 'hjtpx-client',
      iss: process.env.OAUTH_ISSUER || 'https://hjtpx.example.com',
      iat: Math.floor(Date.now() / 1000),
      exp: Math.floor(Date.now() / 1000) + 3600
    };

    return jwt.sign(payload, JWT_SECRET, { algorithm: 'RS256' });
  }

  static async verifyAccessToken(token) {
    try {
      const decoded = jwt.verify(token, JWT_SECRET);
      if (decoded.type !== 'access_token') {
        throw new Error('Invalid token type');
      }
      const isRevoked = await this.isTokenRevoked(decoded.jti || decoded.sub);
      if (isRevoked) {
        throw new Error('Token has been revoked');
      }
      return decoded;
    } catch (error) {
      throw new Error(`Token verification failed: ${error.message}`);
    }
  }

  static async verifyRefreshToken(token) {
    try {
      const decoded = jwt.verify(token, JWT_SECRET);
      if (decoded.type !== 'refresh_token') {
        throw new Error('Invalid token type');
      }
      const isRevoked = await this.isTokenRevoked(decoded.jti);
      if (isRevoked) {
        throw new Error('Refresh token has been revoked');
      }
      return decoded;
    } catch (error) {
      throw new Error(`Refresh token verification failed: ${error.message}`);
    }
  }

  static async revokeToken(token, tokenTypeHint = 'access_token') {
    try {
      const decoded = jwt.decode(token, { complete: true });
      if (!decoded) {
        throw new Error('Invalid token format');
      }

      const jti = decoded.payload.jti || decoded.payload.sub;
      await this.addToBlacklist(jti, decoded.payload.exp);

      await this.logTokenRevocation(jti, tokenTypeHint);

      return true;
    } catch (error) {
      console.error('Token revocation error:', error);
      return false;
    }
  }

  static async isTokenRevoked(jti) {
    const key = `oauth:blacklist:${jti}`;
    const isBlacklisted = await cacheService.get(key);
    return !!isBlacklisted;
  }

  static async addToBlacklist(jti, exp) {
    const key = `oauth:blacklist:${jti}`;
    const ttl = Math.max(0, exp - Math.floor(Date.now() / 1000));
    if (ttl > 0) {
      await cacheService.set(key, 'revoked', ttl);
    }
  }

  static async logTokenRevocation(jti, tokenType) {
    try {
      await pool.query(
        `INSERT INTO oauth_token_revocations (token_jti, token_type, revoked_at)
         VALUES ($1, $2, NOW())
         ON CONFLICT (token_jti) DO NOTHING`,
        [jti, tokenType]
      );
    } catch (error) {
      console.error('Failed to log token revocation:', error);
    }
  }

  static async refreshAccessToken(refreshToken) {
    const decoded = await this.verifyRefreshToken(refreshToken);

    const userResult = await pool.query('SELECT id, email, name, role FROM users WHERE id = $1', [
      decoded.sub
    ]);

    if (userResult.rows.length === 0) {
      throw new Error('User not found');
    }

    const user = userResult.rows[0];

    if (OAUTH_CONFIG.tokenRotation && !OAUTH_CONFIG.reuseRefreshToken) {
      await this.revokeToken(refreshToken, 'refresh_token');
    }

    const newAccessToken = this.generateAccessToken(user, decoded.scope);
    const newRefreshToken = this.generateRefreshToken(user, decoded.clientId);

    return {
      access_token: newAccessToken,
      token_type: 'Bearer',
      expires_in: 3600,
      refresh_token: newRefreshToken
    };
  }

  static async exchangeCodeForTokens(code, codeVerifier, redirectUri, clientId = 'default') {
    const authCodeData = await this.getAuthorizationCode(code);

    if (!authCodeData) {
      throw new Error('Invalid or expired authorization code');
    }

    if (authCodeData.redirectUri !== redirectUri) {
      throw new Error('Redirect URI mismatch');
    }

    if (authCodeData.clientId !== clientId) {
      throw new Error('Client ID mismatch');
    }

    if (authCodeData.expiresAt < Date.now()) {
      await this.deleteAuthorizationCode(code);
      throw new Error('Authorization code expired');
    }

    if (authCodeData.pkce) {
      const PKCE = require('../pkce');
      const isValid = await PKCE.verifyCodeChallenge(codeVerifier, authCodeData.codeChallenge);
      if (!isValid) {
        throw new Error('PKCE verification failed');
      }
    }

    await this.deleteAuthorizationCode(code);

    const user = authCodeData.user;

    const accessToken = this.generateAccessToken(user, authCodeData.scope);
    const refreshToken = this.generateRefreshToken(user, clientId);

    return {
      access_token: accessToken,
      token_type: 'Bearer',
      expires_in: 3600,
      refresh_token: refreshToken,
      scope: authCodeData.scope,
      ...(authCodeData.nonce && {
        id_token: this.generateIdToken({ ...user, nonce: authCodeData.nonce })
      })
    };
  }

  static async registerOAuthUser(provider, providerId, profile) {
    const email = profile.email || `${provider}_${providerId}@oauth.hjtpx.local`;
    const name = profile.name || profile.username || 'OAuth User';

    const existingUser = await pool.query(
      'SELECT id, email, name, role FROM users WHERE oauth_provider = $1 AND oauth_provider_id = $2',
      [provider, providerId]
    );

    if (existingUser.rows.length > 0) {
      await pool.query(
        'UPDATE users SET oauth_token = $1, oauth_refresh_token = $2, oauth_token_expires = $3 WHERE id = $4',
        [profile.accessToken, profile.refreshToken, profile.expiresAt, existingUser.rows[0].id]
      );
      return existingUser.rows[0];
    }

    const result = await pool.query(
      `INSERT INTO users (email, name, password, role, oauth_provider, oauth_provider_id, oauth_token, oauth_refresh_token, oauth_token_expires)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
       RETURNING id, email, name, role`,
      [
        email,
        name,
        await bcrypt.hash(crypto.randomBytes(32).toString('hex'), 10),
        'user',
        provider,
        providerId,
        profile.accessToken,
        profile.refreshToken,
        profile.expiresAt
      ]
    );

    return result.rows[0];
  }

  static async getUserInfo(accessToken) {
    const user = await this.verifyAccessToken(accessToken);
    return {
      sub: user.sub,
      email: user.email,
      name: user.name,
      role: user.role,
      scope: user.scope
    };
  }
}

module.exports = OAuthService;
