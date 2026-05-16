import { test as base, expect } from '@playwright/test';
import { testUsers } from '../utils/test-data';

export interface TestFixtures {
  loginAsAdmin: () => Promise<void>;
  logout: () => Promise<void>;
  clearLocalStorage: () => Promise<void>;
  clearCookies: () => Promise<void>;
}

export const test = base.extend<TestFixtures>({
  loginAsAdmin: async ({ page }, use) => {
    await use(async () => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
    });
  },

  logout: async ({ page }, use) => {
    await use(async () => {
      await page.goto('/admin/dashboard');
      await page.click('text=Logout');
      await expect(page).toHaveURL(/\/admin\/login/);
    });
  },

  clearLocalStorage: async ({ page }, use) => {
    await use(async () => {
      await page.evaluate(() => localStorage.clear());
    });
  },

  clearCookies: async ({ context }, use) => {
    await use(async () => {
      await context.clearCookies();
    });
  },
});

export { expect };
