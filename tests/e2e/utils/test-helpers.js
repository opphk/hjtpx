const { test, expect } = require('@playwright/test');
const { ConsoleErrorChecker } = require('./console-error-checker');
const { ScreenshotManager } = require('./screenshot-manager');

function createTestHelpers(page, testInfo) {
  const consoleChecker = new ConsoleErrorChecker(page, testInfo);
  const screenshotManager = new ScreenshotManager(page, testInfo);

  consoleChecker.setup();

  return {
    consoleChecker,
    screenshotManager,
    page,
    testInfo
  };
}

const baseTest = test.extend({
  testHelpers: async ({ page }, use, testInfo) => {
    const helpers = createTestHelpers(page, testInfo);
    await use(helpers);

    if (testInfo.status === 'failed') {
      await helpers.screenshotManager.capture('test-failure');
    }

    if (helpers.consoleChecker.hasErrors() || helpers.consoleChecker.hasWarnings()) {
      await helpers.consoleChecker.saveLogs();
    }
  }
});

async function waitForPageLoad(page) {
  await page.waitForLoadState('domcontentloaded');
  await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => {});
}

async function fillForm(page, formData) {
  for (const [selector, value] of Object.entries(formData)) {
    await page.fill(selector, value);
  }
}

async function waitForSelectorWithRetry(page, selector, options = {}) {
  const timeout = options.timeout || 10000;
  const retryInterval = options.retryInterval || 500;
  let elapsed = 0;

  while (elapsed < timeout) {
    try {
      const element = page.locator(selector).first();
      await element.waitFor({ state: 'visible', timeout: 1000 });
      return element;
    } catch (error) {
      elapsed += retryInterval;
      await page.waitForTimeout(retryInterval);
    }
  }

  throw new Error(`Selector ${selector} not found after ${timeout}ms`);
}

async function clickAndWait(page, selector, options = {}) {
  await Promise.all([
    page.waitForNavigation({ waitUntil: 'domcontentloaded', timeout: 30000 }).catch(() => {}),
    page.click(selector, options)
  ]);
}

function generateRandomEmail() {
  const timestamp = Date.now();
  const random = Math.floor(Math.random() * 10000);
  return `test-${timestamp}-${random}@example.com`;
}

function generateRandomString(length = 10) {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

module.exports = {
  baseTest,
  test,
  expect,
  waitForPageLoad,
  fillForm,
  waitForSelectorWithRetry,
  clickAndWait,
  generateRandomEmail,
  generateRandomString
};
