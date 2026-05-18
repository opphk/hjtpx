import { test, expect } from '@playwright/test';
import { TestHelpers } from '../../utils/test-helpers';

test.describe('管理端页面完整测试', () => {
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    testHelpers = new TestHelpers();
  });

  test('管理端仪表板页面测试', async ({ page }) => {
    await page.goto('/admin');
    await testHelpers.takeScreenshot(page, 'admin-dashboard');
    
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('管理端统计页面测试', async ({ page }) => {
    await page.goto('/admin/stats');
    await testHelpers.takeScreenshot(page, 'admin-stats-page');
    
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('管理端应用管理页面测试', async ({ page }) => {
    await page.goto('/admin/applications');
    await testHelpers.takeScreenshot(page, 'admin-applications-page');
    
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('管理端日志页面测试', async ({ page }) => {
    await page.goto('/admin/logs');
    await testHelpers.takeScreenshot(page, 'admin-logs-page');
    
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('管理端监控页面测试', async ({ page }) => {
    await page.goto('/admin/monitoring');
    await testHelpers.takeScreenshot(page, 'admin-monitoring-page');
    
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('管理端高级分析页面测试', async ({ page }) => {
    await page.goto('/admin/advanced-analytics');
    await testHelpers.takeScreenshot(page, 'admin-analytics-page');
    
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });
});
