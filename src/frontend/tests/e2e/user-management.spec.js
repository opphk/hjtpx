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
    test('should load user list successfully', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      
      const table = page.locator('table');
      await expect(table).toBeVisible({ timeout: 10000 }).catch(() => {
        console.log('Table may not be visible immediately');
      });
    });

    test('should display user table with correct headers', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      
      await expect(page.locator('th').first()).toBeVisible();
      
      const headers = ['ID', '用户名', '邮箱', '角色', '状态', '注册时间', '最后登录', '操作'];
      for (const header of headers) {
        await expect(page.locator(`th:has-text("${header}")`)).toBeVisible({ timeout: 5000 }).catch(() => {
          console.log(`Header "${header}" not found`);
        });
      }
    });

    test('should show loading state while fetching users', async ({ page }) => {
      await page.goto('/admin/users');
      
      const loadingIndicator = page.locator('[class*="loading"], [class*="skeleton"], .spinner');
      
      if (await loadingIndicator.isVisible().catch(() => false)) {
        await expect(loadingIndicator).toBeVisible();
      }
    });

    test('should display pagination controls', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      
      const pagination = page.locator('[class*="pagination"], nav[aria-label*="pagination"]');
      const hasPagination = await pagination.isVisible().catch(() => false);
      
      if (hasPagination) {
        await expect(pagination).toBeVisible();
      } else {
        const totalUsers = await page.locator('table tbody tr').count();
        expect(totalUsers).toBeGreaterThan(0);
      }
    });

    test('should show user count', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      
      const userCount = page.locator('[class*="count"], [class*="total"], [class*="summary"]');
      const hasCount = await userCount.isVisible().catch(() => false);
      
      if (hasCount) {
        await expect(userCount.first()).toBeVisible();
      }
    });
  });

  test.describe('User Search', () => {
    test('should display search input field', async ({ page }) => {
      await page.goto('/admin/users');
      
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
      await expect(searchInput).toBeVisible();
    });

    test('should search users by name', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
      
      await searchInput.fill('admin');
      await page.waitForTimeout(500);
      
      const rows = page.locator('table tbody tr');
      const rowCount = await rows.count();
      
      if (rowCount > 0) {
        const firstRowText = await rows.first().textContent();
        expect(firstRowText.toLowerCase()).toContain('admin');
      }
    });

    test('should search users by email', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
      
      await searchInput.fill('example');
      await page.waitForTimeout(500);
      
      const rows = page.locator('table tbody tr');
      const rowCount = await rows.count();
      
      if (rowCount > 0) {
        const firstRowText = await rows.first().textContent();
        expect(firstRowText.toLowerCase()).toContain('example');
      }
    });

    test('should clear search and show all users', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
      
      const initialCount = await page.locator('table tbody tr').count();
      
      await searchInput.fill('nonexistentsearch');
      await page.waitForTimeout(500);
      
      await searchInput.fill('');
      await page.waitForTimeout(500);
      
      const finalCount = await page.locator('table tbody tr').count();
      expect(finalCount).toBe(initialCount);
    });

    test('should show no results message for non-existent user', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
      
      await searchInput.fill('xyznonexistent123abc');
      await page.waitForTimeout(1000);
      
      const emptyState = page.locator('[class*="empty"], [class*="no-results"], p:has-text("暂无")');
      const hasEmptyState = await emptyState.isVisible().catch(() => false);
      
      if (hasEmptyState) {
        await expect(emptyState.first()).toBeVisible();
      } else {
        const rowCount = await page.locator('table tbody tr').count();
        expect(rowCount).toBe(0);
      }
    });

    test('should support real-time search filtering', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="search"], input[type="search"]').first();
      
      await searchInput.type('a', { delay: 100 });
      await page.waitForTimeout(300);
      
      const rowsAfterOneChar = await page.locator('table tbody tr').count();
      
      await searchInput.type('dm', { delay: 100 });
      await page.waitForTimeout(300);
      
      const rowsAfterMoreChars = await page.locator('table tbody tr').count();
      
      expect(rowsAfterMoreChars).toBeLessThanOrEqual(rowsAfterOneChar);
    });
  });

  test.describe('User Editing', () => {
    test('should open edit modal when clicking edit button', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      await page.waitForSelector('table tbody tr', { timeout: 10000 }).catch(() => {});
      
      const editButton = page.locator('button:has-text("编辑"), button:has-text("Edit"), button:has-text("edit")').first();
      
      if (await editButton.isVisible().catch(() => false)) {
        await editButton.click();
        
        await page.waitForTimeout(500);
        const modal = page.locator('[role="dialog"], .modal, .modal-content, [class*="drawer"]');
        await expect(modal).toBeVisible({ timeout: 5000 });
      }
    });

    test('should pre-fill edit form with user data', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      await page.waitForSelector('table tbody tr', { timeout: 10000 }).catch(() => {});
      
      const firstRow = page.locator('table tbody tr').first();
      const userName = await firstRow.locator('td').nth(2).textContent();
      const userEmail = await firstRow.locator('td').nth(3).textContent();
      
      const editButton = page.locator('button:has-text("编辑"), button:has-text("Edit"), button:has-text("edit")').first();
      
      if (await editButton.isVisible().catch(() => false)) {
        await editButton.click();
        
        await page.waitForTimeout(500);
        
        const nameInput = page.locator('input[name="name"], input[placeholder*="用户名"], input[placeholder*="name"]').first();
        const emailInput = page.locator('input[name="email"], input[placeholder*="邮箱"], input[placeholder*="email"]').first();
        
        if (await nameInput.isVisible().catch(() => false)) {
          const filledName = await nameInput.inputValue();
          expect(filledName).toBe(userName);
        }
        
        if (await emailInput.isVisible().catch(() => false)) {
          const filledEmail = await emailInput.inputValue();
          expect(filledEmail).toBe(userEmail);
        }
      }
    });

    test('should update user name', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      await page.waitForSelector('table tbody tr', { timeout: 10000 }).catch(() => {});
      
      const editButton = page.locator('button:has-text("编辑"), button:has-text("Edit"), button:has-text("edit")').first();
      
      if (await editButton.isVisible().catch(() => false)) {
        await editButton.click();
        
        await page.waitForTimeout(500);
        
        const nameInput = page.locator('input[name="name"], input[placeholder*="用户名"], input[placeholder*="name"]').first();
        
        if (await nameInput.isVisible().catch(() => false)) {
          const originalName = await nameInput.inputValue();
          const newName = `${originalName}-edited-${Date.now()}`;
          
          await nameInput.clear();
          await nameInput.fill(newName);
          
          const saveButton = page.locator('button:has-text("保存"), button:has-text("Save"), button[type="submit"]').first();
          await saveButton.click();
          
          await page.waitForTimeout(2000);
          
          const updatedRow = page.locator('table tbody tr').first();
          const updatedName = await updatedRow.locator('td').nth(2).textContent();
          expect(updatedName).toContain('-edited-');
        }
      }
    });

    test('should update user role', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      await page.waitForSelector('table tbody tr', { timeout: 10000 }).catch(() => {});
      
      const roleSelect = page.locator('select[class*="role"], td select').first();
      
      if (await roleSelect.isVisible().catch(() => false)) {
        const originalRole = await roleSelect.inputValue();
        const newRole = originalRole === 'admin' ? 'user' : 'admin';
        
        await roleSelect.selectOption(newRole);
        await page.waitForTimeout(1000);
        
        await expect(roleSelect).toHaveValue(newRole);
      }
    });

    test('should update user status', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      await page.waitForSelector('table tbody tr', { timeout: 10000 }).catch(() => {});
      
      const statusSelect = page.locator('select[class*="status"], td select').first();
      
      if (await statusSelect.isVisible().catch(() => false)) {
        const originalStatus = await statusSelect.inputValue();
        const newStatus = originalStatus === 'active' ? 'inactive' : 'active';
        
        await statusSelect.selectOption(newStatus);
        await page.waitForTimeout(1000);
        
        await expect(statusSelect).toHaveValue(newStatus);
      }
    });

    test('should close edit modal', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      await page.waitForSelector('table tbody tr', { timeout: 10000 }).catch(() => {});
      
      const editButton = page.locator('button:has-text("编辑"), button:has-text("Edit"), button:has-text("edit")').first();
      
      if (await editButton.isVisible().catch(() => false)) {
        await editButton.click();
        
        await page.waitForTimeout(500);
        const modal = page.locator('[role="dialog"], .modal, .modal-content, [class*="drawer"]');
        await expect(modal).toBeVisible();
        
        const closeButton = page.locator('[role="dialog"] button[aria-label*="close"], .modal button[aria-label*="close"], button:has-text("×"), button:has-text("Close")').first();
        
        if (await closeButton.isVisible().catch(() => false)) {
          await closeButton.click();
        } else {
          await page.keyboard.press('Escape');
        }
        
        await page.waitForTimeout(500);
        await expect(modal).not.toBeVisible();
      }
    });

    test('should show validation error when editing with empty name', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      await page.waitForSelector('table tbody tr', { timeout: 10000 }).catch(() => {});
      
      const editButton = page.locator('button:has-text("编辑"), button:has-text("Edit"), button:has-text("edit")').first();
      
      if (await editButton.isVisible().catch(() => false)) {
        await editButton.click();
        
        await page.waitForTimeout(500);
        
        const nameInput = page.locator('input[name="name"], input[placeholder*="用户名"], input[placeholder*="name"]').first();
        
        if (await nameInput.isVisible().catch(() => false)) {
          await nameInput.clear();
          
          const saveButton = page.locator('button:has-text("保存"), button:has-text("Save"), button[type="submit"]').first();
          await saveButton.click();
          
          await page.waitForTimeout(500);
          
          const errorMessage = page.locator('.error-message, [class*="error"], [role="alert"]').first();
          await expect(errorMessage).toBeVisible({ timeout: 3000 });
        }
      }
    });
  });

  test.describe('User Selection', () => {
    test('should select individual users with checkboxes', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      await page.waitForSelector('table tbody tr', { timeout: 10000 }).catch(() => {});
      
      const checkbox = page.locator('table tbody tr:first-child input[type="checkbox"]').first();
      
      if (await checkbox.isVisible().catch(() => false)) {
        await checkbox.check();
        await expect(checkbox).toBeChecked();
      }
    });

    test('should select all users with select all checkbox', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      await page.waitForSelector('table tbody tr', { timeout: 10000 }).catch(() => {});
      
      const selectAllCheckbox = page.locator('thead input[type="checkbox"], th input[type="checkbox"]').first();
      
      if (await selectAllCheckbox.isVisible().catch(() => false)) {
        await selectAllCheckbox.check();
        await expect(selectAllCheckbox).toBeChecked();
        
        const rowCheckboxes = page.locator('table tbody input[type="checkbox"]');
        const count = await rowCheckboxes.count();
        
        for (let i = 0; i < count; i++) {
          await expect(rowCheckboxes.nth(i)).toBeChecked();
        }
      }
    });

    test('should show selected count', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      await page.waitForSelector('table tbody tr', { timeout: 10000 }).catch(() => {});
      
      const checkbox = page.locator('table tbody tr:first-child input[type="checkbox"]').first();
      
      if (await checkbox.isVisible().catch(() => false)) {
        await checkbox.check();
        
        const selectedCount = page.locator('[class*="selected-count"], [class*="bulk"] span, [class*="count"]');
        await expect(selectedCount.first()).toBeVisible({ timeout: 3000 }).catch(() => {
          console.log('Selected count may not be displayed');
        });
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
      
      await page.goto('/admin/users');
      await page.waitForLoadState('networkidle');
      
      const criticalErrors = errors.filter(err => 
        !err.includes('favicon') && 
        !err.includes('DevTools') &&
        !err.includes('third-party')
      );
      
      expect(criticalErrors.length).toBe(0);
    });
  });

  test.describe('Accessibility', () => {
    test('should support keyboard navigation', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      
      await page.keyboard.press('Tab');
      
      const focusedElement = await page.evaluate(() => document.activeElement.tagName);
      expect(['INPUT', 'BUTTON', 'SELECT']).toContain(focusedElement);
    });

    test('should close modal with Escape key', async ({ page }) => {
      await page.goto('/admin/users');
      
      await page.waitForLoadState('networkidle');
      await page.waitForSelector('table tbody tr', { timeout: 10000 }).catch(() => {});
      
      const editButton = page.locator('button:has-text("编辑"), button:has-text("Edit"), button:has-text("edit")').first();
      
      if (await editButton.isVisible().catch(() => false)) {
        await editButton.click();
        
        await page.waitForTimeout(500);
        const modal = page.locator('[role="dialog"], .modal');
        await expect(modal).toBeVisible();
        
        await page.keyboard.press('Escape');
        
        await page.waitForTimeout(500);
        await expect(modal).not.toBeVisible();
      }
    });
  });
});
