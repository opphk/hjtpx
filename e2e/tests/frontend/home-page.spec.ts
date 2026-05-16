import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';
import { TestHelpers } from '../../utils/test-helpers';

test.describe('用户端首页完整测试', () => {
  let apiHelper: ApiHelper;
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ page, request }) => {
    apiHelper = new ApiHelper(request);
    testHelpers = new TestHelpers();
  });

  test('首页加载测试', async ({ page }) => {
    console.log('正在测试首页加载...');
    await page.goto('/');
    await testHelpers.takeScreenshot(page, 'home-page-loaded');
    await expect(page).toBeVisible();
    console.log('✅ 首页加载成功');
  });

  test('检查控制台错误', async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('/');
    
    console.log('发现的控制台错误:', consoleErrors);
    expect(consoleErrors.length).toBe(0);
    await testHelpers.takeScreenshot(page, 'home-page-no-errors');
  });

  test('健康检查API测试', async () => {
    const isHealthy = await apiHelper.healthCheck();
    expect(isHealthy).toBeTruthy();
    console.log('✅ 健康检查API正常');
  });
});
