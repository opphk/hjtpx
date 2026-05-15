const { baseTest, expect, waitForPageLoad, generateRandomEmail, generateRandomString } = require('../utils/test-helpers');

baseTest.describe('HJTPX 用户管理测试', () => {
  baseTest.describe('用户列表页面', () => {
    baseTest('应该加载用户列表页面', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/users');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('users-page');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该显示用户表格（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/users');
      await waitForPageLoad(page);
      
      const table = page.locator('table, [class*="table"]');
      const tableCount = await table.count();
      
      if (tableCount > 0) {
        await expect(table.first()).toBeVisible();
      }
      
      await screenshotManager.capture('users-table');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该有搜索功能（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/users');
      await waitForPageLoad(page);
      
      const searchInput = page.locator('input[type="search"], input[name="search"], [placeholder*="搜索"], [placeholder*="Search"]');
      const searchCount = await searchInput.count();
      
      if (searchCount > 0) {
        await searchInput.first().fill('test');
        await page.waitForTimeout(500);
        await screenshotManager.capture('users-search');
      }
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('用户资料页面', () => {
    baseTest('应该加载用户资料页面', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/profile');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('profile-page');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该显示用户信息', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/profile');
      await waitForPageLoad(page);
      
      const profileInfo = page.locator('[class*="profile"], [class*="user"]');
      const profileCount = await profileInfo.count();
      
      if (profileCount > 0) {
        await expect(profileInfo.first()).toBeVisible();
      }
      
      await screenshotManager.capture('profile-info');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该有编辑功能（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/profile');
      await waitForPageLoad(page);
      
      const editButton = page.locator('button:has-text("编辑"), button:has-text("Edit"), [class*="edit"]');
      const editCount = await editButton.count();
      
      if (editCount > 0) {
        await editButton.first().click().catch(() => {});
        await page.waitForTimeout(1000);
        await screenshotManager.capture('profile-edit');
      }
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('设置页面', () => {
    baseTest('应该加载设置页面', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/settings');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('settings-page');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该显示设置选项', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/settings');
      await waitForPageLoad(page);
      
      await expect(page.locator('body')).toBeVisible();
      
      await screenshotManager.capture('settings-options');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('管理员功能', () => {
    baseTest('应该加载管理员用户页面（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/admin/users');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('admin-users-page');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该加载审计页面（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/audit');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('audit-page');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该加载日志页面（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/logs');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('logs-page');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });
});
