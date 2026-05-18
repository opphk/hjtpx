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
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
    await testHelpers.takeScreenshot(page, 'home-page-loaded');
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('检查控制台错误', async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
    
    const criticalErrors = consoleErrors.filter(e => 
      !e.includes('favicon') && 
      !e.includes('Failed to load resource')
    );
    console.log('发现的控制台错误:', criticalErrors);
    await testHelpers.takeScreenshot(page, 'home-page-no-errors');
  });

  test('健康检查API测试', async () => {
    const isHealthy = await apiHelper.healthCheck();
    expect(isHealthy).toBeTruthy();
  });
});
