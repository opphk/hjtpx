import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';
import { testUsers } from '../../utils/test-data';

test.describe('管理端功能测试', () => {
  let apiHelper: ApiHelper;
  let authToken: string;

  test.beforeEach(async ({ request, page }) => {
    apiHelper = new ApiHelper(request);
    const loginResult = await apiHelper.adminLogin(
      testUsers.admin.username,
      testUsers.admin.password
    );
    authToken = loginResult.data.token;
  });

  test.describe('仪表板', () => {
    test('应该能够访问仪表板页面', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      await expect(page.locator('h1, h2')).toContainText(/Dashboard|仪表板/);
    });

    test('API应该能够获取验证统计数据', async () => {
      const stats = await apiHelper.getVerificationStats(authToken);
      expect(stats).toHaveProperty('success', true);
      expect(stats).toHaveProperty('data');
    });
  });

  test.describe('应用管理', () => {
    test('API应该能够获取应用列表', async () => {
      const apps = await apiHelper.getApplications(authToken);
      expect(apps).toHaveProperty('success', true);
      expect(apps).toHaveProperty('data');
    });

    test('API应该能够创建新应用', async () => {
      const appName = 'Test App - ' + Date.now();
      const result = await apiHelper.createApplication(
        authToken,
        appName,
        'Test application'
      );
      expect(result).toHaveProperty('success', true);
    });
  });

  test.describe('日志查询', () => {
    test('API应该能够获取验证日志', async () => {
      const logs = await apiHelper.getLogs(authToken, { limit: 10 });
      expect(logs).toHaveProperty('success', true);
      expect(logs).toHaveProperty('data');
    });
  });

  test.describe('导航菜单', () => {
    test('仪表板应该有导航菜单', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      
      await expect(page.locator('nav, .sidebar, .menu')).toBeVisible();
    });
  });
});
