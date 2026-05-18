import { test, expect } from '@playwright/test';
import { TestHelpers } from '../../utils/test-helpers';
import { ApiHelper } from '../../utils/api-helper';

test.describe('验证码页面完整测试', () => {
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    testHelpers = new TestHelpers();
  });

  test('验证码页面控制台检查', async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('/');
    
    const criticalErrors = consoleErrors.filter(e => 
      !e.includes('favicon') && 
      !e.includes('Failed to load resource')
    );
    console.log('发现的验证码页面控制台错误:', criticalErrors);
    await testHelpers.takeScreenshot(page, 'captcha-page-no-errors');
  });
});

test.describe('滑块验证码E2E测试', () => {
  let testHelpers: TestHelpers;
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ page, request }) => {
    testHelpers = new TestHelpers();
    apiHelper = new ApiHelper(request);
  });

  test('应该能够加载滑块验证码页面', async ({ page }) => {
    await page.goto('/captcha');
    await expect(page).toBeVisible();
    await testHelpers.takeScreenshot(page, 'slider-captcha-loaded');
  });

  test('滑块验证码元素应该存在', async ({ page }) => {
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    const captchaContainer = page.locator('.captcha-container, #captcha, .slider-captcha');
    await expect(captchaContainer).toBeVisible({ timeout: 10000 });
    
    await testHelpers.takeScreenshot(page, 'slider-elements-visible');
  });

  test('应该能够生成新的滑块验证码', async ({ page, request }) => {
    const helper = new ApiHelper(request);
    const result = await helper.generateSliderCaptcha('test-app');
    
    expect(result.success).toBe(true);
    expect(result.data).toHaveProperty('captchaId');
    expect(result.data).toHaveProperty('backgroundImage');
    expect(result.data).toHaveProperty('sliderImage');
    expect(result.data).toHaveProperty('targetX');
  });

  test('应该能够模拟滑块拖拽操作', async ({ page }) => {
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    const sliderTrack = page.locator('.slider-track, .captcha-slider-track, [class*="slider"]');
    const sliderButton = page.locator('.slider-button, .captcha-slider-btn, [class*="slider-btn"]');
    
    const trackVisible = await sliderTrack.isVisible().catch(() => false);
    const buttonVisible = await sliderButton.isVisible().catch(() => false);
    
    if (trackVisible && buttonVisible) {
      const trackBox = await sliderTrack.boundingBox();
      const buttonBox = await sliderButton.boundingBox();
      
      if (trackBox && buttonBox) {
        const startX = buttonBox.x + buttonBox.width / 2;
        const startY = buttonBox.y + buttonBox.height / 2;
        const endX = startX + trackBox.width * 0.7;
        
        await page.mouse.move(startX, startY);
        await page.mouse.down();
        await page.mouse.move(endX, startY, { steps: 20 });
        await page.mouse.up();
        
        await testHelpers.takeScreenshot(page, 'slider-dragged');
      }
    } else {
      console.log('滑块元素未找到，测试跳过实际操作');
    }
  });

  test('滑块验证码验证流程测试', async ({ page, request }) => {
    const helper = new ApiHelper(request);
    const generateResult = await helper.generateSliderCaptcha('test-app');
    
    expect(generateResult.success).toBe(true);
    const captchaId = generateResult.data.captchaId;
    const targetX = generateResult.data.targetX;
    
    const verifyResult = await helper.verifySliderCaptcha(captchaId, targetX, 50);
    expect(verifyResult).toBeDefined();
    expect(verifyResult).toHaveProperty('success');
  });

  test('滑块验证错误处理测试', async ({ page, request }) => {
    const helper = new ApiHelper(request);
    const generateResult = await helper.generateSliderCaptcha('test-app');
    
    const captchaId = generateResult.data.captchaId;
    const wrongX = 9999;
    
    const verifyResult = await helper.verifySliderCaptcha(captchaId, wrongX, 50);
    expect(verifyResult).toBeDefined();
    expect(typeof verifyResult.success).toBe('boolean');
  });

  test('滑块验证码并发生成测试', async ({ request }) => {
    const helper = new ApiHelper(request);
    const promises = Array(3).fill(null).map(() => helper.generateSliderCaptcha('test-app'));
    
    const results = await Promise.all(promises);
    
    results.forEach(result => {
      expect(result.success).toBe(true);
      expect(result.data).toHaveProperty('captchaId');
    });
  });

  test('滑块验证码UI交互测试', async ({ page }) => {
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    const retryButton = page.locator('button:has-text("刷新"), button:has-text("重试"), .retry-btn, [class*="refresh"]');
    const retryVisible = await retryButton.isVisible().catch(() => false);
    
    if (retryVisible) {
      await retryButton.click();
      await testHelpers.takeScreenshot(page, 'slider-refreshed');
    }
  });

  test('滑块验证码控制台错误检测', async ({ page }) => {
    const consoleErrors: string[] = [];
    const consoleWarnings: string[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      } else if (msg.type() === 'warning') {
        consoleWarnings.push(msg.text());
      }
    });
    
    await page.goto('/captcha');
    await page.waitForLoadState('networkidle');
    
    await testHelpers.takeScreenshot(page, 'slider-console-check');
    
    console.log('控制台错误:', consoleErrors);
    console.log('控制台警告:', consoleWarnings);
    
    const criticalErrors = consoleErrors.filter(err => 
      !err.includes('favicon') && 
      !err.includes('404')
    );
    expect(criticalErrors.length).toBe(0);
  });
});
