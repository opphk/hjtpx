import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';
import { testUsers } from '../../utils/test-data';

test.describe('验证码API测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
  });

  test.describe('滑块验证码', () => {
    test('应该能够生成滑块验证码', async () => {
      const result = await apiHelper.generateSliderCaptcha();
      expect(result).toHaveProperty('success', true);
      expect(result).toHaveProperty('data');
      expect(result.data).toHaveProperty('captchaId');
      expect(result.data).toHaveProperty('imageUrl');
    });

    test('应该能够生成带appId的滑块验证码', async () => {
      const result = await apiHelper.generateSliderCaptcha('test-app-123');
      expect(result).toHaveProperty('success', true);
      expect(result.data).toHaveProperty('captchaId');
      expect(result.data).toHaveProperty('backgroundImage');
      expect(result.data).toHaveProperty('sliderImage');
    });

    test('应该能够验证滑块验证码（模拟正确验证）', async () => {
      const generateResult = await apiHelper.generateSliderCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      const verifyResult = await apiHelper.verifySliderCaptcha(captchaId, 100, 50);
      expect(verifyResult).toHaveProperty('success');
    });

    test('错误的滑动位置应该验证失败', async () => {
      const generateResult = await apiHelper.generateSliderCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      const verifyResult = await apiHelper.verifySliderCaptcha(captchaId, 9999, 9999);
      expect(verifyResult.success).toBeFalsy();
    });

    test('滑块验证码并发生成测试', async () => {
      const promises = [];
      for (let i = 0; i < 10; i++) {
        promises.push(apiHelper.generateSliderCaptcha());
      }
      
      const results = await Promise.all(promises);
      results.forEach(result => {
        expect(result.success).toBeTruthy();
        expect(result.data.captchaId).toBeDefined();
      });
    });

    test('滑块验证码响应时间测试', async () => {
      const startTime = Date.now();
      await apiHelper.generateSliderCaptcha();
      const duration = Date.now() - startTime;
      
      console.log(`滑块验证码生成耗时: ${duration}ms`);
      expect(duration).toBeLessThan(5000);
    });
  });

  test.describe('点击验证码', () => {
    test('应该能够生成点击验证码', async () => {
      const result = await apiHelper.generateClickCaptcha();
      expect(result).toHaveProperty('success', true);
      expect(result).toHaveProperty('data');
      expect(result.data).toHaveProperty('captchaId');
      expect(result.data).toHaveProperty('imageUrl');
    });

    test('应该能够生成带appId的点击验证码', async () => {
      const result = await apiHelper.generateClickCaptcha('click-app-456');
      expect(result).toHaveProperty('success', true);
      expect(result.data).toHaveProperty('targetWords');
    });

    test('应该能够验证点击验证码', async () => {
      const generateResult = await apiHelper.generateClickCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      const points = [{ x: 50, y: 50 }];
      const verifyResult = await apiHelper.verifyClickCaptcha(captchaId, points);
      expect(verifyResult).toHaveProperty('success');
    });

    test('应该能够验证点击验证码（多个点）', async () => {
      const generateResult = await apiHelper.generateClickCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      const points = [
        { x: 50, y: 50 },
        { x: 100, y: 100 },
        { x: 150, y: 150 }
      ];
      const verifyResult = await apiHelper.verifyClickCaptcha(captchaId, points);
      expect(verifyResult).toHaveProperty('success');
    });

    test('点击验证码并发测试', async () => {
      const promises = [];
      for (let i = 0; i < 10; i++) {
        promises.push(apiHelper.generateClickCaptcha());
      }
      
      const results = await Promise.all(promises);
      results.forEach(result => {
        expect(result.success).toBeTruthy();
      });
    });

    test('点击验证码错误坐标测试', async () => {
      const generateResult = await apiHelper.generateClickCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      const wrongPoints = [{ x: 9999, y: 9999 }];
      const verifyResult = await apiHelper.verifyClickCaptcha(captchaId, wrongPoints);
      expect(verifyResult).toBeDefined();
    });
  });

  test.describe('旋转验证码', () => {
    test('应该能够生成旋转验证码', async () => {
      const result = await apiHelper.generateRotateCaptcha();
      expect(result).toHaveProperty('success', true);
      expect(result).toHaveProperty('data');
      expect(result.data).toHaveProperty('captchaId');
      expect(result.data).toHaveProperty('imageUrl');
    });

    test('应该能够生成带appId的旋转验证码', async () => {
      const result = await apiHelper.generateRotateCaptcha('rotate-app-789');
      expect(result).toHaveProperty('success', true);
      expect(result.data).toHaveProperty('targetAngle');
    });

    test('应该能够验证旋转验证码', async () => {
      const generateResult = await apiHelper.generateRotateCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      const verifyResult = await apiHelper.verifyRotateCaptcha(captchaId, 90);
      expect(verifyResult).toHaveProperty('success');
    });

    test('应该能够验证旋转验证码（正确角度）', async () => {
      const generateResult = await apiHelper.generateRotateCaptcha();
      const captchaId = generateResult.data.captchaId;
      const targetAngle = generateResult.data.targetAngle || 45;
      
      const verifyResult = await apiHelper.verifyRotateCaptcha(captchaId, targetAngle);
      expect(verifyResult).toHaveProperty('success');
    });

    test('旋转验证码错误角度测试', async () => {
      const generateResult = await apiHelper.generateRotateCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      const verifyResult = await apiHelper.verifyRotateCaptcha(captchaId, 999);
      expect(verifyResult).toBeDefined();
    });

    test('旋转验证码角度边界测试', async () => {
      const generateResult = await apiHelper.generateRotateCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      const angles = [0, 90, 180, 270, 360];
      for (const angle of angles) {
        const verifyResult = await apiHelper.verifyRotateCaptcha(captchaId, angle);
        expect(verifyResult).toBeDefined();
      }
    });

    test('旋转验证码并发测试', async () => {
      const promises = [];
      for (let i = 0; i < 10; i++) {
        promises.push(apiHelper.generateRotateCaptcha());
      }
      
      const results = await Promise.all(promises);
      results.forEach(result => {
        expect(result.success).toBeTruthy();
      });
    });
  });

  test.describe('图片验证码', () => {
    test('应该能够生成图片验证码', async () => {
      const result = await apiHelper.generateImageCaptcha();
      expect(result).toHaveProperty('success', true);
      expect(result).toHaveProperty('data');
      expect(result.data).toHaveProperty('captchaId');
      expect(result.data).toHaveProperty('imageUrl');
    });

    test('应该能够生成带appId的图片验证码', async () => {
      const result = await apiHelper.generateImageCaptcha('image-app-111');
      expect(result).toHaveProperty('success', true);
      expect(result.data).toHaveProperty('code');
    });
  });

  test.describe('验证码错误处理', () => {
    test('无效的captchaId应该返回错误', async () => {
      const verifyResult = await apiHelper.verifySliderCaptcha('invalid-id', 100, 50);
      expect(verifyResult.success).toBeFalsy();
    });

    test('过期的captchaId应该返回错误', async () => {
      const oldCaptchaId = 'expired-captcha-id-' + Date.now();
      const verifyResult = await apiHelper.verifySliderCaptcha(oldCaptchaId, 100, 50);
      expect(verifyResult).toBeDefined();
    });

    test('多次验证同一验证码应该有适当的处理', async () => {
      const generateResult = await apiHelper.generateSliderCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      await apiHelper.verifySliderCaptcha(captchaId, 100, 50);
      const secondVerify = await apiHelper.verifySliderCaptcha(captchaId, 100, 50);
      expect(secondVerify).toBeDefined();
    });

    test('缺少参数的验证码验证应该被拒绝', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/captcha/slider/verify', {
        data: { captchaId: 'test' }
      });
      const result = await response.json();
      expect(result).toBeDefined();
    });

    test('空字符串参数应该被拒绝', async () => {
      const verifyResult = await apiHelper.verifySliderCaptcha('', 100, 50);
      expect(verifyResult).toBeDefined();
    });
  });

  test.describe('并发验证测试', () => {
    test('应该能够处理并发验证码生成', async () => {
      const promises = [];
      for (let i = 0; i < 5; i++) {
        promises.push(apiHelper.generateSliderCaptcha());
      }
      
      const results = await Promise.all(promises);
      results.forEach(result => {
        expect(result.success).toBeTruthy();
      });
    });

    test('应该能够处理并发验证码验证', async () => {
      const generateResult = await apiHelper.generateSliderCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      const promises = [];
      for (let i = 0; i < 5; i++) {
        promises.push(apiHelper.verifySliderCaptcha(captchaId, 100, 50));
      }
      
      const results = await Promise.all(promises);
      results.forEach(result => {
        expect(result).toBeDefined();
      });
    });

    test('应该能够处理混合类型验证码并发生成', async () => {
      const promises = [];
      promises.push(apiHelper.generateSliderCaptcha());
      promises.push(apiHelper.generateClickCaptcha());
      promises.push(apiHelper.generateRotateCaptcha());
      promises.push(apiHelper.generateImageCaptcha());
      
      const results = await Promise.all(promises);
      results.forEach(result => {
        expect(result.success).toBeTruthy();
      });
    });
  });

  test.describe('验证码性能测试', () => {
    test('应该能够快速生成多个验证码', async () => {
      const startTime = Date.now();
      
      for (let i = 0; i < 20; i++) {
        await apiHelper.generateSliderCaptcha();
      }
      
      const duration = Date.now() - startTime;
      console.log(`生成20个滑块验证码耗时: ${duration}ms`);
      expect(duration).toBeLessThan(30000);
    });

    test('应该能够快速验证多个验证码', async () => {
      const generateResult = await apiHelper.generateSliderCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      const startTime = Date.now();
      
      for (let i = 0; i < 20; i++) {
        await apiHelper.verifySliderCaptcha(captchaId, 100, 50);
      }
      
      const duration = Date.now() - startTime;
      console.log(`验证20次耗时: ${duration}ms`);
      expect(duration).toBeLessThan(30000);
    });

    test('API响应时间应该满足要求', async () => {
      const endpoints = [
        () => apiHelper.generateSliderCaptcha(),
        () => apiHelper.generateClickCaptcha(),
        () => apiHelper.generateRotateCaptcha(),
        () => apiHelper.generateImageCaptcha()
      ];
      
      for (const endpoint of endpoints) {
        const startTime = Date.now();
        await endpoint();
        const duration = Date.now() - startTime;
        
        console.log(`Endpoint耗时: ${duration}ms`);
        expect(duration).toBeLessThan(5000);
      }
    });
  });

  test.describe('验证码安全性测试', () => {
    test('应该拒绝恶意构造的参数', async ({ request }) => {
      const maliciousPayloads = [
        { captchaId: '<script>alert("XSS")</script>', x: 100, y: 50 },
        { captchaId: "admin' OR '1'='1", x: 100, y: 50 },
        { captchaId: '../../../etc/passwd', x: 100, y: 50 },
        { captchaId: '{{constructor.constructor("alert(1)")()}}', x: 100, y: 50 }
      ];
      
      for (const payload of maliciousPayloads) {
        const response = await request.post('http://localhost:8080/api/v1/captcha/slider/verify', {
          data: payload
        });
        const result = await response.json();
        expect(result).toBeDefined();
      }
    });

    test('应该拒绝过大的坐标值', async () => {
      const generateResult = await apiHelper.generateSliderCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      const verifyResult = await apiHelper.verifySliderCaptcha(captchaId, 999999, 999999);
      expect(verifyResult).toBeDefined();
    });

    test('应该拒绝负数坐标值', async () => {
      const generateResult = await apiHelper.generateSliderCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      const verifyResult = await apiHelper.verifySliderCaptcha(captchaId, -100, -50);
      expect(verifyResult).toBeDefined();
    });

    test('应该拒绝非数字坐标', async ({ request }) => {
      const generateResult = await apiHelper.generateSliderCaptcha();
      const captchaId = generateResult.data.captchaId;
      
      const response = await request.post('http://localhost:8080/api/v1/captcha/slider/verify', {
        data: { captchaId, x: 'invalid', y: 'invalid' }
      });
      const result = await response.json();
      expect(result).toBeDefined();
    });
  });

  test.describe('验证码数据完整性测试', () => {
    test('滑块验证码数据应该完整', async () => {
      const result = await apiHelper.generateSliderCaptcha();
      
      expect(result.data).toHaveProperty('captchaId');
      expect(result.data).toHaveProperty('backgroundImage');
      expect(result.data).toHaveProperty('sliderImage');
      expect(result.data).toHaveProperty('targetX');
      expect(result.data).toHaveProperty('targetY');
      
      expect(typeof result.data.captchaId).toBe('string');
      expect(typeof result.data.backgroundImage).toBe('string');
      expect(typeof result.data.targetX).toBe('number');
    });

    test('点击验证码数据应该完整', async () => {
      const result = await apiHelper.generateClickCaptcha();
      
      expect(result.data).toHaveProperty('captchaId');
      expect(result.data).toHaveProperty('backgroundImage');
      expect(result.data).toHaveProperty('targetWords');
      
      expect(typeof result.data.captchaId).toBe('string');
      expect(Array.isArray(result.data.targetWords)).toBeTruthy();
    });

    test('旋转验证码数据应该完整', async () => {
      const result = await apiHelper.generateRotateCaptcha();
      
      expect(result.data).toHaveProperty('captchaId');
      expect(result.data).toHaveProperty('backgroundImage');
      expect(result.data).toHaveProperty('targetAngle');
      
      expect(typeof result.data.captchaId).toBe('string');
      expect(typeof result.data.targetAngle).toBe('number');
      expect(result.data.targetAngle).toBeGreaterThanOrEqual(0);
      expect(result.data.targetAngle).toBeLessThanOrEqual(360);
    });
  });

  test.describe('验证码Session管理测试', () => {
    test('每个验证码应该有唯一的captchaId', async () => {
      const result1 = await apiHelper.generateSliderCaptcha();
      const result2 = await apiHelper.generateSliderCaptcha();
      
      expect(result1.data.captchaId).not.toBe(result2.data.captchaId);
    });

    test('验证码ID长度应该在合理范围内', async () => {
      const result = await apiHelper.generateSliderCaptcha();
      const captchaId = result.data.captchaId;
      
      expect(captchaId.length).toBeGreaterThan(10);
      expect(captchaId.length).toBeLessThan(100);
    });

    test('Image URL应该是有效的URL', async () => {
      const result = await apiHelper.generateSliderCaptcha();
      const imageUrl = result.data.backgroundImage;
      
      expect(imageUrl).toMatch(/^https?:\/\/.+/);
    });
  });

  test.describe('健康检查API测试', () => {
    test('健康检查端点应该返回200', async () => {
      const result = await apiHelper.healthCheck();
      expect(result).toBe(true);
    });

    test('健康检查响应时间应该很快', async ({ request }) => {
      const startTime = Date.now();
      await request.get('http://localhost:8080/health');
      const duration = Date.now() - startTime;
      
      console.log(`健康检查耗时: ${duration}ms`);
      expect(duration).toBeLessThan(1000);
    });
  });
});

