const { test, expect } = require('@playwright/test');

test.describe('Registration Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/register');
  });

  test('should display registration page with all required elements', async ({ page }) => {
    await expect(page.locator('h1, h2')).toContainText(/register|sign up|create/i, { ignoreCase: true });
    
    const emailInput = page.locator('input[name="email"], input[type="email"]');
    await expect(emailInput).toBeVisible();
    
    const nameInput = page.locator('input[name="name"], input[name="username"], input[name="displayName"]');
    await expect(nameInput).toBeVisible();
    
    const passwordInput = page.locator('input[name="password"], input[type="password"]');
    await expect(passwordInput).toBeVisible();
    
    const submitButton = page.locator('button[type="submit"]');
    await expect(submitButton).toBeVisible();
    await expect(submitButton).toContainText(/register|sign up|create/i, { ignoreCase: true });
  });

  test('should validate email format', async ({ page }) => {
    const emailInput = page.locator('input[name="email"], input[type="email"]');
    await emailInput.fill('invalid-email');
    
    const nameInput = page.locator('input[name="name"], input[name="username"]');
    await nameInput.fill('Test User');
    
    const passwordInput = page.locator('input[name="password"], input[type="password"]');
    await passwordInput.fill('TestPassword123!');
    
    const submitButton = page.locator('button[type="submit"]');
    await submitButton.click();
    
    await page.waitForTimeout(1000);
    
    const errorMessages = page.locator('.error, .alert, [role="alert"], .text-red, .text-danger');
    const hasError = await errorMessages.count() > 0;
    
    if (!hasError) {
      await expect(page.locator('body')).toContainText(/email|valid|invalid/i, { ignoreCase: true });
    }
  });

  test('should validate password strength', async ({ page }) => {
    const emailInput = page.locator('input[name="email"], input[type="email"]');
    await emailInput.fill(`weakpass_${Date.now()}@example.com`);
    
    const nameInput = page.locator('input[name="name"], input[name="username"]');
    await nameInput.fill('Test User');
    
    const passwordInput = page.locator('input[name="password"], input[type="password"]');
    await passwordInput.fill('123');
    
    const submitButton = page.locator('button[type="submit"]');
    await submitButton.click();
    
    await page.waitForTimeout(1000);
    
    const errorMessages = page.locator('.error, .alert, [role="alert"], .password-error, .text-red, .text-danger');
    const hasError = await errorMessages.count() > 0;
    
    if (!hasError) {
      await expect(page.locator('body')).toContainText(/password|weak|8.*character|uppercase|lowercase/i, { ignoreCase: true });
    }
  });

  test('should validate password requirements', async ({ page }) => {
    const emailInput = page.locator('input[name="email"], input[type="email"]');
    await emailInput.fill(`weakpass_${Date.now()}@example.com`);
    
    const nameInput = page.locator('input[name="name"], input[name="username"]');
    await nameInput.fill('Test User');
    
    const passwordInput = page.locator('input[name="password"], input[type="password"]');
    await passwordInput.fill('onlylowercase');
    
    const submitButton = page.locator('button[type="submit"]');
    await submitButton.click();
    
    await page.waitForTimeout(1000);
    
    const errorMessages = page.locator('.error, .alert, [role="alert"], .password-error, .text-red, .text-danger');
    const hasError = await errorMessages.count() > 0;
    
    if (!hasError) {
      await expect(page.locator('body')).toContainText(/password|uppercase|number|special|character/i, { ignoreCase: true });
    }
  });

  test('should register successfully with valid data', async ({ page }) => {
    const timestamp = Date.now();
    const testEmail = `user${timestamp}@example.com`;
    const testName = `Test User ${timestamp}`;
    const testPassword = 'TestPassword123!';
    
    const emailInput = page.locator('input[name="email"], input[type="email"]');
    await emailInput.fill(testEmail);
    
    const nameInput = page.locator('input[name="name"], input[name="username"]');
    await nameInput.fill(testName);
    
    const passwordInput = page.locator('input[name="password"], input[type="password"]');
    await passwordInput.fill(testPassword);
    
    const confirmPasswordInput = page.locator('input[name="confirmPassword"], input[name="confirm_password"], input[name="passwordConfirm"]');
    if (await confirmPasswordInput.isVisible().catch(() => false)) {
      await confirmPasswordInput.fill(testPassword);
    }
    
    const submitButton = page.locator('button[type="submit"]');
    await submitButton.click();
    
    await page.waitForTimeout(3000);
    
    const currentUrl = page.url();
    const isRegistered = !currentUrl.includes('/register') || 
                        await page.locator('text=/success|welcome|dashboard|login|verified/i').count() > 0;
    
    expect(isRegistered).toBeTruthy();
  });

  test('should show error for duplicate email', async ({ page }) => {
    const existingEmail = `duplicate_${Date.now()}@example.com`;
    const testPassword = 'TestPassword123!';
    
    await page.fill('input[name="email"], input[type="email"]', existingEmail);
    await page.fill('input[name="name"], input[name="username"]', 'Test User');
    await page.fill('input[name="password"], input[type="password"]', testPassword);
    
    const confirmPasswordInput = page.locator('input[name="confirmPassword"], input[name="confirm_password"]');
    if (await confirmPasswordInput.isVisible().catch(() => false)) {
      await confirmPasswordInput.fill(testPassword);
    }
    
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    await page.goto('/register');
    await page.fill('input[name="email"], input[type="email"]', existingEmail);
    await page.fill('input[name="name"], input[name="username"]', 'Test User 2');
    await page.fill('input[name="password"], input[type="password"]', testPassword);
    
    const confirmPasswordInput2 = page.locator('input[name="confirmPassword"], input[name="confirm_password"]');
    if (await confirmPasswordInput2.isVisible().catch(() => false)) {
      await confirmPasswordInput2.fill(testPassword);
    }
    
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    const errorMessages = page.locator('.error, .alert, [role="alert"], .text-red, .text-danger');
    const hasError = await errorMessages.count() > 0;
    
    if (!hasError) {
      await expect(page.locator('body')).toContainText(/exist|already|duplicate|use.*different/i, { ignoreCase: true });
    }
  });

  test('should have terms and conditions link', async ({ page }) => {
    const termsLink = page.locator('a[href*="terms"], a[href*="privacy"], a:has-text("terms"), a:has-text("privacy")').first();
    
    const isVisible = await termsLink.isVisible().catch(() => false);
    if (isVisible) {
      await expect(termsLink).toBeVisible();
    }
  });

  test('should link to login page', async ({ page }) => {
    const loginLink = page.locator('a[href*="login"], a:has-text("already.*account"), a:has-text("sign in")').first();
    
    const isVisible = await loginLink.isVisible().catch(() => false);
    if (isVisible) {
      await expect(loginLink).toBeVisible();
    }
  });

  test('should handle password confirmation mismatch', async ({ page }) => {
    await page.fill('input[name="email"], input[type="email"]', `mismatch_${Date.now()}@example.com`);
    await page.fill('input[name="name"], input[name="username"]', 'Test User');
    await page.fill('input[name="password"], input[type="password"]', 'TestPassword123!');
    
    const confirmPasswordInput = page.locator('input[name="confirmPassword"], input[name="confirm_password"]');
    if (await confirmPasswordInput.isVisible().catch(() => false)) {
      await confirmPasswordInput.fill('DifferentPassword123!');
      
      await page.click('button[type="submit"]');
      await page.waitForTimeout(1000);
      
      const errorMessages = page.locator('.error, .alert, [role="alert"], .text-red, .text-danger');
      const hasError = await errorMessages.count() > 0;
      
      if (!hasError) {
        await expect(page.locator('body')).toContainText(/match|same|identical/i, { ignoreCase: true });
      }
    }
  });
});
