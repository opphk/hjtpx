import { chromium } from 'playwright';

const BASE_URL = 'http://localhost:8080';
const SCREENSHOT_DIR = '/workspace/hjtpx/e2e/test-screenshots/v11';

async function takeScreenshot(page, name) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  const path = `${SCREENSHOT_DIR}/${name}_${timestamp}.png`;
  await page.screenshot({ path, fullPage: false });
  console.log(`Screenshot saved: ${path}`);
  return path;
}

async function checkConsoleErrors(page) {
  const errors = [];
  page.on('console', msg => {
    if (msg.type() === 'error') {
      errors.push(msg.text());
    }
  });
  return errors;
}

async function runTests() {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  const results = [];
  const errors = [];

  const pages = [
    { url: '/', name: 'home-page' },
    { url: '/captcha', name: 'slider-captcha-page' },
    { url: '/click-captcha', name: 'click-captcha-page' },
    { url: '/seamless', name: 'seamless-page' },
    { url: '/admin/login', name: 'admin-login-page' },
  ];

  const consoleErrors = [];

  page.on('console', msg => {
    if (msg.type() === 'error') {
      consoleErrors.push({ page: page.url(), error: msg.text() });
    }
  });

  for (const p of pages) {
    try {
      console.log(`Testing: ${BASE_URL}${p.url}`);
      const response = await page.goto(`${BASE_URL}${p.url}`, {
        waitUntil: 'networkidle',
        timeout: 30000
      });

      const status = response?.status() || 0;
      await page.waitForTimeout(1000);

      const screenshot = await takeScreenshot(page, p.name);

      results.push({
        page: p.url,
        status,
        screenshot,
        success: status >= 200 && status < 400
      });

      console.log(`  Status: ${status}, Screenshot: ${screenshot}`);
    } catch (err) {
      console.error(`  Error: ${err.message}`);
      errors.push({ page: p.url, error: err.message });
      results.push({
        page: p.url,
        status: 0,
        error: err.message,
        success: false
      });
    }
  }

  await browser.close();

  console.log('\n=== Test Summary ===');
  console.log(`Total pages tested: ${pages.length}`);
  console.log(`Successful: ${results.filter(r => r.success).length}`);
  console.log(`Failed: ${results.filter(r => !r.success).length}`);
  console.log(`Console errors found: ${consoleErrors.length}`);

  if (consoleErrors.length > 0) {
    console.log('\n=== Console Errors ===');
    for (const e of consoleErrors) {
      console.log(`[${e.page}] ${e.error}`);
    }
  }

  if (errors.length > 0) {
    console.log('\n=== Page Errors ===');
    for (const e of errors) {
      console.log(`[${e.page}] ${e.error}`);
    }
  }

  return { results, consoleErrors, errors };
}

runTests()
  .then(() => {
    console.log('\nTests completed.');
    process.exit(0);
  })
  .catch(err => {
    console.error('Test failed:', err);
    process.exit(1);
  });
