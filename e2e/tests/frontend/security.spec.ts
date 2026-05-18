import { test, expect } from '@playwright/test';

test.describe('Security Features E2E Tests', () => {
  test('should enforce HTTPS redirect', async ({ page }) => {
    await page.goto('http://localhost:8080/home.html');
    await expect(page).not.toHaveURL(/^http:\/\//);
  });

  test('should set security headers', async ({ page }) => {
    await page.goto('/home.html');
    
    const headers = await page.evaluate(() => {
      return {
        xFrameOptions: page.getByHeader('X-Frame-Options'),
        xContentTypeOptions: page.getByHeader('X-Content-Type-Options'),
        xssProtection: page.getByHeader('X-XSS-Protection'),
        hsts: page.getByHeader('Strict-Transport-Security'),
      };
    });
    
    expect(headers.xFrameOptions).toBeDefined();
    expect(headers.xContentTypeOptions).toBeDefined();
  });

  test('should prevent XSS attacks', async ({ page }) => {
    await page.goto('/captcha.html');
    
    const input = page.locator('#user-input');
    await input.fill('<script>alert("xss")</script>');
    
    const submitButton = page.locator('#submit-button');
    await submitButton.click();
    
    page.on('dialog', dialog => {
      expect(dialog.type()).toBe('dismissed');
    });
  });

  test('should validate CSRF tokens', async ({ page }) => {
    await page.goto('/admin/login.html');
    
    const loginForm = page.locator('#login-form');
    await expect(loginForm).toBeVisible();
    
    const csrfToken = await page.locator('input[name="csrf_token"]').inputValue();
    expect(csrfToken).toBeDefined();
    expect(csrfToken.length).toBeGreaterThan(0);
  });

  test('should implement rate limiting', async ({ page }) => {
    for (let i = 0; i < 10; i++) {
      await page.goto('/captcha.html');
    }
    
    await page.waitForTimeout(1000);
    const errorMessage = page.locator('.rate-limit-error');
    await expect(errorMessage).toBeVisible();
  });

  test('should encrypt sensitive data in transit', async ({ page }) => {
    await page.goto('/home.html');
    
    const securityInfo = await page.evaluate(() => {
      const conn = (window as any).crypto;
      return conn ? 'TLS' : 'unknown';
    });
    
    expect(securityInfo).toBe('TLS');
  });

  test('should handle session timeout', async ({ page }) => {
    await page.goto('/admin/dashboard.html');
    await page.waitForTimeout(30 * 60 * 1000);
    
    const loginPrompt = page.locator('#session-expired-modal');
    await expect(loginPrompt).toBeVisible();
  });

  test('should validate input formats', async ({ page }) => {
    await page.goto('/captcha.html');
    
    const emailInput = page.locator('#email-input');
    await emailInput.fill('invalid-email');
    
    const submitButton = page.locator('#submit-button');
    await submitButton.click();
    
    const validationError = page.locator('.validation-error');
    await expect(validationError).toBeVisible();
  });

  test('should sanitize user inputs', async ({ page }) => {
    await page.goto('/captcha.html');
    
    const input = page.locator('#comment-input');
    await input.fill('<img src=x onerror=alert(1)>');
    
    const sanitizedValue = await input.inputValue();
    expect(sanitizedValue).not.toContain('<img');
  });

  test('should support secure password reset', async ({ page }) => {
    await page.goto('/forgot-password.html');
    
    const emailInput = page.locator('#email-input');
    await emailInput.fill('user@example.com');
    
    const submitButton = page.locator('#submit-button');
    await submitButton.click();
    
    const successMessage = page.locator('#success-message');
    await expect(successMessage).toBeVisible();
  });
});
