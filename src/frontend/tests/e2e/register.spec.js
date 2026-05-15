import { test, expect } from '@playwright/test';

test.describe('User Registration Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/register');
    page.on('console', msg => {
      if (msg.type() === 'error') {
        console.log(`Console Error: ${msg.text()}`);
      }
    });
  });

  test('should register successfully with valid data', async ({ page }) => {
    const timestamp = Date.now();
    const uniqueEmail = `testuser${timestamp}@example.com`;
    
    await page.fill('input[name="name"]', `Test User ${timestamp}`);
    await page.fill('input[name="email"]', uniqueEmail);
    await page.fill('input[name="password"]', 'ValidPassword123');
    await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
    await page.click('button[type="submit"]');
    
    await page.waitForURL(/\/(login|dashboard|verify|success)/i, { timeout: 15000 }).catch(() => {
      console.log('Registration redirect may not have occurred');
    });
    
    const currentUrl = page.url();
    expect(currentUrl).not.toMatch(/\/register$/i);
  });

  test('should fail registration with duplicate email', async ({ page }) => {
    await page.fill('input[name="name"]', 'Duplicate User');
    await page.fill('input[name="email"]', 'admin@example.com');
    await page.fill('input[name="password"]', 'ValidPassword123');
    await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
    await page.click('button[type="submit"]');
    
    await page.waitForTimeout(2000);
    
    const errorVisible = await page.locator('.error, .alert, [role="alert"], .error-message').first().isVisible().catch(() => false);
    expect(errorVisible).toBeTruthy();
  });

  test.describe('Form Validation', () => {
    test('should show validation error for empty name field', async ({ page }) => {
      await page.fill('input[name="email"]', 'newuser@example.com');
      await page.fill('input[name="password"]', 'ValidPassword123');
      await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
      await page.click('button[type="submit"]');
      
      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/用户|不能为空/i);
    });

    test('should show validation error for short name', async ({ page }) => {
      await page.fill('input[name="name"]', 'a');
      await page.fill('input[name="email"]', 'newuser@example.com');
      await page.fill('input[name="password"]', 'ValidPassword123');
      await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
      await page.click('button[type="submit"]');
      
      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/用户|2.*字符|至少/i);
    });

    test('should show validation error for empty email field', async ({ page }) => {
      await page.fill('input[name="name"]', 'New User');
      await page.fill('input[name="password"]', 'ValidPassword123');
      await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
      await page.click('button[type="submit"]');
      
      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/邮箱|不能为空/i);
    });

    test('should show validation error for invalid email format', async ({ page }) => {
      await page.fill('input[name="name"]', 'New User');
      await page.fill('input[name="email"]', 'invalid-email');
      await page.fill('input[name="password"]', 'ValidPassword123');
      await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
      await page.click('button[type="submit"]');
      
      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/邮箱|有效/i);
    });

    test('should show validation error for empty password field', async ({ page }) => {
      await page.fill('input[name="name"]', 'New User');
      await page.fill('input[name="email"]', 'newuser@example.com');
      await page.click('button[type="submit"]');
      
      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|不能为空/i);
    });

    test('should show validation error for short password', async ({ page }) => {
      await page.fill('input[name="name"]', 'New User');
      await page.fill('input[name="email"]', 'newuser@example.com');
      await page.fill('input[name="password"]', 'short');
      await page.fill('input[name="confirmPassword"]', 'short');
      await page.click('button[type="submit"]');
      
      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|8.*字符|至少/i);
    });

    test('should show validation error for password without uppercase', async ({ page }) => {
      await page.fill('input[name="name"]', 'New User');
      await page.fill('input[name="email"]', 'newuser@example.com');
      await page.fill('input[name="password"]', 'validpassword123');
      await page.fill('input[name="confirmPassword"]', 'validpassword123');
      await page.click('button[type="submit"]');
      
      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|大写|uppercase/i);
    });

    test('should show validation error for password without lowercase', async ({ page }) => {
      await page.fill('input[name="name"]', 'New User');
      await page.fill('input[name="email"]', 'newuser@example.com');
      await page.fill('input[name="password"]', 'VALIDPASSWORD123');
      await page.fill('input[name="confirmPassword"]', 'VALIDPASSWORD123');
      await page.click('button[type="submit"]');
      
      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|小写|lowercase/i);
    });

    test('should show validation error for password without number', async ({ page }) => {
      await page.fill('input[name="name"]', 'New User');
      await page.fill('input[name="email"]', 'newuser@example.com');
      await page.fill('input[name="password"]', 'ValidPassword');
      await page.fill('input[name="confirmPassword"]', 'ValidPassword');
      await page.click('button[type="submit"]');
      
      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|数字|number/i);
    });

    test('should show validation error for password mismatch', async ({ page }) => {
      await page.fill('input[name="name"]', 'New User');
      await page.fill('input[name="email"]', 'newuser@example.com');
      await page.fill('input[name="password"]', 'ValidPassword123');
      await page.fill('input[name="confirmPassword"]', 'DifferentPassword123');
      await page.click('button[type="submit"]');
      
      await expect(page.locator('.error-message, [class*="error"]').first()).toBeVisible();
      await expect(page.locator('.error-message, [class*="error"]').first()).toContainText(/密码|不一致|匹配/i);
    });

    test('should show multiple validation errors when all fields are empty', async ({ page }) => {
      await page.click('button[type="submit"]');
      
      await page.waitForTimeout(500);
      const errorMessages = page.locator('.error-message, [class*="error"]');
      const errorCount = await errorMessages.count();
      
      expect(errorCount).toBeGreaterThanOrEqual(3);
    });
  });

  test.describe('UI Elements', () => {
    test('should display all required form elements', async ({ page }) => {
      await expect(page.locator('h1, h2')).toContainText(/注册|register|sign up/i, { ignoreCase: true });
      await expect(page.locator('input[name="name"]')).toBeVisible();
      await expect(page.locator('input[name="email"]')).toBeVisible();
      await expect(page.locator('input[name="password"]')).toBeVisible();
      await expect(page.locator('input[name="confirmPassword"]')).toBeVisible();
      await expect(page.locator('button[type="submit"]')).toBeVisible();
    });

    test('should have correct input types for accessibility', async ({ page }) => {
      await expect(page.locator('input[name="email"]')).toHaveAttribute('type', 'email');
      await expect(page.locator('input[name="password"]')).toHaveAttribute('type', 'password');
      await expect(page.locator('input[name="confirmPassword"]')).toHaveAttribute('type', 'password');
    });

    test('should show loading state during submission', async ({ page }) => {
      await page.fill('input[name="name"]', 'New User');
      await page.fill('input[name="email"]', 'newuser@example.com');
      await page.fill('input[name="password"]', 'ValidPassword123');
      await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
      
      const submitButton = page.locator('button[type="submit"]');
      await submitButton.click();
      
      await page.waitForTimeout(100);
      await expect(submitButton).toBeDisabled();
    });

    test('should navigate to login page', async ({ page }) => {
      const loginLink = page.locator('a:has-text("登录"), a:has-text("login"), a[href="/login"]').first();
      if (await loginLink.isVisible().catch(() => false)) {
        await loginLink.click();
        await expect(page).toHaveURL(/\/login/i);
      }
    });
  });

  test.describe('Keyboard Navigation', () => {
    test('should support tab navigation between fields', async ({ page }) => {
      await page.locator('input[name="name"]').focus();
      await expect(page.locator('input[name="name"]')).toBeFocused();
      
      await page.keyboard.press('Tab');
      await expect(page.locator('input[name="email"]')).toBeFocused();
      
      await page.keyboard.press('Tab');
      await expect(page.locator('input[name="password"]')).toBeFocused();
      
      await page.keyboard.press('Tab');
      await expect(page.locator('input[name="confirmPassword"]')).toBeFocused();
      
      await page.keyboard.press('Tab');
      await expect(page.locator('button[type="submit"]')).toBeFocused();
    });

    test('should submit form with Enter key', async ({ page }) => {
      await page.fill('input[name="name"]', 'New User');
      await page.fill('input[name="email"]', 'newuser@example.com');
      await page.fill('input[name="password"]', 'ValidPassword123');
      await page.fill('input[name="confirmPassword"]', 'ValidPassword123');
      
      await page.keyboard.press('Enter');
      
      await page.waitForTimeout(100);
      const isDisabled = await page.locator('button[type="submit"]').isDisabled();
      expect(isDisabled).toBeTruthy();
    });
  });

  test.describe('Form Data Preservation', () => {
    test('should preserve name and email on validation error', async ({ page }) => {
      const testName = 'Valid User';
      const testEmail = 'valid@example.com';
      
      await page.fill('input[name="name"]', testName);
      await page.fill('input[name="email"]', testEmail);
      await page.fill('input[name="password"]', 'short');
      await page.fill('input[name="confirmPassword"]', 'short');
      await page.click('button[type="submit"]');
      
      await page.waitForTimeout(300);
      
      expect(await page.locator('input[name="name"]').inputValue()).toBe(testName);
      expect(await page.locator('input[name="email"]').inputValue()).toBe(testEmail);
    });

    test('should clear password fields on validation error', async ({ page }) => {
      await page.fill('input[name="name"]', 'Valid User');
      await page.fill('input[name="email"]', 'valid@example.com');
      await page.fill('input[name="password"]', 'short');
      await page.fill('input[name="confirmPassword"]', 'short');
      await page.click('button[type="submit"]');
      
      await page.waitForTimeout(300);
      
      expect(await page.locator('input[name="password"]').inputValue()).toBe('');
      expect(await page.locator('input[name="confirmPassword"]').inputValue()).toBe('');
    });

    test('should clear errors when user starts typing', async ({ page }) => {
      await page.click('button[type="submit"]');
      await page.waitForTimeout(300);
      
      const errorExists = await page.locator('.error-message, [class*="error"]').first().isVisible();
      if (errorExists) {
        await page.fill('input[name="name"]', 'New User');
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
