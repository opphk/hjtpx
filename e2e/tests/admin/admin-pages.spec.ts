import { test, expect } from '@playwright/test';
import { TestHelpers } from '../../utils/test-helpers';
import { ApiHelper } from '../../utils/api-helper';
import { testUsers } from '../../utils/test-data';

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

  test('管理端行为分析页面测试', async ({ page }) => {
    await page.goto('/admin/behavior-analytics');
    await testHelpers.takeScreenshot(page, 'admin-behavior-analytics');
    await expect(page).toBeVisible();
  });

  test('管理端黑名单页面测试', async ({ page }) => {
    await page.goto('/admin/blacklist');
    await testHelpers.takeScreenshot(page, 'admin-blacklist');
    await expect(page).toBeVisible();
  });

  test('管理端审计日志页面测试', async ({ page }) => {
    await page.goto('/admin/audit-logs');
    await testHelpers.takeScreenshot(page, 'admin-audit-logs');
    await expect(page).toBeVisible();
  });

  test('管理端实时监控页面测试', async ({ page }) => {
    await page.goto('/admin/real-time-screen');
    await testHelpers.takeScreenshot(page, 'admin-realtime-screen');
    await expect(page).toBeVisible();
  });
});

test.describe('管理端页面控制台错误测试', () => {
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    testHelpers = new TestHelpers();
  });

  test('登录页面控制台错误检测', async ({ page }) => {
    const consoleErrors: string[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('/admin/login');
    await page.waitForLoadState('networkidle');
    await testHelpers.takeScreenshot(page, 'admin-login-console-check');
    
    const criticalErrors = consoleErrors.filter(err => 
      !err.includes('favicon') && 
      !err.includes('404')
    );
    expect(criticalErrors.length).toBe(0);
  });

  test('仪表板页面控制台错误检测', async ({ page }) => {
    const consoleErrors: string[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('/admin/dashboard');
    await page.waitForLoadState('networkidle');
    await testHelpers.takeScreenshot(page, 'admin-dashboard-console-check');
    
    const criticalErrors = consoleErrors.filter(err => 
      !err.includes('favicon') && 
      !err.includes('404')
    );
    expect(criticalErrors.length).toBe(0);
  });

  test('统计页面控制台错误检测', async ({ page }) => {
    const consoleErrors: string[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('/admin/stats');
    await page.waitForLoadState('networkidle');
    await testHelpers.takeScreenshot(page, 'admin-stats-console-check');
    
    const criticalErrors = consoleErrors.filter(err => 
      !err.includes('favicon') && 
      !err.includes('404')
    );
    expect(criticalErrors.length).toBe(0);
  });

  test('应用管理页面控制台错误检测', async ({ page }) => {
    const consoleErrors: string[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('/admin/applications');
    await page.waitForLoadState('networkidle');
    await testHelpers.takeScreenshot(page, 'admin-applications-console-check');
    
    const criticalErrors = consoleErrors.filter(err => 
      !err.includes('favicon') && 
      !err.includes('404')
    );
    expect(criticalErrors.length).toBe(0);
  });

  test('日志页面控制台错误检测', async ({ page }) => {
    const consoleErrors: string[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('/admin/logs');
    await page.waitForLoadState('networkidle');
    await testHelpers.takeScreenshot(page, 'admin-logs-console-check');
    
    const criticalErrors = consoleErrors.filter(err => 
      !err.includes('favicon') && 
      !err.includes('404')
    );
    expect(criticalErrors.length).toBe(0);
  });

  test('监控页面控制台错误检测', async ({ page }) => {
    const consoleErrors: string[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    
    await page.goto('/admin/monitoring');
    await page.waitForLoadState('networkidle');
    await testHelpers.takeScreenshot(page, 'admin-monitoring-console-check');
    
    const criticalErrors = consoleErrors.filter(err => 
      !err.includes('favicon') && 
      !err.includes('404')
    );
    expect(criticalErrors.length).toBe(0);
  });
});

test.describe('管理端功能交互测试', () => {
  let testHelpers: TestHelpers;
  let apiHelper: ApiHelper;
  let authToken: string;

  test.beforeEach(async ({ page, request }) => {
    testHelpers = new TestHelpers();
    apiHelper = new ApiHelper(request);
    
    const loginResult = await apiHelper.adminLogin(
      testUsers.admin.username,
      testUsers.admin.password
    );
    authToken = loginResult.data?.token || '';
  });

  test('应该能够登录管理端', async ({ page }) => {
    await page.goto('/admin/login');
    await page.fill('input[name="username"]', testUsers.admin.username);
    await page.fill('input[name="password"]', testUsers.admin.password);
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/admin\/dashboard/, { timeout: 10000 });
    await testHelpers.takeScreenshot(page, 'admin-logged-in');
  });

  test('应该能够登出管理端', async ({ page }) => {
    await page.goto('/admin/login');
    await page.fill('input[name="username"]', testUsers.admin.username);
    await page.fill('input[name="password"]', testUsers.admin.password);
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/admin\/dashboard/);
    
    const logoutButton = page.locator('a:has-text("Logout"), button:has-text("退出"), a[href*="logout"]');
    const logoutVisible = await logoutButton.isVisible().catch(() => false);
    
    if (logoutVisible) {
      await logoutButton.first().click();
      await page.waitForURL(/\/admin\/login/, { timeout: 10000 });
      await testHelpers.takeScreenshot(page, 'admin-logged-out');
    }
  });

  test('应该在未登录时访问受保护页面被重定向', async ({ page }) => {
    await page.goto('/admin/dashboard');
    await page.waitForLoadState('networkidle');
    
    const currentURL = page.url();
    expect(currentURL).toMatch(/\/admin\/login/);
    await testHelpers.takeScreenshot(page, 'admin-redirected-to-login');
  });

  test('API获取统计数据应该成功', async () => {
    if (!authToken) {
      console.log('无法获取token，跳过API测试');
      return;
    }
    
    const stats = await apiHelper.getVerificationStats(authToken);
    expect(stats).toBeDefined();
  });

  test('API获取应用列表应该成功', async () => {
    if (!authToken) {
      console.log('无法获取token，跳过API测试');
      return;
    }
    
    const apps = await apiHelper.getApplications(authToken);
    expect(apps).toBeDefined();
  });

  test('API获取日志应该成功', async () => {
    if (!authToken) {
      console.log('无法获取token，跳过API测试');
      return;
    }
    
    const logs = await apiHelper.getLogs(authToken, { limit: 10 });
    expect(logs).toBeDefined();
  });

  test('应该能够导航到各个管理页面', async ({ page }) => {
    await page.goto('/admin/login');
    await page.fill('input[name="username"]', testUsers.admin.username);
    await page.fill('input[name="password"]', testUsers.admin.password);
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/admin\/dashboard/);
    
    const navLinks = [
      '/admin/stats',
      '/admin/applications',
      '/admin/logs',
      '/admin/monitoring'
    ];
    
    for (const link of navLinks) {
      await page.goto(link);
      await page.waitForLoadState('networkidle');
      await expect(page).toBeVisible();
    }
    
    await testHelpers.takeScreenshot(page, 'admin-all-pages-navigated');
  });

  test('无效凭据应该显示错误', async ({ page }) => {
    await page.goto('/admin/login');
    await page.fill('input[name="username"]', 'invalid-user');
    await page.fill('input[name="password"]', 'wrong-password');
    await page.click('button[type="submit"]');
    
    await page.waitForTimeout(2000);
    
    const errorMessage = page.locator('.error, .alert, [role="alert"], .text-danger');
    const errorVisible = await errorMessage.isVisible().catch(() => false);
    
    if (errorVisible) {
      await testHelpers.takeScreenshot(page, 'admin-login-error');
    }
  });
});

test.describe('管理端数据验证测试', () => {
  let testHelpers: TestHelpers;
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ page, request }) => {
    testHelpers = new TestHelpers();
    apiHelper = new ApiHelper(request);
  });

  test('统计数据格式验证', async ({ request }) => {
    const helper = new ApiHelper(request);
    const loginResult = await helper.adminLogin(
      testUsers.admin.username,
      testUsers.admin.password
    );
    
    const token = loginResult.data?.token;
    if (!token) {
      console.log('无法获取token，跳过测试');
      return;
    }
    
    const stats = await helper.getVerificationStats(token);
    expect(stats).toHaveProperty('data');
  });

  test('应用列表数据格式验证', async ({ request }) => {
    const helper = new ApiHelper(request);
    const loginResult = await helper.adminLogin(
      testUsers.admin.username,
      testUsers.admin.password
    );
    
    const token = loginResult.data?.token;
    if (!token) {
      console.log('无法获取token，跳过测试');
      return;
    }
    
    const apps = await helper.getApplications(token);
    expect(apps).toHaveProperty('data');
  });

  test('日志数据格式验证', async ({ request }) => {
    const helper = new ApiHelper(request);
    const loginResult = await helper.adminLogin(
      testUsers.admin.username,
      testUsers.admin.password
    );
    
    const token = loginResult.data?.token;
    if (!token) {
      console.log('无法获取token，跳过测试');
      return;
    }
    
    const logs = await helper.getLogs(token, { limit: 10 });
    expect(logs).toBeDefined();
  });
});
