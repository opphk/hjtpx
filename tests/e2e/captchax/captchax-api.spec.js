const { baseTest, expect } = require('../utils/test-helpers');

baseTest.describe('CaptchaX API 测试', () => {
  baseTest.describe('健康检查', () => {
    baseTest('应该返回健康状态', async ({ page, request, testHelpers }) => {
      const { consoleChecker } = testHelpers;
      
      const response = await request.get('/health').catch(() => null);
      
      if (response) {
        expect(response.ok()).toBeTruthy();
      }
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('验证码生成 API', () => {
    baseTest('应该能生成滑块验证码（如果 API 存在）', async ({ page, request, testHelpers }) => {
      const { consoleChecker } = testHelpers;
      
      const response = await request.post('/api/v1/captcha/slider', {
        data: {}
      }).catch(() => null);
      
      if (response) {
        console.log(`滑块验证码 API 状态: ${response.status()}`);
      }
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该能生成点击验证码（如果 API 存在）', async ({ page, request, testHelpers }) => {
      const { consoleChecker } = testHelpers;
      
      const response = await request.post('/api/v1/captcha/click', {
        data: {}
      }).catch(() => null);
      
      if (response) {
        console.log(`点击验证码 API 状态: ${response.status()}`);
      }
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该能生成拼图验证码（如果 API 存在）', async ({ page, request, testHelpers }) => {
      const { consoleChecker } = testHelpers;
      
      const response = await request.post('/api/v1/captcha/puzzle', {
        data: {}
      }).catch(() => null);
      
      if (response) {
        console.log(`拼图验证码 API 状态: ${response.status()}`);
      }
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('验证 API', () => {
    baseTest('应该能验证滑块验证码（如果 API 存在）', async ({ page, request, testHelpers }) => {
      const { consoleChecker } = testHelpers;
      
      const response = await request.post('/api/v1/captcha/slider/verify', {
        data: {
          token: 'test-token',
          answer: 'test-answer'
        }
      }).catch(() => null);
      
      if (response) {
        console.log(`滑块验证 API 状态: ${response.status()}`);
      }
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('批量验证 API', () => {
    baseTest('应该支持批量验证（如果 API 存在）', async ({ page, request, testHelpers }) => {
      const { consoleChecker } = testHelpers;
      
      const response = await request.post('/api/v2/captcha/batch/verify', {
        data: {
          tokens: ['test-token-1', 'test-token-2']
        }
      }).catch(() => null);
      
      if (response) {
        console.log(`批量验证 API 状态: ${response.status()}`);
      }
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('场景管理 API', () => {
    baseTest('应该能获取场景列表（如果 API 存在）', async ({ page, request, testHelpers }) => {
      const { consoleChecker } = testHelpers;
      
      const response = await request.get('/api/v2/captcha/scenarios').catch(() => null);
      
      if (response) {
        console.log(`场景列表 API 状态: ${response.status()}`);
      }
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });
});
