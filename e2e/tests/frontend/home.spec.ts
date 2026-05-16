import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';

test.describe('用户端首页测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ page, request }) => {
    apiHelper = new ApiHelper(request);
  });

  test('应该能够访问首页', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/行为验证系统/);
    await expect(page.locator('h1')).toBeVisible();
  });

  test('首页应该包含导航链接', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('nav')).toBeVisible();
  });

  test('健康检查API应该正常工作', async () => {
    const isHealthy = await apiHelper.healthCheck();
    expect(isHealthy).toBeTruthy();
  });
});
