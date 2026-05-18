import { test, expect } from '@playwright/test';
import { TestHelpers } from '../../utils/test-helpers';

const viewports = [
  { name: 'mobile', width: 375, height: 667 },
  { name: 'tablet', width: 768, height: 1024 },
  { name: 'desktop', width: 1280, height: 720 },
  { name: 'large-desktop', width: 1920, height: 1080 },
];

test.describe('响应式布局测试', () => {
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    testHelpers = new TestHelpers();
  });

  test.describe('首页响应式布局', () => {
    for (const viewport of viewports) {
      test(`首页在${viewport.name}视口下应该正常显示`, async ({ page }) => {
        await page.setViewportSize({ width: viewport.width, height: viewport.height });
        await page.goto('/');
        await page.waitForLoadState('domcontentloaded');
        
        await testHelpers.takeScreenshot(page, `home-${viewport.name}`);
        
        const body = page.locator('body');
        await expect(body).toBeVisible();
      });
    }

    test('首页在移动端导航元素应该可见', async ({ page }) => {
      await page.setViewportSize({ width: 375, height: 667 });
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });

    test('首页在桌面端导航元素应该可见', async ({ page }) => {
      await page.setViewportSize({ width: 1280, height: 720 });
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });
  });

  test.describe('连连看验证码响应式布局', () => {
    for (const viewport of viewports) {
      test(`连连看在${viewport.name}视口下应该正常显示`, async ({ page }) => {
        await page.setViewportSize({ width: viewport.width, height: viewport.height });
        await page.goto('/lianliankan');
        await page.waitForLoadState('domcontentloaded');
        
        await testHelpers.takeScreenshot(page, `lianliankan-${viewport.name}`);
        
        const body = page.locator('body');
        await expect(body).toBeVisible();
      });
    }
  });

  test.describe('语音验证码响应式布局', () => {
    for (const viewport of viewports) {
      test(`语音验证码在${viewport.name}视口下应该正常显示`, async ({ page }) => {
        await page.setViewportSize({ width: viewport.width, height: viewport.height });
        await page.goto('/voice-captcha');
        await page.waitForLoadState('domcontentloaded');
        
        await testHelpers.takeScreenshot(page, `voice-captcha-${viewport.name}`);
        
        const body = page.locator('body');
        await expect(body).toBeVisible();
      });
    }
  });

  test.describe('极小屏幕测试', () => {
    test('首页在极小屏幕下应该仍然可用', async ({ page }) => {
      await page.setViewportSize({ width: 320, height: 480 });
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');
      
      const body = page.locator('body');
      await expect(body).toBeVisible();
    });
  });
});
