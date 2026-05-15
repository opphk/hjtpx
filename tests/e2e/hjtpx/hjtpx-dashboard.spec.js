const { baseTest, expect, waitForPageLoad } = require('../utils/test-helpers');

baseTest.describe('HJTPX 仪表板测试', () => {
  baseTest.describe('仪表板页面', () => {
    baseTest('应该成功加载仪表板页面', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/dashboard');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('dashboard-page');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该显示主要内容区域', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/dashboard');
      await waitForPageLoad(page);
      
      await expect(page.locator('body')).toBeVisible();
      
      await screenshotManager.capture('dashboard-content');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该显示统计卡片（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/dashboard');
      await waitForPageLoad(page);
      
      const cards = page.locator('.card, [class*="stat"], [class*="count"]');
      const cardCount = await cards.count();
      
      if (cardCount > 0) {
        await expect(cards.first()).toBeVisible();
      }
      
      await screenshotManager.capture('dashboard-cards');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该显示图表（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/dashboard');
      await waitForPageLoad(page);
      
      const charts = page.locator('svg, canvas, [class*="chart"], [class*="graph"]');
      const chartCount = await charts.count();
      
      if (chartCount > 0) {
        await expect(charts.first()).toBeVisible();
      }
      
      await screenshotManager.capture('dashboard-charts');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('仪表板导航', () => {
    baseTest('应该有导航菜单', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/');
      await waitForPageLoad(page);
      
      const nav = page.locator('nav, [class*="nav"], [class*="menu"], header');
      await expect(nav.first()).toBeVisible();
      
      await screenshotManager.capture('navigation-menu');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该能在页面之间导航', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/');
      await waitForPageLoad(page);
      
      const navLinks = page.locator('nav a, header a, [class*="nav"] a');
      const linkCount = await navLinks.count();
      
      if (linkCount > 0) {
        const firstLink = navLinks.first();
        await firstLink.click().catch(() => {});
        await page.waitForTimeout(1000);
        await screenshotManager.capture('after-navigation');
      }
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该有响应式设计', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      const viewports = [
        { width: 1920, height: 1080, name: 'desktop' },
        { width: 768, height: 1024, name: 'tablet' },
        { width: 375, height: 667, name: 'mobile' }
      ];
      
      for (const viewport of viewports) {
        await page.setViewportSize(viewport);
        await page.goto('/');
        await waitForPageLoad(page);
        await screenshotManager.capture(`dashboard-${viewport.name}`);
      }
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('性能测试', () => {
    baseTest('仪表板页面应该快速加载', async ({ page, testHelpers }) => {
      const { consoleChecker } = testHelpers;
      
      const startTime = Date.now();
      await page.goto('/dashboard');
      await waitForPageLoad(page);
      const loadTime = Date.now() - startTime;
      
      console.log(`仪表板加载时间: ${loadTime}ms`);
      
      expect(loadTime).toBeLessThan(10000);
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });
});
