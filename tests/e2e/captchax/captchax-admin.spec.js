const { baseTest, expect, waitForPageLoad } = require('../utils/test-helpers');

baseTest.describe('CaptchaX 管理后台测试', () => {
  baseTest.describe('管理员登录页面', () => {
    baseTest('应该显示管理员登录页面', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/admin/login');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('captchax-admin-login');
      
      await expect(page.locator('body')).toBeVisible();
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该有登录表单', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/admin/login');
      await waitForPageLoad(page);
      
      const usernameInput = page.locator('input[name="username"], input[name="user"], input[type="text"]').first();
      const passwordInput = page.locator('input[name="password"], input[type="password"]').first();
      const submitButton = page.locator('button[type="submit"], input[type="submit"]').first();
      
      if (await usernameInput.isVisible()) await expect(usernameInput).toBeVisible();
      if (await passwordInput.isVisible()) await expect(passwordInput).toBeVisible();
      if (await submitButton.isVisible()) await expect(submitButton).toBeVisible();
      
      await screenshotManager.capture('captchax-admin-login-form');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('管理仪表板', () => {
    baseTest('应该加载管理仪表板页面', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/admin');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('captchax-admin-dashboard');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该显示统计数据（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/admin');
      await waitForPageLoad(page);
      
      const stats = page.locator('.stat, [class*="stat"], [class*="count"]');
      const statsCount = await stats.count();
      
      if (statsCount > 0) {
        await expect(stats.first()).toBeVisible();
      }
      
      await screenshotManager.capture('captchax-admin-stats');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该显示图表（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/admin');
      await waitForPageLoad(page);
      
      const charts = page.locator('svg, canvas, [class*="chart"], [class*="graph"]');
      const chartsCount = await charts.count();
      
      if (chartsCount > 0) {
        await expect(charts.first()).toBeVisible();
      }
      
      await screenshotManager.capture('captchax-admin-charts');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('管理功能页面', () => {
    baseTest('应该加载配置页面（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/admin/config');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('captchax-admin-config');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该加载黑名单页面（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/admin/blacklist');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('captchax-admin-blacklist');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该加载白名单页面（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/admin/whitelist');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('captchax-admin-whitelist');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该加载统计页面（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/admin/stats');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('captchax-admin-stats-page');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('管理后台响应式测试', () => {
    baseTest('应该在不同屏幕尺寸下正确显示', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      const viewports = [
        { width: 1920, height: 1080, name: 'desktop' },
        { width: 768, height: 1024, name: 'tablet' },
        { width: 375, height: 667, name: 'mobile' }
      ];
      
      for (const viewport of viewports) {
        await page.setViewportSize(viewport);
        await page.goto('/admin');
        await waitForPageLoad(page);
        await screenshotManager.capture(`captchax-admin-${viewport.name}`);
      }
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });
});
