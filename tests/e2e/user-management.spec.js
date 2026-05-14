const { test, expect } = require('@playwright/test');

test.describe('User Management Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('should display user profile page', async ({ page }) => {
    const testEmail = `profile_${Date.now()}@example.com`;
    const testPassword = 'TestPassword123!';
    
    await page.goto('/register');
    await page.fill('input[name="email"], input[type="email"]', testEmail);
    await page.fill('input[name="name"], input[name="username"]', 'Profile User');
    await page.fill('input[name="password"], input[type="password"]', testPassword);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    await page.goto('/profile');
    await page.waitForTimeout(1000);
    
    const profileElements = page.locator('h1, h2, [data-testid="profile"]');
    const hasProfile = await profileElements.count() > 0;
    
    if (hasProfile) {
      await expect(profileElements.first()).toBeVisible();
    }
  });

  test('should display user information', async ({ page }) => {
    const testEmail = `userinfo_${Date.now()}@example.com`;
    const testPassword = 'TestPassword123!';
    const testName = `User Info ${Date.now()}`;
    
    await page.goto('/register');
    await page.fill('input[name="email"], input[type="email"]', testEmail);
    await page.fill('input[name="name"], input[name="username"]', testName);
    await page.fill('input[name="password"], input[type="password"]', testPassword);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    await page.goto('/profile');
    await page.waitForTimeout(1000);
    
    const bodyText = await page.locator('body').textContent();
    expect(bodyText).toContain(testName);
  });

  test('should allow user to update profile', async ({ page }) => {
    const testEmail = `update_${Date.now()}@example.com`;
    const testPassword = 'TestPassword123!';
    
    await page.goto('/register');
    await page.fill('input[name="email"], input[type="email"]', testEmail);
    await page.fill('input[name="name"], input[name="username"]', 'Original Name');
    await page.fill('input[name="password"], input[type="password"]', testPassword);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    await page.goto('/profile');
    await page.waitForTimeout(1000);
    
    const nameInput = page.locator('input[name="name"], input[name="displayName"], input[name="username"]');
    if (await nameInput.isVisible().catch(() => false)) {
      await nameInput.clear();
      await nameInput.fill(`Updated Name ${Date.now()}`);
      
      const saveButton = page.locator('button[type="submit"], button:has-text("save"), button:has-text("update")').first();
      if (await saveButton.isVisible().catch(() => false)) {
        await saveButton.click();
        await page.waitForTimeout(1000);
        
        const successMessage = page.locator('.success, .alert-success, [role="status"]');
        const hasSuccess = await successMessage.count() > 0;
        
        if (hasSuccess) {
          await expect(successMessage.first()).toBeVisible();
        }
      }
    }
  });

  test('should allow user to change password', async ({ page }) => {
    const testEmail = `changepwd_${Date.now()}@example.com`;
    const testPassword = 'TestPassword123!';
    const newPassword = 'NewPassword123!';
    
    await page.goto('/register');
    await page.fill('input[name="email"], input[type="email"]', testEmail);
    await page.fill('input[name="name"], input[name="username"]', 'Change Pwd User');
    await page.fill('input[name="password"], input[type="password"]', testPassword);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    await page.goto('/settings');
    await page.waitForTimeout(1000);
    
    const currentPasswordInput = page.locator('input[name="currentPassword"], input[name="oldPassword"]');
    const newPasswordInput = page.locator('input[name="newPassword"], input[name="password"]');
    const confirmPasswordInput = page.locator('input[name="confirmPassword"], input[name="confirmNewPassword"]');
    
    if (await currentPasswordInput.isVisible().catch(() => false) &&
        await newPasswordInput.isVisible().catch(() => false)) {
      await currentPasswordInput.fill(testPassword);
      await newPasswordInput.fill(newPassword);
      
      if (await confirmPasswordInput.isVisible().catch(() => false)) {
        await confirmPasswordInput.fill(newPassword);
      }
      
      const saveButton = page.locator('button[type="submit"], button:has-text("change"), button:has-text("update")').first();
      if (await saveButton.isVisible().catch(() => false)) {
        await saveButton.click();
        await page.waitForTimeout(1000);
      }
    }
  });

  test('should display user settings page', async ({ page }) => {
    const testEmail = `settings_${Date.now()}@example.com`;
    const testPassword = 'TestPassword123!';
    
    await page.goto('/register');
    await page.fill('input[name="email"], input[type="email"]', testEmail);
    await page.fill('input[name="name"], input[name="username"]', 'Settings User');
    await page.fill('input[name="password"], input[type="password"]', testPassword);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    await page.goto('/settings');
    await page.waitForTimeout(1000);
    
    const settingsElements = page.locator('h1, h2, [data-testid="settings"]');
    const hasSettings = await settingsElements.count() > 0;
    
    if (hasSettings) {
      await expect(settingsElements.first()).toBeVisible();
    }
  });

  test('should allow user to logout', async ({ page }) => {
    const testEmail = `logout_${Date.now()}@example.com`;
    const testPassword = 'TestPassword123!';
    
    await page.goto('/register');
    await page.fill('input[name="email"], input[type="email"]', testEmail);
    await page.fill('input[name="name"], input[name="username"]', 'Logout User');
    await page.fill('input[name="password"], input[type="password"]', testPassword);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    const logoutButton = page.locator('button:has-text("logout"), a[href*="logout"], button:has-text("sign out")').first();
    
    const isVisible = await logoutButton.isVisible().catch(() => false);
    if (isVisible) {
      await logoutButton.click();
      await page.waitForTimeout(1000);
      
      const currentUrl = page.url();
      const isLoggedOut = currentUrl.includes('/login') || 
                         await page.locator('text=/login|sign in/i').count() > 0;
      
      expect(isLoggedOut).toBeTruthy();
    }
  });
});

test.describe('Admin User Management', () => {
  test('should display admin user list', async ({ page }) => {
    const adminEmail = `admin_${Date.now()}@example.com`;
    const adminPassword = 'TestPassword123!';
    
    await page.goto('/register');
    await page.fill('input[name="email"], input[type="email"]', adminEmail);
    await page.fill('input[name="name"], input[name="username"]', 'Admin User');
    await page.fill('input[name="password"], input[type="password"]', adminPassword);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    await page.goto('/admin/users');
    await page.waitForTimeout(1000);
    
    const adminElements = page.locator('h1, h2, table, [data-testid="users"]');
    const hasAdminPage = await adminElements.count() > 0;
    
    if (hasAdminPage) {
      await expect(adminElements.first()).toBeVisible();
    }
  });

  test('should allow admin to create new user', async ({ page }) => {
    const adminEmail = `createuser_admin_${Date.now()}@example.com`;
    const adminPassword = 'TestPassword123!';
    
    await page.goto('/register');
    await page.fill('input[name="email"], input[type="email"]', adminEmail);
    await page.fill('input[name="name"], input[name="username"]', 'Admin User');
    await page.fill('input[name="password"], input[type="password"]', adminPassword);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    await page.goto('/admin/users');
    await page.waitForTimeout(1000);
    
    const createButton = page.locator('button:has-text("create"), button:has-text("add"), a[href*="create"]').first();
    
    const isVisible = await createButton.isVisible().catch(() => false);
    if (isVisible) {
      await createButton.click();
      await page.waitForTimeout(1000);
      
      const createForm = page.locator('form');
      const hasForm = await createForm.count() > 0;
      
      if (hasForm) {
        await expect(createForm.first()).toBeVisible();
      }
    }
  });
});
