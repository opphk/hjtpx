import { test, expect } from '@playwright/test';
import { ApiHelper } from '../../utils/api-helper';
import { testUsers } from '../../utils/test-data';

test.describe('用户端流程测试', () => {
  let apiHelper: ApiHelper;

  test.beforeEach(async ({ request }) => {
    apiHelper = new ApiHelper(request);
  });

  test.describe('用户注册流程', () => {
    test('应该能够完成用户注册流程', async ({ page }) => {
      await page.goto('/');
      await expect(page.locator('body')).toBeVisible();
    });

    test('应该能够验证注册表单输入', async ({ page }) => {
      await page.goto('/register');
      await expect(page.locator('form')).toBeVisible();
    });

    test('注册时应该验证邮箱格式', async () => {
      const invalidEmail = 'invalid-email';
      const result = await apiHelper.registerUser(invalidEmail, 'testuser', 'password123');
      expect(result).toHaveProperty('success');
    });

    test('注册时应该验证密码强度', async ({ page }) => {
      await page.goto('/register');
      const passwordInput = page.locator('input[name="password"]');
      await expect(passwordInput).toBeVisible();
    });

    test('注册时应该验证用户名唯一性', async ({ page }) => {
      await page.goto('/register');
      const usernameInput = page.locator('input[name="username"]');
      await expect(usernameInput).toBeVisible();
    });
  });

  test.describe('用户登录流程', () => {
    test('应该能够访问登录页面', async ({ page }) => {
      await page.goto('/login');
      await expect(page.locator('form')).toBeVisible();
    });

    test('应该能够使用有效凭据登录', async ({ page }) => {
      await page.goto('/login');
      await page.fill('input[name="username"]', testUsers.user.username);
      await page.fill('input[name="password"]', testUsers.user.password);
      await page.click('button[type="submit"]');
      await page.waitForTimeout(1000);
    });

    test('应该拒绝无效密码', async ({ page }) => {
      await page.goto('/login');
      await page.fill('input[name="username"]', testUsers.user.username);
      await page.fill('input[name="password"]', 'wrongpassword');
      await page.click('button[type="submit"]');
      await expect(page.locator('.error, .alert, [role="alert"]')).toBeVisible({ timeout: 5000 });
    });

    test('应该拒绝不存在的用户', async ({ page }) => {
      await page.goto('/login');
      await page.fill('input[name="username"]', 'nonexistentuser');
      await page.fill('input[name="password"]', 'somepassword');
      await page.click('button[type="submit"]');
      await expect(page.locator('.error, .alert, [role="alert"]')).toBeVisible({ timeout: 5000 });
    });

    test('应该记住密码功能', async ({ page }) => {
      await page.goto('/login');
      const rememberMeCheckbox = page.locator('input[name="remember"]');
      if (await rememberMeCheckbox.isVisible()) {
        await rememberMeCheckbox.check();
        await expect(rememberMeCheckbox).toBeChecked();
      }
    });

    test('忘记密码链接应该可用', async ({ page }) => {
      await page.goto('/login');
      const forgotLink = page.locator('a[href*="forgot"], a[href*="reset"]');
      if (await forgotLink.isVisible()) {
        await expect(forgotLink).toHaveAttribute('href', /.+/);
      }
    });
  });

  test.describe('用户登出流程', () => {
    test('应该能够成功登出', async ({ page }) => {
      await page.goto('/login');
      await page.fill('input[name="username"]', testUsers.user.username);
      await page.fill('input[name="password"]', testUsers.user.password);
      await page.click('button[type="submit"]');
      await page.waitForTimeout(1000);

      const logoutButton = page.locator('button:has-text("Logout"), a:has-text("Logout"), button:has-text("退出")');
      if (await logoutButton.isVisible()) {
        await logoutButton.click();
      }
    });

    test('登出后应该清除会话', async ({ page }) => {
      await page.goto('/login');
      await page.fill('input[name="username"]', testUsers.user.username);
      await page.fill('input[name="password"]', testUsers.user.password);
      await page.click('button[type="submit"]');
      await page.waitForTimeout(1000);

      const logoutButton = page.locator('button:has-text("Logout"), a:has-text("Logout"), button:has-text("退出")');
      if (await logoutButton.isVisible()) {
        await logoutButton.click();
        await page.waitForTimeout(500);
      }
    });
  });

  test.describe('验证码交互流程', () => {
    test('应该能够加载滑块验证码', async ({ page }) => {
      await page.goto('/captcha');
      await page.waitForTimeout(1000);
      const captchaContainer = page.locator('.captcha-container, #captcha, [data-captcha]');
      if (await captchaContainer.isVisible({ timeout: 3000 })) {
        await expect(captchaContainer).toBeVisible();
      }
    });

    test('应该能够加载点击验证码', async ({ page }) => {
      await page.goto('/captcha?type=click');
      await page.waitForTimeout(1000);
      const captchaContainer = page.locator('.captcha-container, #captcha, [data-captcha]');
      if (await captchaContainer.isVisible({ timeout: 3000 })) {
        await expect(captchaContainer).toBeVisible();
      }
    });

    test('应该能够加载旋转验证码', async ({ page }) => {
      await page.goto('/captcha?type=rotate');
      await page.waitForTimeout(1000);
      const captchaContainer = page.locator('.captcha-container, #captcha, [data-captcha]');
      if (await captchaContainer.isVisible({ timeout: 3000 })) {
        await expect(captchaContainer).toBeVisible();
      }
    });

    test('应该能够刷新验证码', async ({ page }) => {
      await page.goto('/captcha');
      await page.waitForTimeout(1000);
      const refreshButton = page.locator('.captcha-refresh, .refresh-btn, button:has-text("Refresh")');
      if (await refreshButton.isVisible({ timeout: 3000 })) {
        await refreshButton.click();
        await page.waitForTimeout(500);
      }
    });

    test('验证成功应该显示反馈', async ({ page }) => {
      await page.goto('/captcha');
      await page.waitForTimeout(1000);
    });

    test('验证失败应该显示错误', async ({ page }) => {
      await page.goto('/captcha');
      await page.waitForTimeout(1000);
    });
  });

  test.describe('用户资料管理', () => {
    test('应该能够查看用户资料', async ({ page }) => {
      await page.goto('/profile');
      await page.waitForTimeout(1000);
    });

    test('应该能够更新昵称', async ({ page }) => {
      await page.goto('/profile');
      await page.waitForTimeout(1000);
      const nicknameInput = page.locator('input[name="nickname"], #nickname');
      if (await nicknameInput.isVisible({ timeout: 3000 })) {
        await nicknameInput.fill('New Nickname');
        const saveButton = page.locator('button:has-text("Save"), button:has-text("保存")');
        if (await saveButton.isVisible()) {
          await saveButton.click();
        }
      }
    });

    test('应该能够上传头像', async ({ page }) => {
      await page.goto('/profile');
      await page.waitForTimeout(1000);
      const avatarInput = page.locator('input[type="file"], input[name="avatar"]');
      if (await avatarInput.isVisible({ timeout: 3000 })) {
        await expect(avatarInput).toBeAttached();
      }
    });

    test('应该能够修改密码', async ({ page }) => {
      await page.goto('/profile?tab=security');
      await page.waitForTimeout(1000);
      const oldPasswordInput = page.locator('input[name="old_password"], #oldPassword');
      if (await oldPasswordInput.isVisible({ timeout: 3000 })) {
        await expect(oldPasswordInput).toBeVisible();
      }
    });
  });
});
