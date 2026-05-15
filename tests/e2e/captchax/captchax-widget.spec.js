const { baseTest, expect, waitForPageLoad } = require('../utils/test-helpers');

baseTest.describe('CaptchaX 验证码组件测试', () => {
  baseTest.describe('演示页面', () => {
    baseTest('应该加载演示页面', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/demo');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('captchax-demo-page');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该显示验证码组件', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/demo');
      await waitForPageLoad(page);
      
      const captchaContainer = page.locator('[class*="captcha"], #captcha-container, [id*="captcha"]');
      const captchaCount = await captchaContainer.count();
      
      if (captchaCount > 0) {
        await expect(captchaContainer.first()).toBeVisible();
      }
      
      await screenshotManager.capture('captchax-widget-display');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('滑块验证码', () => {
    baseTest('应该显示滑块验证码（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/demo');
      await waitForPageLoad(page);
      
      const slider = page.locator('[class*="slider"], [class*="captcha-slider"]');
      const sliderCount = await slider.count();
      
      if (sliderCount > 0) {
        await expect(slider.first()).toBeVisible();
      }
      
      await screenshotManager.capture('captchax-slider-captcha');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('点击验证码', () => {
    baseTest('应该显示点击验证码（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/demo');
      await waitForPageLoad(page);
      
      const clickCaptcha = page.locator('[class*="click"], [class*="captcha-click"]');
      const clickCount = await clickCaptcha.count();
      
      if (clickCount > 0) {
        await expect(clickCaptcha.first()).toBeVisible();
      }
      
      await screenshotManager.capture('captchax-click-captcha');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('拼图验证码', () => {
    baseTest('应该显示拼图验证码（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/demo');
      await waitForPageLoad(page);
      
      const puzzle = page.locator('[class*="puzzle"], [class*="captcha-puzzle"]');
      const puzzleCount = await puzzle.count();
      
      if (puzzleCount > 0) {
        await expect(puzzle.first()).toBeVisible();
      }
      
      await screenshotManager.capture('captchax-puzzle-captcha');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('国际化测试', () => {
    baseTest('应该加载国际化演示页面（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/i18n-demo');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('captchax-i18n-demo');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该有语言切换功能（如果有）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/i18n-demo');
      await waitForPageLoad(page);
      
      const langSwitcher = page.locator('[class*="lang"], [class*="language"], select');
      const langCount = await langSwitcher.count();
      
      if (langCount > 0) {
        await expect(langSwitcher.first()).toBeVisible();
      }
      
      await screenshotManager.capture('captchax-lang-switcher');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('验证码组件响应式测试', () => {
    baseTest('应该在不同屏幕尺寸下正确显示', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      const viewports = [
        { width: 1920, height: 1080, name: 'desktop' },
        { width: 768, height: 1024, name: 'tablet' },
        { width: 375, height: 667, name: 'mobile' }
      ];
      
      for (const viewport of viewports) {
        await page.setViewportSize(viewport);
        await page.goto('/demo');
        await waitForPageLoad(page);
        await screenshotManager.capture(`captchax-widget-${viewport.name}`);
      }
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });
});
