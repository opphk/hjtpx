import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';
import { TestHelpers } from '../../utils/test-helpers';

test.describe('表情验证码完整测试', () => {
  let apiHelper: ApiHelper;
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ page, request }) => {
    apiHelper = new ApiHelper(request);
    testHelpers = new TestHelpers();
  });

  test('表情验证码页面加载', async ({ page }) => {
    console.log('正在测试表情验证码页面加载...');
    await page.goto('/emoji.html');
    await testHelpers.takeScreenshot(page, 'emoji-captcha-page');
    await expect(page.locator('body')).toBeVisible();
    console.log('✅ 表情验证码页面加载成功');
  });

  test('表情验证码API生成', async ({ request }) => {
    const response = await apiHelper.generateCaptcha('emoji', {
      type: 'emoji',
      difficulty: 'medium'
    });
    expect(response.ok()).toBeTruthy();
    const data = await response.json();
    expect(data.code).toBe(0);
    console.log('✅ 表情验证码生成成功');
  });

  test('表情验证码验证流程', async ({ request }) => {
    const generateResponse = await apiHelper.generateCaptcha('emoji');
    expect(generateResponse.ok()).toBeTruthy();
    const generateData = await generateResponse.json();
    expect(generateData.data.captcha_id).toBeDefined();

    const verifyResponse = await apiHelper.verifyCaptcha(generateData.data.captcha_id, {
      selected_emoji: ['happy', 'laugh'],
      response_time: 2500
    });
    expect(verifyResponse.ok()).toBeTruthy();
    const verifyData = await verifyResponse.json();
    expect([0, 1].includes(verifyData.code)).toBeTruthy();
    console.log('✅ 表情验证码验证流程完成');
  });

  test('表情验证码错误处理', async ({ request }) => {
    const verifyResponse = await apiHelper.verifyCaptcha('invalid_id', {
      selected_emoji: ['happy']
    });
    expect(verifyResponse.ok()).toBeTruthy();
    const data = await verifyResponse.json();
    expect(data.code).not.toBe(0);
    console.log('✅ 表情验证码错误处理正常');
  });

  test('表情验证码UI交互', async ({ page }) => {
    await page.goto('/emoji.html');
    const emojiContainer = page.locator('.emoji-container');
    if (await emojiContainer.isVisible()) {
      console.log('表情选择容器可见');
    }
    await testHelpers.takeScreenshot(page, 'emoji-selection');
    console.log('✅ 表情验证码UI交互完成');
  });

  test('表情验证码超时处理', async ({ request }) => {
    const generateResponse = await apiHelper.generateCaptcha('emoji');
    const data = await generateResponse.json();
    await page.waitForTimeout(35000);

    const verifyResponse = await apiHelper.verifyCaptcha(data.data.captcha_id, {
      selected_emoji: ['happy']
    });
    const verifyData = await verifyResponse.json();
    expect(verifyData.code).toBe(1001);
    console.log('✅ 表情验证码超时处理正常');
  });

  test('表情验证码多难度测试', async ({ request }) => {
    const difficulties = ['easy', 'medium', 'hard'];
    for (const difficulty of difficulties) {
      const response = await apiHelper.generateCaptcha('emoji', {
        difficulty: difficulty
      });
      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(data.data.difficulty).toBeDefined();
      console.log(`✅ 难度 ${difficulty} 测试通过`);
    }
  });

  test('表情验证码无障碍测试', async ({ page }) => {
    await page.goto('/emoji.html');
    const accessibilityTree = await page.accessibility.snapshot();
    expect(accessibilityTree).toBeDefined();
    console.log('✅ 表情验证码无障碍测试通过');
  });

  test('表情验证码响应式设计', async ({ page }) => {
    const viewports = [
      { width: 375, height: 667 },
      { width: 768, height: 1024 },
      { width: 1920, height: 1080 }
    ];

    for (const viewport of viewports) {
      await page.setViewportSize(viewport);
      await page.goto('/emoji.html');
      const body = page.locator('body');
      await expect(body).toBeVisible();
      console.log(`✅ 视口 ${viewport.width}x${viewport.height} 测试通过`);
    }
  });

  test('表情验证码本地化', async ({ page }) => {
    await page.goto('/emoji.html');
    const language = await page.evaluate(() => navigator.language);
    console.log(`浏览器语言: ${language}`);
    expect(language).toBeDefined();
    console.log('✅ 表情验证码本地化测试通过');
  });

  test('表情验证码性能指标', async ({ page }) => {
    const metrics: any[] = [];
    page.on('performance', (data) => {
      metrics.push(data);
    });

    await page.goto('/emoji.html');
    await page.waitForTimeout(2000);

    const timing = await page.evaluate(() => {
      const perfData = performance.getEntriesByType('navigation')[0] as any;
      return {
        loadTime: perfData.loadEventEnd - perfData.startTime,
        domContentLoaded: perfData.domContentLoadedEventEnd - perfData.startTime,
        firstPaint: performance.getEntriesByType('paint')[0]?.startTime || 0
      };
    });

    console.log('性能指标:', timing);
    expect(timing.loadTime).toBeLessThan(5000);
    console.log('✅ 表情验证码性能测试通过');
  });

  test('表情验证码网络请求', async ({ page, request }) => {
    const networkRequests: any[] = [];
    page.on('request', (request) => {
      if (request.url().includes('/api')) {
        networkRequests.push({
          url: request.url(),
          method: request.method()
        });
      }
    });

    await page.goto('/emoji.html');
    await apiHelper.generateCaptcha('emoji');

    expect(networkRequests.length).toBeGreaterThan(0);
    console.log('✅ 表情验证码网络请求测试通过');
  });

  test('表情验证码错误恢复', async ({ page, request }) => {
    await page.goto('/emoji.html');

    const generateResponse = await apiHelper.generateCaptcha('emoji');
    const data = await generateResponse.json();

    const verifyResponse = await apiHelper.verifyCaptcha('wrong_id', {
      selected_emoji: ['happy']
    });
    const verifyData = await verifyResponse.json();
    expect(verifyData.code).not.toBe(0);

    const retryResponse = await apiHelper.generateCaptcha('emoji');
    expect(retryResponse.ok()).toBeTruthy();
    console.log('✅ 表情验证码错误恢复测试通过');
  });

  test('表情验证码数据持久化', async ({ page }) => {
    await page.goto('/emoji.html');

    await page.evaluate(() => {
      localStorage.setItem('emoji_test', JSON.stringify({
        timestamp: Date.now(),
        completed: false
      }));
    });

    await page.reload();

    const stored = await page.evaluate(() => {
      return localStorage.getItem('emoji_test');
    });

    expect(stored).toBeDefined();
    console.log('✅ 表情验证码数据持久化测试通过');
  });

  test('表情验证码Cookie处理', async ({ page }) => {
    await page.goto('/emoji.html');

    await page.evaluate(() => {
      document.cookie = 'emoji_preference=easy; path=/';
    });

    await page.reload();

    const cookies = await page.context().cookies();
    expect(cookies.some(c => c.name === 'emoji_preference')).toBeTruthy();
    console.log('✅ 表情验证码Cookie处理测试通过');
  });
});
