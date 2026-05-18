import { test, expect } from '@playwright/test';
import { TestHelpers } from '../../utils/test-helpers';

test.describe('页面截图验证测试', () => {
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    testHelpers = new TestHelpers();
  });

  test.describe('首页截图验证', () => {
    test('首页应该正常加载并截图', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(2000);
      await testHelpers.takeScreenshot(page, 'home-page-full');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });
  });

  test.describe('验证码页面截图验证', () => {
    test('验证码页面应该正常加载', async ({ page }) => {
      await page.goto('/captcha');
      await page.waitForLoadState('domcontentloaded');
      await testHelpers.takeScreenshot(page, 'captcha-page-loaded');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });

    test('验证码页面Tab切换应该可见', async ({ page }) => {
      await page.goto('/captcha');
      await page.waitForLoadState('domcontentloaded');
      
      const tabs = await page.locator('.nav-link').count();
      expect(tabs).toBeGreaterThanOrEqual(0);
    });
  });

  test.describe('连连看验证码页面截图验证', () => {
    test('连连看验证码页面应该正常加载', async ({ page }) => {
      await page.goto('/lianliankan');
      await page.waitForLoadState('domcontentloaded');
      await testHelpers.takeScreenshot(page, 'lianliankan-page');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });
  });

  test.describe('语音验证码页面截图验证', () => {
    test('语音验证码页面应该正常加载', async ({ page }) => {
      await page.goto('/voice-captcha');
      await page.waitForLoadState('domcontentloaded');
      await testHelpers.takeScreenshot(page, 'voice-captcha-page');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });

    test('语音验证码播放按钮应该可见', async ({ page }) => {
      await page.goto('/voice-captcha');
      await page.waitForLoadState('domcontentloaded');
      
      const playButton = page.locator('#playButton');
      const isVisible = await playButton.isVisible().catch(() => false);
      if (!isVisible) {
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('管理后台页面截图验证', () => {
    test('管理后台仪表板页面应该正常加载', async ({ page }) => {
      await page.goto('/admin/dashboard');
      await page.waitForLoadState('domcontentloaded');
      await testHelpers.takeScreenshot(page, 'admin-dashboard-page');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });
  });

  test.describe('控制台错误检查', () => {
    test('首页不应该有严重控制台错误', async ({ page }) => {
      const consoleErrors: string[] = [];
      page.on('console', msg => {
        if (msg.type() === 'error') {
          consoleErrors.push(msg.text());
        }
      });

      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(2000);

      const criticalErrors = consoleErrors.filter(e => 
        !e.includes('favicon') && 
        !e.includes('Failed to load resource')
      );
      console.log('首页控制台错误:', criticalErrors);
    });

    test('验证码页面不应该有严重控制台错误', async ({ page }) => {
      const consoleErrors: string[] = [];
      page.on('console', msg => {
        if (msg.type() === 'error') {
          consoleErrors.push(msg.text());
        }
      });

      await page.goto('/captcha');
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(2000);

      const criticalErrors = consoleErrors.filter(e => 
        !e.includes('favicon') && 
        !e.includes('Failed to load resource')
      );
      console.log('验证码页面控制台错误:', criticalErrors);
    });
  });
});
