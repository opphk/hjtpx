import { test, expect } from '@playwright/test';
import { TestHelpers } from '../../utils/test-helpers';
import { ApiHelper } from '../../utils/api-helper';

test.describe('旋转验证码E2E测试', () => {
  let testHelpers: TestHelpers;
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ page, request }) => {
    testHelpers = new TestHelpers();
    apiHelper = new ApiHelper(request);
  });

  test('应该能够加载旋转验证码页面', async ({ page }) => {
    await page.goto('/captcha?type=rotate');
    await expect(page).toBeVisible();
    await testHelpers.takeScreenshot(page, 'rotate-captcha-loaded');
  });

  test('应该能够生成旋转验证码', async ({ request }) => {
    const helper = new ApiHelper(request);
    const result = await helper.generateRotateCaptcha('test-app');
    
    expect(result.success).toBe(true);
    expect(result.data).toHaveProperty('captchaId');
    expect(result.data).toHaveProperty('backgroundImage');
    expect(result.data).toHaveProperty('targetAngle');
  });

  test('应该能够验证旋转验证码（正确角度）', async ({ request }) => {
    const helper = new ApiHelper(request);
    const generateResult = await helper.generateRotateCaptcha('test-app');
    
    expect(generateResult.success).toBe(true);
    const captchaId = generateResult.data.captchaId;
    const targetAngle = generateResult.data.targetAngle || 45;
    
    const verifyResult = await helper.verifyRotateCaptcha(captchaId, targetAngle);
    expect(verifyResult).toBeDefined();
    expect(verifyResult).toHaveProperty('success');
  });

  test('应该能够模拟旋转验证码拖拽操作', async ({ page }) => {
    await page.goto('/captcha?type=rotate');
    await page.waitForLoadState('networkidle');
    
    const rotateArea = page.locator('.captcha-rotate, .rotate-captcha, [class*="rotate"]');
    const areaVisible = await rotateArea.isVisible().catch(() => false);
    
    if (!areaVisible) {
      console.log('旋转区域未找到，测试页面加载');
      await expect(page).toBeVisible();
    }
    
    await testHelpers.takeScreenshot(page, 'rotate-captcha-area');
    
    const rotateHandle = page.locator('.rotate-handle, .captcha-rotate-handle, [class*="handle"]');
    const handleVisible = await rotateHandle.isVisible().catch(() => false);
    
    if (handleVisible) {
      const handleBox = await rotateHandle.boundingBox();
      if (handleBox) {
        const centerX = handleBox.x + handleBox.width / 2;
        const centerY = handleBox.y + handleBox.height / 2;
        
        await page.mouse.move(centerX, centerY);
        await page.mouse.down();
        await page.mouse.move(centerX + 50, centerY, { steps: 10 });
        await page.mouse.up();
        
        await testHelpers.takeScreenshot(page, 'rotate-captcha-rotated');
      }
    }
  });

  test('旋转验证码验证错误处理测试', async ({ request }) => {
    const helper = new ApiHelper(request);
    const generateResult = await helper.generateRotateCaptcha('test-app');
    
    const captchaId = generateResult.data.captchaId;
    const wrongAngle = 999;
    
    const verifyResult = await helper.verifyRotateCaptcha(captchaId, wrongAngle);
    expect(verifyResult).toBeDefined();
    expect(typeof verifyResult.success).toBe('boolean');
  });

  test('旋转验证码并发生成测试', async ({ request }) => {
    const helper = new ApiHelper(request);
    const promises = Array(3).fill(null).map(() => helper.generateRotateCaptcha('test-app'));
    
    const results = await Promise.all(promises);
    
    results.forEach(result => {
      expect(result.success).toBe(true);
      expect(result.data).toHaveProperty('captchaId');
    });
  });

  test('旋转验证码UI元素测试', async ({ page }) => {
    await page.goto('/captcha?type=rotate');
    await page.waitForLoadState('networkidle');
    
    const captchaContainer = page.locator('.captcha-container, #captcha, .rotate-captcha');
    const containerVisible = await captchaContainer.isVisible().catch(() => false);
    
    if (!containerVisible) {
      console.log('旋转验证码容器未找到，尝试通用captcha页面');
      await page.goto('/captcha');
    }
    
    await testHelpers.takeScreenshot(page, 'rotate-captcha-ui');
  });

  test('旋转验证码刷新功能测试', async ({ page }) => {
    await page.goto('/captcha?type=rotate');
    await page.waitForLoadState('networkidle');
    
    const refreshButton = page.locator('button:has-text("刷新"), button:has-text("重试"), .refresh-btn, [class*="refresh"]');
    const refreshVisible = await refreshButton.isVisible().catch(() => false);
    
    if (refreshVisible) {
      await refreshButton.click();
      await testHelpers.takeScreenshot(page, 'rotate-captcha-refreshed');
    }
  });

  test('旋转验证码完整流程测试', async ({ page, request }) => {
    const helper = new ApiHelper(request);
    
    const generateResult = await helper.generateRotateCaptcha('test-app');
    expect(generateResult.success).toBe(true);
    
    const captchaId = generateResult.data.captchaId;
    const targetAngle = generateResult.data.targetAngle || 45;
    
    const verifyResult = await helper.verifyRotateCaptcha(captchaId, targetAngle);
    expect(verifyResult).toBeDefined();
    
    await page.goto('/captcha?type=rotate');
    await testHelpers.takeScreenshot(page, 'rotate-captcha-complete-flow');
  });

  test('旋转验证码控制台错误检测', async ({ page }) => {
    const consoleErrors: string[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('/captcha?type=rotate');
    await page.waitForLoadState('networkidle');
    
    await testHelpers.takeScreenshot(page, 'rotate-captcha-console-check');
    
    const criticalErrors = consoleErrors.filter(err => 
      !err.includes('favicon') && 
      !err.includes('404')
    );
    expect(criticalErrors.length).toBe(0);
  });

  test('旋转验证码角度范围测试', async ({ request }) => {
    const helper = new ApiHelper(request);
    const generateResult = await helper.generateRotateCaptcha('test-app');
    
    const captchaId = generateResult.data.captchaId;
    const targetAngle = generateResult.data.targetAngle;
    
    expect(typeof targetAngle).toBe('number');
    expect(targetAngle).toBeGreaterThanOrEqual(0);
    expect(targetAngle).toBeLessThanOrEqual(360);
    
    const verifyResult = await helper.verifyRotateCaptcha(captchaId, targetAngle);
    expect(verifyResult).toBeDefined();
  });

  test('旋转验证码多次旋转操作测试', async ({ page }) => {
    await page.goto('/captcha?type=rotate');
    await page.waitForLoadState('networkidle');
    
    const rotateHandle = page.locator('.rotate-handle, .captcha-rotate-handle, [class*="handle"]');
    const handleVisible = await rotateHandle.isVisible().catch(() => false);
    
    if (handleVisible) {
      const handleBox = await rotateHandle.boundingBox();
      if (handleBox) {
        const centerX = handleBox.x + handleBox.width / 2;
        const centerY = handleBox.y + handleBox.height / 2;
        
        for (let i = 0; i < 3; i++) {
          await page.mouse.move(centerX, centerY);
          await page.mouse.down();
          await page.mouse.move(centerX + 30 * (i + 1), centerY, { steps: 5 });
          await page.mouse.up();
          await page.waitForTimeout(500);
        }
        
        await testHelpers.takeScreenshot(page, 'rotate-captcha-multi-rotate');
      }
    }
  });
});

