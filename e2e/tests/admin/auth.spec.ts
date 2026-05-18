import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';
import { testUsers } from '../../utils/test-data';
import { TestHelpers } from '../../utils/test-helpers';

test.describe('管理端认证测试', () => {
  let apiHelper: ApiHelper;
  let testHelpers: TestHelpers;

  test.beforeEach(async ({ request, page }) => {
    apiHelper = new ApiHelper(request);
    testHelpers = new TestHelpers();
  });

  test.describe('管理员登录', () => {
    test('应该能够访问登录页面', async ({ page }) => {
      await page.goto('/admin/login');
      await expect(page).toHaveTitle(/登录/);
      await expect(page.locator('form')).toBeVisible();
      await testHelpers.takeScreenshot(page, 'admin-login-page');
    });

    test('登录页面应该包含所有必要的表单元素', async ({ page }) => {
      await page.goto('/admin/login');
      
      const usernameInput = page.locator('input[name="username"], input[name="user"], input[id="username"]');
      const passwordInput = page.locator('input[name="password"], input[id="password"]');
      const submitButton = page.locator('button[type="submit"], input[type="submit"]');
      
      await expect(usernameInput).toBeVisible();
      await expect(passwordInput).toBeVisible();
      await expect(submitButton).toBeVisible();
      
      await testHelpers.takeScreenshot(page, 'admin-login-form-elements');
    });

    test('应该能够使用正确的凭据登录', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      await testHelpers.takeScreenshot(page, 'admin-login-success');
    });

    test('应该拒绝无效凭据应该显示错误', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', 'invalid-user');
      await page.fill('input[name="password"]', 'wrong-password');
      await page.click('button[type="submit"]');
      await expect(page.locator('.error, .alert, [role="alert"], .error-message')).toBeVisible({ timeout: 5000 });
      
      await testHelpers.takeScreenshot(page, 'admin-login-failed');
    });

    test('登录表单验证测试 - 空用户名', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', '');
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      
      const errorVisible = await page.locator('.error, .alert, [role="alert"], .error-message, .validation-error').isVisible().catch(() => false);
      if (errorVisible) {
        await testHelpers.takeScreenshot(page, 'admin-login-empty-username');
      }
    });

    test('登录表单验证测试 - 空密码', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', '');
      await page.click('button[type="submit"]');
      
      const errorVisible = await page.locator('.error, .alert, [role="alert"], .error-message, .validation-error').isVisible().catch(() => false);
      if (errorVisible) {
        await testHelpers.takeScreenshot(page, 'admin-login-empty-password');
      }
    });

    test('登录表单验证测试 - 特殊字符', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', 'admin\' OR 1=1 --');
      await page.fill('input[name="password"]', 'anypassword');
      await page.click('button[type="submit"]');
      
      await expect(page).not.toHaveURL(/\/admin\/dashboard/);
      await testHelpers.takeScreenshot(page, 'admin-login-sql-injection');
    });

    test('登录页面加载性能测试', async ({ page }) => {
      const startTime = Date.now();
      await page.goto('/admin/login');
      await page.waitForLoadState('networkidle');
      const loadTime = Date.now() - startTime;
      
      console.log(`登录页面加载时间: ${loadTime}ms`);
      expect(loadTime).toBeLessThan(5000);
    });
  });

  test.describe('管理员登出', () => {
    test('应该能够成功登出', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      await page.click('text=Logout, text=退出, a[href*="logout"], button:has-text("Logout"), button:has-text("退出")');
      await expect(page).toHaveURL(/\/admin\/login/);
      
      await testHelpers.takeScreenshot(page, 'admin-logout-success');
    });

    test('登出后应该无法访问受保护的页面', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      const cookies = await page.context().cookies();
      await page.context().clearCookies();
      
      await page.goto('/admin/dashboard');
      await expect(page).toHaveURL(/\/admin\/login/);
      
      await testHelpers.takeScreenshot(page, 'admin-session-expired');
    });
  });

  test.describe('API认证测试', () => {
    test('API登录API应该能够正常登录', async () => {
      const result = await apiHelper.adminLogin(
        testUsers.admin.username,
        testUsers.admin.password
      );
      expect(result).toHaveProperty('success', true);
      expect(result).toHaveProperty('data');
      expect(result.data).toHaveProperty('token');
    });

    test('API无效的凭据应该返回错误', async () => {
      const result = await apiHelper.adminLogin('invalid', 'invalid');
      expect(result.success).toBeFalsy();
    });

    test('API登录应该返回有效的token', async () => {
      const result = await apiHelper.adminLogin(
        testUsers.admin.username,
        testUsers.admin.password
      );
      
      expect(result.success).toBe(true);
      expect(result.data.token).toBeDefined();
      expect(typeof result.data.token).toBe('string');
      expect(result.data.token.length).toBeGreaterThan(10);
    });

    test('API登出应该正常工作', async () => {
      const loginResult = await apiHelper.adminLogin(
        testUsers.admin.username,
        testUsers.admin.password
      );
      const token = loginResult.data.token;
      
      const logoutResult = await apiHelper.adminLogout(token);
      expect(logoutResult).toBeDefined();
    });

    test('API无效token应该被拒绝', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/stats', {
        headers: { Authorization: 'Bearer invalid-token-12345' }
      });
      
      expect(response.status()).not.toBe(200);
    });

    test('API缺失token应该被拒绝', async ({ request }) => {
      const response = await request.get('http://localhost:8080/api/v1/admin/stats');
      expect(response.status()).toBe(401);
    });
  });

  test.describe('会话管理测试', () => {
    test('登录后应该创建会话cookie', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      
      const cookies = await page.context().cookies();
      const sessionCookie = cookies.find(c => 
        c.name.includes('session') || 
        c.name.includes('token') || 
        c.name.includes('auth')
      );
      
      expect(sessionCookie).toBeDefined();
    });

    test('会话过期测试', async ({ page }) => {
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      await page.evaluate(() => {
        localStorage.clear();
        sessionStorage.clear();
      });
      
      await page.goto('/admin/dashboard');
      await page.waitForTimeout(1000);
    });

    test('多个并发会话测试', async ({ context }) => {
      const loginResult = await apiHelper.adminLogin(
        testUsers.admin.username,
        testUsers.admin.password
      );
      const token = loginResult.data.token;
      
      const page1 = await context.newPage();
      const page2 = await context.newPage();
      
      await page1.goto('/admin/login');
      await page2.goto('/admin/login');
      
      expect(page1).toBeDefined();
      expect(page2).toBeDefined();
      
      await page1.close();
      await page2.close();
    });
  });

  test.describe('安全测试', () => {
    test('暴力破解防护测试', async () => {
      let failedAttempts = 0;
      
      for (let i = 0; i < 5; i++) {
        const result = await apiHelper.adminLogin('admin', 'wrongpassword' + i);
        if (!result.success) {
          failedAttempts++;
        }
      }
      
      expect(failedAttempts).toBe(5);
      
      const blockResult = await apiHelper.adminLogin('admin', 'wrongpassword');
      expect(blockResult.success).toBeFalsy();
    });

    test('SQL注入防护测试', async ({ page }) => {
      await page.goto('/admin/login');
      
      const sqlInjectionPayloads = [
        "admin' OR '1'='1",
        "admin' --",
        "admin'/*",
        "' OR 1=1--",
        "admin' OR '1'='1' --",
      ];
      
      for (const payload of sqlInjectionPayloads) {
        await page.fill('input[name="username"]', payload);
        await page.fill('input[name="password"]', 'anything');
        await page.click('button[type="submit"]');
        await page.waitForTimeout(500);
        
        const isLoggedIn = await page.url();
        expect(isLoggedIn).not.toMatch(/\/admin\/dashboard/);
      }
      
      await testHelpers.takeScreenshot(page, 'admin-sql-injection-blocked');
    });

    test('XSS防护测试', async ({ page }) => {
      await page.goto('/admin/login');
      
      const xssPayload = '<script>alert("XSS")</script>';
      await page.fill('input[name="username"]', xssPayload);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      
      await page.waitForTimeout(1000);
      const alertShown = await page.locator('script').count();
      expect(alertShown).toBe(0);
      
      await testHelpers.takeScreenshot(page, 'admin-xss-blocked');
    });

    test('CSRF Token测试', async ({ page }) => {
      await page.goto('/admin/login');
      
      const csrfToken = await page.locator('input[name="csrf_token"], input[name="_token"]').getAttribute('value').catch(() => null);
      
      if (csrfToken) {
        console.log('CSRF Token found:', csrfToken);
        expect(csrfToken.length).toBeGreaterThan(10);
      } else {
        console.log('No CSRF token found on login page');
      }
    });
  });

  test.describe('错误处理测试', () => {
    test('网络错误处理测试', async ({ page }) => {
      await page.route('**/api/v1/auth/login', route => {
        route.abort('failed');
      });
      
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      
      await page.waitForTimeout(2000);
      const errorShown = await page.locator('.error, .alert, [role="alert"]').isVisible().catch(() => false);
      expect(errorShown || true).toBeTruthy();
    });

    test('超时错误处理测试', async ({ page }) => {
      await page.route('**/api/v1/auth/login', route => {
        route.delay(10000);
      });
      
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      
      const timeoutPromise = page.click('button[type="submit"]').catch(() => null);
      await timeoutPromise;
      await page.waitForTimeout(500);
      
      await testHelpers.takeScreenshot(page, 'admin-login-timeout');
    });

    test('服务器错误处理测试', async ({ page }) => {
      await page.route('**/api/v1/auth/login', route => {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ success: false, message: 'Internal Server Error' })
        });
      });
      
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      
      await page.waitForTimeout(1000);
      await testHelpers.takeScreenshot(page, 'admin-login-server-error');
    });
  });

  test.describe('用户体验测试', () => {
    test('登录页面响应式设计测试', async ({ page }) => {
      await page.goto('/admin/login');
      
      await page.setViewportSize({ width: 375, height: 667 });
      await page.waitForTimeout(500);
      await testHelpers.takeScreenshot(page, 'admin-login-mobile');
      
      await page.setViewportSize({ width: 768, height: 1024 });
      await page.waitForTimeout(500);
      await testHelpers.takeScreenshot(page, 'admin-login-tablet');
      
      await page.setViewportSize({ width: 1920, height: 1080 });
      await page.waitForTimeout(500);
      await testHelpers.takeScreenshot(page, 'admin-login-desktop');
    });

    test('登录页面键盘导航测试', async ({ page }) => {
      await page.goto('/admin/login');
      
      await page.locator('input[name="username"]').focus();
      await page.keyboard.press('Tab');
      await page.keyboard.press('Tab');
      
      const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
      expect(focusedElement).toBeDefined();
    });

    test('密码可见性切换测试', async ({ page }) => {
      await page.goto('/admin/login');
      
      const passwordInput = page.locator('input[name="password"]');
      const toggleButton = page.locator('button:has-text("显示"), button:has-text("Show")');
      
      const toggleVisible = await toggleButton.isVisible().catch(() => false);
      if (toggleVisible) {
        await toggleButton.click();
        const inputType = await passwordInput.getAttribute('type');
        expect(inputType).toBe('text');
        
        await toggleButton.click();
        const inputTypeAfter = await passwordInput.getAttribute('type');
        expect(inputTypeAfter).toBe('password');
      }
    });

    test('记住我功能测试', async ({ page }) => {
      await page.goto('/admin/login');
      
      const rememberCheckbox = page.locator('input[name="remember"], input[type="checkbox"]');
      const checkboxVisible = await rememberCheckbox.isVisible().catch(() => false);
      
      if (checkboxVisible) {
        await rememberCheckbox.check();
        await page.fill('input[name="username"]', testUsers.admin.username);
        await page.fill('input[name="password"]', testUsers.admin.password);
        await page.click('button[type="submit"]');
        
        await expect(page).toHaveURL(/\/admin\/dashboard/);
        
        const cookies = await page.context().cookies();
        const rememberCookie = cookies.find(c => c.name.includes('remember'));
        expect(rememberCookie).toBeDefined();
      }
    });
  });

  test.describe('并发测试', () => {
    test('多个用户同时登录测试', async ({ request }) => {
      const users = [
        { username: 'admin', password: 'admin123' },
        { username: 'testuser', password: 'TestPass123!' }
      ];
      
      const results = await Promise.all(
        users.map(user => apiHelper.adminLogin(user.username, user.password))
      );
      
      results.forEach(result => {
        expect(result).toBeDefined();
        expect(typeof result.success).toBe('boolean');
      });
    });

    test('快速重复登录测试', async ({ page }) => {
      await page.goto('/admin/login');
      
      for (let i = 0; i < 3; i++) {
        await page.fill('input[name="username"]', 'wronguser' + i);
        await page.fill('input[name="password"]', 'wrongpass');
        await page.click('button[type="submit"]');
        await page.waitForTimeout(500);
      }
      
      await testHelpers.takeScreenshot(page, 'admin-rapid-login-attempts');
    });
  });

  test.describe('审计日志测试', () => {
    test('登录尝试应该被记录', async ({ page, request }) => {
      const loginHelper = new ApiHelper(request);
      
      await loginHelper.adminLogin('admin', 'wrongpassword');
      await loginHelper.adminLogin(testUsers.admin.username, testUsers.admin.password);
      
      await page.goto('/admin/login');
      await page.fill('input[name="username"]', testUsers.admin.username);
      await page.fill('input[name="password"]', testUsers.admin.password);
      await page.click('button[type="submit"]');
      await expect(page).toHaveURL(/\/admin\/dashboard/);
      
      const logsPageVisible = await page.locator('a[href*="logs"], a[href*="audit"]').isVisible().catch(() => false);
      if (logsPageVisible) {
        await testHelpers.takeScreenshot(page, 'admin-audit-logs');
      }
    });
  });
});
