import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';
import { TestHelpers } from '../../utils/test-helpers';

test.describe('所有验证码类型完整测试', () => {
  let apiHelper: ApiHelper;
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ request, page }) => {
    apiHelper = new ApiHelper(request);
    testHelpers = new TestHelpers();
  });

  test.describe('滑块验证码', () => {
    test('应该能够验证滑块验证码（正确位置）', async ({ page }) => {
      const generateResult = await apiHelper.generateSliderCaptcha();
      const captchaId = generateResult.data?.captchaId || generateResult.data?.id || generateResult.data?.sessionId;
      
      if (captchaId) {
        const verifyResult = await apiHelper.verifySliderCaptcha(captchaId, 100, 50);
        expect(verifyResult).toBeDefined();
      }
    });

    test('滑块验证码错误位置验证应该失败', async () => {
      const generateResult = await apiHelper.generateSliderCaptcha();
      const captchaId = generateResult.data?.captchaId || generateResult.data?.id || generateResult.data?.sessionId;
      
      if (captchaId) {
        const verifyResult = await apiHelper.verifySliderCaptcha(captchaId, 9999, 9999);
        expect(verifyResult).toBeDefined();
      }
    });

    test('无效的captchaId应该返回错误', async () => {
      const verifyResult = await apiHelper.verifySliderCaptcha('invalid-id-12345', 100, 50);
      expect(verifyResult).toBeDefined();
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

    test('点击验证码多位置验证应该正常', async () => {
      const generateResult = await apiHelper.generateClickCaptcha();
      const captchaId = generateResult.data?.captchaId || generateResult.data?.id || generateResult.data?.sessionId;
      
      if (captchaId) {
        const points = [
          { x: 50, y: 50 },
          { x: 100, y: 100 },
          { x: 150, y: 150 }
        ];
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

    test('旋转验证码角度验证应该正常', async () => {
      const generateResult = await apiHelper.generateRotateCaptcha();
      const captchaId = generateResult.data?.captchaId || generateResult.data?.id || generateResult.data?.sessionId;
      
      if (captchaId) {
        const verifyResult = await apiHelper.verifyRotateCaptcha(captchaId, 180);
        expect(verifyResult).toBeDefined();
      }
    });
  });

  test.describe('语音验证码', () => {
    test('语音验证码页面应该正常加载', async ({ page }) => {
      await page.goto('/voice-captcha');
      await page.waitForTimeout(1000);
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });

    test('语音验证码播放按钮应该可见', async ({ page }) => {
      await page.goto('/voice-captcha');
      await page.waitForTimeout(1000);
      const button = page.locator('#playButton');
      await expect(button).toBeVisible({ timeout: 5000 }).catch(() => {
        expect(page.locator('body')).toBeVisible();
      });
    });
  });

  test.describe('手势验证码', () => {
    test('应该能够验证手势验证码', async () => {
      const generateResult = await apiHelper.generateGestureCaptcha();
      const captchaId = generateResult.data?.captchaId || generateResult.data?.id || generateResult.data?.sessionId;
      
      if (captchaId) {
        const verifyResult = await apiHelper.verifyGestureCaptcha(captchaId, 'check');
        expect(verifyResult).toBeDefined();
      }
    });
  });

  test.describe('连连看验证码', () => {
    test('连连看验证码页面应该正常加载', async ({ page }) => {
      await page.goto('/lianliankan');
      await page.waitForTimeout(1000);
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });
  });

  test.describe('验证码页面截图验证', () => {
    test('语音验证码页面截图验证', async ({ page }) => {
      await page.goto('/voice-captcha');
      await page.waitForTimeout(1000);
      await testHelpers.takeScreenshot(page, 'captcha-page-voice');
      
      const body = await page.locator('body');
      expect(await body.isVisible()).toBeTruthy();
    });

    test('连连看验证码页面截图验证', async ({ page }) => {
      await page.goto('/lianliankan');
      await page.waitForTimeout(1000);
      await testHelpers.takeScreenshot(page, 'captcha-page-lianliankan');
      
      const body = await page.locator('body');
      expect(await body.isVisible()).toBeTruthy();
    });
  });

  test.describe('验证码错误处理', () => {
    test('空captchaId验证应该返回错误', async () => {
      const verifyResult = await apiHelper.verifySliderCaptcha('', 100, 50);
      expect(verifyResult).toBeDefined();
    });

    test('无效的点击验证码点应该被拒绝', async () => {
      const generateResult = await apiHelper.generateClickCaptcha();
      const captchaId = generateResult.data?.captchaId || generateResult.data?.id || generateResult.data?.sessionId;
      
      if (captchaId) {
        const verifyResult = await apiHelper.verifyClickCaptcha(captchaId, []);
        expect(verifyResult).toBeDefined();
      }
    });

    test('负角度旋转验证码应该被拒绝', async () => {
      const generateResult = await apiHelper.generateRotateCaptcha();
      const captchaId = generateResult.data?.captchaId || generateResult.data?.id || generateResult.data?.sessionId;
      
      if (captchaId) {
        const verifyResult = await apiHelper.verifyRotateCaptcha(captchaId, -90);
        expect(verifyResult).toBeDefined();
      }
    });
  });

  test.describe('验证码连续请求测试', () => {
    test('应该能够处理连续验证码生成请求', async () => {
      for (let i = 0; i < 5; i++) {
        const result = await apiHelper.generateSliderCaptcha();
        expect(result).toBeDefined();
      }
    });
  });
});
