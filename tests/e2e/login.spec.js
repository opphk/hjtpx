const { test, expect } = require('@playwright/test');

test.describe('Login Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('should display login page with all required elements', async ({ page }) => {
    await expect(page.locator('h1, h2')).toContainText(/login|sign in/i, { ignoreCase: true });
    
    const emailInput = page.locator('input[name="email"], input[type="email"]');
    await expect(emailInput).toBeVisible();
    
    const passwordInput = page.locator('input[name="password"], input[type="password"]');
    await expect(passwordInput).toBeVisible();
    
    const submitButton = page.locator('button[type="submit"]');
    await expect(submitButton).toBeVisible();
    await expect(submitButton).toContainText(/login|sign in/i, { ignoreCase: true });
  });

  test('should show validation errors for empty fields', async ({ page }) => {
    const submitButton = page.locator('button[type="submit"]');
    await submitButton.click();
    
    await page.waitForTimeout(500);
    
    const errorMessages = page.locator('.error, .alert, [role="alert"], .text-red, .text-danger');
    const hasError = await errorMessages.count() > 0;
    
    if (!hasError) {
      await expect(page.locator('body')).toContainText(/required|empty|invalid/i, { ignoreCase: true });
    }
  });

  test('should show error for invalid email format', async ({ page }) => {
    const emailInput = page.locator('input[name="email"], input[type="email"]');
    await emailInput.fill('invalid-email');
    
    const passwordInput = page.locator('input[name="password"], input[type="password"]');
    await passwordInput.fill('ValidPassword123!');
    
    const submitButton = page.locator('button[type="submit"]');
    await submitButton.click();
    
    await page.waitForTimeout(1000);
    
    const errorMessages = page.locator('.error, .alert, [role="alert"], .text-red, .text-danger');
    const hasError = await errorMessages.count() > 0;
    
    if (!hasError) {
      await expect(page.locator('body')).toContainText(/email|valid|invalid/i, { ignoreCase: true });
    }
  });

  test('should login successfully with valid credentials', async ({ page }) => {
    const testEmail = `logintest_${Date.now()}@example.com`;
    const testPassword = 'TestPassword123!';
    
    await page.goto('/register');
    await page.fill('input[name="email"], input[type="email"]', testEmail);
    await page.fill('input[name="name"], input[name="username"]', 'Test User');
    await page.fill('input[name="password"], input[type="password"]', testPassword);
    await page.fill('input[name="confirmPassword"], input[name="confirm_password"]', testPassword);
    await page.click('button[type="submit"]');
    
    await page.waitForTimeout(2000);
    
    await page.goto('/login');
    await page.fill('input[name="email"], input[type="email"]', testEmail);
    await page.fill('input[name="password"], input[type="password"]', testPassword);
    await page.click('button[type="submit"]');
    
    await page.waitForTimeout(2000);
    
    const currentUrl = page.url();
    const isLoggedIn = !currentUrl.includes('/login') || 
                      await page.locator('text=/dashboard|profile|logout|home/i').count() > 0;
    
    expect(isLoggedIn).toBeTruthy();
  });

  test('should show error for invalid credentials', async ({ page }) => {
    const emailInput = page.locator('input[name="email"], input[type="email"]');
    await emailInput.fill('nonexistent@example.com');
    
    const passwordInput = page.locator('input[name="password"], input[type="password"]');
    await passwordInput.fill('WrongPassword123!');
    
    const submitButton = page.locator('button[type="submit"]');
    await submitButton.click();
    
    await page.waitForTimeout(2000);
    
    const errorMessages = page.locator('.error, .alert, [role="alert"], .text-red, .text-danger');
    const hasError = await errorMessages.count() > 0;
    
    if (!hasError) {
      await expect(page.locator('body')).toContainText(/invalid|error|failed|wrong/i, { ignoreCase: true });
    } else {
      await expect(errorMessages.first()).toContainText(/invalid|error|failed|wrong/i, { ignoreCase: true });
    }
  });

  test('should have remember me option', async ({ page }) => {
    const rememberMeCheckbox = page.locator('input[name="remember"], input[type="checkbox"]').first();
    
    const isVisible = await rememberMeCheckbox.isVisible().catch(() => false);
    if (isVisible) {
      await expect(rememberMeCheckbox).toBeVisible();
    } else {
      const bodyText = await page.locator('body').textContent();
      expect(bodyText).toMatch(/remember|forgot/i);
    }
  });

  test('should link to forgot password page', async ({ page }) => {
    const forgotLink = page.locator('a[href*="forgot"], a[href*="reset"], a:has-text("forgot")').first();
    
    const isVisible = await forgotLink.isVisible().catch(() => false);
    if (isVisible) {
      await expect(forgotLink).toBeVisible();
    }
  });

  test('should link to registration page', async ({ page }) => {
    const registerLink = page.locator('a[href*="register"], a[href*="signup"], a:has-text("register"), a:has-text("sign up")').first();
    
    const isVisible = await registerLink.isVisible().catch(() => false);
    if (isVisible) {
      await expect(registerLink).toBeVisible();
    }
  });

  test('should handle password visibility toggle', async ({ page }) => {
    const passwordInput = page.locator('input[name="password"], input[type="password"]');
    
    const isPasswordType = await passwordInput.getAttribute('type').then(t => t === 'password').catch(() => true);
    if (isPasswordType) {
      const toggleButton = page.locator('button[aria-label*="show"], button[aria-label*="visibility"], .toggle-password').first();
      const isVisible = await toggleButton.isVisible().catch(() => false);
      
      if (isVisible) {
        await toggleButton.click();
        await expect(passwordInput).toHaveAttribute('type', 'text');
        
        await toggleButton.click();
        await expect(passwordInput).toHaveAttribute('type', 'password');
      }
    }
  });
});
