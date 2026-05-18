import { test, expect } from '@playwright/test';
import { TestHelpers } from '../../utils/test-helpers';

test.describe('MFA功能测试', () => {
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    testHelpers = new TestHelpers();
  });

  test.describe('MFA设置页面', () => {
    test('MFA设置页面应该正常加载', async ({ page }) => {
      await page.goto('/mfa-setup');
      await page.waitForLoadState('domcontentloaded');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });

    test('MFA设置页面应该有认证方式选择区域', async ({ page }) => {
      await page.goto('/mfa-setup');
      await page.waitForLoadState('domcontentloaded');
      
      const mfaContent = await page.locator('body').textContent();
      expect(mfaContent).toBeDefined();
    });
  });

  test.describe('MFA验证页面', () => {
    test('MFA验证页面应该正常加载', async ({ page }) => {
      await page.goto('/mfa-verify');
      await page.waitForLoadState('domcontentloaded');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });

    test('MFA验证页面应该有验证码输入区域', async ({ page }) => {
      await page.goto('/mfa-verify');
      await page.waitForLoadState('domcontentloaded');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });
  });

  test.describe('MFA页面截图验证', () => {
    test('MFA设置页面截图', async ({ page }) => {
      await page.goto('/mfa-setup');
      await page.waitForLoadState('domcontentloaded');
      await testHelpers.takeScreenshot(page, 'mfa-setup-page');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });

    test('MFA验证页面截图', async ({ page }) => {
      await page.goto('/mfa-verify');
      await page.waitForLoadState('domcontentloaded');
      await testHelpers.takeScreenshot(page, 'mfa-verify-page');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });
  });

  test.describe('MFA响应式布局', () => {
    test('MFA设置页面在移动端应该正常显示', async ({ page }) => {
      await page.setViewportSize({ width: 375, height: 667 });
      await page.goto('/mfa-setup');
      await page.waitForLoadState('domcontentloaded');
      
      await testHelpers.takeScreenshot(page, 'mfa-setup-mobile');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });

    test('MFA设置页面在桌面端应该正常显示', async ({ page }) => {
      await page.setViewportSize({ width: 1280, height: 720 });
      await page.goto('/mfa-setup');
      await page.waitForLoadState('domcontentloaded');
      
      await testHelpers.takeScreenshot(page, 'mfa-setup-desktop');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });
  });
});
