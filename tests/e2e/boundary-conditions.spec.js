import { test, expect } from '@playwright/test';

test.describe('表单边界条件测试', () => {
  test('用户名最小长度', async ({ page }) => {
    await page.goto('/register');
    
    // 输入1个字符的用户名
    await page.fill('input[name="username"]', 'a');
    await page.fill('input[name="email"]', 'test@example.com');
    await page.fill('input[name="password"]', 'Password123!');
    await page.fill('input[name="confirmPassword"]', 'Password123!');
    
    await page.click('button[type="submit"]');
    
    // 应该显示最小长度错误
    await expect(page.locator('text=用户名至少3个字符')).toBeVisible();
  });
  
  test('用户名最大长度', async ({ page }) => {
    await page.goto('/register');
    
    // 输入超长用户名（50个字符）
    const longUsername = 'a'.repeat(51);
    await page.fill('input[name="username"]', longUsername);
    
    // 应该显示最大长度错误
    await expect(page.locator('text=用户名最多50个字符')).toBeVisible();
  });
  
  test('邮箱格式验证', async ({ page }) => {
    await page.goto('/register');
    
    // 无效邮箱格式
    const invalidEmails = [
      'notanemail',
      '@example.com',
      'test@',
      'test@example',
      'test @example.com'
    ];
    
    for (const email of invalidEmails) {
      await page.fill('input[name="email"]', email);
      await page.click('input[name="username"]'); // 触发验证
      
      await expect(page.locator('text=请输入有效的邮箱地址')).toBeVisible();
    }
  });
  
  test('密码强度要求', async ({ page }) => {
    await page.goto('/register');
    
    const weakPasswords = [
      '123456',
      'password',
      'aaaaaa',
      'Pass123' // 太短
    ];
    
    for (const password of weakPasswords) {
      await page.fill('input[name="password"]', password);
      await page.click('input[name="username"]'); // 触发验证
      
      // 应该有密码强度不足的错误
      const error = page.locator('text=密码强度不足');
      await expect(error).toBeVisible();
    }
  });
});

test.describe('输入特殊字符测试', () => {
  test('XSS攻击防护', async ({ page }) => {
    await page.goto('/register');
    
    const xssPayloads = [
      '<script>alert("XSS")</script>',
      '"><script>alert("XSS")</script>',
      "javascript:alert('XSS')",
      '<img src=x onerror=alert("XSS")>'
    ];
    
    for (const payload of xssPayloads) {
      await page.fill('input[name="username"]', payload);
      
      // 验证输入被转义，不会执行脚本
      const inputValue = await page.locator('input[name="username"]').inputValue();
      expect(inputValue).not.toContain('<script>');
    }
  });
  
  test('SQL注入防护', async ({ page }) => {
    await page.goto('/register');
    
    const sqlPayloads = [
      "'; DROP TABLE users; --",
      "1' OR '1'='1",
      "admin'--",
      "1; DELETE FROM users WHERE '1'='1"
    ];
    
    for (const payload of sqlPayloads) {
      await page.fill('input[name="username"]', payload);
      await page.fill('input[name="email"]', 'test@example.com');
      await page.fill('input[name="password"]', 'Password123!');
      await page.fill('input[name="confirmPassword"]', 'Password123!');
      
      await page.click('button[type="submit"]');
      
      // 应该被拒绝或有错误消息
      await expect(page.locator('.alert-error, .error-message')).toBeVisible();
    }
  });
  
  test('Unicode和特殊字符', async ({ page }) => {
    await page.goto('/register');
    
    const unicodeInputs = [
      '用户名中文',
      'ユーザー名',
      '用户名 with spaces',
      'emoji 😊🎉',
      'special-chars_123.test'
    ];
    
    for (const input of unicodeInputs) {
      await page.fill('input[name="username"]', input);
      await page.click('input[name="email"]');
      
      // 应该接受有效输入
      const inputValue = await page.locator('input[name="username"]').inputValue();
      expect(inputValue).toBe(input);
    }
  });
});

test.describe('并发请求测试', () => {
  test('重复提交表单', async ({ page }) => {
    await page.goto('/register');
    
    await page.fill('input[name="username"]', 'concurrent-test-user');
    await page.fill('input[name="email"]', 'concurrent@example.com');
    await page.fill('input[name="password"]', 'Password123!');
    await page.fill('input[name="confirmPassword"]', 'Password123!');
    
    // 快速多次点击提交
    await Promise.all([
      page.click('button[type="submit"]'),
      page.click('button[type="submit"]'),
      page.click('button[type="submit"]')
    ]);
    
    // 应该只创建一个用户
    await page.waitForResponse(response => 
      response.url().includes('/api/register') && response.status() === 201
    );
    
    // 等待所有请求完成
    await page.waitForLoadState('networkidle');
    
    // 验证只显示一个成功消息
    const successMessages = await page.locator('.alert-success').count();
    expect(successMessages).toBeLessThanOrEqual(1);
  });
});