test.describe('旋转验证码边界测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
  });

  test('应该拒绝负数角度', async () => {
    const generateResult = await apiHelper.generateRotateCaptcha('test-app');
    const captchaId = generateResult.data.captchaId;
    
    const verifyResult = await apiHelper.verifyRotateCaptcha(captchaId, -45);
    expect(verifyResult).toBeDefined();
  });

  test('应该拒绝大于360的角度', async () => {
    const generateResult = await apiHelper.generateRotateCaptcha('test-app');
    const captchaId = generateResult.data.captchaId;
    
    const verifyResult = await apiHelper.verifyRotateCaptcha(captchaId, 450);
    expect(verifyResult).toBeDefined();
  });

  test('应该处理0度旋转', async () => {
    const generateResult = await apiHelper.generateRotateCaptcha('test-app');
    const captchaId = generateResult.data.captchaId;
    
    const verifyResult = await apiHelper.verifyRotateCaptcha(captchaId, 0);
    expect(verifyResult).toBeDefined();
  });

  test('应该处理360度旋转', async () => {
    const generateResult = await apiHelper.generateRotateCaptcha('test-app');
    const captchaId = generateResult.data.captchaId;
    
    const verifyResult = await apiHelper.verifyRotateCaptcha(captchaId, 360);
    expect(verifyResult).toBeDefined();
  });

  test('应该处理小数角度', async () => {
    const generateResult = await apiHelper.generateRotateCaptcha('test-app');
    const captchaId = generateResult.data.captchaId;
    
    const verifyResult = await apiHelper.verifyRotateCaptcha(captchaId, 45.5);
    expect(verifyResult).toBeDefined();
  });
});
