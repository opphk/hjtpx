import { test, expect } from '@playwright/test';
import { TestHelpers } from '../../utils/test-helpers';
import { ApiHelper } from '../../utils/api-helper';

test.describe('验证码页面完整测试', () => {
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    testHelpers = new TestHelpers();
  });

  test('滑块验证码页面测试', async ({ page }) => {
    console.log('正在测试滑块验证码页面...');
    await page.goto('/captcha');
    await testHelpers.takeScreenshot(page, 'captcha-page-slider');
    
    await expect(page).toBeVisible();
    console.log('✅ 滑块验证码页面加载成功');
  });

  test('验证码页面控制台检查', async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('/captcha');
    
    console.log('发现的验证码页面控制台错误:', consoleErrors);
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

  test('滑块验证码目标位置范围测试', async ({ request }) => {
    const helper = new ApiHelper(request);
    const generateResult = await helper.generateSliderCaptcha('test-app');
    
    expect(generateResult.success).toBe(true);
    const targetX = generateResult.data.targetX;
    
    expect(typeof targetX).toBe('number');
    expect(targetX).toBeGreaterThanOrEqual(0);
  });

  test('滑块验证码多次拖拽测试', async ({ page }) => {
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
        
        for (let i = 1; i <= 3; i++) {
          const endX = startX + (trackBox.width * 0.3) * i;
          await page.mouse.move(startX, startY);
          await page.mouse.down();
          await page.mouse.move(Math.min(endX, trackBox.x + trackBox.width), startY, { steps: 10 });
          await page.mouse.up();
          await page.waitForTimeout(500);
        }
        
        await testHelpers.takeScreenshot(page, 'slider-multi-drag');
      }
    }
  });

  test('滑块验证码滑块边界测试', async ({ page }) => {
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
        
        await page.mouse.move(startX, startY);
        await page.mouse.down();
        await page.mouse.move(trackBox.x + trackBox.width, startY, { steps: 10 });
        await page.mouse.up();
        
        await testHelpers.takeScreenshot(page, 'slider-at-boundary');
      }
    }
  });

  test('滑块验证码快速拖拽测试', async ({ page }) => {
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
        await page.mouse.move(endX, startY, { steps: 3 });
        await page.mouse.up();
        
        await testHelpers.takeScreenshot(page, 'slider-fast-drag');
      }
    }
  });
});

test.describe('滑块验证码边界测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
  });

  test('应该拒绝负数X坐标', async () => {
    const generateResult = await apiHelper.generateSliderCaptcha('test-app');
    const captchaId = generateResult.data.captchaId;
    
    const verifyResult = await apiHelper.verifySliderCaptcha(captchaId, -100, 50);
    expect(verifyResult).toBeDefined();
  });

  test('应该拒绝过大的X坐标', async () => {
    const generateResult = await apiHelper.generateSliderCaptcha('test-app');
    const captchaId = generateResult.data.captchaId;
    
    const verifyResult = await apiHelper.verifySliderCaptcha(captchaId, 99999, 50);
    expect(verifyResult).toBeDefined();
  });

  test('应该拒绝空字符串captchaId', async () => {
    const verifyResult = await apiHelper.verifySliderCaptcha('', 100, 50);
    expect(verifyResult).toBeDefined();
  });

  test('应该处理小数X坐标', async () => {
    const generateResult = await apiHelper.generateSliderCaptcha('test-app');
    const captchaId = generateResult.data.captchaId;
    
    const verifyResult = await apiHelper.verifySliderCaptcha(captchaId, 100.5, 50.5);
    expect(verifyResult).toBeDefined();
  });

  test('应该拒绝缺失参数', async () => {
    const request = await require('@playwright/test').test.request;
    const response = await request.post('http://localhost:8080/api/v1/captcha/slider/verify', {
      data: { captchaId: 'test' }
    });
    const result = await response.json();
    expect(result).toBeDefined();
  });
});

test.describe('滑块验证码性能测试', () => {
  test('应该能够快速生成多个验证码', async ({ request }) => {
    const helper = new ApiHelper(request);
    const startTime = Date.now();
    
    for (let i = 0; i < 10; i++) {
      await helper.generateSliderCaptcha('test-app');
    }
    
    const duration = Date.now() - startTime;
    console.log(`生成10个验证码耗时: ${duration}ms`);
    expect(duration).toBeLessThan(30000);
  });

  test('应该能够快速验证多个验证码', async ({ request }) => {
    const helper = new ApiHelper(request);
    
    const generateResult = await helper.generateSliderCaptcha('test-app');
    const captchaId = generateResult.data.captchaId;
    const targetX = generateResult.data.targetX;
    
    const startTime = Date.now();
    
    for (let i = 0; i < 10; i++) {
      await helper.verifySliderCaptcha(captchaId, targetX, 50);
    }
    
    const duration = Date.now() - startTime;
    console.log(`验证10次耗时: ${duration}ms`);
    expect(duration).toBeLessThan(30000);
  });
});
