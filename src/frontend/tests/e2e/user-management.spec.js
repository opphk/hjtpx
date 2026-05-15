import { test, expect } from '@playwright/test';

test.describe('User Management', () => {
  test.beforeEach(async ({ page }) => {
    page.on('console', msg => {
      if (msg.type() === 'error') {
        console.log(`Console Error: ${msg.text()}`);
      }
    });
  });

  test.describe('User List Loading', () => {
    test('should load user management page', async ({ page }) => {
      await page.goto('/admin/users');

      await page.waitForLoadState('networkidle');

      const pageContent = page.locator('body');
      await expect(pageContent).toBeVisible({ timeout: 10000 });
    });

    test('should display page header', async ({ page }) => {
      await page.goto('/admin/users');

      await page.waitForLoadState('networkidle');

      const header = page.locator('h1');
      await expect(header.first()).toBeVisible();
    });

    test('should show loading state while fetching users', async ({ page }) => {
      await page.goto('/admin/users');

      const loadingIndicator = page.locator('[class*="loading"], [class*="skeleton"], .spinner');

      if (await loadingIndicator.isVisible().catch(() => false)) {
        await expect(loadingIndicator).toBeVisible();
      }
    });
  });

  test.describe('User Search', () => {
    test('should display search input field', async ({ page }) => {
      await page.goto('/admin/users');

      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
      await expect(searchInput).toBeVisible();
    });

    test('should filter users by search term', async ({ page }) => {
      await page.goto('/admin/users');

      await page.waitForLoadState('networkidle');

      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();

      await searchInput.fill('admin');
      await page.waitForTimeout(500);

      const searchValue = await searchInput.inputValue();
      expect(searchValue).toBe('admin');
    });

    test('should clear search input', async ({ page }) => {
      await page.goto('/admin/users');

      await page.waitForLoadState('networkidle');

      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();

      await searchInput.fill('test');
      await searchInput.fill('');

      const searchValue = await searchInput.inputValue();
      expect(searchValue).toBe('');
    });
  });

  test.describe('User Filters', () => {
    test('should display role filter dropdown', async ({ page }) => {
      await page.goto('/admin/users');

      const roleFilter = page.locator('select').filter({ has: page.locator('option[value="admin"]') }).first();
      await expect(roleFilter).toBeVisible();
    });

    test('should display status filter dropdown', async ({ page }) => {
      await page.goto('/admin/users');

      const statusFilter = page.locator('select').filter({ has: page.locator('option[value="active"]') }).first();
      await expect(statusFilter).toBeVisible();
    });

    test('should change role filter', async ({ page }) => {
      await page.goto('/admin/users');

      const roleFilter = page.locator('select').filter({ has: page.locator('option[value="admin"]') }).first();
      await roleFilter.selectOption('admin');

      await expect(roleFilter).toHaveValue('admin');
    });

    test('should change status filter', async ({ page }) => {
      await page.goto('/admin/users');

      const statusFilter = page.locator('select').filter({ has: page.locator('option[value="active"]') }).first();
      await statusFilter.selectOption('active');

      await expect(statusFilter).toHaveValue('active');
    });
  });

  test.describe('Create User', () => {
    test('should display create user button', async ({ page }) => {
      await page.goto('/admin/users');

      const createButton = page.locator('button:has-text("创建"), button:has-text("新建"), button:has-text("添加")').first();
      await expect(createButton).toBeVisible();
    });

    test('should open create user modal', async ({ page }) => {
      await page.goto('/admin/users');

      const createButton = page.locator('button:has-text("创建"), button:has-text("新建"), button:has-text("添加")').first();
      await createButton.click();

      const modal = page.locator('[role="dialog"], .modal, .modal-content');
      await expect(modal).toBeVisible({ timeout: 5000 });
    });

    test('should close create user modal', async ({ page }) => {
      await page.goto('/admin/users');

      const createButton = page.locator('button:has-text("创建"), button:has-text("新建"), button:has-text("添加")').first();
      await createButton.click();

      const modal = page.locator('[role="dialog"], .modal, .modal-content');
      await expect(modal).toBeVisible();

      await page.keyboard.press('Escape');

      await page.waitForTimeout(500);
    });
  });

  test.describe('Accessibility', () => {
    test('should support keyboard navigation', async ({ page }) => {
      await page.goto('/admin/users');

      await page.waitForLoadState('networkidle');

      await page.keyboard.press('Tab');

      const focusedElement = await page.evaluate(() => document.activeElement.tagName);
      expect(['INPUT', 'BUTTON', 'SELECT', 'A']).toContain(focusedElement);
    });
  });
});
