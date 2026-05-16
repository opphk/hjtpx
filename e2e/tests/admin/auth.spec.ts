import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';
import { testUsers } from '../../utils/test-data';

test.describe('管理端认证测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
  });

  test.describe('管理员登录', () => {
    test('应该能够访问登录页面', async ({ page }) => {
      await page.goto('/admin/login');
      await expect(page).toHaveTitle(/登录/);
      await expect(page.locator('form')).toBeVisible();
    });

    test('应该能够使用正确的凭据登录', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
    });

    test('应该拒绝无效凭据应该显示错误', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', 'invalid-user');
      await page.fill('input[name="password"]', 'wrong-password');
      await page.click('button[type="submit"]');
      await expect(page.locator('.error, .alert, [role="alert"]')).toBeVisible();
    });
  });

  test.describe('管理员登出', () => {
    test('应该能够成功登出', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      await page.click('text=Logout, text=退出, a[href*="logout"]');
      await expect(page).toHaveURL(/\/admin\/login/);
    });
  });

  test.describe('API认证测试', () => {
    test('API登录API应该能够正常登录', async () => {
      const result = await apiHelper.adminLogin(
        testUsers.admin.username,
        testUsers.admin.password
      );
      expect(result).toHaveProperty('success', true);
      expect(result).toHaveProperty('data');
      expect(result.data).toHaveProperty('token');
    });

    test('API无效的凭据应该返回错误', async () => {
      const result = await apiHelper.adminLogin('invalid', 'invalid');
      expect(result.success).toBeFalsy();
    });
  });
});
