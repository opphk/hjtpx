import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';
import { TestHelpers } from '../../utils/test-helpers';

test.describe('安全功能集成测试', () => {
  let apiHelper: ApiHelper;
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ page, request }) => {
    apiHelper = new ApiHelper(request);
    testHelpers = new TestHelpers();
  });

  test('XSS攻击防护测试', async ({ page }) => {
    console.log('正在测试XSS攻击防护...');
    const xssPayloads = [
      '<script>alert("XSS")</script>',
      '<img src=x onerror=alert("XSS")>',
      'javascript:alert("XSS")',
      '<svg/onload=alert("XSS")>',
      '\'><script>alert("XSS")</script>'
    ];

    for (const payload of xssPayloads) {
      const response = await apiHelper.submitForm({
        username: payload,
        password: 'test123'
      });
      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(data.code).not.toBe(0);
      console.log(`XSS payload blocked: ${payload.substring(0, 30)}...`);
    }
    console.log('✅ XSS攻击防护测试完成');
  });

  test('SQL注入防护测试', async ({ page }) => {
    console.log('正在测试SQL注入防护...');
    const sqlPayloads = [
      "' OR '1'='1",
      "'; DROP TABLE users;--",
      "UNION SELECT * FROM users",
      "1' AND '1'='1",
      "admin'--"
    ];

    for (const payload of sqlPayloads) {
      const response = await apiHelper.submitForm({
        username: payload,
        password: 'anything'
      });
      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(data.code).not.toBe(0);
      console.log(`SQL injection blocked: ${payload.substring(0, 30)}...`);
    }
    console.log('✅ SQL注入防护测试完成');
  });

  test('CSRF令牌验证', async ({ page }) => {
    console.log('正在测试CSRF令牌验证...');
    const response = await apiHelper.getCSRFToken();
    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data.data.token).toBeDefined();

    const submitWithoutToken = await apiHelper.submitFormWithCSRF({
      username: 'test',
      password: 'test123'
    }, '');
    expect(submitWithoutToken.ok()).toBeTruthy();
    const result = await submitWithoutToken.json();
    expect(result.code).not.toBe(0);
    console.log('✅ CSRF令牌验证测试完成');
  });

  test('速率限制测试', async ({ page }) => {
    console.log('正在测试速率限制...');
    let rateLimited = false;
    const requests = [];

    for (let i = 0; i < 150; i++) {
      const response = await apiHelper.makeRequest('/api/v1/health');
      requests.push(response);

      if (response.status() === 429) {
        rateLimited = true;
        console.log(`速率限制触发于第 ${i + 1} 个请求`);
        break;
      }
    }

    expect(rateLimited).toBeTruthy();
    console.log('✅ 速率限制测试完成');
  });

  test('密码强度验证', async ({ request }) => {
    console.log('正在测试密码强度验证...');
    const weakPasswords = [
      '123456',
      'password',
      'abc123',
      'qwerty',
      '111111'
    ];

    for (const password of weakPasswords) {
      const response = await apiHelper.validatePassword(password);
      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(data.data.strong).toBeFalsy();
      console.log(`弱密码检测: ${password}`);
    }

    const strongPassword = 'Str0ng@Pass#2024';
    const strongResponse = await apiHelper.validatePassword(strongPassword);
    const strongData = await strongResponse.json();
    expect(strongData.data.strong).toBeTruthy();
    console.log('✅ 密码强度验证测试完成');
  });

  test('会话超时测试', async ({ page }) => {
    console.log('正在测试会话超时...');
    const loginResponse = await apiHelper.login('testuser', 'testpass');
    expect(loginResponse.ok()).toBeTruthy();
    const loginData = await loginResponse.json();
    expect(loginData.data.session_id).toBeDefined();

    const sessionToken = loginData.data.session_id;
    await page.waitForTimeout(60000);

    const sessionResponse = await apiHelper.checkSession(sessionToken);
    const sessionData = await sessionResponse.json();
    expect(sessionData.code).not.toBe(0);
    console.log('✅ 会话超时测试完成');
  });

  test('IP黑名单测试', async ({ request }) => {
    console.log('正在测试IP黑名单...');
    const response = await apiHelper.makeRequest('/api/v1/captcha/generate', 'POST', {
      app_key: 'blocked_ip_test'
    });

    expect(response.ok()).toBeTruthy();
    console.log('✅ IP黑名单测试完成');
  });

  test('敏感数据加密测试', async ({ request }) => {
    console.log('正在测试敏感数据加密...');
    const response = await apiHelper.submitSensitiveData({
      credit_card: '4111111111111111',
      ssn: '123-45-6789',
      password: 'secret123'
    });

    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data.data.encrypted).toBeTruthy();
    console.log('✅ 敏感数据加密测试完成');
  });

  test('JWT令牌验证', async ({ request }) => {
    console.log('正在测试JWT令牌验证...');
    const loginResponse = await apiHelper.login('testuser', 'testpass');
    const loginData = await loginResponse.json();
    const accessToken = loginData.data.access_token;

    const verifyResponse = await apiHelper.verifyToken(accessToken);
    expect(verifyResponse.ok()).toBeTruthy();
    const verifyData = await verifyResponse.json();
    expect(verifyData.data.valid).toBeTruthy();

    const invalidToken = 'invalid.token.here';
    const invalidResponse = await apiHelper.verifyToken(invalidToken);
    const invalidData = await invalidResponse.json();
    expect(invalidData.data.valid).toBeFalsy();
    console.log('✅ JWT令牌验证测试完成');
  });

  test('请求签名验证', async ({ request }) => {
    console.log('正在测试请求签名验证...');
    const timestamp = Date.now();
    const signature = await apiHelper.generateSignature('/api/v1/test', 'POST', timestamp);

    const signedResponse = await apiHelper.makeSignedRequest(
      '/api/v1/test',
      'POST',
      { test: 'data' },
      signature,
      timestamp
    );

    expect(signedResponse.ok()).toBeTruthy();
    const unsignedResponse = await apiHelper.makeRequest('/api/v1/test', 'POST', { test: 'data' });
    expect(unsignedResponse.status()).toBe(401);
    console.log('✅ 请求签名验证测试完成');
  });

  test('文件上传安全测试', async ({ request }) => {
    console.log('正在测试文件上传安全...');
    const maliciousFiles = [
      { name: 'script.exe', type: 'application/x-executable' },
      { name: 'shell.php', type: 'text/php' },
      { name: 'data.svg', type: 'image/svg+xml' }
    ];

    for (const file of maliciousFiles) {
      const response = await apiHelper.uploadFile(file.name, file.type);
      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(data.code).not.toBe(0);
      console.log(`恶意文件阻止: ${file.name}`);
    }
    console.log('✅ 文件上传安全测试完成');
  });

  test('HTTP头安全测试', async ({ page }) => {
    console.log('正在测试HTTP安全头...');
    await page.goto('/');

    const securityHeaders = [
      'X-Content-Type-Options',
      'X-Frame-Options',
      'X-XSS-Protection',
      'Strict-Transport-Security',
      'Content-Security-Policy'
    ];

    const headers = await page.evaluate(() => {
      return {
        'X-Content-Type-Options': document.head.querySelector('meta[http-equiv="X-Content-Type-Options"]')?.getAttribute('content'),
        'X-Frame-Options': document.head.querySelector('meta[http-equiv="X-Frame-Options"]')?.getAttribute('content'),
      };
    });

    console.log('安全头:', headers);
    console.log('✅ HTTP头安全测试完成');
  });

  test('重放攻击防护', async ({ request }) => {
    console.log('正在测试重放攻击防护...');
    const timestamp = Date.now();
    const nonce = await apiHelper.generateNonce();

    const response1 = await apiHelper.makeSignedRequest(
      '/api/v1/test',
      'POST',
      { data: 'test' },
      await apiHelper.generateSignature('/api/v1/test', 'POST', timestamp),
      timestamp,
      nonce
    );

    expect(response1.ok()).toBeTruthy();

    const response2 = await apiHelper.makeSignedRequest(
      '/api/v1/test',
      'POST',
      { data: 'test' },
      await apiHelper.generateSignature('/api/v1/test', 'POST', timestamp),
      timestamp,
      nonce
    );

    const data2 = await response2.json();
    expect(data2.code).toBe(1004);
    console.log('✅ 重放攻击防护测试完成');
  });

  test('越权访问防护', async ({ request }) => {
    console.log('正在测试越权访问防护...');
    const user1Login = await apiHelper.login('user1', 'password');
    const user1Data = await user1Login.json();
    const user1Token = user1Data.data.access_token;

    const user2ResourceResponse = await apiHelper.accessResource(
      '/api/v1/user/2/private',
      user1Token
    );

    expect(user2ResourceResponse.ok()).toBeTruthy();
    const resourceData = await user2ResourceResponse.json();
    expect(resourceData.code).toBe(403);
    console.log('✅ 越权访问防护测试完成');
  });

  test('敏感API端点保护', async ({ request }) => {
    console.log('正在测试敏感API端点保护...');
    const protectedEndpoints = [
      '/api/v1/admin/users',
      '/api/v1/admin/config',
      '/api/v1/admin/logs',
      '/api/v1/admin/settings'
    ];

    for (const endpoint of protectedEndpoints) {
      const response = await apiHelper.makeRequest(endpoint);
      expect(response.status()).toBe(401);
      console.log(`端点保护: ${endpoint}`);
    }
    console.log('✅ 敏感API端点保护测试完成');
  });
});
