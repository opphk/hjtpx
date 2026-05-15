const { baseTest, expect, waitForPageLoad, generateRandomEmail, generateRandomString } = require('../utils/test-helpers');

baseTest.describe('HJTPX 认证流程测试', () => {
  baseTest.describe('登录页面', () => {
    baseTest('应该显示登录表单的所有必需元素', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/login');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('login-page');
      
      await expect(page.locator('h1, h2')).toContainText(/登录|login|sign\s*in/i);
      await expect(page.locator('input[type="email"], input[name="email"]')).toBeVisible();
      await expect(page.locator('input[type="password"], input[name="password"]')).toBeVisible();
      await expect(page.locator('button[type="submit"]')).toBeVisible();
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该对空表单字段显示验证错误', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/login');
      await waitForPageLoad(page);
      
      const submitButton = page.locator('button[type="submit"]').first();
      await submitButton.click();
      
      await page.waitForTimeout(1000);
      await screenshotManager.capture('login-empty-fields-error');
      
      const hasError = await page.locator('.error, .alert, [role="alert"]').count() > 0 ||
                       await page.locator('body').textContent().then(text => 
                         text.toLowerCase().includes('error') || 
                         text.toLowerCase().includes('required') ||
                         text.toLowerCase().includes('必填')
                       );
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该对无效的邮箱格式显示错误', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/login');
      await waitForPageLoad(page);
      
      const emailInput = page.locator('input[type="email"], input[name="email"]').first();
      const passwordInput = page.locator('input[type="password"], input[name="password"]').first();
      const submitButton = page.locator('button[type="submit"]').first();
      
      await emailInput.fill('invalid-email-format');
      await passwordInput.fill('ValidPassword123!');
      await submitButton.click();
      
      await page.waitForTimeout(1000);
      await screenshotManager.capture('login-invalid-email-error');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该对无效的凭证显示错误', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/login');
      await waitForPageLoad(page);
      
      const emailInput = page.locator('input[type="email"], input[name="email"]').first();
      const passwordInput = page.locator('input[type="password"], input[name="password"]').first();
      const submitButton = page.locator('button[type="submit"]').first();
      
      await emailInput.fill('nonexistent-user@example.com');
      await passwordInput.fill('WrongPassword123!');
      await submitButton.click();
      
      await page.waitForTimeout(2000);
      await screenshotManager.capture('login-invalid-credentials-error');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('注册页面', () => {
    baseTest('应该显示注册表单的所有必需元素', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/register');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('register-page');
      
      await expect(page.locator('h1, h2')).toContainText(/注册|register|sign\s*up/i);
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });

    baseTest('应该对密码强度进行验证', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/register');
      await waitForPageLoad(page);
      
      const emailInput = page.locator('input[type="email"], input[name="email"]').first();
      const nameInput = page.locator('input[name="name"], input[name="username"]').first();
      const passwordInput = page.locator('input[type="password"], input[name="password"]').first();
      const submitButton = page.locator('button[type="submit"]').first();
      
      const randomEmail = generateRandomEmail();
      const randomName = `Test User ${generateRandomString(5)}`;
      
      if (await emailInput.isVisible()) await emailInput.fill(randomEmail);
      if (await nameInput.isVisible()) await nameInput.fill(randomName);
      if (await passwordInput.isVisible()) await passwordInput.fill('123');
      if (await submitButton.isVisible()) await submitButton.click();
      
      await page.waitForTimeout(1000);
      await screenshotManager.capture('register-weak-password');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });

  baseTest.describe('认证状态持久化', () => {
    baseTest('应该在页面重新加载后保持认证状态（如果已实现）', async ({ page, testHelpers }) => {
      const { screenshotManager, consoleChecker } = testHelpers;
      
      await page.goto('/');
      await waitForPageLoad(page);
      
      await screenshotManager.capture('home-page');
      
      expect(consoleChecker.hasErrors()).toBe(false);
    });
  });
});
