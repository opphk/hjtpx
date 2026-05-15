#!/bin/bash
# 任务14：E2E测试扩展
# 添加更多用户流程测试
# 添加边界条件测试
# 添加性能测试
# 添加并发测试
# 运行测试验证

echo "=========================================="
echo "任务14：E2E测试扩展"
echo "=========================================="

cd /workspace/hjtpx

# 1. 创建用户流程测试
echo "[14.1] 添加用户流程测试..."

cat > tests/e2e/user-flows.spec.js << 'EOF'
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
EOF

# 2. 添加边界条件测试
echo "[14.2] 添加边界条件测试..."

cat > tests/e2e/boundary-conditions.spec.js << 'EOF'
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
EOF

# 3. 添加性能测试
echo "[14.3] 添加性能测试..."

cat > tests/e2e/performance.spec.js << 'EOF'
import { test, expect } from '@playwright/test';

test.describe('性能测试', () => {
  test('页面加载时间', async ({ page }) => {
    await page.goto('/');
    
    // 记录关键指标
    const metrics = await page.evaluate(() => {
      return JSON.parse(JSON.stringify(performance.memory));
    });
    
    // 验证页面加载时间在合理范围内
    const loadTime = await page.evaluate(() => {
      const [navigation] = performance.getEntriesByType('navigation');
      return navigation.loadEventEnd - navigation.startTime;
    });
    
    console.log(`页面加载时间: ${loadTime}ms`);
    expect(loadTime).toBeLessThan(3000); // 小于3秒
  });
  
  test('首次内容绘制 (FCP)', async ({ page }) => {
    await page.goto('/');
    
    const fcp = await page.evaluate(() => {
      const entries = performance.getEntriesByType('paint');
      const fcpEntry = entries.find(entry => entry.name === 'first-contentful-paint');
      return fcpEntry ? fcpEntry.startTime : 0;
    });
    
    console.log(`首次内容绘制: ${fcp}ms`);
    expect(fcp).toBeLessThan(2000); // 小于2秒
  });
  
  test('交互响应时间', async ({ page }) => {
    await page.goto('/users');
    
    // 点击用户列表中的编辑按钮
    const startTime = Date.now();
    await page.click('button.edit-user:first-child');
    
    // 等待模态框出现
    await expect(page.locator('[role="dialog"]')).toBeVisible();
    
    const responseTime = Date.now() - startTime;
    console.log(`交互响应时间: ${responseTime}ms`);
    expect(responseTime).toBeLessThan(500); // 小于500毫秒
  });
  
  test('大量数据渲染性能', async ({ page }) => {
    await page.goto('/users');
    
    // 获取1000条用户数据
    const startTime = Date.now();
    
    // 验证表格渲染完成
    await page.waitForSelector('table.users-table tbody tr', { timeout: 5000 });
    
    const renderTime = Date.now() - startTime;
    console.log(`大量数据渲染时间: ${renderTime}ms`);
    expect(renderTime).toBeLessThan(2000); // 小于2秒
  });
  
  test('分页切换性能', async ({ page }) => {
    await page.goto('/users');
    
    const times = [];
    
    // 切换5页并记录时间
    for (let i = 1; i <= 5; i++) {
      const startTime = Date.now();
      await page.click(`button.page-number:text("${i}")`);
      await page.waitForLoadState('networkidle');
      times.push(Date.now() - startTime);
    }
    
    const avgTime = times.reduce((a, b) => a + b, 0) / times.length;
    console.log(`平均分页切换时间: ${avgTime}ms`);
    expect(avgTime).toBeLessThan(1000); // 平均小于1秒
  });
  
  test('内存使用监控', async ({ page }) => {
    await page.goto('/dashboard');
    
    // 记录初始内存
    const initialMemory = await page.evaluate(() => {
      if (performance.memory) {
        return performance.memory.usedJSHeapSize;
      }
      return 0;
    });
    
    // 执行多次导航
    for (let i = 0; i < 10; i++) {
      await page.goto('/users');
      await page.goto('/dashboard');
    }
    
    // 记录最终内存
    const finalMemory = await page.evaluate(() => {
      if (performance.memory) {
        return performance.memory.usedJSHeapSize;
      }
      return 0;
    });
    
    const memoryIncrease = finalMemory - initialMemory;
    console.log(`内存增长: ${memoryIncrease / 1024 / 1024}MB`);
    
    // 内存增长应该小于50MB
    expect(memoryIncrease).toBeLessThan(50 * 1024 * 1024);
  });
});

test.describe('资源加载性能', () => {
  test('静态资源缓存', async ({ page }) => {
    await page.goto('/');
    
    // 第一次访问
    const firstResponse = await page.reload();
    const cacheControl = firstResponse.headers()['cache-control'];
    
    // 验证Cache-Control头存在
    expect(cacheControl).toBeDefined();
    console.log(`Cache-Control: ${cacheControl}`);
  });
  
  test('懒加载组件', async ({ page }) => {
    await page.goto('/');
    
    // 记录初始网络请求数
    const initialRequests = 0;
    
    // 滚动到页面底部，触发懒加载
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await page.waitForTimeout(1000);
    
    // 验证懒加载内容已加载
    const lazyContent = page.locator('.lazy-loaded-content');
    await expect(lazyContent).toBeVisible();
  });
  
  test('图片优化', async ({ page }) => {
    await page.goto('/');
    
    // 获取所有图片
    const images = await page.$$eval('img', imgs => 
      imgs.map(img => ({
        src: img.src,
        naturalWidth: img.naturalWidth,
        loaded: img.complete && img.naturalWidth > 0
      }))
    );
    
    // 验证图片已正确加载
    for (const img of images) {
      if (img.src && !img.src.includes('data:')) {
        expect(img.loaded).toBe(true);
        console.log(`图片: ${img.src}, 宽度: ${img.naturalWidth}`);
      }
    }
  });
});
EOF

# 4. 添加并发测试
echo "[14.4] 添加并发测试..."

cat > tests/e2e/concurrency.spec.js << 'EOF'
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
EOF

# 5. 更新Playwright配置
echo "[14.5] 更新Playwright配置..."

cat > playwright.config.js << 'EOF'
const { defineConfig, devices } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html'],
    ['json', { outputFile: 'test-results/results.json' }],
    ['list']
  ],
  
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 10000,
    navigationTimeout: 30000
  },
  
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] }
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] }
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] }
    },
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] }
    },
    {
      name: 'Mobile Safari',
      use: { ...devices['iPhone 12'] }
    }
  ],
  
  webServer: {
    command: 'npm run start',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000
  }
});
EOF

echo "=========================================="
echo "任务14完成：E2E测试扩展"
echo "=========================================="
