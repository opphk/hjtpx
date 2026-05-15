const request = require('supertest');
const express = require('express');
const jwt = require('jsonwebtoken');

jest.mock('../../src/backend/oauth/services/oauthService', () => ({
  generateAuthorizationCode: jest.fn().mockReturnValue('test_auth_code_123'),
  storeAuthorizationCode: jest.fn().mockResolvedValue(true),
  getAuthorizationCode: jest.fn(),
  deleteAuthorizationCode: jest.fn().mockResolvedValue(true),
  generateAccessToken: jest.fn().mockReturnValue('test_access_token'),
  generateRefreshToken: jest.fn().mockReturnValue('test_refresh_token'),
  verifyAccessToken: jest.fn(),
  verifyRefreshToken: jest.fn(),
  revokeToken: jest.fn().mockResolvedValue(true),
  exchangeCodeForTokens: jest.fn(),
  refreshAccessToken: jest.fn()
}));

jest.mock('../../src/backend/services/cacheService', () => ({
  get: jest.fn(),
  set: jest.fn(),
  delete: jest.fn()
}));

const mockOAuthService = require('../../src/backend/oauth/services/oauthService');
const authorizationRoutes = require('../../src/backend/oauth/routes/authorization');
const tokenRoutes = require('../../src/backend/oauth/routes/token');
const revokeRoutes = require('../../src/backend/oauth/routes/revoke');

describe('OAuth Endpoints', () => {
  let app;

  beforeEach(() => {
    app = express();
    app.use(express.json());
    app.use((req, res, next) => {
      req.user = { id: 'test_user_id', email: 'test@example.com', name: 'Test User', role: 'user' };
      next();
    });
    app.use('/oauth', authorizationRoutes);
    app.use('/oauth', tokenRoutes);
    app.use('/oauth', revokeRoutes);
    jest.clearAllMocks();
  });

  describe('GET /oauth/authorize', () => {
    test('should return error for missing response_type', async () => {
      const response = await request(app)
        .get('/oauth/authorize')
        .query({ client_id: 'test_client', redirect_uri: 'http://localhost/callback' });

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('unsupported_response_type');
    });

    test('should return error for missing client_id', async () => {
      const response = await request(app)
        .get('/oauth/authorize')
        .query({ response_type: 'code', redirect_uri: 'http://localhost/callback' });

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('invalid_request');
    });

    test('should return error for missing redirect_uri', async () => {
      const response = await request(app)
        .get('/oauth/authorize')
        .query({ response_type: 'code', client_id: 'test_client' });

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('invalid_request');
    });

    test('should return error for unsupported response_type', async () => {
      const response = await request(app)
        .get('/oauth/authorize')
        .query({
          response_type: 'token',
          client_id: 'test_client',
          redirect_uri: 'http://localhost/callback'
        });

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('unsupported_response_type');
    });

    test('should return error for invalid redirect_uri', async () => {
      process.env.OAUTH_REDIRECT_URIS = 'http://localhost/callback';

      const response = await request(app)
        .get('/oauth/authorize')
        .query({
          response_type: 'code',
          client_id: 'test_client',
          redirect_uri: 'http://evil.com/callback'
        });

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('invalid_request');
    });

    test('should return error for invalid PKCE challenge method', async () => {
      process.env.OAUTH_REDIRECT_URIS = 'http://localhost/callback';

      const response = await request(app)
        .get('/oauth/authorize')
        .query({
          response_type: 'code',
          client_id: 'test_client',
          redirect_uri: 'http://localhost/callback',
          code_challenge: 'test_challenge',
          code_challenge_method: 'plain'
        });

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('invalid_request');
    });

    test('should return error when PKCE challenge without method', async () => {
      process.env.OAUTH_REDIRECT_URIS = 'http://localhost/callback';

      const response = await request(app)
        .get('/oauth/authorize')
        .query({
          response_type: 'code',
          client_id: 'test_client',
          redirect_uri: 'http://localhost/callback',
          code_challenge: 'test_challenge'
        });

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('invalid_request');
    });

    test('should validate scope', async () => {
      process.env.OAUTH_REDIRECT_URIS = 'http://localhost/callback';

      const response = await request(app)
        .get('/oauth/authorize')
        .query({
          response_type: 'code',
          client_id: 'test_client',
          redirect_uri: 'http://localhost/callback',
          scope: 'invalid_scope'
        });

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('invalid_scope');
    });
  });

  describe('POST /oauth/token', () => {
    test('should return error for missing grant_type', async () => {
      const response = await request(app)
        .post('/oauth/token')
        .send({});

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('invalid_request');
    });

    test('should return error for unsupported grant_type', async () => {
      const response = await request(app)
        .post('/oauth/token')
        .send({ grant_type: 'password' });

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('unsupported_grant_type');
    });

    test('should handle refresh_token grant', async () => {
      mockOAuthService.refreshAccessToken.mockResolvedValue({
        access_token: 'new_access_token',
        token_type: 'Bearer',
        expires_in: 3600,
        refresh_token: 'new_refresh_token'
      });

      const response = await request(app)
        .post('/oauth/token')
        .send({
          grant_type: 'refresh_token',
          refresh_token: 'test_refresh_token'
        });

      expect(response.status).toBe(200);
      expect(response.body.access_token).toBe('new_access_token');
    });

    test('should require code for authorization_code grant', async () => {
      const response = await request(app)
        .post('/oauth/token')
        .send({
          grant_type: 'authorization_code',
          redirect_uri: 'http://localhost/callback',
          client_id: 'test_client'
        });

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('invalid_request');
    });

    test('should require redirect_uri for authorization_code grant', async () => {
      const response = await request(app)
        .post('/oauth/token')
        .send({
          grant_type: 'authorization_code',
          code: 'test_code',
          client_id: 'test_client'
        });

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('invalid_request');
    });
  });

  describe('POST /oauth/revoke', () => {
    test('should return error for missing token', async () => {
      const response = await request(app)
        .post('/oauth/revoke')
        .send({});

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('invalid_request');
    });

    test('should revoke token successfully', async () => {
      mockOAuthService.revokeToken.mockResolvedValue(true);

      const response = await request(app)
        .post('/oauth/revoke')
        .send({
          token: 'test_token',
          token_type_hint: 'access_token'
        });

      expect(response.status).toBe(200);
      expect(response.body.revoked).toBe(true);
    });
  });

  describe('POST /oauth/introspect', () => {
    test('should return error for missing token', async () => {
      const response = await request(app)
        .post('/oauth/introspect')
        .send({});

      expect(response.status).toBe(400);
      expect(response.body.error).toBe('invalid_request');
    });

    test('should return active status for valid token', async () => {
      mockOAuthService.verifyAccessToken.mockResolvedValue({
        sub: 'test_user',
        scope: 'openid profile',
        exp: Math.floor(Date.now() / 1000) + 3600
      });

      const response = await request(app)
        .post('/oauth/introspect')
        .send({
          token: 'valid_token',
          token_type_hint: 'access_token'
        });

      expect(response.status).toBe(200);
      expect(response.body.active).toBe(true);
    });
  });
});
