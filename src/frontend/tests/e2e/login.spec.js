import { test, expect } from '@playwright/test';

test.describe('User Login Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    page.on('console', msg => {
      if (msg.type() === 'error') {
        console.log(`Console Error: ${msg.text()}`);
      }
    });
  });

  test('should login successfully with valid credentials', async ({ page }) => {
    await page.goto('/login');

    await page.fill('input[name="email"]', 'admin@example.com');
    await page.fill('input[name="password"]', 'Admin123!');
    await page.click('button[type="submit"]');

    await page.waitForURL(/\/(dashboard|home)/i, { timeout: 15000 }).catch(() => {
      console.log('Redirect may not have occurred or login failed');
    });

    const currentUrl = page.url();
    expect(currentUrl).not.toMatch(/\/login$/i);
  });

  test('should fail login with incorrect password', async ({ page }) => {
    await page.fill('input[name="email"]', 'admin@example.com');
    await page.fill('input[name="password"]', 'WrongPassword123');
    await page.click('button[type="submit"]');

    const errorVisible = await page.locator('.error, .alert, [role="alert"]').first().isVisible().catch(() => false);
    if (errorVisible) {
      await expect(page.locator('.error, .alert, [role="alert"]').first()).toBeVisible();
    } else {
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|错误|失败/i);
    }

    await expect(page.url()).toMatch(/\/login/i);
  });

  test('should fail login with non-existent email', async ({ page }) => {
    const timestamp = Date.now();
    await page.fill('input[name="email"]', `nonexistent${timestamp}@example.com`);
    await page.fill('input[name="password"]', 'SomePassword123');
    await page.click('button[type="submit"]');

    await page.waitForTimeout(1000);
    const errorVisible = await page.locator('.error, .alert, [role="alert"], .error-message').first().isVisible().catch(() => false);
    expect(errorVisible).toBeTruthy();
  });

  test.describe('Form Validation', () => {
    test('should show validation error for empty email', async ({ page }) => {
      await page.fill('input[name="password"]', 'TestPassword123');
      await page.click('button[type="submit"]');

      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/邮箱|不能为空/i);
    });

    test('should show validation error for empty password', async ({ page }) => {
      await page.fill('input[name="email"]', 'test@example.com');
      await page.click('button[type="submit"]');

      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|不能为空/i);
    });

    test('should show validation error for invalid email format', async ({ page }) => {
      await page.fill('input[name="email"]', 'invalid-email');
      await page.fill('input[name="password"]', 'TestPassword123');
      await page.click('button[type="submit"]');

      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/邮箱|有效/i);
    });

    test('should show validation error for short password', async ({ page }) => {
      await page.fill('input[name="email"]', 'test@example.com');
      await page.fill('input[name="password"]', 'short');
      await page.click('button[type="submit"]');

      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|8.*字符|至少/i);
    });

    test('should show multiple validation errors when all fields are empty', async ({ page }) => {
      await page.click('button[type="submit"]');

      await page.waitForTimeout(500);
      const errorMessages = page.locator('.error-message, [class*="error"]');
      const errorCount = await errorMessages.count();

      expect(errorCount).toBeGreaterThanOrEqual(2);
    });
  });

  test.describe('UI Elements', () => {
    test('should display all required form elements', async ({ page }) => {
      await expect(page.locator('h1, h2')).toContainText(/登录|welcome|sign in/i, { ignoreCase: true });
      await expect(page.locator('input[name="email"]')).toBeVisible();
      await expect(page.locator('input[name="password"]')).toBeVisible();
      await expect(page.locator('button[type="submit"]')).toBeVisible();
    });

    test('should have correct input types for accessibility', async ({ page }) => {
      await expect(page.locator('input[name="email"]')).toHaveAttribute('type', 'email');
      await expect(page.locator('input[name="password"]')).toHaveAttribute('type', 'password');
    });

    test('should show loading state during submission', async ({ page }) => {
      await page.fill('input[name="email"]', 'test@example.com');
      await page.fill('input[name="password"]', 'TestPassword123');

      const submitButton = page.locator('button[type="submit"]');
      await submitButton.click();

      await page.waitForTimeout(100);
      await expect(submitButton).toBeDisabled();
    });

    test('should navigate to registration page', async ({ page }) => {
      const registerLink = page.locator('a:has-text("注册"), a:has-text("register"), a[href="/register"]').first();
      if (await registerLink.isVisible().catch(() => false)) {
        await registerLink.click();
        await expect(page).toHaveURL(/\/register/i);
      }
    });
  });

  test.describe('Keyboard Navigation', () => {
    test('should support tab navigation between fields', async ({ page }) => {
      await page.locator('input[name="email"]').focus();
      await expect(page.locator('input[name="email"]')).toBeFocused();

      await page.keyboard.press('Tab');
      await expect(page.locator('input[name="password"]')).toBeFocused();

      await page.keyboard.press('Tab');
      await expect(page.locator('button[type="submit"]')).toBeFocused();
    });

    test('should submit form with Enter key', async ({ page }) => {
      await page.fill('input[name="email"]', 'test@example.com');
      await page.fill('input[name="password"]', 'TestPassword123');

      await page.keyboard.press('Enter');

      await page.waitForTimeout(100);
      const isDisabled = await page.locator('button[type="submit"]').isDisabled();
      expect(isDisabled).toBeTruthy();
    });
  });

  test.describe('Session Management', () => {
    test('should clear form after failed login', async ({ page }) => {
      await page.fill('input[name="email"]', 'test@example.com');
      await page.fill('input[name="password"]', 'WrongPassword123');
      await page.click('button[type="submit"]');

      await page.waitForTimeout(1000);
      expect(await page.locator('input[name="email"]').inputValue()).toBe('test@example.com');
      expect(await page.locator('input[name="password"]').inputValue()).toBe('');
    });

    test('should clear errors when user starts typing', async ({ page }) => {
      await page.click('button[type="submit"]');
      await page.waitForTimeout(300);

      const errorExists = await page.locator('.error-message, [class*="error"]').first().isVisible();
      if (errorExists) {
        await page.fill('input[name="email"]', 'test@example.com');
        await expect(page.locator('.error-message, [class*="error"]').first()).not.toBeVisible();
      }
    });
  });

  test.describe('Console Error Monitoring', () => {
    test('should not have console errors on page load', async ({ page }) => {
      const errors = [];
      page.on('console', msg => {
        if (msg.type() === 'error') {
          errors.push(msg.text());
        }
      });

      await page.reload();
      await page.waitForLoadState('networkidle');

      const criticalErrors = errors.filter(err =>
        !err.includes('favicon') &&
        !err.includes('DevTools') &&
        !err.includes('third-party')
      );

      expect(criticalErrors.length).toBe(0);
    });
  });
});
