const { test as base } = require('@playwright/test');

const test = base.extend({
  testUser: async ({ request }, use) => {
    const userData = {
      email: `e2e_user_${Date.now()}@test.com`,
      name: `E2E Test User ${Date.now()}`,
      password: 'TestPassword123!'
    };

    const response = await request.post('/api/v1/auth/register', {
      data: userData
    });

    const user = response.ok() ? await response.json() : null;

    await use({ ...userData, data: user });

    if (response.ok()) {
      await request.post('/api/v1/auth/login', {
        data: {
          email: userData.email,
          password: userData.password
        }
      });
    }
  },

  adminUser: async ({ request }, use) => {
    const userData = {
      email: `e2e_admin_${Date.now()}@test.com`,
      name: `E2E Admin User ${Date.now()}`,
      password: 'TestPassword123!',
      role: 'admin'
    };

    const response = await request.post('/api/v1/auth/register', {
      data: userData
    });

    const user = response.ok() ? await response.json() : null;

    await use({ ...userData, data: user });

    if (response.ok()) {
      await request.post('/api/v1/auth/login', {
        data: {
          email: userData.email,
          password: userData.password
        }
      });
    }
  },

  authenticatedPage: async ({ page, testUser }, use) => {
    if (!testUser.data) {
      await use(page);
      return;
    }

    await page.goto('/login');
    await page.fill('input[name="email"], input[type="email"]', testUser.email);
    await page.fill('input[name="password"], input[type="password"]', testUser.password);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(1000);

    await use(page);
  },

  adminPage: async ({ page, adminUser }, use) => {
    if (!adminUser.data) {
      await use(page);
      return;
    }

    await page.goto('/login');
    await page.fill('input[name="email"], input[type="email"]', adminUser.email);
    await page.fill('input[name="password"], input[type="password"]', adminUser.password);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(1000);

    await use(page);
  }
});

module.exports = { test };
