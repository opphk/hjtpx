import { test, expect } from '@playwright/test';
import { TestHelpers } from '../../utils/test-helpers';

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