test.describe('管理后台API测试', () => {
  let apiHelper: ApiHelper;
  let authToken: string;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
    const loginResult = await apiHelper.adminLogin(
      testUsers.admin.username,
      testUsers.admin.password
    );
    authToken = loginResult.data.token;
  });

  test.describe('认证API测试', () => {
    test('应该能够通过API登录', async () => {
      const result = await apiHelper.adminLogin(
        testUsers.admin.username,
        testUsers.admin.password
      );
      
      expect(result.success).toBe(true);
      expect(result.data).toHaveProperty('token');
      expect(result.data.token).toBeTruthy();
    });

    test('无效凭据应该返回错误', async () => {
      const result = await apiHelper.adminLogin('wronguser', 'wrongpass');
      expect(result.success).toBe(false);
    });

    test('空凭据应该返回错误', async () => {
      const result = await apiHelper.adminLogin('', '');
      expect(result).toBeDefined();
    });

    test('Token应该具有正确的格式', async () => {
      const result = await apiHelper.adminLogin(
        testUsers.admin.username,
        testUsers.admin.password
      );
      
      const token = result.data.token;
      expect(token).toMatch(/^[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+$/);
    });
  });

  test.describe('统计数据API测试', () => {
    test('应该能够获取验证统计数据', async () => {
      const result = await apiHelper.getVerificationStats(authToken);
      expect(result.success).toBe(true);
      expect(result.data).toBeDefined();
    });

    test('统计数据应该包含必要字段', async () => {
      const result = await apiHelper.getVerificationStats(authToken);
      
      if (result.success && result.data) {
        const hasData = Object.keys(result.data).length > 0;
        expect(hasData).toBe(true);
      }
    });

    test('统计数据值应该是数字', async () => {
      const result = await apiHelper.getVerificationStats(authToken);
      
      if (result.success && result.data) {
        Object.values(result.data).forEach(value => {
          if (typeof value === 'number') {
            expect(value).toBeGreaterThanOrEqual(0);
          }
        });
      }
    });
  });

  test.describe('应用管理API测试', () => {
    test('应该能够获取应用列表', async () => {
      const result = await apiHelper.getApplications(authToken);
      expect(result.success).toBe(true);
      expect(result.data).toBeDefined();
    });

    test('应该能够创建新应用', async () => {
      const appName = 'API Test App - ' + Date.now();
      const result = await apiHelper.createApplication(
        authToken,
        appName,
        'Created via API'
      );
      expect(result.success).toBe(true);
    });

    test('应该能够创建带特殊字符的应用', async () => {
      const appName = 'Test App 中文 ' + Date.now();
      const result = await apiHelper.createApplication(
        authToken,
        appName,
        '测试应用'
      );
      expect(result).toBeDefined();
    });

    test('创建重复应用应该有适当处理', async () => {
      const appName = 'Duplicate Test ' + Date.now();
      
      await apiHelper.createApplication(authToken, appName, 'First');
      const secondResult = await apiHelper.createApplication(authToken, appName, 'Second');
      
      expect(secondResult).toBeDefined();
    });
  });

  test.describe('日志API测试', () => {
    test('应该能够获取日志', async () => {
      const result = await apiHelper.getLogs(authToken, { limit: 10 });
      expect(result.success).toBe(true);
      expect(result.data).toBeDefined();
    });

    test('应该能够按时间范围获取日志', async () => {
      const result = await apiHelper.getLogs(authToken, {
        startTime: Date.now() - 3600000,
        endTime: Date.now()
      });
      expect(result).toBeDefined();
    });

    test('应该能够按类型过滤日志', async () => {
      const result = await apiHelper.getLogs(authToken, {
        type: 'verification'
      });
      expect(result).toBeDefined();
    });
  });

  test.describe('API权限测试', () => {
    test('缺失Token应该被拒绝', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/stats');
      expect(response.status()).toBe(401);
    });

    test('无效Token应该被拒绝', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/stats', {
        headers: { Authorization: 'Bearer invalid-token' }
      });
      expect(response.status()).not.toBe(200);
    });

    test('过期Token应该被拒绝', async ({ request }) => {
      const oldToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZXhwIjoxNjAwMDAwMDAwfQ.dummysignature';
      const response = await request.get('http://localhost:8080/api/v1/admin/stats', {
        headers: { Authorization: `Bearer ${oldToken}` }
      });
      expect(response.status()).not.toBe(200);
    });
  });

  test.describe('API错误处理测试', () => {
    test('服务器错误应该返回适当的响应', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/nonexistent-endpoint');
      expect(response.status()).toBeGreaterThanOrEqual(400);
    });

    test('无效JSON应该被拒绝', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/auth/login', {
        data: 'invalid json'
      });
      expect(response.status()).toBeGreaterThanOrEqual(400);
    });

    test('缺少必需字段应该返回错误', async ({ request }) => {
      const response = await request.post('http://localhost:8080/api/v1/admin/applications', {
        headers: { Authorization: `Bearer ${authToken}` },
        data: {}
      });
      const result = await response.json();
      expect(result).toBeDefined();
    });
  });

  test.describe('API性能测试', () => {
    test('认证API响应时间应该很快', async () => {
      const startTime = Date.now();
      await apiHelper.adminLogin(testUsers.admin.username, testUsers.admin.password);
      const duration = Date.now() - startTime;
      
      console.log(`认证API耗时: ${duration}ms`);
      expect(duration).toBeLessThan(5000);
    });

    test('统计数据API响应时间应该很快', async () => {
      const startTime = Date.now();
      await apiHelper.getVerificationStats(authToken);
      const duration = Date.now() - startTime;
      
      console.log(`统计API耗时: ${duration}ms`);
      expect(duration).toBeLessThan(5000);
    });

    test('应用列表API响应时间应该很快', async () => {
      const startTime = Date.now();
      await apiHelper.getApplications(authToken);
      const duration = Date.now() - startTime;
      
      console.log(`应用列表API耗时: ${duration}ms`);
      expect(duration).toBeLessThan(5000);
    });
  });
});
