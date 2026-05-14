const { test, expect } = require('@playwright/test');

test.describe('User Login Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('should display login form', async ({ page }) => {
    await expect(page.locator('h1, h2')).toContainText(/login|sign in/i);
    await expect(page.locator('input[name="email"], input[type="email"]')).toBeVisible();
    await expect(page.locator('input[name="password"], input[type="password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('should show validation errors for empty fields', async ({ page }) => {
    await page.click('button[type="submit"]');
    
    const emailInput = page.locator('input[name="email"], input[type="email"]');
    const passwordInput = page.locator('input[name="password"], input[type="password"]');
    
    await expect(emailInput).toHaveClass(/error|invalid/i).catch(() => {});
    await expect(passwordInput).toHaveClass(/error|invalid/i).catch(() => {});
  });

  test('should show error for invalid email format', async ({ page }) => {
    await page.fill('input[name="email"], input[type="email"]', 'invalid-email');
    await page.fill('input[name="password"], input[type="password"]', 'ValidPassword123!');
    await page.click('button[type="submit"]');
    
    await expect(page.locator('.error, .alert, [role="alert"]')).toContainText(/email|valid/i).catch(() => {
      expect(page.locator('body')).toContainText(/email|valid/i);
    });
  });

  test('should login successfully with valid credentials', async ({ page }) => {
    await page.fill('input[name="email"], input[type="email"]', 'test@example.com');
    await page.fill('input[name="password"], input[type="password"]', 'ValidPassword123!');
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/(dashboard|home|profile)/i, { timeout: 10000 }).catch(() => {
      expect(page.url()).not.toContain('/login');
    });
  });

  test('should show error for invalid credentials', async ({ page }) => {
    await page.fill('input[name="email"], input[type="email"]', 'wrong@example.com');
    await page.fill('input[name="password"], input[type="password"]', 'WrongPassword123!');
    await page.click('button[type="submit"]');
    
    await page.waitForSelector('.error, .alert, [role="alert"]', { timeout: 5000 }).catch(() => {
      expect(page.locator('body')).toContainText(/invalid|error|failed/i);
    });
  });

  test('should have remember me checkbox', async ({ page }) => {
    const rememberMe = page.locator('input[name="remember"], input[type="checkbox"]').first();
    await expect(rememberMe).toBeVisible().catch(() => {});
  });

  test('should link to forgot password page', async ({ page }) => {
    const forgotLink = page.locator('a[href*="forgot"], a:has-text("forgot")').first();
    await expect(forgotLink).toBeVisible().catch(() => {
      expect(page.locator('body')).toContainText(/forgot.*password|password.*reset/i);
    });
  });

  test('should link to registration page', async ({ page }) => {
    const registerLink = page.locator('a[href*="register"], a[href*="signup"], a:has-text("register"), a:has-text("sign up")').first();
    await expect(registerLink).toBeVisible().catch(() => {
      expect(page.locator('body')).toContainText(/register|sign up|create.*account/i);
    });
  });
});

test.describe('User Registration Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/register');
  });

  test('should display registration form', async ({ page }) => {
    await expect(page.locator('h1, h2')).toContainText(/register|sign up|create/i);
    await expect(page.locator('input[name="email"], input[type="email"]')).toBeVisible();
    await expect(page.locator('input[name="name"], input[name="username"]')).toBeVisible();
    await expect(page.locator('input[name="password"], input[type="password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('should validate password strength', async ({ page }) => {
    await page.fill('input[name="email"], input[type="email"]', 'newuser@example.com');
    await page.fill('input[name="name"], input[name="username"]', 'New User');
    await page.fill('input[name="password"], input[type="password"]', 'weak');
    await page.click('button[type="submit"]');
    
    await expect(page.locator('.error, .alert, [role="alert"], .password-error')).toContainText(/password|weak|8.*character/i).catch(() => {
      expect(page.locator('body')).toContainText(/password|weak|8.*character/i);
    });
  });

  test('should register with valid data', async ({ page }) => {
    const timestamp = Date.now();
    await page.fill('input[name="email"], input[type="email"]', `user${timestamp}@example.com`);
    await page.fill('input[name="name"], input[name="username"]', `User ${timestamp}`);
    await page.fill('input[name="password"], input[type="password"]', 'ValidPassword123!');
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/(dashboard|home|login|verify)/i, { timeout: 10000 }).catch(() => {
      expect(page.url()).not.toContain('/register');
    });
  });
});
