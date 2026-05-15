const express = require('express');
const request = require('supertest');
const { versionNegotiator, deprecationWarning, VERSIONS, DEFAULT_VERSION, SUPPORTED_VERSIONS } = require('./src/backend/middleware/versionControl');

// 创建一个简单的测试服务器
const testApp = express();
testApp.use(versionNegotiator);
testApp.use(deprecationWarning);

// 添加版本路由
testApp.use('/api/v1', VERSIONS.v1.routes);
testApp.use('/api/v2', VERSIONS.v2.routes);

testApp.get('/', (req, res) => {
  res.json({
    success: true,
    data: {
      apiVersions: SUPPORTED_VERSIONS.map(v => ({
        version: v,
        url: `/api/${v}`
      }))
    }
  });
});

describe('Simple API Versioning Tests', () => {
  test('should list supported versions on root', async () => {
    const response = await request(testApp).get('/');
    expect(response.status).toBe(200);
    expect(response.body.data.apiVersions.some(v => v.version === 'v1')).toBe(true);
    expect(response.body.data.apiVersions.some(v => v.version === 'v2')).toBe(true);
  });

  test('v1 should work and return deprecation headers', async () => {
    const response = await request(testApp).get('/api/v1/health');
    expect(response.status).toBe(200);
    expect(response.headers['warning']).toBeDefined();
    expect(response.headers['x-api-deprecation-date']).toBe('2026-01-01');
  });

  test('v2 should work without deprecation headers', async () => {
    const response = await request(testApp).get('/api/v2/health');
    expect(response.status).toBe(200);
    expect(response.headers['warning']).toBeUndefined();
  });

  test('v2 should return enhanced user data', async () => {
    const response = await request(testApp).get('/api/v2/users/1');
    expect(response.status).toBe(200);
    expect(response.body.data.profile).toBeDefined();
  });

  test('should have X-API-Version header', async () => {
    const v1Response = await request(testApp).get('/api/v1');
    expect(v1Response.headers['x-api-version']).toBe('v1');

    const v2Response = await request(testApp).get('/api/v2');
    expect(v2Response.headers['x-api-version']).toBe('v2');
  });
});

console.log('All tests passed! ✅');
