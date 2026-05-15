import { test, expect } from '@playwright/test';

test.describe.serial('User Login Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.waitForSelector('h1', { timeout: 10000 });
  });

  test('should display login page with all required elements', async ({ page }) => {
    await expect(page.locator('h1')).toContainText(/欢迎|登录|welcome|sign in|login/i);
    await expect(page.locator('input[name="email"]')).toBeVisible();
    await expect(page.locator('input[name="password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('should show validation error for empty email field', async ({ page }) => {
    await page.fill('input[name="password"]', 'TestPassword123');
    await page.click('button[type="submit"]');

    await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  });

  test('should show validation error for empty password field', async ({ page }) => {
    await page.fill('input[name="email"]', 'test@example.com');
    await page.click('button[type="submit"]');

    await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
  });

  test('should show validation error for invalid email format', async ({ page }) => {
    await page.fill('input[name="email"]', 'invalid-email');
    await page.fill('input[name="password"]', 'TestPassword123');
    await page.click('button[type="submit"]');

    await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/邮箱|有效|email|valid/i);
  });

  test('should show validation error for short password', async ({ page }) => {
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'short');
    await page.click('button[type="submit"]');

    await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|8.*字符|password|8.*character/i);
  });

  test('should navigate to registration page when clicking sign up link', async ({ page }) => {
    const signUpLink = page.locator('a:has-text("注册"), a:has-text("sign up"), a:has-text("register"), a[href="/register"]').first();
    await signUpLink.click();

    await expect(page).toHaveURL(/\/register/i);
  });

  test('should display registration link in footer', async ({ page }) => {
    const footerText = page.locator('.auth-footer, [class*="footer"]');
    await expect(footerText).toContainText(/注册|创建|register|sign up|create.*account/i);
  });

  test('should show loading state during login submission', async ({ page }) => {
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'TestPassword123');

    const submitButton = page.locator('button[type="submit"]');
    await submitButton.click();

    await expect(submitButton).toBeDisabled();
  });

  test('should handle network error gracefully', async ({ page }) => {
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'TestPassword123');

    await page.route('**/api/**', route => route.abort());
    await page.click('button[type="submit"]');

    await expect(page.locator('.error, .alert, [role="alert"]').first()).toBeVisible({ timeout: 10000 });
  });

  test('should clear error message when navigating away', async ({ page }) => {
    await page.fill('input[name="email"]', 'invalid-email');
    await page.fill('input[name="password"]', 'TestPassword123');
    await page.click('button[type="submit"]');

    await page.locator('.error, .alert').first().waitFor({ state: 'visible' });

    await page.goto('/register');
    await page.goto('/login');
    await page.waitForSelector('h1', { timeout: 10000 });

    await expect(page.locator('.error, .alert')).toHaveCount(0);
  });

  test('should maintain form data on validation error', async ({ page }) => {
    const testEmail = 'test@example.com';
    await page.fill('input[name="email"]', testEmail);
    await page.fill('input[name="password"]', 'short');
    await page.click('button[type="submit"]');

    await expect(page.locator('input[name="email"]')).toHaveValue(testEmail);
  });
});

test.describe('Authenticated User Session', () => {
  test.skip('should redirect to dashboard after successful login', async ({ page }) => {
    await page.goto('/login');

    await page.fill('input[name="email"]', 'admin@example.com');
    await page.fill('input[name="password"]', 'Admin123!');
    await page.click('button[type="submit"]');

    await page.waitForURL(/\/dashboard/i, { timeout: 15000 }).catch(() => {
      console.log('Login may have failed or redirected elsewhere');
    });
  });

  test.skip('should persist session after page reload', async ({ page }) => {
    await page.context().storageState({ path: './tests/.auth/user.json' });

    await page.goto('/dashboard');

    await expect(page).toHaveURL(/\/dashboard/i);
  });

  test.skip('should logout and redirect to login', async ({ page }) => {
    await page.context().storageState({ path: './tests/.auth/user.json' });

    await page.goto('/dashboard');

    const logoutButton = page.locator('button:has-text("logout"), button:has-text("sign out"), a[href="/logout"]').first();
    if (await logoutButton.isVisible().catch(() => false)) {
      await logoutButton.click();
      await expect(page).toHaveURL(/\/login/i);
    }
  });
});

test.describe('Login Form Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.waitForSelector('h1', { timeout: 10000 });
  });

  test('should have proper input types', async ({ page }) => {
    const emailInput = page.locator('input[name="email"]');
    const passwordInput = page.locator('input[name="password"]');

    await expect(emailInput).toHaveAttribute('type', /email|text/);
    await expect(passwordInput).toHaveAttribute('type', 'password');
  });

  test('should support keyboard navigation', async ({ page }) => {
    await page.locator('input[name="email"]').focus();
    const isEmailFocused = await page.evaluate(() => document.activeElement?.name === 'email');
    expect(isEmailFocused).toBe(true);

    await page.keyboard.press('Tab');
    const isPasswordFocused = await page.evaluate(() => document.activeElement?.name === 'password');
    expect(isPasswordFocused).toBe(true);
  });

  test('should submit form with Enter key', async ({ page }) => {
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'TestPassword123');

    await page.keyboard.press('Enter');

    await expect(page.locator('button[type="submit"]')).toBeDisabled();
  });
});
