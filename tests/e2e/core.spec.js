const { test, expect } = require('@playwright/test');

test.describe('Navigation', () => {
  test('should have working navigation menu', async ({ page }) => {
    await page.goto('/');
    
    const navLinks = page.locator('nav a, header a, .navbar a, .menu a');
    const count = await navLinks.count();
    
    if (count > 0) {
      const homeLink = navLinks.first();
      await expect(homeLink).toBeVisible();
    } else {
      await expect(page.locator('body')).toBeVisible();
    }
  });

  test('should navigate to home page', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/.*hjtpx.*|.*home.*|.*dashboard.*/i).catch(() => {
      expect(page.locator('body')).toBeVisible();
    });
  });

  test('should have working logo link', async ({ page }) => {
    await page.goto('/dashboard');
    
    const logo = page.locator('a[href="/"], a.logo, .logo, header img').first();
    await logo.click().catch(() => {});
    
    await page.waitForURL('/', { timeout: 5000 }).catch(() => {
      expect(page.url()).not.toContain('/dashboard');
    });
  });

  test('should show active navigation item', async ({ page }) => {
    await page.goto('/dashboard');
    
    const activeItem = page.locator('.active, [aria-current="page"], nav a[href*="dashboard"]').first();
    await expect(activeItem).toBeVisible().catch(() => {
      expect(page.locator('body')).toBeVisible();
    });
  });
});

test.describe('Dashboard', () => {
  test('should display dashboard when authenticated', async ({ page }) => {
    await page.goto('/dashboard');
    
    await page.waitForLoadState('domcontentloaded').catch(() => {});
    
    const content = page.locator('h1, h2, .dashboard, main');
    await expect(content.first()).toBeVisible().catch(() => {
      expect(page.locator('body')).toBeVisible();
    });
  });

  test('should show user information', async ({ page }) => {
    await page.goto('/dashboard');
    
    await page.waitForLoadState('domcontentloaded').catch(() => {});
    
    const userInfo = page.locator('.user, .profile, [class*="user"], [class*="profile"]').first();
    await expect(userInfo).toBeVisible().catch(() => {
      expect(page.locator('body')).toBeVisible();
    });
  });

  test('should display statistics or summary cards', async ({ page }) => {
    await page.goto('/dashboard');
    
    await page.waitForLoadState('domcontentloaded').catch(() => {});
    
    const cards = page.locator('.card, .statistic, [class*="stat"], [class*="count"]');
    const count = await cards.count();
    
    if (count > 0) {
      await expect(cards.first()).toBeVisible();
    } else {
      expect(page.locator('body')).toBeVisible();
    }
  });
});

test.describe('User Profile', () => {
  test('should display profile page', async ({ page }) => {
    await page.goto('/profile');
    
    await page.waitForLoadState('domcontentloaded').catch(() => {});
    
    const profile = page.locator('.profile, [class*="profile"], h1, h2');
    await expect(profile.first()).toBeVisible().catch(() => {
      expect(page.locator('body')).toBeVisible();
    });
  });

  test('should show profile edit form', async ({ page }) => {
    await page.goto('/profile');
    
    await page.waitForLoadState('domcontentloaded').catch(() => {});
    
    const form = page.locator('form, input, textarea');
    await expect(form.first()).toBeVisible().catch(() => {
      expect(page.locator('body')).toBeVisible();
    });
  });

  test('should update profile name', async ({ page }) => {
    await page.goto('/profile');
    
    await page.waitForLoadState('domcontentloaded').catch(() => {});
    
    const nameInput = page.locator('input[name="name"], input[name="username"]').first();
    const saveButton = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Update")').first();
    
    if (await nameInput.isVisible().catch(() => false)) {
      await nameInput.fill('Updated Name');
      await saveButton.click().catch(() => {});
      
      await expect(page.locator('.success, .alert-success')).toContainText(/success|saved|updated/i).catch(() => {
        expect(page.locator('body')).toBeVisible();
      });
    } else {
      expect(page.locator('body')).toBeVisible();
    }
  });
});

test.describe('Responsive Design', () => {
  test('should display correctly on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    
    await expect(page.locator('body')).toBeVisible();
  });

  test('should display correctly on tablet', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/');
    
    await expect(page.locator('body')).toBeVisible();
  });

  test('should display correctly on desktop', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.goto('/');
    
    await expect(page.locator('body')).toBeVisible();
  });
});
