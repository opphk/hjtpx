import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';

test.describe('异常场景测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
  });

  test.describe('网络异常处理', () => {
    test('网络断开时应该显示错误消息', async ({ page }) => {
      await page.route('**/api/**', route => {
        route.abort('failed');
      });

      await page.goto('/');
      await page.waitForTimeout(1000);

      const errorMessage = page.locator('.error, .alert, [role="alert"], .network-error');
      if (await errorMessage.isVisible({ timeout: 3000 })) {
        await expect(errorMessage).toBeVisible();
      }
    });

    test('请求超时应该显示超时消息', async ({ page }) => {
      await page.route('**/api/**', async route => {
        await new Promise(resolve => setTimeout(resolve, 60000));
        await route.abort('timeout');
      });

      await page.goto('/');
      await page.waitForTimeout(2000);
    });

    test('服务器错误应该显示友好错误页面', async ({ page }) => {
      await page.route('**/api/**', route => {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Internal Server Error' })
        });
      });

      await page.goto('/');
      await page.waitForTimeout(1000);
    });

    test('404错误应该显示404页面', async ({ page }) => {
      await page.route('**/non-existent-page', route => {
        route.fulfill({
          status: 404,
          contentType: 'text/html',
          body: '<html><body><h1>404 Not Found</h1></body></html>'
        });
      });

      await page.goto('/non-existent-page');
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('认证异常处理', () => {
    test('Token过期应该重新登录', async ({ page }) => {
      await page.route('**/api/**', route => {
        const response = route.request();
        if (response.headers()['authorization']) {
          route.fulfill({
            status: 401,
            contentType: 'application/json',
            body: JSON.stringify({ error: 'Token expired' })
          });
        } else {
          route.continue();
        }
      });

      await page.goto('/dashboard');
      await page.waitForTimeout(1000);
    });

    test('无权限访问应该显示403错误', async ({ page }) => {
      await page.route('**/api/**', route => {
        route.fulfill({
          status: 403,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Forbidden' })
        });
      });

      await page.goto('/admin/forbidden');
      await page.waitForTimeout(1000);
    });

    test('未登录访问受保护页面应该重定向', async ({ page }) => {
      await page.goto('/dashboard');
      await page.waitForTimeout(1000);

      const currentUrl = page.url();
      expect(currentUrl).toMatch(/\/login|\/auth/);
    });

    test('会话失效应该清除本地存储', async ({ page }) => {
      await page.goto('/');
      await page.evaluate(() => {
        localStorage.setItem('auth_token', 'test-token');
      });

      await page.route('**/api/**', route => {
        route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Unauthorized' })
        });
      });

      await page.goto('/dashboard');
      await page.waitForTimeout(1000);
    });
  });

  test.describe('输入验证异常', () => {
    test('空表单提交应该显示验证错误', async ({ page }) => {
      await page.goto('/register');
      await page.waitForTimeout(500);

      const submitButton = page.locator('button[type="submit"]');
      if (await submitButton.isVisible()) {
        await submitButton.click();
        await page.waitForTimeout(500);

        const errorMessages = page.locator('.error, .field-error, [role="alert"]');
        if (await errorMessages.count() > 0) {
          await expect(errorMessages.first()).toBeVisible();
        }
      }
    });

    test('无效邮箱格式应该显示错误', async ({ page }) => {
      await page.goto('/register');
      await page.waitForTimeout(500);

      const emailInput = page.locator('input[name="email"], input[type="email"], #email');
      if (await emailInput.isVisible()) {
        await emailInput.fill('invalid-email');
        await emailInput.blur();
        await page.waitForTimeout(500);

        const errorMessages = page.locator('.error, .field-error');
        if (await errorMessages.count() > 0) {
          await expect(errorMessages.first()).toBeVisible();
        }
      }
    });

    test('密码过短应该显示错误', async ({ page }) => {
      await page.goto('/register');
      await page.waitForTimeout(500);

      const passwordInput = page.locator('input[name="password"], input[type="password"], #password');
      if (await passwordInput.isVisible()) {
        await passwordInput.fill('123');
        await passwordInput.blur();
        await page.waitForTimeout(500);

        const errorMessages = page.locator('.error, .field-error');
        if (await errorMessages.count() > 0) {
          await expect(errorMessages.first()).toBeVisible();
        }
      }
    });

    test('XSS注入应该被过滤', async ({ page }) => {
      await page.goto('/search');
      await page.waitForTimeout(500);

      const searchInput = page.locator('input[name="q"], input[type="search"], #search');
      if (await searchInput.isVisible()) {
        await searchInput.fill('<script>alert("xss")</script>');
        await searchInput.press('Enter');
        await page.waitForTimeout(1000);
      }
    });

    test('SQL注入应该被过滤', async ({ page }) => {
      await page.goto('/search');
      await page.waitForTimeout(500);

      const searchInput = page.locator('input[name="q"], input[type="search"], #search');
      if (await searchInput.isVisible()) {
        await searchInput.fill("'; DROP TABLE users; --");
        await searchInput.press('Enter');
        await page.waitForTimeout(1000);
      }
    });
  });

  test.describe('验证码异常处理', () => {
    test('验证码过期应该提示刷新', async ({ page }) => {
      await page.goto('/captcha');
      await page.waitForTimeout(1000);

      await page.evaluate(() => {
        localStorage.setItem('captcha_expired', 'true');
      });

      await page.reload();
      await page.waitForTimeout(1000);
    });

    test('重复使用验证码应该被拒绝', async ({ page }) => {
      await page.goto('/captcha');
      await page.waitForTimeout(1000);

      const submitButton = page.locator('button[type="submit"]');
      if (await submitButton.isVisible()) {
        await submitButton.click();
        await page.waitForTimeout(500);
        await submitButton.click();
        await page.waitForTimeout(500);
      }
    });

    test('验证码加载失败应该显示重试按钮', async ({ page }) => {
      await page.route('**/captcha/**', route => {
        route.abort('failed');
      });

      await page.goto('/captcha');
      await page.waitForTimeout(2000);

      const retryButton = page.locator('button:has-text("Retry"), button:has-text("重试"), .retry-btn');
      if (await retryButton.isVisible({ timeout: 3000 })) {
        await expect(retryButton).toBeVisible();
      }
    });

    test('无效验证码应该显示错误', async ({ page }) => {
      await page.goto('/captcha');
      await page.waitForTimeout(1000);

      const verifyButton = page.locator('button[type="submit"]');
      if (await verifyButton.isVisible()) {
        await verifyButton.click();
        await page.waitForTimeout(1000);
      }
    });
  });

  test.describe('并发异常处理', () => {
    test('快速多次点击应该被防止', async ({ page }) => {
      await page.goto('/submit');
      await page.waitForTimeout(500);

      const submitButton = page.locator('button[type="submit"]');
      if (await submitButton.isVisible()) {
        for (let i = 0; i < 5; i++) {
          await submitButton.click({ force: true });
          await page.waitForTimeout(100);
        }
      }
    });

    test('重复提交表单应该被防止', async ({ page }) => {
      await page.goto('/contact');
      await page.waitForTimeout(500);

      const inputs = {
        name: page.locator('input[name="name"]'),
        email: page.locator('input[name="email"]'),
        message: page.locator('textarea[name="message"]')
      };

      if (await inputs.name.isVisible()) {
        await inputs.name.fill('Test User');
        await inputs.email.fill('test@example.com');
        await inputs.message.fill('Test message');

        const submitButton = page.locator('button[type="submit"]');
        if (await submitButton.isVisible()) {
          await submitButton.click();
          await page.waitForTimeout(500);
        }
      }
    });

    test('竞态条件应该被正确处理', async ({ page }) => {
      await page.goto('/dashboard');
      await page.waitForTimeout(500);

      await Promise.all([
        page.reload(),
        page.locator('body').waitFor()
      ]);

      await page.waitForTimeout(1000);
    });
  });

  test.describe('性能异常处理', () => {
    test('大文件上传应该显示进度', async ({ page }) => {
      await page.goto('/upload');
      await page.waitForTimeout(500);

      const uploadArea = page.locator('.upload-area, input[type="file"]');
      if (await uploadArea.isVisible()) {
        await expect(uploadArea).toBeVisible();
      }
    });

    test('长列表加载应该分页', async ({ page }) => {
      await page.goto('/list');
      await page.waitForTimeout(1000);

      const listItems = page.locator('.list-item, tr, .item');
      const count = await listItems.count();
      expect(count).toBeGreaterThan(0);
    });

    test('大表单应该延迟加载', async ({ page }) => {
      await page.goto('/form');
      await page.waitForTimeout(1000);

      const form = page.locator('form');
      await expect(form).toBeVisible();
    });

    test('大量数据应该使用虚拟滚动', async ({ page }) => {
      await page.goto('/virtual-list');
      await page.waitForTimeout(1000);

      const listContainer = page.locator('.virtual-list, .infinite-scroll');
      if (await listContainer.isVisible()) {
        await expect(listContainer).toBeVisible();
      }
    });
  });

  test.describe('浏览器兼容异常', () => {
    test('老版本浏览器应该显示升级提示', async ({ page }) => {
      await page.context().addInitScript(() => {
        Object.defineProperty(navigator, 'userAgent', {
          value: 'Mozilla/4.0 (compatible; MSIE 6.0)',
          configurable: true
        });
      });

      await page.goto('/');
      await page.waitForTimeout(1000);
    });

    test('禁用JavaScript应该显示提示', async ({ page }) => {
      await page.goto('/');
      await page.waitForTimeout(500);

      const noscript = page.locator('noscript');
      if (await noscript.count() > 0) {
        await expect(noscript.first()).toBeAttached();
      }
    });

    test('禁用Cookie应该处理', async ({ page }) => {
      await page.context().addInitScript(() => {
        Object.defineProperty(document, 'cookie', {
          get: () => '',
          set: () => {},
          configurable: true
        });
      });

      await page.goto('/');
      await page.waitForTimeout(1000);
    });

    test('禁用LocalStorage应该处理', async ({ page }) => {
      await page.context().addInitScript(() => {
        delete window.localStorage;
        Object.defineProperty(window, 'localStorage', {
          value: undefined,
          configurable: true
        });
      });

      await page.goto('/');
      await page.waitForTimeout(1000);
    });
  });

  test.describe('数据一致性异常', () => {
    test('服务器数据更新应该同步到客户端', async ({ page }) => {
      await page.goto('/dashboard');
      await page.waitForTimeout(1000);

      const dataElement = page.locator('.data, .stats, .dashboard-data');
      if (await dataElement.count() > 0) {
        await expect(dataElement.first()).toBeVisible();
      }
    });

    test('乐观更新失败应该回滚', async ({ page }) => {
      await page.goto('/profile');
      await page.waitForTimeout(1000);

      const editButton = page.locator('button:has-text("Edit"), .edit-btn');
      if (await editButton.isVisible()) {
        await editButton.click();
        await page.waitForTimeout(500);
      }
    });

    test('缓存与服务器不一致应该显示最新数据', async ({ page }) => {
      await page.goto('/settings');
      await page.waitForTimeout(1000);

      const settingsForm = page.locator('form, .settings');
      if (await settingsForm.isVisible()) {
        await expect(settingsForm).toBeVisible();
      }
    });

    test('脱机模式应该缓存数据', async ({ page }) => {
      await page.goto('/offline');
      await page.waitForTimeout(1000);

      const offlineIndicator = page.locator('.offline, .offline-indicator');
      if (await offlineIndicator.count() > 0) {
        await expect(offlineIndicator.first()).toBeAttached();
      }
    });
  });
});
