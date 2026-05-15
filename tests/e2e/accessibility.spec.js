import { test, expect } from '@playwright/test';

test.describe('无障碍功能测试', () => {
  test('键盘导航 - Tab键焦点顺序', async ({ page }) => {
    await page.goto('/register');
    
    // 第一个焦点应该在第一个输入框
    const firstInput = page.locator('input[name="username"]');
    await expect(firstInput).toBeFocused();
    
    // Tab键应该依次聚焦到下一个输入框
    await page.keyboard.press('Tab');
    const emailInput = page.locator('input[name="email"]');
    await expect(emailInput).toBeFocused();
  });
  
  test('模态框 - Escape键关闭', async ({ page }) => {
    await page.goto('/users');
    
    // 打开模态框
    await page.click('button:has-text("编辑")');
    
    // 按Escape键关闭
    await page.keyboard.press('Escape');
    
    // 验证模态框已关闭
    await expect(page.locator('[role="dialog"]')).not.toBeVisible();
  });
  
  test('屏幕阅读器 - ARIA标签', async ({ page }) => {
    await page.goto('/register');
    
    // 验证表单输入有ARIA标签
    const usernameInput = page.locator('input[name="username"]');
    await expect(usernameInput).toHaveAttribute('aria-label', /用户名/);
    
    const emailInput = page.locator('input[name="email"]');
    await expect(emailInput).toHaveAttribute('aria-label', /邮箱/);
  });
  
  test('SkipLink - 跳转功能', async ({ page }) => {
    await page.goto('/');
    
    // 激活SkipLink
    await page.keyboard.press('Tab');
    const skipLink = page.locator('a.skip-link');
    await expect(skipLink).toBeFocused();
    
    // 按Enter跳转到主内容
    await page.keyboard.press('Enter');
    await expect(page.locator('#main-content')).toBeFocused();
  });
  
  test('焦点管理 - 模态框打开/关闭', async ({ page }) => {
    await page.goto('/users');
    
    // 打开模态框前记录焦点
    const editButton = page.locator('button:has-text("编辑")').first();
    
    // 打开模态框
    await editButton.click();
    
    // 验证焦点在模态框内
    const modalInput = page.locator('[role="dialog"] input').first();
    await expect(modalInput).toBeFocused();
    
    // 关闭模态框
    await page.keyboard.press('Escape');
    
    // 验证焦点返回到触发按钮
    await expect(editButton).toBeFocused();
  });
});
