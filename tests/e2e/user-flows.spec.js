import { test, expect } from '@playwright/test';

test.describe('用户注册流程', () => {
  test('完整注册流程', async ({ page }) => {
    await page.goto('/register');
    
    // 填写注册表单
    await page.fill('input[name="username"]', 'testuser123');
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'SecurePass123!');
    await page.fill('input[name="confirmPassword"]', 'SecurePass123!');
    
    // 提交表单
    await page.click('button[type="submit"]');
    
    // 验证成功消息
    await expect(page.locator('.alert-success')).toBeVisible();
    await expect(page.locator('.alert-success')).toContainText('注册成功');
  });
  
  test('注册失败 - 邮箱已存在', async ({ page }) => {
    await page.goto('/register');
    
    // 使用已存在的邮箱
    await page.fill('input[name="username"]', 'newuser');
    await page.fill('input[name="email"]', 'existing@example.com');
    await page.fill('input[name="password"]', 'Password123!');
    await page.fill('input[name="confirmPassword"]', 'Password123!');
    
    await page.click('button[type="submit"]');
    
    // 验证错误消息
    await expect(page.locator('.alert-error')).toContainText('邮箱已存在');
  });
  
  test('注册失败 - 密码不匹配', async ({ page }) => {
    await page.goto('/register');
    
    await page.fill('input[name="username"]', 'testuser');
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'Password123!');
    await page.fill('input[name="confirmPassword"]', 'DifferentPass!');
    
    await page.click('button[type="submit"]');
    
    // 验证密码不匹配错误
    await expect(page.locator('text=密码不匹配')).toBeVisible();
  });
});

test.describe('用户登录流程', () => {
  test('成功登录', async ({ page }) => {
    await page.goto('/login');
    
    await page.fill('input[name="email"]', 'user@example.com');
    await page.fill('input[name="password"]', 'Password123!');
    
    await page.click('button[type="submit"]');
    
    // 验证跳转到仪表板
    await expect(page).toHaveURL('/dashboard');
    await expect(page.locator('text=欢迎')).toBeVisible();
  });
  
  test('登录失败 - 错误密码', async ({ page }) => {
    await page.goto('/login');
    
    await page.fill('input[name="email"]', 'user@example.com');
    await page.fill('input[name="password"]', 'WrongPassword!');
    
    await page.click('button[type="submit"]');
    
    await expect(page.locator('.alert-error')).toContainText('邮箱或密码错误');
  });
});

test.describe('密码重置流程', () => {
  test('请求密码重置', async ({ page }) => {
    await page.goto('/forgot-password');
    
    await page.fill('input[name="email"]', 'user@example.com');
    await page.click('button[type="submit"]');
    
    await expect(page.locator('.alert-success')).toContainText('重置链接已发送');
  });
  
  test('重置密码', async ({ page }) => {
    // 模拟通过邮箱链接访问
    await page.goto('/reset-password?token=test-token-123');
    
    await page.fill('input[name="newPassword"]', 'NewSecurePass123!');
    await page.fill('input[name="confirmPassword"]', 'NewSecurePass123!');
    
    await page.click('button[type="submit"]');
    
    await expect(page.locator('.alert-success')).toContainText('密码重置成功');
  });
});

test.describe('用户管理流程', () => {
  test.beforeEach(async ({ page }) => {
    // 登录为管理员
    await page.goto('/login');
    await page.fill('input[name="email"]', 'admin@example.com');
    await page.fill('input[name="password"]', 'AdminPass123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('/dashboard');
  });
  
  test('查看用户列表', async ({ page }) => {
    await page.goto('/users');
    
    // 验证用户列表加载
    await expect(page.locator('table.users-table')).toBeVisible();
    
    // 验证分页
    await expect(page.locator('.pagination')).toBeVisible();
  });
  
  test('编辑用户信息', async ({ page }) => {
    await page.goto('/users');
    
    // 点击编辑按钮
    await page.click('button.edit-user:first-child');
    
    // 验证模态框打开
    await expect(page.locator('[role="dialog"]')).toBeVisible();
    
    // 修改用户名
    await page.fill('input[name="username"]', 'updated-user');
    
    // 保存
    await page.click('button.save-user');
    
    // 验证成功
    await expect(page.locator('.alert-success')).toContainText('用户已更新');
  });
  
  test('删除用户', async ({ page }) => {
    await page.goto('/users');
    
    // 点击删除按钮
    await page.click('button.delete-user:first-child');
    
    // 确认删除
    await page.click('button.confirm-delete');
    
    // 验证用户被删除
    await expect(page.locator('.alert-success')).toContainText('用户已删除');
  });
});
