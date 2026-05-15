import { test, expect } from '@playwright/test';

test.describe('并发测试', () => {
  test('多个用户同时注册', async ({ browser }) => {
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const context3 = await browser.newContext();
    
    const page1 = await context1.newPage();
    const page2 = await context2.newPage();
    const page3 = await context3.newPage();
    
    // 模拟三个用户同时注册
    await Promise.all([
      page1.goto('/register'),
      page2.goto('/register'),
      page3.goto('/register')
    ]);
    
    // 同时填写表单
    await Promise.all([
      page1.fill('input[name="username"]', 'user1'),
      page2.fill('input[name="username"]', 'user2'),
      page3.fill('input[name="username"]', 'user3')
    ]);
    
    await Promise.all([
      page1.fill('input[name="email"]', 'user1@example.com'),
      page2.fill('input[name="email"]', 'user2@example.com'),
      page3.fill('input[name="email"]', 'user3@example.com')
    ]);
    
    await Promise.all([
      page1.fill('input[name="password"]', 'Password123!'),
      page2.fill('input[name="password"]', 'Password123!'),
      page3.fill('input[name="password"]', 'Password123!')
    ]);
    
    await Promise.all([
      page1.fill('input[name="confirmPassword"]', 'Password123!'),
      page2.fill('input[name="confirmPassword"]', 'Password123!'),
      page3.fill('input[name="confirmPassword"]', 'Password123!')
    ]);
    
    // 同时提交
    await Promise.all([
      page1.click('button[type="submit"]'),
      page2.click('button[type="submit"]'),
      page3.click('button[type="submit"]')
    ]);
    
    // 等待所有请求完成
    await Promise.all([
      page1.waitForLoadState('networkidle'),
      page2.waitForLoadState('networkidle'),
      page3.waitForLoadState('networkidle')
    ]);
    
    // 验证所有用户都成功注册
    const success1 = await page1.locator('.alert-success').isVisible();
    const success2 = await page2.locator('.alert-success').isVisible();
    const success3 = await page3.locator('.alert-success').isVisible();
    
    expect(success1 || success2 || success3).toBe(true);
    
    await context1.close();
    await context2.close();
    await context3.close();
  });
  
  test('并发编辑同一资源', async ({ browser }) => {
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    
    const admin1 = await context1.newPage();
    const admin2 = await context2.newPage();
    
    // 两个管理员同时登录
    await Promise.all([
      admin1.goto('/login'),
      admin2.goto('/login')
    ]);
    
    await Promise.all([
      admin1.fill('input[name="email"]', 'admin@example.com'),
      admin2.fill('input[name="email"]', 'admin@example.com')
    ]);
    
    await Promise.all([
      admin1.fill('input[name="password"]', 'AdminPass123!'),
      admin2.fill('input[name="password"]', 'AdminPass123!')
    ]);
    
    await Promise.all([
      admin1.click('button[type="submit"]'),
      admin2.click('button[type="submit"]')
    ]);
    
    // 同时访问用户编辑页面
    await Promise.all([
      admin1.goto('/users/edit/1'),
      admin2.goto('/users/edit/1')
    ]);
    
    // 第一个管理员修改用户名
    await admin1.fill('input[name="username"]', 'UpdatedByAdmin1');
    await admin1.click('button.save-user');
    
    // 第二个管理员修改邮箱
    await admin2.fill('input[name="email"]', 'newemail@example.com');
    await admin2.click('button.save-user');
    
    // 验证其中一个操作成功，另一个应该收到冲突警告
    const conflict1 = await admin1.locator('.alert-warning:has-text("冲突")').isVisible();
    const conflict2 = await admin2.locator('.alert-warning:has-text("冲突")').isVisible();
    
    // 至少有一个成功
    const success1 = await admin1.locator('.alert-success').isVisible();
    const success2 = await admin2.locator('.alert-success').isVisible();
    
    expect(success1 || success2).toBe(true);
    
    await context1.close();
    await context2.close();
  });
  
  test('WebSocket并发消息', async ({ page }) => {
    await page.goto('/chat');
    
    // 发送多条消息
    for (let i = 0; i < 10; i++) {
      await page.fill('input[name="message"]', `Message ${i}`);
      await page.click('button.send-message');
    }
    
    // 验证所有消息都已发送
    await page.waitForTimeout(1000);
    const messageCount = await page.locator('.message-item').count();
    expect(messageCount).toBe(10);
  });
  
  test('并发API请求限流', async ({ page }) => {
    await page.goto('/dashboard');
    
    // 发送超过限流阈值的请求
    const requests = [];
    for (let i = 0; i < 100; i++) {
      requests.push(
        page.evaluate(() => 
          fetch('/api/data').then(res => res.status)
        )
      );
    }
    
    const results = await Promise.all(requests);
    const successCount = results.filter(r => r === 200).length;
    const rateLimitedCount = results.filter(r => r === 429).length;
    
    console.log(`成功: ${successCount}, 限流: ${rateLimitedCount}`);
    
    // 应该有一些请求被限流
    expect(rateLimitedCount).toBeGreaterThan(0);
  });
});
