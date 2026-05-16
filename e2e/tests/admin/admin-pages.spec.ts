import { test, expect } from '@playwright/test';
import { TestHelpers } from '../../utils/test-helpers';

test.describe('管理端页面完整测试', () => {
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    testHelpers = new TestHelpers();
  });

  test('管理端登录页面测试', async ({ page }) => {
    console.log('正在测试管理端登录页面...');
    await page.goto('/admin/login');
    await testHelpers.takeScreenshot(page, 'admin-login-page');
    
    await expect(page).toBeVisible();
    console.log('✅ 管理端登录页面加载成功');
  });

  test('管理端仪表板页面测试', async ({ page }) => {
    console.log('正在测试管理端仪表板...');
    await page.goto('/admin');
    await testHelpers.takeScreenshot(page, 'admin-dashboard');
    
    await expect(page).toBeVisible();
    console.log('✅ 管理端仪表板页面加载成功');
  });

  test('管理端统计页面测试', async ({ page }) => {
    console.log('正在测试管理端统计页面...');
    await page.goto('/admin/stats');
    await testHelpers.takeScreenshot(page, 'admin-stats-page');
    
    await expect(page).toBeVisible();
    console.log('✅ 管理端统计页面加载成功');
  });

  test('管理端应用管理页面测试', async ({ page }) => {
    console.log('正在测试管理端应用管理页面...');
    await page.goto('/admin/applications');
    await testHelpers.takeScreenshot(page, 'admin-applications-page');
    
    await expect(page).toBeVisible();
    console.log('✅ 管理端应用管理页面加载成功');
  });

  test('管理端日志页面测试', async ({ page }) => {
    console.log('正在测试管理端日志页面...');
    await page.goto('/admin/logs');
    await testHelpers.takeScreenshot(page, 'admin-logs-page');
    
    await expect(page).toBeVisible();
    console.log('✅ 管理端日志页面加载成功');
  });

  test('管理端监控页面测试', async ({ page }) => {
    console.log('正在测试管理端监控页面...');
    await page.goto('/admin/monitoring');
    await testHelpers.takeScreenshot(page, 'admin-monitoring-page');
    
    await expect(page).toBeVisible();
    console.log('✅ 管理端监控页面加载成功');
  });

  test('管理端高级分析页面测试', async ({ page }) => {
    console.log('正在测试管理端高级分析页面...');
    await page.goto('/admin/advanced-analytics');
    await testHelpers.takeScreenshot(page, 'admin-analytics-page');
    
    await expect(page).toBeVisible();
    console.log('✅ 管理端高级分析页面加载成功');
  });
});
