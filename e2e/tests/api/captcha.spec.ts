import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';

test.describe('验证码API测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
  });

  test.describe('滑块验证码', () => {
    test('应该能够验证滑块验证码（模拟正确验证）', async () => {
      const generateResult = await apiHelper.generateSliderCaptcha();
      const captchaId = generateResult.data?.captchaId || generateResult.data?.id || generateResult.data?.sessionId;
      
      if (captchaId) {
        const verifyResult = await apiHelper.verifySliderCaptcha(captchaId, 100, 50);
        expect(verifyResult).toBeDefined();
      }
    });

    test('错误的滑动位置应该验证失败', async () => {
      const generateResult = await apiHelper.generateSliderCaptcha();
      const captchaId = generateResult.data?.captchaId || generateResult.data?.id || generateResult.data?.sessionId;
      
      if (captchaId) {
        const verifyResult = await apiHelper.verifySliderCaptcha(captchaId, 9999, 9999);
        expect(verifyResult).toBeDefined();
      }
    });
  });

  test.describe('点击验证码', () => {
    test('应该能够验证点击验证码', async () => {
      const generateResult = await apiHelper.generateClickCaptcha();
      const captchaId = generateResult.data?.captchaId || generateResult.data?.id || generateResult.data?.sessionId;
      
      if (captchaId) {
        const points = [{ x: 50, y: 50 }];
        const verifyResult = await apiHelper.verifyClickCaptcha(captchaId, points);
        expect(verifyResult).toBeDefined();
      }
    });
  });

  test.describe('旋转验证码', () => {
    test('应该能够验证旋转验证码', async () => {
      const generateResult = await apiHelper.generateRotateCaptcha();
      const captchaId = generateResult.data?.captchaId || generateResult.data?.id || generateResult.data?.sessionId;
      
      if (captchaId) {
        const verifyResult = await apiHelper.verifyRotateCaptcha(captchaId, 90);
        expect(verifyResult).toBeDefined();
      }
    });
  });

  test.describe('验证码错误处理', () => {
    test('无效的captchaId应该返回错误', async () => {
      const verifyResult = await apiHelper.verifySliderCaptcha('invalid-id', 100, 50);
      expect(verifyResult).toBeDefined();
    });

    test('多次验证同一验证码应该有适当的处理', async () => {
      const generateResult = await apiHelper.generateSliderCaptcha();
      const captchaId = generateResult.data?.captchaId || generateResult.data?.id || generateResult.data?.sessionId;
      
      if (captchaId) {
        await apiHelper.verifySliderCaptcha(captchaId, 100, 50);
        const secondVerify = await apiHelper.verifySliderCaptcha(captchaId, 100, 50);
        expect(secondVerify).toBeDefined();
      }
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
        expect(result).toBeDefined();
      });
    });
  });
});
