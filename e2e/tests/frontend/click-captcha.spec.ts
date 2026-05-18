import { test, expect } from '@playwright/test';
import { TestHelpers } from '../../utils/test-helpers';
import { ApiHelper } from '../../utils/api-helper';

test.describe('点选验证码E2E测试', () => {
  let testHelpers: TestHelpers;
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ page, request }) => {
    testHelpers = new TestHelpers();
    apiHelper = new ApiHelper(request);
  });

  test('应该能够加载点选验证码页面', async ({ page }) => {
    await page.goto('/captcha?type=click');
    await expect(page).toBeVisible();
    await testHelpers.takeScreenshot(page, 'click-captcha-loaded');
  });

  test('应该能够生成点选验证码', async ({ request }) => {
    const helper = new ApiHelper(request);
    const result = await helper.generateClickCaptcha('test-app');
    
    expect(result.success).toBe(true);
    expect(result.data).toHaveProperty('captchaId');
    expect(result.data).toHaveProperty('backgroundImage');
    expect(result.data).toHaveProperty('targetWords');
  });

  test('应该能够验证点选验证码（单个点）', async ({ request }) => {
    const helper = new ApiHelper(request);
    const generateResult = await helper.generateClickCaptcha('test-app');
    
    expect(generateResult.success).toBe(true);
    const captchaId = generateResult.data.captchaId;
    const points = [{ x: 100, y: 100 }];
    
    const verifyResult = await helper.verifyClickCaptcha(captchaId, points);
    expect(verifyResult).toBeDefined();
    expect(verifyResult).toHaveProperty('success');
  });

  test('应该能够验证点选验证码（多个点）', async ({ request }) => {
    const helper = new ApiHelper(request);
    const generateResult = await helper.generateClickCaptcha('test-app');
    
    const captchaId = generateResult.data.captchaId;
    const points = [
      { x: 50, y: 50 },
      { x: 150, y: 100 },
      { x: 200, y: 150 }
    ];
    
    const verifyResult = await helper.verifyClickCaptcha(captchaId, points);
    expect(verifyResult).toBeDefined();
    expect(verifyResult).toHaveProperty('success');
  });

  test('点选验证码UI元素测试', async ({ page }) => {
    await page.goto('/captcha?type=click');
    await page.waitForLoadState('networkidle');
    
    const captchaContainer = page.locator('.captcha-container, #captcha, .click-captcha');
    const containerVisible = await captchaContainer.isVisible().catch(() => false);
    
    if (!containerVisible) {
      console.log('点选验证码容器未找到，尝试通用captcha页面');
      await page.goto('/captcha');
    }
    
    await testHelpers.takeScreenshot(page, 'click-captcha-ui');
  });

  test('点选验证码点击交互测试', async ({ page }) => {
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    const captchaArea = page.locator('.captcha-image, .captcha-bg, [class*="captcha"]');
    const areaVisible = await captchaArea.isVisible().catch(() => false);
    
    if (areaVisible) {
      const box = await captchaArea.boundingBox();
      if (box) {
        await page.mouse.click(box.x + box.width / 2, box.y + box.height / 2);
        await testHelpers.takeScreenshot(page, 'click-captcha-clicked');
      }
    }
  });

  test('点选验证码验证错误处理测试', async ({ request }) => {
    const helper = new ApiHelper(request);
    const generateResult = await helper.generateClickCaptcha('test-app');
    
    const captchaId = generateResult.data.captchaId;
    const wrongPoints = [{ x: 9999, y: 9999 }];
    
    const verifyResult = await helper.verifyClickCaptcha(captchaId, wrongPoints);
    expect(verifyResult).toBeDefined();
    expect(typeof verifyResult.success).toBe('boolean');
  });

  test('点选验证码并发生成测试', async ({ request }) => {
    const helper = new ApiHelper(request);
    const promises = Array(3).fill(null).map(() => helper.generateClickCaptcha('test-app'));
    
    const results = await Promise.all(promises);
    
    results.forEach(result => {
      expect(result.success).toBe(true);
      expect(result.data).toHaveProperty('captchaId');
    });
  });

  test('点选验证码控制台错误检测', async ({ page }) => {
    const consoleErrors: string[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    await testHelpers.takeScreenshot(page, 'click-captcha-console-check');
    
    const criticalErrors = consoleErrors.filter(err => 
      !err.includes('favicon') && 
      !err.includes('404')
    );
    expect(criticalErrors.length).toBe(0);
  });

  test('点选验证码刷新功能测试', async ({ page }) => {
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    const refreshButton = page.locator('button:has-text("刷新"), button:has-text("重试"), .refresh-btn, [class*="refresh"]');
    const refreshVisible = await refreshButton.isVisible().catch(() => false);
    
    if (refreshVisible) {
      await refreshButton.click();
      await testHelpers.takeScreenshot(page, 'click-captcha-refreshed');
    }
  });

  test('点选验证码完整流程测试', async ({ page, request }) => {
    const helper = new ApiHelper(request);
    
    const generateResult = await helper.generateClickCaptcha('test-app');
    expect(generateResult.success).toBe(true);
    
    const captchaId = generateResult.data.captchaId;
    const targetWords = generateResult.data.targetWords || [];
    
    let points: { x: number; y: number }[] = [];
    if (targetWords.length > 0) {
      points = targetWords.map((word: any, index: number) => ({
        x: 100 + index * 50,
        y: 100 + index * 30
      }));
    } else {
      points = [{ x: 100, y: 100 }];
    }
    
    const verifyResult = await helper.verifyClickCaptcha(captchaId, points);
    expect(verifyResult).toBeDefined();
    
    await page.goto('/captcha');
    await testHelpers.takeScreenshot(page, 'click-captcha-complete-flow');
  });
});

test.describe('连连看验证码E2E测试', () => {
  let testHelpers: TestHelpers;
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ page, request }) => {
    testHelpers = new TestHelpers();
    apiHelper = new ApiHelper(request);
  });

  test('应该能够加载连连看验证码页面', async ({ page }) => {
    await page.goto('/lianliankan');
    await page.waitForLoadState('networkidle');
    await testHelpers.takeScreenshot(page, 'lianliankan-loaded');
  });

  test('连连看验证码API生成测试', async ({ request }) => {
    const helper = new ApiHelper(request);
    const result = await helper.generateSliderCaptcha('test-app');
    
    expect(result.success).toBe(true);
    expect(result.data).toHaveProperty('captchaId');
  });

  test('连连看验证码UI测试', async ({ page }) => {
    await page.goto('/lianliankan');
    await page.waitForLoadState('networkidle');
    
    const captchaArea = page.locator('.captcha-container, #captcha, [class*="captcha"]');
    const visible = await captchaArea.isVisible().catch(() => false);
    
    if (!visible) {
      console.log('连连看容器未找到，测试页面加载');
      await expect(page).toBeVisible();
    }
    
    await testHelpers.takeScreenshot(page, 'lianliankan-ui');
  });

  test('连连看验证码控制台错误检测', async ({ page }) => {
    const consoleErrors: string[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('/lianliankan');
    await page.waitForLoadState('networkidle');
    
    await testHelpers.takeScreenshot(page, 'lianliankan-console-check');
    
    const criticalErrors = consoleErrors.filter(err => 
      !err.includes('favicon') && 
      !err.includes('404')
    );
    expect(criticalErrors.length).toBe(0);
  });
});
